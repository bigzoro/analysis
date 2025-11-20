package main

import (
	"analysis/internal/db"
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// 这个文件展示了如何使用优化后的数据库查询功能

func main() {
	// ==================== 1. 初始化优化后的数据库连接 ====================
	opt := db.DefaultOptimizedOptions()
	opt.DSN = "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	opt.Automigrate = true
	opt.MaxOpenConns = 25
	opt.MaxIdleConns = 10

	gdb, err := db.OpenMySQLOptimized(opt)
	if err != nil {
		log.Fatal(err)
	}

	// ==================== 2. 创建索引（如果还没有）====================
	if err := db.CreateOptimizedIndexes(gdb); err != nil {
		log.Printf("Warning: failed to create indexes: %v", err)
	}

	// ==================== 3. 使用查询优化器 ====================
	optimizer := db.NewQueryOptimizer(gdb)

	// 优化转账查询
	entity := "binance"
	chain := "ethereum"
	coin := "USDT"
	limit := 50

	q := optimizer.OptimizeTransferQuery(entity, chain, coin, limit)

	var transfers []db.TransferEvent
	startTime := time.Now()
	if err := q.Find(&transfers).Error; err != nil {
		log.Fatal(err)
	}
	duration := time.Since(startTime)

	fmt.Printf("查询完成，耗时: %v, 结果数: %d\n", duration, len(transfers))

	// ==================== 4. 使用缓存查询 ====================
	// 初始化缓存（使用内存缓存作为示例）
	cache := db.NewMemoryCache()
	cacheWrapper := db.NewCacheWrapper(cache, gdb)

	ctx := context.Background()

	// 使用缓存查询日度资金流
	flows, err := cacheWrapper.GetCachedDailyFlows(
		ctx,
		"binance",
		[]string{"BTC", "ETH", "USDT"},
		"2025-01-01",
		"2025-01-31",
		"",
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("获取到 %d 条资金流记录\n", len(flows))

	// ==================== 5. 使用分页查询 ====================
	params := db.PaginationParams{
		Page:     1,
		PageSize: 50,
		MaxSize:  500,
	}

	result, err := db.Paginate[db.TransferEvent](gdb, params, func(q *gorm.DB) *gorm.DB {
		return q.Where("entity = ?", entity)
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("分页结果: 第 %d 页，共 %d 页，总计 %d 条\n",
		result.Page, result.TotalPages, result.Total)

	// ==================== 6. 使用聚合查询 ====================
	start := time.Now().AddDate(0, 0, -7)
	end := time.Now()

	stats, err := db.GetTransferStats(gdb, entity, chain, coin, start, end)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("统计结果: %+v\n", stats)

	// ==================== 7. 批量操作 ====================
	// 批量插入示例
	events := []db.TransferEvent{
		// ... 准备数据
	}

	if err := db.BatchInsert(gdb, events, 1000); err != nil {
		log.Fatal(err)
	}

	// ==================== 8. 监控连接池 ====================
	connStats, err := db.GetConnectionStats(gdb)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("连接池状态: %+v\n", connStats)

	// ==================== 9. 分析查询计划 ====================
	query := optimizer.OptimizeTransferQuery(entity, chain, coin, limit)
	plan, err := optimizer.ExplainQuery(query)
	if err != nil {
		log.Printf("无法分析查询计划: %v", err)
	} else {
		fmt.Printf("查询计划:\n%s\n", plan)
	}
}
