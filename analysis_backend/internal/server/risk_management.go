package server

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RiskManagement 风险管理核心模块
type RiskManagement struct {
	// 风险评估器
	riskAssessor *RiskAssessor

	// 风险控制器
	riskController *RiskController

	// 风险监控器
	riskMonitor *RiskMonitor

	// 风险配置
	config RiskConfig

	// 风险数据存储
	riskData map[string]*RiskProfile
	dataMu   sync.RWMutex

	// 依赖服务
	featureEngineering *FeatureEngineering
	machineLearning    *MachineLearning
	db                 Database
}

// RiskConfig 风险配置
type RiskConfig struct {
	// 风险评估参数
	Assessment struct {
		MaxRiskScore   float64       `json:"max_risk_score"`  // 最大风险分数
		RiskThreshold  float64       `json:"risk_threshold"`  // 风险阈值
		UpdateInterval time.Duration `json:"update_interval"` // 更新间隔
		HistoryWindow  int           `json:"history_window"`  // 历史窗口大小
	} `json:"assessment"`

	// 风险控制参数
	Control struct {
		EnablePositionLimits bool      `json:"enable_position_limits"` // 启用仓位限制
		MaxPositionSize      float64   `json:"max_position_size"`      // 最大仓位大小
		MaxDrawdownLimit     float64   `json:"max_drawdown_limit"`     // 最大回撤限制
		DiversificationMin   int       `json:"diversification_min"`    // 最小分散度
		StopLossLevels       []float64 `json:"stop_loss_levels"`       // 止损级别
	} `json:"control"`

	// 风险监控参数
	Monitoring struct {
		AlertThresholds    map[string]float64 `json:"alert_thresholds"`    // 告警阈值
		MonitoringInterval time.Duration      `json:"monitoring_interval"` // 监控间隔
		ReportInterval     time.Duration      `json:"report_interval"`     // 报告间隔
		EnableRealTime     bool               `json:"enable_real_time"`    // 启用实时监控
	} `json:"monitoring"`

	// 风险类型权重
	RiskWeights struct {
		VolatilityWeight  float64 `json:"volatility_weight"`  // 波动率权重
		LiquidityWeight   float64 `json:"liquidity_weight"`   // 流动性权重
		MarketRiskWeight  float64 `json:"market_risk_weight"` // 市场风险权重
		CreditRiskWeight  float64 `json:"credit_risk_weight"` // 信用风险权重
		OperationalWeight float64 `json:"operational_weight"` // 操作风险权重
	} `json:"risk_weights"`
}

// RiskProfile 风险档案
type RiskProfile struct {
	Symbol         string
	LastUpdated    time.Time
	RiskScore      float64
	RiskLevel      RiskLevel
	RiskFactors    RiskFactors
	PositionLimits PositionLimits
	HistoricalRisk []RiskHistory
	Alerts         []RiskAlert

	// 高级风险指标
	VaR95             float64            `json:"var_95"`
	VaR99             float64            `json:"var_99"`
	CVaR95            float64            `json:"cvar_95"`
	Beta              float64            `json:"beta"`
	StressTestResults []StressTestResult `json:"stress_test_results"`
}

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RiskFactors 风险因子
type RiskFactors struct {
	Volatility  float64 // 波动率风险
	Liquidity   float64 // 流动性风险
	MarketRisk  float64 // 市场风险
	CreditRisk  float64 // 信用风险
	Operational float64 // 操作风险
	Composite   float64 // 综合风险
}

// PositionLimits 仓位限制
type PositionLimits struct {
	MaxPosition     float64   // 最大仓位
	MaxDrawdown     float64   // 最大回撤
	Diversification int       // 分散度要求
	StopLoss        []float64 // 止损点
}

// RiskHistory 风险历史
type RiskHistory struct {
	Timestamp time.Time
	RiskScore float64
	Position  float64
	PnL       float64
}

