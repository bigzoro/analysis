package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"

	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type DataSyncService struct {
	db     *gorm.DB
	server interface{} // 服务器实例，用于调用API
	cfg    *config.Config
	ctx    context.Context
	cancel context.CancelFunc

	// 同步配置
	config DataSyncConfig

	// 同步器
	syncers map[string]DataSyncer

	// 监控
	monitor *DataSyncMonitor

	// 智能调度器
	smartScheduler *SmartScheduler

	// 数据一致性检查器
	consistencyChecker *DataConsistencyChecker

	// 监控系统
	monitoring *MonitoringSystem

	// Redis客户端 - 跨服务缓存
	redisClient *redis.Client

	// 统计更新定时器
	statsUpdateTicker *time.Ticker
}

type DataSyncConfig struct {
	// 同步间隔（分钟）- 支持小数，如0.5表示30秒
	PriceSyncInterval        float64 `yaml:"price_sync_interval"`
	KlineSyncInterval        float64 `yaml:"kline_sync_interval"`
	FuturesSyncInterval      float64 `yaml:"futures_sync_interval"`
	EnableFundingHistory     bool    `yaml:"enable_funding_history"` // 是否启用历史资金费率获取
	FundingHistoryHours      int     `yaml:"funding_history_hours"`  // 历史资金费率获取的时间范围（小时）
	DepthSyncInterval        float64 `yaml:"depth_sync_interval"`
	ExchangeInfoSyncInterval float64 `yaml:"exchange_info_sync_interval"`

	// 同步参数
	MaxRetries            int  `yaml:"max_retries"`
	RetryDelay            int  `yaml:"retry_delay"` // 秒
	BatchSize             int  `yaml:"batch_size"`
	EnableHistoricalSync  bool `yaml:"enable_historical_sync"`
	EnableIncrementalSync bool `yaml:"enable_incremental_sync"` // 是否启用增量同步
	EnableRealtimeGainers bool `yaml:"enable_realtime_gainers"` // 是否启用实时涨幅榜同步器

	// 实时涨幅榜同步器配置
	RealtimeGainers struct {
		Enabled         bool `yaml:"enabled"`
		TopSymbolsCount int  `yaml:"top_symbols_count"`
		UpdateInterval  int  `yaml:"update_interval"`

		// WebSocket连接配置
		WebSocketReconnectDelay int `yaml:"websocket_reconnect_delay"`
		MaxWebSocketConnections int `yaml:"max_websocket_connections"`

		// 缓存配置
		PriceCacheTTL            int `yaml:"price_cache_ttl"`
		BasePriceRefreshInterval int `yaml:"base_price_refresh_interval"`

		// 变化检测阈值
		ChangeDetectThresholdRank   int     `yaml:"change_detect_threshold_rank"`
		ChangeDetectThresholdPrice  float64 `yaml:"change_detect_threshold_price"`
		ChangeDetectThresholdVolume float64 `yaml:"change_detect_threshold_volume"`

		// 数据库保存配置
		SaveBatchSize int `yaml:"save_batch_size"`
		SaveTimeout   int `yaml:"save_timeout"`

		// 快照管理配置
		CleanupInterval        int `yaml:"cleanup_interval"`
		SnapshotRetentionHours int `yaml:"snapshot_retention_hours"`
		MaxSnapshotsPerKind    int `yaml:"max_snapshots_per_kind"`
	} `yaml:"realtime_gainers"`

	// 初始化涨幅榜填充器配置
	InitialGainersPopulator struct {
		Enabled            bool `yaml:"enabled"`
		PopulateOnStartup  bool `yaml:"populate_on_startup"`
		PopulateThreshold  int  `yaml:"populate_threshold"`
		PopulateLimit      int  `yaml:"populate_limit"`
		DataRetentionHours int  `yaml:"data_retention_hours"`
		CleanupInterval    int  `yaml:"cleanup_interval"`
	} `yaml:"initial_gainers_populator"`

	// 数据源配置
	Exchanges      []string `yaml:"exchanges"`
	Symbols        []string `yaml:"symbols"`
	KlineIntervals []string `yaml:"kline_intervals"`

	// 监控配置
	EnableMetrics   bool `yaml:"enable_metrics"`
	MetricsInterval int  `yaml:"metrics_interval"` // 分钟

	// 数据质量检查
	EnableDataValidation bool `yaml:"enable_data_validation"`
	MaxDataAgeMinutes    int  `yaml:"max_data_age_minutes"`

	// 存储配置
	EnableCompression bool `yaml:"enable_compression"`
	RetentionDays     int  `yaml:"retention_days"`

	// 网络配置
	TimeoutSeconds    int `yaml:"timeout_seconds"`
	RateLimitRequests int `yaml:"rate_limit_requests"`
	RateLimitBurst    int `yaml:"rate_limit_burst"`

	// 并发控制 - 优化参数
	WorkerPoolSize       int `yaml:"worker_pool_size"`
	MaxConcurrentSymbols int `yaml:"max_concurrent_symbols"`
	APICallTimeout       int `yaml:"api_call_timeout"`

	// 缓存配置 - 优化参数
	EnableCaching   bool `yaml:"enable_caching"`
	CacheTTLSeconds int  `yaml:"cache_ttl_seconds"`
	CacheMaxSize    int  `yaml:"cache_max_size"`

	// Redis配置 - 跨服务缓存
	EnableRedisCache bool   `yaml:"enable_redis_cache"`
	RedisAddr        string `yaml:"redis_addr"`
	RedisPassword    string `yaml:"redis_password"`
	RedisDB          int    `yaml:"redis_db"`
	RedisKeyPrefix   string `yaml:"redis_key_prefix"`

	// WebSocket配置 - 高频数据同步
	EnableWebSocketSync          bool `yaml:"enable_websocket_sync"`
	WebSocketBatchInterval       int  `yaml:"websocket_batch_interval"`
	WebSocketMaxSymbols          int  `yaml:"websocket_max_symbols"`
	WebSocketReconnectDelay      int  `yaml:"websocket_reconnect_delay"`
	WebSocketHealthCheckInterval int  `yaml:"websocket_health_check_interval"`
	WebSocketEnableAutoAdjust    bool `yaml:"websocket_enable_auto_adjust"`

	// 智能调度器配置
	SmartScheduler struct {
		Enabled              bool    `yaml:"enabled"`
		CheckInterval        int     `yaml:"check_interval"`
		WebSocketGracePeriod int     `yaml:"websocket_grace_period"`
		RestAPIBackoffFactor float64 `yaml:"rest_api_backoff_factor"`
	} `yaml:"smart_scheduler"`

	// 数据一致性检查器配置
	DataConsistency struct {
		Enabled           bool `yaml:"enabled"`
		CheckInterval     int  `yaml:"check_interval"`
		ConsistencyWindow int  `yaml:"consistency_window"`
		MaxDataAge        int  `yaml:"max_data_age"`
	} `yaml:"data_consistency"`

	// 监控系统配置
	Monitoring struct {
		Enabled       bool `yaml:"enabled"`
		CheckInterval int  `yaml:"check_interval"`
		AlertCooldown int  `yaml:"alert_cooldown"`
		Thresholds    struct {
			WebSocketReconnectThreshold int     `yaml:"websocket_reconnect_threshold"`
			WebSocketDowntimeThreshold  int     `yaml:"websocket_downtime_threshold"`
			APIFailureRateThreshold     float64 `yaml:"api_failure_rate_threshold"`
			APILatencyThreshold         int     `yaml:"api_latency_threshold"`
			DataConsistencyThreshold    float64 `yaml:"data_consistency_threshold"`
			DataAgeThreshold            int     `yaml:"data_age_threshold"`
			MemoryUsageThreshold        float64 `yaml:"memory_usage_threshold"`
			CPUUsageThreshold           float64 `yaml:"cpu_usage_threshold"`
			GoroutineCountThreshold     int     `yaml:"goroutine_count_threshold"`
		} `yaml:"thresholds"`
	} `yaml:"monitoring"`

	// 超时和时间常量配置
	Timeouts struct {
		APICallTimeout              int `yaml:"api_call_timeout"`
		WebSocketReadTimeout        int `yaml:"websocket_read_timeout"`
		WebSocketHealthCheckTimeout int `yaml:"websocket_health_check_timeout"`
		WebSocketReconnectDelay     int `yaml:"websocket_reconnect_delay"`
		DataAgeMax                  int `yaml:"data_age_max"`
		ConsistencyCheckInterval    int `yaml:"consistency_check_interval"`
	} `yaml:"timeouts"`
}

