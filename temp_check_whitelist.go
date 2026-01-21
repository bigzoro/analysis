package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 数据库连接
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 查询trading_strategies表中的symbol_whitelist字段
	rows, err := db.Query("SELECT id, name, symbol_whitelist FROM trading_strategies")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("📊 检查trading_strategies表中的symbol_whitelist字段:")
	fmt.Println("==================================================")

	for rows.Next() {
		var id int
		var name string
		var whitelist sql.NullString

		err := rows.Scan(&id, &name, &whitelist)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("ID: %d\n", id)
		fmt.Printf("Name: %s\n", name)
		if whitelist.Valid {
			fmt.Printf("Whitelist: %s\n", whitelist.String)
		} else {
			fmt.Printf("Whitelist: NULL\n")
		}
		fmt.Println("------------------------------")
	}

	// 检查是否有无效的JSON数据
	fmt.Println("\n🔍 检查无效JSON数据:")
	rows2, err := db.Query("SELECT id, name FROM trading_strategies WHERE symbol_whitelist IS NOT NULL AND symbol_whitelist != '' AND JSON_VALID(symbol_whitelist) = 0")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	invalidCount := 0
	for rows2.Next() {
		var id int
		var name string
		rows2.Scan(&id, &name)
		fmt.Printf("❌ 无效JSON - ID: %d, Name: %s\n", id, name)
		invalidCount++
	}

	if invalidCount == 0 {
		fmt.Println("✅ 所有symbol_whitelist字段都是有效的JSON")
	}
}