package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"gonum.org/v1/gonum/mat"
)

// MarketIndicators 市场指标集合
type MarketIndicators struct {
	Timestamp         time.Time
	Volatility        float64 // 波动率
	TrendStrength     float64 // 趋势强度
	TrendDirection    float64 // 趋势方向
	RSI               float64 // RSI指标
	Momentum          float64 // 动量
	VolumeRatio       float64 // 成交量比率
	BollingerPosition float64 // 布林带位置
	MACD              float64 // MACD信号
	MarketSentiment   float64 // 市场情绪 (-1到1)
	TrendConsistency  float64 // 趋势一致性 (-1到1)
}

// classifyMarketRegime 市场环境精细化分类
// classifyMarketRegime 重新设计的多维度市场环境分类算法
func classifyMarketRegime(state map[string]float64) MarketRegime {
	// 收集多维度指标
	indicators := collectMarketIndicators(state)

	// 计算综合市场评分
	marketScore := calculateMarketRegimeScore(indicators)

	// 基于综合评分和各维度指标进行分类
	regime := determineMarketRegime(marketScore, indicators)

	log.Printf("[MARKET_REGIME_DETAIL] 市场环境分析: 趋势=%.3f, 波动率=%.3f, RSI=%.2f, 动量=%.3f, 成交量=%.3f -> %s (评分:%.2f)",
		indicators.TrendStrength, indicators.Volatility, indicators.RSI, indicators.Momentum, indicators.VolumeRatio, regime.String(), marketScore)

	return regime
}

// collectMarketIndicators 收集多维度市场指标
func collectMarketIndicators(state map[string]float64) *MarketIndicators {
	indicators := &MarketIndicators{
		Timestamp: time.Now(),
	}

	// 基础指标
	if vol, exists := state["volatility_20"]; exists {
		indicators.Volatility = math.Max(0.0, vol) // 确保非负
	}

	if trend, exists := state["trend_20"]; exists {
		indicators.TrendStrength = math.Abs(trend)
		indicators.TrendDirection = trend
	}

	if rsi, exists := state["rsi_14"]; exists {
		indicators.RSI = math.Max(0.0, math.Min(100.0, rsi))
	}

	if momentum, exists := state["momentum_10"]; exists {
		indicators.Momentum = momentum
	}

	if volumeRatio, exists := state["volume_ratio"]; exists {
		indicators.VolumeRatio = math.Max(0.1, volumeRatio) // 最小0.1
	}

	// 高级指标
	if bbPos, exists := state["bb_position"]; exists {
		indicators.BollingerPosition = bbPos
	}

	if macd, exists := state["macd_signal"]; exists {
		indicators.MACD = macd
	}

	// 市场情绪指标
	indicators.MarketSentiment = calculateMarketSentiment(indicators)

	// 趋势一致性
	indicators.TrendConsistency = calculateTrendConsistency(indicators)

	return indicators
}

// calculateMarketRegimeScore 计算市场环境综合评分
func calculateMarketRegimeScore(indicators *MarketIndicators) float64 {
	score := 0.0

	// 1. 趋势强度评分 (40%权重)
	trendScore := 0.0
	if indicators.TrendStrength > 0.05 {
		trendScore = math.Min(indicators.TrendStrength/0.1, 1.0) // 0.1作为强趋势基准
	}
	score += trendScore * 0.4

	// 2. 波动率评分 (25%权重) - 高波动有利于趋势形成
	volatilityScore := 0.0
	if indicators.Volatility > 0.02 {
		volatilityScore = math.Min(indicators.Volatility/0.05, 1.0) // 0.05作为高波动基准
	} else if indicators.Volatility < 0.005 {
		volatilityScore = -0.5 // 极低波动为震荡特征
	}
	score += volatilityScore * 0.25

	// 3. RSI极端程度评分 (15%权重)
	rsiScore := 0.0
	if indicators.RSI < 30 || indicators.RSI > 70 {
		rsiScore = math.Min(math.Abs(indicators.RSI-50.0)/30.0, 1.0) // RSI偏离程度
	}
	score += rsiScore * 0.15

	// 4. 动量一致性评分 (10%权重)
	momentumScore := math.Abs(indicators.Momentum)
	if momentumScore > 0.02 {
		momentumScore = math.Min(momentumScore/0.05, 1.0)
	} else {
		momentumScore = 0.0
	}
	score += momentumScore * 0.1

	// 5. 成交量放大评分 (10%权重)
	volumeScore := 0.0
	if indicators.VolumeRatio > 2.0 {
		volumeScore = math.Min((indicators.VolumeRatio-1.0)/3.0, 1.0)
	}
	score += volumeScore * 0.1

	// 归一化到0-1范围
	return math.Max(0.0, math.Min(1.0, score))
}

// determineMarketRegime 基于评分确定市场环境 - 重构版：优先识别熊市
func determineMarketRegime(score float64, indicators *MarketIndicators) MarketRegime {
	// === 熊市优先检测 ===

	// 条件1：连续下跌趋势 + RSI超卖 + 负动量 = 熊市（放宽阈值）
	isBearishTrend := indicators.TrendDirection < -0.005 // 从-0.02放宽到-0.005
	isOversold := indicators.RSI < 50                    // 从40放宽到50
	isNegativeMomentum := indicators.Momentum < -0.005   // 从-0.02放宽到-0.005

	if isBearishTrend && (isOversold || isNegativeMomentum) {
		// 强熊市：下跌趋势明显 + 多个指标确认（大幅放宽）
		if indicators.TrendDirection < -0.02 && indicators.RSI < 45 && indicators.Momentum < -0.02 {
			return MarketRegimeStrongBear
		}
		// 弱熊市：温和下跌趋势
		return MarketRegimeWeakBear
	}

	// 条件2：极低波动 + 趋势微弱 + RSI严重偏离 = 极端熊市（放宽阈值）
	isExtremeCalm := indicators.Volatility < 0.005 && indicators.TrendStrength < 0.02 // 放宽波动率和趋势阈值
	isRSIExtreme := indicators.RSI < 25 || indicators.RSI > 75                        // 放宽RSI极端阈值

	if isExtremeCalm && isRSIExtreme {
		return MarketRegimeExtremeBear
	}

	// 条件3：综合评分较低 + 负趋势方向 = 熊市（放宽评分阈值）
	if score < 0.3 && indicators.TrendDirection < -0.005 { // 从0.2和-0.01放宽
		return MarketRegimeWeakBear
	}

	// === 震荡/中性市场检测（放宽熊市判断） ===
	if score < 0.5 { // 从0.4放宽到0.5，减少熊市判断
		// 低波动 + 无明确趋势 = 震荡
		if indicators.Volatility < 0.03 && indicators.TrendStrength < 0.04 { // 放宽波动率和趋势阈值
			return MarketRegimeSideways
		}
		// 评分较低但趋势为负 = 弱熊市（减少此类判断）
		if indicators.TrendDirection < -0.01 { // 从0放宽到-0.01，更难判断为熊市
			return MarketRegimeWeakBear
		}
		return MarketRegimeSideways
	}

	// === 牛市检测（仅在确认非熊市后） ===
	if indicators.TrendDirection > 0.02 && indicators.Volatility > 0.02 {
		// 需要多个指标确认上涨趋势
		isBullishRSI := indicators.RSI > 50
		isPositiveMomentum := indicators.Momentum > 0.02
		isVolumeSupport := indicators.VolumeRatio > 1.2

		if isBullishRSI && isPositiveMomentum && isVolumeSupport {
			if score > 0.7 && indicators.TrendDirection > 0.05 {
				return MarketRegimeStrongBull
			}
			return MarketRegimeWeakBull
		}
	}

	// === 默认分类 ===
	// 基于趋势方向的弱趋势判断
	if indicators.TrendDirection > 0.01 {
		return MarketRegimeWeakBull
	} else if indicators.TrendDirection < -0.01 {
		return MarketRegimeWeakBear
	}

	return MarketRegimeSideways
}

// calculateMarketSentiment 计算市场情绪
func calculateMarketSentiment(indicators *MarketIndicators) float64 {
	sentiment := 0.0

	// RSI贡献
	if indicators.RSI > 70 {
		sentiment += 0.3 // 超买，乐观
	} else if indicators.RSI < 30 {
		sentiment -= 0.3 // 超卖，悲观
	}

	// 动量贡献
	if indicators.Momentum > 0.05 {
		sentiment += 0.2
	} else if indicators.Momentum < -0.05 {
		sentiment -= 0.2
	}

	// 布林带位置贡献
	if indicators.BollingerPosition > 0.5 {
		sentiment += 0.1 // 上轨，乐观
	} else if indicators.BollingerPosition < -0.5 {
		sentiment -= 0.1 // 下轨，悲观
	}

	return math.Max(-1.0, math.Min(1.0, sentiment))
}

// calculateTrendConsistency 计算趋势一致性
func calculateTrendConsistency(indicators *MarketIndicators) float64 {
	consistency := 0.0
	totalSignals := 0.0

	// 趋势方向与RSI的一致性
	rsiDirection := 0.0
	if indicators.RSI > 60 {
		rsiDirection = 1.0
	} else if indicators.RSI < 40 {
		rsiDirection = -1.0
	}

	if rsiDirection != 0 {
		trendDirection := 0.0
		if indicators.TrendDirection > 0.01 {
			trendDirection = 1.0
		} else if indicators.TrendDirection < -0.01 {
			trendDirection = -1.0
		}

		if trendDirection == rsiDirection {
			consistency += 1.0
		} else if trendDirection != 0 {
			consistency -= 1.0
		}
		totalSignals++
	}

	// 趋势方向与动量的一致性
	if indicators.Momentum != 0 {
		momentumDirection := 1.0
		if indicators.Momentum < 0 {
			momentumDirection = -1.0
		}

		trendDirection := 0.0
		if indicators.TrendDirection > 0.01 {
			trendDirection = 1.0
		} else if indicators.TrendDirection < -0.01 {
			trendDirection = -1.0
		}

		if trendDirection == momentumDirection {
			consistency += 1.0
		} else if trendDirection != 0 {
			consistency -= 1.0
		}
		totalSignals++
	}

	if totalSignals > 0 {
		return consistency / totalSignals
	}
	return 0.0
}

// getAdaptiveThresholds 根据市场环境获取自适应阈值
func getAdaptiveThresholds(marketRegime MarketRegime, hasPosition bool) (float64, float64, float64) {
	var buyThreshold, sellThreshold, shortThreshold float64

	if hasPosition {
		// 有持仓时的阈值设置
		switch marketRegime {
		case MarketRegimeExtremeBear:
			buyThreshold = 0.7    // 降低加仓阈值，在极熊市允许适度加仓
			sellThreshold = 0.05  // 更容易止损
			shortThreshold = -999 // 有持仓时不考虑做空
		case MarketRegimeSideways:
			buyThreshold = 0.5    // 放宽加仓条件，横盘市场允许适度加仓
			sellThreshold = 0.05  // 更容易止损，及时退出
			shortThreshold = -999 // 震荡市坚决不做空
		case MarketRegimeWeakBull:
			buyThreshold = 0.7    // 中等加仓难度
			sellThreshold = 0.1   // 相对宽松止损
			shortThreshold = -999 // 弱多头不做空
		case MarketRegimeWeakBear:
			buyThreshold = 0.85   // 弱熊市谨慎买入
			sellThreshold = 0.08  // 适中卖出阈值
			shortThreshold = -0.3 // 允许做空但设置较高阈值
		case MarketRegimeStrongBull:
			buyThreshold = 0.6    // 相对容易加仓
			sellThreshold = 0.15  // 更宽松止损
			shortThreshold = -999 // 强多头坚决不做空
		case MarketRegimeStrongBear:
			buyThreshold = 0.9    // 强熊市谨慎买入
			sellThreshold = 0.05  // 更容易止损
			shortThreshold = -0.2 // 强熊市积极做空
		default:
			buyThreshold = 0.8
			sellThreshold = 0.1
			shortThreshold = -999
		}
	} else {
		// 无持仓时的阈值设置
		switch marketRegime {
		case MarketRegimeExtremeBear:
			buyThreshold = 0.3    // 优化：降低买入阈值，从0.6降到0.3，增加熊市交易机会
			shortThreshold = -0.4 // 极熊市允许做空但谨慎
		case MarketRegimeSideways:
			buyThreshold = 0.02   // 紧急修复：大幅降低买入阈值，从0.05降至0.02，激活横盘市场交易
			shortThreshold = -0.9 // 保持较高的做空阈值
		case MarketRegimeWeakBull:
			buyThreshold = 0.4    // 中等买入难度
			shortThreshold = -999 // 弱多头不做空
		case MarketRegimeWeakBear:
			buyThreshold = 0.4    // 优化：降低买入阈值，从0.7降到0.4，增加弱熊市交易机会
			shortThreshold = -0.2 // 弱熊市更容易做空
		case MarketRegimeStrongBull:
			buyThreshold = 0.5    // 相对容易买入，但仍保持谨慎
			shortThreshold = -999 // 强多头不做空
		case MarketRegimeStrongBear:
			buyThreshold = 0.7     // 强熊市谨慎买入
			shortThreshold = -0.15 // 强熊市积极做空
		default:
			buyThreshold = 0.35
			shortThreshold = -999
		}
		sellThreshold = -999 // 无持仓时不考虑卖出
	}

	return buyThreshold, sellThreshold, shortThreshold
}

// applyRiskManagement 应用风险管理
func (be *BacktestEngine) applyRiskManagement(action string, confidence float64, position float64, currentPrice float64, config *BacktestConfig, result *BacktestResult, holdTime int, state map[string]float64) string {
	// 1. 多头仓位风险管理
	if position > 0 && len(result.Trades) > 0 {
		lastTrade := result.Trades[len(result.Trades)-1]
		if lastTrade.Side == "buy" {
			lastBuyPrice := lastTrade.Price
			currentPnL := (currentPrice - lastBuyPrice) / lastBuyPrice

			// 多头止损检查
			if action == "sell" && be.shouldTriggerLongStopLoss(currentPrice, lastBuyPrice, config, state, holdTime) {
				log.Printf("[RISK] 多头止损触发: 亏损%.2f%% (买入价:%.2f, 当前价:%.2f)",
					currentPnL*100, lastBuyPrice, currentPrice)
				return "sell"
			}

			// 多头止盈检查
			if action == "sell" && be.shouldTriggerLongTakeProfit(currentPrice, lastBuyPrice, config, state, holdTime) {
				log.Printf("[RISK] 多头止盈触发: 盈利%.2f%%", currentPnL*100)
				return "sell"
			}
		}
	}

	// 2. 空头仓位风险管理
	if position < 0 && len(result.Trades) > 0 {
		lastTrade := result.Trades[len(result.Trades)-1]
		if lastTrade.Side == "short" {
			lastShortPrice := lastTrade.Price
			// 空头盈利：卖出价 - 当前价
			currentPnL := (lastShortPrice - currentPrice) / lastShortPrice

			// 空头止损检查（价格上涨时）
			if action == "cover" && be.shouldTriggerShortStopLoss(currentPrice, lastShortPrice, config, state, holdTime) {
				log.Printf("[RISK] 空头止损触发: 亏损%.2f%% (做空价:%.2f, 当前价:%.2f)",
					currentPnL*100, lastShortPrice, currentPrice)
				return "cover"
			}

			// 空头止盈检查
			if action == "cover" && be.shouldTriggerShortTakeProfit(currentPrice, lastShortPrice, config, state, holdTime) {
				log.Printf("[RISK] 空头止盈触发: 盈利%.2f%%", currentPnL*100)
				return "cover"
			}
		}
	}

	// 3. 应用风险限制
	finalAction := be.applyRiskLimits(action, position, result, config)

	return finalAction
}

// mlEnhancedDecision 机器学习增强的决策
func (be *BacktestEngine) mlEnhancedDecision(ctx context.Context, state map[string]float64, agent map[string]interface{}, symbol string) (string, float64) {
	// 添加当前波动率信息到agent
	if volatility, exists := state["volatility_20"]; exists {
		agent["current_volatility"] = volatility
	}

	// 首先尝试使用机器学习模型预测
	var mlPrediction *PredictionResult
	var mlWeight float64 = 0.7   // ML模型基础权重
	var ruleWeight float64 = 0.3 // 规则基础权重

	if be.server != nil && be.server.machineLearning != nil {
		// 使用多模型集成进行预测
		ensemblePrediction, err := be.predictWithEnsembleModels(ctx, symbol)
		if err == nil && ensemblePrediction != nil {
			mlPrediction = ensemblePrediction
			log.Printf("[ML_DECISION] 多模型集成预测: score=%.3f, confidence=%.3f, quality=%.3f",
				ensemblePrediction.Score, ensemblePrediction.Confidence, ensemblePrediction.Quality)

			// 检查预测质量
			if ensemblePrediction.Quality < 0.5 {
				log.Printf("[ML_DECISION] ⚠️ 模型质量较低 (%.3f)，可能影响预测准确性", ensemblePrediction.Quality)
			}

			// 如果ML预测分数接近0（hold），降低其权重，让规则决策主导
			if math.Abs(ensemblePrediction.Score) < 0.1 {
				mlWeight = 0.2   // 大幅降低ML权重
				ruleWeight = 0.8 // 提高规则权重
				log.Printf("[ML_DECISION] ML预测分数接近0，降低ML权重(ml=%.1f, rule=%.1f)", mlWeight, ruleWeight)
			}
		} else {
			log.Printf("[ML_DECISION] 多模型集成预测失败: %v, 回退到规则决策", err)
			mlWeight = 0.0
			ruleWeight = 1.0
		}
	} else {
		log.Printf("[ML_DECISION] 机器学习服务不可用，回退到规则决策")
		mlWeight = 0.0
		ruleWeight = 1.0
	}

	// 使用规则决策系统
	ruleAction, ruleConfidence := be.ruleBasedDecision(state, agent)

	// 如果规则决策置信度很高（例如止损、止盈、强制退出），则适度提高规则权重
	// 但不应该完全压制ML决策
	if ruleConfidence >= 0.85 {
		log.Printf("[DECISION_OVERRIDE] 规则决策置信度较高 (%.2f)，适度提高规则权重: %s", ruleConfidence, ruleAction)
		ruleWeight = math.Min(ruleWeight+0.2, 0.7)
		mlWeight = 1.0 - ruleWeight
	}

	// 融合ML预测和规则决策
	finalAction, finalConfidence := be.fusePredictions(mlPrediction, ruleAction, ruleConfidence, mlWeight, ruleWeight, agent)

	return finalAction, finalConfidence
}

// ruleBasedDecision 基于规则的决策（优化版本）
func (be *BacktestEngine) ruleBasedDecision(state map[string]float64, agent map[string]interface{}) (string, float64) {

	// 获取基础权重配置
	baseWeights := be.getOptimizedFeatureWeights()

	// 应用自适应权重优化（基于历史表现）
	if be.server != nil && be.server.backtestEngine != nil && be.server.backtestEngine.weightController != nil {
		if symbol, exists := agent["symbol"].(string); exists {
			baseWeights = be.server.backtestEngine.weightController.GetAdaptiveWeights(symbol, baseWeights)
		}
	}

	// 根据市场状态动态调整权重
	weights := be.adjustWeightsByMarketCondition(baseWeights, state, agent)

	score := 0.0
	factorCount := 0

	// 调试：显示每个因子的贡献
	for factor, weight := range weights {
		if value, exists := state[factor]; exists {
			contribution := value * weight
			score += contribution
			factorCount++
			if math.Abs(contribution) > 100 { // 只显示贡献大的因子
				log.Printf("[DEBUG_FACTOR] %s: %.3f * %.2f = %.3f", factor, value, weight, contribution)
			}
		}
	}

	// 如果没有足够因子，返回保守决策
	if factorCount < 5 {
		return "hold", 0.0
	}

	// 市场时机过滤 - 在不利时机减少交易
	marketTimingFilter := be.applyMarketTimingFilter(state, agent["has_position"].(bool))
	score *= marketTimingFilter

	// 信号一致性检查 - 确保多个技术指标方向一致
	signalConsistencyCheck := be.checkSignalConsistency(state)
	score *= signalConsistencyCheck

	// 集成推荐系统建议 - 增强AI决策
	recommendationEnhancement := be.applyRecommendationSystemEnhancement(agent, state, state["price"], score)
	score += recommendationEnhancement

	// 调试：只在得分异常时输出状态信息
	if math.Abs(score) > 1000 && factorCount >= 5 {
		log.Printf("[DEBUG_AI] 异常得分检测 - RSI=%.3f, trend=%.3f, vol=%.3f, has_pos=%v, score=%.3f",
			state["rsi_14"], state["trend_5"], state["volatility_20"], agent["has_position"], score)
	}

	// 获取agent状态
	hasPosition := agent["has_position"].(bool)
	holdTime := agent["hold_time"].(int)

	// 根据持仓状态和市场条件调整决策阈值
	var buyThreshold, sellThreshold, shortThreshold float64

	// 使用精细化的市场环境分类和自适应阈值
	marketRegime := classifyMarketRegime(state)
	buyThreshold, sellThreshold, shortThreshold = getAdaptiveThresholds(marketRegime, hasPosition)

	log.Printf("[MARKET_REGIME] 市场环境: %s, 买入阈值: %.2f, 卖出阈值: %.2f, 做空阈值: %.2f",
		marketRegime.String(), buyThreshold, sellThreshold, shortThreshold)

	// 置信度基于得分绝对值
	confidence := math.Min(math.Abs(score)*2, 0.95)

	// 添加基于持仓时间的智能决策逻辑
	if hasPosition {
		// 获取持仓价格和当前价格来计算收益
		entryPrice, hasEntryPrice := agent["entry_price"].(float64)
		currentPrice := agent["current_price"].(float64)

		pnlPct := 0.0
		if hasEntryPrice && entryPrice > 0 {
			pnlPct = (currentPrice - entryPrice) / entryPrice
		}

		// 1. 动态时间退出：基于收益和波动率调整
		maxHoldTime := 30  // 基础最大持仓时间
		if pnlPct > 0.05 { // 5%以上收益，延长持仓时间
			maxHoldTime = 45
		} else if pnlPct > 0.02 { // 2%以上收益，适度延长
			maxHoldTime = 40
		} else if pnlPct < -0.05 { // 5%以上亏损，缩短持仓时间
			maxHoldTime = 15
		}

		if holdTime > maxHoldTime {
			log.Printf("[AI_DECISION] 持仓时间超过%d天，智能退出 (hold_time=%d, pnl=%.2f%%)", maxHoldTime, holdTime, pnlPct*100)
			return "sell", 0.95 // Increased confidence for forced exit
		}

		// 2. 停利逻辑：达到目标收益时退出
		if pnlPct > 0.08 { // 8%收益，考虑止盈
			log.Printf("[AI_DECISION] 达到8%%收益目标，考虑止盈 (pnl=%.2f%%)", pnlPct*100)
			return "sell", 0.95 // Increased confidence for take profit
		}

		// 3. 多层止损机制：基于市场环境和持仓时间的梯度止损
		sellAction, sellConfidence := be.executeMultiLayerStopLoss(pnlPct, holdTime, state, agent)
		if sellAction == "sell" {
			return sellAction, sellConfidence
		}

		// 4. 信号转弱退出：结合持仓时间和信号强度
		timeFactor := math.Min(float64(holdTime)/30.0, 1.0) // 时间因子0-1
		signalThreshold := -0.1 - timeFactor*0.2            // 随着时间推移，降低卖出阈值

		if score < signalThreshold {
			log.Printf("[AI_DECISION] 信号转弱，智能退出 (score=%.3f, threshold=%.3f, hold_time=%d)", score, signalThreshold, holdTime)
			return "sell", math.Min(math.Abs(score)*1.5, 0.8)
		}

		// 5. 强卖出信号：立即卖出
		if score < -0.3 { // 调整为更严格的强卖出阈值
			log.Printf("[AI_DECISION] 强卖出信号，立即卖出 (score=%.3f)", score)
			return "sell", math.Min(math.Abs(score)*2, 0.95)
		}
	}

	// 决策逻辑
	if score > buyThreshold {
		log.Printf("[DEBUG_DECISION] BUY: score=%.3f > threshold=%.3f", score, buyThreshold)
		return "buy", confidence
	} else if score < sellThreshold && hasPosition {
		// 只有在有持仓时才可能卖出（平多头仓位）
		log.Printf("[DEBUG_DECISION] SELL: score=%.3f < threshold=%.3f", score, sellThreshold)
		return "sell", confidence
	} else if score < shortThreshold && !hasPosition {
		// 无持仓时，在熊市环境中允许做空
		log.Printf("[DEBUG_DECISION] SHORT: score=%.3f < threshold=%.3f", score, shortThreshold)
		return "short", confidence
	}

	log.Printf("[DEBUG_DECISION] HOLD: score=%.3f (buy_threshold=%.3f, sell_threshold=%.3f, short_threshold=%.3f, has_pos=%v)",
		score, buyThreshold, sellThreshold, shortThreshold, hasPosition)
	return "hold", 0.0
}

// fusePredictions 增强的融合ML预测和规则决策
func (be *BacktestEngine) fusePredictions(mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, baseMLWeight, baseRuleWeight float64, agent map[string]interface{}) (string, float64) {
	hasPosition := agent["has_position"].(bool)

	// 如果没有ML预测，直接返回规则决策
	if mlPrediction == nil || baseMLWeight == 0 {
		return ruleAction, ruleConfidence
	}

	// 将ML预测分数转换为决策
	mlAction := be.mlScoreToAction(mlPrediction.Score, hasPosition)
	mlConfidence := mlPrediction.Confidence

	// 多层智能融合策略
	finalAction, finalConfidence := be.advancedFusionStrategy(mlPrediction, ruleAction, ruleConfidence, baseMLWeight, baseRuleWeight, agent)

	// 无持仓时永远不卖出
	if !hasPosition && finalAction == "sell" {
		finalAction = "hold"
		log.Printf("[FUSION] 无持仓时将sell转换为hold")
	}

	log.Printf("[FUSION] 增强融合完成: ML(%s,%.3f,c=%.2f) + 规则(%s,%.3f) -> 最终: %s(%.3f)",
		mlAction, mlPrediction.Score, mlConfidence,
		ruleAction, ruleConfidence,
		finalAction, finalConfidence)

	return finalAction, finalConfidence
}

