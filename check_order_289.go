package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	pdb "analysis/internal/db"
)

func main() {
	// 获取项目根目录
	rootDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatal("无法获取项目根目录:", err)
	}

	// 切换到analysis_backend目录
	backendDir := filepath.Join(rootDir, "analysis_backend")
	if err := os.Chdir(backendDir); err != nil {
		log.Fatal("无法切换到analysis_backend目录:", err)
	}

	// 初始化数据库连接
	db, err := pdb.NewDBConnection()
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 查询订单ID为289的详细信息
	var order pdb.ScheduledOrder
	err = db.First(&order, 289).Error
	if err != nil {
		log.Fatal("查询订单失败:", err)
	}

	fmt.Println("=== 订单ID 289 详细信息 ===")
	fmt.Printf("ID: %d\n", order.ID)
	fmt.Printf("UserID: %d\n", order.UserID)
	fmt.Printf("Exchange: %s\n", order.Exchange)
	fmt.Printf("Testnet: %v\n", order.Testnet)
	fmt.Printf("Symbol: %s\n", order.Symbol)
	fmt.Printf("Side: %s\n", order.Side)
	fmt.Printf("OrderType: %s\n", order.OrderType)
	fmt.Printf("Quantity: %s\n", order.Quantity)
	fmt.Printf("AdjustedQuantity: %s\n", order.AdjustedQuantity)
	fmt.Printf("Price: %s\n", order.Price)
	fmt.Printf("Leverage: %d\n", order.Leverage)
	fmt.Printf("ReduceOnly: %v\n", order.ReduceOnly)
	fmt.Printf("StrategyID: %v\n", order.StrategyID)
	fmt.Printf("ExecutionID: %v\n", order.ExecutionID)
	fmt.Printf("BracketEnabled: %v\n", order.BracketEnabled)
	fmt.Printf("TPPercent: %.2f%%\n", order.TPPercent)
	fmt.Printf("SLPercent: %.2f%%\n", order.SLPercent)
	fmt.Printf("ActualTPPercent: %.2f%%\n", order.ActualTPPercent)
	fmt.Printf("ActualSLPercent: %.2f%%\n", order.ActualSLPercent)
	fmt.Printf("TPPrice: %s\n", order.TPPrice)
	fmt.Printf("SLPrice: %s\n", order.SLPrice)
	fmt.Printf("WorkingType: %s\n", order.WorkingType)
	fmt.Printf("TriggerTime: %v\n", order.TriggerTime)
	fmt.Printf("Status: %s\n", order.Status)
	fmt.Printf("Result: %s\n", order.Result)
	fmt.Printf("ClientOrderId: %s\n", order.ClientOrderId)
	fmt.Printf("ExchangeOrderId: %s\n", order.ExchangeOrderId)
	fmt.Printf("ExecutedQty: %s\n", order.ExecutedQty)
	fmt.Printf("AvgPrice: %s\n", order.AvgPrice)
	fmt.Printf("ParentOrderId: %d\n", order.ParentOrderId)
	fmt.Printf("CloseOrderIds: %s\n", order.CloseOrderIds)
	fmt.Printf("CreatedAt: %v\n", order.CreatedAt)
	fmt.Printf("UpdatedAt: %v\n", order.UpdatedAt)

	// 查询相关的平仓订单
	if order.CloseOrderIds != "" {
		fmt.Println("\n=== 关联的平仓订单 ===")
		fmt.Printf("CloseOrderIds: %s\n", order.CloseOrderIds)
	}

	// 查询父订单（如果是平仓订单）
	if order.ParentOrderId > 0 {
		fmt.Println("\n=== 父订单信息 ===")
		var parentOrder pdb.ScheduledOrder
		err = db.First(&parentOrder, order.ParentOrderId).Error
		if err != nil {
			fmt.Printf("查询父订单失败: %v\n", err)
		} else {
			fmt.Printf("父订单ID: %d\n", parentOrder.ID)
			fmt.Printf("父订单Symbol: %s\n", parentOrder.Symbol)
			fmt.Printf("父订单Side: %s\n", parentOrder.Side)
			fmt.Printf("父订单Status: %s\n", parentOrder.Status)
			fmt.Printf("父订单ExecutedQty: %s\n", parentOrder.ExecutedQty)
			fmt.Printf("父订单AvgPrice: %s\n", parentOrder.AvgPrice)
		}
	}

	// 查询BracketLink信息（如果是一键三连订单）
	var bracketLink pdb.BracketLink
	err = db.Where("schedule_id = ?", order.ID).First(&bracketLink).Error
	if err == nil {
		fmt.Println("\n=== BracketLink 信息 ===")
		fmt.Printf("GroupID: %s\n", bracketLink.GroupID)
		fmt.Printf("EntryClientID: %s\n", bracketLink.EntryClientID)
		fmt.Printf("TPClientID: %s\n", bracketLink.TPClientID)
		fmt.Printf("SLClientID: %s\n", bracketLink.SLClientID)
		fmt.Printf("Status: %s\n", bracketLink.Status)
		fmt.Printf("CreatedAt: %v\n", bracketLink.CreatedAt)
		fmt.Printf("UpdatedAt: %v\n", bracketLink.UpdatedAt)
	}

	// 查询相关的平仓订单
	fmt.Println("\n=== 关联的平仓订单查询 ===")
	var closeOrders []pdb.ScheduledOrder
	err = database.GormDB().Where("parent_order_id = ?", order.ID).Find(&closeOrders).Error
	if err != nil {
		fmt.Printf("查询平仓订单失败: %v\n", err)
	} else if len(closeOrders) > 0 {
		fmt.Printf("找到 %d 个平仓订单:\n", len(closeOrders))
		for i, closeOrder := range closeOrders {
			fmt.Printf("  [%d] ID: %d, Side: %s, Status: %s, ExecutedQty: %s, AvgPrice: %s, Result: %s\n",
				i+1, closeOrder.ID, closeOrder.Side, closeOrder.Status, closeOrder.ExecutedQty, closeOrder.AvgPrice, closeOrder.Result)
		}
	} else {
		fmt.Println("未找到平仓订单")
	}

	// 检查TP/SL订单的状态
	fmt.Println("\n=== TP/SL 订单状态检查 ===")
	tpOrderID := bracketLink.TPClientID
	slOrderID := bracketLink.SLClientID

	// 查询TP订单
	var tpOrder pdb.ScheduledOrder
	tpErr := database.GormDB().Where("client_order_id = ?", tpOrderID).First(&tpOrder).Error
	if tpErr == nil {
		fmt.Printf("止盈订单 - ID: %d, Status: %s, ExecutedQty: %s, AvgPrice: %s\n",
			tpOrder.ID, tpOrder.Status, tpOrder.ExecutedQty, tpOrder.AvgPrice)
	} else {
		fmt.Printf("查询止盈订单失败: %v\n", tpErr)
	}

	// 查询SL订单
	var slOrder pdb.ScheduledOrder
	slErr := database.GormDB().Where("client_order_id = ?", slOrderID).First(&slOrder).Error
	if slErr == nil {
		fmt.Printf("止损订单 - ID: %d, Status: %s, ExecutedQty: %s, AvgPrice: %s\n",
			slOrder.ID, slOrder.Status, slOrder.ExecutedQty, slOrder.AvgPrice)
	} else {
		fmt.Printf("查询止损订单失败: %v\n", slErr)
	}

	// 检查交易所持仓情况
	fmt.Println("\n=== 检查交易所持仓情况 ===")
	if database != nil {
		// 这里可以添加检查交易所持仓的逻辑
		// 由于没有直接的持仓查询API，我们可以通过分析订单历史来推断
		fmt.Printf("开仓订单状态: %s\n", order.Status)
		fmt.Printf("成交数量: %s\n", order.ExecutedQty)
		fmt.Printf("成交价格: %s\n", order.AvgPrice)

		if order.Status == "filled" && order.ExecutedQty != "" && order.AvgPrice != "" {
			fmt.Println("✅ 开仓订单已成交")

			// 检查是否有平仓订单
			if len(closeOrders) > 0 {
				fmt.Println("✅ 找到平仓订单，说明仓位已被平掉")
				for i, closeOrder := range closeOrders {
					fmt.Printf("   平仓订单 %d: 状态=%s, 数量=%s\n", i+1, closeOrder.Status, closeOrder.ExecutedQty)
				}
			} else {
				fmt.Println("⚠️  没有找到平仓订单，但BracketLink状态为active")
				fmt.Println("   这表明TP/SL可能已被触发，但本地没有同步记录")
				fmt.Println("   建议手动检查交易所持仓，或等待下次订单同步")
			}
		}
	}

	// 检查是否有相关的订单同步记录（如果存在的话）
	fmt.Println("\n=== 订单同步状态检查 ===")
	if order.ClientOrderId != "" {
		fmt.Printf("ClientOrderId: %s\n", order.ClientOrderId)
	}
	if order.ExchangeOrderId != "" {
		fmt.Printf("ExchangeOrderId: %s\n", order.ExchangeOrderId)
	}
}
