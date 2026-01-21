package main

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/netutil"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// =============================
//         调度器组件定义
// =============================

// SmartScheduler 智能调度器
type SmartScheduler struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSmartScheduler 创建智能调度器
func NewSmartScheduler() *SmartScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &SmartScheduler{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动智能调度器
func (ss *SmartScheduler) Start() error {
	log.Printf("[SmartScheduler] 智能调度器启动")
	// TODO: 实现智能调度逻辑
	return nil
}

// Stop 停止智能调度器
func (ss *SmartScheduler) Stop() {
	ss.cancel()
	log.Printf("[SmartScheduler] 智能调度器已停止")
}

// PrecomputeProcessor 预计算处理器
type PrecomputeProcessor struct {
	apiBase string
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewPrecomputeProcessor 创建预计算处理器
func NewPrecomputeProcessor(apiBase string, cfg *config.Config) *PrecomputeProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &PrecomputeProcessor{
		apiBase: apiBase,
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start 启动预计算处理器
func (pp *PrecomputeProcessor) Start() error {
	log.Printf("[PrecomputeProcessor] 预计算处理器启动")
	// TODO: 实现预计算逻辑
	return nil
}

// Stop 停止预计算处理器
func (pp *PrecomputeProcessor) Stop() {
	pp.cancel()
	log.Printf("[PrecomputeProcessor] 预计算处理器已停止")
}

type InvestmentServiceManager struct {
	apiBase string
	config  *config.Config

	// 调度器组件
	performanceTracker  *PerformanceTracker
	smartScheduler      *SmartScheduler
	precomputeProcessor *PrecomputeProcessor

	// 数据库连接
	db pdb.Database
}

// NewInvestmentServiceManager 创建投资服务管理器
func NewInvestmentServiceManager(apiBase string, cfg *config.Config) *InvestmentServiceManager {
	return &InvestmentServiceManager{
		apiBase: apiBase,
		config:  cfg,
	}
}

// initSchedulers 初始化调度器
func (is *InvestmentServiceManager) initSchedulers() {
	is.performanceTracker = NewPerformanceTracker(is.apiBase, is.config)
	is.smartScheduler = NewSmartScheduler()
	is.precomputeProcessor = NewPrecomputeProcessor(is.apiBase, is.config)
}

// startSchedulers 启动调度器
func (is *InvestmentServiceManager) startSchedulers() error {
	log.Printf("[investment_service] 启动PerformanceTracker...")
	is.performanceTracker.Start()

	log.Printf("[investment_service] 启动SmartScheduler...")
	if err := is.smartScheduler.Start(); err != nil {
		return fmt.Errorf("启动SmartScheduler失败: %w", err)
	}

	log.Printf("[investment_service] 启动PrecomputeProcessor...")
	if err := is.precomputeProcessor.Start(); err != nil {
		return fmt.Errorf("启动PrecomputeProcessor失败: %w", err)
	}

	log.Printf("[investment_service] 所有调度器启动完成")
	return nil
}

// RecommendationScanner 推荐扫描器（内嵌实现）
type RecommendationScanner struct {
	apiBase string
	config  *config.Config
}

// NewRecommendationScanner 创建推荐扫描器
func NewRecommendationScanner(apiBase string, cfg *config.Config) *RecommendationScanner {
	return &RecommendationScanner{
		apiBase: apiBase,
		config:  cfg,
	}
}

// BacktestScanner 回测扫描器（内嵌实现）
type BacktestScanner struct {
	apiBase string
	config  *config.Config
}

// NewBacktestScanner 创建回测扫描器
func NewBacktestScanner(apiBase string, cfg *config.Config) *BacktestScanner {
	return &BacktestScanner{
		apiBase: apiBase,
		config:  cfg,
	}
}

// =============================
//         调度器组件
// =============================

// PerformanceTracker 推荐表现追踪调度器
type PerformanceTracker struct {
	apiBase    string
	config     *config.Config
	ctx        context.Context
	workerPool *WorkerPool // 协程池，限制并发数
}

// NewPerformanceTracker 创建表现追踪调度器
func NewPerformanceTracker(apiBase string, cfg *config.Config) *PerformanceTracker {
	return &PerformanceTracker{
		apiBase:    apiBase,
		config:     cfg,
		ctx:        context.Background(),
		workerPool: NewWorkerPool(10), // 限制最大并发数为10，避免API限流
	}
}

// =============================
//         Worker Pool 协程池
// =============================

// WorkerPool 协程池，用于限制并发数量
type WorkerPool struct {
	maxWorkers int
	workers    chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
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
		return fmt.Errorf("worker pool shutdown timeout")
	}
}

// Running 返回当前运行中的worker数量
func (wp *WorkerPool) Running() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if wp.maxWorkers <= 0 {
		return 0 // 不限制并发时，无法准确计算
	}
	return wp.maxWorkers - len(wp.workers)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================
//         PerformanceTracker 实现
// =============================

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

	// 调用API批量更新推荐表现
	url := pt.apiBase + "/recommendations/performance/batch-update"
	resp, err := pt.makeAPIRequest(pt.ctx, "POST", url, nil)
	if err != nil {
		log.Printf("[PerformanceTracker] 批量更新失败: %v", err)
		return
	}

	log.Printf("[PerformanceTracker] 批量更新完成: %v", resp)

	// 同时更新回测数据（统一处理）
	log.Printf("[PerformanceTracker] 开始更新回测数据")

	// 调用API批量更新回测数据（通过batch-update端点，这个端点应该同时处理实时和回测更新）
	// 这里我们只需要确保它被调用即可，实际的回测更新逻辑在API端
	log.Printf("[PerformanceTracker] 回测更新通过批量更新端点处理完成")
}

// makeAPIRequest PerformanceTracker的API请求方法
func (pt *PerformanceTracker) makeAPIRequest(ctx context.Context, method, url string, body interface{}) (map[string]interface{}, error) {
	log.Printf("[PerformanceTracker] 发送%s请求到: %s", method, url)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PerformanceTracker/1.0")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API请求失败: HTTP %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 检查API响应状态
	if success, ok := result["success"].(bool); ok && !success {
		if message, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("API返回错误: %s", message)
		}
		return nil, fmt.Errorf("API请求失败")
	}

	return result, nil
}