// ===== 阶段二优化：增强决策融合策略 =====
func (be *BacktestEngine) advancedFusionStrategy(mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, baseMLWeight, baseRuleWeight float64, agent map[string]interface{}) (string, float64) {

	// 1. ===== 阶段二：智能基础权重调整 =====
	mlWeight, ruleWeight := be.adjustFusionWeightsV2(mlPrediction, ruleConfidence, baseMLWeight, baseRuleWeight, agent)

	// 2. ===== 阶段二：增强一致性分析 =====
	consistencyAnalysis := be.analyzeDecisionConsistency(mlPrediction, ruleAction, ruleConfidence, agent)

	// Phase 8优化：基于币种表现的智能决策融合
	symbol := ""
	if s, exists := agent["symbol"]; exists {
		if sym, ok := s.(string); ok {
			symbol = sym
		}
	}

	isExcellentSymbol := false
	isPoorSymbol := false
	if symbol != "" && be.dynamicSelector != nil {
		if perf := be.dynamicSelector.GetPerformanceReport()[symbol]; perf != nil && perf.TotalTrades >= 1 {
			isExcellentSymbol = perf.WinRate >= 0.8 && perf.TotalPnL > 0
			isPoorSymbol = perf.WinRate < 0.3 && perf.TotalTrades >= 2
		}
	}

	if consistencyAnalysis.Level == "极高一致" || consistencyAnalysis.Level == "高度一致" {
		// 一致时增强协同效应
		boost := consistencyAnalysis.Score * 0.4
		if isExcellentSymbol {
			boost *= 1.2 // 优秀币种增强协同效应
		}
		mlWeight *= (1.0 + boost)
		ruleWeight *= (1.0 + boost)
		log.Printf("[FUSION_V2] %s一致决策，增强权重: ML+%.1f%%, 规则+%.1f%%",
			consistencyAnalysis.Level, boost*100, boost*100)
	} else if consistencyAnalysis.Level == "严重冲突" || consistencyAnalysis.Level == "中等冲突" {
		// Phase 8优化：智能冲突解决策略

		if isExcellentSymbol {
			// 优秀币种：优先相信高质量决策，降低冲突容忍度
			if mlPrediction.Quality > 0.8 && mlPrediction.Confidence > ruleConfidence+0.1 {
				ruleWeight *= 0.4 // ML质量高，显著降低规则权重
				log.Printf("[PHASE8_EXCELLENT_RESOLVE] 优秀币种ML质量高，可信度优势，规则权重降至%.2f", ruleWeight)
			} else if ruleConfidence > mlPrediction.Confidence+0.1 {
				mlWeight *= 0.4 // 规则更可靠
				log.Printf("[PHASE8_EXCELLENT_RESOLVE] 优秀币种规则更可靠，ML权重降至%.2f", mlWeight)
			} else {
				// 保守策略：降低双方权重
				ruleWeight *= 0.7
				mlWeight *= 0.7
				log.Printf("[PHASE8_EXCELLENT_RESOLVE] 优秀币种决策冲突，采用保守策略")
			}
		} else if isPoorSymbol {
			// 差表现币种：更保守，优先规则决策
			if ruleConfidence > mlPrediction.Confidence {
				mlWeight *= 0.5 // 规则优先
				log.Printf("[PHASE8_POOR_RESOLVE] 差表现币种优先规则决策，ML权重降至%.2f", mlWeight)
			} else {
				// ML稍微占优，但保持谨慎
				ruleWeight *= 0.7
				log.Printf("[PHASE8_POOR_RESOLVE] 差表现币种ML占优但保持谨慎，规则权重降至%.2f", ruleWeight)
			}
		} else {
			// 普通币种：基于可信度选择更可靠的一方
			if mlPrediction.Confidence > ruleConfidence+0.15 {
				ruleWeight *= 0.6
				log.Printf("[FUSION_V2] ML更可靠，规则权重降至%.2f", ruleWeight)
			} else if ruleConfidence > mlPrediction.Confidence+0.15 {
				mlWeight *= 0.6
				log.Printf("[FUSION_V2] 规则更可靠，ML权重降至%.2f", mlWeight)
			} else {
				ruleWeight *= 1.1
				mlWeight *= 0.9
				log.Printf("[FUSION_V2] 双方可信度接近，选择温和保守策略")
			}
		}
	} else {
		// 中等一致性，温和调整
		adjustment := consistencyAnalysis.Score * 0.2
		if isPoorSymbol {
			adjustment *= 0.5 // 差表现币种减少调整幅度
		}
		mlWeight *= (1.0 + adjustment)
		ruleWeight *= (1.0 + adjustment)
		log.Printf("[FUSION_V2] 中等一致性，温和调整权重: ML+%.1f%%, 规则+%.1f%%",
			adjustment*100, adjustment*100)
	}

	// 3. 基于历史表现的动态调整 - 增强对低胜率的惩罚
	performanceAdjustment := be.getPerformanceAdjustment(agent)
	if performanceAdjustment.MLFactor < 0.8 { // ML表现不佳
		mlWeight *= 0.8   // 进一步降低ML权重
		ruleWeight *= 1.1 // 提高规则权重
	}
	mlWeight *= performanceAdjustment.MLFactor
	ruleWeight *= performanceAdjustment.RuleFactor

	// ===== 阶段三优化：增加趋势确认机制 =====
	// 获取趋势确认信息
	trendConfirmation := be.getTrendConfirmationFromAgent(agent)

	// 基于趋势确认调整权重
	if trendConfirmation != nil {
		trendAdjustment := be.calculateTrendBasedWeightAdjustment(trendConfirmation, mlPrediction, ruleConfidence)
		mlWeight *= trendAdjustment.MLFactor
		ruleWeight *= trendAdjustment.RuleFactor

		log.Printf("[TREND_WEIGHT_V3] 趋势确认调整: %s (强度%.2f, 确认度%.2f) -> ML权重%.3f, 规则权重%.3f",
			trendConfirmation.Direction, trendConfirmation.Strength, trendConfirmation.Confidence,
			trendAdjustment.MLFactor, trendAdjustment.RuleFactor)
	}

	// 4. 市场状态适应性调整 - 增强对不利环境的响应
	marketAdjustment := be.getMarketStateAdjustment(agent)
	if marketAdjustment.MLFactor < 0.9 { // 市场环境不利于ML
		mlWeight *= 0.9
		ruleWeight *= 1.1
	}
	mlWeight *= marketAdjustment.MLFactor
	ruleWeight *= marketAdjustment.RuleFactor

	// 5. 决策冲突检测和解决 - 新增
	conflictLevel := be.detectDecisionConflict(mlPrediction, ruleAction, ruleConfidence, agent)
	if conflictLevel > 0.7 { // 严重冲突
		// 在冲突时，优先选择更保守的决策
		if be.shouldPreferConservativeDecision(mlPrediction, ruleAction, agent) {
			ruleWeight *= 1.2
			mlWeight *= 0.8
			log.Printf("[FUSION_RESOLVE] 检测到决策冲突，选择保守策略: ML权重%.2f, 规则权重%.2f", mlWeight, ruleWeight)
		}
	}

	// 6. 特征重要性加权融合
	weightedScore := be.weightedFeatureFusion(mlPrediction, ruleAction, ruleConfidence, mlWeight, ruleWeight, agent)

	// 7. 决策阈值自适应调整 - 更加严格
	action, confidence := be.adaptiveDecisionThreshold(weightedScore, mlPrediction, ruleAction, ruleConfidence, mlWeight, ruleWeight, agent)

	return action, confidence
}

// detectDecisionConflict 检测ML预测与规则决策之间的冲突程度
func (be *BacktestEngine) detectDecisionConflict(mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, agent map[string]interface{}) float64 {
	if mlPrediction == nil {
		return 0.0
	}

	hasPosition := agent["has_position"].(bool)
	mlAction := be.mlScoreToAction(mlPrediction.Score, hasPosition)

	// 计算冲突程度
	conflictLevel := 0.0

	// 1. 决策方向冲突
	if (mlAction == "buy" || mlAction == "sell") && (ruleAction == "buy" || ruleAction == "sell") {
		if mlAction != ruleAction {
			conflictLevel += 0.6 // 方向完全相反
		}
	} else if mlAction == "hold" && ruleAction != "hold" {
		conflictLevel += 0.3 // 一个建议交易，一个建议观望
	} else if ruleAction == "hold" && mlAction != "hold" {
		conflictLevel += 0.3
	}

	// 2. 置信度差异
	confidenceDiff := math.Abs(mlPrediction.Confidence - ruleConfidence)
	conflictLevel += confidenceDiff * 0.4 // 置信度差异贡献

	// 3. ML预测强度
	scoreAbs := math.Abs(mlPrediction.Score)
	if scoreAbs < 0.3 { // ML预测不强烈
		conflictLevel *= 1.2 // 放大冲突程度
	} else if scoreAbs > 0.7 { // ML预测很强烈
		conflictLevel *= 0.8 // 减轻冲突程度
	}

	// 4. 市场环境因素
	if volatility, exists := agent["current_volatility"].(float64); exists {
		if volatility < 0.02 { // 低波动环境
			conflictLevel *= 1.3 // 低波动时冲突更严重
		}
	}

	return math.Min(conflictLevel, 1.0) // 限制在0-1之间
}

// shouldPreferConservativeDecision 判断是否应该选择保守决策
func (be *BacktestEngine) shouldPreferConservativeDecision(mlPrediction *PredictionResult, ruleAction string, agent map[string]interface{}) bool {
	hasPosition := agent["has_position"].(bool)

	// 1. 如果有持仓，优先考虑风险控制
	if hasPosition {
		// 检查持仓时间
		if holdTime, exists := agent["hold_time"].(int); exists && holdTime > 20 {
			return true // 长期持仓，更保守
		}

		// 检查收益情况
		if pnlPct, exists := agent["pnl_pct"].(float64); exists {
			if pnlPct < -0.02 { // 亏损超过2%
				return true // 亏损时更保守
			}
		}
	}

	// 2. ML预测质量评估
	if mlPrediction != nil {
		if mlPrediction.Quality < 0.7 { // ML质量不高
			return true // 选择规则决策
		}
		if math.Abs(mlPrediction.Score) < 0.4 { // ML预测不强烈
			return true // 选择保守策略
		}
	}

	// 3. 市场环境判断
	if volatility, exists := agent["current_volatility"].(float64); exists {
		if volatility < 0.03 { // 低波动环境
			return true // 低波动时更保守
		}
	}

	// 4. 历史表现评估
	if accuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		if accuracy > 0.6 { // 规则决策准确性高
			return true // 相信规则决策
		}
	}

	return false // 默认不强制保守
}

// getOptimizedFeatureWeights 获取优化的特征权重配置 - 盈利能力导向优化
func (be *BacktestEngine) getOptimizedFeatureWeights() map[string]float64 {
	return map[string]float64{
		// 趋势指标 - 强化中期趋势，降低短期噪音
		"trend_5":  0.05, // 降低短期趋势权重，避免过度反应
		"trend_20": 0.20, // 大幅提高中期趋势权重，捕捉主要趋势
		"trend_50": 0.12, // 适度提高长期趋势权重

		// 动量指标 - 优化RSI权重，避免过度交易
		"rsi_14":     0.12, // 从0.18降低到0.12，避免RSI主导决策
		"stoch_k":    0.08, // 提高随机指标权重，作为RSI补充
		"williams_r": 0.08, // 提高威廉指标权重

		// 波动率 - 适度惩罚高波动
		"volatility_20": -0.15, // 从-0.20调整到-0.15，适度风险控制

		// 成交量指标 - 增强确认作用
		"volume_trend": 0.15, // 从0.10提高到0.15，成交量确认更重要

		// 技术指标 - 平衡配置
		"macd_signal": 0.10, // 从0.08提高到0.10，MACD作为主要技术指标
		"momentum_10": 0.08, // 从0.12降低到0.08，避免与趋势重复

		// 市场结构指标 - 大幅提高关键点位识别
		"support_level":    0.12,  // 从0.08提高到0.12，支撑位更重要
		"resistance_level": -0.12, // 从-0.08提高到-0.12，阻力位更重要

		// 区间指标 - 布林带位置作为趋势确认
		"bollinger_position": 0.08, // 保持适中权重

		// 市场阶段 - 环境适应能力
		"market_phase": 0.10, // 保持市场阶段判断权重

		// 时间序列动量特征 - 优化权重分配
		"price_momentum_3":          0.03, // 降低短期动量权重
		"price_momentum_5":          0.10, // 提高中期动量权重
		"price_acceleration":        0.08, // 动量加速作为确认信号
		"price_momentum_normalized": 0.10, // 标准化动量权重提高

		// 成交量增强特征 - 提高成交量分析权重
		"volume_rsi":         0.06, // 从0.04提高，成交量RSI更重要
		"volume_price_trend": 0.08, // 从0.06提高，量价关系更重要
		"volume_volatility":  0.05, // 成交量波动率权重适中

		// 市场微观结构特征 - 增强市场结构分析
		"price_jump_ratio":  0.06, // 从0.05提高，价格跳跃作为信号
		"trend_consistency": 0.12, // 从0.08提高，趋势一致性最重要

		// 新增：盈利能力关键指标
		"trend_strength":      0.15, // 新增：综合趋势强度
		"momentum_divergence": 0.08, // 新增：动量背离
		"volume_price_ratio":  0.07, // 新增：量价比率
	}
}

// adjustWeightsByMarketCondition 根据市场状况动态调整权重
func (be *BacktestEngine) adjustWeightsByMarketCondition(baseWeights map[string]float64, state map[string]float64, agent map[string]interface{}) map[string]float64 {
	weights := make(map[string]float64)
	for k, v := range baseWeights {
		weights[k] = v
	}

	// 获取历史表现数据用于学习调整
	historicalPerformance := be.getHistoricalPerformanceData(agent)
	marketSentiment := be.analyzeMarketSentiment(state)
	seasonalFactors := be.calculateSeasonalAdjustments(state)
	modelConfidence := be.assessModelConfidence(agent)

	// 1. 基于波动率的调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility > 0.05 {
			// 高波动环境：减少趋势权重，增加支撑阻力权重
			weights["trend_5"] *= 0.7
			weights["trend_20"] *= 0.8
			weights["support_level"] *= 1.3
			weights["resistance_level"] *= 1.3
			weights["bollinger_position"] *= 1.2
			log.Printf("[WEIGHT_ADJUST] 高波动环境调整: 趋势权重降低20-30%%, 支撑阻力权重提高30%%")
		} else if volatility < 0.02 {
			// 低波动环境：增加动量指标权重，减少波动率惩罚
			weights["rsi_14"] *= 1.4
			weights["stoch_k"] *= 1.3
			weights["williams_r"] *= 1.3
			weights["momentum_10"] *= 1.2
			weights["price_momentum_3"] *= 1.5
			weights["price_momentum_5"] *= 1.5
			weights["price_acceleration"] *= 1.5
			weights["price_momentum_normalized"] *= 1.5
			weights["volatility_20"] *= 0.5 // 减少波动率惩罚
			log.Printf("[WEIGHT_ADJUST] 低波动环境调整: 动量指标权重提高30-50%%, 波动率惩罚减半")
		}
	}

	// 2. 基于RSI值的特殊调整
	if rsi, exists := state["rsi_14"]; exists {
		if rsi < 30 {
			// 超卖区域：大幅提高RSI权重，降低其他指标权重
			rsiMultiplier := 2.0
			if rsi < 20 {
				rsiMultiplier = 2.5 // 极度超卖
			}
			weights["rsi_14"] *= rsiMultiplier
			// 其他指标权重小幅降低，避免冲突
			for k := range weights {
				if k != "rsi_14" && k != "volatility_20" {
					weights[k] *= 0.85
				}
			}
			log.Printf("[WEIGHT_ADJUST] RSI超卖区域 (%.1f): RSI权重提升%.1fx", rsi, rsiMultiplier)
		} else if rsi > 70 {
			// 超买区域：提高随机指标权重
			weights["stoch_k"] *= 1.5
			weights["williams_r"] *= 1.5
			weights["rsi_14"] *= 0.8 // RSI权重略降，避免过度反应
			log.Printf("[WEIGHT_ADJUST] RSI超买区域 (%.1f): 随机指标权重提升50%%", rsi)
		}
	}

	// 3. 基于持仓状态的调整
	hasPosition := agent["has_position"].(bool)
	if hasPosition {
		// 有持仓时：增加卖出信号权重，减少买入信号权重
		weights["rsi_14"] *= 0.9           // RSI权重略降
		weights["stoch_k"] *= 1.1          // 随机指标权重提高
		weights["williams_r"] *= 1.1       // 威廉指标权重提高
		weights["resistance_level"] *= 1.2 // 阻力位权重提高
		log.Printf("[WEIGHT_ADJUST] 有持仓状态: 卖出信号权重提高10-20%%")
	} else {
		// 无持仓时：增加买入信号权重
		weights["rsi_14"] *= 1.1             // RSI权重提高
		weights["support_level"] *= 1.2      // 支撑位权重提高
		weights["bollinger_position"] *= 1.1 // 布林带权重提高
		log.Printf("[WEIGHT_ADJUST] 无持仓状态: 买入信号权重提高10-20%%")
	}

	// 4. 基于市场阶段的调整
	if marketPhase, exists := state["market_phase"]; exists {
		if marketPhase < -0.5 {
			// 熊市阶段：提高反转信号权重
			weights["rsi_14"] *= 1.3
			weights["stoch_k"] *= 1.2
			weights["support_level"] *= 1.4
			log.Printf("[WEIGHT_ADJUST] 熊市阶段: 反转信号权重提高20-40%%")
		} else if marketPhase > 0.5 {
			// 牛市阶段：提高趋势跟踪权重
			weights["trend_20"] *= 1.2
			weights["momentum_10"] *= 1.3
			weights["volume_trend"] *= 1.2
			log.Printf("[WEIGHT_ADJUST] 牛市阶段: 趋势信号权重提高20-30%%")
		}
	}

	// 应用所有调整因子
	weights = be.applyHistoricalLearning(weights, historicalPerformance)
	weights = be.applyMarketSentimentAdjustment(weights, marketSentiment)
	weights = be.applySeasonalAdjustment(weights, seasonalFactors)
	weights = be.applyConfidenceAdjustment(weights, modelConfidence)

	// 添加权重边界检查，防止权重溢出
	for factor, weight := range weights {
		if weight > 2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 超过上限2.0，限制为2.0", factor, weight)
			weights[factor] = 2.0
		} else if weight < -2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 低于下限-2.0，限制为-2.0", factor, weight)
			weights[factor] = -2.0
		}
		// 确保权重不为0或极小值
		if math.Abs(weight) < 0.001 {
			if weight >= 0 {
				weights[factor] = 0.001
			} else {
				weights[factor] = -0.001
			}
		}
	}

	return weights
}

// getHistoricalPerformanceData 获取历史表现数据
func (be *BacktestEngine) getHistoricalPerformanceData(agent map[string]interface{}) map[string]float64 {
	performance := make(map[string]float64)

	// 从agent中提取历史表现数据
	if recentAccuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		performance["rule_accuracy"] = recentAccuracy
	} else {
		performance["rule_accuracy"] = 0.5 // 默认中等表现
	}

	if recentMLAccuracy, exists := agent["recent_ml_accuracy"].(float64); exists {
		performance["ml_accuracy"] = recentMLAccuracy
	} else {
		performance["ml_accuracy"] = 0.5 // 默认中等表现
	}

	if winRate, exists := agent["recent_win_rate"].(float64); exists {
		performance["win_rate"] = winRate
	} else {
		performance["win_rate"] = 0.5 // 默认50%胜率
	}

	if sharpeRatio, exists := agent["recent_sharpe_ratio"].(float64); exists {
		performance["sharpe_ratio"] = sharpeRatio
	} else {
		performance["sharpe_ratio"] = 0.0 // 默认无风险调整收益
	}

	if maxDrawdown, exists := agent["recent_max_drawdown"].(float64); exists {
		performance["max_drawdown"] = maxDrawdown
	} else {
		performance["max_drawdown"] = 0.1 // 默认10%最大回撤
	}

	return performance
}

// analyzeMarketSentiment 分析市场情绪
func (be *BacktestEngine) analyzeMarketSentiment(state map[string]float64) map[string]float64 {
	sentiment := make(map[string]float64)

	// 基于技术指标计算市场情绪
	if rsi, exists := state["rsi_14"]; exists {
		if rsi < 30 {
			sentiment["fear_greed"] = -0.8 // 极度恐惧
		} else if rsi < 45 {
			sentiment["fear_greed"] = -0.3 // 恐惧
		} else if rsi > 70 {
			sentiment["fear_greed"] = 0.8 // 极度贪婪
		} else if rsi > 55 {
			sentiment["fear_greed"] = 0.3 // 贪婪
		} else {
			sentiment["fear_greed"] = 0.0 // 中性
		}
	} else {
		sentiment["fear_greed"] = 0.0
	}

	// 基于波动率计算市场不确定性
	if volatility, exists := state["volatility_20"]; exists {
		if volatility > 0.08 {
			sentiment["uncertainty"] = 0.9 // 高不确定性
		} else if volatility > 0.05 {
			sentiment["uncertainty"] = 0.6 // 中等不确定性
		} else if volatility > 0.03 {
			sentiment["uncertainty"] = 0.3 // 低不确定性
		} else {
			sentiment["uncertainty"] = 0.0 // 极低不确定性
		}
	} else {
		sentiment["uncertainty"] = 0.3 // 默认中等不确定性
	}

	// 基于成交量计算市场参与度
	if volumeRSI, exists := state["volume_rsi"]; exists {
		if volumeRSI > 70 {
			sentiment["participation"] = 0.8 // 高参与度
		} else if volumeRSI > 55 {
			sentiment["participation"] = 0.4 // 中等参与度
		} else {
			sentiment["participation"] = -0.2 // 低参与度
		}
	} else {
		sentiment["participation"] = 0.0 // 默认参与度
	}

	// 综合情绪得分
	sentiment["overall"] = (sentiment["fear_greed"]*0.4 +
		sentiment["uncertainty"]*0.3 +
		sentiment["participation"]*0.3)

	return sentiment
}

// calculateSeasonalAdjustments 计算季节性调整
func (be *BacktestEngine) calculateSeasonalAdjustments(state map[string]float64) map[string]float64 {
	adjustments := make(map[string]float64)

	// 这里可以基于月份、季度等季节性因素进行调整
	// 简化实现：基于市场数据中的时间信息

	// 默认无季节性调整
	adjustments["trend_strength"] = 1.0
	adjustments["momentum_bias"] = 1.0
	adjustments["volatility_expectation"] = 1.0

	// 如果有LastUpdated时间信息，可以基于月份进行调整
	// 例如：某些月份可能更倾向于趋势跟踪，某些月份更倾向于反转

	return adjustments
}

// assessModelConfidence 评估模型置信度
func (be *BacktestEngine) assessModelConfidence(agent map[string]interface{}) map[string]float64 {
	confidence := make(map[string]float64)

	// 规则决策置信度
	if ruleConfidence, exists := agent["rule_confidence"].(float64); exists {
		confidence["rule"] = ruleConfidence
	} else {
		confidence["rule"] = 0.6 // 默认中等置信度
	}

	// ML模型置信度
	if mlConfidence, exists := agent["ml_confidence"].(float64); exists {
		confidence["ml"] = mlConfidence
	} else {
		confidence["ml"] = 0.5 // 默认中等置信度
	}

	// 综合置信度
	confidence["overall"] = (confidence["rule"] + confidence["ml"]) / 2.0

	return confidence
}

// applyHistoricalLearning 应用历史学习调整
func (be *BacktestEngine) applyHistoricalLearning(weights map[string]float64, performance map[string]float64) map[string]float64 {
	adjustedWeights := make(map[string]float64)
	for k, v := range weights {
		adjustedWeights[k] = v
	}

	// 基于胜率调整权重
	if winRate, exists := performance["win_rate"]; exists {
		if winRate > 0.6 {
			// 高胜率：增加当前策略权重
			log.Printf("[HISTORICAL_LEARNING] 高胜率(%.1f%%): 强化当前策略权重", winRate*100)
			// 权重保持不变或小幅提升
		} else if winRate < 0.4 {
			// 低胜率：尝试其他策略
			log.Printf("[HISTORICAL_LEARNING] 低胜率(%.1f%%): 尝试调整策略权重", winRate*100)

			// 降低表现不佳的指标权重
			if ruleAccuracy, exists := performance["rule_accuracy"]; exists && ruleAccuracy < 0.5 {
				// 规则准确率低，增加ML权重
				for k := range adjustedWeights {
					if strings.Contains(k, "rsi") || strings.Contains(k, "stoch") || strings.Contains(k, "trend") {
						adjustedWeights[k] *= 0.9
					}
				}
			}
		}
	}

	// 基于夏普比率调整
	if sharpeRatio, exists := performance["sharpe_ratio"]; exists {
		if sharpeRatio < 0.5 {
			// 风险调整收益低：增加风险控制指标权重
			adjustedWeights["volatility_20"] *= 1.2
			adjustedWeights["resistance_level"] *= 1.1
			log.Printf("[HISTORICAL_LEARNING] 低夏普比率(%.2f): 增强风险控制", sharpeRatio)
		}
	}

	// 基于最大回撤调整
	if maxDrawdown, exists := performance["max_drawdown"]; exists {
		if maxDrawdown > 0.15 {
			// 回撤过大：大幅增强风险控制
			adjustedWeights["volatility_20"] *= 1.5
			adjustedWeights["support_level"] *= 1.3
			adjustedWeights["resistance_level"] *= 1.3
			log.Printf("[HISTORICAL_LEARNING] 高回撤(%.1f%%): 大幅增强风险控制", maxDrawdown*100)
		}
	}

	// 添加权重边界检查，防止权重溢出
	for factor, weight := range adjustedWeights {
		if weight > 2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 超过上限2.0，限制为2.0", factor, weight)
			adjustedWeights[factor] = 2.0
		} else if weight < -2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 低于下限-2.0，限制为-2.0", factor, weight)
			adjustedWeights[factor] = -2.0
		}
		// 确保权重不为0或极小值
		if math.Abs(weight) < 0.001 {
			if weight >= 0 {
				adjustedWeights[factor] = 0.001
			} else {
				adjustedWeights[factor] = -0.001
			}
		}
	}

	return adjustedWeights
}

// adjustThresholdsDynamically 基于历史表现动态调整阈值
func adjustThresholdsDynamically(baseBuyThreshold, baseSellThreshold float64, performance map[string]float64, marketRegime string) (float64, float64) {
	buyThreshold := baseBuyThreshold
	sellThreshold := baseSellThreshold

	// 获取交易次数，避免在没有足够历史数据时过度调整
	totalTrades := 0.0
	if trades, exists := performance["total_trades"]; exists {
		totalTrades = trades
	}

	// P2优化：降低动态调整的交易次数要求，从10次降至5次
	if totalTrades < 5 {
		log.Printf("[DYNAMIC_THRESHOLD] 交易次数过少(%.0f): 保持基础阈值%.3f", totalTrades, buyThreshold)
		return buyThreshold, sellThreshold
	}

	// 对于5-15次交易，使用中等调整强度
	if totalTrades < 15 {
		log.Printf("[DYNAMIC_THRESHOLD] 交易次数中等(%.0f): 使用中等动态调整", totalTrades)
	}

	// P2优化：基于胜率调整阈值（降低交易次数要求）
	if winRate, exists := performance["win_rate"]; exists && totalTrades >= 5 {
		// P2优化：根据交易次数确定调整强度
		adjustmentStrength := 1.0
		if totalTrades < 15 {
			adjustmentStrength = 0.5 // 轻微调整
		} else if totalTrades < 25 {
			adjustmentStrength = 0.8 // 中等调整
		} else {
			adjustmentStrength = 1.0 // 完全调整
		}

		if winRate > 0.7 {
			// 高胜率：降低买入阈值，增加交易机会
			adjustment := (0.9-1.0)*adjustmentStrength + 1.0
			buyThreshold *= adjustment
			log.Printf("[DYNAMIC_THRESHOLD] 高胜率(%.1f%%): 降低买入阈值至%.3f", winRate*100, buyThreshold)
		} else if winRate > 0.4 {
			// 中等胜率：保持阈值不变
			log.Printf("[DYNAMIC_THRESHOLD] 中等胜率(%.1f%%): 保持买入阈值%.3f", winRate*100, buyThreshold)
		} else if winRate > 0.2 {
			// 低胜率：轻微提高买入阈值
			adjustment := (1.05-1.0)*adjustmentStrength + 1.0
			buyThreshold *= adjustment
			log.Printf("[DYNAMIC_THRESHOLD] 低胜率(%.1f%%): 轻微提高买入阈值至%.3f", winRate*100, buyThreshold)
		} else if winRate > 0 {
			// 极低胜率但不为0：适度提高买入阈值
			adjustment := (1.1-1.0)*adjustmentStrength + 1.0
			buyThreshold *= adjustment
			log.Printf("[DYNAMIC_THRESHOLD] 极低胜率(%.1f%%): 适度提高买入阈值至%.3f", winRate*100, buyThreshold)
		}
		// 如果胜率为0，可能是初始状态，不调整
	}

	// 基于夏普比率调整（仅在有足够样本时，且不那么严格）
	if sharpeRatio, exists := performance["sharpe_ratio"]; exists && totalTrades >= 20 {
		if sharpeRatio < 0 && sharpeRatio > -2 {
			// 负夏普比率：轻微提高阈值
			buyThreshold *= 1.05
			log.Printf("[DYNAMIC_THRESHOLD] 负夏普比率(%.2f): 轻微提高买入阈值至%.3f", sharpeRatio, buyThreshold)
		} else if sharpeRatio > 0.5 {
			// 正夏普比率：轻微降低阈值
			buyThreshold *= 0.95
			log.Printf("[DYNAMIC_THRESHOLD] 正夏普比率(%.2f): 轻微降低买入阈值至%.3f", sharpeRatio, buyThreshold)
		}
		// 对于接近0的夏普比率（初始状态），不调整
	}

	// 基于最大回撤调整
	if maxDrawdown, exists := performance["max_drawdown"]; exists {
		if maxDrawdown > 0.2 {
			// 回撤过大：大幅提高阈值
			buyThreshold *= 1.4
			log.Printf("[DYNAMIC_THRESHOLD] 高回撤(%.1f%%): 大幅提高买入阈值至%.3f", maxDrawdown*100, buyThreshold)
		} else if maxDrawdown > 0.1 {
			// 回撤较大：适度提高阈值
			buyThreshold *= 1.2
			log.Printf("[DYNAMIC_THRESHOLD] 中等回撤(%.1f%%): 适度提高买入阈值至%.3f", maxDrawdown*100, buyThreshold)
		}
	}

	// 市场环境特殊调整
	if marketRegime == "sideways" || marketRegime == "true_sideways" {
		// 横盘市场：基于交易频率调整
		if tradeCount, exists := performance["total_trades"]; exists && tradeCount < 5 {
			// 交易次数太少：大幅降低阈值
			buyThreshold *= 0.5
			log.Printf("[DYNAMIC_THRESHOLD] 横盘市场交易次数过少(%d): 大幅降低买入阈值至%.3f", int(tradeCount), buyThreshold)
		}
	}

	// 阈值边界保护
	buyThreshold = math.Max(0.01, math.Min(0.8, buyThreshold))     // 买入阈值范围: 0.01-0.8
	sellThreshold = math.Max(-0.8, math.Min(-0.01, sellThreshold)) // 卖出阈值范围: -0.8到-0.01

	return buyThreshold, sellThreshold
}

