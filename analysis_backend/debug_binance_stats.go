package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 模拟Binance24hStats结构
type Binance24hStats struct {
	Symbol             string  `gorm:"size:20;not null" json:"symbol"`
	MarketType         string  `gorm:"size:10;not null" json:"market_type"`
	PriceChange        float64 `gorm:"type:decimal(20,8)" json:"price_change"`
	PriceChangePercent float64 `gorm:"type:decimal(10,4)" json:"price_change_percent"`
	WeightedAvgPrice   float64 `gorm:"type:decimal(20,8)" json:"weighted_avg_price"`
	PrevClosePrice     float64 `gorm:"type:decimal(20,8)" json:"prev_close_price"`
	LastPrice          float64 `gorm:"type:decimal(20,8)" json:"last_price"`
	LastQty            float64 `gorm:"type:decimal(20,8)" json:"last_qty"`
	BidPrice           float64 `gorm:"type:decimal(20,8)" json:"bid_price"`
	BidQty             float64 `gorm:"type:decimal(20,8)" json:"bid_qty"`
	AskPrice           float64 `gorm:"type:decimal(20,8)" json:"ask_price"`
	AskQty             float64 `gorm:"type:decimal(20,8)" json:"ask_qty"`
	OpenPrice          float64 `gorm:"type:decimal(20,8)" json:"open_price"`
	HighPrice          float64 `gorm:"type:decimal(20,8)" json:"high_price"`
	LowPrice           float64 `gorm:"type:decimal(20,8)" json:"low_price"`
	Volume             float64 `gorm:"type:decimal(20,8)" json:"volume"`
	QuoteVolume        float64 `gorm:"type:decimal(20,8)" json:"quote_volume"`
	OpenTime           int64   `gorm:"type:bigint" json:"open_time"`
	CloseTime          int64   `gorm:"type:bigint" json:"close_time"`
	FirstID            int64   `gorm:"type:bigint" json:"first_id"`
	LastID             int64   `gorm:"type:bigint" json:"last_id"`
	Count              int64   `gorm:"type:bigint" json:"count"`
}

func main() {
	fmt.Println("🔍 Binance 24h Stats 表调试工具")
	fmt.Println("================================")

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

	// 1. 检查表是否存在
	var tableExists bool
	err = db.Raw("SHOW TABLES LIKE 'binance_24h_stats'").Scan(&tableExists).Error
	if err != nil {
		log.Printf("❌ 检查表存在性失败: %v", err)
	} else {
		fmt.Println("✅ binance_24h_stats 表存在")
	}

	// 2. 检查表结构
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
		fmt.Println("\n📋 表结构:")
		for _, col := range columns {
			fmt.Printf("  %s: %s\n", col.Field, col.Type)
		}
	}

	// 3. 检查总记录数
	var totalCount int64
	err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats").Scan(&totalCount).Error
	if err != nil {
		log.Printf("❌ 获取总记录数失败: %v", err)
	} else {
		fmt.Printf("\n📊 总记录数: %d\n", totalCount)
	}

	if totalCount > 0 {
		// 4. 检查市场类型分布
		var marketTypes []struct {
			MarketType string
			Count      int64
		}
		err = db.Raw("SELECT market_type, COUNT(*) as count FROM binance_24h_stats GROUP BY market_type").Scan(&marketTypes).Error
		if err != nil {
			log.Printf("❌ 获取市场类型分布失败: %v", err)
		} else {
			fmt.Println("\n🏷️ 市场类型分布:")
			for _, mt := range marketTypes {
				fmt.Printf("  %s: %d 条记录\n", mt.MarketType, mt.Count)
			}
		}

		// 5. 检查一些热门币种
		popularSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}
		fmt.Println("\n🔍 检查热门币种数据:")
		for _, symbol := range popularSymbols {
			var count int64
			err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE symbol = ?", symbol).Scan(&count).Error
			if err != nil {
				log.Printf("❌ 检查 %s 失败: %v", symbol, err)
			} else {
				if count > 0 {
					var stats Binance24hStats
					err = db.Raw("SELECT * FROM binance_24h_stats WHERE symbol = ? ORDER BY created_at DESC LIMIT 1", symbol).Scan(&stats).Error
					if err != nil {
						fmt.Printf("  ❌ %s: 查询失败 - %v\n", symbol, err)
					} else {
						fmt.Printf("  ✅ %s: 最新价格=%.2f, 涨跌幅=%.2f%%\n", symbol, stats.LastPrice, stats.PriceChangePercent)
					}
				} else {
					fmt.Printf("  ❌ %s: 无数据\n", symbol)
				}
			}
		}

		// 6. 检查日志中提到的币种
		logSymbols := []string{"SOPHUSDT", "ROSEUSDT", "GRTUSDT", "ACHUSDT", "IMXUSDT", "SYRUPUSDT"}
		fmt.Println("\n📝 检查日志中提到的币种:")
		for _, symbol := range logSymbols {
			var count int64
			err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE symbol = ?", symbol).Scan(&count).Error
			if err != nil {
				log.Printf("❌ 检查 %s 失败: %v", symbol, err)
			} else {
				if count > 0 {
					fmt.Printf("  ✅ %s: 有 %d 条记录\n", symbol, count)
				} else {
					fmt.Printf("  ❌ %s: 无记录\n", symbol)
				}
			}
		}

		// 7. 检查最近的数据时间
		var latestTime string
		err = db.Raw("SELECT MAX(created_at) FROM binance_24h_stats").Scan(&latestTime).Error
		if err != nil {
			log.Printf("❌ 获取最新数据时间失败: %v", err)
		} else {
			fmt.Printf("\n⏰ 最新数据时间: %s\n", latestTime)
		}

		// 8. 检查数据新鲜度（最近1小时的数据）
		var recentCount int64
		err = db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)").Scan(&recentCount).Error
		if err != nil {
			log.Printf("❌ 获取最近1小时数据失败: %v", err)
		} else {
			fmt.Printf("📅 最近1小时数据: %d 条\n", recentCount)
		}

		// 9. 检查数据量最大的币种
		var topSymbols []struct {
			Symbol string
			Count  int64
		}
		err = db.Raw("SELECT symbol, COUNT(*) as count FROM binance_24h_stats GROUP BY symbol ORDER BY count DESC LIMIT 10").Scan(&topSymbols).Error
		if err != nil {
			log.Printf("❌ 获取数据量最大的币种失败: %v", err)
		} else {
			fmt.Println("\n🏆 数据量最大的币种 TOP 10:")
			for i, ts := range topSymbols {
				fmt.Printf("  %d. %s: %d 条记录\n", i+1, ts.Symbol, ts.Count)
			}
		}
	}

	fmt.Println("\n🎯 调试完成")
	fmt.Println("============")

	// 提供建议
	fmt.Println("\n💡 建议:")
	if totalCount == 0 {
		fmt.Println("• binance_24h_stats 表为空，可能数据同步有问题")
		fmt.Println("• 检查数据同步服务是否正常运行")
	} else {
		fmt.Println("• 表中有数据，检查是否按市场类型过滤")
		fmt.Println("• 确认查询的币种在Binance上是否有对应交易对")
	}
}