package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"analysis/internal/config"
)

// FundingRateResponse 资金费率API响应结构
type FundingRateResponse struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
	MarkPrice   string `json:"markPrice,omitempty"`
}

// PremiumIndexResponse 溢价指数API响应结构
type PremiumIndexResponse struct {
	Symbol               string `json:"symbol"`
	MarkPrice            string `json:"markPrice"`
	IndexPrice           string `json:"indexPrice"`
	EstimatedSettlePrice string `json:"estimatedSettlePrice"`
	LastFundingRate      string `json:"lastFundingRate"`
	InterestRate         string `json:"interestRate"`
	NextFundingTime      int64  `json:"nextFundingTime"`
	Time                 int64  `json:"time"`
}

func main() {
	fmt.Println("🔍 DASHUSDT 资金费率对比分析")
	fmt.Println("=========================================")

	// 加载配置文件并应用代理设置
	cfg := &config.Config{}
	config.MustLoad("config.yaml", cfg)
	config.ApplyProxy(cfg)

	fmt.Printf("✅ 已应用代理配置: enabled=%v\n", cfg.Proxy.Enable)
	if cfg.Proxy.Enable {
		fmt.Printf("   HTTP代理: %s\n", cfg.Proxy.HTTP)
		fmt.Printf("   HTTPS代理: %s\n", cfg.Proxy.HTTPS)
	}

	ctx := context.Background()
	symbol := "DASHUSDT"

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

	fmt.Println("\n" + "="*50)
	fmt.Println("1️⃣ 查询历史资金费率 (fundingRate API)")
	fmt.Println("=" * 50)

	// 查询历史资金费率
	fundingURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/fundingRate?symbol=%s&limit=3", symbol)
	fmt.Printf("📡 API URL: %s\n", fundingURL)

	req, err := http.NewRequestWithContext(ctx, "GET", fundingURL, nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
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

	fmt.Printf("📄 响应: %s\n\n", string(body))

	var fundingRates []FundingRateResponse
	if err := json.Unmarshal(body, &fundingRates); err != nil {
		fmt.Printf("❌ 解析JSON失败: %v\n", err)
		return
	}

	fmt.Println("📊 历史资金费率记录:")
	for i, rate := range fundingRates {
		fundingRate, _ := strconv.ParseFloat(rate.FundingRate, 64)
		fundingTime := time.Unix(rate.FundingTime/1000, 0)

		status := "✅ 已结算"
		if i == 0 {
			status = "🔥 最新结算"
		}

		fmt.Printf("   %s [%d]: %.8f (%6.3f%%) - %s\n",
			status, i+1, fundingRate, fundingRate*100,
			fundingTime.Format("01-02 15:04"))
	}

	fmt.Println("\n" + "="*50)
	fmt.Println("2️⃣ 查询预测资金费率 (premiumIndex API)")
	fmt.Println("=" * 50)

	// 查询预测资金费率
	premiumURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)
	fmt.Printf("📡 API URL: %s\n", premiumURL)

	req2, err := http.NewRequestWithContext(ctx, "GET", premiumURL, nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("📄 响应: %s\n\n", string(body2))

	var premiumIndex PremiumIndexResponse
	if err := json.Unmarshal(body2, &premiumIndex); err != nil {
		fmt.Printf("❌ 解析JSON失败: %v\n", err)
		return
	}

	lastFundingRate, _ := strconv.ParseFloat(premiumIndex.LastFundingRate, 64)
	markPrice, _ := strconv.ParseFloat(premiumIndex.MarkPrice, 64)
	indexPrice, _ := strconv.ParseFloat(premiumIndex.IndexPrice, 64)
	nextFundingTime := time.Unix(premiumIndex.NextFundingTime/1000, 0)

	fmt.Println("📊 预测资金费率信息:")
	fmt.Printf("   🔹 交易对: %s\n", premiumIndex.Symbol)
	fmt.Printf("   💰 上次结算费率: %.8f (%6.3f%%)\n", lastFundingRate, lastFundingRate*100)
	fmt.Printf("   📊 标记价格: %.4f USDT\n", markPrice)
	fmt.Printf("   📈 指数价格: %.4f USDT\n", indexPrice)
	fmt.Printf("   ⏰ 下次结算时间: %s\n", nextFundingTime.Format("01-02 15:04:05"))

	// 计算距离下次结算的时间
	timeUntilNext := time.Until(nextFundingTime)
	if timeUntilNext > 0 {
		hours := int(timeUntilNext.Hours())
		minutes := int(timeUntilNext.Minutes()) % 60
		fmt.Printf("   ⏳ 距离结算: %d小时%d分钟\n", hours, minutes)
	}

	fmt.Println("\n" + "="*50)
	fmt.Println("3️⃣ 对比分析")
	fmt.Println("=" * 50)

	if len(fundingRates) > 0 {
		latestHistorical, _ := strconv.ParseFloat(fundingRates[0].FundingRate, 64)
		fmt.Printf("🔸 网页显示费率: -0.09247%% (%.8f)\n", -0.0009247)
		fmt.Printf("🔸 API历史费率: %6.3f%% (%.8f)\n", latestHistorical*100, latestHistorical)
		fmt.Printf("🔸 API预测费率: %6.3f%% (%.8f)\n", lastFundingRate*100, lastFundingRate)

		fmt.Println("\n📝 分析结果:")
		fmt.Printf("   • 网页显示的是【预测资金费率】或【实时计算费率】\n")
		fmt.Printf("   • API返回的是【已结算的历史资金费率】\n")
		fmt.Printf("   • 预测费率会根据市场情况实时变化\n")
		fmt.Printf("   • 实际结算费率以8小时为周期结算\n")

		fmt.Println("\n💡 建议:")
		fmt.Printf("   • 网页显示更实时，适合查看当前市场预期\n")
		fmt.Printf("   • API历史数据适合策略回测和分析\n")
		fmt.Printf("   • 两者相差大可能是市场波动或计算方式不同\n")
	}
}
