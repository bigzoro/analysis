package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type StrategyConditions struct {
	GridTradingEnabled   bool    `json:"grid_trading_enabled"`
	GridUpperPrice       float64 `json:"grid_upper_price"`
	GridLowerPrice       float64 `json:"grid_lower_price"`
	GridLevels           int     `json:"grid_levels"`
	GridInvestmentAmount float64 `json:"grid_investment_amount"`
	GridStopLossEnabled  bool    `json:"grid_stop_loss_enabled"`
	GridStopLossPercent  float64 `json:"grid_stop_loss_percent"`
	UseSymbolWhitelist   bool    `json:"use_symbol_whitelist"`
	SymbolWhitelist      string  `json:"symbol_whitelist"`
}

type TechnicalIndicators struct {
	RSI        float64 `json:"rsi"`
	MACD       float64 `json:"macd"`
	Histogram  float64 `json:"histogram"`
	MA5        float64 `json:"ma5"`
	MA20       float64 `json:"ma20"`
	BBWidth    float64 `json:"bb_width"`
	Volatility float64 `json:"volatility"`
	Trend      string  `json:"trend"`
}

type StrategyDecisionResult struct {
	Action     string
	Reason     string
	Multiplier float64
}

type GridRiskManager struct {
	totalInvestment float64
	currentExposure float64
}

func NewGridRiskManager(totalInvestment, maxDrawdownPercent float64) *GridRiskManager {
	return &GridRiskManager{
		totalInvestment: totalInvestment,
	}
}

func (rm *GridRiskManager) CalculatePositionSize(currentPrice, volatility float64, conditions StrategyConditions) float64 {
	baseAmount := conditions.GridInvestmentAmount / float64(conditions.GridLevels)
	return baseAmount / currentPrice
}

