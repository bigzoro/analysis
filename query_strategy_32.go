package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 注意：这里需要替换为实际的数据库连接信息
	// 格式: "user:password@tcp(host:port)/database"
	dsn := "user:password@tcp(localhost:3306)/database"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		log.Printf("请确保数据库连接字符串正确")
		log.Printf("示例: user:password@tcp(localhost:3306)/database")
		return
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Printf("数据库连接测试失败: %v", err)
		return
	}

	fmt.Println("=== 查询策略ID为32的配置 ===")

	// 查询策略基本信息
	query := `
		SELECT
			id, name, strategy_type, status,
			mr_stop_loss_multiplier,
			mr_take_profit_multiplier,
			mr_max_position_size,
			mr_max_hold_hours,
			created_at, updated_at
		FROM trading_strategies
		WHERE id = 32
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		fmt.Println("❌ 未找到ID为32的策略记录")
		return
	}

	var id int
	var name, strategyType, status sql.NullString
	var stopLoss, takeProfit, maxPos sql.NullFloat64
	var maxHours sql.NullInt64
	var createdAt, updatedAt sql.NullTime

	err = rows.Scan(&id, &name, &strategyType, &status,
		&stopLoss, &takeProfit, &maxPos, &maxHours,
		&createdAt, &updatedAt)
	if err != nil {
		log.Printf("扫描结果失败: %v", err)
		return
	}

	fmt.Printf("策略ID: %d\n", id)
	fmt.Printf("策略名称: %s\n", nullStringValue(name))
	fmt.Printf("策略类型: %s\n", nullStringValue(strategyType))
	fmt.Printf("状态: %s\n", nullStringValue(status))
	fmt.Printf("创建时间: %v\n", nullTimeValue(createdAt))
	fmt.Printf("更新时间: %v\n", nullTimeValue(updatedAt))

	fmt.Printf("\n=== 均值回归配置 ===\n")
	fmt.Printf("MRStopLossMultiplier: %.6f (Null: %v)\n",
		nullFloat64Value(stopLoss), stopLoss.Valid)
	fmt.Printf("MRTakeProfitMultiplier: %.6f (Null: %v)\n",
		nullFloat64Value(takeProfit), takeProfit.Valid)
	fmt.Printf("MRMaxPositionSize: %.6f (Null: %v)\n",
		nullFloat64Value(maxPos), maxPos.Valid)
	fmt.Printf("MRMaxHoldHours: %d (Null: %v)\n",
		int(nullInt64Value(maxHours)), maxHours.Valid)

	// 分析配置问题
	fmt.Printf("\n=== 配置问题分析 ===\n")

	if !stopLoss.Valid || stopLoss.Float64 == 0 {
		fmt.Printf("❌ MRStopLossMultiplier 无效或为0\n")
		fmt.Printf("   这会导致StopLoss转换时出现问题: 1.0 / 0 = ∞\n")
		fmt.Printf("   建议: 设置为1.5-3.0之间的值\n")
	} else if stopLoss.Float64 < 0 {
		fmt.Printf("❌ MRStopLossMultiplier 为负数: %.3f\n", stopLoss.Float64)
		fmt.Printf("   这会导致StopLoss为负数，无效\n")
	} else if stopLoss.Float64 <= 1.0 {
		fmt.Printf("⚠️  MRStopLossMultiplier 过小: %.3f\n", stopLoss.Float64)
		fmt.Printf("   会导致StopLoss >= 1.0 (100%%止损)\n")
		fmt.Printf("   建议: 设置为1.5-3.0之间的值\n")
	} else {
		fmt.Printf("✅ MRStopLossMultiplier 正常: %.3f\n", stopLoss.Float64)
		convertedStopLoss := 1.0 / stopLoss.Float64
		fmt.Printf("   转换后StopLoss: %.3f%%\n", convertedStopLoss*100)
	}

	if !takeProfit.Valid || takeProfit.Float64 <= 1.0 {
		fmt.Printf("❌ MRTakeProfitMultiplier 无效或过小: %.3f\n", nullFloat64Value(takeProfit))
		fmt.Printf("   应该 > 1.0，比如1.08表示8%%止盈\n")
	} else {
		fmt.Printf("✅ MRTakeProfitMultiplier 正常: %.3f\n", takeProfit.Float64)
		convertedTakeProfit := takeProfit.Float64 - 1.0
		fmt.Printf("   转换后TakeProfit: %.1f%%\n", convertedTakeProfit*100)
	}

	if !maxPos.Valid || maxPos.Float64 <= 0 || maxPos.Float64 > 1 {
		fmt.Printf("❌ MRMaxPositionSize 无效: %.3f\n", nullFloat64Value(maxPos))
		fmt.Printf("   应该在0-1之间，比如0.05表示5%%仓位\n")
	} else {
		fmt.Printf("✅ MRMaxPositionSize 正常: %.1f%%\n", maxPos.Float64*100)
	}

	if !maxHours.Valid || maxHours.Int64 <= 0 {
		fmt.Printf("❌ MRMaxHoldHours 无效: %d\n", int(nullInt64Value(maxHours)))
		fmt.Printf("   应该 > 0，比如24表示24小时\n")
	} else {
		fmt.Printf("✅ MRMaxHoldHours 正常: %d小时\n", int(maxHours.Int64))
	}

	// 检查其他策略的配置作为对比
	fmt.Printf("\n=== 对比其他策略配置 ===\n")

	otherQuery := `
		SELECT id, mr_stop_loss_multiplier, mr_take_profit_multiplier,
		       mr_max_position_size, mr_max_hold_hours
		FROM trading_strategies
		WHERE strategy_type = 'mean_reversion'
		AND mr_stop_loss_multiplier IS NOT NULL
		AND mr_stop_loss_multiplier > 0
		ORDER BY id
		LIMIT 5
	`

	otherRows, err := db.Query(otherQuery)
	if err != nil {
		log.Printf("查询其他策略失败: %v", err)
	} else {
		defer otherRows.Close()

		fmt.Printf("其他正常策略的配置:\n")
		for otherRows.Next() {
			var oid int
			var ost, otp, ops sql.NullFloat64
			var ohr sql.NullInt64
			otherRows.Scan(&oid, &ost, &otp, &ops, &ohr)
			fmt.Printf("  ID %d: StopLoss=%.3f, TakeProfit=%.3f, Position=%.3f, Hours=%d\n",
				oid, nullFloat64Value(ost), nullFloat64Value(otp),
				nullFloat64Value(ops), int(nullInt64Value(ohr)))
		}
	}
}

func nullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "<NULL>"
}

func nullFloat64Value(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0.0
}

func nullInt64Value(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

func nullTimeValue(nt sql.NullTime) string {
	if nt.Valid {
		return nt.Time.Format("2006-01-02 15:04:05")
	}
	return "<NULL>"
}