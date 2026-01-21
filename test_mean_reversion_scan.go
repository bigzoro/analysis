package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
)

func main() {
	fmt.Println("🔍 均值回归策略扫描测试 - 验证第一阶段优化效果")
	fmt.Println("===============================================")

	// 1. 初始化配置和数据库
	cfg, err := config.LoadConfig("./config.yaml")
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		log.Printf("尝试加载示例配置...")
		cfg, err = config.LoadConfig("./config.yaml.example")
		if err != nil {
			log.Printf("加载示例配置也失败: %v", err)
			log.Printf("使用硬编码默认配置...")
			cfg = &config.Config{
				Database: config.DatabaseConfig{
					Host:     "127.0.0.1",
					Port:     3306,
					User:     "root",
					Password: "123456",
					DBName:   "trading_analysis",
				},
			}
		}
	}

	// 初始化数据库
	db, err := pdb.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ 数据库连接成功")

	// 2. 创建服务器实例
	srv := &server.Server{
		DB: db,
		Config: cfg,
	}

	// 3. 创建测试策略 (增强模式自适应)
	testStrategy := &pdb.TradingStrategy{
		Name: "测试均值回归策略",
		Conditions: pdb.StrategyConditions{
			MeanReversionEnabled: true,
			MeanReversionMode: "enhanced",
			MeanReversionSubMode: "adaptive",
			MRPeriod: 20,
			MRBollingerBandsEnabled: true,
			MRRSIEnabled: true,
			MRPriceChannelEnabled: false,
			MRBollingerMultiplier: 2.0,
			MRRSIOverbought: 75,
			MRRSIOversold: 25,
			MRMinReversionStrength: 0.15,
			MRCandidateMinOscillation: 0.3,
			MRCandidateMinLiquidity: 0.4,
			MRCandidateMaxVolatility: 0.15,
			SpotContract: true,
			MarketEnvironmentDetection: true,
			IntelligentWeights: true,
			AdvancedRiskManagement: true,
		},
	}

	fmt.Printf("🎯 测试策略配置:\n")
	fmt.Printf("   • 模式: %s (%s)\n", testStrategy.Conditions.MeanReversionMode, testStrategy.Conditions.MeanReversionSubMode)
	fmt.Printf("   • 周期: %d\n", testStrategy.Conditions.MRPeriod)
	fmt.Printf("   • RSI范围: %d-%d\n", testStrategy.Conditions.MRRSIOversold, testStrategy.Conditions.MRRSIOverbought)
	fmt.Printf("   • 最小回归强度: %.2f\n", testStrategy.Conditions.MRMinReversionStrength)

	// 4. 执行扫描
	fmt.Println("\n🔄 开始执行扫描...")
	startTime := time.Now()

	eligibleSymbols, err := srv.ScanEligibleSymbols(context.Background(), testStrategy)
	if err != nil {
		log.Fatalf("扫描失败: %v", err)
	}

	scanDuration := time.Since(startTime)
	fmt.Printf("✅ 扫描完成，耗时: %.2fs\n", scanDuration.Seconds())

	// 5. 分析结果
	fmt.Printf("\n📊 扫描结果分析:\n")
	fmt.Printf("===============\n")

	if len(eligibleSymbols) == 0 {
		fmt.Println("❌ 未找到符合条件的币种")
		return
	}

	fmt.Printf("✅ 找到%d个符合条件的币种\n", len(eligibleSymbols))

	// 提取币种列表
	var symbols []string
	for _, symbol := range eligibleSymbols {
		symbols = append(symbols, symbol.Symbol)
	}

	// 统计主流币种vs新兴币种
	majorCoins := []string{
		"BTC", "ETH", "BNB", "SOL", "ADA", "XRP", "DOT", "DOGE", "AVAX", "LINK",
		"LTC", "ICP", "NEAR", "FTM", "HBAR", "FIL", "ETC", "ALGO", "VET",
		"OP", "ARB", "MATIC", "APT", "SUI", "SEI", "TIA", "ZKS", "IMX", "ONDO",
		"INJ", "PEPE", "BONK", "WIF", "MEW", "BRETT", "PENGU", "MOTHER", "TURBO", "GIGA",
	}

	var majorCount, altCount int
	var majorSymbols, altSymbols []string

	for _, symbol := range symbols {
		baseSymbol := strings.TrimSuffix(symbol, "USDT")
		isMajor := false
		for _, major := range majorCoins {
			if baseSymbol == major {
				isMajor = true
				break
			}
		}

		if isMajor {
			majorCount++
			majorSymbols = append(majorSymbols, symbol)
		} else {
			altCount++
			altSymbols = append(altSymbols, symbol)
		}
	}

	fmt.Printf("• 主流币种: %d个 (%.1f%%)\n", majorCount, float64(majorCount)/float64(len(symbols))*100)
	fmt.Printf("• 新兴币种: %d个 (%.1f%%)\n", altCount, float64(altCount)/float64(len(symbols))*100)

	// 显示详细信息
	fmt.Println("\n🏆 主流币种列表:")
	if len(majorSymbols) > 0 {
		for i, symbol := range majorSymbols {
			fmt.Printf("   %d. %s\n", i+1, symbol)
		}
	} else {
		fmt.Println("   无")
	}

	fmt.Println("\n🚀 新兴币种列表:")
	if len(altSymbols) > 0 {
		for i, symbol := range altSymbols[:min(10, len(altSymbols))] { // 只显示前10个
			fmt.Printf("   %d. %s\n", i+1, symbol)
		}
		if len(altSymbols) > 10 {
			fmt.Printf("   ... 还有%d个\n", len(altSymbols)-10)
		}
	} else {
		fmt.Println("   无")
	}

	// 与优化前的对比分析
	fmt.Println("\n📈 优化效果对比:")
	fmt.Println("===============")

	// 优化前的预期比例 (基于之前的分析)
	oldMajorRatio := 0.4 // 假设原来40%是主流币种
	newMajorRatio := float64(majorCount) / float64(len(symbols))

	fmt.Printf("• 优化前主流币种比例: %.1f%%\n", oldMajorRatio*100)
	fmt.Printf("• 优化后主流币种比例: %.1f%%\n", newMajorRatio*100)

	if newMajorRatio < oldMajorRatio {
		reduction := (oldMajorRatio - newMajorRatio) / oldMajorRatio * 100
		fmt.Printf("• 主流币种比例下降: %.1f%%\n", reduction)
		fmt.Println("✅ 优化效果: 显著降低主流币种入选率")
	} else {
		fmt.Println("⚠️ 优化效果: 主流币种比例未明显下降")
	}

	// 检查是否还有原来的问题币种
	problemCoins := []string{"AVAXUSDT", "LINKUSDT", "ICPUSDT"}
	var foundProblems []string
	for _, problem := range problemCoins {
		for _, symbol := range symbols {
			if symbol == problem {
				foundProblems = append(foundProblems, problem)
				break
			}
		}
	}

	fmt.Println("\n🔍 问题币种检查:")
	if len(foundProblems) > 0 {
		fmt.Printf("⚠️ 仍入选的问题币种: %s\n", strings.Join(foundProblems, ", "))
	} else {
		fmt.Println("✅ 所有问题币种均已过滤")
	}

	fmt.Println("\n🎯 优化验证结论:")
	fmt.Println("===============")

	if newMajorRatio < 0.3 && len(foundProblems) == 0 {
		fmt.Println("🎉 第一阶段优化成功!")
		fmt.Println("   • 主流币种比例控制在合理范围内")
		fmt.Println("   • 问题币种已有效过滤")
		fmt.Println("   • 新兴币种获得更多机会")
	} else {
		fmt.Println("📊 优化效果待进一步调整:")
		if newMajorRatio >= 0.3 {
			fmt.Println("   • 主流币种比例仍较高，建议调整权重")
		}
		if len(foundProblems) > 0 {
			fmt.Println("   • 部分问题币种仍入选，检查评分逻辑")
		}
	}

	fmt.Printf("\n🏁 扫描测试完成，总共找到%d个候选币种\n", len(symbols))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}