package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("🔍 排查BTRUSDT订单状态显示问题")

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

	// 查询BTRUSDT相关的订单
	fmt.Println("\n📊 查询BTRUSDT相关的所有订单:")
	var allOrders []pdb.ScheduledOrder
	err = gdb.GormDB().
		Where("symbol = ? AND user_id = ?", "BTRUSDT", 0).
		Order("created_at DESC").
		Find(&allOrders).Error

	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个BTRUSDT订单:\n", len(allOrders))
	for i, order := range allOrders {
		fmt.Printf("\n%d. 订单ID: %d\n", i+1, order.ID)
		fmt.Printf("   状态: %s\n", order.Status)
		fmt.Printf("   类型: %s\n", order.OrderType)
		fmt.Printf("   方向: %s\n", order.Side)
		fmt.Printf("   数量: %s\n", order.Quantity)
		fmt.Printf("   ReduceOnly: %v\n", order.ReduceOnly)
		fmt.Printf("   BracketEnabled: %v\n", order.BracketEnabled)
		fmt.Printf("   ParentOrderId: %d\n", order.ParentOrderId)
		fmt.Printf("   CloseOrderIds: '%s'\n", order.CloseOrderIds)
		fmt.Printf("   ClientOrderId: %s\n", order.ClientOrderId)
		fmt.Printf("   创建时间: %s\n", order.CreatedAt.Format("2006-01-02 15:04:05"))

		if order.CloseOrderIds != "" {
			fmt.Printf("   📋 有平仓订单关联: %s\n", order.CloseOrderIds)
			closeOrderIds := parseCloseOrderIds(order.CloseOrderIds)
			for _, closeID := range closeOrderIds {
				var closeOrder pdb.ScheduledOrder
				if err := gdb.GormDB().Where("id = ?", closeID).First(&closeOrder).Error; err == nil {
					fmt.Printf("      - 关联平仓订单 ID: %d, 状态: %s, ReduceOnly: %v\n",
						closeOrder.ID, closeOrder.Status, closeOrder.ReduceOnly)
				} else {
					fmt.Printf("      - 关联平仓订单 ID: %d 查询失败: %v\n", closeID, err)
				}
			}
		} else {
			fmt.Printf("   ⚠️ 无平仓订单关联\n")
		}

		// 检查是否应该显示"已结束"
		shouldShowEnded := false
		reason := ""

		if order.Status == "filled" || order.Status == "completed" {
			if !order.ReduceOnly {
				if order.CloseOrderIds != "" {
					closeOrderIds := parseCloseOrderIds(order.CloseOrderIds)
					if len(closeOrderIds) > 0 {
						shouldShowEnded = true
						reason = "开仓订单 + 有平仓订单关联"
					}
				} else {
					reason = "开仓订单 + 无平仓订单关联"
				}
			} else {
				reason = "平仓订单"
			}
		} else {
			reason = fmt.Sprintf("状态不是filled/completed: %s", order.Status)
		}

		if shouldShowEnded {
			fmt.Printf("   ✅ 应该显示: 已结束 (%s)\n", reason)
		} else {
			fmt.Printf("   ❌ 不显示已结束 (%s)\n", reason)
		}
	}

	// 检查ExternalOperation记录
	fmt.Println("\n📋 检查BTRUSDT的外部操作记录:")
	var externalOps []pdb.ExternalOperation
	err = gdb.GormDB().
		Where("symbol = ?", "BTRUSDT").
		Order("detected_at DESC").
		Find(&externalOps).Error

	if err != nil {
		log.Printf("查询外部操作失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个外部操作记录:\n", len(externalOps))
		for _, op := range externalOps {
			fmt.Printf("  - 类型: %s, 数量: %s -> %s, 状态: %s, 时间: %s\n",
				op.OperationType, op.OldAmount, op.NewAmount, op.Status,
				op.DetectedAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 检查BracketLink记录
	fmt.Println("\n🔗 检查BTRUSDT的Bracket链接:")
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().
		Where("entry_client_id LIKE ? OR tp_client_id LIKE ? OR sl_client_id LIKE ?", "%BTRUSDT%", "%BTRUSDT%", "%BTRUSDT%").
		Find(&bracketLinks).Error

	if err != nil {
		log.Printf("查询BracketLink失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个Bracket链接:\n", len(bracketLinks))
		for _, link := range bracketLinks {
			fmt.Printf("  - GroupID: %s, Status: %s\n", link.GroupID, link.Status)
			fmt.Printf("    Entry: %s, TP: %s, SL: %s\n", link.EntryClientID, link.TPClientID, link.SLClientID)
		}
	}

	fmt.Println("\n🎯 问题诊断:")
	fmt.Println("1. 检查开仓订单状态是否为filled")
	fmt.Println("2. 检查是否有关联的平仓订单")
	fmt.Println("3. 检查CloseOrderIds字段格式是否正确")
	fmt.Println("4. 检查平仓订单是否存在且状态正确")
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