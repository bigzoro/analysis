package core

import (
	"analysis/internal/server/strategy/mean_reversion"
	"fmt"
	"sync"

	pdb "analysis/internal/db"
)

// Registry 策略注册器 - 单例模式
type Registry struct {
	strategies map[string]mean_reversion.MRStrategy
	mu         sync.RWMutex
}

// globalRegistry 全局注册器实例
var globalRegistry *Registry
var registryOnce sync.Once

// GetRegistry 获取全局策略注册器
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			strategies: make(map[string]mean_reversion.MRStrategy),
		}
		// 自动注册内置策略
		// 注意：扫描逻辑已移至scanning包，此处保留注册器以备将来扩展
	})
	return globalRegistry
}

// RegisterStrategy 注册策略
func (r *Registry) RegisterStrategy(name string, strategy mean_reversion.MRStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.strategies[name]; exists {
		return fmt.Errorf("策略 %s 已经注册", name)
	}

	r.strategies[name] = strategy
	return nil
}

// GetStrategy 获取策略
func (r *Registry) GetStrategy(name string) (mean_reversion.MRStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	strategy, exists := r.strategies[name]
	if !exists {
		return nil, fmt.Errorf("策略 %s 未找到", name)
	}

	return strategy, nil
}

// GetAllStrategies 获取所有已注册的策略
func (r *Registry) GetAllStrategies() map[string]mean_reversion.MRStrategy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本以避免外部修改
	result := make(map[string]mean_reversion.MRStrategy)
	for name, strategy := range r.strategies {
		result[name] = strategy
	}

	return result
}

// IsStrategyRegistered 检查策略是否已注册
func (r *Registry) IsStrategyRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.strategies[name]
	return exists
}

// UnregisterStrategy 注销策略
func (r *Registry) UnregisterStrategy(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.strategies[name]; !exists {
		return fmt.Errorf("策略 %s 未注册", name)
	}

	delete(r.strategies, name)
	return nil
}

// ============================================================================
// 便捷函数
// ============================================================================

// GetMRStrategy 获取均值回归策略实例
func GetMRStrategy() (mean_reversion.MRStrategy, error) {
	registry := GetRegistry()
	return registry.GetStrategy("mean_reversion")
}

// ConvertAndValidateConfig 转换并验证配置
func ConvertAndValidateConfig(conditions pdb.StrategyConditions) (*mean_reversion.MeanReversionConfig, error) {
	strategy, err := GetMRStrategy()
	if err != nil {
		return nil, fmt.Errorf("获取策略失败: %w", err)
	}

	configManager := strategy.GetConfigManager()
	config, err := configManager.ConvertToUnifiedConfig(conditions)
	if err != nil {
		return nil, fmt.Errorf("配置转换失败: %w", err)
	}

	if err := configManager.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// QuickScan 执行快速策略扫描
func QuickScan(symbol string, marketData *mean_reversion.StrategyMarketData, config *mean_reversion.MeanReversionConfig) (*mean_reversion.EligibleSymbol, error) {
	strategy, err := GetMRStrategy()
	if err != nil {
		return nil, fmt.Errorf("获取策略失败: %w", err)
	}

	return strategy.Scan(nil, symbol, marketData, config)
}

// ValidateStrategyConfig 验证策略配置
func ValidateStrategyConfig(config *mean_reversion.MeanReversionConfig) error {
	strategy, err := GetMRStrategy()
	if err != nil {
		return fmt.Errorf("获取策略失败: %w", err)
	}

	validator := strategy.GetValidator()
	return validator.ValidateStrategy(config, &mean_reversion.StrategyMarketData{})
}
