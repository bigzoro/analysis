package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 从环境变量获取数据库连接信息
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("=== 涨幅榜重复数据检查 ===")

	// 检查快照数量
	var snapshotCount int64
	err = db.Model(&RealtimeGainersSnapshot{}).Where("kind = ?", "spot").Count(&snapshotCount).Error
	if err != nil {
		log.Fatal("查询快照数量失败:", err)
	}
	fmt.Printf("现货涨幅榜快照数量: %d\n", snapshotCount)

	// 检查最新的快照
	var latestSnapshot struct {
		ID        uint      `json:"id"`
		Kind      string    `json:"kind"`
		Timestamp time.Time `json:"timestamp"`
	}
	err = db.Model(&RealtimeGainersSnapshot{}).
		Where("kind = ?", "spot").
		Order("timestamp DESC").
		First(&latestSnapshot).Error
	if err != nil {
		log.Fatal("查询最新快照失败:", err)
	}
	fmt.Printf("最新快照ID: %d, 时间: %v\n", latestSnapshot.ID, latestSnapshot.Timestamp)

	// 检查该快照下的数据量
	var itemCount int64
	err = db.Model(&RealtimeGainersItem{}).Where("snapshot_id = ?", latestSnapshot.ID).Count(&itemCount).Error
	if err != nil {
		log.Fatal("查询涨幅榜数据量失败:", err)
	}
	fmt.Printf("最新快照下的数据量: %d\n", itemCount)

	// 检查是否有重复的币种
	var duplicates []struct {
		Symbol   string `json:"symbol"`
		Count    int64  `json:"count"`
	}
	err = db.Model(&RealtimeGainersItem{}).
		Select("symbol, COUNT(*) as count").
		Where("snapshot_id = ?", latestSnapshot.ID).
		Group("symbol").
		Having("COUNT(*) > 1").
		Scan(&duplicates).Error

	if err != nil {
		log.Fatal("检查重复数据失败:", err)
	}

	if len(duplicates) > 0 {
		fmt.Printf("\n❌ 发现重复数据:\n")
		for _, dup := range duplicates {
			fmt.Printf("  币种: %s, 重复次数: %d\n", dup.Symbol, dup.Count)
		}
	} else {
		fmt.Println("\n✅ 未发现重复数据")
	}

	// 模拟前端查询
	fmt.Println("\n=== 模拟前端查询结果 ===")
	query := `
		SELECT
			i.symbol,
			i.rank,
			i.current_price,
			i.price_change24h
		FROM realtime_gainers_items i
		WHERE i.snapshot_id = (
			SELECT s.id
			FROM realtime_gainers_snapshots s
			WHERE s.kind = 'spot'
			ORDER BY s.timestamp DESC
			LIMIT 1
		)
		ORDER BY i.rank ASC
		LIMIT 10
	`

	var results []struct {
		Symbol         string  `json:"symbol"`
		Rank           int     `json:"rank"`
		CurrentPrice   float64 `json:"current_price"`
		PriceChange24h float64 `json:"price_change24h"`
	}

	err = db.Raw(query).Scan(&results).Error
	if err != nil {
		log.Fatal("前端查询模拟失败:", err)
	}

	fmt.Printf("前端查询返回 %d 条记录:\n", len(results))
	for i, r := range results {
		fmt.Printf("  %d. %s (排名:%d, 价格:%.2f, 涨幅:%.2f%%)\n",
			i+1, r.Symbol, r.Rank, r.CurrentPrice, r.PriceChange24h)
	}
}

// 为了编译需要添加的结构体定义
type RealtimeGainersSnapshot struct {
	ID        uint      `gorm:"primarykey"`
	Kind      string    `gorm:"size:16;not null"`
	Timestamp time.Time `gorm:"not null"`
}

type RealtimeGainersItem struct {
	ID             uint      `gorm:"primarykey"`
	SnapshotID     uint      `gorm:"not null"`
	Symbol         string    `gorm:"size:32;not null"`
	Rank           int       `gorm:"not null"`
	CurrentPrice   float64   `gorm:"type:decimal(20,8)"`
	PriceChange24h float64   `gorm:"type:decimal(10,4)"`
	Volume24h      float64   `gorm:"type:decimal(30,8)"`
	DataSource     string    `gorm:"size:16"`
	CreatedAt      time.Time
}