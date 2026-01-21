package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// FallbackLevel 降级级别
type FallbackLevel int

const (
	FallbackLevelNone     FallbackLevel = iota // 无降级，正常服务
	FallbackLevelPartial                       // 部分降级，使用缓存数据
	FallbackLevelDegraded                      // 降级服务，使用默认值
	FallbackLevelMinimal                       // 最小服务，基本功能
)

// FallbackStrategy 降级策略管理器
type FallbackStrategy struct {
	currentLevel        FallbackLevel
	levelMu             sync.RWMutex
	componentStatus     map[string]bool // 组件状态：true=正常，false=降级
	statusMu            sync.RWMutex
	fallbackHistory     []FallbackEvent
	historyMu           sync.RWMutex
	maxHistorySize      int
	healthCheckInterval time.Duration
}

// FallbackEvent 降级事件
type FallbackEvent struct {
	Timestamp time.Time
	Component string
	OldLevel  FallbackLevel
	NewLevel  FallbackLevel
	Reason    string
	Resolved  bool
}

// FallbackConfig 降级配置
type FallbackConfig struct {
	EnableAutoFallback  bool           // 启用自动降级
	HealthCheckInterval time.Duration  // 健康检查间隔
	MaxHistorySize      int            // 历史记录最大数量
	ComponentThresholds map[string]int // 组件失败阈值
}

// DefaultFallbackConfig 默认降级配置
func DefaultFallbackConfig() FallbackConfig {
	return FallbackConfig{
		EnableAutoFallback:  true,
		HealthCheckInterval: 30 * time.Second,
		MaxHistorySize:      100,
		ComponentThresholds: map[string]int{
			"database":       3,  // 数据库连续失败3次触发降级
			"coingecko":      5,  // CoinGecko连续失败5次触发降级
			"newsapi":        10, // NewsAPI连续失败10次触发降级
			"twitter":        5,  // Twitter连续失败5次触发降级
			"recommendation": 3,  // 推荐服务连续失败3次触发降级
		},
	}
}

// NewFallbackStrategy 创建降级策略管理器
func NewFallbackStrategy(config FallbackConfig) *FallbackStrategy {
	return &FallbackStrategy{
		currentLevel:        FallbackLevelNone,
		componentStatus:     make(map[string]bool),
		fallbackHistory:     make([]FallbackEvent, 0),
		maxHistorySize:      config.MaxHistorySize,
		healthCheckInterval: config.HealthCheckInterval,
	}
}

// GetCurrentLevel 获取当前降级级别
func (fs *FallbackStrategy) GetCurrentLevel() FallbackLevel {
	fs.levelMu.RLock()
	defer fs.levelMu.RUnlock()
	return fs.currentLevel
}

// SetFallbackLevel 设置降级级别
func (fs *FallbackStrategy) SetFallbackLevel(level FallbackLevel, reason string) {
	fs.levelMu.Lock()
	oldLevel := fs.currentLevel
	fs.currentLevel = level
	fs.levelMu.Unlock()

	// 记录降级事件
	fs.recordFallbackEvent("system", oldLevel, level, reason, false)

	log.Printf("[FallbackStrategy] 降级级别变更为: %s (原因: %s)", fs.levelToString(level), reason)

	// 触发降级处理
	fs.handleFallbackLevelChange(oldLevel, level)
}

// RecordComponentFailure 记录组件失败
func (fs *FallbackStrategy) RecordComponentFailure(component string) {
	fs.statusMu.Lock()
	fs.componentStatus[component] = false
	fs.statusMu.Unlock()

	log.Printf("[FallbackStrategy] 组件失败记录: %s", component)
}

// RecordComponentSuccess 记录组件成功
func (fs *FallbackStrategy) RecordComponentSuccess(component string) {
	fs.statusMu.Lock()
	fs.componentStatus[component] = true
	fs.statusMu.Unlock()

	log.Printf("[FallbackStrategy] 组件恢复记录: %s", component)
}

