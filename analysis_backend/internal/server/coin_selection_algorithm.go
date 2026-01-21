package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// 新一代选币算法框架
// ============================================================================

// CoinSelectionAlgorithm 新一代选币算法
type CoinSelectionAlgorithm struct {
	// 算法配置
	config AlgorithmConfig

	// 评分因子
	factors []ScoringFactor

	// 风险评估器
	riskAssessor *RiskAssessor

	// 市场状态分析器
	marketAnalyzer *MarketStateAnalyzer

	// 策略选择器
	strategySelector *StrategySelector

	// 特征工程模块 ⭐ 新增
	featureEngineering *FeatureEngineering

	// 机器学习模块 ⭐ 新增
	machineLearning *MachineLearning

	// 风险管理模块 ⭐ 新增
	riskManagement *RiskManagement
}

// AlgorithmConfig 算法配置
type AlgorithmConfig struct {
	// 基础配置
	MaxCandidates int     `json:"max_candidates"` // 最大候选数量
	MinScore      float64 `json:"min_score"`      // 最低评分阈值
	RiskTolerance float64 `json:"risk_tolerance"` // 风险容忍度 (0-1)

	// 评分权重
	Weights struct {
		Technical   float64 `json:"technical"`   // 技术指标权重
		Fundamental float64 `json:"fundamental"` // 基本面权重
		Sentiment   float64 `json:"sentiment"`   // 情绪指标权重
		Risk        float64 `json:"risk"`        // 风险权重
		Momentum    float64 `json:"momentum"`    // 动量权重
	} `json:"weights"`

	// 市场适应性
	MarketAdaptation struct {
		EnableDynamicWeights bool    `json:"enable_dynamic_weights"` // 启用动态权重
		BullMarketBias       float64 `json:"bull_market_bias"`       // 牛市偏好
		BearMarketBias       float64 `json:"bear_market_bias"`       // 熊市偏好
		SidewaysBias         float64 `json:"sideways_bias"`          // 震荡市偏好
	} `json:"market_adaptation"`

	// 策略配置
	Strategies struct {
		EnableLong     bool `json:"enable_long"`     // 启用多头策略
		EnableShort    bool `json:"enable_short"`    // 启用空头策略
		EnableRange    bool `json:"enable_range"`    // 启用震荡策略
		EnableMomentum bool `json:"enable_momentum"` // 启用动量策略
	} `json:"strategies"`
}

// DefaultAlgorithmConfig 默认算法配置
func DefaultAlgorithmConfig() AlgorithmConfig {
	return AlgorithmConfig{
		MaxCandidates: 50,
		MinScore:      40.0, // 降低最低分数要求
		RiskTolerance: 0.5,  // 提高风险容忍度，允许风险评分到50分

		Weights: struct {
			Technical   float64 `json:"technical"`
			Fundamental float64 `json:"fundamental"`
			Sentiment   float64 `json:"sentiment"`
			Risk        float64 `json:"risk"`
			Momentum    float64 `json:"momentum"`
		}{
			Technical:   0.25,
			Fundamental: 0.20,
			Sentiment:   0.15,
			Risk:        0.20,
			Momentum:    0.20,
		},

		MarketAdaptation: struct {
			EnableDynamicWeights bool    `json:"enable_dynamic_weights"`
			BullMarketBias       float64 `json:"bull_market_bias"`
			BearMarketBias       float64 `json:"bear_market_bias"`
			SidewaysBias         float64 `json:"sideways_bias"`
		}{
			EnableDynamicWeights: true,
			BullMarketBias:       0.1,
			BearMarketBias:       -0.1,
			SidewaysBias:         0.0,
		},

		Strategies: struct {
			EnableLong     bool `json:"enable_long"`
			EnableShort    bool `json:"enable_short"`
			EnableRange    bool `json:"enable_range"`
			EnableMomentum bool `json:"enable_momentum"`
		}{
			EnableLong:     true,
			EnableShort:    false, // 默认关闭空头策略
			EnableRange:    true,
			EnableMomentum: true,
		},
	}
}

// NewCoinSelectionAlgorithm 创建新的选币算法实例
func NewCoinSelectionAlgorithm(config AlgorithmConfig) *CoinSelectionAlgorithm {
	algorithm := &CoinSelectionAlgorithm{
		config:           config,
		riskAssessor:     NewRiskAssessor(),
		marketAnalyzer:   NewMarketStateAnalyzer(),
		strategySelector: NewStrategySelector(),
	}

	// 初始化评分因子
	algorithm.initializeFactors()

	// 初始化特征工程模块 ⭐ 新增
	algorithm.initializeFeatureEngineering()

	return algorithm
}

// GetMarketAnalyzer 获取市场状态分析器
func (alg *CoinSelectionAlgorithm) GetMarketAnalyzer() *MarketStateAnalyzer {
	return alg.marketAnalyzer
}

// initializeFactors 初始化评分因子
func (alg *CoinSelectionAlgorithm) initializeFactors() {
	alg.factors = []ScoringFactor{
		&TechnicalFactor{weight: alg.config.Weights.Technical},
		&FundamentalFactor{weight: alg.config.Weights.Fundamental},
		&SentimentFactor{weight: alg.config.Weights.Sentiment},
		&MomentumFactor{weight: alg.config.Weights.Momentum},
	}
}

// initializeFeatureEngineering 初始化特征工程模块 ⭐ 新增
func (alg *CoinSelectionAlgorithm) initializeFeatureEngineering() {
	// 这里需要访问数据库和数据融合器
	// 暂时设置为nil，后续在Server初始化时设置
	alg.featureEngineering = nil
}

// SetFeatureEngineering 设置特征工程模块 ⭐ 新增
func (alg *CoinSelectionAlgorithm) SetFeatureEngineering(fe *FeatureEngineering) {
	alg.featureEngineering = fe
}

// SetMachineLearning 设置机器学习模块 ⭐ 新增
func (alg *CoinSelectionAlgorithm) SetMachineLearning(ml *MachineLearning) {
	alg.machineLearning = ml
}

// SetRiskManagement 设置风险管理模块 ⭐ 新增
func (alg *CoinSelectionAlgorithm) SetRiskManagement(rm *RiskManagement) {
	alg.riskManagement = rm
}

