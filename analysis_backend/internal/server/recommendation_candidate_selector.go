package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"
)

// CandidateSelectionStrategy 候选币种选择策略
type CandidateSelectionStrategy struct {
	// 基础过滤条件
	MinMarketCapUSD   float64 // 最小市值（USD）
	MinVolume24hUSD   float64 // 最小24h成交量（USD）
	MinPriceChange24h float64 // 最小24h涨幅（%）
	MaxPriceChange24h float64 // 最大24h涨幅（%），过滤异常涨幅

	// 高级过滤条件
	MinAgeDays        int     // 最小年龄（天数）
	MaxVolatility     float64 // 最大波动率（%）
	MinLiquidityScore float64 // 最小流动性评分（0-100）
	RequireValidRSI   bool    // 要求有效RSI指标
	RequireValidMACD  bool    // 要求有效MACD指标
	MaxBBPosition     float64 // 最大布林带位置（0-1，避免价格过于接近上轨）

	// 候选池大小
	CandidatePoolSize  int // 候选池大小（从涨幅榜取前N个）
	FinalCandidateSize int // 最终候选数量（经过质量筛选后）

	// 质量评分权重
	QualityScoreWeights struct {
		MarketCapWeight   float64 // 市值权重
		VolumeWeight      float64 // 成交量权重
		PriceChangeWeight float64 // 涨幅权重
		LiquidityWeight   float64 // 流动性权重
		AgeWeight         float64 // 年龄权重（新币风险）
		TechnicalWeight   float64 // 技术指标权重
	}
}

// DefaultCandidateSelectionStrategy 默认候选选择策略
func DefaultCandidateSelectionStrategy() CandidateSelectionStrategy {
	return CandidateSelectionStrategy{
		// 基础过滤条件
		MinMarketCapUSD:   5000000, // 最小500万美元市值
		MinVolume24hUSD:   500000,  // 最小50万美元成交量
		MinPriceChange24h: -30,     // 最小涨幅-30%
		MaxPriceChange24h: 300,     // 最大涨幅300%

		// 高级过滤条件
		MinAgeDays:        90,   // 至少90天（避免新币风险）
		MaxVolatility:     250,  // 最大波动率250%
		MinLiquidityScore: 60,   // 最小流动性评分60
		RequireValidRSI:   true, // 要求有效RSI
		RequireValidMACD:  true, // 要求有效MACD
		MaxBBPosition:     0.85, // 价格不能过于接近上轨

		// 候选池大小
		CandidatePoolSize:  80, // 从涨幅榜取前80个
		FinalCandidateSize: 40, // 最终保留40个高质量候选

		// 质量评分权重
		QualityScoreWeights: struct {
			MarketCapWeight   float64
			VolumeWeight      float64
			PriceChangeWeight float64
			LiquidityWeight   float64
			AgeWeight         float64
			TechnicalWeight   float64
		}{
			MarketCapWeight:   0.20, // 市值权重
			VolumeWeight:      0.25, // 成交量权重
			PriceChangeWeight: 0.20, // 涨幅权重
			LiquidityWeight:   0.15, // 流动性权重
			AgeWeight:         0.10, // 年龄权重
			TechnicalWeight:   0.10, // 技术指标权重
		},
	}
}

// EnhancedCandidate 增强的候选币种（包含质量评分）
type EnhancedCandidate struct {
	pdb.BinanceMarketTop
	BaseSymbol    string
	QualityScore  float64 // 质量评分 0-100
	MarketCapUSD  *float64
	HasMarketCap  bool
	StrategyType  string  // 策略类型: "LONG", "SHORT", "RANGE"
	StrategyScore float64 // 策略评分 0-100
}

