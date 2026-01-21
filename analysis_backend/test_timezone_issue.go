package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 时区时间逻辑测试 ===")

	// 模拟CST时区 (UTC+8)，即使当前环境是UTC，我们也要测试时区逻辑
	cst := time.FixedZone("CST", 8*60*60)
	now := time.Now().In(cst)  // 强制转换为CST时区
	nowUTC := time.Now().UTC()

	fmt.Printf("当前本地时间 (CST): %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("当前UTC时间: %s\n", nowUTC.Format("2006-01-02 15:04:05"))
	fmt.Printf("时区偏移: %d小时\n", now.Sub(nowUTC)/time.Hour)

	// 如果偏移为0，强制设置一个偏移来演示问题
	if now.Sub(nowUTC) == 0 {
		fmt.Println("\n⚠️ 当前环境时区偏移为0，强制模拟CST时区问题:")
		now = nowUTC.Add(8 * time.Hour).In(cst)  // 模拟CST时间
		fmt.Printf("模拟CST时间: %s\n", now.Format("2006-01-02 15:04:05"))
		fmt.Printf("实际UTC时间: %s\n", nowUTC.Format("2006-01-02 15:04:05"))
		fmt.Printf("模拟时区偏移: %d小时\n", now.Sub(nowUTC)/time.Hour)
	}

	// 测试24小时前的计算
	timeRange := 24 * time.Hour

	// 错误的计算方式（当前代码）
	cutoffTimeWrong := now.Add(-timeRange)
	fmt.Printf("\n❌ 错误的计算 (使用本地时间):\n")
	fmt.Printf("   cutoffTime = time.Now().Add(-24h)\n")
	fmt.Printf("   cutoffTime = %s\n", cutoffTimeWrong.Format("2006-01-02 15:04:05"))
	fmt.Printf("   相当于UTC: %s\n", cutoffTimeWrong.UTC().Format("2006-01-02 15:04:05"))

	// 正确的计算方式
	cutoffTimeCorrect := nowUTC.Add(-timeRange)
	fmt.Printf("\n✅ 正确的计算 (使用UTC时间):\n")
	fmt.Printf("   cutoffTime = time.Now().UTC().Add(-24h)\n")
	fmt.Printf("   cutoffTime = %s (UTC)\n", cutoffTimeCorrect.Format("2006-01-02 15:04:05"))

	// 计算差异
	diff := cutoffTimeWrong.Sub(cutoffTimeCorrect)
	fmt.Printf("\n🔍 时间差异: %v\n", diff)
	fmt.Printf("   错误的计算会多查询 %d 小时的记录\n", int(diff.Hours()))

	// 实际查询示例
	fmt.Printf("\n📊 查询范围对比:\n")
	fmt.Printf("❌ 错误查询: created_at >= '%s' (实际查询过去%.1f小时)\n",
		cutoffTimeWrong.Format("2006-01-02 15:04:05"), now.Sub(cutoffTimeWrong).Hours())
	fmt.Printf("✅ 正确查询: created_at >= '%s' (实际查询过去%.1f小时)\n",
		cutoffTimeCorrect.Format("2006-01-02 15:04:05"), now.Sub(cutoffTimeCorrect).Hours())

	// 模拟数据库中的记录时间
	fmt.Printf("\n🗄️ 数据库记录时间示例:\n")
	sampleDBTime := time.Date(2026, 1, 20, 1, 30, 0, 0, time.UTC)
	fmt.Printf("   数据库记录时间: %s (UTC)\n", sampleDBTime.Format("2006-01-02 15:04:05"))

	wouldMatchWrong := sampleDBTime.After(cutoffTimeWrong.UTC()) || sampleDBTime.Equal(cutoffTimeWrong.UTC())
	wouldMatchCorrect := sampleDBTime.After(cutoffTimeCorrect) || sampleDBTime.Equal(cutoffTimeCorrect)

	fmt.Printf("   ❌ 错误逻辑匹配: %v\n", wouldMatchWrong)
	fmt.Printf("   ✅ 正确逻辑匹配: %v\n", wouldMatchCorrect)

	fmt.Printf("\n🎯 结论:\n")
	fmt.Printf("   使用本地时间计算cutoffTime会导致查询范围扩大%d小时\n", int(diff.Hours()))
	fmt.Printf("   应该使用UTC时间进行计算以确保时区一致性\n")
}