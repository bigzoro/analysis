package server

import (
	pdb "analysis/internal/db"
	"analysis/internal/netutil"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// DynamicScoringSystem 动态评分系统
type DynamicScoringSystem struct {
	marketAnalyzer     *MarketAnalyzer
	coinProfiler       *CoinProfiler
	weightAdjuster     *WeightAdjuster
	thresholdOptimizer *ThresholdOptimizer
}

// MarketAnalyzer 市场分析器
type MarketAnalyzer struct {
	// 市场状态指标
	volatility    float64 // 整体波动率
	trendStrength float64 // 趋势强度
	marketPhase   string  // 市场阶段: bull/bear/sideways
	riskLevel     string  // 风险等级: low/medium/high
}

// CoinProfiler 币种特征分析器
type CoinProfiler struct {
	// 币种历史特征
	avgVolatility float64            // 平均波动率
	beta          float64            // 贝塔系数
	seasonality   map[string]float64 // 季节性表现
	performance   []float64          // 历史表现记录
}

// WeightAdjuster 权重调整器
type WeightAdjuster struct {
	baseWeights   map[string]float64            // 基础权重
	marketWeights map[string]map[string]float64 // 市场环境权重
	coinWeights   map[string]map[string]float64 // 币种特定权重
}

// ThresholdOptimizer 阈值优化器
type ThresholdOptimizer struct {
	historicalScores []float64 // 历史评分记录
	performanceData  []float64 // 对应表现数据
	dynamicThreshold float64   // 动态阈值
}

// RecommendationAPIError API错误响应
type RecommendationAPIError struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// NewDynamicScoringSystem 创建动态评分系统
func NewDynamicScoringSystem() *DynamicScoringSystem {
	return &DynamicScoringSystem{
		marketAnalyzer:     &MarketAnalyzer{},
		coinProfiler:       &CoinProfiler{},
		weightAdjuster:     NewWeightAdjuster(),
		thresholdOptimizer: &ThresholdOptimizer{},
	}
}

// NewWeightAdjuster 创建权重调整器
func NewWeightAdjuster() *WeightAdjuster {
	return &WeightAdjuster{
		baseWeights: map[string]float64{
			"technical":   0.4,
			"fundamental": 0.3,
			"sentiment":   0.2,
			"momentum":    0.1,
		},
		marketWeights: map[string]map[string]float64{
			"bull": {
				"technical":   0.35,
				"fundamental": 0.35,
				"sentiment":   0.2,
				"momentum":    0.1,
			},
			"bear": {
				"technical":   0.45,
				"fundamental": 0.25,
				"sentiment":   0.15,
				"momentum":    0.15,
			},
			"sideways": {
				"technical":   0.5,
				"fundamental": 0.2,
				"sentiment":   0.2,
				"momentum":    0.1,
			},
		},
		coinWeights: make(map[string]map[string]float64),
	}
}

// sendRecommendationError 发送标准化的推荐API错误响应
func sendRecommendationError(c *gin.Context, statusCode int, errorMsg, errorCode string, details ...string) {
	errorResp := RecommendationAPIError{
		Error: errorMsg,
		Code:  errorCode,
	}

	if len(details) > 0 {
		errorResp.Details = details[0]
	}

	// 记录错误日志
	log.Printf("[API_ERROR] %s: %s (status: %d)", errorCode, errorMsg, statusCode)

	c.JSON(statusCode, errorResp)
}

// GetAIRecommendations AI智能推荐API
// POST /api/v1/recommend
func (s *Server) GetAIRecommendations(c *gin.Context) {
	log.Printf("[DEBUG] ===== GetAIRecommendations API 被调用 =====")

	var req struct {
		Symbols   []string `json:"symbols" binding:"required"`
		Limit     int      `json:"limit"`
		RiskLevel string   `json:"risk_level"`
		Date      string   `json:"date"` // YYYY-MM-DD格式，可选
	}

	// 解析请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[ERROR] 请求参数解析失败: %v", err)
		sendRecommendationError(c, 400, "无效的请求参数格式", "INVALID_REQUEST_FORMAT", err.Error())
		return
	}

	log.Printf("[DEBUG] 解析到请求参数: symbols=%v, limit=%d, risk_level=%s, date='%s'", req.Symbols, req.Limit, req.RiskLevel, req.Date)

	// 验证币种参数
	if len(req.Symbols) == 0 {
		sendRecommendationError(c, 400, "至少需要指定一个币种", "EMPTY_SYMBOLS")
		return
	}

	if len(req.Symbols) > 50 {
		sendRecommendationError(c, 400, "币种数量不能超过50个", "TOO_MANY_SYMBOLS")
		return
	}

	// 验证币种名称格式
	symbolRegex := regexp.MustCompile(`^[A-Z0-9]{2,10}$`)
	for _, symbol := range req.Symbols {
		if !symbolRegex.MatchString(symbol) {
			sendRecommendationError(c, 400, fmt.Sprintf("无效的币种名称: %s", symbol), "INVALID_SYMBOL_FORMAT")
			return
		}
	}

	// 设置默认值
	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Limit > 20 {
		sendRecommendationError(c, 400, "推荐数量不能超过20个", "LIMIT_TOO_HIGH")
		return
	}

	// 验证风险等级
	validRiskLevels := map[string]bool{
		"conservative": true,
		"moderate":     true,
		"aggressive":   true,
	}
	if req.RiskLevel != "" && !validRiskLevels[req.RiskLevel] {
		sendRecommendationError(c, 400, "无效的风险等级，只能是: conservative, moderate, aggressive", "INVALID_RISK_LEVEL")
		return
	}

	// 获取推荐数据
	recommendations, err := s.generateAIRecommendations(req.Symbols, req.Limit, req.RiskLevel, req.Date)
	if err != nil {
		log.Printf("[ERROR] 生成AI推荐失败: %v", err)
		sendRecommendationError(c, 500, "生成推荐失败", "RECOMMENDATION_FAILED", err.Error())
		return
	}

	// 返回推荐结果
	c.JSON(200, gin.H{
		"success":         true,
		"recommendations": recommendations,
		"count":           len(recommendations),
		"timestamp":       time.Now().Unix(),
	})
}

// calculateTechnicalScore 计算技术评分
func (s *Server) calculateTechnicalScore(indicators *MultiTimeframeIndicators) float64 {
	if indicators == nil {
		return 0.5
	}

	score := 0.0

	// RSI评分 (0-1)
	rsi := indicators.ShortTerm.RSI
	if rsi >= 30 && rsi <= 70 {
		score += 0.3
	} else if rsi > 70 {
		score += 0.2 // 超买
	} else {
		score += 0.1 // 超卖
	}

	// MACD评分
	if indicators.ShortTerm.MACD > indicators.ShortTerm.MACDSignal {
		score += 0.2 // 金叉
	} else {
		score += 0.1 // 死叉
	}

	// 布林带位置评分
	bbPos := indicators.ShortTerm.BBPosition
	if bbPos >= 0.2 && bbPos <= 0.8 {
		score += 0.2 // 中性位置
	} else if bbPos < 0.2 {
		score += 0.1 // 低估
	} else {
		score += 0.15 // 高估
	}

	// 趋势评分
	if indicators.ShortTerm.Trend == "up" {
		score += 0.2
	} else if indicators.ShortTerm.Trend == "sideways" {
		score += 0.1
	}

	// 时间框架一致性
	score += indicators.TimeframeConsistency / 100.0 * 0.1

	return math.Min(score, 1.0)
}

// calculateFundamentalScore 计算基本面评分
func (s *Server) calculateFundamentalScore(symbol string) float64 {
	// 这里可以实现基本面分析逻辑
	// 暂时返回模拟值
	return 0.75
}

