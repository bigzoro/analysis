package server

import (
	pdb "analysis/internal/db"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// 策略请求结构
type createStrategyReq struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Conditions  pdb.StrategyConditions `json:"conditions" binding:"required"`
}

type updateStrategyReq struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  pdb.StrategyConditions `json:"conditions"`
}

// 创建策略
func (s *Server) CreateTradingStrategy(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req createStrategyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	// 验证symbol_whitelist是否为有效的JSON（Gin已经自动转换为了datatypes.JSON）
	if len(req.Conditions.SymbolWhitelist) == 0 {
		// 设置为空数组
		emptyArray, _ := json.Marshal([]string{})
		req.Conditions.SymbolWhitelist = datatypes.JSON(emptyArray)
	}

	// 验证symbol_blacklist是否为有效的JSON（Gin已经自动转换为了datatypes.JSON）
	if len(req.Conditions.SymbolBlacklist) == 0 {
		// 设置为空数组
		emptyArray, _ := json.Marshal([]string{})
		req.Conditions.SymbolBlacklist = datatypes.JSON(emptyArray)
	}

	strategy := &pdb.TradingStrategy{
		UserID:      uid,
		Name:        req.Name,
		Description: req.Description,
		Conditions:  req.Conditions,
	}

	if err := pdb.CreateTradingStrategy(s.db.DB(), strategy); err != nil {
		s.DatabaseError(c, "创建策略", err)
		return
	}

	// 转换资金费率为前端显示格式（小数→百分比）
	responseData := *strategy // 复制一份数据
	responseData.Conditions.MinFundingRate *= 100
	responseData.Conditions.FuturesPriceShortMinFundingRate *= 100

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// 更新策略
func (s *Server) UpdateTradingStrategy(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的策略ID")
		return
	}

	var req updateStrategyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	// 获取现有策略
	strategy, err := pdb.GetTradingStrategy(s.db.DB(), uid, uint(strategyID))
	if err != nil {
		s.DatabaseError(c, "获取策略", err)
		return
	}

	// 更新字段
	if req.Name != "" {
		strategy.Name = req.Name
	}
	if req.Description != "" {
		strategy.Description = req.Description
	}

	// ========== 基础条件 ==========
	strategy.Conditions.SpotContract = req.Conditions.SpotContract

	// ========== 交易配置 ==========
	strategy.Conditions.AllowedDirections = req.Conditions.AllowedDirections
	strategy.Conditions.EnableLeverage = req.Conditions.EnableLeverage
	strategy.Conditions.DefaultLeverage = req.Conditions.DefaultLeverage
	strategy.Conditions.MaxLeverage = req.Conditions.MaxLeverage
	strategy.Conditions.SkipHeldPositions = req.Conditions.SkipHeldPositions
	strategy.Conditions.SkipCloseOrdersWithin24Hours = req.Conditions.SkipCloseOrdersWithin24Hours
	strategy.Conditions.SkipCloseOrdersHours = req.Conditions.SkipCloseOrdersHours
	strategy.Conditions.ProfitScalingEnabled = req.Conditions.ProfitScalingEnabled
	strategy.Conditions.ProfitScalingPercent = req.Conditions.ProfitScalingPercent
	strategy.Conditions.ProfitScalingAmount = req.Conditions.ProfitScalingAmount
	strategy.Conditions.ProfitScalingMaxCount = req.Conditions.ProfitScalingMaxCount

	// 整体仓位止盈止损
	strategy.Conditions.OverallStopLossEnabled = req.Conditions.OverallStopLossEnabled
	strategy.Conditions.OverallStopLossPercent = req.Conditions.OverallStopLossPercent
	strategy.Conditions.OverallTakeProfitPercent = req.Conditions.OverallTakeProfitPercent

	// ========== 传统交易策略 ==========
	strategy.Conditions.NoShortBelowMarketCap = req.Conditions.NoShortBelowMarketCap
	strategy.Conditions.MarketCapLimitShort = req.Conditions.MarketCapLimitShort
	strategy.Conditions.ShortOnGainers = req.Conditions.ShortOnGainers
	strategy.Conditions.GainersRankLimit = req.Conditions.GainersRankLimit
	strategy.Conditions.ShortMultiplier = req.Conditions.ShortMultiplier
	strategy.Conditions.LongOnSmallGainers = req.Conditions.LongOnSmallGainers
	strategy.Conditions.MarketCapLimitLong = req.Conditions.MarketCapLimitLong
	strategy.Conditions.GainersRankLimitLong = req.Conditions.GainersRankLimitLong
	strategy.Conditions.LongMultiplier = req.Conditions.LongMultiplier

	// ========== 资金费率过滤 ==========
	strategy.Conditions.FundingRateFilterEnabled = req.Conditions.FundingRateFilterEnabled
	// 转换资金费率：前端输入的百分比格式转换为小数格式存储
	if req.Conditions.MinFundingRate > 0.01 || req.Conditions.MinFundingRate < -0.01 {
		strategy.Conditions.MinFundingRate = req.Conditions.MinFundingRate / 100
	} else {
		strategy.Conditions.MinFundingRate = req.Conditions.MinFundingRate
	}

	// ========== 合约涨幅开空策略 ==========
	strategy.Conditions.FuturesPriceShortStrategyEnabled = req.Conditions.FuturesPriceShortStrategyEnabled
	strategy.Conditions.FuturesPriceShortMaxRank = req.Conditions.FuturesPriceShortMaxRank
	// 转换资金费率：前端输入的百分比格式转换为小数格式存储
	if req.Conditions.FuturesPriceShortMinFundingRate > 0.01 || req.Conditions.FuturesPriceShortMinFundingRate < -0.01 {
		strategy.Conditions.FuturesPriceShortMinFundingRate = req.Conditions.FuturesPriceShortMinFundingRate / 100
	} else {
		strategy.Conditions.FuturesPriceShortMinFundingRate = req.Conditions.FuturesPriceShortMinFundingRate
	}
	strategy.Conditions.FuturesPriceShortLeverage = req.Conditions.FuturesPriceShortLeverage
	// 转换市值：前端输入的万元格式转换为美元格式存储
	strategy.Conditions.FuturesPriceShortMinMarketCap = req.Conditions.FuturesPriceShortMinMarketCap * 10000

	// ========== 技术指标策略 ==========
	strategy.Conditions.MovingAverageEnabled = req.Conditions.MovingAverageEnabled
	strategy.Conditions.MASignalMode = req.Conditions.MASignalMode
	strategy.Conditions.MAType = req.Conditions.MAType
	strategy.Conditions.ShortMAPeriod = req.Conditions.ShortMAPeriod
	strategy.Conditions.LongMAPeriod = req.Conditions.LongMAPeriod
	strategy.Conditions.MACrossSignal = req.Conditions.MACrossSignal
	strategy.Conditions.MATrendFilter = req.Conditions.MATrendFilter
	strategy.Conditions.MATrendDirection = req.Conditions.MATrendDirection

	// ========== 均值回归策略 ==========
	strategy.Conditions.MeanReversionEnabled = req.Conditions.MeanReversionEnabled
	strategy.Conditions.MeanReversionMode = req.Conditions.MeanReversionMode
	strategy.Conditions.MeanReversionSubMode = req.Conditions.MeanReversionSubMode
	strategy.Conditions.MRBollingerBandsEnabled = req.Conditions.MRBollingerBandsEnabled
	strategy.Conditions.MRRSIEnabled = req.Conditions.MRRSIEnabled
	strategy.Conditions.MRPriceChannelEnabled = req.Conditions.MRPriceChannelEnabled
	strategy.Conditions.MRPeriod = req.Conditions.MRPeriod
	strategy.Conditions.MRBollingerMultiplier = req.Conditions.MRBollingerMultiplier
	strategy.Conditions.MRRSIOversold = req.Conditions.MRRSIOversold
	strategy.Conditions.MRRSIOverbought = req.Conditions.MRRSIOverbought
	strategy.Conditions.MRPriceChannelEnabled = req.Conditions.MRPriceChannelEnabled
	strategy.Conditions.MRChannelPeriod = req.Conditions.MRChannelPeriod
	strategy.Conditions.MRMinReversionStrength = req.Conditions.MRMinReversionStrength
	strategy.Conditions.MRSignalMode = req.Conditions.MRSignalMode
	// 设置权重默认值（如果前端未设置）
	if req.Conditions.MRWeightBollingerBands == 0 {
		strategy.Conditions.MRWeightBollingerBands = 1.0 // 布林带权重默认1.0
	} else {
		strategy.Conditions.MRWeightBollingerBands = req.Conditions.MRWeightBollingerBands
	}

	if req.Conditions.MRWeightRSI == 0 {
		strategy.Conditions.MRWeightRSI = 0.8 // RSI权重默认0.8
	} else {
		strategy.Conditions.MRWeightRSI = req.Conditions.MRWeightRSI
	}

	if req.Conditions.MRWeightPriceChannel == 0 {
		strategy.Conditions.MRWeightPriceChannel = 0.6 // 价格通道权重默认0.6
	} else {
		strategy.Conditions.MRWeightPriceChannel = req.Conditions.MRWeightPriceChannel
	}

	if req.Conditions.MRWeightTimeDecay == 0 {
		strategy.Conditions.MRWeightTimeDecay = 0.4 // 时间衰减权重默认0.4
	} else {
		strategy.Conditions.MRWeightTimeDecay = req.Conditions.MRWeightTimeDecay
	}

	// 增强功能开关
	strategy.Conditions.MarketEnvironmentDetection = req.Conditions.MarketEnvironmentDetection
	strategy.Conditions.IntelligentWeights = req.Conditions.IntelligentWeights
	strategy.Conditions.PerformanceMonitoring = req.Conditions.PerformanceMonitoring
	strategy.Conditions.AdvancedRiskManagement = req.Conditions.AdvancedRiskManagement

	// 候选币种筛选标准
	strategy.Conditions.MRCandidateMinOscillation = req.Conditions.MRCandidateMinOscillation
	strategy.Conditions.MRCandidateMinLiquidity = req.Conditions.MRCandidateMinLiquidity
	strategy.Conditions.MRCandidateMaxVolatility = req.Conditions.MRCandidateMaxVolatility

	// 市场环境检测参数
	strategy.Conditions.MREnvTrendThreshold = req.Conditions.MREnvTrendThreshold
	strategy.Conditions.MREnvVolatilityThreshold = req.Conditions.MREnvVolatilityThreshold
	strategy.Conditions.MREnvOscillationThreshold = req.Conditions.MREnvOscillationThreshold

	// 自适应参数
	strategy.Conditions.MRAutoAdjustPeriod = req.Conditions.MRAutoAdjustPeriod
	strategy.Conditions.MRAutoAdjustMultiplier = req.Conditions.MRAutoAdjustMultiplier
	strategy.Conditions.MRAutoAdjustThresholds = req.Conditions.MRAutoAdjustThresholds

	// 风险控制参数
	strategy.Conditions.MRMaxPositionSize = req.Conditions.MRMaxPositionSize
	strategy.Conditions.MRStopLossMultiplier = req.Conditions.MRStopLossMultiplier
	strategy.Conditions.MRTakeProfitMultiplier = req.Conditions.MRTakeProfitMultiplier
	strategy.Conditions.MRMaxHoldHours = req.Conditions.MRMaxHoldHours
	strategy.Conditions.MRMaxDailyLoss = req.Conditions.MRMaxDailyLoss

	// 信号增强选项
	strategy.Conditions.MRRequireMultipleSignals = req.Conditions.MRRequireMultipleSignals
	strategy.Conditions.MRRequireVolumeConfirmation = req.Conditions.MRRequireVolumeConfirmation
	strategy.Conditions.MRRequireTimeFilter = req.Conditions.MRRequireTimeFilter
	strategy.Conditions.MRRequireMarketEnvironmentFilter = req.Conditions.MRRequireMarketEnvironmentFilter

	// ========== 套利策略 ==========
	strategy.Conditions.CrossExchangeArbEnabled = req.Conditions.CrossExchangeArbEnabled
	strategy.Conditions.PriceDiffThreshold = req.Conditions.PriceDiffThreshold
	strategy.Conditions.MinArbAmount = req.Conditions.MinArbAmount
	strategy.Conditions.SpotFutureArbEnabled = req.Conditions.SpotFutureArbEnabled
	strategy.Conditions.BasisThreshold = req.Conditions.BasisThreshold
	strategy.Conditions.FundingRateThreshold = req.Conditions.FundingRateThreshold
	strategy.Conditions.TriangleArbEnabled = req.Conditions.TriangleArbEnabled
	strategy.Conditions.TriangleThreshold = req.Conditions.TriangleThreshold
	strategy.Conditions.BaseSymbols = req.Conditions.BaseSymbols
	strategy.Conditions.StatArbEnabled = req.Conditions.StatArbEnabled
	strategy.Conditions.CointegrationPeriod = req.Conditions.CointegrationPeriod
	strategy.Conditions.ZscoreThreshold = req.Conditions.ZscoreThreshold
	strategy.Conditions.StatArbPairs = req.Conditions.StatArbPairs
	strategy.Conditions.FuturesSpotArbEnabled = req.Conditions.FuturesSpotArbEnabled
	strategy.Conditions.ExpiryThreshold = req.Conditions.ExpiryThreshold
	strategy.Conditions.SpotFutureSpread = req.Conditions.SpotFutureSpread

	// ========== 网格交易策略 ==========
	strategy.Conditions.GridTradingEnabled = req.Conditions.GridTradingEnabled
	strategy.Conditions.GridUpperPrice = req.Conditions.GridUpperPrice
	strategy.Conditions.GridLowerPrice = req.Conditions.GridLowerPrice
	strategy.Conditions.GridLevels = req.Conditions.GridLevels
	strategy.Conditions.GridProfitPercent = req.Conditions.GridProfitPercent
	strategy.Conditions.GridInvestmentAmount = req.Conditions.GridInvestmentAmount
	strategy.Conditions.GridRebalanceEnabled = req.Conditions.GridRebalanceEnabled
	strategy.Conditions.GridStopLossEnabled = req.Conditions.GridStopLossEnabled
	strategy.Conditions.GridStopLossPercent = req.Conditions.GridStopLossPercent

	// ========== 风险控制 ==========
	strategy.Conditions.MaxPositionSize = req.Conditions.MaxPositionSize
	strategy.Conditions.PositionSizeStep = req.Conditions.PositionSizeStep
	strategy.Conditions.DynamicPositioning = req.Conditions.DynamicPositioning
	strategy.Conditions.EnableStopLoss = req.Conditions.EnableStopLoss
	strategy.Conditions.StopLossPercent = req.Conditions.StopLossPercent
	strategy.Conditions.EnableTakeProfit = req.Conditions.EnableTakeProfit
	strategy.Conditions.TakeProfitPercent = req.Conditions.TakeProfitPercent
	strategy.Conditions.EnableMarginLossStopLoss = req.Conditions.EnableMarginLossStopLoss
	strategy.Conditions.MarginLossStopLossPercent = req.Conditions.MarginLossStopLossPercent
	strategy.Conditions.EnableMarginProfitTakeProfit = req.Conditions.EnableMarginProfitTakeProfit
	strategy.Conditions.MarginProfitTakeProfitPercent = req.Conditions.MarginProfitTakeProfitPercent
	strategy.Conditions.VolatilityFilterEnabled = req.Conditions.VolatilityFilterEnabled
	strategy.Conditions.MaxVolatility = req.Conditions.MaxVolatility
	strategy.Conditions.VolatilityPeriod = req.Conditions.VolatilityPeriod

	// ========== 市场时机 ==========
	strategy.Conditions.TimeFilterEnabled = req.Conditions.TimeFilterEnabled
	strategy.Conditions.StartHour = req.Conditions.StartHour
	strategy.Conditions.EndHour = req.Conditions.EndHour
	strategy.Conditions.WeekendTrading = req.Conditions.WeekendTrading
	strategy.Conditions.MarketRegimeFilterEnabled = req.Conditions.MarketRegimeFilterEnabled
	strategy.Conditions.MarketRegimeThreshold = req.Conditions.MarketRegimeThreshold
	strategy.Conditions.PreferredRegime = req.Conditions.PreferredRegime

	// ========== 币种选择 ==========
	strategy.Conditions.UseSymbolWhitelist = req.Conditions.UseSymbolWhitelist
	strategy.Conditions.SymbolWhitelist = req.Conditions.SymbolWhitelist
	strategy.Conditions.UseSymbolBlacklist = req.Conditions.UseSymbolBlacklist
	strategy.Conditions.SymbolBlacklist = req.Conditions.SymbolBlacklist

	if err := pdb.UpdateTradingStrategy(s.db.DB(), strategy); err != nil {
		s.DatabaseError(c, "更新策略", err)
		return
	}

	// 转换资金费率为前端显示格式（小数→百分比）
	responseData := *strategy // 复制一份数据
	responseData.Conditions.MinFundingRate *= 100
	responseData.Conditions.FuturesPriceShortMinFundingRate *= 100
	// 转换市值为前端显示格式（美元→万元）
	responseData.Conditions.FuturesPriceShortMinMarketCap /= 10000

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// 删除策略
func (s *Server) DeleteTradingStrategy(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的策略ID")
		return
	}

	if err := pdb.DeleteTradingStrategy(s.db.DB(), uid, uint(strategyID)); err != nil {
		s.DatabaseError(c, "删除策略", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "策略删除成功",
	})
}

