package db

import (
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// createTestQueryService 创建测试用的查询服务
func createTestQueryService(t *testing.T) *StatsQueryService {
	db, err := gorm.Open(mysql.Open("root:@tcp(localhost:3306)/analysis_test?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
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

	return NewStatsQueryService(db)
}

// TestStatsQueryService_QueryLatest 测试最新数据查询
func TestStatsQueryService_QueryLatest(t *testing.T) {
	service := createTestQueryService(t)
	if service == nil {
		return
	}

	// 清理并准备测试数据
	service.db.Exec("DELETE FROM binance_24h_stats WHERE symbol = 'LATESTTEST'")

	testStats := Binance24hStats{
		Symbol:             "LATESTTEST",
		MarketType:         "spot",
		PriceChange:        100.50,
		PriceChangePercent: 2.5,
		LastPrice:          50000.00,
		Volume:             1500.75,
		CreatedAt:          time.Now(),
	}

	err := Save24hStats(service.db, []Binance24hStats{testStats})
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 测试最新数据查询
	query := &StatsQuery{
		Symbol:     "LATESTTEST",
		MarketType: "spot",
		QueryType:  QueryTypeLatest,
	}

	results, err := service.Query(query)
	if err != nil {
		t.Fatalf("查询最新数据失败: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("期望返回1条记录，实际返回%d条", len(results))
	}

	if results[0].Symbol != "LATESTTEST" {
		t.Errorf("查询结果symbol不匹配: 期望LATESTTEST, 实际%s", results[0].Symbol)
	}

	if results[0].DataSource != "realtime" {
		t.Errorf("数据源标识不正确: 期望realtime, 实际%s", results[0].DataSource)
	}

	// 清理测试数据
	service.db.Exec("DELETE FROM binance_24h_stats WHERE symbol = 'LATESTTEST'")
}

// TestStatsQueryService_QueryTimeRange 测试时间范围查询
func TestStatsQueryService_QueryTimeRange(t *testing.T) {
	service := createTestQueryService(t)
	if service == nil {
		return
	}

	// 清理并准备测试数据
	service.db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'TIMERANGETEST'")

	now := time.Now()
	windowStart := now.Truncate(time.Hour)
	testStats := []Binance24hStatsHistory{
		{
			Symbol:         "TIMERANGETEST",
			MarketType:     "spot",
			WindowStart:    windowStart,
			WindowEnd:      windowStart.Add(time.Hour),
			WindowDuration: 3600,
			LastPrice:      50000.00,
			Volume:         1000.00,
			CreatedAt:      now,
		},
		{
			Symbol:         "TIMERANGETEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(time.Hour),
			WindowEnd:      windowStart.Add(2 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      51000.00,
			Volume:         1100.00,
			CreatedAt:      now.Add(time.Hour),
		},
	}

	err := Save24hStatsHistory(service.db, testStats)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 测试时间范围查询
	startTime := windowStart
	endTime := windowStart.Add(2 * time.Hour)

	query := &StatsQuery{
		Symbol:     "TIMERANGETEST",
		MarketType: "spot",
		QueryType:  QueryTypeTimeRange,
		StartTime:  &startTime,
		EndTime:    &endTime,
	}

	results, err := service.Query(query)
	if err != nil {
		t.Fatalf("查询时间范围数据失败: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("期望返回2条记录，实际返回%d条", len(results))
	}

	// 验证数据源标识
	for _, result := range results {
		if result.DataSource != "history" {
			t.Errorf("数据源标识不正确: 期望history, 实际%s", result.DataSource)
		}
		if result.WindowStart == nil {
			t.Error("历史数据应该包含WindowStart字段")
		}
	}

	// 清理测试数据
	service.db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'TIMERANGETEST'")
}

// TestStatsQueryService_QueryValidation 测试查询参数验证
func TestStatsQueryService_QueryValidation(t *testing.T) {
	service := createTestQueryService(t)
	if service == nil {
		return
	}

	testCases := []struct {
		name        string
		query       *StatsQuery
		expectError bool
	}{
		{
			name:        "nil query",
			query:       nil,
			expectError: true,
		},
		{
			name: "empty symbol",
			query: &StatsQuery{
				Symbol:     "",
				MarketType: "spot",
				QueryType:  QueryTypeLatest,
			},
			expectError: true,
		},
		{
			name: "empty market type",
			query: &StatsQuery{
				Symbol:     "BTCUSDT",
				MarketType: "",
				QueryType:  QueryTypeLatest,
			},
			expectError: true,
		},
		{
			name: "time range query without time",
			query: &StatsQuery{
				Symbol:     "BTCUSDT",
				MarketType: "spot",
				QueryType:  QueryTypeTimeRange,
			},
			expectError: true,
		},
		{
			name: "invalid time range",
			query: &StatsQuery{
				Symbol:     "BTCUSDT",
				MarketType: "spot",
				QueryType:  QueryTypeTimeRange,
				StartTime:  func() *time.Time { t := time.Now(); return &t }(),
				EndTime:    func() *time.Time { t := time.Now().Add(-time.Hour); return &t }(),
			},
			expectError: true,
		},
		{
			name: "valid latest query",
			query: &StatsQuery{
				Symbol:     "BTCUSDT",
				MarketType: "spot",
				QueryType:  QueryTypeLatest,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Query(tc.query)
			if tc.expectError && err == nil {
				t.Error("期望出现错误，但没有错误")
			}
			if !tc.expectError && err != nil {
				t.Errorf("不期望出现错误，但出现了: %v", err)
			}
		})
	}
}

// TestStatsQueryService_Sorting 测试排序功能
func TestStatsQueryService_Sorting(t *testing.T) {
	service := createTestQueryService(t)
	if service == nil {
		return
	}

	// 清理并准备测试数据
	service.db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'SORTTEST'")

	now := time.Now()
	windowStart := now.Truncate(time.Hour)
	testStats := []Binance24hStatsHistory{
		{
			Symbol:         "SORTTEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(2 * time.Hour),
			WindowEnd:      windowStart.Add(3 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      52000.00,
			CreatedAt:      now.Add(2 * time.Hour),
		},
		{
			Symbol:         "SORTTEST",
			MarketType:     "spot",
			WindowStart:    windowStart,
			WindowEnd:      windowStart.Add(time.Hour),
			WindowDuration: 3600,
			LastPrice:      50000.00,
			CreatedAt:      now,
		},
		{
			Symbol:         "SORTTEST",
			MarketType:     "spot",
			WindowStart:    windowStart.Add(time.Hour),
			WindowEnd:      windowStart.Add(2 * time.Hour),
			WindowDuration: 3600,
			LastPrice:      51000.00,
			CreatedAt:      now.Add(time.Hour),
		},
	}

	err := Save24hStatsHistory(service.db, testStats)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	// 测试升序排序
	query := &StatsQuery{
		Symbol:     "SORTTEST",
		MarketType: "spot",
		QueryType:  QueryTypeTimeRange,
		StartTime:  &windowStart,
		EndTime:    func() *time.Time { t := windowStart.Add(3 * time.Hour); return &t }(),
		SortBy:     "window_start",
		SortOrder:  "asc",
	}

	results, err := service.Query(query)
	if err != nil {
		t.Fatalf("查询排序数据失败: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("期望返回3条记录，实际返回%d条", len(results))
	}

	// 验证升序排序
	for i := 1; i < len(results); i++ {
		if results[i].WindowStart != nil && results[i-1].WindowStart != nil {
			if !results[i-1].WindowStart.Before(*results[i].WindowStart) {
				t.Error("升序排序失败")
			}
		}
	}

	// 清理测试数据
	service.db.Exec("DELETE FROM binance_24h_stats_history WHERE symbol = 'SORTTEST'")
}

// TestStatsQueryService_GetStats 测试获取服务统计信息
func TestStatsQueryService_GetStats(t *testing.T) {
	// 创建一个mock的DB连接（不需要真实连接）
	service := NewStatsQueryService(nil)

	stats := service.GetStats()

	if stats["service_name"] != "StatsQueryService" {
		t.Errorf("服务名称不正确: 期望StatsQueryService, 实际%s", stats["service_name"])
	}

	supportedTypes, ok := stats["supported_query_types"].([]string)
	if !ok {
		t.Error("支持的查询类型格式不正确")
	}

	expectedTypes := []string{
		string(QueryTypeLatest),
		string(QueryTypeTimeRange),
		string(QueryTypeTechnical),
		string(QueryTypeBacktest),
		string(QueryTypeMixed),
	}

	if len(supportedTypes) != len(expectedTypes) {
		t.Errorf("支持的查询类型数量不正确: 期望%d, 实际%d", len(expectedTypes), len(supportedTypes))
	}
}

// TestQueryTypeConstants 测试查询类型常量定义
func TestQueryTypeConstants(t *testing.T) {
	expectedTypes := map[QueryType]string{
		QueryTypeLatest:    "latest",
		QueryTypeTimeRange: "time_range",
		QueryTypeTechnical: "technical",
		QueryTypeBacktest:  "backtest",
		QueryTypeMixed:     "mixed",
	}

	for queryType, expectedValue := range expectedTypes {
		if string(queryType) != expectedValue {
			t.Errorf("查询类型常量不正确: %s 期望%s, 实际%s", queryType, expectedValue, string(queryType))
		}
	}
}
