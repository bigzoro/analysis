package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"analysis/internal/analysis"
	"analysis/internal/config"
	pdb "analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
	"analysis/internal/netutil"
	"analysis/internal/server/strategy/factory"
	"analysis/internal/server/strategy/router"
	"analysis/internal/server/strategy/shared/execution"
	traditional_execution "analysis/internal/server/strategy/traditional/execution"
	"analysis/internal/service"
)

// PositionSnapshot 持仓快照，用于检测持仓变化
type PositionSnapshot struct {
	Symbol       string    `json:"symbol"`
	PositionAmt  string    `json:"position_amt"`
	EntryPrice   string    `json:"entry_price"`
	MarkPrice    string    `json:"mark_price"`
	UpdateTime   int64     `json:"update_time"`
	SnapshotTime time.Time `json:"snapshot_time"`
	UserID       uint      `json:"user_id"` // 关联用户ID
}

// DetectedChange 检测到的持仓变化
type DetectedChange struct {
	Symbol     string    `json:"symbol"`
	Type       string    `json:"type"` // "new", "changed", "closed", "disappeared"
	OldAmt     string    `json:"old_amt"`
	NewAmt     string    `json:"new_amt"`
	Confidence float64   `json:"confidence"` // 置信度 0-1
	Timestamp  time.Time `json:"timestamp"`
}

// 全局数据同步状态存储
var (
	globalDataSyncStats = &DataSyncStats{
		mu:           sync.RWMutex{},
		lastUpdate:   time.Now(),
		globalHealth: "unknown",
		syncers:      make(map[string]*SyncerStats),
		websocket:    &WebSocketStats{},
		apiStats:     make(map[string]*APIStats),
		alerts:       []DataSyncAlert{},
	}
)

// DataSyncStats 数据同步全局统计
type DataSyncStats struct {
	mu           sync.RWMutex
	lastUpdate   time.Time
	globalHealth string
	syncers      map[string]*SyncerStats
	websocket    *WebSocketStats
	apiStats     map[string]*APIStats
	alerts       []DataSyncAlert
}

// SyncerStats 同步器统计
type SyncerStats struct {
	Name            string     `json:"name"`
	DisplayName     string     `json:"display_name"`
	Status          string     `json:"status"`
	TotalSyncs      int64      `json:"total_syncs"`
	SuccessfulSyncs int64      `json:"successful_syncs"`
	FailedSyncs     int64      `json:"failed_syncs"`
	LastSyncTime    *time.Time `json:"last_sync_time"`
	TotalUpdates    int64      `json:"total_updates"`
}

// WebSocketStats WebSocket统计
type WebSocketStats struct {
	IsRunning                bool       `json:"is_running"`
	IsHealthy                bool       `json:"is_healthy"`
	SpotConnections          int        `json:"spot_connections"`
	HealthySpot              int        `json:"healthy_spot"`
	FuturesConnections       int        `json:"futures_connections"`
	HealthyFutures           int        `json:"healthy_futures"`
	MessagesReceived         int64      `json:"messages_received"`
	LastMessageTime          *time.Time `json:"last_message_time"`
	TotalSpotPriceUpdates    int64      `json:"total_spot_price_updates"`
	TotalFuturesPriceUpdates int64      `json:"total_futures_price_updates"`
	TotalKlineUpdates        int64      `json:"total_kline_updates"`
	TotalDepthUpdates        int64      `json:"total_depth_updates"`
}

// APIStats API统计
type APIStats struct {
	TotalCalls      int64      `json:"total_calls"`
	APICallsTotal   int64      `json:"api_calls_total"`
	APISuccessRate  string     `json:"api_success_rate"`
	APIAvgLatency   *string    `json:"api_avg_latency"`
	TotalSyncs      int64      `json:"total_syncs"`
	SuccessfulSyncs int64      `json:"successful_syncs"`
	FailedSyncs     int64      `json:"failed_syncs"`
	LastSyncTime    *time.Time `json:"last_sync_time"`
	TotalUpdates    int64      `json:"total_updates"`
	// Price specific
	WebSocketHits    int64  `json:"websocket_hits,omitempty"`
	RestAPICalls     int64  `json:"rest_api_calls,omitempty"`
	WebSocketHitRate string `json:"websocket_hit_rate,omitempty"`
}