// 获取单个策略
func (s *Server) GetTradingStrategy(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的策略ID")
		return
	}

	strategy, err := pdb.GetTradingStrategy(s.db.DB(), uid, uint(strategyID))
	if err != nil {
		s.DatabaseError(c, "获取策略", err)
		return
	}

	// 转换资金费率为前端显示格式（小数→百分比）
	responseData := *strategy // 复制一份数据
	responseData.Conditions.MinFundingRate *= 100
	responseData.Conditions.FuturesPriceShortMinFundingRate *= 100
	// 转换市值为前端显示格式（美元→万元）
	responseData.Conditions.FuturesPriceShortMinMarketCap /= 10000

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// 获取策略列表
func (s *Server) ListTradingStrategies(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategies, err := pdb.ListTradingStrategies(s.db.DB(), uid)
	if err != nil {
		s.DatabaseError(c, "获取策略列表", err)
		return
	}

	// 转换资金费率为前端显示格式（小数→百分比）
	for i := range strategies {
		strategies[i].Conditions.MinFundingRate *= 100
		strategies[i].Conditions.FuturesPriceShortMinFundingRate *= 100
		// 转换市值为前端显示格式（美元→万元）
		strategies[i].Conditions.FuturesPriceShortMinMarketCap /= 10000
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    strategies,
	})
}

