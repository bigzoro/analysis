package main

import (
	"context"
	"fmt"
	"log"

	"analysis/internal/netutil"
)

type BinanceSymbolInfo struct {
	Symbol  string                   `json:"symbol"`
	Status  string                   `json:"status"`
	Filters []map[string]interface{} `json:"filters"`
}

type BinanceExchangeInfo struct {
	Symbols []BinanceSymbolInfo `json:"symbols"`
}

func main() {
	// 直接调用Binance API获取RIVERUSDT的信息
	ctx := context.Background()
	url := "https://fapi.binance.com/fapi/v1/exchangeInfo?symbol=RIVERUSDT"
	var exchangeInfo BinanceExchangeInfo

	fmt.Printf("请求API: %s\n", url)
	if err := netutil.GetJSON(ctx, url, &exchangeInfo); err != nil {
		log.Fatalf("API调用失败: %v", err)
	}
	fmt.Printf("API返回%d个交易对\n", len(exchangeInfo.Symbols))

	if len(exchangeInfo.Symbols) == 0 {
		log.Fatalf("未找到RIVERUSDT交易对信息")
	}

	symbolInfo := exchangeInfo.Symbols[0]

	fmt.Printf("RIVERUSDT 交易对信息:\n")
	fmt.Printf("状态: %s\n", symbolInfo.Status)

	fmt.Printf("过滤器 (共%d个):\n", len(symbolInfo.Filters))
	for i, filter := range symbolInfo.Filters {
		if filterType, ok := filter["filterType"].(string); ok {
			fmt.Printf("  [%d] %s: %v\n", i, filterType, filter)
		} else {
			fmt.Printf("  [%d] 未知过滤器: %v\n", i, filter)
		}
	}

	// 特别关注LOT_SIZE过滤器
	for _, filter := range symbolInfo.Filters {
		if filterType, ok := filter["filterType"].(string); ok && filterType == "LOT_SIZE" {
			fmt.Printf("\nLOT_SIZE过滤器详情:\n")
			for k, v := range filter {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	}
}