package server

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
	"analysis/internal/netutil"
	"analysis/internal/server/strategy/shared/execution"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OrderScheduler struct {
	db         *gorm.DB
	cfg        *config.Config
	ctx        context.Context
	server     *Server     // 引用Server实例，用于智能候选选择器
	workerPool *WorkerPool // 优化：使用协程池限制并发

	// 策略执行锁，防止同一个策略并发执行
	strategyLocks     map[uint]*sync.Mutex
	strategyLockMutex sync.RWMutex
}

// EligibleSymbolResult 符合条件的交易对结果
type EligibleSymbolResult struct {
	Symbol string
	Result StrategyDecisionResult
}

func NewOrderScheduler(db *gorm.DB, cfg *config.Config, server *Server) *OrderScheduler {
	// 优化：限制最大并发数为 10，避免创建过多 goroutine
	return &OrderScheduler{
		db:         db,
		server:     server,
		cfg:        cfg,
		ctx:        context.Background(),
		workerPool: NewWorkerPool(10),
	}
}

func (s *OrderScheduler) Start() {
	log.Printf("[OrderScheduler] Starting order scheduler...")
	go s.loop()
	log.Printf("[OrderScheduler] Order processing loop started")

	go s.strategyExecutionLoop()
	log.Printf("[OrderScheduler] Strategy execution scheduler started")
}

func (s *OrderScheduler) loop() {
	tk := time.NewTicker(1 * time.Second)
	defer tk.Stop()

	for range tk.C {
		s.tick()
	}
}

func (s *OrderScheduler) strategyExecutionLoop() {
	log.Printf("[StrategyScheduler] Strategy execution loop starting...")
	// 每1分钟检查一次策略执行，提供更及时的响应
	tk := time.NewTicker(1 * time.Minute)
	defer tk.Stop()

	for range tk.C {
		log.Printf("[StrategyScheduler] Checking for strategies to execute...")
		s.checkAndExecuteStrategies()
	}
}

func (s *OrderScheduler) checkAndExecuteStrategies() {
	// 首先检查是否有超时的策略执行
	s.checkAndHandleTimeoutExecutions()

	// 获取所有正在运行的策略
	runningStrategies, err := pdb.GetRunningStrategies(s.db)
	if err != nil {
		log.Printf("[StrategyScheduler] Failed to get running strategies: %v", err)
		return
	}

	log.Printf("[StrategyScheduler] Found %d running strategies", len(runningStrategies))

	for _, strategy := range runningStrategies {
		log.Printf("[StrategyScheduler] Checking strategy %d (%s), is_running: %v, last_run_at: %v, run_interval: %d",
			strategy.ID, strategy.Name, strategy.IsRunning, strategy.LastRunAt, strategy.RunInterval)

		// 检查策略状态一致性
		if err := s.checkStrategyConsistency(strategy); err != nil {
			log.Printf("[StrategyScheduler] Strategy consistency check failed for %d: %v", strategy.ID, err)
			continue
		}

		// 检查是否到了执行时间
		if !s.shouldExecuteStrategy(strategy) {
			if strategy.LastRunAt != nil {
				nextRun := strategy.LastRunAt.Add(time.Duration(strategy.RunInterval) * time.Minute)
				log.Printf("[StrategyScheduler] Strategy %d not ready for execution, next run at %v (current time: %v)",
					strategy.ID, nextRun, time.Now())
			} else {
				log.Printf("[StrategyScheduler] Strategy %d waiting for first execution", strategy.ID)
			}
			continue
		}

		log.Printf("[StrategyScheduler] Strategy %d is due for execution, creating execution record", strategy.ID)

		// 自动创建策略执行记录
		if err := s.createStrategyExecutionRecord(strategy); err != nil {
			log.Printf("[StrategyScheduler] Failed to create execution record for strategy %d: %v", strategy.ID, err)
			continue
		}

		log.Printf("[StrategyScheduler] Executing strategy %d", strategy.ID)

		// 获取策略锁，防止并发执行
		lock := s.getStrategyLock(strategy.ID)
		lock.Lock() // 阻塞等待锁
		log.Printf("[StrategyScheduler] Acquired lock for strategy %d", strategy.ID)

		// 在获取锁后检查执行状态
		// 应该有且只有一个pending状态的执行记录等待处理
		var pendingCount int64
		if err := s.db.Model(&pdb.StrategyExecution{}).Where("strategy_id = ? AND status = ?", strategy.ID, "pending").Count(&pendingCount).Error; err != nil {
			log.Printf("[StrategyScheduler] Failed to check pending executions for strategy %d: %v", strategy.ID, err)
			lock.Unlock()
			return
		}

		log.Printf("[StrategyScheduler] Strategy %d has %d pending executions", strategy.ID, pendingCount)

		if pendingCount == 0 {
			log.Printf("[StrategyScheduler] Strategy %d has no pending executions, skipping", strategy.ID)
			lock.Unlock()
			return
		}

		if pendingCount > 1 {
			log.Printf("[StrategyScheduler] Strategy %d has multiple pending executions (%d), cleaning up", strategy.ID, pendingCount)
			// 保留最新的一个，删除其他的
			var executions []pdb.StrategyExecution
			s.db.Where("strategy_id = ? AND status = ?", strategy.ID, "pending").Order("created_at desc").Find(&executions)

			for i := 1; i < len(executions); i++ {
				log.Printf("[StrategyScheduler] Deleting duplicate pending execution %d", executions[i].ID)
				s.db.Delete(&executions[i])
			}
			pendingCount = 1 // 清理后只剩一个
		}

		// 异步执行策略，执行完成后释放锁
		go func() {
			defer func() {
				lock.Unlock()
				if r := recover(); r != nil {
					log.Printf("[StrategyScheduler] Panic in strategy execution goroutine for strategy %d: %v", strategy.ID, r)
				}
			}()
			s.executeStrategy(strategy)
		}()
	}
}

func (s *OrderScheduler) shouldExecuteStrategy(strategy *pdb.TradingStrategy) bool {
	now := time.Now()

	// 如果没有最后运行时间，立即执行
	if strategy.LastRunAt == nil {
		return true
	}

	// 计算下次执行时间
	interval := time.Duration(strategy.RunInterval) * time.Minute
	nextRunTime := strategy.LastRunAt.Add(interval)

	return now.After(nextRunTime) || now.Equal(nextRunTime)
}

// createStrategyExecutionRecord 为策略自动创建执行记录
func (s *OrderScheduler) createStrategyExecutionRecord(strategy *pdb.TradingStrategy) error {
	log.Printf("[StrategyScheduler] Creating automatic execution record for strategy %d", strategy.ID)

	// 创建执行记录，使用策略的默认参数
	execution := &pdb.StrategyExecution{
		StrategyID:     strategy.ID,
		UserID:         strategy.UserID,
		Status:         "pending",
		CurrentStep:    "等待调度器处理",
		StepProgress:   0,
		TotalProgress:  0,
		RunInterval:    strategy.RunInterval, // 继承策略的运行间隔
		MaxRuns:        0,                    // 0表示无限运行
		AutoStop:       false,                // 不自动停止
		CreateOrders:   true,                 // 默认开启订单创建
		ExecutionDelay: 60,                   // 默认60秒延迟
		RunCount:       0,
	}

	// 保存到数据库
	if err := pdb.StartStrategyExecution(s.db, execution); err != nil {
		return fmt.Errorf("创建策略执行记录失败: %w", err)
	}

	// 记录初始日志
	pdb.AppendStrategyExecutionLog(s.db, execution.ID, "策略调度器自动创建执行记录")

	log.Printf("[StrategyScheduler] Successfully created execution record %d for strategy %d", execution.ID, strategy.ID)
	return nil
}

func (s *OrderScheduler) executeStrategy(strategy *pdb.TradingStrategy) {
	// 查找是否有pending状态的执行记录
	var execution pdb.StrategyExecution
	err := s.db.Where("strategy_id = ? AND status = ?", strategy.ID, "pending").First(&execution).Error

	if err != nil {
		// 如果没有pending的执行记录，可能是第一次执行或者有其他问题
		log.Printf("[StrategyScheduler] No pending execution found for strategy %d, skipping: %v", strategy.ID, err)
		return
	}

	// 将执行状态改为running
	if err := pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "running", "开始执行", "", 0, 0, ""); err != nil {
		log.Printf("[StrategyScheduler] Failed to update execution status for strategy %d: %v", strategy.ID, err)
		return
	}

	// 记录初始日志
	pdb.AppendStrategyExecutionLog(s.db, execution.ID, "策略执行调度器开始执行策略")

	// 使用defer确保资源清理和LastRunAt更新
	defer func() {
		if r := recover(); r != nil {
			// 获取详细的堆栈跟踪信息
			stackTrace := make([]byte, 4096)
			stackSize := runtime.Stack(stackTrace, false)
			stackTraceStr := string(stackTrace[:stackSize])

			log.Printf("[StrategyScheduler] Panic in strategy execution %d: %v", execution.ID, r)
			log.Printf("[StrategyScheduler] Stack trace:\n%s", stackTraceStr)

			pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "failed", "执行异常", "", 100, 100, fmt.Sprintf("执行异常: %v\n堆栈跟踪:\n%s", r, stackTraceStr))
			pdb.AppendStrategyExecutionLog(s.db, execution.ID, fmt.Sprintf("策略执行出现异常: %v\n堆栈跟踪:\n%s", r, stackTraceStr))
		}

		// 无论成功还是失败，都更新最后运行时间
		if err := s.db.Model(&pdb.TradingStrategy{}).Where("id = ?", strategy.ID).Update("last_run_at", time.Now()).Error; err != nil {
			log.Printf("[StrategyScheduler] Failed to update last_run_at for strategy %d: %v", strategy.ID, err)
		}
	}()

	// 获取符合条件的交易对
	var eligibleSymbols []string
	var eligibleResults map[string]StrategyDecisionResult

	// 在策略执行前，检查是否需要执行盈利加仓
	if strategy.Conditions.ProfitScalingEnabled {
		log.Printf("[StrategyScheduler] 检查盈利加仓条件...")
		go s.checkProfitScalingForStrategy(strategy)
	}

	// 检查是否启用了币种白名单模式
	useWhitelist := false
	var whitelist []string

	if strategy.Conditions.UseSymbolWhitelist && s.isSymbolWhitelistValid(strategy.Conditions.SymbolWhitelist) {
		// 将datatypes.JSON转换为[]string
		if err := json.Unmarshal(strategy.Conditions.SymbolWhitelist, &whitelist); err != nil {
			log.Printf("[StrategyScheduler] 解析币种白名单失败: %v，使用动态筛选逻辑", err)
		} else {
			useWhitelist = true
		}
	}

	if useWhitelist {
		log.Printf("[StrategyScheduler] 使用币种白名单模式，共%d个指定币种", len(whitelist))
		eligibleSymbols = make([]string, len(whitelist))
		eligibleResults = make(map[string]StrategyDecisionResult)
		copy(eligibleSymbols, whitelist)

		// 对白名单中的每个币种进行基础验证
		for _, symbol := range eligibleSymbols {
			marketData, err := s.getMarketDataForStrategy(symbol)
			if err != nil {
				log.Printf("[StrategyScheduler] 获取%s市场数据失败: %v", symbol, err)
				eligibleResults[symbol] = StrategyDecisionResult{
					Action: "skip",
					Reason: fmt.Sprintf("获取市场数据失败: %v", err),
				}
				continue
			}

			// 执行基础策略检查
			result := executeBasicChecks(symbol, marketData, strategy.Conditions)
			eligibleResults[symbol] = result
			if result.Action == "continue" {
				// 如果基础检查通过，标记为需要完整检查
				eligibleResults[symbol] = StrategyDecisionResult{
					Action: "allow",
					Reason: "白名单币种，等待完整策略检查",
				}
			}
		}
	}

	if !useWhitelist {
		// 使用动态筛选模式
		eligibleSymbols, eligibleResults, err = s.getEligibleSymbolsForStrategy(strategy)
		if err != nil {
			log.Printf("[StrategyScheduler] Failed to get eligible symbols for strategy %d: %v", strategy.ID, err)
			pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "failed", "获取交易对失败", "", 0, 100, err.Error())
			return
		}
	}

	totalSymbols := len(eligibleSymbols)
	if totalSymbols == 0 {
		pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "completed", "无符合条件的交易对", "", 0, 100, "")
		pdb.AppendStrategyExecutionLog(s.db, execution.ID, "未找到符合策略条件的交易对")
		return
	}

	// 执行策略判断
	orderAttempts := 0 // 尝试创建订单的数量
	successCount := 0  // 成功创建订单的数量
	failCount := 0     // 创建订单失败的数量

	for i, symbol := range eligibleSymbols {
		progress := (i * 100) / totalSymbols
		pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "running", fmt.Sprintf("处理交易对 %s", symbol), symbol, 50, progress, "")

		// 创建执行步骤记录
		now := time.Now()
		step := &pdb.StrategyExecutionStep{
			ExecutionID: execution.ID,
			StepName:    fmt.Sprintf("策略判断 - %s", symbol),
			StepType:    "strategy_check",
			Symbol:      symbol,
			Status:      "running",
			StartTime:   &now,
		}

		if err := pdb.CreateStrategyExecutionStep(s.db, step); err != nil {
			log.Printf("[StrategyScheduler] Failed to create execution step: %v", err)
			continue
		}

		// 获取之前筛选时保存的完整策略结果
		result, exists := eligibleResults[symbol]
		if !exists {
			log.Printf("[StrategyScheduler] 警告：找不到%s的策略结果，跳过", symbol)
			continue
		}

		// 添加详细的策略结果诊断日志
		log.Printf("[StrategyScheduler] ===== 策略执行诊断: %s =====", symbol)
		log.Printf("[StrategyScheduler] 策略ID: %d, 策略名称: %s", strategy.ID, strategy.Name)
		log.Printf("[StrategyScheduler] 网格交易启用: %v", strategy.Conditions.GridTradingEnabled)
		log.Printf("[StrategyScheduler] 白名单模式: %v", strategy.Conditions.UseSymbolWhitelist)
		log.Printf("[StrategyScheduler] 决策结果 - 动作: %s, 原因: %s", result.Action, result.Reason)
		log.Printf("[StrategyScheduler] 网格参数 - 上限:%.4f, 下限:%.4f, 层数:%d, 投资:%.2f",
			strategy.Conditions.GridUpperPrice, strategy.Conditions.GridLowerPrice,
			strategy.Conditions.GridLevels, strategy.Conditions.GridInvestmentAmount)

		// 如果是白名单模式且结果是"allow"，说明需要执行完整策略检查
		if strategy.Conditions.UseSymbolWhitelist && result.Action == "allow" && s.server != nil {
			log.Printf("[StrategyScheduler] %s 白名单模式需要完整策略检查，开始执行网格策略", symbol)

			// 获取市场数据
			marketData, err := s.getMarketDataForStrategy(symbol)
			if err != nil {
				log.Printf("[StrategyScheduler] 获取%s市场数据失败: %v", symbol, err)
				result = StrategyDecisionResult{
					Action: "skip",
					Reason: fmt.Sprintf("获取市场数据失败: %v", err),
				}
			} else {
				// 执行完整策略检查（针对网格策略）
				log.Printf("[StrategyScheduler] %s 调用完整策略执行器进行网格决策", symbol)
				result = s.server.executeStrategyWithFullExecutors(context.Background(), symbol, marketData, strategy.Conditions, strategy)
				log.Printf("[StrategyScheduler] %s 完整策略检查完成: action=%s, reason=%s", symbol, result.Action, result.Reason)

				// 更新结果缓存
				eligibleResults[symbol] = result
			}
		}

		// 更新步骤状态
		status := "completed"
		orderCreated := false

		// 只处理实际需要创建订单的情况
		if result.Action == "buy" || result.Action == "sell" || result.Action == "short" {
			orderAttempts++
			log.Printf("[DEBUG] ===== 找到交易信号: %s action=%s =====", symbol, result.Action)
			log.Printf("[DEBUG] execution.CreateOrders=%v", execution.CreateOrders)

			if execution.CreateOrders {
				log.Printf("[DEBUG] >>> 开始创建订单流程: %s %s", symbol, result.Action)
				// 尝试创建订单
				if err := s.createOrderFromStrategyDecision(strategy, symbol, result, execution.ID); err != nil {
					log.Printf("[StrategyScheduler] Failed to create order for %s: %v", symbol, err)
					status = "failed"
					failCount++
				} else {
					orderCreated = true
					successCount++
					log.Printf("[StrategyScheduler] Created order for %s with action %s", symbol, result.Action)
				}
			} else {
				log.Printf("[StrategyScheduler] 跳过为%s创建订单（未开启自动创建）", symbol)
				// 虽然策略判断需要创建订单，但设置了不自动创建，所以标记为跳过
				status = "skipped"
			}
		} else if result.Action == "skip" || result.Action == "no_op" {
			// 策略判断不需要创建订单，直接跳过
			status = "skipped"
		} else if result.Action == "error" {
			// 策略判断出错
			status = "failed"
		}

		// 更新步骤结果信息
		stepResult := fmt.Sprintf("动作: %s, 倍数: %.2f", result.Action, result.Multiplier)
		if orderCreated {
			stepResult += " (已创建订单)"
		}
		pdb.UpdateStrategyExecutionStep(s.db, step.ID, status, result.Reason, "", stepResult)
	}

	// 计算策略执行的总盈亏
	totalPnL := s.calculateStrategyTotalPnL(execution.ID)

	// 重新计算基于实际订单成交状态的统计数据
	var orders []pdb.ScheduledOrder
	if err := s.db.Where("execution_id = ?", execution.ID).Find(&orders).Error; err == nil {
		actualSuccessCount := 0
		actualFailCount := 0
		totalInvestment := 0.0
		currentValue := 0.0

		for _, order := range orders {
			if order.Status == "filled" {
				actualSuccessCount++

				// 计算投资金额和当前价值
				if order.AvgPrice != "" && order.ExecutedQty != "" {
					if entryPrice, err := strconv.ParseFloat(order.AvgPrice, 64); err == nil {
						if quantity, err := strconv.ParseFloat(order.ExecutedQty, 64); err == nil {
							investment := entryPrice * quantity
							totalInvestment += investment

							// 计算当前价值
							if order.Side == "BUY" {
								// 多头仓位：当前价格 × 数量
								if currentPrice, err := s.getCurrentPrice(context.Background(), order.Symbol, "futures"); err == nil {
									currentValue += currentPrice * quantity
								} else {
									// 如果获取当前价格失败，使用开仓价格作为近似值
									currentValue += investment
								}
							} else {
								// 空头仓位：保证金 + 单个订单的盈亏
								margin := investment / float64(order.Leverage)
								// 计算单个订单的盈亏
								if currentPrice, err := s.getCurrentPrice(context.Background(), order.Symbol, "futures"); err == nil {
									orderPnL := (entryPrice - currentPrice) * quantity
									currentValue += margin + orderPnL
								} else {
									// 如果获取当前价格失败，使用保证金作为近似值
									currentValue += margin
								}
							}
						}
					}
				}
			} else if order.Status == "failed" || order.Status == "cancelled" || order.Status == "rejected" {
				actualFailCount++
			}
		}

		// 使用实际成交统计更新计数
		totalOrders := actualSuccessCount + actualFailCount
		actualWinRate := float64(0)
		if totalOrders > 0 {
			actualWinRate = float64(actualSuccessCount) / float64(totalOrders) * 100
		}

		// 计算盈亏百分比
		pnlPercentage := float64(0)
		if totalInvestment > 0 {
			pnlPercentage = (totalPnL / totalInvestment) * 100
		}

		log.Printf("[StrategyScheduler] Final stats - Created: %d orders, Actually executed: %d success, %d failed, Win rate: %.2f%%, PnL: %.8f, PnL%%: %.2f%%, Investment: %.8f, Current Value: %.8f",
			orderAttempts, actualSuccessCount, actualFailCount, actualWinRate, totalPnL, pnlPercentage, totalInvestment, currentValue)

		pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "completed", "执行完成", "", 100, 100, "")
		pdb.UpdateStrategyExecutionResultWithStats(s.db, execution.ID, totalOrders, actualSuccessCount, actualFailCount, totalPnL, actualWinRate, pnlPercentage, totalInvestment, currentValue)
	} else {
		// 如果查询失败，使用原来的统计数据
		log.Printf("[StrategyScheduler] Failed to query orders for final stats: %v, using creation stats", err)
		totalOrders := orderAttempts
		winRate := float64(0)
		if totalOrders > 0 {
			winRate = float64(successCount) / float64(totalOrders) * 100
		}

		pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "completed", "执行完成", "", 100, 100, "")
		pdb.UpdateStrategyExecutionResult(s.db, execution.ID, totalOrders, successCount, failCount, totalPnL, winRate)
	}

	// 增加运行次数
	if err := s.db.Model(execution).Update("run_count", gorm.Expr("run_count + 1")).Error; err != nil {
		log.Printf("[StrategyScheduler] Failed to update run_count for execution %d: %v", execution.ID, err)
	}

	// 日志输出已在上面的统计计算中完成

	// 重新加载execution记录以获取最新的run_count
	if err := s.db.First(&execution, execution.ID).Error; err != nil {
		log.Printf("[StrategyScheduler] Failed to reload execution %d: %v", execution.ID, err)
		return
	}

	// 检查是否需要自动停止策略
	if execution.AutoStop || (execution.MaxRuns > 0 && execution.RunCount >= execution.MaxRuns) {
		log.Printf("[StrategyScheduler] Stopping strategy %d: auto_stop=%v, run_count=%d, max_runs=%d",
			strategy.ID, execution.AutoStop, execution.RunCount, execution.MaxRuns)

		if err := pdb.UpdateStrategyRunningStatus(s.db, strategy.ID, false); err != nil {
			log.Printf("[StrategyScheduler] Failed to auto-stop strategy %d: %v", strategy.ID, err)
		} else {
			log.Printf("[StrategyScheduler] Strategy %d automatically stopped after execution", strategy.ID)
			pdb.AppendStrategyExecutionLog(s.db, execution.ID, "策略已根据启动参数自动停止")
		}
	}
}

// isSymbolWhitelistValid 检查白名单是否有效且非空
func (s *OrderScheduler) isSymbolWhitelistValid(whitelist datatypes.JSON) bool {
	// 检查数据长度不为0且不为"null"
	if len(whitelist) == 0 || string(whitelist) == "null" {
		return false
	}

	// 检查是否为空数组 []
	if string(whitelist) == "[]" {
		return false
	}

	// 尝试解析并检查是否为非空数组
	var symbols []string
	if err := json.Unmarshal(whitelist, &symbols); err != nil {
		return false
	}

	return len(symbols) > 0
}

