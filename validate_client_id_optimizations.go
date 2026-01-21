package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 验证Client Order ID优化结果 ===\n")

	// 模拟safeTimestamp函数
	safeTimestamp := func() int64 {
		ts := time.Now().Unix()
		if ts > 999999999 {
			ts = ts % 1000000000
		}
		return ts
	}

	// 测试数据 - 使用最大可能值
	maxOrderID := uint(9999999999)    // 10位数
	maxTimestamp := int64(9999999999) // 10位数（但会被safeTimestamp限制为9位）
	safeTS := safeTimestamp()

	fmt.Printf("测试参数:\n")
	fmt.Printf("  maxOrderID: %d\n", maxOrderID)
	fmt.Printf("  maxTimestamp: %d\n", maxTimestamp)
	fmt.Printf("  safeTimestamp: %d\n", safeTS)
	fmt.Printf("  safeTimestamp长度: %d位\n", len(fmt.Sprintf("%d", safeTS)))

	fmt.Printf("\n=== 优化后的格式测试 ===\n")

	// 1. PS_格式（原来的PROFIT_SCALING）
	psID := fmt.Sprintf("PS_%d_%d", maxOrderID%10000000, safeTS)
	fmt.Printf("PS格式: %s (长度: %d)\n", psID, len(psID))

	// 2. OC_格式
	shortReasons := []string{"STOP_LOSS", "TAKE_PROFIT", "STOP_ALL"}
	for _, reason := range shortReasons {
		ocID := fmt.Sprintf("OC_%s_%d_%d", reason, maxOrderID%10000000, safeTS)
		fmt.Printf("OC_%s: %s (长度: %d)\n", reason, ocID, len(ocID))
	}

	// 3. EC_格式（原来的external_close）
	ecID := fmt.Sprintf("EC_%d_%d", maxOrderID%10000000, maxOrderID%1000000)
	fmt.Printf("EC格式: %s (长度: %d)\n", ecID, len(ecID))

	// 4. sch-格式
	schID := fmt.Sprintf("sch-%d-%d", maxOrderID%10000000, safeTS)
	fmt.Printf("sch格式: %s (长度: %d)\n", schID, len(schID))

	schEntryID := fmt.Sprintf("sch-%d-%s-%d", maxOrderID%10000000, "entry", safeTS)
	fmt.Printf("sch-entry: %s (长度: %d)\n", schEntryID, len(schEntryID))

	fmt.Printf("\n=== 长度验证 (36字符限制) ===\n")
	maxLimit := 36

	testCases := []struct {
		name string
		id   string
	}{
		{"PS格式", psID},
		{"OC_STOP_LOSS", fmt.Sprintf("OC_%s_%d_%d", "STOP_LOSS", maxOrderID%10000000, safeTS)},
		{"OC_TAKE_PROFIT", fmt.Sprintf("OC_%s_%d_%d", "TAKE_PROFIT", maxOrderID%10000000, safeTS)},
		{"OC_STOP_ALL", fmt.Sprintf("OC_%s_%d_%d", "STOP_ALL", maxOrderID%10000000, safeTS)},
		{"EC格式", ecID},
		{"sch格式", schID},
		{"sch-entry", schEntryID},
	}

	allPassed := true
	for _, tc := range testCases {
		if len(tc.id) <= maxLimit {
			fmt.Printf("✅ %s: %d字符 - 符合要求\n", tc.name, len(tc.id))
		} else {
			fmt.Printf("❌ %s: %d字符 - 超过限制 %d字符\n", tc.name, len(tc.id), len(tc.id)-maxLimit)
			allPassed = false
		}
	}

	fmt.Printf("\n=== 优化对比 ===\n")
	fmt.Printf("修改前可能的问题格式:\n")
	oldPS := fmt.Sprintf("PROFIT_SCALING_%d_%d", maxOrderID, maxTimestamp)
	oldOC := fmt.Sprintf("OVERALL_CLOSE_整体止损_%d_%d", maxOrderID, maxTimestamp)
	oldEC := fmt.Sprintf("external_close_%d_%d", maxOrderID, maxOrderID)

	fmt.Printf("  PROFIT_SCALING: %s (%d字符) ❌\n", oldPS, len(oldPS))
	fmt.Printf("  OVERALL_CLOSE: %s (%d字符) ❌\n", oldOC, len(oldOC))
	fmt.Printf("  external_close: %s (%d字符) ⚠️\n", oldEC, len(oldEC))

	fmt.Printf("\n修改后的安全格式:\n")
	fmt.Printf("  PS_: %s (%d字符) ✅\n", psID, len(psID))
	fmt.Printf("  OC_: %s (%d字符) ✅\n", fmt.Sprintf("OC_STOP_LOSS_%d_%d", maxOrderID%10000000, safeTS), len(fmt.Sprintf("OC_STOP_LOSS_%d_%d", maxOrderID%10000000, safeTS)))
	fmt.Printf("  EC_: %s (%d字符) ✅\n", ecID, len(ecID))

	if allPassed {
		fmt.Printf("\n🎉 所有Client Order ID格式都已优化完成，永不超过36字符限制！\n")
	} else {
		fmt.Printf("\n⚠️ 还有格式需要进一步优化\n")
	}
}
