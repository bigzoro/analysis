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

	// 检查trading_strategies表的结构
	rows, err := db.Query("DESCRIBE trading_strategies")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("trading_strategies表结构:")
	fmt.Println("Field | Type | Null | Key | Default | Extra")
	fmt.Println("------|------|------|-----|---------|-------")

	for rows.Next() {
		var field, typ, null, key, def, extra string
		rows.Scan(&field, &typ, &null, &key, &def, &extra)
		fmt.Printf("%s | %s | %s | %s | %s | %s\n", field, typ, null, key, def, extra)
	}

	// 特别检查symbol_whitelist字段
	fmt.Println("\n🔍 检查symbol_whitelist字段的当前值:")
	rows2, err := db.Query("SELECT id, name, symbol_whitelist FROM trading_strategies WHERE symbol_whitelist IS NOT NULL")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	count := 0
	for rows2.Next() {
		var id int
		var name string
		var whitelist sql.NullString
		rows2.Scan(&id, &name, &whitelist)
		fmt.Printf("ID: %d, Name: %s, Whitelist: %s\n", id, name, whitelist.String)
		count++
	}

	if count == 0 {
		fmt.Println("没有找到symbol_whitelist不为NULL的记录")
	}
}