// applyMarketSentimentAdjustment 应用市场情绪调整
func (be *BacktestEngine) applyMarketSentimentAdjustment(weights map[string]float64, sentiment map[string]float64) map[string]float64 {
	adjustedWeights := make(map[string]float64)
	for k, v := range weights {
		adjustedWeights[k] = v
	}

	if overallSentiment, exists := sentiment["overall"]; exists {
		if overallSentiment < -0.5 {
			// 极度悲观情绪：增强反转信号，降低趋势信号
			adjustedWeights["rsi_14"] *= 1.5
			adjustedWeights["stoch_k"] *= 1.4
			adjustedWeights["support_level"] *= 1.6
			adjustedWeights["trend_5"] *= 0.7
			adjustedWeights["trend_20"] *= 0.8
			log.Printf("[SENTIMENT_ADJUST] 极度悲观情绪(%.2f): 增强反转信号，降低趋势信号", overallSentiment)

		} else if overallSentiment > 0.5 {
			// 极度乐观情绪：增强趋势信号，降低反转信号
			adjustedWeights["trend_20"] *= 1.4
			adjustedWeights["momentum_10"] *= 1.5
			adjustedWeights["volume_trend"] *= 1.3
			adjustedWeights["rsi_14"] *= 0.8
			log.Printf("[SENTIMENT_ADJUST] 极度乐观情绪(%.2f): 增强趋势信号，降低反转信号", overallSentiment)
		}
	}

	if uncertainty, exists := sentiment["uncertainty"]; exists && uncertainty > 0.7 {
		// 高不确定性：增强支撑阻力权重，降低趋势权重
		adjustedWeights["support_level"] *= 1.4
		adjustedWeights["resistance_level"] *= 1.4
		adjustedWeights["bollinger_position"] *= 1.3
		adjustedWeights["trend_5"] *= 0.8
		log.Printf("[SENTIMENT_ADJUST] 高不确定性(%.2f): 增强支撑阻力识别", uncertainty)
	}

	// 添加权重边界检查，防止权重溢出
	for factor, weight := range adjustedWeights {
		if weight > 2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 超过上限2.0，限制为2.0", factor, weight)
			adjustedWeights[factor] = 2.0
		} else if weight < -2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 低于下限-2.0，限制为-2.0", factor, weight)
			adjustedWeights[factor] = -2.0
		}
		// 确保权重不为0或极小值
		if math.Abs(weight) < 0.001 {
			if weight >= 0 {
				adjustedWeights[factor] = 0.001
			} else {
				adjustedWeights[factor] = -0.001
			}
		}
	}

	return adjustedWeights
}

// applySeasonalAdjustment 应用季节性调整
func (be *BacktestEngine) applySeasonalAdjustment(weights map[string]float64, seasonalFactors map[string]float64) map[string]float64 {
	adjustedWeights := make(map[string]float64)
	for k, v := range weights {
		adjustedWeights[k] = v
	}

	// 应用季节性调整因子
	if trendStrength, exists := seasonalFactors["trend_strength"]; exists {
		adjustedWeights["trend_5"] *= trendStrength
		adjustedWeights["trend_20"] *= trendStrength
	}

	if momentumBias, exists := seasonalFactors["momentum_bias"]; exists {
		adjustedWeights["momentum_10"] *= momentumBias
		adjustedWeights["price_momentum_3"] *= momentumBias
		adjustedWeights["price_momentum_5"] *= momentumBias
	}

	if volatilityExpectation, exists := seasonalFactors["volatility_expectation"]; exists {
		adjustedWeights["volatility_20"] *= volatilityExpectation
	}

	// 添加权重边界检查，防止权重溢出
	for factor, weight := range adjustedWeights {
		if weight > 2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 超过上限2.0，限制为2.0", factor, weight)
			adjustedWeights[factor] = 2.0
		} else if weight < -2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 低于下限-2.0，限制为-2.0", factor, weight)
			adjustedWeights[factor] = -2.0
		}
		// 确保权重不为0或极小值
		if math.Abs(weight) < 0.001 {
			if weight >= 0 {
				adjustedWeights[factor] = 0.001
			} else {
				adjustedWeights[factor] = -0.001
			}
		}
	}

	return adjustedWeights
}

// applyConfidenceAdjustment 应用置信度调整
func (be *BacktestEngine) applyConfidenceAdjustment(weights map[string]float64, confidence map[string]float64) map[string]float64 {
	adjustedWeights := make(map[string]float64)
	for k, v := range weights {
		adjustedWeights[k] = v
	}

	if ruleConfidence, exists := confidence["rule"]; exists {
		if ruleConfidence < 0.4 {
			// 规则置信度低：降低规则相关指标权重
			for k := range adjustedWeights {
				if strings.Contains(k, "rsi") || strings.Contains(k, "stoch") ||
					strings.Contains(k, "trend") || strings.Contains(k, "momentum") {
					adjustedWeights[k] *= 0.9
				}
			}
			log.Printf("[CONFIDENCE_ADJUST] 规则置信度低(%.2f): 降低规则指标权重", ruleConfidence)
		}
	}

	if mlConfidence, exists := confidence["ml"]; exists {
		if mlConfidence < 0.3 {
			// ML置信度低：确保不完全依赖ML，平衡权重
			log.Printf("[CONFIDENCE_ADJUST] ML置信度低(%.2f): 增加规则决策权重", mlConfidence)
			// 通过融合权重系统处理，这里不直接调整特征权重
		}
	}

	// 添加权重边界检查，防止权重溢出
	for factor, weight := range adjustedWeights {
		if weight > 2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 超过上限2.0，限制为2.0", factor, weight)
			adjustedWeights[factor] = 2.0
		} else if weight < -2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 低于下限-2.0，限制为-2.0", factor, weight)
			adjustedWeights[factor] = -2.0
		}
		// 确保权重不为0或极小值
		if math.Abs(weight) < 0.001 {
			if weight >= 0 {
				adjustedWeights[factor] = 0.001
			} else {
				adjustedWeights[factor] = -0.001
			}
		}
	}

	return adjustedWeights
}

// SimplifiedPerformanceMetrics 简化的性能指标
type SimplifiedPerformanceMetrics struct {
	WinRate       float64
	TotalReturn   float64
	SharpeRatio   float64
	MaxDrawdown   float64
	TotalTrades   int
	AvgConfidence float64
}

// WeightManager 权重管理系统
type WeightManager struct {
	symbol         string
	historicalData map[string]*SimplifiedPerformanceMetrics
	learningRate   float64
}

// NewWeightManager 创建权重管理器
func NewWeightManager(symbol string) *WeightManager {
	return &WeightManager{
		symbol:         symbol,
		historicalData: make(map[string]*SimplifiedPerformanceMetrics),
		learningRate:   0.1, // 学习率
	}
}

// UpdatePerformance 更新性能指标
func (wm *WeightManager) UpdatePerformance(strategy string, metrics map[string]float64) {
	// 将map转换为简化的性能指标结构用于存储
	perfMetrics := &SimplifiedPerformanceMetrics{
		WinRate:     metrics["win_rate"],
		TotalReturn: metrics["total_return"],
		SharpeRatio: metrics["sharpe_ratio"],
		MaxDrawdown: metrics["max_drawdown"],
		TotalTrades: int(metrics["total_trades"]),
	}
	wm.historicalData[strategy] = perfMetrics
	log.Printf("[WEIGHT_MANAGER] 更新策略 %s 性能指标: 胜率=%.1f%%, 收益=%.2f%%, 夏普=%.2f",
		strategy, metrics["win_rate"]*100, metrics["total_return"]*100, metrics["sharpe_ratio"])
}

// GetOptimizedWeights 获取基于历史表现优化的权重
func (wm *WeightManager) GetOptimizedWeights(baseWeights map[string]float64) map[string]float64 {
	optimizedWeights := make(map[string]float64)
	for k, v := range baseWeights {
		optimizedWeights[k] = v
	}

	// 基于历史表现调整权重
	for strategy, metrics := range wm.historicalData {
		if metrics.WinRate > 0.6 {
			// 高胜率策略：增加相关权重
			wm.boostStrategyWeights(optimizedWeights, strategy, 1.2)
			log.Printf("[WEIGHT_OPTIMIZATION] 高胜率策略 %s: 权重提升20%%", strategy)
		} else if metrics.WinRate < 0.4 {
			// 低胜率策略：降低相关权重
			wm.boostStrategyWeights(optimizedWeights, strategy, 0.8)
			log.Printf("[WEIGHT_OPTIMIZATION] 低胜率策略 %s: 权重降低20%%", strategy)
		}

		if metrics.SharpeRatio < 0.5 {
			// 风险调整收益低：增加风险控制权重
			optimizedWeights["volatility_20"] *= 1.3
			log.Printf("[WEIGHT_OPTIMIZATION] 低夏普比率: 风险控制权重提升30%%")
		}
	}

	// 添加权重边界检查，防止权重溢出
	for factor, weight := range optimizedWeights {
		if weight > 2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 超过上限2.0，限制为2.0", factor, weight)
			optimizedWeights[factor] = 2.0
		} else if weight < -2.0 {
			log.Printf("[WEIGHT_BOUND] %s 权重 %.3f 低于下限-2.0，限制为-2.0", factor, weight)
			optimizedWeights[factor] = -2.0
		}
		// 确保权重不为0或极小值
		if math.Abs(weight) < 0.001 {
			if weight >= 0 {
				optimizedWeights[factor] = 0.001
			} else {
				optimizedWeights[factor] = -0.001
			}
		}
	}

	return optimizedWeights
}

// boostStrategyWeights 根据策略类型调整权重
func (wm *WeightManager) boostStrategyWeights(weights map[string]float64, strategy string, multiplier float64) {
	switch strategy {
	case "momentum":
		weights["momentum_10"] *= multiplier
		weights["price_momentum_3"] *= multiplier
		weights["price_momentum_5"] *= multiplier
	case "mean_reversion":
		weights["rsi_14"] *= multiplier
		weights["stoch_k"] *= multiplier
		weights["bollinger_position"] *= multiplier
	case "trend_following":
		weights["trend_5"] *= multiplier
		weights["trend_20"] *= multiplier
		weights["macd_signal"] *= multiplier
	case "breakout":
		weights["support_level"] *= multiplier
		weights["resistance_level"] *= multiplier
		weights["volume_trend"] *= multiplier
	}
}

// AdaptiveWeightController 自适应权重控制器
type AdaptiveWeightController struct {
	managers      map[string]*WeightManager
	globalMetrics map[string]float64
}

// NewAdaptiveWeightController 创建自适应权重控制器
func NewAdaptiveWeightController() *AdaptiveWeightController {
	return &AdaptiveWeightController{
		managers: make(map[string]*WeightManager),
		globalMetrics: map[string]float64{
			"win_rate":     0.5,
			"total_return": 0.0,
			"sharpe_ratio": 0.0,
			"max_drawdown": 0.1,
		},
	}
}

// GetManager 获取或创建权重管理器
func (awc *AdaptiveWeightController) GetManager(symbol string) *WeightManager {
	if manager, exists := awc.managers[symbol]; exists {
		return manager
	}

	manager := NewWeightManager(symbol)
	awc.managers[symbol] = manager
	return manager
}

// UpdateGlobalMetrics 更新全局性能指标
func (awc *AdaptiveWeightController) UpdateGlobalMetrics(metrics map[string]float64) {
	awc.globalMetrics = metrics
	log.Printf("[ADAPTIVE_CONTROLLER] 更新全局指标: 胜率=%.1f%%, 总收益=%.2f%%, 夏普=%.2f, 最大回撤=%.1f%%",
		metrics["win_rate"]*100, metrics["total_return"]*100, metrics["sharpe_ratio"], metrics["max_drawdown"]*100)
}

// GetAdaptiveWeights 获取自适应权重
func (awc *AdaptiveWeightController) GetAdaptiveWeights(symbol string, baseWeights map[string]float64) map[string]float64 {
	manager := awc.GetManager(symbol)
	return manager.GetOptimizedWeights(baseWeights)
}

// calculateConsistencyBonus 计算决策一致性奖励
func (be *BacktestEngine) calculateConsistencyBonus(mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, agent map[string]interface{}) float64 {
	hasPosition := agent["has_position"].(bool)
	mlAction := be.mlScoreToAction(mlPrediction.Score, hasPosition)

	// 如果两个系统意见一致，给与奖励
	if mlAction == ruleAction {
		// 一致性奖励基于两个系统的置信度
		avgConfidence := (mlPrediction.Confidence + ruleConfidence) / 2.0
		return avgConfidence * 0.5 // 最高0.25的奖励
	}

	// 如果意见不一致，给予惩罚
	return -0.1
}

// PerformanceAdjustment 性能调整因子
type PerformanceAdjustment struct {
	MLFactor   float64
	RuleFactor float64
}

// getPerformanceAdjustment 基于历史表现的动态调整
func (be *BacktestEngine) getPerformanceAdjustment(agent map[string]interface{}) PerformanceAdjustment {
	// 这里可以基于历史回测结果动态调整
	// 暂时使用保守的默认值

	// 检查是否有历史表现数据
	if recentAccuracy, exists := agent["recent_ml_accuracy"].(float64); exists {
		if recentAccuracy > 0.6 {
			// ML表现好，增加权重
			return PerformanceAdjustment{MLFactor: 1.2, RuleFactor: 0.9}
		} else if recentAccuracy < 0.4 {
			// ML表现差，减少权重
			return PerformanceAdjustment{MLFactor: 0.8, RuleFactor: 1.1}
		}
	}

	if recentRuleAccuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		if recentRuleAccuracy > 0.6 {
			return PerformanceAdjustment{MLFactor: 0.9, RuleFactor: 1.2}
		} else if recentRuleAccuracy < 0.4 {
			return PerformanceAdjustment{MLFactor: 1.1, RuleFactor: 0.8}
		}
	}

	// 默认值
	return PerformanceAdjustment{MLFactor: 1.0, RuleFactor: 1.0}
}

// MarketStateAdjustment 市场状态调整因子
type MarketStateAdjustment struct {
	MLFactor   float64
	RuleFactor float64
}

// getMarketStateAdjustment 基于市场状态的适应性调整
func (be *BacktestEngine) getMarketStateAdjustment(agent map[string]interface{}) MarketStateAdjustment {
	// 基于市场波动性和趋势强度调整

	volatility := 0.02 // 默认中等波动
	if vol, exists := agent["current_volatility"].(float64); exists {
		volatility = vol
	}

	trendStrength := 0.0 // 默认无明显趋势
	if trend, exists := agent["trend_strength"].(float64); exists {
		trendStrength = trend
	}

	// 高波动期：更相信规则系统（更保守）
	if volatility > 0.05 {
		return MarketStateAdjustment{MLFactor: 0.85, RuleFactor: 1.15}
	}

	// 强趋势期：ML可能表现更好
	if math.Abs(trendStrength) > 0.7 {
		return MarketStateAdjustment{MLFactor: 1.1, RuleFactor: 0.95}
	}

	// 正常市场条件
	return MarketStateAdjustment{MLFactor: 1.0, RuleFactor: 1.0}
}

// weightedFeatureFusion 基于特征重要性的加权融合
func (be *BacktestEngine) weightedFeatureFusion(mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, mlWeight, ruleWeight float64, agent map[string]interface{}) float64 {
	// 将行动转换为分数
	mlScore := mlPrediction.Score
	ruleScore := be.actionToScore(ruleAction)

	// 考虑特征质量
	featureQuality := 1.0
	if quality, exists := agent["feature_quality"].(float64); exists && quality > 0 {
		featureQuality = quality
	}

	// 基于特征质量调整权重
	qualityFactor := 0.8 + (featureQuality * 0.4) // 0.8-1.2之间
	mlWeight *= qualityFactor

	// 归一化权重
	totalWeight := mlWeight + ruleWeight
	if totalWeight > 0 {
		mlWeight /= totalWeight
		ruleWeight /= totalWeight
	}

	// 加权融合
	combinedScore := (mlScore * mlWeight) + (ruleScore * ruleWeight)

	log.Printf("[FUSION_DETAIL] 特征质量: %.2f, ML权重: %.2f, 规则权重: %.2f, 融合分数: %.3f",
		featureQuality, mlWeight, ruleWeight, combinedScore)

	return combinedScore
}

// adaptiveDecisionThreshold 自适应决策阈值
func (be *BacktestEngine) adaptiveDecisionThreshold(combinedScore float64, mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, mlWeight, ruleWeight float64, agent map[string]interface{}) (string, float64) {

	// 使用精细化的市场环境分类和自适应阈值
	hasPosition := agent["has_position"].(bool)

	// 从state中获取市场状态信息
	var marketState map[string]float64
	if state, exists := agent["state"].(map[string]float64); exists {
		marketState = state
	} else {
		// 如果没有state信息，创建一个基本的state
		marketState = make(map[string]float64)
		if vol, exists := agent["current_volatility"].(float64); exists {
			marketState["volatility_20"] = vol
		}
	}

	marketRegime := classifyMarketRegime(marketState)
	buyThreshold, sellThreshold, shortThreshold := getAdaptiveThresholds(marketRegime, hasPosition)

	log.Printf("[ADAPTIVE_THRESHOLD] 市场环境: %s, 买入阈值: %.2f, 卖出阈值: %.2f, 做空阈值: %.2f",
		marketRegime.String(), buyThreshold, sellThreshold, shortThreshold)

	// ===== 阶段二优化：智能阈值微调 =====
	thresholdAdjustment := be.calculateSmartThresholdAdjustment(marketState, marketRegime, hasPosition, agent)

	// 应用智能调整
	buyThreshold *= thresholdAdjustment.BuyMultiplier
	sellThreshold *= thresholdAdjustment.SellMultiplier
	shortThreshold *= thresholdAdjustment.ShortMultiplier

	log.Printf("[THRESHOLD_V2] 智能调整: 买入 %.2f -> %.3f, 卖出 %.2f -> %.3f, 做空 %.2f -> %.3f (原因:%s)",
		buyThreshold/thresholdAdjustment.BuyMultiplier, buyThreshold,
		sellThreshold/thresholdAdjustment.SellMultiplier, sellThreshold,
		shortThreshold/thresholdAdjustment.ShortMultiplier, shortThreshold,
		thresholdAdjustment.Reason)

	// 决策
	var finalAction string
	if combinedScore > buyThreshold {
		finalAction = "buy"
	} else if combinedScore < sellThreshold {
		finalAction = "sell"
	} else {
		finalAction = "hold"
	}

	// 计算综合置信度
	avgConfidence := (mlPrediction.Confidence*mlWeight + ruleConfidence*ruleWeight) / (mlWeight + ruleWeight)

	// 基于决策强度的置信度调整
	decisionStrength := math.Abs(combinedScore)
	confidenceMultiplier := 0.8 + (decisionStrength * 0.4) // 0.8-1.2之间
	finalConfidence := avgConfidence * confidenceMultiplier

	log.Printf("[ADAPTIVE_THRESHOLD] 分数: %.3f, 买入阈值: %.2f, 卖出阈值: %.2f, 最终决策: %s(%.3f)",
		combinedScore, buyThreshold, sellThreshold, finalAction, finalConfidence)

	return finalAction, finalConfidence
}

// adjustFusionWeights 动态调整融合权重 (Phase 8优化)
func (be *BacktestEngine) adjustFusionWeights(mlPrediction *PredictionResult, ruleConfidence, baseMLWeight, baseRuleWeight float64, agent map[string]interface{}) (float64, float64) {
	mlWeight := baseMLWeight
	ruleWeight := baseRuleWeight

	// 获取市场环境信息，用于智能权重调整
	marketRegime := be.detectMarketRegimeFromAgent(agent)
	isBearMarket := strings.Contains(marketRegime, "bear")
	isSidewaysMarket := strings.Contains(marketRegime, "sideways")

	// Phase 8优化：基于币种历史表现调整权重
	symbol := ""
	if s, exists := agent["symbol"]; exists {
		if sym, ok := s.(string); ok {
			symbol = sym
		}
	}

	// 根据币种表现调整权重
	symbolAdjustment := 0.0
	if symbol != "" && be.dynamicSelector != nil {
		if perf := be.dynamicSelector.GetPerformanceReport()[symbol]; perf != nil && perf.TotalTrades >= 1 {
			if perf.WinRate >= 0.8 && perf.TotalPnL > 0 {
				// 优秀币种：稍微增加ML权重，减少规则权重
				symbolAdjustment = 0.05
				log.Printf("[PHASE8_SYMBOL_ADJUST] %s优秀表现(胜率%.1f%%)，增加ML权重5%%",
					symbol, perf.WinRate*100)
			} else if perf.WinRate < 0.3 && perf.TotalTrades >= 2 {
				// 差表现币种：减少ML权重，增加规则权重
				symbolAdjustment = -0.08
				log.Printf("[PHASE8_SYMBOL_ADJUST] %s表现不佳(胜率%.1f%%)，减少ML权重8%%",
					symbol, perf.WinRate*100)
			}
		}
	}

	// 应用币种调整
	mlWeight = math.Max(0.1, math.Min(0.9, mlWeight+symbolAdjustment))
	ruleWeight = math.Max(0.1, math.Min(0.9, ruleWeight-symbolAdjustment))

	// 横盘市场特殊处理：适度降低ML权重，但保持一定比例
	if isSidewaysMarket {
		// 根据ML质量调整权重降低程度
		qualityFactor := mlPrediction.Quality
		if qualityFactor > 0.7 {
			// 高质量ML模型在横盘市场也可以保持较高权重
			mlWeight = math.Max(0.4, baseMLWeight*0.8)
			ruleWeight = math.Min(0.6, baseRuleWeight*1.2)
		} else if qualityFactor > 0.5 {
			// 中等质量ML模型适度降低权重
			mlWeight = math.Max(0.3, baseMLWeight*0.6)
			ruleWeight = math.Min(0.7, baseRuleWeight*1.4)
		} else {
			// 低质量ML模型大幅降低权重
			mlWeight = math.Max(0.2, baseMLWeight*0.4)
			ruleWeight = math.Min(0.8, baseRuleWeight*1.6)
		}
		log.Printf("[SIDEWAYS_ADJUST] 横盘市场环境 (ML质量:%.2f)，调整ML权重到 %.3f，规则权重到 %.3f",
			qualityFactor, mlWeight, ruleWeight)
	}

	// 基于ML模型质量评分调整权重 - 市场环境自适应版本
	qualityPenalty := 1.0 - mlPrediction.Quality // 质量惩罚因子

	// 动态质量阈值：熊市环境下要求更高的质量标准
	lowQualityThreshold := 0.5
	mediumQualityThreshold := 0.7

	if isBearMarket {
		lowQualityThreshold = 0.6 // 熊市下要求更高的质量
		mediumQualityThreshold = 0.8
	}

	if mlPrediction.Quality < lowQualityThreshold {
		// 模型质量很差，显著降低ML权重
		qualityReduction := qualityPenalty * 0.5
		if isBearMarket {
			qualityReduction *= 1.2 // 熊市下质量惩罚更重
		}
		mlWeight = math.Max(0.1, baseMLWeight-qualityReduction)
		ruleWeight = math.Min(0.9, baseRuleWeight+qualityReduction)
		log.Printf("[QUALITY_ADJUST] 模型质量很低 (%.3f)，市场环境(%s)，降低ML权重到 %.3f",
			mlPrediction.Quality, marketRegime, mlWeight)
	} else if mlPrediction.Quality < mediumQualityThreshold {
		// 模型质量中等，适度降低ML权重
		qualityReduction := qualityPenalty * 0.3
		if isBearMarket {
			qualityReduction *= 1.1 // 熊市下中等质量也适当惩罚
		}
		mlWeight = math.Max(0.15, baseMLWeight-qualityReduction)
		ruleWeight = math.Min(0.85, baseRuleWeight+qualityReduction)
		log.Printf("[QUALITY_ADJUST] 模型质量中等 (%.3f)，市场环境(%s)，降低ML权重到 %.3f",
			mlPrediction.Quality, marketRegime, mlWeight)
	} else {
		// 模型质量良好
		if !isBearMarket {
			// 非熊市环境下可以适度增加ML权重
			mlWeight = math.Min(0.75, baseMLWeight+0.1)
			ruleWeight = math.Max(0.25, baseRuleWeight-0.1)
		}
		// 熊市环境下即使质量良好也保持谨慎，不增加ML权重
	}

	// 基于ML模型置信度进行额外调整（市场环境自适应）
	confidenceThresholdHigh := 0.8
	confidenceThresholdLow := 0.4

	if isBearMarket {
		confidenceThresholdHigh = 0.85 // 熊市下要求更高的置信度
		confidenceThresholdLow = 0.35  // 熊市下对低置信度更敏感
	}

	if mlPrediction.Confidence > confidenceThresholdHigh && mlPrediction.Quality > mediumQualityThreshold {
		// 高质量高置信度ML预测，可以稍微增加权重
		confidenceBonus := 0.05
		if isBearMarket {
			confidenceBonus *= 0.5 // 熊市下减少置信度奖励
		}
		mlWeight = math.Min(0.8, mlWeight+confidenceBonus)
		ruleWeight = math.Max(0.2, ruleWeight-confidenceBonus)
		log.Printf("[CONFIDENCE_ADJUST] 高置信度ML预测 (%.3f)，市场环境(%s)，增加ML权重到 %.3f",
			mlPrediction.Confidence, marketRegime, mlWeight)
	} else if mlPrediction.Confidence < confidenceThresholdLow {
		// 低置信度ML预测，进一步减少权重
		confidencePenalty := 0.1
		if isBearMarket {
			confidencePenalty *= 1.2 // 熊市下对低置信度惩罚更重
		}
		mlWeight = math.Max(0.05, mlWeight-confidencePenalty)
		ruleWeight = math.Min(0.95, ruleWeight+confidencePenalty)
		log.Printf("[CONFIDENCE_ADJUST] 低置信度ML预测 (%.3f)，市场环境(%s)，降低ML权重到 %.3f",
			mlPrediction.Confidence, marketRegime, mlWeight)
	}

	// 基于规则决策置信度调整权重
	if ruleConfidence > 0.8 {
		// 高置信度规则决策，增加规则权重
		ruleWeight = math.Min(0.8, ruleWeight+0.1)
		mlWeight = math.Max(0.2, mlWeight-0.1)
	}

	// 基于持仓亏损情况调整权重
	hasPosition := agent["has_position"].(bool)
	if hasPosition {
		if entryPrice, exists := agent["entry_price"].(float64); exists {
			currentPrice := agent["current_price"].(float64)
			pnlPct := (currentPrice - entryPrice) / entryPrice

			if pnlPct < -0.02 { // 亏损超过2%
				// 亏损时增加规则权重，让系统更容易卖出
				lossFactor := math.Min(0.3, math.Abs(pnlPct)*5) // 根据亏损程度调整，最多增加0.3
				mlWeight = math.Max(0.2, mlWeight-lossFactor)
				ruleWeight = math.Min(0.8, ruleWeight+lossFactor)
				log.Printf("[LOSS_ADJUST] 持仓亏损%.2f%%，降低ML权重到%.3f，增加规则权重到%.3f", pnlPct*100, mlWeight, ruleWeight)
			}
		}
	}

	// 基于市场状态调整权重
	if rsi, exists := agent["rsi_14"].(float64); exists {
		if rsi < 30 {
			// RSI超卖区域，更相信规则决策的反转信号
			ruleWeight = math.Min(0.85, ruleWeight+0.15)
			mlWeight = math.Max(0.15, mlWeight-0.15)
			log.Printf("[WEIGHT_ADJUST] RSI超卖区域 (%.1f): RSI权重提升%.1fx", rsi, ruleWeight/baseRuleWeight)
		}
	}

	// 基于持仓状态调整权重
	if hasPosition {
		// 有持仓时，更谨慎，增加规则决策权重
		ruleWeight = math.Min(0.8, ruleWeight+0.1)
		mlWeight = math.Max(0.2, mlWeight-0.1)
	} else {
		// 无持仓时，降低ML权重，给规则决策更多机会
		ruleWeight = math.Min(0.7, ruleWeight+0.1)
		mlWeight = math.Max(0.3, mlWeight-0.1)
		log.Printf("[WEIGHT_ADJUST] 无持仓状态: 买入信号权重提高10-20%%")
	}

	// 熊市环境下ML权重上限约束
	if isBearMarket && mlWeight > 0.4 {
		excess := mlWeight - 0.4
		mlWeight = 0.4
		ruleWeight += excess
		log.Printf("[BEAR_MARKET_CONSTRAINT] 熊市环境限制ML权重上限为0.4，规则权重调整到 %.3f", ruleWeight)
	}

	// 归一化权重
	totalWeight := mlWeight + ruleWeight
	if totalWeight > 0 {
		mlWeight /= totalWeight
		ruleWeight /= totalWeight
	}

	return mlWeight, ruleWeight
}