// selectCandidatesByMultipleStrategies 多策略候选选择（组合多种方法）
func (s *Server) selectCandidatesByMultipleStrategies(
	ctx context.Context,
	kind string,
) ([]EnhancedCandidate, error) {
	// 策略1：涨幅榜 + 质量筛选（多头策略）
	strategy1 := DefaultCandidateSelectionStrategy()
	candidates1, err := s.selectCandidatesEnhancedWithStrategy(ctx, kind, strategy1, "LONG")
	if err != nil {
		log.Printf("[WARN] Strategy 1 (LONG) failed: %v", err)
		candidates1 = []EnhancedCandidate{}
	}

	// 策略1b：跌幅榜 + 质量筛选（空头策略）
	strategy1b := DefaultCandidateSelectionStrategy()
	strategy1b.MinPriceChange24h = -50 // 允许大幅下跌
	strategy1b.MaxPriceChange24h = -1  // 至少下跌1%
	strategy1b.CandidatePoolSize = 100 // 跌幅榜前100名
	candidates1b, err := s.selectCandidatesEnhancedWithStrategy(ctx, kind, strategy1b, "SHORT")
	if err != nil {
		log.Printf("[WARN] Strategy 1b (SHORT) failed: %v", err)
		candidates1b = []EnhancedCandidate{}
	}

	// 策略4：震荡策略（价格在支撑阻力区间内的币种）
	strategy4 := DefaultCandidateSelectionStrategy()
	strategy4.MinPriceChange24h = -5 // 小幅波动
	strategy4.MaxPriceChange24h = 5  // 小幅波动
	strategy4.CandidatePoolSize = 100
	strategy4.MinMarketCapUSD = 100000 // 相对稳定的币种
	strategy4.MinVolume24hUSD = 10000  // 有一定成交量
	candidates4, err := s.selectCandidatesEnhancedWithStrategy(ctx, kind, strategy4, "RANGE")
	if err != nil {
		log.Printf("[WARN] Strategy 4 (RANGE) failed: %v", err)
		candidates4 = []EnhancedCandidate{}
	}

	// 合并并去重（考虑策略类型）
	candidateMap := make(map[string]EnhancedCandidate)

	// 合并多头策略候选
	for _, c := range candidates1 {
		key := c.Symbol + "_LONG"
		candidateMap[key] = c
	}

	// 合并空头策略候选
	for _, c := range candidates1b {
		key := c.Symbol + "_SHORT"
		candidateMap[key] = c
	}

	// 合并震荡策略候选
	for _, c := range candidates4 {
		key := c.Symbol + "_RANGE"
		candidateMap[key] = c
	}

	// 转换为切片并排序
	mergedCandidates := make([]EnhancedCandidate, 0, len(candidateMap))
	for _, c := range candidateMap {
		mergedCandidates = append(mergedCandidates, c)
	}

	sort.Slice(mergedCandidates, func(i, j int) bool {
		return mergedCandidates[i].QualityScore > mergedCandidates[j].QualityScore
	})

	return mergedCandidates, nil
}

// selectCandidatesEnhancedWithStrategy 带策略类型的候选币种选择方法
func (s *Server) selectCandidatesEnhancedWithStrategy(
	ctx context.Context,
	kind string,
	strategy CandidateSelectionStrategy,
	strategyType string,
) ([]EnhancedCandidate, error) {
	candidates, err := s.selectCandidatesEnhanced(ctx, kind, strategy)
	if err != nil {
		return nil, err
	}

	// 为每个候选设置策略类型和评分
	for i := range candidates {
		candidates[i].StrategyType = strategyType
		candidates[i].StrategyScore = s.calculateStrategyScore(ctx, candidates[i], strategyType)
	}

	return candidates, nil
}

// calculateStrategyScore 计算策略评分
func (s *Server) calculateStrategyScore(ctx context.Context, candidate EnhancedCandidate, strategyType string) float64 {
	switch strategyType {
	case "LONG":
		return s.calculateLongStrategyScore(ctx, candidate)
	case "SHORT":
		return s.calculateShortStrategyScore(ctx, candidate)
	case "RANGE":
		return s.calculateRangeStrategyScore(ctx, candidate)
	default:
		return 50.0 // 默认中等评分
	}
}

// calculateLongStrategyScore 计算多头策略评分
func (s *Server) calculateLongStrategyScore(ctx context.Context, candidate EnhancedCandidate) float64 {
	score := 50.0 // 基础分数

	// 基于技术指标调整分数
	if price, err := strconv.ParseFloat(candidate.LastPrice, 64); err == nil && price > 0 {
		// 获取技术指标
		indicators, err := s.CalculateTechnicalIndicators(ctx, candidate.Symbol, "spot")
		if err == nil && indicators != nil {
			// RSI < 70 (不过热)
			if indicators.RSI < 70 && indicators.RSI > 0 {
				score += 10
			}

			// MACD金叉信号 (MACD > MACDSignal)
			if indicators.MACD > indicators.MACDSignal {
				score += 15
			}

			// 趋势向上
			if indicators.Trend == "up" {
				score += 20
			}
		}
	}

	return math.Max(0, math.Min(100, score))
}

