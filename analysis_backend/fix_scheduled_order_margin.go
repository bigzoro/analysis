package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	"analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🔧 定时合约保证金模式修复方案")
	fmt.Println("============================")

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

	fmt.Println("\n🎯 问题诊断:")

	// 检查FHEUSDT是否有未成交订单
	fmt.Println("1. 检查是否有未成交订单阻止保证金模式设置...")
	if code, body, err := client.SetMarginType("FHEUSDT", "ISOLATED"); err != nil || code >= 400 {
		bodyStr := string(body)
		if contains(bodyStr, "Margin type cannot be changed if there exists open orders") {
			fmt.Println("   ❌ 发现未成交订单 - 这是导致问题的根本原因")
			fmt.Println("   💡 解决方案: 等待订单成交或取消未成交订单")
		} else {
			fmt.Printf("   ❌ 其他错误: %s\n", bodyStr)
		}
	} else {
		fmt.Println("   ✅ 可以设置保证金模式 - 没有未成交订单")
	}

	fmt.Println("\n🔧 修复方案设计:")

	fmt.Println("\n✅ 方案A: 改进定时合约创建逻辑")
	fmt.Println("   修改CreateScheduledOrder函数")
	fmt.Println("   在创建订单时立即尝试设置保证金模式")
	fmt.Println("   即使失败也要记录，供后续处理")

	fmt.Println("\n✅ 方案B: 订单执行前预检查")
	fmt.Println("   在订单执行前检查是否有未成交订单")
	fmt.Println("   如果有，等待或取消后再设置保证金模式")

	fmt.Println("\n✅ 方案C: 后台监控和重试")
	fmt.Println("   启动后台goroutine定期检查")
	fmt.Println("   对设置失败的仓位自动重试设置保证金模式")

	fmt.Println("\n📝 具体实现建议:")

	fmt.Println("\n1️⃣ 修改CreateScheduledOrder:")
	fmt.Println(`   // 在保存订单后，立即尝试设置保证金模式
   if req.StrategyID != nil {
       // 异步设置保证金模式，不阻塞订单创建
       go s.trySetMarginModeForScheduledOrder(ord.ID, *req.StrategyID, req.Symbol)
   }`)

	fmt.Println("\n2️⃣ 添加预检查函数:")
	fmt.Println(`   func (s *Server) trySetMarginModeForScheduledOrder(orderID uint, strategyID uint, symbol string) {
       // 获取策略配置
       // 尝试设置保证金模式
       // 记录结果，无论成功失败
   }`)

	fmt.Println("\n3️⃣ 订单执行时再次尝试:")
	fmt.Println(`   // 在validateOrderPrerequisites之前
   // 或在执行订单后，仓位建立后
   // 再次尝试设置保证金模式`)

	fmt.Println("\n🎯 当前立即可行的临时方案:")

	fmt.Println("\n✅ 手动调整现有仓位:")
	fmt.Println("   1. 打开币安测试网网页端")
	fmt.Println("   2. 进入期货交易页面")
	fmt.Println("   3. 找到FHEUSDT仓位")
	fmt.Println("   4. 点击调整保证金模式为逐仓")

	fmt.Println("\n✅ 等待系统自动调整:")
	fmt.Println("   1. 监控订单状态")
	fmt.Println("   2. 等待所有订单完全成交")
	fmt.Println("   3. 系统会自动重试设置保证金模式")

	fmt.Println("\n📊 验证修复效果:")
	fmt.Println("   运行测试确认保证金模式已正确设置")
	fmt.Println("   检查系统日志中的设置成功记录")

	fmt.Printf("\n⏰ 方案制定时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

// 简单的字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}