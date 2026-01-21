package config

import (
	"analysis/internal/server/strategy/arbitrage"
	"fmt"

	pdb "analysis/internal/db"
)

// Manager 配置管理器实现
type Manager struct{}

// NewManager 创建配置管理器
func NewManager() arbitrage.ConfigManager {
	return &Manager{}
}

// ConvertConfig 将数据库条件转换为套利配置
func (m *Manager) ConvertConfig(conditions pdb.StrategyConditions) *arbitrage.ArbitrageConfig {
	// 设置默认保证金模式
	marginMode := conditions.MarginMode
	if marginMode == "" {
		marginMode = "ISOLATED" // 默认逐仓
	}

	config := &arbitrage.ArbitrageConfig{
		Enabled: conditions.CrossExchangeArbEnabled ||
			conditions.SpotFutureArbEnabled ||
			conditions.TriangleArbEnabled ||
			conditions.StatArbEnabled ||
			conditions.FuturesSpotArbEnabled,

		// 套利类型启用
		CrossExchangeArbEnabled: conditions.CrossExchangeArbEnabled,
		SpotFutureArbEnabled:    conditions.SpotFutureArbEnabled,
		TriangleArbEnabled:      conditions.TriangleArbEnabled,
		StatArbEnabled:          conditions.StatArbEnabled,
		FuturesSpotArbEnabled:   conditions.FuturesSpotArbEnabled,

		// 套利参数（使用默认值，因为原始配置中可能没有这些字段）
		MinProfitThreshold:  0.5,    // 默认0.5%最小利润
		MaxSlippagePercent:  0.2,    // 默认0.2%最大滑点
		MinVolumeThreshold:  1000.0, // 默认1000最小交易量
		ScanIntervalSeconds: 30,     // 默认30秒扫描间隔

		// 三角套利参数
		TriangleMinProfitPercent: 0.3, // 默认0.3%最小利润
		TriangleMaxPathLength:    3,   // 默认3个路径长度

		// 跨交易所套利参数
		ExchangePairs: []string{"binance", "huobi", "okex"}, // 默认交易所对

		// 统计套利参数
		StatArbLookbackPeriod:  100, // 默认100周期回望
		StatArbStdDevThreshold: 2.0, // 默认2.0标准差阈值

		// 杠杆配置
		EnableLeverage:  conditions.EnableLeverage,
		DefaultLeverage: conditions.DefaultLeverage,
		MaxLeverage:     conditions.MaxLeverage,

		// 交易配置
		MarginMode: marginMode,
	}

	return config
}

// ValidateConfig 验证配置
func (m *Manager) ValidateConfig(config *arbitrage.ArbitrageConfig) error {
	if config.MinProfitThreshold <= 0 {
		return fmt.Errorf("最小利润阈值必须大于0")
	}

	if config.MaxSlippagePercent < 0 || config.MaxSlippagePercent > 5 {
		return fmt.Errorf("最大滑点百分比必须在0-5之间")
	}

	if config.MinVolumeThreshold <= 0 {
		return fmt.Errorf("最小交易量阈值必须大于0")
	}

	if config.ScanIntervalSeconds <= 0 {
		return fmt.Errorf("扫描间隔秒数必须大于0")
	}

	if config.TriangleMinProfitPercent <= 0 {
		return fmt.Errorf("三角套利最小利润百分比必须大于0")
	}

	if config.TriangleMaxPathLength <= 2 {
		return fmt.Errorf("三角套利最大路径长度必须大于2")
	}

	if len(config.ExchangePairs) == 0 {
		return fmt.Errorf("交易所对列表不能为空")
	}

	if config.StatArbLookbackPeriod <= 0 {
		return fmt.Errorf("统计套利回望周期必须大于0")
	}

	if config.StatArbStdDevThreshold <= 0 {
		return fmt.Errorf("统计套利标准差阈值必须大于0")
	}

	return nil
}

// DefaultConfig 获取默认配置
func (m *Manager) DefaultConfig() *arbitrage.ArbitrageConfig {
	return &arbitrage.ArbitrageConfig{
		Enabled:                  true,
		CrossExchangeArbEnabled:  true,
		SpotFutureArbEnabled:     true,
		TriangleArbEnabled:       true,
		StatArbEnabled:           false,
		FuturesSpotArbEnabled:    true,
		MinProfitThreshold:       0.5,
		MaxSlippagePercent:       0.2,
		MinVolumeThreshold:       1000.0,
		ScanIntervalSeconds:      30,
		TriangleMinProfitPercent: 0.3,
		TriangleMaxPathLength:    3,
		ExchangePairs:            []string{"binance", "huobi", "okex"},
		StatArbLookbackPeriod:    100,
		StatArbStdDevThreshold:   2.0,
		EnableLeverage:           false, // 默认不启用杠杆
		DefaultLeverage:          1,     // 默认1倍杠杆
		MaxLeverage:              100,   // 默认最大100倍杠杆
		MarginMode:               "ISOLATED", // 默认逐仓
	}
}
