package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 价格获取调试 ===")
	fmt.Println("检查为什么无法获取FILUSDT价格")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 检查binance_24h_stats表中的FILUSDT数据
	fmt.Println("\n📊 第一阶段: 检查24小时统计数据")
	var stats []map[string]interface{}
	db.Raw(`
		SELECT symbol, last_price, volume, created_at
		FROM binance_24h_stats
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 5
	`).Scan(&stats)

	fmt.Printf("FILUSDT最近5条24h统计数据:\n")
	for i, stat := range stats {
		fmt.Printf("%d. 价格: %v, 成交量: %v, 时间: %v\n",
			i+1, stat["last_price"], stat["volume"], stat["created_at"])
	}

	// 2. 检查是否有其他USDT交易对的数据
	fmt.Println("\n📊 第二阶段: 检查其他交易对数据")
	var otherStats []map[string]interface{}
	db.Raw(`
		SELECT symbol, last_price, volume
		FROM binance_24h_stats
		WHERE symbol LIKE '%USDT'
		ORDER BY created_at DESC
		LIMIT 5
	`).Scan(&otherStats)

	fmt.Printf("其他USDT交易对数据:\n")
	for _, stat := range otherStats {
		fmt.Printf("  %s: 价格=%v, 成交量=%v\n",
			stat["symbol"], stat["last_price"], stat["volume"])
	}

	// 3. 检查数据同步状态
	fmt.Println("\n📊 第三阶段: 检查数据同步状态")
	var syncStatus map[string]interface{}
	db.Raw(`
		SELECT COUNT(*) as total_symbols,
			   SUM(CASE WHEN last_price > 0 THEN 1 ELSE 0 END) as valid_prices,
			   MAX(created_at) as latest_update
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
	`).Scan(&syncStatus)

	fmt.Printf("数据同步状态 (最近1小时):\n")
	fmt.Printf("  总交易对数: %v\n", syncStatus["total_symbols"])
	fmt.Printf("  有有效价格的交易对: %v\n", syncStatus["valid_prices"])
	fmt.Printf("  最新更新时间: %v\n", syncStatus["latest_update"])

	// 4. 检查数据同步服务的运行状态
	fmt.Println("\n📊 第四阶段: 检查数据同步配置")

	// 尝试直接查询数据库中的表结构
	fmt.Println("\n📊 第五阶段: 检查表结构")
	var tables []string
	db.Raw("SHOW TABLES LIKE 'binance_24h_stats'").Scan(&tables)

	if len(tables) > 0 {
		fmt.Printf("✅ binance_24h_stats表存在\n")

		// 检查表结构
		var columns []map[string]interface{}
		db.Raw("DESCRIBE binance_24h_stats").Scan(&columns)

		fmt.Printf("表结构:\n")
		for _, col := range columns {
			fmt.Printf("  %s: %s\n", col["Field"], col["Type"])
		}
	} else {
		fmt.Printf("❌ binance_24h_stats表不存在\n")
	}

	// 5. 诊断结论
	fmt.Println("\n📊 第六阶段: 诊断结论")

	if len(stats) == 0 {
		fmt.Printf("❌ 诊断: FILUSDT没有任何24h统计数据\n")
		fmt.Printf("💡 建议: 检查数据同步服务是否正常运行\n")
	} else {
		validPrices := 0
		for _, stat := range stats {
			if price, ok := stat["last_price"].(float64); ok && price > 0 {
				validPrices++
			}
		}

		if validPrices == 0 {
			fmt.Printf("❌ 诊断: FILUSDT有%d条记录，但所有价格都是0或无效\n", len(stats))
			fmt.Printf("💡 建议: 检查币安API数据同步是否有问题\n")
		} else {
			fmt.Printf("✅ 诊断: FILUSDT有有效价格数据\n")
			fmt.Printf("💡 建议: 检查策略代码中的价格获取逻辑\n")
		}
	}

	// 6. 提供解决方案
	fmt.Println("\n🔧 解决方案建议:")
	fmt.Printf("1. 启动数据同步服务确保价格数据是最新的\n")
	fmt.Printf("2. 检查币安API密钥是否有效\n")
	fmt.Printf("3. 验证网络连接是否正常\n")
	fmt.Printf("4. 检查数据同步服务的日志\n")
}
