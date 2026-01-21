package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetRiskReport 风险报告API
// GET /risk/report?symbol=BTC&days=30
func (s *Server) GetRiskReport(c *gin.Context) {
	symbol := c.DefaultQuery("symbol", "")
	days := 30

	if symbol == "" {
		c.JSON(400, gin.H{"error": "缺少交易对参数"})
		return
	}

	if daysStr := c.Query("days"); daysStr != "" {
		if n, err := strconv.Atoi(daysStr); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	ctx := c.Request.Context()

	// 计算风险指标
	report, err := s.calculateRiskMetrics(ctx, symbol, days)
	if err != nil {
		log.Printf("[ERROR] 计算风险指标失败: %v", err)
		c.JSON(500, gin.H{"error": "计算风险指标失败"})
		return
	}

	c.JSON(200, gin.H{
		"symbol":       symbol,
		"period_days":  days,
		"report":       report,
		"generated_at": time.Now(),
	})
}

// AssessRisk 风险评估API
// POST /risk/assess
func (s *Server) AssessRisk(c *gin.Context) {
	var req struct {
		Symbol    string             `json:"symbol" binding:"required"`
		Portfolio map[string]float64 `json:"portfolio,omitempty"` // 投资组合
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	ctx := c.Request.Context()

	if req.Portfolio != nil && len(req.Portfolio) > 0 {
		// 组合风险评估
		risks := s.assessPortfolioRisk(ctx, req.Portfolio)
		c.JSON(200, gin.H{
			"assessment_type": "portfolio",
			"portfolio":       req.Portfolio,
			"risk_assessment": risks,
		})
	} else {
		// 单币种风险评估
		risk := s.assessSingleAssetRisk(ctx, req.Symbol)
		c.JSON(200, gin.H{
			"assessment_type": "single_asset",
			"symbol":          req.Symbol,
			"risk_assessment": risk,
		})
	}
}

// GetRiskAlerts 风险告警API
// GET /risk/alerts?level=high&limit=10
func (s *Server) GetRiskAlerts(c *gin.Context) {
	level := c.DefaultQuery("level", "high")
	limit := 10

	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	alerts := s.generateRiskAlerts(level, limit)

	c.JSON(200, gin.H{
		"alerts":       alerts,
		"total_count":  len(alerts),
		"level":        level,
		"generated_at": time.Now(),
	})
}

// AcknowledgeAlert 确认告警API
// POST /risk/alerts/:id/acknowledge
func (s *Server) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")

	if alertID == "" {
		c.JSON(400, gin.H{"error": "缺少告警ID"})
		return
	}

	// 这里可以实现告警确认逻辑
	// 暂时只是返回成功

	c.JSON(200, gin.H{
		"message":         "告警已确认",
		"alert_id":        alertID,
		"acknowledged_at": time.Now(),
	})
}

// AnalyzePortfolio 投资组合分析API
// POST /risk/portfolio/analyze
func (s *Server) AnalyzePortfolio(c *gin.Context) {
	var req struct {
		Portfolio map[string]float64 `json:"portfolio" binding:"required"`
		Benchmark string             `json:"benchmark,omitempty"` // 基准
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	ctx := c.Request.Context()
	analysis := s.analyzePortfolio(ctx, req.Portfolio, req.Benchmark)

	c.JSON(200, gin.H{
		"portfolio":   req.Portfolio,
		"benchmark":   req.Benchmark,
		"analysis":    analysis,
		"analyzed_at": time.Now(),
	})
}

// calculateRiskMetrics 计算风险指标
func (s *Server) calculateRiskMetrics(ctx context.Context, symbol string, days int) (map[string]interface{}, error) {
	// 获取历史价格数据
	prices, err := s.getHistoricalPrices(ctx, symbol, "daily")
	if err != nil || len(prices) < 30 {
		return nil, err
	}

	// 计算收益率
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// 计算基础统计指标
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	// 计算VaR (95%置信度)
	zScore := 1.645 // 95%置信度的Z分数
	var95 := -(mean + zScore*stdDev)

	// 计算最大回撤
	maxDrawdown := s.calculateMaxDrawdown(prices)

	return map[string]interface{}{
		"mean_return":      mean,
		"volatility":       stdDev,
		"value_at_risk_95": var95,
		"maximum_drawdown": maxDrawdown,
		"sharpe_ratio":     mean / stdDev, // 简化的夏普比率
		"data_points":      len(returns),
	}, nil
}

// assessPortfolioRisk 评估投资组合风险
func (s *Server) assessPortfolioRisk(ctx context.Context, portfolio map[string]float64) map[string]interface{} {
	totalValue := 0.0
	for _, weight := range portfolio {
		totalValue += weight
	}

	// 归一化权重
	normalizedWeights := make(map[string]float64)
	for symbol, weight := range portfolio {
		normalizedWeights[symbol] = weight / totalValue
	}

	return map[string]interface{}{
		"total_assets":          len(portfolio),
		"normalized_weights":    normalizedWeights,
		"diversification_score": s.calculateDiversificationScore(normalizedWeights),
		"risk_level":            "medium", // 简化的风险等级
	}
}

// assessSingleAssetRisk 评估单资产风险
func (s *Server) assessSingleAssetRisk(ctx context.Context, symbol string) map[string]interface{} {
	return map[string]interface{}{
		"symbol":     symbol,
		"risk_level": "medium", // 简化的风险等级
		"volatility": 0.2,      // 简化的波动率
	}
}

// generateRiskAlerts 生成风险告警
func (s *Server) generateRiskAlerts(level string, limit int) []gin.H {
	// 这里可以实现真实的告警生成逻辑
	// 暂时返回模拟数据

	alerts := []gin.H{
		{
			"id":           "alert_001",
			"level":        "high",
			"symbol":       "BTC",
			"type":         "volatility_spike",
			"message":      "BTC波动率异常升高",
			"value":        0.85,
			"threshold":    0.7,
			"timestamp":    time.Now().Add(-time.Hour),
			"acknowledged": false,
		},
		{
			"id":           "alert_002",
			"level":        "medium",
			"symbol":       "ETH",
			"type":         "volume_drop",
			"message":      "ETH成交量显著下降",
			"value":        50000000,
			"threshold":    100000000,
			"timestamp":    time.Now().Add(-2 * time.Hour),
			"acknowledged": false,
		},
	}

	// 根据级别过滤
	filtered := make([]gin.H, 0)
	for _, alert := range alerts {
		if alert["level"] == level || level == "all" {
			filtered = append(filtered, alert)
			if len(filtered) >= limit {
				break
			}
		}
	}

	return filtered
}

// analyzePortfolio 分析投资组合
func (s *Server) analyzePortfolio(ctx context.Context, portfolio map[string]float64, benchmark string) map[string]interface{} {
	return map[string]interface{}{
		"expected_return": 0.12,
		"volatility":      0.18,
		"sharpe_ratio":    0.67,
		"benchmark":       benchmark,
	}
}

// calculateMaxDrawdown 计算最大回撤
func (s *Server) calculateMaxDrawdown(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	maxDrawdown := 0.0
	peak := prices[0]

	for _, price := range prices[1:] {
		if price > peak {
			peak = price
		}

		drawdown := (peak - price) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateDiversificationScore 计算分散化评分
func (s *Server) calculateDiversificationScore(weights map[string]float64) float64 {
	if len(weights) <= 1 {
		return 0.0 // 没有分散化
	}

	// 计算权重熵（熵值越高，分散化程度越高）
	entropy := 0.0
	for _, weight := range weights {
		if weight > 0 {
			entropy -= weight * math.Log(weight)
		}
	}

	// 归一化到0-1（最大熵为log(n)，n为资产数量）
	maxEntropy := math.Log(float64(len(weights)))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}

	return 0.0
}

// generateRiskWarnings 生成风险提示
func (s *Server) generateRiskWarnings(risk struct {
	VolatilityRisk float64
	LiquidityRisk  float64
	MarketRisk     float64
	TechnicalRisk  float64
	OverallRisk    float64
	RiskLevel      string
	RiskWarnings   []string
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
