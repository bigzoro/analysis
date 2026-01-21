package main

import (
	"fmt"
	"math"
	"time"
)

// 复制修复后的函数进行测试
func analyzeTrendAndOscillationFixed(klines []struct {
	Symbol string
	Close  float64
	Time   time.Time
}) (string, float64) {
	if len(klines) < 10 {
		return "数据不足", 0
	}

	// 按币种分组数据，避免混合计算导致的错误
	symbolData := make(map[string][]float64)
	for _, kline := range klines {
		if symbolData[kline.Symbol] == nil {
			symbolData[kline.Symbol] = []float64{}
		}
		symbolData[kline.Symbol] = append(symbolData[kline.Symbol], kline.Close)
	}

	// 计算每个币种的趋势和震荡度
	totalOscillation := 0.0
	totalTrendScore := 0.0
	symbolCount := 0

	for _, prices := range symbolData {
		if len(prices) < 5 {
			continue
		}

		// 计算该币种的趋势得分（-1到1之间，负数表示下跌趋势）
		firstPrice := prices[0]
		lastPrice := prices[len(prices)-1]
		trendChange := (lastPrice - firstPrice) / firstPrice
		totalTrendScore += trendChange

		// 计算该币种的震荡度（使用标准差相对均值，更合理）
		oscillation := calculateSymbolOscillationFixed(prices)
		totalOscillation += oscillation

		symbolCount++
	}

	// 计算平均趋势得分和震荡度
	avgTrendScore := 0.0
	avgOscillation := 0.0

	if symbolCount > 0 {
		avgTrendScore = totalTrendScore / float64(symbolCount)
		avgOscillation = totalOscillation / float64(symbolCount)
	}

	// 基于平均趋势得分判断整体趋势（更合理的阈值）
	trend := "震荡"
	if avgTrendScore > 0.03 { // 平均上涨3%以上
		trend = "上涨"
	} else if avgTrendScore < -0.03 { // 平均下跌3%以上
		trend = "下跌"
	}

	return trend, avgOscillation
}

// 计算单个币种的震荡度
func calculateSymbolOscillationFixed(prices []float64) float64 {
	if len(prices) < 3 {
		return 0
	}

	// 计算价格的标准差相对均值
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	mean := sum / float64(len(prices))

	sumSquares := 0.0
	for _, price := range prices {
		sumSquares += math.Pow(price-mean, 2)
	}
	stdDev := math.Sqrt(sumSquares / float64(len(prices)))

	// 震荡度 = (标准差 / 均值) * 100，限制最大值为20%（避免极端值）
	oscillation := (stdDev / mean) * 100
	if oscillation > 20 {
		oscillation = 20
	}

	return oscillation
}

func main() {
	fmt.Println("🧪 测试修复后的市场分析算法")
	fmt.Println("================================")

	// 模拟一些测试数据（基于实际的币种数据模式）
	testKlines := []struct {
		Symbol string
		Close  float64
		Time   time.Time
	}{
		// BTCUSDT - 轻微下跌
		{"BTCUSDT", 95000, time.Now().AddDate(0, 0, -7)},
		{"BTCUSDT", 94500, time.Now().AddDate(0, 0, -6)},
		{"BTCUSDT", 94000, time.Now().AddDate(0, 0, -5)},
		{"BTCUSDT", 93500, time.Now().AddDate(0, 0, -4)},
		{"BTCUSDT", 93000, time.Now().AddDate(0, 0, -3)},
		{"BTCUSDT", 93200, time.Now().AddDate(0, 0, -2)},
		{"BTCUSDT", 92800, time.Now().AddDate(0, 0, -1)},

		// ETHUSDT - 轻微下跌
		{"ETHUSDT", 3400, time.Now().AddDate(0, 0, -7)},
		{"ETHUSDT", 3380, time.Now().AddDate(0, 0, -6)},
		{"ETHUSDT", 3350, time.Now().AddDate(0, 0, -5)},
		{"ETHUSDT", 3330, time.Now().AddDate(0, 0, -4)},
		{"ETHUSDT", 3310, time.Now().AddDate(0, 0, -3)},
		{"ETHUSDT", 3320, time.Now().AddDate(0, 0, -2)},
		{"ETHUSDT", 3290, time.Now().AddDate(0, 0, -1)},

		// ADAUSDT - 小幅上涨
		{"ADAUSDT", 0.45, time.Now().AddDate(0, 0, -7)},
		{"ADAUSDT", 0.46, time.Now().AddDate(0, 0, -6)},
		{"ADAUSDT", 0.47, time.Now().AddDate(0, 0, -5)},
		{"ADAUSDT", 0.46, time.Now().AddDate(0, 0, -4)},
		{"ADAUSDT", 0.48, time.Now().AddDate(0, 0, -3)},
		{"ADAUSDT", 0.47, time.Now().AddDate(0, 0, -2)},
		{"ADAUSDT", 0.46, time.Now().AddDate(0, 0, -1)},
	}

	trend, oscillation := analyzeTrendAndOscillationFixed(testKlines)

	fmt.Printf("📊 测试结果:\n")
	fmt.Printf("   市场趋势: %s\n", trend)
	fmt.Printf("   震荡度: %.2f%%\n", oscillation)

	// 测试策略评分
	fmt.Println("\n🎯 策略评分测试:")
	fmt.Println("=================")

	// 均值回归策略评分
	mrScore := 5
	if oscillation > 60 {
		mrScore = 9
	} else if oscillation > 40 {
		mrScore = 7
	}
	fmt.Printf("均值回归策略: %d分\n", mrScore)

	// 网格策略评分
	gridScore := 6.0
	if trend == "震荡" {
		gridScore += 3
	} else if trend == "混合" {
		gridScore += 1
	} else {
		gridScore -= 2
	}

	// 模拟低波动率环境
	volatility := 4.25
	if volatility < 30 {
		gridScore += 1
	}

	fmt.Printf("网格策略: %.0f分\n", gridScore)

	winner := "均值回归策略"
	if int(gridScore) > mrScore {
		winner = "网格策略"
	}

	fmt.Printf("\n🏆 排名第一: %s\n", winner)

	if winner == "网格策略" {
		fmt.Println("✅ 修复成功！网格策略现在正确排名第一")
	} else {
		fmt.Println("❌ 修复可能仍有问题，需要进一步调整")
	}
}