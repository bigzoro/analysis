package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// generateReasons 生成推荐理由
func (s *Server) generateReasons(score RecommendationScore, currentPrice float64) []string {
	reasons := []string{}

	// 基于评分生成理由
	if score.TotalScore >= 0.8 {
		reasons = append(reasons, "综合评分优秀")
	} else if score.TotalScore >= 0.6 {
		reasons = append(reasons, "综合评分良好")
	}

	if score.Scores.Technical >= 0.7 {
		reasons = append(reasons, "技术指标向好")
	}

	if score.Scores.Fundamental >= 0.7 {
		reasons = append(reasons, "基本面稳健")
	}

	if score.Scores.Sentiment >= 0.7 {
		reasons = append(reasons, "市场情绪积极")
	}

	if score.Scores.Momentum >= 0.7 {
		reasons = append(reasons, "动量指标强劲")
	}

	// 基于策略类型添加理由
	switch score.StrategyType {
	case "LONG":
		reasons = append(reasons, "适合长期持有策略")
	case "SHORT":
		reasons = append(reasons, "建议适度减仓")
	case "RANGE":
		reasons = append(reasons, "适合区间交易")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "基于综合分析")
	}

	return reasons
}

// generateTechnicalReasons 生成技术分析理由
func (s *Server) generateTechnicalReasons(tech *TechnicalIndicators, currentPrice float64) []string {
	reasons := []string{}

	if tech == nil {
		return reasons
	}

	// RSI分析
	if tech.RSI < 30 {
		reasons = append(reasons, "RSI指标显示超卖")
	} else if tech.RSI > 70 {
		reasons = append(reasons, "RSI指标显示超买")
	} else {
		reasons = append(reasons, "RSI指标在正常区间")
	}

	// 移动平均线分析
	if tech.MA20 > tech.MA50 {
		reasons = append(reasons, "短期均线上穿长期均线")
	} else {
		reasons = append(reasons, "短期均线下穿长期均线")
	}

	// MACD分析
	if tech.MACD > tech.MACDSignal {
		reasons = append(reasons, "MACD金叉信号")
	} else {
		reasons = append(reasons, "MACD死叉信号")
	}

	// 布林带分析
	if currentPrice < tech.BollingerLower {
		reasons = append(reasons, "价格触及布林带下轨")
	} else if currentPrice > tech.BollingerUpper {
		reasons = append(reasons, "价格触及布林带上轨")
	}

	return reasons
}

// getFlowDataForRecommendation 获取资金流数据
func (s *Server) getFlowDataForRecommendation(ctx context.Context) (map[string]float64, error) {
	flowData := make(map[string]float64)

	// 从数据库获取最近24小时的资金流数据
	query := s.db.DB().Model(&pdb.DailyFlow{}).
		Select("coin as symbol, SUM(net) as total_flow").
		Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
		Group("coin")

	rows, err := query.Rows()
	if err != nil {
		return flowData, err
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		var flow float64
		if err := rows.Scan(&symbol, &flow); err == nil {
			flowData[symbol] = flow
		}
	}

	return flowData, nil
}

