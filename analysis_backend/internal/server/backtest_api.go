package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// RunBacktestAPI 运行回测API
// POST /api/backtest/run
func (s *Server) RunBacktestAPI(c *gin.Context) {
	var config BacktestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证配置
	if err := s.validateBacktestConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用analysis模块的回测引擎
	result, err := s.backtestEngine.RunBacktest(c.Request.Context(), config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CompareStrategiesAPI 策略对比API
// POST /api/backtest/compare
func (s *Server) CompareStrategiesAPI(c *gin.Context) {
	var request struct {
		Configs []BacktestConfig `json:"configs" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Configs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一个策略配置"})
		return
	}

	if len(request.Configs) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多只能对比10个策略"})
		return
	}

	// 验证所有配置
	for _, config := range request.Configs {
		if err := s.validateBacktestConfig(config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// 使用analysis模块的回测引擎
	comparison, err := s.backtestEngine.CompareStrategies(c.Request.Context(), request.Configs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    comparison,
	})
}

// BatchBacktestAPI 批量回测API
// POST /api/backtest/batch
func (s *Server) BatchBacktestAPI(c *gin.Context) {
	var request struct {
		Configs []BacktestConfig `json:"configs" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Configs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一个策略配置"})
		return
	}

	if len(request.Configs) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多只能批量运行20个回测"})
		return
	}

	// 验证所有配置
	for _, config := range request.Configs {
		if err := s.validateBacktestConfig(config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// 使用analysis模块的回测引擎
	results, err := s.backtestEngine.RunBatchBacktest(c.Request.Context(), request.Configs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

// OptimizeStrategyAPI 策略优化API
// POST /api/backtest/optimize
func (s *Server) OptimizeStrategyAPI(c *gin.Context) {
	var request struct {
		BaseConfig  BacktestConfig       `json:"base_config" binding:"required"`
		ParamRanges map[string][]float64 `json:"param_ranges" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证基础配置
	if err := s.validateBacktestConfig(request.BaseConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证参数范围
	if len(request.ParamRanges) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必须提供参数范围"})
		return
	}

	// 转换参数范围为StrategyOptimization格式
	optimization := StrategyOptimization{
		Objective:     "sharpe", // 默认优化目标
		Method:        "grid",
		MaxIterations: 100,
		Parameters:    convertParamRangesToParameters(request.ParamRanges),
	}

	// 使用analysis模块的回测引擎
	result, err := s.backtestEngine.OptimizeStrategy(c.Request.Context(), request.BaseConfig, optimization)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetBacktestTemplatesAPI 获取回测模板API
// GET /api/backtest/templates
func (s *Server) GetBacktestTemplatesAPI(c *gin.Context) {
	templates := []BacktestConfig{
		{
			Symbol:      "BTC",
			StartDate:   time.Now().AddDate(0, -6, 0),
			EndDate:     time.Now(),
			InitialCash: 10000,
			Strategy:    "buy_and_hold",
			Timeframe:   "1d",
			MaxPosition: 1.0,
			StopLoss:    0.1,
			TakeProfit:  0.2,
			Commission:  0.001,
		},
		{
			Symbol:      "BTC",
			StartDate:   time.Now().AddDate(0, -3, 0),
			EndDate:     time.Now(),
			InitialCash: 10000,
			Strategy:    "ml_prediction",
			Timeframe:   "1d",
			MaxPosition: 0.5,
			StopLoss:    0.05,
			TakeProfit:  0.1,
			Commission:  0.001,
		},
		{
			Symbol:      "BTC",
			StartDate:   time.Now().AddDate(0, -3, 0),
			EndDate:     time.Now(),
			InitialCash: 10000,
			Strategy:    "ensemble",
			Timeframe:   "1d",
			MaxPosition: 0.3,
			StopLoss:    0.03,
			TakeProfit:  0.08,
			Commission:  0.001,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    templates,
	})
}

// GetAvailableStrategiesAPI 获取可用策略API
// GET /api/backtest/strategies
func (s *Server) GetAvailableStrategiesAPI(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	// 获取用户创建的交易策略
	var tradingStrategies []pdb.TradingStrategy
	if err := s.db.DB().Where("user_id = ?", uid).Find(&tradingStrategies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取策略失败"})
		return
	}

	strategies := []map[string]interface{}{
		{
			"name":         "buy_and_hold",
			"display_name": "买入持有",
			"description":  "简单的买入持有策略，作为基准对比",
			"parameters":   []string{"max_position", "commission"},
			"type":         "predefined",
		},
		{
			"name":         "ml_prediction",
			"display_name": "机器学习预测",
			"description":  "基于机器学习模型的价格预测策略",
			"parameters":   []string{"max_position", "stop_loss", "take_profit", "commission"},
			"type":         "predefined",
		},
		{
			"name":         "ensemble",
			"display_name": "集成学习",
			"description":  "基于集成学习模型的高级预测策略",
			"parameters":   []string{"max_position", "stop_loss", "take_profit", "commission"},
			"type":         "predefined",
		},
	}

	// 添加用户创建的交易策略
	for _, ts := range tradingStrategies {
		strategyDesc := s.generateTradingStrategyDescription(ts)

		// 根据策略复杂度添加回测说明
		backtestNote := ""
		if ts.Conditions.FuturesSpotArbEnabled || ts.Conditions.TriangleArbEnabled ||
			ts.Conditions.CrossExchangeArbEnabled || ts.Conditions.StatArbEnabled {
			backtestNote = " (回测使用简化算法，实际效果请查看执行历史)"
		} else if ts.Conditions.ShortOnGainers || ts.Conditions.LongOnSmallGainers {
			backtestNote = " (回测基于历史表现，策略逻辑在实际执行中应用)"
		}

		strategies = append(strategies, map[string]interface{}{
			"name":         ts.Name, // 直接使用策略名称
			"display_name": ts.Name,
			"description":  strategyDesc + backtestNote,
			"parameters":   []string{"symbol", "start_date", "end_date"},
			"type":         "trading_strategy",
			"strategy_id":  ts.ID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    strategies,
	})
}

// generateTradingStrategyDescription 生成交易策略描述
func (s *Server) generateTradingStrategyDescription(strategy pdb.TradingStrategy) string {
	var conditions []string

	// 基础交易条件
	if strategy.Conditions.SpotContract {
		conditions = append(conditions, "需要现货+合约")
	}

	if strategy.Conditions.SkipHeldPositions {
		conditions = append(conditions, "跳过已在持仓的币种")
	}

	if strategy.Conditions.ShortOnGainers {
		rankLimit := strategy.Conditions.GainersRankLimit
		if rankLimit > 0 {
			conditions = append(conditions, fmt.Sprintf("涨幅前%d做空", rankLimit))
		}
	}

	if strategy.Conditions.LongOnSmallGainers {
		rankLimit := strategy.Conditions.GainersRankLimitLong
		if rankLimit > 0 {
			conditions = append(conditions, fmt.Sprintf("市值<%.0f万涨幅前%d做多",
				strategy.Conditions.MarketCapLimitLong/10000, rankLimit))
		}
	}

	if strategy.Conditions.NoShortBelowMarketCap {
		conditions = append(conditions, fmt.Sprintf("市值<%.0f万不开空",
			strategy.Conditions.MarketCapLimitShort/10000))
	}

	// 套利策略条件
	if strategy.Conditions.FuturesSpotArbEnabled {
		expiryThreshold := strategy.Conditions.ExpiryThreshold
		spreadThreshold := strategy.Conditions.SpotFutureSpread
		if expiryThreshold > 0 {
			conditions = append(conditions, fmt.Sprintf("期现套利(到期<%d天", expiryThreshold))
			if spreadThreshold > 0 {
				conditions[len(conditions)-1] += fmt.Sprintf(",价差>%.1f%%)", spreadThreshold)
			} else {
				conditions[len(conditions)-1] += ")"
			}
		} else {
			conditions = append(conditions, "期现套利")
		}
	}

	if strategy.Conditions.TriangleArbEnabled {
		threshold := strategy.Conditions.TriangleThreshold
		if threshold > 0 {
			conditions = append(conditions, fmt.Sprintf("三角套利(阈值>%.1f%%)", threshold))
		} else {
			conditions = append(conditions, "三角套利")
		}
	}

	if strategy.Conditions.CrossExchangeArbEnabled {
		threshold := strategy.Conditions.PriceDiffThreshold
		if threshold > 0 {
			conditions = append(conditions, fmt.Sprintf("跨交易所套利(价差>%.1f%%)", threshold))
		} else {
			conditions = append(conditions, "跨交易所套利")
		}
	}

	if strategy.Conditions.StatArbEnabled {
		zscore := strategy.Conditions.ZscoreThreshold
		period := strategy.Conditions.CointegrationPeriod
		if zscore > 0 && period > 0 {
			conditions = append(conditions, fmt.Sprintf("统计套利(Z>%.1f,%d天协整)", zscore, period))
		} else {
			conditions = append(conditions, "统计套利")
		}
	}

	// 风险控制条件
	if strategy.Conditions.EnableStopLoss {
		conditions = append(conditions, fmt.Sprintf("止损%.1f%%", strategy.Conditions.StopLossPercent))
	}

	if strategy.Conditions.EnableTakeProfit {
		conditions = append(conditions, fmt.Sprintf("止盈%.1f%%", strategy.Conditions.TakeProfitPercent))
	}

	if len(conditions) == 0 {
		return "自定义交易策略"
	}

	return strings.Join(conditions, "，")
}

// validateBacktestConfig 验证回测配置
func (s *Server) validateBacktestConfig(config BacktestConfig) error {
	if config.Symbol == "" {
		return fmt.Errorf("symbol不能为空")
	}

	if config.StartDate.After(config.EndDate) {
		return fmt.Errorf("开始日期不能晚于结束日期")
	}

	if config.InitialCash <= 0 {
		return fmt.Errorf("初始资金必须大于0")
	}

	if config.Strategy == "" {
		return fmt.Errorf("策略不能为空")
	}

	validStrategies := []string{"buy_and_hold", "ml_prediction", "ensemble"}
	valid := false
	for _, s := range validStrategies {
		if config.Strategy == s {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("不支持的策略: %s", config.Strategy)
	}

	if config.MaxPosition <= 0 || config.MaxPosition > 1 {
		return fmt.Errorf("最大仓位必须在0-1之间")
	}

	if config.StopLoss < 0 || config.StopLoss > 1 {
		return fmt.Errorf("止损比例必须在0-1之间")
	}

	if config.TakeProfit < 0 || config.TakeProfit > 1 {
		return fmt.Errorf("止盈比例必须在0-1之间")
	}

	if config.Commission < 0 || config.Commission > 0.1 {
		return fmt.Errorf("手续费率必须在0-0.1之间")
	}

	return nil
}

// SaveBacktestResultAPI 保存回测结果API
// POST /api/backtest/save
func (s *Server) SaveBacktestResultAPI(c *gin.Context) {
	var request struct {
		Name        string         `json:"name" binding:"required"`
		Description string         `json:"description"`
		Config      BacktestConfig `json:"config" binding:"required"`
		Result      BacktestResult `json:"result" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userIDVal, exists := c.Get("uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}
	userID := userIDVal.(uint)

	// 创建回测记录
	record := &pdb.AsyncBacktestRecord{
		UserID:       userID,
		Symbol:       request.Config.Symbol,
		Strategy:     request.Config.Strategy,
		StartDate:    request.Config.StartDate.Format("2006-01-02"),
		EndDate:      request.Config.EndDate.Format("2006-01-02"),
		Status:       "completed",                 // 手动保存的一定是完成状态
		PositionSize: decimal.NewFromFloat(100.0), // 默认100%
	}

	// 保存回测记录到数据库
	if err := pdb.CreateAsyncBacktestRecord(s.db.DB(), record); err != nil {
		log.Printf("[SaveBacktestResult] 创建回测记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存回测记录失败"})
		return
	}

	// 序列化结果并更新记录
	resultJSONString, serializeErr := s.serializeBacktestResult(&request.Result)
	if serializeErr == nil {
		completedAt := time.Now()
		if updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "completed", &resultJSONString, "", &completedAt); updateErr != nil {
			log.Printf("[SaveBacktestResult] 更新回测记录结果失败: %v", updateErr)
			// 不影响主流程
		}
	} else {
		log.Printf("[SaveBacktestResult] 序列化回测结果失败: %v", serializeErr)
	}

	// 保存交易记录到数据库
	if saveErr := s.saveBacktestTradesToDB(record.ID, &request.Result); saveErr != nil {
		log.Printf("[SaveBacktestResult] 保存交易记录失败: %v", saveErr)
		// 不影响主流程
	}

	log.Printf("[SaveBacktestResult] 回测结果保存成功，记录ID: %d", record.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "回测结果已保存",
		"id":      record.ID,
	})
}

// GetSavedBacktestsAPI 获取保存的回测API
// GET /api/backtest/saved
func (s *Server) GetSavedBacktestsAPI(c *gin.Context) {
	// 获取用户ID
	userIDVal, exists := c.Get("uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}
	userID := userIDVal.(uint)

	// 解析查询参数
	var req struct {
		Page   int    `form:"page,default=1"`
		Limit  int    `form:"limit,default=20"`
		Status string `form:"status"`
		Symbol string `form:"symbol"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的查询参数"})
		return
	}

	// 从数据库查询回测记录
	records, totalCount, err := pdb.GetAsyncBacktestRecords(s.db.DB(), userID, req.Page, req.Limit, req.Status, req.Symbol)
	if err != nil {
		log.Printf("[GetSavedBacktests] 查询回测记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询回测记录失败"})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseRecords,
		"pagination": gin.H{
			"page":  req.Page,
			"limit": req.Limit,
			"total": total,
			"pages": (total + req.Limit - 1) / req.Limit,
		},
	})
}

// convertParamRangesToParameters 将参数范围转换为OptimizationParameter数组
func convertParamRangesToParameters(paramRanges map[string][]float64) []OptimizationParameter {
	var parameters []OptimizationParameter

	for paramName, paramRange := range paramRanges {
		if len(paramRange) >= 2 {
			parameters = append(parameters, OptimizationParameter{
				Name:     paramName,
				Type:     "float",
				MinValue: paramRange[0],
				MaxValue: paramRange[1],
				DataType: "float",
			})
		}
	}

	return parameters
}

// RunStrategyBacktestAPI 基于策略ID运行回测API
// POST /api/backtest/strategy
func (s *Server) RunStrategyBacktestAPI(c *gin.Context) {
	var request struct {
		StrategyID uint      `json:"strategy_id" binding:"required"`
		Symbol     string    `json:"symbol" binding:"required"`
		StartDate  time.Time `json:"start_date"`
		EndDate    time.Time `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户信息
	userIDVal, exists := c.Get("uid")
	if !exists {
		log.Printf("[ERROR] RunStrategyBacktestAPI: 用户未认证")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}
	userID := userIDVal.(uint)
	log.Printf("[DEBUG] RunStrategyBacktestAPI: 用户ID = %d", userID)

	// 获取策略信息
	var strategy pdb.TradingStrategy
	if err := s.db.DB().Where("id = ?", request.StrategyID).First(&strategy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}

	// 设置默认时间范围（如果未提供）
	if request.StartDate.IsZero() {
		request.StartDate = time.Now().AddDate(-1, 0, 0) // 1年前
	}
	if request.EndDate.IsZero() {
		request.EndDate = time.Now() // 现在
	}

	// 将策略配置转换为回测配置
	backtestConfig := s.convertStrategyToBacktestConfig(strategy, request.StrategyID, request.Symbol, request.StartDate, request.EndDate)

	// 检查回测引擎是否已初始化
	if s.backtestEngine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "回测引擎未初始化，请检查服务器配置"})
		return
	}

	// 创建回测记录
	record := &pdb.AsyncBacktestRecord{
		UserID:       userID,
		Symbol:       request.Symbol,
		Strategy:     "strategy", // 使用固定策略类型
		StartDate:    request.StartDate.Format("2006-01-02"),
		EndDate:      request.EndDate.Format("2006-01-02"),
		Status:       "running",
		PositionSize: decimal.NewFromFloat(100.0), // 默认100%
	}

	// 保存回测记录到数据库
	if err := pdb.CreateAsyncBacktestRecord(s.db.DB(), record); err != nil {
		log.Printf("[ERROR] 创建策略回测记录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建回测记录失败"})
		return
	}

	// 执行回测
	result, err := s.backtestEngine.RunBacktest(c.Request.Context(), backtestConfig)
	if err != nil {
		// 更新记录状态为失败
		updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "failed", nil, fmt.Sprintf("回测执行失败: %v", err), nil)
		if updateErr != nil {
			log.Printf("[ERROR] 更新回测记录状态为failed失败 ID=%d: %v", record.ID, updateErr)
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("回测执行失败: %v", err)})
		return
	}

	// 序列化结果
	resultJSONString, serializeErr := s.serializeBacktestResult(result)
	if serializeErr != nil {
		log.Printf("[ERROR] 序列化回测结果失败: %v", serializeErr)
		// 仍然保存记录，但标记为部分成功
	}

	// 保存交易记录到数据库
	if saveErr := s.saveBacktestTradesToDB(record.ID, result); saveErr != nil {
		log.Printf("[ERROR] 保存交易记录失败: %v", saveErr)
		// 继续执行，不影响主流程
	}

	// 更新记录为完成状态
	completedAt := time.Now()
	if updateErr := pdb.UpdateAsyncBacktestRecordStatus(s.db.DB(), record.ID, record.UserID, "completed", &resultJSONString, "", &completedAt); updateErr != nil {
		log.Printf("[ERROR] 更新回测记录状态为completed失败 ID=%d: %v", record.ID, updateErr)
	}

	log.Printf("[INFO] ✅ 策略回测完成并保存记录 ID=%d", record.ID)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      result,
		"record_id": record.ID,
		"strategy": gin.H{
			"id":   strategy.ID,
			"name": strategy.Name,
		},
	})
}

