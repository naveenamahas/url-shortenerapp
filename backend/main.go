package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"urlshortener/config"
	"urlshortener/db"
	"urlshortener/handlers"
	"urlshortener/middleware"
)

func main() {
	_ = godotenv.Load() // optional .env file for local dev
	cfg := config.Load()
	if len(cfg.JWTSecret) < 16 {
		log.Fatal("JWT_SECRET must be set (at least 16 characters)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mdb, err := db.ConnectMongo(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	rdb, err := db.ConnectRedis(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	h := handlers.New(mdb, rdb, cfg)
	h.StartFlusher(ctx, 10*time.Second)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.FrontendOrigin},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:       12 * time.Hour,
	}))

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	auth := api.Group("/auth", middleware.RateLimit(rdb, "auth", 20, time.Minute, middleware.ByIP))
	auth.POST("/signup", h.Signup)
	auth.POST("/login", h.Login)

	links := api.Group("/links", middleware.Auth(cfg.JWTSecret))
	links.GET("", h.ListLinks)
	links.POST("", middleware.RateLimit(rdb, "create", cfg.CreateLimit, time.Minute, middleware.ByUser), h.CreateLink)
	links.GET("/:code/stats", h.Stats)
	links.DELETE("/:code", h.DeleteLink)

	r.GET("/:code", h.Redirect) // the public short link
	r.NoRoute(func(c *gin.Context) { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}) })

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done() // Ctrl+C: stop gracefully and save pending clicks
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
	h.Flush(shutdown)
	log.Println("bye")
}
