package utils

import (
	"github.com/gin-gonic/gin"
)

// JSONResponse sends a structured JSON response
func JSONResponse(c *gin.Context, status int, data any, message string) {
	c.JSON(status, gin.H{
		"status":  status,
		"message": message,
		"data":    data,
	})
}

// JSONError sends a structured error response
func JSONError(c *gin.Context, status int, err error, message string) {
	c.JSON(status, gin.H{
		"status":  status,
		"message": message,
		"error":   err.Error(),
	})
}