// ===== 策略执行相关API =====

// 策略执行请求结构
type startStrategyExecutionReq struct {
	StrategyID     uint    `json:"strategy_id" binding:"required"`
	RunInterval    int     `json:"run_interval"`     // 可选，运行间隔（分钟）
	MaxRuns        int     `json:"max_runs"`         // 可选，最大运行次数，0表示无限
	AutoStop       bool    `json:"auto_stop"`        // 可选，执行后自动停止
	CreateOrders   bool    `json:"create_orders"`    // 可选，是否自动创建订单
	ExecutionDelay int     `json:"execution_delay"`  // 可选，执行延迟（秒）
	PerOrderAmount float64 `json:"per_order_amount"` // 可选，每一单的金额（U单位）
}

type updateStrategyExecutionReq struct {
	Status        string  `json:"status"`
	TotalOrders   int     `json:"total_orders"`
	SuccessOrders int     `json:"success_orders"`
	FailedOrders  int     `json:"failed_orders"`
	TotalPnL      float64 `json:"total_pnl"`
	Logs          string  `json:"logs"`
}

// 开始策略执行
func (s *Server) StartStrategyExecution(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req startStrategyExecutionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	// 检查策略是否存在且属于当前用户
	strategy, err := pdb.GetTradingStrategy(s.db.DB(), uid, req.StrategyID)
	if err != nil {
		s.DatabaseError(c, "获取策略", err)
		return
	}

	// 检查策略是否已经在运行
	if strategy.IsRunning {
		s.ValidationError(c, "strategy_id", "策略正在运行中")
		return
	}

	// 检查是否有pending状态的执行记录，防止重复启动
	var pendingCount int64
	if err := s.db.DB().Model(&pdb.StrategyExecution{}).Where("strategy_id = ? AND status = ?", req.StrategyID, "pending").Count(&pendingCount).Error; err != nil {
		s.DatabaseError(c, "检查执行状态", err)
		return
	}

	if pendingCount > 0 {
		s.ValidationError(c, "strategy_id", "策略已有待执行任务，请等待完成后再启动")
		return
	}

	// 设置默认参数
	if req.RunInterval <= 0 {
		req.RunInterval = 60 // 默认60分钟
	}
	if req.MaxRuns < 0 {
		req.MaxRuns = 0 // 0表示无限
	}

	// 创建执行记录
	execution := &pdb.StrategyExecution{
		StrategyID:     req.StrategyID,
		UserID:         uid,
		Status:         "pending",
		CurrentStep:    "初始化",
		StepProgress:   0,
		TotalProgress:  0,
		RunInterval:    req.RunInterval,
		MaxRuns:        req.MaxRuns,
		AutoStop:       req.AutoStop,
		CreateOrders:   req.CreateOrders,
		ExecutionDelay: req.ExecutionDelay,
		PerOrderAmount: req.PerOrderAmount,
		RunCount:       0,
	}

	// 记录启动参数
	log.Printf("[StrategyStart] 启动参数: CreateOrders=%v, RunInterval=%d, MaxRuns=%d, ExecutionDelay=%d, PerOrderAmount=%.2f",
		req.CreateOrders, req.RunInterval, req.MaxRuns, req.ExecutionDelay, req.PerOrderAmount)

	if err := pdb.StartStrategyExecution(s.db.DB(), execution); err != nil {
		s.DatabaseError(c, "开始策略执行", err)
		return
	}

	// 记录初始日志
	pdb.AppendStrategyExecutionLog(s.db.DB(), execution.ID, "策略执行初始化完成，开始立即执行")

	// 创建初始步骤记录
	now := time.Now()
	initStep := &pdb.StrategyExecutionStep{
		ExecutionID: execution.ID,
		StepName:    "策略初始化",
		StepType:    "initialization",
		Status:      "completed",
		StartTime:   &now,
		EndTime:     &now,
		Duration:    0,
		Result:      "策略执行已初始化，等待调度器处理",
	}
	pdb.CreateStrategyExecutionStep(s.db.DB(), initStep)

	// 更新策略运行状态
	if req.RunInterval <= 0 {
		req.RunInterval = 60 // 默认60分钟
	}
	if err := pdb.UpdateStrategyRunningStatus(s.db.DB(), req.StrategyID, true); err != nil {
		s.DatabaseError(c, "更新策略状态", err)
		return
	}

	// 更新策略运行间隔，并设置last_run_at为过去时间，确保调度器能继续周期执行
	updates := map[string]interface{}{
		"run_interval": req.RunInterval,
		// 设置last_run_at为当前时间，这样调度器会基于此时间计算下次执行
		// 立即执行完成后，调度器会按RunInterval周期继续执行
		"last_run_at": time.Now(),
	}
	s.db.DB().Model(&pdb.TradingStrategy{}).Where("id = ?", req.StrategyID).Updates(updates)

	// 🚀 立即触发一次策略执行（异步，避免阻塞API响应）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[StrategyStart] Panic in immediate execution: %v", r)
			}
		}()

		log.Printf("[StrategyStart] 立即执行策略 %d 的第一次运行", req.StrategyID)

		// 等待一小段时间，确保数据库事务完成
		time.Sleep(100 * time.Millisecond)

		// 最多等待5秒，等待OrderScheduler初始化完成
		maxWait := 50 // 50 * 100ms = 5秒
		for i := 0; i < maxWait; i++ {
			if s.orderScheduler != nil {
				log.Printf("[StrategyStart] OrderScheduler已准备好，开始立即执行")
				// 直接调用执行逻辑，而不是等待调度器检查
				s.orderScheduler.executeStrategy(strategy)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}

		log.Printf("[StrategyStart] OrderScheduler在5秒内未初始化完成，跳过立即执行，将由调度器在下次检查时执行")
	}()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"execution": execution,
		"message":   "策略执行已开始，正在进行首次运行",
	})
}

