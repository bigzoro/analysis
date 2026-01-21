package statistical

import (
	"analysis/internal/server/strategy/arbitrage"
	"context"
	"fmt"
	"math"
	"time"
)

// Scanner 统计套利扫描器实现
type Scanner struct {
	// TODO: 注入依赖，如历史数据提供者、协整分析工具等
}

// NewScanner 创建统计套利扫描器
func NewScanner() arbitrage.StatisticalScanner {
	return &Scanner{}
}

// FindStatArbOpportunities 查找统计套利机会
func (s *Scanner) FindStatArbOpportunities(ctx context.Context, symbols []string, config *arbitrage.ArbitrageConfig) ([]arbitrage.ArbitrageOpportunity, error) {
	var opportunities []arbitrage.ArbitrageOpportunity

	if len(symbols) < 2 {
		return opportunities, fmt.Errorf("统计套利至少需要2个交易对")
	}

	// 检查所有可能的交易对组合
	for i := 0; i < len(symbols); i++ {
		for j := i + 1; j < len(symbols); j++ {
			symbolA := symbols[i]
			symbolB := symbols[j]

			// 获取历史价格数据
			pricesA, err := s.getHistoricalPrices(ctx, symbolA, config.StatArbLookbackPeriod)
			if err != nil {
				continue
			}

			pricesB, err := s.getHistoricalPrices(ctx, symbolB, config.StatArbLookbackPeriod)
			if err != nil {
				continue
			}

			// 计算协整系数
			cointegration, err := s.CalculateCointegration(symbolA, symbolB, pricesA, pricesB)
			if err != nil {
				continue
			}

			// 检查协整关系是否足够强（通常协整检验的p值小于0.05）
			if cointegration < 0.8 { // 简化的阈值
				continue
			}

			// 计算价差
			spread := s.calculatePriceSpread(pricesA, pricesB)
			mean := s.calculateMean(spread)
			stdDev := s.calculateStdDev(spread, mean)

			// 检测均值回归信号
			if s.DetectMeanReversionSignal(spread[len(spread)-1], mean, stdDev) {
				// 计算偏离程度
				deviation := (spread[len(spread)-1] - mean) / stdDev

				opportunity := arbitrage.ArbitrageOpportunity{
					Type:          "statistical",
					Symbol:        fmt.Sprintf("%s-%s", symbolA, symbolB),
					ProfitPercent: math.Abs(deviation) * 0.5, // 简化的利润估计
					ProfitAmount:  0,
					Volume:        100000.0, // 简化
					Confidence:    s.calculateStatArbConfidence(math.Abs(deviation)),
					Timestamp:     time.Now().Unix(),
				}

				opportunities = append(opportunities, opportunity)
			}
		}
	}

	return opportunities, nil
}

// CalculateCointegration 计算协整系数
func (s *Scanner) CalculateCointegration(symbolA, symbolB string, pricesA, pricesB []float64) (float64, error) {
	if len(pricesA) != len(pricesB) || len(pricesA) < 10 {
		return 0.0, fmt.Errorf("价格数据不足或不匹配")
	}

	// 简化的协整检验（实际应该使用Engle-Granger检验）
	// 这里使用相关系数作为协整的近似估计

	correlation := s.calculateCorrelation(pricesA, pricesB)

	// 对于协整关系，相关系数通常很高（>0.8）
	// 这里返回一个基于相关系数的协整度量
	cointegration := math.Abs(correlation)

	return cointegration, nil
}

// DetectMeanReversionSignal 检测均值回归信号
func (s *Scanner) DetectMeanReversionSignal(spread, mean, stdDev float64) bool {
	if stdDev == 0 {
		return false
	}

	// 计算偏离程度（单位：标准差）
	deviation := (spread - mean) / stdDev

	// 如果偏离超过2个标准差，认为有套利机会
	return math.Abs(deviation) >= 2.0
}

// calculateCorrelation 计算相关系数
func (s *Scanner) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) {
		return 0
	}

	n := float64(len(x))
	if n == 0 {
		return 0
	}

	// 计算均值
	meanX := 0.0
	meanY := 0.0
	for i := 0; i < len(x); i++ {
		meanX += x[i]
		meanY += y[i]
	}
	meanX /= n
	meanY /= n

	// 计算协方差和方差
	cov := 0.0
	varX := 0.0
	varY := 0.0

	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		cov += dx * dy
		varX += dx * dx
		varY += dy * dy
	}

	if varX == 0 || varY == 0 {
		return 0
	}

	return cov / math.Sqrt(varX*varY)
}

// calculatePriceSpread 计算价格价差
func (s *Scanner) calculatePriceSpread(pricesA, pricesB []float64) []float64 {
	if len(pricesA) != len(pricesB) {
		return nil
	}

	spread := make([]float64, len(pricesA))
	for i := 0; i < len(pricesA); i++ {
		if pricesB[i] != 0 {
			spread[i] = pricesA[i] / pricesB[i] // 标准化价差
		}
	}

	return spread
}

// calculateMean 计算均值
func (s *Scanner) calculateMean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}

	return sum / float64(len(data))
}

// calculateStdDev 计算标准差
func (s *Scanner) calculateStdDev(data []float64, mean float64) float64 {
	if len(data) <= 1 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff
	}

	return math.Sqrt(sum / float64(len(data)-1))
}

// calculateStatArbConfidence 计算统计套利置信度
func (s *Scanner) calculateStatArbConfidence(deviation float64) float64 {
	// 基于偏离程度计算置信度
	if deviation >= 3.0 {
		return 0.9 // 极大偏离，高置信度
	} else if deviation >= 2.5 {
		return 0.8 // 大偏离，高置信度
	} else if deviation >= 2.0 {
		return 0.6 // 中等偏离，中等置信度
	}

	return 0.4 // 小偏离，低置信度
}

// getHistoricalPrices 获取历史价格（模拟数据）
func (s *Scanner) getHistoricalPrices(ctx context.Context, symbol string, limit int) ([]float64, error) {
	// 模拟历史价格数据
	// 实际实现应该从数据库或API获取真实历史数据

	basePrice := s.getBasePriceForSymbol(symbol)
	prices := make([]float64, limit)

	for i := 0; i < limit; i++ {
		// 添加一些随机波动
		noise := (float64(i%20) - 10.0) / 100.0 // -10% 到 +10% 的噪声
		prices[i] = basePrice * (1 + noise)
	}

	return prices, nil
}

// getBasePriceForSymbol 获取基础价格
func (s *Scanner) getBasePriceForSymbol(symbol string) float64 {
	prices := map[string]float64{
		"BTCUSDT": 50000.0,
		"ETHUSDT": 3000.0,
		"BNBUSDT": 300.0,
		"ADAUSDT": 1.2,
		"SOLUSDT": 100.0,
		"DOTUSDT": 15.0,
	}

	if price, exists := prices[symbol]; exists {
		return price
	}

	return 1.0
}
