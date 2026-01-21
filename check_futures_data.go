package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     false,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * 60 * 1000000000, // 30分钟
		ConnMaxIdleTime: 10 * 60 * 1000000000, // 10分钟
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// 检查期货合约表
	fmt.Println("=== 检查 binance_futures_contracts 表 ===")
	var contractCount int64
	if err := gdb.Model(&pdb.BinanceFuturesContract{}).Count(&contractCount).Error; err != nil {
		log.Printf("查询合约表失败: %v", err)
	} else {
		fmt.Printf("合约表记录数: %d\n", contractCount)
	}

	if contractCount > 0 {
		// 显示前5条记录
		var contracts []pdb.BinanceFuturesContract
		if err := gdb.Model(&pdb.BinanceFuturesContract{}).Limit(5).Find(&contracts).Error; err != nil {
			log.Printf("查询合约记录失败: %v", err)
		} else {
			fmt.Println("前5条合约记录:")
			for _, contract := range contracts {
				fmt.Printf("  - %s (%s)\n", contract.Symbol, contract.Status)
			}
		}
	}

	// 检查资金费率表
	fmt.Println("\n=== 检查 binance_funding_rates 表 ===")
	var fundingCount int64
	if err := gdb.Model(&pdb.BinanceFundingRate{}).Count(&fundingCount).Error; err != nil {
		log.Printf("查询资金费率表失败: %v", err)
	} else {
		fmt.Printf("资金费率表记录数: %d\n", fundingCount)
	}

	if fundingCount > 0 {
		// 显示前5条记录
		var rates []pdb.BinanceFundingRate
		if err := gdb.Model(&pdb.BinanceFundingRate{}).Limit(5).Find(&rates).Error; err != nil {
			log.Printf("查询资金费率记录失败: %v", err)
		} else {
			fmt.Println("前5条资金费率记录:")
			for _, rate := range rates {
				fmt.Printf("  - %s: %.8f\n", rate.Symbol, rate.FundingRate)
			}
		}
	}

	// 检查是否有相关的同步日志表
	fmt.Println("\n=== 检查同步统计 ===")
	type SyncStat struct {
		TableName string
		Count     int64
	}

	stats := []SyncStat{
		{"binance_futures_contracts", 0},
		{"binance_funding_rates", 0},
		{"binance_exchange_info", 0},
	}

	for i, stat := range stats {
		if err := gdb.Table(stat.TableName).Count(&stats[i].Count).Error; err != nil {
			log.Printf("查询表 %s 失败: %v", stat.TableName, err)
		}
	}

	for _, stat := range stats {
		fmt.Printf("表 %s: %d 条记录\n", stat.TableName, stat.Count)
	}
}
