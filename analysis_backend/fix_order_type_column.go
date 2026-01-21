package main

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	log.Println("🔧 修复scheduled_orders表order_type字段长度")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 修改order_type字段长度从16到32
	sql := "ALTER TABLE scheduled_orders MODIFY COLUMN order_type VARCHAR(32) NOT NULL"
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("修改order_type字段失败: %v", err)
	}

	log.Println("✅ order_type字段长度已从16增加到32")
	log.Println("✅ 现在支持TAKE_PROFIT_MARKET和STOP_MARKET订单类型")
}