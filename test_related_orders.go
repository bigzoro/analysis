package main

import (
	"fmt"
	"strconv"
	"strings"
)

// 模拟修复后的getRelatedOrdersSummary中的CloseOrderIds解析逻辑
func parseCloseOrderIds(closeOrderIds string) []uint {
	var ids []uint
	if closeOrderIds == "" {
		return ids
	}

	// 处理逗号分隔的格式，如"1450"或"1450,1451"
	closeOrderIdsStr := strings.TrimSpace(closeOrderIds)

	if closeOrderIdsStr == "" {
		return ids
	}

	// 按逗号分割
	closeOrderIdsArr := strings.Split(closeOrderIdsStr, ",")
	fmt.Printf("   分割后: %v\n", closeOrderIdsArr)

	for _, idStr := range closeOrderIdsArr {
		idStr = strings.TrimSpace(idStr)
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			ids = append(ids, uint(id))
			fmt.Printf("   解析成功: '%s' -> %d\n", idStr, id)
		} else {
			fmt.Printf("   解析失败: '%s' -> %v\n", idStr, err)
		}
	}

	return ids
}

func main() {
	fmt.Println("🧪 测试CloseOrderIds解析和related_orders逻辑")

	// 测试不同的CloseOrderIds格式（修复后的格式）
	testCases := []struct {
		closeOrderIds string
		description   string
	}{
		{"1450", "单个ID"},
		{"1450,1451", "多个ID"},
		{"", "空字符串"},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📋 测试: %s - '%s'\n", tc.description, tc.closeOrderIds)

		// 模拟解析过程
		closeOrderIds := parseCloseOrderIds(tc.closeOrderIds)

		// 模拟related_orders构建
		result := map[string]interface{}{
			"has_parent":   false,
			"has_close":    false,
			"parent_count": 0,
			"close_count":  0,
			"trade_chain":  "",
		}

		if len(closeOrderIds) > 0 {
			result["has_close"] = true
			result["close_count"] = len(closeOrderIds)
			result["close_ids"] = closeOrderIds
			fmt.Printf("   ✅ 设置: has_close=true, close_count=%d\n", len(closeOrderIds))
		} else {
			fmt.Printf("   ❌ 无关联订单\n")
		}

		// 模拟前端判断逻辑
		orderStatus := "filled"
		isReduceOnly := false
		hasClose := result["has_close"].(bool)
		closeCount := result["close_count"].(int)

		shouldShowEnded := false
		if (orderStatus == "filled" || orderStatus == "completed") && !isReduceOnly {
			if hasClose && closeCount > 0 {
				shouldShowEnded = true
			}
		}

		if shouldShowEnded {
			fmt.Printf("   🎯 前端显示: '已结束'\n")
		} else {
			fmt.Printf("   🎯 前端显示: '已成交'\n")
		}
	}

	fmt.Println("\n🎯 关键发现:")
	fmt.Println("要显示'已结束'，需要满足以下条件:")
	fmt.Println("1. order.status ∈ ['filled', 'completed']")
	fmt.Println("2. order.reduce_only = false")
	fmt.Println("3. order.related_orders.has_close = true")
	fmt.Println("4. order.related_orders.close_count > 0")
	fmt.Println("")
	fmt.Println("如果BTRUSDT显示'已成交'而不是'已结束'，说明上述条件之一不满足")
}