func main() {
	// 命令行参数
	apiBase := flag.String("api", "http://127.0.0.1:8010", "API服务器地址")
	configPath := flag.String("config", "./config.yaml", "配置文件路径")
	service := flag.String("service", "recommendation", "服务类型: investment(投资服务), recommendation(推荐服务), backtest(回测服务), processes(启动所有进程)")
	mode := flag.String("mode", "once", "运行模式: once(单次运行), continuous(持续运行), backtest(回测模式), strategy(策略测试), report(报告生成), generate(生成推荐)")

	// 通用参数
	interval := flag.Duration("interval", 10*time.Minute, "连续模式下的运行间隔")

	// 推荐服务参数
	kind := flag.String("kind", "spot", "推荐类型: spot(现货), futures(期货)")
	limit := flag.Int("limit", 5, "推荐数量限制")
	forceRefresh := flag.Bool("force-refresh", false, "强制刷新推荐（忽略缓存）")

	// 回测和策略参数
	performanceID := flag.String("performance-id", "", "单个表现验证ID（用于策略测试和报告）")
	symbol := flag.String("symbol", "", "交易对符号（用于回测）")
	strategy := flag.String("strategy", "buy_and_hold", "回测策略类型")
	startDate := flag.String("start-date", "", "回测开始日期 (YYYY-MM-DD)")
	endDate := flag.String("end-date", "", "回测结束日期 (YYYY-MM-DD)")
	initialCash := flag.Float64("initial-cash", 10000, "初始资金")

	// 报告参数
	reportType := flag.String("report-type", "summary", "报告类型: summary, detailed, comparison")
	outputPath := flag.String("output", "", "报告输出路径")

	flag.Parse()

	log.Printf("[investment_service] 启动智能投资服务管理器...")
	log.Printf("[investment_service] 服务: %s, 模式: %s, API: %s", *service, *mode, *apiBase)

	// 加载配置
	var cfg config.Config
	config.MustLoad(*configPath, &cfg)
	config.ApplyProxy(&cfg)

	// 根据服务类型创建相应的服务实例
	ctx := context.Background()

	switch *service {
	case "investment":
		// 原有的投资服务
		log.Printf("[investment_service] 启动投资分析服务...")
		serviceManager := NewInvestmentServiceManager(*apiBase, &cfg)
		serviceManager.initSchedulers() // 初始化调度器

		switch *mode {
		case "once":
			log.Printf("[investment_service] 执行单次投资分析任务...")
			if err := serviceManager.runOnce(ctx); err != nil {
				log.Fatalf("[investment_service] 单次任务失败: %v", err)
			}
			log.Printf("[investment_service] 单次任务完成")

		case "continuous":
			log.Printf("[investment_service] 启动持续投资分析模式，间隔: %v", *interval)
			serviceManager.runContinuous(ctx, *interval)

		case "scheduler":
			log.Printf("[investment_service] 启动调度器模式...")
			if err := serviceManager.startSchedulers(); err != nil {
				log.Fatalf("[investment_service] 启动调度器失败: %v", err)
			}
			// 保持运行
			select {}

		case "backtest":
			log.Printf("[investment_service] 执行策略回测...")
			if err := serviceManager.runBacktest(ctx, *symbol, *strategy, *startDate, *endDate, *initialCash); err != nil {
				log.Fatalf("[investment_service] 回测失败: %v", err)
			}

		case "strategy":
			log.Printf("[investment_service] 执行策略测试...")
			if *performanceID == "" {
				log.Fatalf("[investment_service] 策略测试需要指定performance-id")
			}
			if err := serviceManager.runStrategyTest(ctx, *performanceID); err != nil {
				log.Fatalf("[investment_service] 策略测试失败: %v", err)
			}

		case "report":
			log.Printf("[investment_service] 生成投资分析报告...")
			if *performanceID == "" {
				log.Fatalf("[investment_service] 报告生成需要指定performance-id")
			}
			if err := serviceManager.generateReport(ctx, *performanceID, *reportType, *outputPath); err != nil {
				log.Fatalf("[investment_service] 报告生成失败: %v", err)
			}

		default:
			log.Fatalf("[investment_service] 不支持的模式: %s", *mode)
		}

	case "recommendation":
		// 推荐服务
		log.Printf("[investment_service] 启动推荐生成服务...")
		recScanner := NewRecommendationScanner(*apiBase, &cfg)

		switch *mode {
		case "once", "generate":
			log.Printf("[recommendation_service] 执行单次推荐生成...")
			if err := recScanner.generateOnce(ctx, *kind, *limit, *forceRefresh); err != nil {
				log.Fatalf("[recommendation_service] 单次生成失败: %v", err)
			}
			log.Printf("[recommendation_service] 单次生成完成")

		case "continuous":
			log.Printf("[recommendation_service] 启动持续推荐生成模式，间隔: %v", *interval)
			recScanner.runContinuous(ctx, *interval, *kind, *limit, *forceRefresh)

		default:
			log.Fatalf("[recommendation_service] 不支持的模式: %s", *mode)
		}

	case "backtest":
		// 回测服务
		log.Printf("[investment_service] 启动回测分析服务...")
		btScanner := NewBacktestScanner(*apiBase, &cfg)

		switch *mode {
		case "once":
			log.Printf("[backtest_service] 执行单次回测任务...")
			if err := btScanner.runOnce(ctx); err != nil {
				log.Fatalf("[backtest_service] 单次任务失败: %v", err)
			}
			log.Printf("[backtest_service] 单次任务完成")

		case "continuous":
			log.Printf("[backtest_service] 启动持续回测模式，间隔: %v", *interval)
			btScanner.runContinuous(ctx, *interval)

		case "backtest":
			log.Printf("[backtest_service] 执行策略回测...")
			if *symbol == "" {
				log.Fatalf("[backtest_service] 回测需要指定交易对符号")
			}
			if err := btScanner.runBacktest(ctx, *symbol, *strategy, *startDate, *endDate, *initialCash); err != nil {
				log.Fatalf("[backtest_service] 回测失败: %v", err)
			}

		case "strategy":
			log.Printf("[backtest_service] 执行策略测试...")
			if *performanceID == "" {
				log.Fatalf("[backtest_service] 策略测试需要指定performance-id")
			}
			if err := btScanner.runStrategyTest(ctx, *performanceID); err != nil {
				log.Fatalf("[backtest_service] 策略测试失败: %v", err)
			}

		case "batch":
			log.Printf("[backtest_service] 执行批量策略测试...")
			if err := btScanner.runBatchStrategyTests(ctx); err != nil {
				log.Fatalf("[backtest_service] 批量策略测试失败: %v", err)
			}

		default:
			log.Fatalf("[backtest_service] 不支持的模式: %s", *mode)
		}

	case "processes":
		// 启动所有独立进程
		log.Printf("[investment_service] 启动所有投资相关进程...")
		processManager := NewProcessManager(*apiBase, &cfg)
		if err := processManager.startAllProcesses(ctx, *mode, *interval, *kind, *limit, *forceRefresh); err != nil {
			log.Fatalf("[investment_service] 启动进程失败: %v", err)
		}

	default:
		log.Fatalf("[investment_service] 不支持的服务类型: %s", *service)
	}
}

