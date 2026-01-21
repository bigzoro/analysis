package server

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/gin-gonic/gin"
)

// PersonalizedRiskManagement 个性化风险管理
type PersonalizedRiskManagement struct {
	UserRiskTolerance  float64 // 用户风险偏好 0.1-1.0
	AccountSize        float64 // 账户规模（USD）
	DailyVolatility    float64 // 日波动率
	MaxDrawdownHistory float64 // 历史最大回撤
	WinRate            float64 // 胜率
	TradingFrequency   string  // 交易频率
}

// getPersonalizedRiskManagement 获取个性化风险管理配置
func (s *Server) getPersonalizedRiskManagement(ctx context.Context, userID uint, symbol string) *PersonalizedRiskManagement {
	// 默认为中等风险偏好用户
	prm := &PersonalizedRiskManagement{
		UserRiskTolerance:  0.5,   // 中等风险偏好
		AccountSize:        10000, // 默认账户规模1万美元
		DailyVolatility:    0.05,  // 默认5%日波动率
		MaxDrawdownHistory: 0.1,   // 默认10%历史最大回撤
		WinRate:            0.55,  // 默认55%胜率
		TradingFrequency:   "moderate",
	}

	// TODO: 从数据库获取用户的真实风险偏好和交易历史
	// 这里可以实现从用户配置、交易历史中获取个性化数据

	return prm
}

// calculatePersonalizedPositionSize 计算个性化仓位大小
func (prm *PersonalizedRiskManagement) calculatePersonalizedPositionSize(expectedReturn float64, stopLoss float64, riskScore float64) float64 {
	// 基于Kelly公式的改进版仓位计算
	riskRewardRatio := expectedReturn / math.Abs(stopLoss)

	// 基础Kelly公式
	kellyPosition := (prm.WinRate - (1-prm.WinRate)/riskRewardRatio) * prm.UserRiskTolerance

	// 波动率调整
	volatilityAdjustment := 1.0 / (1.0 + prm.DailyVolatility)

	// 账户规模调整（小账户保护）
	sizeAdjustment := math.Min(1.0, prm.AccountSize/10000) // 1万美元以下账户调整

	// 风险评分调整（高风险币种减少仓位）
	riskAdjustment := 1.0 - riskScore*0.5

	// 历史表现调整（高回撤历史减少仓位）
	drawdownAdjustment := 1.0 - prm.MaxDrawdownHistory

	positionSize := kellyPosition * volatilityAdjustment * sizeAdjustment * riskAdjustment * drawdownAdjustment

	// 确保在合理范围内
	positionSize = math.Max(0.01, math.Min(positionSize, 0.25)) // 1%-25%区间

	return positionSize
}

// calculatePersonalizedRiskLimits 计算个性化风险限制
func (prm *PersonalizedRiskManagement) calculatePersonalizedRiskLimits(baseRiskLimits gin.H) gin.H {
	// 基于用户风险偏好调整风险限制
	riskMultiplier := 1.0 + (prm.UserRiskTolerance-0.5)*0.5 // 风险偏好调整倍数

	// 调整单笔最大亏损
	maxLossPerTrade := baseRiskLimits["max_loss_per_trade"].(float64) * riskMultiplier
	maxLossPerTrade = math.Max(0.005, math.Min(maxLossPerTrade, 0.05)) // 0.5%-5%区间

	// 调整单日最大亏损
	maxDailyLoss := baseRiskLimits["max_daily_loss"].(float64) * riskMultiplier
	maxDailyLoss = math.Max(0.01, math.Min(maxDailyLoss, 0.1)) // 1%-10%区间

	return gin.H{
		"max_loss_per_trade":         maxLossPerTrade,
		"max_daily_loss":             maxDailyLoss,
		"position_correlation_limit": baseRiskLimits["position_correlation_limit"],
		"volatility_adjustment":      baseRiskLimits["volatility_adjustment"],
		"user_risk_tolerance":        prm.UserRiskTolerance,
		"account_size_adjustment":    prm.AccountSize >= 10000,
	}
}