// SelectCoins 执行选币算法
func (alg *CoinSelectionAlgorithm) SelectCoins(ctx context.Context, marketData []MarketDataPoint, marketState MarketState) ([]CoinRecommendation, error) {
	log.Printf("[CoinSelection] Starting coin selection with %d market data points", len(marketData))

	// 输入数据验证
	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data provided for coin selection")
	}

	if len(marketData) < 3 {
		log.Printf("[WARN] Very limited market data (%d points), results may be unreliable", len(marketData))
	}

	// 1. 特征工程 - 提取市场特征
	features, err := alg.safeExtractFeatures(ctx, marketData, marketState)
	if err != nil {
		log.Printf("[ERROR] Feature extraction failed: %v", err)
		return nil, fmt.Errorf("feature extraction failed: %w", err)
	}
	log.Printf("[CoinSelection] Extracted features for %d coins", len(features))

	if len(features) == 0 {
		return nil, fmt.Errorf("no valid features extracted from market data")
	}

	// 2. 市场状态分析
	marketAnalysis, err := alg.safeAnalyzeMarketState(marketData)
	if err != nil {
		log.Printf("[ERROR] Market state analysis failed: %v", err)
		return nil, fmt.Errorf("market state analysis failed: %w", err)
	}
	log.Printf("[CoinSelection] Market state: %s (avg_change: %.2f%%, volatility: %.2f)",
		marketAnalysis.State, marketAnalysis.AvgChange*100, marketAnalysis.Volatility)

	// 3. 动态权重调整
	dynamicWeights, err := alg.safeCalculateDynamicWeights(marketState, marketAnalysis)
	if err != nil {
		log.Printf("[WARN] Dynamic weights calculation failed, using defaults: %v", err)
		dynamicWeights = alg.config // 使用默认配置
	}

	// 4. 多维度评分
	scores, err := alg.safeCalculateMultiDimensionalScores(features, dynamicWeights)
	if err != nil {
		log.Printf("[ERROR] Multi-dimensional scoring failed: %v", err)
		return nil, fmt.Errorf("multi-dimensional scoring failed: %w", err)
	}
	log.Printf("[CoinSelection] Calculated scores for %d coins", len(scores))

	if len(scores) == 0 {
		return nil, fmt.Errorf("no valid scores calculated")
	}

	// ⭐ 4.5. 机器学习增强评分 (可选)
	if alg.machineLearning != nil {
		scores, err = alg.safeApplyMachineLearningEnhancement(scores, marketData)
		if err != nil {
			log.Printf("[WARN] Machine learning enhancement failed, continuing without ML: %v", err)
		} else {
			log.Printf("[CoinSelection] Applied machine learning enhancement")
		}
	}

	// ⭐ 4.6. 风险控制决策 (可选)
	riskControlled := scores
	if alg.riskManagement != nil {
		riskControlled, err = alg.safeApplyRiskControl(scores)
		if err != nil {
			log.Printf("[WARN] Risk control failed, continuing without risk control: %v", err)
		} else {
			log.Printf("[CoinSelection] Applied risk control filtering")
		}
	}

	// 5. 传统风险评估和过滤
	traditionalRiskFiltered, err := alg.safeApplyRiskFiltering(riskControlled, features)
	if err != nil {
		log.Printf("[WARN] Traditional risk filtering failed, continuing with current scores: %v", err)
		// 将 map 转换为 slice
		traditionalRiskFiltered = make([]*CoinScore, 0, len(riskControlled))
		for _, score := range riskControlled {
			traditionalRiskFiltered = append(traditionalRiskFiltered, score)
		}
	}
	log.Printf("[CoinSelection] After traditional risk filtering: %d coins remain", len(traditionalRiskFiltered))

	if len(traditionalRiskFiltered) == 0 {
		return nil, fmt.Errorf("no coins passed risk filtering")
	}

	// 6. 策略匹配和排序
	finalRecommendations, err := alg.safeApplyStrategySelection(traditionalRiskFiltered, marketState)
	if err != nil {
		log.Printf("[ERROR] Strategy selection failed: %v", err)
		return nil, fmt.Errorf("strategy selection failed: %w", err)
	}
	log.Printf("[CoinSelection] Final recommendations: %d coins", len(finalRecommendations))

	if len(finalRecommendations) == 0 {
		return nil, fmt.Errorf("no final recommendations generated")
	}

	return finalRecommendations, nil
}

// ============================================================================
// 安全方法包装器 (带错误处理)
// ============================================================================

// safeExtractFeatures 安全的特征提取
func (alg *CoinSelectionAlgorithm) safeExtractFeatures(ctx context.Context, marketData []MarketDataPoint, marketState MarketState) (map[string]*CoinFeatures, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Feature extraction panicked: %v", r)
		}
	}()

	features := alg.extractFeatures(ctx, marketData, marketState)
	if features == nil {
		return nil, fmt.Errorf("extractFeatures returned nil")
	}
	return features, nil
}

// safeAnalyzeMarketState 安全的市场状态分析
func (alg *CoinSelectionAlgorithm) safeAnalyzeMarketState(marketData []MarketDataPoint) (*MarketAnalysis, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Market state analysis panicked: %v", r)
		}
	}()

	analysis := alg.marketAnalyzer.AnalyzeMarketState(marketData)
	if analysis.State == "" {
		return nil, fmt.Errorf("market state analysis returned empty state")
	}
	return &analysis, nil
}

// safeCalculateDynamicWeights 安全的动态权重计算
func (alg *CoinSelectionAlgorithm) safeCalculateDynamicWeights(marketState MarketState, marketAnalysis *MarketAnalysis) (AlgorithmConfig, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Dynamic weights calculation panicked: %v", r)
		}
	}()

	weights := alg.calculateDynamicWeights(marketState, *marketAnalysis)
	// AlgorithmConfig 是一个结构体，不能用 len() 检查
	return weights, nil
}

// safeCalculateMultiDimensionalScores 安全的多维度评分
func (alg *CoinSelectionAlgorithm) safeCalculateMultiDimensionalScores(features map[string]*CoinFeatures, weights AlgorithmConfig) (map[string]*CoinScore, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Multi-dimensional scoring panicked: %v", r)
		}
	}()

	scores := alg.calculateMultiDimensionalScores(features, weights)
	if len(scores) == 0 {
		return nil, fmt.Errorf("calculateMultiDimensionalScores returned no scores")
	}
	return scores, nil
}

// safeApplyMachineLearningEnhancement 安全的机器学习增强
func (alg *CoinSelectionAlgorithm) safeApplyMachineLearningEnhancement(scores map[string]*CoinScore, marketData []MarketDataPoint) (map[string]*CoinScore, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Machine learning enhancement panicked: %v", r)
		}
	}()

	enhanced := alg.applyMachineLearningEnhancement(scores, marketData)
	if len(enhanced) == 0 {
		return scores, fmt.Errorf("machine learning enhancement returned no scores")
	}
	return enhanced, nil
}

