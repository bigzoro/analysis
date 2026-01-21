package main

import (
	"fmt"
	"log"
	"time"
	pdb "analysis/internal/db"
	"analysis/internal/server"
	bf "analysis/internal/exchange/binancefutures"
	"analysis/internal/config"
)

func main() {
	fmt.Println("🧪 测试Bracket联动取消修复")
	fmt.Println("===========================")

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
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

	// 创建配置
	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			Binance: config.BinanceConfig{
				IsTestnet: true, // 测试网
				APIKey:    "",
				SecretKey: "",
			},
		},
	}

	// 创建Server实例
	s := &server.Server{
		Db:  gdb,
		Cfg: cfg,
	}

	// 创建币安客户端
	client := bf.New(cfg.Exchange.Binance.IsTestnet, cfg.Exchange.Binance.APIKey, cfg.Exchange.Binance.SecretKey)

	fmt.Println("\n1️⃣ 执行Bracket订单同步测试")

	// 记录修复前的状态
	fmt.Println("记录修复前的XNYUSDT Bracket订单状态...")
	checkXNYUSDTStatus(gdb, "修复前")

	fmt.Println("\n⏳ 请手动运行Order-Sync，然后重新运行此测试脚本检查状态变化")
	fmt.Println("   Order-Sync会自动执行Bracket联动取消逻辑")

	// 检查XNYUSDT的Bracket订单状态变化
	fmt.Println("\n2️⃣ 检查XNYUSDT Bracket订单状态")

	var bracketLink pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ? AND status = ?", "XNYUSDT", "active").First(&bracketLink).Error
	if err != nil {
		if err.Error() == "record not found" {
			fmt.Println("✅ XNYUSDT Bracket订单已被关闭或标记为orphaned")
		} else {
			log.Printf("查询Bracket订单失败: %v", err)
		}
	} else {
		fmt.Printf("❌ XNYUSDT Bracket订单仍然活跃 (ID: %d, 状态: %s)\n", bracketLink.ID, bracketLink.Status)

		// 检查TP/SL订单状态
		checkConditionalOrderStatus(gdb, bracketLink.TPClientID, "止盈")
		checkConditionalOrderStatus(gdb, bracketLink.SLClientID, "止损")
	}

	// 检查活跃的条件订单数量
	fmt.Println("\n3️⃣ 检查活跃条件订单数量")

	var activeConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status NOT IN (?)",
		"XNYUSDT", []string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"cancelled", "filled", "executed"}).Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("XNYUSDT活跃条件订单数量: %d\n", len(activeConditionalOrders))
		if len(activeConditionalOrders) == 0 {
			fmt.Println("✅ 所有条件订单都已被正确取消！")
		} else {
			fmt.Println("❌ 仍有活跃的条件订单:")
			for _, order := range activeConditionalOrders {
				fmt.Printf("   - %s (%s) - 状态: %s\n",
					order.ClientOrderId, order.OrderType, order.Status)
			}
		}
	}

	fmt.Println("\n🎉 Bracket联动取消修复测试完成！")
}

func checkXNYUSDTStatus(gdb pdb.Database, phase string) {
	fmt.Printf("\n=== %s XNYUSDT Bracket状态 ===\n", phase)

	// 检查Bracket订单
	var bracketLinks []pdb.BracketLink
	err := gdb.GormDB().Where("symbol = ?", "XNYUSDT").Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询Bracket订单失败: %v", err)
		return
	}

	fmt.Printf("Bracket订单数量: %d\n", len(bracketLinks))
	for _, link := range bracketLinks {
		fmt.Printf("  ID:%d, GroupID:%s, 状态:%s\n", link.ID, link.GroupID, link.Status)
	}

	// 检查活跃条件订单
	var activeConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status NOT IN (?)",
		"XNYUSDT", []string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"cancelled", "filled", "executed"}).Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
		return
	}

	fmt.Printf("活跃条件订单数量: %d\n", len(activeConditionalOrders))
	for _, order := range activeConditionalOrders {
		fmt.Printf("  %s (%s) - 状态:%s\n",
			order.ClientOrderId, order.OrderType, order.Status)
	}
}

func checkConditionalOrderStatus(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("   %s订单ID为空\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("   ❌ %s订单 %s 查询失败: %v\n", orderType, clientOrderId, err)
		return
	}

	fmt.Printf("   %s订单 %s - 状态: %s\n", orderType, clientOrderId, order.Status)
}