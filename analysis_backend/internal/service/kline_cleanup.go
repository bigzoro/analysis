package service

import (
	"context"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// KlineCleanup K线数据清理服务
type KlineCleanup struct {
	db                  pdb.Database
	interval            time.Duration
	klineRetentionDays  map[string]int // interval -> retention days
	cacheRetentionDays  int            // 缓存保留天数
	priceCacheRetention time.Duration  // 价格缓存过期时间
	stopChan            chan struct{}
	wg                  sync.WaitGroup
}

// NewKlineCleanup 创建K线数据清理服务
func NewKlineCleanup(db pdb.Database, interval time.Duration) *KlineCleanup {
	return &KlineCleanup{
		db:       db,
		interval: interval,
		klineRetentionDays: map[string]int{
			"1m":  7,    // 1分钟线保留7天
			"5m":  30,   // 5分钟线保留30天
			"15m": 90,   // 15分钟线保留90天
			"30m": 180,  // 30分钟线保留180天
			"1h":  180,  // 1小时线保留180天
			"4h":  365,  // 4小时线保留1年
			"1d":  1095, // 1天线保留3年
		},
		cacheRetentionDays:  7,              // 技术指标缓存保留7天
		priceCacheRetention: 24 * time.Hour, // 价格缓存保留24小时
		stopChan:            make(chan struct{}),
	}
}

// Start 启动清理服务
func (c *KlineCleanup) Start(ctx context.Context) {
	log.Printf("[KlineCleanup] Starting with interval %v", c.interval)

	c.wg.Add(1)
	go c.run(ctx)
}

// Stop 停止清理服务
func (c *KlineCleanup) Stop() {
	log.Printf("[KlineCleanup] Stopping...")
	close(c.stopChan)
	c.wg.Wait()
	log.Printf("[KlineCleanup] Stopped")
}

// run 运行清理循环
func (c *KlineCleanup) run(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[KlineCleanup] Context cancelled, stopping")
			return
		case <-c.stopChan:
			log.Printf("[KlineCleanup] Stop signal received, stopping")
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup 执行数据清理
func (c *KlineCleanup) cleanup() {
	log.Printf("[KlineCleanup] Starting data cleanup cycle...")

	startTime := time.Now()
	var totalCleaned int64

	gdb, err := c.db.DB()
	if err != nil {
		log.Printf("[KlineCleanup] Failed to get database connection: %v", err)
		return
	}

	// 清理过期的K线数据
	for interval, days := range c.klineRetentionDays {
		if days <= 0 {
			continue // 不清理
		}

		cleaned, err := c.cleanupKlineData(gdb, interval, days)
		if err != nil {
			log.Printf("[KlineCleanup] Failed to cleanup kline data for %s: %v", interval, err)
		} else if cleaned > 0 {
			log.Printf("[KlineCleanup] Cleaned %d expired %s klines", cleaned, interval)
			totalCleaned += cleaned
		}
	}

	// 清理过期的技术指标缓存
	cacheCleaned, err := c.cleanupTechnicalIndicatorsCache(gdb, c.cacheRetentionDays)
	if err != nil {
		log.Printf("[KlineCleanup] Failed to cleanup technical indicators cache: %v", err)
	} else if cacheCleaned > 0 {
		log.Printf("[KlineCleanup] Cleaned %d expired technical indicators cache entries", cacheCleaned)
		totalCleaned += cacheCleaned
	}

	// 清理过期的价格缓存（基于最后更新时间）
	priceCleaned, err := c.cleanupExpiredPriceCache(gdb, c.priceCacheRetention)
	if err != nil {
		log.Printf("[KlineCleanup] Failed to cleanup expired price cache: %v", err)
	} else if priceCleaned > 0 {
		log.Printf("[KlineCleanup] Cleaned %d expired price cache entries", priceCleaned)
		totalCleaned += priceCleaned
	}

	duration := time.Since(startTime)
	log.Printf("[KlineCleanup] Cleanup cycle completed in %v, total cleaned: %d", duration, totalCleaned)

	// 显示清理后的统计信息
	c.showCleanupStats(gdb)
}

// cleanupKlineData 清理指定间隔的过期K线数据
func (c *KlineCleanup) cleanupKlineData(gdb *gorm.DB, interval string, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := gdb.Where("`interval` = ? AND open_time < ?", interval, cutoffDate).Delete(&pdb.MarketKline{})

	return result.RowsAffected, result.Error
}

// cleanupTechnicalIndicatorsCache 清理过期技术指标缓存
func (c *KlineCleanup) cleanupTechnicalIndicatorsCache(gdb *gorm.DB, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := gdb.Where("calculated_at < ?", cutoffDate).Delete(&pdb.TechnicalIndicatorsCache{})

	return result.RowsAffected, result.Error
}

// cleanupExpiredPriceCache 清理过期的价格缓存
func (c *KlineCleanup) cleanupExpiredPriceCache(gdb *gorm.DB, maxAge time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-maxAge)

	result := gdb.Where("last_updated < ?", cutoffTime).Delete(&pdb.PriceCache{})

	return result.RowsAffected, result.Error
}

// showCleanupStats 显示清理后的统计信息
func (c *KlineCleanup) showCleanupStats(gdb *gorm.DB) {
	stats, err := pdb.GetKlineDataStats(gdb)
	if err != nil {
		log.Printf("[KlineCleanup] Failed to get stats: %v", err)
		return
	}

	log.Printf("[KlineCleanup] Current data stats:")
	log.Printf("  Total klines: %v", stats["total_klines"])
	log.Printf("  Total technical cache: %v", stats["total_technical_cache"])
	log.Printf("  Total price cache: %v", stats["total_price_cache"])

	// 显示各间隔的数据量
	if intervalStats, ok := stats["interval_stats"].([]struct {
		Interval string `json:"interval"`
		Count    int64  `json:"count"`
	}); ok {
		for _, stat := range intervalStats {
			log.Printf("  %s klines: %d", stat.Interval, stat.Count)
		}
	}
}

// ============================================================================
// 手动清理方法（供命令行工具调用）
// ============================================================================

// CleanupExpiredData 执行一次性数据清理
func (c *KlineCleanup) CleanupExpiredData() error {
	log.Printf("[KlineCleanup] Starting manual cleanup...")

	c.cleanup()

	log.Printf("[KlineCleanup] Manual cleanup completed")
	return nil
}

// GetCleanupStats 获取清理统计信息
func (c *KlineCleanup) GetCleanupStats() (map[string]interface{}, error) {
	gdb, err := c.db.DB()
	if err != nil {
		return nil, err
	}

	return pdb.GetKlineDataStats(gdb)
}