type DataSyncMonitor struct {
	mu        sync.RWMutex
	stats     map[string]map[string]interface{}
	startTime time.Time
}

func NewDataSyncService(db *gorm.DB, server interface{}, cfg *config.Config) *DataSyncService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &DataSyncService{
		db:      db,
		server:  server,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
		config:  DataSyncConfig{}, // 使用零值，依赖配置文件提供所有配置
		syncers: make(map[string]DataSyncer),
		monitor: &DataSyncMonitor{
			stats:     make(map[string]map[string]interface{}),
			startTime: time.Now(),
		},
	}

	// 如果数据库为nil，跳过初始化（将在后续设置数据库后重新初始化）
	if db != nil {
		// 初始化Redis客户端
		service.initRedisClient()

		// 初始化同步器
		service.initSyncers()
	}

	return service
}

// initRedisClient 初始化Redis客户端
func (s *DataSyncService) initRedisClient() {
	if !s.config.EnableRedisCache {
		log.Println("[DataSync] Redis cache disabled, using in-memory cache only")
		return
	}

	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     s.config.RedisAddr,
		Password: s.config.RedisPassword,
		DB:       s.config.RedisDB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("[DataSync] ⚠️ Failed to connect to Redis: %v, falling back to in-memory cache", err)
		return
	}

	s.redisClient = rdb
	log.Printf("[DataSync] ✅ Connected to Redis at %s (DB: %d)", s.config.RedisAddr, s.config.RedisDB)
}

// registerConditionalSyncers 注册需要根据配置条件决定的同步器
func (s *DataSyncService) registerConditionalSyncers() {
	log.Printf("[DataSync] ===== 实时涨幅榜同步器注册检查 =====")
	log.Printf("[DataSync] 检查实时涨幅榜配置: EnableRealtimeGainers=%v", s.config.EnableRealtimeGainers)
	log.Printf("[DataSync] 数据库连接状态: %v", s.db != nil)
	log.Printf("[DataSync] 配置文件状态: %v", s.cfg != nil)

	if s.config.EnableRealtimeGainers {
		log.Printf("[DataSync] ✅ 配置启用，开始创建实时涨幅榜同步器...")

		// 创建现货市场涨幅榜同步器
		realtimeGainersSyncerSpot := NewRealtimeGainersSyncerWithKind(s.db, s.cfg, &s.config, "spot")
		if realtimeGainersSyncerSpot != nil {
			s.syncers["realtime_gainers_spot"] = realtimeGainersSyncerSpot
			log.Printf("[DataSync] ✅ 现货市场实时涨幅榜同步器创建成功")
		} else {
			log.Printf("[DataSync] ❌ 现货市场实时涨幅榜同步器创建失败")
		}

		// 创建期货市场涨幅榜同步器
		realtimeGainersSyncerFutures := NewRealtimeGainersSyncerWithKind(s.db, s.cfg, &s.config, "futures")
		if realtimeGainersSyncerFutures != nil {
			s.syncers["realtime_gainers_futures"] = realtimeGainersSyncerFutures
			log.Printf("[DataSync] ✅ 期货市场实时涨幅榜同步器创建成功")
		} else {
			log.Printf("[DataSync] ❌ 期货市场实时涨幅榜同步器创建失败")
		}

		log.Printf("[DataSync] 当前注册的同步器数量: %d", len(s.syncers))
		log.Printf("[DataSync] 已注册的同步器: %v", getSyncerNames(s.syncers))
	} else {
		log.Printf("[DataSync] ❌ 实时涨幅榜同步器未启用 (配置被禁用)")
	}
	log.Printf("[DataSync] ===== 实时涨幅榜同步器注册检查结束 =====")
}

