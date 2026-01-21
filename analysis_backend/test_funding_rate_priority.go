package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"analysis/internal/config"
)

// FundingRateResponse 历史资金费率API响应结构
type FundingRateResponse struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
}

// PremiumIndexResponse 预测资金费率API响应结构
type PremiumIndexResponse struct {
	Symbol          string `json:"symbol"`
	MarkPrice       string `json:"markPrice"`
	IndexPrice      string `json:"indexPrice"`
	LastFundingRate string `json:"lastFundingRate"`
	Time            int64  `json:"time"`
}

func main() {
	fmt.Println("🔍 资金费率优先级测试 - 实时 vs 历史")
	fmt.Println("=========================================")

	// 加载配置文件并应用代理设置
	cfg := &config.Config{}
	config.MustLoad("config.yaml", cfg)
	config.ApplyProxy(cfg)

	fmt.Printf("✅ 已应用代理配置: enabled=%v\n", cfg.Proxy.Enable)

	// 创建带代理的HTTP客户端
	var proxyURL string
	if cfg != nil && cfg.Proxy.Enable {
		if cfg.Proxy.HTTPS != "" {
			proxyURL = cfg.Proxy.HTTPS
		} else if cfg.Proxy.HTTP != "" {
			proxyURL = cfg.Proxy.HTTP
		}
	}

	var transport *http.Transport
	if proxyURL != "" {
		fmt.Printf("🔗 使用代理: %s\n", proxyURL)
		proxyParsedURL, err := url.Parse(proxyURL)
		if err == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyParsedURL),
			}
		} else {
			fmt.Printf("❌ 代理URL解析失败: %v\n", err)
			transport = &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			}
		}
	} else {
		fmt.Println("🔗 不使用代理")
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	ctx := context.Background()
	symbol := "DASHUSDT"

	// 提前声明所有变量避免goto问题
	var realTimeRate float64
	var realTimeTime time.Time
	var resp1 *http.Response
	var premiumIndex PremiumIndexResponse
	var success bool
	var body1 []byte
	var err error

	// 1. 尝试获取实时预测资金费率
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("1️⃣ 优先尝试获取实时预测资金费率 (Premium Index API)")
	fmt.Println(strings.Repeat("=", 60))

	premiumURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)
	fmt.Printf("📡 实时API URL: %s\n", premiumURL)

	req1, err := http.NewRequestWithContext(ctx, "GET", premiumURL, nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		goto FallbackToHistory
	}
	req1.Header.Set("User-Agent", "Mozilla/5.0")

	resp1, err = client.Do(req1)
	if err != nil {
		fmt.Printf("❌ 实时API请求失败: %v\n", err)
		fmt.Println("🔄 切换到历史API...")
		goto FallbackToHistory
	}

	body1, err = io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if err != nil {
		fmt.Printf("❌ 读取实时API响应失败: %v\n", err)
		fmt.Println("🔄 切换到历史API...")
		goto FallbackToHistory
	}

	fmt.Printf("📄 实时API响应: %s\n\n", string(body1))

	if err := json.Unmarshal(body1, &premiumIndex); err != nil {
		fmt.Printf("❌ 解析实时API响应失败: %v\n", err)
		fmt.Println("🔄 切换到历史API...")
		goto FallbackToHistory
	}

	// 成功获取实时数据
	realTimeRate, _ = strconv.ParseFloat(premiumIndex.LastFundingRate, 64)
	realTimeTime = time.Unix(premiumIndex.Time/1000, 0)
	success = true

	fmt.Println("✅ 实时预测资金费率获取成功:")
	fmt.Printf("   🔹 交易对: %s\n", premiumIndex.Symbol)
	fmt.Printf("   💰 实时费率: %.8f (%6.3f%%)\n", realTimeRate, realTimeRate*100)
	fmt.Printf("   📊 标记价格: %s USDT\n", premiumIndex.MarkPrice)
	fmt.Printf("   📈 指数价格: %s USDT\n", premiumIndex.IndexPrice)
	fmt.Printf("   ⏰ 数据时间: %s\n", realTimeTime.Format("01-02 15:04:05"))

FallbackToHistory:
	// 2. 获取历史资金费率进行对比
	historyURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/fundingRate?symbol=%s&limit=3", symbol)
	fmt.Printf("📡 历史API URL: %s\n", historyURL)

	req2, err := http.NewRequestWithContext(ctx, "GET", historyURL, nil)
	if err != nil {
		fmt.Printf("❌ 创建历史API请求失败: %v\n", err)
		return
	}
	req2.Header.Set("User-Agent", "Mozilla/5.0")

	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("❌ 历史API请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		fmt.Printf("❌ 读取历史API响应失败: %v\n", err)
		return
	}

	fmt.Printf("📄 历史API响应: %s\n\n", string(body2))

	var fundingRates []FundingRateResponse
	if err := json.Unmarshal(body2, &fundingRates); err != nil {
		fmt.Printf("❌ 解析历史API响应失败: %v\n", err)
		return
	}

	fmt.Println("📊 历史资金费率记录:")
	for i, rate := range fundingRates {
		historicalRate, _ := strconv.ParseFloat(rate.FundingRate, 64)
		historicalTime := time.Unix(rate.FundingTime/1000, 0)

		status := "✅ 已结算"
		if i == 0 {
			status = "🔥 最新结算"
		}

		fmt.Printf("   %s [%d]: %.8f (%6.3f%%) - %s\n",
			status, i+1, historicalRate, historicalRate*100,
			historicalTime.Format("01-02 15:04:05"))
	}

	// 如果成功获取了实时数据，进行对比
	if success {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("3️⃣ 数据对比分析")
		fmt.Println(strings.Repeat("=", 60))

		if len(fundingRates) > 0 {
			latestHistorical, _ := strconv.ParseFloat(fundingRates[0].FundingRate, 64)

			fmt.Printf("🔸 实时预测费率: %6.3f%% (%.8f)\n", realTimeRate*100, realTimeRate)
			fmt.Printf("🔸 最新历史费率: %6.3f%% (%.8f)\n", latestHistorical*100, latestHistorical)

			diff := realTimeRate - latestHistorical
			fmt.Printf("🔸 差异: %6.3f%% (%.8f)\n", diff*100, diff)

			fmt.Println("\n📝 分析结果:")
			if diff > 0.001 {
				fmt.Printf("   📈 实时费率更高，可能预示空头力量增强\n")
			} else if diff < -0.001 {
				fmt.Printf("   📉 实时费率更低，可能预示多头力量增强\n")
			} else {
				fmt.Printf("   ⚖️ 实时费率与历史费率基本一致\n")
			}

			fmt.Println("\n💡 优先级策略:")
			fmt.Printf("   ✅ 实时数据: 更及时，反映当前市场预期\n")
			fmt.Printf("   🔄 历史数据: 可靠的fallback，确保数据可用性\n")
			fmt.Printf("   📊 结合使用: 实时数据用于决策，历史数据用于分析\n")
		}
	}

	fmt.Println("\n🎯 结论:")
	fmt.Printf("   • 实时预测费率更适合交易决策\n")
	fmt.Printf("   • 历史费率适合回测和趋势分析\n")
	fmt.Printf("   • 双重fallback确保数据高可用性\n")
}
