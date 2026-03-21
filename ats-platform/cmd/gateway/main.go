package main

import (
	"fmt"
	"net/http"
	"strings"
	 "time"

    "github.com/gin-gonic/gin"
)

// 后端服务配置
var services = map[string]string{
    "resume":    "http://localhost:8081",
    "interview": "http://localhost:8082",
    "search":    "http://localhost:8083",
}

// 检查服务状态
func checkServiceStatus(serviceURL string) string {
    client := http.Client{Timeout: 2 * time.Second}
    resp, err := http.Get(serviceURL + "/health")
    if err != nil {
        return "offline"
    }
    defer resp.Body.Close()
    return "ok"
}

// HTML 前端页面
const indexHTML = `<!DOCTYPE html>
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
            <a href="http://localhost:8081" target="_blank">简历管理</a>
            <a href="http://localhost:8082" target="_blank">面试管理</a>
            <a href="http://localhost:8083" target="_blank">简历搜索</a>
        </div>
    </div>

    <script>
        function checkStatus() {
            fetch('/health')
                .then(r => r.json())
                .then(data => {
                    for (var service in data.services) {
                        var card = document.getElementById(service + '-status');
                        if (card) {
                            var status = data.services[service];
                            card.classList.remove('ok', 'error');
                            card.classList.add(status);
                            card.querySelector('.value').textContent = status === 'ok' ? 'Online' : 'Offline';
                        }
                    }
                });
        }
        checkStatus();
        setInterval(checkStatus, 5000);
    </script>
</body>
</html>`

func main() {
    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    // CORS
    r.Use(func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    })

    // 首页
    r.GET("/", func(c *gin.Context) {
        c.Header("Content-Type", "text/html; charset=utf-8")
        c.String(200, indexHTML)
    })

    // 健康检查
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "service": "api-gateway",
            "status":  "ok",
            "services": gin.H{
                "resume":    checkServiceStatus(services["resume"]),
                "interview": checkServiceStatus(services["interview"]),
                "search":    checkServiceStatus(services["search"]),
            },
            "time": time.Now().Format(time.RFC3339),
        })
    })

    // API 代理
    api := r.Group("/api/v1")
    {
        // Resume Service
        api.Any("/resumes", func(c *gin.Context) {
            proxyRequest(c, services["resume"])
        })
        api.Any("/resumes/*id", func(c *gin.Context) {
            proxyRequest(c, services["resume"])
        })

        // Interview Service
        api.Any("/interviews", func(c *gin.Context) {
            proxyRequest(c, services["interview"])
        })
        api.Any("/interviews/*id", func(c *gin.Context) {
            proxyRequest(c, services["interview"])
        })
        api.Any("/portfolios/*id", func(c *gin.Context) {
            proxyRequest(c, services["interview"])
        })

        // Search Service
        api.Any("/search", func(c *gin.Context) {
            proxyRequest(c, services["search"])
        })
        api.Any("/search/*id", func(c *gin.Context) {
            proxyRequest(c, services["search"])
        })
    }

    fmt.Println("API Gateway running on http://0.0.0.0:8080")
    r.Run("0.0.0.0:8080")
}

