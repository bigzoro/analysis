package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("🔄 添加 skip_close_orders_hours 字段到 trading_strategies 表")

	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 添加新字段
	fmt.Println("📝 添加 skip_close_orders_hours 字段...")
	err = db.Exec(`
		ALTER TABLE trading_strategies
		ADD COLUMN skip_close_orders_hours INT DEFAULT 24
	`).Error

	if err != nil {
		log.Printf("添加字段失败: %v", err)
		return
	}

	// 为现有记录设置默认值
	fmt.Println("📝 为现有记录设置默认值...")
	err = db.Exec(`
		UPDATE trading_strategies
		SET skip_close_orders_hours = 24
		WHERE skip_close_orders_within_24_hours = 1
	`).Error

	if err != nil {
		log.Printf("设置默认值失败: %v", err)
		return
	}

	// 设置未启用24小时过滤的记录为0
	err = db.Exec(`
		UPDATE trading_strategies
		SET skip_close_orders_hours = 0
		WHERE skip_close_orders_within_24_hours = 0 OR skip_close_orders_within_24_hours IS NULL
	`).Error

	if err != nil {
		log.Printf("设置未启用记录失败: %v", err)
		return
	}

	fmt.Println("✅ 字段添加和数据迁移完成！")
	fmt.Println("📋 迁移结果:")
	fmt.Println("   - 添加了 skip_close_orders_hours 字段 (INT, DEFAULT 24)")
	fmt.Println("   - 已启用24小时过滤的策略: 设置为24小时")
	fmt.Println("   - 未启用24小时过滤的策略: 设置为0小时")

	// 验证迁移结果
	fmt.Println("\n🔍 验证迁移结果...")
	var count int64
	db.Model(&struct{}{}).Table("trading_strategies").Where("skip_close_orders_hours = 24").Count(&count)
	fmt.Printf("   - 设置为24小时的策略数量: %d\n", count)

	db.Model(&struct{}{}).Table("trading_strategies").Where("skip_close_orders_hours = 0").Count(&count)
	fmt.Printf("   - 设置为0小时的策略数量: %d\n", count)

	fmt.Println("\n🎉 数据库迁移完成！现在可以删除旧字段。")
	fmt.Println("⚠️  注意: 请在确认新功能正常工作后再删除旧字段 skip_close_orders_within_24_hours")
}
