package db

import (
	"time"

	"gorm.io/datatypes"
)

// 资产/资金流（保持不变）
type PortfolioSnapshot struct {
	ID        uint      `gorm:"primaryKey"`
	RunID     string    `gorm:"type:char(36);index:idx_ps_run_ent,unique"`
	Entity    string    `gorm:"size:64;index:idx_ps_run_ent,unique"`
	TotalUSD  string    `gorm:"type:decimal(38,8)"`
	AsOf      time.Time `gorm:"index"`
	CreatedAt time.Time
}

type Holding struct {
	ID        uint   `gorm:"primaryKey"`
	RunID     string `gorm:"type:char(36);index:idx_h_run_ent_chain_sym,unique"`
	Entity    string `gorm:"size:64;index:idx_h_run_ent_chain_sym,unique"`
	Chain     string `gorm:"size:32;index:idx_h_run_ent_chain_sym,unique"`
	Symbol    string `gorm:"size:32;index:idx_h_run_ent_chain_sym,unique"`
	Amount    string `gorm:"type:decimal(38,18)"`
	Decimals  int
	ValueUSD  string `gorm:"type:decimal(38,8)"`
	CreatedAt time.Time
}

type WeeklyFlow struct {
	ID        uint   `gorm:"primaryKey"`
	RunID     string `gorm:"type:char(36);index:idx_w_run_ent_coin_week,unique"`
	Entity    string `gorm:"size:64;index:idx_w_run_ent_coin_week,unique"`
	Coin      string `gorm:"size:16;index:idx_w_run_ent_coin_week,unique"`
	Week      string `gorm:"size:10;index:idx_w_run_ent_coin_week,unique"` // 2025-W35
	In        string `gorm:"type:decimal(38,18)"`
	Out       string `gorm:"type:decimal(38,18)"`
	Net       string `gorm:"type:decimal(38,18)"`
	CreatedAt time.Time
}

type DailyFlow struct {
	ID        uint   `gorm:"primaryKey"`
	RunID     string `gorm:"type:char(36);index:idx_d_run_ent_coin_day,unique"`
	Entity    string `gorm:"size:64;index:idx_d_run_ent_coin_day,unique"`
	Coin      string `gorm:"size:16;index:idx_d_run_ent_coin_day,unique"`
	Day       string `gorm:"type:date;index:idx_d_run_ent_coin_day,unique"`
	In        string `gorm:"type:decimal(38,18)"`
	Out       string `gorm:"type:decimal(38,18)"`
	Net       string `gorm:"type:decimal(38,18)"`
	CreatedAt time.Time
}

// 实时转账事件（复合唯一键 + LogIndex）
type TransferEvent struct {
	ID         uint      `gorm:"primaryKey"`
	RunID      string    `gorm:"type:char(36);index"`
	Entity     string    `gorm:"size:64;uniqueIndex:ux_te"`
	Chain      string    `gorm:"size:32;uniqueIndex:ux_te"`
	Coin       string    `gorm:"size:16;uniqueIndex:ux_te"`
	Direction  string    `gorm:"size:8;uniqueIndex:ux_te"` // "in"/"out"
	Amount     string    `gorm:"type:decimal(38,18)"`
	TxID       string    `gorm:"size:128;uniqueIndex:ux_te"`
	Address    string    `gorm:"size:128;uniqueIndex:ux_te"` // 命中的监控地址
	From       string    `gorm:"size:128"`
	To         string    `gorm:"size:128"`
	LogIndex   int       `gorm:"uniqueIndex:ux_te;default:-1"` // ERC20: 链上 logIndex；原生: -1
	OccurredAt time.Time `gorm:"index"`
	CreatedAt  time.Time
}

