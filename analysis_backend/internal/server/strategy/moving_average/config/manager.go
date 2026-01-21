package config

import (
	"analysis/internal/server/strategy/moving_average"
	"fmt"

	pdb "analysis/internal/db"
)

// Manager 配置管理器实现
type Manager struct{}

// NewManager 创建配置管理器
func NewManager() moving_average.ConfigManager {
	return &Manager{}
}

// ConvertConfig 将数据库条件转换为均线配置
func (m *Manager) ConvertConfig(conditions pdb.StrategyConditions) *moving_average.MovingAverageConfig {
	// 设置默认保证金模式
	marginMode := conditions.MarginMode
	if marginMode == "" {
		marginMode = "ISOLATED" // 默认逐仓
	}

	config := &moving_average.MovingAverageConfig{
		Enabled: conditions.MovingAverageEnabled,

		// 均线参数（使用默认值，因为原始配置中可能没有这些字段）
		ShortPeriod:  5,  // 默认5周期短期均线
		LongPeriod:   20, // 默认20周期长期均线
		SignalPeriod: 10, // 默认10周期信号均线

		// 交叉类型
		UseGoldenCross: true,  // 默认启用金叉
		UseDeathCross:  true,  // 默认启用死叉
		UseTrendFilter: false, // 默认不启用趋势过滤

		// 价格过滤
		MinPriceThreshold:  0.000001, // 最低价格
		MaxPriceThreshold:  1000.0,   // 最高价格
		MinVolumeThreshold: 1000.0,   // 最低交易量

		// 信号确认
		RequireVolumeConfirmation: true, // 默认需要交易量确认
		MinCrossStrength:          0.5,  // 默认最小交叉强度0.5
		ConfirmationPeriod:        3,    // 默认3周期确认

		// 杠杆配置
		EnableLeverage:  conditions.EnableLeverage,
		DefaultLeverage: conditions.DefaultLeverage,
		MaxLeverage:     conditions.MaxLeverage,

		// 交易配置
		MarginMode: marginMode,

		// 候选选择
		MaxCandidates:           50,   // 默认50个
		UseVolumeBasedSelection: true, // 默认使用交易量选择
	}

	return config
}

// ValidateConfig 验证配置
func (m *Manager) ValidateConfig(config *moving_average.MovingAverageConfig) error {
	if config.ShortPeriod <= 0 || config.ShortPeriod >= config.LongPeriod {
		return fmt.Errorf("短期均线周期必须大于0且小于长期均线周期")
	}

	if config.LongPeriod <= config.ShortPeriod {
		return fmt.Errorf("长期均线周期必须大于短期均线周期")
	}

	if config.SignalPeriod <= 0 {
		return fmt.Errorf("信号均线周期必须大于0")
	}

	if config.MinPriceThreshold <= 0 || config.MinPriceThreshold >= config.MaxPriceThreshold {
		return fmt.Errorf("价格阈值设置无效")
	}

	if config.MinVolumeThreshold < 0 {
		return fmt.Errorf("交易量阈值不能为负数")
	}

	if config.MinCrossStrength < 0 || config.MinCrossStrength > 1 {
		return fmt.Errorf("交叉强度阈值必须在0-1之间")
	}

	if config.ConfirmationPeriod <= 0 {
		return fmt.Errorf("确认周期必须大于0")
	}

	if config.MaxCandidates <= 0 || config.MaxCandidates > 200 {
		return fmt.Errorf("候选数量必须在1-200之间")
	}

	return nil
}

// DefaultConfig 获取默认配置
func (m *Manager) DefaultConfig() *moving_average.MovingAverageConfig {
	return &moving_average.MovingAverageConfig{
		Enabled:                   true,
		ShortPeriod:               5,
		LongPeriod:                20,
		SignalPeriod:              10,
		UseGoldenCross:            true,
		UseDeathCross:             true,
		UseTrendFilter:            false,
		MinPriceThreshold:         0.000001,
		MaxPriceThreshold:         1000.0,
		MinVolumeThreshold:        1000.0,
		RequireVolumeConfirmation: true,
		MinCrossStrength:          0.5,
		ConfirmationPeriod:        3,
		EnableLeverage:            false, // 默认不启用杠杆
		DefaultLeverage:           1,     // 默认1倍杠杆
		MaxLeverage:               100,   // 默认最大100倍杠杆
		MarginMode:                "ISOLATED", // 默认逐仓
		MaxCandidates:             50,
		UseVolumeBasedSelection:   true,
	}
}
