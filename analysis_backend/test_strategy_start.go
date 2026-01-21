package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type StartStrategyRequest struct {
	StrategyID     uint    `json:"strategy_id"`
	RunInterval    int     `json:"run_interval"`
	MaxRuns        int     `json:"max_runs"`
	AutoStop       bool    `json:"auto_stop"`
	CreateOrders   bool    `json:"create_orders"`
	ExecutionDelay int     `json:"execution_delay"`
	PerOrderAmount float64 `json:"per_order_amount"`
}

func main() {
	fmt.Println("=== 测试策略启动API ===")

	// 测试数据
	testRequest := StartStrategyRequest{
		StrategyID:     33,
		RunInterval:    60,
		MaxRuns:        0,
		AutoStop:       false,
		CreateOrders:   true,
		ExecutionDelay: 60,
		PerOrderAmount: 100.0, // 测试金额
	}

	// 序列化JSON
	jsonData, err := json.Marshal(testRequest)
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		return
	}

	fmt.Printf("📤 发送请求数据:\n%s\n", string(jsonData))

	// 这里只是演示，实际需要启动后端服务才能测试
	fmt.Println("\n💡 要完全测试需要:")
	fmt.Println("1. 启动后端服务")
	fmt.Println("2. 设置Authorization header")
	fmt.Println("3. 发送POST请求到 /api/strategies/start")

	// 模拟发送请求（如果服务运行的话）
	testURL := "http://localhost:8080/api/strategies/start"

	fmt.Printf("\n🔗 测试URL: %s\n", testURL)
	fmt.Printf("📊 预期结果: PerOrderAmount应该被保存为100.0\n")

	// 如果服务在运行，尝试发送请求
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token") // 需要实际token

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("⚠️ 发送请求失败 (服务可能未启动): %v\n", err)
		fmt.Println("\n🔧 调试建议:")
		fmt.Println("1. 检查后端服务是否在localhost:8080运行")
		fmt.Println("2. 检查Authorization token是否有效")
		fmt.Println("3. 查看后端日志确认API是否收到正确参数")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("📥 响应状态: %s\n", resp.Status)
	fmt.Printf("📥 响应内容:\n%s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ API调用成功")
	} else {
		fmt.Printf("❌ API调用失败 (状态码: %d)\n", resp.StatusCode)
	}
}