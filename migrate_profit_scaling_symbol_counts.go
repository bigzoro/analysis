package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("开始添加profit_scaling_symbol_counts字段到trading_strategies表...")

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

	// 检查字段是否已存在
	var result struct {
		FieldExists int
	}

	checkQuery := `
		SELECT COUNT(*) as field_exists
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = 'trading_strategies'
		AND COLUMN_NAME = 'profit_scaling_symbol_counts'
	`

	if err := gdb.DB().Raw(checkQuery).Scan(&result).Error; err != nil {
		log.Fatalf("检查字段是否存在失败: %v", err)
	}

	if result.FieldExists > 0 {
		fmt.Println("字段 profit_scaling_symbol_counts 已存在，跳过迁移")
		return
	}

	// 添加新字段
	addColumnQuery := `
		ALTER TABLE trading_strategies
		ADD COLUMN profit_scaling_symbol_counts JSON DEFAULT ('{}')
		COMMENT '各币种的盈利加仓计数器，格式：{"BTCUSDT": 1, "ETHUSDT": 0}'
	`

	if err := gdb.DB().Exec(addColumnQuery).Error; err != nil {
		log.Fatalf("添加字段失败: %v", err)
	}

	fmt.Println("✅ 成功添加 profit_scaling_symbol_counts 字段")

	// 可选：迁移现有数据（如果有策略当前有加仓计数，将其迁移到一个默认币种）
	fmt.Println("检查是否有需要迁移的现有数据...")

	var strategiesWithCounts []struct {
		ID                        uint
		ProfitScalingCurrentCount int
	}

	if err := gdb.DB().Table("trading_strategies").
		Where("profit_scaling_current_count > 0").
		Select("id, profit_scaling_current_count").
		Find(&strategiesWithCounts).Error; err != nil {
		log.Printf("查询现有计数器数据失败: %v", err)
	} else if len(strategiesWithCounts) > 0 {
		fmt.Printf("发现 %d 个策略有现有的加仓计数需要迁移\n", len(strategiesWithCounts))

		for _, strategy := range strategiesWithCounts {
			// 查找该策略是否有实际的加仓订单，以确定币种
			var orderSymbol struct {
				Symbol string
			}

			if err := gdb.DB().Table("scheduled_orders").
				Where("strategy_id = ? AND client_order_id LIKE ?", strategy.ID, "PROFIT_SCALING_%").
				Select("symbol").
				Limit(1).
				Scan(&orderSymbol).Error; err != nil || orderSymbol.Symbol == "" {
				// 如果找不到加仓订单，使用默认值
				orderSymbol.Symbol = "UNKNOWN"
			}

			// 更新JSON字段
			updateQuery := `
				UPDATE trading_strategies
				SET profit_scaling_symbol_counts = JSON_OBJECT(?, ?)
				WHERE id = ?
			`

			if err := gdb.DB().Exec(updateQuery, orderSymbol.Symbol, strategy.ProfitScalingCurrentCount, strategy.ID).Error; err != nil {
				log.Printf("迁移策略 %d 的计数器失败: %v", strategy.ID, err)
			} else {
				fmt.Printf("✅ 迁移策略 %d: %s = %d\n", strategy.ID, orderSymbol.Symbol, strategy.ProfitScalingCurrentCount)
			}
		}
	} else {
		fmt.Println("没有发现需要迁移的现有数据")
	}

	fmt.Println("🎉 数据库迁移完成！")
	fmt.Println("\n新的功能特性：")
	fmt.Println("• 每个币种可以独立进行最多N次加仓")
	fmt.Println("• 一个币种的加仓不会影响其他币种")
	fmt.Println("• 整体止损/止盈只重置该币种的计数器")
	fmt.Println("• 策略停止时重置所有币种的计数器")
}
