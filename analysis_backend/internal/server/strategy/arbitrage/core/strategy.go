package core

import (
	"analysis/internal/server/strategy/arbitrage"
	"analysis/internal/server/strategy/arbitrage/config"
	"analysis/internal/server/strategy/arbitrage/scanning"
	"context"
)

// Strategy 套利策略实现
type Strategy struct {
	configManager arbitrage.ConfigManager
	scanner       *scanning.Scanner
}

// NewStrategy 创建套利策略
func NewStrategy() *Strategy {
	configManager := config.NewManager()
	scanner := scanning.NewScanner()

	return &Strategy{
		configManager: configManager,
		scanner:       scanner,
	}
}

// Scan 执行套利策略扫描
func (s *Strategy) Scan(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	return s.scanner.Scan(ctx, config)
}

// IsEnabled 检查策略是否启用
func (s *Strategy) IsEnabled(config *arbitrage.ArbitrageConfig) bool {
	return config.Enabled
}

// ScanTriangleArbitrage 三角套利扫描
func (s *Strategy) ScanTriangleArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	return s.scanner.ScanTriangleArbitrage(ctx, config)
}

// ScanCrossExchangeArbitrage 跨交易所套利扫描
func (s *Strategy) ScanCrossExchangeArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	return s.scanner.ScanCrossExchangeArbitrage(ctx, config)
}

// ScanSpotFutureArbitrage 现货期货套利扫描
func (s *Strategy) ScanSpotFutureArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	return s.scanner.ScanSpotFutureArbitrage(ctx, config)
}

// ScanStatisticalArbitrage 统计套利扫描
func (s *Strategy) ScanStatisticalArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	return s.scanner.ScanStatisticalArbitrage(ctx, config)
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

// GetArbitrageStrategy 获取套利策略实例
func GetArbitrageStrategy() *Strategy {
	if globalStrategy == nil {
		globalStrategy = NewStrategy()
	}
	return globalStrategy
}

// EligibleSymbol 符合条件的交易对信息（本地定义以避免循环导入）
type EligibleSymbol struct {
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"`
	Reason     string  `json:"reason"`
	Multiplier float64 `json:"multiplier"`
	MarketCap  float64 `json:"market_cap"`
}
