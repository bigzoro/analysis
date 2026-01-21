package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// SmartScheduler 智能调度器 - 基于事件驱动和优先级调度
type SmartScheduler struct {
	server *Server
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 调度队列
	taskQueue  chan *ScheduledTask
	workerPool *WorkerPool

	// 调度配置
	config SchedulerConfig

	// 状态跟踪
	running      bool
	lastRunTimes map[string]time.Time
	runStats     map[string]*TaskStats

	// 事件监听器
	eventChan     chan SchedulerEvent
	eventHandlers map[string][]EventHandler

	mu sync.RWMutex
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	// 工作池大小
	WorkerPoolSize int

	// 调度间隔
	PerformanceUpdateInterval time.Duration
	CacheWarmupInterval       time.Duration
	DataCleanupInterval       time.Duration

	// 优先级配置
	MaxConcurrentTasks int
	TaskTimeout        time.Duration

	// 自适应配置
	AdaptiveEnabled   bool
	LoadThreshold     float64 // CPU/内存负载阈值
	BackoffMultiplier float64
	MaxBackoffDelay   time.Duration
}

// ScheduledTask 调度任务
type ScheduledTask struct {
	ID         string
	Type       string
	Priority   int // 1-10, 10最高优先级
	Data       interface{}
	CreatedAt  time.Time
	Deadline   time.Time
	RetryCount int
	MaxRetries int
	Handler    TaskHandler
}

// TaskHandler 任务处理器
type TaskHandler func(ctx context.Context, task *ScheduledTask) error

// TaskStats 任务统计信息
type TaskStats struct {
	TaskType     string
	TotalRuns    int64
	SuccessCount int64
	FailureCount int64
	AvgDuration  time.Duration
	LastRunTime  time.Time
	LastError    error
	SuccessRate  float64
}

// SchedulerEvent 调度器事件
type SchedulerEvent struct {
	Type      string
	TaskID    string
	TaskType  string
	Timestamp time.Time
	Data      interface{}
	Error     error
}

// EventHandler 事件处理器
type EventHandler func(event SchedulerEvent)

// TaskType 任务类型常量
const (
	TaskTypePerformanceUpdate = "performance_update"
	TaskTypeCacheWarmup       = "cache_warmup"
	TaskTypeDataCleanup       = "data_cleanup"
	TaskTypeStrategyBacktest  = "strategy_backtest"
	TaskTypeRecommendationGen = "recommendation_gen"
)

// NewSmartScheduler 创建智能调度器
func NewSmartScheduler(server *Server) *SmartScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	config := SchedulerConfig{
		WorkerPoolSize:            15,
		PerformanceUpdateInterval: 5 * time.Minute, // 更频繁的更新
		CacheWarmupInterval:       10 * time.Minute,
		DataCleanupInterval:       1 * time.Hour,

		MaxConcurrentTasks: 20,
		TaskTimeout:        10 * time.Minute,

		AdaptiveEnabled:   true,
		LoadThreshold:     0.8,
		BackoffMultiplier: 2.0,
		MaxBackoffDelay:   30 * time.Minute,
	}

	return &SmartScheduler{
		server:        server,
		ctx:           ctx,
		cancel:        cancel,
		taskQueue:     make(chan *ScheduledTask, 100),
		workerPool:    NewWorkerPool(config.WorkerPoolSize),
		config:        config,
		lastRunTimes:  make(map[string]time.Time),
		runStats:      make(map[string]*TaskStats),
		eventChan:     make(chan SchedulerEvent, 100),
		eventHandlers: make(map[string][]EventHandler),
	}
}

