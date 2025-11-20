package server

import (
	"context"
	"sync"
)

// ==================== 并发工具函数 ====================

// ParallelMap 并发执行 map 操作（使用 interface{} 类型，避免泛型）
func ParallelMap(
	ctx context.Context,
	items []interface{},
	maxConcurrency int,
	mapper func(context.Context, interface{}) (interface{}, error),
) ([]interface{}, []error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]interface{}, len(items))
	errors := make([]error, len(items))

	// 使用信号量限制并发数
	sem := make(chan struct{}, maxConcurrency)
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

		go func() {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := mapper(ctx, it)
			mu.Lock()
			results[idx] = result
			errors[idx] = err
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results, errors
}

// ParallelFilter 并发执行 filter 操作（使用 interface{} 类型，避免泛型）
func ParallelFilter(
	ctx context.Context,
	items []interface{},
	maxConcurrency int,
	filter func(context.Context, interface{}) (bool, error),
) ([]interface{}, []error) {
	if len(items) == 0 {
		return nil, nil
	}

	keep := make([]bool, len(items))
	errors := make([]error, len(items))

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range items {
		select {
		case <-ctx.Done():
			return nil, []error{ctx.Err()}
		default:
		}

		wg.Add(1)
		idx := i
		it := item

		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			ok, err := filter(ctx, it)
			mu.Lock()
			keep[idx] = ok
			errors[idx] = err
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 收集保留的项目
	result := make([]interface{}, 0, len(items))
	for i, k := range keep {
		if k {
			result = append(result, items[i])
		}
	}

	return result, errors
}

// ParallelForEach 并发执行 forEach 操作（使用 interface{} 类型，避免泛型）
func ParallelForEach(
	ctx context.Context,
	items []interface{},
	maxConcurrency int,
	action func(context.Context, interface{}) error,
) []error {
	if len(items) == 0 {
		return nil
	}

	errors := make([]error, len(items))

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range items {
		select {
		case <-ctx.Done():
			for j := i; j < len(items); j++ {
				errors[j] = ctx.Err()
			}
			return errors
		default:
		}

		wg.Add(1)
		idx := i
		it := item

		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			err := action(ctx, it)
			mu.Lock()
			errors[idx] = err
			mu.Unlock()
		}()
	}

	wg.Wait()
	return errors
}