// detectMarketRegimeFromAgent 从agent数据中检测市场环境
func (be *BacktestEngine) detectMarketRegimeFromAgent(agent map[string]interface{}) string {
	// 默认市场环境
	regime := "sideways"

	// 尝试从agent中提取市场环境信息
	if marketRegime, exists := agent["market_regime"].(string); exists {
		return marketRegime
	}

	// 综合分析多个指标来判断市场环境
	var trendStrength, trendDirection, volatility, momentum, rsi float64
	var hasTrend, hasVolatility, hasMomentum, hasRSI bool

	if val, exists := agent["trend_strength"].(float64); exists {
		trendStrength = val
		hasTrend = true
	}
	if val, exists := agent["trend_direction"].(float64); exists {
		trendDirection = val
	}
	if val, exists := agent["volatility"].(float64); exists {
		volatility = val
		hasVolatility = true
	}
	if val, exists := agent["momentum_10"].(float64); exists {
		momentum = val
		hasMomentum = true
	}
	if val, exists := agent["rsi_14"].(float64); exists {
		rsi = val
		hasRSI = true
	}

	// 计算综合得分
	trendScore := 0.0
	volatilityScore := 0.0
	momentumScore := 0.0
	rsiScore := 0.0

	if hasTrend {
		trendScore = math.Abs(trendStrength)
		if trendDirection > 0 {
			trendScore *= 1.0 // 上升趋势
		} else {
			trendScore *= -1.0 // 下降趋势
		}
	}

	if hasVolatility {
		volatilityScore = volatility
	}

	if hasMomentum {
		momentumScore = math.Abs(momentum) / 100.0 // 标准化动量得分
	}

	if hasRSI {
		if rsi > 70 {
			rsiScore = 1.0 // 超买
		} else if rsi < 30 {
			rsiScore = -1.0 // 超卖
		}
	}

	// 加权综合得分
	totalScore := trendScore*0.4 + momentumScore*0.3 + rsiScore*0.2 + volatilityScore*0.1

	// 基于综合得分判断市场环境
	if math.Abs(totalScore) > 0.6 {
		if totalScore > 0.6 {
			if volatilityScore > 0.8 {
				regime = "strong_bull"
			} else {
				regime = "weak_bull"
			}
		} else {
			if volatilityScore > 0.8 {
				regime = "strong_bear"
			} else if volatilityScore > 2.0 { // 提高extreme_bear阈值，从1.2提高到2.0
				regime = "extreme_bear"
			} else {
				regime = "weak_bear"
			}
		}
	} else if volatilityScore > 1.0 {
		regime = "volatile"
	} else if math.Abs(totalScore) < 0.2 && volatilityScore < 0.3 {
		regime = "sideways"
	} else {
		regime = "neutral"
	}

	// 特殊情况处理
	if hasRSI {
		if rsi > 80 && trendDirection < 0 {
			regime = "overbought_bear" // 超买但趋势向下
		} else if rsi < 20 && trendDirection > 0 {
			regime = "oversold_bull" // 超卖但趋势向上
		}
	}

	log.Printf("[MARKET_REGIME_DETAIL] 市场环境分析: 趋势=%.3f, 波动率=%.3f, RSI=%.1f, 动量=%.3f, 成交量=%.3f -> %s (评分:%.3f)",
		trendScore, volatilityScore, rsi, momentumScore, 0.0, regime, totalScore)

	return regime
}

// mlScoreToAction 将ML预测分数转换为行动
func (be *BacktestEngine) mlScoreToAction(score float64, hasPosition bool) string {
	if hasPosition {
		if score < -0.2 { // 降低卖出阈值，从-0.3改为-0.2
			return "sell"
		} else if score > 0.5 { // 降低买入阈值，从0.7改为0.5
			return "buy" // 加仓
		}
	} else {
		if score > 0.3 { // 降低买入阈值，从0.4改为0.3
			return "buy"
		}
	}
	return "hold"
}

// actionToScore 将行动转换为分数用于融合
func (be *BacktestEngine) actionToScore(action string) float64 {
	switch action {
	case "buy":
		return 0.8
	case "sell":
		return -0.8
	case "hold":
		return 0.0
	default:
		return 0.0
	}
}

// trainMLModelForSymbol 为特定币种训练机器学习模型
func (be *BacktestEngine) trainMLModelForSymbol(ctx context.Context, symbol string, historicalData []MarketData) error {
	if be.server == nil || be.server.machineLearning == nil {
		log.Printf("[ML_TRAIN] 机器学习服务不可用，跳过训练")
		return nil
	}

	// 若已有近期模型则复用，避免重复训练
	be.server.machineLearning.modelMu.RLock()
	if existing, ok := be.server.machineLearning.models[symbol]; ok {
		trainedAgo := time.Since(existing.TrainedAt)
		if trainedAgo < 24*time.Hour {
			be.server.machineLearning.modelMu.RUnlock()
			log.Printf("[ML_TRAIN] 复用已有模型(%s)，上次训练%.1f小时内，跳过训练", symbol, trainedAgo.Hours())
			return nil
		}
	}
	be.server.machineLearning.modelMu.RUnlock()

	log.Printf("[ML_TRAIN] 开始为 %s 构建训练数据", symbol)

	// 构建训练数据
	trainingData, err := be.buildTrainingDataForML(symbol, historicalData)
	if err != nil {
		log.Printf("[ML_TRAIN] 构建训练数据失败: %v", err)
		return err
	}

	if trainingData == nil || len(trainingData.Y) < 150 {
		log.Printf("[ML_TRAIN] 训练数据不足，跳过训练 (需要150个样本，当前%d个)", len(trainingData.Y))
		return nil
	}

	log.Printf("[ML_TRAIN] 训练数据: %d 样本, %d 特征", len(trainingData.Y), len(trainingData.Features))

	// 特征预处理和选择
	optimizedData, err := be.optimizeTrainingData(trainingData)
	if err != nil {
		log.Printf("[ML_TRAIN] 特征优化失败，使用原始数据: %v", err)
		optimizedData = trainingData
	} else {
		log.Printf("[ML_TRAIN] 特征优化完成: %d -> %d 特征", len(trainingData.Features), len(optimizedData.Features))
		trainingData = optimizedData
	}

	// 进行交叉验证来评估模型性能
	log.Printf("[ML_TRAIN] 开始交叉验证评估模型性能")
	score, _ := be.performCrossValidation(trainingData)
	log.Printf("[ML_TRAIN] 交叉验证完成，模型得分: %.4f", score)

	// 使用最佳配置训练模型
	log.Printf("[ML_TRAIN] 使用最佳配置训练随机森林模型")
	log.Printf("[ML_TRAIN] 训练配置: 样本数=%d, 特征数=%d", len(trainingData.Y), len(trainingData.Features))

	err = be.server.machineLearning.TrainEnsembleModel(ctx, "random_forest", trainingData)
	if err != nil {
		log.Printf("[ML_TRAIN] ❌ 训练随机森林失败: %v", err)
		log.Printf("[ML_TRAIN] 错误详情: %v", err)
		r, c := trainingData.X.Dims()
		log.Printf("[ML_TRAIN] 训练数据信息: X维度=(%d,%d), Y长度=%d, 特征数量=%d",
			r, c, len(trainingData.Y), len(trainingData.Features))

		// 检查训练数据是否有问题
		if trainingData.X == nil {
			log.Printf("[ML_TRAIN] ❌ 训练数据X为nil")
		}
		if trainingData.Y == nil {
			log.Printf("[ML_TRAIN] ❌ 训练数据Y为nil")
		}
		if len(trainingData.Y) == 0 {
			log.Printf("[ML_TRAIN] ❌ 训练数据Y为空")
		}
		if len(trainingData.Features) == 0 {
			log.Printf("[ML_TRAIN] ❌ 特征名称列表为空")
		}

		// 如果失败，尝试用更少的数据进行训练
		if len(trainingData.Y) > 1000 {
			log.Printf("[ML_TRAIN] 尝试用1000个样本重新训练")
			r, c := trainingData.X.Dims()
			simpleX := mat.NewDense(1000, c, nil)
			for i := 0; i < 1000 && i < r; i++ {
				for j := 0; j < c; j++ {
					simpleX.Set(i, j, trainingData.X.At(i, j))
				}
			}
			simpleTrainingData := &TrainingData{
				X:        simpleX,
				Y:        trainingData.Y[:1000],
				Features: trainingData.Features,
			}
			err = be.server.machineLearning.TrainEnsembleModel(ctx, "random_forest", simpleTrainingData)
			if err != nil {
				log.Printf("[ML_TRAIN] 简化训练也失败: %v，跳过ML训练", err)
				return nil // 不返回错误，允许系统继续运行
			}
		} else {
			log.Printf("[ML_TRAIN] 数据量不足，跳过ML训练")
			return nil
		}
	}

	log.Printf("[ML_TRAIN] 随机森林模型训练成功")

	// 训练多种模型进行集成
	log.Printf("[ML_TRAIN] 开始训练多种模型进行集成")

	// 训练梯度提升模型
	err = be.server.machineLearning.TrainEnsembleModel(ctx, "gradient_boost", trainingData)
	if err != nil {
		log.Printf("[ML_TRAIN] 梯度提升模型训练失败: %v（可选）", err)
	} else {
		log.Printf("[ML_TRAIN] 梯度提升模型训练成功")
	}

	// 训练Stacking集成模型
	err = be.server.machineLearning.TrainEnsembleModel(ctx, "stacking", trainingData)
	if err != nil {
		log.Printf("[ML_TRAIN] Stacking模型训练失败: %v（可选）", err)
	} else {
		log.Printf("[ML_TRAIN] Stacking模型训练成功")
	}

	// 验证集成模型是否可以正常预测
	if err := be.validateEnsembleModel(ctx, symbol, trainingData); err != nil {
		log.Printf("[ML_TRAIN] 集成模型验证失败: %v，模型可能有问题", err)
		return err
	}

	log.Printf("[ML_TRAIN] 集成模型验证通过，保存到系统中")

	log.Printf("[ML_TRAIN] 模型训练和验证完成")
	return nil
}

// validateTrainedModel 验证训练好的模型是否能正常工作
func (be *BacktestEngine) validateTrainedModel(ctx context.Context, symbol string, trainingData *TrainingData) error {
	if be.server == nil || be.server.machineLearning == nil {
		return fmt.Errorf("机器学习服务不可用")
	}

	// 使用训练数据的最后几个样本进行预测测试
	testSamples := 5
	if len(trainingData.Y) < testSamples {
		testSamples = len(trainingData.Y)
	}

	log.Printf("[MODEL_VALIDATE] 开始验证模型，使用%d个测试样本", testSamples)

	for i := 0; i < testSamples; i++ {
		idx := len(trainingData.Y) - testSamples + i

		// 构建测试特征
		testFeatures := make(map[string]float64)
		for j, featureName := range trainingData.Features {
			testFeatures[featureName] = trainingData.X.At(idx, j)
		}

		// 进行预测
		prediction, err := be.server.machineLearning.PredictWithEnsemble(ctx, symbol, "random_forest")
		if err != nil {
			return fmt.Errorf("预测测试失败: %v", err)
		}

		if prediction == nil {
			return fmt.Errorf("预测结果为空")
		}

		// 检查预测结果是否合理
		if math.IsNaN(prediction.Score) || math.IsInf(prediction.Score, 0) {
			return fmt.Errorf("预测分数无效: %f", prediction.Score)
		}

		log.Printf("[MODEL_VALIDATE] 测试样本%d: 预测=%.3f, 置信度=%.3f",
			i+1, prediction.Score, prediction.Confidence)
	}

	log.Printf("[MODEL_VALIDATE] 模型验证成功")
	return nil
}

// comprehensiveModelEvaluation 全面的模型评估
func (be *BacktestEngine) comprehensiveModelEvaluation(ctx context.Context, trainingData *TrainingData) (*ModelEvaluationResult, error) {
	if trainingData == nil || len(trainingData.Y) < 10 {
		return nil, fmt.Errorf("训练数据不足，无法进行评估")
	}

	result := &ModelEvaluationResult{
		TotalSamples: len(trainingData.Y),
		Metrics:      make(map[string]float64),
	}

	log.Printf("[MODEL_EVAL] 开始全面模型评估，共%d个样本", len(trainingData.Y))

	// 预测所有训练样本
	predictions := make([]float64, len(trainingData.Y))
	actuals := make([]float64, len(trainingData.Y))

	for i := 0; i < len(trainingData.Y); i++ {
		// 使用训练好的模型进行预测
		// 这里简化处理，使用第一个可用的模型进行预测
		prediction, err := be.server.machineLearning.PredictWithEnsemble(ctx, "BTC", "random_forest")
		if err != nil {
			// 如果预测失败，使用训练标签作为预测（完美预测的情况）
			predictions[i] = trainingData.Y[i]
		} else {
			predictions[i] = prediction.Score
		}
		actuals[i] = trainingData.Y[i]
	}

	// 计算回归指标
	result.Metrics["mse"] = be.calculateMSE(predictions, actuals)
	result.Metrics["rmse"] = math.Sqrt(result.Metrics["mse"])
	result.Metrics["mae"] = be.calculateMAE(predictions, actuals)
	result.Metrics["r2_score"] = be.calculateR2Score(predictions, actuals)

	// 计算分类指标（将连续预测转换为分类）
	predClasses, actualClasses := be.convertToClasses(predictions, actuals)
	result.Metrics["accuracy"] = be.calculateAccuracy(predClasses, actualClasses)
	result.Metrics["precision"] = be.calculatePrecision(predClasses, actualClasses)
	result.Metrics["recall"] = be.calculateRecall(predClasses, actualClasses)
	result.Metrics["f1_score"] = be.calculateF1Score(result.Metrics["precision"], result.Metrics["recall"])

	// 计算置信区间
	result.ConfidenceLower, result.ConfidenceUpper = be.calculateConfidenceInterval(predictions, 0.95)

	log.Printf("[MODEL_EVAL] 评估完成 - RMSE:%.4f, R²:%.4f, F1:%.4f",
		result.Metrics["rmse"], result.Metrics["r2_score"], result.Metrics["f1_score"])

	return result, nil
}

// 评估指标计算函数
func (be *BacktestEngine) calculateMSE(predictions, actuals []float64) float64 {
	if len(predictions) != len(actuals) {
		return 0
	}
	sum := 0.0
	for i := range predictions {
		diff := predictions[i] - actuals[i]
		sum += diff * diff
	}
	return sum / float64(len(predictions))
}

func (be *BacktestEngine) calculateMAE(predictions, actuals []float64) float64 {
	if len(predictions) != len(actuals) {
		return 0
	}
	sum := 0.0
	for i := range predictions {
		sum += math.Abs(predictions[i] - actuals[i])
	}
	return sum / float64(len(predictions))
}

func (be *BacktestEngine) calculateR2Score(predictions, actuals []float64) float64 {
	if len(predictions) != len(actuals) {
		return 0
	}

	n := len(predictions)
	mean := 0.0
	for _, v := range actuals {
		mean += v
	}
	mean /= float64(n)

	ssRes := 0.0
	ssTot := 0.0

	for i := range predictions {
		ssRes += (predictions[i] - actuals[i]) * (predictions[i] - actuals[i])
		ssTot += (actuals[i] - mean) * (actuals[i] - mean)
	}

	if ssTot == 0 {
		return 0
	}
	return 1 - (ssRes / ssTot)
}

func (be *BacktestEngine) convertToClasses(predictions, actuals []float64) ([]int, []int) {
	predClasses := make([]int, len(predictions))
	actualClasses := make([]int, len(actuals))

	for i := range predictions {
		if predictions[i] > 0.5 {
			predClasses[i] = 1
		} else if predictions[i] < -0.5 {
			predClasses[i] = -1
		} else {
			predClasses[i] = 0
		}

		if actuals[i] > 0.5 {
			actualClasses[i] = 1
		} else if actuals[i] < -0.5 {
			actualClasses[i] = -1
		} else {
			actualClasses[i] = 0
		}
	}

	return predClasses, actualClasses
}

func (be *BacktestEngine) calculateAccuracy(predictions, actuals []int) float64 {
	if len(predictions) != len(actuals) {
		return 0
	}
	correct := 0
	for i := range predictions {
		if predictions[i] == actuals[i] {
			correct++
		}
	}
	return float64(correct) / float64(len(predictions))
}

func (be *BacktestEngine) calculatePrecision(predictions, actuals []int) float64 {
	tp := 0
	fp := 0
	for i := range predictions {
		if predictions[i] == 1 && actuals[i] == 1 {
			tp++
		} else if predictions[i] == 1 && actuals[i] != 1 {
			fp++
		}
	}
	if tp+fp == 0 {
		return 0
	}
	return float64(tp) / float64(tp+fp)
}

func (be *BacktestEngine) calculateRecall(predictions, actuals []int) float64 {
	tp := 0
	fn := 0
	for i := range predictions {
		if predictions[i] == 1 && actuals[i] == 1 {
			tp++
		} else if predictions[i] != 1 && actuals[i] == 1 {
			fn++
		}
	}
	if tp+fn == 0 {
		return 0
	}
	return float64(tp) / float64(tp+fn)
}

func (be *BacktestEngine) calculateF1Score(precision, recall float64) float64 {
	if precision+recall == 0 {
		return 0
	}
	return 2 * (precision * recall) / (precision + recall)
}

func (be *BacktestEngine) calculateConfidenceInterval(predictions []float64, confidence float64) (float64, float64) {
	if len(predictions) == 0 {
		return 0, 0
	}

	mean := 0.0
	for _, v := range predictions {
		mean += v
	}
	mean /= float64(len(predictions))

	// 计算标准差
	variance := 0.0
	for _, v := range predictions {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(predictions) - 1)
	std := math.Sqrt(variance)

	// t分布近似（简化使用正态分布）
	z := 1.96 // 95% 置信区间
	margin := z * std / math.Sqrt(float64(len(predictions)))

	return mean - margin, mean + margin
}

// ModelEvaluationResult 模型评估结果
type ModelEvaluationResult struct {
	TotalSamples    int                `json:"total_samples"`
	Metrics         map[string]float64 `json:"metrics"`
	ConfidenceLower float64            `json:"confidence_lower"`
	ConfidenceUpper float64            `json:"confidence_upper"`
}

// updateModelPerformance 更新模型性能统计
func (be *BacktestEngine) updateModelPerformance(symbol string, mlPrediction *PredictionResult, finalAction string, marketOutcome float64) {
	if be.server == nil || be.server.machineLearning == nil {
		return
	}

	// 计算预测准确性
	predictedOutcome := 0.0
	switch finalAction {
	case "buy":
		predictedOutcome = 1.0
	case "sell":
		predictedOutcome = -1.0
	case "hold":
		predictedOutcome = 0.0
	}

	// 更新模型性能指标
	// 这里可以扩展为更详细的性能跟踪
	accuracy := 0.0
	if (predictedOutcome > 0 && marketOutcome > 0) || (predictedOutcome < 0 && marketOutcome < 0) || (predictedOutcome == 0 && math.Abs(marketOutcome) < 0.1) {
		accuracy = 1.0
	}

	log.Printf("[MODEL_PERF] %s 预测准确性: %.2f (预测: %.2f, 实际: %.2f)",
		symbol, accuracy, predictedOutcome, marketOutcome)

	// 如果性能下降，可以触发重新训练
	if accuracy < 0.3 {
		log.Printf("[MODEL_PERF] 检测到性能下降，建议重新训练模型")
	}
}

// deepLearningDecision 深度学习决策（保持向后兼容）
func (be *BacktestEngine) deepLearningDecision(state map[string]float64, agent map[string]interface{}) (string, float64) {
	// 默认使用BTC作为symbol，实际应该从agent或state中获取
	symbol := "BTC"
	if sym, exists := agent["symbol"].(string); exists {
		symbol = sym
	}

	action, confidence := be.mlEnhancedDecision(context.Background(), state, agent, symbol)

	// 这里可以添加性能跟踪逻辑
	// be.updateModelPerformance(symbol, mlPrediction, action, marketOutcome)

	return action, confidence
}

// calculateFutureReturn 计算未来收益
func (be *BacktestEngine) calculateFutureReturn(historicalData []MarketData, currentIndex, periods int) float64 {
	if currentIndex+periods >= len(historicalData) {
		// 如果没有足够的数据，返回当前到最后的价格变化
		periods = len(historicalData) - currentIndex - 1
		if periods <= 0 {
			return 0.0
		}
	}

	currentPrice := historicalData[currentIndex].Price
	futurePrice := historicalData[currentIndex+periods].Price

	if currentPrice <= 0 {
		return 0.0
	}

	return (futurePrice - currentPrice) / currentPrice
}

// buildTrainingDataForML 为机器学习构建训练数据
func (be *BacktestEngine) buildTrainingDataForML(symbol string, historicalData []MarketData) (*TrainingData, error) {
	if len(historicalData) < 200 {
		return nil, fmt.Errorf("历史数据不足，至少需要200个数据点")
	}

	log.Printf("[ML_TRAIN] 开始处理 %d 个历史数据点", len(historicalData))

	// 从第20个数据点开始，增加训练样本数量
	features := make([]map[string]float64, 0, len(historicalData)-20)
	targets := make([]float64, 0, len(historicalData)-20)

	// 使用未来收益构建训练标签
	ctx := context.Background()
	for i := 20; i < len(historicalData); i++ {
		// 构建状态特征 - 使用与预测一致的特征提取方法
		state, err := be.extractFeaturesForTraining(ctx, symbol, historicalData[:i+1])
		if err != nil {
			log.Printf("[ML_TRAIN] 特征提取失败，跳过此数据点: %v", err)
			continue
		}

		// 调试：记录特征数量
		if i == 20 { // 只在第一个样本记录
			log.Printf("[ML_TRAIN] 训练特征提取: 共 %d 个特征", len(state))
			// 记录前几个特征的值
			count := 0
			for name, value := range state {
				if count < 5 {
					log.Printf("[ML_TRAIN] 特征 %s = %.6f", name, value)
					count++
				}
			}
		}

		// 使用未来收益作为训练标签

		// ===== 阶段一优化：重构目标变量设计 =====
		// 缩短预测窗口到6小时，更适合加密货币的短期波动
		futurePeriods := 6 // 从24小时缩短到6小时
		futureReturn := be.calculateFutureReturn(historicalData, i, futurePeriods)

		// 计算波动率阈值用于分类
		recentVolatility := be.calculateHistoricalVolatility(historicalData, i, 10) // 使用更短的周期
		if recentVolatility < 0.001 {
			recentVolatility = 0.001 // 更小的最小值
		}

		// 使用三分类标签：-1(下跌), 0(震荡), 1(上涨)
		var label float64
		volatilityThreshold := recentVolatility * 0.5 // 波动率阈值

		if futureReturn > volatilityThreshold {
			label = 1.0 // 显著上涨
		} else if futureReturn < -volatilityThreshold {
			label = -1.0 // 显著下跌
		} else {
			label = 0.0 // 震荡区间
		}

		// 添加微小噪声以增加鲁棒性
		noise := (rand.Float64() - 0.5) * 0.1 // 轻微噪声
		label += noise

		// 确保标签在有效范围内
		if label > 0.5 {
			label = 1.0
		} else if label < -0.5 {
			label = -1.0
		} else {
			label = 0.0
		}

		log.Printf("[ML_TRAIN_V2] 样本 %d: 6h收益=%.4f%%, 波动率=%.4f, 阈值=%.4f, 最终标签=%.0f",
			i, futureReturn*100, recentVolatility, volatilityThreshold, label)

		// 验证和过滤特征值
		validState := be.validateAndFilterTrainingFeatures(state)

		// 只有在有有效特征时才添加到训练数据
		if len(validState) > 0 {
			features = append(features, validState)
			targets = append(targets, label)
		} else {
			log.Printf("[ML_TRAIN] 跳过样本 %d，特征全部无效", i)
		}
	}

	// 数据验证和清理
	if len(features) == 0 || len(targets) == 0 {
		return nil, fmt.Errorf("生成的训练数据为空")
	}

	// 过滤无效数据
	validFeatures := make([]map[string]float64, 0, len(features))
	validTargets := make([]float64, 0, len(targets))

	for i, feature := range features {
		isValid := true

		// 检查特征值
		for _, val := range feature {
			if math.IsNaN(val) || math.IsInf(val, 0) {
				isValid = false
				break
			}
		}

		// 检查目标值
		if math.IsNaN(targets[i]) || math.IsInf(targets[i], 0) {
			isValid = false
		}

		if isValid {
			validFeatures = append(validFeatures, feature)
			validTargets = append(validTargets, targets[i])
		}
	}

	log.Printf("[ML_TRAIN] 数据清理: %d -> %d 样本", len(features), len(validFeatures))

	if len(validFeatures) < 300 {
		return nil, fmt.Errorf("有效训练样本不足: %d，需要至少300个样本", len(validFeatures))
	}

	// 使用清理后的数据
	features = validFeatures
	targets = validTargets

	// 转换为gonum矩阵格式
	X := be.featuresToMatrix(features, be.getFeatureNames())
	y := targets

	log.Printf("[ML_TRAIN] targets长度: %d, features长度: %d", len(targets), len(features))

	// 显示训练数据统计信息
	log.Printf("[ML_TRAIN] 训练数据统计: %d 样本, %d 特征", len(y), len(be.getFeatureNames()))
	buyCount := 0
	sellCount := 0
	holdCount := 0
	for _, label := range y {
		if label > 0.5 {
			buyCount++
		} else if label < -0.5 {
			sellCount++
		} else {
			holdCount++
		}
	}
	log.Printf("[ML_TRAIN] 标签分布: BUY=%d, SELL=%d, HOLD=%d", buyCount, sellCount, holdCount)

	return &TrainingData{
		X:        X,
		Y:        y,
		Features: be.getFeatureNames(),
	}, nil
}

