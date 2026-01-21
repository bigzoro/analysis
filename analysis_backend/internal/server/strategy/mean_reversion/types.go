// Package mean_reversion 均值回归策略模块
// 提供统一的类型定义和接口
package mean_reversion

import (
	"context"
	"time"

	pdb "analysis/internal/db"
)

// ============================================================================
// 基础类型定义
// ============================================================================

// MRSignalMode 均值回归信号模式枚举
type MRSignalMode string

const (
	MRConservativeMode MRSignalMode = "conservative" // 保守模式：高质量信号
	MRBalancedMode     MRSignalMode = "balanced"     // 平衡模式：质量与数量平衡
	MRAggressiveMode   MRSignalMode = "aggressive"   // 激进模式：数量优先
	MRAdaptiveMode     MRSignalMode = "adaptive"     // 自适应模式：动态调整，平衡数量与质量
)

// EligibleSymbol 符合条件的交易信号
type EligibleSymbol struct {
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"` // "buy" 或 "sell"
	Reason     string  `json:"reason"`
	Multiplier float64 `json:"multiplier"`
	MarketCap  float64 `json:"market_cap"`
}

// StrategyScanner接口在strategy_execution.go中定义，这里不再重复定义

// StrategyMarketData 策略分析所需的市场数据
type StrategyMarketData struct {
	Symbol           string
	Prices           []float64
	Volumes          []float64
	MarketCap        float64
	Volatility       float64
	TrendStrength    float64
	OscillationScore float64
}

// ============================================================================
// 配置相关类型
// ============================================================================

// MeanReversionConfig 均值回归策略统一配置
type MeanReversionConfig struct {
	// 策略启用
	Enabled bool `json:"enabled"`

	// 核心参数
	Core struct {
		Mode       MRSignalMode `json:"mode"`       // 信号模式
		Period     int          `json:"period"`     // 计算周期
		Indicators []string     `json:"indicators"` // 启用的指标
	} `json:"core"`

	// 指标特定配置
	Indicators struct {
		Bollinger struct {
			Enabled    bool    `json:"enabled"`
			Multiplier float64 `json:"multiplier"`
			Weight     float64 `json:"weight"`
		} `json:"bollinger"`

		RSI struct {
			Enabled    bool    `json:"enabled"`
			Overbought int     `json:"overbought"`
			Oversold   int     `json:"oversold"`
			Weight     float64 `json:"weight"`
		} `json:"rsi"`

		PriceChannel struct {
			Enabled bool    `json:"enabled"`
			Weight  float64 `json:"weight"`
		} `json:"price_channel"`
	} `json:"indicators"`

	// 信号质量控制
	SignalQuality struct {
		MinStrength    float64 `json:"min_strength"`
		MinConfidence  float64 `json:"min_confidence"`
		MinConsistency float64 `json:"min_consistency"`
		MinQuality     float64 `json:"min_quality"`
	} `json:"signal_quality"`

	// 风险管理
	RiskManagement struct {
		MaxPositionSize float64 `json:"max_position_size"`
		StopLoss        float64 `json:"stop_loss"`
		TakeProfit      float64 `json:"take_profit"`
		MaxHoldHours    int     `json:"max_hold_hours"`
	} `json:"risk_management"`

	// 增强功能
	Enhancements struct {
		MarketEnvironmentDetection bool `json:"market_environment_detection"`
		IntelligentWeights         bool `json:"intelligent_weights"`
		AdaptiveParameters         bool `json:"adaptive_parameters"`
		PerformanceMonitoring      bool `json:"performance_monitoring"`
	} `json:"enhancements"`

	// 杠杆配置
	EnableLeverage  bool `json:"enable_leverage"`  // 是否启用杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
	MaxLeverage     int  `json:"max_leverage"`     // 最大杠杆倍数

	// 交易配置
	MarginMode string `json:"margin_mode"` // 保证金模式: "ISOLATED" 或 "CROSS"
}

// ============================================================================
// 信号处理相关类型
// ============================================================================

// IndicatorSignal 智能指标信号
type IndicatorSignal struct {
	Type       string  // 指标类型
	BuySignal  bool    // 买入信号
	SellSignal bool    // 卖出信号
	BaseWeight float64 // 基础权重
	Quality    float64 // 信号质量 (0.0-1.0)
	Confidence float64 // 信号置信度 (0.0-1.0)
}