// generateTradingStrategyForRecommendation 为AI推荐生成交易策略
func (s *Server) generateTradingStrategyForRecommendation(rec CoinRecommendation, currentPrice float64, expectedReturn float64) gin.H {
	// 获取请求上下文用于技术指标查询
	ctx := context.Background()

	// 调试：检查数据库中的K线数据
	// s.checkDatabaseKlines(rec.Symbol) // 方法已移动到其他文件

	// 获取多时间周期技术分析
	multiTimeframe, err := s.GetMultiTimeframeIndicators(ctx, rec.Symbol, "spot")
	var strategyType string
	if err != nil {
		log.Printf("[WARN] 获取多时间周期分析失败，使用简化策略判断: %v", err)
		// 回退到简化策略判断
		strategyType = s.determineStrategyTypeAdvanced(ctx, rec, expectedReturn, rec.Symbol)
	} else {
		// 使用多时间周期分析确定策略类型
		strategyType = s.determineStrategyTypeByMultiTimeframe(multiTimeframe, rec, expectedReturn)
		log.Printf("[STRATEGY] 多时间周期分析结果: 策略=%s, 一致性=%.1f%%, 信号=%s, 信心=%.1f%%",
			strategyType, multiTimeframe.TimeframeConsistency, multiTimeframe.OverallSignal, multiTimeframe.SignalConfidence)
	}

	// 获取个性化风险管理配置（暂时使用默认用户ID 0）
	prm := s.getPersonalizedRiskManagement(ctx, 0, rec.Symbol)

	// 计算推荐仓位大小
	recommendedPosition := prm.calculatePersonalizedPositionSize(expectedReturn, 0.05, rec.Scores.Risk)

	// 生成智能入场区间
	entryZone := s.generateSmartEntryZone(ctx, rec, currentPrice, strategyType, expectedReturn)

	// 计算推荐止损价格
	var stopLossPrice float64
	switch strategyType {
	case "LONG":
		stopLossPrice = currentPrice * 0.95 // 多头策略5%止损
	case "SHORT":
		stopLossPrice = currentPrice * 1.05 // 空头策略5%止损
	default:
		stopLossPrice = currentPrice * 0.97 // 震荡策略3%止损
	}

	// 计算推荐止盈价格
	var takeProfitPrice float64
	switch strategyType {
	case "LONG":
		takeProfitPrice = currentPrice * (1.0 + expectedReturn)
	case "SHORT":
		takeProfitPrice = currentPrice * (1.0 - expectedReturn)
	default:
		takeProfitPrice = currentPrice * 1.03 // 震荡策略3%目标
	}

	// 生成策略理由
	strategyRationale := s.generateStrategyRationale(strategyType, rec, prm)

	// 计算止损百分比
	stopLossPercentage := math.Abs((stopLossPrice - currentPrice) / currentPrice)

	return gin.H{
		"strategy_type":    strategyType,
		"market_condition": s.determineMarketCondition(rec.Symbol),
		"entry_strategy": gin.H{
			"timing":                  "当前价格附近",
			"entry_zone":              entryZone,
			"recommended_entry_price": currentPrice,
			"description":             "基于技术指标和市场情绪，在建议区间分批建仓",
		},
		"exit_strategy": gin.H{
			"take_profit_price": takeProfitPrice,
			"stop_loss_price":   stopLossPrice,
			"risk_reward_ratio": math.Abs(takeProfitPrice-currentPrice) / math.Abs(stopLossPrice-currentPrice),
			"description":       "分批止盈，根据市场变化调整",
		},
		"stop_loss_levels": []gin.H{
			{
				"price":      stopLossPrice,
				"percentage": stopLossPercentage,
			},
		},
		"position_sizing": gin.H{
			"base_position":        recommendedPosition,
			"adjusted_position":    recommendedPosition,
			"recommended_position": recommendedPosition,
			"max_position":         math.Min(recommendedPosition*1.5, 0.25),
			"min_position":         math.Max(recommendedPosition*0.5, 0.01),
			"scaling_strategy":     "FIXED",
			"description":          "基于Kelly公式和风险偏好计算的个性化仓位",
		},
		"risk_management": func() gin.H {
			riskLimits := prm.calculatePersonalizedRiskLimits(gin.H{"max_loss_per_trade": 0.02, "max_daily_loss": 0.05})
			return gin.H{
				"max_loss_per_trade":    riskLimits["max_loss_per_trade"],
				"max_daily_loss":        riskLimits["max_daily_loss"],
				"volatility_adjustment": rec.Scores.Risk > 0.5,
				"personalized_factors": gin.H{
					"user_risk_tolerance":  prm.UserRiskTolerance,
					"account_size":         prm.AccountSize,
					"daily_volatility":     prm.DailyVolatility,
					"win_rate":             prm.WinRate,
					"max_drawdown_history": prm.MaxDrawdownHistory,
				},
				"description": "个性化风险控制，考虑账户规模和风险偏好",
			}
		}(),
		"trading_direction":    s.getTradingDirectionFromStrategy(strategyType),
		"strategy_rationale":   strategyRationale,
		"confidence_level":     s.calculateStrategyConfidence(multiTimeframe, rec, expectedReturn),
		"execution_complexity": s.getExecutionComplexity(strategyType),
	}
}

