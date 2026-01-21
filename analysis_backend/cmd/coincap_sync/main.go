package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"analysis/internal/config"
	"analysis/internal/db"
	"analysis/internal/server"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func main() {
	// 命令行参数
	action := flag.String("action", "auto-sync", "操作类型: sync(同步资产映射), market-data(同步市值数据), auto-sync(自动同步市值数据), validate(验证映射), search(搜索资产), stats(统计信息)")
	query := flag.String("query", "", "搜索关键词（用于search操作）")
	limit := flag.Int("limit", 100, "搜索结果限制数量（用于search操作）")
	interval := flag.Int("interval", 10080, "自动同步间隔（分钟，默认10080分钟，即7天）")
	cfgPath := flag.String("config", "./config.yaml", "配置文件路径")
	apiKey := flag.String("api-key", "292ca5251c7eab03e55f5f01f960dc635f00e2294e3963d0293764e36ff69080", "CoinCap API密钥（可选）")

	flag.Parse()

	fmt.Printf("[coincap_sync] starting action=%s, config=%s\n", *action, *cfgPath)

	// 1. 加载配置
	var cfg config.Config
	if b, err := os.ReadFile(*cfgPath); err != nil {
		fmt.Printf("[coincap_sync] failed to read config file %s: %v\n", *cfgPath, err)
		return
	} else {
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			fmt.Printf("[coincap_sync] failed to parse config file: %v\n", err)
			return
		}
	}
	config.ApplyProxy(&cfg)
	fmt.Printf("[coincap_sync] config loaded successfully\n")

	// 2. 初始化数据库连接
	database, err := db.OpenMySQL(db.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     true, // 确保表已创建
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	})
	if err != nil {
		fmt.Printf("[coincap_sync] failed to connect to database: %v\n", err)
		return
	}
	defer database.Close()

	// 3. 创建映射服务
	gormDB, err := database.DB()
	if err != nil {
		fmt.Printf("[coincap_sync] failed to get gorm DB: %v\n", err)
		return
	}
	mappingService := db.NewCoinCapMappingService(gormDB)

	// 4. 根据操作类型执行
	ctx := context.Background()

	switch *action {
	case "sync":
		runSyncAction(ctx, mappingService, *apiKey)
	case "market-data":
		runMarketDataSyncAction(ctx, gormDB, *apiKey)
	case "auto-sync":
		runAutoSyncAction(ctx, gormDB, *apiKey, *interval)
	case "validate":
		runValidateAction(ctx, mappingService)
	case "search":
		runSearchAction(ctx, mappingService, *query, *limit)
	case "stats":
		runStatsAction(ctx, mappingService)
	default:
		fmt.Printf("[coincap_sync] unknown action: %s\n", *action)
	}
}

// runSyncAction 执行同步操作
func runSyncAction(ctx context.Context, mappingService *db.CoinCapMappingService, apiKey string) {
	log.Printf("[coincap_sync] 正在同步CoinCap资产映射...")

	// 如果没有提供API密钥，尝试从环境变量获取
	if apiKey == "" {
		// 这里可以添加从环境变量获取API密钥的逻辑
		log.Printf("[coincap_sync] 未提供API密钥，使用免费额度")
	}

	// 创建同步服务
	syncService := server.NewCoinCapAssetSyncService(mappingService, apiKey)

	// 执行同步
	startTime := time.Now()
	err := syncService.SyncAllAssets(ctx)
	duration := time.Since(startTime)

	if err != nil {
		log.Fatalf("[coincap_sync] 同步失败: %v", err)
	}

	log.Printf("[coincap_sync] 同步完成，耗时: %v", duration)
}