// 扫描游标（断点续扫）
type TransferCursor struct {
	ID        uint   `gorm:"primaryKey"`
	Entity    string `gorm:"size:64;uniqueIndex:ux_cursor"`
	Chain     string `gorm:"size:32;uniqueIndex:ux_cursor"`
	Block     uint64 `gorm:"type:bigint unsigned"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CoinCap资产映射表
type CoinCapAssetMapping struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Symbol    string    `gorm:"size:16;uniqueIndex:ux_symbol;not null" json:"symbol"`     // 交易符号，如BTC, ETH
	AssetID   string    `gorm:"size:64;uniqueIndex:ux_asset_id;not null" json:"asset_id"` // CoinCap资产ID，如bitcoin, ethereum
	Name      string    `gorm:"size:128" json:"name"`                                     // 资产全称，如Bitcoin
	Rank      string    `gorm:"size:8" json:"rank"`                                       // 排名
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CoinCap市值数据表 - 存储从CoinCap获取的市值信息
type CoinCapMarketData struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Symbol  string `gorm:"size:16;uniqueIndex:idx_symbol_updated;priority:1;not null" json:"symbol"` // 交易符号，如BTC, ETH
	AssetID string `gorm:"size:64;index" json:"asset_id"`                                            // CoinCap资产ID (id字段)
	Name    string `gorm:"size:128" json:"name"`                                                     // 资产全称
	Rank    string `gorm:"size:8" json:"rank"`                                                       // 排名

	// 价格数据 - 存储为字符串以保持精度
	PriceUSD   string `gorm:"size:64" json:"price_usd"`  // 当前价格（美元）- 字符串格式
	Change24Hr string `gorm:"size:32" json:"change_24h"` // 24h涨跌幅（%）- 字符串格式

	// 市值数据
	MarketCapUSD string `gorm:"size:64;index" json:"market_cap_usd"` // 市值（美元）- 字符串格式

	// 供应量数据
	CirculatingSupply string `gorm:"size:64" json:"circulating_supply"` // 流通供应量 (supply字段)
	TotalSupply       string `gorm:"size:64" json:"total_supply"`       // 总供应量 (maxSupply字段)

	// 交易数据
	Volume24Hr string `gorm:"size:64" json:"volume_24h"` // 24h成交量（美元）
	VWAP24Hr   string `gorm:"size:64" json:"vwap_24h"`   // 24h成交量加权平均价

	// 额外信息
	Explorer string `gorm:"size:256" json:"explorer"` // 区块链浏览器链接

	// 时间戳
	UpdatedAt time.Time `gorm:"index:idx_symbol_updated;priority:2" json:"updated_at"` // CoinCap数据更新时间
	CreatedAt time.Time `json:"created_at"`
}

// 交易策略
type TradingStrategy struct {
	ID          uint   `gorm:"primaryKey"                      json:"id"`
	UserID      uint   `gorm:"index;not null"                  json:"user_id"`
	Name        string `gorm:"size:128;not null"               json:"name"`
	Description string `gorm:"type:text"                       json:"description"`

	// 策略条件
	Conditions StrategyConditions `gorm:"embedded"                json:"conditions"`

	// 运行状态
	IsRunning   bool       `gorm:"default:false"                 json:"is_running"`
	LastRunAt   *time.Time `json:"last_run_at"`
	RunInterval int        `gorm:"default:60"                    json:"run_interval"` // 运行间隔（分钟）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 策略执行记录
type StrategyExecution struct {
	ID         uint       `gorm:"primaryKey"                      json:"id"`
	StrategyID uint       `gorm:"index;not null"                  json:"strategy_id"`
	UserID     uint       `gorm:"index;not null"                  json:"user_id"`
	Status     string     `gorm:"size:32;default:'pending'"      json:"status"` // pending, running, completed, failed, paused
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	Duration   int        `json:"duration"` // 执行时长（秒）

	// 启动参数
	RunInterval    int     `gorm:"default:60"                    json:"run_interval"`     // 运行间隔（分钟）
	MaxRuns        int     `gorm:"default:0"                     json:"max_runs"`         // 最大运行次数，0表示无限
	AutoStop       bool    `gorm:"default:false"                json:"auto_stop"`         // 执行后自动停止
	CreateOrders   bool    `gorm:"default:true"                  json:"create_orders"`    // 是否自动创建订单
	ExecutionDelay int     `gorm:"default:60"                    json:"execution_delay"`  // 执行延迟（秒），订单创建后多久执行
	PerOrderAmount float64 `gorm:"type:decimal(20,8);default:0"  json:"per_order_amount"` // 每一单的金额（U单位），0表示使用默认金额

	// 执行统计
	RunCount int `gorm:"default:0"                          json:"run_count"` // 已运行次数

	// 执行结果
	TotalOrders     int     `gorm:"default:0"                      json:"total_orders"`
	SuccessOrders   int     `gorm:"default:0"                      json:"success_orders"`
	FailedOrders    int     `gorm:"default:0"                      json:"failed_orders"`
	TotalPnL        float64 `gorm:"column:total_pnl;type:decimal(20,8);default:0" json:"total_pnl"`
	WinRate         float64 `gorm:"type:decimal(5,2);default:0"    json:"win_rate"`
	PnlPercentage   float64 `gorm:"type:decimal(8,4);default:0"    json:"pnl_percentage"`   // 盈亏百分比
	TotalInvestment float64 `gorm:"type:decimal(20,8);default:0"   json:"total_investment"` // 买入总金额
	CurrentValue    float64 `gorm:"type:decimal(20,8);default:0"   json:"current_value"`    // 当前资产价值

	// 执行过程跟踪
	CurrentStep   string `gorm:"size:64;default:''"           json:"current_step"`    // 当前执行步骤
	StepProgress  int    `gorm:"default:0"                     json:"step_progress"`  // 步骤进度(0-100)
	TotalProgress int    `gorm:"default:0"                     json:"total_progress"` // 总体进度(0-100)
	CurrentSymbol string `gorm:"size:32;default:''"           json:"current_symbol"`  // 当前处理的交易对
	ErrorMessage  string `gorm:"type:text"                     json:"error_message"`  // 错误信息

	// 执行日志
	Logs string `gorm:"type:text"                          json:"logs"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Strategy TradingStrategy `gorm:"foreignKey:StrategyID"       json:"strategy,omitempty"`
}

// 策略执行步骤记录
type StrategyExecutionStep struct {
	ID           uint       `gorm:"primaryKey"                    json:"id"`
	ExecutionID  uint       `gorm:"index;not null"                json:"execution_id"`
	StepName     string     `gorm:"size:64;not null"             json:"step_name"` // 步骤名称
	StepType     string     `gorm:"size:32;not null"             json:"step_type"` // 步骤类型: strategy_check, market_data, order_placement, risk_check等
	Symbol       string     `gorm:"size:32"                       json:"symbol"`   // 涉及的交易对
	Status       string     `gorm:"size:32;default:'pending'"    json:"status"`    // pending, running, completed, failed, skipped
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	Duration     int        `json:"duration"`                                           // 步骤执行时长（秒）
	Result       string     `gorm:"type:text"                     json:"result"`        // 执行结果描述
	ErrorMessage string     `gorm:"type:text"                     json:"error_message"` // 错误信息
	Data         string     `gorm:"type:text"                     json:"data"`          // 附加数据(JSON格式)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Execution StrategyExecution `gorm:"foreignKey:ExecutionID"   json:"execution,omitempty"`
}

