package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// MarketDataCleanupService 市场数据清理服务
type MarketDataCleanupService struct {
	db            pdb.Database
	retentionDays int
	fullMode      bool
	interval      time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewMarketDataCleanupService 创建市场数据清理服务
func NewMarketDataCleanupService(db *pdb.Database, retentionDays int, fullMode bool, interval time.Duration) *MarketDataCleanupService {
	if retentionDays <= 0 {
		retentionDays = 30 // 默认30天
	}
	if interval <= 0 {
		interval = 24 * time.Hour // 默认每天执行一次
	}

	return &MarketDataCleanupService{
		db:            *db,
		retentionDays: retentionDays,
		fullMode:      fullMode,
		interval:      interval,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动清理服务
func (s *MarketDataCleanupService) Start(ctx context.Context) {
	log.Printf("[MarketDataCleanupService] Starting with retention_days=%d, full_mode=%v, interval=%v",
		s.retentionDays, s.fullMode, s.interval)

	s.wg.Add(1)
	go s.run(ctx)
}

// Stop 停止清理服务
func (s *MarketDataCleanupService) Stop() {
	log.Printf("[MarketDataCleanupService] Stopping...")
	close(s.stopChan)
	s.wg.Wait()
	log.Printf("[MarketDataCleanupService] Stopped")
}

// run 运行清理循环
func (s *MarketDataCleanupService) run(ctx context.Context) {
	defer s.wg.Done()

	// 启动时立即执行一次
	s.performCleanup()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[MarketDataCleanupService] Context cancelled, stopping")
			return
		case <-s.stopChan:
			log.Printf("[MarketDataCleanupService] Stop signal received, stopping")
			return
		case <-ticker.C:
			s.performCleanup()
		}
	}
}

// performCleanup 执行清理操作
func (s *MarketDataCleanupService) performCleanup() {
	log.Printf("[MarketDataCleanupService] Starting cleanup cycle...")

	startTime := time.Now()

	// 获取数据库连接
	gdb, err := s.db.DB()
	if err != nil {
		log.Printf("[MarketDataCleanupService] Failed to get database connection: %v", err)
		return
	}

	// 执行数据清理
	err = pdb.CleanupOldMarketData(gdb, s.retentionDays, s.fullMode)
	if err != nil {
		log.Printf("[MarketDataCleanupService] Cleanup failed: %v", err)
		return
	}

	// 获取统计信息
	stats, err := pdb.GetMarketDataStats(gdb)
	if err != nil {
		log.Printf("[MarketDataCleanupService] Failed to get stats: %v", err)
	} else {
		log.Printf("[MarketDataCleanupService] Cleanup completed in %v. Stats: total_snapshots=%v, total_market_data=%v, unique_symbols=%v",
			time.Since(startTime),
			stats["total_snapshots"],
			stats["total_market_data"],
			stats["unique_symbols"])
	}
}

// CleanupOnce 执行一次性清理
func (s *MarketDataCleanupService) CleanupOnce() error {
	log.Printf("[MarketDataCleanupService] Performing one-time cleanup...")

	gdb, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	return pdb.CleanupOldMarketData(gdb, s.retentionDays, s.fullMode)
}

// GetStats 获取当前统计信息
func (s *MarketDataCleanupService) GetStats() (map[string]interface{}, error) {
	gdb, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	return pdb.GetMarketDataStats(gdb)
}