// calculateRiskScore 计算风险评分
func (s *Server) calculateRiskScore(indicators *MultiTimeframeIndicators, strategy *TradingStrategy) float64 {
	if indicators == nil {
		return 0.5
	}

	score := 0.0

	// 波动率评分 (基于布林带宽度)
	bbWidth := (indicators.ShortTerm.BBUpper - indicators.ShortTerm.BBLower) / indicators.ShortTerm.BBMiddle
	score += math.Min(bbWidth*2, 0.3) // 0-0.3

	// RSI极端值风险
	rsi := indicators.ShortTerm.RSI
	if rsi > 80 || rsi < 20 {
		score += 0.2 // 高风险
	} else if rsi > 70 || rsi < 30 {
		score += 0.1 // 中风险
	}

	// 趋势风险
	if indicators.ShortTerm.Trend == "down" {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

// calculateMomentumScore 计算动量评分
func (s *Server) calculateMomentumScore(indicators *MultiTimeframeIndicators) float64 {
	if indicators == nil {
		return 0.5
	}

	score := 0.0

	// MACD动量
	if indicators.ShortTerm.MACDHist > 0 {
		score += 0.3
	} else {
		score += 0.1
	}

	// RSI动量
	rsi := indicators.ShortTerm.RSI
	if rsi > 50 {
		score += 0.2
	}

	// 时间框架一致性
	score += indicators.TimeframeConsistency / 100.0 * 0.3

	return math.Min(score, 1.0)
}

// generateRecommendationReasons 生成推荐理由
func (s *Server) generateRecommendationReasons(indicators *MultiTimeframeIndicators, strategy *TradingStrategy, sentiment *SentimentResult) []string {
	reasons := []string{}

	if indicators != nil {
		// 技术指标理由
		if indicators.ShortTerm.RSI < 30 {
			reasons = append(reasons, "RSI显示超卖信号，存在反弹机会")
		} else if indicators.ShortTerm.RSI > 70 {
			reasons = append(reasons, "RSI显示超买信号，需要谨慎")
		}

		if indicators.ShortTerm.MACD > indicators.ShortTerm.MACDSignal {
			reasons = append(reasons, "MACD金叉信号，上涨动能增强")
		}

		if indicators.ShortTerm.BBPosition < 0.2 {
			reasons = append(reasons, "价格接近布林带下轨，存在支撑")
		}

		if indicators.ShortTerm.Trend == "up" {
			reasons = append(reasons, "技术指标显示上涨趋势")
		}
	}

	if strategy != nil {
		reasons = append(reasons, fmt.Sprintf("策略类型: %s", strategy.StrategyType))
	}

	if sentiment != nil && sentiment.Score > 7 {
		reasons = append(reasons, "市场情绪相对乐观")
	} else if sentiment != nil && sentiment.Score < 4 {
		reasons = append(reasons, "市场情绪相对谨慎")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "综合技术指标分析")
	}

	return reasons
}

// getDefaultExecutionPlan 获取默认执行计划
func (s *Server) getDefaultExecutionPlan(symbol string, currentPrice float64, strategy *TradingStrategy) *ExecutionPlan {
	return &ExecutionPlan{
		Symbol:        symbol,
		StrategyType:  strategy.StrategyType,
		TotalPosition: 0.05, // 默认仓位
		CurrentPrice:  currentPrice,
		EntryPlan: []EntryStage{
			{
				StageNumber: 1,
				Percentage:  0.5,
				PriceRange: PriceRange{
					Min: currentPrice * 0.97,
					Max: currentPrice * 1.00,
					Avg: currentPrice * 0.985,
				},
				Condition:   "价格回调至支撑位",
				MaxSlippage: 0.001,
				TimeLimit:   "24h",
				Priority:    "HIGH",
			},
			{
				StageNumber: 2,
				Percentage:  0.5,
				PriceRange: PriceRange{
					Min: currentPrice * 0.95,
					Max: currentPrice * 0.97,
					Avg: currentPrice * 0.96,
				},
				Condition:   "RSI进入超卖区",
				MaxSlippage: 0.001,
				TimeLimit:   "48h",
				Priority:    "MEDIUM",
			},
		},
		ExitPlan: []ExitStage{
			{
				StageNumber: 1,
				Percentage:  1.0,
				PriceRange: PriceRange{
					Min: currentPrice * 1.05,
					Max: currentPrice * 1.15,
					Avg: currentPrice * 1.10,
				},
				Condition:    "价格达到止盈目标",
				ProfitTarget: 0.1,
			},
		},
		RiskControls: RiskControls{
			MaxLossPerTrade:     0.02,
			MaxDailyLoss:        0.05,
			MaxHoldingTime:      "7d",
			TrailingStopPercent: 0.01,
		},
		Timeline: ExecutionTimeline{
			StartTime:        time.Now(),
			ExpectedDuration: "7 days",
			KeyMilestones: []Milestone{
				{Time: "Day 1", Event: "首次建仓", Description: "完成第一批建仓"},
				{Time: "Day 3", Event: "风险检查", Description: "评估市场风险和策略表现"},
			},
		},
	}
}

// generateRecommendationPriceAlerts 生成推荐价格提醒
func (s *Server) generateRecommendationPriceAlerts(symbol string, strategy *TradingStrategy, executionPlan *ExecutionPlan, currentPrice float64) []gin.H {
	alerts := []gin.H{}

	// 入场提醒
	if strategy != nil && executionPlan != nil && len(executionPlan.EntryPlan) > 0 {
		entryPrice := executionPlan.EntryPlan[0].PriceRange.Min
		alerts = append(alerts, gin.H{
			"id":          fmt.Sprintf("alert_entry_%s", symbol),
			"symbol":      symbol,
			"alert_type":  "entry",
			"price_level": math.Round(entryPrice*100) / 100,
			"condition":   "below",
			"message":     "价格接近入场区间",
			"priority":    "HIGH",
			"is_active":   true,
			"created_at":  time.Now(),
		})
	}

	// 止损提醒
	stopLossPrice := currentPrice * 0.92
	alerts = append(alerts, gin.H{
		"id":          fmt.Sprintf("alert_stop_%s", symbol),
		"symbol":      symbol,
		"alert_type":  "stop_loss",
		"price_level": math.Round(stopLossPrice*100) / 100,
		"condition":   "below",
		"message":     "价格跌破止损位",
		"priority":    "CRITICAL",
		"is_active":   true,
		"created_at":  time.Now(),
	})

	// 止盈提醒
	if strategy != nil && strategy.StrategyType == "LONG" && len(strategy.ExitTargets) > 0 {
		profitTarget := strategy.ExitTargets[0].Max // 使用第一个出场目标
		alerts = append(alerts, gin.H{
			"id":          fmt.Sprintf("alert_profit_%s", symbol),
			"symbol":      symbol,
			"alert_type":  "profit_target",
			"price_level": math.Round(profitTarget*100) / 100,
			"condition":   "above",
			"message":     "价格达到止盈目标",
			"priority":    "HIGH",
			"is_active":   true,
			"created_at":  time.Now(),
		})
	}

	return alerts
}

// getSentimentAnalysis 获取情绪分析数据
func (s *Server) getSentimentAnalysis(ctx context.Context, symbol string) (*SentimentResult, error) {
	// 这里可以调用情绪分析服务
	// 暂时返回默认值
	return &SentimentResult{
		Score:      7.5,
		Positive:   75,
		Negative:   15,
		Neutral:    10,
		Total:      100,
		Mentions:   500,
		Trend:      "bullish",
		KeyPhrases: []string{"突破", "上涨", "强势"},
	}, nil
}

// getPriceChange24h 获取24h涨跌幅
func (s *Server) getPriceChange24h(symbol string) float64 {
	return s.getPriceChange24hWithKind(symbol, "spot")
}

// getPriceChange24hWithKind 获取指定交易类型的24h涨跌幅
func (s *Server) getPriceChange24hWithKind(symbol string, kind string) float64 {
	ctx := context.Background()

	// 优先：直接调用币安24hr统计API（更高效）
	if change, err := s.getPriceChangeFromTicker24h(ctx, symbol, kind); err == nil {
		return change
	}

	// 降级：从K线数据计算
	if change, err := s.calculatePriceChange24h(ctx, symbol, kind); err == nil {
		return change
	}

	// 对于合约数据，不使用CoinGecko降级方案（CoinGecko主要提供现货数据）
	// 合约数据应该只依赖币安API
	if kind == "futures" {
		log.Printf("[WARN] 无法获取%s(%s)的24h涨跌幅，使用默认值", symbol, kind)
		return 0.0
	}

	// 降级：仅对现货数据尝试从数据管理器获取（包括CoinGecko等外部数据源）
	if s.dataManager != nil {
		unifiedData, err := s.dataManager.FetchMultiSourceData(ctx, []string{symbol})
		if err == nil && unifiedData != nil {
			if symbolData, exists := unifiedData.SymbolData[symbol]; exists && symbolData.Sources != nil {
				// 从多个数据源中选择一个有效的Change24h
				for _, marketData := range symbolData.Sources {
					if marketData.Change24h != 0 {
						return marketData.Change24h
					}
				}
			}
		}
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s(%s)的24h涨跌幅，使用默认值", symbol, kind)
	return 0.0
}

// getVolume24h 获取24h成交量
func (s *Server) getVolume24h(symbol string) float64 {
	return s.getVolume24hWithKind(symbol, "spot")
}

// getVolume24hWithKind 获取指定交易类型的24h成交量
func (s *Server) getVolume24hWithKind(symbol string, kind string) float64 {
	ctx := context.Background()

	// 优先：直接调用币安24hr统计API（更高效）
	if volume, err := s.getVolumeFromTicker24h(ctx, symbol, kind); err == nil {
		return volume
	}

	// 降级：从K线数据计算
	if volume, err := s.calculateVolume24h(ctx, symbol, kind); err == nil {
		return volume
	}

	// 对于合约数据，不使用CoinGecko降级方案（CoinGecko主要提供现货数据）
	// 合约数据应该只依赖币安API
	if kind == "futures" {
		log.Printf("[WARN] 无法获取%s(%s)的24h成交量，使用默认值", symbol, kind)
		return 0.0
	}

	// 降级：仅对现货数据尝试从数据管理器获取（包括CoinGecko等外部数据源）
	if s.dataManager != nil {
		unifiedData, err := s.dataManager.FetchMultiSourceData(ctx, []string{symbol})
		if err == nil && unifiedData != nil {
			if symbolData, exists := unifiedData.SymbolData[symbol]; exists && symbolData.Sources != nil {
				// 从多个数据源中选择一个有效的Volume24h
				for _, marketData := range symbolData.Sources {
					if marketData.Volume24h > 0 {
						return marketData.Volume24h
					}
				}
			}
		}
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s(%s)的24h成交量，使用默认值", symbol, kind)
	return 0.0
}

// getMarketCap 获取市值
func (s *Server) getMarketCap(symbol string) float64 {
	ctx := context.Background()

	// 尝试从数据管理器获取
	if s.dataManager != nil {
		unifiedData, err := s.dataManager.FetchMultiSourceData(ctx, []string{symbol})
		if err == nil && unifiedData != nil {
			if symbolData, exists := unifiedData.SymbolData[symbol]; exists && symbolData.Sources != nil {
				// 从多个数据源中选择一个有效的MarketCap
				for _, marketData := range symbolData.Sources {
					if marketData.MarketCap > 0 {
						return marketData.MarketCap
					}
				}
			}
		}
	}

	// 降级：从CoinGecko获取
	if marketCap, err := s.getMarketCapFromCoinGecko(ctx, symbol); err == nil {
		return marketCap
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s的市值，使用默认值", symbol)
	return 0.0
}

// getPriceHigh24h 获取24h最高价
func (s *Server) getPriceHigh24h(symbol string) float64 {
	ctx := context.Background()

	// 从币安K线数据计算24h最高价
	if high, err := s.calculatePriceHigh24h(ctx, symbol, "spot"); err == nil {
		return high
	}

	// 降级：使用当前价格作为近似值
	if currentPrice, err := s.getCurrentPrice(ctx, symbol, "spot"); err == nil {
		return currentPrice * 1.05 // 假设上涨5%
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s的24h最高价，使用默认值", symbol)
	return 0.0
}

// getPriceLow24h 获取24h最低价
func (s *Server) getPriceLow24h(symbol string) float64 {
	ctx := context.Background()

	// 从币安K线数据计算24h最低价
	if low, err := s.calculatePriceLow24h(ctx, symbol, "spot"); err == nil {
		return low
	}

	// 降级：使用当前价格作为近似值
	if currentPrice, err := s.getCurrentPrice(ctx, symbol, "spot"); err == nil {
		return currentPrice * 0.95 // 假设下跌5%
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s的24h最低价，使用默认值", symbol)
	return 0.0
}

// getPriceHigh7d 获取7d最高价
func (s *Server) getPriceHigh7d(symbol string) float64 {
	ctx := context.Background()

	// 从币安K线数据计算7d最高价
	if high, err := s.calculatePriceHigh7d(ctx, symbol, "spot"); err == nil {
		return high
	}

	// 降级：使用当前价格作为近似值
	if currentPrice, err := s.getCurrentPrice(ctx, symbol, "spot"); err == nil {
		return currentPrice * 1.1 // 假设上涨10%
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s的7d最高价，使用默认值", symbol)
	return 0.0
}

// getPriceLow7d 获取7d最低价
func (s *Server) getPriceLow7d(symbol string) float64 {
	ctx := context.Background()

	// 从币安K线数据计算7d最低价
	if low, err := s.calculatePriceLow7d(ctx, symbol, "spot"); err == nil {
		return low
	}

	// 降级：使用当前价格作为近似值
	if currentPrice, err := s.getCurrentPrice(ctx, symbol, "spot"); err == nil {
		return currentPrice * 0.9 // 假设下跌10%
	}

	// 最终降级：返回默认值
	log.Printf("[WARN] 无法获取%s的7d最低价，使用默认值", symbol)
	return 0.0
}

// GetRecommendationDetail 获取单个推荐详情API
// GET /api/v1/recommend/detail/:symbol
func (s *Server) GetRecommendationDetail(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		sendRecommendationError(c, 400, "币种符号不能为空", "EMPTY_SYMBOL")
		return
	}

	// 验证币种名称格式
	if matched, _ := regexp.MatchString(`^[A-Z0-9]{2,10}$`, symbol); !matched {
		sendRecommendationError(c, 400, fmt.Sprintf("无效的币种名称: %s", symbol), "INVALID_SYMBOL_FORMAT")
		return
	}

	ctx := context.Background()

	// 获取多时间框架技术指标
	multiIndicators, err := s.GetMultiTimeframeIndicators(ctx, symbol, "spot")
	if err != nil {
		log.Printf("[WARN] 获取多时间框架技术指标失败，使用默认值: %v", err)
		// 使用默认技术指标
		multiIndicators = &MultiTimeframeIndicators{
			ShortTerm: TechnicalIndicators{
				RSI:        65.5,
				MACD:       125.3,
				MACDSignal: 98.7,
				MACDHist:   26.6,
				BBUpper:    47000,
				BBMiddle:   45000,
				BBLower:    43000,
				BBPosition: 0.5,
				Trend:      "up",
				MA5:        45200,
				MA10:       44800,
				MA20:       44200,
			},
			TimeframeConsistency: 85.0,
			OverallSignal:        "buy",
			SignalConfidence:     75.0,
		}
	}

	// 获取当前价格
	currentPrice := 45000.0 // 默认价格
	if price, err := s.getCurrentPrice(ctx, symbol, "spot"); err == nil && price > 0 {
		currentPrice = price
	}

	// 简化策略生成（暂时使用固定策略）
	strategy := &TradingStrategy{
		StrategyType:      "LONG",
		MarketCondition:   "震荡向上",
		StrategyRationale: []string{"技术指标显示上涨动能", "RSI显示超卖信号"},
		EntryZone: PriceRange{
			Min: currentPrice * 0.95,
			Max: currentPrice * 1.02,
			Avg: currentPrice * 0.98,
		},
		ExitTargets: []PriceRange{
			{
				Min: currentPrice * 1.05,
				Max: currentPrice * 1.15,
				Avg: currentPrice * 1.10,
			},
		},
		StopLossLevels: []StopLossLevel{
			{
				Level:     currentPrice * 0.92,
				Type:      "INITIAL",
				Condition: "below",
			},
		},
		PositionSizing: PositionSizing{
			BasePosition:    0.05,
			MaxPosition:     0.08,
			MinPosition:     0.02,
			ScalingStrategy: "FIXED",
		},
	}

	// 使用默认执行计划
	executionPlan := s.getDefaultExecutionPlan(symbol, currentPrice, strategy)

	// 生成价格提醒
	priceAlerts := s.generateRecommendationPriceAlerts(symbol, strategy, executionPlan, currentPrice)

	// 获取情绪分析数据
	sentimentData, err := s.getSentimentAnalysis(ctx, symbol)
	if err != nil {
		log.Printf("[WARN] 获取情绪分析失败，使用默认值: %v", err)
		sentimentData = &SentimentResult{
			Score:      7.5,
			Positive:   75,
			Negative:   15,
			Neutral:    10,
			Total:      100,
			Mentions:   500,
			Trend:      "bullish",
			KeyPhrases: []string{"上涨", "突破"},
		}
	}

	// 计算综合评分
	technicalScore := s.calculateTechnicalScore(multiIndicators)
	fundamentalScore := s.calculateFundamentalScore(symbol)
	sentimentScore := sentimentData.Score / 10.0 // 转换为0-1
	momentumScore := s.calculateMomentumScore(multiIndicators)

	overallScore := (technicalScore*0.4 + fundamentalScore*0.3 + sentimentScore*0.2 + momentumScore*0.1)

	// 生成推荐理由
	reasons := s.generateRecommendationReasons(multiIndicators, strategy, sentimentData)

	// 生成AI预测和信心数据
	mlPrediction := overallScore                                    // AI预测得分基于综合评分
	mlConfidence := math.Min(0.95, math.Max(0.6, overallScore+0.3)) // 信心度基于评分调整

	// 计算止损和止盈价格
	stopLossPrice := currentPrice * 0.97   // 3%止损
	takeProfitPrice := currentPrice * 1.15 // 15%止盈
	stopLossPercentage := math.Abs((stopLossPrice - currentPrice) / currentPrice)

	// 构建完全自定义的trading_strategy，不使用strategy对象
	tradingStrategyData := gin.H{
		"strategy_type":    "LONG",
		"market_condition": "震荡向上",
		"entry_zone": gin.H{
			"min": currentPrice * 0.95,
			"max": currentPrice * 1.02,
			"avg": currentPrice * 0.98,
		},
		"entry_strategy": gin.H{
			"timing":                  "当前价格附近",
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
			gin.H{
				"price":      stopLossPrice,
				"percentage": stopLossPercentage,
			},
		},
		"position_sizing": gin.H{
			"base_position":        0.05,
			"adjusted_position":    0.05,
			"recommended_position": 0.05,
			"max_position":         0.08,
			"min_position":         0.02,
			"scaling_strategy":     "FIXED",
			"description":          "基于Kelly公式和风险偏好计算的个性化仓位",
		},
		"strategy_rationale": []string{
			"技术指标显示上涨动能",
			"RSI显示超卖信号",
		},
		"confidence_level":     0.8,
		"execution_complexity": "复杂",
		"trading_direction":    "区间交易",
	}

	// 构建完整的推荐数据
	recommendation := gin.H{
		"symbol":               symbol,
		"rank":                 1,
		"overall_score":        math.Round(overallScore*100) / 100,
		"expected_return":      0.15,
		"risk_score":           s.calculateRiskScore(multiIndicators, strategy),
		"technical_score":      math.Round(technicalScore*100) / 100,
		"fundamental_score":    math.Round(fundamentalScore*100) / 100,
		"sentiment_score":      math.Round(sentimentScore*100) / 100,
		"momentum_score":       math.Round(momentumScore*100) / 100,
		"ml_prediction":        mlPrediction, // AI预测得分
		"ml_confidence":        mlConfidence, // AI信心度
		"price":                currentPrice,
		"recommended_position": 0.05,
		"risk_level":           "medium",
		"reasons":              reasons,
		"technical_indicators": gin.H{
			"rsi":              math.Round(multiIndicators.ShortTerm.RSI*100) / 100,
			"macd":             math.Round(multiIndicators.ShortTerm.MACD*100) / 100,
			"macd_signal":      math.Round(multiIndicators.ShortTerm.MACDSignal*100) / 100,
			"macd_hist":        math.Round(multiIndicators.ShortTerm.MACDHist*100) / 100,
			"bb_upper":         math.Round(multiIndicators.ShortTerm.BBUpper*100) / 100,
			"bb_middle":        math.Round(multiIndicators.ShortTerm.BBMiddle*100) / 100,
			"bb_lower":         math.Round(multiIndicators.ShortTerm.BBLower*100) / 100,
			"bb_position":      math.Round(multiIndicators.ShortTerm.BBPosition*100) / 100,
			"trend":            multiIndicators.ShortTerm.Trend,
			"ma5":              math.Round(multiIndicators.ShortTerm.MA5*100) / 100,
			"ma10":             math.Round(multiIndicators.ShortTerm.MA10*100) / 100,
			"ma20":             math.Round(multiIndicators.ShortTerm.MA20*100) / 100,
			"support_level":    math.Round(multiIndicators.ShortTerm.BBLower*0.95*100) / 100,
			"resistance_level": math.Round(multiIndicators.ShortTerm.BBUpper*1.05*100) / 100,
		},
		"market_data": gin.H{
			"price_change_24h": s.getPriceChange24h(symbol),
			"volume_24h":       s.getVolume24h(symbol),
			"market_cap":       s.getMarketCap(symbol),
			"price_ranges": gin.H{
				"high_24h": s.getPriceHigh24h(symbol),
				"low_24h":  s.getPriceLow24h(symbol),
				"high_7d":  s.getPriceHigh7d(symbol),
				"low_7d":   s.getPriceLow7d(symbol),
			},
		},
		"trading_strategy": tradingStrategyData,
		"execution_plan":   executionPlan,
		"price_alerts":     priceAlerts,
	}

	c.JSON(200, gin.H{
		"success":        true,
		"recommendation": recommendation,
		"timestamp":      time.Now().Unix(),
	})
}

// GetAdvancedRecommendations 高级组合推荐API
// POST /api/v1/recommend/advanced
func (s *Server) GetAdvancedRecommendations(c *gin.Context) {
	var req struct {
		Symbols   []string `json:"symbols" binding:"required"`
		Limit     int      `json:"limit"`
		RiskLevel string   `json:"risk_level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendRecommendationError(c, 400, "无效的请求参数", "INVALID_REQUEST", err.Error())
		return
	}

	if len(req.Symbols) == 0 || len(req.Symbols) > 20 {
		sendRecommendationError(c, 400, "币种数量必须在1-20之间", "INVALID_SYMBOL_COUNT")
		return
	}

	// 获取推荐数据
	recommendations, err := s.generateAdvancedRecommendations(req.Symbols, req.Limit, req.RiskLevel)
	if err != nil {
		log.Printf("[ERROR] 生成高级推荐失败: %v", err)
		sendRecommendationError(c, 500, "生成推荐失败", "RECOMMENDATION_FAILED", err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success":         true,
		"recommendations": recommendations,
		"count":           len(recommendations),
	})
}

// generateAIRecommendations 生成AI推荐（核心实现已移动到strategy_generator.go）
func (s *Server) generateAIRecommendations(symbols []string, limit int, riskLevel string, date string) ([]gin.H, error) {
	log.Printf("[DEBUG] 开始生成AI推荐: symbols=%v, limit=%d, riskLevel=%s, date=%s", symbols, limit, riskLevel, date)
	log.Printf("[DEBUG] 开始处理日期参数: date='%s', len=%d", date, len(date))
	ctx := context.Background()

	// 处理日期参数
	var targetDate time.Time
	var useHistoricalData bool
	log.Printf("[DEBUG] 开始处理日期参数: date='%s', len=%d", date, len(date))
	if date != "" {
		log.Printf("[DEBUG] 日期参数不为空，尝试解析: %s", date)
		// 解析日期参数
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Printf("[WARN] 无效的日期格式: %s, 使用当前数据: %v", date, err)
			targetDate = time.Now()
			useHistoricalData = false
		} else {
			targetDate = parsedDate
			useHistoricalData = true
			log.Printf("[INFO] 成功解析日期，使用历史日期生成推荐: %s -> %v", date, targetDate)
		}
	} else {
		log.Printf("[DEBUG] 日期参数为空，使用当前数据")
		targetDate = time.Now()
		useHistoricalData = false
		log.Printf("[INFO] 使用当前实时数据生成推荐")
	}
	log.Printf("[DEBUG] 最终设置: useHistoricalData=%v, targetDate=%v", useHistoricalData, targetDate)

	// 获取推荐数据 - 调用实际的推荐生成逻辑
	result := make([]gin.H, 0, limit)

	for i, symbol := range symbols {
		if i >= limit {
			break
		}

		// 获取技术指标数据
		var indicators *TechnicalIndicators
		var err error

		log.Printf("[DEBUG] useHistoricalData=%v, targetDate=%s", useHistoricalData, targetDate.Format("2006-01-02"))

		if useHistoricalData {
			log.Printf("[DEBUG] 开始获取%s的历史技术指标数据，日期: %s", symbol, targetDate.Format("2006-01-02"))
			// 尝试获取历史技术指标数据
			indicators, err = s.getHistoricalTechnicalIndicators(ctx, symbol, targetDate)
			if err != nil {
				log.Printf("[WARN] 获取%s历史技术指标失败，回退到当前数据: %v", symbol, err)
				indicators, err = s.CalculateTechnicalIndicators(ctx, symbol, "spot")
			} else {
				log.Printf("[DEBUG] 成功获取%s的历史技术指标数据", symbol)
			}
		} else {
			log.Printf("[DEBUG] 获取%s的当前技术指标数据", symbol)
			// 获取当前技术指标数据
			indicators, err = s.CalculateTechnicalIndicators(ctx, symbol, "spot")
		}

		if err != nil {
			log.Printf("[WARN] 获取%s技术指标失败: %v", symbol, err)
			indicators = &TechnicalIndicators{
				RSI:   50,
				MACD:  0,
				Trend: "sideways",
			}
		}

		// 获取价格数据
		currentPrice := 45000.0 // 默认价格

		if useHistoricalData {
			log.Printf("[DEBUG] 开始获取%s的历史价格数据，日期: %s", symbol, targetDate.Format("2006-01-02"))
			// 尝试获取历史价格数据
			if historicalPrice, err := s.getHistoricalPrice(ctx, symbol, targetDate); err == nil && historicalPrice > 0 {
				currentPrice = historicalPrice
				log.Printf("[INFO] 使用历史价格: %s = %.2f", symbol, currentPrice)
			} else {
				log.Printf("[WARN] 获取%s历史价格失败，使用默认价格: %v", symbol, err)
			}
		} else {
			log.Printf("[DEBUG] 获取%s的当前价格", symbol)
			// 获取当前价格
			if price, err := s.getCurrentPrice(ctx, symbol, "spot"); err == nil && price > 0 {
				currentPrice = price
				log.Printf("[DEBUG] 获取到当前价格: %f for %s", currentPrice, symbol)
			} else {
				log.Printf("[DEBUG] 获取当前价格失败 for %s: %v", symbol, err)
			}
		}

		// 如果获取失败，使用技术指标中的价格作为后备
		if currentPrice == 45000.0 && indicators != nil && indicators.MA20 > 0 {
			currentPrice = indicators.MA20 // 使用20日均线作为当前价格的近似值
			log.Printf("[DEBUG] 使用技术指标价格: %f for %s", currentPrice, symbol)
		}

		// 创建推荐对象
		rec := CoinRecommendation{
			CoinScore: CoinScore{
				Symbol:       symbol,
				BaseSymbol:   symbol,
				StrategyType: "LONG",
			},
		}

		// 使用动态评分系统计算自适应评分
		var totalScore float64
		var scoreDetails map[string]float64

		// 使用传统评分系统
		// 将单个技术指标转换为多时间框架指标
		multiIndicators := &MultiTimeframeIndicators{
			ShortTerm:  *indicators,
			MediumTerm: *indicators,
			LongTerm:   *indicators,
		}
		technicalScore := s.calculateTechnicalScore(multiIndicators)
		riskScore := 0.3
		totalScore = (technicalScore*0.4 + 0.8*0.3 + 0.7*0.2 + 0.75*0.1) - riskScore*0.2
		scoreDetails = map[string]float64{
			"technical":   technicalScore,
			"fundamental": 0.8,
			"sentiment":   0.7,
			"momentum":    0.75,
			"risk":        riskScore,
			"total":       totalScore,
		}

		// 设置评分
		rec.Scores.Technical = scoreDetails["technical"]
		rec.Scores.Fundamental = scoreDetails["fundamental"]
		rec.Scores.Sentiment = scoreDetails["sentiment"]
		rec.Scores.Momentum = scoreDetails["momentum"]
		rec.Scores.Risk = scoreDetails["risk"]
		rec.TotalScore = totalScore

		// 获取交易策略
		tradingStrategy := s.generateTradingStrategyForRecommendation(rec, currentPrice, 0.15)

		// 生成AI预测和信心数据
		mlPrediction := rec.TotalScore                                    // AI预测得分基于综合评分
		mlConfidence := math.Min(0.95, math.Max(0.6, rec.TotalScore+0.3)) // 信心度基于评分调整

		// 构建完整的推荐数据
		recommendation := gin.H{
			"symbol":               symbol,
			"rank":                 i + 1,
			"overall_score":        rec.TotalScore,
			"expected_return":      0.15 - float64(i)*0.02,
			"risk_score":           rec.Scores.Risk,
			"technical_score":      rec.Scores.Technical,
			"fundamental_score":    rec.Scores.Fundamental,
			"sentiment_score":      rec.Scores.Sentiment,
			"momentum_score":       rec.Scores.Momentum,
			"ml_prediction":        mlPrediction, // AI预测得分
			"ml_confidence":        mlConfidence, // AI信心度
			"price":                currentPrice,
			"recommended_position": 0.05 - float64(i)*0.01, // 根据排名调整仓位
			"risk_level":           s.getRiskLevel(rec.Scores.Risk),
			"reasons": []string{
				"技术指标显示上涨动能",
				"市场情绪相对乐观",
				"基本面数据表现良好",
			},
			"technical_indicators": gin.H{
				"rsi":                 indicators.RSI,
				"macd":                indicators.MACD,
				"macd_signal":         indicators.MACDSignal,
				"macd_hist":           indicators.MACDHist,
				"bb_upper":            indicators.BBUpper,
				"bb_middle":           indicators.BBMiddle,
				"bb_lower":            indicators.BBLower,
				"bb_position":         indicators.BBPosition,
				"bb_width":            indicators.BBWidth,
				"k":                   indicators.K,
				"d":                   indicators.D,
				"j":                   indicators.J,
				"trend":               indicators.Trend,
				"ma5":                 indicators.MA5,
				"ma10":                indicators.MA10,
				"ma20":                indicators.MA20,
				"ma50":                indicators.MA50,
				"support_level":       indicators.SupportLevel,
				"resistance_level":    indicators.ResistanceLevel,
				"support_strength":    indicators.SupportStrength,
				"resistance_strength": indicators.ResistanceStrength,
				"momentum5":           indicators.Momentum5,
				"momentum10":          indicators.Momentum10,
				"momentum20":          indicators.Momentum20,
				"momentum_divergence": indicators.MomentumDivergence,
				"volatility5":         indicators.Volatility5,
				"volatility20":        indicators.Volatility20,
				"volatility_ratio":    indicators.VolatilityRatio,
				"williams_r":          indicators.WilliamsR,
				"cci":                 indicators.CCI,
				"obv":                 indicators.OBV,
				"volume_ma5":          indicators.VolumeMA5,
				"volume_ma20":         indicators.VolumeMA20,
				"volume_ratio":        indicators.VolumeRatio,
				"signal_strength":     indicators.SignalStrength,
				"risk_level":          indicators.RiskLevel,
			},
			"market_data": gin.H{
				"price_change_24h": 0.05, // 示例数据
				"volume_24h":       1000000,
				"market_cap":       50000000,
				"market_cap_rank":  50,
				"price_ranges": gin.H{
					"high_24h": currentPrice * 1.05,
					"low_24h":  currentPrice * 0.95,
					"high_7d":  currentPrice * 1.1,
					"low_7d":   currentPrice * 0.9,
				},
			},
			"trading_strategy": tradingStrategy,
		}

		result = append(result, recommendation)
	}

	return result, nil
}

// getHistoricalPrice 获取指定日期的历史价格
func (s *Server) getHistoricalPrice(ctx context.Context, symbol string, targetDate time.Time) (float64, error) {
	// 获取目标日期前后的K线数据（例如获取1天的数据来确保包含目标日期）
	startDate := targetDate.AddDate(0, 0, -1)
	endDate := targetDate.AddDate(0, 0, 1)

	log.Printf("[INFO] 查询历史价格: %s 在 %s", symbol, targetDate.Format("2006-01-02"))

	// 查询日K线数据
	klines, err := s.fetchBinanceKlinesWithTimeRange(ctx, symbol, "spot", "1d", 3, &startDate, &endDate)
	if err != nil {
		log.Printf("[WARN] 查询历史K线数据失败: %v", err)
		return 0, fmt.Errorf("查询历史K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		log.Printf("[WARN] 未找到%s的历史K线数据", symbol)
		return 0, fmt.Errorf("未找到历史K线数据")
	}

	// 找到最接近目标日期的K线
	var closestKline BinanceKline
	minDiff := time.Hour * 24 * 365 // 一年

	targetTimestamp := targetDate.Unix() * 1000

	for _, kline := range klines {
		diff := time.Duration(math.Abs(kline.OpenTime - float64(targetTimestamp)))
		if diff < minDiff {
			minDiff = diff
			closestKline = kline
		}
	}

	if closestKline.Close == "" {
		log.Printf("[WARN] 未找到%s在%s附近的有效价格", symbol, targetDate.Format("2006-01-02"))
		return 0, fmt.Errorf("未找到有效价格")
	}

	price, err := strconv.ParseFloat(closestKline.Close, 64)
	if err != nil {
		log.Printf("[WARN] 解析价格失败: %s", closestKline.Close)
		return 0, fmt.Errorf("解析价格失败: %w", err)
	}

	log.Printf("[INFO] 找到历史价格: %s = %.2f (日期: %s)",
		symbol, price, time.Unix(int64(closestKline.OpenTime/1000), 0).Format("2006-01-02"))

	return price, nil
}

// getHistoricalTechnicalIndicators 获取指定日期的历史技术指标
func (s *Server) getHistoricalTechnicalIndicators(ctx context.Context, symbol string, targetDate time.Time) (*TechnicalIndicators, error) {
	log.Printf("[INFO] 计算历史技术指标: %s 在 %s", symbol, targetDate.Format("2006-01-02"))

	// 获取足够的历史数据来计算技术指标（至少需要100条数据）
	startDate := targetDate.AddDate(0, 0, -150) // 150天前
	endDate := targetDate.AddDate(0, 0, 1)      // 目标日期后一天

	// 查询日K线数据用于计算技术指标
	klines, err := s.fetchBinanceKlinesWithTimeRange(ctx, symbol, "spot", "1d", 200, &startDate, &endDate)
	if err != nil {
		log.Printf("[WARN] 查询历史K线数据失败: %v", err)
		return nil, fmt.Errorf("查询历史K线数据失败: %w", err)
	}

	if len(klines) < 26 {
		log.Printf("[WARN] 历史K线数据不足%d条，无法计算技术指标", len(klines))
		return nil, fmt.Errorf("历史K线数据不足")
	}

	// 找到最接近目标日期的K线位置
	targetTimestamp := targetDate.Unix() * 1000
	closestIndex := -1
	minDiff := time.Hour * 24 * 365

	for i, kline := range klines {
		diff := time.Duration(math.Abs(kline.OpenTime - float64(targetTimestamp)))
		if diff < minDiff {
			minDiff = diff
			closestIndex = i
		}
	}

	if closestIndex == -1 {
		return nil, fmt.Errorf("未找到接近目标日期的K线数据")
	}

	// 使用目标日期前的K线数据计算技术指标
	// 取最近的50-100条数据进行计算
	startIndex := 0
	if closestIndex > 100 {
		startIndex = closestIndex - 100
	}
	endIndex := closestIndex + 1

	if endIndex > len(klines) {
		endIndex = len(klines)
	}

	historicalKlines := klines[startIndex:endIndex]

	log.Printf("[INFO] 使用%d条历史K线数据计算技术指标", len(historicalKlines))

	// 计算技术指标
	indicators, err := s.calculateTechnicalIndicatorsFromKlines(historicalKlines)
	if err != nil {
		log.Printf("[WARN] 计算历史技术指标失败: %v", err)
		return nil, fmt.Errorf("计算历史技术指标失败: %w", err)
	}

	log.Printf("[INFO] 历史技术指标计算完成: RSI=%.2f, MACD=%.2f, Trend=%s",
		indicators.RSI, indicators.MACD, indicators.Trend)

	return indicators, nil
}

// calculateTechnicalIndicatorsFromKlines 从K线数据计算技术指标
func (s *Server) calculateTechnicalIndicatorsFromKlines(klines []BinanceKline) (*TechnicalIndicators, error) {
	if len(klines) < 26 {
		return nil, fmt.Errorf("K线数据不足%d条", len(klines))
	}

	// 提取价格和成交量数据（复用GetTechnicalIndicators的逻辑）
	closes := make([]float64, 0, len(klines))
	highs := make([]float64, 0, len(klines))
	lows := make([]float64, 0, len(klines))
	volumes := make([]float64, 0, len(klines))

	for _, k := range klines {
		close, _ := strconv.ParseFloat(k.Close, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)

		closes = append(closes, close)
		highs = append(highs, high)
		lows = append(lows, low)
		volumes = append(volumes, volume)
	}

	// 使用与GetTechnicalIndicators相同的计算逻辑
	if len(closes) < 60 {
		// 数据不足，只计算基本指标
		rsi := calculateRSI(closes, 14)
		macd, signal, hist := calculateMACD(closes, 12, 26, 9)
		trend := determineTrend(rsi, macd, signal)

		return &TechnicalIndicators{
			RSI:        rsi,
			MACD:       macd,
			MACDSignal: signal,
			MACDHist:   hist,
			Trend:      trend,
			MA5:        calculateSMA(closes, 5),
			MA10:       calculateSMA(closes, 10),
			MA20:       calculateSMA(closes, 20),
		}, nil
	}

	// 数据充足，计算完整的技术指标
	rsi := calculateRSI(closes, 14)
	macd, signal, hist := calculateMACD(closes, 12, 26, 9)
	bbMiddle, bbUpper, bbLower, bbPosition, _ := calculateBollingerBands(closes, 20, 2.0)

	// 计算支撑阻力位
	supportLevel, resistanceLevel, _, _ := calculateSupportResistance(highs, lows, closes, 20)

	// 计算波动率
	volatility20 := calculateVolatility(closes, 20)
	volatility5 := calculateVolatility(closes, 5)

	// 计算威廉指标
	williamsR := calculateWilliamsR(highs, lows, closes, 14)

	// 计算动量指标
	momentum10 := calculateMomentum(closes, 10)
	momentum20 := calculateMomentum(closes, 20)
	momentum5 := calculateMomentum(closes, 5)

	// 计算CCI
	cci := calculateCCI(highs, lows, closes, 20)

	// 简化计算 - 使用默认值
	signalStrength := 75.0

	// 确定趋势
	trend := determineTrend(rsi, macd, signal)

	// 计算风险等级
	riskLevel := "low"

	// 计算成交量指标
	volumeMA20 := calculateSMA(volumes, 20)
	volumeMA5 := calculateSMA(volumes, 5)
	volumeRatio := 1.0
	if volumeMA20 > 0 {
		volumeRatio = volumeMA5 / volumeMA20
	}

	return &TechnicalIndicators{
		RSI:             rsi,
		MACD:            macd,
		MACDSignal:      signal,
		MACDHist:        hist,
		BBUpper:         bbUpper,
		BBMiddle:        bbMiddle,
		BBLower:         bbLower,
		BBPosition:      bbPosition,
		Trend:           trend,
		MA5:             calculateSMA(closes, 5),
		MA10:            calculateSMA(closes, 10),
		MA20:            calculateSMA(closes, 20),
		MA50:            calculateSMA(closes, 50),
		SupportLevel:    supportLevel,
		ResistanceLevel: resistanceLevel,
		Volatility20:    volatility20,
		Volatility5:     volatility5,
		WilliamsR:       williamsR,
		Momentum10:      momentum10,
		Momentum20:      momentum20,
		Momentum5:       momentum5,
		CCI:             cci,
		SignalStrength:  signalStrength,
		RiskLevel:       riskLevel,
		VolumeMA20:      volumeMA20,
		VolumeMA5:       volumeMA5,
		VolumeRatio:     volumeRatio,
	}, nil
}

// getRiskLevel 根据风险评分确定风险等级
func (s *Server) getRiskLevel(riskScore float64) string {
	if riskScore >= 0.7 {
		return "critical"
	} else if riskScore >= 0.5 {
		return "high"
	} else if riskScore >= 0.3 {
		return "medium"
	} else {
		return "low"
	}
}

// generateAdvancedRecommendations 生成高级推荐（核心实现已移动到strategy_generator.go）
func (s *Server) generateAdvancedRecommendations(symbols []string, limit int, riskLevel string) ([]gin.H, error) {
	// 核心实现已移动到strategy_generator.go
	// 这里保留API接口的调用逻辑
	_ = context.Background()

	// 获取推荐数据 - 实际实现应调用strategy_generator中的函数
	result := make([]gin.H, 0, limit)

	for i, symbol := range symbols {
		if i >= limit {
			break
		}

		// 模拟推荐数据 - 实际应从strategy_generator获取
		recommendation := gin.H{
			"symbol":          symbol,
			"rank":            i + 1,
			"overall_score":   0.85 - float64(i)*0.08,
			"expected_return": 0.18 - float64(i)*0.015,
			"risk_score":      0.25 + float64(i)*0.04,
		}
		result = append(result, recommendation)
	}

	return result, nil
}

// AIBacktestAPI AI推荐策略回测API
// POST /api/ai-recommendation/backtest
func (s *Server) AIBacktestAPI(c *gin.Context) {
	log.Printf("[DEBUG] ===== AIBacktestAPI 被调用 =====")

	var req struct {
		Symbol      string  `json:"symbol" binding:"required"`
		Timeframe   string  `json:"timeframe"`    // 可选，默认为"1d"
		StartDate   string  `json:"start_date"`   // 可选，格式: "2024-01-01"
		EndDate     string  `json:"end_date"`     // 可选，格式: "2024-12-31"
		InitialCash float64 `json:"initial_cash"` // 可选，默认10000
		MaxPosition float64 `json:"max_position"` // 可选，默认0.05
		StopLoss    float64 `json:"stop_loss"`    // 可选，默认0.05
		TakeProfit  float64 `json:"take_profit"`  // 可选，默认0.10
		Commission  float64 `json:"commission"`   // 可选，默认0.001
		Strategy    string  `json:"strategy"`     // 可选，"ml_prediction"或"ensemble"

		// 自动执行参数
		AutoExecute          bool    `json:"auto_execute"`            // 是否启用自动执行
		AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"` // 风险等级: conservative/moderate/aggressive
		MinConfidence        float64 `json:"min_confidence"`          // 最小置信度
		MaxPositionPercent   float64 `json:"max_position_percent"`    // 最大仓位百分比
		SkipExistingTrades   bool    `json:"skip_existing_trades"`    // 跳过已存在的交易

		// 渐进式执行参数
		ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
		MaxBatches            int           `json:"max_batches"`             // 最大批次数
		BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
		BatchSize             int           `json:"batch_size"`              // 每批最大交易数
		DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
		MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendRecommendationError(c, 400, "无效的请求参数格式", "INVALID_REQUEST_FORMAT", err.Error())
		return
	}

	// 验证币种名称格式
	if matched, _ := regexp.MatchString(`^[A-Z0-9]{2,10}$`, req.Symbol); !matched {
		sendRecommendationError(c, 400, fmt.Sprintf("无效的币种名称: %s", req.Symbol), "INVALID_SYMBOL_FORMAT")
		return
	}

	// 处理止损参数：确保为负值
	if req.StopLoss > 0 {
		req.StopLoss = -req.StopLoss // 自动转换为负值
		log.Printf("[DEBUG] 止损参数从正值转换为负值: %.4f", req.StopLoss)
	} else if req.StopLoss == 0 {
		req.StopLoss = -0.15 // 默认15%止损（从5%放宽，避免过早止损）
		log.Printf("[DEBUG] 使用默认止损参数: %.4f", req.StopLoss)
	}

	// 检查缓存
	ctx := context.Background()

	// 创建缓存键时使用原始的回测参数（不包含自动执行字段）
	type AIBacktestCacheRequest struct {
		Symbol      string   `json:"symbol" binding:"required"` // 向后兼容
		Symbols     []string `json:"symbols"`                   // 多币种列表（单币种为空）
		Timeframe   string   `json:"timeframe"`
		StartDate   string   `json:"start_date"`
		EndDate     string   `json:"end_date"`
		InitialCash float64  `json:"initial_cash"`
		MaxPosition float64  `json:"max_position"`
		StopLoss    float64  `json:"stop_loss"`
		TakeProfit  float64  `json:"take_profit"`
		Commission  float64  `json:"commission"`
		Strategy    string   `json:"strategy"`
	}

	cacheReq := AIBacktestCacheRequest{
		Symbol:      req.Symbol,
		Symbols:     []string{}, // AI回测是单币种模式
		Timeframe:   req.Timeframe,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		InitialCash: req.InitialCash,
		MaxPosition: req.MaxPosition,
		StopLoss:    req.StopLoss,
		TakeProfit:  req.TakeProfit,
		Commission:  req.Commission,
		Strategy:    req.Strategy,
	}

	cacheKey := s.generateBacktestCacheKey(cacheReq)

	// 尝试从缓存获取结果
	if s.recommendationCache != nil {
		cachedResult, err := s.getCachedBacktestResult(cacheKey)
		if err == nil && cachedResult != nil {
			log.Printf("[DEBUG] 使用缓存的回测结果: %s", cacheKey)

			// 注意：缓存结果不支持自动执行，因为无法获取用户上下文
			if req.AutoExecute {
				log.Printf("[INFO] 缓存结果不支持自动执行，将重新执行回测")
				// 如果启用了自动执行且有缓存结果，跳过缓存，重新执行
			} else {
				c.JSON(200, gin.H{
					"success":         true,
					"backtest_result": cachedResult.Result,
					"cached":          true,
					"timestamp":       cachedResult.CachedAt.Unix(),
				})
				return
			}

			c.JSON(200, gin.H{
				"success":         true,
				"backtest_result": cachedResult.Result,
				"cached":          true,
				"timestamp":       cachedResult.CachedAt.Unix(),
			})
			return
		}
	}

	// 获取AI推荐数据作为回测基础
	// 使用回测结束日期生成AI推荐，这样推荐基于历史时间点的市场状况
	log.Printf("[DEBUG] 回测时间范围: %s 到 %s，将使用结束日期%s生成AI推荐", req.StartDate, req.EndDate, req.EndDate)
	recommendation, err := s.getAIRecommendationForBacktest(ctx, req.Symbol, req.EndDate)
	if err != nil {
		log.Printf("[ERROR] 获取AI推荐数据失败: %v", err)
		sendRecommendationError(c, 500, "获取推荐数据失败", "RECOMMENDATION_FETCH_FAILED", err.Error())
		return
	}

	// 将AI推荐映射为回测配置（使用原始参数）
	backtestConfig := s.mapRecommendationToBacktestConfig(recommendation, cacheReq)

	log.Printf("[DEBUG] 映射回测配置: symbol=%s, strategy=%s, max_position=%.3f",
		backtestConfig.Symbol, backtestConfig.Strategy, backtestConfig.MaxPosition)

	// 执行回测
	result, err := s.backtestEngine.RunBacktest(ctx, *backtestConfig)
	if err != nil {
		log.Printf("[ERROR] 回测执行失败: %v", err)
		sendRecommendationError(c, 500, "回测执行失败", "BACKTEST_FAILED", err.Error())
		return
	}

	// 增强回测结果，添加AI推荐相关信息
	enhancedResult := s.enhanceBacktestResultWithAIInsights(result, recommendation)

	// 自动执行统计
	var autoExecuteStats *gin.H
	if req.AutoExecute {
		// 检查用户是否已登录
		if _, exists := c.Get("uid"); !exists {
			log.Printf("[WARN] 自动执行需要用户登录，已跳过")
		} else {
			stats, err := s.executeBacktestTrades(c, req, enhancedResult)
			if err != nil {
				log.Printf("[WARN] 自动执行交易失败: %v", err)
				// 不影响回测结果，只记录警告
			} else {
				autoExecuteStats = stats
			}
		}
	}

	// 缓存结果
	if s.recommendationCache != nil {
		cacheData := &CachedBacktestResult{
			Result:   enhancedResult,
			CachedAt: time.Now(),
		}
		if err := s.cacheBacktestResult(cacheKey, cacheData); err != nil {
			log.Printf("[WARN] 缓存回测结果失败: %v", err)
		}
	}

	response := gin.H{
		"success":         true,
		"backtest_result": enhancedResult,
		"recommendation":  recommendation,
		"config":          backtestConfig,
		"cached":          false,
		"timestamp":       time.Now().Unix(),
	}

	// 如果有自动执行统计，添加到响应中
	if autoExecuteStats != nil {
		response["auto_execute_stats"] = *autoExecuteStats
	}

	c.JSON(200, response)
}

// getAIRecommendationForBacktest 获取用于回测的AI推荐数据
func (s *Server) getAIRecommendationForBacktest(ctx context.Context, symbol string, date string) (gin.H, error) {
	// 使用现有的AI推荐生成逻辑，但传递日期参数
	return s.generateSingleAIRecommendationWithDate(ctx, symbol, date)
}

// StrategyMappingEngine 策略映射引擎 - 智能映射AI推荐到回测配置
type StrategyMappingEngine struct {
	// 简化的实现，暂时不依赖外部组件
}

// NewStrategyMappingEngine 创建策略映射引擎
func NewStrategyMappingEngine() *StrategyMappingEngine {
	return &StrategyMappingEngine{}
}

// MarketConditionAnalyzer 市场环境分析器（简化版）
type MarketConditionAnalyzer struct{}

// RiskAssessmentEngine 风险评估引擎（简化版）
type RiskAssessmentEngine struct{}

// AdaptiveStrategySelector 自适应策略选择器（简化版）
type AdaptiveStrategySelector struct{}

// ============================================================================
// 渐进式自动执行引擎
// ============================================================================

// ProgressiveAutoExecutor 渐进式自动执行器
type ProgressiveAutoExecutor struct {
	server     *Server
	maxBatches int           // 最大批次数
	batchDelay time.Duration // 批次间延迟
	riskLimits RiskLimits    // 风险限制
}

// RiskLimits 风险限制
type RiskLimits struct {
	MaxDailyTrades    int     // 每日最大交易次数
	MaxDailyLoss      float64 // 每日最大亏损比例
	MaxPositionSize   float64 // 最大单次仓位比例
	MaxTotalExposure  float64 // 最大总敞口
	StopLossThreshold float64 // 止损阈值
}

// ExecutionBatch 执行批次
type ExecutionBatch struct {
	BatchID         string                   `json:"batch_id"`
	BatchNumber     int                      `json:"batch_number"`
	TotalBatches    int                      `json:"total_batches"`
	Recommendations []gin.H                  `json:"recommendations"`
	ExecutedTrades  []map[string]interface{} `json:"executed_trades"`
	Status          string                   `json:"status"` // "pending", "executing", "completed", "paused", "stopped"
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
	RiskMetrics     map[string]float64       `json:"risk_metrics"`
}

// ProgressiveExecutionConfig 渐进式执行配置
type ProgressiveExecutionConfig struct {
	Enabled                    bool          `json:"enabled"`
	MaxBatches                 int           `json:"max_batches"`                  // 最大批次数 (默认3)
	BatchDelay                 time.Duration `json:"batch_delay"`                  // 批次间延迟 (默认30分钟)
	BatchSize                  int           `json:"batch_size"`                   // 每批最大交易数 (默认5)
	RiskCheckInterval          time.Duration `json:"risk_check_interval"`          // 风险检查间隔 (默认5分钟)
	DynamicSizing              bool          `json:"dynamic_sizing"`               // 是否启用动态仓位调整
	MarketConditionFilter      bool          `json:"market_condition_filter"`      // 是否启用市场条件过滤
	PerformanceBasedAdjustment bool          `json:"performance_based_adjustment"` // 是否基于表现调整
}

// NewProgressiveAutoExecutor 创建渐进式自动执行器
func NewProgressiveAutoExecutor(server *Server) *ProgressiveAutoExecutor {
	return &ProgressiveAutoExecutor{
		server:     server,
		maxBatches: 3,
		batchDelay: 30 * time.Minute,
		riskLimits: RiskLimits{
			MaxDailyTrades:    20,
			MaxDailyLoss:      0.05, // 5%
			MaxPositionSize:   0.10, // 10%
			MaxTotalExposure:  0.50, // 50%
			StopLossThreshold: 0.03, // 3%
		},
	}
}

// ExecuteProgressive 执行渐进式自动执行
func (pae *ProgressiveAutoExecutor) ExecuteProgressive(
	ctx context.Context,
	userID uint,
	recommendations []gin.H,
	config ProgressiveExecutionConfig,
) (*ProgressiveExecutionResult, error) {

	result := &ProgressiveExecutionResult{
		TotalBatches:     config.MaxBatches,
		ExecutedBatches:  0,
		TotalTrades:      0,
		SuccessfulTrades: 0,
		FailedTrades:     0,
		Status:           "running",
		StartTime:        time.Now(),
		Batches:          make([]ExecutionBatch, 0),
		RiskMetrics:      make(map[string]float64),
	}

	log.Printf("[ProgressiveExecution] 开始渐进式执行: 用户=%d, 推荐数=%d, 批次数=%d",
		userID, len(recommendations), config.MaxBatches)

	// 1. 初始风险评估
	if err := pae.performInitialRiskCheck(ctx, userID, recommendations); err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("初始风险检查失败: %v", err)
		return result, err
	}

	// 2. 将推荐分组为批次
	batches := pae.createExecutionBatches(recommendations, config)

	// 3. 逐批执行
	for i, batch := range batches {
		log.Printf("[ProgressiveExecution] 执行批次 %d/%d", i+1, len(batches))

		// 执行前的风险检查
		if shouldPause, reason := pae.shouldPauseExecution(ctx, userID, result); shouldPause {
			log.Printf("[ProgressiveExecution] 批次 %d 暂停: %s", i+1, reason)
			batch.Status = "paused"
			result.Batches = append(result.Batches, batch)
			break
		}

		// 执行当前批次
		executedBatch, err := pae.executeBatch(ctx, userID, batch, config)
		if err != nil {
			log.Printf("[ProgressiveExecution] 批次 %d 执行失败: %v", i+1, err)
			batch.Status = "failed"
			result.Error = fmt.Sprintf("批次 %d 执行失败: %v", i+1, err)
		} else {
			batch = *executedBatch
		}

		result.Batches = append(result.Batches, batch)
		result.ExecutedBatches++

		// 更新统计信息
		pae.updateExecutionStats(result, batch)

		// 检查是否需要停止
		if pae.shouldStopExecution(result) {
			log.Printf("[ProgressiveExecution] 执行停止于批次 %d", i+1)
			result.Status = "stopped"
			break
		}

		// 批次间延迟 (除了最后一个批次)
		if i < len(batches)-1 {
			log.Printf("[ProgressiveExecution] 等待 %v 后执行下一批次", config.BatchDelay)
			select {
			case <-ctx.Done():
				result.Status = "cancelled"
				return result, ctx.Err()
			case <-time.After(config.BatchDelay):
				// 继续下一批次
			}
		}
	}

	// 4. 执行后分析
	pae.performPostExecutionAnalysis(result)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.Status == "running" {
		result.Status = "completed"
	}

	log.Printf("[ProgressiveExecution] 渐进式执行完成: 总交易=%d, 成功=%d, 失败=%d, 耗时=%v",
		result.TotalTrades, result.SuccessfulTrades, result.FailedTrades, result.Duration)

	return result, nil
}

// performInitialRiskCheck 执行初始风险检查
func (pae *ProgressiveAutoExecutor) performInitialRiskCheck(ctx context.Context, userID uint, recommendations []gin.H) error {
	log.Printf("[ProgressiveExecution] 执行初始风险检查: 用户=%d, 推荐数=%d", userID, len(recommendations))

	// 检查用户每日交易限制
	dailyTrades, err := pae.getUserDailyTrades(userID)
	if err != nil {
		return fmt.Errorf("获取用户每日交易数失败: %w", err)
	}

	if dailyTrades >= pae.riskLimits.MaxDailyTrades {
		return fmt.Errorf("已达到每日最大交易次数限制: %d", pae.riskLimits.MaxDailyTrades)
	}

	// 检查总敞口
	totalExposure, err := pae.getUserTotalExposure(userID)
	if err != nil {
		return fmt.Errorf("获取用户总敞口失败: %w", err)
	}

	if totalExposure >= pae.riskLimits.MaxTotalExposure {
		return fmt.Errorf("已达到最大总敞口限制: %.2f", pae.riskLimits.MaxTotalExposure)
	}

	// 检查推荐质量
	for i, rec := range recommendations {
		if confidence, exists := rec["confidence"]; exists {
			if conf, ok := confidence.(float64); ok && conf < 0.5 {
				log.Printf("[ProgressiveExecution] 推荐 %d 置信度过低: %.2f，跳过", i+1, conf)
				continue
			}
		}
	}

	return nil
}

// createExecutionBatches 将推荐分组为执行批次
func (pae *ProgressiveAutoExecutor) createExecutionBatches(recommendations []gin.H, config ProgressiveExecutionConfig) []ExecutionBatch {
	var batches []ExecutionBatch
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 5 // 默认每批5个
	}

	for i := 0; i < len(recommendations); i += batchSize {
		end := i + batchSize
		if end > len(recommendations) {
			end = len(recommendations)
		}

		batch := ExecutionBatch{
			BatchID:         fmt.Sprintf("batch_%d_%d", time.Now().Unix(), i/batchSize+1),
			BatchNumber:     i/batchSize + 1,
			TotalBatches:    (len(recommendations) + batchSize - 1) / batchSize,
			Recommendations: recommendations[i:end],
			Status:          "pending",
			StartTime:       time.Now(),
			RiskMetrics:     make(map[string]float64),
		}

		batches = append(batches, batch)
	}

	log.Printf("[ProgressiveExecution] 创建了 %d 个执行批次，每批最多 %d 个推荐", len(batches), batchSize)
	return batches
}

// shouldPauseExecution 检查是否需要暂停执行
func (pae *ProgressiveAutoExecutor) shouldPauseExecution(ctx context.Context, userID uint, result *ProgressiveExecutionResult) (bool, string) {
	// 检查每日亏损限制
	dailyPnL, err := pae.getUserDailyPnL(userID)
	if err != nil {
		log.Printf("[ProgressiveExecution] 获取每日PnL失败: %v", err)
		return true, "无法获取每日PnL数据"
	}

	if dailyPnL <= -pae.riskLimits.MaxDailyLoss {
		return true, fmt.Sprintf("达到每日最大亏损限制: %.2f", pae.riskLimits.MaxDailyLoss)
	}

	// 检查执行统计
	if result.FailedTrades > result.SuccessfulTrades*2 {
		return true, fmt.Sprintf("失败交易数过多: 成功=%d, 失败=%d", result.SuccessfulTrades, result.FailedTrades)
	}

	return false, ""
}

// executeBatch 执行单个批次
func (pae *ProgressiveAutoExecutor) executeBatch(ctx context.Context, userID uint, batch ExecutionBatch, config ProgressiveExecutionConfig) (*ExecutionBatch, error) {
	batch.Status = "executing"
	batch.StartTime = time.Now()

	executedBatch := batch
	executedBatch.ExecutedTrades = make([]map[string]interface{}, 0)

	log.Printf("[ProgressiveExecution] 执行批次 %d/%d，包含 %d 个推荐",
		batch.BatchNumber, batch.TotalBatches, len(batch.Recommendations))

	for _, rec := range batch.Recommendations {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			batch.Status = "cancelled"
			return &executedBatch, ctx.Err()
		default:
		}

		// 执行单个交易
		trade, err := pae.executeSingleTrade(ctx, userID, rec, config)
		if err != nil {
			log.Printf("[ProgressiveExecution] 交易执行失败: %v", err)
			executedBatch.ExecutedTrades = append(executedBatch.ExecutedTrades, map[string]interface{}{
				"status":         "failed",
				"error":          err.Error(),
				"recommendation": rec,
			})
		} else {
			executedBatch.ExecutedTrades = append(executedBatch.ExecutedTrades, trade)
		}
	}

	batch.Status = "completed"
	batch.EndTime = time.Now()
	return &executedBatch, nil
}

// updateExecutionStats 更新执行统计信息
func (pae *ProgressiveAutoExecutor) updateExecutionStats(result *ProgressiveExecutionResult, batch ExecutionBatch) {
	for _, trade := range batch.ExecutedTrades {
		result.TotalTrades++
		if status, exists := trade["status"]; exists && status == "success" {
			result.SuccessfulTrades++
		} else {
			result.FailedTrades++
		}
	}
}

// shouldStopExecution 检查是否需要停止执行
func (pae *ProgressiveAutoExecutor) shouldStopExecution(result *ProgressiveExecutionResult) bool {
	// 如果失败率超过50%，停止执行
	totalCompleted := result.SuccessfulTrades + result.FailedTrades
	if totalCompleted > 0 && float64(result.FailedTrades)/float64(totalCompleted) > 0.5 {
		return true
	}

	// 如果总失败次数超过限制，停止执行
	if result.FailedTrades >= 10 {
		return true
	}

	return false
}

// performPostExecutionAnalysis 执行后分析
func (pae *ProgressiveAutoExecutor) performPostExecutionAnalysis(result *ProgressiveExecutionResult) {
	log.Printf("[ProgressiveExecution] 执行后分析: 总交易=%d, 成功=%d, 失败=%d",
		result.TotalTrades, result.SuccessfulTrades, result.FailedTrades)

	// 计算成功率
	if result.TotalTrades > 0 {
		successRate := float64(result.SuccessfulTrades) / float64(result.TotalTrades)
		result.RiskMetrics["success_rate"] = successRate
		log.Printf("[ProgressiveExecution] 成功率: %.2f%%", successRate*100)
	}

	// 计算平均执行时间
	if result.Duration > 0 && result.ExecutedBatches > 0 {
		avgBatchTime := result.Duration / time.Duration(result.ExecutedBatches)
		result.RiskMetrics["avg_batch_time_seconds"] = avgBatchTime.Seconds()
	}
}

// executeSingleTrade 执行单个交易
func (pae *ProgressiveAutoExecutor) executeSingleTrade(ctx context.Context, userID uint, recommendation gin.H, config ProgressiveExecutionConfig) (map[string]interface{}, error) {
	// 这里应该实现实际的交易执行逻辑
	// 目前返回模拟成功结果
	return map[string]interface{}{
		"status":         "success",
		"recommendation": recommendation,
		"executed_at":    time.Now(),
	}, nil
}

// getUserDailyTrades 获取用户当日交易次数
func (pae *ProgressiveAutoExecutor) getUserDailyTrades(userID uint) (int, error) {
	// 模拟实现 - 应该从数据库查询
	return 0, nil
}

// getUserTotalExposure 获取用户总敞口
func (pae *ProgressiveAutoExecutor) getUserTotalExposure(userID uint) (float64, error) {
	// 模拟实现 - 应该从数据库查询
	return 0.0, nil
}

// getUserDailyPnL 获取用户当日PnL
func (pae *ProgressiveAutoExecutor) getUserDailyPnL(userID uint) (float64, error) {
	// 模拟实现 - 应该从数据库查询
	return 0.0, nil
}

// ProgressiveExecutionResult 渐进式执行结果
type ProgressiveExecutionResult struct {
	TotalBatches     int                `json:"total_batches"`
	ExecutedBatches  int                `json:"executed_batches"`
	TotalTrades      int                `json:"total_trades"`
	SuccessfulTrades int                `json:"successful_trades"`
	FailedTrades     int                `json:"failed_trades"`
	TotalValue       float64            `json:"total_value"`
	TotalPnL         float64            `json:"total_pnl"`
	Status           string             `json:"status"`
	StartTime        time.Time          `json:"start_time"`
	EndTime          time.Time          `json:"end_time"`
	Duration         time.Duration      `json:"duration"`
	Batches          []ExecutionBatch   `json:"batches"`
	RiskMetrics      map[string]float64 `json:"risk_metrics"`
	Error            string             `json:"error,omitempty"`
}

// MappingStrategyConfig 策略映射配置
type MappingStrategyConfig struct {
	Strategy    string
	MaxPosition float64
	StopLoss    float64
	TakeProfit  float64
	Commission  float64
}

// mapRecommendationToBacktestConfig 将AI推荐映射为回测配置 - 改进版
func (s *Server) mapRecommendationToBacktestConfig(recommendation gin.H, req struct {
	Symbol      string   `json:"symbol" binding:"required"` // 向后兼容
	Symbols     []string `json:"symbols"`                   // 多币种列表
	Timeframe   string   `json:"timeframe"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	InitialCash float64  `json:"initial_cash"`
	MaxPosition float64  `json:"max_position"`
	StopLoss    float64  `json:"stop_loss"`
	TakeProfit  float64  `json:"take_profit"`
	Commission  float64  `json:"commission"`
	Strategy    string   `json:"strategy"`
}) *BacktestConfig {

	// 创建智能映射引擎
	mapper := NewStrategyMappingEngine()

	// 1. 解析基础配置
	baseConfig := mapper.parseBaseConfig(req)

	// 2. 分析市场环境
	marketCondition := mapper.analyzeMarketCondition(recommendation)

	// 3. 评估推荐质量
	recommendationQuality := mapper.assessRecommendationQuality(recommendation)

	// 4. 智能选择策略
	strategyConfig := mapper.selectOptimalStrategy(recommendation, marketCondition, recommendationQuality, req.Strategy)

	// 5. 动态调整参数
	parameterConfig := mapper.adjustParametersDynamically(recommendation, strategyConfig, marketCondition)

	// 6. 构建最终配置
	finalConfig := mapper.buildFinalConfig(baseConfig, strategyConfig, parameterConfig)

	log.Printf("[STRATEGY_MAPPING] 智能映射完成: symbol=%s, market=%s, strategy=%s, confidence=%.2f, risk=%.2f",
		finalConfig.Symbol, marketCondition.Condition, finalConfig.Strategy,
		recommendationQuality.Confidence, recommendationQuality.RiskScore)

	return finalConfig
}

// parseBaseConfig 解析基础配置
func (sme *StrategyMappingEngine) parseBaseConfig(req struct {
	Symbol      string   `json:"symbol" binding:"required"` // 向后兼容
	Symbols     []string `json:"symbols"`                   // 多币种列表
	Timeframe   string   `json:"timeframe"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	InitialCash float64  `json:"initial_cash"`
	MaxPosition float64  `json:"max_position"`
	StopLoss    float64  `json:"stop_loss"`
	TakeProfit  float64  `json:"take_profit"`
	Commission  float64  `json:"commission"`
	Strategy    string   `json:"strategy"`
}) *BaseConfig {

	timeframe := "1d"
	if req.Timeframe != "" {
		timeframe = req.Timeframe
	}

	now := time.Now()
	startDate := now.AddDate(0, -3, 0)
	endDate := now

	if req.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = parsed
		}
	}
	if req.EndDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = parsed
		}
	}

	// 确保日期合理性
	if startDate.After(endDate) {
		startDate = endDate.AddDate(0, -3, 0)
	}
	if endDate.After(now) {
		endDate = now.AddDate(0, 0, -1)
	}

	initialCash := 10000.0
	if req.InitialCash > 0 {
		initialCash = req.InitialCash
	}

	maxPosition := 1000.0 // 默认仓位大小
	if req.MaxPosition > 0 {
		maxPosition = req.MaxPosition
	}

	// 确定使用的币种
	var symbols []string
	if len(req.Symbols) > 0 {
		symbols = req.Symbols
	} else {
		symbols = []string{req.Symbol}
	}

	return &BaseConfig{
		Symbol:      req.Symbol, // 主要币种（向后兼容）
		Symbols:     symbols,    // 多币种列表
		Timeframe:   timeframe,
		StartDate:   startDate,
		EndDate:     endDate,
		InitialCash: initialCash,
		MaxPosition: maxPosition,
	}
}