// 停止策略执行
func (s *Server) StopStrategyExecution(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "策略ID格式错误", err)
		return
	}

	// 检查策略是否存在且属于当前用户
	_, err = pdb.GetTradingStrategy(s.db.DB(), uid, uint(strategyID))
	if err != nil {
		s.DatabaseError(c, "获取策略", err)
		return
	}

	// 停止所有相关的执行记录
	executions, err := pdb.GetRunningStrategyExecutions(s.db.DB(), uid)
	if err != nil {
		s.DatabaseError(c, "获取运行中的执行", err)
		return
	}

	stoppedCount := 0
	for _, execution := range executions {
		if execution.StrategyID == uint(strategyID) {
			// 停止执行并记录日志
			if err := pdb.StopStrategyExecution(s.db.DB(), execution.ID); err != nil {
				log.Printf("[StrategyStop] Failed to stop execution %d: %v", execution.ID, err)
				continue // 跳过错误，继续停止其他执行
			}

			// 添加停止日志
			pdb.AppendStrategyExecutionLog(s.db.DB(), execution.ID, "策略被用户手动停止")
			stoppedCount++
		}
	}

	// 更新策略运行状态
	if err := pdb.UpdateStrategyRunningStatus(s.db.DB(), uint(strategyID), false); err != nil {
		s.DatabaseError(c, "更新策略状态", err)
		return
	}

	// 重置策略的加仓计数器，让用户可以重新开始
	// 重置策略级别的计数器（兼容旧逻辑）
	if err := s.db.DB().Model(&pdb.TradingStrategy{}).Where("id = ? AND user_id = ?", strategyID, uid).
		Update("profit_scaling_current_count", 0).Error; err != nil {
		log.Printf("[StrategyStop] Failed to reset strategy-level profit scaling counter for strategy %d: %v", strategyID, err)
		// 不返回错误，因为这不是关键操作
	}

	// 重置所有币种的加仓计数器
	if err := s.db.DB().Model(&pdb.TradingStrategy{}).Where("id = ? AND user_id = ?", strategyID, uid).
		Update("profit_scaling_symbol_counts", "{}").Error; err != nil {
		log.Printf("[StrategyStop] Failed to reset symbol-level profit scaling counters for strategy %d: %v", strategyID, err)
		// 不返回错误，因为这不是关键操作
	}

	// 记录策略停止事件
	log.Printf("[StrategyStop] Strategy %d stopped, %d executions terminated", strategyID, stoppedCount)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stopped": stoppedCount,
		"message": "策略执行已停止",
	})
}

