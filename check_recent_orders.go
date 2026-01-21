package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("=== 检查最近创建的订单 ===\n")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 检查最近1小时内创建的订单
	rows, err := db.Query(`
		SELECT id, symbol, side, reduce_only, parent_order_id, client_order_id, status, created_at
		FROM scheduled_orders
		WHERE user_id = ? AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)
		ORDER BY created_at DESC
	`, 1)

	if err != nil {
		log.Printf("查询最近订单失败: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("最近1小时内创建的订单:")
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

		orderType := "开仓"
		if reduceOnly {
			orderType = "平仓"
		}
		if clientOrderId != "" && (clientOrderId[:13] == "PROFIT_SCALING" || len(clientOrderId) > 13 && clientOrderId[:13] == "PROFIT_SCALING") {
			orderType = "加仓"
		}

		parentInfo := "无"
		if parentOrderId != nil && *parentOrderId > 0 {
			parentInfo = fmt.Sprintf("%d", *parentOrderId)
		}

		fmt.Printf("  %s订单 %d (%s): 父订单=%s, 状态=%s, 时间=%s, ClientID=%s\n",
			orderType, id, symbol, parentInfo, status, createdAt, clientOrderId)
		count++
	}

	if count == 0 {
		fmt.Println("  最近1小时内没有创建任何订单")
	}

	// 检查最近24小时内是否有PROFIT_SCALING相关的订单
	fmt.Println("\n检查最近24小时内的PROFIT_SCALING订单:")
	rows2, err := db.Query(`
		SELECT id, symbol, side, reduce_only, parent_order_id, client_order_id, status, created_at
		FROM scheduled_orders
		WHERE user_id = ? AND client_order_id LIKE 'PROFIT_SCALING%' AND created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY created_at DESC
	`, 1)

	if err != nil {
		log.Printf("查询PROFIT_SCALING订单失败: %v", err)
		return
	}
	defer rows2.Close()

	count2 := 0
	for rows2.Next() {
		var id uint
		var symbol, side, clientOrderId, status string
		var reduceOnly bool
		var parentOrderId *uint
		var createdAt string

		err := rows2.Scan(&id, &symbol, &side, &reduceOnly, &parentOrderId, &clientOrderId, &status, &createdAt)
		if err != nil {
			log.Printf("扫描PROFIT_SCALING行失败: %v", err)
			continue
		}

		parentInfo := "无"
		if parentOrderId != nil && *parentOrderId > 0 {
			parentInfo = fmt.Sprintf("%d", *parentOrderId)
		}

		fmt.Printf("  加仓订单 %d (%s): 父订单=%s, 状态=%s, 时间=%s\n",
			id, symbol, parentInfo, status, createdAt)
		count2++
	}

	if count2 == 0 {
		fmt.Println("  最近24小时内没有PROFIT_SCALING订单")
		fmt.Println("  这说明盈利加仓功能可能没有正常工作")
	}
}
