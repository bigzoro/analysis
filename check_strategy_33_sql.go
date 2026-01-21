package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🔍 检查策略ID 33的市值过滤设置 (SQL查询)")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}
	defer db.Close()

	// 查询策略的基本信息
	fmt.Println("\n📊 查询策略基本信息:")
	query := "SELECT id, name, user_id FROM trading_strategies WHERE id = 33"
	row := db.QueryRow(query)

	var id int
	var name string
	var userID int
	err = row.Scan(&id, &name, &userID)
	if err != nil {
		log.Printf("❌ 查询策略失败: %v", err)
		return
	}

	fmt.Printf("   策略ID: %d\n", id)
	fmt.Printf("   策略名称: %s\n", name)
	fmt.Printf("   用户ID: %d\n", userID)

	// 查询市值过滤字段
	fmt.Println("\n🎯 查询合约涨幅开空策略市值过滤字段:")
	query = `SELECT
		futures_price_short_strategy_enabled,
		futures_price_short_min_market_cap,
		futures_price_short_max_rank,
		futures_price_short_min_funding_rate,
		futures_price_short_leverage
	FROM trading_strategies WHERE id = 33`

	row = db.QueryRow(query)

	var enabled bool
	var minMarketCap float64
	var maxRank int
	var minFundingRate float64
	var leverage float64

	err = row.Scan(&enabled, &minMarketCap, &maxRank, &minFundingRate, &leverage)
	if err != nil {
		log.Printf("❌ 查询市值过滤字段失败: %v", err)
		return
	}

	fmt.Printf("   合约涨幅开空策略启用: %v\n", enabled)
	fmt.Printf("   最低市值要求: %.0f万\n", minMarketCap)
	fmt.Printf("   最大排名限制: %d\n", maxRank)
	fmt.Printf("   最低资金费率: %.4f%%\n", minFundingRate*100)
	fmt.Printf("   开空杠杆倍数: %.1f\n", leverage)

	// 检查原始JSON数据
	fmt.Println("\n🔍 查询原始conditions JSON数据:")
	query = "SELECT conditions FROM trading_strategies WHERE id = 33"
	row = db.QueryRow(query)

	var conditions string
	err = row.Scan(&conditions)
	if err != nil {
		log.Printf("❌ 查询conditions失败: %v", err)
	} else {
		fmt.Printf("   原始JSON: %s\n", conditions)
	}

	fmt.Printf("\n🎯 问题分析:\n")
	if minMarketCap == 0 {
		fmt.Printf("   ❌ 市值过滤条件为0，意味着不限制市值大小\n")
		fmt.Printf("   💡 这就是为什么日志显示 '41377749800万 ≥ 0万' 的原因\n")
		fmt.Printf("   🔧 解决方案: 在前端界面设置市值过滤条件为大于0的值\n")
	} else {
		fmt.Printf("   ✅ 市值过滤条件正常: %.0f万\n", minMarketCap)
	}

	fmt.Printf("\n📋 建议操作:\n")
	fmt.Printf("1. 在前端策略配置页面启用合约涨幅开空策略\n")
	fmt.Printf("2. 设置市值过滤条件 (例如: 1000万)\n")
	fmt.Printf("3. 保存策略配置\n")
	fmt.Printf("4. 重新测试策略验证\n")
}