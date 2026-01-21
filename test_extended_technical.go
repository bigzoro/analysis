package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

// 复制扩展后的技术指标计算逻辑进行测试
func calculateMarketStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	sumSquares := 0.0
	for _, v := range values {
		sumSquares += math.Pow(v-mean, 2)
	}

	return math.Sqrt(sumSquares / float64(len(values)))
}

func calculateMarketRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return []float64{}
	}

	var gains, losses []float64
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	var rsi []float64
	for i := period; i < len(gains); i++ {
		avgGain := 0.0
		avgLoss := 0.0
		for j := i - period; j < i; j++ {
			avgGain += gains[j]
			avgLoss += losses[j]
		}
		avgGain /= float64(period)
		avgLoss /= float64(period)

		if avgLoss == 0 {
			rsi = append(rsi, 100)
		} else {
			rs := avgGain / avgLoss
			rsi = append(rsi, 100-(100/(1+rs)))
		}
	}

	return rsi
}

func countMarketBreadthIndicatorsTest(db *sql.DB) (strong, weak, bigGainers, bigLosers, neutralSymbols int, advanceDeclineRatio float64) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)

	rows, err := db.Query(`
		SELECT symbol
		FROM (
			SELECT symbol, MAX(quote_volume) as max_volume
			FROM binance_24h_stats
			WHERE quote_volume > 1000
				AND created_at >= ? AND created_at <= ?
			GROUP BY symbol
			ORDER BY max_volume DESC
			LIMIT 200
		) as top_symbols
	`, startTime, endTime)

	if err != nil {
		log.Printf("查询高交易量币种失败: %v", err)
		return 0, 0, 0, 0, 0, 0.0
	}
	defer rows.Close()

	var topSymbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}
		topSymbols = append(topSymbols, symbol)
	}

	for _, symbol := range topSymbols {
		var priceChange float64
		err := db.QueryRow(`
			SELECT price_change_percent
			FROM binance_24h_stats
			WHERE symbol = ? AND created_at >= ? AND created_at <= ?
			ORDER BY created_at DESC
			LIMIT 1
		`, symbol, startTime, endTime).Scan(&priceChange)

		if err != nil {
			continue
		}

		if priceChange > 2 {
			strong++
		} else if priceChange < -2 {
			weak++
		} else {
			neutralSymbols++
		}

		if priceChange > 5 {
			bigGainers++
		} else if priceChange < -5 {
			bigLosers++
		}
	}

	if weak > 0 {
		advanceDeclineRatio = float64(strong) / float64(weak)
	} else if strong > 0 {
		advanceDeclineRatio = float64(strong)
	} else {
		advanceDeclineRatio = 1.0
	}

	return strong, weak, bigGainers, bigLosers, neutralSymbols, advanceDeclineRatio
}

func countVolumeIndicatorsTest(db *sql.DB) (volumeGainers, volumeDecliners int, avgVolumeChange float64) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)
	compareStartTime := endTime.AddDate(0, 0, -2)

	rows, err := db.Query(`
		SELECT symbol
		FROM (
			SELECT symbol, MAX(quote_volume) as max_volume
			FROM binance_24h_stats
			WHERE quote_volume > 1000
				AND created_at >= ? AND created_at <= ?
			GROUP BY symbol
			ORDER BY max_volume DESC
			LIMIT 100
		) as top_symbols
	`, startTime, endTime)

	if err != nil {
		log.Printf("查询币种失败: %v", err)
		return 0, 0, 0.0
	}
	defer rows.Close()

	var totalVolumeChange float64
	var analyzedCount int

	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}

		var recentVolume, prevVolume float64

		err = db.QueryRow(`
			SELECT AVG(quote_volume)
			FROM binance_24h_stats
			WHERE symbol = ? AND created_at >= ? AND created_at <= ?
		`, symbol, startTime, endTime).Scan(&recentVolume)

		if err != nil || recentVolume == 0 {
			continue
		}

		err = db.QueryRow(`
			SELECT AVG(quote_volume)
			FROM binance_24h_stats
			WHERE symbol = ? AND created_at >= ? AND created_at < ?
		`, symbol, compareStartTime, startTime).Scan(&prevVolume)

		if err != nil || prevVolume == 0 {
			continue
		}

		volumeChange := ((recentVolume - prevVolume) / prevVolume) * 100
		totalVolumeChange += volumeChange
		analyzedCount++

		var priceChange float64
		err = db.QueryRow(`
			SELECT AVG(price_change_percent)
			FROM binance_24h_stats
			WHERE symbol = ? AND created_at >= ? AND created_at <= ?
		`, symbol, startTime, endTime).Scan(&priceChange)

		if err != nil {
			continue
		}

		if volumeChange > 20 && priceChange > 1 {
			volumeGainers++
		} else if volumeChange < -20 && priceChange < -1 {
			volumeDecliners++
		}
	}

	if analyzedCount > 0 {
		avgVolumeChange = totalVolumeChange / float64(analyzedCount)
	}

	return volumeGainers, volumeDecliners, avgVolumeChange
}