// getFlowDataForRecommendationForDate 获取指定日期的资金流数据
func (s *Server) getFlowDataForRecommendationForDate(ctx context.Context, targetDate time.Time) (map[string]float64, error) {
	flowData := make(map[string]float64)

	// 获取指定日期的资金流数据
	startDate := targetDate.Truncate(24 * time.Hour)
	endDate := startDate.Add(24 * time.Hour)

	query := s.db.DB().Model(&pdb.DailyFlow{}).
		Select("coin as symbol, SUM(net) as total_flow").
		Where("day >= ? AND day < ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Group("coin")

	rows, err := query.Rows()
	if err != nil {
		return flowData, err
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		var flow float64
		if err := rows.Scan(&symbol, &flow); err == nil {
			flowData[symbol] = flow
		}
	}

	return flowData, nil
}

// getAnnouncementDataForRecommendation 获取公告数据
func (s *Server) getAnnouncementDataForRecommendation(ctx context.Context) (map[string]bool, error) {
	announcementData := make(map[string]bool)

	// 获取最近24小时的公告数据
	query := s.db.DB().Model(&pdb.Announcement{}).
		Select("DISTINCT symbol").
		Where("created_at >= ? AND importance > ?", time.Now().Add(-24*time.Hour), 0.5)

	rows, err := query.Rows()
	if err != nil {
		return announcementData, err
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err == nil {
			announcementData[symbol] = true
		}
	}

	return announcementData, nil
}

// getAnnouncementDataForRecommendationForDate 获取指定日期的公告数据
func (s *Server) getAnnouncementDataForRecommendationForDate(ctx context.Context, targetDate time.Time) (map[string]bool, error) {
	announcementData := make(map[string]bool)

	// 获取指定日期的公告数据
	startDate := targetDate.Truncate(24 * time.Hour)
	endDate := startDate.Add(24 * time.Hour)

	query := s.db.DB().Model(&pdb.Announcement{}).
		Select("DISTINCT symbol").
		Where("created_at >= ? AND created_at < ? AND importance > ?", startDate, endDate, 0.5)

	rows, err := query.Rows()
	if err != nil {
		return announcementData, err
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err == nil {
			announcementData[symbol] = true
		}
	}

	return announcementData, nil
}

// getHistoricalPrices 获取历史价格数据
func (s *Server) getHistoricalPrices(ctx context.Context, symbol, timeRange string) ([]float64, error) {
	var prices []float64

	// 根据时间范围确定查询参数
	var days int
	switch timeRange {
	case "1d":
		days = 1
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		days = 30
	}

	// 从数据库查询历史价格
	query := s.db.DB().Model(&pdb.MarketKline{}).
		Select("close_price").
		Where("symbol = ? AND open_time >= ?", symbol, time.Now().AddDate(0, 0, -days)).
		Order("open_time ASC")

	rows, err := query.Rows()
	if err != nil {
		return prices, err
	}
	defer rows.Close()

	for rows.Next() {
		var price float64
		if err := rows.Scan(&price); err == nil {
			prices = append(prices, price)
		}
	}

	return prices, nil
}

// getHistoricalReturns 获取历史收益率数据
func (s *Server) getHistoricalReturns(ctx context.Context, symbol, timeRange string) ([]float64, error) {
	prices, err := s.getHistoricalPrices(ctx, symbol, timeRange)
	if err != nil || len(prices) < 2 {
		return []float64{}, err
	}

	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	return returns, nil
}

// getMarketReturns 获取市场收益率数据
func (s *Server) getMarketReturns(marketData []MarketDataPoint) []float64 {
	if len(marketData) < 2 {
		return []float64{}
	}

	returns := make([]float64, len(marketData)-1)
	for i := 1; i < len(marketData); i++ {
		if marketData[i-1].Price != 0 {
			returns[i-1] = (marketData[i].Price - marketData[i-1].Price) / marketData[i-1].Price
		}
	}

	return returns
}

// generateMockMarketReturns 生成模拟市场收益率
func (s *Server) generateMockMarketReturns(length int) []float64 {
	returns := make([]float64, length)
	for i := range returns {
		// 生成均值为0.0001，标准差为0.02的正态分布随机数
		returns[i] = 0.0001 + 0.02*(math.Sqrt(-2*math.Log(0.001+0.998*math.Abs(0.5)))*math.Cos(2*math.Pi*math.Abs(0.5)))
	}
	return returns
}

// calculateRiskBudget 计算风险预算
func (s *Server) calculateRiskBudget(weights map[string]float64, returns map[string][]float64, totalBudget float64) *RiskBudget {
	// 简化的风险预算计算
	riskContributions := make(map[string]float64)
	totalRisk := 0.0

	for symbol, weight := range weights {
		// 简化的风险贡献计算（基于权重和波动率）
		assetReturns := returns[symbol]
		if len(assetReturns) > 0 {
			volatility := s.calculateVolatility(assetReturns)
			riskContribution := weight * volatility
			riskContributions[symbol] = riskContribution
			totalRisk += riskContribution
		}
	}

	// 分配预算
	allocated := make(map[string]float64)
	for symbol, riskContribution := range riskContributions {
		if totalRisk > 0 {
			allocated[symbol] = (riskContribution / totalRisk) * totalBudget
		}
	}

	return &RiskBudget{
		TotalBudget:  totalBudget,
		AssetBudgets: allocated,
		Utilization:  riskContributions,
	}
}

// calculateVolatility 计算波动率
func (s *Server) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 计算均值
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// 计算方差
	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1) // 样本方差

	return math.Sqrt(variance)
}

// sumMapValues 计算map中所有值的总和
func (s *Server) sumMapValues(m map[string]float64) float64 {
	total := 0.0
	for _, v := range m {
		total += v
	}
	return total
}