func (s *OrderScheduler) getEligibleSymbolsForStrategy(strategy *pdb.TradingStrategy) ([]string, map[string]StrategyDecisionResult, error) {
	// 直接使用和前端扫描相同的候选选择器逻辑
	var candidates []string
	var eligibleResults map[string]StrategyDecisionResult

	// 如果有Server实例，直接使用扫描器注册表（和前端扫描完全一致）
	if s.server != nil && s.server.scannerRegistry != nil {
		log.Printf("[StrategyScheduler] 使用统一的扫描器注册表，策略ID: %d", strategy.ID)

		// 选择合适的扫描器（和前端扫描完全相同）
		scanner := s.server.scannerRegistry.SelectScanner(strategy)
		if scanner == nil {
			log.Printf("[StrategyScheduler] 未找到合适的扫描器，使用降级方案")
		} else {
			log.Printf("[StrategyScheduler] 使用扫描器: %s", scanner.GetStrategyType())

			// 执行扫描
			rawResults, err := scanner.Scan(context.Background(), strategy)
			if err != nil {
				log.Printf("[StrategyScheduler] 扫描器执行失败，使用降级方案: %v", err)
			} else {
				log.Printf("[StrategyScheduler] 扫描器找到%d个候选结果", len(rawResults))

				// 转换结果为字符串数组和决策结果映射
				eligibleSymbols := make([]string, 0, len(rawResults))
				eligibleResults = make(map[string]StrategyDecisionResult)

				for _, raw := range rawResults {
					if symbolMap, ok := raw.(map[string]interface{}); ok {
						// 安全获取字符串值
						getStringValue := func(m map[string]interface{}, key string) string {
							if val, ok := m[key]; ok {
								if str, ok := val.(string); ok {
									return str
								}
							}
							return ""
						}

						// 安全获取float64值
						getFloat64Value := func(m map[string]interface{}, key string) float64 {
							if val, ok := m[key]; ok {
								if f, ok := val.(float64); ok {
									return f
								}
							}
							return 0.0
						}

						symbol := getStringValue(symbolMap, "symbol")
						if symbol != "" {
							eligibleSymbols = append(eligibleSymbols, symbol)
							eligibleResults[symbol] = StrategyDecisionResult{
								Action:     getStringValue(symbolMap, "action"),
								Reason:     getStringValue(symbolMap, "reason"),
								Multiplier: getFloat64Value(symbolMap, "multiplier"),
							}
						}
					}
				}

				log.Printf("[StrategyScheduler] 扫描器处理完成，找到%d个符合条件的币种", len(eligibleSymbols))
				return eligibleSymbols, eligibleResults, nil
			}
		}
	}

	// 降级方案：根据策略类型使用不同的数据源
	if strategy.Conditions.MovingAverageEnabled {
		// 均线策略：使用交易量数据获取高活跃币种
		gdb := s.db
		type VolumeStats struct {
			Symbol      string
			Volume      float64
			QuoteVolume float64
		}

		var volumeStats []VolumeStats
		err := gdb.Table("binance_24h_stats").
			Select("symbol, AVG(volume) as volume, AVG(quote_volume) as quote_volume").
			Where("market_type = ? AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)", "spot").
			Group("symbol").
			Having("COUNT(*) >= 1").
			Order("AVG(quote_volume) DESC").
			Limit(50).
			Scan(&volumeStats).Error

		if err == nil && len(volumeStats) > 0 {
			log.Printf("[StrategyScheduler] 均线策略使用交易量数据，找到%d个候选币种", len(volumeStats))
			for _, stat := range volumeStats {
				candidates = append(candidates, stat.Symbol)
			}
		} else {
			log.Printf("[StrategyScheduler] 获取交易量数据失败，使用涨幅榜降级: %v", err)
		}
	}

	return s.filterEligibleSymbols(strategy, candidates)
}

// filterEligibleSymbols 筛选符合策略条件的交易对
func (s *OrderScheduler) filterEligibleSymbols(strategy *pdb.TradingStrategy, candidates []string) ([]string, map[string]StrategyDecisionResult, error) {
	log.Printf("[StrategyScheduler] 开始筛选符合策略条件的交易对，候选币种共%d个", len(candidates))

	var eligibleSymbols []string
	eligibleResults := make(map[string]StrategyDecisionResult)
	for _, symbol := range candidates {
		// 获取市场数据（包括现货/期货状态）
		marketData, err := s.getMarketDataForStrategy(symbol)
		if err != nil {
			log.Printf("[StrategyScheduler] 获取%s市场数据失败: %v", symbol, err)
			continue
		}

		log.Printf("[StrategyScheduler] 检查%s: 排名=%d, 市值=%.0f万, HasSpot=%v, HasFutures=%v",
			symbol, marketData.GainersRank, marketData.MarketCap/10000, marketData.HasSpot, marketData.HasFutures)

		// 对于需要期货交易的策略，先检查交易对是否支持期货
		if strategy.Conditions.ShortOnGainers || strategy.Conditions.FuturesSpotArbEnabled {
			useTestnet := s.cfg.Exchange.Binance.IsTestnet
			futuresClient := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)
			supported, err := futuresClient.IsSymbolSupported(symbol)
			if err != nil {
				log.Printf("[StrategyScheduler] 检查%s期货支持失败: %v", symbol, err)
				continue
			}
			if !supported {
				log.Printf("[StrategyScheduler] %s不支持期货交易，跳过", symbol)
				continue
			}
		}

		// 检查是否启用跳过已在持仓的币种
		if strategy.Conditions.SkipHeldPositions {
			hasOpenPosition, err := s.checkOpenPositionForSymbol(strategy.UserID, symbol)
			if err != nil {
				log.Printf("[StrategyScheduler] 检查%s持仓状态失败: %v", symbol, err)
				continue
			}
			if hasOpenPosition {
				log.Printf("[StrategyScheduler] %s已有未平仓持仓，跳过", symbol)
				continue
			}
		}

		// 执行策略判断（复用strategy_execution.go中的逻辑）
		result := executeStrategyLogic(strategy, symbol, marketData)

		// 如果返回allow且有Server实例，进一步处理外部依赖
		if result.Action == "allow" && s.server != nil {
			log.Printf("[StrategyScheduler] %s 需要外部依赖，执行完整策略检查", symbol)
			result = s.server.executeStrategyWithFullExecutors(context.Background(), symbol, marketData, strategy.Conditions, strategy)
			log.Printf("[StrategyScheduler] %s 完整策略检查结果: action=%s, reason=%s", symbol, result.Action, result.Reason)
		} else if result.Action == "allow" {
			log.Printf("[StrategyScheduler] %s 需要外部依赖但无Server实例，跳过完整检查", symbol)
			continue // 没有Server实例，无法进行完整检查，跳过这个币种
		}

		log.Printf("[StrategyScheduler] 策略判断%s: action=%s, reason=%s", symbol, result.Action, result.Reason)

		// 只收集会触发交易动作的币种（buy或sell或short），不再将allow当作符合条件
		if result.Action == "buy" || result.Action == "sell" || result.Action == "short" {
			eligibleSymbols = append(eligibleSymbols, symbol)
			eligibleResults[symbol] = result
			log.Printf("[StrategyScheduler] 符合条件的交易对: %s (%s)", symbol, result.Reason)
		}
	}

	log.Printf("[StrategyScheduler] 筛选完成，找到%d个符合条件的交易对", len(eligibleSymbols))

	// 如果没有符合条件的交易对，返回空列表（而不是默认列表）
	// 这会让策略执行状态变为"无符合条件的交易对"
	return eligibleSymbols, eligibleResults, nil
}

// 根据策略决策自动创建订单
func (s *OrderScheduler) createOrderFromStrategyDecision(strategy *pdb.TradingStrategy, symbol string, decision StrategyDecisionResult, executionID uint) error {
	// 获取执行配置
	execution, err := pdb.GetStrategyExecution(s.db, strategy.UserID, executionID)
	if err != nil {
		log.Printf("[StrategyScheduler] 获取策略执行配置失败: %v", err)
		// 回退到默认配置
		execution = &pdb.StrategyExecution{ExecutionDelay: 60, CreateOrders: true, PerOrderAmount: 0} // 默认60秒，开启自动创建，每一单金额为0（使用默认）
	} else {
		log.Printf("[StrategyScheduler] 获取到执行配置: CreateOrders=%v, ExecutionDelay=%d, PerOrderAmount=%.2f",
			execution.CreateOrders, execution.ExecutionDelay, execution.PerOrderAmount)
	}
	// 计算杠杆倍数
	leverage := int(decision.Multiplier)
	if leverage < 1 {
		leverage = 1
	}

	// 构建订单参数
	// 将策略动作转换为订单方向
	var orderSide string
	switch decision.Action {
	case "buy":
		orderSide = "BUY"
	case "sell", "short": // short也使用SELL订单（开空仓）
		orderSide = "SELL"
	default:
		orderSide = strings.ToUpper(decision.Action)
	}

	order := &pdb.ScheduledOrder{
		UserID:      strategy.UserID,
		Exchange:    "binance_futures", // 默认交易所
		Testnet:     true,              // 默认测试网
		Symbol:      symbol,
		Side:        orderSide, // BUY 或 SELL
		OrderType:   "MARKET",  // 默认市价单
		Quantity:    "0.001",   // 默认数量，根据交易对可以调整
		Price:       "",
		Leverage:    leverage,
		ReduceOnly:  false,
		StrategyID:  &strategy.ID,                                                          // 关联策略
		ExecutionID: &executionID,                                                          // 关联执行记录
		TriggerTime: time.Now().Add(time.Duration(execution.ExecutionDelay) * time.Second), // 根据配置延迟执行
		Status:      "pending",
		BracketEnabled: strategy.Conditions.EnableStopLoss || strategy.Conditions.EnableTakeProfit ||
			strategy.Conditions.EnableMarginLossStopLoss || strategy.Conditions.EnableMarginProfitTakeProfit, // 根据策略条件启用一键三连（包含保证金止盈止损）
		TPPercent:   strategy.Conditions.TakeProfitPercent,                                                   // 从策略读取止盈百分比
		SLPercent:   strategy.Conditions.StopLossPercent,                                                     // 从策略读取止损百分比
		WorkingType: "MARK_PRICE",                                                                            // 默认使用标记价格
	}

	// 智能计算订单数量（基于币种特点和账户配置）
	order.Quantity = s.calculateSmartOrderQuantity(symbol, leverage, execution.PerOrderAmount)
	log.Printf("[StrategyScheduler] 计算订单数量: %s, 杠杆: %d, 每一单金额: %.2f USDT, 数量: %s",
		symbol, leverage, execution.PerOrderAmount, order.Quantity)

	// 在创建订单前尝试设置保证金模式（阶段一优化：包含重试机制和详细日志）
	log.Printf("[StrategyScheduler] 根据策略配置设置保证金模式...")
	marginResult := s.setMarginTypeForStrategy(strategy, symbol)
	if !marginResult.Success {
		log.Printf("[StrategyScheduler] 保证金模式设置失败: 交易对=%s, 目标模式=%s, 重试次数=%d, 错误=%v",
			symbol, marginResult.MarginType, marginResult.RetryCount, marginResult.Error)

		// 根据错误类型提供不同的处理建议
		if strings.Contains(marginResult.Error.Error(), "存在未成交订单") {
			log.Printf("[StrategyScheduler] 💡 此错误是正常的: 存在未成交订单时无法更改保证金模式")
			log.Printf("[StrategyScheduler] 💡 解决方案: 1) 等待订单成交 2) 手动调整保证金模式 3) 取消未成交订单")
		}

		// 不返回错误，继续创建订单（保证金模式问题不应该阻止交易）
		log.Printf("[StrategyScheduler] 继续创建订单 (保证金模式设置失败不影响订单创建)")
	} else {
		log.Printf("[StrategyScheduler] ✅ 保证金模式设置成功: %s -> %s", symbol, marginResult.MarginType)
	}

	// 创建订单
	log.Printf("[StrategyScheduler] 开始创建订单: userID=%d, symbol=%s, side=%s, quantity=%s", strategy.UserID, symbol, order.Side, order.Quantity)
	log.Printf("[StrategyScheduler] 订单详情: %+v", order)
	if err := s.db.Create(order).Error; err != nil {
		log.Printf("[StrategyScheduler] 数据库创建订单失败: %v", err)
		// 尝试更详细的错误信息
		log.Printf("[StrategyScheduler] 订单字段检查: UserID=%d, Symbol=%s, Side=%s, Quantity=%s, Leverage=%d",
			order.UserID, order.Symbol, order.Side, order.Quantity, order.Leverage)
		return fmt.Errorf("创建订单失败: %v", err)
	}

	log.Printf("[StrategyScheduler] Auto-created order %d for symbol %s with action %s", order.ID, symbol, decision.Action)
	return nil
}

// MarginModeResult 保证金模式设置结果
type MarginModeResult struct {
	Success    bool
	MarginType string
	Error      error
	RetryCount int
	Duration   time.Duration
}

// setMarginTypeForStrategy 根据策略配置设置保证金模式（阶段一优化版）
func (s *OrderScheduler) setMarginTypeForStrategy(strategy *pdb.TradingStrategy, symbol string) *MarginModeResult {
	startTime := time.Now()
	result := &MarginModeResult{
		Success:    false,
		RetryCount: 0,
	}

	// 根据策略的MarginMode设置保证金模式
	marginType := "CROSSED" // 默认全仓
	if strategy.Conditions.MarginMode == "ISOLATED" {
		marginType = "ISOLATED"
	}
	result.MarginType = marginType

	log.Printf("[MarginMode] 开始设置保证金模式: 策略ID=%d, 交易对=%s, 目标模式=%s",
		strategy.ID, symbol, marginType)

	// 创建币安客户端
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	c := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	// 执行设置操作，包含重试机制
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result.RetryCount = attempt

		log.Printf("[MarginMode] 尝试设置 (第%d/%d次): %s -> %s",
			attempt, maxRetries, symbol, marginType)

		code, body, err := c.SetMarginType(symbol, marginType)

		if err == nil && code < 400 {
			// 设置成功
			result.Success = true
			result.Duration = time.Since(startTime)
			log.Printf("[MarginMode] ✅ 设置成功: %s -> %s (耗时: %.2fs)",
				symbol, marginType, result.Duration.Seconds())
			return result
		}

		// 分析错误原因
		bodyStr := string(body)
		result.Error = fmt.Errorf("设置保证金模式失败: code=%d body=%s err=%v", code, bodyStr, err)

		// 检查是否是不可重试的错误
		if strings.Contains(bodyStr, "Margin type cannot be changed if there exists open orders") {
			log.Printf("[MarginMode] ❌ 存在未成交订单，无法设置保证金模式: %s (第%d次尝试)",
				symbol, attempt)
			log.Printf("[MarginMode] 💡 建议: 等待订单成交后再设置，或手动调整保证金模式")
			result.Error = fmt.Errorf("存在未成交订单，暂时无法设置保证金模式: %s", symbol)
			break
		}

		// 检查是否已经是目标模式（这应该是成功的情况）
		if strings.Contains(bodyStr, "No need to change margin type") ||
			strings.Contains(bodyStr, "-4046") {
			log.Printf("[MarginMode] ✅ 保证金模式已经是目标模式: %s -> %s (第%d次尝试)",
				symbol, marginType, attempt)
			log.Printf("[MarginMode] 💡 无需更改，保证金模式设置成功")
			result.Success = true
			result.Duration = time.Since(startTime)
			return result
		}

		// 检查是否是其他不可重试的错误
		if strings.Contains(bodyStr, "Invalid symbol") ||
			strings.Contains(bodyStr, "Invalid marginType") {
			log.Printf("[MarginMode] ❌ 参数错误，无需重试: %s - %s", symbol, bodyStr)
			break
		}

		// 对于网络错误或其他临时错误，进行重试
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * time.Second
			log.Printf("[MarginMode] ⏳ 临时错误，%v后重试: %s (第%d/%d次)",
				waitTime, symbol, attempt, maxRetries)
			time.Sleep(waitTime)
		} else {
			log.Printf("[MarginMode] ❌ 达到最大重试次数，设置失败: %s (共尝试%d次)",
				symbol, maxRetries)
		}
	}

	result.Duration = time.Since(startTime)
	if !result.Success {
		log.Printf("[MarginMode] ❌ 最终失败: %s -> %s (耗时: %.2fs, 错误: %v)",
			symbol, marginType, result.Duration.Seconds(), result.Error)
	}
	return result
}

// checkOpenPositionForSymbol 检查用户是否有指定币种的未平仓持仓
func (s *OrderScheduler) checkOpenPositionForSymbol(userID uint, symbol string) (bool, error) {
	// 查询该用户该币种的未完成订单
	var count int64
	err := s.db.Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND status IN (?, ?, ?, ?)",
			userID, symbol, "pending", "processing", "sent", "filled").
		Count(&count).Error

	if err != nil {
		return false, err
	}

	// 如果有未完成的订单，说明有持仓
	return count > 0, nil
}

// calculateSmartOrderQuantity 智能计算订单数量
func (s *OrderScheduler) calculateSmartOrderQuantity(symbol string, leverage int, perOrderAmount float64) string {
	// 首先尝试动态计算（基于实时价格）
	if dynamicQty := s.calculateDynamicOrderQuantity(symbol, leverage, perOrderAmount); dynamicQty != "" {
		return dynamicQty
	}

	// 如果动态计算失败，使用基础配置兜底
	return s.getFallbackOrderQuantity(symbol)
}

// calculateDynamicOrderQuantity 基于实时价格动态计算订单数量
func (s *OrderScheduler) calculateDynamicOrderQuantity(symbol string, leverage int, perOrderAmount float64) string {
	price, err := s.getCurrentPrice(context.Background(), symbol, "futures")
	if err != nil || price <= 0 {
		// 价格获取失败，但如果用户指定了金额，我们需要基于估算价格继续计算
		if perOrderAmount > 0 {
			log.Printf("[scheduler] 价格获取失败，但用户指定了金额 %.2f USDT，将使用估算价格继续计算", perOrderAmount)
			price = s.estimatePriceForSymbol(symbol)
			if price <= 0 {
				log.Printf("[scheduler] 价格估算也失败，返回空字符串使用fallback")
				return ""
			}
			log.Printf("[scheduler] 使用估算价格 %.6f 进行计算", price)
		} else {
			// 价格获取失败，返回空字符串let fallback处理
			return ""
		}
	}

	// 使用指定的每一单金额，如果为0则使用默认逻辑
	var targetNotional float64
	if perOrderAmount > 0 {
		// 用户指定的金额代表保证金，计算名义价值 = 保证金 × 杠杆
		targetMargin := perOrderAmount
		targetNotional = targetMargin * float64(leverage)
		log.Printf("[scheduler] 使用用户指定的保证金: %.2f USDT, 杠杆: %dx, 计算名义价值: %.2f USDT",
			targetMargin, leverage, targetNotional)
	} else {
		// 根据杠杆调整目标名义价值（默认逻辑）
		if leverage <= 2 {
			targetNotional = 80.0 // 低杠杆，目标80 USDT
		} else if leverage <= 5 {
			targetNotional = 50.0 // 中杠杆，目标50 USDT
		} else if leverage <= 10 {
			targetNotional = 30.0 // 高杠杆，目标30 USDT
		} else {
			targetNotional = 20.0 // 超高杠杆，目标20 USDT
		}
		log.Printf("[scheduler] 使用默认名义价值: %.2f USDT (杠杆: %dx)", targetNotional, leverage)
	}

	// 计算需要的精确数量
	requiredQty := targetNotional / price

	// 获取步长信息以便正确调整
	stepSize, _, _, _, err := s.getLotSizeAndMinNotional(symbol, "futures")
	if err != nil || stepSize <= 0 {
		stepSize = 1.0 // 默认步长
	}

	// 调整数量到合适的步长倍数
	adjustedQty := math.Ceil(requiredQty/stepSize) * stepSize

	// 确保数量在合理范围内
	if adjustedQty < 0.000001 { // 防止数量过小
		adjustedQty = 0.000001
	} else if adjustedQty > 1000000 { // 防止数量过大
		adjustedQty = 1000000
	}

	// 验证最终名义价值是否合理
	finalNotional := adjustedQty * price
	if finalNotional < 5.0 { // 确保至少满足最低名义价值要求
		adjustedQty = math.Ceil(5.0/price/stepSize) * stepSize
	}

	margin := targetNotional / float64(leverage)
	log.Printf("[scheduler] 动态计算%s数量: 价格=%.6f, 杠杆=%dx, 目标名义价值=%.1f, 保证金=%.2f, 计算数量=%.6f",
		symbol, price, leverage, targetNotional, margin, adjustedQty)

	return strconv.FormatFloat(adjustedQty, 'f', -1, 64)
}

// estimatePriceForSymbol 根据交易对估算价格（当实时价格获取失败时使用）
func (s *OrderScheduler) estimatePriceForSymbol(symbol string) float64 {
	// 移除USDT后缀获取基础币种
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	baseSymbol = strings.TrimSuffix(baseSymbol, "BUSD")
	baseSymbol = strings.TrimSuffix(baseSymbol, "USDC")

	// 基于币种估算价格（近似值）
	priceEstimates := map[string]float64{
		"BTC":  60000.0,
		"ETH":  3000.0,
		"BNB":  400.0,
		"ADA":  0.5,
		"SOL":  100.0,
		"DOT":  8.0,
		"DOGE": 0.08,
		"SHIB": 0.00002,
		"XRP":  0.5,
		"LINK": 15.0,
		"LTC":  80.0,
		"BCH":  300.0,
		"ETC":  20.0,
		"XNY":  0.004,
		"BTR":  0.004,
		"FHE":  0.2,
		"ARC":  0.06,
	}

	if price, exists := priceEstimates[baseSymbol]; exists {
		log.Printf("[scheduler] 为 %s 使用估算价格 %.6f", symbol, price)
		return price
	}

	// 对于未知币种，返回中等价格
	log.Printf("[scheduler] %s 使用默认估算价格 1.0", symbol)
	return 1.0
}

// getFallbackOrderQuantity 获取兜底订单数量（原有的硬编码逻辑）
func (s *OrderScheduler) getFallbackOrderQuantity(symbol string) string {
	// 基础数量配置（根据币种特点）
	baseQuantities := map[string]string{
		"BTCUSDT":  "0.001",
		"ETHUSDT":  "0.01",
		"BNBUSDT":  "0.1",
		"ADAUSDT":  "100",
		"SOLUSDT":  "10",
		"DOTUSDT":  "10",
		"DOGEUSDT": "1000",
		"SHIBUSDT": "1000000",
		"XRPUSDT":  "100",
		"LINKUSDT": "10",
		"LTCUSDT":  "1",
		"BCHUSDT":  "0.1",
		"ETCUSDT":  "10",
	}

	// 如果有预定义的数量，使用它
	if qty, exists := baseQuantities[symbol]; exists {
		return qty
	}

	// 对于未定义的币种，根据币种后缀智能推断
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	baseSymbol = strings.TrimSuffix(baseSymbol, "BUSD")
	baseSymbol = strings.TrimSuffix(baseSymbol, "USDC")

	switch {
	case strings.Contains(baseSymbol, "BTC"):
		return "0.001"
	case strings.Contains(baseSymbol, "ETH"):
		return "0.01"
	case len(baseSymbol) <= 3: // 主流币种
		return "1"
	case strings.Contains(strings.ToLower(baseSymbol), "doge") || strings.Contains(strings.ToLower(baseSymbol), "shib"):
		return "1000" // 狗狗币类
	default:
		return "10" // 默认中等数量
	}
}

// 检查并处理超时的策略执行
func (s *OrderScheduler) checkAndHandleTimeoutExecutions() {
	// 获取所有运行时间超过30分钟的执行记录
	timeoutThreshold := time.Now().Add(-30 * time.Minute)

	var timeoutExecutions []pdb.StrategyExecution
	err := s.db.Where("status = ? AND start_time < ?", "running", timeoutThreshold).Find(&timeoutExecutions).Error
	if err != nil {
		log.Printf("[StrategyScheduler] Failed to get timeout executions: %v", err)
		return
	}

	for _, execution := range timeoutExecutions {
		log.Printf("[StrategyScheduler] Handling timeout execution %d for strategy %d", execution.ID, execution.StrategyID)

		// 标记执行为失败
		pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "failed", "执行超时", "", 100, 100, "执行时间超过30分钟自动停止")
		pdb.AppendStrategyExecutionLog(s.db, execution.ID, "策略执行超时，已自动停止")

		// 更新执行持续时间
		pdb.UpdateStrategyExecutionDuration(s.db, execution.ID)
	}
}

// 计算策略执行的总盈亏
func (s *OrderScheduler) calculateStrategyTotalPnL(executionID uint) float64 {
	// 查询所有由该策略执行创建的订单（无论状态如何）
	var orders []pdb.ScheduledOrder
	err := s.db.Where("execution_id = ?", executionID).Find(&orders).Error
	if err != nil {
		log.Printf("[StrategyScheduler] Failed to query orders for execution %d: %v", executionID, err)
		return 0
	}

	totalPnL := 0.0
	filledCount := 0

	for _, order := range orders {
		if order.Status == "filled" && order.AvgPrice != "" {
			// 对于已成交的订单，尝试计算盈亏
			pnl, err := s.calculateOrderPnL(&order)
			if err != nil {
				log.Printf("[StrategyScheduler] Failed to calculate PnL for order %d: %v", order.ID, err)
				continue
			}
			totalPnL += pnl
			filledCount++
		}
	}

	log.Printf("[StrategyScheduler] Calculated total PnL for execution %d: %.8f (based on %d filled orders out of %d total orders)",
		executionID, totalPnL, filledCount, len(orders))

	return totalPnL
}

