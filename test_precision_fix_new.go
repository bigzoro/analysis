package main

import (
	"fmt"
	"log"
	"math"
	"strings"

	pdb "analysis/internal/db"

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

// 模拟 calculateSmartTargetNotional 函数
func calculateSmartTargetNotional(price float64, symbol string, leverage int) float64 {
	// 根据杠杆倍数设置最小保证金目标
	minMarginTarget := 10.0 // 目标保证金至少10 USDT

	// 计算需要的名义价值：保证金 × 杠杆
	baseTarget := minMarginTarget * float64(leverage)

	// 确保名义价值不低于币安最低要求
	baseTarget = math.Max(baseTarget, 5.0)

	// 根据价格区间调整目标 - 更细粒度的分类
	var target float64
	if price < 0.0001 { // 极极低价币种（<0.01美分）
		target = math.Max(baseTarget, 50.0) // 大幅提高目标
	} else if price < 0.001 { // 极低价币种（<0.1美分）
		target = math.Max(baseTarget, 30.0) // 提高到30 USDT目标
	} else if price < 0.01 { // 低价币种（<1美分）
		target = math.Max(baseTarget, 20.0) // 稍微提高目标
	} else if price < 0.1 { // 中低价币种（<10美分）
		target = math.Max(baseTarget, 15.0) // 小幅提高目标
	} else if price > 100 { // 高价币种（>100 USDT）
		target = math.Max(baseTarget, 5.0) // 保持最低要求
	} else {
		target = baseTarget // 中等价格币种使用杠杆计算的目标
	}

	// 特殊币种调整
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	if strings.Contains(strings.ToLower(baseSymbol), "shib") || strings.Contains(strings.ToLower(baseSymbol), "doge") {
		target = math.Max(target, baseTarget) // meme币使用杠杆计算的目标
	}

	// 对于特定已知低价币种，进一步调整
	if strings.Contains(strings.ToLower(baseSymbol), "arc") {
		target = math.Max(target, baseTarget+5.0) // ARC特殊处理
	}

	fmt.Printf("%s 价格=%.8f, 杠杆=%dx, 目标名义价值=%.1f USDT (保证金≥%.1f USDT)\n",
		symbol, price, leverage, target, target/float64(leverage))

	return target
}

func main() {
	fmt.Println("=== 测试保证金计算调整 ===")

	// 测试不同价格区间的币种
	testCases := []struct {
		symbol   string
		price    float64
		leverage int
	}{
		{"DASHUSDT", 0.0005, 3},  // 低价币种，3倍杠杆
		{"ARCUSDT", 0.0001, 5},   // 极低价币种，5倍杠杆
		{"SHIBUSDT", 0.00001, 1}, // 超低价币种，1倍杠杆
		{"BTCUSDT", 50000, 10},   // 高价币种，10倍杠杆
		{"ETHUSDT", 3000, 5},     // 中价币种，5倍杠杆
	}

	for _, tc := range testCases {
		fmt.Printf("\n🔍 测试 %s:\n", tc.symbol)
		targetNotional := calculateSmartTargetNotional(tc.price, tc.symbol, tc.leverage)
		margin := targetNotional / float64(tc.leverage)
		fmt.Printf("   ✅ 最终保证金: %.2f USDT\n", margin)
	}
}
