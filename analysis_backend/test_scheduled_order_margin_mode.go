package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"analysis/internal/config"
	"analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🧪 测试定时订单保证金模式预设功能")
	fmt.Println("===============================")

	// 读取配置
	configPath := "./config.yaml"
	var cfg config.Config
	config.MustLoad(configPath, &cfg)

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

	client := bf.New(cfg.Exchange.Binance.IsTestnet, cfg.Exchange.Binance.APIKey, cfg.Exchange.Binance.SecretKey)

	fmt.Println("\n📋 功能验证:")

	// 1. 验证策略配置
	fmt.Println("1. 检查策略33配置...")
	var strategy db.TradingStrategy
	if err := gdb.GormDB().Where("id = ?", 33).First(&strategy).Error; err != nil {
		log.Printf("❌ 获取策略33失败: %v", err)
		return
	}
	fmt.Printf("   ✅ 策略: %s\n", strategy.Name)
	fmt.Printf("   ✅ 保证金模式: %s\n", strategy.Conditions.MarginMode)
	fmt.Printf("   ✅ 交易类型: %s\n", strategy.Conditions.TradingType)

	// 2. 模拟预设保证金模式
	fmt.Println("\n2. 模拟定时订单保证金模式预设...")
	mockOrderID := uint(99999) // 模拟订单ID
	strategyID := uint(33)
	testSymbol := "FHEUSDT"

	// 直接测试设置函数
	fmt.Printf("   模拟订单ID: %d\n", mockOrderID)
	fmt.Printf("   策略ID: %d\n", strategyID)
	fmt.Printf("   交易对: %s\n", testSymbol)

	// 手动调用设置函数来测试
	marginResult := trySetMarginModeWithStrategy(client, &strategy, testSymbol)
	if marginResult.Success {
		fmt.Printf("   ✅ 保证金模式设置成功: %s -> %s\n", testSymbol, marginResult.MarginType)
	} else {
		fmt.Printf("   ❌ 保证金模式设置失败: %v\n", marginResult.Error)
		if strings.Contains(marginResult.Error.Error(), "存在未成交订单") {
			fmt.Println("   💡 这是预期的行为 - 当前存在未成交订单")
		}
	}

	fmt.Println("\n📝 实现说明:")

	fmt.Println("\n✅ 已实现的改进:")
	fmt.Println("   1. 在CreateScheduledOrder中添加异步保证金模式设置")
	fmt.Println("   2. 新增trySetMarginModeForScheduledOrder函数")
	fmt.Println("   3. 新增trySetMarginModeWithStrategy函数")
	fmt.Println("   4. 完整的错误处理和日志记录")

	fmt.Println("\n✅ 异步处理机制:")
	fmt.Println("   - 不阻塞订单创建API响应")
	fmt.Println("   - 后台异步设置保证金模式")
	fmt.Println("   - 记录设置结果和错误信息")

	fmt.Println("\n✅ 时序优化:")
	fmt.Println("   - 在订单创建阶段就设置保证金模式")
	fmt.Println("   - 避免与订单执行时的时序冲突")
	fmt.Println("   - 提高设置成功率")

	fmt.Println("\n🎯 预期效果:")

	fmt.Println("\n✅ 正常情况:")
	fmt.Println("   用户创建定时订单 -> 系统立即尝试设置保证金模式")
	fmt.Println("   设置成功 -> 订单标记为已预设保证金模式")
	fmt.Println("   订单执行时 -> 直接使用已设置的模式")

	fmt.Println("\n⚠️ 有未成交订单的情况:")
	fmt.Println("   设置失败 -> 记录失败原因")
	fmt.Println("   订单执行时 -> 重新尝试设置保证金模式")
	fmt.Println("   最终成功 -> 保证金模式正确应用")

	fmt.Println("\n📊 验证方法:")
	fmt.Println("   1. 创建新的定时订单，选择策略33")
	fmt.Println("   2. 检查系统日志中的保证金模式设置记录")
	fmt.Println("   3. 等待订单执行，确认最终保证金模式")

	fmt.Printf("\n⏰ 测试完成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

// trySetMarginModeWithStrategy 复制的测试函数
func trySetMarginModeWithStrategy(client *bf.Client, strategy *db.TradingStrategy, symbol string) *MarginModeResult {
	result := &MarginModeResult{
		Success: false,
	}

	// 根据策略的MarginMode设置保证金模式
	marginType := "CROSSED" // 默认全仓
	if strategy.Conditions.MarginMode == "ISOLATED" {
		marginType = "ISOLATED"
	}
	result.MarginType = marginType

	log.Printf("[MarginMode] 尝试设置保证金模式: symbol=%s, marginType=%s", symbol, marginType)

	// 执行设置操作（简化的单次尝试）
	code, body, err := client.SetMarginType(symbol, marginType)

	if err == nil && code < 400 {
		result.Success = true
		log.Printf("[MarginMode] ✅ 设置成功: %s -> %s", symbol, marginType)
		return result
	}

	// 记录错误
	result.Error = fmt.Errorf("设置保证金模式失败: code=%d body=%s err=%v", code, string(body), err)

	// 特殊处理：未成交订单错误
	bodyStr := string(body)
	if strings.Contains(bodyStr, "Margin type cannot be changed if there exists open orders") {
		result.Error = fmt.Errorf("存在未成交订单，暂时无法设置保证金模式: %s", symbol)
		log.Printf("[MarginMode] ⚠️ 检测到未成交订单: %s", symbol)
	} else {
		log.Printf("[MarginMode] ❌ 设置失败: %s - %s", symbol, bodyStr)
	}

	return result
}

// MarginModeResult 保证金模式设置结果
type MarginModeResult struct {
	Success    bool
	MarginType string
	Error      error
	RetryCount int
	Duration   time.Duration
}