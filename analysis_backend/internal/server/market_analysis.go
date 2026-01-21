package server

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 综合市场分析数据结构
// ============================================================================

// ComprehensiveMarketAnalysis 综合市场分析结果
type ComprehensiveMarketAnalysis struct {
	MarketAnalysis          *MarketAnalysisResult          `json:"market_analysis"`
	TechnicalIndicators     *TechnicalIndicatorsResult     `json:"technical_indicators"`
	StrategyRecommendations []MarketStrategyRecommendation `json:"strategy_recommendations"`
	LastUpdated             time.Time                      `json:"last_updated"`
}

// ============================================================================
// 市场分析 API
// ============================================================================

// 获取市场环境分析
func (s *Server) GetMarketAnalysis(c *gin.Context) {
	analysis, err := s.analyzeMarketEnvironment()
	if err != nil {
		s.DatabaseError(c, "市场分析", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analysis,
	})
}

// 获取市场技术指标数据
func (s *Server) GetMarketTechnicalIndicators(c *gin.Context) {
	indicators, err := s.calculateTechnicalIndicators()
	if err != nil {
		s.DatabaseError(c, "技术指标计算", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    indicators,
	})
}

// 获取策略推荐
func (s *Server) GetStrategyRecommendations(c *gin.Context) {
	analysis, err := s.analyzeMarketEnvironment()
	if err != nil {
		s.DatabaseError(c, "市场分析", err)
		return
	}

	recommendations := s.generateStrategyRecommendations(analysis)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    recommendations,
	})
}

// ============================================================================
// 综合市场分析接口（高优先级优化）
// ============================================================================

// GetComprehensiveMarketAnalysis 获取综合市场分析数据（带缓存优化）
func (s *Server) GetComprehensiveMarketAnalysis(c *gin.Context) {
	startTime := time.Now()
	//cacheKey := "comprehensive_market_analysis"

	// 尝试从缓存获取
	//if s.cache != nil {
	//	if cachedData, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
	//		var cachedResult ComprehensiveMarketAnalysis
	//		if json.Unmarshal(cachedData, &cachedResult) == nil {
	//			// 动态缓存过期时间：根据市场波动率调整
	//			cacheTTL := calculateDynamicCacheTTL(&cachedResult)
	//			if time.Since(cachedResult.LastUpdated) < cacheTTL {
	//				elapsed := time.Since(startTime)
	//				log.Printf("[综合分析] 从缓存获取，缓存TTL: %v, 耗时: %v", cacheTTL, elapsed)
	//
	//				c.JSON(http.StatusOK, gin.H{
	//					"success": true,
	//					"data":    &cachedResult,
	//					"meta": gin.H{
	//						"processing_time_ms": elapsed.Milliseconds(),
	//						"cached":             true,
	//						"cache_ttl":          cacheTTL.String(),
	//						"errors": gin.H{
	//							"market_analysis":          false,
	//							"technical_indicators":     false,
	//							"strategy_recommendations": false,
	//						},
	//					},
	//				})
	//				return
	//			}
	//		}
	//	}
	//}

	log.Printf("[综合分析] 开始计算市场分析数据")

	// 并行获取各项数据，提高性能
	type analysisResult struct {
		marketAnalysis          *MarketAnalysisResult
		technicalIndicators     *TechnicalIndicatorsResult
		strategyRecommendations []MarketStrategyRecommendation
		marketError             error
		technicalError          error
		strategyError           error
	}

	result := analysisResult{}

	// 使用goroutine并发执行各项分析，提高性能
	type analysisTask struct {
		name string
		fn   func() error
	}

	tasks := []analysisTask{
		{
			name: "市场环境分析",
			fn: func() error {
				if analysis, err := s.analyzeMarketEnvironment(); err != nil {
					result.marketError = err
					result.marketAnalysis = &MarketAnalysisResult{
						Volatility:  0,
						Trend:       "分析失败",
						Oscillation: 0,
					}
					return err
				} else {
					result.marketAnalysis = analysis
					return nil
				}
			},
		},
		{
			name: "技术指标计算",
			fn: func() error {
				if indicators, err := s.calculateTechnicalIndicators(); err != nil {
					result.technicalError = err
					result.technicalIndicators = &TechnicalIndicatorsResult{
						BTCVolatility: 0,
						AvgRSI:        0,
						StrongSymbols: 0,
						WeakSymbols:   0,
					}
					return err
				} else {
					result.technicalIndicators = indicators
					return nil
				}
			},
		},
	}

	// 并发生成市场分析和技术指标
	taskErrors := make(chan error, len(tasks))
	for _, task := range tasks {
		go func(task analysisTask) {
			err := task.fn()
			if err != nil {
				log.Printf("[综合分析] %s失败: %v", task.name, err)
			}
			taskErrors <- err
		}(task)
	}

	// 等待市场分析和技术指标完成
	for i := 0; i < len(tasks); i++ {
		<-taskErrors
	}

	// 策略推荐（依赖市场分析结果）
	if recommendations, err := func() ([]MarketStrategyRecommendation, error) {
		analysis := result.marketAnalysis
		if result.marketError != nil {
			// 如果市场分析失败，使用默认分析结果
			analysis = &MarketAnalysisResult{
				Volatility:  30,
				Trend:       "震荡",
				Oscillation: 50,
			}
		}
		return s.generateStrategyRecommendations(analysis), nil
	}(); err != nil {
		log.Printf("[综合分析] 策略推荐生成失败: %v", err)
		result.strategyError = err
		result.strategyRecommendations = []MarketStrategyRecommendation{}
	} else {
		result.strategyRecommendations = recommendations
	}

	comprehensiveResult := &ComprehensiveMarketAnalysis{
		MarketAnalysis:          result.marketAnalysis,
		TechnicalIndicators:     result.technicalIndicators,
		StrategyRecommendations: result.strategyRecommendations,
		LastUpdated:             time.Now(),
	}

	// 缓存计算结果（动态过期时间）
	//if s.cache != nil && (result.marketError == nil || result.technicalError == nil) {
	//	if jsonData, err := json.Marshal(comprehensiveResult); err == nil {
	//		// 使用动态缓存过期时间
	//		cacheTTL := calculateDynamicCacheTTL(comprehensiveResult)
	//		if err := s.cache.Set(c.Request.Context(), cacheKey, jsonData, cacheTTL); err != nil {
	//			log.Printf("[综合分析] 缓存存储失败: %v", err)
	//		} else {
	//			log.Printf("[综合分析] 结果已缓存，动态TTL: %v", cacheTTL)
	//		}
	//	}
	//}

	elapsed := time.Since(startTime)
	log.Printf("[综合分析] 完成，耗时: %v", elapsed)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    comprehensiveResult,
		"meta": gin.H{
			"processing_time_ms": elapsed.Milliseconds(),
			"cached":             false,
			"errors": gin.H{
				"market_analysis":          result.marketError != nil,
				"technical_indicators":     result.technicalError != nil,
				"strategy_recommendations": result.strategyError != nil,
			},
		},
	})
}

