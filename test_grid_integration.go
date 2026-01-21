package main

import (
	"fmt"
	"math"
)

// 简化的网格策略执行器（用于测试）
type GridTradingStrategyExecutor struct{}

func (e *GridTradingStrategyExecutor) ValidateGridParameters(conditions StrategyConditions) []string {
	var errors []string

	if !conditions.GridTradingEnabled {
		return errors
	}

	if conditions.GridUpperPrice <= 0 {
		errors = append(errors, "网格上限价格必须大于0")
	}

	if conditions.GridLowerPrice <= 0 {
		errors = append(errors, "网格下限价格必须大于0")
	}

	if conditions.GridUpperPrice <= conditions.GridLowerPrice {
		errors = append(errors, "网格上限价格必须大于下限价格")
	}

	if conditions.GridLevels <= 0 {
		errors = append(errors, "网格层数必须大于0")
	}

	if conditions.GridLevels > 100 {
		errors = append(errors, "网格层数不能超过100层")
	}

	if conditions.GridInvestmentAmount <= 0 {
		errors = append(errors, "网格投资金额必须大于0")
	}

	return errors
}

func (e *GridTradingStrategyExecutor) CreateGridOrders(symbol string, upperPrice, lowerPrice float64, levels int, investmentAmount float64) ([]GridOrder, error) {
	if upperPrice <= lowerPrice || levels <= 0 || investmentAmount <= 0 {
		return nil, fmt.Errorf("无效的网格参数")
	}

	gridSpacing := (upperPrice - lowerPrice) / float64(levels)
	gridAmount := investmentAmount / float64(levels)

	var orders []GridOrder
	orderID := uint(1)

	// 创建买入订单
	for i := 0; i < levels; i++ {
		buyPrice := lowerPrice + float64(i)*gridSpacing
		buyQuantity := gridAmount / buyPrice

		order := GridOrder{
			ID:        orderID,
			Symbol:    symbol,
			Side:      "buy",
			Price:     buyPrice,
			Quantity:  buyQuantity,
			GridLevel: i,
			Status:    "pending",
		}
		orders = append(orders, order)
		orderID++
	}

	// 创建卖出订单
	for i := levels; i >= 0; i-- {
		sellPrice := lowerPrice + float64(i)*gridSpacing
		sellQuantity := gridAmount / sellPrice

		order := GridOrder{
			ID:        orderID,
			Symbol:    symbol,
			Side:      "sell",
			Price:     sellPrice,
			Quantity:  sellQuantity,
			GridLevel: i,
			Status:    "pending",
		}
		orders = append(orders, order)
		orderID++
	}

	return orders, nil
}

func (e *GridTradingStrategyExecutor) GetGridMetrics(conditions StrategyConditions, currentPrice float64) map[string]interface{} {
	metrics := make(map[string]interface{})

	if !conditions.GridTradingEnabled {
		metrics["enabled"] = false
		return metrics
	}

	metrics["enabled"] = true
	metrics["upper_price"] = conditions.GridUpperPrice
	metrics["lower_price"] = conditions.GridLowerPrice
	metrics["levels"] = conditions.GridLevels
	metrics["profit_percent"] = conditions.GridProfitPercent
	metrics["investment_amount"] = conditions.GridInvestmentAmount

	if conditions.GridLevels > 0 {
		gridSpacing := (conditions.GridUpperPrice - conditions.GridLowerPrice) / float64(conditions.GridLevels)
		metrics["grid_spacing"] = gridSpacing
		metrics["grid_spacing_percent"] = (gridSpacing / ((conditions.GridUpperPrice + conditions.GridLowerPrice) / 2)) * 100
	}

	metrics["current_price"] = currentPrice
	metrics["in_range"] = currentPrice >= conditions.GridLowerPrice && currentPrice <= conditions.GridUpperPrice

	if conditions.GridLevels > 0 && conditions.GridUpperPrice > conditions.GridLowerPrice {
		currentLevel := int(math.Floor((currentPrice - conditions.GridLowerPrice) / ((conditions.GridUpperPrice - conditions.GridLowerPrice) / float64(conditions.GridLevels))))
		if currentLevel < 0 {
			currentLevel = 0
		}
		if currentLevel >= conditions.GridLevels {
			currentLevel = conditions.GridLevels - 1
		}
		metrics["current_grid_level"] = currentLevel
	}

	return metrics
}

