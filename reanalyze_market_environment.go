package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"

	_ "github.com/go-sql-driver/mysql"
)

// 市场环境重新分析器
type MarketEnvironmentReanalyzer struct {
	db *sql.DB
}

type MarketMetrics struct {
	// 整体市场指标
	TotalSymbols      int
	ActiveSymbols     int
	AvgPriceChange    float64
	AvgVolume         float64
	MarketCapWeightedChange float64

	// 波动率指标
	VolatilityDistribution map[string]int // 波动率区间分布
	AvgVolatility       float64
	HighVolatilityCount int

	// 趋势指标
	BullishSymbols     int
	BearishSymbols     int
	SidewaysSymbols    int
	StrongTrendSymbols int

	// 成交量指标
	HighVolumeSymbols  int
	AvgVolumeRatio     float64

	// 时间序列指标
	RecentTrendStrength float64
	MomentumScore       float64
	MarketRegime        string
	RegimeConfidence    float64

	// 详细分析
	TopGainers         []SymbolChange
	TopLosers          []SymbolChange
	VolatilityLeaders  []SymbolChange
	VolumeLeaders      []SymbolChange
}

type SymbolChange struct {
	Symbol           string
	PriceChange      float64
	Volume           float64
	MarketCap        float64
	Volatility       float64
}