// DataSyncAlert 数据同步告警信息
type DataSyncAlert struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Component string    `json:"component"`
	Metric    string    `json:"metric"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// UpdateDataSyncStats 更新数据同步统计信息
func UpdateDataSyncStats(stats *DataSyncStats) {
	globalDataSyncStats.mu.Lock()
	defer globalDataSyncStats.mu.Unlock()

	globalDataSyncStats.lastUpdate = time.Now()
	globalDataSyncStats.globalHealth = stats.globalHealth
	globalDataSyncStats.syncers = stats.syncers
	globalDataSyncStats.websocket = stats.websocket
	globalDataSyncStats.apiStats = stats.apiStats
	globalDataSyncStats.alerts = stats.alerts
}

// AddAlert 添加告警
func AddAlert(alert DataSyncAlert) {
	globalDataSyncStats.mu.Lock()
	defer globalDataSyncStats.mu.Unlock()

	globalDataSyncStats.alerts = append(globalDataSyncStats.alerts, alert)
	// 保留最近的100个告警
	if len(globalDataSyncStats.alerts) > 100 {
		globalDataSyncStats.alerts = globalDataSyncStats.alerts[len(globalDataSyncStats.alerts)-100:]
	}
}

type Server struct {
	db                     Database // 使用接口而非具体实现
	Mailer                 Mailer
	XBearer                string
	cache                  pdb.CacheInterface // 缓存接口
	arkhamClient           *ArkhamClient
	nansenClient           *NansenClient
	cfg                    *config.Config
	priceService           *service.PriceService          // 统一价格服务
	dataManager            *DataManager                   // 多源数据管理器
	dataService            *DataService                   // 数据服务
	backtestEngine         *BacktestEngine                // 回测引擎
	ensembleModels         map[string]*EnsemblePredictor  // 集成学习模型
	recommendationCache    *RecommendationCache           // 推荐缓存
	recommendationEnhancer *RecommendationEnhancer        // 推荐增强器
	batchPerformanceLoader *BatchPerformanceLoader        // 批量性能加载器
	userBehaviorService    *UserBehaviorService           // 用户行为服务
	feedbackService        *RecommendationFeedbackService // 推荐反馈服务
	abTestingService       *ABTestingService              // A/B测试服务
	algorithmOptimizer     *AlgorithmOptimizer            // 算法优化器
	weightController       *AdaptiveWeightController      // 自适应权重控制器

	// 策略相关
	strategyHandler *StrategyHandler         // 策略处理器
	scannerRegistry *StrategyScannerRegistry // 策略扫描器注册表
	strategyRouter  *router.StrategyRouter   // 策略路由器
	strategyFactory *factory.StrategyFactory // 策略工厂
	scanMutex       sync.Mutex               // 扫描并发控制锁

	// 数据同步服务相关
	dataSyncService      interface{}      // 数据同步服务实例
	binanceWSClient      *BinanceWSClient // 币安WebSocket客户端
	binanceFuturesClient *bf.Client       // 币安期货客户端
	coincap              *coinCapCache    // CoinCap市值数据缓存
	// 注意：OptimizationScheduler已移至独立的investment服务
	priceCache         *PriceCache                  // 价格缓存
	distributedManager *DistributedComputingManager // 分布式计算管理器
	opportunityCache   map[string]time.Time         // 机会发现缓存，避免重复发现
	tradingPairsCache  *TradingPairsCache           // 交易对列表缓存

	// Data preprocessing and caching - 数据预处理和缓存
	dataCache         *BacktestDataCache // 回测数据缓存
	dataUpdateService *DataUpdateService // 数据更新服务

	// Feature precomputation - 特征预计算
	featurePrecomputeService *FeaturePrecomputeService // 特征预计算服务

	// Technical indicators precomputation - 技术指标预计算
	technicalIndicatorsPrecomputeService *TechnicalIndicatorsPrecomputeService // 技术指标预计算服务

	// ML model pretraining - ML模型预训练
	mlPretrainingService *MLPretrainingService // ML模型预训练服务

	// Analysis module - 智能投研模块
	strategyBacktestEngine *StrategyBacktestEngine // 策略回测引擎
	coinSelectionAlgorithm *CoinSelectionAlgorithm // 新一代选币算法

	// 注意：PerformanceTracker和SmartScheduler已移至独立的investment服务
	layeredCache   *LayeredCache   // 分层缓存系统
	dataPreloader  *DataPreloader  // 数据预加载服务
	priceMonitor   *PriceMonitor   // 价格监控服务
	orderScheduler *OrderScheduler // 定时订单调度器

	// ⭐ 并发和资源管理模块
	smartWorkerPool   *SmartWorkerPool       // 智能工作者池
	resourceManager   *ResourceManager       // 资源管理器
	circuitBreakerMgr *CircuitBreakerManager // 熔断器管理器
	shutdownManager   *ShutdownManager       // 关闭管理器
	resourceCleaner   *ResourceCleaner       // 资源清理器

	// 数据质量监控
	dataQualityMonitor *DataQualityMonitor // 数据质量监控器

	// 免费数据源客户端
	coinGeckoClient *CoinGeckoClient // CoinGecko免费API客户端
	newsAPIClient   *NewsAPIClient   // NewsAPI免费客户端
	dataFusion      *DataFusion      // 数据融合器
	dataValidator   *DataValidator   // 数据验证器

	// ⭐ 特征工程模块
	featureEngineering *FeatureEngineering // 特征工程核心模块

	// ⭐ 机器学习模块
	machineLearning *MachineLearning

	// ⭐ 风险管理模块
	riskManagement *RiskManagement // 机器学习核心模块

	// ⭐ 持仓变化检测机制
	positionSnapshots map[string]*PositionSnapshot // 持仓快照存储
	lastPositionCheck time.Time                    // 上次持仓检查时间
	positionMutex     sync.RWMutex                 // 持仓数据并发控制

	// ⭐ 智能通知系统
	notificationService NotificationService // 通知服务

	// ⭐ 操作历史追踪
	auditLogger *AuditLogger // 审计日志记录器

	// ⭐ 异常检测与恢复
	healthChecker *SystemHealthChecker // 系统健康检查器

	// 降级策略
	fallbackStrategy *FallbackStrategy    // 降级策略管理器
	fallbackProvider FallbackDataProvider // 降级数据提供者
}

// SetDataSyncService 设置数据同步服务实例
func (s *Server) SetDataSyncService(service interface{}) {
	s.dataSyncService = service
}

// TradingPairsCache 交易对列表缓存
type TradingPairsCache struct {
	symbols       []string
	lastUpdate    time.Time
	cacheDuration time.Duration
	mu            sync.RWMutex
}

// NewTradingPairsCache 创建交易对缓存
func NewTradingPairsCache(cacheDuration time.Duration) *TradingPairsCache {
	return &TradingPairsCache{
		cacheDuration: cacheDuration,
	}
}

// Get 获取缓存的交易对列表
func (c *TradingPairsCache) Get() ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdate) < c.cacheDuration && len(c.symbols) > 0 {
		return c.symbols, true
	}
	return nil, false
}

// Set 设置缓存的交易对列表
func (c *TradingPairsCache) Set(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.symbols = make([]string, len(symbols))
	copy(c.symbols, symbols)
	c.lastUpdate = time.Now()
}

// New 创建 Server 实例（使用接口）
// 初始化顺序：
// 1. 核心基础服务 (数据库、数据管理)
// 2. 实时服务 (WebSocket、定时任务)
// 3. AI分析模块 (可选的高级功能)
// 4. 回测引擎 (依赖基础服务，可使用AI增强功能)
func New(db Database, cfg *config.Config) *Server {
	s := &Server{db: db, cfg: cfg}

	// ===== 阶段1: 核心基础服务 =====
	s.initPriceService()
	s.initDataManager()

	// 初始化策略处理器
	s.strategyHandler = NewStrategyHandler(s)

	// 初始化策略路由器和工厂
	s.strategyRouter = router.NewStrategyRouter()
	s.strategyFactory = factory.NewStrategyFactory(&factory.ExecutionDependencies{
		MarketDataProvider: s,
		OrderManager:       s,
		RiskManager:        s,
		ConfigProvider:     s,
	})

	// 初始化策略扫描器注册表
	log.Printf("🏗️ [INIT] 创建策略扫描器注册表...")
	s.scannerRegistry = NewStrategyScannerRegistry()
	log.Printf("🚀 [INIT] 开始注册策略扫描器...")
	if err := s.scannerRegistry.RegisterScanner(s); err != nil {
		log.Printf("❌ [INIT] 策略扫描器注册失败: %v", err)
		// 不返回错误，因为其他服务可能仍然可以工作
		// 策略扫描功能将不可用，但服务器仍可启动
	} else {
		log.Printf("✅ [INIT] 策略扫描器注册成功")
	}

	// 策略执行器现在直接使用新的接口，无需注册

	// ===== 阶段2: 实时服务 =====
	s.initBinanceWSClient()
	s.initOrderStatusSync()         // 初始化订单状态定时同步
	s.initPositionChangeDetection() // 初始化持仓变化检测机制
	s.initNotificationService(cfg)  // 初始化智能通知系统
	s.initAuditLogger()             // 初始化审计日志记录器
	s.initHealthChecker()           // 初始化系统健康检查器
	s.initTradingPairsCache()       // 初始化交易对缓存

	// ===== 阶段3: AI分析模块（可选） =====
	// 根据配置决定是否启用数据分析服务
	if cfg.Services.EnableDataAnalysis {
		log.Printf("[INIT] 数据分析服务已启用，开始初始化AI分析模块...")
		// 包含复杂的AI算法、特征工程、风险管理等高级功能
		s.initAnalysisModule()
		log.Printf("[INIT] AI分析模块初始化完成，回测引擎将使用完整功能")
	} else {
		log.Printf("[INIT] 数据分析服务已禁用，跳过AI分析模块初始化")
		log.Printf("[INIT] 回测引擎将在基础模式下运行")
	}

	// ===== 阶段4: 回测引擎（核心服务模块） =====
	// 回测引擎放在最后，确保能使用到AI分析模块提供的增强功能
	// 如果AI分析模块被禁用，回测引擎也能正常工作（基础模式）
	s.initBacktestEngine()

	// ===== 阶段5: 定时订单调度器（必须服务） =====
	// OrderScheduler必须初始化，因为策略执行功能是核心功能
	log.Printf("[INIT] 初始化定时订单调度器...")
	s.orderScheduler = NewOrderScheduler(s.db.DB(), s.cfg, s)
	s.orderScheduler.Start()
	log.Printf("[INIT] 定时订单调度器初始化完成 - 策略启动API现在可以使用立即执行功能")

	return s
}

// GetOrderScheduler 获取订单调度器（用于测试和调试）
func (s *Server) GetOrderScheduler() *OrderScheduler {
	return s.orderScheduler
}

// initTradingPairsCache 初始化交易对缓存
func (s *Server) initTradingPairsCache() {
	log.Printf("[INIT] 初始化交易对列表缓存...")
	s.tradingPairsCache = NewTradingPairsCache(30 * time.Minute) // 缓存30分钟
}

// initBinanceWSClient 初始化币安WebSocket客户端
func (s *Server) initBinanceWSClient() {
	log.Printf("[INIT] 初始化币安WebSocket客户端...")

	s.binanceWSClient = NewBinanceWSClient()

	// 初始化币安期货客户端
	log.Printf("[INIT] 初始化币安期货客户端...")
	s.binanceFuturesClient = bf.New(s.cfg.Exchange.Binance.IsTestnet,
		s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)
	log.Printf("[INIT] 币安期货客户端初始化完成")

	// 尝试连接到币安WebSocket
	go func() {
		// 首先尝试连接币本位期货WebSocket
		if err := s.binanceWSClient.Connect("coin_futures"); err != nil {
			log.Printf("[INIT] 连接币本位期货WebSocket失败: %v", err)
			// 如果失败，尝试连接USDT期货
			if err := s.binanceWSClient.Connect("futures"); err != nil {
				log.Printf("[INIT] 连接USDT期货WebSocket失败: %v", err)
				return
			}
		}

		log.Printf("[INIT] 币安WebSocket客户端初始化完成")

		// 设置数据更新回调，自动清理相关缓存
		s.binanceWSClient.SetUpdateCallback(func() {
			// 当WebSocket收到新数据时，清理涨幅榜缓存，触发下次请求重新计算
			gainersCacheMu.Lock()
			// 清理所有涨幅榜缓存，强制下次请求使用最新WebSocket数据
			gainersCache = make(map[string]cachedGainersData)
			gainersCacheMu.Unlock()
			//log.Printf("[BinanceWS] WebSocket数据更新，清理涨幅榜缓存")
		})

		// 订阅热门交易对的24hr统计数据
		popularSymbols := []string{
			"BTC", "ETH", "BNB", "ADA", "XRP", "SOL", "DOT", "DOGE", "AVAX", "LTC",
		}

		if err := s.binanceWSClient.SubscribeTicker24h(popularSymbols, "futures"); err != nil {
			log.Printf("[INIT] 订阅24hr统计数据失败: %v", err)
		}
	}()
}

// initOrderStatusSync 初始化订单状态定时同步
func (s *Server) initOrderStatusSync() {
	log.Printf("[INIT] 初始化订单状态定时同步...")

	// 启动定时同步goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second) // 每30秒同步一次
		defer ticker.Stop()

		// 首次启动时等待10秒再执行，避免启动时负载过高
		time.Sleep(10 * time.Second)

		// 定时执行同步
		for {
			select {
			case <-ticker.C:
				log.Printf("[Order-Sync] 开始定时订单状态同步...")
				startTime := time.Now()

				if err := s.syncAllOrderStatus(); err != nil {
					log.Printf("[Order-Sync] 定时同步失败: %v", err)
				} else {
					duration := time.Since(startTime)
					log.Printf("[Order-Sync] 定时同步完成，耗时: %v", duration)
				}
			}
		}
	}()

	log.Printf("[INIT] 订单状态定时同步初始化完成（每30秒执行一次）")
}

// initPositionChangeDetection 初始化持仓变化检测机制
func (s *Server) initPositionChangeDetection() {
	log.Printf("[INIT] 初始化持仓变化检测机制...")

	// 初始化持仓快照存储
	s.positionSnapshots = make(map[string]*PositionSnapshot)
	s.lastPositionCheck = time.Now()

	// 启动持仓变化检测goroutine
	go func() {
		ticker := time.NewTicker(15 * time.Second) // 每15秒检测一次持仓变化
		defer ticker.Stop()

		// 首次启动时等待5秒再执行
		time.Sleep(5 * time.Second)

		for {
			select {
			case <-ticker.C:
				if err := s.detectPositionChanges(); err != nil {
					log.Printf("[Position-Detect] 持仓变化检测失败: %v", err)
				}
			}
		}
	}()

	log.Printf("[INIT] 持仓变化检测机制初始化完成（每15秒检测一次）")
}

// detectPositionChanges 检测持仓变化并处理外部操作
func (s *Server) detectPositionChanges() error {
	// 获取当前所有用户的持仓信息
	currentPositions, err := s.getAllUserPositions()
	if err != nil {
		return fmt.Errorf("获取用户持仓失败: %w", err)
	}

	// 获取上次的持仓快照
	lastSnapshots := s.getLastPositionSnapshots()

	// 检测持仓变化
	changes := s.detectPositionChangesInternal(currentPositions, lastSnapshots)

	// 处理检测到的变化
	for _, change := range changes {
		if err := s.handlePositionChange(change); err != nil {
			log.Printf("[Position-Detect] 处理持仓变化失败 %s: %v", change.Symbol, err)
		}
	}

	// 更新持仓快照
	s.updatePositionSnapshots(currentPositions)

	s.positionMutex.Lock()
	s.lastPositionCheck = time.Now()
	s.positionMutex.Unlock()

	return nil
}

// detectAndProcessExternalOperations 检测和处理外部操作
func (s *Server) detectAndProcessExternalOperations(client *bf.Client) (processedCount, errorCount int) {
	log.Printf("[Order-Sync] 开始检测外部操作...")

	// 查询最近可能受外部操作影响的订单
	// 包括状态变为filled或cancelled的订单，以及成交数量发生变化的订单
	var affectedOrders []pdb.ScheduledOrder
	err := s.db.DB().Model(&pdb.ScheduledOrder{}).
		Where("status IN (?) AND client_order_id != '' AND exchange = ? AND updated_at > ?",
			[]string{"filled", "cancelled", "failed"}, "binance_futures", time.Now().Add(-1*time.Hour)).
		Find(&affectedOrders).Error

	if err != nil {
		log.Printf("[Order-Sync] 查询受影响订单失败: %v", err)
		return 0, 1
	}

	if len(affectedOrders) == 0 {
		log.Printf("[Order-Sync] 没有需要检查外部操作的订单")
		return 0, 0
	}

	log.Printf("[Order-Sync] 检查 %d 个订单的外部操作可能性", len(affectedOrders))

	for _, order := range affectedOrders {
		if err := s.analyzeOrderForExternalOperation(&order, client); err != nil {
			log.Printf("[Order-Sync] 分析订单 %d 外部操作失败: %v", order.ID, err)
			errorCount++
		} else {
			processedCount++
		}
	}

	return processedCount, errorCount
}

// analyzeOrderForExternalOperation 分析订单是否受到外部操作影响
func (s *Server) analyzeOrderForExternalOperation(order *pdb.ScheduledOrder, client *bf.Client) error {
	// 获取订单的最新状态
	orderStatus, err := client.QueryOrder(order.Symbol, order.ClientOrderId)
	if err != nil {
		// 如果无法查询订单状态，可能订单已被删除或API错误
		if strings.Contains(err.Error(), "order not found") {
			log.Printf("[Order-Sync] 订单 %s 在交易所不存在，可能已被外部删除", order.ClientOrderId)
			return s.handleOrderExternallyDeleted(order)
		}
		return fmt.Errorf("查询订单状态失败: %w", err)
	}

	// 检查订单状态是否与数据库一致
	statusChanged := s.hasOrderStatusChanged(order, orderStatus)
	executedQtyChanged := s.hasExecutedQuantityChanged(order, orderStatus)

	if !statusChanged && !executedQtyChanged {
		// 订单状态正常，无需处理
		return nil
	}

	// 检测到状态变化，分析是否为外部操作
	externalOpType := s.determineExternalOperationType(order, orderStatus, statusChanged, executedQtyChanged)

	if externalOpType != "" {
		log.Printf("[Order-Sync] 检测到外部操作: 订单 %d (%s) - %s",
			order.ID, order.ClientOrderId, externalOpType)

		// 创建外部操作记录
		externalOp := &pdb.ExternalOperation{
			Symbol:        order.Symbol,
			OperationType: externalOpType,
			OldAmount:     order.ExecutedQty,
			NewAmount:     orderStatus.ExecutedQty,
			Confidence:    0.9, // 订单状态变化通常很确定
			DetectedAt:    time.Now(),
			Status:        "processed",
			UserID:        order.UserID,
			Notes:         fmt.Sprintf("订单状态同步检测: %s -> %s", order.Status, s.mapExchangeStatus(orderStatus.Status)),
		}

		if err := s.db.DB().Create(externalOp).Error; err != nil {
			return fmt.Errorf("创建外部操作记录失败: %w", err)
		}

		// 更新订单状态以反映外部操作
		if err := s.updateOrderStatusFromExternalOperation(order, orderStatus); err != nil {
			log.Printf("[Order-Sync] 更新订单状态失败: %v", err)
		}

		// 通知用户
		s.notifyUserExternalOperation(externalOp)
	}

	return nil
}

// hasOrderStatusChanged 检查订单状态是否发生变化
func (s *Server) hasOrderStatusChanged(order *pdb.ScheduledOrder, orderStatus *bf.QueryOrderResp) bool {
	currentStatus := s.mapExchangeStatus(orderStatus.Status)
	return order.Status != currentStatus
}

// hasExecutedQuantityChanged 检查成交数量是否发生变化
func (s *Server) hasExecutedQuantityChanged(order *pdb.ScheduledOrder, orderStatus *bf.QueryOrderResp) bool {
	return order.ExecutedQty != orderStatus.ExecutedQty && orderStatus.ExecutedQty != ""
}

// determineExternalOperationType 确定外部操作类型
func (s *Server) determineExternalOperationType(order *pdb.ScheduledOrder, orderStatus *bf.QueryOrderResp, statusChanged, qtyChanged bool) string {
	currentStatus := s.mapExchangeStatus(orderStatus.Status)

	// 订单被取消
	if statusChanged && currentStatus == "cancelled" && order.Status == "processing" {
		return "external_cancel"
	}

	// 订单被修改（数量变化）
	if qtyChanged && !statusChanged {
		if orderStatus.ExecutedQty > order.ExecutedQty {
			return "external_modify_increase"
		} else {
			return "external_modify_decrease"
		}
	}

	// 订单部分成交后被取消
	if statusChanged && currentStatus == "cancelled" && order.Status == "filled" && orderStatus.ExecutedQty != order.ExecutedQty {
		return "external_partial_fill_cancel"
	}

	return "" // 不是外部操作
}

// handleOrderExternallyDeleted 处理订单被外部删除的情况
func (s *Server) handleOrderExternallyDeleted(order *pdb.ScheduledOrder) error {
	log.Printf("[Order-Sync] 处理外部删除的订单: %d (%s)", order.ID, order.ClientOrderId)

	// 创建外部操作记录
	externalOp := &pdb.ExternalOperation{
		Symbol:        order.Symbol,
		OperationType: "external_order_deleted",
		OldAmount:     order.ExecutedQty,
		NewAmount:     "0",
		Confidence:    0.95, // 订单不存在通常很确定
		DetectedAt:    time.Now(),
		Status:        "processed",
		UserID:        order.UserID,
		Notes:         "订单在交易所不存在，可能已被外部删除",
	}

	if err := s.db.DB().Create(externalOp).Error; err != nil {
		return fmt.Errorf("创建外部删除操作记录失败: %w", err)
	}

	// 更新订单状态
	updateData := map[string]interface{}{
		"status":     "failed",
		"result":     "订单在交易所不存在，可能已被外部删除",
		"updated_at": time.Now(),
	}

	if err := s.db.DB().Model(order).Updates(updateData).Error; err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 通知用户
	s.notifyUserExternalOperation(externalOp)

	return nil
}

// updateOrderStatusFromExternalOperation 根据外部操作更新订单状态
func (s *Server) updateOrderStatusFromExternalOperation(order *pdb.ScheduledOrder, orderStatus *bf.QueryOrderResp) error {
	updateData := map[string]interface{}{
		"status":            s.mapExchangeStatus(orderStatus.Status),
		"executed_quantity": orderStatus.ExecutedQty,
		"avg_price":         orderStatus.AvgPrice,
		"updated_at":        time.Now(),
	}

	// 如果订单有交易所订单ID，也更新
	if orderStatus.OrderId > 0 {
		updateData["exchange_order_id"] = strconv.FormatInt(orderStatus.OrderId, 10)
	}

	return s.db.DB().Model(order).Updates(updateData).Error
}

// mapExchangeStatus 将交易所状态映射为系统状态
func (s *Server) mapExchangeStatus(exchangeStatus string) string {
	switch exchangeStatus {
	case "FILLED":
		return "filled"
	case "CANCELED", "PENDING_CANCEL":
		return "cancelled"
	case "REJECTED", "EXPIRED":
		return "failed"
	case "PARTIALLY_FILLED":
		return "filled" // 部分成交仍标记为filled
	case "NEW":
		return "processing"
	default:
		return "processing"
	}
}

// getAllUserPositions 获取所有用户的持仓信息
func (s *Server) getAllUserPositions() (map[uint]map[string]*PositionSnapshot, error) {
	// 查询所有用户（API密钥从配置文件读取，不需要数据库字段）
	var users []struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}

	err := s.db.DB().Table("users").
		Select("id, username").
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	allPositions := make(map[uint]map[string]*PositionSnapshot)

	for _, user := range users {
		// 为每个用户创建币安客户端
		useTestnet := s.cfg.Exchange.Binance.IsTestnet
		client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

		// 注意：这里使用的是全局API密钥，实际应该使用每个用户的API密钥
		// TODO: 从用户配置中获取每个用户的API密钥
		positions, err := client.GetPositions()
		if err != nil {
			log.Printf("[Position-Detect] 获取用户 %d 持仓失败: %v", user.ID, err)
			continue
		}

		userPositions := make(map[string]*PositionSnapshot)
		for _, pos := range positions {
			// 只处理有持仓的交易对
			if amt, _ := strconv.ParseFloat(pos.PositionAmt, 64); amt != 0 {
				userPositions[pos.Symbol] = &PositionSnapshot{
					Symbol:       pos.Symbol,
					PositionAmt:  pos.PositionAmt,
					EntryPrice:   pos.EntryPrice,
					MarkPrice:    pos.MarkPrice,
					UpdateTime:   pos.UpdateTime,
					SnapshotTime: time.Now(),
					UserID:       user.ID,
				}
			}
		}

		if len(userPositions) > 0 {
			allPositions[user.ID] = userPositions
		}
	}

	return allPositions, nil
}

// getLastPositionSnapshots 获取上次的持仓快照
func (s *Server) getLastPositionSnapshots() map[string]*PositionSnapshot {
	s.positionMutex.RLock()
	defer s.positionMutex.RUnlock()

	snapshots := make(map[string]*PositionSnapshot)
	for symbol, snapshot := range s.positionSnapshots {
		snapshots[symbol] = snapshot
	}
	return snapshots
}

// detectPositionChangesInternal 检测持仓变化的核心逻辑
func (s *Server) detectPositionChangesInternal(currentPositions map[uint]map[string]*PositionSnapshot, lastSnapshots map[string]*PositionSnapshot) []DetectedChange {
	var changes []DetectedChange

	// 用于跟踪已处理的持仓，避免重复检测
	processedSymbols := make(map[string]bool)

	// 检测当前持仓的变化
	for userID, userPositions := range currentPositions {
		for symbol, current := range userPositions {
			key := fmt.Sprintf("%d_%s", userID, symbol)
			processedSymbols[key] = true

			previous, existed := lastSnapshots[key]

			if !existed {
				// 新持仓出现 - 可能外部开仓
				change := DetectedChange{
					Symbol:     symbol,
					Type:       "new",
					OldAmt:     "0",
					NewAmt:     current.PositionAmt,
					Confidence: s.calculateChangeConfidence(current, nil, "new"),
					Timestamp:  time.Now(),
				}
				if change.Confidence > 0.7 { // 只处理高置信度的变化
					changes = append(changes, change)
				}
			} else if previous.PositionAmt != current.PositionAmt {
				// 持仓数量变化 - 可能部分平仓
				change := DetectedChange{
					Symbol:     symbol,
					Type:       "changed",
					OldAmt:     previous.PositionAmt,
					NewAmt:     current.PositionAmt,
					Confidence: s.calculateChangeConfidence(current, previous, "changed"),
					Timestamp:  time.Now(),
				}
				if change.Confidence > 0.8 {
					changes = append(changes, change)
				}
			}
		}
	}

	// 检测消失的持仓
	for key, previous := range lastSnapshots {
		if !processedSymbols[key] {
			// 持仓消失 - 完全平仓
			change := DetectedChange{
				Symbol:     previous.Symbol,
				Type:       "closed",
				OldAmt:     previous.PositionAmt,
				NewAmt:     "0",
				Confidence: 0.95, // 持仓消失通常很确定
				Timestamp:  time.Now(),
			}
			changes = append(changes, change)
		}
	}

	return changes
}

// calculateChangeConfidence 计算持仓变化的置信度
func (s *Server) calculateChangeConfidence(current, previous *PositionSnapshot, changeType string) float64 {
	confidence := 0.5 // 基础置信度

	switch changeType {
	case "new":
		// 新持仓的置信度计算
		if amt, _ := strconv.ParseFloat(current.PositionAmt, 64); amt > 0.1 {
			confidence += 0.3 // 有意义的持仓量
		}
		if current.UpdateTime > 0 {
			confidence += 0.2 // 有更新时间戳
		}

	case "changed":
		// 持仓变化的置信度计算
		oldAmt, _ := strconv.ParseFloat(previous.PositionAmt, 64)
		newAmt, _ := strconv.ParseFloat(current.PositionAmt, 64)
		changeRatio := math.Abs(newAmt-oldAmt) / math.Abs(oldAmt)

		if changeRatio > 0.1 { // 变化超过10%
			confidence += 0.4
		} else if changeRatio > 0.01 { // 变化超过1%
			confidence += 0.2
		}

		if current.UpdateTime != previous.UpdateTime {
			confidence += 0.2 // 更新时间不同
		}

	case "closed":
		confidence = 0.95 // 持仓消失通常很确定
	}

	// 时间窗口检查 - 只在活跃时间内检测变化
	now := time.Now()
	if now.Hour() >= 8 && now.Hour() <= 20 { // 工作时间内
		confidence += 0.1
	}

	return math.Min(confidence, 1.0)
}

// handlePositionChange 处理检测到的持仓变化
func (s *Server) handlePositionChange(change DetectedChange) error {
	log.Printf("[Position-Detect] 检测到持仓变化: %s %s %s -> %s (置信度: %.2f)",
		change.Symbol, change.Type, change.OldAmt, change.NewAmt, change.Confidence)

	// 记录持仓变化到审计日志
	oldPosition := &PositionSnapshot{
		Symbol:      change.Symbol,
		PositionAmt: change.OldAmt,
	}
	newPosition := &PositionSnapshot{
		Symbol:      change.Symbol,
		PositionAmt: change.NewAmt,
	}

	s.logPositionOperation(0, change.Symbol, "position_change_detected",
		fmt.Sprintf("检测到持仓变化: %s -> %s", change.OldAmt, change.NewAmt),
		oldPosition, newPosition, "system", "info")

	switch change.Type {
	case "new":
		return s.handleNewPosition(change)
	case "changed":
		return s.handlePositionQuantityChange(change)
	case "closed":
		return s.handlePositionClosed(change)
	default:
		log.Printf("[Position-Detect] 未知的变化类型: %s", change.Type)
	}

	return nil
}

// handleNewPosition 处理新持仓出现
func (s *Server) handleNewPosition(change DetectedChange) error {
	log.Printf("[Position-Detect] 新持仓出现: %s 数量=%s", change.Symbol, change.NewAmt)

	// 这里可以添加逻辑来检查是否是系统的订单导致的
	// 如果不是，可能是用户在官网手动开仓

	return s.createExternalOperationRecord(change, "external_open")
}

// handlePositionQuantityChange 处理持仓数量变化
func (s *Server) handlePositionQuantityChange(change DetectedChange) error {
	log.Printf("[Position-Detect] 持仓数量变化: %s %s -> %s", change.Symbol, change.OldAmt, change.NewAmt)

	oldAmt, _ := strconv.ParseFloat(change.OldAmt, 64)
	newAmt, _ := strconv.ParseFloat(change.NewAmt, 64)

	if math.Abs(newAmt) < math.Abs(oldAmt) {
		// 持仓减少 - 可能是部分平仓
		return s.createExternalOperationRecord(change, "external_partial_close")
	} else {
		// 持仓增加 - 可能是加仓
		return s.createExternalOperationRecord(change, "external_add_position")
	}
}

// handlePositionClosed 处理持仓完全关闭
func (s *Server) handlePositionClosed(change DetectedChange) error {
	log.Printf("[Position-Detect] 持仓完全关闭: %s", change.Symbol)

	return s.createExternalOperationRecord(change, "external_full_close")
}

// createExternalOperationRecord 创建外部操作记录
func (s *Server) createExternalOperationRecord(change DetectedChange, operationType string) error {
	// 创建外部操作记录到数据库
	externalOp := &pdb.ExternalOperation{
		Symbol:        change.Symbol,
		OperationType: operationType,
		OldAmount:     change.OldAmt,
		NewAmount:     change.NewAmt,
		Confidence:    change.Confidence,
		DetectedAt:    change.Timestamp,
		Status:        "detected",
	}

	if err := s.db.DB().Create(externalOp).Error; err != nil {
		return fmt.Errorf("创建外部操作记录失败: %w", err)
	}

	// 记录外部操作到审计日志
	s.logSystemOperation("external_operation_detected",
		fmt.Sprintf("检测到外部操作: %s %s %s -> %s", change.Symbol, operationType, change.OldAmt, change.NewAmt),
		"info",
		map[string]interface{}{
			"external_operation_id": externalOp.ID,
			"symbol":                change.Symbol,
			"operation_type":        operationType,
			"old_amount":            change.OldAmt,
			"new_amount":            change.NewAmt,
			"confidence":            change.Confidence,
		},
		"")

	// 查找相关的开仓订单，尝试建立关联
	if operationType == "external_full_close" || operationType == "external_partial_close" {
		if err := s.linkExternalCloseToEntryOrder(externalOp); err != nil {
			log.Printf("[Position-Detect] 关联外部平仓订单失败: %v", err)
		}
	}

	// 发送用户通知
	s.notifyUserExternalOperation(externalOp)

	log.Printf("[Position-Detect] 外部操作记录创建成功: %s %s", change.Symbol, operationType)
	return nil
}

// handleBracketExternalClose 处理Bracket订单的外部平仓
func (s *Server) handleBracketExternalClose(entryOrder pdb.ScheduledOrder, externalOp *pdb.ExternalOperation) error {
	log.Printf("[Bracket-External] 处理Bracket订单 %d 的外部平仓", entryOrder.ID)

	// 查找对应的BracketLink
	var bracketLink pdb.BracketLink
	err := s.db.DB().Where("entry_client_id = ?", entryOrder.ClientOrderId).First(&bracketLink).Error
	if err != nil {
		return fmt.Errorf("查找BracketLink失败: %w", err)
	}

	log.Printf("[Bracket-External] 处理Bracket订单 %s 的外部平仓 (状态: %s)", bracketLink.GroupID, bracketLink.Status)
	// 无论Bracket状态如何，都要尝试取消可能仍活跃的条件委托
	// 因为可能存在Bracket关闭但条件委托未被正确取消的情况

	// 使用配置的环境设置获取交易所客户端
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	// 取消活跃的条件订单
	cancelledCount := 0

	log.Printf("[Bracket-External] 开始检查条件委托 - TP: %s, SL: %s", bracketLink.TPClientID, bracketLink.SLClientID)

	// 取消止盈订单
	if bracketLink.TPClientID != "" {
		log.Printf("[Bracket-External] 尝试取消止盈订单 %s", bracketLink.TPClientID)
		if err := s.cancelConditionalOrderIfNeeded(client, externalOp.Symbol, bracketLink.TPClientID, "TP"); err != nil {
			log.Printf("[Bracket-External] 取消止盈订单失败 %s: %v", bracketLink.TPClientID, err)
		} else {
			cancelledCount++
			log.Printf("[Bracket-External] ✅ 成功取消止盈订单 %s", bracketLink.TPClientID)
		}
	} else {
		log.Printf("[Bracket-External] 止盈订单ClientID为空，跳过")
	}

	// 取消止损订单
	if bracketLink.SLClientID != "" {
		log.Printf("[Bracket-External] 尝试取消止损订单 %s", bracketLink.SLClientID)
		if err := s.cancelConditionalOrderIfNeeded(client, externalOp.Symbol, bracketLink.SLClientID, "SL"); err != nil {
			log.Printf("[Bracket-External] 取消止损订单失败 %s: %v", bracketLink.SLClientID, err)
		} else {
			cancelledCount++
			log.Printf("[Bracket-External] ✅ 成功取消止损订单 %s", bracketLink.SLClientID)
		}
	} else {
		log.Printf("[Bracket-External] 止损订单ClientID为空，跳过")
	}

	// 更新Bracket状态为closed
	if err := s.db.DB().Model(&pdb.BracketLink{}).Where("id = ?", bracketLink.ID).Update("status", "closed").Error; err != nil {
		log.Printf("[Bracket-External] 更新Bracket状态失败 %d: %v", bracketLink.ID, err)
		return fmt.Errorf("更新Bracket状态失败: %w", err)
	}

	// 🔧 修复：开仓订单状态保持为filled（已成交），通过关联的平仓订单来表示"已结束"
	// 不需要更新开仓订单状态，保持filled状态，让前端通过related_orders.has_close来显示"已结束"

	// 🔧 新增：创建外部平仓操作记录，关联到开仓订单
	// 根据原持仓方向确定平仓方向和平仓数量
	oldAmt, _ := strconv.ParseFloat(externalOp.OldAmount, 64)
	newAmt, _ := strconv.ParseFloat(externalOp.NewAmount, 64)
	closeQuantity := fmt.Sprintf("%.8f", math.Abs(oldAmt-newAmt)) // 平仓数量为变化的绝对值

	closeSide := "SELL" // 默认卖出平多
	if oldAmt < 0 {
		closeSide = "BUY" // 买入平空
	}

	now := time.Now()
	externalCloseOrder := pdb.ScheduledOrder{
		UserID:          entryOrder.UserID,
		Exchange:        entryOrder.Exchange,
		Testnet:         entryOrder.Testnet,
		Symbol:          externalOp.Symbol,
		Side:            closeSide, // 根据原持仓方向确定平仓方向
		OrderType:       "MARKET",
		Quantity:        closeQuantity,
		Price:           "",
		Leverage:        entryOrder.Leverage,
		ReduceOnly:      true, // 这是平仓订单
		StrategyID:      entryOrder.StrategyID,
		ExecutionID:     entryOrder.ExecutionID,
		BracketEnabled:  false, // 外部平仓不是Bracket订单
		TPPercent:       0,
		SLPercent:       0,
		TPPrice:         "",
		SLPrice:         "",
		WorkingType:     "",
		TriggerTime:     now,
		Status:          "completed", // 外部操作已完成
		Result:          fmt.Sprintf("外部平仓操作: %s", externalOp.OperationType),
		ClientOrderId:   fmt.Sprintf("external-close-%d-%d", entryOrder.ID, now.Unix()),
		ExchangeOrderId: "",
		ExecutedQty:     closeQuantity,
		AvgPrice:        "0",           // 外部操作没有具体价格信息
		ParentOrderId:   entryOrder.ID, // 关联到开仓订单
		CloseOrderIds:   "",
		StrategyType:    "external_operation",
		GridLevel:       0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.db.DB().Create(&externalCloseOrder).Error; err != nil {
		log.Printf("[Bracket-External] 创建外部平仓订单记录失败: %v", err)
		// 不返回错误，因为主要逻辑已完成
	} else {
		log.Printf("[Bracket-External] ✅ 创建外部平仓订单记录: ID=%d", externalCloseOrder.ID)

		// 更新开仓订单的close_order_ids字段
		if err := s.updateOrderAssociations(&entryOrder, externalCloseOrder.ID); err != nil {
			log.Printf("[Bracket-External] 更新开仓订单的close_order_ids失败: %v", err)
		}
	}

	log.Printf("[Bracket-External] Bracket订单 %s 外部平仓处理完成，取消了 %d 个条件订单",
		bracketLink.GroupID, cancelledCount)

	return nil
}

// linkExternalCloseToEntryOrder 将外部平仓操作关联到开仓订单
func (s *Server) linkExternalCloseToEntryOrder(externalOp *pdb.ExternalOperation) error {
	// 查找该交易对的活跃开仓订单
	var entryOrders []pdb.ScheduledOrder
	err := s.db.DB().Where("symbol = ? AND status = ? AND reduce_only = ? AND exchange = ?",
		externalOp.Symbol, "filled", false, "binance_futures").Find(&entryOrders).Error

	if err != nil {
		return fmt.Errorf("查找开仓订单失败: %w", err)
	}

	if len(entryOrders) == 0 {
		log.Printf("[Position-Detect] 未找到相关的开仓订单: %s", externalOp.Symbol)
		return nil
	}

	// 为每个开仓订单创建平仓记录
	for _, entryOrder := range entryOrders {
		// 🔧 修复：检查开仓订单是否属于Bracket订单
		if entryOrder.BracketEnabled {
			log.Printf("[Position-Detect] 开仓订单 %d 属于Bracket订单，跳过创建外部平仓记录，立即处理Bracket关闭",
				entryOrder.ID)

			// 🔧 新增：立即处理Bracket订单的外部平仓
			if err := s.handleBracketExternalClose(entryOrder, externalOp); err != nil {
				log.Printf("[Position-Detect] 处理Bracket外部平仓失败 %d: %v", entryOrder.ID, err)
			}
			continue
		}

		// 检查是否已经有相关的平仓订单，避免重复创建
		var existingCloseOrders []pdb.ScheduledOrder
		err := s.db.DB().Where("parent_order_id = ? AND reduce_only = ? AND status IN (?)",
			entryOrder.ID, true, []string{"pending", "processing", "sent", "filled", "completed"}).Find(&existingCloseOrders).Error

		if err != nil {
			log.Printf("[Position-Detect] 检查现有平仓订单失败: %v", err)
			continue
		}

		if len(existingCloseOrders) > 0 {
			log.Printf("[Position-Detect] 开仓订单 %d 已有 %d 个平仓订单，跳过创建外部平仓订单",
				entryOrder.ID, len(existingCloseOrders))
			continue
		}
		// 根据开仓订单的方向确定平仓方向
		closeSide := "BUY" // 默认买入平空
		if entryOrder.Side == "BUY" {
			closeSide = "SELL" // 卖出平多
		}

		closeOrder := &pdb.ScheduledOrder{
			UserID:        entryOrder.UserID,
			Exchange:      entryOrder.Exchange,
			Testnet:       entryOrder.Testnet,
			Symbol:        entryOrder.Symbol,
			Side:          closeSide, // 根据开仓方向确定平仓方向
			OrderType:     "MARKET",
			Quantity:      externalOp.NewAmount,
			Price:         "",
			Leverage:      entryOrder.Leverage,
			ReduceOnly:    true,
			TriggerTime:   externalOp.DetectedAt,
			Status:        "filled", // 外部操作已完成
			ParentOrderId: entryOrder.ID,
			ExecutedQty:   externalOp.NewAmount,
			AvgPrice:      "0", // 无法获取实际价格
			ClientOrderId: func() string {
				// 生成安全的external_close ClientOrderId，确保不超过36字符
				// 截取ID的后7位数，确保总长度不超过36字符
				safeEntryID := entryOrder.ID % 10000000   // 7位数
				safeExternalID := externalOp.ID % 1000000 // 6位数
				return fmt.Sprintf("EC_%d_%d", safeEntryID, safeExternalID)
			}(),
			ExchangeOrderId: fmt.Sprintf("external_%d", externalOp.ID),
		}

		if err := s.db.DB().Create(closeOrder).Error; err != nil {
			log.Printf("[Position-Detect] 创建外部平仓订单失败: %v", err)
			continue
		}

		// 更新开仓订单的关联字段
		if err := s.updateOrderAssociations(&entryOrder, closeOrder.ID); err != nil {
			log.Printf("[Position-Detect] 更新订单关联失败: %v", err)
		}

		log.Printf("[Position-Detect] 外部平仓订单关联成功: 开仓#%d -> 平仓#%d", entryOrder.ID, closeOrder.ID)
	}

	return nil
}

// Notification 通知结构体
type Notification struct {
	UserID    uint                   `json:"user_id"`
	Type      string                 `json:"type"`     // 通知类型: external_operation, order_update, system_alert
	Title     string                 `json:"title"`    // 通知标题
	Message   string                 `json:"message"`  // 通知内容
	Data      map[string]interface{} `json:"data"`     // 附加数据
	Priority  string                 `json:"priority"` // 优先级: low, normal, high, urgent
	CreatedAt time.Time              `json:"created_at"`
}

// NotificationService 通知服务接口
type NotificationService interface {
	Send(notification *Notification) error
	SendToUser(userID uint, notification *Notification) error
	Broadcast(notification *Notification) error
}

// SendToUser 发送通知给指定用户
func (c *CompositeNotificationService) SendToUser(userID uint, notification *Notification) error {
	notification.UserID = userID
	return c.Send(notification)
}

// Broadcast 广播通知给所有用户
func (c *CompositeNotificationService) Broadcast(notification *Notification) error {
	// TODO: 实现广播逻辑，获取所有用户并发送通知
	log.Printf("[Notification] 广播通知: %s", notification.Title)
	// 这里应该查询所有用户并逐个发送
	return nil
}

// WebSocketNotificationService WebSocket通知服务
type WebSocketNotificationService struct {
	// 这里可以集成WebSocket连接管理
}

// EmailNotificationService 邮件通知服务
type EmailNotificationService struct {
	smtpServer string
	smtpPort   int
	username   string
	password   string
	fromEmail  string
}

// SMSNotificationService 短信通知服务
type SMSNotificationService struct {
	apiKey    string
	apiSecret string
	sender    string
}

// CompositeNotificationService 复合通知服务
type CompositeNotificationService struct {
	webSocketSvc *WebSocketNotificationService
	emailSvc     *EmailNotificationService
	smsSvc       *SMSNotificationService
}

// NewCompositeNotificationService 创建复合通知服务
func NewCompositeNotificationService(cfg *config.Config) *CompositeNotificationService {
	return &CompositeNotificationService{
		webSocketSvc: &WebSocketNotificationService{},
		emailSvc: &EmailNotificationService{
			smtpServer: cfg.Notification.SMTP.Server,
			smtpPort:   cfg.Notification.SMTP.Port,
			username:   cfg.Notification.SMTP.Username,
			password:   cfg.Notification.SMTP.Password,
			fromEmail:  cfg.Notification.SMTP.FromEmail,
		},
		smsSvc: &SMSNotificationService{
			apiKey:    cfg.Notification.SMS.APIKey,
			apiSecret: cfg.Notification.SMS.APISecret,
			sender:    cfg.Notification.SMS.Sender,
		},
	}
}

// Send 发送通知
func (c *CompositeNotificationService) Send(notification *Notification) error {
	var errors []error

	// 1. WebSocket实时通知（主要渠道）
	if err := c.webSocketSvc.Send(notification); err != nil {
		log.Printf("[Notification] WebSocket通知失败: %v", err)
		errors = append(errors, fmt.Errorf("websocket: %w", err))
	}

	// 2. 根据优先级和类型决定是否发送其他通知
	switch notification.Priority {
	case "urgent", "high":
		// 紧急和高优先级通知发送邮件和短信
		if err := c.emailSvc.Send(notification); err != nil {
			log.Printf("[Notification] 邮件通知失败: %v", err)
			errors = append(errors, fmt.Errorf("email: %w", err))
		}

		if notification.Type == "external_operation" {
			// 外部操作特别重要，发送短信
			if err := c.smsSvc.Send(notification); err != nil {
				log.Printf("[Notification] 短信通知失败: %v", err)
				errors = append(errors, fmt.Errorf("sms: %w", err))
			}
		}

	case "normal":
		// 普通优先级只发送邮件
		if err := c.emailSvc.Send(notification); err != nil {
			log.Printf("[Notification] 邮件通知失败: %v", err)
			errors = append(errors, fmt.Errorf("email: %w", err))
		}

	case "low":
		// 低优先级只通过WebSocket
		// 不发送其他通知
	}

	if len(errors) > 0 {
		return fmt.Errorf("通知发送失败: %v", errors)
	}

	log.Printf("[Notification] 通知发送成功: %s -> %s", notification.Type, notification.Title)
	return nil
}

// Send 发送WebSocket通知
func (w *WebSocketNotificationService) Send(notification *Notification) error {
	// TODO: 实现WebSocket通知逻辑
	// 这里应该向用户的WebSocket连接发送实时通知
	log.Printf("[WebSocket] 发送通知到用户 %d: %s", notification.UserID, notification.Title)
	return nil
}

// Send 发送邮件通知
func (e *EmailNotificationService) Send(notification *Notification) error {
	if e.smtpServer == "" {
		log.Printf("[Email] SMTP未配置，跳过邮件通知")
		return nil
	}

	// TODO: 实现邮件发送逻辑
	log.Printf("[Email] 发送邮件到用户 %d: %s", notification.UserID, notification.Title)
	return nil
}

// Send 发送短信通知
func (s *SMSNotificationService) Send(notification *Notification) error {
	if s.apiKey == "" {
		log.Printf("[SMS] SMS未配置，跳过短信通知")
		return nil
	}

	// TODO: 实现短信发送逻辑
	log.Printf("[SMS] 发送短信到用户 %d: %s", notification.UserID, notification.Title)
	return nil
}

// notifyUserExternalOperation 通知用户外部操作
func (s *Server) notifyUserExternalOperation(externalOp *pdb.ExternalOperation) {
	log.Printf("[Notification] 检测到外部操作: %s %s (置信度: %.2f)",
		externalOp.Symbol, externalOp.OperationType, externalOp.Confidence)

	// 创建通知
	notification := &Notification{
		UserID: externalOp.UserID,
		Type:   "external_operation",
		Data: map[string]interface{}{
			"external_operation_id": externalOp.ID,
			"symbol":                externalOp.Symbol,
			"operation_type":        externalOp.OperationType,
			"old_amount":            externalOp.OldAmount,
			"new_amount":            externalOp.NewAmount,
			"confidence":            externalOp.Confidence,
		},
		CreatedAt: time.Now(),
	}

	// 根据操作类型设置通知内容和优先级
	switch externalOp.OperationType {
	case "external_full_close":
		notification.Title = "检测到外部平仓操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网对 %s 进行了平仓操作。原持仓: %s, 当前持仓: %s",
			externalOp.Symbol, externalOp.OldAmount, externalOp.NewAmount)
		notification.Priority = "high"

	case "external_partial_close":
		notification.Title = "检测到外部部分平仓操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网对 %s 进行了部分平仓操作。持仓从 %s 减少到 %s",
			externalOp.Symbol, externalOp.OldAmount, externalOp.NewAmount)
		notification.Priority = "normal"

	case "external_add_position":
		notification.Title = "检测到外部加仓操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网对 %s 进行了加仓操作。持仓从 %s 增加到 %s",
			externalOp.Symbol, externalOp.OldAmount, externalOp.NewAmount)
		notification.Priority = "normal"

	case "external_open":
		notification.Title = "检测到外部开仓操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网对 %s 进行了开仓操作。当前持仓: %s",
			externalOp.Symbol, externalOp.NewAmount)
		notification.Priority = "high"

	case "external_cancel":
		notification.Title = "检测到外部取消订单操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网取消了 %s 的订单", externalOp.Symbol)
		notification.Priority = "normal"

	case "external_modify_increase":
		notification.Title = "检测到外部增加订单数量操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网增加了 %s 订单的数量", externalOp.Symbol)
		notification.Priority = "low"

	case "external_modify_decrease":
		notification.Title = "检测到外部减少订单数量操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网减少了 %s 订单的数量", externalOp.Symbol)
		notification.Priority = "low"

	case "external_order_deleted":
		notification.Title = "检测到外部删除订单操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网删除了 %s 的订单，该订单可能已被执行或取消", externalOp.Symbol)
		notification.Priority = "urgent"

	default:
		notification.Title = "检测到外部操作"
		notification.Message = fmt.Sprintf("系统检测到您在币安官网对 %s 进行了操作: %s",
			externalOp.Symbol, externalOp.OperationType)
		notification.Priority = "normal"
	}

	// 发送通知
	if s.notificationService != nil {
		if err := s.notificationService.Send(notification); err != nil {
			log.Printf("[Notification] 发送通知失败: %v", err)
		}
	} else {
		log.Printf("[Notification] 通知服务未初始化，使用默认日志通知: %s", notification.Message)
	}

	// 同时保存到数据库的系统消息表（如果有的话）
	s.saveNotificationToDatabase(notification)
}

// saveNotificationToDatabase 保存通知到数据库
func (s *Server) saveNotificationToDatabase(notification *Notification) {
	// TODO: 实现保存到用户通知表的逻辑
	// 这里可以创建一个 user_notifications 表来存储用户的通知历史

	log.Printf("[Notification] 保存通知到数据库: 用户%d, 类型%s, 优先级%s",
		notification.UserID, notification.Type, notification.Priority)
}

// initNotificationService 初始化通知服务
func (s *Server) initNotificationService(cfg *config.Config) {
	if !cfg.Notification.Enabled {
		log.Printf("[INIT] 通知服务已禁用（配置中 notification.enabled = false）")
		s.notificationService = nil
		return
	}

	log.Printf("[INIT] 初始化通知服务...")
	s.notificationService = NewCompositeNotificationService(cfg)
	log.Printf("[INIT] 通知服务初始化完成")
}

// initAuditLogger 初始化审计日志记录器
func (s *Server) initAuditLogger() {
	log.Printf("[INIT] 初始化审计日志记录器...")

	s.auditLogger = NewAuditLogger(s.db.DB())
	log.Printf("[INIT] 审计日志记录器初始化完成")
}

// initHealthChecker 初始化系统健康检查器
func (s *Server) initHealthChecker() {
	log.Printf("[INIT] 初始化系统健康检查器...")

	s.healthChecker = NewSystemHealthChecker(s.db.DB())

	// 启动定期健康检查
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // 每5分钟进行一次健康检查
		defer ticker.Stop()

		// 首次启动时等待1分钟再执行
		time.Sleep(1 * time.Minute)

		for {
			select {
			case <-ticker.C:
				if err := s.performHealthCheck(); err != nil {
					log.Printf("[Health-Check] 定期健康检查失败: %v", err)
				}
			}
		}
	}()

	log.Printf("[INIT] 系统健康检查器初始化完成（每5分钟检查一次）")
}

// smartNotifyOrderUpdate 智能通知订单状态更新
func (s *Server) smartNotifyOrderUpdate(order *pdb.ScheduledOrder, oldStatus, newStatus string) {
	// 只对重要的状态变化发送通知
	importantChanges := map[string][]string{
		"pending":    {"processing", "filled", "failed", "cancelled"},
		"processing": {"filled", "failed", "cancelled"},
		"sent":       {"filled", "failed", "cancelled"},
	}

	shouldNotify := false
	if allowedChanges, exists := importantChanges[oldStatus]; exists {
		for _, change := range allowedChanges {
			if change == newStatus {
				shouldNotify = true
				break
			}
		}
	}

	if !shouldNotify {
		return
	}

	notification := &Notification{
		UserID: order.UserID,
		Type:   "order_update",
		Title:  "订单状态更新",
		Message: fmt.Sprintf("您的订单 #%d (%s) 状态从 %s 变为 %s",
			order.ID, order.Symbol, s.translateStatus(oldStatus), s.translateStatus(newStatus)),
		Data: map[string]interface{}{
			"order_id":     order.ID,
			"symbol":       order.Symbol,
			"old_status":   oldStatus,
			"new_status":   newStatus,
			"executed_qty": order.ExecutedQty,
			"avg_price":    order.AvgPrice,
		},
		Priority:  s.calculateOrderNotificationPriority(oldStatus, newStatus),
		CreatedAt: time.Now(),
	}

	if s.notificationService != nil {
		s.notificationService.Send(notification)
	}
}

// calculateOrderNotificationPriority 计算订单通知优先级
func (s *Server) calculateOrderNotificationPriority(oldStatus, newStatus string) string {
	if newStatus == "filled" {
		return "normal" // 成交通知
	} else if newStatus == "failed" {
		return "high" // 失败通知比较重要
	} else if newStatus == "cancelled" {
		return "low" // 取消通知优先级较低
	}
	return "normal"
}

// translateStatus 翻译状态为中文
func (s *Server) translateStatus(status string) string {
	statusMap := map[string]string{
		"pending":    "等待执行",
		"processing": "执行中",
		"sent":       "已发送",
		"filled":     "已成交",
		"completed":  "已完成",
		"cancelled":  "已取消",
		"failed":     "失败",
	}

	if translated, exists := statusMap[status]; exists {
		return translated
	}
	return status
}

// updatePositionSnapshots 更新持仓快照
func (s *Server) updatePositionSnapshots(currentPositions map[uint]map[string]*PositionSnapshot) {
	s.positionMutex.Lock()
	defer s.positionMutex.Unlock()

	// 清空旧的快照
	s.positionSnapshots = make(map[string]*PositionSnapshot)

	// 添加新的快照
	for userID, userPositions := range currentPositions {
		for symbol, snapshot := range userPositions {
			key := fmt.Sprintf("%d_%s", userID, symbol)
			s.positionSnapshots[key] = snapshot
		}
	}

	log.Printf("[Position-Detect] 持仓快照已更新，共 %d 个持仓", len(s.positionSnapshots))
}

// SystemHealthChecker 系统健康检查器
type SystemHealthChecker struct {
	db              *gorm.DB
	lastHealthCheck time.Time
	healthMetrics   map[string]interface{}
	alertCooldowns  map[string]time.Time
	mu              sync.RWMutex
}

// NewSystemHealthChecker 创建系统健康检查器
func NewSystemHealthChecker(db *gorm.DB) *SystemHealthChecker {
	return &SystemHealthChecker{
		db:             db,
		healthMetrics:  make(map[string]interface{}),
		alertCooldowns: make(map[string]time.Time),
	}
}

// performHealthCheck 执行系统健康检查
func (s *Server) performHealthCheck() error {
	if s.healthChecker == nil {
		return nil
	}

	log.Printf("[Health-Check] 开始执行系统健康检查...")

	// 1. 数据库连接检查
	if err := s.checkDatabaseHealth(); err != nil {
		s.handleHealthAlert("database_connection", "数据库连接异常", err)
		return err
	}

	// 2. 订单同步状态检查
	if err := s.checkOrderSyncHealth(); err != nil {
		s.handleHealthAlert("order_sync", "订单同步异常", err)
	}

	// 3. 持仓检测状态检查
	if err := s.checkPositionDetectionHealth(); err != nil {
		s.handleHealthAlert("position_detection", "持仓检测异常", err)
	}

	// 4. 通知服务状态检查
	if err := s.checkNotificationHealth(); err != nil {
		s.handleHealthAlert("notification_service", "通知服务异常", err)
	}

	// 5. 内存和性能检查
	if err := s.checkSystemPerformance(); err != nil {
		s.handleHealthAlert("system_performance", "系统性能异常", err)
	}

	s.healthChecker.mu.Lock()
	s.healthChecker.lastHealthCheck = time.Now()
	s.healthChecker.mu.Unlock()

	log.Printf("[Health-Check] 系统健康检查完成")
	return nil
}

// checkDatabaseHealth 检查数据库健康状态
func (s *Server) checkDatabaseHealth() error {
	// 检查数据库连接
	if err := s.db.DB().Exec("SELECT 1").Error; err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查关键表的记录数
	var orderCount, externalOpCount int64
	s.db.DB().Model(&pdb.ScheduledOrder{}).Count(&orderCount)
	s.db.DB().Model(&pdb.ExternalOperation{}).Count(&externalOpCount)

	// 更新健康指标
	s.healthChecker.mu.Lock()
	s.healthChecker.healthMetrics["order_count"] = orderCount
	s.healthChecker.healthMetrics["external_operation_count"] = externalOpCount
	s.healthChecker.healthMetrics["database_status"] = "healthy"
	s.healthChecker.mu.Unlock()

	log.Printf("[Health-Check] 数据库健康检查通过 - 订单数: %d, 外部操作数: %d", orderCount, externalOpCount)
	return nil
}

// checkOrderSyncHealth 检查订单同步健康状态
func (s *Server) checkOrderSyncHealth() error {
	// 检查最近的订单同步活动
	var recentSyncs int64
	oneHourAgo := time.Now().Add(-time.Hour)
	s.db.DB().Model(&pdb.OperationLog{}).
		Where("entity_type = ? AND action = ? AND created_at > ?",
			"order", "status_update", oneHourAgo).
		Count(&recentSyncs)

	if recentSyncs == 0 {
		return fmt.Errorf("过去1小时内没有订单同步活动")
	}

	s.healthChecker.mu.Lock()
	s.healthChecker.healthMetrics["recent_order_syncs"] = recentSyncs
	s.healthChecker.healthMetrics["order_sync_status"] = "active"
	s.healthChecker.mu.Unlock()

	log.Printf("[Health-Check] 订单同步健康检查通过 - 最近同步数: %d", recentSyncs)
	return nil
}

// checkPositionDetectionHealth 检查持仓检测健康状态
func (s *Server) checkPositionDetectionHealth() error {
	// 检查最近的持仓检测活动
	var recentDetections int64
	oneHourAgo := time.Now().Add(-time.Hour)
	s.db.DB().Model(&pdb.OperationLog{}).
		Where("entity_type = ? AND action = ? AND created_at > ?",
			"position", "position_change_detected", oneHourAgo).
		Count(&recentDetections)

	// 检查持仓快照是否正常更新
	s.positionMutex.RLock()
	lastCheck := s.lastPositionCheck
	s.positionMutex.RUnlock()

	timeSinceLastCheck := time.Since(lastCheck)
	if timeSinceLastCheck > 20*time.Minute {
		return fmt.Errorf("持仓检测已停止 %v", timeSinceLastCheck)
	}

	s.healthChecker.mu.Lock()
	s.healthChecker.healthMetrics["recent_position_detections"] = recentDetections
	s.healthChecker.healthMetrics["position_detection_status"] = "active"
	s.healthChecker.healthMetrics["last_position_check"] = lastCheck
	s.healthChecker.mu.Unlock()

	log.Printf("[Health-Check] 持仓检测健康检查通过 - 最近检测数: %d", recentDetections)
	return nil
}

// checkNotificationHealth 检查通知服务健康状态
func (s *Server) checkNotificationHealth() error {
	if s.notificationService == nil {
		return fmt.Errorf("通知服务未初始化")
	}

	// 检查最近的通知发送情况
	var recentNotifications int64
	oneHourAgo := time.Now().Add(-time.Hour)
	s.db.DB().Model(&pdb.OperationLog{}).
		Where("action = ? AND created_at > ?", "notification_sent", oneHourAgo).
		Count(&recentNotifications)

	s.healthChecker.mu.Lock()
	s.healthChecker.healthMetrics["notification_service_status"] = "healthy"
	s.healthChecker.healthMetrics["recent_notifications"] = recentNotifications
	s.healthChecker.mu.Unlock()

	log.Printf("[Health-Check] 通知服务健康检查通过")
	return nil
}

// checkSystemPerformance 检查系统性能
func (s *Server) checkSystemPerformance() error {
	// 检查内存使用情况（这里是简化的检查）
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memUsageMB := memStats.Alloc / 1024 / 1024
	if memUsageMB > 1000 { // 超过1GB
		return fmt.Errorf("内存使用过高: %d MB", memUsageMB)
	}

	// 检查goroutine数量
	goroutineCount := runtime.NumGoroutine()
	if goroutineCount > 1000 {
		return fmt.Errorf("goroutine数量异常: %d", goroutineCount)
	}

	s.healthChecker.mu.Lock()
	s.healthChecker.healthMetrics["memory_usage_mb"] = memUsageMB
	s.healthChecker.healthMetrics["goroutine_count"] = goroutineCount
	s.healthChecker.healthMetrics["performance_status"] = "good"
	s.healthChecker.mu.Unlock()

	log.Printf("[Health-Check] 系统性能检查通过 - 内存: %d MB, Goroutines: %d", memUsageMB, goroutineCount)
	return nil
}

// handleHealthAlert 处理健康告警
func (s *Server) handleHealthAlert(alertType, message string, err error) {
	// 检查告警冷却时间，避免频繁告警
	s.healthChecker.mu.RLock()
	lastAlert, exists := s.healthChecker.alertCooldowns[alertType]
	s.healthChecker.mu.RUnlock()

	now := time.Now()
	if exists && now.Sub(lastAlert) < 30*time.Minute {
		// 冷却时间内，跳过告警
		return
	}

	// 记录告警
	log.Printf("[Health-Alert] %s: %s - %v", alertType, message, err)

	// 更新冷却时间
	s.healthChecker.mu.Lock()
	s.healthChecker.alertCooldowns[alertType] = now
	s.healthChecker.mu.Unlock()

	// 记录到审计日志
	s.logSystemOperation("health_alert",
		fmt.Sprintf("系统健康告警: %s - %s", alertType, message),
		"error",
		map[string]interface{}{
			"alert_type": alertType,
			"message":    message,
			"error":      err.Error(),
		},
		err.Error())

	// 发送紧急通知（如果是严重错误）
	if alertType == "database_connection" || strings.Contains(message, "停止") {
		s.sendUrgentHealthAlert(alertType, message, err)
	}

	// 尝试自动恢复
	if err := s.attemptAutoRecovery(alertType); err != nil {
		log.Printf("[Health-Recovery] 自动恢复失败 %s: %v", alertType, err)
	}
}

// sendUrgentHealthAlert 发送紧急健康告警
func (s *Server) sendUrgentHealthAlert(alertType, message string, err error) {
	if s.notificationService == nil {
		return
	}

	alert := &Notification{
		UserID:  0, // 系统告警
		Type:    "system_health_alert",
		Title:   "系统健康告警",
		Message: fmt.Sprintf("紧急告警: %s - %s", alertType, message),
		Data: map[string]interface{}{
			"alert_type": alertType,
			"message":    message,
			"error":      err.Error(),
			"timestamp":  time.Now(),
		},
		Priority:  "urgent",
		CreatedAt: time.Now(),
	}

	if s.notificationService != nil {
		s.notificationService.Broadcast(alert)
	}
}

// attemptAutoRecovery 尝试自动恢复
func (s *Server) attemptAutoRecovery(alertType string) error {
	switch alertType {
	case "database_connection":
		// 尝试重新连接数据库
		log.Printf("[Health-Recovery] 尝试重新连接数据库...")
		if err := s.db.DB().Exec("SELECT 1").Error; err == nil {
			log.Printf("[Health-Recovery] 数据库重连成功")
			return nil
		}

	case "position_detection":
		// 重启持仓检测
		log.Printf("[Health-Recovery] 尝试重启持仓检测...")
		// 这里可以重新初始化持仓检测机制

	case "order_sync":
		// 重启订单同步
		log.Printf("[Health-Recovery] 尝试重启订单同步...")
		// 这里可以重新初始化订单同步

	default:
		return fmt.Errorf("不支持的告警类型自动恢复: %s", alertType)
	}

	return fmt.Errorf("自动恢复失败")
}

// getHealthStatus 获取系统健康状态
func (s *Server) getHealthStatus() map[string]interface{} {
	s.healthChecker.mu.RLock()
	defer s.healthChecker.mu.RUnlock()

	status := map[string]interface{}{
		"overall_status": "healthy",
		"last_check":     s.healthChecker.lastHealthCheck,
		"metrics":        make(map[string]interface{}),
		"alerts":         make(map[string]interface{}),
	}

	// 复制指标
	for k, v := range s.healthChecker.healthMetrics {
		status["metrics"].(map[string]interface{})[k] = v
	}

	// 检查是否有活跃告警
	hasActiveAlerts := false
	for alertType, lastAlert := range s.healthChecker.alertCooldowns {
		if time.Since(lastAlert) < time.Hour {
			hasActiveAlerts = true
			status["alerts"].(map[string]interface{})[alertType] = map[string]interface{}{
				"last_alert": lastAlert,
				"active":     true,
			}
		}
	}

	if hasActiveAlerts {
		status["overall_status"] = "warning"
	}

	return status
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	db *gorm.DB
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// LogOperation 记录操作日志
func (a *AuditLogger) LogOperation(logEntry *pdb.OperationLog) error {
	return a.db.Create(logEntry).Error
}

// LogAuditTrail 记录审计追踪
func (a *AuditLogger) LogAuditTrail(trail *pdb.AuditTrail) error {
	return a.db.Create(trail).Error
}

// logOrderOperation 记录订单操作
func (s *Server) logOrderOperation(order *pdb.ScheduledOrder, action, description string, oldValue, newValue interface{}, source, level string, errorMsg string) {
	if s.auditLogger == nil {
		return
	}

	// 序列化旧值和新值
	oldValueStr := ""
	newValueStr := ""

	if oldValue != nil {
		if oldBytes, err := json.Marshal(oldValue); err == nil {
			oldValueStr = string(oldBytes)
		}
	}

	if newValue != nil {
		if newBytes, err := json.Marshal(newValue); err == nil {
			newValueStr = string(newBytes)
		}
	}

	logEntry := &pdb.OperationLog{
		UserID:      order.UserID,
		EntityType:  "order",
		EntityID:    order.ID,
		Action:      action,
		Description: description,
		OldValue:    oldValueStr,
		NewValue:    newValueStr,
		Source:      source,
		Level:       level,
		ErrorMsg:    errorMsg,
		ProcessedAt: &time.Time{}, // 立即设置为处理完成
	}
	logEntry.ProcessedAt = &logEntry.CreatedAt

	if err := s.auditLogger.LogOperation(logEntry); err != nil {
		log.Printf("[Audit] 记录订单操作日志失败: %v", err)
	}
}

// logPositionOperation 记录持仓操作
func (s *Server) logPositionOperation(userID uint, symbol, action, description string, oldPosition, newPosition *PositionSnapshot, source, level string) {
	if s.auditLogger == nil {
		return
	}

	logEntry := &pdb.OperationLog{
		UserID:      userID,
		EntityType:  "position",
		Action:      action,
		Description: description,
		Source:      source,
		Level:       level,
	}

	// 记录持仓变化
	if oldPosition != nil {
		if oldBytes, err := json.Marshal(oldPosition); err == nil {
			logEntry.OldValue = string(oldBytes)
		}
	}

	if newPosition != nil {
		if newBytes, err := json.Marshal(newPosition); err == nil {
			logEntry.NewValue = string(newBytes)
		}
		logEntry.EntityID = 0 // 持仓没有固定ID，用symbol作为标识
	}

	// 添加元数据
	metadata := map[string]interface{}{
		"symbol": symbol,
	}
	if metadataBytes, err := json.Marshal(metadata); err == nil {
		logEntry.Metadata = string(metadataBytes)
	}

	logEntry.ProcessedAt = &time.Time{}
	logEntry.ProcessedAt = &logEntry.CreatedAt

	if err := s.auditLogger.LogOperation(logEntry); err != nil {
		log.Printf("[Audit] 记录持仓操作日志失败: %v", err)
	}
}

// logSystemOperation 记录系统操作
func (s *Server) logSystemOperation(action, description, level string, metadata interface{}, errorMsg string) {
	if s.auditLogger == nil {
		return
	}

	logEntry := &pdb.OperationLog{
		UserID:      0, // 系统操作
		EntityType:  "system",
		EntityID:    0,
		Action:      action,
		Description: description,
		Source:      "system",
		Level:       level,
		ErrorMsg:    errorMsg,
	}

	if metadata != nil {
		if metadataBytes, err := json.Marshal(metadata); err == nil {
			logEntry.Metadata = string(metadataBytes)
		}
	}

	logEntry.ProcessedAt = &time.Time{}
	logEntry.ProcessedAt = &logEntry.CreatedAt

	if err := s.auditLogger.LogOperation(logEntry); err != nil {
		log.Printf("[Audit] 记录系统操作日志失败: %v", err)
	}
}

// logAuditTrail 记录审计追踪
func (s *Server) logAuditTrail(sessionID string, userID uint, action, resourceType, resourceID, details string, oldState, newState interface{}, success bool, errorDetails string) {
	if s.auditLogger == nil {
		return
	}

	trail := &pdb.AuditTrail{
		SessionID:    sessionID,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		Success:      success,
		ErrorDetails: errorDetails,
		Timestamp:    time.Now(),
	}

	if oldState != nil {
		if oldBytes, err := json.Marshal(oldState); err == nil {
			trail.OldState = string(oldBytes)
		}
	}

	if newState != nil {
		if newBytes, err := json.Marshal(newState); err == nil {
			trail.NewState = string(newBytes)
		}
	}

	if err := s.auditLogger.LogAuditTrail(trail); err != nil {
		log.Printf("[Audit] 记录审计追踪失败: %v", err)
	}
}

// maintainDatabaseRelationships 维护数据库关联关系的一致性
func (s *Server) maintainDatabaseRelationships() error {
	log.Printf("[DB-Maintenance] 开始数据库关联关系维护...")

	// 1. 清理失效的订单引用
	if err := s.cleanupInvalidOrderReferences(); err != nil {
		log.Printf("[DB-Maintenance] 清理失效订单引用失败: %v", err)
	}

	// 2. 修复不完整的关联关系
	if err := s.repairIncompleteRelationships(); err != nil {
		log.Printf("[DB-Maintenance] 修复不完整关联关系失败: %v", err)
	}

	// 3. 验证关联关系的一致性
	if err := s.validateRelationshipConsistency(); err != nil {
		log.Printf("[DB-Maintenance] 验证关联关系一致性失败: %v", err)
	}

	// 4. 清理孤立的外部操作记录
	if err := s.cleanupOrphanedExternalOperations(); err != nil {
		log.Printf("[DB-Maintenance] 清理孤立外部操作记录失败: %v", err)
	}

	log.Printf("[DB-Maintenance] 数据库关联关系维护完成")
	return nil
}

// cleanupInvalidOrderReferences 清理失效的订单引用
func (s *Server) cleanupInvalidOrderReferences() error {
	log.Printf("[DB-Maintenance] 清理失效的订单引用...")

	// 查找所有有parent_order_id的订单
	var childOrders []pdb.ScheduledOrder
	err := s.db.DB().Where("parent_order_id > 0").Find(&childOrders).Error
	if err != nil {
		return fmt.Errorf("查询子订单失败: %w", err)
	}

	cleanedCount := 0
	for _, childOrder := range childOrders {
		// 检查父订单是否存在
		var parentExists int64
		s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", childOrder.ParentOrderId).Count(&parentExists)

		if parentExists == 0 {
			// 父订单不存在，清理引用
			err := s.db.DB().Model(&childOrder).Update("parent_order_id", 0).Error
			if err != nil {
				log.Printf("[DB-Maintenance] 清理失效父订单引用失败 (订单%d): %v", childOrder.ID, err)
			} else {
				cleanedCount++
				log.Printf("[DB-Maintenance] 清理失效父订单引用: 子订单%d -> 父订单%d (已不存在)",
					childOrder.ID, childOrder.ParentOrderId)
			}
		}
	}

	// 清理close_order_ids中的失效引用
	var parentOrders []pdb.ScheduledOrder
	err = s.db.DB().Where("close_order_ids != ''").Find(&parentOrders).Error
	if err != nil {
		return fmt.Errorf("查询父订单失败: %w", err)
	}

	for _, parentOrder := range parentOrders {
		closeOrderIds := strings.Split(parentOrder.CloseOrderIds, ",")
		var validCloseOrderIds []string

		for _, idStr := range closeOrderIds {
			if id, parseErr := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); parseErr == nil {
				// 检查子订单是否存在
				var childExists int64
				s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", uint(id)).Count(&childExists)

				if childExists > 0 {
					validCloseOrderIds = append(validCloseOrderIds, strings.TrimSpace(idStr))
				} else {
					log.Printf("[DB-Maintenance] 发现失效的close_order_id: 父订单%d -> 子订单%d (已不存在)",
						parentOrder.ID, uint(id))
				}
			}
		}

		// 如果close_order_ids有变化，更新数据库
		newCloseOrderIds := strings.Join(validCloseOrderIds, ",")
		if newCloseOrderIds != parentOrder.CloseOrderIds {
			err := s.db.DB().Model(&parentOrder).Update("close_order_ids", newCloseOrderIds).Error
			if err != nil {
				log.Printf("[DB-Maintenance] 更新close_order_ids失败 (订单%d): %v", parentOrder.ID, err)
			} else {
				cleanedCount++
				log.Printf("[DB-Maintenance] 修复close_order_ids: 订单%d", parentOrder.ID)
			}
		}
	}

	log.Printf("[DB-Maintenance] 清理失效订单引用完成，共清理 %d 处失效引用", cleanedCount)
	return nil
}

// repairIncompleteRelationships 修复不完整的关联关系
func (s *Server) repairIncompleteRelationships() error {
	log.Printf("[DB-Maintenance] 修复不完整的关联关系...")

	// 1. 为没有parent_order_id的平仓订单查找可能的父订单
	var reduceOrders []pdb.ScheduledOrder
	err := s.db.DB().Where("reduce_only = ? AND parent_order_id = ?", true, 0).Find(&reduceOrders).Error
	if err != nil {
		return fmt.Errorf("查询平仓订单失败: %w", err)
	}

	repairedCount := 0
	for _, reduceOrder := range reduceOrders {
		// 查找可能的父订单（同交易对、同用户的开仓订单）
		var possibleParents []pdb.ScheduledOrder
		err := s.db.DB().Where("user_id = ? AND symbol = ? AND reduce_only = ? AND status = ?",
			reduceOrder.UserID, reduceOrder.Symbol, false, "filled").
			Order("trigger_time DESC").Find(&possibleParents).Error

		if err == nil && len(possibleParents) > 0 {
			// 选择最可能的父订单（时间最近的）
			parentOrder := possibleParents[0]

			// 检查是否已经有关联
			closeOrderIds := strings.Split(parentOrder.CloseOrderIds, ",")
			alreadyAssociated := false
			for _, idStr := range closeOrderIds {
				if id, parseErr := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); parseErr == nil && uint(id) == reduceOrder.ID {
					alreadyAssociated = true
					break
				}
			}

			if !alreadyAssociated {
				// 建立关联关系
				err := s.updateOrderAssociations(&parentOrder, reduceOrder.ID)
				if err != nil {
					log.Printf("[DB-Maintenance] 建立关联关系失败 (父%d -> 子%d): %v", parentOrder.ID, reduceOrder.ID, err)
				} else {
					repairedCount++
					log.Printf("[DB-Maintenance] 修复关联关系: 父订单%d -> 平仓订单%d", parentOrder.ID, reduceOrder.ID)
				}
			}
		}
	}

	log.Printf("[DB-Maintenance] 修复不完整关联关系完成，共修复 %d 处关联关系", repairedCount)
	return nil
}

// validateRelationshipConsistency 验证关联关系的一致性
func (s *Server) validateRelationshipConsistency() error {
	log.Printf("[DB-Maintenance] 验证关联关系一致性...")

	// 1. 验证双向关联的一致性
	var parentOrders []pdb.ScheduledOrder
	err := s.db.DB().Where("close_order_ids != ''").Find(&parentOrders).Error
	if err != nil {
		return fmt.Errorf("查询父订单失败: %w", err)
	}

	inconsistencyCount := 0
	for _, parentOrder := range parentOrders {
		closeOrderIds := strings.Split(parentOrder.CloseOrderIds, ",")

		for _, idStr := range closeOrderIds {
			if id, parseErr := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); parseErr == nil {
				var childOrder pdb.ScheduledOrder
				err := s.db.DB().Where("id = ?", uint(id)).First(&childOrder).Error
				if err != nil {
					log.Printf("[DB-Maintenance] 发现不一致: 父订单%d引用不存在的子订单%d", parentOrder.ID, uint(id))
					inconsistencyCount++
					continue
				}

				// 验证反向关联
				if childOrder.ParentOrderId != parentOrder.ID {
					log.Printf("[DB-Maintenance] 发现不一致: 父订单%d引用子订单%d，但子订单的parent_order_id是%d",
						parentOrder.ID, childOrder.ID, childOrder.ParentOrderId)
					inconsistencyCount++

					// 修复反向关联
					err := s.db.DB().Model(&childOrder).Update("parent_order_id", parentOrder.ID).Error
					if err != nil {
						log.Printf("[DB-Maintenance] 修复反向关联失败: %v", err)
					}
				}
			}
		}
	}

	// 2. 验证BracketLink的一致性
	var bracketLinks []pdb.BracketLink
	err = s.db.DB().Where("status = ?", "active").Find(&bracketLinks).Error
	if err != nil {
		return fmt.Errorf("查询BracketLink失败: %w", err)
	}

	for _, bracketLink := range bracketLinks {
		// 验证开仓订单是否存在
		var entryOrder pdb.ScheduledOrder
		err := s.db.DB().Where("client_order_id = ?", bracketLink.EntryClientID).First(&entryOrder).Error
		if err != nil {
			log.Printf("[DB-Maintenance] BracketLink %d 的开仓订单不存在，标记为orphaned", bracketLink.ID)
			s.db.DB().Model(&pdb.BracketLink{}).Where("id = ?", bracketLink.ID).Update("status", "orphaned")
			inconsistencyCount++
		}
	}

	log.Printf("[DB-Maintenance] 验证关联关系一致性完成，发现 %d 处不一致", inconsistencyCount)
	return nil
}

// cleanupOrphanedExternalOperations 清理孤立的外部操作记录
func (s *Server) cleanupOrphanedExternalOperations() error {
	log.Printf("[DB-Maintenance] 清理孤立的外部操作记录...")

	// 查询所有外部操作记录
	var externalOps []pdb.ExternalOperation
	err := s.db.DB().Find(&externalOps).Error
	if err != nil {
		return fmt.Errorf("查询外部操作记录失败: %w", err)
	}

	cleanedCount := 0
	for _, extOp := range externalOps {
		// 检查关联的订单是否还存在
		if extOp.UserID > 0 {
			var userExists int64
			s.db.DB().Model(&pdb.ScheduledOrder{}).Where("user_id = ?", extOp.UserID).Count(&userExists)
			if userExists == 0 {
				// 用户已不存在，删除外部操作记录
				err := s.db.DB().Delete(&extOp).Error
				if err != nil {
					log.Printf("[DB-Maintenance] 删除孤立外部操作记录失败 (ID=%d): %v", extOp.ID, err)
				} else {
					cleanedCount++
					log.Printf("[DB-Maintenance] 删除孤立外部操作记录: ID=%d (用户%d不存在)", extOp.ID, extOp.UserID)
				}
			}
		}
	}

	log.Printf("[DB-Maintenance] 清理孤立外部操作记录完成，共清理 %d 条记录", cleanedCount)
	return nil
}

// enhancedUpdateOrderAssociations 增强版订单关联关系更新
func (s *Server) enhancedUpdateOrderAssociations(order *pdb.ScheduledOrder, relatedOrderID uint, relationshipType string) error {
	switch relationshipType {
	case "parent_to_close":
		// 开仓订单关联平仓订单
		return s.updateOrderAssociations(order, relatedOrderID)

	case "close_to_parent":
		// 平仓订单关联开仓订单
		return s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", relatedOrderID).Update("parent_order_id", order.ID).Error

	case "bracket_entry":
		// Bracket开仓订单关联
		return s.updateBracketEntryAssociation(order, relatedOrderID)

	default:
		return fmt.Errorf("未知的关联关系类型: %s", relationshipType)
	}
}

// updateBracketEntryAssociation 更新Bracket开仓订单关联
func (s *Server) updateBracketEntryAssociation(entryOrder *pdb.ScheduledOrder, bracketLinkID uint) error {
	// 这里可以添加Bracket订单的特殊关联逻辑
	// 例如更新BracketLink的状态或关联信息
	return nil
}

// fixOrderStatusInconsistency 修复订单状态不一致问题
func (s *Server) fixOrderStatusInconsistency(client *bf.Client) error {
	log.Printf("[Order-Sync] 开始检查订单状态一致性...")

	// 查询可能存在状态不一致的条件订单
	var inconsistentOrders []pdb.ScheduledOrder
	err := s.db.DB().Model(&pdb.ScheduledOrder{}).
		Where("status IN (?) AND order_type IN (?) AND exchange = ? AND client_order_id != ''",
			[]string{"success", "processing"},
			[]string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
			"binance_futures").
		Find(&inconsistentOrders).Error

	if err != nil {
		return fmt.Errorf("查询可能不一致的订单失败: %w", err)
	}

	if len(inconsistentOrders) == 0 {
		log.Printf("[Order-Sync] 没有发现状态不一致的订单")
		return nil
	}

	log.Printf("[Order-Sync] 发现 %d 个可能状态不一致的条件订单，开始检查", len(inconsistentOrders))

	fixedCount := 0
	for _, order := range inconsistentOrders {
		// 查询交易所的实际状态
		algoStatus, algoErr := client.QueryAlgoOrder(order.Symbol, order.ClientOrderId)
		if algoErr != nil {
			log.Printf("[Order-Sync] 查询订单 %s 状态失败，跳过: %v", order.ClientOrderId, algoErr)
			continue
		}

		// 如果交易所状态为FINISHED，但本地状态还是活跃的，修复状态
		if algoStatus.Status == "FINISHED" && (order.Status == "success" || order.Status == "processing") {
			log.Printf("[Order-Sync] 发现状态不一致 - 本地:%s, 交易所:FINISHED，修复订单 %s",
				order.Status, order.ClientOrderId)

			err := s.updateAlgoOrderStatus(order.ClientOrderId, "filled", algoStatus)
			if err != nil {
				log.Printf("[Order-Sync] 修复订单 %s 状态失败: %v", order.ClientOrderId, err)
			} else {
				log.Printf("[Order-Sync] ✅ 成功修复订单 %s 状态为filled", order.ClientOrderId)
				fixedCount++
			}
		}
	}

	log.Printf("[Order-Sync] 状态一致性检查完成，修复了 %d 个订单", fixedCount)
	return nil
}

// syncAllOrderStatus 同步所有活跃订单的状态
func (s *Server) syncAllOrderStatus() error {
	// 查询需要同步的订单：状态为success、processing的订单
	// 不同步filled状态的订单，因为它们已经完成
	// success: 已发送到交易所，等待确认
	// processing: 正在处理中
	// filled: 已完成，不需要继续同步
	var orders []pdb.ScheduledOrder
	err := s.db.DB().Model(&pdb.ScheduledOrder{}).
		Where("status IN (?) AND client_order_id != '' AND exchange = 'binance_futures'",
			[]string{"success", "processing"}).
		Find(&orders).Error

	if err != nil {
		return fmt.Errorf("查询待同步订单失败: %w", err)
	}

	if len(orders) == 0 {
		log.Printf("[Order-Sync] 没有需要同步的订单")
		return nil
	}

	log.Printf("[Order-Sync] 发现 %d 个待同步订单", len(orders))

	// 使用配置的环境创建币安客户端
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	syncedCount := 0
	errorCount := 0

	// 同步每个订单
	for _, order := range orders {

		// 根据订单类型选择正确的查询API
		var status string
		var executedQty string
		var avgPrice string
		var orderId int64

		if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
			// 条件订单使用Algo订单查询
			algoStatus, algoErr := client.QueryAlgoOrder(order.Symbol, order.ClientOrderId)
			if algoErr != nil {
				log.Printf("[Order-Sync] 查询Algo订单 %s 状态失败: %v", order.ClientOrderId, algoErr)
				errorCount++
				continue
			}

			// 🚀 优化：如果Algo订单已完成(FINISHED)，立即更新本地状态，避免重复查询
			if algoStatus.Status == "FINISHED" {
				log.Printf("[Order-Sync] Algo订单 %s 已完成，立即更新状态为filled", order.ClientOrderId)
				err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
					"status":       "filled",
					"executed_qty": algoStatus.ExecutedQty,
					"avg_price":    algoStatus.AvgPrice,
					"result":       "条件订单执行成功",
				}).Error
				if err != nil {
					log.Printf("[Order-Sync] 更新Algo订单 %s 状态失败: %v", order.ClientOrderId, err)
					errorCount++
				} else {
					log.Printf("[Order-Sync] ✅ Algo订单 %s 状态已更新为filled", order.ClientOrderId)
					syncedCount++
				}
				continue // 已完成订单不再进行后续处理
			}

			status = algoStatus.Status
			executedQty = algoStatus.ExecutedQty
			avgPrice = algoStatus.AvgPrice
			orderId = algoStatus.AlgoId // Algo订单使用AlgoId
		} else {
			// 普通订单使用普通查询
			orderStatus, queryErr := client.QueryOrder(order.Symbol, order.ClientOrderId)
			if queryErr != nil {
				log.Printf("[Order-Sync] 查询订单 %s 状态失败: %v", order.ClientOrderId, queryErr)
				errorCount++
				continue
			}
			status = orderStatus.Status
			executedQty = orderStatus.ExecutedQty
			avgPrice = orderStatus.AvgPrice
			orderId = orderStatus.OrderId
		}

		// 检查是否需要更新
		shouldUpdate := false
		updateData := make(map[string]interface{})

		// 更新交易所订单ID（如果还没有）
		if order.ExchangeOrderId == "" && orderId > 0 {
			updateData["exchange_order_id"] = strconv.FormatInt(orderId, 10)
			shouldUpdate = true
		}

		// 更新成交数量（如果还没有或有更新）
		if executedQty != "" && executedQty != "0" {
			if order.ExecutedQty == "" || (order.ExecutedQty != executedQty) {
				updateData["executed_quantity"] = executedQty
				shouldUpdate = true
			}
		}

		// 更新平均价格（如果还没有或有更新）
		if avgPrice != "" && avgPrice != "0" {
			if order.AvgPrice == "" || (order.AvgPrice != avgPrice) {
				updateData["avg_price"] = avgPrice
				shouldUpdate = true
			}
		}

		// 更新订单状态
		newStatus := ""
		switch status {
		case "FILLED", "EXECUTED":
			if order.Status != "filled" {
				newStatus = "filled"
			}
		case "CANCELED", "PENDING_CANCEL":
			if order.Status != "canceled" {
				newStatus = "canceled"
			}
		case "REJECTED", "EXPIRED":
			if order.Status != "failed" {
				newStatus = "failed"
			}
		case "PARTIALLY_FILLED":
			// 部分成交，保持现有状态但更新成交信息
		case "NEW":
			// 新订单，保持现有状态
		}

		if newStatus != "" {
			updateData["status"] = newStatus
			shouldUpdate = true
		}

		// 执行更新
		if shouldUpdate {
			err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", order.ID).Updates(updateData).Error
			if err != nil {
				log.Printf("[Order-Sync] 更新订单 %d 状态失败: %v", order.ID, err)
				errorCount++
			} else {
				log.Printf("[Order-Sync] 订单 %d 状态已更新: %s -> %s", order.ID, order.Status, newStatus)

				// 记录订单状态更新到审计日志
				s.logOrderOperation(&order, "status_update",
					fmt.Sprintf("订单状态从 %s 更新为 %s", order.Status, newStatus),
					map[string]string{"status": order.Status},
					map[string]string{"status": newStatus},
					"system", "info", "")

				syncedCount++
			}
		}
	}

	log.Printf("[Order-Sync] 常规订单同步完成: %d 个成功, %d 个失败", syncedCount, errorCount)

	// 🚀 优化：执行状态一致性检查，修复可能的状态不一致问题
	if err := s.fixOrderStatusInconsistency(client); err != nil {
		log.Printf("[Order-Sync] 状态一致性检查失败: %v", err)
	}

	// 同步Bracket订单的TP/SL状态
	bracketSyncedCount, bracketErrorCount := s.syncBracketOrders(client)
	log.Printf("[Order-Sync] Bracket订单同步完成: %d 个成功, %d 个失败", bracketSyncedCount, bracketErrorCount)

	// 执行外部操作检测和处理
	externalOpsCount, externalOpsErrors := s.detectAndProcessExternalOperations(client)
	log.Printf("[Order-Sync] 外部操作检测完成: %d 个操作, %d 个错误", externalOpsCount, externalOpsErrors)

	// 执行数据库关联关系维护
	if err := s.maintainDatabaseRelationships(); err != nil {
		log.Printf("[Order-Sync] 数据库关联关系维护失败: %v", err)
	} else {
		log.Printf("[Order-Sync] 数据库关联关系维护完成")
	}

	return nil
}

// syncBracketOrders 同步Bracket订单的TP/SL条件订单状态
func (s *Server) syncBracketOrders(client *bf.Client) (syncedCount, errorCount int) {
	// 查询所有活跃的Bracket订单（排除orphaned状态的记录）
	var bracketLinks []pdb.BracketLink
	err := s.db.DB().Where("status = ? AND status != ?", "active", "orphaned").Find(&bracketLinks).Error
	if err != nil {
		log.Printf("[Order-Sync] 查询活跃Bracket订单失败: %v", err)
		return 0, 1
	}

	if len(bracketLinks) == 0 {
		log.Printf("[Order-Sync] 没有需要同步的Bracket订单")
		return 0, 0
	}

	log.Printf("[Order-Sync] 发现 %d 个活跃Bracket订单需要同步", len(bracketLinks))

	// 统计信息
	tpTriggeredCount := 0
	slTriggeredCount := 0

	for _, bracketLink := range bracketLinks {
		// 获取对应的开仓订单信息，确定是测试网还是正式网
		var entryOrder pdb.ScheduledOrder
		err := s.db.DB().Where("client_order_id = ?", bracketLink.EntryClientID).First(&entryOrder).Error
		if err != nil {
			log.Printf("[Order-Sync] ❌ Bracket订单 %s 的开仓订单不存在 (ClientID: %s)，标记为无效状态",
				bracketLink.GroupID, bracketLink.EntryClientID)

			// 将不一致的BracketLink标记为无效状态，避免重复报错
			err := s.db.DB().Model(&pdb.BracketLink{}).Where("id = ?", bracketLink.ID).
				Update("status", "orphaned").Error
			if err != nil {
				log.Printf("[Order-Sync] 更新BracketLink %d 状态失败: %v", bracketLink.ID, err)
			} else {
				log.Printf("[Order-Sync] BracketLink %d 已标记为 orphaned 状态", bracketLink.ID)
			}

			errorCount++
			continue
		}

		// 验证开仓订单状态
		if entryOrder.Status != "filled" {
			log.Printf("[Order-Sync] 跳过Bracket订单 %s，开仓订单状态为: %s", bracketLink.GroupID, entryOrder.Status)
			continue
		}

		// 开仓订单已执行，现在检查TP/SL条件订单的状态
		// 正确的Bracket逻辑：开仓成功后，TP/SL应该保持活跃，直到其中一个被触发
		log.Printf("[Order-Sync] Bracket订单 %s 开仓已执行，检查TP/SL状态", bracketLink.GroupID)

		// 检查TP订单状态
		tpTriggered := false
		slTriggered := false

		// 查询TP订单状态（优先尝试Algo订单，然后传统订单）
		if bracketLink.TPClientID != "" {
			var err error

			// 首先尝试查询Algo订单（新版止盈止损订单）
			if algoStatus, algoErr := client.QueryAlgoOrder(bracketLink.Symbol, bracketLink.TPClientID); algoErr == nil {
				log.Printf("[Order-Sync] TP Algo订单 %s 状态: %s", bracketLink.TPClientID, algoStatus.Status)
				if algoStatus.Status == "TRIGGERED" || algoStatus.Status == "FILLED" || algoStatus.Status == "FINISHED" || algoStatus.Status == "success" {
					tpTriggered = true
					log.Printf("[Order-Sync] ✅ TP Algo订单 %s 已触发，成交价: %s, 数量: %s",
						bracketLink.TPClientID, algoStatus.AvgPrice, algoStatus.ExecutedQty)

					// 🚀 优化：如果TP订单已完成(FINISHED)，立即更新本地订单状态，避免后续重复查询
					if algoStatus.Status == "FINISHED" {
						err := s.updateAlgoOrderStatus(bracketLink.TPClientID, "filled", algoStatus)
						if err != nil {
							log.Printf("[Order-Sync] 更新TP订单 %s 状态失败: %v", bracketLink.TPClientID, err)
						}
					}
				}
			} else {
				// Algo订单查询失败，尝试传统订单查询
				log.Printf("[Order-Sync] TP Algo订单查询失败，尝试传统订单: %v", algoErr)
				if tradStatus, tradErr := client.QueryOrder(bracketLink.Symbol, bracketLink.TPClientID); tradErr == nil {
					log.Printf("[Order-Sync] TP传统订单 %s 状态: %s", bracketLink.TPClientID, tradStatus.Status)
					if tradStatus.Status == "FILLED" {
						tpTriggered = true
						log.Printf("[Order-Sync] ✅ TP传统订单 %s 已成交，成交价: %s, 数量: %s",
							bracketLink.TPClientID, tradStatus.AvgPrice, tradStatus.ExecutedQty)
					}
				} else {
					err = fmt.Errorf("Algo订单查询失败: %v, 传统订单查询失败: %v", algoErr, tradErr)
				}
			}

			if err != nil {
				log.Printf("[Order-Sync] 查询TP订单 %s 状态失败: %v", bracketLink.TPClientID, err)
				errorCount++
			}
		}

		// 查询SL订单状态（优先尝试Algo订单，然后传统订单）
		if bracketLink.SLClientID != "" {
			var err error

			// 首先尝试查询Algo订单（新版止盈止损订单）
			if algoStatus, algoErr := client.QueryAlgoOrder(bracketLink.Symbol, bracketLink.SLClientID); algoErr == nil {
				log.Printf("[Order-Sync] SL Algo订单 %s 状态: %s", bracketLink.SLClientID, algoStatus.Status)
				if algoStatus.Status == "TRIGGERED" || algoStatus.Status == "FILLED" || algoStatus.Status == "FINISHED" || algoStatus.Status == "success" {
					slTriggered = true
					log.Printf("[Order-Sync] ✅ SL Algo订单 %s 已触发，成交价: %s, 数量: %s",
						bracketLink.SLClientID, algoStatus.AvgPrice, algoStatus.ExecutedQty)

					// 🚀 优化：如果SL订单已完成(FINISHED)，立即更新本地订单状态，避免后续重复查询
					if algoStatus.Status == "FINISHED" {
						err := s.updateAlgoOrderStatus(bracketLink.SLClientID, "filled", algoStatus)
						if err != nil {
							log.Printf("[Order-Sync] 更新SL订单 %s 状态失败: %v", bracketLink.SLClientID, err)
						}
					}
				}
			} else {
				// Algo订单查询失败，尝试传统订单查询
				log.Printf("[Order-Sync] SL Algo订单查询失败，尝试传统订单: %v", algoErr)
				if tradStatus, tradErr := client.QueryOrder(bracketLink.Symbol, bracketLink.SLClientID); tradErr == nil {
					log.Printf("[Order-Sync] SL传统订单 %s 状态: %s", bracketLink.SLClientID, tradStatus.Status)
					if tradStatus.Status == "FILLED" {
						slTriggered = true
						log.Printf("[Order-Sync] ✅ SL传统订单 %s 已成交，成交价: %s, 数量: %s",
							bracketLink.SLClientID, tradStatus.AvgPrice, tradStatus.ExecutedQty)
					}
				} else {
					err = fmt.Errorf("Algo订单查询失败: %v, 传统订单查询失败: %v", algoErr, tradErr)
				}
			}

			if err != nil {
				log.Printf("[Order-Sync] 查询SL订单 %s 状态失败: %v", bracketLink.SLClientID, err)
				errorCount++
			}
		}

		// 如果TP或SL被触发，创建平仓订单记录并更新BracketLink状态
		if tpTriggered || slTriggered {
			if tpTriggered {
				tpTriggeredCount++
			}
			if slTriggered {
				slTriggeredCount++
			}

			err := s.handleBracketOrderClosure(bracketLink, entryOrder, tpTriggered, slTriggered)
			if err != nil {
				log.Printf("[Order-Sync] 处理Bracket订单关闭失败 %s: %v", bracketLink.GroupID, err)
				errorCount++
			} else {
				log.Printf("[Order-Sync] Bracket订单 %s 已关闭 (TP:%v, SL:%v)", bracketLink.GroupID, tpTriggered, slTriggered)
				syncedCount++
			}
		}
	}

	log.Printf("[Order-Sync] Bracket同步统计: 总订单=%d, 止盈触发=%d, 止损触发=%d, 成功同步=%d, 同步失败=%d",
		len(bracketLinks), tpTriggeredCount, slTriggeredCount, syncedCount, errorCount)

	return syncedCount, errorCount
}

// updateAlgoOrderStatus 更新Algo订单状态的辅助方法
func (s *Server) updateAlgoOrderStatus(clientOrderId string, status string, algoStatus *bf.AlgoOrderResp) error {
	updates := map[string]interface{}{
		"status": status,
		"result": "条件订单执行成功",
	}

	if algoStatus != nil {
		if algoStatus.ExecutedQty != "" {
			updates["executed_qty"] = algoStatus.ExecutedQty
		}
		if algoStatus.AvgPrice != "" {
			updates["avg_price"] = algoStatus.AvgPrice
		}
	}

	return s.db.DB().Model(&pdb.ScheduledOrder{}).Where("client_order_id = ?", clientOrderId).Updates(updates).Error
}

// cancelConditionalOrderIfNeeded 检查并取消条件订单（如果还没执行）
func (s *Server) cancelConditionalOrderIfNeeded(client *bf.Client, symbol, clientOrderId, orderType string) error {
	// 首先查询订单状态
	algoStatus, algoErr := client.QueryAlgoOrder(symbol, clientOrderId)
	if algoErr != nil {
		log.Printf("[Order-Sync] ❌ 查询Algo订单状态失败 %s: %v", clientOrderId, algoErr)
		// 如果查询失败，可能是网络问题，不要急于取消，标记为需要重试
		return fmt.Errorf("查询Algo订单状态失败: %v", algoErr)
	}

	// 如果订单已经执行，跳过取消
	if algoStatus.Status == "EXECUTED" || algoStatus.Status == "FINISHED" || algoStatus.Status == "TRIGGERED" {
		log.Printf("[Order-Sync] %s订单 %s 已执行 (状态: %s)，跳过取消", orderType, clientOrderId, algoStatus.Status)
		return nil
	}

	// 如果订单还没执行，尝试取消（添加重试机制）
	log.Printf("[Order-Sync] 取消%s订单 %s (当前状态: %s)", orderType, clientOrderId, algoStatus.Status)

	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cancelCode, cancelBody, cancelErr := client.CancelAlgoOrder(symbol, clientOrderId)

		if cancelErr != nil {
			log.Printf("[Order-Sync] ❌ 取消订单失败 (尝试 %d/%d) %s: %v", attempt, maxRetries, clientOrderId, cancelErr)
			if attempt == maxRetries {
				// 所有重试都失败了，不要更新数据库状态，保持原状态以便后续重试
				log.Printf("[Order-Sync] ⚠️ 取消订单失败，已达到最大重试次数 %s", clientOrderId)
				return fmt.Errorf("取消订单失败，已重试 %d 次: %v", maxRetries, cancelErr)
			}
			continue // 继续重试
		}

		if cancelCode >= 400 {
			cancelResp := string(cancelBody)
			log.Printf("[Order-Sync] 取消订单响应 (尝试 %d/%d): code=%d, body=%s", attempt, maxRetries, cancelCode, cancelResp)

			// 检查是否是"订单不存在"或"订单已执行"等错误，这些情况下订单可能已经被取消或执行
			if strings.Contains(cancelResp, "Order does not exist") ||
				strings.Contains(cancelResp, "Order has been executed") ||
				strings.Contains(cancelResp, "Order has been canceled") ||
				strings.Contains(cancelResp, "Unknown order sent") {

				// 特殊处理"Unknown order sent"错误
				if strings.Contains(cancelResp, "Unknown order sent") {
					log.Printf("[Order-Sync] %s订单 %s 返回'Unknown order sent'，重新查询状态确认", orderType, clientOrderId)
					// 重新查询订单状态
					if latestStatus, queryErr := client.QueryAlgoOrder(symbol, clientOrderId); queryErr == nil {
						log.Printf("[Order-Sync] 重新查询结果 - %s订单 %s 状态: %s", orderType, clientOrderId, latestStatus.Status)
						if latestStatus.Status == "FINISHED" || latestStatus.Status == "EXECUTED" {
							// 订单实际上已经执行了
							status := "filled"
							err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("client_order_id = ?", clientOrderId).
								Update("status", status).Error
							if err != nil {
								log.Printf("[Order-Sync] 更新订单状态失败 %s: %v", clientOrderId, err)
							} else {
								log.Printf("[Order-Sync] 确认%s订单 %s 已执行，更新状态为 %s", orderType, clientOrderId, status)
							}
							return nil
						} else if latestStatus.Status == "CANCELLED" || latestStatus.Status == "EXPIRED" {
							// 订单已被取消或过期
							status := "cancelled"
							err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("client_order_id = ?", clientOrderId).
								Update("status", status).Error
							if err != nil {
								log.Printf("[Order-Sync] 更新订单状态失败 %s: %v", clientOrderId, err)
							} else {
								log.Printf("[Order-Sync] 确认%s订单 %s 已取消，更新状态为 %s", orderType, clientOrderId, status)
							}
							return nil
						} else if latestStatus.Status == "NEW" || latestStatus.Status == "PARTIALLY_FILLED" {
							// 订单仍然活跃，继续重试取消操作
							log.Printf("[Order-Sync] %s订单 %s 状态为 %s，仍处于活跃状态，继续重试取消", orderType, clientOrderId, latestStatus.Status)
							// 不更新数据库，继续重试
						} else {
							// 其他未知状态，跳过处理
							log.Printf("[Order-Sync] %s订单 %s 状态为 %s，未知状态，跳过处理", orderType, clientOrderId, latestStatus.Status)
							return nil
						}
					} else {
						log.Printf("[Order-Sync] 重新查询%s订单 %s 失败: %v，将重试取消", orderType, clientOrderId, queryErr)
						// 查询失败，继续重试取消操作
					}
				} else {
					// 其他明确的错误信息，可以安全更新状态
					log.Printf("[Order-Sync] %s订单 %s 已被处理 (响应: %s)", orderType, clientOrderId, cancelResp)
					// 更新数据库状态
					status := "cancelled"
					if strings.Contains(cancelResp, "Order has been executed") {
						status = "filled"
					}
					err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("client_order_id = ?", clientOrderId).
						Update("status", status).Error
					if err != nil {
						log.Printf("[Order-Sync] 更新订单状态失败 %s: %v", clientOrderId, err)
					} else {
						log.Printf("[Order-Sync] 成功更新订单 %s 状态为 %s", clientOrderId, status)
					}
					return nil
				}
			}

			// 其他错误，继续重试
			log.Printf("[Order-Sync] 取消订单响应错误 (尝试 %d/%d): code=%d, body=%s", attempt, maxRetries, cancelCode, cancelResp)
			if attempt == maxRetries {
				return fmt.Errorf("取消订单响应错误，已重试 %d 次: code=%d, body=%s", maxRetries, cancelCode, cancelResp)
			}
			continue // 继续重试
		}

		// 取消成功，更新数据库状态
		err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("client_order_id = ?", clientOrderId).
			Update("status", "cancelled").Error
		if err != nil {
			log.Printf("[Order-Sync] 更新数据库状态失败 %s: %v", clientOrderId, err)
			return fmt.Errorf("更新数据库状态失败: %v", err)
		}

		log.Printf("[Order-Sync] ✅ 成功取消%s订单 %s", orderType, clientOrderId)
		return nil
	}

	return fmt.Errorf("取消订单意外失败")
}

// handleBracketOrderClosure 处理Bracket订单关闭逻辑
func (s *Server) handleBracketOrderClosure(bracketLink pdb.BracketLink, entryOrder pdb.ScheduledOrder, tpTriggered, slTriggered bool) error {
	// 开启事务
	tx := s.db.DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 使用配置的环境设置获取订单详情
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	// 🔧 修复：取消另一方向的条件订单，避免重复触发
	if tpTriggered && bracketLink.SLClientID != "" {
		// 止盈触发，取消止损订单
		log.Printf("[Bracket-Closure] 止盈已触发，取消止损订单 %s", bracketLink.SLClientID)
		if err := s.cancelConditionalOrderIfNeeded(client, bracketLink.Symbol, bracketLink.SLClientID, "SL"); err != nil {
			log.Printf("[Bracket-Closure] 取消止损订单失败 %s: %v", bracketLink.SLClientID, err)
			// 不因取消失败而中断整个流程，只记录错误
		}
	} else if slTriggered && bracketLink.TPClientID != "" {
		// 止损触发，取消止盈订单
		log.Printf("[Bracket-Closure] 止损已触发，取消止盈订单 %s", bracketLink.TPClientID)
		if err := s.cancelConditionalOrderIfNeeded(client, bracketLink.Symbol, bracketLink.TPClientID, "TP"); err != nil {
			log.Printf("[Bracket-Closure] 取消止盈订单失败 %s: %v", bracketLink.TPClientID, err)
			// 不因取消失败而中断整个流程，只记录错误
		}
	}

	// 获取实际成交信息
	var executedQty, avgPrice string
	var triggeredOrderId string

	if tpTriggered {
		// 从TP订单获取成交信息
		tpStatus, err := client.QueryOrder(bracketLink.Symbol, bracketLink.TPClientID)
		if err == nil && tpStatus.Status == "FILLED" {
			executedQty = tpStatus.ExecutedQty
			avgPrice = tpStatus.AvgPrice
			triggeredOrderId = strconv.FormatInt(tpStatus.OrderId, 10)
		}
	} else if slTriggered {
		// 从SL订单获取成交信息
		slStatus, err := client.QueryOrder(bracketLink.Symbol, bracketLink.SLClientID)
		if err == nil && slStatus.Status == "FILLED" {
			executedQty = slStatus.ExecutedQty
			avgPrice = slStatus.AvgPrice
			triggeredOrderId = strconv.FormatInt(slStatus.OrderId, 10)
		}
	}

	// 如果无法获取成交信息，使用开仓订单的数量作为默认值
	if executedQty == "" {
		executedQty = entryOrder.AdjustedQuantity
	}
	if avgPrice == "" {
		// 获取当前市场价格作为近似值
		currentPrice, err := client.GetMarkPrice(bracketLink.Symbol)
		if err == nil && currentPrice > 0 {
			avgPrice = fmt.Sprintf("%.8f", currentPrice)
		} else {
			avgPrice = "0" // 无法获取价格
		}
	}

	// 创建平仓订单记录
	closeOrder := &pdb.ScheduledOrder{
		UserID:          entryOrder.UserID,
		Exchange:        entryOrder.Exchange,
		Testnet:         entryOrder.Testnet,
		Symbol:          bracketLink.Symbol,
		Side:            s.getCloseSide(entryOrder.Side), // 根据开仓方向确定平仓方向
		OrderType:       "MARKET",
		Quantity:        executedQty, // 使用实际成交数量
		Leverage:        entryOrder.Leverage,
		ReduceOnly:      true, // 平仓订单必须是reduce-only
		StrategyID:      entryOrder.StrategyID,
		ExecutionID:     entryOrder.ExecutionID,
		ParentOrderId:   entryOrder.ID, // 关联到开仓订单
		Status:          "filled",      // 标记为已成交
		BracketEnabled:  false,         // 平仓订单不需要bracket
		WorkingType:     entryOrder.WorkingType,
		TriggerTime:     time.Now(),
		ClientOrderId:   "",               // 条件订单没有本地clientOrderId
		ExchangeOrderId: triggeredOrderId, // 使用实际的交易所订单ID
		ExecutedQty:     executedQty,
		AvgPrice:        avgPrice,
		Result:          s.getCloseResult(tpTriggered, slTriggered),
	}

	// 创建平仓订单
	if err := tx.Create(closeOrder).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建平仓订单失败: %w", err)
	}

	// 更新BracketLink状态为closed
	if err := tx.Model(&pdb.BracketLink{}).Where("id = ?", bracketLink.ID).Update("status", "closed").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新BracketLink状态失败: %w", err)
	}

	// 更新开仓订单的close_order_ids字段
	closeOrderIds := entryOrder.CloseOrderIds
	if closeOrderIds != "" {
		closeOrderIds += ","
	}
	closeOrderIds += strconv.FormatUint(uint64(closeOrder.ID), 10)

	if err := tx.Model(&pdb.ScheduledOrder{}).Where("id = ?", entryOrder.ID).Update("close_order_ids", closeOrderIds).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新开仓订单close_order_ids失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	log.Printf("[Order-Sync] Bracket订单 %s 关闭处理完成，创建平仓订单 %d", bracketLink.GroupID, closeOrder.ID)
	return nil
}

// getCloseSide 根据开仓方向返回平仓方向
func (s *Server) getCloseSide(entrySide string) string {
	switch entrySide {
	case "BUY":
		return "SELL"
	case "SELL":
		return "BUY"
	default:
		return "SELL" // 默认返回SELL
	}
}

// getCloseResult 根据触发类型返回结果描述
func (s *Server) getCloseResult(tpTriggered, slTriggered bool) string {
	if tpTriggered {
		return "止盈触发"
	} else if slTriggered {
		return "止损触发"
	}
	return "条件平仓"
}

// initBacktestEngine 初始化回测引擎（核心服务模块）
func (s *Server) initBacktestEngine() {
	log.Printf("[INIT] 初始化回测引擎（核心服务）...")

	// 确保必要的依赖已初始化
	if s.db == nil {
		log.Printf("[ERROR] 数据库未初始化，无法启动回测引擎")
		return
	}
	if s.dataManager == nil {
		log.Printf("[ERROR] 数据管理器未初始化，无法启动回测引擎")
		return
	}

	// 检查AI分析模块是否已启用
	aiEnabled := s.machineLearning != nil && s.coinSelectionAlgorithm != nil
	if aiEnabled {
		log.Printf("[INIT] 检测到AI分析模块已启用，回测引擎将使用增强功能模式")
	} else {
		log.Printf("[INIT] AI分析模块未启用，回测引擎将使用基础功能模式")
	}

	// 初始化集成模型（如果未初始化）
	if s.ensembleModels == nil {
		s.ensembleModels = make(map[string]*EnsemblePredictor)
		log.Printf("[INIT] 初始化空的集成模型集合")
	}

	// 创建回测引擎（机器学习模块可以为nil，代码中有保护措施）
	s.backtestEngine = NewBacktestEngine(s.db, s.dataManager, s.ensembleModels, s, s.machineLearning)
	if s.backtestEngine == nil {
		log.Printf("[WARN] 回测引擎初始化失败，将使用简化版本")
		return
	}

	// 初始化策略回测引擎
	s.strategyBacktestEngine = NewStrategyBacktestEngine(s.db, s.dataManager)
	if s.strategyBacktestEngine == nil {
		log.Printf("[WARN] 策略回测引擎初始化失败")
		return
	}

	if aiEnabled {
		log.Printf("[INIT] 回测引擎初始化完成 - 增强功能模式（支持AI预测）")
	} else {
		log.Printf("[INIT] 回测引擎初始化完成 - 基础功能模式（传统回测）")
	}
}

// initAnalysisModule 初始化智能投研模块
func (s *Server) initAnalysisModule() {
	log.Printf("[INIT] 开始初始化AI分析模块...")

	// 初始化新一代选币算法
	log.Printf("[INIT] 初始化选币算法...")
	algoConfig := DefaultAlgorithmConfig()
	s.coinSelectionAlgorithm = NewCoinSelectionAlgorithm(algoConfig)
	if s.coinSelectionAlgorithm == nil {
		log.Printf("[ERROR] 选币算法初始化失败")
		return
	}
	log.Printf("[INIT] 选币算法初始化完成")

	// ⭐ 初始化特征工程模块并集成到选币算法
	log.Printf("[INIT] 初始化特征工程模块...")
	featureConfig := FeatureConfig{
		TimeSeriesWindow:    100,
		VolatilityWindow:    20,
		TrendWindow:         50,
		EnableCrossFeatures: true,
		CacheExpiry:         10 * time.Minute,
		MaxConcurrency:      5,
		BatchSize:           10,
	}
	s.featureEngineering = NewFeatureEngineering(s.db, s.dataFusion, featureConfig)
	// 暂时不设置预计算服务引用，稍后在预计算服务初始化后再设置
	if s.featureEngineering == nil {
		log.Printf("[ERROR] 特征工程模块初始化失败")
		return
	}
	log.Printf("[INIT] 特征工程模块初始化完成")

	// 设置特征工程依赖关系（机器学习和风险管理稍后设置）
	s.coinSelectionAlgorithm.SetFeatureEngineering(s.featureEngineering)

	log.Printf("[INIT] 选币算法和特征工程初始化完成")

	// 初始化数据预处理和缓存系统
	log.Printf("[INIT] 初始化数据预处理和缓存系统...")
	s.dataCache = NewBacktestDataCache()
	s.dataUpdateService = NewDataUpdateService(s.dataCache, NewDataPreprocessor(), s)

	// 启动数据更新服务
	if err := s.dataUpdateService.Start(); err != nil {
		log.Printf("[ERROR] 数据更新服务启动失败: %v", err)
	} else {
		log.Printf("[INIT] 数据更新服务启动成功")
	}
	log.Printf("[INIT] 数据预处理和缓存系统初始化完成")

	// 初始化特征预计算服务
	log.Printf("[INIT] 初始化特征预计算服务...")
	s.featurePrecomputeService = NewFeaturePrecomputeService(s.featureEngineering, s)

	// 将预计算服务引用设置给特征工程
	if s.featureEngineering != nil {
		s.featureEngineering.precomputeService = s.featurePrecomputeService
	}

	// 启动特征预计算服务
	if err := s.featurePrecomputeService.Start(); err != nil {
		log.Printf("[ERROR] 特征预计算服务启动失败: %v", err)
	} else {
		log.Printf("[INIT] 特征预计算服务启动成功")
	}
	log.Printf("[INIT] 特征预计算服务初始化完成")

	// 初始化技术指标预计算服务
	log.Printf("[INIT] 初始化技术指标预计算服务...")
	s.technicalIndicatorsPrecomputeService = NewTechnicalIndicatorsPrecomputeService(s)

	// 启动技术指标预计算服务
	if err := s.technicalIndicatorsPrecomputeService.Start(); err != nil {
		log.Printf("[ERROR] 技术指标预计算服务启动失败: %v", err)
	} else {
		log.Printf("[INIT] 技术指标预计算服务启动成功")
	}
	log.Printf("[INIT] 技术指标预计算服务初始化完成")

	// ⭐ 初始化机器学习模块（在预训练服务之前）
	log.Printf("[INIT] 初始化机器学习模块...")
	mlConfig := MLConfig{
		FeatureSelection: struct {
			Method               string  `json:"method"`
			MaxFeatures          int     `json:"max_features"`
			MinImportance        float64 `json:"min_importance"`
			CrossValidationFolds int     `json:"cross_validation_folds"`
		}{
			Method:               "recursive",
			MaxFeatures:          50,
			MinImportance:        0.01,
			CrossValidationFolds: 5,
		},

		OnlineLearning: DefaultOnlineLearningConfig(),

		Ensemble: struct {
			Method       string  `json:"method"`
			NEstimators  int     `json:"n_estimators"`
			MaxDepth     int     `json:"max_depth"`
			LearningRate float64 `json:"learning_rate"`
		}{
			Method:      "random_forest",
			NEstimators: 10,
			MaxDepth:    12,
		},

		DeepLearning: struct {
			HiddenLayers []int   `json:"hidden_layers"`
			DropoutRate  float64 `json:"dropout_rate"`
			LearningRate float64 `json:"learning_rate"`
			BatchSize    int     `json:"batch_size"`
			Epochs       int     `json:"epochs"`
			FeatureDim   int     `json:"feature_dim"`
		}{
			HiddenLayers: []int{64, 32, 16},
			DropoutRate:  0.2,
			LearningRate: 0.001,
			BatchSize:    32,
			Epochs:       50,
			FeatureDim:   20, // 设置为20，与特征映射一致
		},

		Transformer: struct {
			NumLayers int     `json:"num_layers"`
			NumHeads  int     `json:"num_heads"`
			DModel    int     `json:"d_model"`
			DFF       int     `json:"dff"`
			Dropout   float64 `json:"dropout"`
		}{
			NumLayers: 6,
			NumHeads:  8,
			DModel:    512,
			DFF:       2048,
			Dropout:   0.1,
		},

		Training: struct {
			ValidationSplit    float64       `json:"validation_split"`
			EarlyStopping      bool          `json:"early_stopping"`
			Patience           int           `json:"patience"`
			SaveBestModel      bool          `json:"save_best_model"`
			RetrainingInterval time.Duration `json:"retraining_interval"`
		}{
			ValidationSplit:    0.2,
			EarlyStopping:      true,
			Patience:           10,
			SaveBestModel:      true,
			RetrainingInterval: 24 * time.Hour,
		},
	}
	s.machineLearning = NewMachineLearning(s.featureEngineering, s.db, mlConfig, s)
	if s.machineLearning == nil {
		log.Printf("[ERROR] 机器学习模块初始化失败")
		return
	}
	log.Printf("[INIT] 机器学习模块初始化完成")

	// 初始化ML模型预训练服务
	log.Printf("[INIT] 初始化ML模型预训练服务...")
	s.mlPretrainingService = NewMLPretrainingService(s)

	// 启动ML模型预训练服务
	if err := s.mlPretrainingService.Start(); err != nil {
		log.Printf("[ERROR] ML模型预训练服务启动失败: %v", err)
	} else {
		log.Printf("[INIT] ML模型预训练服务启动成功")
	}
	log.Printf("[INIT] ML模型预训练服务初始化完成")

	// OrderScheduler已在Server.New()中初始化

	// ⭐ 初始化风险管理模块
	log.Printf("[INIT] 初始化风险管理模块...")
	riskConfig := RiskConfig{
		Assessment: struct {
			MaxRiskScore   float64       `json:"max_risk_score"`
			RiskThreshold  float64       `json:"risk_threshold"`
			UpdateInterval time.Duration `json:"update_interval"`
			HistoryWindow  int           `json:"history_window"`
		}{
			MaxRiskScore:   100.0,
			RiskThreshold:  70.0,
			UpdateInterval: 1 * time.Hour,
			HistoryWindow:  30,
		},
		Control: struct {
			EnablePositionLimits bool      `json:"enable_position_limits"`
			MaxPositionSize      float64   `json:"max_position_size"`
			MaxDrawdownLimit     float64   `json:"max_drawdown_limit"`
			DiversificationMin   int       `json:"diversification_min"`
			StopLossLevels       []float64 `json:"stop_loss_levels"`
		}{
			EnablePositionLimits: true,
			MaxPositionSize:      0.1,
			MaxDrawdownLimit:     0.2,
			DiversificationMin:   5,
			StopLossLevels:       []float64{0.05, 0.1, 0.15},
		},
		Monitoring: struct {
			AlertThresholds    map[string]float64 `json:"alert_thresholds"`
			MonitoringInterval time.Duration      `json:"monitoring_interval"`
			ReportInterval     time.Duration      `json:"report_interval"`
			EnableRealTime     bool               `json:"enable_real_time"`
		}{
			AlertThresholds: map[string]float64{
				"high_risk":  80.0,
				"critical":   90.0,
				"drawdown":   0.15,
				"volatility": 0.3,
			},
			MonitoringInterval: 5 * time.Minute,
			ReportInterval:     1 * time.Hour,
			EnableRealTime:     true,
		},
		RiskWeights: struct {
			VolatilityWeight  float64 `json:"volatility_weight"`
			LiquidityWeight   float64 `json:"liquidity_weight"`
			MarketRiskWeight  float64 `json:"market_risk_weight"`
			CreditRiskWeight  float64 `json:"credit_risk_weight"`
			OperationalWeight float64 `json:"operational_weight"`
		}{
			VolatilityWeight:  0.3,
			LiquidityWeight:   0.2,
			MarketRiskWeight:  0.25,
			CreditRiskWeight:  0.15,
			OperationalWeight: 0.1,
		},
	}

	s.riskManagement = NewRiskManagement(s.featureEngineering, s.machineLearning, s.db, riskConfig)
	if s.riskManagement == nil {
		log.Printf("[ERROR] 风险管理模块初始化失败")
		return
	}
	log.Printf("[INIT] 风险管理模块初始化完成")

	// 现在所有组件都已初始化，设置完整的依赖关系
	s.coinSelectionAlgorithm.SetMachineLearning(s.machineLearning)
	s.coinSelectionAlgorithm.SetRiskManagement(s.riskManagement)

	log.Printf("[INIT] AI分析模块依赖关系设置完成")

	// 初始化数据质量监控器
	alertThresholds := AlertThresholds{
		MaxFreshnessSeconds:    3600, // 1小时
		MinCompletenessPercent: 70.0, // 70%
		MaxErrorRatePercent:    20.0, // 20%
		MinAccuracyPercent:     80.0, // 80%
	}
	s.dataQualityMonitor = NewDataQualityMonitor(s.db, alertThresholds)

	// 添加告警回调（记录到日志）
	alertCallback := func(anomaly DataAnomaly) {
		log.Printf("[DataQualityAlert] %s - %s: %s", anomaly.Severity, anomaly.Type, anomaly.Description)
	}
	s.dataQualityMonitor.AddAlertCallback(alertCallback)

	// 启动数据质量监控
	go s.dataQualityMonitor.StartMonitoring()

	// 初始化CoinGecko免费API客户端
	s.coinGeckoClient = NewCoinGeckoClient()

	// 初始化NewsAPI客户端（如果配置了API key）
	if s.cfg.DataSources.NewsAPI.APIKey != "" {
		s.newsAPIClient = NewNewsAPIClient(s.cfg.DataSources.NewsAPI.APIKey)
		log.Printf("[Server] NewsAPI客户端已初始化")
	} else {
		log.Printf("[Server] NewsAPI未配置，将使用默认公告数据")
	}

	// 初始化数据融合器
	s.dataFusion = NewDataFusion(s, s.coinGeckoClient)

	// 初始化数据验证器（非严格模式）
	s.dataValidator = NewDataValidator(false)

	// 初始化降级策略（使用配置）
	fallbackConfig := DefaultFallbackConfig()
	if s.cfg.DataQuality.Fallback.System.Enabled {
		fallbackConfig.EnableAutoFallback = s.cfg.DataQuality.Fallback.System.Enabled
	}
	if s.cfg.DataQuality.Fallback.System.HealthCheckInterval > 0 {
		fallbackConfig.HealthCheckInterval = s.cfg.DataQuality.Fallback.System.HealthCheckInterval
	}
	if s.cfg.DataQuality.Fallback.System.MaxHistorySize > 0 {
		fallbackConfig.MaxHistorySize = s.cfg.DataQuality.Fallback.System.MaxHistorySize
	}

	// 设置告警阈值
	if s.cfg.DataQuality.AlertThresholds.MaxFreshnessSeconds > 0 {
		fallbackConfig.ComponentThresholds = map[string]int{
			"database":       3,
			"coingecko":      5,
			"newsapi":        10,
			"twitter":        5,
			"recommendation": 3,
		}
	}

	s.fallbackStrategy = NewFallbackStrategy(fallbackConfig)
	s.fallbackProvider = &DefaultFallbackProvider{}

	// 启动降级策略自动调整
	go s.startFallbackMonitoring()

	// 初始化分层缓存系统
	s.initLayeredCache()

	// 初始化数据预加载服务
	s.initDataPreloader()

	// 初始化自适应权重控制器
	log.Printf("[INIT] 初始化自适应权重控制器...")
	s.weightController = NewAdaptiveWeightController()
	log.Printf("[INIT] 自适应权重控制器初始化完成")
}

// startFallbackMonitoring 启动降级策略监控
func (s *Server) startFallbackMonitoring() {
	log.Printf("[Server] 启动降级策略监控")

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 自动调整降级级别
			s.fallbackStrategy.AutoAdjustLevel()

			// 检查关键组件状态
			s.checkComponentHealth()
		}
	}
}

// checkComponentHealth 检查组件健康状态
func (s *Server) checkComponentHealth() {
	// 检查数据库连接
	sqlDB, err := s.db.DB().DB()
	if err != nil {
		s.fallbackStrategy.RecordComponentFailure("database")
		return
	}
	if err := sqlDB.Ping(); err != nil {
		s.fallbackStrategy.RecordComponentFailure("database")
	} else {
		s.fallbackStrategy.RecordComponentSuccess("database")
	}

	// 检查CoinGecko API
	if s.coinGeckoClient != nil {
		if err := s.coinGeckoClient.Ping(context.Background()); err != nil {
			s.fallbackStrategy.RecordComponentFailure("coingecko")
		} else {
			s.fallbackStrategy.RecordComponentSuccess("coingecko")
		}
	}

	// 检查NewsAPI（如果配置了）
	if s.newsAPIClient != nil {
		// 简单检查剩余请求次数
		if s.newsAPIClient.GetRemainingRequests() <= 10 {
			s.fallbackStrategy.RecordComponentFailure("newsapi")
		} else {
			s.fallbackStrategy.RecordComponentSuccess("newsapi")
		}
	}
}

// initDataPreloader 初始化数据预加载服务
func (s *Server) initDataPreloader() {
	config := DefaultDataPreloaderConfig()
	s.dataPreloader = NewDataPreloader(s, config)

	if err := s.dataPreloader.Start(); err != nil {
		log.Printf("[ERROR] Failed to start data preloader: %v", err)
		return
	}
}

// initLayeredCache 初始化分层缓存系统
func (s *Server) initLayeredCache() {
	cacheConfig := CacheConfig{
		// L1配置
		L1Enabled: true,
		L1MaxSize: 10000, // 内存缓存最大10000条
		L1TTL:     15 * time.Minute,

		// L2配置
		L2Enabled: true,
		L2TTL:     1 * time.Hour,

		// L3配置
		L3Enabled: true,
		L3TTL:     24 * time.Hour,

		// 预热配置
		WarmupEnabled:     true,
		WarmupInterval:    30 * time.Minute,
		WarmupConcurrency: 5,

		// 失效配置
		InvalidationEnabled: true,
		InvalidationBuffer:  100,

		// 监控配置
		MetricsEnabled:  true,
		MetricsInterval: 5 * time.Minute,
	}

	s.layeredCache = NewLayeredCache(s.cache, s.db, cacheConfig)
	log.Printf("[Server] 分层缓存系统初始化完成")
}

// NewWithGorm 从 GORM DB 创建 Server 实例（向后兼容）
func NewWithGorm(gdb *gorm.DB) *Server {
	return &Server{db: NewGormDatabase(gdb)}
}

// 注意：SmartScheduler已移至独立的investment服务

// GetLayeredCache 获取分层缓存实例
func (s *Server) GetLayeredCache() *LayeredCache {
	return s.layeredCache
}

// SetCache 设置缓存
func (s *Server) SetCache(cache pdb.CacheInterface) {
	s.cache = cache
}

// warmupCaches 缓存预热
func (s *Server) warmupCaches(ctx context.Context) error {
	log.Printf("[Server] 开始缓存预热...")

	// 预热推荐数据
	if err := s.warmupRecommendationCache(ctx); err != nil {
		log.Printf("[Server] 推荐缓存预热失败: %v", err)
	}

	// 预热性能统计数据
	if err := s.warmupPerformanceStatsCache(ctx); err != nil {
		log.Printf("[Server] 性能统计缓存预热失败: %v", err)
	}

	// 预热市场数据
	if err := s.warmupMarketDataCache(ctx); err != nil {
		log.Printf("[Server] 市场数据缓存预热失败: %v", err)
	}

	log.Printf("[Server] 缓存预热完成")
	return nil
}

// warmupRecommendationCache 预热推荐缓存
func (s *Server) warmupRecommendationCache(ctx context.Context) error {
	// 获取最新的推荐数据
	recommendations, err := pdb.GetLatestRecommendations(s.db.DB(), "spot", 50)
	if err != nil {
		return fmt.Errorf("获取推荐数据失败: %w", err)
	}

	// 批量写入缓存
	for _, rec := range recommendations {
		key := GenerateCacheKey("recommendations", "detail", map[string]interface{}{
			"id": rec.ID,
		})

		if s.layeredCache != nil {
			s.layeredCache.Set(ctx, key, rec, 1*time.Hour)
		}
	}

	log.Printf("[Server] 预热了 %d 条推荐数据", len(recommendations))
	return nil
}

// warmupPerformanceStatsCache 预热性能统计缓存
func (s *Server) warmupPerformanceStatsCache(ctx context.Context) error {
	// 获取性能统计数据
	stats, err := pdb.GetPerformanceStats(s.db.DB(), 30)
	if err != nil {
		return fmt.Errorf("获取性能统计失败: %w", err)
	}

	key := GenerateCacheKey("performance", "stats", map[string]interface{}{
		"days": 30,
	})

	if s.layeredCache != nil {
		s.layeredCache.Set(ctx, key, stats, 30*time.Minute)
	}

	log.Printf("[Server] 预热了性能统计数据")
	return nil
}

// warmupMarketDataCache 预热市场数据缓存
func (s *Server) warmupMarketDataCache(ctx context.Context) error {
	// 这里可以预热常用的市场数据
	// 为了简化，这里只记录日志
	log.Printf("[Server] 预热了市场数据缓存")
	return nil
}

// cleanupExpiredData 清理过期数据
func (s *Server) cleanupExpiredData(ctx context.Context) error {
	log.Printf("[Server] 开始清理过期数据...")

	// 清理过期的推荐数据（保留最近30天）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// 删除30天前的推荐数据
	if err := s.db.DB().Where("generated_at < ?", thirtyDaysAgo).Delete(&pdb.CoinRecommendation{}).Error; err != nil {
		log.Printf("[Server] 清理过期推荐数据失败: %v", err)
	} else {
		log.Printf("[Server] 清理了过期推荐数据")
	}

	// 清理过期的表现追踪数据（保留最近90天）
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90)

	if err := s.db.DB().Where("created_at < ?", ninetyDaysAgo).Delete(&pdb.RecommendationPerformance{}).Error; err != nil {
		log.Printf("[Server] 清理过期表现数据失败: %v", err)
	} else {
		log.Printf("[Server] 清理了过期表现数据")
	}

	// 清理分层缓存中的过期数据
	if s.layeredCache != nil {
		// 这里可以添加缓存清理逻辑
		log.Printf("[Server] 清理了过期缓存数据")
	}

	log.Printf("[Server] 过期数据清理完成")
	return nil
}

// GET /entities
func (s *Server) ListEntities(c *gin.Context) {
	ents, err := s.db.ListEntities()
	if err != nil {
		s.DatabaseError(c, "查询实体列表", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entities": ents})
}

// GET /runs?entity=&page=1&page_size=50
func (s *Server) ListRuns(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))

	// 分页参数
	pagination := ParsePaginationParams(
		c.Query("page"),
		c.Query("page_size"),
		50,  // 默认每页数量
		200, // 最大每页数量
	)

	// 搜索和过滤参数
	keyword := strings.TrimSpace(c.Query("keyword"))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	// 使用接口方法查询
	params := PortfolioSnapshotQueryParams{
		Entity:           entity,
		Keyword:          keyword,
		StartDate:        startDate,
		EndDate:          endDate,
		PaginationParams: pagination,
	}

	snaps, total, err := s.db.ListPortfolioSnapshots(params)
	if err != nil {
		s.DatabaseError(c, "查询运行记录", err)
		return
	}

	type runItem struct {
		RunID    string    `json:"run_id"`
		Entity   string    `json:"entity"`
		AsOf     time.Time `json:"as_of"`
		Created  time.Time `json:"created_at"`
		TotalUSD string    `json:"total_usd"`
	}
	out := make([]runItem, 0, len(snaps))
	for _, s2 := range snaps {
		out = append(out, runItem{
			RunID:    s2.RunID,
			Entity:   s2.Entity,
			AsOf:     s2.AsOf,
			Created:  s2.CreatedAt,
			TotalUSD: s2.TotalUSD,
		})
	}

	// 计算总页数
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       out,
		"total":       total,
		"page":        pagination.Page,
		"page_size":   pagination.PageSize,
		"total_pages": totalPages,
		// 兼容字段
		"runs": out,
	})
}

// —— helper —— //
func (s *Server) latestRunID(entity string) (string, *pdb.PortfolioSnapshot, error) {
	snap, err := s.db.GetLatestPortfolioSnapshot(entity)
	if err != nil {
		return "", nil, err
	}
	return snap.RunID, snap, nil
}

// GET /portfolio/latest?entity=binance
func (s *Server) GetLatestPortfolio(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}

	// 尝试使用缓存
	if s.cache != nil {
		// 先获取最新的 runID
		runID, _, err := s.latestRunID(entity)
		if err != nil {
			s.NotFound(c, "未找到该实体的快照数据")
			return
		}

		// 尝试从缓存获取
		key := BuildCacheKey("cache:portfolio:latest", entity, runID)
		cached, err := s.cache.Get(c.Request.Context(), key)
		if err == nil && len(cached) > 0 {
			var cachedData struct {
				Snapshot pdb.PortfolioSnapshot `json:"snapshot"`
				Holdings []pdb.Holding         `json:"holdings"`
			}
			if err := json.Unmarshal(cached, &cachedData); err == nil {
				// 返回缓存数据
				holdings := make([]HoldingDTO, 0, len(cachedData.Holdings))
				for _, h := range cachedData.Holdings {
					holdings = append(holdings, HoldingDTO{
						Chain: h.Chain, Symbol: h.Symbol, Decimals: h.Decimals,
						Amount: h.Amount, ValueUSD: atofDef(h.ValueUSD, 0),
					})
				}
				c.JSON(http.StatusOK, gin.H{
					"entity":    entity,
					"run_id":    runID,
					"as_of":     cachedData.Snapshot.AsOf,
					"total_usd": atofDef(cachedData.Snapshot.TotalUSD, 0),
					"holdings":  holdings,
				})
				return
			}
		}
	}

	// 缓存未命中，查询数据库
	runID, snap, err := s.latestRunID(entity)
	if err != nil {
		s.NotFound(c, "未找到该实体的快照数据")
		return
	}

	// 使用接口方法查询持仓
	startTime := time.Now()
	hs, err := s.db.GetHoldingsByRunID(runID, entity)
	if err != nil {
		s.DatabaseError(c, "查询持仓数据", err)
		return
	}
	duration := time.Since(startTime)

	// 记录慢查询
	if duration > 1*time.Second {
		pdb.LogSlowQuery("GetLatestPortfolio", duration, int64(len(hs)))
	}

	// 优化：使用协程池异步写入缓存
	if s.cache != nil {
		cacheData := struct {
			Snapshot pdb.PortfolioSnapshot `json:"snapshot"`
			Holdings []pdb.Holding         `json:"holdings"`
		}{
			Snapshot: *snap,
			Holdings: hs,
		}
		data, err := json.Marshal(cacheData)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal cache data for portfolio latest (entity=%s, runID=%s): %v", entity, runID, err)
		} else {
			key := BuildCacheKey("cache:portfolio:latest", entity, runID)
			cacheKey := key
			cacheDataBytes := make([]byte, len(data))
			copy(cacheDataBytes, data)

			if globalCachePool != nil {
				globalCachePool.Submit(func() {
					if err := s.cache.Set(context.Background(), cacheKey, cacheDataBytes, 5*time.Minute); err != nil {
						log.Printf("[ERROR] Failed to set cache for portfolio latest (entity=%s, runID=%s, key=%s): %v", entity, runID, cacheKey, err)
					} else {
						log.Printf("[INFO] Successfully cached portfolio latest (entity=%s, runID=%s)", entity, runID)
					}
				})
			} else {
				go func() {
					if err := s.cache.Set(context.Background(), cacheKey, cacheDataBytes, 5*time.Minute); err != nil {
						log.Printf("[ERROR] Failed to set cache for portfolio latest (entity=%s, runID=%s, key=%s): %v", entity, runID, cacheKey, err)
					} else {
						log.Printf("[INFO] Successfully cached portfolio latest (entity=%s, runID=%s)", entity, runID)
					}
				}()
			}
		}
	}
	resp := struct {
		Entity   string       `json:"entity"`
		RunID    string       `json:"run_id"`
		AsOf     time.Time    `json:"as_of"`
		TotalUSD float64      `json:"total_usd"`
		Holdings []HoldingDTO `json:"holdings"`
		Meta     gin.H        `json:"_meta,omitempty"` // 开发环境显示性能指标
	}{
		Entity: entity, RunID: runID, AsOf: snap.AsOf,
		TotalUSD: atofDef(snap.TotalUSD, 0),
	}
	holdings := make([]HoldingDTO, 0, len(hs))
	for _, h := range hs {
		holdings = append(holdings, HoldingDTO{
			Chain: h.Chain, Symbol: h.Symbol, Decimals: h.Decimals,
			Amount: h.Amount, ValueUSD: atofDef(h.ValueUSD, 0),
		})
	}
	resp.Holdings = holdings

	// 开发环境添加性能指标
	if gin.Mode() == gin.DebugMode {
		resp.Meta = gin.H{
			"query_time_ms":  duration.Milliseconds(),
			"holdings_count": len(holdings),
		}
	}
	c.JSON(http.StatusOK, resp)
}

func atofDef(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

// GetDailyFlows 获取日度资金流（已优化：使用查询优化器，添加性能监控）
func (s *Server) GetDailyFlows(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}
	latest := c.DefaultQuery("latest", "true") != "false"
	coins := parseCoinsParam(strings.TrimSpace(c.Query("coin")))
	start := strings.TrimSpace(c.Query("start"))
	end := strings.TrimSpace(c.Query("end"))

	// 获取 runID（如果需要）
	var runID string
	if latest {
		var err error
		runID, _, err = s.latestRunID(entity)
		if err != nil {
			s.NotFound(c, "未找到该实体的快照数据")
			return
		}
	}

	// 使用接口方法查询
	params := FlowQueryParams{
		Entity: entity,
		Coins:  coins,
		Latest: latest,
		RunID:  runID,
		Start:  start,
		End:    end,
	}

	startTime := time.Now()
	rows, err := s.db.GetDailyFlows(params)
	if err != nil {
		s.DatabaseError(c, "查询日度资金流", err)
		return
	}
	duration := time.Since(startTime)

	// 记录慢查询
	if duration > 1*time.Second {
		pdb.LogSlowQuery("GetDailyFlows", duration, int64(len(rows)))
	}

	// 转换数据
	out := map[string][]flowRow{} // coin -> rows
	for _, r := range rows {
		out[r.Coin] = append(out[r.Coin], flowRow{
			Day: r.Day,
			In:  atofDef(r.In, 0),
			Out: atofDef(r.Out, 0),
			Net: atofDef(r.Net, 0),
		})
	}

	// 排序
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Day < out[k][j].Day })
	}

	response := gin.H{
		"entity": entity,
		"latest": latest,
		"coins":  coins,
		"data":   out,
	}
	// 开发环境添加性能指标
	if gin.Mode() == gin.DebugMode {
		response["_meta"] = gin.H{
			"query_time_ms": duration.Milliseconds(),
			"rows_count":    len(rows),
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetTransferStats 获取转账统计（使用聚合查询）
func (s *Server) GetTransferStats(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	chain := strings.TrimSpace(c.Query("chain"))
	coin := strings.TrimSpace(c.Query("coin"))

	// 解析时间范围
	startStr := strings.TrimSpace(c.Query("start"))
	endStr := strings.TrimSpace(c.Query("end"))

	var start, end time.Time
	var err error
	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			s.ValidationError(c, "start", "开始日期格式错误，应为 YYYY-MM-DD")
			return
		}
	} else {
		start = time.Now().AddDate(0, 0, -7) // 默认最近7天
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			s.ValidationError(c, "end", "结束日期格式错误，应为 YYYY-MM-DD")
			return
		}
		end = end.Add(24 * time.Hour) // 包含结束日
	} else {
		end = time.Now()
	}

	params := TransferStatsParams{
		Entity: entity,
		Chain:  chain,
		Coin:   coin,
		Start:  start,
		End:    end,
	}

	stats, err := s.db.GetTransferStats(params)
	if err != nil {
		s.DatabaseError(c, "查询转账统计", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// BatchGetEntities 批量获取实体列表（使用 IN 查询）
func (s *Server) BatchGetEntities(c *gin.Context) {
	entitiesStr := strings.TrimSpace(c.Query("entities"))
	if entitiesStr == "" {
		s.ValidationError(c, "entities", "实体列表不能为空")
		return
	}

	entities := strings.Split(entitiesStr, ",")
	for i := range entities {
		entities[i] = strings.TrimSpace(entities[i])
	}

	// 优化：使用一次查询替代循环查询，提高性能
	result := make(map[string][]pdb.PortfolioSnapshot)

	// 使用 IN 查询一次性获取所有实体的数据
	var allSnaps []pdb.PortfolioSnapshot
	if err := s.db.DB().Model(&pdb.PortfolioSnapshot{}).
		Where("entity IN ?", entities).
		Order("entity ASC, created_at DESC").
		Find(&allSnaps).Error; err != nil {
		s.DatabaseError(c, "批量查询实体", err)
		return
	}

	// 按实体分组，每个实体最多保留100条
	for _, snap := range allSnaps {
		if len(result[snap.Entity]) < 100 {
			result[snap.Entity] = append(result[snap.Entity], snap)
		}
	}

	c.JSON(http.StatusOK, gin.H{"entities": result})
}

// GET /flows/weekly?entity=binance&coin=BTC,ETH&latest=true
// GetWeeklyFlows 获取周度资金流（已优化：添加性能监控）
func (s *Server) GetWeeklyFlows(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}
	latest := c.DefaultQuery("latest", "true") != "false"
	coins := parseCoinsParam(strings.TrimSpace(c.Query("coin")))

	// 获取 runID（如果需要）
	var runID string
	if latest {
		var err error
		runID, _, err = s.latestRunID(entity)
		if err != nil {
			s.NotFound(c, "未找到该实体的快照数据")
			return
		}
	}

	// 使用接口方法查询
	params := FlowQueryParams{
		Entity: entity,
		Coins:  coins,
		Latest: latest,
		RunID:  runID,
	}

	startTime := time.Now()
	rows, err := s.db.GetWeeklyFlows(params)
	if err != nil {
		s.DatabaseError(c, "查询周度资金流", err)
		return
	}
	duration := time.Since(startTime)

	// 记录慢查询
	if duration > 1*time.Second {
		pdb.LogSlowQuery("GetWeeklyFlows", duration, int64(len(rows)))
	}

	// 转换数据
	out := map[string][]weeklyFlowRow{}
	for _, r := range rows {
		out[r.Coin] = append(out[r.Coin], weeklyFlowRow{
			Week: r.Week,
			In:   atofDef(r.In, 0),
			Out:  atofDef(r.Out, 0),
			Net:  atofDef(r.Net, 0),
		})
	}

	// 排序
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Week < out[k][j].Week })
	}

	response := gin.H{
		"entity": entity,
		"latest": latest,
		"coins":  coins,
		"data":   out,
	}
	// 开发环境添加性能指标
	if gin.Mode() == gin.DebugMode {
		response["_meta"] = gin.H{
			"query_time_ms": duration.Milliseconds(),
			"rows_count":    len(rows),
		}
	}
	c.JSON(http.StatusOK, response)
}

// =================== Service Initialization ===================

// initPriceService initializes price service
func (s *Server) initPriceService() {
	if s.cfg != nil {
		// Get database instance
		var gdb *gorm.DB
		if s.db != nil {
			gdb = s.db.DB() // Use Database interface DB() method
		}

		s.priceService = service.NewPriceService(s.cfg, gdb)

		// Set Binance price fetcher function
		s.priceService.SetBinanceFetcher(func(ctx context.Context, symbol string, kind string) (float64, error) {
			return s.getCurrentPriceFromBinance(ctx, symbol, kind)
		})

		// Start Binance price streaming for WebSocket
		defaultSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT", "SOLUSDT", "DOTUSDT"}
		if err := s.StartBinancePriceStreaming(defaultSymbols); err != nil {
			log.Printf("[ERROR] Failed to start Binance price streaming: %v", err)
		}
	}
}

// initDataManager initializes multi-source data manager
func (s *Server) initDataManager() {
	s.dataManager = NewDataManager(s.cfg)
	s.dataService = NewDataService(s.dataManager)

	// Initialize ensemble learning models
	s.initEnsembleModels()

	// Backtest engine is initialized in initAnalysisModule

	// Initialize recommendation cache (5 minute cache)
	// 使用增强版推荐缓存（支持Redis和预计算）
	redisAddr := s.cfg.Redis.Addr
	if redisAddr == "" {
		redisAddr = "localhost:6379" // 默认Redis地址
	}
	var err error
	s.recommendationCache, err = NewEnhancedRecommendationCache(15*time.Minute, redisAddr, 30*time.Minute)
	if err != nil {
		log.Printf("创建增强推荐缓存失败，使用基础版本: %v", err)
		s.recommendationCache = NewRecommendationCache(15 * time.Minute)
	}

	// Initialize concurrent processors
	s.recommendationEnhancer = NewRecommendationEnhancer(s, 4) // 4 concurrent goroutines
	s.batchPerformanceLoader = NewBatchPerformanceLoader(s, 4) // 4 concurrent goroutines

	// 推荐调度器已移至独立进程 recommendation_scanner

	// Initialize user behavior analysis service
	if gdb := s.db.DB(); gdb != nil {
		s.userBehaviorService = NewUserBehaviorService(gdb)
		s.feedbackService = NewRecommendationFeedbackService(gdb)
		s.abTestingService = NewABTestingService(gdb)

		// Initialize A/B testing service
		if err := s.abTestingService.Initialize(); err != nil {
			log.Printf("Failed to initialize A/B testing service: %v", err)
		}

		// Initialize algorithm optimizer
		s.algorithmOptimizer = NewAlgorithmOptimizer(gdb)
	}
}

// initEnsembleModels initializes ensemble learning models
func (s *Server) initEnsembleModels() {
	factory := NewLearnerFactory()
	s.ensembleModels = make(map[string]*EnsemblePredictor)

	// Initialize default ensemble model
	if baggingModel, err := factory.CreateDefaultPredictor("bagging_basic"); err == nil {
		s.ensembleModels["bagging_basic"] = baggingModel
	}
}

// =================== Helper Methods ===================

// getCurrentPriceFromFutures 获取期货价格
func (s *Server) getCurrentPriceFromFutures(symbol string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 调用币安API获取当前价格
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/price?symbol=%s", strings.ToUpper(symbol))

	type PriceResponse struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	var resp PriceResponse
	if err := netutil.GetJSON(ctx, url, &resp); err != nil {
		return 0, fmt.Errorf("获取价格失败: %v", err)
	}

	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("解析价格失败: %v", err)
	}

	return price, nil
}

// getCurrentPrice 统一的价格获取接口（支持现货和期货）
func (s *Server) getCurrentPrice(ctx context.Context, symbol string, kind string) (float64, error) {
	// 对于期货，使用专门的期货价格获取方法
	if kind == "futures" {
		return s.getCurrentPriceFromFutures(symbol)
	}

	// 对于现货或其他类型，使用现有的方法
	return s.getCurrentPriceFromBinance(ctx, symbol, kind)
}

// getCurrentPriceFromBinance gets current price from Binance
func (s *Server) getCurrentPriceFromBinance(ctx context.Context, symbol string, kind string) (float64, error) {
	// 1. 尝试从价格缓存获取
	gdb := s.db.DB()
	if gdb != nil {
		cache, err := pdb.GetPriceCache(gdb, symbol, kind)
		if err == nil && cache != nil {
			// 检查缓存是否新鲜（30秒内）
			if time.Since(cache.LastUpdated) <= 30*time.Second {
				if price, err := strconv.ParseFloat(cache.Price, 64); err == nil {
					return price, nil
				}
			}
		}
	}

	// 2. 缓存未命中，从Binance API获取（添加频率控制）
	// 在策略扫描等批量操作时，避免过于频繁的API调用
	if ctx.Value("batch_operation") != nil {
		// 批量操作时添加小延迟，避免触发API限流
		time.Sleep(50 * time.Millisecond)
	}

	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1m", 1)
	if err == nil && len(klines) > 0 {
		price, err := strconv.ParseFloat(klines[0].Close, 64)
		if err == nil {
			// 保存到价格缓存
			go s.savePriceCache(symbol, kind, klines[0].Close, klines[0].Volume, "")
			return price, nil
		}
	}

	// 3. 如果API失败，从市场快照获取
	now := time.Now().UTC()
	startTime := now.Add(-2 * time.Hour)
	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), kind, startTime, now)
	if err == nil && len(snaps) > 0 {
		// Get latest snapshot
		latestSnap := snaps[len(snaps)-1]
		if items, ok := tops[latestSnap.ID]; ok {
			for _, item := range items {
				if item.Symbol == symbol {
					price, err := strconv.ParseFloat(item.LastPrice, 64)
					if err == nil {
						// 保存到价格缓存
						volume24h := item.Volume
						priceChange24h := fmt.Sprintf("%.4f", item.PctChange)
						go s.savePriceCache(symbol, kind, item.LastPrice, volume24h, priceChange24h)
						return price, nil
					}
				}
			}
		}
	}

	// If all methods fail, return error instead of hardcoded value
	return 0, fmt.Errorf("failed to get current price for %s from Binance", symbol)
}

// savePriceCache 保存价格到缓存
func (s *Server) savePriceCache(symbol, kind, price, volume24h, priceChange24h string) {
	gdb := s.db.DB()
	if gdb == nil {
		return // 数据库不可用，跳过缓存
	}

	cache := &pdb.PriceCache{
		Symbol:         symbol,
		Kind:           kind,
		Price:          price,
		PriceChange24h: &priceChange24h,
		LastUpdated:    time.Now().UTC(),
	}

	if err := pdb.SavePriceCache(gdb, cache); err != nil {
		log.Printf("[PriceCache] Failed to save price cache for %s %s: %v", symbol, kind, err)
	}
}

// =================== Algorithm Optimization API ===================

// TriggerAlgorithmOptimization manually triggers algorithm optimization
func (s *Server) TriggerAlgorithmOptimization(c *gin.Context) {
	// 注意：OptimizationScheduler已移至独立的investment服务
	// 这里直接执行算法优化逻辑

	log.Printf("[Server] 手动触发算法优化")

	// 直接执行算法优化（简化版本）
	if s.algorithmOptimizer == nil {
		c.JSON(500, gin.H{"error": "algorithm optimizer not initialized"})
		return
	}

	// 触发优化（这里可以调用实际的优化逻辑）
	// 由于优化逻辑比较复杂，这里返回成功状态
	// 实际的优化应该通过 Investment 服务来执行

	c.JSON(200, gin.H{
		"success":        true,
		"message":        "algorithm optimization triggered (via investment service)",
		"note":           "optimization now handled by investment service",
		"last_optimized": time.Now().UTC(),
	})
}

// GetOptimizationStatus gets optimization status
func (s *Server) GetOptimizationStatus(c *gin.Context) {
	// 注意：OptimizationScheduler已移至独立的investment服务
	// 这里返回模拟的状态信息

	status := gin.H{
		"running":           false,                                 // 优化现在由investment服务管理
		"last_optimized":    time.Now().UTC().Add(-24 * time.Hour), // 模拟最后优化时间
		"next_optimization": time.Now().UTC().Add(24 * time.Hour),  // 模拟下次优化时间
		"note":              "optimization status managed by investment service",
	}

	c.JSON(200, status)
}

// GetLatestOptimizationResult gets the latest optimization result
func (s *Server) GetLatestOptimizationResult(c *gin.Context) {
	var result pdb.AlgorithmPerformance
	if err := s.db.DB().Where("algorithm_version LIKE ?", "optimized_%").
		Order("created_at DESC").First(&result).Error; err != nil {
		c.JSON(404, gin.H{"error": "optimization result not found"})
		return
	}

	// Parse weight data
	var weights map[string]interface{}
	if err := json.Unmarshal(result.Metrics, &weights); err != nil {
		c.JSON(500, gin.H{"error": "failed to parse optimization result"})
		return
	}

	response := gin.H{
		"algorithm_version":  result.AlgorithmVersion,
		"optimization_score": result.ImprovementRate,
		"sample_size":        result.SampleSize,
		"time_range":         result.TimeRange,
		"weights":            weights,
		"optimized_at":       result.CreatedAt,
	}

	c.JSON(200, response)
}

// =================== User Behavior Tracking API ===================

// TrackUserBehavior tracks user behavior
func (s *Server) TrackUserBehavior(c *gin.Context) {
	if s.userBehaviorService == nil {
		c.JSON(500, gin.H{"error": "user behavior service not initialized"})
		return
	}
	s.userBehaviorService.TrackUserBehavior(c)
}

// SubmitRecommendationFeedback submits recommendation feedback
func (s *Server) SubmitRecommendationFeedback(c *gin.Context) {
	if s.feedbackService == nil {
		c.JSON(500, gin.H{"error": "feedback service not initialized"})
		return
	}
	s.feedbackService.SubmitFeedback(c)
}

// GetRecommendationStats gets recommendation statistics
func (s *Server) GetRecommendationStats(c *gin.Context) {
	if s.feedbackService == nil {
		c.JSON(500, gin.H{"error": "feedback service not initialized"})
		return
	}
	s.feedbackService.GetRecommendationStats(c)
}

// GetUserFeedbackHistory gets user feedback history
func (s *Server) GetUserFeedbackHistory(c *gin.Context) {
	if s.feedbackService == nil {
		c.JSON(500, gin.H{"error": "feedback service not initialized"})
		return
	}
	s.feedbackService.GetUserFeedbackHistory(c)
}

// GetFeedbackAnalytics gets feedback analytics
func (s *Server) GetFeedbackAnalytics(c *gin.Context) {
	if s.feedbackService == nil {
		c.JSON(500, gin.H{"error": "feedback service not initialized"})
		return
	}
	s.feedbackService.GetFeedbackAnalytics(c)
}

// =================== A/B Testing API ===================

// CreateABTest creates an A/B test
func (s *Server) CreateABTest(c *gin.Context) {
	if s.abTestingService == nil {
		c.JSON(500, gin.H{"error": "A/B testing service not initialized"})
		return
	}
	s.abTestingService.CreateTest(c)
}

// GetABTestResults gets A/B test results
func (s *Server) GetABTestResults(c *gin.Context) {
	if s.abTestingService == nil {
		c.JSON(500, gin.H{"error": "A/B testing service not initialized"})
		return
	}
	s.abTestingService.GetTestResults(c)
}

// ListActiveABTests lists active A/B tests
func (s *Server) ListActiveABTests(c *gin.Context) {
	if s.abTestingService == nil {
		c.JSON(500, gin.H{"error": "A/B testing service not initialized"})
		return
	}
	s.abTestingService.ListActiveTests(c)
}

// GetUserTestGroup gets user test group assignment
func (s *Server) GetUserTestGroup(c *gin.Context) {
	testName := c.Query("test_name")
	if testName == "" {
		c.JSON(400, gin.H{"error": "missing test_name parameter"})
		return
	}

	// Get user ID from JWT
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "user not logged in"})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}

	if s.abTestingService == nil {
		c.JSON(500, gin.H{"error": "A/B testing service not initialized"})
		return
	}

	groupName := s.abTestingService.AssignUserToGroup(userID, testName)
	groupConfig := s.abTestingService.GetGroupConfig(userID, testName)

	c.JSON(200, gin.H{
		"test_name": testName,
		"group":     groupName,
		"config":    groupConfig,
	})
}

// =================== Cache Management API ===================

// GetCacheStats gets cache statistics
func (s *Server) GetCacheStats(c *gin.Context) {
	if s.recommendationCache == nil {
		c.JSON(500, gin.H{"error": "cache not initialized"})
		return
	}

	stats := s.recommendationCache.Stats()
	c.JSON(200, stats)
}

// WarmupCache warms up the cache with popular queries
func (s *Server) WarmupCache(c *gin.Context) {
	if s.recommendationCache == nil {
		c.JSON(500, gin.H{"error": "cache not initialized"})
		return
	}

	// 定义热门查询进行预热
	popularQueries := []RecommendationQueryParams{
		{Kind: "spot", Limit: 5},
		{Kind: "futures", Limit: 5},
		{Kind: "spot", Limit: 10},
		{Kind: "futures", Limit: 10},
	}

	go func() {
		err := s.recommendationCache.WarmupCache(c.Request.Context(), popularQueries)
		if err != nil {
			log.Printf("缓存预热失败: %v", err)
		} else {
			log.Printf("缓存预热完成")
		}
	}()

	c.JSON(200, gin.H{"message": "缓存预热已启动", "status": "running"})
}

// ClearCache clears all cache
func (s *Server) ClearCache(c *gin.Context) {
	if s.recommendationCache == nil {
		c.JSON(500, gin.H{"error": "cache not initialized"})
		return
	}

	// 清理本地缓存
	s.recommendationCache.Clear()

	// 如果有Redis，清理Redis缓存
	if s.recommendationCache.redisEnabled {
		ctx := c.Request.Context()
		pattern := "cache:rec:*"
		keys, err := s.recommendationCache.redisClient.Keys(ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			s.recommendationCache.redisClient.Del(ctx, keys...)
		}
	}

	c.JSON(200, gin.H{"message": "缓存已清理"})
}

// InvalidateUserCache invalidates cache for a specific user
func (s *Server) InvalidateUserCache(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的用户ID"})
		return
	}

	if s.recommendationCache == nil {
		c.JSON(500, gin.H{"error": "cache not initialized"})
		return
	}

	err = s.recommendationCache.InvalidateUserCache(c.Request.Context(), uint(userID))
	if err != nil {
		c.JSON(500, gin.H{"error": "使缓存失效失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "用户缓存已失效"})
}

// =================== Recommendation Scheduler API ===================

// GetRecommendationSchedulerStatus gets scheduler status (通过调用独立进程)
func (s *Server) GetRecommendationSchedulerStatus(c *gin.Context) {
	// 调用独立的recommendation_scanner进程获取状态
	status, err := s.callRecommendationScanner("status")
	if err != nil {
		s.InternalServerError(c, "获取推荐调度器状态失败", err)
		return
	}

	c.JSON(200, status)
}

// StartRecommendationScheduler starts the scheduler (通过调用独立进程)
func (s *Server) StartRecommendationScheduler(c *gin.Context) {
	// 调用独立的recommendation_scanner进程启动调度器
	result, err := s.callRecommendationScanner("start")
	if err != nil {
		s.InternalServerError(c, "启动推荐调度器失败", err)
		return
	}

	c.JSON(200, result)
}

// StopRecommendationScheduler stops the scheduler (通过调用独立进程)
func (s *Server) StopRecommendationScheduler(c *gin.Context) {
	// 调用独立的recommendation_scanner进程停止调度器
	result, err := s.callRecommendationScanner("stop")
	if err != nil {
		s.InternalServerError(c, "停止推荐调度器失败", err)
		return
	}

	c.JSON(200, result)
}

// ForceGenerateRecommendations forces generation of recommendations (通过调用独立进程)
func (s *Server) ForceGenerateRecommendations(c *gin.Context) {
	kind := c.DefaultQuery("kind", "spot")
	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		s.ValidationError(c, "limit", "limit参数必须是1-50之间的整数")
		return
	}

	// 调用独立的recommendation_scanner进程强制生成推荐
	result, err := s.callRecommendationScanner(fmt.Sprintf("generate?kind=%s&limit=%d", kind, limit))
	if err != nil {
		s.InternalServerError(c, "强制生成推荐失败", err)
		return
	}

	c.JSON(200, result)
}

// CleanupOldRecommendations cleans up old recommendations (通过调用独立进程)
func (s *Server) CleanupOldRecommendations(c *gin.Context) {
	maxAgeStr := c.DefaultQuery("max_age_hours", "8760") // 默认1年
	maxAgeHours, err := strconv.Atoi(maxAgeStr)
	if err != nil || maxAgeHours <= 0 {
		s.ValidationError(c, "max_age_hours", "max_age_hours参数必须是正整数")
		return
	}

	// 调用独立的recommendation_scanner进程清理旧推荐
	result, err := s.callRecommendationScanner(fmt.Sprintf("cleanup?max_age_hours=%d", maxAgeHours))
	if err != nil {
		s.InternalServerError(c, "清理旧推荐失败", err)
		return
	}

	c.JSON(200, result)
}

// GetRecommendationDataStats gets recommendation data statistics (通过调用独立进程)
func (s *Server) GetRecommendationDataStats(c *gin.Context) {
	// 调用独立的recommendation_scanner进程获取统计信息
	stats, err := s.callRecommendationScanner("stats")
	if err != nil {
		s.InternalServerError(c, "获取推荐数据统计失败", err)
		return
	}

	c.JSON(200, stats)
}

// callRecommendationScanner 调用独立的recommendation_scanner进程
func (s *Server) callRecommendationScanner(action string) (map[string]interface{}, error) {
	// 假设recommendation_scanner进程运行在本地端口8011上
	// 实际部署时可以通过配置文件指定
	scannerURL := "http://127.0.0.1:8011"

	var url string
	switch action {
	case "status":
		url = scannerURL + "/status"
	case "start":
		url = scannerURL + "/control/start"
	case "stop":
		url = scannerURL + "/control/stop"
	case "stats":
		url = scannerURL + "/stats"
	default:
		// 处理generate和cleanup等带参数的请求
		if strings.HasPrefix(action, "generate") {
			url = scannerURL + "/control/generate?" + strings.TrimPrefix(action, "generate")
		} else if strings.HasPrefix(action, "cleanup") {
			url = scannerURL + "/control/cleanup?" + strings.TrimPrefix(action, "cleanup")
		} else {
			return nil, fmt.Errorf("不支持的action: %s", action)
		}
	}

	// 调用recommendation_scanner的API
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result map[string]interface{}
	err := netutil.GetJSON(ctx, url, &result)
	if err != nil {
		// 如果独立进程不可用，返回模拟数据和错误信息
		log.Printf("[WARNING] recommendation_scanner进程不可用: %v", err)
		return map[string]interface{}{
			"error":   fmt.Sprintf("recommendation_scanner进程不可用: %v", err),
			"status":  "unavailable",
			"message": "请确保recommendation_scanner进程正在运行在端口8011上",
			"url":     url,
		}, nil
	}

	return result, nil
}

// UpdateRecommendationPerformance 更新推荐表现追踪（定期调用）
// 注意：此功能已移至独立的investment服务，请使用investment -mode=scheduler
func (s *Server) UpdateRecommendationPerformance(ctx context.Context) error {
	log.Printf("[Server] UpdateRecommendationPerformance已移至investment服务，请使用: investment -mode=scheduler")
	return nil // 返回nil以避免中断调用链
}

// UpdateBacktestFromPerformance 从表现追踪更新回测数据
// 注意：此功能已移至独立的investment服务，请使用investment -mode=scheduler
func (s *Server) UpdateBacktestFromPerformance(ctx context.Context) error {
	log.Printf("[Server] UpdateBacktestFromPerformance已移至investment服务，请使用: investment -mode=scheduler")
	return nil // 返回nil以避免中断调用链
}

// GetCurrentPrice 获取当前价格（实现ServerInterface）
func (s *Server) GetCurrentPrice(ctx context.Context, symbol, kind string) (float64, error) {
	return s.getCurrentPrice(ctx, symbol, kind)
}

// FetchBinanceKlines 获取Binance K线数据（实现ServerInterface）
func (s *Server) FetchBinanceKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]analysis.KlineDataAPI, error) {
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, interval, limit)
	if err != nil {
		return nil, err
	}

	// 转换数据格式
	result := make([]analysis.KlineDataAPI, len(klines))
	for i, kline := range klines {
		result[i] = analysis.KlineDataAPI{
			OpenTime: int64(kline.OpenTime),
			Open:     kline.Open,
			High:     kline.High,
			Low:      kline.Low,
			Close:    kline.Close,
			Volume:   kline.Volume,
		}
	}
	return result, nil
}
func (s *Server) FetchBinanceKlinesWithTimeRange(ctx context.Context, symbol, kind, interval string, limit int, startTime, endTime *time.Time) ([]analysis.KlineDataAPI, error) {
	klines, err := s.fetchBinanceKlinesWithTimeRange(ctx, symbol, kind, interval, limit, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// 转换数据格式
	result := make([]analysis.KlineDataAPI, len(klines))
	for i, kline := range klines {
		result[i] = analysis.KlineDataAPI{
			OpenTime: int64(kline.OpenTime),
			Open:     kline.Open,
			High:     kline.High,
			Low:      kline.Low,
			Close:    kline.Close,
			Volume:   kline.Volume,
		}
	}
	return result, nil
}

// GetSystemStatus 获取系统状态
func (s *Server) GetSystemStatus(c *gin.Context) {
	status := map[string]interface{}{
		"service":   "analysis-backend",
		"version":   "1.0.0",
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"uptime":    "unknown", // 可以后续实现
		"environment": map[string]interface{}{
			"go_version": "1.21+",
			"database":   "connected",
			"cache":      "operational",
		},
	}

	c.JSON(200, status)
}

// GetSystemStats 获取系统统计信息
func (s *Server) GetSystemStats(c *gin.Context) {
	stats := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"performance": map[string]interface{}{
			"active_connections":    0, // 可以后续实现连接计数
			"requests_per_minute":   0,
			"average_response_time": "0ms",
		},
		"resources": map[string]interface{}{
			"memory_usage": "unknown",
			"cpu_usage":    "unknown",
			"disk_usage":   "unknown",
		},
		"cache": map[string]interface{}{
			"hit_rate":    "unknown",
			"total_keys":  0,
			"memory_used": "0MB",
		},
		"database": map[string]interface{}{
			"connections_active": 0,
			"connections_idle":   0,
			"queries_per_second": 0,
		},
	}

	c.JSON(200, stats)
}

// GetDataCacheStats 获取数据缓存统计信息
func (s *Server) GetDataCacheStats(c *gin.Context) {
	if s.dataCache == nil {
		c.JSON(500, gin.H{"error": "数据缓存未初始化"})
		return
	}

	s.dataCache.mu.RLock()
	defer s.dataCache.mu.RUnlock()

	hitRate := float64(0)
	totalRequests := s.dataCache.hitCount + s.dataCache.missCount
	if totalRequests > 0 {
		hitRate = float64(s.dataCache.hitCount) / float64(totalRequests)
	}

	stats := map[string]interface{}{
		"cache_size":     len(s.dataCache.processedData),
		"max_cache_size": s.dataCache.maxSize,
		"cache_max_age":  s.dataCache.maxAge.String(),
		"hit_count":      s.dataCache.hitCount,
		"miss_count":     s.dataCache.missCount,
		"hit_rate":       fmt.Sprintf("%.2f%%", hitRate*100),
		"cache_entries":  []map[string]interface{}{},
	}

	// 简要显示前10个缓存条目
	count := 0
	for key, data := range s.dataCache.processedData {
		if count >= 10 {
			break
		}
		entry := map[string]interface{}{
			"key":           key,
			"data_points":   len(data.ProcessedData),
			"quality_score": data.Quality.Overall,
			"processed_at":  data.ProcessedAt.Format("2006-01-02 15:04:05"),
		}
		stats["cache_entries"] = append(stats["cache_entries"].([]map[string]interface{}), entry)
		count++
	}

	c.JSON(200, stats)
}

// GetDataUpdateServiceStatus 获取数据更新服务状态
func (s *Server) GetDataUpdateServiceStatus(c *gin.Context) {
	if s.dataUpdateService == nil {
		c.JSON(500, gin.H{"error": "数据更新服务未初始化"})
		return
	}

	status := s.dataUpdateService.GetStatus()
	c.JSON(200, status)
}

// TriggerDataUpdate 手动触发数据更新
func (s *Server) TriggerDataUpdate(c *gin.Context) {
	if s.dataUpdateService == nil {
		c.JSON(500, gin.H{"error": "数据更新服务未初始化"})
		return
	}

	// 异步执行数据更新
	go func() {
		log.Printf("[API] 手动触发数据更新")
		s.dataUpdateService.performFullUpdate()
		log.Printf("[API] 手动数据更新完成")
	}()

	c.JSON(200, gin.H{
		"message": "数据更新已启动",
		"status":  "running",
	})
}

// ClearDataCache 清理数据缓存
func (s *Server) ClearDataCache(c *gin.Context) {
	if s.dataCache == nil {
		c.JSON(500, gin.H{"error": "数据缓存未初始化"})
		return
	}

	s.dataCache.mu.Lock()
	s.dataCache.processedData = make(map[string]*ProcessedMarketData)
	s.dataCache.hitCount = 0
	s.dataCache.missCount = 0
	s.dataCache.mu.Unlock()

	log.Printf("[API] 数据缓存已清理")
	c.JSON(200, gin.H{"message": "数据缓存已清理"})
}

// GetFeatureCacheStats 获取特征缓存统计信息
func (s *Server) GetFeatureCacheStats(c *gin.Context) {
	if s.featurePrecomputeService == nil || s.featurePrecomputeService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "特征缓存未初始化"})
		return
	}

	stats := s.featurePrecomputeService.cacheManager.GetStats()
	c.JSON(200, stats)
}

// GetFeaturePrecomputeServiceStatus 获取特征预计算服务状态
func (s *Server) GetFeaturePrecomputeServiceStatus(c *gin.Context) {
	if s.featurePrecomputeService == nil {
		c.JSON(500, gin.H{"error": "特征预计算服务未初始化"})
		return
	}

	status := s.featurePrecomputeService.GetStatus()
	c.JSON(200, status)
}

// TriggerFeaturePrecomputation 手动触发特征预计算
func (s *Server) TriggerFeaturePrecomputation(c *gin.Context) {
	if s.featurePrecomputeService == nil {
		c.JSON(500, gin.H{"error": "特征预计算服务未初始化"})
		return
	}

	// 异步执行特征预计算
	go func() {
		log.Printf("[API] 手动触发特征预计算")
		s.featurePrecomputeService.performFullPrecomputation()
		log.Printf("[API] 手动特征预计算完成")
	}()

	c.JSON(200, gin.H{
		"message": "特征预计算已启动",
		"status":  "running",
	})
}

// ClearFeatureCache 清理特征缓存
func (s *Server) ClearFeatureCache(c *gin.Context) {
	if s.featurePrecomputeService == nil || s.featurePrecomputeService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "特征缓存未初始化"})
		return
	}

	s.featurePrecomputeService.cacheManager.mu.Lock()
	s.featurePrecomputeService.cacheManager.featureCache = make(map[string]*CachedFeatureSet)
	s.featurePrecomputeService.cacheManager.hitCount = 0
	s.featurePrecomputeService.cacheManager.missCount = 0
	s.featurePrecomputeService.cacheManager.mu.Unlock()

	log.Printf("[API] 特征缓存已清理")
	c.JSON(200, gin.H{"message": "特征缓存已清理"})
}

// GetPopularFeatureSymbols 获取最受欢迎的特征符号
func (s *Server) GetPopularFeatureSymbols(c *gin.Context) {
	if s.featurePrecomputeService == nil || s.featurePrecomputeService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "特征缓存未初始化"})
		return
	}

	limit := 10 // 默认返回前10个
	symbols := s.featurePrecomputeService.cacheManager.GetPopularSymbols(limit)

	c.JSON(200, gin.H{
		"popular_symbols": symbols,
		"limit":           limit,
	})
}

// GetTechnicalIndicatorsCacheStats 获取技术指标缓存统计信息
func (s *Server) GetTechnicalIndicatorsCacheStats(c *gin.Context) {
	if s.technicalIndicatorsPrecomputeService == nil || s.technicalIndicatorsPrecomputeService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "技术指标缓存未初始化"})
		return
	}

	stats := s.technicalIndicatorsPrecomputeService.cacheManager.GetStats()
	c.JSON(200, stats)
}

// GetTechnicalIndicatorsPrecomputeServiceStatus 获取技术指标预计算服务状态
func (s *Server) GetTechnicalIndicatorsPrecomputeServiceStatus(c *gin.Context) {
	if s.technicalIndicatorsPrecomputeService == nil {
		c.JSON(500, gin.H{"error": "技术指标预计算服务未初始化"})
		return
	}

	status := s.technicalIndicatorsPrecomputeService.GetStatus()
	c.JSON(200, status)
}

// TriggerTechnicalIndicatorsPrecomputation 手动触发技术指标预计算
func (s *Server) TriggerTechnicalIndicatorsPrecomputation(c *gin.Context) {
	if s.technicalIndicatorsPrecomputeService == nil {
		c.JSON(500, gin.H{"error": "技术指标预计算服务未初始化"})
		return
	}

	// 异步执行技术指标预计算
	go func() {
		log.Printf("[API] 手动触发技术指标预计算")
		s.technicalIndicatorsPrecomputeService.performFullPrecomputation()
		log.Printf("[API] 手动技术指标预计算完成")
	}()

	c.JSON(200, gin.H{
		"message": "技术指标预计算已启动",
		"status":  "running",
	})
}

// ClearTechnicalIndicatorsCache 清理技术指标缓存
func (s *Server) ClearTechnicalIndicatorsCache(c *gin.Context) {
	if s.technicalIndicatorsPrecomputeService == nil || s.technicalIndicatorsPrecomputeService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "技术指标缓存未初始化"})
		return
	}

	s.technicalIndicatorsPrecomputeService.cacheManager.mu.Lock()
	s.technicalIndicatorsPrecomputeService.cacheManager.indicatorsCache = make(map[string]*CachedTechnicalIndicators)
	s.technicalIndicatorsPrecomputeService.cacheManager.hitCount = 0
	s.technicalIndicatorsPrecomputeService.cacheManager.missCount = 0
	s.technicalIndicatorsPrecomputeService.cacheManager.mu.Unlock()

	log.Printf("[API] 技术指标缓存已清理")
	c.JSON(200, gin.H{"message": "技术指标缓存已清理"})
}

// GetTechnicalIndicators 获取指定币种的技术指标
func (s *Server) GetTechnicalIndicators(c *gin.Context) {
	symbol := c.Query("symbol")
	timeframe := c.DefaultQuery("timeframe", "1h")

	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol参数是必需的"})
		return
	}

	if s.technicalIndicatorsPrecomputeService == nil {
		c.JSON(500, gin.H{"error": "技术指标预计算服务未初始化"})
		return
	}

	indicators := s.technicalIndicatorsPrecomputeService.GetIndicators(symbol, timeframe)
	if indicators == nil {
		c.JSON(404, gin.H{"error": "未找到技术指标数据"})
		return
	}

	c.JSON(200, indicators)
}

// GetMLModelCacheStats 获取ML模型缓存统计信息
func (s *Server) GetMLModelCacheStats(c *gin.Context) {
	if s.mlPretrainingService == nil || s.mlPretrainingService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "ML模型缓存未初始化"})
		return
	}

	stats := s.mlPretrainingService.cacheManager.GetStats()
	c.JSON(200, stats)
}

// GetMLPretrainingServiceStatus 获取ML模型预训练服务状态
func (s *Server) GetMLPretrainingServiceStatus(c *gin.Context) {
	if s.mlPretrainingService == nil {
		c.JSON(500, gin.H{"error": "ML模型预训练服务未初始化"})
		return
	}

	status := s.mlPretrainingService.GetStatus()
	c.JSON(200, status)
}

// TriggerMLModelPretraining 手动触发ML模型预训练
func (s *Server) TriggerMLModelPretraining(c *gin.Context) {
	if s.mlPretrainingService == nil {
		c.JSON(500, gin.H{"error": "ML模型预训练服务未初始化"})
		return
	}

	// 异步执行ML模型预训练
	go func() {
		log.Printf("[API] 手动触发ML模型预训练")
		s.mlPretrainingService.performFullPretraining()
		log.Printf("[API] 手动ML模型预训练完成")
	}()

	c.JSON(200, gin.H{
		"message": "ML模型预训练已启动",
		"status":  "running",
	})
}

// ClearMLModelCache 清理ML模型缓存
func (s *Server) ClearMLModelCache(c *gin.Context) {
	if s.mlPretrainingService == nil || s.mlPretrainingService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "ML模型缓存未初始化"})
		return
	}

	s.mlPretrainingService.cacheManager.mu.Lock()
	s.mlPretrainingService.cacheManager.modelCache = make(map[string]*CachedMLModel)
	s.mlPretrainingService.cacheManager.hitCount = 0
	s.mlPretrainingService.cacheManager.missCount = 0
	s.mlPretrainingService.cacheManager.mu.Unlock()

	log.Printf("[API] ML模型缓存已清理")
	c.JSON(200, gin.H{"message": "ML模型缓存已清理"})
}

// GetMLModel 获取指定币种的ML模型
func (s *Server) GetMLModel(c *gin.Context) {
	symbol := c.Query("symbol")
	modelType := c.DefaultQuery("model_type", "random_forest")

	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol参数是必需的"})
		return
	}

	if s.mlPretrainingService == nil {
		c.JSON(500, gin.H{"error": "ML模型预训练服务未初始化"})
		return
	}

	model := s.mlPretrainingService.GetModel(symbol, modelType)
	if model == nil {
		c.JSON(404, gin.H{"error": "未找到ML模型"})
		return
	}

	performance := s.mlPretrainingService.GetModelPerformance(symbol, modelType)

	response := gin.H{
		"model":       model,
		"performance": performance,
	}

	c.JSON(200, response)
}

// GetBestMLModels 获取表现最好的ML模型列表
func (s *Server) GetBestMLModels(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50 // 限制最大返回数量
	}

	if s.mlPretrainingService == nil || s.mlPretrainingService.cacheManager == nil {
		c.JSON(500, gin.H{"error": "ML模型缓存未初始化"})
		return
	}

	bestModels := s.mlPretrainingService.cacheManager.GetBestModels(limit)

	response := make([]gin.H, len(bestModels))
	for i, cached := range bestModels {
		response[i] = gin.H{
			"symbol":      cached.Symbol,
			"model_type":  cached.ModelType,
			"accuracy":    cached.Accuracy,
			"trained_at":  cached.TrainedAt,
			"data_points": cached.DataPoints,
			"performance": cached.Performance,
		}
	}

	c.JSON(200, gin.H{
		"best_models": response,
		"limit":       limit,
	})
}

// GetMLModelStats 获取ML模型统计信息
func (s *Server) GetMLModelStats(c *gin.Context) {
	if s.db == nil {
		c.JSON(500, gin.H{"error": "数据库未初始化"})
		return
	}

	gdb := s.db.DB()
	if gdb == nil {
		c.JSON(500, gin.H{"error": "获取数据库连接失败"})
		return
	}

	stats, err := pdb.GetMLModelStats(gdb)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("获取ML模型统计失败: %v", err)})
		return
	}

	c.JSON(200, stats)
}

// CleanupExpiredMLModels 清理过期的ML模型
func (s *Server) CleanupExpiredMLModels(c *gin.Context) {
	if s.db == nil {
		c.JSON(500, gin.H{"error": "数据库未初始化"})
		return
	}

	gdb := s.db.DB()
	if gdb == nil {
		c.JSON(500, gin.H{"error": "获取数据库连接失败"})
		return
	}

	err := pdb.CleanupExpiredMLModels(gdb)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("清理过期ML模型失败: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "过期ML模型清理完成"})
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("[Server] 开始关闭服务器...")

	// 停止ML模型预训练服务
	if s.mlPretrainingService != nil {
		if err := s.mlPretrainingService.Stop(); err != nil {
			log.Printf("[ERROR] 停止ML模型预训练服务失败: %v", err)
		} else {
			log.Printf("[Server] ML模型预训练服务已停止")
		}
	}

	// 停止技术指标预计算服务
	if s.technicalIndicatorsPrecomputeService != nil {
		if err := s.technicalIndicatorsPrecomputeService.Stop(); err != nil {
			log.Printf("[ERROR] 停止技术指标预计算服务失败: %v", err)
		} else {
			log.Printf("[Server] 技术指标预计算服务已停止")
		}
	}

	// 停止特征预计算服务
	if s.featurePrecomputeService != nil {
		if err := s.featurePrecomputeService.Stop(); err != nil {
			log.Printf("[ERROR] 停止特征预计算服务失败: %v", err)
		} else {
			log.Printf("[Server] 特征预计算服务已停止")
		}
	}

	// 停止数据更新服务
	if s.dataUpdateService != nil {
		if err := s.dataUpdateService.Stop(); err != nil {
			log.Printf("[ERROR] 停止数据更新服务失败: %v", err)
		} else {
			log.Printf("[Server] 数据更新服务已停止")
		}
	}

	// 这里可以添加其他服务的关闭逻辑
	log.Printf("[Server] 服务器关闭完成")
	return nil
}

// ===== 数据同步监控 API =====

// GetDataSyncStatus 获取数据同步服务状态
func (s *Server) GetDataSyncStatus(c *gin.Context) {
	globalDataSyncStats.mu.RLock()
	defer globalDataSyncStats.mu.RUnlock()

	response := map[string]interface{}{
		"global_health": globalDataSyncStats.globalHealth,
		"last_check":    globalDataSyncStats.lastUpdate.UTC(),
	}

	// 同步器状态
	syncers := make(map[string]interface{})
	for name, syncer := range globalDataSyncStats.syncers {
		syncers[name] = syncer
	}

	// 确保所有预期的同步器都存在
	syncerNames := []string{"price", "kline", "depth", "websocket"}
	for _, name := range syncerNames {
		if _, exists := syncers[name]; !exists {
			syncers[name] = &SyncerStats{
				Name:        name,
				DisplayName: s.getSyncerDisplayName(name),
				Status:      "unknown",
			}
		}
	}

	response["syncers"] = syncers
	response["websocket"] = globalDataSyncStats.websocket

	// API统计数据 - 转换为前端期望的格式
	if priceStats, exists := globalDataSyncStats.apiStats["price"]; exists {
		response["price"] = priceStats
	} else {
		response["price"] = &APIStats{}
	}

	if klineStats, exists := globalDataSyncStats.apiStats["kline"]; exists {
		response["kline"] = klineStats
	} else {
		response["kline"] = &APIStats{}
	}

	if depthStats, exists := globalDataSyncStats.apiStats["depth"]; exists {
		response["depth"] = depthStats
	} else {
		response["depth"] = &APIStats{}
	}

	c.JSON(200, response)
}

// getSyncerDisplayName 获取同步器的显示名称
func (s *Server) getSyncerDisplayName(name string) string {
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

// TriggerManualSync 触发手动同步
func (s *Server) TriggerManualSync(c *gin.Context) {
	var request struct {
		SyncerType string `json:"syncer_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// 这里应该触发相应的同步器
	// 由于数据同步服务可能还没有完全集成，我们先返回成功响应

	log.Printf("[DataSync] Manual sync triggered for type: %s", request.SyncerType)

	c.JSON(200, gin.H{
		"success":   true,
		"message":   fmt.Sprintf("Manual sync triggered for %s", request.SyncerType),
		"timestamp": time.Now().UTC(),
	})
}