// 计算单个订单的盈亏
func (s *OrderScheduler) calculateOrderPnL(order *pdb.ScheduledOrder) (float64, error) {
	if order.AvgPrice == "" {
		return 0, fmt.Errorf("no avg price")
	}

	entryPrice, err := strconv.ParseFloat(order.AvgPrice, 64)
	if err != nil || entryPrice <= 0 {
		return 0, fmt.Errorf("invalid entry price: %s", order.AvgPrice)
	}

	// 获取当前市场价格
	ctx := context.Background()
	currentPrice, err := s.getCurrentPrice(ctx, order.Symbol, "futures")
	if err != nil {
		return 0, fmt.Errorf("failed to get current price: %v", err)
	}

	// 获取执行数量
	quantity := 0.0
	if order.ExecutedQty != "" {
		quantity, err = strconv.ParseFloat(order.ExecutedQty, 64)
		if err != nil {
			quantity, err = strconv.ParseFloat(order.AdjustedQuantity, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid quantity")
			}
		}
	} else if order.AdjustedQuantity != "" {
		quantity, err = strconv.ParseFloat(order.AdjustedQuantity, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid quantity")
		}
	} else {
		return 0, fmt.Errorf("no quantity information")
	}

	// 基础盈亏计算（未考虑杠杆和合约大小）
	var pnl float64
	if order.Side == "BUY" {
		// 多头：(当前价格 - 开仓价格) * 数量
		pnl = (currentPrice - entryPrice) * quantity
	} else {
		// 空头：(开仓价格 - 当前价格) * 数量
		pnl = (entryPrice - currentPrice) * quantity
	}

	// 考虑杠杆（如果有的话）
	if order.Leverage > 1 {
		pnl *= float64(order.Leverage)
	}

	// 考虑合约面值（简化处理，对于USDT结算的合约，面值近似为1）
	// 实际应该根据具体合约查询面值信息

	return pnl, nil
}

// 获取策略执行锁
func (s *OrderScheduler) getStrategyLock(strategyID uint) *sync.Mutex {
	s.strategyLockMutex.Lock()
	defer s.strategyLockMutex.Unlock()

	if s.strategyLocks == nil {
		s.strategyLocks = make(map[uint]*sync.Mutex)
	}

	if lock, exists := s.strategyLocks[strategyID]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	s.strategyLocks[strategyID] = lock
	return lock
}

// 检查策略状态一致性
func (s *OrderScheduler) checkStrategyConsistency(strategy *pdb.TradingStrategy) error {
	// 检查是否有正在运行的执行记录
	var runningExecutions []pdb.StrategyExecution
	err := s.db.Where("strategy_id = ? AND status = ?", strategy.ID, "running").Find(&runningExecutions).Error
	if err != nil {
		return fmt.Errorf("failed to check running executions: %v", err)
	}

	// 如果策略标记为运行中，但没有正在运行的执行记录，修复状态
	if strategy.IsRunning && len(runningExecutions) == 0 {
		// 检查最近是否有成功的执行记录
		var recentExecution pdb.StrategyExecution
		err := s.db.Where("strategy_id = ?", strategy.ID).
			Order("created_at desc").
			First(&recentExecution).Error

		if err == nil && recentExecution.Status == "completed" {
			// 如果有成功的执行记录，保持运行状态
			return nil
		} else {
			// 没有成功的执行记录，停止策略
			log.Printf("[StrategyScheduler] Strategy %d marked as running but no active executions, stopping", strategy.ID)
			return pdb.UpdateStrategyRunningStatus(s.db, strategy.ID, false)
		}
	}

	// 如果策略标记为停止，但有正在运行的执行记录，停止这些执行
	if !strategy.IsRunning && len(runningExecutions) > 0 {
		log.Printf("[StrategyScheduler] Strategy %d marked as stopped but has running executions, cleaning up", strategy.ID)
		for _, execution := range runningExecutions {
			pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "stopped", "策略已停止", "", 100, 100, "策略被手动停止")
			pdb.AppendStrategyExecutionLog(s.db, execution.ID, "策略已被手动停止")
		}
	}

	// 如果策略标记为运行中，但有长时间运行的执行记录，可能是残留的僵尸记录
	if strategy.IsRunning && len(runningExecutions) > 0 {
		now := time.Now()
		for _, execution := range runningExecutions {
			// 检查执行是否超过30分钟（可能是僵尸进程）
			if now.Sub(execution.StartTime) > 30*time.Minute {
				log.Printf("[StrategyScheduler] Found zombie execution %d for strategy %d (running for %v), cleaning up",
					execution.ID, strategy.ID, now.Sub(execution.StartTime))

				// 标记为失败并停止策略
				pdb.UpdateStrategyExecutionStatus(s.db, execution.ID, "failed", "僵尸进程清理", "", 100, 100, "执行时间过长，可能是上次程序异常退出导致")
				pdb.AppendStrategyExecutionLog(s.db, execution.ID, "检测到僵尸执行记录，已清理")

				// 停止策略，等待用户手动重启
				log.Printf("[StrategyScheduler] Stopping strategy %d due to zombie execution", strategy.ID)
				return pdb.UpdateStrategyRunningStatus(s.db, strategy.ID, false)
			}
		}
	}

	return nil
}

func (s *OrderScheduler) tick() {
	now := time.Now().UTC()

	var batch []pdb.ScheduledOrder
	// 取到期且尚未处理的订单
	if err := s.db.
		Where("status = ? AND trigger_time <= ?", "pending", now).
		Order("trigger_time asc").
		Limit(20).
		Find(&batch).Error; err != nil {
		return
	}
	for _, ord := range batch {
		// 乐观推进状态，防止并发重复执行
		res := s.db.Model(&pdb.ScheduledOrder{}).
			Where("id = ? AND status = ?", ord.ID, "pending").
			Update("status", "processing")
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		// 优化：使用协程池提交任务，限制并发数量
		order := ord // 避免闭包问题
		s.workerPool.Submit(func() {
			s.execute(order)
		})
	}
}

// executeStrategyCheck 执行订单关联的策略判断
// 返回值：shouldContinue - 是否继续执行订单，modifiedOrder - 修改后的订单（可能为nil），reason - 跳过原因
func (s *OrderScheduler) executeStrategyCheck(o pdb.ScheduledOrder) (shouldContinue bool, modifiedOrder *pdb.ScheduledOrder, reason string) {
	// 如果订单没有关联策略，直接继续
	if o.StrategyID == nil {
		return true, nil, ""
	}

	// 更新执行状态：开始策略判断
	if o.ExecutionID != nil {
		pdb.UpdateStrategyExecutionStatus(s.db, *o.ExecutionID, "running", "策略判断", o.Symbol, 10, 10, "")
		pdb.AppendStrategyExecutionLog(s.db, *o.ExecutionID, fmt.Sprintf("开始对交易对 %s 执行策略判断", o.Symbol))

		// 创建策略判断步骤
		now := time.Now()
		judgeStep := &pdb.StrategyExecutionStep{
			ExecutionID: *o.ExecutionID,
			StepName:    fmt.Sprintf("策略判断 - %s", o.Symbol),
			StepType:    "strategy_check",
			Symbol:      o.Symbol,
			Status:      "running",
			StartTime:   &now,
		}
		pdb.CreateStrategyExecutionStep(s.db, judgeStep)
	}

	// 获取策略
	strategy, err := pdb.GetTradingStrategy(s.db, o.UserID, *o.StrategyID)
	if err != nil {
		log.Printf("[scheduler] Failed to get strategy %d for user %d: %v", *o.StrategyID, o.UserID, err)
		if o.ExecutionID != nil {
			pdb.UpdateStrategyExecutionStatus(s.db, *o.ExecutionID, "running", "策略判断", o.Symbol, 20, 15, fmt.Sprintf("获取策略失败: %v", err))
			pdb.AppendStrategyExecutionLog(s.db, *o.ExecutionID, fmt.Sprintf("获取策略失败: %v", err))
		}
		return false, nil, fmt.Sprintf("获取策略失败: %v", err)
	}

	// 获取市场数据
	marketData, err := s.getMarketDataForStrategy(o.Symbol)
	if err != nil {
		log.Printf("[scheduler] Failed to get market data for %s: %v", o.Symbol, err)
		if o.ExecutionID != nil {
			pdb.UpdateStrategyExecutionStatus(s.db, *o.ExecutionID, "running", "获取市场数据", o.Symbol, 30, 20, fmt.Sprintf("获取市场数据失败: %v", err))
			pdb.AppendStrategyExecutionLog(s.db, *o.ExecutionID, fmt.Sprintf("获取 %s 市场数据失败: %v", o.Symbol, err))
		}
		return false, nil, fmt.Sprintf("获取市场数据失败: %v", err)
	}

	if o.ExecutionID != nil {
		pdb.UpdateStrategyExecutionStatus(s.db, *o.ExecutionID, "running", "执行策略逻辑", o.Symbol, 50, 30, "")
		pdb.AppendStrategyExecutionLog(s.db, *o.ExecutionID, fmt.Sprintf("获取 %s 市场数据成功，开始执行策略逻辑", o.Symbol))
	}

	// 执行策略逻辑
	strategyResult := executeStrategyLogic(strategy, o.Symbol, marketData)
	if strategyResult.Action == "skip" {
		return false, nil, fmt.Sprintf("策略判断跳过: %s", strategyResult.Reason)
	}

	// 根据策略结果调整订单参数
	if strategyResult.Action == "buy" {
		modified := o
		modified.Side = "BUY"
		return true, &modified, ""
	} else if strategyResult.Action == "sell" {
		modified := o
		modified.Side = "SELL"
		return true, &modified, ""
	}

	// 策略允许继续，返回原订单
	return true, nil, ""
}

