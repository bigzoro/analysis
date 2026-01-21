package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./analysis.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 查看表结构
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '%market%'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Market tables:")
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Println("-", name)
	}
}