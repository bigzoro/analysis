package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type TradingOrder struct {
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

type StrategyExecution struct {
	ID              uint      `json:"id"`
	UserID          uint      `json:"user_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	SpotContract    bool      `json:"spot_contract"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Duration        int64     `json:"duration"`
	TotalOrders     int       `json:"total_orders"`
	SuccessOrders   int       `json:"success_orders"`
	FailedOrders    int       `json:"failed_orders"`
	TotalPnL        float64   `json:"total_pnl"`
	WinRate         float64   `json:"win_rate"`
	Logs            string    `json:"logs"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CurrentStep     string    `json:"current_step"`
	StepProgress    int       `json:"step_progress"`
	TotalProgress   int       `json:"total_progress"`
	CurrentSymbol   string    `json:"current_symbol"`
	ErrorMessage    string    `json:"error_message"`
	RunInterval     int       `json:"run_interval"`
	MaxRuns         int       `json:"max_runs"`
	AutoStop        bool      `json:"auto_stop"`
	CreateOrders    bool      `json:"create_orders"`
	RunCount        int       `json:"run_count"`
	PnLPercentage   float64   `json:"pnl_percentage"`
	TotalInvestment float64   `json:"total_investment"`
	CurrentValue    float64   `json:"current_value"`
	EnableLeverage  bool      `json:"enable_leverage"`
	AllowedDirs     string    `json:"allowed_directions"`
	ExecutionDelay  int64     `json:"execution_delay"`
}

func main() {
	fmt.Println("=== 网格交易策略绩效分析 ===")
	fmt.Println("分析FIL网格策略的交易时间间隔、交易次数和盈利情况")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 分析策略执行记录
	fmt.Println("\n📊 第一阶段: 策略执行记录分析")
	analyzeStrategyExecutions(db)

	// 2. 分析交易订单详情
	fmt.Println("\n📊 第二阶段: 交易订单详细分析")
	analyzeTradingOrders(db)

	// 3. 分析交易时间间隔
	fmt.Println("\n📊 第三阶段: 交易时间间隔分析")
	analyzeTradingIntervals(db)

	// 4. 分析盈利情况
	fmt.Println("\n📊 第四阶段: 盈利情况分析")
	analyzeProfitability(db)

	// 5. 综合绩效评估
	fmt.Println("\n📊 第五阶段: 综合绩效评估")
	comprehensiveAnalysis(db)
}

func analyzeStrategyExecutions(db *gorm.DB) {
	var executions []StrategyExecution
	result := db.Where("strategy_id = ?", 29).Order("created_at DESC").Limit(10).Find(&executions)

	if result.Error != nil {
		fmt.Printf("查询策略执行记录失败: %v\n", result.Error)
		return
	}

	fmt.Printf("策略ID 29的执行记录 (最近10条):\n")
	fmt.Printf("%-5s %-20s %-10s %-8s %-8s %-8s %-12s %-8s\n",
		"ID", "开始时间", "状态", "总订单", "成功", "失败", "PnL", "胜率")

	totalExecutions := len(executions)
	totalOrders := 0
	totalSuccess := 0
	totalFailed := 0
	totalPnL := 0.0

	for _, exec := range executions {
		status := "完成"
		if exec.CurrentStep != "completed" && exec.CurrentStep != "" {
			status = exec.CurrentStep
		}

		fmt.Printf("%-5d %-20s %-10s %-8d %-8d %-8d %-12.4f %-8.1f%%\n",
			exec.ID,
			exec.StartTime.Format("01-02 15:04"),
			status,
			exec.TotalOrders,
			exec.SuccessOrders,
			exec.FailedOrders,
			exec.TotalPnL,
			exec.WinRate)

		totalOrders += exec.TotalOrders
		totalSuccess += exec.SuccessOrders
		totalFailed += exec.FailedOrders
		totalPnL += exec.TotalPnL
	}

	fmt.Printf("\n执行统计汇总:\n")
	fmt.Printf("总执行次数: %d\n", totalExecutions)
	fmt.Printf("平均每次订单: %.1f\n", float64(totalOrders)/float64(totalExecutions))
	fmt.Printf("成功率: %.1f%%\n", float64(totalSuccess)/float64(totalOrders)*100)
	fmt.Printf("总PnL: %.4f USDT\n", totalPnL)
}

func analyzeTradingOrders(db *gorm.DB) {
	var orders []TradingOrder
	result := db.Where("strategy_id = ? AND symbol = ?", 29, "FILUSDT").Order("created_at DESC").Limit(50).Find(&orders)

	if result.Error != nil {
		fmt.Printf("查询交易订单失败: %v\n", result.Error)
		return
	}

	fmt.Printf("FIL网格策略的交易订单 (最近50条):\n")
	fmt.Printf("%-5s %-12s %-6s %-10s %-12s %-8s %-12s\n",
		"ID", "时间", "方向", "状态", "数量", "价格", "网格层")

	buyOrders := 0
	sellOrders := 0
	filledOrders := 0

	for _, order := range orders {
		side := order.Side
		if side == "BUY" {
			buyOrders++
		} else if side == "SELL" {
			sellOrders++
		}

		if order.Status == "FILLED" {
			filledOrders++
		}

		fmt.Printf("%-5d %-12s %-6s %-10s %-12s %-8s %-12d\n",
			order.ID,
			order.CreatedAt.Format("01-02 15:04"),
			side,
			order.Status,
			order.Quantity,
			order.Price,
			order.GridLevel)
	}

	fmt.Printf("\n订单统计:\n")
	fmt.Printf("总订单数: %d\n", len(orders))
	fmt.Printf("买入订单: %d\n", buyOrders)
	fmt.Printf("卖出订单: %d\n", sellOrders)
	fmt.Printf("已成交订单: %d\n", filledOrders)
	fmt.Printf("成交率: %.1f%%\n", float64(filledOrders)/float64(len(orders))*100)
}

func analyzeTradingIntervals(db *gorm.DB) {
	var orders []TradingOrder
	result := db.Where("strategy_id = ? AND symbol = ? AND status = ?", 29, "FILUSDT", "FILLED").
		Order("created_at ASC").Find(&orders)

	if result.Error != nil || len(orders) < 2 {
		fmt.Printf("查询交易间隔失败或订单不足: %v\n", result.Error)
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
		fmt.Printf("总观察时间: %v\n", orders[len(orders)-1].CreatedAt.Sub(orders[0].CreatedAt))
		fmt.Printf("每小时交易频率: %.2f 次\n", float64(len(orders))/orders[len(orders)-1].CreatedAt.Sub(orders[0].CreatedAt).Hours())
	}
}

func analyzeProfitability(db *gorm.DB) {
	var executions []StrategyExecution
	result := db.Where("strategy_id = ?", 29).Find(&executions)

	if result.Error != nil {
		fmt.Printf("查询盈利数据失败: %v\n", result.Error)
		return
	}

	fmt.Printf("盈利情况分析:\n")

	totalExecutions := len(executions)
	profitableExecutions := 0
	totalPnL := 0.0
	totalInvestment := 0.0
	totalOrders := 0
	totalSuccessOrders := 0

	for _, exec := range executions {
		if exec.TotalPnL > 0 {
			profitableExecutions++
		}
		totalPnL += exec.TotalPnL
		totalInvestment += exec.TotalInvestment
		totalOrders += exec.TotalOrders
		totalSuccessOrders += exec.SuccessOrders
	}

	fmt.Printf("总执行次数: %d\n", totalExecutions)
	fmt.Printf("盈利执行次数: %d\n", profitableExecutions)
	fmt.Printf("胜率: %.1f%%\n", float64(profitableExecutions)/float64(totalExecutions)*100)
	fmt.Printf("总PnL: %.4f USDT\n", totalPnL)
	fmt.Printf("总投资: %.4f USDT\n", totalInvestment)
	if totalInvestment > 0 {
		fmt.Printf("总收益率: %.2f%%\n", totalPnL/totalInvestment*100)
	}
	fmt.Printf("总订单数: %d\n", totalOrders)
	fmt.Printf("成功订单数: %d\n", totalSuccessOrders)
	if totalOrders > 0 {
		fmt.Printf("订单成功率: %.1f%%\n", float64(totalSuccessOrders)/float64(totalOrders)*100)
	}
}

func comprehensiveAnalysis(db *gorm.DB) {
	fmt.Printf("综合绩效评估:\n")

	// 查询最近7天的执行情况
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var recentExecutions []StrategyExecution
	db.Where("strategy_id = ? AND created_at >= ?", 29, sevenDaysAgo).Find(&recentExecutions)

	var recentOrders []TradingOrder
	db.Where("strategy_id = ? AND created_at >= ?", 29, sevenDaysAgo).Find(&recentOrders)

	fmt.Printf("最近7天统计:\n")
	fmt.Printf("策略执行次数: %d\n", len(recentExecutions))
	fmt.Printf("交易订单总数: %d\n", len(recentOrders))

	filledOrders := 0
	buyOrders := 0
	sellOrders := 0

	for _, order := range recentOrders {
		if order.Status == "FILLED" {
			filledOrders++
		}
		if order.Side == "BUY" {
			buyOrders++
		} else if order.Side == "SELL" {
			sellOrders++
		}
	}

	fmt.Printf("已成交订单: %d\n", filledOrders)
	fmt.Printf("买入订单: %d\n", buyOrders)
	fmt.Printf("卖出订单: %d\n", sellOrders)

	if len(recentOrders) > 0 {
		fmt.Printf("成交率: %.1f%%\n", float64(filledOrders)/float64(len(recentOrders))*100)
	}

	// 评估策略表现
	if len(recentExecutions) > 0 {
		fmt.Printf("\n策略表现评估:\n")

		if filledOrders > 10 {
			fmt.Printf("✅ 交易活跃度: 高 (日均成交%.1f笔)\n", float64(filledOrders)/7.0)
		} else if filledOrders > 5 {
			fmt.Printf("⚠️ 交易活跃度: 中等 (日均成交%.1f笔)\n", float64(filledOrders)/7.0)
		} else {
			fmt.Printf("❌ 交易活跃度: 低 (日均成交%.1f笔)\n", float64(filledOrders)/7.0)
		}

		if buyOrders > sellOrders {
			fmt.Printf("📈 交易偏向: 买入为主 (买入:卖出 = %d:%d)\n", buyOrders, sellOrders)
		} else if sellOrders > buyOrders {
			fmt.Printf("📉 交易偏向: 卖出为主 (买入:卖出 = %d:%d)\n", buyOrders, sellOrders)
		} else {
			fmt.Printf("⚖️ 交易偏向: 均衡 (买入:卖出 = %d:%d)\n", buyOrders, sellOrders)
		}
	} else {
		fmt.Printf("❌ 策略表现: 无执行记录\n")
	}
}
