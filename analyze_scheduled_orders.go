package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ScheduledOrder struct {
	ID              uint      `json:"id"`
	UserID          uint      `json:"user_id"`
	Exchange        string    `json:"exchange"`
	Testnet         bool      `json:"testnet"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	OrderType       string    `json:"order_type"`
	Quantity        string    `json:"quantity"`
	Price           string    `json:"price"`
	Leverage        int       `json:"leverage"`
	ReduceOnly      bool      `json:"reduce_only"`
	TriggerTime     time.Time `json:"trigger_time"`
	Status          string    `json:"status"`
	Result          string    `json:"result"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	BracketEnabled  bool      `json:"bracket_enabled"`
	TPPercent       float64   `json:"tp_percent"`
	SLPercent       float64   `json:"sl_percent"`
	TPPrice         string    `json:"tp_price"`
	SLPrice         string    `json:"sl_price"`
	WorkingType     string    `json:"working_type"`
	StrategyID      uint      `json:"strategy_id"`
	AdjustedQty     string    `json:"adjusted_quantity"`
	ClientOrderID   string    `json:"client_order_id"`
	ExchangeOrderID string    `json:"exchange_order_id"`
	ExecutedQty     string    `json:"executed_qty"`
	AvgPrice        string    `json:"avg_price"`
	ExecutedQtyAlt  string    `json:"executed_quantity"`
	ActualTPPercent float64   `json:"actual_tp_percent"`
	ActualSLPercent float64   `json:"actual_sl_percent"`
	ParentOrderID   int       `json:"parent_order_id"`
	CloseOrderIDs   string    `json:"close_order_ids"`
	ExecutionID     uint      `json:"execution_id"`
	ArbType         string    `json:"arb_type"`
	RelatedOrderID  uint      `json:"related_order_id"`
	StrategyType    string    `json:"strategy_type"`
	GridLevel       int       `json:"grid_level"`
}

func main() {
	fmt.Println("=== 调度订单分析 ===")
	fmt.Println("分析scheduled_orders表的交易记录")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 分析FIL网格策略的调度订单
	fmt.Println("\n📊 第一阶段: FIL网格策略调度订单分析")
	analyzeScheduledOrders(db)

	// 2. 分析交易时间间隔
	fmt.Println("\n📊 第二阶段: 交易时间间隔分析")
	analyzeOrderIntervals(db)

	// 3. 分析订单状态分布
	fmt.Println("\n📊 第三阶段: 订单状态分析")
	analyzeOrderStatus(db)

	// 4. 分析盈利情况
	fmt.Println("\n📊 第四阶段: 盈利分析")
	analyzePnL(db)
}

func analyzeScheduledOrders(db *gorm.DB) {
	var orders []ScheduledOrder
	result := db.Where("strategy_id = ? AND symbol = ?", 29, "FILUSDT").Order("created_at DESC").Limit(50).Find(&orders)

	if result.Error != nil {
		fmt.Printf("查询调度订单失败: %v\n", result.Error)
		return
	}

	fmt.Printf("FIL网格策略的调度订单 (最近50条):\n")
	fmt.Printf("%-5s %-12s %-6s %-10s %-12s %-8s %-12s %-15s\n",
		"ID", "时间", "方向", "状态", "数量", "价格", "网格层", "执行ID")

	buyOrders := 0
	sellOrders := 0
	filledOrders := 0
	pendingOrders := 0
	cancelledOrders := 0

	byStatus := make(map[string]int)

	for _, order := range orders {
		side := order.Side
		if side == "BUY" {
			buyOrders++
		} else if side == "SELL" {
			sellOrders++
		}

		byStatus[order.Status]++

		if order.Status == "FILLED" {
			filledOrders++
		} else if order.Status == "PENDING" || order.Status == "NEW" {
			pendingOrders++
		} else if order.Status == "CANCELLED" || order.Status == "CANCELED" {
			cancelledOrders++
		}

		fmt.Printf("%-5d %-12s %-6s %-10s %-12s %-8s %-12d %-15d\n",
			order.ID,
			order.CreatedAt.Format("01-02 15:04"),
			side,
			order.Status,
			order.Quantity,
			order.Price,
			order.GridLevel,
			order.ExecutionID)
	}

	fmt.Printf("\n订单统计:\n")
	fmt.Printf("总订单数: %d\n", len(orders))
	fmt.Printf("买入订单: %d\n", buyOrders)
	fmt.Printf("卖出订单: %d\n", sellOrders)
	fmt.Printf("已成交订单: %d\n", filledOrders)
	fmt.Printf("待处理订单: %d\n", pendingOrders)
	fmt.Printf("已取消订单: %d\n", cancelledOrders)

	fmt.Printf("\n状态分布:\n")
	for status, count := range byStatus {
		fmt.Printf("  %s: %d\n", status, count)
	}
}

