package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("=== 分析Bracket订单（止盈止损订单）执行情况 ===")

	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	// 1. 检查BracketLink记录
	fmt.Printf("\n📊 Bracket订单链接统计:\n")
	var bracketStats struct {
		TotalBrackets int    `json:"total_brackets"`
		ActiveCount   int    `json:"active_count"`
		ClosedCount   int    `json:"closed_count"`
		CancelledCount int   `json:"cancelled_count"`
	}

	bracketQuery := `
		SELECT
			COUNT(*) as total_brackets,
			SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_count,
			SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END) as closed_count,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_count
		FROM bracket_links
	`

	gdb.GormDB().Raw(bracketQuery).Scan(&bracketStats)

	fmt.Printf("  总Bracket订单组: %d\n", bracketStats.TotalBrackets)
	fmt.Printf("  活跃Bracket订单: %d\n", bracketStats.ActiveCount)
	fmt.Printf("  已关闭Bracket订单: %d\n", bracketStats.ClosedCount)
	fmt.Printf("  已取消Bracket订单: %d\n", bracketStats.CancelledCount)

	if bracketStats.TotalBrackets > 0 {
		// 2. 查看最近的Bracket订单详情
		fmt.Printf("\n📝 最近5个Bracket订单详情:\n")
		var brackets []struct {
			ID            uint   `json:"id"`
			ScheduleID    uint   `json:"schedule_id"`
			Symbol        string `json:"symbol"`
			GroupID       string `json:"group_id"`
			EntryClientID string `json:"entry_client_id"`
			TPClientID    string `json:"tp_client_id"`
			SLClientID    string `json:"sl_client_id"`
			Status        string `json:"status"`
			CreatedAt     string `json:"created_at"`
		}

		detailQuery := `
			SELECT id, schedule_id, symbol, group_id, entry_client_id, tp_client_id, sl_client_id, status,
				   DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') as created_at
			FROM bracket_links
			ORDER BY created_at DESC LIMIT 5
		`

		gdb.GormDB().Raw(detailQuery).Scan(&brackets)

		for i, bracket := range brackets {
			fmt.Printf("  %d. Bracket ID: %d\n", i+1, bracket.ID)
			fmt.Printf("     交易对: %s\n", bracket.Symbol)
			fmt.Printf("     调度ID: %d\n", bracket.ScheduleID)
			fmt.Printf("     组ID: %s\n", bracket.GroupID)
			fmt.Printf("     状态: %s\n", bracket.Status)
			fmt.Printf("     创建时间: %s\n", bracket.CreatedAt)
			fmt.Printf("     入场单ID: %s\n", bracket.EntryClientID)
			fmt.Printf("     止盈单ID: %s\n", bracket.TPClientID)
			fmt.Printf("     止损单ID: %s\n", bracket.SLClientID)
			fmt.Println()
		}

		// 3. 检查这些订单的执行情况
		fmt.Printf("\n💰 订单执行情况分析:\n")

		// 获取所有相关的client order ids
		var allClientIDs []string
		for _, bracket := range brackets {
			if bracket.EntryClientID != "" {
				allClientIDs = append(allClientIDs, bracket.EntryClientID)
			}
			if bracket.TPClientID != "" {
				allClientIDs = append(allClientIDs, bracket.TPClientID)
			}
			if bracket.SLClientID != "" {
				allClientIDs = append(allClientIDs, bracket.SLClientID)
			}
		}

		if len(allClientIDs) > 0 {
			// 检查这些订单在scheduled_orders表中的状态
			var scheduledOrders []struct {
				ClientOrderID string `json:"client_order_id"`
				Status        string `json:"status"`
				Symbol        string `json:"symbol"`
				Side          string `json:"side"`
			}

			scheduledQuery := `
				SELECT client_order_id, status, symbol, side
				FROM scheduled_orders
				WHERE client_order_id IN (?)
			`

			gdb.GormDB().Raw(scheduledQuery, allClientIDs).Scan(&scheduledOrders)

			fmt.Printf("  调度订单状态:\n")
			for _, order := range scheduledOrders {
				fmt.Printf("    %s (%s %s): %s\n", order.ClientOrderID, order.Symbol, order.Side, order.Status)
			}

			// 检查orders表中的实际订单
			var realOrders []struct {
				ClientOrderID string  `json:"client_order_id"`
				Status        string  `json:"status"`
				Side          string  `json:"side"`
				Symbol        string  `json:"symbol"`
				Quantity      float64 `json:"quantity"`
				Price         float64 `json:"price"`
				Pnl           float64 `json:"pnl"`
			}

			ordersQuery := `
				SELECT client_order_id, status, side, symbol, quantity, price, pnl
				FROM orders
				WHERE client_order_id IN (?)
				ORDER BY created_at DESC
			`

			gdb.GormDB().Raw(ordersQuery, allClientIDs).Scan(&realOrders)

			fmt.Printf("\n  实际订单记录:\n")
			for _, order := range realOrders {
				fmt.Printf("    %s: %s %s %.4f@%.4f, 状态:%s, 盈亏:%.4f\n",
					order.ClientOrderID, order.Side, order.Symbol, order.Quantity, order.Price, order.Status, order.Pnl)
			}
		}
	}

	// 4. 分析止盈止损订单的触发情况
	fmt.Printf("\n🎯 止盈止损分析:\n")

	// 检查是否有止盈止损订单被执行
	tpSlStats := struct {
		TPOrders int `json:"tp_orders"`
		SLOrders int `json:"sl_orders"`
		TPFilled int `json:"tp_filled"`
		SLFilled int `json:"sl_filled"`
	}{}

	tpSlQuery := `
		SELECT
			SUM(CASE WHEN side = 'SELL' AND reduce_only = 1 AND client_order_id LIKE '%-tp' THEN 1 ELSE 0 END) as tp_orders,
			SUM(CASE WHEN side = 'SELL' AND reduce_only = 1 AND client_order_id LIKE '%-sl' THEN 1 ELSE 0 END) as sl_orders,
			SUM(CASE WHEN side = 'SELL' AND reduce_only = 1 AND client_order_id LIKE '%-tp' AND status = 'filled' THEN 1 ELSE 0 END) as tp_filled,
			SUM(CASE WHEN side = 'SELL' AND reduce_only = 1 AND client_order_id LIKE '%-sl' AND status = 'filled' THEN 1 ELSE 0 END) as sl_filled
		FROM orders
		WHERE (client_order_id LIKE '%-tp' OR client_order_id LIKE '%-sl')
	`

	gdb.GormDB().Raw(tpSlQuery).Scan(&tpSlStats)

	fmt.Printf("  止盈订单总数: %d\n", tpSlStats.TPOrders)
	fmt.Printf("  止损订单总数: %d\n", tpSlStats.SLOrders)
	fmt.Printf("  已执行止盈订单: %d\n", tpSlStats.TPFilled)
	fmt.Printf("  已执行止损订单: %d\n", tpSlStats.SLFilled)

	if tpSlStats.TPOrders > 0 {
		tpSuccessRate := float64(tpSlStats.TPFilled) / float64(tpSlStats.TPOrders) * 100
		fmt.Printf("  止盈成功率: %.1f%%\n", tpSuccessRate)
	}

	if tpSlStats.SLOrders > 0 {
		slSuccessRate := float64(tpSlStats.SLFilled) / float64(tpSlStats.SLOrders) * 100
		fmt.Printf("  止损成功率: %.1f%%\n", slSuccessRate)
	}

	// 5. 检查策略33相关的Bracket订单
	fmt.Printf("\n🎲 策略33相关分析:\n")

	var strategy33Brackets []struct {
		ID         uint   `json:"id"`
		GroupID    string `json:"group_id"`
		Symbol     string `json:"symbol"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
	}

	strategy33Query := `
		SELECT bl.id, bl.group_id, bl.symbol, bl.status, DATE_FORMAT(bl.created_at, '%Y-%m-%d %H:%i:%s') as created_at
		FROM bracket_links bl
		INNER JOIN scheduled_orders so ON bl.schedule_id = so.id
		WHERE so.strategy_id = 33
		ORDER BY bl.created_at DESC LIMIT 3
	`

	gdb.GormDB().Raw(strategy33Query).Scan(&strategy33Brackets)

	fmt.Printf("  策略33的Bracket订单:\n")
	for _, bracket := range strategy33Brackets {
		fmt.Printf("    ID:%d, 交易对:%s, 状态:%s, 创建时间:%s\n",
			bracket.ID, bracket.Symbol, bracket.Status, bracket.CreatedAt)
	}

	// 6. 总结分析
	fmt.Printf("\n📋 总结分析:\n")

	if bracketStats.TotalBrackets > 0 {
		fmt.Printf("✅ 已创建 %d 个Bracket订单组\n", bracketStats.TotalBrackets)

		if bracketStats.ActiveCount > 0 {
			fmt.Printf("✅ 当前有 %d 个活跃Bracket订单\n", bracketStats.ActiveCount)
		}

		if tpSlStats.TPFilled > 0 || tpSlStats.SLFilled > 0 {
			fmt.Printf("✅ 止盈止损功能已执行: 止盈%d次, 止损%d次\n", tpSlStats.TPFilled, tpSlStats.SLFilled)
			fmt.Printf("🎯 保证金止盈止损功能正在正常工作！\n")
		} else if tpSlStats.TPOrders > 0 || tpSlStats.SLOrders > 0 {
			fmt.Printf("⏳ 已创建止盈止损订单但尚未触发\n")
			fmt.Printf("📝 订单创建正常，等待市场条件触发\n")
		} else {
			fmt.Printf("⚠️  没有找到止盈止损订单执行记录\n")
			fmt.Printf("🔍 需要进一步检查订单创建流程\n")
		}
	} else {
		fmt.Printf("📝 尚未创建任何Bracket订单\n")
		fmt.Printf("ℹ️  可能是因为策略尚未触发或配置问题\n")
	}
}