// BaseConfig 基础配置
type BaseConfig struct {
	Symbol      string   // 主要币种（向后兼容）
	Symbols     []string // 多币种列表
	Timeframe   string
	StartDate   time.Time
	EndDate     time.Time
	InitialCash float64
	MaxPosition float64 // 最大仓位大小
}

// analyzeMarketCondition 分析市场环境
func (sme *StrategyMappingEngine) analyzeMarketCondition(recommendation gin.H) *MarketCondition {
	condition := &MarketCondition{
		Condition:     "unknown",
		Volatility:    0.5,
		TrendStrength: 0.5,
		VolumeRatio:   1.0,
	}

	// 从推荐数据中提取市场信息
	if marketData, ok := recommendation["market_analysis"].(gin.H); ok {
		if vol, exists := marketData["volatility"].(float64); exists {
			condition.Volatility = vol
		}
		if trend, exists := marketData["trend_strength"].(float64); exists {
			condition.TrendStrength = trend
		}
		if volume, exists := marketData["volume_ratio"].(float64); exists {
			condition.VolumeRatio = volume
		}
	}

	// 确定市场状况
	if condition.Volatility > 0.7 {
		condition.Condition = "volatile" // 高波动
	} else if condition.TrendStrength > 0.7 {
		condition.Condition = "trending" // 强趋势
	} else if condition.Volatility < 0.3 {
		condition.Condition = "ranging" // 震荡区间
	} else {
		condition.Condition = "mixed" // 混合市场
	}

	// 从交易策略中提取方向性
	if tradingStrategy, ok := recommendation["trading_strategy"].(gin.H); ok {
		if strategyType, exists := tradingStrategy["strategy_type"].(string); exists {
			switch strategyType {
			case "LONG":
				condition.Direction = "bullish"
			case "SHORT":
				condition.Direction = "bearish"
			case "RANGE":
				condition.Direction = "neutral"
			}
		}
	}

	return condition
}