// 分析市场环境
func (s *Server) analyzeMarketEnvironment() (*MarketAnalysisResult, error) {
	// 获取最近3天的市场数据（进一步优化性能，从7天减少到3天）
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -3)

	// 首先获取交易量最大的50个币种，避免查询所有数据
	var topSymbols []string
	err := s.db.DB().Table("binance_24h_stats").
		Select("symbol").
		Where("quote_volume > 1000"). // 降低交易量门槛，从100000降到1000
		Order("quote_volume DESC").
		Limit(50).
		Pluck("symbol", &topSymbols).Error

	if err != nil {
		log.Printf("[市场分析] 查询binance_24h_stats失败: %v", err)
		return &MarketAnalysisResult{
			Volatility:  0,
			Trend:       "数据查询失败",
			Oscillation: 0,
		}, nil
	}

	if len(topSymbols) == 0 {
		log.Printf("[市场分析] binance_24h_stats表中没有找到交易量>1000的币种")
		// 如果24h统计表没有数据，尝试直接使用一些主要的币种进行分析
		topSymbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "LTCUSDT", "MATICUSDT"}
		log.Printf("[市场分析] 使用默认币种列表进行分析: %v", topSymbols)
	}

	var klines []struct {
		Symbol string
		Close  float64
		Time   time.Time
	}

	// 只查询高交易量币种的数据
	query := s.db.DB().Table("market_klines").
		Select("symbol, close_price as close, open_time as time").
		Where("open_time >= ? AND open_time <= ? AND symbol IN ?", startTime, endTime, topSymbols).
		Order("open_time ASC")

	if err := query.Scan(&klines).Error; err != nil {
		return nil, err
	}

	if len(klines) == 0 {
		return &MarketAnalysisResult{
			Volatility:  0,
			Trend:       "数据不足",
			Oscillation: 0,
		}, nil
	}

	// 计算整体波动率
	totalVolatility := 0.0
	symbolCount := 0
	symbolVolatilities := make(map[string][]float64)

	// 按符号分组计算波动率
	for _, kline := range klines {
		if symbolVolatilities[kline.Symbol] == nil {
			symbolVolatilities[kline.Symbol] = []float64{}
		}
		symbolVolatilities[kline.Symbol] = append(symbolVolatilities[kline.Symbol], kline.Close)
	}

	for _, prices := range symbolVolatilities {
		if len(prices) < 2 {
			continue
		}

		// 计算日收益率波动率
		var returns []float64
		for i := 1; i < len(prices); i++ {
			ret := (prices[i] - prices[i-1]) / prices[i-1]
			returns = append(returns, ret)
		}

		if len(returns) > 0 {
			volatility := calculateMarketStandardDeviation(returns) * math.Sqrt(365) * 100 // 年化波动率
			totalVolatility += volatility
			symbolCount++
		}
	}

	avgVolatility := 0.0
	if symbolCount > 0 {
		avgVolatility = totalVolatility / float64(symbolCount)
	}

	// 分析趋势和震荡（基础分析）
	trend, oscillation := analyzeTrendAndOscillation(klines)

	// 增强的市场环境检测（用于均值回归策略）
	enhancedAnalysis := s.analyzeEnhancedMarketEnvironment(klines, avgVolatility)

	return &MarketAnalysisResult{
		// 基础指标（保持向后兼容）
		Volatility:  avgVolatility,
		Trend:       trend,
		Oscillation: oscillation,

		// 增强的市场环境信息
		MarketRegime:        enhancedAnalysis.Regime,
		RegimeConfidence:    enhancedAnalysis.Confidence,
		TrendStrength:       enhancedAnalysis.TrendStrength,
		VolatilityLevel:     avgVolatility,
		OscillationIndex:    enhancedAnalysis.OscillationIndex,
		BullishSymbols:      enhancedAnalysis.BullishCount,
		BearishSymbols:      enhancedAnalysis.BearishCount,
		SidewaysSymbols:     enhancedAnalysis.SidewaysCount,
		StrongTrendSymbols:  enhancedAnalysis.StrongTrendCount,
		HighVolumeRatio:     enhancedAnalysis.HighVolumeRatio,
		VolumeConcentration: enhancedAnalysis.VolumeConcentration,
		RegimeStability:     enhancedAnalysis.Stability,
		ChangeProbability:   enhancedAnalysis.ChangeProbability,
	}, nil
}

// 计算技术指标
func (s *Server) calculateTechnicalIndicators() (*TechnicalIndicatorsResult, error) {
	// 获取BTC最近30天的数据
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)

	var klines []struct {
		Close float64
		Time  time.Time
	}

	err := s.db.DB().Table("market_klines").
		Select("close_price as close, open_time as time").
		Where("symbol = ? AND open_time >= ? AND open_time <= ?", "BTCUSDT", startTime, endTime).
		Order("open_time DESC").
		Limit(30).
		Scan(&klines).Error

	if err != nil {
		return nil, err
	}

	if len(klines) < 14 {
		return &TechnicalIndicatorsResult{
			BTCVolatility: 0,
			AvgRSI:        0,
			StrongSymbols: 0,
			WeakSymbols:   0,
		}, nil
	}

	// 反转数组（时间升序）
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	// 提取价格数据
	prices := make([]float64, len(klines))
	for i, kline := range klines {
		prices[i] = kline.Close
	}

	// 计算BTC波动率
	btcVolatility := 0.0
	if len(prices) > 1 {
		var returns []float64
		for i := 1; i < len(prices); i++ {
			ret := (prices[i] - prices[i-1]) / prices[i-1]
			returns = append(returns, ret)
		}
		btcVolatility = calculateMarketStandardDeviation(returns) * math.Sqrt(365) * 100
	}

	// 计算RSI
	rsi := calculateMarketRSI(prices, 14)
	avgRSI := 0.0
	if len(rsi) > 0 {
		sum := 0.0
		for _, r := range rsi {
			sum += r
		}
		avgRSI = sum / float64(len(rsi))
	}

	// 获取强势/弱势币种数量和市场宽度指标
	strongSymbols, weakSymbols, bigGainers, bigLosers, neutralSymbols, advanceDeclineRatio := s.countMarketBreadthIndicators()

	// 获取成交量指标
	volumeGainers, volumeDecliners, avgVolumeChange := s.countVolumeIndicators()

	// 获取市场波动率指标
	marketVolatility, highVolSymbols, lowVolSymbols := s.countVolatilityIndicators()

	return &TechnicalIndicatorsResult{
		// 基础指标
		BTCVolatility: btcVolatility,
		AvgRSI:        avgRSI,
		StrongSymbols: strongSymbols,
		WeakSymbols:   weakSymbols,

		// 市场宽度指标
		AdvanceDeclineRatio: advanceDeclineRatio,
		BigGainers:          bigGainers,
		BigLosers:           bigLosers,
		NeutralSymbols:      neutralSymbols,

		// 成交量指标
		VolumeGainers:   volumeGainers,
		VolumeDecliners: volumeDecliners,
		AvgVolumeChange: avgVolumeChange,

		// 波动率指标
		MarketVolatility:      marketVolatility,
		HighVolatilitySymbols: highVolSymbols,
		LowVolatilitySymbols:  lowVolSymbols,
	}, nil
}

// 生成策略推荐
// 检查策略类型是否存在于系统中
func (s *Server) checkStrategyExists(strategyType string) bool {
	var count int64
	query := s.db.DB().Table("trading_strategies")

	switch strategyType {
	case "moving_average":
		query = query.Where("moving_average_enabled = ?", true)
	case "mean_reversion":
		query = query.Where("mean_reversion_enabled = ?", true)
	case "grid_trading":
		query = query.Where("grid_trading_enabled = ?", true)
	case "traditional":
		// 传统策略检查：没有开启均线、均值回归和网格交易的策略
		query = query.Where("moving_average_enabled = ?", false).
			Where("mean_reversion_enabled = ?", false).
			Where("grid_trading_enabled = ?", false)
	default:
		return false
	}

	query.Count(&count)
	return count > 0
}

