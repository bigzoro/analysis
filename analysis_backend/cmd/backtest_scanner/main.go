// cmd/backtest_scanner/main.go
package main

import (
	"analysis/internal/config"
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"analysis/internal/netutil"
)

// =============================
//         回测扫描器
// =============================

// BacktestScanner 回测扫描器
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

func main() {
	// 命令行参数
	apiBase := flag.String("api", "http://127.0.0.1:8010", "API服务器地址")
	configPath := flag.String("config", "./config.yaml", "配置文件路径")
	mode := flag.String("mode", "continuous", "运行模式: once(单次运行), continuous(持续运行), backtest(执行回测), strategy(策略测试), batch(批量处理)")
	interval := flag.Duration("interval", 1*time.Hour, "连续模式下的运行间隔")

	// 回测参数
	symbol := flag.String("symbol", "", "交易对符号（用于单个回测）")
	strategy := flag.String("strategy", "buy_and_hold", "回测策略类型")
	startDate := flag.String("start-date", "", "回测开始日期 (YYYY-MM-DD)")
	endDate := flag.String("end-date", "", "回测结束日期 (YYYY-MM-DD)")
	initialCash := flag.Float64("initial-cash", 10000, "初始资金")

	// 策略测试参数
	performanceID := flag.String("performance-id", "", "单个表现验证ID")

	flag.Parse()

	log.Printf("[backtest_scanner] 启动回测扫描器...")
	log.Printf("[backtest_scanner] API: %s, 模式: %s", *apiBase, *mode)

	// 加载配置
	var cfg config.Config
	config.MustLoad(*configPath, &cfg)
	config.ApplyProxy(&cfg)

	// 创建扫描器
	scanner := NewBacktestScanner(*apiBase, &cfg)

	// 根据模式运行
	ctx := context.Background()

	switch *mode {
	case "once":
		log.Printf("[backtest_scanner] 执行单次回测任务...")
		if err := scanner.runOnce(ctx); err != nil {
			log.Fatalf("[backtest_scanner] 单次任务失败: %v", err)
		}
		log.Printf("[backtest_scanner] 单次任务完成")

	case "continuous":
		log.Printf("[backtest_scanner] 启动持续回测模式，间隔: %v", *interval)
		scanner.runContinuous(ctx, *interval)

	case "backtest":
		log.Printf("[backtest_scanner] 执行策略回测...")
		if *symbol == "" {
			log.Fatalf("[backtest_scanner] 回测需要指定交易对符号")
		}
		if err := scanner.runBacktest(ctx, *symbol, *strategy, *startDate, *endDate, *initialCash); err != nil {
			log.Fatalf("[backtest_scanner] 回测失败: %v", err)
		}

	case "strategy":
		log.Printf("[backtest_scanner] 执行策略测试...")
		if *performanceID == "" {
			log.Fatalf("[backtest_scanner] 策略测试需要指定performance-id")
		}
		if err := scanner.runStrategyTest(ctx, *performanceID); err != nil {
			log.Fatalf("[backtest_scanner] 策略测试失败: %v", err)
		}

	case "batch":
		log.Printf("[backtest_scanner] 执行批量策略测试...")
		if err := scanner.runBatchStrategyTests(ctx); err != nil {
			log.Fatalf("[backtest_scanner] 批量策略测试失败: %v", err)
		}

	default:
		log.Fatalf("[backtest_scanner] 不支持的模式: %s", *mode)
	}
}

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

	url := bs.apiBase + "/recommendations/backtest"
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

// makeAPIRequest 发送API请求的辅助方法
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