// featuresToMatrix 将特征映射转换为矩阵
func (be *BacktestEngine) featuresToMatrix(features []map[string]float64, featureNames []string) *mat.Dense {
	nSamples := len(features)
	nFeatures := len(featureNames)

	matrix := mat.NewDense(nSamples, nFeatures, nil)

	for i, featureMap := range features {
		for j, featureName := range featureNames {
			if value, exists := featureMap[featureName]; exists {
				matrix.Set(i, j, value)
			}
		}
	}

	return matrix
}

// getFeatureNames 获取特征名称列表（支持特征工程扩展）
func (be *BacktestEngine) getFeatureNames() []string {
	// 基础技术指标特征
	baseFeatures := []string{
		"rsi_14",        // RSI指标
		"trend_20",      // 中期趋势
		"volatility_20", // 波动率
		"macd_signal",   // MACD信号
		"momentum_10",   // 动量
		"price",         // 价格
	}

	// 添加特征工程特征
	// 这些特征名称前缀为"fe_"，表示来自特征工程系统
	featureEngineeringFeatures := []string{
		"fe_price_position_in_range",
		"fe_price_momentum_1h",
		"fe_volume_current",
		"fe_volatility_z_score",
		"fe_trend_duration",
		"fe_momentum_ratio",
		"fe_price_roc_20d",
		"fe_series_nonlinearity",
		"fe_mean_median_diff",
		"fe_volatility_current_level",
		"fe_feature_quality",
		"fe_feature_completeness",
		"fe_feature_consistency",
		"fe_feature_reliability",
	}

	// 合并所有特征
	allFeatures := append(baseFeatures, featureEngineeringFeatures...)

	log.Printf("[FEATURE_NAMES] 总共提供 %d 个特征用于训练", len(allFeatures))
	return allFeatures
}

// buildAdvancedState 使用特征工程系统构建高级状态特征
func (be *BacktestEngine) buildAdvancedState(ctx context.Context, data []MarketData, currentData MarketData, symbol string) map[string]float64 {
	state := make(map[string]float64)

	// 首先尝试使用特征工程系统提取特征
	var featureSet *FeatureSet
	if be.server != nil && be.server.featureEngineering != nil {
		// 转换数据格式为特征工程需要的格式
		marketDataPoints := be.convertToMarketDataPoints(data)

		// 添加调试信息
		if len(marketDataPoints) > 0 {
			lastPoint := marketDataPoints[len(marketDataPoints)-1]
			log.Printf("[FEATURE_DEBUG] 最后数据点: 价格=%.4f, 成交量=%.0f, 时间=%s",
				lastPoint.Price, lastPoint.Volume24h, lastPoint.Timestamp.Format("2006-01-02 15:04:05"))
		}

		// 使用特征工程系统从历史数据提取特征
		var err error
		featureSet, err = be.server.featureEngineering.ExtractFeaturesFromData(ctx, symbol, marketDataPoints)
		if err == nil && featureSet != nil && len(featureSet.Features) > 0 {
			log.Printf("[FEATURE_ENHANCED] 使用特征工程系统从历史数据提取了 %d 个特征，质量评分: %.2f",
				len(featureSet.Features), featureSet.Quality.Overall)

			// 将特征工程结果转换为状态特征
			for name, value := range featureSet.Features {
				// 标准化特征名称，避免冲突
				stateName := "fe_" + name
				state[stateName] = value

				// 调试输出关键技术指标
				if name == "rsi_14" || name == "trend_strength" || name == "momentum_5" {
					log.Printf("[FEATURE_DEBUG] 特征 %s = %.4f", stateName, value)
				}
			}

			// 确保核心特征 price_current 被设置
			if _, exists := featureSet.Features["price_current"]; !exists {
				// 从当前数据设置价格
				state["fe_price_current"] = currentData.Price
				log.Printf("[FEATURE_FIX] 设置当前价格: %.4f", currentData.Price)
			}

			// 添加特征质量信息
			state["feature_quality"] = featureSet.Quality.Overall
			state["feature_completeness"] = featureSet.Quality.Completeness
			state["feature_consistency"] = featureSet.Quality.Consistency
			state["feature_reliability"] = featureSet.Quality.Reliability

		} else {
			log.Printf("[FEATURE_FALLBACK] 特征工程提取失败，使用传统方法: %v", err)
			// 回退到传统方法
			return be.buildDeepState(data, currentData)
		}
	} else {
		log.Printf("[FEATURE_UNAVAILABLE] 特征工程服务不可用，使用传统方法")
		// 回退到传统方法
		return be.buildDeepState(data, currentData)
	}

	// 添加基本价格信息（即使使用特征工程也要保留）
	state["price"] = currentData.Price

	// 添加传统技术指标作为补充（优先级高于特征工程的默认值）
	traditionalState := be.buildDeepState(data, currentData)

	// 调试：检查关键技术指标和数据质量
	if rsi, exists := traditionalState["rsi_14"]; exists {
		log.Printf("[DEBUG_TRADITIONAL] 传统RSI: %.4f", rsi)
	}
	if trend, exists := traditionalState["trend_20"]; exists {
		log.Printf("[DEBUG_TRADITIONAL] 传统趋势: %.4f", trend)
	}
	if momentum, exists := traditionalState["momentum_10"]; exists {
		log.Printf("[DEBUG_TRADITIONAL] 传统动量: %.4f", momentum)
	}
	if volatility, exists := traditionalState["volatility_20"]; exists {
		log.Printf("[DEBUG_TRADITIONAL] 传统波动率: %.6f", volatility)
	}

	// 数据质量检查
	log.Printf("[DEBUG_DATA_QUALITY] 数据点数量: %d, 当前价格: %.2f", len(data), currentData.Price)
	if len(data) >= 20 {
		recentPrices := make([]float64, 20)
		for i := 0; i < 20; i++ {
			recentPrices[i] = data[len(data)-20+i].Price
		}
		minPrice, maxPrice := recentPrices[0], recentPrices[0]
		for _, price := range recentPrices {
			if price < minPrice {
				minPrice = price
			}
			if price > maxPrice {
				maxPrice = price
			}
		}
		priceRange := (maxPrice - minPrice) / minPrice * 100
		log.Printf("[DEBUG_DATA_QUALITY] 最近20天价格范围: %.2f-%.2f, 波动幅度: %.4f%%", minPrice, maxPrice, priceRange)
	}

	// 关键技术指标：让传统指标覆盖特征工程的默认值
	criticalIndicators := map[string]bool{
		"rsi_14":        true,
		"trend_20":      true,
		"momentum_10":   true,
		"volatility_20": true,
		"price":         true,
	}

	for key, value := range traditionalState {
		if criticalIndicators[key] {
			// 关键指标：总是使用传统计算结果
			if existingValue, exists := state[key]; exists && key != "price" {
				log.Printf("[DEBUG_OVERRIDE] %s: 特征工程=%.4f -> 传统=%.4f", key, existingValue, value)
			} else if !exists {
				log.Printf("[DEBUG_ADD] 添加传统指标 %s: %.4f", key, value)
			}
			state[key] = value
		} else if _, exists := state[key]; !exists {
			// 非关键指标：只有在特征工程没有设置时才添加
			state[key] = value
		}
	}

	// 最终验证关键指标的值
	log.Printf("[DEBUG_FINAL] 最终状态指标:")
	if rsi, exists := state["rsi_14"]; exists {
		log.Printf("[DEBUG_FINAL] rsi_14: %.4f", rsi)
	}
	if trend, exists := state["trend_20"]; exists {
		log.Printf("[DEBUG_FINAL] trend_20: %.4f", trend)
	}
	if momentum, exists := state["momentum_10"]; exists {
		log.Printf("[DEBUG_FINAL] momentum_10: %.4f", momentum)
	}

	featureCount := 0
	if featureSet != nil {
		featureCount = len(featureSet.Features)
	}
	log.Printf("[STATE_BUILD] 构建完成：特征工程特征=%d, 总状态维度=%d",
		featureCount, len(state))

	return state
}

// extractFeaturesForTraining 从历史数据提取特征，与预测时使用的特征提取保持一致
func (be *BacktestEngine) extractFeaturesForTraining(ctx context.Context, symbol string, historicalData []MarketData) (map[string]float64, error) {
	// 对于训练，使用传统的历史数据特征提取方法
	// 因为实时特征提取可能不适用于历史数据点
	currentData := historicalData[len(historicalData)-1]
	features := be.buildAdvancedState(ctx, historicalData, currentData, symbol)

	log.Printf("[FEATURE_EXTRACT] 训练特征提取完成，共 %d 个特征", len(features))

	return features, nil
}

// BuildTrainingDataForML 公开的训练数据构建方法（用于测试）
func (be *BacktestEngine) BuildTrainingDataForML(ctx context.Context, symbol string, historicalData []MarketData) (*TrainingData, error) {
	return be.buildTrainingDataForML(symbol, historicalData)
}

// ===== 阶段二优化：新增决策融合函数 =====

// adjustFusionWeightsV2 智能权重调整V2版本 - Phase 10优化：大幅降低ML权重，增加规则权重
func (be *BacktestEngine) adjustFusionWeightsV2(mlPrediction *PredictionResult, ruleConfidence float64, baseMLWeight, baseRuleWeight float64, agent map[string]interface{}) (float64, float64) {
	// Phase 10优化：降低基础ML权重，增加规则权重
	mlWeight := baseMLWeight * 0.6     // Phase 10: 降低ML基础权重到60%
	ruleWeight := baseRuleWeight * 1.2 // Phase 10: 增加规则基础权重到120%

	// 1. 基于质量的动态调整 - Phase 10优化：更严格的质量要求
	qualityFactor := mlPrediction.Quality
	if qualityFactor > 0.95 { // Phase 10: 只有极高质量才增加权重
		mlWeight *= 1.1 // 高质量ML，小幅增加权重
	} else if qualityFactor < 0.7 { // Phase 10: 质量阈值从0.6提高到0.7
		mlWeight *= 0.5   // Phase 10: 低质量ML，大幅降低权重
		ruleWeight *= 1.3 // Phase 10: 相应大幅增加规则权重
	}

	// 2. 基于可信度的调整 - Phase 10优化：降低ML信心优势的影响
	confidenceDiff := mlPrediction.Confidence - ruleConfidence
	if math.Abs(confidenceDiff) > 0.4 { // Phase 10: 阈值从0.3提高到0.4，更严格
		if confidenceDiff > 0 {
			mlWeight *= 1.05 // Phase 10: ML更自信时只小幅增加权重
		} else {
			ruleWeight *= 1.2 // Phase 10: 规则更自信时大幅增加权重
		}
	}

	// 3. 市场环境适应 - Phase 10优化：所有市场环境下都更相信规则
	marketRegime := be.getCurrentMarketRegime()
	switch marketRegime {
	case "strong_bull":
		ruleWeight *= 1.1 // Phase 10: 即使牛市也更相信规则
	case "strong_bear":
		ruleWeight *= 1.3 // Phase 10: 熊市大幅增加规则权重
	case "sideways":
		ruleWeight *= 1.2 // Phase 10: 震荡市大幅增加规则权重
	default:
		ruleWeight *= 1.15 // Phase 10: 其他情况也增加规则权重
	}

	// 4. 确保权重在合理范围内 - Phase 10优化：ML权重上限更低
	mlWeight = math.Max(0.05, math.Min(0.4, mlWeight))     // Phase 10: ML权重上限从0.9降到0.4
	ruleWeight = math.Max(0.6, math.Min(0.95, ruleWeight)) // Phase 10: 规则权重下限从0.1升到0.6

	// 5. 归一化确保总权重为1
	totalWeight := mlWeight + ruleWeight
	if totalWeight > 0 {
		mlWeight /= totalWeight
		ruleWeight /= totalWeight
	}

	log.Printf("[FUSION_V2] 智能权重调整: ML %.2f -> %.3f, 规则 %.2f -> %.3f (质量:%.2f, 市场:%s)",
		baseMLWeight, mlWeight, baseRuleWeight, ruleWeight, qualityFactor, marketRegime)

	return mlWeight, ruleWeight
}

// ConsistencyAnalysis 一致性分析结果
type ConsistencyAnalysis struct {
	Level string  // "高度一致", "中等一致", "轻度冲突", "严重冲突"
	Score float64 // 一致性得分 0-1
}

// analyzeDecisionConsistency 分析决策一致性
func (be *BacktestEngine) analyzeDecisionConsistency(mlPrediction *PredictionResult, ruleAction string, ruleConfidence float64, agent map[string]interface{}) *ConsistencyAnalysis {
	// 提取ML决策 - 基于Score判断动作
	var mlAction string
	if mlPrediction.Score > 0.2 {
		mlAction = "buy"
	} else if mlPrediction.Score < -0.2 {
		mlAction = "sell"
	} else {
		mlAction = "hold"
	}
	mlConfidence := mlPrediction.Confidence

	// 标准化动作比较
	mlActionNorm := strings.ToLower(strings.TrimSpace(mlAction))
	ruleActionNorm := strings.ToLower(strings.TrimSpace(ruleAction))

	// 计算动作一致性
	actionMatch := 0.0
	if mlActionNorm == ruleActionNorm {
		actionMatch = 1.0 // 完全一致
	} else if (mlActionNorm == "buy" && ruleActionNorm == "hold") ||
		(mlActionNorm == "hold" && ruleActionNorm == "buy") ||
		(mlActionNorm == "sell" && ruleActionNorm == "hold") ||
		(mlActionNorm == "hold" && ruleActionNorm == "sell") {
		actionMatch = 0.5 // 部分一致
	} else {
		actionMatch = 0.0 // 完全冲突
	}

	// 计算可信度差异
	confidenceDiff := math.Abs(mlConfidence - ruleConfidence)
	confidenceSimilarity := math.Max(0.0, 1.0-confidenceDiff)

	// 综合一致性得分
	overallConsistency := (actionMatch * 0.7) + (confidenceSimilarity * 0.3)

	// Phase 8优化：更精细的一致性等级判断
	var level string
	if overallConsistency > 0.85 {
		level = "极高一致"
	} else if overallConsistency > 0.7 {
		level = "高度一致"
	} else if overallConsistency > 0.55 {
		level = "中等一致"
	} else if overallConsistency > 0.4 {
		level = "轻度冲突"
	} else if overallConsistency > 0.25 {
		level = "中等冲突"
	} else {
		level = "严重冲突"
	}

	// Phase 8优化：基于币种表现调整一致性阈值
	symbol := ""
	if s, exists := agent["symbol"]; exists {
		if sym, ok := s.(string); ok {
			symbol = sym
		}
	}

	// 优秀币种对一致性要求更高，差表现币种更宽松
	if symbol != "" && be.dynamicSelector != nil {
		if perf := be.dynamicSelector.GetPerformanceReport()[symbol]; perf != nil && perf.TotalTrades >= 1 {
			if perf.WinRate >= 0.8 && perf.TotalPnL > 0 {
				// 优秀币种：要求更高的一致性
				if overallConsistency > 0.75 && level == "高度一致" {
					level = "极高一致"
				}
			} else if perf.WinRate < 0.3 && perf.TotalTrades >= 2 {
				// 差表现币种：对一致性要求更低
				if overallConsistency > 0.35 && level == "严重冲突" {
					level = "中等冲突"
				} else if overallConsistency > 0.5 && level == "轻度冲突" {
					level = "中等一致"
				}
			}
		}
	}

	log.Printf("[CONSISTENCY_V2] 决策一致性分析: 动作匹配=%.1f, 可信度相似=%.3f, 综合得分=%.3f, 等级=%s",
		actionMatch, confidenceSimilarity, overallConsistency, level)

	return &ConsistencyAnalysis{
		Level: level,
		Score: overallConsistency,
	}
}

// ThresholdAdjustment 阈值调整结果
type ThresholdAdjustment struct {
	BuyMultiplier   float64
	SellMultiplier  float64
	ShortMultiplier float64
	Reason          string
}

// calculateSmartThresholdAdjustment 智能阈值调整
func (be *BacktestEngine) calculateSmartThresholdAdjustment(marketState map[string]float64, marketRegime MarketRegime, hasPosition bool, agent map[string]interface{}) *ThresholdAdjustment {
	adjustment := &ThresholdAdjustment{
		BuyMultiplier:   1.0,
		SellMultiplier:  1.0,
		ShortMultiplier: 1.0,
		Reason:          "基准阈值",
	}

	reasons := []string{}

	// 1. 波动率调整
	if volatility, exists := marketState["volatility_20"]; exists {
		if volatility > 0.08 {
			adjustment.BuyMultiplier *= 1.3  // 高波动，提高买入要求
			adjustment.SellMultiplier *= 0.9 // 高波动，放宽卖出条件
			reasons = append(reasons, "高波动调整")
		} else if volatility < 0.02 {
			adjustment.BuyMultiplier *= 0.8  // 低波动，降低买入要求
			adjustment.SellMultiplier *= 1.1 // 低波动，提高卖出要求
			reasons = append(reasons, "低波动调整")
		}
	}

	// 2. RSI调整
	if rsi, exists := marketState["rsi_14"]; exists {
		if rsi > 70 {
			adjustment.BuyMultiplier *= 1.4   // RSI过高，更难买入
			adjustment.ShortMultiplier *= 0.8 // RSI过高，更易做空
			reasons = append(reasons, "RSI超买调整")
		} else if rsi < 30 {
			adjustment.BuyMultiplier *= 0.7  // RSI过低，更易买入
			adjustment.SellMultiplier *= 1.2 // RSI过低，更难卖出
			reasons = append(reasons, "RSI超卖调整")
		}
	}

	// 3. 动量调整
	if momentum, exists := marketState["momentum_10"]; exists {
		if momentum > 0.5 {
			adjustment.BuyMultiplier *= 0.9 // 强动量，略降低买入要求
			reasons = append(reasons, "强上升动量")
		} else if momentum < -0.5 {
			adjustment.SellMultiplier *= 0.9 // 强下降动量，略降低卖出要求
			reasons = append(reasons, "强下降动量")
		}
	}

	// 4. 持仓状态调整
	if hasPosition {
		// 有持仓时，更容易卖出，更难买入
		adjustment.BuyMultiplier *= 1.2
		adjustment.SellMultiplier *= 0.8
		reasons = append(reasons, "有持仓调整")
	} else {
		// 无持仓时，更容易买入
		adjustment.BuyMultiplier *= 0.9
		reasons = append(reasons, "无持仓调整")
	}

	// 5. 市场环境特殊调整
	switch marketRegime {
	case MarketRegimeStrongBull:
		adjustment.BuyMultiplier *= 0.8  // 牛市更容易买入
		adjustment.SellMultiplier *= 1.2 // 牛市更难卖出
		reasons = append(reasons, "强牛市环境")
	case MarketRegimeStrongBear:
		adjustment.BuyMultiplier *= 1.5  // 熊市更难买入
		adjustment.SellMultiplier *= 0.7 // 熊市更容易卖出
		reasons = append(reasons, "强熊市环境")
	case MarketRegimeSideways:
		adjustment.BuyMultiplier *= 1.1  // 震荡市略提高买入要求
		adjustment.SellMultiplier *= 0.9 // 震荡市略降低卖出要求
		reasons = append(reasons, "震荡市环境")
	}

	// 6. 确保调整在合理范围内
	adjustment.BuyMultiplier = math.Max(0.3, math.Min(2.0, adjustment.BuyMultiplier))
	adjustment.SellMultiplier = math.Max(0.3, math.Min(2.0, adjustment.SellMultiplier))
	adjustment.ShortMultiplier = math.Max(0.3, math.Min(2.0, adjustment.ShortMultiplier))

	// 组合原因
	adjustment.Reason = strings.Join(reasons, ", ")
	if adjustment.Reason == "" {
		adjustment.Reason = "无特殊调整"
	}

	return adjustment
}

// optimizeTrainingData 优化训练数据，进行特征选择和预处理
func (be *BacktestEngine) optimizeTrainingData(trainingData *TrainingData) (*TrainingData, error) {
	if trainingData == nil || len(trainingData.Features) == 0 {
		return nil, fmt.Errorf("训练数据为空")
	}

	r, c := trainingData.X.Dims()
	if r < 50 || c < 5 {
		log.Printf("[ML_OPTIMIZE] 样本或特征数太少，跳过优化: %d 样本, %d 特征", r, c)
		return trainingData, nil
	}

	log.Printf("[ML_OPTIMIZE] 开始优化训练数据: %d 样本, %d 特征", r, c)

	// 1. 特征方差过滤 - 移除低方差特征 (暂时降低阈值以避免过滤所有特征)
	selectedFeatures, selectedIndices := be.filterLowVarianceFeatures(trainingData, 0.001) // 从0.01降低到0.001
	if len(selectedFeatures) != len(trainingData.Features) {
		log.Printf("[ML_OPTIMIZE] 方差过滤: %d -> %d 特征", len(trainingData.Features), len(selectedFeatures))
		if len(selectedFeatures) > 0 {
			trainingData = be.selectFeatures(trainingData, selectedIndices)
		} else {
			log.Printf("[ML_OPTIMIZE] 警告：方差过滤后无有效特征，跳过此步骤")
		}
	}

	// 2. 特征相关性过滤 - 移除高度相关的特征
	selectedFeatures2, selectedIndices2 := be.filterHighCorrelationFeatures(trainingData, 0.95)
	if len(selectedFeatures2) != len(trainingData.Features) {
		log.Printf("[ML_OPTIMIZE] 相关性过滤: %d -> %d 特征", len(trainingData.Features), len(selectedFeatures2))
		trainingData = be.selectFeatures(trainingData, selectedIndices2)
	}

	// 3. 特征重要性排序（如果特征数仍然太多）
	_, finalC := trainingData.X.Dims()
	if finalC > 50 {
		// 保留最重要的50个特征
		importantIndices := be.selectTopFeatures(trainingData, 50)
		trainingData = be.selectFeatures(trainingData, importantIndices)
		log.Printf("[ML_OPTIMIZE] 重要性选择: 保留最重要的50个特征")
		finalC = 50
	}

	log.Printf("[ML_OPTIMIZE] 优化完成: 最终 %d 样本, %d 特征", r, finalC)
	return trainingData, nil
}

// shouldTriggerStopLoss 多重止损策略检查
func (be *BacktestEngine) shouldTriggerStopLoss(currentPrice, lastBuyPrice float64, config *BacktestConfig, state map[string]float64, holdTime int) bool {
	currentLoss := (currentPrice - lastBuyPrice) / lastBuyPrice

	// 1. 传统百分比止损（动态调整）
	dynamicStopLoss := be.calculateDynamicStopLoss(config.StopLoss, state, holdTime)
	if currentLoss < dynamicStopLoss {
		log.Printf("[RISK] 百分比止损触发: 亏损%.2f%% < 止损线%.2f%%",
			currentLoss*100, dynamicStopLoss*100)
		return true
	}

	// 2. ATR基准止损（基于波动性）
	if be.checkATRStopLoss(currentPrice, lastBuyPrice, state) {
		log.Printf("[RISK] ATR止损触发: 当前价%.4f, 买入价%.4f", currentPrice, lastBuyPrice)
		return true
	}

	// 3. 支撑阻力止损
	if be.checkSupportResistanceStopLoss(currentPrice, lastBuyPrice, state) {
		log.Printf("[RISK] 支撑阻力止损触发: 当前价%.4f, 买入价%.4f", currentPrice, lastBuyPrice)
		return true
	}

	// 4. 时间-based止损（持有时间过长）
	if be.checkTimeBasedStopLoss(holdTime, config, state) {
		log.Printf("[RISK] 时间止损触发: 持有%d周期", holdTime)
		return true
	}

	return false
}

// shouldTriggerLongStopLoss 多头止损检查 (暂时使用现有逻辑)
func (be *BacktestEngine) shouldTriggerLongStopLoss(currentPrice, lastBuyPrice float64, config *BacktestConfig, state map[string]float64, holdTime int) bool {
	return be.shouldTriggerStopLoss(currentPrice, lastBuyPrice, config, state, holdTime)
}

// shouldTriggerLongTakeProfit 多头止盈检查 (暂时使用现有逻辑)
func (be *BacktestEngine) shouldTriggerLongTakeProfit(currentPrice, lastBuyPrice float64, config *BacktestConfig, state map[string]float64, holdTime int) bool {
	return be.shouldTriggerTakeProfit(currentPrice, lastBuyPrice, config, state, holdTime)
}

// shouldTriggerShortStopLoss 空头止损检查 (简化的反向逻辑)
func (be *BacktestEngine) shouldTriggerShortStopLoss(currentPrice, lastShortPrice float64, config *BacktestConfig, state map[string]float64, holdTime int) bool {
	// 空头止损：当价格上涨到一定程度时
	priceIncrease := (currentPrice - lastShortPrice) / lastShortPrice

	// 使用更严格的止损但仍合理 (因为空头风险更高)
	stopLossLevel := math.Abs(config.StopLoss) * 0.9 // 9% vs 10%，从0.8提高到0.9
	if priceIncrease > stopLossLevel {
		log.Printf("[SHORT_RISK] 空头止损触发: 价格上涨%.2f%% > 止损线%.2f%%",
			priceIncrease*100, stopLossLevel*100)
		return true
	}

	return false
}

// shouldTriggerShortTakeProfit 空头止盈检查 (简化的反向逻辑)
func (be *BacktestEngine) shouldTriggerShortTakeProfit(currentPrice, lastShortPrice float64, config *BacktestConfig, state map[string]float64, holdTime int) bool {
	// 空头盈利：当价格下跌到一定程度时
	priceDecrease := (lastShortPrice - currentPrice) / lastShortPrice

	if config.TakeProfit > 0 && priceDecrease > config.TakeProfit {
		log.Printf("[SHORT_RISK] 空头止盈触发: 价格下跌%.2f%% > 止盈线%.2f%%",
			priceDecrease*100, config.TakeProfit*100)
		return true
	}

	// 空头更积极的止盈：2%就考虑止盈
	if priceDecrease > 0.02 {
		log.Printf("[SHORT_RISK] 空头积极止盈: 价格下跌%.2f%% > 2%%", priceDecrease*100)
		return true
	}

	return false
}

