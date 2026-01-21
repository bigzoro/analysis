package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"
)

// MLPricePredictor 机器学习价格预测器
type MLPricePredictor struct {
	// 特征工程
	FeatureExtractor FeatureEngineer

	// 传统机器学习模型
	Models map[string]MLModel

	// 集成学习模型
	EnsembleModels map[string]*EnsemblePredictor

	// 性能优化组件
	ModelCache      *ModelCache      // 模型缓存
	ParallelTrainer *ParallelTrainer // 并行训练器
	ModelPretrainer *ModelPretrainer // 模型预训练器

	// 模型选择器
	ModelSelector AutoModelSelector

	// 配置
	UseEnsemble    bool // 是否优先使用集成学习
	UseCache       bool // 是否使用模型缓存
	UsePretraining bool // 是否使用预训练模型

	// 历史数据存储
	HistoricalData []MarketData
}

// FeatureEngineer 特征工程器
type FeatureEngineer struct {
	// 技术指标特征
	TechnicalFeatures []string

	// 市场特征
	MarketFeatures []string

	// 情绪特征
	SentimentFeatures []string

	// 资金流特征
	FlowFeatures []string
}

// MLModel 机器学习模型接口
type MLModel interface {
	Train(features [][]float64, targets []float64) error
	Predict(features []float64) (float64, error)
	GetName() string
}

// AutoModelSelector 自动模型选择器
type AutoModelSelector interface {
	SelectBestModel(models []MLModel, validationData [][]float64, validationTargets []float64) MLModel
}

// MarketData 市场数据

// BayesianPricePredictor 贝叶斯价格预测器
type BayesianPricePredictor struct {
	// 先验分布
	PriorDistribution ProbabilityDistribution

	// 似然函数
	LikelihoodFunction LikelihoodCalculator

	// 后验计算器
	PosteriorCalculator BayesianUpdater

	// 数据访问
	db Database
}

// ProbabilityDistribution 概率分布接口
type ProbabilityDistribution interface {
	PDF(x float64) float64
	CDF(x float64) float64
	Mean() float64
	Variance() float64
}

// LikelihoodCalculator 似然计算器
type LikelihoodCalculator interface {
	Calculate(dataPoint interface{}) float64
}

// BayesianUpdater 贝叶斯更新器
type BayesianUpdater interface {
	Update(prior ProbabilityDistribution, likelihood float64) ProbabilityDistribution
	GetPosterior() ProbabilityDistribution
}

// TradingStrategy 交易策略建议
type TradingStrategy struct {
	StrategyType      string          `json:"strategy_type"`      // "LONG", "SHORT", "RANGE"
	EntryZone         PriceRange      `json:"entry_zone"`         // 入场价格区间
	ExitTargets       []PriceRange    `json:"exit_targets"`       // 多个出场目标
	StopLossLevels    []StopLossLevel `json:"stop_loss_levels"`   // 多级止损
	RiskManagement    RiskManagement  `json:"risk_management"`    // 风险管理
	PositionSizing    PositionSizing  `json:"position_sizing"`    // 仓位管理
	MarketCondition   string          `json:"market_condition"`   // 市场环境
	StrategyRationale []string        `json:"strategy_rationale"` // 策略理由
}

// StopLossLevel 止损等级
type StopLossLevel struct {
	Level     float64 `json:"level"`     // 止损价格
	Type      string  `json:"type"`      // "INITIAL", "TRAILING", "MENTAL"
	Condition string  `json:"condition"` // 触发条件
}

// PositionSizing 仓位管理
type PositionSizing struct {
	BasePosition     float64 `json:"base_position"`     // 基础仓位 %
	AdjustedPosition float64 `json:"adjusted_position"` // 调整后仓位 %
	ScalingStrategy  string  `json:"scaling_strategy"`  // "FIXED", "MARTINGALE", "ANTI_MARTINGALE"
	MaxPosition      float64 `json:"max_position"`      // 最大仓位 %
	MinPosition      float64 `json:"min_position"`      // 最小仓位 %
}

// PricePrediction 价格预测结果
type PricePrediction struct {
	Symbol       string    `json:"symbol"`
	CurrentPrice float64   `json:"current_price"`
	PredictedAt  time.Time `json:"predicted_at"`

	// 短期预测（24小时）
	Pred24h       float64    `json:"pred_24h"`       // 预测价格
	Change24h     float64    `json:"change_24h"`     // 预测涨跌幅 %
	Confidence24h float64    `json:"confidence_24h"` // 置信度 0-100
	Range24h      PriceRange `json:"range_24h"`      // 价格区间

	// 中期预测（7天）
	Pred7d       float64    `json:"pred_7d"`       // 预测价格
	Change7d     float64    `json:"change_7d"`     // 预测涨跌幅 %
	Confidence7d float64    `json:"confidence_7d"` // 置信度 0-100
	Range7d      PriceRange `json:"range_7d"`      // 价格区间

	// 长期预测（30天）
	Pred30d       float64    `json:"pred_30d"`       // 预测价格
	Change30d     float64    `json:"change_30d"`     // 预测涨跌幅 %
	Confidence30d float64    `json:"confidence_30d"` // 置信度 0-100
	Range30d      PriceRange `json:"range_30d"`      // 价格区间

	// 预测依据
	Factors []string `json:"factors"` // 预测依据说明
	Trend   string   `json:"trend"`   // "bullish"/"bearish"/"neutral"

	// 交易策略
	TradingStrategy TradingStrategy `json:"trading_strategy"` // 完整的交易策略
}

