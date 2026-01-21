package server

import (
	"time"
)

// ExecutionPlan 详细执行计划
type ExecutionPlan struct {
	Symbol        string  `json:"symbol"`
	StrategyType  string  `json:"strategy_type"`
	TotalPosition float64 `json:"total_position"` // 总仓位百分比
	CurrentPrice  float64 `json:"current_price"`  // 当前价格

	// 建仓计划
	EntryPlan []EntryStage `json:"entry_plan"`

	// 出场计划
	ExitPlan []ExitStage `json:"exit_plan"`

	// 风险控制
	RiskControls RiskControls `json:"risk_controls"`

	// 监控指标
	MonitoringMetrics []string `json:"monitoring_metrics"`

	// 调整条件
	AdjustmentTriggers []AdjustmentTrigger `json:"adjustment_triggers"`

	// 执行时间表
	Timeline ExecutionTimeline `json:"timeline"`
}

// EntryStage 建仓阶段
type EntryStage struct {
	StageNumber int        `json:"stage_number"` // 第几批
	Percentage  float64    `json:"percentage"`   // 仓位百分比 (相对于总仓位)
	PriceRange  PriceRange `json:"price_range"`  // 价格区间
	Condition   string     `json:"condition"`    // 执行条件
	MaxSlippage float64    `json:"max_slippage"` // 最大滑点
	TimeLimit   string     `json:"time_limit"`   // 时间限制
	Priority    string     `json:"priority"`     // 优先级: "high", "medium", "low"
}

// ExitStage 出场阶段
type ExitStage struct {
	StageNumber     int        `json:"stage_number"`
	Percentage      float64    `json:"percentage"`
	PriceRange      PriceRange `json:"price_range"`
	Condition       string     `json:"condition"`
	ProfitTarget    float64    `json:"profit_target"`     // 利润目标
	RiskRewardRatio float64    `json:"risk_reward_ratio"` // 风险收益比
}

// RiskControls 风险控制
type RiskControls struct {
	MaxLossPerTrade          float64 `json:"max_loss_per_trade"`         // 单笔最大亏损
	MaxDailyLoss             float64 `json:"max_daily_loss"`             // 单日最大亏损
	MaxHoldingTime           string  `json:"max_holding_time"`           // 最大持有时间
	TrailingStop             bool    `json:"trailing_stop"`              // 是否使用追踪止损
	TrailingStopPercent      float64 `json:"trailing_stop_percent"`      // 追踪止损百分比
	PositionCorrelationLimit float64 `json:"position_correlation_limit"` // 仓位相关性限制
}

// AdjustmentTrigger 调整触发条件
type AdjustmentTrigger struct {
	Condition string  `json:"condition"` // 触发条件
	Action    string  `json:"action"`    // 执行动作
	Threshold float64 `json:"threshold"` // 阈值
	Priority  string  `json:"priority"`  // 优先级
}

// ExecutionTimeline 执行时间表
type ExecutionTimeline struct {
	StartTime        time.Time   `json:"start_time"`
	ExpectedDuration string      `json:"expected_duration"` // 预期持续时间
	KeyMilestones    []Milestone `json:"key_milestones"`    // 关键里程碑
	RiskCheckPoints  []string    `json:"risk_check_points"` // 风险检查点
}

// Milestone 里程碑
type Milestone struct {
	Time        string `json:"time"`        // 时间点
	Event       string `json:"event"`       // 事件
	Description string `json:"description"` // 描述
}

// PriceRange 价格区间
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// generateDetailedExecutionPlan 生成详细的执行计划
func (s *Server) generateDetailedExecutionPlan(rec CoinRecommendation, strategyType string, currentPrice float64, expectedReturn float64, recommendedPosition float64) *ExecutionPlan {
	plan := &ExecutionPlan{
		Symbol:             rec.Symbol,
		StrategyType:       strategyType,
		TotalPosition:      recommendedPosition,
		CurrentPrice:       currentPrice,
		EntryPlan:          s.generateEntryPlan(strategyType, currentPrice, recommendedPosition),
		ExitPlan:           s.generateExitPlan(strategyType, currentPrice, expectedReturn, recommendedPosition),
		RiskControls:       s.generateRiskControls(strategyType, recommendedPosition),
		MonitoringMetrics:  s.generateMonitoringMetrics(),
		AdjustmentTriggers: s.generateAdjustmentTriggers(strategyType),
		Timeline:           s.generateExecutionTimeline(strategyType),
	}

	return plan
}