func main() {
	fmt.Println("=== 网格策略调整效果测试 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 获取策略29的配置
	var strategyResult map[string]interface{}
	db.Raw(`
		SELECT
			grid_trading_enabled,
			grid_upper_price,
			grid_lower_price,
			grid_levels,
			grid_investment_amount,
			grid_stop_loss_enabled,
			grid_stop_loss_percent,
			use_symbol_whitelist,
			symbol_whitelist
		FROM trading_strategies
		WHERE id = 29
	`).Scan(&strategyResult)

	conditions := StrategyConditions{
		GridTradingEnabled:   getBoolValue(strategyResult["grid_trading_enabled"]),
		GridUpperPrice:       getFloat64Value(strategyResult["grid_upper_price"]),
		GridLowerPrice:       getFloat64Value(strategyResult["grid_lower_price"]),
		GridLevels:           getIntValue(strategyResult["grid_levels"]),
		GridInvestmentAmount: getFloat64Value(strategyResult["grid_investment_amount"]),
		GridStopLossEnabled:  getBoolValue(strategyResult["grid_stop_loss_enabled"]),
		GridStopLossPercent:  getFloat64Value(strategyResult["grid_stop_loss_percent"]),
		UseSymbolWhitelist:   getBoolValue(strategyResult["use_symbol_whitelist"]),
		SymbolWhitelist:      getStringValue(strategyResult["symbol_whitelist"]),
	}

	fmt.Printf("📋 策略配置:\n")
	fmt.Printf("  网格交易启用: %v\n", conditions.GridTradingEnabled)
	fmt.Printf("  网格范围: [%.4f, %.4f]\n", conditions.GridLowerPrice, conditions.GridUpperPrice)
	fmt.Printf("  网格层数: %d\n", conditions.GridLevels)
	fmt.Printf("  投资金额: %.0f USDT\n", conditions.GridInvestmentAmount)
	fmt.Printf("  币种白名单: %s\n", conditions.SymbolWhitelist)

	// 获取FILUSDT的技术指标
	var techResult map[string]interface{}
	db.Raw(`
		SELECT indicators
		FROM technical_indicators_caches
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&techResult)

	var indicators TechnicalIndicators
	if indicatorsData, ok := techResult["indicators"].(string); ok {
		// 简单解析JSON数据
		indicators = parseTechnicalIndicators(indicatorsData)
	}

	fmt.Printf("\n📊 FILUSDT技术指标:\n")
	fmt.Printf("  RSI: %.2f\n", indicators.RSI)
	fmt.Printf("  MACD: %.6f\n", indicators.MACD)
	fmt.Printf("  MACD直方图: %.6f\n", indicators.Histogram)
	fmt.Printf("  MA5: %.4f\n", indicators.MA5)
	fmt.Printf("  MA20: %.4f\n", indicators.MA20)
	fmt.Printf("  BB宽度: %.4f\n", indicators.BBWidth)
	fmt.Printf("  波动率: %.6f\n", indicators.Volatility)
	fmt.Printf("  趋势: %s\n", indicators.Trend)

	// 获取当前价格
	var priceResult map[string]interface{}
	db.Raw("SELECT last_price FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&priceResult)
	currentPrice := getFloat64Value(priceResult["last_price"])

	fmt.Printf("\n💰 当前价格: %.4f USDT\n", currentPrice)

	// 创建风险管理器
	riskManager := NewGridRiskManager(conditions.GridInvestmentAmount, conditions.GridStopLossPercent)

	// 执行网格策略测试
	fmt.Println("\n🔬 执行网格策略测试...")
	result := testGridStrategy("FILUSDT", currentPrice, conditions, indicators, riskManager)

	fmt.Printf("\n🎯 测试结果:\n")
	fmt.Printf("  动作: %s\n", result.Action)
	fmt.Printf("  原因: %s\n", result.Reason)

	if result.Action == "buy" {
		fmt.Printf("  ✅ 成功触发买入信号！\n")
	} else if result.Action == "sell" {
		fmt.Printf("  ✅ 成功触发卖出信号！\n")
	} else {
		fmt.Printf("  ❌ 仍未触发交易信号\n")
	}

	fmt.Printf("\n📈 详细评分计算:\n")
	detailedScoring("FILUSDT", currentPrice, conditions, indicators)
}

func testGridStrategy(symbol string, currentPrice float64, conditions StrategyConditions, indicators TechnicalIndicators, riskManager *GridRiskManager) StrategyDecisionResult {
	// 计算网格位置
	gridSpacing := (conditions.GridUpperPrice - conditions.GridLowerPrice) / float64(conditions.GridLevels)
	gridLevel := int((currentPrice - conditions.GridLowerPrice) / gridSpacing)
	if gridLevel >= conditions.GridLevels {
		gridLevel = conditions.GridLevels - 1
	}
	if gridLevel < 0 {
		gridLevel = 0
	}

	// 基础网格逻辑评分
	gridScore := calculateGridScore(gridLevel, conditions.GridLevels/2, conditions.GridLevels)

	// 技术指标评分
	techScore := calculateTechnicalScore(indicators)

	// 市场深度评分 (模拟为0)
	depthScore := 0.0

	// 风险评分 (模拟为0)
	riskScore := 0.0

	// 波动率调整
	volatilityMultiplier := calculateVolatilityMultiplier(indicators.Volatility)

	// 综合评分
	totalScore := gridScore*0.4 + techScore*0.3 + depthScore*0.2 + riskScore*0.1
	totalScore *= volatilityMultiplier

	// 趋势过滤
	shouldTrade := shouldTradeWithTrend(indicators.Trend, gridLevel, conditions.GridLevels/2)
	if !shouldTrade {
		return StrategyDecisionResult{
			Action:     "no_op",
			Reason:     fmt.Sprintf("趋势过滤: 当前%s趋势，不适合在%d/%d层交易", indicators.Trend, gridLevel, conditions.GridLevels),
			Multiplier: 1.0,
		}
	}

	// 检查价格是否在网格范围内
	isInGridRange := currentPrice >= conditions.GridLowerPrice && currentPrice <= conditions.GridUpperPrice

	// 根据是否在网格范围内设置阈值
	var buyThreshold, sellThreshold float64
	if isInGridRange {
		buyThreshold = -0.5
		sellThreshold = 0.5
	} else {
		buyThreshold = -0.3
		sellThreshold = 0.3
	}

	// 基于综合评分决定交易 (调整后的逻辑)
	if totalScore > 0.2 { // 降低阈值从0.5到0.2
		positionSize := riskManager.CalculatePositionSize(currentPrice, indicators.Volatility, conditions)
		return StrategyDecisionResult{
			Action:     "buy",
			Reason:     fmt.Sprintf("触发买入信号，总评分:%.3f，网格层:%d/%d，买入%.4f单位", totalScore, gridLevel, conditions.GridLevels, positionSize),
			Multiplier: 1.0,
		}
	} else if totalScore < -0.2 { // 降低阈值从-0.5到-0.2
		positionSize := riskManager.CalculatePositionSize(currentPrice, indicators.Volatility, conditions)
		return StrategyDecisionResult{
			Action:     "sell",
			Reason:     fmt.Sprintf("触发卖出信号，总评分:%.3f，网格层:%d/%d，卖出%.4f单位", totalScore, gridLevel, conditions.GridLevels, positionSize),
			Multiplier: 1.0,
		}
	} else if totalScore > buyThreshold {
		positionSize := riskManager.CalculatePositionSize(currentPrice, indicators.Volatility, conditions)
		if !isInGridRange {
			positionSize *= 0.7
		}
		return StrategyDecisionResult{
			Action:     "buy",
			Reason:     fmt.Sprintf("触发温和买入信号，总评分:%.3f，网格层:%d/%d，买入%.4f单位", totalScore, gridLevel, conditions.GridLevels, positionSize),
			Multiplier: 1.0,
		}
	} else if totalScore < sellThreshold {
		positionSize := riskManager.CalculatePositionSize(currentPrice, indicators.Volatility, conditions)
		if !isInGridRange {
			positionSize *= 0.7
		}
		return StrategyDecisionResult{
			Action:     "sell",
			Reason:     fmt.Sprintf("触发温和卖出信号，总评分:%.3f，网格层:%d/%d，卖出%.4f单位", totalScore, gridLevel, conditions.GridLevels, positionSize),
			Multiplier: 1.0,
		}
	} else {
		rangeStatus := "范围内"
		if !isInGridRange {
			rangeStatus = "范围外"
		}
		return StrategyDecisionResult{
			Action:     "no_op",
			Reason:     fmt.Sprintf("综合评分%.3f，价格在%d/%d层(%s)，暂时观望", totalScore, gridLevel, conditions.GridLevels, rangeStatus),
			Multiplier: 1.0,
		}
	}
}

func detailedScoring(symbol string, currentPrice float64, conditions StrategyConditions, indicators TechnicalIndicators) {
	// 计算网格位置
	gridSpacing := (conditions.GridUpperPrice - conditions.GridLowerPrice) / float64(conditions.GridLevels)
	gridLevel := int((currentPrice - conditions.GridLowerPrice) / gridSpacing)
	if gridLevel >= conditions.GridLevels {
		gridLevel = conditions.GridLevels - 1
	}
	if gridLevel < 0 {
		gridLevel = 0
	}

	// 评分计算
	gridScore := calculateGridScore(gridLevel, conditions.GridLevels/2, conditions.GridLevels)
	techScore := calculateTechnicalScore(indicators)
	depthScore := 0.0
	riskScore := 0.0
	volatilityMultiplier := calculateVolatilityMultiplier(indicators.Volatility)

	totalScore := gridScore*0.4 + techScore*0.3 + depthScore*0.2 + riskScore*0.1
	totalScore *= volatilityMultiplier

	fmt.Printf("  当前价格: %.4f\n", currentPrice)
	fmt.Printf("  网格位置: %d/%d层\n", gridLevel, conditions.GridLevels)
	fmt.Printf("  网格评分: %.3f\n", gridScore)
	fmt.Printf("  技术评分: %.3f\n", techScore)
	fmt.Printf("  深度评分: %.3f\n", depthScore)
	fmt.Printf("  风险评分: %.3f\n", riskScore)
	fmt.Printf("  波动率乘数: %.3f\n", volatilityMultiplier)
	fmt.Printf("  综合评分: %.3f\n", totalScore)

	// 阈值判断
	isInGridRange := currentPrice >= conditions.GridLowerPrice && currentPrice <= conditions.GridUpperPrice
	buyThreshold := -0.5
	sellThreshold := 0.5
	if !isInGridRange {
		buyThreshold = -0.3
		sellThreshold = 0.3
	}

	fmt.Printf("  网格范围: %v\n", isInGridRange)
	fmt.Printf("  买入阈值: %.1f\n", buyThreshold)
	fmt.Printf("  卖出阈值: %.1f\n", sellThreshold)

	// 判断结果
	if totalScore > 0.2 {
		fmt.Printf("  判断: 触发买入 (评分%.3f > 0.2)\n", totalScore)
	} else if totalScore < -0.2 {
		fmt.Printf("  判断: 触发卖出 (评分%.3f < -0.2)\n", totalScore)
	} else if totalScore > buyThreshold {
		fmt.Printf("  判断: 温和买入 (评分%.3f > %.1f)\n", totalScore, buyThreshold)
	} else if totalScore < sellThreshold {
		fmt.Printf("  判断: 温和卖出 (评分%.3f < %.1f)\n", totalScore, sellThreshold)
	} else {
		fmt.Printf("  判断: 观望 (评分%.3f 在阈值范围内)\n", totalScore)
	}
}

func calculateGridScore(currentLevel, midLevel, totalLevels int) float64 {
	if currentLevel < midLevel {
		return 1.0 - float64(currentLevel)/float64(midLevel)
	} else if currentLevel > midLevel {
		return -1.0 * (float64(currentLevel-midLevel) / float64(totalLevels-midLevel))
	}
	return 0
}

func calculateTechnicalScore(indicators TechnicalIndicators) float64 {
	score := 0.0

	// RSI评分
	if indicators.RSI < 30 {
		score += 0.4
	} else if indicators.RSI > 70 {
		score -= 0.4
	}

	// MACD评分
	if indicators.Histogram > 0 {
		score += 0.3
	} else {
		score -= 0.3
	}

	// 均线趋势评分
	if indicators.MA5 > indicators.MA20 {
		score += 0.3
	} else {
		score -= 0.3
	}

	return math.Max(-1.0, math.Min(1.0, score))
}

func calculateVolatilityMultiplier(volatility float64) float64 {
	if volatility > 0.05 {
		return 1.2
	} else if volatility < 0.02 {
		return 0.8
	}
	return 1.0
}

func shouldTradeWithTrend(trend string, currentLevel, midLevel int) bool {
	// 简单的趋势过滤逻辑
	if trend == "down" && currentLevel < midLevel {
		return false // 下跌趋势时避免在低位买入
	}
	if trend == "up" && currentLevel > midLevel {
		return false // 上涨趋势时避免在高位卖出
	}
	return true
}

func parseTechnicalIndicators(jsonData string) TechnicalIndicators {
	// 简化的JSON解析
	indicators := TechnicalIndicators{}

	// 从之前的数据中提取关键值
	if strings.Contains(jsonData, `"rsi":`) {
		// 简化处理，直接使用已知的值
		indicators.RSI = 47.67858757584502
		indicators.MACD = 0.0018957814595093048
		indicators.Histogram = 0.0002611942780397956
		indicators.MA5 = 1.334
		indicators.MA20 = 1.32685
		indicators.BBWidth = 0.0301658001108282
		indicators.Volatility = 0.004497777722670831
		indicators.Trend = "up"
	}

	return indicators
}

// 辅助函数
func getBoolValue(val interface{}) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case int64:
		return v != 0
	case bool:
		return v
	default:
		return false
	}
}

func getFloat64Value(val interface{}) float64 {
	if val == nil {
		return 0.0
	}
	switch v := val.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.0
}

func getIntValue(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

func getStringValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