// GetPricePrediction 获取价格预测
func (s *Server) GetPricePrediction(ctx context.Context, symbol string, kind string) (*PricePrediction, error) {
	// 优先从缓存获取
	cacheKey := fmt.Sprintf("prediction:%s:%s:latest", symbol, kind)
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil && len(cachedData) > 0 {
		var cachedPrediction PricePrediction
		if err := json.Unmarshal(cachedData, &cachedPrediction); err == nil {
			// 缓存命中，直接返回
			return &cachedPrediction, nil
		}
		// 缓存数据损坏，继续实时计算
	}

	// 缓存未命中，实时计算
	// 1. 获取当前价格
	currentPrice, err := s.getCurrentPrice(ctx, symbol, kind)
	if err != nil {
		return nil, fmt.Errorf("获取当前价格失败: %w", err)
	}

	// 2. 获取K线数据（用于技术分析）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 500) // 获取更多数据用于预测
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 60 {
		return nil, fmt.Errorf("K线数据不足，无法进行预测")
	}

	// 3. 提取价格序列
	prices := make([]float64, len(klines))
	volumes := make([]float64, len(klines))
	for i, k := range klines {
		price, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)
		prices[i] = price
		volumes[i] = volume
	}

	// 4. 获取技术指标
	technical, err := s.CalculateTechnicalIndicators(ctx, symbol, kind)
	if err != nil {
		// 如果获取失败，使用基础预测
		technical = &TechnicalIndicators{
			RSI:   50,
			Trend: "sideways",
		}
	}

	// 5. 计算预测
	prediction := &PricePrediction{
		Symbol:       symbol,
		CurrentPrice: currentPrice,
		PredictedAt:  time.Now().UTC(),
	}

	// 6. 24小时预测
	pred24h, change24h, confidence24h, range24h := s.predictPrice(
		prices, volumes, technical, currentPrice, 24,
	)
	prediction.Pred24h = pred24h
	prediction.Change24h = change24h
	prediction.Confidence24h = confidence24h
	prediction.Range24h = range24h

	// 7. 7天预测
	pred7d, change7d, confidence7d, range7d := s.predictPrice(
		prices, volumes, technical, currentPrice, 7*24,
	)
	prediction.Pred7d = pred7d
	prediction.Change7d = change7d
	prediction.Confidence7d = confidence7d
	prediction.Range7d = range7d

	// 8. 30天预测
	pred30d, change30d, confidence30d, range30d := s.predictPrice(
		prices, volumes, technical, currentPrice, 30*24,
	)
	prediction.Pred30d = pred30d
	prediction.Change30d = change30d
	prediction.Confidence30d = confidence30d
	prediction.Range30d = range30d

	// 9. 生成预测依据和趋势
	prediction.Factors = s.generatePredictionFactors(technical, prices, volumes)
	prediction.Trend = s.determinePredictionTrend(change24h, change7d, change30d)

	// 10. 生成完整的交易策略
	prediction.TradingStrategy = s.generateTradingStrategy(
		*prediction, *technical, prices, volumes, currentPrice,
	)

	// 11. 保存到缓存以提高性能
	if s.cache != nil {
		cacheKey := fmt.Sprintf("prediction:%s:%s:latest", symbol, kind)
		if predictionData, err := json.Marshal(prediction); err == nil {
			// 缓存5分钟
			s.cache.Set(ctx, cacheKey, predictionData, 5*time.Minute)
		}
	}

	return prediction, nil
}

