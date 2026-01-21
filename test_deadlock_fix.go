package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔬 开始死锁解决方案验证测试")

	// 加载配置
	cfg, err := config.LoadConfig("analysis_backend/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     cfg.Database.Automigrate,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	gdb := db.GormDB()

	// 测试参数
	numSymbols := 50    // 模拟50个交易对
	numKlinesPerSymbol := 20 // 每个交易对20条K线
	numWorkers := 10    // 10个并发工作者
	testDuration := 60 * time.Second // 测试持续时间

	fmt.Printf("📊 测试参数: %d 交易对 × %d K线 = %d 总操作, %d 并发工作者\n",
		numSymbols, numKlinesPerSymbol, numSymbols*numKlinesPerSymbol, numWorkers)

	// 准备测试数据
	symbols := generateTestSymbols(numSymbols)
	fmt.Printf("📋 生成测试交易对: %v\n", symbols[:min(10, len(symbols))]) // 只显示前10个

	// 统计变量
	var stats struct {
		mu             sync.Mutex
		totalAttempts  int64
		totalSuccess   int64
		totalDeadlocks int64
		totalErrors    int64
		startTime      time.Time
	}

	stats.startTime = time.Now()

	// 启动统计报告goroutine
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats.mu.Lock()
				elapsed := time.Since(stats.startTime)
				successRate := float64(stats.totalSuccess) / float64(stats.totalAttempts) * 100
				deadlockRate := float64(stats.totalDeadlocks) / float64(stats.totalAttempts) * 100
				stats.mu.Unlock()

				fmt.Printf("📈 [%v] 总尝试:%d 成功:%d(%.1f%%) 死锁:%d(%.1f%%) 错误:%d\n",
					elapsed.Round(time.Second),
					stats.totalAttempts,
					stats.totalSuccess,
					successRate,
					stats.totalDeadlocks,
					deadlockRate,
					stats.totalErrors)
			}
		}
	}()

	// 启动并发测试工作者
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// 随机选择一个交易对
					symbol := symbols[rand.Intn(len(symbols))]

					// 生成随机K线数据
					klines := generateRandomKlines(symbol, numKlinesPerSymbol)

					// 尝试保存K线数据
					stats.mu.Lock()
					stats.totalAttempts++
					stats.mu.Unlock()

					err := pdb.SaveMarketKlines(gdb, klines)

					stats.mu.Lock()
					if err != nil {
						stats.totalErrors++

						// 检查是否是死锁错误
						errStr := err.Error()
						if containsString(errStr, "deadlock") || containsString(errStr, "1213") || containsString(errStr, "40001") {
							stats.totalDeadlocks++
							fmt.Printf("🔴 工作者%d 检测到死锁: %s - %v\n", workerID, symbol, err)
						} else {
							fmt.Printf("⚠️ 工作者%d 其他错误: %s - %v\n", workerID, symbol, err)
						}
					} else {
						stats.totalSuccess++
						if stats.totalSuccess%100 == 0 {
							fmt.Printf("✅ 工作者%d 成功保存: %s (%d 条K线)\n", workerID, symbol, len(klines))
						}
					}
					stats.mu.Unlock()

					// 短暂延迟，避免过于激进
					time.Sleep(time.Duration(10+rand.Intn(50)) * time.Millisecond)
				}
			}
		}(i)
	}

	// 等待测试完成
	wg.Wait()

	// 输出最终统计
	stats.mu.Lock()
	finalElapsed := time.Since(stats.startTime)
	finalSuccessRate := float64(stats.totalSuccess) / float64(stats.totalAttempts) * 100
	finalDeadlockRate := float64(stats.totalDeadlocks) / float64(stats.totalAttempts) * 100
	stats.mu.Unlock()

	fmt.Println("\n🎯 测试完成!")
	fmt.Printf("⏱️ 总耗时: %v\n", finalElapsed)
	fmt.Printf("📊 总尝试次数: %d\n", stats.totalAttempts)
	fmt.Printf("✅ 成功次数: %d (%.2f%%)\n", stats.totalSuccess, finalSuccessRate)
	fmt.Printf("🔴 死锁次数: %d (%.2f%%)\n", stats.totalDeadlocks, finalDeadlockRate)
	fmt.Printf("⚠️ 其他错误: %d\n", stats.totalErrors)

	// 评估结果
	if finalDeadlockRate < 1.0 { // 死锁率低于1%
		fmt.Println("🎉 死锁解决方案有效! 死锁率控制在可接受范围内")
	} else if finalDeadlockRate < 5.0 { // 死锁率低于5%
		fmt.Println("⚠️ 死锁解决方案基本有效，但仍有优化空间")
	} else {
		fmt.Println("❌ 死锁解决方案需要进一步优化")
	}
}

// generateTestSymbols 生成测试用的交易对列表
func generateTestSymbols(count int) []string {
	symbols := make([]string, count)
	for i := 0; i < count; i++ {
		symbols[i] = fmt.Sprintf("TEST%dUSDT", i+1)
	}
	return symbols
}

// generateRandomKlines 为指定交易对生成随机K线数据
func generateRandomKlines(symbol string, count int) []pdb.MarketKline {
	klines := make([]pdb.MarketKline, count)

	baseTime := time.Now().Add(-24 * time.Hour) // 从24小时前开始

	for i := 0; i < count; i++ {
		openTime := baseTime.Add(time.Duration(i) * time.Minute)

		// 生成随机价格数据
		basePrice := 1.0 + rand.Float64()*100.0 // 1-101之间的随机价格
		open := basePrice + rand.Float64()*2.0 - 1.0
		close := open + (rand.Float64()*4.0 - 2.0)
		high := math.Max(open, close) + rand.Float64()*2.0
		low := math.Min(open, close) - rand.Float64()*2.0
		volume := rand.Float64() * 10000

		klines[i] = pdb.MarketKline{
			Symbol:     symbol,
			Kind:       "spot",
			Interval:   "1m",
			OpenTime:   openTime,
			OpenPrice:  fmt.Sprintf("%.8f", open),
			HighPrice:  fmt.Sprintf("%.8f", high),
			LowPrice:   fmt.Sprintf("%.8f", low),
			ClosePrice: fmt.Sprintf("%.8f", close),
			Volume:     fmt.Sprintf("%.8f", volume),
		}
	}

	return klines
}

// containsString 检查字符串是否包含子串（大小写不敏感）
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}