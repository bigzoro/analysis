package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"analysis/internal/util"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// PriceCache 价格缓存结构
type PriceCache struct {
	mu        sync.RWMutex
	cache     map[string]*CachedPrice
	cacheTime time.Duration
}

type CachedPrice struct {
	Price     float64
	Timestamp time.Time
}

// NewPriceCache 创建价格缓存
func NewPriceCache(cacheTime time.Duration) *PriceCache {
	return &PriceCache{
		cache:     make(map[string]*CachedPrice),
		cacheTime: cacheTime,
	}
}

// Get 获取缓存的价格
func (pc *PriceCache) Get(key string) (float64, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if cached, exists := pc.cache[key]; exists {
		if time.Since(cached.Timestamp) < pc.cacheTime {
			return cached.Price, true
		}
	}
	return 0, false
}

// Set 设置缓存的价格
func (pc *PriceCache) Set(key string, price float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cache[key] = &CachedPrice{
		Price:     price,
		Timestamp: time.Now(),
	}
}

// PerformanceTracker 推荐表现追踪调度器
type PerformanceTracker struct {
	server     *Server
	ctx        context.Context
	workerPool *WorkerPool // 协程池，限制并发数
}

// NewPerformanceTracker 创建表现追踪调度器
func NewPerformanceTracker(s *Server) *PerformanceTracker {
	return &PerformanceTracker{
		server:     s,
		ctx:        context.Background(),
		workerPool: NewWorkerPool(10), // 限制最大并发数为10，避免API限流
	}
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
	if err := pt.server.updateRecommendationPerformanceWithPool(pt.ctx, pt.workerPool); err != nil {
		log.Printf("[PerformanceTracker] 更新失败: %v", err)
	}

	// 同时更新回测数据（统一处理）
	log.Printf("[PerformanceTracker] 开始更新回测数据")
	if err := pt.server.updateBacktestFromPerformanceWithPool(pt.ctx, pt.workerPool); err != nil {
		log.Printf("[PerformanceTracker] 回测更新失败: %v", err)
	}
}

