package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 数据库连接
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("检查外键约束...")

	// 查询外键约束
	query := `
		SELECT
			kcu.TABLE_NAME,
			kcu.COLUMN_NAME,
			kcu.CONSTRAINT_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			rc.DELETE_RULE,
			rc.UPDATE_RULE
		FROM
			information_schema.KEY_COLUMN_USAGE kcu
		LEFT JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
			ON kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
			AND kcu.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE
			kcu.REFERENCED_TABLE_SCHEMA = 'analysis'
			AND (kcu.REFERENCED_TABLE_NAME = 'trading_strategies'
			OR kcu.REFERENCED_TABLE_NAME = 'strategy_executions')
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Failed to query foreign keys:", err)
	}
	defer rows.Close()

	fmt.Printf("%-20s %-15s %-25s %-20s %-20s %-15s %-15s\n",
		"TABLE_NAME", "COLUMN_NAME", "CONSTRAINT_NAME", "REF_TABLE", "REF_COLUMN", "DELETE_RULE", "UPDATE_RULE")
	fmt.Println("-----------------------------------------------------------------------------------------------")

	found := false
	for rows.Next() {
		var tableName, columnName, constraintName, refTable, refColumn, deleteRule, updateRule sql.NullString
		err := rows.Scan(&tableName, &columnName, &constraintName, &refTable, &refColumn, &deleteRule, &updateRule)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		fmt.Printf("%-20s %-15s %-25s %-20s %-20s %-15s %-15s\n",
			tableName.String, columnName.String, constraintName.String,
			refTable.String, refColumn.String, deleteRule.String, updateRule.String)
		found = true
	}

	if !found {
		fmt.Println("未找到引用 trading_strategies 表的外键约束")
	}

	// 检查 strategy_executions 表中的数据
	fmt.Println("\n检查 strategy_executions 表中的数据...")
	countQuery := "SELECT COUNT(*) FROM strategy_executions WHERE strategy_id NOT IN (SELECT id FROM trading_strategies)"
	var count int
	err = db.QueryRow(countQuery).Scan(&count)
	if err != nil {
		log.Fatal("Failed to count orphaned records:", err)
	}

	if count > 0 {
		fmt.Printf("⚠️ 发现 %d 条孤立记录（strategy_executions表中引用了不存在的strategy_id）\n", count)

		// 显示一些示例
		exampleQuery := `
			SELECT se.id, se.strategy_id, se.status, se.created_at
			FROM strategy_executions se
			WHERE se.strategy_id NOT IN (SELECT id FROM trading_strategies)
			LIMIT 5
		`
		rows, err := db.Query(exampleQuery)
		if err == nil {
			defer rows.Close()
			fmt.Println("孤立记录示例:")
			for rows.Next() {
				var id, strategyId int
				var status, createdAt string
				rows.Scan(&id, &strategyId, &status, &createdAt)
				fmt.Printf("  ID: %d, StrategyID: %d, Status: %s, Created: %s\n", id, strategyId, status, createdAt)
			}
		}
	} else {
		fmt.Println("✅ strategy_executions 表中没有孤立记录")
	}
}