// TableName 指定表名
func (StrategyExecutionStep) TableName() string {
	return "strategy_execution_steps"
}

// 策略条件结构体
type StrategyConditions struct {
	// ========== 基础条件 ==========
	SpotContract bool   `json:"spot_contract"` // 需要有现货+合约
	TradingType  string `json:"trading_type"`  // 交易类型: "spot", "futures", "both"

	// ========== 交易配置 ==========

	// 交易方向和杠杆
	AllowedDirections string `json:"allowed_directions" gorm:"type:text"` // 允许的交易方向: "LONG,SHORT"
	EnableLeverage    bool   `json:"enable_leverage"`                     // 是否启用杠杆配置
	DefaultLeverage   int    `json:"default_leverage"`                    // 默认杠杆倍数
	MaxLeverage       int    `json:"max_leverage"`                        // 最大杠杆倍数
	MarginMode        string `json:"margin_mode"`                         // 保证金模式: "ISOLATED" 或 "CROSS"
	SkipHeldPositions bool   `json:"skip_held_positions"`                 // 是否跳过已在持仓的币种

	// 平仓订单过滤
	SkipCloseOrdersWithin24Hours bool `json:"skip_close_orders_within_24_hours"` // 是否跳过24小时内的平仓订单（已废弃）
	SkipCloseOrdersHours         int  `json:"skip_close_orders_hours"`           // 跳过平仓记录的小时数（0表示不跳过）

	// 盈利加仓策略
	ProfitScalingEnabled      bool           `json:"profit_scaling_enabled"`                        // 是否启用盈利加仓
	ProfitScalingPercent      float64        `json:"profit_scaling_percent"`                        // 触发加仓的盈利百分比
	ProfitScalingAmount       float64        `json:"profit_scaling_amount"`                         // 加仓金额（U单位）
	ProfitScalingMaxCount     int            `json:"profit_scaling_max_count"`                      // 每个币种的最大加仓次数
	ProfitScalingCurrentCount int            `json:"profit_scaling_current_count"`                  // 策略级别的已加仓次数（兼容旧逻辑）
	ProfitScalingSymbolCounts datatypes.JSON `json:"profit_scaling_symbol_counts" gorm:"type:json"` // 各币种的加仓计数器，格式：{"BTCUSDT": 1, "ETHUSDT": 0}

	// 整体仓位止盈止损
	OverallStopLossEnabled   bool    `json:"overall_stop_loss_enabled"`   // 是否启用整体止损
	OverallStopLossPercent   float64 `json:"overall_stop_loss_percent"`   // 整体止损百分比（亏损触发）
	OverallTakeProfitPercent float64 `json:"overall_take_profit_percent"` // 整体止盈百分比（盈利触发，百分比>0即启用）

	// ========== 币种选择 ==========
	UseSymbolWhitelist bool           `json:"use_symbol_whitelist"`              // 是否使用币种白名单模式
	SymbolWhitelist    datatypes.JSON `json:"symbol_whitelist" gorm:"type:json"` // 用户指定的交易币种列表
	UseSymbolBlacklist bool           `json:"use_symbol_blacklist"`              // 是否使用币种黑名单模式
	SymbolBlacklist    datatypes.JSON `json:"symbol_blacklist" gorm:"type:json"` // 用户指定的禁止交易币种列表

	// ========== 传统交易策略 ==========

	// 开空相关条件
	NoShortBelowMarketCap bool    `json:"no_short_below_market_cap"` // 不开空市值限制启用
	MarketCapLimitShort   float64 `json:"market_cap_limit_short"`    // 开空市值限制（万）

	// 涨幅开空条件
	ShortOnGainers   bool    `json:"short_on_gainers"`   // 涨幅开空启用
	GainersRankLimit int     `json:"gainers_rank_limit"` // 开空涨幅排名限制
	ShortMultiplier  float64 `json:"short_multiplier"`   // 开空倍数

	// 涨幅开多条件
	LongOnSmallGainers   bool    `json:"long_on_small_gainers"`   // 小市值开多启用
	MarketCapLimitLong   float64 `json:"market_cap_limit_long"`   // 开多市值限制（万）
	GainersRankLimitLong int     `json:"gainers_rank_limit_long"` // 开多涨幅排名限制
	LongMultiplier       float64 `json:"long_multiplier"`         // 开多倍数

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

	// ========== 技术指标策略 ==========

	// 均线策略
	MovingAverageEnabled bool   `json:"moving_average_enabled"` // 均线策略启用
	MAType               string `json:"ma_type"`                // 均线类型: "SMA", "EMA", "WMA"
	ShortMAPeriod        int    `json:"short_ma_period"`        // 短期均线周期
	LongMAPeriod         int    `json:"long_ma_period"`         // 长期均线周期
	MACrossSignal        string `json:"ma_cross_signal"`        // 交叉信号: "GOLDEN_CROSS", "DEATH_CROSS", "BOTH"
	MATrendFilter        bool   `json:"ma_trend_filter"`        // 趋势过滤启用
	MATrendDirection     string `json:"ma_trend_direction"`     // 趋势方向: "UP", "DOWN", "BOTH"
	MASignalMode         string `json:"ma_signal_mode"`         // 信号模式: "QUALITY_FIRST", "QUANTITY_FIRST"

	// ========== 均值回归策略 ==========

	MeanReversionEnabled    bool    `json:"mean_reversion_enabled"`     // 均值回归策略启用
	MRBollingerBandsEnabled bool    `json:"mr_bollinger_bands_enabled"` // 布林带均值回归启用
	MRRSIEnabled            bool    `json:"mr_rsi_enabled"`             // RSI均值回归启用
	MRPriceChannelEnabled   bool    `json:"mr_price_channel_enabled"`   // 价格通道均值回归启用
	MRPeriod                int     `json:"mr_period"`                  // 均值回归计算周期
	MRBollingerMultiplier   float64 `json:"mr_bollinger_multiplier"`    // 布林带倍数 (默认2.0)
	MRRSIOverbought         int     `json:"mr_rsi_overbought"`          // RSI超买阈值 (默认70)
	MRRSIOversold           int     `json:"mr_rsi_oversold"`            // RSI超卖阈值 (默认30)
	MRChannelPeriod         int     `json:"mr_channel_period"`          // 价格通道周期
	MRMinReversionStrength  float64 `json:"mr_min_reversion_strength"`  // 最小回归强度
	MRSignalMode            string  `json:"mr_signal_mode"`             // 信号模式: "CONSERVATIVE", "AGGRESSIVE"

	// ========== 均值回归策略增强功能 ==========

	// 策略版本和模式
	MeanReversionMode    string `json:"mean_reversion_mode"`     // 策略模式: "basic" 或 "enhanced"
	MeanReversionSubMode string `json:"mean_reversion_sub_mode"` // 子模式: "conservative" 或 "aggressive"

	// 增强功能开关
	MarketEnvironmentDetection bool `json:"market_environment_detection"` // 市场环境检测启用
	IntelligentWeights         bool `json:"intelligent_weights"`          // 智能信号权重启用
	AdvancedRiskManagement     bool `json:"advanced_risk_management"`     // 高级风险管理启用
	AdaptiveParameters         bool `json:"adaptive_parameters"`          // 自适应参数启用
	PerformanceMonitoring      bool `json:"performance_monitoring"`       // 性能监控启用
	ModeSwitching              bool `json:"mode_switching"`               // 模式切换启用

	// 市场环境检测参数
	MREnvTrendThreshold       float64 `json:"mr_env_trend_threshold"`       // 趋势强度阈值
	MREnvVolatilityThreshold  float64 `json:"mr_env_volatility_threshold"`  // 波动率阈值
	MREnvOscillationThreshold float64 `json:"mr_env_oscillation_threshold"` // 震荡指数阈值

	// 智能权重参数
	MRWeightBollingerBands float64 `json:"mr_weight_bollinger_bands"` // 布林带权重
	MRWeightRSI            float64 `json:"mr_weight_rsi"`             // RSI权重
	MRWeightPriceChannel   float64 `json:"mr_weight_price_channel"`   // 价格通道权重
	MRWeightTimeDecay      float64 `json:"mr_weight_time_decay"`      // 时间衰减权重

	// 高级风险管理参数
	MRMaxDailyLoss         float64 `json:"mr_max_daily_loss"`         // 每日最大亏损比例
	MRMaxPositionSize      float64 `json:"mr_max_position_size"`      // 最大仓位比例
	MRStopLossMultiplier   float64 `json:"mr_stop_loss_multiplier"`   // 止损倍数
	MRTakeProfitMultiplier float64 `json:"mr_take_profit_multiplier"` // 止盈倍数
	MRMaxHoldHours         int     `json:"mr_max_hold_hours"`         // 最大持仓小时数

	// 自适应参数
	MRAutoAdjustPeriod     bool `json:"mr_auto_adjust_period"`     // 自动调整周期
	MRAutoAdjustMultiplier bool `json:"mr_auto_adjust_multiplier"` // 自动调整倍数
	MRAutoAdjustThresholds bool `json:"mr_auto_adjust_thresholds"` // 自动调整阈值

	// 候选币种优化参数
	MRCandidateMinOscillation float64 `json:"mr_candidate_min_oscillation"` // 最小振荡性要求
	MRCandidateMinLiquidity   float64 `json:"mr_candidate_min_liquidity"`   // 最小流动性要求
	MRCandidateMaxVolatility  float64 `json:"mr_candidate_max_volatility"`  // 最大波动率限制

	// 保守模式特殊要求
	MRRequireMultipleSignals         bool `json:"mr_require_multiple_signals"`          // 需要多重信号确认
	MRRequireVolumeConfirmation      bool `json:"mr_require_volume_confirmation"`       // 需要成交量确认
	MRRequireTimeFilter              bool `json:"mr_require_time_filter"`               // 需要时间过滤
	MRRequireMarketEnvironmentFilter bool `json:"mr_require_market_environment_filter"` // 需要市场环境过滤

	// ========== 网格交易策略 ==========

	GridTradingEnabled   bool    `json:"grid_trading_enabled"`   // 网格交易策略启用
	GridUpperPrice       float64 `json:"grid_upper_price"`       // 网格上限价格
	GridLowerPrice       float64 `json:"grid_lower_price"`       // 网格下限价格
	GridLevels           int     `json:"grid_levels"`            // 网格层数
	GridProfitPercent    float64 `json:"grid_profit_percent"`    // 网格利润百分比
	GridInvestmentAmount float64 `json:"grid_investment_amount"` // 网格投资金额(USDT)
	GridRebalanceEnabled bool    `json:"grid_rebalance_enabled"` // 网格再平衡启用
	GridStopLossEnabled  bool    `json:"grid_stop_loss_enabled"` // 网格止损启用
	GridStopLossPercent  float64 `json:"grid_stop_loss_percent"` // 网格止损百分比

	// ========== 套利策略 ==========

	// 跨交易所套利
	CrossExchangeArbEnabled bool    `json:"cross_exchange_arb_enabled"` // 跨交易所套利启用
	PriceDiffThreshold      float64 `json:"price_diff_threshold"`       // 价差阈值(%)
	MinArbAmount            float64 `json:"min_arb_amount"`             // 最小套利金额

	// 现货-合约套利
	SpotFutureArbEnabled bool    `json:"spot_future_arb_enabled"` // 现货-合约套利启用
	BasisThreshold       float64 `json:"basis_threshold"`         // 基差阈值(%)
	FundingRateThreshold float64 `json:"funding_rate_threshold"`  // 资金费率阈值(%)

	// 三角套利
	TriangleArbEnabled bool    `json:"triangle_arb_enabled"` // 三角套利启用
	TriangleThreshold  float64 `json:"triangle_threshold"`   // 三角套利阈值(%)
	BaseSymbols        string  `json:"base_symbols"`         // 基础交易对列表(逗号分隔)

	// 统计套利
	StatArbEnabled      bool    `json:"stat_arb_enabled"`     // 统计套利启用
	CointegrationPeriod int     `json:"cointegration_period"` // 协整检验周期(天)
	ZscoreThreshold     float64 `json:"zscore_threshold"`     // Z分数阈值
	StatArbPairs        string  `json:"stat_arb_pairs"`       // 统计套利对列表

	// 期现套利
	FuturesSpotArbEnabled bool    `json:"futures_spot_arb_enabled"` // 期现套利启用
	ExpiryThreshold       int     `json:"expiry_threshold"`         // 到期时间阈值(天)
	SpotFutureSpread      float64 `json:"spot_future_spread"`       // 期现价差阈值(%)

	// ========== 风险控制 ==========

	// 仓位管理
	MaxPositionSize    float64 `json:"max_position_size"`   // 最大仓位大小(%)
	PositionSizeStep   float64 `json:"position_size_step"`  // 仓位调整步长(%)
	DynamicPositioning bool    `json:"dynamic_positioning"` // 动态仓位管理

	// 止损止盈
	EnableStopLoss    bool    `json:"enable_stop_loss"`    // 启用止损
	StopLossPercent   float64 `json:"stop_loss_percent"`   // 止损百分比
	EnableTakeProfit  bool    `json:"enable_take_profit"`  // 启用止盈
	TakeProfitPercent float64 `json:"take_profit_percent"` // 止盈百分比

	// 保证金损失止损
	EnableMarginLossStopLoss  bool    `json:"enable_margin_loss_stop_loss"`  // 启用保证金损失止损
	MarginLossStopLossPercent float64 `json:"margin_loss_stop_loss_percent"` // 保证金损失止损百分比

	// 保证金盈利止盈
	EnableMarginProfitTakeProfit  bool    `json:"enable_margin_profit_take_profit"`  // 启用保证金盈利止盈
	MarginProfitTakeProfitPercent float64 `json:"margin_profit_take_profit_percent"` // 保证金盈利止盈百分比

	// 波动率过滤
	VolatilityFilterEnabled bool    `json:"volatility_filter_enabled"` // 波动率过滤启用
	MaxVolatility           float64 `json:"max_volatility"`            // 最大波动率(%)
	VolatilityPeriod        int     `json:"volatility_period"`         // 波动率计算周期(天)

	// ========== 市场时机 ==========

	// 时间过滤
	TimeFilterEnabled bool `json:"time_filter_enabled"` // 时间过滤启用
	StartHour         int  `json:"start_hour"`          // 开始小时(UTC)
	EndHour           int  `json:"end_hour"`            // 结束小时(UTC)
	WeekendTrading    bool `json:"weekend_trading"`     // 周末交易

	// 市场状态过滤
	MarketRegimeFilterEnabled bool    `json:"market_regime_filter_enabled"` // 市场状态过滤启用
	MarketRegimeThreshold     float64 `json:"market_regime_threshold"`      // 市场状态阈值
	PreferredRegime           string  `json:"preferred_regime"`             // 偏好市场状态(bull/bear/sideways)
}