// estimateMarketTrend 估算当前市场趋势（简化版）
func (s *Server) estimateMarketTrend(ctx context.Context) string {
	// 在实际实现中，这里应该分析最近的市场数据
	// 这里使用简化的估算逻辑

	now := time.Now()
	hour := now.Hour()

	// 简化的市场趋势估算（基于时间）
	if hour >= 9 && hour <= 15 { // 白天交易时间
		return "neutral"
	} else if hour >= 0 && hour <= 6 { // 亚洲交易时间
		return "bull"
	} else {
		return "bear"
	}
}

// calculateTrendAdjustment 根据市场趋势调整评分
func (s *Server) calculateTrendAdjustment(scores struct {
	baseScore   float64
	volatility  float64
	trend       float64
	fundamental float64
	marketCap   string
	category    string
}, marketTrend string) struct {
	technical   float64
	fundamental float64
	sentiment   float64
	momentum    float64
	risk        float64
} {
	adjustment := struct {
		technical   float64
		fundamental float64
		sentiment   float64
		momentum    float64
		risk        float64
	}{0, 0, 0, 0, 0}

	// 根据市场趋势调整不同类别的币种
	switch marketTrend {
	case "bull":
		// 牛市偏好高风险高收益币种
		if scores.category == "defi" || scores.category == "smart_contract" {
			adjustment.momentum += 0.1
			adjustment.sentiment += 0.05
		}
		if scores.marketCap == "large" {
			adjustment.fundamental += 0.02
		}
	case "bear":
		// 熊市偏好避险资产
		if scores.category == "store_of_value" {
			adjustment.fundamental += 0.08
			adjustment.risk -= 0.05
		}
		adjustment.technical += 0.02 // 技术分析在熊市更重要
	case "neutral":
		// 中性市场，均衡调整
		adjustment.fundamental += 0.01
		adjustment.technical += 0.01
	}

	return adjustment
}

// calculateAdaptiveScore 计算自适应综合评分
func (s *Server) calculateAdaptiveScore(technical, fundamental, sentiment, momentum, risk float64, category string) float64 {
	// 根据币种类别调整权重
	var weights struct {
		technical   float64
		fundamental float64
		sentiment   float64
		momentum    float64
		risk        float64
	}

	switch category {
	case "store_of_value":
		// 避险资产：基本面和技术分析更重要
		weights = struct {
			technical   float64
			fundamental float64
			sentiment   float64
			momentum    float64
			risk        float64
		}{0.2, 0.4, 0.15, 0.15, 0.1}
	case "smart_contract":
		// 智能合约平台：技术创新更重要
		weights = struct {
			technical   float64
			fundamental float64
			sentiment   float64
			momentum    float64
			risk        float64
		}{0.3, 0.25, 0.2, 0.2, 0.05}
	case "defi":
		// DeFi：创新和市场情绪更重要
		weights = struct {
			technical   float64
			fundamental float64
			sentiment   float64
			momentum    float64
			risk        float64
		}{0.2, 0.15, 0.25, 0.3, 0.1}
	case "oracle":
		// 预言机：技术可靠性和基本面重要
		weights = struct {
			technical   float64
			fundamental float64
			sentiment   float64
			momentum    float64
			risk        float64
		}{0.35, 0.3, 0.15, 0.15, 0.05}
	default:
		// 默认权重
		weights = struct {
			technical   float64
			fundamental float64
			sentiment   float64
			momentum    float64
			risk        float64
		}{0.25, 0.25, 0.2, 0.25, 0.05}
	}

	// 计算加权评分
	score := technical*weights.technical +
		fundamental*weights.fundamental +
		sentiment*weights.sentiment +
		momentum*weights.momentum -
		risk*weights.risk // 风险是负权重

	// 确保评分在合理范围内
	return math.Max(0, math.Min(1, score))
}

