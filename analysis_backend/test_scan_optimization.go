package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
	"analysis/internal/server"
)

func main() {
	fmt.Println("🧪 均值回归策略扫描优化效果测试")
	fmt.Println("=================================")

	// 初始化数据库连接
	dsn := "root:@tcp(localhost:3306)/analysis?charset=utf8mb4&parseTime=True&loc=Local"
	dbOptions := db.Options{
		DSN:             dsn,
		Automigrate:     false,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	gdb, err := db.OpenMySQL(dbOptions)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	// 创建简化的服务器实例用于测试
	srv := &server.Server{
		DB: gdb,
	}

	// 手动创建扫描器注册表
	scannerRegistry := &server.StrategyScannerRegistry{}
	scannerRegistry.Initialize()

	// 创建均值回归策略扫描器
	meanReversionScanner := &server.MeanReversionStrategyScanner{
		Server: srv,
	}
	scannerRegistry.RegisterScanner("mean_reversion", meanReversionScanner)
	srv.ScannerRegistry = scannerRegistry

	// 创建测试策略 (均值回归增强模式，自适应子模式)
	testStrategy := &pdb.TradingStrategy{
		Name: "测试均值回归策略",
		Conditions: pdb.StrategyConditions{
			MeanReversionEnabled: true,
			MeanReversionMode:    "enhanced",
			MeanReversionSubMode: "adaptive",
			SpotContract:         true, // 必须有现货+合约

			// 技术指标配置
			MRBollingerBandsEnabled: true,
			MRRSIEnabled:           true,
			MRPeriod:               20,
			MRBollingerMultiplier:  2.0,
			MRRSIOverbought:        75,
			MRRSIOversold:          25,
			MRMinReversionStrength: 0.15,

			// 增强功能
			MarketEnvironmentDetection: true,
			IntelligentWeights:         true,
			AdvancedRiskManagement:     true,
		},
	}

	fmt.Println("\n📋 测试策略配置:")
	fmt.Printf("• 策略名称: %s\n", testStrategy.Name)
	fmt.Printf("• 策略模式: %s (%s)\n", testStrategy.Conditions.MeanReversionMode, testStrategy.Conditions.MeanReversionSubMode)
	fmt.Printf("• 技术指标: 布林带(RSI) 周期:%d 倍数:%.1f RSI:%d/%d\n",
		testStrategy.Conditions.MRPeriod,
		testStrategy.Conditions.MRBollingerMultiplier,
		testStrategy.Conditions.MRRSIOverbought,
		testStrategy.Conditions.MRRSIOversold)
	fmt.Printf("• 现货+合约要求: %t\n", testStrategy.Conditions.SpotContract)

	// 选择扫描器
	scanner := srv.ScannerRegistry.SelectScanner(testStrategy)
	if scanner == nil {
		log.Fatal("未找到合适的扫描器")
	}

	fmt.Printf("\n🔍 使用扫描器: %s\n", scanner.GetStrategyType())

	// 执行扫描
	fmt.Println("\n⏳ 开始扫描 (这可能需要一些时间)...")
	scanStartTime := time.Now()

	eligibleSymbols, err := scanner.Scan(context.Background(), testStrategy)
	scanDuration := time.Since(scanStartTime)

	if err != nil {
		log.Fatalf("扫描失败: %v", err)
	}

	fmt.Printf("\n✅ 扫描完成!\n")
	fmt.Printf("• 耗时: %v\n", scanDuration)
	fmt.Printf("• 发现符合条件的币种: %d个\n", len(eligibleSymbols))

	// 分析扫描结果
	fmt.Println("\n📊 扫描结果分析:")
	fmt.Println("===============")

	if len(eligibleSymbols) == 0 {
		fmt.Println("⚠️  未发现任何符合条件的币种")
		return
	}

	// 统计主流币种vs新兴币种
	majorCoinCount := 0
	altCoinCount := 0
	totalScore := 0.0

	majorCoins := []string{
		"BTC", "ETH", "BNB", "SOL", "ADA", "XRP", "DOT", "DOGE", "AVAX", "LINK",
		"LTC", "ICP", "NEAR", "FTM", "HBAR", "FIL", "ETC", "ALGO", "VET",
		"OP", "ARB", "MATIC", "APT", "SUI", "SEI", "TIA", "ZKS", "IMX", "ONDO",
		"INJ", "PEPE", "BONK", "WIF", "MEW", "BRETT", "PENGU", "MOTHER", "TURBO", "GIGA",
	}

	fmt.Println("扫描到的币种列表:")
	fmt.Println("-----------------")

	for i, symbol := range eligibleSymbols {
		// 提取基础币种名称
		baseSymbol := symbol.Symbol
		if len(baseSymbol) > 4 && baseSymbol[len(baseSymbol)-4:] == "USDT" {
			baseSymbol = baseSymbol[:len(baseSymbol)-4]
		}

		// 判断是否为主流币种
		isMajor := false
		for _, coin := range majorCoins {
			if baseSymbol == coin {
				isMajor = true
				majorCoinCount++
				break
			}
		}
		if !isMajor {
			altCoinCount++
		}

		coinType := "新兴币种"
		if isMajor {
			coinType = "主流币种"
		}

		fmt.Printf("%2d. %-12s (%s)\n", i+1, symbol.Symbol, coinType)

		// 累加评分 (如果有的话)
		if symbol.Score > 0 {
			totalScore += symbol.Score
		}
	}

	// 统计分析
	fmt.Println("\n📈 统计分析:")
	fmt.Println("============")
	fmt.Printf("• 主流币种: %d个 (%.1f%%)\n", majorCoinCount, float64(majorCoinCount)/float64(len(eligibleSymbols))*100)
	fmt.Printf("• 新兴币种: %d个 (%.1f%%)\n", altCoinCount, float64(altCoinCount)/float64(len(eligibleSymbols))*100)

	if majorCoinCount > 0 && altCoinCount > 0 {
		ratio := float64(altCoinCount) / float64(majorCoinCount)
		fmt.Printf("• 新兴vs主流比例: %.2f:1\n", ratio)
	}

	avgScore := 0.0
	if len(eligibleSymbols) > 0 {
		avgScore = totalScore / float64(len(eligibleSymbols))
		if avgScore > 0 {
			fmt.Printf("• 平均评分: %.3f\n", avgScore)
		}
	}

	// 与优化前的结果对比 (基于之前的分析)
	fmt.Println("\n🔄 与优化前对比:")
	fmt.Println("===============")

	// 优化前的结果 (基于之前的分析)
	oldMajorCount := 3  // AVAX, LINK, ICP
	oldAltCount := 8    // 其他币种
	oldTotalCount := 11 // 假设只显示了11个

	fmt.Printf("• 优化前主流币种: %d个 → 优化后: %d个 ", oldMajorCount, majorCoinCount)
	if majorCoinCount < oldMajorCount {
		fmt.Printf("(✅ 减少%d个)\n", oldMajorCount-majorCoinCount)
	} else if majorCoinCount > oldMajorCount {
		fmt.Printf("(⚠️ 增加%d个)\n", majorCoinCount-oldMajorCount)
	} else {
		fmt.Printf("(➖ 持平)\n")
	}

	fmt.Printf("• 优化前新兴币种: %d个 → 优化后: %d个 ", oldAltCount, altCoinCount)
	if altCoinCount > oldAltCount {
		fmt.Printf("(✅ 增加%d个)\n", altCoinCount-oldAltCount)
	} else if altCoinCount < oldAltCount {
		fmt.Printf("(⚠️ 减少%d个)\n", oldAltCount-altCoinCount)
	} else {
		fmt.Printf("(➖ 持平)\n")
	}

	// 计算优化效果
	if majorCoinCount > 0 {
		newRatio := float64(altCoinCount) / float64(majorCoinCount)
		oldRatio := float64(oldAltCount) / float64(oldMajorCount)
		ratioChange := (newRatio - oldRatio) / oldRatio * 100

		fmt.Printf("• 新兴vs主流比例变化: %.2f%% ", ratioChange)
		if ratioChange > 0 {
			fmt.Printf("(✅ 改善)\n")
		} else {
			fmt.Printf("(⚠️ 恶化)\n")
		}
	}

	fmt.Println("\n🎯 优化效果评估:")
	fmt.Println("===============")

	if majorCoinCount <= 2 && altCoinCount >= 10 {
		fmt.Println("✅ 优秀: 主流币种比例显著降低，新兴币种优势明显")
	} else if majorCoinCount <= 4 && altCoinCount >= 8 {
		fmt.Println("✅ 良好: 主流币种比例适中，优化效果明显")
	} else if majorCoinCount <= 6 && altCoinCount >= 6 {
		fmt.Println("⚠️ 一般: 主流币种比例仍较高，需要进一步调整")
	} else {
		fmt.Println("❌ 需要改进: 主流币种占比过高，优化效果不佳")
	}

	fmt.Println("\n🚀 第一阶段优化测试完成!")
}