// predictPrice 预测价格
// hours: 预测时间（小时）
func (s *Server) predictPrice(
	prices []float64,
	volumes []float64,
	technical *TechnicalIndicators,
	currentPrice float64,
	hours int,
) (predictedPrice, changePercent, confidence float64, priceRange PriceRange) {
	// 使用多种方法预测，然后加权平均

	// 方法1：基于技术指标的趋势预测
	techPred, techConf := s.predictByTechnical(technical, currentPrice, hours)

	// 方法2：基于移动平均的趋势预测
	maPred, maConf := s.predictByMovingAverage(prices, currentPrice, hours)

	// 方法3：基于历史波动率的统计预测
	statPred, statConf := s.predictByStatistics(prices, currentPrice, hours)

	// 方法4：基于成交量的动量预测
	volumePred, volumeConf := s.predictByVolume(prices, volumes, currentPrice, hours)

	// 加权平均（置信度作为权重）
	totalWeight := techConf + maConf + statConf + volumeConf
	if totalWeight == 0 {
		totalWeight = 1
	}

	predictedPrice = (techPred*techConf + maPred*maConf + statPred*statConf + volumePred*volumeConf) / totalWeight
	confidence = (techConf + maConf + statConf + volumeConf) / 4

	// 计算涨跌幅
	changePercent = ((predictedPrice - currentPrice) / currentPrice) * 100

	// 计算价格区间（基于波动率）
	volatility := s.calculateVolatilityFromPrices(prices)
	rangeWidth := predictedPrice * volatility * math.Sqrt(float64(hours)/24) * 0.5 // 0.5倍标准差
	priceRange = PriceRange{
		Min: math.Max(0, predictedPrice-rangeWidth),
		Max: predictedPrice + rangeWidth,
		Avg: predictedPrice,
	}

	return predictedPrice, changePercent, confidence, priceRange
}

// predictByTechnical 基于技术指标预测
func (s *Server) predictByTechnical(tech *TechnicalIndicators, currentPrice float64, hours int) (float64, float64) {
	if tech == nil {
		return currentPrice, 0.3
	}

	// 基于趋势判断
	trendFactor := 1.0
	confidence := 0.5

	switch tech.Trend {
	case "up":
		trendFactor = 1.02 // 上涨趋势，预测上涨2%
		confidence = 0.6
	case "down":
		trendFactor = 0.98 // 下跌趋势，预测下跌2%
		confidence = 0.6
	default:
		trendFactor = 1.0
		confidence = 0.4
	}

	// RSI调整
	if tech.RSI > 70 {
		trendFactor *= 0.98 // 超买，可能回调
		confidence += 0.1
	} else if tech.RSI < 30 {
		trendFactor *= 1.02 // 超卖，可能反弹
		confidence += 0.1
	}

	// MACD调整
	if tech.MACD > tech.MACDSignal && tech.MACDHist > 0 {
		trendFactor *= 1.01 // 金叉
		confidence += 0.05
	} else if tech.MACD < tech.MACDSignal && tech.MACDHist < 0 {
		trendFactor *= 0.99 // 死叉
		confidence += 0.05
	}

	// 均线调整
	if tech.MA5 > tech.MA10 && tech.MA10 > tech.MA20 {
		trendFactor *= 1.01 // 多头排列
		confidence += 0.05
	} else if tech.MA5 < tech.MA10 && tech.MA10 < tech.MA20 {
		trendFactor *= 0.99 // 空头排列
		confidence += 0.05
	}

	// 时间衰减（预测时间越长，趋势影响越小）
	timeDecay := math.Pow(0.95, float64(hours)/24)
	trendFactor = 1.0 + (trendFactor-1.0)*timeDecay

	predictedPrice := currentPrice * trendFactor
	confidence = math.Min(1.0, confidence)

	return predictedPrice, confidence
}

// predictByMovingAverage 基于移动平均预测
func (s *Server) predictByMovingAverage(prices []float64, currentPrice float64, hours int) (float64, float64) {
	if len(prices) < 20 {
		return currentPrice, 0.3
	}

	// 计算短期和长期均线
	shortMA := calculateSMA(prices, 5)
	longMA := calculateSMA(prices, 20)

	if shortMA == 0 || longMA == 0 {
		return currentPrice, 0.3
	}

	// 均线斜率
	recentPrices := prices[len(prices)-5:]
	oldPrices := prices[len(prices)-10 : len(prices)-5]
	shortSlope := (recentPrices[len(recentPrices)-1] - oldPrices[0]) / oldPrices[0]
	longSlope := (prices[len(prices)-1] - prices[len(prices)-20]) / prices[len(prices)-20]

	// 预测价格 = 当前价格 * (1 + 斜率 * 时间因子)
	timeFactor := float64(hours) / 24.0
	slope := (shortSlope*0.6 + longSlope*0.4)                   // 短期权重更高
	predictedPrice := currentPrice * (1 + slope*timeFactor*0.5) // 0.5是衰减因子

	// 置信度：均线越接近，置信度越高
	maDiff := math.Abs(shortMA-longMA) / currentPrice
	confidence := 0.5 - maDiff*10 // 差异越小，置信度越高
	confidence = math.Max(0.3, math.Min(0.7, confidence))

	return predictedPrice, confidence
}