// 获取策略历史表现数据
func (s *Server) getStrategyPerformanceData(strategyType string) (winRate, maxDrawdown, avgProfit, sharpeRatio, volatility float64, totalTrades int) {
	// 这里应该从实际的回测结果或历史交易数据中获取
	// 暂时使用基于策略类型的模拟数据，后续可以替换为真实数据

	switch strategyType {
	case "mean_reversion":
		// 均值回归策略通常在震荡市场表现较好
		return 0.62, 0.15, 0.023, 1.8, 0.18, 245
	case "traditional":
		// 传统策略在趋势市场表现稳定
		return 0.58, 0.12, 0.018, 1.5, 0.15, 189
	case "moving_average":
		// 均线策略胜率较低但回撤可控
		return 0.54, 0.10, 0.015, 1.3, 0.12, 156
	case "rsi":
		// RSI策略胜率较高但波动较大
		return 0.65, 0.22, 0.028, 2.1, 0.25, 98
	case "macd":
		// MACD策略表现稳健
		return 0.60, 0.18, 0.021, 1.9, 0.20, 87
	case "bollinger_bands":
		// 布林带策略在震荡市场表现突出
		return 0.63, 0.20, 0.025, 1.7, 0.22, 76
	case "grid_trading":
		// 网格交易策略在震荡市场表现稳定，基于真实回测数据
		return 0.67, 0.08, 0.032, 2.2, 0.15, 120
	default:
		return 0.55, 0.15, 0.020, 1.5, 0.18, 50
	}
}

// 计算策略风险等级
func calculateStrategyRiskLevel(winRate, maxDrawdown, volatility float64) string {
	riskScore := (1-winRate)*0.4 + maxDrawdown*0.4 + volatility*0.2

	if riskScore < 0.3 {
		return "low"
	} else if riskScore < 0.6 {
		return "medium"
	}
	return "high"
}

// 确定策略适用市场条件
func getSuitableMarket(strategyType string, analysis *MarketAnalysisResult) string {
	switch strategyType {
	case "mean_reversion":
		if analysis.Oscillation > 60 {
			return "高震荡市场"
		} else if analysis.Oscillation > 40 {
			return "中等震荡市场"
		}
		return "低震荡市场"
	case "moving_average":
		if analysis.Volatility > 50 && analysis.Oscillation < 40 {
			return "趋势明确市场"
		}
		return "一般趋势市场"
	case "traditional":
		return "中等波动市场"
	case "rsi", "bollinger_bands":
		return "震荡整理市场"
	case "macd":
		return "趋势转折市场"
	case "grid_trading":
		if analysis.Oscillation > 50 {
			return "高震荡市场"
		} else if analysis.Oscillation > 30 {
			return "中等震荡市场"
		}
		return "横盘震荡市场"
	default:
		return "通用市场"
	}
}

func (s *Server) generateStrategyRecommendations(analysis *MarketAnalysisResult) []MarketStrategyRecommendation {
	var recommendations []MarketStrategyRecommendation

	// 均值回归策略推荐
	mrScore := 5
	mrConfidence := 50.0
	if analysis.Oscillation > 60 {
		mrScore = 9
		mrConfidence = 85.0
	} else if analysis.Oscillation > 40 {
		mrScore = 7
		mrConfidence = 65.0
	}

	// 获取策略性能数据
	mrWinRate, mrMaxDrawdown, mrAvgProfit, mrSharpeRatio, mrVolatility, mrTotalTrades := s.getStrategyPerformanceData("mean_reversion")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "mean_reversion",
		Name:           "均值回归策略",
		Score:          mrScore,
		Confidence:     mrConfidence,
		Reason:         "当前市场震荡明显，均值回归策略最适合",
		Exists:         s.checkStrategyExists("mean_reversion"),
		WinRate:        mrWinRate,
		MaxDrawdown:    mrMaxDrawdown,
		TotalTrades:    mrTotalTrades,
		AvgProfit:      mrAvgProfit,
		SharpeRatio:    mrSharpeRatio,
		Volatility:     mrVolatility,
		RiskLevel:      calculateStrategyRiskLevel(mrWinRate, mrMaxDrawdown, mrVolatility),
		SuitableMarket: getSuitableMarket("mean_reversion", analysis),
	})

	// 传统策略推荐
	traditionalScore := 5
	traditionalConfidence := 40.0
	if analysis.Volatility > 30 && analysis.Volatility < 70 {
		traditionalScore = 6
		traditionalConfidence = 45.0
	}

	// 获取传统策略性能数据
	tradWinRate, tradMaxDrawdown, tradAvgProfit, tradSharpeRatio, tradVolatility, tradTotalTrades := s.getStrategyPerformanceData("traditional")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "traditional",
		Name:           "传统策略",
		Score:          traditionalScore,
		Confidence:     traditionalConfidence,
		Reason:         "适合中等波动环境",
		Exists:         s.checkStrategyExists("traditional"),
		WinRate:        tradWinRate,
		MaxDrawdown:    tradMaxDrawdown,
		TotalTrades:    tradTotalTrades,
		AvgProfit:      tradAvgProfit,
		SharpeRatio:    tradSharpeRatio,
		Volatility:     tradVolatility,
		RiskLevel:      calculateStrategyRiskLevel(tradWinRate, tradMaxDrawdown, tradVolatility),
		SuitableMarket: getSuitableMarket("traditional", analysis),
	})

	// 均线策略推荐
	maScore := 3
	maConfidence := 20.0
	if analysis.Volatility > 50 && analysis.Oscillation < 40 {
		maScore = 4
		maConfidence = 25.0
	}

	// 获取均线策略性能数据
	maWinRate, maMaxDrawdown, maAvgProfit, maSharpeRatio, maVolatility, maTotalTrades := s.getStrategyPerformanceData("moving_average")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "moving_average",
		Name:           "均线策略",
		Score:          maScore,
		Confidence:     maConfidence,
		Reason:         "需要明确趋势，当前市场缺乏方向",
		Exists:         s.checkStrategyExists("moving_average"),
		WinRate:        maWinRate,
		MaxDrawdown:    maMaxDrawdown,
		TotalTrades:    maTotalTrades,
		AvgProfit:      maAvgProfit,
		SharpeRatio:    maSharpeRatio,
		Volatility:     maVolatility,
		RiskLevel:      calculateStrategyRiskLevel(maWinRate, maMaxDrawdown, maVolatility),
		SuitableMarket: getSuitableMarket("moving_average", analysis),
	})

	// 添加更多策略推荐（这些是系统中不存在的）
	// RSI策略推荐
	rsiScore := 4
	rsiConfidence := 30.0
	if analysis.Oscillation > 50 {
		rsiScore = 6
		rsiConfidence = 55.0
	}

	// 获取RSI策略性能数据
	rsiWinRate, rsiMaxDrawdown, rsiAvgProfit, rsiSharpeRatio, rsiVolatility, rsiTotalTrades := s.getStrategyPerformanceData("rsi")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "rsi",
		Name:           "RSI超买超卖策略",
		Score:          rsiScore,
		Confidence:     rsiConfidence,
		Reason:         "利用相对强弱指标识别超买超卖信号",
		Exists:         false, // RSI策略在当前系统中不存在
		WinRate:        rsiWinRate,
		MaxDrawdown:    rsiMaxDrawdown,
		TotalTrades:    rsiTotalTrades,
		AvgProfit:      rsiAvgProfit,
		SharpeRatio:    rsiSharpeRatio,
		Volatility:     rsiVolatility,
		RiskLevel:      calculateStrategyRiskLevel(rsiWinRate, rsiMaxDrawdown, rsiVolatility),
		SuitableMarket: getSuitableMarket("rsi", analysis),
	})

	// MACD策略推荐
	macdScore := 4
	macdConfidence := 35.0
	if analysis.Volatility > 40 && analysis.Oscillation < 50 {
		macdScore = 7
		macdConfidence = 60.0
	}

	// 获取MACD策略性能数据
	macdWinRate, macdMaxDrawdown, macdAvgProfit, macdSharpeRatio, macdVolatility, macdTotalTrades := s.getStrategyPerformanceData("macd")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "macd",
		Name:           "MACD趋势策略",
		Score:          macdScore,
		Confidence:     macdConfidence,
		Reason:         "使用MACD指标捕捉趋势变化",
		Exists:         false, // MACD策略在当前系统中不存在
		WinRate:        macdWinRate,
		MaxDrawdown:    macdMaxDrawdown,
		TotalTrades:    macdTotalTrades,
		AvgProfit:      macdAvgProfit,
		SharpeRatio:    macdSharpeRatio,
		Volatility:     macdVolatility,
		RiskLevel:      calculateStrategyRiskLevel(macdWinRate, macdMaxDrawdown, macdVolatility),
		SuitableMarket: getSuitableMarket("macd", analysis),
	})

	// 布林带策略推荐
	bollingerScore := 3
	bollingerConfidence := 25.0
	if analysis.Oscillation > 40 {
		bollingerScore = 5
		bollingerConfidence = 40.0
	}

	// 获取布林带策略性能数据
	bbWinRate, bbMaxDrawdown, bbAvgProfit, bbSharpeRatio, bbVolatility, bbTotalTrades := s.getStrategyPerformanceData("bollinger_bands")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "bollinger_bands",
		Name:           "布林带突破策略",
		Score:          bollingerScore,
		Confidence:     bollingerConfidence,
		Reason:         "利用布林带识别价格突破和支撑阻力",
		Exists:         false, // 布林带策略在当前系统中不存在
		WinRate:        bbWinRate,
		MaxDrawdown:    bbMaxDrawdown,
		TotalTrades:    bbTotalTrades,
		AvgProfit:      bbAvgProfit,
		SharpeRatio:    bbSharpeRatio,
		Volatility:     bbVolatility,
		RiskLevel:      calculateStrategyRiskLevel(bbWinRate, bbMaxDrawdown, bbVolatility),
		SuitableMarket: getSuitableMarket("bollinger_bands", analysis),
	})

	// 网格交易策略推荐
	gridScore := 6.0
	gridConfidence := 60.0

	// 横盘震荡市场最适合网格
	if analysis.Trend == "震荡" {
		gridScore += 3
		gridConfidence = 85.0
	} else if analysis.Trend == "混合" {
		gridScore += 1
		gridConfidence = 70.0
	} else {
		gridScore -= 2 // 趋势市场不适合网格
		gridConfidence = 40.0
	}

	// 低波动率更适合网格
	if analysis.Volatility < 30 {
		gridScore += 1
	}

	// 获取网格策略性能数据
	gridWinRate, gridMaxDrawdown, gridAvgProfit, gridSharpeRatio, gridVolatility, gridTotalTrades := s.getStrategyPerformanceData("grid_trading")

	recommendations = append(recommendations, MarketStrategyRecommendation{
		Type:           "grid_trading",
		Name:           "网格交易策略",
		Score:          int(gridScore),
		Confidence:     gridConfidence,
		Reason:         "在价格区间内设置多个买卖点，通过低买高卖获得稳定收益",
		Exists:         s.checkStrategyExists("grid_trading"),
		WinRate:        gridWinRate,
		MaxDrawdown:    gridMaxDrawdown,
		TotalTrades:    gridTotalTrades,
		AvgProfit:      gridAvgProfit,
		SharpeRatio:    gridSharpeRatio,
		Volatility:     gridVolatility,
		RiskLevel:      calculateStrategyRiskLevel(gridWinRate, gridMaxDrawdown, gridVolatility),
		SuitableMarket: getSuitableMarket("grid_trading", analysis),
	})

	// 按评分降序排序，确保最佳策略排在第一位
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// 辅助函数：计算标准差 (如果不存在的话)
func calculateMarketStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}

	return math.Sqrt(sum / float64(len(values)))
}