// 获取策略执行记录
func (s *Server) ListStrategyExecutions(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Query("strategy_id")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var strategyID uint
	if strategyIDStr != "" {
		if id, err := strconv.ParseUint(strategyIDStr, 10, 32); err == nil {
			strategyID = uint(id)
		}
	}

	executions, err := pdb.ListStrategyExecutions(s.db.DB(), uid, strategyID, limit)
	if err != nil {
		s.DatabaseError(c, "获取执行记录", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    executions,
	})
}

// 获取策略执行详情
func (s *Server) GetStrategyExecution(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	executionIDStr := c.Param("execution_id")
	executionID, err := strconv.ParseUint(executionIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "执行ID格式错误", err)
		return
	}

	execution, err := pdb.GetStrategyExecution(s.db.DB(), uid, uint(executionID))
	if err != nil {
		s.DatabaseError(c, "获取执行详情", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    execution,
	})
}

// 获取策略健康状态
func (s *Server) GetStrategyHealth(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "策略ID格式错误", err)
		return
	}

	// 获取策略信息
	strategy, err := pdb.GetTradingStrategy(s.db.DB(), uid, uint(strategyID))
	if err != nil {
		s.DatabaseError(c, "获取策略", err)
		return
	}

	// 获取最近的执行记录
	var recentExecution pdb.StrategyExecution
	err = s.db.DB().Where("strategy_id = ? AND user_id = ?", strategyID, uid).
		Order("created_at desc").
		First(&recentExecution).Error

	health := gin.H{
		"strategy_id":    strategy.ID,
		"is_running":     strategy.IsRunning,
		"run_interval":   strategy.RunInterval,
		"last_run_at":    strategy.LastRunAt,
		"next_run_time":  nil,
		"status":         "unknown",
		"last_execution": nil,
	}

	// 计算下次运行时间
	if strategy.LastRunAt != nil {
		interval := time.Duration(strategy.RunInterval) * time.Minute
		nextRunTime := strategy.LastRunAt.Add(interval)
		health["next_run_time"] = nextRunTime

		// 判断状态
		now := time.Now()
		if strategy.IsRunning {
			if now.After(nextRunTime) {
				health["status"] = "pending_execution"
			} else {
				health["status"] = "waiting"
			}
		} else {
			health["status"] = "stopped"
		}
	} else if strategy.IsRunning {
		health["status"] = "never_executed"
		health["next_run_time"] = time.Now()
	}

	// 添加最近执行信息
	if err == nil {
		health["last_execution"] = gin.H{
			"id":           recentExecution.ID,
			"status":       recentExecution.Status,
			"start_time":   recentExecution.StartTime,
			"end_time":     recentExecution.EndTime,
			"duration":     recentExecution.Duration,
			"total_orders": recentExecution.TotalOrders,
			"win_rate":     recentExecution.WinRate,
		}

		// 如果有正在运行的执行，更新状态
		if recentExecution.Status == "running" {
			health["status"] = "executing"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    health,
	})
}