// generateEntryPlan 生成建仓计划
func (s *Server) generateEntryPlan(strategyType string, currentPrice float64, totalPosition float64) []EntryStage {
	var stages []EntryStage

	switch strategyType {
	case "LONG":
		// 多头策略：分3批建仓，优先级递减
		stages = []EntryStage{
			{
				StageNumber: 1,
				Percentage:  0.4, // 40%仓位
				PriceRange: PriceRange{
					Min: currentPrice * 0.98,
					Max: currentPrice * 1.00,
					Avg: currentPrice * 0.99,
				},
				Condition:   "价格回调至支撑位附近",
				MaxSlippage: 0.005,
				TimeLimit:   "24小时内",
				Priority:    "high",
			},
			{
				StageNumber: 2,
				Percentage:  0.4, // 40%仓位
				PriceRange: PriceRange{
					Min: currentPrice * 1.00,
					Max: currentPrice * 1.02,
					Avg: currentPrice * 1.01,
				},
				Condition:   "价格突破前期高点",
				MaxSlippage: 0.008,
				TimeLimit:   "48小时内",
				Priority:    "medium",
			},
			{
				StageNumber: 3,
				Percentage:  0.2, // 20%仓位
				PriceRange: PriceRange{
					Min: currentPrice * 1.02,
					Max: currentPrice * 1.05,
					Avg: currentPrice * 1.035,
				},
				Condition:   "强势突破确认",
				MaxSlippage: 0.01,
				TimeLimit:   "72小时内",
				Priority:    "low",
			},
		}

	case "SHORT":
		// 空头策略：分3批建仓
		stages = []EntryStage{
			{
				StageNumber: 1,
				Percentage:  0.4,
				PriceRange: PriceRange{
					Min: currentPrice * 1.00,
					Max: currentPrice * 1.02,
					Avg: currentPrice * 1.01,
				},
				Condition:   "价格触及阻力位",
				MaxSlippage: 0.005,
				TimeLimit:   "24小时内",
				Priority:    "high",
			},
			{
				StageNumber: 2,
				Percentage:  0.4,
				PriceRange: PriceRange{
					Min: currentPrice * 0.98,
					Max: currentPrice * 1.00,
					Avg: currentPrice * 0.99,
				},
				Condition:   "价格跌破支撑位",
				MaxSlippage: 0.008,
				TimeLimit:   "48小时内",
				Priority:    "medium",
			},
			{
				StageNumber: 3,
				Percentage:  0.2,
				PriceRange: PriceRange{
					Min: currentPrice * 0.95,
					Max: currentPrice * 0.98,
					Avg: currentPrice * 0.965,
				},
				Condition:   "恐慌性抛售确认",
				MaxSlippage: 0.01,
				TimeLimit:   "72小时内",
				Priority:    "low",
			},
		}

	default: // RANGE
		// 震荡策略：分2批建仓
		stages = []EntryStage{
			{
				StageNumber: 1,
				Percentage:  0.6,
				PriceRange: PriceRange{
					Min: currentPrice * 0.97,
					Max: currentPrice * 1.00,
					Avg: currentPrice * 0.985,
				},
				Condition:   "价格接近布林带下轨",
				MaxSlippage: 0.003,
				TimeLimit:   "24小时内",
				Priority:    "medium",
			},
			{
				StageNumber: 2,
				Percentage:  0.4,
				PriceRange: PriceRange{
					Min: currentPrice * 1.00,
					Max: currentPrice * 1.03,
					Avg: currentPrice * 1.015,
				},
				Condition:   "价格接近布林带上轨",
				MaxSlippage: 0.005,
				TimeLimit:   "48小时内",
				Priority:    "low",
			},
		}
	}

	return stages
}

