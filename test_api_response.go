package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("🔍 检查API响应中的技术指标数据")

	// 先测试健康检查接口
	fmt.Println("\n🏥 测试健康检查接口:")
	resp, err := http.Get("http://localhost:8010/healthz")
	if err != nil {
		fmt.Printf("❌ 健康检查请求失败: %v\n", err)
		fmt.Println("💡 请确保后端服务正在运行")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("健康检查响应: %s\n", string(body))

	// 等待一下
	time.Sleep(1 * time.Second)

	// 调用市场分析API
	fmt.Println("\n📊 测试市场分析API:")
	resp2, err := http.Get("http://localhost:8010/api/market-analysis/comprehensive")
	if err != nil {
		fmt.Printf("❌ API请求失败: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	if resp2.StatusCode != 200 {
		fmt.Printf("❌ API返回错误状态码: %d\n", resp2.StatusCode)
		fmt.Printf("响应内容: %s\n", string(body2))
		fmt.Println("💡 这个API可能需要认证")
		return
	}

	fmt.Println("✅ API请求成功")

	// 解析响应
	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			MarketAnalysis struct {
				Volatility  float64 `json:"volatility"`
				Trend       string  `json:"trend"`
				Oscillation float64 `json:"oscillation"`
			} `json:"market_analysis"`
			TechnicalIndicators struct {
				BTCVolatility float64 `json:"btc_volatility"`
				AvgRSI        float64 `json:"avg_rsi"`
				StrongSymbols int     `json:"strong_symbols"`
				WeakSymbols   int     `json:"weak_symbols"`
			} `json:"technical_indicators"`
			StrategyRecommendations []interface{} `json:"strategy_recommendations"`
		} `json:"data"`
		Meta struct {
			Cached             bool    `json:"cached"`
			ProcessingTimeMs   float64 `json:"processing_time_ms"`
			CacheTTL          string  `json:"cache_ttl"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Printf("❌ 解析JSON失败: %v\n", err)
		fmt.Printf("原始响应: %s\n", string(body))
		return
	}

	if !apiResp.Success {
		fmt.Println("❌ API返回失败状态")
		return
	}

	fmt.Println("\n📊 API响应数据分析:")

	// 检查市场分析数据
	ma := apiResp.Data.MarketAnalysis
	fmt.Printf("市场分析:\n")
	fmt.Printf("  波动率: %.2f%%\n", ma.Volatility)
	fmt.Printf("  趋势: %s\n", ma.Trend)
	fmt.Printf("  震荡度: %.2f%%\n", ma.Oscillation)

	// 检查技术指标数据
	ti := apiResp.Data.TechnicalIndicators
	fmt.Printf("\n技术指标:\n")
	fmt.Printf("  BTC波动率: %.2f%%\n", ti.BTCVolatility)
	fmt.Printf("  平均RSI: %.2f\n", ti.AvgRSI)
	fmt.Printf("  强势币种: %d\n", ti.StrongSymbols)
	fmt.Printf("  弱势币种: %d\n", ti.WeakSymbols)

	// 检查元数据
	fmt.Printf("\n元数据:\n")
	fmt.Printf("  缓存: %v\n", apiResp.Meta.Cached)
	fmt.Printf("  处理时间: %.0fms\n", apiResp.Meta.ProcessingTimeMs)
	if apiResp.Meta.CacheTTL != "" {
		fmt.Printf("  缓存TTL: %s\n", apiResp.Meta.CacheTTL)
	}

	// 诊断问题
	fmt.Println("\n🔍 问题诊断:")

	if ti.BTCVolatility == 0 && ti.AvgRSI == 0 && ti.StrongSymbols == 0 && ti.WeakSymbols == 0 {
		fmt.Println("❌ 所有技术指标都为0 - 后端技术指标计算失败")
	} else {
		fmt.Println("✅ 技术指标数据正常")
	}

	// 检查具体值
	if ti.BTCVolatility > 0 {
		fmt.Println("✅ BTC波动率正常")
	} else {
		fmt.Println("❌ BTC波动率为0")
	}

	if ti.AvgRSI > 0 {
		fmt.Println("✅ 平均RSI正常")
	} else {
		fmt.Println("❌ 平均RSI为0")
	}

	if ti.StrongSymbols >= 0 && ti.WeakSymbols >= 0 {
		fmt.Printf("✅ 强弱币种数据存在 (强势:%d, 弱势:%d)\n", ti.StrongSymbols, ti.WeakSymbols)
	} else {
		fmt.Println("❌ 强弱币种数据异常")
	}

	// 检查数据一致性
	if ma.Volatility > 0 && ti.BTCVolatility > 0 {
		fmt.Println("✅ 市场分析和技术指标数据都正常")
	} else if ma.Volatility == 0 && ti.BTCVolatility == 0 {
		fmt.Println("❌ 市场分析和技术指标都为0 - 可能是数据源问题")
	} else {
		fmt.Println("⚠️ 数据不一致 - 部分数据正常，部分为0")
	}
}