package server

import (
	"fmt"
	"net/http"
)

// ==================== 错误类型和错误码 ====================

// ErrorCode 错误码
type ErrorCode string

const (
	// 通用错误码
	ErrorCodeInternal           ErrorCode = "INTERNAL_ERROR"      // 内部错误
	ErrorCodeInvalidInput       ErrorCode = "INVALID_INPUT"       // 输入无效
	ErrorCodeNotFound           ErrorCode = "NOT_FOUND"           // 资源不存在
	ErrorCodeUnauthorized       ErrorCode = "UNAUTHORIZED"        // 未授权
	ErrorCodeForbidden          ErrorCode = "FORBIDDEN"           // 禁止访问
	ErrorCodeConflict           ErrorCode = "CONFLICT"            // 冲突
	ErrorCodeRateLimit          ErrorCode = "RATE_LIMIT_EXCEEDED" // 限流
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE" // 服务不可用

	// 业务错误码
	ErrorCodeDatabase   ErrorCode = "DATABASE_ERROR"    // 数据库错误
	ErrorCodeCache      ErrorCode = "CACHE_ERROR"       // 缓存错误
	ErrorCodeValidation ErrorCode = "VALIDATION_ERROR"  // 验证错误
	ErrorCodeAuth       ErrorCode = "AUTH_ERROR"        // 认证错误
	ErrorCodePermission ErrorCode = "PERMISSION_DENIED" // 权限不足
)

// AppError 应用错误
type AppError struct {
	Code       ErrorCode `json:"code"`               // 错误码
	Message    string    `json:"message"`            // 用户友好的错误消息
	Details    string    `json:"details,omitempty"`  // 详细错误信息（仅开发环境）
	HTTPStatus int       `json:"-"`                  // HTTP 状态码（不序列化）
	Err        error     `json:"-"`                  // 原始错误（不序列化）
	TraceID    string    `json:"trace_id,omitempty"` // 追踪ID
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建应用错误
func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// WithError 添加原始错误
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// WithDetails 添加详细信息
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithTraceID 添加追踪ID
func (e *AppError) WithTraceID(traceID string) *AppError {
	e.TraceID = traceID
	return e
}

// ==================== 预定义错误 ====================

var (
	// 通用错误
	ErrInternal           = NewAppError(ErrorCodeInternal, "服务器内部错误", http.StatusInternalServerError)
	ErrInvalidInput       = NewAppError(ErrorCodeInvalidInput, "请求参数错误", http.StatusBadRequest)
	ErrNotFound           = NewAppError(ErrorCodeNotFound, "资源不存在", http.StatusNotFound)
	ErrUnauthorized       = NewAppError(ErrorCodeUnauthorized, "未授权，请先登录", http.StatusUnauthorized)
	ErrForbidden          = NewAppError(ErrorCodeForbidden, "禁止访问", http.StatusForbidden)
	ErrConflict           = NewAppError(ErrorCodeConflict, "资源冲突", http.StatusConflict)
	ErrRateLimit          = NewAppError(ErrorCodeRateLimit, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
	ErrServiceUnavailable = NewAppError(ErrorCodeServiceUnavailable, "服务暂时不可用", http.StatusServiceUnavailable)

	// 业务错误
	ErrDatabase   = NewAppError(ErrorCodeDatabase, "数据库操作失败", http.StatusInternalServerError)
	ErrCache      = NewAppError(ErrorCodeCache, "缓存操作失败", http.StatusInternalServerError)
	ErrValidation = NewAppError(ErrorCodeValidation, "参数验证失败", http.StatusBadRequest)
	ErrAuth       = NewAppError(ErrorCodeAuth, "认证失败", http.StatusUnauthorized)
	ErrPermission = NewAppError(ErrorCodePermission, "权限不足", http.StatusForbidden)
)

// ==================== 错误包装函数 ====================

// WrapError 包装错误为 AppError
func WrapError(err error, code ErrorCode, message string, httpStatus int) *AppError {
	if err == nil {
		return nil
	}

	appErr, ok := err.(*AppError)
	if ok {
		return appErr
	}

	return NewAppError(code, message, httpStatus).WithError(err)
}

// WrapDatabaseError 包装数据库错误
func WrapDatabaseError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}

	message := "数据库操作失败"
	if operation != "" {
		message = operation + "失败"
	}

	return WrapError(err, ErrorCodeDatabase, message, http.StatusInternalServerError)
}

// WrapValidationError 包装验证错误
func WrapValidationError(field, message string) *AppError {
	if message == "" {
		message = "参数验证失败"
	}
	if field != "" {
		message = field + ": " + message
	}
	return NewAppError(ErrorCodeValidation, message, http.StatusBadRequest)
}

// WrapNotFoundError 包装未找到错误
func WrapNotFoundError(resource string) *AppError {
	message := "资源不存在"
	if resource != "" {
		message = resource + "不存在"
	}
	return NewAppError(ErrorCodeNotFound, message, http.StatusNotFound)
}

// IsAppError 检查是否为 AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError 转换为 AppError
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}
