package main

import (
	"encoding/json"
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

func main() {
	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer gdb.Close()

	// 查询ID为22的策略
	var strategy pdb.TradingStrategy
	err = gdb.GormDB().First(&strategy, 22).Error
	if err != nil {
		log.Fatalf("查询策略失败: %v", err)
	}

	// 格式化输出
	fmt.Printf("策略ID: %d\n", strategy.ID)
	fmt.Printf("策略名称: %s\n", strategy.Name)
	fmt.Printf("创建时间: %s\n", strategy.CreatedAt)
	fmt.Printf("更新时间: %s\n", strategy.UpdatedAt)

	fmt.Println("\n策略条件详情:")
	conditionsJSON, _ := json.MarshalIndent(strategy.Conditions, "", "  ")
	fmt.Printf("%s\n", conditionsJSON)

	// 分析策略类型
	fmt.Println("\n策略分析:")
	conditions := strategy.Conditions

	if conditions.SpotContract {
		fmt.Println("✓ 需要现货+合约交易对")
	}

	if conditions.NoShortBelowMarketCap {
		fmt.Printf("✓ 市值<%d万不开空\n", conditions.MarketCapLimitShort)
	}

	if conditions.ShortOnGainers {
		fmt.Printf("✓ 涨幅前%d名 & 市值>%d万 → 开空 %.1f倍\n",
			conditions.GainersRankLimit, conditions.MarketCapLimitShort, conditions.ShortMultiplier)
	}

	if conditions.LongOnSmallGainers {
		fmt.Printf("✓ 市值<%d万 & 涨幅前%d名 → 开多 %.1f倍\n",
			conditions.MarketCapLimitLong, conditions.GainersRankLimitLong, conditions.LongMultiplier)
	}

	if conditions.CrossExchangeArbEnabled {
		fmt.Printf("✓ 跨交易所套利：价差超过%.1f%%\n", conditions.PriceDiffThreshold)
	}

	if conditions.SpotFutureArbEnabled {
		fmt.Printf("✓ 现货-合约套利：基差超过%.1f%%\n", conditions.BasisThreshold)
	}

	if conditions.TriangleArbEnabled {
		fmt.Printf("✓ 三角套利：阈值超过%.1f%%\n", conditions.TriangleThreshold)
	}

	if conditions.StatArbEnabled {
		fmt.Printf("✓ 统计套利：Z分数超过%.1f\n", conditions.ZscoreThreshold)
	}

	if conditions.EnableStopLoss {
		fmt.Printf("✓ 止损设置：%.1f%%\n", conditions.StopLossPercent)
	}

	if conditions.EnableTakeProfit {
		fmt.Printf("✓ 止盈设置：%.1f%%\n", conditions.TakeProfitPercent)
	}

	if conditions.EnableLeverage {
		fmt.Printf("✓ 杠杆交易：默认%d倍 (最大%d倍)\n",
			conditions.DefaultLeverage, conditions.MaxLeverage)
	}
}
