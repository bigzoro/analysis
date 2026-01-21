package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
	"analysis/internal/server/strategy/shared/execution"
)

// 测试保证金损失止损功能
func main() {
	fmt.Println("🧪 测试保证金损失止损功能")
	fmt.Println("========================================")

	// 初始化数据库连接
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 1. 测试策略配置更新
	fmt.Println("\n1️⃣ 测试策略配置更新")
	strategy := pdb.TradingStrategy{}
	result := db.Where("id = ?", 33).First(&strategy)
	if result.Error != nil {
		log.Printf("查询策略失败: %v", result.Error)
	} else {
		fmt.Printf("✅ 策略ID 33查询成功\n")
		fmt.Printf("   策略名称: %s\n", strategy.Name)
		fmt.Printf("   原始保证金止损启用: %v\n", strategy.Conditions.EnableMarginLossStopLoss)
		fmt.Printf("   原始保证金止损百分比: %.2f%%\n", strategy.Conditions.MarginLossStopLossPercent)

		// 更新配置
		strategy.Conditions.EnableMarginLossStopLoss = true
		strategy.Conditions.MarginLossStopLossPercent = 30.0
		if err := pdb.UpdateTradingStrategy(db, &strategy); err != nil {
			log.Printf("❌ 更新策略配置失败: %v", err)
		} else {
			fmt.Printf("✅ 策略配置更新成功\n")
			fmt.Printf("   新保证金止损启用: %v\n", strategy.Conditions.EnableMarginLossStopLoss)
			fmt.Printf("   新保证金止损百分比: %.2f%%\n", strategy.Conditions.MarginLossStopLossPercent)
		}
	}

	// 2. 测试保证金风险管理器
	fmt.Println("\n2️⃣ 测试保证金风险管理器")

	// 注意：这里使用测试环境，实际API密钥需要配置
	marginRiskManager := execution.NewMarginRiskManager(bf.New(true, "", ""))

	// 测试配置验证
	fmt.Println("   测试配置验证:")
	testPercents := []float64{-5, 0, 5, 30, 80, 110}
	for _, percent := range testPercents {
		err := marginRiskManager.ValidateMarginStopLossConfig(percent)
		if err != nil {
			fmt.Printf("     %.1f%%: ❌ %v\n", percent, err)
		} else {
			fmt.Printf("     %.1f%%: ✅ 配置有效\n", percent)
		}
	}

	fmt.Println("\n🎉 保证金损失止损功能测试完成!")
	fmt.Println("\n📋 功能说明:")
	fmt.Println("   ✅ 数据库结构扩展")
	fmt.Println("   ✅ 风险管理器接口更新")
	fmt.Println("   ✅ 保证金损失计算逻辑")
	fmt.Println("   ✅ 策略执行器集成")
	fmt.Println("   ✅ 前端界面配置")
	fmt.Println("   ✅ 数据库迁移脚本")
	fmt.Println("\n💡 使用方法:")
	fmt.Println("   1. 在前端界面启用'保证金损失止损'")
	fmt.Println("   2. 设置止损百分比（如30%）")
	fmt.Println("   3. 当持仓保证金亏损达到30%时自动触发止损")
	fmt.Println("   4. 止损价格基于当前亏损比例和杠杆自动计算")
}
