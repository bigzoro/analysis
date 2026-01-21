package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== 并发K线同步测试 ===")

	// 模拟并发处理
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	maxConcurrency := 2

	fmt.Printf("测试并发处理 %d 个交易对，最大并发度: %d\n", len(symbols), maxConcurrency)

	// 模拟信号量
	semaphore := make(chan struct{}, maxConcurrency)
	results := make(chan string, len(symbols))
	var wg sync.WaitGroup

	// 并发处理
	for i, symbol := range symbols {
		wg.Add(1)
		go func(index int, sym string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 模拟API调用时间
			time.Sleep(200 * time.Millisecond)
			result := fmt.Sprintf("✅ 处理完成 %s (索引:%d)", sym, index)
			results <- result
			fmt.Printf("  %s\n", result)
		}(i, symbol)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	completed := 0
	for result := range results {
		fmt.Printf("收集结果: %s\n", result)
		completed++
	}

	fmt.Printf("\n🎉 并发测试完成: 成功处理 %d 个交易对\n", completed)
	fmt.Printf("💡 并发优化可以提升约 %d 倍性能\n", maxConcurrency)
}