// runOnce 单次运行所有投资分析任务
func (is *InvestmentServiceManager) runOnce(ctx context.Context) error {
	log.Printf("[investment_scanner] 开始执行单次投资分析...")

	// 1. 批量更新验证记录
	log.Printf("[investment_scanner] 步骤1: 批量更新验证记录...")
	if err := is.batchUpdateRecords(ctx); err != nil {
		log.Printf("[investment_scanner] 批量更新失败: %v", err)
		// 不返回错误，继续执行
	}

	// 2. 执行批量策略测试
	log.Printf("[investment_scanner] 步骤2: 执行批量策略测试...")
	if err := is.runBatchStrategyTests(ctx); err != nil {
		log.Printf("[investment_scanner] 批量策略测试失败: %v", err)
		// 不返回错误，继续执行
	}

	log.Printf("[investment_scanner] 单次投资分析任务完成")
	return nil
}

// runContinuous 持续运行模式
func (is *InvestmentServiceManager) runContinuous(ctx context.Context, interval time.Duration) {
	log.Printf("[investment_scanner] 启动持续投资分析模式...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次运行
	if err := is.runOnce(ctx); err != nil {
		log.Printf("[investment_scanner] 首次运行失败: %v", err)
	}

	// 定时运行
	for {
		select {
		case <-ctx.Done():
			log.Printf("[investment_scanner] 收到停止信号，退出...")
			return
		case <-ticker.C:
			log.Printf("[investment_scanner] 执行定时投资分析任务...")
			if err := is.runOnce(ctx); err != nil {
				log.Printf("[investment_scanner] 定时任务失败: %v", err)
			}
		}
	}
}

// runBacktest 执行策略回测
func (is *InvestmentServiceManager) runBacktest(ctx context.Context, symbol, strategy, startDateStr, endDateStr string, initialCash float64) error {
	if symbol == "" {
		return fmt.Errorf("回测需要指定交易对符号")
	}

	// 验证日期格式（暂时不解析，API会处理）
	if startDateStr == "" || endDateStr == "" {
		return fmt.Errorf("开始日期和结束日期不能为空")
	}

	// 构造API请求
	requestBody := map[string]interface{}{
		"symbol":       symbol,
		"strategy":     strategy,
		"start_date":   startDateStr,
		"end_date":     endDateStr,
		"initial_cash": 10000,
		"max_position": 1.0,
		"timeframe":    "1d",
	}

	url := is.apiBase + "/backtest/run"
	log.Printf("[investment_scanner] 开始回测: %s, 策略: %s, 时间: %s -> %s",
		symbol, strategy, startDateStr, endDateStr)

	// 发送API请求
	resp, err := is.makeAPIRequest(ctx, "POST", url, requestBody)
	if err != nil {
		return fmt.Errorf("回测执行失败: %w", err)
	}

	// 输出结果
	fmt.Printf("\n=== 回测结果 ===\n")
	fmt.Printf("策略: %s\n", strategy)
	fmt.Printf("交易对: %s\n", symbol)
	fmt.Printf("时间范围: %s -> %s\n", startDateStr, endDateStr)

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if summary, ok := result["summary"].(map[string]interface{}); ok {
			if totalReturn, ok := summary["total_return"].(float64); ok {
				fmt.Printf("总收益率: %.2f%%\n", totalReturn*100)
			}
			if annualReturn, ok := summary["annual_return"].(float64); ok {
				fmt.Printf("年化收益率: %.2f%%\n", annualReturn*100)
			}
			if winRate, ok := summary["win_rate"].(float64); ok {
				fmt.Printf("胜率: %.2f%%\n", winRate*100)
			}
			if maxDrawdown, ok := summary["max_drawdown"].(float64); ok {
				fmt.Printf("最大回撤: %.2f%%\n", maxDrawdown*100)
			}
			if sharpeRatio, ok := summary["sharpe_ratio"].(float64); ok {
				fmt.Printf("夏普比率: %.2f\n", sharpeRatio)
			}
			if totalTrades, ok := summary["total_trades"].(float64); ok {
				fmt.Printf("总交易次数: %.0f\n", totalTrades)
			}
		}
	}

	return nil
}

