package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func main() {
	// 设置代理（参考config.yaml中的设置）
	proxyURL, err := url.Parse("http://127.0.0.1:10808")
	if err != nil {
		fmt.Printf("❌ 代理URL解析失败: %v\n", err)
		return
	}

	// 创建使用代理的HTTP客户端
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	client := &http.Client{
		Transport: transport,
	}

	fmt.Printf("✅ 已设置代理: %s\n", proxyURL.String())

	// 测试几个有问题的交易对
	testSymbols := []string{"VETUSDT", "DOGSUSDT", "ROSEUSDT", "CRVUSDT"}

	fmt.Println("🔍 测试币安期货API响应格式...")

	for _, symbol := range testSymbols {
		fmt.Printf("\n--- 测试 %s ---\n", symbol)

		// 构造币本位期货URL
		coinMURL := fmt.Sprintf("https://dapi.binance.com/dapi/v1/ticker/24hr?symbol=%sUSD_PERP", symbol)
		fmt.Printf("币本位期货URL: %s\n", coinMURL)

		testAPIResponse(client, coinMURL, "币本位期货")

		// 构造USDT期货URL
		usdtURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/24hr?symbol=%s", symbol)
		fmt.Printf("USDT期货URL: %s\n", usdtURL)

		testAPIResponse(client, usdtURL, "USDT期货")
	}
}

func testAPIResponse(client *http.Client, url, apiType string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败 (%s): %v\n", apiType, err)
		return
	}

	// 设置User-Agent，避免被识别为爬虫
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败 (%s): %v\n", apiType, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败 (%s): %v\n", apiType, err)
		return
	}

	fmt.Printf("📄 响应状态 (%s): %s\n", apiType, resp.Status)
	fmt.Printf("📄 原始响应 (%s): %s\n", apiType, string(body))

	// 尝试解析为单个对象
	var singleObj struct {
		Symbol             string `json:"symbol"`
		PriceChangePercent string `json:"priceChangePercent"`
		Code               int    `json:"code,omitempty"`
		Msg                string `json:"msg,omitempty"`
	}

	if err := json.Unmarshal(body, &singleObj); err != nil {
		fmt.Printf("❌ 解析为单个对象失败 (%s): %v\n", apiType, err)

		// 尝试解析为数组
		var arrayResp []struct {
			Symbol             string `json:"symbol"`
			PriceChangePercent string `json:"priceChangePercent"`
			Code               int    `json:"code,omitempty"`
			Msg                string `json:"msg,omitempty"`
		}

		if err := json.Unmarshal(body, &arrayResp); err != nil {
			fmt.Printf("❌ 解析为数组也失败 (%s): %v\n", apiType, err)
		} else {
			fmt.Printf("✅ 解析为数组成功 (%s)，包含 %d 个元素\n", apiType, len(arrayResp))
			if len(arrayResp) > 0 {
				fmt.Printf("   第一个元素: symbol=%s, change=%s\n",
					arrayResp[0].Symbol, arrayResp[0].PriceChangePercent)
			}
		}
	} else {
		fmt.Printf("✅ 解析为单个对象成功 (%s): symbol=%s, change=%s\n",
			apiType, singleObj.Symbol, singleObj.PriceChangePercent)
	}
}
