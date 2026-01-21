package server

import "time"

// MarketRegime 市场环境枚举
type MarketRegime int

const (
	MarketRegimeExtremeBear MarketRegime = iota // 极熊市：极低波动，极弱趋势，完全停止交易
	MarketRegimeSideways                        // 震荡市：低波动，无明确趋势，适合高频短线
	MarketRegimeWeakBull                        // 弱多头：中等波动，温和上涨趋势
	MarketRegimeWeakBear                        // 弱空头：中等波动，温和下跌趋势
	MarketRegimeStrongBull                      // 强多头：高波动，强劲上涨趋势，全仓做多
	MarketRegimeStrongBear                      // 强空头：高波动，强劲下跌趋势，全仓做空
)

// String 返回市场环境的字符串表示
func (mr MarketRegime) String() string {
	switch mr {
	case MarketRegimeExtremeBear:
		return "extreme_bear"
	case MarketRegimeSideways:
		return "sideways"
	case MarketRegimeWeakBull:
		return "weak_bull"
	case MarketRegimeWeakBear:
		return "weak_bear"
	case MarketRegimeStrongBull:
		return "strong_bull"
	case MarketRegimeStrongBear:
		return "strong_bear"
	default:
		return "unknown"
	}
}

// TechnicalIndicators 技术指标
type TechnicalIndicators struct {
	// 现有指标
	RSI        float64 `json:"rsi"`         // 相对强弱指标 0-100
	MACD       float64 `json:"macd"`        // MACD值
	MACDSignal float64 `json:"macd_signal"` // MACD信号线
	MACDHist   float64 `json:"macd_hist"`   // MACD柱状图
	Trend      string  `json:"trend"`       // "up"/"down"/"sideways"

	// 新增指标：布林带
	BBUpper        float64 `json:"bb_upper"`        // 布林带上轨
	BBMiddle       float64 `json:"bb_middle"`       // 布林带中轨（SMA20）
	BBLower        float64 `json:"bb_lower"`        // 布林带下轨
	BollingerUpper float64 `json:"bollinger_upper"` // 布林带上轨（别名）
	BollingerLower float64 `json:"bollinger_lower"` // 布林带下轨（别名）
	BBWidth        float64 `json:"bb_width"`        // 布林带宽度（百分比）
	BBPosition     float64 `json:"bb_position"`     // 价格在布林带中的位置 0-1（0=下轨，1=上轨）

	// 新增指标：KDJ
	K float64 `json:"k"` // K值 0-100
	D float64 `json:"d"` // D值 0-100
	J float64 `json:"j"` // J值 0-100

	// 新增指标：均线系统
	MA5   float64 `json:"ma5"`   // 5日均线
	MA10  float64 `json:"ma10"`  // 10日均线
	MA20  float64 `json:"ma20"`  // 20日均线
	MA50  float64 `json:"ma50"`  // 50日均线
	MA60  float64 `json:"ma60"`  // 60日均线
	MA200 float64 `json:"ma200"` // 200日均线（如果有足够数据）

	// 新增指标：成交量
	OBV         float64 `json:"obv"`          // 能量潮（On-Balance Volume）
	VolumeMA5   float64 `json:"volume_ma5"`   // 5日成交量均线
	VolumeMA20  float64 `json:"volume_ma20"`  // 20日成交量均线
	VolumeRatio float64 `json:"volume_ratio"` // 成交量比率（当前成交量/20日均量）

	// 新增指标：支撑位/阻力位
	SupportLevel       float64 `json:"support_level"`       // 支撑位价格
	ResistanceLevel    float64 `json:"resistance_level"`    // 阻力位价格
	SupportStrength    float64 `json:"support_strength"`    // 支撑位强度 0-100
	ResistanceStrength float64 `json:"resistance_strength"` // 阻力位强度 0-100

	// 新增指标：动量指标
	Momentum5          float64 `json:"momentum_5"`          // 5日动量
	Momentum10         float64 `json:"momentum_10"`         // 10日动量
	Momentum20         float64 `json:"momentum_20"`         // 20日动量
	MomentumDivergence float64 `json:"momentum_divergence"` // 动量发散度

	// 新增指标：波动率指标
	Volatility      float64 `json:"volatility"`       // 波动率（通用）
	Volatility5     float64 `json:"volatility_5"`     // 5日波动率
	Volatility20    float64 `json:"volatility_20"`    // 20日波动率
	VolatilityRatio float64 `json:"volatility_ratio"` // 波动率比率

	// 新增指标：威廉指标
	WilliamsR float64 `json:"williams_r"` // 威廉指标 -100到0

	// 新增指标：顺势指标
	CCI float64 `json:"cci"` // 商品通道指数

	// 新增指标：信号强度和风险等级
	SignalStrength float64 `json:"signal_strength"` // 信号强度 0-100
	RiskLevel      string  `json:"risk_level"`      // "low"/"medium"/"high"/"critical"

	// 新增指标：网格交易专用字段（兼容性字段）
	Signal    float64 `json:"signal"`    // MACD信号线（别名）
	Histogram float64 `json:"histogram"` // MACD柱状图（别名）
}

