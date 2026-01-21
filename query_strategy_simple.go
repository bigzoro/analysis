package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("=== 策略ID 27 详细信息 ===")

	// 查询所有字段
	rows, err := db.Query("SELECT * FROM trading_strategies WHERE id = 27")
	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Fatal("获取列名失败:", err)
	}

	fmt.Println("表字段:", columns)

	if rows.Next() {
		// 创建一个interface{}切片来存储值
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			log.Fatal("扫描失败:", err)
		}

		// 打印所有字段值
		for i, col := range columns {
			fmt.Printf("%s: %v\n", col, values[i])
		}
	} else {
		fmt.Println("未找到策略ID 27")
	}
}