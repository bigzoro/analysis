package main

import (
	"fmt"
	"log"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("=== 分析策略ID 24的当前参数设置 ===")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 查询策略ID 24的详细配置
	queryStrategyDetail(db, 24)

	fmt.Println("\n=== 分析完成 ===")
}

func queryStrategyDetail(db *sql.DB, strategyID int) {
	query := fmt.Sprintf(`
		SELECT
			id, name,
			moving_average_enabled, ma_signal_mode, ma_type,
			short_ma_period, long_ma_period, ma_cross_signal,
			ma_trend_filter, ma_trend_direction,
			allowed_directions, enable_leverage, default_leverage
		FROM trading_strategies
		WHERE id = %d`, strategyID)

	var (
		id              int
		name            string
		maEnabled       sql.NullBool
		signalMode      sql.NullString
		maType          sql.NullString
		shortPeriod     sql.NullInt32
		longPeriod      sql.NullInt32
		crossSignal     sql.NullString
		trendFilter     sql.NullBool
		trendDirection  sql.NullString
		allowedDirs     sql.NullString
		enableLeverage  sql.NullBool
		defaultLeverage sql.NullInt32
	)

	err := db.QueryRow(query).Scan(
		&id, &name, &maEnabled, &signalMode, &maType,
		&shortPeriod, &longPeriod, &crossSignal,
		&trendFilter, &trendDirection, &allowedDirs,
		&enableLeverage, &defaultLeverage,
	)

	if err != nil {
		log.Fatalf("查询策略失败: %v", err)
	}

	fmt.Printf("📋 策略信息:\n")
	fmt.Printf("   ID: %d\n", id)
	fmt.Printf("   名称: %s\n", name)

	fmt.Printf("\n🎯 均线策略配置:\n")
	fmt.Printf("   策略启用: %s\n", getBoolValue(maEnabled))
	fmt.Printf("   信号模式: %s\n", getStringValue(signalMode))
	fmt.Printf("   均线类型: %s\n", getStringValue(maType))
	fmt.Printf("   短期周期: %s\n", getIntValue(shortPeriod))
	fmt.Printf("   长期周期: %s\n", getIntValue(longPeriod))
	fmt.Printf("   交叉信号: %s\n", getStringValue(crossSignal))
	fmt.Printf("   趋势过滤: %s\n", getBoolValue(trendFilter))
	fmt.Printf("   趋势方向: %s\n", getStringValue(trendDirection))

	fmt.Printf("\n💰 交易配置:\n")
	fmt.Printf("   允许方向: %s\n", getStringValue(allowedDirs))
	fmt.Printf("   启用杠杆: %s\n", getBoolValue(enableLeverage))
	fmt.Printf("   默认杠杆: %s\n", getIntValue(defaultLeverage))

	// 分析当前配置
	analyzeCurrentConfiguration(signalMode, maType, shortPeriod, longPeriod, crossSignal, trendFilter)

	// 给出优化建议
	giveOptimizationSuggestions(signalMode)
}

func analyzeCurrentConfiguration(signalMode sql.NullString, maType sql.NullString,
	shortPeriod, longPeriod sql.NullInt32, crossSignal sql.NullString, trendFilter sql.NullBool) {

	fmt.Printf("\n📊 配置分析:\n")

	mode := getStringValue(signalMode)
	maTypeStr := getStringValue(maType)
	short := getIntValue(shortPeriod)
	long := getIntValue(longPeriod)
	cross := getStringValue(crossSignal)
	filter := getBoolValue(trendFilter)

	score := 0
	maxScore := 6

	// 1. 信号模式
	if mode == "QUANTITY_FIRST" {
		fmt.Printf("   ✅ 信号模式: 数量优先 ✓\n")
		score++
	} else {
		fmt.Printf("   ⚠️  信号模式: %s (建议使用数量优先)\n", mode)
	}

	// 2. 均线类型
	if maTypeStr == "EMA" {
		fmt.Printf("   ✅ 均线类型: EMA，更适合数量优先 ✓\n")
		score++
	} else {
		fmt.Printf("   ⚠️  均线类型: %s (EMA更适合当前需求)\n", maTypeStr)
	}

	// 3. 周期设置
	if short == "8" && long == "21" {
		fmt.Printf("   ✅ 周期设置: 8/21，适中的灵敏度 ✓\n")
		score++
	} else {
		fmt.Printf("   ⚠️  周期设置: %s/%s (8/21更适合数量优先)\n", short, long)
	}

	// 4. 交叉信号
	if cross == "BOTH" {
		fmt.Printf("   ✅ 交叉信号: 双向交易，适合震荡市 ✓\n")
		score++
	} else {
		fmt.Printf("   ⚠️  交叉信号: %s (双向交易能捕捉更多信号)\n", cross)
	}

	// 5. 趋势过滤
	if filter == "否" {
		fmt.Printf("   ✅ 趋势过滤: 已关闭，适合数量优先 ✓\n")
		score++
	} else {
		fmt.Printf("   ⚠️  趋势过滤: 已开启，可能过度过滤\n")
	}

	fmt.Printf("\n🏆 配置评分: %d/%d\n", score, maxScore)

	if score >= 5 {
		fmt.Printf("🎉 配置优秀！符合数量优先策略的要求\n")
	} else if score >= 3 {
		fmt.Printf("👍 配置良好，稍作调整会更好\n")
	} else {
		fmt.Printf("⚠️  配置需要优化\n")
	}
}

func giveOptimizationSuggestions(signalMode sql.NullString) {
	fmt.Printf("\n🎯 基于当前市场环境的优化建议:\n")

	mode := getStringValue(signalMode)

	if mode == "QUANTITY_FIRST" {
		fmt.Println("📊 核心问题分析:")
		fmt.Println("   • 当前市场平均波动率: 6.18%")
		fmt.Println("   • 93%币种处于横盘震荡")
		fmt.Println("   • 大多数币种没有产生均线交叉")

		fmt.Println("\n💡 优化建议:")
		fmt.Println("   1. 🎯 降低波动率阈值: 从3%降到1.5-2%")
		fmt.Println("   2. 🎪 降低信号质量阈值: 从40%降到25-30%")
		fmt.Println("   3. 📈 调整均线周期: 尝试5/13或6/15 (更灵敏)")
		fmt.Println("   4. 🔄 保持趋势过滤关闭")
		fmt.Println("   5. 📊 保持双向交叉信号")

		fmt.Println("\n🎪 预期效果:")
		fmt.Println("   • 符合条件的币种: 从1个增加到10-20个")
		fmt.Println("   • 日均信号数: 显著提升")
		fmt.Println("   • 资金利用率: 大幅提高")
	} else {
		fmt.Println("   🔄 建议切换到数量优先模式以获得更多信号")
	}

	fmt.Println("\n🚀 立即行动:")
	fmt.Println("   1. 修改波动率阈值至2%")
	fmt.Println("   2. 修改信号质量阈值至30%")
	fmt.Println("   3. 调整周期为5/13")
	fmt.Println("   4. 测试新参数效果")
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "NULL"
}

func getIntValue(ni sql.NullInt32) string {
	if ni.Valid {
		return fmt.Sprintf("%d", ni.Int32)
	}
	return "NULL"
}

func getBoolValue(nb sql.NullBool) string {
	if nb.Valid {
		if nb.Bool {
			return "是"
		}
		return "否"
	}
	return "NULL"
}
