package main

import (
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

const indexHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ATS 招聘管理平台</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: linear-gradient(135deg, #1976d3, #1565c0); color: white; padding: 20px; text-align: center; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header h1 { font-size: 28px; margin-bottom: 5px; }
        .header p { opacity: 0.9; }
        .status-bar { display: flex; gap: 20px; margin-bottom: 20px; flex-wrap: wrap; }
        .status-card { flex: 1; background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); min-width: 200px; }
        .status-card h3 { color: #666; font-size: 14px; margin-bottom: 5px; }
        .status-card .value { font-size: 20px; font-weight: bold; }
        .status-card.ok .value { color: #4caf50; }
        .status-card.error .value { color: #f44336; }
        .links { margin-top: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .links a { padding: 10px 20px; background: #1976d3; color: white; text-decoration: none; border-radius: 4px; }
        .links a:hover { background: #1565c0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>ATS 招聘管理平台</h1>
            <p>API Gateway - 统一入口</p>
        </div>

        <div class="status-bar">
            <div class="status-card" id="resume-status">
                <h3>简历服务</h3>
                <div class="value">检查中...</div>
            </div>
            <div class="status-card" id="interview-status">
                <h3>面试服务</h3>
                <div class="value">检查中...</div>
            </div>
            <div class="status-card" id="search-status">
                <h3>搜索服务</h3>
                <div class="value">检查中...</div>
            </div>
        </div>

        <div class="links">
            <a href="%s" target="_blank">简历管理</a>
            <a href="%s" target="_blank">面试管理</a>
            <a href="%s" target="_blank">简历搜索</a>
        </div>
    </div>

    <script>
        function checkStatus() {
            fetch('/health')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    for (var service in data.services) {
                        var card = document.getElementById(service + '-status');
                        if (!card) {
                            continue;
                        }
                        var item = data.services[service];
                        var status = item.status || 'error';
                        card.classList.remove('ok', 'error');
                        card.classList.add(status === 'ok' ? 'ok' : 'error');
                        card.querySelector('.value').textContent = status === 'ok' ? 'Online' : 'Offline';
                    }
                });
        }
        checkStatus();
        setInterval(checkStatus, 5000);
    </script>
</body>
</html>`

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
		resumeURL := discoverServiceURL(discovery, sharedconsul.ResumeServiceBaseName, sharedconsul.ProtocolHTTP)
		interviewURL := discoverServiceURL(discovery, sharedconsul.InterviewServiceBaseName, sharedconsul.ProtocolHTTP)
		searchURL := discoverServiceURL(discovery, sharedconsul.SearchServiceBaseName, sharedconsul.ProtocolHTTP)

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, fmt.Sprintf(indexHTMLTemplate, resumeURL, interviewURL, searchURL))
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