func (s *DataSyncService) initSyncers() {
	// 创建Redis缓存实例
	redisCache := NewRedisInvalidSymbolCache(s.redisClient, s.config.RedisKeyPrefix, time.Hour*24)

	// 价格同步器
	priceSyncer := NewPriceSyncer(s.db, s.cfg, &s.config, redisCache)
	s.syncers["price"] = priceSyncer

	// K线同步器
	//s.syncers["kline"] = NewKlineSyncer(s.db, s.server, s.cfg, &s.config, redisCache)

	// 期货信息同步器
	s.syncers["futures"] = NewFuturesSyncer(s.db, s.cfg, &s.config)

	// 深度同步器
	//s.syncers["depth"] = NewDepthSyncer(s.db, s.cfg, &s.config, redisCache)

	// 市场统计数据同步器 - 同步24小时市场统计数据，包括价格、交易量、买卖盘口等完整信息
	s.syncers["market_stats"] = NewMarketStatsSyncer(s.db, s.cfg, &s.config, redisCache)

	// 交易对信息同步器
	s.syncers["exchange_info"] = NewExchangeInfoSyncer(s.db, s.cfg, &s.config)

	// 涨幅榜初始化数据填充器 - 系统启动时提供初始涨幅榜数据
	initialGainersPopulator := NewInitialGainersPopulator(s.db, s.cfg, &s.config)
	s.syncers["initial_gainers"] = initialGainersPopulator

	// WebSocket同步器（实验性）
	if s.config.EnableWebSocketSync {
		websocketSyncer := NewWebSocketSyncer(s.db, &s.config)
		s.syncers["websocket"] = websocketSyncer

		// 设置价格同步器的WebSocket引用
		priceSyncer.SetWebSocketSyncer(websocketSyncer)

		// 初始化智能调度器
		if s.config.SmartScheduler.Enabled {
			// 检查必要的同步器是否存在
			klineSyncer, hasKline := s.syncers["kline"]
			if !hasKline {
				log.Printf("[DataSync] ⚠️  Kline syncer not available, skipping smart scheduler initialization")
			} else {
				s.smartScheduler = NewSmartSchedulerWithConfig(
					websocketSyncer,
					klineSyncer.(*KlineSyncer),
					s.syncers["depth"].(*DepthSyncer),
					priceSyncer,
					&s.config,
				)
				log.Printf("[DataSync] Smart scheduler initialized with config")
			}
		}

		// 初始化数据一致性检查器
		if s.config.DataConsistency.Enabled {
			// 检查必要的同步器是否存在
			klineSyncer, hasKline := s.syncers["kline"]
			if !hasKline {
				log.Printf("[DataSync] ⚠️  Kline syncer not available, skipping data consistency checker initialization")
			} else {
				s.consistencyChecker = NewDataConsistencyCheckerWithConfig(
					s.db,
					websocketSyncer,
					klineSyncer.(*KlineSyncer),
					s.syncers["depth"].(*DepthSyncer),
					priceSyncer,
					&s.config,
				)
				log.Printf("[DataSync] Data consistency checker initialized with config")
			}
		}

		// 初始化监控系统
		if s.config.Monitoring.Enabled {
			s.monitoring = NewMonitoringSystem(s)
			log.Printf("[DataSync] Monitoring system initialized")
		}
	}
}

