package main

import (
	"fmt"
	"strings"
	"time"
)

// 标准错误类型定义
type ErrorType string

const (
	ErrorTypeNetwork     ErrorType = "network"
	ErrorTypeAPI         ErrorType = "api"
	ErrorTypeData        ErrorType = "data"
	ErrorTypeConfig      ErrorType = "config"
	ErrorTypeSystem      ErrorType = "system"
	ErrorTypeValidation  ErrorType = "validation"
	ErrorTypeTimeout     ErrorType = "timeout"
	ErrorTypeRateLimit   ErrorType = "rate_limit"
	ErrorTypeAuth        ErrorType = "auth"
	ErrorTypeUnknown     ErrorType = "unknown"
)

// 标准错误码定义
type ErrorCode string

const (
	// 网络相关错误
	ErrCodeNetworkTimeout     ErrorCode = "NETWORK_TIMEOUT"
	ErrCodeNetworkUnreachable ErrorCode = "NETWORK_UNREACHABLE"
	ErrCodeNetworkDNS         ErrorCode = "NETWORK_DNS"

	// API相关错误
	ErrCodeAPIInvalidRequest  ErrorCode = "API_INVALID_REQUEST"
	ErrCodeAPIUnauthorized    ErrorCode = "API_UNAUTHORIZED"
	ErrCodeAPIRateLimited     ErrorCode = "API_RATE_LIMITED"
	ErrCodeAPIServerError     ErrorCode = "API_SERVER_ERROR"

	// 数据相关错误
	ErrCodeDataNotFound       ErrorCode = "DATA_NOT_FOUND"
	ErrCodeDataInvalid        ErrorCode = "DATA_INVALID"
	ErrCodeDataCorrupted      ErrorCode = "DATA_CORRUPTED"

	// 配置相关错误
	ErrCodeConfigMissing      ErrorCode = "CONFIG_MISSING"
	ErrCodeConfigInvalid      ErrorCode = "CONFIG_INVALID"

	// 系统相关错误
	ErrCodeSystemResource     ErrorCode = "SYSTEM_RESOURCE"
	ErrCodeSystemInternal     ErrorCode = "SYSTEM_INTERNAL"

	// 验证相关错误
	ErrCodeValidationRequired ErrorCode = "VALIDATION_REQUIRED"
	ErrCodeValidationFormat   ErrorCode = "VALIDATION_FORMAT"

	// 其他错误
	ErrCodeUnknown            ErrorCode = "UNKNOWN"
)

// StandardizedError 标准化的错误结构
type StandardizedError struct {
	Type        ErrorType   `json:"type"`
	Code        ErrorCode   `json:"code"`
	Message     string      `json:"message"`
	Details     string      `json:"details,omitempty"`
	Component   string      `json:"component"`
	Operation   string      `json:"operation"`
	Severity    string      `json:"severity"` // "low", "medium", "high", "critical"
	Timestamp   string      `json:"timestamp"`
	Retryable   bool        `json:"retryable"`
	Underlying  error       `json:"-"` // 底层错误，不序列化
}

// Error 实现error接口
func (e *StandardizedError) Error() string {
	return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Code, e.Message, e.Details)
}

// NewStandardizedError 创建标准化的错误
func NewStandardizedError(errType ErrorType, code ErrorCode, message, details, component, operation string) *StandardizedError {
	return &StandardizedError{
		Type:      errType,
		Code:      code,
		Message:   message,
		Details:   details,
		Component: component,
		Operation: operation,
		Severity:  getSeverityForErrorType(errType, code),
		Timestamp: fmt.Sprintf("%d", getCurrentTimestamp()),
		Retryable: isRetryableError(code),
	}
}

// NewStandardizedErrorFromError 从现有错误创建标准化的错误
func NewStandardizedErrorFromError(err error, errType ErrorType, component, operation string) *StandardizedError {
	if err == nil {
		return nil
	}

	code := classifyErrorCode(err)
	message := getErrorMessage(code)
	details := err.Error()

	return &StandardizedError{
		Type:       errType,
		Code:       code,
		Message:    message,
		Details:    details,
		Component:  component,
		Operation:  operation,
		Severity:   getSeverityForErrorType(errType, code),
		Timestamp:  fmt.Sprintf("%d", getCurrentTimestamp()),
		Retryable:  isRetryableError(code),
		Underlying: err,
	}
}

// classifyErrorCode 根据错误内容分类错误码
func classifyErrorCode(err error) ErrorCode {
	if err == nil {
		return ErrCodeUnknown
	}

	errStr := strings.ToLower(err.Error())

	// 网络相关
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return ErrCodeNetworkTimeout
	}
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return ErrCodeNetworkUnreachable
	}
	if strings.Contains(errStr, "dns") {
		return ErrCodeNetworkDNS
	}

	// API相关
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "way too many requests") {
		return ErrCodeAPIRateLimited
	}
	if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "-1001") {
		return ErrCodeAPIUnauthorized
	}
	if strings.Contains(errStr, "invalid") && strings.Contains(errStr, "request") {
		return ErrCodeAPIInvalidRequest
	}
	if strings.Contains(errStr, "server error") || strings.Contains(errStr, "500") {
		return ErrCodeAPIServerError
	}

	// 数据相关
	if strings.Contains(errStr, "not found") {
		return ErrCodeDataNotFound
	}
	if strings.Contains(errStr, "invalid") && strings.Contains(errStr, "data") {
		return ErrCodeDataInvalid
	}

	// 默认未知错误
	return ErrCodeUnknown
}

