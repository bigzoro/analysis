package arbitrage

import (
	"context"

	pdb "analysis/internal/db"
)

// ============================================================================
// 类型定义和接口
// ============================================================================

// ArbitrageConfig 套利策略配置
type ArbitrageConfig struct {
	Enabled bool `json:"enabled"`

	// 套利类型启用
	CrossExchangeArbEnabled bool `json:"cross_exchange_arb_enabled"` // 跨交易所套利
	SpotFutureArbEnabled    bool `json:"spot_future_arb_enabled"`    // 现货期货套利
	TriangleArbEnabled      bool `json:"triangle_arb_enabled"`       // 三角套利
	StatArbEnabled          bool `json:"stat_arb_enabled"`           // 统计套利
	FuturesSpotArbEnabled   bool `json:"futures_spot_arb_enabled"`   // 期货现货套利

	// 套利参数
	MinProfitThreshold  float64 `json:"min_profit_threshold"`   // 最小利润阈值
	MaxSlippagePercent  float64 `json:"max_slippage_percent"`   // 最大滑点百分比
	MinVolumeThreshold  float64 `json:"min_volume_threshold"`   // 最小交易量阈值
	ScanIntervalSeconds int     `json:"scan_interval_seconds"`  // 扫描间隔秒数

	// 三角套利参数
	TriangleMinProfitPercent float64 `json:"triangle_min_profit_percent"` // 三角套利最小利润百分比
	TriangleMaxPathLength    int     `json:"triangle_max_path_length"`    // 三角套利最大路径长度

	// 跨交易所套利参数
	ExchangePairs []string `json:"exchange_pairs"` // 交易所对列表

	// 统计套利参数
	StatArbLookbackPeriod int     `json:"stat_arb_lookback_period"` // 回望周期
	StatArbStdDevThreshold float64 `json:"stat_arb_std_dev_threshold"` // 标准差阈值

	// 杠杆配置
	EnableLeverage  bool `json:"enable_leverage"`  // 是否启用杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
	MaxLeverage     int  `json:"max_leverage"`     // 最大杠杆倍数

	// 交易配置
	MarginMode string `json:"margin_mode"` // 保证金模式: "ISOLATED" 或 "CROSS"
}

// ArbitrageOpportunity 套利机会
type ArbitrageOpportunity struct {
	Type         string  `json:"type"`          // "triangle", "cross_exchange", "spot_future", "statistical"
	Symbol       string  `json:"symbol"`        // 交易对
	ProfitPercent float64 `json:"profit_percent"` // 预期利润百分比
	ProfitAmount float64 `json:"profit_amount"` // 预期利润金额
	Path         []string `json:"path,omitempty"` // 三角套利路径
	ExchangeA    string  `json:"exchange_a,omitempty"` // 交易所A
	ExchangeB    string  `json:"exchange_b,omitempty"` // 交易所B
	PriceA       float64 `json:"price_a,omitempty"` // 交易所A价格
	PriceB       float64 `json:"price_b,omitempty"` // 交易所B价格
	Volume       float64 `json:"volume"`       // 交易量
	Confidence   float64 `json:"confidence"`   // 置信度
	Timestamp    int64   `json:"timestamp"`    // 时间戳
	Reason       string  `json:"reason"`       // 套利理由
}

// ValidationResult 验证结果
type ValidationResult struct {
	Opportunity  *ArbitrageOpportunity `json:"opportunity"`
	IsValid      bool                  `json:"is_valid"`
	Action       string                `json:"action"` // "arbitrage"
	Reason       string                `json:"reason"`
	RiskLevel    string                `json:"risk_level"` // "low", "medium", "high"
	Score        float64               `json:"score"`  // 0.0-1.0
}

// TrianglePath 三角套利路径
type TrianglePath struct {
	Path         []string  `json:"path"`          // 路径，如 ["BTCUSDT", "ETHBTC", "ETHUSDT"]
	ProfitPercent float64  `json:"profit_percent"` // 利润百分比
	StartAmount  float64   `json:"start_amount"`  // 起始金额
	EndAmount    float64   `json:"end_amount"`   // 结束金额
	Volumes      []float64 `json:"volumes"`      // 各段交易量
}