// generateExitPlan 生成出场计划
func (s *Server) generateExitPlan(strategyType string, currentPrice float64, expectedReturn float64, totalPosition float64) []ExitStage {
	var stages []ExitStage

	switch strategyType {
	case "LONG":
		// 多头策略出场计划
		targetPrice1 := currentPrice * (1.0 + expectedReturn*0.5)
		targetPrice2 := currentPrice * (1.0 + expectedReturn*0.8)
		targetPrice3 := currentPrice * (1.0 + expectedReturn)

		stages = []ExitStage{
			{
				StageNumber: 1,
				Percentage:  0.3,
				PriceRange: PriceRange{
					Min: targetPrice1 * 0.98,
					Max: targetPrice1 * 1.02,
					Avg: targetPrice1,
				},
				Condition:       "达到第一利润目标",
				ProfitTarget:    expectedReturn * 0.5,
				RiskRewardRatio: 1.5,
			},
			{
				StageNumber: 2,
				Percentage:  0.4,
				PriceRange: PriceRange{
					Min: targetPrice2 * 0.98,
					Max: targetPrice2 * 1.02,
					Avg: targetPrice2,
				},
				Condition:       "达到第二利润目标",
				ProfitTarget:    expectedReturn * 0.8,
				RiskRewardRatio: 2.0,
			},
			{
				StageNumber: 3,
				Percentage:  0.3,
				PriceRange: PriceRange{
					Min: targetPrice3 * 0.95,
					Max: targetPrice3 * 1.05,
					Avg: targetPrice3,
				},
				Condition:       "达到最终利润目标或技术信号反转",
				ProfitTarget:    expectedReturn,
				RiskRewardRatio: 2.5,
			},
		}

	case "SHORT":
		// 空头策略出场计划
		targetPrice1 := currentPrice * (1.0 - expectedReturn*0.5)
		targetPrice2 := currentPrice * (1.0 - expectedReturn*0.8)
		targetPrice3 := currentPrice * (1.0 - expectedReturn)

		stages = []ExitStage{
			{
				StageNumber: 1,
				Percentage:  0.3,
				PriceRange: PriceRange{
					Min: targetPrice1 * 0.98,
					Max: targetPrice1 * 1.02,
					Avg: targetPrice1,
				},
				Condition:       "达到第一利润目标",
				ProfitTarget:    expectedReturn * 0.5,
				RiskRewardRatio: 1.5,
			},
			{
				StageNumber: 2,
				Percentage:  0.4,
				PriceRange: PriceRange{
					Min: targetPrice2 * 0.98,
					Max: targetPrice2 * 1.02,
					Avg: targetPrice2,
				},
				Condition:       "达到第二利润目标",
				ProfitTarget:    expectedReturn * 0.8,
				RiskRewardRatio: 2.0,
			},
			{
				StageNumber: 3,
				Percentage:  0.3,
				PriceRange: PriceRange{
					Min: targetPrice3 * 0.95,
					Max: targetPrice3 * 1.05,
					Avg: targetPrice3,
				},
				Condition:       "达到最终利润目标",
				ProfitTarget:    expectedReturn,
				RiskRewardRatio: 2.5,
			},
		}

	default: // RANGE
		// 震荡策略出场计划
		supportLevel := currentPrice * 0.95
		resistanceLevel := currentPrice * 1.05

		stages = []ExitStage{
			{
				StageNumber: 1,
				Percentage:  0.5,
				PriceRange: PriceRange{
					Min: supportLevel * 0.98,
					Max: supportLevel * 1.02,
					Avg: supportLevel,
				},
				Condition:       "价格触及支撑位",
				ProfitTarget:    0.03,
				RiskRewardRatio: 1.0,
			},
			{
				StageNumber: 2,
				Percentage:  0.5,
				PriceRange: PriceRange{
					Min: resistanceLevel * 0.98,
					Max: resistanceLevel * 1.02,
					Avg: resistanceLevel,
				},
				Condition:       "价格触及阻力位",
				ProfitTarget:    0.03,
				RiskRewardRatio: 1.0,
			},
		}
	}

	return stages
}

// generateRiskControls 生成风险控制
func (s *Server) generateRiskControls(strategyType string, totalPosition float64) RiskControls {
	baseRisk := RiskControls{
		MaxLossPerTrade:          0.02,
		MaxDailyLoss:             0.05,
		MaxHoldingTime:           "7天",
		TrailingStop:             true,
		TrailingStopPercent:      0.05,
		PositionCorrelationLimit: 0.7,
	}

	// 根据策略类型调整风险控制
	switch strategyType {
	case "LONG":
		baseRisk.MaxLossPerTrade = 0.025
		baseRisk.MaxHoldingTime = "10天"
	case "SHORT":
		baseRisk.MaxLossPerTrade = 0.03
		baseRisk.MaxHoldingTime = "5天"
	case "RANGE":
		baseRisk.MaxLossPerTrade = 0.015
		baseRisk.MaxHoldingTime = "3天"
		baseRisk.TrailingStop = false
	}

	// 根据仓位大小调整风险
	if totalPosition > 0.2 {
		baseRisk.MaxLossPerTrade *= 0.8 // 大仓位时降低单笔风险
	}

	return baseRisk
}