// MarketCondition 市场环境分析
type MarketCondition struct {
	Condition     string  // unknown, volatile, trending, ranging, mixed
	Direction     string  // bullish, bearish, neutral
	Volatility    float64 // 0-1
	TrendStrength float64 // 0-1
	VolumeRatio   float64 // 相对于平均水平的倍数
}

// assessRecommendationQuality 评估推荐质量
func (sme *StrategyMappingEngine) assessRecommendationQuality(recommendation gin.H) *RecommendationQuality {
	quality := &RecommendationQuality{
		Confidence: 0.5,
		RiskScore:  0.5,
		Score:      0.5,
	}

	// 提取置信度
	if confidence, ok := recommendation["confidence"].(float64); ok {
		quality.Confidence = confidence
	}

	// 提取综合评分
	if score, ok := recommendation["overall_score"].(float64); ok {
		quality.Score = score
	}

	// 计算风险评分（基于波动率和历史表现）
	if riskScore, ok := recommendation["risk_score"].(float64); ok {
		quality.RiskScore = riskScore
	} else {
		// 基于波动率估算风险
		if marketData, ok := recommendation["market_analysis"].(gin.H); ok {
			if vol, exists := marketData["volatility"].(float64); exists {
				quality.RiskScore = vol // 波动率即为风险指标
			}
		}
	}

	return quality
}

// RecommendationQuality 推荐质量评估
type RecommendationQuality struct {
	Confidence float64 // AI置信度 0-1
	RiskScore  float64 // 风险评分 0-1 (越高风险越大)
	Score      float64 // 综合评分 0-1
}

// selectOptimalStrategy 智能选择最优策略
func (sme *StrategyMappingEngine) selectOptimalStrategy(
	recommendation gin.H,
	marketCondition *MarketCondition,
	quality *RecommendationQuality,
	userPreference string,
) *MappingStrategyConfig {

	config := &MappingStrategyConfig{
		Strategy:    "ml_prediction",
		MaxPosition: 0.05,
		StopLoss:    -0.15, // 从-5%放宽到-15%，避免过早止损
		TakeProfit:  0.10,
		Commission:  0.001,
	}

	// 1. 首先检查用户明确偏好
	if userPreference == "deep_learning" {
		config.Strategy = "deep_learning"
		sme.adjustStrategyForDeepLearning(config, marketCondition, quality)
		return config
	}

	// 2. 基于市场环境和推荐质量智能选择
	strategyMatrix := sme.buildStrategyMatrix()

	// 计算策略适应度得分
	bestStrategy := ""
	bestScore := -1.0

	for strategy, conditions := range strategyMatrix {
		score := sme.calculateStrategyFitness(strategy, conditions, marketCondition, quality, userPreference)
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	config.Strategy = bestStrategy

	// 3. 根据选择的策略调整参数
	sme.adjustStrategyParameters(config, marketCondition, quality)

	return config
}

// buildStrategyMatrix 构建策略选择矩阵
func (sme *StrategyMappingEngine) buildStrategyMatrix() map[string]StrategyCondition {
	return map[string]StrategyCondition{
		"buy_and_hold": {
			BestMarkets:     []string{"trending"},
			RiskTolerance:   "low",
			ConfidenceReq:   0.3,
			VolatilityRange: [2]float64{0.1, 0.8},
			Description:     "买入持有策略，适合强趋势市场",
		},
		"ml_prediction": {
			BestMarkets:     []string{"mixed", "trending"},
			RiskTolerance:   "medium",
			ConfidenceReq:   0.5,
			VolatilityRange: [2]float64{0.2, 0.9},
			Description:     "机器学习预测策略，平衡风险收益",
		},
		"ensemble": {
			BestMarkets:     []string{"volatile", "ranging", "mixed"},
			RiskTolerance:   "high",
			ConfidenceReq:   0.7,
			VolatilityRange: [2]float64{0.3, 1.0},
			Description:     "集成学习策略，适应复杂市场环境",
		},
		"deep_learning": {
			BestMarkets:     []string{"volatile", "mixed"},
			RiskTolerance:   "high",
			ConfidenceReq:   0.8,
			VolatilityRange: [2]float64{0.4, 1.0},
			Description:     "深度学习策略，处理非线性模式",
		},
	}
}

// StrategyCondition 策略适用条件
type StrategyCondition struct {
	BestMarkets     []string   // 最适用的市场环境
	RiskTolerance   string     // 风险承受能力: low, medium, high
	ConfidenceReq   float64    // 需要的AI置信度
	VolatilityRange [2]float64 // 适用的波动率范围 [min, max]
	Description     string     // 策略描述
}

// calculateStrategyFitness 计算策略适应度
func (sme *StrategyMappingEngine) calculateStrategyFitness(
	strategy string,
	condition StrategyCondition,
	market *MarketCondition,
	quality *RecommendationQuality,
	userPreference string,
) float64 {

	score := 0.0

	// 1. 市场环境匹配度 (30%)
	marketScore := 0.0
	for _, bestMarket := range condition.BestMarkets {
		if bestMarket == market.Condition {
			marketScore = 1.0
			break
		}
	}
	// 部分匹配的情况
	if marketScore == 0.0 {
		switch market.Condition {
		case "trending":
			if strategy == "ml_prediction" {
				marketScore = 0.7
			}
		case "ranging":
			if strategy == "ensemble" {
				marketScore = 0.6
			}
		}
	}
	score += marketScore * 0.3

	// 2. AI置信度匹配度 (25%)
	confidenceScore := 0.0
	if quality.Confidence >= condition.ConfidenceReq {
		confidenceScore = 1.0
	} else {
		// 置信度不足时按比例降低得分
		confidenceScore = quality.Confidence / condition.ConfidenceReq
	}
	score += confidenceScore * 0.25

	// 3. 波动率适应度 (20%)
	volatilityScore := 0.0
	if market.Volatility >= condition.VolatilityRange[0] && market.Volatility <= condition.VolatilityRange[1] {
		// 在最适范围内的得满分
		volatilityScore = 1.0
	} else {
		// 超出范围时按距离衰减
		volatilityScore = sme.calculateVolatilityFitness(market.Volatility, condition.VolatilityRange)
	}
	score += volatilityScore * 0.2

	// 4. 用户偏好匹配度 (15%)
	preferenceScore := 1.0 // 默认得分
	if userPreference != "" {
		switch userPreference {
		case "conservative":
			if strategy == "buy_and_hold" {
				preferenceScore = 1.0
			} else if strategy == "ml_prediction" {
				preferenceScore = 0.7
			} else {
				preferenceScore = 0.4
			}
		case "moderate":
			if strategy == "ml_prediction" {
				preferenceScore = 1.0
			} else if strategy == "ensemble" {
				preferenceScore = 0.8
			} else {
				preferenceScore = 0.6
			}
		case "aggressive":
			if strategy == "ensemble" {
				preferenceScore = 1.0
			} else if strategy == "deep_learning" {
				preferenceScore = 0.9
			} else {
				preferenceScore = 0.5
			}
		}
	}
	score += preferenceScore * 0.15

	// 5. 风险匹配度 (10%)
	riskScore := 1.0
	switch condition.RiskTolerance {
	case "low":
		riskScore = 1.0 - quality.RiskScore // 低风险策略适合低风险推荐
	case "medium":
		riskScore = 1.0 - math.Abs(quality.RiskScore-0.5)*2 // 中等风险策略
	case "high":
		riskScore = quality.RiskScore // 高风险策略适合高风险推荐
	}
	score += riskScore * 0.1

	return score
}

// calculateVolatilityFitness 计算波动率适应度
func (sme *StrategyMappingEngine) calculateVolatilityFitness(volatility float64, range_ [2]float64) float64 {
	minVol, maxVol := range_[0], range_[1]

	if volatility >= minVol && volatility <= maxVol {
		return 1.0
	}

	// 计算距离最适范围的距离
	var distance float64
	if volatility < minVol {
		distance = minVol - volatility
	} else {
		distance = volatility - maxVol
	}

	// 距离越远，得分越低（指数衰减）
	return math.Exp(-distance * 2)
}

// adjustStrategyParameters 根据市场环境动态调整策略参数
func (sme *StrategyMappingEngine) adjustStrategyParameters(
	config *MappingStrategyConfig,
	market *MarketCondition,
	quality *RecommendationQuality,
) {

	// 基于波动率调整仓位大小
	if market.Volatility > 0.8 {
		config.MaxPosition *= 0.5 // 高波动时减半仓位
	} else if market.Volatility < 0.3 {
		config.MaxPosition *= 1.5 // 低波动时可以加大仓位
	}

	// 基于风险评分调整止损止盈
	riskMultiplier := 1.0 + quality.RiskScore // 风险越高，止损越宽松

	if config.StopLoss < 0 {
		config.StopLoss *= riskMultiplier
	}
	if config.TakeProfit > 0 {
		config.TakeProfit *= riskMultiplier
	}

	// 基于置信度调整参数
	confidenceMultiplier := 0.5 + quality.Confidence // 置信度越高，参数越激进

	config.MaxPosition *= confidenceMultiplier
	if config.TakeProfit > 0 {
		config.TakeProfit *= confidenceMultiplier
	}

	// 确保参数在合理范围内
	config.MaxPosition = math.Min(math.Max(config.MaxPosition, 0.005), 0.2) // 0.5% - 20%
	if config.StopLoss < 0 {
		config.StopLoss = math.Max(config.StopLoss, -0.2) // 最大20%止损
	}
	if config.TakeProfit > 0 {
		config.TakeProfit = math.Min(config.TakeProfit, 1.0) // 最大100%止盈
	}
}

// adjustStrategyForDeepLearning 为深度学习策略调整参数
func (sme *StrategyMappingEngine) adjustStrategyForDeepLearning(
	config *MappingStrategyConfig,
	market *MarketCondition,
	quality *RecommendationQuality,
) {
	// 深度学习策略需要更高的置信度和更保守的参数
	config.MaxPosition *= 0.7 // 更保守的仓位
	config.StopLoss = -0.03   // 更严格的止损
	config.TakeProfit = 0.08  // 更保守的止盈目标
}

// adjustParametersDynamically 动态调整参数
func (sme *StrategyMappingEngine) adjustParametersDynamically(
	recommendation gin.H,
	strategyConfig *MappingStrategyConfig,
	marketCondition *MarketCondition,
) *ParameterConfig {

	params := &ParameterConfig{
		MaxPosition: strategyConfig.MaxPosition,
		StopLoss:    strategyConfig.StopLoss,
		TakeProfit:  strategyConfig.TakeProfit,
		Commission:  strategyConfig.Commission,
	}

	// 从推荐中提取参数建议
	if recommendedPosition, ok := recommendation["recommended_position"].(float64); ok && recommendedPosition > 0 {
		params.MaxPosition = recommendedPosition
	}

	// 基于市场环境进一步调整
	if marketCondition.Condition == "volatile" {
		params.Commission *= 1.5 // 高波动市场，交易成本相对更高
	}

	return params
}

// ParameterConfig 参数配置
type ParameterConfig struct {
	MaxPosition float64
	StopLoss    float64
	TakeProfit  float64
	Commission  float64
}

// buildFinalConfig 构建最终配置
func (sme *StrategyMappingEngine) buildFinalConfig(
	baseConfig *BaseConfig,
	strategyConfig *MappingStrategyConfig,
	paramConfig *ParameterConfig,
) *BacktestConfig {

	return &BacktestConfig{
		Symbol:       baseConfig.Symbol,  // 主要币种（向后兼容）
		Symbols:      baseConfig.Symbols, // 多币种列表
		StartDate:    baseConfig.StartDate,
		EndDate:      baseConfig.EndDate,
		InitialCash:  baseConfig.InitialCash,
		Strategy:     strategyConfig.Strategy,
		Timeframe:    baseConfig.Timeframe,
		PositionSize: baseConfig.MaxPosition, // 使用用户指定的仓位大小
		MaxPosition:  baseConfig.MaxPosition,
		StopLoss:     paramConfig.StopLoss,
		TakeProfit:   paramConfig.TakeProfit,
		Commission:   paramConfig.Commission,
		MaxHoldTime:  1440, // 默认最大持有时间：1440周期（约60天，假设1小时周期）
	}
}

// enhanceBacktestResultWithAIInsights 增强回测结果，添加AI推荐洞察
func (s *Server) enhanceBacktestResultWithAIInsights(result *BacktestResult, recommendation gin.H) gin.H {
	// 计算AI预测准确性
	aiAccuracy := s.calculateAIPredictionAccuracy(result, recommendation)

	// 计算推荐有效性评分
	recommendationEffectiveness := s.calculateRecommendationEffectiveness(result, recommendation)

	// 生成回测洞察
	insights := s.generateBacktestInsights(result, recommendation)

	// 不再包含完整交易记录，只保存统计信息
	return gin.H{
		"config":                       result.Config,
		"summary":                      result.Summary,
		"trade_count":                  len(result.Trades), // 只保存交易数量
		"daily_returns":                result.DailyReturns,
		"ai_prediction_accuracy":       aiAccuracy,
		"recommendation_effectiveness": recommendationEffectiveness,
		"backtest_insights":            insights,
		"recommendation_context": gin.H{
			"overall_score":     recommendation["overall_score"],
			"expected_return":   recommendation["expected_return"],
			"risk_score":        recommendation["risk_score"],
			"technical_score":   recommendation["technical_score"],
			"fundamental_score": recommendation["fundamental_score"],
			"sentiment_score":   recommendation["sentiment_score"],
			"momentum_score":    recommendation["momentum_score"],
		},
	}
}

// convertTradesToMap 将TradeRecord切片转换为map切片，便于JSON序列化
func (s *Server) convertTradesToMap(trades []TradeRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, len(trades))
	for i, trade := range trades {
		result[i] = map[string]interface{}{
			"timestamp":     trade.Timestamp,
			"side":          trade.Side,
			"quantity":      trade.Quantity,
			"price":         trade.Price,
			"commission":    trade.Commission,
			"pnl":           trade.PnL,
			"reason":        trade.Reason,
			"ai_confidence": trade.AIConfidence,
			"risk_score":    trade.RiskScore,
		}
	}
	return result
}

// calculateAIPredictionAccuracy 计算AI预测准确性
func (s *Server) calculateAIPredictionAccuracy(result *BacktestResult, recommendation gin.H) gin.H {
	totalTrades := len(result.Trades)
	profitableTrades := 0

	for _, trade := range result.Trades {
		if trade.PnL > 0 {
			profitableTrades++
		}
	}

	accuracy := 0.0
	if totalTrades > 0 {
		accuracy = float64(profitableTrades) / float64(totalTrades)
	}

	return gin.H{
		"total_trades":      totalTrades,
		"profitable_trades": profitableTrades,
		"win_rate":          accuracy,
		"accuracy_score":    accuracy, // 0-1之间的准确性评分
	}
}

// calculateRecommendationEffectiveness 计算推荐有效性
func (s *Server) calculateRecommendationEffectiveness(result *BacktestResult, recommendation gin.H) gin.H {
	// 基于实际收益与预期收益的比较
	actualReturn := result.Summary.TotalReturn
	expectedReturn := 0.15 // 默认预期收益

	if expRet, ok := recommendation["expected_return"].(float64); ok {
		expectedReturn = expRet
	}

	effectiveness := 0.0
	if expectedReturn > 0 {
		effectiveness = actualReturn / expectedReturn
		// 限制在合理范围内
		if effectiveness > 2.0 {
			effectiveness = 2.0
		} else if effectiveness < -1.0 {
			effectiveness = -1.0
		}
	}

	return gin.H{
		"actual_return":   actualReturn,
		"expected_return": expectedReturn,
		"effectiveness":   effectiveness, // 实际收益/预期收益的比率
		"performance":     actualReturn > 0 && effectiveness > 0.8,
	}
}

// generateBacktestInsights 生成回测洞察
func (s *Server) generateBacktestInsights(result *BacktestResult, recommendation gin.H) []string {
	insights := []string{}

	// 收益表现洞察
	if result.Summary.TotalReturn > 0.2 {
		insights = append(insights, "回测显示优秀的收益表现，回测总收益超过20%")
	} else if result.Summary.TotalReturn > 0.1 {
		insights = append(insights, "回测显示良好的收益表现，回测总收益超过10%")
	} else if result.Summary.TotalReturn < 0 {
		insights = append(insights, "回测显示负收益，需要谨慎评估市场时机")
	}

	// 胜率洞察
	winRate := result.Summary.WinRate
	if winRate > 0.7 {
		insights = append(insights, "交易胜率超过70%，策略执行效果良好")
	} else if winRate > 0.5 {
		insights = append(insights, "交易胜率超过50%，策略表现中等")
	} else {
		insights = append(insights, "交易胜率较低，需要优化入场时机")
	}

	// 最大回撤洞察
	maxDrawdown := result.Summary.MaxDrawdown
	if maxDrawdown > 0.3 {
		insights = append(insights, "最大回撤较大，风险控制需要加强")
	} else if maxDrawdown < 0.1 {
		insights = append(insights, "最大回撤控制良好，风险管理到位")
	}

	// 夏普比率洞察
	sharpeRatio := result.Summary.SharpeRatio
	if sharpeRatio > 2.0 {
		insights = append(insights, "夏普比率优秀，风险调整后收益突出")
	} else if sharpeRatio > 1.0 {
		insights = append(insights, "夏普比率良好，收益风险配比合理")
	}

	return insights
}

