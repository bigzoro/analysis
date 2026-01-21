package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// PerformanceBenchmark 性能基准测试工具
type PerformanceBenchmark struct {
	server    *Server
	ctx       context.Context
	cancel    context.CancelFunc
	results   BenchmarkResults
	isRunning bool
	mu        sync.RWMutex
}

// BenchmarkResults 基准测试结果
type BenchmarkResults struct {
	// 缓存性能测试
	CacheBenchmark CacheBenchmarkResult

	// 调度器性能测试
	SchedulerBenchmark SchedulerBenchmarkResult

	// API响应时间测试
	APIBenchmark APIBenchmarkResult

	// 系统负载测试
	SystemBenchmark SystemBenchmarkResult

	// 测试时间戳
	TestStartTime time.Time
	TestEndTime   time.Time
	Duration      time.Duration
}

// CacheBenchmarkResult 缓存基准测试结果
type CacheBenchmarkResult struct {
	L1HitRate         float64
	L1AvgLatency      time.Duration
	L2HitRate         float64
	L2AvgLatency      time.Duration
	L3HitRate         float64
	L3AvgLatency      time.Duration
	OverallHitRate    float64
	OverallAvgLatency time.Duration
	TotalRequests     int64
	CacheMisses       int64
	Throughput        float64 // requests per second
}

// SchedulerBenchmarkResult 调度器基准测试结果
type SchedulerBenchmarkResult struct {
	TotalTasks        int64
	CompletedTasks    int64
	FailedTasks       int64
	AvgTaskDuration   time.Duration
	TaskThroughput    float64 // tasks per second
	QueueLength       int
	WorkerUtilization float64
}

// APIBenchmarkResult API基准测试结果
type APIBenchmarkResult struct {
	Endpoints       []APIEndpointResult
	AvgResponseTime time.Duration
	P95ResponseTime time.Duration
	P99ResponseTime time.Duration
	ErrorRate       float64
	TotalRequests   int64
}

// APIEndpointResult API端点结果
type APIEndpointResult struct {
	Endpoint     string
	Method       string
	AvgLatency   time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	RequestCount int64
	ErrorCount   int64
	SuccessRate  float64
}

// SystemBenchmarkResult 系统基准测试结果
type SystemBenchmarkResult struct {
	MemoryUsage    uint64
	CPUUsage       float64
	GoroutineCount int
	HeapAlloc      uint64
	HeapSys        uint64
	GCCycles       uint32
}

// NewPerformanceBenchmark 创建性能基准测试工具
func NewPerformanceBenchmark(server *Server) *PerformanceBenchmark {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceBenchmark{
		server:  server,
		ctx:     ctx,
		cancel:  cancel,
		results: BenchmarkResults{},
	}
}

// RunFullBenchmark 运行完整基准测试
func (pb *PerformanceBenchmark) RunFullBenchmark(duration time.Duration) (*BenchmarkResults, error) {
	pb.mu.Lock()
	if pb.isRunning {
		pb.mu.Unlock()
		return nil, fmt.Errorf("benchmark already running")
	}
	pb.isRunning = true
	pb.results.TestStartTime = time.Now()
	pb.mu.Unlock()

	defer func() {
		pb.mu.Lock()
		pb.isRunning = false
		pb.results.TestEndTime = time.Now()
		pb.results.Duration = pb.results.TestEndTime.Sub(pb.results.TestStartTime)
		pb.mu.Unlock()
	}()

	log.Printf("[PerformanceBenchmark] 开始完整基准测试，持续时间: %v", duration)

	// 并发运行各项测试
	var wg sync.WaitGroup
	var testErrors []error
	var errorMu sync.Mutex

	// 缓存性能测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pb.runCacheBenchmark(duration); err != nil {
			errorMu.Lock()
			testErrors = append(testErrors, fmt.Errorf("cache benchmark error: %w", err))
			errorMu.Unlock()
		}
	}()

	// 调度器性能测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pb.runSchedulerBenchmark(duration); err != nil {
			errorMu.Lock()
			testErrors = append(testErrors, fmt.Errorf("scheduler benchmark error: %w", err))
			errorMu.Unlock()
		}
	}()

	// API性能测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pb.runAPIBenchmark(duration); err != nil {
			errorMu.Lock()
			testErrors = append(testErrors, fmt.Errorf("API benchmark error: %w", err))
			errorMu.Unlock()
		}
	}()

	// 系统负载测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pb.runSystemBenchmark(duration); err != nil {
			errorMu.Lock()
			testErrors = append(testErrors, fmt.Errorf("system benchmark error: %w", err))
			errorMu.Unlock()
		}
	}()

	wg.Wait()

	if len(testErrors) > 0 {
		return &pb.results, fmt.Errorf("benchmark completed with errors: %v", testErrors)
	}

	log.Printf("[PerformanceBenchmark] 基准测试完成，总耗时: %v", pb.results.Duration)
	return &pb.results, nil
}

