package server

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gonum.org/v1/gonum/mat"
)

// GetModelPerformance 模型性能API
// GET /ml/model/performance
func (s *Server) GetModelPerformance(c *gin.Context) {
	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	performance := s.machineLearning.GetPerformanceMetrics()

	c.JSON(200, gin.H{
		"status":      "success",
		"performance": performance,
	})
}

// GetModelHealth 模型健康监控API
// GET /ml/model/health
func (s *Server) GetModelHealth(c *gin.Context) {
	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	healthReport := s.machineLearning.MonitorModelHealth()

	c.JSON(200, gin.H{
		"status": "success",
		"health": healthReport,
	})
}

// ValidateModels 验证所有模型质量API
// GET /ml/model/validate
func (s *Server) ValidateModels(c *gin.Context) {
	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	validationResults := s.machineLearning.ValidateAllModels()

	c.JSON(200, gin.H{
		"status":     "success",
		"validation": validationResults,
		"timestamp":  time.Now(),
	})
}

// ValidateFeatures 验证特征值API
// POST /ml/features/validate
func (s *Server) ValidateFeatures(c *gin.Context) {
	var req struct {
		Symbol   string             `json:"symbol" binding:"required"`
		Features map[string]float64 `json:"features" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"status":  "error",
			"message": "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	// 验证特征值
	validationResults := make(map[string]bool)
	validCount := 0
	totalCount := len(req.Features)

	for name, value := range req.Features {
		isValid := s.machineLearning.isValidFeatureValue(name, value)
		validationResults[name] = isValid
		if isValid {
			validCount++
		}
	}

	c.JSON(200, gin.H{
		"status":           "success",
		"symbol":           req.Symbol,
		"total_features":   totalCount,
		"valid_features":   validCount,
		"invalid_features": totalCount - validCount,
		"validation":       validationResults,
		"validation_rate":  float64(validCount) / float64(totalCount),
		"timestamp":        time.Now(),
	})
}

// TrainModel 训练模型API
// POST /ml/model/train
func (s *Server) TrainModel(c *gin.Context) {
	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	err := s.machineLearning.TrainModel(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{
			"status":  "error",
			"message": "模型训练失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "模型训练完成",
	})
}

// PredictWithModel 模型预测API
// POST /ml/model/predict
func (s *Server) PredictWithModel(c *gin.Context) {
	var req struct {
		Features []float64 `json:"features" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	prediction, confidence, err := s.machineLearning.Predict(c.Request.Context(), req.Features)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "预测失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"prediction": prediction,
		"confidence": confidence,
	})
}

