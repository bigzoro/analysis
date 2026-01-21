package server

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// ==================== Worker Pool（协程池）====================

// WorkerPool 协程池，用于限制并发数量
type WorkerPool struct {
	maxWorkers int
	workers    chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

// NewWorkerPool 创建协程池
// maxWorkers: 最大并发数，0 表示不限制
func NewWorkerPool(maxWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}
	if maxWorkers > 0 {
		wp.workers = make(chan struct{}, maxWorkers)
	}
	return wp
}

// Submit 提交任务到协程池
func (wp *WorkerPool) Submit(task func()) {
	if wp.maxWorkers > 0 {
		// 等待获取工作槽位
		select {
		case wp.workers <- struct{}{}:
		case <-wp.ctx.Done():
			return
		}
	}

	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		if wp.maxWorkers > 0 {
			defer func() { <-wp.workers }()
		}

		// 检查上下文是否已取消
		select {
		case <-wp.ctx.Done():
			return
		default:
		}

		task()
	}()
}

// SubmitWithTimeout 提交带超时的任务
func (wp *WorkerPool) SubmitWithTimeout(task func(), timeout time.Duration) {
	if wp.maxWorkers > 0 {
		select {
		case wp.workers <- struct{}{}:
		case <-wp.ctx.Done():
			return
		case <-time.After(timeout):
			return // 超时，放弃任务
		}
	}

	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		if wp.maxWorkers > 0 {
			defer func() { <-wp.workers }()
		}

		ctx, cancel := context.WithTimeout(wp.ctx, timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			task()
			close(done)
		}()

		select {
		case <-done:
			// 任务完成
		case <-ctx.Done():
			// 超时或取消
		}
	}()
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// Running 返回当前运行中的worker数量
func (wp *WorkerPool) Running() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if wp.maxWorkers <= 0 {
		return 0 // 不限制并发时，无法准确计算
	}
	return wp.maxWorkers - len(wp.workers)
}

// Resize 动态调整协程池大小
func (wp *WorkerPool) Resize(newMaxWorkers int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if newMaxWorkers == wp.maxWorkers {
		return
	}

	if newMaxWorkers <= 0 {
		// 取消并发限制
		if wp.workers != nil {
			close(wp.workers)
		}
		wp.workers = nil
		wp.maxWorkers = 0
		return
	}

	if wp.maxWorkers <= 0 {
		// 从无限制改为有限制
		wp.workers = make(chan struct{}, newMaxWorkers)
	} else {
		// 调整现有通道大小
		oldWorkers := wp.workers
		wp.workers = make(chan struct{}, newMaxWorkers)

		// 复制现有的令牌
		oldLen := len(oldWorkers)
		newLen := min(oldLen, newMaxWorkers)
		for i := 0; i < newLen; i++ {
			<-oldWorkers
			wp.workers <- struct{}{}
		}
	}

	wp.maxWorkers = newMaxWorkers
}

// Shutdown 优雅关闭协程池
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	wp.cancel()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}
}

// ErrShutdownTimeout 关闭超时错误
var ErrShutdownTimeout = &ShutdownTimeoutError{}

type ShutdownTimeoutError struct{}

func (e *ShutdownTimeoutError) Error() string {
	return "worker pool shutdown timeout"
}

// ==================== 并发安全的批量处理 ====================

// BatchProcessor 批量处理器
type BatchProcessor struct {
	pool *WorkerPool
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(maxConcurrency int) *BatchProcessor {
	return &BatchProcessor{
		pool: NewWorkerPool(maxConcurrency),
	}
}

// ProcessBatch 并发处理批量任务（使用接口类型，避免泛型）
func (bp *BatchProcessor) ProcessBatch(
	items []interface{},
	processor func(interface{}) (interface{}, error),
) ([]interface{}, []error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]interface{}, len(items))
	errors := make([]error, len(items))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range items {
		wg.Add(1)
		idx := i
		it := item

		bp.pool.Submit(func() {
			defer wg.Done()
			result, err := processor(it)
			mu.Lock()
			results[idx] = result
			errors[idx] = err
			mu.Unlock()
		})
	}

	wg.Wait()
	return results, errors
}

