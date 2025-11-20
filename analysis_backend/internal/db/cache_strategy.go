package db

import (
	"sync"
	"time"
)

// ==================== 缓存策略优化 ====================

// CacheTTL 缓存 TTL 配置
type CacheTTL struct {
	// 实时数据：1-5 分钟
	RealTime time.Duration
	
	// 聚合数据：5-15 分钟
	Aggregate time.Duration
	
	// 静态数据：30 分钟 - 1 小时
	Static time.Duration
	
	// 长期数据：1-24 小时
	LongTerm time.Duration
}

// DefaultCacheTTL 默认缓存 TTL 配置
var DefaultCacheTTL = CacheTTL{
	RealTime:  2 * time.Minute,   // 实时数据：2分钟
	Aggregate: 10 * time.Minute,  // 聚合数据：10分钟
	Static:    30 * time.Minute,   // 静态数据：30分钟
	LongTerm:  2 * time.Hour,     // 长期数据：2小时
}

// CacheType 缓存类型
type CacheType int

const (
	CacheTypeRealTime CacheType = iota // 实时数据（市场数据、最新持仓等）
	CacheTypeAggregate                 // 聚合数据（资金流、统计等）
	CacheTypeStatic                    // 静态数据（实体列表、黑名单等）
	CacheTypeLongTerm                  // 长期数据（历史数据、归档数据等）
)

// GetTTL 根据缓存类型获取 TTL
func (ttl *CacheTTL) GetTTL(cacheType CacheType) time.Duration {
	switch cacheType {
	case CacheTypeRealTime:
		return ttl.RealTime
	case CacheTypeAggregate:
		return ttl.Aggregate
	case CacheTypeStatic:
		return ttl.Static
	case CacheTypeLongTerm:
		return ttl.LongTerm
	default:
		return ttl.Aggregate
	}
}

// ==================== 缓存键命名规范 ====================

// CacheKeyBuilder 缓存键构建器
type CacheKeyBuilder struct {
	prefix string
	version string
}

// NewCacheKeyBuilder 创建缓存键构建器
func NewCacheKeyBuilder(prefix, version string) *CacheKeyBuilder {
	return &CacheKeyBuilder{
		prefix:  prefix,
		version: version,
	}
}

// Build 构建缓存键
// 格式: {prefix}:{version}:{type}:{key}
func (b *CacheKeyBuilder) Build(cacheType string, keyParts ...string) string {
	key := b.prefix
	if b.version != "" {
		key += ":" + b.version
	}
	key += ":" + cacheType
	for _, part := range keyParts {
		if part != "" {
			key += ":" + part
		}
	}
	return key
}

// BuildPattern 构建缓存键模式（用于批量删除）
func (b *CacheKeyBuilder) BuildPattern(cacheType string, keyParts ...string) string {
	key := b.prefix
	if b.version != "" {
		key += ":" + b.version
	}
	key += ":" + cacheType
	for _, part := range keyParts {
		if part != "" {
			key += ":" + part
		}
	}
	return key + "*"
}

// ==================== 缓存统计 ====================

// CacheStats 缓存统计
type CacheStats struct {
	Hits       int64         // 命中次数
	Misses     int64         // 未命中次数
	Sets       int64         // 设置次数
	Deletes    int64         // 删除次数
	Errors     int64         // 错误次数
	HitRate    float64       // 命中率
	LastReset  time.Time     // 上次重置时间
}

// CacheStatsCollector 缓存统计收集器
type CacheStatsCollector struct {
	stats map[string]*CacheStats
	mu    sync.RWMutex
}

var globalStatsCollector = &CacheStatsCollector{
	stats: make(map[string]*CacheStats),
}

// RecordHit 记录命中
func (c *CacheStatsCollector) RecordHit(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	stats := c.getOrCreateStats(keyPrefix)
	stats.Hits++
	c.updateHitRate(stats)
}

// RecordMiss 记录未命中
func (c *CacheStatsCollector) RecordMiss(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	stats := c.getOrCreateStats(keyPrefix)
	stats.Misses++
	c.updateHitRate(stats)
}

// RecordSet 记录设置
func (c *CacheStatsCollector) RecordSet(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	stats := c.getOrCreateStats(keyPrefix)
	stats.Sets++
}

// RecordDelete 记录删除
func (c *CacheStatsCollector) RecordDelete(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	stats := c.getOrCreateStats(keyPrefix)
	stats.Deletes++
}

// RecordError 记录错误
func (c *CacheStatsCollector) RecordError(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	stats := c.getOrCreateStats(keyPrefix)
	stats.Errors++
}

// GetStats 获取统计信息
func (c *CacheStatsCollector) GetStats(keyPrefix string) *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.getOrCreateStats(keyPrefix)
	// 返回副本，避免并发修改
	return &CacheStats{
		Hits:      stats.Hits,
		Misses:    stats.Misses,
		Sets:      stats.Sets,
		Deletes:   stats.Deletes,
		Errors:    stats.Errors,
		HitRate:   stats.HitRate,
		LastReset: stats.LastReset,
	}
}

// GetAllStats 获取所有统计信息
func (c *CacheStatsCollector) GetAllStats() map[string]*CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make(map[string]*CacheStats)
	for k, v := range c.stats {
		result[k] = &CacheStats{
			Hits:      v.Hits,
			Misses:    v.Misses,
			Sets:      v.Sets,
			Deletes:   v.Deletes,
			Errors:    v.Errors,
			HitRate:   v.HitRate,
			LastReset: v.LastReset,
		}
	}
	return result
}

// ResetStats 重置统计信息
func (c *CacheStatsCollector) ResetStats(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if stats, ok := c.stats[keyPrefix]; ok {
		stats.Hits = 0
		stats.Misses = 0
		stats.Sets = 0
		stats.Deletes = 0
		stats.Errors = 0
		stats.HitRate = 0
		stats.LastReset = time.Now()
	}
}

func (c *CacheStatsCollector) getOrCreateStats(keyPrefix string) *CacheStats {
	if stats, ok := c.stats[keyPrefix]; ok {
		return stats
	}
	stats := &CacheStats{
		LastReset: time.Now(),
	}
	c.stats[keyPrefix] = stats
	return stats
}

func (c *CacheStatsCollector) updateHitRate(stats *CacheStats) {
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total) * 100
	}
}

// GetCacheStats 获取全局缓存统计
func GetCacheStats(keyPrefix string) *CacheStats {
	return globalStatsCollector.GetStats(keyPrefix)
}

// GetAllCacheStats 获取所有缓存统计
func GetAllCacheStats() map[string]*CacheStats {
	return globalStatsCollector.GetAllStats()
}

