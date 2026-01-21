package main

import (
	"fmt"
	"log"

	"analysis/analysis_backend/internal/db"
	"analysis/analysis_backend/internal/server"
)

func main() {
	fmt.Println("🔍 验证策略推荐排序修复")

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

	// 获取策略推荐
	recommendations := srv.GenerateStrategyRecommendations(analysis)

	fmt.Printf("\n🎪 策略推荐排序结果:\n")

	for i, rec := range recommendations {
		if i >= 3 { // 只显示前3个
			break
		}
		fmt.Printf("%d. %s (评分: %.1f)\n", i+1, rec.Name, rec.Score)
	}

	// 检查网格策略是否排在第一位
	if len(recommendations) > 0 && recommendations[0].Type == "grid_trading" {
		fmt.Println("✅ 网格策略正确排在第一位")
	} else if len(recommendations) > 0 {
		fmt.Printf("❌ 第一位是: %s (评分: %.1f)\n", recommendations[0].Name, recommendations[0].Score)
	}
}