// 获取策略执行步骤详情
func (s *Server) GetStrategyExecutionSteps(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	executionIDStr := c.Param("execution_id")
	executionID, err := strconv.ParseUint(executionIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "执行ID格式错误", err)
		return
	}

	// 验证执行记录属于当前用户
	execution, err := pdb.GetStrategyExecution(s.db.DB(), uid, uint(executionID))
	if err != nil {
		s.DatabaseError(c, "获取执行记录", err)
		return
	}

	// 获取执行步骤
	steps, err := pdb.GetStrategyExecutionSteps(s.db.DB(), uint(executionID))
	if err != nil {
		s.DatabaseError(c, "获取执行步骤", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"execution": execution,
			"steps":     steps,
		},
	})
}

// 获取策略执行统计
func (s *Server) GetStrategyExecutionStats(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "策略ID格式错误", err)
		return
	}

	// 解析分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "5") // 默认5条记录

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 5
	}

	// 限制每页最大数量
	if pageSize > 50 {
		pageSize = 50
	}

	// 获取分页的执行记录
	executions, totalRecords, err := pdb.ListStrategyExecutionsPaged(s.db.DB(), uid, uint(strategyID), page, pageSize)
	if err != nil {
		s.DatabaseError(c, "获取执行记录", err)
		return
	}

	// 获取所有执行记录用于计算总体统计（这个可以优化，但为了保持兼容性暂时保留）
	allExecutions, err := pdb.ListStrategyExecutions(s.db.DB(), uid, uint(strategyID), 1000) // 获取足够多的记录用于统计
	if err != nil {
		s.DatabaseError(c, "获取统计数据", err)
		return
	}

	// 计算统计数据
	totalExecutions := len(allExecutions)
	var totalOrders, successOrders, failedOrders int
	var totalPnL, totalInvestment, currentValue float64
	var avgWinRate float64

	for _, execution := range allExecutions {
		// 重新计算基于实际订单成交状态的统计数据
		var orders []pdb.ScheduledOrder
		if err := s.db.DB().Where("execution_id = ?", execution.ID).Find(&orders).Error; err == nil {
			actualSuccessCount := 0
			actualFailCount := 0
			var recalculatedPnL, executionInvestment, executionCurrentValue float64

			for _, order := range orders {
				if order.Status == "filled" {
					actualSuccessCount++
					// 重新计算单个订单的盈亏
					if pnl, err := s.calculateOrderPnL(&order); err == nil {
						recalculatedPnL += pnl
					}

					// 计算投资金额（开仓时的价值）
					if order.AvgPrice != "" && order.ExecutedQty != "" {
						if entryPrice, err := strconv.ParseFloat(order.AvgPrice, 64); err == nil {
							if quantity, err := strconv.ParseFloat(order.ExecutedQty, 64); err == nil {
								investment := entryPrice * quantity
								executionInvestment += investment

								// 计算当前价值
								if order.Side == "BUY" {
									// 多头仓位：当前价格 × 数量
									if currentPrice, err := s.getCurrentPrice(context.Background(), order.Symbol, "futures"); err == nil {
										executionCurrentValue += currentPrice * quantity
									} else {
										// 如果获取当前价格失败，使用开仓价格作为近似值
										executionCurrentValue += investment
									}
								} else {
									// 空头仓位：由于是做空，当前价值 = 保证金 + 盈亏
									// 这里简化处理，假设保证金等于投资金额（实际应该根据杠杆计算）
									margin := investment / float64(order.Leverage)
									executionCurrentValue += margin + recalculatedPnL
								}
							}
						}
					}
				} else if order.Status == "failed" || order.Status == "cancelled" || order.Status == "rejected" {
					actualFailCount++
				}
			}

			// 更新统计数据
			executionTotalOrders := actualSuccessCount + actualFailCount
			totalOrders += executionTotalOrders
			successOrders += actualSuccessCount
			failedOrders += actualFailCount
			totalPnL += recalculatedPnL
			totalInvestment += executionInvestment
			currentValue += executionCurrentValue

			// 计算胜率和盈亏百分比
			if executionTotalOrders > 0 {
				executionWinRate := float64(actualSuccessCount) / float64(executionTotalOrders) * 100
				avgWinRate += executionWinRate

				// 计算盈亏百分比
				var executionPnlPercentage float64
				if executionInvestment > 0 {
					executionPnlPercentage = (recalculatedPnL / executionInvestment) * 100
				}

				// 如果数据库中的数据与实际不符，更新数据库
				if execution.TotalOrders != executionTotalOrders || execution.SuccessOrders != actualSuccessCount ||
					execution.FailedOrders != actualFailCount || execution.TotalPnL != recalculatedPnL ||
					execution.PnlPercentage != executionPnlPercentage || execution.TotalInvestment != executionInvestment ||
					execution.CurrentValue != executionCurrentValue {
					pdb.UpdateStrategyExecutionResultWithStats(s.db.DB(), execution.ID, executionTotalOrders, actualSuccessCount, actualFailCount, recalculatedPnL, executionWinRate, executionPnlPercentage, executionInvestment, executionCurrentValue)
					log.Printf("[StrategyStats] Updated execution %d stats: orders=%d, success=%d, failed=%d, pnl=%.8f, winRate=%.2f%%, pnlPct=%.2f%%, investment=%.8f, currentValue=%.8f",
						execution.ID, executionTotalOrders, actualSuccessCount, actualFailCount, recalculatedPnL, executionWinRate, executionPnlPercentage, executionInvestment, executionCurrentValue)
				}
			}
		} else {
			// 如果查询失败，使用数据库中已有的数据
			log.Printf("[StrategyStats] Failed to query orders for execution %d: %v, using cached stats", execution.ID, err)
			totalOrders += execution.TotalOrders
			successOrders += execution.SuccessOrders
			failedOrders += execution.FailedOrders
			totalPnL += execution.TotalPnL
			totalInvestment += execution.TotalInvestment
			currentValue += execution.CurrentValue
			avgWinRate += execution.WinRate
		}
	}

	if totalExecutions > 0 {
		avgWinRate /= float64(totalExecutions)
	}

	// 计算总体盈亏百分比
	var totalPnlPercentage float64
	if totalInvestment > 0 {
		totalPnlPercentage = (totalPnL / totalInvestment) * 100
	}

	// 计算分页信息
	totalPages := (int(totalRecords) + pageSize - 1) / pageSize

	stats := gin.H{
		"total_executions":     totalExecutions,
		"total_orders":         totalOrders,
		"success_orders":       successOrders,
		"failed_orders":        failedOrders,
		"total_pnl":            totalPnL,
		"total_pnl_percentage": totalPnlPercentage,
		"total_investment":     totalInvestment,
		"current_value":        currentValue,
		"avg_win_rate":         avgWinRate,
		"executions":           executions,
		"pagination": gin.H{
			"page":          page,
			"page_size":     pageSize,
			"total_pages":   totalPages,
			"total_records": totalRecords,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// 获取策略相关的订单记录
func (s *Server) GetStrategyOrders(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	strategyIDStr := c.Param("id")
	strategyID, err := strconv.ParseUint(strategyIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "策略ID格式错误", err)
		return
	}

	// 解析分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10") // 订单记录默认10条

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 限制每页最大数量
	if pageSize > 50 {
		pageSize = 50
	}

	// 获取该策略的订单记录
	orders, totalRecords, err := s.getStrategyOrdersPaged(uid, uint(strategyID), page, pageSize)
	if err != nil {
		s.DatabaseError(c, "获取策略订单记录", err)
		return
	}

	// 格式化订单数据
	enhancedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		operationType := getFuturesOperationType(order.Side, order.ReduceOnly)
		relatedOrders := s.getRelatedOrdersSummary(order)

		// 获取正确的成交数量
		executedQuantity := order.ExecutedQty
		if executedQuantity == "" {
			// 如果ExecutedQty为空，尝试从数据库直接查询executed_quantity字段
			var result struct {
				ExecutedQuantity string
			}
			s.db.DB().Table("scheduled_orders").Select("executed_quantity").Where("id = ?", order.ID).Scan(&result)
			executedQuantity = result.ExecutedQuantity
		}

		enhancedOrders[i] = gin.H{
			"id":                order.ID,
			"exchange":          order.Exchange,
			"testnet":           order.Testnet,
			"symbol":            order.Symbol,
			"side":              order.Side,
			"order_type":        order.OrderType,
			"quantity":          order.Quantity,
			"adjusted_quantity": order.AdjustedQuantity,
			"price":             order.Price,
			"leverage":          order.Leverage,
			"reduce_only":       order.ReduceOnly,
			"strategy_id":       order.StrategyID,
			"execution_id":      order.ExecutionID,
			"trigger_time":      order.TriggerTime,
			"status":            order.Status,
			"client_order_id":   order.ClientOrderId,
			"exchange_order_id": order.ExchangeOrderId,
			"executed_quantity": executedQuantity,
			"avg_price":         order.AvgPrice,
			"bracket_enabled":   order.BracketEnabled,
			"tp_percent":        order.TPPercent,
			"sl_percent":        order.SLPercent,
			"tp_price":          order.TPPrice,
			"sl_price":          order.SLPrice,
			"working_type":      order.WorkingType,
			"created_at":        order.CreatedAt,
			"operation_type":    operationType,
			"related_orders":    relatedOrders,
		}
	}

	// 计算分页信息
	totalPages := (int(totalRecords) + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"orders": enhancedOrders,
			"pagination": gin.H{
				"page":          page,
				"page_size":     pageSize,
				"total_pages":   totalPages,
				"total_records": totalRecords,
			},
		},
	})
}