func main() {
	fmt.Println("🔬 市场环境重新分析器")
	fmt.Println("====================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &MarketEnvironmentReanalyzer{db: db}

	// 1. 获取当前24小时市场数据
	fmt.Println("\n📊 第一步: 获取当前市场数据")
	metrics, err := analyzer.analyzeCurrentMarketData()
	if err != nil {
		log.Printf("分析市场数据失败: %v", err)
		return
	}

	// 2. 分析波动率分布
	fmt.Println("\n📈 第二步: 分析波动率分布")
	volatilityAnalysis := analyzer.analyzeVolatilityDistribution(metrics)

	// 3. 分析趋势结构
	fmt.Println("\n📉 第三步: 分析趋势结构")
	trendAnalysis := analyzer.analyzeTrendStructure(metrics)

	// 4. 分析成交量结构
	fmt.Println("\n💹 第四步: 分析成交量结构")
	volumeAnalysis := analyzer.analyzeVolumeStructure(metrics)

	// 5. 确定市场环境
	fmt.Println("\n🌍 第五步: 确定市场环境")
	marketRegime := analyzer.determineMarketRegime(metrics, volatilityAnalysis, trendAnalysis, volumeAnalysis)

	// 6. 生成策略建议
	fmt.Println("\n🎯 第六步: 生成策略建议")
	strategyRecommendations := analyzer.generateStrategyRecommendations(marketRegime, metrics)

	// 显示完整分析报告
	analyzer.displayComprehensiveAnalysis(metrics, volatilityAnalysis, trendAnalysis, volumeAnalysis, marketRegime, strategyRecommendations)

	fmt.Println("\n🎉 市场环境重新分析完成！")
}

func (mera *MarketEnvironmentReanalyzer) analyzeCurrentMarketData() (*MarketMetrics, error) {
	metrics := &MarketMetrics{
		VolatilityDistribution: make(map[string]int),
		TopGainers:            make([]SymbolChange, 0),
		TopLosers:             make([]SymbolChange, 0),
		VolatilityLeaders:     make([]SymbolChange, 0),
		VolumeLeaders:         make([]SymbolChange, 0),
	}

	// 获取最近24小时的活跃交易对数据
	query := `
		SELECT
			symbol,
			price_change_percent,
			volume,
			quote_volume,
			high_price,
			low_price,
			open_price,
			close_time
		FROM binance_24h_stats
		WHERE market_type = 'spot'
			AND quote_volume > 100000  -- 只分析有足够流动性的交易对
			AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY quote_volume DESC
		LIMIT 200`  // 取前200个最活跃的交易对

	rows, err := mera.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询市场数据失败: %w", err)
	}
	defer rows.Close()

	var totalChange, totalVolume, weightedChange float64
	var symbolChanges []SymbolChange

	for rows.Next() {
		var symbol string
		var priceChange, volume, quoteVolume, highPrice, lowPrice, openPrice float64
		var closeTime int64

		err := rows.Scan(&symbol, &priceChange, &volume, &quoteVolume, &highPrice, &lowPrice, &openPrice, &closeTime)
		if err != nil {
			continue
		}

		// 计算波动率 (日波动幅度)
		volatility := math.Abs((highPrice - lowPrice) / openPrice)

		symbolChange := SymbolChange{
			Symbol:      symbol,
			PriceChange: priceChange,
			Volume:      quoteVolume, // 使用quote_volume作为流动性指标
			Volatility:  volatility,
		}

		symbolChanges = append(symbolChanges, symbolChange)

		// 累加统计
		totalChange += priceChange
		totalVolume += quoteVolume
		weightedChange += priceChange * quoteVolume
	}

	metrics.TotalSymbols = len(symbolChanges)
	metrics.ActiveSymbols = len(symbolChanges)

	if metrics.TotalSymbols > 0 {
		metrics.AvgPriceChange = totalChange / float64(metrics.TotalSymbols)
		metrics.AvgVolume = totalVolume / float64(metrics.TotalSymbols)
		metrics.MarketCapWeightedChange = weightedChange / totalVolume
	}

	// 分析每个交易对
	for _, sc := range symbolChanges {
		// 波动率分布
		var volRange string
		switch {
		case sc.Volatility < 0.02:
			volRange = "极低波动(<2%)"
		case sc.Volatility < 0.05:
			volRange = "低波动(2-5%)"
		case sc.Volatility < 0.10:
			volRange = "中等波动(5-10%)"
		case sc.Volatility < 0.20:
			volRange = "高波动(10-20%)"
		default:
			volRange = "极高波动(>20%)"
		}
		metrics.VolatilityDistribution[volRange]++

		// 趋势分类
		if sc.PriceChange > 5 {
			metrics.BullishSymbols++
		} else if sc.PriceChange < -5 {
			metrics.BearishSymbols++
		} else {
			metrics.SidewaysSymbols++
		}

		if math.Abs(sc.PriceChange) > 10 {
			metrics.StrongTrendSymbols++
		}

		// 高波动计数
		if sc.Volatility > 0.10 {
			metrics.HighVolatilityCount++
		}

		// 高成交量计数
		if sc.Volume > metrics.AvgVolume*2 {
			metrics.HighVolumeSymbols++
		}
	}

	// 排序获取前几名
	sort.Slice(symbolChanges, func(i, j int) bool {
		return symbolChanges[i].PriceChange > symbolChanges[j].PriceChange
	})

	// 前10涨幅榜
	for i := 0; i < len(symbolChanges) && i < 10; i++ {
		metrics.TopGainers = append(metrics.TopGainers, symbolChanges[i])
	}

	// 后10跌幅榜
	sort.Slice(symbolChanges, func(i, j int) bool {
		return symbolChanges[i].PriceChange < symbolChanges[j].PriceChange
	})

	for i := 0; i < len(symbolChanges) && i < 10; i++ {
		metrics.TopLosers = append(metrics.TopLosers, symbolChanges[i])
	}

	// 波动率前10
	sort.Slice(symbolChanges, func(i, j int) bool {
		return symbolChanges[i].Volatility > symbolChanges[j].Volatility
	})

	for i := 0; i < len(symbolChanges) && i < 10; i++ {
		metrics.VolatilityLeaders = append(metrics.VolatilityLeaders, symbolChanges[i])
	}

	// 成交量前10
	sort.Slice(symbolChanges, func(i, j int) bool {
		return symbolChanges[i].Volume > symbolChanges[j].Volume
	})

	for i := 0; i < len(symbolChanges) && i < 10; i++ {
		metrics.VolumeLeaders = append(metrics.VolumeLeaders, symbolChanges[i])
	}

	// 计算平均波动率
	var totalVolatility float64
	for _, sc := range symbolChanges {
		totalVolatility += sc.Volatility
	}
	if len(symbolChanges) > 0 {
		metrics.AvgVolatility = totalVolatility / float64(len(symbolChanges))
	}

	// 计算成交量比率
	if metrics.AvgVolume > 0 {
		metrics.AvgVolumeRatio = float64(metrics.HighVolumeSymbols) / float64(metrics.TotalSymbols)
	}

	return metrics, nil
}

