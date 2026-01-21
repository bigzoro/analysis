package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

func main() {
	// 模拟订单172的数据
	entryPrice := 4.1844
	closePrice := 4.12835
	quantity := 2.0

	// 计算利润
	var pnl float64
	// 空头交易：利润 = (开仓价 - 平仓价) * 数量
	pnl = (entryPrice - closePrice) * quantity

	fmt.Printf("开仓价格: %.6f\n", entryPrice)
	fmt.Printf("平仓价格: %.6f\n", closePrice)
	fmt.Printf("数量: %.1f\n", quantity)
	fmt.Printf("利润计算: (%.6f - %.6f) * %.1f = %.6f\n", entryPrice, closePrice, quantity, pnl)

	// 模拟字符串转换
	entryPriceStr := "4.1844000"
	closePriceStr := "4.1283500"
	quantityStr := "2"

	closePriceParsed, _ := strconv.ParseFloat(closePriceStr, 64)
	quantityParsed, _ := strconv.ParseFloat(quantityStr, 64)

	pnl2 := (entryPrice - closePriceParsed) * quantityParsed
	fmt.Printf("字符串转换后利润: %.6f\n", pnl2)

	// 模拟数据库中的数据
	fmt.Println("\n=== 数据库数据模拟 ===")
	fmt.Printf("订单172 (开仓): side=SELL, avg_price=4.1844000, executed_quantity=2\n")
	fmt.Printf("订单173 (平仓): side=BUY, avg_price=4.1283500, executed_quantity=2\n")
	fmt.Printf("预期利润: %.6f USDT\n", pnl)
}
