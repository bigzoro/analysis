package server

import (
	"context"
	"sync"
	"time"
)

// ==================== 限流器 ====================

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context) bool
	Wait(ctx context.Context) error
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	capacity    int64         // 桶容量
	tokens      int64         // 当前令牌数
	refillRate  time.Duration // 补充速率
	lastRefill  time.Time     // 上次补充时间
	mu          sync.Mutex
}

// NewTokenBucket 创建令牌桶限流器
// capacity: 桶容量（最大并发数）
// refillRate: 补充速率（每 refillRate 补充一个令牌）
func NewTokenBucket(capacity int64, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许执行（非阻塞）
func (tb *TokenBucket) Allow(ctx context.Context) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 补充令牌
	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

// Wait 等待直到允许执行（阻塞）
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow(ctx) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(tb.refillRate):
			// 等待一个补充周期
		}
	}
}

// refill 补充令牌
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int64(elapsed / tb.refillRate)

	if tokensToAdd > 0 {
		newTokens := tb.tokens + tokensToAdd
		if newTokens > tb.capacity {
			tb.tokens = tb.capacity
		} else {
			tb.tokens = newTokens
		}
		tb.lastRefill = now
	}
}

// SlidingWindow 滑动窗口限流器
type SlidingWindow struct {
	windowSize  time.Duration // 窗口大小
	maxRequests int64         // 窗口内最大请求数
	requests    []time.Time   // 请求时间戳
	mu          sync.Mutex
}

// NewSlidingWindow 创建滑动窗口限流器
// windowSize: 窗口大小
// maxRequests: 窗口内最大请求数
func NewSlidingWindow(windowSize time.Duration, maxRequests int64) *SlidingWindow {
	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0, maxRequests),
	}
}

// Allow 检查是否允许执行
func (sw *SlidingWindow) Allow(ctx context.Context) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	// 移除窗口外的请求
	validRequests := sw.requests[:0]
	for _, t := range sw.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	sw.requests = validRequests

	// 检查是否超过限制
	if int64(len(sw.requests)) >= sw.maxRequests {
		return false
	}

	// 添加当前请求
	sw.requests = append(sw.requests, now)
	return true
}

// Wait 等待直到允许执行
func (sw *SlidingWindow) Wait(ctx context.Context) error {
	for {
		if sw.Allow(ctx) {
			return nil
		}

		// 计算需要等待的时间
		sw.mu.Lock()
		if len(sw.requests) > 0 {
			oldest := sw.requests[0]
			waitTime := sw.windowSize - time.Since(oldest)
			sw.mu.Unlock()

			if waitTime > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitTime):
				}
			}
		} else {
			sw.mu.Unlock()
		}
	}
}

