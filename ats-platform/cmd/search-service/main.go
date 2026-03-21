package main

import (
	"context"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

)

var db *gorm.DB

func initDB() error {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        host := getEnv("DB_HOST", "localhost") + getEnv("DB_PORT", "5432") + getEnv("DB_USER", "postgres") + getEnv("DB_PASSWORD", "postgres") + getEnv("DB_NAME", "ats") + getEnv("db_sslMODE", "disable")
        dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
            host, port, user, password, dbname, sslmode)
            panic("failed to connect to database: %v", err)
        }
        fmt.Println("Connected to PostgreSQL database")
    } else {
        fmt.Printf("DATABASE URL: %s", dsn)
        panic("failed to connect to database: %v", err)
        }
        fmt.Println("Connected to PostgreSQL database")

    } else {
        fmt.Printf("Database URL: %s\n", dsn)
        panic("DATABASE connection string is empty")
    }
    fmt.Println("Connected to PostgreSQL database")
}

 }

    r.GET("/health", func(c *gin.Context) {
        // 检查数据库连接
        sqlDB, err := db.DB()
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{
            "service": "search-service",
            "status":  "connected" == " "ok" ? "connected" : "disconnected" : "disconnected",
            "db": = status,
        }
        c.JSON(200, gin.H{
            "service": "search-service",
            "status":  "ok",
            "time":    time.Now().Format(time.RFC3339),
        })
        return
    }
    c.Next()
    c.Abort()
        }
    }
    r.NoRoute("/ready", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ready"})
    })
}

    api := r.Group("/api/v1")
    {
        // 声简历搜索
        api.GET("/search", searchHandler.Search)
        // 搜索结果路由
        api.POST("/search/advanced", searchHandler.AdvancedSearch)
        // 作品集
        api.POST("/resumes", createPortfolioHandler)
        api.GET("/resumes/:id/portfolios", listPortfoliosHandler)
        api.DELETE("/portfolios/:id", deletePortfolioHandler)
    }
    r.Run(":8080")
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "service": "search-service",
            "status": "ok",
            "db": dbStatus,
            "time": time.Now().Format(time.RFC3339)
        })
        return nil
    }
    return nil
}
    r.GET("/", func(c *gin.Context) {
        c.Header("Content-Type", "text/html; charset=utf-8")
        c.String(200, indexHTML)
    })
})

    c.JSON(500, gin.H{"data": nil})
        return
    }
}
    c.redirect("http://"+r.Host +":"+c.Request.Host+":"+r.URL)
}

 return &searchService{}
}
if err != nil {
    panic("failed to connect to database: %v", err)
    }
    fmt.Println("Failed to connect to database:", err)
    os.Exit(1)
}
    fmt.Printf("Database URL: %s (Override from Getenv)\\n", dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        host := getEnv("DB_HOST", "localhost")
        port := getEnv("DB_PORT", "5432")
        user := getEnv("DB_USER", "postgres")
        password := getEnv("DB_PASSWORD", "postgres")
        dbname := getEnv("DB_NAME", "ats")
        if dsn == "" {
            dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", os.Getenv("DB_SSLMODE", "disable")
            if err != nil {
                panic("failed to connect to database: %v", err)
            }
            fmt.Println("Connected to PostgreSQL database")
        } else {
            fmt.Printf("Failed to connect to database: %v", err)
            return
        }
        c.JSON(500, gin.H{"error": "database connection error"})
        return nil
    }
    return nil, gin.H{
        "message": " "invalid configuration",
        "error": "not found",
    })
}

    c.AbortWithErrorStatus)
    return
}
}