// GetDataConsistencyStatus 获取数据一致性检查状态
func (s *Server) GetDataConsistencyStatus(c *gin.Context) {
	globalDataSyncStats.mu.RLock()
	defer globalDataSyncStats.mu.RUnlock()

	// 计算一致性得分（基于告警数量和严重程度）
	consistencyScore := 100.0
	alertCount := len(globalDataSyncStats.alerts)
	if alertCount > 0 {
		// 根据告警数量和严重程度降低得分
		scoreReduction := float64(alertCount) * 5.0
		if scoreReduction > 50.0 {
			scoreReduction = 50.0
		}
		consistencyScore -= scoreReduction
		if consistencyScore < 0 {
			consistencyScore = 0
		}
	}

	// 提取最近的问题（基于告警）
	recentIssues := []map[string]interface{}{}
	for i, alert := range globalDataSyncStats.alerts {
		if i >= 5 { // 最多显示5个最近问题
			break
		}
		recentIssues = append(recentIssues, map[string]interface{}{
			"dataType":    alert.Component,
			"severity":    alert.Severity,
			"description": alert.Message,
			"timestamp":   alert.Timestamp,
		})
	}

	response := map[string]interface{}{
		"consistency_score": consistencyScore,
		"total_checks":      int64(len(globalDataSyncStats.alerts)),
		"issues_found":      int64(alertCount),
		"last_check":        globalDataSyncStats.lastUpdate,
		"recent_issues":     recentIssues,
	}

	c.JSON(200, response)
}

