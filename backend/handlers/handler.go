// Package handlers contains the HTTP endpoints.
package handlers

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"urlshortener/config"
)

// Handler holds everything the endpoints need (dependency injection, no globals).
type Handler struct {
	Users    *mongo.Collection
	Links    *mongo.Collection
	Redis    *redis.Client
	Cfg      config.Config
	SelfHost string // host of BaseURL, used to block redirect loops
}

func New(d *mongo.Database, r *redis.Client, cfg config.Config) *Handler {
	host := ""
	if u, err := url.Parse(cfg.BaseURL); err == nil {
		host = u.Host
	}
	return &Handler{Users: d.Collection("users"), Links: d.Collection("links"), Redis: r, Cfg: cfg, SelfHost: host}
}

func fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

func currentUser(c *gin.Context) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(c.GetString("userID"))
	return id, err == nil
}
