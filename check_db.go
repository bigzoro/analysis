package main

import (
	"analysis/internal/config"
	"analysis/internal/db"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	database, err := db.NewDatabaseWithConfig(cfg)
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer database.Close()

	// 检查定时订单数据
	var orders []pdb.ScheduledOrder
	err = database.DB().Find(&orders).Error
	if err != nil {
		log.Printf("查询订单失败: %v", err)
		return
	}

	log.Printf("总订单数: %d", len(orders))

	// 检查关联关系
	openOrders := 0
	closeOrders := 0
	linkedOrders := 0

	for _, order := range orders {
		if order.ReduceOnly {
			closeOrders++
			if order.ParentOrderId > 0 {
				linkedOrders++
			}
		} else {
			openOrders++
			if order.CloseOrderIds != "" {
				linkedOrders++
			}
		}
	}

	log.Printf("开仓订单数: %d", openOrders)
	log.Printf("平仓订单数: %d", closeOrders)
	log.Printf("有关联关系的订单数: %d", linkedOrders)

	// 显示一些具体的关联关系
	log.Printf("\n具体的关联关系:")
	for _, order := range orders {
		if order.ReduceOnly && order.ParentOrderId > 0 {
			log.Printf("平仓订单 %d -> 开仓订单 %d (交易对: %s)", order.ID, order.ParentOrderId, order.Symbol)
		} else if !order.ReduceOnly && order.CloseOrderIds != "" {
			log.Printf("开仓订单 %d -> 平仓订单 %s (交易对: %s)", order.ID, order.CloseOrderIds, order.Symbol)
		}
	}
}