type VolatilityAnalysis struct {
	DominantVolatilityRange string
	VolatilityDiversity     float64
	HighVolatilityRatio     float64
	LowVolatilityRatio      float64
	VolatilityStability     float64
	VolatilityTrend         string
}

func (mera *MarketEnvironmentReanalyzer) analyzeVolatilityDistribution(metrics *MarketMetrics) *VolatilityAnalysis {
	analysis := &VolatilityAnalysis{}

	// 找到主导波动率区间
	maxCount := 0
	for volRange, count := range metrics.VolatilityDistribution {
		if count > maxCount {
			maxCount = count
			analysis.DominantVolatilityRange = volRange
		}
	}

	// 计算波动率多样性 (熵)
	total := float64(metrics.TotalSymbols)
	diversity := 0.0
	for _, count := range metrics.VolatilityDistribution {
		if count > 0 {
			p := float64(count) / total
			diversity -= p * math.Log2(p)
		}
	}
	analysis.VolatilityDiversity = diversity

	// 计算高低波动比率
	analysis.HighVolatilityRatio = float64(metrics.HighVolatilityCount) / float64(metrics.TotalSymbols)
	analysis.LowVolatilityRatio = float64(metrics.VolatilityDistribution["极低波动(<2%)"]+metrics.VolatilityDistribution["低波动(2-5%)"]) / float64(metrics.TotalSymbols)

	// 波动率稳定性 (基于分布集中度)
	analysis.VolatilityStability = float64(maxCount) / float64(metrics.TotalSymbols)

	// 波动率趋势判断
	if analysis.LowVolatilityRatio > 0.6 {
		analysis.VolatilityTrend = "极低波动环境"
	} else if analysis.LowVolatilityRatio > 0.4 {
		analysis.VolatilityTrend = "低波动环境"
	} else if analysis.HighVolatilityRatio > 0.3 {
		analysis.VolatilityTrend = "高波动环境"
	} else {
		analysis.VolatilityTrend = "中等波动环境"
	}

	return analysis
}

type TrendAnalysis struct {
	TrendDirection      string
	TrendStrength       float64
	BullBearRatio       float64
	StrongTrendRatio    float64
	MarketSentiment     string
	TrendConsistency    float64
	TrendDiversity      float64
}

func (mera *MarketEnvironmentReanalyzer) analyzeTrendStructure(metrics *MarketMetrics) *TrendAnalysis {
	analysis := &TrendAnalysis{}

	total := float64(metrics.TotalSymbols)

	// 趋势方向
	if metrics.BullishSymbols > int(float64(metrics.BearishSymbols)*1.5) {
		analysis.TrendDirection = "强势上涨"
	} else if metrics.BearishSymbols > int(float64(metrics.BullishSymbols)*1.5) {
		analysis.TrendDirection = "强势下跌"
	} else if math.Abs(float64(metrics.BullishSymbols-metrics.BearishSymbols)) < total*0.1 {
		analysis.TrendDirection = "震荡整理"
	} else {
		analysis.TrendDirection = "温和上涨"
	}

	// 趋势强度
	strongTrendRatio := float64(metrics.StrongTrendSymbols) / total
	analysis.TrendStrength = strongTrendRatio

	// 多空比率
	if metrics.BearishSymbols > 0 {
		analysis.BullBearRatio = float64(metrics.BullishSymbols) / float64(metrics.BearishSymbols)
	} else {
		analysis.BullBearRatio = float64(metrics.BullishSymbols)
	}

	// 强趋势占比
	analysis.StrongTrendRatio = strongTrendRatio

	// 市场情绪
	if analysis.BullBearRatio > 2.0 && strongTrendRatio > 0.3 {
		analysis.MarketSentiment = "极度乐观"
	} else if analysis.BullBearRatio > 1.5 && strongTrendRatio > 0.2 {
		analysis.MarketSentiment = "乐观"
	} else if analysis.BullBearRatio < 0.5 && strongTrendRatio > 0.2 {
		analysis.MarketSentiment = "悲观"
	} else if math.Abs(analysis.BullBearRatio-1.0) < 0.2 {
		analysis.MarketSentiment = "中性"
	} else {
		analysis.MarketSentiment = "温和"
	}

	// 趋势一致性 (强趋势占比)
	analysis.TrendConsistency = strongTrendRatio

	// 趋势多样性
	diversity := 0.0
	if metrics.BullishSymbols > 0 {
		p := float64(metrics.BullishSymbols) / total
		diversity -= p * math.Log2(p)
	}
	if metrics.BearishSymbols > 0 {
		p := float64(metrics.BearishSymbols) / total
		diversity -= p * math.Log2(p)
	}
	if metrics.SidewaysSymbols > 0 {
		p := float64(metrics.SidewaysSymbols) / total
		diversity -= p * math.Log2(p)
	}
	analysis.TrendDiversity = diversity

	return analysis
}