// PriceData 价格数据
type PriceData struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Exchange  string  `json:"exchange,omitempty"`
	Timestamp int64   `json:"timestamp"`
}

// ============================================================================
// 核心接口定义
// ============================================================================

// ArbitrageStrategy 套利策略核心接口
type ArbitrageStrategy interface {
	// 核心功能
	Scan(ctx context.Context, config *ArbitrageConfig) ([]ValidationResult, error)
	IsEnabled(config *ArbitrageConfig) bool

	// 套利类型
	ScanTriangleArbitrage(ctx context.Context, config *ArbitrageConfig) ([]ValidationResult, error)
	ScanCrossExchangeArbitrage(ctx context.Context, config *ArbitrageConfig) ([]ValidationResult, error)
	ScanSpotFutureArbitrage(ctx context.Context, config *ArbitrageConfig) ([]ValidationResult, error)
	ScanStatisticalArbitrage(ctx context.Context, config *ArbitrageConfig) ([]ValidationResult, error)

	// 适配器方法
	ToStrategyScanner() interface{} // 返回StrategyScanner接口，由调用方处理类型转换
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	ConvertConfig(conditions pdb.StrategyConditions) *ArbitrageConfig
	ValidateConfig(config *ArbitrageConfig) error
	DefaultConfig() *ArbitrageConfig
}

// TriangleArbitrageScanner 三角套利扫描器接口
type TriangleArbitrageScanner interface {
	FindArbitragePaths(ctx context.Context, baseSymbols []string) ([]TrianglePath, error)
	ValidateTrianglePath(ctx context.Context, path TrianglePath, config *ArbitrageConfig) (*ValidationResult, error)
	CalculateTriangleProfit(path []string, amounts []float64) (float64, error)
}

// CrossExchangeScanner 跨交易所扫描器接口
type CrossExchangeScanner interface {
	CompareExchangePrices(ctx context.Context, symbol string, exchanges []string) ([]ArbitrageOpportunity, error)
	GetExchangePrice(ctx context.Context, symbol, exchange string) (*PriceData, error)
	CalculateCrossExchangeSpread(priceA, priceB float64) float64
}

// SpotFutureScanner 现货期货扫描器接口
type SpotFutureScanner interface {
	CompareSpotFuturePrices(ctx context.Context, symbol string) ([]ArbitrageOpportunity, error)
	GetSpotPrice(ctx context.Context, symbol string) (*PriceData, error)
	GetFuturePrice(ctx context.Context, symbol string) (*PriceData, error)
	CalculateBasisSpread(spotPrice, futurePrice float64) float64
}

// StatisticalScanner 统计套利扫描器接口
type StatisticalScanner interface {
	FindStatArbOpportunities(ctx context.Context, symbols []string, config *ArbitrageConfig) ([]ArbitrageOpportunity, error)
	CalculateCointegration(symbolA, symbolB string, pricesA, pricesB []float64) (float64, error)
	DetectMeanReversionSignal(spread, mean, stdDev float64) bool
}

// RiskValidator 风险验证器接口
type RiskValidator interface {
	ValidateArbitrageRisk(ctx context.Context, opportunity *ArbitrageOpportunity) (*ValidationResult, error)
	AssessLiquidityRisk(volume float64, config *ArbitrageConfig) string
	AssessSlippageRisk(expectedProfit, maxSlippage float64) string
	CalculateOverallRiskScore(opportunity *ArbitrageOpportunity) float64
}

// DataProvider 数据提供者接口
type DataProvider interface {
	GetPriceData(ctx context.Context, symbol string) (*PriceData, error)
	GetMultiExchangePrices(ctx context.Context, symbol string, exchanges []string) (map[string]*PriceData, error)
	GetTrianglePathPrices(ctx context.Context, path []string) (map[string]*PriceData, error)
	GetHistoricalPrices(ctx context.Context, symbol string, limit int) ([]PriceData, error)
}