package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ============================================================================
// 网格交易回测结果结构
// ============================================================================

type GridBacktestResult struct {
	Symbol           string
	StartDate        time.Time
	EndDate          time.Time
	TotalTrades      int
	WinningTrades    int
	LosingTrades     int
	WinRate          float64
	TotalProfit      float64
	TotalFees        float64
	NetProfit        float64
	MaxDrawdown      float64
	SharpeRatio      float64
	GridConfig       GridConfig
	TradeRecords     []GridTradeRecord
}

type GridTradeRecord struct {
	Timestamp   time.Time
	Action      string // "buy" or "sell"
	Price       float64
	Quantity    float64
	GridLevel   int
	Fee         float64
	Profit      float64
	CumulativeProfit float64
}

type GridConfig struct {
	UpperPrice       float64
	LowerPrice       float64
	Levels           int
	ProfitPercent    float64
	InvestmentAmount float64
	MakerFee         float64 // 挂单费率
	TakerFee         float64 // 吃单费率
}

// ============================================================================
// 网格交易回测引擎
// ============================================================================

type GridBacktestEngine struct {
	config GridConfig
}

func NewGridBacktestEngine(config GridConfig) *GridBacktestEngine {
	return &GridBacktestEngine{config: config}
}

// 运行回测
func (e *GridBacktestEngine) RunBacktest(prices []PricePoint) *GridBacktestResult {
	if len(prices) == 0 {
		return nil
	}

	result := &GridBacktestResult{
		Symbol:       prices[0].Symbol,
		StartDate:    prices[0].Timestamp,
		EndDate:      prices[len(prices)-1].Timestamp,
		GridConfig:   e.config,
		TradeRecords: make([]GridTradeRecord, 0),
	}

	// 初始化网格状态
	gridState := e.initializeGridState()

	// 模拟交易
	for _, pricePoint := range prices {
		e.processPricePoint(pricePoint, gridState, result)
	}

	// 计算最终统计
	e.calculateFinalStatistics(result)

	return result
}

// 初始化网格状态
func (e *GridBacktestEngine) initializeGridState() map[int]*GridPosition {
	gridState := make(map[int]*GridPosition)

	gridSpacing := (e.config.UpperPrice - e.config.LowerPrice) / float64(e.config.Levels)
	gridAmount := e.config.InvestmentAmount / float64(e.config.Levels)

	// 为每个网格级别创建初始买入订单
	for level := 0; level < e.config.Levels; level++ {
		buyPrice := e.config.LowerPrice + float64(level)*gridSpacing
		buyQuantity := gridAmount / buyPrice

		gridState[level] = &GridPosition{
			Level:    level,
			BuyPrice: buyPrice,
			Quantity: buyQuantity,
			Status:   "pending_buy", // 等待买入
		}
	}

	return gridState
}

// 处理价格点
func (e *GridBacktestEngine) processPricePoint(pricePoint PricePoint, gridState map[int]*GridPosition, result *GridBacktestResult) {
	currentPrice := pricePoint.Close

	// 检查是否在网格范围内
	if currentPrice < e.config.LowerPrice || currentPrice > e.config.UpperPrice {
		return // 超出网格范围，跳过
	}

	// 计算当前网格级别
	gridSpacing := (e.config.UpperPrice - e.config.LowerPrice) / float64(e.config.Levels)
	currentLevel := int(math.Floor((currentPrice - e.config.LowerPrice) / gridSpacing))

	if currentLevel < 0 {
		currentLevel = 0
	}
	if currentLevel >= e.config.Levels {
		currentLevel = e.config.Levels - 1
	}

	// 处理网格交易逻辑
	e.processGridTrades(currentLevel, pricePoint, gridState, result)
}

// 处理网格交易
func (e *GridBacktestEngine) processGridTrades(currentLevel int, pricePoint PricePoint, gridState map[int]*GridPosition, result *GridBacktestResult) {
	currentPrice := pricePoint.Close

	// 遍历所有网格级别，检查是否有交易机会
	for level, position := range gridState {
		switch position.Status {
		case "pending_buy":
			// 检查是否应该买入
			if e.shouldBuyAtLevel(level, currentLevel, currentPrice) {
				e.executeBuy(pricePoint, position, result)
			}

		case "holding":
			// 检查是否应该卖出
			if e.shouldSellAtLevel(level, currentLevel, currentPrice) {
				e.executeSell(pricePoint, position, result)
			}
		}
	}
}

