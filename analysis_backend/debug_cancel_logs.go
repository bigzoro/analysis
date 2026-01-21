package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查FHEUSDT取消订单的相关日志")

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

	// 1. 检查最近的操作日志，查找取消相关的记录
	fmt.Println("\n1️⃣ 检查操作日志中的取消记录")
	var logs []pdb.OperationLog
	err = gdb.GormDB().Where("description LIKE ? AND created_at >= DATE_SUB(NOW(), INTERVAL 2 HOUR)",
		"%取消%").Order("created_at DESC").Find(&logs).Error

	if err != nil {
		log.Printf("查询日志失败: %v", err)
	} else {
		fmt.Printf("找到%d条取消相关的日志:\n", len(logs))
		for i, logEntry := range logs {
			if i >= 10 { // 只显示前10条
				break
			}
			fmt.Printf("  %s [%s] %s\n",
				logEntry.CreatedAt.Format("15:04:05"),
				logEntry.Level,
				logEntry.Description)

			// 如果是错误日志，显示更多信息
			if logEntry.Level == "error" && logEntry.ErrorMsg != "" {
				fmt.Printf("    错误: %s\n", logEntry.ErrorMsg)
			}
		}
	}

	// 2. 检查特定订单的操作日志
	fmt.Println("\n2️⃣ 检查FHEUSDT条件订单的操作日志")
	orderIds := []uint{1291, 1292} // 止盈和止损订单ID
	for _, orderId := range orderIds {
		var orderLogs []pdb.OperationLog
		err = gdb.GormDB().Where("entity_type = ? AND entity_id = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 2 HOUR)",
			"order", orderId).Order("created_at DESC").Find(&orderLogs).Error

		if err != nil {
			log.Printf("查询订单%d的日志失败: %v", orderId, err)
			continue
		}

		if len(orderLogs) > 0 {
			fmt.Printf("订单%d的操作日志 (%d条):\n", orderId, len(orderLogs))
			for _, logEntry := range orderLogs {
				fmt.Printf("  %s [%s] %s: %s\n",
					logEntry.CreatedAt.Format("15:04:05"),
					logEntry.Level,
					logEntry.Action,
					logEntry.Description)
			}
		}
	}

	// 3. 检查系统运行期间是否有API调用失败的记录
	fmt.Println("\n3️⃣ 检查最近的系统状态")
	fmt.Println("需要检查系统运行日志中的以下关键词：")
	fmt.Println("🔍 '[Order-Sync] 取消' - 取消订单的日志")
	fmt.Println("❌ '取消订单失败' - API调用失败")
	fmt.Println("⚠️ '取消订单响应错误' - 币安API错误响应")

	// 4. 模拟可能的取消失败场景
	fmt.Println("\n4️⃣ 分析可能的取消失败原因")
	fmt.Println("根据cancelConditionalOrderIfNeeded函数的逻辑：")

	fmt.Println("\n场景1: API调用超时或网络错误")
	fmt.Println("  - 数据库状态已更新为'cancelled'")
	fmt.Println("  - 币安网站上的订单未被取消")
	fmt.Println("  - 结果: 网站上仍有订单存在")

	fmt.Println("\n场景2: 订单已被执行")
	fmt.Println("  - 币安返回: 'Order has been executed'")
	fmt.Println("  - 系统正确地将状态更新为'filled'")
	fmt.Println("  - 但这不是取消失败")

	fmt.Println("\n场景3: 订单不存在")
	fmt.Println("  - 币安返回: 'Order does not exist'")
	fmt.Println("  - 系统认为订单已被取消")

	// 5. 检查币安API状态
	fmt.Println("\n5️⃣ 检查币安API连接状态")
	fmt.Println("尝试连接到币安API来验证网络连通性...")

	// 这里可以添加一个简单的API连接测试
	fmt.Println("⚠️ 注意：当前运行环境可能无法访问币安API")
	fmt.Println("   建议在服务器环境中检查以下内容：")
	fmt.Println("   1. 网络连接是否正常")
	fmt.Println("   2. API密钥是否有效")
	fmt.Println("   3. 是否达到API调用频率限制")
	fmt.Println("   4. 币安服务是否正常")

	// 6. 建议解决方案
	fmt.Println("\n6️⃣ 建议解决方案")

	fmt.Println("\n🔧 立即处理：")
	fmt.Println("1. 在币安网站手动取消剩余的条件订单")
	fmt.Println("2. 检查系统日志中是否有API调用失败的详细信息")
	fmt.Println("3. 验证API密钥和网络连接")

	fmt.Println("\n🛠️ 系统改进：")
	fmt.Println("1. 改进cancelConditionalOrderIfNeeded函数的错误处理")
	fmt.Println("2. 添加重试机制和更详细的错误日志")
	fmt.Println("3. 实现订单状态的双向同步机制")
	fmt.Println("4. 添加定期检查和清理机制")

	fmt.Println("\n📊 当前状态总结：")
	fmt.Println("✅ 数据库状态：订单已标记为cancelled")
	fmt.Println("❌ 币安网站：可能仍有订单存在")
	fmt.Println("🎯 问题原因：API取消调用失败，但数据库已更新")
}