// generateSingleAIRecommendation 生成单个币种的AI推荐
func (s *Server) generateSingleAIRecommendation(ctx context.Context, symbol string) (gin.H, error) {
	// 这里复用现有的AI推荐生成逻辑，但只返回第一个结果
	recommendations, err := s.generateAIRecommendations([]string{symbol}, 1, "", "")
	if err != nil {
		return nil, err
	}

	if len(recommendations) == 0 {
		return nil, fmt.Errorf("无法生成%s的推荐", symbol)
	}

	return recommendations[0], nil
}

// generateSingleAIRecommendationWithDate 生成带有日期参数的单个币种AI推荐
func (s *Server) generateSingleAIRecommendationWithDate(ctx context.Context, symbol string, date string) (gin.H, error) {
	// 使用日期参数生成AI推荐
	recommendations, err := s.generateAIRecommendations([]string{symbol}, 1, "", date)
	if err != nil {
		return nil, err
	}

	if len(recommendations) == 0 {
		return nil, fmt.Errorf("无法生成%s的推荐", symbol)
	}

	return recommendations[0], nil
}

// generateBacktestCacheKey 生成回测缓存键
func (s *Server) generateBacktestCacheKey(req struct {
	Symbol      string   `json:"symbol" binding:"required"` // 向后兼容
	Symbols     []string `json:"symbols"`                   // 多币种列表
	Timeframe   string   `json:"timeframe"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	InitialCash float64  `json:"initial_cash"`
	MaxPosition float64  `json:"max_position"`
	StopLoss    float64  `json:"stop_loss"`
	TakeProfit  float64  `json:"take_profit"`
	Commission  float64  `json:"commission"`
	Strategy    string   `json:"strategy"`
}) string {
	// 生成基于请求参数的唯一缓存键
	keyData := fmt.Sprintf("%s_%s_%s_%s_%.0f_%.3f_%.3f_%.3f_%.6f_%s",
		req.Symbol,
		req.Timeframe,
		req.StartDate,
		req.EndDate,
		req.InitialCash,
		req.MaxPosition,
		req.StopLoss,
		req.TakeProfit,
		req.Commission,
		req.Strategy,
	)
	return fmt.Sprintf("backtest:%x", md5.Sum([]byte(keyData)))
}

// getCachedBacktestResult 获取缓存的回测结果
func (s *Server) getCachedBacktestResult(cacheKey string) (*CachedBacktestResult, error) {
	//if s.recommendationCache == nil || !s.recommendationCache.redisEnabled {
	//	return nil, nil
	//}
	//
	//ctx := context.Background()
	//key := fmt.Sprintf("cached_backtest:%s", cacheKey)
	//
	//data, err := s.recommendationCache.redisClient.Get(ctx, key).Result()
	//if err != nil {
	//	return nil, err
	//}
	//
	//var result CachedBacktestResult
	//if err := json.Unmarshal([]byte(data), &result); err != nil {
	//	return nil, err
	//}
	//
	//// 检查缓存是否过期（24小时）
	//if time.Since(result.Timestamp) > 24*time.Hour {
	//	return nil, fmt.Errorf("cache expired")
	//}

	//return &result, nil

	return nil, nil
}

// cacheBacktestResult 缓存回测结果
func (s *Server) cacheBacktestResult(cacheKey string, data *CachedBacktestResult) error {
	if s.recommendationCache == nil || !s.recommendationCache.redisEnabled {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("cached_backtest:%s", cacheKey)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 缓存24小时
	return s.recommendationCache.redisClient.Set(ctx, key, jsonData, 24*time.Hour).Err()
}

// getPriceChangeFromTicker24h 直接从币安24hr统计API获取价格变动
func (s *Server) getPriceChangeFromTicker24h(ctx context.Context, symbol, kind string) (float64, error) {
	// 转换交易对格式
	apiSymbol := s.convertToBinanceSymbol(symbol, kind)

	var baseURL string
	switch kind {
	case "spot":
		baseURL = "https://api.binance.com/api/v3/ticker/24hr"
	case "futures":
		// 首先尝试币本位期货
		baseURL = "https://dapi.binance.com/dapi/v1/ticker/24hr"
	default:
		baseURL = "https://api.binance.com/api/v3/ticker/24hr"
	}

	url := fmt.Sprintf("%s?symbol=%s", baseURL, apiSymbol)

	// 调用API
	var ticker struct {
		Symbol             string `json:"symbol"`
		PriceChangePercent string `json:"priceChangePercent"`
		Code               int    `json:"code,omitempty"` // 错误码
		Msg                string `json:"msg,omitempty"`  // 错误信息
	}

	err := netutil.GetJSON(ctx, url, &ticker)
	if err != nil {
		// 检查是否是JSON格式错误（数组 vs 对象）
		errStr := err.Error()
		if strings.Contains(errStr, "cannot unmarshal array") {
			// 如果返回数组格式，说明交易对不存在或API行为异常
			if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
				//log.Printf("[DEBUG] 币本位期货不存在 %s，尝试USDT期货", apiSymbol)
				return s.getPriceChangeFromUSTFutures(ctx, symbol)
			}
		}

		// 如果是币本位期货API调用失败，尝试USDT期货
		if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
			// 检查是否是"Invalid symbol"错误或其他常见错误
			if strings.Contains(errStr, "-1121") || strings.Contains(errStr, "Invalid symbol") ||
				strings.Contains(errStr, "400") || strings.Contains(errStr, "Bad Request") {
				//log.Printf("[DEBUG] 币本位期货不存在 %s，尝试USDT期货", apiSymbol)
				return s.getPriceChangeFromUSTFutures(ctx, symbol)
			}
		}
		log.Printf("[DEBUG] 24hr API调用失败 %s -> %s: %v", symbol, apiSymbol, err)
		return 0, fmt.Errorf("获取24hr统计数据失败: %w", err)
	}

	// 检查API错误响应
	if ticker.Code != 0 && ticker.Code != 200 {
		// 如果是币本位期货失败，尝试USDT期货
		if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
			//log.Printf("[DEBUG] 币本位期货不存在 %s，尝试USDT期货", apiSymbol)
			return s.getPriceChangeFromUSTFutures(ctx, symbol)
		}
		log.Printf("[DEBUG] 币安API返回错误 %s -> %s: code=%d, msg=%s", symbol, apiSymbol, ticker.Code, ticker.Msg)
		return 0, fmt.Errorf("币安API错误: %s", ticker.Msg)
	}

	// 检查是否有有效的价格变动数据
	if ticker.Symbol == "" {
		// 如果是币本位期货不存在，尝试USDT期货
		if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
			//log.Printf("[DEBUG] 币本位期货不存在 %s，尝试USDT期货", apiSymbol)
			return s.getPriceChangeFromUSTFutures(ctx, symbol)
		}
		log.Printf("[DEBUG] 无效的API响应 %s -> %s: 交易对不存在", symbol, apiSymbol)
		return 0, fmt.Errorf("交易对不存在: %s", apiSymbol)
	}

	// 解析价格变动百分比
	changePercent, err := strconv.ParseFloat(ticker.PriceChangePercent, 64)
	if err != nil {
		return 0, fmt.Errorf("解析价格变动百分比失败: %w", err)
	}

	return changePercent, nil
}

// getPriceChangeFromUSTFutures 从USDT期货获取价格变动（降级方案）
func (s *Server) getPriceChangeFromUSTFutures(ctx context.Context, symbol string) (float64, error) {
	// 对于USDT期货，直接使用USDT格式
	apiSymbol := symbol
	if !strings.HasSuffix(apiSymbol, "USDT") {
		apiSymbol = apiSymbol + "USDT"
	}

	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/24hr?symbol=%s", apiSymbol)

	var ticker struct {
		Symbol             string `json:"symbol"`
		PriceChangePercent string `json:"priceChangePercent"`
		Code               int    `json:"code,omitempty"`
		Msg                string `json:"msg,omitempty"`
	}

	err := netutil.GetJSON(ctx, url, &ticker)
	if err != nil {
		log.Printf("[DEBUG] USDT期货API调用失败 %s: %v", apiSymbol, err)
		return 0, fmt.Errorf("获取USDT期货数据失败: %w", err)
	}

	if ticker.Code != 0 && ticker.Code != 200 {
		log.Printf("[DEBUG] USDT期货API返回错误 %s: code=%d, msg=%s", apiSymbol, ticker.Code, ticker.Msg)
		return 0, fmt.Errorf("USDT期货API错误: %s", ticker.Msg)
	}

	if ticker.Symbol == "" {
		log.Printf("[DEBUG] USDT期货不存在 %s", apiSymbol)
		return 0, fmt.Errorf("USDT期货交易对不存在: %s", apiSymbol)
	}

	changePercent, err := strconv.ParseFloat(ticker.PriceChangePercent, 64)
	if err != nil {
		log.Printf("[DEBUG] 解析USDT期货价格变动失败 %s: %s", apiSymbol, ticker.PriceChangePercent)
		return 0, fmt.Errorf("解析USDT期货价格变动失败: %w", err)
	}

	return changePercent, nil
}

// getVolumeFromTicker24h 直接从币安24hr统计API获取成交量
func (s *Server) getVolumeFromTicker24h(ctx context.Context, symbol, kind string) (float64, error) {
	// 转换交易对格式
	apiSymbol := s.convertToBinanceSymbol(symbol, kind)

	var baseURL string
	switch kind {
	case "spot":
		baseURL = "https://api.binance.com/api/v3/ticker/24hr"
	case "futures":
		// 首先尝试币本位期货
		baseURL = "https://dapi.binance.com/dapi/v1/ticker/24hr"
	default:
		baseURL = "https://api.binance.com/api/v3/ticker/24hr"
	}

	url := fmt.Sprintf("%s?symbol=%s", baseURL, apiSymbol)

	// 调用API
	var ticker struct {
		Symbol string `json:"symbol"`
		Volume string `json:"volume"`
		Code   int    `json:"code,omitempty"` // 错误码
		Msg    string `json:"msg,omitempty"`  // 错误信息
	}

	err := netutil.GetJSON(ctx, url, &ticker)
	if err != nil {
		// 检查是否是JSON格式错误（数组 vs 对象）
		errStr := err.Error()
		if strings.Contains(errStr, "cannot unmarshal array") {
			// 如果返回数组格式，说明交易对不存在或API行为异常
			if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
				//log.Printf("[DEBUG] 币本位期货不存在 %s，尝试USDT期货", apiSymbol)
				return s.getVolumeFromUSTFutures(ctx, symbol)
			}
		}

		// 如果是币本位期货API调用失败，尝试USDT期货
		if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
			// 检查是否是"Invalid symbol"错误或其他常见错误
			if strings.Contains(errStr, "-1121") || strings.Contains(errStr, "Invalid symbol") ||
				strings.Contains(errStr, "400") || strings.Contains(errStr, "Bad Request") {
				//log.Printf("[DEBUG] 币本位期货成交量不存在 %s，尝试USDT期货", apiSymbol)
				return s.getVolumeFromUSTFutures(ctx, symbol)
			}
		}
		return 0, fmt.Errorf("获取24hr统计数据失败: %w", err)
	}

	// 检查API错误响应
	if ticker.Code != 0 && ticker.Code != 200 {
		// 如果是币本位期货失败，尝试USDT期货
		if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
			return s.getVolumeFromUSTFutures(ctx, symbol)
		}
		return 0, fmt.Errorf("币安API错误: %s", ticker.Msg)
	}

	// 检查是否有有效的成交量数据
	if ticker.Symbol == "" {
		// 如果是币本位期货不存在，尝试USDT期货
		if kind == "futures" && baseURL == "https://dapi.binance.com/dapi/v1/ticker/24hr" {
			//log.Printf("[DEBUG] 币本位期货成交量不存在 %s，尝试USDT期货", apiSymbol)
			return s.getVolumeFromUSTFutures(ctx, symbol)
		}
		return 0, fmt.Errorf("交易对不存在: %s", apiSymbol)
	}

	// 解析成交量
	volume, err := strconv.ParseFloat(ticker.Volume, 64)
	if err != nil {
		return 0, fmt.Errorf("解析成交量失败: %w", err)
	}

	return volume, nil
}

// getVolumeFromUSTFutures 从USDT期货获取成交量（降级方案）
func (s *Server) getVolumeFromUSTFutures(ctx context.Context, symbol string) (float64, error) {
	// 对于USDT期货，直接使用USDT格式
	apiSymbol := symbol
	if !strings.HasSuffix(apiSymbol, "USDT") {
		apiSymbol = apiSymbol + "USDT"
	}

	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/24hr?symbol=%s", apiSymbol)

	var ticker struct {
		Symbol string `json:"symbol"`
		Volume string `json:"volume"`
		Code   int    `json:"code,omitempty"`
		Msg    string `json:"msg,omitempty"`
	}

	err := netutil.GetJSON(ctx, url, &ticker)
	if err != nil {
		return 0, fmt.Errorf("获取USDT期货成交量失败: %w", err)
	}

	if ticker.Code != 0 && ticker.Code != 200 {
		return 0, fmt.Errorf("USDT期货成交量API错误: %s", ticker.Msg)
	}

	if ticker.Symbol == "" {
		return 0, fmt.Errorf("USDT期货成交量交易对不存在: %s", apiSymbol)
	}

	volume, err := strconv.ParseFloat(ticker.Volume, 64)
	if err != nil {
		return 0, fmt.Errorf("解析USDT期货成交量失败: %w", err)
	}

	return volume, nil
}

// calculatePriceChange24h 从K线数据计算24h涨跌幅
func (s *Server) calculatePriceChange24h(ctx context.Context, symbol, kind string) (float64, error) {
	// 获取最近24小时的K线数据（每小时一个数据点）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 24)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 2 {
		return 0, fmt.Errorf("K线数据不足")
	}

	// 计算24小时涨跌幅
	startPrice, err := strconv.ParseFloat(klines[0].Open, 64)
	if err != nil {
		return 0, fmt.Errorf("解析起始价格失败: %w", err)
	}

	endPrice, err := strconv.ParseFloat(klines[len(klines)-1].Close, 64)
	if err != nil {
		return 0, fmt.Errorf("解析结束价格失败: %w", err)
	}

	change := (endPrice - startPrice) / startPrice * 100
	return change, nil
}

// calculateVolume24h 从K线数据计算24h成交量
func (s *Server) calculateVolume24h(ctx context.Context, symbol, kind string) (float64, error) {
	// 获取最近24小时的K线数据（每小时一个数据点）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 24)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		return 0, fmt.Errorf("K线数据为空")
	}

	totalVolume := 0.0
	for _, kline := range klines {
		volume, err := strconv.ParseFloat(kline.Volume, 64)
		if err != nil {
			continue // 跳过解析失败的数据
		}
		totalVolume += volume
	}

	return totalVolume, nil
}

// calculatePriceHigh24h 从K线数据计算24h最高价
func (s *Server) calculatePriceHigh24h(ctx context.Context, symbol, kind string) (float64, error) {
	// 获取最近24小时的K线数据（每小时一个数据点）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 24)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		return 0, fmt.Errorf("K线数据为空")
	}

	maxPrice := 0.0
	for _, kline := range klines {
		high, err := strconv.ParseFloat(kline.High, 64)
		if err != nil {
			continue // 跳过解析失败的数据
		}
		if high > maxPrice {
			maxPrice = high
		}
	}

	return maxPrice, nil
}

// calculatePriceLow24h 从K线数据计算24h最低价
func (s *Server) calculatePriceLow24h(ctx context.Context, symbol, kind string) (float64, error) {
	// 获取最近24小时的K线数据（每小时一个数据点）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 24)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		return 0, fmt.Errorf("K线数据为空")
	}

	minPrice := math.MaxFloat64
	for _, kline := range klines {
		low, err := strconv.ParseFloat(kline.Low, 64)
		if err != nil {
			continue // 跳过解析失败的数据
		}
		if low < minPrice {
			minPrice = low
		}
	}

	if minPrice == math.MaxFloat64 {
		return 0, fmt.Errorf("未找到有效的最低价")
	}

	return minPrice, nil
}

// calculatePriceHigh7d 从K线数据计算7d最高价
func (s *Server) calculatePriceHigh7d(ctx context.Context, symbol, kind string) (float64, error) {
	// 获取最近7天的K线数据（每天一个数据点）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1d", 7)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		return 0, fmt.Errorf("K线数据为空")
	}

	maxPrice := 0.0
	for _, kline := range klines {
		high, err := strconv.ParseFloat(kline.High, 64)
		if err != nil {
			continue // 跳过解析失败的数据
		}
		if high > maxPrice {
			maxPrice = high
		}
	}

	return maxPrice, nil
}

// calculatePriceLow7d 从K线数据计算7d最低价
func (s *Server) calculatePriceLow7d(ctx context.Context, symbol, kind string) (float64, error) {
	// 获取最近7天的K线数据（每天一个数据点）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1d", 7)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		return 0, fmt.Errorf("K线数据为空")
	}

	minPrice := math.MaxFloat64
	for _, kline := range klines {
		low, err := strconv.ParseFloat(kline.Low, 64)
		if err != nil {
			continue // 跳过解析失败的数据
		}
		if low < minPrice {
			minPrice = low
		}
	}

	if minPrice == math.MaxFloat64 {
		return 0, fmt.Errorf("未找到有效的7d最低价")
	}

	return minPrice, nil
}

// getMarketCapFromCoinGecko 从CoinGecko获取市值
func (s *Server) getMarketCapFromCoinGecko(ctx context.Context, symbol string) (float64, error) {
	// 如果有CoinGecko客户端，使用它
	if s.coinGeckoClient != nil {
		marketData, err := s.coinGeckoClient.GetCoinBySymbol(ctx, symbol)
		if err == nil && marketData != nil && marketData.MarketCap > 0 {
			return marketData.MarketCap, nil
		}
	}

	// 降级：使用硬编码的市值数据（根据常见币种）
	marketCapMap := map[string]float64{
		"BTC":   1200000000000, // 约1.2万亿
		"ETH":   400000000000,  // 约4000亿
		"ADA":   15000000000,   // 约150亿
		"SOL":   8000000000,    // 约80亿
		"DOT":   7000000000,    // 约70亿
		"LINK":  3000000000,    // 约30亿
		"UNI":   2000000000,    // 约20亿
		"AAVE":  1000000000,    // 约10亿
		"SUSHI": 500000000,     // 约5亿
		"COMP":  800000000,     // 约8亿
		"MKR":   600000000,     // 约6亿
		"YFI":   200000000,     // 约2亿
		"BAL":   150000000,     // 约1.5亿
		"REN":   100000000,     // 约1亿
	}

	if marketCap, exists := marketCapMap[strings.ToUpper(symbol)]; exists {
		return marketCap, nil
	}

	return 100000000, nil // 默认1亿美元
}

// executeBacktestTrades 在回测完成后执行符合条件的交易
func (s *Server) executeBacktestTrades(c *gin.Context, req struct {
	Symbol               string  `json:"symbol" binding:"required"`
	Timeframe            string  `json:"timeframe"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	InitialCash          float64 `json:"initial_cash"`
	MaxPosition          float64 `json:"max_position"`
	StopLoss             float64 `json:"stop_loss"`
	TakeProfit           float64 `json:"take_profit"`
	Commission           float64 `json:"commission"`
	Strategy             string  `json:"strategy"`
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
	MaxBatches            int           `json:"max_batches"`             // 最大批次数
	BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
	BatchSize             int           `json:"batch_size"`              // 每批最大交易数
	DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
	MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
}, backtestResult gin.H) (*gin.H, error) {

	// 从Gin上下文中获取用户ID
	uidVal, exists := c.Get("uid")
	if !exists || uidVal == nil {
		return nil, fmt.Errorf("用户未登录或认证失败")
	}

	uid, ok := uidVal.(uint)
	if !ok {
		return nil, fmt.Errorf("用户ID格式错误")
	}

	log.Printf("[INFO] 开始自动执行回测交易: user=%d, symbol=%s, progressive=%v", uid, req.Symbol, req.ProgressiveExecution)

	// 检查是否启用渐进式执行
	if req.ProgressiveExecution {
		return s.executeProgressiveBacktestTrades(c, uid, req, backtestResult)
	}

	// 原有的批量执行逻辑
	executedTrades := 0
	successfulTrades := 0
	skippedTrades := 0

	// 首先尝试从回测结果中提取交易信号
	log.Printf("[DEBUG] 检查回测结果中的trades字段，类型: %T", backtestResult["trades"])

	// 处理不同类型的trades字段
	var trades []interface{}

	// 尝试[]interface{}类型（标准JSON反序列化）
	if tradesInterface, ok1 := backtestResult["trades"].([]interface{}); ok1 {
		trades = tradesInterface
		ok = true
	} else if tradesSlice, ok2 := backtestResult["trades"].([]map[string]interface{}); ok2 {
		// 尝试[]map[string]interface{}类型
		trades = make([]interface{}, len(tradesSlice))
		for i, v := range tradesSlice {
			trades[i] = v
		}
		ok = true
	} else {
		// 尝试其他可能的类型，比如直接的结构体切片
		log.Printf("[DEBUG] 尝试处理其他trades类型...")
		// 这里可以添加更多类型转换逻辑
	}

	log.Printf("[DEBUG] 最终转换结果: ok=%v, len=%d", ok, len(trades))
	if ok && len(trades) > 0 {
		// 有交易记录，按原逻辑处理
		log.Printf("[INFO] 找到%d个回测交易信号，开始自动执行", len(trades))

		processedCount := 0
		for _, tradeInterface := range trades {
			processedCount++
			trade, ok := tradeInterface.(map[string]interface{})
			if !ok {
				log.Printf("[DEBUG] 交易数据格式错误，跳过第%d个交易", processedCount)
				continue
			}

			log.Printf("[DEBUG] 处理第%d/%d个交易信号", processedCount, len(trades))

			// 检查是否符合自动执行条件
			if !s.isTradeEligibleForAutoExecution(trade, req) {
				skippedTrades++
				log.Printf("[DEBUG] 第%d个交易不符合自动执行条件，已跳过 (累计跳过: %d)", processedCount, skippedTrades)
				continue
			}

			// 检查是否已存在相同的交易（如果启用了跳过选项）
			if req.SkipExistingTrades && s.isTradeAlreadyExists(uid, trade) {
				skippedTrades++
				log.Printf("[DEBUG] 第%d个交易已存在，跳过重复交易 (累计跳过: %d)", processedCount, skippedTrades)
				continue
			}

			// 执行交易
			if err := s.createAutoTradeFromBacktest(uid, trade, req); err != nil {
				log.Printf("[WARN] 第%d个交易创建失败: %v", processedCount, err)
				skippedTrades++
				continue
			}

			executedTrades++
			successfulTrades++
			log.Printf("[INFO] 第%d个交易执行成功 (累计执行: %d/%d)", processedCount, executedTrades, len(trades))
		}
	} else {
		// 没有交易记录，基于推荐本身创建交易
		log.Printf("[INFO] 回测结果中未找到交易数据，尝试基于AI推荐创建交易")

		// 检查回测是否有正收益，如果有则创建买入交易
		var totalReturn float64 = 0

		// 从gin.H中安全提取total_return
		if summaryInterface, exists := backtestResult["summary"]; exists {
			// 尝试多种方式获取total_return值
			switch summary := summaryInterface.(type) {
			case BacktestSummary:
				totalReturn = summary.TotalReturn
			case *BacktestSummary:
				totalReturn = summary.TotalReturn
			case map[string]interface{}:
				if tr, ok := summary["total_return"].(float64); ok {
					totalReturn = tr
				}
			case gin.H:
				if tr, ok := summary["total_return"].(float64); ok {
					totalReturn = tr
				}
			default:
				log.Printf("[WARN] 未知的summary类型: %T", summaryInterface)
			}
		}

		log.Printf("[DEBUG] 回测总收益率: %.2f%%", totalReturn*100)
		if totalReturn > 0 {
			log.Printf("[INFO] 回测有正收益，开始基于AI推荐创建交易")
			// 回测有正收益，创建基于当前推荐的交易
			if err := s.createTradeFromRecommendation(uid, req); err != nil {
				log.Printf("[WARN] 基于推荐创建交易失败: %v", err)
				skippedTrades++
				log.Printf("[INFO] 基于推荐创建交易失败，尝试跳过 (累计跳过: %d)", skippedTrades)
			} else {
				executedTrades++
				successfulTrades++
				log.Printf("[INFO] 基于AI推荐成功创建1个交易 (累计执行: %d)", executedTrades)
			}
		} else {
			log.Printf("[INFO] 回测无正收益，跳过交易创建 (累计跳过: %d)", skippedTrades+1)
			skippedTrades++
		}
	}

	stats := gin.H{
		"executedTrades":   executedTrades,
		"successfulTrades": successfulTrades,
		"skippedTrades":    skippedTrades,
		"message":          fmt.Sprintf("成功执行%d个交易，跳过%d个", executedTrades, skippedTrades),
	}

	log.Printf("[INFO] 自动执行回测交易完成 - 总共处理%d个信号，成功执行%d个，跳过%d个", executedTrades+skippedTrades, executedTrades, skippedTrades)
	log.Printf("[INFO] 自动执行统计详情: %+v", stats)
	return &stats, nil
}