// safeApplyRiskControl 安全的风险控制
func (alg *CoinSelectionAlgorithm) safeApplyRiskControl(scores map[string]*CoinScore) (map[string]*CoinScore, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Risk control panicked: %v", r)
		}
	}()

	controlled := alg.applyRiskControl(scores)
	return controlled, nil
}

// safeApplyRiskFiltering 安全的风险过滤
func (alg *CoinSelectionAlgorithm) safeApplyRiskFiltering(scores map[string]*CoinScore, features map[string]*CoinFeatures) ([]*CoinScore, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Risk filtering panicked: %v", r)
		}
	}()

	filtered := alg.applyRiskFiltering(scores, features)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no coins passed risk filtering")
	}
	return filtered, nil
}

// safeApplyStrategySelection 安全的策略选择
func (alg *CoinSelectionAlgorithm) safeApplyStrategySelection(scores []*CoinScore, marketState MarketState) ([]CoinRecommendation, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] Strategy selection panicked: %v", r)
		}
	}()

	recommendations := alg.applyStrategySelection(scores, marketState)
	if len(recommendations) == 0 {
		return nil, fmt.Errorf("applyStrategySelection returned no recommendations")
	}
	return recommendations, nil
}

// getDefaultWeights 获取默认权重
func (alg *CoinSelectionAlgorithm) getDefaultWeights() map[string]float64 {
	return map[string]float64{
		"technical":   0.25,
		"fundamental": 0.20,
		"sentiment":   0.15,
		"momentum":    0.20,
		"risk":        0.20,
	}
}

// ============================================================================
// 核心数据结构
// ============================================================================

// MarketDataPoint 市场数据点
type MarketDataPoint struct {
	Symbol            string
	BaseSymbol        string
	Price             float64
	PriceChange24h    float64
	Volume24h         float64
	MarketCap         *float64
	FlowTrend         *FlowTrendResult
	SentimentData     *SentimentResult
	TechnicalData     *TechnicalIndicators
	AnnouncementScore *AnnouncementScore
	Timestamp         time.Time
}

// CoinFeatures 币种特征
type CoinFeatures struct {
	Symbol string

	// 基础特征
	Price          float64
	PriceChange24h float64
	Volume24h      float64
	MarketCap      *float64
	Volatility     float64

	// 技术特征
	RSI           float64
	MACD          float64
	MACDSignal    float64
	BBPosition    float64
	KDJ           [3]float64 // K, D, J
	MA5           float64
	MA10          float64
	MA20          float64
	TrendStrength float64
	VolumeRatio   float64

	// 基本面特征
	FlowScore         float64
	SentimentScore    float64
	AnnouncementScore float64
	HeatScore         float64

	// 动量特征
	Momentum1h   float64
	Momentum4h   float64
	Momentum24h  float64
	Acceleration float64

	// 风险特征
	RiskVolatility float64
	RiskLiquidity  float64
	RiskMarket     float64
	RiskTechnical  float64
	OverallRisk    float64

	// ⭐ 新特征工程字段
	TimeSeriesFeatures  map[string]float64 // 时间序列特征
	VolatilityFeatures  map[string]float64 // 波动率特征
	TrendFeatures       map[string]float64 // 趋势特征
	MomentumFeatures    map[string]float64 // 动量特征
	CrossFeatures       map[string]float64 // 交叉特征
	StatisticalFeatures map[string]float64 // 统计特征
}

// CoinScore 币种评分
type CoinScore struct {
	Symbol       string
	BaseSymbol   string
	TotalScore   float64
	StrategyType string

	Scores struct {
		Technical   float64
		Fundamental float64
		Sentiment   float64
		Risk        float64
		Momentum    float64
	}

	RiskMetrics struct {
		VolatilityRisk float64
		LiquidityRisk  float64
		MarketRisk     float64
		TechnicalRisk  float64
		OverallRisk    float64
		RiskLevel      string
	}

	Features   *CoinFeatures
	Reasons    []string
	Confidence float64 // 置信度 (0-1)
}

// CoinRecommendation 币种推荐结果
type CoinRecommendation struct {
	CoinScore
	Rank          int
	Data          MarketDataPoint
	Prediction    *PricePrediction
	RecommendedAt time.Time
}

// ============================================================================
// 特征工程
// ============================================================================

// extractFeatures 提取特征
func (alg *CoinSelectionAlgorithm) extractFeatures(ctx context.Context, marketData []MarketDataPoint, marketState MarketState) map[string]*CoinFeatures {
	features := make(map[string]*CoinFeatures)

	for _, data := range marketData {
		feature := &CoinFeatures{
			Symbol:         data.Symbol,
			Price:          data.Price,
			PriceChange24h: data.PriceChange24h,
			Volume24h:      data.Volume24h,
			MarketCap:      data.MarketCap,
		}

		// ⭐ 使用新的特征工程模块提取高级特征
		if alg.featureEngineering != nil {
			// 提取时间序列特征
			timeSeriesFeatures, err := alg.extractTimeSeriesFeatures(ctx, data)
			if err == nil {
				feature.TimeSeriesFeatures = timeSeriesFeatures
			}

			// 提取波动率特征
			volatilityFeatures, err := alg.extractVolatilityFeatures(ctx, data)
			if err == nil {
				feature.VolatilityFeatures = volatilityFeatures
			}

			// 提取趋势特征
			trendFeatures, err := alg.extractTrendFeatures(ctx, data)
			if err == nil {
				feature.TrendFeatures = trendFeatures
			}

			// 提取动量特征
			momentumFeatures, err := alg.extractMomentumFeaturesAdvanced(ctx, data)
			if err == nil {
				feature.MomentumFeatures = momentumFeatures
			}

			// 提取交叉特征
			crossFeatures, err := alg.extractCrossFeatures(ctx, data)
			if err == nil {
				feature.CrossFeatures = crossFeatures
			}

			// 提取统计特征
			statisticalFeatures, err := alg.extractStatisticalFeatures(ctx, data)
			if err == nil {
				feature.StatisticalFeatures = statisticalFeatures
			}
		} else {
			// 回退到原有特征提取方法
			alg.extractLegacyFeatures(feature, data, marketState)
		}

		// 提取技术指标特征 (保持兼容性)
		alg.extractTechnicalFeatures(feature, data)

		// 提取基本面特征
		alg.extractFundamentalFeatures(feature, data)

		// 提取动量特征 (原有方法)
		alg.extractMomentumFeatures(feature, data)

		// 提取风险特征
		alg.extractRiskFeatures(feature, data, marketState)

		features[data.Symbol] = feature
	}

	return features
}