// 特征数据缓存表
type FeatureCache struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol       string    `gorm:"size:32;not null;uniqueIndex:uk_symbol_time_window" json:"symbol"` // 交易对符号
	Features     string    `gorm:"type:json;not null" json:"features"`                               // 特征数据(JSON格式)
	ComputedAt   time.Time `gorm:"not null;index" json:"computed_at"`                                // 计算时间
	ExpiresAt    time.Time `gorm:"not null;index:idx_expires_cleanup" json:"expires_at"`             // 过期时间
	FeatureCount int       `gorm:"default:0" json:"feature_count"`                                   // 特征数量
	QualityScore float64   `gorm:"type:decimal(3,2);default:0" json:"quality_score"`                 // 质量评分(0-1)
	Source       string    `gorm:"size:32;default:'computed';index" json:"source"`                   // 数据来源(computed/realtime)
	TimeWindow   int       `gorm:"default:24;uniqueIndex:uk_symbol_time_window" json:"time_window"`  // 时间窗口(小时)
	DataPoints   int       `gorm:"default:0" json:"data_points"`                                     // 数据点数量
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (FeatureCache) TableName() string {
	return "feature_cache"
}

// ML模型存储表
type MLModel struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol          string    `gorm:"size:32;not null;uniqueIndex:uk_symbol_model_type" json:"symbol"`     // 交易对符号
	ModelType       string    `gorm:"size:32;not null;uniqueIndex:uk_symbol_model_type" json:"model_type"` // 模型类型(random_forest/gradient_boost/stacking/neural_network)
	ModelName       string    `gorm:"size:128;not null" json:"model_name"`                                 // 模型名称
	ModelData       []byte    `gorm:"type:longblob" json:"model_data"`                                     // 序列化的模型数据
	Performance     string    `gorm:"type:json;not null" json:"performance"`                               // 性能指标(JSON格式)
	TrainedAt       time.Time `gorm:"not null;index" json:"trained_at"`                                    // 训练完成时间
	ExpiresAt       time.Time `gorm:"not null;index:idx_expires_cleanup" json:"expires_at"`                // 过期时间
	TrainingSamples int       `gorm:"not null" json:"training_samples"`                                    // 训练样本数量
	FeatureCount    int       `gorm:"not null" json:"feature_count"`                                       // 特征数量
	Accuracy        float64   `gorm:"type:decimal(5,4);not null;index" json:"accuracy"`                    // 准确率(0-1)
	Precision       float64   `gorm:"type:decimal(5,4);default:0" json:"precision"`                        // 精确率
	Recall          float64   `gorm:"type:decimal(5,4);default:0" json:"recall"`                           // 召回率
	F1Score         float64   `gorm:"type:decimal(5,4);default:0" json:"f1_score"`                         // F1分数
	AUC             float64   `gorm:"type:decimal(5,4);default:0" json:"auc"`                              // AUC值
	SharpeRatio     float64   `gorm:"type:decimal(5,4);default:0" json:"sharpe_ratio"`                     // 夏普比率
	MaxDrawdown     float64   `gorm:"type:decimal(5,4);default:0" json:"max_drawdown"`                     // 最大回撤
	WinRate         float64   `gorm:"type:decimal(5,4);default:0" json:"win_rate"`                         // 胜率
	ProfitFactor    float64   `gorm:"type:decimal(6,4);default:0" json:"profit_factor"`                    // 盈利因子
	Status          string    `gorm:"size:16;default:'active';index" json:"status"`                        // 状态(active/deprecated)
	Version         int       `gorm:"default:1" json:"version"`                                            // 模型版本号
	Description     string    `gorm:"type:text" json:"description,omitempty"`                              // 模型描述
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定表名
func (MLModel) TableName() string {
	return "ml_models"
}

