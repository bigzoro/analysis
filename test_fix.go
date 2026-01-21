package main

import (
	"fmt"
	"time"
)

// 模拟类型定义
type Conditions struct {
	ExpiryThreshold int
}

// 测试类型转换
func testTypeConversion() {
	daysToExpiry := 25.5
	conditions := Conditions{ExpiryThreshold: 30}

	if daysToExpiry > float64(conditions.ExpiryThreshold) {
		fmt.Printf("✅ 类型转换正确: %.1f > %d\n", daysToExpiry, conditions.ExpiryThreshold)
	} else {
		fmt.Printf("❌ 类型转换错误: %.1f <= %d\n", daysToExpiry, conditions.ExpiryThreshold)
	}
}

func main() {
	testTypeConversion()
	fmt.Println("类型转换测试完成")
}
