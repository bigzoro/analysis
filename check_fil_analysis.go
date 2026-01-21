package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 分析FILUSDT网格策略为什么没有交易 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 获取FILUSDT价格
	var priceResult map[string]interface{}
	db.Raw("SELECT symbol, last_price FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&priceResult)

	currentPrice := 1.339 // 从输出中看到的价格
	gridUpper := 1.4919874999999998
	gridLower := 1.1700125000000001

	fmt.Printf("📊 价格分析:\n")
	fmt.Printf("  当前FILUSDT价格: %.4f\n", currentPrice)
	fmt.Printf("  网格范围: [%.4f, %.4f]\n", gridLower, gridUpper)

	if currentPrice >= gridLower && currentPrice <= gridUpper {
		fmt.Printf("  ✅ 价格在网格范围内\n")

		gridSpacing := (gridUpper - gridLower) / 20
		gridLevel := int((currentPrice - gridLower) / gridSpacing)
		if gridLevel >= 20 {
			gridLevel = 19
		}
		if gridLevel < 0 {
			gridLevel = 0
		}

		fmt.Printf("  网格层级: %d/20\n", gridLevel)
		fmt.Printf("  网格间距: %.6f\n", gridSpacing)

		// 计算网格位置评分
		midLevel := 10 // 20层的中点
		if gridLevel < midLevel {
			gridScore := 1.0 - float64(gridLevel)/float64(midLevel)
			fmt.Printf("  网格评分: %.3f (低层级，倾向买入)\n", gridScore)
		} else if gridLevel > midLevel {
			gridScore := -1.0 * (float64(gridLevel-midLevel) / float64(20-midLevel))
			fmt.Printf("  网格评分: %.3f (高层级，倾向卖出)\n", gridScore)
		} else {
			fmt.Printf("  网格评分: 0.0 (中间层级，中性)\n")
		}
	}

	// 检查技术指标表结构
	fmt.Println("\n=== 技术指标表结构 ===")
	var columns []map[string]interface{}
	db.Raw("DESCRIBE technical_indicators_caches").Scan(&columns)
	for _, col := range columns {
		fmt.Printf("  %s: %s\n", col["Field"], col["Type"])
	}

	// 查询技术指标
	fmt.Println("\n=== FILUSDT技术指标 ===")
	var techResult map[string]interface{}
	techQuery := `
		SELECT *
		FROM technical_indicators_caches
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`
	db.Raw(techQuery).Scan(&techResult)

	if len(techResult) > 0 {
		fmt.Printf("技术指标数据:\n")
		for k, v := range techResult {
			if k != "symbol" && k != "created_at" && k != "updated_at" {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	} else {
		fmt.Printf("❌ 没有找到FILUSDT的技术指标数据\n")
		fmt.Printf("这可能是网格策略没有交易的主要原因！\n")
	}

	// 检查策略执行日志
	fmt.Println("\n=== 策略执行日志分析 ===")
	fmt.Println("从用户提供的日志看:")
	fmt.Println("1. ✅ 策略调度器开始执行")
	fmt.Println("2. ✅ 使用币种白名单模式")
	fmt.Println("3. ✅ 市场数据获取成功")
	fmt.Println("4. ❌ 最终统计: 创建0个订单")

	fmt.Println("\n🎯 可能的根本原因:")
	fmt.Println("1. 技术指标数据缺失 - 导致评分计算失败")
	fmt.Println("2. 评分阈值设置过高 - 需要达到0.5或-0.5才交易")
	fmt.Println("3. 趋势过滤器阻止交易")
	fmt.Println("4. 风险管理器阻止交易")
}
