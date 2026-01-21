package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"analysis/internal/config"
)

// PremiumIndexResponse 预测资金费率API响应结构
type PremiumIndexResponse struct {
	Symbol          string `json:"symbol"`
	MarkPrice       string `json:"markPrice"`
	IndexPrice      string `json:"indexPrice"`
	LastFundingRate string `json:"lastFundingRate"`
	Time            int64  `json:"time"`
}

// ExchangeInfoResponse 合约信息API响应结构
type ExchangeInfoResponse struct {
	Symbols []SymbolInfo `json:"symbols"`
}

type SymbolInfo struct {
	Symbol string `json:"symbol"`
	Status string `json:"status"`
}

// FundingRateAnalysis 资金费率分析结果
type FundingRateAnalysis struct {
	Symbol           string
	FundingRate      float64
	Price            float64
	RatePercentage   float64
	SuitableForShort bool // 是否适合做空
}

func main() {
	fmt.Println("🎯 合约做空策略 - 资金费率优化分析")
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

	fmt.Println("\n" + "="*70)
	fmt.Println("1️⃣ 获取活跃合约列表")
	fmt.Println("=" * 70)

	// 获取所有活跃的期货合约
	exchangeURL := "https://fapi.binance.com/fapi/v1/exchangeInfo"
	req, err := http.NewRequestWithContext(ctx, "GET", exchangeURL, nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 获取合约信息失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	var exchangeInfo ExchangeInfoResponse
	if err := json.Unmarshal(body, &exchangeInfo); err != nil {
		fmt.Printf("❌ 解析合约信息失败: %v\n", err)
		return
	}

	// 过滤出活跃的USDT合约
	var activeSymbols []string
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Status == "TRADING" && len(symbol.Symbol) > 4 {
			if symbol.Symbol[len(symbol.Symbol)-4:] == "USDT" {
				activeSymbols = append(activeSymbols, symbol.Symbol)
			}
		}
	}

	fmt.Printf("✅ 找到 %d 个活跃的USDT合约\n", len(activeSymbols))

	fmt.Println("\n" + "="*70)
	fmt.Println("2️⃣ 获取资金费率数据")
	fmt.Println("=" * 70)

	// 获取前50个合约的资金费率（避免API限制）
	analysisLimit := 50
	if len(activeSymbols) > analysisLimit {
		activeSymbols = activeSymbols[:analysisLimit]
	}

	var analyses []FundingRateAnalysis

	fmt.Printf("🔄 分析 %d 个合约的资金费率...\n", len(activeSymbols))

	for i, symbol := range activeSymbols {
		if i > 0 && i%10 == 0 {
			fmt.Printf("   已处理 %d/%d 个合约...\n", i, len(activeSymbols))
		}

		// 获取资金费率
		premiumURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)
		req2, err := http.NewRequestWithContext(ctx, "GET", premiumURL, nil)
		if err != nil {
			continue
		}
		req2.Header.Set("User-Agent", "Mozilla/5.0")

		resp2, err := client.Do(req2)
		if err != nil {
			continue
		}

		body2, err := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		if err != nil {
			continue
		}

		var premium PremiumIndexResponse
		if err := json.Unmarshal(body2, &premium); err != nil {
			continue
		}

		fundingRate, err := strconv.ParseFloat(premium.LastFundingRate, 64)
		if err != nil {
			continue
		}

		markPrice, err := strconv.ParseFloat(premium.MarkPrice, 64)
		if err != nil {
			continue
		}

		analysis := FundingRateAnalysis{
			Symbol:           symbol,
			FundingRate:      fundingRate,
			Price:            markPrice,
			RatePercentage:   fundingRate * 100,
			SuitableForShort: fundingRate > 0, // 正资金费率适合做空
		}

		analyses = append(analyses, analysis)
	}

	fmt.Printf("✅ 成功获取 %d 个合约的资金费率数据\n", len(analyses))

	fmt.Println("\n" + "="*70)
	fmt.Println("3️⃣ 资金费率分布分析")
	fmt.Println("=" * 70)

	// 按资金费率排序
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].FundingRate > analyses[j].FundingRate
	})

	// 统计分布
	var positiveRates, negativeRates []FundingRateAnalysis
	for _, analysis := range analyses {
		if analysis.FundingRate > 0 {
			positiveRates = append(positiveRates, analysis)
		} else {
			negativeRates = append(negativeRates, analysis)
		}
	}

	fmt.Printf("📊 资金费率分布统计:\n")
	fmt.Printf("   🔴 正资金费率合约: %d 个 (%.1f%%)\n", len(positiveRates), float64(len(positiveRates))/float64(len(analyses))*100)
	fmt.Printf("   🔵 负资金费率合约: %d 个 (%.1f%%)\n", len(negativeRates), float64(len(negativeRates))/float64(len(analyses))*100)

	fmt.Println("\n" + "="*70)
	fmt.Println("4️⃣ 最适合做空的合约 (正资金费率)")
	fmt.Println("=" * 70)

	fmt.Println("🏆 最佳做空合约 (资金费率从高到低):")
	fmt.Printf("%-12s %-12s %-8s %-s\n", "合约", "资金费率", "价格", "状态")
	fmt.Println(strings.Repeat("-", 50))

	for i, analysis := range positiveRates {
		if i >= 20 { // 只显示前20个
			break
		}
		status := "✅ 推荐"
		fmt.Printf("%-12s %-12.4f %-8.2f %s\n",
			analysis.Symbol,
			analysis.RatePercentage,
			analysis.Price,
			status)
	}

	fmt.Println("\n" + "="*70)
	fmt.Println("5️⃣ 策略优化建议")
	fmt.Println("=" * 70)

	fmt.Println("🎯 当前策略配置:")
	fmt.Println("   • 最低资金费率: -0.5000% (允许负费率)")
	fmt.Println("   • 资金费率过滤: 关闭")
	fmt.Println("   • 问题: 会选择不利于做空的合约")

	fmt.Println("\n💡 优化建议:")

	if len(positiveRates) > 0 {
		avgPositiveRate := 0.0
		for _, analysis := range positiveRates {
			avgPositiveRate += analysis.RatePercentage
		}
		avgPositiveRate /= float64(len(positiveRates))

		minRecommendedRate := avgPositiveRate * 0.5 // 建议最低费率为平均值的50%

		fmt.Printf("   ✅ 启用资金费率过滤\n")
		fmt.Printf("   ✅ 设置最低资金费率: %.3f%% (平均正费率的50%%)\n", minRecommendedRate)
		fmt.Printf("   ✅ 建议范围: %.3f%% ~ %.3f%%\n", minRecommendedRate, avgPositiveRate*1.5)

		fmt.Println("\n   📈 预期效果:")
		fmt.Printf("      • 避免选择负费率合约 (当前%d个)\n", len(negativeRates))
		fmt.Printf("      • 优先选择正费率合约 (当前%d个)\n", len(positiveRates))
		fmt.Printf("      • 降低持仓成本，提高盈利概率\n")
	}

	fmt.Println("\n" + "="*70)
	fmt.Println("6️⃣ DASHUSDT 具体分析")
	fmt.Println("=" * 70)

	// 找到DASHUSDT的数据
	var dashAnalysis *FundingRateAnalysis
	for _, analysis := range analyses {
		if analysis.Symbol == "DASHUSDT" {
			dashAnalysis = &analysis
			break
		}
	}

	if dashAnalysis != nil {
		fmt.Printf("🔍 DASHUSDT 当前状态:\n")
		fmt.Printf("   💰 资金费率: %.4f%% (%s)\n",
			dashAnalysis.RatePercentage,
			func() string {
				if dashAnalysis.SuitableForShort {
					return "✅ 适合做空"
				}
				return "❌ 不适合做空"
			}())

		if dashAnalysis.SuitableForShort {
			fmt.Println("   📊 DASHUSDT 当前资金费率为正，适合做空策略")
		} else {
			fmt.Printf("   ⚠️  DASHUSDT 当前资金费率为负，不利于做空\n")
			fmt.Printf("      建议等待费率转正，或选择其他正费率合约\n")
		}
	}

	fmt.Println("\n" + "="*70)
	fmt.Println("7️⃣ 配置更新建议")
	fmt.Println("=" * 70)

	fmt.Println("🔧 建议的策略配置更新:")
	fmt.Println(`
# 在策略配置中更新：
funding_rate_filter_enabled: true          # 启用资金费率过滤
min_funding_rate: 0.01                     # 最低资金费率 0.01%
                                           # 或使用上面计算的推荐值

# 这样可以确保只选择正资金费率的合约进行做空
`)

	fmt.Println("🎯 总结:")
	fmt.Printf("   • 正资金费率合约获得做空优势\n")
	fmt.Printf("   • 负资金费率合约增加做空成本\n")
	fmt.Printf("   • 建议只选择正费率合约进行做空\n")
	fmt.Printf("   • 当前市场有 %d 个正费率合约可供选择\n", len(positiveRates))
}