// BacktestConfig 回测配置
type BacktestConfig struct {
	Symbol               string    `json:"symbol"`  // 主要币种（向后兼容）
	Symbols              []string  `json:"symbols"` // 多币种列表，为空时使用Symbol
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	Strategy             string    `json:"strategy"`
	InitialCash          float64   `json:"initial_cash"`
	PositionSize         float64   `json:"position_size"`
	StopLoss             float64   `json:"stop_loss"`
	TakeProfit           float64   `json:"take_profit"`
	MaxPosition          float64   `json:"max_position"`
	RiskLevel            string    `json:"risk_level"`
	Timeframe            string    `json:"timeframe"`
	Commission           float64   `json:"commission"`             // 手续费率
	MaxHoldTime          int       `json:"max_hold_time"`          // 最大持有时间（周期）
	MaxDrawdown          float64   `json:"max_drawdown"`           // 最大回撤限制
	MaxDailyLoss         float64   `json:"max_daily_loss"`         // 最大单日损失
	MaxConsecutiveLosses int       `json:"max_consecutive_losses"` // 最大连续亏损次数
	MinCapitalRatio      float64   `json:"min_capital_ratio"`      // 最低资本比例

	// 用户策略相关字段
	UserStrategyID uint `json:"user_strategy_id,omitempty"` // 用户策略ID，为0表示普通回测
}

// SymbolPerformance 单个币种的性能统计
type SymbolPerformance struct {
	Symbol        string  `json:"symbol"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalReturn   float64 `json:"total_return"`
	AvgWin        float64 `json:"avg_win"`
	AvgLoss       float64 `json:"avg_loss"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	ProfitFactor  float64 `json:"profit_factor"`
	ExposureTime  float64 `json:"exposure_time"` // 持仓时间占比
}

// BacktestResult 回测结果
type BacktestResult struct {
	Config          BacktestConfig                `json:"config"`
	Summary         BacktestSummary               `json:"summary"`
	Trades          []TradeRecord                 `json:"trades"`
	DailyReturns    []DailyReturn                 `json:"daily_returns"`
	RiskMetrics     RiskMetrics                   `json:"risk_metrics"`
	Performance     PerformanceMetrics            `json:"performance"`
	PortfolioValues []float64                     `json:"portfolio_values"` // 组合价值历史
	SymbolStats     map[string]*SymbolPerformance `json:"symbol_stats"`     // 每个币种的性能统计
	TotalReturn     float64                       `json:"total_return"`
	WinRate         float64                       `json:"win_rate"`
	MaxDrawdown     float64                       `json:"max_drawdown"`
	SharpeRatio     float64                       `json:"sharpe_ratio"`
}

