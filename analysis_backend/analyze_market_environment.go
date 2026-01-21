package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	fmt.Println("=== 市场环境深度分析系统 ===")

	// 1. 读取配置文件
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 2. 连接数据库
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	gormDB, _ := db.DB()

	// 3. 执行市场环境分析
	analyzer := NewMarketEnvironmentAnalyzer(gormDB)
	report := analyzer.AnalyzeMarketEnvironment()

	// 4. 输出分析报告
	report.PrintReport()

	fmt.Println("\n=== 分析完成 ===")
}

// MarketEnvironmentReport 市场环境分析报告
type MarketEnvironmentReport struct {
	TimeRange             time.Duration
	TotalSymbols          int
	ActiveSymbols         int
	AverageVolatility     float64
	MedianVolatility      float64
	HighVolatilityCount   int
	LowVolatilityCount    int
	TrendingSymbols       int
	OscillatingSymbols    int
	BullishSymbols        int
	BearishSymbols        int
	MarketRegime          string
	RegimeConfidence      float64
	TopGainers            []SymbolStats
	TopLosers             []SymbolStats
	VolatilityDistribution map[string]int
}

// SymbolStats 币种统计信息
type SymbolStats struct {
	Symbol      string
	PriceChange float64
	Volume      float64
	Volatility  float64
	Trend       string
}

// MarketEnvironmentAnalyzer 市场环境分析器
type MarketEnvironmentAnalyzer struct {
	db *gorm.DB
}

// NewMarketEnvironmentAnalyzer 创建分析器
func NewMarketEnvironmentAnalyzer(db *gorm.DB) *MarketEnvironmentAnalyzer {
	return &MarketEnvironmentAnalyzer{db: db}
}

// AnalyzeMarketEnvironment 执行市场环境分析
func (a *MarketEnvironmentAnalyzer) AnalyzeMarketEnvironment() *MarketEnvironmentReport {
	report := &MarketEnvironmentReport{
		TimeRange:            24 * time.Hour,
		VolatilityDistribution: make(map[string]int),
	}

	// 1. 获取24小时统计数据
	symbolStats := a.get24hStats()

	// 2. 计算基础统计
	report.TotalSymbols = len(symbolStats)
	report.ActiveSymbols = a.countActiveSymbols(symbolStats)

	// 3. 分析波动率
	volatilities := a.analyzeVolatility(symbolStats, report)

	// 4. 分析趋势
	a.analyzeTrends(symbolStats, report)

	// 5. 判断市场状态
	a.determineMarketRegime(report, volatilities)

	// 6. 生成排行榜
	report.TopGainers = a.getTopGainers(symbolStats, 10)
	report.TopLosers = a.getTopLosers(symbolStats, 10)

	return report
}

// get24hStats 获取24小时统计数据
func (a *MarketEnvironmentAnalyzer) get24hStats() []SymbolStats {
	var stats []struct {
		Symbol       string
		PriceChange  float64
		QuoteVolume  float64
		HighPrice    float64
		LowPrice     float64
		CreatedAt    time.Time
	}

	// 查询最近24小时的数据
	a.db.Table("binance_24h_stats").
		Select("symbol, price_change_percent as price_change, quote_volume, high_price, low_price, created_at").
		Where("created_at >= ? AND market_type = ? AND quote_volume > 100000",
			time.Now().Add(-24*time.Hour), "spot").
		Order("quote_volume DESC").
		Limit(200).
		Scan(&stats)

	symbolStats := make([]SymbolStats, 0, len(stats))
	for _, stat := range stats {
		// 计算波动率：(最高价-最低价)/最低价
		volatility := 0.0
		if stat.LowPrice > 0 {
			volatility = (stat.HighPrice - stat.LowPrice) / stat.LowPrice * 100
		}

		symbolStats = append(symbolStats, SymbolStats{
			Symbol:      stat.Symbol,
			PriceChange: stat.PriceChange,
			Volume:      stat.QuoteVolume,
			Volatility:  volatility,
		})
	}

	return symbolStats
}

// countActiveSymbols 统计活跃币种数量
func (a *MarketEnvironmentAnalyzer) countActiveSymbols(stats []SymbolStats) int {
	count := 0
	for _, stat := range stats {
		if stat.Volume > 1000000 { // 24h交易量超过100万美元
			count++
		}
	}
	return count
}

