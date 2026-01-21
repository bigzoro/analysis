package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("=== 检查订单的父子关系 ===\n")

	// 直接执行SQL查询
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 检查是否有parent_order_id不为NULL的订单
	rows, err := db.Query(`
		SELECT id, symbol, side, reduce_only, parent_order_id, client_order_id, status
		FROM scheduled_orders
		WHERE user_id = ? AND parent_order_id IS NOT NULL
		ORDER BY created_at DESC
		LIMIT 20
	`, 1)

	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("具有父订单的订单:")
	count := 0
	for rows.Next() {
		var id uint
		var symbol, side, clientOrderId, status string
		var reduceOnly bool
		var parentOrderId uint

		err := rows.Scan(&id, &symbol, &side, &reduceOnly, &parentOrderId, &clientOrderId, &status)
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

		fmt.Printf("  %s订单 %d (%s): 父订单=%d\n",
			orderType, id, symbol, parentOrderId)
		count++
	}

	if count == 0 {
		fmt.Println("  没有找到任何具有父订单的订单")
	}

	fmt.Printf("\n总共找到 %d 个具有父订单的订单\n", count)

	// 检查最近的SCRTUSDT订单
	fmt.Println("\nSCRTUSDT的最近订单:")
	rows2, err := db.Query(`
		SELECT id, symbol, side, reduce_only, parent_order_id, client_order_id, status, created_at
		FROM scheduled_orders
		WHERE user_id = ? AND symbol = ?
		ORDER BY created_at DESC
		LIMIT 10
	`, 1, "SCRTUSDT")

	if err != nil {
		log.Printf("查询SCRTUSDT订单失败: %v", err)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var id uint
		var symbol, side, clientOrderId, status string
		var reduceOnly bool
		var parentOrderId *uint
		var createdAt time.Time

		err := rows2.Scan(&id, &symbol, &side, &reduceOnly, &parentOrderId, &clientOrderId, &status, &createdAt)
		if err != nil {
			log.Printf("扫描SCRTUSDT行失败: %v", err)
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
		if parentOrderId != nil {
			parentInfo = fmt.Sprintf("%d", *parentOrderId)
		}

		fmt.Printf("  %s订单 %d: 父订单=%s, 状态=%s, 时间=%s\n",
			orderType, id, parentInfo, status, createdAt.Format("15:04:05"))
	}
}
