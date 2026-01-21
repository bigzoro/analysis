package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// generateRecommendations 生成推荐（当前日期）
func (s *Server) generateRecommendations(ctx context.Context, kind string, limit int) ([]pdb.CoinRecommendation, error) {
	return s.generateRecommendationsForDate(ctx, kind, limit, time.Now().UTC())
}

// generateRecommendationsForDate 为指定日期生成推荐（使用历史数据）
func (s *Server) generateRecommendationsForDate(ctx context.Context, kind string, limit int, targetDate time.Time) ([]pdb.CoinRecommendation, error) {
	// 使用新一代选币算法
	if s.coinSelectionAlgorithm != nil {
		return s.generateRecommendationsWithNewAlgorithm(ctx, kind, limit, targetDate)
	}

	// 降级到原有算法
	log.Printf("[INFO] New algorithm not available, falling back to legacy algorithm")
	return s.generateRecommendationsWithLegacyAlgorithm(ctx, kind, limit, targetDate)
}

// generateRecommendationsWithNewAlgorithm 使用新算法生成推荐
func (s *Server) generateRecommendationsWithNewAlgorithm(ctx context.Context, kind string, limit int, targetDate time.Time) ([]pdb.CoinRecommendation, error) {
	log.Printf("[INFO] Using new coin selection algorithm for %s recommendations", kind)

	// 1. 获取市场数据
	marketData, err := s.getMarketDataForAlgorithm(ctx, kind, targetDate)
	if err != nil {
		log.Printf("[WARN] Failed to get market data for new algorithm: %v", err)
		return s.generateRecommendationsWithLegacyAlgorithm(ctx, kind, limit, targetDate)
	}

	if len(marketData) == 0 {
		log.Printf("[WARN] No market data available, falling back to legacy algorithm")
		return s.generateRecommendationsWithLegacyAlgorithm(ctx, kind, limit, targetDate)
	}

	// 2. 获取市场状态（使用实际的市场数据分析）
	marketAnalysis := s.coinSelectionAlgorithm.GetMarketAnalyzer().AnalyzeMarketState(marketData)

	// 转换为MarketState格式
	marketState := MarketState{
		State:        marketAnalysis.State,
		AvgChange:    marketAnalysis.AvgChange,
		UpRatio:      marketAnalysis.UpRatio,
		Volatility:   marketAnalysis.Volatility,
		VolumeChange: marketAnalysis.VolumeChange,
	}

	// 3. 执行选币算法
	recommendations, err := s.coinSelectionAlgorithm.SelectCoins(ctx, marketData, marketState)
	if err != nil {
		log.Printf("[WARN] New algorithm failed: %v, falling back to legacy", err)
		return s.generateRecommendationsWithLegacyAlgorithm(ctx, kind, limit, targetDate)
	}

	// 4. 转换为数据库格式
	return s.convertAlgorithmResultsToDBFormat(recommendations, kind, limit)
}

// convertAlgorithmResultsToDBFormat 将算法结果转换为数据库格式
func (s *Server) convertAlgorithmResultsToDBFormat(recommendations []CoinRecommendation, kind string, limit int) ([]pdb.CoinRecommendation, error) {
	dbRecs := make([]pdb.CoinRecommendation, 0, len(recommendations))

	for i, rec := range recommendations {
		if i >= limit {
			break
		}

		// 生成推荐理由
		reasons := rec.Reasons
		if len(reasons) == 0 {
			reasons = []string{"基于AI算法智能推荐"}
		}

		// 将字符串数组转换为JSON格式的理由
		reasonsJSON, _ := json.Marshal(reasons)

		dbRec := pdb.CoinRecommendation{
			Symbol:         rec.Symbol,
			Kind:           kind,
			TotalScore:     rec.TotalScore,
			MarketScore:    rec.Scores.Fundamental, // 使用Fundamental作为Market分数
			FlowScore:      rec.Scores.Momentum,    // 使用Momentum作为Flow分数
			HeatScore:      rec.Scores.Technical,   // 使用Technical作为Heat分数
			EventScore:     0.5,                    // 默认中等事件分数
			SentimentScore: rec.Scores.Sentiment,
			Reasons:        datatypes.JSON(reasonsJSON),
			Rank:           rec.Rank,
			CreatedAt:      rec.RecommendedAt,
		}

		dbRecs = append(dbRecs, dbRec)
	}

	return dbRecs, nil
}