// validateOrderPrerequisites 验证订单前提条件（交易对支持、杠杆设置）
func (s *OrderScheduler) validateOrderPrerequisites(c *bf.Client, o pdb.ScheduledOrder) error {
	// 验证交易对是否支持期货交易
	supported, err := c.IsSymbolSupported(o.Symbol)
	if err != nil {
		return fmt.Errorf("failed to check symbol support: %v", err)
	}
	if !supported {
		return fmt.Errorf("symbol %s does not support futures trading", o.Symbol)
	}

	// 可选：设置杠杆（改进版：添加持仓检查和错误容忍）
	if o.Leverage > 0 {
		// 首先检查是否有持仓，如果有持仓则跳过杠杆设置
		positions, posErr := c.GetPositions()
		if posErr == nil {
			for _, pos := range positions {
				if strings.ToUpper(pos.Symbol) == o.Symbol && pos.PositionAmt != "0" {
					log.Printf("[scheduler] %s 存在持仓(%.4s)，跳过杠杆设置，使用当前杠杆", o.Symbol, pos.PositionAmt)
					return nil
				}
			}
		}

		// 尝试设置杠杆，最多重试3次
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			code, body, err := c.SetLeverage(o.Symbol, o.Leverage)
			if err == nil && code < 400 {
				// 杠杆设置成功
				log.Printf("[scheduler] 杠杆设置成功: %s -> %dx", o.Symbol, o.Leverage)
				break
			}

			log.Printf("[scheduler] 杠杆设置失败 (尝试 %d/%d): %s, code=%d, body=%s, err=%v",
				attempt, maxRetries, o.Symbol, code, string(body), err)

			// 如果是最后一次尝试，记录错误但不中断订单执行
			if attempt == maxRetries {
				log.Printf("[scheduler] ⚠️ 杠杆设置最终失败，继续执行订单: %s (将使用当前杠杆)", o.Symbol)
				// 不返回错误，让订单继续执行
				break
			}

			// 等待后重试
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	// 注意：保证金模式已在订单创建前设置 (createOrderFromStrategyDecision)
	// 这里不再重复设置，避免与已有订单/持仓冲突

	return nil
}

// prepareOrderPrecision 准备订单的精度调整
// 返回值：adjustedQuantity, adjustedPrice, error
func (s *OrderScheduler) prepareOrderPrecision(o pdb.ScheduledOrder) (string, string, error) {
	// 调整数量和价格精度以避免"Precision is over the maximum defined for this asset"错误
	adjustedQuantity := s.adjustQuantityPrecision(o.Symbol, o.Quantity, o.OrderType)
	// 只有限价单才需要调整价格精度，市价单不需要价格参数
	var adjustedPrice string
	if strings.ToUpper(o.OrderType) == "LIMIT" {
		adjustedPrice = s.adjustPricePrecision(o.Symbol, o.Price)
	} else {
		// 市价单不需要价格参数
		adjustedPrice = ""
	}

	// 保存调整后的数量到数据库
	if adjustedQuantity != o.Quantity {
		_ = s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", o.ID).
			Update("adjusted_quantity", adjustedQuantity).Error
	}

	// 验证精度信息是否有效：检查数据库中是否有该交易对的过滤器信息
	hasValidPrecision := s.hasValidExchangeInfo(o.Symbol)
	if !hasValidPrecision {
		log.Printf("[scheduler] 警告: %s 的精度信息无效，无法从数据库获取过滤器信息", o.Symbol)
		return "", "", fmt.Errorf("无法获取 %s 的精度信息，精度调整失败", o.Symbol)
	}

	// 检查调整是否合理：对于市价单，主要检查数量调整；对于限价单，同时检查价格和数量
	var precisionAdjusted bool
	if strings.ToUpper(o.OrderType) == "LIMIT" {
		// 限价单：价格和数量都需要调整
		precisionAdjusted = (adjustedQuantity != "" && adjustedPrice != "")
	} else {
		// 市价单：只需要数量调整，价格为空是正常的
		precisionAdjusted = (adjustedQuantity != "")
	}

	if !precisionAdjusted {
		log.Printf("[scheduler] 警告: %s 的精度调整结果无效", o.Symbol)
		return "", "", fmt.Errorf("无法获取 %s 的精度信息，精度调整失败", o.Symbol)
	}

	log.Printf("[scheduler] %s 精度调整完成: 数量 %s -> %s, 价格 %s -> %s",
		o.Symbol, o.Quantity, adjustedQuantity, o.Price, adjustedPrice)

	return adjustedQuantity, adjustedPrice, nil
}

// safeTimestamp 生成安全的9位时间戳，确保ClientOrderId长度不会超限
func safeTimestamp() int64 {
	ts := time.Now().Unix()
	// 限制为9位数（到2286年），确保各种ClientOrderId格式都不会超过36字符
	if ts > 999999999 {
		ts = ts % 1000000000
	}
	return ts
}

// generateClientOrderID 生成客户端订单ID
func (s *OrderScheduler) generateClientOrderID(orderID uint, suffix string) string {
	if suffix == "" {
		return fmt.Sprintf("sch-%d-%d", orderID, safeTimestamp())
	}
	return fmt.Sprintf("sch-%d-%s-%d", orderID, suffix, safeTimestamp())
}

// prepareBracketOrder 准备 Bracket 订单的基本信息和验证
// 返回值：adjustedQuantity, adjustedPrice, entryCID, gid, error
func (s *OrderScheduler) prepareBracketOrder(o pdb.ScheduledOrder) (string, string, string, string, error) {
	// 准备订单精度
	adjustedQuantity, adjustedPrice, err := s.prepareOrderPrecision(o)
	if err != nil {
		return "", "", "", "", err
	}

	// 生成全局订单组ID
	gid := s.generateClientOrderID(o.ID, "")

	// 为主订单生成clientOrderId
	entryCID := s.generateClientOrderID(o.ID, "entry")

	// 验证名义价值是否满足账户级别的更严格限制（5 USDT for non-reduce-only orders）- 简化版
	if !o.ReduceOnly {
		ctx := context.Background()
		currentPrice, priceErr := s.getCurrentPrice(ctx, o.Symbol, "futures")
		if priceErr == nil {
			if qty, parseErr := strconv.ParseFloat(adjustedQuantity, 64); parseErr == nil {
				// 对于限价单，使用用户设置的价格计算名义价值；对于市价单，使用当前市场价格
				var notionalPrice float64
				if strings.ToUpper(o.OrderType) == "LIMIT" && adjustedPrice != "" && adjustedPrice != "0" {
					// 限价单：使用用户设置的价格
					if priceVal, priceErr := strconv.ParseFloat(adjustedPrice, 64); priceErr == nil {
						notionalPrice = priceVal
						log.Printf("[scheduler] 限价单使用用户设置价格计算名义价值: %.8f", notionalPrice)
					} else {
						notionalPrice = currentPrice
						log.Printf("[scheduler] 限价单价格解析失败，使用当前市场价格: %.8f", notionalPrice)
					}
				} else {
					// 市价单：使用当前市场价格
					notionalPrice = currentPrice
					log.Printf("[scheduler] 市价单使用当前市场价格计算名义价值: %.8f", notionalPrice)
				}

				// 统一的名义价值验证和调整逻辑
				newAdjustedQuantity, skipOrder, skipReason := s.validateAndAdjustNotional(
					o.Symbol, o.OrderType, qty, notionalPrice, adjustedQuantity, o.Leverage)
				if !skipOrder {
					adjustedQuantity = newAdjustedQuantity // 使用调整后的数量
				}

				if skipOrder {
					log.Printf("[scheduler] 名义价值验证失败，跳过订单: %s", skipReason)
					return "", "", "", "", fmt.Errorf("名义价值验证失败: %s", skipReason)
				}

				// 保证金充足性检查
				sufficient, requiredMargin, availableMargin, marginReason := s.checkMarginSufficiency(
					o.Symbol, qty, notionalPrice, o.Leverage)

				if !sufficient {
					log.Printf("[scheduler] 保证金检查失败: %s", marginReason)
					return "", "", "", "", fmt.Errorf("保证金检查失败: %s", marginReason)
				}

				log.Printf("[scheduler] 保证金检查通过: 所需%.2f USDT，账户可用%.2f USDT",
					requiredMargin, availableMargin)
			}
		}
	}

	return adjustedQuantity, adjustedPrice, entryCID, gid, nil
}

// placeBracketOrder 执行 Bracket 订单的下单和 TP/SL 设置
func (s *OrderScheduler) placeBracketOrder(c *bf.Client, o pdb.ScheduledOrder, adjustedQuantity, adjustedPrice, entryCID, gid string) (success bool, result string) {
	// 使用包含精度重试的下单函数
	_, _, _, success, result = s.handleOrderPlacementWithRetry(c, o, adjustedQuantity, adjustedPrice, entryCID)
	if !success {
		return false, result
	}

	// 获取策略配置，检查是否有保证金止盈止损配置
	var effectiveTPPercent, effectiveSLPercent float64
	if o.StrategyID != nil {
		strategy, err := pdb.GetTradingStrategy(s.db, o.UserID, *o.StrategyID)
		if err == nil {
			log.Printf("[scheduler] 获取到策略配置，用于调整止盈止损百分比")

			// 根据策略配置确定有效的止盈止损百分比
			// 优先使用保证金止盈止损，其次使用传统止盈止损
			if strategy.Conditions.EnableMarginProfitTakeProfit && strategy.Conditions.MarginProfitTakeProfitPercent > 0 {
				effectiveTPPercent = strategy.Conditions.MarginProfitTakeProfitPercent
				log.Printf("[scheduler] 使用保证金盈利止盈: %.2f%%", effectiveTPPercent)
			} else if strategy.Conditions.EnableTakeProfit && strategy.Conditions.TakeProfitPercent > 0 {
				effectiveTPPercent = strategy.Conditions.TakeProfitPercent
				log.Printf("[scheduler] 使用传统止盈: %.2f%%", effectiveTPPercent)
			} else {
				effectiveTPPercent = o.TPPercent // 使用订单中的默认值
			}

			if strategy.Conditions.EnableMarginLossStopLoss && strategy.Conditions.MarginLossStopLossPercent > 0 {
				effectiveSLPercent = strategy.Conditions.MarginLossStopLossPercent
				log.Printf("[scheduler] 使用保证金损失止损: %.2f%%", effectiveSLPercent)
			} else if strategy.Conditions.EnableStopLoss && strategy.Conditions.StopLossPercent > 0 {
				effectiveSLPercent = strategy.Conditions.StopLossPercent
				log.Printf("[scheduler] 使用传统止损: %.2f%%", effectiveSLPercent)
			} else {
				effectiveSLPercent = o.SLPercent // 使用订单中的默认值
			}
		} else {
			log.Printf("[scheduler] 获取策略配置失败，使用订单默认值: %v", err)
			effectiveTPPercent = o.TPPercent
			effectiveSLPercent = o.SLPercent
		}
	} else {
		// 没有关联策略，使用订单中的默认值
		effectiveTPPercent = o.TPPercent
		effectiveSLPercent = o.SLPercent
	}

	// 计算参考入场价
	refPx := ""
	if strings.ToUpper(o.OrderType) == "MARKET" || o.Price == "" {
		if px, e := c.GetMarkPrice(o.Symbol); e == nil && px > 0 {
			refPx = fmt.Sprintf("%.8f", px)
		}
	} else {
		refPx = o.Price
	}
	// 若百分比存在，按参考价计算 TP/SL 绝对值
	var tpPrice, slPrice string

	// 如果使用保证金止盈止损配置，则使用真正的保证金计算
	var useMarginCalculation bool
	if o.StrategyID != nil {
		strategy, err := pdb.GetTradingStrategy(s.db, o.UserID, *o.StrategyID)
		if err == nil {
			useMarginCalculation = strategy.Conditions.EnableMarginLossStopLoss || strategy.Conditions.EnableMarginProfitTakeProfit
		}
	}

	// 将 adjustedQuantity 转换为 float64
	quantityFloat, _ := strconv.ParseFloat(adjustedQuantity, 64)

	// 调试日志：检查保证金计算的条件
	log.Printf("[scheduler] 保证金计算条件检查: symbol=%s, useMarginCalculation=%v, refPx='%s', quantityFloat=%.4f, leverage=%d",
		o.Symbol, useMarginCalculation, refPx, quantityFloat, o.Leverage)

	if useMarginCalculation && refPx != "" && quantityFloat > 0 && o.Leverage > 0 {
		// 使用真正的保证金止盈止损计算
		log.Printf("[scheduler] 使用保证金止盈止损计算: symbol=%s, refPx=%s, quantity=%.4f, leverage=%d, TP%%=%.2f, SL%%=%.2f",
			o.Symbol, refPx, quantityFloat, o.Leverage, effectiveTPPercent, effectiveSLPercent)

		refPriceFloat, _ := strconv.ParseFloat(refPx, 64)
		isLong := strings.ToUpper(o.Side) == "BUY"

		marginRiskManager := execution.NewMarginRiskManager(c)

		// 计算保证金止损价格
		if effectiveSLPercent > 0 {
			stopPrice, err := marginRiskManager.CalculateEstimatedMarginStopLoss(
				refPriceFloat, quantityFloat, float64(o.Leverage), effectiveSLPercent, isLong)
			if err != nil {
				log.Printf("[scheduler] 保证金止损价格计算失败，使用传统计算: %v", err)
				// 回退到传统计算
				f := refPriceFloat
				if isLong {
					rawSlPrice := f * (1.0 - effectiveSLPercent/100.0)
					slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawSlPrice))
				} else {
					rawSlPrice := f * (1.0 + effectiveSLPercent/100.0)
					slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawSlPrice))
				}
			} else {
				slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", stopPrice))
				log.Printf("[scheduler] 保证金止损价格: %.8f -> %s", stopPrice, slPrice)
			}
		}

		// 计算保证金止盈价格
		if effectiveTPPercent > 0 {
			takeProfitPrice, err := marginRiskManager.CalculateEstimatedMarginTakeProfit(
				refPriceFloat, quantityFloat, float64(o.Leverage), effectiveTPPercent, isLong)
			if err != nil {
				log.Printf("[scheduler] 保证金止盈价格计算失败，使用传统计算: %v", err)
				// 回退到传统计算
				f := refPriceFloat
				if isLong {
					rawTpPrice := f * (1.0 + effectiveTPPercent/100.0)
					tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawTpPrice))
				} else {
					rawTpPrice := f * (1.0 - effectiveTPPercent/100.0)
					tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawTpPrice))
				}
			} else {
				tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", takeProfitPrice))
				log.Printf("[scheduler] 保证金止盈价格: %.8f -> %s", takeProfitPrice, tpPrice)
			}
		}
	} else {
		// 使用传统价格百分比计算
		log.Printf("[scheduler] 使用传统价格百分比计算TP/SL: symbol=%s, side=%s, refPx=%s, TP%%=%.2f, SL%%=%.2f",
			o.Symbol, o.Side, refPx, effectiveTPPercent, effectiveSLPercent)

		if effectiveTPPercent > 0 && refPx != "" {
			f, _ := strconv.ParseFloat(refPx, 64)
			if strings.ToUpper(o.Side) == "BUY" {
				rawTpPrice := f * (1.0 + effectiveTPPercent/100.0)
				tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawTpPrice))
				log.Printf("[scheduler] BUY止盈价格计算: %.8f * (1 + %.2f/100) = %.8f -> %s",
					f, effectiveTPPercent, rawTpPrice, tpPrice)
			} else {
				rawTpPrice := f * (1.0 - effectiveTPPercent/100.0)
				tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawTpPrice))
				log.Printf("[scheduler] SELL止盈价格计算: %.8f * (1 - %.2f/100) = %.8f -> %s",
					f, effectiveTPPercent, rawTpPrice, tpPrice)
			}
		}
		if effectiveSLPercent > 0 && refPx != "" {
			f, _ := strconv.ParseFloat(refPx, 64)
			if strings.ToUpper(o.Side) == "BUY" {
				rawSlPrice := f * (1.0 - effectiveSLPercent/100.0)
				slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawSlPrice))
				log.Printf("[scheduler] BUY止损价格计算: %.8f * (1 - %.2f/100) = %.8f -> %s",
					f, effectiveSLPercent, rawSlPrice, slPrice)
			} else {
				rawSlPrice := f * (1.0 + effectiveSLPercent/100.0)
				slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", rawSlPrice))
				log.Printf("[scheduler] SELL止损价格计算: %.8f * (1 + %.2f/100) = %.8f -> %s",
					f, effectiveSLPercent, rawSlPrice, slPrice)
			}
		}
	}
	if tpPrice == "" && strings.TrimSpace(o.TPPrice) != "" {
		tpPrice = s.adjustPricePrecision(o.Symbol, strings.TrimSpace(o.TPPrice))
		log.Printf("[scheduler] 使用固定止盈价格: %s", tpPrice)
	}
	if slPrice == "" && strings.TrimSpace(o.SLPrice) != "" {
		fixedSLPrice := strings.TrimSpace(o.SLPrice)
		log.Printf("[scheduler] 尝试使用固定止损价格: '%s'", fixedSLPrice)

		// 验证固定止损价格是否有效
		if slPriceFloat, parseErr := strconv.ParseFloat(fixedSLPrice, 64); parseErr != nil {
			log.Printf("[scheduler] 错误: 固定止损价格无效 '%s', 无法解析为float: %v", fixedSLPrice, parseErr)
		} else if slPriceFloat <= 0 {
			log.Printf("[scheduler] 错误: 固定止损价格无效 '%s', 必须大于0", fixedSLPrice)
		} else {
			slPrice = s.adjustPricePrecision(o.Symbol, fixedSLPrice)
			log.Printf("[scheduler] 使用固定止损价格: %s -> %s", fixedSLPrice, slPrice)
		}
	}

	// 验证WorkingType参数
	validWorkingTypes := map[string]bool{"MARK_PRICE": true, "CONTRACT_PRICE": true}
	if o.WorkingType == "" {
		o.WorkingType = "MARK_PRICE" // 设置默认值
		log.Printf("[scheduler] 使用默认WorkingType: MARK_PRICE")
	} else if !validWorkingTypes[o.WorkingType] {
		log.Printf("[scheduler] 警告: 无效的WorkingType %s，使用默认值MARK_PRICE", o.WorkingType)
		o.WorkingType = "MARK_PRICE"
	}

	// 验证TP/SL价格的合理性
	if tpPrice != "" && slPrice != "" {
		if err := s.validateAndAdjustTPSLPrices(o, &tpPrice, &slPrice, refPx); err != nil {
			result = err.Error()
			return false, result
		}
	}

	// 检查是否会立即触发
	if refPx != "" {
		refVal, refErr := strconv.ParseFloat(refPx, 64)
		if refErr == nil {
			if tpPrice != "" {
				tpVal, tpErr := strconv.ParseFloat(tpPrice, 64)
				if tpErr == nil && strings.ToUpper(o.Side) == "BUY" && tpVal <= refVal {
					log.Printf("[scheduler] 警告: BUY订单止盈价(%.8f) <= 当前价(%.8f)，可能立即触发", tpVal, refVal)
				} else if tpErr == nil && strings.ToUpper(o.Side) == "SELL" && tpVal >= refVal {
					log.Printf("[scheduler] 警告: SELL订单止盈价(%.8f) >= 当前价(%.8f)，可能立即触发", tpVal, refVal)
				}
			}
			if slPrice != "" {
				slVal, slErr := strconv.ParseFloat(slPrice, 64)
				if slErr == nil && strings.ToUpper(o.Side) == "BUY" && slVal >= refVal {
					log.Printf("[scheduler] 警告: BUY订单止损价(%.8f) >= 当前价(%.8f)，可能立即触发", slVal, refVal)
				} else if slErr == nil && strings.ToUpper(o.Side) == "SELL" && slVal <= refVal {
					log.Printf("[scheduler] 警告: SELL订单止损价(%.8f) <= 当前价(%.8f)，可能立即触发", slVal, refVal)
				}
			}
		}
	}

	// 验证adjustedQuantity不为空
	if adjustedQuantity == "" {
		result = "adjusted quantity is empty for bracket order, cannot place TP/SL orders"
		log.Printf("[scheduler] 错误: %s", result)
		return false, result
	}

	// 对于bracket订单，也需要检查名义价值是否满足账户级别的更严格限制（5 USDT）
	ctx := context.Background()
	currentPrice, priceErr := s.getCurrentPrice(ctx, o.Symbol, "futures")
	if priceErr == nil {
		if qty, parseErr := strconv.ParseFloat(adjustedQuantity, 64); parseErr == nil {
			// 使用统一的名义价值验证和调整逻辑
			newAdjustedQuantity, skipOrder, skipReason := s.validateAndAdjustNotional(
				o.Symbol, o.OrderType, qty, currentPrice, adjustedQuantity, o.Leverage)
			if !skipOrder {
				adjustedQuantity = newAdjustedQuantity // 使用调整后的数量
			}

			if skipOrder {
				log.Printf("[scheduler] 名义价值验证失败，跳过订单: %s", skipReason)
				return false, skipReason
			}
		}
	}

	// 挂 reduceOnly 的出场单（closePosition=true）
	exitSide := "SELL"
	if strings.ToUpper(o.Side) == "SELL" {
		exitSide = "BUY"
	}

	var tpCIDBuilder strings.Builder
	tpCIDBuilder.Grow(len(gid) + 3)
	tpCIDBuilder.WriteString(gid)
	tpCIDBuilder.WriteString("-tp")
	tpCID := tpCIDBuilder.String()

	var slCIDBuilder strings.Builder
	slCIDBuilder.Grow(len(gid) + 3)
	slCIDBuilder.WriteString(gid)
	slCIDBuilder.WriteString("-sl")
	slCID := slCIDBuilder.String()

	// 保存实际使用的TP/SL百分比
	actualTPPercent := effectiveTPPercent
	actualSLPercent := effectiveSLPercent

	// 获取最新的市场价格用于计算实际百分比
	var marketPriceForPercent float64
	if ctx := context.Background(); true {
		if price, err := s.getCurrentPrice(ctx, o.Symbol, "futures"); err == nil {
			marketPriceForPercent = price
		} else {
			// 如果获取失败，使用refPx作为备选
			if refPxFloat, err := strconv.ParseFloat(refPx, 64); err == nil {
				marketPriceForPercent = refPxFloat
			}
		}
	}

	// 如果价格被调整过，计算实际百分比
	if tpPrice != "" && marketPriceForPercent > 0 {
		if tpPriceFloat, err := strconv.ParseFloat(tpPrice, 64); err == nil {
			if strings.ToUpper(o.Side) == "BUY" {
				actualTPPercent = ((tpPriceFloat - marketPriceForPercent) / marketPriceForPercent) * 100
			} else {
				actualTPPercent = ((marketPriceForPercent - tpPriceFloat) / marketPriceForPercent) * 100
			}
		}
	}

	if slPrice != "" && marketPriceForPercent > 0 {
		if slPriceFloat, err := strconv.ParseFloat(slPrice, 64); err == nil {
			if strings.ToUpper(o.Side) == "BUY" {
				actualSLPercent = ((marketPriceForPercent - slPriceFloat) / marketPriceForPercent) * 100
			} else {
				actualSLPercent = ((slPriceFloat - marketPriceForPercent) / marketPriceForPercent) * 100
			}
		}
	}

	// 更新数据库中的实际百分比
	if actualTPPercent != effectiveTPPercent || actualSLPercent != effectiveSLPercent {
		updateData := map[string]interface{}{}
		if tpPrice != "" {
			updateData["actual_tp_percent"] = actualTPPercent
		}
		if slPrice != "" {
			updateData["actual_sl_percent"] = actualSLPercent
		}
		if len(updateData) > 0 {
			err := s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", o.ID).Updates(updateData).Error
			if err != nil {
				log.Printf("[scheduler] 更新实际TP/SL百分比失败: %v", err)
			} else {
				log.Printf("[scheduler] 已更新实际TP/SL百分比: TP=%.2f%%, SL=%.2f%%", actualTPPercent, actualSLPercent)
			}
		}
	}

	// 下TP/SL单，记录成功/失败状态
	tpSuccess := false
	slSuccess := false
	var errors []string

	if tpPrice != "" {
		// 在下止盈单前，使用止盈价格重新验证名义价值
		tpPriceFloat, parseErr := strconv.ParseFloat(tpPrice, 64)
		tpAdjustedQuantity := adjustedQuantity
		if parseErr == nil {
			// 解析数量用于名义价值计算
			if tpQty, qtyErr := strconv.ParseFloat(adjustedQuantity, 64); qtyErr == nil && tpQty > 0 {
				// 使用止盈价格验证名义价值
				newAdjustedQuantity, skipOrder, skipReason := s.validateAndAdjustNotional(
					o.Symbol, "TAKE_PROFIT_MARKET", tpQty, tpPriceFloat, adjustedQuantity, o.Leverage)

				if skipOrder {
					log.Printf("[scheduler] 止盈单名义价值验证失败，跳过下单: %s", skipReason)
					errors = append(errors, fmt.Sprintf("TP跳过: %s", skipReason))
					tpPrice = "" // 标记为不需要下单
				} else {
					tpAdjustedQuantity = newAdjustedQuantity
					if tpAdjustedQuantity != adjustedQuantity {
						log.Printf("[scheduler] 止盈单数量已调整: %s -> %s (使用止盈价格验证)",
							adjustedQuantity, tpAdjustedQuantity)
					}
				}
			}
		}

		if tpPrice != "" {
			log.Printf("[scheduler] 准备下止盈单: symbol=%s, side=%s, tpPrice=%s, quantity=%s, tpCID=%s",
				o.Symbol, exitSide, tpPrice, tpAdjustedQuantity, tpCID)

			// 尝试下止盈单
			tpPlaced := false

			// 首先尝试默认的WorkingType
			if code, body, err := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
				tpPrice, tpAdjustedQuantity, o.WorkingType, true, true, tpCID); err != nil || code >= 400 {
				errorMsg := string(body)
				// 检查是否是精度错误，如果是则重试
				if strings.Contains(errorMsg, "Precision is over the maximum defined for this asset") {
					log.Printf("[scheduler] TP精度错误，尝试自动调整: %s", o.Symbol)

					// 首先尝试调整数量精度
					newTpQuantity := s.autoAdjustQuantityPrecision(o.Symbol, tpAdjustedQuantity, "TAKE_PROFIT_MARKET")
					if newTpQuantity != tpAdjustedQuantity {
						log.Printf("[scheduler] TP尝试数量精度调整: %s -> %s", tpAdjustedQuantity, newTpQuantity)
						if code2, body2, err2 := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
							tpPrice, newTpQuantity, o.WorkingType, true, true, tpCID); err2 == nil && code2 < 400 {
							tpPlaced = true
							log.Printf("[scheduler] TP数量精度调整成功: symbol=%s, tpCID=%s", o.Symbol, tpCID)
							_ = body2 // 避免未使用变量的编译错误
						} else {
							// 数量调整失败，尝试价格精度调整
							log.Printf("[scheduler] TP数量精度调整失败，尝试价格精度重试: %s", o.Symbol)
							strictTpPrice := s.adjustPricePrecisionStrict(o.Symbol, tpPrice)
							if strictTpPrice != tpPrice {
								log.Printf("[scheduler] TP使用严格价格精度重试: %s -> %s", tpPrice, strictTpPrice)
								if code3, body3, err3 := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
									strictTpPrice, tpAdjustedQuantity, o.WorkingType, true, true, tpCID); err3 == nil && code3 < 400 {
									tpPlaced = true
									log.Printf("[scheduler] TP价格精度重试成功: symbol=%s, tpCID=%s", o.Symbol, tpCID)
								} else {
									// 如果还是失败，尝试切换WorkingType
									altWorkingType := "CONTRACT_PRICE"
									if o.WorkingType == "CONTRACT_PRICE" {
										altWorkingType = "MARK_PRICE"
									}
									log.Printf("[scheduler] TP尝试切换WorkingType: %s -> %s", o.WorkingType, altWorkingType)
									if code4, body4, err4 := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
										strictTpPrice, tpAdjustedQuantity, altWorkingType, true, true, tpCID); err4 == nil && code4 < 400 {
										tpPlaced = true
										log.Printf("[scheduler] TP WorkingType切换成功: symbol=%s, tpCID=%s, workingType=%s", o.Symbol, tpCID, altWorkingType)
									} else {
										errors = append(errors, fmt.Sprintf("TP重试失败: qty=%s, price=%s, altWorkingType=%s, err=%s", tpAdjustedQuantity, strictTpPrice, altWorkingType, string(body4)))
									}
									_ = body3 // 避免未使用变量的编译错误
								}
							} else {
								errors = append(errors, fmt.Sprintf("TP价格精度调整失败: %s", tpPrice))
							}
							_ = body2 // 避免未使用变量的编译错误
						}
					} else {
						// 没有可调整的数量，尝试价格精度调整
						strictTpPrice := s.adjustPricePrecisionStrict(o.Symbol, tpPrice)
						if strictTpPrice != tpPrice {
							log.Printf("[scheduler] TP使用严格价格精度重试: %s -> %s", tpPrice, strictTpPrice)
							if code2, body2, err2 := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
								strictTpPrice, tpAdjustedQuantity, o.WorkingType, true, true, tpCID); err2 == nil && code2 < 400 {
								tpPlaced = true
								log.Printf("[scheduler] TP价格精度重试成功: symbol=%s, tpCID=%s", o.Symbol, tpCID)
							} else {
								// 尝试切换WorkingType
								altWorkingType := "CONTRACT_PRICE"
								if o.WorkingType == "CONTRACT_PRICE" {
									altWorkingType = "MARK_PRICE"
								}
								log.Printf("[scheduler] TP尝试切换WorkingType: %s -> %s", o.WorkingType, altWorkingType)
								if code3, body3, err3 := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
									strictTpPrice, tpAdjustedQuantity, altWorkingType, true, true, tpCID); err3 == nil && code3 < 400 {
									tpPlaced = true
									log.Printf("[scheduler] TP WorkingType切换成功: symbol=%s, tpCID=%s, workingType=%s", o.Symbol, tpCID, altWorkingType)
								} else {
									errors = append(errors, fmt.Sprintf("TP重试失败: price=%s, altWorkingType=%s, err=%s", strictTpPrice, altWorkingType, string(body3)))
								}
								_ = body2 // 避免未使用变量的编译错误
							}
						} else {
							errors = append(errors, fmt.Sprintf("TP精度错误无法调整: %s", tpPrice))
						}
					}
				} else {
					errors = append(errors, fmt.Sprintf("TP失败: code=%d body=%s err=%v", code, string(body), err))
				}
				if !tpPlaced {
					log.Printf("[scheduler] 止盈单失败: %s", errors[len(errors)-1])
				}
			} else {
				tpSuccess = true
				log.Printf("[scheduler] 止盈单下单成功: symbol=%s, tpCID=%s", o.Symbol, tpCID)
			}
			if tpPlaced {
				tpSuccess = true
			}
		}

		if slPrice != "" {
			// 在下止损单前，使用止损价格重新验证名义价值
			slPriceFloat, parseErr := strconv.ParseFloat(slPrice, 64)
			slAdjustedQuantity := adjustedQuantity
			if parseErr == nil {
				// 解析数量用于名义价值计算
				if slQty, qtyErr := strconv.ParseFloat(adjustedQuantity, 64); qtyErr == nil && slQty > 0 {
					// 使用止损价格验证名义价值
					newAdjustedQuantity, skipOrder, skipReason := s.validateAndAdjustNotional(
						o.Symbol, "STOP_MARKET", slQty, slPriceFloat, adjustedQuantity, o.Leverage)

					if skipOrder {
						log.Printf("[scheduler] 止损单名义价值验证失败，跳过下单: %s", skipReason)
						errors = append(errors, fmt.Sprintf("SL跳过: %s", skipReason))
						slPrice = "" // 标记为不需要下单
					} else {
						slAdjustedQuantity = newAdjustedQuantity
						if slAdjustedQuantity != adjustedQuantity {
							log.Printf("[scheduler] 止损单数量已调整: %s -> %s (使用止损价格验证)",
								adjustedQuantity, slAdjustedQuantity)
						}
					}
				}
			}

			if slPrice != "" {
				log.Printf("[scheduler] 准备下止损单: symbol=%s, side=%s, slPrice='%s' (len=%d), quantity=%s, slCID=%s",
					o.Symbol, exitSide, slPrice, len(slPrice), slAdjustedQuantity, slCID)

				// 验证slPrice是否有效
				if slPriceFloat, parseErr := strconv.ParseFloat(slPrice, 64); parseErr != nil {
					log.Printf("[scheduler] 错误: slPrice无效 '%s', 无法解析为float: %v", slPrice, parseErr)
					slPrice = "" // 标记为无效，跳过创建
				} else if slPriceFloat <= 0 {
					log.Printf("[scheduler] 错误: slPrice无效 '%s', 必须大于0", slPrice)
					slPrice = "" // 标记为无效，跳过创建
				}
			}

			if slPrice != "" {
				// 尝试下止损单
				slPlaced := false
				if code, body, err := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
					slPrice, slAdjustedQuantity, o.WorkingType, true, true, slCID); err != nil || code >= 400 {
					errorMsg := string(body)
					// 检查是否是精度错误，如果是则重试
					if strings.Contains(errorMsg, "Precision is over the maximum defined for this asset") {
						log.Printf("[scheduler] SL精度错误，尝试自动调整: %s", o.Symbol)

						// 首先尝试调整数量精度
						newSlQuantity := s.autoAdjustQuantityPrecision(o.Symbol, slAdjustedQuantity, "STOP_MARKET")
						if newSlQuantity != slAdjustedQuantity {
							log.Printf("[scheduler] SL尝试数量精度调整: %s -> %s", slAdjustedQuantity, newSlQuantity)
							if code2, body2, err2 := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
								slPrice, newSlQuantity, o.WorkingType, true, true, slCID); err2 == nil && code2 < 400 {
								slPlaced = true
								log.Printf("[scheduler] SL数量精度调整成功: symbol=%s, slCID=%s", o.Symbol, slCID)
								_ = body2 // 避免未使用变量的编译错误
							} else {
								// 数量调整失败，尝试价格精度调整
								log.Printf("[scheduler] SL数量精度调整失败，尝试价格精度重试: %s", o.Symbol)
								strictSlPrice := s.adjustPricePrecisionStrict(o.Symbol, slPrice)
								if strictSlPrice != slPrice {
									log.Printf("[scheduler] SL使用严格价格精度重试: %s -> %s", slPrice, strictSlPrice)
									if code3, body3, err3 := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
										strictSlPrice, slAdjustedQuantity, o.WorkingType, true, true, slCID); err3 == nil && code3 < 400 {
										slPlaced = true
										log.Printf("[scheduler] SL价格精度重试成功: symbol=%s, slCID=%s", o.Symbol, slCID)
									} else {
										// 如果还是失败，尝试切换WorkingType
										altWorkingType := "CONTRACT_PRICE"
										if o.WorkingType == "CONTRACT_PRICE" {
											altWorkingType = "MARK_PRICE"
										}
										log.Printf("[scheduler] SL尝试切换WorkingType: %s -> %s", o.WorkingType, altWorkingType)
										if code4, body4, err4 := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
											strictSlPrice, slAdjustedQuantity, altWorkingType, true, true, slCID); err4 == nil && code4 < 400 {
											slPlaced = true
											log.Printf("[scheduler] SL WorkingType切换成功: symbol=%s, slCID=%s, workingType=%s", o.Symbol, slCID, altWorkingType)
										} else {
											errors = append(errors, fmt.Sprintf("SL重试失败: qty=%s, price=%s, altWorkingType=%s, err=%s", slAdjustedQuantity, strictSlPrice, altWorkingType, string(body4)))
										}
										_ = body3 // 避免未使用变量的编译错误
									}
								} else {
									errors = append(errors, fmt.Sprintf("SL价格精度调整失败: %s", slPrice))
								}
								_ = body2 // 避免未使用变量的编译错误
							}
						} else {
							// 没有可调整的数量，尝试价格精度调整
							strictSlPrice := s.adjustPricePrecisionStrict(o.Symbol, slPrice)
							if strictSlPrice != slPrice {
								log.Printf("[scheduler] SL使用严格价格精度重试: %s -> %s", slPrice, strictSlPrice)
								if code2, body2, err2 := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
									strictSlPrice, slAdjustedQuantity, o.WorkingType, true, true, slCID); err2 == nil && code2 < 400 {
									slPlaced = true
									log.Printf("[scheduler] SL价格精度重试成功: symbol=%s, slCID=%s", o.Symbol, slCID)
								} else {
									// 尝试切换WorkingType
									altWorkingType := "CONTRACT_PRICE"
									if o.WorkingType == "CONTRACT_PRICE" {
										altWorkingType = "MARK_PRICE"
									}
									log.Printf("[scheduler] SL尝试切换WorkingType: %s -> %s", o.WorkingType, altWorkingType)
									if code3, body3, err3 := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
										strictSlPrice, slAdjustedQuantity, altWorkingType, true, true, slCID); err3 == nil && code3 < 400 {
										slPlaced = true
										log.Printf("[scheduler] SL WorkingType切换成功: symbol=%s, slCID=%s, workingType=%s", o.Symbol, slCID, altWorkingType)
									} else {
										errors = append(errors, fmt.Sprintf("SL重试失败: price=%s, altWorkingType=%s, err=%s", strictSlPrice, altWorkingType, string(body3)))
									}
									_ = body2 // 避免未使用变量的编译错误
								}
							} else {
								errors = append(errors, fmt.Sprintf("SL精度错误无法调整: %s", slPrice))
							}
						}
					} else {
						errors = append(errors, fmt.Sprintf("SL失败: code=%d body=%s err=%v", code, string(body), err))
					}
					if !slPlaced {
						log.Printf("[scheduler] 止损单失败: %s", errors[len(errors)-1])
					}
				} else {
					slSuccess = true
					log.Printf("[scheduler] 止损单下单成功: symbol=%s, slCID=%s", o.Symbol, slCID)
				}
				if slPlaced {
					slSuccess = true
				}
			}
		}

		// 检查TP/SL下单结果
		if len(errors) > 0 {
			result = strings.Join(errors, " | ")
			// 如果TP和SL都失败，整个bracket订单失败
			if !tpSuccess && !slSuccess {
				log.Printf("[scheduler] TP和SL都失败，bracket订单执行失败")
				return false, result
			}
			// 如果只有一个失败，记录警告但继续
			log.Printf("[scheduler] 部分TP/SL下单失败，继续执行: %s", result)
		}
		// 为成功的TP/SL订单创建数据库记录
		if tpSuccess {
			tpOrder := &pdb.ScheduledOrder{
				UserID:         o.UserID,
				Exchange:       o.Exchange,
				Testnet:        o.Testnet,
				Symbol:         o.Symbol,
				Side:           exitSide,
				OrderType:      "TAKE_PROFIT_MARKET",
				Quantity:       tpAdjustedQuantity,
				Price:          tpPrice,
				Leverage:       o.Leverage,
				ReduceOnly:     true, // TP/SL订单都是reduce-only
				WorkingType:    o.WorkingType,
				ClientOrderId:  tpCID,
				StrategyID:     o.StrategyID,
				ExecutionID:    o.ExecutionID,
				Status:         "pending",  // 条件订单初始状态为pending
				TriggerTime:    time.Now(), // 条件订单创建时间
				ParentOrderId:  o.ID,       // 关联到主订单
				BracketEnabled: false,      // TP/SL订单本身不是bracket订单
			}
			if err := s.db.Create(tpOrder).Error; err != nil {
				log.Printf("[scheduler] 创建TP订单数据库记录失败: %v", err)
			} else {
				log.Printf("[scheduler] 已创建TP订单数据库记录: ID=%d, ClientID=%s", tpOrder.ID, tpCID)
			}
		}

		if slSuccess {
			slOrder := &pdb.ScheduledOrder{
				UserID:         o.UserID,
				Exchange:       o.Exchange,
				Testnet:        o.Testnet,
				Symbol:         o.Symbol,
				Side:           exitSide,
				OrderType:      "STOP_MARKET",
				Quantity:       adjustedQuantity, // 使用原始数量，因为slAdjustedQuantity可能未定义
				Price:          slPrice,
				Leverage:       o.Leverage,
				ReduceOnly:     true, // TP/SL订单都是reduce-only
				WorkingType:    o.WorkingType,
				ClientOrderId:  slCID,
				StrategyID:     o.StrategyID,
				ExecutionID:    o.ExecutionID,
				Status:         "pending",  // 条件订单初始状态为pending
				TriggerTime:    time.Now(), // 条件订单创建时间
				ParentOrderId:  o.ID,       // 关联到主订单
				BracketEnabled: false,      // TP/SL订单本身不是bracket订单
			}
			if err := s.db.Create(slOrder).Error; err != nil {
				log.Printf("[scheduler] 创建SL订单数据库记录失败: %v", err)
			} else {
				log.Printf("[scheduler] 已创建SL订单数据库记录: ID=%d, ClientID=%s", slOrder.ID, slCID)
			}
		}

		// 保存 BracketLink 记录（忽略错误）
		_ = s.db.Create(&pdb.BracketLink{
			ScheduleID:    o.ID,
			Symbol:        o.Symbol,
			GroupID:       gid,
			EntryClientID: entryCID, // 现在记录entry的clientId
			TPClientID:    tpCID,
			SLClientID:    slCID,
			Status:        "active",
		}).Error
	}
	return true, ""
}

