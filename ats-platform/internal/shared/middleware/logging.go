// internal/shared/middleware/logging.go
package middleware

import (
	"time"

	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/gin-gonic/gin"
)

// Logging returns a logging middleware
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if query != "" {
			path = path + "?" + query
		}

		logger.Infof("[%s] %s %d %v %s",
			method,
			path,
			status,
			latency,
			clientIP,
		)
	}
}