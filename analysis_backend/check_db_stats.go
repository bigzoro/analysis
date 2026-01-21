package main

import (
	pdb "analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("检查数据库中的 binance_24h_stats 数据...")

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	db := gdb.DB

	// 检查表是否存在
	var tableExists bool
	err = db.Raw("SHOW TABLES LIKE 'binance_24h_stats'").Scan(&tableExists).Error
	if err != nil {
		log.Printf("检查表是否存在失败: %v", err)
	} else if !tableExists {
		fmt.Println("❌ binance_24h_stats 表不存在")
		return
	}

	// 检查总记录数
	var totalCount int64
	err = db.Model(&pdb.Binance24hStats{}).Count(&totalCount).Error
	if err != nil {
		log.Printf("查询总记录数失败: %v", err)
	} else {
		fmt.Printf("📊 binance_24h_stats 表总记录数: %d\n", totalCount)
	}

	// 检查 spot 市场数据
	var spotCount int64
	err = db.Model(&pdb.Binance24hStats{}).Where("market_type = ?", "spot").Count(&spotCount).Error
	if err != nil {
		log.Printf("查询 spot 数据失败: %v", err)
	} else {
		fmt.Printf("📊 spot 市场记录数: %d\n", spotCount)
	}

	// 检查 futures 市场数据
	var futuresCount int64
	err = db.Model(&pdb.Binance24hStats{}).Where("market_type = ?", "futures").Count(&futuresCount).Error
	if err != nil {
		log.Printf("查询 futures 数据失败: %v", err)
	} else {
		fmt.Printf("📊 futures 市场记录数: %d\n", futuresCount)
	}

	// 检查最近1小时的数据
	var recentCount int64
	err = db.Model(&pdb.Binance24hStats{}).Where("created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)").Count(&recentCount).Error
	if err != nil {
		log.Printf("查询最近1小时数据失败: %v", err)
	} else {
		fmt.Printf("📊 最近1小时记录数: %d\n", recentCount)
	}

	// 检查涨幅榜数据（涨幅 > 0 且有交易量）
	var gainersCount int64
	err = db.Model(&pdb.Binance24hStats{}).
		Where("market_type = ? AND price_change_percent > 0 AND volume > 0 AND last_price > 0 AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)", "spot").
		Count(&gainersCount).Error
	if err != nil {
		log.Printf("查询涨幅榜数据失败: %v", err)
	} else {
		fmt.Printf("📊 涨幅榜候选数 (涨幅>0): %d\n", gainersCount)
	}

	// 检查同时有spot和futures的币种
	var bothMarketsCount int64
	query := `
		SELECT COUNT(DISTINCT s.symbol) as count
		FROM binance_24h_stats s
		INNER JOIN binance_24h_stats f ON s.symbol = f.symbol AND f.market_type = 'futures'
		WHERE s.market_type = 'spot' AND s.created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			AND s.volume > 0 AND s.last_price > 0
			AND f.created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
	`
	err = db.Raw(query).Scan(&bothMarketsCount).Error
	if err != nil {
		log.Printf("查询同时有两种市场的币种失败: %v", err)
	} else {
		fmt.Printf("📊 同时有spot+futures的币种数: %d\n", bothMarketsCount)
	}

	// 显示涨幅前5的币种
	fmt.Println("\n涨幅前5的币种:")
	var topGainers []struct {
		Symbol             string  `json:"symbol"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		MarketType         string  `json:"market_type"`
	}

	err = db.Model(&pdb.Binance24hStats{}).
		Select("symbol, price_change_percent, volume, market_type").
		Where("market_type = ? AND price_change_percent > 0 AND volume > 0 AND last_price > 0 AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)", "spot").
		Order("price_change_percent DESC").
		Limit(5).
		Scan(&topGainers).Error

	if err != nil {
		log.Printf("查询涨幅前5失败: %v", err)
	} else {
		for i, gainer := range topGainers {
			fmt.Printf("  %d. %s: %.2f%% (交易量: %.0f)\n", i+1, gainer.Symbol, gainer.PriceChangePercent, gainer.Volume)
		}
	}

	// 检查资金费率数据
	var fundingCount int64
	err = db.Model(&pdb.BinanceFundingRate{}).Count(&fundingCount).Error
	if err != nil {
		log.Printf("查询资金费率数据失败: %v", err)
	} else {
		fmt.Printf("💰 资金费率记录数: %d\n", fundingCount)
	}
}