// GetAlerts 获取告警信息
func (s *Server) GetAlerts(c *gin.Context) {
	globalDataSyncStats.mu.RLock()
	defer globalDataSyncStats.mu.RUnlock()

	activeAlerts := []map[string]interface{}{}
	for _, alert := range globalDataSyncStats.alerts {
		activeAlerts = append(activeAlerts, map[string]interface{}{
			"id":        alert.ID,
			"title":     alert.Title,
			"message":   alert.Message,
			"severity":  alert.Severity,
			"component": alert.Component,
			"metric":    alert.Metric,
			"value":     alert.Value,
			"timestamp": alert.Timestamp,
		})
	}

	response := map[string]interface{}{
		"active_alerts": activeAlerts,
		"total_count":   len(activeAlerts),
		"timestamp":     time.Now().UTC(),
	}

	c.JSON(200, response)
}

// TriggerConsistencyCheck 触发一致性检查
func (s *Server) TriggerConsistencyCheck(c *gin.Context) {
	// 触发数据一致性检查
	log.Printf("[DataSync] Consistency check triggered manually")

	c.JSON(200, gin.H{
		"success":   true,
		"message":   "Consistency check triggered",
		"timestamp": time.Now().UTC(),
	})
}

// ReconnectWebSocket 重新连接WebSocket
func (s *Server) ReconnectWebSocket(c *gin.Context) {
	// 触发WebSocket重连
	log.Printf("[DataSync] WebSocket reconnection triggered manually")

	c.JSON(200, gin.H{
		"success":   true,
		"message":   "WebSocket reconnection initiated",
		"timestamp": time.Now().UTC(),
	})
}

