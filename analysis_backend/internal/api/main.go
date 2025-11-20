// internal/api/main.go
package main

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	pmServerToken = flag.String("pm-server-token", "", "Postmark Server Token")
	pmFrom        = flag.String("pm-from", "", "Postmark From email")
	pmTo          = flag.String("pm-to", "", "Comma-separated recipients")
	pmStream      = flag.String("pm-stream", "outbound", "Postmark message stream")

	xBearer = flag.String("x-bearer", "", "Twitter/X API Bearer token (can also be set via TWITTER_BEARER_TOKEN env var)")
)

func main() {
	addr := flag.String("addr", ":8010", "listen addr")
	cfgPath := flag.String("config", "./config.yaml", "config path")
	corsOrigins := flag.String("cors", "*", "cors origins, comma separated")
	flag.Parse()

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
		panic(err)
	}

	// 自动迁移所有表
	if err := gdb.AutoMigrate(
		&pdb.User{},
		&pdb.CoinRecommendation{},
		&pdb.BacktestRecord{},
		&pdb.SimulatedTrade{},
	); err != nil {
		panic(err)
	}

	// 使用依赖注入：创建数据库接口实现
	db := server.NewGormDatabase(gdb)
	api := server.New(db)
	
	// 优化：初始化缓存写入协程池（限制并发数为 50）
	server.InitCachePool(50)
	defer func() {
		if err := server.ShutdownCachePool(10 * time.Second); err != nil {
			fmt.Printf("Warning: Failed to shutdown cache pool: %v\n", err)
		}
	}()

	// 初始化缓存
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
	c.AllowMethods = []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions}
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
	r.GET("/sync/cursor", server.GetCursor(gdb))
	r.POST("/sync/cursor", server.GetCursor(gdb))
	r.POST("/ingest/events", server.IngestEvents(gdb))

	r.POST("/ingest/binance/market", api.IngestBinanceMarket)

	pub := r.Group("/")
	{
		// Twitter 接口（带缓存，3分钟）
		pub.GET("/twitter/posts", 
			server.CacheMiddleware(cache, pdb.CacheTypeAggregate, 3*time.Minute, server.TwitterPostsCacheKey),
			api.ListTwitterPosts)
		pub.GET("/twitter/fetch", api.FetchTwitterUserPosts)
	}

	// 公告接口（带缓存，5分钟）
	r.GET("/announcements/recent", 
		server.CacheMiddleware(cache, pdb.CacheTypeAggregate, 5*time.Minute, server.AnnouncementsCacheKey),
		api.ListAnnouncements)
	r.GET("/announcements/latest-time", api.GetLatestAnnouncementTime)

	// 公开的黑名单查询接口（供 collector 使用，已废弃，collector 不再使用黑名单）

	r.POST("/ingest/binance/announcements", api.IngestBinanceAnnouncements)
	r.POST("/ingest/upbit/announcements", api.IngestUpbitAnnouncements)
	r.POST("/ingest/:source/announcements", api.IngestGenericAnnouncements) // 通用接口：okx, bybit, coincarp, cryptopanic, coinmarketcal

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

		// 市场数据接口（带缓存，2分钟）
		priv.GET("/market/binance/top", 
			server.CacheMiddleware(cache, pdb.CacheTypeRealTime, 2*time.Minute, server.MarketCacheKey),
			api.GetBinanceMarket)

		// 黑名单管理
		priv.GET("/market/binance/blacklist", api.ListBinanceBlacklist)
		priv.POST("/market/binance/blacklist", api.AddBinanceBlacklist)
		priv.DELETE("/market/binance/blacklist/:kind/:symbol", api.DeleteBinanceBlacklist)

		priv.POST("/orders/schedule", api.CreateScheduledOrder)
		priv.GET("/orders/schedule", api.ListScheduledOrders)
		priv.POST("/orders/schedule/:id/cancel", api.CancelScheduledOrder)

		// 币种推荐
		priv.GET("/recommendations/coins", api.GetCoinRecommendations)
		
		// 回测功能
		priv.GET("/recommendations/backtest", api.GetBacktestRecords)
		priv.GET("/recommendations/backtest/stats", api.GetBacktestStats)
		priv.POST("/recommendations/backtest", api.CreateBacktestFromRecommendation)
		priv.POST("/recommendations/backtest/:id/update", api.UpdateBacktestRecord)
		
		// 模拟交易
		priv.POST("/recommendations/simulation/trade", api.CreateSimulatedTrade)
		priv.GET("/recommendations/simulation/trades", api.GetSimulatedTrades)
		priv.POST("/recommendations/simulation/trades/:id/close", api.CloseSimulatedTrade)
		priv.POST("/recommendations/simulation/trades/:id/update-price", api.UpdateSimulatedTradePrice)
	}

	// ws
	r.GET("/ws/transfers", server.WSTransfers)

	fmt.Println("API listening at", *addr)
	if err := r.Run(*addr); err != nil {
		panic(err)
	}
}
