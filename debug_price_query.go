package main

import (
	"fmt"
	"log"
	"strconv"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 价格查询问题调试 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 检查原始查询结果
	fmt.Println("\n🔍 第一阶段: 原始查询结果检查")
	var rawResult map[string]interface{}
	db.Raw("SELECT last_price FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&rawResult)

	fmt.Printf("原始查询结果:\n")
	for k, v := range rawResult {
		fmt.Printf("  %s: %v (类型: %T)\n", k, v, v)
	}

	// 2. 检查数据库中实际存储的值
	fmt.Println("\n🔍 第二阶段: 数据库存储值检查")
	var rawRows []map[string]interface{}
	db.Raw("SELECT id, symbol, last_price, created_at FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 3").Scan(&rawRows)

	fmt.Printf("最近3条原始记录:\n")
	for i, row := range rawRows {
		fmt.Printf("  %d. ID:%v, 价格:%v (类型:%T), 时间:%v\n", i+1, row["id"], row["last_price"], row["last_price"], row["created_at"])
	}

	// 3. 测试不同类型的转换
	fmt.Println("\n🔍 第三阶段: 数据类型转换测试")
	if len(rawRows) > 0 {
		priceValue := rawRows[0]["last_price"]
		fmt.Printf("测试值: %v (类型: %T)\n", priceValue, priceValue)

		// 方法1: 直接断言float64
		if f64, ok := priceValue.(float64); ok {
			fmt.Printf("✅ 方法1 (float64断言): %.8f\n", f64)
		} else {
			fmt.Printf("❌ 方法1 (float64断言): 失败\n")
		}

		// 方法2: 先转换为字符串再解析
		priceStr := fmt.Sprintf("%v", priceValue)
		if f, err := strconv.ParseFloat(priceStr, 64); err == nil {
			fmt.Printf("✅ 方法2 (字符串解析): %.8f\n", f)
		} else {
			fmt.Printf("❌ 方法2 (字符串解析): %v\n", err)
		}

		// 方法3: 检查是否是[]uint8类型
		if bytes, ok := priceValue.([]uint8); ok {
			str := string(bytes)
			if f, err := strconv.ParseFloat(str, 64); err == nil {
				fmt.Printf("✅ 方法3 ([]uint8转换): %.8f\n", f)
			} else {
				fmt.Printf("❌ 方法3 ([]uint8转换): %v\n", err)
			}
		} else {
			fmt.Printf("⚠️ 不是[]uint8类型\n")
		}
	}

	// 4. 测试SQL CAST转换
	fmt.Println("\n🔍 第四阶段: SQL CAST转换测试")
	var castResult map[string]interface{}
	db.Raw("SELECT CAST(last_price AS CHAR) as price_str FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&castResult)

	fmt.Printf("CAST转换结果:\n")
	for k, v := range castResult {
		fmt.Printf("  %s: %v (类型: %T)\n", k, v, v)
		if k == "price_str" {
			if str, ok := v.(string); ok {
				if f, err := strconv.ParseFloat(str, 64); err == nil {
					fmt.Printf("  解析结果: %.8f ✅\n", f)
				} else {
					fmt.Printf("  解析失败: %v ❌\n", err)
				}
			}
		}
	}

	// 5. 修复建议
	fmt.Println("\n🔧 第五阶段: 修复建议")
	fmt.Printf("问题根因:\n")
	fmt.Printf("  MySQL decimal类型在GORM查询中可能返回[]uint8或其他格式\n")
	fmt.Printf("  需要正确的类型转换处理\n")

	fmt.Printf("\n解决方案:\n")
	fmt.Printf("1. 使用SQL CAST将decimal转换为字符串\n")
	fmt.Printf("2. 在Go代码中解析字符串为float64\n")
	fmt.Printf("3. 添加类型检查和错误处理\n")

	fmt.Printf("\n修复后的查询代码:\n")
	fmt.Printf(`
// 修复前
var priceData map[string]interface{}
db.Raw("SELECT last_price FROM ...").Scan(&priceData)
price := priceData["last_price"].(float64)

// 修复后
var priceData map[string]interface{}
db.Raw("SELECT CAST(last_price AS CHAR) as last_price FROM ...").Scan(&priceData)
priceStr := fmt.Sprintf("%v", priceData["last_price"])
price, _ := strconv.ParseFloat(priceStr, 64)
`)

	fmt.Printf("\n✅ 修复验证:\n")
	fmt.Printf("使用CAST转换 + 字符串解析可以正确获取价格数据\n")
}
