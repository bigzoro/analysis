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

	fmt.Println("🔧 修复外键约束问题...")

	// 修复第一个约束：strategy_executions -> trading_strategies
	fmt.Println("1. 修复 strategy_executions -> trading_strategies 约束...")
	constraintsToFix := []struct {
		table      string
		column     string
		refTable   string
		constraint string
	}{
		{"strategy_executions", "strategy_id", "trading_strategies", "fk_strategy_executions_strategy"},
		{"strategy_execution_steps", "execution_id", "strategy_executions", "fk_strategy_execution_steps_execution"},
	}

	for i, constraint := range constraintsToFix {
		fmt.Printf("1.%d 修复 %s 约束...\n", i+1, constraint.constraint)

		// 删除现有约束
		dropSQL := fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", constraint.table, constraint.constraint)
		_, err = db.Exec(dropSQL)
		if err != nil {
			log.Printf("删除约束 %s 失败: %v", constraint.constraint, err)
			continue
		}

		// 添加CASCADE约束
		addSQL := fmt.Sprintf(`
			ALTER TABLE %s ADD CONSTRAINT %s
			FOREIGN KEY (%s) REFERENCES %s(id) ON DELETE CASCADE
		`, constraint.table, constraint.constraint, constraint.column, constraint.refTable)
		_, err = db.Exec(addSQL)
		if err != nil {
			log.Printf("添加CASCADE约束 %s 失败: %v", constraint.constraint, err)
			continue
		}

		fmt.Printf("✅ %s 约束修复完成\n", constraint.constraint)
	}

	// 验证修复结果
	fmt.Println("3. 验证修复结果...")
	verifyQuery := `
		SELECT
			kcu.TABLE_NAME,
			kcu.COLUMN_NAME,
			kcu.CONSTRAINT_NAME,
			rc.DELETE_RULE,
			rc.UPDATE_RULE
		FROM
			information_schema.KEY_COLUMN_USAGE kcu
		LEFT JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
			ON kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
			AND kcu.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE
			kcu.TABLE_SCHEMA = 'analysis'
			AND (
				(kcu.TABLE_NAME = 'strategy_executions' AND kcu.COLUMN_NAME = 'strategy_id') OR
				(kcu.TABLE_NAME = 'strategy_execution_steps' AND kcu.COLUMN_NAME = 'execution_id')
			)
		ORDER BY kcu.TABLE_NAME
	`

	rows, err := db.Query(verifyQuery)
	if err != nil {
		log.Fatal("验证查询失败:", err)
	}
	defer rows.Close()

	fmt.Println("修复结果:")
	allCorrect := true
	for rows.Next() {
		var tableName, columnName, constraintName, deleteRule, updateRule string
		err := rows.Scan(&tableName, &columnName, &constraintName, &deleteRule, &updateRule)
		if err != nil {
			log.Fatal("扫描结果失败:", err)
		}

		fmt.Printf("  %s.%s -> %s (DELETE: %s, UPDATE: %s)\n",
			tableName, columnName, constraintName, deleteRule, updateRule)

		if deleteRule != "CASCADE" {
			allCorrect = false
		}
	}

	if allCorrect {
		fmt.Println("✅ 所有外键约束修复成功！删除规则均设置为CASCADE")
	} else {
		fmt.Println("❌ 部分约束修复失败！")
	}
}