func (s *DataSyncService) Start(initialSyncMode string) error {
	log.Printf("[DataSync] Starting data synchronization service...")

	// 在清理缓存之前，先同步交易对信息，确保数据库数据最新
	if exchangeInfoSyncer, exists := s.syncers["exchange_info"]; exists {
		log.Printf("[DataSync] 📋 Pre-syncing exchange info before cache cleanup...")

		// 创建带超时的上下文，避免阻塞太久（最多30秒）
		syncCtx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		defer cancel()

		if err := exchangeInfoSyncer.Sync(syncCtx); err != nil {
			log.Printf("[DataSync] ⚠️ Failed to pre-sync exchange info: %v", err)
			// 不因为这个错误而停止启动，继续执行
		} else {
			log.Printf("[DataSync] ✅ Exchange info pre-sync completed")
		}
	} else {
		log.Printf("[DataSync] ⚠️ Exchange info syncer not found, skipping pre-sync")
	}

	// 清理Redis缓存中的过期无效符号
	if s.redisClient != nil {
		redisCache := NewRedisInvalidSymbolCache(s.redisClient, s.config.RedisKeyPrefix, time.Hour*24)
		if err := redisCache.CleanupInvalidSymbols(s.db); err != nil {
			log.Printf("[DataSync] ⚠️ Failed to cleanup invalid symbols cache: %v", err)
		} else {
			log.Printf("[DataSync] ✅ Invalid symbols cache cleanup completed")
		}
	}

	// 如果配置中没有指定交易对，则从数据库动态获取
	// 注意：这里获取的是所有交易对，但各个同步器会根据自身需求过滤
	if len(s.config.Symbols) == 0 {
		log.Printf("[DataSync] No symbols configured, fetching from database...")
		symbols, err := pdb.GetUSDTTradingPairs(s.db)
		if err != nil {
			log.Printf("[DataSync] Failed to fetch symbols from database: %v", err)
			return fmt.Errorf("failed to fetch symbols from database: %w", err)
		}

		if len(symbols) == 0 {
			log.Printf("[DataSync] No symbols found in database, using default fallback symbols...")

			// 使用核心交易对作为fallback，避免空列表导致同步失败
			coreSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
			symbols = coreSymbols

			log.Printf("[DataSync] Using %d core symbols as fallback: %v", len(symbols), symbols)
		}

		s.config.Symbols = symbols
		log.Printf("[DataSync] Dynamically loaded %d symbols from database", len(symbols))
	}

	log.Printf("[DataSync] Configuration: Price=%.0fm, Kline=%.0fm, Futures=%.0fm, Depth=%.0fm",
		s.config.PriceSyncInterval, s.config.KlineSyncInterval,
		s.config.FuturesSyncInterval, s.config.DepthSyncInterval)
	log.Printf("[DataSync] Symbols to sync: %d symbols", len(s.config.Symbols))
	log.Printf("[DataSync] Exchanges: %v", s.config.Exchanges)

	// 根据初始同步模式决定如何执行初始同步测试
	switch initialSyncMode {
	case "skip":
		log.Printf("[DataSync] Skipping initial sync test as requested")
	case "ordered":
		log.Printf("[DataSync] Running initial sync test in ordered mode...")

		// 定义首次同步的执行顺序：先同步交易对信息，再同步市场数据，最后同步涨幅榜相关数据
		orderedSyncers := []string{"exchange_info", "market_stats", "initial_gainers", "realtime_gainers"}
		executedSyncers := make(map[string]bool)

		log.Printf("[DataSync] Ordered syncers to test: %v", orderedSyncers)
		log.Printf("[DataSync] Available syncers: %v", getSyncerNames(s.syncers))

		// 按指定顺序执行关键同步器
		for _, syncerName := range orderedSyncers {
			if syncerName == "realtime_gainers" {
				// 特殊处理实时涨幅榜：它被注册为两个同步器 (spot 和 futures)
				realtimeSyncers := []string{"realtime_gainers_spot", "realtime_gainers_futures"}
				allRealtimePassed := true

				for _, rtSyncerName := range realtimeSyncers {
					if syncer, exists := s.syncers[rtSyncerName]; exists {
						log.Printf("[DataSync] Testing syncer: %s (part of %s)", rtSyncerName, syncerName)
						if err := syncer.Sync(s.ctx); err != nil {
							log.Printf("[DataSync] ❌ Initial sync test failed for %s: %v", rtSyncerName, err)
							allRealtimePassed = false
						} else {
							log.Printf("[DataSync] ✅ Initial sync test passed for %s", rtSyncerName)
						}
						executedSyncers[rtSyncerName] = true
					} else {
						log.Printf("[DataSync] ⚠️  Realtime gainers syncer %s not found, skipping", rtSyncerName)
					}
				}

				if allRealtimePassed {
					log.Printf("[DataSync] ✅ Initial sync test passed for realtime_gainers")
				} else {
					log.Printf("[DataSync] ❌ Initial sync test failed for realtime_gainers")
				}
				executedSyncers[syncerName] = true
			} else if syncer, exists := s.syncers[syncerName]; exists {
				log.Printf("[DataSync] Testing syncer: %s (ordered)", syncerName)
				if err := syncer.Sync(s.ctx); err != nil {
					log.Printf("[DataSync] ❌ Initial sync test failed for %s: %v", syncerName, err)
				} else {
					log.Printf("[DataSync] ✅ Initial sync test passed for %s", syncerName)
				}
				executedSyncers[syncerName] = true
			} else {
				log.Printf("[DataSync] ⚠️  Syncer %s not found in syncers map, skipping", syncerName)
			}
		}

		// 执行剩余的同步器（跳过已执行的）
		for name, syncer := range s.syncers {
			if executedSyncers[name] {
				continue
			}
			log.Printf("[DataSync] Testing syncer: %s", name)
			if err := syncer.Sync(s.ctx); err != nil {
				log.Printf("[DataSync] ❌ Initial sync test failed for %s: %v", name, err)
			} else {
				log.Printf("[DataSync] ✅ Initial sync test passed for %s", name)
			}
		}
	case "random":
		log.Printf("[DataSync] Running initial sync test in random mode...")

		// 随机顺序执行所有同步器
		for name, syncer := range s.syncers {
			log.Printf("[DataSync] Testing syncer: %s", name)
			if err := syncer.Sync(s.ctx); err != nil {
				log.Printf("[DataSync] ❌ Initial sync test failed for %s: %v", name, err)
			} else {
				log.Printf("[DataSync] ✅ Initial sync test passed for %s", name)
			}
		}
	default:
		log.Printf("[DataSync] Unknown initial sync mode '%s', defaulting to 'ordered'", initialSyncMode)
		// 递归调用，使用默认的 ordered 模式
		return s.Start("ordered")
	}

	// 启动所有同步器
	for name, syncer := range s.syncers {
		log.Printf("[DataSync] Starting syncer: %s", name)

		var interval time.Duration
		switch name {
		case "price":
			interval = time.Duration(s.config.PriceSyncInterval*60) * time.Second
		case "kline":
			interval = time.Duration(s.config.KlineSyncInterval*60) * time.Second
		case "futures":
			interval = time.Duration(s.config.FuturesSyncInterval*60) * time.Second
		case "depth":
			interval = time.Duration(s.config.DepthSyncInterval*60) * time.Second
		case "market_stats":
			interval = time.Duration(s.config.KlineSyncInterval*60) * time.Second
		case "exchange_info":
			interval = time.Duration(s.config.ExchangeInfoSyncInterval*60) * time.Second
		case "initial_gainers":
			// 初始化填充器只在启动时运行一次，不需要定期运行
			log.Printf("[DataSync] Starting initial gainers populator...")
			go syncer.Start(s.ctx, 0) // 传递0间隔，表示一次性运行
			continue
		default:
			interval = 5 * time.Minute
		}

		log.Printf("[DataSync] %s syncer will run every %v", name, interval)
		go syncer.Start(s.ctx, interval)
	}

	// 启动智能调度器
	if s.smartScheduler != nil {
		log.Printf("[DataSync] Starting smart scheduler for intelligent WebSocket/REST API coordination")
		s.smartScheduler.Start()
	}

	// 启动数据一致性检查器
	if s.consistencyChecker != nil {
		log.Printf("[DataSync] Starting data consistency checker")
		s.consistencyChecker.Start()
	}

	// 启动监控系统
	if s.monitoring != nil {
		log.Printf("[DataSync] Starting monitoring system")
		s.monitoring.Start()
	}

	// WebSocket状态检查
	if websocketSyncer, exists := s.syncers["websocket"]; exists {
		go func() {
			// 等待10秒让WebSocket建立连接
			time.Sleep(10 * time.Second)

			if ws, ok := websocketSyncer.(*WebSocketSyncer); ok {
				healthStatus := ws.GetHealthStatus()
				log.Printf("[DataSync] 📊 WebSocket startup status check:")
				log.Printf("[DataSync]   - Running: %v", healthStatus["is_running"])
				log.Printf("[DataSync]   - Healthy: %v", healthStatus["is_healthy"])
				log.Printf("[DataSync]   - Spot connections: %v/%v healthy",
					healthStatus["healthy_spot"], healthStatus["spot_connections"])
				log.Printf("[DataSync]   - Futures connections: %v/%v healthy",
					healthStatus["healthy_futures"], healthStatus["futures_connections"])
				log.Printf("[DataSync]   - Messages received: %v", healthStatus["messages_received"])
				log.Printf("[DataSync]   - Last message: %v", healthStatus["time_since_last_message"])
			}
		}()
	}

	// 启动监控
	if s.config.EnableMetrics {
		log.Printf("[DataSync] Starting metrics reporter (every %d minutes)", s.config.MetricsInterval)
		go s.startMetricsReporter()
	}

	// 启动心跳日志
	go s.startHeartbeat()

	// 启动健康检查
	go s.startHealthCheck()

	// 启动统计信息更新器
	log.Printf("[DataSync] Starting stats updater")
	s.startStatsUpdater()

	log.Printf("[DataSync] Data synchronization service started successfully")
	log.Printf("[DataSync] Service will continue running. Press Ctrl+C to stop.")
	log.Printf("[DataSync] 💡 Tips:")
	log.Printf("[DataSync]   - Use 'test-sync' to validate all syncers")
	log.Printf("[DataSync]   - Use 'sync-once kline' to test kline sync")
	log.Printf("[DataSync]   - Check logs for detailed performance metrics")

	return nil
}

