package core

import (
	"analysis/internal/server/strategy/moving_average"
	"analysis/internal/server/strategy/moving_average/config"
	"analysis/internal/server/strategy/moving_average/scanning"
	"context"
)

// Strategy 均线策略实现
type Strategy struct {
	configManager moving_average.ConfigManager
	scanner       *scanning.Scanner
}

// NewStrategy 创建均线策略
func NewStrategy() *Strategy {
	configManager := config.NewManager()
	scanner := scanning.NewScanner()

	return &Strategy{
		configManager: configManager,
		scanner:       scanner,
	}
}

// Scan 执行均线策略扫描
func (s *Strategy) Scan(ctx context.Context, config *moving_average.MovingAverageConfig) ([]moving_average.ValidationResult, error) {
	return s.scanner.Scan(ctx, config)
}

// IsEnabled 检查策略是否启用
func (s *Strategy) IsEnabled(config *moving_average.MovingAverageConfig) bool {
	return config.Enabled
}

// DetectCrossSignals 检测交叉信号
func (s *Strategy) DetectCrossSignals(ctx context.Context, symbol string, config *moving_average.MovingAverageConfig) ([]moving_average.CrossSignal, error) {
	return s.scanner.DetectCrossSignals(ctx, symbol, config)
}

// ValidateSignals 验证信号
func (s *Strategy) ValidateSignals(ctx context.Context, signals []moving_average.CrossSignal, config *moving_average.MovingAverageConfig) ([]moving_average.ValidationResult, error) {
	return s.scanner.ValidateSignals(ctx, signals, config)
}

// ToStrategyScanner 创建适配器
func (s *Strategy) ToStrategyScanner() interface{} {
	return s.scanner.ToStrategyScanner()
}

// ============================================================================
// 注册表
// ============================================================================

var globalStrategy *Strategy

// GetMovingAverageStrategy 获取均线策略实例
func GetMovingAverageStrategy() *Strategy {
	if globalStrategy == nil {
		globalStrategy = NewStrategy()
	}
	return globalStrategy
}
