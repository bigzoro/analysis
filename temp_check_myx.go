package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ExchangeInfo struct {
	Symbols []Symbol `json:"symbols"`
}

type Symbol struct {
	Symbol string `json:"symbol"`
	Status string `json:"status"`
}

func main() {
	// 检查现货
	fmt.Println("=== 检查现货API ===")
	checkAPI("https://api.binance.com/api/v3/exchangeInfo", "MYXUSDT")

	// 检查期货
	fmt.Println("\n=== 检查期货API ===")
	checkAPI("https://fapi.binance.com/fapi/v1/exchangeInfo", "MYXUSDT")
}

func checkAPI(url, symbol string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	var info ExchangeInfo
	if err := json.Unmarshal(body, &info); err != nil {
		log.Printf("解析JSON失败: %v", err)
		return
	}

	found := false
	for _, s := range info.Symbols {
		if s.Symbol == symbol {
			fmt.Printf("✅ 找到 %s，状态: %s\n", symbol, s.Status)
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ 未找到 %s\n", symbol)
	}
}