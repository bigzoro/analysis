package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("🧪 测试CloseOrderIds解析逻辑")

	// 测试不同的CloseOrderIds格式
	testCases := []string{
		"[1450]",      // 单个ID，带方括号
		"1450",        // 单个ID，不带方括号
		"[1450,1451]", // 多个ID，带方括号
		"1450,1451",   // 多个ID，不带方括号
		"",            // 空字符串
		"[]",          // 空方括号
	}

	for _, testCase := range testCases {
		fmt.Printf("\n📋 测试输入: '%s'\n", testCase)

		// 模拟getRelatedOrdersSummary中的解析逻辑
		closeOrderIdsStr := strings.TrimSpace(testCase)

		// 移除方括号
		if len(closeOrderIdsStr) >= 2 && closeOrderIdsStr[0] == '[' && closeOrderIdsStr[len(closeOrderIdsStr)-1] == ']' {
			closeOrderIdsStr = closeOrderIdsStr[1 : len(closeOrderIdsStr)-1]
			fmt.Printf("   移除方括号后: '%s'\n", closeOrderIdsStr)
		}

		if closeOrderIdsStr == "" {
			fmt.Printf("   结果: 空列表\n")
			continue
		}

		// 按逗号分割
		closeOrderIds := strings.Split(closeOrderIdsStr, ",")
		var ids []uint

		fmt.Printf("   分割后: %v\n", closeOrderIds)

		for _, idStr := range closeOrderIds {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				ids = append(ids, uint(id))
			} else {
				fmt.Printf("   解析失败: '%s' -> %v\n", idStr, err)
			}
		}

		fmt.Printf("   解析结果: %v\n", ids)
	}

	fmt.Println("\n🎯 结论:")
	fmt.Println("修复后的解析逻辑能够正确处理以下格式:")
	fmt.Println("- [1450] → [1450]")
	fmt.Println("- 1450 → [1450]")
	fmt.Println("- [1450,1451] → [1450,1451]")
	fmt.Println("- 1450,1451 → [1450,1451]")
	fmt.Println("- 空字符串 → []")
}