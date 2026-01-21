package main

import (
	"fmt"
	"log"
	"analysis/internal/db"
)

func main() {
	fmt.Println("妫€鏌ョ瓥鐣D 33鐨勫畬鏁撮厤缃紙鍖呮嫭鐩堝埄鍔犱粨锛?..")

	// 杩炴帴鏁版嵁搴?	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("鏁版嵁搴撹繛鎺ュけ璐? %v", err)
	}
	defer gdb.Close()

	var strategy db.TradingStrategy
	result := gdb.GormDB().Where("id = ?", 33).First(&strategy)
	if result.Error != nil {
		log.Fatalf("鏌ヨ绛栫暐澶辫触: %v", result.Error)
	}

	fmt.Printf("绛栫暐ID: %d\n", strategy.ID)
	fmt.Printf("绛栫暐鍚嶇О: %s\n", strategy.Name)

	// 瑙ｆ瀽鏉′欢閰嶇疆
	conditions := strategy.Conditions
	fmt.Printf("\n鏉犳潌閰嶇疆:\n")
	fmt.Printf("  鏉犳潌鍊嶆暟: %.1fx\n", conditions.FuturesPriceShortLeverage)
	fmt.Printf("  淇濊瘉閲戞ā寮? %s\n", conditions.MarginMode)

	fmt.Printf("\n鐩堝埄鍔犱粨閰嶇疆:\n")
	fmt.Printf("  鐩堝埄鍔犱粨鍚敤: %v\n", conditions.ProfitScalingEnabled)
	if conditions.ProfitScalingEnabled {
		fmt.Printf("  瑙﹀彂鐩堝埄鐧惧垎姣? %.2f%%\n", conditions.ProfitScalingPercent)
		fmt.Printf("  鍔犱粨閲戦: %.2f USDT\n", conditions.ProfitScalingAmount)
		fmt.Printf("  鏈€澶у姞浠撴鏁? %d\n", conditions.ProfitScalingMaxCount)
		fmt.Printf("  褰撳墠宸插姞浠撴鏁? %d\n", conditions.ProfitScalingCurrentCount)
	}

	fmt.Printf("\n姝㈢泩姝㈡崯閰嶇疆:\n")
	fmt.Printf("  姝㈡崯鍚敤: %v\n", conditions.EnableStopLoss)
	fmt.Printf("  姝㈡崯鐧惧垎姣? %.2f%%\n", conditions.StopLossPercent)
	fmt.Printf("  姝㈢泩鍚敤: %v\n", conditions.EnableTakeProfit)
	fmt.Printf("  姝㈢泩鐧惧垎姣? %.2f%%\n", conditions.TakeProfitPercent)

	fmt.Printf("\n鍏朵粬閰嶇疆:\n")
	fmt.Printf("  浜ゆ槗绫诲瀷: %s\n", conditions.TradingType)
	fmt.Printf("  璺宠繃宸叉湁鎸佷粨: %v\n", conditions.SkipHeldPositions)
}
