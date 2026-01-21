package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ============================================================================
// 优雅关闭管理器 - 确保系统平稳退出
// ============================================================================

// ShutdownCallback 关闭回调函数
type ShutdownCallback func(ctx context.Context) error

// ShutdownPhase 关闭阶段
type ShutdownPhase int

const (
	PhasePreShutdown  ShutdownPhase = iota // 预关闭阶段
	PhaseShutdown                          // 关闭阶段
	PhasePostShutdown                      // 后关闭阶段
)

// ShutdownManager 优雅关闭管理器
type ShutdownManager struct {
	// 回调函数，按阶段分组
	callbacks map[ShutdownPhase][]ShutdownCallback

	// 超时设置
	phasesTimeout map[ShutdownPhase]time.Duration

	// 状态
	isShuttingDown int32
	shutdownCh     chan struct{}
	doneCh         chan struct{}

	// 同步
	mu sync.RWMutex

	// 信号处理
	signalCh chan os.Signal
}

// NewShutdownManager 创建关闭管理器
func NewShutdownManager() *ShutdownManager {
	sm := &ShutdownManager{
		callbacks: make(map[ShutdownPhase][]ShutdownCallback),
		phasesTimeout: map[ShutdownPhase]time.Duration{
			PhasePreShutdown:  5 * time.Second,
			PhaseShutdown:     30 * time.Second,
			PhasePostShutdown: 10 * time.Second,
		},
		shutdownCh: make(chan struct{}),
		doneCh:     make(chan struct{}),
		signalCh:   make(chan os.Signal, 1),
	}

	// 注册系统信号处理
	signal.Notify(sm.signalCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动信号监听协程
	go sm.signalHandler()

	return sm
}

// RegisterCallback 注册关闭回调函数
func (sm *ShutdownManager) RegisterCallback(phase ShutdownPhase, callback ShutdownCallback) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.callbacks[phase] = append(sm.callbacks[phase], callback)
}

// SetPhaseTimeout 设置阶段超时时间
func (sm *ShutdownManager) SetPhaseTimeout(phase ShutdownPhase, timeout time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.phasesTimeout[phase] = timeout
}

// WaitShutdown 等待关闭信号
func (sm *ShutdownManager) WaitShutdown() {
	<-sm.shutdownCh
}

// Shutdown 执行优雅关闭
func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&sm.isShuttingDown, 0, 1) {
		// 已经在关闭中
		return nil
	}

	log.Println("[ShutdownManager] 开始优雅关闭...")

	// 关闭shutdownCh，通知所有等待者
	close(sm.shutdownCh)

	var errors []error

	// 执行各阶段的关闭回调
	phases := []ShutdownPhase{PhasePreShutdown, PhaseShutdown, PhasePostShutdown}

	for _, phase := range phases {
		if err := sm.executePhase(ctx, phase); err != nil {
			errors = append(errors, err)
			log.Printf("[ShutdownManager] 阶段 %v 执行失败: %v", phase, err)
		}
	}

	// 关闭完成
	close(sm.doneCh)
	log.Println("[ShutdownManager] 优雅关闭完成")

	if len(errors) > 0 {
		return &ShutdownError{Errors: errors}
	}

	return nil
}

// executePhase 执行单个阶段
func (sm *ShutdownManager) executePhase(ctx context.Context, phase ShutdownPhase) error {
	sm.mu.RLock()
	callbacks := make([]ShutdownCallback, len(sm.callbacks[phase]))
	copy(callbacks, sm.callbacks[phase])
	timeout := sm.phasesTimeout[phase]
	sm.mu.RUnlock()

	if len(callbacks) == 0 {
		log.Printf("[ShutdownManager] 阶段 %v 没有注册回调，跳过", phase)
		return nil
	}

	log.Printf("[ShutdownManager] 执行阶段 %v，包含 %d 个回调", phase, len(callbacks))

	// 创建阶段上下文
	phaseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 并发执行所有回调
	errCh := make(chan error, len(callbacks))
	var wg sync.WaitGroup

	for i, callback := range callbacks {
		wg.Add(1)
		go func(idx int, cb ShutdownCallback) {
			defer wg.Done()

			log.Printf("[ShutdownManager] 执行阶段 %v 的回调 %d", phase, idx)

			// 执行回调
			if err := cb(phaseCtx); err != nil {
				errCh <- &CallbackError{
					Phase: phase,
					Index: idx,
					Err:   err,
				}
			}
		}(i, callback)
	}

	// 等待所有回调完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有回调都完成了
		close(errCh)
	case <-phaseCtx.Done():
		// 超时了
		return &PhaseTimeoutError{
			Phase:   phase,
			Timeout: timeout,
		}
	}

	// 收集错误
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return &PhaseExecutionError{
			Phase:  phase,
			Errors: errors,
		}
	}

	log.Printf("[ShutdownManager] 阶段 %v 执行完成", phase)
	return nil
}