// getAnnouncementData 获取公告数据
func (s *Server) getAnnouncementData(ctx context.Context) (map[string]bool, error) {
	return s.getAnnouncementDataForRecommendation(ctx)
}

// getSentimentData 获取情绪数据
func (s *Server) getSentimentData(ctx context.Context) (map[string]*SentimentResult, error) {
	// 这里简化实现，实际应该从数据库或缓存获取
	// 暂时返回空map，表示没有情绪数据
	return make(map[string]*SentimentResult), nil
}

// generateRecommendationsWithLegacyAlgorithm 使用传统算法生成推荐
func (s *Server) generateRecommendationsWithLegacyAlgorithm(ctx context.Context, kind string, limit int, targetDate time.Time) ([]pdb.CoinRecommendation, error) {
	log.Printf("[INFO] Using legacy recommendation algorithm for %s recommendations", kind)

	// 1. 获取市场数据
	marketData, err := s.getMarketDataForAlgorithm(ctx, kind, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data available for legacy algorithm")
	}

	// 2. 获取资金流数据
	flowData, err := s.getFlowDataForRecommendation(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get flow data: %v", err)
		flowData = make(map[string]float64)
	}

	// 3. 获取公告数据
	announcementData, err := s.getAnnouncementData(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get announcement data: %v", err)
		announcementData = make(map[string]bool)
	}

	// 4. 获取情绪数据
	sentimentData, err := s.getSentimentData(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get sentiment data: %v", err)
	}

	// 5. 分析市场状态
	marketState := s.analyzeMarketStateFromDataPoints(marketData)

	// 6. 计算动态权重
	weights := s.calculateDynamicWeights(marketState)

	// 7. 计算每个币种的评分
	var scores []RecommendationScore
	for _, item := range marketData {
		symbol := item.Symbol

		// 获取该币种的辅助数据
		var flowTrend *FlowTrendResult
		var announcementScore *AnnouncementScore

		// 将 MarketDataPoint 转换为 pdb.BinanceMarketTop 格式
		dbItem := pdb.BinanceMarketTop{
			Symbol:    item.Symbol,
			LastPrice: fmt.Sprintf("%.8f", item.Price),
			PctChange: item.PriceChange24h,
			Volume:    fmt.Sprintf("%.8f", item.Volume24h),
		}
		if item.MarketCap != nil {
			dbItem.MarketCapUSD = item.MarketCap
		}

		// 这里简化实现，实际应该从缓存或数据库获取
		score := s.calculateScore(dbItem, flowData, announcementData, sentimentData[symbol], flowTrend, announcementScore, weights, marketState)
		scores = append(scores, score)
	}

	// 8. 按评分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	// 9. 转换为CoinRecommendation格式
	var recommendations []CoinRecommendation
	for i, score := range scores {
		if i >= limit {
			break
		}

		rec := CoinRecommendation{
			CoinScore: CoinScore{
				Symbol:     score.Symbol,
				TotalScore: score.TotalScore,
				Scores: struct {
					Technical   float64
					Fundamental float64
					Sentiment   float64
					Risk        float64
					Momentum    float64
				}{
					Technical:   score.Scores.Technical,
					Fundamental: score.Scores.Market, // 使用Market分数作为Fundamental
					Sentiment:   score.Scores.Sentiment,
					Risk:        score.Scores.Risk,
					Momentum:    score.Scores.Momentum,
				},
				Reasons: score.Reasons,
			},
			Rank:          i + 1,
			RecommendedAt: time.Now().UTC(),
		}
		recommendations = append(recommendations, rec)
	}

	// 10. 转换为数据库格式
	return s.convertAlgorithmResultsToDBFormat(recommendations, kind, limit)
}