func analyzeOrderIntervals(db *gorm.DB) {
	var orders []ScheduledOrder
	result := db.Where("strategy_id = ? AND symbol = ? AND status = ?", 29, "FILUSDT", "FILLED").
		Order("created_at ASC").Find(&orders)

	if result.Error != nil {
		fmt.Printf("查询订单间隔失败: %v\n", result.Error)
		return
	}

	if len(orders) < 2 {
		fmt.Printf("成交订单不足，无法分析时间间隔 (当前成交订单: %d)\n", len(orders))
		return
	}

	fmt.Printf("交易时间间隔分析 (基于%d个已成交订单):\n", len(orders))

	intervals := make([]time.Duration, 0)
	for i := 1; i < len(orders); i++ {
		interval := orders[i].CreatedAt.Sub(orders[i-1].CreatedAt)
		intervals = append(intervals, interval)
		fmt.Printf("订单 %d -> %d: %v\n", orders[i-1].ID, orders[i].ID, interval)
	}

	if len(intervals) > 0 {
		totalInterval := time.Duration(0)
		minInterval := intervals[0]
		maxInterval := intervals[0]

		for _, interval := range intervals {
			totalInterval += interval
			if interval < minInterval {
				minInterval = interval
			}
			if interval > maxInterval {
				maxInterval = interval
			}
		}

		avgInterval := totalInterval / time.Duration(len(intervals))

		fmt.Printf("\n时间间隔统计:\n")
		fmt.Printf("平均间隔: %v\n", avgInterval)
		fmt.Printf("最小间隔: %v\n", minInterval)
		fmt.Printf("最大间隔: %v\n", maxInterval)
		if len(orders) > 1 {
			totalTime := orders[len(orders)-1].CreatedAt.Sub(orders[0].CreatedAt)
			fmt.Printf("总观察时间: %v\n", totalTime)
			fmt.Printf("每小时交易频率: %.2f 次\n", float64(len(orders))/totalTime.Hours())
		}
	}
}

func analyzeOrderStatus(db *gorm.DB) {
	var statusStats []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	query := `
		SELECT status, COUNT(*) as count
		FROM scheduled_orders
		WHERE strategy_id = ? AND symbol = ?
		GROUP BY status
		ORDER BY count DESC
	`
	db.Raw(query, 29, "FILUSDT").Scan(&statusStats)

	fmt.Printf("订单状态分布:\n")
	for _, stat := range statusStats {
		fmt.Printf("  %s: %d\n", stat.Status, stat.Count)
	}

	// 分析执行ID分布
	var executionStats []struct {
		ExecutionID uint `json:"execution_id"`
		Count       int  `json:"count"`
	}

	execQuery := `
		SELECT execution_id, COUNT(*) as count
		FROM scheduled_orders
		WHERE strategy_id = ? AND symbol = ? AND execution_id IS NOT NULL
		GROUP BY execution_id
		ORDER BY execution_id DESC
		LIMIT 10
	`
	db.Raw(execQuery, 29, "FILUSDT").Scan(&executionStats)

	fmt.Printf("\n按执行ID分组的订单数:\n")
	for _, stat := range executionStats {
		fmt.Printf("  执行ID %d: %d 个订单\n", stat.ExecutionID, stat.Count)
	}
}

