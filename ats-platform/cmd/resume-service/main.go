package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 配置
type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func loadConfig() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8081"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "ats"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// Resume 简历模型
type Resume struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Email     string    `gorm:"size:100" json:"email"`
	Phone     string    `gorm:"size:20" json:"phone"`
	Source    string    `gorm:"size:50" json:"source"`
	Status    string    `gorm:"size:20;default:pending" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Resume) TableName() string {
	return "resumes"
}

// CreateResumeRequest 创建简历请求
type CreateResumeRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Source string `json:"source"`
}

// UpdateResumeRequest 更新简历请求
type UpdateResumeRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

// Handler
type ResumeHandler struct {
	db *gorm.DB
}

func NewResumeHandler(db *gorm.DB) *ResumeHandler {
	return &ResumeHandler{db: db}
}

func (h *ResumeHandler) Create(c *gin.Context) {
	var req CreateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resume := &Resume{
		ID:        uuid.New(),
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Source:    req.Source,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.Create(resume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": resume})
}

func (h *ResumeHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var resume Resume
	if err := h.db.Where("id = ?", id).First(&resume).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resume})
}

func (h *ResumeHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var resumes []Resume
	var total int64

	h.db.Model(&Resume{}).Count(&total)
	h.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&resumes)

	c.JSON(http.StatusOK, gin.H{"data": resumes, "total": total})
}

func (h *ResumeHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var resume Resume
	if err := h.db.Where("id = ?", id).First(&resume).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req UpdateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	h.db.Model(&resume).Updates(updates)
	h.db.Where("id = ?", id).First(&resume)

	c.JSON(http.StatusOK, gin.H{"data": resume})
}

func (h *ResumeHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result := h.db.Where("id = ?", id).Delete(&Resume{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// HTML 前端页面
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ATS 简历管理</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: Arial, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #1976d3; margin-bottom: 10px; }
        h2 { color: #333; margin: 20px 0 10px; border-bottom: 2px solid #1976d3; padding-bottom: 5px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; color: #555; }
        input, select { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
        button { padding: 10px 20px; background: #1976d3; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 5px 5px 5px 0; }
        button:hover { background: #1565c0; }
        button.danger { background: #d32f2f; }
        button.danger:hover { background: #c62828; }
        button.success { background: #388e3c; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 12px; border: 1px solid #ddd; text-align: left; }
        th { background: #f5f5f5; font-weight: bold; }
        .status-pending { color: #ff9800; font-weight: bold; }
        .status-reviewing { color: #1976d3; font-weight: bold; }
        .status-interviewed { color: #388e3c; font-weight: bold; }
        .status-hired { color: #4caf50; font-weight: bold; }
        .status-rejected { color: #d32f2f; font-weight: bold; }
        .badge { padding: 3px 8px; border-radius: 12px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📄 ATS 简历管理</h1>
        <p style="color: #666; margin-bottom: 20px;">简历列表和管理 - PostgreSQL版本</p>

        <h2>创建简历</h2>
        <form id="createForm">
            <div class="form-group">
                <label>姓名:</label>
                <input type="text" id="name" required placeholder="请输入姓名">
            </div>
            <div class="form-group">
                <label>邮箱:</label>
                <input type="email" id="email" required placeholder="请输入邮箱">
            </div>
            <div class="form-group">
                <label>电话:</label>
                <input type="tel" id="phone" placeholder="请输入电话">
            </div>
            <div class="form-group">
                <label>来源:</label>
                <select id="source">
                    <option value="boss">Boss直聘</option>
                    <option value="lagou">拉勾</option>
                    <option value="zhilian">智联招聘</option>
                    <option value="other">其他</option>
                </select>
            </div>
            <button type="submit">创建简历</button>
        </form>

        <h2>简历列表</h2>
        <table id="resumeTable">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>姓名</th>
                    <th>邮箱</th>
                    <th>电话</th>
                    <th>来源</th>
                    <th>状态</th>
                    <th>操作</th>
                </tr>
            </thead>
            <tbody></tbody>
        </table>
    </div>

    <script>
        function loadResumes() {
            fetch('/api/v1/resumes')
                .then(r => r.json())
                .then(d => {
                    const tbody = document.querySelector('#resumeTable tbody');
                    tbody.innerHTML = '';
                    (d.data || []).forEach(r => {
                        tbody.innerHTML += '<tr>' +
                            '<td>' + r.id.substring(0,8) + '...</td>' +
                            '<td>' + r.name + '</td>' +
                            '<td>' + r.email + '</td>' +
                            '<td>' + (r.phone || '-') + '</td>' +
                            '<td>' + r.source + '</td>' +
                            '<td class="status-' + r.status + '">' + r.status + '</td>' +
                            '<td>' +
                                '<button onclick="viewResume(\'' + r.id + '\')">查看</button> ' +
                                '<button class="danger" onclick="deleteResume(\'' + r.id + '\')">删除</button>' +
                            '</td>' +
                            '</tr>';
                    });
                })
                .catch(e => console.error('Error:', e));
        }

        document.getElementById('createForm').addEventListener('submit', function(e) {
            e.preventDefault();
            const data = {
                name: document.getElementById('name').value,
                email: document.getElementById('email').value,
                phone: document.getElementById('phone').value,
                source: document.getElementById('source').value
            };
            fetch('/api/v1/resumes', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(data)
            })
            .then(r => r.json())
            .then(d => {
                if (d.data) {
                    alert('简历创建成功!');
                    document.getElementById('createForm').reset();
                    loadResumes();
                } else {
                    alert('创建失败: ' + (d.error || '未知错误'));
                }
            });
        });

        function viewResume(id) {
            fetch('/api/v1/resumes/' + id)
                .then(r => r.json())
                .then(d => {
                    alert('简历详情:\n' + JSON.stringify(d.data, null, 2));
                });
        }

        function deleteResume(id) {
            if (confirm('确定删除该简历?')) {
                fetch('/api/v1/resumes/' + id, { method: 'DELETE' })
                    .then(r => r.json())
                    .then(d => {
                        alert('删除成功');
                        loadResumes();
                    });
            }
        }

        // 初始加载
        loadResumes();
    </script>
</body>
</html>`

func main() {
	config := loadConfig()

	// 连接数据库
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database: %v", err))
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("Failed to get DB: %v", err))
	}
	defer sqlDB.Close()

	fmt.Println("✅ Connected to PostgreSQL database")

	// 创建 Handler
	handler := NewResumeHandler(db)

	// 设置 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
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
		// 检查数据库连接
		dbStatus := "ok"
		if err := sqlDB.Ping(); err != nil {
			dbStatus = "error: " + err.Error()
		}
		c.JSON(200, gin.H{
			"service": "resume-service",
			"status":  "ok",
			"db":      dbStatus,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/ready", func(c *gin.Context) {
		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		api.POST("/resumes", handler.Create)
		api.GET("/resumes/:id", handler.GetByID)
		api.GET("/resumes", handler.List)
		api.PUT("/resumes/:id", handler.Update)
		api.DELETE("/resumes/:id", handler.Delete)
	}

	addr := "0.0.0.0:" + config.ServerPort
	fmt.Printf("🚀 Resume Service running on http://%s\n", addr)
	fmt.Printf("   Database: %s@%s:%s/%s\n", config.DBUser, config.DBHost, config.DBPort, config.DBName)

	if err := r.Run(addr); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