// analyzeMarketStateFromDataPoints 从数据点分析市场状态
func (s *Server) analyzeMarketStateFromDataPoints(dataPoints []MarketDataPoint) MarketState {
	if len(dataPoints) == 0 {
		return MarketState{State: "neutral", AvgChange: 0, UpRatio: 0.5, Volatility: 0.5, VolumeChange: 0}
	}

	totalChange := 0.0
	upCount := 0
	totalVolatility := 0.0
	totalVolumeChange := 0.0

	for _, point := range dataPoints {
		totalChange += point.PriceChange24h
		if point.PriceChange24h > 0 {
			upCount++
		}
		totalVolatility += math.Abs(point.PriceChange24h)
		// 简化的成交量变化计算
		totalVolumeChange += point.Volume24h / 1000000 // 归一化
	}

	avgChange := totalChange / float64(len(dataPoints))
	upRatio := float64(upCount) / float64(len(dataPoints))
	avgVolatility := totalVolatility / float64(len(dataPoints))
	avgVolumeChange := totalVolumeChange / float64(len(dataPoints))

	// 确定市场状态
	var state string
	if avgChange > 0.05 && upRatio > 0.6 {
		state = "bull"
	} else if avgChange < -0.05 && upRatio < 0.4 {
		state = "bear"
	} else {
		state = "neutral"
	}

	return MarketState{
		State:        state,
		AvgChange:    avgChange,
		UpRatio:      upRatio,
		Volatility:   avgVolatility,
		VolumeChange: avgVolumeChange,
	}
}

// analyzeMarketState 分析市场状态
func (s *Server) analyzeMarketState(candidates []pdb.BinanceMarketTop) MarketState {
	if len(candidates) == 0 {
		return MarketState{State: "neutral", AvgChange: 0, UpRatio: 0.5, Volatility: 0.5, VolumeChange: 0}
	}

	totalChange := 0.0
	upCount := 0
	totalVolatility := 0.0

	for _, item := range candidates {
		change := item.PctChange
		totalChange += change
		if change > 0 {
			upCount++
		}
		totalVolatility += math.Abs(change)
	}

	avgChange := totalChange / float64(len(candidates))
	upRatio := float64(upCount) / float64(len(candidates))
	avgVolatility := totalVolatility / float64(len(candidates))

	// 确定市场状态
	var state string
	if avgChange > 2.0 && upRatio > 0.6 {
		state = "bull"
	} else if avgChange < -2.0 && upRatio < 0.4 {
		state = "bear"
	} else {
		state = "neutral"
	}

	return MarketState{
		State:      state,
		AvgChange:  avgChange,
		UpRatio:    upRatio,
		Volatility: avgVolatility,
	}
}

// calculateDynamicWeights 根据市场状态计算动态权重
func (s *Server) calculateDynamicWeights(marketState MarketState) DynamicWeights {
	weights := DynamicWeights{
		MarketWeight:    0.25,
		FlowWeight:      0.25,
		HeatWeight:      0.20,
		EventWeight:     0.15,
		SentimentWeight: 0.15,
	}

	// 根据市场状态调整权重
	switch marketState.State {
	case "bull":
		// 牛市中，资金流和情绪因子更重要
		weights.FlowWeight += 0.1
		weights.SentimentWeight += 0.05
		weights.MarketWeight -= 0.1
		weights.HeatWeight -= 0.05

	case "bear":
		// 熊市中，事件和市场表现因子更重要
		weights.EventWeight += 0.1
		weights.MarketWeight += 0.1
		weights.FlowWeight -= 0.1
		weights.SentimentWeight -= 0.05

	case "neutral":
		// 中性市场，均衡调整
		// 保持默认权重
	}

	// 确保权重总和为1
	total := weights.MarketWeight + weights.FlowWeight + weights.HeatWeight + weights.EventWeight + weights.SentimentWeight
	if total > 0 {
		weights.MarketWeight /= total
		weights.FlowWeight /= total
		weights.HeatWeight /= total
		weights.EventWeight /= total
		weights.SentimentWeight /= total
	}

	return weights
}

