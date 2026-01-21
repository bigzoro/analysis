package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("🔍 调试BTRUSDT Bracket订单状态")

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

	// 1. 检查BTRUSDT的开仓订单
	fmt.Println("\n📊 1. 检查BTRUSDT开仓订单:")
	var entryOrders []pdb.ScheduledOrder
	err = gdb.GormDB().
		Where("symbol = ? AND reduce_only = false AND user_id = ?", "BTRUSDT", 0).
		Order("created_at DESC").
		Find(&entryOrders).Error

	if err != nil {
		log.Printf("查询开仓订单失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个开仓订单:\n", len(entryOrders))
		for i, order := range entryOrders {
			fmt.Printf("\n%d. 订单ID: %d\n", i+1, order.ID)
			fmt.Printf("   状态: %s\n", order.Status)
			fmt.Printf("   BracketEnabled: %v\n", order.BracketEnabled)
			fmt.Printf("   ClientOrderId: %s\n", order.ClientOrderId)
			fmt.Printf("   CloseOrderIds: '%s'\n", order.CloseOrderIds)

			// 检查BracketEnabled逻辑
			if order.BracketEnabled {
				fmt.Printf("   ✅ 是Bracket订单\n")

				// 检查BracketLink
				var bracketLink pdb.BracketLink
				err := gdb.GormDB().Where("entry_client_id = ?", order.ClientOrderId).First(&bracketLink).Error
				if err != nil {
					fmt.Printf("   ❌ BracketLink不存在: %v\n", err)
				} else {
					fmt.Printf("   ✅ BracketLink存在: GroupID=%s, Status=%s\n", bracketLink.GroupID, bracketLink.Status)
				}
			} else {
				fmt.Printf("   ❌ 不是Bracket订单\n")
			}

			// 检查关联的平仓订单
			if order.CloseOrderIds != "" {
				closeOrderIds := parseCloseOrderIds(order.CloseOrderIds)
				fmt.Printf("   📋 有关联的平仓订单: %v\n", closeOrderIds)

				for _, closeID := range closeOrderIds {
					var closeOrder pdb.ScheduledOrder
					if err := gdb.GormDB().Where("id = ?", closeID).First(&closeOrder).Error; err == nil {
						fmt.Printf("      - 平仓订单ID: %d, 状态: %s, ReduceOnly: %v\n",
							closeOrder.ID, closeOrder.Status, closeOrder.ReduceOnly)
					} else {
						fmt.Printf("      - 平仓订单ID: %d 查询失败: %v\n", closeID, err)
					}
				}
			}
		}
	}

	// 2. 检查BTRUSDT的平仓订单
	fmt.Println("\n📊 2. 检查BTRUSDT平仓订单:")
	var closeOrders []pdb.ScheduledOrder
	err = gdb.GormDB().
		Where("symbol = ? AND reduce_only = true AND user_id = ?", "BTRUSDT", 0).
		Order("created_at DESC").
		Find(&closeOrders).Error

	if err != nil {
		log.Printf("查询平仓订单失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个平仓订单:\n", len(closeOrders))
		for i, order := range closeOrders {
			fmt.Printf("\n%d. 平仓订单ID: %d\n", i+1, order.ID)
			fmt.Printf("   状态: %s\n", order.Status)
			fmt.Printf("   ParentOrderId: %d\n", order.ParentOrderId)
			fmt.Printf("   ClientOrderId: %s\n", order.ClientOrderId)
			fmt.Printf("   创建时间: %s\n", order.CreatedAt.Format("2006-01-02 15:04:05"))

			// 检查父订单
			if order.ParentOrderId > 0 {
				var parentOrder pdb.ScheduledOrder
				if err := gdb.GormDB().Where("id = ?", order.ParentOrderId).First(&parentOrder).Error; err == nil {
					fmt.Printf("   父订单状态: %s, BracketEnabled: %v\n", parentOrder.Status, parentOrder.BracketEnabled)
				} else {
					fmt.Printf("   父订单查询失败: %v\n", err)
				}
			}
		}
	}

	// 3. 检查ExternalOperation记录
	fmt.Println("\n📊 3. 检查BTRUSDT的外部操作记录:")
	var externalOps []pdb.ExternalOperation
	err = gdb.GormDB().
		Where("symbol = ?", "BTRUSDT").
		Order("detected_at DESC").
		Limit(5).
		Find(&externalOps).Error

	if err != nil {
		log.Printf("查询外部操作失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个外部操作记录:\n", len(externalOps))
		for _, op := range externalOps {
			fmt.Printf("  - ID: %d, 类型: %s, 数量: %s -> %s, 状态: %s\n",
				op.ID, op.OperationType, op.OldAmount, op.NewAmount, op.Status)
			fmt.Printf("    时间: %s\n", op.DetectedAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 4. 分析问题
	fmt.Println("\n🎯 问题分析:")

	// 检查是否有Bracket订单
	bracketOrderCount := 0
	for _, order := range entryOrders {
		if order.BracketEnabled {
			bracketOrderCount++
		}
	}

	if bracketOrderCount == 0 {
		fmt.Println("❌ 没有找到Bracket订单 (BracketEnabled=true)")
		fmt.Println("   可能原因: 订单创建时没有启用Bracket功能")
	} else {
		fmt.Printf("✅ 找到 %d 个Bracket订单\n", bracketOrderCount)

		// 检查是否有外部平仓记录
		fullCloseCount := 0
		for _, op := range externalOps {
			if op.OperationType == "external_full_close" {
				fullCloseCount++
			}
		}

		if fullCloseCount == 0 {
			fmt.Println("❌ 没有找到外部完全平仓记录")
			fmt.Println("   可能原因: 持仓检测没有正确识别平仓操作")
		} else {
			fmt.Printf("✅ 找到 %d 个外部完全平仓记录\n", fullCloseCount)

			// 检查是否有平仓订单创建
			closeOrderCount := len(closeOrders)
			if closeOrderCount == 0 {
				fmt.Println("❌ 没有找到平仓订单记录")
				fmt.Println("   可能原因: handleBracketExternalClose没有正确执行")
			} else {
				fmt.Printf("✅ 找到 %d 个平仓订单记录\n", closeOrderCount)

				// 检查关联关系
				validAssociationCount := 0
				for _, entryOrder := range entryOrders {
					if entryOrder.CloseOrderIds != "" {
						validAssociationCount++
					}
				}

				if validAssociationCount == 0 {
					fmt.Println("❌ 没有找到有效的订单关联 (CloseOrderIds为空)")
					fmt.Println("   可能原因: CloseOrderIds没有正确设置")
				} else {
					fmt.Printf("✅ 找到 %d 个有效关联\n", validAssociationCount)
					fmt.Println("   问题可能在前端解析或显示逻辑")
				}
			}
		}
	}

	fmt.Println("\n🔍 排查步骤:")
	fmt.Println("1. 检查订单是否为Bracket订单 (BracketEnabled=true)")
	fmt.Println("2. 检查是否有外部操作记录 (external_full_close)")
	fmt.Println("3. 检查是否创建了平仓订单")
	fmt.Println("4. 检查CloseOrderIds是否正确设置")
	fmt.Println("5. 检查前端API返回的related_orders数据")
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