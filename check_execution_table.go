package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 检查strategy_executions表结构 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 检查表结构
	var columns []map[string]interface{}
	db.Raw("DESCRIBE strategy_executions").Scan(&columns)

	fmt.Printf("strategy_executions表结构:\n")
	for _, col := range columns {
		fmt.Printf("  %s: %s\n", col["Field"], col["Type"])
	}

	// 检查最近的记录
	var executions []map[string]interface{}
	db.Raw("SELECT * FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 1").Scan(&executions)

	if len(executions) > 0 {
		fmt.Printf("\n最近的策略执行记录:\n")
		for k, v := range executions[0] {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}