// IsComponentHealthy 检查组件是否健康
func (fs *FallbackStrategy) IsComponentHealthy(component string) bool {
	fs.statusMu.RLock()
	defer fs.statusMu.RUnlock()
	status, exists := fs.componentStatus[component]
	return !exists || status // 不存在的组件默认为健康
}

// GetComponentStatus 获取所有组件状态
func (fs *FallbackStrategy) GetComponentStatus() map[string]bool {
	fs.statusMu.RLock()
	defer fs.statusMu.RUnlock()

	status := make(map[string]bool)
	for k, v := range fs.componentStatus {
		status[k] = v
	}
	return status
}

// EvaluateFallbackLevel 评估当前应该的降级级别
func (fs *FallbackStrategy) EvaluateFallbackLevel() FallbackLevel {
	componentStatus := fs.GetComponentStatus()

	// 计算失败组件数量
	failedCount := 0
	criticalFailed := false

	for component, healthy := range componentStatus {
		if !healthy {
			failedCount++
			// 检查是否为关键组件
			if component == "database" || component == "recommendation" {
				criticalFailed = true
			}
		}
	}

	// 根据失败情况确定降级级别
	if criticalFailed {
		return FallbackLevelMinimal
	} else if failedCount >= 3 {
		return FallbackLevelDegraded
	} else if failedCount >= 1 {
		return FallbackLevelPartial
	} else {
		return FallbackLevelNone
	}
}

// AutoAdjustLevel 自动调整降级级别
func (fs *FallbackStrategy) AutoAdjustLevel() {
	newLevel := fs.EvaluateFallbackLevel()
	currentLevel := fs.GetCurrentLevel()

	if newLevel != currentLevel {
		reason := fmt.Sprintf("自动评估：%d个组件失败", fs.countFailedComponents())
		fs.SetFallbackLevel(newLevel, reason)
	}
}

// ShouldUseCache 检查是否应该使用缓存
func (fs *FallbackStrategy) ShouldUseCache() bool {
	level := fs.GetCurrentLevel()
	return level >= FallbackLevelPartial
}

// ShouldUseDefaults 检查是否应该使用默认值
func (fs *FallbackStrategy) ShouldUseDefaults() bool {
	level := fs.GetCurrentLevel()
	return level >= FallbackLevelDegraded
}

// GetFallbackRecommendation 获取降级建议
func (fs *FallbackStrategy) GetFallbackRecommendation() map[string]interface{} {
	level := fs.GetCurrentLevel()
	componentStatus := fs.GetComponentStatus()

	recommendation := map[string]interface{}{
		"current_level":     fs.levelToString(level),
		"component_status":  componentStatus,
		"use_cache":         fs.ShouldUseCache(),
		"use_defaults":      fs.ShouldUseDefaults(),
		"recommendations":   fs.getLevelRecommendations(level),
		"failed_components": fs.getFailedComponents(),
	}

	return recommendation
}

// 私有方法

func (fs *FallbackStrategy) recordFallbackEvent(component string, oldLevel, newLevel FallbackLevel, reason string, resolved bool) {
	fs.historyMu.Lock()
	defer fs.historyMu.Unlock()

	event := FallbackEvent{
		Timestamp: time.Now(),
		Component: component,
		OldLevel:  oldLevel,
		NewLevel:  newLevel,
		Reason:    reason,
		Resolved:  resolved,
	}

	fs.fallbackHistory = append(fs.fallbackHistory, event)

	// 限制历史记录大小
	if len(fs.fallbackHistory) > fs.maxHistorySize {
		fs.fallbackHistory = fs.fallbackHistory[1:]
	}
}

func (fs *FallbackStrategy) handleFallbackLevelChange(oldLevel, newLevel FallbackLevel) {
	// 处理降级级别变化时的逻辑
	switch newLevel {
	case FallbackLevelNone:
		log.Printf("[FallbackStrategy] 服务已恢复正常")
	case FallbackLevelPartial:
		log.Printf("[FallbackStrategy] 启用部分降级模式，使用缓存数据")
	case FallbackLevelDegraded:
		log.Printf("[FallbackStrategy] 启用降级模式，使用默认值")
	case FallbackLevelMinimal:
		log.Printf("[FallbackStrategy] 启用最小服务模式")
	}
}

