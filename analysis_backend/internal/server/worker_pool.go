package server

import (
	"context"
	"sync"
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

