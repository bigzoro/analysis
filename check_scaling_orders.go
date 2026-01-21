package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("=== 检查加仓订单 ===\n")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 查找所有加仓订单
	rows, err := db.Query(`
		SELECT id, symbol, side, reduce_only, parent_order_id, client_order_id, status, created_at
		FROM scheduled_orders
		WHERE user_id = ? AND client_order_id LIKE '%PROFIT_SCALING%'
		ORDER BY created_at DESC
		LIMIT 20
	`, 1)

	if err != nil {
		log.Printf("查询加仓订单失败: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("找到的加仓订单:")
	count := 0
	for rows.Next() {
		var id uint
		var symbol, side, clientOrderId, status string
		var reduceOnly bool
		var parentOrderId *uint
		var createdAt string

		err := rows.Scan(&id, &symbol, &side, &reduceOnly, &parentOrderId, &clientOrderId, &status, &createdAt)
		if err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		parentInfo := "无"
		if parentOrderId != nil {
			parentInfo = fmt.Sprintf("%d", *parentOrderId)
		}

		fmt.Printf("  加仓订单 %d (%s): 父订单=%s, 状态=%s, 时间=%s\n",
			id, symbol, parentInfo, status, createdAt)
		count++
	}

	if count == 0 {
		fmt.Println("  没有找到任何加仓订单")
	} else {
		fmt.Printf("\n总共找到 %d 个加仓订单\n", count)
	}

	// 检查最近的所有订单，看看是否有parent_order_id的设置
	fmt.Println("\n检查最近的订单父子关系:")
	rows2, err := db.Query(`
		SELECT id, symbol, side, reduce_only, parent_order_id, client_order_id, status
		FROM scheduled_orders
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 30
	`, 1)

	if err != nil {
		log.Printf("查询最近订单失败: %v", err)
		return
	}
	defer rows2.Close()

	fmt.Println("最近30个订单的父子关系:")
	for rows2.Next() {
		var id uint
		var symbol, side, clientOrderId, status string
		var reduceOnly bool
		var parentOrderId *uint

		err := rows2.Scan(&id, &symbol, &side, &reduceOnly, &parentOrderId, &clientOrderId, &status)
		if err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		orderType := "开仓"
		if reduceOnly {
			orderType = "平仓"
		}
		if clientOrderId != "" && strings.Contains(clientOrderId, "PROFIT_SCALING") {
			orderType = "加仓"
		}

		parentInfo := "无"
		if parentOrderId != nil && *parentOrderId > 0 {
			parentInfo = fmt.Sprintf("%d", *parentOrderId)
		}

		if parentOrderId != nil && *parentOrderId > 0 {
			fmt.Printf("  %s订单 %d (%s): 父订单=%s\n",
				orderType, id, symbol, parentInfo)
		}
	}
}
