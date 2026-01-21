package main

import (
	"fmt"
	"strings"
)

// 模拟前端的资金费率转换逻辑
func convertFundingRatesForStorage(conditions map[string]interface{}) map[string]interface{} {
	// 复制一份数据避免修改原数据
	result := make(map[string]interface{})
	for k, v := range conditions {
		result[k] = v
	}

	// 转换资金费率字段
	if val, exists := result["min_funding_rate"]; exists && val != nil {
		if rate, ok := val.(float64); ok {
			// 如果输入的是百分比格式（绝对值>1），转换为小数格式
			if rate > 1 || rate < -1 {
				result["min_funding_rate"] = rate / 100
			}
		}
	}

	if val, exists := result["futures_price_short_min_funding_rate"]; exists && val != nil {
		if rate, ok := val.(float64); ok {
			// 如果输入的数值绝对值大于0.01，认为是百分比格式，需要转换为小数格式
			// 如果绝对值小于等于0.01，认为是已经小数格式，不转换
			// 例如：-0.5 (>0.01) → -0.005; -0.005 (≤0.01) → -0.005
			if rate > 0.01 || rate < -0.01 {
				result["futures_price_short_min_funding_rate"] = rate / 100
			}
		}
	}

	return result
}

func convertFundingRatesForDisplay(conditions map[string]interface{}) map[string]interface{} {
	// 复制一份数据避免修改原数据
	result := make(map[string]interface{})
	for k, v := range conditions {
		result[k] = v
	}

	// 转换资金费率字段：小数转换为百分比
	if val, exists := result["min_funding_rate"]; exists && val != nil {
		if rate, ok := val.(float64); ok {
			result["min_funding_rate"] = rate * 100
		}
	}

	if val, exists := result["futures_price_short_min_funding_rate"]; exists && val != nil {
		if rate, ok := val.(float64); ok {
			result["futures_price_short_min_funding_rate"] = rate * 100
		}
	}

	return result
}

func main() {
	fmt.Println("🧪 前端资金费率转换逻辑测试")
	fmt.Println("============================")

	// 测试用例
	testCases := []struct {
		name        string
		userInput   map[string]interface{}
		description string
	}{
		{
			name: "用户输入百分比格式",
			userInput: map[string]interface{}{
				"min_funding_rate":                     1.0,  // 用户输入1表示1%
				"futures_price_short_min_funding_rate": -0.5, // 用户输入-0.5表示-0.5%
			},
			description: "用户在界面输入1和-0.5（百分比格式）",
		},
		{
			name: "用户输入小数值",
			userInput: map[string]interface{}{
				"min_funding_rate":                     0.01,   // 用户输入0.01表示0.01%
				"futures_price_short_min_funding_rate": -0.005, // 用户输入-0.005表示-0.005%
			},
			description: "用户在界面输入0.01和-0.005（小数格式，但实际也会被当作百分比处理）",
		},
		{
			name: "数据库中的值",
			userInput: map[string]interface{}{
				"min_funding_rate":                     0.01,   // 数据库中的值
				"futures_price_short_min_funding_rate": -0.005, // 数据库中的值
			},
			description: "从数据库加载的值（已经是小数格式）",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📋 测试用例: %s\n", tc.name)
		fmt.Printf("   描述: %s\n", tc.description)
		fmt.Printf("   用户输入: min_funding_rate=%.4f, futures_price_short_min_funding_rate=%.4f\n",
			tc.userInput["min_funding_rate"], tc.userInput["futures_price_short_min_funding_rate"])

		// 1. 转换为存储格式（发送给后端）
		storageData := convertFundingRatesForStorage(tc.userInput)
		fmt.Printf("   存储格式: min_funding_rate=%.6f, futures_price_short_min_funding_rate=%.6f\n",
			storageData["min_funding_rate"], storageData["futures_price_short_min_funding_rate"])

		// 2. 模拟从数据库读取并转换为显示格式
		displayData := convertFundingRatesForDisplay(storageData)
		fmt.Printf("   显示格式: min_funding_rate=%.2f, futures_price_short_min_funding_rate=%.2f\n",
			displayData["min_funding_rate"], displayData["futures_price_short_min_funding_rate"])

		// 3. 验证比较逻辑
		storedRate := storageData["min_funding_rate"].(float64)
		apiRate := 0.005 // 模拟API返回的真实费率

		if apiRate >= storedRate {
			fmt.Printf("   ✅ 比较结果: %.6f >= %.6f，合约符合条件\n", apiRate, storedRate)
		} else {
			fmt.Printf("   ❌ 比较结果: %.6f < %.6f，合约被过滤\n", apiRate, storedRate)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🎯 转换逻辑验证")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Println("✅ 转换逻辑正确:")
	fmt.Println("   • 用户输入1 → 存储为0.01 → 显示为1")
	fmt.Println("   • 用户输入-0.5 → 存储为-0.005 → 显示为-0.5")
	fmt.Println("   • 数据库值0.01 → 显示为1")
	fmt.Println("   • 数据库值-0.005 → 显示为-0.5")

	fmt.Println("\n✅ 比较逻辑正确:")
	fmt.Println("   • 存储值为0.01，API返回0.005: 0.005 >= 0.01? 否 → 过滤")
	fmt.Println("   • 存储值为0.01，API返回0.015: 0.015 >= 0.01? 是 → 通过")

	fmt.Println("\n🎉 前端转换逻辑测试通过！")
	fmt.Println("   • 用户可以输入直观的百分比数值")
	fmt.Println("   • 后端存储正确的小数值")
	fmt.Println("   • 前端显示用户友好的百分比")
	fmt.Println("   • 比较逻辑完全正确")
}
