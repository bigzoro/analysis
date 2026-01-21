package traditional

import (
	"context"

	pdb "analysis/internal/db"
)

// ============================================================================
// 类型定义和接口
// ============================================================================

// TraditionalConfig 传统策略配置
type TraditionalConfig struct {
	Enabled            bool `json:"enabled"`
	ShortOnGainers     bool `json:"short_on_gainers"`      // 涨幅榜开空
	LongOnSmallGainers bool `json:"long_on_small_gainers"` // 小幅上涨开多

	// 排名限制
	GainersRankLimit     int `json:"gainers_rank_limit"`      // 涨幅榜排名限制
	GainersRankLimitLong int `json:"gainers_rank_limit_long"` // 开多涨幅榜排名限制

	// 价格过滤
	MinPriceThreshold float64 `json:"min_price_threshold"` // 最低价格阈值
	MaxPriceThreshold float64 `json:"max_price_threshold"` // 最高价格阈值

	// 交易量过滤
	MinVolumeThreshold float64 `json:"min_volume_threshold"` // 最低交易量阈值

	// 涨跌幅过滤
	MaxChangePercent float64 `json:"max_change_percent"` // 最大涨跌幅百分比
	MinChangePercent float64 `json:"min_change_percent"` // 最小涨跌幅百分比

	// 杠杆配置
	EnableLeverage  bool `json:"enable_leverage"`  // 是否启用杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
	MaxLeverage     int  `json:"max_leverage"`     // 最大杠杆倍数

	// 资金费率过滤
	FundingRateFilterEnabled bool    `json:"funding_rate_filter_enabled"` // 资金费率过滤启用
	MinFundingRate           float64 `json:"min_funding_rate"`            // 最低资金费率要求（%）

	// 合约涨幅排名过滤
	FuturesPriceRankFilterEnabled bool `json:"futures_price_rank_filter_enabled"` // 合约涨幅排名过滤启用
	MaxFuturesPriceRank           int  `json:"max_futures_price_rank"`            // 合约涨幅最大排名（前N名）

	// 新增：合约涨幅开空策略
	FuturesPriceShortStrategyEnabled bool    `json:"futures_price_short_strategy_enabled"` // 合约涨幅开空策略启用
	FuturesPriceShortMaxRank         int     `json:"futures_price_short_max_rank"`         // 合约涨幅最大排名
	FuturesPriceShortMinFundingRate  float64 `json:"futures_price_short_min_funding_rate"` // 最低资金费率要求
	FuturesPriceShortLeverage        float64 `json:"futures_price_short_leverage"`         // 开空倍数
	FuturesPriceShortMinMarketCap    float64 `json:"futures_price_short_min_market_cap"`   // 最低市值要求（万）

	// 交易配置
	MarginMode string `json:"margin_mode"` // 保证金模式: "ISOLATED" 或 "CROSS"

	// 候选数量限制
	MaxCandidates int `json:"max_candidates"` // 最大候选数量

	// 持仓过滤
	SkipHeldPositions bool `json:"skip_held_positions"` // 是否跳过已有持仓的币种

	// 平仓订单过滤
	SkipCloseOrdersWithin24Hours bool `json:"skip_close_orders_within_24_hours"` // 是否跳过24小时内的平仓订单（已废弃）
	SkipCloseOrdersHours         int  `json:"skip_close_orders_hours"`           // 跳过平仓记录的小时数（0表示不跳过）

	// 币种黑名单
	UseSymbolBlacklist bool     `json:"use_symbol_blacklist"` // 是否启用币种黑名单
	SymbolBlacklist    []string `json:"symbol_blacklist"`     // 黑名单币种列表

	// 盈利加仓策略
	ProfitScalingEnabled bool    `json:"profit_scaling_enabled"` // 是否启用盈利加仓
	ProfitScalingPercent float64 `json:"profit_scaling_percent"` // 触发加仓的盈利百分比
	ProfitScalingAmount  float64 `json:"profit_scaling_amount"`  // 加仓金额（U单位）
}

// CandidateWithRank 带有排名的候选币种
type CandidateWithRank struct {
	Symbol        string  `json:"symbol"`
	Rank          int     `json:"rank"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"change_percent"`
	Volume        float64 `json:"volume"`
	MarketCap     float64 `json:"market_cap"`
}

// SelectionResult 选择结果
type SelectionResult struct {
	Candidates []CandidateWithRank `json:"candidates"`
	Strategy   string              `json:"strategy"` // "short_gainers" 或 "long_small_gainers"
	Reason     string              `json:"reason"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Symbol  string  `json:"symbol"`
	IsValid bool    `json:"is_valid"`
	Action  string  `json:"action"` // "short" 或 "long"
	Reason  string  `json:"reason"`
	Score   float64 `json:"score"` // 0.0-1.0
}

// ============================================================================
// 核心接口定义
// ============================================================================

// TraditionalStrategy 传统策略核心接口
type TraditionalStrategy interface {
	// 核心功能
	Scan(ctx context.Context, config *TraditionalConfig) ([]ValidationResult, error)
	IsEnabled(config *TraditionalConfig) bool

	// 策略执行
	ExecuteShortOnGainers(ctx context.Context, config *TraditionalConfig) ([]ValidationResult, error)
	ExecuteLongOnSmallGainers(ctx context.Context, config *TraditionalConfig) ([]ValidationResult, error)

	// 适配器方法
	ToStrategyScanner() interface{} // 返回StrategyScanner接口，由调用方处理类型转换
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	ConvertConfig(conditions pdb.StrategyConditions) *TraditionalConfig
	ValidateConfig(config *TraditionalConfig) error
	DefaultConfig() *TraditionalConfig
}

// CandidateSelector 候选选择器接口
type CandidateSelector interface {
	SelectGainersWithRank(ctx context.Context, limit int, marketType string) ([]CandidateWithRank, error)
	SelectSmallGainersWithRank(ctx context.Context, limit int) ([]CandidateWithRank, error)
	SelectByVolume(ctx context.Context, limit int) ([]CandidateWithRank, error)
}

// RankCalculator 排名计算器接口
type RankCalculator interface {
	CalculateGainersRank(candidates []CandidateWithRank) []CandidateWithRank
	CalculateVolumeRank(candidates []CandidateWithRank) []CandidateWithRank
	FilterByRank(candidates []CandidateWithRank, maxRank int) []CandidateWithRank
}

// PriceValidator 价格验证器接口
type PriceValidator interface {
	ValidatePriceRange(price float64, config *TraditionalConfig) bool
	ValidateVolume(volume float64, config *TraditionalConfig) bool
	ValidateChangePercent(changePercent float64, config *TraditionalConfig) bool
}

// StrategyValidator 策略验证器接口
type StrategyValidator interface {
	ValidateForShort(candidate *CandidateWithRank, config *TraditionalConfig) *ValidationResult
	ValidateForLong(candidate *CandidateWithRank, config *TraditionalConfig) *ValidationResult
	ValidateForFuturesPriceShort(candidate *CandidateWithRank, config *TraditionalConfig) *ValidationResult
	CalculateSuitabilityScore(candidate *CandidateWithRank, config *TraditionalConfig) float64
}
