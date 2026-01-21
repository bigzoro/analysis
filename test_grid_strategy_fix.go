package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"analysis/internal/db"
	"analysis/internal/server"
)

func main() {
	fmt.Println("=== 测试网格策略修复效果 ===")

	// 初始化数据库连接
	dbConfig := db.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}

	gdb, err := db.OpenMySQL(dbConfig)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	// 创建服务器实例
	srv := &server.Server{
		DB: gdb.GormDB(),
	}

	// 获取策略数据
	var strategy db.TradingStrategy
	if err := gdb.GormDB().Where("id = ?", 29).First(&strategy).Error; err != nil {
		log.Fatalf("获取策略失败: %v", err)
	}

	fmt.Printf("📋 策略信息:\n")
	fmt.Printf("  ID: %d\n", strategy.ID)
	fmt.Printf("  名称: %s\n", strategy.Name)
	fmt.Printf("  网格启用: %v\n", strategy.Conditions.GridTradingEnabled)
	fmt.Printf("  网格上限: %.8f\n", strategy.Conditions.GridUpperPrice)
	fmt.Printf("  网格下限: %.8f\n", strategy.Conditions.GridLowerPrice)
	fmt.Printf("  网格层数: %d\n", strategy.Conditions.GridLevels)
	fmt.Printf("  投资金额: %.2f\n", strategy.Conditions.GridInvestmentAmount)

	// 创建调度器实例
	scheduler := &server.OrderScheduler{
		Server: srv,
	}

	// 获取市场数据
	symbol := "FILUSDT"
	marketData, err := scheduler.GetMarketDataForStrategy(symbol)
	if err != nil {
		log.Printf("获取市场数据失败，使用默认值: %v", err)
		// 使用默认市场数据
		marketData = server.StrategyMarketData{
			Symbol:      symbol,
			MarketCap:   1000000000,
			GainersRank: 50,
			HasSpot:     true,
			HasFutures:  true,
		}
	}

	fmt.Printf("\n📊 市场数据:\n")
	fmt.Printf("  交易对: %s\n", marketData.Symbol)
	fmt.Printf("  市值: %.0f\n", marketData.MarketCap)
	fmt.Printf("  排名: %d\n", marketData.GainersRank)
	fmt.Printf("  有现货: %v\n", marketData.HasSpot)
	fmt.Printf("  有期货: %v\n", marketData.HasFutures)

	// 创建网格策略执行器
	executor := &server.GridTradingStrategyExecutor{}

	// 测试策略执行
	fmt.Printf("\n🔬 执行网格策略测试...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := executor.ExecuteFull(ctx, srv, symbol, marketData, strategy.Conditions, &strategy)

	fmt.Printf("\n🎯 策略执行结果:\n")
	fmt.Printf("  动作: %s\n", result.Action)
	fmt.Printf("  原因: %s\n", result.Reason)
	fmt.Printf("  倍数: %.2f\n", result.Multiplier)

	// 分析结果
	fmt.Printf("\n📈 结果分析:\n")
	switch result.Action {
	case "buy":
		fmt.Printf("✅ 修复成功! 策略返回买入信号\n")
		fmt.Printf("🎯 预期效果: 调度器将创建买入订单\n")
	case "sell":
		fmt.Printf("✅ 修复成功! 策略返回卖出信号\n")
		fmt.Printf("🎯 预期效果: 调度器将创建卖出订单\n")
	case "no_op":
		fmt.Printf("⚠️ 策略返回观望 - 分析原因:\n")
		fmt.Printf("   原因: %s\n", result.Reason)

		// 检查是否是价格范围问题
		currentPrice, priceErr := srv.GetCurrentPrice(ctx, symbol, "spot")
		if priceErr != nil {
			fmt.Printf("❌ 无法获取价格: %v\n", priceErr)
		} else {
			fmt.Printf("   当前价格: %.8f\n", currentPrice)
			fmt.Printf("   网格范围: [%.8f, %.8f]\n", strategy.Conditions.GridLowerPrice, strategy.Conditions.GridUpperPrice)

			if currentPrice >= strategy.Conditions.GridLowerPrice && currentPrice <= strategy.Conditions.GridUpperPrice {
				fmt.Printf("   ✅ 价格在范围内，可能是评分不足\n")
			} else {
				fmt.Printf("   ❌ 价格超出范围\n")
			}
		}
	case "skip":
		fmt.Printf("❌ 策略跳过执行 - 检查配置:\n")
		fmt.Printf("   原因: %s\n", result.Reason)
	default:
		fmt.Printf("⚠️ 未知动作: %s\n", result.Action)
	}

	// 验证网格参数
	fmt.Printf("\n🔍 网格参数验证:\n")
	if strategy.Conditions.GridUpperPrice > 0 && strategy.Conditions.GridLowerPrice > 0 {
		fmt.Printf("✅ 网格参数读取正常\n")
		if strategy.Conditions.GridUpperPrice > strategy.Conditions.GridLowerPrice {
			fmt.Printf("✅ 网格范围有效: [%.4f, %.4f]\n", strategy.Conditions.GridLowerPrice, strategy.Conditions.GridUpperPrice)
		} else {
			fmt.Printf("❌ 网格范围无效: 上限 <= 下限\n")
		}
	} else {
		fmt.Printf("❌ 网格参数异常: 存在零值或负值\n")
	}

	// 总结
	fmt.Printf("\n📋 测试总结:\n")
	if result.Action == "buy" || result.Action == "sell" {
		fmt.Printf("✅ 网格策略修复成功!\n")
		fmt.Printf("✅ 策略现在能够产生交易信号\n")
		fmt.Printf("✅ 调度器应该能够创建订单\n")
	} else {
		fmt.Printf("⚠️ 网格策略仍需进一步调试\n")
		fmt.Printf("💡 建议检查: 价格数据、评分计算、阈值设置\n")
	}

	fmt.Printf("\n🎯 下一步行动:\n")
	fmt.Printf("1. 运行实际策略调度测试\n")
	fmt.Printf("2. 检查订单是否成功创建\n")
	fmt.Printf("3. 验证交易执行结果\n")
}