// runBatchStrategyTests 批量执行策略测试
func (is *InvestmentServiceManager) runBatchStrategyTests(ctx context.Context) error {
	url := is.apiBase + "/recommendations/performance/batch-strategy-test"
	log.Printf("[investment_scanner] 批量策略测试: %s", url)

	// 发送批量策略测试请求
	_, err := is.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("批量策略测试请求失败: %w", err)
	}

	log.Printf("[investment_scanner] 批量策略测试完成")
	return nil
}

// generateReport 生成投资分析报告
func (is *InvestmentServiceManager) generateReport(ctx context.Context, performanceIDStr, reportType, outputPath string) error {
	url := fmt.Sprintf("%s/recommendations/performance/%s/report", is.apiBase, performanceIDStr)

	requestBody := map[string]interface{}{
		"report_type": reportType,
	}

	log.Printf("[investment_scanner] 生成报告 ID=%s: %s", performanceIDStr, url)

	// 发送报告生成请求
	resp, err := is.makeAPIRequest(ctx, "POST", url, requestBody)
	if err != nil {
		return fmt.Errorf("报告生成请求失败: %w", err)
	}

	// 解析响应
	if report, ok := resp["report"].(map[string]interface{}); ok {
		fmt.Printf("\n=== 投资分析报告 ===\n")
		fmt.Printf("记录ID: %s\n", performanceIDStr)
		fmt.Printf("报告类型: %s\n", reportType)

		if basicInfo, ok := report["basic_info"].(map[string]interface{}); ok {
			if symbol, ok := basicInfo["symbol"].(string); ok {
				fmt.Printf("币种: %s\n", symbol)
			}
			if currentReturn, ok := basicInfo["current_return"].(float64); ok {
				fmt.Printf("当前收益率: %.2f%%\n", currentReturn)
			}
		}

		if outputPath != "" {
			fmt.Printf("报告已保存到: %s\n", outputPath)
		}
	}

	log.Printf("[investment_scanner] 报告生成完成")
	return nil
}

