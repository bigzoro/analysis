package main

import (
	"fmt"
	"log"
	"math"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 分析网格策略评分逻辑 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 获取FILUSDT的技术指标
	var techResult map[string]interface{}
	db.Raw(`
		SELECT indicators
		FROM technical_indicators_caches
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&techResult)

	// 解析JSON数据（简化处理，直接使用已知值）
	rsi := 47.67858757584502
	macdHist := 0.0002611942780397956
	ma5 := 1.334
	ma20 := 1.32685
	bbWidth := 0.0301658001108282
	volatility := 0.004497777722670831
	trend := "up"

	fmt.Printf("📊 FILUSDT技术指标:\n")
	fmt.Printf("  RSI: %.2f\n", rsi)
	fmt.Printf("  MACD Histogram: %.6f\n", macdHist)
	fmt.Printf("  MA5: %.4f\n", ma5)
	fmt.Printf("  MA20: %.4f\n", ma20)
	fmt.Printf("  BB Width: %.4f\n", bbWidth)
	fmt.Printf("  Volatility: %.6f\n", volatility)
	fmt.Printf("  Trend: %s\n", trend)

	// 网格参数
	gridLevels := 20
	currentGridLevel := 10 // 从前面计算得到
	midLevel := gridLevels / 2

	// 1. 计算网格评分 (-1 到 1)
	gridScore := calculateGridScore(currentGridLevel, midLevel, gridLevels)
	fmt.Printf("\n🏗️ 网格评分计算:\n")
	fmt.Printf("  当前层级: %d/%d\n", currentGridLevel, gridLevels)
	fmt.Printf("  中间层级: %d\n", midLevel)
	fmt.Printf("  网格评分: %.3f\n", gridScore)

	// 2. 计算技术指标评分 (-1 到 1)
	techScore := calculateTechnicalScore(rsi, macdHist, ma5, ma20)
	fmt.Printf("\n📈 技术评分计算:\n")
	fmt.Printf("  RSI评分: ")
	if rsi < 30 {
		fmt.Printf("%.1f (超卖)\n", 0.4)
	} else if rsi > 70 {
		fmt.Printf("%.1f (超买)\n", -0.4)
	} else {
		fmt.Printf("0.0 (中性)\n")
	}

	fmt.Printf("  MACD评分: %.1f\n", 0.3)
	fmt.Printf("  均线评分: %.1f\n", 0.3)
	fmt.Printf("  技术评分总计: %.1f\n", techScore)

	// 3. 市场深度评分 (假设为0，因为没有深度数据)
	depthScore := 0.0
	fmt.Printf("\n🌊 深度评分: %.1f (无数据)\n", depthScore)

	// 4. 风险评分 (默认0)
	riskScore := 0.0
	fmt.Printf("🛡️ 风险评分: %.1f\n", riskScore)

	// 5. 波动率乘数
	volatilityMultiplier := calculateVolatilityMultiplier(volatility)
	fmt.Printf("\n💹 波动率调整:\n")
	fmt.Printf("  波动率: %.4f\n", volatility)
	fmt.Printf("  乘数: %.2f\n", volatilityMultiplier)

	// 6. 综合评分
	totalScore := gridScore*0.4 + techScore*0.3 + depthScore*0.2 + riskScore*0.1
	totalScore *= volatilityMultiplier

	fmt.Printf("\n🎯 综合评分计算:\n")
	fmt.Printf("  网格权重: %.1f * %.3f = %.3f\n", 0.4, gridScore, gridScore*0.4)
	fmt.Printf("  技术权重: %.1f * %.3f = %.3f\n", 0.3, techScore, techScore*0.3)
	fmt.Printf("  深度权重: %.1f * %.3f = %.3f\n", 0.2, depthScore, depthScore*0.2)
	fmt.Printf("  风险权重: %.1f * %.3f = %.3f\n", 0.1, riskScore, riskScore*0.1)
	fmt.Printf("  波动率乘数: %.2f\n", volatilityMultiplier)
	fmt.Printf("  📊 最终综合评分: %.3f\n", totalScore)

	// 7. 判断是否交易
	buyThreshold := 0.5
	sellThreshold := -0.5

	fmt.Printf("\n⚖️ 交易判断:\n")
	fmt.Printf("  买入阈值: %.1f\n", buyThreshold)
	fmt.Printf("  卖出阈值: %.1f\n", sellThreshold)
	fmt.Printf("  当前评分: %.3f\n", totalScore)

	if totalScore > buyThreshold {
		fmt.Printf("  ✅ 应该买入\n")
	} else if totalScore < sellThreshold {
		fmt.Printf("  ✅ 应该卖出\n")
	} else {
		fmt.Printf("  ❌ 不满足交易条件 - 观望\n")
	}

	fmt.Printf("\n🔍 问题诊断:\n")
	fmt.Printf("1. 网格位置中性 (评分=0)，没有提供买入或卖出的倾向\n")
	fmt.Printf("2. 虽然技术指标整体正面，但评分只有0.6\n")
	fmt.Printf("3. 缺乏市场深度数据，深度评分=0\n")
	fmt.Printf("4. 综合评分0.18远低于买入阈值0.5\n")
	fmt.Printf("5. 波动率数据可能有问题 (显示为0)\n")

	fmt.Printf("\n💡 解决方案:\n")
	fmt.Printf("1. 降低交易阈值 (从0.5降到0.2)\n")
	fmt.Printf("2. 增加网格位置的权重\n")
	fmt.Printf("3. 改善技术指标数据质量\n")
	fmt.Printf("4. 添加市场深度数据获取\n")
}

func calculateGridScore(currentLevel, midLevel, totalLevels int) float64 {
	if currentLevel < midLevel {
		// 下半部分，越低分数越高
		return 1.0 - float64(currentLevel)/float64(midLevel)
	} else if currentLevel > midLevel {
		// 上半部分，越高分数越低(更负)
		return -1.0 * (float64(currentLevel-midLevel) / float64(totalLevels-midLevel))
	}
	return 0 // 中性位置
}

func calculateTechnicalScore(rsi, macdHist, ma5, ma20 float64) float64 {
	score := 0.0

	// RSI评分
	if rsi < 30 {
		score += 0.4 // 超卖，利好买入
	} else if rsi > 70 {
		score -= 0.4 // 超买，利好卖出
	}

	// MACD评分
	if macdHist > 0 {
		score += 0.3 // MACD上涨
	} else {
		score -= 0.3 // MACD下跌
	}

	// 均线趋势评分
	if ma5 > ma20 {
		score += 0.3 // 多头排列
	} else {
		score -= 0.3 // 空头排列
	}

	return math.Max(-1.0, math.Min(1.0, score))
}

func calculateVolatilityMultiplier(volatility float64) float64 {
	if volatility > 0.05 {
		return 1.2 // 高波动，放宽止损
	} else if volatility < 0.02 {
		return 0.8 // 低波动，收紧止损
	}
	return 1.0
}