// Start 启动智能调度器
func (ss *SmartScheduler) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.running {
		return fmt.Errorf("scheduler already running")
	}

	ss.running = true
	log.Printf("[SmartScheduler] 启动智能调度器...")

	// 启动事件处理器
	ss.wg.Add(1)
	go ss.eventProcessor()

	// 启动任务调度器
	ss.wg.Add(1)
	go ss.taskScheduler()

	// 启动定期任务
	ss.wg.Add(1)
	go ss.periodicTaskManager()

	// 启动性能监控
	ss.wg.Add(1)
	go ss.performanceMonitor()

	// 启动自适应调整器
	if ss.config.AdaptiveEnabled {
		ss.wg.Add(1)
		go ss.adaptiveAdjuster()
	}

	log.Printf("[SmartScheduler] 智能调度器启动完成")
	return nil
}

// Stop 停止调度器
func (ss *SmartScheduler) Stop() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.running {
		return nil
	}

	log.Printf("[SmartScheduler] 停止智能调度器...")
	ss.running = false
	ss.cancel()

	close(ss.taskQueue)
	close(ss.eventChan)

	ss.wg.Wait()
	log.Printf("[SmartScheduler] 智能调度器已停止")
	return nil
}

// ScheduleTask 调度任务
func (ss *SmartScheduler) ScheduleTask(taskType string, priority int, data interface{}, handler TaskHandler) error {
	task := &ScheduledTask{
		ID:         generateTaskID(taskType),
		Type:       taskType,
		Priority:   priority,
		Data:       data,
		CreatedAt:  time.Now(),
		Deadline:   time.Now().Add(ss.config.TaskTimeout),
		MaxRetries: 3,
		Handler:    handler,
	}

	select {
	case ss.taskQueue <- task:
		ss.emitEvent(SchedulerEvent{
			Type:     "task_scheduled",
			TaskID:   task.ID,
			TaskType: task.Type,
			Data:     task,
		})
		return nil
	default:
		return fmt.Errorf("task queue full")
	}
}

// ScheduleDelayedTask 调度延迟任务
func (ss *SmartScheduler) ScheduleDelayedTask(taskType string, priority int, delay time.Duration, data interface{}, handler TaskHandler) error {
	task := &ScheduledTask{
		ID:         generateTaskID(taskType),
		Type:       taskType,
		Priority:   priority,
		Data:       data,
		CreatedAt:  time.Now(),
		Deadline:   time.Now().Add(delay + ss.config.TaskTimeout),
		MaxRetries: 3,
		Handler:    handler,
	}

	time.AfterFunc(delay, func() {
		select {
		case ss.taskQueue <- task:
			ss.emitEvent(SchedulerEvent{
				Type:     "task_scheduled_delayed",
				TaskID:   task.ID,
				TaskType: task.Type,
				Data:     task,
			})
		default:
			log.Printf("[SmartScheduler] 延迟任务队列已满: %s", task.ID)
		}
	})

	return nil
}

// OnEvent 注册事件处理器
func (ss *SmartScheduler) OnEvent(eventType string, handler EventHandler) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.eventHandlers[eventType] = append(ss.eventHandlers[eventType], handler)
}

// GetStats 获取调度器统计信息
func (ss *SmartScheduler) GetStats() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["running"] = ss.running
	stats["queue_size"] = len(ss.taskQueue)
	stats["worker_pool_size"] = ss.config.WorkerPoolSize

	taskStats := make(map[string]interface{})
	for taskType, stat := range ss.runStats {
		taskStats[taskType] = map[string]interface{}{
			"total_runs":    stat.TotalRuns,
			"success_count": stat.SuccessCount,
			"failure_count": stat.FailureCount,
			"success_rate":  stat.SuccessRate,
			"avg_duration":  stat.AvgDuration.String(),
			"last_run":      stat.LastRunTime,
		}
	}
	stats["task_stats"] = taskStats

	return stats
}

