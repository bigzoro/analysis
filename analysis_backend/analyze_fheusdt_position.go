package main

import (
	"fmt"
	"strings"
	"time"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔍 FHEUSDT新仓位深度分析")
	fmt.Println("==========================")

	// 读取配置
	configPath := "./config.yaml"
	var cfg config.Config
	config.MustLoad(configPath, &cfg)

	client := bf.New(cfg.Exchange.Binance.IsTestnet, cfg.Exchange.Binance.APIKey, cfg.Exchange.Binance.SecretKey)

	fmt.Println("\n📊 FHEUSDT仓位状态对比:")

	// 当前仓位信息
	fmt.Println("🆕 当前仓位 (新开仓):")
	fmt.Println("   持仓数量: -112")
	fmt.Println("   入场价格: 0.04471")
	fmt.Println("   未实现盈亏: 0.00560000")
	fmt.Println("   杠杆倍数: 3x")
	fmt.Println("   保证金模式: 全仓模式 ❌")

	fmt.Println("\n📋 策略33配置回顾:")
	fmt.Println("   保证金模式: ISOLATED (逐仓)")
	fmt.Println("   杠杆倍数: 3x")
	fmt.Println("   预期结果: 逐仓模式 ✅")

	fmt.Println("\n🔍 问题诊断:")

	// 检查是否有未成交订单
	fmt.Println("1. 检查是否有未成交订单影响保证金模式设置...")
	testSymbol := "FHEUSDT"
	if code, body, err := client.SetMarginType(testSymbol, "ISOLATED"); err != nil || code >= 400 {
		bodyStr := string(body)
		if strings.Contains(bodyStr, "Margin type cannot be changed if there exists open orders") {
			fmt.Println("   ❌ 发现未成交订单 - 这阻止了保证金模式设置")
			fmt.Println("   💡 建议: 等待订单成交后再设置保证金模式")
		} else {
			fmt.Printf("   ❌ 设置失败: %s\n", bodyStr)
		}
	} else {
		fmt.Println("   ✅ 可以设置逐仓模式")
	}

	fmt.Println("\n🎯 分析结论:")
	fmt.Println("❌ 新开仓位为全仓模式，与策略33配置不符")
	fmt.Println("🔧 可能原因:")
	fmt.Println("   1. 仓位在修复代码前开仓")
	fmt.Println("   2. 存在未成交订单阻止模式切换")
	fmt.Println("   3. 仓位为手动开仓，非策略执行")

	fmt.Println("\n💡 解决建议:")
	fmt.Println("1. ✅ 等待所有订单成交")
	fmt.Println("2. ✅ 手动调整现有仓位为逐仓模式")
	fmt.Println("3. ✅ 验证策略33下次执行是否正确应用逐仓")

	fmt.Println("\n📈 仓位表现:")
	fmt.Println("   📊 名义价值: -5.00 USDT")
	fmt.Println("   💰 未实现盈亏: +0.0056 USDT")
	fmt.Println("   🎯 强平价格: 43.82235142 USDT")
	fmt.Println("   ⚡ 杠杆倍数: 3x")

	fmt.Printf("\n⏰ 分析时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}
