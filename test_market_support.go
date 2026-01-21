package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 测试市场支持检查
func main() {
	// 数据库连接
	dsn := "root:password@tcp(localhost:3306)/crypto_analysis?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("🔍 测试交易对市场支持情况")
	fmt.Println("=" * 50)

	// 测试一些交易对的市场支持
	testSymbols := []string{"BTCUSDT", "ETHUSDT", "ZBTUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}

	for _, symbol := range testSymbols {
		var count int64
		err := db.Table("exchange_info").Where("symbol = ? AND is_spot = 1", symbol).Count(&count).Error
		if err != nil {
			fmt.Printf("%-10s: 数据库查询错误: %v\n", symbol, err)
			continue
		}

		hasSpot := count > 0

		err = db.Table("futures_contracts").Where("symbol = ?", symbol).Count(&count).Error
		if err != nil {
			fmt.Printf("%-10s: 数据库查询错误: %v\n", symbol, err)
			continue
		}

		hasFutures := count > 0

		status := "❌ 无市场支持"
		if hasSpot && hasFutures {
			status = "✅ 现货+期货"
		} else if hasSpot {
			status = "📊 仅现货"
		} else if hasFutures {
			status = "🔄 仅期货"
		}

		fmt.Printf("%-10s: %s (现货:%v, 期货:%v)\n", symbol, status, hasSpot, hasFutures)
	}

	// 检查ZBTUSDT的详细信息
	fmt.Println("\n📋 ZBTUSDT详细信息:")
	var spotInfo struct {
		Symbol    string
		IsSpot    bool
		IsMargin  bool
		IsFutures bool
		Status    string
	}

	err = db.Table("exchange_info").Where("symbol = ?", "ZBTUSDT").First(&spotInfo).Error
	if err != nil {
		fmt.Printf("ZBTUSDT现货信息查询失败: %v\n", err)
	} else {
		fmt.Printf("现货状态: %s, 保证金: %v\n", spotInfo.Status, spotInfo.IsMargin)
	}

	var futuresInfo struct {
		Symbol       string
		Status       string
		ContractType string
	}

	err = db.Table("futures_contracts").Where("symbol = ?", "ZBTUSDT").First(&futuresInfo).Error
	if err != nil {
		fmt.Printf("ZBTUSDT期货信息查询失败: %v\n", err)
	} else {
		fmt.Printf("期货状态: %s, 合约类型: %s\n", futuresInfo.Status, futuresInfo.ContractType)
	}
}