// signalHandler 信号处理器
func (sm *ShutdownManager) signalHandler() {
	sig := <-sm.signalCh
	log.Printf("[ShutdownManager] 收到关闭信号: %v", sig)

	// 异步执行关闭
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := sm.Shutdown(ctx); err != nil {
			log.Printf("[ShutdownManager] 优雅关闭失败: %v", err)
			os.Exit(1)
		}
	}()
}

// IsShuttingDown 检查是否正在关闭
func (sm *ShutdownManager) IsShuttingDown() bool {
	return atomic.LoadInt32(&sm.isShuttingDown) == 1
}

// WaitDone 等待关闭完成
func (sm *ShutdownManager) WaitDone() {
	<-sm.doneCh
}

// 错误类型定义

// ShutdownError 关闭错误
type ShutdownError struct {
	Errors []error
}

func (e *ShutdownError) Error() string {
	return "shutdown completed with errors"
}

// PhaseTimeoutError 阶段超时错误
type PhaseTimeoutError struct {
	Phase   ShutdownPhase
	Timeout time.Duration
}

func (e *PhaseTimeoutError) Error() string {
	return fmt.Sprintf("phase %v timeout after %v", e.Phase, e.Timeout)
}

// PhaseExecutionError 阶段执行错误
type PhaseExecutionError struct {
	Phase  ShutdownPhase
	Errors []error
}

func (e *PhaseExecutionError) Error() string {
	return fmt.Sprintf("phase %v execution failed with %d errors", e.Phase, len(e.Errors))
}

// CallbackError 回调错误
type CallbackError struct {
	Phase ShutdownPhase
	Index int
	Err   error
}

func (e *CallbackError) Error() string {
	return fmt.Sprintf("callback %d in phase %v failed: %v", e.Index, e.Phase, e.Err)
}

// ============================================================================
// 资源清理器 - 管理各种资源的安全清理
// ============================================================================

// ResourceCleaner 资源清理器
type ResourceCleaner struct {
	resources []ResourceCleanup
	mu        sync.RWMutex
}

// ResourceCleanup 资源清理接口
type ResourceCleanup interface {
	Name() string
	Cleanup(ctx context.Context) error
	Priority() int // 清理优先级，数字越大越先清理
}

// NewResourceCleaner 创建资源清理器
func NewResourceCleaner() *ResourceCleaner {
	return &ResourceCleaner{
		resources: make([]ResourceCleanup, 0),
	}
}

// Register 注册资源清理器
func (rc *ResourceCleaner) Register(cleanup ResourceCleanup) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.resources = append(rc.resources, cleanup)

	// 按优先级排序（降序）
	for i := len(rc.resources) - 1; i > 0; i-- {
		if rc.resources[i].Priority() > rc.resources[i-1].Priority() {
			rc.resources[i], rc.resources[i-1] = rc.resources[i-1], rc.resources[i]
		} else {
			break
		}
	}
}

