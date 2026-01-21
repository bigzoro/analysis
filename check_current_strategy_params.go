package main

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"database/sql"
)

func main() {
	fmt.Println("=== 查询策略ID 23的当前参数设置 ===")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 查询策略参数
	query := `
		SELECT
			id, name,
			JSON_EXTRACT(conditions, '$.moving_average_enabled') as ma_enabled,
			JSON_EXTRACT(conditions, '$.ma_signal_mode') as signal_mode,
			JSON_EXTRACT(conditions, '$.ma_type') as ma_type,
			JSON_EXTRACT(conditions, '$.short_ma_period') as short_period,
			JSON_EXTRACT(conditions, '$.long_ma_period') as long_period,
			JSON_EXTRACT(conditions, '$.ma_cross_signal') as cross_signal,
			JSON_EXTRACT(conditions, '$.ma_trend_filter') as trend_filter,
			JSON_EXTRACT(conditions, '$.ma_trend_direction') as trend_direction
		FROM trading_strategies
		WHERE id = 23`

	var (
		id              int
		name            string
		maEnabled       sql.NullString
		signalMode      sql.NullString
		maType          sql.NullString
		shortPeriod     sql.NullString
		longPeriod      sql.NullString
		crossSignal     sql.NullString
		trendFilter     sql.NullString
		trendDirection  sql.NullString
	)

	err = db.QueryRow(query).Scan(
		&id, &name, &maEnabled, &signalMode, &maType,
		&shortPeriod, &longPeriod, &crossSignal,
		&trendFilter, &trendDirection,
	)

	if err != nil {
		log.Fatal("查询策略失败:", err)
	}

	fmt.Printf("📋 策略信息:\n")
	fmt.Printf("   ID: %d\n", id)
	fmt.Printf("   名称: %s\n", name)

	fmt.Printf("\n🎯 当前均线策略参数:\n")
	fmt.Printf("   策略启用: %s\n", getBoolValue(maEnabled))
	fmt.Printf("   信号模式: %s\n", getStringValue(signalMode))
	fmt.Printf("   均线类型: %s\n", getStringValue(maType))
	fmt.Printf("   短期周期: %s\n", getStringValue(shortPeriod))
	fmt.Printf("   长期周期: %s\n", getStringValue(longPeriod))
	fmt.Printf("   交叉信号: %s\n", getStringValue(crossSignal))
	fmt.Printf("   趋势过滤: %s\n", getBoolValue(trendFilter))
	fmt.Printf("   趋势方向: %s\n", getStringValue(trendDirection))

	// 显示当前验证阈值
	fmt.Printf("\n🔍 当前验证阈值:\n")
	currentMode := getStringValue(signalMode)
	showCurrentThresholds(currentMode)

	// 分析参数合理性
	fmt.Printf("\n📊 参数合理性分析:\n")
	analyzeParameterReasonableness(
		getStringValue(signalMode),
		getStringValue(maType),
		getStringValue(shortPeriod),
		getStringValue(longPeriod),
		getStringValue(crossSignal),
		getBoolValue(trendFilter),
	)

	fmt.Printf("\n💡 优化建议:\n")
	showOptimizationSuggestions(currentMode)

	fmt.Println("\n=== 参数查询完成 ===")
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "未设置"
}

func getBoolValue(ns sql.NullString) string {
	if ns.Valid && ns.String == "true" {
		return "是"
	}
	return "否"
}

func showCurrentThresholds(mode string) {
	fmt.Printf("   信号模式: %s\n", mode)

	switch mode {
	case "QUALITY_FIRST":
		fmt.Println("   波动率阈值: ≥8.00%")
		fmt.Println("   趋势强度阈值: ≥0.200%")
		fmt.Println("   信号质量阈值: ≥70%")
		fmt.Println("   严格模式: 是")
	case "QUANTITY_FIRST":
		fmt.Println("   波动率阈值: ≥3.00%")
		fmt.Println("   趋势强度阈值: ≥0.050%")
		fmt.Println("   信号质量阈值: ≥40%")
		fmt.Println("   严格模式: 否")
	default:
		fmt.Println("   波动率阈值: ≥5.00% (默认)")
		fmt.Println("   趋势强度阈值: ≥0.100% (默认)")
		fmt.Println("   信号质量阈值: ≥50% (默认)")
		fmt.Println("   严格模式: 否 (默认)")
	}
}

