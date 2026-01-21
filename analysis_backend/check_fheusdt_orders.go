package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("🔍 检查FHEUSDT订单详情")

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

	fmt.Println("\n1️⃣ 检查开仓订单#1289")
	var entryOrder pdb.ScheduledOrder
	err = gdb.GormDB().Where("id = ?", 1289).First(&entryOrder).Error
	if err != nil {
		log.Printf("查询开仓订单失败: %v", err)
	} else {
		fmt.Printf("开仓订单详情:\n")
		fmt.Printf("  ID: %d\n", entryOrder.ID)
		fmt.Printf("  客户端ID: %s\n", entryOrder.ClientOrderId)
		fmt.Printf("  状态: %s\n", entryOrder.Status)
		fmt.Printf("  类型: %s\n", entryOrder.OrderType)
		fmt.Printf("  方向: %s\n", entryOrder.Side)
		fmt.Printf("  数量: %s\n", entryOrder.Quantity)
		fmt.Printf("  价格: %s\n", entryOrder.Price)
		fmt.Printf("  执行数量: %s\n", entryOrder.ExecutedQty)
		fmt.Printf("  平均价格: %s\n", entryOrder.AvgPrice)
		fmt.Printf("  创建时间: %s\n", entryOrder.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  交易所订单ID: %s\n", entryOrder.ExchangeOrderId)
		fmt.Printf("  关联平仓订单: %s\n", entryOrder.CloseOrderIds)

		// 检查是否是Bracket订单
		if entryOrder.BracketEnabled {
			fmt.Printf("  Bracket订单: true\n")
			fmt.Printf("  TP百分比: %.2f%%\n", entryOrder.TPPercent)
			fmt.Printf("  SL百分比: %.2f%%\n", entryOrder.SLPercent)
		}
	}

	fmt.Println("\n2️⃣ 检查平仓订单#1295")
	var closeOrder pdb.ScheduledOrder
	err = gdb.GormDB().Where("id = ?", 1295).First(&closeOrder).Error
	if err != nil {
		log.Printf("查询平仓订单失败: %v", err)
	} else {
		fmt.Printf("平仓订单详情:\n")
		fmt.Printf("  ID: %d\n", closeOrder.ID)
		fmt.Printf("  客户端ID: %s\n", closeOrder.ClientOrderId)
		fmt.Printf("  状态: %s\n", closeOrder.Status)
		fmt.Printf("  类型: %s\n", closeOrder.OrderType)
		fmt.Printf("  方向: %s\n", closeOrder.Side)
		fmt.Printf("  数量: %s\n", closeOrder.Quantity)
		fmt.Printf("  执行数量: %s\n", closeOrder.ExecutedQty)
		fmt.Printf("  平均价格: %s\n", closeOrder.AvgPrice)
		fmt.Printf("  创建时间: %s\n", closeOrder.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  交易所订单ID: %s\n", closeOrder.ExchangeOrderId)
		fmt.Printf("  父订单ID: %d\n", closeOrder.ParentOrderId)
		fmt.Printf("  ReduceOnly: %v\n", closeOrder.ReduceOnly)
		fmt.Printf("  结果: %s\n", closeOrder.Result)
	}

	fmt.Println("\n3️⃣ 检查外部操作记录")
	var externalOps []pdb.ExternalOperation
	err = gdb.GormDB().Where("symbol = ? AND operation_type = ?",
		"FHEUSDT", "external_full_close").Order("detected_at DESC").Limit(5).Find(&externalOps).Error
	if err != nil {
		log.Printf("查询外部操作失败: %v", err)
	} else {
		fmt.Printf("找到%d条外部完全平仓记录:\n", len(externalOps))
		for i, op := range externalOps {
			fmt.Printf("  %d. ID:%d 时间:%s 置信度:%.2f\n",
				i+1, op.ID, op.DetectedAt.Format("15:04:05"), op.Confidence)
			fmt.Printf("     原持仓:%s -> 当前持仓:%s\n", op.OldAmount, op.NewAmount)
		}
	}

	fmt.Println("\n4️⃣ 检查Bracket订单状态")
	var brackets []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "FHEUSDT").Order("created_at DESC").Find(&brackets).Error
	if err != nil {
		log.Printf("查询Bracket订单失败: %v", err)
	} else {
		fmt.Printf("找到%d个FHEUSDT Bracket订单:\n", len(brackets))
		for _, bracket := range brackets {
			fmt.Printf("  Bracket %s - 状态:%s\n", bracket.GroupID, bracket.Status)
			fmt.Printf("    开仓:%s, TP:%s, SL:%s\n", bracket.EntryClientID, bracket.TPClientID, bracket.SLClientID)

			// 检查是否包含订单1289
			if bracket.EntryClientID != "" {
				// 从ClientOrderId提取ID
				if id, err := extractOrderIdFromClientId(bracket.EntryClientID); err == nil && id == 1289 {
					fmt.Printf("    🎯 这个Bracket包含开仓订单#1289！\n")
				}
			}
		}
	}

	fmt.Println("\n5️⃣ 分析结论")
	fmt.Println("基于以上数据分析：")

	if entryOrder.ID > 0 && closeOrder.ID > 0 {
		if entryOrder.BracketEnabled {
			fmt.Println("✅ 开仓订单是Bracket订单，包含止盈止损设置")
			fmt.Printf("✅ 止损百分比: %.2f%%, 止盈百分比: %.2f%%\n",
				entryOrder.SLPercent, entryOrder.TPPercent)

			if closeOrder.ReduceOnly && closeOrder.ParentOrderId == entryOrder.ID {
				fmt.Println("✅ 平仓订单正确关联到开仓订单")
				fmt.Println("🎯 结论：这很可能是通过止损或止盈自动平仓！")
				fmt.Println("   原因：")
				fmt.Println("   1. Bracket订单设置了止盈止损")
				fmt.Println("   2. 持仓从-25直接变为0，没有中间状态")
				fmt.Println("   3. 系统检测为external_full_close并关联订单")
				fmt.Println("   4. 置信度0.95很高")
			}
		} else {
			fmt.Println("❓ 开仓订单不是Bracket订单")
			fmt.Println("🤔 可能是手动平仓或系统外的其他操作")
		}
	} else {
		fmt.Println("❌ 无法获取完整的订单信息")
	}
}

func extractOrderIdFromClientId(clientOrderId string) (int, error) {
	// 尝试从ClientOrderId中提取订单ID
	// 格式可能是 "sch-{id}-..." 或其他
	if len(clientOrderId) > 4 && clientOrderId[:4] == "sch-" {
		// 移除前缀，找到数字部分
		parts := strings.Split(clientOrderId[4:], "-")
		if len(parts) > 0 {
			return strconv.Atoi(parts[0])
		}
	}
	return 0, fmt.Errorf("无法解析ClientOrderId: %s", clientOrderId)
}