// RiskAlert 风险告警
type RiskAlert struct {
	ID        string
	Timestamp time.Time
	AlertType string
	Severity  string
	Message   string
	Symbol    string
	RiskScore float64
	Threshold float64
}

// RiskDecision 风险决策
type RiskDecision struct {
	Symbol          string
	CanTrade        bool
	MaxPosition     float64
	RiskScore       float64
	RiskLevel       RiskLevel
	Recommendations []string
	Warnings        []string
}

// =================== 高级风险指标类型 ===================

// AdvancedRiskMetrics 高级风险指标
type AdvancedRiskMetrics struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`

	// 传统风险指标
	Volatility   float64 `json:"volatility"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	SortinoRatio float64 `json:"sortino_ratio"`

	// VaR指标
	VaR95  float64 `json:"var_95"`
	VaR99  float64 `json:"var_99"`
	CVaR95 float64 `json:"cvar_95"`

	// 市场风险指标
	Beta  float64 `json:"beta"`
	Alpha float64 `json:"alpha"`
	R2    float64 `json:"r_squared"`

	// 流动性指标
	BidAskSpread     float64 `json:"bid_ask_spread"`
	TurnoverRatio    float64 `json:"turnover_ratio"`
	IlliquidityRatio float64 `json:"illiquidity_ratio"`

	// 压力测试结果
	StressTestResults []StressTestResult `json:"stress_test_results"`
}

// PortfolioRisk 投资组合风险
type PortfolioRisk struct {
	TotalValue        float64                       `json:"total_value"`
	TotalRisk         float64                       `json:"total_risk"`
	Diversification   float64                       `json:"diversification"`
	Correlation       map[string]map[string]float64 `json:"correlation"`
	AssetWeights      map[string]float64            `json:"asset_weights"`
	RiskContribution  map[string]float64            `json:"risk_contribution"`
	EfficientFrontier []EfficientPoint              `json:"efficient_frontier"`
}

// EfficientPoint 有效前沿点
type EfficientPoint struct {
	Return  float64            `json:"return"`
	Risk    float64            `json:"risk"`
	Weights map[string]float64 `json:"weights"`
}

// RiskBudget 风险预算
type RiskBudget struct {
	TotalBudget  float64            `json:"total_budget"`
	AssetBudgets map[string]float64 `json:"asset_budgets"`
	Utilization  map[string]float64 `json:"utilization"`
	Rebalancing  []RebalanceAction  `json:"rebalancing"`
}

// RebalanceAction 再平衡动作
type RebalanceAction struct {
	Symbol  string  `json:"symbol"`
	Current float64 `json:"current_weight"`
	Target  float64 `json:"target_weight"`
	Action  string  `json:"action"` // "BUY", "SELL", "HOLD"
	Amount  float64 `json:"amount"`
}

// RiskAssessor 风险评估器
type RiskAssessor struct {
	config RiskConfig
}

// RiskController 风险控制器
type RiskController struct {
	config RiskConfig
}

// RiskMonitor 风险监控器
type RiskMonitor struct {
	config        RiskConfig
	alerts        []RiskAlert
	alertsMu      sync.RWMutex
	subscribers   []RiskAlertSubscriber
	subscribersMu sync.RWMutex
}

// RiskAlertSubscriber 风险告警订阅者
type RiskAlertSubscriber interface {
	OnRiskAlert(alert RiskAlert)
}

// NewRiskManagement 创建风险管理实例
func NewRiskManagement(featureEngineering *FeatureEngineering, machineLearning *MachineLearning, db Database, config RiskConfig) *RiskManagement {
	rm := &RiskManagement{
		riskAssessor:       &RiskAssessor{config: config},
		riskController:     &RiskController{config: config},
		riskMonitor:        &RiskMonitor{config: config},
		config:             config,
		riskData:           make(map[string]*RiskProfile),
		featureEngineering: featureEngineering,
		machineLearning:    machineLearning,
		db:                 db,
	}

	// 设置默认配置
	rm.setDefaultConfig()

	// 初始化风险监控
	rm.initializeRiskMonitoring()

	log.Printf("[RiskManagement] 风险管理模块初始化完成")
	return rm
}

