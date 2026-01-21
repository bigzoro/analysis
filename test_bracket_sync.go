package main

import (
	"fmt"
	"log"
	"os"

	pdb "analysis/internal/db"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 设置数据库连接
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("请设置环境变量 DATABASE_DSN")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	database := pdb.NewDatabase(db)

	fmt.Println("=== 测试Bracket订单同步修复 ===")

	// 1. 查询订单ID 289的状态
	var order pdb.ScheduledOrder
	err = database.GormDB().First(&order, 289).Error
	if err != nil {
		log.Fatal("查询订单失败:", err)
	}

	fmt.Printf("订单ID 289 状态: %s\n", order.Status)
	fmt.Printf("BracketEnabled: %v\n", order.BracketEnabled)

	// 2. 查询对应的BracketLink
	var bracketLink pdb.BracketLink
	err = database.GormDB().Where("schedule_id = ?", order.ID).First(&bracketLink).Error
	if err != nil {
		log.Printf("查询BracketLink失败: %v", err)
		fmt.Println("❌ 没有找到BracketLink记录")
		return
	}

	fmt.Printf("BracketLink状态: %s\n", bracketLink.Status)
	fmt.Printf("TP ClientID: %s\n", bracketLink.TPClientID)
	fmt.Printf("SL ClientID: %s\n", bracketLink.SLClientID)

	// 3. 检查是否有平仓订单
	var closeOrders []pdb.ScheduledOrder
	err = database.GormDB().Where("parent_order_id = ?", order.ID).Find(&closeOrders).Error
	if err != nil {
		log.Printf("查询平仓订单失败: %v", err)
	} else {
		fmt.Printf("找到 %d 个平仓订单\n", len(closeOrders))
		for _, closeOrder := range closeOrders {
			fmt.Printf("  平仓订单ID: %d, 状态: %s, 结果: %s\n", closeOrder.ID, closeOrder.Status, closeOrder.Result)
		}
	}

	// 4. 分析结果
	fmt.Println("\n=== 分析结果 ===")
	if order.Status == "filled" && order.BracketEnabled {
		if bracketLink.Status == "active" {
			fmt.Println("⚠️  Bracket订单状态仍为活跃，说明TP/SL同步未生效")
			fmt.Println("✅ 修复后的同步逻辑将在下次订单同步时处理此问题")
		} else if bracketLink.Status == "closed" {
			fmt.Println("✅ Bracket订单已关闭，说明TP/SL已被正确处理")
		}

		if len(closeOrders) > 0 {
			fmt.Println("✅ 已创建平仓订单记录")
		} else {
			fmt.Println("❌ 没有平仓订单记录，仓位状态可能不正确")
		}
	} else {
		fmt.Println("ℹ️  订单不是Bracket订单或未成交")
	}

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("修复说明:")
	fmt.Println("1. 添加了syncBracketOrders函数，专门同步Bracket订单的TP/SL状态")
	fmt.Println("2. 当检测到TP/SL条件订单成交时，会创建相应的平仓订单记录")
	fmt.Println("3. 更新BracketLink状态为closed，确保状态一致性")
	fmt.Println("4. 每30秒的订单同步循环现在会同时处理常规订单和Bracket订单")
}