// convertStrategyToBacktestConfig 将策略配置转换为回测配置
func (s *Server) convertStrategyToBacktestConfig(strategy pdb.TradingStrategy, strategyID uint, symbol string, startDate, endDate time.Time) BacktestConfig {
	config := BacktestConfig{
		Symbol:         symbol,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCash:    10000, // 默认初始资金
		Timeframe:      "1d",
		MaxPosition:    0.5,        // 默认最大仓位50%
		Commission:     0.001,      // 默认手续费0.1%
		UserStrategyID: strategyID, // 设置用户策略ID，用于区分策略回测
	}

	// 根据策略条件设置参数
	if strategy.Conditions.StopLossPercent > 0 {
		config.StopLoss = strategy.Conditions.StopLossPercent
	}
	if strategy.Conditions.TakeProfitPercent > 0 {
		config.TakeProfit = strategy.Conditions.TakeProfitPercent
	}

	// 根据策略类型选择合适的回测策略
	// 注意：回测引擎只支持以下策略类型：
	// - "buy_and_hold": 买入持有策略
	// - "ml_prediction": 机器学习预测策略
	// - "ensemble": 集成学习策略
	// - "deep_learning": 深度学习策略
	//
	// 对于复杂的交易策略（如套利、排名筛选等），回测使用简化算法：
	// - 提供历史数据的基准表现
	// - 不能完全重现策略的执行逻辑
	// - 真正的策略验证需要查看实际执行历史

	if s.machineLearning != nil {
		// AI分析模块已启用，可以使用高级策略

		if strategy.Conditions.MeanReversionEnabled {
			// 均值回归策略 - 这是最重要的策略类型，需要特殊处理
			// 注意：当前的回测引擎不支持均值回归策略的完整逻辑
			// 这里使用机器学习预测作为近似，但实际效果会打折
			config.Strategy = "ml_prediction"
			log.Printf("[INFO] 均值回归策略回测：使用机器学习预测作为近似，无法完全重现策略逻辑")
		} else if strategy.Conditions.FuturesSpotArbEnabled || strategy.Conditions.TriangleArbEnabled ||
			strategy.Conditions.CrossExchangeArbEnabled || strategy.Conditions.StatArbEnabled {
			// 套利策略需要智能判断，使用机器学习预测
			config.Strategy = "ml_prediction"
		} else if strategy.Conditions.ShortOnGainers || strategy.Conditions.LongOnSmallGainers {
			// 基于排名的交易策略，适合使用集成学习
			config.Strategy = "ensemble"
		} else if strategy.Conditions.SpotContract {
			// 基础的现货+合约策略，使用机器学习预测
			config.Strategy = "ml_prediction"
		} else {
			// 默认使用机器学习预测
			config.Strategy = "ml_prediction"
		}
	} else {
		// AI分析模块未启用，只能使用基础策略

		if strategy.Conditions.MeanReversionEnabled {
			// 均值回归策略 - 即使没有AI，也要特殊处理
			// 使用买入持有作为基础，但这无法反映真实的策略逻辑
			config.Strategy = "buy_and_hold"
			log.Printf("[WARNING] 均值回归策略回测：AI模块未启用，回测结果仅供参考，无法反映真实策略表现")
		} else if strategy.Conditions.FuturesSpotArbEnabled || strategy.Conditions.TriangleArbEnabled ||
			strategy.Conditions.CrossExchangeArbEnabled || strategy.Conditions.StatArbEnabled {
			// 套利策略在没有AI的情况下，使用买入持有作为基础
			// 实际的套利逻辑需要在策略执行阶段处理
			config.Strategy = "buy_and_hold"
		} else if strategy.Conditions.ShortOnGainers || strategy.Conditions.LongOnSmallGainers ||
			strategy.Conditions.SpotContract {
			// 基础交易策略，使用买入持有
			// 实际的交易逻辑需要在策略执行阶段处理
			config.Strategy = "buy_and_hold"
		} else {
			// 默认使用买入持有策略
			config.Strategy = "buy_and_hold"
		}
	}

	return config
}

