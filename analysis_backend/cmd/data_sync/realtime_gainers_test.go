package main

import (
	"context"
	"log"
	"testing"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestRealtimeGainersSyncer 实时涨幅榜同步器测试
func TestRealtimeGainersSyncer(t *testing.T) {
	// 创建测试数据库连接
	db, err := createTestDB()
	if err != nil {
		t.Skipf("跳过测试：无法连接数据库: %v", err)
		return
	}

	// 创建测试配置
	cfg := &config.Config{}
	config := &DataSyncConfig{
		EnableRealtimeGainers: true,
	}

	// 创建同步器
	syncer := NewRealtimeGainersSyncer(db, cfg, config)
	if syncer == nil {
		t.Fatal("创建RealtimeGainersSyncer失败")
	}

	// 测试基本功能
	t.Run("BasicFunctionality", func(t *testing.T) {
		testBasicFunctionality(t, syncer)
	})

	// 测试性能
	t.Run("PerformanceTest", func(t *testing.T) {
		testPerformance(t, syncer)
	})

	// 测试统计信息
	t.Run("StatsTest", func(t *testing.T) {
		testStats(t, syncer)
	})
}

// testBasicFunctionality 测试基本功能
func testBasicFunctionality(t *testing.T, syncer *RealtimeGainersSyncer) {
	// 测试获取热门交易对
	topSymbols := syncer.getTopSymbolsFromDB()
	if len(topSymbols) == 0 {
		t.Log("警告：没有找到热门交易对，可能数据库为空")
		return
	}

	t.Logf("找到 %d 个热门交易对: %v", len(topSymbols), topSymbols[:min(5, len(topSymbols))])

	// 测试涨幅榜计算
	syncer.recalculateGainers()

	// 检查是否生成了涨幅榜
	syncer.currentGainersMux.RLock()
	gainersCount := len(syncer.currentGainers)
	syncer.currentGainersMux.RUnlock()

	if gainersCount == 0 {
		t.Log("警告：没有生成涨幅榜数据")
		return
	}

	t.Logf("生成了 %d 个涨幅榜项目", gainersCount)

	// 验证排名连续性
	for i, gainer := range syncer.currentGainers {
		expectedRank := i + 1
		if gainer.Rank != expectedRank {
			t.Errorf("排名不连续：期望 %d，实际 %d", expectedRank, gainer.Rank)
		}
	}
}

// testPerformance 测试性能
func testPerformance(t *testing.T, syncer *RealtimeGainersSyncer) {
	// 模拟价格更新
	testSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}

	startTime := time.Now()

	// 发送多个价格更新
	for i := 0; i < 10; i++ {
		for _, symbol := range testSymbols {
			price := 50000.0 + float64(i)*100 // 模拟价格变化
			update := PriceUpdate{
				Symbol:    symbol,
				Price:     price,
				Volume:    1000.0 + float64(i)*10,
				Timestamp: time.Now(),
				Source:    "test",
			}

			select {
			case syncer.priceUpdateChan <- update:
				// 发送成功
			default:
				t.Log("价格更新通道已满，跳过")
			}
		}

		// 等待处理
		time.Sleep(100 * time.Millisecond)
	}

	processingTime := time.Since(startTime)
	t.Logf("处理10轮价格更新耗时: %v", processingTime)

	// 检查统计信息
	stats := syncer.GetInternalStats()
	updatesReceived := stats["price_updates_received"].(int64)
	calculations := stats["gainers_calculations"].(int64)

	t.Logf("接收价格更新: %d, 执行计算: %d", updatesReceived, calculations)

	// 性能断言
	if processingTime > 5*time.Second {
		t.Errorf("处理时间过长: %v", processingTime)
	}
}

// testStats 测试统计信息
func testStats(t *testing.T, syncer *RealtimeGainersSyncer) {
	stats := syncer.GetInternalStats()

	// 检查必要的统计字段
	requiredFields := []string{
		"is_running",
		"start_time",
		"uptime",
		"price_updates_received",
		"gainers_calculations",
		"errors_count",
	}

	for _, field := range requiredFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("缺少统计字段: %s", field)
		}
	}

	t.Logf("统计信息完整，包含 %d 个字段", len(stats))

	// 打印关键统计信息
	t.Logf("运行状态: %v", stats["is_running"])
	t.Logf("运行时间: %v", stats["uptime"])
	t.Logf("价格更新: %d", stats["price_updates_received"])
	t.Logf("涨幅计算: %d", stats["gainers_calculations"])
	t.Logf("错误次数: %d", stats["errors_count"])
}