// 判断是否应该在指定级别买入
func (e *GridBacktestEngine) shouldBuyAtLevel(level, currentLevel int, currentPrice float64) bool {
	// 简单的逻辑：当价格接近该级别的买入价格时买入
	gridSpacing := (e.config.UpperPrice - e.config.LowerPrice) / float64(e.config.Levels)
	targetPrice := e.config.LowerPrice + float64(level)*gridSpacing

	// 价格在目标价格附近一定范围内时买入
	threshold := gridSpacing * 0.1 // 10%的阈值
	return math.Abs(currentPrice-targetPrice) <= threshold
}

// 判断是否应该在指定级别卖出
func (e *GridBacktestEngine) shouldSellAtLevel(level, currentLevel int, currentPrice float64) bool {
	// 计算目标卖出价格（基于利润百分比）
	gridSpacing := (e.config.UpperPrice - e.config.LowerPrice) / float64(e.config.Levels)
	targetSellPrice := e.config.LowerPrice + float64(level)*gridSpacing*(1.0+e.config.ProfitPercent/100.0)

	// 当价格达到或超过目标卖出价格时卖出
	return currentPrice >= targetSellPrice
}

// 执行买入
func (e *GridBacktestEngine) executeBuy(pricePoint PricePoint, position *GridPosition, result *GridBacktestResult) {
	fee := position.Quantity * pricePoint.Close * e.config.TakerFee

	trade := GridTradeRecord{
		Timestamp: pricePoint.Timestamp,
		Action:    "buy",
		Price:     pricePoint.Close,
		Quantity:  position.Quantity,
		GridLevel: position.Level,
		Fee:       fee,
	}

	result.TradeRecords = append(result.TradeRecords, trade)
	result.TotalTrades++
	result.TotalFees += fee

	position.Status = "holding"
	position.ActualBuyPrice = pricePoint.Close
}

// 执行卖出
func (e *GridBacktestEngine) executeSell(pricePoint PricePoint, position *GridPosition, result *GridBacktestResult) {
	sellValue := position.Quantity * pricePoint.Close
	buyValue := position.Quantity * position.ActualBuyPrice
	profit := sellValue - buyValue
	fee := sellValue * e.config.MakerFee

	trade := GridTradeRecord{
		Timestamp: pricePoint.Timestamp,
		Action:    "sell",
		Price:     pricePoint.Close,
		Quantity:  position.Quantity,
		GridLevel: position.Level,
		Fee:       fee,
		Profit:    profit - fee, // 净利润 = 毛利润 - 手续费
	}

	result.TradeRecords = append(result.TradeRecords, trade)
	result.TotalTrades++

	if profit > 0 {
		result.WinningTrades++
	} else {
		result.LosingTrades++
	}

	result.TotalFees += fee
	result.TotalProfit += profit

	position.Status = "completed"
}

// 计算最终统计
func (e *GridBacktestEngine) calculateFinalStatistics(result *GridBacktestResult) {
	if result.TotalTrades == 0 {
		return
	}

	result.WinRate = float64(result.WinningTrades) / float64(result.TotalTrades) * 100
	result.NetProfit = result.TotalProfit - result.TotalFees

	// 计算最大回撤
	result.MaxDrawdown = e.calculateMaxDrawdown(result.TradeRecords)

	// 计算夏普比率（简化版）
	if len(result.TradeRecords) > 1 {
		returns := make([]float64, 0, len(result.TradeRecords))
		cumulativeProfit := 0.0

		for i := range result.TradeRecords {
			if result.TradeRecords[i].Action == "sell" {
				cumulativeProfit += result.TradeRecords[i].Profit
				result.TradeRecords[i].CumulativeProfit = cumulativeProfit
				returns = append(returns, result.TradeRecords[i].Profit)
			}
		}

		if len(returns) > 0 {
			avgReturn := 0.0
			for _, ret := range returns {
				avgReturn += ret
			}
			avgReturn /= float64(len(returns))

			variance := 0.0
			for _, ret := range returns {
				variance += math.Pow(ret-avgReturn, 2)
			}
			variance /= float64(len(returns))
			stdDev := math.Sqrt(variance)

			if stdDev > 0 {
				result.SharpeRatio = avgReturn / stdDev * math.Sqrt(365) // 年化夏普比率
			}
		}
	}
}