type VolumeAnalysis struct {
	VolumeConcentration float64
	HighVolumeRatio     float64
	VolumeTrend         string
	LiquidityQuality    float64
	VolumeStability     float64
}

func (mera *MarketEnvironmentReanalyzer) analyzeVolumeStructure(metrics *MarketMetrics) *VolumeAnalysis {
	analysis := &VolumeAnalysis{}

	// 成交量集中度 (前10名占比)
	var top10Volume float64
	for i := 0; i < len(metrics.VolumeLeaders) && i < 10; i++ {
		top10Volume += metrics.VolumeLeaders[i].Volume
	}
	if metrics.AvgVolume*float64(metrics.TotalSymbols) > 0 {
		analysis.VolumeConcentration = top10Volume / (metrics.AvgVolume * float64(metrics.TotalSymbols))
	}

	// 高成交量占比
	analysis.HighVolumeRatio = float64(metrics.HighVolumeSymbols) / float64(metrics.TotalSymbols)

	// 成交量趋势
	if analysis.HighVolumeRatio > 0.4 {
		analysis.VolumeTrend = "高活跃度"
	} else if analysis.HighVolumeRatio > 0.2 {
		analysis.VolumeTrend = "中等活跃度"
	} else {
		analysis.VolumeTrend = "低活跃度"
	}

	// 流动性质量 (基于成交量分布)
	analysis.LiquidityQuality = analysis.HighVolumeRatio * (1 - analysis.VolumeConcentration)

	// 成交量稳定性 (基于高活跃交易对占比)
	analysis.VolumeStability = analysis.HighVolumeRatio

	return analysis
}

type MarketRegimeDetermination struct {
	PrimaryRegime    string
	SecondaryRegime  string
	Confidence       float64
	KeyIndicators    map[string]float64
	Rationale        []string
	RegimeStability  float64
	ChangeProbability float64
}