// createTestDB 创建测试数据库连接
func createTestDB() (*gorm.DB, error) {
	// 这里应该使用测试数据库配置
	// 为了演示，我们使用环境变量或默认配置

	dsn := "user:password@tcp(localhost:3306)/analysis_test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// min 返回两个数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BenchmarkRealtimeGainersSyncer 基准测试
func BenchmarkRealtimeGainersSyncer(b *testing.B) {
	// 创建同步器（使用nil数据库进行基准测试）
	cfg := &config.Config{}
	config := &DataSyncConfig{}
	syncer := NewRealtimeGainersSyncer(nil, cfg, config)

	// 重置基准测试计时器
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 模拟价格更新处理
		update := PriceUpdate{
			Symbol:    "BTCUSDT",
			Price:     50000.0,
			Volume:    1000.0,
			Timestamp: time.Now(),
			Source:    "benchmark",
		}

		// 这里可以测试价格缓存的性能
		if syncer.priceCache != nil {
			syncer.priceCache.UpdatePrice(update)
		}
	}
}

// TestChangeDetector 测试变化检测器
func TestChangeDetector(t *testing.T) {
	detector := NewChangeDetector()

	// 测试初始状态
	if detector.HasSignificantChanges([]RealtimeGainerItem{}) {
		t.Log("初始状态下应该没有显著变化")
	}

	// 创建测试数据
	gainers := []RealtimeGainerItem{
		{Symbol: "BTCUSDT", Rank: 1, ChangePercent: 2.5},
		{Symbol: "ETHUSDT", Rank: 2, ChangePercent: 1.8},
		{Symbol: "BNBUSDT", Rank: 3, ChangePercent: 0.5},
	}

	// 设置最后保存状态
	detector.UpdateLastGainers(gainers)

	// 测试无变化
	if detector.HasSignificantChanges(gainers) {
		t.Error("相同数据应该没有显著变化")
	}

	// 测试有变化
	changedGainers := []RealtimeGainerItem{
		{Symbol: "BTCUSDT", Rank: 1, ChangePercent: 5.0}, // 大幅上涨
		{Symbol: "ETHUSDT", Rank: 2, ChangePercent: 1.8},
		{Symbol: "BNBUSDT", Rank: 3, ChangePercent: 0.5},
	}

	if !detector.HasSignificantChanges(changedGainers) {
		t.Error("价格大幅变化应该检测到显著变化")
	}

	t.Log("变化检测器测试通过")
}

// TestPriceCache 测试价格缓存
func TestPriceCache(t *testing.T) {
	cache := NewRealtimePriceCache()

	// 测试价格更新
	update := PriceUpdate{
		Symbol:    "BTCUSDT",
		Price:     50000.0,
		Volume:    1000.0,
		Timestamp: time.Now(),
		Source:    "test",
	}

	cache.UpdatePrice(update)

	// 测试价格获取
	priceData, exists := cache.GetPrice("BTCUSDT")
	if !exists {
		t.Error("应该能够获取刚更新的价格")
	}

	if priceData.LastPrice != 50000.0 {
		t.Errorf("价格不匹配：期望 50000.0，实际 %.4f", priceData.LastPrice)
	}

	// 测试缓存统计
	stats := cache.GetCacheStats()
	if stats["entries_count"].(int) != 1 {
		t.Errorf("缓存条目数不正确：期望 1，实际 %d", stats["entries_count"])
	}

	t.Log("价格缓存测试通过")
}

// 性能测试报告生成
func generatePerformanceReport(syncer *RealtimeGainersSyncer) {
	log.Println("=== 实时涨幅榜同步器性能报告 ===")

	stats := syncer.GetInternalStats()

	log.Printf("运行状态: %v", stats["is_running"])
	log.Printf("运行时间: %s", stats["uptime"])
	log.Printf("WebSocket连接: %d", stats["active_ws_connections"])
	log.Printf("价格更新接收: %d", stats["price_updates_received"])
	log.Printf("涨幅榜计算: %d", stats["gainers_calculations"])
	log.Printf("数据保存触发: %d", stats["saves_triggered"])
	log.Printf("平均计算时间: %s", stats["avg_calculation_time"])
	log.Printf("平均保存时间: %s", stats["avg_save_time"])
	log.Printf("错误次数: %d", stats["errors_count"])

	if err := stats["last_error"]; err != nil && err != "nil" {
		log.Printf("最后错误: %v", err)
	}

	log.Println("=== 报告结束 ===")
}