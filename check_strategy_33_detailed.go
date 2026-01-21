package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TradingStrategy 结构体（简化版）
type TradingStrategy struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Conditions  string    `json:"conditions"` // JSON字符串
	IsRunning   bool      `json:"is_running"`
	LastRunAt   *time.Time `json:"last_run_at"`
	RunInterval int       `json:"run_interval"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StrategyConditions 策略条件结构体
type StrategyConditions struct {
	// 基础条件
	SpotContract bool   `json:"spot_contract"`
	TradingType  string `json:"trading_type"`

	// 交易配置
	MaxInvestment     float64 `json:"max_investment"`
	PerOrderAmount    float64 `json:"per_order_amount"`
	MinOrderAmount    float64 `json:"min_order_amount"`
	MaxOpenOrders     int     `json:"max_open_orders"`
	Leverage          int     `json:"leverage"`
	MarginMode        string  `json:"margin_mode"`
	ProfitScalingMode bool    `json:"profit_scaling_mode"`

	// 市场条件
	MarketCapLimit        float64 `json:"market_cap_limit"`
	VolumeLimit           float64 `json:"volume_limit"`
	PriceChangePercent    float64 `json:"price_change_percent"`
	MinPriceChangePercent float64 `json:"min_price_change_percent"`
	MaxPriceChangePercent float64 `json:"max_price_change_percent"`

	// 技术指标
	RSIPeriod              int     `json:"rsi_period"`
	RSIOverbought          float64 `json:"rsi_overbought"`
	RSIOversold            float64 `json:"rsi_oversold"`
	MACDShortPeriod        int     `json:"macd_short_period"`
	MACDLongPeriod         int     `json:"macd_long_period"`
	MACDSignalPeriod       int     `json:"macd_signal_period"`
	BollingerPeriod        int     `json:"bollinger_period"`
	BollingerDeviation     float64 `json:"bollinger_deviation"`
	MAFastPeriod           int     `json:"ma_fast_period"`
	MASlowPeriod           int     `json:"ma_slow_period"`
	MAType                 string  `json:"ma_type"`
	TrendStrengthThreshold float64 `json:"trend_strength_threshold"`

	// 止损止盈
	StopLossPercent     float64 `json:"stop_loss_percent"`
	TakeProfitPercent   float64 `json:"take_profit_percent"`
	TrailingStopEnabled bool    `json:"trailing_stop_enabled"`
	TrailingStopPercent float64 `json:"trailing_stop_percent"`

	// 特殊过滤器
	NoShortBelowMarketCap bool    `json:"no_short_below_market_cap"`
	MarketCapLimitShort   float64 `json:"market_cap_limit_short"`
	ShortOnGainers        bool    `json:"short_on_gainers"`
	LongOnDippers         bool    `json:"long_on_dippers"`

	// 资金费率过滤
	FundingRateFilterEnabled bool    `json:"funding_rate_filter_enabled"`
	MinFundingRate           float64 `json:"min_funding_rate"`
	MaxFundingRate           float64 `json:"max_funding_rate"`

	// 合约排名过滤
	FuturesPriceRankFilterEnabled bool `json:"futures_price_rank_filter_enabled"`
	MaxFuturesPriceRank           int  `json:"max_futures_price_rank"`

	// 波动率过滤
	VolatilityFilterEnabled bool    `json:"volatility_filter_enabled"`
	MinVolatility           float64 `json:"min_volatility"`
	MaxVolatility           float64 `json:"max_volatility"`

	// 时间过滤
	TimeFilterEnabled bool   `json:"time_filter_enabled"`
	StartHour         int    `json:"start_hour"`
	EndHour           int    `json:"end_hour"`
	TradingDays       string `json:"trading_days"`

	// 高级策略
	StrategyType             string  `json:"strategy_type"`
	MeanReversionEnabled     bool    `json:"mean_reversion_enabled"`
	GridTradingEnabled       bool    `json:"grid_trading_enabled"`
	MomentumEnabled          bool    `json:"momentum_enabled"`
	ScalpingEnabled          bool    `json:"scalping_enabled"`
	ArbitrageEnabled         bool    `json:"arbitrage_enabled"`
	FundingRateArbitrageMode string  `json:"funding_rate_arbitrage_mode"`

	// 网格交易参数
	GridLevels         int     `json:"grid_levels"`
	GridSpacingPercent float64 `json:"grid_spacing_percent"`
	GridProfitPercent  float64 `json:"grid_profit_percent"`
	GridMaxInvestment  float64 `json:"grid_max_investment"`

	// 均值回归参数
	MeanReversionThreshold float64 `json:"mean_reversion_threshold"`
	MeanReversionPeriod    int     `json:"mean_reversion_period"`

	// 动量策略参数
	MomentumPeriod          int     `json:"momentum_period"`
	MomentumThreshold       float64 `json:"momentum_threshold"`
	MomentumStrengthEnabled bool    `json:"momentum_strength_enabled"`

	// 策略权重
	TechnicalWeight    float64 `json:"technical_weight"`
	FundamentalWeight  float64 `json:"fundamental_weight"`
	MarketSentimentWeight float64 `json:"market_sentiment_weight"`
	RiskWeight         float64 `json:"risk_weight"`
}

func main() {
	fmt.Println("=== 详细分析策略ID 33的配置 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 查询策略基本信息
	var strategy TradingStrategy
	result := db.Where("id = ?", 33).First(&strategy)
	if result.Error != nil {
		log.Fatalf("查询策略失败: %v", result.Error)
	}

	fmt.Printf("=== 策略基本信息 ===\n")
	fmt.Printf("ID: %d\n", strategy.ID)
	fmt.Printf("用户ID: %d\n", strategy.UserID)
	fmt.Printf("策略名称: %s\n", strategy.Name)
	fmt.Printf("策略描述: %s\n", strategy.Description)
	fmt.Printf("运行状态: %v\n", strategy.IsRunning)
	fmt.Printf("运行间隔: %d 分钟\n", strategy.RunInterval)
	if strategy.LastRunAt != nil {
		fmt.Printf("最后运行时间: %v\n", strategy.LastRunAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("最后运行时间: 从未运行\n")
	}
	fmt.Printf("创建时间: %v\n", strategy.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("更新时间: %v\n", strategy.UpdatedAt.Format("2006-01-02 15:04:05"))

	// 解析策略条件
	fmt.Printf("\n=== 策略条件分析 ===\n")
	if strategy.Conditions == "" {
		fmt.Printf("⚠️ 策略条件为空，可能需要检查数据库或配置\n")

		// 尝试从其他表获取相关信息
		fmt.Printf("\n🔍 尝试从其他表获取策略信息...\n")

		// 检查是否有相关的网格策略配置
		var gridConfig map[string]interface{}
		gridQuery := "SELECT * FROM grid_trading_configs WHERE strategy_id = ?"
		db.Raw(gridQuery, 33).Scan(&gridConfig)

		if len(gridConfig) > 0 {
			fmt.Printf("📊 发现网格交易配置:\n")
			for k, v := range gridConfig {
				fmt.Printf("  %s: %v\n", k, v)
			}
		} else {
			fmt.Printf("❌ 未找到网格交易配置\n")
		}

		// 检查是否有均值回归配置
		var meanReversionConfig map[string]interface{}
		mrQuery := "SELECT * FROM mean_reversion_configs WHERE strategy_id = ?"
		db.Raw(mrQuery, 33).Scan(&meanReversionConfig)

		if len(meanReversionConfig) > 0 {
			fmt.Printf("📊 发现均值回归配置:\n")
			for k, v := range meanReversionConfig {
				fmt.Printf("  %s: %v\n", k, v)
			}
		} else {
			fmt.Printf("❌ 未找到均值回归配置\n")
		}

		return
	}

	var conditions StrategyConditions
	if err := json.Unmarshal([]byte(strategy.Conditions), &conditions); err != nil {
		fmt.Printf("解析策略条件失败: %v\n", err)
		fmt.Printf("原始条件JSON: %s\n", strategy.Conditions)
		return
	}

	analyzeStrategyConditions(conditions)

	// 检查策略执行记录
	fmt.Printf("\n=== 策略执行历史 ===\n")
	var executionCount int64
	db.Model(&struct{}{}).Table("strategy_executions").Where("strategy_id = ?", 33).Count(&executionCount)
	fmt.Printf("总执行次数: %d\n", executionCount)

	if executionCount > 0 {
		var executions []map[string]interface{}
		execQuery := `
			SELECT id, status, start_time, end_time, duration, total_orders, success_orders,
				   total_pnl, win_rate, pnl_percentage, total_investment, current_value,
				   error_message
			FROM strategy_executions
			WHERE strategy_id = ?
			ORDER BY created_at DESC LIMIT 5
		`
		db.Raw(execQuery, 33).Scan(&executions)

		fmt.Printf("\n最近5次执行记录:\n")
		for _, exec := range executions {
			fmt.Printf("执行ID: %v\n", exec["id"])
			fmt.Printf("  状态: %v\n", exec["status"])
			fmt.Printf("  开始时间: %v\n", exec["start_time"])
			if exec["end_time"] != nil {
				fmt.Printf("  结束时间: %v\n", exec["end_time"])
			}
			fmt.Printf("  执行时长: %v 秒\n", exec["duration"])
			fmt.Printf("  总订单数: %v\n", exec["total_orders"])
			fmt.Printf("  成功订单数: %v\n", exec["success_orders"])
			fmt.Printf("  总盈亏: %.4f\n", exec["total_pnl"])
			fmt.Printf("  胜率: %.2f%%\n", exec["win_rate"])
			fmt.Printf("  盈亏百分比: %.4f%%\n", exec["pnl_percentage"])
			fmt.Printf("  总投资: %.4f\n", exec["total_investment"])
			fmt.Printf("  当前价值: %.4f\n", exec["current_value"])
			if exec["error_message"] != nil && exec["error_message"] != "" {
				fmt.Printf("  错误信息: %v\n", exec["error_message"])
			}
			fmt.Println()
		}
	}

	// 检查相关的订单记录
	fmt.Printf("\n=== 相关订单统计 ===\n")
	var orderStats map[string]interface{}
	orderQuery := `
		SELECT
			COUNT(*) as total_orders,
			SUM(CASE WHEN status = 'filled' THEN 1 ELSE 0 END) as filled_orders,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_orders,
			SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END) as open_orders,
			SUM(CASE WHEN side = 'BUY' THEN 1 ELSE 0 END) as buy_orders,
			SUM(CASE WHEN side = 'SELL' THEN 1 ELSE 0 END) as sell_orders
		FROM orders
		WHERE strategy_id = ?
	`
	db.Raw(orderQuery, 33).Scan(&orderStats)

	if orderStats["total_orders"] != nil {
		fmt.Printf("总订单数: %v\n", orderStats["total_orders"])
		fmt.Printf("已成交订单: %v\n", orderStats["filled_orders"])
		fmt.Printf("已取消订单: %v\n", orderStats["cancelled_orders"])
		fmt.Printf("未成交订单: %v\n", orderStats["open_orders"])
		fmt.Printf("买入订单: %v\n", orderStats["buy_orders"])
		fmt.Printf("卖出订单: %v\n", orderStats["sell_orders"])
	}

	// 检查调度记录
	fmt.Printf("\n=== 调度记录统计 ===\n")
	var scheduleStats map[string]interface{}
	scheduleQuery := `
		SELECT
			COUNT(*) as total_scheduled,
			SUM(CASE WHEN status = 'executed' THEN 1 ELSE 0 END) as executed_orders,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_orders,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_scheduled
		FROM scheduled_orders
		WHERE strategy_id = ?
	`
	db.Raw(scheduleQuery, 33).Scan(&scheduleStats)

	if scheduleStats["total_scheduled"] != nil {
		fmt.Printf("总调度订单数: %v\n", scheduleStats["total_scheduled"])
		fmt.Printf("已执行调度订单: %v\n", scheduleStats["executed_orders"])
		fmt.Printf("待执行调度订单: %v\n", scheduleStats["pending_orders"])
		fmt.Printf("已取消调度订单: %v\n", scheduleStats["cancelled_scheduled"])
	}
}

func analyzeStrategyConditions(conditions StrategyConditions) {
	fmt.Printf("📊 交易类型: %s\n", conditions.TradingType)
	fmt.Printf("🏪 合约要求: %v\n", conditions.SpotContract)
	fmt.Printf("⚡ 杠杆倍数: %d\n", conditions.Leverage)
	fmt.Printf("💰 保证金模式: %s\n", conditions.MarginMode)

	fmt.Printf("\n💵 资金配置:\n")
	fmt.Printf("  最大投资: %.2f U\n", conditions.MaxInvestment)
	fmt.Printf("  每单金额: %.2f U\n", conditions.PerOrderAmount)
	fmt.Printf("  最小订单: %.2f U\n", conditions.MinOrderAmount)
	fmt.Printf("  最大开仓订单: %d\n", conditions.MaxOpenOrders)
	fmt.Printf("  利润缩放模式: %v\n", conditions.ProfitScalingMode)

	fmt.Printf("\n📈 市场条件:\n")
	fmt.Printf("  市值限制: %.0f 万U\n", conditions.MarketCapLimit)
	fmt.Printf("  成交量限制: %.0f\n", conditions.VolumeLimit)
	fmt.Printf("  价格变动范围: %.2f%% ~ %.2f%%\n", conditions.MinPriceChangePercent, conditions.MaxPriceChangePercent)

	fmt.Printf("\n📊 技术指标:\n")
	if conditions.RSIPeriod > 0 {
		fmt.Printf("  RSI周期: %d, 超买: %.1f, 超卖: %.1f\n",
			conditions.RSIPeriod, conditions.RSIOverbought, conditions.RSIOversold)
	}
	if conditions.MACDShortPeriod > 0 {
		fmt.Printf("  MACD参数: 短期%d, 长期%d, 信号%d\n",
			conditions.MACDShortPeriod, conditions.MACDLongPeriod, conditions.MACDSignalPeriod)
	}
	if conditions.BollingerPeriod > 0 {
		fmt.Printf("  布林带: 周期%d, 偏差%.1f\n",
			conditions.BollingerPeriod, conditions.BollingerDeviation)
	}
	if conditions.MAFastPeriod > 0 {
		fmt.Printf("  移动平均: 快线%d, 慢线%d, 类型:%s\n",
			conditions.MAFastPeriod, conditions.MASlowPeriod, conditions.MAType)
	}

	fmt.Printf("\n🛡️ 风险管理:\n")
	fmt.Printf("  止损百分比: %.2f%%\n", conditions.StopLossPercent)
	fmt.Printf("  止盈百分比: %.2f%%\n", conditions.TakeProfitPercent)
	if conditions.TrailingStopEnabled {
		fmt.Printf("  追踪止损: 启用, 百分比: %.2f%%\n", conditions.TrailingStopPercent)
	}

	fmt.Printf("\n🎯 特殊过滤器:\n")
	if conditions.NoShortBelowMarketCap {
		fmt.Printf("  开空市值限制: %.0f 万U以下不开空\n", conditions.MarketCapLimitShort)
	}
	fmt.Printf("  涨幅开空: %v\n", conditions.ShortOnGainers)
	fmt.Printf("  跌幅开多: %v\n", conditions.LongOnDippers)

	if conditions.FundingRateFilterEnabled {
		fmt.Printf("  资金费率过滤: %.4f%% ~ %.4f%%\n",
			conditions.MinFundingRate, conditions.MaxFundingRate)
	}

	if conditions.FuturesPriceRankFilterEnabled {
		fmt.Printf("  合约排名过滤: 前 %d 名\n", conditions.MaxFuturesPriceRank)
	}

	if conditions.VolatilityFilterEnabled {
		fmt.Printf("  波动率过滤: %.2f%% ~ %.2f%%\n",
			conditions.MinVolatility, conditions.MaxVolatility)
	}

	if conditions.TimeFilterEnabled {
		fmt.Printf("  时间过滤: %d:00 - %d:00\n", conditions.StartHour, conditions.EndHour)
		if conditions.TradingDays != "" {
			fmt.Printf("  交易日: %s\n", conditions.TradingDays)
		}
	}

	fmt.Printf("\n🚀 策略类型: %s\n", conditions.StrategyType)
	fmt.Printf("  均值回归: %v\n", conditions.MeanReversionEnabled)
	fmt.Printf("  网格交易: %v\n", conditions.GridTradingEnabled)
	fmt.Printf("  动量策略: %v\n", conditions.MomentumEnabled)
	fmt.Printf("  剥头皮: %v\n", conditions.ScalpingEnabled)
	fmt.Printf("  套利: %v\n", conditions.ArbitrageEnabled)

	if conditions.GridTradingEnabled {
		fmt.Printf("\n📊 网格交易参数:\n")
		fmt.Printf("  网格层数: %d\n", conditions.GridLevels)
		fmt.Printf("  网格间距: %.2f%%\n", conditions.GridSpacingPercent)
		fmt.Printf("  网格利润: %.2f%%\n", conditions.GridProfitPercent)
		fmt.Printf("  网格最大投资: %.2f U\n", conditions.GridMaxInvestment)
	}

	if conditions.MeanReversionEnabled {
		fmt.Printf("\n📊 均值回归参数:\n")
		fmt.Printf("  阈值: %.2f\n", conditions.MeanReversionThreshold)
		fmt.Printf("  周期: %d\n", conditions.MeanReversionPeriod)
	}

	if conditions.MomentumEnabled {
		fmt.Printf("\n📊 动量策略参数:\n")
		fmt.Printf("  周期: %d\n", conditions.MomentumPeriod)
		fmt.Printf("  阈值: %.2f\n", conditions.MomentumThreshold)
		fmt.Printf("  强度启用: %v\n", conditions.MomentumStrengthEnabled)
	}

	fmt.Printf("\n⚖️ 策略权重:\n")
	fmt.Printf("  技术指标权重: %.2f\n", conditions.TechnicalWeight)
	fmt.Printf("  基本面权重: %.2f\n", conditions.FundamentalWeight)
	fmt.Printf("  市场情绪权重: %.2f\n", conditions.MarketSentimentWeight)
	fmt.Printf("  风险权重: %.2f\n", conditions.RiskWeight)
}