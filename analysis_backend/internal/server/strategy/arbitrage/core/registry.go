package core

import (
	"analysis/internal/server/strategy/arbitrage"
)

// Registry 套利策略注册表
type Registry struct {
	strategy arbitrage.ArbitrageStrategy
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		strategy: GetArbitrageStrategy(),
	}
}

// GetArbitrageStrategy 获取套利策略
func (r *Registry) GetArbitrageStrategy() arbitrage.ArbitrageStrategy {
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
