package main

import (
	"fmt"
	"log"

	"analysis/analysis_backend/internal/db"
	"analysis/analysis_backend/internal/server"
)

func main() {
	fmt.Println("🔍 验证策略评分修复")

	database, err := db.NewDatabase()
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		return
	}
	defer database.Close()

	srv := &server.Server{DB: database}

	analysis, err := srv.AnalyzeMarketEnvironment()
	if err != nil {
		log.Printf("市场环境分析失败: %v", err)
		return
	}

	fmt.Printf("📊 当前环境: 震荡=%.2f%%, 趋势=%s, 波动率=%.2f%%\n", analysis.Oscillation, analysis.Trend, analysis.Volatility)

	// 网格策略评分
	gridScore := 6.0
	if analysis.Trend == "震荡" {
		gridScore += 3
	} else if analysis.Trend == "混合" {
		gridScore += 1
	} else {
		gridScore -= 2
	}
	if analysis.Volatility < 30 {
		gridScore += 1
	}

	// 均值回归策略评分
	mrScore := 5.0
	if analysis.Oscillation > 60 {
		mrScore = 9
	} else if analysis.Oscillation > 40 {
		mrScore = 7
	}

	fmt.Printf("🎪 评分结果: 网格=%.1f, 均值回归=%.1f\n", gridScore, mrScore)

	if gridScore > mrScore {
		fmt.Println("✅ 网格策略得分更高，应该排第一")
	} else {
		fmt.Println("❌ 均值回归策略得分更高")
	}
}