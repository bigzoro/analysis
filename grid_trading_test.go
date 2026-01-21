package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
)

// ============================================================================
// 网格交易测试套件
// ============================================================================

func TestMain(m *testing.M) {
	// 设置测试环境
	setupTestEnvironment()
	defer teardownTestEnvironment()

	// 运行测试
	code := m.Run()
	os.Exit(code)
}

// ============================================================================
// 测试环境设置
// ============================================================================

var testDB *gorm.DB
var testServer *server.Server
var testConfig *config.Config

func setupTestEnvironment() {
	fmt.Println("🔧 设置网格交易测试环境...")

	// 加载配置
	cfg, err := config.MustLoad("config.yaml", &config.Config{})
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	testConfig = cfg

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	dbConn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	testDB = dbConn

	// 初始化服务器
	srv, err := server.NewServer(testDB, cfg)
	if err != nil {
		log.Fatalf("❌ 初始化服务器失败: %v", err)
	}
	testServer = srv

	fmt.Println("✅ 测试环境设置完成")
}

func teardownTestEnvironment() {
	fmt.Println("🧹 清理测试环境...")

	if testDB != nil {
		sqlDB, _ := testDB.DB()
		sqlDB.Close()
	}

	fmt.Println("✅ 测试环境清理完成")
}

// ============================================================================
// 网格交易策略执行器测试
// ============================================================================

func TestGridTradingStrategyExecutor_GetStrategyType(t *testing.T) {
	executor := &server.GridTradingStrategyExecutor{}

	strategyType := executor.GetStrategyType()
	expected := "grid_trading"

	if strategyType != expected {
		t.Errorf("❌ GetStrategyType() = %v, 期望 %v", strategyType, expected)
	} else {
		t.Logf("✅ GetStrategyType() = %v", strategyType)
	}
}

func TestGridTradingStrategyExecutor_IsEnabled(t *testing.T) {
	executor := &server.GridTradingStrategyExecutor{}

	tests := []struct {
		name       string
		conditions pdb.StrategyConditions
		expected   bool
	}{
		{
			name: "网格交易启用",
			conditions: pdb.StrategyConditions{
				GridTradingEnabled: true,
			},
			expected: true,
		},
		{
			name: "网格交易未启用",
			conditions: pdb.StrategyConditions{
				GridTradingEnabled: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.IsEnabled(tt.conditions)
			if result != tt.expected {
				t.Errorf("❌ IsEnabled() = %v, 期望 %v", result, tt.expected)
			} else {
				t.Logf("✅ %s: IsEnabled() = %v", tt.name, result)
			}
		})
	}
}

func TestGridTradingStrategyExecutor_CalculateTechnicalIndicators(t *testing.T) {
	executor := &server.GridTradingStrategyExecutor{}

	ctx := context.Background()
	symbol := "BTCUSDT"

	// 测试技术指标计算（需要真实的K线数据）
	indicators := executor.calculateTechnicalIndicators(ctx, testServer, symbol)

	if indicators.RSI == 0 {
		t.Logf("⚠️ RSI为0，可能没有足够的K线数据")
	} else {
		t.Logf("✅ 计算技术指标成功 - RSI: %.2f, MACD: %.4f, 趋势: %s",
			indicators.RSI, indicators.MACD, indicators.Trend)
	}
}

func TestGridTradingStrategyExecutor_ExecuteFull(t *testing.T) {
	executor := &server.GridTradingStrategyExecutor{}

	ctx := context.Background()
	symbol := "BTCUSDT"

	// 创建测试用的市场数据
	marketData := server.StrategyMarketData{
		Symbol:      symbol,
		Price:       45000.0,
		Change24h:   2.5,
		Volume24h:   1000000.0,
		MarketCap:   850000000000.0,
		HasSpot:     true,
		HasFutures:  true,
		GainersRank: 1,
	}

	// 创建测试用的策略条件
	conditions := pdb.StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       50000.0,
		GridLowerPrice:       40000.0,
		GridLevels:           10,
		GridProfitPercent:    1.0,
		GridInvestmentAmount: 10000.0,
		GridStopLossEnabled:  true,
		GridStopLossPercent:  10.0,
		DynamicPositioning:   true,
		MaxPositionSize:      50.0,
	}

	// 执行完整策略
	result := executor.ExecuteFull(ctx, testServer, symbol, marketData, conditions)

	t.Logf("🎯 策略执行结果:")
	t.Logf("   动作: %s", result.Action)
	t.Logf("   原因: %s", result.Reason)
	t.Logf("   价格: %.2f", result.Price)
	t.Logf("   数量: %.4f", result.Quantity)
	t.Logf("   乘数: %.2f", result.Multiplier)

	if result.Action == "skip" {
		t.Logf("⚠️ 策略跳过执行: %s", result.Reason)
	} else {
		t.Logf("✅ 策略执行成功")
	}
}

