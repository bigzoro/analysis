package server

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"
)

// DataFusion 数据融合器
type DataFusion struct {
	server          *Server
	coinGeckoClient *CoinGeckoClient
	cache           map[string]*FusionResult
	cacheExpiry     time.Duration
}

// FusionResult 数据融合结果
type FusionResult struct {
	Symbol      string
	Data        MarketDataPoint
	Sources     []string // 数据来源
	Confidence  float64  // 数据置信度 (0-1)
	LastUpdated time.Time
	Quality     DataQuality // 数据质量评估
}

// DataQuality 数据质量评估
type DataQuality struct {
	Completeness float64 // 完整性 (0-1)
	Consistency  float64 // 一致性 (0-1)
	Freshness    float64 // 新鲜度 (0-1)
	Overall      float64 // 综合质量 (0-1)
}

// NewDataFusion 创建数据融合器
func NewDataFusion(server *Server, coinGeckoClient *CoinGeckoClient) *DataFusion {
	return &DataFusion{
		server:          server,
		coinGeckoClient: coinGeckoClient,
		cache:           make(map[string]*FusionResult),
		cacheExpiry:     5 * time.Minute, // 缓存5分钟
	}
}

// GetFusedMarketData 获取融合后的市场数据
func (df *DataFusion) GetFusedMarketData(ctx context.Context, kind string, symbols []string) (map[string]*FusionResult, error) {
	results := make(map[string]*FusionResult)

	// 1. 获取现有数据库数据
	existingData, err := df.getExistingMarketData(ctx, kind, symbols)
	if err != nil {
		log.Printf("[DataFusion] 获取现有数据失败: %v", err)
	}

	// 2. 对缺失或质量不佳的数据，使用CoinGecko补充
	for _, symbol := range symbols {
		symbolUpper := strings.ToUpper(symbol)

		// 检查缓存
		if cached, exists := df.cache[symbolUpper]; exists && time.Since(cached.LastUpdated) < df.cacheExpiry {
			results[symbolUpper] = cached
			continue
		}

		// 获取现有数据
		var baseData *MarketDataPoint
		if existing, exists := existingData[symbolUpper]; exists {
			baseData = &existing
		}

		// 融合数据
		fusionResult, err := df.fuseDataForSymbol(ctx, symbolUpper, baseData)
		if err != nil {
			log.Printf("[DataFusion] 融合%s数据失败: %v", symbolUpper, err)
			// 如果融合失败但有基础数据，仍然使用
			if baseData != nil {
				fusionResult = &FusionResult{
					Symbol:      symbolUpper,
					Data:        *baseData,
					Sources:     []string{"database"},
					Confidence:  0.5,
					LastUpdated: time.Now(),
					Quality: DataQuality{
						Completeness: 0.6,
						Consistency:  0.5,
						Freshness:    0.5,
						Overall:      0.5,
					},
				}
			}
		}

		if fusionResult != nil {
			results[symbolUpper] = fusionResult
			df.cache[symbolUpper] = fusionResult
		}
	}

	return results, nil
}

// getExistingMarketData 获取现有市场数据
func (df *DataFusion) getExistingMarketData(ctx context.Context, kind string, symbols []string) (map[string]MarketDataPoint, error) {
	// 获取最新的市场快照
	snaps, tops, err := pdb.ListBinanceMarket(df.server.db.DB(), kind, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		return nil, err
	}

	if len(snaps) == 0 {
		return nil, fmt.Errorf("no market snapshots found")
	}

	// 使用最新的快照
	latestSnap := snaps[len(snaps)-1]
	candidates, exists := tops[latestSnap.ID]
	if !exists || len(candidates) == 0 {
		return nil, fmt.Errorf("no market data in latest snapshot")
	}

	result := make(map[string]MarketDataPoint)

	for _, item := range candidates {
		// 转换价格
		price, err := parseFloatSafe(item.LastPrice)
		if err != nil || price <= 0 {
			continue
		}

		volume, _ := parseFloatSafe(item.Volume)

		symbol := strings.ToUpper(item.Symbol)
		result[symbol] = MarketDataPoint{
			Symbol:         symbol,
			BaseSymbol:     extractBaseSymbol(symbol),
			Price:          price,
			PriceChange24h: item.PctChange,
			Volume24h:      volume,
			MarketCap:      item.MarketCapUSD,
			Timestamp:      item.CreatedAt,
		}
	}

	return result, nil
}

// fuseDataForSymbol 为单个币种融合数据
func (df *DataFusion) fuseDataForSymbol(ctx context.Context, symbol string, existingData *MarketDataPoint) (*FusionResult, error) {
	result := &FusionResult{
		Symbol:      symbol,
		Sources:     []string{},
		LastUpdated: time.Now(),
	}

	// 1. 获取CoinGecko数据
	cgData, cgErr := df.coinGeckoClient.GetCoinBySymbol(ctx, symbol)
	if cgErr != nil {
		log.Printf("[DataFusion] CoinGecko数据获取失败 %s: %v", symbol, cgErr)
	} else {
		result.Sources = append(result.Sources, "coingecko")
	}

	// 2. 如果有现有数据作为基础
	if existingData != nil {
		result.Data = *existingData
		result.Sources = append(result.Sources, "database")
		result.Confidence = 0.7
	}

	// 3. 用CoinGecko数据增强
	if cgData != nil {
		result = df.enhanceWithCoinGeckoData(result, *cgData)
		result.Confidence = 0.9
	}

	// 4. 评估数据质量
	result.Quality = df.assessDataQuality(result)

	return result, nil
}