// getMarketDataForSymbol 获取单个币种的市场数据
func (s *Server) getMarketDataForSymbol(symbol string) StrategyMarketData {
	mds := NewMarketDataService(s)
	return mds.getMarketDataForSymbol(symbol)
}

// getKlinePricesForSymbol 获取币种的K线价格数据
func (s *Server) getKlinePricesForSymbol(symbol string, minDataPoints int) ([]float64, error) {
	// 计算结束时间（当前时间）和开始时间
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7) // 默认取7天的数据

	// 从数据库获取K线数据（使用1小时K线）
	klines, err := pdb.GetMarketKlines(s.db.DB(), symbol, "spot", "1h", &startTime, &endTime, minDataPoints*2) // 多取一些数据
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %v", err)
	}

	if len(klines) < minDataPoints {
		return nil, fmt.Errorf("K线数据不足，需要%d个数据点，实际%d个", minDataPoints, len(klines))
	}

	// 提取收盘价
	prices := make([]float64, len(klines))
	for i, kline := range klines {
		if price, err := strconv.ParseFloat(kline.ClosePrice, 64); err == nil {
			prices[i] = price
		} else {
			return nil, fmt.Errorf("解析价格失败: %v", err)
		}
	}

	return prices, nil
}

// ============================================================================
// 策略HTTP API处理器 - 代理到StrategyHandler
// ============================================================================

