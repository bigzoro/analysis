package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("🧪 测试订单详情页面的止盈止损百分比计算")
	fmt.Println("==========================================")

	// 基于订单1418的实际数据
	fmt.Println("\n📊 订单1418的实际数据:")
	fmt.Println("交易对: XNYUSDT")
	fmt.Println("方向: SELL (空头)")
	fmt.Println("杠杆: 3x")
	fmt.Println("成交数量: 69003")
	fmt.Println("成交均价: 0.004338 USDT")
	fmt.Println("名义价值: 69003 × 0.004338 = 299.34 USDT")
	fmt.Println("保证金: 299.34 ÷ 3 = 99.78 USDT")

	// 模拟订单数据
	order := map[string]interface{}{
		"id":              1418,
		"symbol":          "XNYUSDT",
		"side":            "SELL",
		"leverage":        3.0,
		"adjusted_quantity": "69003",
		"avg_price":       "0.004338",
		"tp_percent":      2.0,  // 用户设置的止盈百分比
		"sl_percent":      1.0,  // 用户设置的止损百分比
		"actual_tp_percent": 2.5, // 实际使用的止盈百分比
		"actual_sl_percent": 1.2, // 实际使用的止损百分比
		"tp_price":        "0.004230", // 止盈价格
		"sl_price":        "0.004392", // 止损价格
	}

	fmt.Println("\n🔍 数据库中的百分比数据:")
	fmt.Printf("用户设置止盈百分比 (tp_percent): %.2f%%\n", order["tp_percent"])
	fmt.Printf("用户设置止损百分比 (sl_percent): %.2f%%\n", order["sl_percent"])
	fmt.Printf("实际止盈百分比 (actual_tp_percent): %.2f%%\n", order["actual_tp_percent"])
	fmt.Printf("实际止损百分比 (actual_sl_percent): %.2f%%\n", order["actual_sl_percent"])

	fmt.Println("\n💰 价格数据:")
	fmt.Printf("成交均价: %s USDT\n", order["avg_price"])
	fmt.Printf("止盈价格: %s USDT\n", order["tp_price"])
	fmt.Printf("止损价格: %s USDT\n", order["sl_price"])

	fmt.Println("\n🔬 验证百分比计算是否正确:")

	// 解析数据
	entryPrice, _ := strconv.ParseFloat(order["avg_price"].(string), 64)
	tpPrice, _ := strconv.ParseFloat(order["tp_price"].(string), 64)
	slPrice, _ := strconv.ParseFloat(order["sl_price"].(string), 64)
	leverage := order["leverage"].(float64)
	isLong := strings.ToUpper(order["side"].(string)) == "BUY"

	fmt.Printf("\n📈 基于价格重新计算百分比:\n")
	fmt.Printf("入场价格: %.8f USDT\n", entryPrice)
	fmt.Printf("是否多头仓位: %v\n", isLong)
	fmt.Printf("杠杆倍数: %.1f\n", leverage)

	// 计算实际百分比（基于价格）
	var calculatedTPPercent, calculatedSLPercent float64

	if isLong {
		// 多头仓位
		if tpPrice > entryPrice {
			calculatedTPPercent = ((tpPrice - entryPrice) / entryPrice) * 100
		}
		if slPrice < entryPrice {
			calculatedSLPercent = ((entryPrice - slPrice) / entryPrice) * 100
		}
	} else {
		// 空头仓位 (SELL)
		if tpPrice < entryPrice {
			calculatedTPPercent = ((entryPrice - tpPrice) / entryPrice) * 100
		}
		if slPrice > entryPrice {
			calculatedSLPercent = ((slPrice - entryPrice) / entryPrice) * 100
		}
	}

	fmt.Printf("\n基于价格计算的实际百分比:\n")
	fmt.Printf("止盈百分比: %.4f%%\n", calculatedTPPercent)
	fmt.Printf("止损百分比: %.4f%%\n", calculatedSLPercent)

	fmt.Printf("\n对比数据库中的actual百分比:\n")
	fmt.Printf("数据库止盈百分比: %.4f%%\n", order["actual_tp_percent"])
	fmt.Printf("数据库止损百分比: %.4f%%\n", order["actual_sl_percent"])

	// 验证是否匹配
	tpMatch := abs(calculatedTPPercent-order["actual_tp_percent"].(float64)) < 0.01
	slMatch := abs(calculatedSLPercent-order["actual_sl_percent"].(float64)) < 0.01

	fmt.Printf("\n✅ 验证结果:\n")
	if tpMatch {
		fmt.Printf("✅ 止盈百分比计算正确\n")
	} else {
		fmt.Printf("❌ 止盈百分比计算错误\n")
		fmt.Printf("   期望: %.4f%%, 实际: %.4f%%\n", calculatedTPPercent, order["actual_tp_percent"])
	}

	if slMatch {
		fmt.Printf("✅ 止损百分比计算正确\n")
	} else {
		fmt.Printf("❌ 止损百分比计算错误\n")
		fmt.Printf("   期望: %.4f%%, 实际: %.4f%%\n", calculatedSLPercent, order["actual_sl_percent"])
	}

	fmt.Println("\n📋 分析保证金止盈止损计算:")

	// 模拟保证金止盈止损计算
	marginTPPercent := 2.5  // 保证金止盈2.5%
	marginSLPercent := 1.2  // 保证金止损1.2%

	quantity, _ := strconv.ParseFloat(order["adjusted_quantity"].(string), 64)
	notional := quantity * entryPrice  // 名义价值
	initialMargin := notional / leverage  // 初始保证金

	fmt.Printf("\n保证金计算参数:\n")
	fmt.Printf("名义价值: %.2f USDT\n", notional)
	fmt.Printf("初始保证金: %.2f USDT\n", initialMargin)
	fmt.Printf("保证金止盈百分比: %.2f%%\n", marginTPPercent)
	fmt.Printf("保证金止损百分比: %.2f%%\n", marginSLPercent)

	// 保证金止盈计算
	targetProfit := initialMargin * (marginTPPercent / 100)
	marginTPPriceChange := targetProfit / quantity

	var marginTPPrice float64
	if isLong {
		marginTPPrice = entryPrice + marginTPPriceChange
	} else {
		marginTPPrice = entryPrice - marginTPPriceChange
	}

	// 保证金止损计算
	targetLoss := initialMargin * (marginSLPercent / 100)
	marginSLPriceChange := targetLoss / quantity

	var marginSLPrice float64
	if isLong {
		marginSLPrice = entryPrice - marginSLPriceChange
	} else {
		marginSLPrice = entryPrice + marginSLPriceChange
	}

	fmt.Printf("\n保证金止盈计算:\n")
	fmt.Printf("目标盈利: %.4f USDT\n", targetProfit)
	fmt.Printf("价格变动: %.8f USDT\n", marginTPPriceChange)
	fmt.Printf("止盈价格: %.8f USDT\n", marginTPPrice)

	fmt.Printf("\n保证金止损计算:\n")
	fmt.Printf("目标亏损: %.4f USDT\n", targetLoss)
	fmt.Printf("价格变动: %.8f USDT\n", marginSLPriceChange)
	fmt.Printf("止损价格: %.8f USDT\n", marginSLPrice)

	// 验证价格是否匹配
	tpPriceMatch := abs(marginTPPrice-tpPrice) < 0.000001
	slPriceMatch := abs(marginSLPrice-slPrice) < 0.000001

	fmt.Printf("\n🔍 价格验证:\n")
	if tpPriceMatch {
		fmt.Printf("✅ 止盈价格计算正确\n")
	} else {
		fmt.Printf("❌ 止盈价格计算错误\n")
		fmt.Printf("   期望: %.8f, 实际: %.8f\n", marginTPPrice, tpPrice)
	}

	if slPriceMatch {
		fmt.Printf("✅ 止损价格计算正确\n")
	} else {
		fmt.Printf("❌ 止损价格计算错误\n")
		fmt.Printf("   期望: %.8f, 实际: %.8f\n", marginSLPrice, slPrice)
	}

	fmt.Println("\n🎯 结论:")

	if tpMatch && slMatch && tpPriceMatch && slPriceMatch {
		fmt.Println("✅ 订单详情页面的止盈止损百分比显示正确！")
		fmt.Println("   - 百分比计算基于实际价格变动")
		fmt.Println("   - 保证金止盈止损价格计算正确")
		fmt.Println("   - 前端显示逻辑正常")
	} else {
		fmt.Println("❌ 发现止盈止损百分比计算或显示问题")
		if !tpMatch {
			fmt.Println("   - 止盈百分比计算不正确")
		}
		if !slMatch {
			fmt.Println("   - 止损百分比计算不正确")
		}
		if !tpPriceMatch {
			fmt.Println("   - 止盈价格与保证金计算不匹配")
		}
		if !slPriceMatch {
			fmt.Println("   - 止损价格与保证金计算不匹配")
		}
	}

	fmt.Println("\n💡 止盈止损百分比计算说明:")
	fmt.Println("1. 用户设置百分比(tp_percent/sl_percent): 用户在创建订单时设置的预期百分比")
	fmt.Println("2. 实际百分比(actual_tp_percent/actual_sl_percent): 基于最终成交价格和止盈止损价格计算的实际百分比")
	fmt.Println("3. 保证金止盈止损: 基于保证金亏损/盈利百分比计算的价格，与传统价格百分比不同")
	fmt.Println("4. 百分比计算公式(空头): ((入场价格 - 止盈价格) / 入场价格) × 100%")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}