// 分页获取策略相关的订单记录
func (s *Server) getStrategyOrdersPaged(userID, strategyID uint, page, pageSize int) ([]*pdb.ScheduledOrder, int64, error) {
	var orders []*pdb.ScheduledOrder
	var total int64

	offset := (page - 1) * pageSize
	// 只查询由策略自动执行创建的订单（有execution_id的订单）
	query := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("user_id = ? AND strategy_id = ? AND execution_id IS NOT NULL", userID, strategyID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := s.db.DB().Where("user_id = ? AND strategy_id = ? AND execution_id IS NOT NULL", userID, strategyID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&orders).Error

	return orders, total, err
}

// DELETE /strategies/executions/:id
func (s *Server) DeleteStrategyExecution(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	idStr := c.Param("execution_id")

	// 解析 ID
	executionID64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "execution_id", "无效的执行记录ID")
		return
	}
	executionID := uint(executionID64)

	// 删除执行记录
	if err := s.db.DeleteStrategyExecution(uid, executionID); err != nil {
		s.DatabaseError(c, "删除策略执行记录", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// calculateStrategyTotalPnL 计算策略执行的总盈亏
func (s *Server) calculateStrategyTotalPnL(executionID uint) float64 {
	// 查询所有由该策略执行创建的订单（无论状态如何）
	var orders []pdb.ScheduledOrder
	err := s.db.DB().Where("execution_id = ?", executionID).Find(&orders).Error
	if err != nil {
		log.Printf("[StrategyStats] Failed to query orders for execution %d: %v", executionID, err)
		return 0
	}

	totalPnL := 0.0
	filledCount := 0

	for _, order := range orders {
		if order.Status == "filled" && order.AvgPrice != "" {
			// 对于已成交的订单，尝试计算盈亏
			pnl, err := s.calculateOrderPnL(&order)
			if err != nil {
				log.Printf("[StrategyStats] Failed to calculate PnL for order %d: %v", order.ID, err)
				continue
			}
			totalPnL += pnl
			filledCount++
		}
	}

	log.Printf("[StrategyStats] Calculated total PnL for execution %d: %.8f (based on %d filled orders out of %d total orders)",
		executionID, totalPnL, filledCount, len(orders))

	return totalPnL
}

// calculateOrderPnL 计算单个订单的盈亏
func (s *Server) calculateOrderPnL(order *pdb.ScheduledOrder) (float64, error) {
	// 平仓订单不应该单独计算盈亏，盈亏应该从开仓订单计算
	if order.ReduceOnly {
		return 0, nil
	}

	if order.AvgPrice == "" {
		return 0, fmt.Errorf("no avg price")
	}

	entryPrice, err := strconv.ParseFloat(order.AvgPrice, 64)
	if err != nil || entryPrice <= 0 {
		return 0, fmt.Errorf("invalid entry price: %s", order.AvgPrice)
	}

	// 获取执行数量
	quantity := 0.0
	if order.ExecutedQty != "" {
		quantity, err = strconv.ParseFloat(order.ExecutedQty, 64)
		if err != nil {
			quantity, err = strconv.ParseFloat(order.AdjustedQuantity, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid quantity")
			}
		}
	} else if order.AdjustedQuantity != "" {
		quantity, err = strconv.ParseFloat(order.AdjustedQuantity, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid quantity")
		}
	} else {
		return 0, fmt.Errorf("no quantity information")
	}

	var totalPnL float64

	// 检查是否有平仓订单（已实现盈亏）
	if order.CloseOrderIds != "" {
		// 计算已实现盈亏
		realizedPnL := s.calculateRealizedPnL(order, entryPrice)
		totalPnL += realizedPnL

		// 检查是否已被完全平仓
		totalClosedQty := s.getTotalClosedQuantity(order)
		if totalClosedQty >= quantity {
			// 已被完全平仓，只有已实现盈亏，未实现盈亏为0
			// 考虑杠杆
			if order.Leverage > 1 {
				totalPnL *= float64(order.Leverage)
			}
			return totalPnL, nil
		}

		// 部分持仓：计算剩余持仓的未实现盈亏
		remainingQty := quantity - totalClosedQty
		ctx := context.Background()
		currentPrice, err := s.getCurrentPrice(ctx, order.Symbol, "futures")
		if err != nil {
			// 如果获取当前价格失败，只返回已实现盈亏
			log.Printf("[calculateOrderPnL] Failed to get current price for %s: %v, returning realized PnL only", order.Symbol, err)
			if order.Leverage > 1 {
				totalPnL *= float64(order.Leverage)
			}
			return totalPnL, nil
		}

		var unrealizedPnL float64
		if order.Side == "BUY" {
			// 多头持仓
			unrealizedPnL = remainingQty * (currentPrice - entryPrice)
		} else {
			// 空头持仓
			unrealizedPnL = remainingQty * (entryPrice - currentPrice)
		}
		totalPnL += unrealizedPnL
	} else {
		// 没有平仓订单：计算全部持仓的未实现盈亏
		ctx := context.Background()
		currentPrice, err := s.getCurrentPrice(ctx, order.Symbol, "futures")
		if err != nil {
			return 0, fmt.Errorf("failed to get current price: %v", err)
		}

		var unrealizedPnL float64
		if order.Side == "BUY" {
			// 多头：(当前价格 - 开仓价格) * 数量
			unrealizedPnL = (currentPrice - entryPrice) * quantity
		} else {
			// 空头：(开仓价格 - 当前价格) * 数量
			unrealizedPnL = (entryPrice - currentPrice) * quantity
		}
		totalPnL = unrealizedPnL
	}

	// 考虑杠杆（如果有的话）
	if order.Leverage > 1 {
		totalPnL *= float64(order.Leverage)
	}

	return totalPnL, nil
}