// calculateShortStrategyScore 计算空头策略评分
func (s *Server) calculateShortStrategyScore(ctx context.Context, candidate EnhancedCandidate) float64 {
	score := 50.0 // 基础分数

	// 基于技术指标调整分数
	if price, err := strconv.ParseFloat(candidate.LastPrice, 64); err == nil && price > 0 {
		// 获取技术指标
		indicators, err := s.CalculateTechnicalIndicators(ctx, candidate.Symbol, "spot")
		if err == nil && indicators != nil {
			// RSI > 30 (不过冷)
			if indicators.RSI > 30 && indicators.RSI > 0 {
				score += 10
			}

			// MACD死叉信号 (MACD < MACDSignal)
			if indicators.MACD < indicators.MACDSignal {
				score += 15
			}

			// 趋势向下
			if indicators.Trend == "down" {
				score += 20
			}
		}
	}

	return math.Max(0, math.Min(100, score))
}

// calculateRangeStrategyScore 计算震荡策略评分
func (s *Server) calculateRangeStrategyScore(ctx context.Context, candidate EnhancedCandidate) float64 {
	score := 50.0 // 基础分数

	// 震荡策略适合在支撑位和阻力位之间的币种
	if price, err := strconv.ParseFloat(candidate.LastPrice, 64); err == nil && price > 0 {
		// 获取技术指标
		indicators, err := s.CalculateTechnicalIndicators(ctx, candidate.Symbol, "spot")
		if err == nil && indicators != nil {
			// RSI在30-70之间
			if indicators.RSI >= 30 && indicators.RSI <= 70 && indicators.RSI > 0 {
				score += 10
			}

			// 价格在布林带中轨附近
			if indicators.BBPosition >= 0.3 && indicators.BBPosition <= 0.7 {
				score += 15
			}
		}
	}

	return math.Max(0, math.Min(100, score))
}

