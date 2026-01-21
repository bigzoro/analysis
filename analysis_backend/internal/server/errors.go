package server

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIError 统一的 API 错误响应格式（优化：使用错误码）
type APIError struct {
	Code     string `json:"code"`               // 错误码
	Message  string `json:"message"`            // 用户友好的错误消息
	Details  string `json:"details,omitempty"`  // 详细错误信息（仅开发环境）
	TraceID  string `json:"trace_id,omitempty"` // 追踪 ID（用于日志关联）
	HTTPCode int    `json:"-"`                  // HTTP 状态码（不序列化）
}

// ErrorResponse 发送统一的错误响应（优化：支持 AppError）
func (s *Server) ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	// 生成追踪ID
	traceID := generateTraceID(c)

	// 如果 err 是 AppError，使用其信息
	var appErr *AppError
	if err != nil {
		if ae, ok := err.(*AppError); ok {
			appErr = ae
			statusCode = ae.HTTPStatus
			if ae.Message != "" {
				message = ae.Message
			}
			if ae.TraceID != "" {
				traceID = ae.TraceID
			}
		}
	}

	response := APIError{
		Code:     string(ErrorCodeInternal),
		Message:  message,
		TraceID:  traceID,
		HTTPCode: statusCode,
	}

	// 如果是 AppError，使用其错误码
	if appErr != nil {
		response.Code = string(appErr.Code)
	}

	// 开发环境显示详细错误信息
	if gin.Mode() == gin.DebugMode {
		if err != nil {
			response.Details = err.Error()
			// 在开发环境下，可以包含堆栈信息
			response.Details += "\n" + string(debug.Stack())
		}
	}

	// 记录错误日志
	s.logError(c, statusCode, message, err, traceID)

	c.JSON(statusCode, response)
}

// ==================== 错误日志记录 ====================

// logError 记录错误日志
func (s *Server) logError(c *gin.Context, statusCode int, message string, err error, traceID string) {
	// 只记录 4xx 和 5xx 错误
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

	// 根据状态码选择日志级别
	if statusCode >= 500 {
		log.Printf("%s", logMsg)
		// 生产环境可以集成更完善的日志系统（如 logrus、zap）
	} else {
		// 4xx 错误使用较低的日志级别
		if gin.Mode() == gin.DebugMode {
			log.Printf("[WARN] %s", logMsg)
		}
	}
}

// ==================== 追踪ID生成 ====================

// generateTraceID 生成追踪ID
func generateTraceID(c *gin.Context) string {
	// 尝试从请求头获取追踪ID（如果已有）
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		return traceID
	}

	// 尝试从上下文获取
	if traceID, exists := c.Get("trace_id"); exists {
		if id, ok := traceID.(string); ok && id != "" {
			return id
		}
	}

	// 生成新的追踪ID
	traceID := uuid.New().String()
	c.Set("trace_id", traceID)
	return traceID
}

// ==================== 错误处理中间件 ====================

// ErrorHandlerMiddleware 错误处理中间件（捕获 panic 和未处理的错误）
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 记录 panic 信息
				log.Printf("[PANIC] %v", r)
				if gin.Mode() == gin.DebugMode {
					log.Printf("[PANIC] Stack: %s", string(debug.Stack()))
				}

				// 返回错误响应
				c.JSON(http.StatusInternalServerError, APIError{
					Code:     string(ErrorCodeInternal),
					Message:  "服务器内部错误",
					TraceID:  generateTraceID(c),
					HTTPCode: http.StatusInternalServerError,
				})
				c.Abort()
			}
		}()

		c.Next()

		// 检查是否有未处理的错误
		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last()
			if lastErr != nil {
				// 如果错误还没有被处理，统一处理
				if !c.Writer.Written() {
					// 这里可以添加统一的错误处理逻辑
					// 暂时由各个 handler 自己处理
				}
			}
		}
	}
}

// HandleError 处理错误（统一入口）
func (s *Server) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 如果是 AppError，直接使用
	if appErr, ok := err.(*AppError); ok {
		s.ErrorResponse(c, appErr.HTTPStatus, appErr.Message, appErr)
		return
	}

	// 根据错误类型判断
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.NotFound(c, "资源不存在")
		return
	}

	// 默认作为内部错误处理
	s.InternalServerError(c, "服务器内部错误", err)
}

// BadRequest 400 错误：请求参数错误（优化：使用错误码）
func (s *Server) BadRequest(c *gin.Context, message string, err error) {
	if message == "" {
		message = "请求参数错误"
	}
	appErr := ErrInvalidInput.WithError(err).WithDetails(message)
	s.ErrorResponse(c, http.StatusBadRequest, message, appErr)
}

// Unauthorized 401 错误：未授权（优化：使用错误码）
func (s *Server) Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "未授权，请先登录"
	}
	appErr := ErrUnauthorized.WithDetails(message)
	s.ErrorResponse(c, http.StatusUnauthorized, message, appErr)
}

// Forbidden 403 错误：禁止访问（优化：使用错误码）
func (s *Server) Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "禁止访问"
	}
	appErr := ErrForbidden.WithDetails(message)
	s.ErrorResponse(c, http.StatusForbidden, message, appErr)
}

// NotFound 404 错误：资源不存在（优化：使用错误码）
func (s *Server) NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "资源不存在"
	}
	appErr := ErrNotFound.WithDetails(message)
	s.ErrorResponse(c, http.StatusNotFound, message, appErr)
}

// InternalServerError 500 错误：服务器内部错误（优化：使用错误码）
func (s *Server) InternalServerError(c *gin.Context, message string, err error) {
	if message == "" {
		message = "服务器内部错误"
	}
	appErr := ErrInternal.WithError(err).WithDetails(message)
	s.ErrorResponse(c, http.StatusInternalServerError, message, appErr)
}

// ValidationError 参数验证错误（优化：使用错误码）
func (s *Server) ValidationError(c *gin.Context, field, message string) {
	if message == "" {
		message = "参数验证失败"
	}
	if field != "" {
		message = field + ": " + message
	}
	appErr := WrapValidationError(field, message)
	s.ErrorResponse(c, http.StatusBadRequest, message, appErr)
}

// JSONBindError JSON 绑定错误（优化：使用错误码）
func (s *Server) JSONBindError(c *gin.Context, err error) {
	appErr := ErrInvalidInput.WithError(err).WithDetails("请求数据格式错误")
	s.ErrorResponse(c, http.StatusBadRequest, "请求数据格式错误", appErr)
}

// DatabaseError 数据库错误（优化：使用错误码）
func (s *Server) DatabaseError(c *gin.Context, operation string, err error) {
	appErr := WrapDatabaseError(err, operation)
	s.ErrorResponse(c, http.StatusInternalServerError, appErr.Message, appErr)
}