// calculateScore 计算推荐得分
func (s *Server) calculateScore(
	item pdb.BinanceMarketTop,
	flowData map[string]float64,
	announcementData map[string]bool,
	sentimentData *SentimentResult,
	flowTrendData *FlowTrendResult,
	announcementScore *AnnouncementScore,
	weights DynamicWeights,
	marketState MarketState,
) RecommendationScore {
	symbol := item.Symbol
	baseSymbol := extractBaseSymbol(symbol)

	// 基础市场数据评分
	marketScore := s.calculateMarketScore(item, marketState)

	// 资金流评分
	flowScore := 0.5 // 默认中等评分
	if flow, ok := flowData[symbol]; ok {
		flowScore = math.Min(math.Max(flow/1000000, 0), 1) // 归一化到0-1
	}

	// 热度评分（基于成交量和市值）
	heatScore := s.calculateHeatScore(item)

	// 事件评分（基于公告）
	eventScore := 0.0
	if announcementData[symbol] {
		eventScore = 1.0
	}
	if announcementScore != nil {
		eventScore = math.Max(eventScore, announcementScore.TotalScore)
	}

	// 情绪评分
	sentimentScore := 0.5 // 默认中等评分
	if sentimentData != nil {
		sentimentScore = sentimentData.Score / 10.0 // 归一化到0-1
	}

	// 计算综合评分
	totalScore := marketScore*weights.MarketWeight +
		flowScore*weights.FlowWeight +
		heatScore*weights.HeatWeight +
		eventScore*weights.EventWeight +
		sentimentScore*weights.SentimentWeight

	// 确定策略类型
	strategyType := s.determineStrategyType(marketScore, flowScore, heatScore)

	// 生成推荐理由
	reasons := s.generateReasons(RecommendationScore{
		Symbol:       symbol,
		BaseSymbol:   baseSymbol,
		TotalScore:   totalScore,
		StrategyType: strategyType,
		Scores: Scores{
			Market:      marketScore,
			Flow:        flowScore,
			Heat:        heatScore,
			Event:       eventScore,
			Sentiment:   sentimentScore,
			Fundamental: 0, // 暂时设为0
		},
	}, 0)

	return RecommendationScore{
		Symbol:       symbol,
		BaseSymbol:   baseSymbol,
		TotalScore:   totalScore,
		StrategyType: strategyType,
		Scores: Scores{
			Market:      marketScore,
			Flow:        flowScore,
			Heat:        heatScore,
			Event:       eventScore,
			Sentiment:   sentimentScore,
			Fundamental: 0, // 暂时设为0
		},
		Reasons:             reasons,
		Confidence:          totalScore,
		RiskLevel:           s.calculateRiskLevel(totalScore),
		ExpectedReturn:      totalScore * 0.15, // 简化的预期收益
		RecommendedPosition: totalScore * 0.2,  // 简化的仓位建议
		MarketCap:           0,                 // 暂时不使用
		Volume24h:           0,                 // 暂时不使用
		PriceChange24h:      0,                 // 暂时不使用
		LastUpdated:         time.Now(),
	}
}

// calculateMarketScore 计算市场表现评分
func (s *Server) calculateMarketScore(item pdb.BinanceMarketTop, marketState MarketState) float64 {
	priceChange := item.PctChange

	// 基础评分基于价格变化
	baseScore := 0.5 + priceChange/200 // -100%到+100%的变化归一化到0-1

	// 根据市场状态调整评分
	switch marketState.State {
	case "bull":
		if priceChange > 0 {
			baseScore += 0.2 // 牛市中上涨更重要
		}
	case "bear":
		if priceChange < 0 {
			baseScore += 0.1 // 熊市中下跌幅度较小相对较好
		}
	}

	return math.Min(math.Max(baseScore, 0), 1)
}

// calculateHeatScore 计算热度评分
func (s *Server) calculateHeatScore(item pdb.BinanceMarketTop) float64 {
	// 基于成交量和市值计算热度
	volume, _ := strconv.ParseFloat(item.Volume, 64)
	marketCap := 0.0
	if item.MarketCapUSD != nil {
		marketCap = *item.MarketCapUSD
	}

	// 归一化成交量（假设平均成交量为1亿）
	volumeScore := math.Min(volume/100000000, 1)

	// 归一化市值（假设大盘币种市值更高）
	marketCapScore := 0.5
	if marketCap > 10000000000 { // 100亿市值
		marketCapScore = 1.0
	} else if marketCap > 1000000000 { // 10亿市值
		marketCapScore = 0.8
	} else if marketCap > 100000000 { // 1亿市值
		marketCapScore = 0.6
	}

	return (volumeScore + marketCapScore) / 2
}

