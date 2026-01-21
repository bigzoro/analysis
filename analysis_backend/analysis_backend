package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"analysis/internal/analysis"
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

// 基于8天数据计算盈利情况
func main() {
	fmt.Println("=== 基于8天数据计算均线策略盈利情况 ===")

	// 1. 读取配置文件
	cfg, err := loadConfig("analysis_backend/config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 2. 连接数据库
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	fmt.Printf("✅ 数据库连接成功\n")

	// 3. 获取策略22的配置
	strategy, err := getStrategyByID(db, 22)
	if err != nil {
		log.Fatalf("获取策略22失败: %v", err)
	}

	// 4. 计算8天的盈利情况
	calculate8DayProfit(db, strategy)
}

func calculate8DayProfit(db pdb.Database, strategy *pdb.TradingStrategy) {
	gdb := db.GormDB()

	// 获取可用的交易对
	var symbols []string
	err := gdb.Model(&pdb.MarketKline{}).
		Where("kind = ? AND `interval` = ?", "spot", "1h").
		Distinct("symbol").
		Order("symbol ASC").
		Limit(10). // 只取前10个作为样本
		Pluck("symbol", &symbols).Error

	if err != nil {
		log.Printf("获取交易对列表失败: %v", err)
		return
	}

	fmt.Printf("🎯 使用前%d个交易对进行8天盈利计算\n", len(symbols))
	fmt.Printf("📊 策略: %s (SMA %d-%d)\n",
		strategy.Name, strategy.Conditions.ShortMAPeriod, strategy.Conditions.LongMAPeriod)
	fmt.Printf("💰 初始本金: $10,000\n")
	fmt.Printf("⏰ 测试周期: 8天\n\n")

	totalInitialCapital := 10000.0
	totalFinalCapital := 10000.0
	totalTrades := 0
	winningTrades := 0

	for _, symbol := range symbols {
		fmt.Printf("🔬 计算 %s 的8天盈利:\n", symbol)

		// 获取最近8天的数据（192小时 = 8天 * 24小时）
		capital, trades, wins := calculateSymbolProfit(gdb, strategy, symbol, 192)
		totalFinalCapital += (capital - 10000.0) // 每个交易对独立计算
		totalTrades += trades
		winningTrades += wins

		fmt.Printf("  💰 盈利: $%.2f\n", capital-10000.0)
		fmt.Printf("  📊 交易次数: %d\n", trades)
		fmt.Printf("  🏆 胜率: %.1f%%\n", float64(wins)/float64(trades)*100)
		fmt.Printf("  💵 最终金额: $%.2f\n\n", capital)
	}

	// 计算总体统计
	totalProfit := totalFinalCapital - totalInitialCapital
	totalReturn := (totalProfit / totalInitialCapital) * 100
	winRate := float64(winningTrades) / float64(totalTrades) * 100

	fmt.Printf("🏆 总体8天盈利汇总:\n")
	fmt.Printf("💰 总盈利: $%.2f\n", totalProfit)
	fmt.Printf("📈 总收益率: %.2f%%\n", totalReturn)
	fmt.Printf("🎯 总交易次数: %d\n", totalTrades)
	fmt.Printf("🏆 总体胜率: %.1f%%\n", winRate)
	fmt.Printf("💵 最终本金: $%.2f\n", totalFinalCapital)

	// 按天计算
	dailyReturn := totalReturn / 8.0
	dailyProfit := totalProfit / 8.0

	fmt.Printf("\n📅 按天平均:\n")
	fmt.Printf("💰 日均盈利: $%.2f\n", dailyProfit)
	fmt.Printf("📈 日均收益率: %.2f%%\n", dailyReturn)

	// 月化收益估算（按30天计算）
	monthlyReturn := dailyReturn * 30
	monthlyProfit := dailyProfit * 30

	fmt.Printf("\n📊 月化收益估算 (30天):\n")
	fmt.Printf("💰 月均盈利: $%.2f\n", monthlyProfit)
	fmt.Printf("📈 月化收益率: %.2f%%\n", monthlyReturn)

	// 风险评估
	fmt.Printf("\n⚠️  风险提示:\n")
	fmt.Printf("📝 这只是基于历史数据的回测结果\n")
	fmt.Printf("📝 实际交易会受到 slippage、交易费用等影响\n")
	fmt.Printf("📝 建议在实盘前进行更全面的压力测试\n")
}

func calculateSymbolProfit(gdb *gorm.DB, strategy *pdb.TradingStrategy, symbol string, hours int) (float64, int, int) {
	// 获取指定小时数的数据
	var klines []pdb.MarketKline
	err := gdb.(*gorm.DB).Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Order("open_time DESC").
		Limit(hours).
		Find(&klines).Error

	if err != nil || len(klines) < strategy.Conditions.LongMAPeriod+10 {
		return 10000.0, 0, 0 // 返回初始本金
	}

	// 反转顺序，从旧到新
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	// 初始化技术指标
	ti := analysis.NewTechnicalIndicators()

	// 计算均线
	maType := analysis.SMA
	if strategy.Conditions.MAType == "EMA" {
		maType = analysis.EMA
	}

	prices := make([]float64, len(klines))
	timestamps := make([]time.Time, len(klines))

	for i, kline := range klines {
		price, _ := strconv.ParseFloat(kline.ClosePrice, 64)
		prices[i] = price
		timestamps[i] = kline.OpenTime
	}

	shortMA := ti.CalculateMovingAverage(prices, strategy.Conditions.ShortMAPeriod, maType)
	longMA := ti.CalculateMovingAverage(prices, strategy.Conditions.LongMAPeriod, maType)

	if len(shortMA) == 0 || len(longMA) == 0 {
		return 10000.0, 0, 0
	}

	// 模拟交易
	capital := 10000.0
	position := 0.0
	entryPrice := 0.0
	trades := 0
	wins := 0

	// 使用较短的数组长度
	maxLength := len(shortMA)
	if len(longMA) < maxLength {
		maxLength = len(longMA)
	}
	if len(prices) < maxLength {
		maxLength = len(prices)
	}

	for i := strategy.Conditions.LongMAPeriod; i < maxLength; i++ {
		currentPrice := prices[i]
		shortValue := shortMA[i]
		longValue := longMA[i]

		// 检测交叉信号
		goldenCross := false
		deathCross := false

		if i > 0 {
			prevShort := shortMA[i-1]
			prevLong := longMA[i-1]

			// 金叉
			if prevShort <= prevLong && shortValue > longValue {
				goldenCross = true
			}
			// 死叉
			if prevShort >= prevLong && shortValue < longValue {
				deathCross = true
			}
		}

		// 止损止盈检查
		stopLossTriggered := false
		takeProfitTriggered := false

		if position != 0 {
			priceChange := (currentPrice - entryPrice) / entryPrice * 100

			if position > 0 {
				if strategy.Conditions.EnableStopLoss && priceChange <= -strategy.Conditions.StopLossPercent {
					stopLossTriggered = true
				}
				if strategy.Conditions.EnableTakeProfit && priceChange >= strategy.Conditions.TakeProfitPercent {
					takeProfitTriggered = true
				}
			} else {
				if strategy.Conditions.EnableStopLoss && -priceChange <= -strategy.Conditions.StopLossPercent {
					stopLossTriggered = true
				}
				if strategy.Conditions.EnableTakeProfit && -priceChange >= strategy.Conditions.TakeProfitPercent {
					takeProfitTriggered = true
				}
			}
		}

		// 交易逻辑
		if position == 0 {
			positionSize := strategy.Conditions.MaxPositionSize / 100.0

			if goldenCross && (strategy.Conditions.AllowedDirections == "LONG" || strategy.Conditions.AllowedDirections == "LONG,SHORT") {
				position = positionSize
				entryPrice = currentPrice
				trades++
			} else if deathCross && (strategy.Conditions.AllowedDirections == "SHORT" || strategy.Conditions.AllowedDirections == "LONG,SHORT") {
				position = -positionSize
				entryPrice = currentPrice
				trades++
			}
		} else {
			shouldClose := stopLossTriggered || takeProfitTriggered ||
				(position > 0 && deathCross) || (position < 0 && goldenCross)

			if shouldClose {
				exitPrice := currentPrice
				quantity := position
				if position < 0 {
					quantity = -position
				}

				// 计算盈亏
				var pnl float64
				if position > 0 {
					pnl = (exitPrice - entryPrice) / entryPrice * quantity * capital
				} else {
					pnl = (entryPrice - exitPrice) / entryPrice * quantity * capital
				}

				capital += pnl

				if pnl > 0 {
					wins++
				}

				position = 0
				entryPrice = 0
			}
		}
	}

	return capital, trades, wins
}

// 其他辅助函数（复用之前的代码）
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

func getStrategyByID(db pdb.Database, strategyID int) (*pdb.TradingStrategy, error) {
	gdb := db.GormDB()

	var strategy pdb.TradingStrategy
	err := gdb.First(&strategy, strategyID).Error
	if err != nil {
		return nil, fmt.Errorf("查询策略失败: %v", err)
	}

	return &strategy, nil
}
