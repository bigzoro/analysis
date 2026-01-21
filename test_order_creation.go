package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"analysis/internal/pdb"
)

func main() {
	// 连接数据库
	dsn := "root:123456@tcp(127.0.0.1:3306)/trading?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 测试订单创建
	order := &pdb.ScheduledOrder{
		UserID:         1,
		Exchange:       "binance_futures",
		Testnet:        true,
		Symbol:         "FETUSDT",
		Side:           "SELL",
		OrderType:      "MARKET",
		Quantity:       "10",
		Price:          "",
		Leverage:       1,
		ReduceOnly:     false,
		TriggerTime:    time.Now(),
		Status:         "pending",
		BracketEnabled: false,
		TPPercent:      0,
		SLPercent:      0,
		WorkingType:    "MARK_PRICE",
	}

	// 创建订单
	if err := db.Create(order).Error; err != nil {
		log.Printf("创建订单失败: %v", err)
	} else {
		log.Printf("成功创建订单 ID: %d", order.ID)
	}

	// 查询刚创建的订单
	var createdOrder pdb.ScheduledOrder
	if err := db.First(&createdOrder, order.ID).Error; err != nil {
		log.Printf("查询订单失败: %v", err)
	} else {
		fmt.Printf("订单详情: %+v\n", createdOrder)
	}
}