func (s *DataSyncService) Stop() {
	log.Printf("[DataSync] Stopping data synchronization service...")

	s.cancel()

	// 停止智能调度器
	if s.smartScheduler != nil {
		log.Printf("[DataSync] Stopping smart scheduler")
		s.smartScheduler.Stop()
	}

	// 停止数据一致性检查器
	if s.consistencyChecker != nil {
		log.Printf("[DataSync] Stopping data consistency checker")
		s.consistencyChecker.Stop()
	}

	// 停止监控系统
	if s.monitoring != nil {
		log.Printf("[DataSync] Stopping monitoring system")
		s.monitoring.Stop()
	}

	// 停止所有同步器
	for name, syncer := range s.syncers {
		log.Printf("[DataSync] Stopping syncer: %s", name)
		syncer.Stop()
	}

	log.Printf("[DataSync] Data synchronization service stopped")
}

func (s *DataSyncService) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒心跳一次
	defer ticker.Stop()

	heartbeatCount := 0

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("[DataSync] Heartbeat stopped")
			return
		case <-ticker.C:
			heartbeatCount++
			uptime := time.Since(s.monitor.startTime)

			// 检查数据库连接
			dbHealthy := s.checkDatabaseHealth()

			status := "✅"
			if !dbHealthy {
				status = "❌"
			}

			log.Printf("[DataSync] %s Heartbeat #%d - Uptime: %v - DB: %s",
				status, heartbeatCount, formatDuration(uptime),
				map[bool]string{true: "healthy", false: "unhealthy"}[dbHealthy])
		}
	}
}

func (s *DataSyncService) startHealthCheck() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟进行一次健康检查
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