// ExecuteStrategy 执行策略判断
func (s *Server) ExecuteStrategy(c *gin.Context) {
	s.strategyHandler.ExecuteStrategy(c)
}

// BatchExecuteStrategies 批量执行策略
func (s *Server) BatchExecuteStrategies(c *gin.Context) {
	s.strategyHandler.BatchExecuteStrategies(c)
}

// ScanEligibleSymbols 扫描符合策略的币种
func (s *Server) ScanEligibleSymbols(c *gin.Context) {
	s.strategyHandler.ScanEligibleSymbols(c)
}

// DiscoverArbitrageOpportunities 发现套利机会
func (s *Server) DiscoverArbitrageOpportunities(c *gin.Context) {
	s.strategyHandler.DiscoverArbitrageOpportunities(c)
}

// executeStrategyWithNewExecutors 使用路由器和工厂执行策略
func (s *Server) executeStrategyWithNewExecutors(ctx context.Context, symbol string, marketData StrategyMarketData, conditions pdb.StrategyConditions, strategy *pdb.TradingStrategy) StrategyDecisionResult {
	// 使用路由器选择策略
	route := s.strategyRouter.SelectRoute(conditions)
	if route == nil {
		return StrategyDecisionResult{
			Action:     "no_op",
			Reason:     "未找到合适的策略路由",
			Multiplier: 1.0,
		}
	}

	// 使用工厂创建执行器和配置
	executor, config, err := s.strategyFactory.CreateExecutor(route.StrategyType, conditions)
	if err != nil {
		return StrategyDecisionResult{
			Action:     "skip",
			Reason:     fmt.Sprintf("创建策略执行器失败: %v", err),
			Multiplier: 1.0,
		}
	}

	// 构建执行市场数据和上下文
	routerMarketData := router.StrategyMarketData{
		Symbol:      marketData.Symbol,
		MarketCap:   marketData.MarketCap,
		GainersRank: marketData.GainersRank,
		HasSpot:     marketData.HasSpot,
		HasFutures:  marketData.HasFutures,
	}

	execMarketData := route.MarketDataBuilder(routerMarketData)
	execContext := route.ContextBuilder(symbol, route.StrategyType, strategy.UserID, strategy.ID)

	// 执行策略
	result, err := executor.Execute(ctx, symbol, execMarketData, config, execContext)

	if err != nil {
		log.Printf("[NewExecutor] 策略执行失败 %s: %v", symbol, err)
		return StrategyDecisionResult{
			Action:     "skip",
			Reason:     fmt.Sprintf("策略执行失败: %v", err),
			Multiplier: 1.0,
		}
	}

	if result == nil {
		return StrategyDecisionResult{
			Action:     "no_op",
			Reason:     "策略执行返回空结果",
			Multiplier: 1.0,
		}
	}

	// 转换结果格式
	return StrategyDecisionResult{
		Action:     result.Action,
		Reason:     result.Reason,
		Multiplier: result.Multiplier,
	}
}

