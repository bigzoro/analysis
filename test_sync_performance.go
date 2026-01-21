package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	fmt.Println("🚀 K线同步性能测试")
	fmt.Println("=" * 50)

	// 启动同步服务进行测试
	fmt.Println("1. 启动数据同步服务...")

	// 设置超时时间为5分钟
	timeout := 5 * time.Minute
	startTime := time.Now()

	// 运行同步命令
	cmd := exec.Command("go", "run", "./analysis_backend/cmd/data_sync/main.go", "sync", "klines", "--market", "spot", "--interval", "1m")
	cmd.Dir = "d:\\code\\analysis2"

	// 获取命令输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("命令执行出错: %v", err)
	}

	executionTime := time.Since(startTime)

	fmt.Printf("2. 同步执行结果:\n")
	fmt.Printf("   ⏱️ 执行时间: %v\n", executionTime.Round(time.Second))

	outputStr := string(output)
	fmt.Printf("   📄 输出长度: %d 字符\n", len(outputStr))

	// 分析输出结果
	fmt.Println("3. 性能分析:")

	// 查找goroutine统计
	goroutineRegex := regexp.MustCompile(`Goroutine统计 - 开始:(\d+), 结束:(\d+), 差异:([+-]?\d+)`)
	if matches := goroutineRegex.FindStringSubmatch(outputStr); len(matches) > 3 {
		start, _ := strconv.Atoi(matches[1])
		end, _ := strconv.Atoi(matches[2])
		diff, _ := strconv.Atoi(matches[3])

		fmt.Printf("   🔄 Goroutine统计: 开始%d → 结束%d (差异:%+d)\n", start, end, diff)

		if diff > 50 {
			fmt.Printf("   ⚠️ Goroutine泄漏警告: 增加了%d个goroutine\n", diff)
		} else {
			fmt.Printf("   ✅ Goroutine状态正常\n")
		}
	}

	// 查找同步统计
	syncRegex := regexp.MustCompile(`同步统计 - 成功率:([\d.]+)% \| 用时:([^\|]+)`)
	if matches := syncRegex.FindStringSubmatch(outputStr); len(matches) > 2 {
		successRate := matches[1]
		duration := matches[2]

		fmt.Printf("   📊 同步成功率: %s%%\n", successRate)
		fmt.Printf("   ⏱️ 同步总用时: %s\n", duration)

		// 评估性能
		if strings.Contains(successRate, "100.0") {
			fmt.Printf("   ✅ 同步完全成功!\n")
		} else {
			fmt.Printf("   ⚠️ 同步有失败，成功率: %s%%\n", successRate)
		}
	}

	// 查找批次处理信息
	batchRegex := regexp.MustCompile(`处理批次 (\d+)/(\d+)`)
	batches := batchRegex.FindAllString(outputStr, -1)
	if len(batches) > 0 {
		fmt.Printf("   📦 批次处理: %d 个批次\n", len(batches))
	}

	// 查找死锁信息
	if strings.Contains(outputStr, "死锁") {
		fmt.Printf("   ❌ 检测到死锁错误!\n")

		deadlockCount := strings.Count(outputStr, "死锁")
		fmt.Printf("   🔴 死锁次数: %d\n", deadlockCount)
	} else {
		fmt.Printf("   ✅ 未检测到死锁\n")
	}

	// 查找API限流信息
	if strings.Contains(outputStr, "rate limited") || strings.Contains(outputStr, "API限流") {
		fmt.Printf("   ⚠️ 检测到API限流\n")
	} else {
		fmt.Printf("   ✅ 未检测到API限流\n")
	}

	// 性能评估
	fmt.Println("4. 总体评估:")

	if executionTime < timeout {
		fmt.Printf("   ✅ 在规定时间内完成 (<%v)\n", timeout)
	} else {
		fmt.Printf("   ❌ 超时! 超过%v\n", timeout)
	}

	if strings.Contains(outputStr, "同步完成") {
		fmt.Printf("   ✅ 同步过程正常结束\n")
	} else {
		fmt.Printf("   ❌ 同步未正常完成\n")
	}

	fmt.Println("\n📋 详细日志:")
	fmt.Println(strings.Repeat("-", 50))

	// 只显示关键日志行
	lines := strings.Split(outputStr, "\n")
	keyLines := 0

	for _, line := range lines {
		// 显示重要信息
		if strings.Contains(line, "[KlineSyncer]") &&
		   (strings.Contains(line, "开始") ||
		    strings.Contains(line, "完成") ||
		    strings.Contains(line, "错误") ||
		    strings.Contains(line, "死锁") ||
		    strings.Contains(line, "Goroutine") ||
		    strings.Contains(line, "批次")) {
			fmt.Println(line)
			keyLines++
			if keyLines > 20 { // 限制输出行数
				fmt.Println("... (日志过长，已截断)")
				break
			}
		}
	}
}