// getMarketDataForAlgorithm 获取算法使用的市场数据
func (s *Server) getMarketDataForAlgorithm(ctx context.Context, kind string, targetDate time.Time) ([]MarketDataPoint, error) {
	log.Printf("[MARKET_DATA] 开始获取%s类型的市场数据，目标时间: %s", kind, targetDate.Format(time.RFC3339))

	// 首先尝试精确时间范围查询
	timeRange := 1 * time.Hour
	startTime := targetDate.Add(-timeRange)
	endTime := targetDate.Add(timeRange)

	query := s.db.DB().Model(&pdb.BinanceMarketTop{}).
		Select("binance_market_tops.symbol, binance_market_tops.last_price, binance_market_tops.pct_change, binance_market_tops.volume, binance_market_tops.market_cap_usd, binance_market_tops.created_at").
		Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
		Where("binance_market_snapshots.kind = ? AND binance_market_tops.created_at >= ? AND binance_market_tops.created_at <= ?", kind, startTime, endTime).
		Order("binance_market_tops.created_at DESC").
		Limit(1000)

	var items []pdb.BinanceMarketTop
	if err := query.Find(&items).Error; err != nil {
		log.Printf("[MARKET_DATA] 精确时间范围查询失败: %v", err)
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}

	// 如果精确查询没有找到数据，尝试更宽泛的时间范围
	if len(items) == 0 {
		log.Printf("[MARKET_DATA] 精确查询未找到数据，尝试更宽泛的时间范围...")
		timeRange = 24 * time.Hour // 扩大到24小时
		startTime = targetDate.Add(-timeRange)
		endTime = targetDate.Add(timeRange)

		query = s.db.DB().Model(&pdb.BinanceMarketTop{}).
			Select("binance_market_tops.symbol, binance_market_tops.last_price, binance_market_tops.pct_change, binance_market_tops.volume, binance_market_tops.market_cap_usd, binance_market_tops.created_at").
			Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
			Where("binance_market_snapshots.kind = ? AND binance_market_tops.created_at >= ? AND binance_market_tops.created_at <= ?", kind, startTime, endTime).
			Order("binance_market_tops.created_at DESC").
			Limit(1000)

		if err := query.Find(&items).Error; err != nil {
			log.Printf("[MARKET_DATA] 宽泛时间范围查询失败: %v", err)
			return nil, fmt.Errorf("failed to query market data with extended range: %w", err)
		}
	}

	// 如果仍然没有数据，尝试获取最新的数据
	if len(items) == 0 {
		log.Printf("[MARKET_DATA] 时间范围查询未找到数据，尝试获取最新数据...")
		query = s.db.DB().Model(&pdb.BinanceMarketTop{}).
			Select("binance_market_tops.symbol, binance_market_tops.last_price, binance_market_tops.pct_change, binance_market_tops.volume, binance_market_tops.market_cap_usd, binance_market_tops.created_at").
			Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
			Where("binance_market_snapshots.kind = ?", kind).
			Order("binance_market_tops.created_at DESC").
			Limit(1000)

		if err := query.Find(&items).Error; err != nil {
			log.Printf("[MARKET_DATA] 最新数据查询失败: %v", err)
			return nil, fmt.Errorf("failed to query latest market data: %w", err)
		}
	}

	if len(items) == 0 {
		log.Printf("[MARKET_DATA] 数据库中没有找到任何%s类型的数据", kind)
		return nil, fmt.Errorf("no market data found for kind: %s", kind)
	}

	log.Printf("[MARKET_DATA] 成功获取%d条市场数据记录", len(items))

	// 转换为MarketDataPoint格式
	var dataPoints []MarketDataPoint
	validCount := 0
	for _, item := range items {
		// 数据验证
		price := parseFloat(item.LastPrice)
		volume := parseFloat(item.Volume)

		if price <= 0 || volume <= 0 {
			continue // 跳过无效数据
		}

		point := MarketDataPoint{
			Symbol:         item.Symbol,
			Price:          price,
			Volume24h:      volume,
			PriceChange24h: item.PctChange,
			MarketCap:      item.MarketCapUSD,
			Timestamp:      item.CreatedAt,
		}
		dataPoints = append(dataPoints, point)
		validCount++
	}

	log.Printf("[MARKET_DATA] 数据转换完成，有效数据点: %d/%d", validCount, len(items))

	if len(dataPoints) == 0 {
		return nil, fmt.Errorf("no valid market data points found")
	}

	return dataPoints, nil
}

