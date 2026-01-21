package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

// ==================== Redis 缓存实现 ====================

// RedisCache Redis 缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建 Redis 缓存实例
func NewRedisCache(redisClient *redis.Client) *RedisCache {
	return &RedisCache{
		client: redisClient,
	}
}

// NewRedisCacheFromOptions 从配置选项创建 Redis 缓存
func NewRedisCacheFromOptions(addr, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		// 禁用维护通知功能，避免旧版本 Redis 的警告
		// 标准 Redis 服务器不支持 CLIENT MAINT_NOTIFICATIONS 命令
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

// Get 从缓存获取
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found")
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// Set 设置缓存
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists 检查键是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

// DeletePattern 按模式删除缓存（优化：使用管道批量删除）
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator() // 每次扫描100个键
	batchSize := 100                                       // 每批删除100个键

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())

		// 达到批次大小时，批量删除
		if len(keys) >= batchSize {
			if err := r.batchDelete(ctx, keys); err != nil {
				return err
			}
			keys = keys[:0] // 清空切片，保留容量
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	// 删除剩余的键
	if len(keys) > 0 {
		return r.batchDelete(ctx, keys)
	}

	return nil
}

// batchDelete 批量删除键（使用管道优化）
func (r *RedisCache) batchDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Close 关闭 Redis 连接
func (r *RedisCache) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// ==================== 内存缓存实现（用于测试或小规模部署）====================

// MemoryCache 内存缓存实现
type MemoryCache struct {
	data map[string]cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		data: make(map[string]cacheItem),
	}

	// 启动清理协程
	go mc.cleanup()

	return mc
}

func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	if time.Now().After(item.expiresAt) {
		delete(m.data, key)
		return nil, fmt.Errorf("key expired")
	}

	return item.value, nil
}

func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.data[key]
	if !ok {
		return false, nil
	}

	if time.Now().After(item.expiresAt) {
		return false, nil
	}

	return true, nil
}

// cleanup 定期清理过期键
func (m *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, item := range m.data {
			if now.After(item.expiresAt) {
				delete(m.data, key)
			}
		}
		m.mu.Unlock()
	}
}
