package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code    Code        `json:"code"`
	Message string      `json:"message"`
	Errors  []ErrorItem `json:"errors,omitempty"`
}

func Success(c *gin.Context, message string, data any) {
	if message == "" {
		message = "success"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    CodeSuccess,
		"message": message,
		"data":    data,
	})
}

func Error(c *gin.Context, status int, code Code, message string, errors ...ErrorItem) {
	resp := ErrorResponse{
		Code:    code,
		Message: message,
	}
	if len(errors) > 0 {
		resp.Errors = errors
	}
	c.JSON(status, resp)
}

func BadRequest(c *gin.Context, message string, errors ...ErrorItem) {
	Error(c, http.StatusBadRequest, CodeBadRequest, message, errors...)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

func TokenExpired(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, CodeTokenExpired, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, CodeForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, CodeConflict, message)
}

func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, CodeTooManyRequests, message)
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeInternal, message)
}