// updateRecommendationPerformanceWithPool 使用协程池并发更新推荐表现
func (s *Server) updateRecommendationPerformanceWithPool(ctx context.Context, workerPool *WorkerPool) error {
	// 使用优化后的统一查询函数，获取需要实时更新和回测更新的记录
	realtimePerfs, backtestPerfs, err := pdb.GetPerformancesNeedingUpdate(s.db.DB(), 100)
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
		return s.updateRecommendationPerformanceSerial(ctx, realtimePerfs, now)
	}

	// 使用协程池并发处理
	var wg sync.WaitGroup

	// 处理实时更新记录（价格更新）
	for _, perf := range realtimePerfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := s.updateOneRecommendationPerformance(ctx, perf, now); err != nil {
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

	// 处理回测更新记录（历史价格更新）
	for _, perf := range backtestPerfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := s.updateOneBacktestPerformance(ctx, perf, now); err != nil {
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
func (s *Server) updateRecommendationPerformanceSerial(ctx context.Context, perfs []pdb.RecommendationPerformance, now time.Time) error {
	updatedCount := 0
	for _, perf := range perfs {
		if err := s.updateOneRecommendationPerformance(ctx, perf, now); err != nil {
			log.Printf("[PerformanceTracker] 更新记录失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			continue
		}
		updatedCount++
	}
	log.Printf("[PerformanceTracker] 成功更新 %d 条记录", updatedCount)
	return nil
}

// updateOneRecommendationPerformance 更新单条推荐表现记录
func (s *Server) updateOneRecommendationPerformance(ctx context.Context, perf pdb.RecommendationPerformance, now time.Time) error {
	// 计算推荐后的时间差
	timeSinceRecommendation := now.Sub(perf.RecommendedAt)

	// 获取当前价格（带缓存机制）
	currentPrice, err := s.getCachedPrice(ctx, perf.Symbol, perf.Kind, now)
	if err != nil {
		return fmt.Errorf("获取 %s 当前价格失败: %w", perf.Symbol, err)
	}

	if err != nil {
		return fmt.Errorf("获取 %s 当前价格失败（已重试）: %w", perf.Symbol, err)
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
		rating := s.calculatePerformanceRating(*perf.Return24h)
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
		return pdb.UpdateRecommendationPerformance(s.db.DB(), &perf)
	}, &saveRetryConfig)

	if err != nil {
		return fmt.Errorf("保存更新失败（已重试）: %w", err)
	}

	return nil
}

// calculatePerformanceRating 计算表现评级
func (s *Server) calculatePerformanceRating(return24h float64) string {
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
func (s *Server) CreatePerformanceTracking(ctx context.Context, rec *pdb.CoinRecommendation) error {
	// 检查是否已存在追踪记录
	existing, err := pdb.GetRecommendationPerformance(s.db.DB(), rec.ID)
	if err == nil && existing != nil {
		// 已存在，不重复创建
		return nil
	}

	// 获取推荐时的价格
	recommendedPrice, err := s.getCurrentPrice(ctx, rec.Symbol, rec.Kind)
	if err != nil {
		return fmt.Errorf("获取推荐时价格失败: %w", err)
	}

	// 创建追踪记录
	perf := &pdb.RecommendationPerformance{
		RecommendationID: rec.ID,
		Symbol:           rec.Symbol,
		BaseSymbol:       rec.BaseSymbol,
		Kind:             rec.Kind,
		RecommendedAt:    rec.GeneratedAt,
		RecommendedPrice: recommendedPrice,
		TotalScore:       rec.TotalScore,
		MarketScore:      rec.MarketScore,
		FlowScore:        rec.FlowScore,
		HeatScore:        rec.HeatScore,
		EventScore:       rec.EventScore,
		SentimentScore:   rec.SentimentScore,
		Status:           "tracking",
		BacktestStatus:   "pending", // 初始回测状态
	}

	if err := pdb.CreateRecommendationPerformance(s.db.DB(), perf); err != nil {
		return fmt.Errorf("创建追踪记录失败: %w", err)
	}

	log.Printf("[PerformanceTracker] 为推荐 %d (%s) 创建追踪记录", rec.ID, rec.Symbol)
	return nil
}

// GetRecommendationPerformanceAPI 获取推荐表现追踪数据
// GET /recommendations/performance?recommendation_id=123&symbol=BTCUSDT&limit=10
func (s *Server) GetRecommendationPerformanceAPI(c *gin.Context) {
	recommendationIDStr := c.Query("recommendation_id")
	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	limit := 10

	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	var perfs []pdb.RecommendationPerformance
	var err error

	if recommendationIDStr != "" {
		// 根据推荐ID查询
		recommendationID, err := strconv.ParseUint(recommendationIDStr, 10, 32)
		if err != nil {
			s.ValidationError(c, "recommendation_id", "无效的推荐ID")
			return
		}
		perf, err := pdb.GetRecommendationPerformance(s.db.DB(), uint(recommendationID))
		if err != nil {
			s.DatabaseError(c, "查询推荐表现", err)
			return
		}
		if perf != nil {
			perfs = []pdb.RecommendationPerformance{*perf}
		} else {
			// 记录不存在，返回空数组
			perfs = []pdb.RecommendationPerformance{}
		}
	} else if symbol != "" {
		// 根据币种查询
		perfs, err = pdb.GetPerformanceBySymbol(s.db.DB(), symbol, limit)
		if err != nil {
			s.DatabaseError(c, "查询币种表现", err)
			return
		}
	} else {
		s.ValidationError(c, "", "必须提供 recommendation_id 或 symbol 参数")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"performances": perfs,
		"total":        len(perfs),
	})
}

// GetBatchRecommendationPerformanceAPI 批量获取推荐表现追踪数据
// GET /recommendations/performance/batch?recommendation_ids=1,2,3,4,5
func (s *Server) GetBatchRecommendationPerformanceAPI(c *gin.Context) {
	idsStr := strings.TrimSpace(c.Query("recommendation_ids"))
	if idsStr == "" {
		s.ValidationError(c, "recommendation_ids", "推荐ID列表不能为空")
		return
	}

	// 解析ID列表
	parts := strings.Split(idsStr, ",")
	ids := make([]uint, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if id, err := strconv.ParseUint(part, 10, 32); err == nil {
			ids = append(ids, uint(id))
		}
	}

	if len(ids) == 0 {
		s.ValidationError(c, "recommendation_ids", "无效的推荐ID列表")
		return
	}

	// 限制批量查询数量
	if len(ids) > 50 {
		s.ValidationError(c, "recommendation_ids", "批量查询数量不能超过50个")
		return
	}

	// 批量查询
	perfs, err := pdb.GetPerformanceByRecommendationIDs(s.db.DB(), ids)
	if err != nil {
		s.DatabaseError(c, "批量查询推荐表现", err)
		return
	}

	// 转换为映射：recommendation_id -> performance
	result := make(map[uint]pdb.RecommendationPerformance)
	for _, perf := range perfs {
		result[perf.RecommendationID] = perf
	}

	c.JSON(http.StatusOK, gin.H{
		"performances": result,
		"total":        len(result),
	})
}

// GetPerformanceStatsAPI 获取表现统计
// GET /recommendations/performance/stats?days=30
func (s *Server) GetPerformanceStatsAPI(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if n, err := strconv.Atoi(daysStr); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	stats, err := pdb.GetPerformanceStats(s.db.DB(), days)
	if err != nil {
		s.DatabaseError(c, "查询表现统计", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFactorPerformanceStatsAPI 获取因子表现统计（用于反馈循环）
// GET /recommendations/performance/factor-stats?days=30
func (s *Server) GetFactorPerformanceStatsAPI(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if n, err := strconv.Atoi(daysStr); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	stats, err := pdb.GetFactorPerformanceStats(s.db.DB(), days)
	if err != nil {
		s.DatabaseError(c, "查询因子表现统计", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GenerateBacktestReportAPI 生成详细的回测报告API
// POST /recommendations/performance/report
func (s *Server) GenerateBacktestReportAPI(c *gin.Context) {
	var request struct {
		PerformanceID uint   `json:"performance_id" binding:"required"`
		ReportType    string `json:"report_type"` // "summary", "detailed", "comparison"
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.ReportType == "" {
		request.ReportType = "summary"
	}

	// 获取表现记录
	perf, err := pdb.GetRecommendationPerformanceByID(s.db.DB(), request.PerformanceID)
	if err != nil {
		s.DatabaseError(c, "查询表现记录", err)
		return
	}
	if perf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "表现记录不存在"})
		return
	}

	report, err := s.generateBacktestReport(perf, request.ReportType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  report,
	})
}

// generateBacktestReport 生成回测报告
func (s *Server) generateBacktestReport(perf *pdb.RecommendationPerformance, reportType string) (map[string]interface{}, error) {
	report := map[string]interface{}{
		"performance_id": perf.ID,
		"symbol":         perf.Symbol,
		"base_symbol":    perf.BaseSymbol,
		"report_type":    reportType,
		"generated_at":   time.Now().UTC(),
	}

	// 基础信息
	report["basic_info"] = map[string]interface{}{
		"recommended_at":    perf.RecommendedAt,
		"recommended_price": perf.RecommendedPrice,
		"total_score":       perf.TotalScore,
		"current_price":     perf.CurrentPrice,
		"current_return":    perf.CurrentReturn,
	}

	// 历史表现数据
	historicalData := map[string]interface{}{}
	if perf.Return24h != nil {
		historicalData["return_24h"] = *perf.Return24h
	}
	if perf.Return7d != nil {
		historicalData["return_7d"] = *perf.Return7d
	}
	if perf.Return30d != nil {
		historicalData["return_30d"] = *perf.Return30d
	}
	report["historical_performance"] = historicalData

	// 风险指标
	riskMetrics := map[string]interface{}{}
	if perf.MaxGain != nil {
		riskMetrics["max_gain"] = *perf.MaxGain
	}
	if perf.MaxDrawdown != nil {
		riskMetrics["max_drawdown"] = *perf.MaxDrawdown
	}
	report["risk_metrics"] = riskMetrics

	// 策略表现（如果有）
	if perf.ActualReturn != nil {
		strategyMetrics := map[string]interface{}{
			"actual_return":          *perf.ActualReturn,
			"holding_period_minutes": perf.HoldingPeriod,
			"exit_reason":            perf.ExitReason,
		}
		if perf.MaxFavorableExcursion != nil {
			strategyMetrics["max_favorable_excursion"] = *perf.MaxFavorableExcursion
		}
		if perf.MaxAdverseExcursion != nil {
			strategyMetrics["max_adverse_excursion"] = *perf.MaxAdverseExcursion
		}
		report["strategy_performance"] = strategyMetrics
	}

	// 评级和状态
	report["rating"] = map[string]interface{}{
		"performance_rating": perf.PerformanceRating,
		"is_win":             perf.IsWin,
		"status":             perf.Status,
		"backtest_status":    perf.BacktestStatus,
	}

	// 根据报告类型添加额外信息
	switch reportType {
	case "detailed":
		report["detailed_analysis"] = s.generateDetailedAnalysis(perf)
	case "comparison":
		report["comparison_analysis"] = s.generateComparisonAnalysis(perf)
	}

	return report, nil
}

// generateDetailedAnalysis 生成详细分析
func (s *Server) generateDetailedAnalysis(perf *pdb.RecommendationPerformance) map[string]interface{} {
	analysis := map[string]interface{}{}

	// 时间序列分析
	timeAnalysis := map[string]interface{}{
		"days_since_recommendation": int(time.Since(perf.RecommendedAt).Hours() / 24),
		"tracking_status":           perf.Status,
		"last_updated":              perf.LastUpdatedAt,
	}
	analysis["time_analysis"] = timeAnalysis

	// 表现趋势分析
	if perf.Return24h != nil && perf.Return7d != nil && perf.Return30d != nil {
		trend := "unknown"
		if *perf.Return30d > *perf.Return7d && *perf.Return7d > *perf.Return24h {
			trend = "improving"
		} else if *perf.Return30d < *perf.Return7d && *perf.Return7d < *perf.Return24h {
			trend = "declining"
		} else {
			trend = "mixed"
		}
		analysis["performance_trend"] = trend
	}

	// 因子贡献分析（基于得分）
	factorAnalysis := map[string]interface{}{
		"market_score":    perf.MarketScore,
		"flow_score":      perf.FlowScore,
		"heat_score":      perf.HeatScore,
		"event_score":     perf.EventScore,
		"sentiment_score": perf.SentimentScore,
	}
	analysis["factor_analysis"] = factorAnalysis

	return analysis
}

// generateComparisonAnalysis 生成对比分析
func (s *Server) generateComparisonAnalysis(perf *pdb.RecommendationPerformance) map[string]interface{} {
	comparison := map[string]interface{}{}

	// 获取相似币种的表现数据进行对比
	similarPerfs, err := pdb.GetPerformanceBySymbol(s.db.DB(), perf.Symbol, 10)
	if err == nil && len(similarPerfs) > 1 {
		// 计算平均表现
		totalReturn24h := 0.0
		count := 0
		for _, p := range similarPerfs {
			if p.Return24h != nil && p.ID != perf.ID {
				totalReturn24h += *p.Return24h
				count++
			}
		}

		if count > 0 {
			avgReturn24h := totalReturn24h / float64(count)
			comparison["peer_average_return_24h"] = avgReturn24h
			if perf.Return24h != nil {
				comparison["vs_peer_performance"] = *perf.Return24h - avgReturn24h
			}
		}
	}

	return comparison
}

// GetPerformanceTrendAPI 获取推荐表现趋势数据（按日期聚合）
// GET /recommendations/performance/trend?days=30&interval=daily
func (s *Server) GetPerformanceTrendAPI(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if n, err := strconv.Atoi(daysStr); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	interval := strings.ToLower(strings.TrimSpace(c.Query("interval")))
	if interval == "" {
		interval = "daily" // 默认按天
	}

	startTime := time.Now().UTC().AddDate(0, 0, -days)

	// 查询推荐表现数据
	var perfs []pdb.RecommendationPerformance
	err := s.db.DB().Model(&pdb.RecommendationPerformance{}).
		Where("recommended_at >= ?", startTime).
		Order("recommended_at ASC").
		Find(&perfs).Error
	if err != nil {
		s.DatabaseError(c, "查询表现趋势", err)
		return
	}

	// 按日期聚合数据
	type DailyStats struct {
		Date         string  `json:"date"`
		Count        int     `json:"count"`
		AvgReturn24h float64 `json:"avg_return_24h"`
		AvgReturn7d  float64 `json:"avg_return_7d"`
		AvgReturn30d float64 `json:"avg_return_30d"`
		WinRate24h   float64 `json:"win_rate_24h"`
		WinRate7d    float64 `json:"win_rate_7d"`
		WinRate30d   float64 `json:"win_rate_30d"`
		MaxGain      float64 `json:"max_gain"`
		MaxDrawdown  float64 `json:"max_drawdown"`
		// 新增策略相关字段
		AvgStrategyReturn float64 `json:"avg_strategy_return"`
		StrategyWinRate   float64 `json:"strategy_win_rate"`
		AvgHoldingPeriod  float64 `json:"avg_holding_period"`
	}

	// 按日期分组
	dailyMap := make(map[string]*DailyStats)
	for _, perf := range perfs {
		date := perf.RecommendedAt.UTC().Format("2006-01-02")
		stats, ok := dailyMap[date]
		if !ok {
			stats = &DailyStats{Date: date}
			dailyMap[date] = stats
		}

		stats.Count++

		// 聚合24h数据
		if perf.Return24h != nil {
			stats.AvgReturn24h += *perf.Return24h
			if *perf.Return24h > 0 {
				stats.WinRate24h++
			}
		}

		// 聚合7d数据
		if perf.Return7d != nil {
			stats.AvgReturn7d += *perf.Return7d
			if *perf.Return7d > 0 {
				stats.WinRate7d++
			}
		}

		// 聚合30d数据
		if perf.Return30d != nil {
			stats.AvgReturn30d += *perf.Return30d
			if *perf.Return30d > 0 {
				stats.WinRate30d++
			}
		}

		// 最大涨幅和最大回撤
		if perf.MaxGain != nil && *perf.MaxGain > stats.MaxGain {
			stats.MaxGain = *perf.MaxGain
		}
		if perf.MaxDrawdown != nil && *perf.MaxDrawdown < stats.MaxDrawdown {
			stats.MaxDrawdown = *perf.MaxDrawdown
		}

		// 聚合策略数据
		if perf.ActualReturn != nil {
			stats.AvgStrategyReturn += *perf.ActualReturn
			if *perf.ActualReturn > 0 {
				stats.StrategyWinRate++
			}
		}
		if perf.HoldingPeriod != nil {
			stats.AvgHoldingPeriod += float64(*perf.HoldingPeriod)
		}
	}

	// 计算平均值和胜率
	trendData := make([]DailyStats, 0, len(dailyMap))
	for _, stats := range dailyMap {
		if stats.Count > 0 {
			stats.AvgReturn24h /= float64(stats.Count)
			stats.AvgReturn7d /= float64(stats.Count)
			stats.AvgReturn30d /= float64(stats.Count)
			stats.WinRate24h = (stats.WinRate24h / float64(stats.Count)) * 100
			stats.WinRate7d = (stats.WinRate7d / float64(stats.Count)) * 100
			stats.WinRate30d = (stats.WinRate30d / float64(stats.Count)) * 100

			// 计算策略相关平均值
			stats.AvgStrategyReturn /= float64(stats.Count)
			stats.StrategyWinRate = (stats.StrategyWinRate / float64(stats.Count)) * 100
			stats.AvgHoldingPeriod /= float64(stats.Count)
		}
		trendData = append(trendData, *stats)
	}

	// 按日期排序
	sort.Slice(trendData, func(i, j int) bool {
		return trendData[i].Date < trendData[j].Date
	})

	// 补齐缺失的日期（可选，如果需要连续的时间序列）
	// 这里暂时不补齐，前端可以处理

	c.JSON(http.StatusOK, gin.H{
		"days":     days,
		"interval": interval,
		"data":     trendData,
	})
}

// AdjustWeightsByPerformance 根据历史表现调整权重（反馈循环）
func (s *Server) AdjustWeightsByPerformance(ctx context.Context) (*DynamicWeights, error) {
	// 使用新的动态权重学习器
	weightLearner := NewDynamicWeightLearner()

	// 从历史数据学习
	err := weightLearner.LearnFromHistoricalData(ctx, s.db)
	if err != nil {
		log.Printf("[WARN] Dynamic weight learning failed: %v", err)
		// 降级到默认权重
		return &DynamicWeights{
			MarketWeight:    0.30,
			FlowWeight:      0.25,
			HeatWeight:      0.20,
			EventWeight:     0.15,
			SentimentWeight: 0.10,
		}, nil
	}

	// 计算自适应权重
	adaptiveWeights := weightLearner.CalculateAdaptiveWeights()

	// 如果没有足够的权重数据，返回默认权重
	if len(adaptiveWeights) == 0 {
		return &DynamicWeights{
			MarketWeight:    0.30,
			FlowWeight:      0.25,
			HeatWeight:      0.20,
			EventWeight:     0.15,
			SentimentWeight: 0.10,
		}, nil
	}

	return &DynamicWeights{
		MarketWeight:    adaptiveWeights["market"],
		FlowWeight:      adaptiveWeights["flow"],
		HeatWeight:      adaptiveWeights["heat"],
		EventWeight:     adaptiveWeights["event"],
		SentimentWeight: adaptiveWeights["sentiment"],
	}, nil
}

// getCachedPrice 获取带缓存的价格数据
func (s *Server) getCachedPrice(ctx context.Context, symbol, kind string, now time.Time) (float64, error) {
	// 如果还没有初始化价格缓存，创建一个
	if s.priceCache == nil {
		s.priceCache = NewPriceCache(30 * time.Second) // 30秒缓存
	}

	cacheKey := fmt.Sprintf("%s_%s", symbol, kind)

	// 尝试从缓存获取
	if price, found := s.priceCache.Get(cacheKey); found {
		return price, nil
	}

	// 缓存未命中，从API获取
	retryConfig := util.RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	var currentPrice float64
	err := util.Retry(ctx, func() error {
		klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1m", 1)
		if err != nil {
			return fmt.Errorf("从Binance获取 %s K线数据失败: %w", symbol, err)
		}
		if len(klines) == 0 {
			return fmt.Errorf("Binance返回的 %s K线数据为空", symbol)
		}
		price, err := strconv.ParseFloat(klines[0].Close, 64)
		if err != nil {
			return fmt.Errorf("解析 %s 价格失败: %w", symbol, err)
		}
		if price <= 0 {
			return fmt.Errorf("获取到的 %s 价格无效: %f", symbol, price)
		}
		currentPrice = price
		return nil
	}, &retryConfig)

	if err != nil {
		return 0, err
	}

	// 缓存结果
	s.priceCache.Set(cacheKey, currentPrice)
	return currentPrice, nil
}

// updateBacktestFromPerformanceWithPool 使用协程池并发更新回测数据
func (s *Server) updateBacktestFromPerformanceWithPool(ctx context.Context, workerPool *WorkerPool) error {
	// 获取待更新的回测记录
	perfs, err := pdb.GetPendingBacktests(s.db.DB(), 50)
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
		return s.updateBacktestFromPerformanceSerial(ctx, perfs, now)
	}

	// 使用协程池并发处理
	var wg sync.WaitGroup
	for _, perf := range perfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := s.updateOneBacktestPerformance(ctx, perf, now); err != nil {
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
func (s *Server) updateBacktestFromPerformanceSerial(ctx context.Context, perfs []pdb.RecommendationPerformance, now time.Time) error {
	updatedCount := 0
	for _, perf := range perfs {
		if err := s.updateOneBacktestPerformance(ctx, perf, now); err != nil {
			log.Printf("[BacktestUpdater] 更新记录失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			continue
		}
		updatedCount++
	}
	log.Printf("[BacktestUpdater] 成功更新 %d 条记录", updatedCount)
	return nil
}

// updateOneBacktestPerformance 更新单条回测记录
func (s *Server) updateOneBacktestPerformance(ctx context.Context, perf pdb.RecommendationPerformance, now time.Time) error {
	// 检查配置
	if s.cfg == nil || !s.cfg.Pricing.Enable {
		return fmt.Errorf("价格服务未启用")
	}

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
	// Binance API支持获取历史K线数据，完全免费且无需API密钥
	// 注意：Binance API返回的是从当前时间往前推的K线数据
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

		klines, err := s.fetchBinanceKlinesWithTimeRange(ctx, perf.Symbol, perf.Kind, interval, actualLimit, &startTime, &endTime)
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
		minDiff := float64(1 << 62)

		// 先查找目标时间之后的K线（最接近的）
		for i, kline := range klines {
			diff := kline.OpenTime - float64(targetUnix)
			if diff >= 0 && diff < minDiff {
				minDiff = diff
				bestIdx = i
			}
		}

		// 如果找不到目标时间之后的K线，查找目标时间之前的K线（允许12小时误差）
		if bestIdx < 0 {
			for i, kline := range klines {
				diff := float64(targetUnix) - kline.OpenTime
				if diff >= 0 && diff <= 12*3600*1000 {
					bestIdx = i
					break
				}
			}
		}

		if bestIdx >= 0 && bestIdx < len(klines) {
			price, err := strconv.ParseFloat(klines[bestIdx].Close, 64)
			if err == nil {
				log.Printf("[BacktestUpdater] 找到价格 (Symbol: %s, RecommendedAt: %s, TargetTime: %s, Price: %f, KlineTime: %s)",
					perf.Symbol,
					recommendedAt.Format("2006-01-02 15:04:05"),
					targetTime.Format("2006-01-02 15:04:05"),
					price,
					time.Unix(int64(klines[bestIdx].OpenTime/1000), 0).Format("2006-01-02 15:04:05"))
				return &price
			} else {
				log.Printf("[BacktestUpdater] 解析价格失败 (Symbol: %s): %v", perf.Symbol, err)
			}
		} else {
			firstTime := ""
			lastTime := ""
			if len(klines) > 0 {
				firstTime = time.Unix(int64(klines[0].OpenTime/1000), 0).Format("2006-01-02 15:04:05")
				lastTime = time.Unix(int64(klines[len(klines)-1].OpenTime/1000), 0).Format("2006-01-02 15:04:05")
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
		rating := s.calculatePerformanceRating(*perf.Return24h)
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
		return pdb.UpdateRecommendationPerformance(s.db.DB(), &perf)
	}, &saveRetryConfig)

	if err != nil {
		return fmt.Errorf("保存更新失败（已重试）: %w", err)
	}

	return nil
}

// BatchUpdateRecommendationPerformance 批量更新推荐表现记录
// POST /recommendations/performance/batch-update
func (s *Server) BatchUpdateRecommendationPerformance(c *gin.Context) {
	log.Printf("[BatchUpdateRecommendationPerformance] 开始批量更新推荐表现记录")

	var req struct {
		Ids []uint `json:"ids,omitempty"` // 可选：指定要更新的ID列表，为空则更新所有pending的记录
	}

	// 检查是否有请求体，如果有则解析JSON
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			s.JSONBindError(c, err)
			return
		}
	}

	// 获取需要更新的记录
	var perfs []pdb.RecommendationPerformance
	var err error

	if len(req.Ids) > 0 {
		// 更新指定的记录
		perfs, err = pdb.GetPerformanceByRecommendationIDs(s.db.DB(), req.Ids)
		if err != nil {
			s.DatabaseError(c, "查询指定表现记录", err)
			return
		}
	} else {
		// 更新所有pending状态的记录（限制数量避免一次性处理太多）
		perfs, err = pdb.GetPendingBacktests(s.db.DB(), 50) // 限制为50条
		if err != nil {
			s.DatabaseError(c, "查询待更新表现记录", err)
			return
		}
	}

	if len(perfs) == 0 {
		log.Printf("[BatchUpdateRecommendationPerformance] 没有找到需要更新的记录")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "没有找到需要更新的记录",
			"updated": 0,
		})
		return
	}

	log.Printf("[BatchUpdateRecommendationPerformance] 找到 %d 条待更新记录", len(perfs))

	// 使用工作池并发更新
	workerPool := NewWorkerPool(5) // 限制并发数为5
	var mu sync.Mutex
	var successCount int
	var errorCount int
	var errors []string

	for _, perf := range perfs {
		perfCopy := perf // 复制以避免闭包问题
		workerPool.Submit(func() {
			if err := s.updateOneRecommendationPerformance(c.Request.Context(), perfCopy, time.Now().UTC()); err != nil {
				mu.Lock()
				errorCount++
				errors = append(errors, fmt.Sprintf("ID %d: %v", perfCopy.ID, err))
				mu.Unlock()
				log.Printf("[BatchUpdateRecommendationPerformance] 更新失败 ID=%d: %v", perfCopy.ID, err)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
				log.Printf("[BatchUpdateRecommendationPerformance] 更新成功 ID=%d", perfCopy.ID)
			}
		})
	}

	// 等待所有更新完成
	workerPool.Wait()

	log.Printf("[BatchUpdateRecommendationPerformance] 批量更新完成: 成功 %d, 失败 %d", successCount, errorCount)

	response := gin.H{
		"success":       true,
		"message":       fmt.Sprintf("批量更新完成: 成功 %d, 失败 %d", successCount, errorCount),
		"total":         len(perfs),
		"updated":       successCount,
		"errors":        errorCount,
		"error_details": errors,
	}

	c.JSON(http.StatusOK, response)
}

// BatchStrategyTest 执行批量策略测试
// POST /recommendations/performance/batch-strategy-test
func (s *Server) BatchStrategyTest(c *gin.Context) {
	log.Printf("[BatchStrategyTest] 开始批量策略测试")

	var req struct {
		Ids []uint `json:"ids,omitempty"` // 可选：指定要测试的ID列表，为空则测试所有completed但未测试的记录
	}

	// 检查是否有请求体，如果有则解析JSON
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			s.JSONBindError(c, err)
			return
		}
	}

	// 获取需要测试的记录
	var perfs []pdb.RecommendationPerformance
	var err error

	if len(req.Ids) > 0 {
		// 测试指定的记录
		perfs, err = pdb.GetPerformanceByRecommendationIDs(s.db.DB(), req.Ids)
		if err != nil {
			s.DatabaseError(c, "查询指定表现记录", err)
			return
		}
	} else {
		// 获取所有需要策略回测的记录
		perfs, err = pdb.GetPerformancesNeedingStrategyBacktest(s.db.DB(), 30) // 限制为30条
		if err != nil {
			s.DatabaseError(c, "查询待测试表现记录", err)
			return
		}
	}

	if len(perfs) == 0 {
		log.Printf("[BatchStrategyTest] 没有找到需要测试的记录")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "没有找到需要测试的记录",
			"tested":  0,
		})
		return
	}

	log.Printf("[BatchStrategyTest] 找到 %d 条待测试记录", len(perfs))

	// 使用工作池并发测试
	workerPool := NewWorkerPool(3) // 限制并发数为3，避免API限流
	var mu sync.Mutex
	var successCount int
	var errorCount int
	var errors []string

	for _, perf := range perfs {
		perfCopy := perf // 复制以避免闭包问题
		workerPool.Submit(func() {
			// 初始化策略回测引擎并执行测试
			strategyEngine := NewStrategyBacktestEngine(s.db, s.dataManager)
			if _, err := strategyEngine.ExecuteStrategyBacktest(&perfCopy); err != nil {
				mu.Lock()
				errorCount++
				errors = append(errors, fmt.Sprintf("ID %d: %v", perfCopy.ID, err))
				mu.Unlock()
				log.Printf("[BatchStrategyTest] 策略测试失败 ID=%d: %v", perfCopy.ID, err)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
				log.Printf("[BatchStrategyTest] 策略测试成功 ID=%d", perfCopy.ID)
			}
		})
	}

	// 等待所有测试完成
	workerPool.Wait()

	log.Printf("[BatchStrategyTest] 批量策略测试完成: 成功 %d, 失败 %d", successCount, errorCount)

	response := gin.H{
		"success":       true,
		"message":       fmt.Sprintf("批量策略测试完成: 成功 %d, 失败 %d", successCount, errorCount),
		"total":         len(perfs),
		"tested":        successCount,
		"errors":        errorCount,
		"error_details": errors,
	}

	c.JSON(http.StatusOK, response)
}