func (mera *MarketEnvironmentReanalyzer) determineMarketRegime(metrics *MarketMetrics, volAnalysis *VolatilityAnalysis, trendAnalysis *TrendAnalysis, volumeAnalysis *VolumeAnalysis) *MarketRegimeDetermination {
	determination := &MarketRegimeDetermination{
		KeyIndicators: make(map[string]float64),
		Rationale:     make([]string, 0),
	}

	// 计算各种指标
	determination.KeyIndicators["volatility_level"] = metrics.AvgVolatility
	determination.KeyIndicators["trend_strength"] = trendAnalysis.TrendStrength
	determination.KeyIndicators["bull_bear_ratio"] = trendAnalysis.BullBearRatio
	determination.KeyIndicators["high_volume_ratio"] = volumeAnalysis.HighVolumeRatio
	determination.KeyIndicators["volatility_stability"] = volAnalysis.VolatilityStability

	// 市场环境判断逻辑
	var bullScore, bearScore, sidewaysScore, volatileScore float64

	// 基于波动率的评分
	if volAnalysis.LowVolatilityRatio > 0.5 {
		sidewaysScore += 0.3
		bullScore += 0.1
	} else if volAnalysis.HighVolatilityRatio > 0.3 {
		volatileScore += 0.4
	} else {
		bullScore += 0.1
		sidewaysScore += 0.1
	}

	// 基于趋势的评分
	if trendAnalysis.BullBearRatio > 1.5 {
		bullScore += 0.4
	} else if trendAnalysis.BullBearRatio < 0.67 {
		bearScore += 0.4
	} else {
		sidewaysScore += 0.3
	}

	// 基于强趋势占比的评分
	if trendAnalysis.StrongTrendRatio > 0.25 {
		if trendAnalysis.BullBearRatio > 1.2 {
			bullScore += 0.3
		} else if trendAnalysis.BullBearRatio < 0.83 {
			bearScore += 0.3
		}
	} else {
		sidewaysScore += 0.2
	}

	// 基于成交量的评分
	if volumeAnalysis.HighVolumeRatio > 0.35 {
		bullScore += 0.2
		sidewaysScore += 0.1
	} else if volumeAnalysis.HighVolumeRatio < 0.15 {
		sidewaysScore += 0.2
	}

	// 确定主要市场环境
	maxScore := math.Max(math.Max(bullScore, bearScore), math.Max(sidewaysScore, volatileScore))

	if maxScore == bullScore && bullScore > 0.5 {
		determination.PrimaryRegime = "bull_trend"
		determination.Confidence = bullScore
		determination.Rationale = append(determination.Rationale,
			fmt.Sprintf("上涨趋势明显，多空比为%.2f", trendAnalysis.BullBearRatio))
	} else if maxScore == bearScore && bearScore > 0.5 {
		determination.PrimaryRegime = "bear_trend"
		determination.Confidence = bearScore
		determination.Rationale = append(determination.Rationale,
			fmt.Sprintf("下跌趋势明显，多空比为%.2f", trendAnalysis.BullBearRatio))
	} else if maxScore == volatileScore && volatileScore > 0.4 {
		determination.PrimaryRegime = "high_volatility"
		determination.Confidence = volatileScore
		determination.Rationale = append(determination.Rationale,
			fmt.Sprintf("高波动环境，波动率达%.1f%%", metrics.AvgVolatility*100))
	} else {
		determination.PrimaryRegime = "sideways"
		determination.Confidence = sidewaysScore
		determination.Rationale = append(determination.Rationale,
			"市场整体呈现震荡整理态势")
	}

	// 次要环境判断
	scores := map[string]float64{
		"bull_trend":       bullScore,
		"bear_trend":       bearScore,
		"sideways":         sidewaysScore,
		"high_volatility":  volatileScore,
	}

	// 找到次高分
	var secondMaxScore float64
	for _, score := range scores {
		if score < maxScore && score > secondMaxScore {
			secondMaxScore = score
		}
	}

	for regime, score := range scores {
		if score == secondMaxScore && score > 0.2 {
			determination.SecondaryRegime = regime
			break
		}
	}

	// 环境稳定性 (基于一致性指标)
	determination.RegimeStability = (trendAnalysis.TrendConsistency + volAnalysis.VolatilityStability + volumeAnalysis.VolumeStability) / 3.0

	// 变化概率 (基于多样性指标)
	determination.ChangeProbability = (trendAnalysis.TrendDiversity + volAnalysis.VolatilityDiversity) / 2.0

	return determination
}

type StrategyRecommendations struct {
	PrimaryStrategy     string
	SecondaryStrategies []string
	AvoidStrategies     []string
	ExpectedPerformance map[string]float64
	RiskConsiderations  []string
	ImplementationPriority []string
	MarketTiming        string
}

