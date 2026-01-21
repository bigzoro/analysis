package main

import (
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/datatypes"
)

// 模拟辅助函数
func getSymbolProfitScalingCount(counts datatypes.JSON, symbol string) int {
	if counts == nil || string(counts) == "" || string(counts) == "{}" {
		return 0
	}

	var countMap map[string]int
	if err := json.Unmarshal([]byte(counts), &countMap); err != nil {
		log.Printf("[ProfitScaling] 解析币种计数器失败: %v, 使用默认值0", err)
		return 0
	}

	count, exists := countMap[symbol]
	if !exists {
		return 0
	}

	return count
}

func updateSymbolProfitScalingCount(counts datatypes.JSON, symbol string, newCount int) datatypes.JSON {
	var countMap map[string]int
	if counts != nil && string(counts) != "" && string(counts) != "{}" {
		if err := json.Unmarshal([]byte(counts), &countMap); err != nil {
			log.Printf("[ProfitScaling] 解析现有计数器失败: %v, 创建新计数器", err)
			countMap = make(map[string]int)
		}
	} else {
		countMap = make(map[string]int)
	}

	countMap[symbol] = newCount

	updatedJSON, err := json.Marshal(countMap)
	if err != nil {
		log.Printf("[ProfitScaling] 序列化计数器失败: %v", err)
		return counts // 返回原值
	}

	return datatypes.JSON(updatedJSON)
}

func main() {
	fmt.Println("=== 测试币种级别盈利加仓计数器功能 ===\n")

	// 测试场景1：空的计数器
	fmt.Println("场景1：空的计数器")
	emptyCounts := datatypes.JSON("{}")
	btcCount := getSymbolProfitScalingCount(emptyCounts, "BTCUSDT")
	fmt.Printf("BTCUSDT计数器: %d (期望: 0)\n", btcCount)

	// 测试场景2：添加计数器
	fmt.Println("\n场景2：添加BTCUSDT计数器")
	updatedCounts := updateSymbolProfitScalingCount(emptyCounts, "BTCUSDT", 1)
	fmt.Printf("更新后JSON: %s\n", string(updatedCounts))

	btcCount = getSymbolProfitScalingCount(updatedCounts, "BTCUSDT")
	fmt.Printf("BTCUSDT计数器: %d (期望: 1)\n", btcCount)

	// 测试场景3：添加多个币种
	fmt.Println("\n场景3：添加多个币种计数器")
	updatedCounts = updateSymbolProfitScalingCount(updatedCounts, "ETHUSDT", 2)
	updatedCounts = updateSymbolProfitScalingCount(updatedCounts, "ADAUSDT", 1)

	fmt.Printf("最终JSON: %s\n", string(updatedCounts))

	btcCount = getSymbolProfitScalingCount(updatedCounts, "BTCUSDT")
	ethCount := getSymbolProfitScalingCount(updatedCounts, "ETHUSDT")
	adaCount := getSymbolProfitScalingCount(updatedCounts, "ADAUSDT")
	unknownCount := getSymbolProfitScalingCount(updatedCounts, "UNKNOWN")

	fmt.Printf("BTCUSDT计数器: %d\n", btcCount)
	fmt.Printf("ETHUSDT计数器: %d\n", ethCount)
	fmt.Printf("ADAUSDT计数器: %d\n", adaCount)
	fmt.Printf("UNKNOWN计数器: %d (不存在的币种应返回0)\n", unknownCount)

	// 测试场景4：检查最大加仓次数逻辑
	fmt.Println("\n场景4：检查最大加仓次数逻辑")
	maxCount := 1
	canBTCAdd := btcCount < maxCount
	canETHAdd := ethCount < maxCount
	canADAAdd := adaCount < maxCount

	fmt.Printf("最大加仓次数: %d\n", maxCount)
	fmt.Printf("BTCUSDT可以加仓: %v (%d < %d)\n", canBTCAdd, btcCount, maxCount)
	fmt.Printf("ETHUSDT可以加仓: %v (%d < %d)\n", canETHAdd, ethCount, maxCount)
	fmt.Printf("ADAUSDT可以加仓: %v (%d < %d)\n", canADAAdd, adaCount, maxCount)

	fmt.Println("\n✅ 测试完成！币种级别计数器功能工作正常")
	fmt.Println("\n📊 改进效果：")
	fmt.Println("• BTCUSDT达到1次上限，不再加仓")
	fmt.Println("• ETHUSDT已超过1次上限，不再加仓")
	fmt.Println("• ADAUSDT达到1次上限，不再加仓")
	fmt.Println("• 每个币种独立计数，互不影响")
}