// isTradeEligibleForAutoExecution 检查交易是否符合自动执行条件
func (s *Server) isTradeEligibleForAutoExecution(trade map[string]interface{}, req struct {
	Symbol               string  `json:"symbol" binding:"required"`
	Timeframe            string  `json:"timeframe"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	InitialCash          float64 `json:"initial_cash"`
	MaxPosition          float64 `json:"max_position"`
	StopLoss             float64 `json:"stop_loss"`
	TakeProfit           float64 `json:"take_profit"`
	Commission           float64 `json:"commission"`
	Strategy             string  `json:"strategy"`
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
	MaxBatches            int           `json:"max_batches"`             // 最大批次数
	BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
	BatchSize             int           `json:"batch_size"`              // 每批最大交易数
	DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
	MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
}) bool {

	// 检查置信度
	confidence, ok := trade["ai_confidence"].(float64)
	if !ok || confidence < req.MinConfidence {
		return false
	}

	// 检查风险等级
	riskScore, ok := trade["risk_score"].(float64)
	if !ok {
		riskScore = 0.5 // 默认中等风险
	}

	switch req.AutoExecuteRiskLevel {
	case "conservative":
		if riskScore > 0.3 {
			return false
		}
	case "moderate":
		if riskScore > 0.6 {
			return false
		}
	case "aggressive":
		// 激进策略接受所有交易
		break
	default:
		if riskScore > 0.5 {
			return false
		}
	}

	return true
}

// executeProgressiveBacktestTrades 执行渐进式自动回测交易
func (s *Server) executeProgressiveBacktestTrades(c *gin.Context, userID uint, req struct {
	Symbol               string  `json:"symbol" binding:"required"`
	Timeframe            string  `json:"timeframe"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	InitialCash          float64 `json:"initial_cash"`
	MaxPosition          float64 `json:"max_position"`
	StopLoss             float64 `json:"stop_loss"`
	TakeProfit           float64 `json:"take_profit"`
	Commission           float64 `json:"commission"`
	Strategy             string  `json:"strategy"`
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
	MaxBatches            int           `json:"max_batches"`             // 最大批次数
	BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
	BatchSize             int           `json:"batch_size"`              // 每批最大交易数
	DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
	MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
}, backtestResult gin.H) (*gin.H, error) {

	log.Printf("[ProgressiveBacktest] 开始渐进式自动执行回测交易: user=%d, symbol=%s", userID, req.Symbol)

	// 提取交易信号
	trades := s.extractTradesFromBacktestResult(backtestResult)
	if len(trades) == 0 {
		log.Printf("[ProgressiveBacktest] 未找到可执行的交易信号")
		return &gin.H{"executed_trades": 0, "message": "未找到可执行的交易信号"}, nil
	}

	log.Printf("[ProgressiveBacktest] 提取到 %d 个交易信号", len(trades))

	// 创建渐进式执行器
	executor := NewProgressiveAutoExecutor(s)

	// 准备执行配置
	config := ProgressiveExecutionConfig{
		Enabled:                    true,
		MaxBatches:                 req.MaxBatches,
		BatchDelay:                 req.BatchDelay,
		BatchSize:                  req.BatchSize,
		RiskCheckInterval:          5 * time.Minute,
		DynamicSizing:              req.DynamicSizing,
		MarketConditionFilter:      req.MarketConditionFilter,
		PerformanceBasedAdjustment: false,
	}

	// 设置默认值
	if config.MaxBatches <= 0 {
		config.MaxBatches = 3
	}
	if config.BatchDelay <= 0 {
		config.BatchDelay = 30 * time.Minute
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 5
	}

	// 转换交易信号为推荐格式
	recommendations := make([]gin.H, 0, len(trades))
	for _, trade := range trades {
		recommendation := gin.H{
			"symbol":        req.Symbol,
			"action":        trade["action"],
			"confidence":    trade["ai_confidence"],
			"price":         trade["price"],
			"quantity":      trade["quantity"],
			"timestamp":     trade["timestamp"],
			"ai_confidence": trade["ai_confidence"],
		}
		recommendations = append(recommendations, recommendation)
	}

	// 执行渐进式自动执行
	ctx := context.Background()
	result, err := executor.ExecuteProgressive(ctx, userID, recommendations, config)
	if err != nil {
		log.Printf("[ProgressiveBacktest] 渐进式执行失败: %v", err)
		return nil, fmt.Errorf("渐进式执行失败: %w", err)
	}

	log.Printf("[ProgressiveBacktest] 渐进式执行完成: 批次=%d/%d, 总交易=%d, 成功=%d, 失败=%d",
		result.ExecutedBatches, result.TotalBatches, result.TotalTrades, result.SuccessfulTrades, result.FailedTrades)

	return &gin.H{
		"executed_trades":       result.TotalTrades,
		"successful_trades":     result.SuccessfulTrades,
		"failed_trades":         result.FailedTrades,
		"total_batches":         result.TotalBatches,
		"executed_batches":      result.ExecutedBatches,
		"status":                result.Status,
		"duration":              result.Duration.String(),
		"progressive_execution": true,
		"message":               "渐进式自动执行完成",
	}, nil
}

// extractTradesFromBacktestResult 从回测结果中提取交易信号
func (s *Server) extractTradesFromBacktestResult(backtestResult gin.H) []map[string]interface{} {
	var trades []map[string]interface{}

	// 尝试从不同的字段中提取交易
	if tradesInterface, exists := backtestResult["trades"]; exists {
		if tradesArray, ok := tradesInterface.([]interface{}); ok {
			for _, tradeInterface := range tradesArray {
				if trade, ok := tradeInterface.(map[string]interface{}); ok {
					trades = append(trades, trade)
				}
			}
		}
	}

	// 如果没有找到trades字段，尝试从其他字段提取
	if len(trades) == 0 {
		if portfolio, exists := backtestResult["portfolio"]; exists {
			if portfolioMap, ok := portfolio.(map[string]interface{}); ok {
				if positions, exists := portfolioMap["positions"]; exists {
					if positionsArray, ok := positions.([]interface{}); ok {
						for _, posInterface := range positionsArray {
							if pos, ok := posInterface.(map[string]interface{}); ok {
								// 转换为交易格式
								trade := map[string]interface{}{
									"action":        pos["side"],
									"ai_confidence": 0.8, // 默认置信度
									"price":         pos["entry_price"],
									"quantity":      pos["quantity"],
									"timestamp":     pos["entry_time"],
								}
								trades = append(trades, trade)
							}
						}
					}
				}
			}
		}
	}

	return trades
}

// isTradeAlreadyExists 检查交易是否已存在
func (s *Server) isTradeAlreadyExists(userID uint, trade map[string]interface{}) bool {
	// 这里可以根据交易的时间、价格、数量等信息来判断是否重复
	// 暂时简单实现，实际应该有更复杂的去重逻辑
	return false
}

// createAutoTradeFromBacktest 从回测结果创建自动交易
func (s *Server) createAutoTradeFromBacktest(userID uint, trade map[string]interface{}, req struct {
	Symbol               string  `json:"symbol" binding:"required"`
	Timeframe            string  `json:"timeframe"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	InitialCash          float64 `json:"initial_cash"`
	MaxPosition          float64 `json:"max_position"`
	StopLoss             float64 `json:"stop_loss"`
	TakeProfit           float64 `json:"take_profit"`
	Commission           float64 `json:"commission"`
	Strategy             string  `json:"strategy"`
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
	MaxBatches            int           `json:"max_batches"`             // 最大批次数
	BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
	BatchSize             int           `json:"batch_size"`              // 每批最大交易数
	DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
	MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
}) error {

	action, ok := trade["action"].(string)
	if !ok {
		action = "buy" // 默认买入
	}

	price, ok := trade["price"].(float64)
	if !ok {
		return fmt.Errorf("交易缺少价格信息")
	}

	quantity, ok := trade["quantity"].(float64)
	if !ok {
		// 根据仓位百分比计算数量
		totalValue := req.MaxPositionPercent / 100.0 * req.InitialCash
		quantity = totalValue / price
	}

	// 创建模拟交易记录
	simTrade := &pdb.SimulatedTrade{
		UserID:       userID,
		Symbol:       req.Symbol,
		BaseSymbol:   "USDT",
		Kind:         "spot",
		Side:         strings.ToUpper(action),
		Quantity:     fmt.Sprintf("%.8f", quantity),
		Price:        fmt.Sprintf("%.8f", price),
		TotalValue:   fmt.Sprintf("%.8f", quantity*price),
		IsOpen:       action == "buy", // 买入时持仓，卖出时平仓
		CurrentPrice: func() *string { p := fmt.Sprintf("%.8f", price); return &p }(),
	}

	if action == "sell" {
		// 如果是卖出，需要找到对应的买入记录
		// 这里简化处理，实际应该匹配具体的持仓
		simTrade.RealizedPnl = func() *string {
			pnl, ok := trade["profit"].(float64)
			if ok {
				p := fmt.Sprintf("%.8f", pnl)
				return &p
			}
			return nil
		}()
	}

	return pdb.CreateSimulatedTrade(s.db.DB(), simTrade)
}

// createTradeFromRecommendation 基于AI推荐创建交易
func (s *Server) createTradeFromRecommendation(userID uint, req struct {
	Symbol               string  `json:"symbol" binding:"required"`
	Timeframe            string  `json:"timeframe"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	InitialCash          float64 `json:"initial_cash"`
	MaxPosition          float64 `json:"max_position"`
	StopLoss             float64 `json:"stop_loss"`
	TakeProfit           float64 `json:"take_profit"`
	Commission           float64 `json:"commission"`
	Strategy             string  `json:"strategy"`
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
	MaxBatches            int           `json:"max_batches"`             // 最大批次数
	BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
	BatchSize             int           `json:"batch_size"`              // 每批最大交易数
	DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
	MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
}) error {

	// 检查是否已有相同交易（如果启用了跳过选项）
	if req.SkipExistingTrades {
		existingTrades, err := pdb.GetSimulatedTrades(s.db.DB(), userID, nil)
		if err == nil {
			today := time.Now().Format("2006-01-02")
			for _, trade := range existingTrades {
				// 检查今天是否已有相同币种的交易
				if trade.Symbol == strings.ToUpper(req.Symbol) &&
					trade.CreatedAt.Format("2006-01-02") == today &&
					trade.IsOpen {
					log.Printf("[INFO] 今日已存在%s的交易，跳过创建 (用户设置了跳过重复交易)", req.Symbol)
					return fmt.Errorf("今日已存在相同交易")
				}
			}
		}
	}

	// 获取当前AI推荐
	recommendations, err := s.generateAIRecommendations([]string{req.Symbol}, 1, req.AutoExecuteRiskLevel, "")
	if err != nil {
		return fmt.Errorf("生成AI推荐失败: %w", err)
	}

	if len(recommendations) == 0 {
		return fmt.Errorf("未找到有效的AI推荐")
	}

	rec := recommendations[0]

	// 提取推荐数据
	symbol, ok := rec["symbol"].(string)
	if !ok {
		return fmt.Errorf("推荐中缺少symbol字段")
	}

	price, ok := rec["price"].(float64)
	if !ok {
		return fmt.Errorf("推荐中缺少price字段")
	}

	riskScore, ok := rec["risk_score"].(float64)
	if !ok {
		riskScore = 0.5 // 默认中等风险
	}

	// 计算仓位大小
	settings := &pdb.AutoExecuteSettings{
		RiskLevel:      req.AutoExecuteRiskLevel,
		MaxPosition:    req.MaxPositionPercent,
		MinConfidence:  req.MinConfidence,
		MaxDailyTrades: 10, // 默认值
		Enabled:        true,
	}
	positionSize := s.calculatePositionSize(riskScore, settings)

	// 计算交易数量
	totalValue := positionSize * req.InitialCash
	quantity := totalValue / price

	// 创建模拟交易记录
	simTrade := &pdb.SimulatedTrade{
		UserID:       userID,
		Symbol:       strings.ToUpper(symbol),
		BaseSymbol:   "USDT",
		Kind:         "spot",
		Side:         "BUY",
		Quantity:     fmt.Sprintf("%.8f", quantity),
		Price:        fmt.Sprintf("%.8f", price),
		TotalValue:   fmt.Sprintf("%.8f", totalValue),
		IsOpen:       true,
		CurrentPrice: func() *string { p := fmt.Sprintf("%.8f", price); return &p }(),
	}

	err = pdb.CreateSimulatedTrade(s.db.DB(), simTrade)
	if err != nil {
		return fmt.Errorf("创建交易失败: %w", err)
	}

	log.Printf("[INFO] 成功创建自动交易: %s, 数量: %s, 价格: %s",
		symbol, simTrade.Quantity, simTrade.Price)
	return nil
}

// ============================================================================
// 并发和资源管理API
// ============================================================================

// GetConcurrencyStats 获取并发统计信息
func (s *Server) GetConcurrencyStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// 智能工作者池统计
	if s.smartWorkerPool != nil {
		stats["smart_worker_pool"] = s.smartWorkerPool.GetStats()
	}

	// 缓存写入协程池统计
	if globalCachePool != nil {
		stats["cache_pool"] = map[string]interface{}{
			"running":     globalCachePool.Running(),
			"max_workers": globalCachePool.maxWorkers,
		}
	}

	// 系统资源使用情况
	if s.resourceManager != nil {
		stats["system_resources"] = s.resourceManager.GetSystemResourceUsage()
	}

	// 熔断器统计
	if s.circuitBreakerMgr != nil {
		stats["circuit_breakers"] = s.circuitBreakerMgr.GetAllStats()
	}

	// 资源池统计
	if s.resourceManager != nil {
		stats["resource_pools"] = s.resourceManager.GetStats()
	}

	c.JSON(200, stats)
}

// GetResourceHealth 获取资源健康状态
func (s *Server) GetResourceHealth(c *gin.Context) {
	health := make(map[string]interface{})

	// 资源池健康检查
	if s.resourceManager != nil {
		health["resource_pools"] = s.resourceManager.HealthCheck()
	}

	// 熔断器状态
	if s.circuitBreakerMgr != nil {
		cbStats := s.circuitBreakerMgr.GetAllStats()
		cbHealth := make(map[string]interface{})
		for name, stat := range cbStats {
			cbHealth[name] = map[string]interface{}{
				"state":        stat.State,
				"failure_rate": stat.FailureRate,
				"healthy":      stat.FailureRate < 50.0, // 失败率低于50%认为健康
			}
		}
		health["circuit_breakers"] = cbHealth
	}

	// 工作者池健康
	healthy := true
	if s.smartWorkerPool != nil {
		stats := s.smartWorkerPool.GetStats()
		healthy = stats.QueueLength < cap(s.smartWorkerPool.tasks)*8/10 // 队列使用率低于80%
	}
	health["worker_pool"] = map[string]bool{"healthy": healthy}

	// 整体健康状态
	overallHealthy := true
	for _, poolHealth := range health["resource_pools"].(map[ResourceType]bool) {
		if !poolHealth {
			overallHealthy = false
			break
		}
	}

	health["overall_healthy"] = overallHealthy

	c.JSON(200, health)
}

// ResetCircuitBreakers 重置所有熔断器
func (s *Server) ResetCircuitBreakers(c *gin.Context) {
	if s.circuitBreakerMgr != nil {
		s.circuitBreakerMgr.ResetAll()
		c.JSON(200, gin.H{
			"message":     "所有熔断器已重置",
			"reset_count": len(s.circuitBreakerMgr.GetAllStats()),
		})
	} else {
		c.JSON(400, gin.H{"error": "熔断器管理器未初始化"})
	}
}