// analyzeVolatility 分析波动率
func (a *MarketEnvironmentAnalyzer) analyzeVolatility(stats []SymbolStats, report *MarketEnvironmentReport) []float64 {
	volatilities := make([]float64, 0, len(stats))

	for _, stat := range stats {
		if stat.Volatility > 0 {
			volatilities = append(volatilities, stat.Volatility)

			// 统计波动率分布
			if stat.Volatility < 1 {
				report.VolatilityDistribution["<1%"]++
			} else if stat.Volatility < 2 {
				report.VolatilityDistribution["1-2%"]++
			} else if stat.Volatility < 5 {
				report.VolatilityDistribution["2-5%"]++
			} else if stat.Volatility < 10 {
				report.VolatilityDistribution["5-10%"]++
			} else {
				report.VolatilityDistribution[">10%"]++
			}

			// 统计高低波动率币种
			if stat.Volatility > 5 {
				report.HighVolatilityCount++
			} else if stat.Volatility < 1 {
				report.LowVolatilityCount++
			}
		}
	}

	if len(volatilities) > 0 {
		report.AverageVolatility = calculateAverage(volatilities)
		sort.Float64s(volatilities)
		report.MedianVolatility = volatilities[len(volatilities)/2]
	}

	return volatilities
}

// analyzeTrends 分析趋势
func (a *MarketEnvironmentAnalyzer) analyzeTrends(stats []SymbolStats, report *MarketEnvironmentReport) {
	for _, stat := range stats {
		// 根据价格变化判断趋势
		if stat.PriceChange > 5 {
			report.BullishSymbols++
			stat.Trend = "bullish"
		} else if stat.PriceChange < -5 {
			report.BearishSymbols++
			stat.Trend = "bearish"
		} else {
			report.OscillatingSymbols++
			stat.Trend = "oscillating"
		}

		// 判断是否有明显趋势
		if math.Abs(stat.PriceChange) > 2 {
			report.TrendingSymbols++
		}
	}
}

// determineMarketRegime 判断市场状态
func (a *MarketEnvironmentAnalyzer) determineMarketRegime(report *MarketEnvironmentReport, volatilities []float64) {
	avgVolatility := report.AverageVolatility
	bullRatio := float64(report.BullishSymbols) / float64(report.TotalSymbols)
	bearRatio := float64(report.BearishSymbols) / float64(report.TotalSymbols)
	trendRatio := float64(report.TrendingSymbols) / float64(report.TotalSymbols)

	// 市场状态判断逻辑
	if avgVolatility < 2.0 && trendRatio < 0.3 {
		report.MarketRegime = "极度低迷 (Deep Freeze)"
		report.RegimeConfidence = 0.9
	} else if avgVolatility < 3.0 && bullRatio < 0.2 && bearRatio < 0.2 {
		report.MarketRegime = "横盘震荡 (Sideways)"
		report.RegimeConfidence = 0.8
	} else if bearRatio > 0.4 && avgVolatility > 4.0 {
		report.MarketRegime = "恐慌下跌 (Panic Selling)"
		report.RegimeConfidence = 0.85
	} else if bullRatio > 0.4 && avgVolatility > 4.0 {
		report.MarketRegime = "强劲上涨 (Strong Bull)"
		report.RegimeConfidence = 0.85
	} else if avgVolatility > 5.0 {
		report.MarketRegime = "高波动 (High Volatility)"
		report.RegimeConfidence = 0.7
	} else {
		report.MarketRegime = "温和调整 (Mild Adjustment)"
		report.RegimeConfidence = 0.6
	}
}

// getTopGainers 获取涨幅榜
func (a *MarketEnvironmentAnalyzer) getTopGainers(stats []SymbolStats, limit int) []SymbolStats {
	// 按涨幅降序排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].PriceChange > stats[j].PriceChange
	})

	if len(stats) > limit {
		return stats[:limit]
	}
	return stats
}

// getTopLosers 获取跌幅榜
func (a *MarketEnvironmentAnalyzer) getTopLosers(stats []SymbolStats, limit int) []SymbolStats {
	// 按跌幅升序排序（最负的在前面）
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].PriceChange < stats[j].PriceChange
	})

	if len(stats) > limit {
		return stats[:limit]
	}
	return stats
}

