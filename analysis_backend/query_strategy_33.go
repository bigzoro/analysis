package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("=== 查询策略ID 33的配置 ===")

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

	// 查询策略33的止盈止损配置
	var strategy struct {
		ID                          uint    `json:"id"`
		Name                        string  `json:"name"`
		EnableStopLoss              bool    `json:"enable_stop_loss"`
		EnableTakeProfit            bool    `json:"enable_take_profit"`
		EnableMarginLossStopLoss    bool    `json:"enable_margin_loss_stop_loss"`
		EnableMarginProfitTakeProfit bool   `json:"enable_margin_profit_take_profit"`
		TakeProfitPercent           float64 `json:"take_profit_percent"`
		StopLossPercent             float64 `json:"stop_loss_percent"`
		MarginProfitTakeProfitPercent float64 `json:"margin_profit_take_profit_percent"`
		MarginLossStopLossPercent   float64 `json:"margin_loss_stop_loss_percent"`
	}

	err = gdb.GormDB().Table("trading_strategies").Where("id = ?", 33).Select("id, name, enable_stop_loss, enable_take_profit, enable_margin_loss_stop_loss, enable_margin_profit_take_profit, take_profit_percent, stop_loss_percent, margin_profit_take_profit_percent, margin_loss_stop_loss_percent").First(&strategy).Error

	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}

	fmt.Printf("策略ID: %d\n", strategy.ID)
	fmt.Printf("名称: %s\n", strategy.Name)
	fmt.Printf("止损启用: %v\n", strategy.EnableStopLoss)
	fmt.Printf("止盈启用: %v\n", strategy.EnableTakeProfit)
	fmt.Printf("保证金止损启用: %v\n", strategy.EnableMarginLossStopLoss)
	fmt.Printf("保证金止盈启用: %v\n", strategy.EnableMarginProfitTakeProfit)
	fmt.Printf("止盈百分比: %.2f%%\n", strategy.TakeProfitPercent)
	fmt.Printf("止损百分比: %.2f%%\n", strategy.StopLossPercent)
	fmt.Printf("保证金止盈百分比: %.2f%%\n", strategy.MarginProfitTakeProfitPercent)
	fmt.Printf("保证金止损百分比: %.2f%%\n", strategy.MarginLossStopLossPercent)

	// 检查BracketEnabled的计算结果
	bracketEnabled := strategy.EnableStopLoss || strategy.EnableTakeProfit ||
		strategy.EnableMarginLossStopLoss || strategy.EnableMarginProfitTakeProfit

	fmt.Printf("\nBracketEnabled (一键三连): %v\n", bracketEnabled)

	if bracketEnabled {
		fmt.Println("\n启用的止盈止损配置:")
		if strategy.EnableTakeProfit {
			fmt.Printf("- 传统止盈: %.2f%%\n", strategy.TakeProfitPercent)
		}
		if strategy.EnableStopLoss {
			fmt.Printf("- 传统止损: %.2f%%\n", strategy.StopLossPercent)
		}
		if strategy.EnableMarginProfitTakeProfit {
			fmt.Printf("- 保证金止盈: %.2f%%\n", strategy.MarginProfitTakeProfitPercent)
		}
		if strategy.EnableMarginLossStopLoss {
			fmt.Printf("- 保证金止损: %.2f%%\n", strategy.MarginLossStopLossPercent)
		}
	}
}