// taskScheduler 任务调度器 - 核心调度逻辑
func (ss *SmartScheduler) taskScheduler() {
	defer ss.wg.Done()

	// 任务队列，按优先级排序
	var pendingTasks []*ScheduledTask

	for {
		select {
		case task, ok := <-ss.taskQueue:
			if !ok {
				return
			}

			// 添加到待处理队列
			pendingTasks = append(pendingTasks, task)

			// 按优先级排序（优先级高的在前）
			sort.Slice(pendingTasks, func(i, j int) bool {
				return pendingTasks[i].Priority > pendingTasks[j].Priority
			})

		case <-ss.ctx.Done():
			return
		}

		// 处理待处理任务
		for len(pendingTasks) > 0 {
			select {
			case <-ss.ctx.Done():
				return
			default:
			}

			// 检查是否超过并发限制
			if ss.workerPool.Running() >= ss.config.MaxConcurrentTasks {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// 获取最高优先级任务
			task := pendingTasks[0]
			pendingTasks = pendingTasks[1:]

			// 检查任务是否过期
			if time.Now().After(task.Deadline) {
				ss.emitEvent(SchedulerEvent{
					Type:     "task_expired",
					TaskID:   task.ID,
					TaskType: task.Type,
					Error:    fmt.Errorf("task deadline exceeded"),
				})
				continue
			}

			// 提交任务到工作池
			ss.workerPool.Submit(func() {
				ss.executeTask(task)
			})
		}
	}
}

// executeTask 执行任务
func (ss *SmartScheduler) executeTask(task *ScheduledTask) {
	startTime := time.Now()

	// 创建任务上下文
	ctx, cancel := context.WithDeadline(ss.ctx, task.Deadline)
	defer cancel()

	ss.emitEvent(SchedulerEvent{
		Type:     "task_started",
		TaskID:   task.ID,
		TaskType: task.Type,
	})

	// 执行任务
	err := task.Handler(ctx, task)
	duration := time.Since(startTime)

	// 更新统计信息
	ss.updateTaskStats(task.Type, duration, err == nil, err)

	if err != nil {
		task.RetryCount++

		// 检查是否可以重试
		if task.RetryCount < task.MaxRetries {
			// 计算退避延迟
			backoffDelay := time.Duration(math.Pow(2, float64(task.RetryCount))) * time.Second
			if backoffDelay > ss.config.MaxBackoffDelay {
				backoffDelay = ss.config.MaxBackoffDelay
			}

			// 重新调度任务
			ss.ScheduleDelayedTask(task.Type, task.Priority-1, backoffDelay, task.Data, task.Handler)

			ss.emitEvent(SchedulerEvent{
				Type:     "task_retry",
				TaskID:   task.ID,
				TaskType: task.Type,
				Data:     map[string]interface{}{"retry_count": task.RetryCount, "delay": backoffDelay},
			})
		} else {
			ss.emitEvent(SchedulerEvent{
				Type:     "task_failed",
				TaskID:   task.ID,
				TaskType: task.Type,
				Error:    err,
			})
		}
	} else {
		ss.emitEvent(SchedulerEvent{
			Type:     "task_completed",
			TaskID:   task.ID,
			TaskType: task.Type,
			Data:     map[string]interface{}{"duration": duration},
		})
	}
}

// periodicTaskManager 定期任务管理器
func (ss *SmartScheduler) periodicTaskManager() {
	defer ss.wg.Done()

	// 立即执行一次
	ss.schedulePeriodicTasks()

	// 设置定时器
	performanceTicker := time.NewTicker(ss.config.PerformanceUpdateInterval)
	defer performanceTicker.Stop()

	cacheTicker := time.NewTicker(ss.config.CacheWarmupInterval)
	defer cacheTicker.Stop()

	cleanupTicker := time.NewTicker(ss.config.DataCleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-performanceTicker.C:
			ss.schedulePeriodicTasks()

		case <-cacheTicker.C:
			ss.scheduleCacheWarmup()

		case <-cleanupTicker.C:
			ss.scheduleDataCleanup()

		case <-ss.ctx.Done():
			return
		}
	}
}