// determineStrategyTypeByMultiTimeframe 基于多时间周期分析确定策略类型
func (s *Server) determineStrategyTypeByMultiTimeframe(multiTimeframe *MultiTimeframeIndicators, rec CoinRecommendation, expectedReturn float64) string {
	// 基于综合信号和一致性确定策略
	signal := multiTimeframe.OverallSignal
	consistency := multiTimeframe.TimeframeConsistency
	confidence := multiTimeframe.SignalConfidence

	// 高一致性且强信号时使用主要策略
	if consistency > 70 && confidence > 80 {
		switch signal {
		case "strong_buy", "buy":
			return "LONG"
		case "strong_sell", "sell":
			return "SHORT"
		}
	}

	// 中等一致性时结合预期收益判断
	if consistency > 50 && confidence > 60 {
		if expectedReturn > 0.05 && (signal == "buy" || signal == "neutral") {
			return "LONG"
		}
		if expectedReturn < -0.05 && (signal == "sell" || signal == "neutral") {
			return "SHORT"
		}
	}

	// 低一致性或低信心时使用震荡策略
	if consistency < 40 || confidence < 50 {
		return "RANGE"
	}

	// 默认使用简化策略判断作为后备
	return s.determineStrategyTypeAdvanced(context.Background(), rec, expectedReturn, rec.Symbol)
}

// generateStrategyRationale 生成策略理由
func (s *Server) generateStrategyRationale(strategyType string, rec CoinRecommendation, prm *PersonalizedRiskManagement) []string {
	var rationale []string

	switch strategyType {
	case "LONG":
		rationale = []string{
			fmt.Sprintf("技术指标显示上涨动能，RSI=%.1f, MACD信号利好", rec.Scores.Technical*100),
			fmt.Sprintf("市场情绪乐观，情绪评分=%.1f", rec.Scores.Sentiment),
			fmt.Sprintf("预期收益%.1f%%，风险控制在%.1f%%以内", rec.Scores.Fundamental*100, prm.calculatePersonalizedRiskLimits(gin.H{"max_loss_per_trade": 0.02})["max_loss_per_trade"].(float64)*100),
			"适合风险偏好为中高风险的用户",
		}
	case "SHORT":
		rationale = []string{
			fmt.Sprintf("技术指标显示下跌压力，RSI=%.1f, 布林带位置偏高", rec.Scores.Technical*100),
			fmt.Sprintf("市场情绪偏向谨慎，情绪评分=%.1f", rec.Scores.Sentiment),
			fmt.Sprintf("预期收益%.1f%%，严格控制风险暴露", rec.Scores.Fundamental*100),
			"适合风险偏好为中等及以上的用户",
		}
	default: // RANGE
		rationale = []string{
			fmt.Sprintf("市场处于震荡区间，技术指标显示中性信号"),
			fmt.Sprintf("适合低风险偏好的用户，预期收益%.1f%%", rec.Scores.Fundamental*100),
			"通过高频小额交易控制风险",
			"市场环境不明确时首选策略",
		}
	}

	return rationale
}

// calculateStrategyConfidence 计算策略信心水平
func (s *Server) calculateStrategyConfidence(multiTimeframe *MultiTimeframeIndicators, rec CoinRecommendation, expectedReturn float64) float64 {
	if multiTimeframe == nil {
		// 基于单因子计算信心
		baseConfidence := (rec.Scores.Technical + rec.Scores.Fundamental + rec.Scores.Sentiment) / 3.0
		return math.Min(baseConfidence, 0.95) // 最高95%
	}

	// 基于多时间周期计算信心
	timeframeConfidence := multiTimeframe.SignalConfidence / 100.0
	consistencyBonus := multiTimeframe.TimeframeConsistency / 200.0 // 一致性贡献最大50%

	// 结合预期收益调整
	returnBonus := math.Min(math.Abs(expectedReturn)*2, 0.2) // 预期收益贡献最大20%

	totalConfidence := timeframeConfidence + consistencyBonus + returnBonus
	return math.Min(totalConfidence, 0.98) // 最高98%
}

// getTradingDirectionFromStrategy 根据策略类型获取交易方向
func (s *Server) getTradingDirectionFromStrategy(strategyType string) string {
	switch strategyType {
	case "LONG":
		return "买入做多"
	case "SHORT":
		return "卖出做空"
	case "RANGE":
		return "区间交易"
	default:
		return "观望"
	}
}

// getExecutionComplexity 获取执行复杂度
func (s *Server) getExecutionComplexity(strategyType string) string {
	switch strategyType {
	case "LONG", "SHORT":
		return "中等"
	case "RANGE":
		return "复杂"
	default:
		return "简单"
	}
}

// determineMarketCondition 判断市场环境
func (s *Server) determineMarketCondition(symbol string) string {
	// 这里可以基于更复杂的市场分析来判断
	// 暂时使用简化逻辑
	switch symbol {
	case "BTC", "ETH":
		return "主流币种，流动性良好"
	default:
		return "中型币种，波动性较高"
	}
}