// selectCandidatesEnhanced 增强的候选币种选择方法
func (s *Server) selectCandidatesEnhanced(
	ctx context.Context,
	kind string,
	strategy CandidateSelectionStrategy,
) ([]EnhancedCandidate, error) {
	// 1. 获取市场数据（扩大时间范围，获取更多数据）
	now := time.Now().UTC()
	startTime := now.Add(-24 * time.Hour) // 扩大到24小时，获取更多快照

	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), kind, startTime, now)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	if len(snaps) == 0 {
		log.Printf("[CandidateSelection] 警告：没有找到市场快照数据（时间范围：%s 到 %s，kind=%s）", startTime.Format(time.RFC3339), now.Format(time.RFC3339), kind)
		log.Printf("[CandidateSelection] 尝试查询更早的数据...")

		// 尝试查询更早的数据（最多7天）
		startTime = now.Add(-7 * 24 * time.Hour)
		snaps, tops, err = pdb.ListBinanceMarket(s.db.DB(), kind, startTime, now)
		if err != nil {
			return nil, fmt.Errorf("获取市场数据失败: %w", err)
		}

		if len(snaps) == 0 {
			log.Printf("[CandidateSelection] 错误：7天内都没有找到市场快照数据，可能market_scanner未运行")
			return []EnhancedCandidate{}, fmt.Errorf("没有找到市场快照数据，请确保market_scanner正在运行")
		}

		log.Printf("[CandidateSelection] 找到 %d 个历史快照（时间范围：%s 到 %s）", len(snaps), startTime.Format(time.RFC3339), now.Format(time.RFC3339))
	}

	log.Printf("[CandidateSelection] 找到 %d 个市场快照", len(snaps))

	// 2. 合并所有快照的数据，去重并取最新数据
	symbolMap := make(map[string]pdb.BinanceMarketTop)
	totalItems := 0
	for _, snap := range snaps {
		if items, ok := tops[snap.ID]; ok {
			totalItems += len(items)
			for _, item := range items {
				// 如果已存在，保留排名更靠前的（rank更小）
				if existing, exists := symbolMap[item.Symbol]; !exists || item.Rank < existing.Rank {
					symbolMap[item.Symbol] = item
				}
			}
		}
	}

	log.Printf("[CandidateSelection] 合并后得到 %d 个唯一币种（总数据项：%d）", len(symbolMap), totalItems)

	// 3. 转换为切片并按涨幅排序
	allCandidates := make([]pdb.BinanceMarketTop, 0, len(symbolMap))
	for _, item := range symbolMap {
		allCandidates = append(allCandidates, item)
	}

	// 按涨幅排序
	sort.Slice(allCandidates, func(i, j int) bool {
		return allCandidates[i].PctChange > allCandidates[j].PctChange
	})

	// 4. 取前N个作为候选池
	candidatePoolSize := strategy.CandidatePoolSize
	if candidatePoolSize <= 0 {
		candidatePoolSize = 100 // 默认100
	}
	if len(allCandidates) > candidatePoolSize {
		allCandidates = allCandidates[:candidatePoolSize]
	}

	log.Printf("[CandidateSelection] 候选池大小：%d（从 %d 个币种中选取）", len(allCandidates), len(symbolMap))

	// 5. 获取黑名单
	blacklist, err := s.db.GetBinanceBlacklist(kind)
	if err != nil {
		log.Printf("[WARN] Failed to get blacklist: %v", err)
		blacklist = []string{}
	}
	blacklistMap := make(map[string]bool)
	for _, sym := range blacklist {
		blacklistMap[strings.ToUpper(sym)] = true
	}

	// 6. 批量获取市值数据（使用CoinCap）
	coinCapCache := newCoinCapCache()
	enhancedCandidates := make([]EnhancedCandidate, 0, len(allCandidates))

	// 统计过滤原因
	filterStats := struct {
		blacklist     int
		priceChange   int
		noBaseSymbol  int
		marketCapFail int
		marketCapLow  int
		volumeLow     int
		passed        int
	}{}

	for _, item := range allCandidates {
		// 过滤黑名单
		if blacklistMap[item.Symbol] {
			filterStats.blacklist++
			continue
		}

		// 基础过滤：涨幅范围
		if item.PctChange < strategy.MinPriceChange24h || item.PctChange > strategy.MaxPriceChange24h {
			filterStats.priceChange++
			continue
		}

		baseSymbol := extractBaseSymbol(item.Symbol)
		if baseSymbol == "" {
			filterStats.noBaseSymbol++
			continue
		}

		// 获取市值数据（允许失败，不强制要求）
		meta, err := coinCapCache.Get(ctx, baseSymbol)
		var marketCapUSD *float64
		hasMarketCap := false
		if err != nil {
			filterStats.marketCapFail++
			// 如果CoinCap获取失败，尝试使用Binance数据中的市值
			if item.MarketCapUSD != nil {
				marketCapUSD = item.MarketCapUSD
				hasMarketCap = true
				// 使用Binance市值数据进行过滤
				if *marketCapUSD < strategy.MinMarketCapUSD {
					filterStats.marketCapLow++
					continue
				}
			} else {
				// 完全没有市值数据，仍然保留（降级策略）
				// 不进行市值过滤
			}
		} else if meta.MarketCapUSD != nil {
			marketCapUSD = meta.MarketCapUSD
			hasMarketCap = true

			// 市值过滤（只有在有市值数据时才过滤）
			if *marketCapUSD < strategy.MinMarketCapUSD {
				filterStats.marketCapLow++
				continue
			}
		} else if item.MarketCapUSD != nil {
			// CoinCap没有数据，但Binance有，使用Binance数据
			marketCapUSD = item.MarketCapUSD
			hasMarketCap = true
			if *marketCapUSD < strategy.MinMarketCapUSD {
				filterStats.marketCapLow++
				continue
			}
		}

		// 成交量过滤（更宽松的策略）
		volume, _ := strconv.ParseFloat(item.Volume, 64)
		var volumeUSD float64
		price, priceErr := strconv.ParseFloat(item.LastPrice, 64)
		if priceErr == nil && price > 0 {
			// 使用价格计算USD成交量
			volumeUSD = volume * price
		} else {
			// 如果没有价格数据，使用原始成交量（不进行USD转换）
			// 这种情况下，如果原始成交量很大，也认为通过
			volumeUSD = volume
		}

		// 成交量过滤：只有在有价格数据且成交量过低时才过滤
		// 如果没有价格数据，跳过成交量过滤（避免误杀）
		if priceErr == nil && price > 0 && volumeUSD < strategy.MinVolume24hUSD {
			filterStats.volumeLow++
			continue
		}

		// 高级过滤：币种年龄检查
		if strategy.MinAgeDays > 0 {
			// 这里需要实现币种年龄检查逻辑
			// 暂时跳过，未来可以通过API获取币种上线时间
		}

		// 高级过滤：技术指标检查
		if strategy.RequireValidRSI || strategy.RequireValidMACD {
			// 获取技术指标（使用简化的检查）
			technical, err := s.CalculateTechnicalIndicators(ctx, item.Symbol, kind)
			if err != nil || technical == nil {
				// 如果获取失败，跳过技术指标过滤
				log.Printf("[CandidateSelection] 获取技术指标失败: %s, %v", item.Symbol, err)
			} else {
				// RSI检查
				if strategy.RequireValidRSI && (technical.RSI < 0 || technical.RSI > 100) {
					continue // RSI无效，过滤掉
				}

				// MACD检查
				if strategy.RequireValidMACD && (technical.MACD == 0 && technical.MACDSignal == 0) {
					continue // MACD无效，过滤掉
				}

				// 布林带位置检查
				if strategy.MaxBBPosition > 0 && technical.BBPosition > strategy.MaxBBPosition {
					continue // 价格过于接近上轨，过滤掉
				}
			}
		}

		// 计算质量评分（即使没有市值数据也计算）
		qualityScore := s.calculateQualityScore(item, marketCapUSD, volumeUSD, strategy)

		enhancedCandidates = append(enhancedCandidates, EnhancedCandidate{
			BinanceMarketTop: item,
			BaseSymbol:       baseSymbol,
			QualityScore:     qualityScore,
			MarketCapUSD:     marketCapUSD,
			HasMarketCap:     hasMarketCap,
		})
		filterStats.passed++
	}

	log.Printf("[CandidateSelection] 过滤统计 - 黑名单:%d 涨幅范围:%d 无基础符号:%d 市值获取失败:%d 市值过低:%d 成交量过低:%d 通过:%d",
		filterStats.blacklist, filterStats.priceChange, filterStats.noBaseSymbol,
		filterStats.marketCapFail, filterStats.marketCapLow, filterStats.volumeLow, filterStats.passed)

	// 如果过滤后候选太少，使用更宽松的策略重新筛选
	if len(enhancedCandidates) < 10 {
		log.Printf("[CandidateSelection] 警告：过滤后候选太少（%d个），使用更宽松的策略", len(enhancedCandidates))

		// 重新筛选，放宽条件
		enhancedCandidates = make([]EnhancedCandidate, 0, len(allCandidates))
		for _, item := range allCandidates {
			if blacklistMap[item.Symbol] {
				continue
			}

			// 只过滤极端涨幅
			if item.PctChange < -80 || item.PctChange > 1000 {
				continue
			}

			baseSymbol := extractBaseSymbol(item.Symbol)
			if baseSymbol == "" {
				continue
			}

			// 尝试获取市值，但不强制要求
			meta, _ := coinCapCache.Get(ctx, baseSymbol)
			var marketCapUSD *float64
			if meta.MarketCapUSD != nil {
				marketCapUSD = meta.MarketCapUSD
			} else if item.MarketCapUSD != nil {
				marketCapUSD = item.MarketCapUSD
			}

			// 计算成交量
			volume, _ := strconv.ParseFloat(item.Volume, 64)
			price, _ := strconv.ParseFloat(item.LastPrice, 64)
			volumeUSD := volume * price
			if price <= 0 {
				volumeUSD = volume
			}

			qualityScore := s.calculateQualityScore(item, marketCapUSD, volumeUSD, strategy)

			enhancedCandidates = append(enhancedCandidates, EnhancedCandidate{
				BinanceMarketTop: item,
				BaseSymbol:       baseSymbol,
				QualityScore:     qualityScore,
				MarketCapUSD:     marketCapUSD,
				HasMarketCap:     marketCapUSD != nil,
			})
		}

		log.Printf("[CandidateSelection] 宽松策略筛选后得到 %d 个候选", len(enhancedCandidates))
	}

	// 7. 按质量评分排序
	sort.Slice(enhancedCandidates, func(i, j int) bool {
		// 优先按质量评分排序，如果质量评分相同，按涨幅排序
		if math.Abs(enhancedCandidates[i].QualityScore-enhancedCandidates[j].QualityScore) < 0.01 {
			return enhancedCandidates[i].PctChange > enhancedCandidates[j].PctChange
		}
		return enhancedCandidates[i].QualityScore > enhancedCandidates[j].QualityScore
	})

	// 8. 取前N个高质量候选
	finalSize := strategy.FinalCandidateSize
	if finalSize <= 0 {
		finalSize = 50 // 默认50
	}
	if len(enhancedCandidates) > finalSize {
		enhancedCandidates = enhancedCandidates[:finalSize]
	}

	log.Printf("[CandidateSelection] 从 %d 个币种中筛选出 %d 个高质量候选", len(allCandidates), len(enhancedCandidates))

	return enhancedCandidates, nil
}

