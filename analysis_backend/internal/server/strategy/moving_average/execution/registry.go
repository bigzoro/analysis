package execution

import (
	"analysis/internal/server/strategy/shared/execution"
)

// ============================================================================
// 执行器注册表
// ============================================================================

// Registry 执行器注册表
type Registry struct {
	executors map[string]execution.StrategyExecutor
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]execution.StrategyExecutor),
	}
}

// Register 注册执行器
func (r *Registry) Register(strategyType string, executor execution.StrategyExecutor) {
	r.executors[strategyType] = executor
}

// Get 获取执行器
func (r *Registry) Get(strategyType string) execution.StrategyExecutor {
	return r.executors[strategyType]
}

// GetAll 获取所有执行器
func (r *Registry) GetAll() map[string]execution.StrategyExecutor {
	return r.executors
}

// CreateMovingAverageExecutor 创建均线策略执行器
func CreateMovingAverageExecutor(deps *ExecutionDependencies) execution.StrategyExecutor {
	return NewExecutor(deps)
}

// DefaultDependencies 创建默认依赖（用于测试或简单场景）
func DefaultDependencies() *ExecutionDependencies {
	return &ExecutionDependencies{
		// 这里可以设置默认的依赖实现
		// 实际使用时应该注入真实的依赖
	}
}
