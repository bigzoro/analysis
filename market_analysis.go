package main

import (
	"fmt"
	"log"
	"math"
	"sort"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 计算线性回归趋势
func calculateLinearTrend(prices []float64) (slope float64, r2 float64) {
	n := float64(len(prices))
	if n < 2 {
		return 0, 0
	}

	// 计算x轴 (时间序列)
	var x []float64
	for i := 0; i < len(prices); i++ {
		x = append(x, float64(i))
	}

	// 计算均值
	sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < len(prices); i++ {
		sumX += x[i]
		sumY += prices[i]
		sumXY += x[i] * prices[i]
		sumXX += x[i] * x[i]
	}

	// 计算斜率
	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 计算R² (决定系数)
	yMean := sumY / n
	ssRes, ssTot := 0.0, 0.0
	for i := 0; i < len(prices); i++ {
		predicted := slope*x[i] + (sumY - slope*sumX)/n
		ssRes += (prices[i] - predicted) * (prices[i] - predicted)
		ssTot += (prices[i] - yMean) * (prices[i] - yMean)
	}

	if ssTot != 0 {
		r2 = 1 - ssRes/ssTot
	}

	return slope, r2
}

// 计算ADX (平均定向运动指数)
func calculateADX(highs, lows, closes []float64, period int) float64 {
	if len(highs) < period*2 {
		return 0
	}

	var trs, plusDMs, minusDMs []float64

	for i := 1; i < len(highs); i++ {
		// True Range
		tr1 := highs[i] - lows[i]
		tr2 := math.Abs(highs[i] - closes[i-1])
		tr3 := math.Abs(lows[i] - closes[i-1])
		tr := math.Max(tr1, math.Max(tr2, tr3))
		trs = append(trs, tr)

		// Directional Movement
		dmPlus := highs[i] - highs[i-1]
		dmMinus := lows[i-1] - lows[i]

		plusDM := 0.0
		if dmPlus > dmMinus && dmPlus > 0 {
			plusDM = dmPlus
		}

		minusDM := 0.0
		if dmMinus > dmPlus && dmMinus > 0 {
			minusDM = dmMinus
		}

		plusDMs = append(plusDMs, plusDM)
		minusDMs = append(minusDMs, minusDM)
	}

	// 计算平均值
	avgTR := 0.0
	for i := 0; i < period && i < len(trs); i++ {
		avgTR += trs[i]
	}
	if period > 0 {
		avgTR /= float64(period)
	}

	avgPlusDM := 0.0
	for i := 0; i < period && i < len(plusDMs); i++ {
		avgPlusDM += plusDMs[i]
	}
	if period > 0 {
		avgPlusDM /= float64(period)
	}

	avgMinusDM := 0.0
	for i := 0; i < period && i < len(minusDMs); i++ {
		avgMinusDM += minusDMs[i]
	}
	if period > 0 {
		avgMinusDM /= float64(period)
	}

	// 计算DI
	plusDI := 0.0
	if avgTR > 0 {
		plusDI = (avgPlusDM / avgTR) * 100
	}

	minusDI := 0.0
	if avgTR > 0 {
		minusDI = (avgMinusDM / avgTR) * 100
	}

	// 计算ADX
	dx := 0.0
	if plusDI+minusDI > 0 {
		dx = math.Abs(plusDI-minusDI) / (plusDI + minusDI) * 100
	}

	return dx
}

// 计算最大回撤
func calculateMaxDrawdown(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}

	maxPrice := prices[0]
	maxDrawdown := 0.0

	for _, price := range prices {
		if price > maxPrice {
			maxPrice = price
		}

		if maxPrice > 0 {
			drawdown := (maxPrice - price) / maxPrice
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown * 100
}

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 分析多个主要币种
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}

	fmt.Printf("🔬 加密货币市场深度趋势分析\n")
	fmt.Printf("=====================================\n\n")

	for _, symbol := range symbols {
		fmt.Printf("📊 %s 深度趋势分析\n", symbol)
		fmt.Printf("-----------------------------\n")

		// 获取不同周期的数据
		periods := []int{90, 180, 360} // 3个月、6个月、1年

		for _, days := range periods {
			var prices []float64
			query := `
				SELECT close_price
				FROM market_klines
				WHERE symbol = ? AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
				AND open_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
				ORDER BY open_time ASC
			`
			err = db.Raw(query, symbol, days).Scan(&prices).Error
			if err != nil || len(prices) < 30 {
				fmt.Printf("• %d天数据不足\n", days)
				continue
			}

			// 基本统计
			startPrice := prices[0]
			endPrice := prices[len(prices)-1]
			totalReturn := (endPrice - startPrice) / startPrice * 100

			// 计算日收益率
			var dailyReturns []float64
			for i := 1; i < len(prices); i++ {
				ret := (prices[i] - prices[i-1]) / prices[i-1]
				dailyReturns = append(dailyReturns, ret)
			}

			// 计算波动率
			var sumSquares float64
			meanReturn := 0.0
			for _, ret := range dailyReturns {
				meanReturn += ret
			}
			meanReturn /= float64(len(dailyReturns))

			for _, ret := range dailyReturns {
				sumSquares += (ret - meanReturn) * (ret - meanReturn)
			}
			volatility := math.Sqrt(sumSquares / float64(len(dailyReturns))) * math.Sqrt(365) * 100

			// 线性回归趋势分析
			slope, r2 := calculateLinearTrend(prices)

			// 趋势强度指标
			trendStrength := math.Abs(totalReturn) / volatility
			regressionStrength := math.Abs(slope) / (volatility / 100) // 斜率相对波动率的强度

			// 最大回撤
			maxDD := calculateMaxDrawdown(prices)

			fmt.Printf("• %d天周期:\n", days)
			fmt.Printf("  总收益率: %.2f%%\n", totalReturn)
			fmt.Printf("  年化波动率: %.2f%%\n", volatility)
			fmt.Printf("  最大回撤: %.2f%%\n", maxDD)
			fmt.Printf("  线性回归R²: %.3f (%.1f%%)\n", r2, r2*100)
			fmt.Printf("  回归斜率: %.6f\n", slope)
			fmt.Printf("  趋势强度: %.2f\n", trendStrength)
			fmt.Printf("  回归强度: %.2f\n", regressionStrength)

			// 趋势判断
			trendDirection := "震荡"
			if totalReturn > 5 {
				trendDirection = "上涨"
			} else if totalReturn < -5 {
				trendDirection = "下跌"
			}

			intensity := "弱"
			if trendStrength > 1.5 {
				intensity = "极强"
			} else if trendStrength > 1.0 {
				intensity = "强"
			} else if trendStrength > 0.5 {
				intensity = "中等"
			}

			fmt.Printf("  趋势判断: %s趋势 (%s)\n", trendDirection, intensity)

			// ADX分析 (仅对360天数据)
			if days == 360 && len(prices) >= 50 {
				// 简化ADX计算 (使用收盘价作为高低价的近似)
				var highs, lows []float64
				for i, price := range prices {
					highs = append(highs, price*1.02) // 近似高价
					lows = append(lows, price*0.98)   // 近似低价
					if i > 0 {
						// 确保连续性
						highs[i] = math.Max(highs[i], highs[i-1])
						lows[i] = math.Min(lows[i], lows[i-1])
					}
				}
				adx := calculateADX(highs, lows, prices, 14)
				fmt.Printf("  ADX指标: %.2f", adx)
				if adx > 25 {
					fmt.Printf(" (强趋势)")
				} else if adx < 20 {
					fmt.Printf(" (弱趋势)")
				} else {
					fmt.Printf(" (中等趋势)")
				}
				fmt.Println()
			}
			fmt.Println()
		}
	}

	// 市场整体分析
	fmt.Printf("🌍 市场整体趋势分析\n")
	fmt.Printf("========================\n")

	// 分析主流币种的相关性
	var correlations []float64
	for i := 0; i < len(symbols)-1; i++ {
		for j := i + 1; j < len(symbols); j++ {
			var prices1, prices2 []float64
			query1 := `
				SELECT close_price FROM market_klines
				WHERE symbol = ? AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
				AND open_time >= DATE_SUB(NOW(), INTERVAL 360 DAY)
				ORDER BY open_time ASC
			`
			query2 := `
				SELECT close_price FROM market_klines
				WHERE symbol = ? AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
				AND open_time >= DATE_SUB(NOW(), INTERVAL 360 DAY)
				ORDER BY open_time ASC
			`

			db.Raw(query1, symbols[i]).Scan(&prices1)
			db.Raw(query2, symbols[j]).Scan(&prices2)

			if len(prices1) == len(prices2) && len(prices1) > 30 {
				// 计算相关系数
				corr := calculateCorrelation(prices1, prices2)
				correlations = append(correlations, corr)
			}
		}
	}

	if len(correlations) > 0 {
		avgCorr := 0.0
		for _, c := range correlations {
			avgCorr += c
		}
		avgCorr /= float64(len(correlations))

		fmt.Printf("• 主流币种平均相关性: %.3f\n", avgCorr)
		if avgCorr > 0.8 {
			fmt.Printf("• 市场特征: 高度同步 (系统性风险高)\n")
		} else if avgCorr > 0.6 {
			fmt.Printf("• 市场特征: 中等同步 (部分系统性风险)\n")
		} else {
			fmt.Printf("• 市场特征: 分散化 (个股机会多)\n")
		}
	}

	// 均值回归适应性分析
	fmt.Printf("\n🎯 均值回归策略适应性评估\n")
	fmt.Printf("==============================\n")

	var symbolScores []struct {
		symbol string
		score  float64
		reason string
	}

	for _, symbol := range symbols {
		var prices []float64
		query := `
			SELECT close_price FROM market_klines
			WHERE symbol = ? AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
			AND open_time >= DATE_SUB(NOW(), INTERVAL 360 DAY)
			ORDER BY open_time ASC
		`
		db.Raw(query, symbol).Scan(&prices)

		if len(prices) < 50 {
			continue
		}

		// 计算日收益率
		var dailyReturns []float64
		for i := 1; i < len(prices); i++ {
			ret := (prices[i] - prices[i-1]) / prices[i-1]
			dailyReturns = append(dailyReturns, ret)
		}

		// 均值回归指标
		meanReturn := 0.0
		for _, ret := range dailyReturns {
			meanReturn += ret
		}
		meanReturn /= float64(len(dailyReturns))

		// 计算偏离程度
		deviations := 0.0
		for _, ret := range dailyReturns {
			deviations += math.Abs(ret - meanReturn)
		}
		avgDeviation := deviations / float64(len(dailyReturns))

		// 趋势强度
		totalReturn := (prices[len(prices)-1] - prices[0]) / prices[0]
		trendStrength := math.Abs(totalReturn) / (avgDeviation * math.Sqrt(365) * 100)

		// 评分 (0-100, 越高越适合均值回归)
		score := 0.0
		reason := ""

		if trendStrength < 0.5 {
			score += 40 // 弱趋势
			reason += "弱趋势 "
		} else if trendStrength < 1.0 {
			score += 20 // 中等趋势
			reason += "中等趋势 "
		} else {
			score += 0 // 强趋势
			reason += "强趋势 "
		}

		if avgDeviation > 0.02 {
			score += 30 // 高波动
			reason += "高波动 "
		} else if avgDeviation > 0.01 {
			score += 20 // 中等波动
			reason += "中等波动 "
		} else {
			score += 10 // 低波动
			reason += "低波动 "
		}

		// 反转频率 (简化计算)
		reversals := 0
		for i := 1; i < len(dailyReturns)-1; i++ {
			if (dailyReturns[i-1] > 0 && dailyReturns[i] < 0) ||
			   (dailyReturns[i-1] < 0 && dailyReturns[i] > 0) {
				reversals++
			}
		}
		reversalRate := float64(reversals) / float64(len(dailyReturns)-2)

		if reversalRate > 0.3 {
			score += 30 // 高反转率
			reason += "高反转率"
		} else if reversalRate > 0.2 {
			score += 20 // 中等反转率
			reason += "中等反转率"
		} else {
			score += 10 // 低反转率
			reason += "低反转率"
		}

		symbolScores = append(symbolScores, struct {
			symbol string
			score  float64
			reason string
		}{symbol, score, reason})
	}

	// 排序输出
	sort.Slice(symbolScores, func(i, j int) bool {
		return symbolScores[i].score > symbolScores[j].score
	})

	fmt.Printf("币种均值回归适应性排名:\n")
	for i, s := range symbolScores {
		suitability := "不适合"
		if s.score >= 70 {
			suitability = "非常适合"
		} else if s.score >= 50 {
			suitability = "较适合"
		} else if s.score >= 30 {
			suitability = "一般"
		}

		fmt.Printf("%d. %s: %.1f分 (%s) - %s\n", i+1, s.symbol, s.score, suitability, s.reason)
	}

	fmt.Printf("\n📊 最终结论\n")
	fmt.Printf("==============\n")

	// 整体市场趋势判断
	btcTrendStrength := 0.0
	if len(symbolScores) > 0 {
		// 简化：用第一个币种(BTC)的趋势强度作为代表
		var btcPrices []float64
		db.Raw(`
			SELECT close_price FROM market_klines
			WHERE symbol = 'BTCUSDT' AND kind = 'spot' AND `+"`interval`"+` = '1d'
			AND open_time >= DATE_SUB(NOW(), INTERVAL 360 DAY)
			ORDER BY open_time ASC
		`).Scan(&btcPrices)

		if len(btcPrices) >= 2 {
			totalReturn := (btcPrices[len(btcPrices)-1] - btcPrices[0]) / btcPrices[0] * 100

			var dailyReturns []float64
			for i := 1; i < len(btcPrices); i++ {
				ret := (btcPrices[i] - btcPrices[i-1]) / btcPrices[i-1]
				dailyReturns = append(dailyReturns, ret)
			}

			var sumSquares float64
			meanReturn := 0.0
			for _, ret := range dailyReturns {
				meanReturn += ret
			}
			meanReturn /= float64(len(dailyReturns))

			for _, ret := range dailyReturns {
				sumSquares += (ret - meanReturn) * (ret - meanReturn)
			}
			volatility := math.Sqrt(sumSquares / float64(len(dailyReturns))) * math.Sqrt(365) * 100

			btcTrendStrength = math.Abs(totalReturn) / volatility
		}
	}

	if btcTrendStrength > 1.5 {
		fmt.Printf("❌ 市场判断: 极强趋势市场\n")
		fmt.Printf("   均值回归策略完全不适用\n")
		fmt.Printf("   建议: 转型为趋势跟随策略\n")
	} else if btcTrendStrength > 1.0 {
		fmt.Printf("⚠️ 市场判断: 强趋势市场\n")
		fmt.Printf("   均值回归策略高风险\n")
		fmt.Printf("   建议: 大幅调整参数或考虑其他策略\n")
	} else if btcTrendStrength > 0.5 {
		fmt.Printf("🟡 市场判断: 中等趋势市场\n")
		fmt.Printf("   均值回归策略需要谨慎使用\n")
		fmt.Printf("   建议: 优化参数并严格控制风险\n")
	} else {
		fmt.Printf("✅ 市场判断: 震荡市场\n")
		fmt.Printf("   均值回归策略适用\n")
		fmt.Printf("   建议: 继续优化策略参数\n")
	}

	avgScore := 0.0
	for _, s := range symbolScores {
		avgScore += s.score
	}
	avgScore /= float64(len(symbolScores))

	fmt.Printf("   平均适应性评分: %.1f/100\n", avgScore)
	if avgScore >= 60 {
		fmt.Printf("   整体评估: 市场环境相对适合均值回归\n")
	} else {
		fmt.Printf("   整体评估: 市场环境不适合均值回归\n")
	}
}

// 计算相关系数
func calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}

	n := float64(len(x))
	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2-sumX*sumX)*(n*sumY2-sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}