// BinanceExchangeInfo Binance交易对信息
type BinanceExchangeInfo struct {
	ID                         uint   `gorm:"primarykey" json:"id"`
	Symbol                     string `gorm:"size:20;not null;index" json:"symbol"` // 移除uniqueIndex，改为普通index
	Status                     string `gorm:"size:20;not null" json:"status"`
	BaseAsset                  string `gorm:"size:20;not null" json:"base_asset"`
	QuoteAsset                 string `gorm:"size:20;not null" json:"quote_asset"`
	MarketType                 string `gorm:"size:10;not null;default:spot" json:"market_type"` // 新增：市场类型
	BaseAssetPrecision         int    `gorm:"not null" json:"base_asset_precision"`
	QuoteAssetPrecision        int    `gorm:"not null" json:"quote_asset_precision"`
	BaseCommissionPrecision    int    `gorm:"not null" json:"base_commission_precision"`
	QuoteCommissionPrecision   int    `gorm:"not null" json:"quote_commission_precision"`
	OrderTypes                 string `gorm:"type:text" json:"order_types"` // JSON数组
	IcebergAllowed             bool   `gorm:"default:false" json:"iceberg_allowed"`
	OcoAllowed                 bool   `gorm:"default:false" json:"oco_allowed"`
	QuoteOrderQtyMarketAllowed bool   `gorm:"default:false" json:"quote_order_qty_market_allowed"`
	AllowTrailingStop          bool   `gorm:"default:false" json:"allow_trailing_stop"`
	CancelReplaceAllowed       bool   `gorm:"default:false" json:"cancel_replace_allowed"`
	IsSpotTradingAllowed       bool   `gorm:"default:true" json:"is_spot_trading_allowed"`
	IsMarginTradingAllowed     bool   `gorm:"default:false" json:"is_margin_trading_allowed"`
	Filters                    string `gorm:"type:text" json:"filters"`     // JSON格式
	Permissions                string `gorm:"type:text" json:"permissions"` // JSON数组

	// 状态管理字段 - 软删除支持
	IsActive       bool       `gorm:"default:true;index" json:"is_active"` // 是否活跃交易对
	DeactivatedAt  *time.Time `gorm:"index" json:"deactivated_at"`         // 下架时间
	LastSeenActive *time.Time `gorm:"index" json:"last_seen_active"`       // 最后一次活跃时间

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (BinanceExchangeInfo) TableName() string {
	return "binance_exchange_info"
}

// BinanceFuturesContract 期货合约信息
type BinanceFuturesContract struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	Symbol             string    `gorm:"size:20;not null;uniqueIndex" json:"symbol"`
	Status             string    `gorm:"size:20;not null" json:"status"`
	ContractType       string    `gorm:"size:50" json:"contract_type"`
	BaseAsset          string    `gorm:"size:20;not null" json:"base_asset"`
	QuoteAsset         string    `gorm:"size:20;not null" json:"quote_asset"`
	MarginAsset        string    `gorm:"size:20" json:"margin_asset"`
	PricePrecision     int       `json:"price_precision"`
	QuantityPrecision  int       `json:"quantity_precision"`
	BaseAssetPrecision int       `json:"base_asset_precision"`
	QuotePrecision     int       `json:"quote_precision"`
	UnderlyingType     string    `gorm:"size:20" json:"underlying_type"`
	UnderlyingSubType  string    `gorm:"type:text" json:"underlying_sub_type"` // JSON array
	SettlePlan         int       `json:"settle_plan"`
	TriggerProtect     float64   `gorm:"type:decimal(5,4)" json:"trigger_protect"`
	Filters            string    `gorm:"type:text" json:"filters"`
	OrderTypes         string    `gorm:"type:text" json:"order_types"`
	TimeInForce        string    `gorm:"type:text" json:"time_in_force"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TableName 指定表名
func (BinanceFuturesContract) TableName() string {
	return "binance_futures_contracts"
}

// BinanceFundingRate 资金费率
type BinanceFundingRate struct {
	ID                   uint      `gorm:"primarykey" json:"id"`
	Symbol               string    `gorm:"size:20;not null" json:"symbol"`
	FundingRate          float64   `gorm:"type:decimal(10,8);not null" json:"funding_rate"`
	FundingTime          int64     `gorm:"not null" json:"funding_time"`
	MarkPrice            float64   `gorm:"type:decimal(20,8)" json:"mark_price"`
	IndexPrice           float64   `gorm:"type:decimal(20,8)" json:"index_price"`
	EstimatedSettlePrice float64   `gorm:"type:decimal(20,8)" json:"estimated_settle_price"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (BinanceFundingRate) TableName() string {
	return "binance_funding_rates"
}

// BinanceOrderBookDepth 订单簿深度
type BinanceOrderBookDepth struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Symbol       string    `gorm:"size:20;not null" json:"symbol"`
	MarketType   string    `gorm:"size:10;not null" json:"market_type"`
	LastUpdateID int64     `gorm:"not null" json:"last_update_id"`
	Bids         string    `gorm:"type:text;not null" json:"bids"`
	Asks         string    `gorm:"type:text;not null" json:"asks"`
	SnapshotTime int64     `json:"snapshot_time"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名
func (BinanceOrderBookDepth) TableName() string {
	return "binance_order_book_depth"
}

// BinanceTrade 实时交易数据
type BinanceTrade struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Symbol       string    `gorm:"size:20;not null;index:idx_symbol_trade_time,priority:1" json:"symbol"`
	MarketType   string    `gorm:"size:10;not null;index:idx_symbol_trade_time,priority:2" json:"market_type"`
	TradeID      int64     `gorm:"not null;uniqueIndex:uniq_trade,priority:1" json:"trade_id"`
	Price        string    `gorm:"size:32;not null" json:"price"`
	Quantity     string    `gorm:"size:32;not null" json:"quantity"`
	TradeTime    int64     `gorm:"not null;index:idx_symbol_trade_time,priority:3;uniqueIndex:uniq_trade,priority:2" json:"trade_time"`
	IsBuyerMaker bool      `gorm:"not null" json:"is_buyer_maker"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名
func (BinanceTrade) TableName() string {
	return "binance_trades"
}

// Binance24hStats 24小时统计数据
type Binance24hStats struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	Symbol             string    `gorm:"size:20;not null" json:"symbol"`
	MarketType         string    `gorm:"size:10;not null" json:"market_type"`
	PriceChange        float64   `gorm:"type:decimal(20,8)" json:"price_change"`
	PriceChangePercent float64   `gorm:"type:decimal(10,4)" json:"price_change_percent"`
	WeightedAvgPrice   float64   `gorm:"type:decimal(20,8)" json:"weighted_avg_price"`
	PrevClosePrice     float64   `gorm:"type:decimal(20,8)" json:"prev_close_price"`
	LastPrice          float64   `gorm:"type:decimal(20,8)" json:"last_price"`
	LastQty            float64   `gorm:"type:decimal(20,8)" json:"last_qty"` // 最后交易数量
	BidPrice           float64   `gorm:"type:decimal(20,8)" json:"bid_price"`
	BidQty             float64   `gorm:"type:decimal(20,8)" json:"bid_qty"` // 买一档数量
	AskPrice           float64   `gorm:"type:decimal(20,8)" json:"ask_price"`
	AskQty             float64   `gorm:"type:decimal(20,8)" json:"ask_qty"` // 卖一档数量
	OpenPrice          float64   `gorm:"type:decimal(20,8)" json:"open_price"`
	HighPrice          float64   `gorm:"type:decimal(20,8)" json:"high_price"`
	LowPrice           float64   `gorm:"type:decimal(20,8)" json:"low_price"`
	Volume             float64   `gorm:"type:decimal(30,8)" json:"volume"`
	QuoteVolume        float64   `gorm:"type:decimal(30,8)" json:"quote_volume"`
	OpenTime           int64     `json:"open_time"`
	CloseTime          int64     `json:"close_time"`
	FirstId            int64     `json:"first_id"` // 第一笔交易ID
	LastId             int64     `json:"last_id"`  // 最后一笔交易ID
	Count              int64     `json:"count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Binance24hStats) TableName() string {
	return "binance_24h_stats"
}

// Binance24hStatsHistory 24小时统计数据历史表 - 存储完整时间序列
type Binance24hStatsHistory struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	Symbol     string `gorm:"size:20;not null" json:"symbol"`
	MarketType string `gorm:"size:10;not null" json:"market_type"`
	// 时间窗口标识 - 核心新增字段
	WindowStart    time.Time `gorm:"not null;index" json:"window_start"`           // 时间窗口开始时间
	WindowEnd      time.Time `gorm:"not null" json:"window_end"`                   // 时间窗口结束时间
	WindowDuration int       `gorm:"not null;default:3600" json:"window_duration"` // 窗口持续时间(秒，默认1小时)
	// 完整统计数据 - 与实时表相同
	PriceChange        float64   `gorm:"type:decimal(20,8)" json:"price_change"`
	PriceChangePercent float64   `gorm:"type:decimal(10,4)" json:"price_change_percent"`
	WeightedAvgPrice   float64   `gorm:"type:decimal(20,8)" json:"weighted_avg_price"`
	PrevClosePrice     float64   `gorm:"type:decimal(20,8)" json:"prev_close_price"`
	LastPrice          float64   `gorm:"type:decimal(20,8)" json:"last_price"`
	LastQty            float64   `gorm:"type:decimal(20,8)" json:"last_qty"`
	BidPrice           float64   `gorm:"type:decimal(20,8)" json:"bid_price"`
	BidQty             float64   `gorm:"type:decimal(20,8)" json:"bid_qty"`
	AskPrice           float64   `gorm:"type:decimal(20,8)" json:"ask_price"`
	AskQty             float64   `gorm:"type:decimal(20,8)" json:"ask_qty"`
	OpenPrice          float64   `gorm:"type:decimal(20,8)" json:"open_price"`
	HighPrice          float64   `gorm:"type:decimal(20,8)" json:"high_price"`
	LowPrice           float64   `gorm:"type:decimal(20,8)" json:"low_price"`
	Volume             float64   `gorm:"type:decimal(30,8)" json:"volume"`
	QuoteVolume        float64   `gorm:"type:decimal(30,8)" json:"quote_volume"`
	OpenTime           int64     `json:"open_time"`
	CloseTime          int64     `json:"close_time"`
	FirstId            int64     `json:"first_id"`
	LastId             int64     `json:"last_id"`
	Count              int64     `json:"count"`
	CreatedAt          time.Time `json:"created_at"`
}

// TableName 指定历史表表名
func (Binance24hStatsHistory) TableName() string {
	return "binance_24h_stats_history"
}

// 过滤器修正记录 - 记录API数据修正情况，用于监控和分析
type FilterCorrection struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Symbol   string `gorm:"size:32;index;not null" json:"symbol"` // 交易对
	Exchange string `gorm:"size:32;not null;default:'binance'" json:"exchange"`

	// 原始API数据
	OriginalStepSize    float64 `gorm:"type:decimal(20,10)" json:"original_step_size"`
	OriginalMinNotional float64 `gorm:"type:decimal(20,8)" json:"original_min_notional"`
	OriginalMaxQty      float64 `gorm:"type:decimal(30,8)" json:"original_max_qty"`
	OriginalMinQty      float64 `gorm:"type:decimal(20,8)" json:"original_min_qty"`

	// 修正后的数据
	CorrectedStepSize    float64 `gorm:"type:decimal(20,10)" json:"corrected_step_size"`
	CorrectedMinNotional float64 `gorm:"type:decimal(20,8)" json:"corrected_min_notional"`
	CorrectedMaxQty      float64 `gorm:"type:decimal(30,8)" json:"corrected_max_qty"`
	CorrectedMinQty      float64 `gorm:"type:decimal(20,8)" json:"corrected_min_qty"`

	// 修正原因和类型
	CorrectionType   string `gorm:"size:64;not null" json:"correction_type"`           // step_size, min_notional, range_check, etc.
	CorrectionReason string `gorm:"size:256" json:"correction_reason"`                 // 详细修正原因
	IsSmallCapSymbol bool   `gorm:"not null;default:false" json:"is_small_cap_symbol"` // 是否为小币种

	// 统计和分析
	CorrectionCount int       `gorm:"not null;default:1" json:"correction_count"` // 该交易对被修正次数
	LastCorrectedAt time.Time `gorm:"index" json:"last_corrected_at"`             // 最后修正时间

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (FilterCorrection) TableName() string {
	return "filter_corrections"
}
