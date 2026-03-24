package main

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	sharedconsul "github.com/example/ats-platform/internal/shared/consul"
	"github.com/gin-gonic/gin"
)

type config struct {
	HTTPHost   string
	HTTPPort   string
	ConsulHost string
	ConsulPort string
}

type routeTarget struct {
	ServiceKey      string
	BaseServiceName string
	Protocol        sharedconsul.Protocol
}

var routeTargets = []struct {
	Prefix string
	Target routeTarget
}{
	{
		Prefix: "/resumes/",
		Target: routeTarget{
			ServiceKey:      "interview",
			BaseServiceName: sharedconsul.InterviewServiceBaseName,
			Protocol:        sharedconsul.ProtocolHTTP,
		},
	},
	{
		Prefix: "/resumes",
		Target: routeTarget{
			ServiceKey:      "resume",
			BaseServiceName: sharedconsul.ResumeServiceBaseName,
			Protocol:        sharedconsul.ProtocolHTTP,
		},
	},
	{
		Prefix: "/interviews",
		Target: routeTarget{
			ServiceKey:      "interview",
			BaseServiceName: sharedconsul.InterviewServiceBaseName,
			Protocol:        sharedconsul.ProtocolHTTP,
		},
	},
	{
		Prefix: "/portfolios",
		Target: routeTarget{
			ServiceKey:      "interview",
			BaseServiceName: sharedconsul.InterviewServiceBaseName,
			Protocol:        sharedconsul.ProtocolHTTP,
		},
	},
	{
		Prefix: "/search",
		Target: routeTarget{
			ServiceKey:      "search",
			BaseServiceName: sharedconsul.SearchServiceBaseName,
			Protocol:        sharedconsul.ProtocolHTTP,
		},
	},
}

//go:embed static/index.html
var indexHTML string

func loadConfig() *config {
	return &config{
		HTTPHost:   getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		ConsulHost: getEnv("CONSUL_HOST", "127.0.0.1"),
		ConsulPort: getEnv("CONSUL_PORT", "8500"),
	}
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func main() {
	cfg := loadConfig()
	discovery, err := newServiceDiscovery(cfg.ConsulHost + ":" + cfg.ConsulPort)
	if err != nil {
		panic(fmt.Sprintf("failed to create consul discovery client: %v", err))
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, indexHTML)
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "api-gateway",
			"status":  "ok",
			"services": gin.H{
				"resume":    checkServiceStatus(discovery, sharedconsul.ResumeServiceBaseName, sharedconsul.ProtocolHTTP),
				"interview": checkServiceStatus(discovery, sharedconsul.InterviewServiceBaseName, sharedconsul.ProtocolHTTP),
				"search":    checkServiceStatus(discovery, sharedconsul.SearchServiceBaseName, sharedconsul.ProtocolHTTP),
			},
			"time": time.Now().Format(time.RFC3339),
		})
	})

	r.Any("/api/v1/*action", func(c *gin.Context) {
		proxyHandler(c, discovery)
	})

	addr := cfg.HTTPHost + ":" + cfg.HTTPPort
	fmt.Printf("API Gateway running on http://%s\n", addr)
	if err := r.Run(addr); err != nil {
		panic(fmt.Sprintf("failed to run gateway: %v", err))
	}
}

func discoverServiceURL(discovery *serviceDiscovery, baseName string, protocol sharedconsul.Protocol) string {
	instance, err := discovery.resolve(baseName, protocol)
	if err != nil {
		return "#"
	}
	return instance.externalURL()
}

func checkServiceStatus(discovery *serviceDiscovery, baseName string, protocol sharedconsul.Protocol) gin.H {
	instance, err := discovery.resolve(baseName, protocol)
	if err != nil {
		return gin.H{
			"status": "error",
			"error":  err.Error(),
		}
	}

	resp, err := http.Get(instance.baseURL() + "/health")
	if err != nil {
		return gin.H{
			"status":  "error",
			"address": instance.baseURL(),
			"error":   err.Error(),
		}
	}
	defer resp.Body.Close()

	status := "ok"
	if resp.StatusCode >= http.StatusBadRequest {
		status = "error"
	}

	return gin.H{
		"status":      status,
		"address":     instance.baseURL(),
		"status_code": resp.StatusCode,
	}
}

func resolveRouteTarget(path string) (*routeTarget, bool) {
	for _, item := range routeTargets {
		if item.Prefix == "/resumes/" {
			if strings.HasPrefix(path, "/resumes/") &&
				(strings.Contains(path, "/interviews") || strings.Contains(path, "/portfolios")) {
				target := item.Target
				return &target, true
			}
			continue
		}
		if strings.HasPrefix(path, item.Prefix) {
			target := item.Target
			return &target, true
		}
	}
	return nil, false
}

func proxyHandler(c *gin.Context, discovery *serviceDiscovery) {
	path := c.Param("action")
	target, ok := resolveRouteTarget(path)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "unknown service route",
			"path":  path,
		})
		return
	}

	serviceName := sharedconsul.ServiceName(target.BaseServiceName, target.Protocol)
	instance, err := discovery.resolve(target.BaseServiceName, target.Protocol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "service discovery failed",
			"service": serviceName,
			"detail":  err.Error(),
		})
		return
	}

	targetURL := instance.baseURL() + "/api/v1" + path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create upstream request",
			"service": serviceName,
			"detail":  err.Error(),
		})
		return
	}

	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":    "upstream request failed",
			"service":  serviceName,
			"upstream": instance.baseURL(),
			"detail":   err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":    "failed to stream upstream response",
			"service":  serviceName,
			"upstream": instance.baseURL(),
			"detail":   err.Error(),
		})
	}
}