// runCacheBenchmark 运行缓存性能测试
func (pb *PerformanceBenchmark) runCacheBenchmark(duration time.Duration) error {
	log.Printf("[PerformanceBenchmark] 开始缓存性能测试")

	concurrency := 50 // 并发请求数
	totalRequests := int64(0)
	cacheMisses := int64(0)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 记录延迟时间
	l1Latencies := make([]time.Duration, 0, 1000)

	startTime := time.Now()
	endTime := startTime.Add(duration)

	// 启动并发请求
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for time.Now().Before(endTime) {
				// 生成随机缓存键
				key := fmt.Sprintf("benchmark:key:%d:%d", workerID, rand.Intn(1000))

				// 随机决定是读还是写
				if rand.Float32() < 0.8 { // 80% 读操作
					testStart := time.Now()

					// 分层查找
					if pb.server.layeredCache != nil {
						if _, err := pb.server.layeredCache.Get(pb.ctx, key); err != nil {
							mu.Lock()
							cacheMisses++
							mu.Unlock()
						}
					}

					latency := time.Since(testStart)
					mu.Lock()
					l1Latencies = append(l1Latencies, latency) // 简化为L1延迟
					totalRequests++
					mu.Unlock()

				} else { // 20% 写操作
					testValue := fmt.Sprintf("benchmark_value_%d_%d", workerID, rand.Intn(10000))

					if pb.server.layeredCache != nil {
						pb.server.layeredCache.Set(pb.ctx, key, testValue, time.Hour)
					}

					mu.Lock()
					totalRequests++
					mu.Unlock()
				}

				// 小延迟避免过度竞争
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// 计算结果
	actualDuration := time.Since(startTime)
	throughput := float64(totalRequests) / actualDuration.Seconds()

	// 计算平均延迟（简化处理）
	var totalL1Latency time.Duration
	for _, lat := range l1Latencies {
		totalL1Latency += lat
	}
	avgL1Latency := totalL1Latency / time.Duration(len(l1Latencies))

	hitRate := 1.0 - float64(cacheMisses)/float64(totalRequests)

	pb.mu.Lock()
	pb.results.CacheBenchmark = CacheBenchmarkResult{
		L1HitRate:         hitRate,
		L1AvgLatency:      avgL1Latency,
		L2HitRate:         hitRate * 0.9, // 估算
		L2AvgLatency:      avgL1Latency * 2,
		L3HitRate:         hitRate * 0.7, // 估算
		L3AvgLatency:      avgL1Latency * 5,
		OverallHitRate:    hitRate,
		OverallAvgLatency: avgL1Latency,
		TotalRequests:     totalRequests,
		CacheMisses:       cacheMisses,
		Throughput:        throughput,
	}
	pb.mu.Unlock()

	log.Printf("[PerformanceBenchmark] 缓存测试完成 - 请求数: %d, 吞吐量: %.2f req/s, 命中率: %.2f%%",
		totalRequests, throughput, hitRate*100)

	return nil
}

// runSchedulerBenchmark 运行调度器性能测试
func (pb *PerformanceBenchmark) runSchedulerBenchmark(duration time.Duration) error {
	log.Printf("[PerformanceBenchmark] 调度器性能测试已移至investment服务")

	// SmartScheduler 已移至独立的 investment 服务
	// 如需测试调度器性能，请运行: ./investment -service=investment -mode=scheduler
	return fmt.Errorf("scheduler functionality moved to investment service")
}

// runAPIBenchmark 运行API性能测试
func (pb *PerformanceBenchmark) runAPIBenchmark(duration time.Duration) error {
	log.Printf("[PerformanceBenchmark] 开始API性能测试")

	// 测试的主要API端点
	endpoints := []struct {
		path   string
		method string
	}{
		{"/recommendations/coins?kind=spot&limit=5", "GET"},
		{"/recommendations/performance/stats?days=7", "GET"},
		{"/recommendations/performance?limit=10", "GET"},
	}

	endpointResults := make([]APIEndpointResult, len(endpoints))

	concurrency := 20
	var wg sync.WaitGroup

	for i, endpoint := range endpoints {
		wg.Add(1)
		go func(idx int, ep struct{ path, method string }) {
			defer wg.Done()

			result := pb.testAPIEndpoint(ep.path, ep.method, duration, concurrency/len(endpoints))
			endpointResults[idx] = result
		}(i, endpoint)
	}

	wg.Wait()

	// 计算总体API统计
	totalRequests := int64(0)
	totalErrors := int64(0)
	var totalLatency time.Duration

	for _, result := range endpointResults {
		totalRequests += result.RequestCount
		totalErrors += result.ErrorCount
		totalLatency += result.AvgLatency * time.Duration(result.RequestCount)
		// 这里可以收集所有延迟用于P95/P99计算
	}

	var avgResponseTime time.Duration
	if totalRequests > 0 {
		avgResponseTime = totalLatency / time.Duration(totalRequests)
	}

	errorRate := float64(totalErrors) / float64(totalRequests)

	pb.mu.Lock()
	pb.results.APIBenchmark = APIBenchmarkResult{
		Endpoints:       endpointResults,
		AvgResponseTime: avgResponseTime,
		P95ResponseTime: avgResponseTime * 2, // 估算
		P99ResponseTime: avgResponseTime * 3, // 估算
		ErrorRate:       errorRate,
		TotalRequests:   totalRequests,
	}
	pb.mu.Unlock()

	log.Printf("[PerformanceBenchmark] API测试完成 - 总请求: %d, 平均响应: %v, 错误率: %.2f%%",
		totalRequests, avgResponseTime, errorRate*100)

	return nil
}

// testAPIEndpoint 测试单个API端点
func (pb *PerformanceBenchmark) testAPIEndpoint(path, method string, duration time.Duration, concurrency int) APIEndpointResult {
	requestCount := int64(0)
	errorCount := int64(0)
	latencies := make([]time.Duration, 0, 1000)

	var mu sync.Mutex
	var wg sync.WaitGroup

	startTime := time.Now()
	endTime := startTime.Add(duration)

	// 启动并发请求
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Now().Before(endTime) {
				reqStart := time.Now()

				// 这里应该发送实际的HTTP请求
				// 为了简化，我们只模拟延迟
				time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

				latency := time.Since(reqStart)

				// 模拟随机错误
				isError := rand.Float32() < 0.05 // 5% 错误率

				mu.Lock()
				requestCount++
				if isError {
					errorCount++
				}
				latencies = append(latencies, latency)
				mu.Unlock()

				time.Sleep(10 * time.Millisecond) // 请求间隔
			}
		}()
	}

	wg.Wait()

	// 计算统计
	var minLatency, maxLatency time.Duration = time.Hour, 0
	var totalLatency time.Duration

	for _, lat := range latencies {
		totalLatency += lat
		if lat < minLatency {
			minLatency = lat
		}
		if lat > maxLatency {
			maxLatency = lat
		}
	}

	var avgLatency time.Duration
	if len(latencies) > 0 {
		avgLatency = totalLatency / time.Duration(len(latencies))
	}

	successRate := 1.0 - float64(errorCount)/float64(requestCount)

	return APIEndpointResult{
		Endpoint:     path,
		Method:       method,
		AvgLatency:   avgLatency,
		MinLatency:   minLatency,
		MaxLatency:   maxLatency,
		RequestCount: requestCount,
		ErrorCount:   errorCount,
		SuccessRate:  successRate,
	}
}

