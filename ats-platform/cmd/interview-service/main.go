package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/example/ats-platform/internal/interview/model"
)

var db *gorm.DB

// HTML 前端页面
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ATS 面试管理</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: Arial, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        h1 { color: #1976d3; margin-bottom: 20px; }
        h2 { color: #333; margin: 20px 0 10px; border-bottom: 2px solid #1976d3; padding-bottom: 5px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, select, textarea { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        button { padding: 10px 20px; background: #1976d3; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        button:hover { background: #1565c0; }
        button.danger { background: #d32f2f; }
        button.success { background: #388e3c; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { padding: 10px; border: 1px solid #ddd; text-align: left; }
        th { background: #f5f5f5; }
        .status-scheduled { color: #1976d3; }
        .status-completed { color: #388e3c; }
        .status-cancelled { color: #d32f2f; }
        .section { margin-bottom: 30px; padding: 15px; background: #fafafa; border-radius: 4px; }
        .api-result { background: #e8f5e9; padding: 10px; margin-top: 10px; border-radius: 4px; font-family: monospace; white-space: pre-wrap; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎯 ATS 面试管理系统</h1>
        <p>管理面试安排、面评和作品集</p>

        <div class="section">
            <h2>📅 创建面试</h2>
            <div class="form-group">
                <label>简历ID:</label>
                <input type="text" id="resumeId" value="550e8400-e29b-41d4-a716-446655440001">
            </div>
            <div class="form-group">
                <label>面试轮次:</label>
                <select id="round">
                    <option value="1">第一轮</option>
                    <option value="2">第二轮</option>
                    <option value="3">第三轮</option>
                </select>
            </div>
            <div class="form-group">
                <label>面试官:</label>
                <input type="text" id="interviewer" value="张经理">
            </div>
            <div class="form-group">
                <label>预约时间:</label>
                <input type="datetime-local" id="scheduledAt">
            </div>
            <button onclick="createInterview()">创建面试</button>
        </div>

        <div class="section">
            <h2>📋 查询面试</h2>
            <div class="form-group">
                <label>简历ID:</label>
                <input type="text" id="searchResumeId" value="550e8400-e29b-41d4-a716-446655440001">
                <button onclick="listInterviews()">查询面试</button>
            </div>
            <table id="interviewTable">
                <thead>
                    <tr><th>ID</th><th>轮次</th><th>面试官</th><th>预约时间</th><th>状态</th><th>操作</th></tr>
                </thead>
                <tbody></tbody>
            </table>
        </div>

        <div class="section">
            <h2>⭐ 提交面评</h2>
            <div class="form-group">
                <label>面试ID:</label>
                <input type="text" id="feedbackInterviewId">
            </div>
            <div class="form-group">
                <label>评分 (1-5):</label>
                <select id="rating">
                    <option value="5">5 - 优秀</option>
                    <option value="4">4 - 良好</option>
                    <option value="3">3 - 一般</option>
                    <option value="2">2 - 较差</option>
                    <option value="1">1 - 不合格</option>
                </select>
            </div>
            <div class="form-group">
                <label>推荐意见:</label>
                <select id="recommendation">
                    <option value="strong_yes">强烈推荐</option>
                    <option value="yes">推荐</option>
                    <option value="no">不推荐</option>
                    <option value="strong_no">强烈不推荐</option>
                </select>
            </div>
            <div class="form-group">
                <label>面评内容:</label>
                <textarea id="feedbackContent" rows="3"></textarea>
            </div>
            <button onclick="submitFeedback()">提交面评</button>
        </div>

        <div class="section">
            <h2>📁 作品集</h2>
            <div class="form-group">
                <label>简历ID:</label>
                <input type="text" id="portfolioResumeId">
            </div>
            <div class="form-group">
                <label>标题:</label>
                <input type="text" id="portfolioTitle">
            </div>
            <div class="form-group">
                <label>类型:</label>
                <select id="portfolioType">
                    <option value="pdf">PDF</option>
                    <option value="link">链接</option>
                    <option value="image">图片</option>
                </select>
            </div>
            <div class="form-group">
                <label>URL:</label>
                <input type="text" id="portfolioUrl">
            </div>
            <button onclick="createPortfolio()">添加作品集</button>
            <button onclick="listPortfolios()">查看作品集</button>
            <table id="portfolioTable">
                <thead>
                    <tr><th>ID</th><th>标题</th><th>类型</th><th>URL</th><th>操作</th></tr>
                </thead>
                <tbody></tbody>
            </table>
        </div>

        <div id="result" class="api-result" style="display:none;"></div>
    </div>

    <script>
        function showResult(data) {
            const el = document.getElementById('result');
            el.style.display = 'block';
            el.textContent = JSON.stringify(data, null, 2);
        }

        function createInterview() {
            const scheduledAtVal = document.getElementById('scheduledAt').value;
            const data = {
                resume_id: document.getElementById('resumeId').value,
                round: parseInt(document.getElementById('round').value),
                interviewer: document.getElementById('interviewer').value,
                scheduled_at: scheduledAtVal ? scheduledAtVal + ':00+08:00' : new Date().toISOString()
            };
            fetch('/api/v1/interviews', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(data)
            })
            .then(r => r.json())
            .then(d => { showResult(d); listInterviews(); })
            .catch(e => showResult({error: e.message}));
        }

        function listInterviews() {
            const resumeId = document.getElementById('searchResumeId').value;
            fetch('/api/v1/resumes/' + resumeId + '/interviews')
                .then(r => r.json())
            .then(d => {
                showResult(d);
                const tbody = document.querySelector('#interviewTable tbody');
                tbody.innerHTML = '';
                (d.data || []).forEach(i => {
                    const statusClass = 'status-' + i.status;
                    tbody.innerHTML += '<tr>' +
                        '<td>' + i.id.substring(0, 8) + '...</td>' +
                        '<td>' + i.round + '</td>' +
                        '<td>' + i.interviewer + '</td>' +
                        '<td>' + new Date(i.scheduled_at).toLocaleString() + '</td>' +
                        '<td class="' + statusClass + '">' + i.status + '</td>' +
                        '<td>' +
                            '<button class="success" onclick="completeInterview(\'' + i.id + '\')">完成</button> ' +
                            '<button class="danger" onclick="cancelInterview(\'' + i.id + '\')">取消</button> ' +
                        '<button onclick="showFeedbackForm(\'' + i.id + '\')">面评</button> ' +
                        '</td>' +
                        '</tr>';
                    });
                })
                .catch(e => showResult({error: e.message}));
        }
        function completeInterview(id) {
            fetch('/api/v1/interviews/' + id + '/status', {
                method: 'PUT',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({status: 'completed'})
            })
            .then(r => r.json())
            .then(d => { showResult(d); listInterviews(); })
            .catch(e => showResult({error: e.message}));
        }
        function cancelInterview(id) {
            fetch('/api/v1/interviews/' + id + '/status', {
                method: 'PUT',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({status: 'cancelled'})
            })
            .then(r => r.json())
            .then(d => { showResult(d); listInterviews(); })
            .catch(e => showResult({error: e.message}));
        }
        function showFeedbackForm(interviewId) {
            document.getElementById('feedbackInterviewId').value = interviewId;
        }
        function submitFeedback() {
            const interviewId = document.getElementById('feedbackInterviewId').value;
            const data = {
                rating: parseInt(document.getElementById('rating').value),
                content: document.getElementById('feedbackContent').value,
                recommendation: document.getElementById('recommendation').value
            };
            fetch('/api/v1/interviews/' + interviewId + '/feedback', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(data)
            })
            .then(r => r.json())
            .then(d => { showResult(d); })
            .catch(e => showResult({error: e.message}));
        }
        function createPortfolio() {
            const resumeId = document.getElementById('portfolioResumeId').value;
            const data = {
                title: document.getElementById('portfolioTitle').value,
                file_url: document.getElementById('portfolioUrl').value,
                file_type: document.getElementById('portfolioType').value
            };
            fetch('/api/v1/resumes/' + resumeId + '/portfolios', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(data)
            })
            .then(r => r.json())
            .then(d => { showResult(d); listPortfolios(); })
            .catch(e => showResult({error: e.message}));
        }
        function listPortfolios() {
            const resumeId = document.getElementById('portfolioResumeId').value;
            fetch('/api/v1/resumes/' + resumeId + '/portfolios')
                .then(r => r.json())
            .then(d => {
                showResult(d);
                const tbody = document.querySelector('#portfolioTable tbody');
                tbody.innerHTML = '';
                (d.data || []).forEach(p => {
                    tbody.innerHTML += '<tr>' +
                        '<td>' + p.id.substring(0, 8) + '...</td>' +
                        '<td>' + p.title + '</td>' +
                        '<td>' + p.file_type + '</td>' +
                        '<td><a href="' + p.file_url + '" target="_blank">查看</a></td>' +
                        '<td><button class="danger" onclick="deletePortfolio(\'' + p.id + '\')">删除</button></td>' +
                        '</tr>';
                    });
                })
                .catch(e => showResult({error: e.message}));
        }
        function deletePortfolio(id) {
            fetch('/api/v1/portfolios/' + id, {method: 'DELETE'})
                .then(r => r.json())
            .then(d => { showResult(d); listPortfolios(); })
            .catch(e => showResult({error: e.message}));
        }
        // 初始化
        document.getElementById('scheduledAt').value = new Date().toISOString().slice(0, 16);
    </script>
</body>
</html>`

func initDB() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		password := getEnv("DB_PASSWORD", "postgres")
		dbname := getEnv("DB_NAME", "ats")
		sslmode := getEnv("DB_SSLMODE", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// 自动迁移
	return db.AutoMigrate(&model.Interview{}, &model.Feedback{}, &model.Portfolio{})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// 初始化数据库
	if err := initDB(); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected to PostgreSQL database")

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
		sqlDB, err := db.DB()
		dbStatus := "connected"
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}
		c.JSON(200, gin.H{
			"service": "interview-service",
			"status":  "ok",
			"db":      dbStatus,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	// API 路由
	api := r.Group("/api/v1")
	{
		// 面试
		api.POST("/interviews", createInterviewHandler)
		api.GET("/interviews/:id", getInterviewHandler)
		api.PUT("/interviews/:id/status", updateInterviewStatusHandler)
		api.DELETE("/interviews/:id", deleteInterviewHandler)
		api.GET("/resumes/:id/interviews", listInterviewsByResumeHandler)
		// 面评
		api.POST("/interviews/:id/feedback", submitFeedbackHandler)
		api.GET("/interviews/:id/feedback", getFeedbackHandler)
		// 作品集
		api.POST("/resumes/:id/portfolios", createPortfolioHandler)
		api.GET("/resumes/:id/portfolios", listPortfoliosHandler)
		api.DELETE("/portfolios/:id", deletePortfolioHandler)
	}

	println("Interview Service running on http://0.0.0.0:8082")
	r.Run("0.0.0.0:8082")
}

// Handlers
func createInterviewHandler(c *gin.Context) {
	var req model.CreateInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	resumeID, err := uuid.Parse(req.ResumeID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid resume_id"})
		return
	}
	interview := &model.Interview{
		ID:          uuid.New(),
		ResumeID:    resumeID,
		Round:       req.Round,
		Interviewer: req.Interviewer,
		ScheduledAt: req.ScheduledAt,
		Status:      model.InterviewStatusScheduled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(interview).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"data": interview})
}

func getInterviewHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	var interview model.Interview
	if err := db.First(&interview, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, gin.H{"data": interview})
}

func listInterviewsByResumeHandler(c *gin.Context) {
	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid resume_id"})
		return
	}
	var interviews []model.Interview
	if err := db.Where("resume_id = ?", resumeID).Find(&interviews).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": interviews})
}

func updateInterviewStatusHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	var req model.UpdateInterviewStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	var interview model.Interview
	if err := db.First(&interview, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	interview.Status = model.InterviewStatus(req.Status)
	interview.UpdatedAt = time.Now()
	if err := db.Save(&interview).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": interview})
}

func deleteInterviewHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if err := db.Delete(&model.Interview{}, "id = ?", id).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "deleted"})
}

func submitFeedbackHandler(c *gin.Context) {
	interviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid interview_id"})
		return
	}
	var req model.SubmitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	feedback := &model.Feedback{
		ID:             uuid.New(),
		InterviewID:    interviewID,
		Rating:         req.Rating,
		Content:        req.Content,
		Recommendation: model.Recommendation(req.Recommendation),
		CreatedAt:      time.Now(),
	}
	if err := db.Create(feedback).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"data": feedback})
}

func getFeedbackHandler(c *gin.Context) {
	interviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid interview_id"})
		return
	}
	var feedback model.Feedback
	if err := db.Where("interview_id = ?", interviewID).First(&feedback).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, gin.H{"data": feedback})
}

func createPortfolioHandler(c *gin.Context) {
	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid resume_id"})
		return
	}
	var req model.CreatePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	portfolio := &model.Portfolio{
		ID:        uuid.New(),
		ResumeID:  resumeID,
		Title:     req.Title,
		FileURL:   req.FileURL,
		FileType:  model.FileType(req.FileType),
		CreatedAt: time.Now(),
	}
	if err := db.Create(portfolio).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"data": portfolio})
}

func listPortfoliosHandler(c *gin.Context) {
	resumeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid resume_id"})
		return
	}
	var portfolios []model.Portfolio
	if err := db.Where("resume_id = ?", resumeID).Find(&portfolios).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": portfolios})
}

func deletePortfolioHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if err := db.Delete(&model.Portfolio{}, "id = ?", id).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "deleted"})
}