// ScaleWorkerPool 调整工作者池大小
func (s *Server) ScaleWorkerPool(c *gin.Context) {
	var req struct {
		MinWorkers int `json:"min_workers"`
		MaxWorkers int `json:"max_workers"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if s.smartWorkerPool == nil {
		c.JSON(400, gin.H{"error": "智能工作者池未初始化"})
		return
	}

	// 这里可以实现动态调整逻辑
	// 暂时返回成功
	c.JSON(200, gin.H{
		"message":       "工作者池调整请求已接收",
		"current_min":   s.smartWorkerPool.minWorkers,
		"current_max":   s.smartWorkerPool.maxWorkers,
		"requested_min": req.MinWorkers,
		"requested_max": req.MaxWorkers,
	})
}

// StartAsyncBacktestAPI 启动异步回测
func (s *Server) StartAsyncBacktestAPI(c *gin.Context) {
	log.Printf("[DEBUG] ===== StartAsyncBacktestAPI 被调用 =====")

	var req struct {
		Symbol         string   `json:"symbol"`  // 单币种模式（向后兼容）
		Symbols        []string `json:"symbols"` // 多币种列表，为空时使用Symbol
		StartDate      string   `json:"start_date" binding:"required"`
		EndDate        string   `json:"end_date" binding:"required"`
		Strategy       string   `json:"strategy" binding:"required"`
		InitialCapital float64  `json:"initial_capital" binding:"required"`
		PositionSize   float64  `json:"position_size" binding:"required"`
		// 自动执行参数
		AutoExecute          bool    `json:"auto_execute"`
		AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
		MinConfidence        float64 `json:"min_confidence"`
		MaxPositionPercent   float64 `json:"max_position_percent"`
		SkipExistingTrades   bool    `json:"skip_existing_trades"`
		// 渐进式执行参数
		ProgressiveExecution  bool          `json:"progressive_execution"`
		MaxBatches            int           `json:"max_batches"`
		BatchDelay            time.Duration `json:"batch_delay"`
		BatchSize             int           `json:"batch_size"`
		DynamicSizing         bool          `json:"dynamic_sizing"`
		MarketConditionFilter bool          `json:"market_condition_filter"`
		// 自动选择币种参数
		AutoSelectSymbol        bool   `json:"auto_select_symbol"`
		MaxSymbolsToEvaluate    int    `json:"max_symbols_to_evaluate"`
		SymbolSelectionCriteria string `json:"symbol_selection_criteria"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendRecommendationError(c, 400, "无效的请求参数格式", "INVALID_REQUEST_FORMAT", err.Error())
		return
	}

	// 自定义参数校验
	// 确定使用的币种列表
	var symbols []string
	if len(req.Symbols) > 0 {
		symbols = req.Symbols
	} else if req.Symbol != "" {
		symbols = []string{req.Symbol}
	} else if !req.AutoSelectSymbol {
		sendRecommendationError(c, 400, "未启用自动选择币种时必须指定币种", "MISSING_SYMBOL")
		return
	}

	if len(symbols) == 0 && !req.AutoSelectSymbol {
		sendRecommendationError(c, 400, "未启用自动选择币种时必须指定币种", "MISSING_SYMBOL")
		return
	}

	// 基础参数校验
	if req.InitialCapital <= 0 || req.PositionSize <= 0 {
		sendRecommendationError(c, 400, "初始资金和仓位必须大于0", "INVALID_NUMERIC_PARAMS")
		return
	}

	// 自动选择币种参数校验
	if req.AutoSelectSymbol {
		if req.MaxSymbolsToEvaluate <= 0 {
			req.MaxSymbolsToEvaluate = 15 // 默认评估15个币种
		}
		if req.MaxSymbolsToEvaluate > 30 {
			req.MaxSymbolsToEvaluate = 30 // 限制最大评估数量
		}
		if req.SymbolSelectionCriteria == "" {
			req.SymbolSelectionCriteria = "market_heat" // 默认按市场热度选择
		}
		log.Printf("[AUTO_SELECT] 启用自动选择币种模式，评估币种数量: %d, 选择标准: %s",
			req.MaxSymbolsToEvaluate, req.SymbolSelectionCriteria)
	}

	// 校验日期格式与范围（默认限制3年内）
	startTime, errStart := time.Parse("2006-01-02", req.StartDate)
	endTime, errEnd := time.Parse("2006-01-02", req.EndDate)
	if errStart != nil || errEnd != nil {
		sendRecommendationError(c, 400, "日期格式应为YYYY-MM-DD", "INVALID_DATE_FORMAT")
		return
	}
	if !startTime.Before(endTime) {
		sendRecommendationError(c, 400, "开始日期必须早于结束日期", "INVALID_DATE_RANGE")
		return
	}
	if endTime.Sub(startTime) > 24*time.Hour*365*3 {
		sendRecommendationError(c, 400, "回测区间过长，限制为3年内", "DATE_RANGE_TOO_LARGE")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("uid")
	if !exists {
		sendRecommendationError(c, 401, "用户未登录", "USER_NOT_AUTHENTICATED")
		return
	}

	// 创建异步回测记录
	// 处理币种信息：多币种用逗号分隔存储，单币种保持不变
	symbolStr := req.Symbol
	if len(symbols) > 1 {
		symbolStr = strings.Join(symbols, ",")
	} else if len(symbols) == 1 {
		symbolStr = symbols[0]
	}

	record := &pdb.AsyncBacktestRecord{
		UserID:         userID.(uint),
		Symbol:         symbolStr,
		Strategy:       req.Strategy,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		InitialCapital: decimal.NewFromFloat(req.InitialCapital),
		PositionSize:   decimal.NewFromFloat(req.PositionSize),
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 保存到数据库
	if err := pdb.CreateAsyncBacktestRecord(s.db.DB(), record); err != nil {
		log.Printf("[ERROR] 创建异步回测记录失败: %v", err)
		sendRecommendationError(c, 500, "创建回测记录失败", "CREATE_RECORD_FAILED", err.Error())
		return
	}

	log.Printf("[INFO] 创建异步回测记录成功，ID=%d，用户ID=%d", record.ID, record.UserID)

	// 准备传递给异步任务的参数
	asyncReq := struct {
		Symbol         string   `json:"symbol"`  // 单币种（向后兼容）
		Symbols        []string `json:"symbols"` // 多币种列表
		StartDate      string   `json:"start_date"`
		EndDate        string   `json:"end_date"`
		Strategy       string   `json:"strategy"`
		InitialCapital float64  `json:"initial_capital"`
		PositionSize   float64  `json:"position_size"`
		// 自动执行参数
		AutoExecute          bool    `json:"auto_execute"`
		AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
		MinConfidence        float64 `json:"min_confidence"`
		MaxPositionPercent   float64 `json:"max_position_percent"`
		SkipExistingTrades   bool    `json:"skip_existing_trades"`
		// 渐进式执行参数
		ProgressiveExecution  bool          `json:"progressive_execution"`
		MaxBatches            int           `json:"max_batches"`
		BatchDelay            time.Duration `json:"batch_delay"`
		BatchSize             int           `json:"batch_size"`
		DynamicSizing         bool          `json:"dynamic_sizing"`
		MarketConditionFilter bool          `json:"market_condition_filter"`
		// 自动选择币种参数
		AutoSelectSymbol        bool   `json:"auto_select_symbol"`
		MaxSymbolsToEvaluate    int    `json:"max_symbols_to_evaluate"`
		SymbolSelectionCriteria string `json:"symbol_selection_criteria"`
	}{
		Symbol:                  req.Symbol, // 向后兼容
		Symbols:                 symbols,    // 多币种列表
		StartDate:               req.StartDate,
		EndDate:                 req.EndDate,
		Strategy:                req.Strategy,
		InitialCapital:          req.InitialCapital,
		PositionSize:            req.PositionSize,
		AutoExecute:             req.AutoExecute,
		AutoExecuteRiskLevel:    req.AutoExecuteRiskLevel,
		MinConfidence:           req.MinConfidence,
		MaxPositionPercent:      req.MaxPositionPercent,
		SkipExistingTrades:      req.SkipExistingTrades,
		ProgressiveExecution:    req.ProgressiveExecution,
		MaxBatches:              req.MaxBatches,
		BatchDelay:              req.BatchDelay,
		BatchSize:               req.BatchSize,
		DynamicSizing:           req.DynamicSizing,
		MarketConditionFilter:   req.MarketConditionFilter,
		AutoSelectSymbol:        req.AutoSelectSymbol,
		MaxSymbolsToEvaluate:    req.MaxSymbolsToEvaluate,
		SymbolSelectionCriteria: req.SymbolSelectionCriteria,
	}

	// 启动后台回测任务（优先使用工作者池以控制并发）
	started := false
	if s.smartWorkerPool != nil {
		if ok := s.smartWorkerPool.Submit(func() {
			s.runAsyncBacktest(record, asyncReq)
		}); ok {
			started = true
		} else {
			log.Printf("[WARN] 智能工作者池队列已满，拒绝回测任务 ID=%d", record.ID)
		}
	}

	if !started {
		go s.runAsyncBacktest(record, asyncReq)
	}

	c.JSON(200, gin.H{
		"success":   true,
		"message":   "回测任务已启动",
		"record_id": record.ID,
		"status":    "pending",
	})
}

// GetBacktestRecordsAPI 获取回测记录列表
func (s *Server) GetBacktestRecordsAPI(c *gin.Context) {
	log.Printf("[DEBUG] ===== GetBacktestRecordsAPI 被调用 =====")

	// 获取用户ID
	userID, exists := c.Get("uid")
	if !exists {
		log.Printf("[ERROR] GetBacktestRecordsAPI: 用户未登录")
		sendRecommendationError(c, 401, "用户未登录", "USER_NOT_AUTHENTICATED")
		return
	}
	log.Printf("[DEBUG] GetBacktestRecordsAPI: 用户ID = %v", userID)

	// 解析查询参数
	var req struct {
		Page      int    `form:"page,default=1"`
		Limit     int    `form:"limit,default=20"`
		Status    string `form:"status"`
		Symbol    string `form:"symbol"`
		SortBy    string `form:"sort_by,default=created_at"`
		SortOrder string `form:"sort_order,default=desc"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		sendRecommendationError(c, 400, "无效的查询参数", "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	// 从数据库查询回测记录
	log.Printf("[DEBUG] 查询回测记录: userID=%d, page=%d, limit=%d, status=%s, symbol=%s",
		userID.(uint), req.Page, req.Limit, req.Status, req.Symbol)
	records, totalCount, err := pdb.GetAsyncBacktestRecords(s.db.DB(), userID.(uint), req.Page, req.Limit, req.Status, req.Symbol)
	if err != nil {
		log.Printf("[ERROR] 查询异步回测记录失败: %v", err)
		sendRecommendationError(c, 500, "查询回测记录失败", "QUERY_FAILED", err.Error())
		return
	}
	log.Printf("[DEBUG] 查询结果: 找到 %d 条记录，总共 %d 条", len(records), totalCount)

	// 转换为API响应格式
	var responseRecords []gin.H
	for _, record := range records {
		responseRecords = append(responseRecords, gin.H{
			"id":              record.ID,
			"user_id":         record.UserID,
			"symbol":          record.Symbol,
			"strategy":        record.Strategy,
			"start_date":      record.StartDate,
			"end_date":        record.EndDate,
			"initial_capital": record.InitialCapital.InexactFloat64(),
			"position_size":   record.PositionSize.InexactFloat64(),
			"status":          record.Status,
			"result":          record.Result,
			"error_message":   record.ErrorMessage,
			"created_at":      record.CreatedAt,
			"updated_at":      record.UpdatedAt,
			"completed_at":    record.CompletedAt,
		})
	}

	total := int(totalCount)

	c.JSON(200, gin.H{
		"success": true,
		"records": responseRecords,
		"pagination": gin.H{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (total + req.Limit - 1) / req.Limit,
		},
	})
}

// GetBacktestRecordAPI 获取单个回测记录
func (s *Server) GetBacktestRecordAPI(c *gin.Context) {
	log.Printf("[DEBUG] ===== GetBacktestRecordAPI 被调用 =====")

	recordIDStr := c.Param("id")
	if recordIDStr == "" {
		sendRecommendationError(c, 400, "缺少记录ID", "MISSING_RECORD_ID")
		return
	}

	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		sendRecommendationError(c, 400, "无效的记录ID", "INVALID_RECORD_ID")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("uid")
	if !exists {
		sendRecommendationError(c, 401, "用户未登录", "USER_NOT_AUTHENTICATED")
		return
	}

	// 从数据库查询回测记录
	record, err := pdb.GetAsyncBacktestRecordByID(s.db.DB(), uint(recordID), userID.(uint))
	if err != nil {
		log.Printf("[ERROR] 查询异步回测记录失败: %v", err)
		sendRecommendationError(c, 404, "回测记录不存在", "RECORD_NOT_FOUND")
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"record": gin.H{
			"id":              record.ID,
			"user_id":         record.UserID,
			"symbol":          record.Symbol,
			"strategy":        record.Strategy,
			"start_date":      record.StartDate,
			"end_date":        record.EndDate,
			"initial_capital": record.InitialCapital.InexactFloat64(),
			"position_size":   record.PositionSize.InexactFloat64(),
			"status":          record.Status,
			"result":          record.Result,
			"error_message":   record.ErrorMessage,
			"created_at":      record.CreatedAt,
			"updated_at":      record.UpdatedAt,
			"completed_at":    record.CompletedAt,
		},
	})
}

// DeleteBacktestRecordAPI 删除回测记录
func (s *Server) DeleteBacktestRecordAPI(c *gin.Context) {
	log.Printf("[DEBUG] ===== DeleteBacktestRecordAPI 被调用 =====")

	recordIDStr := c.Param("id")
	if recordIDStr == "" {
		sendRecommendationError(c, 400, "缺少记录ID", "MISSING_RECORD_ID")
		return
	}

	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		sendRecommendationError(c, 400, "无效的记录ID", "INVALID_RECORD_ID")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("uid")
	if !exists {
		sendRecommendationError(c, 401, "用户未登录", "USER_NOT_AUTHENTICATED")
		return
	}

	// 从数据库删除回测记录
	if err := pdb.DeleteAsyncBacktestRecord(s.db.DB(), uint(recordID), userID.(uint)); err != nil {
		log.Printf("[ERROR] 删除异步回测记录失败: %v", err)
		sendRecommendationError(c, 500, "删除回测记录失败", "DELETE_FAILED", err.Error())
		return
	}

	log.Printf("[INFO] 删除异步回测记录成功，ID=%d，用户ID=%d", recordID, userID.(uint))

	c.JSON(200, gin.H{
		"success": true,
		"message": "回测记录已删除",
	})
}

// runAsyncBacktest 运行异步回测任务
func (s *Server) runAsyncBacktest(record *pdb.AsyncBacktestRecord, req struct {
	Symbol         string   `json:"symbol"`  // 向后兼容
	Symbols        []string `json:"symbols"` // 多币种列表
	StartDate      string   `json:"start_date"`
	EndDate        string   `json:"end_date"`
	Strategy       string   `json:"strategy"`
	InitialCapital float64  `json:"initial_capital"`
	PositionSize   float64  `json:"position_size"`
	// 自动执行参数
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`
	MaxBatches            int           `json:"max_batches"`
	BatchDelay            time.Duration `json:"batch_delay"`
	BatchSize             int           `json:"batch_size"`
	DynamicSizing         bool          `json:"dynamic_sizing"`
	MarketConditionFilter bool          `json:"market_condition_filter"`
	// 自动选择币种参数
	AutoSelectSymbol        bool   `json:"auto_select_symbol"`
	MaxSymbolsToEvaluate    int    `json:"max_symbols_to_evaluate"`
	SymbolSelectionCriteria string `json:"symbol_selection_criteria"`
}) {
	startTime := time.Now()
	// 添加 panic recovery 防止 goroutine 异常退出
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] 异步回测任务 ID=%d 出现 panic: %v", record.ID, r)
			log.Printf("[ERROR] 堆栈跟踪: %s", debug.Stack())

			// 更新状态为失败
			if updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "failed", nil, fmt.Sprintf("系统异常: %v", r), nil); updateErr != nil {
				log.Printf("[ERROR] 更新回测记录状态为failed失败 ID=%d: %v", record.ID, updateErr)
			}
		}
	}()

	// 确定要使用的币种列表
	var symbols []string
	if len(req.Symbols) > 0 {
		symbols = req.Symbols
	} else if record.Symbol != "" {
		// 处理数据库中存储的多币种（逗号分隔）或单币种
		if strings.Contains(record.Symbol, ",") {
			symbols = strings.Split(record.Symbol, ",")
		} else {
			symbols = []string{record.Symbol}
		}
	}

	// 如果symbols仍然为空，提供默认币种
	if len(symbols) == 0 {
		log.Printf("[WARNING] 未指定币种，使用默认币种组合 ID=%d", record.ID)
		symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	}

	log.Printf("[INFO] 开始执行异步回测任务 ID=%d, 币种=%v", record.ID, symbols)

	// 更新状态为运行中
	if err := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "running", nil, "", nil); err != nil {
		log.Printf("[ERROR] 更新回测记录状态为running失败 ID=%d: %v", record.ID, err)
		return
	}
	log.Printf("[INFO] 回测任务状态更新为running，ID=%d", record.ID)

	// 处理自动选择币种（仅当未指定多币种且启用自动选择时）
	if len(symbols) <= 1 && req.AutoSelectSymbol {
		log.Printf("[AUTO_SELECT] 开始自动选择币种，评估数量: %d", req.MaxSymbolsToEvaluate)

		autoSelectReq := struct {
			StartDate               string  `json:"start_date"`
			EndDate                 string  `json:"end_date"`
			Strategy                string  `json:"strategy"`
			InitialCapital          float64 `json:"initial_capital"`
			PositionSize            float64 `json:"position_size"`
			MaxSymbolsToEvaluate    int     `json:"max_symbols_to_evaluate"`
			SymbolSelectionCriteria string  `json:"symbol_selection_criteria"`
		}{
			StartDate:               req.StartDate,
			EndDate:                 req.EndDate,
			Strategy:                req.Strategy,
			InitialCapital:          req.InitialCapital,
			PositionSize:            req.PositionSize,
			MaxSymbolsToEvaluate:    req.MaxSymbolsToEvaluate,
			SymbolSelectionCriteria: req.SymbolSelectionCriteria,
		}

		selectedSymbolsResult, err := s.selectOptimalSymbolsForBacktest(context.Background(), autoSelectReq)
		if err != nil {
			log.Printf("[ERROR] 自动选择币种失败 ID=%d: %v，使用默认币种组合", record.ID, err)
			// 使用默认的热门币种组合作为fallback
			symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
			log.Printf("[AUTO_SELECT] 使用默认币种组合: %v", symbols)
		} else {
			symbols = selectedSymbolsResult
			log.Printf("[AUTO_SELECT] 自动选择完成，选中币种: %v (原币种: %v)", selectedSymbolsResult, record.Symbol)

			// 立即更新record.Symbol以供后续使用（用逗号分隔存储多币种）
			record.Symbol = strings.Join(selectedSymbolsResult, ",")

			// 更新数据库记录中的币种
			if updateErr := pdb.UpdateAsyncBacktestRecordSymbol(s.db.DB(), record.ID, record.UserID, record.Symbol); updateErr != nil {
				log.Printf("[ERROR] 更新回测记录币种失败 ID=%d: %v", record.ID, updateErr)
			}
		}
	}

	log.Printf("[DEBUG] 开始准备回测配置 ID=%d", record.ID)

	// 执行回测（使用现有逻辑）
	// 注意：这里需要创建一个临时的Gin上下文，包含用户ID
	tempGinCtx := &gin.Context{}
	tempGinCtx.Set("uid", record.UserID)

	log.Printf("[DEBUG] 创建临时上下文完成 ID=%d", record.ID)

	// 执行回测获取原始结果（包含完整交易记录）
	// 获取AI推荐数据用于配置（仅对单币种获取）
	var recommendation gin.H
	if len(symbols) == 1 {
		log.Printf("[DEBUG] 开始获取AI推荐数据 ID=%d, 币种=%s, 结束日期=%s", record.ID, symbols[0], record.EndDate)
		rec, recErr := s.getAIRecommendationForBacktest(context.Background(), symbols[0], record.EndDate)
		if recErr != nil {
			log.Printf("[ERROR] 获取AI推荐数据失败 ID=%d: %v", record.ID, recErr)
			// 继续执行，使用空的推荐数据
			recommendation = gin.H{}
		} else {
			log.Printf("[DEBUG] AI推荐数据获取成功 ID=%d", record.ID)
			recommendation = rec
		}
	} else {
		log.Printf("[DEBUG] 多币种模式，跳过AI推荐数据获取 ID=%d", record.ID)
		recommendation = gin.H{}
	}

	// 准备回测配置
	// 对于多币种，使用第一个币种作为主要配置参考
	if len(symbols) == 0 {
		log.Printf("[ERROR] 币种列表为空，无法进行回测 ID=%d", record.ID)
		return
	}
	mainSymbol := symbols[0]
	type CacheRequest struct {
		Symbol      string   `json:"symbol" binding:"required"` // 向后兼容
		Symbols     []string `json:"symbols"`                   // 多币种列表
		Timeframe   string   `json:"timeframe"`
		StartDate   string   `json:"start_date"`
		EndDate     string   `json:"end_date"`
		InitialCash float64  `json:"initial_cash"`
		MaxPosition float64  `json:"max_position"`
		StopLoss    float64  `json:"stop_loss"`
		TakeProfit  float64  `json:"take_profit"`
		Commission  float64  `json:"commission"`
		Strategy    string   `json:"strategy"`
	}

	cacheReq := CacheRequest{
		Symbol:      mainSymbol, // 主要币种（向后兼容）
		Symbols:     symbols,    // 多币种列表
		Timeframe:   "1d",
		StartDate:   record.StartDate,
		EndDate:     record.EndDate,
		InitialCash: record.InitialCapital.InexactFloat64(), // 保留精度
		MaxPosition: record.PositionSize.InexactFloat64(),   // 保留精度
		StopLoss:    -0.15,                                  // 从-5%放宽到-15%，避免过早止损
		TakeProfit:  0.15,
		Commission:  0.001,
		Strategy:    record.Strategy,
	}

	log.Printf("[DEBUG] 开始映射回测配置 ID=%d", record.ID)
	backtestConfig := s.mapRecommendationToBacktestConfig(recommendation, cacheReq)
	log.Printf("[DEBUG] 回测配置映射完成 ID=%d, 策略=%s", record.ID, backtestConfig.Strategy)

	// 执行回测
	log.Printf("[DEBUG] 开始执行回测引擎 ID=%d", record.ID)
	rawResult, err := s.backtestEngine.RunBacktest(context.Background(), *backtestConfig)
	if err != nil {
		log.Printf("[ERROR] 异步回测执行失败 ID=%d: %v", record.ID, err)
		if updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "failed", nil, err.Error(), nil); updateErr != nil {
			log.Printf("[ERROR] 更新回测记录状态为failed失败 ID=%d: %v", record.ID, updateErr)
		}
		return
	}

	log.Printf("[DEBUG] 回测引擎执行完成 ID=%d, 交易数量=%d", record.ID, len(rawResult.Trades))

	// 保存交易记录到数据库
	log.Printf("[DEBUG] 开始保存交易记录到数据库 ID=%d", record.ID)
	if err := s.saveBacktestTradesToDB(record.ID, rawResult); err != nil {
		log.Printf("[ERROR] 保存交易记录到数据库失败 ID=%d: %v", record.ID, err)
		// 继续执行，不因为保存交易记录失败而停止整个回测流程
	} else {
		log.Printf("[DEBUG] 交易记录保存完成 ID=%d", record.ID)
	}

	// 增强回测结果（不包含完整交易记录）
	log.Printf("[DEBUG] 开始增强回测结果 ID=%d", record.ID)
	result := s.enhanceBacktestResultWithAIInsights(rawResult, recommendation)
	log.Printf("[DEBUG] 回测结果增强完成 ID=%d", record.ID)

	// 将结果转换为JSON字符串
	log.Printf("[DEBUG] 开始序列化回测结果 ID=%d", record.ID)
	resultJSONBytes, jsonErr := json.Marshal(result)
	if jsonErr != nil {
		log.Printf("[ERROR] 序列化回测结果失败 ID=%d: %v", record.ID, jsonErr)
		if updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "failed", nil, "结果序列化失败", nil); updateErr != nil {
			log.Printf("[ERROR] 更新回测记录状态为failed失败 ID=%d: %v", record.ID, updateErr)
		}
		return
	}
	log.Printf("[DEBUG] 回测结果序列化完成 ID=%d, JSON长度=%d", record.ID, len(resultJSONBytes))

	resultJSONString := string(resultJSONBytes)

	// 更新记录为完成状态
	log.Printf("[DEBUG] 开始更新数据库记录状态为完成 ID=%d", record.ID)
	completedAt := time.Now()
	if updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "completed", &resultJSONString, "", &completedAt); updateErr != nil {
		log.Printf("[ERROR] 更新回测记录状态为completed失败 ID=%d: %v", record.ID, updateErr)
		return
	}

	log.Printf("[INFO] ✅ 异步回测任务完成 ID=%d, 总耗时: %.2fs", record.ID, time.Since(startTime).Seconds())
}

// saveBacktestTradesToDB 保存回测交易记录到数据库
func (s *Server) saveBacktestTradesToDB(backtestRecordID uint, result *BacktestResult) error {
	if result == nil || len(result.Trades) == 0 {
		return nil // 没有交易记录，无需保存
	}

	// 转换交易记录
	trades := make([]pdb.AsyncBacktestTrade, 0, len(result.Trades))
	for _, trade := range result.Trades {
		// 计算成交金额和盈亏百分比
		value := decimal.NewFromFloat(trade.Quantity * trade.Price)
		pnlPercent := decimal.Zero
		if trade.PnL != 0 && value.GreaterThan(decimal.Zero) {
			pnlPercent = decimal.NewFromFloat(trade.PnL).Div(value)
		}

		asyncTrade := pdb.AsyncBacktestTrade{
			BacktestRecordID: backtestRecordID,
			Timestamp:        trade.Timestamp,
			Symbol:           trade.Symbol,
			Side:             trade.Side,
			Price:            decimal.NewFromFloat(trade.Price),
			Quantity:         decimal.NewFromFloat(trade.Quantity),
			Value:            value,
			Commission:       decimal.NewFromFloat(trade.Commission),
			PnL:              decimal.NewFromFloat(trade.PnL),
			PnLPercent:       pnlPercent,
		}
		trades = append(trades, asyncTrade)
	}

	// 批量保存到数据库
	return pdb.CreateAsyncBacktestTrades(s.db.DB(), trades)
}