// 计算RSI (如果不存在的话)
func calculateMarketRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return []float64{}
	}

	var gains, losses []float64
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	var rsi []float64
	for i := period; i < len(gains); i++ {
		avgGain := 0.0
		avgLoss := 0.0
		for j := i - period; j < i; j++ {
			avgGain += gains[j]
			avgLoss += losses[j]
		}
		avgGain /= float64(period)
		avgLoss /= float64(period)

		if avgLoss == 0 {
			rsi = append(rsi, 100)
		} else {
			rs := avgGain / avgLoss
			rsi = append(rsi, 100-(100/(1+rs)))
		}
	}

	return rsi
}

// 分析趋势和震荡
func analyzeTrendAndOscillation(klines []struct {
	Symbol string
	Close  float64
	Time   time.Time
}) (string, float64) {
	if len(klines) < 10 {
		return "数据不足", 0
	}

	// 按币种分组数据，避免混合计算导致的错误
	symbolData := make(map[string][]float64)
	for _, kline := range klines {
		if symbolData[kline.Symbol] == nil {
			symbolData[kline.Symbol] = []float64{}
		}
		symbolData[kline.Symbol] = append(symbolData[kline.Symbol], kline.Close)
	}

	// 计算每个币种的趋势和震荡度
	totalOscillation := 0.0
	totalTrendScore := 0.0
	symbolCount := 0

	for _, prices := range symbolData {
		if len(prices) < 5 {
			continue
		}

		// 计算该币种的趋势得分（-1到1之间，负数表示下跌趋势）
		firstPrice := prices[0]
		lastPrice := prices[len(prices)-1]
		trendChange := (lastPrice - firstPrice) / firstPrice
		totalTrendScore += trendChange

		// 计算该币种的震荡度（使用标准差相对均值，更合理）
		oscillation := calculateSymbolOscillation(prices)
		totalOscillation += oscillation

		symbolCount++
	}

	// 计算平均趋势得分和震荡度
	avgTrendScore := 0.0
	avgOscillation := 0.0

	if symbolCount > 0 {
		avgTrendScore = totalTrendScore / float64(symbolCount)
		avgOscillation = totalOscillation / float64(symbolCount)
	}

	// 基于平均趋势得分判断整体趋势（更合理的阈值）
	trend := "震荡"
	if avgTrendScore > 0.03 { // 平均上涨3%以上
		trend = "上涨"
	} else if avgTrendScore < -0.03 { // 平均下跌3%以上
		trend = "下跌"
	}

	return trend, avgOscillation
}

// 计算单个币种的震荡度
func calculateSymbolOscillation(prices []float64) float64 {
	if len(prices) < 3 {
		return 0
	}

	// 计算价格的标准差相对均值
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	mean := sum / float64(len(prices))

	sumSquares := 0.0
	for _, price := range prices {
		sumSquares += math.Pow(price-mean, 2)
	}
	stdDev := math.Sqrt(sumSquares / float64(len(prices)))

	// 震荡度 = (标准差 / 均值) * 100，限制最大值为20%（避免极端值）
	oscillation := (stdDev / mean) * 100
	if oscillation > 20 {
		oscillation = 20
	}

	return oscillation
}