// extractTechnicalFeatures 提取技术指标特征
func (alg *CoinSelectionAlgorithm) extractTechnicalFeatures(feature *CoinFeatures, data MarketDataPoint) {
	if data.TechnicalData == nil {
		return
	}

	tech := data.TechnicalData
	feature.RSI = tech.RSI
	feature.MACD = tech.MACD
	feature.MACDSignal = tech.MACDSignal
	feature.BBPosition = tech.BBPosition
	feature.KDJ = [3]float64{tech.K, tech.D, tech.J}
	feature.MA5 = tech.MA5
	feature.MA10 = tech.MA10
	feature.MA20 = tech.MA20
	feature.VolumeRatio = tech.VolumeRatio

	// 计算趋势强度
	feature.TrendStrength = alg.calculateTrendStrength(tech)
}

// extractFundamentalFeatures 提取基本面特征
func (alg *CoinSelectionAlgorithm) extractFundamentalFeatures(feature *CoinFeatures, data MarketDataPoint) {
	// 资金流评分
	if data.FlowTrend != nil {
		feature.FlowScore = CalculateFlowScoreWithTrend(
			data.FlowTrend.Flow24h,
			data.FlowTrend.Trend3d,
			data.FlowTrend.Trend7d,
		)
	}

	// 情绪评分
	if data.SentimentData != nil {
		feature.SentimentScore = data.SentimentData.Score
	}

	// 公告评分
	if data.AnnouncementScore != nil {
		feature.AnnouncementScore = data.AnnouncementScore.TotalScore
	}

	// 热度评分（基于市值和成交量）
	feature.HeatScore = alg.calculateHeatScore(feature)
}

// extractMomentumFeatures 提取动量特征
func (alg *CoinSelectionAlgorithm) extractMomentumFeatures(feature *CoinFeatures, data MarketDataPoint) {
	// 简化版动量计算（实际应该基于历史数据）
	feature.Momentum1h = data.PriceChange24h / 24 // 估算1小时动量
	feature.Momentum4h = data.PriceChange24h / 6  // 估算4小时动量
	feature.Momentum24h = data.PriceChange24h

	// 计算加速度（动量变化率）
	feature.Acceleration = feature.Momentum4h - feature.Momentum1h
}

// extractRiskFeatures 提取风险特征
func (alg *CoinSelectionAlgorithm) extractRiskFeatures(feature *CoinFeatures, data MarketDataPoint, marketState MarketState) {
	// 波动率风险 - 根据市场状态调整
	absChange := math.Abs(data.PriceChange24h)

	// 在牛市环境中，降低波动率风险评分（因为涨幅是正常现象）
	var volatilityMultiplier float64 = 1.0
	if marketState.State == "bull" {
		volatilityMultiplier = 0.6 // 在牛市时降低波动率风险60%
	}

	if absChange > 30 {
		feature.RiskVolatility = 100 * volatilityMultiplier
	} else if absChange > 20 {
		feature.RiskVolatility = 80 * volatilityMultiplier
	} else if absChange > 10 {
		feature.RiskVolatility = 60 * volatilityMultiplier
	} else {
		feature.RiskVolatility = absChange * 2 * volatilityMultiplier
	}

	// 流动性风险
	if data.Volume24h < 1000000 {
		feature.RiskLiquidity = 80
	} else if data.Volume24h < 5000000 {
		feature.RiskLiquidity = 60
	} else {
		feature.RiskLiquidity = 20
	}

	// 市场风险
	if data.MarketCap != nil {
		if *data.MarketCap < 50000000 {
			feature.RiskMarket = 80
		} else if *data.MarketCap < 200000000 {
			feature.RiskMarket = 60
		} else {
			feature.RiskMarket = 20
		}
	} else {
		feature.RiskMarket = 70 // 没有市值数据，风险较高
	}

	// 技术风险
	feature.RiskTechnical = 30 // 基础风险评分

	// 综合风险
	feature.OverallRisk = (feature.RiskVolatility*0.3 +
		feature.RiskLiquidity*0.3 +
		feature.RiskMarket*0.25 +
		feature.RiskTechnical*0.15)
}

// ============================================================================
// 评分因子系统
// ============================================================================

// ScoringFactor 评分因子接口
type ScoringFactor interface {
	Name() string
	Weight() float64
	Score(features *CoinFeatures, marketState MarketState) float64
}

// TechnicalFactor 技术指标因子
type TechnicalFactor struct {
	weight float64
}

func (f *TechnicalFactor) Name() string    { return "technical" }
func (f *TechnicalFactor) Weight() float64 { return f.weight }

func (f *TechnicalFactor) Score(features *CoinFeatures, marketState MarketState) float64 {
	score := 50.0 // 基础分数

	// RSI 评分
	if features.RSI > 0 {
		if features.RSI < 30 {
			score += 20 // 超卖，可能反弹
		} else if features.RSI > 70 {
			score -= 10 // 超买，谨慎
		} else {
			score += 5 // 正常区间
		}
	}

	// MACD 评分
	if features.MACD > features.MACDSignal {
		score += 15 // 金叉信号
	} else if features.MACD < features.MACDSignal {
		score -= 10 // 死叉信号
	}

	// 布林带位置评分
	if features.BBPosition > 0 {
		if features.BBPosition < 0.2 {
			score += 10 // 接近下轨，可能反弹
		} else if features.BBPosition > 0.8 {
			score -= 10 // 接近上轨，可能回调
		}
	}

	// 均线系统评分
	if features.MA5 > 0 && features.MA10 > 0 && features.MA20 > 0 {
		if features.MA5 > features.MA10 && features.MA10 > features.MA20 {
			score += 15 // 多头排列
		} else if features.MA5 < features.MA10 && features.MA10 < features.MA20 {
			score -= 15 // 空头排列
		}
	}

	return math.Max(0, math.Min(100, score))
}

// FundamentalFactor 基本面因子
type FundamentalFactor struct {
	weight float64
}

func (f *FundamentalFactor) Name() string    { return "fundamental" }
func (f *FundamentalFactor) Weight() float64 { return f.weight }

func (f *FundamentalFactor) Score(features *CoinFeatures, marketState MarketState) float64 {
	score := 50.0

	// 资金流评分
	score += features.FlowScore * 0.3

	// 情绪评分
	score += features.SentimentScore * 0.2

	// 公告评分
	score += features.AnnouncementScore * 0.2

	// 热度评分
	score += features.HeatScore * 0.3

	return math.Max(0, math.Min(100, score))
}

// SentimentFactor 情绪因子
type SentimentFactor struct {
	weight float64
}

