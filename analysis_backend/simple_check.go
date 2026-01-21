package main

import (
	"fmt"
	"log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 直接连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 检查期货快照数量
	var snapshotCount int64
	db.Table("realtime_gainers_snapshots").Where("kind = ?", "futures").Count(&snapshotCount)
	fmt.Printf("futures快照数量: %d\n", snapshotCount)

	// 检查期货项目数量
	var itemsCount int64
	db.Table("realtime_gainers_items").
		Joins("JOIN realtime_gainers_snapshots s ON realtime_gainers_items.snapshot_id = s.id").
		Where("s.kind = ?", "futures").
		Count(&itemsCount)
	fmt.Printf("futures项目数量: %d\n", itemsCount)

	// 检查现货快照数量
	db.Table("realtime_gainers_snapshots").Where("kind = ?", "spot").Count(&snapshotCount)
	fmt.Printf("spot快照数量: %d\n", snapshotCount)

	// 检查现货项目数量
	db.Table("realtime_gainers_items").
		Joins("JOIN realtime_gainers_snapshots s ON realtime_gainers_items.snapshot_id = s.id").
		Where("s.kind = ?", "spot").
		Count(&itemsCount)
	fmt.Printf("spot项目数量: %d\n", itemsCount)
}