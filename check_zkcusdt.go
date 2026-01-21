package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ExchangeInfoResp struct {
	Symbols []struct {
		Symbol     string `json:"symbol"`
		Status     string `json:"status"`
		BaseAsset  string `json:"baseAsset"`
		QuoteAsset string `json:"quoteAsset"`
	} `json:"symbols"`
}

func checkSymbolInExchange(url, exchangeName, symbol string) {
	fmt.Printf("\n=== 检查 %s 在 %s 中的状态 ===\n", symbol, exchangeName)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	var exchangeInfo ExchangeInfoResp
	if err := json.Unmarshal(body, &exchangeInfo); err != nil {
		fmt.Printf("❌ 解析JSON失败: %v\n", err)
		return
	}

	found := false
	for _, s := range exchangeInfo.Symbols {
		if s.Symbol == symbol {
			found = true
			fmt.Printf("✅ 找到交易对: %s\n", symbol)
			fmt.Printf("   状态: %s\n", s.Status)
			fmt.Printf("   基础资产: %s\n", s.BaseAsset)
			fmt.Printf("   计价资产: %s\n", s.QuoteAsset)

			if s.Status == "TRADING" {
				fmt.Printf("✅ 该交易对支持交易\n")
			} else {
				fmt.Printf("⚠️ 该交易对状态为: %s (不支持交易)\n", s.Status)
			}
			break
		}
	}

	if !found {
		fmt.Printf("❌ 未找到交易对: %s\n", symbol)
		fmt.Printf("💡 这意味着该交易对在 %s 中不存在\n", exchangeName)
	}
}

func main() {
	symbol := "ZKCUSDT"

	// 检查现货
	checkSymbolInExchange("https://api.binance.com/api/v3/exchangeInfo", "币安现货", symbol)

	// 检查期货
	checkSymbolInExchange("https://fapi.binance.com/fapi/v1/exchangeInfo", "币安期货", symbol)

	// 检查币本位期货
	checkSymbolInExchange("https://dapi.binance.com/dapi/v1/exchangeInfo", "币安币本位期货", symbol)
}



