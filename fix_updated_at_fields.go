package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Binance24hStats 24小时统计数据模型
type Binance24hStats struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	Symbol             string    `gorm:"size:20;not null" json:"symbol"`
	MarketType         string    `gorm:"size:10;not null" json:"market_type"`
	PriceChange        float64   `gorm:"type:decimal(20,8)" json:"price_change"`
	PriceChangePercent float64   `gorm:"type:decimal(10,4)" json:"price_change_percent"`
	WeightedAvgPrice   float64   `gorm:"type:decimal(20,8)" json:"weighted_avg_price"`
	PrevClosePrice     float64   `gorm:"type:decimal(20,8)" json:"prev_close_price"`
	LastPrice          float64   `gorm:"type:decimal(20,8)" json:"last_price"`
	LastQty            float64   `gorm:"type:decimal(20,8)" json:"last_qty"`
	BidPrice           float64   `gorm:"type:decimal(20,8)" json:"bid_price"`
	BidQty             float64   `gorm:"type:decimal(20,8)" json:"bid_qty"`
	AskPrice           float64   `gorm:"type:decimal(20,8)" json:"ask_price"`
	AskQty             float64   `gorm:"type:decimal(20,8)" json:"ask_qty"`
	OpenPrice          float64   `gorm:"type:decimal(20,8)" json:"open_price"`
	HighPrice          float64   `gorm:"type:decimal(20,8)" json:"high_price"`
	LowPrice           float64   `gorm:"type:decimal(20,8)" json:"low_price"`
	Volume             float64   `gorm:"type:decimal(30,8)" json:"volume"`
	QuoteVolume        float64   `gorm:"type:decimal(30,8)" json:"quote_volume"`
	OpenTime           int64     `json:"open_time"`
	CloseTime          int64     `json:"close_time"`
	FirstId            int64     `json:"first_id"`
	LastId             int64     `json:"last_id"`
	Count              int64     `json:"count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Binance24hStats) TableName() string {
	return "binance_24h_stats"
}

// BinanceFundingRate 资金费率数据模型
type BinanceFundingRate struct {
	ID                   uint      `gorm:"primarykey" json:"id"`
	Symbol               string    `gorm:"size:20;not null" json:"symbol"`
	FundingRate          float64   `gorm:"type:decimal(10,8);not null" json:"funding_rate"`
	FundingTime          int64     `gorm:"not null" json:"funding_time"`
	MarkPrice            float64   `gorm:"type:decimal(20,8)" json:"mark_price"`
	IndexPrice           float64   `gorm:"type:decimal(20,8)" json:"index_price"`
	EstimatedSettlePrice float64   `gorm:"type:decimal(20,8)" json:"estimated_settle_price"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (BinanceFundingRate) TableName() string {
	return "binance_funding_rates"
}

func main() {
	fmt.Println("=== 通过GORM AutoMigrate修复updated_at字段 ===")

	// 从环境变量获取数据库连接信息
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	}

	fmt.Printf("连接数据库: %s\n", dsn)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 使用AutoMigrate来更新表结构
	fmt.Println("\n=== 执行AutoMigrate ===")
	err = db.AutoMigrate(&Binance24hStats{}, &BinanceFundingRate{})
	if err != nil {
		log.Fatalf("AutoMigrate失败: %v", err)
	}

	fmt.Println("✅ AutoMigrate执行成功")

	// 检查字段定义
	fmt.Println("\n=== 检查字段定义 ===")

	var tables = []string{"binance_24h_stats", "binance_funding_rates"}
	for _, table := range tables {
		fmt.Printf("\n--- 表: %s ---\n", table)

		type ColumnInfo struct {
			Field   string `json:"Field"`
			Type    string `json:"Type"`
			Null    string `json:"Null"`
			Default string `json:"Default"`
			Extra   string `json:"Extra"`
		}

		var columns []ColumnInfo
		err := db.Raw("DESCRIBE " + table).Scan(&columns).Error
		if err != nil {
			log.Printf("获取表结构失败 %s: %v", table, err)
			continue
		}

		for _, col := range columns {
			if col.Field == "updated_at" {
				fmt.Printf("updated_at字段信息:\n")
				fmt.Printf("  类型: %s\n", col.Type)
				fmt.Printf("  可空: %s\n", col.Null)
				fmt.Printf("  默认值: %s\n", col.Default)
				fmt.Printf("  额外属性: %s\n", col.Extra)

				// 检查是否正确设置了自动更新
				if col.Type == "timestamp" && col.Extra == "DEFAULT_GENERATED on update CURRENT_TIMESTAMP" {
					fmt.Printf("✅ 字段定义正确！\n")
				} else {
					fmt.Printf("⚠️  字段定义可能需要手动调整\n")
				}
				break
			}
		}
	}

	fmt.Println("\n=== 修复完成 ===")
	fmt.Println("现在重启服务，GORM会正确处理updated_at字段的自动更新")
}
