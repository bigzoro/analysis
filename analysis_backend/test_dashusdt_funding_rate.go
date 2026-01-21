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
	Symbol               string `json:"symbol"`
	FundingRate          string `json:"fundingRate"`
	FundingTime          int64  `json:"fundingTime"`
	MarkPrice            string `json:"markPrice,omitempty"`
	IndexPrice           string `json:"indexPrice,omitempty"`
	EstimatedSettlePrice string `json:"estimatedSettlePrice,omitempty"`
}

func main() {
	fmt.Println("🔍 查询 DASHUSDT 资金费率")
	fmt.Println("========================================")

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

	// 构建API URL
	apiURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/fundingRate?symbol=%s&limit=1", symbol)
	fmt.Printf("📡 API请求URL: %s\n\n", apiURL)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

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

	// 发送请求
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ API响应错误: HTTP %d\n", resp.StatusCode)
		return
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("📄 原始响应: %s\n\n", string(body))

	// 解析JSON响应
	var rates []FundingRateResponse
	if err := json.Unmarshal(body, &rates); err != nil {
		fmt.Printf("❌ 解析JSON失败: %v\n", err)
		return
	}

	if len(rates) == 0 {
		fmt.Println("❌ 未找到资金费率数据")
		return
	}

	rate := rates[0]

	// 解析数值
	fundingRate, err := strconv.ParseFloat(rate.FundingRate, 64)
	if err != nil {
		fmt.Printf("❌ 解析资金费率失败: %v\n", err)
		return
	}

	markPrice := 0.0
	if rate.MarkPrice != "" {
		markPrice, _ = strconv.ParseFloat(rate.MarkPrice, 64)
	}

	indexPrice := 0.0
	if rate.IndexPrice != "" {
		indexPrice, _ = strconv.ParseFloat(rate.IndexPrice, 64)
	}

	estimatedSettlePrice := 0.0
	if rate.EstimatedSettlePrice != "" {
		estimatedSettlePrice, _ = strconv.ParseFloat(rate.EstimatedSettlePrice, 64)
	}

	// 转换为时间
	fundingTime := time.Unix(rate.FundingTime/1000, 0)

	fmt.Println("✅ DASHUSDT 资金费率查询结果:")
	fmt.Println("=======================================")
	fmt.Printf("🔹 交易对: %s\n", rate.Symbol)
	fmt.Printf("💰 资金费率: %.8f (%.4f%%)\n", fundingRate, fundingRate*100)
	fmt.Printf("⏰ 资金费率时间: %s\n", fundingTime.Format("2006-01-02 15:04:05"))

	if markPrice > 0 {
		fmt.Printf("📊 标记价格: %.8f USDT\n", markPrice)
	}
	if indexPrice > 0 {
		fmt.Printf("📈 指数价格: %.8f USDT\n", indexPrice)
	}
	if estimatedSettlePrice > 0 {
		fmt.Printf("🎯 预估结算价格: %.8f USDT\n", estimatedSettlePrice)
	}

	fmt.Println("\n📝 资金费率含义:")
	if fundingRate > 0 {
		fmt.Printf("   💸 正数资金费率: 持有多头需要支付 %.4f%% 的资金费率\n", fundingRate*100)
	} else if fundingRate < 0 {
		fmt.Printf("   💰 负数资金费率: 持有空头获得 %.4f%% 的资金费率补贴\n", -fundingRate*100)
	} else {
		fmt.Println("   ⚖️ 资金费率为0: 多空双方平衡")
	}

	fmt.Println("\n⏱️ 结算频率: 每8小时结算一次")
	fmt.Printf("📅 下次结算时间: %s\n", fundingTime.Add(8*time.Hour).Format("2006-01-02 15:04:05"))
}