// calculateAverage 计算平均值
func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// PrintReport 打印分析报告
func (r *MarketEnvironmentReport) PrintReport() {
	fmt.Println("📊 市场环境深度分析报告")
	fmt.Println("═══════════════════════════════════════════════")

	fmt.Printf("⏰ 分析时间范围: 最近%d小时\n", int(r.TimeRange.Hours()))
	fmt.Printf("📈 总计币种数量: %d个\n", r.TotalSymbols)
	fmt.Printf("🔥 活跃币种数量: %d个 (交易量>100万美元)\n", r.ActiveSymbols)
	fmt.Println()

	fmt.Println("🌊 波动率分析")
	fmt.Println("───────────────────────────────")
	fmt.Printf("📊 平均波动率: %.2f%%\n", r.AverageVolatility)
	fmt.Printf("🎯 中位波动率: %.2f%%\n", r.MedianVolatility)
	fmt.Printf("📈 高波动币种: %d个 (>5%%)\n", r.HighVolatilityCount)
	fmt.Printf("📉 低波动币种: %d个 (<1%%)\n", r.LowVolatilityCount)
	fmt.Println()

	fmt.Println("📊 波动率分布")
	fmt.Println("───────────────────────────────")
	for level, count := range r.VolatilityDistribution {
		fmt.Printf("• %s: %d个币种\n", level, count)
	}
	fmt.Println()

	fmt.Println("📈 趋势分析")
	fmt.Println("───────────────────────────────")
	fmt.Printf("🐂 强势上涨: %d个币种 (>+5%%)\n", r.BullishSymbols)
	fmt.Printf("🐻 强势下跌: %d个币种 (<-5%%)\n", r.BearishSymbols)
	fmt.Printf("🔄 横盘震荡: %d个币种 (±5%%以内)\n", r.OscillatingSymbols)
	fmt.Printf("📊 有趋势币种: %d个币种 (>±2%%)\n", r.TrendingSymbols)
	fmt.Println()

	fmt.Printf("🎯 市场状态判断: %s (置信度: %.1f%%)\n", r.MarketRegime, r.RegimeConfidence*100)
	fmt.Println()

	r.printMarketRegimeAnalysis()
	r.printTopMovers()
	r.printStrategyImplications()
}

func (r *MarketEnvironmentReport) printMarketRegimeAnalysis() {
	fmt.Println("🔍 市场状态深度分析")
	fmt.Println("───────────────────────────────")

	switch r.MarketRegime {
	case "极度低迷 (Deep Freeze)":
		fmt.Println("❄️ 当前市场极度低迷，投资者情绪冰冷")
		fmt.Println("📊 特点：极低波动率，几乎没有明确趋势")
		fmt.Println("🎯 原因：投资者观望，缺乏交易热情")
		fmt.Println("⚠️ 影响：所有趋势策略都会表现不佳")

	case "横盘震荡 (Sideways)":
		fmt.Println("🔄 市场处于横盘震荡整理阶段")
		fmt.Println("📊 特点：价格在均线附近窄幅波动")
		fmt.Println("🎯 原因：多空力量平衡，等待新催化剂")
		fmt.Println("⚠️ 影响：均线策略容易产生假信号")

	case "恐慌下跌 (Panic Selling)":
		fmt.Println("📉 市场恐慌性抛售，风险偏好急剧下降")
		fmt.Println("📊 特点：高波动率，大幅下跌")
		fmt.Println("🎯 原因：负面消息或突发事件")
		fmt.Println("⚠️ 影响：适合反转策略，但风险极高")

	case "强劲上涨 (Strong Bull)":
		fmt.Println("🚀 市场强劲上涨，风险偏好回暖")
		fmt.Println("📊 特点：高波动率，大幅上涨")
		fmt.Println("🎯 原因：积极消息或资金涌入")
		fmt.Println("⚠️ 影响：趋势策略表现优秀")

	case "高波动 (High Volatility)":
		fmt.Println("🌊 市场波动剧烈，机会与风险并存")
		fmt.Println("📊 特点：价格大幅波动，成交活跃")
		fmt.Println("🎯 原因：重大事件或消息面影响")
		fmt.Println("⚠️ 影响：日内交易策略更适用")

	case "温和调整 (Mild Adjustment)":
		fmt.Println("📊 市场温和调整，多空分歧不大")
		fmt.Println("📊 特点：适中波动，有一定趋势")
		fmt.Println("🎯 原因：正常的市场调整过程")
		fmt.Println("⚠️ 影响：适合稳健的趋势策略")
	}
	fmt.Println()
}