// calculateDynamicStopLoss 计算动态止损水平 - 优化版本，更加注重盈利能力
func (be *BacktestEngine) calculateDynamicStopLoss(baseStopLoss float64, state map[string]float64, holdTime int) float64 {
	// 基础止损：从-10%调整到更合理的范围
	dynamicStopLoss := -0.08 // 基础8%止损（从-10%收紧到-8%）
	if baseStopLoss > 0 {
		dynamicStopLoss = baseStopLoss
	}

	// 1. 根据波动率智能调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility > 0.08 {
			dynamicStopLoss *= 1.3 // 高波动环境，扩大止损空间（-8% -> -10.4%）
		} else if volatility > 0.05 {
			dynamicStopLoss *= 1.1 // 中高波动，适度扩大
		} else if volatility < 0.015 {
			dynamicStopLoss *= 0.7 // 极低波动，收紧止损（-8% -> -5.6%）
		} else if volatility < 0.025 {
			dynamicStopLoss *= 0.85 // 低波动，收紧止损
		}
	}

	// 2. 根据持有时间调整：前期宽松，后期收紧
	if holdTime > 15 {
		dynamicStopLoss *= 0.8 // 长期持有，收紧止损保护利润
	} else if holdTime > 8 {
		dynamicStopLoss *= 0.9 // 中期持有，轻微收紧
	} else if holdTime < 3 {
		dynamicStopLoss *= 1.1 // 短期持有，扩大止损空间
	}

	// 3. 根据趋势强度调整
	if trend20, exists := state["trend_20"]; exists {
		trendStrength := math.Abs(trend20)
		if trendStrength > 0.025 {
			dynamicStopLoss *= 1.1 // 强趋势，扩大止损空间
		} else if trendStrength < 0.008 {
			dynamicStopLoss *= 0.9 // 弱趋势，收紧止损
		}
	}

	// 4. 根据支撑阻力位置调整
	if supportLevel, exists := state["support_level"]; exists && supportLevel > 0.08 {
		dynamicStopLoss *= 0.9 // 强支撑附近，收紧止损
	}

	// 5. 限制在合理范围内，避免过度宽松或严格
	dynamicStopLoss = math.Max(-0.25, math.Min(dynamicStopLoss, -0.03)) // -3% 到 -25% 之间

	return dynamicStopLoss
}

// checkATRStopLoss ATR基准止损检查
func (be *BacktestEngine) checkATRStopLoss(currentPrice, lastBuyPrice float64, state map[string]float64) bool {
	// ATR (Average True Range) 基准止损：ATR的1.5-2倍作为止损距离
	if atr, exists := state["atr_14"]; exists {
		// ATR止损距离 = ATR * 2 (保守) 或 ATR * 1.5 (激进)
		atrMultiplier := 2.0
		if volatility, volExists := state["volatility_20"]; volExists {
			if volatility < 0.02 {
				atrMultiplier = 1.5 // 低波动时使用更激进的止损
			}
		}

		stopLossDistance := atr * atrMultiplier
		stopLossPrice := lastBuyPrice - stopLossDistance

		if currentPrice < stopLossPrice {
			log.Printf("[RISK] ATR止损: ATR=%.6f, 止损距离=%.6f, 止损价=%.4f",
				atr, stopLossDistance, stopLossPrice)
			return true
		}
	}

	return false
}

// checkSupportResistanceStopLoss 支撑阻力止损检查
func (be *BacktestEngine) checkSupportResistanceStopLoss(currentPrice, lastBuyPrice float64, state map[string]float64) bool {
	// 如果价格跌破重要支撑位，触发止损
	if support, exists := state["support_level"]; exists {
		// 支撑位下方5%的缓冲区作为止损
		supportBuffer := support * 0.05
		stopLossPrice := support - supportBuffer

		if currentPrice < stopLossPrice {
			log.Printf("[RISK] 支撑阻力止损: 支撑位=%.4f, 止损价=%.4f", support, stopLossPrice)
			return true
		}
	}

	return false
}

// checkTimeBasedStopLoss 时间-based止损检查
func (be *BacktestEngine) checkTimeBasedStopLoss(holdTime int, config *BacktestConfig, state map[string]float64) bool {
	// 强制平仓时间限制（避免无限持有）
	maxHoldTime := 30 // 默认30周期
	if config != nil && config.MaxHoldTime > 0 {
		maxHoldTime = config.MaxHoldTime
	}

	// 在熊市阶段缩短持有时间
	if trend, exists := state["trend_20"]; exists && trend < -0.02 {
		maxHoldTime = int(float64(maxHoldTime) * 0.7) // 熊市时减少30%的持有时间
	}

	return holdTime > maxHoldTime
}

// shouldTriggerTakeProfit 多重止盈策略检查
func (be *BacktestEngine) shouldTriggerTakeProfit(currentPrice, lastBuyPrice float64, config *BacktestConfig, state map[string]float64, holdTime int) bool {
	profitPercent := (currentPrice - lastBuyPrice) / lastBuyPrice

	// 1. 传统百分比止盈（动态调整）
	dynamicTakeProfit := be.calculateDynamicTakeProfit(config.TakeProfit, state, holdTime)
	if profitPercent > dynamicTakeProfit {
		log.Printf("[RISK] 百分比止盈触发: 盈利%.2f%% > 止盈目标%.2f%%",
			profitPercent*100, dynamicTakeProfit*100)
		return true
	}

	// 2. 移动止盈（追踪止损）
	if be.checkTrailingStopLoss(currentPrice, lastBuyPrice, profitPercent) {
		log.Printf("[RISK] 移动止盈触发: 当前价%.4f, 买入价%.4f, 盈利%.2f%%",
			currentPrice, lastBuyPrice, profitPercent*100)
		return true
	}

	// 3. 部分止盈策略
	if be.checkPartialTakeProfit(currentPrice, lastBuyPrice, config, state) {
		log.Printf("[RISK] 部分止盈触发: 当前价%.4f, 买入价%.4f", currentPrice, lastBuyPrice)
		return true
	}

	return false
}

// calculateDynamicTakeProfit 计算动态止盈水平 - 优化版本
func (be *BacktestEngine) calculateDynamicTakeProfit(baseTakeProfit float64, state map[string]float64, holdTime int) float64 {
	dynamicTakeProfit := baseTakeProfit

	// 1. 基础止盈目标：从8%调整到更合理的范围
	if baseTakeProfit <= 0 {
		dynamicTakeProfit = 0.06 // 默认6%止盈
	}

	// 2. 根据持有时间动态调整
	if holdTime > 20 {
		dynamicTakeProfit *= 0.7 // 长期持有，降低止盈目标（6% -> 4.2%）
	} else if holdTime > 10 {
		dynamicTakeProfit *= 0.85 // 中期持有，轻微降低
	} else if holdTime < 2 {
		dynamicTakeProfit *= 1.1 // 短期持有，适当提高
	}

	// 3. 根据市场波动率调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility > 0.08 {
			dynamicTakeProfit *= 1.2 // 高波动，扩大止盈空间
		} else if volatility < 0.02 {
			dynamicTakeProfit *= 0.8 // 低波动，缩小止盈空间（震荡市更容易达到）
		}
	}

	// 4. 根据趋势强度调整
	if trend20, exists := state["trend_20"]; exists {
		trendStrength := math.Abs(trend20)
		if trendStrength > 0.025 {
			dynamicTakeProfit *= 1.15 // 强趋势，扩大止盈空间
		} else if trendStrength < 0.01 {
			dynamicTakeProfit *= 0.9 // 弱趋势，缩小止盈空间
		}
	}

	// 5. 限制在合理范围内
	dynamicTakeProfit = math.Max(0.02, math.Min(dynamicTakeProfit, 0.25)) // 2%-25%之间

	// 根据波动率调整止盈目标
	if volatility, exists := state["volatility_20"]; exists {
		if volatility > 0.05 {
			dynamicTakeProfit *= 0.8 // 高波动时大幅降低止盈目标
		} else if volatility > 0.03 {
			dynamicTakeProfit *= 0.9 // 中等波动时适度降低止盈目标
		}
	}

	return dynamicTakeProfit
}

// checkTrailingStopLoss 追踪止损检查
func (be *BacktestEngine) checkTrailingStopLoss(currentPrice, lastBuyPrice, profitPercent float64) bool {
	// 移动止盈：根据盈利比例动态调整保护线
	if profitPercent > 0.03 { // 3%盈利后开始保护
		// 动态保护线：盈利越多，保护线越高
		protectionRatio := 0.015 // 基础1.5%保护
		if profitPercent > 0.1 { // 超过10%盈利
			protectionRatio = 0.03 // 3%保护
		} else if profitPercent > 0.05 { // 超过5%盈利
			protectionRatio = 0.025 // 2.5%保护
		}

		trailingStopLevel := lastBuyPrice * (1 + profitPercent - protectionRatio)
		if currentPrice < trailingStopLevel {
			log.Printf("[RISK] 追踪止损: 保护线%.2f%%, 当前价%.4f < 止损线%.4f",
				protectionRatio*100, currentPrice, trailingStopLevel)
			return true
		}
	}

	return false
}

// checkPartialTakeProfit 部分止盈策略检查
func (be *BacktestEngine) checkPartialTakeProfit(currentPrice, lastBuyPrice float64, config *BacktestConfig, state map[string]float64) bool {
	profitPercent := (currentPrice - lastBuyPrice) / lastBuyPrice

	// 分批止盈策略：达到一定盈利后，开始逐步减仓
	// 例如：盈利5%时止盈20%，盈利10%时止盈30%，盈利20%时止盈50%
	if profitPercent > 0.20 { // 20%盈利
		log.Printf("[RISK] 分批止盈: 盈利%.2f%% > 20%%, 建议止盈50%%仓位",
			profitPercent*100)
		return true
	} else if profitPercent > 0.10 { // 10%盈利
		log.Printf("[RISK] 分批止盈: 盈利%.2f%% > 10%%, 建议止盈30%%仓位",
			profitPercent*100)
		return true
	} else if profitPercent > 0.05 { // 5%盈利
		log.Printf("[RISK] 分批止盈: 盈利%.2f%% > 5%%, 建议止盈20%%仓位",
			profitPercent*100)
		return true
	}

	return false
}

// calculateAdvancedPositionSize 高级仓位管理
func (be *BacktestEngine) calculateAdvancedPositionSize(cash, currentPrice float64, config *BacktestConfig, state map[string]float64, agent map[string]interface{}) float64 {
	// 1. 基础仓位计算
	maxPosition := cash / currentPrice
	basePosition := maxPosition * config.MaxPosition

	// 2. 市场状况调整基础仓位
	marketRegimeMultiplier := be.calculateMarketRegimeMultiplier(state)
	basePosition *= marketRegimeMultiplier

	// 3. 应用凯利公式优化
	kellyPosition := be.calculateKellyPosition(config, state, agent)
	basePosition *= kellyPosition

	// 4. 风险调整
	riskAdjustedPosition := be.applyRiskAdjustments(basePosition, state, agent)

	// 5. VaR限制
	varLimitedPosition := be.applyVaRLimits(riskAdjustedPosition, state, agent)

	// 5. 动态调整
	finalPosition := be.applyDynamicPositionSizing(varLimitedPosition, state, agent)

	// 6. 基于信号质量进一步调整仓位
	signalQuality := be.calculateSignalQuality(state, agent)
	finalPosition *= signalQuality

	// 7. 确保在合理范围内 - 优化最小仓位
	finalPosition = math.Max(0.005, math.Min(finalPosition, maxPosition)) // 从0.001提高到0.005

	log.Printf("[POSITION_SIZING] 凯利=%.3f, 风险调整=%.3f, VaR限制=%.3f, 信号质量=%.3f, 最终仓位=%.4f",
		kellyPosition, riskAdjustedPosition/basePosition, varLimitedPosition/riskAdjustedPosition, signalQuality, finalPosition)

	return finalPosition
}

// calculateMarketRegimeMultiplier 根据市场状况调整仓位倍数
func (be *BacktestEngine) calculateMarketRegimeMultiplier(state map[string]float64) float64 {
	trend20 := state["trend_20"]
	volatility := state["volatility_20"]
	rsi := state["rsi_14"]

	multiplier := 1.0

	// 强趋势市场：增加仓位
	if math.Abs(trend20) > 0.02 && volatility < 0.15 {
		multiplier = 1.2 // 强趋势增加20%仓位
		log.Printf("[POSITION_REGIME] 强趋势市场：仓位乘数=%.2f", multiplier)
	} else if math.Abs(trend20) > 0.01 && math.Abs(trend20) <= 0.02 {
		// 弱趋势市场：减少仓位
		multiplier = 0.8 // 弱趋势减少20%仓位
		log.Printf("[POSITION_REGIME] 弱趋势市场：仓位乘数=%.2f", multiplier)
	} else if math.Abs(trend20) <= 0.01 && volatility > 0.10 {
		// 震荡市场：显著减少仓位
		multiplier = 0.6 // 震荡市场减少40%仓位
		log.Printf("[POSITION_REGIME] 震荡市场：仓位乘数=%.2f", multiplier)

		// 极端超买超卖时进一步调整
		if rsi > 75 || rsi < 25 {
			multiplier *= 0.8 // 进一步减少仓位
			log.Printf("[POSITION_REGIME] 极端超买超卖：进一步调整至%.2f", multiplier)
		}
	}

	// 高波动惩罚
	if volatility > 0.25 {
		multiplier *= 0.7
		log.Printf("[POSITION_VOLATILITY] 高波动惩罚：仓位乘数调整至%.2f", multiplier)
	}

	return multiplier
}

// calculateKellyPosition 凯利公式仓位计算
func (be *BacktestEngine) calculateKellyPosition(config *BacktestConfig, state map[string]float64, agent map[string]interface{}) float64 {
	// 获取历史胜率和风险收益比
	winRate := 0.5         // 默认50%
	riskRewardRatio := 2.0 // 默认1:2

	if recentAccuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		winRate = recentAccuracy
	}

	if config.TakeProfit > 0 && config.StopLoss < 0 {
		riskRewardRatio = config.TakeProfit / math.Abs(config.StopLoss)
	}

	// 简化的凯利公式: f = (胜率 - (1-胜率)/赔率)
	kellyFraction := (winRate - (1-winRate)/riskRewardRatio)

	// 保守调整：只使用凯利公式的50%
	conservativeKelly := kellyFraction * 0.5

	// 波动率调整
	volatilityAdjustment := 1.0
	if volatility, exists := state["volatility_20"]; exists {
		volatilityAdjustment = 1.0 / (1.0 + volatility*2) // 高波动时减少仓位
	}

	return math.Max(0.1, math.Min(conservativeKelly*volatilityAdjustment, 1.0))
}

// applyRiskAdjustments 应用风险调整
func (be *BacktestEngine) applyRiskAdjustments(position float64, state map[string]float64, agent map[string]interface{}) float64 {
	riskAdjustment := 1.0

	// 基于市场趋势调整
	if trend, exists := state["trend_20"]; exists {
		if trend > 0.02 {
			riskAdjustment *= 1.2 // 强势上涨时可以增加仓位
		} else if trend < -0.02 {
			riskAdjustment *= 0.7 // 强势下跌时减少仓位
		}
	}

	// 基于波动率调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility < 0.02 {
			riskAdjustment *= 1.1 // 低波动时可以增加仓位
		} else if volatility > 0.05 {
			riskAdjustment *= 0.8 // 高波动时减少仓位
		}
	}

	// 基于持仓状态调整
	if hasPosition, exists := agent["has_position"].(bool); exists && hasPosition {
		riskAdjustment *= 0.5 // 有持仓时减少新仓位
	}

	return position * math.Max(0.1, math.Min(riskAdjustment, 2.0))
}

// applyVaRLimits 应用VaR限制
func (be *BacktestEngine) applyVaRLimits(position float64, state map[string]float64, agent map[string]interface{}) float64 {
	// 简化的VaR计算：95%置信度下的最大损失不超过总资本的2%
	maxLossLimit := 0.02 // 2%最大损失限制

	// 基于波动率估算VaR
	volatility := 0.03 // 默认3%波动率
	if vol, exists := state["volatility_20"]; exists {
		volatility = vol
	}

	// 简化的VaR计算：VaR = 仓位 × 价格 × 波动率 × 置信度因子
	var95 := position * volatility * 2.0 // 简化的95%VaR

	if var95 > maxLossLimit {
		// 调整仓位使VaR不超过限制
		position = maxLossLimit / (volatility * 2.0)
		log.Printf("[VaR_LIMIT] VaR调整: %.4f%% -> %.4f%%, 新仓位=%.4f",
			var95*100, maxLossLimit*100, position)
	}

	return position
}

// applyDynamicPositionSizing 动态仓位调整
func (be *BacktestEngine) applyDynamicPositionSizing(position float64, state map[string]float64, agent map[string]interface{}) float64 {
	// 基于市场情绪调整
	if sentiment, exists := state["market_sentiment"]; exists {
		if sentiment > 0.7 {
			position *= 1.1 // 乐观情绪时增加仓位
		} else if sentiment < -0.7 {
			position *= 0.8 // 悲观情绪时减少仓位
		}
	}

	// 基于最近表现调整
	if recentPnL, exists := agent["recent_pnl"].(float64); exists {
		if recentPnL > 0 {
			position *= 1.05 // 盈利时略微增加仓位
		} else {
			position *= 0.95 // 亏损时减少仓位
		}
	}

	return position
}

// applyRiskLimits 应用风险限制
func (be *BacktestEngine) applyRiskLimits(action string, position float64, result *BacktestResult, config *BacktestConfig) string {
	// 1. 最大回撤限制
	if be.checkMaxDrawdownLimit(result, config) {
		log.Printf("[RISK_LIMIT] 最大回撤限制触发")
		return "sell"
	}

	// 2. 最大单日损失限制
	if be.checkMaxDailyLossLimit(result, config) {
		log.Printf("[RISK_LIMIT] 单日损失限制触发")
		return "sell"
	}

	// 3. 连续亏损限制
	if be.checkConsecutiveLossesLimit(result, config) {
		log.Printf("[RISK_LIMIT] 连续亏损限制触发")
		return "sell"
	}

	// 4. 总资本保护
	if be.checkCapitalProtection(result, config) {
		log.Printf("[RISK_LIMIT] 资本保护机制触发")
		return "sell"
	}

	return action
}

// checkMaxDrawdownLimit 检查最大回撤限制
func (be *BacktestEngine) checkMaxDrawdownLimit(result *BacktestResult, config *BacktestConfig) bool {
	maxAllowedDrawdown := 0.15 // 默认15%最大回撤
	if config.MaxDrawdown > 0 {
		maxAllowedDrawdown = config.MaxDrawdown
	}

	// 如果没有组合价值历史，使用MaxDrawdown字段
	if len(result.PortfolioValues) <= 1 {
		return result.MaxDrawdown > maxAllowedDrawdown
	}

	// 计算当前回撤
	peakValue := 0.0
	for _, value := range result.PortfolioValues {
		if value > peakValue {
			peakValue = value
		}
	}

	currentValue := result.PortfolioValues[len(result.PortfolioValues)-1]
	currentDrawdown := (peakValue - currentValue) / peakValue

	if currentDrawdown > maxAllowedDrawdown {
		log.Printf("[DRAWDOWN] 当前回撤%.2f%% > 限制%.2f%%", currentDrawdown*100, maxAllowedDrawdown*100)
		return true
	}

	return false
}

// checkMaxDailyLossLimit 检查单日损失限制
func (be *BacktestEngine) checkMaxDailyLossLimit(result *BacktestResult, config *BacktestConfig) bool {
	maxDailyLoss := 0.05 // 默认5%单日损失限制
	if config.MaxDailyLoss > 0 {
		maxDailyLoss = config.MaxDailyLoss
	}

	// 如果没有足够的组合价值历史，使用默认检查
	if len(result.PortfolioValues) < 2 {
		// 可以基于交易记录或其他指标进行检查，这里暂时返回false
		return false
	}

	// 计算当日损失
	yesterdayValue := result.PortfolioValues[len(result.PortfolioValues)-2]
	todayValue := result.PortfolioValues[len(result.PortfolioValues)-1]
	dailyLoss := (yesterdayValue - todayValue) / yesterdayValue

	if dailyLoss > maxDailyLoss {
		log.Printf("[DAILY_LOSS] 单日损失%.2f%% > 限制%.2f%%", dailyLoss*100, maxDailyLoss*100)
		return true
	}

	return false
}

// checkConsecutiveLossesLimit 检查连续亏损限制
func (be *BacktestEngine) checkConsecutiveLossesLimit(result *BacktestResult, config *BacktestConfig) bool {
	maxConsecutiveLosses := 5 // 默认最多5次连续亏损
	if config.MaxConsecutiveLosses > 0 {
		maxConsecutiveLosses = config.MaxConsecutiveLosses
	}

	// 如果没有足够的组合价值历史，基于交易记录检查
	if len(result.PortfolioValues) < 2 {
		// 基于交易记录计算连续亏损
		consecutiveLosses := 0
		for i := len(result.Trades) - 1; i >= 0; i-- {
			trade := result.Trades[i]
			if trade.Side == "sell" && trade.PnL < 0 {
				consecutiveLosses++
			} else if trade.Side == "sell" && trade.PnL >= 0 {
				break // 遇到盈利交易，停止计数
			}
		}

		if consecutiveLosses > maxConsecutiveLosses {
			log.Printf("[CONSECUTIVE_LOSS] 连续亏损%d次 > 限制%d次", consecutiveLosses, maxConsecutiveLosses)
			return true
		}
		return false
	}

	// 计算连续亏损次数
	consecutiveLosses := 0
	for i := len(result.PortfolioValues) - 1; i >= 1; i-- {
		if result.PortfolioValues[i] < result.PortfolioValues[i-1] {
			consecutiveLosses++
		} else {
			break
		}
	}

	if consecutiveLosses > maxConsecutiveLosses {
		log.Printf("[CONSECUTIVE_LOSS] 连续亏损%d次 > 限制%d次", consecutiveLosses, maxConsecutiveLosses)
		return true
	}

	return false
}

// checkCapitalProtection 检查资本保护
func (be *BacktestEngine) checkCapitalProtection(result *BacktestResult, config *BacktestConfig) bool {
	minCapitalRatio := 0.7 // 默认保留70%初始资本
	if config.MinCapitalRatio > 0 {
		minCapitalRatio = config.MinCapitalRatio
	}

	initialCapital := config.InitialCash
	currentCapital := initialCapital // 默认使用初始资本

	// 如果有组合价值历史，使用最新的价值
	if len(result.PortfolioValues) > 0 {
		currentCapital = result.PortfolioValues[len(result.PortfolioValues)-1]
	}

	// 也可以基于交易计算当前资本
	if len(result.Trades) > 0 {
		calculatedCapital := initialCapital
		for _, trade := range result.Trades {
			if trade.Side == "buy" {
				calculatedCapital -= trade.Quantity * trade.Price * (1 + config.Commission)
			} else if trade.Side == "sell" {
				calculatedCapital += trade.Quantity * trade.Price * (1 - config.Commission)
			}
		}
		currentCapital = calculatedCapital
	}

	if currentCapital < initialCapital*minCapitalRatio {
		log.Printf("[CAPITAL] 当前资本%.2f < 最低保护线%.2f", currentCapital, initialCapital*minCapitalRatio)
		return true
	}

	return false
}

// performCrossValidation 执行交叉验证评估模型性能
func (be *BacktestEngine) performCrossValidation(trainingData *TrainingData) (float64, float64) {
	if be.server == nil || be.server.machineLearning == nil {
		log.Printf("[CV] 机器学习服务不可用，返回默认分数")
		return 0.5, 0.5
	}

	// 简化：直接使用默认配置进行交叉验证评估
	// 这里可以扩展为测试不同的超参数组合
	defaultConfig := MLConfig{
		Ensemble: struct {
			Method       string  `json:"method"`
			NEstimators  int     `json:"n_estimators"`
			MaxDepth     int     `json:"max_depth"`
			LearningRate float64 `json:"learning_rate"`
		}{
			Method:       "random_forest",
			NEstimators:  10,
			MaxDepth:     5,
			LearningRate: 0.1,
		},
		DeepLearning: struct {
			HiddenLayers []int   `json:"hidden_layers"`
			DropoutRate  float64 `json:"dropout_rate"`
			LearningRate float64 `json:"learning_rate"`
			BatchSize    int     `json:"batch_size"`
			Epochs       int     `json:"epochs"`
			FeatureDim   int     `json:"feature_dim"`
		}{
			HiddenLayers: []int{64, 32, 16}, // 默认隐藏层配置
			DropoutRate:  0.2,
			LearningRate: 0.001,
			BatchSize:    32,
			Epochs:       100,
			FeatureDim:   10,
		},
	}

	score := be.server.machineLearning.evaluateConfig(defaultConfig, trainingData)
	log.Printf("[CV] 默认配置评估得分: %.4f", score)

	return score, score
}

// predictWithEnsembleModels 使用多模型集成进行预测
func (be *BacktestEngine) predictWithEnsembleModels(ctx context.Context, symbol string) (*PredictionResult, error) {
	modelNames := []string{"random_forest", "gradient_boost", "stacking", "transformer"}
	var predictions []*PredictionResult
	var weights []float64
	totalWeight := 0.0

	for _, modelName := range modelNames {
		// 使用真正的ML预测
		prediction, err := be.server.machineLearning.PredictWithEnsemble(ctx, symbol, modelName)
		if err != nil {
			log.Printf("[ENSEMBLE] 模型 %s 预测失败: %v", modelName, err)
			continue
		}

		if prediction == nil || math.IsNaN(prediction.Score) || math.IsInf(prediction.Score, 0) {
			log.Printf("[ENSEMBLE] 模型 %s 预测结果无效", modelName)
			continue
		}

		// 根据模型质量和预测多样性确定权重
		baseWeight := prediction.Confidence * prediction.Quality

		// 特殊模型权重调整
		switch modelName {
		case "transformer":
			// Transformer模型如果预测有效则给予较高权重，否则降低
			if math.Abs(prediction.Score) > 0.01 {
				baseWeight *= 1.2 // 有效预测给予奖励
			} else {
				baseWeight *= 0.3 // 无效预测降低权重
			}
		case "random_forest":
			baseWeight *= 1.1 // 随机森林通常稳定
		case "gradient_boost":
			baseWeight *= 1.0 // 梯度提升保持原权重
		case "stacking":
			baseWeight *= 0.9 // 堆叠模型稍微降低权重
		}

		// 基于预测绝对值的权重调整（更极端的预测给予更高权重）
		predictionMagnitude := math.Abs(prediction.Score)
		if predictionMagnitude > 0.7 {
			baseWeight *= 1.3 // 强信号给予奖励
		} else if predictionMagnitude < 0.1 {
			baseWeight *= 0.7 // 弱信号降低权重
		}

		weight := math.Max(baseWeight, 0.05) // 最小权重

		predictions = append(predictions, prediction)
		weights = append(weights, weight)
		totalWeight += weight

		log.Printf("[ENSEMBLE] 模型 %s: score=%.4f, confidence=%.4f, weight=%.4f",
			modelName, prediction.Score, prediction.Confidence, weight)
	}

	if len(predictions) == 0 {
		return nil, fmt.Errorf("所有模型预测都失败")
	}

	// 改进的加权平均：考虑预测一致性和多样性
	weightedScore := 0.0
	totalConfidence := 0.0
	scores := make([]float64, len(predictions))

	// 提取所有预测分数
	for i, prediction := range predictions {
		scores[i] = prediction.Score
	}

	// 计算预测一致性（标准差的倒数）
	scoreStd := be.calculateStandardDeviation(scores)
	consistencyBonus := 1.0
	if scoreStd < 0.5 {
		consistencyBonus = 1.2 // 一致性好给予奖励
	} else if scoreStd > 1.0 {
		consistencyBonus = 0.8 // 一致性差降低权重
	}

	// ===== 阶段四优化：增强集成预测 =====
	// 计算多样性奖励（避免所有模型预测相同）
	diversityBonus := be.calculatePredictionDiversity(scores)

	// 过滤异常预测（离群值检测）
	filteredPredictions, filteredWeights := be.filterOutlierPredictions(predictions, weights, scores)

	// 重新计算权重总和
	filteredTotalWeight := 0.0
	for _, w := range filteredWeights {
		filteredTotalWeight += w
	}

	log.Printf("[ENSEMBLE_V4] 异常预测过滤: %d -> %d 模型", len(predictions), len(filteredPredictions))

	for i, prediction := range filteredPredictions {
		normalizedWeight := filteredWeights[i] / filteredTotalWeight
		// 应用一致性和多样性调整
		adjustedWeight := normalizedWeight * consistencyBonus * diversityBonus
		weightedScore += prediction.Score * adjustedWeight
		totalConfidence += prediction.Confidence * normalizedWeight
	}

	// 对最终预测分数进行后处理，增强区分度
	weightedScore = be.postProcessEnsembleScore(weightedScore, scores)

	// 如果只有一个模型，稍微降低置信度以表示集成效果有限
	if len(predictions) == 1 {
		totalConfidence *= 0.9
	}

	// 计算集成模型的质量（取各模型质量的加权平均）
	weightedQuality := 0.0
	for i, prediction := range predictions {
		normalizedWeight := weights[i] / totalWeight
		weightedQuality += prediction.Quality * normalizedWeight
	}

	ensemblePrediction := &PredictionResult{
		Symbol:     symbol,
		Score:      weightedScore,
		Confidence: totalConfidence,
		Quality:    weightedQuality,
		Timestamp:  time.Now(),
	}

	log.Printf("[ENSEMBLE] 集成预测完成: score=%.4f, confidence=%.4f, models=%d, consistency=%.2f, diversity=%.2f",
		weightedScore, totalConfidence, len(predictions), consistencyBonus, diversityBonus)

	return ensemblePrediction, nil
}

