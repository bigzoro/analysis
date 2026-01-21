package cross_exchange

import (
	"analysis/internal/server/strategy/arbitrage"
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// Scanner 跨交易所套利扫描器实现
type Scanner struct {
	// TODO: 注入依赖，如多交易所数据提供者等
}

// NewScanner 创建跨交易所套利扫描器
func NewScanner() arbitrage.CrossExchangeScanner {
	return &Scanner{}
}

// CompareExchangePrices 比较交易所价格
func (s *Scanner) CompareExchangePrices(ctx context.Context, symbol string, exchanges []string) ([]arbitrage.ArbitrageOpportunity, error) {
	var opportunities []arbitrage.ArbitrageOpportunity

	if len(exchanges) < 2 {
		return opportunities, fmt.Errorf("至少需要2个交易所进行比较")
	}

	// 收集所有交易所的价格
	prices := make(map[string]*arbitrage.PriceData)

	for _, exchange := range exchanges {
		price, err := s.GetExchangePrice(ctx, symbol, exchange)
		if err != nil {
			log.Printf("[CrossExchange] 获取%s在%s的价格失败: %v", symbol, exchange, err)
			continue
		}
		prices[exchange] = price
	}

	if len(prices) < 2 {
		return opportunities, fmt.Errorf("有效的价格数据不足")
	}

	// 找出最优买卖价差
	minPrice := float64(1<<63 - 1) // 最大float64
	maxPrice := 0.0
	minExchange := ""
	maxExchange := ""

	for exchange, price := range prices {
		if price.Price < minPrice {
			minPrice = price.Price
			minExchange = exchange
		}
		if price.Price > maxPrice {
			maxPrice = price.Price
			maxExchange = exchange
		}
	}

	// 计算价差百分比
	if minPrice > 0 {
		spreadPercent := ((maxPrice - minPrice) / minPrice) * 100

		// 检查是否有套利机会（价差大于交易成本）
		minProfitThreshold := 0.5 // 最小0.5%的价差才有套利价值

		if spreadPercent >= minProfitThreshold {
			opportunity := arbitrage.ArbitrageOpportunity{
				Type:          "cross_exchange",
				Symbol:        symbol,
				ExchangeA:     minExchange,
				ExchangeB:     maxExchange,
				PriceA:        minPrice,
				PriceB:        maxPrice,
				ProfitPercent: spreadPercent,
				ProfitAmount:  0, // 需要计算具体金额
				Volume:        math.Min(prices[minExchange].Volume, prices[maxExchange].Volume),
				Confidence:    s.calculateCrossExchangeConfidence(spreadPercent),
				Timestamp:     time.Now().Unix(),
			}

			opportunities = append(opportunities, opportunity)
		}
	}

	return opportunities, nil
}

// GetExchangePrice 获取交易所价格
func (s *Scanner) GetExchangePrice(ctx context.Context, symbol, exchange string) (*arbitrage.PriceData, error) {
	// 模拟从不同交易所获取价格
	// 实际实现应该调用真实的交易所API

	basePrice := s.getBasePriceForSymbol(symbol)

	// 不同交易所的价格差异（模拟）
	exchangeMultipliers := map[string]float64{
		"binance":  1.0,   // 基准价格
		"huobi":    1.002, // 高于基准0.2%
		"okex":     0.998, // 低于基准0.2%
		"coinbase": 1.001, // 高于基准0.1%
	}

	multiplier, exists := exchangeMultipliers[exchange]
	if !exists {
		multiplier = 1.0
	}

	price := basePrice * multiplier
	volume := s.getSimulatedVolume(symbol)

	return &arbitrage.PriceData{
		Symbol:    symbol,
		Price:     price,
		Volume:    volume,
		Exchange:  exchange,
		Timestamp: time.Now().Unix(),
	}, nil
}

// calculateCrossExchangeSpread 计算跨交易所价差（私有方法）
func (s *Scanner) calculateCrossExchangeSpread(priceA, priceB float64) float64 {
	if priceA == 0 {
		return 0
	}

	return ((priceB - priceA) / priceA) * 100
}

// calculateCrossExchangeConfidence 计算跨交易所套利置信度
func (s *Scanner) calculateCrossExchangeConfidence(spreadPercent float64) float64 {
	// 基于价差大小计算置信度
	if spreadPercent >= 2.0 {
		return 0.9 // 大价差，高置信度
	} else if spreadPercent >= 1.0 {
		return 0.7 // 中等价差，中等置信度
	} else if spreadPercent >= 0.5 {
		return 0.5 // 小价差，低置信度
	}

	return 0.3 // 很小的价差
}

// getBasePriceForSymbol 获取交易对的基础价格
func (s *Scanner) getBasePriceForSymbol(symbol string) float64 {
	// 模拟价格数据
	basePrices := map[string]float64{
		"BTCUSDT": 50000.0,
		"ETHUSDT": 3000.0,
		"BNBUSDT": 300.0,
		"ADAUSDT": 1.2,
		"SOLUSDT": 100.0,
		"DOTUSDT": 15.0,
	}

	if price, exists := basePrices[symbol]; exists {
		return price
	}

	return 1.0 // 默认价格
}

// getSimulatedVolume 获取模拟交易量
func (s *Scanner) getSimulatedVolume(symbol string) float64 {
	// 基于交易对规模返回不同的交易量
	volumes := map[string]float64{
		"BTCUSDT": 1000000.0,
		"ETHUSDT": 800000.0,
		"BNBUSDT": 600000.0,
		"ADAUSDT": 500000.0,
		"SOLUSDT": 400000.0,
		"DOTUSDT": 300000.0,
	}

	if volume, exists := volumes[symbol]; exists {
		return volume
	}

	return 100000.0 // 默认交易量
}

// CalculateCrossExchangeSpread 计算跨交易所价差
func (s *Scanner) CalculateCrossExchangeSpread(priceA, priceB float64) float64 {
	// TODO: 实现跨交易所价差计算逻辑
	// 这里应该：
	// 1. 计算两个价格的百分比差异
	// 2. 考虑交易费率
	// 3. 返回净价差

	if priceA == 0 {
		return 0
	}

	return ((priceB - priceA) / priceA) * 100
}
