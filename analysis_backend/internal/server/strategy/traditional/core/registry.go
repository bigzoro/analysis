package core

import (
	"analysis/internal/server/strategy/traditional"
	pdb "analysis/internal/db"
)

// Registry 传统策略注册表
type Registry struct {
	strategy traditional.TraditionalStrategy
	database pdb.Database
}

// NewRegistry 创建注册表
func NewRegistry(db interface{}) *Registry {
	return &Registry{
		strategy: GetTraditionalStrategy(db),
	}
}

// GetTraditionalStrategy 获取传统策略
func (r *Registry) GetTraditionalStrategy() traditional.TraditionalStrategy {
	return r.strategy
}

// DefaultRegistry 默认注册表实例
var defaultRegistry *Registry

// GetRegistry 获取默认注册表
func GetRegistry(db interface{}) *Registry {
	if defaultRegistry == nil {
		defaultRegistry = NewRegistry(db)
	}
	return defaultRegistry
}
