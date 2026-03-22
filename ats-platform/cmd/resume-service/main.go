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

	grpcHandler "github.com/example/ats-platform/internal/resume/grpc"
	"github.com/example/ats-platform/internal/resume/handler"
	"github.com/example/ats-platform/internal/resume/repository"
	"github.com/example/ats-platform/internal/resume/service"
	"github.com/example/ats-platform/internal/shared/database"
	"github.com/example/ats-platform/internal/shared/events"
	"github.com/example/ats-platform/internal/shared/llm"
	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/example/ats-platform/internal/shared/middleware"
	"github.com/example/ats-platform/internal/shared/pb/resume"
	"github.com/example/ats-platform/internal/shared/storage"
)

//go:embed static/index.html
var indexHTML string

// Config holds application configuration
type Config struct {
	HTTPHost      string
	HTTPPort      string
	GRPCHost      string
	GRPCPort      string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	MinioEndpoint string
	MinioUser     string
	MinioPassword string
	MinioBucket   string
	MinioUseSSL   bool
	RedisAddr     string
	RedisStream   string
	LLMBaseURL    string
	LLMAPIKey     string
	LLMModel      string
}

func loadConfig() *Config {
	return &Config{
		HTTPHost:      getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:      getEnv("HTTP_PORT", "8081"),
		GRPCHost:      getEnv("GRPC_HOST", "0.0.0.0"),
		GRPCPort:      getEnv("GRPC_PORT", "9090"),
		DBHost:        getEnv("DB_HOST", "192.168.250.233"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "ats"),
		MinioEndpoint: getEnv("MINIO_ENDPOINT", "192.168.250.233:9000"),
		MinioUser:     getEnv("MINIO_USER", "minioadmin"),
		MinioPassword: getEnv("MINIO_PASSWORD", "minioadmin"),
		MinioBucket:   getEnv("MINIO_BUCKET", "resumes"),
		MinioUseSSL:   getEnv("MINIO_USE_SSL", "false") == "true",
		RedisAddr:     getEnv("REDIS_ADDR", "192.168.250.233:6379"),
		RedisStream:   getEnv("REDIS_STREAM", "resume:events"),
		LLMBaseURL:    getEnv("LLM_BASE_URL", "https://api.moonshot.cn/v1"),
		LLMAPIKey:     getEnv("LLM_API_KEY", "sk-RlKlY5b6lPVV8nyD8zjmJYlNWNI7j9TEgcQFi3gfz9OObOAs"),
		LLMModel:      getEnv("LLM_MODEL", "moonshot-v1-8k"),
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

	// Initialize MinIO storage
	minioStorage, err := storage.NewMinIOClient(storage.MinIOConfig{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioUser,
		SecretKey: cfg.MinioPassword,
		UseSSL:    cfg.MinioUseSSL,
		Bucket:    cfg.MinioBucket,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create MinIO client: %v", err))
	}

	if err := minioStorage.EnsureBucket(ctx); err != nil {
		fmt.Printf("⚠️  Warning: Failed to ensure bucket: %v\n", err)
	}

	fmt.Printf("✅ Connected to MinIO: %s/%s\n", cfg.MinioEndpoint, cfg.MinioBucket)

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

	// Initialize LLM client (optional)
	var llmClient *llm.Client
	if cfg.LLMBaseURL != "" && cfg.LLMAPIKey != "" {
		llmClient = llm.NewClient(llm.Config{
			BaseURL: cfg.LLMBaseURL,
			APIKey:  cfg.LLMAPIKey,
			Model:   cfg.LLMModel,
		})
		fmt.Printf("✅ LLM client initialized: %s (%s)\n", cfg.LLMBaseURL, cfg.LLMModel)
	}

	// Initialize layered architecture
	resumeRepo := repository.NewGormRepository(postgresClient.GetDB())
	var resumeSvc service.ResumeService
	if llmClient != nil {
		resumeSvc = service.NewResumeServiceWithLLM(resumeRepo, minioStorage, publisher, llmClient)
	} else {
		resumeSvc = service.NewResumeService(resumeRepo, minioStorage, publisher)
	}
	resumeHandler := handler.NewResumeHandler(resumeSvc)

	// Start gRPC server
	go func() {
		grpcAddr := cfg.GRPCHost + ":" + cfg.GRPCPort
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			fmt.Printf("❌ gRPC listen failed: %v\n", err)
			return
		}
		grpcSrv := grpc.NewServer()
		resume.RegisterResumeServiceServer(grpcSrv, grpcHandler.NewResumeServiceServer(resumeSvc))
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

		minioStatus := "ok"
		if err := minioStorage.EnsureBucket(ctx); err != nil {
			minioStatus = "error: " + err.Error()
		}

		redisStatus := "ok"
		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		}

		c.JSON(200, gin.H{
			"service": "resume-service",
			"status":  "ok",
			"db":      dbStatus,
			"minio":   minioStatus,
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
		api.POST("/resumes", resumeHandler.Create)
		api.POST("/resumes/upload", resumeHandler.UploadAndParse)
		api.GET("/resumes/:id", resumeHandler.GetByID)
		api.GET("/resumes", resumeHandler.List)
		api.PUT("/resumes/:id", resumeHandler.Update)
		api.DELETE("/resumes/:id", resumeHandler.Delete)
		api.PUT("/resumes/:id/status", resumeHandler.UpdateStatus)
		api.POST("/resumes/:id/file", resumeHandler.UploadFile)
		api.POST("/resumes/:id/parse", resumeHandler.ParseResume)
	}

	// Start HTTP server
	addr := cfg.HTTPHost + ":" + cfg.HTTPPort
	fmt.Printf("🚀 HTTP Server running on http://%s\n", addr)
	fmt.Printf("   gRPC: %s:%s\n", cfg.GRPCHost, cfg.GRPCPort)
	fmt.Printf("   Database: %s@%s:%s/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	fmt.Printf("   MinIO: %s/%s\n", cfg.MinioEndpoint, cfg.MinioBucket)
	fmt.Printf("   Redis: %s (stream: %s)\n", cfg.RedisAddr, cfg.RedisStream)
	if llmClient != nil {
		fmt.Printf("   LLM: %s (%s)\n", cfg.LLMBaseURL, cfg.LLMModel)
	}
	fmt.Printf("   Architecture: Handler → Service -> Repository\n")

	if err := router.Run(addr); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