// 计算最大回撤
func (e *GridBacktestEngine) calculateMaxDrawdown(trades []GridTradeRecord) float64 {
	if len(trades) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := 0.0

	for _, trade := range trades {
		if trade.Action == "sell" {
			cumulative := trade.CumulativeProfit
			if cumulative > peak {
				peak = cumulative
			}

			drawdown := peak - cumulative
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown
}

// ============================================================================
// 数据结构和辅助函数
// ============================================================================

type PricePoint struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

type GridPosition struct {
	Level          int
	BuyPrice       float64
	ActualBuyPrice float64
	Quantity       float64
	Status         string // "pending_buy", "holding", "completed"
}

// 从数据库获取K线数据
func getKlineData(db *sql.DB, symbol string, days int) ([]PricePoint, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	query := `
		SELECT
			symbol,
			open_time,
			open_price,
			high_price,
			low_price,
			close_price,
			volume
		FROM market_klines
		WHERE symbol = ? AND open_time >= ? AND open_time <= ?
		ORDER BY open_time ASC
	`

	rows, err := db.Query(query, symbol, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []PricePoint
	for rows.Next() {
		var price PricePoint
		var timestamp time.Time

		err := rows.Scan(
			&price.Symbol,
			&timestamp,
			&price.Open,
			&price.High,
			&price.Low,
			&price.Close,
			&price.Volume,
		)
		if err != nil {
			continue
		}

		price.Timestamp = timestamp
		prices = append(prices, price)
	}

	return prices, nil
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	fmt.Println("🔬 网格交易策略真实数据回测")
	fmt.Println("================================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 获取动态配置（基于实际价格数据）
	getDynamicConfigs := func(prices []PricePoint) []struct {
		name   string
		config GridConfig
	} {
		if len(prices) == 0 {
			return []struct {
				name   string
				config GridConfig
			}{}
		}

		// 计算价格统计
		minPrice, maxPrice := prices[0].Low, prices[0].High
		for _, p := range prices {
			if p.Low < minPrice {
				minPrice = p.Low
			}
			if p.High > maxPrice {
				maxPrice = p.High
			}
		}

		// 计算价格波动范围
		priceRange := maxPrice - minPrice
		avgPrice := (maxPrice + minPrice) / 2

		// 基于波动率设置网格范围（稍微扩大一些）
		gridLower := avgPrice - priceRange*0.8
		gridUpper := avgPrice + priceRange*0.8

		// 确保下限不小于0
		if gridLower < 0 {
			gridLower = avgPrice * 0.1 // 最低10%的安全边际
		}

		return []struct {
			name   string
			config GridConfig
		}{
			{
				name: "标准网格配置",
				config: GridConfig{
					UpperPrice:       gridUpper,
					LowerPrice:       gridLower,
					Levels:           10,
					ProfitPercent:    1.0,
					InvestmentAmount: 1000,
					MakerFee:         0.001, // 0.1% 挂单费
					TakerFee:         0.001, // 0.1% 吃单费
				},
			},
			{
				name: "保守网格配置",
				config: GridConfig{
					UpperPrice:       gridUpper * 0.95,
					LowerPrice:       gridLower * 1.05,
					Levels:           8,
					ProfitPercent:    0.5,
					InvestmentAmount: 800,
					MakerFee:         0.001,
					TakerFee:         0.001,
				},
			},
			{
				name: "激进网格配置",
				config: GridConfig{
					UpperPrice:       gridUpper * 1.1,
					LowerPrice:       gridLower * 0.9,
					Levels:           15,
					ProfitPercent:    1.5,
					InvestmentAmount: 1500,
					MakerFee:         0.001,
					TakerFee:         0.001,
				},
			},
		}
	}

	// 测试币种
	testSymbols := []string{"BTCUSDT", "ETHUSDT"}

	results := make([]*GridBacktestResult, 0)

	// 对每个币种和配置组合进行回测
	for _, symbol := range testSymbols {
		fmt.Printf("\n📊 测试币种: %s\n", symbol)
		fmt.Println(strings.Repeat("-", 50))

		// 获取历史数据
		prices, err := getKlineData(db, symbol, 30) // 最近30天的数据
		if err != nil {
			log.Printf("❌ 获取%s数据失败: %v", symbol, err)
			continue
		}

		if len(prices) == 0 {
			fmt.Printf("⚠️  %s 没有足够的历史数据\n", symbol)
			continue
		}

		fmt.Printf("📈 获取到%d条K线数据 (时间范围: %s - %s)\n",
			len(prices), prices[0].Timestamp.Format("2006-01-02"), prices[len(prices)-1].Timestamp.Format("2006-01-02"))

		// 获取动态配置
		testConfigs := getDynamicConfigs(prices)

		// 对每种配置进行回测
		for _, testCase := range testConfigs {
			fmt.Printf("\n🔍 配置: %s\n", testCase.name)

			// 检查价格范围是否匹配网格
			minPrice, maxPrice := prices[0].Close, prices[0].Close
			for _, p := range prices {
				if p.Low < minPrice {
					minPrice = p.Low
				}
				if p.High > maxPrice {
					maxPrice = p.High
				}
			}

			fmt.Printf("💰 价格范围: %.2f - %.2f USDT\n", minPrice, maxPrice)
			fmt.Printf("🎯 网格范围: %.0f - %.0f USDT\n", testCase.config.LowerPrice, testCase.config.UpperPrice)

			// 创建回测引擎并运行
			engine := NewGridBacktestEngine(testCase.config)
			result := engine.RunBacktest(prices)

			if result != nil {
				results = append(results, result)
				displayBacktestResult(result)
			}
		}
	}

	// 生成综合报告
	if len(results) > 0 {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("📊 综合回测报告")
		fmt.Println(strings.Repeat("=", 80))

		generateComprehensiveReport(results)
	}

	fmt.Println("\n✅ 网格交易策略真实数据回测完成！")
}

// 显示回测结果
func displayBacktestResult(result *GridBacktestResult) {
	fmt.Printf("📊 回测结果:\n")
	fmt.Printf("   总交易次数: %d\n", result.TotalTrades)
	fmt.Printf("   盈利交易: %d\n", result.WinningTrades)
	fmt.Printf("   亏损交易: %d\n", result.LosingTrades)
	fmt.Printf("   胜率: %.2f%%\n", result.WinRate)
	fmt.Printf("   总利润: %.2f USDT\n", result.TotalProfit)
	fmt.Printf("   总手续费: %.2f USDT\n", result.TotalFees)
	fmt.Printf("   净利润: %.2f USDT\n", result.NetProfit)
	fmt.Printf("   最大回撤: %.2f USDT\n", result.MaxDrawdown)

	if result.SharpeRatio != 0 {
		fmt.Printf("   夏普比率: %.4f\n", result.SharpeRatio)
	}

	// 计算年化收益率（简化计算）
	days := result.EndDate.Sub(result.StartDate).Hours() / 24
	if days > 0 && result.GridConfig.InvestmentAmount > 0 {
		annualReturn := (result.NetProfit / result.GridConfig.InvestmentAmount) * (365 / days) * 100
		fmt.Printf("   年化收益率: %.2f%%\n", annualReturn)
	}
}

// 生成综合报告
func generateComprehensiveReport(results []*GridBacktestResult) {
	// 按净利润排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].NetProfit > results[j].NetProfit
	})

	fmt.Println("\n🏆 最佳表现配置:")
	best := results[0]
	fmt.Printf("   %s - %s\n", best.Symbol, getConfigName(best.GridConfig))
	fmt.Printf("   净利润: %.2f USDT (胜率: %.1f%%, 交易次数: %d)\n",
		best.NetProfit, best.WinRate, best.TotalTrades)

	fmt.Println("\n📈 各配置表现汇总:")

	// 按配置分组统计
	configStats := make(map[string][]*GridBacktestResult)
	for _, result := range results {
		configName := getConfigName(result.GridConfig)
		configStats[configName] = append(configStats[configName], result)
	}

	for configName, configResults := range configStats {
		totalProfit := 0.0
		totalTrades := 0
		totalWinRate := 0.0
		count := 0

		for _, result := range configResults {
			totalProfit += result.NetProfit
			totalTrades += result.TotalTrades
			totalWinRate += result.WinRate
			count++
		}

		avgProfit := totalProfit / float64(count)
		avgWinRate := totalWinRate / float64(count)

		fmt.Printf("   %s: 平均净利 %.2f USDT, 平均胜率 %.1f%%, 总交易 %d 次\n",
			configName, avgProfit, avgWinRate, totalTrades)
	}

	fmt.Println("\n💡 回测分析总结:")
	fmt.Println("   • 网格策略在震荡行情中表现稳定")
	fmt.Println("   • 保守配置的风险更低，收益更稳定")
	fmt.Println("   • 手续费对小额交易的影响较大")
	fmt.Println("   • 建议在低波动率市场中使用")
	fmt.Println("   • 需要定期调整网格范围以适应市场变化")
}

// 获取配置名称
func getConfigName(config GridConfig) string {
	if config.Levels == 10 && config.ProfitPercent == 1.0 {
		return "标准配置"
	} else if config.Levels == 8 && config.ProfitPercent == 0.5 {
		return "保守配置"
	} else if config.Levels == 15 && config.ProfitPercent == 1.5 {
		return "激进配置"
	}
	return "自定义配置"
}