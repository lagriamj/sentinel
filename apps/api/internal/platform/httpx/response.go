package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes the API error envelope and stops the handler chain.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorBody{
		Error: errorDetail{Code: code, Message: message},
	})
}

// JSON writes a success response.
func JSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

// NotFound writes the response used for unknown routes.
func NotFound(c *gin.Context) {
	Error(c, http.StatusNotFound, "not_found", "resource not found")
}
