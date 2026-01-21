package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// ============================================================================
// 资源池管理系统
// ============================================================================

// ResourceType 资源类型
type ResourceType int

const (
	ResourceTypeDatabase ResourceType = iota
	ResourceTypeRedis
	ResourceTypeHTTPClient
	ResourceTypeWorkerPool
	ResourceTypeMemory
	ResourceTypeFileHandle
)

// ResourcePoolConfig 资源池配置
type ResourcePoolConfig struct {
	Type                ResourceType
	MinSize             int           // 最小连接数
	MaxSize             int           // 最大连接数
	IdleTimeout         time.Duration // 空闲超时
	MaxLifetime         time.Duration // 最大生命周期
	HealthCheckInterval time.Duration // 健康检查间隔
	RetryAttempts       int           // 重试次数
	RetryDelay          time.Duration // 重试延迟
}

// ResourcePool 通用资源池接口
type ResourcePool interface {
	Get(ctx context.Context) (interface{}, error)
	Put(resource interface{}) error
	Close() error
	Stats() ResourcePoolStats
}

// ResourcePoolStats 资源池统计信息
type ResourcePoolStats struct {
	Type               ResourceType
	ActiveCount        int           // 活跃连接数
	IdleCount          int           // 空闲连接数
	TotalCount         int           // 总连接数
	CreatedCount       int64         // 已创建连接数
	DestroyedCount     int64         // 已销毁连接数
	WaitCount          int64         // 等待获取连接数
	WaitDuration       time.Duration // 平均等待时间
	MaxLifetimeReached int64         // 达到最大生命周期的连接数
}

// DatabaseResourcePool 数据库连接池
type DatabaseResourcePool struct {
	config ResourcePoolConfig
	db     *sql.DB
	mu     sync.RWMutex
	stats  ResourcePoolStats
	ctx    context.Context
	cancel context.CancelFunc
}

// NewDatabaseResourcePool 创建数据库资源池
func NewDatabaseResourcePool(db *sql.DB, config ResourcePoolConfig) *DatabaseResourcePool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &DatabaseResourcePool{
		config: config,
		db:     db,
		ctx:    ctx,
		cancel: cancel,
		stats: ResourcePoolStats{
			Type: ResourceTypeDatabase,
		},
	}

	// 设置数据库连接池参数
	db.SetMaxOpenConns(config.MaxSize)
	db.SetMaxIdleConns(config.MaxSize / 2)
	db.SetConnMaxLifetime(config.MaxLifetime)

	// 启动健康检查
	go pool.healthChecker()

	return pool
}

// Get 获取数据库连接
func (drp *DatabaseResourcePool) Get(ctx context.Context) (interface{}, error) {
	atomic.AddInt64(&drp.stats.WaitCount, 1)
	startTime := time.Now()

	conn, err := drp.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	waitDuration := time.Since(startTime)
	// 注意：这里应该使用互斥锁而不是原子操作，因为time.Duration不是int64
	drp.mu.Lock()
	drp.stats.WaitDuration += waitDuration
	drp.stats.ActiveCount++
	drp.mu.Unlock()

	return conn, nil
}

// Put 归还数据库连接
func (drp *DatabaseResourcePool) Put(resource interface{}) error {
	conn, ok := resource.(*sql.Conn)
	if !ok {
		return fmt.Errorf("invalid resource type for database pool")
	}

	err := conn.Close()
	if err != nil {
		log.Printf("[DatabasePool] Error closing connection: %v", err)
	}

	drp.mu.Lock()
	drp.stats.ActiveCount--
	drp.mu.Unlock()

	return nil
}

// Close 关闭数据库资源池
func (drp *DatabaseResourcePool) Close() error {
	drp.cancel()
	return drp.db.Close()
}

// Stats 获取统计信息
func (drp *DatabaseResourcePool) Stats() ResourcePoolStats {
	drp.mu.RLock()
	defer drp.mu.RUnlock()

	stats := drp.stats
	stats.IdleCount = drp.config.MaxSize - stats.ActiveCount
	stats.TotalCount = drp.config.MaxSize

	return stats
}

