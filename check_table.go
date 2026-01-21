package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 数据库连接
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 查询表结构
	type ColumnInfo struct {
		Field   string
		Type    string
		Null    string
		Key     string
		Default interface{}
		Extra   string
	}

	var columns []ColumnInfo
	err = db.Raw("DESCRIBE trading_strategies").Scan(&columns).Error
	if err != nil {
		log.Fatal("Failed to describe table:", err)
	}

	fmt.Println("trading_strategies table structure:")
	for _, col := range columns {
		fmt.Printf("- %s (%s)\n", col.Field, col.Type)
	}
}