// ============================================================================
// 网格交易扫描器测试
// ============================================================================

func TestGridTradingStrategyScanner_Scan(t *testing.T) {
	scanner := &server.GridTradingStrategyScanner{
		server:            testServer,
		marketDataService: pdb.NewCoinCapMarketDataService(testDB),
	}

	ctx := context.Background()

	// 创建测试策略
	strategy := &pdb.TradingStrategy{
		Conditions: pdb.StrategyConditions{
			GridTradingEnabled:   true,
			GridUpperPrice:       1000.0, // 使用动态范围
			GridLowerPrice:       10.0,
			GridLevels:           10,
			GridProfitPercent:    1.0,
			GridInvestmentAmount: 10000.0,
			SpotContract:         true,
		},
	}

	// 执行扫描
	eligibleSymbols, err := scanner.Scan(ctx, strategy)
	if err != nil {
		t.Errorf("❌ 网格扫描失败: %v", err)
		return
	}

	t.Logf("🔍 网格交易扫描结果:")
	t.Logf("   找到适合币种数量: %d", len(eligibleSymbols))

	for i, symbol := range eligibleSymbols {
		t.Logf("   %d. %s - 当前价格: %.4f, 波动率评分: %.2f",
			i+1, symbol.Symbol, symbol.CurrentPrice, symbol.VolatilityScore)

		if i >= 4 { // 只显示前5个
			t.Logf("   ... 还有 %d 个币种", len(eligibleSymbols)-5)
			break
		}
	}

	if len(eligibleSymbols) == 0 {
		t.Logf("⚠️ 未找到适合网格交易的币种")
	} else {
		t.Logf("✅ 成功找到 %d 个适合网格交易的币种", len(eligibleSymbols))
	}
}

func TestGridTradingStrategyScanner_CheckGridTradingSuitability(t *testing.T) {
	scanner := &server.GridTradingStrategyScanner{
		server:            testServer,
		marketDataService: pdb.NewCoinCapMarketDataService(testDB),
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		symbol     string
		conditions pdb.StrategyConditions
		expectNil  bool
	}{
		{
			name:   "BTC网格适应性测试",
			symbol: "BTCUSDT",
			conditions: pdb.StrategyConditions{
				GridTradingEnabled: true,
				GridLevels:         10,
			},
			expectNil: false, // BTC应该能通过基本检查
		},
		{
			name:   "不存在币种测试",
			symbol: "NONEXISTENTCOIN",
			conditions: pdb.StrategyConditions{
				GridTradingEnabled: true,
				GridLevels:         10,
			},
			expectNil: true, // 不存在的币种应该返回nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marketData := server.StrategyMarketData{
				Symbol:  tt.symbol,
				HasSpot: true,
			}

			result := scanner.checkGridTradingSuitability(ctx, tt.symbol, marketData, tt.conditions)

			if tt.expectNil && result != nil {
				t.Errorf("❌ 期望返回nil，但得到了结果")
			} else if !tt.expectNil && result == nil {
				t.Logf("⚠️ %s: 返回nil，可能该币种不适合网格交易", tt.name)
			} else if result != nil {
				t.Logf("✅ %s: 币种 %s 通过网格适应性检查", tt.name, tt.symbol)
				t.Logf("   当前价格: %.4f, 波动率评分: %.2f, 流动性评分: %.2f",
					result.CurrentPrice, result.VolatilityScore, result.LiquidityScore)
			}
		})
	}
}

// ============================================================================
// 网格订单管理器测试
// ============================================================================

