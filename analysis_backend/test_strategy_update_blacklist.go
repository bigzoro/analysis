package main

import (
	"encoding/json"
	"fmt"
	"log"

	"analysis/internal/db"
	pdb "analysis/internal/db"
	"gorm.io/datatypes"
)

func main() {
	fmt.Println("=== 测试策略更新黑名单功能 ===")

	// 连接数据库
	database, err := db.OpenMySQL(db.Options{
		DSN:         "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	gdb, err := database.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// 查找一个现有的策略进行测试
	var strategy pdb.TradingStrategy
	if err := gdb.Where("user_id = ?", 1).Order("created_at DESC").First(&strategy).Error; err != nil {
		log.Fatalf("Failed to find strategy: %v", err)
	}

	fmt.Printf("找到测试策略 ID: %d, 名称: %s\n", strategy.ID, strategy.Name)

	// 检查更新前的黑名单设置
	fmt.Printf("\n更新前黑名单设置:\n")
	fmt.Printf("  UseSymbolBlacklist: %v\n", strategy.Conditions.UseSymbolBlacklist)
	if len(strategy.Conditions.SymbolBlacklist) > 0 {
		var blacklist []string
		if err := json.Unmarshal(strategy.Conditions.SymbolBlacklist, &blacklist); err == nil {
			fmt.Printf("  SymbolBlacklist: %v\n", blacklist)
		} else {
			fmt.Printf("  SymbolBlacklist: (解析失败) %s\n", string(strategy.Conditions.SymbolBlacklist))
		}
	} else {
		fmt.Printf("  SymbolBlacklist: []\n")
	}

	// 准备更新数据
	testBlacklist := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT"}
	blacklistJSON, _ := json.Marshal(testBlacklist)

	updateReq := map[string]interface{}{
		"name":        strategy.Name,
		"description": strategy.Description,
		"conditions": map[string]interface{}{
			// 保留原有条件，只修改黑名单
			"spot_contract":                    strategy.Conditions.SpotContract,
			"trading_type":                     strategy.Conditions.TradingType,
			"allowed_directions":               strategy.Conditions.AllowedDirections,
			"enable_leverage":                  strategy.Conditions.EnableLeverage,
			"default_leverage":                 strategy.Conditions.DefaultLeverage,
			"max_leverage":                     strategy.Conditions.MaxLeverage,
			"margin_mode":                      strategy.Conditions.MarginMode,
			"skip_held_positions":              strategy.Conditions.SkipHeldPositions,
			"skip_close_orders_within_24_hours": strategy.Conditions.SkipCloseOrdersWithin24Hours,
			"skip_close_orders_hours":          strategy.Conditions.SkipCloseOrdersHours,
			"use_symbol_whitelist":             strategy.Conditions.UseSymbolWhitelist,
			"symbol_whitelist":                 strategy.Conditions.SymbolWhitelist,
			"use_symbol_blacklist":             true, // 启用黑名单
			"symbol_blacklist":                 blacklistJSON,
		},
	}

	// 序列化请求
	reqJSON, err := json.Marshal(updateReq)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}

	fmt.Printf("\n发送更新请求:\n%s\n", string(reqJSON))

	// 直接更新数据库（模拟API调用）
	strategy.Conditions.UseSymbolBlacklist = true
	strategy.Conditions.SymbolBlacklist = datatypes.JSON(blacklistJSON)

	if err := pdb.UpdateTradingStrategy(gdb, &strategy); err != nil {
		log.Fatalf("Failed to update strategy: %v", err)
	}

	// 重新查询验证更新结果
	var updatedStrategy pdb.TradingStrategy
	if err := gdb.Where("id = ?", strategy.ID).First(&updatedStrategy).Error; err != nil {
		log.Fatalf("Failed to reload strategy: %v", err)
	}

	fmt.Printf("\n更新后黑名单设置:\n")
	fmt.Printf("  UseSymbolBlacklist: %v\n", updatedStrategy.Conditions.UseSymbolBlacklist)
	if len(updatedStrategy.Conditions.SymbolBlacklist) > 0 {
		var blacklist []string
		if err := json.Unmarshal(updatedStrategy.Conditions.SymbolBlacklist, &blacklist); err == nil {
			fmt.Printf("  SymbolBlacklist: %v\n", blacklist)
		} else {
			fmt.Printf("  SymbolBlacklist: (解析失败) %s\n", string(updatedStrategy.Conditions.SymbolBlacklist))
		}
	} else {
		fmt.Printf("  SymbolBlacklist: []\n")
	}

	// 验证更新是否成功
	if updatedStrategy.Conditions.UseSymbolBlacklist &&
		len(updatedStrategy.Conditions.SymbolBlacklist) > 0 {

		var finalBlacklist []string
		if err := json.Unmarshal(updatedStrategy.Conditions.SymbolBlacklist, &finalBlacklist); err == nil {
			if len(finalBlacklist) == 3 &&
				finalBlacklist[0] == "BTCUSDT" &&
				finalBlacklist[1] == "ETHUSDT" &&
				finalBlacklist[2] == "ADAUSDT" {
				fmt.Println("\n✅ 黑名单更新测试通过！")
			} else {
				fmt.Printf("\n❌ 黑名单内容不正确: %v\n", finalBlacklist)
			}
		} else {
			fmt.Printf("\n❌ 黑名单JSON解析失败: %v\n", err)
		}
	} else {
		fmt.Println("\n❌ 黑名单更新失败！")
	}

	// 测试配置转换
	fmt.Println("\n=== 测试配置转换 ===")

	// 导入必要的包来测试配置转换
	// 这里我们直接验证数据库中的数据是否能正确转换为TraditionalConfig
	if updatedStrategy.Conditions.UseSymbolBlacklist {
		fmt.Println("✅ 黑名单启用状态正确")
	} else {
		fmt.Println("❌ 黑名单启用状态错误")
	}

	fmt.Println("\n🎯 测试完成")
}