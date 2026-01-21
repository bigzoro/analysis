package server

import (
	"context"
	"encoding/json"
	"log"
	"time"

	pdb "analysis/internal/db"
)

// ==================== 缓存预热机制 ====================

// CacheWarmup 缓存预热器
type CacheWarmup struct {
	server *Server
}

// NewCacheWarmup 创建缓存预热器
func NewCacheWarmup(server *Server) *CacheWarmup {
	return &CacheWarmup{server: server}
}

// WarmupCommonData 预热常用数据
func (cw *CacheWarmup) WarmupCommonData(ctx context.Context) error {
	if cw.server.cache == nil {
		return nil
	}

	log.Println("[CacheWarmup] Starting cache warmup...")

	// 预热实体列表
	if err := cw.warmupEntities(ctx); err != nil {
		log.Printf("[CacheWarmup] Failed to warmup entities: %v", err)
	}

	// 预热黑名单（常用类型）
	for _, kind := range []string{"spot", "futures"} {
		if err := cw.warmupBlacklist(ctx, kind); err != nil {
			log.Printf("[CacheWarmup] Failed to warmup blacklist (kind=%s): %v", kind, err)
		}
	}

	// 预热推荐数据
	if err := cw.warmupRecommendations(ctx); err != nil {
		log.Printf("[CacheWarmup] Failed to warmup recommendations: %v", err)
	}

	log.Println("[CacheWarmup] Cache warmup completed")
	return nil
}

// warmupEntities 预热实体列表
func (cw *CacheWarmup) warmupEntities(ctx context.Context) error {
	entities, err := cw.server.db.ListEntities()
	if err != nil {
		return err
	}

	// 实体列表通常变化不频繁，可以缓存较长时间
	key := "cache:v1:entities:list"
	data, err := json.Marshal(entities)
	if err != nil {
		return err
	}

	// 使用静态数据 TTL（30分钟）
	ttl := pdb.DefaultCacheTTL.GetTTL(pdb.CacheTypeStatic)
	return cw.server.cache.Set(ctx, key, data, ttl)
}

// warmupBlacklist 预热黑名单
func (cw *CacheWarmup) warmupBlacklist(ctx context.Context, kind string) error {
	blacklist, err := cw.server.db.GetBinanceBlacklist(kind)
	if err != nil {
		return err
	}

	key := BuildCacheKey("cache:v1:blacklist", kind)
	data, err := json.Marshal(blacklist)
	if err != nil {
		return err
	}

	// 使用静态数据 TTL（30分钟）
	ttl := pdb.DefaultCacheTTL.GetTTL(pdb.CacheTypeStatic)
	return cw.server.cache.Set(ctx, key, data, ttl)
}

// WarmupPortfolio 预热投资组合数据（针对常用实体）
func (cw *CacheWarmup) WarmupPortfolio(ctx context.Context, entities []string) error {
	if cw.server.cache == nil {
		return nil
	}

	for _, entity := range entities {
		// 获取最新快照
		snap, err := cw.server.db.GetLatestPortfolioSnapshot(entity)
		if err != nil {
			log.Printf("[CacheWarmup] Failed to get latest portfolio for %s: %v", entity, err)
			continue
		}

		// 获取持仓
		holdings, err := cw.server.db.GetHoldingsByRunID(snap.RunID, entity)
		if err != nil {
			log.Printf("[CacheWarmup] Failed to get holdings for %s: %v", entity, err)
			continue
		}

		// 构建缓存数据
		result := struct {
			Snapshot *pdb.PortfolioSnapshot `json:"snapshot"`
			Holdings []pdb.Holding          `json:"holdings"`
		}{
			Snapshot: snap,
			Holdings: holdings,
		}

		data, err := json.Marshal(result)
		if err != nil {
			log.Printf("[CacheWarmup] Failed to marshal portfolio data for %s: %v", entity, err)
			continue
		}

		// 设置缓存
		key := BuildCacheKey("cache:v1:portfolio:latest", entity)
		ttl := pdb.DefaultCacheTTL.GetTTL(pdb.CacheTypeRealTime)
		if err := cw.server.cache.Set(ctx, key, data, ttl); err != nil {
			log.Printf("[CacheWarmup] Failed to set portfolio cache for %s: %v", entity, err)
		}
	}

	return nil
}

// warmupRecommendations 预热推荐数据
func (cw *CacheWarmup) warmupRecommendations(ctx context.Context) error {
	if cw.server.recommendationCache == nil {
		log.Println("[CacheWarmup] Recommendation cache not available, skipping recommendation warmup")
		return nil
	}

	log.Println("[CacheWarmup] Warming up recommendation cache...")

	// 预热常用推荐查询
	commonQueries := []RecommendationQueryParams{
		{Kind: "spot", Limit: 5},
		{Kind: "futures", Limit: 5},
		{Kind: "spot", Limit: 10},
		{Kind: "futures", Limit: 10},
	}

	for _, query := range commonQueries {
		log.Printf("[CacheWarmup] Warming up recommendation: kind=%s, limit=%d", query.Kind, query.Limit)

		// 调用推荐缓存预热
		if _, err := cw.server.recommendationCache.GetRecommendationsWithCache(ctx, query); err != nil {
			log.Printf("[CacheWarmup] Failed to warmup recommendation (kind=%s, limit=%d): %v", query.Kind, query.Limit, err)
		} else {
			log.Printf("[CacheWarmup] Successfully warmed up recommendation: kind=%s, limit=%d", query.Kind, query.Limit)
		}
	}

	log.Println("[CacheWarmup] Recommendation cache warmup completed")
	return nil
}

// StartPeriodicWarmup 启动定期预热
func (cw *CacheWarmup) StartPeriodicWarmup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次
	_ = cw.WarmupCommonData(ctx)

	for {
		select {
		case <-ticker.C:
			_ = cw.WarmupCommonData(ctx)
		case <-ctx.Done():
			return
		}
	}
}
