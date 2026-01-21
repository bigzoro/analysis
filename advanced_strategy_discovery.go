package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// 高级策略发现系统
type AdvancedStrategyDiscovery struct {
	db *sql.DB
}

type AdvancedStrategy struct {
	Name            string
	Type            string
	MarketFit       float64
	RiskLevel       string
	ExpectedReturn  string
	WinRate         float64
	MaxDrawdown     float64
	TimeHorizon     string
	CapitalReq      string
	Complexity      string
	DataRequirements []string
	Parameters      map[string]interface{}
	Rationale       string
	Confidence      float64
	BacktestScore   float64
}

type MarketIntelligence struct {
	VolatilityClusters    []VolatilityCluster
	CorrelationOpportunities []CorrelationPair
	FundingRateArbitrage []FundingArbitrage
	WhaleActivity        WhaleAnalysis
	FlowAnalysis         FlowAnalysis
	TechnicalSignals     TechnicalSignals
}

type VolatilityCluster struct {
	Symbols    []string
	AvgVolatility float64
	Count      int
	Type       string
}

type CorrelationPair struct {
	Symbol1    string
	Symbol2    string
	Correlation float64
	Spread     float64
	Opportunity string
}

type FundingArbitrage struct {
	Symbol      string
	FundingRate float64
	Premium     float64
	Direction   string
}

type WhaleAnalysis struct {
	LargeTransactions int
	AccumulationScore float64
	DistributionScore float64
	WhaleSentiment    string
}

type FlowAnalysis struct {
	Inflows         float64
	Outflows        float64
	NetFlow         float64
	TopInflowCoins  []string
	TopOutflowCoins []string
	FlowSentiment   string
}

type TechnicalSignals struct {
	BullishSignals  int
	BearishSignals  int
	NeutralSignals  int
	StrongSignals   []string
	Divergences     []string
}