// executeConditionalOrder 执行条件订单（TAKE_PROFIT_MARKET/STOP_MARKET）
func (s *OrderScheduler) executeConditionalOrder(c *bf.Client, o pdb.ScheduledOrder) (success bool, result string) {
	log.Printf("[ConditionalOrder] 执行条件订单: %s, type=%s, clientId=%s", o.Symbol, o.OrderType, o.ClientOrderId)

	// 条件订单应该已经在Bracket订单创建时提交到交易所了
	// 这里只需要验证订单状态或进行必要的重试

	// 检查订单是否已经有ClientOrderId（应该有）
	if o.ClientOrderId == "" {
		return false, "条件订单缺少ClientOrderId"
	}

	// 尝试查询Algo订单状态来验证是否成功创建
	algoOrderStatus, err := c.QueryAlgoOrder(o.Symbol, o.ClientOrderId)
	if err != nil {
		log.Printf("[ConditionalOrder] 查询Algo订单状态失败: %s, %v", o.ClientOrderId, err)
		return false, fmt.Sprintf("查询条件订单状态失败: %v", err)
	}

	// 检查Algo订单状态 - Algo订单有特殊的生命周期
	log.Printf("[ConditionalOrder] Algo订单状态: %s, status=%s, algoId=%d",
		o.ClientOrderId, algoOrderStatus.Status, algoOrderStatus.AlgoId)

	// Algo订单的正常状态
	validStatuses := map[string]bool{
		"NEW":      true, // 已创建（初始状态）
		"WORKING":  true, // 工作中
		"EXECUTED": true, // 已执行
		"FINISHED": true, // 已完成
	}

	if validStatuses[algoOrderStatus.Status] {
		log.Printf("[ConditionalOrder] Algo条件订单状态正常: %s, status=%s", o.ClientOrderId, algoOrderStatus.Status)
		return true, "条件订单执行成功"
	} else if algoOrderStatus.Status == "CANCELED" || algoOrderStatus.Status == "EXPIRED" {
		log.Printf("[ConditionalOrder] Algo条件订单已取消/过期: %s, status=%s", o.ClientOrderId, algoOrderStatus.Status)
		return true, "条件订单已完成" // 取消/过期也是正常的结束状态
	} else {
		log.Printf("[ConditionalOrder] Algo条件订单状态异常: %s, status=%s", o.ClientOrderId, algoOrderStatus.Status)
		return false, fmt.Sprintf("条件订单状态异常: %s", algoOrderStatus.Status)
	}
}

// executeRegularOrder 执行普通订单（非Bracket订单）
func (s *OrderScheduler) executeRegularOrder(c *bf.Client, o pdb.ScheduledOrder) (success bool, result string) {
	// 准备订单精度
	adjustedQuantity, adjustedPrice, err := s.prepareOrderPrecision(o)
	if err != nil {
		return false, err.Error()
	}

	// 执行订单前置交易检查（名义价值、保证金）
	finalQuantity, skip, reason := s.validateOrderPreTradeChecks(o, adjustedQuantity, adjustedPrice)
	if skip {
		return false, reason
	}

	// 为非Bracket订单生成clientOrderId
	// 如果订单已经有ClientOrderId（比如加仓订单），使用已有的；否则生成新的
	var nonBracketCID string
	if o.ClientOrderId != "" {
		nonBracketCID = o.ClientOrderId
		log.Printf("[OrderExecute] 使用已有的ClientOrderId: %s (订单ID: %d)", nonBracketCID, o.ID)
	} else {
		nonBracketCID = s.generateClientOrderID(o.ID, "")
		log.Printf("[OrderExecute] 生成新的ClientOrderId: %s (订单ID: %d)", nonBracketCID, o.ID)
	}

	// 使用包含精度重试的下单函数
	_, _, _, success, result = s.handleOrderPlacementWithRetry(c, o, finalQuantity, adjustedPrice, nonBracketCID)
	return success, result
}

// handleOrderPlacementWithRetry 处理订单下单，包含精度重试逻辑
func (s *OrderScheduler) handleOrderPlacementWithRetry(c *bf.Client, o pdb.ScheduledOrder, quantity, price, clientOrderID string) (code int, body []byte, orderID string, success bool, result string) {
	// 第一次尝试下单
	code, body, err := c.PlaceOrder(o.Symbol, o.Side, o.OrderType, quantity, price, o.ReduceOnly, clientOrderID)
	if err == nil && code < 400 {
		// 下单成功，解析响应
		return s.parseOrderResponse(o, clientOrderID, code, body)
	}

	// 检查是否是精度错误，如果是则尝试重试
	errorMsg := string(body)
	if strings.Contains(errorMsg, "Precision is over the maximum defined for this asset") {
		log.Printf("[scheduler] 检测到精度错误，尝试自动调整: %s", o.Symbol)

		// 首先尝试调整数量精度
		newQuantity := s.autoAdjustQuantityPrecision(o.Symbol, quantity, o.OrderType)
		if newQuantity != quantity {
			log.Printf("[scheduler] 尝试数量精度调整: %s -> %s", quantity, newQuantity)
			code2, body2, err2 := c.PlaceOrder(o.Symbol, o.Side, o.OrderType, newQuantity, price, o.ReduceOnly, clientOrderID)
			if err2 == nil && code2 < 400 {
				log.Printf("[scheduler] 数量精度调整成功: %s", o.Symbol)
				return s.parseOrderResponse(o, clientOrderID, code2, body2)
			}
		}

		// 数量调整失败，尝试价格精度调整
		log.Printf("[scheduler] 数量精度调整失败，尝试价格精度重试: %s", o.Symbol)
		stricterPrice := s.adjustPricePrecisionStrict(o.Symbol, price)
		if stricterPrice != price {
			log.Printf("[scheduler] 使用更严格的价格精度重试: %s -> %s", price, stricterPrice)
			code3, body3, err3 := c.PlaceOrder(o.Symbol, o.Side, o.OrderType, quantity, stricterPrice, o.ReduceOnly, clientOrderID)
			if err3 == nil && code3 < 400 {
				log.Printf("[scheduler] 价格精度重试成功: %s", o.Symbol)
				return s.parseOrderResponse(o, clientOrderID, code3, body3)
			} else {
				result = fmt.Sprintf("precision retry failed: original_price=%s, retry_price=%s, err=%s",
					price, stricterPrice, string(body3))
				log.Printf("[scheduler] 精度重试失败: %s", result)
				return code3, body3, "", false, result
			}
		} else {
			result = fmt.Sprintf("precision error: symbol=%s, qty=%s, price=%s, err=%s",
				o.Symbol, quantity, price, errorMsg)
			log.Printf("[scheduler] 精度错误详情: %s", result)
			return code, body, "", false, result
		}
	} else if strings.Contains(errorMsg, "Order's notional must be no smaller than 5") {
		result = fmt.Sprintf("notional too small: symbol=%s, quantity=%s, final_notional < 5 USDT required for non-reduce-only orders, err=%s",
			o.Symbol, quantity, errorMsg)
		return code, body, "", false, result
	} else {
		result = fmt.Sprintf("order failed: code=%d body=%s err=%v", code, string(body), err)
		return code, body, "", false, result
	}
}

// parseOrderResponse 解析订单响应并更新数据库
func (s *OrderScheduler) parseOrderResponse(o pdb.ScheduledOrder, clientOrderID string, code int, body []byte) (int, []byte, string, bool, string) {
	// 解析订单响应
	orderResp, parseErr := bf.ParsePlaceOrderResp(body)
	if parseErr == nil && orderResp != nil {
		// 更新数据库中的订单跟踪信息
		updateData := map[string]interface{}{
			"client_order_id": clientOrderID,
		}
		if orderResp.OrderId > 0 {
			updateData["exchange_order_id"] = strconv.FormatInt(orderResp.OrderId, 10)
		}
		if orderResp.Status != "" {
			// 如果订单已经成交，更新状态
			if orderResp.Status == "FILLED" {
				updateData["status"] = "filled"
			}
		}
		_ = s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", o.ID).Updates(updateData).Error
		return code, body, strconv.FormatInt(orderResp.OrderId, 10), true, ""
	}

	return code, body, "", true, ""
}

// checkProfitScalingForStrategy 检查策略相关的所有持仓是否需要盈利加仓
func (s *OrderScheduler) checkProfitScalingForStrategy(strategy *pdb.TradingStrategy) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ProfitScaling] Panic in strategy profit scaling check for strategy %d: %v", strategy.ID, r)
		}
	}()

	log.Printf("[ProfitScaling] 检查策略 %d (%s)的所有持仓盈利情况", strategy.ID, strategy.Name)

	// 获取该策略相关的所有已成交非平仓订单，按币种分组
	var orders []pdb.ScheduledOrder
	err := s.db.Where("strategy_id = ? AND status = ? AND reduce_only = ?",
		strategy.ID, "filled", false).Find(&orders).Error

	if err != nil {
		log.Printf("[ProfitScaling] 查询策略订单失败: %v", err)
		return
	}

	if len(orders) == 0 {
		log.Printf("[ProfitScaling] 策略 %d 没有已成交的持仓订单", strategy.ID)
		return
	}

	// 按币种分组订单
	symbolOrders := make(map[string][]pdb.ScheduledOrder)
	for _, order := range orders {
		symbolOrders[order.Symbol] = append(symbolOrders[order.Symbol], order)
	}

	// 检查每个币种的持仓盈利情况
	for symbol, symbolOrders := range symbolOrders {
		// 过滤掉数据不完整的订单
		var validOrders []pdb.ScheduledOrder
		var invalidOrders []pdb.ScheduledOrder

		for _, order := range symbolOrders {
			isValid := true

			// 检查基本数据完整性
			if order.AvgPrice == "" || order.AvgPrice == "0" ||
				order.ExecutedQty == "" || order.ExecutedQty == "0" {
				isValid = false
			}

			// 检查数据一致性：AvgPrice和ExecutedQty必须同时有值或同时为空
			if order.AvgPrice != "" && order.AvgPrice != "0" && (order.ExecutedQty == "" || order.ExecutedQty == "0") {
				isValid = false
			}

			if order.ExecutedQty != "" && order.ExecutedQty != "0" && (order.AvgPrice == "" || order.AvgPrice == "0") {
				isValid = false
			}

			if isValid {
				validOrders = append(validOrders, order)
			} else {
				invalidOrders = append(invalidOrders, order)
			}
		}

		// 如果有数据不完整的订单，尝试同步它们的数据
		if len(invalidOrders) > 0 {
			log.Printf("[ProfitScaling] %s 有 %d 个订单数据不完整，尝试同步", symbol, len(invalidOrders))
			if err := s.syncFilledOrderData(invalidOrders); err != nil {
				log.Printf("[ProfitScaling] 同步 %s 订单数据失败: %v", symbol, err)
			} else {
				// 同步成功后，从数据库重新查询该币种的所有订单以获取最新数据
				log.Printf("[ProfitScaling] 数据同步成功，重新查询 %s 的订单数据", symbol)
				var refreshedOrders []pdb.ScheduledOrder
				err := s.db.Where("user_id = ? AND symbol = ? AND status = ? AND reduce_only = false",
					strategy.UserID, symbol, "filled").
					Find(&refreshedOrders).Error

				if err != nil {
					log.Printf("[ProfitScaling] 重新查询 %s 订单失败: %v", symbol, err)
					// 如果重新查询失败，继续使用旧数据
					validOrders = symbolOrders
				} else {
					log.Printf("[ProfitScaling] 重新查询到 %s 的 %d 个订单", symbol, len(refreshedOrders))
					for i, order := range refreshedOrders {
						log.Printf("[ProfitScaling] 重新查询订单[%d]: ID=%d, ExecutedQty='%s', AvgPrice='%s'",
							i, order.ID, order.ExecutedQty, order.AvgPrice)
					}
					// 使用重新查询的数据重新进行验证
					validOrders = []pdb.ScheduledOrder{}
					invalidOrders = []pdb.ScheduledOrder{}

					for _, order := range refreshedOrders {
						isValid := true

						// 检查基本数据完整性
						if order.AvgPrice == "" || order.AvgPrice == "0" ||
							order.ExecutedQty == "" || order.ExecutedQty == "0" {
							isValid = false
						}

						// 检查数据一致性：AvgPrice和ExecutedQty必须同时有值或同时为空
						if order.AvgPrice != "" && order.AvgPrice != "0" && (order.ExecutedQty == "" || order.ExecutedQty == "0") {
							isValid = false
						}

						if order.ExecutedQty != "" && order.ExecutedQty != "0" && (order.AvgPrice == "" || order.AvgPrice == "0") {
							isValid = false
						}

						if isValid {
							validOrders = append(validOrders, order)
						} else {
							invalidOrders = append(invalidOrders, order)
							log.Printf("[ProfitScaling] 重新验证后订单 %d 仍无效: AvgPrice='%s', ExecutedQty='%s'",
								order.ID, order.AvgPrice, order.ExecutedQty)
						}
					}
				}
			}
		}

		if len(validOrders) == 0 {
			log.Printf("[ProfitScaling] %s 没有有效的数据用于盈利计算", symbol)
			continue
		}

		// 方案一：检查实际持仓状态
		hasActualPosition, positionAmt, err := s.checkActualPosition(symbol)
		if err != nil {
			log.Printf("[ProfitScaling] 检查 %s 实际持仓状态失败: %v，跳过盈利加仓", symbol, err)
			continue
		}

		if !hasActualPosition {
			log.Printf("[ProfitScaling] %s 无实际持仓，跳过盈利加仓检查", symbol)
			continue
		}

		// 方案二：检查24小时内是否有平仓订单（如果策略启用了此选项）
		if strategy.Conditions.SkipCloseOrdersHours > 0 {
			hasRecentCloseOrder, err := s.checkRecentCloseOrderForProfitScaling(strategy, symbol, 24*time.Hour)
			if err != nil {
				log.Printf("[ProfitScaling] 检查 %s 24小时内平仓订单失败: %v，为了安全跳过", symbol, err)
				continue
			}

			if hasRecentCloseOrder {
				log.Printf("[ProfitScaling] %s 24小时内有平仓记录，跳过盈利加仓检查", symbol)
				continue
			}
		}

		log.Printf("[ProfitScaling] %s 开始检查盈利加仓，当前持仓: %s", symbol, positionAmt)

		profitPercent, currentPrice, err := s.calculatePositionProfitPercentForOrders(validOrders)
		if err != nil {
			log.Printf("[ProfitScaling] 计算%s盈利失败: %v", symbol, err)
			continue
		}

		// 考虑杠杆倍数放大利润百分比
		leverage := strategy.Conditions.FuturesPriceShortLeverage
		if leverage <= 0 {
			leverage = 1.0 // 默认无杠杆
		}
		leveragedProfitPercent := profitPercent * leverage

		log.Printf("[ProfitScaling] %s 当前持仓盈利: %.2f%% (杠杆前) / %.2f%% (杠杆后 %.1fx), 阈值: %.2f%%, 当前价格: %.8f",
			symbol, profitPercent*100, leveragedProfitPercent*100, leverage, strategy.Conditions.ProfitScalingPercent, currentPrice)

		// 检查整体仓位止盈止损
		if strategy.Conditions.OverallStopLossEnabled {
			overallProfitPercent := leveragedProfitPercent * 100

			// 检查整体止损（只有当止损百分比>0时才检查）
			if strategy.Conditions.OverallStopLossPercent > 0 && overallProfitPercent <= -strategy.Conditions.OverallStopLossPercent {
				// 在触发整体止损前，先检查实际持仓状态
				hasActualPosition, positionAmt, err := s.checkActualPosition(symbol)
				if err != nil {
					log.Printf("[OverallStopLoss] 检查 %s 实际持仓状态失败: %v，跳过整体止损", symbol, err)
					continue
				}

				if !hasActualPosition {
					log.Printf("[OverallStopLoss] %s 检测到亏损 %.2f%% 但实际无持仓，跳过整体止损", symbol, overallProfitPercent)
					continue
				}

				log.Printf("[OverallStopLoss] %s 整体仓位亏损 %.2f%% (持仓: %s)，达到止损阈值 %.2f%%，触发整体平仓",
					symbol, overallProfitPercent, positionAmt, strategy.Conditions.OverallStopLossPercent)
				log.Printf("[OverallStopLoss] %s 整体仓位亏损 %.2f%% 达到止损阈值 %.2f%%，触发整体平仓",
					symbol, overallProfitPercent, strategy.Conditions.OverallStopLossPercent)
				if err := s.createOverallCloseOrders(strategy, symbol, "整体止损"); err != nil {
					log.Printf("[OverallStopLoss] 创建整体平仓订单失败 %s: %v", symbol, err)
				} else {
					log.Printf("[OverallStopLoss] 成功为 %s 创建整体止损平仓订单", symbol)
					// 重置该币种的加仓计数器
					newSymbolCounts := resetSymbolProfitScalingCount(strategy.Conditions.ProfitScalingSymbolCounts, symbol)
					strategy.Conditions.ProfitScalingSymbolCounts = newSymbolCounts
					s.db.Model(&pdb.TradingStrategy{}).Where("id = ?", strategy.ID).Update("profit_scaling_symbol_counts", newSymbolCounts)
				}
				continue // 已经触发止损，跳过加仓检查
			}

			// 检查整体止盈（只有当止盈百分比>0时才检查）
			if strategy.Conditions.OverallTakeProfitPercent > 0 && overallProfitPercent >= strategy.Conditions.OverallTakeProfitPercent {
				log.Printf("[OverallTakeProfit] %s 整体仓位盈利 %.2f%% 达到止盈阈值 %.2f%%，触发整体平仓",
					symbol, overallProfitPercent, strategy.Conditions.OverallTakeProfitPercent)
				if err := s.createOverallCloseOrders(strategy, symbol, "整体止盈"); err != nil {
					log.Printf("[OverallTakeProfit] 创建整体平仓订单失败 %s: %v", symbol, err)
				} else {
					log.Printf("[OverallTakeProfit] 成功为 %s 创建整体止盈平仓订单", symbol)
					// 重置该币种的加仓计数器
					newSymbolCounts := resetSymbolProfitScalingCount(strategy.Conditions.ProfitScalingSymbolCounts, symbol)
					strategy.Conditions.ProfitScalingSymbolCounts = newSymbolCounts
					s.db.Model(&pdb.TradingStrategy{}).Where("id = ?", strategy.ID).Update("profit_scaling_symbol_counts", newSymbolCounts)
				}
				continue // 已经触发止盈，跳过加仓检查
			}
		}

		// 检查是否达到加仓阈值（使用杠杆放大后的利润百分比）
		if leveragedProfitPercent*100 >= strategy.Conditions.ProfitScalingPercent {
			log.Printf("[ProfitScaling] %s 达到加仓条件 %.2f%% (杠杆后) >= %.2f%%", symbol, leveragedProfitPercent*100, strategy.Conditions.ProfitScalingPercent)

			// 检查加仓次数限制（币种级别）
			symbolCount := getSymbolProfitScalingCount(strategy.Conditions.ProfitScalingSymbolCounts, symbol)
			if symbolCount >= strategy.Conditions.ProfitScalingMaxCount {
				log.Printf("[ProfitScaling] %s 已达到最大加仓次数 %d/%d，跳过本次加仓",
					symbol, symbolCount, strategy.Conditions.ProfitScalingMaxCount)
				continue
			}

			// 检查是否已经有该策略该币种的待执行盈利加仓订单
			var existingScalingOrders []pdb.ScheduledOrder
			err := s.db.Where("strategy_id = ? AND symbol = ? AND status IN (?) AND client_order_id LIKE ?",
				strategy.ID, symbol, []string{"pending", "processing", "sent"}, "PS_%").
				Find(&existingScalingOrders).Error

			if err != nil {
				log.Printf("[ProfitScaling] 查询现有加仓订单失败 %s: %v", symbol, err)
				continue
			}

			if len(existingScalingOrders) > 0 {
				log.Printf("[ProfitScaling] %s 已有 %d 个待执行的盈利加仓订单，跳过本次加仓", symbol, len(existingScalingOrders))
				continue
			}

			log.Printf("[ProfitScaling] %s 开始创建新的盈利加仓订单", symbol)

			// 使用第一个订单作为模板创建加仓订单
			if err := s.createProfitScalingOrder(symbolOrders[0], strategy, strategy.Conditions.ProfitScalingAmount); err != nil {
				log.Printf("[ProfitScaling] 创建加仓订单失败 %s: %v", symbol, err)
			} else {
				log.Printf("[ProfitScaling] 成功为 %s 创建盈利加仓订单", symbol)

				// 更新币种的加仓次数计数器
				newSymbolCounts := updateSymbolProfitScalingCount(strategy.Conditions.ProfitScalingSymbolCounts, symbol, symbolCount+1)
				updateData := map[string]interface{}{
					"profit_scaling_symbol_counts": newSymbolCounts,
				}
				if err := s.db.Model(&pdb.TradingStrategy{}).Where("id = ?", strategy.ID).Updates(updateData).Error; err != nil {
					log.Printf("[ProfitScaling] 更新币种加仓计数器失败: %v", err)
				} else {
					strategy.Conditions.ProfitScalingSymbolCounts = newSymbolCounts
					log.Printf("[ProfitScaling] 策略 %d %s加仓计数器更新为 %d/%d",
						strategy.ID, symbol, symbolCount+1, strategy.Conditions.ProfitScalingMaxCount)
				}
			}
		}
	}
}