// schedulePeriodicTasks 调度定期任务
func (ss *SmartScheduler) schedulePeriodicTasks() {
	// 性能更新任务
	ss.ScheduleTask(TaskTypePerformanceUpdate, 8, nil, func(ctx context.Context, task *ScheduledTask) error {
		return ss.server.updateRecommendationPerformanceWithPool(ctx, ss.workerPool)
	})

	// 策略回测任务
	ss.ScheduleTask(TaskTypeStrategyBacktest, 7, nil, func(ctx context.Context, task *ScheduledTask) error {
		return ss.server.updateBacktestFromPerformanceWithPool(ctx, ss.workerPool)
	})
}

// scheduleCacheWarmup 调度缓存预热任务
func (ss *SmartScheduler) scheduleCacheWarmup() {
	ss.ScheduleTask(TaskTypeCacheWarmup, 6, nil, func(ctx context.Context, task *ScheduledTask) error {
		return ss.server.warmupCaches(ctx)
	})
}

// scheduleDataCleanup 调度数据清理任务
func (ss *SmartScheduler) scheduleDataCleanup() {
	ss.ScheduleTask(TaskTypeDataCleanup, 5, nil, func(ctx context.Context, task *ScheduledTask) error {
		return ss.server.cleanupExpiredData(ctx)
	})
}

// performanceMonitor 性能监控器
func (ss *SmartScheduler) performanceMonitor() {
	defer ss.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ss.monitorPerformance()
		case <-ss.ctx.Done():
			return
		}
	}
}

// monitorPerformance 监控性能指标
func (ss *SmartScheduler) monitorPerformance() {
	// 检查队列长度
	queueLength := len(ss.taskQueue)
	if queueLength > 50 {
		log.Printf("[SmartScheduler] 警告: 任务队列长度过高 (%d)", queueLength)
	}

	// 检查工作池状态
	running := ss.workerPool.Running()
	if running >= ss.config.WorkerPoolSize {
		log.Printf("[SmartScheduler] 警告: 工作池负载过高 (%d/%d)", running, ss.config.WorkerPoolSize)
	}

	// 记录性能指标
	ss.emitEvent(SchedulerEvent{
		Type: "performance_metrics",
		Data: map[string]interface{}{
			"queue_length":     queueLength,
			"running_workers":  running,
			"worker_pool_size": ss.config.WorkerPoolSize,
		},
	})
}

// adaptiveAdjuster 自适应调整器
func (ss *SmartScheduler) adaptiveAdjuster() {
	defer ss.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ss.adjustConfiguration()
		case <-ss.ctx.Done():
			return
		}
	}
}

// adjustConfiguration 自适应调整配置
func (ss *SmartScheduler) adjustConfiguration() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// 获取系统负载（简化版）
	systemLoad := ss.getSystemLoad()

	// 根据负载调整工作池大小
	if systemLoad > ss.config.LoadThreshold {
		// 系统负载高，减少并发
		newSize := int(float64(ss.config.WorkerPoolSize) * 0.8)
		if newSize >= 5 {
			ss.config.WorkerPoolSize = newSize
			ss.workerPool.Resize(newSize)
			log.Printf("[SmartScheduler] 系统负载高，调整工作池大小到: %d", newSize)
		}
	} else if systemLoad < ss.config.LoadThreshold*0.5 {
		// 系统负载低，增加并发
		newSize := int(float64(ss.config.WorkerPoolSize) * 1.2)
		if newSize <= 30 {
			ss.config.WorkerPoolSize = newSize
			ss.workerPool.Resize(newSize)
			log.Printf("[SmartScheduler] 系统负载低，调整工作池大小到: %d", newSize)
		}
	}

	// 调整调度间隔
	if systemLoad > ss.config.LoadThreshold {
		// 负载高时，增加间隔
		ss.config.PerformanceUpdateInterval = time.Duration(float64(ss.config.PerformanceUpdateInterval) * 1.5)
		if ss.config.PerformanceUpdateInterval > 30*time.Minute {
			ss.config.PerformanceUpdateInterval = 30 * time.Minute
		}
	} else if systemLoad < ss.config.LoadThreshold*0.3 {
		// 负载低时，减少间隔
		ss.config.PerformanceUpdateInterval = time.Duration(float64(ss.config.PerformanceUpdateInterval) * 0.8)
		if ss.config.PerformanceUpdateInterval < 2*time.Minute {
			ss.config.PerformanceUpdateInterval = 2 * time.Minute
		}
	}
}