// setDefaultConfig 设置默认配置
func (rm *RiskManagement) setDefaultConfig() {
	// 风险评估默认配置
	if rm.config.Assessment.MaxRiskScore == 0 {
		rm.config.Assessment.MaxRiskScore = 100.0
	}
	if rm.config.Assessment.RiskThreshold == 0 {
		rm.config.Assessment.RiskThreshold = 70.0
	}
	if rm.config.Assessment.UpdateInterval == 0 {
		rm.config.Assessment.UpdateInterval = 1 * time.Hour
	}
	if rm.config.Assessment.HistoryWindow == 0 {
		rm.config.Assessment.HistoryWindow = 30
	}

	// 风险控制默认配置
	if rm.config.Control.MaxPositionSize == 0 {
		rm.config.Control.MaxPositionSize = 0.1 // 10% 最大仓位
	}
	if rm.config.Control.MaxDrawdownLimit == 0 {
		rm.config.Control.MaxDrawdownLimit = 0.2 // 20% 最大回撤
	}
	if rm.config.Control.DiversificationMin == 0 {
		rm.config.Control.DiversificationMin = 5
	}
	if len(rm.config.Control.StopLossLevels) == 0 {
		rm.config.Control.StopLossLevels = []float64{0.05, 0.1, 0.15} // 5%, 10%, 15% 止损
	}

	// 风险权重默认配置
	if rm.config.RiskWeights.VolatilityWeight == 0 {
		rm.config.RiskWeights.VolatilityWeight = 0.3
	}
	if rm.config.RiskWeights.LiquidityWeight == 0 {
		rm.config.RiskWeights.LiquidityWeight = 0.2
	}
	if rm.config.RiskWeights.MarketRiskWeight == 0 {
		rm.config.RiskWeights.MarketRiskWeight = 0.25
	}
	if rm.config.RiskWeights.CreditRiskWeight == 0 {
		rm.config.RiskWeights.CreditRiskWeight = 0.15
	}
	if rm.config.RiskWeights.OperationalWeight == 0 {
		rm.config.RiskWeights.OperationalWeight = 0.1
	}

	// 监控默认配置
	if rm.config.Monitoring.AlertThresholds == nil {
		rm.config.Monitoring.AlertThresholds = map[string]float64{
			"high_risk":  80.0,
			"critical":   90.0,
			"drawdown":   0.15,
			"volatility": 0.3,
		}
	}
	if rm.config.Monitoring.MonitoringInterval == 0 {
		rm.config.Monitoring.MonitoringInterval = 5 * time.Minute
	}
	if rm.config.Monitoring.ReportInterval == 0 {
		rm.config.Monitoring.ReportInterval = 1 * time.Hour
	}
}

// initializeRiskMonitoring 初始化风险监控
func (rm *RiskManagement) initializeRiskMonitoring() {
	// 启动监控goroutine
	go rm.startRiskMonitoring()
}

// AssessRisk 评估风险
func (rm *RiskManagement) AssessRisk(ctx context.Context, symbol string) (*RiskProfile, error) {
	rm.dataMu.Lock()
	defer rm.dataMu.Unlock()

	// 检查缓存
	if profile, exists := rm.riskData[symbol]; exists {
		// 检查是否需要更新
		if time.Since(profile.LastUpdated) < rm.config.Assessment.UpdateInterval {
			return profile, nil
		}
	}

	// 重新评估风险
	profile, err := rm.riskAssessor.Assess(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("风险评估失败: %w", err)
	}

	// 缓存结果
	rm.riskData[symbol] = profile

	return profile, nil
}

