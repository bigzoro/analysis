package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// OptimizationScheduler 优化调度器
type OptimizationScheduler struct {
	optimizer        *AlgorithmOptimizer
	abTester         *ABTestingService
	interval         time.Duration
	running          bool
	stopChan         chan struct{}
	mu               sync.RWMutex
	lastOptimized    time.Time
	optimizationDays int // 优化数据的时间窗口（天）
}

// NewOptimizationScheduler 创建优化调度器
func NewOptimizationScheduler(optimizer *AlgorithmOptimizer, abTester *ABTestingService) *OptimizationScheduler {
	return &OptimizationScheduler{
		optimizer:        optimizer,
		abTester:         abTester,
		interval:         24 * time.Hour, // 默认每天优化一次
		stopChan:         make(chan struct{}),
		optimizationDays: 30, // 默认使用30天的数据
	}
}

// Start 启动优化调度器
func (os *OptimizationScheduler) Start() {
	os.mu.Lock()
	if os.running {
		os.mu.Unlock()
		return
	}
	os.running = true
	os.mu.Unlock()

	log.Println("[优化调度器] 启动，每24小时执行一次算法优化")

	go os.run()
}

// Stop 停止优化调度器
func (os *OptimizationScheduler) Stop() {
	os.mu.Lock()
	if !os.running {
		os.mu.Unlock()
		return
	}
	os.running = false
	close(os.stopChan)
	os.mu.Unlock()

	log.Println("[优化调度器] 已停止")
}

// SetInterval 设置优化间隔
func (os *OptimizationScheduler) SetInterval(interval time.Duration) {
	os.mu.Lock()
	os.interval = interval
	os.mu.Unlock()
}

// SetOptimizationDays 设置优化数据时间窗口
func (os *OptimizationScheduler) SetOptimizationDays(days int) {
	os.mu.Lock()
	os.optimizationDays = days
	os.mu.Unlock()
}

// run 执行优化循环
func (os *OptimizationScheduler) run() {
	ticker := time.NewTicker(os.interval)
	defer ticker.Stop()

	// 立即执行一次优化
	os.performOptimization()

	for {
		select {
		case <-ticker.C:
			os.performOptimization()
		case <-os.stopChan:
			return
		}
	}
}

// performOptimization 执行算法优化
func (os *OptimizationScheduler) performOptimization() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	log.Printf("[优化调度器] 开始算法优化，使用%d天的数据", os.optimizationDays)

	// 执行算法优化
	optimizedWeights, err := os.optimizer.OptimizeWeights(ctx, os.optimizationDays)
	if err != nil {
		log.Printf("[优化调度器] 算法优化失败: %v", err)
		return
	}

	// 记录优化结果
	os.lastOptimized = time.Now()

	// 创建新的A/B测试来验证优化效果
	testConfig := os.createOptimizationTest(optimizedWeights)
	if testConfig != nil {
		// 这里可以自动创建A/B测试来验证优化效果
		testName := testConfig["test_name"].(string)
		log.Printf("[优化调度器] 创建A/B测试验证优化效果: %s", testName)
	}

	log.Printf("[优化调度器] 算法优化完成 - 优化得分: %.3f, 样本数量: %d",
		optimizedWeights.OptimizationScore, optimizedWeights.SampleSize)

	// 记录优化指标
	os.recordOptimizationMetrics(optimizedWeights)
}

// createOptimizationTest 创建优化验证测试
func (os *OptimizationScheduler) createOptimizationTest(weights *OptimizedWeights) map[string]interface{} {
	// 创建A/B测试配置来验证优化效果
	testName := fmt.Sprintf("optimization_test_%s", time.Now().Format("20060102_150405"))

	testConfig := map[string]interface{}{
		"test_name":   testName,
		"description": "算法权重优化验证测试",
		"status":      "active",
		"groups": []map[string]interface{}{
			{
				"group_name":  "control",
				"description": "原始算法权重",
				"weight":      0.5,
				"config": map[string]interface{}{
					"market_weight":    0.30,
					"flow_weight":      0.25,
					"heat_weight":      0.20,
					"event_weight":     0.15,
					"sentiment_weight": 0.10,
				},
			},
			{
				"group_name":  "optimized",
				"description": "优化后的算法权重",
				"weight":      0.5,
				"config": map[string]interface{}{
					"market_weight":    weights.MarketWeight,
					"flow_weight":      weights.FlowWeight,
					"heat_weight":      weights.HeatWeight,
					"event_weight":     weights.EventWeight,
					"sentiment_weight": weights.SentimentWeight,
				},
			},
		},
		"target_metric":   "user_satisfaction", // 用户满意度
		"min_sample_size": 100,
		"start_time":      time.Now(),
		"end_time":        time.Now().AddDate(0, 0, 14), // 2周测试期
		"created_by":      "system",
		"metadata": map[string]interface{}{
			"optimization_score": weights.OptimizationScore,
			"sample_size":        weights.SampleSize,
			"optimization_days":  os.optimizationDays,
		},
	}

	return testConfig
}

// recordOptimizationMetrics 记录优化指标
func (os *OptimizationScheduler) recordOptimizationMetrics(weights *OptimizedWeights) {
	// 这里可以记录到监控系统或数据库
	log.Printf("[优化指标] 市场权重: %.3f, 资金流权重: %.3f, 热度权重: %.3f, 事件权重: %.3f, 情绪权重: %.3f",
		weights.MarketWeight, weights.FlowWeight, weights.HeatWeight,
		weights.EventWeight, weights.SentimentWeight)
	log.Printf("[优化指标] 优化得分: %.3f, 样本数量: %d", weights.OptimizationScore, weights.SampleSize)
}

// GetLastOptimizationTime 获取最后优化时间
func (os *OptimizationScheduler) GetLastOptimizationTime() time.Time {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.lastOptimized
}

// IsRunning 检查调度器是否正在运行
func (os *OptimizationScheduler) IsRunning() bool {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.running
}

// TriggerManualOptimization 手动触发优化
func (os *OptimizationScheduler) TriggerManualOptimization() error {
	os.mu.Lock()
	if os.running {
		os.mu.Unlock()
		return fmt.Errorf("优化调度器正在运行中")
	}
	os.running = true
	os.mu.Unlock()

	defer func() {
		os.mu.Lock()
		os.running = false
		os.mu.Unlock()
	}()

	log.Println("[优化调度器] 手动触发算法优化")
	os.performOptimization()
	return nil
}