func (f *SentimentFactor) Name() string    { return "sentiment" }
func (f *SentimentFactor) Weight() float64 { return f.weight }

func (f *SentimentFactor) Score(features *CoinFeatures, marketState MarketState) float64 {
	return features.SentimentScore
}

// MomentumFactor 动量因子
type MomentumFactor struct {
	weight float64
}

func (f *MomentumFactor) Name() string    { return "momentum" }
func (f *MomentumFactor) Weight() float64 { return f.weight }

func (f *MomentumFactor) Score(features *CoinFeatures, marketState MarketState) float64 {
	score := 50.0

	// 24小时动量
	if features.Momentum24h > 10 {
		score += 20
	} else if features.Momentum24h < -10 {
		score -= 20
	}

	// 加速度
	if features.Acceleration > 5 {
		score += 10 // 加速上涨
	} else if features.Acceleration < -5 {
		score -= 10 // 加速下跌
	}

	return math.Max(0, math.Min(100, score))
}

// ============================================================================
// 评分计算
// ============================================================================

// calculateDynamicWeights 计算动态权重
func (alg *CoinSelectionAlgorithm) calculateDynamicWeights(marketState MarketState, marketAnalysis MarketAnalysis) AlgorithmConfig {
	weights := alg.config

	if !alg.config.MarketAdaptation.EnableDynamicWeights {
		return weights
	}

	// 根据市场状态调整权重
	switch marketAnalysis.State {
	case "bull":
		// 牛市：增加动量权重，减少风险权重
		weights.Weights.Momentum += alg.config.MarketAdaptation.BullMarketBias
		weights.Weights.Risk -= alg.config.MarketAdaptation.BullMarketBias

	case "bear":
		// 熊市：增加基本面权重，减少动量权重
		weights.Weights.Fundamental += math.Abs(alg.config.MarketAdaptation.BearMarketBias)
		weights.Weights.Momentum += alg.config.MarketAdaptation.BearMarketBias

	case "sideways":
		// 震荡市：增加技术指标权重
		weights.Weights.Technical += alg.config.MarketAdaptation.SidewaysBias
	}

	// 归一化权重
	totalWeight := weights.Weights.Technical + weights.Weights.Fundamental +
		weights.Weights.Sentiment + weights.Weights.Risk + weights.Weights.Momentum

	if totalWeight > 0 {
		weights.Weights.Technical /= totalWeight
		weights.Weights.Fundamental /= totalWeight
		weights.Weights.Sentiment /= totalWeight
		weights.Weights.Risk /= totalWeight
		weights.Weights.Momentum /= totalWeight
	}

	return weights
}

// calculateMultiDimensionalScores 计算多维度评分
func (alg *CoinSelectionAlgorithm) calculateMultiDimensionalScores(features map[string]*CoinFeatures, weights AlgorithmConfig) map[string]*CoinScore {
	scores := make(map[string]*CoinScore)

	for symbol, feature := range features {
		score := &CoinScore{
			Symbol:     symbol,
			BaseSymbol: extractBaseSymbol(symbol),
			Features:   feature,
		}

		// 计算各维度评分
		marketState := MarketState{} // 这里应该传入实际的市场状态
		score.Scores.Technical = alg.factors[0].Score(feature, marketState) * weights.Weights.Technical
		score.Scores.Fundamental = alg.factors[1].Score(feature, marketState) * weights.Weights.Fundamental
		score.Scores.Sentiment = alg.factors[2].Score(feature, marketState) * weights.Weights.Sentiment
		score.Scores.Momentum = alg.factors[3].Score(feature, marketState) * weights.Weights.Momentum

		// 风险评分（负权重）
		riskScore := 100 - feature.OverallRisk
		score.Scores.Risk = riskScore * weights.Weights.Risk

		// 总分计算
		score.TotalScore = score.Scores.Technical + score.Scores.Fundamental +
			score.Scores.Sentiment + score.Scores.Risk + score.Scores.Momentum

		// 风险指标
		score.RiskMetrics.VolatilityRisk = feature.RiskVolatility
		score.RiskMetrics.LiquidityRisk = feature.RiskLiquidity
		score.RiskMetrics.MarketRisk = feature.RiskMarket
		score.RiskMetrics.TechnicalRisk = feature.RiskTechnical
		score.RiskMetrics.OverallRisk = feature.OverallRisk

		if score.RiskMetrics.OverallRisk < 30 {
			score.RiskMetrics.RiskLevel = "low"
		} else if score.RiskMetrics.OverallRisk < 60 {
			score.RiskMetrics.RiskLevel = "medium"
		} else {
			score.RiskMetrics.RiskLevel = "high"
		}

		// 生成推荐理由
		score.Reasons = alg.generateReasons(score, feature)

		// 计算置信度
		score.Confidence = alg.calculateConfidence(score)

		scores[symbol] = score
	}

	return scores
}

// applyRiskFiltering 应用风险过滤
func (alg *CoinSelectionAlgorithm) applyRiskFiltering(scores map[string]*CoinScore, features map[string]*CoinFeatures) []*CoinScore {
	var filtered []*CoinScore

	for _, score := range scores {
		// 最低评分过滤
		if score.TotalScore < alg.config.MinScore {
			log.Printf("[DEBUG] Filtering %s: score %.1f < min_score %.1f", score.Symbol, score.TotalScore, alg.config.MinScore)
			continue
		}

		// 风险容忍度过滤
		riskThreshold := (1 - alg.config.RiskTolerance) * 100
		if score.RiskMetrics.OverallRisk > riskThreshold {
			log.Printf("[DEBUG] Filtering %s: risk %.1f > threshold %.1f", score.Symbol, score.RiskMetrics.OverallRisk, riskThreshold)
			continue
		}

		log.Printf("[DEBUG] Keeping %s: score=%.1f, risk=%.1f", score.Symbol, score.TotalScore, score.RiskMetrics.OverallRisk)
		filtered = append(filtered, score)
	}

	// 按总分排序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].TotalScore > filtered[j].TotalScore
	})

	// 限制候选数量
	if len(filtered) > alg.config.MaxCandidates {
		filtered = filtered[:alg.config.MaxCandidates]
	}

	return filtered
}

