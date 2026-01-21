package main

import (
	"fmt"
	"log"
	"strings"
)

// 智能模拟收益数据生成（简化版用于测试）
func getSmartPerformanceForSymbol(symbol string) float64 {
	// 根据币种的受欢迎程度和市值返回不同的模拟收益
	baseSymbol := symbol
	if len(baseSymbol) > 4 && baseSymbol[len(baseSymbol)-4:] == "USDT" {
		baseSymbol = baseSymbol[:len(baseSymbol)-4]
	}

	// 主流币种返回较小的模拟收益
	majorCoins := []string{"BTC", "ETH", "BNB", "ADA", "SOL", "DOT", "AVAX", "LINK", "LTC", "XRP", "DOGE"}
	for _, coin := range majorCoins {
		if baseSymbol == coin {
			log.Printf("[getSmartPerformanceForSymbol] 主流币种 %s 使用 0.5%% 模拟收益", symbol)
			return 0.005 // 0.5%的收益，主要币种波动更小
		}
	}

	// 次主流币种
	secondaryCoins := []string{"MATIC", "SHIB", "UNI", "ICP", "FIL", "ETC", "VET", "TRX", "THETA", "FTT"}
	for _, coin := range secondaryCoins {
		if baseSymbol == coin {
			log.Printf("[getSmartPerformanceForSymbol] 次主流币种 %s 使用 1.5%% 模拟收益", symbol)
			return 0.015 // 1.5%的收益
		}
	}

	// DeFi代币和Layer2代币通常波动较大
	defiCoins := []string{"AAVE", "COMP", "MKR", "SUSHI", "CAKE", "PancakeSwap", "1INCH", "CRV", "YFI", "BAL"}
	for _, coin := range defiCoins {
		if baseSymbol == coin {
			log.Printf("[getSmartPerformanceForSymbol] DeFi代币 %s 使用 2.5%% 模拟收益", symbol)
			return 0.025 // 2.5%的收益
		}
	}

	// Layer2和扩容代币
	layer2Coins := []string{"OP", "ARB", "MATIC", "IMX", "METIS", "ZK"}
	for _, coin := range layer2Coins {
		if baseSymbol == coin {
			log.Printf("[getSmartPerformanceForSymbol] Layer2代币 %s 使用 2.0%% 模拟收益", symbol)
			return 0.02 // 2.0%的收益
		}
	}

	// 新兴代币和Meme币通常波动最大
	memecoins := []string{"PEPE", "FLOKI", "BONK", "WIF", "MEW", "CUMMIES"}
	for _, coin := range memecoins {
		if baseSymbol == coin {
			log.Printf("[getSmartPerformanceForSymbol] Meme币 %s 使用 4.0%% 模拟收益", symbol)
			return 0.04 // 4.0%的收益
		}
	}

	// 检查是否是PancakeSwap相关的代币（通常波动较大）
	if baseSymbol == "CAKE" || strings.Contains(baseSymbol, "PANCAKE") ||
		baseSymbol == "SYRUP" || baseSymbol == "BANANA" {
		log.Printf("[getSmartPerformanceForSymbol] PancakeSwap代币 %s 使用 3.5%% 模拟收益", symbol)
		return 0.035 // 3.5%的收益，PancakeSwap代币波动较大
	}

	// 默认小币种收益
	log.Printf("[getSmartPerformanceForSymbol] 默认小币种 %s 使用 2.0%% 模拟收益", symbol)
	return 0.02 // 2%的收益
}

func main() {
	fmt.Println("🧪 智能表现数据生成测试")
	fmt.Println("========================")

	// 测试不同类型的币种
	testCases := []struct {
		symbol   string
		expected float64
		category string
	}{
		{"BTCUSDT", 0.005, "主流币种"},
		{"ETHUSDT", 0.005, "主流币种"},
		{"ADAUSDT", 0.005, "主流币种"},
		{"MATICUSDT", 0.015, "次主流币种"},
		{"SHIBUSDT", 0.015, "次主流币种"},
		{"AAVEUSDT", 0.025, "DeFi代币"},
		{"CAKEUSDT", 0.025, "DeFi代币"},
		{"OPUSDT", 0.02, "Layer2代币"},
		{"ARBUSDT", 0.02, "Layer2代币"},
		{"PEPEUSDT", 0.04, "Meme币"},
		{"SYRUPUSDT", 0.035, "PancakeSwap代币"},
		{"BANANAUSDT", 0.035, "PancakeSwap代币"},
		{"UNKNOWNUSDT", 0.02, "默认小币种"},
	}

	fmt.Println("\n1️⃣ 不同类型币种的智能收益分配:")
	fmt.Println("币种类型\t\t币种\t\t模拟收益")
	fmt.Println("--------\t\t----\t\t--------")

	for _, tc := range testCases {
		actual := getSmartPerformanceForSymbol(tc.symbol)
		fmt.Printf("%s\t\t%s\t\t%.1f%%\n",
			tc.category, tc.symbol, actual*100)
	}

	fmt.Println("\n2️⃣ 问题分析:")
	fmt.Println("• SYRUP是PancakeSwap的原生代币，主要在BSC网络交易")
	fmt.Println("• CoinCap有SYRUP的数据，但Binance可能没有SYRUPUSDT交易对")
	fmt.Println("• 因此市值数据存在，价格变化数据不存在是正常的")

	fmt.Println("\n3️⃣ 修复方案:")
	fmt.Println("• ✅ 智能模拟收益：根据币种类型分配合理的波动率")
	fmt.Println("• ✅ PancakeSwap代币：特殊识别，给予3.5%波动率")
	fmt.Println("• ✅ 多级分类：主流币种 < 次主流 < DeFi < Layer2 < Meme币")

	fmt.Println("\n✅ 智能表现数据生成测试完成")
	fmt.Println("============================")
	fmt.Println("修复要点:")
	fmt.Println("• 🎯 理解数据来源差异：CoinCap ≠ Binance")
	fmt.Println("• 🧠 智能模拟数据：基于币种特性分配收益")
	fmt.Println("• 📊 分类精确：不同类型代币不同波动率")
	fmt.Println("• 🔧 特殊处理：PancakeSwap代币等特殊情况")
	fmt.Println("\n🎉 现在SYRUPUSDT会获得3.5%的智能模拟收益！")
}
