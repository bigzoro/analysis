package core

import (
	"analysis/internal/server/strategy/grid_trading"
	"analysis/internal/server/strategy/grid_trading/config"
	"analysis/internal/server/strategy/grid_trading/scanning"
	"context"
)

// Strategy 网格交易策略实现
type Strategy struct {
	configManager grid_trading.ConfigManager
	scanner       *scanning.Scanner
}

// NewStrategy 创建网格交易策略
func NewStrategy() *Strategy {
	configManager := config.NewManager()
	scanner := scanning.NewScanner()

	return &Strategy{
		configManager: configManager,
		scanner:       scanner,
	}
}

// Scan 执行网格交易策略扫描
func (s *Strategy) Scan(ctx context.Context, config *grid_trading.GridTradingConfig) ([]grid_trading.CandidateResult, error) {
	return s.scanner.Scan(ctx, config)
}

// IsEnabled 检查策略是否启用
func (s *Strategy) IsEnabled(config *grid_trading.GridTradingConfig) bool {
	return config.Enabled
}

// CalculateGridRange 计算网格范围
func (s *Strategy) CalculateGridRange(currentPrice float64, config *grid_trading.GridTradingConfig) grid_trading.GridRange {
	return s.scanner.CalculateGridRange(currentPrice, config)
}

// ValidateGridParameters 验证网格参数
func (s *Strategy) ValidateGridParameters(config *grid_trading.GridTradingConfig) error {
	return s.configManager.ValidateConfig(config)
}

// ============================================================================
// 适配器和注册表
// ============================================================================

// ToStrategyScanner 创建适配器
func (s *Strategy) ToStrategyScanner() interface{} {
	return s.scanner.ToStrategyScanner()
}

// ============================================================================
// 注册表
// ============================================================================

var globalStrategy *Strategy

// GetGridTradingStrategy 获取网格交易策略实例
func GetGridTradingStrategy() *Strategy {
	if globalStrategy == nil {
		globalStrategy = NewStrategy()
	}
	return globalStrategy
}

// EligibleSymbol 符合条件的交易对信息（本地定义以避免循环导入）
type EligibleSymbol struct {
	Symbol      string  `json:"symbol"`
	Action      string  `json:"action"`
	Reason      string  `json:"reason"`
	Multiplier  float64 `json:"multiplier"`
	MarketCap   float64 `json:"market_cap"`
	GainersRank int     `json:"gainers_rank,omitempty"`
	// 三角套利专用字段
	TrianglePath []string `json:"triangle_path,omitempty"`
}
