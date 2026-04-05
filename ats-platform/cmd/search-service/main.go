package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/example/ats-platform/internal/search/handler"
	"github.com/example/ats-platform/internal/search/model"
	"github.com/example/ats-platform/internal/search/repository"
	"github.com/example/ats-platform/internal/search/service"
	"github.com/example/ats-platform/internal/shared/consul"
	"github.com/example/ats-platform/internal/shared/database"
	"github.com/example/ats-platform/internal/shared/events"
	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/example/ats-platform/internal/shared/middleware"
)

//go:embed static/index.html
var indexHTML string

type Config struct {
	ServiceName     string
	HTTPHost        string
	HTTPPort        string
	ConsulHost      string
	ConsulPort      string
	ESAddresses     []string
	ESUsername      string
	ESPassword      string
	ESAPIKey        string
	ESCloudID       string
	ESIndex         string
	ESInsecure      bool
	RedisAddr       string
	RedisStream     string
	RedisGroup      string
	RedisConsumer   string
	ConsumerEnabled bool
	ServiceAddress  string
}

func loadConfig() *Config {
	serviceName := getEnv("SERVICE_NAME", "search-service")
	httpPort := getEnv("HTTP_PORT", "8083")

	return &Config{
		ServiceName:     serviceName,
		HTTPHost:        getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:        httpPort,
		ConsulHost:      getEnv("CONSUL_HOST", "127.0.0.1"),
		ConsulPort:      getEnv("CONSUL_PORT", "8500"),
		ESAddresses:     splitAndTrim(getEnv("ES_ADDRESSES", "http://localhost:9200")),
		ESUsername:      getEnv("ES_USERNAME", ""),
		ESPassword:      getEnv("ES_PASSWORD", ""),
		ESAPIKey:        getEnv("ES_API_KEY", ""),
		ESCloudID:       getEnv("ES_CLOUD_ID", ""),
		ESIndex:         getEnv("ES_INDEX", "resumes"),
		ESInsecure:      getEnv("ES_INSECURE_SKIP_VERIFY", "false") == "true",
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisStream:     getEnv("REDIS_STREAM", "resume:events"),
		RedisGroup:      getEnv("REDIS_GROUP", "search-service"),
		RedisConsumer:   getEnv("REDIS_CONSUMER", fmt.Sprintf("%s-%s", serviceName, uuid.NewString())),
		ConsumerEnabled: getEnv("REDIS_CONSUMER_ENABLED", "true") != "false",
		ServiceAddress:  getEnv("SERVICE_ADDRESS", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "://") {
			part = "http://" + part
		}
		result = append(result, part)
	}
	if len(result) == 0 {
		return []string{"http://localhost:9200"}
	}
	return result
}

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	if err := logger.Init(logger.Config{
		Level:       "debug",
		Development: true,
	}); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	esClient, err := database.NewESClient(database.ESConfig{
		Addresses:          cfg.ESAddresses,
		Username:           cfg.ESUsername,
		Password:           cfg.ESPassword,
		APIKey:             cfg.ESAPIKey,
		CloudID:            cfg.ESCloudID,
		InsecureSkipVerify: cfg.ESInsecure,
	})
	if err != nil {
		log.Fatalf("failed to create elasticsearch client: %v", err)
	}
	defer esClient.Close()

	if err := esClient.Ping(); err != nil {
		log.Fatalf("failed to connect elasticsearch: %v", err)
	}
	logger.Infof("Connected to Elasticsearch: %s", strings.Join(cfg.ESAddresses, ","))

	esRepo := repository.NewESRepository(esClient.GetClient(), cfg.ESIndex)
	if repoImpl, ok := esRepo.(interface{ EnsureIndex(context.Context) error }); ok {
		if err := repoImpl.EnsureIndex(ctx); err != nil {
			log.Fatalf("failed to ensure elasticsearch index: %v", err)
		}
	}

	searchSvc := service.NewSearchService(esRepo)
	searchHandler := handler.NewSearchHandler(searchSvc)

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	redisReady := redisClient.Ping(ctx).Err() == nil
	if redisReady {
		logger.Infof("Connected to Redis: %s (stream: %s)", cfg.RedisAddr, cfg.RedisStream)
	} else {
		logger.Warnf("Redis connection failed: %v", redisClient.Ping(ctx).Err())
	}

	consulClient, err := consul.NewConsul(cfg.ConsulHost + ":" + cfg.ConsulPort)
	if err != nil {
		log.Fatalf("new consul client failed: %v", err)
	}
	ip, err := consul.ResolveServiceAddress(cfg.ServiceAddress)
	if err != nil {
		log.Fatalf("resolve service address failed: %v", err)
	}
	httpPort, _ := strconv.Atoi(cfg.HTTPPort)
	instanceUUID := uuid.NewString()
	httpEndpoint := consul.Endpoint{
		BaseName: consul.SearchServiceBaseName,
		Protocol: consul.ProtocolHTTP,
		IP:       ip,
		Port:     httpPort,
	}
	serviceID := consul.EndpointServiceID(httpEndpoint, instanceUUID)
	if err := consulClient.DeregisterEndpointInstances(httpEndpoint); err != nil {
		log.Fatalf("deregister stale service instances failed: %v", err)
	}
	if err := consulClient.RegisterEndpoint(httpEndpoint, instanceUUID); err != nil {
		log.Fatalf("register service failed: %v", err)
	}
	logger.Infof("Registered HTTP service to Consul with ID: %s", serviceID)

	var consumerCancel context.CancelFunc
	if cfg.ConsumerEnabled && redisReady {
		consumerCtx, cancel := context.WithCancel(context.Background())
		consumerCancel = cancel
		consumer := events.NewStreamConsumer(
			redisClient,
			cfg.RedisStream,
			cfg.RedisGroup,
			cfg.RedisConsumer,
			func(ctx context.Context, event events.ResumeEvent) error {
				return handleSearchEvent(ctx, searchSvc, event)
			},
		)
		go func() {
			logger.Infof(
				"Starting Redis stream consumer: stream=%s group=%s consumer=%s",
				cfg.RedisStream,
				cfg.RedisGroup,
				cfg.RedisConsumer,
			)
			if err := consumer.Start(consumerCtx); err != nil && err != context.Canceled {
				logger.Errorf("stream consumer stopped with error: %v", err)
			}
		}()
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.Recovery(), middleware.Logging(), middleware.CORS())

	router.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, indexHTML)
	})
	router.GET("/health", func(c *gin.Context) {
		redisStatus := "ok"
		if !redisReady {
			redisStatus = "degraded"
		}
		c.JSON(http.StatusOK, gin.H{
			"service": "search-service",
			"status":  "ok",
			"redis":   redisStatus,
			"time":    time.Now().Format(time.RFC3339),
		})
	})
	router.GET("/ready", func(c *gin.Context) {
		if err := esClient.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	api := router.Group("/api/v1")
	{
		api.GET("/search", searchHandler.Search)
		api.POST("/search/advanced", searchHandler.AdvancedSearch)
	}

	httpSrv := &http.Server{
		Addr:    cfg.HTTPHost + ":" + cfg.HTTPPort,
		Handler: router,
	}

	go func() {
		logger.Infof("HTTP Server running on http://%s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down search-service...")

	if consumerCancel != nil {
		consumerCancel()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("HTTP shutdown failed: %v", err)
	} else {
		logger.Info("HTTP server closed")
	}

	if err := consulClient.Deregister(serviceID); err != nil {
		logger.Errorf("Consul deregister failed: %v", err)
	} else {
		logger.Info("Service deregistered from Consul")
	}

	if err := redisClient.Close(); err != nil {
		logger.Errorf("Redis close failed: %v", err)
	} else {
		logger.Info("Redis client closed")
	}
}

func handleSearchEvent(ctx context.Context, searchSvc service.SearchService, event events.ResumeEvent) error {
	logger.Infof("Processing resume event: resume_id=%s action=%s", event.ResumeID, event.Action)

	switch event.Action {
	case events.ActionCreated, events.ActionUpdated:
		var payload events.ResumeDocumentPayload
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return fmt.Errorf("unmarshal resume payload: %w", err)
		}

		doc := buildResumeDocument(event.ResumeID, payload)
		if err := searchSvc.IndexResume(ctx, doc); err != nil {
			return fmt.Errorf("index resume document: %w", err)
		}
		return nil

	case events.ActionDeleted:
		err := searchSvc.DeleteResume(ctx, event.ResumeID)
		if err != nil && err != repository.ErrNotFound {
			return fmt.Errorf("delete resume document: %w", err)
		}
		return nil

	case events.ActionStatusChanged:
		var payload events.ResumeStatusChangedPayload
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return fmt.Errorf("unmarshal status payload: %w", err)
		}
		if payload.NewStatus == "" {
			return fmt.Errorf("missing new_status in status_changed payload")
		}
		err := searchSvc.UpdateResumeStatus(ctx, event.ResumeID, payload.NewStatus)
		if err != nil && err != repository.ErrNotFound {
			return fmt.Errorf("update resume status: %w", err)
		}
		return nil

	case events.ActionParsed:
		return nil

	default:
		logger.Warnf("Ignoring unsupported event action: %s", event.Action)
		return nil
	}
}

