package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 检查BTCUSDT的日线数据
	var count int64
	db.Model(&struct{}{}).Table("market_klines").
		Where("symbol = ? AND kind = 'spot' AND `interval` = '1d'", "BTCUSDT").
		Count(&count)
	fmt.Printf("BTCUSDT日线数据条数: %d\n", count)

	// 检查最近的数据
	var prices []float64
	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ? AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
		AND open_time >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		ORDER BY open_time DESC
		LIMIT 5
	`
	err = db.Raw(query, "BTCUSDT").Scan(&prices).Error
	if err != nil {
		fmt.Printf("查询BTCUSDT数据失败: %v\n", err)
	} else {
		fmt.Printf("BTCUSDT最近5天收盘价: %v\n", prices)
	}

	// 检查ETHUSDT的数据
	db.Model(&struct{}{}).Table("market_klines").
		Where("symbol = ? AND kind = 'spot' AND `interval` = '1d'", "ETHUSDT").
		Count(&count)
	fmt.Printf("ETHUSDT日线数据条数: %d\n", count)
}