func (e *GridTradingStrategyExecutor) OptimizeGridParameters(historicalPrices []float64, targetReturn, maxDrawdown float64) StrategyConditions {
	if len(historicalPrices) < 10 {
		return StrategyConditions{
			GridTradingEnabled:   true,
			GridLevels:           10,
			GridProfitPercent:    1.0,
			GridInvestmentAmount: 1000.0,
		}
	}

	minPrice, maxPrice := historicalPrices[0], historicalPrices[0]
	for _, price := range historicalPrices {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}

	volatility := e.calculateVolatility(historicalPrices)

	return StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       maxPrice * 1.1,
		GridLowerPrice:       minPrice * 0.9,
		GridLevels:           e.calculateOptimalLevels(volatility),
		GridProfitPercent:    targetReturn / float64(e.calculateOptimalLevels(volatility)),
		GridInvestmentAmount: 1000.0,
	}
}

func (e *GridTradingStrategyExecutor) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.05
	}

	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	sum := 0.0
	for _, ret := range returns {
		sum += ret * ret
	}

	return math.Sqrt(sum / float64(len(returns)))
}

func (e *GridTradingStrategyExecutor) calculateOptimalLevels(volatility float64) int {
	if volatility > 0.1 {
		return 5
	} else if volatility > 0.05 {
		return 8
	}
	return 12
}

// 数据结构定义
type StrategyConditions struct {
	GridTradingEnabled   bool    `json:"grid_trading_enabled"`
	GridUpperPrice       float64 `json:"grid_upper_price"`
	GridLowerPrice       float64 `json:"grid_lower_price"`
	GridLevels           int     `json:"grid_levels"`
	GridProfitPercent    float64 `json:"grid_profit_percent"`
	GridInvestmentAmount float64 `json:"grid_investment_amount"`
	GridRebalanceEnabled bool    `json:"grid_rebalance_enabled"`
	GridStopLossEnabled  bool    `json:"grid_stop_loss_enabled"`
	GridStopLossPercent  float64 `json:"grid_stop_loss_percent"`
}