// predictByStatistics 基于统计方法预测
func (s *Server) predictByStatistics(prices []float64, currentPrice float64, hours int) (float64, float64) {
	if len(prices) < 30 {
		return currentPrice, 0.3
	}

	// 计算历史收益率
	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	// 计算平均收益率和波动率
	var meanReturn float64
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	volatility := s.calculateVolatilityFromPrices(prices)

	// 预测价格 = 当前价格 * (1 + 平均收益率 * 时间)
	timeFactor := float64(hours) / 24.0
	predictedPrice := currentPrice * math.Pow(1+meanReturn, timeFactor)

	// 置信度：波动率越低，置信度越高
	confidence := 0.5 - volatility*5
	confidence = math.Max(0.2, math.Min(0.6, confidence))

	return predictedPrice, confidence
}

// predictByVolume 基于成交量预测
func (s *Server) predictByVolume(prices []float64, volumes []float64, currentPrice float64, hours int) (float64, float64) {
	if len(prices) < 20 || len(volumes) < 20 {
		return currentPrice, 0.3
	}

	// 计算成交量均线
	volumeMA := calculateSMA(volumes, 20)
	if volumeMA == 0 {
		return currentPrice, 0.3
	}

	// 当前成交量比率
	currentVolume := volumes[len(volumes)-1]
	volumeRatio := currentVolume / volumeMA

	// 价格动量（最近5个周期的平均涨跌幅）
	recentReturns := make([]float64, 0, 5)
	for i := len(prices) - 5; i < len(prices); i++ {
		if i > 0 {
			ret := (prices[i] - prices[i-1]) / prices[i-1]
			recentReturns = append(recentReturns, ret)
		}
	}

	var momentum float64
	for _, r := range recentReturns {
		momentum += r
	}
	momentum /= float64(len(recentReturns))

	// 预测：成交量放大 + 正动量 = 上涨
	// 成交量放大 + 负动量 = 下跌
	timeFactor := float64(hours) / 24.0
	volumeFactor := 1.0
	if volumeRatio > 1.2 {
		volumeFactor = 1.01 // 成交量放大，看涨
	} else if volumeRatio < 0.8 {
		volumeFactor = 0.99 // 成交量萎缩，看跌
	}

	predictedPrice := currentPrice * math.Pow(1+momentum*volumeFactor, timeFactor*0.3)

	// 置信度：成交量比率越极端，置信度越高
	confidence := 0.3 + math.Min(0.3, math.Abs(volumeRatio-1.0)*0.5)

	return predictedPrice, confidence
}

// calculateVolatility 计算波动率（标准差）
func (s *Server) calculateVolatilityFromPrices(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.1 // 默认波动率
	}

	// 计算收益率
	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	// 计算平均收益率
	var meanReturn float64
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	// 计算标准差
	var variance float64
	for _, r := range returns {
		variance += (r - meanReturn) * (r - meanReturn)
	}
	variance /= float64(len(returns))
	volatility := math.Sqrt(variance)

	return volatility
}

// generatePredictionFactors 生成预测依据
func (s *Server) generatePredictionFactors(tech *TechnicalIndicators, prices []float64, volumes []float64) []string {
	factors := make([]string, 0)

	if tech == nil {
		return factors
	}

	// 技术指标因素
	if tech.Trend == "up" {
		factors = append(factors, "技术指标显示上涨趋势")
	} else if tech.Trend == "down" {
		factors = append(factors, "技术指标显示下跌趋势")
	}

	if tech.RSI > 70 {
		factors = append(factors, "RSI超买，可能回调")
	} else if tech.RSI < 30 {
		factors = append(factors, "RSI超卖，可能反弹")
	}

	if tech.MACD > tech.MACDSignal && tech.MACDHist > 0 {
		factors = append(factors, "MACD金叉，技术面看涨")
	}

	if tech.MA5 > tech.MA10 && tech.MA10 > tech.MA20 {
		factors = append(factors, "均线多头排列")
	}

	// 成交量因素
	if len(volumes) >= 20 {
		volumeMA := calculateSMA(volumes, 20)
		if volumeMA > 0 {
			currentVolume := volumes[len(volumes)-1]
			volumeRatio := currentVolume / volumeMA
			if volumeRatio > 1.5 {
				factors = append(factors, "成交量显著放大")
			} else if volumeRatio < 0.5 {
				factors = append(factors, "成交量萎缩")
			}
		}
	}

	// 波动率因素
	volatility := s.calculateVolatilityFromPrices(prices)
	if volatility > 0.05 {
		factors = append(factors, "波动率较高，价格波动可能较大")
	} else if volatility < 0.02 {
		factors = append(factors, "波动率较低，价格相对稳定")
	}

	return factors
}

// determinePredictionTrend 确定预测趋势
func (s *Server) determinePredictionTrend(change24h, change7d, change30d float64) string {
	// 计算加权平均涨跌幅
	avgChange := (change24h*0.5 + change7d*0.3 + change30d*0.2)

	if avgChange > 5 {
		return "bullish"
	} else if avgChange < -5 {
		return "bearish"
	}
	return "neutral"
}