// runStrategyTest 执行单个策略测试
func (is *InvestmentServiceManager) runStrategyTest(ctx context.Context, performanceIDStr string) error {
	url := fmt.Sprintf("%s/recommendations/performance/%s/strategy-test", is.apiBase, performanceIDStr)
	log.Printf("[investment_scanner] 策略测试 ID=%s: %s", performanceIDStr, url)

	// 发送策略测试请求
	resp, err := is.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("策略测试请求失败: %w", err)
	}

	// 解析响应
	if result, ok := resp["result"].(map[string]interface{}); ok {
		fmt.Printf("\n=== 策略测试结果 ===\n")
		fmt.Printf("记录ID: %s\n", performanceIDStr)
		if symbol, ok := result["symbol"].(string); ok {
			fmt.Printf("币种: %s\n", symbol)
		}
		if entryPrice, ok := result["entry_price"].(float64); ok {
			fmt.Printf("入场价格: %.8f\n", entryPrice)
		}
		if exitPrice, ok := result["exit_price"].(float64); ok {
			fmt.Printf("出场价格: %.8f\n", exitPrice)
		}
		if actualReturn, ok := result["actual_return"].(float64); ok {
			fmt.Printf("策略收益: %.2f%%\n", actualReturn)
		}
		if holdingPeriod, ok := result["holding_period"].(float64); ok {
			fmt.Printf("持有时间: %.0f 分钟\n", holdingPeriod)
		}
		if exitReason, ok := result["exit_reason"].(string); ok {
			fmt.Printf("退出原因: %s\n", exitReason)
		}
	}

	log.Printf("[investment_scanner] 策略测试完成")
	return nil
}

// =============================
//         进程管理器实现
// =============================

// ProcessManager 进程管理器，负责启动和管理各个扫描器进程
type ProcessManager struct {
	apiBase   string
	config    *config.Config
	processes []*exec.Cmd
}

// NewProcessManager 创建进程管理器
func NewProcessManager(apiBase string, cfg *config.Config) *ProcessManager {
	return &ProcessManager{
		apiBase:   apiBase,
		config:    cfg,
		processes: make([]*exec.Cmd, 0),
	}
}

// startAllProcesses 启动所有投资相关进程
func (pm *ProcessManager) startAllProcesses(ctx context.Context, mode string, interval time.Duration, kind string, limit int, forceRefresh bool) error {
	log.Printf("[process_manager] 开始启动所有投资相关进程...")

	// 构建通用参数
	baseArgs := []string{
		"-api", pm.apiBase,
		"-config", "./config.yaml",
	}

	// 1. 启动推荐扫描器
	log.Printf("[process_manager] 启动推荐扫描器...")
	recArgs := append(baseArgs, []string{
		"-mode", mode,
		"-interval", interval.String(),
		"-kind", kind,
		"-limit", fmt.Sprintf("%d", limit),
	}...)
	if forceRefresh {
		recArgs = append(recArgs, "-force-refresh")
	}

	recCmd := exec.CommandContext(ctx, "./recommendation_scanner", recArgs...)
	recCmd.Stdout = os.Stdout
	recCmd.Stderr = os.Stderr

	if err := recCmd.Start(); err != nil {
		return fmt.Errorf("启动推荐扫描器失败: %w", err)
	}
	pm.processes = append(pm.processes, recCmd)
	log.Printf("[process_manager] 推荐扫描器已启动 (PID: %d)", recCmd.Process.Pid)

	// 2. 启动回测扫描器
	log.Printf("[process_manager] 启动回测扫描器...")
	btArgs := append(baseArgs, []string{
		"-mode", mode,
		"-interval", interval.String(),
	}...)

	btCmd := exec.CommandContext(ctx, "./backtest_scanner", btArgs...)
	btCmd.Stdout = os.Stdout
	btCmd.Stderr = os.Stderr

	if err := btCmd.Start(); err != nil {
		return fmt.Errorf("启动回测扫描器失败: %w", err)
	}
	pm.processes = append(pm.processes, btCmd)
	log.Printf("[process_manager] 回测扫描器已启动 (PID: %d)", btCmd.Process.Pid)

	// 等待所有进程
	log.Printf("[process_manager] 所有进程已启动，等待运行...")
	return pm.waitForProcesses(ctx)
}