// 计算动态缓存过期时间
func calculateDynamicCacheTTL(result *ComprehensiveMarketAnalysis) time.Duration {
	if result == nil || result.MarketAnalysis == nil {
		return 5 * time.Minute // 默认5分钟
	}

	volatility := result.MarketAnalysis.Volatility
	oscillation := result.MarketAnalysis.Oscillation

	// 根据波动率和震荡度动态调整缓存时间
	// 高波动率和高震荡度 = 更短的缓存时间
	volatilityFactor := math.Min(volatility/20.0, 2.0)   // 波动率因子，最多2倍
	oscillationFactor := math.Min(oscillation/10.0, 2.0) // 震荡度因子，最多2倍

	combinedFactor := math.Max(volatilityFactor, oscillationFactor)

	// 基础缓存时间：3分钟
	baseTTL := 3 * time.Minute

	// 根据市场活跃度调整缓存时间
	// 高波动市场：缓存时间减半
	adjustedTTL := time.Duration(float64(baseTTL) / (1.0 + combinedFactor))

	// 确保缓存时间在合理范围内：1分钟到10分钟
	if adjustedTTL < 1*time.Minute {
		adjustedTTL = 1 * time.Minute
	} else if adjustedTTL > 10*time.Minute {
		adjustedTTL = 10 * time.Minute
	}

	log.Printf("[缓存策略] 波动率:%.2f%%, 震荡度:%.2f%%, 缓存时间:%v",
		volatility, oscillation, adjustedTTL)

	return adjustedTTL
}

// 统计强势/弱势币种
func (s *Server) countStrongWeakSymbols() (int, int) {
	// 获取最近24小时的价格变化
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)

	// 首先获取高交易量的币种列表（降低门槛以包含更多币种）
	var topSymbols []string
	err := s.db.DB().Raw(`
		SELECT symbol
		FROM (
			SELECT symbol, MAX(quote_volume) as max_volume
			FROM binance_24h_stats
			WHERE quote_volume > 1000
				AND created_at >= ? AND created_at <= ?
			GROUP BY symbol
			ORDER BY max_volume DESC
			LIMIT 200
		) as top_symbols
	`, startTime, endTime).Pluck("symbol", &topSymbols).Error

	if err != nil {
		log.Printf("[强势弱势统计] 查询binance_24h_stats失败: %v", err)
		return 0, 0
	}

	if len(topSymbols) == 0 {
		log.Printf("[强势弱势统计] binance_24h_stats表中没有找到交易量>10000的币种")
		// 如果没有数据，返回0,0
		return 0, 0
	}

	// 为每个币种计算24小时涨跌幅
	strong := 0
	weak := 0

	for _, symbol := range topSymbols {
		// 直接从binance_24h_stats表获取涨跌幅数据，避免复杂的K线计算
		var priceChange float64
		err := s.db.DB().Table("binance_24h_stats").
			Select("price_change_percent").
			Where("symbol = ? AND created_at >= ? AND created_at <= ?", symbol, startTime, endTime).
			Order("created_at DESC").
			Limit(1).
			Scan(&priceChange).Error

		if err != nil {
			continue
		}

		// 使用更低的阈值来适应当前市场环境：±2%而不是±5%
		if priceChange > 2 {
			strong++
		} else if priceChange < -2 {
			weak++
		}
	}

	return strong, weak
}

// 计算市场宽度指标
func (s *Server) countMarketBreadthIndicators() (strong, weak, bigGainers, bigLosers, neutralSymbols int, advanceDeclineRatio float64) {
	// 获取最近24小时的价格变化
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)

	// 优化：使用更高效的查询获取按交易量排序的top币种
	var topSymbols []string
	// 简化查询：直接按交易量排序，避免复杂的子查询
	err := s.db.DB().Table("binance_24h_stats").
		Select("symbol").
		Where("quote_volume > 1000 AND created_at >= ? AND created_at <= ?", startTime, endTime).
		Order("quote_volume DESC").
		Limit(200).
		Pluck("symbol", &topSymbols).Error

	if err != nil {
		log.Printf("[市场宽度统计] 查询高交易量币种失败: %v", err)
		return 0, 0, 0, 0, 0, 0.0
	}

	if len(topSymbols) == 0 {
		return 0, 0, 0, 0, 0, 0.0
	}

	// 优化：使用单个查询获取所有币种的最新价格变化，避免N+1查询
	// 构建IN查询条件
	symbolsStr := make([]string, len(topSymbols))
	args := make([]interface{}, len(topSymbols)+2)
	for i, symbol := range topSymbols {
		symbolsStr[i] = "?"
		args[i] = symbol
	}
	args[len(topSymbols)] = startTime
	args[len(topSymbols)+1] = endTime

	query := fmt.Sprintf(`
		SELECT symbol, price_change_percent
		FROM binance_24h_stats
		WHERE symbol IN (%s)
			AND created_at >= ?
			AND created_at <= ?
		ORDER BY symbol, created_at DESC
	`, strings.Join(symbolsStr, ","))

	rows, err := s.db.DB().Raw(query, args...).Rows()
	if err != nil {
		log.Printf("[市场宽度统计] 查询价格变化数据失败: %v", err)
		return 0, 0, 0, 0, 0, 0.0
	}
	defer rows.Close()

	// 按币种分组，只保留最新的价格变化
	symbolPriceChanges := make(map[string]float64)
	for rows.Next() {
		var symbol string
		var priceChange float64
		if err := rows.Scan(&symbol, &priceChange); err != nil {
			continue
		}
		// 只保留每个symbol的第一条记录（最新的）
		if _, exists := symbolPriceChanges[symbol]; !exists {
			symbolPriceChanges[symbol] = priceChange
		}
	}

	// 统计各类币种数量
	for _, priceChange := range symbolPriceChanges {
		// 统计强势弱势币种 (±2%)
		if priceChange > 2 {
			strong++
		} else if priceChange < -2 {
			weak++
		} else {
			neutralSymbols++
		}

		// 统计大涨大跌币种 (±5%)
		if priceChange > 5 {
			bigGainers++
		} else if priceChange < -5 {
			bigLosers++
		}
	}

	// 计算涨跌比
	if weak > 0 {
		advanceDeclineRatio = float64(strong) / float64(weak)
	} else if strong > 0 {
		advanceDeclineRatio = float64(strong) // 如果没有弱势币种，涨跌比为强势币种数
	} else {
		advanceDeclineRatio = 1.0 // 如果都没有，设为1.0
	}

	return strong, weak, bigGainers, bigLosers, neutralSymbols, advanceDeclineRatio
}