func (s *DataSyncService) performHealthCheck() {
	log.Printf("[DataSync] 🔍 Performing health check...")

	issues := 0

	// 检查数据库连接
	if !s.checkDatabaseHealth() {
		log.Printf("[DataSync] ❌ Database connection unhealthy")
		issues++
	} else {
		log.Printf("[DataSync] ✅ Database connection healthy")
	}

	// 检查同步器状态
	for name, syncer := range s.syncers {
		stats := syncer.GetStats()
		lastSync, ok := stats["last_sync_time"]
		if !ok {
			log.Printf("[DataSync] ⚠️ %s syncer has no sync history", name)
			issues++
			continue
		}

		// 检查最后同步时间
		if lastSyncTime, ok := lastSync.(time.Time); ok {
			timeSinceLastSync := time.Since(lastSyncTime)
			if timeSinceLastSync > 10*time.Minute {
				log.Printf("[DataSync] ⚠️ %s syncer last synced %v ago", name, timeSinceLastSync)
				issues++
			} else {
				log.Printf("[DataSync] ✅ %s syncer healthy (last sync: %v ago)", name, timeSinceLastSync)
			}
		}
	}

	if issues == 0 {
		log.Printf("[DataSync] 🎉 Health check passed - all systems operational")
	} else {
		log.Printf("[DataSync] ⚠️ Health check found %d issues - check logs above", issues)
	}
}

func (s *DataSyncService) checkDatabaseHealth() bool {
	// 简单的数据库健康检查
	db, err := s.db.DB()
	if err != nil {
		return false
	}

	// 尝试执行一个简单的查询
	var result int
	row := db.QueryRow("SELECT 1")
	err = row.Scan(&result)
	return err == nil && result == 1
}

func (s *DataSyncService) startMetricsReporter() {
	ticker := time.NewTicker(time.Duration(s.config.MetricsInterval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.reportMetrics()
		}
	}
}

func (s *DataSyncService) reportMetrics() {
	log.Printf("[DataSync] === Data Sync Metrics Report ===")

	s.monitor.mu.RLock()
	defer s.monitor.mu.RUnlock()

	totalUptime := time.Since(s.monitor.startTime)

	for name, syncer := range s.syncers {
		stats := syncer.GetStats()
		log.Printf("[DataSync] %s Syncer Stats:", strings.Title(name))
		for key, value := range stats {
			log.Printf("[DataSync]   %s: %v", key, value)
		}
	}

	log.Printf("[DataSync] Total Uptime: %v", totalUptime)
	log.Printf("[DataSync] === End Metrics Report ===")
}

func (s *DataSyncService) SyncOnce(syncerName string) error {
	if syncer, exists := s.syncers[syncerName]; exists {
		log.Printf("[DataSync] Running one-time sync for: %s", syncerName)
		return syncer.Sync(s.ctx)
	}
	return fmt.Errorf("syncer not found: %s", syncerName)
}

func (s *DataSyncService) GetStatus() map[string]interface{} {
	s.monitor.mu.RLock()
	defer s.monitor.mu.RUnlock()

	status := map[string]interface{}{
		"service":    "data_sync",
		"start_time": s.monitor.startTime,
		"uptime":     time.Since(s.monitor.startTime).String(),
		"syncers":    make(map[string]interface{}),
	}

	for name, syncer := range s.syncers {
		status["syncers"].(map[string]interface{})[name] = syncer.GetStats()
	}

	return status
}