// determineStrategyType 确定策略类型
func (s *Server) determineStrategyType(marketScore, flowScore, heatScore float64) string {
	if marketScore > 0.7 && flowScore > 0.6 {
		return "LONG" // 强势上涨
	} else if marketScore < 0.3 && heatScore < 0.4 {
		return "RANGE" // 震荡整理
	} else {
		return "SHORT" // 适度回调
	}
}

// calculateRiskLevel 计算风险等级
func (s *Server) calculateRiskLevel(score float64) string {
	if score >= 0.8 {
		return "low"
	} else if score >= 0.6 {
		return "medium"
	} else {
		return "high"
	}
}

// formatRecommendations 格式化推荐结果
func formatRecommendations(recs []pdb.CoinRecommendation, s *Server, ctx context.Context) []gin.H {
	formatted := make([]gin.H, 0, len(recs))

	for _, rec := range recs {
		// 获取当前价格
		currentPrice := 0.0
		if price, err := s.getCurrentPrice(ctx, rec.Symbol, rec.Kind); err == nil {
			currentPrice = price
		}

		// 解析推荐理由
		var reasons []string
		if err := json.Unmarshal(rec.Reasons, &reasons); err != nil {
			// 如果解析失败，使用默认理由
			reasons = []string{"基于综合分析"}
		}
		if len(reasons) == 0 {
			reasons = []string{"基于综合分析"}
		}

		item := gin.H{
			"symbol":          rec.Symbol,
			"rank":            rec.Rank,
			"total_score":     rec.TotalScore,
			"market_score":    rec.MarketScore,
			"flow_score":      rec.FlowScore,
			"heat_score":      rec.HeatScore,
			"event_score":     rec.EventScore,
			"sentiment_score": rec.SentimentScore,
			"current_price":   currentPrice,
			"reasons":         reasons,
			"generated_at":    rec.CreatedAt,
			"prediction":      nil, // 暂时不支持价格预测
		}

		formatted = append(formatted, item)
	}

	return formatted
}

