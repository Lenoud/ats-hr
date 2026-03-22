package main

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcHandler "github.com/example/ats-platform/internal/interview/grpc"
	"github.com/example/ats-platform/internal/interview/handler"
	"github.com/example/ats-platform/internal/interview/repository"
	"github.com/example/ats-platform/internal/interview/service"
	"github.com/example/ats-platform/internal/shared/database"
	"github.com/example/ats-platform/internal/shared/events"
	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/example/ats-platform/internal/shared/middleware"
	"github.com/example/ats-platform/internal/shared/pb/interview"
)

//go:embed static/index.html
var indexHTML string

// Config holds application configuration
type Config struct {
	HTTPHost    string
	HTTPPort    string
	GRPCHost    string
	GRPCPort    string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	RedisAddr   string
	RedisStream string
}

func loadConfig() *Config {
	return &Config{
		GRPCHost:    getEnv("GRPC_HOST", "0.0.0.0"),
		GRPCPort:    getEnv("GRPC_PORT", "9091"),
		HTTPHost:    getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:    getEnv("HTTP_PORT", "8082"),
		DBHost:      getEnv("DB_HOST", "192.168.250.233"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "postgres"),
		DBName:      getEnv("DB_NAME", "ats"),
		RedisAddr:   getEnv("REDIS_ADDR", "192.168.250.233:6379"),
		RedisStream: getEnv("REDIS_STREAM", "interview:events"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:       "debug",
		Development: true,
	}); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Initialize PostgreSQL database
	postgresClient, err := database.NewPostgresClient(database.PostgresConfig{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database: %v", err))
	}
	defer postgresClient.Close()

	fmt.Println("✅ Connected to PostgreSQL database")

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("⚠️  Warning: Redis connection failed: %v\n", err)
	}

	// Initialize event publisher
	var publisher *events.EventPublisher
	if redisClient.Ping(ctx).Err() == nil {
		publisher = events.NewEventPublisher(redisClient, cfg.RedisStream)
		fmt.Printf("✅ Connected to Redis: %s (stream: %s)\n", cfg.RedisAddr, cfg.RedisStream)
	}

	// Initialize layered architecture
	interviewRepo := repository.NewInterviewRepository(postgresClient.GetDB())
	feedbackRepo := repository.NewFeedbackRepository(postgresClient.GetDB())
	portfolioRepo := repository.NewPortfolioRepository(postgresClient.GetDB())

	interviewSvc := service.NewInterviewService(interviewRepo)
	feedbackSvc := service.NewFeedbackService(feedbackRepo, interviewRepo)
	portfolioSvc := service.NewPortfolioService(portfolioRepo)

	interviewHandler := handler.NewInterviewHandler(interviewSvc)
	feedbackHandler := handler.NewFeedbackHandler(feedbackSvc)
	portfolioHandler := handler.NewPortfolioHandler(portfolioSvc)

	// Start gRPC server
	go func() {
		grpcAddr := cfg.GRPCHost + ":" + cfg.GRPCPort
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			fmt.Printf("❌ gRPC listen failed: %v\n", err)
			return
		}
		grpcSrv := grpc.NewServer()
		interview.RegisterInterviewServiceServer(grpcSrv, grpcHandler.NewInterviewServiceServer(interviewSvc))
		interview.RegisterFeedbackServiceServer(grpcSrv, grpcHandler.NewFeedbackServiceServer(feedbackSvc))
		interview.RegisterPortfolioServiceServer(grpcSrv, grpcHandler.NewPortfolioServiceServer(portfolioSvc))
		reflection.Register(grpcSrv)
		fmt.Printf("🚀 gRPC Server running on %s\n", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.Logging())
	router.Use(middleware.CORS())

	// Home page
	router.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, indexHTML)
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := postgresClient.Ping(); err != nil {
			dbStatus = "error: " + err.Error()
		}

		redisStatus := "ok"
		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		}

		c.JSON(200, gin.H{
			"service": "interview-service",
			"status":  "ok",
			"db":      dbStatus,
			"redis":   redisStatus,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		if err := postgresClient.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Interview routes
		api.POST("/interviews", interviewHandler.Create)
		api.GET("/interviews/:id", interviewHandler.GetByID)
		api.PUT("/interviews/:id/status", interviewHandler.UpdateStatus)
		api.DELETE("/interviews/:id", interviewHandler.Delete)

		// Interview feedback routes
		api.POST("/interviews/:id/feedback", feedbackHandler.Submit)
		api.GET("/interviews/:id/feedback", feedbackHandler.GetByInterviewID)

		// Resume interviews routes
		api.GET("/resumes/:id/interviews", interviewHandler.ListByResumeID)

		// Portfolio routes
		api.POST("/resumes/:id/portfolios", portfolioHandler.Create)
		api.GET("/resumes/:id/portfolios", portfolioHandler.ListByResumeID)
		api.DELETE("/portfolios/:id", portfolioHandler.Delete)
	}

	// Start HTTP server
	addr := cfg.HTTPHost + ":" + cfg.HTTPPort
	fmt.Printf("🚀 HTTP Server running on http://%s\n", addr)
	fmt.Printf("   gRPC: %s:%s\n", cfg.GRPCHost, cfg.GRPCPort)
	fmt.Printf("   Database: %s@%s:%s/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	fmt.Printf("   Redis: %s (stream: %s)\n", cfg.RedisAddr, cfg.RedisStream)
	if publisher != nil {
		fmt.Printf("   Events: Enabled\n")
	}
	fmt.Printf("   Architecture: Handler → Service -> Repository\n")

	if err := router.Run(addr); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
