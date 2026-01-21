package core

import (
	"analysis/internal/server/strategy/grid_trading"
)

// Registry 网格交易策略注册表
type Registry struct {
	strategy grid_trading.GridTradingStrategy
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		strategy: GetGridTradingStrategy(),
	}
}

// GetGridTradingStrategy 获取网格交易策略
func (r *Registry) GetGridTradingStrategy() grid_trading.GridTradingStrategy {
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
