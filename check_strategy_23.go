package main

import (
	"fmt"
	"log"
	"strconv"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	fmt.Println("=== 分析策略ID 23 配置 ===")

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

	// 3. 查询策略ID 23
	strategy, err := getStrategyByID(gormDB, 23)
	if err != nil {
		log.Fatalf("查询策略失败: %v", err)
	}

	// 4. 分析策略配置
	analyzeStrategyConfiguration(strategy)

	fmt.Println("\n=== 分析完成 ===")
}

func getStrategyByID(gormDB *gorm.DB, id uint) (*pdb.TradingStrategy, error) {
	var strategy pdb.TradingStrategy
	err := gormDB.Preload("Conditions").Where("id = ?", id).First(&strategy).Error
	if err != nil {
		return nil, err
	}
	return &strategy, nil
}

func analyzeStrategyConfiguration(strategy *pdb.TradingStrategy) {
	fmt.Printf("📋 策略基本信息:\n")
	fmt.Printf("   ID: %d\n", strategy.ID)
	fmt.Printf("   名称: %s\n", strategy.Name)
	fmt.Printf("   状态: %s\n", strategy.Status)

	fmt.Printf("\n🎯 均线策略配置:\n")
	conditions := strategy.Conditions

	if !conditions.MovingAverageEnabled {
		fmt.Printf("   ❌ 均线策略未启用\n")
		return
	}

	fmt.Printf("   ✅ 均线策略已启用\n")
	fmt.Printf("   信号模式: %s\n", conditions.MASignalMode)

	// 分析信号模式
	analyzeSignalMode(conditions.MASignalMode)

	fmt.Printf("   均线类型: %s\n", conditions.MAType)
	fmt.Printf("   周期: %d/%d\n", conditions.ShortMAPeriod, conditions.LongMAPeriod)
	fmt.Printf("   交叉信号: %s\n", conditions.MACrossSignal)
	fmt.Printf("   趋势过滤: %v\n", conditions.MATrendFilter)
	if conditions.MATrendFilter {
		fmt.Printf("   趋势方向: %s\n", conditions.MATrendDirection)
	}

	// 验证配置是否符合预期
	validateConfiguration(conditions)

	// 分析验证阈值
	fmt.Printf("\n🔍 验证阈值分析:\n")
	thresholds := getMAValidationThresholds(conditions.MASignalMode)
	fmt.Printf("   波动率阈值: ≥%.2f%%\n", thresholds.MinVolatility*100)
	fmt.Printf("   趋势强度阈值: ≥%.4f\n", thresholds.MinTrendStrength)
	fmt.Printf("   信号质量阈值: ≥%.1f\n", thresholds.MinSignalQuality)
	fmt.Printf("   严格模式: %v\n", thresholds.StrictMode)

	// 给出总体评价
	fmt.Printf("\n📊 配置评价:\n")
	giveOverallAssessment(conditions)
}

func analyzeSignalMode(mode string) {
	switch mode {
	case "QUALITY_FIRST":
		fmt.Printf("   📋 模式说明: 质量优先 - 高品质、低数量\n")
		fmt.Printf("   🎯 适合: 保守投资者，重视信号质量\n")
	case "QUANTITY_FIRST":
		fmt.Printf("   📋 模式说明: 数量优先 - 中等品质、高数量\n")
		fmt.Printf("   🎯 适合: 活跃交易者，追求资金效率\n")
	default:
		fmt.Printf("   📋 模式说明: 默认平衡模式\n")
		fmt.Printf("   ⚠️  注意: 使用了默认设置\n")
	}
}