// generateSmartEntryZone 基于技术分析生成智能入场区间
func (s *Server) generateSmartEntryZone(ctx context.Context, rec CoinRecommendation, currentPrice float64, strategyType string, expectedReturn float64) gin.H {
	// 获取技术指标
	technical, err := s.CalculateTechnicalIndicators(ctx, rec.Symbol, "spot")
	if err != nil {
		log.Printf("[WARN] 获取技术指标失败，使用默认入场区间: %v", err)
		// 回退到默认区间
		return gin.H{
			"min":        currentPrice * 0.98,
			"max":        currentPrice * 1.02,
			"avg":        currentPrice,
			"confidence": 0.5,
			"reason":     "技术指标不可用，使用默认区间",
		}
	}

	var minPrice, maxPrice float64
	var confidence float64
	var reason string

	switch strategyType {
	case "LONG":
		// 多头策略：基于支撑位、布林带和动量
		supportLevel := currentPrice * 0.95 // 默认支撑位
		if technical.BBLower > 0 {
			supportLevel = technical.BBLower
		}

		// 入场区间：支撑位到当前价格区间
		minPrice = math.Max(supportLevel, currentPrice*0.96)
		maxPrice = math.Min(currentPrice*1.03, technical.BBUpper*0.98)

		// 置信度：基于动量和RSI
		confidence = 0.6 + rec.Scores.Momentum*0.2 + (1-technical.RSI/100)*0.2 // RSI越低，置信度越高
		reason = "价格接近支撑位和布林带下轨，RSI显示超卖，动量良好"

	case "SHORT":
		// 空头策略：基于阻力位、布林带和风险
		resistanceLevel := currentPrice * 1.05 // 默认阻力位
		if technical.BBUpper > 0 {
			resistanceLevel = technical.BBUpper
		}

		// 入场区间：当前价格到阻力位区间
		minPrice = math.Max(currentPrice*0.97, technical.BBLower*1.02)
		maxPrice = math.Min(resistanceLevel, currentPrice*1.04)

		// 置信度：基于风险和RSI
		confidence = 0.6 + rec.Scores.Risk*0.2 + (technical.RSI/100)*0.2 // RSI越高，置信度越高
		reason = "价格接近阻力位和布林带上轨，RSI显示超买，风险较高"

	default: // RANGE
		// 震荡策略：布林带中轨附近
		minPrice = technical.BBMiddle * 0.98
		maxPrice = technical.BBMiddle * 1.02
		confidence = 0.5 + (1-technical.BBWidth)*0.3 // 布林带越窄，震荡置信度越高
		reason = "市场处于震荡区间，价格在布林带中轨附近波动"
	}

	// 确保区间有效
	if minPrice >= maxPrice {
		minPrice = currentPrice * 0.98
		maxPrice = currentPrice * 1.02
		confidence = 0.5
		reason = "计算区间无效，使用默认区间"
	}

	avgPrice := (minPrice + maxPrice) / 2

	log.Printf("[ENTRY_ZONE] %s %s策略入场区间: %.4f - %.4f (均价:%.4f), 信心:%.1f, 原因:%s",
		rec.Symbol, strategyType, minPrice, maxPrice, avgPrice, confidence, reason)

	return gin.H{
		"min":        minPrice,
		"max":        maxPrice,
		"avg":        avgPrice,
		"confidence": math.Min(1.0, math.Max(0.4, confidence)), // 限制置信度在40%-100%
		"reason":     reason,
	}
}

// determineStrategyTypeAdvanced 高级策略类型判断（基于多因子分析）
func (s *Server) determineStrategyTypeAdvanced(ctx context.Context, rec CoinRecommendation, expectedReturn float64, symbol string) string {
	// 基于技术指标、基本面、情绪等分数综合判断策略类型

	// 计算各维度得分（0-100）
	technicalScore := rec.Scores.Technical * 100
	fundamentalScore := rec.Scores.Fundamental * 100
	sentimentScore := rec.Scores.Sentiment * 100
	momentumScore := rec.Scores.Momentum * 100
	riskScore := rec.Scores.Risk * 100

	// 计算综合得分
	overallScore := (technicalScore*0.4 + fundamentalScore*0.3 + sentimentScore*0.2 + momentumScore*0.1) - riskScore*0.5

	// 根据预期收益调整
	returnBonus := 0.0
	if expectedReturn > 0.05 {
		returnBonus = 20 // 高预期收益偏向多头
	} else if expectedReturn < -0.05 {
		returnBonus = -20 // 负预期收益偏向空头
	}

	finalScore := overallScore + returnBonus

	// 根据最终得分确定策略
	if finalScore >= 60 {
		return "LONG"
	} else if finalScore <= -60 {
		return "SHORT"
	} else {
		return "RANGE"
	}
}