type GridOrder struct {
	ID        uint    `json:"id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	GridLevel int     `json:"grid_level"`
	Status    string  `json:"status"`
}

func main() {
	fmt.Println("🔗 网格交易策略集成测试")
	fmt.Println("================================")

	// 创建网格策略执行器
	executor := &GridTradingStrategyExecutor{}

	// 测试不同的网格配置
	testConfigs := []struct {
		name string
		config StrategyConditions
		description string
	}{
		{
			name: "标准网格配置",
			config: StrategyConditions{
				GridTradingEnabled:   true,
				GridUpperPrice:       50000,
				GridLowerPrice:       45000,
				GridLevels:           10,
				GridProfitPercent:    1.0,
				GridInvestmentAmount: 1000,
				GridStopLossEnabled:  true,
				GridStopLossPercent:  10.0,
			},
			description: "标准10层网格，1%利润率",
		},
		{
			name: "保守网格配置",
			config: StrategyConditions{
				GridTradingEnabled:   true,
				GridUpperPrice:       48000,
				GridLowerPrice:       46000,
				GridLevels:           8,
				GridProfitPercent:    0.5,
				GridInvestmentAmount: 500,
				GridStopLossEnabled:  true,
				GridStopLossPercent:  5.0,
			},
			description: "保守8层网格，0.5%利润率",
		},
		{
			name: "激进网格配置",
			config: StrategyConditions{
				GridTradingEnabled:   true,
				GridUpperPrice:       55000,
				GridLowerPrice:       40000,
				GridLevels:           15,
				GridProfitPercent:    2.0,
				GridInvestmentAmount: 2000,
				GridStopLossEnabled:  true,
				GridStopLossPercent:  15.0,
			},
			description: "激进15层网格，2%利润率",
		},
	}

	fmt.Println("\n📊 测试不同网格配置")
	fmt.Println("───────────────────────────────")

	for i, test := range testConfigs {
		fmt.Printf("\n%d. %s\n", i+1, test.name)
		fmt.Printf("   配置: %s\n", test.description)

		// 验证参数
		errors := executor.ValidateGridParameters(test.config)
		if len(errors) > 0 {
			fmt.Printf("   ❌ 参数验证失败: %v\n", errors)
			continue
		}

		fmt.Printf("   ✅ 参数验证通过\n")

		// 测试网格订单创建
		orders, err := executor.CreateGridOrders("BTCUSDT",
			test.config.GridUpperPrice,
			test.config.GridLowerPrice,
			test.config.GridLevels,
			test.config.GridInvestmentAmount)

		if err != nil {
			fmt.Printf("   ❌ 创建订单失败: %v\n", err)
			continue
		}

		fmt.Printf("   📋 创建订单: %d个\n", len(orders))

		// 显示订单统计
		buyCount := 0
		sellCount := 0
		for _, order := range orders {
			if order.Side == "buy" {
				buyCount++
			} else {
				sellCount++
			}
		}
		fmt.Printf("   💰 买入订单: %d个, 卖出订单: %d个\n", buyCount, sellCount)

		// 测试网格指标
		metrics := executor.GetGridMetrics(test.config, 47500)
		if enabled, ok := metrics["enabled"].(bool); ok && enabled {
			if level, ok := metrics["current_grid_level"].(int); ok {
				fmt.Printf("   📈 当前网格级别: %d/%d\n", level, test.config.GridLevels)
			}
			if spacing, ok := metrics["grid_spacing"].(float64); ok {
				fmt.Printf("   📏 网格间距: %.2f USDT\n", spacing)
			}
		}
	}

	fmt.Println("\n🎯 网格策略决策测试")
	fmt.Println("───────────────────────────────")

	// 使用标准配置测试决策逻辑
	standardConfig := StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       50000,
		GridLowerPrice:       45000,
		GridLevels:           10,
		GridProfitPercent:    1.0,
		GridInvestmentAmount: 1000,
	}

	// 模拟不同价格的决策
	testPrices := []float64{45500, 46500, 47500, 48500, 49500}

	for _, price := range testPrices {
		// 注意：这里我们无法直接调用ExecuteFull，因为需要真实的服务器实例
		// 所以我们直接测试网格指标和决策逻辑
		metrics := executor.GetGridMetrics(standardConfig, price)

		fmt.Printf("\n💰 价格 %.0f USDT:\n", price)

		if level, ok := metrics["current_grid_level"].(int); ok {
			fmt.Printf("   📍 网格级别: %d/%d\n", level, standardConfig.GridLevels)

			// 模拟决策逻辑
			midLevel := standardConfig.GridLevels / 2
			if level < midLevel {
				fmt.Printf("   📈 建议: 买入 (价格在网格下半部分)\n")
			} else if level > midLevel {
				fmt.Printf("   📉 建议: 卖出 (价格在网格上半部分)\n")
			} else {
				fmt.Printf("   🔄 建议: 观望 (价格在中性位置)\n")
			}
		}

		if inRange, ok := metrics["in_range"].(bool); ok {
			if inRange {
				fmt.Printf("   ✅ 价格在网格范围内\n")
			} else {
				fmt.Printf("   ⚠️  价格超出网格范围\n")
			}
		}
	}

	fmt.Println("\n🔧 网格参数优化测试")
	fmt.Println("───────────────────────────────")

	// 测试参数优化
	historicalPrices := []float64{
		45000, 45500, 46000, 46500, 47000, 47500, 48000, 48500, 49000, 49500,
		50000, 49500, 49000, 48500, 48000, 47500, 47000, 46500, 46000, 45500,
	}

	optimizedConfig := executor.OptimizeGridParameters(historicalPrices, 2.0, 10.0)

	fmt.Printf("历史价格范围: %.0f - %.0f\n", 45000.0, 50000.0)
	fmt.Printf("优化后网格范围: %.2f - %.2f\n", optimizedConfig.GridLowerPrice, optimizedConfig.GridUpperPrice)
	fmt.Printf("优化后网格层数: %d\n", optimizedConfig.GridLevels)
	fmt.Printf("优化后利润率: %.2f%%\n", optimizedConfig.GridProfitPercent)
	fmt.Printf("优化后投资金额: %.0f USDT\n", optimizedConfig.GridInvestmentAmount)

	fmt.Println("\n📈 网格策略分析")
	fmt.Println("───────────────────────────────")

	// 分析网格策略的优缺点
	fmt.Println("✅ 优点:")
	fmt.Println("   • 适应震荡行情，适合当前市场环境")
	fmt.Println("   • 自动化执行，无需人工干预")
	fmt.Println("   • 风险可控，预设止损机制")
	fmt.Println("   • 收益稳定，适合长期投资")

	fmt.Println("\n⚠️  注意事项:")
	fmt.Println("   • 不适合单边趋势行情")
	fmt.Println("   • 交易费用可能影响小幅利润")
	fmt.Println("   • 需要充足资金维持网格")
	fmt.Println("   • 极端行情可能突破网格范围")

	fmt.Println("\n🎯 建议配置:")
	fmt.Println("   • 震荡行情: 8-12层网格，0.5-1%利润率")
	fmt.Println("   • 投资金额: 根据总资金的1-2%分配")
	fmt.Println("   • 止损设置: 5-10%作为安全边界")
	fmt.Println("   • 监控频率: 定期检查网格状态")

	fmt.Println("\n✅ 网格交易策略集成测试完成！")
}