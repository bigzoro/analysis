package main

import (
	"fmt"
	"log"
	"math"
)

// 简化的数据结构用于测试
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

// GridOrder 网格订单结构
type GridOrder struct {
	ID        uint
	Symbol    string
	Side      string // "buy" or "sell"
	Price     float64
	Quantity  float64
	GridLevel int
	Status    string // "pending", "filled", "cancelled"
}

// 简化的网格策略执行器
type GridTradingStrategyExecutor struct{}

// ValidateGridParameters 验证网格参数
func (e *GridTradingStrategyExecutor) ValidateGridParameters(conditions StrategyConditions) []string {
	var errors []string

	if !conditions.GridTradingEnabled {
		return errors // 如果未启用，不验证参数
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

	if conditions.GridProfitPercent < 0 {
		errors = append(errors, "网格利润百分比不能为负数")
	}

	if conditions.GridStopLossPercent < 0 || conditions.GridStopLossPercent > 100 {
		errors = append(errors, "网格止损百分比必须在0-100之间")
	}

	return errors
}

// CreateGridOrders 创建网格订单
func (e *GridTradingStrategyExecutor) CreateGridOrders(symbol string, upperPrice, lowerPrice float64, levels int, investmentAmount float64) ([]GridOrder, error) {
	if upperPrice <= lowerPrice || levels <= 0 || investmentAmount <= 0 {
		return nil, fmt.Errorf("无效的网格参数")
	}

	// 计算网格间距
	gridSpacing := (upperPrice - lowerPrice) / float64(levels)

	// 计算每个网格的投资金额
	gridAmount := investmentAmount / float64(levels)

	var orders []GridOrder
	orderID := uint(1)

	// 创建买入订单（从下往上）
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

	// 创建卖出订单（从上往下）
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

// GetGridMetrics 获取网格指标
func (e *GridTradingStrategyExecutor) GetGridMetrics(conditions StrategyConditions, currentPrice float64) map[string]interface{} {
	metrics := make(map[string]interface{})

	if !conditions.GridTradingEnabled {
		metrics["enabled"] = false
		return metrics
	}

	// 基本参数
	metrics["enabled"] = true
	metrics["upper_price"] = conditions.GridUpperPrice
	metrics["lower_price"] = conditions.GridLowerPrice
	metrics["levels"] = conditions.GridLevels
	metrics["profit_percent"] = conditions.GridProfitPercent
	metrics["investment_amount"] = conditions.GridInvestmentAmount

	// 计算指标
	if conditions.GridLevels > 0 {
		gridSpacing := (conditions.GridUpperPrice - conditions.GridLowerPrice) / float64(conditions.GridLevels)
		metrics["grid_spacing"] = gridSpacing
		metrics["grid_spacing_percent"] = (gridSpacing / ((conditions.GridUpperPrice + conditions.GridLowerPrice) / 2)) * 100
	}

	// 当前状态
	metrics["current_price"] = currentPrice
	metrics["in_range"] = currentPrice >= conditions.GridLowerPrice && currentPrice <= conditions.GridUpperPrice

	if conditions.GridLevels > 0 && conditions.GridUpperPrice > conditions.GridLowerPrice {
		currentLevel := int(math.Floor((currentPrice-conditions.GridLowerPrice) / ((conditions.GridUpperPrice - conditions.GridLowerPrice) / float64(conditions.GridLevels))))
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

// OptimizeGridParameters 优化网格参数
func (e *GridTradingStrategyExecutor) OptimizeGridParameters(historicalPrices []float64, targetReturn, maxDrawdown float64) StrategyConditions {
	if len(historicalPrices) < 50 {
		// 数据不足，返回默认参数
		return StrategyConditions{
			GridTradingEnabled:   true,
			GridLevels:           10,
			GridProfitPercent:    1.0,
			GridInvestmentAmount: 1000.0,
			GridStopLossEnabled:  true,
			GridStopLossPercent:  10.0,
		}
	}

	// 计算价格统计
	minPrice, maxPrice := historicalPrices[0], historicalPrices[0]
	sum := 0.0

	for _, price := range historicalPrices {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
		sum += price
	}

	avgPrice := sum / float64(len(historicalPrices))
	volatility := e.calculateVolatility(historicalPrices)

	// 基于波动率确定网格范围和层数
	priceRange := maxPrice - minPrice
	safetyMargin := volatility * 2 // 2倍波动率作为安全边际

	// 设置网格参数
	conditions := StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       avgPrice + (priceRange/2)*1.2 + safetyMargin,
		GridLowerPrice:       avgPrice - (priceRange/2)*1.2 - safetyMargin,
		GridLevels:           e.calculateOptimalLevels(volatility),
		GridProfitPercent:    targetReturn / float64(e.calculateOptimalLevels(volatility)),
		GridInvestmentAmount: 1000.0,
		GridRebalanceEnabled: true,
		GridStopLossEnabled:  true,
		GridStopLossPercent:  maxDrawdown,
	}

	return conditions
}

// calculateVolatility 计算波动率
func (e *GridTradingStrategyExecutor) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	sum := 0.0
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	sumSquares := 0.0
	for _, ret := range returns {
		sumSquares += math.Pow(ret-mean, 2)
	}

	return math.Sqrt(sumSquares / float64(len(returns)))
}

// calculateOptimalLevels 计算最优网格层数
func (e *GridTradingStrategyExecutor) calculateOptimalLevels(volatility float64) int {
	// 基于波动率确定层数：波动率越高，层数越少
	if volatility > 0.1 { // 高波动
		return 5
	} else if volatility > 0.05 { // 中等波动
		return 8
	} else { // 低波动
		return 12
	}
}

func main() {
	fmt.Println("🎯 测试网格交易策略")
	fmt.Println("=====================================")

	// 创建网格策略执行器
	executor := &GridTradingStrategyExecutor{}

	// 测试参数验证
	fmt.Println("\n1. 测试参数验证")
	testParams := []StrategyConditions{
		{GridTradingEnabled: false}, // 未启用
		{GridTradingEnabled: true, GridUpperPrice: 100, GridLowerPrice: 50, GridLevels: 10}, // 正常参数
		{GridTradingEnabled: true, GridUpperPrice: 50, GridLowerPrice: 100}, // 上限小于下限
		{GridTradingEnabled: true, GridUpperPrice: 100, GridLowerPrice: 50, GridLevels: 0}, // 层数为0
	}

	for i, params := range testParams {
		errors := executor.ValidateGridParameters(params)
		fmt.Printf("测试参数%d: %d个错误\n", i+1, len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	// 测试网格订单创建
	fmt.Println("\n2. 测试网格订单创建")
	orders, err := executor.CreateGridOrders("BTCUSDT", 50000, 45000, 10, 1000)
	if err != nil {
		log.Printf("创建网格订单失败: %v", err)
	} else {
		fmt.Printf("成功创建%d个网格订单\n", len(orders))
		fmt.Printf("示例订单:\n")
		for i, order := range orders[:5] { // 只显示前5个
			fmt.Printf("  %d. %s %s 价格:%.2f 数量:%.6f\n",
				i+1, order.Symbol, order.Side, order.Price, order.Quantity)
		}
	}

	// 测试网格指标计算
	fmt.Println("\n3. 测试网格指标计算")
	params := StrategyConditions{
		GridTradingEnabled:   true,
		GridUpperPrice:       50000,
		GridLowerPrice:       45000,
		GridLevels:           10,
		GridProfitPercent:    1.0,
		GridInvestmentAmount: 1000,
	}

	metrics := executor.GetGridMetrics(params, 47500)
	fmt.Printf("网格指标:\n")
	for key, value := range metrics {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// 测试参数优化
	fmt.Println("\n4. 测试参数优化")
	historicalPrices := []float64{45000, 46000, 47000, 48000, 49000, 50000, 49500, 48500, 47500, 46500}
	optimizedParams := executor.OptimizeGridParameters(historicalPrices, 2.0, 10.0)
	fmt.Printf("优化后的参数:\n")
	fmt.Printf("  上限价格: %.2f\n", optimizedParams.GridUpperPrice)
	fmt.Printf("  下限价格: %.2f\n", optimizedParams.GridLowerPrice)
	fmt.Printf("  网格层数: %d\n", optimizedParams.GridLevels)
	fmt.Printf("  利润百分比: %.2f%%\n", optimizedParams.GridProfitPercent)
	fmt.Printf("  投资金额: %.2f\n", optimizedParams.GridInvestmentAmount)

	fmt.Println("\n✅ 网格策略测试完成！")
}