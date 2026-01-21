package main

import (
	"fmt"
	"log"
	"strconv"

	"analysis/internal/db"
)

func main() {
	fmt.Println("=== 检查订单ID 1418的详细信息 ===")

	// 初始化数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer gdb.Close()

	// 查询订单ID 1418
	var order db.ScheduledOrder
	result := gdb.GormDB().Where("id = ?", 1418).First(&order)
	if result.Error != nil {
		log.Fatalf("查询订单失败: %v", result.Error)
	}

	fmt.Printf("订单基本信息:\n")
	fmt.Printf("  ID: %d\n", order.ID)
	fmt.Printf("  交易对: %s\n", order.Symbol)
	fmt.Printf("  方向: %s\n", order.Side)
	fmt.Printf("  类型: %s\n", order.OrderType)
	fmt.Printf("  状态: %s\n", order.Status)
	fmt.Printf("  杠杆: %d\n", order.Leverage)
	fmt.Printf("  原始数量: %s\n", order.Quantity)
	fmt.Printf("  调整后数量: %s\n", order.AdjustedQuantity)
	fmt.Printf("  执行数量: %s\n", order.ExecutedQty)
	fmt.Printf("  平均价格: %s\n", order.AvgPrice)
	fmt.Printf("  创建时间: %s\n", order.CreatedAt.Format("2006-01-02 15:04:05"))

	// 计算名义价值
	if order.AdjustedQuantity != "" && order.AvgPrice != "" {
		// 解析数量和价格
		var quantity, price float64
		if qty, err := strconv.ParseFloat(order.AdjustedQuantity, 64); err == nil {
			quantity = qty
		}
		if px, err := strconv.ParseFloat(order.AvgPrice, 64); err == nil {
			price = px
		}

		notionalValue := quantity * price
		margin := notionalValue / float64(order.Leverage)

		fmt.Printf("\n计算结果:\n")
		fmt.Printf("  成交数量: %.6f\n", quantity)
		fmt.Printf("  成交价格: %.6f\n", price)
		fmt.Printf("  名义价值: %.2f USDT (成交金额)\n", notionalValue)
		fmt.Printf("  杠杆倍数: %d\n", order.Leverage)
		fmt.Printf("  保证金: %.2f USDT (用户设置金额)\n", margin)

		fmt.Printf("\n分析:\n")
		if margin >= 95 && margin <= 105 {
			fmt.Printf("  ✅ 用户设置约100 USDT，保证金%.2f USDT，符合预期\n", margin)
			fmt.Printf("  ✅ 页面显示300 USDT是名义价值，不是保证金\n")
			fmt.Printf("  ✅ 名义价值 = 保证金 × 杠杆 = %.2f × %d = %.2f\n", margin, order.Leverage, notionalValue)
		} else {
			fmt.Printf("  ❌ 计算结果异常，保证金%.2f USDT不在预期范围内\n", margin)
		}
	} else {
		fmt.Printf("\n缺少成交数据，无法计算名义价值\n")
	}

	// 检查执行ID和PerOrderAmount
	if order.ExecutionID != nil {
		fmt.Printf("\n执行信息:\n")
		var execution db.StrategyExecution
		if err := gdb.GormDB().Where("id = ?", *order.ExecutionID).First(&execution).Error; err == nil {
			fmt.Printf("  执行ID: %d\n", *order.ExecutionID)
			fmt.Printf("  每一单金额: %.2f USDT\n", execution.PerOrderAmount)
			fmt.Printf("  执行时间: %s\n", execution.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 检查BracketLink信息
	fmt.Printf("\nBracket订单信息:\n")
	if order.BracketEnabled {
		var bracketLink db.BracketLink
		if err := gdb.GormDB().Where("entry_client_id = ?", order.ClientOrderId).First(&bracketLink).Error; err == nil {
			fmt.Printf("  Bracket GroupID: %s\n", bracketLink.GroupID)
			fmt.Printf("  Bracket状态: %s\n", bracketLink.Status)
			fmt.Printf("  TP ClientID: %s\n", bracketLink.TPClientID)
			fmt.Printf("  SL ClientID: %s\n", bracketLink.SLClientID)
		} else {
			fmt.Printf("  未找到BracketLink信息\n")
		}
	} else {
		fmt.Printf("  非Bracket订单\n")
	}

	// 尝试计算名义价值（使用原始数量和价格）
	if order.Quantity != "" && order.AvgPrice != "" {
		if qty, err := strconv.ParseFloat(order.Quantity, 64); err == nil {
			if price, err := strconv.ParseFloat(order.AvgPrice, 64); err == nil {
				notionalValue := qty * price
				margin := notionalValue / float64(order.Leverage)

				fmt.Printf("\n基于原始数据的计算:\n")
				fmt.Printf("  原始数量: %.0f\n", qty)
				fmt.Printf("  平均价格: %.6f\n", price)
				fmt.Printf("  名义价值: %.2f USDT\n", notionalValue)
				fmt.Printf("  保证金: %.2f USDT\n", margin)
				fmt.Printf("  杠杆倍数: %d\n", order.Leverage)

				if margin >= 95 && margin <= 105 {
					fmt.Printf("  ✅ 保证金约%.2f USDT，符合用户设置的100 USDT\n", margin)
					fmt.Printf("  ✅ 页面显示300 USDT是名义价值，不是保证金\n")
				} else {
					fmt.Printf("  ⚠️ 保证金%.2f USDT与用户设置不符\n", margin)
				}
			}
		}
	}

	fmt.Printf("\n结论:\n")
	fmt.Printf("• 用户在策略启动时设置每一单金额: 100 USDT (保证金)\n")
	fmt.Printf("• 杠杆倍数: %d\n", order.Leverage)
	fmt.Printf("• 理论名义价值: 100 × %d = %d USDT\n", order.Leverage, 100*order.Leverage)
	fmt.Printf("• 如果页面显示成交金额为名义价值，那么显示300 USDT是正确的\n")
	fmt.Printf("• 如果页面显示的是保证金，那么就是BUG，需要修复\n")
}