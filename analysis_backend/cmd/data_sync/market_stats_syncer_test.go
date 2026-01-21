package main

import (
	"testing"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// createTestDBForSync 创建用于同步测试的数据库连接
func createTestDBForSync(t *testing.T) *gorm.DB {
	// 测试数据库配置 - 请根据实际情况修改
	dsn := "root:@tcp(localhost:3306)/analysis_test?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("跳过测试：无法连接测试数据库: %v", err)
		return nil
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&pdb.Binance24hStats{}, &pdb.Binance24hStatsHistory{})
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	return db
}

// TestDualTableSaveFunctionality 测试双表保存功能的独立测试
func TestDualTableSaveFunctionality(t *testing.T) {
	db := createTestDBForSync(t)
	if db == nil {
		return
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats WHERE symbol = 'SYNCTEST'")
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'SYNCTEST'")

	// 创建测试配置
	cfg := &config.Config{}
	config := &DataSyncConfig{
		MaxRetries: 3,
		RetryDelay: 1,
	}

	// 创建同步器
	syncer := NewMarketStatsSyncer(db, cfg, config, nil)
	if syncer == nil {
		t.Fatal("创建MarketStatsSyncer失败")
	}

	// 准备测试数据
	testStats := pdb.Binance24hStats{
		Symbol:             "SYNCTEST",
		MarketType:         "spot",
		PriceChange:        150.75,
		PriceChangePercent: 3.25,
		WeightedAvgPrice:   47500.50,
		PrevClosePrice:     46250.00,
		LastPrice:          47750.75,
		LastQty:            0.5,
		BidPrice:           47750.00,
		BidQty:             1.25,
		AskPrice:           47751.00,
		AskQty:             2.75,
		OpenPrice:          46250.00,
		HighPrice:          48500.00,
		LowPrice:           46000.00,
		Volume:             1250.75,
		QuoteVolume:        59700000.00,
		OpenTime:           time.Now().Add(-time.Hour).Unix() * 1000,
		CloseTime:          time.Now().Unix() * 1000,
		FirstId:            123456789,
		LastId:             123456799,
		Count:              10,
	}

	// 计算时间窗口（直接调用方法）
	windowStart, windowEnd := syncer.calculateCurrentTimeWindow()

	// 创建历史统计数据
	historyStats := pdb.Binance24hStatsHistory{
		Symbol:         testStats.Symbol,
		MarketType:     testStats.MarketType,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		WindowDuration: 3600,
		PriceChange:    testStats.PriceChange,
		LastPrice:      testStats.LastPrice,
		Volume:         testStats.Volume,
		CreatedAt:      time.Now(),
	}

	// 执行双表保存
	err := syncer.saveStatsDualTable(testStats, historyStats)
	if err != nil {
		t.Fatalf("双表保存测试失败: %v", err)
	}

	// 验证实时表数据
	var realtimeRecord pdb.Binance24hStats
	err = db.Where("symbol = ? AND market_type = ?", "SYNCTEST", "spot").First(&realtimeRecord).Error
	if err != nil {
		t.Fatalf("查询实时表数据失败: %v", err)
	}

	if realtimeRecord.LastPrice != testStats.LastPrice {
		t.Errorf("实时表最新价格不匹配: 期望%.2f, 实际%.2f", testStats.LastPrice, realtimeRecord.LastPrice)
	}

	// 验证历史表数据
	var historyRecords []pdb.Binance24hStatsHistory
	err = db.Where("symbol = ? AND market_type = ?", "SYNCTEST", "spot").Find(&historyRecords).Error
	if err != nil {
		t.Fatalf("查询历史表数据失败: %v", err)
	}

	if len(historyRecords) != 1 {
		t.Errorf("期望历史表有1条记录，实际有%d条", len(historyRecords))
	}

	historyRecord := historyRecords[0]
	if historyRecord.LastPrice != testStats.LastPrice {
		t.Errorf("历史表最新价格不匹配: 期望%.2f, 实际%.2f", testStats.LastPrice, historyRecord.LastPrice)
	}

	if historyRecord.WindowDuration != 3600 {
		t.Errorf("历史表时间窗口持续时间不正确: 期望3600, 实际%d", historyRecord.WindowDuration)
	}

	// 测试时间窗口唯一性 - 再次保存相同时间窗口的数据
	err = syncer.saveStatsDualTable(testStats, historyStats)
	if err != nil {
		t.Fatalf("重复保存测试失败: %v", err)
	}

	// 验证历史表记录数没有增加（唯一性约束生效）
	var finalHistoryCount int64
	db.Model(&pdb.Binance24hStatsHistory{}).Where("symbol = ?", "SYNCTEST").Count(&finalHistoryCount)
	if finalHistoryCount != 1 {
		t.Errorf("唯一性约束失败：期望仍然是1条记录，实际变为%d条", finalHistoryCount)
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats WHERE symbol = 'SYNCTEST'")
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'SYNCTEST'")
}

// TestTimeWindowCalculation 测试时间窗口计算功能
func TestTimeWindowCalculation(t *testing.T) {
	db := createTestDBForSync(t)
	if db == nil {
		return
	}

	// 创建同步器
	cfg := &config.Config{}
	config := &DataSyncConfig{}
	syncer := NewMarketStatsSyncer(db, cfg, config, nil)

	// 测试时间窗口计算
	windowStart, windowEnd := syncer.calculateCurrentTimeWindow()

	// 验证时间窗口对齐到小时
	if windowStart.Minute() != 0 || windowStart.Second() != 0 || windowStart.Nanosecond() != 0 {
		t.Error("时间窗口开始时间未正确对齐到小时")
	}

	// 验证时间窗口持续时间为1小时
	duration := windowEnd.Sub(windowStart)
	if duration != time.Hour {
		t.Errorf("时间窗口持续时间不正确: 期望1小时, 实际%s", duration)
	}

	// 验证时间窗口在当前时间范围内
	now := time.Now()
	if windowStart.After(now) || windowEnd.Before(now) {
		t.Errorf("时间窗口不在当前时间范围内: %v - %v", windowStart, windowEnd)
	}
}

// TestDataConsistencyCheck 测试数据一致性检查
func TestDataConsistencyCheck(t *testing.T) {
	db := createTestDBForSync(t)
	if db == nil {
		return
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats WHERE symbol = 'CONSISTENCYTEST'")
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'CONSISTENCYTEST'")

	// 准备测试数据 - 模拟数据不一致的情况
	now := time.Now()
	realtimeStats := pdb.Binance24hStats{
		Symbol:     "CONSISTENCYTEST",
		MarketType: "spot",
		LastPrice:  50000.00,
		CreatedAt:  now,
	}

	historyStats := pdb.Binance24hStatsHistory{
		Symbol:      "CONSISTENCYTEST",
		MarketType:  "spot",
		WindowStart: now.Truncate(time.Hour),
		WindowEnd:   now.Truncate(time.Hour).Add(time.Hour),
		LastPrice:   50000.00,
		CreatedAt:   now,
	}

	// 分别保存到两张表（模拟数据同步过程）
	err := pdb.Save24hStats(db, []pdb.Binance24hStats{realtimeStats})
	if err != nil {
		t.Fatalf("保存实时表数据失败: %v", err)
	}

	err = pdb.Save24hStatsHistory(db, []pdb.Binance24hStatsHistory{historyStats})
	if err != nil {
		t.Fatalf("保存历史表数据失败: %v", err)
	}

	// 验证数据一致性 - 检查两表是否都有数据
	var realtimeCount, historyCount int64
	db.Model(&pdb.Binance24hStats{}).Where("symbol = 'CONSISTENCYTEST'").Count(&realtimeCount)
	db.Model(&pdb.Binance24hStatsHistory{}).Where("symbol = 'CONSISTENCYTEST'").Count(&historyCount)

	if realtimeCount != 1 {
		t.Errorf("实时表数据不一致: 期望1条记录，实际%d条", realtimeCount)
	}

	if historyCount != 1 {
		t.Errorf("历史表数据不一致: 期望1条记录，实际%d条", historyCount)
	}

	// 验证数据内容一致性
	var realtimeData pdb.Binance24hStats
	var historyData pdb.Binance24hStatsHistory

	db.Where("symbol = 'CONSISTENCYTEST'").First(&realtimeData)
	db.Where("symbol = 'CONSISTENCYTEST'").First(&historyData)

	if realtimeData.LastPrice != historyData.LastPrice {
		t.Errorf("数据内容不一致: 实时表价格%.2f, 历史表价格%.2f",
			realtimeData.LastPrice, historyData.LastPrice)
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats WHERE symbol = 'CONSISTENCYTEST'")
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'CONSISTENCYTEST'")
}

// TestConcurrentDualTableOperations 测试并发双表操作
func TestConcurrentDualTableOperations(t *testing.T) {
	db := createTestDBForSync(t)
	if db == nil {
		return
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats WHERE symbol LIKE 'CONCURRENT%'")
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol LIKE 'CONCURRENT%'")

	// 创建测试配置
	cfg := &config.Config{}
	config := &DataSyncConfig{}

	// 创建多个同步器实例
	syncers := make([]*MarketStatsSyncer, 3)
	for i := range syncers {
		syncers[i] = NewMarketStatsSyncer(db, cfg, config, nil)
	}

	// 准备并发测试数据
	testData := []struct {
		symbol string
		price  float64
	}{
		{"CONCURRENT1", 50000.00},
		{"CONCURRENT2", 51000.00},
		{"CONCURRENT3", 52000.00},
	}

	// 并发执行双表保存
	done := make(chan bool, len(testData))
	errors := make(chan error, len(testData))

	for i, data := range testData {
		go func(index int, symbol string, price float64) {
			defer func() { done <- true }()

			syncer := syncers[index%len(syncers)]

			// 准备测试数据
			stats := pdb.Binance24hStats{
				Symbol:     symbol,
				MarketType: "spot",
				LastPrice:  price,
				Volume:     1000.00,
			}

			// 执行双表保存
			historyStats := syncer.createHistoryStatsFromRealtime(stats)
			err := syncer.saveStatsDualTable(stats, historyStats)
			if err != nil {
				errors <- err
				return
			}
		}(i, data.symbol, data.price)
	}

	// 等待所有goroutine完成
	for i := 0; i < len(testData); i++ {
		<-done
	}

	// 检查是否有错误
	close(errors)
	var hasError bool
	for err := range errors {
		t.Errorf("并发测试失败: %v", err)
		hasError = true
	}

	if hasError {
		return
	}

	// 验证并发结果
	var realtimeCount, historyCount int64
	db.Model(&pdb.Binance24hStats{}).Where("symbol LIKE 'CONCURRENT%'").Count(&realtimeCount)
	db.Model(&pdb.Binance24hStatsHistory{}).Where("symbol LIKE 'CONCURRENT%'").Count(&historyCount)

	if realtimeCount != 3 {
		t.Errorf("并发测试实时表记录数不正确: 期望3条，实际%d条", realtimeCount)
	}

	if historyCount != 3 {
		t.Errorf("并发测试历史表记录数不正确: 期望3条，实际%d条", historyCount)
	}

	// 清理测试数据
	db.Exec("DELETE FROM binance_24h_stats WHERE symbol LIKE 'CONCURRENT%'")
	db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol LIKE 'CONCURRENT%'")
}
