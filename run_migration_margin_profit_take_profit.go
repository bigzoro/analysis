package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("🔄 执行数据库迁移: 添加保证金盈利止盈字段")

	// 读取SQL文件
	sqlContent, err := os.ReadFile("migrations/033_add_margin_profit_take_profit_fields.sql")
	if err != nil {
		log.Fatalf("读取SQL文件失败: %v", err)
	}

	fmt.Println("SQL迁移内容:")
	fmt.Println(string(sqlContent))

	// 这里应该连接数据库并执行SQL
	// 由于这是一个示例，我们只显示SQL内容
	fmt.Println("\n⚠️  请手动执行上述SQL语句来完成数据库迁移")
	fmt.Println("   或者将此SQL文件放到你的数据库迁移工具中执行")

	fmt.Println("\n📋 新增字段说明:")
	fmt.Println("   enable_margin_profit_take_profit: 是否启用基于保证金盈利的止盈机制")
	fmt.Println("   margin_profit_take_profit_percent: 当保证金盈利达到此百分比时触发止盈")

	fmt.Println("\n🎉 保证金盈利止盈功能数据库迁移准备完成!")
}
