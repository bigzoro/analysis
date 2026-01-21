package config

import (
	"analysis/internal/server/strategy/grid_trading"
	"fmt"

	pdb "analysis/internal/db"
)

// Manager 配置管理器实现
type Manager struct{}

// NewManager 创建配置管理器
func NewManager() grid_trading.ConfigManager {
	return &Manager{}
}

// ConvertConfig 将数据库条件转换为网格交易配置
func (m *Manager) ConvertConfig(conditions pdb.StrategyConditions) *grid_trading.GridTradingConfig {
	// 设置默认保证金模式
	marginMode := conditions.MarginMode
	if marginMode == "" {
		marginMode = "ISOLATED" // 默认逐仓
	}

	config := &grid_trading.GridTradingConfig{
		Enabled: conditions.GridTradingEnabled,

		// 网格参数 - 使用默认值，因为原始配置中可能没有这些字段
		GridLevels:         10,   // 默认10个网格层
		GridSpacingPercent: 2.0,  // 默认2%间距
		MinGridRange:       5.0,  // 默认5%最小范围
		MaxGridRange:       20.0, // 默认20%最大范围

		// 评分阈值
		MinVolatilityScore:    0.6, // 默认0.6
		MinLiquidityScore:     0.7, // 默认0.7
		MinStabilityScore:     0.5, // 默认0.5
		OverallScoreThreshold: 0.7, // 默认0.7

		// 杠杆配置
		EnableLeverage:  conditions.EnableLeverage,
		DefaultLeverage: conditions.DefaultLeverage,
		MaxLeverage:     conditions.MaxLeverage,

		// 交易配置
		MarginMode: marginMode,

		// 候选选择
		MaxCandidates: 50,    // 默认50个
		UseMarketCap:  true,  // 默认使用市值
		UseVolume:     false, // 默认不使用交易量
	}

	return config
}

// ValidateConfig 验证配置
func (m *Manager) ValidateConfig(config *grid_trading.GridTradingConfig) error {
	if config.GridLevels <= 0 {
		return fmt.Errorf("网格层数必须大于0")
	}

	if config.GridSpacingPercent <= 0 || config.GridSpacingPercent > 10 {
		return fmt.Errorf("网格间距百分比必须在0-10之间")
	}

	if config.MinGridRange <= 0 || config.MinGridRange >= config.MaxGridRange {
		return fmt.Errorf("网格范围设置无效")
	}

	if config.MaxCandidates <= 0 || config.MaxCandidates > 200 {
		return fmt.Errorf("候选数量必须在1-200之间")
	}

	return nil
}

// DefaultConfig 获取默认配置
func (m *Manager) DefaultConfig() *grid_trading.GridTradingConfig {
	return &grid_trading.GridTradingConfig{
		Enabled:               true,
		GridLevels:            10,
		GridSpacingPercent:    2.0,
		MinGridRange:          5.0,
		MaxGridRange:          20.0,
		MinVolatilityScore:    0.6,
		MinLiquidityScore:     0.7,
		MinStabilityScore:     0.5,
		OverallScoreThreshold: 0.7,
		EnableLeverage:        false, // 默认不启用杠杆
		DefaultLeverage:       1,     // 默认1倍杠杆
		MaxLeverage:           100,   // 默认最大100倍杠杆
		MarginMode:            "ISOLATED", // 默认逐仓
		MaxCandidates:         50,
		UseMarketCap:          true,
		UseVolume:             false,
	}
}
