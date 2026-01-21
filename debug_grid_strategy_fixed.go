package main

import (
	"fmt"
	"log"
	"strconv"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 网格策略调试 (修复版) ===")
	fmt.Println("分析策略执行日志和阈值调整效果")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 检查策略执行记录
	fmt.Println("\n📊 第一阶段: 策略执行记录分析")
	var executions []map[string]interface{}
	db.Raw(`
		SELECT id, total_orders, success_orders, failed_orders, total_pnl, win_rate, created_at
		FROM strategy_executions
		WHERE strategy_id = 29
		ORDER BY created_at DESC
		LIMIT 5
	`).Scan(&executions)

	fmt.Printf("最近5次策略执行:\n")
	for _, exec := range executions {
		fmt.Printf("执行ID: %v, 订单: %v, 成功: %v, 失败: %v, PnL: %v, 胜率: %v%%, 时间: %v\n",
			exec["id"], exec["total_orders"], exec["success_orders"],
			exec["failed_orders"], exec["total_pnl"], exec["win_rate"], exec["created_at"])
	}

	// 2. 检查FILUSDT价格 (修复版本)
	fmt.Println("\n📊 第二阶段: FILUSDT价格检查")
	var priceRows []map[string]interface{}
	db.Raw("SELECT last_price FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&priceRows)

	currentPrice := 0.0
	if len(priceRows) > 0 {
		priceStr := fmt.Sprintf("%v", priceRows[0]["last_price"])
		if p, err := strconv.ParseFloat(priceStr, 64); err == nil {
			currentPrice = p
		}
	}

	fmt.Printf("FILUSDT当前价格: %.4f\n", currentPrice)

	// 3. 检查策略配置
	fmt.Println("\n📊 第三阶段: 策略配置检查")
	var strategy map[string]interface{}
	db.Raw(`
		SELECT grid_trading_enabled, grid_upper_price, grid_lower_price, grid_levels,
			   grid_investment_amount, grid_stop_loss_enabled
		FROM trading_strategies WHERE id = 29
	`).Scan(&strategy)

	fmt.Printf("策略配置:\n")
	gridUpper := 0.0
	gridLower := 0.0
	gridLevels := 0

	for k, v := range strategy {
		fmt.Printf("  %s: %v\n", k, v)

		if k == "grid_upper_price" {
			if str, ok := v.(string); ok {
				if p, err := strconv.ParseFloat(str, 64); err == nil {
					gridUpper = p
				}
			}
		}
		if k == "grid_lower_price" {
			if str, ok := v.(string); ok {
				if p, err := strconv.ParseFloat(str, 64); err == nil {
					gridLower = p
				}
			}
		}
		if k == "grid_levels" {
			if i, ok := v.(int64); ok {
				gridLevels = int(i)
			}
		}
	}

	fmt.Printf("\n解析后的配置:\n")
	fmt.Printf("网格范围: [%.4f, %.4f]\n", gridLower, gridUpper)
	fmt.Printf("网格层数: %d\n", gridLevels)

	// 4. 检查价格是否在网格范围内
	fmt.Println("\n📊 第四阶段: 网格范围检查")
	if currentPrice >= gridLower && currentPrice <= gridUpper && currentPrice > 0 {
		fmt.Printf("✅ 价格%.4f在网格范围内[%.4f, %.4f]\n", currentPrice, gridLower, gridUpper)

		// 计算网格位置
		gridSpacing := (gridUpper - gridLower) / float64(gridLevels)
		gridLevel := int((currentPrice - gridLower) / gridSpacing)
		if gridLevel >= gridLevels {
			gridLevel = gridLevels - 1
		}
		if gridLevel < 0 {
			gridLevel = 0
		}

		fmt.Printf("网格层级: %d/%d\n", gridLevel, gridLevels)
		fmt.Printf("网格间距: %.6f\n", gridSpacing)

		// 计算理论评分
		midLevel := gridLevels / 2
		gridScore := calculateGridScore(gridLevel, midLevel, gridLevels)
		techScore := 0.6 // 基于之前的分析
		totalScore := gridScore*0.4 + techScore*0.3

		fmt.Printf("\n理论评分计算:\n")
		fmt.Printf("网格评分: %.3f\n", gridScore)
		fmt.Printf("技术评分: %.3f\n", techScore)
		fmt.Printf("综合评分: %.3f\n", totalScore)

		fmt.Printf("\n阈值判断:\n")
		fmt.Printf("调整前阈值: >0.5 (强烈买入)\n")
		fmt.Printf("调整后阈值: >0.2 (买入)\n")

		if totalScore > 0.5 {
			fmt.Printf("调整前: ✅ 触发交易\n")
		} else {
			fmt.Printf("调整前: ❌ 不触发交易\n")
		}

		if totalScore > 0.2 {
			fmt.Printf("调整后: ✅ 触发交易\n")
		} else {
			fmt.Printf("调整后: ❌ 不触发交易\n")
		}

	} else {
		fmt.Printf("❌ 价格%.4f超出网格范围[%.4f, %.4f]\n", currentPrice, gridLower, gridUpper)
		if currentPrice == 0 {
			fmt.Printf("💡 原因: 价格数据获取失败\n")
		}
	}

	// 5. 总结分析
	fmt.Println("\n📊 第五阶段: 问题诊断和解决方案")

	if currentPrice == 0 {
		fmt.Printf("🔍 核心问题: FILUSDT价格数据获取失败\n")
		fmt.Printf("📋 影响: 策略无法判断价格是否在网格范围内\n")
		fmt.Printf("🎯 结果: 策略返回'no_op'，不创建订单\n")

		fmt.Printf("\n🔧 解决方案:\n")
		fmt.Printf("1. 检查数据同步服务是否正常运行\n")
		fmt.Printf("2. 验证币安API连接是否正常\n")
		fmt.Printf("3. 确认数据库中的价格数据格式\n")
		fmt.Printf("4. 修复价格查询的类型转换问题\n")

	} else if currentPrice < gridLower || currentPrice > gridUpper {
		fmt.Printf("🔍 核心问题: FILUSDT价格超出网格范围\n")
		fmt.Printf("📋 当前价格: %.4f\n", currentPrice)
		fmt.Printf("📋 网格范围: [%.4f, %.4f]\n", gridLower, gridUpper)
		fmt.Printf("🎯 结果: 策略等待价格回档，不创建订单\n")

		fmt.Printf("\n🔧 解决方案:\n")
		fmt.Printf("1. 调整网格范围以包含当前价格\n")
		fmt.Printf("2. 等待价格回到网格范围内\n")
		fmt.Printf("3. 启用网格自动调整功能\n")

	} else {
		fmt.Printf("🔍 分析结果: 价格在范围内，阈值调整应该生效\n")
		fmt.Printf("📋 检查: 确保代码修改已正确部署\n")
		fmt.Printf("🎯 预期: 策略应该产生交易信号\n")
	}
}

func calculateGridScore(currentLevel, midLevel, totalLevels int) float64 {
	if currentLevel < midLevel {
		return 1.0 - float64(currentLevel)/float64(midLevel)
	} else if currentLevel > midLevel {
		return -1.0 * (float64(currentLevel-midLevel) / float64(totalLevels-midLevel))
	}
	return 0
}
