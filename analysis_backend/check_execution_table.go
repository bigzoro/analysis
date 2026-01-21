package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查策略执行表结构")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}

	// 检查strategy_executions表结构
	fmt.Printf("📋 strategy_executions 表结构:\n")
	var columns []map[string]interface{}
	gdb.Raw("DESCRIBE strategy_executions").Scan(&columns)

	for _, col := range columns {
		field := fmt.Sprintf("%v", col["Field"])
		fieldType := fmt.Sprintf("%v", col["Type"])
		null := fmt.Sprintf("%v", col["Null"])
		key := fmt.Sprintf("%v", col["Key"])
		defaultValue := fmt.Sprintf("%v", col["Default"])
		extra := fmt.Sprintf("%v", col["Extra"])

		fmt.Printf("  %-20s %-15s %-5s %-5s %-10s %s\n",
			field, fieldType, null, key, defaultValue, extra)
	}

	// 检查最近的执行记录
	fmt.Printf("\n📊 最近的执行记录:\n")
	var executions []map[string]interface{}
	err = gdb.Raw(`
		SELECT * FROM strategy_executions
		WHERE strategy_id = 29
		ORDER BY created_at DESC
		LIMIT 3
	`).Scan(&executions).Error

	if err != nil {
		log.Printf("❌ 查询执行记录失败: %v", err)
	} else {
		for i, exec := range executions {
			fmt.Printf("  执行 #%d:\n", i+1)
			for k, v := range exec {
				fmt.Printf("    %-15s: %v\n", k, v)
			}
			fmt.Println()
		}
	}

	// 检查scheduled_orders表结构
	fmt.Printf("📋 scheduled_orders 表结构:\n")
	var orderColumns []map[string]interface{}
	gdb.Raw("DESCRIBE scheduled_orders").Scan(&orderColumns)
	for _, col := range orderColumns {
		field := fmt.Sprintf("%v", col["Field"])
		fieldType := fmt.Sprintf("%v", col["Type"])
		fmt.Printf("  %-20s %s\n", field, fieldType)
	}
}