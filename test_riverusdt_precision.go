package main

import (
	"fmt"
	"log"

	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	// 创建Binance期货客户端
	client := bf.NewClient("", "", "")

	// 获取exchange info
	info, err := client.GetExchangeInfo()
	if err != nil {
		log.Fatalf("获取exchange info失败: %v", err)
	}

	// 查找RIVERUSDT
	for _, symbol := range info.Symbols {
		if symbol.Symbol == "RIVERUSDT" {
			fmt.Printf("RIVERUSDT 交易对信息:\n")
			fmt.Printf("状态: %s\n", symbol.Status)
			fmt.Printf("合约类型: %s\n", symbol.ContractType)

			fmt.Printf("过滤器:\n")
			for i, filter := range symbol.Filters {
				fmt.Printf("  [%d] %v\n", i, filter)
			}
			break
		}
	}
}