// BacktestSummary 回测摘要
type BacktestSummary struct {
	TotalTrades     int     `json:"total_trades"`
	WinningTrades   int     `json:"winning_trades"`
	LosingTrades    int     `json:"losing_trades"`
	WinRate         float64 `json:"win_rate"`
	TotalReturn     float64 `json:"total_return"`
	AnnualReturn    float64 `json:"annual_return"`
	MaxDrawdown     float64 `json:"max_drawdown"`
	SharpeRatio     float64 `json:"sharpe_ratio"`
	Volatility      float64 `json:"volatility"`
	AvgTradeReturn  float64 `json:"avg_trade_return"`
	TotalCommission float64 `json:"total_commission"`
}

// TradeRecord 交易记录
type TradeRecord struct {
	Symbol       string     `json:"symbol"`
	Side         string     `json:"side"` // "buy", "sell"
	Quantity     float64    `json:"quantity"`
	Price        float64    `json:"price"`
	Timestamp    time.Time  `json:"timestamp"`
	Commission   float64    `json:"commission"`
	PnL          float64    `json:"pnl"`
	ExitPrice    *float64   `json:"exit_price,omitempty"`
	ExitTime     *time.Time `json:"exit_time,omitempty"`
	Reason       string     `json:"reason"`
	AIConfidence float64    `json:"ai_confidence,omitempty"` // Added
	RiskScore    float64    `json:"risk_score,omitempty"`    // Added
}

// DailyReturn 每日收益
type DailyReturn struct {
	Date   time.Time `json:"date"`
	Value  float64   `json:"value"`
	Return float64   `json:"return"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	ValueAtRisk_95    float64 `json:"value_at_risk_95"`
	ValueAtRisk_99    float64 `json:"value_at_risk_99"`
	ExpectedShortfall float64 `json:"expected_shortfall"`
	Beta              float64 `json:"beta"`
	Volatility        float64 `json:"volatility"`
	DownsideDeviation float64 `json:"downside_deviation"`
}

// PerformanceMetrics 绩效指标
type PerformanceMetrics struct {
	TotalReturn      float64 `json:"total_return"`
	AnnualReturn     float64 `json:"annual_return"`
	Volatility       float64 `json:"volatility"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	SortinoRatio     float64 `json:"sortino_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	InformationRatio float64 `json:"information_ratio"`
	OmegaRatio       float64 `json:"omega_ratio"`
	GainToPainRatio  float64 `json:"gain_to_pain_ratio"`
	RecoveryFactor   float64 `json:"recovery_factor"`
	ProfitFactor     float64 `json:"profit_factor"`
	PayoffRatio      float64 `json:"payoff_ratio"`
	KRatio           float64 `json:"k_ratio"`
	SafeFDrawdown    float64 `json:"safe_f_drawdown"`
	WinRate          float64 `json:"win_rate"`
	AvgWin           float64 `json:"avg_win"`
	AvgLoss          float64 `json:"avg_loss"`
	Expectancy       float64 `json:"expectancy"`
}

// WalkForwardAnalysis 步进分析
type WalkForwardAnalysis struct {
	InSamplePeriod    int                 `json:"in_sample_period"`     // 样本内周期(月)
	OutOfSamplePeriod int                 `json:"out_of_sample_period"` // 样本外周期(月)
	StepSize          int                 `json:"step_size"`            // 步长(月)
	StartDate         time.Time           `json:"start_date"`
	EndDate           time.Time           `json:"end_date"`
	Windows           []WalkForwardWindow `json:"windows"`
	Summary           WalkForwardSummary  `json:"summary"`
}

// WalkForwardResult 步进分析结果
type WalkForwardResult struct {
	Analysis WalkForwardAnalysis `json:"analysis"`
}

// WalkForwardWindow 步进窗口
type WalkForwardWindow struct {
	StartDate   time.Time      `json:"start_date"`
	EndDate     time.Time      `json:"end_date"`
	InSample    BacktestResult `json:"in_sample"`
	OutOfSample BacktestResult `json:"out_of_sample"`
}

// WalkForwardSummary 步进分析摘要
type WalkForwardSummary struct {
	TotalWindows   int     `json:"total_windows"`
	AvgOutOfSample float64 `json:"avg_out_of_sample_return"`
	MaxOutOfSample float64 `json:"max_out_of_sample_return"`
	MinOutOfSample float64 `json:"min_out_of_sample_return"`
	Consistency    float64 `json:"consistency"`
}

// MonteCarloAnalysis 蒙特卡洛分析
type MonteCarloAnalysis struct {
	Simulations         int                    `json:"simulations"`      // 模拟次数
	ConfidenceLevel     float64                `json:"confidence_level"` // 置信水平
	BootstrapSize       int                    `json:"bootstrap_size"`   // 自举样本大小
	Distribution        MonteCarloDistribution `json:"distribution"`
	Scenarios           []MonteCarloScenario   `json:"scenarios"`
	ConfidenceIntervals []ConfidenceInterval   `json:"confidence_intervals"`
}

// MonteCarloResult 蒙特卡洛分析结果
type MonteCarloResult struct {
	Analysis MonteCarloAnalysis `json:"analysis"`
}

// MonteCarloDistribution 蒙特卡洛分布
type MonteCarloDistribution struct {
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"std_dev"`
	Skewness  float64 `json:"skewness"`
	Kurtosis  float64 `json:"kurtosis"`
	MinReturn float64 `json:"min_return"`
	MaxReturn float64 `json:"max_return"`
}