func TestGridOrderManager_CalculateGridLevels(t *testing.T) {
	// 创建网格订单管理器
	gom := &server.GridOrderManager{
		server: testServer,
		conditions: pdb.StrategyConditions{
			GridUpperPrice: 100.0,
			GridLowerPrice: 50.0,
			GridLevels:     5,
		},
	}

	levels := gom.calculateGridLevels()

	expectedLevels := 5
	if len(levels) != expectedLevels {
		t.Errorf("❌ calculateGridLevels() 返回 %d 个层级，期望 %d", len(levels), expectedLevels)
	} else {
		t.Logf("✅ calculateGridLevels() 返回 %d 个网格层级", len(levels))

		for i, level := range levels {
			t.Logf("   层级 %d: 价格 %.2f, 数量 %.4f", i+1, level.price, level.quantity)
		}
	}
}

func TestGridOrderManager_CalculateSmartPositionSize(t *testing.T) {
	gom := &server.GridOrderManager{
		server: testServer,
		conditions: pdb.StrategyConditions{
			GridInvestmentAmount: 1000.0,
			GridLevels:           10,
		},
	}

	basePrice := 100.0
	positionSize := gom.calculateSmartPositionSize(basePrice, 1.0)

	expectedSize := 1000.0 / 10 / basePrice // 每层级投资额 / 价格

	t.Logf("💰 智能仓位计算:")
	t.Logf("   基础价格: %.2f", basePrice)
	t.Logf("   计算仓位: %.6f", positionSize)
	t.Logf("   期望大小: %.6f", expectedSize)

	if math.Abs(positionSize-expectedSize) > 0.000001 {
		t.Errorf("❌ calculateSmartPositionSize() = %.6f, 期望 %.6f", positionSize, expectedSize)
	} else {
		t.Logf("✅ 仓位大小计算正确")
	}
}

// ============================================================================
// 网格风险管理器测试
// ============================================================================

func TestGridRiskManager_GetRiskMetrics(t *testing.T) {
	// 创建网格风险管理器
	grm := &server.GridRiskManager{
		positionHistory: []server.GridPosition{
			{Symbol: "BTCUSDT", TotalQuantity: 1.0, AvgPrice: 45000.0, UnrealizedPnL: 1000.0},
			{Symbol: "ETHUSDT", TotalQuantity: 10.0, AvgPrice: 3000.0, UnrealizedPnL: -500.0},
		},
	}

	metrics := grm.GetRiskMetrics()

	t.Logf("📊 风险指标:")
	for key, value := range metrics {
		t.Logf("   %s: %v", key, value)
	}

	// 检查关键指标是否存在
	requiredMetrics := []string{"total_positions", "total_pnl", "win_rate"}
	for _, metric := range requiredMetrics {
		if _, exists := metrics[metric]; !exists {
			t.Errorf("❌ 缺少必需的风险指标: %s", metric)
		}
	}

	t.Logf("✅ 风险指标计算完成")
}

func TestGridRiskManager_CalculateWinRate(t *testing.T) {
	grm := &server.GridRiskManager{
		positionHistory: []server.GridPosition{
			{UnrealizedPnL: 1000.0}, // 盈利
			{UnrealizedPnL: -500.0}, // 亏损
			{UnrealizedPnL: 2000.0}, // 盈利
			{UnrealizedPnL: -100.0}, // 亏损
		},
	}

	winRate := grm.calculateWinRate()
	expectedWinRate := 0.5 // 2个盈利，2个亏损，胜率50%

	t.Logf("🎯 胜率计算:")
	t.Logf("   计算胜率: %.2f", winRate)
	t.Logf("   期望胜率: %.2f", expectedWinRate)

	if math.Abs(winRate-expectedWinRate) > 0.001 {
		t.Errorf("❌ calculateWinRate() = %.2f, 期望 %.2f", winRate, expectedWinRate)
	} else {
		t.Logf("✅ 胜率计算正确")
	}
}

// ============================================================================
// 集成测试 - 完整网格交易流程
// ============================================================================

