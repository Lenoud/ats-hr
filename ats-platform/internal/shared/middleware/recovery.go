// internal/shared/middleware/recovery.go
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/example/ats-platform/internal/shared/logger"
	"github.com/gin-gonic/gin"
)

// Recovery returns a recovery middleware
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("Panic recovered: %v\n%s", err, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}