// MakeRiskDecision 做出风险决策
func (rm *RiskManagement) MakeRiskDecision(ctx context.Context, symbol string, requestedPosition float64) (*RiskDecision, error) {
	// 获取风险档案
	profile, err := rm.AssessRisk(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("获取风险档案失败: %w", err)
	}

	// 应用风险控制策略
	decision := rm.riskController.MakeDecision(profile, requestedPosition)

	// 检查是否需要触发告警
	rm.checkAndTriggerAlerts(profile)

	return decision, nil
}

// GetRiskReport 生成风险报告
func (rm *RiskManagement) GetRiskReport(ctx context.Context) (*RiskReport, error) {
	rm.dataMu.RLock()
	defer rm.dataMu.RUnlock()

	report := &RiskReport{
		GeneratedAt: time.Now(),
		Symbols:     make([]RiskSummary, 0),
		OverallRisk: RiskSummary{},
		Alerts:      make([]RiskAlert, 0),
	}

	totalRiskScore := 0.0
	highRiskCount := 0
	criticalRiskCount := 0

	// 汇总所有符号的风险
	for symbol, profile := range rm.riskData {
		summary := RiskSummary{
			Symbol:      symbol,
			RiskScore:   profile.RiskScore,
			RiskLevel:   profile.RiskLevel,
			LastUpdated: profile.LastUpdated,
		}
		report.Symbols = append(report.Symbols, summary)

		totalRiskScore += profile.RiskScore
		if profile.RiskLevel == RiskLevelHigh {
			highRiskCount++
		}
		if profile.RiskLevel == RiskLevelCritical {
			criticalRiskCount++
		}

		// 添加活跃告警
		for _, alert := range profile.Alerts {
			if time.Since(alert.Timestamp) < 24*time.Hour { // 最近24小时的告警
				report.Alerts = append(report.Alerts, alert)
			}
		}
	}

	// 计算整体风险
	if len(report.Symbols) > 0 {
		report.OverallRisk.RiskScore = totalRiskScore / float64(len(report.Symbols))
		report.OverallRisk.Symbol = "PORTFOLIO"
		report.OverallRisk.LastUpdated = time.Now()

		// 确定整体风险等级
		if criticalRiskCount > 0 {
			report.OverallRisk.RiskLevel = RiskLevelCritical
		} else if highRiskCount > len(report.Symbols)/3 {
			report.OverallRisk.RiskLevel = RiskLevelHigh
		} else if report.OverallRisk.RiskScore > rm.config.Assessment.RiskThreshold {
			report.OverallRisk.RiskLevel = RiskLevelMedium
		} else {
			report.OverallRisk.RiskLevel = RiskLevelLow
		}
	}

	// 按风险分数排序
	sort.Slice(report.Symbols, func(i, j int) bool {
		return report.Symbols[i].RiskScore > report.Symbols[j].RiskScore
	})

	return report, nil
}

// UpdateRiskData 更新风险数据
func (rm *RiskManagement) UpdateRiskData(ctx context.Context, symbol string, position float64, pnl float64) error {
	rm.dataMu.Lock()
	defer rm.dataMu.Unlock()

	profile, exists := rm.riskData[symbol]
	if !exists {
		profile = &RiskProfile{
			Symbol:         symbol,
			RiskFactors:    RiskFactors{},
			PositionLimits: PositionLimits{},
			HistoricalRisk: make([]RiskHistory, 0),
		}
		rm.riskData[symbol] = profile
	}

	// 添加历史风险记录
	history := RiskHistory{
		Timestamp: time.Now(),
		RiskScore: profile.RiskScore,
		Position:  position,
		PnL:       pnl,
	}

	profile.HistoricalRisk = append(profile.HistoricalRisk, history)

	// 限制历史记录数量
	maxHistory := rm.config.Assessment.HistoryWindow
	if len(profile.HistoricalRisk) > maxHistory {
		profile.HistoricalRisk = profile.HistoricalRisk[len(profile.HistoricalRisk)-maxHistory:]
	}

	return nil
}