// ProcessBatchWithContext 带上下文的批量处理（使用接口类型，避免泛型）
func (bp *BatchProcessor) ProcessBatchWithContext(
	ctx context.Context,
	items []interface{},
	processor func(context.Context, interface{}) (interface{}, error),
) ([]interface{}, []error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]interface{}, len(items))
	errors := make([]error, len(items))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range items {
		select {
		case <-ctx.Done():
			// 上下文已取消，填充错误
			for j := i; j < len(items); j++ {
				errors[j] = ctx.Err()
			}
			return results, errors
		default:
		}

		wg.Add(1)
		idx := i
		it := item

		bp.pool.Submit(func() {
			defer wg.Done()
			result, err := processor(ctx, it)
			mu.Lock()
			results[idx] = result
			errors[idx] = err
			mu.Unlock()
		})
	}

	wg.Wait()
	return results, errors
}

// Shutdown 关闭批量处理器
func (bp *BatchProcessor) Shutdown(timeout time.Duration) error {
	return bp.pool.Shutdown(timeout)
}

// ============================================================================
// 智能工作者池 - 自适应并发控制
// ============================================================================

// SmartWorkerPool 智能工作者池，具备自适应能力
type SmartWorkerPool struct {
	// 基础配置
	minWorkers     int   // 最小工作者数量
	maxWorkers     int   // 最大工作者数量
	currentWorkers int32 // 当前工作者数量（原子操作）
	targetWorkers  int32 // 目标工作者数量

	// 自适应配置
	adaptiveEnabled    bool          // 是否启用自适应
	targetLatency      time.Duration // 目标延迟
	scaleUpThreshold   float64       // 扩容阈值（CPU使用率）
	scaleDownThreshold float64       // 缩容阈值（CPU使用率）
	checkInterval      time.Duration // 检查间隔

	// 任务队列
	tasks chan func()   // 任务队列
	stop  chan struct{} // 停止信号

	// 统计信息
	stats SmartWorkerStats

	// 同步机制
	wg     sync.WaitGroup
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// SmartWorkerStats 智能工作者池统计
type SmartWorkerStats struct {
	TasksProcessed    int64         // 已处理任务数
	TasksQueued       int64         // 队列中的任务数
	TasksRejected     int64         // 被拒绝的任务数
	AvgProcessingTime time.Duration // 平均处理时间
	MaxProcessingTime time.Duration // 最大处理时间
	MinProcessingTime time.Duration // 最小处理时间
	LastScaleTime     time.Time     // 上次扩缩容时间
	WorkerCount       int32         // 当前工作者数量
	QueueLength       int           // 队列长度
}

// NewSmartWorkerPool 创建智能工作者池
func NewSmartWorkerPool(minWorkers, maxWorkers int, queueSize int) *SmartWorkerPool {
	if minWorkers <= 0 {
		minWorkers = 1
	}
	if maxWorkers < minWorkers {
		maxWorkers = minWorkers
	}
	if queueSize <= 0 {
		queueSize = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	swp := &SmartWorkerPool{
		minWorkers:         minWorkers,
		maxWorkers:         maxWorkers,
		currentWorkers:     int32(minWorkers),
		targetWorkers:      int32(minWorkers),
		adaptiveEnabled:    true,
		targetLatency:      100 * time.Millisecond,
		scaleUpThreshold:   0.8, // 80% CPU使用率
		scaleDownThreshold: 0.3, // 30% CPU使用率
		checkInterval:      10 * time.Second,
		tasks:              make(chan func(), queueSize),
		stop:               make(chan struct{}),
		ctx:                ctx,
		cancel:             cancel,
	}

	// 初始化统计信息
	swp.stats.MinProcessingTime = time.Hour // 初始化为较大值

	// 启动初始工作者
	swp.startWorkers(int(minWorkers))

	// 启动自适应管理器
	if swp.adaptiveEnabled {
		go swp.adaptiveManager()
	}

	return swp
}

// Submit 提交任务
func (swp *SmartWorkerPool) Submit(task func()) bool {
	select {
	case swp.tasks <- task:
		atomic.AddInt64(&swp.stats.TasksQueued, 1)
		return true
	default:
		atomic.AddInt64(&swp.stats.TasksRejected, 1)
		return false
	}
}

// SubmitWithTimeout 带超时的任务提交
func (swp *SmartWorkerPool) SubmitWithTimeout(task func(), timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case swp.tasks <- task:
		atomic.AddInt64(&swp.stats.TasksQueued, 1)
		return true
	case <-timer.C:
		atomic.AddInt64(&swp.stats.TasksRejected, 1)
		return false
	case <-swp.ctx.Done():
		return false
	}
}

// startWorkers 启动指定数量的工作者
func (swp *SmartWorkerPool) startWorkers(count int) {
	for i := 0; i < count; i++ {
		swp.wg.Add(1)
		go swp.worker()
	}
}

// worker 工作者协程
func (swp *SmartWorkerPool) worker() {
	defer swp.wg.Done()

	for {
		select {
		case task := <-swp.tasks:
			atomic.AddInt64(&swp.stats.TasksQueued, -1)
			swp.executeTask(task)
		case <-swp.stop:
			return
		case <-swp.ctx.Done():
			return
		}
	}
}

// executeTask 执行任务并记录统计信息
func (swp *SmartWorkerPool) executeTask(task func()) {
	startTime := time.Now()

	defer func() {
		processingTime := time.Since(startTime)
		atomic.AddInt64(&swp.stats.TasksProcessed, 1)

		// 更新统计信息
		swp.mu.Lock()
		defer swp.mu.Unlock()

		// 更新平均处理时间
		processed := atomic.LoadInt64(&swp.stats.TasksProcessed)
		if processed == 1 {
			swp.stats.AvgProcessingTime = processingTime
		} else {
			// 滑动平均
			swp.stats.AvgProcessingTime = time.Duration(
				(int64(swp.stats.AvgProcessingTime)*99 + int64(processingTime)) / 100,
			)
		}

		// 更新最大/最小处理时间
		if processingTime > swp.stats.MaxProcessingTime {
			swp.stats.MaxProcessingTime = processingTime
		}
		if processingTime < swp.stats.MinProcessingTime {
			swp.stats.MinProcessingTime = processingTime
		}
	}()

	// 执行任务
	task()
}

// adaptiveManager 自适应管理器
func (swp *SmartWorkerPool) adaptiveManager() {
	ticker := time.NewTicker(swp.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			swp.adjustWorkers()
		case <-swp.stop:
			return
		case <-swp.ctx.Done():
			return
		}
	}
}