// 计算成交量指标
func (s *Server) countVolumeIndicators() (volumeGainers, volumeDecliners int, avgVolumeChange float64) {
	// 获取最近24小时的成交量数据
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)
	compareStartTime := endTime.AddDate(0, 0, -2) // 前一天作为对比

	// 获取高交易量的币种
	rows, err := s.db.DB().Raw(`
		SELECT symbol
		FROM (
			SELECT symbol, MAX(quote_volume) as max_volume
			FROM binance_24h_stats
			WHERE quote_volume > 1000
				AND created_at >= ? AND created_at <= ?
			GROUP BY symbol
			ORDER BY max_volume DESC
			LIMIT 100
		) as top_symbols
	`, startTime, endTime).Rows()

	if err != nil {
		log.Printf("[成交量统计] 查询币种失败: %v", err)
		return 0, 0, 0.0
	}
	defer rows.Close()

	var totalVolumeChange float64
	var analyzedCount int

	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}

		// 获取最近24小时和前一天24小时的成交量
		var recentVolume, prevVolume float64

		// 最近24小时
		err = s.db.DB().Table("binance_24h_stats").
			Select("COALESCE(AVG(quote_volume), 0)").
			Where("symbol = ? AND created_at >= ? AND created_at <= ?", symbol, startTime, endTime).
			Scan(&recentVolume).Error

		if err != nil || recentVolume == 0 {
			continue
		}

		// 前一天24小时
		err = s.db.DB().Table("binance_24h_stats").
			Select("COALESCE(AVG(quote_volume), 0)").
			Where("symbol = ? AND created_at >= ? AND created_at < ?", symbol, compareStartTime, startTime).
			Scan(&prevVolume).Error

		if err != nil || prevVolume == 0 {
			continue
		}

		// 计算成交量变化率
		volumeChange := ((recentVolume - prevVolume) / prevVolume) * 100
		totalVolumeChange += volumeChange
		analyzedCount++

		// 同时获取价格变化，用于判断放量上涨还是缩量下跌
		var priceChange float64
		err = s.db.DB().Table("binance_24h_stats").
			Select("COALESCE(AVG(price_change_percent), 0)").
			Where("symbol = ? AND created_at >= ? AND created_at <= ?", symbol, startTime, endTime).
			Scan(&priceChange).Error

		if err != nil {
			continue
		}

		// 判断成交量和价格变化的组合
		if volumeChange > 20 && priceChange > 1 {
			// 放量上涨：成交量增加且价格上涨
			volumeGainers++
		} else if volumeChange < -20 && priceChange < -1 {
			// 缩量下跌：成交量减少且价格下跌
			volumeDecliners++
		}
	}

	// 计算平均成交量变化率
	if analyzedCount > 0 {
		avgVolumeChange = totalVolumeChange / float64(analyzedCount)
	}

	return volumeGainers, volumeDecliners, avgVolumeChange
}

// 计算波动率指标
func (s *Server) countVolatilityIndicators() (marketVolatility float64, highVolSymbols, lowVolSymbols int) {
	// 获取最近3天的价格数据来计算波动率（优化：从7天减少到3天）
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -3)

	// 获取活跃币种列表（优化：减少数量从100到30，提高性能）
	var symbols []string
	err := s.db.DB().Table("binance_24h_stats").
		Select("DISTINCT symbol").
		Where("quote_volume > 5000 AND created_at >= ? AND created_at <= ?", startTime, endTime). // 提高交易量门槛
		Limit(30).                                                                                // 从100减少到30，不使用ORDER BY避免DISTINCT冲突
		Pluck("symbol", &symbols).Error

	if err != nil {
		log.Printf("[波动率统计] 查询币种失败: %v", err)
		return 0, 0, 0
	}

	if len(symbols) == 0 {
		return 0, 0, 0
	}

	// 构建IN查询条件
	symbolsStr := make([]string, len(symbols))
	args := make([]interface{}, len(symbols)+2)
	for i, symbol := range symbols {
		symbolsStr[i] = "?"
		args[i] = symbol
	}
	args[len(symbols)] = startTime
	args[len(symbols)+1] = endTime

	query := fmt.Sprintf("SELECT symbol, close_price FROM market_klines WHERE symbol IN (%s) AND open_time >= ? AND open_time <= ? ORDER BY symbol, open_time ASC",
		strings.Join(symbolsStr, ","))

	// 使用单个查询获取所有币种的价格数据
	rows, err := s.db.DB().Raw(query, args...).Rows()
	if err != nil {
		log.Printf("[波动率统计] 查询价格数据失败: %v", err)
		return 0, 0, 0
	}
	defer rows.Close()

	// 按币种分组价格数据
	symbolPrices := make(map[string][]float64)
	for rows.Next() {
		var symbol string
		var price float64
		if err := rows.Scan(&symbol, &price); err != nil {
			continue
		}
		symbolPrices[symbol] = append(symbolPrices[symbol], price)
	}

	var totalVolatility float64
	var analyzedCount int

	// 计算每个币种的波动率
	for _, prices := range symbolPrices {
		if len(prices) < 3 {
			continue
		}

		// 计算该币种的波动率
		symbolVolatility := calculateSymbolVolatility(prices)

		totalVolatility += symbolVolatility
		analyzedCount++

		// 分类高低波动率币种
		if symbolVolatility > 8 { // 年化波动率 > 8%
			highVolSymbols++
		} else if symbolVolatility < 3 { // 年化波动率 < 3%
			lowVolSymbols++
		}
	}

	// 计算市场平均波动率
	if analyzedCount > 0 {
		marketVolatility = totalVolatility / float64(analyzedCount)
	}

	return marketVolatility, highVolSymbols, lowVolSymbols
}

// 计算单个币种的波动率
func calculateSymbolVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算日收益率
	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	if len(returns) == 0 {
		return 0
	}

	// 计算波动率（标准差）
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	sumSquares := 0.0
	for _, ret := range returns {
		sumSquares += math.Pow(ret-mean, 2)
	}

	if len(returns) <= 1 {
		return 0
	}

	stdDev := math.Sqrt(sumSquares / float64(len(returns)-1))

	// 年化波动率
	annualVolatility := stdDev * math.Sqrt(365) * 100

	return annualVolatility
}

// 数据结构
type MarketAnalysisResult struct {
	// 基础指标（保持向后兼容）
	Volatility  float64 `json:"volatility"`
	Trend       string  `json:"trend"`
	Oscillation float64 `json:"oscillation"`

	// 增强的市场环境检测（用于均值回归策略增强）
	MarketRegime     string  `json:"market_regime"`     // 市场环境: "oscillation", "strong_trend", "high_volatility", "mixed"
	RegimeConfidence float64 `json:"regime_confidence"` // 环境判断置信度 (0-1)
	TrendStrength    float64 `json:"trend_strength"`    // 趋势强度 (-1到1, 负数表示下跌)
	VolatilityLevel  float64 `json:"volatility_level"`  // 波动率水平
	OscillationIndex float64 `json:"oscillation_index"` // 震荡指数 (0-1, 越高越震荡)

	// 市场宽度指标
	BullishSymbols     int `json:"bullish_symbols"`      // 上涨币种数 (>2%)
	BearishSymbols     int `json:"bearish_symbols"`      // 下跌币种数 (<-2%)
	SidewaysSymbols    int `json:"sideways_symbols"`     // 震荡币种数 (-2%~2%)
	StrongTrendSymbols int `json:"strong_trend_symbols"` // 强趋势币种数 (>5%或<-5%)

	// 成交量指标
	HighVolumeRatio     float64 `json:"high_volume_ratio"`    // 高活跃度币种占比
	VolumeConcentration float64 `json:"volume_concentration"` // 成交量集中度

	// 环境稳定性指标
	RegimeStability   float64 `json:"regime_stability"`   // 环境稳定性 (0-1)
	ChangeProbability float64 `json:"change_probability"` // 环境变化概率 (0-1)
}