func TestGridTradingIntegration(t *testing.T) {
	t.Logf("🚀 开始网格交易集成测试")

	ctx := context.Background()
	symbol := "BTCUSDT"

	// 1. 测试策略执行器
	t.Logf("📈 第1步: 测试网格交易策略执行器")
	executor := &server.GridTradingStrategyExecutor{}

	marketData := server.StrategyMarketData{
		Symbol:     symbol,
		Price:      45000.0,
		HasSpot:    true,
		HasFutures: true,
	}

	conditions := pdb.StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       50000.0,
		GridLowerPrice:       40000.0,
		GridLevels:           10,
		GridProfitPercent:    1.0,
		GridInvestmentAmount: 10000.0,
	}

	result := executor.ExecuteFull(ctx, testServer, symbol, marketData, conditions)
	t.Logf("   策略执行结果: %s - %s", result.Action, result.Reason)

	// 2. 测试扫描器
	t.Logf("🔍 第2步: 测试网格交易扫描器")
	scanner := &server.GridTradingStrategyScanner{
		server:            testServer,
		marketDataService: pdb.NewCoinCapMarketDataService(testDB),
	}

	strategy := &pdb.TradingStrategy{Conditions: conditions}
	eligibleSymbols, err := scanner.Scan(ctx, strategy)
	if err != nil {
		t.Errorf("❌ 扫描器测试失败: %v", err)
	} else {
		t.Logf("   扫描到 %d 个适合币种", len(eligibleSymbols))
	}

	// 3. 测试订单管理器
	t.Logf("📋 第3步: 测试网格订单管理器")
	gom := &server.GridOrderManager{
		server:     testServer,
		conditions: conditions,
		symbol:     symbol,
	}

	gridLevels := gom.calculateGridLevels()
	t.Logf("   计算出 %d 个网格层级", len(gridLevels))

	// 4. 测试风险管理器
	t.Logf("⚠️ 第4步: 测试网格风险管理器")
	grm := &server.GridRiskManager{
		positionHistory: []server.GridPosition{
			{Symbol: symbol, TotalQuantity: 0.1, AvgPrice: 45000.0, UnrealizedPnL: 500.0},
		},
	}

	riskMetrics := grm.GetRiskMetrics()
	t.Logf("   计算出 %d 个风险指标", len(riskMetrics))

	t.Logf("✅ 网格交易集成测试完成")
}

// ============================================================================
// 性能测试
// ============================================================================

func BenchmarkGridTradingStrategyExecutor_ExecuteFull(b *testing.B) {
	executor := &server.GridTradingStrategyExecutor{}

	ctx := context.Background()
	symbol := "BTCUSDT"

	marketData := server.StrategyMarketData{
		Symbol:     symbol,
		Price:      45000.0,
		HasSpot:    true,
		HasFutures: true,
	}

	conditions := pdb.StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       50000.0,
		GridLowerPrice:       40000.0,
		GridLevels:           10,
		GridProfitPercent:    1.0,
		GridInvestmentAmount: 10000.0,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = executor.ExecuteFull(ctx, testServer, symbol, marketData, conditions)
	}
}

func BenchmarkGridTradingStrategyScanner_CheckSuitability(b *testing.B) {
	scanner := &server.GridTradingStrategyScanner{
		server:            testServer,
		marketDataService: pdb.NewCoinCapMarketDataService(testDB),
	}

	ctx := context.Background()
	symbol := "BTCUSDT"

	marketData := server.StrategyMarketData{
		Symbol:  symbol,
		HasSpot: true,
	}

	conditions := pdb.StrategyConditions{
		GridTradingEnabled: true,
		GridLevels:         10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = scanner.checkGridTradingSuitability(ctx, symbol, marketData, conditions)
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// createTestStrategy 创建测试用的策略配置
func createTestStrategy() *pdb.TradingStrategy {
	return &pdb.TradingStrategy{
		Name: "网格交易测试策略",
		Conditions: pdb.StrategyConditions{
			GridTradingEnabled:   true,
			GridUpperPrice:       1000.0,
			GridLowerPrice:       10.0,
			GridLevels:           10,
			GridProfitPercent:    1.0,
			GridInvestmentAmount: 10000.0,
			GridStopLossEnabled:  true,
			GridStopLossPercent:  10.0,
			DynamicPositioning:   true,
			MaxPositionSize:      50.0,
			SpotContract:         true,
		},
	}
}

// logTestResult 记录测试结果
func logTestResult(t *testing.T, testName string, success bool, message string) {
	if success {
		t.Logf("✅ %s: %s", testName, message)
	} else {
		t.Errorf("❌ %s: %s", testName, message)
	}
}
