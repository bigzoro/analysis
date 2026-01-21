package main

import (
	"fmt"
	"time"
)

func main() {
	// 模拟配置中的值
	priceSyncInterval := 0.5 // 0.5分钟 = 30秒

	// 错误的转换方式（会导致0）
	wrongInterval := time.Duration(priceSyncInterval) * time.Minute
	fmt.Printf("错误的转换: %v (duration: %d)\n", wrongInterval, int64(wrongInterval))

	// 正确的转换方式
	correctInterval := time.Duration(priceSyncInterval * 60) * time.Second
	fmt.Printf("正确的转换: %v (duration: %d)\n", correctInterval, int64(correctInterval))

	// 测试NewTicker
	fmt.Println("\n测试NewTicker:")
	if wrongInterval == 0 {
		fmt.Println("❌ 错误的interval为0，会导致panic")
	} else {
		fmt.Println("✅ 错误的interval不为0")
	}

	if correctInterval > 0 {
		fmt.Println("✅ 正确的interval大于0，可以正常使用")
	} else {
		fmt.Println("❌ 正确的interval为0")
	}
}
