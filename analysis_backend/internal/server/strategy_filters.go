package server

import (
	"math"
	"strings"
)

// ============================================================================
// 策略过滤器
// ============================================================================

// FilterStableCoins 过滤稳定币交易对
func FilterStableCoins(symbols []string) []string {
	// 常见的稳定币后缀
	stableCoinSuffixes := []string{"USDT", "BUSD", "USDC", "DAI", "FRAX", "TUSD", "USDP", "UST"}
	var filtered []string

	for _, symbol := range symbols {
		isStableCoin := false

		// 检查是否以稳定币结尾（排除基础货币是稳定币的情况）
		for _, suffix := range stableCoinSuffixes {
			if strings.HasSuffix(symbol, suffix) {
				// 进一步检查：如果是A-USDT这样的格式，需要排除
				// 但如果是BTCUSDT这样的，需要保留
				baseCurrency := strings.TrimSuffix(symbol, suffix)
				if len(baseCurrency) > 0 && baseCurrency != "USD" && baseCurrency != "USD1" {
					// 这是一个正常的交易对，不是稳定币本身
					continue
				}
				isStableCoin = true
				break
			}
		}

		// 特殊处理：一些特定的稳定币交易对
		if symbol == "BFUSDUSDT" || symbol == "BUSDUSDT" {
			isStableCoin = true
		}

		if !isStableCoin {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// FilterByVolatility 按波动率过滤资产
func FilterByVolatility(symbols []string, minVolatilityPercent float64) []string {
	var filtered []string

	for _, symbol := range symbols {
		// 计算24小时波动率
		volatility := calculate24hVolatility(symbol)
		if volatility >= minVolatilityPercent/100.0 { // 转换为小数
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// FilterByMarketCap 按市值过滤资产
func FilterByMarketCap(symbols []string, minMarketCapUSD float64) []string {
	var filtered []string

	for _, symbol := range symbols {
		// 获取市值数据
		marketCap := getMarketCapUSD(symbol)
		if marketCap >= minMarketCapUSD {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// ValidateVolatilityForMA 验证均线策略的波动率要求
func ValidateVolatilityForMA(symbol string, prices []float64, minVolatilityPercent float64) bool {
	if len(prices) < 2 {
		return false
	}

	// 计算价格波动率
	var changes []float64
	for i := 1; i < len(prices); i++ {
		change := math.Abs(prices[i]-prices[i-1]) / prices[i-1] * 100
		changes = append(changes, change)
	}

	if len(changes) == 0 {
		return false
	}

	// 计算平均波动率
	totalChange := 0.0
	for _, change := range changes {
		totalChange += change
	}
	avgVolatility := totalChange / float64(len(changes))

	return avgVolatility >= minVolatilityPercent
}

// ValidateTrendStrength 验证趋势强度
func ValidateTrendStrength(shortMA, longMA []float64, minStrength float64) bool {
	if len(shortMA) == 0 || len(longMA) == 0 {
		return false
	}

	latestShort := shortMA[len(shortMA)-1]
	latestLong := longMA[len(longMA)-1]

	// 计算趋势强度 (短期均线相对长期均线的偏离程度)
	if latestLong == 0 {
		return false
	}

	trendStrength := math.Abs(latestShort-latestLong) / latestLong

	return trendStrength >= minStrength
}

// AssessSignalQuality 评估信号质量
func AssessSignalQuality(shortMA, longMA []float64, prices []float64) float64 {
	if len(shortMA) < 2 || len(longMA) < 2 || len(prices) < 2 {
		return 0.0
	}

	// 1. 趋势一致性检查 (短期和长期趋势方向是否一致)
	shortTrend := shortMA[len(shortMA)-1] > shortMA[len(shortMA)-2]
	longTrend := longMA[len(longMA)-1] > longMA[len(longMA)-2]
	trendConsistency := 0.0
	if shortTrend == longTrend {
		trendConsistency = 1.0
	}

	// 2. 价格确认 (当前价格是否支持信号方向)
	latestPrice := prices[len(prices)-1]
	latestShort := shortMA[len(shortMA)-1]
	latestLong := longMA[len(longMA)-1]

	priceConfirmation := 0.0
	if latestShort > latestLong {
		// 金叉信号 - 价格应该在短期均线上方
		if latestPrice > latestShort {
			priceConfirmation = 1.0
		} else if latestPrice > latestLong {
			priceConfirmation = 0.5
		}
	} else {
		// 死叉信号 - 价格应该在短期均线下方
		if latestPrice < latestShort {
			priceConfirmation = 1.0
		} else if latestPrice < latestLong {
			priceConfirmation = 0.5
		}
	}

	// 3. 信号强度 (均线偏离程度)
	signalStrength := math.Min(math.Abs(latestShort-latestLong)/latestLong, 0.1) / 0.1

	// 综合评分 (0.0-1.0)
	quality := (trendConsistency*0.4 + priceConfirmation*0.4 + signalStrength*0.2)

	return math.Min(quality, 1.0)
}

// ============================================================================
// 辅助函数
// ============================================================================

// calculate24hVolatility 计算24小时波动率 (简化实现)
func calculate24hVolatility(symbol string) float64 {
	// 这里应该从数据库查询真实的波动率数据
	// 简化实现，返回模拟值

	// 特殊处理稳定币
	if strings.Contains(symbol, "USDT") && len(symbol) > 4 {
		base := strings.TrimSuffix(symbol, "USDT")
		if base == "BUSD" || base == "USDC" || base == "BFUSD" {
			return 0.001 // 稳定币极低波动率
		}
	}

	// 主流币种较高波动率
	if symbol == "BTCUSDT" || symbol == "ETHUSDT" {
		return 0.05 // 5%
	}

	// 其他币种中等波动率
	return 0.02 // 2%
}

// getMarketCapUSD 获取市值数据 (简化实现)
func getMarketCapUSD(symbol string) float64 {
	// 这里应该从CoinCap或其他数据源获取真实的市值数据
	// 简化实现，返回模拟值

	marketCapMap := map[string]float64{
		"BTCUSDT":  1e12, // 1万亿美元
		"ETHUSDT":  3e11, // 3000亿美元
		"BNBUSDT":  5e10, // 500亿美元
		"ADAUSDT":  2e10, // 200亿美元
		"SOLUSDT":  1e10, // 100亿美元
		"DOTUSDT":  8e9,  // 80亿美元
		"DOGEUSDT": 3e9,  // 30亿美元
		"AVAXUSDT": 2e9,  // 20亿美元
		"LTCUSDT":  1e9,  // 10亿美元
		"LINKUSDT": 8e8,  // 8亿美元
	}

	if cap, exists := marketCapMap[symbol]; exists {
		return cap
	}

	// 默认中等市值
	return 1e8 // 1亿美元
}

// calculateAvgVolatility 计算平均波动率
func calculateAvgVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	// 计算价格波动率
	var changes []float64
	for i := 1; i < len(prices); i++ {
		change := math.Abs(prices[i]-prices[i-1]) / prices[i-1] * 100
		changes = append(changes, change)
	}

	if len(changes) == 0 {
		return 0.0
	}

	// 计算平均波动率
	totalChange := 0.0
	for _, change := range changes {
		totalChange += change
	}
	avgVolatility := totalChange / float64(len(changes))

	return avgVolatility
}

// ============================================================================
// 配置结构
// ============================================================================

// MAValidationThresholds 均线策略验证阈值
type MAValidationThresholds struct {
	MinVolatility    float64 // 最小波动率 (%)
	MinTrendStrength float64 // 最小趋势强度
	MinSignalQuality float64 // 最小信号质量
	StrictMode       bool    // 是否为严格模式
}

// GetMAValidationThresholds 根据信号模式获取验证阈值
func GetMAValidationThresholds(signalMode string) MAValidationThresholds {
	switch signalMode {
	case "QUALITY_FIRST":
		// 质量优先：严格验证，高品质信号
		return MAValidationThresholds{
			MinVolatility:    0.08,  // 波动率 ≥ 0.08%
			MinTrendStrength: 0.002, // 趋势强度 ≥ 0.2%
			MinSignalQuality: 0.7,   // 信号质量 ≥ 70%
			StrictMode:       true,
		}
	case "QUANTITY_FIRST":
		// 数量优先：宽松验证，更多信号 (针对当前市场环境优化)
		return MAValidationThresholds{
			MinVolatility:    0.015,  // 波动率 ≥ 1.50% (从3%降低，适应当前市场)
			MinTrendStrength: 0.0002, // 趋势强度 ≥ 0.02% (进一步放宽)
			MinSignalQuality: 0.25,   // 信号质量 ≥ 25% (从40%降低，适应震荡市)
			StrictMode:       false,
		}
	default:
		// 默认平衡模式
		return MAValidationThresholds{
			MinVolatility:    0.05,  // 波动率 ≥ 0.05%
			MinTrendStrength: 0.001, // 趋势强度 ≥ 0.1%
			MinSignalQuality: 0.5,   // 信号质量 ≥ 50%
			StrictMode:       false,
		}
	}
}

// MAStrategyConfig 均线策略配置
type MAStrategyConfig struct {
	// 候选过滤
	ExcludeStableCoins   bool    `yaml:"exclude_stable_coins"`
	MinVolatilityPercent float64 `yaml:"min_volatility_percent"`
	MinMarketCapUSD      float64 `yaml:"min_market_cap_usd"`

	// 信号过滤
	MinTrendStrength          float64 `yaml:"min_trend_strength"`
	MinSignalQuality          float64 `yaml:"min_signal_quality"`
	RequireVolumeConfirmation bool    `yaml:"require_volume_confirmation"`

	// 风险控制
	MaxPositionSizePercent float64 `yaml:"max_position_size_percent"`
	EnableStopLoss         bool    `yaml:"enable_stop_loss"`
	StopLossPercent        float64 `yaml:"stop_loss_percent"`
}