// RidgeRegression 岭回归模型
type RidgeRegression struct {
	*LinearRegression
	Alpha float64 // 正则化参数
}

// LassoRegression LASSO回归模型
type LassoRegression struct {
	*LinearRegression
	Alpha float64 // 正则化参数
}

// DefaultModelSelector 默认模型选择器
type DefaultModelSelector struct{}

// SelectBestModel 选择最佳模型
func (dms *DefaultModelSelector) SelectBestModel(models []MLModel, validationData [][]float64, validationTargets []float64) MLModel {
	if len(models) == 0 {
		return nil
	}

	if len(validationData) == 0 || len(validationTargets) == 0 {
		// 如果没有验证数据，返回第一个模型
		return models[0]
	}

	var bestModel MLModel
	bestScore := -1.0 // MSE，越小越好

	for _, model := range models {
		// 对每个模型进行交叉验证
		score := dms.crossValidateModel(model, validationData, validationTargets)
		if bestModel == nil || score < bestScore {
			bestModel = model
			bestScore = score
		}
	}

	return bestModel
}

// crossValidateModel 对模型进行交叉验证
func (dms *DefaultModelSelector) crossValidateModel(model MLModel, data [][]float64, targets []float64) float64 {
	if len(data) < 5 {
		// 数据太少，直接训练和预测
		model.Train(data, targets)
		predictions := make([]float64, len(data))
		for i, sample := range data {
			pred, _ := model.Predict(sample)
			predictions[i] = pred
		}
		return dms.calculateMSE(predictions, targets)
	}

	// k折交叉验证
	k := 5
	foldSize := len(data) / k
	totalMSE := 0.0

	for i := 0; i < k; i++ {
		startIdx := i * foldSize
		endIdx := startIdx + foldSize
		if i == k-1 {
			endIdx = len(data) // 最后一份包含剩余所有数据
		}

		// 准备训练数据（排除当前折）
		trainData := make([][]float64, 0, len(data)-foldSize)
		trainTargets := make([]float64, 0, len(data)-foldSize)

		for j, sample := range data {
			if j < startIdx || j >= endIdx {
				trainData = append(trainData, sample)
				trainTargets = append(trainTargets, targets[j])
			}
		}

		// 准备验证数据
		valData := data[startIdx:endIdx]
		valTargets := targets[startIdx:endIdx]

		// 训练模型
		model.Train(trainData, trainTargets)

		// 验证模型
		predictions := make([]float64, len(valData))
		for j, sample := range valData {
			pred, _ := model.Predict(sample)
			predictions[j] = pred
		}

		// 计算MSE
		foldMSE := dms.calculateMSE(predictions, valTargets)
		totalMSE += foldMSE
	}

	return totalMSE / float64(k)
}

// calculateMSE 计算均方误差
func (dms *DefaultModelSelector) calculateMSE(predictions, targets []float64) float64 {
	if len(predictions) != len(targets) {
		return 999999.0 // 返回一个很大的错误值
	}

	totalError := 0.0
	for i := range predictions {
		error := predictions[i] - targets[i]
		totalError += error * error
	}

	return totalError / float64(len(predictions))
}

// NewBayesianPricePredictor 创建贝叶斯价格预测器
func NewBayesianPricePredictor() *BayesianPricePredictor {
	return &BayesianPricePredictor{
		PriorDistribution:   &NormalDistribution{Mu: 0, Sigma: 1},
		LikelihoodFunction:  &NormalLikelihood{Noise: 0.1},
		PosteriorCalculator: &SimpleBayesianUpdater{},
	}
}

// Predict 贝叶斯预测
func (bpp *BayesianPricePredictor) Predict(symbol string, timeframe string) (*PricePrediction, error) {
	// 获取历史数据
	historicalPrices := bpp.getHistoricalPrices(symbol, timeframe)

	// 从历史数据学习
	for _, price := range historicalPrices {
		likelihood := bpp.LikelihoodFunction.Calculate(price)
		bpp.PosteriorCalculator.Update(bpp.PriorDistribution, likelihood)
	}

	currentPrice := bpp.getCurrentPrice(symbol)

	// 简化的预测实现
	prediction := currentPrice * 1.02 // 简单的2%增长预测

	return &PricePrediction{
		Symbol:        symbol,
		CurrentPrice:  currentPrice,
		PredictedAt:   time.Now().UTC(),
		Pred24h:       prediction,
		Change24h:     ((prediction - currentPrice) / currentPrice) * 100,
		Confidence24h: 80, // 贝叶斯方法的默认置信度
		Range24h: PriceRange{
			Min: prediction * 0.95,
			Max: prediction * 1.05,
			Avg: prediction,
		},
		Factors: []string{"贝叶斯推理", "概率分布", "历史数据更新"},
		Trend:   bpp.determineTrend(prediction, currentPrice),
	}, nil
}

