// internal/api/main.go
package main

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	pmServerToken = flag.String("pm-server-token", "", "Postmark Server Token")
	pmFrom        = flag.String("pm-from", "", "Postmark From email")
	pmTo          = flag.String("pm-to", "", "Comma-separated recipients")
	pmStream      = flag.String("pm-stream", "outbound", "Postmark message stream")

	xBearer = flag.String("x-bearer", "", "Twitter/X API Bearer token (can also be set via TWITTER_BEARER_TOKEN env var)")
)

// cleanupZombieStrategies 清理后端重启时状态为running或pending但实际已停止的僵尸策略
func cleanupZombieStrategies(gdb *gorm.DB) error {
	log.Printf("[Cleanup] Starting zombie strategies cleanup...")

	// 查找所有状态为running或pending的策略执行记录（pending的也可能是僵尸进程）
	var zombieExecutions []pdb.StrategyExecution
	if err := gdb.Where("status IN ?", []string{"running", "pending"}).Find(&zombieExecutions).Error; err != nil {
		return fmt.Errorf("failed to find zombie executions: %w", err)
	}

	if len(zombieExecutions) == 0 {
		log.Printf("[Cleanup] No zombie executions found")
		return nil
	}

	log.Printf("[Cleanup] Found %d potentially zombie executions", len(zombieExecutions))

	cleanedCount := 0
	for _, execution := range zombieExecutions {
		// 后端重启时，所有状态为running或pending的执行都是僵尸进程（因为goroutine不会持久化）
		// 标记为失败并记录日志
		if err := pdb.UpdateStrategyExecutionStatus(gdb, execution.ID, "failed", "后端重启时检测到僵尸进程", "", 100, 100, "后端服务重启时发现该策略执行未正常完成"); err != nil {
			log.Printf("[Cleanup] Failed to update zombie execution %d: %v", execution.ID, err)
			continue
		}

		// 计算运行时间用于日志
		elapsed := "未知"
		if !execution.StartTime.IsZero() {
			elapsed = fmt.Sprintf("%.1f分钟", time.Since(execution.StartTime).Minutes())
		}

		pdb.AppendStrategyExecutionLog(gdb, execution.ID, fmt.Sprintf("后端重启时检测到僵尸进程，已自动标记为失败（运行时间：%s）", elapsed))

		// 对于有僵尸进程的策略，停止策略运行状态，避免立即重新启动
		// 让用户手动重新启动策略，确保策略配置是最新的
		if err := pdb.UpdateStrategyRunningStatus(gdb, execution.StrategyID, false); err != nil {
			log.Printf("[Cleanup] Failed to stop strategy %d after cleaning zombie execution: %v", execution.StrategyID, err)
		} else {
			log.Printf("[Cleanup] Stopped strategy %d due to zombie execution cleanup", execution.StrategyID)
		}

		cleanedCount++
		log.Printf("[Cleanup] Cleaned zombie execution %d (strategy %d, status: %s, elapsed: %s)",
			execution.ID, execution.StrategyID, execution.Status, elapsed)
	}

	log.Printf("[Cleanup] Zombie strategies cleanup completed. Cleaned %d zombie executions", cleanedCount)
	return nil
}