// Cleanup 清理所有资源
func (rc *ResourceCleaner) Cleanup(ctx context.Context) error {
	rc.mu.RLock()
	resources := make([]ResourceCleanup, len(rc.resources))
	copy(resources, rc.resources)
	rc.mu.RUnlock()

	log.Printf("[ResourceCleaner] 开始清理 %d 个资源", len(resources))

	var errors []error

	for _, resource := range resources {
		log.Printf("[ResourceCleaner] 清理资源: %s", resource.Name())

		if err := resource.Cleanup(ctx); err != nil {
			log.Printf("[ResourceCleaner] 资源 %s 清理失败: %v", resource.Name(), err)
			errors = append(errors, &ResourceCleanupError{
				Resource: resource.Name(),
				Err:      err,
			})
		} else {
			log.Printf("[ResourceCleaner] 资源 %s 清理完成", resource.Name())
		}
	}

	if len(errors) > 0 {
		return &ResourceCleanupErrors{Errors: errors}
	}

	return nil
}

// ResourceCleanupError 资源清理错误
type ResourceCleanupError struct {
	Resource string
	Err      error
}

func (e *ResourceCleanupError) Error() string {
	return fmt.Sprintf("failed to cleanup resource %s: %v", e.Resource, e.Err)
}

// ResourceCleanupErrors 多个资源清理错误
type ResourceCleanupErrors struct {
	Errors []error
}

func (e *ResourceCleanupErrors) Error() string {
	return fmt.Sprintf("resource cleanup failed with %d errors", len(e.Errors))
}

// ============================================================================
// 预定义资源清理器实现
// ============================================================================

// DatabaseCleanup 数据库清理器
type DatabaseCleanup struct {
	name        string
	cleanupFunc func(ctx context.Context) error
	priority    int
}

func NewDatabaseCleanup(name string, cleanupFunc func(ctx context.Context) error) *DatabaseCleanup {
	return &DatabaseCleanup{
		name:        name,
		cleanupFunc: cleanupFunc,
		priority:    100, // 高优先级
	}
}

func (dc *DatabaseCleanup) Name() string {
	return dc.name
}

func (dc *DatabaseCleanup) Cleanup(ctx context.Context) error {
	return dc.cleanupFunc(ctx)
}

func (dc *DatabaseCleanup) Priority() int {
	return dc.priority
}

// CacheCleanup 缓存清理器
type CacheCleanup struct {
	name        string
	cleanupFunc func(ctx context.Context) error
	priority    int
}

func NewCacheCleanup(name string, cleanupFunc func(ctx context.Context) error) *CacheCleanup {
	return &CacheCleanup{
		name:        name,
		cleanupFunc: cleanupFunc,
		priority:    90, // 高优先级
	}
}

func (cc *CacheCleanup) Name() string {
	return cc.name
}

func (cc *CacheCleanup) Cleanup(ctx context.Context) error {
	return cc.cleanupFunc(ctx)
}

func (cc *CacheCleanup) Priority() int {
	return cc.priority
}

// WorkerPoolCleanup 工作池清理器
type WorkerPoolCleanup struct {
	name        string
	cleanupFunc func(ctx context.Context) error
	priority    int
}

func NewWorkerPoolCleanup(name string, cleanupFunc func(ctx context.Context) error) *WorkerPoolCleanup {
	return &WorkerPoolCleanup{
		name:        name,
		cleanupFunc: cleanupFunc,
		priority:    80, // 中高优先级
	}
}

func (wpc *WorkerPoolCleanup) Name() string {
	return wpc.name
}

func (wpc *WorkerPoolCleanup) Cleanup(ctx context.Context) error {
	return wpc.cleanupFunc(ctx)
}

func (wpc *WorkerPoolCleanup) Priority() int {
	return wpc.priority
}

// 全局实例
var (
	globalShutdownManager *ShutdownManager
	globalResourceCleaner *ResourceCleaner
	shutdownOnce          sync.Once
	cleanerOnce           sync.Once
)

// GetGlobalShutdownManager 获取全局关闭管理器
func GetGlobalShutdownManager() *ShutdownManager {
	shutdownOnce.Do(func() {
		globalShutdownManager = NewShutdownManager()
	})
	return globalShutdownManager
}

// GetGlobalResourceCleaner 获取全局资源清理器
func GetGlobalResourceCleaner() *ResourceCleaner {
	cleanerOnce.Do(func() {
		globalResourceCleaner = NewResourceCleaner()
	})
	return globalResourceCleaner
}
