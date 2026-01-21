package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 模拟Server结构
type Server struct {
	db *gorm.DB
}

// 模拟数据库连接
func setupTestDB() *gorm.DB {
	dsn := "user:password@tcp(localhost:3306)/analysis?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("无法连接到测试数据库，将跳过测试: %v", err)
		return nil
	}
	return db
}

// 测试波动率指标计算的性能
func testVolatilityIndicators() {
	server := &Server{db: setupTestDB()}
	if server.db == nil {
		fmt.Println("跳过测试：无法连接数据库")
		return
	}

	fmt.Println("开始测试波动率指标计算...")
	startTime := time.Now()

	marketVolatility, highVolSymbols, lowVolSymbols := server.countVolatilityIndicators()

	duration := time.Since(startTime)

	fmt.Printf("测试结果:\n")
	fmt.Printf("市场平均波动率: %.2f%%\n", marketVolatility)
	fmt.Printf("高波动率币种: %d\n", highVolSymbols)
	fmt.Printf("低波动率币种: %d\n", lowVolSymbols)
	fmt.Printf("执行时间: %v\n", duration)

	if duration < 500*time.Millisecond {
		fmt.Println("✅ 性能优化成功：查询时间小于500ms")
	} else {
		fmt.Printf("⚠️  查询时间较慢: %v\n", duration)
	}
}

func main() {
	testVolatilityIndicators()
}