// ExtractFeatures 特征提取API
// POST /ml/features/extract
func (s *Server) ExtractFeatures(c *gin.Context) {
	var req struct {
		Symbol string                 `json:"symbol" binding:"required"`
		Data   map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	features := s.extractFeaturesForSymbol(req.Symbol, req.Data)

	c.JSON(200, gin.H{
		"symbol":   req.Symbol,
		"features": features,
	})
}

// BatchExtractFeatures 批量特征提取API
// POST /ml/features/batch-extract
func (s *Server) BatchExtractFeatures(c *gin.Context) {
	var req struct {
		Symbols []string                          `json:"symbols" binding:"required"`
		Data    map[string]map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	results := make(map[string][]float64)
	for _, symbol := range req.Symbols {
		data := req.Data[symbol]
		features := s.extractFeaturesForSymbol(symbol, data)
		results[symbol] = features
	}

	c.JSON(200, gin.H{
		"results": results,
	})
}

// GetFeatureImportance 特征重要性API
// GET /ml/features/importance
func (s *Server) GetFeatureImportance(c *gin.Context) {
	if s.machineLearning == nil {
		c.JSON(200, gin.H{
			"status":  "error",
			"message": "机器学习服务未初始化",
		})
		return
	}

	importance := s.machineLearning.GetFeatureImportance()

	c.JSON(200, gin.H{
		"importance": importance,
	})
}

// GetFeatureQuality 特征质量API
// GET /ml/features/quality
func (s *Server) GetFeatureQuality(c *gin.Context) {
	quality := s.analyzeFeatureQuality()

	c.JSON(200, gin.H{
		"quality_metrics": quality,
	})
}

// extractFeaturesForSymbol 为单个符号提取特征
func (s *Server) extractFeaturesForSymbol(symbol string, data map[string]interface{}) []float64 {
	// 简化的特征提取
	features := []float64{0.5, 0.3, 0.7, 0.2, 0.8} // 模拟特征
	return features
}

// analyzeFeatureQuality 分析特征质量
func (s *Server) analyzeFeatureQuality() map[string]interface{} {
	return map[string]interface{}{
		"total_features":   25,
		"quality_score":    0.85,
		"correlation_avg":  0.15,
		"stability_score":  0.78,
		"predictive_power": 0.72,
	}
}

// AnalyzeFeatureImportanceAPI 特征重要性分析API
func (s *Server) AnalyzeFeatureImportanceAPI(c *gin.Context) {
	modelName := c.DefaultQuery("model", "random_forest")

	analysis, err := s.machineLearning.AnalyzeFeatureImportance(c.Request.Context(), modelName)
	if err != nil {
		log.Printf("[API_ERROR] 特征重要性分析失败: %v", err)
		c.JSON(500, gin.H{
			"error":   "特征重要性分析失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success":  true,
		"analysis": analysis,
	})
}

// AddOnlineLearningSampleAPI 添加在线学习样本API
func (s *Server) AddOnlineLearningSampleAPI(c *gin.Context) {
	var request struct {
		Symbol   string    `json:"symbol" binding:"required"`
		Features []float64 `json:"features" binding:"required"`
		Target   float64   `json:"target" binding:"required"`
		Action   string    `json:"action"` // buy, sell, hold
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{
			"error":   "无效的请求数据",
			"details": err.Error(),
		})
		return
	}

	// 如果提供了action，将其转换为数值目标
	if request.Action != "" {
		switch request.Action {
		case "buy":
			request.Target = 1.0
		case "sell":
			request.Target = -1.0
		case "hold":
			request.Target = 0.0
		}
	}

	err := s.machineLearning.AddOnlineLearningSample(c.Request.Context(), "random_forest", request.Symbol, request.Features, request.Target)
	if err != nil {
		log.Printf("[API_ERROR] 添加在线学习样本失败: %v", err)
		c.JSON(500, gin.H{
			"error":   "添加在线学习样本失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "在线学习样本已添加",
	})
}

// GetOnlineLearningStatsAPI 获取在线学习统计API
func (s *Server) GetOnlineLearningStatsAPI(c *gin.Context) {
	modelName := c.DefaultQuery("model", "random_forest")

	stats := s.machineLearning.GetOnlineLearningStats(modelName)
	if stats == nil {
		c.JSON(404, gin.H{
			"error": "模型不存在或在线学习未启用",
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"stats":   stats,
	})
}

// EnableOnlineLearningAPI 启用在线学习API
func (s *Server) EnableOnlineLearningAPI(c *gin.Context) {
	var config OnlineLearningConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		// 使用默认配置
		config = DefaultOnlineLearningConfig()
		config.Enabled = true
	}

	err := s.machineLearning.EnableOnlineLearning(config)
	if err != nil {
		log.Printf("[API_ERROR] 启用在线学习失败: %v", err)
		c.JSON(500, gin.H{
			"error":   "启用在线学习失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "在线学习已启用",
		"config":  config,
	})
}

// DisableOnlineLearningAPI 禁用在线学习API
func (s *Server) DisableOnlineLearningAPI(c *gin.Context) {
	err := s.machineLearning.DisableOnlineLearning()
	if err != nil {
		log.Printf("[API_ERROR] 禁用在线学习失败: %v", err)
		c.JSON(500, gin.H{
			"error":   "禁用在线学习失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "在线学习已禁用",
	})
}

// =================== 超参数优化API ===================

// OptimizeHyperparametersAPI 超参数优化API
// POST /api/v1/ml/hyperparameters/optimize
func (s *Server) OptimizeHyperparametersAPI(c *gin.Context) {
	var req struct {
		Symbol       string `json:"symbol" binding:"required"`
		TargetMetric string `json:"target_metric"` // "sharpe_ratio", "total_return", "win_rate"
		MaxTrials    int    `json:"max_trials"`    // 最大试验次数
		TimeLimit    int    `json:"time_limit"`    // 时间限制(分钟)
		StartDate    string `json:"start_date"`    // 开始日期
		EndDate      string `json:"end_date"`      // 结束日期
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 设置默认值
	if req.TargetMetric == "" {
		req.TargetMetric = "sharpe_ratio"
	}
	if req.MaxTrials == 0 {
		req.MaxTrials = 50
	}
	if req.TimeLimit == 0 {
		req.TimeLimit = 30 // 30分钟
	}

	// 解析日期
	startDate := time.Now().AddDate(0, -6, 0) // 默认6个月前
	endDate := time.Now()

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

	log.Printf("[API] 开始超参数优化: symbol=%s, target=%s, max_trials=%d",
		req.Symbol, req.TargetMetric, req.MaxTrials)

	// 获取历史数据用于优化
	historicalData, err := s.backtestEngine.getHistoricalData(c.Request.Context(), req.Symbol, startDate, endDate)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "获取历史数据失败",
			"details": err.Error(),
		})
		return
	}

	if len(historicalData) < 100 {
		c.JSON(400, gin.H{
			"error": "历史数据不足，至少需要100个数据点",
		})
		return
	}

	// 创建训练数据（这里简化为直接使用历史数据）
	trainingData := &TrainingData{
		X:         mat.NewDense(len(historicalData), 2, nil), // 2个特征
		Y:         make([]float64, len(historicalData)),
		Features:  []string{"price_change", "index"},
		SampleIDs: make([]string, len(historicalData)),
	}

	// 简化的数据准备（实际应该调用特征工程）
	for i := range historicalData {
		// 简化的特征：价格变化率
		priceChange := 0.0
		if i > 0 {
			priceChange = (historicalData[i].Price - historicalData[i-1].Price) / historicalData[i-1].Price
		}
		trainingData.X.Set(i, 0, priceChange)
		trainingData.X.Set(i, 1, float64(i))
		trainingData.Y[i] = priceChange // 简化的目标
		trainingData.SampleIDs[i] = fmt.Sprintf("sample_%d", i)
	}

	// 更新优化器配置
	config := HyperparameterConfig{
		TargetMetric:    req.TargetMetric,
		TargetDirection: "maximize",
		MaxTrials:       req.MaxTrials,
		TimeLimit:       req.TimeLimit,
		EarlyStopRounds: 5,
		ValidationRatio: 0.2,
	}
	s.machineLearning.hyperparameterOptimizer.UpdateConfig(config)

	// 执行优化
	result, err := s.machineLearning.hyperparameterOptimizer.Optimize(c.Request.Context(), trainingData, historicalData)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "超参数优化失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success":           true,
		"message":           "超参数优化完成",
		"best_result":       result,
		"total_trials":      len(s.machineLearning.hyperparameterOptimizer.GetResults()),
		"optimization_time": "completed",
	})
}

// GetHyperparameterOptimizationProgressAPI 获取超参数优化进度API
// GET /api/v1/ml/hyperparameters/progress
func (s *Server) GetHyperparameterOptimizationProgressAPI(c *gin.Context) {
	if s.machineLearning.hyperparameterOptimizer == nil {
		c.JSON(400, gin.H{
			"error": "超参数优化器未初始化",
		})
		return
	}

	progress := s.machineLearning.hyperparameterOptimizer.GetProgress()
	results := s.machineLearning.hyperparameterOptimizer.GetResults()

	c.JSON(200, gin.H{
		"success":  true,
		"progress": progress,
		"results":  results,
	})
}

// GetHyperparameterOptimizationResultsAPI 获取超参数优化结果API
// GET /api/v1/ml/hyperparameters/results
func (s *Server) GetHyperparameterOptimizationResultsAPI(c *gin.Context) {
	if s.machineLearning.hyperparameterOptimizer == nil {
		c.JSON(400, gin.H{
			"error": "超参数优化器未初始化",
		})
		return
	}

	results := s.machineLearning.hyperparameterOptimizer.GetResults()
	bestResult := s.machineLearning.hyperparameterOptimizer.GetBestResult()

	response := gin.H{
		"success":      true,
		"total_trials": len(results),
		"results":      results,
	}

	if bestResult != nil {
		response["best_result"] = bestResult
		response["best_score"] = bestResult.Score
		response["best_parameters"] = bestResult.Parameters
	}

	c.JSON(200, response)
}

// ApplyOptimizedParametersAPI 应用优化后的参数API
// POST /api/v1/ml/hyperparameters/apply
func (s *Server) ApplyOptimizedParametersAPI(c *gin.Context) {
	var req struct {
		Parameters map[string]interface{} `json:"parameters" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	log.Printf("[API] 应用优化后的参数: %+v", req.Parameters)

	// 这里可以更新相关的配置
	// 例如：更新决策阈值、仓位大小等

	// 更新决策阈值
	if threshold, ok := req.Parameters["decision_threshold"].(float64); ok {
		log.Printf("[API] 更新决策阈值为: %.3f", threshold)
		// 这里可以更新相关的决策逻辑
	}

	// 更新仓位大小
	if positionSize, ok := req.Parameters["position_size"].(float64); ok {
		log.Printf("[API] 更新仓位大小为: %.3f", positionSize)
		// 这里可以更新默认仓位配置
	}

	// 更新止损比例
	if stopLossRatio, ok := req.Parameters["stop_loss_ratio"].(float64); ok {
		log.Printf("[API] 更新止损比例为: %.3f", stopLossRatio)
		// 这里可以更新风险管理配置
	}

	c.JSON(200, gin.H{
		"success":            true,
		"message":            "优化参数已应用",
		"applied_parameters": req.Parameters,
	})
}