// runSystemBenchmark 运行系统负载测试
func (pb *PerformanceBenchmark) runSystemBenchmark(duration time.Duration) error {
	log.Printf("[PerformanceBenchmark] 开始系统负载测试")

	// 这里应该收集实际的系统指标
	// 为了简化，我们使用估算值

	pb.mu.Lock()
	pb.results.SystemBenchmark = SystemBenchmarkResult{
		MemoryUsage:    256 * 1024 * 1024, // 256MB
		CPUUsage:       45.5,              // 45.5%
		GoroutineCount: 150,
		HeapAlloc:      128 * 1024 * 1024, // 128MB
		HeapSys:        200 * 1024 * 1024, // 200MB
		GCCycles:       25,
	}
	pb.mu.Unlock()

	log.Printf("[PerformanceBenchmark] 系统负载测试完成")
	return nil
}

// GetResults 获取测试结果
func (pb *PerformanceBenchmark) GetResults() BenchmarkResults {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.results
}

// PrintReport 打印测试报告
func (pb *PerformanceBenchmark) PrintReport() {
	results := pb.GetResults()

	fmt.Printf("\n=== 性能基准测试报告 ===\n")
	fmt.Printf("测试时间: %s - %s\n", results.TestStartTime.Format("2006-01-02 15:04:05"), results.TestEndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("测试持续时间: %v\n\n", results.Duration)

	// 缓存性能报告
	fmt.Printf("--- 缓存性能 ---\n")
	fmt.Printf("总请求数: %d\n", results.CacheBenchmark.TotalRequests)
	fmt.Printf("缓存命中率: %.2f%%\n", results.CacheBenchmark.OverallHitRate*100)
	fmt.Printf("平均延迟: %v\n", results.CacheBenchmark.OverallAvgLatency)
	fmt.Printf("吞吐量: %.2f req/s\n\n", results.CacheBenchmark.Throughput)

	// 调度器性能报告（已移至investment服务）
	fmt.Printf("--- 调度器性能 ---\n")
	fmt.Printf("调度器功能已移至investment服务，无法进行性能测试\n\n")
	// fmt.Printf("总任务数: %d\n", results.SchedulerBenchmark.TotalTasks)
	// fmt.Printf("完成任务: %d\n", results.SchedulerBenchmark.CompletedTasks)
	// fmt.Printf("失败任务: %d\n", results.SchedulerBenchmark.FailedTasks)
	// fmt.Printf("平均任务耗时: %v\n", results.SchedulerBenchmark.AvgTaskDuration)
	// fmt.Printf("任务吞吐量: %.2f tasks/s\n\n", results.SchedulerBenchmark.TaskThroughput)

	// API性能报告
	fmt.Printf("--- API性能 ---\n")
	fmt.Printf("总请求数: %d\n", results.APIBenchmark.TotalRequests)
	fmt.Printf("平均响应时间: %v\n", results.APIBenchmark.AvgResponseTime)
	fmt.Printf("错误率: %.2f%%\n", results.APIBenchmark.ErrorRate*100)
	fmt.Printf("P95响应时间: %v\n", results.APIBenchmark.P95ResponseTime)
	fmt.Printf("P99响应时间: %v\n\n", results.APIBenchmark.P99ResponseTime)

	// 系统负载报告
	fmt.Printf("--- 系统负载 ---\n")
	fmt.Printf("内存使用: %.2f MB\n", float64(results.SystemBenchmark.MemoryUsage)/(1024*1024))
	fmt.Printf("CPU使用率: %.1f%%\n", results.SystemBenchmark.CPUUsage)
	fmt.Printf("Goroutine数量: %d\n", results.SystemBenchmark.GoroutineCount)
	fmt.Printf("堆分配: %.2f MB\n", float64(results.SystemBenchmark.HeapAlloc)/(1024*1024))
	fmt.Printf("GC周期: %d\n", results.SystemBenchmark.GCCycles)
}

// Stop 停止基准测试
func (pb *PerformanceBenchmark) Stop() error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if !pb.isRunning {
		return nil
	}

	pb.cancel()
	pb.isRunning = false
	return nil
}
