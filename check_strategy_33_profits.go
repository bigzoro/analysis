package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("检查策略ID 33的完整配置（包括盈利加仓）...")

	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	var strategy db.TradingStrategy
	result := gdb.GormDB().Where("id = ?", 33).First(&strategy)
	if result.Error != nil {
		log.Fatalf("查询策略失败: %v", result.Error)
	}

	fmt.Printf("策略ID: %d\n", strategy.ID)
	fmt.Printf("策略名称: %s\n", strategy.Name)

	// 解析条件配置
	conditions := strategy.Conditions
	fmt.Printf("\n杠杆配置:\n")
	fmt.Printf("  杠杆倍数: %.1fx\n", conditions.FuturesPriceShortLeverage)
	fmt.Printf("  保证金模式: %s\n", conditions.MarginMode)

	fmt.Printf("\n盈利加仓配置:\n")
	fmt.Printf("  盈利加仓启用: %v\n", conditions.ProfitScalingEnabled)
	if conditions.ProfitScalingEnabled {
		fmt.Printf("  触发盈利百分比: %.2f%%\n", conditions.ProfitScalingPercent)
		fmt.Printf("  加仓金额: %.2f USDT\n", conditions.ProfitScalingAmount)
		fmt.Printf("  最大加仓次数: %d\n", conditions.ProfitScalingMaxCount)
		fmt.Printf("  当前已加仓次数: %d\n", conditions.ProfitScalingCurrentCount)
	}

	fmt.Printf("\n止盈止损配置:\n")
	fmt.Printf("  止损启用: %v\n", conditions.EnableStopLoss)
	fmt.Printf("  止损百分比: %.2f%%\n", conditions.StopLossPercent)
	fmt.Printf("  止盈启用: %v\n", conditions.EnableTakeProfit)
	fmt.Printf("  止盈百分比: %.2f%%\n", conditions.TakeProfitPercent)

	fmt.Printf("\n其他配置:\n")
	fmt.Printf("  交易类型: %s\n", conditions.TradingType)
	fmt.Printf("  跳过已有持仓: %v\n", conditions.SkipHeldPositions)
}