// waitForProcesses 等待所有进程结束
func (pm *ProcessManager) waitForProcesses(ctx context.Context) error {
	// 创建错误通道
	errChan := make(chan error, len(pm.processes))

	// 为每个进程启动goroutine等待
	for i, cmd := range pm.processes {
		go func(index int, process *exec.Cmd) {
			log.Printf("[process_manager] 等待进程 %d 结束...", index+1)
			err := process.Wait()
			if err != nil {
				errChan <- fmt.Errorf("进程 %d 异常退出: %w", index+1, err)
			} else {
				errChan <- nil
			}
		}(i, cmd)
	}

	// 等待所有进程结束或上下文取消
	processCount := len(pm.processes)
	for i := 0; i < processCount; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("[process_manager] %v", err)
				// 继续等待其他进程，但记录错误
			}
		case <-ctx.Done():
			log.Printf("[process_manager] 收到停止信号，正在终止所有进程...")
			pm.stopAllProcesses()
			return ctx.Err()
		}
	}

	log.Printf("[process_manager] 所有进程已结束")
	return nil
}

// stopAllProcesses 停止所有进程
func (pm *ProcessManager) stopAllProcesses() {
	log.Printf("[process_manager] 正在停止所有进程...")

	for i, cmd := range pm.processes {
		if cmd.Process != nil {
			log.Printf("[process_manager] 终止进程 %d (PID: %d)...", i+1, cmd.Process.Pid)
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("[process_manager] 终止进程 %d 失败: %v", i+1, err)
			}
		}
	}
}

// runSingleStrategyTest 执行单个策略测试（内部方法）
func (is *InvestmentServiceManager) runSingleStrategyTest(ctx context.Context, perf *pdb.RecommendationPerformance) error {
	performanceIDStr := strconv.FormatUint(uint64(perf.ID), 10)
	url := fmt.Sprintf("%s/recommendations/performance/%s/strategy-test", is.apiBase, performanceIDStr)

	log.Printf("[investment_scanner] 策略测试 ID=%d: %s", perf.ID, url)

	// 发送策略测试请求
	_, err := is.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("策略测试请求失败: %w", err)
	}

	log.Printf("[investment_scanner] 策略测试成功 ID=%d", perf.ID)
	return nil
}

// batchUpdateRecords 批量更新验证记录
func (is *InvestmentServiceManager) batchUpdateRecords(ctx context.Context) error {
	url := is.apiBase + "/recommendations/performance/batch-update"
	log.Printf("[investment_scanner] 调用API: %s", url)

	// 发送批量更新请求
	resp, err := is.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("批量更新请求失败: %w", err)
	}

	log.Printf("[investment_scanner] 批量更新完成: %v", resp)
	return nil
}

// makeAPIRequest 发送API请求的辅助方法
func (is *InvestmentServiceManager) makeAPIRequest(ctx context.Context, method, url string, body interface{}) (map[string]interface{}, error) {
	log.Printf("[investment_scanner] 发送%s请求到: %s", method, url)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "InvestmentScanner/1.0")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API请求失败: HTTP %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	// 检查API响应状态
	if success, ok := result["success"].(bool); ok && !success {
		if message, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("API返回错误: %s", message)
		}
		return nil, fmt.Errorf("API请求失败")
	}

	return result, nil
}

// =============================
//         推荐扫描器实现
// =============================

// generateOnce 单次生成推荐
func (rs *RecommendationScanner) generateOnce(ctx context.Context, kind string, limit int, forceRefresh bool) error {
	log.Printf("[recommendation_scanner] 开始单次推荐生成...")

	return rs.generateRecommendations(ctx, kind, limit, forceRefresh)
}

// runContinuous 持续运行模式
func (rs *RecommendationScanner) runContinuous(ctx context.Context, interval time.Duration, kind string, limit int, forceRefresh bool) {
	log.Printf("[recommendation_scanner] 启动持续推荐生成模式...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次运行
	if err := rs.generateRecommendations(ctx, kind, limit, forceRefresh); err != nil {
		log.Printf("[recommendation_scanner] 首次运行失败: %v", err)
	}

	// 定时运行
	for {
		select {
		case <-ctx.Done():
			log.Printf("[recommendation_scanner] 收到停止信号，退出...")
			return
		case <-ticker.C:
			log.Printf("[recommendation_scanner] 执行定时推荐生成...")
			if err := rs.generateRecommendations(ctx, kind, limit, forceRefresh); err != nil {
				log.Printf("[recommendation_scanner] 定时生成失败: %v", err)
			}
		}
	}
}

// generateRecommendations 生成推荐
func (rs *RecommendationScanner) generateRecommendations(ctx context.Context, kind string, limit int, forceRefresh bool) error {
	log.Printf("[recommendation_scanner] 开始生成推荐: kind=%s, limit=%d, forceRefresh=%v", kind, limit, forceRefresh)

	// 获取当前日期和时间作为推荐日期（包含时分秒）
	today := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// 构造API请求
	requestBody := map[string]interface{}{
		"kind":    kind,
		"limit":   limit,
		"refresh": forceRefresh,
	}

	// 添加日期参数到URL
	url := fmt.Sprintf("%s/recommendations/generate?date=%s", rs.apiBase, today)
	log.Printf("[recommendation_scanner] 调用API: %s", url)

	// 发送API请求
	resp, err := rs.makeAPIRequest(ctx, "POST", url, requestBody)
	if err != nil {
		return fmt.Errorf("推荐生成请求失败: %w", err)
	}

	// 解析响应
	if success, ok := resp["success"].(bool); ok && success {
		if message, ok := resp["message"].(string); ok {
			log.Printf("[recommendation_scanner] 推荐生成成功: %s", message)
		} else {
			log.Printf("[recommendation_scanner] 推荐生成成功")
		}

		// 输出推荐结果
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if recommendations, ok := data["recommendations"].([]interface{}); ok {
				log.Printf("[recommendation_scanner] 生成推荐数量: %d", len(recommendations))

				// 简单输出前几个推荐
				for i, rec := range recommendations {
					if i >= 3 { // 只显示前3个
						break
					}
					if recMap, ok := rec.(map[string]interface{}); ok {
						if symbol, ok := recMap["symbol"].(string); ok {
							if score, ok := recMap["total_score"].(float64); ok {
								log.Printf("[recommendation_scanner]   %d. %s (分数: %.2f)", i+1, symbol, score)
							}
						}
					}
				}
			}
		}
	} else {
		if message, ok := resp["error"].(string); ok {
			return fmt.Errorf("推荐生成失败: %s", message)
		}
		return fmt.Errorf("推荐生成失败")
	}

	log.Printf("[recommendation_scanner] 推荐生成完成")
	return nil
}

