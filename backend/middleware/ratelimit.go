package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit is a fixed-window limiter backed by Redis.
// INCR is atomic, so it stays correct even with many concurrent requests
// or multiple backend instances. keyFn decides who is being limited.
func RateLimit(rdb *redis.Client, name string, limit int, window time.Duration, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := fmt.Sprintf("rl:%s:%s", name, keyFn(c))

		n, err := rdb.Incr(ctx, key).Result()
		if err != nil { // if Redis is down, don't block users
			c.Next()
			return
		}
		if n == 1 {
			rdb.Expire(ctx, key, window)
		}
		if int(n) > limit {
			ttl, _ := rdb.TTL(ctx, key).Result()
			if ttl < 0 { // safety: a key must never live forever
				rdb.Expire(ctx, key, window)
				ttl = window
			}
			c.Header("Retry-After", strconv.Itoa(int(ttl.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "too many requests, please slow down"})
			return
		}
		c.Next()
	}
}

func ByIP(c *gin.Context) string   { return c.ClientIP() }
func ByUser(c *gin.Context) string { return c.GetString("userID") }
