package analysis

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"analysis/internal/util"

	pdb "analysis/internal/db"
)

// PerformanceTracker 推荐表现追踪调度器
type PerformanceTracker struct {
	server     ServerInterface
	ctx        context.Context
	workerPool WorkerPoolInterface // 协程池，限制并发数
}

// ServerInterface 服务接口
type ServerInterface interface {
	UpdateRecommendationPerformance(ctx context.Context) error
	UpdateBacktestFromPerformance(ctx context.Context) error
	GetCurrentPrice(ctx context.Context, symbol, kind string) (float64, error)
	FetchBinanceKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]KlineDataAPI, error)
	FetchBinanceKlinesWithTimeRange(ctx context.Context, symbol, kind, interval string, limit int, startTime, endTime *time.Time) ([]KlineDataAPI, error)
}

// ServerAdapter 服务器适配器
type ServerAdapter struct {
	server interface{} // 实际的Server实例
}

// NewServerAdapter 创建服务器适配器
func NewServerAdapter(server interface{}) *ServerAdapter {
	return &ServerAdapter{server: server}
}

// 这里需要根据实际的Server方法来实现接口方法
// 暂时保持空实现，需要在Server中添加相应方法

// WorkerPoolInterface 协程池接口
type WorkerPoolInterface interface {
	Submit(func())
}

// NewPerformanceTracker 创建表现追踪调度器
func NewPerformanceTracker(server ServerInterface) *PerformanceTracker {
	return &PerformanceTracker{
		server:     server,
		ctx:        context.Background(),
		workerPool: nil, // 可以传入具体的协程池实现
	}
}

// SetWorkerPool 设置协程池
func (pt *PerformanceTracker) SetWorkerPool(pool WorkerPoolInterface) {
	pt.workerPool = pool
}

// Start 启动定期更新任务（每10分钟更新一次）
func (pt *PerformanceTracker) Start() {
	go pt.loop()
}

func (pt *PerformanceTracker) loop() {
	// 启动时先执行一次
	pt.tick()

	// 每10分钟执行一次
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pt.tick()
	}
}

func (pt *PerformanceTracker) tick() {
	log.Printf("[PerformanceTracker] 开始更新推荐表现追踪")
	if err := pt.server.UpdateRecommendationPerformance(pt.ctx); err != nil {
		log.Printf("[PerformanceTracker] 更新失败: %v", err)
	}

	// 同时更新回测数据（统一处理）
	log.Printf("[PerformanceTracker] 开始更新回测数据")
	if err := pt.server.UpdateBacktestFromPerformance(pt.ctx); err != nil {
		log.Printf("[PerformanceTracker] 回测更新失败: %v", err)
	}
}

// UpdateRecommendationPerformance 更新推荐表现追踪（定期调用）
// 职责：只更新实时价格和当前收益率，不更新历史时间点的价格（24h/7d/30d）
func (pt *PerformanceTracker) UpdateRecommendationPerformance(ctx context.Context, db Database) error {
	return pt.updateRecommendationPerformanceWithPool(ctx, nil, db)
}

