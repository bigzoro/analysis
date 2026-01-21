package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	"analysis/analysis_backend/internal/db"
	"analysis/analysis_backend/internal/server"
)

func main() {
	fmt.Println("🔍 验证策略推荐排序修复")
	fmt.Println("========================")

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

	// 获取策略推荐
	recommendations := srv.GenerateStrategyRecommendations(analysis)

	fmt.Printf("\n🎪 策略推荐排序结果:\n")
	fmt.Println("───────────────────────────────")

	for i, rec := range recommendations {
		if i >= 5 { // 只显示前5个
			break
		}

		fmt.Printf("%d. %s (评分: %.1f, 置信度: %.1f%%)\n",
			i+1, rec.Name, rec.Score, rec.Confidence)
		fmt.Printf("   适用市场: %s\n", rec.SuitableMarket)
		fmt.Printf("   风险等级: %s\n", rec.RiskLevel)
		fmt.Printf("   推荐原因: %s\n\n", rec.Reason)
	}

	// 验证排序是否正确
	if len(recommendations) > 1 {
		isSorted := true
		for i := 0; i < len(recommendations)-1; i++ {
			if recommendations[i].Score < recommendations[i+1].Score {
				isSorted = false
				break
			}
		}

		if isSorted {
			fmt.Println("✅ 策略排序正确：评分按降序排列")
		} else {
			fmt.Println("❌ 策略排序错误：评分未按降序排列")
		}
	}

	// 检查网格策略是否排在第一位
	if len(recommendations) > 0 && recommendations[0].Type == "grid_trading" {
		fmt.Println("✅ 网格策略正确排在第一位")
	} else if len(recommendations) > 0 {
		fmt.Printf("❌ 网格策略未排在第一位，第一位是: %s (评分: %.1f)\n",
			recommendations[0].Name, recommendations[0].Score)

		// 显示所有策略的评分
		fmt.Println("\n📊 所有策略评分详情:")
		for _, rec := range recommendations {
			fmt.Printf("   %s: %.1f 分\n", rec.Name, rec.Score)
		}
	}
}