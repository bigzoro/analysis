package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"analysis/internal/netutil"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// GetBacktestRecords 获取回测记录
// GET /recommendations/backtest?page=1&limit=20&status=completed&symbol=BTC&start_date=2024-01-01&end_date=2024-12-31&sort_by=recommended_at&sort_order=desc
// 优先使用 RecommendationPerformance，兼容 BacktestRecord
// 支持策略回测结果显示
func (s *Server) GetBacktestRecords(c *gin.Context) {
	// 分页参数
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if n, err := strconv.Atoi(pageStr); err == nil && n > 0 {
			page = n
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	// 筛选参数
	status := strings.TrimSpace(c.Query("status"))
	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	// 排序参数
	sortBy := strings.TrimSpace(c.Query("sort_by"))
	if sortBy == "" {
		sortBy = "recommended_at"
	}
	sortOrder := strings.TrimSpace(c.Query("sort_order"))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// 优先从 RecommendationPerformance 查询（统一数据源）
	var perfs []pdb.RecommendationPerformance
	var total int64

	q := s.db.DB().Model(&pdb.RecommendationPerformance{})

	// 状态筛选（映射到 backtest_status）
	if status != "" {
		q = q.Where("backtest_status = ?", status)
	} else {
		// 默认只显示有回测数据的记录
		q = q.Where("backtest_status != ''")
	}

	// 币种筛选
	if symbol != "" {
		q = q.Where("base_symbol = ? OR symbol = ?", symbol, symbol)
	}

	// 日期筛选
	if startDate != "" {
		q = q.Where("recommended_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		q = q.Where("recommended_at <= ?", endDate+" 23:59:59")
	}

	// 获取总数
	if err := q.Count(&total).Error; err != nil {
		s.DatabaseError(c, "查询回测记录总数", err)
		return
	}

	// 排序
	orderClause := sortBy + " " + sortOrder
	q = q.Order(orderClause)

	// 分页
	offset := (page - 1) * limit
	err := q.Offset(offset).Limit(limit).Find(&perfs).Error
	if err != nil {
		s.DatabaseError(c, "查询回测记录", err)
		return
	}

	// 转换为扩展格式（包含推荐得分和表现指标）
	records := make([]gin.H, 0, len(perfs))
	for _, perf := range perfs {
		rec := gin.H{
			"id":                perf.ID,
			"recommendation_id": perf.RecommendationID,
			"symbol":            perf.Symbol,
			"base_symbol":       perf.BaseSymbol,
			"recommended_at":    perf.RecommendedAt,
			"recommended_price": fmt.Sprintf("%.8f", perf.RecommendedPrice),
			"status":            perf.BacktestStatus,
			"total_score":       perf.TotalScore, // 推荐得分
		}

		// 转换价格字段
		if perf.Price24h != nil {
			rec["price_after_24h"] = fmt.Sprintf("%.8f", *perf.Price24h)
		}
		if perf.Price7d != nil {
			rec["price_after_7d"] = fmt.Sprintf("%.8f", *perf.Price7d)
		}
		if perf.Price30d != nil {
			rec["price_after_30d"] = fmt.Sprintf("%.8f", *perf.Price30d)
		}

		// 转换收益率字段
		if perf.Return24h != nil {
			rec["performance_24h"] = *perf.Return24h
		}
		if perf.Return7d != nil {
			rec["performance_7d"] = *perf.Return7d
		}
		if perf.Return30d != nil {
			rec["performance_30d"] = *perf.Return30d
		}

		// 添加表现指标
		if perf.MaxGain != nil {
			rec["max_gain"] = *perf.MaxGain
		}
		if perf.MaxDrawdown != nil {
			rec["max_drawdown"] = *perf.MaxDrawdown
		}

		// 添加策略执行信息
		if perf.EntryPrice != nil {
			rec["entry_price"] = fmt.Sprintf("%.8f", *perf.EntryPrice)
		}
		if perf.EntryTime != nil {
			rec["entry_time"] = perf.EntryTime
		}
		if perf.ExitPrice != nil {
			rec["exit_price"] = fmt.Sprintf("%.8f", *perf.ExitPrice)
		}
		if perf.ExitTime != nil {
			rec["exit_time"] = perf.ExitTime
		}
		if perf.ExitReason != "" {
			rec["exit_reason"] = perf.ExitReason
		}

		// 添加策略绩效指标
		if perf.ActualReturn != nil {
			rec["actual_return"] = *perf.ActualReturn
		}
		if perf.HoldingPeriod != nil {
			rec["holding_period"] = *perf.HoldingPeriod
		}
		if perf.MaxFavorableExcursion != nil {
			rec["max_favorable_excursion"] = *perf.MaxFavorableExcursion
		}
		if perf.MaxAdverseExcursion != nil {
			rec["max_adverse_excursion"] = *perf.MaxAdverseExcursion
		}

		// 添加策略配置信息
		if perf.StrategyConfig != nil {
			rec["strategy_config"] = perf.StrategyConfig
		}
		if perf.EntryConditions != nil {
			rec["entry_conditions"] = perf.EntryConditions
		}
		if perf.ExitConditions != nil {
			rec["exit_conditions"] = perf.ExitConditions
		}

		records = append(records, rec)
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   total,
		"page":    page,
		"limit":   limit,
		"pages":   (total + int64(limit) - 1) / int64(limit), // 总页数
	})
}

// GetBacktestStats 获取回测统计
// GET /recommendations/backtest/stats
// 使用 RecommendationPerformance 数据计算统计
func (s *Server) GetBacktestStats(c *gin.Context) {
	// 使用统一的 RecommendationPerformance 统计
	stats, err := pdb.GetPerformanceStats(s.db.DB(), 30) // 最近30天
	if err != nil {
		s.DatabaseError(c, "查询回测统计", err)
		return
	}

	// 转换为回测统计格式（兼容前端）
	backtestStats := map[string]interface{}{
		"total":               stats["total"],
		"completed":           stats["completed_24h"], // 使用24h完成数
		"avg_performance_24h": stats["avg_return_24h"],
		"avg_performance_7d":  stats["avg_return_7d"],
		"avg_performance_30d": stats["avg_return_30d"],
		"win_rate_24h":        stats["win_rate_24h"],
		"win_rate_7d":         stats["win_rate_7d"],
		"win_rate_30d":        stats["win_rate_30d"],
		"max_gain":            stats["max_gain"],
		"max_drawdown":        stats["max_drawdown"],
	}

	c.JSON(http.StatusOK, backtestStats)
}

// CreateBacktestFromRecommendation 从推荐创建回测记录
// POST /recommendations/backtest
func (s *Server) CreateBacktestFromRecommendation(c *gin.Context) {
	var req struct {
		RecommendationID uint   `json:"recommendation_id"`
		Symbol           string `json:"symbol"`
		BaseSymbol       string `json:"base_symbol"`
		Price            string `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Symbol == "" || req.Price == "" {
		s.ValidationError(c, "", "symbol和price不能为空")
		return
	}

	rec := &pdb.BacktestRecord{
		RecommendationID: req.RecommendationID,
		Symbol:           strings.ToUpper(req.Symbol),
		BaseSymbol:       strings.ToUpper(req.BaseSymbol),
		RecommendedAt:    time.Now().UTC(),
		RecommendedPrice: req.Price,
		Status:           "pending",
	}

	if err := pdb.CreateBacktestRecord(s.db.DB(), rec); err != nil {
		s.DatabaseError(c, "创建回测记录", err)
		return
	}

	// 异步更新回测结果（定期任务会处理）
	go s.updateBacktestResult(context.Background(), rec.ID)

	c.JSON(http.StatusOK, gin.H{"id": rec.ID})
}

// updateBacktestResult 更新回测结果（异步）
func (s *Server) updateBacktestResult(ctx context.Context, recordID uint) {
	// 获取回测记录
	var rec pdb.BacktestRecord
	if err := s.db.DB().First(&rec, recordID).Error; err != nil {
		log.Printf("[backtest] 获取回测记录失败 (ID=%d): %v", recordID, err)
		return
	}

	// 检查配置
	if s.cfg == nil || !s.cfg.Pricing.Enable {
		log.Printf("[backtest] 价格服务未启用，跳过回测更新 (ID=%d)", recordID)
		return
	}

	// 获取币种ID
	coinID := s.cfg.Pricing.Map[strings.ToUpper(rec.BaseSymbol)]
	if coinID == "" {
		log.Printf("[backtest] 不支持的币种: %s (ID=%d)", rec.BaseSymbol, recordID)
		rec.Status = "failed"
		pdb.UpdateBacktestRecord(s.db.DB(), &rec)
		return
	}

	// 计算目标时间点
	now := time.Now().UTC()
	recommendedAt := rec.RecommendedAt.UTC()
	time24h := recommendedAt.Add(24 * time.Hour)
	time7d := recommendedAt.Add(7 * 24 * time.Hour)
	time30d := recommendedAt.Add(30 * 24 * time.Hour)

	// 获取历史价格数据（最多获取30天的数据）
	days := 30
	if now.Sub(recommendedAt) < 30*24*time.Hour {
		days = int(now.Sub(recommendedAt).Hours()/24) + 1
	}
	if days < 1 {
		days = 1
	}

	// 构建CoinGecko API URL
	endpoint := s.cfg.Pricing.CoinGeckoEndpoint
	baseURL := endpoint
	if strings.Contains(endpoint, "/api/v3") {
		parts := strings.Split(endpoint, "/api/v3")
		if len(parts) > 0 {
			baseURL = strings.TrimSuffix(parts[0], "/")
		}
	}

	var url string
	if days <= 1 {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d&interval=hourly", baseURL, coinID, days)
	} else {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d", baseURL, coinID, days)
	}

	// 创建带超时的context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 获取价格数据（带重试）
	var resp struct {
		Prices [][]float64 `json:"prices"`
	}
	var lastErr error
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("[backtest] 重试获取价格数据 (ID=%d, 尝试 %d/%d): %v", recordID, attempt+1, maxRetries, lastErr)
			select {
			case <-ctx.Done():
				log.Printf("[backtest] 获取价格数据超时 (ID=%d)", recordID)
				rec.Status = "failed"
				pdb.UpdateBacktestRecord(s.db.DB(), &rec)
				return
			case <-time.After(delay):
			}
		}

		err := netutil.GetJSON(ctx, url, &resp)
		if err == nil {
			break
		}
		lastErr = err

		errStr := strings.ToLower(err.Error())
		isTimeout := strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "tls handshake timeout")
		isNetworkErr := strings.Contains(errStr, "connection") || strings.Contains(errStr, "eof") || isTimeout

		if !isNetworkErr || attempt == maxRetries-1 {
			log.Printf("[backtest] 获取价格数据失败 (ID=%d): %v", recordID, err)
			rec.Status = "failed"
			pdb.UpdateBacktestRecord(s.db.DB(), &rec)
			return
		}
	}

	if lastErr != nil {
		log.Printf("[backtest] 获取价格数据最终失败 (ID=%d): %v", recordID, lastErr)
		rec.Status = "failed"
		pdb.UpdateBacktestRecord(s.db.DB(), &rec)
		return
	}

	// 解析推荐价格
	recommendedPrice, err := strconv.ParseFloat(rec.RecommendedPrice, 64)
	if err != nil {
		log.Printf("[backtest] 解析推荐价格失败 (ID=%d): %v", recordID, err)
		rec.Status = "failed"
		pdb.UpdateBacktestRecord(s.db.DB(), &rec)
		return
	}

	// 查找最接近目标时间点的价格
	findPriceAtTime := func(targetTime time.Time, prices [][]float64) *float64 {
		if len(prices) == 0 {
			return nil
		}
		targetUnix := float64(targetTime.Unix()) * 1000 // CoinGecko使用毫秒时间戳

		// 如果目标时间在未来，返回nil
		if targetUnix > prices[len(prices)-1][0] {
			return nil
		}

		// 二分查找最接近的时间点
		bestIdx := -1
		minDiff := float64(1 << 62)
		for i, p := range prices {
			diff := p[0] - targetUnix
			if diff >= 0 && diff < minDiff {
				minDiff = diff
				bestIdx = i
			}
		}

		if bestIdx >= 0 && minDiff < 24*3600*1000 { // 允许1天内的误差
			price := prices[bestIdx][1]
			return &price
		}

		// 如果找不到精确匹配，使用最接近的时间点
		for i, p := range prices {
			diff := p[0] - targetUnix
			if diff >= -12*3600*1000 && diff <= 12*3600*1000 { // 允许12小时误差
				price := p[1]
				return &price
			}
			if diff > 0 {
				// 已经过了目标时间，使用前一个点
				if i > 0 {
					price := prices[i-1][1]
					return &price
				}
				break
			}
		}

		return nil
	}

	// 计算各时间点的价格和收益率
	updatePriceAndPerformance := func(targetTime time.Time, priceField **string, perfField **float64) {
		if now.Before(targetTime) {
			// 时间还未到，不更新
			return
		}

		price := findPriceAtTime(targetTime, resp.Prices)
		if price != nil {
			priceStr := fmt.Sprintf("%.8f", *price)
			*priceField = &priceStr

			// 计算收益率
			performance := ((*price - recommendedPrice) / recommendedPrice) * 100
			*perfField = &performance
		}
	}

	// 更新24h、7d、30d的价格和收益率
	updatePriceAndPerformance(time24h, &rec.PriceAfter24h, &rec.Performance24h)
	updatePriceAndPerformance(time7d, &rec.PriceAfter7d, &rec.Performance7d)
	updatePriceAndPerformance(time30d, &rec.PriceAfter30d, &rec.Performance30d)

	// 更新状态
	rec.Status = "completed"
	if err := pdb.UpdateBacktestRecord(s.db.DB(), &rec); err != nil {
		log.Printf("[backtest] 更新回测记录失败 (ID=%d): %v", recordID, err)
		return
	}

	log.Printf("[backtest] 成功更新回测记录 (ID=%d, Symbol=%s)", recordID, rec.BaseSymbol)
}

// UpdateBacktestRecord 手动触发更新回测记录
// POST /recommendations/backtest/:id/update
func (s *Server) UpdateBacktestRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的ID")
		return
	}

	// 检查记录是否存在
	var rec pdb.BacktestRecord
	if err := s.db.DB().First(&rec, id).Error; err != nil {
		s.NotFound(c, "回测记录不存在")
		return
	}

	// 异步更新回测结果
	go s.updateBacktestResult(context.Background(), uint(id))

	c.JSON(http.StatusOK, gin.H{
		"updated": 1,
		"message": "回测更新已启动，请稍后刷新查看结果",
	})
}

// BatchUpdateBacktestRecords 批量更新回测记录
// POST /recommendations/backtest/batch-update
func (s *Server) BatchUpdateBacktestRecords(c *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"` // 可选：指定要更新的ID列表，为空则更新所有pending的记录
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	var perfs []pdb.RecommendationPerformance
	var err error

	if len(req.Ids) > 0 {
		// 更新指定的记录（按RecommendationPerformance ID）
		perfs, err = pdb.GetPerformanceByRecommendationIDs(s.db.DB(), req.Ids)
		if err != nil {
			s.DatabaseError(c, "查询指定回测记录", err)
			return
		}
	} else {
		// 更新所有pending状态的记录
		perfs, err = pdb.GetPendingBacktests(s.db.DB(), 100) // 限制为100条，避免一次性处理太多
		if err != nil {
			s.DatabaseError(c, "查询待更新回测记录", err)
			return
		}
	}

	if len(perfs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"updated": 0,
			"message": "没有需要更新的回测记录",
		})
		return
	}

	// 使用协程池并发更新（复用现有的逻辑）
	workerPool := NewWorkerPool(10) // 限制并发数为10
	var wg sync.WaitGroup
	updatedCount := 0
	failedCount := 0
	var mu sync.Mutex

	for _, perf := range perfs {
		wg.Add(1)
		perf := perf // 避免闭包问题
		workerPool.Submit(func() {
			defer wg.Done()
			if err := s.updateOneBacktestPerformance(context.Background(), perf, time.Now().UTC()); err != nil {
				mu.Lock()
				failedCount++
				mu.Unlock()
				log.Printf("[BatchUpdate] 更新记录失败 (ID: %d, Symbol: %s): %v", perf.ID, perf.Symbol, err)
			} else {
				mu.Lock()
				updatedCount++
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"updated": updatedCount,
		"failed":  failedCount,
		"message": fmt.Sprintf("批量更新完成: 成功 %d 条, 失败 %d 条", updatedCount, failedCount),
	})
}

// ExecuteStrategyBacktest 执行策略回测
// POST /recommendations/backtest/strategy
func (s *Server) ExecuteStrategyBacktest(c *gin.Context) {
	var req struct {
		PerformanceID uint `json:"performance_id"` // RecommendationPerformance ID
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	// 获取性能记录
	perf, err := pdb.GetRecommendationPerformanceByID(s.db.DB(), req.PerformanceID)
	if err != nil {
		s.DatabaseError(c, "查询性能记录", err)
		return
	}
	if perf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "性能记录不存在"})
		return
	}

	// 初始化策略回测引擎
	strategyEngine := NewStrategyBacktestEngine(s.db, s.dataManager)

	// 执行策略回测
	result, err := strategyEngine.ExecuteStrategyBacktest(perf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("策略回测执行失败: %v", err)})
		return
	}

	// 保存结果
	strategyConfig, _ := strategyEngine.parseStrategyConfig(perf)
	err = strategyEngine.SaveStrategyExecutionResult(perf.ID, result, strategyConfig)
	if err != nil {
		s.DatabaseError(c, "保存策略回测结果", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "策略回测完成",
		"result": gin.H{
			"entry_price":             result.EntryPrice,
			"entry_time":              result.EntryTime,
			"exit_price":              result.ExitPrice,
			"exit_time":               result.ExitTime,
			"exit_reason":             result.ExitReason,
			"return":                  result.Return,
			"holding_period":          result.HoldingPeriodMinutes,
			"max_favorable_excursion": result.MaxFavorableExcursion,
			"max_adverse_excursion":   result.MaxAdverseExcursion,
		},
	})
}