// SubscribeAlerts 订阅风险告警
func (rm *RiskManagement) SubscribeAlerts(subscriber RiskAlertSubscriber) {
	rm.riskMonitor.subscribersMu.Lock()
	defer rm.riskMonitor.subscribersMu.Unlock()

	rm.riskMonitor.subscribers = append(rm.riskMonitor.subscribers, subscriber)
}

// GetRiskStatistics 获取风险统计信息
func (rm *RiskManagement) GetRiskStatistics() map[string]interface{} {
	rm.dataMu.RLock()
	defer rm.dataMu.RUnlock()

	stats := map[string]interface{}{
		"total_symbols":       len(rm.riskData),
		"high_risk_count":     0,
		"critical_risk_count": 0,
		"avg_risk_score":      0.0,
		"last_updated":        time.Now(),
	}

	totalScore := 0.0
	highRiskCount := 0
	criticalRiskCount := 0

	for _, profile := range rm.riskData {
		totalScore += profile.RiskScore
		if profile.RiskLevel == RiskLevelHigh {
			highRiskCount++
		}
		if profile.RiskLevel == RiskLevelCritical {
			criticalRiskCount++
		}
	}

	if len(rm.riskData) > 0 {
		stats["avg_risk_score"] = totalScore / float64(len(rm.riskData))
	}
	stats["high_risk_count"] = highRiskCount
	stats["critical_risk_count"] = criticalRiskCount

	return stats
}

// startRiskMonitoring 启动风险监控
func (rm *RiskManagement) startRiskMonitoring() {
	ticker := time.NewTicker(rm.config.Monitoring.MonitoringInterval)
	defer ticker.Stop()

	reportTicker := time.NewTicker(rm.config.Monitoring.ReportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.performRiskMonitoring()
		case <-reportTicker.C:
			rm.generateRiskReport()
		}
	}
}

// performRiskMonitoring 执行风险监控
func (rm *RiskManagement) performRiskMonitoring() {
	ctx := context.Background()

	rm.dataMu.RLock()
	symbols := make([]string, 0, len(rm.riskData))
	for symbol := range rm.riskData {
		symbols = append(symbols, symbol)
	}
	rm.dataMu.RUnlock()

	// 检查每个符号的风险状态
	for _, symbol := range symbols {
		profile, err := rm.AssessRisk(ctx, symbol)
		if err != nil {
			log.Printf("[RiskMonitor] 风险评估失败 %s: %v", symbol, err)
			continue
		}

		rm.checkAndTriggerAlerts(profile)
	}
}

// generateRiskReport 生成风险报告
func (rm *RiskManagement) generateRiskReport() {
	ctx := context.Background()
	report, err := rm.GetRiskReport(ctx)
	if err != nil {
		log.Printf("[RiskMonitor] 生成风险报告失败: %v", err)
		return
	}

	log.Printf("[RiskMonitor] 风险报告生成 - 总计%d个符号，平均风险%.2f，告警%d个",
		len(report.Symbols), report.OverallRisk.RiskScore, len(report.Alerts))
}

// checkAndTriggerAlerts 检查并触发告警
func (rm *RiskManagement) checkAndTriggerAlerts(profile *RiskProfile) {
	alerts := rm.riskMonitor.checkThresholds(profile)

	for _, alert := range alerts {
		// 添加到档案
		profile.Alerts = append(profile.Alerts, alert)

		// 限制告警数量
		if len(profile.Alerts) > 100 {
			profile.Alerts = profile.Alerts[len(profile.Alerts)-100:]
		}

		// 通知订阅者
		rm.riskMonitor.notifySubscribers(alert)
	}
}

// ============================================================================
// 风险报告结构
// ============================================================================

// RiskReport 风险报告
type RiskReport struct {
	GeneratedAt time.Time
	Symbols     []RiskSummary
	OverallRisk RiskSummary
	Alerts      []RiskAlert
}

// RiskSummary 风险摘要
type RiskSummary struct {
	Symbol      string
	RiskScore   float64
	RiskLevel   RiskLevel
	LastUpdated time.Time
}

