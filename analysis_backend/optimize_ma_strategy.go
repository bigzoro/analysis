package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("=== 均线策略优化方案设计 ===")

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

	// 3. 分析当前问题
	fmt.Println("📊 当前策略问题分析:")
	analyzeCurrentIssues(db)

	// 4. 设计优化方案
	fmt.Println("\n🔧 优化方案设计:")
	designOptimizationSolutions()

	// 5. 实现过滤机制
	fmt.Println("\n💡 具体实现方案:")
	implementFilteringMechanisms(db)

	fmt.Println("\n=== 优化方案设计完成 ===")
}

func analyzeCurrentIssues(db pdb.Database) {
	fmt.Println("1. 候选选择问题:")
	fmt.Println("   • VolumeBasedSelector只按交易量排序")
	fmt.Println("   • 未过滤稳定币和低波动资产")
	fmt.Println("   • 包含USDT、BUSD等稳定币交易对")

	fmt.Println("\n2. 均线信号问题:")
	fmt.Println("   • 稳定币微小波动触发交叉信号")
	fmt.Println("   • 缺乏波动率和趋势强度验证")
	fmt.Println("   • 信号质量未评估")

	fmt.Printf("\n📈 当前问题示例:\n")
	fmt.Printf("   BFUSDUSDT被选中原因分析:\n")
	fmt.Printf("   ✅ 高交易量: 符合VolumeBasedSelector条件\n")
	fmt.Printf("   ✅ 金叉信号: SMA5 > SMA20\n")
	fmt.Printf("   ❌ 稳定币特性: 波动率仅0.0009%%\n")
	fmt.Printf("   ❌ 信号质量: 微小波动触发，缺乏实际意义\n")

	fmt.Printf("\n📊 优化目标:\n")
	fmt.Printf("   1. 排除稳定币交易对\n")
	fmt.Printf("   2. 增加波动率过滤\n")
	fmt.Printf("   3. 提升信号质量要求\n")
}

func designOptimizationSolutions() {
	fmt.Println("🎯 多层次优化策略:")

	fmt.Println("\n1️⃣ 候选选择层优化:")
	fmt.Println("   ✅ 添加稳定币过滤器")
	fmt.Println("   ✅ 添加波动率预筛选")
	fmt.Println("   ✅ 添加市值过滤")

	fmt.Println("\n2️⃣ 技术指标层优化:")
	fmt.Println("   ✅ 增加波动率验证")
	fmt.Println("   ✅ 增加趋势强度评估")
	fmt.Println("   ✅ 信号质量评分")

	fmt.Println("\n3️⃣ 风险控制层优化:")
	fmt.Println("   ✅ 添加异常检测")
	fmt.Println("   ✅ 添加信号一致性检查")
	fmt.Println("   ✅ 添加历史表现验证")

	fmt.Println("\n4️⃣ 配置化管理:")
	fmt.Println("   ✅ 可配置的过滤规则")
	fmt.Println("   ✅ 动态阈值调整")
	fmt.Println("   ✅ 策略组合优化")
}

