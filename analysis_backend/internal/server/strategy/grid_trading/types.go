package grid_trading

import (
	"context"

	pdb "analysis/internal/db"
)

// ============================================================================
// 类型定义和接口
// ============================================================================

// GridTradingConfig 网格交易策略配置
type GridTradingConfig struct {
	Enabled bool `json:"enabled"`

	// 网格参数
	GridLevels         int     `json:"grid_levels"`          // 网格层数
	GridSpacingPercent float64 `json:"grid_spacing_percent"` // 网格间距百分比
	MinGridRange       float64 `json:"min_grid_range"`       // 最小网格范围
	MaxGridRange       float64 `json:"max_grid_range"`       // 最大网格范围

	// 评分阈值
	MinVolatilityScore    float64 `json:"min_volatility_score"`    // 最小波动率评分
	MinLiquidityScore     float64 `json:"min_liquidity_score"`     // 最小流动性评分
	MinStabilityScore     float64 `json:"min_stability_score"`     // 最小稳定性评分
	OverallScoreThreshold float64 `json:"overall_score_threshold"` // 整体评分阈值

	// 杠杆配置
	EnableLeverage  bool `json:"enable_leverage"`  // 是否启用杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
	MaxLeverage     int  `json:"max_leverage"`     // 最大杠杆倍数

	// 交易配置
	MarginMode string `json:"margin_mode"` // 保证金模式: "ISOLATED" 或 "CROSS"

	// 候选选择
	MaxCandidates int  `json:"max_candidates"` // 最大候选数量
	UseMarketCap  bool `json:"use_market_cap"` // 使用市值选择
	UseVolume     bool `json:"use_volume"`     // 使用交易量选择
}

// ScoringResult 评分结果
type ScoringResult struct {
	VolatilityScore float64 `json:"volatility_score"` // 波动率评分 (0.0-1.0)
	LiquidityScore  float64 `json:"liquidity_score"`  // 流动性评分 (0.0-1.0)
	StabilityScore  float64 `json:"stability_score"`  // 稳定性评分 (0.0-1.0)
	OverallScore    float64 `json:"overall_score"`    // 整体评分 (0.0-1.0)
}

// GridRange 网格范围
type GridRange struct {
	Lower float64 `json:"lower"` // 下限价格
	Upper float64 `json:"upper"` // 上限价格
}

// CandidateResult 候选结果
type CandidateResult struct {
	Symbol     string        `json:"symbol"`
	Score      ScoringResult `json:"score"`
	GridRange  GridRange     `json:"grid_range"`
	MarketCap  float64       `json:"market_cap"`
	Reason     string        `json:"reason"`
	IsEligible bool          `json:"is_eligible"`
}

// ============================================================================
// 核心接口定义
// ============================================================================

// GridTradingStrategy 网格交易策略核心接口
type GridTradingStrategy interface {
	// 核心功能
	Scan(ctx context.Context, config *GridTradingConfig) ([]CandidateResult, error)
	IsEnabled(config *GridTradingConfig) bool

	// 网格计算
	CalculateGridRange(currentPrice float64, config *GridTradingConfig) GridRange
	ValidateGridParameters(config *GridTradingConfig) error

	// 适配器方法
	ToStrategyScanner() interface{} // 返回StrategyScanner接口，由调用方处理类型转换
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	ConvertConfig(conditions pdb.StrategyConditions) *GridTradingConfig
	ValidateConfig(config *GridTradingConfig) error
	DefaultConfig() *GridTradingConfig
}

// ScoringEngine 评分引擎接口
type ScoringEngine interface {
	CalculateVolatilityScore(ctx context.Context, symbol string) (float64, error)
	CalculateLiquidityScore(ctx context.Context, symbol string) (float64, error)
	CalculateStabilityScore(ctx context.Context, symbol string) (float64, error)
	CalculateOverallScore(volatility, liquidity, stability float64) float64
}

// CandidateSelector 候选选择器接口
type CandidateSelector interface {
	SelectByMarketCap(ctx context.Context, maxCount int) ([]string, error)
	SelectByVolume(ctx context.Context, maxCount int) ([]string, error)
	FallbackToDefaults(maxCount int) ([]string, error)
}

// GridCalculator 网格计算器接口
type GridCalculator interface {
	CalculateDynamicRange(currentPrice float64, config *GridTradingConfig) GridRange
	ValidateRange(gridRange GridRange, config *GridTradingConfig) bool
}

// Validator 验证器接口
type Validator interface {
	ValidateCandidate(ctx context.Context, symbol string, config *GridTradingConfig) (*CandidateResult, error)
	ValidateScoringResult(result *ScoringResult, config *GridTradingConfig) bool
}
