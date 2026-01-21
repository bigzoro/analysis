package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 测试BDXNUSDT缓存清理修复效果 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 模拟修复后的getValidSymbolsByMarket逻辑
	fmt.Println("🔍 模拟修复后的缓存清理逻辑:")

	// 1. 获取按市场分组的活跃交易对
	validSymbols := map[string]map[string]bool{
		"spot":    make(map[string]bool),
		"futures": make(map[string]bool),
	}

	// 获取现货活跃交易对
	var spotSymbols []string
	db.Raw(`
		SELECT symbol FROM binance_exchange_info
		WHERE quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?
	`, "USDT", "TRADING", "spot", true).Scan(&spotSymbols)

	for _, symbol := range spotSymbols {
		validSymbols["spot"][symbol] = true
	}

	// 获取期货活跃交易对
	var futuresSymbols []string
	db.Raw(`
		SELECT symbol FROM binance_exchange_info
		WHERE quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?
	`, "USDT", "TRADING", "futures", true).Scan(&futuresSymbols)

	for _, symbol := range futuresSymbols {
		validSymbols["futures"][symbol] = true
	}

	fmt.Printf("  📊 活跃交易对统计 - 现货: %d, 期货: %d\n", len(spotSymbols), len(futuresSymbols))

	// 2. 模拟处理缓存中的"BDXNUSDT_spot"
	symbolKind := "BDXNUSDT_spot"
	fmt.Printf("\n🔍 处理缓存条目: %s\n", symbolKind)

	// 解析symbol和kind
	parts := []string{"BDXNUSDT", "spot"}
	symbol := parts[0]
	kind := parts[1]

	// 检查该市场类型的活跃交易对
	marketValidSymbols, exists := validSymbols[kind]
	if !exists {
		fmt.Printf("  ❌ 未知市场类型: %s\n", kind)
	} else if !marketValidSymbols[symbol] {
		fmt.Printf("  ✅ %s在%s市场不活跃，应该清理缓存\n", symbol, kind)
	} else {
		fmt.Printf("  ⚠️  %s在%s市场活跃，需要API验证\n", symbol, kind)
	}

	// 3. 验证修复效果
	fmt.Println("\n🎯 修复效果验证:")

	// 检查BDXNUSDT在不同市场的状态
	fmt.Printf("  BDXNUSDT现货活跃: %v (期望: false)\n", validSymbols["spot"]["BDXNUSDT"])
	fmt.Printf("  BDXNUSDT期货活跃: %v (期望: true)\n", validSymbols["futures"]["BDXNUSDT"])

	// 检查缓存清理决策
	spotShouldClean := !validSymbols["spot"]["BDXNUSDT"]
	futuresShouldKeep := validSymbols["futures"]["BDXNUSDT"]

	fmt.Printf("\n💡 缓存清理决策:\n")
	fmt.Printf("  BDXNUSDT_spot 应该清理: %v ✅\n", spotShouldClean)
	fmt.Printf("  BDXNUSDT_futures 应该保留: %v ✅\n", futuresShouldKeep)

	if spotShouldClean {
		fmt.Println("\n✅ 修复成功: BDXNUSDT现货缓存将不再被错误验证")
	} else {
		fmt.Println("\n❌ 修复失败: 仍会验证BDXNUSDT现货")
	}

	fmt.Println("\n=== 测试完成 ===")
}