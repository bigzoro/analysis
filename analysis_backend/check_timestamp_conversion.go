package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 时间戳转换测试 ===")

	// 测试一些可能的时间戳值
	testTimestamps := []int64{
		1736371200000, // 2025-01-09 00:00:00 UTC (毫秒)
		1609459200000, // 2021-01-01 00:00:00 UTC (毫秒)
		1672531200000, // 2023-01-01 00:00:00 UTC (毫秒)
		1704067200000, // 2024-01-01 00:00:00 UTC (毫秒)
		1735689600000, // 2025-01-01 00:00:00 UTC (毫秒)
		1767225600000, // 2026-01-01 00:00:00 UTC (毫秒)
		1798761600000, // 2027-01-01 00:00:00 UTC (毫秒)
		1830297600000, // 2028-01-01 00:00:00 UTC (毫秒)
	}

	for _, ts := range testTimestamps {
		// 当前代码的转换方式
		convertedTime := time.Unix(ts/1000, (ts%1000)*1000000)

		// 直接转换（另一种方式）
		directTime := time.UnixMilli(ts)

		fmt.Printf("原始时间戳: %d ms\n", ts)
		fmt.Printf("  当前转换: %s\n", convertedTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("  直接转换: %s\n", directTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("  是否相同: %v\n", convertedTime.Equal(directTime))
		fmt.Println()
	}

	fmt.Println("=== 可能的错误时间戳 ===")

	// 测试一些可能的错误时间戳（可能的原因）
	errorTimestamps := []int64{
		2611020720510000, // 如果把日期当作时间戳
		2023122304450000, // 如果格式错误
		1736371200000000, // 多了一个000
	}

	for _, ts := range errorTimestamps {
		convertedTime := time.Unix(ts/1000, (ts%1000)*1000000)
		fmt.Printf("错误时间戳 %d -> %s\n", ts, convertedTime.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("\n=== 当前时间对比 ===")
	now := time.Now()
	fmt.Printf("当前时间: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("当前时间戳(秒): %d\n", now.Unix())
	fmt.Printf("当前时间戳(毫秒): %d\n", now.UnixMilli())
}