package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🔄 均值回归策略增强功能数据库迁移")
	fmt.Println("=================================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	fmt.Println("\n📋 检查现有表结构...")

	// 检查trading_strategies表是否存在
	var tableName string
	err = db.QueryRow("SHOW TABLES LIKE 'trading_strategies'").Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatal("trading_strategies表不存在，请先运行基础迁移")
		}
		log.Fatal("检查表存在性失败:", err)
	}

	if tableName != "trading_strategies" {
		log.Fatal("trading_strategies表不存在，请先运行基础迁移")
	}

	fmt.Println("✅ trading_strategies表存在")

	// 获取现有列
	existingColumns := make(map[string]bool)
	rows, err := db.Query("DESCRIBE trading_strategies")
	if err != nil {
		log.Fatal("获取表结构失败:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var field, typ, null, key string
		var def, extra sql.NullString
		err := rows.Scan(&field, &typ, &null, &key, &def, &extra)
		if err != nil {
			log.Fatal("扫描列信息失败:", err)
		}
		existingColumns[field] = true
	}

	fmt.Printf("📊 发现%d个现有列\n", len(existingColumns))

	// 需要添加的新列
	newColumns := []struct {
		name     string
		sqlType  string
		defValue string
		comment  string
	}{
		// 策略版本和模式
		{"mean_reversion_mode", "VARCHAR(20)", "'basic'", "策略模式: basic/enhanced"},
		{"mean_reversion_sub_mode", "VARCHAR(20)", "'conservative'", "子模式: conservative/aggressive"},

		// 增强功能开关
		{"market_environment_detection", "TINYINT(1)", "0", "市场环境检测启用"},
		{"intelligent_weights", "TINYINT(1)", "0", "智能信号权重启用"},
		{"advanced_risk_management", "TINYINT(1)", "0", "高级风险管理启用"},
		{"adaptive_parameters", "TINYINT(1)", "0", "自适应参数启用"},
		{"performance_monitoring", "TINYINT(1)", "0", "性能监控启用"},
		{"mode_switching", "TINYINT(1)", "0", "模式切换启用"},

		// 市场环境检测参数
		{"mr_env_trend_threshold", "DECIMAL(10,4)", "0.7", "趋势强度阈值"},
		{"mr_env_volatility_threshold", "DECIMAL(10,4)", "0.3", "波动率阈值"},
		{"mr_env_oscillation_threshold", "DECIMAL(10,4)", "0.6", "震荡指数阈值"},

		// 智能权重参数
		{"mr_weight_bollinger_bands", "DECIMAL(5,2)", "1.0", "布林带权重"},
		{"mr_weight_rsi", "DECIMAL(5,2)", "1.0", "RSI权重"},
		{"mr_weight_price_channel", "DECIMAL(5,2)", "1.0", "价格通道权重"},
		{"mr_weight_time_decay", "DECIMAL(5,2)", "0.2", "时间衰减权重"},

		// 高级风险管理参数
		{"mr_max_daily_loss", "DECIMAL(5,4)", "0.03", "每日最大亏损比例(3%)"},
		{"mr_max_position_size", "DECIMAL(5,4)", "0.02", "最大仓位比例(2%)"},
		{"mr_stop_loss_multiplier", "DECIMAL(5,2)", "2.0", "止损倍数"},
		{"mr_take_profit_multiplier", "DECIMAL(5,2)", "3.0", "止盈倍数"},
		{"mr_max_hold_hours", "INT", "24", "最大持仓小时数"},

		// 自适应参数
		{"mr_auto_adjust_period", "TINYINT(1)", "0", "自动调整周期"},
		{"mr_auto_adjust_multiplier", "TINYINT(1)", "0", "自动调整倍数"},
		{"mr_auto_adjust_thresholds", "TINYINT(1)", "0", "自动调整阈值"},

		// 候选币种优化参数
		{"mr_candidate_min_oscillation", "DECIMAL(5,2)", "0.5", "最小振荡性要求"},
		{"mr_candidate_min_liquidity", "DECIMAL(10,2)", "1000000", "最小流动性要求(100万USDT)"},
		{"mr_candidate_max_volatility", "DECIMAL(5,4)", "0.15", "最大波动率限制(15%)"},
	}

	fmt.Println("\n🔧 开始添加新列...")

	addedCount := 0
	for _, col := range newColumns {
		if existingColumns[col.name] {
			fmt.Printf("⏭️  列%s已存在，跳过\n", col.name)
			continue
		}

		sql := fmt.Sprintf("ALTER TABLE trading_strategies ADD COLUMN %s %s DEFAULT %s COMMENT '%s'",
			col.name, col.sqlType, col.defValue, col.comment)

		_, err := db.Exec(sql)
		if err != nil {
			log.Printf("❌ 添加列%s失败: %v", col.name, err)
			continue
		}

		fmt.Printf("✅ 成功添加列: %s\n", col.name)
		addedCount++
	}

	fmt.Printf("\n🎉 迁移完成！共添加了%d个新列\n", addedCount)

	// 设置默认值给现有记录
	fmt.Println("\n📝 为现有均值回归策略设置默认增强参数...")

	updateSQL := `
		UPDATE trading_strategies
		SET
			mean_reversion_mode = 'basic',
			mean_reversion_sub_mode = 'conservative',
			market_environment_detection = 0,
			intelligent_weights = 0,
			advanced_risk_management = 0,
			adaptive_parameters = 0,
			performance_monitoring = 0,
			mode_switching = 0,
			mr_env_trend_threshold = 0.7,
			mr_env_volatility_threshold = 0.3,
			mr_env_oscillation_threshold = 0.6,
			mr_weight_bollinger_bands = 1.0,
			mr_weight_rsi = 1.0,
			mr_weight_price_channel = 1.0,
			mr_weight_time_decay = 0.2,
			mr_max_daily_loss = 0.03,
			mr_max_position_size = 0.02,
			mr_stop_loss_multiplier = 2.0,
			mr_take_profit_multiplier = 3.0,
			mr_max_hold_hours = 24,
			mr_auto_adjust_period = 0,
			mr_auto_adjust_multiplier = 0,
			mr_auto_adjust_thresholds = 0,
			mr_candidate_min_oscillation = 0.5,
			mr_candidate_min_liquidity = 1000000,
			mr_candidate_max_volatility = 0.15
		WHERE mean_reversion_enabled = 1
	`

	result, err := db.Exec(updateSQL)
	if err != nil {
		log.Printf("❌ 设置默认值失败: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✅ 为%d个现有均值回归策略设置了默认增强参数\n", rowsAffected)
	}

	// 验证迁移结果
	fmt.Println("\n🔍 验证迁移结果...")

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM trading_strategies WHERE mean_reversion_enabled = 1 AND mean_reversion_mode = 'basic'").Scan(&count)
	if err != nil {
		log.Printf("❌ 验证失败: %v", err)
	} else {
		fmt.Printf("✅ 验证成功：%d个策略已设置为基础模式\n", count)
	}

	fmt.Println("\n🎉 数据库迁移完成！")
	fmt.Println("\n📚 使用说明：")
	fmt.Println("1. 现有策略自动设为'basic'模式，保持原有行为")
	fmt.Println("2. 新创建策略可选择'enhanced'模式启用增强功能")
	fmt.Println("3. 前端可通过这些新字段控制增强功能")
	fmt.Println("4. 建议逐步迁移策略到增强模式进行测试")
}