// updateRecommendationPerformanceWithPool 使用协程池并发更新推荐表现
func (pt *PerformanceTracker) updateRecommendationPerformanceWithPool(ctx context.Context, workerPool WorkerPoolInterface, db Database) error {
	// 使用统一查询函数，获取需要实时更新和回测更新的记录
	realtimePerfs, backtestPerfs, err := pdb.GetPerformancesNeedingUpdate(db.DB(), 100)
	if err != nil {
		return fmt.Errorf("获取需要更新的记录失败: %w", err)
	}

	totalRecords := len(realtimePerfs) + len(backtestPerfs)
	if totalRecords == 0 {
		return nil
	}

	log.Printf("[PerformanceTracker] 开始更新 %d 条实时记录和 %d 条回测记录", len(realtimePerfs), len(backtestPerfs))

	now := time.Now().UTC()
	var mu sync.Mutex
	updatedCount := 0
	failedCount := 0

	// 如果没有提供协程池，使用默认的串行处理
	if workerPool == nil {
		return pt.updateRecommendationPerformanceSerial(ctx, realtimePerfs, now, db)
	}

	// 使用协程池并发处理
	var wg sync.WaitGroup
	for _, perf := range realtimePerfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := pt.updateOneRecommendationPerformance(ctx, perf, now, db); err != nil {
				mu.Lock()
				failedCount++
				mu.Unlock()
				log.Printf("[PerformanceTracker] 实时更新失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			} else {
				mu.Lock()
				updatedCount++
				mu.Unlock()
			}
		})
	}

	for _, perf := range backtestPerfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := pt.updateOneBacktestPerformance(ctx, perf, now, db); err != nil {
				mu.Lock()
				failedCount++
				mu.Unlock()
				log.Printf("[PerformanceTracker] 回测更新失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			} else {
				mu.Lock()
				updatedCount++
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	log.Printf("[PerformanceTracker] 更新完成: 成功 %d 条, 失败 %d 条", updatedCount, failedCount)
	return nil
}

// updateRecommendationPerformanceSerial 串行更新（用于没有协程池的情况）
func (pt *PerformanceTracker) updateRecommendationPerformanceSerial(ctx context.Context, perfs []pdb.RecommendationPerformance, now time.Time, db Database) error {
	updatedCount := 0
	for _, perf := range perfs {
		if err := pt.updateOneRecommendationPerformance(ctx, perf, now, db); err != nil {
			log.Printf("[PerformanceTracker] 更新记录失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			continue
		}
		updatedCount++
	}
	log.Printf("[PerformanceTracker] 成功更新 %d 条记录", updatedCount)
	return nil
}

// updateOneRecommendationPerformance 更新单条推荐表现记录
func (pt *PerformanceTracker) updateOneRecommendationPerformance(ctx context.Context, perf pdb.RecommendationPerformance, now time.Time, db Database) error {
	gormDB := db.DB()

	// 计算推荐后的时间差
	timeSinceRecommendation := now.Sub(perf.RecommendedAt)

	// 获取当前价格（带缓存机制）
	currentPrice, err := pt.getCachedPrice(ctx, perf.Symbol, perf.Kind, now)
	if err != nil {
		return fmt.Errorf("获取 %s 当前价格失败: %w", perf.Symbol, err)
	}

	// 更新当前价格和收益率
	perf.CurrentPrice = &currentPrice
	currentReturn := ((currentPrice - perf.RecommendedPrice) / perf.RecommendedPrice) * 100
	perf.CurrentReturn = &currentReturn

	// 只更新1h价格（使用实时价格，因为1h是短期数据）
	// 注意：24h/7d/30d价格由 UpdateBacktestFromPerformance 使用历史价格更新
	if timeSinceRecommendation >= 1*time.Hour && perf.Price1h == nil {
		perf.Price1h = &currentPrice
		return1h := currentReturn
		perf.Return1h = &return1h
	}

	// 如果已经过了24小时但Return24h还是nil，说明UpdateBacktestFromPerformance还没有更新
	// 这里不主动更新历史价格（保持职责分离），但可以记录日志提醒
	if timeSinceRecommendation >= 24*time.Hour && perf.Return24h == nil {
		log.Printf("[PerformanceTracker] 警告: 推荐 %d (%s) 已过24小时但Return24h仍为nil，等待UpdateBacktestFromPerformance更新", perf.ID, perf.Symbol)
	}

	// 如果24h历史价格已更新（由回测函数更新），则更新业务逻辑字段
	// 注意：这里不更新Price24h，只更新基于历史价格计算的业务字段
	if perf.Return24h != nil && perf.IsWin == nil {
		// 更新是否盈利（基于历史价格计算的24h收益率）
		isWin := *perf.Return24h > 0
		perf.IsWin = &isWin

		// 更新表现评级（基于历史价格计算的24h收益率）
		rating := pt.calculatePerformanceRating(*perf.Return24h)
		perf.PerformanceRating = &rating
	}

	// 如果30天历史价格已更新，标记为已完成
	if perf.Return30d != nil && perf.Status != "completed" {
		perf.Status = "completed"
		completedAt := now
		perf.CompletedAt = &completedAt
	}

	// 更新最大涨幅和最大回撤（基于实时价格）
	if perf.MaxGain == nil || currentReturn > *perf.MaxGain {
		perf.MaxGain = &currentReturn
	}
	if perf.MaxDrawdown == nil || currentReturn < *perf.MaxDrawdown {
		perf.MaxDrawdown = &currentReturn
	}

	// 更新最后更新时间
	lastUpdated := now
	perf.LastUpdatedAt = &lastUpdated

	// 保存更新（带重试）
	saveRetryConfig := util.RetryConfig{
		MaxRetries:   2,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	err = util.Retry(ctx, func() error {
		return pdb.UpdateRecommendationPerformance(gormDB, &perf)
	}, &saveRetryConfig)

	if err != nil {
		return fmt.Errorf("保存更新失败（已重试）: %w", err)
	}

	return nil
}

// calculatePerformanceRating 计算表现评级
func (pt *PerformanceTracker) calculatePerformanceRating(return24h float64) string {
	if return24h >= 20 {
		return "excellent"
	} else if return24h >= 10 {
		return "good"
	} else if return24h >= 0 {
		return "average"
	} else {
		return "poor"
	}
}

// CreatePerformanceTracking 为推荐创建表现追踪记录
func (pt *PerformanceTracker) CreatePerformanceTracking(ctx context.Context, rec interface{}, db Database) error {
	// 这里需要适配推荐记录的类型
	// 暂时简化实现
	return fmt.Errorf("需要实现推荐记录类型适配")
}

// getCachedPrice 获取带缓存的价格数据
func (pt *PerformanceTracker) getCachedPrice(ctx context.Context, symbol, kind string, now time.Time) (float64, error) {
	// 这里应该实现价格缓存逻辑
	// 暂时直接获取价格
	return pt.server.GetCurrentPrice(ctx, symbol, kind)
}

// UpdateBacktestFromPerformance 从表现追踪更新回测数据
func (pt *PerformanceTracker) UpdateBacktestFromPerformance(ctx context.Context, db Database) error {
	return pt.updateBacktestFromPerformanceWithPool(ctx, nil, db)
}

// updateBacktestFromPerformanceWithPool 使用协程池并发更新回测数据
func (pt *PerformanceTracker) updateBacktestFromPerformanceWithPool(ctx context.Context, workerPool WorkerPoolInterface, db Database) error {
	// 获取待更新的回测记录
	perfs, err := pdb.GetPendingBacktests(db.DB(), 50)
	if err != nil {
		return fmt.Errorf("获取待更新回测记录失败: %w", err)
	}

	if len(perfs) == 0 {
		return nil
	}

	log.Printf("[BacktestUpdater] 开始更新 %d 条回测记录", len(perfs))

	now := time.Now().UTC()
	var mu sync.Mutex
	updatedCount := 0
	failedCount := 0

	// 如果没有提供协程池，使用默认的串行处理
	if workerPool == nil {
		return pt.updateBacktestFromPerformanceSerial(ctx, perfs, now, db)
	}

	// 使用协程池并发处理
	var wg sync.WaitGroup
	for _, perf := range perfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := pt.updateOneBacktestPerformance(ctx, perf, now, db); err != nil {
				mu.Lock()
				failedCount++
				mu.Unlock()
				log.Printf("[BacktestUpdater] 更新记录失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			} else {
				mu.Lock()
				updatedCount++
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	log.Printf("[BacktestUpdater] 更新完成: 成功 %d 条, 失败 %d 条", updatedCount, failedCount)
	return nil
}

// updateBacktestFromPerformanceSerial 串行更新（用于没有协程池的情况）
func (pt *PerformanceTracker) updateBacktestFromPerformanceSerial(ctx context.Context, perfs []pdb.RecommendationPerformance, now time.Time, db Database) error {
	updatedCount := 0
	for _, perf := range perfs {
		if err := pt.updateOneBacktestPerformance(ctx, perf, now, db); err != nil {
			log.Printf("[BacktestUpdater] 更新记录失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			continue
		}
		updatedCount++
	}
	log.Printf("[BacktestUpdater] 成功更新 %d 条记录", updatedCount)
	return nil
}

// updateOneBacktestPerformance 更新单条回测记录
func (pt *PerformanceTracker) updateOneBacktestPerformance(ctx context.Context, perf pdb.RecommendationPerformance, now time.Time, db Database) error {
	// 检查配置
	// 注意：现在使用Binance API替代CoinGecko，不需要CoinGecko ID
	// Binance API完全免费，直接使用交易对符号即可

	// 计算需要更新的时间点
	recommendedAt := perf.RecommendedAt.UTC()
	time24h := recommendedAt.Add(24 * time.Hour)
	time7d := recommendedAt.Add(7 * 24 * time.Hour)
	time30d := recommendedAt.Add(30 * 24 * time.Hour)

	// 检查哪些时间点需要更新
	needUpdate24h := now.After(time24h) && perf.Price24h == nil
	needUpdate7d := now.After(time7d) && perf.Price7d == nil
	needUpdate30d := now.After(time30d) && perf.Price30d == nil

	if !needUpdate24h && !needUpdate7d && !needUpdate30d {
		return nil // 无需更新
	}

	// 使用Binance API获取历史价格（免费替代CoinGecko）
	findPriceAtTime := func(targetTime time.Time) *float64 {
		// 计算需要获取的K线数量和时间间隔
		timeDiff := targetTime.Sub(recommendedAt)
		var interval string
		var intervalHours float64

		// 根据时间差选择合适的K线间隔
		if timeDiff <= 7*24*time.Hour {
			// 7天内使用1小时K线
			interval = "1h"
			intervalHours = 1.0
		} else if timeDiff <= 30*24*time.Hour {
			// 30天内使用4小时K线
			interval = "4h"
			intervalHours = 4.0
		} else {
			// 超过30天使用日K线
			interval = "1d"
			intervalHours = 24.0
		}

		// 计算从推荐时间到目标时间需要多少根K线
		requiredKlines := int(timeDiff.Hours() / intervalHours)
		if requiredKlines < 1 {
			requiredKlines = 1
		}

		// 使用Binance API的startTime和endTime参数获取指定时间范围的K线
		// 计算时间范围：从推荐时间往前推一些，到目标时间往后推一些
		startTime := recommendedAt.Add(-2 * time.Hour) // 往前推2小时作为缓冲
		endTime := targetTime.Add(2 * time.Hour)       // 往后推2小时作为缓冲

		// 计算需要获取的K线数量
		timeRange := endTime.Sub(startTime)
		actualLimit := int(timeRange.Hours()/intervalHours) + 5 // 多获取5根作为缓冲
		if actualLimit > 1000 {
			actualLimit = 1000
		}
		if actualLimit < 10 {
			actualLimit = 10 // 至少获取10根
		}

		klines, err := pt.server.FetchBinanceKlinesWithTimeRange(ctx, perf.Symbol, perf.Kind, interval, actualLimit, &startTime, &endTime)
		if err != nil {
			log.Printf("[BacktestUpdater] 获取Binance K线数据失败 (Symbol: %s, Interval: %s, Limit: %d): %v", perf.Symbol, interval, actualLimit, err)
			return nil
		}

		if len(klines) == 0 {
			log.Printf("[BacktestUpdater] Binance返回的K线数据为空 (Symbol: %s, Interval: %s)", perf.Symbol, interval)
			return nil
		}
		if err != nil {
			log.Printf("[BacktestUpdater] 获取Binance K线数据失败 (Symbol: %s, Interval: %s, Limit: %d): %v", perf.Symbol, interval, actualLimit, err)
			return nil
		}

		if len(klines) == 0 {
			log.Printf("[BacktestUpdater] Binance返回的K线数据为空 (Symbol: %s, Interval: %s)", perf.Symbol, interval)
			return nil
		}

		// 查找最接近目标时间的K线
		targetUnix := targetTime.Unix() * 1000 // Binance使用毫秒时间戳
		bestIdx := -1
		minDiff := int64(1 << 62)

		// 先查找目标时间之后的K线（最接近的）
		for i, kline := range klines {
			diff := kline.OpenTime - targetUnix
			if diff >= 0 && diff < minDiff {
				minDiff = diff
				bestIdx = i
			}
		}

		// 如果找不到目标时间之后的K线，查找目标时间之前的K线（允许12小时误差）
		if bestIdx < 0 {
			for i, kline := range klines {
				diff := targetUnix - kline.OpenTime
				if diff >= 0 && diff <= 12*3600*1000 {
					bestIdx = i
					break
				}
			}
		}

		if bestIdx >= 0 && bestIdx < len(klines) {
			// 解析字符串价格为float64
			priceStr := klines[bestIdx].Close
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				log.Printf("[BacktestUpdater] 价格解析失败 (Symbol: %s, PriceStr: %s): %v", perf.Symbol, priceStr, err)
				return nil
			}
			log.Printf("[BacktestUpdater] 找到价格 (Symbol: %s, RecommendedAt: %s, TargetTime: %s, Price: %f, KlineTime: %s)",
				perf.Symbol,
				recommendedAt.Format("2006-01-02 15:04:05"),
				targetTime.Format("2006-01-02 15:04:05"),
				price,
				time.Unix(klines[bestIdx].OpenTime/1000, 0).Format("2006-01-02 15:04:05"))
			return &price
		} else {
			firstTime := ""
			lastTime := ""
			if len(klines) > 0 {
				firstTime = time.Unix(klines[0].OpenTime/1000, 0).Format("2006-01-02 15:04:05")
				lastTime = time.Unix(klines[len(klines)-1].OpenTime/1000, 0).Format("2006-01-02 15:04:05")
			}
			log.Printf("[BacktestUpdater] 未找到匹配的K线 (Symbol: %s, RecommendedAt: %s, TargetTime: %s, KlinesCount: %d, FirstKlineTime: %s, LastKlineTime: %s)",
				perf.Symbol,
				recommendedAt.Format("2006-01-02 15:04:05"),
				targetTime.Format("2006-01-02 15:04:05"),
				len(klines),
				firstTime,
				lastTime)
		}

		return nil
	}

	// 更新各时间点的价格和收益率
	recommendedPrice := perf.RecommendedPrice

	if needUpdate24h {
		price24h := findPriceAtTime(time24h)
		if price24h != nil {
			perf.Price24h = price24h
			return24h := ((*price24h - recommendedPrice) / recommendedPrice) * 100
			perf.Return24h = &return24h
		}
	}

	if needUpdate7d {
		price7d := findPriceAtTime(time7d)
		if price7d != nil {
			perf.Price7d = price7d
			return7d := ((*price7d - recommendedPrice) / recommendedPrice) * 100
			perf.Return7d = &return7d
		}
	}

	if needUpdate30d {
		price30d := findPriceAtTime(time30d)
		if price30d != nil {
			perf.Price30d = price30d
			return30d := ((*price30d - recommendedPrice) / recommendedPrice) * 100
			perf.Return30d = &return30d
			perf.BacktestStatus = "completed"
		}
	}

	// 更新回测状态（改进的逻辑）
	timeSinceRecommendation := now.Sub(recommendedAt)

	// 根据时间和数据情况更新状态
	if perf.BacktestStatus == "pending" || perf.BacktestStatus == "" {
		// 如果已经有任何历史价格数据，设置为tracking
		if perf.Price24h != nil || perf.Price7d != nil || perf.Price30d != nil {
			perf.BacktestStatus = "tracking"
		} else if timeSinceRecommendation >= 30*24*time.Hour {
			// 如果已经超过30天仍然没有数据，可能是数据获取失败，设置为failed
			perf.BacktestStatus = "failed"
		} else if timeSinceRecommendation >= 24*time.Hour {
			// 如果超过24小时但还没有24h数据，设置为failed（可能API问题）
			perf.BacktestStatus = "failed"
		}
		// 如果还没到24小时，保持pending状态
	} else if perf.BacktestStatus == "tracking" {
		// 如果已经获取到30天数据，设置为completed
		if perf.Return30d != nil {
			perf.BacktestStatus = "completed"
		} else if timeSinceRecommendation >= 35*24*time.Hour {
			// 如果超过35天仍然没有30天数据，设置为completed（避免无限等待）
			perf.BacktestStatus = "completed"
		}
	}

	// 如果24h历史价格已更新，更新业务逻辑字段（IsWin, PerformanceRating）
	// 注意：IsWin的设置不影响BacktestStatus和Status的设置
	// BacktestStatus和Status是独立的状态字段
	if perf.Return24h != nil {
		// 更新是否盈利（基于历史价格计算的24h收益率）
		// 注意：每次更新时都重新计算IsWin，因为Return24h可能会更新
		isWin := *perf.Return24h > 0
		perf.IsWin = &isWin

		// 更新表现评级（基于历史价格计算的24h收益率）
		rating := pt.calculatePerformanceRating(*perf.Return24h)
		perf.PerformanceRating = &rating
	}

	// 如果30天历史价格已更新，同时更新Status为completed（这是表现追踪状态，不是回测状态）
	if perf.Return30d != nil && perf.Status != "completed" {
		perf.Status = "completed"
		completedAt := now
		perf.CompletedAt = &completedAt
	}

	// 保存更新（带重试）
	saveRetryConfig := util.RetryConfig{
		MaxRetries:   2,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	err := util.Retry(ctx, func() error {
		return pdb.UpdateRecommendationPerformance(db.DB(), &perf)
	}, &saveRetryConfig)

	if err != nil {
		return fmt.Errorf("保存更新失败（已重试）: %w", err)
	}

	return nil
}

// KlineDataAPI API响应使用的K线数据（字符串类型）
type KlineDataAPI struct {
	OpenTime int64  `json:"openTime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume"`
}

// KlineDataNumeric 数值类型的K线数据（用于内部计算）
type KlineDataNumeric struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}
