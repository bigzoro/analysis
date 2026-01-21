package main

import (
	"encoding/json"
	"fmt"
	"log"

	"analysis/internal/config"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔍 FHEUSDT 保证金模式查询工具")
	fmt.Println("================================")

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

	fmt.Println("\n📋 查询 FHEUSDT 保证金模式...")

	// 获取 FHEUSDT 的保证金模式
	status, response, err := client.GetMarginType("FHEUSDT")
	if err != nil {
		log.Printf("❌ 获取保证金模式失败: %v", err)
		fmt.Println("\n🔍 故障排除:")
		fmt.Println("1. 检查网络连接")
		fmt.Println("2. 确认 FHEUSDT 交易对存在")
		fmt.Println("3. 检查API权限")
		return
	}

	fmt.Printf("✅ API响应状态码: %d\n", status)

	if status != 200 {
		fmt.Printf("❌ API响应失败，状态码: %d\n", status)
		fmt.Printf("响应内容: %s\n", string(response))
		return
	}

	// 解析响应
	var positions []struct {
		Symbol       string `json:"symbol"`
		MarginType   string `json:"marginType"`
		Isolated     bool   `json:"isolated"`
		PositionAmt  string `json:"positionAmt"`
		EntryPrice   string `json:"entryPrice"`
		Leverage     string `json:"leverage"`
	}

	err = json.Unmarshal(response, &positions)
	if err != nil {
		log.Printf("❌ 解析响应失败: %v", err)
		fmt.Printf("原始响应: %s\n", string(response))
		return
	}

	// 查找 FHEUSDT
	var fheusdtPosition *struct {
		Symbol       string `json:"symbol"`
		MarginType   string `json:"marginType"`
		Isolated     bool   `json:"isolated"`
		PositionAmt  string `json:"positionAmt"`
		EntryPrice   string `json:"entryPrice"`
		Leverage     string `json:"leverage"`
	}

	for i, pos := range positions {
		if pos.Symbol == "FHEUSDT" {
			fheusdtPosition = &positions[i]
			break
		}
	}

	if fheusdtPosition == nil {
		fmt.Println("❌ 未找到 FHEUSDT 的持仓信息")
		fmt.Println("可能的原因:")
		fmt.Println("1. 没有 FHEUSDT 的持仓")
		fmt.Println("2. 持仓数量为0")
		fmt.Println("3. API权限不足")

		// 显示所有持仓信息作为参考
		fmt.Println("\n📊 当前所有持仓:")
		for _, pos := range positions {
			if pos.PositionAmt != "0" && pos.PositionAmt != "0.0" {
				fmt.Printf("  %s: %s (杠杆:%s, 保证金模式:%s)\n",
					pos.Symbol, pos.PositionAmt, pos.Leverage, pos.MarginType)
			}
		}
		return
	}

	fmt.Println("\n🎯 FHEUSDT 保证金模式详情:")
	fmt.Printf("  交易对: %s\n", fheusdtPosition.Symbol)
	fmt.Printf("  持仓数量: %s\n", fheusdtPosition.PositionAmt)
	fmt.Printf("  入场价格: %s\n", fheusdtPosition.EntryPrice)
	fmt.Printf("  杠杆倍数: %s\n", fheusdtPosition.Leverage)
	fmt.Printf("  保证金模式: %s\n", fheusdtPosition.MarginType)
	fmt.Printf("  是否逐仓: %v\n", fheusdtPosition.Isolated)

	// 根据保证金模式给出结论
	switch fheusdtPosition.MarginType {
	case "isolated", "ISOLATED":
		fmt.Println("\n✅ 结论: 当前 FHEUSDT 持仓使用 逐仓模式")
		fmt.Println("💡 逐仓模式: 每个交易对独立保证金，风险可控")
	case "crossed", "CROSSED":
		fmt.Println("\n⚠️  结论: 当前 FHEUSDT 持仓使用 全仓模式")
		fmt.Println("💡 全仓模式: 共享账户保证金，风险较高")
	default:
		fmt.Printf("\n❓ 结论: 未知保证金模式: %s\n", fheusdtPosition.MarginType)
	}

	// 显示其他相关信息
	fmt.Printf("\n📈 持仓健康度分析:\n")
	if fheusdtPosition.PositionAmt != "0" && fheusdtPosition.PositionAmt != "0.0" {
		fmt.Println("  ✅ 有活跃持仓")
	} else {
		fmt.Println("  ⚠️  持仓数量为0")
	}

	fmt.Println("\n🎯 查询完成!")
}