// adjustWorkers 调整工作者数量
func (swp *SmartWorkerPool) adjustWorkers() {
	currentWorkers := atomic.LoadInt32(&swp.currentWorkers)
	queueLength := len(swp.tasks)

	// 计算系统负载
	loadFactor := float64(queueLength) / float64(cap(swp.tasks))

	// 计算CPU使用率（简化实现，实际应该从系统监控获取）
	cpuUsage := swp.estimateCPUUsage()

	var newTargetWorkers int32

	// 自适应扩缩容逻辑
	if loadFactor > 0.8 || cpuUsage > swp.scaleUpThreshold {
		// 高负载：增加工作者
		newTargetWorkers = int32(math.Min(float64(swp.maxWorkers), float64(currentWorkers)*1.5))
	} else if loadFactor < 0.2 && cpuUsage < swp.scaleDownThreshold && currentWorkers > int32(swp.minWorkers) {
		// 低负载：减少工作者
		newTargetWorkers = int32(math.Max(float64(swp.minWorkers), float64(currentWorkers)*0.8))
	} else {
		newTargetWorkers = currentWorkers
	}

	// 应用扩缩容
	if newTargetWorkers != currentWorkers {
		swp.scaleWorkers(newTargetWorkers)
		swp.stats.LastScaleTime = time.Now()
	}
}