// runMarketDataSyncAction 执行市值数据同步操作
func runMarketDataSyncAction(ctx context.Context, gormDB *gorm.DB, apiKey string) {
	log.Printf("[coincap_sync] 正在同步CoinCap市值数据...")

	// 如果没有提供API密钥，使用默认值
	if apiKey == "" {
		apiKey = "7edc027b9c9d0e605eef5fac52df638d00373f1e573cf457b333f0afa170a202"
		log.Printf("[coincap_sync] 使用默认API密钥")
	}

	// 创建市值数据同步服务
	marketDataService := db.NewCoinCapMarketDataService(gormDB)
	syncService := server.NewCoinCapMarketDataSyncService(marketDataService, apiKey)

	// 执行市值数据同步
	startTime := time.Now()
	err := syncService.SyncAllMarketData(ctx)
	duration := time.Since(startTime)

	if err != nil {
		log.Fatalf("[coincap_sync] 市值数据同步失败: %v", err)
	}

	log.Printf("[coincap_sync] 市值数据同步完成，耗时: %v", duration)

	// 显示同步统计信息
	stats, err := marketDataService.GetMarketDataStats(ctx)
	if err != nil {
		log.Printf("[coincap_sync] 获取统计信息失败: %v", err)
	} else {
		log.Printf("[coincap_sync] 市值数据统计:")
		log.Printf("  总记录数量: %v", stats["total_records"])
		if latestUpdate, ok := stats["latest_update"].(time.Time); ok {
			log.Printf("  最后更新时间: %v", latestUpdate.Format("2006-01-02 15:04:05"))
		}
	}
}

// runValidateAction 执行验证操作
func runValidateAction(ctx context.Context, mappingService *db.CoinCapMappingService) {
	log.Printf("[coincap_sync] 正在验证资产映射数据完整性...")

	syncService := server.NewCoinCapAssetSyncService(mappingService, "")

	err := syncService.ValidateMappings(ctx)
	if err != nil {
		log.Fatalf("[coincap_sync] 验证失败: %v", err)
	}

	log.Printf("[coincap_sync] 验证完成，数据完整性正常")
}

// runSearchAction 执行搜索操作
func runSearchAction(ctx context.Context, mappingService *db.CoinCapMappingService, query string, limit int) {
	if query == "" {
		log.Printf("[coincap_sync] 搜索关键词不能为空")
		return
	}

	log.Printf("[coincap_sync] 正在搜索资产，关键词: %s，限制: %d", query, limit)

	syncService := server.NewCoinCapAssetSyncService(mappingService, "")

	results, err := syncService.SearchAssets(ctx, query)
	if err != nil {
		log.Fatalf("[coincap_sync] 搜索失败: %v", err)
	}

	log.Printf("[coincap_sync] 找到 %d 个匹配结果:", len(results))
	for i, result := range results {
		if i >= limit {
			break
		}
		log.Printf("  %d. %s (%s) - %s [排名: %s]",
			i+1, result.Symbol, result.AssetID, result.Name, result.Rank)
	}
}

// runStatsAction 执行统计操作
func runStatsAction(ctx context.Context, mappingService *db.CoinCapMappingService) {
	log.Printf("[coincap_sync] 正在获取映射统计信息...")

	stats, err := mappingService.GetMappingStats(ctx)
	if err != nil {
		log.Fatalf("[coincap_sync] 获取统计信息失败: %v", err)
	}

	log.Printf("[coincap_sync] 映射统计信息:")
	log.Printf("  总映射数量: %v", stats["total_mappings"])
	if latestUpdate, ok := stats["latest_update"].(time.Time); ok {
		log.Printf("  最后更新时间: %v", latestUpdate.Format("2006-01-02 15:04:05"))
	}

	// 显示一些热门资产示例
	syncService := server.NewCoinCapAssetSyncService(mappingService, "")
	popular, err := syncService.GetPopularSymbols(ctx, 10)
	if err != nil {
		log.Printf("[coincap_sync] 获取热门资产失败: %v", err)
	} else {
		log.Printf("  热门资产示例 (前10名): %v", popular)
	}
}

