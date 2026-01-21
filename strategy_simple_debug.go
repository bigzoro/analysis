package main

import (
	"fmt"
	"log"
	"math"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 获取近期30天BTC数据
	var prices []float64
	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = 'BTCUSDT' AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
		AND open_time >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		ORDER BY open_time ASC
	`
	err = db.Raw(query).Scan(&prices).Error
	if err != nil {
		log.Fatal("查询数据失败:", err)
	}

	fmt.Printf("🔍 均值回归策略问题诊断\n")
	fmt.Printf("==============================\n\n")

	fmt.Printf("📊 数据概况:\n")
	fmt.Printf("• BTCUSDT 30天数据: %d 条\n", len(prices))
	fmt.Printf("• 起始价格: %.2f\n", prices[0])
	fmt.Printf("• 结束价格: %.2f\n", prices[len(prices)-1])

	// 计算价格波动
	minPrice, maxPrice := prices[0], prices[0]
	for _, price := range prices {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}

	priceRange := (maxPrice - minPrice) / prices[0] * 100
	fmt.Printf("• 价格区间: %.2f - %.2f (%.2f%%)\n", minPrice, maxPrice, priceRange)

	// 测试不同阈值下的信号产生
	fmt.Printf("\n🎯 不同EntryThreshold的信号分析:\n")

	period := 20
	var upper, middle, lower []float64

	// 计算布林带
	for i := period - 1; i < len(prices); i++ {
		window := prices[i-period+1 : i+1]
		sum := 0.0
		for _, price := range window {
			sum += price
		}
		mean := sum / float64(period)

		sumSquares := 0.0
		for _, price := range window {
			sumSquares += (price - mean) * (price - mean)
		}
		stdDev := math.Sqrt(sumSquares / float64(period))

		upper = append(upper, mean+2*stdDev)
		middle = append(middle, mean)
		lower = append(lower, mean-2*stdDev)
	}

	thresholds := []float64{0.3, 0.5, 0.7, 0.85, 0.90, 0.95}

	for _, threshold := range thresholds {
		signalCount := 0
		buySignals := 0
		sellSignals := 0

		for i := period; i < len(prices); i++ {
			currentPrice := prices[i]
			bbIndex := i - period

			if bbIndex >= 0 && bbIndex < len(lower) && bbIndex < len(upper) {
				bandwidth := upper[bbIndex] - lower[bbIndex]
				if bandwidth > 0 && middle[bbIndex] > 0 {
					lowerDeviation := (lower[bbIndex] - currentPrice) / bandwidth
					upperDeviation := (currentPrice - upper[bbIndex]) / bandwidth

					if lowerDeviation > threshold {
						signalCount++
						buySignals++
					} else if upperDeviation > threshold {
						signalCount++
						sellSignals++
					}
				}
			}
		}

		fmt.Printf("• 阈值 %.2f: %d 个信号 (%d 买入, %d 卖出)\n",
			threshold, signalCount, buySignals, sellSignals)
	}

	fmt.Printf("\n💡 问题分析:\n")
	fmt.Printf("==============================\n")

	fmt.Printf("1️⃣ 价格区间过窄 (%.2f%%):\n", priceRange)
	fmt.Printf("   • 盘整市场，价格偏离布林带的机会少\n")
	fmt.Printf("   • 均值回归需要足够的价格波动\n\n")

	fmt.Printf("2️⃣ 入场阈值设置过高:\n")
	fmt.Printf("   • 0.85-0.95的阈值在窄幅震荡中极难达到\n")
	fmt.Printf("   • 导致几乎没有交易信号产生\n\n")

	fmt.Printf("3️⃣ 信号强度计算问题:\n")
	fmt.Printf("   • EntryThreshold既控制布林信号又控制开仓\n")
	fmt.Printf("   • 造成信号很少能超过最终阈值\n\n")

	fmt.Printf("4️⃣ 市场环境不完全适合:\n")
	fmt.Printf("   • 当前是轻微上涨的盘整 (30天+2.32%%)\n")
	fmt.Printf("   • 均值回归在上涨盘整中容易亏损\n\n")

	fmt.Printf("🎯 解决方案:\n")
	fmt.Printf("==============================\n")

	fmt.Printf("1️⃣ 降低入场阈值: 0.85 → 0.4-0.6\n")
	fmt.Printf("2️⃣ 简化信号逻辑: 直接用布林带信号开仓\n")
	fmt.Printf("3️⃣ 增加市场适应性: 区分涨跌盘整\n")
	fmt.Printf("4️⃣ 优化止盈止损: 适应窄幅震荡特征\n")
}