func (mera *MarketEnvironmentReanalyzer) generateStrategyRecommendations(regime *MarketRegimeDetermination, metrics *MarketMetrics) *StrategyRecommendations {
	recs := &StrategyRecommendations{
		ExpectedPerformance: make(map[string]float64),
		RiskConsiderations:  make([]string, 0),
	}

	switch regime.PrimaryRegime {
	case "bull_trend":
		recs.PrimaryStrategy = "动量策略"
		recs.SecondaryStrategies = []string{"突破策略", "趋势跟随策略"}
		recs.AvoidStrategies = []string{"均值回归策略", "反转策略"}
		recs.ExpectedPerformance["动量策略"] = 0.25
		recs.ExpectedPerformance["突破策略"] = 0.20
		recs.ExpectedPerformance["趋势跟随"] = 0.18
		recs.MarketTiming = "立即执行，持续监控趋势强度"

	case "bear_trend":
		recs.PrimaryStrategy = "做空策略"
		recs.SecondaryStrategies = []string{"反转策略", "对冲策略"}
		recs.AvoidStrategies = []string{"动量策略", "突破策略"}
		recs.ExpectedPerformance["做空策略"] = 0.20
		recs.ExpectedPerformance["反转策略"] = 0.15
		recs.MarketTiming = "谨慎执行，注意反弹风险"

	case "high_volatility":
		recs.PrimaryStrategy = "波动率套利策略"
		recs.SecondaryStrategies = []string{"统计套利", "期权策略"}
		recs.AvoidStrategies = []string{"趋势跟随策略", "均线策略"}
		recs.ExpectedPerformance["波动率套利"] = 0.15
		recs.ExpectedPerformance["统计套利"] = 0.12
		recs.MarketTiming = "等待波动率回落再执行"

	case "sideways":
		recs.PrimaryStrategy = "均值回归策略"
		recs.SecondaryStrategies = []string{"网格交易", "区间交易"}
		recs.AvoidStrategies = []string{"动量策略", "趋势跟随策略"}
		recs.ExpectedPerformance["均值回归"] = 0.15
		recs.ExpectedPerformance["网格交易"] = 0.12
		recs.MarketTiming = "适合长期持有，耐心等待机会"
	}

	recs.RiskConsiderations = []string{
		"市场环境可能快速变化，需要动态调整策略",
		"极端事件可能导致策略失效",
		"流动性风险需要特别关注",
		"执行成本可能影响小幅收益策略",
	}

	recs.ImplementationPriority = []string{
		"完善市场环境检测机制",
		"实现策略动态切换",
		"建立风险监控体系",
		"准备备用策略方案",
	}

	return recs
}