// healthChecker 健康检查器
func (drp *DatabaseResourcePool) healthChecker() {
	ticker := time.NewTicker(drp.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			drp.performHealthCheck()
		case <-drp.ctx.Done():
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (drp *DatabaseResourcePool) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := drp.db.PingContext(ctx); err != nil {
		log.Printf("[DatabasePool] Health check failed: %v", err)
		// 这里可以实现连接重建逻辑
	} else {
		log.Printf("[DatabasePool] Health check passed")
	}
}

// RedisResourcePool Redis连接池
type RedisResourcePool struct {
	config ResourcePoolConfig
	client *redis.Client
	mu     sync.RWMutex
	stats  ResourcePoolStats
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRedisResourcePool 创建Redis资源池
func NewRedisResourcePool(client *redis.Client, config ResourcePoolConfig) *RedisResourcePool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &RedisResourcePool{
		config: config,
		client: client,
		ctx:    ctx,
		cancel: cancel,
		stats: ResourcePoolStats{
			Type: ResourceTypeRedis,
		},
	}

	// 启动健康检查
	go pool.healthChecker()

	return pool
}

// Get 获取Redis客户端
func (rrp *RedisResourcePool) Get(ctx context.Context) (interface{}, error) {
	atomic.AddInt64(&rrp.stats.WaitCount, 1)
	startTime := time.Now()

	// 检查连接健康
	if err := rrp.client.Ping(ctx).Err(); err != nil {
		waitDuration := time.Since(startTime)
		rrp.mu.Lock()
		rrp.stats.WaitDuration += waitDuration
		rrp.mu.Unlock()
		return nil, fmt.Errorf("redis connection unhealthy: %w", err)
	}

	waitDuration := time.Since(startTime)
	rrp.mu.Lock()
	rrp.stats.WaitDuration += waitDuration
	rrp.stats.ActiveCount++
	rrp.mu.Unlock()

	return rrp.client, nil
}

// Put 归还Redis客户端
func (rrp *RedisResourcePool) Put(resource interface{}) error {
	rrp.mu.Lock()
	rrp.stats.ActiveCount--
	rrp.mu.Unlock()
	return nil
}

// Close 关闭Redis资源池
func (rrp *RedisResourcePool) Close() error {
	rrp.cancel()
	return rrp.client.Close()
}

// Stats 获取统计信息
func (rrp *RedisResourcePool) Stats() ResourcePoolStats {
	rrp.mu.RLock()
	defer rrp.mu.RUnlock()

	stats := rrp.stats
	stats.IdleCount = 1 // Redis客户端通常是单例
	stats.TotalCount = 1

	return stats
}

// healthChecker 健康检查器
func (rrp *RedisResourcePool) healthChecker() {
	ticker := time.NewTicker(rrp.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rrp.performHealthCheck()
		case <-rrp.ctx.Done():
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (rrp *RedisResourcePool) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rrp.client.Ping(ctx).Err(); err != nil {
		log.Printf("[RedisPool] Health check failed: %v", err)
	} else {
		log.Printf("[RedisPool] Health check passed")
	}
}

// ResourceManager 资源管理器
type ResourceManager struct {
	pools  map[ResourceType]ResourcePool
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewResourceManager 创建资源管理器
func NewResourceManager() *ResourceManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceManager{
		pools:  make(map[ResourceType]ResourcePool),
		ctx:    ctx,
		cancel: cancel,
	}
}

// RegisterPool 注册资源池
func (rm *ResourceManager) RegisterPool(poolType ResourceType, pool ResourcePool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.pools[poolType] = pool
	log.Printf("[ResourceManager] Registered pool: %v", poolType)
}

// GetPool 获取资源池
func (rm *ResourceManager) GetPool(poolType ResourceType) ResourcePool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.pools[poolType]
}

// GetResource 获取资源
func (rm *ResourceManager) GetResource(ctx context.Context, poolType ResourceType) (interface{}, error) {
	pool := rm.GetPool(poolType)
	if pool == nil {
		return nil, fmt.Errorf("pool not found: %v", poolType)
	}

	return pool.Get(ctx)
}

// ReturnResource 归还资源
func (rm *ResourceManager) ReturnResource(poolType ResourceType, resource interface{}) error {
	pool := rm.GetPool(poolType)
	if pool == nil {
		return fmt.Errorf("pool not found: %v", poolType)
	}

	return pool.Put(resource)
}

// GetStats 获取所有资源池统计信息
func (rm *ResourceManager) GetStats() map[ResourceType]ResourcePoolStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := make(map[ResourceType]ResourcePoolStats)
	for poolType, pool := range rm.pools {
		stats[poolType] = pool.Stats()
	}

	return stats
}

// HealthCheck 执行健康检查
func (rm *ResourceManager) HealthCheck() map[ResourceType]bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	results := make(map[ResourceType]bool)
	for poolType, pool := range rm.pools {
		// 简单的健康检查：检查是否能获取和归还资源
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		resource, err := pool.Get(ctx)
		if err != nil {
			results[poolType] = false
			cancel()
			continue
		}

		err = pool.Put(resource)
		results[poolType] = err == nil

		cancel()
	}

	return results
}

// Close 关闭所有资源池
func (rm *ResourceManager) Close() error {
	rm.cancel()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	var errors []error
	for poolType, pool := range rm.pools {
		if err := pool.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close pool %v: %w", poolType, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("resource manager close errors: %v", errors)
	}

	return nil
}

// GetSystemResourceUsage 获取系统资源使用情况
func (rm *ResourceManager) GetSystemResourceUsage() SystemResourceUsage {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemResourceUsage{
		Goroutines:  runtime.NumGoroutine(),
		MemoryUsed:  m.Alloc,
		MemoryTotal: m.Sys,
		GCCycles:    m.NumGC,
		LastGC:      time.Unix(0, int64(m.LastGC)),
		HeapObjects: m.HeapObjects,
		PoolStats:   rm.GetStats(),
	}
}

// SystemResourceUsage 系统资源使用情况
type SystemResourceUsage struct {
	Goroutines  int                                `json:"goroutines"`
	MemoryUsed  uint64                             `json:"memory_used"`
	MemoryTotal uint64                             `json:"memory_total"`
	GCCycles    uint32                             `json:"gc_cycles"`
	LastGC      time.Time                          `json:"last_gc"`
	HeapObjects uint64                             `json:"heap_objects"`
	PoolStats   map[ResourceType]ResourcePoolStats `json:"pool_stats"`
}

// 全局资源管理器实例
var globalResourceManager *ResourceManager

// GetGlobalResourceManager 获取全局资源管理器
func GetGlobalResourceManager() *ResourceManager {
	if globalResourceManager == nil {
		globalResourceManager = NewResourceManager()
	}
	return globalResourceManager
}
