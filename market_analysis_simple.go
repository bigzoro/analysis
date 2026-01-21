package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("=== 市场环境快速分析 ===")

	// 数据库连接
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 1. 基本市场概览
	fmt.Println("\n📊 基本市场概览 (24小时)")
	fmt.Println("───────────────────────────────")
	basicStats := getBasicStats(db)
	fmt.Printf("总币种数: %d\n", basicStats.TotalSymbols)
	fmt.Printf("活跃币种: %d\n", basicStats.ActiveSymbols)
	fmt.Printf("平均涨跌幅: %.2f%%\n", basicStats.AvgPriceChange)
	fmt.Printf("平均波动率: %.2f%%\n", basicStats.AvgVolatility)

	// 2. 波动率分布
	fmt.Println("\n🌊 波动率分布")
	fmt.Println("───────────────────────────────")
	volatilityDist := getVolatilityDistribution(db)
	for _, dist := range volatilityDist {
		fmt.Printf("• %s: %d个币种\n", dist.Range, dist.Count)
	}

	// 3. 趋势分析
	fmt.Println("\n📈 趋势分析")
	fmt.Println("───────────────────────────────")
	trendStats := getTrendStats(db)
	fmt.Printf("🐂 强势上涨: %d个币种\n", trendStats.Bullish)
	fmt.Printf("🐻 强势下跌: %d个币种\n", trendStats.Bearish)
	fmt.Printf("🔄 横盘震荡: %d个币种\n", trendStats.Oscillating)
	fmt.Printf("📊 有趋势币种: %d个币种\n", trendStats.Trending)

	// 4. 涨幅榜TOP5
	fmt.Println("\n🏆 涨幅榜 TOP5")
	fmt.Println("───────────────────────────────")
	topGainers := getTopMovers(db, "DESC", 5)
	for i, mover := range topGainers {
		fmt.Printf("%d. %-12s %+6.2f%% (波动率: %.1f%%)\n",
			i+1, mover.Symbol, mover.Change, mover.Volatility)
	}

	// 5. 跌幅榜TOP5
	fmt.Println("\n📉 跌幅榜 TOP5")
	fmt.Println("───────────────────────────────")
	topLosers := getTopMovers(db, "ASC", 5)
	for i, mover := range topLosers {
		fmt.Printf("%d. %-12s %+6.2f%% (波动率: %.1f%%)\n",
			i+1, mover.Symbol, mover.Change, mover.Volatility)
	}

	// 6. 市场状态判断
	fmt.Println("\n🎯 市场状态判断")
	fmt.Println("───────────────────────────────")
	marketState := analyzeMarketState(basicStats, trendStats)
	fmt.Printf("市场状态: %s\n", marketState.Regime)
	fmt.Printf("置信度: %.1f%%\n", marketState.Confidence*100)
	fmt.Printf("主要特征: %s\n", marketState.Description)

	// 7. 对策略的影响
	fmt.Println("\n🎪 对量化策略的影响")
	fmt.Println("───────────────────────────────")
	strategyImpact := analyzeStrategyImpact(basicStats, trendStats)
	for strategy, impact := range strategyImpact {
		fmt.Printf("%s: %s\n", strategy, impact)
	}

	fmt.Println("\n=== 分析完成 ===")
}

// 数据结构
type BasicStats struct {
	TotalSymbols   int
	ActiveSymbols  int
	AvgPriceChange float64
	AvgVolatility  float64
}

type VolatilityDist struct {
	Range string
	Count int
}

type TrendStats struct {
	Bullish     int
	Bearish     int
	Oscillating int
	Trending    int
	Total       int
}

type SymbolMover struct {
	Symbol     string
	Change     float64
	Volatility float64
}

type MarketState struct {
	Regime      string
	Confidence  float64
	Description string
}

// 查询函数
func getBasicStats(db *sql.DB) BasicStats {
	query := `
		SELECT COUNT(*) as total_symbols,
		       COUNT(CASE WHEN quote_volume > 1000000 THEN 1 END) as active_symbols,
		       AVG(price_change_percent) as avg_price_change,
		       AVG((high_price - low_price) / low_price * 100) as avg_volatility
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		    AND market_type = 'spot'
		    AND quote_volume > 100000`

	var stats BasicStats
	err := db.QueryRow(query).Scan(&stats.TotalSymbols, &stats.ActiveSymbols, &stats.AvgPriceChange, &stats.AvgVolatility)
	if err != nil {
		log.Printf("查询基本统计失败: %v", err)
	}
	return stats
}

