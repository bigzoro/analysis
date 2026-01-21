package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"analysis/internal/analysis"
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("=== 调试BFUSDUSDT策略扫描问题 ===")

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

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}

	// 3. 创建服务器实例（简化版）
	srv := &server.Server{
		db:  db,
		cfg: cfg,
	}

	// 4. 获取策略22
	var strategy pdb.TradingStrategy
	err = gdb.First(&strategy, 22).Error
	if err != nil {
		log.Fatalf("获取策略22失败: %v", err)
	}

	fmt.Printf("策略ID: %d\n", strategy.ID)
	fmt.Printf("策略名称: %s\n", strategy.Name)
	fmt.Printf("均线启用: %v\n", strategy.Conditions.MovingAverageEnabled)
	fmt.Printf("短期均线周期: %d\n", strategy.Conditions.ShortMAPeriod)
	fmt.Printf("长期均线周期: %d\n", strategy.Conditions.LongMAPeriod)
	fmt.Printf("交叉信号类型: %s\n", strategy.Conditions.MACrossSignal)

	// 5. 检查VolumeBasedSelector是否选择了BFUSDUSDT
	fmt.Println("\n=== 检查VolumeBasedSelector ===")
	candidates, err := selectCandidatesByVolumeDebug(srv, &strategy, 50)
	if err != nil {
		log.Printf("获取候选币种失败: %v", err)
		return
	}

	fmt.Printf("选择了%d个候选币种\n", len(candidates))
	found := false
	for i, symbol := range candidates {
		if symbol == "BFUSDUSDT" {
			fmt.Printf("✅ BFUSDUSDT在候选名单中，排名 #%d\n", i+1)
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ BFUSDUSDT不在候选名单中\n")
		fmt.Printf("前10个候选: %v\n", candidates[:min(10, len(candidates))])

		// 检查BFUSDUSDT的交易量
		var bfusdtVolume struct {
			QuoteVolume float64
			Count       int64
		}
		gdb.Table("binance_24h_stats").
			Select("COALESCE(AVG(quote_volume), 0) as quote_volume, COUNT(*) as count").
			Where("symbol = ? AND market_type = ? AND created_at >= ?", "BFUSDUSDT", "spot", time.Now().Add(-24*time.Hour)).
			Scan(&bfusdtVolume)

		fmt.Printf("BFUSDUSDT 24h平均交易量: %.0f USD, 记录数: %d\n", bfusdtVolume.QuoteVolume, bfusdtVolume.Count)
		if bfusdtVolume.QuoteVolume < 1000000 {
			fmt.Printf("❌ BFUSDUSDT交易量不足100万美元，不符合候选条件\n")
		}
		return
	}

	// 6. 如果BFUSDUSDT在候选名单中，检查均线策略
	fmt.Println("\n=== 检查BFUSDUSDT均线策略 ===")
	maScanner := &server.MovingAverageStrategyScanner{
		server: srv,
	}

	eligibleSymbols, err := maScanner.Scan(context.Background(), &strategy)
	if err != nil {
		log.Printf("扫描失败: %v", err)
		return
	}

	fmt.Printf("扫描完成，发现%d个符合条件的币种\n", len(eligibleSymbols))

	found = false
	for _, eligible := range eligibleSymbols {
		if eligible.Symbol == "BFUSDUSDT" {
			fmt.Printf("✅ BFUSDUSDT符合条件!\n")
			fmt.Printf("   动作: %s\n", eligible.Action)
			fmt.Printf("   原因: %s\n", eligible.Reason)
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ BFUSDUSDT不符合均线策略条件\n")

		// 手动检查BFUSDUSDT的均线情况
		fmt.Println("\n=== 手动检查BFUSDUSDT均线计算 ===")
		checkBFUSDUTMovingAverage(gdb, strategy.Conditions)
	}
}

func checkBFUSDUTMovingAverage(gdb pdb.Database, conditions pdb.StrategyConditions) {
	// 获取BFUSDUSDT的价格数据
	prices, err := getKlinePricesForSymbol(gdb, "BFUSDUSDT", conditions.LongMAPeriod+10)
	if err != nil {
		fmt.Printf("获取BFUSDUSDT价格数据失败: %v\n", err)
		return
	}

	fmt.Printf("BFUSDUSDT价格数据点数: %d\n", len(prices))
	if len(prices) < conditions.LongMAPeriod {
		fmt.Printf("❌ 数据不足，需要至少%d个点，当前%d个\n", conditions.LongMAPeriod, len(prices))
		return
	}

	// 计算均线
	ti := analysis.NewTechnicalIndicators()
	shortMA := ti.CalculateMovingAverage(prices, conditions.ShortMAPeriod, analysis.SMA)
	longMA := ti.CalculateMovingAverage(prices, conditions.LongMAPeriod, analysis.SMA)

	if len(shortMA) == 0 || len(longMA) == 0 {
		fmt.Printf("❌ 均线计算失败\n")
		return
	}

	fmt.Printf("均线计算成功，短期均线长度: %d, 长期均线长度: %d\n", len(shortMA), len(longMA))

	// 检查最新交叉信号
	goldenCross, deathCross := ti.DetectMACross(shortMA, longMA)
	fmt.Printf("金叉信号: %v, 死叉信号: %v\n", goldenCross, deathCross)

	// 显示最新的均线值
	if len(shortMA) > 0 && len(longMA) > 0 {
		lastShort := shortMA[len(shortMA)-1]
		lastLong := longMA[len(longMA)-1]
		fmt.Printf("最新短期均线(SMA%d): %.6f\n", conditions.ShortMAPeriod, lastShort)
		fmt.Printf("最新长期均线(SMA%d): %.6f\n", conditions.LongMAPeriod, lastLong)

		if lastShort > lastLong {
			fmt.Printf("📈 当前趋势: 短期均线在长期均线之上\n")
		} else {
			fmt.Printf("📉 当前趋势: 短期均线在长期均线之下\n")
		}
	}

	// 检查趋势过滤
	if conditions.MATrendFilter {
		uptrend, downtrend := ti.DetectMATrend(shortMA, longMA)
		fmt.Printf("上升趋势: %v, 下降趋势: %v\n", uptrend, downtrend)
		fmt.Printf("趋势方向要求: %s\n", conditions.MATrendDirection)

		trendCheckPassed := true
		switch conditions.MATrendDirection {
		case "UP":
			trendCheckPassed = uptrend
		case "DOWN":
			trendCheckPassed = downtrend
		case "BOTH":
			trendCheckPassed = true
		default:
			trendCheckPassed = true
		}

		if !trendCheckPassed {
			fmt.Printf("❌ 趋势过滤未通过\n")
		} else {
			fmt.Printf("✅ 趋势过滤通过\n")
		}
	}
}

func getKlinePricesForSymbol(gdb pdb.Database, symbol string, limit int) ([]float64, error) {
	var klines []pdb.MarketKline
	err := gdb.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Order("open_time DESC").
		Limit(limit).
		Find(&klines).Error

	if err != nil {
		return nil, err
	}

	// 反转顺序，从旧到新
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	prices := make([]float64, len(klines))
	for i, kline := range klines {
		price, err := strconv.ParseFloat(kline.ClosePrice, 64)
		if err != nil {
			return nil, err
		}
		prices[i] = price
	}

	return prices, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 以下是辅助函数
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

// 按交易量选择候选币种（复制自VolumeBasedSelector）
func selectCandidatesByVolumeDebug(srv *server.Server, strategy *pdb.TradingStrategy, maxCount int) ([]string, error) {
	log.Printf("[VolumeBasedSelector] 基于交易量选择前%d个候选币种", maxCount)

	// 从数据库获取交易量最大的币种
	gdb := srv.DB.DB()

	var volumeStats []struct {
		Symbol      string
		Volume      float64
		QuoteVolume float64
		PriceChange float64
		Count       int64
	}

	// 查询最近24小时的交易统计，从binance_24h_stats表获取数据
	err := gdb.Table("binance_24h_stats").
		Select("symbol, AVG(volume) as volume, AVG(quote_volume) as quote_volume, AVG(price_change_percent) as price_change, COUNT(*) as count").
		Where("market_type = ? AND created_at >= ?", "spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Having("COUNT(*) >= 1").         // 至少有1条记录
		Order("AVG(quote_volume) DESC"). // 按报价交易量排序
		Limit(maxCount * 2).             // 多取一些备用
		Scan(&volumeStats).Error

	if err != nil {
		log.Printf("[VolumeBasedSelector] 查询交易量数据失败: %v，使用涨幅榜降级", err)
		return fallbackToGainersDebug(srv, maxCount)
	}

	// 筛选出有足够交易量的币种
	var candidates []string
	for _, stat := range volumeStats {
		// 对于策略，降低交易量门槛到10万美元
		minVolume := 100000.0 // 10万美元作为最低门槛
		if stat.QuoteVolume > minVolume {
			candidates = append(candidates, stat.Symbol)
			if len(candidates) >= maxCount*2 { // 多取一些用于过滤
				break
			}
		}
	}

	if len(candidates) == 0 {
		log.Printf("[VolumeBasedSelector] 未找到足够交易量的币种(最低%.0f)，使用优化降级", 100000.0)
		return fallbackToVolumeOptimizedDebug(srv, maxCount)
	}

	log.Printf("[VolumeBasedSelector] 初步筛选出%d个高交易量候选币种", len(candidates))

	// 应用过滤器
	originalCount := len(candidates)

	// 1. 过滤稳定币 (如果策略需要)
	if strategy.Conditions.MovingAverageEnabled {
		// 对于均线策略，默认过滤稳定币
		candidates = filterStableCoinsDebug(candidates)
		log.Printf("[VolumeBasedSelector] 过滤稳定币: %d → %d", originalCount, len(candidates))
	}

	// 确保有足够的候选币种
	if len(candidates) < maxCount {
		log.Printf("[VolumeBasedSelector] 过滤后候选不足%d个，使用涨幅榜补充", maxCount)
	}

	// 限制数量
	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}

	showCount := 5
	if len(candidates) < 5 {
		showCount = len(candidates)
	}
	log.Printf("[VolumeBasedSelector] 最终选择了%d个候选币种: %v", len(candidates), candidates[:showCount])
	return candidates, nil
}

// 过滤稳定币
func filterStableCoinsDebug(symbols []string) []string {
	stableCoins := []string{"USDT", "USDC", "BUSD", "DAI", "TUSD", "USDP", "FRAX", "LUSD", "USDN"}
	var filtered []string

	for _, symbol := range symbols {
		isStable := false
		for _, stable := range stableCoins {
			if strings.Contains(symbol, stable) {
				isStable = true
				break
			}
		}
		if !isStable {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// 降级到涨幅榜
func fallbackToGainersDebug(srv *server.Server, maxCount int) ([]string, error) {
	// 直接从 binance_24h_stats 查询涨幅最大的币种
	var results []struct {
		Symbol string
	}

	query := `
		SELECT symbol
		FROM binance_24h_stats
		WHERE market_type = 'futures'
			AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			AND volume > 1000000
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT ?
	`

	err := srv.DB.DB().Raw(query, maxCount).Scan(&results).Error
	if err != nil {
		log.Printf("[VolumeBasedSelector] 从 binance_24h_stats 查询涨幅榜失败: %v", err)
		return []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}, nil
	}

	if len(results) == 0 {
		log.Printf("[VolumeBasedSelector] 未找到有效的涨幅榜数据")
		return []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}, nil
	}

	var candidates []string
	for _, result := range results {
		candidates = append(candidates, result.Symbol)
	}

	log.Printf("[VolumeBasedSelector] 从 binance_24h_stats 选择了 %d 个涨幅榜候选币种", len(candidates))
	return candidates, nil
}

// 优化的交易量降级策略
func fallbackToVolumeOptimizedDebug(srv *server.Server, maxCount int) ([]string, error) {
	log.Printf("[VolumeBasedSelector] 执行优化降级策略")

	// 策略1：查询最近1小时内的所有spot市场数据，不限制交易量
	var results1 []struct {
		Symbol      string
		QuoteVolume float64
	}

	query1 := `
		SELECT symbol, AVG(quote_volume) as quote_volume
		FROM binance_24h_stats
		WHERE market_type = 'spot'
			AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
		GROUP BY symbol
		ORDER BY AVG(quote_volume) DESC
		LIMIT ?
	`

	err1 := srv.DB.DB().Raw(query1, maxCount*2).Scan(&results1).Error
	if err1 == nil && len(results1) > 0 {
		var candidates []string
		for _, result := range results1 {
			candidates = append(candidates, result.Symbol)
			if len(candidates) >= maxCount {
				break
			}
		}
		log.Printf("[VolumeBasedSelector] 优化降级1成功: 找到%d个币种", len(candidates))
		return candidates, nil
	}

	// 策略2：查询所有市场类型的最近数据
	var results2 []struct {
		Symbol string
	}

	query2 := `
		SELECT DISTINCT symbol
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)
		ORDER BY created_at DESC
		LIMIT ?
	`

	err2 := srv.DB.DB().Raw(query2, maxCount*3).Scan(&results2).Error
	if err2 == nil && len(results2) > 0 {
		var candidates []string
		for _, result := range results2 {
			candidates = append(candidates, result.Symbol)
			if len(candidates) >= maxCount {
				break
			}
		}
		log.Printf("[VolumeBasedSelector] 优化降级2成功: 找到%d个币种", len(candidates))
		return candidates, nil
	}

	// 策略3：硬编码主要币种列表
	candidates := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
		"DOTUSDT", "AVAXUSDT", "LINKUSDT", "LTCUSDT", "XRPUSDT",
		"DOGEUSDT", "MATICUSDT", "SHIBUSDT", "UNIUSDT", "ICPUSDT",
		"FILUSDT", "ETCUSDT", "VETUSDT", "TRXUSDT", "THETAUSDT",
		"FTTUSDT", "ALGOUSDT", "ATOMUSDT", "CAKEUSDT", "SUSHIUSDT",
		"COMPUSDT", "MKRUSDT", "AAVEUSDT", "CRVUSDT", "YFIUSDT",
		"BALUSDT", "IMXUSDT", "GRTUSDT", "ACHUSDT", "ROSEUSDT",
		"USTCUSDT", "DATAUSDT", "BIOUSDT", "OMUSDT", "ORDIUSDT",
		"JUPUSDT", "0GUSDT", "PEOPLEUSDT", "WBTCUSDT",
	}

	// 限制数量
	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}

	log.Printf("[VolumeBasedSelector] 优化降级3: 使用预定义币种列表 (%d个)", len(candidates))
	return candidates, nil
}
