package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	var count int64
	db.Model(&pdb.AsyncBacktestRecord{}).Count(&count)
	fmt.Printf("Total async backtest records: %d\n", count)

	if count > 0 {
		var records []pdb.AsyncBacktestRecord
		db.Order("id DESC").Limit(10).Find(&records)
		fmt.Println("Recent records:")
		for _, r := range records {
			fmt.Printf("ID: %d, UserID: %d, Symbol: %s, Status: %s, CreatedAt: %s\n",
				r.ID, r.UserID, r.Symbol, r.Status, r.CreatedAt)
		}
	}
}
