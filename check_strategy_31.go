package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 检查策略ID 31的配置 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 查询策略
	type TradingStrategy struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Conditions  string `json:"conditions"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
		UserID      uint   `json:"user_id"`
		IsActive    bool   `json:"is_active"`
		Description string `json:"description"`
	}

	var strategy TradingStrategy
	result := db.Table("trading_strategies").Where("id = ?", 31).First(&strategy)
	if result.Error != nil {
		log.Fatalf("查询策略失败: %v", result.Error)
	}

	fmt.Printf("策略ID: %d\n", strategy.ID)
	fmt.Printf("策略名称: %s\n", strategy.Name)
	fmt.Printf("是否激活: %v\n", strategy.IsActive)
	fmt.Printf("用户ID: %d\n", strategy.UserID)
	fmt.Printf("描述: %s\n", strategy.Description)
	fmt.Printf("创建时间: %s\n", strategy.CreatedAt)
	fmt.Printf("更新时间: %s\n", strategy.UpdatedAt)
	fmt.Printf("完整配置JSON:\n%s\n", strategy.Conditions)
}