// getErrorMessage 根据错误码获取标准消息
func getErrorMessage(code ErrorCode) string {
	messages := map[ErrorCode]string{
		ErrCodeNetworkTimeout:     "Network request timed out",
		ErrCodeNetworkUnreachable: "Network destination unreachable",
		ErrCodeNetworkDNS:         "DNS resolution failed",

		ErrCodeAPIInvalidRequest:  "Invalid API request",
		ErrCodeAPIUnauthorized:    "API authentication failed",
		ErrCodeAPIRateLimited:     "API rate limit exceeded",
		ErrCodeAPIServerError:     "API server error",

		ErrCodeDataNotFound:       "Requested data not found",
		ErrCodeDataInvalid:        "Data validation failed",
		ErrCodeDataCorrupted:      "Data corruption detected",

		ErrCodeConfigMissing:      "Required configuration missing",
		ErrCodeConfigInvalid:      "Configuration validation failed",

		ErrCodeSystemResource:     "System resource exhausted",
		ErrCodeSystemInternal:     "Internal system error",

		ErrCodeValidationRequired: "Required field missing",
		ErrCodeValidationFormat:   "Invalid data format",

		ErrCodeUnknown:            "Unknown error occurred",
	}

	if msg, exists := messages[code]; exists {
		return msg
	}
	return "Unknown error occurred"
}

// getSeverityForErrorType 根据错误类型和错误码确定严重程度
func getSeverityForErrorType(errType ErrorType, code ErrorCode) string {
	// 关键错误
	if code == ErrCodeAPIServerError || code == ErrCodeSystemInternal {
		return "critical"
	}

	// 高严重程度错误
	if code == ErrCodeAPIRateLimited || code == ErrCodeNetworkUnreachable ||
		code == ErrCodeDataCorrupted || code == ErrCodeConfigMissing {
		return "high"
	}

	// 中等严重程度错误
	if code == ErrCodeAPIUnauthorized || code == ErrCodeNetworkTimeout ||
		code == ErrCodeDataInvalid || code == ErrCodeConfigInvalid {
		return "medium"
	}

	// 低严重程度错误
	return "low"
}

// isRetryableError 判断错误是否可以重试
func isRetryableError(code ErrorCode) bool {
	nonRetryable := []ErrorCode{
		ErrCodeAPIInvalidRequest,
		ErrCodeAPIUnauthorized,
		ErrCodeConfigMissing,
		ErrCodeConfigInvalid,
		ErrCodeValidationRequired,
		ErrCodeValidationFormat,
		ErrCodeDataNotFound,
	}

	for _, nonRetry := range nonRetryable {
		if code == nonRetry {
			return false
		}
	}

	return true
}

// getCurrentTimestamp 获取当前时间戳（毫秒）
func getCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}

// Error helpers - 便捷的错误创建函数

// NewNetworkError 创建网络错误
func NewNetworkError(operation, details string, err error) *StandardizedError {
	return NewStandardizedErrorFromError(err, ErrorTypeNetwork, "network", operation).WithDetails(details)
}

// NewAPIError 创建API错误
func NewAPIError(operation, details string, err error) *StandardizedError {
	return NewStandardizedErrorFromError(err, ErrorTypeAPI, "api", operation).WithDetails(details)
}

// NewDataError 创建数据错误
func NewDataError(operation, details string, err error) *StandardizedError {
	return NewStandardizedErrorFromError(err, ErrorTypeData, "data", operation).WithDetails(details)
}

// NewConfigError 创建配置错误
func NewConfigError(operation, details string, err error) *StandardizedError {
	return NewStandardizedErrorFromError(err, ErrorTypeConfig, "config", operation).WithDetails(details)
}

// NewSystemError 创建系统错误
func NewSystemError(operation, details string, err error) *StandardizedError {
	return NewStandardizedErrorFromError(err, ErrorTypeSystem, "system", operation).WithDetails(details)
}

// WithDetails 添加额外详情
func (e *StandardizedError) WithDetails(details string) *StandardizedError {
	if e.Details != "" {
		e.Details += ": " + details
	} else {
		e.Details = details
	}
	return e
}

// WithComponent 设置组件名
func (e *StandardizedError) WithComponent(component string) *StandardizedError {
	e.Component = component
	return e
}

// WithSeverity 设置严重程度
func (e *StandardizedError) WithSeverity(severity string) *StandardizedError {
	e.Severity = severity
	return e
}
