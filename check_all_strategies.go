package main

import (
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"database/sql"
)

func main() {
	fmt.Println("=== 检查数据库中的所有策略 ===")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 查询所有策略
	rows, err := db.Query("SELECT id, name, ma_signal_mode, ma_type, short_ma_period, long_ma_period, moving_average_enabled FROM trading_strategies ORDER BY id")
	if err != nil {
		log.Fatal("查询策略失败:", err)
	}
	defer rows.Close()

	fmt.Println("数据库中的所有策略:")
	fmt.Printf("%-3s %-20s %-15s %-8s %-12s %-12s %-10s\n",
		"ID", "名称", "信号模式", "均线类型", "短期周期", "长期周期", "MA启用")
	fmt.Println(strings.Repeat("-", 85))

	count := 0
	for rows.Next() {
		var id int
		var name string
		var signalMode, maType sql.NullString
		var shortPeriod, longPeriod sql.NullInt32
		var maEnabled sql.NullBool

		err := rows.Scan(&id, &name, &signalMode, &maType, &shortPeriod, &longPeriod, &maEnabled)
		if err != nil {
			continue
		}

		signalModeStr := getStringValue(signalMode)
		maTypeStr := getStringValue(maType)
		shortStr := getIntValue(shortPeriod)
		longStr := getIntValue(longPeriod)
		enabledStr := getBoolValue(maEnabled)

		fmt.Printf("%-3d %-20s %-15s %-8s %-12s %-12s %-10s\n",
			id, truncateString(name, 18), signalModeStr, maTypeStr, shortStr, longStr, enabledStr)
		count++
	}

	fmt.Printf("\n总计: %d个策略\n", count)

	// 如果没有找到策略23，显示最近的策略
	if count > 0 {
		fmt.Println("\n💡 建议:")
		fmt.Println("1. 检查策略ID是否正确")
		fmt.Println("2. 如果是新创建的策略，ID可能不同")
		fmt.Println("3. 查看最新的策略ID")
	}
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