// MonteCarloScenario 蒙特卡洛情景
type MonteCarloScenario struct {
	Return      float64 `json:"return"`
	Probability float64 `json:"probability"`
	Description string  `json:"description"`
}

// ConfidenceInterval 置信区间
type ConfidenceInterval struct {
	Level      float64 `json:"level"`
	LowerBound float64 `json:"lower_bound"`
	UpperBound float64 `json:"upper_bound"`
}

// StrategyOptimization 策略优化
type StrategyOptimization struct {
	Parameters     []OptimizationParameter `json:"parameters"`
	Objective      string                  `json:"objective"` // "sharpe", "return", "win_rate", "drawdown"
	Method         string                  `json:"method"`    // "grid", "random", "genetic"
	MaxIterations  int                     `json:"max_iterations"`
	PopulationSize int                     `json:"population_size"` // 遗传算法种群大小
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	Parameters     map[string]interface{} `json:"parameters"`
	ObjectiveValue float64                `json:"objective_value"`
	Constraints    map[string]bool        `json:"constraints"`
}

// OptimizationParameter 优化参数
type OptimizationParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	DefaultValue interface{} `json:"default_value"`
	MinValue     interface{} `json:"min_value"`
	MaxValue     interface{} `json:"max_value"`
	StepSize     interface{} `json:"step_size"`
	DataType     string      `json:"data_type"` // "int", "float", "string"
}

// AttributionAnalysis 归因分析
type AttributionAnalysis struct {
	TimeHorizon     string             `json:"time_horizon"`
	BenchmarkSymbol string             `json:"benchmark_symbol"`
	StartDate       time.Time          `json:"start_date"`
	EndDate         time.Time          `json:"end_date"`
	TotalReturn     float64            `json:"total_return"`
	BenchmarkReturn float64            `json:"benchmark_return"`
	ActiveReturn    float64            `json:"active_return"`
	Attribution     map[string]float64 `json:"attribution"`
	RiskAttribution map[string]float64 `json:"risk_attribution"`
}

// RiskAttribution 风险归因
type RiskAttribution struct {
	TotalVolatility       float64            `json:"total_volatility"`
	BenchmarkVolatility   float64            `json:"benchmark_volatility"`
	ActiveRisk            float64            `json:"active_risk"`
	AssetAllocationRisk   float64            `json:"asset_allocation_risk"`
	SecuritySelectionRisk float64            `json:"security_selection_risk"`
	RiskDecomposition     map[string]float64 `json:"risk_decomposition"`
}
