package main

import (
	"fmt"
	"log"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("=== 分析策略ID 25的当前参数设置 ===")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 查询策略ID 25的详细配置
	queryStrategyDetail(db, 25)

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
	analyzeMeanReversionConfiguration(signalMode, maType, shortPeriod, longPeriod, crossSignal, trendFilter, maEnabled)
}

func analyzeMeanReversionConfiguration(signalMode sql.NullString, maType sql.NullString,
	shortPeriod, longPeriod sql.NullInt32, crossSignal sql.NullString, trendFilter sql.NullBool, maEnabled sql.NullBool) {

	fmt.Printf("\n📊 均值回归策略配置分析:\n")

	mode := getStringValue(signalMode)
	maTypeStr := getStringValue(maType)
	short := getIntValue(shortPeriod)
	long := getIntValue(longPeriod)
	cross := getStringValue(crossSignal)
	filter := getBoolValue(trendFilter)
	enabled := getBoolValue(maEnabled)

	score := 0
	maxScore := 7

	// 1. 均线策略启用状态
	if enabled == "否" {
		fmt.Printf("   ✅ 均线策略: 已禁用 (适合均值回归策略)\n")
		score++
	} else {
		fmt.Printf("   ⚠️  均线策略: 已启用 (均值回归策略通常不需要均线)\n")
	}

	// 2. 信号模式
	if mode == "QUALITY_FIRST" {
		fmt.Printf("   ✅ 信号模式: 质量优先 ✓ (适合均值回归)\n")
		score++
	} else {
		fmt.Printf("   ⚠️  信号模式: %s (均值回归建议使用质量优先)\n", mode)
	}

	// 3. 均线类型 (即使禁用也分析)
	if maTypeStr == "SMA" {
		fmt.Printf("   ✅ 均线类型: SMA ✓ (简单移动平均，适合均值回归)\n")
		score++
	} else {
		fmt.Printf("   ⚠️  均线类型: %s (SMA更适合均值回归策略)\n", maTypeStr)
	}

	// 4. 周期设置
	if short == "5" && long == "20" {
		fmt.Printf("   ✅ 周期设置: 5/20 ✓ (适合短期均值回归)\n")
		score++
	} else {
		fmt.Printf("   ⚠️  周期设置: %s/%s (5/20更适合均值回归)\n", short, long)
	}

	// 5. 交叉信号
	if cross == "BOTH" {
		fmt.Printf("   ✅ 交叉信号: 双向交易 ✓ (均值回归需要双向)\n")
		score++
	} else {
		fmt.Printf("   ⚠️  交叉信号: %s (均值回归需要双向交易)\n", cross)
	}

	// 6. 趋势过滤
	if filter == "否" {
		fmt.Printf("   ✅ 趋势过滤: 已关闭 ✓ (均值回归不依赖趋势)\n")
		score++
	} else {
		fmt.Printf("   ⚠️  趋势过滤: 已开启 (均值回归策略不需要趋势过滤)\n")
	}

	fmt.Printf("\n🏆 均值回归配置评分: %d/%d\n", score, maxScore)

	if score >= 6 {
		fmt.Printf("🎉 配置优秀！符合均值回归策略的要求\n")
	} else if score >= 4 {
		fmt.Printf("👍 配置良好，稍作调整会更好\n")
	} else {
		fmt.Printf("⚠️  配置需要优化以适应均值回归策略\n")
	}

	// 给出均值回归策略的具体建议
	giveMeanReversionSuggestions()
}

func giveMeanReversionSuggestions() {
	fmt.Printf("\n🎯 均值回归策略优化建议:\n")

	fmt.Println("📊 核心策略原理:")
	fmt.Println("   • 均值回归: 价格偏离均值时会回归")
	fmt.Println("   • 信号质量优先: 寻找高质量的反转机会")
	fmt.Println("   • 短期操作: 利用短期价格异常")

	fmt.Println("\n💡 配置建议:")
	fmt.Println("   1. 🎯 保持均线策略禁用 (均值回归不依赖均线)")
	fmt.Println("   2. 📊 信号模式: QUALITY_FIRST")
	fmt.Println("   3. 📈 周期设置: 5/20 (短期均值回归)")
	fmt.Println("   4. 🔄 双向交易: 捕捉上下反转机会")
	fmt.Println("   5. 🚫 趋势过滤: 保持关闭")

	fmt.Println("\n🎪 策略特点:")
	fmt.Println("   • 适合震荡行情和横盘市场")
	fmt.Println("   • 关注价格与均值的偏离程度")
	fmt.Println("   • 寻找超买超卖的反转信号")
	fmt.Println("   • 强调信号质量而非数量")

	fmt.Println("\n🚀 实施要点:")
	fmt.Println("   1. 监控价格偏离均值的标准差")
	fmt.Println("   2. 设置合理的止损和止盈")
	fmt.Println("   3. 避免在强趋势中操作")
	fmt.Println("   4. 结合成交量确认信号")
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