// getSystemLoad 获取系统负载（简化版）
func (ss *SmartScheduler) getSystemLoad() float64 {
	// 这里可以集成更复杂的系统监控
	// 目前基于队列长度和工作池使用率估算

	queueLength := len(ss.taskQueue)
	runningWorkers := ss.workerPool.Running()

	load := float64(queueLength)/100.0 + float64(runningWorkers)/float64(ss.config.WorkerPoolSize)
	if load > 1.0 {
		load = 1.0
	}

	return load
}

// eventProcessor 事件处理器
func (ss *SmartScheduler) eventProcessor() {
	defer ss.wg.Done()

	for {
		select {
		case event, ok := <-ss.eventChan:
			if !ok {
				return
			}

			ss.handleEvent(event)

		case <-ss.ctx.Done():
			return
		}
	}
}

// emitEvent 发送事件
func (ss *SmartScheduler) emitEvent(event SchedulerEvent) {
	event.Timestamp = time.Now()

	select {
	case ss.eventChan <- event:
	default:
		// 事件通道已满，丢弃事件
		log.Printf("[SmartScheduler] 事件通道已满，丢弃事件: %s", event.Type)
	}
}

// handleEvent 处理事件
func (ss *SmartScheduler) handleEvent(event SchedulerEvent) {
	ss.mu.RLock()
	handlers := ss.eventHandlers[event.Type]
	ss.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}

	// 默认日志记录
	switch event.Type {
	case "task_failed":
		log.Printf("[SmartScheduler] 任务失败: %s (%s) - %v", event.TaskID, event.TaskType, event.Error)
	case "task_completed":
		if data, ok := event.Data.(map[string]interface{}); ok {
			if duration, ok := data["duration"].(time.Duration); ok {
				log.Printf("[SmartScheduler] 任务完成: %s (%s) - 耗时: %v", event.TaskID, event.TaskType, duration)
			}
		}
	}
}

// updateTaskStats 更新任务统计信息
func (ss *SmartScheduler) updateTaskStats(taskType string, duration time.Duration, success bool, err error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	stat, exists := ss.runStats[taskType]
	if !exists {
		stat = &TaskStats{TaskType: taskType}
		ss.runStats[taskType] = stat
	}

	stat.TotalRuns++
	stat.LastRunTime = time.Now()

	if success {
		stat.SuccessCount++
	} else {
		stat.FailureCount++
		stat.LastError = err
	}

	// 更新平均耗时
	if stat.TotalRuns == 1 {
		stat.AvgDuration = duration
	} else {
		stat.AvgDuration = time.Duration((int64(stat.AvgDuration)*(stat.TotalRuns-1) + int64(duration)) / stat.TotalRuns)
	}

	// 更新成功率
	if stat.TotalRuns > 0 {
		stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.TotalRuns)
	}
}

// generateTaskID 生成任务ID
func generateTaskID(taskType string) string {
	return fmt.Sprintf("%s_%d", taskType, time.Now().UnixNano())
}

// 预定义的任务处理器
func (ss *SmartScheduler) performanceUpdateHandler(ctx context.Context, task *ScheduledTask) error {
	return ss.server.updateRecommendationPerformanceWithPool(ctx, ss.workerPool)
}

func (ss *SmartScheduler) strategyBacktestHandler(ctx context.Context, task *ScheduledTask) error {
	return ss.server.updateBacktestFromPerformanceWithPool(ctx, ss.workerPool)
}

func (ss *SmartScheduler) cacheWarmupHandler(ctx context.Context, task *ScheduledTask) error {
	return ss.server.warmupCaches(ctx)
}

func (ss *SmartScheduler) dataCleanupHandler(ctx context.Context, task *ScheduledTask) error {
	return ss.server.cleanupExpiredData(ctx)
}