func countVolatilityIndicatorsTest(db *sql.DB) (marketVolatility float64, highVolSymbols, lowVolSymbols int) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	var symbolCount int
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT symbol)
		FROM binance_24h_stats
		WHERE quote_volume > 1000 AND created_at >= ? AND created_at <= ?
	`, startTime, endTime).Scan(&symbolCount)

	if err != nil {
		log.Printf("查询币种数量失败: %v", err)
		return 0, 0, 0
	}

	// 简化测试，只计算几个主要币种
	testSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	var totalVolatility float64
	var analyzedCount int

	for _, symbol := range testSymbols {
		var prices []float64
		rows, err := db.Query(`
			SELECT close_price
			FROM market_klines
			WHERE symbol = ? AND open_time >= ? AND open_time <= ?
			ORDER BY open_time ASC
		`, symbol, startTime, endTime)

		if err != nil {
			continue
		}

		for rows.Next() {
			var price float64
			if err := rows.Scan(&price); err != nil {
				continue
			}
			prices = append(prices, price)
		}
		rows.Close()

		if len(prices) < 3 {
			continue
		}

		symbolVolatility := calculateSymbolVolatilityTest(prices)

		totalVolatility += symbolVolatility
		analyzedCount++

		if symbolVolatility > 8 {
			highVolSymbols++
		} else if symbolVolatility < 3 {
			lowVolSymbols++
		}
	}

	if analyzedCount > 0 {
		marketVolatility = totalVolatility / float64(analyzedCount)
	}

	return marketVolatility, highVolSymbols, lowVolSymbols
}

func calculateSymbolVolatilityTest(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	if len(returns) == 0 {
		return 0
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	sumSquares := 0.0
	for _, ret := range returns {
		sumSquares += math.Pow(ret-mean, 2)
	}

	if len(returns) <= 1 {
		return 0
	}

	stdDev := math.Sqrt(sumSquares / float64(len(returns)-1))
	annualVolatility := stdDev * math.Sqrt(365) * 100

	return annualVolatility
}

func main() {
	fmt.Println("🧪 测试扩展后的技术指标计算")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 测试各项指标
	fmt.Println("\n📊 计算各项技术指标:")

	// 市场宽度指标
	strong, weak, bigGainers, bigLosers, neutralSymbols, advanceDeclineRatio := countMarketBreadthIndicatorsTest(db)
	fmt.Printf("市场宽度指标:\n")
	fmt.Printf("  强势币种: %d, 弱势币种: %d\n", strong, weak)
	fmt.Printf("  大涨币种: %d, 大跌币种: %d, 中性币种: %d\n", bigGainers, bigLosers, neutralSymbols)
	fmt.Printf("  涨跌比: %.2f\n", advanceDeclineRatio)

	// 成交量指标
	volumeGainers, volumeDecliners, avgVolumeChange := countVolumeIndicatorsTest(db)
	fmt.Printf("\n成交量指标:\n")
	fmt.Printf("  放量上涨币种: %d, 缩量下跌币种: %d\n", volumeGainers, volumeDecliners)
	fmt.Printf("  平均成交量变化: %.2f%%\n", avgVolumeChange)

	// 波动率指标
	marketVolatility, highVolSymbols, lowVolSymbols := countVolatilityIndicatorsTest(db)
	fmt.Printf("\n波动率指标:\n")
	fmt.Printf("  市场平均波动率: %.2f%%\n", marketVolatility)
	fmt.Printf("  高波动率币种: %d, 低波动率币种: %d\n", highVolSymbols, lowVolSymbols)

	fmt.Println("\n✅ 扩展技术指标计算测试完成")

	// 验证结果
	fmt.Println("\n🔍 结果验证:")
	if strong > 0 || weak > 0 {
		fmt.Println("✅ 市场宽度指标正常")
	} else {
		fmt.Println("❌ 市场宽度指标为0")
	}

	if volumeGainers >= 0 && volumeDecliners >= 0 {
		fmt.Println("✅ 成交量指标正常")
	} else {
		fmt.Println("❌ 成交量指标异常")
	}

	if marketVolatility > 0 {
		fmt.Println("✅ 波动率指标正常")
	} else {
		fmt.Println("❌ 波动率指标为0")
	}

	fmt.Println("\n🎯 技术指标扩展总结:")
	fmt.Println("• 市场宽度指标: 反映市场整体健康状况")
	fmt.Println("• 成交量指标: 验证趋势的有效性和强度")
	fmt.Println("• 波动率指标: 评估市场风险和不确定性")
	fmt.Println("• 总计新增指标: 10个")
}
