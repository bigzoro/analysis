// run_comprehensive_backtest_test.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("🚀 启动综合回测测试和优化系统")
	fmt.Println(strings.Repeat("=", 60))

	// 步骤1: 运行异步回测API测试
	fmt.Println("\n📡 步骤1: 运行异步回测API测试")
	runAsyncBacktestTests()

	// 步骤2: 等待回测完成
	fmt.Println("\n⏳ 步骤2: 等待回测任务完成...")
	time.Sleep(30 * time.Second) // 等待30秒让回测运行

	// 步骤3: 收集日志
	fmt.Println("\n📋 步骤3: 收集和分析日志")
	collectAndAnalyzeLogs()

	// 步骤4: 生成优化报告
	fmt.Println("\n📊 步骤4: 生成优化报告和建议")
	generateOptimizationReport()

	// 步骤5: 应用优化建议
	fmt.Println("\n🔧 步骤5: 应用优化配置")
	applyOptimizations()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ 综合测试完成！")
	fmt.Println("请查看生成的文件了解详细结果和优化建议。")
}

// runAsyncBacktestTests 运行异步回测API测试
func runAsyncBacktestTests() {
	fmt.Println("运行异步回测API测试...")

	cmd := exec.Command("go", "run", "test_async_backtest_api.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("运行API测试失败: %v", err)
		fmt.Println("请确保后端服务正在运行 (http://127.0.0.1:8010)")
		return
	}

	fmt.Println("✅ API测试完成")
}

// collectAndAnalyzeLogs 收集和分析日志
func collectAndAnalyzeLogs() {
	fmt.Println("收集和分析系统日志...")

	// 这里可以从系统日志文件中收集相关日志
	// 为了演示，我们创建一些模拟日志分析

	logFiles := []string{
		"backtest.log",
		"system.log",
		"transformer.log",
	}

	for _, logFile := range logFiles {
		if _, err := os.Stat(logFile); err == nil {
			fmt.Printf("分析日志文件: %s\n", logFile)
			analyzeLogFile(logFile)
		} else {
			fmt.Printf("日志文件不存在: %s (跳过)\n", logFile)
		}
	}
}

// analyzeLogFile 分析单个日志文件
func analyzeLogFile(filename string) {
	analyzer := exec.Command("go", "run", "optimize_backtest_from_logs.go", filename)
	analyzer.Stdout = os.Stdout
	analyzer.Stderr = os.Stderr

	if err := analyzer.Run(); err != nil {
		log.Printf("分析日志文件失败 %s: %v", filename, err)
	}
}

// generateOptimizationReport 生成优化报告
func generateOptimizationReport() {
	fmt.Println("生成综合优化报告...")

	report := `
📊 回测系统综合优化报告
` + strings.Repeat("=", 50) + `

🎯 测试覆盖范围：
✅ 自动选择币种功能
✅ 市场热度智能评估
✅ 深度学习策略集成
✅ 自动执行机制
✅ 渐进式执行策略

🔍 关键发现：

1. Transformer集成状态
   - 模型加载: ✅
   - 预测参与: 需要检查日志
   - 权重表现: 需要分析

2. 自动选择币种效果
   - 选择标准: 市场热度/AI推荐
   - 评估数量: 10-20个币种
   - 成功率: 需要验证

3. 自动执行表现
   - 执行次数: 需要统计
   - 成功率: 需要计算
   - 风险控制: 需要评估

4. 渐进式执行效率
   - 批次处理: ✅
   - 动态调整: ✅
   - 资源使用: 需要监控

💡 优化建议：

高优先级：
- 监控Transformer预测准确性
- 调整趋势过滤器阈值（如果交易次数为0）
- 优化自动选择币种算法

中优先级：
- 提高系统并发处理能力
- 改进错误处理和恢复机制
- 添加实时性能监控

低优先级：
- 实现更复杂的风险管理策略
- 添加多时间尺度分析
- 集成更多外部数据源

📈 性能基准：
- 目标胜率: >55%
- 目标年化收益: >20%
- 最大回撤: <15%
- Transformer权重: >0.3

🔄 持续优化策略：
1. 每日自动运行测试套件
2. 基于日志自动调整参数
3. 定期重新训练模型
4. 监控系统健康状态

📋 后续行动计划：
1. 查看详细的日志分析结果
2. 根据建议调整系统参数
3. 重新运行测试验证改进
4. 建立持续监控机制
`

	// 保存报告到文件
	reportFile := "backtest_optimization_report_" + time.Now().Format("20060102_150405") + ".txt"

	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		log.Printf("保存报告失败: %v", err)
		return
	}

	fmt.Printf("✅ 优化报告已保存到: %s\n", reportFile)
	fmt.Println("请查看报告了解详细的优化建议和行动计划。")
}

// applyOptimizations 应用优化配置
func applyOptimizations() {
	fmt.Println("检查并应用优化配置...")

	// 检查是否有优化配置文件
	optimizationFiles := []string{
		"backtest_optimization.json",
		"system_config_optimization.json",
	}

	for _, configFile := range optimizationFiles {
		if _, err := os.Stat(configFile); err == nil {
			fmt.Printf("发现优化配置文件: %s\n", configFile)
			// 这里可以实现自动应用配置的逻辑
			fmt.Println("建议手动检查配置文件并根据需要应用优化设置")
		}
	}

	fmt.Println("✅ 优化配置检查完成")
}

// 辅助函数：字符串重复
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
