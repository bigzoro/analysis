package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("📋 完整检查策略ID 33的所有配置")
	fmt.Println("=====================================")

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	var strategy pdb.TradingStrategy
	result := gdb.GormDB().Where("id = ?", 33).First(&strategy)
	if result.Error != nil {
		log.Fatalf("查询策略失败: %v", result.Error)
	}

	conditions := strategy.Conditions

	fmt.Printf("🎯 策略基本信息:\n")
	fmt.Printf("   ID: %d\n", strategy.ID)
	fmt.Printf("   名称: %s\n", strategy.Name)
	fmt.Printf("   用户ID: %d\n", strategy.UserID)
	fmt.Printf("   运行状态: %v\n", strategy.IsRunning)

	fmt.Printf("\n📊 传统策略配置:\n")
	fmt.Printf("   涨幅开空: %v\n", conditions.ShortOnGainers)
	fmt.Printf("   小市值涨幅开多: %v\n", conditions.LongOnSmallGainers)
	fmt.Printf("   合约涨幅开空策略: %v\n", conditions.FuturesPriceShortStrategyEnabled)
	if conditions.FuturesPriceShortStrategyEnabled {
		fmt.Printf("     最大排名: %d\n", conditions.FuturesPriceShortMaxRank)
		fmt.Printf("     最低资金费率: %.4f%%\n", conditions.FuturesPriceShortMinFundingRate*100)
		fmt.Printf("     杠杆倍数: %.1fx\n", conditions.FuturesPriceShortLeverage)
	}
	fmt.Printf("   交易类型: %s\n", conditions.TradingType)
	fmt.Printf("   资金费率过滤启用: %v\n", conditions.FundingRateFilterEnabled)

	fmt.Printf("\n⚙️  交易配置:\n")
	fmt.Printf("   杠杆配置: %v\n", conditions.EnableLeverage)
	fmt.Printf("   保证金模式: %s\n", conditions.MarginMode)
	fmt.Printf("   跳过已有持仓: %v\n", conditions.SkipHeldPositions)

	fmt.Printf("\n💰 盈利加仓配置:\n")
	fmt.Printf("   盈利加仓启用: %v\n", conditions.ProfitScalingEnabled)
	if conditions.ProfitScalingEnabled {
		fmt.Printf("     触发盈利百分比: %.2f%%\n", conditions.ProfitScalingPercent)
		fmt.Printf("     加仓金额: %.2f USDT\n", conditions.ProfitScalingAmount)
		fmt.Printf("     最大加仓次数: %d\n", conditions.ProfitScalingMaxCount)
		fmt.Printf("     当前已加仓次数: %d\n", conditions.ProfitScalingCurrentCount)
	}

	fmt.Printf("\n🛡️  传统止盈止损配置:\n")
	fmt.Printf("   止损启用: %v\n", conditions.EnableStopLoss)
	fmt.Printf("   止损百分比: %.2f%%\n", conditions.StopLossPercent)
	fmt.Printf("   止盈启用: %v\n", conditions.EnableTakeProfit)
	fmt.Printf("   止盈百分比: %.2f%%\n", conditions.TakeProfitPercent)

	fmt.Printf("\n🏦 保证金止盈止损配置:\n")
	fmt.Printf("   保证金损失止损启用: %v\n", conditions.EnableMarginLossStopLoss)
	fmt.Printf("   保证金损失止损百分比: %.2f%%\n", conditions.MarginLossStopLossPercent)
	fmt.Printf("   保证金盈利止盈启用: %v\n", conditions.EnableMarginProfitTakeProfit)
	fmt.Printf("   保证金盈利止盈百分比: %.2f%%\n", conditions.MarginProfitTakeProfitPercent)

	fmt.Printf("\n📈 整体仓位止盈止损配置:\n")
	fmt.Printf("   整体止损启用: %v\n", conditions.OverallStopLossEnabled)
	fmt.Printf("   整体止损百分比: %.2f%%\n", conditions.OverallStopLossPercent)
	fmt.Printf("   整体止盈百分比: %.2f%%\n", conditions.OverallTakeProfitPercent)

	fmt.Printf("\n🔧 其他参数配置:\n")
	fmt.Printf("   涨幅排名限制: %d\n", conditions.GainersRankLimit)
	fmt.Printf("   开多涨幅排名限制: %d\n", conditions.LongGainersRankLimit)
	fmt.Printf("   开空市值限制: %.0f万\n", conditions.MarketCapLimitShort/10000)
	fmt.Printf("   开多市值限制: %.0f万\n", conditions.MarketCapLimitLong/10000)
	fmt.Printf("   不开空市值阈值: %.0f万\n", conditions.MarketCapLimitShort/10000)
	fmt.Printf("   默认杠杆倍数: %d\n", conditions.DefaultLeverage)
	fmt.Printf("   最大持仓小时数: %d\n", conditions.MaxHoldHours)
	fmt.Printf("   最大仓位大小: %.2f%%\n", conditions.MaxPositionSize)

	fmt.Printf("\n✅ 配置检查完成!\n")
}