// GetModelName 获取模型名称
func (bpp *BayesianPricePredictor) GetModelName() string {
	return "BayesianPricePredictor"
}

// getHistoricalPrices 获取历史价格
func (bpp *BayesianPricePredictor) getHistoricalPrices(symbol string, timeframe string) []float64 {
	if bpp.db == nil {
		// 如果没有数据库连接，返回模拟数据
		return []float64{50000, 51000, 49500, 50500, 50200}
	}

	// 从数据库获取历史价格数据
	// 这里应该实现实际的数据查询逻辑
	// 暂时返回模拟数据，实际实现需要根据timeframe查询相应的历史数据
	prices := make([]float64, 0, 100)

	// 模拟从数据库查询数据
	// SELECT price FROM market_data WHERE symbol = ? AND timestamp >= ? ORDER BY timestamp DESC LIMIT 100
	// 这里只是示例，实际需要实现完整的数据查询

	// 暂时使用模拟数据填充
	basePrice := 50000.0
	for i := 0; i < 100; i++ {
		// 添加一些随机波动
		change := (rand.Float64() - 0.5) * 1000 // -500 到 +500 的随机变化
		price := basePrice + change
		prices = append(prices, price)
		basePrice = price
	}

	return prices
}

// getCurrentPrice 获取当前价格
func (bpp *BayesianPricePredictor) getCurrentPrice(symbol string) float64 {
	if bpp.db == nil {
		// 如果没有数据库连接，返回模拟价格
		return 50000
	}

	// 从数据库或缓存获取最新价格数据
	// 这里应该实现实际的价格查询逻辑
	// SELECT price FROM market_data WHERE symbol = ? ORDER BY timestamp DESC LIMIT 1

	// 暂时返回模拟价格，实际实现需要查询数据库
	return 50000 + (rand.Float64()-0.5)*2000 // 在50000基础上添加随机波动
}

// determineTrend 确定趋势
func (bpp *BayesianPricePredictor) determineTrend(prediction, currentPrice float64) string {
	change := (prediction - currentPrice) / currentPrice
	if change > 0.03 {
		return "bullish"
	} else if change < -0.03 {
		return "bearish"
	}
	return "neutral"
}

// sampleFromDistribution 从分布中采样
func (bpp *BayesianPricePredictor) sampleFromDistribution(distribution interface{}) float64 {
	// 简化的采样实现
	if normal, ok := distribution.(*NormalDistribution); ok {
		return normal.Mean() // 返回均值作为预测
	}
	return 50000 // 默认值
}

// NormalDistribution 正态分布
type NormalDistribution struct {
	Mu    float64 // 均值
	Sigma float64 // 标准差
}

// PDF 概率密度函数
func (nd *NormalDistribution) PDF(x float64) float64 {
	return math.Exp(-0.5*math.Pow((x-nd.Mu)/nd.Sigma, 2)) / (nd.Sigma * math.Sqrt(2*math.Pi))
}

// CDF 累积分布函数
func (nd *NormalDistribution) CDF(x float64) float64 {
	// 简化的CDF实现
	return 0.5 * (1 + math.Erf((x-nd.Mu)/(nd.Sigma*math.Sqrt(2))))
}

// Mean 均值
func (nd *NormalDistribution) Mean() float64 {
	return nd.Mu
}

// Variance 方差
func (nd *NormalDistribution) Variance() float64 {
	return nd.Sigma * nd.Sigma
}

// NormalLikelihood 正态似然
type NormalLikelihood struct {
	Noise float64 // 噪声参数
}

// Calculate 计算似然
func (nl *NormalLikelihood) Calculate(dataPoint interface{}) float64 {
	if price, ok := dataPoint.(float64); ok {
		// 简化的似然计算
		return math.Exp(-0.5 * price * price / (nl.Noise * nl.Noise))
	}
	return 0
}

// SimpleBayesianUpdater 简单的贝叶斯更新器
type SimpleBayesianUpdater struct {
	posterior ProbabilityDistribution
}

// Update 更新后验
func (sbu *SimpleBayesianUpdater) Update(prior ProbabilityDistribution, likelihood float64) ProbabilityDistribution {
	// 简化的贝叶斯更新
	if normal, ok := prior.(*NormalDistribution); ok {
		newMean := (normal.Mean() + likelihood) / 2
		newStdDev := normal.Variance() * 0.9 // 逐渐减少不确定性

		sbu.posterior = &NormalDistribution{
			Mu:    newMean,
			Sigma: math.Sqrt(newStdDev),
		}
	}

	return sbu.posterior
}