// serializeBacktestResult 序列化回测结果
func (s *Server) serializeBacktestResult(result *BacktestResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("result is nil")
	}

	resultJSONBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化失败: %v", err)
	}

	return string(resultJSONBytes), nil
}

// GetFilterCorrectionStats 获取过滤器修正统计信息
func (s *Server) GetFilterCorrectionStats(c *gin.Context) {
	stats, err := pdb.GetFilterCorrectionStats(s.db.DB())
	if err != nil {
		s.DatabaseError(c, "获取过滤器修正统计", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetFilterCorrectionsBySymbol 获取指定交易对的修正历史
func (s *Server) GetFilterCorrectionsBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		s.ValidationError(c, "symbol", "交易对不能为空")
		return
	}

	corrections, err := pdb.GetFilterCorrectionsBySymbol(s.db.DB(), strings.ToUpper(symbol))
	if err != nil {
		s.DatabaseError(c, "获取交易对修正历史", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    corrections,
		"total":   len(corrections),
	})
}

// CleanupOldFilterCorrections 清理旧的过滤器修正记录
func (s *Server) CleanupOldFilterCorrections(c *gin.Context) {
	// 默认保留30天的记录
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 30
	}

	if err := pdb.CleanupOldCorrections(s.db.DB(), days); err != nil {
		s.DatabaseError(c, "清理旧修正记录", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已清理%d天前的修正记录", days),
	})
}
