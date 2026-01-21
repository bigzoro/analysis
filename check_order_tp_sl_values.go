package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查订单列表中止盈止损百分比的来源")

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

	// 查询最近的10个Bracket订单
	fmt.Println("\n📊 查询最近10个Bracket订单的止盈止损设置")
	var orders []pdb.ScheduledOrder
	err = gdb.GormDB().
		Where("bracket_enabled = ? AND status IN (?)", true, []string{"pending", "processing", "filled", "completed"}).
		Order("created_at DESC").
		Limit(10).
		Find(&orders).Error

	if err != nil {
		log.Printf("查询订单失败: %v", err)
		return
	}

	fmt.Printf("找到%d个订单:\n", len(orders))
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("%-4s %-12s %-8s %-8s %-8s %-8s %-15s\n",
		"ID", "Symbol", "TP%", "SL%", "ActTP%", "ActSL%", "StrategyID")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")

	for _, order := range orders {
		strategyID := "NULL"
		if order.StrategyID != nil {
			strategyID = fmt.Sprintf("%d", *order.StrategyID)
		}

		fmt.Printf("%-4d %-12s %-8.2f %-8.2f %-8.2f %-8.2f %-15s\n",
			order.ID,
			order.Symbol,
			order.TPPercent,
			order.SLPercent,
			order.ActualTPPercent,
			order.ActualSLPercent,
			strategyID)
	}

	fmt.Println("────────────────────────────────────────────────────────────────────────────────")

	// 检查是否有相同的策略ID
	if len(orders) > 0 {
		strategyID := orders[0].StrategyID
		if strategyID != nil {
			fmt.Printf("\n🔍 检查策略ID %d的配置:\n", *strategyID)

			strategy, err := pdb.GetTradingStrategy(gdb.DB(), orders[0].UserID, *strategyID)
			if err != nil {
				log.Printf("获取策略失败: %v", err)
			} else {
				fmt.Printf("策略名称: %s\n", strategy.Name)
				fmt.Printf("传统止盈: %.2f%%\n", strategy.Conditions.TakeProfitPercent)
				fmt.Printf("传统止损: %.2f%%\n", strategy.Conditions.StopLossPercent)
				fmt.Printf("保证金止盈: %.2f%%\n", strategy.Conditions.MarginProfitTakeProfitPercent)
				fmt.Printf("保证金止损: %.2f%%\n", strategy.Conditions.MarginLossStopLossPercent)

				fmt.Printf("\n止盈配置:\n")
				fmt.Printf("  启用传统止盈: %v\n", strategy.Conditions.EnableTakeProfit)
				fmt.Printf("  启用保证金止盈: %v\n", strategy.Conditions.EnableMarginProfitTakeProfit)

				fmt.Printf("\n止损配置:\n")
				fmt.Printf("  启用传统止损: %v\n", strategy.Conditions.EnableStopLoss)
				fmt.Printf("  启用保证金止损: %v\n", strategy.Conditions.EnableMarginLossStopLoss)
			}
		}
	}

	// 分析为什么都是相同的值
	fmt.Println("\n🔍 分析结果:")

	// 检查所有订单的百分比是否相同
	allSameTP := true
	allSameSL := true
	firstTP := orders[0].TPPercent
	firstSL := orders[0].SLPercent

	for _, order := range orders[1:] {
		if order.TPPercent != firstTP {
			allSameTP = false
		}
		if order.SLPercent != firstSL {
			allSameSL = false
		}
	}

	if allSameTP && allSameSL {
		fmt.Printf("✅ 所有订单使用相同的止盈(%.2f%%)和止损(%.2f%%)百分比\n", firstTP, firstSL)
		fmt.Println("💡 可能原因:")
		fmt.Println("   1. 所有订单使用相同的策略配置")
		fmt.Println("   2. 策略配置中设置了固定的百分比值")
		fmt.Println("   3. TimedOrderForm中的默认值被使用")
	} else {
		fmt.Println("❌ 订单的止盈止损百分比不完全相同")
	}

	// 检查是否有未关联策略的订单
	hasNullStrategy := false
	for _, order := range orders {
		if order.StrategyID == nil {
			hasNullStrategy = true
			break
		}
	}

	if hasNullStrategy {
		fmt.Println("\n⚠️  发现有订单未关联策略，这些订单可能使用默认值")
	}

	fmt.Println("\n💡 解决建议:")
	fmt.Println("1. 检查TimedOrderForm.vue中的默认值设置")
	fmt.Println("2. 检查策略配置中的止盈止损百分比")
	fmt.Println("3. 确认订单创建时是否正确传递了百分比参数")
	fmt.Println("4. 检查订单列表是否应该显示实际百分比而不是原始百分比")
}