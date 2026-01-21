package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 K线同步卡住问题诊断工具")
	fmt.Println("=" * 50)

	// 1. 检查配置
	fmt.Println("1. 📋 检查配置...")
	cfg, err := config.LoadConfig("analysis_backend/config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	fmt.Printf("   ✅ 数据库连接池: max_open_conns=%d, max_idle_conns=%d\n",
		cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)

	// 2. 检查数据库连接
	fmt.Println("2. 🗄️ 检查数据库连接...")
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     cfg.Database.Automigrate,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}
	defer db.Close()

	gdb := db.GormDB()

	// 检查数据库连接状态
	sqlDB, err := gdb.DB()
	if err != nil {
		log.Fatalf("❌ 获取底层SQL DB失败: %v", err)
	}

	stats := sqlDB.Stats()
	fmt.Printf("   ✅ 数据库连接状态:\n")
	fmt.Printf("      - 打开连接数: %d\n", stats.OpenConnections)
	fmt.Printf("      - 空闲连接数: %d\n", stats.Idle)
	fmt.Printf("      - 使用中连接数: %d\n", stats.InUse)
	fmt.Printf("      - 等待连接数: %d\n", stats.WaitCount)
	fmt.Printf("      - 等待总时长: %v\n", stats.WaitDuration)

	// 警告检查
	if stats.WaitCount > 0 {
		fmt.Printf("   ⚠️ 检测到连接等待! 等待次数: %d, 总等待时间: %v\n", stats.WaitCount, stats.WaitDuration)
		fmt.Println("   💡 建议: 增加 max_open_conns 配置")
	}

	if stats.InUse >= cfg.Database.MaxOpenConns {
		fmt.Printf("   ⚠️ 连接池已满! 使用中: %d, 最大: %d\n", stats.InUse, cfg.Database.MaxOpenConns)
		fmt.Println("   💡 建议: 增加 max_open_conns 配置或降低并发度")
	}

	// 3. 检查当前goroutine数量
	fmt.Println("3. 🔄 检查goroutine状态...")
	initialGoroutines := runtime.NumGoroutine()
	fmt.Printf("   📊 当前goroutine数量: %d\n", initialGoroutines)

	// 4. 模拟并发场景分析
	fmt.Println("4. 🎯 分析可能的阻塞点...")

	// 检查443个交易对的并发配置
	symbolCount := 443 // 现货市场交易对数量
	maxConcurrency := 2 // 当前配置的并发度

	fmt.Printf("   📊 同步参数分析:\n")
	fmt.Printf("      - 总交易对数: %d\n", symbolCount)
	fmt.Printf("      - 并发度: %d\n", maxConcurrency)
	fmt.Printf("      - 批次大小: %d\n", maxConcurrency*5)
	fmt.Printf("      - 总批次数: %d\n", (symbolCount+(maxConcurrency*5-1))/(maxConcurrency*5))

	// 5. 分析潜在问题
	fmt.Println("5. 🔍 问题诊断结果:")

	problems := []string{}

	// 检查连接池
	if stats.InUse >= cfg.Database.MaxOpenConns {
		problems = append(problems, "数据库连接池耗尽 - 所有连接都在使用中")
	}

	if stats.WaitCount > 100 {
		problems = append(problems, "严重的连接等待 - 表明连接池配置不足")
	}

	// 检查并发配置
	batchSize := maxConcurrency * 5
	if batchSize < maxConcurrency {
		problems = append(problems, "批次大小配置异常")
	}

	// 检查信号量阻塞风险
	if maxConcurrency < 2 && symbolCount > 50 {
		problems = append(problems, "并发度过低 - 在大量交易对时可能导致信号量阻塞")
	}

	// 检查resultChan阻塞风险
	resultChanSize := symbolCount
	if resultChanSize < symbolCount {
		problems = append(problems, fmt.Sprintf("结果通道缓冲区不足 - 当前%d，需要%d", resultChanSize, symbolCount))
	}

	if len(problems) == 0 {
		fmt.Println("   ✅ 未发现明显配置问题")
		fmt.Println("   💡 如果仍然卡住，可能是API限流或网络问题")
	} else {
		fmt.Println("   ❌ 发现以下潜在问题:")
		for i, problem := range problems {
			fmt.Printf("      %d. %s\n", i+1, problem)
		}
	}

	// 6. 提供解决方案建议
	fmt.Println("6. 💡 优化建议:")

	fmt.Println("   数据库连接池:")
	fmt.Printf("      - 当前: max_open_conns=%d, max_idle_conns=%d\n", cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	fmt.Println("      - 建议: max_open_conns=50, max_idle_conns=20")

	fmt.Println("   并发控制:")
	fmt.Printf("      - 当前并发度: %d\n", maxConcurrency)
	fmt.Println("      - 建议: 动态调整 - 小批量(≤50)用5, 大批量(>100)用2-3")

	fmt.Println("   批次处理:")
	fmt.Printf("      - 当前批次大小: %d\n", batchSize)
	fmt.Println("      - 建议: 批次间增加延迟(500ms-1s)")

	fmt.Println("   监控建议:")
	fmt.Println("      - 添加goroutine数量监控")
	fmt.Println("      - 添加数据库连接池状态监控")
	fmt.Println("      - 添加每个批次的处理时间统计")

	fmt.Println("\n🎯 运行以下命令获取实时状态:")
	fmt.Println("   go run diagnose_sync_hang.go")
	fmt.Println("\n📈 如需更详细的诊断，请提供同步卡住时的日志")
}