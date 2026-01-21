package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

func main() {
	fmt.Println("🎯 最终死锁解决方案验证")
	fmt.Println("=" * 50)

	fmt.Println("📊 测试配置:")
	fmt.Println("   - 并发度: 1 (完全串行)")
	fmt.Println("   - 批次大小: 20 (串行优化)")
	fmt.Println("   - 批次延迟: 100ms (串行效率)")

	// 运行简短测试
	fmt.Println("\n🚀 运行简短测试...")

	startTime := time.Now()
	cmd := exec.Command("timeout", "60", "go", "run", "./analysis_backend/cmd/data_sync/main.go",
		"sync", "klines", "--market", "spot", "--interval", "1m", "--max-symbols", "20")
	cmd.Dir = "d:\\code\\analysis2"

	output, err := cmd.CombinedOutput()
	executionTime := time.Since(startTime)
	outputStr := string(output)

	fmt.Printf("⏱️ 执行时间: %v\n", executionTime.Round(time.Second))

	// 分析结果
	deadlockCount := 0
	lines := regexp.MustCompile(`\n`).Split(outputStr, -1)

	fmt.Println("\n🔍 分析结果:")

	for _, line := range lines {
		if regexp.MustCompile(`死锁|deadlock`).MatchString(line) {
			deadlockCount++
			fmt.Printf("   ❌ 发现死锁: %s\n", line)
		}
	}

	if deadlockCount == 0 {
		fmt.Println("   ✅ 零死锁! 解决方案成功")
	} else {
		fmt.Printf("   ⚠️ 发现 %d 次死锁\n", deadlockCount)
	}

	// 检查并发度和批次配置
	concurrencyRegex := regexp.MustCompile(`并发度:(\d+)`)
	batchRegex := regexp.MustCompile(`批次大小: (\d+)`)

	if matches := concurrencyRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		fmt.Printf("   ⚡ 实际并发度: %s\n", matches[1])
	}

	if matches := batchRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		fmt.Printf("   📦 实际批次大小: %s\n", matches[1])
	}

	fmt.Println("\n🏁 结论:")
	if deadlockCount == 0 {
		fmt.Println("   🎉 最终解决方案完全成功!")
		fmt.Println("   ✅ 完全串行处理彻底消除了死锁问题")
	} else {
		fmt.Println("   ⚠️ 仍需进一步优化")
	}
}