// SignalStrength 信号强度评估结果
type SignalStrength struct {
	BuyStrength   float64 // 买入信号强度
	SellStrength  float64 // 卖出信号强度
	Confidence    float64 // 整体置信度
	Consistency   float64 // 信号一致性
	Quality       float64 // 信号质量
	ActiveSignals int     // 活跃信号数量
}

// SignalDecision 信号决策结果
type SignalDecision struct {
	Action     string  // "buy", "sell", "hold"
	Strength   float64 // 信号强度
	Confidence float64 // 信号置信度
	Reason     string  // 决策原因
}

// ============================================================================
// 指标接口定义
// ============================================================================

// MRIndicator 均值回归技术指标接口
type MRIndicator interface {
	Name() string
	Calculate(prices []float64, params map[string]interface{}) (IndicatorSignal, error)
	GetDefaultParams() map[string]interface{}
	ValidateParams(params map[string]interface{}) error
}

// MRIndicatorFactory 均值回归指标工厂接口
type MRIndicatorFactory interface {
	Create(name string, params map[string]interface{}) (MRIndicator, error)
	GetAvailableIndicators() []string
}

// ============================================================================
// 信号处理器接口
// ============================================================================

// MRSignalProcessor 均值回归信号处理器接口
type MRSignalProcessor interface {
	ProcessSignals(signals []*IndicatorSignal, config *MeanReversionConfig) (*SignalStrength, error)
	MakeDecision(strength *SignalStrength, config *MeanReversionConfig) (*SignalDecision, error)
}

// ============================================================================
// 风险管理接口
// ============================================================================

// MRRiskManager 均值回归风险管理器接口
type MRRiskManager interface {
	CalculatePositionSize(entryPrice float64, stopLoss float64, totalCapital float64, config *MeanReversionConfig) float64
	CalculateStopLoss(entryPrice float64, direction string, config *MeanReversionConfig) float64
	CalculateTakeProfit(entryPrice float64, stopLoss float64, direction string, config *MeanReversionConfig) float64
	ValidateRiskLimits(positionSize float64, dailyLoss float64, config *MeanReversionConfig) bool
}

// ============================================================================
// 配置管理接口
// ============================================================================

// MRConfigManager 均值回归配置管理器接口
type MRConfigManager interface {
	ConvertToUnifiedConfig(conditions pdb.StrategyConditions) (*MeanReversionConfig, error)
	ValidateConfig(config *MeanReversionConfig) error
	GetDefaultConfig(mode MRSignalMode) *MeanReversionConfig
}

// ============================================================================
// 验证器接口
// ============================================================================

// MRValidator 均值回归策略验证器接口
type MRValidator interface {
	ValidateStrategy(config *MeanReversionConfig, marketData *StrategyMarketData) error
	Backtest(config *MeanReversionConfig, historicalData []StrategyMarketData, startTime, endTime time.Time) (*BacktestResult, error)
	StressTest(config *MeanReversionConfig, scenarios []StressTestScenario) (*StressTestResult, error)
}

// BacktestResult 回测结果
type BacktestResult struct {
	TotalTrades int
	WinRate     float64
	TotalPnL    float64
	MaxDrawdown float64
	SharpeRatio float64
	Trades      []TradeRecord
}

// TradeRecord 交易记录
type TradeRecord struct {
	Symbol     string
	Side       string
	EntryTime  time.Time
	EntryPrice float64
	ExitTime   time.Time
	ExitPrice  float64
	Quantity   float64
	PnL        float64
	PnLPercent float64
	Reason     string
}

// StressTestScenario 压力测试场景
type StressTestScenario struct {
	Name            string
	Description     string
	MarketData      []StrategyMarketData
	ExpectedOutcome string
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	PassedScenarios []string
	FailedScenarios []string
	OverallScore    float64
	Recommendations []string
}

// ============================================================================
// 核心策略接口
// ============================================================================

// MRStrategy 均值回归策略核心接口
type MRStrategy interface {
	// 核心功能
	Scan(ctx context.Context, symbol string, marketData *StrategyMarketData, config *MeanReversionConfig) (*EligibleSymbol, error)

	// 配置管理
	GetConfigManager() MRConfigManager

	// 组件访问
	GetIndicatorFactory() MRIndicatorFactory
	GetSignalProcessor() MRSignalProcessor
	GetRiskManager() MRRiskManager
	GetValidator() MRValidator

	// 元信息
	GetStrategyType() string
	IsEnabled(config *MeanReversionConfig) bool

	// 适配器方法
	ToStrategyScanner() interface{} // 返回StrategyScanner接口，由调用方处理类型转换
}
