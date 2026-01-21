package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ExchangeInfoResp struct {
	Symbols []struct {
		Symbol     string `json:"symbol"`
		Status     string `json:"status"`
		BaseAsset  string `json:"baseAsset"`
		QuoteAsset string `json:"quoteAsset"`
	} `json:"symbols"`
}

func main() {
	symbol := "ZKCUSDT"
	fmt.Printf("检查交易对: %s\n", symbol)

	// 调用币安期货API
	resp, err := http.Get("https://fapi.binance.com/fapi/v1/exchangeInfo")
	if err != nil {
		fmt.Printf("❌ API调用失败: %v\n", err)
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

	fmt.Printf("✅ 成功获取交易所信息，交易对数量: %d\n", len(exchangeInfo.Symbols))

	// 查找ZKCUSDT
	symbol = strings.ToUpper(symbol)
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
		fmt.Printf("📋 前10个交易对示例:\n")
		for i, s := range exchangeInfo.Symbols {
			if i >= 10 {
				break
			}
			fmt.Printf("   %s (状态: %s)\n", s.Symbol, s.Status)
		}
	}
}