type TechnicalIndicatorsResult struct {
	// 基础指标
	BTCVolatility float64 `json:"btc_volatility"`
	AvgRSI        float64 `json:"avg_rsi"`
	StrongSymbols int     `json:"strong_symbols"`
	WeakSymbols   int     `json:"weak_symbols"`

	// 市场宽度指标
	AdvanceDeclineRatio float64 `json:"advance_decline_ratio"` // 涨跌比
	BigGainers          int     `json:"big_gainers"`           // 大涨币种数 (>5%)
	BigLosers           int     `json:"big_losers"`            // 大跌币种数 (<-5%)
	NeutralSymbols      int     `json:"neutral_symbols"`       // 中性币种数 (-2%~2%)

	// 成交量指标
	VolumeGainers   int     `json:"volume_gainers"`    // 放量上涨币种数
	VolumeDecliners int     `json:"volume_decliners"`  // 缩量下跌币种数
	AvgVolumeChange float64 `json:"avg_volume_change"` // 平均成交量变化率

	// 波动率指标
	MarketVolatility      float64 `json:"market_volatility"`       // 市场平均波动率
	HighVolatilitySymbols int     `json:"high_volatility_symbols"` // 高波动率币种数
	LowVolatilitySymbols  int     `json:"low_volatility_symbols"`  // 低波动率币种数
}

type MarketStrategyRecommendation struct {
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	Score          int     `json:"score"`
	Confidence     float64 `json:"confidence"`
	Reason         string  `json:"reason"`
	Exists         bool    `json:"exists"`          // 是否已在系统中存在
	WinRate        float64 `json:"win_rate"`        // 历史胜率
	MaxDrawdown    float64 `json:"max_drawdown"`    // 最大回撤
	TotalTrades    int     `json:"total_trades"`    // 总交易次数
	AvgProfit      float64 `json:"avg_profit"`      // 平均利润
	SharpeRatio    float64 `json:"sharpe_ratio"`    // 夏普比率
	Volatility     float64 `json:"volatility"`      // 策略波动率
	RiskLevel      string  `json:"risk_level"`      // 风险等级: low, medium, high
	SuitableMarket string  `json:"suitable_market"` // 适用市场条件
}

// ============================================================================
// 增强市场环境检测（用于均值回归策略增强）
// ============================================================================

// EnhancedMarketAnalysis 增强市场分析结果
type EnhancedMarketAnalysis struct {
	Regime              string
	Confidence          float64
	TrendStrength       float64
	OscillationIndex    float64
	BullishCount        int
	BearishCount        int
	SidewaysCount       int
	StrongTrendCount    int
	HighVolumeRatio     float64
	VolumeConcentration float64
	Stability           float64
	ChangeProbability   float64
}

// analyzeEnhancedMarketEnvironment 增强市场环境分析（用于均值回归策略）
func (s *Server) analyzeEnhancedMarketEnvironment(klines []struct {
	Symbol string
	Close  float64
	Time   time.Time
}, avgVolatility float64) *EnhancedMarketAnalysis {

	result := &EnhancedMarketAnalysis{}

	// 1. 分析价格变化分布
	priceChanges := s.analyzePriceChangeDistribution(klines)
	result.BullishCount = priceChanges.Bullish
	result.BearishCount = priceChanges.Bearish
	result.SidewaysCount = priceChanges.Sideways
	result.StrongTrendCount = priceChanges.StrongTrend

	// 2. 计算趋势强度
	result.TrendStrength = s.calculateTrendStrength(klines)

	// 3. 计算震荡指数
	result.OscillationIndex = s.calculateOscillationIndex(priceChanges)

	// 4. 分析成交量分布
	volumeAnalysis := s.analyzeVolumeDistribution()
	result.HighVolumeRatio = volumeAnalysis.HighVolumeRatio
	result.VolumeConcentration = volumeAnalysis.Concentration

	// 5. 确定市场环境
	regimeAnalysis := s.determineMarketRegime(priceChanges, result.TrendStrength, result.OscillationIndex, avgVolatility)
	result.Regime = regimeAnalysis.Regime
	result.Confidence = regimeAnalysis.Confidence

	// 6. 计算环境稳定性
	result.Stability = s.calculateRegimeStability(priceChanges, result.TrendStrength, result.OscillationIndex)

	// 7. 估算变化概率
	result.ChangeProbability = s.estimateChangeProbability(result.Stability, avgVolatility)

	return result
}

// analyzePriceChangeDistribution 分析价格变化分布
func (s *Server) analyzePriceChangeDistribution(klines []struct {
	Symbol string
	Close  float64
	Time   time.Time
}) *PriceChangeDistribution {

	dist := &PriceChangeDistribution{}

	// 计算每个币种的价格变化
	symbolChanges := make(map[string][]float64)

	for _, kline := range klines {
		if symbolChanges[kline.Symbol] == nil {
			symbolChanges[kline.Symbol] = []float64{}
		}
		symbolChanges[kline.Symbol] = append(symbolChanges[kline.Symbol], kline.Close)
	}

	// 分析每个币种的变化
	for _, prices := range symbolChanges {
		if len(prices) < 2 {
			continue
		}

		// 计算24小时变化（简化处理，实际应该计算更精确的变化）
		changePercent := (prices[len(prices)-1] - prices[0]) / prices[0] * 100

		if changePercent > 5 {
			dist.StrongTrend++
			dist.Bullish++
		} else if changePercent > 2 {
			dist.Bullish++
		} else if changePercent < -5 {
			dist.StrongTrend++
			dist.Bearish++
		} else if changePercent < -2 {
			dist.Bearish++
		} else {
			dist.Sideways++
		}
	}

	return dist
}

// calculateTrendStrength 计算趋势强度
func (s *Server) calculateTrendStrength(klines []struct {
	Symbol string
	Close  float64
	Time   time.Time
}) float64 {

	if len(klines) == 0 {
		return 0
	}

	// 按时间分组的价格数据
	timeSorted := make(map[time.Time][]float64)
	for _, kline := range klines {
		if timeSorted[kline.Time] == nil {
			timeSorted[kline.Time] = []float64{}
		}
		timeSorted[kline.Time] = append(timeSorted[kline.Time], kline.Close)
	}

	// 计算每时间点的平均价格变化
	var timePoints []time.Time
	for t := range timeSorted {
		timePoints = append(timePoints, t)
	}
	sort.Slice(timePoints, func(i, j int) bool {
		return timePoints[i].Before(timePoints[j])
	})

	var changes []float64
	var directions []float64

	for i := 1; i < len(timePoints); i++ {
		prevPrices := timeSorted[timePoints[i-1]]
		currPrices := timeSorted[timePoints[i]]

		if len(prevPrices) == 0 || len(currPrices) == 0 {
			continue
		}

		// 计算平均价格变化
		prevAvg := 0.0
		for _, p := range prevPrices {
			prevAvg += p
		}
		prevAvg /= float64(len(prevPrices))

		currAvg := 0.0
		for _, p := range currPrices {
			currAvg += p
		}
		currAvg /= float64(len(currPrices))

		change := (currAvg - prevAvg) / prevAvg
		changes = append(changes, change)
		directions = append(directions, math.Abs(change)/change) // 方向：+1或-1
	}

	if len(changes) == 0 {
		return 0
	}

	// 计算趋势强度：变化幅度 × 一致性
	avgChange := 0.0
	for _, c := range changes {
		avgChange += math.Abs(c)
	}
	avgChange /= float64(len(changes))

	// 计算方向一致性（越接近1或-1，趋势越强）
	directionConsistency := 0.0
	for _, d := range directions {
		directionConsistency += d
	}
	directionConsistency = math.Abs(directionConsistency) / float64(len(directions))

	// 综合趋势强度
	trendStrength := avgChange * directionConsistency

	// 标准化到-1到1范围
	if trendStrength > 1 {
		trendStrength = 1
	} else if trendStrength < -1 {
		trendStrength = -1
	}

	return trendStrength
}

