package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 检查所有Client Order ID格式的长度 ===\n")

	// 测试数据
	orderID := uint(999999)        // 6位数
	timestamp := time.Now().Unix() // 10位数

	// 1. PROFIT_SCALING格式
	profitScalingID := fmt.Sprintf("PROFIT_SCALING_%d_%d", orderID, timestamp)
	fmt.Printf("PROFIT_SCALING: %s (长度: %d)\n", profitScalingID, len(profitScalingID))

	// 2. OC_格式（整体平仓）
	shortReasons := []string{"STOP_LOSS", "TAKE_PROFIT", "STOP_ALL"}
	for _, reason := range shortReasons {
		ocID := fmt.Sprintf("OC_%s_%d_%d", reason, orderID, timestamp)
		fmt.Printf("OC_%s: %s (长度: %d)\n", reason, ocID, len(ocID))
	}

	// 3. external_close格式
	externalOpID := uint(888888)
	externalCloseID := fmt.Sprintf("external_close_%d_%d", orderID, externalOpID)
	fmt.Printf("external_close: %s (长度: %d)\n", externalCloseID, len(externalCloseID))

	// 4. sch-格式
	schID := fmt.Sprintf("sch-%d-%d", orderID, timestamp)
	fmt.Printf("sch: %s (长度: %d)\n", schID, len(schID))

	schEntryID := fmt.Sprintf("sch-%d-%s-%d", orderID, "entry", timestamp)
	fmt.Printf("sch-entry: %s (长度: %d)\n", schEntryID, len(schEntryID))

	// 5. 检查边界情况 - 更大的数字
	maxOrderID := uint(9999999999)    // 10位数
	maxTimestamp := int64(9999999999) // 10位数（未来的时间戳）

	fmt.Printf("\n边界情况测试 (最大值):\n")
	maxProfitScalingID := fmt.Sprintf("PROFIT_SCALING_%d_%d", maxOrderID, maxTimestamp)
	fmt.Printf("Max PROFIT_SCALING: %s (长度: %d)\n", maxProfitScalingID, len(maxProfitScalingID))

	maxOCID := fmt.Sprintf("OC_%s_%d_%d", "STOP_LOSS", maxOrderID, maxTimestamp)
	fmt.Printf("Max OC_STOP_LOSS: %s (长度: %d)\n", maxOCID, len(maxOCID))

	maxExternalCloseID := fmt.Sprintf("external_close_%d_%d", maxOrderID, maxOrderID)
	fmt.Printf("Max external_close: %s (长度: %d)\n", maxExternalCloseID, len(maxExternalCloseID))

	maxSchID := fmt.Sprintf("sch-%d-%d", maxOrderID, maxTimestamp)
	fmt.Printf("Max sch: %s (长度: %d)\n", maxSchID, len(maxSchID))

	fmt.Printf("\n=== 长度检查结果 (36字符限制) ===\n")
	maxLimit := 36
	checkLimit := func(name, id string) {
		if len(id) <= maxLimit {
			fmt.Printf("✅ %s: %d字符 - 符合要求\n", name, len(id))
		} else {
			fmt.Printf("❌ %s: %d字符 - 超过限制 %d字符\n", name, len(id), len(id)-maxLimit)
		}
	}

	checkLimit("PROFIT_SCALING", profitScalingID)
	for _, reason := range shortReasons {
		id := fmt.Sprintf("OC_%s_%d_%d", reason, orderID, timestamp)
		checkLimit(fmt.Sprintf("OC_%s", reason), id)
	}
	checkLimit("external_close", externalCloseID)
	checkLimit("sch", schID)
	checkLimit("sch-entry", schEntryID)
	checkLimit("Max PROFIT_SCALING", maxProfitScalingID)
	checkLimit("Max OC_STOP_LOSS", maxOCID)
	checkLimit("Max external_close", maxExternalCloseID)
	checkLimit("Max sch", maxSchID)

	fmt.Printf("\n=== 总结 ===\n")
	fmt.Printf("1. 所有当前格式都在36字符限制内\n")
	fmt.Printf("2. 边界情况下PROFIT_SCALING可能接近或达到限制\n")
	fmt.Printf("3. external_close在边界情况下可能超过限制\n")
	fmt.Printf("4. 需要监控orderID和timestamp的位数增长\n")
}
