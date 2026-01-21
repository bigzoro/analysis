package config

import (
	"analysis/internal/server/strategy/traditional"
	"encoding/json"
	"fmt"

	pdb "analysis/internal/db"

	"gorm.io/datatypes"
)

// Manager 配置管理器实现
type Manager struct{}

// NewManager 创建配置管理器
func NewManager() traditional.ConfigManager {
	return &Manager{}
}

// ConvertConfig 将数据库条件转换为传统配置
func (m *Manager) ConvertConfig(conditions pdb.StrategyConditions) *traditional.TraditionalConfig {
	// 设置默认保证金模式
	marginMode := conditions.MarginMode
	if marginMode == "" {
		marginMode = "ISOLATED" // 默认逐仓
	}

	config := &traditional.TraditionalConfig{
		Enabled:            conditions.ShortOnGainers || conditions.LongOnSmallGainers || conditions.FuturesPriceShortStrategyEnabled,
		ShortOnGainers:     conditions.ShortOnGainers,
		LongOnSmallGainers: conditions.LongOnSmallGainers,

		// 排名限制
		GainersRankLimit:     int(conditions.GainersRankLimit),
		GainersRankLimitLong: int(conditions.GainersRankLimitLong),

		// 价格过滤（使用默认值，因为原始配置中可能没有这些字段）
		MinPriceThreshold:  0.000001, // 最低价格
		MaxPriceThreshold:  1000.0,   // 最高价格
		MinVolumeThreshold: 1000.0,   // 最低交易量
		MaxChangePercent:   50.0,     // 最大涨跌幅50%
		MinChangePercent:   -50.0,    // 最小涨跌幅-50%

		// 杠杆配置
		EnableLeverage:  conditions.EnableLeverage,
		DefaultLeverage: conditions.DefaultLeverage,
		MaxLeverage:     conditions.MaxLeverage,

		// 资金费率过滤
		FundingRateFilterEnabled: conditions.FundingRateFilterEnabled,
		MinFundingRate:           conditions.MinFundingRate,

		// 合约涨幅排名过滤
		FuturesPriceRankFilterEnabled: conditions.FuturesPriceRankFilterEnabled,
		MaxFuturesPriceRank:           conditions.MaxFuturesPriceRank,

		// 新增：合约涨幅开空策略
		FuturesPriceShortStrategyEnabled: conditions.FuturesPriceShortStrategyEnabled,
		FuturesPriceShortMinMarketCap:    conditions.FuturesPriceShortMinMarketCap,
		FuturesPriceShortMaxRank:         conditions.FuturesPriceShortMaxRank,
		FuturesPriceShortMinFundingRate:  conditions.FuturesPriceShortMinFundingRate,
		FuturesPriceShortLeverage:        conditions.FuturesPriceShortLeverage,

		// 交易配置
		MarginMode: marginMode,

		// 候选数量限制
		MaxCandidates: 50, // 默认50个

		// 持仓过滤
		SkipHeldPositions: conditions.SkipHeldPositions,

		// 平仓订单过滤
		SkipCloseOrdersWithin24Hours: conditions.SkipCloseOrdersWithin24Hours, // 已废弃
		SkipCloseOrdersHours:         conditions.SkipCloseOrdersHours,

		// 币种黑名单
		UseSymbolBlacklist: conditions.UseSymbolBlacklist,
		SymbolBlacklist:    convertJSONToStringSlice(conditions.SymbolBlacklist),

		// 盈利加仓策略
		ProfitScalingEnabled: conditions.ProfitScalingEnabled,
		ProfitScalingPercent: conditions.ProfitScalingPercent,
		ProfitScalingAmount:  conditions.ProfitScalingAmount,
	}

	return config
}

// ValidateConfig 验证配置
func (m *Manager) ValidateConfig(config *traditional.TraditionalConfig) error {
	if config.GainersRankLimit <= 0 || config.GainersRankLimit > 500 {
		return fmt.Errorf("涨幅榜排名限制必须在1-500之间")
	}

	if config.GainersRankLimitLong <= 0 || config.GainersRankLimitLong > 500 {
		return fmt.Errorf("开多涨幅榜排名限制必须在1-500之间")
	}

	if config.MinPriceThreshold <= 0 || config.MinPriceThreshold >= config.MaxPriceThreshold {
		return fmt.Errorf("价格阈值设置无效")
	}

	if config.MinVolumeThreshold < 0 {
		return fmt.Errorf("交易量阈值不能为负数")
	}

	if config.MaxChangePercent <= config.MinChangePercent {
		return fmt.Errorf("涨跌幅范围设置无效")
	}

	if config.MaxCandidates <= 0 || config.MaxCandidates > 200 {
		return fmt.Errorf("候选数量必须在1-200之间")
	}

	return nil
}

// DefaultConfig 获取默认配置
func (m *Manager) DefaultConfig() *traditional.TraditionalConfig {
	return &traditional.TraditionalConfig{
		Enabled:                       true,
		ShortOnGainers:                true,
		LongOnSmallGainers:            true,
		GainersRankLimit:              100,
		GainersRankLimitLong:          50,
		MinPriceThreshold:             0.000001,
		MaxPriceThreshold:             1000.0,
		MinVolumeThreshold:            1000.0,
		MaxChangePercent:              50.0,
		MinChangePercent:              -50.0,
		EnableLeverage:                false, // 默认不启用杠杆
		DefaultLeverage:               1,     // 默认1倍杠杆
		MaxLeverage:                   100,   // 默认最大100倍杠杆
		FundingRateFilterEnabled:      false, // 默认不启用资金费率过滤
		MinFundingRate:                0.01,  // 默认最低资金费率0.01%
		FuturesPriceRankFilterEnabled: false, // 默认不启用合约涨幅排名过滤
		MaxFuturesPriceRank:           5,     // 默认前5名
		// 新增：合约涨幅开空策略默认值
		FuturesPriceShortStrategyEnabled: false,
		FuturesPriceShortMinMarketCap:    1000, // 默认1000万市值
		FuturesPriceShortMaxRank:         5,
		FuturesPriceShortMinFundingRate:  -0.005,
		FuturesPriceShortLeverage:        3.0,
		MarginMode:                       "ISOLATED", // 默认逐仓
		MaxCandidates:                    50,
	}
}

// convertJSONToStringSlice 将datatypes.JSON转换为[]string
func convertJSONToStringSlice(jsonData datatypes.JSON) []string {
	if len(jsonData) == 0 {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal(jsonData, &result); err != nil {
		// 如果解析失败，返回空切片
		return []string{}
	}

	return result
}