// runAutoSyncAction 执行自动同步市值数据操作
func runAutoSyncAction(ctx context.Context, gormDB *gorm.DB, apiKey string, intervalMinutes int) {
	log.Printf("[coincap_sync] 开始自动同步市值数据，间隔: %d 分钟", intervalMinutes)

	// 创建市值数据同步服务
	marketDataService := db.NewCoinCapMarketDataService(gormDB)
	syncService := server.NewCoinCapMarketDataSyncService(marketDataService, apiKey)

	// 创建信号通道用于优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建定时器
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	// 计数器
	syncCount := 0
	lastSyncTime := time.Time{}

	// 执行首次同步
	log.Printf("[coincap_sync] 执行首次同步...")
	startTime := time.Now()
	err := syncService.SyncAllMarketData(ctx)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[coincap_sync] 首次同步失败: %v", err)
	} else {
		syncCount++
		lastSyncTime = time.Now()
		log.Printf("[coincap_sync] 首次同步完成，耗时: %v", duration)
	}

	// 显示初始统计信息
	showSyncStats(gormDB, syncCount, lastSyncTime)

	log.Printf("[coincap_sync] 自动同步已启动，按 Ctrl+C 退出...")

	// 主循环
	for {
		select {
		case <-sigChan:
			log.Printf("[coincap_sync] 收到退出信号，正在停止自动同步...")
			log.Printf("[coincap_sync] 总共执行了 %d 次同步", syncCount)
			return

		case <-ticker.C:
			log.Printf("[coincap_sync] 开始定时同步 (第%d次)...", syncCount+1)

			startTime := time.Now()
			err := syncService.SyncAllMarketData(ctx)
			duration := time.Since(startTime)

			if err != nil {
				log.Printf("[coincap_sync] 定时同步失败: %v", err)
			} else {
				syncCount++
				lastSyncTime = time.Now()
				log.Printf("[coincap_sync] 定时同步完成，耗时: %v", duration)

				// 每10次同步显示一次统计信息
				if syncCount%10 == 0 {
					showSyncStats(gormDB, syncCount, lastSyncTime)
				}
			}

			// 检查是否达到24小时，如果是则显示详细统计
			if syncCount > 0 && syncCount%(24*60/intervalMinutes) == 0 {
				log.Printf("[coincap_sync] 已运行24小时，显示详细统计...")
				showDetailedStats(gormDB, syncCount, lastSyncTime)
			}
		}
	}
}

// showSyncStats 显示同步统计信息
func showSyncStats(gormDB *gorm.DB, syncCount int, lastSyncTime time.Time) {
	marketDataService := db.NewCoinCapMarketDataService(gormDB)

	stats, err := marketDataService.GetMarketDataStats(context.Background())
	if err != nil {
		log.Printf("[coincap_sync] 获取统计信息失败: %v", err)
		return
	}

	log.Printf("[coincap_sync] 同步统计 (执行%d次):", syncCount)
	log.Printf("  数据记录总数: %v", stats["total_records"])
	if latestUpdate, ok := stats["latest_update"].(time.Time); ok {
		log.Printf("  数据最后更新: %v", latestUpdate.Format("2006-01-02 15:04:05"))
	}
	if !lastSyncTime.IsZero() {
		log.Printf("  最后同步时间: %v", lastSyncTime.Format("2006-01-02 15:04:05"))
	}
}

// showDetailedStats 显示详细统计信息
func showDetailedStats(gormDB *gorm.DB, syncCount int, lastSyncTime time.Time) {
	marketDataService := db.NewCoinCapMarketDataService(gormDB)

	// 获取市值分布统计
	ctx := context.Background()

	// 小市值币种 (<5000万)
	smallCap, err := marketDataService.GetSymbolsByMarketCapRange(ctx, 0, 50000000)
	if err == nil {
		log.Printf("  小市值币种 (<5000万): %d 个", len(smallCap))
	}

	// 中市值币种 (5000万-5亿)
	midCap, err := marketDataService.GetSymbolsByMarketCapRange(ctx, 50000000, 500000000)
	if err == nil {
		log.Printf("  中市值币种 (5000万-5亿): %d 个", len(midCap))
	}

	// 大市值币种 (>5亿)
	largeCap, err := marketDataService.GetSymbolsByMarketCapRange(ctx, 500000000, 1000000000000)
	if err == nil {
		log.Printf("  大市值币种 (>5亿): %d 个", len(largeCap))
	}

	log.Printf("  总同步次数: %d", syncCount)
	log.Printf("  服务运行时间: %v", time.Since(time.Now().Add(-time.Duration(syncCount)*time.Hour)))
}
