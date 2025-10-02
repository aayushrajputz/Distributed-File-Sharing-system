package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		log.Printf("[%s] %s %s | Status: %d | Latency: %v | IP: %s | User-Agent: %s",
			c.Request.Method,
			path,
			query,
			c.Writer.Status(),
			latency,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}
}
