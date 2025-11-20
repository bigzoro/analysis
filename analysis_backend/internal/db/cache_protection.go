package db

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ==================== 缓存保护机制 ====================

// CacheProtection 缓存保护配置
type CacheProtection struct {
	// 防止缓存穿透：对于不存在的键，也缓存一个空值
	PreventPenetration bool
	EmptyValueTTL      time.Duration // 空值缓存时间（通常较短）
	
	// 防止缓存击穿：使用互斥锁保护热点数据
	PreventBreakdown bool
	LockTimeout      time.Duration // 锁超时时间
	
	// 防止缓存雪崩：随机化 TTL
	PreventAvalanche bool
	TTLRandomFactor  float64 // TTL 随机因子（0.1 表示 ±10%）
}

// DefaultCacheProtection 默认缓存保护配置
var DefaultCacheProtection = CacheProtection{
	PreventPenetration: true,
	EmptyValueTTL:      1 * time.Minute,
	PreventBreakdown:   true,
	LockTimeout:        5 * time.Second,
	PreventAvalanche:   true,
	TTLRandomFactor:    0.1, // ±10%
}

// ProtectedCache 带保护的缓存包装器
type ProtectedCache struct {
	cache     CacheInterface
	protection CacheProtection
	locks     map[string]*sync.Mutex
	mu        sync.RWMutex
}

// NewProtectedCache 创建带保护的缓存
func NewProtectedCache(cache CacheInterface, protection CacheProtection) *ProtectedCache {
	return &ProtectedCache{
		cache:      cache,
		protection: protection,
		locks:      make(map[string]*sync.Mutex),
	}
}

// Get 获取缓存（带保护）
func (p *ProtectedCache) Get(ctx context.Context, key string) ([]byte, error) {
	// 尝试从缓存获取
	data, err := p.cache.Get(ctx, key)
	if err == nil {
		// 检查是否是空值标记
		if len(data) == 1 && data[0] == 0 {
			// 这是空值标记，返回未找到
			return nil, fmt.Errorf("key not found")
		}
		return data, nil
	}
	
	// 缓存未命中
	if p.protection.PreventPenetration {
		// 检查是否有空值标记
		emptyKey := p.getEmptyKey(key)
		emptyData, emptyErr := p.cache.Get(ctx, emptyKey)
		if emptyErr == nil && len(emptyData) == 1 && emptyData[0] == 0 {
			// 存在空值标记，说明之前查询过，确实不存在
			return nil, fmt.Errorf("key not found")
		}
	}
	
	return nil, err
}

// Set 设置缓存（带保护）
func (p *ProtectedCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// 防止缓存雪崩：随机化 TTL
	if p.protection.PreventAvalanche {
		ttl = p.randomizeTTL(ttl)
	}
	
	return p.cache.Set(ctx, key, value, ttl)
}

// SetEmpty 设置空值标记（防止缓存穿透）
func (p *ProtectedCache) SetEmpty(ctx context.Context, key string) error {
	if !p.protection.PreventPenetration {
		return nil
	}
	
	emptyKey := p.getEmptyKey(key)
	emptyValue := []byte{0} // 空值标记
	return p.cache.Set(ctx, emptyKey, emptyValue, p.protection.EmptyValueTTL)
}

// GetWithLock 带锁的获取（防止缓存击穿）
func (p *ProtectedCache) GetWithLock(ctx context.Context, key string, loader func() ([]byte, time.Duration, error)) ([]byte, error) {
	// 先尝试从缓存获取
	data, err := p.Get(ctx, key)
	if err == nil {
		return data, nil
	}
	
	if !p.protection.PreventBreakdown {
		// 不使用锁保护，直接加载
		return p.loadAndSet(ctx, key, loader)
	}
	
	// 获取或创建锁
	lock := p.getLock(key)
	
	// 尝试获取锁（带超时）
	lockAcquired := make(chan bool, 1)
	go func() {
		lock.Lock()
		lockAcquired <- true
	}()
	
	select {
	case <-lockAcquired:
		defer lock.Unlock()
		
		// 再次检查缓存（双重检查）
		data, err := p.Get(ctx, key)
		if err == nil {
			return data, nil
		}
		
		// 加载数据
		return p.loadAndSet(ctx, key, loader)
		
	case <-time.After(p.protection.LockTimeout):
		// 锁超时，直接加载（可能重复加载，但避免死锁）
		return p.loadAndSet(ctx, key, loader)
		
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// loadAndSet 加载数据并设置缓存
func (p *ProtectedCache) loadAndSet(ctx context.Context, key string, loader func() ([]byte, time.Duration, error)) ([]byte, error) {
	data, ttl, err := loader()
	if err != nil {
		// 加载失败，设置空值标记（防止缓存穿透）
		_ = p.SetEmpty(ctx, key)
		return nil, err
	}
	
	if len(data) == 0 {
		// 数据为空，设置空值标记
		_ = p.SetEmpty(ctx, key)
		return nil, fmt.Errorf("key not found")
	}
	
	// 设置缓存
	if setErr := p.Set(ctx, key, data, ttl); setErr != nil {
		// 设置缓存失败不影响返回数据
		// 可以记录日志
	}
	
	return data, nil
}

// Delete 删除缓存
func (p *ProtectedCache) Delete(ctx context.Context, key string) error {
	// 同时删除空值标记
	emptyKey := p.getEmptyKey(key)
	_ = p.cache.Delete(ctx, emptyKey)
	return p.cache.Delete(ctx, key)
}

// Exists 检查键是否存在
func (p *ProtectedCache) Exists(ctx context.Context, key string) (bool, error) {
	return p.cache.Exists(ctx, key)
}

// getEmptyKey 获取空值标记键
func (p *ProtectedCache) getEmptyKey(key string) string {
	return key + ":empty"
}

// getLock 获取或创建锁
func (p *ProtectedCache) getLock(key string) *sync.Mutex {
	p.mu.RLock()
	lock, ok := p.locks[key]
	p.mu.RUnlock()
	
	if ok {
		return lock
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// 双重检查
	if lock, ok := p.locks[key]; ok {
		return lock
	}
	
	lock = &sync.Mutex{}
	p.locks[key] = lock
	return lock
}

// randomizeTTL 随机化 TTL（防止缓存雪崩）
func (p *ProtectedCache) randomizeTTL(baseTTL time.Duration) time.Duration {
	if p.protection.TTLRandomFactor <= 0 {
		return baseTTL
	}
	
	// 生成 -factor 到 +factor 之间的随机因子
	factor := 1.0 + (rand.Float64()*2.0-1.0)*p.protection.TTLRandomFactor
	return time.Duration(float64(baseTTL) * factor)
}