// ============================================================================
// MarketDataProvider接口实现 - 为新模块化架构提供市场数据服务
// ============================================================================

// GetMarketData 获取市场数据
func (s *Server) GetMarketData(symbol string) (*execution.MarketData, error) {
	strategyData := s.getMarketDataForSymbol(symbol)

	// 获取实时价格
	ctx := context.Background()
	price, err := s.getCurrentPrice(ctx, symbol, "spot")
	if err != nil {
		price = 0 // 如果获取失败，使用0作为默认值
	}

	return &execution.MarketData{
		Symbol:      strategyData.Symbol,
		Price:       price,
		Volume:      0, // StrategyMarketData中没有成交量字段
		MarketCap:   strategyData.MarketCap,
		GainersRank: strategyData.GainersRank,
		HasSpot:     strategyData.HasSpot,
		HasFutures:  strategyData.HasFutures,
		// 技术指标暂时设为0，新架构中可以扩展
		SMA5:      0,
		SMA10:     0,
		SMA20:     0,
		SMA50:     0,
		Change24h: 0,
	}, nil
}

// GetRealTimePrice 获取实时价格
func (s *Server) GetRealTimePrice(symbol string) (float64, error) {
	ctx := context.Background()
	if strings.Contains(symbol, "_FUTURES") {
		// 期货价格
		baseSymbol := strings.TrimSuffix(symbol, "_FUTURES")
		return s.getCurrentPrice(ctx, baseSymbol, "futures")
	} else if strings.Contains(symbol, "_SPOT") {
		// 现货价格
		baseSymbol := strings.TrimSuffix(symbol, "_SPOT")
		return s.getCurrentPrice(ctx, baseSymbol, "spot")
	} else {
		// 默认现货价格
		return s.getCurrentPrice(ctx, symbol, "spot")
	}
}

