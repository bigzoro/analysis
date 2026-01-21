package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	// 加载配置
	var cfg config.Config
	config.MustLoad("config.yaml", &cfg)
	config.ApplyProxy(&cfg)

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  false,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer gdb.Close()

	fmt.Println("开始生成测试策略数据...")

	// 检查是否已有策略
	var existingStrategies []pdb.TradingStrategy
	err = gdb.GormDB().Find(&existingStrategies).Error
	if err != nil {
		log.Fatalf("查询现有策略失败: %v", err)
	}

	strategies := []pdb.TradingStrategy{
		{
			UserID:      1,
			Name:        "均值回归策略",
			Description: "基于均值回归原理的交易策略",
			IsRunning:   false,
		},
		{
			UserID:      1,
			Name:        "传统交易策略",
			Description: "经典的技术分析交易策略",
			IsRunning:   false,
		},
		{
			UserID:      1,
			Name:        "均线策略",
			Description: "基于移动平均线的趋势跟踪策略",
			IsRunning:   false,
		},
		{
			UserID:      1,
			Name:        "网格交易策略",
			Description: "网格交易策略",
			IsRunning:   false,
		},
	}

	// 创建策略
	for i := range strategies {
		strategy := &strategies[i]

		// 检查是否已存在
		var existing pdb.TradingStrategy
		err = gdb.GormDB().Where("name = ? AND user_id = ?", strategy.Name, strategy.UserID).First(&existing).Error
		if err == nil {
			// 已存在，使用现有ID
			strategy.ID = existing.ID
			fmt.Printf("策略已存在: %s (ID: %d)\n", strategy.Name, strategy.ID)
			continue
		}

		// 创建新策略
		err = gdb.GormDB().Create(strategy).Error
		if err != nil {
			log.Printf("创建策略失败 %s: %v", strategy.Name, err)
			continue
		}
		fmt.Printf("创建策略成功: %s (ID: %d)\n", strategy.Name, strategy.ID)
	}

	// 为每个策略生成一些执行记录
	rand.Seed(time.Now().UnixNano())
	now := time.Now()

	strategyConfigs := []struct {
		name      string
		winRate   float64
		totalPnL  float64
		executions int
	}{
		{"均值回归策略", 0.62, 0.023, 15},
		{"传统交易策略", 0.58, 0.018, 12},
		{"均线策略", 0.54, 0.015, 10},
		{"网格交易策略", 0.67, 0.032, 8},
	}

	for _, config := range strategyConfigs {
		// 找到策略
		var strategy pdb.TradingStrategy
		err = gdb.GormDB().Where("name = ?", config.name).First(&strategy).Error
		if err != nil {
			log.Printf("未找到策略 %s: %v", config.name, err)
			continue
		}

		fmt.Printf("为策略 %s 生成 %d 条执行记录...\n", config.name, config.executions)

		// 生成执行记录
		for i := 0; i < config.executions; i++ {
			// 随机生成一些变化
			winRateVariation := (rand.Float64() - 0.5) * 0.1 // ±5%
			pnlVariation := (rand.Float64() - 0.5) * 0.01     // ±0.5%

			execution := pdb.StrategyExecution{
				StrategyID:     strategy.ID,
				UserID:         strategy.UserID,
				Status:         "completed",
				StartTime:      now.Add(-time.Duration(i*24) * time.Hour),
				EndTime:        &[]time.Time{now.Add(-time.Duration(i*24-2) * time.Hour)}[0],
				Duration:       7200, // 2小时
				TotalOrders:    rand.Intn(20) + 5,
				SuccessOrders:  0, // 稍后计算
				TotalPnL:       config.totalPnL + pnlVariation,
				WinRate:        config.winRate + winRateVariation,
				PnlPercentage:  (config.totalPnL + pnlVariation) * 100,
				TotalInvestment: 1000.0 + rand.Float64()*500,
				CurrentValue:   0, // 稍后计算
			}

			// 计算成功订单数
			execution.SuccessOrders = int(float64(execution.TotalOrders) * execution.WinRate / 100)

			// 计算当前价值
			execution.CurrentValue = execution.TotalInvestment + execution.TotalPnL*execution.TotalInvestment

			err = gdb.GormDB().Create(&execution).Error
			if err != nil {
				log.Printf("创建执行记录失败: %v", err)
			}
		}
	}

	fmt.Println("测试数据生成完成！")
	fmt.Println("现在重新启动应用，策略推荐将使用真实的性能数据。")
}