// checkAndExecuteProfitScaling 检查并执行盈利加仓策略（单个订单，已废弃）
func (s *OrderScheduler) checkAndExecuteProfitScaling(order pdb.ScheduledOrder) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ProfitScaling] Panic in profit scaling check for order %d: %v", order.ID, r)
		}
	}()

	// 获取策略配置
	strategy, err := pdb.GetTradingStrategy(s.db, order.UserID, *order.StrategyID)
	if err != nil {
		log.Printf("[ProfitScaling] Failed to get strategy %d: %v", *order.StrategyID, err)
		return
	}

	// 检查是否启用了盈利加仓
	if !strategy.Conditions.ProfitScalingEnabled {
		return
	}

	log.Printf("[ProfitScaling] 检查订单 %d (%s)的盈利加仓条件", order.ID, order.Symbol)

	// 计算当前持仓的盈利情况
	profitPercent, err := s.calculatePositionProfitPercent(order.UserID, order.Symbol)
	if err != nil {
		log.Printf("[ProfitScaling] Failed to calculate profit for %s: %v", order.Symbol, err)
		return
	}

	log.Printf("[ProfitScaling] %s 当前盈利: %.2f%%, 阈值: %.2f%%", order.Symbol, profitPercent*100, strategy.Conditions.ProfitScalingPercent)

	// 检查是否达到加仓阈值
	if profitPercent*100 >= strategy.Conditions.ProfitScalingPercent {
		log.Printf("[ProfitScaling] %s 达到加仓条件，开始创建加仓订单", order.Symbol)

		// 创建加仓订单
		if err := s.createProfitScalingOrder(order, strategy, strategy.Conditions.ProfitScalingAmount); err != nil {
			log.Printf("[ProfitScaling] Failed to create profit scaling order for %s: %v", order.Symbol, err)
		} else {
			log.Printf("[ProfitScaling] 成功为 %s 创建盈利加仓订单", order.Symbol)
		}
	}
}

// checkActualPosition 检查指定币种的实际持仓状态
func (s *OrderScheduler) checkActualPosition(symbol string) (bool, string, error) {
	// 使用Binance客户端获取实际持仓信息
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	positions, err := client.GetPositions()
	if err != nil {
		// 如果API调用失败，返回false（避免因为网络问题误触发止损）
		log.Printf("[PositionCheck] 获取持仓信息失败 %s: %v，使用保守策略", symbol, err)
		return false, "0", nil
	}

	// 查找指定币种的持仓
	for _, pos := range positions {
		if pos.Symbol == symbol {
			positionAmt := pos.PositionAmt
			// 检查持仓数量是否不为0
			if positionAmt != "0" && positionAmt != "0.0" && positionAmt != "" {
				return true, positionAmt, nil
			}
			break
		}
	}

	return false, "0", nil
}

// checkRecentCloseOrderForProfitScaling 检查指定时间内该策略是否有平仓订单（用于盈利加仓过滤）
func (s *OrderScheduler) checkRecentCloseOrderForProfitScaling(strategy *pdb.TradingStrategy, symbol string, timeRange time.Duration) (bool, error) {
	// 检查最近N小时内是否有该策略完成的平仓订单
	// 使用UTC时间确保与数据库时区一致（数据库配置loc=UTC）
	// 平仓订单可能有多种完成状态：filled, completed, success
	var closeOrderCount int64
	cutoffTime := time.Now().UTC().Add(-timeRange)

	err := s.db.Table("scheduled_orders").
		Where("strategy_id = ? AND symbol = ? AND status IN (?) AND reduce_only = ? AND created_at >= ?",
			strategy.ID, symbol, []string{"filled", "completed", "success"}, true, cutoffTime).
		Count(&closeOrderCount).Error

	if err != nil {
		log.Printf("[ProfitScaling] 检查24小时内平仓订单失败 %s: %v", symbol, err)
		return false, err
	}

	if closeOrderCount > 0 {
		log.Printf("[ProfitScaling] 发现策略 %d 在 %s 有 %d 个近期平仓订单", strategy.ID, symbol, closeOrderCount)
		return true, nil
	}

	return false, nil
}

// getSymbolProfitScalingCount 获取指定币种的盈利加仓计数器
func getSymbolProfitScalingCount(counts datatypes.JSON, symbol string) int {
	if counts == nil || string(counts) == "" || string(counts) == "{}" {
		return 0
	}

	var countMap map[string]int
	if err := json.Unmarshal([]byte(counts), &countMap); err != nil {
		log.Printf("[ProfitScaling] 解析币种计数器失败: %v, 使用默认值0", err)
		return 0
	}

	count, exists := countMap[symbol]
	if !exists {
		return 0
	}

	return count
}

// updateSymbolProfitScalingCount 更新指定币种的盈利加仓计数器
func updateSymbolProfitScalingCount(counts datatypes.JSON, symbol string, newCount int) datatypes.JSON {
	var countMap map[string]int
	if counts != nil && string(counts) != "" && string(counts) != "{}" {
		if err := json.Unmarshal([]byte(counts), &countMap); err != nil {
			log.Printf("[ProfitScaling] 解析现有计数器失败: %v, 创建新计数器", err)
			countMap = make(map[string]int)
		}
	} else {
		countMap = make(map[string]int)
	}

	countMap[symbol] = newCount

	updatedJSON, err := json.Marshal(countMap)
	if err != nil {
		log.Printf("[ProfitScaling] 序列化计数器失败: %v", err)
		return counts // 返回原值
	}

	return datatypes.JSON(updatedJSON)
}

// resetSymbolProfitScalingCount 重置指定币种的盈利加仓计数器为0
func resetSymbolProfitScalingCount(counts datatypes.JSON, symbol string) datatypes.JSON {
	return updateSymbolProfitScalingCount(counts, symbol, 0)
}

// calculatePositionProfitPercentForOrders 计算指定订单列表的持仓盈利百分比
func (s *OrderScheduler) calculatePositionProfitPercentForOrders(orders []pdb.ScheduledOrder) (float64, float64, error) {
	if len(orders) == 0 {
		return 0, 0, fmt.Errorf("no orders provided")
	}

	// 验证所有订单都是同一个币种
	symbol := orders[0].Symbol
	for i, order := range orders {
		if order.Symbol != symbol {
			return 0, 0, fmt.Errorf("订单币种不一致: 订单[%d]是%s, 但期望是%s", i, order.Symbol, symbol)
		}
	}

	// 获取当前价格
	ctx := context.Background()
	currentPrice, err := s.server.getCurrentPrice(ctx, symbol, "futures")
	if err != nil {
		return 0, 0, err
	}

	log.Printf("[ProfitScaling] %s 当前价格: %.8f", symbol, currentPrice)

	// 计算平均持仓成本，考虑订单方向
	totalCost := 0.0
	totalQuantity := 0.0
	positionSide := "" // BUY 或 SELL

	for _, order := range orders {
		// 确保订单有完整的成交数据
		if order.AvgPrice == "" || order.AvgPrice == "0" ||
			order.ExecutedQty == "" || order.ExecutedQty == "0" {
			log.Printf("[ProfitScaling] 订单 %d 数据不完整，跳过: AvgPrice='%s', ExecutedQty='%s'",
				order.ID, order.AvgPrice, order.ExecutedQty)
			continue
		}

		// 额外的验证：如果AvgPrice有值但ExecutedQty为空，这也是无效数据
		if order.AvgPrice != "" && order.AvgPrice != "0" && (order.ExecutedQty == "" || order.ExecutedQty == "0") {
			log.Printf("[ProfitScaling] 订单 %d 数据不一致，AvgPrice有值但ExecutedQty为空，跳过: AvgPrice='%s', ExecutedQty='%s'",
				order.ID, order.AvgPrice, order.ExecutedQty)
			continue
		}

		if (order.ExecutedQty != "" && order.ExecutedQty != "0") && (order.AvgPrice == "" || order.AvgPrice == "0") {
			log.Printf("[ProfitScaling] 订单 %d 数据不一致，ExecutedQty有值但AvgPrice为空，跳过: AvgPrice='%s', ExecutedQty='%s'",
				order.ID, order.AvgPrice, order.ExecutedQty)
			continue
		}

		price, err := strconv.ParseFloat(order.AvgPrice, 64)
		if err != nil {
			log.Printf("[ProfitScaling] 订单 %d AvgPrice解析失败: %v", order.ID, err)
			continue
		}

		quantity, err := strconv.ParseFloat(order.ExecutedQty, 64)
		if err != nil {
			log.Printf("[ProfitScaling] 订单 %d ExecutedQty解析失败: %v", order.ID, err)
			continue
		}

		// 记录持仓方向
		if positionSide == "" {
			positionSide = order.Side
		} else if positionSide != order.Side {
			log.Printf("[ProfitScaling] 警告: %s 存在不同方向的订单，当前=%s, 新订单=%s", symbol, positionSide, order.Side)
		}

		log.Printf("[ProfitScaling] 订单 %d: Side=%s, AvgPrice=%.8f, ExecutedQty=%.8f, 成本=%.8f",
			order.ID, order.Side, price, quantity, price*quantity)

		totalCost += price * quantity
		totalQuantity += quantity
	}

	if totalQuantity == 0 {
		return 0, 0, fmt.Errorf("no valid quantity found (total orders: %d)", len(orders))
	}

	avgCost := totalCost / totalQuantity
	log.Printf("[ProfitScaling] %s 平均持仓成本: %.8f, 总数量: %.8f, 持仓方向: %s",
		symbol, avgCost, totalQuantity, positionSide)

	// 根据持仓方向计算盈利百分比
	var profitPercent float64
	if positionSide == "SELL" {
		// 做空：价格下跌时盈利
		profitPercent = (avgCost - currentPrice) / avgCost
		log.Printf("[ProfitScaling] %s 做空盈利计算: (%.8f - %.8f) / %.8f = %.4f",
			symbol, avgCost, currentPrice, avgCost, profitPercent)
	} else {
		// 做多：价格上涨时盈利
		profitPercent = (currentPrice - avgCost) / avgCost
		log.Printf("[ProfitScaling] %s 做多盈利计算: (%.8f - %.8f) / %.8f = %.4f",
			symbol, currentPrice, avgCost, avgCost, profitPercent)
	}

	return profitPercent, currentPrice, nil
}

// calculatePositionProfitPercent 计算指定币种的持仓盈利百分比（兼容旧接口）
func (s *OrderScheduler) calculatePositionProfitPercent(userID uint, symbol string) (float64, error) {
	// 查询该用户该币种的所有已成交的非平仓订单
	var orders []pdb.ScheduledOrder
	err := s.db.Where("user_id = ? AND symbol = ? AND status = ? AND reduce_only = ?",
		userID, symbol, "filled", false).Find(&orders).Error

	if err != nil {
		return 0, err
	}

	profitPercent, _, err := s.calculatePositionProfitPercentForOrders(orders)
	return profitPercent, err
}

// createProfitScalingOrder 创建盈利加仓订单
func (s *OrderScheduler) createProfitScalingOrder(originalOrder pdb.ScheduledOrder, strategy *pdb.TradingStrategy, marginAmount float64) error {
	log.Printf("[ProfitScaling] 开始创建加仓订单，原订单ID: %d, 币种: %s, 加仓保证金: %.2f USDT",
		originalOrder.ID, originalOrder.Symbol, marginAmount)

	// 获取当前价格作为订单价格
	ctx := context.Background()
	currentPrice, err := s.server.getCurrentPrice(ctx, originalOrder.Symbol, "futures")
	if err != nil {
		log.Printf("[ProfitScaling] 获取价格失败 %s: %v", originalOrder.Symbol, err)
		return fmt.Errorf("failed to get current price: %v", err)
	}

	log.Printf("[ProfitScaling] 当前价格 %s: %.8f", originalOrder.Symbol, currentPrice)

	// 计算名义价值 = 保证金 × 杠杆
	notionalValue := marginAmount * float64(originalOrder.Leverage)
	log.Printf("[ProfitScaling] 名义价值计算 %s: %.2f USDT × %.1f 倍 = %.2f USDT",
		originalOrder.Symbol, marginAmount, float64(originalOrder.Leverage), notionalValue)

	// 计算加仓数量（基于名义价值）
	quantity := notionalValue / currentPrice
	log.Printf("[ProfitScaling] 加仓数量计算 %s: %.2f USDT / %.8f = %.8f",
		originalOrder.Symbol, notionalValue, currentPrice, quantity)

	// 创建新的加仓订单
	// 注意：加仓订单不继承Bracket设置，因为加仓后整体仓位发生变化，
	// 止损价格需要基于新的总仓位重新计算，而不是沿用单个订单的设置
	// 生成安全的PROFIT_SCALING ClientOrderId，确保不超过36字符
	timestamp := time.Now().Unix()
	// 限制时间戳为9位数（到2286年），确保总长度不超过36字符
	if timestamp > 999999999 {
		timestamp = timestamp % 1000000000
	}
	clientOrderId := fmt.Sprintf("PS_%d_%d", originalOrder.ID, timestamp)
	scalingOrder := &pdb.ScheduledOrder{
		UserID:         originalOrder.UserID,
		Symbol:         originalOrder.Symbol,
		Side:           originalOrder.Side, // 使用相同的方向（买入更多）
		OrderType:      "MARKET",
		Quantity:       fmt.Sprintf("%.8f", quantity),
		Price:          "",
		Leverage:       originalOrder.Leverage,
		ReduceOnly:     false, // 加仓订单不是平仓订单
		StrategyID:     originalOrder.StrategyID,
		ExecutionID:    originalOrder.ExecutionID,
		ParentOrderId:  originalOrder.ID, // 加仓订单引用原始开仓订单作为父订单
		Status:         "pending",
		TriggerTime:    time.Now(),
		ClientOrderId:  clientOrderId,
		BracketEnabled: false, // 加仓订单不设置独立的止损，因为需要考虑整体仓位
		TPPercent:      0,     // 不设置独立的止盈
		SLPercent:      0,     // 不设置独立的止损
		WorkingType:    originalOrder.WorkingType,
		Testnet:        originalOrder.Testnet,
		Exchange:       originalOrder.Exchange,
	}

	log.Printf("[ProfitScaling] 准备保存加仓订单 %s: 数量=%.8f, 杠杆=%.1f, 父订单=%d",
		originalOrder.Symbol, quantity, float64(originalOrder.Leverage), originalOrder.ID)

	// 保存订单到数据库
	if err := s.db.Create(scalingOrder).Error; err != nil {
		log.Printf("[ProfitScaling] 保存加仓订单失败 %s: %v", originalOrder.Symbol, err)
		return fmt.Errorf("failed to create profit scaling order: %v", err)
	}

	log.Printf("[ProfitScaling] ✅ 成功创建加仓订单 %d (%s) for %s, 保证金: %.2f USDT, 名义价值: %.2f USDT, 数量: %.8f, ClientID: %s",
		scalingOrder.ID, clientOrderId, originalOrder.Symbol, marginAmount, notionalValue, quantity, clientOrderId)

	return nil
}

// createOverallCloseOrders 创建整体平仓订单（止盈或止损）
func (s *OrderScheduler) createOverallCloseOrders(strategy *pdb.TradingStrategy, symbol, reason string) error {
	// 查询该策略在该币种上的所有活跃订单
	var activeOrders []pdb.ScheduledOrder
	err := s.db.Where("strategy_id = ? AND symbol = ? AND status IN (?) AND reduce_only = false",
		strategy.ID, symbol, []string{"filled"}).Find(&activeOrders).Error

	if err != nil {
		return fmt.Errorf("查询活跃订单失败: %w", err)
	}

	if len(activeOrders) == 0 {
		return fmt.Errorf("没有找到活跃的持仓订单")
	}

	log.Printf("[OverallClose] 为 %s 创建 %d 个平仓订单 (%s)", symbol, len(activeOrders), reason)

	createdCount := 0
	for _, order := range activeOrders {
		// 获取当前价格用于市价平仓
		ctx := context.Background()
		currentPrice, err := s.server.getCurrentPrice(ctx, symbol, "futures")
		if err != nil {
			log.Printf("[OverallClose] 获取当前价格失败 %s: %v", symbol, err)
			continue
		}

		// 使用简短的reason标识符以符合36字符长度限制
		shortReason := reason
		switch reason {
		case "整体止损":
			shortReason = "STOP_LOSS"
		case "整体止盈":
			shortReason = "TAKE_PROFIT"
		case "整体止损止盈":
			shortReason = "STOP_ALL"
		default:
			// 如果reason太长，截取前8个字符
			if len(reason) > 8 {
				shortReason = reason[:8]
			}
		}

		// 创建平仓订单
		closeOrder := &pdb.ScheduledOrder{
			UserID:        order.UserID,
			Symbol:        symbol,
			Side:          s.getOppositeSide(order.Side), // 相反方向平仓
			OrderType:     "MARKET",
			Quantity:      order.ExecutedQty, // 平掉全部持仓
			Price:         "",
			Leverage:      order.Leverage,
			ReduceOnly:    true, // 平仓订单
			StrategyID:    &strategy.ID,
			ExecutionID:   order.ExecutionID,
			Status:        "pending",
			TriggerTime:   time.Now(),
			ClientOrderId: fmt.Sprintf("OC_%s_%d_%d", shortReason, order.ID, safeTimestamp()),
			WorkingType:   order.WorkingType,
			Testnet:       order.Testnet,
			Exchange:      order.Exchange,
		}

		// 保存平仓订单
		if err := s.db.Create(closeOrder).Error; err != nil {
			log.Printf("[OverallClose] 创建平仓订单失败 %s order %d: %v", symbol, order.ID, err)
			continue
		}

		log.Printf("[OverallClose] 创建平仓订单 %d: %s %s %s (原订单 %d, 当前价格: %.8f)",
			closeOrder.ID, symbol, closeOrder.Side, closeOrder.Quantity, order.ID, currentPrice)
		createdCount++
	}

	if createdCount == 0 {
		return fmt.Errorf("未能创建任何平仓订单")
	}

	log.Printf("[OverallClose] 成功为 %s 创建 %d/%d 个平仓订单 (%s)",
		symbol, createdCount, len(activeOrders), reason)
	return nil
}

// getOppositeSide 获取相反的交易方向
func (s *OrderScheduler) getOppositeSide(side string) string {
	switch side {
	case "BUY":
		return "SELL"
	case "SELL":
		return "BUY"
	default:
		return side
	}
}