func analyzeParameterReasonableness(mode, maType, shortPeriod, longPeriod, crossSignal, trendFilter string) {
	score := 0
	maxScore := 6

	// 1. 信号模式分析
	if mode == "QUANTITY_FIRST" {
		fmt.Printf("   ✅ 信号模式: 选择了数量优先，适合当前需求\n")
		score++
	} else {
		fmt.Printf("   ⚠️  信号模式: %s，可能过于严格\n", mode)
	}

	// 2. 均线类型分析
	if maType == "EMA" {
		fmt.Printf("   ✅ 均线类型: EMA，更适合数量优先策略\n")
		score++
	} else if maType == "SMA" {
		fmt.Printf("   ⚠️  均线类型: SMA，相对不敏感，可能错过信号\n")
	} else {
		fmt.Printf("   ❌ 均线类型: %s，无效设置\n", maType)
	}

	// 3. 周期设置分析
	if shortPeriod == "8" && longPeriod == "21" {
		fmt.Printf("   ✅ 周期设置: 8/21，适中的灵敏度\n")
		score++
	} else if shortPeriod == "5" && longPeriod == "20" {
		fmt.Printf("   ⚠️  周期设置: 5/20，相对保守\n")
		score++
	} else {
		fmt.Printf("   ⚠️  周期设置: %s/%s，需要评估\n", shortPeriod, longPeriod)
	}

	// 4. 交叉信号分析
	if crossSignal == "BOTH" {
		fmt.Printf("   ✅ 交叉信号: 双向交易，适合震荡市\n")
		score++
	} else {
		fmt.Printf("   ⚠️  交叉信号: %s，限制了交易机会\n", crossSignal)
	}

	// 5. 趋势过滤分析
	if trendFilter == "否" {
		fmt.Printf("   ✅ 趋势过滤: 已关闭，适合数量优先\n")
		score++
	} else {
		fmt.Printf("   ⚠️  趋势过滤: 已开启，可能过度过滤\n")
	}

	// 6. 综合评分
	fmt.Printf("\n🏆 参数合理性评分: %d/%d\n", score, maxScore)
	if score >= 5 {
		fmt.Printf("🎉 参数设置优秀！\n")
	} else if score >= 3 {
		fmt.Printf("👍 参数设置良好，有优化空间\n")
	} else {
		fmt.Printf("⚠️  参数设置需要调整\n")
	}
}

func showOptimizationSuggestions(mode string) {
	fmt.Println("🎯 基于当前市场环境的优化建议:")

	if mode == "QUANTITY_FIRST" {
		fmt.Println("1. 📊 降低波动率阈值: 从3%降到1.5-2%")
		fmt.Println("2. 🎯 降低信号质量阈值: 从40%降到25-30%")
		fmt.Println("3. 📈 调整均线周期: 考虑5/13或8/21，更灵敏")
		fmt.Println("4. 🔄 确认趋势过滤: 保持关闭状态")
		fmt.Println("5. 📊 优化交叉信号: 保持双向交易")
	} else {
		fmt.Println("1. 🔄 切换到数量优先模式")
		fmt.Println("2. 📈 使用EMA而非SMA")
		fmt.Println("3. 📊 调整周期为8/21")
		fmt.Println("4. 🔄 关闭趋势过滤")
	}

	fmt.Println("\n💡 关键调整:")
	fmt.Println("- 当前市场平均波动率6.18%，3%阈值过高")
	fmt.Println("- 93%币种处于震荡状态，交叉信号难产生")
	fmt.Println("- 信号质量40%阈值在震荡市过于严格")
}
