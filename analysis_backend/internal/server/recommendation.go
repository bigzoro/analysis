package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// DynamicWeights 动态权重结构
type DynamicWeights struct {
	MarketWeight    float64 // 市场表现权重
	FlowWeight      float64 // 资金流权重
	HeatWeight      float64 // 市场热度权重
	EventWeight     float64 // 事件权重
	SentimentWeight float64 // 情绪权重
}

// MarketState 市场状态
type MarketState struct {
	State          string  // "bull" / "bear" / "sideways"
	AvgChange      float64 // 平均涨幅
	UpRatio        float64 // 上涨币种比例
	Volatility     float64 // 波动率
	VolumeChange   float64 // 成交量变化
}

// RecommendationScore 推荐评分结构
type RecommendationScore struct {
	Symbol     string
	BaseSymbol string
	TotalScore float64
	Scores     struct {
		Market    float64
		Flow      float64
		Heat      float64
		Event     float64
		Sentiment float64
	}
	Data struct {
		Price           float64
		PriceChange24h  float64
		Volume24h       float64
		MarketCapUSD    *float64
		NetFlow24h      float64
		HasNewListing   bool
		HasAnnouncement bool
		TwitterMentions int
		FlowTrend       *FlowTrendResult      // 资金流趋势数据
		AnnouncementScore *AnnouncementScore  // 公告重要性得分
	}
	Technical *TechnicalIndicators // 技术指标
	Prediction *PricePrediction    // 价格预测
	Risk struct {
		VolatilityRisk  float64 // 波动率风险 0-100
		LiquidityRisk   float64 // 流动性风险 0-100
		MarketRisk      float64 // 市场风险 0-100
		TechnicalRisk   float64 // 技术风险 0-100
		OverallRisk     float64 // 综合风险 0-100
		RiskLevel       string  // "low"/"medium"/"high"
		RiskWarnings    []string // 风险提示
	}
	Reasons []string
}

// GetCoinRecommendations 获取币种推荐
// GET /recommendations/coins?kind=spot&limit=5&refresh=false
func (s *Server) GetCoinRecommendations(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	if kind != "spot" && kind != "futures" {
		kind = "spot"
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 10 {
			limit = n
		}
	}

	refresh := c.DefaultQuery("refresh", "false") == "true"

	// 尝试从缓存获取
	if !refresh && s.cache != nil {
		cacheKey := fmt.Sprintf("cache:recommendations:%s", kind)
		cached, err := s.cache.Get(c.Request.Context(), cacheKey)
		if err == nil && len(cached) > 0 {
			var cachedData struct {
				GeneratedAt     time.Time                `json:"generated_at"`
				Recommendations []pdb.CoinRecommendation `json:"recommendations"`
			}
			if err := json.Unmarshal(cached, &cachedData); err == nil {
				// 检查缓存是否过期（5分钟）
				if time.Since(cachedData.GeneratedAt) < 5*time.Minute {
					// 格式化推荐结果
					formattedRecs := formatRecommendations(cachedData.Recommendations)
					
					// 为缓存结果也添加价格预测（实时计算）
					ctx := c.Request.Context()
					for i, rec := range formattedRecs {
						symbol, ok := rec["symbol"].(string)
						if ok && symbol != "" {
							prediction, err := s.GetPricePrediction(ctx, symbol, kind)
							if err == nil && prediction != nil {
								formattedRecs[i]["prediction"] = prediction
							}
						}
					}
					
					c.JSON(http.StatusOK, gin.H{
						"generated_at":    cachedData.GeneratedAt,
						"kind":            kind,
						"recommendations": formattedRecs,
						"cached":          true,
					})
					return
				}
			}
		}
	}

	// 生成新推荐
	recommendations, err := s.generateRecommendations(c.Request.Context(), kind, limit)
	if err != nil {
		s.InternalServerError(c, "生成推荐失败", err)
		return
	}

	// 保存到数据库
	generatedAt := time.Now().UTC()
	if err := pdb.SaveRecommendations(s.db.DB(), kind, generatedAt, recommendations); err != nil {
		log.Printf("[ERROR] Failed to save recommendations: %v", err)
		// 不返回错误，继续返回推荐结果
	}

	// 缓存结果
	if s.cache != nil {
		cacheData := struct {
			GeneratedAt     time.Time                `json:"generated_at"`
			Recommendations []pdb.CoinRecommendation `json:"recommendations"`
		}{
			GeneratedAt:     generatedAt,
			Recommendations: recommendations,
		}
		data, err := json.Marshal(cacheData)
		if err == nil {
			cacheKey := fmt.Sprintf("cache:recommendations:%s", kind)
			if globalCachePool != nil {
				globalCachePool.Submit(func() {
					s.cache.Set(context.Background(), cacheKey, data, 5*time.Minute)
				})
			} else {
				go func() {
					s.cache.Set(context.Background(), cacheKey, data, 5*time.Minute)
				}()
			}
		}
	}

	// 格式化推荐结果
	formattedRecs := formatRecommendations(recommendations)
	
	// 为每个推荐添加价格预测（实时计算，限制并发）
	ctx := c.Request.Context()
	for i, rec := range formattedRecs {
		symbol, ok := rec["symbol"].(string)
		if ok && symbol != "" {
			prediction, err := s.GetPricePrediction(ctx, symbol, kind)
			if err == nil && prediction != nil {
				formattedRecs[i]["prediction"] = prediction
			}
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"generated_at":    generatedAt,
		"kind":            kind,
		"recommendations": formattedRecs,
		"cached":          false,
	})
}

