package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// DistributedTask 分布式任务
type DistributedTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // 任务类型: "feature_extraction", "model_training", "prediction", "backtest"
	Priority    int                    `json:"priority"`    // 优先级: 1-10, 10最高
	Data        map[string]interface{} `json:"data"`        // 任务数据
	WorkerID    string                 `json:"worker_id"`   // 分配的工作节点ID
	Status      string                 `json:"status"`      // 状态: "pending", "running", "completed", "failed"
	Result      interface{}            `json:"result"`      // 任务结果
	Error       string                 `json:"error"`       // 错误信息
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at"`
	Timeout     time.Duration          `json:"timeout"`     // 超时时间
}

// WorkerNode 工作节点
type WorkerNode struct {
	ID           string            `json:"id"`
	Address      string            `json:"address"`
	Capabilities map[string]bool   `json:"capabilities"` // 支持的任务类型
	Status       string            `json:"status"`       // "active", "inactive", "busy"
	LastSeen     time.Time         `json:"last_seen"`
	Load         float64           `json:"load"`         // 当前负载 0.0-1.0
	TasksProcessed int             `json:"tasks_processed"`
	mu           sync.RWMutex
}

// DistributedComputingConfig 分布式计算配置
type DistributedComputingConfig struct {
	Enabled           bool          `json:"enabled"`
	MaxWorkers        int           `json:"max_workers"`
	TaskTimeout       time.Duration `json:"task_timeout"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	LoadBalance       bool          `json:"load_balance"`
	TaskQueueSize     int           `json:"task_queue_size"`
	WorkerTimeout     time.Duration `json:"worker_timeout"`
}

// DefaultDistributedConfig 返回默认分布式配置
func DefaultDistributedConfig() DistributedComputingConfig {
	return DistributedComputingConfig{
		Enabled:           true,
		MaxWorkers:        10,
		TaskTimeout:       5 * time.Minute,
		HeartbeatInterval: 30 * time.Second,
		LoadBalance:       true,
		TaskQueueSize:     1000,
		WorkerTimeout:     2 * time.Minute,
	}
}

// DistributedComputingManager 分布式计算管理器
type DistributedComputingManager struct {
	config       DistributedComputingConfig
	workers      map[string]*WorkerNode
	tasks        map[string]*DistributedTask
	taskQueue    chan *DistributedTask
	results      chan *TaskResult
	workerPool   chan *WorkerNode
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID string      `json:"task_id"`
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

// NewDistributedComputingManager 创建分布式计算管理器
func NewDistributedComputingManager(config DistributedComputingConfig) *DistributedComputingManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &DistributedComputingManager{
		config:     config,
		workers:    make(map[string]*WorkerNode),
		tasks:      make(map[string]*DistributedTask),
		taskQueue:  make(chan *DistributedTask, config.TaskQueueSize),
		results:    make(chan *TaskResult, config.TaskQueueSize),
		workerPool: make(chan *WorkerNode, config.MaxWorkers),
		ctx:        ctx,
		cancel:     cancel,
	}

	if config.Enabled {
		manager.start()
		log.Printf("[DISTRIBUTED] 分布式计算管理器已启动")
	}

	return manager
}

// start 启动分布式计算服务
func (dcm *DistributedComputingManager) start() {
	// 启动任务调度器
	dcm.wg.Add(1)
	go dcm.taskScheduler()

	// 启动结果处理器
	dcm.wg.Add(1)
	go dcm.resultProcessor()

	// 启动心跳检测
	dcm.wg.Add(1)
	go dcm.heartbeatChecker()

	log.Printf("[DISTRIBUTED] 分布式计算服务已启动")
}

// stop 停止分布式计算服务
func (dcm *DistributedComputingManager) stop() {
	dcm.cancel()
	close(dcm.taskQueue)
	close(dcm.results)
	dcm.wg.Wait()
	log.Printf("[DISTRIBUTED] 分布式计算服务已停止")
}

// RegisterWorker 注册工作节点
func (dcm *DistributedComputingManager) RegisterWorker(workerID, address string, capabilities map[string]bool) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if len(dcm.workers) >= dcm.config.MaxWorkers {
		return fmt.Errorf("达到最大工作节点数量限制: %d", dcm.config.MaxWorkers)
	}

	worker := &WorkerNode{
		ID:             workerID,
		Address:        address,
		Capabilities:   capabilities,
		Status:         "active",
		LastSeen:       time.Now(),
		Load:           0.0,
		TasksProcessed: 0,
	}

	dcm.workers[workerID] = worker
	select {
	case dcm.workerPool <- worker:
	default:
		// 工作池已满，跳过
	}

	log.Printf("[DISTRIBUTED] 工作节点 %s 已注册，地址: %s", workerID, address)
	return nil
}

// UnregisterWorker 注销工作节点
func (dcm *DistributedComputingManager) UnregisterWorker(workerID string) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if worker, exists := dcm.workers[workerID]; exists {
		worker.Status = "inactive"
		log.Printf("[DISTRIBUTED] 工作节点 %s 已注销", workerID)
	}
}

// SubmitTask 提交分布式任务
func (dcm *DistributedComputingManager) SubmitTask(taskType string, priority int, data map[string]interface{}, timeout time.Duration) (string, error) {
	if !dcm.config.Enabled {
		return "", fmt.Errorf("分布式计算未启用")
	}

	taskID := fmt.Sprintf("task_%d_%s", time.Now().UnixNano(), taskType)

	task := &DistributedTask{
		ID:        taskID,
		Type:      taskType,
		Priority:  priority,
		Data:      data,
		Status:    "pending",
		CreatedAt: time.Now(),
		Timeout:   timeout,
	}

	if task.Timeout == 0 {
		task.Timeout = dcm.config.TaskTimeout
	}

	dcm.mu.Lock()
	dcm.tasks[taskID] = task
	dcm.mu.Unlock()

	select {
	case dcm.taskQueue <- task:
		log.Printf("[DISTRIBUTED] 任务 %s 已提交，类型: %s，优先级: %d", taskID, taskType, priority)
	default:
		return "", fmt.Errorf("任务队列已满")
	}

	return taskID, nil
}

// GetTaskStatus 获取任务状态
func (dcm *DistributedComputingManager) GetTaskStatus(taskID string) (*DistributedTask, error) {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	task, exists := dcm.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", taskID)
	}

	return task, nil
}

// CancelTask 取消任务
func (dcm *DistributedComputingManager) CancelTask(taskID string) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	task, exists := dcm.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	if task.Status == "completed" || task.Status == "failed" {
		return fmt.Errorf("任务 %s 已经完成，无法取消", taskID)
	}

	task.Status = "cancelled"
	log.Printf("[DISTRIBUTED] 任务 %s 已取消", taskID)
	return nil
}

// taskScheduler 任务调度器
func (dcm *DistributedComputingManager) taskScheduler() {
	defer dcm.wg.Done()

	for {
		select {
		case <-dcm.ctx.Done():
			return
		case task := <-dcm.taskQueue:
			if task == nil {
				continue
			}

			// 查找合适的工作节点
			worker := dcm.selectWorker(task)
			if worker == nil {
				// 没有合适的工作节点，重新放回队列
				go func() {
					time.Sleep(1 * time.Second)
					select {
					case dcm.taskQueue <- task:
					default:
						log.Printf("[DISTRIBUTED] 任务 %s 重新入队失败", task.ID)
					}
				}()
				continue
			}

			// 分配任务给工作节点
			dcm.assignTaskToWorker(task, worker)
		}
	}
}

// selectWorker 选择合适的工作节点
func (dcm *DistributedComputingManager) selectWorker(task *DistributedTask) *WorkerNode {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	var candidates []*WorkerNode

	// 筛选有能力处理该任务的工作节点
	for _, worker := range dcm.workers {
		if worker.Status != "active" {
			continue
		}

		// 检查能力匹配
		if capable, exists := worker.Capabilities[task.Type]; !exists || !capable {
			continue
		}

		// 检查负载
		if worker.Load >= 0.9 {
			continue
		}

		candidates = append(candidates, worker)
	}

	if len(candidates) == 0 {
		return nil
	}

	// 如果启用负载均衡，选择负载最轻的节点
	if dcm.config.LoadBalance {
		minLoad := 1.0
		var selectedWorker *WorkerNode

		for _, worker := range candidates {
			if worker.Load < minLoad {
				minLoad = worker.Load
				selectedWorker = worker
			}
		}

		return selectedWorker
	}

	// 否则随机选择
	return candidates[rand.Intn(len(candidates))]
}

// assignTaskToWorker 将任务分配给工作节点
func (dcm *DistributedComputingManager) assignTaskToWorker(task *DistributedTask, worker *WorkerNode) {
	task.WorkerID = worker.ID
	task.Status = "running"
	now := time.Now()
	task.StartedAt = &now

	worker.mu.Lock()
	worker.Load += 0.1 // 增加负载
	if worker.Load > 1.0 {
		worker.Load = 1.0
	}
	worker.mu.Unlock()

	log.Printf("[DISTRIBUTED] 任务 %s 已分配给工作节点 %s", task.ID, worker.ID)

	// 模拟任务执行（实际实现需要通过网络调用工作节点）
	go dcm.executeTask(task, worker)
}

// executeTask 执行任务
func (dcm *DistributedComputingManager) executeTask(task *DistributedTask, worker *WorkerNode) {
	defer func() {
		worker.mu.Lock()
		worker.Load -= 0.1 // 减少负载
		if worker.Load < 0.0 {
			worker.Load = 0.0
		}
		worker.TasksProcessed++
		worker.mu.Unlock()
	}()

	// 设置超时
	ctx, cancel := context.WithTimeout(dcm.ctx, task.Timeout)
	defer cancel()

	// 执行任务（根据任务类型调用相应的处理函数）
	var result interface{}
	var err error

	switch task.Type {
	case "feature_extraction":
		result, err = dcm.executeFeatureExtraction(ctx, task.Data)
	case "model_training":
		result, err = dcm.executeModelTraining(ctx, task.Data)
	case "prediction":
		result, err = dcm.executePrediction(ctx, task.Data)
	case "backtest":
		result, err = dcm.executeBacktest(ctx, task.Data)
	default:
		err = fmt.Errorf("不支持的任务类型: %s", task.Type)
	}

	// 发送结果
	taskResult := &TaskResult{
		TaskID: task.ID,
		Result: result,
	}

	if err != nil {
		taskResult.Error = err.Error()
		task.Status = "failed"
		task.Error = err.Error()
	} else {
		task.Status = "completed"
		task.Result = result
	}

	now := time.Now()
	task.CompletedAt = &now

	select {
	case dcm.results <- taskResult:
	default:
		log.Printf("[DISTRIBUTED] 结果队列已满，任务 %s 结果丢失", task.ID)
	}
}

// resultProcessor 结果处理器
func (dcm *DistributedComputingManager) resultProcessor() {
	defer dcm.wg.Done()

	for {
		select {
		case <-dcm.ctx.Done():
			return
		case result := <-dcm.results:
			if result == nil {
				continue
			}

			log.Printf("[DISTRIBUTED] 任务 %s 执行完成", result.TaskID)

			// 这里可以添加结果的后处理逻辑
			// 比如更新数据库、触发后续任务等
		}
	}
}

// heartbeatChecker 心跳检测器
func (dcm *DistributedComputingManager) heartbeatChecker() {
	defer dcm.wg.Done()

	ticker := time.NewTicker(dcm.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dcm.ctx.Done():
			return
		case <-ticker.C:
			dcm.checkWorkerHealth()
		}
	}
}

// checkWorkerHealth 检查工作节点健康状态
func (dcm *DistributedComputingManager) checkWorkerHealth() {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	now := time.Now()
	timeout := dcm.config.WorkerTimeout

	for id, worker := range dcm.workers {
		if now.Sub(worker.LastSeen) > timeout {
			if worker.Status != "inactive" {
				log.Printf("[DISTRIBUTED] 工作节点 %s 心跳超时，标记为非活跃", id)
				worker.Status = "inactive"
			}
		}
	}
}

// 执行各种任务类型的函数
func (dcm *DistributedComputingManager) executeFeatureExtraction(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	// 模拟特征提取任务
	symbol, ok := data["symbol"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少symbol参数")
	}

	log.Printf("[DISTRIBUTED] 执行特征提取任务: %s", symbol)

	// 这里实现实际的特征提取逻辑
	time.Sleep(2 * time.Second) // 模拟处理时间

	result := map[string]interface{}{
		"symbol":    symbol,
		"features":  []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		"timestamp": time.Now(),
	}

	return result, nil
}

func (dcm *DistributedComputingManager) executeModelTraining(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	modelType, ok := data["model_type"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少model_type参数")
	}

	log.Printf("[DISTRIBUTED] 执行模型训练任务: %s", modelType)

	// 这里实现实际的模型训练逻辑
	time.Sleep(5 * time.Second) // 模拟处理时间

	result := map[string]interface{}{
		"model_type": modelType,
		"accuracy":   0.85,
		"loss":       0.15,
		"timestamp":  time.Now(),
	}

	return result, nil
}

func (dcm *DistributedComputingManager) executePrediction(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	symbol, ok := data["symbol"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少symbol参数")
	}

	log.Printf("[DISTRIBUTED] 执行预测任务: %s", symbol)

	// 这里实现实际的预测逻辑
	time.Sleep(1 * time.Second) // 模拟处理时间

	result := map[string]interface{}{
		"symbol":     symbol,
		"prediction": 0.75,
		"confidence": 0.82,
		"timestamp":  time.Now(),
	}

	return result, nil
}

func (dcm *DistributedComputingManager) executeBacktest(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	strategy, ok := data["strategy"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少strategy参数")
	}

	log.Printf("[DISTRIBUTED] 执行回测任务: %s", strategy)

	// 这里实现实际的回测逻辑
	time.Sleep(10 * time.Second) // 模拟处理时间

	result := map[string]interface{}{
		"strategy":    strategy,
		"total_return": 15.3,
		"sharpe_ratio": 1.2,
		"max_drawdown": -8.5,
		"win_rate":    0.62,
		"timestamp":   time.Now(),
	}

	return result, nil
}

// GetStats 获取分布式计算统计信息
func (dcm *DistributedComputingManager) GetStats() map[string]interface{} {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	activeWorkers := 0
	totalTasks := 0
	completedTasks := 0
	failedTasks := 0

	for _, worker := range dcm.workers {
		if worker.Status == "active" {
			activeWorkers++
		}
		totalTasks += worker.TasksProcessed
	}

	for _, task := range dcm.tasks {
		switch task.Status {
		case "completed":
			completedTasks++
		case "failed":
			failedTasks++
		}
	}

	return map[string]interface{}{
		"enabled":          dcm.config.Enabled,
		"active_workers":   activeWorkers,
		"total_workers":    len(dcm.workers),
		"total_tasks":      totalTasks,
		"completed_tasks":  completedTasks,
		"failed_tasks":     failedTasks,
		"pending_tasks":    len(dcm.taskQueue),
		"queue_size":       dcm.config.TaskQueueSize,
	}
}
