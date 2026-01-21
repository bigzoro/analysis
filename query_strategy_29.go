package main

import (
	"fmt"
	"log"
	"os/exec"
)

func main() {
	fmt.Println("=== 查询策略ID 29的数据库配置 ===")

	// 使用MySQL命令行直接查询
	cmd := exec.Command("mysql", "-h127.0.0.1", "-uroot", "-proot", "analysis", "-e",
		"SELECT id, name, conditions FROM trading_strategies WHERE id = 29;")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("执行MySQL查询失败: %v", err)
	}

	fmt.Printf("数据库查询结果:\n%s\n", string(output))

	// 尝试用JSON格式显示conditions字段
	fmt.Println("\n=== 解析网格交易相关字段 ===")
	cmd2 := exec.Command("mysql", "-h127.0.0.1", "-uroot", "-proot", "analysis", "-e",
		"SELECT id, name, JSON_EXTRACT(conditions, '$.grid_trading_enabled') as grid_enabled, JSON_EXTRACT(conditions, '$.grid_upper_price') as upper_price, JSON_EXTRACT(conditions, '$.grid_lower_price') as lower_price, JSON_EXTRACT(conditions, '$.grid_levels') as grid_levels, JSON_EXTRACT(conditions, '$.grid_investment_amount') as investment, JSON_EXTRACT(conditions, '$.use_symbol_whitelist') as use_whitelist, JSON_EXTRACT(conditions, '$.symbol_whitelist') as whitelist FROM trading_strategies WHERE id = 29;")

	output2, err := cmd2.CombinedOutput()
	if err != nil {
		log.Fatalf("执行JSON查询失败: %v", err)
	}

	fmt.Printf("JSON字段解析结果:\n%s\n", string(output2))
}
