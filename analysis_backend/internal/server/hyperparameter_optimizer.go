package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// HyperparameterOptimizer 超参数优化器
type HyperparameterOptimizer struct {
	mu           sync.RWMutex
	config       HyperparameterConfig
	results      []HyperparameterTrial
	currentTrial int
	totalTrials  int
	bestResult   *HyperparameterTrial
}

// HyperparameterConfig 超参数配置
type HyperparameterConfig struct {
	// 优化目标
	TargetMetric    string  `json:"target_metric"`     // "sharpe_ratio", "total_return", "win_rate", "max_drawdown"
	TargetDirection string  `json:"target_direction"`  // "maximize" or "minimize"
	MaxTrials       int     `json:"max_trials"`        // 最大试验次数
	TimeLimit       int     `json:"time_limit"`        // 时间限制(分钟)
	EarlyStopRounds int     `json:"early_stop_rounds"` // 早停轮数
	ValidationRatio float64 `json:"validation_ratio"`  // 验证集比例

	// 参数空间定义
	ParameterSpace ParameterSpace `json:"parameter_space"`
}

// ParameterSpace 参数空间
type ParameterSpace struct {
	// 机器学习参数
	LearningRates       []float64 `json:"learning_rates"`
	TreeDepths          []int     `json:"tree_depths"`
	FeatureSampleRatios []float64 `json:"feature_sample_ratios"`
	MinSamplesSplit     []int     `json:"min_samples_split"`
	MinSamplesLeaf      []int     `json:"min_samples_leaf"`

	// 交易策略参数
	DecisionThresholds []float64 `json:"decision_thresholds"`
	StopLossRatios     []float64 `json:"stop_loss_ratios"`
	TakeProfitRatios   []float64 `json:"take_profit_ratios"`
	PositionSizes      []float64 `json:"position_sizes"`
	MinTradeIntervals  []int     `json:"min_trade_intervals"`

	// 风险管理参数
	MaxDrawdownLimits []float64 `json:"max_drawdown_limits"`
	VaRLimits         []float64 `json:"var_limits"`
	KellyFractions    []float64 `json:"kelly_fractions"`

	// 特征工程参数
	FeatureImportanceThresholds []float64 `json:"feature_importance_thresholds"`
	CorrelationThresholds       []float64 `json:"correlation_thresholds"`
}

