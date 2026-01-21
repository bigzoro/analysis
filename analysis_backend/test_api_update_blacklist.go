package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"analysis/internal/db"
	pdb "analysis/internal/db"
	"analysis/internal/server/strategy/traditional/config"
	"gorm.io/datatypes"
)

func main() {
	fmt.Println("=== API黑名单更新功能验证 ===")

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

	// 查找策略ID 33
	var strategy pdb.TradingStrategy
	if err := gdb.Where("id = ? AND user_id = ?", 33, 1).First(&strategy).Error; err != nil {
		log.Fatalf("Failed to find strategy 33: %v", err)
	}

	fmt.Printf("策略ID: %d, 名称: %s\n", strategy.ID, strategy.Name)

	// 记录更新前的状态
	fmt.Printf("\n更新前状态:\n")
	fmt.Printf("  UseSymbolBlacklist: %v\n", strategy.Conditions.UseSymbolBlacklist)
	fmt.Printf("  SymbolBlacklist长度: %d\n", len(strategy.Conditions.SymbolBlacklist))

	// 模拟API调用 - 更新黑名单
	fmt.Printf("\n模拟API更新操作...\n")

	// 准备新的黑名单数据
	testBlacklist := []string{"SOLUSDT", "DOTUSDT", "LINKUSDT"}
	blacklistJSON, _ := json.Marshal(testBlacklist)

	// 更新策略条件
	strategy.Conditions.UseSymbolBlacklist = true
	strategy.Conditions.SymbolBlacklist = datatypes.JSON(blacklistJSON)
	strategy.UpdatedAt = time.Now()

	// 执行数据库更新
	if err := pdb.UpdateTradingStrategy(gdb, &strategy); err != nil {
		log.Fatalf("Failed to update strategy: %v", err)
	}

	fmt.Printf("✅ 数据库更新成功\n")

	// 重新加载策略验证
	var updatedStrategy pdb.TradingStrategy
	if err := gdb.Where("id = ?", 33).First(&updatedStrategy).Error; err != nil {
		log.Fatalf("Failed to reload strategy: %v", err)
	}

	fmt.Printf("\n更新后状态:\n")
	fmt.Printf("  UseSymbolBlacklist: %v\n", updatedStrategy.Conditions.UseSymbolBlacklist)

	if len(updatedStrategy.Conditions.SymbolBlacklist) > 0 {
		var blacklist []string
		if err := json.Unmarshal(updatedStrategy.Conditions.SymbolBlacklist, &blacklist); err == nil {
			fmt.Printf("  SymbolBlacklist: %v\n", blacklist)

			// 验证内容
			if len(blacklist) == 3 &&
				blacklist[0] == "SOLUSDT" &&
				blacklist[1] == "DOTUSDT" &&
				blacklist[2] == "LINKUSDT" {
				fmt.Printf("✅ 黑名单内容更新正确\n")
			} else {
				fmt.Printf("❌ 黑名单内容不匹配\n")
			}
		} else {
			fmt.Printf("❌ 黑名单JSON解析失败: %v\n", err)
		}
	} else {
		fmt.Printf("❌ 黑名单为空\n")
	}

	// 测试配置转换是否正常工作
	fmt.Printf("\n测试配置转换:\n")

	// 导入配置管理器
	manager := config.NewManager()
	traditionalConfig := manager.ConvertConfig(updatedStrategy.Conditions)

	fmt.Printf("  UseSymbolBlacklist: %v\n", traditionalConfig.UseSymbolBlacklist)
	fmt.Printf("  SymbolBlacklist长度: %d\n", len(traditionalConfig.SymbolBlacklist))

	if traditionalConfig.UseSymbolBlacklist && len(traditionalConfig.SymbolBlacklist) == 3 {
		fmt.Printf("✅ 配置转换成功\n")
	} else {
		fmt.Printf("❌ 配置转换失败\n")
	}

	fmt.Printf("\n🎯 API黑名单更新功能验证完成\n")
}