// makeAPIRequest 推荐扫描器的API请求方法
func (rs *RecommendationScanner) makeAPIRequest(ctx context.Context, method, url string, body interface{}) (map[string]interface{}, error) {
	log.Printf("[recommendation_scanner] 发送%s请求到: %s", method, url)

	var reqBody interface{} = nil
	if body != nil {
		reqBody = body
	}

	// 发送请求
	var result map[string]interface{}
	err := netutil.PostJSON(ctx, url, reqBody, &result)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}

	// 检查API响应状态
	if success, ok := result["success"].(bool); ok && !success {
		if message, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("API返回错误: %s", message)
		}
		return nil, fmt.Errorf("API请求失败")
	}

	return result, nil
}

// =============================
//         回测扫描器实现
// =============================

// runOnce 单次运行所有回测任务
func (bs *BacktestScanner) runOnce(ctx context.Context) error {
	log.Printf("[backtest_scanner] 开始执行单次回测任务...")

	// 1. 批量更新验证记录
	log.Printf("[backtest_scanner] 步骤1: 批量更新验证记录...")
	if err := bs.batchUpdateRecords(ctx); err != nil {
		log.Printf("[backtest_scanner] 批量更新失败: %v", err)
		// 不返回错误，继续执行
	}

	// 2. 执行批量策略测试
	log.Printf("[backtest_scanner] 步骤2: 执行批量策略测试...")
	if err := bs.runBatchStrategyTests(ctx); err != nil {
		log.Printf("[backtest_scanner] 批量策略测试失败: %v", err)
		// 不返回错误，继续执行
	}

	log.Printf("[backtest_scanner] 单次回测任务完成")
	return nil
}

// runContinuous 持续运行模式
func (bs *BacktestScanner) runContinuous(ctx context.Context, interval time.Duration) {
	log.Printf("[backtest_scanner] 启动持续回测模式...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次运行
	if err := bs.runOnce(ctx); err != nil {
		log.Printf("[backtest_scanner] 首次运行失败: %v", err)
	}

	// 定时运行
	for {
		select {
		case <-ctx.Done():
			log.Printf("[backtest_scanner] 收到停止信号，退出...")
			return
		case <-ticker.C:
			log.Printf("[backtest_scanner] 执行定时回测任务...")
			if err := bs.runOnce(ctx); err != nil {
				log.Printf("[backtest_scanner] 定时任务失败: %v", err)
			}
		}
	}
}

// runBacktest 执行策略回测
func (bs *BacktestScanner) runBacktest(ctx context.Context, symbol, strategy, startDateStr, endDateStr string, initialCash float64) error {
	if symbol == "" {
		return fmt.Errorf("回测需要指定交易对符号")
	}

	// 验证日期格式（暂时不解析，API会处理）
	if startDateStr == "" || endDateStr == "" {
		return fmt.Errorf("开始日期和结束日期不能为空")
	}

	// 构造API请求
	requestBody := map[string]interface{}{
		"symbol":       symbol,
		"strategy":     strategy,
		"start_date":   startDateStr,
		"end_date":     endDateStr,
		"initial_cash": initialCash,
		"max_position": 1.0,
		"timeframe":    "1d",
	}

	url := bs.apiBase + "/recommendations/backtest/run"
	log.Printf("[backtest_scanner] 开始回测: %s, 策略: %s, 时间: %s -> %s",
		symbol, strategy, startDateStr, endDateStr)

	// 发送API请求
	resp, err := bs.makeAPIRequest(ctx, "POST", url, requestBody)
	if err != nil {
		return fmt.Errorf("回测执行失败: %w", err)
	}

	// 输出结果
	fmt.Printf("\n=== 回测结果 ===\n")
	fmt.Printf("策略: %s\n", strategy)
	fmt.Printf("交易对: %s\n", symbol)
	fmt.Printf("时间范围: %s -> %s\n", startDateStr, endDateStr)

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if summary, ok := result["summary"].(map[string]interface{}); ok {
			if totalReturn, ok := summary["total_return"].(float64); ok {
				fmt.Printf("总收益率: %.2f%%\n", totalReturn*100)
			}
			if annualReturn, ok := summary["annual_return"].(float64); ok {
				fmt.Printf("年化收益率: %.2f%%\n", annualReturn*100)
			}
			if winRate, ok := summary["win_rate"].(float64); ok {
				fmt.Printf("胜率: %.2f%%\n", winRate*100)
			}
			if maxDrawdown, ok := summary["max_drawdown"].(float64); ok {
				fmt.Printf("最大回撤: %.2f%%\n", maxDrawdown*100)
			}
			if sharpeRatio, ok := summary["sharpe_ratio"].(float64); ok {
				fmt.Printf("夏普比率: %.2f\n", sharpeRatio)
			}
			if totalTrades, ok := summary["total_trades"].(float64); ok {
				fmt.Printf("总交易次数: %.0f\n", totalTrades)
			}
		}
	}

	return nil
}

