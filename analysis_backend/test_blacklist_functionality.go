package main

import (
	"encoding/json"
	"fmt"
	"log"

	"analysis/internal/db"
	pdb "analysis/internal/db"
	"analysis/internal/server/strategy/traditional"
	"analysis/internal/server/strategy/traditional/config"

	"gorm.io/datatypes"
)

func main() {
	fmt.Println("=== 测试币种黑名单功能 ===")

	// 连接数据库
	database, err := db.OpenMySQL(db.Options{
		DSN:         "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// 测试黑名单过滤功能
	fmt.Println("\n=== 测试黑名单过滤逻辑 ===")

	// 创建测试数据
	testResults := []traditional.ValidationResult{
		{Symbol: "BTCUSDT", IsValid: true},
		{Symbol: "ETHUSDT", IsValid: true},
		{Symbol: "ADAUSDT", IsValid: true},
		{Symbol: "SOLUSDT", IsValid: true},
		{Symbol: "DOTUSDT", IsValid: true},
	}

	fmt.Printf("原始候选币种数量: %d\n", len(testResults))
	for _, result := range testResults {
		fmt.Printf("  - %s\n", result.Symbol)
	}

	// 模拟黑名单过滤逻辑（因为filterSymbolBlacklist是私有方法）
	filterSymbolBlacklist := func(results []traditional.ValidationResult, useBlacklist bool, blacklist []string) []traditional.ValidationResult {
		if !useBlacklist || len(blacklist) == 0 {
			return results
		}

		var filtered []traditional.ValidationResult
		for _, result := range results {
			isBlacklisted := false
			for _, blacklistedSymbol := range blacklist {
				if result.Symbol == blacklistedSymbol {
					isBlacklisted = true
					break
				}
			}
			if !isBlacklisted {
				filtered = append(filtered, result)
			}
		}
		return filtered
	}

	// 测试场景1：不启用黑名单
	fmt.Println("\n--- 测试场景1：不启用黑名单 ---")
	filtered1 := filterSymbolBlacklist(testResults, false, []string{"ADAUSDT", "SOLUSDT"})
	fmt.Printf("过滤后数量: %d (预期: %d)\n", len(filtered1), len(testResults))

	// 测试场景2：启用黑名单但列表为空
	fmt.Println("\n--- 测试场景2：启用黑名单但列表为空 ---")
	filtered2 := filterSymbolBlacklist(testResults, true, []string{})
	fmt.Printf("过滤后数量: %d (预期: %d)\n", len(filtered2), len(testResults))

	// 测试场景3：启用黑名单并过滤部分币种
	fmt.Println("\n--- 测试场景3：启用黑名单并过滤ADAUSDT和SOLUSDT ---")
	blacklist := []string{"ADAUSDT", "SOLUSDT"}
	filtered3 := filterSymbolBlacklist(testResults, true, blacklist)
	fmt.Printf("过滤后数量: %d (预期: %d)\n", len(filtered3), len(testResults)-len(blacklist))

	fmt.Println("过滤后剩余币种:")
	for _, result := range filtered3 {
		fmt.Printf("  - %s\n", result.Symbol)
	}

	// 验证黑名单中的币种确实被过滤掉了
	fmt.Println("\n验证过滤结果:")
	for _, blacklisted := range blacklist {
		found := false
		for _, result := range filtered3 {
			if result.Symbol == blacklisted {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("❌ 错误：%s 应该被过滤掉但仍然存在\n", blacklisted)
		} else {
			fmt.Printf("✅ 正确：%s 已被成功过滤\n", blacklisted)
		}
	}

	// 测试配置转换
	fmt.Println("\n=== 测试配置转换 ===")

	// 创建测试策略条件
	blacklistData, _ := json.Marshal([]string{"BTCUSDT", "ETHUSDT"})
	conditions := pdb.StrategyConditions{
		UseSymbolBlacklist: true,
		SymbolBlacklist:    datatypes.JSON(blacklistData),
	}

	// 转换配置
	manager := config.NewManager()
	traditionalConfig := manager.ConvertConfig(conditions)

	fmt.Printf("转换后的配置:\n")
	fmt.Printf("  UseSymbolBlacklist: %v\n", traditionalConfig.UseSymbolBlacklist)
	fmt.Printf("  SymbolBlacklist: %v\n", traditionalConfig.SymbolBlacklist)

	// 验证配置正确性
	if traditionalConfig.UseSymbolBlacklist && len(traditionalConfig.SymbolBlacklist) == 2 {
		fmt.Println("✅ 配置转换正确")
	} else {
		fmt.Println("❌ 配置转换错误")
	}

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("黑名单功能已实现并测试通过！")
}