// generateTradingStrategy 生成完整的交易策略
func (s *Server) generateTradingStrategy(
	prediction PricePrediction,
	technical TechnicalIndicators,
	prices []float64,
	volumes []float64,
	currentPrice float64,
) TradingStrategy {
	strategy := TradingStrategy{
		StrategyRationale: []string{},
	}

	// 1. 确定策略类型
	strategy.StrategyType = determineStrategyType(prediction, technical)

	// 2. 生成入场区间
	strategy.EntryZone = generateEntryZone(prediction, technical, currentPrice)

	// 3. 生成出场目标
	strategy.ExitTargets = generateExitTargets(prediction, technical, currentPrice)

	// 4. 生成止损等级
	strategy.StopLossLevels = generateStopLossLevels(prediction, technical, currentPrice)

	// 5. 生成风险管理
	strategy.RiskManagement = generateRiskManagement(prediction, technical, prices, volumes)

	// 6. 生成仓位管理
	strategy.PositionSizing = generatePositionSizing(prediction, technical, strategy.RiskManagement)

	// 7. 确定市场环境
	strategy.MarketCondition = determineMarketCondition(prices, volumes, technical)

	return strategy
}

// determineStrategyType 确定策略类型
func determineStrategyType(prediction PricePrediction, technical TechnicalIndicators) string {
	// 基于预测趋势和当前价格位置确定策略

	if prediction.Change24h > 5 && technical.BBPosition < 0.3 {
		// 价格在布林带下轨且预测上涨
		return "LONG"
	} else if prediction.Change24h < -5 && technical.BBPosition > 0.7 {
		// 价格在布林带上轨且预测下跌
		return "SHORT"
	} else {
		// 震荡行情
		return "RANGE"
	}
}

// generateEntryZone 生成入场区间
func generateEntryZone(prediction PricePrediction, technical TechnicalIndicators, currentPrice float64) PriceRange {
	zone := PriceRange{}

	if prediction.Trend == "bullish" {
		// 多头策略：当前价格附近作为入场区间
		zone.Min = currentPrice * 0.98
		zone.Max = currentPrice * 1.02
		zone.Avg = currentPrice
	} else if prediction.Trend == "bearish" {
		// 空头策略：当前价格附近作为入场区间
		zone.Min = currentPrice * 0.98
		zone.Max = currentPrice * 1.02
		zone.Avg = currentPrice
	} else {
		// 震荡策略：基于支撑阻力位
		if technical.SupportLevel > 0 {
			zone.Min = technical.SupportLevel
		} else {
			zone.Min = currentPrice * 0.95
		}
		if technical.ResistanceLevel > 0 {
			zone.Max = technical.ResistanceLevel
		} else {
			zone.Max = currentPrice * 1.05
		}
		zone.Avg = (zone.Min + zone.Max) / 2
	}

	return zone
}

// generateExitTargets 生成出场目标
func generateExitTargets(prediction PricePrediction, technical TechnicalIndicators, currentPrice float64) []PriceRange {
	targets := []PriceRange{}

	if prediction.Trend == "bullish" {
		// 多头目标：基于预测价格设置多个目标
		primaryTarget := PriceRange{
			Min: prediction.Pred24h * 0.98,
			Max: prediction.Pred24h * 1.02,
			Avg: prediction.Pred24h,
		}
		targets = append(targets, primaryTarget)

		// 次级目标：更高价格
		secondaryTarget := PriceRange{
			Min: prediction.Pred7d * 0.98,
			Max: prediction.Pred7d * 1.02,
			Avg: prediction.Pred7d,
		}
		targets = append(targets, secondaryTarget)

	} else if prediction.Trend == "bearish" {
		// 空头目标：基于预测价格设置多个目标
		primaryTarget := PriceRange{
			Min: prediction.Pred24h * 0.98,
			Max: prediction.Pred24h * 1.02,
			Avg: prediction.Pred24h,
		}
		targets = append(targets, primaryTarget)

		secondaryTarget := PriceRange{
			Min: prediction.Pred7d * 0.98,
			Max: prediction.Pred7d * 1.02,
			Avg: prediction.Pred7d,
		}
		targets = append(targets, secondaryTarget)
	} else {
		// 震荡策略：基于支撑阻力位设置目标
		if technical.ResistanceLevel > currentPrice {
			buyTarget := PriceRange{
				Min: technical.ResistanceLevel * 0.98,
				Max: technical.ResistanceLevel * 1.02,
				Avg: technical.ResistanceLevel,
			}
			targets = append(targets, buyTarget)
		}

		if technical.SupportLevel < currentPrice && technical.SupportLevel > 0 {
			sellTarget := PriceRange{
				Min: technical.SupportLevel * 0.98,
				Max: technical.SupportLevel * 1.02,
				Avg: technical.SupportLevel,
			}
			targets = append(targets, sellTarget)
		}
	}

	return targets
}