func validateConfiguration(conditions pdb.StrategyConditions) {
	fmt.Printf("\n✅ 配置验证:\n")

	// 检查均线参数
	if conditions.ShortMAPeriod >= conditions.LongMAPeriod {
		fmt.Printf("   ❌ 均线参数错误: 短期周期(%d)不应大于等于长期周期(%d)\n",
			conditions.ShortMAPeriod, conditions.LongMAPeriod)
	} else {
		fmt.Printf("   ✅ 均线参数合理: %d日短期线 vs %d日长期线\n",
			conditions.ShortMAPeriod, conditions.LongMAPeriod)
	}

	// 检查交叉信号类型
	validSignals := []string{"GOLDEN_CROSS", "DEATH_CROSS", "BOTH"}
	isValidSignal := false
	for _, signal := range validSignals {
		if conditions.MACrossSignal == signal {
			isValidSignal = true
			break
		}
	}
	if isValidSignal {
		fmt.Printf("   ✅ 交叉信号类型有效: %s\n", conditions.MACrossSignal)
	} else {
		fmt.Printf("   ❌ 交叉信号类型无效: %s\n", conditions.MACrossSignal)
	}

	// 检查信号模式
	validModes := []string{"QUALITY_FIRST", "QUANTITY_FIRST"}
	isValidMode := false
	for _, mode := range validModes {
		if conditions.MASignalMode == mode {
			isValidMode = true
			break
		}
	}
	if isValidMode {
		fmt.Printf("   ✅ 信号模式有效: %s\n", conditions.MASignalMode)
	} else {
		fmt.Printf("   ⚠️  信号模式为空或无效，使用默认模式\n")
	}
}

type MAValidationThresholds struct {
	MinVolatility    float64
	MinTrendStrength float64
	MinSignalQuality float64
	StrictMode       bool
}

func getMAValidationThresholds(signalMode string) MAValidationThresholds {
	switch signalMode {
	case "QUALITY_FIRST":
		return MAValidationThresholds{
			MinVolatility:    0.08,
			MinTrendStrength: 0.002,
			MinSignalQuality: 0.7,
			StrictMode:       true,
		}
	case "QUANTITY_FIRST":
		return MAValidationThresholds{
			MinVolatility:    0.03,
			MinTrendStrength: 0.0005,
			MinSignalQuality: 0.4,
			StrictMode:       false,
		}
	default:
		return MAValidationThresholds{
			MinVolatility:    0.05,
			MinTrendStrength: 0.001,
			MinSignalQuality: 0.5,
			StrictMode:       false,
		}
	}
}

func giveOverallAssessment(conditions pdb.StrategyConditions) {
	score := 0
	maxScore := 5

	// 1. 信号模式配置
	if conditions.MASignalMode == "QUANTITY_FIRST" {
		score++
		fmt.Printf("   ✅ 信号模式正确: 选择了数量优先模式\n")
	} else if conditions.MASignalMode == "QUALITY_FIRST" {
		fmt.Printf("   ⚠️  信号模式: 选择了质量优先模式 (不是数量优先)\n")
	} else {
		fmt.Printf("   ⚠️  信号模式: 使用默认模式\n")
	}

	// 2. 均线参数合理性
	if conditions.ShortMAPeriod < conditions.LongMAPeriod {
		score++
		fmt.Printf("   ✅ 均线参数合理\n")
	} else {
		fmt.Printf("   ❌ 均线参数不合理\n")
	}

	// 3. 交叉信号配置
	if conditions.MACrossSignal == "BOTH" {
		score++
		fmt.Printf("   ✅ 交叉信号配置合适 (双向交易)\n")
	} else {
		fmt.Printf("   ⚠️  交叉信号配置: 单向信号可能减少交易机会\n")
	}

	// 4. 趋势过滤设置
	if !conditions.MATrendFilter {
		score++
		fmt.Printf("   ✅ 趋势过滤关闭: 适合数量优先策略\n")
	} else {
		fmt.Printf("   ⚠️  趋势过滤开启: 可能进一步减少信号数量\n")
	}

	// 5. 总体评价
	fmt.Printf("\n🏆 总体评分: %d/%d\n", score, maxScore)

	if score >= 4 {
		fmt.Printf("🎉 策略配置优秀！完全符合数量优先模式的要求\n")
	} else if score >= 3 {
		fmt.Printf("👍 策略配置良好，但还有优化空间\n")
	} else {
		fmt.Printf("⚠️  策略配置需要调整\n")
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