// generateRecommendations 生成推荐
func (s *Server) generateRecommendations(ctx context.Context, kind string, limit int) ([]pdb.CoinRecommendation, error) {
	// 1. 获取市场数据（最新的涨幅榜）
	now := time.Now().UTC()
	startTime := now.Add(-2 * time.Hour) // 最近2小时的数据
	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), kind, startTime, now)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	if len(snaps) == 0 {
		return []pdb.CoinRecommendation{}, nil
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
	// 收集所有候选币种的基础符号
	baseSymbolsForFlow := make([]string, 0, len(candidates))
	baseSymbolSetForFlow := make(map[string]bool)
	for _, item := range candidates {
		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol != "" && !baseSymbolSetForFlow[baseSymbol] {
			baseSymbolsForFlow = append(baseSymbolsForFlow, baseSymbol)
			baseSymbolSetForFlow[baseSymbol] = true
		}
	}
	
	// 批量获取资金流趋势数据
	flowTrendDataMap, err := s.GetFlowTrendForSymbols(ctx, baseSymbolsForFlow)
	if err != nil {
		log.Printf("[WARN] Failed to get flow trend data: %v", err)
		flowTrendDataMap = make(map[string]*FlowTrendResult)
	}

	// 4. 获取公告数据（最近24小时）
	announcementData, err := s.getAnnouncementDataForRecommendation(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get announcement data: %v", err)
		announcementData = make(map[string]bool)
	}

	// 4.5. 获取公告重要性得分
	// 收集所有候选币种的基础符号（用于公告评分）
	baseSymbolsForAnnouncement := make([]string, 0, len(candidates))
	baseSymbolSetForAnnouncement := make(map[string]bool)
	for _, item := range candidates {
		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol != "" && !baseSymbolSetForAnnouncement[baseSymbol] {
			baseSymbolsForAnnouncement = append(baseSymbolsForAnnouncement, baseSymbol)
			baseSymbolSetForAnnouncement[baseSymbol] = true
		}
	}
	
	// 批量获取公告重要性得分（查询最近7天的公告）
	announcementScores, err := s.GetAnnouncementScoresForSymbols(ctx, baseSymbolsForAnnouncement, 7)
	if err != nil {
		log.Printf("[WARN] Failed to get announcement scores: %v", err)
		announcementScores = make(map[string]*AnnouncementScore)
	}

	// 4.5. 获取Twitter情绪数据
	// 收集所有候选币种的基础符号
	baseSymbols := make([]string, 0, len(candidates))
	baseSymbolSet := make(map[string]bool)
	for _, item := range candidates {
		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol != "" && !baseSymbolSet[baseSymbol] {
			baseSymbols = append(baseSymbols, baseSymbol)
			baseSymbolSet[baseSymbol] = true
		}
	}
	
	// 批量获取情绪数据
	twitterSentimentData, err := s.GetTwitterSentimentForSymbols(ctx, baseSymbols)
	if err != nil {
		log.Printf("[WARN] Failed to get Twitter sentiment data: %v", err)
		twitterSentimentData = make(map[string]*SentimentResult)
	}

	// 5. 计算每个币种的得分
	scores := make([]RecommendationScore, 0, len(candidates))
	for _, item := range candidates {
		// 过滤黑名单
		if blacklistMap[item.Symbol] {
			continue
		}

		// 提取基础币种
		baseSymbol := extractBaseSymbol(item.Symbol)

		// 获取该币种的Twitter情绪数据
		var sentimentData *SentimentResult
		if data, ok := twitterSentimentData[baseSymbol]; ok {
			sentimentData = data
		}

		// 获取该币种的资金流趋势数据
		var flowTrendData *FlowTrendResult
		if data, ok := flowTrendDataMap[baseSymbol]; ok {
			flowTrendData = data
		}

		// 获取该币种的公告重要性得分
		var announcementScore *AnnouncementScore
		if score, ok := announcementScores[baseSymbol]; ok {
			announcementScore = score
		}

		// 计算得分
		score := s.calculateScoreWithKind(ctx, item, baseSymbol, flowData, announcementData, sentimentData, flowTrendData, announcementScore, kind)
		
		// 获取价格预测
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
	weights := s.calculateDynamicWeights(marketState)
	
	// 使用动态权重重新计算总分
	for i := range scores {
		scores[i].TotalScore = scores[i].Scores.Market*weights.MarketWeight +
			scores[i].Scores.Flow*weights.FlowWeight +
			scores[i].Scores.Heat*weights.HeatWeight +
			scores[i].Scores.Event*weights.EventWeight +
			scores[i].Scores.Sentiment*weights.SentimentWeight
	}

	// 7. 按总分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	// 8. 取前N个
	if len(scores) > limit {
		scores = scores[:limit]
	}

	// 9. 转换为数据库模型
	recommendations := make([]pdb.CoinRecommendation, 0, len(scores))
	for i, score := range scores {
		reasonsJSON, _ := json.Marshal(score.Reasons)
		riskWarningsJSON, _ := json.Marshal(score.Risk.RiskWarnings)
		
		// 序列化技术指标
		var technicalJSON []byte
		if score.Technical != nil {
			technicalJSON, _ = json.Marshal(score.Technical)
		}
		
		rec := pdb.CoinRecommendation{
			Kind:            kind,
			Symbol:          score.Symbol,
			BaseSymbol:      score.BaseSymbol,
			Rank:            i + 1,
			TotalScore:      score.TotalScore,
			MarketScore:     score.Scores.Market,
			FlowScore:       score.Scores.Flow,
			HeatScore:       score.Scores.Heat,
			EventScore:      score.Scores.Event,
			SentimentScore:  score.Scores.Sentiment,
			PriceChange24h:  &score.Data.PriceChange24h,
			Volume24h:       &score.Data.Volume24h,
			MarketCapUSD:    score.Data.MarketCapUSD,
			NetFlow24h:      &score.Data.NetFlow24h,
			HasNewListing:   score.Data.HasNewListing,
			HasAnnouncement: score.Data.HasAnnouncement,
			Reasons:         reasonsJSON,
			// 风险评级字段
			VolatilityRisk:  &score.Risk.VolatilityRisk,
			LiquidityRisk:   &score.Risk.LiquidityRisk,
			MarketRisk:      &score.Risk.MarketRisk,
			TechnicalRisk:   &score.Risk.TechnicalRisk,
			OverallRisk:     &score.Risk.OverallRisk,
			RiskLevel:       &score.Risk.RiskLevel,
			RiskWarnings:    riskWarningsJSON,
			// 技术指标
			TechnicalIndicators: technicalJSON,
			GeneratedAt:     now,
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// calculateScore 计算币种得分
func (s *Server) calculateScore(
	item pdb.BinanceMarketTop,
	baseSymbol string,
	flowData map[string]float64,
	announcementData map[string]bool,
	sentimentData *SentimentResult,
	flowTrendData *FlowTrendResult,
	announcementScore *AnnouncementScore,
) RecommendationScore {
	score := RecommendationScore{
		Symbol:     item.Symbol,
		BaseSymbol: baseSymbol,
	}

	// 解析价格和涨幅
	price, _ := strconv.ParseFloat(item.LastPrice, 64)
	volume, _ := strconv.ParseFloat(item.Volume, 64)
	priceChange24h := item.PctChange

	score.Data.Price = price
	score.Data.PriceChange24h = priceChange24h
	score.Data.Volume24h = volume
	score.Data.MarketCapUSD = item.MarketCapUSD

	// 因子1：市场表现（30%）
	// 涨幅得分（15%）
	var priceChangeScore float64
	if priceChange24h > 20 {
		priceChangeScore = 15
	} else if priceChange24h > 10 {
		priceChangeScore = 10 + (priceChange24h-10)/10*5 // 10-15分
	} else if priceChange24h > 5 {
		priceChangeScore = 5 + (priceChange24h-5)/5*5 // 5-10分
	} else if priceChange24h > 0 {
		priceChangeScore = priceChange24h / 5 * 5 // 0-5分
	} else {
		priceChangeScore = 0
	}

	// 成交量得分（10%）
	volumeScore := math.Min(10, math.Log10(volume+1)/10*10) // 对数归一化

	// 排名得分（5%）
	var rankScore float64
	if item.Rank <= 5 {
		rankScore = 5
	} else if item.Rank <= 10 {
		rankScore = 3
	} else if item.Rank <= 20 {
		rankScore = 1
	}

	score.Scores.Market = priceChangeScore + volumeScore + rankScore

	// 因子2：资金流（25%）
	netFlow := flowData[baseSymbol] // 单位：USD（24h净流入）
	score.Data.NetFlow24h = netFlow

		// 保存趋势数据到score中，用于生成推荐理由
		score.Data.FlowTrend = flowTrendData

		// 使用趋势数据计算资金流得分
		var flowScore float64
		if flowTrendData != nil && flowTrendData.Flow24h != 0 {
			// 使用趋势分析计算得分
			// 公式：24h净流入 * 0.6 + 3天趋势 * 0.3 + 7天趋势 * 0.1
			flowScore = CalculateFlowScoreWithTrend(
				flowTrendData.Flow24h,
				flowTrendData.Trend3d,
				flowTrendData.Trend7d,
			)
			
			// 如果有大额资金流入，额外加分
			if flowTrendData.LargeFlow {
				flowScore *= 1.2 // 增加20%
			}
			
			// 如果出现反转信号，额外加分（反转可能是买入信号）
			if flowTrendData.Reversal && flowTrendData.Flow24h > 0 {
				flowScore *= 1.15 // 增加15%
			}
		} else {
		// 没有趋势数据，使用原来的简单计算
		if netFlow > 10000000 { // 1000万
			flowScore = 15
		} else if netFlow > 5000000 { // 500万
			flowScore = 10 + (netFlow-5000000)/5000000*5
		} else if netFlow > 1000000 { // 100万
			flowScore = 5 + (netFlow-1000000)/4000000*5
		} else if netFlow > 0 {
			flowScore = netFlow / 1000000 * 5
		} else {
			flowScore = 0
		}
		
		// 流入趋势得分（10%），简化处理，有净流入就给10分
		trendScore := 0.0
		if netFlow > 0 {
			trendScore = 10
		}
		flowScore += trendScore
	}

	score.Scores.Flow = math.Min(25, flowScore) // 资金流因子最高25分

	// 因子3：市场热度（20%）
	// 市值得分（10%）
	var marketCapScore float64
	if item.MarketCapUSD != nil {
		mcap := *item.MarketCapUSD
		if mcap > 10000000000 { // 100亿
			marketCapScore = 10
		} else if mcap > 1000000000 { // 10亿
			marketCapScore = 7
		} else if mcap > 100000000 { // 1亿
			marketCapScore = 5
		} else {
			marketCapScore = 3
		}
	} else {
		marketCapScore = 3 // 默认值
	}

	// 流动性得分（10%），基于成交量
	liquidityScore := math.Min(10, volumeScore) // 复用成交量得分

	score.Scores.Heat = marketCapScore + liquidityScore

	// 因子4：公告与事件（15%）
	hasAnnouncement := announcementData[baseSymbol]
	score.Data.HasAnnouncement = hasAnnouncement
	score.Data.AnnouncementScore = announcementScore

	var eventScore float64
	if announcementScore != nil {
		// 使用重要性得分（最高30分，但事件因子最高15分）
		// 将30分制转换为15分制
		eventScore = (announcementScore.TotalScore / 30.0) * 15.0
		eventScore = math.Min(15, eventScore) // 限制最高15分
	} else if hasAnnouncement {
		// 有公告但没有评分数据，使用默认值
		eventScore = 10 // 降低默认值，因为无法确定重要性
	} else {
		eventScore = 0
	}

	score.Scores.Event = eventScore

	// 因子5：Twitter情绪（10%）
	if sentimentData != nil && sentimentData.Total > 0 {
		// 使用实际的情绪得分（0-10分）
		score.Scores.Sentiment = sentimentData.Score
		score.Data.TwitterMentions = sentimentData.Mentions
		
		// 如果推文数量太少，降低权重
		if sentimentData.Total < 10 {
			score.Scores.Sentiment = score.Scores.Sentiment * 0.7 // 降低30%
		}
	} else {
		// 没有情绪数据，使用默认值
		score.Scores.Sentiment = 5.0
		score.Data.TwitterMentions = 0
	}

	// 技术指标将在 calculateScoreWithKind 中获取
	score.Technical = nil

	// 先使用基础权重计算总分（用于排序前的初步筛选）
	// 注意：最终总分会在generateRecommendations中使用动态权重重新计算
	score.TotalScore = score.Scores.Market + score.Scores.Flow +
		score.Scores.Heat + score.Scores.Event + score.Scores.Sentiment

	// 计算风险评级（在calculateScoreWithKind中计算）
	// 这里先初始化默认值
	score.Risk = struct {
		VolatilityRisk  float64
		LiquidityRisk   float64
		MarketRisk      float64
		TechnicalRisk   float64
		OverallRisk     float64
		RiskLevel       string
		RiskWarnings    []string
	}{}

	// 生成推荐理由
	score.Reasons = s.generateReasons(score)

	return score
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
) RecommendationScore {
	// 先计算基础得分
	score := s.calculateScore(item, baseSymbol, flowData, announcementData, sentimentData, flowTrendData, announcementScore)

	// 获取技术指标
	// 优先使用Binance API，失败则使用历史快照数据
	technical, err := s.GetTechnicalIndicators(ctx, item.Symbol, kind)
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
		if technical.DistanceToSupport < 2 && technical.DistanceToSupport > 0 {
			adjustment *= 1.05 // 接近支撑位，可能反弹
		}
		if technical.DistanceToResistance < 2 && technical.DistanceToResistance > 0 {
			adjustment *= 0.95 // 接近阻力位，可能回调
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

	// 计算风险评级
	score.Risk = s.calculateRiskMetrics(score, item)

	// 生成推荐理由
	score.Reasons = s.generateReasons(score)

	return score
}

// generateReasons 生成推荐理由
func (s *Server) generateReasons(score RecommendationScore) []string {
	reasons := make([]string, 0)

	if score.Data.PriceChange24h > 5 {
		reasons = append(reasons, fmt.Sprintf("24h涨幅%.2f%%，表现强劲", score.Data.PriceChange24h))
	} else if score.Data.PriceChange24h > 0 {
		reasons = append(reasons, fmt.Sprintf("24h涨幅%.2f%%，稳步上涨", score.Data.PriceChange24h))
	}

	// 资金流理由（使用趋势数据）
	if score.Data.FlowTrend != nil {
		trend := score.Data.FlowTrend
		
		// 24h净流入
		if trend.Flow24h > 10000000 {
			reasons = append(reasons, fmt.Sprintf("交易所净流入%.0f万USD，资金面良好", trend.Flow24h/10000))
		} else if trend.Flow24h > 0 {
			reasons = append(reasons, fmt.Sprintf("交易所净流入%.0f万USD", trend.Flow24h/10000))
		}
		
		// 趋势理由
		if trend.Trend3d > 30 {
			reasons = append(reasons, fmt.Sprintf("3天资金流增长%.1f%%，资金加速流入", trend.Trend3d))
		} else if trend.Trend3d > 10 {
			reasons = append(reasons, fmt.Sprintf("3天资金流增长%.1f%%，资金持续流入", trend.Trend3d))
		} else if trend.Trend3d < -30 {
			reasons = append(reasons, fmt.Sprintf("3天资金流下降%.1f%%，资金加速流出", math.Abs(trend.Trend3d)))
		}
		
		// 反转信号
		if trend.Reversal && trend.Flow24h > 0 {
			reasons = append(reasons, "资金流出现反转信号，从流出转为流入")
		}
		
		// 大额资金流入
		if trend.LargeFlow {
			reasons = append(reasons, "出现大额资金流入，市场关注度提升")
		}
		
		// 加速度
		if trend.Acceleration > 20 {
			reasons = append(reasons, fmt.Sprintf("资金流加速度%.1f%%，流入速度加快", trend.Acceleration))
		}
	} else {
		// 没有趋势数据，使用简单逻辑
		if score.Data.NetFlow24h > 10000000 {
			reasons = append(reasons, fmt.Sprintf("交易所净流入%.0f万USD，资金面良好", score.Data.NetFlow24h/10000))
		} else if score.Data.NetFlow24h > 0 {
			reasons = append(reasons, fmt.Sprintf("交易所净流入%.0f万USD", score.Data.NetFlow24h/10000))
		}
	}

	if score.Data.MarketCapUSD != nil && *score.Data.MarketCapUSD > 10000000000 {
		reasons = append(reasons, "市值排名靠前，流动性充足")
	}

	// 公告理由（使用重要性得分）
	if score.Data.AnnouncementScore != nil {
		annScore := score.Data.AnnouncementScore
		if annScore.Importance == "high" {
			exchangeName := annScore.Exchange
			if exchangeName == "" {
				exchangeName = "交易所"
			}
			reasons = append(reasons, fmt.Sprintf("重要公告：%s（得分%.1f，%s）", 
				annScore.Details, annScore.TotalScore, exchangeName))
		} else if annScore.Importance == "medium" {
			reasons = append(reasons, fmt.Sprintf("公告：%s（得分%.1f）", 
				annScore.Details, annScore.TotalScore))
		} else {
			reasons = append(reasons, fmt.Sprintf("有相关公告（得分%.1f）", annScore.TotalScore))
		}
		
		// 添加详细信息
		if annScore.VerifiedBonus > 1.0 {
			reasons = append(reasons, "官方验证公告，可信度高")
		}
		if annScore.HeatScore > 3.0 {
			reasons = append(reasons, "公告热度较高，市场关注度提升")
		}
		if annScore.CategoryScore >= 9.0 {
			reasons = append(reasons, "新币上线或重大事件，市场影响较大")
		}
	} else if score.Data.HasAnnouncement {
		reasons = append(reasons, "最近有重要公告或事件")
	}

	// 添加Twitter情绪理由
	if score.Data.TwitterMentions > 0 {
		if score.Scores.Sentiment > 7 {
			reasons = append(reasons, fmt.Sprintf("Twitter情绪积极（提及%d次，得分%.1f）", score.Data.TwitterMentions, score.Scores.Sentiment))
		} else if score.Scores.Sentiment < 3 {
			reasons = append(reasons, fmt.Sprintf("Twitter情绪偏负面（提及%d次，得分%.1f），需谨慎", score.Data.TwitterMentions, score.Scores.Sentiment))
		} else if score.Data.TwitterMentions > 50 {
			reasons = append(reasons, fmt.Sprintf("Twitter讨论热度较高（提及%d次）", score.Data.TwitterMentions))
		}
	}

	// 添加技术指标理由
	if score.Technical != nil {
		tech := score.Technical
		reasons = append(reasons, s.generateTechnicalReasons(tech)...)
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "综合评分较高，值得关注")
	}

	return reasons
}

// generateTechnicalReasons 生成技术指标相关的推荐理由
func (s *Server) generateTechnicalReasons(tech *TechnicalIndicators) []string {
	reasons := make([]string, 0)

	// RSI理由
	if tech.RSI > 70 {
		reasons = append(reasons, fmt.Sprintf("RSI %.1f，处于超买区域，需注意回调风险", tech.RSI))
	} else if tech.RSI < 30 {
		reasons = append(reasons, fmt.Sprintf("RSI %.1f，处于超卖区域，可能反弹", tech.RSI))
	}

	// MACD理由
	if tech.MACD > tech.MACDSignal && tech.MACDHist > 0 {
		reasons = append(reasons, "MACD金叉，技术面看涨")
	} else if tech.MACD < tech.MACDSignal && tech.MACDHist < 0 {
		reasons = append(reasons, "MACD死叉，技术面偏弱")
	}

	// 布林带理由
	if tech.BBPosition > 0 {
		if tech.BBPosition < 0.2 {
			reasons = append(reasons, "价格接近布林带下轨，可能反弹")
		} else if tech.BBPosition > 0.8 {
			reasons = append(reasons, "价格接近布林带上轨，需注意回调")
		}
		if tech.BBWidth > 5 {
			reasons = append(reasons, fmt.Sprintf("布林带宽度%.1f%%，波动率较高", tech.BBWidth))
		}
	}

	// KDJ理由
	if tech.K > 0 && tech.D > 0 {
		if tech.K > tech.D && tech.K < 80 {
			reasons = append(reasons, fmt.Sprintf("KDJ金叉（K=%.1f, D=%.1f），短期看涨", tech.K, tech.D))
		} else if tech.K < tech.D && tech.K > 20 {
			reasons = append(reasons, fmt.Sprintf("KDJ死叉（K=%.1f, D=%.1f），短期偏弱", tech.K, tech.D))
		} else if tech.K > 80 {
			reasons = append(reasons, "KDJ超买，需注意回调")
		} else if tech.K < 20 {
			reasons = append(reasons, "KDJ超卖，可能反弹")
		}
	}

	// 均线系统理由
	if tech.MA5 > 0 && tech.MA10 > 0 && tech.MA20 > 0 {
		if tech.MA5 > tech.MA10 && tech.MA10 > tech.MA20 {
			reasons = append(reasons, "均线多头排列，趋势向上")
		} else if tech.MA5 < tech.MA10 && tech.MA10 < tech.MA20 {
			reasons = append(reasons, "均线空头排列，趋势向下")
		}
	}

	// 成交量理由
	if tech.VolumeRatio > 1.5 {
		reasons = append(reasons, fmt.Sprintf("成交量放大%.1f倍，市场活跃度提升", tech.VolumeRatio))
	} else if tech.VolumeRatio < 0.5 {
		reasons = append(reasons, "成交量萎缩，市场活跃度较低")
	}

	// 支撑位/阻力位理由
	if tech.DistanceToSupport > 0 && tech.DistanceToSupport < 2 {
		reasons = append(reasons, fmt.Sprintf("价格接近支撑位（距离%.1f%%），可能反弹", tech.DistanceToSupport))
	}
	if tech.DistanceToResistance > 0 && tech.DistanceToResistance < 2 {
		reasons = append(reasons, fmt.Sprintf("价格接近阻力位（距离%.1f%%），需注意突破", tech.DistanceToResistance))
	}

	// 趋势理由
	if tech.Trend == "up" {
		reasons = append(reasons, "技术指标显示上涨趋势")
	} else if tech.Trend == "down" {
		reasons = append(reasons, "技术指标显示下跌趋势")
	}

	return reasons
}

// getFlowDataForRecommendation 获取资金流数据用于推荐
// 返回：币种 -> 24h净流入（USD）
func (s *Server) getFlowDataForRecommendation(ctx context.Context) (map[string]float64, error) {
	// 获取最近3天的资金流数据（用于计算趋势）
	today := time.Now().UTC()
	day3Ago := today.AddDate(0, 0, -3)
	
	var flows []pdb.DailyFlow
	err := s.db.DB().Where("day >= ? AND day <= ?", day3Ago.Format("2006-01-02"), today.Format("2006-01-02")).
		Find(&flows).Error
	if err != nil {
		return nil, err
	}

	if len(flows) == 0 {
		return make(map[string]float64), nil
	}

	// 收集所有需要查询价格的币种
	coinSet := make(map[string]bool)
	for _, flow := range flows {
		coinSet[flow.Coin] = true
	}
	
	// 获取币种价格（使用CoinCap或CoinGecko）
	coins := make([]string, 0, len(coinSet))
	for coin := range coinSet {
		coins = append(coins, coin)
	}
	
	// 使用CoinCap获取价格（支持更多币种）
	priceMap := make(map[string]float64)
	coinCapCache := newCoinCapCache()
	for _, coin := range coins {
		meta, err := coinCapCache.Get(ctx, coin)
		if err == nil && meta.MarketCapUSD != nil {
			// 从市值和流通量计算价格
			if meta.Circulating != nil && *meta.Circulating > 0 {
				price := *meta.MarketCapUSD / *meta.Circulating
				priceMap[coin] = price
			}
		}
	}
	
	// 如果CoinCap没有，尝试从Binance市场数据获取价格
	if len(priceMap) < len(coins) {
		var marketTops []pdb.BinanceMarketTop
		s.db.DB().Where("symbol LIKE ?", "%USDT").Find(&marketTops)
		for _, top := range marketTops {
			baseSymbol := extractBaseSymbol(top.Symbol)
			if _, exists := priceMap[baseSymbol]; !exists {
				if price, err := strconv.ParseFloat(top.LastPrice, 64); err == nil {
					priceMap[baseSymbol] = price
				}
			}
		}
	}

	// 按币种聚合净流入（转换为USD）
	// 只使用今天的数据作为24h净流入
	todayStr := today.Format("2006-01-02")
	result := make(map[string]float64)
	
	for _, flow := range flows {
		if flow.Day != todayStr {
			continue // 只计算今天的
		}
		
		net, err := strconv.ParseFloat(flow.Net, 64)
		if err != nil {
			continue
		}
		
		// 如果Net字段已经是USD，直接使用
		// 否则根据币种价格转换
		if price, ok := priceMap[flow.Coin]; ok && price > 0 {
			// Net是币种数量，需要转换为USD
			netUSD := net * price
			result[flow.Coin] += netUSD
		} else {
			// 如果无法获取价格，假设Net已经是USD（向后兼容）
			result[flow.Coin] += net
		}
	}

	return result, nil
}

// getAnnouncementDataForRecommendation 获取公告数据用于推荐
// 返回：币种 -> 是否有公告
func (s *Server) getAnnouncementDataForRecommendation(ctx context.Context) (map[string]bool, error) {
	// 查询最近7天的公告（扩大时间范围）
	since := time.Now().UTC().AddDate(0, 0, -7)

	var announcements []pdb.Announcement
	err := s.db.DB().Where("release_time >= ?", since).
		Order("release_time DESC").
		Find(&announcements).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	
	// 常见币种列表（扩展版）
	commonCoins := []string{
		"BTC", "ETH", "SOL", "BNB", "XRP", "ADA", "DOGE", "TON", "AVAX", "DOT",
		"MATIC", "LINK", "UNI", "ATOM", "ETC", "LTC", "BCH", "XLM", "ALGO", "FIL",
		"APT", "ARB", "OP", "SUI", "SEI", "TIA", "INJ", "NEAR", "FTM", "AAVE",
		"COMP", "MKR", "SNX", "CRV", "SUSHI", "1INCH", "CAKE", "PEPE", "SHIB", "FLOKI",
	}
	
	// 从Binance市场数据获取所有交易对的基础币种
	var marketTops []pdb.BinanceMarketTop
	s.db.DB().Select("DISTINCT symbol").Find(&marketTops)
	allCoins := make(map[string]bool)
	for _, top := range marketTops {
		baseSymbol := extractBaseSymbol(top.Symbol)
		if baseSymbol != "" {
			allCoins[baseSymbol] = true
		}
	}
	
	// 合并常见币种和所有交易对币种
	allSymbols := make(map[string]bool)
	for _, coin := range commonCoins {
		allSymbols[coin] = true
	}
	for coin := range allCoins {
		allSymbols[coin] = true
	}

	for _, ann := range announcements {
		// 方法1：从Tags中提取（最准确）
		tags := ann.Tags.Data()
		if len(tags) > 0 {
			for _, tag := range tags {
				tagUpper := strings.ToUpper(strings.TrimSpace(tag))
				if allSymbols[tagUpper] {
					result[tagUpper] = true
				}
			}
		}
		
		// 方法2：从标题中提取
		titleSymbols := extractSymbolsFromTextAdvanced(ann.Title, allSymbols)
		for _, sym := range titleSymbols {
			result[sym] = true
		}
		
		// 方法3：从摘要中提取
		if ann.Summary != "" {
			summarySymbols := extractSymbolsFromTextAdvanced(ann.Summary, allSymbols)
			for _, sym := range summarySymbols {
				result[sym] = true
			}
		}
		
		// 方法4：从交易所字段提取（如果是新币上线）
		if ann.Exchange != "" && (ann.Category == "newcoin" || ann.IsEvent) {
			// 尝试从标题中提取新币符号
			// 新币上线通常格式为 "XXX Listing on Binance"
			titleUpper := strings.ToUpper(ann.Title)
			if strings.Contains(titleUpper, "LISTING") || strings.Contains(titleUpper, "上线") {
				// 提取可能的币种符号（2-10个字母，全大写）
				words := strings.Fields(titleUpper)
				for _, word := range words {
					word = strings.Trim(word, ".,!?()[]{}")
					if len(word) >= 2 && len(word) <= 10 && isAllUpper(word) {
						if allSymbols[word] {
							result[word] = true
						}
					}
				}
			}
		}
	}

	return result, nil
}

// extractSymbolsFromText 从文本中提取币种符号（简化版，保持向后兼容）
func extractSymbolsFromText(text string) []string {
	commonCoins := []string{"BTC", "ETH", "SOL", "BNB", "XRP", "ADA", "DOGE", "TON", "AVAX", "DOT"}
	coinSet := make(map[string]bool)
	for _, coin := range commonCoins {
		coinSet[coin] = true
	}
	return extractSymbolsFromTextAdvanced(text, coinSet)
}

// extractSymbolsFromTextAdvanced 从文本中提取币种符号（高级版）
// 支持从候选币种列表中匹配
func extractSymbolsFromTextAdvanced(text string, candidateSymbols map[string]bool) []string {
	result := make([]string, 0)
	upperText := strings.ToUpper(text)
	found := make(map[string]bool) // 去重

	// 按长度从长到短排序候选币种（避免短币种匹配到长币种的一部分）
	sortedSymbols := make([]string, 0, len(candidateSymbols))
	for sym := range candidateSymbols {
		sortedSymbols = append(sortedSymbols, sym)
	}
	sort.Slice(sortedSymbols, func(i, j int) bool {
		return len(sortedSymbols[i]) > len(sortedSymbols[j])
	})

	// 匹配币种符号
	for _, coin := range sortedSymbols {
		if found[coin] {
			continue
		}
		// 使用单词边界匹配（避免部分匹配）
		// 例如：避免 "BTC" 匹配到 "BTCE"
		pattern := "\\b" + regexp.QuoteMeta(coin) + "\\b"
		matched, _ := regexp.MatchString(pattern, upperText)
		if matched {
			result = append(result, coin)
			found[coin] = true
		}
	}

	return result
}

// isAllUpper 检查字符串是否全为大写字母
func isAllUpper(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return len(s) > 0
}

// extractBaseSymbol 从交易对提取基础币种
func extractBaseSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// 处理期货后缀
	if strings.HasSuffix(symbol, "USD_PERP") {
		return strings.TrimSuffix(symbol, "USD_PERP")
	}

	// 处理现货后缀
	suffixes := []string{"USDT", "USDC", "FDUSD", "BUSD", "TUSD", "USDP"}
	for _, suf := range suffixes {
		if strings.HasSuffix(symbol, suf) {
			return strings.TrimSuffix(symbol, suf)
		}
	}

	return symbol
}

// formatRecommendations 格式化推荐结果
func formatRecommendations(recs []pdb.CoinRecommendation) []gin.H {
	// 解析风险警告JSON
	result := make([]gin.H, 0, len(recs))
	for _, rec := range recs {
		var reasons []string
		if len(rec.Reasons) > 0 {
			json.Unmarshal(rec.Reasons, &reasons)
		}

		var riskWarnings []string
		if len(rec.RiskWarnings) > 0 {
			json.Unmarshal(rec.RiskWarnings, &riskWarnings)
		}

		var technical *TechnicalIndicators
		if len(rec.TechnicalIndicators) > 0 {
			var tech TechnicalIndicators
			if err := json.Unmarshal(rec.TechnicalIndicators, &tech); err == nil {
				technical = &tech
			}
		}

		item := gin.H{
			"rank":        rec.Rank,
			"symbol":      rec.Symbol,
			"base_symbol": rec.BaseSymbol,
			"total_score": rec.TotalScore,
			"scores": gin.H{
				"market":    rec.MarketScore,
				"flow":      rec.FlowScore,
				"heat":      rec.HeatScore,
				"event":     rec.EventScore,
				"sentiment": rec.SentimentScore,
			},
			"data": gin.H{
				"price":            getFloatValue(rec.PriceChange24h),
				"price_change_24h": getFloatValue(rec.PriceChange24h),
				"volume_24h":       getFloatValue(rec.Volume24h),
				"market_cap_usd":   rec.MarketCapUSD,
				"net_flow_24h":     getFloatValue(rec.NetFlow24h),
				"has_new_listing":  rec.HasNewListing,
				"has_announcement": rec.HasAnnouncement,
			},
			"risk": gin.H{
				"volatility_risk": getFloatValue(rec.VolatilityRisk),
				"liquidity_risk":  getFloatValue(rec.LiquidityRisk),
				"market_risk":     getFloatValue(rec.MarketRisk),
				"technical_risk":   getFloatValue(rec.TechnicalRisk),
				"overall_risk":    getFloatValue(rec.OverallRisk),
				"risk_level":       getStringValue(rec.RiskLevel),
				"risk_warnings":    riskWarnings,
			},
			"technical": technical,
			"reasons": reasons,
		}
		result = append(result, item)
	}
	return result
}

// getFloatValue 安全获取浮点数值
func getFloatValue(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// getStringValue 安全获取字符串值
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// analyzeMarketState 分析市场状态
func (s *Server) analyzeMarketState(candidates []pdb.BinanceMarketTop) MarketState {
	state := MarketState{
		State: "sideways", // 默认震荡市
	}

	if len(candidates) == 0 {
		return state
	}

	// 计算平均涨幅
	var totalChange float64
	var upCount int
	var changes []float64

	for _, item := range candidates {
		change := item.PctChange
		totalChange += change
		changes = append(changes, change)
		if change > 0 {
			upCount++
		}
	}

	state.AvgChange = totalChange / float64(len(candidates))
	state.UpRatio = float64(upCount) / float64(len(candidates))

	// 计算波动率（标准差）
	if len(changes) > 1 {
		var variance float64
		for _, change := range changes {
			variance += (change - state.AvgChange) * (change - state.AvgChange)
		}
		state.Volatility = math.Sqrt(variance / float64(len(changes)))
	}

	// 判断市场状态
	// 牛市：平均涨幅>5%，上涨比例>60%，波动率适中
	// 熊市：平均涨幅<-3%，上涨比例<40%，波动率较高
	// 震荡市：其他情况
	if state.AvgChange > 5 && state.UpRatio > 0.6 && state.Volatility < 15 {
		state.State = "bull"
	} else if state.AvgChange < -3 && state.UpRatio < 0.4 {
		state.State = "bear"
	} else {
		state.State = "sideways"
	}

	return state
}

// calculateDynamicWeights 根据市场状态计算动态权重
func (s *Server) calculateDynamicWeights(marketState MarketState) DynamicWeights {
	weights := DynamicWeights{}

	switch marketState.State {
	case "bull":
		// 牛市：提高市场表现权重，降低资金流和事件权重
		// 牛市时市场表现更重要，资金流和事件相对次要
		weights.MarketWeight = 0.35  // 提高市场表现权重
		weights.FlowWeight = 0.20    // 降低资金流权重
		weights.HeatWeight = 0.20    // 保持热度权重
		weights.EventWeight = 0.10   // 降低事件权重
		weights.SentimentWeight = 0.15 // 提高情绪权重（牛市情绪更重要）

	case "bear":
		// 熊市：提高资金流和事件权重，降低市场表现权重
		// 熊市时资金流和事件驱动更重要，市场表现相对次要
		weights.MarketWeight = 0.20  // 降低市场表现权重
		weights.FlowWeight = 0.30    // 提高资金流权重（资金流入更重要）
		weights.HeatWeight = 0.15    // 降低热度权重
		weights.EventWeight = 0.20   // 提高事件权重（事件驱动更重要）
		weights.SentimentWeight = 0.15 // 保持情绪权重

	case "sideways":
		// 震荡市：平衡各因子权重
		weights.MarketWeight = 0.30
		weights.FlowWeight = 0.25
		weights.HeatWeight = 0.20
		weights.EventWeight = 0.15
		weights.SentimentWeight = 0.10

	default:
		// 默认权重（震荡市）
		weights.MarketWeight = 0.30
		weights.FlowWeight = 0.25
		weights.HeatWeight = 0.20
		weights.EventWeight = 0.15
		weights.SentimentWeight = 0.10
	}

	// 根据市场波动率微调权重
	// 高波动时，降低市场表现权重，提高资金流权重
	if marketState.Volatility > 20 {
		weights.MarketWeight *= 0.9
		weights.FlowWeight *= 1.1
		weights.HeatWeight *= 0.95
	}

	// 根据上涨比例微调权重
	// 上涨比例高时，提高市场表现权重
	if marketState.UpRatio > 0.7 {
		weights.MarketWeight *= 1.05
		weights.FlowWeight *= 0.95
	} else if marketState.UpRatio < 0.3 {
		// 上涨比例低时，提高资金流和事件权重
		weights.MarketWeight *= 0.95
		weights.FlowWeight *= 1.05
		weights.EventWeight *= 1.05
	}

	// 归一化权重（确保总和为1）
	total := weights.MarketWeight + weights.FlowWeight + weights.HeatWeight +
		weights.EventWeight + weights.SentimentWeight
	if total > 0 {
		weights.MarketWeight /= total
		weights.FlowWeight /= total
		weights.HeatWeight /= total
		weights.EventWeight /= total
		weights.SentimentWeight /= total
	}

	return weights
}

// calculateRiskMetrics 计算风险指标
func (s *Server) calculateRiskMetrics(score RecommendationScore, item pdb.BinanceMarketTop) struct {
	VolatilityRisk  float64
	LiquidityRisk   float64
	MarketRisk      float64
	TechnicalRisk   float64
	OverallRisk     float64
	RiskLevel       string
	RiskWarnings    []string
} {
	risk := struct {
		VolatilityRisk  float64
		LiquidityRisk   float64
		MarketRisk      float64
		TechnicalRisk   float64
		OverallRisk     float64
		RiskLevel       string
		RiskWarnings    []string
	}{}

	// 1. 波动率风险（基于24h涨幅的绝对值）
	absChange := math.Abs(score.Data.PriceChange24h)
	if absChange > 30 {
		risk.VolatilityRisk = 100 // 极高波动
	} else if absChange > 20 {
		risk.VolatilityRisk = 80
	} else if absChange > 10 {
		risk.VolatilityRisk = 60
	} else if absChange > 5 {
		risk.VolatilityRisk = 40
	} else {
		risk.VolatilityRisk = 20
	}

	// 2. 流动性风险（基于成交量和市值）
	volumeRisk := 0.0
	marketCapRisk := 0.0

	// 成交量风险
	if score.Data.Volume24h < 1000000 { // 小于100万USD
		volumeRisk = 80
	} else if score.Data.Volume24h < 5000000 { // 小于500万USD
		volumeRisk = 60
	} else if score.Data.Volume24h < 20000000 { // 小于2000万USD
		volumeRisk = 40
	} else {
		volumeRisk = 20
	}

	// 市值风险
	if score.Data.MarketCapUSD != nil {
		if *score.Data.MarketCapUSD < 10000000 { // 小于1000万USD
			marketCapRisk = 80
		} else if *score.Data.MarketCapUSD < 50000000 { // 小于5000万USD
			marketCapRisk = 60
		} else if *score.Data.MarketCapUSD < 200000000 { // 小于2亿USD
			marketCapRisk = 40
		} else {
			marketCapRisk = 20
		}
	} else {
		marketCapRisk = 70 // 无市值数据，风险较高
	}

	risk.LiquidityRisk = (volumeRisk + marketCapRisk) / 2

	// 3. 市场风险（基于排名和市值）
	marketRisk := 0.0
	if item.Rank > 100 {
		marketRisk = 70
	} else if item.Rank > 50 {
		marketRisk = 50
	} else if item.Rank > 20 {
		marketRisk = 30
	} else {
		marketRisk = 15
	}

	// 如果市值很小，增加市场风险
	if score.Data.MarketCapUSD != nil && *score.Data.MarketCapUSD < 100000000 {
		marketRisk += 20
	}

	risk.MarketRisk = math.Min(100, marketRisk)

	// 4. 技术风险（基于新币上线和事件）
	technicalRisk := 0.0
	if score.Data.HasNewListing {
		technicalRisk += 30 // 新币上线风险
	}
	if absChange > 20 {
		technicalRisk += 20 // 极端波动
	}
	if score.Data.MarketCapUSD != nil && *score.Data.MarketCapUSD < 50000000 {
		technicalRisk += 20 // 小市值币种
	}

	risk.TechnicalRisk = math.Min(100, technicalRisk)

	// 5. 综合风险（加权平均）
	risk.OverallRisk = (risk.VolatilityRisk*0.3 + risk.LiquidityRisk*0.3 + 
		risk.MarketRisk*0.25 + risk.TechnicalRisk*0.15)

	// 6. 风险等级
	if risk.OverallRisk < 30 {
		risk.RiskLevel = "low"
	} else if risk.OverallRisk < 60 {
		risk.RiskLevel = "medium"
	} else {
		risk.RiskLevel = "high"
	}

	// 7. 生成风险提示
	risk.RiskWarnings = s.generateRiskWarnings(risk, score)

	return risk
}

// generateRiskWarnings 生成风险提示
func (s *Server) generateRiskWarnings(risk struct {
	VolatilityRisk  float64
	LiquidityRisk   float64
	MarketRisk      float64
	TechnicalRisk   float64
	OverallRisk     float64
	RiskLevel       string
	RiskWarnings    []string
}, score RecommendationScore) []string {
	warnings := make([]string, 0)

	if risk.VolatilityRisk > 70 {
		warnings = append(warnings, fmt.Sprintf("价格波动极大（24h涨跌幅%.2f%%），存在较高波动风险", score.Data.PriceChange24h))
	}

	if risk.LiquidityRisk > 70 {
		warnings = append(warnings, "流动性较低，可能存在买卖价差较大或难以成交的风险")
	}

	if risk.MarketRisk > 70 {
		warnings = append(warnings, "市值较小或排名靠后，市场认可度较低，存在归零风险")
	}

	if risk.TechnicalRisk > 50 {
		if score.Data.HasNewListing {
			warnings = append(warnings, "新币上线，项目成熟度较低，存在技术或团队风险")
		}
	}

	if score.Data.MarketCapUSD != nil && *score.Data.MarketCapUSD < 50000000 {
		warnings = append(warnings, "市值小于5000万USD，属于小市值币种，风险较高")
	}

	if score.Data.Volume24h < 2000000 {
		warnings = append(warnings, "24h成交量较低，可能存在流动性不足的风险")
	}

	if risk.OverallRisk > 70 {
		warnings = append(warnings, "综合风险评级：高风险，建议谨慎投资，控制仓位")
	} else if risk.OverallRisk > 50 {
		warnings = append(warnings, "综合风险评级：中等风险，建议适度投资")
	}

	if len(warnings) == 0 {
		warnings = append(warnings, "风险评级：低风险，但仍需注意市场波动")
	}

	return warnings
}