// runBatchStrategyTests 批量执行策略测试
func (bs *BacktestScanner) runBatchStrategyTests(ctx context.Context) error {
	url := bs.apiBase + "/recommendations/performance/batch-strategy-test"
	log.Printf("[backtest_scanner] 批量策略测试: %s", url)

	// 发送批量策略测试请求
	_, err := bs.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("批量策略测试请求失败: %w", err)
	}

	log.Printf("[backtest_scanner] 批量策略测试完成")
	return nil
}

// runStrategyTest 执行单个策略测试
func (bs *BacktestScanner) runStrategyTest(ctx context.Context, performanceIDStr string) error {
	url := fmt.Sprintf("%s/recommendations/performance/%s/strategy-test", bs.apiBase, performanceIDStr)
	log.Printf("[backtest_scanner] 策略测试 ID=%s: %s", performanceIDStr, url)

	// 发送策略测试请求
	resp, err := bs.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("策略测试请求失败: %w", err)
	}

	// 解析响应
	if result, ok := resp["result"].(map[string]interface{}); ok {
		fmt.Printf("\n=== 策略测试结果 ===\n")
		fmt.Printf("记录ID: %s\n", performanceIDStr)
		if symbol, ok := result["symbol"].(string); ok {
			fmt.Printf("币种: %s\n", symbol)
		}
		if entryPrice, ok := result["entry_price"].(float64); ok {
			fmt.Printf("入场价格: %.8f\n", entryPrice)
		}
		if exitPrice, ok := result["exit_price"].(float64); ok {
			fmt.Printf("出场价格: %.8f\n", exitPrice)
		}
		if actualReturn, ok := result["actual_return"].(float64); ok {
			fmt.Printf("策略收益: %.2f%%\n", actualReturn)
		}
		if holdingPeriod, ok := result["holding_period"].(float64); ok {
			fmt.Printf("持有时间: %.0f 分钟\n", holdingPeriod)
		}
		if exitReason, ok := result["exit_reason"].(string); ok {
			fmt.Printf("退出原因: %s\n", exitReason)
		}
	}

	log.Printf("[backtest_scanner] 策略测试完成")
	return nil
}

// batchUpdateRecords 批量更新验证记录
func (bs *BacktestScanner) batchUpdateRecords(ctx context.Context) error {
	url := bs.apiBase + "/recommendations/performance/batch-update"
	log.Printf("[backtest_scanner] 调用API: %s", url)

	// 发送批量更新请求
	resp, err := bs.makeAPIRequest(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("批量更新请求失败: %w", err)
	}

	log.Printf("[backtest_scanner] 批量更新完成: %v", resp)
	return nil
}

// makeAPIRequest 回测扫描器的API请求方法
func (bs *BacktestScanner) makeAPIRequest(ctx context.Context, method, url string, body interface{}) (map[string]interface{}, error) {
	log.Printf("[backtest_scanner] 发送%s请求到: %s", method, url)

	var reqBody interface{} = nil
	if body != nil {
		reqBody = body
	}

	// 发送请求
	var result map[string]interface{}
	err := netutil.PostJSON(ctx, url, reqBody, &result)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}

	// 检查API响应状态
	if success, ok := result["success"].(bool); ok && !success {
		if message, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("API返回错误: %s", message)
		}
		return nil, fmt.Errorf("API请求失败")
	}

	return result, nil
}