// calculateScoreWithKind 计算币种得分（带kind参数，用于获取技术指标）
func (s *Server) calculateScoreWithKind(
	ctx context.Context,
	item pdb.BinanceMarketTop,
	baseSymbol string,
	flowData map[string]float64,
	announcementData map[string]bool,
	sentimentData *SentimentResult,
	flowTrendData *FlowTrendResult,
	announcementScore *AnnouncementScore,
	kind string,
	currentPrice float64,
	strategyType string, // 新增策略类型参数
) RecommendationScore {
	// 构造市场状态（简化处理）
	marketState := s.analyzeMarketState([]pdb.BinanceMarketTop{item})

	// 构造动态权重
	weights := s.calculateDynamicWeights(marketState)

	// 先计算基础得分
	score := s.calculateScore(item, flowData, announcementData, sentimentData, flowTrendData, announcementScore, weights, marketState)

	// 设置基础信息
	score.BaseSymbol = baseSymbol
	score.StrategyType = strategyType

	// 获取技术指标
	// 优先使用Binance API，失败则使用历史快照数据
	technical, err := s.CalculateTechnicalIndicators(ctx, item.Symbol, kind)
	if err != nil {
		// 如果API获取失败，尝试使用历史数据
		technical, _ = s.GetTechnicalIndicatorsFromHistory(ctx, item.Symbol, kind)
	}
	score.Technical = technical

	// 根据技术指标调整得分
	if technical != nil {
		adjustment := 1.0

		// RSI调整：RSI在30-70之间为健康，超出范围降低得分
		if technical.RSI > 70 {
			adjustment *= 0.9 // 超买，降低得分
		} else if technical.RSI < 30 {
			adjustment *= 1.05 // 超卖，可能反弹
		}

		// MACD调整：MACD > Signal 为买入信号
		if technical.MACD > technical.MACDSignal && technical.MACDHist > 0 {
			adjustment *= 1.1 // 金叉
		} else if technical.MACD < technical.MACDSignal && technical.MACDHist < 0 {
			adjustment *= 0.9 // 死叉
		}

		// 布林带调整
		if technical.BBPosition < 0.2 {
			adjustment *= 1.08 // 接近下轨，可能反弹
		} else if technical.BBPosition > 0.8 {
			adjustment *= 0.92 // 接近上轨，可能回调
		}

		// KDJ调整
		if technical.K > technical.D && technical.K < 80 {
			adjustment *= 1.05 // KDJ金叉
		} else if technical.K < technical.D && technical.K > 20 {
			adjustment *= 0.95 // KDJ死叉
		}

		// 均线系统调整
		if technical.MA5 > technical.MA10 && technical.MA10 > technical.MA20 {
			adjustment *= 1.1 // 多头排列
		} else if technical.MA5 < technical.MA10 && technical.MA10 < technical.MA20 {
			adjustment *= 0.9 // 空头排列
		}

		// 成交量调整
		if technical.VolumeRatio > 1.5 {
			adjustment *= 1.05 // 成交量放大，可能突破
		} else if technical.VolumeRatio < 0.5 {
			adjustment *= 0.95 // 成交量萎缩
		}

		// 支撑位/阻力位调整
		if technical.SupportLevel > 0 {
			distanceToSupport := (currentPrice - technical.SupportLevel) / currentPrice * 100
			if distanceToSupport < 2 && distanceToSupport > 0 {
				adjustment *= 1.05 // 接近支撑位，可能反弹
			}
		}
		if technical.ResistanceLevel > 0 {
			distanceToResistance := (technical.ResistanceLevel - currentPrice) / currentPrice * 100
			if distanceToResistance < 2 && distanceToResistance > 0 {
				adjustment *= 0.95 // 接近阻力位，可能回调
			}
		}

		// 趋势调整
		if technical.Trend == "up" {
			adjustment *= 1.05
		} else if technical.Trend == "down" {
			adjustment *= 0.95
		}

		// 应用调整
		score.Scores.Market *= adjustment
	}

	// 重新计算总分（技术指标调整后）
	score.TotalScore = score.Scores.Market + score.Scores.Flow +
		score.Scores.Heat + score.Scores.Event + score.Scores.Sentiment

	// 计算风险评级（使用现有的风险等级计算）
	score.RiskLevel = s.calculateRiskLevel(score.TotalScore)

	// 生成推荐理由
	score.Reasons = s.generateReasons(score, currentPrice)

	return score
}

// generateRecommendationsSimple 简单推荐生成方法（原有方法，作为降级方案）
func (s *Server) generateRecommendationsSimple(ctx context.Context, kind string, limit int) ([]pdb.CoinRecommendation, error) {
	return s.generateRecommendationsSimpleForDate(ctx, kind, limit, time.Now().UTC())
}