func buildResumeDocument(resumeID string, payload events.ResumeDocumentPayload) *model.ResumeDocument {
	skills := extractStringSlice(payload.ParsedData, "skills")
	education := strings.Join(extractStringSlice(payload.ParsedData, "education"), "\n")
	workHistory := strings.Join(extractStringSlice(payload.ParsedData, "work_experience"), "\n")

	createdAt := payload.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := payload.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	return model.NewResumeDocument(
		resumeID,
		payload.Name,
		payload.Email,
		skills,
		extractExperienceYears(payload.ParsedData),
		education,
		workHistory,
		payload.Status,
		payload.Source,
		createdAt,
		updatedAt,
	)
}

func extractExperienceYears(parsedData map[string]interface{}) int {
	rawItems, ok := parsedData["work_experience"].([]interface{})
	if !ok {
		return 0
	}
	if len(rawItems) == 0 {
		return 0
	}
	return len(rawItems)
}

func extractStringSlice(parsedData map[string]interface{}, key string) []string {
	raw, ok := parsedData[key]
	if !ok || raw == nil {
		return nil
	}

	switch value := raw.(type) {
	case []string:
		return value
	case []interface{}:
		result := make([]string, 0, len(value))
		for _, item := range value {
			switch typed := item.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					result = append(result, typed)
				}
			case map[string]interface{}:
				bytes, err := json.Marshal(typed)
				if err == nil {
					result = append(result, string(bytes))
				}
			default:
				result = append(result, fmt.Sprint(typed))
			}
		}
		return result
	default:
		return []string{fmt.Sprint(value)}
	}
}
