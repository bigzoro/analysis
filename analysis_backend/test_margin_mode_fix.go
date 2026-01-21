package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔧 测试保证金模式设置修复")
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

	// 创建币安期货客户端
	client := bf.New(useTestnet, apiKey, secretKey)

	// 测试设置保证金模式
	testSymbol := "FHEUSDT"
	fmt.Printf("\n🔄 测试设置保证金模式: %s\n", testSymbol)

	// 测试设置为逐仓模式
	fmt.Println("1. 设置为逐仓模式...")
	if code, body, err := client.SetMarginType(testSymbol, "ISOLATED"); err != nil || code >= 400 {
		log.Printf("❌ 设置逐仓模式失败: code=%d body=%s err=%v", code, string(body), err)
	} else {
		fmt.Println("✅ 逐仓模式设置成功")
	}

	fmt.Printf("\n🎯 测试完成!\n")
}