// getFallbackMarketData 获取后备市场数据（当主要数据源失败时使用）
func (s *Server) getFallbackMarketData(ctx context.Context, kind string, symbols []string) ([]MarketDataPoint, error) {
	log.Printf("[FALLBACK_DATA] 开始获取%s类型的后备市场数据，指定币种: %v", kind, symbols)

	var query *gorm.DB

	if len(symbols) == 0 {
		// 如果没有指定符号，获取市值最大的主流币种
		log.Printf("[FALLBACK_DATA] 未指定币种，获取市值最大的币种...")
		query = s.db.DB().Model(&pdb.BinanceMarketTop{}).
			Select("binance_market_tops.symbol, binance_market_tops.last_price, binance_market_tops.pct_change, binance_market_tops.volume, binance_market_tops.market_cap_usd, binance_market_tops.created_at").
			Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
			Where("binance_market_snapshots.kind = ? AND binance_market_tops.market_cap_usd IS NOT NULL", kind).
			Order("binance_market_tops.market_cap_usd DESC").
			Limit(50) // 减少数量以提高数据质量
	} else {
		// 获取指定符号的数据
		log.Printf("[FALLBACK_DATA] 获取指定币种的数据...")
		query = s.db.DB().Model(&pdb.BinanceMarketTop{}).
			Select("binance_market_tops.symbol, binance_market_tops.last_price, binance_market_tops.pct_change, binance_market_tops.volume, binance_market_tops.market_cap_usd, binance_market_tops.created_at").
			Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
			Where("binance_market_snapshots.kind = ? AND binance_market_tops.symbol IN ?", kind, symbols).
			Order("binance_market_tops.created_at DESC")
	}

	var items []pdb.BinanceMarketTop
	if err := query.Find(&items).Error; err != nil {
		log.Printf("[FALLBACK_DATA] 后备数据查询失败: %v", err)
		return nil, fmt.Errorf("failed to query fallback market data: %w", err)
	}

	if len(items) == 0 {
		log.Printf("[FALLBACK_DATA] 未找到任何后备市场数据")
		return nil, fmt.Errorf("no fallback market data found")
	}

	log.Printf("[FALLBACK_DATA] 找到%d条后备数据记录", len(items))

	// 转换为MarketDataPoint格式，并进行数据验证
	var dataPoints []MarketDataPoint
	validCount := 0
	invalidCount := 0

	for _, item := range items {
		// 数据验证
		price := parseFloat(item.LastPrice)
		volume := parseFloat(item.Volume)

		if price <= 0 {
			invalidCount++
			continue // 跳过价格无效的数据
		}

		if volume <= 0 {
			volume = 1.0 // 为没有成交量的数据设置最小值
		}

		point := MarketDataPoint{
			Symbol:         item.Symbol,
			Price:          price,
			Volume24h:      volume,
			PriceChange24h: item.PctChange,
			MarketCap:      item.MarketCapUSD,
			Timestamp:      item.CreatedAt,
		}
		dataPoints = append(dataPoints, point)
		validCount++
	}

	log.Printf("[FALLBACK_DATA] 数据转换完成，有效数据点: %d/%d (无效: %d)",
		validCount, len(items), invalidCount)

	if len(dataPoints) == 0 {
		return nil, fmt.Errorf("no valid fallback market data points found")
	}

	return dataPoints, nil
}

// getCurrentMarketState 获取当前市场状态
func (s *Server) getCurrentMarketState(ctx context.Context, kind string) MarketState {
	// 这里应该实现市场状态检测逻辑
	// 暂时返回默认状态
	return MarketState{
		State:        "sideways",
		AvgChange:    0.0,
		Volatility:   5.0,
		UpRatio:      0.5,
		VolumeChange: 0.0,
	}
}

// getFlowDataForSymbol 获取单个交易对的资金流数据
func (s *Server) getFlowDataForSymbol(ctx context.Context, baseSymbol string) (FlowTrendResult, error) {
	// 这里应该实现获取单个交易对资金流数据的逻辑
	// 暂时返回默认值
	return FlowTrendResult{
		Flow24h: 0.0,
		Trend3d: 0.0,
		Trend7d: 0.0,
	}, nil
}

// getAnnouncementDataForSymbol 获取单个交易对的公告数据
func (s *Server) getAnnouncementDataForSymbol(ctx context.Context, baseSymbol string) (AnnouncementScore, error) {
	// 如果NewsAPI客户端未配置，返回默认值
	if s.newsAPIClient == nil {
		return AnnouncementScore{
			TotalScore: 0.0,
			Importance: "low",
		}, nil
	}

	// 使用NewsAPI获取相关新闻
	announcements, err := s.newsAPIClient.GetCryptoNews(ctx, baseSymbol, 7) // 最近7天的新闻
	if err != nil {
		log.Printf("[Announcement] 获取%s公告数据失败: %v，使用默认值", baseSymbol, err)
		return AnnouncementScore{
			TotalScore: 0.0,
			Importance: "low",
		}, nil
	}

	// 转换为公告评分
	score := s.newsAPIClient.ConvertToAnnouncementScore(announcements)

	log.Printf("[Announcement] %s 公告评分: %.2f (重要性: %s, 新闻数量: %d)",
		baseSymbol, score.TotalScore, score.Importance, len(announcements))

	return score, nil
}

// parseFloat 安全地解析字符串为float64
func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}
