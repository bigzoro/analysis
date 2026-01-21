package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查XNYUSDT订单的止盈止损百分比")

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	// 查询XNYUSDT的Bracket订单
	fmt.Println("\n📊 查询XNYUSDT的Bracket订单")
	var orders []pdb.ScheduledOrder
	err = gdb.GormDB().
		Where("symbol = ? AND bracket_enabled = ? AND status = ?",
			"XNYUSDT", true, "filled").
		Order("created_at DESC").
		Limit(5).
		Find(&orders).Error

	if err != nil {
		log.Printf("查询订单失败: %v", err)
		return
	}

	fmt.Printf("找到%d个XNYUSDT Bracket订单:\n", len(orders))

	for _, order := range orders {
		fmt.Printf("\n🎯 订单 #%d (%s)\n", order.ID, order.ClientOrderId)
		fmt.Printf("  方向: %s\n", order.Side)
		fmt.Printf("  杠杆: %.1f\n", order.Leverage)
		fmt.Printf("  成交数量: %s\n", order.AdjustedQuantity)
		fmt.Printf("  成交均价: %s\n", order.AvgPrice)

		fmt.Printf("  用户设置止盈: %.5f%%\n", order.TPPercent)
		fmt.Printf("  实际止盈百分比: %.5f%%\n", order.ActualTPPercent)
		fmt.Printf("  止盈价格: %s\n", order.TPPrice)

		fmt.Printf("  用户设置止损: %.5f%%\n", order.SLPercent)
		fmt.Printf("  实际止损百分比: %.5f%%\n", order.ActualSLPercent)
		fmt.Printf("  止损价格: %s\n", order.SLPrice)

		// 检查是否有BracketLink
		var bracket pdb.BracketLink
		err = gdb.GormDB().Where("entry_client_id = ?", order.ClientOrderId).First(&bracket).Error
		if err == nil {
			fmt.Printf("  Bracket状态: %s\n", bracket.Status)
		}

		// 验证百分比计算
		fmt.Printf("\n🔬 验证百分比计算:\n")
		verifyPercentCalculation(order)
		fmt.Println("  ─────────────────────────────────")
	}
}

func verifyPercentCalculation(order pdb.ScheduledOrder) {
	if order.AdjustedQuantity == "" || order.AvgPrice == "" {
		fmt.Printf("  缺少成交数据，跳过验证\n")
		return
	}

	// 解析数据
	entryPrice := parseFloat(order.AvgPrice)
	tpPrice := parseFloat(order.TPPrice)
	slPrice := parseFloat(order.SLPrice)
	isLong := order.Side == "BUY"

	fmt.Printf("  入场价格: %.8f\n", entryPrice)
	fmt.Printf("  止盈价格: %.8f\n", tpPrice)
	fmt.Printf("  止损价格: %.8f\n", slPrice)
	fmt.Printf("  是否多头: %v\n", isLong)

	// 计算实际百分比
	var calculatedTPPercent, calculatedSLPercent float64

	if isLong {
		// 多头仓位
		if tpPrice > entryPrice {
			calculatedTPPercent = ((tpPrice - entryPrice) / entryPrice) * 100
		}
		if slPrice < entryPrice {
			calculatedSLPercent = ((entryPrice - slPrice) / entryPrice) * 100
		}
	} else {
		// 空头仓位
		if tpPrice < entryPrice {
			calculatedTPPercent = ((entryPrice - tpPrice) / entryPrice) * 100
		}
		if slPrice > entryPrice {
			calculatedSLPercent = ((slPrice - entryPrice) / entryPrice) * 100
		}
	}

	fmt.Printf("  计算得止盈百分比: %.5f%%\n", calculatedTPPercent)
	fmt.Printf("  数据库实际止盈: %.5f%%\n", order.ActualTPPercent)
	fmt.Printf("  计算得止损百分比: %.5f%%\n", calculatedSLPercent)
	fmt.Printf("  数据库实际止损: %.5f%%\n", order.ActualSLPercent)

	// 检查匹配度
	tpDiff := abs(calculatedTPPercent - order.ActualTPPercent)
	slDiff := abs(calculatedSLPercent - order.ActualSLPercent)

	fmt.Printf("  止盈差异: %.5f%%\n", tpDiff)
	fmt.Printf("  止损差异: %.5f%%\n", slDiff)

	if tpDiff < 0.01 {
		fmt.Printf("  ✅ 止盈百分比计算正确\n")
	} else {
		fmt.Printf("  ❌ 止盈百分比计算不正确\n")
	}

	if slDiff < 0.01 {
		fmt.Printf("  ✅ 止损百分比计算正确\n")
	} else {
		fmt.Printf("  ❌ 止损百分比计算不正确\n")
	}
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}