func main() {
	addr := flag.String("addr", ":8010", "listen addr")
	cfgPath := flag.String("config", "./config.yaml", "config path")
	corsOrigins := flag.String("cors", "*", "cors origins, comma separated")
	testMode := flag.Bool("test", false, "run in test mode with minimal setup")
	flag.Parse()

	// 测试模式：只启动最基本的HTTP服务器
	if *testMode {
		fmt.Println("Running in test mode...")
		r := gin.New()
		r.GET("/healthz", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true, "time": time.Now().UTC()})
		})
		r.GET("/recommendations/coins", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"test":    true,
				"message": "Test mode: API server is working",
				"time":    time.Now().UTC(),
			})
		})
		fmt.Printf("Test server listening on %s\n", *addr)
		if err := r.Run(*addr); err != nil {
			fmt.Printf("Failed to start test server: %v\n", err)
		}
		return
	}

	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)
	config.ApplyProxy(&cfg)

	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  cfg.Database.Automigrate,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		fmt.Printf("Warning: Failed to connect to database, continuing without DB: %v\n", err)
		gdb = nil
	}

	// 自动迁移所有表（如果数据库可用）
	if gdb != nil {
		if err := gdb.GormDB().AutoMigrate(
			&pdb.User{},
			&pdb.CoinRecommendation{},
			&pdb.RecommendationPerformance{}, // 添加推荐表现追踪表
			&pdb.BacktestRecord{},
			&pdb.SimulatedTrade{},
			&pdb.AsyncBacktestRecord{}, // 异步回测记录
			&pdb.AsyncBacktestTrade{},  // 异步回测交易记录
			&pdb.ABTestConfig{},        // A/B测试配置
			&pdb.ABTestResult{},        // A/B测试结果
			&pdb.ScheduledOrder{},      // 定时合约单
			&pdb.TradingStrategy{},     // 交易策略
			// 用户行为追踪表
			&pdb.UserBehavior{},
			&pdb.UserPreference{},
			&pdb.UserRecommendationFeedback{},
			&pdb.UserBehaviorAnalysis{},
			&pdb.AlgorithmPerformance{},
			&pdb.NansenWhaleWatch{},        // Nansen 大户监控
			&pdb.RealtimeGainersSnapshot{}, // 实时涨幅榜快照
			&pdb.RealtimeGainersItem{},     // 涨幅榜数据项
			&pdb.BinanceFuturesContract{},  // 币安期货合约信息
		); err != nil {
			fmt.Printf("Warning: Failed to migrate database: %v\n", err)
		}
	}

	// 使用依赖注入：创建数据库接口实现
	var db server.Database
	if gdb != nil {
		db = server.NewGormDatabase(gdb.GormDB())
	}
	api := server.New(db, &cfg)

	// 优化：初始化缓存写入协程池（限制并发数为 50）
	server.InitCachePool(50)

	// 清理僵尸策略状态（后端重启时将状态为running但实际已停止的策略标记为failed）
	if gdb != nil {
		if err := cleanupZombieStrategies(gdb.GormDB()); err != nil {
			fmt.Printf("Warning: Failed to cleanup zombie strategies: %v\n", err)
			log.Printf("[Main] Warning: Failed to cleanup zombie strategies: %v", err)
		} else {
			fmt.Println("Zombie strategies cleanup completed")
			log.Printf("[Main] Zombie strategies cleanup completed")
		}
	}

	// 注意：OrderScheduler将在HTTP服务器启动后通过Server初始化时启动
	// 这里不再单独启动，以避免Server引用问题
	defer func() {
		if err := server.ShutdownCachePool(10 * time.Second); err != nil {
			fmt.Printf("Warning: Failed to shutdown cache pool: %v\n", err)
		}
		// 关闭服务器
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := api.Shutdown(ctx); err != nil {
			fmt.Printf("Warning: Failed to shutdown server: %v\n", err)
		}
	}()

	// 初始化缓存nu
	var cache pdb.CacheInterface
	if cfg.Redis.Enable && cfg.Redis.Addr != "" {
		// 使用 Redis 缓存
		redisCache, err := pdb.NewRedisCacheFromOptions(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			fmt.Printf("Warning: Failed to connect to Redis, using memory cache: %v\n", err)
			cache = pdb.NewMemoryCache()
		} else {
			cache = redisCache
			fmt.Println("Redis cache enabled")
		}
	} else {
		// 使用内存缓存（默认）
		cache = pdb.NewMemoryCache()
		fmt.Println("Memory cache enabled")
	}
	api.SetCache(cache)

	// Check for Arkham configuration - support both top-level and whale_monitoring.arkham
	arkhamBaseURL := cfg.Arkham.BaseURL
	arkhamAPIKey := cfg.Arkham.APIKey

	// If top-level arkham config is empty, try whale_monitoring.arkham
	if arkhamBaseURL == "" && cfg.WhaleMonitoring.Arkham.BaseURL != "" {
		arkhamBaseURL = cfg.WhaleMonitoring.Arkham.BaseURL
		arkhamAPIKey = cfg.WhaleMonitoring.Arkham.APIKey
	}

	if arkhamBaseURL != "" {
		arkhamClient := server.NewArkhamClient(arkhamBaseURL, arkhamAPIKey, nil)
		api.SetArkhamClient(arkhamClient)
		fmt.Println("Arkham client configured")
	}

	// Check for Nansen configuration - support both top-level and whale_monitoring.nansen
	nansenBaseURL := cfg.Nansen.BaseURL
	nansenAPIKey := cfg.Nansen.APIKey

	// If top-level nansen config is empty, try whale_monitoring.nansen
	if nansenBaseURL == "" && cfg.WhaleMonitoring.Nansen.BaseURL != "" {
		nansenBaseURL = cfg.WhaleMonitoring.Nansen.BaseURL
		nansenAPIKey = cfg.WhaleMonitoring.Nansen.APIKey
	}

	if nansenBaseURL != "" {
		nansenClient := server.NewNansenClient(nansenBaseURL, nansenAPIKey, nil)
		api.SetNansenClient(nansenClient)
		fmt.Println("Nansen client configured")
	}

	if *pmServerToken != "" && *pmFrom != "" && *pmTo != "" {
		recipients := strings.Split(*pmTo, ",")
		api.Mailer = server.NewPostmarkMailer(*pmServerToken, *pmFrom, recipients, *pmStream)
	}

	// 优化：安全地获取 Twitter Bearer Token（优先级：命令行参数 > 环境变量 > 配置文件）
	api.XBearer = strings.TrimSpace(*xBearer)
	if api.XBearer == "" {
		// 尝试从环境变量获取
		if envToken := os.Getenv("TWITTER_BEARER_TOKEN"); envToken != "" {
			api.XBearer = strings.TrimSpace(envToken)
		}
	}
	if api.XBearer == "" {
		// 最后尝试从配置文件获取
		if cfg.Twitter.Bearer != "" {
			api.XBearer = cfg.Twitter.Bearer
		}
	}
	// 如果仍然为空，记录警告但不强制退出（某些功能可能不需要 Twitter API）
	if api.XBearer == "" {
		log.Printf("[WARN] Twitter Bearer Token not configured. Twitter-related features may not work.")
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	// 优化：添加统一的错误处理中间件
	r.Use(server.ErrorHandlerMiddleware())

	fmt.Println("Setting up routes...")

	// CORS
	c := cors.DefaultConfig()
	if *corsOrigins == "*" {
		c.AllowAllOrigins = true
	} else {
		var origins []string
		for _, o := range strings.Split(*corsOrigins, ",") {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			origins = append(origins, o)
		}
		c.AllowOrigins = origins
	}
	c.AllowCredentials = true
	c.AllowHeaders = []string{"Authorization", "Content-Type", "Origin"}
	c.AllowMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions}
	r.Use(cors.New(c))

	// 健康检查
	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true, "time": time.Now().UTC()})
	})

	// 登录注册
	r.POST("/auth/register", api.Register)
	r.POST("/auth/login", api.Login)
	r.GET("/me", api.JWTAuth(), api.Me)

	// cursor & ingest events
	r.GET("/sync/cursor", server.GetCursor(gdb.GormDB()))
	r.POST("/sync/cursor", server.GetCursor(gdb.GormDB()))
	r.POST("/ingest/events", server.IngestEvents(gdb.GormDB()))

	r.POST("/ingest/binance/market", api.IngestBinanceMarket)

	pub := r.Group("/")
	{
		// Twitter 接口（带缓存，3分钟）
		pub.GET("/twitter/posts",
			server.CacheMiddleware(cache, pdb.CacheTypeAggregate, 3*time.Minute, server.TwitterPostsCacheKey),
			api.ListTwitterPosts)
		pub.GET("/twitter/fetch", api.FetchTwitterUserPosts)

		// 市场数据接口（带缓存，2分钟）- 公开访问，无需登录
		pub.GET("/market/binance/top",
			server.CacheMiddleware(cache, pdb.CacheTypeRealTime, 2*time.Minute, server.MarketCacheKey),
			api.GetBinanceMarket)
		pub.GET("/market/binance/realtime-gainers",
			server.CacheMiddleware(cache, pdb.CacheTypeRealTime, 30*time.Second, server.MarketCacheKey),
			api.GetRealTimeGainers)
		pub.GET("/market/binance/realtime-gainers/history",
			server.CacheMiddleware(cache, pdb.CacheTypeRealTime, 1*time.Minute, server.MarketCacheKey),
			api.GetRealtimeGainersHistoryAPI)
		pub.GET("/market/binance/realtime-gainers/stats", api.GetRealtimeGainersStatsAPI)
		pub.GET("/market/price-history", api.GetMarketPriceHistory)
		pub.GET("/api/v1/market/price/:symbol", api.GetCurrentPriceHTTP)
		pub.POST("/api/v1/market/batch-prices", api.GetBatchCurrentPrices)
		pub.GET("/api/v1/market/klines/:symbol", api.GetKlines)
		pub.GET("/api/v1/market/symbols", api.GetAvailableSymbols)
		pub.GET("/api/v1/market/symbols-with-marketcap", api.GetSymbolsWithMarketCap)
		pub.GET("/api/v1/market/symbol-analysis/:symbol", api.AnalyzeSymbolForGridTrading)
		pub.GET("/api/v1/market/grid-symbols", api.GetGridTradingSymbols)
		pub.GET("/api/v1/market/grid-analysis/:symbol", api.AnalyzeGridStrategy)
		pub.GET("/api/v1/recommend/performance/:symbol", api.GetRecommendationPerformance)
		pub.GET("/api/v1/sentiment/:symbol", api.GetSentimentAnalysis)

		// 投资服务批量操作接口（公开访问，无需登录）
		pub.POST("/recommendations/performance/batch-update", api.BatchUpdateRecommendationPerformance)
		pub.POST("/recommendations/performance/batch-strategy-test", api.BatchStrategyTest)
	}

	// 公告接口（带缓存，5分钟）
	r.GET("/announcements/recent",
		server.CacheMiddleware(cache, pdb.CacheTypeAggregate, 5*time.Minute, server.AnnouncementsCacheKey),
		api.ListAnnouncements)
	r.GET("/announcements/latest-time", api.GetLatestAnnouncementTime)

	// 推荐接口（临时公开用于测试）
	r.GET("/recommendations/coins", api.GetCoinRecommendations)
	r.GET("/recommendations/historical", api.GetHistoricalRecommendations)
	r.GET("/recommendations/times", api.GetRecommendationTimeList)
	r.POST("/recommendations/generate", api.GenerateRecommendationsForDate)

	// 新增：AI推荐API v1接口（兼容前端调用）
	fmt.Println("Setting up AI recommendation route: POST /api/v1/recommend")
	r.POST("/api/v1/recommend", api.GetAIRecommendations)
	r.GET("/api/v1/recommend/detail/:symbol", api.GetRecommendationDetail)
	r.POST("/api/v1/recommend/advanced", api.GetAdvancedRecommendations)

	r.GET("/api/v1/risk/report", api.GetRiskReport)
	r.POST("/api/v1/risk/assess", api.AssessRisk)
	r.GET("/api/v1/risk/alerts", api.GetRiskAlerts)
	r.POST("/api/v1/risk/alerts/:alertId/acknowledge", api.AcknowledgeAlert)
	r.POST("/api/v1/risk/portfolio/analyze", api.AnalyzePortfolio)
	r.POST("/api/v1/features/extract", api.ExtractFeatures)
	r.POST("/api/v1/features/batch-extract", api.BatchExtractFeatures)
	r.GET("/api/v1/features/importance", api.GetFeatureImportance)
	r.GET("/api/v1/features/quality", api.GetFeatureQuality)
	r.GET("/api/v1/ml/models/performance", api.GetModelPerformance)
	r.POST("/api/v1/ml/models/train", api.TrainModel)
	r.POST("/api/v1/ml/predict", api.PredictWithModel)
	r.GET("/api/v1/ml/models/health", api.GetModelHealth)
	r.GET("/api/v1/ml/models/validate", api.ValidateModels)
	r.POST("/api/v1/ml/features/validate", api.ValidateFeatures)

	// 超参数优化API
	r.POST("/api/v1/ml/hyperparameters/optimize", api.OptimizeHyperparametersAPI)
	r.GET("/api/v1/ml/hyperparameters/progress", api.GetHyperparameterOptimizationProgressAPI)
	r.GET("/api/v1/ml/hyperparameters/results", api.GetHyperparameterOptimizationResultsAPI)
	r.POST("/api/v1/ml/hyperparameters/apply", api.ApplyOptimizedParametersAPI)

	// Transformer模型API
	r.POST("/api/v1/ml/transformer/train", api.TrainTransformerModelAPI)
	r.POST("/api/v1/ml/transformer/predict", api.PredictWithTransformerAPI)
	r.POST("/api/v1/ml/transformer/features", api.ExtractTransformerFeaturesAPI)
	r.POST("/api/v1/ml/transformer/test-integration", api.TestTransformerIntegrationAPI)

	// 高级特征工程API
	r.POST("/api/v1/features/advanced-extract", api.ExtractAdvancedFeaturesAPI)
	r.GET("/api/v1/features/importance-analysis", api.GetFeatureImportanceAnalysisAPI)

	// 高级风险管理API
	r.GET("/api/v1/risk/advanced-metrics", api.GetAdvancedRiskMetricsAPI)
	r.POST("/api/v1/risk/stress-test", api.PerformStressTestAPI)
	r.POST("/api/v1/risk/portfolio/optimize", api.OptimizePortfolioAPI)
	r.POST("/api/v1/risk/budget", api.GetRiskBudgetAPI)

	// 高级回测分析API
	r.POST("/api/v1/backtest/walk-forward", api.RunWalkForwardAnalysisAPI)
	r.POST("/api/v1/backtest/monte-carlo", api.RunMonteCarloAnalysisAPI)
	r.POST("/api/v1/backtest/optimize", api.RunStrategyOptimizationAPI)
	r.POST("/api/v1/backtest/attribution", api.RunAttributionAnalysisAPI)

	// WebSocket 实时推荐
	r.GET("/ws/recommend", api.WSRecommendations)

	// WebSocket 实时价格
	r.GET("/ws/prices", api.WSPrices)

	// WebSocket 实时涨幅榜
	r.GET("/ws/realtime-gainers", api.WSRealTimeGainers)

	// 系统状态和监控
	r.GET("/api/v1/status", api.GetSystemStatus)
	r.GET("/api/v1/stats", api.GetSystemStats)

	// 数据同步监控接口
	r.GET("/api/data-sync/status", api.GetDataSyncStatus)
	r.POST("/api/data-sync/trigger", api.TriggerManualSync)
	r.GET("/api/data-sync/consistency", api.GetDataConsistencyStatus)
	r.GET("/api/data-sync/alerts", api.GetAlerts)
	r.POST("/api/data-sync/consistency/check", api.TriggerConsistencyCheck)
	r.POST("/api/data-sync/websocket/reconnect", api.ReconnectWebSocket)

	// 数据质量监控接口
	r.GET("/data-quality/report", api.GetDataQualityReport)
	r.GET("/fallback/status", api.GetFallbackStatus)

	// 公开的黑名单查询接口（供 collector 使用，已废弃，collector 不再使用黑名单）

	r.POST("/ingest/binance/announcements", api.IngestBinanceAnnouncements)
	r.POST("/ingest/upbit/announcements", api.IngestUpbitAnnouncements)
	r.POST("/ingest/:source/announcements", api.IngestGenericAnnouncements) // 通用接口：okx, bybit, coincarp, cryptopanic, coinmarketcal

	// 大户监控接口（公开访问，只读操作）
	r.GET("/whales/watchlist", server.ListWhaleWatches(api))
	r.POST("/whales/watchlist", server.CreateWhaleWatch(api))
	r.DELETE("/whales/watchlist/:address", server.DeleteWhaleWatch(api))

	// 需要鉴权的
	priv := r.Group("/")
	priv.Use(api.JWTAuth())
	{
		priv.GET("/entities", api.ListEntities)
		priv.GET("/runs", api.ListRuns)

		// 投资组合接口（带缓存，1分钟）
		priv.GET("/portfolio/latest",
			server.CacheMiddleware(cache, pdb.CacheTypeRealTime, 1*time.Minute, server.PortfolioCacheKey),
			api.GetLatestPortfolio)
		// 资金流接口（带缓存，5分钟）
		priv.GET("/flows/daily",
			server.CacheMiddleware(cache, pdb.CacheTypeAggregate, 5*time.Minute, server.FlowsCacheKey),
			api.GetDailyFlows)
		priv.GET("/flows/weekly", api.GetWeeklyFlows)
		priv.GET("/flows/daily_by_chain", api.GetDailyFlowsByChain)
		priv.GET("/transfers/recent", server.ListTransfers(api))
		priv.GET("/whales/arkham", server.ListArkhamWatches(api))
		priv.POST("/whales/arkham", server.CreateArkhamWatch(api))
		priv.POST("/whales/arkham/query", server.QueryArkhamAddress(api))
		priv.DELETE("/whales/arkham/:address", server.DeleteArkhamWatch(api))
		priv.POST("/whales/arkham/sync", server.TriggerArkhamSync(api))
		priv.GET("/whales/nansen", server.ListNansenWatches(api))
		priv.POST("/whales/nansen", server.CreateNansenWatch(api))
		priv.POST("/whales/nansen/query", server.QueryNansenAddress(api))
		priv.DELETE("/whales/nansen/:address", server.DeleteNansenWatch(api))
		priv.POST("/whales/nansen/sync", server.TriggerNansenSync(api))

		// 黑名单管理
		priv.GET("/market/binance/blacklist", api.ListBinanceBlacklist)
		priv.POST("/market/binance/blacklist", api.AddBinanceBlacklist)
		priv.DELETE("/market/binance/blacklist/:kind/:symbol", api.DeleteBinanceBlacklist)

		// 涨幅榜数据管理
		priv.POST("/market/binance/realtime-gainers/clean", api.CleanRealtimeGainersDataAPI)

		priv.POST("/orders/schedule", api.CreateScheduledOrder)
		priv.POST("/orders/schedule/batch", api.CreateBatchScheduledOrders)
		priv.GET("/orders/schedule", api.ListScheduledOrders)
		priv.GET("/orders/schedule/:id", api.GetScheduledOrderDetail)
		priv.POST("/orders/schedule/:id/cancel", api.CancelScheduledOrder)
		priv.POST("/orders/schedule/:id/close-position", api.ClosePosition)
		priv.DELETE("/orders/schedule/:id", api.DeleteScheduledOrder)

		// 交易策略管理
		priv.POST("/strategies", api.CreateTradingStrategy)
		priv.GET("/strategies", api.ListTradingStrategies)
		priv.GET("/strategies/:id", api.GetTradingStrategy)
		priv.PUT("/strategies/:id", api.UpdateTradingStrategy)
		priv.DELETE("/strategies/:id", api.DeleteTradingStrategy)

		// 策略执行
		priv.POST("/strategies/execute", api.ExecuteStrategy)
		priv.POST("/strategies/batch-execute", api.BatchExecuteStrategies)
		priv.POST("/strategies/scan-eligible", api.ScanEligibleSymbols)
		priv.POST("/strategies/discover-arbitrage", api.DiscoverArbitrageOpportunities)

		// 策略运行管理
		priv.POST("/strategies/:id/start", api.StartStrategyExecution)
		priv.POST("/strategies/:id/stop", api.StopStrategyExecution)
		priv.GET("/strategies/:id/health", api.GetStrategyHealth)
		priv.GET("/strategies/executions", api.ListStrategyExecutions)
		priv.GET("/strategies/executions/:execution_id", api.GetStrategyExecution)
		priv.GET("/strategies/executions/:execution_id/steps", api.GetStrategyExecutionSteps)
		priv.DELETE("/strategies/executions/:execution_id", api.DeleteStrategyExecution)
		priv.GET("/strategies/:id/stats", api.GetStrategyExecutionStats)
		priv.GET("/strategies/:id/orders", api.GetStrategyOrders)

		// 币种推荐 - 暂时跳过认证用于测试
		// priv.GET("/recommendations/coins", api.GetCoinRecommendations)
		// priv.GET("/recommendations/historical", api.GetHistoricalRecommendations)
		// priv.GET("/recommendations/times", api.GetRecommendationTimeList)
		// priv.POST("/recommendations/generate", api.GenerateRecommendationsForDate)

		// 推荐接口已移至根路由公开注册（临时用于测试）
		// pub.GET("/recommendations/coins", api.GetCoinRecommendations)
		// pub.GET("/recommendations/historical", api.GetHistoricalRecommendations)
		// pub.GET("/recommendations/times", api.GetRecommendationTimeList)
		// pub.POST("/recommendations/generate", api.GenerateRecommendationsForDate)

		// 回测功能
		priv.GET("/recommendations/backtest", api.GetBacktestRecords)
		priv.GET("/recommendations/backtest/stats", api.GetBacktestStats)
		priv.POST("/recommendations/backtest", api.CreateBacktestFromRecommendation)
		priv.POST("/recommendations/backtest/:id/update", api.UpdateBacktestRecord)
		priv.POST("/recommendations/backtest/batch-update", api.BatchUpdateBacktestRecords)

		// 策略回测功能
		priv.POST("/recommendations/backtest/strategy", api.ExecuteStrategyBacktest)
		priv.POST("/recommendations/backtest/strategy/test", api.TestStrategyBacktest)
		priv.POST("/recommendations/backtest/strategy/batch", api.BatchExecuteStrategyBacktest)

		// 模拟交易
		priv.POST("/recommendations/simulation/trade", api.CreateSimulatedTrade)
		priv.GET("/recommendations/simulation/trades", api.GetSimulatedTrades)
		priv.POST("/recommendations/simulation/trades/:id/close", api.CloseSimulatedTrade)
		priv.POST("/recommendations/simulation/trades/:id/update-price", api.UpdateSimulatedTradePrice)

		// 自动执行设置
		priv.GET("/user/auto-execute/settings", api.GetAutoExecuteSettings)
		priv.PUT("/user/auto-execute/settings", api.UpdateAutoExecuteSettings)
		priv.POST("/recommendations/auto-execute", api.ExecuteRecommendations)
		priv.DELETE("/recommendations/simulations/trades", api.ClearUserTrades)

		// 推荐表现追踪
		priv.GET("/recommendations/performance", api.GetRecommendationPerformanceAPI)
		priv.GET("/recommendations/performance/batch", api.GetBatchRecommendationPerformanceAPI)
		priv.GET("/recommendations/performance/stats", api.GetPerformanceStatsAPI)
		priv.GET("/recommendations/performance/factor-stats", api.GetFactorPerformanceStatsAPI)
		priv.GET("/recommendations/performance/trend", api.GetPerformanceTrendAPI)

		// 市场分析
		// 综合市场分析接口（推荐使用）
		priv.GET("/market-analysis/comprehensive", api.GetComprehensiveMarketAnalysis)

		// 保留原有独立接口（向后兼容）
		priv.GET("/market-analysis/environment", api.GetMarketAnalysis)
		priv.GET("/market-analysis/technical-indicators", api.GetMarketTechnicalIndicators)
		priv.GET("/market-analysis/strategy-recommendations", api.GetStrategyRecommendations)

		// AI推荐策略回测API (需要认证)
		priv.POST("/api/ai-recommendation/backtest", api.AIBacktestAPI)

		// 异步回测API
		priv.POST("/api/backtest/async/start", api.StartAsyncBacktestAPI)
		priv.GET("/api/backtest/async/records", api.GetBacktestRecordsAPI)
		priv.GET("/api/backtest/async/records/:id", api.GetBacktestRecordAPI)
		priv.GET("/api/backtest/async/trades/:recordId", api.GetBacktestTradesAPI)
		priv.DELETE("/api/backtest/async/records/:id", api.DeleteBacktestRecordAPI)

		// 多源数据服务 - 暂时注释，待实现
		// priv.GET("/data/multi-source", api.GetMultiSourceData)
		// priv.GET("/data/symbol/:symbol", api.GetSymbolData)
		// priv.GET("/data/sources", api.GetDataSources)
		// priv.POST("/data/refresh", api.RefreshData)
		// priv.GET("/data/quality-report", api.GetDataQualityReport)

		// 回测服务 - 暂时注释，待实现
		priv.POST("/backtest/run", api.RunBacktestAPI)
		priv.POST("/backtest/strategy", api.RunStrategyBacktestAPI)
		// priv.POST("/backtest/compare", api.CompareStrategies)
		// priv.POST("/backtest/batch", api.BatchBacktest)
		// priv.POST("/backtest/optimize", api.OptimizeStrategy)
		priv.GET("/backtest/templates", api.GetBacktestTemplatesAPI)
		priv.GET("/backtest/strategies", api.GetAvailableStrategiesAPI)
		priv.POST("/backtest/save", api.SaveBacktestResultAPI)
		priv.GET("/backtest/saved", api.GetSavedBacktestsAPI)

		// 过滤器修正统计
		priv.GET("/backtest/filter-corrections/stats", api.GetFilterCorrectionStats)
		priv.GET("/backtest/filter-corrections/:symbol", api.GetFilterCorrectionsBySymbol)
		priv.POST("/backtest/filter-corrections/cleanup", api.CleanupOldFilterCorrections)

		// 用户行为追踪
		priv.POST("/user/behavior/track", api.TrackUserBehavior)
		priv.POST("/user/feedback", api.SubmitRecommendationFeedback)
		priv.GET("/user/feedback/history", api.GetUserFeedbackHistory)
		priv.GET("/recommendations/stats", api.GetRecommendationStats)

		// Price alert routes
		priv.POST("/price-alerts", api.CreatePriceAlert)
		priv.GET("/price-alerts", api.GetPriceAlerts)
		priv.DELETE("/price-alerts/:id", api.DeletePriceAlert)
		priv.GET("/price-monitor/stats", api.GetPriceMonitorStats)
		priv.POST("/price-monitor/start", api.StartPriceMonitor)
		priv.POST("/price-monitor/stop", api.StopPriceMonitor)

		// Cache management routes
		priv.GET("/cache/stats", api.GetCacheStats)
		priv.POST("/cache/warmup", api.WarmupCache)
		priv.POST("/cache/clear", api.ClearCache)
		priv.POST("/cache/invalidate/user/:userId", api.InvalidateUserCache)

		// Data preprocessing and caching routes
		priv.GET("/data/cache/stats", api.GetDataCacheStats)
		priv.GET("/data/update/status", api.GetDataUpdateServiceStatus)
		priv.POST("/data/update/trigger", api.TriggerDataUpdate)
		priv.POST("/data/cache/clear", api.ClearDataCache)

		// Feature precomputation routes
		priv.GET("/feature/cache/stats", api.GetFeatureCacheStats)
		priv.GET("/feature/precompute/status", api.GetFeaturePrecomputeServiceStatus)
		priv.POST("/feature/precompute/trigger", api.TriggerFeaturePrecomputation)
		priv.POST("/feature/cache/clear", api.ClearFeatureCache)
		priv.GET("/feature/popular-symbols", api.GetPopularFeatureSymbols)

		// Technical indicators precomputation routes
		priv.GET("/technical-indicators/cache/stats", api.GetTechnicalIndicatorsCacheStats)
		priv.GET("/technical-indicators/precompute/status", api.GetTechnicalIndicatorsPrecomputeServiceStatus)
		priv.POST("/technical-indicators/precompute/trigger", api.TriggerTechnicalIndicatorsPrecomputation)
		priv.POST("/technical-indicators/cache/clear", api.ClearTechnicalIndicatorsCache)
		priv.GET("/technical-indicators", api.GetTechnicalIndicators)

		// ML model pretraining routes
		priv.GET("/ml-models/cache/stats", api.GetMLModelCacheStats)
		priv.GET("/ml-models/pretraining/status", api.GetMLPretrainingServiceStatus)
		priv.POST("/ml-models/pretraining/trigger", api.TriggerMLModelPretraining)
		priv.POST("/ml-models/cache/clear", api.ClearMLModelCache)
		priv.GET("/ml-models", api.GetMLModel)
		priv.GET("/ml-models/best", api.GetBestMLModels)
		priv.GET("/ml-models/stats", api.GetMLModelStats)
		priv.POST("/ml-models/cleanup", api.CleanupExpiredMLModels)

		// Concurrency and resource management routes
		priv.GET("/concurrency/stats", api.GetConcurrencyStats)
		priv.GET("/resources/health", api.GetResourceHealth)
		priv.POST("/circuit-breakers/reset", api.ResetCircuitBreakers)
		priv.POST("/worker-pool/scale", api.ScaleWorkerPool)

		// Data preloader management routes
		priv.GET("/data-preloader/stats", api.GetDataPreloaderStats)
		priv.POST("/data-preloader/symbols", api.AddPreloaderSymbol)
		priv.DELETE("/data-preloader/symbols/:symbol", api.RemovePreloaderSymbol)
		priv.POST("/data-preloader/trigger-update", api.TriggerDataPreloaderUpdate)

		// Recommendation scheduler routes (调用独立的进程服务)
		priv.GET("/scheduler/status", api.GetRecommendationSchedulerStatus)
		priv.POST("/scheduler/start", api.StartRecommendationScheduler)
		priv.POST("/scheduler/stop", api.StopRecommendationScheduler)
		priv.POST("/scheduler/generate", api.ForceGenerateRecommendations)
		priv.POST("/scheduler/cleanup", api.CleanupOldRecommendations)
		priv.GET("/scheduler/stats", api.GetRecommendationDataStats)
		priv.GET("/analytics/feedback", api.GetFeedbackAnalytics)

		// A/B测试
		priv.POST("/ab-test", api.CreateABTest)
		priv.GET("/ab-test/:test_name/results", api.GetABTestResults)
		priv.GET("/ab-test/active", api.ListActiveABTests)
		priv.GET("/ab-test/group", api.GetUserTestGroup)

		// 算法优化
		priv.POST("/optimization/trigger", api.TriggerAlgorithmOptimization)
		priv.GET("/optimization/status", api.GetOptimizationStatus)
		priv.GET("/optimization/latest-result", api.GetLatestOptimizationResult)
	}

	// 注意：PerformanceTracker和SmartScheduler已移至独立的investment服务
	// 如需启动调度器，请运行: ./investment -service=investment -mode=scheduler

	// ws
	r.GET("/ws/transfers", server.WSTransfers)

	fmt.Println("API listening at", *addr)
	if err := r.Run(*addr); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		panic(err)
	}
}