// calculateQualityScore 计算币种质量评分
func (s *Server) calculateQualityScore(
	item pdb.BinanceMarketTop,
	marketCapUSD *float64,
	volumeUSD float64,
	strategy CandidateSelectionStrategy,
) float64 {
	score := 0.0
	weights := strategy.QualityScoreWeights

	// 1. 市值评分（0-100）
	var marketCapScore float64
	if marketCapUSD != nil {
		// 市值越大，评分越高（对数归一化）
		if *marketCapUSD > 0 {
			// 使用对数归一化：log10(mcap) / log10(max_mcap) * 100
			// 假设最大市值为1万亿
			maxMcap := 1e12
			if *marketCapUSD >= maxMcap {
				marketCapScore = 100
			} else {
				logMcap := math.Log10(*marketCapUSD + 1)
				logMax := math.Log10(maxMcap)
				marketCapScore = (logMcap / logMax) * 100
			}
		}
	}
	score += marketCapScore * weights.MarketCapWeight

	// 2. 成交量评分（0-100）
	var volumeScore float64
	if volumeUSD > 0 {
		// 成交量越大，评分越高（对数归一化）
		maxVolume := 1e10 // 假设最大成交量为100亿
		if volumeUSD >= maxVolume {
			volumeScore = 100
		} else {
			logVolume := math.Log10(volumeUSD + 1)
			logMax := math.Log10(maxVolume)
			volumeScore = (logVolume / logMax) * 100
		}
	}
	score += volumeScore * weights.VolumeWeight

	// 3. 涨幅评分（0-100）
	// 涨幅在合理范围内（5%-50%）得分较高
	var priceChangeScore float64
	if item.PctChange > 0 {
		if item.PctChange >= 5 && item.PctChange <= 50 {
			// 最佳涨幅区间：5%-50%
			priceChangeScore = 100
		} else if item.PctChange < 5 {
			// 涨幅太小，按比例给分
			priceChangeScore = (item.PctChange / 5) * 100
		} else {
			// 涨幅过大，可能不稳定，降低评分
			priceChangeScore = math.Max(0, 100-(item.PctChange-50)*2)
		}
	} else {
		// 负涨幅，给较低分
		priceChangeScore = math.Max(0, 50+item.PctChange*2)
	}
	score += priceChangeScore * weights.PriceChangeWeight

	// 4. 流动性评分（基于成交量和市值）
	var liquidityScore float64
	if marketCapUSD != nil && volumeUSD > 0 {
		// 成交量/市值比率（换手率）
		turnoverRate := volumeUSD / *marketCapUSD
		// 换手率在合理范围内（0.1-1.0）得分较高
		if turnoverRate >= 0.1 && turnoverRate <= 1.0 {
			liquidityScore = 100
		} else if turnoverRate < 0.1 {
			liquidityScore = (turnoverRate / 0.1) * 100
		} else {
			// 换手率过高，可能不稳定
			liquidityScore = math.Max(0, 100-(turnoverRate-1.0)*50)
		}
	}
	score += liquidityScore * weights.LiquidityWeight

	// 5. 年龄评分（基于币种上线时间，越老越稳定）
	var ageScore float64
	// 这里需要实现币种年龄检查逻辑
	// 暂时给默认评分
	ageScore = 80 // 默认中等评分，未来可通过API获取真实年龄
	score += ageScore * weights.AgeWeight

	// 6. 技术指标评分（基于RSI、MACD等指标）
	var technicalScore float64
	// 这里可以添加技术指标评分逻辑
	// 暂时给默认评分
	technicalScore = 75 // 默认良好评分
	score += technicalScore * weights.TechnicalWeight

	// 归一化到0-100
	score = math.Max(0, math.Min(100, score))

	return score
}
