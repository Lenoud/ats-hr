package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/ats-platform/internal/search/handler"
	"github.com/example/ats-platform/internal/search/model"
	"github.com/example/ats-platform/internal/search/repository"
	"github.com/example/ats-platform/internal/search/service"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	esRepo := repository.NewMockRepository()
	searchSvc := service.NewSearchService(esRepo)
	searchHandler := handler.NewSearchHandler(searchSvc)

	seedTestData(esRepo)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "search-service",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "search-service",
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, indexHTML)
	})

	api := router.Group("/api/v1")
	{
		api.GET("/search", searchHandler.Search)
		api.POST("/search/advanced", searchHandler.AdvancedSearch)
		api.POST("/resumes", createResumeHandler(esRepo))
	}

	addr := "0.0.0.0:8083"
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		fmt.Printf("Search Service running at http://%s\n", addr)
		fmt.Printf("Test page: http://%s/\n", addr)
		fmt.Printf("API: http://%s/api/v1/search\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	fmt.Println("Server stopped")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func createResumeHandler(repo *repository.MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ResumeID        string   `json:"resume_id"`
			Name            string   `json:"name"`
			Email           string   `json:"email"`
			Skills          []string `json:"skills"`
			ExperienceYears int      `json:"experience_years"`
			Education       string   `json:"education"`
			WorkHistory     string   `json:"work_history"`
			Status          string   `json:"status"`
			Source          string   `json:"source"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		doc := &model.ResumeDocument{
			ResumeID:        req.ResumeID,
			Name:            req.Name,
			Email:           req.Email,
			Skills:          req.Skills,
			ExperienceYears: req.ExperienceYears,
			Education:       req.Education,
			WorkHistory:     req.WorkHistory,
			Status:          req.Status,
			Source:          req.Source,
			CreatedAt:       time.Now(),
		}

		if err := repo.Index(c.Request.Context(), doc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"code": 0,
			"data": doc,
			"msg":  "indexed",
		})
	}
}

func seedTestData(repo *repository.MockRepository) {
	testData := []*model.ResumeDocument{
		{ResumeID: "r001", Name: "Zhang San", Email: "zhangsan@example.com", Skills: []string{"Go", "MySQL", "Redis"}, ExperienceYears: 5, Education: "Bachelor", Status: "parsed", Source: "LinkedIn", CreatedAt: time.Now()},
		{ResumeID: "r002", Name: "Li Si", Email: "lisi@example.com", Skills: []string{"Java", "Spring", "K8s"}, ExperienceYears: 8, Education: "Master", Status: "parsed", Source: "Boss", CreatedAt: time.Now()},
		{ResumeID: "r003", Name: "Wang Wu", Email: "wangwu@example.com", Skills: []string{"Python", "TensorFlow"}, ExperienceYears: 3, Education: "PhD", Status: "pending", Source: "LinkedIn", CreatedAt: time.Now()},
		{ResumeID: "r004", Name: "Zhao Liu", Email: "zhaoliu@example.com", Skills: []string{"React", "Vue", "TypeScript"}, ExperienceYears: 4, Education: "Bachelor", Status: "parsed", Source: "Boss", CreatedAt: time.Now()},
		{ResumeID: "r005", Name: "Sun Qi", Email: "sunqi@example.com", Skills: []string{"Go", "gRPC", "PostgreSQL"}, ExperienceYears: 6, Education: "Bachelor", Status: "parsed", Source: "Liepin", CreatedAt: time.Now()},
	}
	for _, doc := range testData {
		_ = repo.Index(context.Background(), doc)
	}
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>ATS Search Service</title>
    <style>
        body { font-family: sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1000px; margin: 0 auto; }
        .card { background: white; padding: 20px; margin: 10px 0; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        h1 { color: #333; }
        h2 { color: #555; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        input, select { padding: 8px; margin: 5px 0; width: 100%; border: 1px solid #ddd; border-radius: 4px; }
        button { padding: 10px 20px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        button:hover { background: #0056b3; }
        .result { background: #fafafa; padding: 15px; margin: 10px 0; border-left: 4px solid #007bff; }
        .skill { display: inline-block; background: #e3f2fd; color: #1e40af; padding: 2px 8px; margin: 2px; border-radius: 4px; font-size: 12px; }
        .form-row { display: flex; gap: 10px; }
        .form-row > div { flex: 1; }
    </style>
</head>
<body>
    <div class="container">
        <h1>ATS Search Service</h1>
        <div class="card">
            <h2>Search Resumes</h2>
            <div class="form-row">
                <div><input type="text" id="q" placeholder="Search by name or keyword"></div>
                <div><select id="status"><option value="">All Status</option><option value="pending">Pending</option><option value="parsed">Parsed</option></select></div>
                <div><select id="source"><option value="">All Sources</option><option value="LinkedIn">LinkedIn</option><option value="Boss">Boss</option><option value="Liepin">Liepin</option></select></div>
            </div>
            <button onclick="search()">Search</button>
            <button onclick="clearForm()" style="background:#6c757d">Clear</button>
        </div>
        <div class="card">
            <h2>Results <span id="total"></span></h2>
            <div id="results"></div>
        </div>
        <div class="card">
            <h2>Add Resume (Test)</h2>
            <div class="form-row">
                <div><input type="text" id="newId" placeholder="Resume ID"></div>
                <div><input type="text" id="newName" placeholder="Name"></div>
                <div><input type="text" id="newEmail" placeholder="Email"></div>
            </div>
            <div class="form-row">
                <div><input type="text" id="newSkills" placeholder="Skills (comma separated)"></div>
                <div><input type="number" id="newExp" placeholder="Years"></div>
                <div><select id="newEdu"><option value="Bachelor">Bachelor</option><option value="Master">Master</option><option value="PhD">PhD</option></select></div>
            </div>
            <button onclick="addResume()">Add Resume</button>
        </div>
    </div>
    <script>
        let page = 1;
        async function search() {
            const params = new URLSearchParams();
            if (document.getElementById('q').value) params.append('query', document.getElementById('q').value);
            if (document.getElementById('status').value) params.append('status', document.getElementById('status').value);
            if (document.getElementById('source').value) params.append('source', document.getElementById('source').value);
            params.append('page', page);
            params.append('page_size', 10);
            const res = await fetch('/api/v1/search?' + params);
            const data = await res.json();
            showResults(data);
        }
        function showResults(data) {
            const results = document.getElementById('results');
            const total = document.getElementById('total');
            if (data.code !== 0) { results.innerHTML = '<p style="color:red">Error: ' + data.msg + '</p>'; return; }
            const docs = data.data.list || [];
            total.textContent = '(' + data.data.total + ' total)';
            if (docs.length === 0) { results.innerHTML = '<p>No results found</p>'; return; }
            let html = '';
            docs.forEach(d => {
                html += '<div class="result"><strong>' + (d.name || 'Unknown') + '</strong> (' + (d.experience_years || 0) + ' years, ' + (d.education || 'N/A') + ')<br>';
                html += '<small>' + (d.email || '') + ' | ' + (d.source || '') + ' | ' + (d.status || '') + '</small><br>';
                if (d.skills) html += d.skills.map(s => '<span class="skill">' + s + '</span>').join('');
                html += '</div>';
            });
            results.innerHTML = html;
        }
        function clearForm() {
            document.getElementById('q').value = '';
            document.getElementById('status').value = '';
            document.getElementById('source').value = '';
            page = 1;
            search();
        }
        async function addResume() {
            const doc = {
                resume_id: document.getElementById('newId').value,
                name: document.getElementById('newName').value,
                email: document.getElementById('newEmail').value,
                skills: document.getElementById('newSkills').value.split(',').map(s => s.trim()),
                experience_years: parseInt(document.getElementById('newExp').value) || 0,
                education: document.getElementById('newEdu').value,
                status: 'parsed',
                source: 'Manual'
            };
            if (!doc.resume_id || !doc.name) { alert('Please enter ID and Name'); return; }
            const res = await fetch('/api/v1/resumes', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(doc) });
            const data = await res.json();
            if (data.code === 0) { alert('Added!'); search(); }
            else alert('Error: ' + data.msg);
        }
        search();
    </script>
</body>
</html>`