func (r *MarketEnvironmentReport) printTopMovers() {
	fmt.Println("🏆 涨幅榜 TOP10")
	fmt.Println("───────────────────────────────")
	for i, symbol := range r.TopGainers[:10] {
		fmt.Printf("%2d. %-12s %+7.2f%% (波动率: %.2f%%, 成交量: %.0f万)\n",
			i+1, symbol.Symbol, symbol.PriceChange, symbol.Volatility, symbol.Volume/10000)
	}
	fmt.Println()

	fmt.Println("📉 跌幅榜 TOP10")
	fmt.Println("───────────────────────────────")
	for i, symbol := range r.TopLosers[:10] {
		fmt.Printf("%2d. %-12s %+7.2f%% (波动率: %.2f%%, 成交量: %.0f万)\n",
			i+1, symbol.Symbol, symbol.PriceChange, symbol.Volatility, symbol.Volume/10000)
	}
	fmt.Println()
}

func (r *MarketEnvironmentReport) printStrategyImplications() {
	fmt.Println("🎯 对量化策略的影响分析")
	fmt.Println("───────────────────────────────")

	fmt.Printf("📈 均线策略: ")
	if r.AverageVolatility < 2.0 {
		fmt.Printf("❌ 不适合 - 波动率过低，难以产生有效信号\n")
	} else if r.AverageVolatility < 5.0 {
		fmt.Printf("⚠️ 谨慎使用 - 需要降低阈值，适度放宽条件\n")
	} else {
		fmt.Printf("✅ 适合使用 - 高波动环境利于趋势捕捉\n")
	}

	fmt.Printf("📊 统计套利: ")
	trendRatio := float64(r.TrendingSymbols) / float64(r.TotalSymbols)
	if trendRatio > 0.6 {
		fmt.Printf("✅ 机会较多 - 币种间走势分化明显\n")
	} else if trendRatio > 0.3 {
		fmt.Printf("⚠️ 适度机会 - 存在一定套利空间\n")
	} else {
		fmt.Printf("❌ 机会较少 - 市场同质化严重\n")
	}

	fmt.Printf("🔄 反转策略: ")
	if r.OscillatingSymbols > r.TrendingSymbols {
		fmt.Printf("✅ 适合使用 - 震荡市有利于反转\n")
	} else {
		fmt.Printf("⚠️ 谨慎使用 - 趋势明显时反转风险高\n")
	}

	fmt.Printf("🎪 波动率策略: ")
	if r.HighVolatilityCount > 20 {
		fmt.Printf("✅ 大有可为 - 高波动环境机会多\n")
	} else if r.HighVolatilityCount > 10 {
		fmt.Printf("⚠️ 适度机会 - 部分币种波动较大\n")
	} else {
		fmt.Printf("❌ 不太适合 - 整体波动率偏低\n")
	}

	fmt.Println()
	fmt.Println("💡 策略优化建议:")
	fmt.Printf("• 波动率阈值建议: %.1f%% (当前平均波动率)\n", r.AverageVolatility)
	fmt.Printf("• 趋势强度阈值建议: %.2f%%\n", r.AverageVolatility*0.5)
	if r.AverageVolatility < 2.0 {
		fmt.Println("• 建议大幅降低过滤条件，或暂停均线策略")
		fmt.Println("• 考虑增加反转策略或区间交易策略")
	} else if r.AverageVolatility < 4.0 {
		fmt.Println("• 建议适度降低波动率和质量要求")
		fmt.Println("• 可以考虑结合多个技术指标")
	} else {
		fmt.Println("• 当前环境适合大多数技术策略")
		fmt.Println("• 可以提高信号质量要求")
	}
}

// 辅助函数
func loadConfig(configPath string) (*config.Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var cfg config.Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &cfg, nil
}

func connectDatabase(dbConfig struct {
	DSN          string `yaml:"dsn"`
	Automigrate  bool   `yaml:"automigrate"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}) (pdb.Database, error) {
	options := pdb.Options{
		DSN:          dbConfig.DSN,
		Automigrate:  false,
		MaxOpenConns: dbConfig.MaxOpenConns,
		MaxIdleConns: dbConfig.MaxIdleConns,
	}

	return pdb.OpenMySQL(options)
}
