package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MarketAnalysisResponse struct {
	Success bool `json:"success"`
	Data    struct {
		MarketAnalysis struct {
			Volatility  float64 `json:"volatility"`
			Trend       string  `json:"trend"`
			Oscillation float64 `json:"oscillation"`
		} `json:"market_analysis"`
		StrategyRecommendations []struct {
			Name  string  `json:"name"`
			Score int     `json:"score"`
			Type  string  `json:"type"`
		} `json:"strategy_recommendations"`
	} `json:"data"`
}

func main() {
	fmt.Println("🔗 测试市场分析API修复效果")
	fmt.Println("============================")

	// 等待服务启动
	fmt.Println("⏳ 等待API服务启动...")
	time.Sleep(3 * time.Second)

	// 测试市场分析API
	resp, err := http.Get("http://localhost:8010/api/market-analysis/comprehensive")
	if err != nil {
		fmt.Printf("❌ API请求失败: %v\n", err)
		fmt.Println("💡 请确保后端服务已启动")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	if resp.StatusCode != 200 {
		fmt.Printf("❌ API返回错误状态码: %d\n", resp.StatusCode)
		fmt.Printf("响应内容: %s\n", string(body))
		return
	}

	var apiResp MarketAnalysisResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Printf("❌ 解析JSON失败: %v\n", err)
		fmt.Printf("原始响应: %s\n", string(body))
		return
	}

	if !apiResp.Success {
		fmt.Println("❌ API返回失败状态")
		return
	}

	fmt.Println("✅ API请求成功！")
	fmt.Println()

	// 显示市场分析结果
	ma := apiResp.Data.MarketAnalysis
	fmt.Printf("📊 市场分析结果:\n")
	fmt.Printf("   波动率: %.2f%%\n", ma.Volatility)
	fmt.Printf("   趋势: %s\n", ma.Trend)
	fmt.Printf("   震荡度: %.2f%%\n", ma.Oscillation)
	fmt.Println()

	// 显示策略推荐排名
	fmt.Println("🎯 策略推荐排名:")
	fmt.Println("================")

	strategies := apiResp.Data.StrategyRecommendations
	for i, strategy := range strategies {
		if i >= 3 { // 只显示前3名
			break
		}
		fmt.Printf("%d. %s (得分: %d)\n", i+1, strategy.Name, strategy.Score)
	}

	// 验证修复效果
	if len(strategies) > 0 {
		topStrategy := strategies[0]

		fmt.Println()
		fmt.Printf("🏆 当前第一名策略: %s\n", topStrategy.Name)

		if topStrategy.Type == "grid_trading" {
			fmt.Println("✅ 修复成功！网格策略现在正确排名第一")
			fmt.Println("🎉 问题已解决：震荡度计算错误导致的策略排名异常已修复")
		} else if topStrategy.Type == "mean_reversion" {
			fmt.Println("❌ 修复可能不完全：均值回归策略仍排第一")
			fmt.Printf("💡 震荡度: %.2f%% (应 < 40%% 才给均值回归高分)\n", ma.Oscillation)
			if ma.Oscillation > 40 {
				fmt.Println("📝 原因：震荡度仍较高，可能需要进一步调整阈值")
			}
		} else {
			fmt.Printf("🤔 其他策略排第一: %s\n", topStrategy.Name)
		}
	}

	// 显示修复前后对比
	fmt.Println()
	fmt.Println("🔄 修复前后对比:")
	fmt.Println("===============")
	fmt.Println("修复前:")
	fmt.Println("  • 趋势: 下跌")
	fmt.Println("  • 震荡度: 436.15% (异常高)")
	fmt.Println("  • 均值回归策略: 9分 (第一)")
	fmt.Println("  • 网格策略: 5分")
	fmt.Println()
	fmt.Println("修复后:")
	fmt.Printf("  • 趋势: %s\n", ma.Trend)
	fmt.Printf("  • 震荡度: %.2f%%\n", ma.Oscillation)
	if len(strategies) > 0 {
		fmt.Printf("  • 第一名: %s\n", strategies[0].Name)
		if len(strategies) > 1 {
			fmt.Printf("  • 第二名: %s\n", strategies[1].Name)
		}
	}
}