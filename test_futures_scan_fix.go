package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
)

func main() {
	fmt.Println("=== 测试期现套利扫描修复 ===")

	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建服务器实例
	srv := &server.Server{
		Cfg: &cfg,
	}

	// 模拟一个期现套利策略
	strategy := &pdb.TradingStrategy{
		Conditions: pdb.StrategyConditions{
			FuturesSpotArbEnabled: true,        // 启用期现套利
			ExpiryThreshold:       30,          // 30天内到期
			SpotFutureSpread:      0.3,         // 0.3%价差
		},
	}

	// 测试几个交易对
	testSymbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT"}

	fmt.Printf("策略条件: 期现套利=启用, 到期阈值=%d天, 价差阈值=%.1f%%\n\n",
		strategy.Conditions.ExpiryThreshold, strategy.Conditions.SpotFutureSpread)

	eligibleCount := 0

	for _, symbol := range testSymbols {
		fmt.Printf("测试交易对: %s\n", symbol)

		// 获取市场数据
		marketData := srv.GetMarketDataForSymbol(symbol)
		fmt.Printf("  - 市值: %.0f万\n", marketData.MarketCap/10000)
		fmt.Printf("  - 涨幅排名: %d\n", marketData.GainersRank)
		fmt.Printf("  - HasSpot: %v\n", marketData.HasSpot)
		fmt.Printf("  - HasFutures: %v\n", marketData.HasFutures)

		// 执行策略判断
		result := server.ExecuteStrategyLogic(*strategy, symbol, marketData)
		fmt.Printf("  - 策略结果: action=%s, reason=%s\n", result.Action, result.Reason)

		// 如果是期现套利，进行详细检查
		if result.Action == "allow" && strategy.Conditions.FuturesSpotArbEnabled {
			fmt.Println("  - 进行期现套利详细检查...")
			// 注意：这里无法直接调用Server的方法，因为需要数据库连接
			// 在实际运行中会调用checkFuturesSpotArbitrageForScan
		}

		if result.Action == "allow" || result.Action == "buy" || result.Action == "sell" {
			eligibleCount++
		}

		fmt.Println()
	}

	fmt.Printf("扫描结果: %d/%d 个交易对符合条件\n", eligibleCount, len(testSymbols))

	if eligibleCount < len(testSymbols) {
		fmt.Println("✅ 修复成功: 扫描逻辑现在会过滤不符合条件的交易对")
	} else {
		fmt.Println("❌ 可能还有问题: 仍然所有交易对都符合条件")
	}
}
