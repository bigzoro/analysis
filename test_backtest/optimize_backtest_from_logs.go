// optimize_backtest_from_logs.go
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LogAnalyzer 日志分析器
type LogAnalyzer struct {
	logFile     string
	metrics     BacktestMetrics
	issues      []string
	optimizations []OptimizationSuggestion
}

// BacktestMetrics 回测指标
type BacktestMetrics struct {
	TotalTrades         int
	WinRate            float64
	TotalReturn        float64
	MaxDrawdown        float64
	SharpeRatio        float64
	TransformerWeight   float64
	AutoExecuteCount    int
	SkippedTrades       int
	ErrorCount         int
	ProcessingTime     time.Duration
}

// OptimizationSuggestion 优化建议
type OptimizationSuggestion struct {
	Type        string
	Description string
	Severity    string // "high", "medium", "low"
	Action      string
	ExpectedBenefit string
}

// NewLogAnalyzer 创建日志分析器
func NewLogAnalyzer(logFile string) *LogAnalyzer {
	return &LogAnalyzer{
		logFile: logFile,
		metrics: BacktestMetrics{},
		issues:  []string{},
		optimizations: []OptimizationSuggestion{},
	}
}

// AnalyzeLogs 分析日志文件
func (la *LogAnalyzer) AnalyzeLogs() error {
	file, err := os.Open(la.logFile)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// 分析每一行日志
		la.analyzeLine(line)

		// 每1000行显示进度
		if lineCount%1000 == 0 {
			fmt.Printf("已处理 %d 行日志...\n", lineCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 生成优化建议
	la.generateOptimizations()

	return nil
}

// analyzeLine 分析单行日志
func (la *LogAnalyzer) analyzeLine(line string) {
	// 交易统计
	if strings.Contains(line, "[RunBacktest] 回测完成") {
		la.parseBacktestResults(line)
	}

	// Transformer参与情况
	if strings.Contains(line, "[ENSEMBLE] 模型 transformer") {
		la.parseTransformerMetrics(line)
	}

	// 自动执行统计
	if strings.Contains(line, "[AUTO_EXECUTE]") {
		la.metrics.AutoExecuteCount++
	}

	// 跳过交易统计
	if strings.Contains(line, "skip_existing_trades") || strings.Contains(line, "SkipExistingTrades") {
		la.metrics.SkippedTrades++
	}

	// 错误统计
	if strings.Contains(line, "[ERROR]") || strings.Contains(line, "❌") {
		la.metrics.ErrorCount++
		la.issues = append(la.issues, line)
	}

	// 趋势过滤器分析
	if strings.Contains(line, "[TREND_FILTER]") {
		la.analyzeTrendFilter(line)
	}

	// 自动选择币种分析
	if strings.Contains(line, "[AUTO_SELECT]") {
		la.analyzeAutoSelect(line)
	}

	// 处理时间分析
	if strings.Contains(line, "Processing time") || strings.Contains(line, "处理时间") {
		la.parseProcessingTime(line)
	}
}

// parseBacktestResults 解析回测结果
func (la *LogAnalyzer) parseBacktestResults(line string) {
	// 示例日志: [RunBacktest] 回测完成: 总收益率=15.23%, 胜率=68.50%, 交易次数=127

	re := regexp.MustCompile(`总收益率=([0-9.-]+)%.*胜率=([0-9.-]+)%.*交易次数=([0-9]+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 4 {
		if totalReturn, err := strconv.ParseFloat(matches[1], 64); err == nil {
			la.metrics.TotalReturn = totalReturn
		}
		if winRate, err := strconv.ParseFloat(matches[2], 64); err == nil {
			la.metrics.WinRate = winRate
		}
		if trades, err := strconv.Atoi(matches[3]); err == nil {
			la.metrics.TotalTrades = trades
		}
	}
}

// parseTransformerMetrics 解析Transformer指标
func (la *LogAnalyzer) parseTransformerMetrics(line string) {
	// 示例日志: [ENSEMBLE] 模型 transformer: score=0.45, confidence=0.82, weight=0.30

	re := regexp.MustCompile(`weight=([0-9.]+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 2 {
		if weight, err := strconv.ParseFloat(matches[1], 64); err == nil {
			la.metrics.TransformerWeight = weight
		}
	}
}

// analyzeTrendFilter 分析趋势过滤器
func (la *LogAnalyzer) analyzeTrendFilter(line string) {
	if strings.Contains(line, "完全禁止交易") {
		la.issues = append(la.issues, "趋势过滤器过于严格："+line)
	}
}

// analyzeAutoSelect 分析自动选择币种
func (la *LogAnalyzer) analyzeAutoSelect(line string) {
	// 记录自动选择的相关信息
	if strings.Contains(line, "启用自动选择币种模式") {
		fmt.Println("✓ 自动选择币种功能已启用")
	}
}

// parseProcessingTime 解析处理时间
func (la *LogAnalyzer) parseProcessingTime(line string) {
	// 解析处理时间（如果有的话）
}

// generateOptimizations 生成优化建议
func (la *LogAnalyzer) generateOptimizations() {
	// 基于分析结果生成优化建议

	// 1. 检查交易次数
	if la.metrics.TotalTrades == 0 {
		la.optimizations = append(la.optimizations, OptimizationSuggestion{
			Type: "交易频率",
			Description: "回测期间没有产生任何交易",
			Severity: "high",
			Action: "降低趋势过滤器阈值或调整市场条件判断",
			ExpectedBenefit: "产生交易信号",
		})
	} else if la.metrics.TotalTrades < 10 {
		la.optimizations = append(la.optimizations, OptimizationSuggestion{
			Type: "交易频率",
			Description: fmt.Sprintf("交易次数过少 (%d 次)", la.metrics.TotalTrades),
			Severity: "medium",
			Action: "调整仓位大小或放宽入场条件",
			ExpectedBenefit: "增加交易频率",
		})
	}

	// 2. 检查胜率
	if la.metrics.WinRate < 50.0 && la.metrics.TotalTrades > 0 {
		la.optimizations = append(la.optimizations, OptimizationSuggestion{
			Type: "胜率优化",
			Description: fmt.Sprintf("胜率偏低 (%.1f%%)", la.metrics.WinRate),
			Severity: "medium",
			Action: "调整止损/止盈比例或改进入场时机",
			ExpectedBenefit: "提升胜率",
		})
	}

	// 3. 检查Transformer权重
	if la.metrics.TransformerWeight < 0.2 {
		la.optimizations = append(la.optimizations, OptimizationSuggestion{
			Type: "Transformer优化",
			Description: fmt.Sprintf("Transformer权重过低 (%.2f)", la.metrics.TransformerWeight),
			Severity: "medium",
			Action: "增加Transformer初始权重或改善模型表现",
			ExpectedBenefit: "提升AI决策质量",
		})
	}

	// 4. 检查错误数量
	if la.metrics.ErrorCount > 5 {
		la.optimizations = append(la.optimizations, OptimizationSuggestion{
			Type: "系统稳定性",
			Description: fmt.Sprintf("发现 %d 个错误", la.metrics.ErrorCount),
			Severity: "high",
			Action: "检查系统配置和数据质量",
			ExpectedBenefit: "提升系统稳定性",
		})
	}

	// 5. 检查收益表现
	if la.metrics.TotalReturn < 0 && la.metrics.TotalTrades > 0 {
		la.optimizations = append(la.optimizations, OptimizationSuggestion{
			Type: "收益优化",
			Description: fmt.Sprintf("总收益率负数 (%.2f%%)", la.metrics.TotalReturn),
			Severity: "high",
			Action: "调整策略参数或更换交易策略",
			ExpectedBenefit: "改善收益表现",
		})
	}
}

// PrintReport 打印分析报告
func (la *LogAnalyzer) PrintReport() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 回测日志分析报告")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n🔢 核心指标:")
	fmt.Printf("  交易次数: %d\n", la.metrics.TotalTrades)
	fmt.Printf("  胜率: %.2f%%\n", la.metrics.WinRate)
	fmt.Printf("  总收益率: %.2f%%\n", la.metrics.TotalReturn)
	fmt.Printf("  Transformer权重: %.2f\n", la.metrics.TransformerWeight)
	fmt.Printf("  自动执行次数: %d\n", la.metrics.AutoExecuteCount)
	fmt.Printf("  跳过交易数: %d\n", la.metrics.SkippedTrades)
	fmt.Printf("  错误数量: %d\n", la.metrics.ErrorCount)

	fmt.Println("\n⚠️ 发现的问题:")
	if len(la.issues) == 0 {
		fmt.Println("  ✅ 没有发现严重问题")
	} else {
		for i, issue := range la.issues {
			if i >= 5 { // 只显示前5个问题
				fmt.Printf("  ... 还有 %d 个问题\n", len(la.issues)-5)
				break
			}
			fmt.Printf("  • %s\n", issue)
		}
	}

	fmt.Println("\n💡 优化建议:")
	if len(la.optimizations) == 0 {
		fmt.Println("  ✅ 系统表现良好，无需优化")
	} else {
		for _, opt := range la.optimizations {
			severityIcon := map[string]string{
				"high":   "🔴",
				"medium": "🟡",
				"low":    "🟢",
			}

			fmt.Printf("  %s [%s] %s\n", severityIcon[opt.Severity], opt.Type, opt.Description)
			fmt.Printf("    建议行动: %s\n", opt.Action)
			fmt.Printf("    预期收益: %s\n\n", opt.ExpectedBenefit)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

// SaveOptimizationConfig 生成优化配置
func (la *LogAnalyzer) SaveOptimizationConfig(filename string) error {
	config := map[string]interface{}{
		"analysis_time": time.Now().Format("2006-01-02 15:04:05"),
		"metrics": la.metrics,
		"issues": la.issues,
		"optimizations": la.optimizations,
		"recommended_config": la.generateRecommendedConfig(),
	}

	// 这里可以保存为JSON文件用于后续优化
	fmt.Printf("优化配置已生成，可保存到文件: %s\n", filename)
	_ = config // 避免未使用变量警告
	return nil
}

// generateRecommendedConfig 生成推荐配置
func (la *LogAnalyzer) generateRecommendedConfig() map[string]interface{} {
	config := make(map[string]interface{})

	// 基于分析结果生成推荐配置
	if la.metrics.TotalTrades == 0 {
		config["trend_filter_threshold"] = 0.05 // 降低趋势阈值
		config["market_condition_filter"] = false // 关闭市场条件过滤
	}

	if la.metrics.TransformerWeight < 0.2 {
		config["transformer_initial_weight"] = 0.5 // 提高Transformer权重
	}

	if la.metrics.WinRate < 50.0 {
		config["stop_loss_multiplier"] = 0.8 // 调整止损比例
		config["take_profit_multiplier"] = 1.2 // 调整止盈比例
	}

	return config
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run optimize_backtest_from_logs.go <日志文件路径>")
		fmt.Println("示例: go run optimize_backtest_from_logs.go backtest.log")
		os.Exit(1)
	}

	logFile := os.Args[1]

	fmt.Printf("开始分析日志文件: %s\n", logFile)

	analyzer := NewLogAnalyzer(logFile)

	if err := analyzer.AnalyzeLogs(); err != nil {
		log.Fatalf("分析日志失败: %v", err)
	}

	analyzer.PrintReport()

	// 保存优化配置
	configFile := strings.TrimSuffix(logFile, ".log") + "_optimization.json"
	if err := analyzer.SaveOptimizationConfig(configFile); err != nil {
		fmt.Printf("保存优化配置失败: %v\n", err)
	}
}