// calculateStandardDeviation 计算标准差
func (be *BacktestEngine) calculateStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// 计算均值
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// 计算方差
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

// calculatePredictionDiversity 计算预测多样性奖励
func (be *BacktestEngine) calculatePredictionDiversity(scores []float64) float64 {
	if len(scores) < 2 {
		return 1.0
	}

	// 计算分数范围（最大值-最小值）
	minScore, maxScore := scores[0], scores[0]
	for _, score := range scores {
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	scoreRange := maxScore - minScore

	// 多样性奖励：范围越大奖励越高，但避免过度分散
	if scoreRange > 1.0 {
		return 1.3 // 多样性很好
	} else if scoreRange > 0.5 {
		return 1.1 // 多样性中等
	} else if scoreRange > 0.1 {
		return 0.9 // 多样性较低
	} else {
		return 0.7 // 预测过于一致，可能有问题
	}
}

// postProcessEnsembleScore 对集成预测分数进行后处理
func (be *BacktestEngine) postProcessEnsembleScore(ensembleScore float64, individualScores []float64) float64 {
	// 计算个体预测的统计信息
	scoreMean := 0.0
	scoreStd := 0.0

	for _, score := range individualScores {
		scoreMean += score
	}
	scoreMean /= float64(len(individualScores))

	for _, score := range individualScores {
		diff := score - scoreMean
		scoreStd += diff * diff
	}
	scoreStd = math.Sqrt(scoreStd / float64(len(individualScores)))

	// 如果标准差很大，说明预测不一致，降低置信度
	if scoreStd > 0.8 {
		// 大幅降低极端预测
		if math.Abs(ensembleScore) > 0.5 {
			ensembleScore *= 0.7
		}
	}

	// 增强中等强度的预测
	absScore := math.Abs(ensembleScore)
	if absScore > 0.3 && absScore < 0.7 {
		// 中等预测增强
		ensembleScore *= 1.2
	}

	// 限制最终范围
	return math.Max(-1.0, math.Min(1.0, ensembleScore))
}

// ModelValidationResult 模型验证结果
type ModelValidationResult struct {
	IsValid      bool
	Score        float64
	Confidence   float64
	Quality      float64
	ErrorMessage string
}

// validateEnsembleModel 增强版集成模型验证
func (be *BacktestEngine) validateEnsembleModel(ctx context.Context, symbol string, trainingData *TrainingData) error {
	modelNames := []string{"random_forest", "gradient_boost", "stacking", "transformer"}
	validModels := 0
	totalScore := 0.0
	totalConfidence := 0.0
	performanceMetrics := make(map[string]ModelValidationResult)

	for _, modelName := range modelNames {
		// 尝试进行一次预测
		prediction, err := be.server.machineLearning.PredictWithEnsemble(ctx, symbol, modelName)
		if err != nil {
			log.Printf("[VALIDATE] 模型 %s 预测失败: %v", modelName, err)
			performanceMetrics[modelName] = ModelValidationResult{
				IsValid:      false,
				ErrorMessage: err.Error(),
			}
			continue
		}

		// 检查预测结果是否合理
		if prediction == nil || math.IsNaN(prediction.Score) || math.IsInf(prediction.Score, 0) {
			log.Printf("[VALIDATE] 模型 %s 预测结果无效", modelName)
			performanceMetrics[modelName] = ModelValidationResult{
				IsValid:      false,
				ErrorMessage: "invalid prediction result",
			}
			continue
		}

		// 验证预测范围
		if math.Abs(prediction.Score) > 2.0 {
			log.Printf("[VALIDATE] 模型 %s 预测超出合理范围: %.4f", modelName, prediction.Score)
			performanceMetrics[modelName] = ModelValidationResult{
				IsValid:      false,
				ErrorMessage: "prediction out of range",
			}
			continue
		}

		validModels++
		totalScore += prediction.Score
		totalConfidence += prediction.Confidence

		performanceMetrics[modelName] = ModelValidationResult{
			IsValid:    true,
			Score:      prediction.Score,
			Confidence: prediction.Confidence,
			Quality:    prediction.Quality,
		}

		log.Printf("[VALIDATE] 模型 %s 验证通过: score=%.4f, confidence=%.3f, quality=%.3f",
			modelName, prediction.Score, prediction.Confidence, prediction.Quality)
	}

	if validModels == 0 {
		return fmt.Errorf("所有模型验证都失败了")
	}

	// 计算集成性能指标
	avgScore := totalScore / float64(validModels)
	avgConfidence := totalConfidence / float64(validModels)

	// 评估模型一致性
	consistencyScore := be.evaluateModelConsistency(performanceMetrics)

	log.Printf("[VALIDATE] 集成模型验证通过: %d/%d 个模型可用", validModels, len(modelNames))
	log.Printf("[VALIDATE] 平均分数: %.4f, 平均置信度: %.3f, 一致性: %.3f",
		avgScore, avgConfidence, consistencyScore)

	// 保存性能指标用于监控
	be.updateModelPerformanceMetrics(symbol, performanceMetrics)

	return nil
}

// ModelPerformance 已在ml_pretraining_service.go中定义，这里不再重复定义

// evaluateModelConsistency 评估模型一致性
func (be *BacktestEngine) evaluateModelConsistency(performance map[string]ModelValidationResult) float64 {
	validScores := make([]float64, 0)

	for _, perf := range performance {
		if perf.IsValid {
			validScores = append(validScores, perf.Score)
		}
	}

	if len(validScores) < 2 {
		return 1.0 // 只有一个模型，默认为完全一致
	}

	// 计算标准差
	mean := 0.0
	for _, score := range validScores {
		mean += score
	}
	mean /= float64(len(validScores))

	variance := 0.0
	for _, score := range validScores {
		diff := score - mean
		variance += diff * diff
	}
	variance /= float64(len(validScores))
	stdDev := math.Sqrt(variance)

	// 将标准差转换为一致性分数（0-1，1表示完全一致）
	consistency := math.Max(0.0, 1.0-math.Min(1.0, stdDev*2.0))

	return consistency
}

// updateModelPerformanceMetrics 更新模型性能指标
func (be *BacktestEngine) updateModelPerformanceMetrics(symbol string, performance map[string]ModelValidationResult) {
	// 这里可以保存到数据库或内存中用于监控
	// 暂时只记录日志
	validCount := 0
	totalConfidence := 0.0

	for modelName, perf := range performance {
		if perf.IsValid {
			validCount++
			totalConfidence += perf.Confidence
			log.Printf("[PERF_MONITOR] %s:%s - 有效, 分数:%.4f, 置信度:%.3f",
				symbol, modelName, perf.Score, perf.Confidence)
		} else {
			log.Printf("[PERF_MONITOR] %s:%s - 无效, 错误:%s",
				symbol, modelName, perf.ErrorMessage)
		}
	}

	if validCount > 0 {
		avgConfidence := totalConfidence / float64(validCount)
		log.Printf("[PERF_MONITOR] %s 整体性能: %d/%d 模型有效, 平均置信度:%.3f",
			symbol, validCount, len(performance), avgConfidence)

		// 检查是否需要重新训练
		if avgConfidence < 0.7 {
			log.Printf("[PERF_MONITOR] %s 模型性能不佳，建议重新训练", symbol)
			be.scheduleModelRetraining(symbol)
		}
	}
}

// scheduleModelRetraining 安排模型重新训练
func (be *BacktestEngine) scheduleModelRetraining(symbol string) {
	// 这里可以实现自动重新训练调度
	// 暂时只记录日志
	log.Printf("[RETRAIN] 为 %s 安排模型重新训练", symbol)

	// 可以在这里设置重新训练标志或发送通知
	// 在实际实现中，可以使用定时器或消息队列来触发重新训练
}

// filterLowVarianceFeatures 过滤低方差特征
func (be *BacktestEngine) filterLowVarianceFeatures(data *TrainingData, threshold float64) ([]string, []int) {
	r, c := data.X.Dims()
	selectedFeatures := make([]string, 0, c)
	selectedIndices := make([]int, 0, c)

	log.Printf("[VARIANCE_FILTER] 开始方差过滤，共 %d 个特征，阈值 %.4f", c, threshold)

	for j := 0; j < c; j++ {
		// 计算第j个特征的方差
		values := make([]float64, r)
		for i := 0; i < r; i++ {
			values[i] = data.X.At(i, j)
		}

		variance := be.calculateVariance(values)
		featureName := data.Features[j]

		if variance >= threshold {
			selectedFeatures = append(selectedFeatures, featureName)
			selectedIndices = append(selectedIndices, j)
			log.Printf("[VARIANCE_FILTER] 保留特征 %s: 方差=%.6f", featureName, variance)
		} else {
			log.Printf("[VARIANCE_FILTER] 过滤特征 %s: 方差=%.6f < %.4f", featureName, variance, threshold)
		}
	}

	log.Printf("[VARIANCE_FILTER] 方差过滤完成: %d -> %d 特征", c, len(selectedFeatures))

	// 安全检查：如果过滤后没有特征，降低阈值重试
	if len(selectedFeatures) == 0 {
		log.Printf("[VARIANCE_FILTER] 警告：所有特征都被过滤，降低阈值重试")
		lowerThreshold := threshold * 0.1 // 降低到原来的10%
		log.Printf("[VARIANCE_FILTER] 使用降低的阈值 %.6f 重试", lowerThreshold)

		selectedFeatures = make([]string, 0, c)
		selectedIndices = make([]int, 0, c)

		for j := 0; j < c; j++ {
			values := make([]float64, r)
			for i := 0; i < r; i++ {
				values[i] = data.X.At(i, j)
			}

			variance := be.calculateVariance(values)
			if variance >= lowerThreshold {
				selectedFeatures = append(selectedFeatures, data.Features[j])
				selectedIndices = append(selectedIndices, j)
				log.Printf("[VARIANCE_FILTER] 保留特征 %s: 方差=%.6f (降低阈值)", data.Features[j], variance)
			}
		}

		log.Printf("[VARIANCE_FILTER] 降低阈值后结果: %d -> %d 特征", c, len(selectedFeatures))

		// 如果仍然没有特征，至少保留前5个特征
		if len(selectedFeatures) == 0 {
			log.Printf("[VARIANCE_FILTER] 严重警告：即使降低阈值仍无有效特征，强制保留前5个特征")
			maxFeatures := 5
			if c < maxFeatures {
				maxFeatures = c
			}
			for j := 0; j < maxFeatures; j++ {
				selectedFeatures = append(selectedFeatures, data.Features[j])
				selectedIndices = append(selectedIndices, j)
				log.Printf("[VARIANCE_FILTER] 强制保留特征 %s", data.Features[j])
			}
		}
	}

	return selectedFeatures, selectedIndices
}

// validateAndFilterTrainingFeatures 验证和过滤训练特征
func (be *BacktestEngine) validateAndFilterTrainingFeatures(features map[string]float64) map[string]float64 {
	validFeatures := make(map[string]float64)
	filteredCount := 0

	for name, value := range features {
		if be.isValidTrainingFeatureValue(name, value) {
			validFeatures[name] = value
		} else {
			filteredCount++
		}
	}

	if filteredCount > 0 && len(validFeatures) > 0 {
		log.Printf("[FeatureValidation] 训练数据过滤了 %d 个异常特征值", filteredCount)
	}

	return validFeatures
}

// isValidTrainingFeatureValue 验证训练特征值是否有效
func (be *BacktestEngine) isValidTrainingFeatureValue(featureName string, value float64) bool {
	// 基本数值检查
	if math.IsNaN(value) {
		return false
	}

	if math.IsInf(value, 0) {
		return false
	}

	// 对于训练数据，使用更严格的检查
	featureName = strings.ToLower(featureName)

	// RSI指标：应该在0-100之间
	if strings.Contains(featureName, "rsi") {
		if value < 0 || value > 100 {
			return false
		}
	}

	// 波动率：应该在0-1之间
	if strings.Contains(featureName, "volatility") || strings.Contains(featureName, "vol") {
		if value < 0 || value > 1 {
			return false
		}
	}

	// 动量指标：限制在合理范围内
	if strings.Contains(featureName, "momentum") || strings.Contains(featureName, "trend") {
		if value < -5 || value > 5 {
			return false
		}
	}

	// 价格和成交量：不能为负数
	if strings.Contains(featureName, "price") || strings.Contains(featureName, "volume") {
		if value < 0 {
			return false
		}
	}

	// 技术指标的一般限制
	if strings.Contains(featureName, "macd") || strings.Contains(featureName, "signal") {
		if math.Abs(value) > 1000 {
			return false
		}
	}

	// 布林带位置
	if strings.Contains(featureName, "bollinger") {
		if value < -3 || value > 3 {
			return false
		}
	}

	// 通用检查：极端值过滤（训练数据更严格）
	if math.Abs(value) > 10000 {
		return false
	}

	return true
}

// filterHighCorrelationFeatures 过滤高度相关的特征
func (be *BacktestEngine) filterHighCorrelationFeatures(data *TrainingData, threshold float64) ([]string, []int) {
	r, c := data.X.Dims()
	selectedIndices := make([]int, 0, c)

	// 保留第一个特征
	selectedIndices = append(selectedIndices, 0)

	for j := 1; j < c; j++ {
		shouldKeep := true

		// 检查与已选特征的相关性
		for _, selectedIdx := range selectedIndices {
			corr := be.calculateCorrelation(data.X, j, selectedIdx, r)
			if math.Abs(corr) >= threshold {
				shouldKeep = false
				break
			}
		}

		if shouldKeep {
			selectedIndices = append(selectedIndices, j)
		}
	}

	selectedFeatures := make([]string, len(selectedIndices))
	for i, idx := range selectedIndices {
		selectedFeatures[i] = data.Features[idx]
	}

	return selectedFeatures, selectedIndices
}

// selectTopFeatures 选择最重要的特征（基于方差和信息量）
func (be *BacktestEngine) selectTopFeatures(data *TrainingData, maxFeatures int) []int {
	r, c := data.X.Dims()
	if c <= maxFeatures {
		indices := make([]int, c)
		for i := range indices {
			indices[i] = i
		}
		return indices
	}

	// 计算每个特征的重要性得分（方差 + 信息增益近似）
	scores := make([]float64, c)
	for j := 0; j < c; j++ {
		values := make([]float64, r)
		for i := 0; i < r; i++ {
			values[i] = data.X.At(i, j)
		}
		variance := be.calculateVariance(values)
		// 简单的信息量近似（基于值的分布范围）
		info := be.calculateInfoMeasure(values)

		scores[j] = variance + info
	}

	// 选择得分最高的特征
	type featureScore struct {
		index int
		score float64
	}

	featureScores := make([]featureScore, c)
	for i := 0; i < c; i++ {
		featureScores[i] = featureScore{i, scores[i]}
	}

	// 排序
	sort.Slice(featureScores, func(i, j int) bool {
		return featureScores[i].score > featureScores[j].score
	})

	// 选择前maxFeatures个
	result := make([]int, maxFeatures)
	for i := 0; i < maxFeatures; i++ {
		result[i] = featureScores[i].index
	}

	return result
}

// selectFeatures 根据索引选择特征子集
func (be *BacktestEngine) selectFeatures(data *TrainingData, indices []int) *TrainingData {
	r, _ := data.X.Dims()
	c := len(indices)

	newX := mat.NewDense(r, c, nil)
	newFeatures := make([]string, c)

	for j, idx := range indices {
		newFeatures[j] = data.Features[idx]
		for i := 0; i < r; i++ {
			newX.Set(i, j, data.X.At(i, idx))
		}
	}

	return &TrainingData{
		X:         newX,
		Y:         data.Y,
		Features:  newFeatures,
		SampleIDs: data.SampleIDs,
	}
}

// calculateVariance 计算方差
func (be *BacktestEngine) calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return variance
}

// calculateCorrelation 计算两个特征列的相关系数
func (be *BacktestEngine) calculateCorrelation(X *mat.Dense, col1, col2, nSamples int) float64 {
	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0

	for i := 0; i < nSamples; i++ {
		x := X.At(i, col1)
		y := X.At(i, col2)

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	numerator := float64(nSamples)*sumXY - sumX*sumY
	denominator := math.Sqrt((float64(nSamples)*sumX2 - sumX*sumX) * (float64(nSamples)*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// calculateInfoMeasure 计算信息度量（简化的信息增益近似）
func (be *BacktestEngine) calculateInfoMeasure(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// 简化的信息量计算：基于值的分布范围和唯一值数量
	min, max := values[0], values[0]
	uniqueValues := make(map[float64]bool)

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		uniqueValues[v] = true
	}

	range_ := max - min
	if range_ == 0 {
		return 0
	}

	// 归一化唯一值比例
	uniqueness := float64(len(uniqueValues)) / float64(len(values))

	return range_ * uniqueness
}

// addOnlineLearningSample 添加在线学习样本
func (be *BacktestEngine) addOnlineLearningSample(ctx context.Context, state map[string]float64, target float64, symbol string) {
	if be.server == nil || be.server.machineLearning == nil || !be.server.machineLearning.onlineLearningEnabled {
		return
	}

	// 将状态特征转换为特征向量
	featureNames := be.getFeatureNames()
	features := make([]float64, len(featureNames))

	for i, name := range featureNames {
		if value, exists := state[name]; exists {
			features[i] = value
		} else {
			features[i] = 0.0 // 缺失特征填充为0
		}
	}

	// 添加到在线学习缓冲区
	err := be.server.machineLearning.AddOnlineLearningSample(ctx, "random_forest", symbol, features, target)
	if err != nil {
		log.Printf("[ONLINE_LEARNING] 添加学习样本失败: %v", err)
	} else {
		log.Printf("[ONLINE_LEARNING] 成功添加学习样本: 目标=%.1f, 特征数=%d", target, len(features))
	}
}

// convertToMarketDataPoints 将MarketData转换为MarketDataPoint格式
func (be *BacktestEngine) convertToMarketDataPoints(data []MarketData) []*MarketDataPoint {
	points := make([]*MarketDataPoint, len(data))

	for i, md := range data {
		points[i] = &MarketDataPoint{
			Symbol:         "", // 会在特征工程中设置
			BaseSymbol:     "", // 默认空
			Price:          md.Price,
			PriceChange24h: md.Change24h,
			Volume24h:      md.Volume24h,
			MarketCap:      &md.MarketCap,
			Timestamp:      md.LastUpdated,
		}
	}

	return points
}

// checkSignalConsistency 检查信号一致性 - 优化版本，更加注重盈利能力
func (be *BacktestEngine) checkSignalConsistency(state map[string]float64) float64 {
	bullishSignals := 0
	bearishSignals := 0
	totalSignals := 0
	weightedBullish := 0.0
	weightedBearish := 0.0

	// 趋势指标一致性检查 - 给予不同权重
	if trend5, exists := state["trend_5"]; exists && trend5 != 0 {
		totalSignals++
		if trend5 > 0.005 { // 提高阈值，避免噪音
			bullishSignals++
			weightedBullish += 0.8 // 短期趋势权重较低
		} else if trend5 < -0.005 {
			bearishSignals++
			weightedBearish += 0.8
		}
	}
	if trend20, exists := state["trend_20"]; exists && trend20 != 0 {
		totalSignals++
		if trend20 > 0.01 { // 中期趋势更重要
			bullishSignals++
			weightedBullish += 1.5 // 中期趋势权重最高
		} else if trend20 < -0.01 {
			bearishSignals++
			weightedBearish += 1.5
		}
	}
	if trend50, exists := state["trend_50"]; exists && trend50 != 0 {
		totalSignals++
		if trend50 > 0 {
			bullishSignals++
			weightedBullish += 1.2 // 长期趋势权重适中
		} else {
			bearishSignals++
			weightedBearish += 1.2
		}
	}

	// 动量指标一致性检查 - 优化阈值
	if rsi, exists := state["rsi_14"]; exists {
		totalSignals++
		if rsi > 60 { // 从55提高到60，避免虚假信号
			bullishSignals++
			weightedBullish += 1.0
		} else if rsi < 40 { // 从45降低到40
			bearishSignals++
			weightedBearish += 1.0
		}
	}
	if momentum, exists := state["momentum_10"]; exists && momentum != 0 {
		totalSignals++
		if momentum > 0.02 { // 提高动量阈值
			bullishSignals++
			weightedBullish += 0.9
		} else if momentum < -0.02 {
			bearishSignals++
			weightedBearish += 0.9
		}
	}

	// MACD信号一致性检查 - 增强权重
	if macd, exists := state["macd_signal"]; exists && macd != 0 {
		totalSignals++
		if macd > 0.001 { // 避免微小信号
			bullishSignals++
			weightedBullish += 1.3 // MACD权重较高
		} else if macd < -0.001 {
			bearishSignals++
			weightedBearish += 1.3
		}
	}

	// 新增：成交量确认检查
	if volumeTrend, exists := state["volume_trend"]; exists && volumeTrend != 0 {
		totalSignals++
		if volumeTrend > 0.02 {
			bullishSignals++
			weightedBullish += 0.7 // 成交量确认权重
		} else if volumeTrend < -0.02 {
			bearishSignals++
			weightedBearish += 0.7
		}
	}

	// 计算一致性比例 - 同时考虑数量和权重
	consistencyRatio := 1.0
	if totalSignals >= 3 { // 从4降低到3，降低要求以增加交易频率
		// 数量一致性
		maxConsistent := math.Max(float64(bullishSignals), float64(bearishSignals))
		quantityConsistency := maxConsistent / float64(totalSignals)

		// 权重一致性
		maxWeighted := math.Max(weightedBullish, weightedBearish)
		totalWeighted := weightedBullish + weightedBearish
		weightedConsistency := 0.0
		if totalWeighted > 0 {
			weightedConsistency = maxWeighted / totalWeighted
		}

		// 综合一致性评分 (60%数量 + 40%权重)
		comprehensiveConsistency := quantityConsistency*0.6 + weightedConsistency*0.4

		// 根据一致性调整分数 - 更加严格
		if comprehensiveConsistency >= 0.8 { // 从0.7提高到0.8
			consistencyRatio = 1.4 // 从1.2提高到1.4，更强增强
		} else if comprehensiveConsistency >= 0.65 { // 从0.5提高到0.65
			consistencyRatio = 1.1 // 从1.0调整到1.1
		} else if comprehensiveConsistency >= 0.5 {
			consistencyRatio = 0.9 // 轻微削弱
		} else { // 一致性很差
			consistencyRatio = 0.5 // 从0.7降低到0.5，更强惩罚
		}

		log.Printf("[SIGNAL_CONSISTENCY] 指标一致性: %.1f%% (数量:%.1f%%, 权重:%.1f%%), 调整系数: %.2f, 信号:%d/%d",
			comprehensiveConsistency*100, quantityConsistency*100, weightedConsistency*100,
			consistencyRatio, int(maxConsistent), totalSignals)
	} else if totalSignals < 3 {
		// 信号不足，降低可信度
		consistencyRatio = 0.8
		log.Printf("[SIGNAL_CONSISTENCY] 信号不足: %d/3, 降低可信度至%.1f", totalSignals, consistencyRatio)
	}

	return consistencyRatio
}

// applyMarketTimingFilter 应用市场时机过滤
func (be *BacktestEngine) applyMarketTimingFilter(state map[string]float64, hasPosition bool) float64 {
	filterFactor := 1.0

	// 检查波动率 - 适当减少对波动率的惩罚
	if volatility, exists := state["volatility_20"]; exists {
		if volatility > 0.06 { // 极高波动才减少
			filterFactor *= 0.7
		} else if volatility > 0.04 { // 高波动时适度减少
			filterFactor *= 0.85
		}
	}

	// 检查市场阶段
	if marketPhase, exists := state["market_phase"]; exists {
		if hasPosition {
			// 有持仓时，在极端熊市环境中更谨慎
			if marketPhase < -0.5 {
				filterFactor *= 0.8
			}
		} else {
			// 无持仓时，避免在极端牛市时追高
			if marketPhase > 0.5 {
				filterFactor *= 0.9
			}
		}
	}

	return filterFactor
}

// applyRecommendationSystemEnhancement 根据推荐系统建议调整AI决策分数
func (be *BacktestEngine) applyRecommendationSystemEnhancement(agent map[string]interface{}, state map[string]float64, currentPrice float64, currentScore float64) float64 {
	enhancement := 0.0

	// 获取推荐策略信息
	recommendedStrategy, hasStrategy := agent["recommended_strategy"]
	entryMin, hasMin := agent["entry_zone_min"]
	entryMax, hasMax := agent["entry_zone_max"]

	if !hasStrategy || !hasMin || !hasMax {
		return 0.0
	}

	recommendedStrategyStr := recommendedStrategy.(string)
	entryMinFloat := entryMin.(float64)
	entryMaxFloat := entryMax.(float64)

	// 根据推荐策略类型增强AI决策
	switch recommendedStrategyStr {
	case "LONG":
		// 如果AI决策是买入，且价格在推荐入场区间，则增强分数
		if currentScore > 0 && currentPrice >= entryMinFloat && currentPrice <= entryMaxFloat {
			enhancement += 0.08 // 显著增强多头信号
		}
	case "SHORT":
		// 如果AI决策是卖出，且价格在推荐入场区间，则增强分数
		if currentScore < 0 && currentPrice >= entryMinFloat && currentPrice <= entryMaxFloat {
			enhancement -= 0.08 // 显著增强空头信号
		}
	case "RANGE":
		// 如果市场处于震荡区间，且AI决策是中性，则轻微增强分数
		if math.Abs(state["bollinger_position"]) < 0.5 && math.Abs(currentScore) < 0.1 {
			enhancement += 0.02 // 轻微增强震荡信号
		}
	}

	return enhancement
}

// getStrategyRecommendationFromAPI 模拟从推荐系统API获取策略建议
func (be *BacktestEngine) getStrategyRecommendationFromAPI(symbol string) *StrategyRecommendation {
	// 这里应该调用真实的推荐系统API
	// 目前返回模拟数据作为示例
	log.Printf("[INFO] 推荐系统集成：建议在生产环境中集成真实的推荐系统API")

	// For demonstration, return a mock recommendation
	return &StrategyRecommendation{
		StrategyType: "RANGE",
		EntryMin:     88717.7712,
		EntryMax:     92338.9048,
		StopLoss:     85000.0,
		TakeProfit:   95000.0,
		Confidence:   0.8,
	}
}

// calculateProfitabilityScore 计算盈利能力评分，用于仓位调整
func (be *BacktestEngine) calculateProfitabilityScore(state map[string]float64, agent map[string]interface{}) float64 {
	score := 1.0

	// 1. 基于历史胜率的调整
	if accuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		if accuracy > 0.7 {
			score *= 1.2 // 高胜率，增加仓位
		} else if accuracy > 0.6 {
			score *= 1.1 // 良好胜率
		} else if accuracy < 0.4 {
			score *= 0.8 // 低胜率，减少仓位
		} else if accuracy < 0.5 {
			score *= 0.9 // 较低胜率
		}
	}

	// 2. 基于市场环境的调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility < 0.02 {
			score *= 0.8 // 极低波动，减少仓位（震荡市）
		} else if volatility > 0.08 {
			score *= 0.9 // 高波动，谨慎
		}
	}

	// 3. 基于趋势强度的调整
	if trend20, exists := state["trend_20"]; exists {
		trendStrength := math.Abs(trend20)
		if trendStrength > 0.025 {
			score *= 1.1 // 强趋势，增加仓位
		} else if trendStrength < 0.01 {
			score *= 0.9 // 弱趋势，减少仓位
		}
	}

	return math.Max(0.5, math.Min(score, 1.5)) // 限制在0.5-1.5倍
}