// scaleWorkers 缩放工作者数量
func (swp *SmartWorkerPool) scaleWorkers(newCount int32) {
	currentCount := atomic.LoadInt32(&swp.currentWorkers)

	if newCount > currentCount {
		// 扩容
		count := newCount - currentCount
		swp.startWorkers(int(count))
		atomic.AddInt32(&swp.currentWorkers, count)
		atomic.StoreInt32(&swp.targetWorkers, newCount)
	} else if newCount < currentCount {
		// 缩容：通过上下文取消让工作者自然退出
		atomic.StoreInt32(&swp.targetWorkers, newCount)
		// 注意：实际缩容需要更复杂的逻辑，这里简化处理
	}
}

// estimateCPUUsage 估算CPU使用率（简化实现）
func (swp *SmartWorkerPool) estimateCPUUsage() float64 {
	// 这里应该从系统监控获取真实的CPU使用率
	// 暂时基于队列长度和处理时间估算

	queueLength := len(swp.tasks)
	queueCapacity := cap(swp.tasks)
	queueUtilization := float64(queueLength) / float64(queueCapacity)

	processingTime := swp.stats.AvgProcessingTime
	latencyFactor := float64(processingTime) / float64(swp.targetLatency)

	// 综合估算CPU使用率
	estimatedCPU := (queueUtilization + math.Min(latencyFactor, 2.0)) / 3.0
	return math.Min(1.0, math.Max(0.0, estimatedCPU))
}

// GetStats 获取统计信息
func (swp *SmartWorkerPool) GetStats() SmartWorkerStats {
	swp.mu.RLock()
	defer swp.mu.RUnlock()

	return SmartWorkerStats{
		TasksProcessed:    atomic.LoadInt64(&swp.stats.TasksProcessed),
		TasksQueued:       atomic.LoadInt64(&swp.stats.TasksQueued),
		TasksRejected:     atomic.LoadInt64(&swp.stats.TasksRejected),
		AvgProcessingTime: swp.stats.AvgProcessingTime,
		MaxProcessingTime: swp.stats.MaxProcessingTime,
		MinProcessingTime: swp.stats.MinProcessingTime,
		LastScaleTime:     swp.stats.LastScaleTime,
		WorkerCount:       atomic.LoadInt32(&swp.currentWorkers),
		QueueLength:       len(swp.tasks),
	}
}

// SetAdaptiveConfig 设置自适应配置
func (swp *SmartWorkerPool) SetAdaptiveConfig(config map[string]interface{}) {
	swp.mu.Lock()
	defer swp.mu.Unlock()

	if val, ok := config["enabled"].(bool); ok {
		swp.adaptiveEnabled = val
	}
	if val, ok := config["target_latency"].(time.Duration); ok {
		swp.targetLatency = val
	}
	if val, ok := config["scale_up_threshold"].(float64); ok {
		swp.scaleUpThreshold = val
	}
	if val, ok := config["scale_down_threshold"].(float64); ok {
		swp.scaleDownThreshold = val
	}
	if val, ok := config["check_interval"].(time.Duration); ok {
		swp.checkInterval = val
	}
}

// Shutdown 优雅关闭
func (swp *SmartWorkerPool) Shutdown(timeout time.Duration) error {
	// 发送停止信号
	close(swp.stop)
	swp.cancel()

	// 等待工作者完成或超时
	done := make(chan struct{})
	go func() {
		swp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}
}
