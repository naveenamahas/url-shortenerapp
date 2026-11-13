package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"urlshortener/models"
	"urlshortener/utils"
)

const dayLayout = "2006-01-02"

type linkView struct {
	Code      string     `json:"code"`
	ShortURL  string     `json:"shortUrl"`
	URL       string     `json:"url"`
	Clicks    int64      `json:"clicks"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (h *Handler) view(l models.Link, pending int64) linkView {
	return linkView{
		Code:      l.Code,
		ShortURL:  strings.TrimRight(h.Cfg.BaseURL, "/") + "/" + l.Code,
		URL:       l.URL,
		Clicks:    l.Clicks + pending, // saved in Mongo + not-yet-flushed in Redis
		ExpiresAt: l.ExpiresAt,
		CreatedAt: l.CreatedAt,
	}
}

type createReq struct {
	URL       string     `json:"url"`
	Alias     string     `json:"alias"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// POST /api/links
func (h *Handler) CreateLink(c *gin.Context) {
	uid, ok := currentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "login required")
		return
	}
	var in createReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	clean, err := utils.ValidateURL(in.URL, h.SelfHost)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now()) {
		fail(c, http.StatusBadRequest, "expiry must be in the future")
		return
	}
	link := models.Link{ID: primitive.NewObjectID(), URL: clean, UserID: uid, CreatedAt: time.Now().UTC()}
	if in.ExpiresAt != nil {
		t := in.ExpiresAt.UTC()
		link.ExpiresAt = &t
	}
	ctx := c.Request.Context()

	if alias := strings.TrimSpace(in.Alias); alias != "" {
		// Custom alias: if taken, tell the user clearly (409).
		if err := utils.ValidateAlias(alias); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		link.Code = alias
		if _, err := h.Links.InsertOne(ctx, link); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				fail(c, http.StatusConflict, "this alias is already taken")
				return
			}
			fail(c, http.StatusInternalServerError, "could not save link")
			return
		}
	} else {
		// Random code: insert and rely on the unique index. If we collide, retry.
		saved := false
		for i := 0; i < 5 && !saved; i++ {
			code, err := utils.NewCode(7)
			if err != nil {
				fail(c, http.StatusInternalServerError, "could not generate code")
				return
			}
			link.Code = code
			if _, err := h.Links.InsertOne(ctx, link); err != nil {
				if mongo.IsDuplicateKeyError(err) {
					continue // collision, try another code
				}
				fail(c, http.StatusInternalServerError, "could not save link")
				return
			}
			saved = true
		}
		if !saved {
			fail(c, http.StatusInternalServerError, "could not generate a unique code, try again")
			return
		}
	}
	c.JSON(http.StatusCreated, h.view(link, 0))
}

// GET /api/links  (only the logged-in user's links)
func (h *Handler) ListLinks(c *gin.Context) {
	uid, ok := currentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "login required")
		return
	}
	ctx := c.Request.Context()
	cur, err := h.Links.Find(ctx, bson.M{"userId": uid},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if err != nil {
		fail(c, http.StatusInternalServerError, "could not load links")
		return
	}
	var links []models.Link
	if err := cur.All(ctx, &links); err != nil {
		fail(c, http.StatusInternalServerError, "could not load links")
		return
	}
	// One round trip to Redis for all pending click counters.
	pipe := h.Redis.Pipeline()
	cmds := make([]interface{ Int64() (int64, error) }, len(links))
	for i, l := range links {
		cmds[i] = pipe.HGet(ctx, "pending:"+l.Code, "total")
	}
	_, _ = pipe.Exec(ctx) // redis.Nil for links with no pending clicks is expected
	out := make([]linkView, len(links))
	for i, l := range links {
		p, _ := cmds[i].Int64()
		out[i] = h.view(l, p)
	}
	c.JSON(http.StatusOK, gin.H{"links": out})
}

// DELETE /api/links/:code
func (h *Handler) DeleteLink(c *gin.Context) {
	uid, ok := currentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "login required")
		return
	}
	code := c.Param("code")
	res, err := h.Links.DeleteOne(c.Request.Context(), bson.M{"code": code, "userId": uid})
	if err != nil {
		fail(c, http.StatusInternalServerError, "could not delete link")
		return
	}
	if res.DeletedCount == 0 {
		fail(c, http.StatusNotFound, "link not found")
		return
	}
	h.Redis.Del(c.Request.Context(), "link:"+code, "pending:"+code) // stop serving from cache
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type dayStat struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

// GET /api/links/:code/stats  -> last 7 days of clicks
func (h *Handler) Stats(c *gin.Context) {
	uid, ok := currentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "login required")
		return
	}
	ctx := c.Request.Context()
	code := c.Param("code")
	var link models.Link
	err := h.Links.FindOne(ctx, bson.M{"code": code, "userId": uid}).Decode(&link)
	if err == mongo.ErrNoDocuments {
		fail(c, http.StatusNotFound, "link not found")
		return
	} else if err != nil {
		fail(c, http.StatusInternalServerError, "could not load stats")
		return
	}
	pending, _ := h.Redis.HGetAll(ctx, "pending:"+code).Result()
	total, _ := strconv.ParseInt(pending["total"], 10, 64)

	days := make([]dayStat, 7)
	for i := 0; i < 7; i++ {
		d := time.Now().UTC().AddDate(0, 0, i-6).Format(dayLayout)
		n := link.Daily[d]
		if v, ok := pending[d]; ok {
			p, _ := strconv.ParseInt(v, 10, 64)
			n += p
		}
		days[i] = dayStat{Date: d, Clicks: n}
	}
	c.JSON(http.StatusOK, gin.H{"code": code, "clicks": link.Clicks + total, "days": days})
}