// applyStrategySelection 应用策略选择
func (alg *CoinSelectionAlgorithm) applyStrategySelection(scores []*CoinScore, marketState MarketState) []CoinRecommendation {
	var recommendations []CoinRecommendation

	for i, score := range scores {
		// 确定策略类型
		strategyType := alg.determineStrategyType(score, marketState)
		score.StrategyType = strategyType

		recommendation := CoinRecommendation{
			CoinScore:     *score,
			Rank:          i + 1,
			RecommendedAt: time.Now(),
		}

		// 这里应该填充 MarketDataPoint 数据
		// recommendation.Data = ...

		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// NewRiskAssessor 创建风险评估器
func NewRiskAssessor() *RiskAssessor {
	return &RiskAssessor{}
}

// ============================================================================
// 辅助函数
// ============================================================================

// calculateTrendStrength 计算趋势强度
func (alg *CoinSelectionAlgorithm) calculateTrendStrength(tech *TechnicalIndicators) float64 {
	strength := 0.0

	// 均线排列强度
	if tech.MA5 > tech.MA10 && tech.MA10 > tech.MA20 {
		strength += 30 // 多头排列
	} else if tech.MA5 < tech.MA10 && tech.MA10 < tech.MA20 {
		strength -= 30 // 空头排列
	}

	// MACD 强度
	if tech.MACD > tech.MACDSignal {
		strength += 20
	} else {
		strength -= 20
	}

	// RSI 强度
	if tech.RSI > 60 {
		strength += 10
	} else if tech.RSI < 40 {
		strength -= 10
	}

	return math.Max(-100, math.Min(100, strength))
}

// calculateHeatScore 计算热度评分
func (alg *CoinSelectionAlgorithm) calculateHeatScore(features *CoinFeatures) float64 {
	score := 0.0

	// 市值评分
	if features.MarketCap != nil {
		if *features.MarketCap > 10000000000 { // 100亿
			score += 30
		} else if *features.MarketCap > 1000000000 { // 10亿
			score += 20
		} else if *features.MarketCap > 100000000 { // 1亿
			score += 10
		}
	}

	// 成交量评分
	if features.Volume24h > 100000000 { // 1亿成交量
		score += 20
	} else if features.Volume24h > 10000000 { // 1000万
		score += 15
	} else if features.Volume24h > 1000000 { // 100万
		score += 10
	}

	return math.Min(100, score)
}

// determineStrategyType 确定策略类型
func (alg *CoinSelectionAlgorithm) determineStrategyType(score *CoinScore, marketState MarketState) string {
	// 基于评分特征确定最适合的策略

	if !alg.config.Strategies.EnableShort && score.Scores.Momentum < 40 {
		// 如果不允许空头策略，且动量为负，选择震荡策略
		if alg.config.Strategies.EnableRange {
			return "RANGE"
		}
		return "LONG"
	}

	if score.Scores.Momentum > 60 && alg.config.Strategies.EnableMomentum {
		return "MOMENTUM"
	}

	if score.Scores.Technical > 60 {
		if score.Features.RSI < 40 && alg.config.Strategies.EnableLong {
			return "LONG" // 超卖，适合多头
		}
		if score.Features.RSI > 60 && alg.config.Strategies.EnableShort {
			return "SHORT" // 超买，适合空头
		}
	}

	if alg.config.Strategies.EnableRange {
		return "RANGE"
	}

	return "LONG" // 默认多头策略
}

// generateReasons 生成推荐理由
func (alg *CoinSelectionAlgorithm) generateReasons(score *CoinScore, features *CoinFeatures) []string {
	reasons := []string{}

	// 技术指标理由
	if score.Scores.Technical > 70 {
		if features.RSI < 30 {
			reasons = append(reasons, "RSI显示超卖，可能反弹")
		}
		if features.MACD > features.MACDSignal {
			reasons = append(reasons, "MACD金叉，技术面看涨")
		}
		if features.BBPosition < 0.2 {
			reasons = append(reasons, "价格接近布林带下轨，支持位较强")
		}
	}

	// 动量理由
	if score.Scores.Momentum > 60 {
		if features.Momentum24h > 10 {
			reasons = append(reasons, fmt.Sprintf("24h涨幅%.1f%%，上涨动能强劲", features.Momentum24h))
		}
		if features.Acceleration > 5 {
			reasons = append(reasons, "上涨加速，动量正向增强")
		}
	}

	// 基本面理由
	if score.Scores.Fundamental > 60 {
		if features.FlowScore > 15 {
			reasons = append(reasons, "资金流入积极，市场关注度高")
		}
		if features.SentimentScore > 7 {
			reasons = append(reasons, "社交媒体情绪正面")
		}
		if features.AnnouncementScore > 20 {
			reasons = append(reasons, "近期有重要公告")
		}
	}

	// 风险评估理由
	if score.RiskMetrics.OverallRisk < 40 {
		reasons = append(reasons, "综合风险较低，相对安全")
	} else if score.RiskMetrics.OverallRisk > 70 {
		reasons = append(reasons, "风险较高，建议控制仓位")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "综合评分较高，值得关注")
	}

	return reasons
}

// calculateConfidence 计算置信度
func (alg *CoinSelectionAlgorithm) calculateConfidence(score *CoinScore) float64 {
	confidence := 0.5 // 基础置信度

	// 基于评分一致性计算置信度
	scores := []float64{
		score.Scores.Technical,
		score.Scores.Fundamental,
		score.Scores.Sentiment,
		score.Scores.Momentum,
	}

	// 计算标准差（评分一致性）
	mean := (scores[0] + scores[1] + scores[2] + scores[3]) / 4
	variance := 0.0
	for _, s := range scores {
		variance += (s - mean) * (s - mean)
	}
	stdDev := math.Sqrt(variance / 4)

	// 标准差越小，置信度越高
	if stdDev < 10 {
		confidence += 0.3 // 评分很一致
	} else if stdDev < 20 {
		confidence += 0.1 // 评分较一致
	} else {
		confidence -= 0.2 // 评分不一致
	}

	// 基于风险水平调整置信度
	if score.RiskMetrics.OverallRisk < 30 {
		confidence += 0.1 // 低风险，提高置信度
	} else if score.RiskMetrics.OverallRisk > 70 {
		confidence -= 0.2 // 高风险，降低置信度
	}

	return math.Max(0.1, math.Min(0.95, confidence))
}

// ============================================================================
// 市场状态分析器
// ============================================================================

// MarketStateAnalyzer 市场状态分析器
type MarketStateAnalyzer struct{}

// MarketAnalysis 市场分析结果
type MarketAnalysis struct {
	State        string  `json:"state"`         // bull/bear/sideways
	AvgChange    float64 `json:"avg_change"`    // 平均涨幅
	Volatility   float64 `json:"volatility"`    // 波动率
	UpRatio      float64 `json:"up_ratio"`      // 上涨比例
	VolumeChange float64 `json:"volume_change"` // 成交量变化
	Strength     float64 `json:"strength"`      // 市场强度
}

// NewMarketStateAnalyzer 创建市场状态分析器
func NewMarketStateAnalyzer() *MarketStateAnalyzer {
	return &MarketStateAnalyzer{}
}

// AnalyzeMarketState 分析市场状态
func (analyzer *MarketStateAnalyzer) AnalyzeMarketState(marketData []MarketDataPoint) MarketAnalysis {
	if len(marketData) == 0 {
		return MarketAnalysis{State: "sideways"}
	}

	analysis := MarketAnalysis{}

	// 计算平均涨幅
	totalChange := 0.0
	upCount := 0
	changes := make([]float64, len(marketData))

	for i, data := range marketData {
		changes[i] = data.PriceChange24h
		totalChange += data.PriceChange24h
		if data.PriceChange24h > 0 {
			upCount++
		}
	}

	analysis.AvgChange = totalChange / float64(len(marketData))
	analysis.UpRatio = float64(upCount) / float64(len(marketData))

	// 计算波动率（标准差）
	if len(changes) > 1 {
		var variance float64
		for _, change := range changes {
			variance += (change - analysis.AvgChange) * (change - analysis.AvgChange)
		}
		analysis.Volatility = math.Sqrt(variance / float64(len(changes)))
	}

	// 判断市场状态
	if analysis.AvgChange > 3 && analysis.UpRatio > 0.6 {
		analysis.State = "bull"
		analysis.Strength = (analysis.AvgChange * 0.7) + (analysis.UpRatio * 100 * 0.3)
	} else if analysis.AvgChange < -2 && analysis.UpRatio < 0.4 {
		analysis.State = "bear"
		analysis.Strength = math.Abs(analysis.AvgChange*0.7) + ((1 - analysis.UpRatio) * 100 * 0.3)
	} else {
		analysis.State = "sideways"
		analysis.Strength = 50 - math.Abs(analysis.AvgChange)
	}

	return analysis
}

// ============================================================================
// 风险评估器
// ============================================================================

// RiskAssessor 风险评估器

// ============================================================================
// 策略选择器
// ============================================================================

// StrategySelector 策略选择器
type StrategySelector struct{}

// NewStrategySelector 创建策略选择器
func NewStrategySelector() *StrategySelector {
	return &StrategySelector{}
}

// ============================================================================
// ⭐ 新特征工程集成方法
// ============================================================================

// extractTimeSeriesFeatures 提取时间序列特征 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractTimeSeriesFeatures(ctx context.Context, data MarketDataPoint) (map[string]float64, error) {
	if alg.featureEngineering == nil {
		return nil, fmt.Errorf("feature engineering not initialized")
	}

	featureSet, err := alg.featureEngineering.ExtractFeatures(ctx, data.Symbol)
	if err != nil {
		return nil, err
	}

	// 返回时间序列相关的特征
	timeSeriesFeatures := make(map[string]float64)
	for name, value := range featureSet.Features {
		if isTimeSeriesFeature(name) {
			timeSeriesFeatures[name] = value
		}
	}

	return timeSeriesFeatures, nil
}

