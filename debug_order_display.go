package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("=== 调试订单显示问题 ===\n")

	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
		return
	}
	defer gdb.Close()

	// 查询最近的订单，查看ParentOrderId字段
	fmt.Println("1. 检查订单的ParentOrderId字段：")
	var orders []struct {
		ID            uint   `json:"id"`
		Symbol        string `json:"symbol"`
		Side          string `json:"side"`
		ReduceOnly    bool   `json:"reduce_only"`
		ParentOrderId uint   `json:"parent_order_id"`
		ClientOrderId string `json:"client_order_id"`
		Status        string `json:"status"`
	}

	err = gdb.DB().Table("scheduled_orders").
		Where("user_id = ? AND symbol = ?", 1, "SCRTUSDT").
		Order("created_at DESC").
		Limit(10).
		Select("id, symbol, side, reduce_only, parent_order_id, client_order_id, status").
		Find(&orders).Error

	if err != nil {
		log.Printf("查询订单失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个SCRTUSDT订单:\n", len(orders))
	for _, order := range orders {
		parentInfo := "无"
		if order.ParentOrderId > 0 {
			parentInfo = fmt.Sprintf("父订单%d", order.ParentOrderId)
		}

		orderType := "开仓"
		if order.ReduceOnly {
			orderType = "平仓"
		}
		if order.ClientOrderId != "" && contains(order.ClientOrderId, "PROFIT_SCALING") {
			orderType = "加仓"
		}

		fmt.Printf("  ID:%d %s %s → %s (%s)\n",
			order.ID, order.Symbol, order.Side, orderType, parentInfo)
	}

	fmt.Println("\n2. 检查是否有开仓订单的子订单：")

	// 查找开仓订单
	var openOrders []struct {
		ID     uint
		Symbol string
	}

	err = gdb.DB().Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND reduce_only = false AND status IN (?)",
			1, "SCRTUSDT", []string{"pending", "processing", "filled", "completed"}).
		Select("id, symbol").
		Find(&openOrders).Error

	if err != nil {
		log.Printf("查询开仓订单失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个开仓订单:\n", len(openOrders))
	for _, openOrder := range openOrders {
		// 检查平仓订单
		var closeCount int64
		gdb.DB().Table("scheduled_orders").
			Where("parent_order_id = ? AND reduce_only = true", openOrder.ID).
			Count(&closeCount)

		// 检查加仓订单
		var scalingCount int64
		gdb.DB().Table("scheduled_orders").
			Where("parent_order_id = ? AND reduce_only = false", openOrder.ID).
			Count(&scalingCount)

		fmt.Printf("  开仓订单 %d: %d个平仓订单, %d个加仓订单\n",
			openOrder.ID, closeCount, scalingCount)

		if scalingCount > 0 {
			var scalingOrders []struct {
				ID            uint
				ClientOrderId string
			}
			gdb.DB().Table("scheduled_orders").
				Where("parent_order_id = ? AND reduce_only = false", openOrder.ID).
				Select("id, client_order_id").
				Find(&scalingOrders)

			for _, so := range scalingOrders {
				fmt.Printf("    ├── 加仓订单 %d (%s)\n", so.ID, so.ClientOrderId)
			}
		}
	}

	fmt.Println("\n3. 模拟后端API返回的数据结构：")

	// 模拟一个开仓订单的数据结构
	mockOpenOrder := map[string]interface{}{
		"id":          100,
		"symbol":      "SCRTUSDT",
		"reduce_only": false,
		"related_orders": map[string]interface{}{
			"has_close":      true,
			"close_count":    1,
			"close_ids":      []uint{101},
			"has_scaling":    true,
			"scaling_count":  2,
			"scaling_ids":    []uint{102, 103},
			"has_children":   true,
			"children_count": 3,
			"children_ids":   []uint{101, 102, 103},
			"trade_chain":    "交易链 #100",
		},
	}

	fmt.Printf("开仓订单数据结构:\n")
	fmt.Printf("  ID: %v\n", mockOpenOrder["id"])
	fmt.Printf("  Symbol: %v\n", mockOpenOrder["symbol"])
	fmt.Printf("  ReduceOnly: %v\n", mockOpenOrder["reduce_only"])
	fmt.Printf("  RelatedOrders: %+v\n", mockOpenOrder["related_orders"])

	fmt.Println("\n4. 模拟前端处理逻辑：")

	// 模拟前端的computed逻辑
	order := mockOpenOrder
	childOrders := []interface{}{}

	// 模拟平仓订单查找
	if relatedOrders, ok := order["related_orders"].(map[string]interface{}); ok {
		if hasClose, ok := relatedOrders["has_close"].(bool); ok && hasClose {
			if closeIds, ok := relatedOrders["close_ids"].([]uint); ok {
				fmt.Printf("找到 %d 个平仓订单ID: %v\n", len(closeIds), closeIds)
				// 模拟找到的平仓订单
				childOrders = append(childOrders, map[string]interface{}{
					"id": 101, "operation_type": "平空", "reduce_only": true,
				})
			}
		}

		// 模拟加仓订单查找
		if hasScaling, ok := relatedOrders["has_scaling"].(bool); ok && hasScaling {
			if scalingIds, ok := relatedOrders["scaling_ids"].([]uint); ok {
				fmt.Printf("找到 %d 个加仓订单ID: %v\n", len(scalingIds), scalingIds)
				// 模拟找到的加仓订单
				childOrders = append(childOrders, map[string]interface{}{
					"id": 102, "operation_type": "开空", "reduce_only": false,
				})
				childOrders = append(childOrders, map[string]interface{}{
					"id": 103, "operation_type": "开空", "reduce_only": false,
				})
			}
		}
	}

	fmt.Printf("最终子订单数量: %d\n", len(childOrders))
	for _, child := range childOrders {
		if childMap, ok := child.(map[string]interface{}); ok {
			fmt.Printf("  ├── 子订单 %v: %v\n",
				childMap["id"], childMap["operation_type"])
		}
	}

	fmt.Println("\n5. 问题排查建议：")

	fmt.Printf("如果加仓订单仍显示为独立卡片，请检查：\n")
	fmt.Printf("1. 后端API是否返回了正确的has_scaling和scaling_ids字段\n")
	fmt.Printf("2. 前端computed属性中的条件判断是否正确\n")
	fmt.Printf("3. 订单的ParentOrderId字段是否正确设置\n")
	fmt.Printf("4. 前端的childOrders数组是否正确填充\n")
	fmt.Printf("\n")

	fmt.Printf("调试步骤：\n")
	fmt.Printf("1. 在浏览器开发者工具中检查API响应数据\n")
	fmt.Printf("2. 在前端添加console.log调试computed逻辑\n")
	fmt.Printf("3. 检查数据库中订单的parent_order_id字段值\n")
	fmt.Printf("4. 验证后端getRelatedOrdersSummary函数的返回值\n")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || contains(s[1:], substr) || contains(s[:len(s)-1], substr))
}
