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

	// 查询ID为21的策略
	var strategy pdb.TradingStrategy
	err = gdb.GormDB().First(&strategy, 21).Error
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
	fmt.Println("\n========== 策略分析 ==========")
	conditions := strategy.Conditions

	// 基本交易类型
	fmt.Println("📊 交易类型分析:")
	if conditions.SpotContract {
		fmt.Println("✓ 需要现货+合约交易对")
	}

	if conditions.FuturesSpotArbEnabled {
		fmt.Println("✓ 现货-期货套利策略")
	}

	if conditions.CrossExchangeArbEnabled {
		fmt.Printf("✓ 跨交易所套利：价差超过%.1f%%\n", conditions.PriceDiffThreshold)
	}

	if conditions.TriangleArbEnabled {
		fmt.Printf("✓ 三角套利：阈值超过%.1f%%\n", conditions.TriangleThreshold)
	}

	if conditions.StatArbEnabled {
		fmt.Printf("✓ 统计套利：Z分数超过%.1f\n", conditions.ZscoreThreshold)
	}

	// 做多策略分析
	fmt.Println("\n💹 做多策略分析:")
	if conditions.LongOnSmallGainers {
		fmt.Printf("✓ 小盘股涨幅策略：市值<%d万 & 涨幅前%d名 → 开多 %.1f倍\n",
			conditions.MarketCapLimitLong, conditions.GainersRankLimitLong, conditions.LongMultiplier)
	}

	// 做空策略分析
	fmt.Println("\n📉 做空策略分析:")
	if conditions.ShortOnGainers {
		fmt.Printf("✓ 热门股做空策略：涨幅前%d名 & 市值>%d万 → 开空 %.1f倍\n",
			conditions.GainersRankLimit, conditions.MarketCapLimitShort, conditions.ShortMultiplier)
	}

	if conditions.NoShortBelowMarketCap {
		fmt.Printf("✓ 市值保护：市值<%d万不开空\n", conditions.MarketCapLimitShort)
	}

	// 网格交易分析
	fmt.Println("\n🔄 网格交易分析:")
	if conditions.GridTradingEnabled {
		fmt.Printf("✓ 网格交易启用：每格间距%.1f%%\n", conditions.GridSpacing)
		if conditions.GridLevels > 0 {
			fmt.Printf("✓ 网格层数：%d层\n", conditions.GridLevels)
		}
		if conditions.GridMinVolume > 0 {
			fmt.Printf("✓ 最小交易量：%.2f\n", conditions.GridMinVolume)
		}
	}

	// 风险控制分析
	fmt.Println("\n🛡️ 风险控制分析:")
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

	if conditions.MaxPositionSize > 0 {
		fmt.Printf("✓ 最大仓位：%.1f%%\n", conditions.MaxPositionSize)
	}

	if conditions.DailyLossLimit > 0 {
		fmt.Printf("✓ 日亏损限制：%.1f%%\n", conditions.DailyLossLimit)
	}

	// 技术指标分析
	fmt.Println("\n📈 技术指标分析:")
	if conditions.RSIEnabled {
		fmt.Printf("✓ RSI指标：超卖<%d 开多，超买>%d 开空\n",
			conditions.RSIOversold, conditions.RSIBuySignal)
	}

	if conditions.MACD {
		fmt.Println("✓ MACD指标启用")
	}

	if conditions.BollingerBands {
		fmt.Printf("✓ 布林带指标：标准差%.1f倍\n", conditions.BBStdDev)
	}

	if conditions.VolumeAnalysis {
		fmt.Println("✓ 成交量分析启用")
	}

	// 市场条件分析
	fmt.Println("\n🌍 市场条件分析:")
	if conditions.VolatilityFilter {
		fmt.Printf("✓ 波动率过滤：最小波动率%.1f%%\n", conditions.MinVolatility)
	}

	if conditions.LiquidityFilter {
		fmt.Printf("✓ 流动性过滤：最小流动性%.2f\n", conditions.MinLiquidity)
	}

	if conditions.MarketCapFilter {
		fmt.Printf("✓ 市值过滤：%d万-%d万\n",
			conditions.MinMarketCap, conditions.MaxMarketCap)
	}

	if conditions.VolumeFilter {
		fmt.Printf("✓ 成交量过滤：最小%d万\n", conditions.MinVolume)
	}

	// 运行配置分析
	fmt.Println("\n⚙️ 运行配置分析:")
	fmt.Printf("运行间隔：%d分钟\n", strategy.RunInterval)
	fmt.Printf("运行状态：%t\n", strategy.IsRunning)
	if strategy.LastRunAt != nil {
		fmt.Printf("最后运行：%s\n", strategy.LastRunAt)
	}

	// 策略总结
	fmt.Println("\n========== 策略总结 ==========")
	fmt.Printf("策略名称：%s (ID:%d)\n", strategy.Name, strategy.ID)

	// 判断主要策略类型
	if conditions.ShortOnGainers && conditions.NoShortBelowMarketCap {
		fmt.Println("主要类型：📉 反转做空策略 - 做空热门股，保护小盘股")
	} else if conditions.LongOnSmallGainers {
		fmt.Println("主要类型：💹 价值投资策略 - 投资小盘潜力股")
	} else if conditions.FuturesSpotArbEnabled {
		fmt.Println("主要类型：🔄 套利策略 - 现货期货价差套利")
	} else if conditions.GridTradingEnabled {
		fmt.Println("主要类型：📊 网格交易策略 - 震荡行情获利")
	} else if conditions.StatArbEnabled {
		fmt.Println("主要类型：📈 统计套利策略 - 基于统计模型")
	} else {
		fmt.Println("主要类型：🤔 混合策略 - 需要进一步分析")
	}

	// 风险等级评估
	riskLevel := "低风险"
	if conditions.EnableLeverage && conditions.MaxLeverage > 5 {
		riskLevel = "高风险"
	} else if conditions.EnableLeverage || conditions.ShortOnGainers {
		riskLevel = "中等风险"
	}
	fmt.Printf("风险等级：%s\n", riskLevel)

	// 适用市场环境
	marketEnv := "震荡市"
	if conditions.LongOnSmallGainers && conditions.ShortOnGainers {
		marketEnv = "多空皆宜"
	} else if conditions.ShortOnGainers {
		marketEnv = "熊市/调整市"
	} else if conditions.GridTradingEnabled {
		marketEnv = "震荡市"
	}
	fmt.Printf("适用环境：%s\n", marketEnv)
}