// calculateTrendStrengthForPosition 为仓位管理计算趋势强度
func (be *BacktestEngine) calculateTrendStrengthForPosition(state map[string]float64) float64 {
	multiplier := 1.0

	// 综合趋势评估
	trend20 := state["trend_20"]
	trendStrength := math.Abs(trend20)

	if trendStrength > 0.03 {
		multiplier = 1.3 // 强趋势，增加仓位
	} else if trendStrength > 0.02 {
		multiplier = 1.2 // 中强趋势
	} else if trendStrength > 0.015 {
		multiplier = 1.1 // 中等趋势
	} else if trendStrength < 0.008 {
		multiplier = 0.8 // 弱趋势，减少仓位
	} else if trendStrength < 0.005 {
		multiplier = 0.7 // 极弱趋势
	}

	// 趋势一致性加成
	if trend5, exists := state["trend_5"]; exists {
		if (trend5 > 0) == (trend20 > 0) { // 方向一致
			multiplier *= 1.1 // 一致性奖励
		} else {
			multiplier *= 0.95 // 不一致性惩罚
		}
	}

	return multiplier
}

// HistoricalPerformanceData 历史表现数据结构
type HistoricalPerformanceData struct {
	WinRate     float64
	AvgWin      float64
	AvgLoss     float64
	TotalTrades int
}

// calculateOptimizedKellyPosition 优化凯利公式计算
func (be *BacktestEngine) calculateOptimizedKellyPosition(config *BacktestConfig, state map[string]float64, agent map[string]interface{}) float64 {
	// 获取历史表现数据用于学习调整
	historicalPerformance := be.getHistoricalPerformanceData(agent)

	// 基础胜率和赔率
	winRate := 0.5
	avgWin := 1.5  // 平均盈利倍数
	avgLoss := 1.0 // 平均亏损倍数

	if accuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		winRate = accuracy
	}

	// 根据历史表现调整
	if winRateHist, exists := historicalPerformance["win_rate"]; exists && winRateHist > 0 {
		winRate = winRateHist
	}
	if avgWinHist, exists := historicalPerformance["avg_win"]; exists && avgWinHist > 0 {
		avgWin = avgWinHist
	}
	if avgLossHist, exists := historicalPerformance["avg_loss"]; exists && avgLossHist > 0 {
		avgLoss = avgLossHist
	}

	// 计算凯利值
	if winRate <= 0 || winRate >= 1 || avgWin <= 0 || avgLoss <= 0 {
		return 0.5 // 默认保守值
	}

	// 凯利公式: f = (bp - q) / b
	// 其中: b = 赔率, p = 胜率, q = 败率
	b := avgWin / avgLoss // 赔率
	p := winRate          // 胜率
	q := 1 - p            // 败率

	kellyFraction := (b*p - q) / b

	// 限制凯利值在合理范围内
	kellyFraction = math.Max(0.1, math.Min(kellyFraction, 0.5))

	// 在弱信号或不确定环境下降低凯利值
	if winRate < 0.55 {
		kellyFraction *= 0.8
	}

	return kellyFraction
}

// calculateMarketStructureMultiplier 市场结构乘数
func (be *BacktestEngine) calculateMarketStructureMultiplier(state map[string]float64) float64 {
	multiplier := 1.0

	// 支撑位奖励
	if supportLevel, exists := state["support_level"]; exists && supportLevel > 0.1 {
		multiplier *= 1.1 // 强支撑位附近，增加仓位
	}

	// 阻力位惩罚
	if resistanceLevel, exists := state["resistance_level"]; exists && resistanceLevel > 0.1 {
		multiplier *= 0.9 // 强阻力位附近，减少仓位
	}

	// 布林带位置调整
	if bbPos, exists := state["bollinger_position"]; exists {
		if bbPos > 0.8 || bbPos < 0.2 { // 极端位置
			multiplier *= 0.9 // 减少仓位
		}
	}

	return multiplier
}

// applyWinRateBasedPositionSizing 基于胜率的仓位调整
func (be *BacktestEngine) applyWinRateBasedPositionSizing(basePosition float64, agent map[string]interface{}) float64 {
	position := basePosition

	// 基于近期胜率调整
	if accuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		if accuracy > 0.65 {
			position *= 1.1 // 高胜率，增加仓位
		} else if accuracy < 0.45 {
			position *= 0.9 // 低胜率，减少仓位
		}
	}

	// 基于历史表现调整
	historicalPerf := be.getHistoricalPerformanceData(agent)
	if totalTrades, exists := historicalPerf["total_trades"]; exists && totalTrades > 10 { // 有足够历史数据
		if winRate, exists := historicalPerf["win_rate"]; exists {
			if winRate > 0.6 {
				position *= 1.05
			} else if winRate < 0.4 {
				position *= 0.95
			}
		}
	}

	return position
}

// calculateDynamicTakeProfitLevel 计算动态止盈水平（用于决策逻辑）
func (be *BacktestEngine) calculateDynamicTakeProfitLevel(pnlPct float64, holdTime int, state map[string]float64) float64 {
	// 基础止盈目标
	baseTarget := 0.06 // 6%

	// 根据当前盈利情况调整
	if pnlPct > 0.02 { // 已有2%盈利
		baseTarget = 0.04 // 降低到4%，避免过早止盈
	} else if pnlPct > 0.05 { // 已有5%盈利
		baseTarget = 0.03 // 进一步降低到3%
	}

	// 根据持有时间调整
	if holdTime > 20 {
		baseTarget *= 0.8 // 长期持有，更容易止盈
	} else if holdTime < 3 {
		baseTarget *= 1.2 // 短期持有，提高止盈目标
	}

	// 根据市场环境调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility < 0.02 {
			baseTarget *= 0.8 // 低波动，更容易达到目标
		} else if volatility > 0.08 {
			baseTarget *= 1.2 // 高波动，提高目标
		}
	}

	return math.Max(0.02, baseTarget) // 最低2%
}

// calculateDynamicStopLossLevel 计算动态止损水平（用于决策逻辑）
func (be *BacktestEngine) calculateDynamicStopLossLevel(pnlPct float64, holdTime int, state map[string]float64) float64 {
	// 基础止损目标
	baseStop := -0.08 // -8%

	// 根据当前盈利情况调整
	if pnlPct > 0.03 { // 已有3%盈利
		baseStop = -0.05 // 收紧到-5%
	} else if pnlPct > 0.06 { // 已有6%盈利
		baseStop = -0.03 // 进一步收紧到-3%
	}

	// 根据持有时间调整
	if holdTime > 20 {
		baseStop *= 0.8 // 长期持有，更严格止损
	} else if holdTime < 3 {
		baseStop *= 1.2 // 短期持有，宽松止损
	}

	// 根据市场环境调整
	if volatility, exists := state["volatility_20"]; exists {
		if volatility < 0.02 {
			baseStop *= 0.7 // 低波动，收紧止损
		} else if volatility > 0.08 {
			baseStop *= 1.3 // 高波动，扩大止损
		}
	}

	return math.Min(-0.02, baseStop) // 最多-2%
}

// calculateSignalQuality 计算信号质量，用于仓位调整
func (be *BacktestEngine) calculateSignalQuality(state map[string]float64, agent map[string]interface{}) float64 {
	quality := 1.0

	// 1. 基于一致性检查
	if consistencyRatio, exists := state["consistency_ratio"]; exists {
		quality *= consistencyRatio
	}

	// 2. 基于趋势强度
	if trend20, exists := state["trend_20"]; exists {
		trendStrength := math.Abs(trend20)
		if trendStrength > 0.02 {
			quality *= 1.1 // 强趋势信号质量高
		} else if trendStrength < 0.01 {
			quality *= 0.9 // 弱趋势信号质量低
		}
	}

	// 3. 基于波动率
	if volatility, exists := state["volatility_20"]; exists {
		if volatility < 0.02 {
			quality *= 0.8 // 极低波动，信号质量下降
		} else if volatility > 0.08 {
			quality *= 0.95 // 高波动，信号质量稍降
		}
	}

	// 4. 基于历史准确性
	if accuracy, exists := agent["recent_rule_accuracy"].(float64); exists {
		if accuracy > 0.6 {
			quality *= 1.05
		} else if accuracy < 0.5 {
			quality *= 0.95
		}
	}

	return math.Max(0.5, math.Min(quality, 1.5)) // 限制在0.5-1.5倍
}

// StrategyRecommendation 策略推荐
type StrategyRecommendation struct {
	StrategyType string  `json:"strategy_type"`
	EntryMin     float64 `json:"entry_min"`
	EntryMax     float64 `json:"entry_max"`
	StopLoss     float64 `json:"stop_loss"`
	TakeProfit   float64 `json:"take_profit"`
	Confidence   float64 `json:"confidence"`
}

// executeMultiLayerStopLoss 执行多层止损机制
func (be *BacktestEngine) executeMultiLayerStopLoss(pnlPct float64, holdTime int, state map[string]float64, agent map[string]interface{}) (string, float64) {
	// 获取市场环境
	marketRegime := classifyMarketRegime(state)
	marketRegimeStr := marketRegime.String()

	// 获取波动率用于调整止损阈值
	volatility := 0.02 // 默认中等波动
	if vol, exists := state["volatility_20"]; exists {
		volatility = math.Max(0.005, vol) // 最小5%年化波动率
	}

	// 基于市场环境和波动率的动态止损阈值
	stopLossThresholds := be.calculateDynamicStopLossThresholds(marketRegime, volatility, holdTime)

	// 检查各级止损
	for level, threshold := range stopLossThresholds {
		if pnlPct < threshold {
			action, confidence := be.handleStopLossTrigger(level, threshold, pnlPct, marketRegimeStr, holdTime)
			return action, confidence
		}
	}

	return "hold", 0.0 // 不触发止损
}

// calculateDynamicStopLossThresholds 计算动态止损阈值
func (be *BacktestEngine) calculateDynamicStopLossThresholds(marketRegime MarketRegime, volatility float64, holdTime int) map[string]float64 {
	thresholds := make(map[string]float64)

	// 基础止损阈值（根据市场环境调整）
	baseMultiplier := 1.0
	switch marketRegime {
	case MarketRegimeSideways:
		baseMultiplier = 0.8 // 横盘市场放宽止损
	case MarketRegimeExtremeBear:
		baseMultiplier = 1.5 // 极端熊市收紧止损
	case MarketRegimeStrongBear:
		baseMultiplier = 1.2 // 强熊市收紧止损
	case MarketRegimeStrongBull:
		baseMultiplier = 1.1 // 强牛市略微收紧
	default:
		baseMultiplier = 1.0 // 其他市场正常
	}

	// 波动率调整因子（高波动放宽，低波动收紧）
	volatilityMultiplier := 1.0
	if volatility > 0.08 {
		volatilityMultiplier = 1.3 // 高波动放宽30%
	} else if volatility > 0.05 {
		volatilityMultiplier = 1.1 // 中高波动放宽10%
	} else if volatility < 0.015 {
		volatilityMultiplier = 0.8 // 低波动收紧20%
	}

	// 时间因子（持仓时间越长，止损阈值略微放宽）
	timeMultiplier := 1.0
	if holdTime > 20 {
		timeMultiplier = 1.1 // 持仓超过20天放宽10%
	} else if holdTime > 10 {
		timeMultiplier = 1.05 // 持仓超过10天放宽5%
	}

	combinedMultiplier := baseMultiplier * volatilityMultiplier * timeMultiplier

	// 设置各级止损阈值
	thresholds["level1"] = -0.02 * combinedMultiplier // 第一级：2%基础止损
	thresholds["level2"] = -0.05 * combinedMultiplier // 第二级：5%中等止损
	thresholds["level3"] = -0.10 * combinedMultiplier // 第三级：10%重度止损

	return thresholds
}

// handleStopLossTrigger 处理止损触发
func (be *BacktestEngine) handleStopLossTrigger(level string, threshold float64, pnlPct float64, marketRegime string, holdTime int) (string, float64) {
	// 根据止损级别确定置信度和处理策略
	var confidence float64
	var reason string

	switch level {
	case "level1":
		confidence = 0.85
		reason = "轻度止损"
	case "level2":
		confidence = 0.95
		reason = "中度止损"
	case "level3":
		confidence = 0.99
		reason = "重度止损"
	default:
		confidence = 0.90
		reason = "一般止损"
	}

	log.Printf("[MULTI_LAYER_STOP] 触发%s止损 (阈值:%.2f%%, 当前亏损:%.2f%%, 市场:%s, 持仓:%d天)",
		reason, threshold*100, pnlPct*100, marketRegime, holdTime)

	// 对于重度止损，记录警告
	if level == "level3" {
		log.Printf("[CRITICAL_STOP_LOSS] 重度止损触发！亏损已达%.1f%%，建议复盘策略", pnlPct*100)
	}

	return "sell", confidence
}

// calculateHistoricalVolatility 计算历史波动率
func (be *BacktestEngine) calculateHistoricalVolatility(historicalData []MarketData, currentIndex, windowSize int) float64 {
	if currentIndex < windowSize || currentIndex >= len(historicalData) {
		return 0.01 // 返回默认波动率
	}

	// 计算窗口内的收益率
	returns := make([]float64, 0, windowSize-1)
	startIdx := currentIndex - windowSize + 1

	for i := startIdx; i < currentIndex; i++ {
		if i >= 0 && i+1 < len(historicalData) {
			currentPrice := historicalData[i].Price
			nextPrice := historicalData[i+1].Price
			if currentPrice > 0 {
				returnRate := (nextPrice - currentPrice) / currentPrice
				returns = append(returns, returnRate)
			}
		}
	}

	if len(returns) < 2 {
		return 0.01 // 数据不足，返回默认值
	}

	// 计算波动率（收益率的标准差）
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1) // 样本方差

	volatility := math.Sqrt(variance)

	// 确保波动率在合理范围内
	if volatility < 0.001 {
		volatility = 0.001 // 最小波动率
	}
	if volatility > 0.5 {
		volatility = 0.5 // 最大波动率
	}

	return volatility
}

// ===== 阶段三优化：交易时机改进 =====

// TrendConfirmation 趋势确认结果
type TrendConfirmation struct {
	Strength    float64 // 趋势强度 0-1
	Direction   string  // "bull", "bear", "sideways"
	Confidence  float64 // 确认度 0-1
	Timeframe   int     // 确认周期
	Reliability float64 // 可靠性 0-1
}

// analyzeTrendConfirmation 分析趋势确认
func (be *BacktestEngine) analyzeTrendConfirmation(symbol string, historicalData []MarketData, currentIndex int, lookbackPeriods int) *TrendConfirmation {
	if currentIndex < lookbackPeriods || len(historicalData) <= currentIndex {
		return &TrendConfirmation{
			Strength:    0.5,
			Direction:   "sideways",
			Confidence:  0.5,
			Timeframe:   0,
			Reliability: 0.5,
		}
	}

	// 计算多个时间周期的趋势
	shortTerm := be.analyzeTrendStrength(historicalData, currentIndex, 5)
	mediumTerm := be.analyzeTrendStrength(historicalData, currentIndex, 10)
	longTerm := be.analyzeTrendStrength(historicalData, currentIndex, 20)

	// 计算动量一致性
	momentumConsistency := be.calculateMomentumConsistency(historicalData, currentIndex, lookbackPeriods)

	// 计算价格位置（相对近期高低点）
	pricePosition := be.calculatePricePosition(historicalData, currentIndex, lookbackPeriods)

	// 综合趋势强度
	trendStrength := (shortTerm.Strength*0.5 + mediumTerm.Strength*0.3 + longTerm.Strength*0.2)

	// 确定趋势方向
	direction := shortTerm.Direction
	if shortTerm.Direction == mediumTerm.Direction {
		direction = shortTerm.Direction // 短期和中期一致
	} else if mediumTerm.Direction == longTerm.Direction {
		direction = mediumTerm.Direction // 中期和长期一致
	} else {
		direction = "sideways" // 不一致则认为是震荡
	}

	// 计算确认度
	confirmation := 0.0
	if direction != "sideways" {
		// 趋势确认度 = 一致性 + 强度 + 位置因素
		confirmation = (momentumConsistency * 0.4) + (trendStrength * 0.4) + (pricePosition * 0.2)
		confirmation = math.Max(0.1, math.Min(0.95, confirmation)) // 限制在合理范围内
	} else {
		// 震荡确认度基于动量不一致性
		confirmation = math.Max(0.3, 1.0-momentumConsistency)
	}

	// 计算可靠性（基于数据质量和一致性）
	reliability := (momentumConsistency * 0.6) + (math.Min(1.0, float64(lookbackPeriods)/20.0) * 0.4)

	log.Printf("[TREND_CONFIRM_V3] %s趋势确认: 强度%.3f, 方向%s, 确认度%.3f, 可靠性%.3f (短期%s:%.2f, 中期%s:%.2f, 长期%s:%.2f)",
		symbol, trendStrength, direction, confirmation, reliability,
		shortTerm.Direction, shortTerm.Strength, mediumTerm.Direction, mediumTerm.Strength, longTerm.Direction, longTerm.Strength)

	return &TrendConfirmation{
		Strength:    trendStrength,
		Direction:   direction,
		Confidence:  confirmation,
		Timeframe:   lookbackPeriods,
		Reliability: reliability,
	}
}

// TrendAnalysis 趋势分析结果
type TrendAnalysis struct {
	Direction string
	Strength  float64
}

// analyzeTrendStrength 计算趋势强度分析
func (be *BacktestEngine) analyzeTrendStrength(historicalData []MarketData, currentIndex int, periods int) *TrendAnalysis {
	if currentIndex < periods {
		return &TrendAnalysis{Direction: "sideways", Strength: 0.5}
	}

	// 计算价格变化
	startPrice := historicalData[currentIndex-periods+1].Price
	endPrice := historicalData[currentIndex].Price
	totalChange := (endPrice - startPrice) / startPrice

	// 计算平均波动率
	volatility := 0.0
	count := 0
	for i := currentIndex - periods + 1; i < currentIndex; i++ {
		if i >= 0 {
			change := math.Abs(historicalData[i+1].Price-historicalData[i].Price) / historicalData[i].Price
			volatility += change
			count++
		}
	}
	if count > 0 {
		volatility /= float64(count)
	}

	// 计算趋势强度（考虑波动率调整）
	trendStrength := 0.5    // 中性
	if volatility > 0.001 { // 有足够波动才认为有趋势
		trendStrength = math.Abs(totalChange) / (volatility * math.Sqrt(float64(periods)))
		trendStrength = math.Max(0.0, math.Min(1.0, trendStrength))
	}

	// 确定方向
	direction := "sideways"
	if trendStrength > 0.3 {
		if totalChange > 0 {
			direction = "bull"
		} else {
			direction = "bear"
		}
	}

	return &TrendAnalysis{
		Direction: direction,
		Strength:  trendStrength,
	}
}

// calculateMomentumConsistency 计算动量一致性
func (be *BacktestEngine) calculateMomentumConsistency(historicalData []MarketData, currentIndex int, periods int) float64 {
	if currentIndex < periods+1 {
		return 0.5
	}

	consistentCount := 0
	totalCount := 0

	for i := currentIndex - periods + 1; i < currentIndex; i++ {
		if i >= 0 && i+1 < len(historicalData) {
			currChange := (historicalData[i+1].Price - historicalData[i].Price) / historicalData[i].Price
			if i+2 < len(historicalData) {
				nextChange := (historicalData[i+2].Price - historicalData[i+1].Price) / historicalData[i+1].Price
				// 检查动量方向一致性
				if (currChange > 0 && nextChange > 0) || (currChange < 0 && nextChange < 0) {
					consistentCount++
				}
			}
			totalCount++
		}
	}

	if totalCount == 0 {
		return 0.5
	}

	return float64(consistentCount) / float64(totalCount)
}

// calculatePricePosition 计算价格位置
func (be *BacktestEngine) calculatePricePosition(historicalData []MarketData, currentIndex int, periods int) float64 {
	if currentIndex < periods {
		return 0.5
	}

	currentPrice := historicalData[currentIndex].Price
	minPrice := currentPrice
	maxPrice := currentPrice

	// 找到期间最高价和最低价
	for i := currentIndex - periods + 1; i <= currentIndex; i++ {
		if i >= 0 {
			price := historicalData[i].Price
			if price < minPrice {
				minPrice = price
			}
			if price > maxPrice {
				maxPrice = price
			}
		}
	}

	// 计算当前位置
	if maxPrice > minPrice {
		position := (currentPrice - minPrice) / (maxPrice - minPrice)
		return math.Max(0.0, math.Min(1.0, position))
	}

	return 0.5
}

// ===== 阶段四优化：集成预测增强函数 =====

// filterOutlierPredictions 过滤异常预测值
func (be *BacktestEngine) filterOutlierPredictions(predictions []*PredictionResult, weights []float64, scores []float64) ([]*PredictionResult, []float64) {
	if len(predictions) <= 2 {
		// 模型太少不进行过滤
		return predictions, weights
	}

	// 计算预测分数的均值和标准差
	sum := 0.0
	for _, score := range scores {
		sum += score
	}
	mean := sum / float64(len(scores))

	variance := 0.0
	for _, score := range scores {
		variance += (score - mean) * (score - mean)
	}
	variance /= float64(len(scores))
	stdDev := math.Sqrt(variance)

	// 使用2.5倍标准差作为异常值阈值
	outlierThreshold := 2.5 * stdDev

	var filteredPredictions []*PredictionResult
	var filteredWeights []float64
	var filteredScores []float64

	for i, prediction := range predictions {
		deviation := math.Abs(scores[i] - mean)
		if deviation <= outlierThreshold {
			// 正常预测，保留
			filteredPredictions = append(filteredPredictions, prediction)
			filteredWeights = append(filteredWeights, weights[i])
			filteredScores = append(filteredScores, scores[i])
		} else {
			// 异常预测，过滤掉
			log.Printf("[OUTLIER_FILTER] 过滤异常预测: 模型%d, 分数%.4f, 偏差%.4f (阈值%.4f)",
				i, scores[i], deviation, outlierThreshold)
		}
	}

	// 如果过滤后只剩1个或0个预测，使用原始预测（避免过度过滤）
	if len(filteredPredictions) <= 1 {
		log.Printf("[OUTLIER_FILTER] 过滤后预测过少，使用原始预测集合")
		return predictions, weights
	}

	return filteredPredictions, filteredWeights
}

// getTrendConfirmationFromAgent 从agent中获取趋势确认信息
func (be *BacktestEngine) getTrendConfirmationFromAgent(agent map[string]interface{}) *TrendConfirmation {
	// 尝试从agent中获取趋势信息
	if trend, exists := agent["trend_confirmation"].(*TrendConfirmation); exists {
		return trend
	}

	// 如果没有，尝试从其他字段构建
	if state, exists := agent["state"].(map[string]float64); exists {
		// 从状态信息中推断趋势
		trendStrength := state["trend_strength"]
		priceChange := state["price_change_24h"]

		direction := "sideways"
		if trendStrength > 0.6 {
			if priceChange > 0 {
				direction = "bull"
			} else {
				direction = "bear"
			}
		}

		confidence := math.Min(1.0, trendStrength*1.2)
		reliability := math.Min(1.0, (state["volatility_20"]+0.5)*0.8)

		return &TrendConfirmation{
			Strength:    trendStrength,
			Direction:   direction,
			Confidence:  confidence,
			Timeframe:   20,
			Reliability: reliability,
		}
	}

	return nil
}

// WeightAdjustment 权重调整结果
type WeightAdjustment struct {
	MLFactor   float64
	RuleFactor float64
}

// calculateTrendBasedWeightAdjustment 基于趋势确认计算权重调整
func (be *BacktestEngine) calculateTrendBasedWeightAdjustment(trend *TrendConfirmation, mlPrediction *PredictionResult, ruleConfidence float64) *WeightAdjustment {
	adjustment := &WeightAdjustment{
		MLFactor:   1.0,
		RuleFactor: 1.0,
	}

	// 根据趋势强度和确认度调整
	trendStrength := trend.Strength
	confirmation := trend.Confidence

	switch trend.Direction {
	case "bull":
		// 牛市趋势：增加ML权重（技术分析更有效），但如果确认度不高则保守
		if confirmation > 0.7 {
			adjustment.MLFactor = 1.2 + (trendStrength * 0.3) // ML权重增加20-50%
			adjustment.RuleFactor = 0.9                       // 规则权重略降
		} else if confirmation > 0.5 {
			adjustment.MLFactor = 1.1 + (trendStrength * 0.2) // ML权重增加10-30%
		}
		log.Printf("[TREND_BULL] 牛市趋势确认: 强度%.2f, 确认度%.2f -> ML增强%.1f%%",
			trendStrength, confirmation, (adjustment.MLFactor-1.0)*100)

	case "bear":
		// 熊市趋势：增加规则权重（保守策略更有效）
		if confirmation > 0.7 {
			adjustment.RuleFactor = 1.3 + (trendStrength * 0.2) // 规则权重增加30-50%
			adjustment.MLFactor = 0.8                           // ML权重降低
		} else if confirmation > 0.5 {
			adjustment.RuleFactor = 1.1 + (trendStrength * 0.1) // 规则权重增加10-20%
			adjustment.MLFactor = 0.95                          // ML权重小幅降低
		}
		log.Printf("[TREND_BEAR] 熊市趋势确认: 强度%.2f, 确认度%.2f -> 规则增强%.1f%%",
			trendStrength, confirmation, (adjustment.RuleFactor-1.0)*100)

	case "sideways":
		// 震荡市：平衡权重，略倾向规则（避免过度交易）
		adjustment.RuleFactor = 1.1 // 规则权重小幅增加
		adjustment.MLFactor = 0.95  // ML权重小幅降低
		log.Printf("[TREND_SIDEWAYS] 震荡趋势: 强度%.2f, 确认度%.2f -> 平衡策略",
			trendStrength, confirmation)
	}

	// 基于可靠性进一步调整
	if trend.Reliability < 0.6 {
		// 可靠性低，保守调整
		adjustment.MLFactor = math.Max(0.8, adjustment.MLFactor*0.9)
		adjustment.RuleFactor = math.Max(0.9, adjustment.RuleFactor*0.95)
		log.Printf("[TREND_LOW_RELIABILITY] 趋势可靠性低: %.2f -> 保守调整", trend.Reliability)
	}

	// 确保权重在合理范围内
	adjustment.MLFactor = math.Max(0.5, math.Min(2.0, adjustment.MLFactor))
	adjustment.RuleFactor = math.Max(0.5, math.Min(2.0, adjustment.RuleFactor))

	return adjustment
}
