package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== trading_strategies 表结构 ===")
	rows, err := db.Query("DESCRIBE trading_strategies")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var field, typ, null, key, def, extra string
		rows.Scan(&field, &typ, &null, &key, &def, &extra)
		fmt.Printf("%-25s %-15s %-5s %-5s %-10s %s\n", field, typ, null, key, def, extra)
	}
}
