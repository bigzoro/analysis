package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	bf "analysis/internal/exchange/binancefutures"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Exchange struct {
		Environment string `yaml:"environment"`
		Binance     struct {
			Testnet struct {
				APIKey    string `yaml:"api_key"`
				SecretKey string `yaml:"secret_key"`
				Enabled   bool   `yaml:"enabled"`
			} `yaml:"testnet"`
			Mainnet struct {
				APIKey    string `yaml:"api_key"`
				SecretKey string `yaml:"secret_key"`
				Enabled   bool   `yaml:"enabled"`
			} `yaml:"mainnet"`
		} `yaml:"binance"`
	} `yaml:"exchange"`
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	fmt.Println("🔍 读取配置文件并获取Binance账户信息")
	fmt.Println("=====================================")

	// 尝试多个可能的配置文件位置
	configPaths := []string{
		"../config.yaml", // 根目录的配置文件
		"config.yaml",
		"./config.yaml",
		"cmd/data_sync/config.yaml",
		"../cmd/data_sync/config.yaml",
	}

	var config *Config
	var configPath string
	var err error

	for _, path := range configPaths {
		config, err = loadConfig(path)
		if err == nil {
			configPath = path
			break
		}
	}

	if err != nil {
		fmt.Printf("❌ 找不到配置文件，尝试的路径: %v\n", configPaths)
		fmt.Println("请确保config.yaml文件存在并包含正确的配置")
		return
	}

	fmt.Printf("✅ 成功加载配置文件: %s\n", configPath)

	// 根据environment选择配置
	var apiKey, secretKey string
	var isTestnet bool

	environment := config.Exchange.Environment
	if environment == "testnet" {
		apiKey = config.Exchange.Binance.Testnet.APIKey
		secretKey = config.Exchange.Binance.Testnet.SecretKey
		isTestnet = true
	} else if environment == "mainnet" {
		apiKey = config.Exchange.Binance.Mainnet.APIKey
		secretKey = config.Exchange.Binance.Mainnet.SecretKey
		isTestnet = false
	} else {
		fmt.Printf("❌ 无效的环境配置: %s\n", environment)
		fmt.Println("environment 必须是 'testnet' 或 'mainnet'")
		return
	}

	fmt.Printf("🔧 配置信息:\n")
	fmt.Printf("  环境: %s\n", environment)
	fmt.Printf("  测试网: %v\n", isTestnet)
	fmt.Printf("  API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-4:])
	fmt.Printf("  Secret Key: %s...%s\n", secretKey[:8], secretKey[len(secretKey)-4:])

	// 检查API密钥是否为空
	if apiKey == "" {
		fmt.Printf("❌ %s API Key未配置\n", environment)
		fmt.Printf("请在config.yaml的exchange.binance.%s.api_key中设置正确的API Key\n", environment)
		return
	}

	if secretKey == "" {
		fmt.Printf("❌ %s Secret Key未配置\n", environment)
		fmt.Printf("请在config.yaml的exchange.binance.%s.secret_key中设置正确的Secret Key\n", environment)
		return
	}

	fmt.Println("\n📋 测试API连接...")

	// 创建币安期货客户端
	client := bf.New(isTestnet, apiKey, secretKey)

	// 测试基本的exchange info获取
	info, err := client.GetExchangeInfo()
	if err != nil {
		log.Printf("❌ 获取交易所信息失败: %v", err)
		fmt.Println("\n🔍 故障排除:")
		fmt.Println("1. 检查网络连接")
		fmt.Println("2. 确认测试网/主网设置正确")
		fmt.Println("3. 验证API密钥是否有效")
		return
	}

	fmt.Printf("✅ 成功连接到交易所，共有%d个交易对\n", len(info.Symbols))

	fmt.Println("\n🔑 获取账户信息...")

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

		if isTestnet {
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

	// 首先检查RIVERUSDT
	riverFound := false
	for _, position := range accountInfo.Positions {
		if position.Symbol == "RIVERUSDT" {
			riverFound = true
			fmt.Printf("  %s (重点关注):\n", position.Symbol)
			fmt.Printf("    持仓数量: %s\n", position.PositionAmt)
			fmt.Printf("    持仓方向: %s\n", position.PositionSide)
			fmt.Printf("    入场价格: %s\n", position.EntryPrice)
			fmt.Printf("    未实现盈亏: %s\n", position.UnrealizedProfit)
			fmt.Printf("    杠杆倍数: %s\n", position.Leverage)

			marginMode := "全仓"
			if position.Isolated {
				marginMode = "逐仓"
			}
			fmt.Printf("    保证金模式: %s\n", marginMode)

			if position.Isolated {
				fmt.Printf("    逐仓钱包: %s USDT\n", position.IsolatedWallet)
			}
			fmt.Println()
			break
		}
	}

	if !riverFound {
		fmt.Println("  RIVERUSDT: 未找到持仓记录")
		fmt.Println()
	}

	activePositions := 0
	for _, position := range accountInfo.Positions {
		if position.PositionAmt != "0" && position.PositionAmt != "0.0" && position.PositionAmt != "" {
			activePositions++
		}
	}

	if activePositions == 0 {
		fmt.Println("  无活跃持仓")
	}

	fmt.Printf("\n🎯 获取完成!\n")

	// 分析账户盈亏情况
	fmt.Printf("\n📊 账户盈亏分析:\n")

	totalMargin, _ := strconv.ParseFloat(accountInfo.TotalMarginBalance, 64)

	// 计算USDT和USDC的总价值（假设USDC价值稳定≈1 USDT）
	usdtBalance := 0.0
	usdcBalance := 0.0
	btcBalance := 0.0

	for _, asset := range accountInfo.Assets {
		if asset.Asset == "USDT" {
			usdtBalance, _ = strconv.ParseFloat(asset.WalletBalance, 64)
		} else if asset.Asset == "USDC" {
			usdcBalance, _ = strconv.ParseFloat(asset.WalletBalance, 64)
		} else if asset.Asset == "BTC" {
			btcBalance, _ = strconv.ParseFloat(asset.WalletBalance, 64)
		}
	}

	// 估算BTC价值（使用当前价格）
	btcValue := btcBalance * 95000 // 假设BTC价格约95,000 USDT

	totalValue := usdtBalance + usdcBalance + btcValue

	// 分析可能的初始资金
	// 通常测试网账户会有初始资金，比如10000 USDT
	estimatedInitial := 10000.0 // 假设初始资金为10000 USDT
	totalPnL := totalValue - estimatedInitial
	totalPnLPercent := (totalPnL / estimatedInitial) * 100

	fmt.Printf("💰 当前总资产价值: %.2f USDT\n", totalValue)
	fmt.Printf("   ├── USDT余额: %.2f USDT\n", usdtBalance)
	fmt.Printf("   ├── USDC余额: %.2f USDT (按1:1估值)\n", usdcBalance)
	fmt.Printf("   └── BTC余额: %.6f BTC (≈%.2f USDT @95,000)\n", btcBalance, btcValue)

	if totalPnL >= 0 {
		fmt.Printf("📈 总盈亏: +%.2f USDT (+%.2f%%)\n", totalPnL, totalPnLPercent)
		fmt.Printf("🎉 账户整体盈利!\n")
	} else {
		fmt.Printf("📉 总盈亏: %.2f USDT (%.2f%%)\n", totalPnL, totalPnLPercent)
		fmt.Printf("⚠️  账户整体亏损\n")
	}

	// 分析保证金使用情况
	if accountInfo.AvailableBalance == "0.00000000" {
		fmt.Println("\n⚠️  可用保证金为0，请检查:")
		fmt.Println("   - 账户是否已在期货账户中存入资金")
		fmt.Println("   - 资金是否从现货账户划转到期货账户")
		fmt.Println("   - API权限是否包含读取余额权限")
	} else {
		fmt.Printf("\n✅ 账户正常，可用保证金: %s USDT\n", accountInfo.AvailableBalance)

		availableMargin, _ := strconv.ParseFloat(accountInfo.AvailableBalance, 64)
		marginUtilization := ((totalMargin - availableMargin) / totalMargin) * 100
		fmt.Printf("📊 保证金使用率: %.1f%%\n", marginUtilization)
	}

	// 总结
	fmt.Printf("\n🏆 总结:\n")
	if totalPnL >= 0 {
		fmt.Printf("   ✅ 账户盈利 %.2f USDT\n", totalPnL)
	} else {
		fmt.Printf("   ⚠️  账户亏损 %.2f USDT\n", -totalPnL)
	}

	var totalUnrealizedPnL float64
	for _, position := range accountInfo.Positions {
		if position.PositionAmt != "0" && position.PositionAmt != "0.0" && position.PositionAmt != "" {
			activePositions++
			if pnl, err := strconv.ParseFloat(position.UnrealizedProfit, 64); err == nil {
				totalUnrealizedPnL += pnl
			}

			// 特别显示RIVERUSDT的详细信息
			if position.Symbol == "RIVERUSDT" {
				marginMode := "全仓"
				if position.Isolated {
					marginMode = "逐仓"
				}
				fmt.Printf("    保证金模式: %s\n", marginMode)
				if position.Isolated {
					fmt.Printf("    逐仓钱包: %s USDT\n", position.IsolatedWallet)
				}
			}
		}
	}

	fmt.Printf("   📊 活跃持仓: %d 个\n", activePositions)
	if activePositions > 0 {
		fmt.Printf("   💰 未实现盈亏: %.2f USDT\n", totalUnrealizedPnL)
	} else {
		fmt.Printf("   💰 未实现盈亏: 0.00 USDT (无活跃持仓)\n")
	}
}