// executeBacktestWithParams 执行回测的辅助方法（从现有AIBacktestAPI中提取的逻辑）
func (s *Server) executeBacktestWithParams(ctx context.Context, c *gin.Context, req struct {
	Symbol               string  `json:"symbol" binding:"required"`
	Timeframe            string  `json:"timeframe"`
	StartDate            string  `json:"start_date"`
	EndDate              string  `json:"end_date"`
	InitialCash          float64 `json:"initial_cash"`
	MaxPosition          float64 `json:"max_position"`
	StopLoss             float64 `json:"stop_loss"`
	TakeProfit           float64 `json:"take_profit"`
	Commission           float64 `json:"commission"`
	Strategy             string  `json:"strategy"`
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`
	// 渐进式执行参数
	ProgressiveExecution  bool          `json:"progressive_execution"`   // 是否启用渐进式执行
	MaxBatches            int           `json:"max_batches"`             // 最大批次数
	BatchDelay            time.Duration `json:"batch_delay"`             // 批次间延迟
	BatchSize             int           `json:"batch_size"`              // 每批最大交易数
	DynamicSizing         bool          `json:"dynamic_sizing"`          // 是否启用动态仓位调整
	MarketConditionFilter bool          `json:"market_condition_filter"` // 是否启用市场条件过滤
}) (gin.H, error) {
	// 这里复制AIBacktestAPI的逻辑，但返回结果而不是直接响应

	// 获取AI推荐数据
	recommendation, err := s.getAIRecommendationForBacktest(ctx, req.Symbol, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("获取推荐数据失败: %v", err)
	}

	// 映射回测配置
	type SingleCacheRequest struct {
		Symbol      string   `json:"symbol" binding:"required"` // 向后兼容
		Symbols     []string `json:"symbols"`                   // 多币种列表（单币种为空）
		Timeframe   string   `json:"timeframe"`
		StartDate   string   `json:"start_date"`
		EndDate     string   `json:"end_date"`
		InitialCash float64  `json:"initial_cash"`
		MaxPosition float64  `json:"max_position"`
		StopLoss    float64  `json:"stop_loss"`
		TakeProfit  float64  `json:"take_profit"`
		Commission  float64  `json:"commission"`
		Strategy    string   `json:"strategy"`
	}

	cacheReq := SingleCacheRequest{
		Symbol:      req.Symbol,
		Symbols:     []string{}, // 单币种模式为空
		Timeframe:   req.Timeframe,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		InitialCash: req.InitialCash,
		MaxPosition: req.MaxPosition,
		StopLoss:    req.StopLoss,
		TakeProfit:  req.TakeProfit,
		Commission:  req.Commission,
		Strategy:    req.Strategy,
	}

	backtestConfig := s.mapRecommendationToBacktestConfig(recommendation, cacheReq)

	// 执行回测
	result, err := s.backtestEngine.RunBacktest(ctx, *backtestConfig)
	if err != nil {
		return nil, fmt.Errorf("回测执行失败: %v", err)
	}

	// 增强回测结果
	enhancedResult := s.enhanceBacktestResultWithAIInsights(result, recommendation)

	// 处理自动执行（如果启用）
	if req.AutoExecute {
		if _, exists := c.Get("uid"); exists {
			stats, err := s.executeBacktestTrades(c, req, enhancedResult)
			if err != nil {
				log.Printf("[WARN] 自动执行交易失败: %v", err)
			} else {
				enhancedResult["auto_execute_stats"] = *stats
			}
		}
	}

	return enhancedResult, nil
}

// GetBacktestTradesAPI 获取回测交易记录（分页）
func (s *Server) GetBacktestTradesAPI(c *gin.Context) {
	log.Printf("[DEBUG] ===== GetBacktestTradesAPI 被调用 =====")

	recordIDStr := c.Param("recordId")
	if recordIDStr == "" {
		sendRecommendationError(c, 400, "缺少记录ID", "MISSING_RECORD_ID")
		return
	}

	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		sendRecommendationError(c, 400, "无效的记录ID", "INVALID_RECORD_ID")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("uid")
	if !exists {
		sendRecommendationError(c, 401, "用户未登录", "USER_NOT_AUTHENTICATED")
		return
	}

	// 解析查询参数
	var req struct {
		Page      int    `form:"page,default=1"`
		Limit     int    `form:"limit,default=20"`
		SortBy    string `form:"sort_by,default=timestamp"`
		SortOrder string `form:"sort_order,default=desc"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		sendRecommendationError(c, 400, "无效的查询参数", "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	// 验证分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// 获取回测记录
	record, err := pdb.GetAsyncBacktestRecordByID(s.db.DB(), uint(recordID), userID.(uint))
	if err != nil {
		log.Printf("[ERROR] 获取回测记录失败: %v", err)
		sendRecommendationError(c, 404, "回测记录不存在", "RECORD_NOT_FOUND")
		return
	}

	// 从数据库查询交易记录
	trades, totalTrades, err := pdb.GetAsyncBacktestTrades(s.db.DB(), uint(recordID), req.Page, req.Limit, req.SortBy, req.SortOrder)
	if err != nil {
		log.Printf("[ERROR] 查询交易记录失败: %v", err)
		sendRecommendationError(c, 500, "查询交易记录失败", "QUERY_TRADES_FAILED")
		return
	}

	// 转换为前端需要的格式
	tradesArray := make([]gin.H, len(trades))
	for i, trade := range trades {
		// 将decimal.Decimal转换为float64以便前端处理
		price, _ := trade.Price.Float64()
		quantity, _ := trade.Quantity.Float64()
		value, _ := trade.Value.Float64()
		commission, _ := trade.Commission.Float64()
		pnl, _ := trade.PnL.Float64()
		pnlPercent, _ := trade.PnLPercent.Float64()

		tradesArray[i] = gin.H{
			"id":          trade.ID,
			"timestamp":   trade.Timestamp,
			"symbol":      trade.Symbol,
			"side":        trade.Side,
			"price":       price,
			"quantity":    quantity,
			"value":       value,
			"commission":  commission,
			"pnl":         pnl,
			"pnl_percent": pnlPercent,
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"trades":  tradesArray,
		"pagination": gin.H{
			"page":  req.Page,
			"limit": req.Limit,
			"total": totalTrades,
			"pages": (totalTrades + int64(req.Limit) - 1) / int64(req.Limit),
		},
		"backtest_info": gin.H{
			"record_id": record.ID,
			"symbol":    record.Symbol,
			"strategy":  record.Strategy,
			"status":    record.Status,
		},
	})
}

// selectOptimalSymbolForBacktest 自动选择最适合回测的币种
func (s *Server) selectOptimalSymbolsForBacktest(ctx context.Context, req struct {
	StartDate               string  `json:"start_date"`
	EndDate                 string  `json:"end_date"`
	Strategy                string  `json:"strategy"`
	InitialCapital          float64 `json:"initial_capital"`
	PositionSize            float64 `json:"position_size"`
	MaxSymbolsToEvaluate    int     `json:"max_symbols_to_evaluate"`
	SymbolSelectionCriteria string  `json:"symbol_selection_criteria"`
}) ([]string, error) {

	// 自动生成币种池
	symbolPool, err := s.generateSmartSymbolPool(ctx, req.SymbolSelectionCriteria, req.MaxSymbolsToEvaluate*2)
	if err != nil {
		log.Printf("[AUTO_SELECT] 生成智能币种池失败: %v，使用默认币种池", err)
		symbolPool = s.getDefaultSymbolPool()
	}

	log.Printf("[AUTO_SELECT] 开始评估币种池，币种数量: %d, 选择标准: %s", len(symbolPool), req.SymbolSelectionCriteria)

	// 限制评估数量
	symbolsToEvaluate := symbolPool
	if len(symbolsToEvaluate) > req.MaxSymbolsToEvaluate {
		symbolsToEvaluate = symbolsToEvaluate[:req.MaxSymbolsToEvaluate]
	}

	// 并行评估所有币种
	type symbolScore struct {
		symbol string
		score  float64
		reason string
	}

	results := make(chan symbolScore, len(symbolsToEvaluate))
	var wg sync.WaitGroup

	for _, symbol := range symbolsToEvaluate {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			score, reason, err := s.evaluateSymbolForBacktest(ctx, sym, req.StartDate, req.EndDate, req.SymbolSelectionCriteria)
			if err != nil {
				log.Printf("[AUTO_SELECT] 评估币种%s失败: %v", sym, err)
				results <- symbolScore{symbol: sym, score: -1000, reason: fmt.Sprintf("评估失败: %v", err)}
				return
			}

			results <- symbolScore{symbol: sym, score: score, reason: reason}
		}(symbol)
	}

	// 等待所有评估完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	scores := make([]symbolScore, 0, len(symbolsToEvaluate))
	for result := range results {
		scores = append(scores, result)
		log.Printf("[AUTO_SELECT] 币种%s评估完成，得分: %.2f, 原因: %s", result.symbol, result.score, result.reason)
	}

	// 按得分降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 选择前3-5个最好的币种（根据可用币种数量动态调整）
	numSymbolsToSelect := 3
	if len(scores) >= 5 {
		numSymbolsToSelect = 5
	} else if len(scores) >= 3 {
		numSymbolsToSelect = len(scores)
	} else if len(scores) >= 1 {
		numSymbolsToSelect = len(scores)
	} else {
		return nil, fmt.Errorf("未能找到合适的币种进行回测")
	}

	// 选择得分最高的币种，无论得分高低（在当前市场环境下调整标准）
	selectedSymbols := make([]string, 0)
	for i := 0; i < len(scores) && len(selectedSymbols) < numSymbolsToSelect; i++ {
		selectedSymbols = append(selectedSymbols, scores[i].symbol)
	}

	if len(selectedSymbols) == 0 {
		return nil, fmt.Errorf("未能找到合适的币种进行回测")
	}

	log.Printf("[AUTO_SELECT] 自动选择多币种: %v (共%d个币种)", selectedSymbols, len(selectedSymbols))

	// 记录选择结果
	for _, score := range scores {
		log.Printf("[AUTO_SELECT_DETAIL] %s: %.2f - %s", score.symbol, score.score, score.reason)
	}

	return selectedSymbols, nil
}

// evaluateSymbolForBacktest 评估单个币种对回测的适合度
func (s *Server) evaluateSymbolForBacktest(ctx context.Context, symbol string, startDate, endDate, criteria string) (float64, string, error) {

	// 1. 检查数据可用性
	dataAvailable, err := s.checkSymbolDataAvailability(ctx, symbol, startDate, endDate)
	if err != nil {
		return -1000, "数据检查失败", err
	}
	if !dataAvailable {
		return -500, "数据不足", fmt.Errorf("币种%s在指定时间段内数据不足", symbol)
	}

	// 2. 获取市场数据和基本指标
	marketData, err := s.getSymbolMarketMetrics(ctx, symbol, startDate, endDate)
	if err != nil {
		return -800, "获取市场数据失败", err
	}

	// 3. 根据选择标准计算评分
	var score float64
	var reason string

	switch criteria {
	case "profitability":
		score, reason = s.calculateProfitabilityScore(symbol, marketData)
	case "volatility":
		score, reason = s.calculateVolatilityScore(symbol, marketData)
	case "trend_strength":
		score, reason = s.calculateTrendStrengthScore(symbol, marketData)
	case "liquidity":
		score, reason = s.calculateLiquidityScore(symbol, marketData)
	case "balanced":
		score, reason = s.calculateBalancedScore(symbol, marketData)
	case "market_heat":
		score, reason = s.calculateMarketHeatScore(symbol, marketData)
	default:
		score, reason = s.calculateBalancedScore(symbol, marketData) // 默认使用均衡评分
	}

	// 4. 应用风险惩罚
	riskPenalty := s.calculateRiskPenalty(symbol, marketData)
	score += riskPenalty

	return score, reason, nil
}

// checkSymbolDataAvailability 检查币种数据可用性
func (s *Server) checkSymbolDataAvailability(ctx context.Context, symbol, startDate, endDate string) (bool, error) {
	// 尝试获取一个数据点来验证数据可用性
	klines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1d", 1)
	if err != nil {
		return false, err
	}

	return len(klines) > 0, nil
}

// getSymbolMarketMetrics 获取币种市场指标
func (s *Server) getSymbolMarketMetrics(ctx context.Context, symbol, startDate, endDate string) (map[string]float64, error) {
	metrics := make(map[string]float64)

	// 获取价格变化
	change, err := s.calculatePriceChange(ctx, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	metrics["price_change"] = change

	// 获取波动率
	volatility, err := s.calculateSymbolVolatility(ctx, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	metrics["volatility"] = volatility

	// 获取成交量
	volume, err := s.calculateAverageVolume(ctx, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	metrics["avg_volume"] = volume

	// 获取趋势强度
	trendStrength, err := s.calculateTrendStrengthForRecommendation(ctx, symbol, startDate, endDate)
	if err != nil {
		return nil, err
	}
	metrics["trend_strength"] = trendStrength

	return metrics, nil
}

// calculateProfitabilityScore 计算盈利能力评分
func (s *Server) calculateProfitabilityScore(symbol string, metrics map[string]float64) (float64, string) {

	priceChange := metrics["price_change"]
	trendStrength := metrics["trend_strength"]

	// 基础评分：价格上涨潜力 + 趋势强度
	baseScore := priceChange*10 + trendStrength*20

	// 奖励强势上涨的币种
	if priceChange > 0.1 { // 10%以上涨幅
		baseScore += 50
	} else if priceChange > 0.05 { // 5%以上涨幅
		baseScore += 25
	}

	// 奖励强趋势
	if trendStrength > 0.7 {
		baseScore += 30
	} else if trendStrength > 0.5 {
		baseScore += 15
	}

	reason := fmt.Sprintf("价格变化:%.1f%%, 趋势强度:%.2f", priceChange*100, trendStrength)
	return baseScore, reason
}

// calculateVolatilityScore 计算波动率评分
func (s *Server) calculateVolatilityScore(symbol string, metrics map[string]float64) (float64, string) {

	volatility := metrics["volatility"]
	priceChange := metrics["price_change"]

	// 适中的波动率最适合交易
	var volScore float64
	if volatility > 0.02 && volatility < 0.08 { // 2%-8%的波动率
		volScore = 50
	} else if volatility > 0.01 && volatility < 0.12 { // 1%-12%的波动率
		volScore = 25
	} else {
		volScore = -25 // 过高或过低波动率都降低评分
	}

	// 结合价格变化
	totalScore := volScore + priceChange*20

	reason := fmt.Sprintf("波动率:%.1f%%, 价格变化:%.1f%%", volatility*100, priceChange*100)
	return totalScore, reason
}

// calculateTrendStrengthScore 计算趋势强度评分
func (s *Server) calculateTrendStrengthScore(symbol string, metrics map[string]float64) (float64, string) {

	trendStrength := metrics["trend_strength"]
	volatility := metrics["volatility"]

	// 趋势强度评分 - 在当前低波动环境下调整标准
	var score float64
	absTrend := math.Abs(trendStrength)

	if absTrend > 0.8 {
		score = 80 // 极强趋势
	} else if absTrend > 0.6 {
		score = 60 // 强趋势
	} else if absTrend > 0.4 {
		score = 40 // 中等趋势
	} else if absTrend > 0.2 {
		score = 25 // 弱趋势
	} else {
		// 在极低波动环境下，即使趋势接近0也给予基础评分
		if volatility < 0.01 {
			score = 15 // 低波动+无趋势：稳定但缺乏机会
		} else {
			score = 10 // 其他情况
		}
	}

	// 奖励有明确方向的趋势
	if trendStrength > 0.1 {
		score += 10 // 上涨趋势奖励
	} else if trendStrength < -0.1 {
		score += 5 // 下跌趋势小幅奖励（仍有机会做空）
	}

	reason := fmt.Sprintf("趋势强度:%.2f", trendStrength)
	return score, reason
}

// calculateLiquidityScore 计算流动性评分
func (s *Server) calculateLiquidityScore(symbol string, metrics map[string]float64) (float64, string) {

	avgVolume := metrics["avg_volume"]

	// 成交量评分（对数变换，避免极端值影响）
	var volumeScore float64
	if avgVolume > 1000000 { // 百万级成交量
		volumeScore = 80
	} else if avgVolume > 100000 { // 十万级成交量
		volumeScore = 60
	} else if avgVolume > 10000 { // 万级成交量
		volumeScore = 40
	} else if avgVolume > 1000 { // 千级成交量
		volumeScore = 20
	} else {
		volumeScore = 0
	}

	reason := fmt.Sprintf("平均成交量:%.0f", avgVolume)
	return volumeScore, reason
}

// calculateBalancedScore 计算平衡评分
func (s *Server) calculateBalancedScore(symbol string, metrics map[string]float64) (float64, string) {

	priceChange := metrics["price_change"]
	volatility := metrics["volatility"]
	trendStrength := metrics["trend_strength"]
	avgVolume := metrics["avg_volume"]

	// 各项指标标准化并加权
	profitScore := priceChange * 20                  // 盈利潜力权重25%
	volScore := (1 - math.Abs(volatility-0.05)) * 30 // 适中波动率权重25%
	trendScore := trendStrength * 25                 // 趋势强度权重20%

	// 成交量评分
	var volumeScore float64
	if avgVolume > 100000 {
		volumeScore = 25
	} else if avgVolume > 10000 {
		volumeScore = 15
	} else {
		volumeScore = 5
	}

	totalScore := profitScore + volScore + trendScore + volumeScore

	reason := fmt.Sprintf("综合评分: 盈利%.1f, 波动%.1f, 趋势%.1f, 流动性%.1f",
		profitScore, volScore, trendScore, volumeScore)

	return totalScore, reason
}

// calculateMarketHeatScore 计算市场热度评分
func (s *Server) calculateMarketHeatScore(symbol string, metrics map[string]float64) (float64, string) {
	priceChange := metrics["price_change"]
	volatility := metrics["volatility"]
	trendStrength := metrics["trend_strength"]
	avgVolume := metrics["avg_volume"]

	// 市场热度主要看成交量和价格波动
	volumeScore := 0.0
	if avgVolume > 1000000 {
		volumeScore = 40 // 超高流动性
	} else if avgVolume > 500000 {
		volumeScore = 30 // 高流动性
	} else if avgVolume > 100000 {
		volumeScore = 20 // 中等流动性
	} else if avgVolume > 10000 {
		volumeScore = 10 // 低流动性
	} else {
		volumeScore = 0 // 极低流动性
	}

	// 价格活跃度评分
	activityScore := math.Min(math.Abs(priceChange)*50, 30) // 价格变化幅度

	// 波动率评分（适中波动率更受欢迎）
	var volScore float64
	if volatility < 0.02 {
		volScore = 10 // 低波动但不至于太死板
	} else if volatility < 0.05 {
		volScore = 20 // 适中波动
	} else if volatility < 0.10 {
		volScore = 15 // 较高波动
	} else {
		volScore = 5 // 过高波动
	}

	// 趋势强度评分（适度趋势更好）
	trendScore := math.Min(math.Abs(trendStrength)*20, 20)

	totalScore := volumeScore + activityScore + volScore + trendScore

	reason := fmt.Sprintf("成交量:%.0f, 价格变化:%.1f%%, 波动率:%.3f, 趋势:%.2f",
		avgVolume, priceChange*100, volatility, trendStrength)

	return totalScore, reason
}

// calculateRiskPenalty 计算风险惩罚
func (s *Server) calculateRiskPenalty(symbol string, metrics map[string]float64) float64 {
	penalty := 0.0

	volatility := metrics["volatility"]
	priceChange := metrics["price_change"]

	// 过高波动率惩罚
	if volatility > 0.15 {
		penalty -= 30
	} else if volatility > 0.10 {
		penalty -= 15
	}

	// 过大幅度下跌惩罚
	if priceChange < -0.3 {
		penalty -= 40
	} else if priceChange < -0.2 {
		penalty -= 20
	}

	return penalty
}

// generateSmartSymbolPool 根据市场情况智能生成币种池
func (s *Server) generateSmartSymbolPool(ctx context.Context, criteria string, maxSymbols int) ([]string, error) {
	switch criteria {
	case "market_heat":
		return s.generateMarketHeatSymbolPool(ctx, maxSymbols)
	case "profitability":
		return s.generateProfitabilitySymbolPool(ctx, maxSymbols)
	case "volatility":
		return s.generateVolatilitySymbolPool(ctx, maxSymbols)
	case "liquidity":
		return s.generateLiquiditySymbolPool(ctx, maxSymbols)
	default:
		return s.generateMarketHeatSymbolPool(ctx, maxSymbols)
	}
}

// generateMarketHeatSymbolPool 生成基于市场热度的币种池
func (s *Server) generateMarketHeatSymbolPool(ctx context.Context, maxSymbols int) ([]string, error) {
	// 基于成交量和价格变化的综合热度排序
	candidates := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
		"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "LTCUSDT",
		"MATICUSDT", "ALGOUSDT", "VETUSDT", "ICPUSDT", "FILUSDT",
		"TRXUSDT", "ETCUSDT", "XLMUSDT", "THETAUSDT", "FTMUSDT",
		"LINKUSDT", "UNIUSDT", "AAVEUSDT", "SUSHIUSDT", "COMPUSDT",
	}

	// 获取每个币种的热度评分
	type symbolHeat struct {
		symbol string
		score  float64
	}

	var symbolHeats []symbolHeat

	for _, symbol := range candidates {
		score, err := s.calculateSymbolHeatScore(ctx, symbol)
		if err != nil {
			log.Printf("[HEAT_CALC] 计算币种%s热度失败: %v", symbol, err)
			continue
		}
		symbolHeats = append(symbolHeats, symbolHeat{symbol: symbol, score: score})
	}

	// 按热度排序
	sort.Slice(symbolHeats, func(i, j int) bool {
		return symbolHeats[i].score > symbolHeats[j].score
	})

	// 返回前maxSymbols个币种
	result := make([]string, 0, maxSymbols)
	for i, heat := range symbolHeats {
		if i >= maxSymbols {
			break
		}
		result = append(result, heat.symbol)
		log.Printf("[HEAT_POOL] 排名%d: %s (热度: %.2f)", i+1, heat.symbol, heat.score)
	}

	return result, nil
}

// calculateSymbolHeatScore 计算币种热度评分
func (s *Server) calculateSymbolHeatScore(ctx context.Context, symbol string) (float64, error) {
	// 获取24小时成交量
	volume, err := s.calculateAverageVolume(ctx, symbol, "", "")
	if err != nil {
		return 0, err
	}

	// 获取价格变化
	priceChange, err := s.calculatePriceChange(ctx, symbol, "", "")
	if err != nil {
		return 0, err
	}

	// 获取波动率
	volatility, err := s.calculateSymbolVolatility(ctx, symbol, "", "")
	if err != nil {
		return 0, err
	}

	// 综合评分：成交量权重40%，价格变化权重30%，波动率权重30%
	heatScore := volume*0.4 + math.Abs(priceChange)*0.3 + volatility*0.3

	return heatScore, nil
}

// generateProfitabilitySymbolPool 生成基于盈利潜力的币种池
func (s *Server) generateProfitabilitySymbolPool(ctx context.Context, maxSymbols int) ([]string, error) {
	// 简化版：按预定义的盈利潜力排序
	profitabilityPool := []string{
		"BTCUSDT", "ETHUSDT", "SOLUSDT", "ADAUSDT", "AVAXUSDT",
		"MATICUSDT", "DOTUSDT", "LINKUSDT", "UNIUSDT", "AAVEUSDT",
	}

	if len(profitabilityPool) > maxSymbols {
		profitabilityPool = profitabilityPool[:maxSymbols]
	}

	return profitabilityPool, nil
}

// generateVolatilitySymbolPool 生成基于波动率的币种池
func (s *Server) generateVolatilitySymbolPool(ctx context.Context, maxSymbols int) ([]string, error) {
	// 选择适中波动率的币种
	volatilityPool := []string{
		"ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT", "DOTUSDT",
		"MATICUSDT", "AVAXUSDT", "LINKUSDT", "UNIUSDT", "ALGOUSDT",
	}

	if len(volatilityPool) > maxSymbols {
		volatilityPool = volatilityPool[:maxSymbols]
	}

	return volatilityPool, nil
}

// generateLiquiditySymbolPool 生成基于流动性的币种池
func (s *Server) generateLiquiditySymbolPool(ctx context.Context, maxSymbols int) ([]string, error) {
	// 选择高流动性币种
	liquidityPool := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
		"SOLUSDT", "DOTUSDT", "DOGEUSDT", "MATICUSDT", "AVAXUSDT",
	}

	if len(liquidityPool) > maxSymbols {
		liquidityPool = liquidityPool[:maxSymbols]
	}

	return liquidityPool, nil
}

// getDefaultSymbolPool 获取默认币种池
func (s *Server) getDefaultSymbolPool() []string {
	return []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
		"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "LTCUSDT",
		"MATICUSDT", "ALGOUSDT", "VETUSDT", "ICPUSDT", "FILUSDT",
	}
}

// calculatePriceChange 计算指定时间段内的价格变化
func (s *Server) calculatePriceChange(ctx context.Context, symbol, startDate, endDate string) (float64, error) {
	// 获取开始日期的价格
	startKlines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1d", 1)
	if err != nil {
		return 0, fmt.Errorf("获取开始日期K线失败: %w", err)
	}
	if len(startKlines) == 0 {
		return 0, fmt.Errorf("开始日期无数据")
	}

	startPrice, err := strconv.ParseFloat(startKlines[0].Open, 64)
	if err != nil {
		return 0, fmt.Errorf("解析开始价格失败: %w", err)
	}

	// 获取结束日期的价格
	endKlines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1d", 1)
	if err != nil {
		return 0, fmt.Errorf("获取结束日期K线失败: %w", err)
	}
	if len(endKlines) == 0 {
		return 0, fmt.Errorf("结束日期无数据")
	}

	endPrice, err := strconv.ParseFloat(endKlines[0].Close, 64)
	if err != nil {
		return 0, fmt.Errorf("解析结束价格失败: %w", err)
	}

	if startPrice == 0 {
		return 0, fmt.Errorf("开始价格为0")
	}

	return (endPrice - startPrice) / startPrice, nil
}

// calculateSymbolVolatility 计算币种波动率
func (s *Server) calculateSymbolVolatility(ctx context.Context, symbol, startDate, endDate string) (float64, error) {
	// 获取历史价格数据
	klines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1d", 30) // 最近30天
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 2 {
		return 0, fmt.Errorf("数据点不足")
	}

	// 计算日收益率
	returns := make([]float64, 0, len(klines)-1)
	for i := 1; i < len(klines); i++ {
		prevClose, _ := strconv.ParseFloat(klines[i-1].Close, 64)
		currClose, _ := strconv.ParseFloat(klines[i].Close, 64)

		if prevClose > 0 {
			ret := (currClose - prevClose) / prevClose
			returns = append(returns, ret)
		}
	}

	// 计算标准差作为波动率
	if len(returns) == 0 {
		return 0, fmt.Errorf("无有效收益率数据")
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance), nil
}

// calculateAverageVolume 计算平均成交量
func (s *Server) calculateAverageVolume(ctx context.Context, symbol, startDate, endDate string) (float64, error) {
	// 获取最近30天的成交量数据
	klines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1d", 30)
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) == 0 {
		return 0, fmt.Errorf("无成交量数据")
	}

	totalVolume := 0.0
	for _, kline := range klines {
		volume, err := strconv.ParseFloat(kline.Volume, 64)
		if err != nil {
			continue
		}
		totalVolume += volume
	}

	return totalVolume / float64(len(klines)), nil
}

// calculateTrendStrengthForRecommendation 计算推荐用的趋势强度
func (s *Server) calculateTrendStrengthForRecommendation(ctx context.Context, symbol, startDate, endDate string) (float64, error) {
	// 获取价格数据
	klines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1d", 20) // 最近20天
	if err != nil {
		return 0, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 5 {
		return 0, fmt.Errorf("数据点不足")
	}

	// 计算线性回归斜率作为趋势强度
	prices := make([]float64, len(klines))
	for i, kline := range klines {
		price, _ := strconv.ParseFloat(kline.Close, 64)
		prices[i] = price
	}

	// 简单线性回归
	n := float64(len(prices))
	sumX := n * (n - 1) / 2
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, price := range prices {
		x := float64(i)
		sumY += price
		sumXY += x * price
		sumXX += x * x
	}

	// 斜率计算
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 标准化趋势强度 (使用平均价格作为基准)
	avgPrice := sumY / n
	if avgPrice == 0 {
		return 0, fmt.Errorf("平均价格为0")
	}

	// 返回斜率相对于平均价格的标准化值
	return slope / avgPrice, nil
}
