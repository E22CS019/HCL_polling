package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"pollster-backend/internal/config"
	"pollster-backend/internal/handlers"
	"pollster-backend/internal/middleware"
	"pollster-backend/internal/repository"
	"pollster-backend/internal/services"
	"pollster-backend/internal/sse"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// ── Mongo ─────────────────────────────────────────────────────────────────
	mongoClient, err := repository.NewMongoClient(cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	db := mongoClient.Database(cfg.MongoDB)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}()

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb, err := repository.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo, err := repository.NewUserRepository(db)
	if err != nil {
		log.Fatalf("user repo: %v", err)
	}
	pollRepo, err := repository.NewPollRepository(db)
	if err != nil {
		log.Fatalf("poll repo: %v", err)
	}

	// ── SSE Broker ────────────────────────────────────────────────────────────
	broker := sse.NewBroker(rdb)

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
	pollSvc := services.NewPollService(pollRepo, rdb, broker)

	// ── Handlers ──────────────────────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(authSvc)
	pollHandler := handlers.NewPollHandler(pollSvc, broker)

	// ── Gin Router ────────────────────────────────────────────────────────────
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// CORS — allow the React dev server and any configured production origins.
	origins := strings.Split(cfg.AllowedOrigins, ",")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Voter-Fingerprint"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Routes ────────────────────────────────────────────────────────────────
	api := r.Group("/api/v1")

	// Auth
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", middleware.RequireAuth(authSvc), authHandler.Me)
	}

	// Polls — static paths must be registered before wildcard /:id
	polls := api.Group("/polls")
	{
		polls.GET("", pollHandler.ListRecent)
		polls.POST("", middleware.RequireAuth(authSvc), pollHandler.CreatePoll)
		polls.GET("/mine", middleware.RequireAuth(authSvc), pollHandler.GetMyPolls)

		polls.GET("/:id", pollHandler.GetPoll)
		polls.GET("/:id/results", pollHandler.GetResults)
		polls.GET("/:id/stream", pollHandler.StreamResults)    // SSE
		polls.GET("/:id/vote-status", pollHandler.CheckVoteStatus)
		polls.POST("/:id/vote", pollHandler.Vote)
		polls.POST("/:id/close", middleware.RequireAuth(authSvc), pollHandler.ClosePoll)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── Server ────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE streams need unlimited write time
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down gracefully…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}
