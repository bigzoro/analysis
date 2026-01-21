package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔍 Binance期货账户信息测试工具")
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
		fmt.Println("请检查 config.yaml 中的 exchange.binance 配置")
		return
	}

	fmt.Printf("\n🔧 配置: %s网络\n", map[bool]string{true: "测试网", false: "主网"}[useTestnet])
	fmt.Printf("🔑 API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-4:])

	// 创建币安期货客户端
	client := bf.New(useTestnet, apiKey, secretKey)

	fmt.Println("\n📋 测试基本连接...")

	// 测试基本的exchange info获取
	info, err := client.GetExchangeInfo()
	if err != nil {
		log.Printf("❌ 获取交易所信息失败: %v", err)
		fmt.Println("\n🔍 故障排除:")
		fmt.Println("1. 检查网络连接")
		fmt.Println("2. 确认测试网/主网设置正确")
		return
	}

	fmt.Printf("✅ 成功连接到交易所，共有%d个交易对\n", len(info.Symbols))

	// 查找RIVERUSDT
	riverFound := false
	for _, symbol := range info.Symbols {
		if symbol.Symbol == "RIVERUSDT" {
			fmt.Printf("✅ 找到RIVERUSDT交易对: %s\n", symbol.Status)
			riverFound = true
			break
		}
	}

	if !riverFound {
		fmt.Println("❌ 未找到RIVERUSDT交易对")
	}

	fmt.Println("\n🔑 测试账户信息获取...")

	// 获取账户信息
	accountInfo, err := client.GetAccountInfo()
	if err != nil {
		log.Printf("❌ 获取账户信息失败: %v", err)

		fmt.Println("\n🔍 故障排除:")
		fmt.Println("1. 检查API密钥是否正确")
		fmt.Println("2. 确认API密钥有以下权限:")
		fmt.Println("   - 读取账户信息权限")
		fmt.Println("   - 期货交易权限")
		fmt.Println("3. 确认账户已开通期货交易")
		fmt.Println("4. 检查IP白名单设置")
		fmt.Println("5. 确认系统时间同步")

		if cfg.Exchange.Binance.IsTestnet {
			fmt.Println("6. 测试网API密钥获取: https://testnet.binance.vision")
		} else {
			fmt.Println("6. 主网API密钥获取: https://www.binance.com")
		}
		return
	}

	fmt.Println("✅ 成功获取账户信息!")

	// 显示账户概览
	fmt.Printf("\n💰 账户概览:\n")
	fmt.Printf("  可用保证金: %s USDT\n", accountInfo.AvailableBalance)
	fmt.Printf("  钱包余额: %s USDT\n", accountInfo.TotalWalletBalance)
	fmt.Printf("  保证金余额: %s USDT\n", accountInfo.TotalMarginBalance)
	fmt.Printf("  是否可交易: %v\n", accountInfo.CanTrade)
	fmt.Printf("  是否可入金: %v\n", accountInfo.CanDeposit)
	fmt.Printf("  是否可出金: %v\n", accountInfo.CanWithdraw)

	// 显示资产详情
	fmt.Printf("\n📊 资产详情:\n")
	for _, asset := range accountInfo.Assets {
		if asset.WalletBalance != "0.00000000" {
			fmt.Printf("  %s:\n", asset.Asset)
			fmt.Printf("    钱包余额: %s\n", asset.WalletBalance)
			fmt.Printf("    未实现盈亏: %s\n", asset.UnrealizedProfit)
			fmt.Printf("    保证金余额: %s\n", asset.MarginBalance)
			fmt.Printf("    可用余额: %s\n", asset.AvailableBalance)
			fmt.Printf("    初始保证金: %s\n", asset.InitialMargin)
			fmt.Printf("    维持保证金: %s\n", asset.MaintMargin)
		}
	}

	// 显示持仓信息
	fmt.Printf("\n📈 持仓信息:\n")
	activePositions := 0
	for _, position := range accountInfo.Positions {
		if position.PositionAmt != "0" && position.PositionAmt != "0.0" && position.PositionAmt != "" {
			activePositions++
			fmt.Printf("  %s:\n", position.Symbol)
			fmt.Printf("    持仓数量: %s\n", position.PositionAmt)
			fmt.Printf("    持仓方向: %s\n", position.PositionSide)
			fmt.Printf("    入场价格: %s\n", position.EntryPrice)
			fmt.Printf("    未实现盈亏: %s\n", position.UnrealizedProfit)
			fmt.Printf("    杠杆倍数: %s\n", position.Leverage)
		}
	}

	if activePositions == 0 {
		fmt.Println("  无活跃持仓")
	}

	fmt.Printf("\n🎯 测试完成!\n")
	if accountInfo.AvailableBalance == "0.00000000" {
		fmt.Println("⚠️  可用保证金为0，请检查:")
		fmt.Println("   - 账户是否已在期货账户中存入资金")
		fmt.Println("   - 资金是否从现货账户划转到期货账户")
		fmt.Println("   - API权限是否包含读取余额权限")
	} else {
		fmt.Printf("✅ 账户正常，可用保证金: %s USDT\n", accountInfo.AvailableBalance)
	}
}