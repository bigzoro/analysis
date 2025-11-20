package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ==================== Redis 缓存层 ====================

// CacheInterface 缓存接口
type CacheInterface interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// CacheWrapper 缓存包装器
type CacheWrapper struct {
	cache CacheInterface
	db    *gorm.DB
}

func NewCacheWrapper(cache CacheInterface, db *gorm.DB) *CacheWrapper {
	return &CacheWrapper{
		cache: cache,
		db:    db,
	}
}

// ==================== 缓存键生成 ====================

func cacheKey(prefix string, parts ...interface{}) string {
	key := prefix
	for _, part := range parts {
		key += ":" + fmt.Sprintf("%v", part)
	}
	return key
}

// ==================== 缓存策略 ====================

// CacheStrategy 缓存策略
type CacheStrategy struct {
	TTL           time.Duration
	InvalidateOn  []string // 触发失效的操作
	RefreshOnMiss bool     // 缓存未命中时是否刷新
}

var (
	// 实时数据缓存策略（短TTL）
	RealTimeCache = CacheStrategy{
		TTL:           1 * time.Minute,
		InvalidateOn:  []string{"insert", "update"},
		RefreshOnMiss: true,
	}

	// 聚合数据缓存策略（中等TTL）
	AggregateCache = CacheStrategy{
		TTL:           5 * time.Minute,
		InvalidateOn:  []string{"insert"},
		RefreshOnMiss: true,
	}

	// 静态数据缓存策略（长TTL）
	StaticCache = CacheStrategy{
		TTL:           1 * time.Hour,
		InvalidateOn:  []string{"update"},
		RefreshOnMiss: false,
	}
)

// ==================== 缓存查询方法 ====================

// GetCachedTransfers 获取缓存的转账列表
func (cw *CacheWrapper) GetCachedTransfers(ctx context.Context, entity, chain, coin string, limit int, cursor string) ([]TransferEvent, string, error) {
	key := cacheKey("transfers", entity, chain, coin, limit, cursor)

	// 尝试从缓存获取
	if cw.cache != nil {
		cached, err := cw.cache.Get(ctx, key)
		if err == nil && len(cached) > 0 {
			var result struct {
				Items  []TransferEvent `json:"items"`
				Cursor string          `json:"cursor"`
			}
			if err := json.Unmarshal(cached, &result); err == nil {
				return result.Items, result.Cursor, nil
			}
		}
	}

	// 缓存未命中，查询数据库
	var items []TransferEvent
	q := cw.db.Model(&TransferEvent{})

	if entity != "" {
		q = q.Where("entity = ?", entity)
	}
	if chain != "" {
		q = q.Where("chain = ?", chain)
	}
	if coin != "" {
		q = q.Where("coin = ?", coin)
	}

	if cursor != "" {
		// 解析游标并应用
		// ... 游标逻辑
	}

	if err := q.Order("occurred_at DESC").Limit(limit).Find(&items).Error; err != nil {
		return nil, "", err
	}

	// 存入缓存
	if cw.cache != nil {
		result := struct {
			Items  []TransferEvent `json:"items"`
			Cursor string          `json:"cursor"`
		}{
			Items:  items,
			Cursor: "", // 计算下一个游标
		}
		data, _ := json.Marshal(result)
		cw.cache.Set(ctx, key, data, RealTimeCache.TTL)
	}

	return items, "", nil
}

// GetCachedDailyFlows 获取缓存的日度资金流
func (cw *CacheWrapper) GetCachedDailyFlows(ctx context.Context, entity string, coins []string, start, end string, runID string) ([]DailyFlow, error) {
	key := cacheKey("flows:daily", entity, fmt.Sprintf("%v", coins), start, end, runID)

	// 尝试从缓存获取
	if cw.cache != nil {
		cached, err := cw.cache.Get(ctx, key)
		if err == nil && len(cached) > 0 {
			var flows []DailyFlow
			if err := json.Unmarshal(cached, &flows); err == nil {
				return flows, nil
			}
		}
	}

	// 查询数据库
	var flows []DailyFlow
	q := cw.db.Model(&DailyFlow{}).Where("entity = ?", entity)

	if runID != "" {
		q = q.Where("run_id = ?", runID)
	}
	if len(coins) > 0 {
		q = q.Where("coin IN ?", coins)
	}
	if start != "" {
		q = q.Where("day >= ?", start)
	}
	if end != "" {
		q = q.Where("day <= ?", end)
	}

	if err := q.Order("coin ASC, day ASC").Find(&flows).Error; err != nil {
		return nil, err
	}

	// 存入缓存
	if cw.cache != nil {
		data, _ := json.Marshal(flows)
		cw.cache.Set(ctx, key, data, AggregateCache.TTL)
	}

	return flows, nil
}

// GetCachedPortfolio 获取缓存的资产组合
func (cw *CacheWrapper) GetCachedPortfolio(ctx context.Context, entity, runID string) (*PortfolioSnapshot, []Holding, error) {
	key := cacheKey("portfolio", entity, runID)

	// 尝试从缓存获取
	if cw.cache != nil {
		cached, err := cw.cache.Get(ctx, key)
		if err == nil && len(cached) > 0 {
			var result struct {
				Snapshot PortfolioSnapshot `json:"snapshot"`
				Holdings []Holding         `json:"holdings"`
			}
			if err := json.Unmarshal(cached, &result); err == nil {
				return &result.Snapshot, result.Holdings, nil
			}
		}
	}

	// 查询数据库
	var snap PortfolioSnapshot
	if err := cw.db.Where("entity = ? AND run_id = ?", entity, runID).
		Order("created_at DESC").First(&snap).Error; err != nil {
		return nil, nil, err
	}

	var holdings []Holding
	if err := cw.db.Where("run_id = ? AND entity = ?", runID, entity).
		Order("chain ASC, symbol ASC").Find(&holdings).Error; err != nil {
		return nil, nil, err
	}

	// 存入缓存
	if cw.cache != nil {
		result := struct {
			Snapshot PortfolioSnapshot `json:"snapshot"`
			Holdings []Holding         `json:"holdings"`
		}{
			Snapshot: snap,
			Holdings: holdings,
		}
		data, _ := json.Marshal(result)
		cw.cache.Set(ctx, key, data, StaticCache.TTL)
	}

	return &snap, holdings, nil
}

// ==================== 缓存失效 ====================

// InvalidateCache 失效缓存
func (cw *CacheWrapper) InvalidateCache(ctx context.Context, pattern string) error {
	if cw.cache == nil {
		return nil
	}

	// 如果缓存支持模式删除（Redis），使用模式删除
	if redisCache, ok := cw.cache.(*RedisCache); ok {
		return redisCache.DeletePattern(ctx, pattern)
	}

	// 否则直接删除特定键
	return cw.cache.Delete(ctx, pattern)
}

// InvalidateTransferCache 失效转账相关缓存
func (cw *CacheWrapper) InvalidateTransferCache(ctx context.Context, entity string) error {
	pattern := cacheKey("transfers", entity, "*")
	return cw.InvalidateCache(ctx, pattern)
}

// InvalidateFlowCache 失效资金流相关缓存
func (cw *CacheWrapper) InvalidateFlowCache(ctx context.Context, entity string) error {
	pattern := cacheKey("flows:*", entity)
	return cw.InvalidateCache(ctx, pattern)
}

// InvalidatePortfolioCache 失效资产组合相关缓存
func (cw *CacheWrapper) InvalidatePortfolioCache(ctx context.Context, entity string) error {
	pattern := cacheKey("portfolio", entity, "*")
	return cw.InvalidateCache(ctx, pattern)
}