// extractVolatilityFeatures 提取波动率特征 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractVolatilityFeatures(ctx context.Context, data MarketDataPoint) (map[string]float64, error) {
	if alg.featureEngineering == nil {
		return nil, fmt.Errorf("feature engineering not initialized")
	}

	featureSet, err := alg.featureEngineering.ExtractFeatures(ctx, data.Symbol)
	if err != nil {
		return nil, err
	}

	// 返回波动率相关的特征
	volatilityFeatures := make(map[string]float64)
	for name, value := range featureSet.Features {
		if isVolatilityFeature(name) {
			volatilityFeatures[name] = value
		}
	}

	return volatilityFeatures, nil
}

// extractTrendFeatures 提取趋势特征 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractTrendFeatures(ctx context.Context, data MarketDataPoint) (map[string]float64, error) {
	if alg.featureEngineering == nil {
		return nil, fmt.Errorf("feature engineering not initialized")
	}

	featureSet, err := alg.featureEngineering.ExtractFeatures(ctx, data.Symbol)
	if err != nil {
		return nil, err
	}

	// 返回趋势相关的特征
	trendFeatures := make(map[string]float64)
	for name, value := range featureSet.Features {
		if isTrendFeature(name) {
			trendFeatures[name] = value
		}
	}

	return trendFeatures, nil
}

// extractMomentumFeaturesAdvanced 提取高级动量特征 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractMomentumFeaturesAdvanced(ctx context.Context, data MarketDataPoint) (map[string]float64, error) {
	if alg.featureEngineering == nil {
		return nil, fmt.Errorf("feature engineering not initialized")
	}

	featureSet, err := alg.featureEngineering.ExtractFeatures(ctx, data.Symbol)
	if err != nil {
		return nil, err
	}

	// 返回动量相关的特征
	momentumFeatures := make(map[string]float64)
	for name, value := range featureSet.Features {
		if isMomentumFeature(name) {
			momentumFeatures[name] = value
		}
	}

	return momentumFeatures, nil
}

// extractCrossFeatures 提取交叉特征 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractCrossFeatures(ctx context.Context, data MarketDataPoint) (map[string]float64, error) {
	if alg.featureEngineering == nil {
		return nil, fmt.Errorf("feature engineering not initialized")
	}

	featureSet, err := alg.featureEngineering.ExtractFeatures(ctx, data.Symbol)
	if err != nil {
		return nil, err
	}

	// 返回交叉相关的特征
	crossFeatures := make(map[string]float64)
	for name, value := range featureSet.Features {
		if isCrossFeature(name) {
			crossFeatures[name] = value
		}
	}

	return crossFeatures, nil
}

// extractStatisticalFeatures 提取统计特征 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractStatisticalFeatures(ctx context.Context, data MarketDataPoint) (map[string]float64, error) {
	if alg.featureEngineering == nil {
		return nil, fmt.Errorf("feature engineering not initialized")
	}

	featureSet, err := alg.featureEngineering.ExtractFeatures(ctx, data.Symbol)
	if err != nil {
		return nil, err
	}

	// 返回统计相关的特征
	statisticalFeatures := make(map[string]float64)
	for name, value := range featureSet.Features {
		if isStatisticalFeature(name) {
			statisticalFeatures[name] = value
		}
	}

	return statisticalFeatures, nil
}

// extractLegacyFeatures 回退到原有特征提取方法 ⭐ 新增
func (alg *CoinSelectionAlgorithm) extractLegacyFeatures(feature *CoinFeatures, data MarketDataPoint, marketState MarketState) {
	// 这里可以保留原有的特征提取逻辑作为回退方案
	// 暂时留空，具体实现根据需要添加
}

