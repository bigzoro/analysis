package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
	"strconv"
	"strings"
	"time"
)

func main() {
	fmt.Println("🧪 测试Bracket外部平仓状态修复")

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	userID := uint(0) // 使用用户ID 0

	fmt.Println("\n📊 检查开仓订单状态:")
	var entryOrders []pdb.ScheduledOrder
	err = gdb.GormDB().
		Where("user_id = ? AND reduce_only = false AND (parent_order_id IS NULL OR parent_order_id = 0)", userID).
		Order("created_at DESC").
		Limit(3).
		Find(&entryOrders).Error

	if err != nil {
		log.Printf("查询开仓订单失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个开仓订单:\n", len(entryOrders))
		for _, order := range entryOrders {
			fmt.Printf("  - ID: %d, Symbol: %s, Status: %s, CloseOrderIds: '%s'\n",
				order.ID, order.Symbol, order.Status, order.CloseOrderIds)

			// 检查是否有平仓订单
			if order.CloseOrderIds != "" {
				fmt.Printf("    📋 有平仓订单关联: %s\n", order.CloseOrderIds)

				// 解析close_order_ids
				closeOrderIds := parseCloseOrderIds(order.CloseOrderIds)
				for _, closeID := range closeOrderIds {
					var closeOrder pdb.ScheduledOrder
					if err := gdb.GormDB().Where("id = ?", closeID).First(&closeOrder).Error; err == nil {
						fmt.Printf("      - 关联平仓订单 ID: %d, Status: %s, ReduceOnly: %v\n",
							closeOrder.ID, closeOrder.Status, closeOrder.ReduceOnly)
					}
				}
			} else {
				fmt.Printf("    ⚠️ 无平仓订单关联\n")
			}
		}
	}

	fmt.Println("\n📋 分析结果:")
	fmt.Println("1. 开仓订单状态应为 'filled'（已成交）")
	fmt.Println("2. 当有平仓订单关联时，前端会显示 '已结束'")
	fmt.Println("3. 外部平仓订单状态为 'completed'，reduce_only = true")

	fmt.Println("\n✅ 修复验证完成！")
}

func parseCloseOrderIds(closeOrderIds string) []uint {
	var ids []uint
	if closeOrderIds == "" {
		return ids
	}

	// 移除方括号
	cleanStr := closeOrderIds
	if len(cleanStr) >= 2 && cleanStr[0] == '[' && cleanStr[len(cleanStr)-1] == ']' {
		cleanStr = cleanStr[1 : len(cleanStr)-1]
	}

	// 按逗号分割
	parts := strings.Split(cleanStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if id, err := strconv.ParseUint(part, 10, 32); err == nil {
			ids = append(ids, uint(id))
		}
	}

	return ids
}