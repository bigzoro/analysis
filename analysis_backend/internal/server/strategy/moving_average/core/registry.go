package core

import (
	"analysis/internal/server/strategy/moving_average"
)

// Registry 均线策略注册表
type Registry struct {
	strategy moving_average.MovingAverageStrategy
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		strategy: GetMovingAverageStrategy(),
	}
}

// GetMovingAverageStrategy 获取均线策略
func (r *Registry) GetMovingAverageStrategy() moving_average.MovingAverageStrategy {
	return r.strategy
}

// DefaultRegistry 默认注册表实例
var defaultRegistry *Registry

// GetRegistry 获取默认注册表
func GetRegistry() *Registry {
	if defaultRegistry == nil {
		defaultRegistry = NewRegistry()
	}
	return defaultRegistry
}
