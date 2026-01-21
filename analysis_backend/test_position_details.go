package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔍 FHEUSDT仓位详情查询工具")
	fmt.Println("=====================================")

	// 自动读取配置文件
	configPath := "./config.yaml"
	fmt.Printf("📄 正在读取配置文件: %s\n", configPath)

	var cfg config.Config
	config.MustLoad(configPath, &cfg)

	// 使用配置中的API密钥
	apiKey := cfg.Exchange.Binance.APIKey
	secretKey := cfg.Exchange.Binance.SecretKey
	useTestnet := cfg.Exchange.Binance.IsTestnet

	if apiKey == "" || secretKey == "" {
		fmt.Println("❌ 配置文件中未找到API密钥")
		return
	}

	fmt.Printf("\n🔧 配置: %s网络\n", map[bool]string{true: "测试网", false: "主网"}[useTestnet])
	fmt.Printf("🔑 API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-4:])

	// 创建币安期货客户端
	client := bf.New(useTestnet, apiKey, secretKey)

	fmt.Println("\n📋 获取详细持仓信息...")

	// 获取所有持仓的详细信息
	positions, err := client.GetPositions()
	if err != nil {
		log.Printf("❌ 获取持仓详情失败: %v", err)
		return
	}

	fmt.Printf("✅ 成功获取%d个持仓详情\n", len(positions))

	// 查找FHEUSDT的详细信息
	fheFound := false
	for _, position := range positions {
		if position.Symbol == "FHEUSDT" && position.PositionAmt != "0" && position.PositionAmt != "0.0" {
			fheFound = true
			fmt.Printf("\n📊 FHEUSDT详细仓位信息:\n")
			fmt.Printf("  交易对: %s\n", position.Symbol)
			fmt.Printf("  持仓数量: %s\n", position.PositionAmt)
			fmt.Printf("  持仓方向: %s\n", position.PositionSide)
			fmt.Printf("  入场价格: %s\n", position.EntryPrice)
			fmt.Printf("  标记价格: %s\n", position.MarkPrice)
			fmt.Printf("  未实现盈亏: %s\n", position.UnRealizedProfit)
			fmt.Printf("  杠杆倍数: %s\n", position.Leverage)
			fmt.Printf("  强平价格: %s\n", position.LiquidationPrice)

			// 仓位模式判断
			marginType := "全仓模式"
			if position.MarginType == "isolated" {
				marginType = "逐仓模式"
			}
			fmt.Printf("  仓位模式: %s\n", marginType)

			if position.MarginType == "isolated" {
				fmt.Printf("  逐仓保证金: %s USDT\n", position.IsolatedMargin)
				fmt.Printf("  逐仓钱包: %s USDT\n", position.IsolatedWallet)
			}

			fmt.Printf("  名义价值: %s\n", position.Notional)
			fmt.Printf("  自动追加保证金: %s\n", position.IsAutoAddMargin)
			break
		}
	}

	if !fheFound {
		fmt.Println("❌ 未找到FHEUSDT的活跃持仓")
	}

	fmt.Printf("\n🎯 查询完成!\n")
}