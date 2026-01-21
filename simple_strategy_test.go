package main

import (
	"fmt"
	"log"

	"analysis/internal/server"
)

func main() {
	fmt.Println("开始测试策略注册修复...")

	// 创建策略扫描器注册表
	registry := server.NewStrategyScannerRegistry()

	// 创建一个模拟的服务器（只包含必要的方法）
	mockServer := &mockServer{}

	// 尝试注册扫描器
	err := registry.RegisterScanner(mockServer)
	if err != nil {
		log.Printf("❌ 策略注册失败: %v", err)
		return
	}

	fmt.Println("✅ 策略注册成功！")

	// 检查已注册的扫描器
	scanners := registry.GetRegisteredScannerTypes()
	fmt.Printf("已注册的扫描器: %v\n", scanners)
}

// mockServer 模拟服务器，只实现必要的方法
type mockServer struct{}

func (m *mockServer) DB() interface{} { return nil }