// TestStrategyBacktest 测试单个记录的策略回测
// POST /recommendations/backtest/strategy/test
func (s *Server) TestStrategyBacktest(c *gin.Context) {
	var req struct {
		PerformanceID uint `json:"performance_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 获取性能记录
	perf, err := pdb.GetRecommendationPerformanceByID(s.db.DB(), req.PerformanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询性能记录失败: %v", err)})
		return
	}
	if perf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "性能记录不存在"})
		return
	}

	// 初始化策略回测引擎
	strategyEngine := NewStrategyBacktestEngine(s.db, s.dataManager)

	// 执行策略回测
	result, err := strategyEngine.ExecuteStrategyBacktest(perf)
	if err != nil {
		fmt.Printf("策略回测执行失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("策略回测执行失败: %v", err)})
		return
	}

	// 保存结果
	strategyConfig, _ := strategyEngine.parseStrategyConfig(perf)
	err = strategyEngine.SaveStrategyExecutionResult(perf.ID, result, strategyConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存结果失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "策略回测测试完成",
		"result": gin.H{
			"performance_id":          perf.ID,
			"entry_price":             result.EntryPrice,
			"entry_time":              result.EntryTime,
			"exit_price":              result.ExitPrice,
			"exit_time":               result.ExitTime,
			"exit_reason":             result.ExitReason,
			"return":                  result.Return,
			"holding_period":          result.HoldingPeriodMinutes,
			"max_favorable_excursion": result.MaxFavorableExcursion,
			"max_adverse_excursion":   result.MaxAdverseExcursion,
		},
	})
}

// BatchExecuteStrategyBacktest 批量执行策略回测
// POST /recommendations/backtest/strategy/batch
func (s *Server) BatchExecuteStrategyBacktest(c *gin.Context) {
	var req struct {
		PerformanceIDs []uint `json:"performance_ids"` // RecommendationPerformance IDs
		Limit          int    `json:"limit"`           // 限制处理数量，默认50
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	// 获取性能记录
	var perfs []pdb.RecommendationPerformance
	var err error

	if len(req.PerformanceIDs) > 0 {
		perfs, err = pdb.GetPerformanceByRecommendationIDs(s.db.DB(), req.PerformanceIDs)
	} else {
		// 获取需要策略回测的记录（没有actual_return的记录）
		perfs, err = pdb.GetPerformancesNeedingStrategyBacktest(s.db.DB(), req.Limit)
	}

	if err != nil {
		s.DatabaseError(c, "查询待策略回测记录", err)
		return
	}

	if len(perfs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":   "没有需要策略回测的记录",
			"processed": 0,
		})
		return
	}

	// 初始化策略回测引擎
	strategyEngine := NewStrategyBacktestEngine(s.db, s.dataManager)

	processedCount := 0
	successCount := 0
	failedCount := 0

	// 逐个执行策略回测
	for _, perf := range perfs {
		// 记录开始处理
		fmt.Printf("开始处理记录 ID: %d, 推荐时间: %s\n", perf.ID, perf.RecommendedAt)

		result, err := strategyEngine.ExecuteStrategyBacktest(&perf)
		if err != nil {
			fmt.Printf("策略回测失败 ID: %d, 错误: %v\n", perf.ID, err)
			failedCount++
			continue
		}

		strategyConfig, _ := strategyEngine.parseStrategyConfig(&perf)
		err = strategyEngine.SaveStrategyExecutionResult(perf.ID, result, strategyConfig)
		if err != nil {
			fmt.Printf("保存结果失败 ID: %d, 错误: %v\n", perf.ID, err)
			failedCount++
			continue
		}

		fmt.Printf("策略回测成功 ID: %d, 收益: %.2f%%, 退出原因: %s\n",
			perf.ID, result.Return, result.ExitReason)
		successCount++
		processedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("批量策略回测完成: 处理 %d 条, 成功 %d 条, 失败 %d 条",
			processedCount, successCount, failedCount),
		"processed": processedCount,
		"success":   successCount,
		"failed":    failedCount,
	})
}

// =================== 高级回测分析API ===================

// RunWalkForwardAnalysisAPI 执行走步前进分析API
func (s *Server) RunWalkForwardAnalysisAPI(c *gin.Context) {
	var req struct {
		Symbol            string    `json:"symbol" binding:"required"`
		StartDate         time.Time `json:"start_date" binding:"required"`
		EndDate           time.Time `json:"end_date" binding:"required"`
		Strategy          string    `json:"strategy"`
		InSamplePeriod    int       `json:"in_sample_period"`
		OutOfSamplePeriod int       `json:"out_of_sample_period"`
		StepSize          int       `json:"step_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// 设置默认值
	if req.Strategy == "" {
		req.Strategy = "ml_prediction"
	}
	if req.InSamplePeriod == 0 {
		req.InSamplePeriod = 12 // 12个月
	}
	if req.OutOfSamplePeriod == 0 {
		req.OutOfSamplePeriod = 3 // 3个月
	}
	if req.StepSize == 0 {
		req.StepSize = 3 // 3个月步长
	}

	ctx := c.Request.Context()

	// 创建基础回测配置
	baseConfig := BacktestConfig{
		Symbol:      req.Symbol,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		InitialCash: 10000,
		Strategy:    req.Strategy,
		MaxPosition: 1.0,
		StopLoss:    0.05,
		TakeProfit:  0.10,
		Commission:  0.001,
	}

	// 创建走步前进分析配置
	analysis := WalkForwardAnalysis{
		InSamplePeriod:    req.InSamplePeriod,
		OutOfSamplePeriod: req.OutOfSamplePeriod,
		StepSize:          req.StepSize,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
	}

	// 执行走步前进分析
	result, err := s.backtestEngine.RunWalkForwardAnalysis(ctx, baseConfig, analysis)
	if err != nil {
		c.JSON(500, gin.H{"error": "走步前进分析失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"walk_forward_analysis": result,
		"analysis_timestamp":    time.Now().Unix(),
	})
}

// RunMonteCarloAnalysisAPI 执行蒙特卡洛分析API
func (s *Server) RunMonteCarloAnalysisAPI(c *gin.Context) {
	var req struct {
		Symbol        string    `json:"symbol" binding:"required"`
		StartDate     time.Time `json:"start_date" binding:"required"`
		EndDate       time.Time `json:"end_date" binding:"required"`
		Strategy      string    `json:"strategy"`
		Simulations   int       `json:"simulations"`
		BootstrapSize int       `json:"bootstrap_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// 设置默认值
	if req.Strategy == "" {
		req.Strategy = "ml_prediction"
	}
	if req.Simulations == 0 {
		req.Simulations = 1000
	}
	if req.BootstrapSize == 0 {
		req.BootstrapSize = 252 // 一年的交易日
	}

	ctx := c.Request.Context()

	// 创建基础回测配置
	baseConfig := BacktestConfig{
		Symbol:      req.Symbol,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		InitialCash: 10000,
		Strategy:    req.Strategy,
		MaxPosition: 1.0,
		StopLoss:    0.05,
		TakeProfit:  0.10,
		Commission:  0.001,
	}

	// 创建蒙特卡洛分析配置
	analysis := MonteCarloAnalysis{
		Simulations:     req.Simulations,
		ConfidenceLevel: 0.95,
		BootstrapSize:   req.BootstrapSize,
	}

	// 执行蒙特卡洛分析
	result, err := s.backtestEngine.RunMonteCarloAnalysis(ctx, baseConfig, analysis)
	if err != nil {
		c.JSON(500, gin.H{"error": "蒙特卡洛分析失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"monte_carlo_analysis": result,
		"analysis_timestamp":   time.Now().Unix(),
	})
}

// RunStrategyOptimizationAPI 执行策略优化API
func (s *Server) RunStrategyOptimizationAPI(c *gin.Context) {
	var req struct {
		Symbol        string                  `json:"symbol" binding:"required"`
		StartDate     time.Time               `json:"start_date" binding:"required"`
		EndDate       time.Time               `json:"end_date" binding:"required"`
		Strategy      string                  `json:"strategy"`
		Parameters    []OptimizationParameter `json:"parameters"`
		Method        string                  `json:"method"`
		MaxIterations int                     `json:"max_iterations"`
		Objective     string                  `json:"objective"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// 设置默认值
	if req.Strategy == "" {
		req.Strategy = "ml_prediction"
	}
	if req.Method == "" {
		req.Method = "grid"
	}
	if req.MaxIterations == 0 {
		req.MaxIterations = 100
	}
	if req.Objective == "" {
		req.Objective = "sharpe"
	}

	// 设置默认优化参数
	if len(req.Parameters) == 0 {
		req.Parameters = []OptimizationParameter{
			{
				Name:         "stop_loss",
				Type:         "float",
				MinValue:     0.01,
				MaxValue:     0.10,
				StepSize:     0.01,
				DefaultValue: 0.05,
			},
			{
				Name:         "take_profit",
				Type:         "float",
				MinValue:     0.05,
				MaxValue:     0.20,
				StepSize:     0.02,
				DefaultValue: 0.10,
			},
		}
	}

	ctx := c.Request.Context()

	// 创建基础回测配置
	baseConfig := BacktestConfig{
		Symbol:      req.Symbol,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		InitialCash: 10000,
		Strategy:    req.Strategy,
		MaxPosition: 1.0,
		Commission:  0.001,
	}

	// 创建策略优化配置
	optimization := StrategyOptimization{
		Parameters:     req.Parameters,
		Objective:      req.Objective,
		Method:         req.Method,
		MaxIterations:  req.MaxIterations,
		PopulationSize: 50,
	}

	// 执行策略优化
	result, err := s.backtestEngine.RunStrategyOptimization(ctx, baseConfig, optimization)
	if err != nil {
		c.JSON(500, gin.H{"error": "策略优化失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"strategy_optimization":  result,
		"optimization_timestamp": time.Now().Unix(),
	})
}

// RunAttributionAnalysisAPI 执行归因分析API
func (s *Server) RunAttributionAnalysisAPI(c *gin.Context) {
	var req struct {
		Symbol          string    `json:"symbol" binding:"required"`
		BenchmarkSymbol string    `json:"benchmark_symbol" binding:"required"`
		StartDate       time.Time `json:"start_date" binding:"required"`
		EndDate         time.Time `json:"end_date" binding:"required"`
		Strategy        string    `json:"strategy"`
		TimeHorizon     string    `json:"time_horizon"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// 设置默认值
	if req.Strategy == "" {
		req.Strategy = "ml_prediction"
	}
	if req.TimeHorizon == "" {
		req.TimeHorizon = "monthly"
	}

	ctx := c.Request.Context()

	// 创建基础回测配置
	baseConfig := BacktestConfig{
		Symbol:      req.Symbol,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		InitialCash: 10000,
		Strategy:    req.Strategy,
		MaxPosition: 1.0,
		StopLoss:    0.05,
		TakeProfit:  0.10,
		Commission:  0.001,
	}

	// 创建归因分析配置
	analysis := AttributionAnalysis{
		TimeHorizon:     req.TimeHorizon,
		BenchmarkSymbol: req.BenchmarkSymbol,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
	}

	// 执行归因分析
	result, err := s.backtestEngine.RunAttributionAnalysis(ctx, baseConfig, analysis)
	if err != nil {
		c.JSON(500, gin.H{"error": "归因分析失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"attribution_analysis": result,
		"analysis_timestamp":   time.Now().Unix(),
	})
}
