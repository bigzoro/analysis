package main

import (
	"context"
	"fmt"
	"time"
	"analysis/internal/netutil"
)

type BinanceExchangeInfo struct {
	Symbols []struct {
		Symbol  string                   `json:"symbol"`
		Status  string                   `json:"status"`
		Filters []map[string]interface{} `json:"filters"`
	} `json:"symbols"`
}

func main() {
	var info BinanceExchangeInfo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := netutil.GetJSON(ctx, "https://fapi.binance.com/fapi/v1/exchangeInfo?symbol=ALCHUSDT", &info)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(info.Symbols) == 0 {
		fmt.Println("No symbols found")
		return
	}

	fmt.Printf("Symbol: %s\n", info.Symbols[0].Symbol)
	fmt.Printf("Status: %s\n", info.Symbols[0].Status)

	for _, f := range info.Symbols[0].Filters {
		fmt.Printf("Filter: %v\n", f)
	}
}
