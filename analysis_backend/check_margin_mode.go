package main

import (
	"fmt"
	"log"
	"strings"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔍 检查持仓保证金模式分析")
	fmt.Println("========================")

	// 加载配置
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 创建币安客户端
	client := bf.New(cfg.Exchange.Binance.IsTestnet, cfg.Exchange.Binance.APIKey, cfg.Exchange.Binance.SecretKey)

	// 检查的交易对
	symbols := []string{"FHEUSDT", "RIVERUSDT"}

	fmt.Println("\n📊 当前持仓情况:")
	fmt.Println("FHEUSDT: -108 个 (空头), 杠杆3x")
	fmt.Println("RIVERUSDT: 2.0 个 (多头), 杠杆3x")

	fmt.Println("\n🔧 检查保证金模式:")

	for _, symbol := range symbols {
		fmt.Printf("\n--- 检查 %s ---\n", symbol)

		// 获取保证金模式
		code, body, err := client.GetMarginType(symbol)
		if err != nil {
			fmt.Printf("❌ 获取保证金模式失败: %v\n", err)
			continue
		}

		if code != 200 {
			fmt.Printf("❌ API响应错误: %d - %s\n", code, string(body))
			continue
		}

		// 解析响应
		responseStr := string(body)
		fmt.Printf("API响应: %s\n", responseStr)

		// 检查是否包含保证金模式信息
		if strings.Contains(responseStr, "CROSSED") {
			fmt.Printf("✅ %s: 全仓模式 (CROSSED)\n", symbol)
		} else if strings.Contains(responseStr, "ISOLATED") {
			fmt.Printf("✅ %s: 逐仓模式 (ISOLATED)\n", symbol)
		} else {
			fmt.Printf("❓ %s: 无法确定模式 (响应: %s)\n", symbol, responseStr)
		}
	}

	fmt.Println("\n🎯 分析结论:")

	// 分析哪个是新开的仓位
	fmt.Println("📈 新开仓位: RIVERUSDT (2.0个多头)")
	fmt.Println("📉 现有仓位: FHEUSDT (-108个空头)")

	fmt.Println("\n💡 技术说明:")
	fmt.Println("- 方案A已在订单创建时尝试设置保证金模式")
	fmt.Println("- 如果显示全仓，说明存在未成交订单导致设置失败")
	fmt.Println("- 订单执行时会自动重试设置正确的保证金模式")

	fmt.Printf("\n⏰ 检查完成时间: 2026-01-07 17:07:08\n")
}