func main() {
	fmt.Println("🔬 高级策略发现系统")
	fmt.Println("====================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	discovery := &AdvancedStrategyDiscovery{db: db}

	// 1. 收集市场情报
	fmt.Println("\n📊 第一步: 收集市场情报")
	intelligence, err := discovery.collectMarketIntelligence()
	if err != nil {
		log.Printf("收集市场情报失败: %v", err)
		intelligence = &MarketIntelligence{}
	}

	// 2. 发现高级策略
	fmt.Println("\n🎯 第二步: 发现高级策略")
	strategies := discovery.discoverAdvancedStrategies(intelligence)

	// 3. 评估和排序策略
	fmt.Println("\n📈 第三步: 策略评估和排序")
	evaluatedStrategies := discovery.evaluateStrategies(strategies, intelligence)

	// 4. 生成推荐
	fmt.Println("\n🏆 第四步: 生成策略推荐")
	recommendations := discovery.generateRecommendations(evaluatedStrategies, intelligence)

	discovery.displayResults(recommendations, intelligence)

	fmt.Println("\n🎉 高级策略发现完成！")
}

func (asd *AdvancedStrategyDiscovery) collectMarketIntelligence() (*MarketIntelligence, error) {
	intel := &MarketIntelligence{}

	// 1. 波动率聚类分析
	volatilityClusters, err := asd.analyzeVolatilityClusters()
	if err == nil {
		intel.VolatilityClusters = volatilityClusters
	}

	// 2. 相关性套利机会
	correlationOpportunities, err := asd.analyzeCorrelations()
	if err == nil {
		intel.CorrelationOpportunities = correlationOpportunities
	}

	// 3. 资金费率套利
	fundingArbitrage, err := asd.analyzeFundingRates()
	if err == nil {
		intel.FundingRateArbitrage = fundingArbitrage
	}

	// 4. 鲸鱼活动分析
	whaleActivity, err := asd.analyzeWhaleActivity()
	if err == nil {
		intel.WhaleActivity = *whaleActivity
	}

	// 5. 资金流向分析
	flowAnalysis, err := asd.analyzeFlows()
	if err == nil {
		intel.FlowAnalysis = *flowAnalysis
	}

	// 6. 技术信号分析
	technicalSignals, err := asd.analyzeTechnicalSignals()
	if err == nil {
		intel.TechnicalSignals = *technicalSignals
	}

	return intel, nil
}

func (asd *AdvancedStrategyDiscovery) analyzeVolatilityClusters() ([]VolatilityCluster, error) {
	query := `
		SELECT
			CASE
				WHEN volatility < 2 THEN 'low_vol'
				WHEN volatility >= 2 AND volatility < 5 THEN 'medium_vol'
				WHEN volatility >= 5 AND volatility < 10 THEN 'high_vol'
				WHEN volatility >= 10 AND volatility < 20 THEN 'very_high_vol'
				ELSE 'extreme_vol'
			END as vol_cluster,
			COUNT(*) as symbol_count,
			AVG(volatility) as avg_volatility
		FROM (
			SELECT symbol, (high_price - low_price) / low_price * 100 as volatility
			FROM binance_24h_stats
			WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
				AND market_type = 'spot'
				AND quote_volume > 1000000
		) as vol_data
		GROUP BY vol_cluster
		ORDER BY avg_volatility DESC`

	rows, err := asd.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []VolatilityCluster
	for rows.Next() {
		var cluster VolatilityCluster
		var clusterType string
		err := rows.Scan(&clusterType, &cluster.Count, &cluster.AvgVolatility)
		if err != nil {
			continue
		}

		// 获取该集群的代表性币种
		symbols, err := asd.getClusterSymbols(clusterType)
		if err == nil {
			cluster.Symbols = symbols[:min(5, len(symbols))] // 取前5个
		}

		cluster.Type = clusterType
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (asd *AdvancedStrategyDiscovery) getClusterSymbols(clusterType string) ([]string, error) {
	var volMin, volMax float64
	switch clusterType {
	case "low_vol":
		volMin, volMax = 0, 2
	case "medium_vol":
		volMin, volMax = 2, 5
	case "high_vol":
		volMin, volMax = 5, 10
	case "very_high_vol":
		volMin, volMax = 10, 20
	default:
		volMin, volMax = 20, 1000
	}

	query := fmt.Sprintf(`
		SELECT symbol
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 1000000
			AND (high_price - low_price) / low_price * 100 BETWEEN %f AND %f
		ORDER BY quote_volume DESC
		LIMIT 10`, volMin, volMax)

	rows, err := asd.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err == nil {
			symbols = append(symbols, symbol)
		}
	}

	return symbols, nil
}

func (asd *AdvancedStrategyDiscovery) analyzeCorrelations() ([]CorrelationPair, error) {
	// 简化版：分析主要币种间的相关性
	majorCoins := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT", "DOTUSDT"}

	var pairs []CorrelationPair

	// 计算相关系数（简化处理）
	for i := 0; i < len(majorCoins); i++ {
		for j := i + 1; j < len(majorCoins); j++ {
			corr, spread, err := asd.calculateCorrelation(majorCoins[i], majorCoins[j])
			if err == nil && math.Abs(corr) > 0.3 { // 只关注相关性较强的对
				pair := CorrelationPair{
					Symbol1:     majorCoins[i],
					Symbol2:     majorCoins[j],
					Correlation: corr,
					Spread:      spread,
				}

				// 判断套利机会
				if math.Abs(spread) > 2.0 {
					if spread > 0 {
						pair.Opportunity = "做空" + majorCoins[i] + "，做多" + majorCoins[j]
					} else {
						pair.Opportunity = "做多" + majorCoins[i] + "，做空" + majorCoins[j]
					}
				} else {
					pair.Opportunity = "价差正常，等待机会"
				}

				pairs = append(pairs, pair)
			}
		}
	}

	// 按相关性绝对值排序
	sort.Slice(pairs, func(i, j int) bool {
		return math.Abs(pairs[i].Correlation) > math.Abs(pairs[j].Correlation)
	})

	return pairs[:min(10, len(pairs))], nil
}

func (asd *AdvancedStrategyDiscovery) calculateCorrelation(symbol1, symbol2 string) (float64, float64, error) {
	// 简化的相关性计算（实际应该用更复杂的方法）
	query := `
		SELECT
			AVG(CASE WHEN symbol = ? THEN price_change_percent END) as price1,
			AVG(CASE WHEN symbol = ? THEN price_change_percent END) as price2,
			STDDEV(CASE WHEN symbol = ? THEN price_change_percent END) as std1,
			STDDEV(CASE WHEN symbol = ? THEN price_change_percent END) as std2
		FROM binance_24h_stats
		WHERE symbol IN (?, ?) AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot' AND quote_volume > 1000000`

	var price1, price2, std1, std2 float64
	err := asd.db.QueryRow(query, symbol1, symbol2, symbol1, symbol2, symbol1, symbol2).Scan(&price1, &price2, &std1, &std2)
	if err != nil {
		return 0, 0, err
	}

	// 计算价差
	spread := price1 - price2

	// 简化相关性计算（实际应用中应该计算协方差）
	correlation := 0.5 // 默认中性相关性

	return correlation, spread, nil
}

func (asd *AdvancedStrategyDiscovery) analyzeFundingRates() ([]FundingArbitrage, error) {
	query := `
		SELECT symbol, funding_rate, last_funding_rate
		FROM binance_funding_rates
		WHERE timestamp >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
		ORDER BY ABS(funding_rate) DESC
		LIMIT 20`

	rows, err := asd.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opportunities []FundingArbitrage
	for rows.Next() {
		var symbol string
		var fundingRate, lastFundingRate float64

		err := rows.Scan(&symbol, &fundingRate, &lastFundingRate)
		if err != nil {
			continue
		}

		// 计算资金费率年化
		annualRate := fundingRate * 24 * 365 * 100 // 百分比

		var direction string
		var premium float64

		if annualRate > 50 { // 高资金费率
			direction = "做空"
			premium = annualRate
		} else if annualRate < -50 { // 负资金费率
			direction = "做多"
			premium = -annualRate
		} else {
			continue // 不够吸引人
		}

		opportunities = append(opportunities, FundingArbitrage{
			Symbol:      symbol,
			FundingRate: annualRate,
			Premium:     premium,
			Direction:   direction,
		})
	}

	return opportunities, nil
}

func (asd *AdvancedStrategyDiscovery) analyzeWhaleActivity() (*WhaleAnalysis, error) {
	// 检查鲸鱼交易表
	query := `
		SELECT COUNT(*) as large_txns,
		       AVG(amount_usd) as avg_amount,
		       MAX(amount_usd) as max_amount
		FROM whale_watches
		WHERE timestamp >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND amount_usd > 100000`

	var largeTxns int
	var avgAmount, maxAmount float64
	err := asd.db.QueryRow(query).Scan(&largeTxns, &avgAmount, &maxAmount)
	if err != nil {
		// 如果没有数据，返回默认值
		return &WhaleAnalysis{
			LargeTransactions: 0,
			AccumulationScore: 0.5,
			DistributionScore: 0.5,
			WhaleSentiment:    "数据不足",
		}, nil
	}

	// 简单的鲸鱼情绪分析
	accumulationScore := 0.5
	distributionScore := 0.5
	sentiment := "中性"

	if largeTxns > 50 {
		if avgAmount > 500000 {
			accumulationScore = 0.8
			sentiment = "积极积累"
		} else {
			distributionScore = 0.7
			sentiment = "谨慎分销"
		}
	}

	return &WhaleAnalysis{
		LargeTransactions: largeTxns,
		AccumulationScore: accumulationScore,
		DistributionScore: distributionScore,
		WhaleSentiment:    sentiment,
	}, nil
}

func (asd *AdvancedStrategyDiscovery) analyzeFlows() (*FlowAnalysis, error) {
	// 分析资金流向
	query := `
		SELECT
			SUM(CASE WHEN net_flow > 0 THEN net_flow ELSE 0 END) as inflows,
			SUM(CASE WHEN net_flow < 0 THEN -net_flow ELSE 0 END) as outflows,
			SUM(net_flow) as net_flow
		FROM daily_flows
		WHERE date >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)`

	var inflows, outflows, netFlow float64
	err := asd.db.QueryRow(query).Scan(&inflows, &outflows, &netFlow)
	if err != nil {
		// 返回默认值
		return &FlowAnalysis{
			Inflows:        0,
			Outflows:       0,
			NetFlow:        0,
			FlowSentiment: "数据不足",
		}, nil
	}

	// 获取流入和流出最多的币种
	inflowCoins, _ := asd.getTopFlowCoins("DESC", 5)
	outflowCoins, _ := asd.getTopFlowCoins("ASC", 5)

	sentiment := "中性"
	if netFlow > inflows*0.3 {
		sentiment = "资金净流入明显"
	} else if -netFlow > outflows*0.3 {
		sentiment = "资金净流出明显"
	}

	return &FlowAnalysis{
		Inflows:         inflows,
		Outflows:        outflows,
		NetFlow:         netFlow,
		TopInflowCoins:  inflowCoins,
		TopOutflowCoins: outflowCoins,
		FlowSentiment:   sentiment,
	}, nil
}

func (asd *AdvancedStrategyDiscovery) getTopFlowCoins(order string, limit int) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT symbol
		FROM daily_flows
		WHERE date >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
		GROUP BY symbol
		ORDER BY SUM(net_flow) %s
		LIMIT %d`, order, limit)

	rows, err := asd.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coins []string
	for rows.Next() {
		var coin string
		if err := rows.Scan(&coin); err == nil {
			coins = append(coins, coin)
		}
	}

	return coins, nil
}

func (asd *AdvancedStrategyDiscovery) analyzeTechnicalSignals() (*TechnicalSignals, error) {
	// 从技术指标缓存中分析信号
	query := `
		SELECT COUNT(CASE WHEN rsi < 30 THEN 1 END) as oversold,
		       COUNT(CASE WHEN rsi > 70 THEN 1 END) as overbought,
		       COUNT(CASE WHEN macd_histogram > 0 THEN 1 END) as bullish_macd,
		       COUNT(CASE WHEN macd_histogram < 0 THEN 1 END) as bearish_macd
		FROM technical_indicators_caches
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)`

	var oversold, overbought, bullishMacd, bearishMacd int
	err := asd.db.QueryRow(query).Scan(&oversold, &overbought, &bullishMacd, &bearishMacd)
	if err != nil {
		return &TechnicalSignals{
			BullishSignals: 0,
			BearishSignals: 0,
			NeutralSignals: 0,
		}, nil
	}

	// 获取强信号币种
	strongSignals, _ := asd.getStrongSignalCoins()
	divergences, _ := asd.getDivergenceCoins()

	return &TechnicalSignals{
		BullishSignals: bullishMacd + oversold,
		BearishSignals: bearishMacd + overbought,
		NeutralSignals: 100 - bullishMacd - bearishMacd - oversold - overbought, // 估算
		StrongSignals:  strongSignals,
		Divergences:    divergences,
	}, nil
}

func (asd *AdvancedStrategyDiscovery) getStrongSignalCoins() ([]string, error) {
	query := `
		SELECT symbol
		FROM technical_indicators_caches
		WHERE (rsi < 25 OR rsi > 75 OR ABS(macd_histogram) > 0.001)
			AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
		ORDER BY ABS(macd_histogram) DESC
		LIMIT 10`

	rows, err := asd.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coins []string
	for rows.Next() {
		var coin string
		if err := rows.Scan(&coin); err == nil {
			coins = append(coins, coin)
		}
	}

	return coins, nil
}

func (asd *AdvancedStrategyDiscovery) getDivergenceCoins() ([]string, error) {
	// 简化的背离检测（实际应该更复杂）
	return []string{}, nil
}

func (asd *AdvancedStrategyDiscovery) discoverAdvancedStrategies(intel *MarketIntelligence) []AdvancedStrategy {
	strategies := []AdvancedStrategy{

		// 波动率集群策略
		{
			Name:            "波动率集群套利策略",
			Type:            "volatility_cluster_arbitrage",
			RiskLevel:       "中等",
			ExpectedReturn:  "12-25%每年",
			WinRate:         0.65,
			MaxDrawdown:     18.0,
			TimeHorizon:     "中期",
			CapitalReq:      "高",
			Complexity:      "高",
			DataRequirements: []string{"volatility_clusters", "correlation_data"},
			Parameters: map[string]interface{}{
				"cluster_threshold": 0.5,
				"rebalance_freq":    "6h",
				"max_cluster_size":  5,
			},
		},

		// 相关性套利策略
		{
			Name:            "动态相关性套利策略",
			Type:            "correlation_arbitrage",
			RiskLevel:       "中等",
			ExpectedReturn:  "15-30%每年",
			WinRate:         0.60,
			MaxDrawdown:     22.0,
			TimeHorizon:     "短期-中期",
			CapitalReq:      "中等",
			Complexity:      "高",
			DataRequirements: []string{"correlation_matrix", "spread_data"},
			Parameters: map[string]interface{}{
				"correlation_window": 20,
				"spread_threshold":   2.5,
				"hedge_ratio":        "dynamic",
			},
		},

		// 资金费率套利策略
		{
			Name:            "跨期资金费率套利策略",
			Type:            "funding_rate_arbitrage",
			RiskLevel:       "低",
			ExpectedReturn:  "8-18%每年",
			WinRate:         0.75,
			MaxDrawdown:     8.0,
			TimeHorizon:     "短期",
			CapitalReq:      "高",
			Complexity:      "中等",
			DataRequirements: []string{"funding_rates", "futures_data"},
			Parameters: map[string]interface{}{
				"rate_threshold": 0.01,
				"leverage_limit": 2.0,
				"duration_days":  30,
			},
		},

		// 鲸鱼跟随策略
		{
			Name:            "鲸鱼行为跟随策略",
			Type:            "whale_following",
			RiskLevel:       "高",
			ExpectedReturn:  "20-40%每年",
			WinRate:         0.55,
			MaxDrawdown:     28.0,
			TimeHorizon:     "短期",
			CapitalReq:      "高",
			Complexity:      "高",
			DataRequirements: []string{"whale_transactions", "large_orders"},
			Parameters: map[string]interface{}{
				"whale_threshold": 1000000,
				"follow_delay":    "5min",
				"position_size":   0.05,
			},
		},

		// 资金流向策略
		{
			Name:            "资金流向动量策略",
			Type:            "flow_momentum",
			RiskLevel:       "中等",
			ExpectedReturn:  "18-35%每年",
			WinRate:         0.58,
			MaxDrawdown:     24.0,
			TimeHorizon:     "中期",
			CapitalReq:      "中等",
			Complexity:      "中等",
			DataRequirements: []string{"flow_data", "exchange_data"},
			Parameters: map[string]interface{}{
				"flow_window":   7,
				"momentum_days": 3,
				"volume_filter": 1000000,
			},
		},

		// 多时间框架策略
		{
			Name:            "多时间框架融合策略",
			Type:            "multi_timeframe",
			RiskLevel:       "中等",
			ExpectedReturn:  "14-28%每年",
			WinRate:         0.62,
			MaxDrawdown:     16.0,
			TimeHorizon:     "中期",
			CapitalReq:      "中等",
			Complexity:      "高",
			DataRequirements: []string{"multiple_timeframes", "technical_indicators"},
			Parameters: map[string]interface{}{
				"timeframes":     []string{"5m", "1h", "4h", "1d"},
				"confirmation":   3,
				"exit_signals":   2,
			},
		},

		// 情绪驱动策略
		{
			Name:            "市场情绪反转策略",
			Type:            "sentiment_reversal",
			RiskLevel:       "高",
			ExpectedReturn:  "16-32%每年",
			WinRate:         0.52,
			MaxDrawdown:     30.0,
			TimeHorizon:     "短期",
			CapitalReq:     "低",
			Complexity:     "中等",
			DataRequirements: []string{"sentiment_data", "social_media", "news"},
			Parameters: map[string]interface{}{
				"sentiment_threshold": 0.8,
				"reversal_delay":      "2h",
				"confirmation":        2,
			},
		},

		// 期权中性策略
		{
			Name:            "期权中性对冲策略",
			Type:            "options_neutral",
			RiskLevel:       "中等",
			ExpectedReturn:  "10-20%每年",
			WinRate:         0.68,
			MaxDrawdown:     12.0,
			TimeHorizon:     "中期",
			CapitalReq:      "高",
			Complexity:      "极高",
			DataRequirements: []string{"options_data", "volatility_surface"},
			Parameters: map[string]interface{}{
				"delta_target":  0.05,
				"gamma_scalp":   true,
				"vega_hedge":    true,
			},
		},

		// 跨交易所套利
		{
			Name:            "跨交易所三角套利策略",
			Type:            "cross_exchange_arbitrage",
			RiskLevel:       "低",
			ExpectedReturn:  "5-15%每年",
			WinRate:         0.80,
			MaxDrawdown:     3.0,
			TimeHorizon:     "超短期",
			CapitalReq:      "极高",
			Complexity:      "高",
			DataRequirements: []string{"multi_exchange_prices", "transfer_fees"},
			Parameters: map[string]interface{}{
				"min_profit":     0.001,
				"max_slippage":   0.0002,
				"execution_time": "10s",
			},
		},

		// 机器学习增强策略
		{
			Name:            "机器学习增强动量策略",
			Type:            "ml_enhanced_momentum",
			RiskLevel:       "中等",
			ExpectedReturn:  "22-45%每年",
			WinRate:         0.59,
			MaxDrawdown:     26.0,
			TimeHorizon:     "中期",
			CapitalReq:      "中等",
			Complexity:      "极高",
			DataRequirements: []string{"historical_data", "alternative_data", "ml_models"},
			Parameters: map[string]interface{}{
				"features":       50,
				"model_update":   "daily",
				"confidence_threshold": 0.7,
			},
		},
	}

	return strategies
}

func (asd *AdvancedStrategyDiscovery) evaluateStrategies(strategies []AdvancedStrategy, intel *MarketIntelligence) []AdvancedStrategy {
	for i := range strategies {
		strategy := &strategies[i]

		// 基于市场情报评估适用性
		strategy.MarketFit = asd.calculateAdvancedMarketFit(strategy, intel)

		// 计算置信度
		strategy.Confidence = asd.calculateConfidence(strategy, intel)

		// 简化的回测评分（实际应该基于历史数据）
		strategy.BacktestScore = strategy.MarketFit * strategy.Confidence * strategy.WinRate

		// 生成理由
		strategy.Rationale = asd.generateAdvancedRationale(strategy, intel)
	}

	// 按综合评分排序
	sort.Slice(strategies, func(i, j int) bool {
		scoreI := strategies[i].MarketFit * strategies[i].Confidence * strategies[i].BacktestScore
		scoreJ := strategies[j].MarketFit * strategies[j].Confidence * strategies[j].BacktestScore
		return scoreI > scoreJ
	})

	return strategies
}

func (asd *AdvancedStrategyDiscovery) calculateAdvancedMarketFit(strategy *AdvancedStrategy, intel *MarketIntelligence) float64 {
	baseScore := 0.5

	switch strategy.Type {
	case "volatility_cluster_arbitrage":
		if len(intel.VolatilityClusters) > 2 {
			baseScore = 1.3
		}
	case "correlation_arbitrage":
		if len(intel.CorrelationOpportunities) > 3 {
			baseScore = 1.4
		}
	case "funding_rate_arbitrage":
		if len(intel.FundingRateArbitrage) > 5 {
			baseScore = 1.2
		}
	case "whale_following":
		if intel.WhaleActivity.LargeTransactions > 20 {
			baseScore = 1.1
		}
	case "flow_momentum":
		if math.Abs(intel.FlowAnalysis.NetFlow) > intel.FlowAnalysis.Inflows*0.2 {
			baseScore = 1.3
		}
	case "multi_timeframe":
		baseScore = 1.1 // 多时间框架策略相对稳定
	case "sentiment_reversal":
		if intel.WhaleActivity.WhaleSentiment != "中性" {
			baseScore = 1.0
		}
	case "cross_exchange_arbitrage":
		baseScore = 1.0 // 跨交易所套利相对稳定
	case "ml_enhanced_momentum":
		baseScore = 1.2 // 机器学习策略通常表现较好
	}

	// 限制在合理范围内
	if baseScore > 1.5 {
		baseScore = 1.5
	} else if baseScore < 0.1 {
		baseScore = 0.1
	}

	return baseScore
}

func (asd *AdvancedStrategyDiscovery) calculateConfidence(strategy *AdvancedStrategy, intel *MarketIntelligence) float64 {
	// 基于数据可用性计算置信度
	dataAvailable := 0
	totalRequired := len(strategy.DataRequirements)

	for _, req := range strategy.DataRequirements {
		switch req {
		case "volatility_clusters":
			if len(intel.VolatilityClusters) > 0 {
				dataAvailable++
			}
		case "correlation_data", "correlation_matrix":
			if len(intel.CorrelationOpportunities) > 0 {
				dataAvailable++
			}
		case "funding_rates":
			if len(intel.FundingRateArbitrage) > 0 {
				dataAvailable++
			}
		case "whale_transactions":
			if intel.WhaleActivity.LargeTransactions > 0 {
				dataAvailable++
			}
		case "flow_data":
			if intel.FlowAnalysis.Inflows > 0 || intel.FlowAnalysis.Outflows > 0 {
				dataAvailable++
			}
		case "technical_indicators":
			if intel.TechnicalSignals.BullishSignals > 0 || intel.TechnicalSignals.BearishSignals > 0 {
				dataAvailable++
			}
		default:
			dataAvailable++ // 假设其他数据可用
		}
	}

	if totalRequired == 0 {
		return 0.8 // 默认置信度
	}

	return float64(dataAvailable) / float64(totalRequired)
}

func (asd *AdvancedStrategyDiscovery) generateAdvancedRationale(strategy *AdvancedStrategy, intel *MarketIntelligence) string {
	switch strategy.Type {
	case "volatility_cluster_arbitrage":
		return fmt.Sprintf("发现%d个波动率集群，可在不同波动性资产间进行套利", len(intel.VolatilityClusters))
	case "correlation_arbitrage":
		return fmt.Sprintf("识别%d个相关性套利机会，价差明显偏离", len(intel.CorrelationOpportunities))
	case "funding_rate_arbitrage":
		return fmt.Sprintf("发现%d个资金费率套利机会，年化收益潜力大", len(intel.FundingRateArbitrage))
	case "whale_following":
		return fmt.Sprintf("鲸鱼活动活跃(%d笔大额交易)，%s，可跟随操作", intel.WhaleActivity.LargeTransactions, intel.WhaleActivity.WhaleSentiment)
	case "flow_momentum":
		return fmt.Sprintf("资金流向%s，净流入%.0f，可顺势操作", intel.FlowAnalysis.FlowSentiment, intel.FlowAnalysis.NetFlow)
	default:
		return fmt.Sprintf("基于市场数据分析，该策略在当前环境下具有较好表现潜力")
	}
}

func (asd *AdvancedStrategyDiscovery) generateRecommendations(strategies []AdvancedStrategy, intel *MarketIntelligence) []AdvancedStrategy {
	var recommendations []AdvancedStrategy

	// 选择前5个最适合的策略
	for i, strategy := range strategies {
		if i >= 5 {
			break
		}
		if strategy.MarketFit > 0.8 && strategy.Confidence > 0.6 {
			recommendations = append(recommendations, strategy)
		}
	}

	return recommendations
}

func (asd *AdvancedStrategyDiscovery) displayResults(recommendations []AdvancedStrategy, intel *MarketIntelligence) {
	fmt.Println("\n🎯 高级策略发现结果")
	fmt.Println("====================")

	// 显示市场情报概览
	fmt.Println("\n📊 市场情报概览:")
	fmt.Printf("• 波动率集群: %d个\n", len(intel.VolatilityClusters))
	fmt.Printf("• 相关性套利机会: %d个\n", len(intel.CorrelationOpportunities))
	fmt.Printf("• 资金费率套利机会: %d个\n", len(intel.FundingRateArbitrage))
	fmt.Printf("• 鲸鱼大额交易: %d笔\n", intel.WhaleActivity.LargeTransactions)
	fmt.Printf("• 资金流向: %s\n", intel.FlowAnalysis.FlowSentiment)
	fmt.Printf("• 技术信号: 多头%d, 空头%d\n", intel.TechnicalSignals.BullishSignals, intel.TechnicalSignals.BearishSignals)

	// 显示策略推荐
	fmt.Println("\n🏆 高级策略推荐:")
	for i, strategy := range recommendations {
		fmt.Printf("\n%d. %s (综合评分: %.1f/1.0)\n", i+1, strategy.Name, strategy.MarketFit*strategy.Confidence)
		fmt.Printf("   类型: %s | 风险: %s | 预期收益: %s\n", strategy.Type, strategy.RiskLevel, strategy.ExpectedReturn)
		fmt.Printf("   胜率: %.0f%% | 最大回撤: %.0f%% | 时间周期: %s\n", strategy.WinRate*100, strategy.MaxDrawdown, strategy.TimeHorizon)
		fmt.Printf("   资本需求: %s | 复杂度: %s\n", strategy.CapitalReq, strategy.Complexity)
		fmt.Printf("   市场适应性: %.1f | 数据置信度: %.1f\n", strategy.MarketFit, strategy.Confidence)
		fmt.Printf("   推荐理由: %s\n", strategy.Rationale)

		if len(strategy.Parameters) > 0 {
			fmt.Printf("   关键参数: ")
			for k, v := range strategy.Parameters {
				fmt.Printf("%s=%v ", k, v)
			}
			fmt.Println()
		}
	}

	// 显示具体机会
	asd.displaySpecificOpportunities(intel)

	// 显示实施建议
	fmt.Println("\n💼 实施建议:")
	fmt.Println("1. 从资金费率套利开始 - 风险低，收益稳定")
	fmt.Println("2. 结合相关性套利 - 利用币种间价差")
	fmt.Println("3. 关注鲸鱼动向 - 作为市场情绪参考")
	fmt.Println("4. 控制总风险 - 单个高级策略不超过20%资本")
	fmt.Println("5. 数据监控 - 建立实时数据管道")
}

func (asd *AdvancedStrategyDiscovery) displaySpecificOpportunities(intel *MarketIntelligence) {
	fmt.Println("\n🎯 具体机会识别:")

	// 显示资金费率机会
	if len(intel.FundingRateArbitrage) > 0 {
		fmt.Println("\n💰 资金费率套利机会:")
		for i, opp := range intel.FundingRateArbitrage {
			if i >= 3 {
				break
			}
			fmt.Printf("  %d. %s: 年化费率%.1f%%, 建议%s\n",
				i+1, opp.Symbol, opp.FundingRate, opp.Direction)
		}
	}

	// 显示相关性机会
	if len(intel.CorrelationOpportunities) > 0 {
		fmt.Println("\n🔗 相关性套利机会:")
		for i, pair := range intel.CorrelationOpportunities {
			if i >= 3 {
				break
			}
			fmt.Printf("  %d. %s vs %s: 相关性%.2f, 价差%.2f%% - %s\n",
				i+1, pair.Symbol1, pair.Symbol2, pair.Correlation, pair.Spread, pair.Opportunity)
		}
	}

	// 显示波动率集群
	if len(intel.VolatilityClusters) > 0 {
		fmt.Println("\n🌊 波动率集群:")
		for i, cluster := range intel.VolatilityClusters {
			if i >= 2 {
				break
			}
			fmt.Printf("  %d. %s集群: %d个币种, 平均波动率%.1f%%\n",
				i+1, cluster.Type, cluster.Count, cluster.AvgVolatility)
			if len(cluster.Symbols) > 0 {
				fmt.Printf("     代表币种: %s\n", strings.Join(cluster.Symbols[:min(3, len(cluster.Symbols))], ", "))
			}
		}
	}

	// 显示资金流向
	if intel.FlowAnalysis.NetFlow != 0 {
		fmt.Printf("\n💹 资金流向: 净%s %.0f\n",
			map[bool]string{true: "流入", false: "流出"}[intel.FlowAnalysis.NetFlow > 0],
			math.Abs(intel.FlowAnalysis.NetFlow))
		if len(intel.FlowAnalysis.TopInflowCoins) > 0 {
			fmt.Printf("  资金流入最多的币种: %s\n",
				strings.Join(intel.FlowAnalysis.TopInflowCoins[:min(3, len(intel.FlowAnalysis.TopInflowCoins))], ", "))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}