func analyzePnL(db *gorm.DB) {
	// 从strategy_executions表获取PnL数据
	var executions []struct {
		ID            uint      `json:"id"`
		TotalPnL      float64   `json:"total_pnl"`
		TotalOrders   int       `json:"total_orders"`
		SuccessOrders int       `json:"success_orders"`
		WinRate       float64   `json:"win_rate"`
		CreatedAt     time.Time `json:"created_at"`
	}

	result := db.Table("strategy_executions").Where("strategy_id = ?", 29).Find(&executions)

	if result.Error != nil {
		fmt.Printf("查询PnL数据失败: %v\n", result.Error)
		return
	}

	fmt.Printf("策略执行PnL分析:\n")

	totalExecutions := len(executions)
	profitableExecutions := 0
	totalPnL := 0.0
	totalOrders := 0
	totalSuccessOrders := 0

	fmt.Printf("%-5s %-12s %-8s %-8s %-8s %-12s\n",
		"执行ID", "日期", "总订单", "成功", "失败", "PnL")

	for _, exec := range executions {
		failedOrders := exec.TotalOrders - exec.SuccessOrders

		if exec.TotalPnL > 0 {
			profitableExecutions++
		}

		totalPnL += exec.TotalPnL
		totalOrders += exec.TotalOrders
		totalSuccessOrders += exec.SuccessOrders

		fmt.Printf("%-5d %-12s %-8d %-8d %-8d %-12.4f\n",
			exec.ID,
			exec.CreatedAt.Format("01-02"),
			exec.TotalOrders,
			exec.SuccessOrders,
			failedOrders,
			exec.TotalPnL)
	}

	fmt.Printf("\n汇总统计:\n")
	fmt.Printf("总执行次数: %d\n", totalExecutions)
	fmt.Printf("盈利执行次数: %d\n", profitableExecutions)
	if totalExecutions > 0 {
		fmt.Printf("胜率: %.1f%%\n", float64(profitableExecutions)/float64(totalExecutions)*100)
	}
	fmt.Printf("总PnL: %.4f USDT\n", totalPnL)
	fmt.Printf("总订单数: %d\n", totalOrders)
	fmt.Printf("成功订单数: %d\n", totalSuccessOrders)
	if totalOrders > 0 {
		fmt.Printf("订单成功率: %.1f%%\n", float64(totalSuccessOrders)/float64(totalOrders)*100)
	}

	// 分析最近7天的表现
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var recentOrders []ScheduledOrder
	db.Where("strategy_id = ? AND symbol = ? AND created_at >= ?", 29, "FILUSDT", sevenDaysAgo).Find(&recentOrders)

	fmt.Printf("\n最近7天订单统计:\n")
	fmt.Printf("总订单数: %d\n", len(recentOrders))

	filled := 0
	buy := 0
	sell := 0

	for _, order := range recentOrders {
		if order.Status == "FILLED" {
			filled++
		}
		if order.Side == "BUY" {
			buy++
		} else if order.Side == "SELL" {
			sell++
		}
	}

	fmt.Printf("已成交: %d\n", filled)
	fmt.Printf("买入: %d\n", buy)
	fmt.Printf("卖出: %d\n", sell)

	if len(recentOrders) > 0 {
		fmt.Printf("成交率: %.1f%%\n", float64(filled)/float64(len(recentOrders))*100)
	}

	// 绩效评估
	fmt.Printf("\n📈 绩效评估:\n")
	if len(recentOrders) == 0 {
		fmt.Printf("❌ 完全无交易活动\n")
	} else if filled == 0 {
		fmt.Printf("⚠️ 有订单创建但全部未成交\n")
	} else if filled < 5 {
		fmt.Printf("⚠️ 交易活动较低 (7天成交%d笔)\n", filled)
	} else {
		fmt.Printf("✅ 有一定交易活动 (7天成交%d笔)\n", filled)
	}
}