func (mera *MarketEnvironmentReanalyzer) displayComprehensiveAnalysis(metrics *MarketMetrics, volAnalysis *VolatilityAnalysis, trendAnalysis *TrendAnalysis, volumeAnalysis *VolumeAnalysis, regime *MarketRegimeDetermination, recs *StrategyRecommendations) {
	fmt.Println("📊 市场环境全面分析报告")
	fmt.Println("======================")

	// 市场概览
	fmt.Println("\n🌍 市场概览:")
	fmt.Printf("• 活跃交易对: %d个\n", metrics.ActiveSymbols)
	fmt.Printf("• 平均价格变化: %.2f%%\n", metrics.AvgPriceChange)
	fmt.Printf("• 加权平均变化: %.2f%%\n", metrics.MarketCapWeightedChange)
	fmt.Printf("• 平均波动率: %.2f%%\n", metrics.AvgVolatility*100)
	fmt.Printf("• 平均成交量: %.0f USDT\n", metrics.AvgVolume)

	// 波动率分析
	fmt.Println("\n📈 波动率分析:")
	fmt.Printf("• 主导波动区间: %s\n", volAnalysis.DominantVolatilityRange)
	fmt.Printf("• 波动率稳定性: %.1f%%\n", volAnalysis.VolatilityStability*100)
	fmt.Printf("• 高波动占比: %.1f%%\n", volAnalysis.HighVolatilityRatio*100)
	fmt.Printf("• 低波动占比: %.1f%%\n", volAnalysis.LowVolatilityRatio*100)
	fmt.Printf("• 波动率趋势: %s\n", volAnalysis.VolatilityTrend)

	fmt.Println("波动率分布:")
	for volRange, count := range metrics.VolatilityDistribution {
		percentage := float64(count) / float64(metrics.TotalSymbols) * 100
		fmt.Printf("  • %s: %d个 (%.1f%%)\n", volRange, count, percentage)
	}

	// 趋势分析
	fmt.Println("\n📉 趋势分析:")
	fmt.Printf("• 趋势方向: %s\n", trendAnalysis.TrendDirection)
	fmt.Printf("• 趋势强度: %.1f%%\n", trendAnalysis.TrendStrength*100)
	fmt.Printf("• 多空比率: %.2f\n", trendAnalysis.BullBearRatio)
	fmt.Printf("• 强趋势占比: %.1f%%\n", trendAnalysis.StrongTrendRatio*100)
	fmt.Printf("• 市场情绪: %s\n", trendAnalysis.MarketSentiment)

	fmt.Printf("趋势分布:\n")
	fmt.Printf("  • 上涨币种: %d个\n", metrics.BullishSymbols)
	fmt.Printf("  • 下跌币种: %d个\n", metrics.BearishSymbols)
	fmt.Printf("  • 震荡币种: %d个\n", metrics.SidewaysSymbols)

	// 成交量分析
	fmt.Println("\n💹 成交量分析:")
	fmt.Printf("• 成交量趋势: %s\n", volumeAnalysis.VolumeTrend)
	fmt.Printf("• 高活跃占比: %.1f%%\n", volumeAnalysis.HighVolumeRatio*100)
	fmt.Printf("• 流动性质量: %.2f\n", volumeAnalysis.LiquidityQuality)
	fmt.Printf("• 成交量集中度: %.2f\n", volumeAnalysis.VolumeConcentration)

	// 市场环境判断
	fmt.Println("\n🌟 市场环境判断:")
	fmt.Printf("• 主要环境: %s (信心: %.1f%%)\n", regime.PrimaryRegime, regime.Confidence*100)
	if regime.SecondaryRegime != "" {
		fmt.Printf("• 次要环境: %s\n", regime.SecondaryRegime)
	}
	fmt.Printf("• 环境稳定性: %.1f%%\n", regime.RegimeStability*100)
	fmt.Printf("• 变化概率: %.1f%%\n", regime.ChangeProbability*100)

	fmt.Println("判断依据:")
	for _, reason := range regime.Rationale {
		fmt.Printf("  • %s\n", reason)
	}

	// 关键指标汇总
	fmt.Println("\n📊 关键指标汇总:")
	fmt.Printf("• 波动率水平: %.1f%%\n", regime.KeyIndicators["volatility_level"]*100)
	fmt.Printf("• 趋势强度: %.1f%%\n", regime.KeyIndicators["trend_strength"]*100)
	fmt.Printf("• 多空比率: %.2f\n", regime.KeyIndicators["bull_bear_ratio"])
	fmt.Printf("• 高活跃度: %.1f%%\n", regime.KeyIndicators["high_volume_ratio"]*100)

	// 策略建议
	fmt.Println("\n🎯 策略建议:")
	fmt.Printf("• 主要策略: %s\n", recs.PrimaryStrategy)
	fmt.Printf("• 辅助策略: %s\n", fmt.Sprintf("%v", recs.SecondaryStrategies))
	fmt.Printf("• 避免策略: %s\n", fmt.Sprintf("%v", recs.AvoidStrategies))
	fmt.Printf("• 市场时机: %s\n", recs.MarketTiming)

	fmt.Println("预期表现:")
	for strategy, performance := range recs.ExpectedPerformance {
		fmt.Printf("  • %s: %.1f%% 年化收益\n", strategy, performance*100)
	}

	fmt.Println("风险考虑:")
	for _, risk := range recs.RiskConsiderations {
		fmt.Printf("  • %s\n", risk)
	}

	// 前十大涨幅/跌幅
	fmt.Println("\n📈 前十大涨幅:")
	for i, gainer := range metrics.TopGainers {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s: %.2f%% (波动:%.1f%%)\n",
			i+1, gainer.Symbol, gainer.PriceChange, gainer.Volatility*100)
	}

	fmt.Println("\n📉 前十大跌幅:")
	for i, loser := range metrics.TopLosers {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s: %.2f%% (波动:%.1f%%)\n",
			i+1, loser.Symbol, loser.PriceChange, loser.Volatility*100)
	}

	fmt.Println("\n💹 前十大成交量:")
	for i, leader := range metrics.VolumeLeaders {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s: %.0f USDT (涨跌:%.2f%%)\n",
			i+1, leader.Symbol, leader.Volume, leader.PriceChange)
	}

	fmt.Println("\n⚡ 前十大波动率:")
	for i, volatile := range metrics.VolatilityLeaders {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s: %.1f%% (涨跌:%.2f%%)\n",
			i+1, volatile.Symbol, volatile.Volatility*100, volatile.PriceChange)
	}
}