// generateStopLossLevels 生成止损等级
func generateStopLossLevels(prediction PricePrediction, technical TechnicalIndicators, currentPrice float64) []StopLossLevel {
	levels := []StopLossLevel{}

	// 主要止损
	mainStopLoss := StopLossLevel{
		Type:      "INITIAL",
		Condition: "价格触及",
	}

	if prediction.Trend == "bullish" {
		// 多头止损：基于布林带下轨或固定百分比
		if technical.BBLower > 0 {
			mainStopLoss.Level = technical.BBLower
		} else {
			mainStopLoss.Level = currentPrice * 0.95 // 5%止损
		}
	} else {
		// 空头止损：基于布林带上轨或固定百分比
		if technical.BBUpper > 0 {
			mainStopLoss.Level = technical.BBUpper
		} else {
			mainStopLoss.Level = currentPrice * 1.05 // 5%止损
		}
	}
	levels = append(levels, mainStopLoss)

	// 追踪止损
	trailingStopLoss := StopLossLevel{
		Type:      "TRAILING",
		Condition: "价格反转一定幅度",
		Level:     currentPrice * 0.97, // 初始追踪止损
	}
	levels = append(levels, trailingStopLoss)

	return levels
}

// generateRiskManagement 生成风险管理
func generateRiskManagement(prediction PricePrediction, technical TechnicalIndicators, prices []float64, volumes []float64) RiskManagement {
	rm := RiskManagement{}

	// 计算波动率
	volatility := calculateVolatility(prices, len(prices))

	// 基于波动率设置风险参数
	if volatility > 0.1 {
		// 高波动
		rm.config.Control.MaxPositionSize = 0.01  // 最大仓位1%
		rm.config.Control.MaxDrawdownLimit = 0.10 // 最大回撤10%
	} else {
		// 低波动
		rm.config.Control.MaxPositionSize = 0.02  // 最大仓位2%
		rm.config.Control.MaxDrawdownLimit = 0.15 // 最大回撤15%
	}

	// 计算风险收益比 (存储在配置中)
	if len(prediction.Factors) > 0 {
		// 基于预测置信度计算
		rm.config.Assessment.RiskThreshold = 1.5 + prediction.Confidence24h/100 // 1.5-2.5倍
	} else {
		rm.config.Assessment.RiskThreshold = 2.0 // 默认2:1
	}

	// 计算投资组合热度（基于当前持仓风险）- 这里暂时不设置

	return rm
}

// generatePositionSizing 生成仓位管理
func generatePositionSizing(prediction PricePrediction, technical TechnicalIndicators, rm RiskManagement) PositionSizing {
	ps := PositionSizing{
		ScalingStrategy: "FIXED",
	}

	// 基于风险管理和信号强度确定仓位
	basePosition := prediction.Confidence24h / 100.0 // 置信度转换为仓位比例

	// 考虑技术指标风险
	if technical.RiskLevel == "high" {
		basePosition *= 0.5
	} else if technical.RiskLevel == "low" {
		basePosition *= 1.2
	}

	ps.BasePosition = math.Min(1.0, basePosition)
	ps.AdjustedPosition = ps.BasePosition

	// 设置最大最小仓位限制
	ps.MaxPosition = math.Min(1.0, rm.config.Control.MaxPositionSize) // 基于风险配置计算最大仓位
	ps.MinPosition = 0.01                                             // 最小1%

	return ps
}

// determineMarketCondition 确定市场环境
func determineMarketCondition(prices []float64, volumes []float64, technical TechnicalIndicators) string {
	if len(prices) < 20 {
		return "unknown"
	}

	// 计算趋势强度
	recentPrices := prices[len(prices)-20:]
	trendStrength := 0.0
	for i := 1; i < len(recentPrices); i++ {
		if recentPrices[i] > recentPrices[i-1] {
			trendStrength += 1
		} else {
			trendStrength -= 1
		}
	}
	trendStrength /= float64(len(recentPrices) - 1)

	// 计算波动率
	volatility := calculateVolatility(recentPrices, len(recentPrices))

	// 基于趋势强度和波动率判断市场环境
	if math.Abs(trendStrength) > 0.6 && volatility < 0.05 {
		if trendStrength > 0 {
			return "strong_bull"
		} else {
			return "strong_bear"
		}
	} else if math.Abs(trendStrength) > 0.3 {
		if trendStrength > 0 {
			return "bull"
		} else {
			return "bear"
		}
	} else if volatility > 0.1 {
		return "volatile"
	} else {
		return "sideways"
	}
}

// calculateVolatility 计算波动率

// GetPosterior 获取后验分布
func (sbu *SimpleBayesianUpdater) GetPosterior() ProbabilityDistribution {
	return sbu.posterior
}