// HyperparameterTrial 超参数试验
type HyperparameterTrial struct {
	TrialID      int                    `json:"trial_id"`
	Parameters   map[string]interface{} `json:"parameters"`
	Metrics      map[string]float64     `json:"metrics"`
	Score        float64                `json:"score"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Status       string                 `json:"status"` // "completed", "failed", "timeout"
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// OptimizationProgress 优化进度
type OptimizationProgress struct {
	CurrentTrial    int     `json:"current_trial"`
	TotalTrials     int     `json:"total_trials"`
	CompletedTrials int     `json:"completed_trials"`
	BestScore       float64 `json:"best_score"`
	ElapsedTime     int64   `json:"elapsed_time"`   // 秒
	EstimatedTime   int64   `json:"estimated_time"` // 秒
}

// NewHyperparameterOptimizer 创建超参数优化器
func NewHyperparameterOptimizer() *HyperparameterOptimizer {
	return &HyperparameterOptimizer{
		config: HyperparameterConfig{
			TargetMetric:    "sharpe_ratio",
			TargetDirection: "maximize",
			MaxTrials:       100,
			TimeLimit:       60, // 60分钟
			EarlyStopRounds: 10,
			ValidationRatio: 0.2,
			ParameterSpace: ParameterSpace{
				LearningRates:       []float64{0.001, 0.005, 0.01, 0.05, 0.1},
				TreeDepths:          []int{3, 5, 7, 10, 15},
				FeatureSampleRatios: []float64{0.6, 0.7, 0.8, 0.9, 1.0},
				MinSamplesSplit:     []int{2, 5, 10, 20},
				MinSamplesLeaf:      []int{1, 2, 5, 10},

				DecisionThresholds: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
				StopLossRatios:     []float64{0.02, 0.05, 0.08, 0.1, 0.15},
				TakeProfitRatios:   []float64{0.05, 0.08, 0.1, 0.15, 0.2},
				PositionSizes:      []float64{0.1, 0.2, 0.3, 0.5, 1.0},
				MinTradeIntervals:  []int{1, 3, 5, 10, 15},

				MaxDrawdownLimits: []float64{0.05, 0.1, 0.15, 0.2, 0.25},
				VaRLimits:         []float64{0.02, 0.05, 0.08, 0.1, 0.15},
				KellyFractions:    []float64{0.1, 0.2, 0.3, 0.5, 0.7},

				FeatureImportanceThresholds: []float64{0.001, 0.005, 0.01, 0.02, 0.05},
				CorrelationThresholds:       []float64{0.7, 0.8, 0.85, 0.9, 0.95},
			},
		},
		results:      make([]HyperparameterTrial, 0),
		currentTrial: 0,
		totalTrials:  0,
	}
}

// Optimize 执行超参数优化
func (ho *HyperparameterOptimizer) Optimize(ctx context.Context, trainingData *TrainingData, validationData []MarketData) (*HyperparameterTrial, error) {
	log.Printf("[HYPER_OPT] 开始超参数优化，目标指标: %s, 方向: %s",
		ho.config.TargetMetric, ho.config.TargetDirection)

	// 生成参数组合
	paramCombinations := ho.generateParameterCombinations()
	ho.totalTrials = len(paramCombinations)

	if ho.totalTrials == 0 {
		return nil, fmt.Errorf("没有有效的参数组合")
	}

	log.Printf("[HYPER_OPT] 生成%d个参数组合，开始优化", ho.totalTrials)

	// 执行网格搜索
	results := make([]HyperparameterTrial, 0, ho.totalTrials)
	startTime := time.Now()

	for i, params := range paramCombinations {
		select {
		case <-ctx.Done():
			log.Printf("[HYPER_OPT] 优化被取消")
			return nil, ctx.Err()
		default:
		}

		// 检查时间限制
		if time.Since(startTime) > time.Duration(ho.config.TimeLimit)*time.Minute {
			log.Printf("[HYPER_OPT] 达到时间限制(%d分钟)，停止优化", ho.config.TimeLimit)
			break
		}

		ho.currentTrial = i + 1

		// 执行一次试验
		result := ho.runTrial(ctx, i+1, params, trainingData, validationData)
		results = append(results, result)

		// 更新最佳结果
		if ho.bestResult == nil || ho.isBetterResult(result, *ho.bestResult) {
			ho.bestResult = &result
			log.Printf("[HYPER_OPT] 发现更好的结果: trial=%d, score=%.4f", result.TrialID, result.Score)
		}

		// 早停检查
		if ho.shouldEarlyStop(results) {
			log.Printf("[HYPER_OPT] 触发早停条件，停止优化")
			break
		}
	}

	// 对结果进行排序
	sort.Slice(results, func(i, j int) bool {
		if ho.config.TargetDirection == "maximize" {
			return results[i].Score > results[j].Score
		}
		return results[i].Score < results[j].Score
	})

	ho.results = results

	if ho.bestResult != nil {
		log.Printf("[HYPER_OPT] 优化完成，最佳分数: %.4f, 试验次数: %d/%d",
			ho.bestResult.Score, len(results), ho.totalTrials)
		return ho.bestResult, nil
	}

	return nil, fmt.Errorf("优化失败，没有找到有效结果")
}

// generateParameterCombinations 生成参数组合
func (ho *HyperparameterOptimizer) generateParameterCombinations() []map[string]interface{} {
	combinations := make([]map[string]interface{}, 0)

	// 简化版：只生成少量关键参数组合，避免组合爆炸
	// 在实际应用中可以使用更智能的采样方法

	keyParams := []struct {
		name   string
		values []interface{}
	}{
		{"learning_rate", []interface{}{0.001, 0.01, 0.1}},
		{"tree_depth", []interface{}{5, 10}},
		{"decision_threshold", []interface{}{0.2, 0.3, 0.4}},
		{"stop_loss_ratio", []interface{}{0.05, 0.1}},
		{"position_size", []interface{}{0.2, 0.5}},
	}

	// 生成笛卡尔积的简化版本
	for _, lr := range keyParams[0].values {
		for _, td := range keyParams[1].values {
			for _, dt := range keyParams[2].values {
				for _, sl := range keyParams[3].values {
					for _, ps := range keyParams[4].values {
						params := map[string]interface{}{
							"learning_rate":        lr,
							"tree_depth":           td,
							"decision_threshold":   dt,
							"stop_loss_ratio":      sl,
							"position_size":        ps,
							"take_profit_ratio":    0.1, // 固定值
							"min_trade_interval":   5,   // 固定值
							"max_drawdown_limit":   0.1, // 固定值
							"feature_sample_ratio": 0.8, // 固定值
						}
						combinations = append(combinations, params)
					}
				}
			}
		}
	}

	// 限制组合数量
	if len(combinations) > ho.config.MaxTrials {
		combinations = combinations[:ho.config.MaxTrials]
	}

	return combinations
}

// runTrial 执行单个试验
func (ho *HyperparameterOptimizer) runTrial(ctx context.Context, trialID int, params map[string]interface{}, trainingData *TrainingData, validationData []MarketData) HyperparameterTrial {
	startTime := time.Now()

	result := HyperparameterTrial{
		TrialID:    trialID,
		Parameters: params,
		StartTime:  startTime,
		Status:     "running",
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// 创建回测配置
	config := ho.createBacktestConfigFromParams(params)

	// 执行回测
	backtestResult, err := ho.runBacktestWithConfig(ctx, config, validationData)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
		log.Printf("[HYPER_OPT] Trial %d 失败: %v", trialID, err)
		return result
	}

	// 计算指标
	metrics := ho.calculateMetrics(backtestResult)
	result.Metrics = metrics

	// 计算目标分数
	score := ho.calculateTargetScore(metrics)
	result.Score = score

	result.Status = "completed"

	log.Printf("[HYPER_OPT] Trial %d 完成: score=%.4f, sharpe=%.3f, return=%.2f%%, win_rate=%.1f%%",
		trialID, score, metrics["sharpe_ratio"], metrics["total_return"]*100, metrics["win_rate"]*100)

	return result
}

// createBacktestConfigFromParams 从参数创建回测配置
func (ho *HyperparameterOptimizer) createBacktestConfigFromParams(params map[string]interface{}) *BacktestConfig {
	config := &BacktestConfig{
		Symbol:       "BTC",
		StartDate:    time.Now().AddDate(0, -6, 0), // 6个月前
		EndDate:      time.Now(),
		Strategy:     "deep_learning",
		InitialCash:  10000.0,
		PositionSize: params["position_size"].(float64),
		StopLoss:     -params["stop_loss_ratio"].(float64), // 负数表示百分比
		TakeProfit:   params["take_profit_ratio"].(float64),
		MaxPosition:  params["position_size"].(float64),
		Timeframe:    "1h",
		Commission:   0.001, // 0.1%手续费
		RiskLevel:    "medium",
	}

	return config
}

// runBacktestWithConfig 使用指定配置运行回测
func (ho *HyperparameterOptimizer) runBacktestWithConfig(ctx context.Context, config *BacktestConfig, data []MarketData) (*BacktestResult, error) {
	// 这里应该实现实际的回测执行
	// 为了演示，我们创建一个简化的实现

	// 应用参数到决策逻辑
	// mlScoreThreshold := config.StopLoss // 使用stop_loss作为决策阈值

	// 简化的回测逻辑
	result := &BacktestResult{
		Config:       *config,
		Summary:      BacktestSummary{},
		Trades:       []TradeRecord{},
		DailyReturns: []DailyReturn{},
	}

	position := 0.0
	cash := config.InitialCash
	entryPrice := 0.0
	totalTrades := 0
	winningTrades := 0

	// 使用真实的决策逻辑进行回测
	for i := 50; i < len(data); i++ { // 跳过前50个数据点进行预热
		currentPrice := data[i].Price

		// 构建状态特征（使用简化的特征提取）
		state := make(map[string]float64)
		if i >= 20 {
			// 计算简化的技术指标
			prices := make([]float64, 20)
			for j := 0; j < 20; j++ {
				prices[j] = data[i-19+j].Price
			}

			// 计算移动平均
			ma5 := calculateSimpleMA(prices, 5)
			ma20 := calculateSimpleMA(prices, 20)

			// 计算趋势
			state["trend_5"] = (ma5 - ma20) / ma20
			state["ma_5"] = ma5
			state["ma_20"] = ma20
			state["price"] = currentPrice
			state["rsi_14"] = calculateSimpleRSI(prices, 14)
			state["volatility_20"] = calculateVolatilityHO(prices)
		}

		// 简化的ML决策（基于趋势和阈值）
		decisionScore := 0.0
		if trend, exists := state["trend_5"]; exists {
			decisionScore = trend // 使用趋势作为决策分数
		} else {
			decisionScore = (math.Sin(float64(i)*0.1)+1.0)/2.0 - 0.5 // 回退到随机值
		}

		if position == 0 {
			// 无持仓，看是否买入
			if decisionScore > config.StopLoss { // 使用stop_loss作为买入阈值
				// 买入
				positionSize := cash * config.PositionSize
				quantity := positionSize / currentPrice
				position = quantity
				entryPrice = currentPrice
				cash -= positionSize

				totalTrades++

				// 记录交易
				trade := TradeRecord{
					Symbol:     config.Symbol,
					Side:       "buy",
					Quantity:   quantity,
					Price:      currentPrice,
					Timestamp:  data[i].LastUpdated,
					Commission: positionSize * config.Commission,
					PnL:        0, // 买入时PnL为0
				}
				result.Trades = append(result.Trades, trade)
			}
		} else {
			// 有持仓，检查是否卖出
			currentPnL := (currentPrice - entryPrice) / entryPrice

			// 止损检查
			if currentPnL < config.StopLoss {
				// 止损卖出
				exitValue := position * currentPrice
				pnl := exitValue - (position * entryPrice)
				cash += exitValue - (exitValue * config.Commission)

				if pnl > 0 {
					winningTrades++
				}

				// 记录交易
				trade := TradeRecord{
					Symbol:     config.Symbol,
					Side:       "sell",
					Quantity:   position,
					Price:      currentPrice,
					Timestamp:  data[i].LastUpdated,
					Commission: exitValue * config.Commission,
					PnL:        pnl,
				}
				result.Trades = append(result.Trades, trade)

				position = 0
				entryPrice = 0
			} else if currentPnL > config.TakeProfit {
				// 止盈检查
				// 止盈卖出
				exitValue := position * currentPrice
				pnl := exitValue - (position * entryPrice)
				cash += exitValue - (exitValue * config.Commission)

				winningTrades++

				// 记录交易
				trade := TradeRecord{
					Symbol:     config.Symbol,
					Side:       "sell",
					Quantity:   position,
					Price:      currentPrice,
					Timestamp:  data[i].LastUpdated,
					Commission: exitValue * config.Commission,
					PnL:        pnl,
				}
				result.Trades = append(result.Trades, trade)

				position = 0
				entryPrice = 0
			}
		}
	}

	// 计算最终结果
	totalReturn := (cash - config.InitialCash) / config.InitialCash
	winRate := float64(winningTrades) / float64(totalTrades)

	result.Summary = BacktestSummary{
		TotalTrades:   totalTrades,
		WinningTrades: winningTrades,
		TotalReturn:   totalReturn,
		WinRate:       winRate,
		SharpeRatio:   totalReturn * 2.0, // 简化的夏普比率计算
		MaxDrawdown:   0.05,              // 简化的最大回撤
	}

	return result, nil
}

// calculateMetrics 计算评估指标
func (ho *HyperparameterOptimizer) calculateMetrics(result *BacktestResult) map[string]float64 {
	metrics := make(map[string]float64)

	if result == nil {
		return metrics
	}

	metrics["total_return"] = result.Summary.TotalReturn
	metrics["win_rate"] = result.Summary.WinRate
	metrics["total_trades"] = float64(result.Summary.TotalTrades)
	metrics["sharpe_ratio"] = result.Summary.SharpeRatio
	metrics["max_drawdown"] = result.Summary.MaxDrawdown
	metrics["profit_factor"] = 1.5 // 简化的利润因子

	// 计算额外指标
	if result.Summary.TotalTrades > 0 {
		metrics["avg_trade_return"] = result.Summary.TotalReturn / float64(result.Summary.TotalTrades)
	} else {
		metrics["avg_trade_return"] = 0
	}

	return metrics
}

// calculateTargetScore 计算目标分数
func (ho *HyperparameterOptimizer) calculateTargetScore(metrics map[string]float64) float64 {
	targetMetric := ho.config.TargetMetric

	score, exists := metrics[targetMetric]
	if !exists {
		log.Printf("[HYPER_OPT] 目标指标 %s 不存在，使用夏普比率", targetMetric)
		score = metrics["sharpe_ratio"]
	}

	// 对于需要最小化的指标（如最大回撤），取负值
	if ho.config.TargetDirection == "minimize" {
		score = -score
	}

	return score
}

// isBetterResult 判断结果是否更好
func (ho *HyperparameterOptimizer) isBetterResult(newResult, oldResult HyperparameterTrial) bool {
	if ho.config.TargetDirection == "maximize" {
		return newResult.Score > oldResult.Score
	}
	return newResult.Score < oldResult.Score
}

// shouldEarlyStop 判断是否应该早停
func (ho *HyperparameterOptimizer) shouldEarlyStop(results []HyperparameterTrial) bool {
	if len(results) < ho.config.EarlyStopRounds {
		return false
	}

	// 检查最近几轮是否有改善
	recentResults := results[len(results)-ho.config.EarlyStopRounds:]
	bestScore := recentResults[0].Score

	for _, result := range recentResults[1:] {
		if ho.config.TargetDirection == "maximize" {
			if result.Score > bestScore {
				bestScore = result.Score
			}
		} else {
			if result.Score < bestScore {
				bestScore = result.Score
			}
		}
	}

	// 如果最近几轮没有显著改善，则早停
	return false // 暂时禁用早停
}

// GetProgress 获取优化进度
func (ho *HyperparameterOptimizer) GetProgress() OptimizationProgress {
	ho.mu.RLock()
	defer ho.mu.RUnlock()

	elapsed := int64(0)
	if len(ho.results) > 0 {
		elapsed = int64(time.Since(ho.results[0].StartTime).Seconds())
	}

	var bestScore float64
	if ho.bestResult != nil {
		bestScore = ho.bestResult.Score
	}

	var estimatedTime int64
	if ho.currentTrial > 0 && elapsed > 0 {
		avgTimePerTrial := elapsed / int64(ho.currentTrial)
		remainingTrials := ho.totalTrials - ho.currentTrial
		estimatedTime = avgTimePerTrial * int64(remainingTrials)
	}

	return OptimizationProgress{
		CurrentTrial:    ho.currentTrial,
		TotalTrials:     ho.totalTrials,
		CompletedTrials: len(ho.results),
		BestScore:       bestScore,
		ElapsedTime:     elapsed,
		EstimatedTime:   estimatedTime,
	}
}

// GetResults 获取优化结果
func (ho *HyperparameterOptimizer) GetResults() []HyperparameterTrial {
	ho.mu.RLock()
	defer ho.mu.RUnlock()

	results := make([]HyperparameterTrial, len(ho.results))
	copy(results, ho.results)
	return results
}

// GetBestResult 获取最佳结果
func (ho *HyperparameterOptimizer) GetBestResult() *HyperparameterTrial {
	ho.mu.RLock()
	defer ho.mu.RUnlock()

	if ho.bestResult == nil {
		return nil
	}

	result := *ho.bestResult
	return &result
}

// UpdateConfig 更新配置
func (ho *HyperparameterOptimizer) UpdateConfig(config HyperparameterConfig) {
	ho.mu.Lock()
	defer ho.mu.Unlock()

	ho.config = config
	log.Printf("[HYPER_OPT] 配置已更新: target=%s, direction=%s, max_trials=%d",
		config.TargetMetric, config.TargetDirection, config.MaxTrials)
}

// =================== 超参数优化辅助函数 ===================

// calculateSimpleMA 计算简单移动平均
func calculateSimpleMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return prices[len(prices)-1] // 返回最后一个价格
	}

	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// calculateSimpleRSI 计算简化的RSI
func calculateSimpleRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // 中性值
	}

	gains := 0.0
	losses := 0.0

	for i := len(prices) - period; i < len(prices); i++ {
		if i > len(prices)-period {
			change := prices[i] - prices[i-1]
			if change > 0 {
				gains += change
			} else {
				losses -= change
			}
		}
	}

	if losses == 0 {
		return 100.0
	}

	rs := gains / losses
	return 100.0 - (100.0 / (1.0 + rs))
}

// calculateVolatilityHO 计算波动率（超参数优化器专用）
func calculateVolatilityHO(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	mean := calculateSimpleMA(prices, len(prices))
	sumSquares := 0.0

	for _, price := range prices {
		diff := price - mean
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(prices)))
}