// =================== 高级风险管理API ===================

// GetAdvancedRiskMetricsAPI 获取高级风险指标API
func (s *Server) GetAdvancedRiskMetricsAPI(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "需要指定交易对符号"})
		return
	}

	ctx := c.Request.Context()

	// 获取市场数据
	marketData, err := s.getMarketDataForAlgorithm(ctx, "spot", time.Now())
	if err != nil {
		c.JSON(500, gin.H{"error": "获取市场数据失败", "details": err.Error()})
		return
	}

	// 计算高级风险指标
	metrics, err := s.calculateAdvancedRiskMetrics(ctx, symbol, marketData)
	if err != nil {
		c.JSON(500, gin.H{"error": "计算风险指标失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"metrics":          metrics,
		"calculation_time": time.Now().Unix(),
	})
}

// PerformStressTestAPI 执行压力测试API
func (s *Server) PerformStressTestAPI(c *gin.Context) {
	var req struct {
		Symbol    string           `json:"symbol" binding:"required"`
		Scenarios []StressScenario `json:"scenarios"`
		TimeRange string           `json:"time_range"` // "1d", "7d", "30d"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// 设置默认场景
	if len(req.Scenarios) == 0 {
		req.Scenarios = []StressScenario{
			{Name: "轻度下跌", Description: "价格下跌10%", Shock: -0.1, Probability: 0.3},
			{Name: "中度下跌", Description: "价格下跌20%", Shock: -0.2, Probability: 0.15},
			{Name: "重度下跌", Description: "价格下跌30%", Shock: -0.3, Probability: 0.05},
			{Name: "闪电崩盘", Description: "价格下跌50%", Shock: -0.5, Probability: 0.01},
		}
	}

	ctx := c.Request.Context()

	// 获取历史价格数据
	prices, err := s.getHistoricalPrices(ctx, req.Symbol, req.TimeRange)
	if err != nil {
		c.JSON(500, gin.H{"error": "获取历史价格失败", "details": err.Error()})
		return
	}

	// 执行压力测试
	results := s.riskManagement.riskAssessor.calculateStressTest(prices, req.Scenarios)

	c.JSON(200, gin.H{
		"symbol":              req.Symbol,
		"time_range":          req.TimeRange,
		"stress_test_results": results,
		"scenarios_tested":    len(req.Scenarios),
		"test_timestamp":      time.Now().Unix(),
	})
}

// OptimizePortfolioAPI 投资组合优化API
func (s *Server) OptimizePortfolioAPI(c *gin.Context) {
	var req struct {
		Symbols      []string           `json:"symbols" binding:"required"`
		TimeRange    string             `json:"time_range"`
		TargetReturn float64            `json:"target_return"`
		Constraints  map[string]float64 `json:"constraints"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	if len(req.Symbols) == 0 {
		c.JSON(400, gin.H{"error": "至少需要指定一个交易对"})
		return
	}

	ctx := c.Request.Context()

	// 获取历史收益数据
	returns := make(map[string][]float64)
	for _, symbol := range req.Symbols {
		historicalReturns, err := s.getHistoricalReturns(ctx, symbol, req.TimeRange)
		if err != nil {
			log.Printf("[Portfolio] 获取%s历史收益失败: %v", symbol, err)
			continue
		}
		returns[symbol] = historicalReturns
	}

	if len(returns) == 0 {
		c.JSON(500, gin.H{"error": "无法获取任何历史收益数据"})
		return
	}

	// 计算当前权重（等权重）
	weights := make(map[string]float64)
	weight := 1.0 / float64(len(returns))
	for symbol := range returns {
		weights[symbol] = weight
	}

	// 计算当前组合风险
	portfolioRisk, err := s.riskManagement.riskController.CalculatePortfolioRisk(weights, returns)
	if err != nil {
		c.JSON(500, gin.H{"error": "计算组合风险失败", "details": err.Error()})
		return
	}

	// 优化投资组合
	optimalWeights, err := s.riskManagement.riskController.OptimizePortfolio(req.TargetReturn, returns, req.Constraints)
	if err != nil {
		c.JSON(500, gin.H{"error": "投资组合优化失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"current_portfolio":      portfolioRisk,
		"optimal_weights":        optimalWeights,
		"target_return":          req.TargetReturn,
		"optimization_timestamp": time.Now().Unix(),
	})
}

// GetRiskBudgetAPI 获取风险预算API
func (s *Server) GetRiskBudgetAPI(c *gin.Context) {
	var req struct {
		Symbols     []string           `json:"symbols" binding:"required"`
		Weights     map[string]float64 `json:"weights"`
		TotalBudget float64            `json:"total_budget"`
		TimeRange   string             `json:"time_range"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 获取历史收益数据
	returns := make(map[string][]float64)
	for _, symbol := range req.Symbols {
		historicalReturns, err := s.getHistoricalReturns(ctx, symbol, req.TimeRange)
		if err != nil {
			log.Printf("[RiskBudget] 获取%s历史收益失败: %v", symbol, err)
			continue
		}
		returns[symbol] = historicalReturns
	}

	// 计算风险预算
	riskBudget := s.calculateRiskBudget(req.Weights, returns, req.TotalBudget)

	c.JSON(200, gin.H{
		"risk_budget":      riskBudget,
		"total_budget":     req.TotalBudget,
		"assets_count":     len(req.Symbols),
		"calculation_time": time.Now().Unix(),
	})
}

// calculateAdvancedRiskMetrics 计算高级风险指标
func (s *Server) calculateAdvancedRiskMetrics(ctx context.Context, symbol string, marketData []MarketDataPoint) (*AdvancedRiskMetrics, error) {
	metrics := &AdvancedRiskMetrics{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	// 筛选指定symbol的数据
	var symbolData []MarketDataPoint
	var prices []float64
	var returns []float64

	for _, data := range marketData {
		if data.Symbol == symbol {
			symbolData = append(symbolData, data)
			prices = append(prices, data.Price)

			// 计算收益率
			if len(prices) > 1 {
				ret := (prices[len(prices)-1] - prices[len(prices)-2]) / prices[len(prices)-2]
				returns = append(returns, ret)
			}
		}
	}

	if len(prices) < 30 {
		return nil, fmt.Errorf("数据点不足，至少需要30个数据点")
	}

	// 计算传统风险指标
	metrics.Volatility = s.riskManagement.riskAssessor.calculateVolatility(returns)
	metrics.MaxDrawdown = s.riskManagement.riskAssessor.calculateMaxDrawdown(prices)
	metrics.SharpeRatio = s.riskManagement.riskAssessor.calculateSharpeRatio(returns, 0.02) // 2%无风险利率
	metrics.SortinoRatio = s.riskManagement.riskAssessor.calculateSortinoRatio(returns, 0.02)

	// 计算VaR指标
	if vaR95, err := s.riskManagement.riskAssessor.calculateVaR(returns, 0.95); err == nil {
		metrics.VaR95 = vaR95
	}
	if vaR99, err := s.riskManagement.riskAssessor.calculateVaR(returns, 0.99); err == nil {
		metrics.VaR99 = vaR99
	}

	// 计算贝塔系数（使用BTC作为市场基准）
	marketReturns := s.getMarketReturns(marketData)
	metrics.Beta = s.riskManagement.riskAssessor.calculateBeta(returns, marketReturns)

	// 执行压力测试
	scenarios := []StressScenario{
		{Name: "轻度压力", Description: "-10%冲击", Shock: -0.1},
		{Name: "中度压力", Description: "-20%冲击", Shock: -0.2},
		{Name: "重度压力", Description: "-30%冲击", Shock: -0.3},
	}
	metrics.StressTestResults = s.riskManagement.riskAssessor.calculateStressTest(prices, scenarios)

	return metrics, nil
}
