package db

import (
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// createTestDB 创建测试数据库连接
func createTestDB(t *testing.T) *gorm.DB {
	// 测试数据库配置 - 请根据实际情况修改
	dsn := "root:@tcp(localhost:3306)/analysis_test?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 测试时减少日志输出
	})
	if err != nil {
		t.Skipf("跳过测试：无法连接测试数据库: %v", err)
		return nil
	}

	// 自动迁移测试表结构
	err = db.AutoMigrate(&Binance24hStatsHistory{})
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	return db
}

// TestSave24hStatsHistory 测试历史统计数据保存功能
func TestSave24hStatsHistory(t *testing.T) {
	db := createTestDB(t)
	if db == nil {
		return
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol LIKE 'TEST%'")

	// 创建测试数据
	now := time.Now()
	testStats := []Binance24hStatsHistory{
		{
			Symbol:             "TESTBTC",
			MarketType:         "spot",
			WindowStart:        now.Truncate(time.Hour),
			WindowEnd:          now.Truncate(time.Hour).Add(time.Hour),
			WindowDuration:     3600,
			PriceChange:        100.50,
			PriceChangePercent: 2.5,
			LastPrice:          42000.00,
			Volume:             1500.75,
			QuoteVolume:        63031500.00,
			CreatedAt:          now,
		},
		{
			Symbol:             "TESTETH",
			MarketType:         "spot",
			WindowStart:        now.Truncate(time.Hour),
			WindowEnd:          now.Truncate(time.Hour).Add(time.Hour),
			WindowDuration:     3600,
			PriceChange:        50.25,
			PriceChangePercent: 1.8,
			LastPrice:          2800.00,
			Volume:             2500.50,
			QuoteVolume:        7001400.00,
			CreatedAt:          now,
		},
	}

	// 测试保存功能
	err := Save24hStatsHistory(db, testStats)
	if err != nil {
		t.Fatalf("保存历史统计数据失败: %v", err)
	}

	// 验证数据是否正确保存
	var count int64
	db.Model(&Binance24hStatsHistory{}).Where("symbol LIKE 'TEST%'").Count(&count)
	if count != 2 {
		t.Errorf("期望保存2条记录，实际保存%d条", count)
	}

	// 验证数据内容
	var savedStats []Binance24hStatsHistory
	db.Where("symbol LIKE 'TEST%'").Find(&savedStats)

	for _, expected := range testStats {
		found := false
		for _, actual := range savedStats {
			if actual.Symbol == expected.Symbol && actual.WindowStart.Equal(expected.WindowStart) {
				found = true
				if actual.PriceChange != expected.PriceChange {
					t.Errorf("TESTBTC价格变化不匹配: 期望%.2f, 实际%.2f", expected.PriceChange, actual.PriceChange)
				}
				if actual.LastPrice != expected.LastPrice {
					t.Errorf("%s最新价格不匹配: 期望%.2f, 实际%.2f", expected.Symbol, expected.LastPrice, actual.LastPrice)
				}
				break
			}
		}
		if !found {
			t.Errorf("未找到保存的记录: %s", expected.Symbol)
		}
	}

	// 测试去重功能 - 再次保存相同数据
	err = Save24hStatsHistory(db, testStats)
	if err != nil {
		t.Fatalf("重复保存测试失败: %v", err)
	}

	// 验证记录数没有增加（去重生效）
	db.Model(&Binance24hStatsHistory{}).Where("symbol LIKE 'TEST%'").Count(&count)
	if count != 2 {
		t.Errorf("去重功能失败：期望仍然是2条记录，实际变为%d条", count)
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol LIKE 'TEST%'")
}

// TestGet24hStatsHistory 测试历史统计数据查询功能
func TestGet24hStatsHistory(t *testing.T) {
	db := createTestDB(t)
	if db == nil {
		return
	}

	// 清理并准备测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'QUERYTEST'")

	now := time.Now()
	windowStart := now.Truncate(time.Hour)
	testStats := []Binance24hStatsHistory{
		{
			Symbol:         "QUERYTEST",
			MarketType:     "spot",
			WindowStart:    windowStart,
			WindowEnd:      windowStart.Add(time.Hour),
			WindowDuration: 3600,
			LastPrice:      50000.00,
			Volume:         1000.00,
			CreatedAt:      now,
		},
		{
			Symbol:         "QUERYTEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(time.Hour),
			WindowEnd:      windowStart.Add(2 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      51000.00,
			Volume:         1100.00,
			CreatedAt:      now.Add(time.Hour),
		},
		{
			Symbol:         "QUERYTEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(2 * time.Hour),
			WindowEnd:      windowStart.Add(3 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      52000.00,
			Volume:         1200.00,
			CreatedAt:      now.Add(2 * time.Hour),
		},
	}

	err := Save24hStatsHistory(db, testStats)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 测试时间范围查询
	startTime := windowStart
	endTime := windowStart.Add(2 * time.Hour)

	results, err := Get24hStatsHistory(db, "QUERYTEST", "spot", startTime, endTime)
	if err != nil {
		t.Fatalf("查询历史统计数据失败: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("期望查询到2条记录，实际查询到%d条", len(results))
	}

	// 验证查询结果按时间排序
	if len(results) >= 2 {
		if !results[0].WindowStart.Before(results[1].WindowStart) {
			t.Error("查询结果未按时间正序排序")
		}
	}

	// 验证数据内容
	found := make(map[time.Time]bool)
	for _, result := range results {
		found[result.WindowStart] = true
		if result.LastPrice < 50000.00 || result.LastPrice > 52000.00 {
			t.Errorf("查询到的价格超出预期范围: %.2f", result.LastPrice)
		}
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'QUERYTEST'")
}

// TestGetLatest24hStatsHistory 测试获取最新历史统计数据
func TestGetLatest24hStatsHistory(t *testing.T) {
	db := createTestDB(t)
	if db == nil {
		return
	}

	// 清理并准备测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'LATESTTEST'")

	now := time.Now()
	windowStart := now.Truncate(time.Hour)
	testStats := []Binance24hStatsHistory{
		{
			Symbol:         "LATESTTEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(-2 * time.Hour),
			WindowEnd:      windowStart.Add(-time.Hour),
			WindowDuration: 3600,
			LastPrice:      48000.00,
			CreatedAt:      now.Add(-2 * time.Hour),
		},
		{
			Symbol:         "LATESTTEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(-time.Hour),
			WindowEnd:      windowStart,
			WindowDuration: 3600,
			LastPrice:      49000.00,
			CreatedAt:      now.Add(-time.Hour),
		},
		{
			Symbol:         "LATESTTEST",
			MarketType:     "spot",
			WindowStart:    windowStart,
			WindowEnd:      windowStart.Add(time.Hour),
			WindowDuration: 3600,
			LastPrice:      50000.00,
			CreatedAt:      now,
		},
	}

	err := Save24hStatsHistory(db, testStats)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 测试获取最新记录
	latest, err := GetLatest24hStatsHistory(db, "LATESTTEST", "spot")
	if err != nil {
		t.Fatalf("获取最新历史统计数据失败: %v", err)
	}

	if latest == nil {
		t.Fatal("期望获取到最新记录，但返回nil")
	}

	if latest.LastPrice != 50000.00 {
		t.Errorf("最新记录价格不正确: 期望50000.00, 实际%.2f", latest.LastPrice)
	}

	if !latest.WindowStart.Equal(windowStart) {
		t.Errorf("最新记录时间窗口不正确: 期望%s, 实际%s", windowStart, latest.WindowStart)
	}

	// 测试不存在的记录
	nonExistent, err := GetLatest24hStatsHistory(db, "NONEXISTENT", "spot")
	if err != nil {
		t.Fatalf("查询不存在记录时出错: %v", err)
	}
	if nonExistent != nil {
		t.Error("查询不存在记录时应该返回nil")
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'LATESTTEST'")
}

// TestDeleteExpired24hStatsHistory 测试删除过期历史数据
func TestDeleteExpired24hStatsHistory(t *testing.T) {
	db := createTestDB(t)
	if db == nil {
		return
	}

	// 清理并准备测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'DELETETEST'")

	now := time.Now()
	testStats := []Binance24hStatsHistory{
		{
			Symbol:         "DELETETEST",
			MarketType:     "spot",
			WindowStart:    now.Add(-48 * time.Hour), // 2天前
			WindowEnd:      now.Add(-47 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      45000.00,
			CreatedAt:      now.Add(-48 * time.Hour),
		},
		{
			Symbol:         "DELETETEST",
			MarketType:     "spot",
			WindowStart:    now.Add(-2 * time.Hour), // 2小时前
			WindowEnd:      now.Add(-time.Hour),
			WindowDuration: 3600,
			LastPrice:      48000.00,
			CreatedAt:      now.Add(-2 * time.Hour),
		},
		{
			Symbol:         "DELETETEST",
			MarketType:     "spot",
			WindowStart:    now.Add(-time.Hour), // 1小时前
			WindowEnd:      now,
			WindowDuration: 3600,
			LastPrice:      49000.00,
			CreatedAt:      now.Add(-time.Hour),
		},
	}

	err := Save24hStatsHistory(db, testStats)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 验证初始数据
	var initialCount int64
	db.Model(&Binance24hStatsHistory{}).Where("symbol = 'DELETETEST'").Count(&initialCount)
	if initialCount != 3 {
		t.Errorf("期望初始有3条记录，实际有%d条", initialCount)
	}

	// 删除24小时前的过期数据
	cutoffTime := now.Add(-24 * time.Hour)
	err = DeleteExpired24hStatsHistory(db, cutoffTime)
	if err != nil {
		t.Fatalf("删除过期数据失败: %v", err)
	}

	// 验证删除结果 - 应该只剩下2条记录（24小时内和2小时前的）
	var remainingCount int64
	db.Model(&Binance24hStatsHistory{}).Where("symbol = 'DELETETEST'").Count(&remainingCount)
	if remainingCount != 2 {
		t.Errorf("删除后期望剩余2条记录，实际剩余%d条", remainingCount)
	}

	// 验证最早的记录（48小时前）已被删除
	var oldestRecord Binance24hStatsHistory
	db.Where("symbol = 'DELETETEST'").Order("window_start ASC").First(&oldestRecord)
	if oldestRecord.WindowStart.Before(cutoffTime) {
		t.Error("最早的记录应该已被删除")
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'DELETETEST'")
}

// TestGet24hStatsHistoryStats 测试获取历史表统计信息
func TestGet24hStatsHistoryStats(t *testing.T) {
	db := createTestDB(t)
	if db == nil {
		return
	}

	// 清理并准备测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol IN ('STATS1', 'STATS2', 'STATS3')")

	now := time.Now()
	baseTime := now.Add(-24 * time.Hour)
	testStats := []Binance24hStatsHistory{
		{
			Symbol:         "STATS1",
			MarketType:     "spot",
			WindowStart:    baseTime,
			WindowEnd:      baseTime.Add(time.Hour),
			WindowDuration: 3600,
			LastPrice:      50000.00,
			CreatedAt:      baseTime,
		},
		{
			Symbol:         "STATS2",
			MarketType:     "spot",
			WindowStart:    baseTime.Add(time.Hour),
			WindowEnd:      baseTime.Add(2 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      51000.00,
			CreatedAt:      baseTime.Add(time.Hour),
		},
		{
			Symbol:         "STATS3",
			MarketType:     "futures",
			WindowStart:    baseTime.Add(2 * time.Hour),
			WindowEnd:      baseTime.Add(3 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      52000.00,
			CreatedAt:      baseTime.Add(2 * time.Hour),
		},
	}

	err := Save24hStatsHistory(db, testStats)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 测试获取统计信息
	stats, err := Get24hStatsHistoryStats(db)
	if err != nil {
		t.Fatalf("获取历史表统计信息失败: %v", err)
	}

	// 验证统计结果
	if stats["total_records"].(int64) != 3 {
		t.Errorf("总记录数不正确: 期望3, 实际%d", stats["total_records"])
	}

	if stats["unique_symbols"].(int64) != 3 {
		t.Errorf("唯一交易对数不正确: 期望3, 实际%d", stats["unique_symbols"])
	}

	if stats["unique_markets"].(int64) != 2 {
		t.Errorf("唯一市场数不正确: 期望2, 实际%d", stats["unique_markets"])
	}

	// 验证时间范围计算
	dataPeriodDays := stats["data_period_days"].(float64)
	if dataPeriodDays < 0.04 { // 至少1小时 = 0.0417天
		t.Errorf("数据周期天数不正确: %.4f天", dataPeriodDays)
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol IN ('STATS1', 'STATS2', 'STATS3')")
}

// TestDeduplicate24hStatsHistory 测试去重功能
func TestDeduplicate24hStatsHistory(t *testing.T) {
	now := time.Now()
	windowStart := now.Truncate(time.Hour)

	// 创建包含重复数据的测试切片
	stats := []Binance24hStatsHistory{
		{
			Symbol:      "BTCUSDT",
			MarketType:  "spot",
			WindowStart: windowStart,
			LastPrice:   50000.00,
		},
		{
			Symbol:      "BTCUSDT",
			MarketType:  "spot",
			WindowStart: windowStart,
			LastPrice:   51000.00, // 相同时间窗口，不同数据
		},
		{
			Symbol:      "ETHUSDT",
			MarketType:  "spot",
			WindowStart: windowStart,
			LastPrice:   3000.00,
		},
		{
			Symbol:      "BTCUSDT",
			MarketType:  "spot",
			WindowStart: windowStart.Add(time.Hour), // 不同时间窗口
			LastPrice:   52000.00,
		},
	}

	// 测试去重
	deduplicated := deduplicate24hStatsHistory(stats)

	// 验证结果：应该只保留3条记录（相同时间窗口的BTCUSDT记录被去重）
	if len(deduplicated) != 3 {
		t.Errorf("去重后期望剩余3条记录，实际剩余%d条", len(deduplicated))
	}

	// 验证去重逻辑：相同symbol+market_type+window_start的记录只保留第一条
	found := make(map[string]int)
	for _, stat := range deduplicated {
		key := stat.Symbol + ":" + stat.MarketType + ":" + stat.WindowStart.String()
		found[key]++
	}

	for key, count := range found {
		if count > 1 {
			t.Errorf("去重失败：键%s出现了%d次", key, count)
		}
	}
}
