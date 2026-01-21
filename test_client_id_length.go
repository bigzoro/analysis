package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 测试Client Order ID长度 ===\n")

	// 旧格式（有问题的）
	oldFormat1 := fmt.Sprintf("OVERALL_CLOSE_%s_%d_%d", "整体止损", 966, time.Now().Unix())
	oldFormat2 := fmt.Sprintf("OVERALL_CLOSE_%s_%d_%d", "整体止盈", 967, time.Now().Unix())

	fmt.Printf("旧格式:\n")
	fmt.Printf("  %s (长度: %d)\n", oldFormat1, len(oldFormat1))
	fmt.Printf("  %s (长度: %d)\n", oldFormat2, len(oldFormat2))

	// 新格式（修复后的）
	getShortReason := func(reason string) string {
		switch reason {
		case "整体止损":
			return "STOP_LOSS"
		case "整体止盈":
			return "TAKE_PROFIT"
		case "整体止损止盈":
			return "STOP_ALL"
		default:
			if len(reason) > 8 {
				return reason[:8]
			}
			return reason
		}
	}

	newFormat1 := fmt.Sprintf("OC_%s_%d_%d", getShortReason("整体止损"), 966, time.Now().Unix())
	newFormat2 := fmt.Sprintf("OC_%s_%d_%d", getShortReason("整体止盈"), 967, time.Now().Unix())
	newFormat3 := fmt.Sprintf("OC_%s_%d_%d", getShortReason("整体止损止盈"), 968, time.Now().Unix())

	fmt.Printf("\n新格式:\n")
	fmt.Printf("  %s (长度: %d)\n", newFormat1, len(newFormat1))
	fmt.Printf("  %s (长度: %d)\n", newFormat2, len(newFormat2))
	fmt.Printf("  %s (长度: %d)\n", newFormat3, len(newFormat3))

	// 检查是否符合36字符限制
	maxLength := 36
	fmt.Printf("\n长度检查 (最大%d字符):\n", maxLength)
	checkLength := func(id string) {
		if len(id) <= maxLength {
			fmt.Printf("  ✅ %s (长度: %d) - 符合要求\n", id, len(id))
		} else {
			fmt.Printf("  ❌ %s (长度: %d) - 超过限制\n", id, len(id))
		}
	}

	checkLength(newFormat1)
	checkLength(newFormat2)
	checkLength(newFormat3)

	fmt.Printf("\n=== 修复总结 ===\n")
	fmt.Printf("1. 将 OVERALL_CLOSE_ 前缀改为 OC_\n")
	fmt.Printf("2. 对中文reason进行映射为英文缩写\n")
	fmt.Printf("3. 新格式: OC_{SHORT_REASON}_{ORDER_ID}_{TIMESTAMP}\n")
	fmt.Printf("4. 确保总长度不超过36字符\n")
}
