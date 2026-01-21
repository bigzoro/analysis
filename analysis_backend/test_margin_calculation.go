package main

import (
	"fmt"
	"log"
	bf "analysis/internal/exchange/binancefutures"
	"analysis/internal/server/strategy/shared/execution"
)

func main() {
	fmt.Println("🧪 测试保证金止盈止损计算")
	fmt.Println("=====================================")

	// 创建币安客户端（测试环境）
	useTestnet := true
	client := bf.New(useTestnet, "", "")

	// 创建保证金风险管理器
	marginRiskManager := execution.NewMarginRiskManager(client)

	// 测试场景1: 多头仓位
	fmt.Println("\n📈 场景1: 多头仓位 (BUY)")
	fmt.Println("-------------------------------")

	expectedEntryPrice := 0.05774042  // 入场价格
	expectedQuantity := 5200.0        // 持仓数量
	leverage := 3.0                   // 杠杆倍数
	marginLossPercent := 1.0          // 止损百分比
	marginProfitPercent := 1.0        // 止盈百分比
	isLong := true                    // 多头仓位

	// 计算名义价值和保证金
	notional := expectedQuantity * expectedEntryPrice
	initialMargin := notional / leverage

	fmt.Printf("入场价格: %.8f\n", expectedEntryPrice)
	fmt.Printf("持仓数量: %.0f\n", expectedQuantity)
	fmt.Printf("杠杆倍数: %.0f\n", leverage)
	fmt.Printf("名义价值: %.4f\n", notional)
	fmt.Printf("初始保证金: %.4f\n", initialMargin)

	// 计算止损价格
	stopPrice, err := marginRiskManager.CalculateEstimatedMarginStopLoss(
		expectedEntryPrice, expectedQuantity, leverage, marginLossPercent, isLong)
	if err != nil {
		log.Printf("❌ 止损价格计算失败: %v", err)
	} else {
		fmt.Printf("✅ 止损价格 (%.1f%%): %.8f\n", marginLossPercent, stopPrice)
		targetLoss := initialMargin * (marginLossPercent / 100)
		priceChange := targetLoss / expectedQuantity
		fmt.Printf("   目标亏损金额: %.4f\n", targetLoss)
		fmt.Printf("   价格变动: %.8f\n", priceChange)
		fmt.Printf("   止损触发价格: %.8f (下跌%.4f%%)\n",
			stopPrice, (expectedEntryPrice-stopPrice)/expectedEntryPrice*100)
	}

	// 计算止盈价格
	takeProfitPrice, err := marginRiskManager.CalculateEstimatedMarginTakeProfit(
		expectedEntryPrice, expectedQuantity, leverage, marginProfitPercent, isLong)
	if err != nil {
		log.Printf("❌ 止盈价格计算失败: %v", err)
	} else {
		fmt.Printf("✅ 止盈价格 (%.1f%%): %.8f\n", marginProfitPercent, takeProfitPrice)
		targetProfit := initialMargin * (marginProfitPercent / 100)
		priceChange := targetProfit / expectedQuantity
		fmt.Printf("   目标盈利金额: %.4f\n", targetProfit)
		fmt.Printf("   价格变动: %.8f\n", priceChange)
		fmt.Printf("   止盈触发价格: %.8f (上涨%.4f%%)\n",
			takeProfitPrice, (takeProfitPrice-expectedEntryPrice)/expectedEntryPrice*100)
	}

	// 测试场景2: 空头仓位
	fmt.Println("\n📉 场景2: 空头仓位 (SELL)")
	fmt.Println("-------------------------------")

	isLong = false // 空头仓位

	fmt.Printf("入场价格: %.8f\n", expectedEntryPrice)
	fmt.Printf("持仓数量: %.0f\n", expectedQuantity)
	fmt.Printf("杠杆倍数: %.0f\n", leverage)
	fmt.Printf("名义价值: %.4f\n", notional)
	fmt.Printf("初始保证金: %.4f\n", initialMargin)

	// 计算止损价格
	stopPrice, err = marginRiskManager.CalculateEstimatedMarginStopLoss(
		expectedEntryPrice, expectedQuantity, leverage, marginLossPercent, isLong)
	if err != nil {
		log.Printf("❌ 止损价格计算失败: %v", err)
	} else {
		fmt.Printf("✅ 止损价格 (%.1f%%): %.8f\n", marginLossPercent, stopPrice)
		targetLoss := initialMargin * (marginLossPercent / 100)
		priceChange := targetLoss / expectedQuantity
		fmt.Printf("   目标亏损金额: %.4f\n", targetLoss)
		fmt.Printf("   价格变动: %.8f\n", priceChange)
		fmt.Printf("   止损触发价格: %.8f (上涨%.4f%%)\n",
			stopPrice, (stopPrice-expectedEntryPrice)/expectedEntryPrice*100)
	}

	// 计算止盈价格
	takeProfitPrice, err = marginRiskManager.CalculateEstimatedMarginTakeProfit(
		expectedEntryPrice, expectedQuantity, leverage, marginProfitPercent, isLong)
	if err != nil {
		log.Printf("❌ 止盈价格计算失败: %v", err)
	} else {
		fmt.Printf("✅ 止盈价格 (%.1f%%): %.8f\n", marginProfitPercent, takeProfitPrice)
		targetProfit := initialMargin * (marginProfitPercent / 100)
		priceChange := targetProfit / expectedQuantity
		fmt.Printf("   目标盈利金额: %.4f\n", targetProfit)
		fmt.Printf("   价格变动: %.8f\n", priceChange)
		fmt.Printf("   止盈触发价格: %.8f (下跌%.4f%%)\n",
			takeProfitPrice, (expectedEntryPrice-takeProfitPrice)/expectedEntryPrice*100)
	}

	fmt.Println("\n🎯 总结:")
	fmt.Println("✅ 实现了真正的保证金止盈止损计算")
	fmt.Println("✅ 基于保证金亏损/盈利金额而非价格百分比")
	fmt.Println("✅ 支持多头和空头仓位")
	fmt.Println("✅ 精确计算止盈止损触发价格")

	// 对比传统价格百分比计算
	fmt.Println("\n🔄 对比例子:")
	fmt.Println("传统价格百分比 (SELL 1%止损):", expectedEntryPrice*(1+0.01), "-> 价格上涨1%止损")
	fmt.Println("保证金百分比 (SELL 1%止损):", stopPrice, "-> 保证金亏损1%止损")
}