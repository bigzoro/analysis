package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ==================== 独立 Handler 的错误处理辅助函数 ====================

// ErrorResponseHelper 独立 handler 的错误响应辅助函数
// 用于不是 Server 方法的 handler 函数
func ErrorResponseHelper(c *gin.Context, statusCode int, message string, err error) {
	traceID := generateTraceID(c)
	
	response := APIError{
		Code:     string(ErrorCodeInternal),
		Message:  message,
		TraceID:  traceID,
		HTTPCode: statusCode,
	}

	// 如果是 AppError，使用其信息
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			response.Code = string(appErr.Code)
			response.HTTPCode = appErr.HTTPStatus
			if appErr.Message != "" {
				response.Message = appErr.Message
			}
			if appErr.TraceID != "" {
				response.TraceID = appErr.TraceID
			}
		}
	}

	// 开发环境显示详细错误信息
	if gin.Mode() == gin.DebugMode && err != nil {
		response.Details = err.Error()
	}

	// 记录错误日志
	logErrorHelper(c, statusCode, message, err, traceID)

	c.JSON(statusCode, response)
}

// BadRequestHelper 400 错误辅助函数
func BadRequestHelper(c *gin.Context, message string, err error) {
	if message == "" {
		message = "请求参数错误"
	}
	appErr := ErrInvalidInput.WithError(err).WithDetails(message)
	ErrorResponseHelper(c, http.StatusBadRequest, message, appErr)
}

// InternalServerErrorHelper 500 错误辅助函数
func InternalServerErrorHelper(c *gin.Context, message string, err error) {
	if message == "" {
		message = "服务器内部错误"
	}
	appErr := ErrInternal.WithError(err).WithDetails(message)
	ErrorResponseHelper(c, http.StatusInternalServerError, message, appErr)
}

// ValidationErrorHelper 验证错误辅助函数
func ValidationErrorHelper(c *gin.Context, field, message string) {
	if message == "" {
		message = "参数验证失败"
	}
	if field != "" {
		message = field + ": " + message
	}
	appErr := WrapValidationError(field, message)
	ErrorResponseHelper(c, http.StatusBadRequest, message, appErr)
}

// DatabaseErrorHelper 数据库错误辅助函数
func DatabaseErrorHelper(c *gin.Context, operation string, err error) {
	appErr := WrapDatabaseError(err, operation)
	ErrorResponseHelper(c, http.StatusInternalServerError, appErr.Message, appErr)
}

// JSONBindErrorHelper JSON 绑定错误辅助函数
func JSONBindErrorHelper(c *gin.Context, err error) {
	appErr := ErrInvalidInput.WithError(err).WithDetails("请求数据格式错误")
	ErrorResponseHelper(c, http.StatusBadRequest, "请求数据格式错误", appErr)
}

// logErrorHelper 记录错误日志（辅助函数版本）
func logErrorHelper(c *gin.Context, statusCode int, message string, err error, traceID string) {
	if statusCode < 400 {
		return
	}

	// 优化：使用字符串构建器构建日志消息
	var method, path, query string
	if c != nil {
		method = c.Request.Method
		path = c.Request.URL.Path
		query = c.Request.URL.RawQuery
	}
	logMsg := FormatErrorLog(traceID, statusCode, message, err, method, path, query)

	if statusCode >= 500 {
		log.Printf("%s", logMsg)
	} else {
		if gin.Mode() == gin.DebugMode {
			log.Printf("[WARN] %s", logMsg)
		}
	}
}