// generateMonitoringMetrics 生成监控指标
func (s *Server) generateMonitoringMetrics() []string {
	return []string{
		"价格变化率 (1h, 4h, 24h)",
		"交易量变化",
		"技术指标信号 (RSI, MACD, 布林带)",
		"市场情绪指标",
		"相关资产价格变化",
		"资金流入流出情况",
		"持仓盈亏实时监控",
		"止损触发预警",
	}
}

// generateAdjustmentTriggers 生成调整触发条件
func (s *Server) generateAdjustmentTriggers(strategyType string) []AdjustmentTrigger {
	triggers := []AdjustmentTrigger{
		{
			Condition: "价格波动超过10%",
			Action:    "重新评估仓位大小",
			Threshold: 0.1,
			Priority:  "high",
		},
		{
			Condition: "技术指标出现反转信号",
			Action:    "考虑提前出场",
			Threshold: 0.8,
			Priority:  "high",
		},
		{
			Condition: "市场整体出现重大变化",
			Action:    "调整整体风险暴露",
			Threshold: 0.05,
			Priority:  "medium",
		},
		{
			Condition: "持有时间超过预期",
			Action:    "评估是否继续持有",
			Threshold: 7 * 24, // 7天
			Priority:  "medium",
		},
	}

	// 根据策略类型添加特定触发条件
	switch strategyType {
	case "LONG":
		triggers = append(triggers, AdjustmentTrigger{
			Condition: "价格跌破重要支撑位",
			Action:    "考虑加仓或止损",
			Threshold: 0.95,
			Priority:  "high",
		})
	case "SHORT":
		triggers = append(triggers, AdjustmentTrigger{
			Condition: "价格突破重要阻力位",
			Action:    "考虑减仓或止损",
			Threshold: 1.05,
			Priority:  "high",
		})
	}

	return triggers
}

// generateExecutionTimeline 生成执行时间表
func (s *Server) generateExecutionTimeline(strategyType string) ExecutionTimeline {
	now := time.Now()
	timeline := ExecutionTimeline{
		StartTime:        now,
		ExpectedDuration: "7天",
		KeyMilestones: []Milestone{
			{
				Time:        "立即",
				Event:       "开始监控",
				Description: "设置价格警报和监控指标",
			},
			{
				Time:        "24小时内",
				Event:       "第一批建仓",
				Description: "执行第一阶段建仓计划",
			},
			{
				Time:        "48小时内",
				Event:       "第二批建仓",
				Description: "根据市场情况执行第二阶段建仓",
			},
		},
		RiskCheckPoints: []string{
			"建仓后24小时",
			"建仓后48小时",
			"持有期间每日",
			"技术指标出现警告信号时",
		},
	}

	// 根据策略类型调整时间表
	switch strategyType {
	case "LONG":
		timeline.ExpectedDuration = "10天"
		timeline.KeyMilestones = append(timeline.KeyMilestones,
			Milestone{
				Time:        "达到第一目标时",
				Event:       "部分出场",
				Description: "实现部分利润",
			},
			Milestone{
				Time:        "7-10天后",
				Event:       "最终评估",
				Description: "根据市场情况决定继续持有或全部出场",
			},
		)
	case "SHORT":
		timeline.ExpectedDuration = "5天"
		timeline.KeyMilestones = append(timeline.KeyMilestones,
			Milestone{
				Time:        "达到第一目标时",
				Event:       "部分平仓",
				Description: "锁定部分利润",
			},
			Milestone{
				Time:        "3-5天后",
				Event:       "最终评估",
				Description: "根据市场情况决定继续持有或全部平仓",
			},
		)
	case "RANGE":
		timeline.ExpectedDuration = "3天"
		timeline.KeyMilestones = append(timeline.KeyMilestones,
			Milestone{
				Time:        "触及支撑/阻力时",
				Event:       "轮换仓位",
				Description: "从一侧换到另一侧",
			},
			Milestone{
				Time:        "2-3天后",
				Event:       "清仓评估",
				Description: "根据震荡强度决定是否继续",
			},
		)
	}

	return timeline
}