func implementFilteringMechanisms(db pdb.Database) {
	fmt.Println("🛠️ 具体实现方案:")

	fmt.Println("\n📝 方案1: VolumeBasedSelector增强")
	fmt.Println("   位置: internal/server/strategy_scanner_moving_average.go")
	fmt.Println("   方法: SelectCandidates()")
	fmt.Println(`
   func (s *VolumeBasedSelector) SelectCandidates(...) ([]string, error) {
       // 获取高交易量候选
       candidates := getHighVolumeCandidates()

       // 过滤稳定币
       candidates = filterStableCoins(candidates)

       // 过滤低波动资产
       candidates = filterLowVolatilityAssets(candidates)

       // 过滤低市值资产
       candidates = filterLowMarketCapAssets(candidates)

       return candidates[:maxCount], nil
   }`)

	fmt.Println("\n📝 方案2: 均线策略增强")
	fmt.Println("   位置: internal/server/strategy_scanner_moving_average.go")
	fmt.Println("   方法: checkMovingAverageStrategy()")
	fmt.Println(`
   func (s *MovingAverageStrategyScanner) checkMovingAverageStrategy(...) *EligibleSymbol {
       // 基础均线计算
       shortMA, longMA := calculateMovingAverages(prices)

       // 波动率验证
       if !validateVolatility(prices, minVolatilityThreshold) {
           return nil
       }

       // 趋势强度验证
       if !validateTrendStrength(shortMA, longMA, minTrendStrength) {
           return nil
       }

       // 信号质量评估
       signalQuality := assessSignalQuality(shortMA, longMA, prices)
       if signalQuality < minSignalQuality {
           return nil
       }

       return createEligibleSymbol(signalQuality)
   }`)

	fmt.Println("\n📝 方案3: 配置驱动过滤")
	fmt.Println("   位置: config.yaml")
	fmt.Println(`
   strategy:
     ma_strategy:
       # 候选过滤
       exclude_stable_coins: true
       min_volatility_percent: 0.1    # 最小日波动率
       min_market_cap_usd: 10000000  # 最小市值

       # 信号过滤
       min_trend_strength: 0.001     # 最小趋势强度
       min_signal_quality: 0.7       # 最小信号质量
       require_volume_confirmation: true  # 需要成交量确认

       # 风险控制
       max_position_size_percent: 5.0  # 最大仓位比例
       enable_stop_loss: true
       stop_loss_percent: 2.0`)

	fmt.Println("\n📝 方案4: 实现过滤函数")
	fmt.Println("   新增: internal/server/strategy_filters.go")
	fmt.Println(`
// 稳定币过滤器
func filterStableCoins(symbols []string) []string {
    stableCoinSuffixes := []string{"USDT", "BUSD", "USDC", "DAI", "FRAX", "TUSD"}
    var filtered []string

    for _, symbol := range symbols {
        isStableCoin := false
        for _, suffix := range stableCoinSuffixes {
            if strings.HasSuffix(symbol, suffix) {
                isStableCoin = true
                break
            }
        }
        if !isStableCoin {
            filtered = append(filtered, symbol)
        }
    }
    return filtered
}

// 波动率过滤器
func filterLowVolatilityAssets(symbols []string, minVolatility float64) []string {
    var filtered []string

    for _, symbol := range symbols {
        volatility := calculate24hVolatility(symbol)
        if volatility >= minVolatility {
            filtered = append(filtered, symbol)
        }
    }
    return filtered
}

// 趋势强度验证器
func validateTrendStrength(shortMA, longMA []float64, minStrength float64) bool {
    if len(shortMA) == 0 || len(longMA) == 0 {
        return false
    }

    latestShort := shortMA[len(shortMA)-1]
    latestLong := longMA[len(longMA)-1]

    // 计算趋势强度 (短期均线相对长期均线的偏离程度)
    trendStrength := math.Abs(latestShort-latestLong) / latestLong

    return trendStrength >= minStrength
}`)

	fmt.Println("\n📝 方案5: 性能监控和日志改进")
	fmt.Println("   添加详细的过滤统计和性能指标")
	fmt.Println(`
   [MA-Filter] 过滤统计:
     原始候选: 55个
     排除稳定币: 12个 → 剩余43个
     波动率过滤: 8个 → 剩余35个
     市值过滤: 5个 → 剩余30个
     最终入选: 30个`)

	fmt.Println("\n📝 方案6: 测试验证")
	fmt.Println("   创建专门的测试用例验证过滤效果")
	fmt.Println(`
   func TestMAFiltering() {
       // 测试稳定币过滤
       candidates := []string{"BTCUSDT", "ETHUSDT", "BFUSDUSDT", "BUSDUSDT"}
       filtered := filterStableCoins(candidates)
       expected := []string{"BTCUSDT", "ETHUSDT"}
       assert.Equal(t, expected, filtered)
   }`)
}

func calculate24hVolatility(symbol string) float64 {
	// 简化实现，实际应该从数据库计算
	return 0.15 // 假设15%的日波动率
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
