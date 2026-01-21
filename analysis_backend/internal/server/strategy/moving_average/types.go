package moving_average

import (
	"context"

	pdb "analysis/internal/db"
)

// ============================================================================
// 类型定义和接口
// ============================================================================

// MovingAverageConfig 均线策略配置
type MovingAverageConfig struct {
	Enabled bool `json:"enabled"`

	// 均线参数
	ShortPeriod    int     `json:"short_period"`     // 短期均线周期
	LongPeriod     int     `json:"long_period"`      // 长期均线周期
	SignalPeriod   int     `json:"signal_period"`    // 信号均线周期（可选）

	// 交叉类型
	UseGoldenCross  bool    `json:"use_golden_cross"`  // 金叉（短期上穿长期）
	UseDeathCross   bool    `json:"use_death_cross"`   // 死叉（短期下穿长期）
	UseTrendFilter  bool    `json:"use_trend_filter"`  // 趋势过滤

	// 价格过滤
	MinPriceThreshold     float64 `json:"min_price_threshold"`     // 最低价格阈值
	MaxPriceThreshold     float64 `json:"max_price_threshold"`     // 最高价格阈值

	// 交易量过滤
	MinVolumeThreshold    float64 `json:"min_volume_threshold"`    // 最低交易量阈值

	// 信号确认
	RequireVolumeConfirmation bool     `json:"require_volume_confirmation"` // 需要交易量确认
	MinCrossStrength          float64  `json:"min_cross_strength"`          // 最小交叉强度
	ConfirmationPeriod        int      `json:"confirmation_period"`         // 确认周期

	// 杠杆配置
	EnableLeverage  bool `json:"enable_leverage"`  // 是否启用杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
	MaxLeverage     int  `json:"max_leverage"`     // 最大杠杆倍数

	// 交易配置
	MarginMode string `json:"margin_mode"` // 保证金模式: "ISOLATED" 或 "CROSS"

	// 候选选择
	MaxCandidates         int `json:"max_candidates"` // 最大候选数量
	UseVolumeBasedSelection bool `json:"use_volume_based_selection"` // 使用交易量选择
}

// IndicatorData 技术指标数据
type IndicatorData struct {
	Symbol       string    `json:"symbol"`
	ShortMA      []float64 `json:"short_ma"`       // 短期均线
	LongMA       []float64 `json:"long_ma"`        // 长期均线
	SignalMA     []float64 `json:"signal_ma"`      // 信号均线（可选）
	Prices       []float64 `json:"prices"`         // 价格数据
	Volumes      []float64 `json:"volumes"`        // 交易量数据
	Timestamp    int64     `json:"timestamp"`      // 时间戳
}

// CrossSignal 交叉信号
type CrossSignal struct {
	Symbol          string  `json:"symbol"`
	SignalType      string  `json:"signal_type"`       // "golden_cross", "death_cross"
	CrossPrice      float64 `json:"cross_price"`       // 交叉价格
	CrossStrength   float64 `json:"cross_strength"`    // 交叉强度
	VolumeConfirmed bool    `json:"volume_confirmed"`  // 交易量确认
	Timestamp       int64   `json:"timestamp"`         // 信号时间
	Confidence      float64 `json:"confidence"`        // 信号置信度
}

// ValidationResult 验证结果
type ValidationResult struct {
	Symbol      string       `json:"symbol"`
	Signal      *CrossSignal `json:"signal"`
	IsValid     bool         `json:"is_valid"`
	Action      string       `json:"action"` // "buy", "sell"
	Reason      string       `json:"reason"`
	Score       float64      `json:"score"`  // 0.0-1.0
	MarketCap   float64      `json:"market_cap"`
}

// ============================================================================
// 核心接口定义
// ============================================================================

// MovingAverageStrategy 均线策略核心接口
type MovingAverageStrategy interface {
	// 核心功能
	Scan(ctx context.Context, config *MovingAverageConfig) ([]ValidationResult, error)
	IsEnabled(config *MovingAverageConfig) bool

	// 信号生成
	DetectCrossSignals(ctx context.Context, symbol string, config *MovingAverageConfig) ([]CrossSignal, error)
	ValidateSignals(ctx context.Context, signals []CrossSignal, config *MovingAverageConfig) ([]ValidationResult, error)

	// 适配器方法
	ToStrategyScanner() interface{} // 返回StrategyScanner接口，由调用方处理类型转换
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	ConvertConfig(conditions pdb.StrategyConditions) *MovingAverageConfig
	ValidateConfig(config *MovingAverageConfig) error
	DefaultConfig() *MovingAverageConfig
}

// IndicatorCalculator 技术指标计算器接口
type IndicatorCalculator interface {
	CalculateSMA(prices []float64, period int) []float64
	CalculateEMA(prices []float64, period int) []float64
	DetectCross(shortMA, longMA []float64) (crosses []CrossInfo)
	CalculateCrossStrength(shortMA, longMA []float64, crossIndex int) float64
}

// SignalProcessor 信号处理器接口
type SignalProcessor interface {
	ProcessGoldenCross(signal *CrossSignal, config *MovingAverageConfig) *ValidationResult
	ProcessDeathCross(signal *CrossSignal, config *MovingAverageConfig) *ValidationResult
	ConfirmWithVolume(signal *CrossSignal, volumes []float64, config *MovingAverageConfig) bool
	CalculateSignalConfidence(signal *CrossSignal, config *MovingAverageConfig) float64
}

// CandidateSelector 候选选择器接口
type CandidateSelector interface {
	SelectByVolume(ctx context.Context, maxCount int) ([]string, error)
	SelectByMarketCap(ctx context.Context, maxCount int) ([]string, error)
	FallbackToDefaults(maxCount int) ([]string, error)
}

// DataProvider 数据提供者接口
type DataProvider interface {
	GetPriceData(ctx context.Context, symbol string, limit int) ([]float64, error)
	GetVolumeData(ctx context.Context, symbol string, limit int) ([]float64, error)
	GetLatestPrice(ctx context.Context, symbol string) (float64, error)
}

// Validator 验证器接口
type Validator interface {
	ValidatePriceRange(price float64, config *MovingAverageConfig) bool
	ValidateVolume(volume float64, config *MovingAverageConfig) bool
	ValidateCrossStrength(strength float64, config *MovingAverageConfig) bool
	CalculateOverallScore(result *ValidationResult, config *MovingAverageConfig) float64
}

// CrossInfo 交叉信息
type CrossInfo struct {
	Index   int     `json:"index"`    // 交叉点索引
	Type    string  `json:"type"`     // "golden" 或 "death"
	Price   float64 `json:"price"`    // 交叉价格
	Strength float64 `json:"strength"` // 交叉强度
}