// syncFilledOrderData 同步已成交订单的数据（AvgPrice和ExecutedQty）
func (s *OrderScheduler) syncFilledOrderData(orders []pdb.ScheduledOrder) error {
	if len(orders) == 0 {
		return nil
	}

	// 使用配置的环境创建币安客户端
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	syncedCount := 0
	for _, order := range orders {
		// 根据订单类型选择正确的查询API
		var executedQty string
		var avgPrice string

		if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
			// 条件订单使用Algo订单查询
			algoStatus, algoErr := client.QueryAlgoOrder(order.Symbol, order.ClientOrderId)
			if algoErr != nil {
				log.Printf("[Order-Sync] 查询Algo订单 %s 状态失败: %v", order.ClientOrderId, algoErr)
				continue
			}
			executedQty = algoStatus.ExecutedQty
			avgPrice = algoStatus.AvgPrice
		} else {
			// 普通订单使用普通查询
			orderStatus, queryErr := client.QueryOrder(order.Symbol, order.ClientOrderId)
			if queryErr != nil {
				log.Printf("[Order-Sync] 查询订单 %s 状态失败: %v", order.ClientOrderId, queryErr)
				continue
			}
			executedQty = orderStatus.ExecutedQty
			avgPrice = orderStatus.AvgPrice
		}

		// 更新成交数据
		updateData := make(map[string]interface{})
		shouldUpdate := false

		// 准备更新字段
		updateFields := pdb.ScheduledOrder{}

		// 检查是否需要更新成交数量
		if executedQty != "" && executedQty != "0" {
			if order.ExecutedQty == "" || order.ExecutedQty != executedQty {
				updateFields.ExecutedQty = executedQty
				log.Printf("[Order-Sync] 订单 %d ExecutedQty 需要更新: '%s' -> '%s'",
					order.ID, order.ExecutedQty, executedQty)
			}
		} else if order.ExecutedQty != "" && order.ExecutedQty != "0" {
			// 如果交易所返回的ExecutedQty为空但数据库中有值，记录警告
			log.Printf("[Order-Sync] 警告: 订单 %d 交易所返回ExecutedQty为空，但数据库中有值: '%s'",
				order.ID, order.ExecutedQty)
		}

		// 检查是否需要更新平均价格
		if avgPrice != "" && avgPrice != "0" {
			if order.AvgPrice == "" || order.AvgPrice != avgPrice {
				updateFields.AvgPrice = avgPrice
				log.Printf("[Order-Sync] 订单 %d AvgPrice 需要更新: '%s' -> '%s'",
					order.ID, order.AvgPrice, avgPrice)
			}
		} else if order.AvgPrice != "" && order.AvgPrice != "0" {
			// 如果交易所返回的AvgPrice为空但数据库中有值，记录警告
			log.Printf("[Order-Sync] 警告: 订单 %d 交易所返回AvgPrice为空，但数据库中有值: '%s'",
				order.ID, order.AvgPrice)
		}

		// 检查是否有字段需要更新
		if updateFields.ExecutedQty != "" || updateFields.AvgPrice != "" {
			log.Printf("[ProfitScaling] 更新订单 %d 数据: ExecutedQty='%s', AvgPrice='%s'",
				order.ID, updateFields.ExecutedQty, updateFields.AvgPrice)
			err := s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", order.ID).Updates(updateFields).Error
			if err != nil {
				log.Printf("[ProfitScaling] 更新订单 %d 数据失败: %v", order.ID, err)
			} else {
				log.Printf("[ProfitScaling] 成功同步订单 %d 的成交数据", order.ID)
				syncedCount++
				shouldUpdate = true

				// 验证更新是否成功
				var verifyOrder pdb.ScheduledOrder
				if verifyErr := s.db.Where("id = ?", order.ID).First(&verifyOrder).Error; verifyErr == nil {
					log.Printf("[ProfitScaling] 验证更新结果: ID=%d, ExecutedQty='%s', AvgPrice='%s'",
						verifyOrder.ID, verifyOrder.ExecutedQty, verifyOrder.AvgPrice)
				} else {
					log.Printf("[ProfitScaling] 验证更新失败: %v", verifyErr)
				}
			}
		}

		// 如果有数据需要更新
		if shouldUpdate {
			log.Printf("[ProfitScaling] 更新订单 %d 数据: %+v", order.ID, updateData)
			err := s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", order.ID).Updates(updateData).Error
			if err != nil {
				log.Printf("[ProfitScaling] 更新订单 %d 数据失败: %v", order.ID, err)
			} else {
				log.Printf("[ProfitScaling] 成功同步订单 %d 的成交数据", order.ID)
				syncedCount++

				// 验证更新是否成功
				var verifyOrder pdb.ScheduledOrder
				if verifyErr := s.db.Where("id = ?", order.ID).First(&verifyOrder).Error; verifyErr == nil {
					log.Printf("[ProfitScaling] 验证更新结果: ID=%d, ExecutedQty='%s', AvgPrice='%s'",
						verifyOrder.ID, verifyOrder.ExecutedQty, verifyOrder.AvgPrice)
				} else {
					log.Printf("[ProfitScaling] 验证更新失败: %v", verifyErr)
				}
			}
		}
	}

	// 检查Bracket订单的联动取消逻辑
	// 如果某个Bracket订单被执行了，需要取消其他相关的Bracket订单
	for _, order := range orders {
		// 根据订单类型选择正确的查询API
		var status string
		var executedQty string

		if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
			// 条件订单使用Algo订单查询
			algoStatus, algoErr := client.QueryAlgoOrder(order.Symbol, order.ClientOrderId)
			if algoErr != nil {
				log.Printf("[Bracket-Sync] 查询Algo订单 %s 状态失败: %v", order.ClientOrderId, algoErr)
				continue
			}
			status = algoStatus.Status
			executedQty = algoStatus.ExecutedQty
		} else {
			// 普通订单使用普通查询
			orderStatus, queryErr := client.QueryOrder(order.Symbol, order.ClientOrderId)
			if queryErr != nil {
				log.Printf("[Bracket-Sync] 查询订单 %s 状态失败: %v", order.ClientOrderId, queryErr)
				continue
			}
			status = orderStatus.Status
			executedQty = orderStatus.ExecutedQty
		}

		// 检查是否是已成交的订单
		if status == "FILLED" || status == "EXECUTED" ||
			(executedQty != "" && executedQty != "0") {
			// 检查是否有相关的BracketLink
			var bracketLink pdb.BracketLink
			if bracketErr := s.db.Where("entry_client_id = ? OR tp_client_id = ? OR sl_client_id = ?",
				order.ClientOrderId, order.ClientOrderId, order.ClientOrderId).
				First(&bracketLink).Error; bracketErr == nil {
				log.Printf("[Bracket-Sync] 检测到Bracket订单执行: %s (GroupID: %s)",
					order.ClientOrderId, bracketLink.GroupID)

				// 收集需要取消的其他Bracket订单
				var ordersToCancel []string
				var orderTypes []string

				if bracketLink.EntryClientID == order.ClientOrderId {
					// 开仓订单执行了，取消TP和SL订单
					if bracketLink.TPClientID != "" {
						ordersToCancel = append(ordersToCancel, bracketLink.TPClientID)
						orderTypes = append(orderTypes, "止盈")
					}
					if bracketLink.SLClientID != "" {
						ordersToCancel = append(ordersToCancel, bracketLink.SLClientID)
						orderTypes = append(orderTypes, "止损")
					}
					log.Printf("[Bracket-Sync] 开仓订单执行，准备取消相关TP/SL订单")
				} else if bracketLink.TPClientID == order.ClientOrderId {
					// 止盈订单执行了，取消SL订单
					if bracketLink.SLClientID != "" {
						ordersToCancel = append(ordersToCancel, bracketLink.SLClientID)
						orderTypes = append(orderTypes, "止损")
					}
					log.Printf("[Bracket-Sync] 止盈订单执行，准备取消止损订单")
				} else if bracketLink.SLClientID == order.ClientOrderId {
					// 止损订单执行了，取消TP订单
					if bracketLink.TPClientID != "" {
						ordersToCancel = append(ordersToCancel, bracketLink.TPClientID)
						orderTypes = append(orderTypes, "止盈")
					}
					log.Printf("[Bracket-Sync] 止损订单执行，准备取消止盈订单")
				}

				// 执行取消操作
				for i, clientOrderId := range ordersToCancel {
					log.Printf("[Bracket-Sync] 准备取消%s订单: %s", orderTypes[i], clientOrderId)

					// 首先检查订单状态
					var orderToCancel pdb.ScheduledOrder
					err := s.db.Where("client_order_id = ?", clientOrderId).First(&orderToCancel).Error
					if err != nil {
						log.Printf("[Bracket-Sync] 查询待取消订单失败 %s: %v", clientOrderId, err)
						continue
					}

					// 特殊处理：如果是条件订单，总是尝试取消，除非明确已执行
					shouldSkip := false
					if orderToCancel.OrderType == "TAKE_PROFIT_MARKET" || orderToCancel.OrderType == "STOP_MARKET" {
						algoStatus, algoErr := client.QueryAlgoOrder(order.Symbol, clientOrderId)
						if algoErr != nil {
							log.Printf("[Bracket-Sync] 查询Algo订单状态失败 %s: %v", clientOrderId, algoErr)
							// 查询失败时仍尝试取消，因为网络问题不应该阻止取消
						} else {
							// 只有明确已执行的Algo订单才跳过取消
							if algoStatus.Status == "EXECUTED" || algoStatus.Status == "FINISHED" {
								log.Printf("[Bracket-Sync] Algo订单%s已执行，跳过取消 (状态: %s)", clientOrderId, algoStatus.Status)
								shouldSkip = true
							}
						}
					} else {
						// 对于普通订单，如果已执行则跳过取消
						if orderToCancel.Status == "filled" || orderToCancel.Status == "executed" {
							log.Printf("[Bracket-Sync] 订单%s已执行，跳过取消 (状态: %s)", clientOrderId, orderToCancel.Status)
							shouldSkip = true
						}
					}

					if shouldSkip {
						continue
					}

					log.Printf("[Bracket-Sync] 执行取消%s订单: %s", orderTypes[i], clientOrderId)

					// 取消交易所订单
					cancelCode, cancelBody, cancelErr := client.CancelOrder(order.Symbol, clientOrderId)
					if cancelErr != nil {
						log.Printf("[Bracket-Sync] 取消订单失败 %s: %v", clientOrderId, cancelErr)
					} else if cancelCode >= 400 {
						cancelResp := string(cancelBody)
						log.Printf("[Bracket-Sync] 取消订单响应错误 %s: code=%d, body=%s",
							clientOrderId, cancelCode, cancelResp)

						// 如果是"订单不存在"或"订单已执行"等错误，更新状态
						if strings.Contains(cancelResp, "Order does not exist") ||
							strings.Contains(cancelResp, "Order has been executed") {
							updateErr := s.db.Model(&pdb.ScheduledOrder{}).
								Where("client_order_id = ?", clientOrderId).
								Update("status", "filled").Error
							if updateErr != nil {
								log.Printf("[Bracket-Sync] 更新订单状态失败 %s: %v", clientOrderId, updateErr)
							} else {
								log.Printf("[Bracket-Sync] 订单%s状态更新为filled", clientOrderId)
							}
						}
					} else {
						log.Printf("[Bracket-Sync] 成功取消订单: %s", clientOrderId)

						// 更新数据库中的订单状态
						updateErr := s.db.Model(&pdb.ScheduledOrder{}).
							Where("client_order_id = ?", clientOrderId).
							Update("status", "cancelled").Error
						if updateErr != nil {
							log.Printf("[Bracket-Sync] 更新订单状态失败 %s: %v", clientOrderId, updateErr)
						}
					}
				}

				// 更新BracketLink状态
				bracketUpdates := make(map[string]interface{})
				if bracketLink.EntryClientID == order.ClientOrderId {
					bracketUpdates["status"] = "closed"
				} else {
					// 部分订单执行，标记为部分完成
					bracketUpdates["status"] = "partial"
				}

				s.db.Model(&pdb.BracketLink{}).Where("id = ?", bracketLink.ID).Updates(bracketUpdates)
				log.Printf("[Bracket-Sync] 更新BracketLink状态: ID=%d, Status=%s",
					bracketLink.ID, bracketUpdates["status"])
			}
		}
	}

	log.Printf("[ProfitScaling] 订单数据同步完成: %d/%d 个订单更新成功", syncedCount, len(orders))
	return nil
}

// validateOrderPreTradeChecks 执行订单前置交易检查（名义价值、保证金）
func (s *OrderScheduler) validateOrderPreTradeChecks(o pdb.ScheduledOrder, quantity, price string) (adjustedQuantity string, skip bool, reason string) {
	if o.ReduceOnly {
		// ReduceOnly 订单不需要这些检查
		return quantity, false, ""
	}

	ctx := context.Background()
	currentPrice, priceErr := s.getCurrentPrice(ctx, o.Symbol, "futures")
	if priceErr != nil {
		log.Printf("[scheduler] 获取当前价格失败: %v", priceErr)
		return quantity, false, "" // 不跳过，继续执行
	}

	qty, parseErr := strconv.ParseFloat(quantity, 64)
	if parseErr != nil {
		log.Printf("[scheduler] 解析数量失败: %v", parseErr)
		return quantity, false, ""
	}

	// 计算名义价值价格
	var notionalPrice float64
	if strings.ToUpper(o.OrderType) == "LIMIT" && price != "" && price != "0" {
		// 限价单：使用用户设置的价格
		if priceVal, priceErr := strconv.ParseFloat(price, 64); priceErr == nil {
			notionalPrice = priceVal
			log.Printf("[scheduler] 限价单使用用户设置价格计算名义价值: %.8f", notionalPrice)
		} else {
			notionalPrice = currentPrice
			log.Printf("[scheduler] 限价单价格解析失败，使用当前市场价格: %.8f", notionalPrice)
		}
	} else {
		// 市价单：使用当前市场价格
		notionalPrice = currentPrice
		log.Printf("[scheduler] 市价单使用当前市场价格计算名义价值: %.8f", notionalPrice)
	}

	// 统一的名义价值验证和调整逻辑
	newAdjustedQuantity, skipOrder, skipReason := s.validateAndAdjustNotional(
		o.Symbol, o.OrderType, qty, notionalPrice, quantity, o.Leverage)
	if !skipOrder {
		adjustedQuantity = newAdjustedQuantity
	}

	if skipOrder {
		// 如果名义价值不足，尝试重新调整数量精度（使用最新的价格）
		log.Printf("[scheduler] 名义价值验证失败，尝试重新调整数量精度: %s", skipReason)
		reAdjustedQuantity := s.adjustQuantityPrecision(o.Symbol, quantity, o.OrderType)
		if reAdjustedQuantity != quantity {
			// 重新验证调整后的数量
			reQty, parseErr := strconv.ParseFloat(reAdjustedQuantity, 64)
			if parseErr == nil {
				reNotional := reQty * notionalPrice
				if reNotional >= 5.0 {
					log.Printf("[scheduler] 重新调整后名义价值满足要求: %s %.4f USDT", o.Symbol, reNotional)
					adjustedQuantity = reAdjustedQuantity
					skipOrder = false
					skipReason = ""
				} else {
					log.Printf("[scheduler] 即使重新调整名义价值仍不足: %s %.4f USDT", o.Symbol, reNotional)
				}
			}
		}

		if skipOrder {
			log.Printf("[scheduler] 名义价值验证最终失败，跳过订单: %s", skipReason)
			return adjustedQuantity, true, skipReason
		}
	}

	// 保证金充足性检查
	sufficient, requiredMargin, availableMargin, marginReason := s.checkMarginSufficiency(
		o.Symbol, qty, notionalPrice, o.Leverage)

	if !sufficient {
		log.Printf("[scheduler] 保证金检查失败: %s", marginReason)
		return adjustedQuantity, true, marginReason
	}

	log.Printf("[scheduler] 保证金检查通过: 所需%.2f USDT，账户可用%.2f USDT",
		requiredMargin, availableMargin)

	return adjustedQuantity, false, ""
}

// executeBracketOrder 执行 Bracket 订单（包含TP/SL）
func (s *OrderScheduler) executeBracketOrder(c *bf.Client, o pdb.ScheduledOrder) (success bool, result string) {
	// 准备 Bracket 订单的基本信息和验证
	adjustedQuantity, adjustedPrice, entryCID, gid, err := s.prepareBracketOrder(o)
	if err != nil {
		return false, err.Error()
	}

	// 执行 Bracket 订单的下单和 TP/SL 设置
	return s.placeBracketOrder(c, o, adjustedQuantity, adjustedPrice, entryCID, gid)
}

// executeExchangeOrder 根据交易所类型执行订单
// 返回值：success - 是否执行成功，result - 执行结果或错误信息
func (s *OrderScheduler) executeExchangeOrder(o pdb.ScheduledOrder) (success bool, result string) {
	ex := strings.ToLower(o.Exchange)

	switch ex {
	case "binance_futures":
		// 使用配置的环境设置，而不是订单的Testnet字段
		useTestnet := s.cfg.Exchange.Binance.IsTestnet
		c := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

		// 验证订单前提条件（交易对支持、杠杆设置）
		// 注意：保证金模式已在订单创建前设置
		if err := s.validateOrderPrerequisites(c, o); err != nil {
			return false, err.Error()
		}
		// 一键三连：若启用则下进场单后挂 TP/SL
		if o.BracketEnabled {
			return s.executeBracketOrder(c, o)
		}

		// 检查是否为条件订单（TP/SL订单）
		if o.OrderType == "TAKE_PROFIT_MARKET" || o.OrderType == "STOP_MARKET" {
			return s.executeConditionalOrder(c, o)
		}

		// 处理普通订单（非Bracket订单）
		return s.executeRegularOrder(c, o)
	default:
		return false, "unsupported exchange: " + ex
	}
}

func (s *OrderScheduler) execute(o pdb.ScheduledOrder) {
	// 执行策略判断
	shouldContinue, modifiedOrder, reason := s.executeStrategyCheck(o)
	if !shouldContinue {
		s.fail(o.ID, reason)
		return
	}

	// 如果策略修改了订单，使用修改后的订单
	if modifiedOrder != nil {
		o = *modifiedOrder
	}

	// 执行交易所订单
	success, result := s.executeExchangeOrder(o)
	if !success {
		s.fail(o.ID, result)
		return
	}

	// 更新数据库状态
	_ = s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", o.ID).
		Updates(map[string]any{"status": "success", "result": result})
}

// getMarketDataForStrategy 获取策略执行所需的市场数据
func (s *OrderScheduler) getMarketDataForStrategy(symbol string) (StrategyMarketData, error) {
	data := StrategyMarketData{
		Symbol:      symbol,
		MarketCap:   1000000000, // 默认10亿市值
		GainersRank: 50,         // 默认第50名
		HasSpot:     false,      // 默认没有现货
		HasFutures:  false,      // 默认没有合约
	}

	// 解析基础货币（去除USDT等）
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	baseSymbol = strings.TrimSuffix(baseSymbol, "BUSD")
	baseSymbol = strings.TrimSuffix(baseSymbol, "USDC")

	// 从涨幅榜获取最新的排名和市值数据（优化版本）
	gainers, err := s.getGainersFrom24hStats("futures", 50) // 获取前50名
	if err != nil {
		log.Printf("[Strategy] 获取涨幅榜数据失败，使用默认值: %v", err)
	} else {
		// 在涨幅榜中查找当前币种
		for _, gainer := range gainers {
			if gainer.Symbol == symbol {
				data.GainersRank = gainer.Rank

				// 使用交易量作为市值估算（简化计算）
				// 实际市值应该从专门的市场数据API获取
				if gainer.Volume24h > 1000000 { // 交易量大于100万美元
					data.MarketCap = float64(gainer.Volume24h * 10) // 粗略估算市值
				}
				break
			}
		}
	}

	// 使用币安API实时检查是否有现货和合约支持
	// 检查期货合约支持
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	futuresClient := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)
	if supported, err := futuresClient.IsSymbolSupported(symbol); err == nil && supported {
		data.HasFutures = true
	}

	// 检查是否有现货和合约（通过查询binance_24h_stats表）
	// 查询现货数据
	var spotCount int64
	if err := s.db.Model(&pdb.Binance24hStats{}).
		Where("market_type = ? AND symbol = ?", "spot", symbol).
		Count(&spotCount).Error; err == nil && spotCount > 0 {
		data.HasSpot = true
	}

	// 查询合约数据
	var futuresCount int64
	if err := s.db.Model(&pdb.Binance24hStats{}).
		Where("market_type = ? AND symbol = ?", "futures", symbol).
		Count(&futuresCount).Error; err == nil && futuresCount > 0 {
		data.HasFutures = true
	}

	// 市值数据仍然从原有的涨幅榜逻辑获取（如果有的话）
	// 或者可以考虑从其他地方获取市值数据

	// 如果没有从涨幅榜获取到市值，使用硬编码的默认值
	if data.MarketCap == 1000000000 {
		switch baseSymbol {
		case "BTC":
			data.MarketCap = 1000000000000 // 1万亿
		case "ETH":
			data.MarketCap = 300000000000 // 3000亿
		case "BNB":
			data.MarketCap = 80000000000 // 800亿
		case "ADA":
			data.MarketCap = 20000000000 // 200亿
		case "SOL":
			data.MarketCap = 15000000000 // 150亿
		default:
			data.MarketCap = 5000000000 // 50亿
		}
	}

	log.Printf("[scheduler] Market data for %s: spot=%v, futures=%v, marketCap=%.1f亿, rank=%d",
		symbol, data.HasSpot, data.HasFutures, data.MarketCap/100000000, data.GainersRank)

	return data, nil
}

// BinanceSymbolInfo 表示交易对的精度信息
type BinanceSymbolInfo struct {
	Symbol  string                   `json:"symbol"`
	Status  string                   `json:"status"`
	Filters []map[string]interface{} `json:"filters"`
}

// BinanceExchangeInfo 表示交易所信息
type BinanceExchangeInfo struct {
	Symbols []BinanceSymbolInfo `json:"symbols"`
}

// checkMarginSufficiency 检查保证金是否充足
func (s *OrderScheduler) checkMarginSufficiency(symbol string, quantity float64, price float64, leverage int) (sufficient bool, requiredMargin float64, availableMargin float64, reason string) {
	// 计算所需保证金：名义价值 / 杠杆
	notionalValue := quantity * price
	requiredMargin = notionalValue / float64(leverage)

	log.Printf("[scheduler] 保证金检查: %s 数量=%.4f 价格=%.8f 杠杆=%dx 所需保证金=%.4f USDT",
		symbol, quantity, price, leverage, requiredMargin)

	// 获取真实的账户信息
	ctx := context.Background()
	accountInfo, err := s.getAccountInfo(ctx)
	if err != nil {
		log.Printf("[scheduler] 获取账户信息失败，使用保守策略: %v", err)
		// 获取失败时使用保守策略：假设只有很少的可用保证金
		availableMargin = 10.0
		reason = fmt.Sprintf("无法获取账户信息，采用保守策略。所需%.2f USDT，假定可用%.2f USDT。",
			requiredMargin, availableMargin)
		if requiredMargin > availableMargin {
			return false, requiredMargin, availableMargin, reason
		}
		return true, requiredMargin, availableMargin, ""
	}

	log.Printf("[scheduler] 成功获取账户信息: 可用保证金=%s USDT", accountInfo.AvailableBalance)

	// 解析可用保证金
	availableMargin, parseErr := strconv.ParseFloat(accountInfo.AvailableBalance, 64)
	if parseErr != nil {
		log.Printf("[scheduler] 解析可用保证金失败: %v，使用保守策略", parseErr)
		availableMargin = 10.0
	}

	log.Printf("[scheduler] 账户可用保证金: %.4f USDT", availableMargin)

	if requiredMargin > availableMargin {
		reason = fmt.Sprintf("保证金不足: 需要%.2f USDT，账户可用%.2f USDT。建议降低杠杆或减少仓位。",
			requiredMargin, availableMargin)
		return false, requiredMargin, availableMargin, reason
	}

	return true, requiredMargin, availableMargin, ""
}

// getAccountInfo 获取账户信息
func (s *OrderScheduler) getAccountInfo(ctx context.Context) (*bf.AccountInfo, error) {
	// 检查API密钥是否已配置
	if s.cfg.Exchange.Binance.APIKey == "" || s.cfg.Exchange.Binance.APIKey == "your_binance_api_key_here" {
		log.Printf("[scheduler] API密钥未配置，使用模拟账户信息")
		// 返回模拟账户信息
		return &bf.AccountInfo{
			AvailableBalance:   "10000.00", // 模拟10000 USDT可用保证金
			TotalWalletBalance: "10000.00",
			TotalMarginBalance: "10000.00",
		}, nil
	}

	if s.cfg.Exchange.Binance.SecretKey == "" || s.cfg.Exchange.Binance.SecretKey == "your_binance_secret_key_here" {
		log.Printf("[scheduler] API密钥未配置，使用模拟账户信息")
		// 返回模拟账户信息
		return &bf.AccountInfo{
			AvailableBalance:   "10000.00", // 模拟10000 USDT可用保证金
			TotalWalletBalance: "10000.00",
			TotalMarginBalance: "10000.00",
		}, nil
	}

	// 创建币安期货客户端，使用配置的环境设置
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	accountInfo, err := client.GetAccountInfo()
	if err != nil {
		return nil, fmt.Errorf("获取账户信息失败: %w", err)
	}

	return accountInfo, nil
}

// getLotSizeAndMinNotional 获取交易对的LOT_SIZE步长、最小名义价值和最大数量限制
func (s *OrderScheduler) getLotSizeAndMinNotional(symbol string, orderType string) (stepSize, minNotional, maxQty, minQty float64, err error) {
	// 从数据库获取交易对信息
	log.Printf("[scheduler] 从数据库获取 %s 的交易对信息", symbol)
	exchangeInfo, err := pdb.GetExchangeInfo(s.db, strings.ToUpper(symbol))
	if err != nil {
		log.Printf("[scheduler] 数据库查询失败: %v", err)
		return 0, 0, 0, 0, fmt.Errorf("获取exchangeInfo失败: %v", err)
	}

	// 解析过滤器信息
	var filters []map[string]interface{}
	if err := json.Unmarshal([]byte(exchangeInfo.Filters), &filters); err != nil {
		log.Printf("[scheduler] 解析过滤器JSON失败: %v", err)
		return 0, 0, 0, 0, fmt.Errorf("解析过滤器信息失败: %v", err)
	}

	log.Printf("[scheduler] %s 从数据库获取到%d个过滤器", symbol, len(filters))

	// 记录所有过滤器信息用于调试
	log.Printf("[scheduler] %s 过滤器信息 (共%d个):", symbol, len(filters))
	for i, filter := range filters {
		if filterType, ok := filter["filterType"].(string); ok {
			log.Printf("[scheduler]   [%d] %s: %v", i, filterType, filter)
		} else {
			log.Printf("[scheduler]   [%d] 未知过滤器: %v", i, filter)
		}
	}

	// 添加数据验证，确保获取的信息是合理的
	defer func() {
		if err == nil {
			// 验证stepSize不应该太小（避免精度问题）
			if stepSize < 0.0001 && stepSize > 0 {
				log.Printf("[scheduler] 警告: %s stepSize %.8f 过小，可能存在问题", symbol, stepSize)
			}
			// 验证minNotional在合理范围内
			if minNotional > 10000 || (minNotional > 0 && minNotional < 1) {
				log.Printf("[scheduler] 警告: %s minNotional %.2f 不合理，可能存在问题", symbol, minNotional)
			}
			// 检测特定的错误模式：如果获取到可疑的默认值，可能是API数据错误
			if stepSize == 0.001 && minNotional == 100 && maxQty == 1000 {
				log.Printf("[scheduler] 检测到可疑的默认值模式 (%s)，这可能表明API返回了错误数据", symbol)
			}
		}
	}()

	var minQtyLocal float64 = 0
	maxQty = 0 // 初始化返回值
	minQty = 0 // 初始化返回值

	// 根据订单类型选择正确的LOT_SIZE过滤器
	lotSizeFilterType := "LOT_SIZE"
	if strings.ToUpper(orderType) == "MARKET" {
		// 对于bracket订单，允许使用更高的LOT_SIZE限制而不是MARKET_LOT_SIZE
		// 注意：这可能违反交易所限制，请谨慎使用
		lotSizeFilterType = "LOT_SIZE" // 使用限价单的限制 (maxQty=1000)
		// lotSizeFilterType = "MARKET_LOT_SIZE"  // 原始交易所限制 (maxQty=120)
	}

	// 查找所有相关过滤器
	for _, filter := range filters {
		if filterType, ok := filter["filterType"].(string); ok {
			switch filterType {
			case lotSizeFilterType: // LOT_SIZE 或 MARKET_LOT_SIZE
				if stepSizeStr, ok := filter["stepSize"].(string); ok {
					stepSize, err = strconv.ParseFloat(stepSizeStr, 64)
					if err != nil {
						return 0, 0, 0, 0, fmt.Errorf("解析stepSize失败: %v", err)
					}
				}
				if maxQtyStr, ok := filter["maxQty"].(string); ok {
					maxQty, _ = strconv.ParseFloat(maxQtyStr, 64)
				}
				if minQtyStr, ok := filter["minQty"].(string); ok {
					minQtyLocal, _ = strconv.ParseFloat(minQtyStr, 64)
				}
			case "LOT_SIZE": // 如果是MARKET订单，也记录LIMIT订单的限制用于参考
				if lotSizeFilterType == "MARKET_LOT_SIZE" {
					if maxQtyStr, ok := filter["maxQty"].(string); ok {
						if limitMaxQty, _ := strconv.ParseFloat(maxQtyStr, 64); limitMaxQty > 0 && limitMaxQty < maxQty {
							log.Printf("[scheduler] %s LIMIT订单maxQty=%f比MARKET订单maxQty=%f更严格，使用LIMIT限制", symbol, limitMaxQty, maxQty)
							maxQty = limitMaxQty
						}
					}
				}
			case "MIN_NOTIONAL":
				if minNotionalStr, ok := filter["notional"].(string); ok {
					minNotional, err = strconv.ParseFloat(minNotionalStr, 64)
					if err != nil {
						log.Printf("[scheduler] 解析minNotional失败: %v，使用默认值5.0", err)
						minNotional = 5.0 // 默认最小名义价值
					}
				}
			case "MAX_POSITION":
				if maxQtyStr, ok := filter["maxQty"].(string); ok {
					if max, _ := strconv.ParseFloat(maxQtyStr, 64); max > 0 && (maxQty == 0 || max < maxQty) {
						maxQty = max
					}
				}
			}
		}
	}

	// 如果没有找到LOT_SIZE过滤器，使用默认值
	if stepSize == 0 {
		log.Printf("[scheduler] 未找到 %s 的LOT_SIZE过滤器，使用默认步长1", symbol)
		stepSize = 1 // 使用更保守的默认值
	}

	// 如果没有找到MIN_NOTIONAL，使用默认值
	if minNotional == 0 {
		minNotional = 5.0 // 默认5 USDT最小名义价值
	}

	// 设置minQty的返回值
	if minQtyLocal > 0 {
		minQty = minQtyLocal
	} else {
		minQty = 1 // 默认最小数量，与stepSize保持一致
	}

	// 智能验证和修正过滤器数据
	stepSize, minNotional, maxQty, minQty = s.validateAndCorrectFilters(symbol, stepSize, minNotional, maxQty, minQty)

	log.Printf("[scheduler] %s 最终精度信息: stepSize=%.6f, minNotional=%.2f, maxQty=%.2f, minQty=%.6f",
		symbol, stepSize, minNotional, maxQty, minQty)

	return stepSize, minNotional, maxQty, minQty, nil
}

