package util

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries   int           // 最大重试次数
	InitialDelay time.Duration // 初始延迟（指数退避的基数）
	MaxDelay     time.Duration // 最大延迟
	Multiplier   float64       // 延迟倍数（默认2.0，指数退避）
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryableError 可重试的错误类型
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// 网络相关错误（可重试）
	retryableKeywords := []string{
		"timeout",
		"deadline exceeded",
		"connection",
		"network",
		"temporary",
		"tls handshake timeout",
		"eof",
		"no such host",
		"connection refused",
		"connection reset",
		"i/o timeout",
	}

	for _, keyword := range retryableKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	// HTTP 5xx 错误（可重试）
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	// 检查是否是RetryableError类型
	if retryableErr, ok := err.(*RetryableError); ok {
		return retryableErr.Retryable
	}

	return false
}

// Retry 执行带重试的函数
// fn: 要执行的函数，返回error表示失败
// config: 重试配置，如果为nil则使用默认配置
func Retry(ctx context.Context, fn func() error, config *RetryConfig) error {
	if config == nil {
		defaultConfig := DefaultRetryConfig()
		config = &defaultConfig
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// 执行函数
		err := fn()
		if err == nil {
			return nil // 成功
		}

		lastErr = err

		// 检查是否可重试
		if !IsRetryableError(err) {
			return fmt.Errorf("不可重试的错误: %w", err)
		}

		// 如果已经是最后一次尝试，不再等待
		if attempt == config.MaxRetries {
			break
		}

		// 计算延迟时间（指数退避）
		delay := time.Duration(float64(config.InitialDelay) * pow(config.Multiplier, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// 等待，但检查context是否取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("重试被取消: %w", ctx.Err())
		case <-time.After(delay):
			// 继续重试
		}
	}

	return fmt.Errorf("重试 %d 次后仍然失败: %w", config.MaxRetries+1, lastErr)
}

// RetryWithResult 执行带重试的函数（有返回值）
// fn: 要执行的函数，返回结果和error
// config: 重试配置，如果为nil则使用默认配置
func RetryWithResult[T any](ctx context.Context, fn func() (T, error), config *RetryConfig) (T, error) {
	var zero T

	if config == nil {
		defaultConfig := DefaultRetryConfig()
		config = &defaultConfig
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// 执行函数
		result, err := fn()
		if err == nil {
			return result, nil // 成功
		}

		lastErr = err

		// 检查是否可重试
		if !IsRetryableError(err) {
			return zero, fmt.Errorf("不可重试的错误: %w", err)
		}

		// 如果已经是最后一次尝试，不再等待
		if attempt == config.MaxRetries {
			break
		}

		// 计算延迟时间（指数退避）
		delay := time.Duration(float64(config.InitialDelay) * pow(config.Multiplier, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// 等待，但检查context是否取消
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("重试被取消: %w", ctx.Err())
		case <-time.After(delay):
			// 继续重试
		}
	}

	return zero, fmt.Errorf("重试 %d 次后仍然失败: %w", config.MaxRetries+1, lastErr)
}

// pow 计算幂（简单实现）
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
