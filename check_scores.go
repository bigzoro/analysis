package main

import (
	"fmt"
	"log"

	"analysis/analysis_backend/internal/db"
	"analysis/analysis_backend/internal/server"
)

func main() {
	fmt.Println("🔍 检查策略评分")

	// 初始化数据库
	database, err := db.NewDatabase()
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		return
	}
	defer database.Close()

	// 创建服务器实例
	srv := &server.Server{
		DB: database,
	}

	// 获取市场环境分析
	analysis, err := srv.AnalyzeMarketEnvironment()
	if err != nil {
		log.Printf("市场环境分析失败: %v", err)
		return
	}

	fmt.Printf("📊 当前市场环境:\n")
	fmt.Printf("   震荡程度: %.2f%%\n", analysis.Oscillation)
	fmt.Printf("   整体趋势: %s\n", analysis.Trend)
	fmt.Printf("   波动率: %.2f%%\n", analysis.Volatility)

	// 模拟评分计算
	fmt.Printf("\n🎪 策略评分计算:\n")

	// 网格策略评分
	gridScore := 6.0
	gridConfidence := 60.0
	fmt.Printf("网格策略基础评分: %.1f\n", gridScore)

	if analysis.Trend == "震荡" {
		gridScore += 3
		gridConfidence = 85.0
		fmt.Printf("  + 横盘震荡市场加成: +3.0 → %.1f\n", gridScore)
	} else if analysis.Trend == "混合" {
		gridScore += 1
		gridConfidence = 70.0
		fmt.Printf("  + 混合市场加成: +1.0 → %.1f\n", gridScore)
	} else {
		gridScore -= 2
		gridConfidence = 40.0
		fmt.Printf("  - 趋势市场减分: -2.0 → %.1f\n", gridScore)
	}

	if analysis.Volatility < 30 {
		gridScore += 1
		fmt.Printf("  + 低波动率加成: +1.0 → %.1f\n", gridScore)
	}

	fmt.Printf("网格策略最终评分: %.1f, 置信度: %.1f%%\n", gridScore, gridConfidence)

	// 均值回归策略评分
	mrScore := 5.0
	mrConfidence := 50.0
	fmt.Printf("\n均值回归策略基础评分: %.1f\n", mrScore)

	if analysis.Oscillation > 60 {
		mrScore = 9
		mrConfidence = 85.0
		fmt.Printf("  震荡>60%%: 评分=9.0\n")
	} else if analysis.Oscillation > 40 {
		mrScore = 7
		mrConfidence = 65.0
		fmt.Printf("  震荡>40%%: 评分=7.0\n")
	} else {
		fmt.Printf("  震荡≤40%%: 评分=5.0\n")
	}

	fmt.Printf("均值回归策略最终评分: %.1f, 置信度: %.1f%%\n", mrScore, mrConfidence)

	fmt.Printf("\n🏆 预测结果:\n")
	if gridScore > mrScore {
		fmt.Printf("✅ 网格策略应该排第一 (%.1f > %.1f)\n", gridScore, mrScore)
	} else {
		fmt.Printf("❌ 均值回归策略排第一 (%.1f > %.1f)\n", mrScore, gridScore)
	}
}