// 特征分类辅助函数 ⭐ 新增

func isTimeSeriesFeature(name string) bool {
	timeSeriesPrefixes := []string{
		"price_momentum_", "volume_roc_", "volume_ratio_",
		"hour_of_day", "day_of_week", "month_of_year",
		"price_vs_day_ago_same_hour", "price_vs_hourly_average",
	}

	for _, prefix := range timeSeriesPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isVolatilityFeature(name string) bool {
	volatilityPrefixes := []string{
		"volatility_", "price_std", "price_cv", "price_z_score",
		"value_at_risk_", "expected_shortfall_", "max_drawdown_risk",
	}

	for _, prefix := range volatilityPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isTrendFeature(name string) bool {
	trendPrefixes := []string{
		"trend_", "ma_", "price_position_in_range",
		"price_near_", "price_at_mid", "price_extreme_",
		"golden_cross_", "death_cross_", "support_breakout_",
		"resistance_breakout_", "double_top_", "double_bottom_",
		"head_shoulders_", "pattern_strength",
	}

	for _, prefix := range trendPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isMomentumFeature(name string) bool {
	momentumPrefixes := []string{
		"momentum_", "price_roc_", "volume_rsi",
		"volume_momentum_", "momentum_acceleration",
		"momentum_reversal_", "momentum_decay",
		"momentum_support_", "momentum_resistance_",
	}

	for _, prefix := range momentumPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isCrossFeature(name string) bool {
	crossPrefixes := []string{
		"rsi_macd_", "rsi_bb_", "ma_alignment", "ma_slope_diff",
		"price_volume_", "volume_price_", "momentum_volume_",
		"trend_momentum_", "momentum_trend_", "acceleration_",
		"feature_", "effective_trend", "trend_divergence",
		"multi_timeframe_", "momentum_time_",
	}

	for _, prefix := range crossPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isStatisticalFeature(name string) bool {
	statisticalPrefixes := []string{
		"price_mean", "price_median", "price_q", "price_iqr",
		"price_skewness", "price_kurtosis", "jarque_bera_",
		"normality_", "distribution_", "autocorrelation_",
		"series_", "chaos_", "anomaly_", "outlier_",
	}

	for _, prefix := range statisticalPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// ============================================================================
// ⭐ 机器学习增强方法
// ============================================================================

// applyMachineLearningEnhancement 应用机器学习增强 ⭐ 新增
func (alg *CoinSelectionAlgorithm) applyMachineLearningEnhancement(scores map[string]*CoinScore, marketData []MarketDataPoint) map[string]*CoinScore {
	if alg.machineLearning == nil {
		return scores
	}

	// 为每个币种生成机器学习预测
	enhancedScores := make(map[string]*CoinScore)

	for symbol, score := range scores {
		// 使用机器学习进行预测
		prediction, err := alg.machineLearning.PredictWithEnsemble(context.Background(), symbol, "random_forest")
		if err != nil {
			log.Printf("[ML Enhancement] 预测失败 %s: %v", symbol, err)
			enhancedScores[symbol] = score
			continue
		}

		// 结合传统评分和机器学习预测
		mlWeight := 0.3          // 机器学习权重
		traditionalWeight := 0.7 // 传统评分权重

		enhancedScore := &CoinScore{
			Symbol:     score.Symbol,
			TotalScore: score.TotalScore*traditionalWeight + prediction.Score*mlWeight,
			Scores: struct {
				Technical   float64
				Fundamental float64
				Sentiment   float64
				Risk        float64
				Momentum    float64
			}{
				Technical:   score.Scores.Technical,
				Fundamental: score.Scores.Fundamental,
				Sentiment:   score.Scores.Sentiment,
				Risk:        score.Scores.Risk,
				Momentum:    score.Scores.Momentum,
			},
		}

		// 重新计算各维度权重
		totalWeight := traditionalWeight + mlWeight
		enhancedScore.TotalScore /= totalWeight

		enhancedScores[symbol] = enhancedScore

		log.Printf("[ML Enhancement] %s 原始评分: %.4f, ML预测: %.4f, 增强后: %.4f",
			symbol, score.TotalScore, prediction.Score, enhancedScore.TotalScore)
	}

	return enhancedScores
}

// applyRiskControl 应用风险控制决策 ⭐ 新增
func (alg *CoinSelectionAlgorithm) applyRiskControl(scores map[string]*CoinScore) map[string]*CoinScore {
	if alg.riskManagement == nil {
		return scores
	}

	filteredScores := make(map[string]*CoinScore)

	for symbol, score := range scores {
		// 获取风险决策
		decision, err := alg.riskManagement.MakeRiskDecision(context.Background(), symbol, 0.1) // 假设10%仓位
		if err != nil {
			log.Printf("[Risk Control] 获取风险决策失败 %s: %v", symbol, err)
			filteredScores[symbol] = score
			continue
		}

		// 如果风险过高，降低评分或过滤掉
		if !decision.CanTrade {
			log.Printf("[Risk Control] 过滤高风险资产: %s (风险等级: %s)", symbol, decision.RiskLevel)
			continue // 过滤掉这个资产
		}

		// 根据风险等级调整评分
		riskAdjustment := 1.0
		switch decision.RiskLevel {
		case "high":
			riskAdjustment = 0.7 // 高风险资产评分降低30%
		case "critical":
			riskAdjustment = 0.5 // 关键风险资产评分降低50%
		case "medium":
			riskAdjustment = 0.9 // 中等风险资产评分降低10%
		case "low":
			riskAdjustment = 1.0 // 低风险资产保持原评分
		}

		// 应用风险调整
		adjustedScore := &CoinScore{
			Symbol:     score.Symbol,
			TotalScore: score.TotalScore * riskAdjustment,
			Scores: struct {
				Technical   float64
				Fundamental float64
				Sentiment   float64
				Risk        float64
				Momentum    float64
			}{
				Technical:   score.Scores.Technical * riskAdjustment,
				Fundamental: score.Scores.Fundamental * riskAdjustment,
				Sentiment:   score.Scores.Sentiment * riskAdjustment,
				Risk:        score.Scores.Risk,
				Momentum:    score.Scores.Momentum * riskAdjustment,
			},
		}

		filteredScores[symbol] = adjustedScore

		log.Printf("[Risk Control] %s 风险等级: %s, 评分调整: %.2f -> %.2f",
			symbol, decision.RiskLevel, score.TotalScore, adjustedScore.TotalScore)
	}

	return filteredScores
}

// ============================================================================
// 工具函数
// ============================================================================
