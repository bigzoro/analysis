package main

import (
	"fmt"
	"log"
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

	// 查询最新的策略执行记录
	var executions []pdb.StrategyExecution
	if err := db.Order("created_at DESC").Limit(5).Find(&executions).Error; err != nil {
		log.Printf("查询执行记录失败: %v", err)
		return
	}

	fmt.Println("最近的策略执行记录:")
	for _, exec := range executions {
		fmt.Printf("ID: %d, StrategyID: %d, CreateOrders: %v, Status: %s, CreatedAt: %v\n",
			exec.ID, exec.StrategyID, exec.CreateOrders, exec.Status, exec.CreatedAt)
	}

	// 如果有执行记录，查询相关的订单
	if len(executions) > 0 {
		var orders []pdb.ScheduledOrder
		if err := db.Where("execution_id = ?", executions[0].ID).Find(&orders).Error; err != nil {
			log.Printf("查询订单失败: %v", err)
		} else {
			fmt.Printf("Execution %d 的订单数量: %d\n", executions[0].ID, len(orders))
		}
	}
}


