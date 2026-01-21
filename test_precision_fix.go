package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type OrderScheduler struct {
	db *gorm.DB
}

// 模拟 prepareOrderPrecision 函数的逻辑
func (s *OrderScheduler) prepareOrderPrecision(symbol, quantity, price, orderType string) error {
	// 模拟精度调整（这里只是测试逻辑）
	var adjustedQuantity, adjustedPrice string

	// 模拟调整数量和价格
	adjustedQuantity = quantity // 假设数量已经符合精度
	if orderType == "LIMIT" {
		adjustedPrice = price
	} else {
		adjustedPrice = ""
	}

	// 验证精度信息是否有效
	hasValidPrecision := s.hasValidExchangeInfo(symbol)
	if !hasValidPrecision {
		return fmt.Errorf("无法获取 %s 的精度信息，精度调整失败", symbol)
	}

	// 检查调整是否合理
	var precisionAdjusted bool
	if orderType == "LIMIT" {
		precisionAdjusted = (adjustedQuantity != "" && adjustedPrice != "")
	} else {
		precisionAdjusted = (adjustedQuantity != "")
	}

	if !precisionAdjusted {
		return fmt.Errorf("无法获取 %s 的精度信息，精度调整失败", symbol)
	}

	fmt.Printf("✅ %s 精度调整成功: 数量 %s, 价格 %s\n", symbol, adjustedQuantity, adjustedPrice)
	return nil
}

// hasValidExchangeInfo 检查数据库中是否有有效的交易所信息
func (s *OrderScheduler) hasValidExchangeInfo(symbol string) bool {
	// 从数据库获取交易对信息
	exchangeInfo, err := pdb.GetExchangeInfo(s.db, symbol)
	if err != nil {
		log.Printf("检查 %s 交易所信息失败: %v", symbol, err)
		return false
	}

	// 检查过滤器信息是否存在且不为空
	if exchangeInfo.Filters == "" || len(exchangeInfo.Filters) < 10 {
		log.Printf("%s 的过滤器信息为空或过短", symbol)
		return false
	}

	fmt.Printf("✅ %s 找到有效的过滤器信息 (长度: %d)\n", symbol, len(exchangeInfo.Filters))
	return true
}

func main() {
	fmt.Println("=== 测试精度调整修复 ===")

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("analysis_backend/analysis.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	scheduler := &OrderScheduler{db: db}

	// 测试 DASHUSDT 市价单（根据日志信息）
	fmt.Println("\n🔍 测试 DASHUSDT 市价单...")
	err = scheduler.prepareOrderPrecision("DASHUSDT", "0.857", "", "MARKET")
	if err != nil {
		fmt.Printf("❌ DASHUSDT 测试失败: %v\n", err)
	} else {
		fmt.Printf("✅ DASHUSDT 测试成功\n")
	}

	// 测试其他交易对
	testSymbols := []string{"BTCUSDT", "ETHUSDT", "FILUSDT"}
	for _, symbol := range testSymbols {
		fmt.Printf("\n🔍 测试 %s...\n", symbol)
		err = scheduler.prepareOrderPrecision(symbol, "0.001", "50000", "LIMIT")
		if err != nil {
			fmt.Printf("❌ %s 测试失败: %v\n", symbol, err)
		}
	}
}
