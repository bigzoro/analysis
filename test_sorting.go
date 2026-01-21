package main

import (
	"fmt"
	"sort"
	"strconv"
)

// 模拟 Binance24hrTicker 结构
type Binance24hrTicker struct {
	Symbol              string
	PriceChangePercent  string
	Volume              string
}

func main() {
	// 模拟一些测试数据
	tickers := []Binance24hrTicker{
		{Symbol: "BTCUSDT", PriceChangePercent: "5.25", Volume: "12345.67"},
		{Symbol: "ETHUSDT", PriceChangePercent: "3.80", Volume: "23456.78"},
		{Symbol: "BIFIUSDT", PriceChangePercent: "25.50", Volume: "123.45"}, // 涨幅最高，但成交量最低
		{Symbol: "PEPEUSDT", PriceChangePercent: "2.10", Volume: "98765.43"}, // 涨幅最低，但成交量最高
		{Symbol: "ADAUSDT", PriceChangePercent: "8.90", Volume: "34567.89"},
	}

	fmt.Println("原始数据:")
	for i, ticker := range tickers {
		fmt.Printf("%d. %s: 涨幅=%.2f%%, 成交量=%s\n", i+1, ticker.Symbol, parseFloat(ticker.PriceChangePercent), ticker.Volume)
	}

	// 按涨幅降序排序（修改后的逻辑）
	sort.Slice(tickers, func(i, j int) bool {
		pctI := parseFloat(tickers[i].PriceChangePercent)
		pctJ := parseFloat(tickers[j].PriceChangePercent)
		return pctI > pctJ
	})

	fmt.Println("\n按涨幅排序后（新逻辑）:")
	for i, ticker := range tickers {
		fmt.Printf("第%d名: %s (涨幅=%.2f%%, 成交量=%s)\n",
			i+1, ticker.Symbol, parseFloat(ticker.PriceChangePercent), ticker.Volume)
	}

	// 对比：按成交量排序（旧逻辑）
	sort.Slice(tickers, func(i, j int) bool {
		volI := parseFloat(tickers[i].Volume)
		volJ := parseFloat(tickers[j].Volume)
		return volI > volJ
	})

	fmt.Println("\n按成交量排序（旧逻辑，仅供对比）:")
	for i, ticker := range tickers {
		fmt.Printf("第%d名: %s (涨幅=%.2f%%, 成交量=%s)\n",
			i+1, ticker.Symbol, parseFloat(ticker.PriceChangePercent), ticker.Volume)
	}
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}



