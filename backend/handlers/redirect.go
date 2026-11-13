package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"urlshortener/models"
)

const cacheTTL = 24 * time.Hour

// What we keep in Redis for each link: only what a redirect needs.
type cachedLink struct {
	URL       string     `json:"u"`
	ExpiresAt *time.Time `json:"e,omitempty"`
}

// GET /:code
// 1) look in Redis  2) on a miss read MongoDB and fill the cache
// 3) count the click atomically in Redis  4) 302 redirect.
func (h *Handler) Redirect(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")
	key := "link:" + code

	var cl cachedLink
	hit := false
	if raw, err := h.Redis.Get(ctx, key).Result(); err == nil {
		hit = json.Unmarshal([]byte(raw), &cl) == nil
	}

	if !hit {
		var link models.Link
		err := h.Links.FindOne(ctx, bson.M{"code": code}).Decode(&link)
		if err == mongo.ErrNoDocuments {
			fail(c, http.StatusNotFound, "short link not found")
			return
		} else if err != nil {
			fail(c, http.StatusInternalServerError, "something went wrong")
			return
		}
		cl = cachedLink{URL: link.URL, ExpiresAt: link.ExpiresAt}
		ttl := cacheTTL
		if cl.ExpiresAt != nil {
			remaining := time.Until(*cl.ExpiresAt)
			if remaining <= 0 {
				fail(c, http.StatusGone, "this link has expired")
				return
			}
			if remaining < ttl { // cache entry must not outlive the link
				ttl = remaining
			}
		}
		if b, err := json.Marshal(cl); err == nil {
			h.Redis.Set(ctx, key, b, ttl)
		}
	}

	if cl.ExpiresAt != nil && time.Now().After(*cl.ExpiresAt) {
		fail(c, http.StatusGone, "this link has expired")
		return
	}

	h.recordClick(c, code)
	// 302 (not 301): browsers must not cache it, otherwise repeat visits
	// would skip our server and never be counted.
	c.Redirect(http.StatusFound, cl.URL)
}

// recordClick uses atomic HINCRBY in one MULTI/EXEC transaction:
// total counter, today's counter, and a "dirty" set so the flusher knows
// which links have new clicks.
func (h *Handler) recordClick(c *gin.Context, code string) {
	ctx := c.Request.Context()
	pk := "pending:" + code
	pipe := h.Redis.TxPipeline()
	pipe.HIncrBy(ctx, pk, "total", 1)
	pipe.HIncrBy(ctx, pk, time.Now().UTC().Format(dayLayout), 1)
	pipe.SAdd(ctx, "clicks:dirty", code)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("click not counted for %s: %v", code, err)
	}
}

// StartFlusher periodically moves click counters from Redis into MongoDB so
// they are stored permanently, without a Mongo write on every redirect.
func (h *Handler) StartFlusher(ctx context.Context, every time.Duration) {
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.Flush(ctx)
			}
		}
	}()
}

// Flush: RENAME makes the hand-off atomic. Clicks arriving during the flush
// go into a fresh "pending" hash and are picked up next time, so none are lost.
func (h *Handler) Flush(ctx context.Context) {
	codes, err := h.Redis.SPopN(ctx, "clicks:dirty", 500).Result()
	if err != nil {
		return
	}
	for _, code := range codes {
		tmp := "flushing:" + code
		if err := h.Redis.Rename(ctx, "pending:"+code, tmp).Err(); err != nil {
			continue // nothing pending
		}
		data, err := h.Redis.HGetAll(ctx, tmp).Result()
		if err != nil {
			continue
		}
		inc := bson.M{}
		for field, v := range data {
			n, _ := strconv.ParseInt(v, 10, 64)
			if field == "total" {
				inc["clicks"] = n
			} else {
				inc["daily."+field] = n
			}
		}
		if len(inc) == 0 {
			h.Redis.Del(ctx, tmp)
			continue
		}
		if _, err := h.Links.UpdateOne(ctx, bson.M{"code": code}, bson.M{"$inc": inc}); err != nil {
			log.Printf("flush failed for %s: %v", code, err)
			continue // keep tmp so the data is not silently dropped
		}
		h.Redis.Del(ctx, tmp)
	}
}
