package main

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
)

func main() {
	client := resty.New()

	fmt.Println("=== 测试回测API ===")

	// 测试获取回测记录列表
	fmt.Println("\n1. 测试获取回测记录列表...")
	resp, err := client.R().
		SetQueryParams(map[string]string{
			"page":  "1",
			"limit": "10",
		}).
		Get("http://localhost:8080/api/backtest/async/records")

	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}

	fmt.Printf("响应状态码: %d\n", resp.StatusCode())
	fmt.Printf("响应内容: %s\n", resp.Body())

	// 测试创建策略回测
	fmt.Println("\n2. 测试创建策略回测...")
	testData := map[string]interface{}{
		"strategy_id": 1,
		"symbol":      "BTCUSDT",
		"start_date":  "2024-01-01T00:00:00.000Z",
		"end_date":    "2024-12-01T00:00:00.000Z",
	}

	resp2, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(testData).
		Post("http://localhost:8080/api/backtest/strategy")

	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}

	fmt.Printf("响应状态码: %d\n", resp2.StatusCode())
	fmt.Printf("响应内容: %s\n", resp2.Body())

	fmt.Println("\n✅ API测试完成")
}
