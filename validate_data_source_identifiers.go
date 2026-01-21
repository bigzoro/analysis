package main

import (
	"fmt"
	"log"
	"strings"
)

// 验证所有data_source标识符的长度是否符合要求
func main() {
	// 收集所有使用的数据源标识符
	identifiers := []string{
		"websocket",       // 9 chars - WebSocket实时数据
		"http_api",        // 7 chars - HTTP API数据
		"init_populate",   // 12 chars - 初始化填充数据
		"realtime_ws",     // 11 chars - 实时同步器数据
		"manual_sync",     // 11 chars - 手动同步数据
		"syncer",          // 6 chars - 同步器数据
		"24h_stats",       // 9 chars - 24小时统计数据
		"kline_calc",      // 10 chars - K线计算数据
		"batch_import",    // 11 chars - 批量导入数据
		"api_fallback",    // 12 chars - API降级数据
		"cache_restore",   // 13 chars - 缓存恢复数据
	}

	fmt.Println("=== 数据源标识符长度验证 ===")
	fmt.Println("数据库字段限制: VARCHAR(32)")
	fmt.Println("当前所有标识符:")
	fmt.Println()

	maxLength := 0
	allValid := true

	for _, id := range identifiers {
		length := len(id)
		if length > maxLength {
			maxLength = length
		}

		status := "✅"
		if length > 32 {
			status = "❌"
			allValid = false
		}

		fmt.Printf("%-15s %2d chars %s\n", id, length, status)
	}

	fmt.Println()
	fmt.Printf("最长标识符: %d 字符\n", maxLength)
	fmt.Printf("数据库字段: VARCHAR(32) - 剩余空间: %d 字符\n", 32-maxLength)

	if allValid {
		fmt.Println("✅ 所有标识符长度都符合要求!")
	} else {
		fmt.Println("❌ 存在不符合要求的标识符!")
		log.Fatal("标识符长度验证失败")
	}

	fmt.Println()
	fmt.Println("=== 建议的标识符命名规范 ===")
	fmt.Println("1. 使用小写字母和下划线")
	fmt.Println("2. 语义清晰，易于理解")
	fmt.Println("3. 长度控制在32字符以内")
	fmt.Println("4. 避免使用空格和特殊字符")
	fmt.Println("5. 保持命名一致性")

	fmt.Println()
	fmt.Println("=== 使用示例 ===")
	fmt.Println(`// 在代码中使用`)
	fmt.Println(`DataSource: "websocket",    // WebSocket实时数据`)
	fmt.Println(`DataSource: "http_api",     // HTTP API获取的数据`)
	fmt.Println(`DataSource: "init_populate", // 初始化填充的数据`)
	fmt.Println(`DataSource: "realtime_ws",   // 实时同步器处理的数据`)
}