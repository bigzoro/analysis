package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// KlineWarmer K线数据预热服务
type KlineWarmer struct {
	db         pdb.Database
	interval   time.Duration
	symbols    []string
	kinds      []string
	intervals  []string
	dataPoints map[string]int // interval -> dataPoints
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewKlineWarmer 创建K线数据预热服务
func NewKlineWarmer(db pdb.Database, interval time.Duration) *KlineWarmer {
	return &KlineWarmer{
		db:        db,
		interval:  interval,
		symbols:   []string{}, // 动态从活跃交易对获取
		kinds:     []string{"spot", "futures"},
		intervals: []string{"1h", "4h", "1d"}, // 只预热常用间隔
		dataPoints: map[string]int{
			"1h": 200, // 200小时数据
			"4h": 150, // 150个4小时数据
			"1d": 100, // 100天数据
		},
		stopChan: make(chan struct{}),
	}
}

// Start 启动预热服务
func (w *KlineWarmer) Start(ctx context.Context) {
	log.Printf("[KlineWarmer] Starting with interval %v", w.interval)

	w.wg.Add(1)
	go w.run(ctx)
}

// Stop 停止预热服务
func (w *KlineWarmer) Stop() {
	log.Printf("[KlineWarmer] Stopping...")
	close(w.stopChan)
	w.wg.Wait()
	log.Printf("[KlineWarmer] Stopped")
}

// run 运行预热循环
func (w *KlineWarmer) run(ctx context.Context) {
	defer w.wg.Done()

	// 启动时立即执行一次
	w.warmup()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[KlineWarmer] Context cancelled, stopping")
			return
		case <-w.stopChan:
			log.Printf("[KlineWarmer] Stop signal received, stopping")
			return
		case <-ticker.C:
			w.warmup()
		}
	}
}

// warmup 执行数据预热
func (w *KlineWarmer) warmup() {
	log.Printf("[KlineWarmer] Starting data warmup cycle...")

	startTime := time.Now()

	// 获取活跃交易对
	activeSymbols, err := w.getActiveSymbols()
	if err != nil {
		log.Printf("[KlineWarmer] Failed to get active symbols: %v", err)
		return
	}

	log.Printf("[KlineWarmer] Found %d active symbols to warm up", len(activeSymbols))

	// 预热计数器
	totalErrors := 0

	// 为每个交易对预热数据
	semaphore := make(chan struct{}, 5) // 限制并发数为5
	errorChan := make(chan error, len(activeSymbols)*len(w.kinds)*len(w.intervals))

	for _, symbol := range activeSymbols {
		for _, kind := range w.kinds {
			for _, interval := range w.intervals {
				go func(sym, k, intv string) {
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					if err := w.warmupSymbol(sym, k, intv); err != nil {
						errorChan <- err
					}
				}(symbol, kind, interval)
			}
		}
	}

	// 等待所有goroutine完成
	for i := 0; i < 5; i++ {
		semaphore <- struct{}{}
	}

	// 收集错误统计
	close(errorChan)
	for err := range errorChan {
		log.Printf("[KlineWarmer] Warmup error: %v", err)
		totalErrors++
	}

	duration := time.Since(startTime)
	log.Printf("[KlineWarmer] Warmup cycle completed in %v, errors: %d", duration, totalErrors)

	// 显示统计信息
	w.showWarmupStats()
}

// warmupSymbol 预热单个交易对的数据
func (w *KlineWarmer) warmupSymbol(symbol, kind, interval string) error {
	gdb, err := w.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	dataPoints := w.dataPoints[interval]

	// 检查数据是否已经足够新鲜
	maxAge := getMaxAgeForInterval(interval)
	isFresh, err := pdb.IsKlineDataFresh(gdb, symbol, kind, interval, maxAge)
	if err != nil {
		return fmt.Errorf("failed to check data freshness: %w", err)
	}

	if isFresh {
		// 数据已经足够新鲜，跳过
		return nil
	}

	// 这里需要调用Server的方法来获取和保存K线数据
	// 由于这是一个服务层，我们需要通过接口或回调来实现
	// 暂时返回nil，表示跳过实际的API调用
	// 实际实现时，需要注入一个获取K线数据的函数

	log.Printf("[KlineWarmer] Would warm up %s %s %s with %d data points", symbol, kind, interval, dataPoints)
	return nil
}

// getActiveSymbols 获取活跃交易对列表
func (w *KlineWarmer) getActiveSymbols() ([]string, error) {
	gdb, err := w.db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// 从最近的市场快照中获取活跃交易对
	now := time.Now().UTC()
	startTime := now.Add(-24 * time.Hour) // 最近24小时的数据

	snaps, tops, err := pdb.ListBinanceMarket(gdb, "spot", startTime, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get market snapshots: %w", err)
	}

	if len(snaps) == 0 {
		return []string{}, nil
	}

	// 收集所有唯一的交易对
	symbolSet := make(map[string]bool)
	for _, snap := range snaps {
		if items, ok := tops[snap.ID]; ok {
			for _, item := range items {
				// 只选择有成交量的活跃交易对
				if volume, err := strconv.ParseFloat(item.Volume, 64); err == nil && volume > 1000000 { // 成交量大于100万
					symbolSet[item.Symbol] = true
				}
			}
		}
	}

	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}

	// 限制数量，避免预热过多数据
	if len(symbols) > 100 {
		symbols = symbols[:100]
	}

	return symbols, nil
}

// showWarmupStats 显示预热统计信息
func (w *KlineWarmer) showWarmupStats() {
	gdb, err := w.db.DB()
	if err != nil {
		log.Printf("[KlineWarmer] Failed to get database connection for stats")
		return
	}

	stats, err := pdb.GetKlineDataStats(gdb)
	if err != nil {
		log.Printf("[KlineWarmer] Failed to get stats: %v", err)
		return
	}

	log.Printf("[KlineWarmer] Data stats:")
	log.Printf("  Total klines: %v", stats["total_klines"])
	log.Printf("  Total technical cache: %v", stats["total_technical_cache"])
	log.Printf("  Total price cache: %v", stats["total_price_cache"])
	if oldest, ok := stats["oldest_kline"].(string); ok {
		log.Printf("  Oldest data: %s", oldest)
	}
	if newest, ok := stats["newest_kline"].(string); ok {
		log.Printf("  Newest data: %s", newest)
	}
}

// ============================================================================
// 工具函数
// ============================================================================

// getMaxAgeForInterval 根据K线间隔获取最大年龄
func getMaxAgeForInterval(interval string) time.Duration {
	switch interval {
	case "1m":
		return 5 * time.Minute
	case "5m":
		return 15 * time.Minute
	case "15m":
		return 30 * time.Minute
	case "30m":
		return 1 * time.Hour
	case "1h":
		return 2 * time.Hour
	case "4h":
		return 6 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}
