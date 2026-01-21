package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// MarketDataCompressionService 市场数据压缩服务
type MarketDataCompressionService struct {
	db                pdb.Database
	compressAfterDays int
	interval          time.Duration
	stopChan          chan struct{}
	wg                sync.WaitGroup
}

// NewMarketDataCompressionService 创建市场数据压缩服务
func NewMarketDataCompressionService(db *pdb.Database, compressAfterDays int, interval time.Duration) *MarketDataCompressionService {
	if compressAfterDays <= 0 {
		compressAfterDays = 90 // 默认90天后压缩
	}
	if interval <= 0 {
		interval = 7 * 24 * time.Hour // 默认每周执行一次
	}

	return &MarketDataCompressionService{
		db:                *db,
		compressAfterDays: compressAfterDays,
		interval:          interval,
		stopChan:          make(chan struct{}),
	}
}

// Start 启动压缩服务
func (s *MarketDataCompressionService) Start(ctx context.Context) {
	log.Printf("[MarketDataCompressionService] Starting with compress_after_days=%d, interval=%v",
		s.compressAfterDays, s.interval)

	s.wg.Add(1)
	go s.run(ctx)
}

// Stop 停止压缩服务
func (s *MarketDataCompressionService) Stop() {
	log.Printf("[MarketDataCompressionService] Stopping...")
	close(s.stopChan)
	s.wg.Wait()
	log.Printf("[MarketDataCompressionService] Stopped")
}

// run 运行压缩循环
func (s *MarketDataCompressionService) run(ctx context.Context) {
	defer s.wg.Done()

	// 启动时立即执行一次
	s.performCompression()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[MarketDataCompressionService] Context cancelled, stopping")
			return
		case <-s.stopChan:
			log.Printf("[MarketDataCompressionService] Stop signal received, stopping")
			return
		case <-ticker.C:
			s.performCompression()
		}
	}
}

// performCompression 执行压缩操作
func (s *MarketDataCompressionService) performCompression() {
	log.Printf("[MarketDataCompressionService] Starting compression cycle...")

	startTime := time.Now()

	// 获取数据库连接
	gdb, err := s.db.DB()
	if err != nil {
		log.Printf("[MarketDataCompressionService] Failed to get database connection: %v", err)
		return
	}

	// 显示压缩前的统计信息
	if stats, err := pdb.GetDataCompressionStats(gdb); err == nil {
		log.Printf("[MarketDataCompressionService] Pre-compression stats: %+v", stats)
	}

	// 执行数据压缩
	err = pdb.CompressOldMarketData(gdb, s.compressAfterDays)
	if err != nil {
		log.Printf("[MarketDataCompressionService] Compression failed: %v", err)
		return
	}

	// 显示压缩后的统计信息
	if stats, err := pdb.GetDataCompressionStats(gdb); err == nil {
		log.Printf("[MarketDataCompressionService] Post-compression stats: %+v", stats)
	}

	log.Printf("[MarketDataCompressionService] Compression cycle completed in %v", time.Since(startTime))
}

// CompressOnce 执行一次性压缩
func (s *MarketDataCompressionService) CompressOnce() error {
	log.Printf("[MarketDataCompressionService] Performing one-time compression...")

	gdb, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	return pdb.CompressOldMarketData(gdb, s.compressAfterDays)
}

// GetStats 获取压缩统计信息
func (s *MarketDataCompressionService) GetStats() (map[string]interface{}, error) {
	gdb, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	return pdb.GetDataCompressionStats(gdb)
}
