package server

import (
	"bidding-tracker/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLoggerMiddleware logs incoming requests with timing
func RequestLoggerMiddleware(c *gin.Context) {
	start := time.Now()

	c.Next() // process request

	utils.Info("HTTP Request", map[string]any{
		"method":  c.Request.Method,
		"path":    c.Request.URL.Path,
		"status":  c.Writer.Status(),
		"latency": time.Since(start).String(),
	})
}
