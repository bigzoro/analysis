package server

import (
	"context"
	"log"
	"sync"

	pdb "analysis/internal/db"
)

// ConcurrentProcessor 并发处理器
type ConcurrentProcessor struct {
	maxWorkers int
}

// NewConcurrentProcessor 创建并发处理器
func NewConcurrentProcessor(maxWorkers int) *ConcurrentProcessor {
	return &ConcurrentProcessor{
		maxWorkers: maxWorkers,
	}
}

// RecommendationEnhancer 推荐增强器（并发处理价格预测和技术指标）
type RecommendationEnhancer struct {
	server     *Server
	maxWorkers int
}

// NewRecommendationEnhancer 创建推荐增强器
func NewRecommendationEnhancer(server *Server, maxWorkers int) *RecommendationEnhancer {
	return &RecommendationEnhancer{
		server:     server,
		maxWorkers: maxWorkers,
	}
}

// EnhancementResult 增强结果
type EnhancementResult struct {
	Index      int
	Symbol     string
	Prediction interface{}
	Technical  interface{}
	Error      error
}

// EnhanceRecommendations 并发增强推荐数据
func (re *RecommendationEnhancer) EnhanceRecommendations(ctx context.Context, recommendations []map[string]interface{}, kind string) []map[string]interface{} {
	if len(recommendations) == 0 {
		return recommendations
	}

	// 创建工作队列
	jobs := make(chan EnhancementJob, len(recommendations))
	results := make(chan EnhancementResult, len(recommendations))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < re.maxWorkers; i++ {
		wg.Add(1)
		go re.worker(ctx, jobs, results, &wg, kind)
	}

	// 发送工作任务
	go func() {
		defer close(jobs)
		for i, rec := range recommendations {
			symbol, ok := rec["symbol"].(string)
			if ok && symbol != "" {
				jobs <- EnhancementJob{
					Index:  i,
					Symbol: symbol,
					Data:   rec,
				}
			}
		}
	}()

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	resultMap := make(map[int]EnhancementResult)
	for result := range results {
		resultMap[result.Index] = result
	}

	// 应用结果到推荐列表
	for i, rec := range recommendations {
		if result, exists := resultMap[i]; exists {
			if result.Error == nil {
				if result.Prediction != nil {
					rec["prediction"] = result.Prediction
				}
				if result.Technical != nil {
					rec["technical"] = result.Technical
				}
			} else {
				log.Printf("[WARN] Failed to enhance recommendation for %s: %v", result.Symbol, result.Error)
			}
		}
	}

	return recommendations
}

// EnhancementJob 增强任务
type EnhancementJob struct {
	Index  int
	Symbol string
	Data   map[string]interface{}
}

// worker 工作协程
func (re *RecommendationEnhancer) worker(ctx context.Context, jobs <-chan EnhancementJob, results chan<- EnhancementResult, wg *sync.WaitGroup, kind string) {
	defer wg.Done()

	for job := range jobs {
		result := EnhancementResult{
			Index:  job.Index,
			Symbol: job.Symbol,
		}

		// 并发生成价格预测和技术指标
		var predictionMu, technicalMu sync.Mutex
		var prediction, technical interface{}
		var predErr, techErr error

		var innerWg sync.WaitGroup
		innerWg.Add(2)

		// 并发获取价格预测
		go func() {
			defer innerWg.Done()
			pred, err := re.server.GetPricePrediction(ctx, job.Symbol, kind)
			predictionMu.Lock()
			prediction = pred
			predErr = err
			predictionMu.Unlock()
		}()

		// 并发获取技术指标
		go func() {
			defer innerWg.Done()
			tech, err := re.server.GetTechnicalIndicatorsWithSignals(ctx, job.Symbol, kind, 0)
			technicalMu.Lock()
			technical = tech
			techErr = err
			technicalMu.Unlock()
		}()

		innerWg.Wait()

		// 设置结果
		if predErr == nil {
			result.Prediction = prediction
		}
		if techErr == nil {
			result.Technical = technical
		}

		// 如果两个都失败了，才设置错误
		if predErr != nil && techErr != nil {
			result.Error = predErr // 使用第一个错误
		}

		results <- result
	}
}

// BatchPerformanceLoader 批量性能加载器
type BatchPerformanceLoader struct {
	server     *Server
	maxWorkers int
}

// NewBatchPerformanceLoader 创建批量性能加载器
func NewBatchPerformanceLoader(server *Server, maxWorkers int) *BatchPerformanceLoader {
	return &BatchPerformanceLoader{
		server:     server,
		maxWorkers: maxWorkers,
	}
}

// LoadBatchPerformances 批量加载推荐性能数据
func (bpl *BatchPerformanceLoader) LoadBatchPerformances(ctx context.Context, recommendationIds []int) (map[int][]pdb.RecommendationPerformance, error) {
	if len(recommendationIds) == 0 {
		return make(map[int][]pdb.RecommendationPerformance), nil
	}

	// 创建工作队列
	jobs := make(chan int, len(recommendationIds))
	results := make(chan PerformanceResult, len(recommendationIds))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < bpl.maxWorkers; i++ {
		wg.Add(1)
		go bpl.performanceWorker(ctx, jobs, results, &wg)
	}

	// 发送工作任务
	go func() {
		defer close(jobs)
		for _, id := range recommendationIds {
			jobs <- id
		}
	}()

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	performanceMap := make(map[int][]pdb.RecommendationPerformance)
	for result := range results {
		if result.Error == nil && len(result.Performances) > 0 {
			performanceMap[result.RecommendationID] = result.Performances
		}
	}

	return performanceMap, nil
}

// PerformanceResult 性能结果
type PerformanceResult struct {
	RecommendationID int
	Performances     []pdb.RecommendationPerformance
	Error            error
}

// performanceWorker 性能工作协程
func (bpl *BatchPerformanceLoader) performanceWorker(ctx context.Context, jobs <-chan int, results chan<- PerformanceResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for recommendationID := range jobs {
		result := PerformanceResult{
			RecommendationID: recommendationID,
		}

		// 查询性能数据
		perf, err := pdb.GetRecommendationPerformance(bpl.server.db.DB(), uint(recommendationID))
		if err != nil {
			result.Error = err
		} else if perf != nil {
			result.Performances = []pdb.RecommendationPerformance{*perf}
		}

		results <- result
	}
}

// BaseRecommendationData 基础推荐数据（简化版本）
type BaseRecommendationData struct {
	MarketData  []pdb.BinanceMarketTop
	FlowData    map[string]float64
	Blacklist   []string
	MarketState MarketState
	Kind        string
}
