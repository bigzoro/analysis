package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// 熔断器实现 - 防止级联故障
// ============================================================================

// CircuitState 熔断器状态
type CircuitState int32

const (
	StateClosed   CircuitState = iota // 关闭状态：正常工作
	StateOpen                         // 开启状态：熔断，所有请求快速失败
	StateHalfOpen                     // 半开启状态：允许部分请求通过，测试服务是否恢复
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name string

	// 配置
	failureThreshold int           // 失败阈值
	successThreshold int           // 成功阈值（半开启状态）
	timeout          time.Duration // 超时时间
	resetTimeout     time.Duration // 重置超时（从开启到半开启）

	// 状态
	state        int32     // 当前状态
	failures     int32     // 连续失败次数
	successes    int32     // 半开启状态下的连续成功次数
	lastFailTime time.Time // 最后失败时间
	nextAttempt  time.Time // 下次尝试时间

	// 统计
	totalRequests  int64
	totalFailures  int64
	totalSuccesses int64
	totalTimeouts  int64

	mu sync.RWMutex
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	ResetTimeout     time.Duration
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:             config.Name,
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		timeout:          config.Timeout,
		resetTimeout:     config.ResetTimeout,
		state:            int32(StateClosed),
	}
}

// Call 执行带熔断器的函数调用
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	// 检查熔断器状态
	state := CircuitState(atomic.LoadInt32(&cb.state))

	switch state {
	case StateOpen:
		// 检查是否可以尝试重置
		if time.Now().Before(cb.nextAttempt) {
			atomic.AddInt64(&cb.totalRequests, 1)
			return ErrCircuitOpen
		}

		// 切换到半开启状态
		atomic.StoreInt32(&cb.state, int32(StateHalfOpen))
		atomic.StoreInt32(&cb.successes, 0)
		fallthrough

	case StateHalfOpen:
		// 半开启状态：允许请求通过，但记录结果
		atomic.AddInt64(&cb.totalRequests, 1)

		done := make(chan error, 1)
		go func() {
			done <- fn()
		}()

		select {
		case err := <-done:
			if err != nil {
				cb.recordFailure()
				return err
			} else {
				cb.recordSuccess()
				return nil
			}
		case <-time.After(cb.timeout):
			atomic.AddInt64(&cb.totalTimeouts, 1)
			cb.recordFailure()
			return ErrCircuitTimeout
		case <-ctx.Done():
			return ctx.Err()
		}

	case StateClosed:
		// 正常状态
		atomic.AddInt64(&cb.totalRequests, 1)

		done := make(chan error, 1)
		go func() {
			done <- fn()
		}()

		select {
		case err := <-done:
			if err != nil {
				cb.recordFailure()
				return err
			} else {
				cb.recordSuccess()
				return nil
			}
		case <-time.After(cb.timeout):
			atomic.AddInt64(&cb.totalTimeouts, 1)
			cb.recordFailure()
			return ErrCircuitTimeout
		case <-ctx.Done():
			return ctx.Err()
		}

	default:
		return ErrCircuitUnknownState
	}
}

// recordFailure 记录失败
func (cb *CircuitBreaker) recordFailure() {
	atomic.AddInt64(&cb.totalFailures, 1)

	failures := atomic.AddInt32(&cb.failures, 1)
	state := CircuitState(atomic.LoadInt32(&cb.state))

	if state == StateClosed && int(failures) >= cb.failureThreshold {
		// 切换到开启状态
		atomic.StoreInt32(&cb.state, int32(StateOpen))
		cb.nextAttempt = time.Now().Add(cb.resetTimeout)
		cb.lastFailTime = time.Now()
	}
}

// recordSuccess 记录成功
func (cb *CircuitBreaker) recordSuccess() {
	atomic.AddInt64(&cb.totalSuccesses, 1)

	state := CircuitState(atomic.LoadInt32(&cb.state))

	if state == StateHalfOpen {
		successes := atomic.AddInt32(&cb.successes, 1)
		if int(successes) >= cb.successThreshold {
			// 切换到关闭状态
			atomic.StoreInt32(&cb.state, int32(StateClosed))
			atomic.StoreInt32(&cb.failures, 0)
			atomic.StoreInt32(&cb.successes, 0)
		}
	} else if state == StateClosed {
		// 重置失败计数
		atomic.StoreInt32(&cb.failures, 0)
	}
}

