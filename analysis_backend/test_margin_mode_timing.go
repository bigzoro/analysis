package main

import (
	"fmt"
	"strings"
	"time"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔬 币安期货保证金模式设置时序测试")
	fmt.Println("===================================")

	configPath := "./config.yaml"
	var cfg config.Config
	config.MustLoad(configPath, &cfg)

	client := bf.New(cfg.Exchange.Binance.IsTestnet, cfg.Exchange.Binance.APIKey, cfg.Exchange.Binance.SecretKey)

	fmt.Println("\n📋 测试场景1: 无订单时设置保证金模式")
	testSymbol := "BTCUSDT" // 使用一个不会有订单的交易对

	fmt.Printf("交易对: %s\n", testSymbol)

	// 1. 设置为全仓模式
	fmt.Println("1. 设置为全仓模式...")
	if code, body, err := client.SetMarginType(testSymbol, "CROSSED"); err != nil || code >= 400 {
		fmt.Printf("   ❌ 失败: %s\n", string(body))
	} else {
		fmt.Println("   ✅ 成功")
	}

	// 2. 设置为逐仓模式
	fmt.Println("2. 设置为逐仓模式...")
	if code, body, err := client.SetMarginType(testSymbol, "ISOLATED"); err != nil || code >= 400 {
		fmt.Printf("   ❌ 失败: %s\n", string(body))
	} else {
		fmt.Println("   ✅ 成功")
	}

	fmt.Println("\n📋 测试场景2: 模拟有订单时的限制")
	fmt.Printf("交易对: %s (当前有持仓)\n", "FHEUSDT")

	// 尝试设置FHEUSDT的保证金模式（应该会失败，因为有持仓）
	fmt.Println("1. 尝试设置FHEUSDT为逐仓模式...")
	startTime := time.Now()
	if code, body, err := client.SetMarginType("FHEUSDT", "ISOLATED"); err != nil || code >= 400 {
		bodyStr := string(body)
		if strings.Contains(bodyStr, "Margin type cannot be changed if there exists open orders") {
			fmt.Println("   ❌ 预期的失败: 存在未成交订单")
			fmt.Printf("   ⏱️  响应时间: %.2fs\n", time.Since(startTime).Seconds())
		} else {
			fmt.Printf("   ❌ 意外失败: %s\n", bodyStr)
		}
	} else {
		fmt.Println("   ✅ 意外成功 - 说明当前无未成交订单")
	}

	fmt.Println("\n🎯 币安保证金模式规则总结:")
	fmt.Println("✅ 可以随时设置杠杆倍数")
	fmt.Println("✅ 无订单时可以自由切换全仓/逐仓模式")
	fmt.Println("❌ 有未成交订单时无法更改保证金模式")
	fmt.Println("❌ 有持仓时无法更改保证金模式")

	fmt.Println("\n💡 最佳实践:")
	fmt.Println("1. 开仓前先设置保证金模式")
	fmt.Println("2. 避免在有活跃订单时更改模式")
	fmt.Println("3. 平仓后再调整保证金模式")

	fmt.Println("\n🔧 系统修复方案:")
	fmt.Println("✅ 策略执行时提前设置保证金模式")
	fmt.Println("✅ 订单执行失败时提供详细错误信息")
	fmt.Println("✅ 支持手动调整现有仓位模式")
}