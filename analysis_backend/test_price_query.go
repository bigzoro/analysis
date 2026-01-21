package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	fmt.Println("🔍 测试价格变化查询问题")
	fmt.Println("========================")

	// 获取数据库连接信息
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = ""
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "analysis"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 测试日志中提到的币种
	testSymbols := []string{
		"ATOMUSDT", "ALGOUSDT", "CAKEUSDT", "ROSEUSDT", "GRTUSDT",
		"ACHUSDT", "IMXUSDT", "SYRUPUSDT", "USTCUSDT", "DATAUSDT",
	}

	fmt.Println("\n🧪 测试价格变化查询:")
	fmt.Println("币种\t\tSpot市场\tFutures市场\t任意市场\t最新时间")
	fmt.Println("----\t\t--------\t-----------\t--------\t--------")

	for _, symbol := range testSymbols {
		fmt.Printf("%s\t", symbol)

		// 1. 查询spot市场
		var spotCount int64
		err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE symbol = ? AND market_type = 'spot'", symbol).Scan(&spotCount).Error
		if err != nil {
			fmt.Printf("❌\t\t")
		} else {
			fmt.Printf("%d\t\t", spotCount)
		}

		// 2. 查询futures市场
		var futuresCount int64
		err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE symbol = ? AND market_type = 'futures'", symbol).Scan(&futuresCount).Error
		if err != nil {
			fmt.Printf("❌\t\t")
		} else {
			fmt.Printf("%d\t\t", futuresCount)
		}

		// 3. 查询任意市场
		var anyCount int64
		err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE symbol = ?", symbol).Scan(&anyCount).Error
		if err != nil {
			fmt.Printf("❌\t\t")
		} else {
			fmt.Printf("%d\t\t", anyCount)
		}

		// 4. 获取最新数据时间
		if anyCount > 0 {
			var latestTime string
			err = db.Raw("SELECT MAX(created_at) FROM binance_24h_stats WHERE symbol = ?", symbol).Scan(&latestTime).Error
			if err != nil {
				fmt.Printf("❌")
			} else {
				fmt.Printf("%s", latestTime[:19]) // 只显示日期时间部分
			}
		} else {
			fmt.Printf("无数据")
		}

		fmt.Println()
	}

	// 检查表结构
	fmt.Println("\n📋 检查binance_24h_stats表结构:")
	var columns []struct {
		Field   string
		Type    string
		Null    string
		Key     string
		Default *string
		Extra   string
	}
	err = db.Raw("DESCRIBE binance_24h_stats").Scan(&columns).Error
	if err != nil {
		log.Printf("❌ 获取表结构失败: %v", err)
	} else {
		fmt.Println("字段名\t\t\t类型\t\t可空")
		fmt.Println("------\t\t\t----\t\t----")
		for _, col := range columns {
			fmt.Printf("%-20s\t%-15s\t%s\n", col.Field, col.Type, col.Null)
		}
	}

	fmt.Println("\n🎯 分析结果:")
	fmt.Println("• 如果Spot/Futures/任意市场都是0，说明币种在Binance上没有交易数据")
	fmt.Println("• 如果有数据但查询仍然失败，可能是数据类型或时间过滤问题")
	fmt.Println("• 建议检查数据同步服务是否正常运行")
}