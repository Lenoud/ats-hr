package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcHandler "github.com/example/ats-platform/internal/resume/grpc"
	"github.com/example/ats-platform/internal/resume/handler"
	"github.com/example/ats-platform/internal/resume/repository"
	"github.com/example/ats-platform/internal/resume/service"
	"github.com/example/ats-platform/internal/shared/consul"
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

type Config struct {
	ServiceName   string
	HTTPHost      string
	HTTPPort      string
	GRPCHost      string
	GRPCPort      string
	ConsulHost    string
	ConsulPort    string
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
		ServiceName:   getEnv("SERVICE_NAME", "resume-service"),
		HTTPHost:      getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:      getEnv("HTTP_PORT", "8081"),
		GRPCHost:      getEnv("GRPC_HOST", "0.0.0.0"),
		GRPCPort:      getEnv("GRPC_PORT", "9090"),
		ConsulHost:    getEnv("CONSUL_HOST", "192.168.1.40"),
		ConsulPort:    getEnv("CONSUL_PORT", "8500"),
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

	// 日志初始化
	if err := logger.Init(logger.Config{
		Level:       "debug",
		Development: true,
	}); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// DB
	postgresClient, err := database.NewPostgresClient(database.PostgresConfig{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
	})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	defer postgresClient.Close()
	fmt.Println("✅ Connected to PostgreSQL database")

	// MinIO
	minioStorage, err := storage.NewMinIOClient(storage.MinIOConfig{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioUser,
		SecretKey: cfg.MinioPassword,
		UseSSL:    cfg.MinioUseSSL,
		Bucket:    cfg.MinioBucket,
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}
	if err := minioStorage.EnsureBucket(ctx); err != nil {
		fmt.Printf("⚠️  Warning: Failed to ensure bucket: %v\n", err)
	}
	fmt.Printf("✅ Connected to MinIO: %s/%s\n", cfg.MinioEndpoint, cfg.MinioBucket)

	// Redis
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("⚠️  Warning: Redis connection failed: %v\n", err)
	}

	// Event publisher
	var publisher *events.EventPublisher
	if redisClient.Ping(ctx).Err() == nil {
		publisher = events.NewEventPublisher(redisClient, cfg.RedisStream)
		fmt.Printf("✅ Connected to Redis: %s (stream: %s)\n", cfg.RedisAddr, cfg.RedisStream)
	}
	if publisher != nil {
		fmt.Printf("✅ Redis Events: Enabled\n")
	}

	// LLM
	var llmClient *llm.Client
	if cfg.LLMBaseURL != "" && cfg.LLMAPIKey != "" {
		llmClient = llm.NewClient(llm.Config{
			BaseURL: cfg.LLMBaseURL,
			APIKey:  cfg.LLMAPIKey,
			Model:   cfg.LLMModel,
		})
		fmt.Printf("✅ LLM client initialized: %s (%s)\n", cfg.LLMBaseURL, cfg.LLMModel)
	}

	// Consul Registration
	consulClient, err := consul.NewConsul(cfg.ConsulHost + ":" + cfg.ConsulPort)
	if err != nil {
		log.Fatalf("NewConsul failed: %v", err)
	}
	ipObj, err := consul.GetOutboundIP()
	if err != nil {
		log.Fatalf("GetOutboundIP failed: %v", err)
	}
	ip := ipObj.String()
	grpcPort, _ := strconv.Atoi(cfg.GRPCPort)
	uuid := uuid.NewString()
	serviceId := fmt.Sprintf("%s-%s-%d-%s", cfg.ServiceName, ip, grpcPort, uuid)
	if err := consulClient.RegisterService(cfg.ServiceName, ip, grpcPort, uuid); err != nil {
		log.Fatalf("RegisterService failed: %v", err)
	}
	fmt.Println("✅ Registered service to Consul with ID:", serviceId)

	// initialize service and handlers
	resumeRepo := repository.NewGormRepository(postgresClient.GetDB())
	var resumeSvc service.ResumeService
	if llmClient != nil {
		resumeSvc = service.NewResumeServiceWithLLM(resumeRepo, minioStorage, publisher, llmClient)
	} else {
		resumeSvc = service.NewResumeService(resumeRepo, minioStorage, publisher)
	}
	resumeHandler := handler.NewResumeHandler(resumeSvc)

	// gRPC Server
	var grpcSrv *grpc.Server
	go func() {
		grpcAddr := cfg.GRPCHost + ":" + cfg.GRPCPort
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("gRPC listen failed: %v", err)
		}

		grpcSrv = grpc.NewServer()
		resume.RegisterResumeServiceServer(grpcSrv, grpcHandler.NewResumeServiceServer(resumeSvc))
		reflection.Register(grpcSrv)

		fmt.Printf("🚀 gRPC Server running on %s\n", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			fmt.Printf("gRPC server closed: %v\n", err)
		}
	}()

	// HTTP Server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.Recovery(), middleware.Logging(), middleware.CORS())

	router.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, indexHTML)
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "resume-service",
			"status":  "ok",
			"time":    time.Now().Format(time.RFC3339),
		})
	})
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

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

	// 优雅退出改造：用 http.Server 替代 router.Run()
	httpSrv := &http.Server{
		Addr:    cfg.HTTPHost + ":" + cfg.HTTPPort,
		Handler: router,
	}

	// ====================== 启动 HTTP ======================
	go func() {
		fmt.Printf("🚀 HTTP Server running on http://%s\n", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// ====================== 优雅退出核心逻辑 ======================
	// 1. 监听系统信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞等待信号
	fmt.Println("\n🛑 开始优雅关闭服务...")

	// 2. Close HTTP Server with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP关闭失败: %v", err)
	} else {
		fmt.Println("✅ HTTP服务已关闭")
	}

	// 3. Close gRPC
	if grpcSrv != nil {
		grpcSrv.GracefulStop()
		fmt.Println("✅ gRPC服务已关闭")
	}

	// 4. Consul Deregister
	if err := consulClient.Deregister(serviceId); err != nil {
		log.Printf("Consul反注册失败: %v", err)
	} else {
		fmt.Println("✅ 服务已从Consul注销")
	}

	// Close Redis
	if err := redisClient.Close(); err != nil {
		log.Printf("Redis关闭失败: %v", err)
	} else {
		fmt.Println("✅ Redis已关闭")
	}
}
