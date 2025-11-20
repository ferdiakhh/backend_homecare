package utils

import (
	"github.com/gin-gonic/gin"
)

// Format response standar biar frontend enak bacanya
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // omitempty: kalau null, ga usah dimunculin
}

func APIResponse(c *gin.Context, code int, success bool, message string, data interface{}) {
	c.JSON(code, Response{
		Success: success,
		Message: message,
		Data:    data,
	})
}
