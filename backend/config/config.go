// Package config reads settings from environment variables (.env in dev).
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDB        string
	RedisAddr      string
	RedisPassword  string
	JWTSecret      string
	BaseURL        string // public base of short links, e.g. http://localhost:8080
	FrontendOrigin string // allowed CORS origin
	CreateLimit    int    // max link creations per user per minute
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	limit, err := strconv.Atoi(get("CREATE_LIMIT_PER_MIN", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	return Config{
		Port:           get("PORT", "8080"),
		MongoURI:       get("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:        get("MONGO_DB", "urlshortener"),
		RedisAddr:      get("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  get("REDIS_PASSWORD", ""),
		JWTSecret:      get("JWT_SECRET", ""),
		BaseURL:        get("BASE_URL", "http://localhost:8080"),
		FrontendOrigin: get("FRONTEND_ORIGIN", "http://localhost:5173"),
		CreateLimit:    limit,
	}
}
