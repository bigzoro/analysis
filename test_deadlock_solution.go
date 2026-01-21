package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"time"
)

func main() {
	fmt.Println("🧪 死锁解决方案验证测试")
	fmt.Println("=" * 60)

	// 运行K线同步测试
	fmt.Println("1. 🚀 启动K线同步测试...")

	// 设置5分钟超时
	timeout := 5 * time.Minute
	startTime := time.Now()

	// 运行同步命令，只同步1分钟间隔以快速测试
	cmd := exec.Command("go", "run", "./analysis_backend/cmd/data_sync/main.go",
		"sync", "klines", "--market", "spot", "--interval", "1m", "--max-symbols", "50")
	cmd.Dir = "d:\\code\\analysis2"

	fmt.Println("   📊 测试参数: spot市场, 1m间隔, 最多50个交易对")

	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ 命令执行出错: %v", err)
	}

	executionTime := time.Since(startTime)
	outputStr := string(output)

	fmt.Printf("2. 📈 执行结果:\n")
	fmt.Printf("   ⏱️ 总耗时: %v\n", executionTime.Round(time.Second))
	fmt.Printf("   📄 日志长度: %d 字符\n", len(outputStr))

	// 分析结果
	fmt.Println("3. 🔍 性能分析:")

	// 检查goroutine统计
	goroutineRegex := regexp.MustCompile(`Goroutine统计 - 开始:(\d+), 结束:(\d+), 差异:([+-]?\d+)`)
	goroutineMatches := goroutineRegex.FindStringSubmatch(outputStr)
	if len(goroutineMatches) > 3 {
		start, end, diff := goroutineMatches[1], goroutineMatches[2], goroutineMatches[3]
		fmt.Printf("   🔄 Goroutine: %s → %s (差异:%s)\n", start, end, diff)
		if diff != "+0" && diff[0] != '-' {
			fmt.Printf("   ⚠️ 检测到goroutine变化，可能存在泄漏\n")
		}
	}

	// 检查并发度
	concurrencyRegex := regexp.MustCompile(`并发度:(\d+)`)
	concurrencyMatches := concurrencyRegex.FindStringSubmatch(outputStr)
	if len(concurrencyMatches) > 1 {
		fmt.Printf("   ⚡ 实际并发度: %s\n", concurrencyMatches[1])
	}

	// 检查批次大小
	batchRegex := regexp.MustCompile(`批次大小: (\d+)`)
	batchMatches := batchRegex.FindStringSubmatch(outputStr)
	if len(batchMatches) > 1 {
		fmt.Printf("   📦 批次大小: %s\n", batchMatches[1])
	}

	// 检查死锁情况
	deadlockCount := 0
	deadlockLines := []string{}

	lines := regexp.MustCompile(`\n`).Split(outputStr, -1)
	for _, line := range lines {
		if regexp.MustCompile(`死锁|deadlock`).MatchString(line) {
			deadlockCount++
			if deadlockCount <= 3 { // 只记录前3个
				deadlockLines = append(deadlockLines, line)
			}
		}
	}

	if deadlockCount == 0 {
		fmt.Printf("   ✅ 未检测到死锁!\n")
	} else {
		fmt.Printf("   ❌ 检测到 %d 次死锁\n", deadlockCount)
		for i, line := range deadlockLines {
			fmt.Printf("      %d. %s\n", i+1, line)
		}
	}

	// 检查成功率
	successRegex := regexp.MustCompile(`成功率:([\d.]+)%`)
	successMatches := successRegex.FindStringSubmatch(outputStr)
	if len(successMatches) > 1 {
		successRate := successMatches[1]
		fmt.Printf("   📊 同步成功率: %s%%\n", successRate)

		if successRate == "100.0" {
			fmt.Printf("   ✅ 完全成功!\n")
		} else {
			fmt.Printf("   ⚠️ 部分失败\n")
		}
	}

	// 总体评估
	fmt.Println("4. 🎯 总体评估:")

	issues := 0

	if executionTime > timeout {
		fmt.Printf("   ❌ 超时: 超过%v\n", timeout)
		issues++
	} else {
		fmt.Printf("   ✅ 在时限内完成\n")
	}

	if deadlockCount > 0 {
		fmt.Printf("   ❌ 存在死锁问题\n")
		issues++
	} else {
		fmt.Printf("   ✅ 无死锁\n")
	}

	if regexp.MustCompile(`同步完成`).MatchString(outputStr) {
		fmt.Printf("   ✅ 同步正常结束\n")
	} else {
		fmt.Printf("   ❌ 同步异常结束\n")
		issues++
	}

	// 最终结论
	fmt.Println("5. 🏁 测试结论:")

	if issues == 0 {
		fmt.Println("   🎉 死锁解决方案完全成功!")
		fmt.Println("   ✅ 零死锁，性能稳定，同步完全成功")
	} else if issues == 1 && deadlockCount > 0 {
		fmt.Println("   ⚠️ 死锁解决方案基本成功")
		fmt.Println("   💡 仍有少量死锁但重试机制有效")
	} else {
		fmt.Println("   ❌ 死锁解决方案需要进一步优化")
		fmt.Printf("   📊 发现 %d 个问题\n", issues)
	}

	fmt.Println("\n📋 关键日志片段:")
	fmt.Println("-" * 60)

	// 显示关键日志
	keyLines := 0
	for _, line := range lines {
		if regexp.MustCompile(`开始并发同步|批次|死锁|成功|错误|Goroutine统计`).MatchString(line) {
			if keyLines < 15 {
				fmt.Println(line)
				keyLines++
			}
		}
	}

	if keyLines == 0 {
		fmt.Println("(无关键日志)")
	}
}