func getVolatilityDistribution(db *sql.DB) []VolatilityDist {
	query := `
		SELECT
		    CASE
		        WHEN volatility < 1 THEN '<1%'
		        WHEN volatility < 2 THEN '1-2%'
		        WHEN volatility < 5 THEN '2-5%'
		        WHEN volatility < 10 THEN '5-10%'
		        ELSE '>10%'
		    END as volatility_range,
		    COUNT(*) as symbol_count
		FROM (
		    SELECT (high_price - low_price) / low_price * 100 as volatility
		    FROM binance_24h_stats
		    WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		        AND market_type = 'spot'
		        AND quote_volume > 100000
		) as vol_stats
		GROUP BY volatility_range
		ORDER BY symbol_count DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("查询波动率分布失败: %v", err)
		return nil
	}
	defer rows.Close()

	var distributions []VolatilityDist
	for rows.Next() {
		var dist VolatilityDist
		if err := rows.Scan(&dist.Range, &dist.Count); err != nil {
			continue
		}
		distributions = append(distributions, dist)
	}
	return distributions
}

func getTrendStats(db *sql.DB) TrendStats {
	query := `
		SELECT
		    COUNT(CASE WHEN price_change_percent > 5 THEN 1 END) as bullish_symbols,
		    COUNT(CASE WHEN price_change_percent < -5 THEN 1 END) as bearish_symbols,
		    COUNT(CASE WHEN ABS(price_change_percent) <= 5 THEN 1 END) as oscillating_symbols,
		    COUNT(CASE WHEN ABS(price_change_percent) > 2 THEN 1 END) as trending_symbols,
		    COUNT(*) as total_analyzed
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		    AND market_type = 'spot'
		    AND quote_volume > 100000`

	var stats TrendStats
	err := db.QueryRow(query).Scan(&stats.Bullish, &stats.Bearish, &stats.Oscillating, &stats.Trending, &stats.Total)
	if err != nil {
		log.Printf("查询趋势统计失败: %v", err)
	}
	return stats
}

func getTopMovers(db *sql.DB, order string, limit int) []SymbolMover {
	query := fmt.Sprintf(`
		SELECT symbol, price_change_percent,
		       (high_price - low_price) / low_price * 100 as volatility
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		    AND market_type = 'spot'
		    AND quote_volume > 100000
		ORDER BY price_change_percent %s
		LIMIT %d`, order, limit)

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("查询涨跌榜失败: %v", err)
		return nil
	}
	defer rows.Close()

	var movers []SymbolMover
	for rows.Next() {
		var mover SymbolMover
		if err := rows.Scan(&mover.Symbol, &mover.Change, &mover.Volatility); err != nil {
			continue
		}
		movers = append(movers, mover)
	}
	return movers
}

func analyzeMarketState(basic BasicStats, trend TrendStats) MarketState {
	avgVolatility := basic.AvgVolatility
	bullRatio := float64(trend.Bullish) / float64(trend.Total)
	bearRatio := float64(trend.Bearish) / float64(trend.Total)
	trendRatio := float64(trend.Trending) / float64(trend.Total)

	if avgVolatility < 2.0 && trendRatio < 0.3 {
		return MarketState{
			Regime:      "极度低迷 (Deep Freeze)",
			Confidence:  0.9,
			Description: "极低波动，几乎无趋势，投资者极度谨慎",
		}
	} else if avgVolatility < 3.0 && bullRatio < 0.2 && bearRatio < 0.2 {
		return MarketState{
			Regime:      "横盘震荡 (Sideways)",
			Confidence:  0.8,
			Description: "波动适中，多空平衡，缺乏明确方向",
		}
	} else if bearRatio > 0.4 {
		return MarketState{
			Regime:      "恐慌下跌 (Panic)",
			Confidence:  0.85,
			Description: "高比例币种下跌，市场恐慌情绪浓厚",
		}
	} else if bullRatio > 0.4 {
		return MarketState{
			Regime:      "强劲上涨 (Bull Run)",
			Confidence:  0.85,
			Description: "高比例币种上涨，市场乐观情绪高涨",
		}
	} else {
		return MarketState{
			Regime:      "温和调整 (Adjustment)",
			Confidence:  0.6,
			Description: "市场正常调整，多空力量相对平衡",
		}
	}
}

func analyzeStrategyImpact(basic BasicStats, trend TrendStats) map[string]string {
	impact := make(map[string]string)

	avgVolatility := basic.AvgVolatility
	trendRatio := float64(trend.Trending) / float64(trend.Total)

	// 均线策略
	if avgVolatility < 2.0 {
		impact["📈 均线策略"] = "❌ 极不适合 - 波动率过低，难以产生有效信号"
	} else if avgVolatility < 4.0 {
		impact["📈 均线策略"] = "⚠️ 谨慎使用 - 需要大幅降低阈值"
	} else {
		impact["📈 均线策略"] = "✅ 适合使用 - 高波动环境利于趋势捕捉"
	}

	// 统计套利
	if trendRatio > 0.6 {
		impact["📊 统计套利"] = "✅ 机会较多 - 币种间走势分化明显"
	} else if trendRatio > 0.3 {
		impact["📊 统计套利"] = "⚠️ 适度机会 - 存在一定套利空间"
	} else {
		impact["📊 统计套利"] = "❌ 机会较少 - 市场同质化严重"
	}

	// 反转策略
	if trend.Oscillating > trend.Trending {
		impact["🔄 反转策略"] = "✅ 适合使用 - 震荡市有利于反转"
	} else {
		impact["🔄 反转策略"] = "⚠️ 谨慎使用 - 趋势明显时反转风险高"
	}

	return impact
}