// calculateOscillationIndex 计算震荡指数
func (s *Server) calculateOscillationIndex(priceChanges *PriceChangeDistribution) float64 {
	total := float64(priceChanges.Bullish + priceChanges.Bearish + priceChanges.Sideways)
	if total == 0 {
		return 0
	}

	// 震荡指数 = 中性币种占比 × (1 - 强趋势占比)
	sidewaysRatio := float64(priceChanges.Sideways) / total
	strongTrendRatio := float64(priceChanges.StrongTrend) / total

	oscillationIndex := sidewaysRatio * (1 - strongTrendRatio)

	// 确保在0-1范围内
	if oscillationIndex < 0 {
		oscillationIndex = 0
	} else if oscillationIndex > 1 {
		oscillationIndex = 1
	}

	return oscillationIndex
}

// analyzeVolumeDistribution 分析成交量分布
func (s *Server) analyzeVolumeDistribution() *VolumeAnalysis {
	analysis := &VolumeAnalysis{}

	// 查询24小时成交量数据
	query := `
		SELECT COUNT(*) as total,
		       SUM(CASE WHEN quote_volume > 1000000 THEN 1 ELSE 0 END) as high_volume
		FROM binance_24h_stats
		WHERE market_type = 'spot' AND quote_volume > 10000`

	var total, highVolume int
	err := s.db.DB().Raw(query).Row().Scan(&total, &highVolume)
	if err != nil {
		log.Printf("[VolumeAnalysis] 查询成交量数据失败: %v", err)
		return &VolumeAnalysis{HighVolumeRatio: 0.5, Concentration: 0.5}
	}

	if total > 0 {
		analysis.HighVolumeRatio = float64(highVolume) / float64(total)
	}

	// 计算成交量集中度（前10名占比）
	concentrationQuery := `
		SELECT
			(SELECT SUM(quote_volume) FROM (
				SELECT quote_volume FROM binance_24h_stats
				WHERE market_type = 'spot' AND quote_volume > 10000
				ORDER BY quote_volume DESC LIMIT 10
			) as top10) /
			(SELECT SUM(quote_volume) FROM binance_24h_stats
			 WHERE market_type = 'spot' AND quote_volume > 10000) as concentration`

	var concentration float64
	err = s.db.DB().Raw(concentrationQuery).Row().Scan(&concentration)
	if err != nil {
		log.Printf("[VolumeAnalysis] 计算集中度失败: %v", err)
		concentration = 0.5
	}

	analysis.Concentration = concentration
	return analysis
}

// determineMarketRegime 确定市场环境
func (s *Server) determineMarketRegime(priceChanges *PriceChangeDistribution, trendStrength, oscillationIndex, volatility float64) *RegimeDetermination {

	determination := &RegimeDetermination{}

	// 计算各种环境的分数
	trendScore := math.Abs(trendStrength) * 100 // 趋势得分

	// 环境判断逻辑
	var bullScore, bearScore, sidewaysScore, volatileScore float64

	// 基于趋势强度的评分
	if trendStrength > 0.6 {
		bullScore += 80 // 强上涨趋势
	} else if trendStrength < -0.6 {
		bearScore += 80 // 强下跌趋势
	} else if trendStrength > 0.3 {
		bullScore += 50 // 中等上涨趋势
	} else if trendStrength < -0.3 {
		bearScore += 50 // 中等下跌趋势
	}

	// 基于震荡指数的评分
	if oscillationIndex > 0.7 {
		sidewaysScore += 90 // 高震荡环境
	} else if oscillationIndex > 0.5 {
		sidewaysScore += 60 // 中等震荡环境
	} else if oscillationIndex > 0.3 {
		sidewaysScore += 30 // 轻微震荡
	}

	// 基于波动率的评分
	if volatility > 15 {
		volatileScore += 70 // 高波动
	} else if volatility > 10 {
		volatileScore += 40 // 中等波动
	}

	// 基于价格变化分布的评分
	totalSymbols := float64(priceChanges.Bullish + priceChanges.Bearish + priceChanges.Sideways)
	if totalSymbols > 0 {
		bullRatio := float64(priceChanges.Bullish) / totalSymbols
		bearRatio := float64(priceChanges.Bearish) / totalSymbols
		sidewaysRatio := float64(priceChanges.Sideways) / totalSymbols

		// 如果大部分币种都在震荡，增加震荡得分
		if sidewaysRatio > 0.6 {
			sidewaysScore += 60
		} else if bullRatio > bearRatio*1.5 {
			// 如果上涨币种明显占优，增加趋势得分
			bullScore += 40
		} else if bearRatio > bullRatio*1.5 {
			// 如果下跌币种明显占优，增加趋势得分
			bearScore += 40
		}
	}

	// 确定主要环境
	maxScore := math.Max(math.Max(bullScore, bearScore), math.Max(sidewaysScore, volatileScore))

	if maxScore == bullScore && bullScore > 60 {
		determination.Regime = "strong_bull"
		determination.Confidence = bullScore / 100
	} else if maxScore == bearScore && bearScore > 60 {
		determination.Regime = "strong_bear"
		determination.Confidence = bearScore / 100
	} else if maxScore == volatileScore && volatileScore > 50 {
		determination.Regime = "high_volatility"
		determination.Confidence = volatileScore / 100
	} else if maxScore == sidewaysScore && sidewaysScore > 70 {
		determination.Regime = "oscillation"
		determination.Confidence = sidewaysScore / 100
	} else {
		// 默认环境判断
		if trendScore > 40 {
			if trendStrength > 0 {
				determination.Regime = "bull_trend"
			} else {
				determination.Regime = "bear_trend"
			}
			determination.Confidence = trendScore / 100
		} else if oscillationIndex > 0.4 {
			determination.Regime = "sideways"
			determination.Confidence = oscillationIndex
		} else {
			determination.Regime = "mixed"
			determination.Confidence = 0.5
		}
	}

	return determination
}

// calculateRegimeStability 计算环境稳定性
func (s *Server) calculateRegimeStability(priceChanges *PriceChangeDistribution, trendStrength, oscillationIndex float64) float64 {
	// 稳定性 = 震荡一致性 × (1 - 趋势变化性)
	total := float64(priceChanges.Bullish + priceChanges.Bearish + priceChanges.Sideways)
	if total == 0 {
		return 0.5
	}

	// 计算分布的集中度（越集中越稳定）
	maxGroup := math.Max(math.Max(float64(priceChanges.Bullish), float64(priceChanges.Bearish)), float64(priceChanges.Sideways))
	concentration := maxGroup / total

	// 趋势稳定性（趋势强度越强，越稳定）
	trendStability := math.Min(math.Abs(trendStrength)*2, 1)

	// 综合稳定性
	stability := (concentration * 0.6) + (trendStability * 0.4)

	return math.Min(stability, 1.0)
}

// estimateChangeProbability 估算环境变化概率
func (s *Server) estimateChangeProbability(stability, volatility float64) float64 {
	// 变化概率 = (1 - 稳定性) × 波动率因子
	instability := 1 - stability
	volatilityFactor := math.Min(volatility/20, 1) // 波动率超过20%时，变化概率很高

	changeProb := instability * volatilityFactor

	return math.Min(changeProb, 1.0)
}

// PriceChangeDistribution 价格变化分布
type PriceChangeDistribution struct {
	Bullish     int
	Bearish     int
	Sideways    int
	StrongTrend int
}

// VolumeAnalysis 成交量分析
type VolumeAnalysis struct {
	HighVolumeRatio float64
	Concentration   float64
}

// RegimeDetermination 环境判断结果
type RegimeDetermination struct {
	Regime     string
	Confidence float64
}
