package server

import (
	"fmt"
	"log"
	"runtime/debug"
)

// BacktestError 回测错误类型
type BacktestError struct {
	Code    string
	Message string
	Details map[string]interface{}
	Cause   error
	Stack   string
}

// Error 实现error接口
func (e *BacktestError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	errors []BacktestError
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		errors: make([]BacktestError, 0),
	}
}

// HandleError 处理错误
func (eh *ErrorHandler) HandleError(code, message string, cause error, details map[string]interface{}) error {
	backtestErr := BacktestError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
		Stack:   string(debug.Stack()),
	}

	eh.errors = append(eh.errors, backtestErr)

	// 记录错误日志
	log.Printf("[ERROR] %s: %s", code, message)
	if cause != nil {
		log.Printf("[ERROR] Cause: %v", cause)
	}
	if details != nil {
		for k, v := range details {
			log.Printf("[ERROR] %s: %v", k, v)
		}
	}

	return &backtestErr
}

// HandleDataError 处理数据相关错误
func (eh *ErrorHandler) HandleDataError(operation string, symbol string, cause error) error {
	details := map[string]interface{}{
		"operation": operation,
		"symbol":    symbol,
	}
	return eh.HandleError("DATA_ERROR", "数据处理失败", cause, details)
}

// HandleStrategyError 处理策略相关错误
func (eh *ErrorHandler) HandleStrategyError(strategy string, cause error) error {
	details := map[string]interface{}{
		"strategy": strategy,
	}
	return eh.HandleError("STRATEGY_ERROR", "策略执行失败", cause, details)
}

// HandleValidationError 处理验证错误
func (eh *ErrorHandler) HandleValidationError(field string, value interface{}, cause error) error {
	details := map[string]interface{}{
		"field": field,
		"value": value,
	}
	return eh.HandleError("VALIDATION_ERROR", "配置验证失败", cause, details)
}

// HasErrors 检查是否有错误
func (eh *ErrorHandler) HasErrors() bool {
	return len(eh.errors) > 0
}

// GetErrors 获取所有错误
func (eh *ErrorHandler) GetErrors() []BacktestError {
	return eh.errors
}

// ClearErrors 清除错误
func (eh *ErrorHandler) ClearErrors() {
	eh.errors = make([]BacktestError, 0)
}

// GetLastError 获取最后一个错误
func (eh *ErrorHandler) GetLastError() *BacktestError {
	if len(eh.errors) == 0 {
		return nil
	}
	return &eh.errors[len(eh.errors)-1]
}

// ErrorRecovery 错误恢复策略
type ErrorRecovery struct {
	MaxRetries   int
	RetryDelay   int // 毫秒
	FallbackData interface{}
}

// RecoveryHandler 恢复处理器
type RecoveryHandler struct {
	recoveryStrategies map[string]ErrorRecovery
}

// NewRecoveryHandler 创建恢复处理器
func NewRecoveryHandler() *RecoveryHandler {
	return &RecoveryHandler{
		recoveryStrategies: make(map[string]ErrorRecovery),
	}
}

// SetRecoveryStrategy 设置恢复策略
func (rh *RecoveryHandler) SetRecoveryStrategy(errorCode string, strategy ErrorRecovery) {
	rh.recoveryStrategies[errorCode] = strategy
}

// Recover 执行错误恢复
func (rh *RecoveryHandler) Recover(errorCode string, operation func() error) error {
	strategy, exists := rh.recoveryStrategies[errorCode]
	if !exists {
		return operation() // 没有恢复策略，直接执行
	}

	var lastErr error
	for i := 0; i <= strategy.MaxRetries; i++ {
		err := operation()
		if err == nil {
			return nil // 成功执行
		}

		lastErr = err

		// 如果不是最后一次重试，等待后重试
		if i < strategy.MaxRetries {
			log.Printf("[RECOVERY] 尝试 %d/%d 失败，重试中: %v", i+1, strategy.MaxRetries+1, err)
			// 这里可以添加等待逻辑
		}
	}

	// 所有重试都失败，返回最后一次错误
	log.Printf("[RECOVERY] 所有重试都失败，返回最后错误: %v", lastErr)
	return lastErr
}