func main() {
	// 命令行参数
	action := flag.String("action", "start", "操作类型: start(启动服务), test-sync(测试所有同步器), sync-once(单次同步), status(状态查询)")
	syncerName := flag.String("syncer", "", "同步器名称 (用于sync-once操作)")
	configPath := flag.String("config", "./config.yaml", "配置文件路径")
	initialSyncMode := flag.String("initial-sync-mode", "ordered", "初始同步模式: skip(跳过), ordered(顺序执行), random(随机执行)")

	flag.Parse()

	fmt.Printf("[data_sync] Starting data synchronizati on service, action=%s\n", *action)

	// 一次性读取并解析配置文件
	fmt.Printf("[data_sync] Attempting to load config from: %s\n", *configPath)

	// 获取当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		fmt.Printf("[data_sync] Current working directory: %s\n", cwd)
	}

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("[data_sync] Failed to read config file %s: %v\n", *configPath, err)
		return
	}
	fmt.Printf("[data_sync] Successfully read config file: %s (%d bytes)\n", *configPath, len(configData))

	// 一次性解析整个配置文件
	var fullConfig map[string]interface{}
	if err := yaml.Unmarshal(configData, &fullConfig); err != nil {
		fmt.Printf("[data_sync] Failed to parse config file: %v\n", err)
		return
	}

	// 打印所有顶级配置项
	fmt.Printf("[data_sync] Found top-level config sections:\n")
	for key := range fullConfig {
		fmt.Printf("[data_sync]   - %s\n", key)
	}

	// 将配置数据转换回YAML格式，用于加载主配置
	mainConfigYaml, err := yaml.Marshal(fullConfig)
	if err != nil {
		fmt.Printf("[data_sync] Failed to marshal config for main config: %v\n", err)
		return
	}

	// 加载主配置
	var cfg config.Config
	if err := yaml.Unmarshal(mainConfigYaml, &cfg); err != nil {
		fmt.Printf("[data_sync] Failed to parse main config: %v\n", err)
		return
	}
	config.ApplyProxy(&cfg)

	// 预创建数据同步服务（数据库暂时为nil）
	syncService := NewDataSyncService(nil, nil, &cfg)

	// 加载同步服务配置
	// 从已解析的配置data_sync段加载
	configLoaded := false

	if dataSyncSection, exists := fullConfig["data_sync"]; exists {
		fmt.Printf("[data_sync] Found data_sync section in config\n")
		dataSyncBytes, err := yaml.Marshal(dataSyncSection)
		if err == nil {
			var syncCfg DataSyncConfig
			if err := yaml.Unmarshal(dataSyncBytes, &syncCfg); err == nil {
				// 调试：输出解析后的配置
				fmt.Printf("[data_sync] YAML中包含enable_realtime_gainers: %v\n", containsKey(dataSyncBytes, "enable_realtime_gainers"))

				// 验证配置
				if err := validateSyncConfig(&syncCfg); err != nil {
					fmt.Printf("[data_sync] Invalid sync config in main config: %v\n", err)
					return
				}

				syncService.config = syncCfg
				fmt.Printf("[data_sync] Loaded sync config from main config file: %s\n", *configPath)

				// 调试：输出完整的加载配置内容
				configJson, _ := json.MarshalIndent(syncCfg, "", "  ")
				fmt.Printf("[data_sync] 加载的完整配置内容:\n%s\n", string(configJson))

				configLoaded = true
			}
		}
	}

	if !configLoaded {
		fmt.Printf("[data_sync] Using default configuration\n")
	}

	// 配置加载完毕后，初始化数据库和服务
	// 初始化数据库（优化连接池配置）
	database, err := pdb.OpenMySQL(pdb.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     true,
		MaxOpenConns:    20, // 增加连接数以支持并发同步
		MaxIdleConns:    10, // 增加空闲连接数
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute, // 添加空闲超时
	})
	if err != nil {
		fmt.Printf("[data_sync] Failed to connect to database: %v\n", err)
		return
	}
	defer database.Close()

	gdb, err := database.DB()
	if err != nil {
		fmt.Printf("[data_sync] Failed to get database instance: %v\n", err)
		return
	}

	// 设置数据库连接到已创建的服务
	syncService.db = gdb

	// 重新初始化依赖数据库的组件
	syncService.initRedisClient()
	syncService.initSyncers()

	// 注册条件同步器（需要在数据库和配置都准备好后进行）
	if configLoaded {
		syncService.registerConditionalSyncers()
	}

	// 最终配置验证
	if err := validateSyncConfig(&syncService.config); err != nil {
		fmt.Printf("[data_sync] Final configuration validation failed: %v\n", err)
		return
	}

	// 处理不同操作
	switch *action {
	case "test-sync":
		// 测试所有同步器
		fmt.Println("[data_sync] Starting test sync for all syncers...")
		fmt.Println("[data_sync] This will test each syncer once and show detailed results")

		totalSyncers := len(syncService.syncers)
		successfulSyncers := 0

		for name, syncer := range syncService.syncers {
			fmt.Printf("[data_sync] Testing syncer: %s\n", name)
			startTime := time.Now()

			if err := syncer.Sync(syncService.ctx); err != nil {
				fmt.Printf("[data_sync] ❌ %s sync failed: %v\n", name, err)
			} else {
				duration := time.Since(startTime)
				fmt.Printf("[data_sync] ✅ %s sync succeeded in %v\n", name, duration)
				successfulSyncers++
			}

			// 显示统计信息
			stats := syncer.GetStats()
			fmt.Printf("[data_sync]   Stats: %v\n", stats)
			fmt.Println()
		}

		fmt.Printf("[data_sync] Test sync completed: %d/%d syncers successful\n", successfulSyncers, totalSyncers)

		if successfulSyncers == totalSyncers {
			fmt.Println("[data_sync] 🎉 All syncers are working correctly!")
		} else {
			fmt.Printf("[data_sync] ⚠️  %d syncers have issues, check logs above\n", totalSyncers-successfulSyncers)
		}

		return

	case "start":
		// 启动服务
		if err := syncService.Start(*initialSyncMode); err != nil {
			fmt.Printf("[data_sync] Failed to start service: %v\n", err)
			return
		}

		// 等待信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		fmt.Println("[data_sync] Service started. Press Ctrl+C to stop.")

		<-sigChan
		fmt.Println("\n[data_sync] Received shutdown signal")

		syncService.Stop()

	case "sync-once":
		// 单次同步
		if *syncerName == "" {
			fmt.Println("[data_sync] Error: syncer name is required for sync-once operation")
			fmt.Println("[data_sync] Available syncers: price, kline, futures, depth, market_stats, exchange_info, initial_gainers, realtime_gainers")
			fmt.Println("[data_sync] Example: -action sync-once -syncer price")
			return
		}

		fmt.Printf("[data_sync] Starting one-time sync for syncer: %s\n", *syncerName)
		startTime := time.Now()

		if err := syncService.SyncOnce(*syncerName); err != nil {
			fmt.Printf("[data_sync] ❌ Sync failed for %s: %v\n", *syncerName, err)
			os.Exit(1)
		} else {
			duration := time.Since(startTime)
			fmt.Printf("[data_sync] ✅ Sync completed successfully for %s in %v\n", *syncerName, duration)
		}

	case "status":
		// 查询状态
		status := syncService.GetStatus()
		fmt.Printf("[data_sync] Service Status:\n")
		fmt.Printf("  Uptime: %v\n", status["uptime"])
		fmt.Printf("  Start Time: %v\n", status["start_time"])
		fmt.Printf("  Configured Symbols: %v\n", syncService.config.Symbols)
		fmt.Printf("  Total Symbols: %d\n", len(syncService.config.Symbols))

		if syncers, ok := status["syncers"].(map[string]interface{}); ok {
			fmt.Printf("  Syncers:\n")
			for name, stats := range syncers {
				fmt.Printf("    %s:\n", name)
				if statsMap, ok := stats.(map[string]interface{}); ok {
					for key, value := range statsMap {
						fmt.Printf("      %s: %v\n", key, value)
					}
				}
			}
		}

	default:
		fmt.Printf("[data_sync] Unknown action: %s\n", *action)
		fmt.Println("[data_sync] Available actions:")
		fmt.Println("[data_sync]   start     - 启动数据同步服务")
		fmt.Println("[data_sync]   test-sync - 测试所有同步器功能")
		fmt.Println("[data_sync]   sync-once - 单次同步指定同步器")
		fmt.Println("[data_sync]   status    - 查看服务状态")
		fmt.Println("[data_sync] Examples:")
		fmt.Println("[data_sync]   -action start")
		fmt.Println("[data_sync]   -action start -initial-sync-mode=skip")    // 跳过初始同步测试
		fmt.Println("[data_sync]   -action start -initial-sync-mode=random")  // 随机顺序执行初始同步
		fmt.Println("[data_sync]   -action start -initial-sync-mode=ordered") // 顺序执行初始同步（默认）
		fmt.Println("[data_sync]   -action test-sync")
		fmt.Println("[data_sync]   -action sync-once -syncer price")
		os.Exit(1)
	}
}