// generateRecommendationsSimpleForDate 为指定日期生成推荐（简单方法）
func (s *Server) generateRecommendationsSimpleForDate(ctx context.Context, kind string, limit int, targetDate time.Time) ([]pdb.CoinRecommendation, error) {
	// 1. 获取市场数据（相对于目标日期的涨幅榜）
	// 计算目标日期当天的开始和结束时间
	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location()).UTC()
	dayEnd := dayStart.Add(24 * time.Hour)
	startTime := dayStart.Add(-24 * time.Hour) // 扩大到24小时，获取更多数据
	endTime := dayEnd

	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), kind, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	if len(snaps) == 0 {
		log.Printf("[WARN] Simple method: 没有找到市场快照数据（时间范围：%s 到 %s，kind=%s，targetDate=%s）", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), kind, targetDate.Format(time.RFC3339))

		// 尝试查询更早的数据
		startTime = dayStart.Add(-7 * 24 * time.Hour)
		snaps, tops, err = pdb.ListBinanceMarket(s.db.DB(), kind, startTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("获取市场数据失败: %w", err)
		}

		if len(snaps) == 0 {
			log.Printf("[ERROR] Simple method: 7天内都没有找到市场快照数据")
			return []pdb.CoinRecommendation{}, fmt.Errorf("没有找到市场快照数据，请确保market_scanner正在运行")
		}

		log.Printf("[INFO] Simple method: 使用历史快照数据（%d个快照）", len(snaps))
	}

	// 获取最新的快照
	latestSnap := snaps[len(snaps)-1]
	candidates := tops[latestSnap.ID]

	// 2. 获取黑名单
	blacklist, err := s.db.GetBinanceBlacklist(kind)
	if err != nil {
		log.Printf("[WARN] Failed to get blacklist: %v", err)
		blacklist = []string{}
	}
	blacklistMap := make(map[string]bool)
	for _, sym := range blacklist {
		blacklistMap[strings.ToUpper(sym)] = true
	}

	// 3. 获取资金流数据（最近24小时）
	flowData, err := s.getFlowDataForRecommendation(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get flow data: %v", err)
		flowData = make(map[string]float64) // 空数据
	}

	// 3.5. 获取资金流趋势数据
	baseSymbolsForFlow := make([]string, 0, len(candidates))
	baseSymbolSetForFlow := make(map[string]bool)
	for _, item := range candidates {
		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol != "" && !baseSymbolSetForFlow[baseSymbol] {
			baseSymbolsForFlow = append(baseSymbolsForFlow, baseSymbol)
			baseSymbolSetForFlow[baseSymbol] = true
		}
	}

	flowTrendDataMap, err := s.GetFlowTrendForSymbols(ctx, baseSymbolsForFlow)
	if err != nil {
		log.Printf("[WARN] Failed to get flow trend data: %v", err)
		flowTrendDataMap = make(map[string]*FlowTrendResult)
	}

	// 4. 获取公告数据
	announcementData, err := s.getAnnouncementDataForRecommendation(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get announcement data: %v", err)
		announcementData = make(map[string]bool)
	}

	baseSymbolsForAnnouncement := make([]string, 0, len(candidates))
	baseSymbolSetForAnnouncement := make(map[string]bool)
	for _, item := range candidates {
		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol != "" && !baseSymbolSetForAnnouncement[baseSymbol] {
			baseSymbolsForAnnouncement = append(baseSymbolsForAnnouncement, baseSymbol)
			baseSymbolSetForAnnouncement[baseSymbol] = true
		}
	}

	announcementScores, err := s.GetAnnouncementScoresForSymbols(ctx, baseSymbolsForAnnouncement, 7)
	if err != nil {
		log.Printf("[WARN] Failed to get announcement scores: %v", err)
		announcementScores = make(map[string]*AnnouncementScore)
	}

	// 4.5. 获取Twitter情绪数据
	baseSymbols := make([]string, 0, len(candidates))
	baseSymbolSet := make(map[string]bool)
	for _, item := range candidates {
		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol != "" && !baseSymbolSet[baseSymbol] {
			baseSymbols = append(baseSymbols, baseSymbol)
			baseSymbolSet[baseSymbol] = true
		}
	}

	twitterSentimentData, err := s.GetTwitterSentimentForSymbols(ctx, baseSymbols)
	if err != nil {
		log.Printf("[WARN] Failed to get Twitter sentiment data: %v", err)
		twitterSentimentData = make(map[string]*SentimentResult)
	}

	// 5. 计算每个币种的得分
	scores := make([]RecommendationScore, 0, len(candidates))
	for _, item := range candidates {
		if blacklistMap[strings.ToUpper(item.Symbol)] {
			continue
		}

		baseSymbol := extractBaseSymbol(item.Symbol)

		var sentimentData *SentimentResult
		if data, ok := twitterSentimentData[baseSymbol]; ok {
			sentimentData = data
		}

		var flowTrendData *FlowTrendResult
		if data, ok := flowTrendDataMap[baseSymbol]; ok {
			flowTrendData = data
		}

		var announcementScore *AnnouncementScore
		if score, ok := announcementScores[baseSymbol]; ok {
			announcementScore = score
		}

		// 获取当前价格
		currentPrice, _ := strconv.ParseFloat(item.LastPrice, 64)
		if currentPrice == 0 {
			// 如果没有价格，尝试从API获取
			if price, err := s.getCurrentPrice(ctx, item.Symbol, kind); err == nil {
				currentPrice = price
			}
		}

		score := s.calculateScoreWithKind(ctx, item, baseSymbol, flowData, announcementData, sentimentData, flowTrendData, announcementScore, kind, currentPrice, "LONG")

		prediction, err := s.GetPricePrediction(ctx, item.Symbol, kind)
		if err == nil && prediction != nil {
			score.Prediction = prediction
		}

		if score.TotalScore > 0 {
			scores = append(scores, score)
		}
	}

	// 6. 分析市场状态并计算动态权重
	marketState := s.analyzeMarketState(candidates)
	performanceWeights, err := s.AdjustWeightsByPerformance(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to adjust weights by performance: %v", err)
		performanceWeights = nil
	}

	var weights DynamicWeights
	if performanceWeights != nil {
		baseWeights := s.calculateDynamicWeights(marketState)
		weights = DynamicWeights{
			MarketWeight:    performanceWeights.MarketWeight*0.7 + baseWeights.MarketWeight*0.3,
			FlowWeight:      performanceWeights.FlowWeight*0.7 + baseWeights.FlowWeight*0.3,
			HeatWeight:      performanceWeights.HeatWeight*0.7 + baseWeights.HeatWeight*0.3,
			EventWeight:     performanceWeights.EventWeight*0.7 + baseWeights.EventWeight*0.3,
			SentimentWeight: performanceWeights.SentimentWeight*0.7 + baseWeights.SentimentWeight*0.3,
		}
		total := weights.MarketWeight + weights.FlowWeight + weights.HeatWeight + weights.EventWeight + weights.SentimentWeight
		if total > 0 {
			weights.MarketWeight /= total
			weights.FlowWeight /= total
			weights.HeatWeight /= total
			weights.EventWeight /= total
			weights.SentimentWeight /= total
		}
	} else {
		weights = s.calculateDynamicWeights(marketState)
	}

	for i := range scores {
		scores[i].TotalScore = scores[i].Scores.Market*weights.MarketWeight +
			scores[i].Scores.Flow*weights.FlowWeight +
			scores[i].Scores.Heat*weights.HeatWeight +
			scores[i].Scores.Event*weights.EventWeight +
			scores[i].Scores.Sentiment*weights.SentimentWeight
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	if len(scores) > limit {
		scores = scores[:limit]
	}

	recommendations := make([]pdb.CoinRecommendation, 0, len(scores))
	for i, score := range scores {
		reasonsJSON, _ := json.Marshal(score.Reasons)
		riskWarningsJSON, _ := json.Marshal(score.Risk.RiskWarnings)

		var technicalJSON []byte
		if score.Technical != nil {
			technicalJSON, _ = json.Marshal(score.Technical)
		}

		// 序列化价格预测
		var predictionJSON []byte
		if score.Prediction != nil {
			predictionJSON, _ = json.Marshal(score.Prediction)
		}

		rec := pdb.CoinRecommendation{
			Kind:                kind,
			Symbol:              score.Symbol,
			BaseSymbol:          score.BaseSymbol,
			Rank:                i + 1,
			TotalScore:          score.TotalScore,
			StrategyType:        score.StrategyType,
			MarketScore:         score.Scores.Market,
			FlowScore:           score.Scores.Flow,
			HeatScore:           score.Scores.Heat,
			EventScore:          score.Scores.Event,
			SentimentScore:      score.Scores.Sentiment,
			PriceChange24h:      &score.Data.PriceChange24h,
			Volume24h:           &score.Data.Volume24h,
			MarketCapUSD:        score.Data.MarketCapUSD,
			NetFlow24h:          &score.Data.NetFlow24h,
			HasNewListing:       score.Data.HasNewListing,
			HasAnnouncement:     score.Data.HasAnnouncement,
			Reasons:             reasonsJSON,
			VolatilityRisk:      &score.Risk.VolatilityRisk,
			LiquidityRisk:       &score.Risk.LiquidityRisk,
			MarketRisk:          &score.Risk.MarketRisk,
			TechnicalRisk:       &score.Risk.TechnicalRisk,
			OverallRisk:         &score.Risk.OverallRisk,
			RiskLevel:           &score.Risk.RiskLevel,
			RiskWarnings:        riskWarningsJSON,
			TechnicalIndicators: technicalJSON,
			PricePrediction:     predictionJSON,
			GeneratedAt:         time.Now().UTC(), // 使用当前时间作为生成时间
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}
