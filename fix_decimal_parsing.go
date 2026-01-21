package main

import (
	"fmt"
	"log"
	"strconv"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TradingStrategy 修复版本
type TradingStrategy struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"index;not null"`
	Name        string `gorm:"size:128;not null"`
	Description string `gorm:"type:text"`

	// 单独定义网格字段以确保正确解析
	GridTradingEnabled   bool    `gorm:"column:grid_trading_enabled"`
	GridUpperPrice       float64 `gorm:"column:grid_upper_price;type:decimal(20,8)"`
	GridLowerPrice       float64 `gorm:"column:grid_lower_price;type:decimal(20,8)"`
	GridLevels           int     `gorm:"column:grid_levels"`
	GridInvestmentAmount float64 `gorm:"column:grid_investment_amount;type:decimal(20,8)"`
	GridStopLossEnabled  bool    `gorm:"column:grid_stop_loss_enabled"`
	GridStopLossPercent  float64 `gorm:"column:grid_stop_loss_percent;type:decimal(5,2)"`
	UseSymbolWhitelist   bool    `gorm:"column:use_symbol_whitelist"`
	SymbolWhitelist      string  `gorm:"column:symbol_whitelist;type:json"`

	IsRunning   bool `gorm:"default:false"`
	RunInterval int  `gorm:"default:60"`
}

func main() {
	fmt.Println("=== 修复decimal类型解析问题 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 读取原始数据（字符串格式）
	fmt.Println("\n📊 第一阶段: 读取原始decimal数据")
	var rawData map[string]interface{}
	query := `
		SELECT
			grid_trading_enabled,
			CAST(grid_upper_price AS CHAR) as grid_upper_price_str,
			CAST(grid_lower_price AS CHAR) as grid_lower_price_str,
			grid_levels,
			CAST(grid_investment_amount AS CHAR) as grid_investment_amount_str,
			grid_stop_loss_enabled,
			CAST(grid_stop_loss_percent AS CHAR) as grid_stop_loss_percent_str,
			use_symbol_whitelist,
			symbol_whitelist
		FROM trading_strategies
		WHERE id = 29
	`
	db.Raw(query).Scan(&rawData)

	fmt.Printf("原始数据:\n")
	for k, v := range rawData {
		fmt.Printf("  %s: %v (类型: %T)\n", k, v, v)
	}

	// 2. 手动解析decimal字符串
	fmt.Println("\n📊 第二阶段: 手动解析decimal值")
	gridUpperPriceStr := fmt.Sprintf("%v", rawData["grid_upper_price_str"])
	gridLowerPriceStr := fmt.Sprintf("%v", rawData["grid_lower_price_str"])
	investmentStr := fmt.Sprintf("%v", rawData["grid_investment_amount_str"])
	stopLossStr := fmt.Sprintf("%v", rawData["grid_stop_loss_percent_str"])

	gridUpperPrice, _ := strconv.ParseFloat(gridUpperPriceStr, 64)
	gridLowerPrice, _ := strconv.ParseFloat(gridLowerPriceStr, 64)
	gridInvestmentAmount, _ := strconv.ParseFloat(investmentStr, 64)
	gridStopLossPercent, _ := strconv.ParseFloat(stopLossStr, 64)

	fmt.Printf("解析结果:\n")
	fmt.Printf("  grid_upper_price: %s -> %.8f\n", gridUpperPriceStr, gridUpperPrice)
	fmt.Printf("  grid_lower_price: %s -> %.8f\n", gridLowerPriceStr, gridLowerPrice)
	fmt.Printf("  grid_investment_amount: %s -> %.2f\n", investmentStr, gridInvestmentAmount)
	fmt.Printf("  grid_stop_loss_percent: %s -> %.2f%%\n", stopLossStr, gridStopLossPercent)

	// 3. 测试GORM直接读取（可能失败）
	fmt.Println("\n📊 第三阶段: 测试GORM直接读取")
	var strategy TradingStrategy
	result := db.Where("id = ?", 29).First(&strategy)
	if result.Error != nil {
		fmt.Printf("GORM读取失败: %v\n", result.Error)
	} else {
		fmt.Printf("GORM读取结果:\n")
		fmt.Printf("  grid_upper_price: %.8f\n", strategy.GridUpperPrice)
		fmt.Printf("  grid_lower_price: %.8f\n", strategy.GridLowerPrice)
		fmt.Printf("  grid_investment_amount: %.2f\n", strategy.GridInvestmentAmount)
		fmt.Printf("  grid_stop_loss_percent: %.2f\n", strategy.GridStopLossPercent)
	}

	// 4. 验证修复效果
	fmt.Println("\n📊 第四阶段: 验证网格范围检查")
	currentPrice := 1.3390 // FILUSDT当前价格

	fmt.Printf("当前价格: %.4f\n", currentPrice)
	fmt.Printf("网格范围: [%.4f, %.4f]\n", gridLowerPrice, gridUpperPrice)

	if currentPrice >= gridLowerPrice && currentPrice <= gridUpperPrice {
		fmt.Printf("✅ 价格在网格范围内 - 修复成功!\n")

		// 计算网格位置和评分
		gridLevels := 20
		gridSpacing := (gridUpperPrice - gridLowerPrice) / float64(gridLevels)
		gridLevel := int((currentPrice - gridLowerPrice) / gridSpacing)
		if gridLevel >= gridLevels {
			gridLevel = gridLevels - 1
		}
		if gridLevel < 0 {
			gridLevel = 0
		}

		fmt.Printf("网格层级: %d/%d\n", gridLevel, gridLevels)
		fmt.Printf("网格间距: %.6f\n", gridSpacing)

		// 模拟评分计算
		midLevel := gridLevels / 2
		gridScore := calculateGridScore(gridLevel, midLevel, gridLevels)
		techScore := 0.6 // 基于之前的分析
		totalScore := gridScore*0.4 + techScore*0.3

		fmt.Printf("网格评分: %.3f\n", gridScore)
		fmt.Printf("技术评分: %.3f\n", techScore)
		fmt.Printf("综合评分: %.3f\n", totalScore)

		if totalScore > 0.2 {
			fmt.Printf("🎯 决策结果: 触发买入信号 ✅\n")
		} else {
			fmt.Printf("🎯 决策结果: 观望\n")
		}

	} else {
		fmt.Printf("❌ 价格超出网格范围 - 仍需修复\n")
	}

	// 5. 创建修复建议
	fmt.Println("\n🔧 第五阶段: 修复方案建议")

	fmt.Printf("问题根因:\n")
	fmt.Printf("  MySQL decimal类型在GORM中的自动转换失败\n")
	fmt.Printf("  StrategyConditions结构体字段定义正常，但运行时转换异常\n")

	fmt.Printf("\n解决方案:\n")
	fmt.Printf("1. 修改网格策略中的数据读取逻辑\n")
	fmt.Printf("2. 添加decimal字符串到float64的手动转换\n")
	fmt.Printf("3. 增加数据验证和错误处理\n")

	fmt.Printf("\n预期效果:\n")
	fmt.Printf("✅ 网格范围正确读取: [%.4f, %.4f]\n", gridLowerPrice, gridUpperPrice)
	fmt.Printf("✅ 价格在范围内判断正确\n")
	fmt.Printf("✅ 评分计算正常\n")
	fmt.Printf("✅ 触发交易信号\n")
}

func calculateGridScore(currentLevel, midLevel, totalLevels int) float64 {
	if currentLevel < midLevel {
		return 1.0 - float64(currentLevel)/float64(midLevel)
	} else if currentLevel > midLevel {
		return -1.0 * (float64(currentLevel-midLevel) / float64(totalLevels-midLevel))
	}
	return 0
}