// enhanceWithCoinGeckoData 用CoinGecko数据增强
func (df *DataFusion) enhanceWithCoinGeckoData(result *FusionResult, cgData CoinGeckoMarketData) *FusionResult {
	// 如果没有基础数据，直接使用CoinGecko数据
	if len(result.Sources) == 1 && result.Sources[0] == "coingecko" {
		result.Data = df.coinGeckoClient.ConvertToMarketDataPoint(cgData)
		return result
	}

	// 融合价格数据（加权平均）
	if result.Data.Price > 0 && cgData.CurrentPrice > 0 {
		// 数据库权重0.7，CoinGecko权重0.3（因为数据库数据更实时）
		result.Data.Price = result.Data.Price*0.7 + cgData.CurrentPrice*0.3
	} else if cgData.CurrentPrice > 0 {
		result.Data.Price = cgData.CurrentPrice
	}

	// 融合价格变化
	if result.Data.PriceChange24h == 0 && cgData.PriceChangePct24h != 0 {
		result.Data.PriceChange24h = cgData.PriceChangePct24h
	}

	// 融合成交量（取较大值）
	if cgData.TotalVolume > result.Data.Volume24h {
		result.Data.Volume24h = cgData.TotalVolume
	}

	// 增强市值数据
	if result.Data.MarketCap == nil && cgData.MarketCap > 0 {
		result.Data.MarketCap = &cgData.MarketCap
	}

	// 更新时间戳（取最新）
	cgTime, err := time.Parse(time.RFC3339, cgData.LastUpdated)
	if err == nil && cgTime.After(result.Data.Timestamp) {
		result.Data.Timestamp = cgTime
	}

	return result
}

// assessDataQuality 评估数据质量
func (df *DataFusion) assessDataQuality(result *FusionResult) DataQuality {
	quality := DataQuality{}

	// 完整性评估
	completenessScore := 0
	totalFields := 4 // price, price_change, volume, market_cap

	if result.Data.Price > 0 {
		completenessScore++
	}
	if result.Data.PriceChange24h != 0 {
		completenessScore++
	}
	if result.Data.Volume24h > 0 {
		completenessScore++
	}
	if result.Data.MarketCap != nil && *result.Data.MarketCap > 0 {
		completenessScore++
	}

	quality.Completeness = float64(completenessScore) / float64(totalFields)

	// 新鲜度评估（基于时间戳）
	freshnessHours := time.Since(result.Data.Timestamp).Hours()
	if freshnessHours < 1 {
		quality.Freshness = 1.0
	} else if freshnessHours < 6 {
		quality.Freshness = 0.8
	} else if freshnessHours < 24 {
		quality.Freshness = 0.6
	} else {
		quality.Freshness = 0.3
	}

	// 一致性评估（基于数据来源数量）
	sourceCount := len(result.Sources)
	if sourceCount >= 2 {
		quality.Consistency = 0.9
	} else if sourceCount == 1 {
		quality.Consistency = 0.7
	} else {
		quality.Consistency = 0.3
	}

	// 综合质量
	quality.Overall = (quality.Completeness*0.4 + quality.Freshness*0.3 + quality.Consistency*0.3)

	return quality
}

// GetDataQualityStats 获取数据质量统计
func (df *DataFusion) GetDataQualityStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cache_size":   len(df.cache),
		"cache_expiry": df.cacheExpiry.String(),
		"sources":      []string{"database", "coingecko"},
	}

	// 统计缓存中的数据质量分布
	qualityDistribution := map[string]int{
		"excellent": 0, // >0.8
		"good":      0, // 0.6-0.8
		"fair":      0, // 0.4-0.6
		"poor":      0, // <0.4
	}

	for _, result := range df.cache {
		switch {
		case result.Quality.Overall > 0.8:
			qualityDistribution["excellent"]++
		case result.Quality.Overall > 0.6:
			qualityDistribution["good"]++
		case result.Quality.Overall > 0.4:
			qualityDistribution["fair"]++
		default:
			qualityDistribution["poor"]++
		}
	}

	stats["quality_distribution"] = qualityDistribution
	return stats
}

// CleanExpiredCache 清理过期缓存
func (df *DataFusion) CleanExpiredCache() {
	now := time.Now()
	for symbol, result := range df.cache {
		if now.Sub(result.LastUpdated) > df.cacheExpiry {
			delete(df.cache, symbol)
		}
	}
}

// parseFloatSafe 安全的float64解析
func parseFloatSafe(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// extractBaseSymbol 提取基础符号（简化版）
func extractBaseSymbol(symbol string) string {
	// 移除常见的交易对后缀
	symbol = strings.TrimSuffix(symbol, "USDT")
	symbol = strings.TrimSuffix(symbol, "BUSD")
	symbol = strings.TrimSuffix(symbol, "USDC")
	symbol = strings.TrimSuffix(symbol, "BTC")
	symbol = strings.TrimSuffix(symbol, "ETH")

	return strings.ToLower(symbol)
}