// GetStats 获取统计信息
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	return CircuitBreakerStats{
		Name:           cb.name,
		State:          CircuitState(atomic.LoadInt32(&cb.state)),
		Failures:       atomic.LoadInt32(&cb.failures),
		Successes:      atomic.LoadInt32(&cb.successes),
		TotalRequests:  atomic.LoadInt64(&cb.totalRequests),
		TotalFailures:  atomic.LoadInt64(&cb.totalFailures),
		TotalSuccesses: atomic.LoadInt64(&cb.totalSuccesses),
		TotalTimeouts:  atomic.LoadInt64(&cb.totalTimeouts),
		LastFailTime:   cb.lastFailTime,
		NextAttempt:    cb.nextAttempt,
		FailureRate:    cb.calculateFailureRate(),
	}
}

// calculateFailureRate 计算失败率
func (cb *CircuitBreaker) calculateFailureRate() float64 {
	total := atomic.LoadInt64(&cb.totalRequests)
	failures := atomic.LoadInt64(&cb.totalFailures)

	if total == 0 {
		return 0.0
	}

	return float64(failures) / float64(total) * 100.0
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.successes, 0)
	cb.lastFailTime = time.Time{}
	cb.nextAttempt = time.Time{}
}

// CircuitBreakerStats 熔断器统计信息
type CircuitBreakerStats struct {
	Name           string       `json:"name"`
	State          CircuitState `json:"state"`
	Failures       int32        `json:"failures"`
	Successes      int32        `json:"successes"`
	TotalRequests  int64        `json:"total_requests"`
	TotalFailures  int64        `json:"total_failures"`
	TotalSuccesses int64        `json:"total_successes"`
	TotalTimeouts  int64        `json:"total_timeouts"`
	LastFailTime   time.Time    `json:"last_fail_time"`
	NextAttempt    time.Time    `json:"next_attempt"`
	FailureRate    float64      `json:"failure_rate"`
}

// 预定义错误
var (
	ErrCircuitOpen         = &CircuitBreakerError{msg: "circuit breaker is open"}
	ErrCircuitTimeout      = &CircuitBreakerError{msg: "circuit breaker timeout"}
	ErrCircuitUnknownState = &CircuitBreakerError{msg: "circuit breaker in unknown state"}
)

// CircuitBreakerError 熔断器错误
type CircuitBreakerError struct {
	msg string
}

func (e *CircuitBreakerError) Error() string {
	return e.msg
}

// ============================================================================
// 熔断器管理器
// ============================================================================

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate 获取或创建熔断器
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	if breaker, exists := cbm.breakers[name]; exists {
		return breaker
	}

	config.Name = name
	breaker := NewCircuitBreaker(config)
	cbm.breakers[name] = breaker

	return breaker
}

// Get 获取熔断器
func (cbm *CircuitBreakerManager) Get(name string) *CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	return cbm.breakers[name]
}

// GetAllStats 获取所有熔断器统计信息
func (cbm *CircuitBreakerManager) GetAllStats() map[string]CircuitBreakerStats {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for name, breaker := range cbm.breakers {
		stats[name] = breaker.GetStats()
	}

	return stats
}

// ResetAll 重置所有熔断器
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	for _, breaker := range cbm.breakers {
		breaker.Reset()
	}
}

// 全局熔断器管理器
var globalCircuitBreakerManager *CircuitBreakerManager

// GetGlobalCircuitBreakerManager 获取全局熔断器管理器
func GetGlobalCircuitBreakerManager() *CircuitBreakerManager {
	if globalCircuitBreakerManager == nil {
		globalCircuitBreakerManager = NewCircuitBreakerManager()
	}
	return globalCircuitBreakerManager
}






