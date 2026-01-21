package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 策略执行时间间隔分析 ===\n")

	// 用户配置
	runInterval := 1     // 运行间隔：1分钟
	executionDelay := 30 // 执行延迟：30秒

	fmt.Printf("用户配置:\n")
	fmt.Printf("  运行间隔: %d 分钟\n", runInterval)
	fmt.Printf("  执行延迟: %d 秒\n", executionDelay)
	fmt.Printf("\n")

	// 系统机制
	checkInterval := 1 * time.Minute // 策略检查循环：每1分钟
	tickInterval := 1 * time.Second  // 订单执行循环：每1秒

	fmt.Printf("系统机制:\n")
	fmt.Printf("  策略检查循环: 每 %v\n", checkInterval)
	fmt.Printf("  订单执行循环: 每 %v\n", tickInterval)
	fmt.Printf("\n")

	fmt.Println("📊 执行时间线分析：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("时刻 T0: 策略启动\n")
	fmt.Printf("├─ 策略检查循环开始运行 (每1分钟检查)\n")
	fmt.Printf("├─ 订单执行循环开始运行 (每1秒检查)\n")
	fmt.Printf("\n")

	fmt.Printf("时刻 T1: 策略检查循环发现策略需要执行\n")
	fmt.Printf("├─ 创建执行记录\n")
	fmt.Printf("├─ 设置订单TriggerTime = 当前时间 + %d秒\n", executionDelay)
	fmt.Printf("├─ 订单状态: pending\n")
	fmt.Printf("\n")

	fmt.Printf("时刻 T2: 订单执行循环发现订单到期 (TriggerTime <= 当前时间)\n")
	fmt.Printf("├─ 订单状态: pending → processing\n")
	fmt.Printf("├─ 开始实际执行订单\n")
	fmt.Printf("\n")

	fmt.Printf("时刻 T3: 订单执行完成\n")
	fmt.Printf("├─ 更新LastRunAt = 当前时间\n")
	fmt.Printf("├─ 等待下次执行周期\n")
	fmt.Printf("\n")

	fmt.Println("⏰ 关键时间点计算：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("假设当前时间: 10:00:00\n")
	fmt.Printf("\n")

	// 第一次执行
	fmt.Printf("📅 第一次执行:\n")
	fmt.Printf("10:00:00 - 策略检查循环开始\n")
	fmt.Printf("10:00:00 - 发现策略需要执行 (LastRunAt = null)\n")
	fmt.Printf("10:00:00 - 创建执行记录，TriggerTime = 10:00:00 + 30s = 10:00:30\n")
	fmt.Printf("10:00:30 - 订单开始执行\n")
	fmt.Printf("10:00:35 - 订单执行完成，LastRunAt = 10:00:35\n")
	fmt.Printf("\n")

	// 第二次执行
	fmt.Printf("📅 第二次执行:\n")
	fmt.Printf("10:01:00 - 策略检查循环检查 (第1分钟)\n")
	fmt.Printf("          - 计算下次执行时间: 10:00:35 + 1分钟 = 10:01:35\n")
	fmt.Printf("          - 当前时间 10:01:00 < 10:01:35，跳过\n")
	fmt.Printf("\n")

	fmt.Printf("10:01:35 - 策略检查循环检查 (第1分35秒)\n")
	fmt.Printf("          - 当前时间 10:01:35 >= 10:01:35，准备执行\n")
	fmt.Printf("10:01:35 - 创建执行记录，TriggerTime = 10:01:35 + 30s = 10:02:05\n")
	fmt.Printf("10:02:05 - 订单开始执行\n")
	fmt.Printf("10:02:10 - 订单执行完成，LastRunAt = 10:02:10\n")
	fmt.Printf("\n")

	fmt.Println("🔄 周期性分析：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("策略执行周期 = 运行间隔 + 执行延迟\n")
	fmt.Printf("               = %d分钟 + %d秒\n", runInterval, executionDelay)
	fmt.Printf("               = %.1f分钟\n", float64(runInterval*60+executionDelay)/60)
	fmt.Printf("\n")

	fmt.Printf("实际执行频率: 每 %.1f 分钟执行一次\n", float64(runInterval*60+executionDelay)/60)
	fmt.Printf("理论执行间隔: %d分钟%d秒\n", runInterval, executionDelay)
	fmt.Printf("\n")

	fmt.Println("⚡ 性能影响：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("1. 系统响应延迟: 最快%d秒 (ExecutionDelay)\n", executionDelay)
	fmt.Printf("2. 执行时间窗口: ±%v (检查循环间隔)\n", checkInterval)
	fmt.Printf("3. 并发处理能力: 每秒最多处理20个到期订单\n")
	fmt.Printf("4. 资源消耗: 两个后台goroutine持续运行\n")
	fmt.Printf("\n")

	fmt.Println("🎯 实际运行频率：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("• 理论间隔: %d分钟%d秒 = %d秒\n", runInterval, executionDelay, runInterval*60+executionDelay)
	fmt.Printf("• 实际间隔: 约%.1f分钟 (包含检查和执行时间)\n", float64(runInterval*60+executionDelay)/60)
	fmt.Printf("• 执行次数: 每小时约%.1f次\n", 3600/float64(runInterval*60+executionDelay))
	fmt.Printf("\n")

	fmt.Println("📋 配置优化建议：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	suggestions := []struct {
		config    string
		current   string
		suggested string
		reason    string
	}{
		{"运行间隔", "1分钟", "5-10分钟", "减少系统负载"},
		{"执行延迟", "30秒", "10-60秒", "平衡响应速度和稳定性"},
		{"检查频率", "1分钟", "30秒-2分钟", "根据策略重要性调整"},
	}

	for _, s := range suggestions {
		fmt.Printf("• %s: %s → %s (%s)\n", s.config, s.current, s.suggested, s.reason)
	}

	fmt.Printf("\n💡 结论：\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("在你的配置下 (运行间隔1分钟，执行延迟30秒)，\n")
	fmt.Printf("策略大约每 %.1f 分钟执行一次，\n", float64(runInterval*60+executionDelay)/60)
	fmt.Printf("比理论的1分钟间隔稍长，这是由于执行延迟和系统处理时间。\n")
}