// getPriceFilterInfo 获取交易对的价格过滤器信息
func (s *OrderScheduler) getPriceFilterInfo(symbol string) (tickSize, minPrice, maxPrice float64, err error) {
	// 从数据库获取交易对信息
	log.Printf("[scheduler] 从数据库获取 %s 的价格过滤器信息", symbol)
	exchangeInfo, err := pdb.GetExchangeInfo(s.db, strings.ToUpper(symbol))
	if err != nil {
		log.Printf("[scheduler] 数据库查询失败: %v", err)
		return 0, 0, 0, fmt.Errorf("获取exchangeInfo失败: %v", err)
	}

	// 解析过滤器信息
	var filters []map[string]interface{}
	if err := json.Unmarshal([]byte(exchangeInfo.Filters), &filters); err != nil {
		log.Printf("[scheduler] 解析过滤器JSON失败: %v", err)
		return 0, 0, 0, fmt.Errorf("解析过滤器信息失败: %v", err)
	}

	log.Printf("[scheduler] %s 从数据库获取到%d个过滤器", symbol, len(filters))

	// 查找PRICE_FILTER过滤器
	for i, filter := range filters {
		filterType, hasType := filter["filterType"]
		if !hasType {
			log.Printf("[scheduler] %s filter[%d] 缺少filterType字段: %+v", symbol, i, filter)
			continue
		}

		ft, ok := filterType.(string)
		if !ok {
			log.Printf("[scheduler] %s filter[%d] filterType不是字符串: %T = %v", symbol, i, filterType, filterType)
			continue
		}

		if ft == "PRICE_FILTER" {
			log.Printf("[scheduler] %s 找到PRICE_FILTER: %+v", symbol, filter)

			if tickSizeStr, ok := filter["tickSize"].(string); ok {
				tickSize, err = strconv.ParseFloat(tickSizeStr, 64)
				if err != nil {
					return 0, 0, 0, fmt.Errorf("解析tickSize失败: %v", err)
				}
			} else {
				log.Printf("[scheduler] %s PRICE_FILTER缺少tickSize字段", symbol)
			}

			if minPriceStr, ok := filter["minPrice"].(string); ok {
				minPrice, _ = strconv.ParseFloat(minPriceStr, 64)
				log.Printf("[scheduler] %s 解析minPrice: %s -> %.8f", symbol, minPriceStr, minPrice)
			} else {
				log.Printf("[scheduler] %s PRICE_FILTER缺少minPrice字段", symbol)
			}

			if maxPriceStr, ok := filter["maxPrice"].(string); ok {
				maxPrice, _ = strconv.ParseFloat(maxPriceStr, 64)
				log.Printf("[scheduler] %s 解析maxPrice: %s -> %.0f", symbol, maxPriceStr, maxPrice)
			} else {
				log.Printf("[scheduler] %s PRICE_FILTER缺少maxPrice字段", symbol)
			}
			break
		} else {
			log.Printf("[scheduler] %s filter[%d] filterType=%s，跳过", symbol, i, ft)
		}
	}

	// 如果没有找到PRICE_FILTER过滤器，使用默认值
	if tickSize == 0 {
		log.Printf("[scheduler] 未找到 %s 的PRICE_FILTER过滤器，使用更严格的默认值", symbol)
		tickSize = 0.00000100 // 使用更严格的精度避免"Precision is over the maximum"错误
		minPrice = 0.00000001
		maxPrice = 999999999
	}

	// 智能检测和纠正API数据异常
	originalTickSize := tickSize
	originalMinPrice := minPrice

	// 检测tickSize异常
	if tickSize <= 0 || tickSize >= 0.01 || tickSize > 1 {
		log.Printf("[scheduler] 检测到 %s tickSize异常: %.8f，使用自适应精度", symbol, tickSize)
		// 获取当前价格来确定合适的精度
		currentPrice, err := s.getCurrentPriceFromFutures(context.Background(), symbol)
		if err != nil {
			currentPrice = 1.0 // 默认价格用于确定精度
		}

		// 基于价格范围选择合适的tickSize
		if currentPrice < 0.1 {
			tickSize = 0.000001
		} else if currentPrice < 1.0 {
			tickSize = 0.00001
		} else if currentPrice < 10.0 {
			tickSize = 0.0001
		} else if currentPrice < 100.0 {
			tickSize = 0.001
		} else {
			tickSize = 0.01
		}
		log.Printf("[scheduler] %s 自适应tickSize: %.8f -> %.8f (价格: %.4f)", symbol, originalTickSize, tickSize, currentPrice)
	}

	// 检测minPrice异常
	if minPrice <= 0 || minPrice >= 500 {
		log.Printf("[scheduler] 检测到 %s minPrice异常: %.8f，使用合理默认值", symbol, minPrice)
		minPrice = 0.00000001
		log.Printf("[scheduler] %s minPrice调整: %.8f -> %.8f", symbol, originalMinPrice, minPrice)
	}

	// 检测maxPrice异常（通常不需要调整，除非明显不合理）
	if maxPrice > 0 && maxPrice < 1000000 {
		log.Printf("[scheduler] %s maxPrice %.0f 可能偏低，但保持原值", symbol, maxPrice)
	}

	return tickSize, minPrice, maxPrice, nil
}

// getPriceTickSize 获取交易对的价格TICK_SIZE（向后兼容）
func (s *OrderScheduler) getPriceTickSize(symbol string) (float64, error) {
	tickSize, _, _, err := s.getPriceFilterInfo(symbol)
	return tickSize, err
}

// hasValidExchangeInfo 检查数据库中是否有有效的交易所信息
func (s *OrderScheduler) hasValidExchangeInfo(symbol string) bool {
	// 从数据库获取交易对信息
	exchangeInfo, err := pdb.GetExchangeInfo(s.db, strings.ToUpper(symbol))
	if err != nil {
		log.Printf("[scheduler] 检查 %s 交易所信息失败: %v", symbol, err)
		return false
	}

	// 检查过滤器信息是否存在且不为空
	if exchangeInfo.Filters == "" || len(exchangeInfo.Filters) < 10 {
		log.Printf("[scheduler] %s 的过滤器信息为空或过短", symbol)
		return false
	}

	// 尝试解析过滤器信息，验证格式是否正确
	var filters []map[string]interface{}
	if err := json.Unmarshal([]byte(exchangeInfo.Filters), &filters); err != nil {
		log.Printf("[scheduler] %s 的过滤器信息JSON格式错误: %v", symbol, err)
		return false
	}

	// 检查是否包含必要的过滤器
	hasPriceFilter := false
	hasLotSize := false
	for _, filter := range filters {
		if filterType, ok := filter["filterType"].(string); ok {
			switch filterType {
			case "PRICE_FILTER":
				hasPriceFilter = true
			case "LOT_SIZE":
				hasLotSize = true
			}
		}
	}

	if !hasPriceFilter || !hasLotSize {
		log.Printf("[scheduler] %s 缺少必要的过滤器 (PRICE_FILTER: %v, LOT_SIZE: %v)",
			symbol, hasPriceFilter, hasLotSize)
		return false
	}

	return true
}

// getCurrentPrice 获取交易对的当前价格（用于名义价值计算）
func (s *OrderScheduler) getCurrentPrice(ctx context.Context, symbol string, kind string) (float64, error) {
	// 1. 优先从API获取最新价格
	var price float64
	var err error

	if kind == "futures" {
		log.Printf("[scheduler] 从API获取 %s 期货价格", symbol)
		price, err = s.getCurrentPriceFromFutures(ctx, symbol)
		if err == nil {
			return price, nil
		}
		log.Printf("[scheduler] API获取失败，使用数据库缓存作为后备: %v", err)
	} else if kind == "spot" {
		// 添加批量操作延迟，避免API限流
		if ctx.Value("batch_operation") != nil {
			time.Sleep(50 * time.Millisecond)
		}

		// 先尝试市场快照数据
		now := time.Now().UTC()
		startTime := now.Add(-2 * time.Hour)
		snaps, tops, err := pdb.ListBinanceMarket(s.db, kind, startTime, now)
		if err == nil && len(snaps) > 0 {
			// Get latest snapshot
			latestSnap := snaps[len(snaps)-1]
			if items, ok := tops[latestSnap.ID]; ok {
				for _, item := range items {
					if item.Symbol == symbol {
						price, err := strconv.ParseFloat(item.LastPrice, 64)
						if err == nil {
							return price, nil
						}
					}
				}
			}
		}

		// 如果市场快照没有数据，从Binance API获取
		log.Printf("[scheduler] 从Binance API获取 %s 现货价格", symbol)
		price, err = s.getCurrentPriceFromSpot(ctx, symbol)
		if err == nil {
			return price, nil
		}
		log.Printf("[scheduler] API获取失败，使用数据库缓存作为后备: %v", err)
	} else {
		return 0, fmt.Errorf("不支持的价格类型: %s", kind)
	}

	// 2. API获取失败时，使用数据库缓存作为后备
	if s.db != nil {
		cache, err := pdb.GetPriceCache(s.db, symbol, kind)
		if err == nil && cache != nil {
			// 使用较宽松的缓存时间（5分钟），因为这是后备选项
			if time.Since(cache.LastUpdated.UTC()) <= 5*time.Minute {
				if cachePrice, parseErr := strconv.ParseFloat(cache.Price, 64); parseErr == nil {
					log.Printf("[scheduler] 从数据库缓存获取 %s %s价格作为后备: %.6f", symbol, kind, cachePrice)
					return cachePrice, nil
				}
			} else {
				log.Printf("[scheduler] 数据库缓存过期，跳过使用")
			}
		}
	}

	// 所有方法都失败
	return 0, fmt.Errorf("无法获取 %s 的价格，所有方法都失败: %w", symbol, err)
}

// getCurrentPriceFromFutures 获取期货价格（实时调用，无缓存）
func (s *OrderScheduler) getCurrentPriceFromFutures(ctx context.Context, symbol string) (float64, error) {
	// 如果没有提供context，使用默认的2秒超时（更短以提高实时性）
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
	}

	// 使用标记价格而不是最新成交价格来计算未实现盈亏
	// 标记价格更稳定，更适合盈亏计算
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", strings.ToUpper(symbol))

	type PriceResponse struct {
		Symbol    string `json:"symbol"`
		MarkPrice string `json:"markPrice"`
	}

	var resp PriceResponse
	if err := netutil.GetJSON(ctx, url, &resp); err != nil {
		return 0, fmt.Errorf("获取价格失败: %v", err)
	}

	price, err := strconv.ParseFloat(resp.MarkPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("解析价格失败: %v", err)
	}

	return price, nil
}

// getCurrentPriceFromSpot 获取现货价格（实时调用，无缓存）
func (s *OrderScheduler) getCurrentPriceFromSpot(ctx context.Context, symbol string) (float64, error) {
	// 如果没有提供context，使用默认的2秒超时
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
	}

	// 调用Binance现货价格API
	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", strings.ToUpper(symbol))

	type PriceResponse struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	var resp PriceResponse
	if err := netutil.GetJSON(ctx, url, &resp); err != nil {
		return 0, fmt.Errorf("获取现货价格失败: %v", err)
	}

	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("解析现货价格失败: %v", err)
	}

	return price, nil
}

func (s *OrderScheduler) fail(id uint, reason string) {
	log.Println("[scheduler] order fail:", reason)
	_ = s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", id).
		Updates(map[string]any{"status": "failed", "result": reason})
}

// 从 binance_24h_stats 直接查询涨幅榜数据（优化版本）
func (s *OrderScheduler) getGainersFrom24hStats(marketType string, limit int) ([]pdb.RealtimeGainersItem, error) {
	var results []struct {
		Symbol             string
		PriceChangePercent float64
		Volume             float64
		LastPrice          float64
		Ranking            int
	}

	query := `
		SELECT
			symbol,
			price_change_percent,
			volume,
			last_price,
			ROW_NUMBER() OVER (ORDER BY price_change_percent DESC, volume DESC) as ranking
		FROM binance_24h_stats
		WHERE market_type = ? AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT ?
	`

	err := s.db.Raw(query, marketType, limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询涨幅榜数据失败: %w", err)
	}

	// 转换为 RealtimeGainersItem 格式以保持兼容性
	var gainers []pdb.RealtimeGainersItem
	for _, result := range results {
		gainers = append(gainers, pdb.RealtimeGainersItem{
			Symbol:         result.Symbol,
			Rank:           result.Ranking,
			CurrentPrice:   result.LastPrice,
			PriceChange24h: result.PriceChangePercent,
			Volume24h:      result.Volume,
			DataSource:     "24h_stats",
			CreatedAt:      time.Now(), // 使用当前时间作为创建时间
		})
	}

	return gainers, nil
}

// 智能候选选择器逻辑（从strategy_candidate_selector.go迁移）
func (s *OrderScheduler) selectBestSelectorForStrategy(strategy *pdb.TradingStrategy) string {
	conditions := strategy.Conditions

	// 传统策略：涨幅榜选择器（因为策略基于排名）
	if conditions.ShortOnGainers || conditions.LongOnSmallGainers {
		return "volume_based"
	}

	// 均线策略：交易量选择器（需要活跃市场）
	if conditions.MovingAverageEnabled {
		return "volume_based"
	}

	// 套利策略：流动性选择器
	if conditions.CrossExchangeArbEnabled || conditions.SpotFutureArbEnabled ||
		conditions.TriangleArbEnabled || conditions.StatArbEnabled {
		return "volume_based"
	}

	// 均值回归策略：智能选择器
	if conditions.MeanReversionEnabled {
		return "volume_based"
	}

	// 默认使用交易量选择器
	return "volume_based"
}

// 按交易量选择候选币种（从VolumeBasedSelector迁移）
func (s *OrderScheduler) selectCandidatesByVolume(ctx context.Context, strategy *pdb.TradingStrategy, maxCount int) ([]string, error) {
	log.Printf("[VolumeBasedSelector] 基于交易量选择前%d个候选币种", maxCount)

	// 从数据库获取交易量最大的币种
	gdb := s.server.db.DB()

	var volumeStats []struct {
		Symbol      string
		Volume      float64
		QuoteVolume float64
		PriceChange float64
		Count       int64
	}

	// 查询最近24小时的交易统计，从binance_24h_stats表获取数据
	err := gdb.Table("binance_24h_stats").
		Select("symbol, AVG(volume) as volume, AVG(quote_volume) as quote_volume, AVG(price_change_percent) as price_change, COUNT(*) as count").
		Where("market_type = ? AND created_at >= ?", "spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Having("COUNT(*) >= 1"). // 至少有1条记录
		Order("AVG(quote_volume) DESC"). // 按报价交易量排序
		Limit(maxCount * 2). // 多取一些备用
		Scan(&volumeStats).Error

	if err != nil {
		log.Printf("[VolumeBasedSelector] 查询交易量数据失败: %v，使用涨幅榜降级", err)
		return s.fallbackToGainersForScheduler(maxCount)
	}

	// 筛选出有足够交易量的币种
	var candidates []string
	for _, stat := range volumeStats {
		// 对于均值回归策略，降低交易量门槛到10万美元
		minVolume := 100000.0 // 10万美元作为最低门槛
		if stat.QuoteVolume > minVolume {
			candidates = append(candidates, stat.Symbol)
			if len(candidates) >= maxCount*2 { // 多取一些用于过滤
				break
			}
		}
	}

	if len(candidates) == 0 {
		log.Printf("[VolumeBasedSelector] 未找到足够交易量的币种(最低%.0f)，使用优化降级", 100000.0)
		return s.fallbackToVolumeOptimizedForScheduler(maxCount)
	}

	log.Printf("[VolumeBasedSelector] 初步筛选出%d个高交易量候选币种", len(candidates))

	// 应用过滤器
	originalCount := len(candidates)

	// 1. 过滤稳定币 (如果策略需要)
	if strategy.Conditions.MovingAverageEnabled {
		// 对于均线策略，默认过滤稳定币
		candidates = s.filterStableCoins(candidates)
		log.Printf("[VolumeBasedSelector] 过滤稳定币: %d → %d", originalCount, len(candidates))
	}

	// 2. 过滤低波动资产 (如果需要)
	// 这里可以根据配置添加波动率过滤
	// candidates = FilterByVolatility(candidates, 0.1) // 最小0.1%波动率

	// 确保有足够的候选币种
	if len(candidates) < maxCount {
		log.Printf("[VolumeBasedSelector] 过滤后候选不足%d个，使用涨幅榜补充", maxCount)
		// 这里可以补充其他候选币种
	}

	// 限制数量
	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}

	showCount := 5
	if len(candidates) < 5 {
		showCount = len(candidates)
	}
	log.Printf("[VolumeBasedSelector] 最终选择了%d个候选币种: %v", len(candidates), candidates[:showCount])
	return candidates, nil
}

// 过滤稳定币
func (s *OrderScheduler) filterStableCoins(symbols []string) []string {
	stableCoins := []string{"USDT", "USDC", "BUSD", "DAI", "TUSD", "USDP", "FRAX", "LUSD", "USDN"}
	var filtered []string

	for _, symbol := range symbols {
		isStable := false
		for _, stable := range stableCoins {
			if strings.Contains(symbol, stable) {
				isStable = true
				break
			}
		}
		if !isStable {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// 降级到涨幅榜（优化版本：直接从 binance_24h_stats 查询）
func (s *OrderScheduler) fallbackToGainersForScheduler(maxCount int) ([]string, error) {
	// 直接从 binance_24h_stats 查询涨幅最大的币种
	var results []struct {
		Symbol string
	}

	query := `
		SELECT symbol
		FROM binance_24h_stats
		WHERE market_type = 'futures'
			AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			AND volume > 1000000  -- 过滤低成交量的币种
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT ?
	`

	err := s.server.db.DB().Raw(query, maxCount).Scan(&results).Error
	if err != nil {
		log.Printf("[VolumeBasedSelector] 从 binance_24h_stats 查询涨幅榜失败: %v", err)
		// 最后的降级：硬编码主要币种
		return []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}, nil
	}

	if len(results) == 0 {
		log.Printf("[VolumeBasedSelector] 未找到有效的涨幅榜数据")
		return []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}, nil
	}

	var candidates []string
	for _, result := range results {
		candidates = append(candidates, result.Symbol)
	}

	log.Printf("[VolumeBasedSelector] 从 binance_24h_stats 选择了 %d 个涨幅榜候选币种", len(candidates))
	return candidates, nil
}

// 优化的交易量降级策略
func (s *OrderScheduler) fallbackToVolumeOptimizedForScheduler(maxCount int) ([]string, error) {
	log.Printf("[VolumeBasedSelector] 执行优化降级策略")

	// 策略1：查询最近1小时内的所有spot市场数据，不限制交易量
	var results1 []struct {
		Symbol      string
		QuoteVolume float64
	}

	query1 := `
		SELECT symbol, AVG(quote_volume) as quote_volume
		FROM binance_24h_stats
		WHERE market_type = 'spot'
			AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
		GROUP BY symbol
		ORDER BY AVG(quote_volume) DESC
		LIMIT ?
	`

	err1 := s.server.db.DB().Raw(query1, maxCount*2).Scan(&results1).Error
	if err1 == nil && len(results1) > 0 {
		var candidates []string
		for _, result := range results1 {
			candidates = append(candidates, result.Symbol)
			if len(candidates) >= maxCount {
				break
			}
		}
		log.Printf("[VolumeBasedSelector] 优化降级1成功: 找到%d个币种", len(candidates))
		return candidates, nil
	}

	// 策略2：查询所有市场类型的最近数据
	var results2 []struct {
		Symbol string
	}

	query2 := `
		SELECT DISTINCT symbol
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)
		ORDER BY created_at DESC
		LIMIT ?
	`

	err2 := s.server.db.DB().Raw(query2, maxCount*3).Scan(&results2).Error
	if err2 == nil && len(results2) > 0 {
		var candidates []string
		for _, result := range results2 {
			candidates = append(candidates, result.Symbol)
			if len(candidates) >= maxCount {
				break
			}
		}
		log.Printf("[VolumeBasedSelector] 优化降级2成功: 找到%d个币种", len(candidates))
		return candidates, nil
	}

	// 策略3：硬编码适合均值回归策略的币种列表
	meanReversionCandidates := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
		"DOTUSDT", "AVAXUSDT", "LINKUSDT", "LTCUSDT", "XRPUSDT",
		"DOGEUSDT", "MATICUSDT", "SHIBUSDT", "UNIUSDT", "ICPUSDT",
		"FILUSDT", "ETCUSDT", "VETUSDT", "TRXUSDT", "THETAUSDT",
		"FTTUSDT", "ALGOUSDT", "ATOMUSDT", "CAKEUSDT", "SUSHIUSDT",
		"COMPUSDT", "MKRUSDT", "AAVEUSDT", "CRVUSDT", "YFIUSDT",
		"BALUSDT", "IMXUSDT", "GRTUSDT", "ACHUSDT", "ROSEUSDT",
		"USTCUSDT", "DATAUSDT", "BIOUSDT", "OMUSDT", "ORDIUSDT",
		"JUPUSDT", "0GUSDT", "PEOPLEUSDT", "WBTCUSDT",
	}

	// 限制数量
	if len(meanReversionCandidates) > maxCount {
		meanReversionCandidates = meanReversionCandidates[:maxCount]
	}

	log.Printf("[VolumeBasedSelector] 优化降级3: 使用预定义币种列表 (%d个)", len(meanReversionCandidates))
	return meanReversionCandidates, nil
}