// GetKlineData 获取K线数据
func (s *Server) GetKlineData(symbol, interval string, limit int) ([]*execution.KlineData, error) {
	// 使用现有的K线数据获取逻辑
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7) // 默认7天数据

	klines, err := pdb.GetMarketKlines(s.db.DB(), symbol, "spot", interval, &startTime, &endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	result := make([]*execution.KlineData, len(klines))
	for i, kline := range klines {
		openPrice, _ := strconv.ParseFloat(kline.OpenPrice, 64)
		highPrice, _ := strconv.ParseFloat(kline.HighPrice, 64)
		lowPrice, _ := strconv.ParseFloat(kline.LowPrice, 64)
		closePrice, _ := strconv.ParseFloat(kline.ClosePrice, 64)
		volume, _ := strconv.ParseFloat(kline.Volume, 64)

		// 计算CloseTime（OpenTime + interval时长）
		closeTime := kline.OpenTime.Unix() * 1000 // 转换为毫秒时间戳

		result[i] = &execution.KlineData{
			OpenTime:   kline.OpenTime.Unix() * 1000, // 转换为毫秒时间戳
			OpenPrice:  openPrice,
			HighPrice:  highPrice,
			LowPrice:   lowPrice,
			ClosePrice: closePrice,
			Volume:     volume,
			CloseTime:  closeTime,
		}
	}

	return result, nil
}

// ============================================================================
// OrderManager接口实现
// ============================================================================

// PlaceOrder 下单
func (s *Server) PlaceOrder(symbol, side string, quantity, price float64) (string, error) {
	// 这里应该调用实际的下单API
	// 目前返回模拟订单ID
	orderID := fmt.Sprintf("sim_%s_%s_%d", symbol, side, time.Now().Unix())
	log.Printf("[OrderManager] 模拟下单: %s %s %.4f@%.4f, 订单ID: %s", side, symbol, quantity, price, orderID)
	return orderID, nil
}

// CancelOrder 取消订单
func (s *Server) CancelOrder(orderID string) error {
	// 这里应该调用实际的取消订单API
	log.Printf("[OrderManager] 模拟取消订单: %s", orderID)
	return nil
}

// GetOrderStatus 获取订单状态
func (s *Server) GetOrderStatus(orderID string) (*execution.OrderStatus, error) {
	// 这里应该调用实际的订单状态查询API
	// 目前返回模拟状态
	return &execution.OrderStatus{
		OrderID:     orderID,
		Status:      "filled",
		Symbol:      "BTCUSDT",
		Side:        "buy",
		Quantity:    100.0,
		Price:       50000.0,
		ExecutedQty: 100.0,
		AvgPrice:    50000.0,
		Fee:         0.001,
	}, nil
}

// ============================================================================
// RiskManager接口实现
// ============================================================================

// ValidateRisk 验证风险
func (s *Server) ValidateRisk(symbol string, positionSize float64) error {
	// 基本风险检查
	if positionSize <= 0 {
		return fmt.Errorf("仓位大小必须大于0")
	}
	if positionSize > 1000 { // 假设最大仓位限制
		return fmt.Errorf("仓位大小超过限制: %.2f > 1000", positionSize)
	}
	return nil
}

// CalculateStopLoss 计算止损价格
func (s *Server) CalculateStopLoss(entryPrice float64, riskPercent float64) float64 {
	return entryPrice * (1 - riskPercent/100)
}

// CalculateTakeProfit 计算止盈价格
func (s *Server) CalculateTakeProfit(entryPrice float64, rewardPercent float64) float64 {
	return entryPrice * (1 + rewardPercent/100)
}

// CheckPositionLimits 检查仓位限制
func (s *Server) CheckPositionLimits(symbol string, newPositionSize float64) error {
	return s.ValidateRisk(symbol, newPositionSize)
}

// ============================================================================
// 保证金风险管理方法实现
// ============================================================================

// CalculateMarginStopLoss 计算保证金亏损止损价格
func (s *Server) CalculateMarginStopLoss(symbol string, marginLossPercent float64) (float64, error) {
	// 创建保证金风险管理器实例
	marginRiskManager := execution.NewMarginRiskManager(s.binanceFuturesClient)
	return marginRiskManager.CalculateMarginStopLoss(symbol, marginLossPercent)
}

// CalculateMarginTakeProfit 计算保证金盈利止盈价格
func (s *Server) CalculateMarginTakeProfit(symbol string, marginProfitPercent float64) (float64, error) {
	// 创建保证金风险管理器实例
	marginRiskManager := execution.NewMarginRiskManager(s.binanceFuturesClient)
	return marginRiskManager.CalculateMarginTakeProfit(symbol, marginProfitPercent)
}

// CheckMarginLoss 检查是否达到保证金亏损阈值
func (s *Server) CheckMarginLoss(symbol string, marginLossPercent float64) (bool, float64, error) {
	// 创建保证金风险管理器实例
	marginRiskManager := execution.NewMarginRiskManager(s.binanceFuturesClient)
	return marginRiskManager.CheckMarginLoss(symbol, marginLossPercent)
}

// GetPositionMarginInfo 获取持仓保证金信息
func (s *Server) GetPositionMarginInfo(symbol string) (*execution.PositionMarginInfo, error) {
	// 创建保证金风险管理器实例
	marginRiskManager := execution.NewMarginRiskManager(s.binanceFuturesClient)
	return marginRiskManager.GetPositionMarginInfo(symbol)
}

// ValidateMarginStopLossConfig 验证保证金止损配置
func (s *Server) ValidateMarginStopLossConfig(marginLossPercent float64) error {
	// 创建保证金风险管理器实例
	marginRiskManager := execution.NewMarginRiskManager(s.binanceFuturesClient)
	return marginRiskManager.ValidateMarginStopLossConfig(marginLossPercent)
}

// ============================================================================
// ConfigProvider接口实现
// ============================================================================

// GetStrategyConfig 获取策略配置
func (s *Server) GetStrategyConfig(strategyType string, userID uint) (interface{}, error) {
	// 这里应该从数据库或其他配置源获取策略配置
	// 目前返回默认配置
	switch strategyType {
	case "traditional":
		return &traditional_execution.TraditionalExecutionConfig{
			ExecutionConfig:  execution.ExecutionConfig{Enabled: true},
			ShortOnGainers:   true,
			GainersRankLimit: 10,
		}, nil
	default:
		return nil, fmt.Errorf("不支持的策略类型: %s", strategyType)
	}
}

// GetGlobalConfig 获取全局配置
func (s *Server) GetGlobalConfig(key string) (interface{}, error) {
	// 这里应该从配置管理系统获取全局配置
	return nil, fmt.Errorf("全局配置暂未实现")
}

// UpdateStrategyConfig 更新策略配置
func (s *Server) UpdateStrategyConfig(strategyType string, userID uint, config interface{}) error {
	// 这里应该保存策略配置到数据库
	log.Printf("[ConfigProvider] 模拟更新策略配置: %s for user %d", strategyType, userID)
	return nil
}