// 工具函数：解析字符串数组
func parseStringArray(str string) []string {
	if str == "" {
		return nil
	}
	return strings.Split(str, ",")
}

// validateSyncConfig 验证同步配置的有效性
func validateSyncConfig(config *DataSyncConfig) error {
	// 验证交易对（如果配置了的话）
	for _, symbol := range config.Symbols {
		if symbol == "" {
			return fmt.Errorf("empty symbol found in configuration")
		}
		// 验证交易对格式 (应以USDT结尾)
		if !strings.HasSuffix(strings.ToUpper(symbol), "USDT") {
			return fmt.Errorf("invalid symbol format: %s (should end with USDT)", symbol)
		}
	}

	// 验证交易所
	validExchanges := map[string]bool{"binance": true, "okx": true, "huobi": true}
	for _, exchange := range config.Exchanges {
		if !validExchanges[strings.ToLower(exchange)] {
			return fmt.Errorf("unsupported exchange: %s", exchange)
		}
	}

	// 验证K线间隔
	validIntervals := map[string]bool{
		"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "2h": true, "4h": true, "6h": true, "8h": true, "12h": true,
		"1d": true, "3d": true, "1w": true, "1M": true,
	}
	for _, interval := range config.KlineIntervals {
		if !validIntervals[interval] {
			return fmt.Errorf("invalid kline interval: %s", interval)
		}
	}

	// 验证时间间隔（支持小数，如0.5表示30秒）
	if config.PriceSyncInterval <= 0 || config.PriceSyncInterval > 3600 {
		return fmt.Errorf("invalid price sync interval: %.1f (must be 0.1-3600 minutes)", config.PriceSyncInterval)
	}
	if config.KlineSyncInterval <= 0 || config.KlineSyncInterval > 3600 {
		return fmt.Errorf("invalid kline sync interval: %.1f (must be 0.1-3600 minutes)", config.KlineSyncInterval)
	}
	if config.FuturesSyncInterval <= 0 || config.FuturesSyncInterval > 3600 {
		return fmt.Errorf("invalid futures sync interval: %.1f (must be 0.1-3600 minutes)", config.FuturesSyncInterval)
	}
	if config.FundingHistoryHours < 0 || config.FundingHistoryHours > 720 {
		return fmt.Errorf("invalid funding history hours: %d (must be 0-720 hours, 0 means use default 4 hours)", config.FundingHistoryHours)
	}
	if config.DepthSyncInterval <= 0 || config.DepthSyncInterval > 3600 {
		return fmt.Errorf("invalid depth sync interval: %.1f (must be 0.1-3600 minutes)", config.DepthSyncInterval)
	}
	if config.ExchangeInfoSyncInterval <= 0 || config.ExchangeInfoSyncInterval > 3600 {
		return fmt.Errorf("invalid exchange info sync interval: %.1f (must be 0.1-3600 minutes)", config.ExchangeInfoSyncInterval)
	}

	// 验证其他参数
	if config.MaxRetries < 0 || config.MaxRetries > 10 {
		return fmt.Errorf("invalid max retries: %d (must be 0-10)", config.MaxRetries)
	}
	if config.BatchSize <= 0 || config.BatchSize > 1000 {
		return fmt.Errorf("invalid batch size: %d (must be 1-1000)", config.BatchSize)
	}

	return nil
}

// 工具函数：格式化持续时间
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// startStatsUpdater 启动统计信息更新器
func (s *DataSyncService) startStatsUpdater() {
	s.statsUpdateTicker = time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-s.statsUpdateTicker.C:
				s.updateGlobalStats()
			case <-s.ctx.Done():
				s.statsUpdateTicker.Stop()
				return
			}
		}
	}()
}

// updateGlobalStats 更新全局统计信息
func (s *DataSyncService) updateGlobalStats() {
	// 由于DataSyncStats的字段是私有的，我们通过AddAlert等函数来间接更新
	// 这里暂时不实现复杂的统计收集逻辑，后续可以扩展
	log.Printf("[DataSync] Stats update triggered (placeholder implementation)")
}

// getSyncerDisplayName 获取同步器显示名称
func (s *DataSyncService) getSyncerDisplayName(name string) string {
	names := map[string]string{
		"price":     "价格同步器",
		"kline":     "K线同步器",
		"depth":     "深度同步器",
		"websocket": "WebSocket同步器",
	}
	if displayName, exists := names[name]; exists {
		return displayName
	}
	return name
}

// getSyncerNames 获取同步器名称列表
func getSyncerNames(syncers map[string]DataSyncer) []string {
	names := make([]string, 0, len(syncers))
	for name := range syncers {
		names = append(names, name)
	}
	return names
}

// containsKey 检查YAML数据中是否包含指定的键
func containsKey(yamlData []byte, key string) bool {
	return bytes.Contains(yamlData, []byte(key+":"))
}