func (fs *FallbackStrategy) levelToString(level FallbackLevel) string {
	switch level {
	case FallbackLevelNone:
		return "正常"
	case FallbackLevelPartial:
		return "部分降级"
	case FallbackLevelDegraded:
		return "降级服务"
	case FallbackLevelMinimal:
		return "最小服务"
	default:
		return "未知"
	}
}

func (fs *FallbackStrategy) countFailedComponents() int {
	status := fs.GetComponentStatus()
	count := 0
	for _, healthy := range status {
		if !healthy {
			count++
		}
	}
	return count
}

func (fs *FallbackStrategy) getFailedComponents() []string {
	status := fs.GetComponentStatus()
	failed := make([]string, 0)
	for component, healthy := range status {
		if !healthy {
			failed = append(failed, component)
		}
	}
	return failed
}

func (fs *FallbackStrategy) getLevelRecommendations(level FallbackLevel) []string {
	recommendations := make([]string, 0)

	switch level {
	case FallbackLevelNone:
		recommendations = append(recommendations, "所有服务正常运行")
	case FallbackLevelPartial:
		recommendations = append(recommendations, "使用缓存数据以提高响应速度")
		recommendations = append(recommendations, "减少外部API调用频率")
	case FallbackLevelDegraded:
		recommendations = append(recommendations, "使用默认值替代缺失数据")
		recommendations = append(recommendations, "简化推荐算法计算")
	case FallbackLevelMinimal:
		recommendations = append(recommendations, "提供基础推荐服务")
		recommendations = append(recommendations, "禁用高级功能")
	}

	failedComponents := fs.getFailedComponents()
	if len(failedComponents) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("检查以下组件: %v", failedComponents))
	}

	return recommendations
}

// FallbackDataProvider 降级数据提供者接口
type FallbackDataProvider interface {
	GetFallbackMarketData(ctx context.Context, symbol string) (*MarketDataPoint, error)
	GetFallbackTechnicalData(ctx context.Context, symbol string) (*TechnicalIndicators, error)
	GetFallbackSentimentData(ctx context.Context, symbol string) (*SentimentResult, error)
}

// DefaultFallbackProvider 默认降级数据提供者
type DefaultFallbackProvider struct{}

// GetFallbackMarketData 获取降级市场数据
func (dfp *DefaultFallbackProvider) GetFallbackMarketData(ctx context.Context, symbol string) (*MarketDataPoint, error) {
	// 提供合理的默认值
	return &MarketDataPoint{
		Symbol:         symbol,
		BaseSymbol:     extractBaseSymbol(symbol),
		Price:          1.0,     // 默认价格
		PriceChange24h: 0.0,     // 无变化
		Volume24h:      1000000, // 100万默认成交量
		Timestamp:      time.Now(),
	}, nil
}

// GetFallbackTechnicalData 获取降级技术指标数据
func (dfp *DefaultFallbackProvider) GetFallbackTechnicalData(ctx context.Context, symbol string) (*TechnicalIndicators, error) {
	// 提供中性技术指标
	return &TechnicalIndicators{
		RSI:        50.0, // 中性RSI
		MACD:       0.0,  // MACD为0
		MACDSignal: 0.0,
		BBPosition: 0.0, // 价格在布林带中间
		MA5:        1.0, // 默认均线
		MA10:       1.0,
		MA20:       1.0,
	}, nil
}

// GetFallbackSentimentData 获取降级情绪数据
func (dfp *DefaultFallbackProvider) GetFallbackSentimentData(ctx context.Context, symbol string) (*SentimentResult, error) {
	// 提供中性情绪
	return &SentimentResult{
		Score:    5.0, // 中性分数
		Neutral:  1,
		Total:    1,
		Mentions: 0,
		Trend:    "neutral",
	}, nil
}
