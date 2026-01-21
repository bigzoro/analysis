package main

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// ===== API速率限制器 =====

// APIRateLimiter API请求速率限制器
type APIRateLimiter struct {
	tokens    int           // 当前可用令牌数
	capacity  int           // 令牌桶容量
	refillRate time.Duration // 令牌补充间隔
	lastRefill time.Time     // 上次补充时间
	mu        sync.Mutex
}

func NewAPIRateLimiter(capacity int, refillRate time.Duration) *APIRateLimiter {
	return &APIRateLimiter{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// WaitForToken 等待获取令牌
func (r *APIRateLimiter) WaitForToken(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()

		// 计算需要补充的令牌数
		elapsed := now.Sub(r.lastRefill)
		tokensToAdd := int(elapsed / r.refillRate)

		if tokensToAdd > 0 {
			r.tokens = min(r.capacity, r.tokens+tokensToAdd)
			r.lastRefill = now
		}

		if r.tokens > 0 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		r.mu.Unlock()

		// 等待补充间隔
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.refillRate):
			// 继续循环等待
		}
	}
}

// 细粒度的API速率限制器 - 针对不同Binance API的限制
var (
	// 价格API限制器 - 每秒最多10个请求
	PriceAPIRateLimiter = NewAPIRateLimiter(8, 1*time.Second) // 保守设置为8个

	// K线API限制器 - 每秒最多10个请求
	KlineAPIRateLimiter = NewAPIRateLimiter(5, 1*time.Second) // 更保守，5个/秒

	// 深度API限制器 - 每秒最多10个请求
	DepthAPIRateLimiter = NewAPIRateLimiter(5, 1*time.Second) // 更保守，5个/秒

	// 其他API的全局限制器
	GlobalAPIRateLimiter = NewAPIRateLimiter(5, 1*time.Second) // 通用限制器
)

// 工具函数
func parseFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ===== 同步器接口 =====

type DataSyncer interface {
	Name() string
	Start(ctx context.Context, interval time.Duration)
	Stop()
	Sync(ctx context.Context) error
	GetStats() map[string]interface{}
}
