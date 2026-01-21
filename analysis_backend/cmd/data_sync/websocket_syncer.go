package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	pdb "analysis/internal/db"
	"analysis/internal/server"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// WebSocketConnection WebSocket连接封装
type WebSocketConnection struct {
	conn       *websocket.Conn
	connType   string   // "spot" or "futures"
	symbols    []string // 此连接订阅的交易对
	lastActive time.Time
	isHealthy  bool
	mu         sync.RWMutex
}

// WebSocketConnectionPool 连接池
type WebSocketConnectionPool struct {
	connections    []*WebSocketConnection
	maxConnPerType int // 每种类型的最大连接数
	mu             sync.RWMutex
}

// WebSocketSyncer WebSocket数据同步器
type WebSocketSyncer struct {
	db        *gorm.DB
	config    *DataSyncConfig
	isRunning bool
	mu        sync.RWMutex

	// 连接池管理
	spotPool    *WebSocketConnectionPool // 现货连接池
	futuresPool *WebSocketConnectionPool // 期货连接池

	// 数据缓存
	priceCache   map[string]PriceData
	futuresCache map[string]FuturesData
	klineCache   map[string]KlineData // 实时K线数据缓存
	depthCache   map[string]DepthData // 深度数据缓存
	tradeCache   []TradeData          // 交易数据缓存（使用切片，因为交易是顺序的）
	cacheMu      sync.RWMutex

	// 订阅的交易对
	subscribedSymbols []string

	// 重连保护
	lastReconnectTime time.Time
	reconnectCooldown time.Duration

	// 性能监控
	stats struct {
		mu                       sync.RWMutex
		messagesReceived         int64
		messagesProcessed        int64
		totalSpotPriceUpdates    int64
		totalFuturesPriceUpdates int64
		lastMessageTime          time.Time
		reconnectCount           int64
		cacheHitRate             float64
		averageProcessingTime    time.Duration
		healthCheckFailures      int64
	}
}

// PriceData 价格数据
type PriceData struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"` // 与数据库Price字段保持一致，使用字符串
	Time   int64  `json:"time"`
}

// FuturesData 期货数据
type FuturesData struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"` // 与数据库Price字段保持一致，使用字符串
	Time   int64  `json:"time"`
}

// KlineData K线数据
type KlineData struct {
	Symbol      string `json:"symbol"`
	Interval    string `json:"interval"`   // 时间间隔，如 "1m", "5m", "1h"
	OpenTime    int64  `json:"open_time"`  // K线开盘时间
	CloseTime   int64  `json:"close_time"` // K线收盘时间
	OpenPrice   string `json:"open_price"`
	HighPrice   string `json:"high_price"`
	LowPrice    string `json:"low_price"`
	ClosePrice  string `json:"close_price"`
	Volume      string `json:"volume"`
	QuoteVolume string `json:"quote_volume,omitempty"`
	TradeCount  int    `json:"trade_count,omitempty"`
}

// DepthData 深度数据
type DepthData struct {
	Symbol       string     `json:"symbol"`
	LastUpdateID int64      `json:"last_update_id"`
	Bids         [][]string `json:"bids"` // [[price, quantity], ...]
	Asks         [][]string `json:"asks"` // [[price, quantity], ...]
	Timestamp    int64      `json:"timestamp"`
}

// TradeData 交易数据
type TradeData struct {
	Symbol       string `json:"symbol"`
	TradeID      int64  `json:"trade_id"`
	Price        string `json:"price"`
	Quantity     string `json:"quantity"`
	TradeTime    int64  `json:"trade_time"`
	IsBuyerMaker bool   `json:"is_buyer_maker"` // true表示买方是挂单方
}

// NewWebSocketSyncer 创建WebSocket同步器
func NewWebSocketSyncer(db *gorm.DB, config *DataSyncConfig) *WebSocketSyncer {
	// 默认每个类型最多10个连接，支持分布式订阅
	maxConnPerType := 10
	if config.WebSocketMaxSymbols > 100 {
		// 如果订阅的交易对很多，进一步增加连接数
		maxConnPerType = 20
	}

	return &WebSocketSyncer{
		db:                db,
		config:            config,
		spotPool:          NewWebSocketConnectionPool(maxConnPerType),
		futuresPool:       NewWebSocketConnectionPool(maxConnPerType),
		priceCache:        make(map[string]PriceData),
		futuresCache:      make(map[string]FuturesData),
		klineCache:        make(map[string]KlineData),
		depthCache:        make(map[string]DepthData),
		tradeCache:        make([]TradeData, 0),
		reconnectCooldown: 5 * time.Second, // 5秒重连冷却时间
		isRunning:         false,
	}
}

// Start 启动WebSocket连接
func (s *WebSocketSyncer) Start(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	log.Printf("[WebSocketSyncer] Starting WebSocket connection...")

	// 连接到Binance WebSocket
	if err := s.connect(); err != nil {
		log.Printf("[WebSocketSyncer] Failed to connect: %v", err)
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		return
	}

	// 订阅数据流
	if err := s.subscribeToStreams(); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe: %v", err)
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		return
	}

	// 启动数据接收循环
	go s.receiveLoop(ctx)

	// 启动定期批量保存
	go s.batchSaveLoop(ctx, interval)

	// 启动健康检查和自动调整
	go s.healthCheckLoop(ctx)
}

// Stop 停止WebSocket连接
func (s *WebSocketSyncer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.isRunning = false

	// 停止所有连接池中的连接
	s.stopConnectionPool(s.spotPool, "spot")
	s.stopConnectionPool(s.futuresPool, "futures")

	log.Printf("[WebSocketSyncer] Stopped")
}

// stopConnectionPool 停止连接池中的所有连接
func (s *WebSocketSyncer) stopConnectionPool(pool *WebSocketConnectionPool, poolType string) {
	connections := pool.GetAllConnections()
	for _, conn := range connections {
		conn.mu.Lock()
		if conn.conn != nil {
			conn.conn.Close()
			conn.conn = nil
		}
		conn.isHealthy = false
		conn.mu.Unlock()
	}
	log.Printf("[WebSocketSyncer] Stopped %d %s connections", len(connections), poolType)
}

// connect 建立WebSocket连接
func (s *WebSocketSyncer) connect() error {
	// 建立现货WebSocket连接池
	if err := s.initializeSpotConnections(); err != nil {
		return fmt.Errorf("failed to initialize spot connections: %w", err)
	}

	// 建立期货WebSocket连接池
	if err := s.initializeFuturesConnections(); err != nil {
		log.Printf("[WebSocketSyncer] Failed to initialize futures connections: %v, continuing with spot only", err)
		// 期货连接失败不影响现货连接，继续运行
	}

	return nil
}

// initializeSpotConnections 初始化现货连接池
func (s *WebSocketSyncer) initializeSpotConnections() error {
	log.Printf("[WebSocketSyncer] Initializing spot connection pool...")

	// 至少创建一个连接
	conn, err := s.createConnection("spot")
	if err != nil {
		return fmt.Errorf("failed to create initial spot connection: %w", err)
	}
	s.spotPool.AddConnection(conn)

	log.Printf("[WebSocketSyncer] Spot connection pool initialized")
	return nil
}

// initializeFuturesConnections 初始化期货连接池
func (s *WebSocketSyncer) initializeFuturesConnections() error {
	log.Printf("[WebSocketSyncer] Initializing futures connection pool...")

	// 至少创建一个连接
	conn, err := s.createConnection("futures")
	if err != nil {
		return fmt.Errorf("failed to create initial futures connection: %w", err)
	}
	s.futuresPool.AddConnection(conn)

	log.Printf("[WebSocketSyncer] Futures connection pool initialized")
	return nil
}

// createConnection 创建指定类型的连接
func (s *WebSocketSyncer) createConnection(connType string) (*WebSocketConnection, error) {
	var url string
	if connType == "futures" {
		url = "wss://fstream.binance.com/ws"
	} else {
		url = "wss://stream.binance.com:9443/ws"
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s websocket: %w", connType, err)
	}

	wsConn := &WebSocketConnection{
		conn:       conn,
		connType:   connType,
		symbols:    make([]string, 0),
		lastActive: time.Now(),
		isHealthy:  true,
	}

	log.Printf("[WebSocketSyncer] Created %s WebSocket connection", connType)
	return wsConn, nil
}

// subscribeToStreams 订阅数据流
func (s *WebSocketSyncer) subscribeToStreams() error {
	// 获取智能筛选的交易对
	symbols, err := s.getSmartSymbolsToSubscribe()
	if err != nil {
		return fmt.Errorf("failed to get smart symbols: %w", err)
	}

	s.subscribedSymbols = symbols

	// 订阅现货数据流 - 分散到多个连接以避免单连接过载
	if err := s.subscribeSpotStreamsDistributed(symbols); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe spot streams: %v", err)
	}

	// 订阅期货数据流 - 分散到多个连接以避免单连接过载
	if err := s.subscribeFuturesStreamsDistributed(symbols); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe futures streams: %v", err)
	}

	log.Printf("[WebSocketSyncer] Smart subscribed to streams for %d symbols",
		len(symbols))
	return nil
}

// subscribeSpotStreamsDistributed 分散订阅现货数据流到多个连接
func (s *WebSocketSyncer) subscribeSpotStreamsDistributed(symbols []string) error {
	const maxStreamsPerConnection = 100 // 每个连接最多100个流

	// 1. 价格流 - 使用专门的连接
	tickerStreams := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		tickerStreams = append(tickerStreams, fmt.Sprintf("%s@ticker", strings.ToLower(symbol)))
	}
	if err := s.subscribeStreamsToDedicatedConnection(tickerStreams, "ticker", maxStreamsPerConnection); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe ticker streams: %v", err)
	}

	// 2. K线流 - 使用专门的连接，每种间隔分开
	klineIntervals := []string{"1m", "5m", "1h"}
	totalKlineStreams := 0
	for _, interval := range klineIntervals {
		klineStreams := make([]string, 0, len(symbols))
		for _, symbol := range symbols {
			klineStreams = append(klineStreams, fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval))
		}
		totalKlineStreams += len(klineStreams)
		if err := s.subscribeStreamsToDedicatedConnection(klineStreams, fmt.Sprintf("kline_%s", interval), maxStreamsPerConnection); err != nil {
			log.Printf("[WebSocketSyncer] Failed to subscribe %s kline streams: %v", interval, err)
		}
	}

	// 3. 深度流 - 仅为最重要的交易对订阅
	depthLimit := 10
	if len(symbols) < depthLimit {
		depthLimit = len(symbols)
	}
	depthSymbols := symbols[:depthLimit]
	depthStreams := make([]string, 0, len(depthSymbols))
	for _, symbol := range depthSymbols {
		depthStreams = append(depthStreams, fmt.Sprintf("%s@depth@100ms", strings.ToLower(symbol)))
	}
	if err := s.subscribeStreamsToDedicatedConnection(depthStreams, "depth", maxStreamsPerConnection); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe depth streams: %v", err)
	}

	// 4. 交易流 - 仅为最重要的交易对订阅
	tradeLimit := 20
	if len(symbols) < tradeLimit {
		tradeLimit = len(symbols)
	}
	tradeSymbols := symbols[:tradeLimit]
	tradeStreams := make([]string, 0, len(tradeSymbols))
	for _, symbol := range tradeSymbols {
		tradeStreams = append(tradeStreams, fmt.Sprintf("%s@trade", strings.ToLower(symbol)))
	}
	if err := s.subscribeStreamsToDedicatedConnection(tradeStreams, "trade", maxStreamsPerConnection); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe trade streams: %v", err)
	}

	totalStreams := len(tickerStreams) + totalKlineStreams + len(depthStreams) + len(tradeStreams)
	log.Printf("[WebSocketSyncer] Distributed subscription: %d symbols -> %d total streams across multiple connections",
		len(symbols), totalStreams)
	return nil
}

// subscribeStreamsToDedicatedConnection 为特定类型的流创建专用连接并订阅
func (s *WebSocketSyncer) subscribeStreamsToDedicatedConnection(streams []string, streamType string, maxStreamsPerConnection int) error {
	if len(streams) == 0 {
		return nil
	}

	// 将流分组，每组最多maxStreamsPerConnection个
	streamGroups := s.groupStreams(streams, maxStreamsPerConnection)

	totalConnections := 0
	for i, group := range streamGroups {
		// 为每个组创建专用连接
		conn, err := s.createConnection(fmt.Sprintf("spot_%s_%d", streamType, i))
		if err != nil {
			log.Printf("[WebSocketSyncer] Failed to create connection for %s group %d: %v", streamType, i, err)
			continue
		}

		// 将连接添加到连接池
		s.spotPool.AddConnection(conn)

		// 发送订阅消息
		subscribeMsg := map[string]interface{}{
			"method": "SUBSCRIBE",
			"params": group,
			"id":     i + 1,
		}

		if err := conn.conn.WriteJSON(subscribeMsg); err != nil {
			log.Printf("[WebSocketSyncer] Failed to subscribe %s group %d: %v", streamType, i, err)
			continue
		}

		log.Printf("[WebSocketSyncer] Subscribed %s group %d: %d streams", streamType, i, len(group))
		totalConnections++
	}

	if totalConnections == 0 {
		return fmt.Errorf("failed to create any connections for %s streams", streamType)
	}

	log.Printf("[WebSocketSyncer] Created %d connections for %s streams (%d total streams)",
		totalConnections, streamType, len(streams))
	return nil
}

// groupStreams 将流分组，每组最多maxStreamsPerConnection个
func (s *WebSocketSyncer) groupStreams(streams []string, maxStreamsPerConnection int) [][]string {
	var groups [][]string
	for i := 0; i < len(streams); i += maxStreamsPerConnection {
		end := i + maxStreamsPerConnection
		if end > len(streams) {
			end = len(streams)
		}
		groups = append(groups, streams[i:end])
	}
	return groups
}

// subscribeFuturesStreams 订阅期货数据流
func (s *WebSocketSyncer) subscribeFuturesStreams(symbols []string, conn *WebSocketConnection) error {
	streams := make([]string, 0, len(symbols)*5) // 价格 + K线 + 深度

	// 价格流
	for _, symbol := range symbols {
		streams = append(streams, fmt.Sprintf("%s@ticker", strings.ToLower(symbol)))
	}

	// K线流 - 与现货保持一致的时间间隔
	klineIntervals := []string{"1m", "5m", "1h"}
	for _, symbol := range symbols {
		for _, interval := range klineIntervals {
			streams = append(streams, fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval))
		}
	}

	// 深度流 - 限制数量以控制数据量
	depthLimit := 10
	if len(symbols) < depthLimit {
		depthLimit = len(symbols)
	}
	depthSymbols := symbols[:depthLimit]
	for _, symbol := range depthSymbols {
		streams = append(streams, fmt.Sprintf("%s@depth@100ms", strings.ToLower(symbol)))
	}

	// 交易流 - 限制数量以控制数据量
	tradeLimit := 20
	if len(symbols) < tradeLimit {
		tradeLimit = len(symbols)
	}
	tradeSymbols := symbols[:tradeLimit]
	for _, symbol := range tradeSymbols {
		streams = append(streams, fmt.Sprintf("%s@aggTrade", strings.ToLower(symbol)))
	}

	subscribeMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": streams,
		"id":     2,
	}

	if conn == nil || conn.conn == nil {
		return fmt.Errorf("futures connection not available")
	}

	if err := conn.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send futures subscribe message: %w", err)
	}

	log.Printf("[WebSocketSyncer] Subscribed to %d futures streams (%d tickers + %d klines + %d depths + %d trades)",
		len(streams), len(symbols), len(symbols)*len(klineIntervals), len(depthSymbols), len(tradeSymbols))
	return nil
}

// subscribeFuturesStreamsDistributed 分散订阅期货数据流到多个连接
func (s *WebSocketSyncer) subscribeFuturesStreamsDistributed(symbols []string) error {
	const maxStreamsPerConnection = 100 // 每个连接最多100个流

	// 1. 价格流 - 使用专门的连接
	tickerStreams := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		tickerStreams = append(tickerStreams, fmt.Sprintf("%s@ticker", strings.ToLower(symbol)))
	}
	if err := s.subscribeStreamsToDedicatedConnection(tickerStreams, "futures_ticker", maxStreamsPerConnection); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe futures ticker streams: %v", err)
	}

	// 2. K线流 - 使用专门的连接，每种间隔分开
	klineIntervals := []string{"1m", "5m", "1h"}
	totalKlineStreams := 0
	for _, interval := range klineIntervals {
		klineStreams := make([]string, 0, len(symbols))
		for _, symbol := range symbols {
			klineStreams = append(klineStreams, fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval))
		}
		totalKlineStreams += len(klineStreams)
		if err := s.subscribeStreamsToDedicatedConnection(klineStreams, fmt.Sprintf("futures_kline_%s", interval), maxStreamsPerConnection); err != nil {
			log.Printf("[WebSocketSyncer] Failed to subscribe futures %s kline streams: %v", interval, err)
		}
	}

	// 3. 深度流 - 仅为最重要的交易对订阅
	depthLimit := 10
	if len(symbols) < depthLimit {
		depthLimit = len(symbols)
	}
	depthSymbols := symbols[:depthLimit]
	depthStreams := make([]string, 0, len(depthSymbols))
	for _, symbol := range depthSymbols {
		depthStreams = append(depthStreams, fmt.Sprintf("%s@depth@100ms", strings.ToLower(symbol)))
	}
	if err := s.subscribeStreamsToDedicatedConnection(depthStreams, "futures_depth", maxStreamsPerConnection); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe futures depth streams: %v", err)
	}

	// 4. 交易流 - 仅为最重要的交易对订阅
	tradeLimit := 20
	if len(symbols) < tradeLimit {
		tradeLimit = len(symbols)
	}
	tradeSymbols := symbols[:tradeLimit]
	tradeStreams := make([]string, 0, len(tradeSymbols))
	for _, symbol := range tradeSymbols {
		tradeStreams = append(tradeStreams, fmt.Sprintf("%s@aggTrade", strings.ToLower(symbol)))
	}
	if err := s.subscribeStreamsToDedicatedConnection(tradeStreams, "futures_trade", maxStreamsPerConnection); err != nil {
		log.Printf("[WebSocketSyncer] Failed to subscribe futures trade streams: %v", err)
	}

	totalStreams := len(tickerStreams) + totalKlineStreams + len(depthStreams) + len(tradeStreams)
	log.Printf("[WebSocketSyncer] Distributed futures subscription: %d symbols -> %d total streams across multiple connections",
		len(symbols), totalStreams)
	return nil
}

// getSmartSymbolsToSubscribe 智能选择需要订阅的交易对
func (s *WebSocketSyncer) getSmartSymbolsToSubscribe() ([]string, error) {
	// 获取所有可用的USDT交易对
	allSymbols, err := pdb.GetUSDTTradingPairs(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get all symbols: %w", err)
	}

	maxSymbols := s.config.WebSocketMaxSymbols
	if maxSymbols <= 0 {
		maxSymbols = 200 // 默认值
	}

	// 如果总交易对不超过限制，直接返回全部
	if len(allSymbols) <= maxSymbols {
		return allSymbols, nil
	}

	// 智能筛选策略：按交易活跃度和市值排序
	smartSymbols, err := s.rankSymbolsByActivity(allSymbols)
	if err != nil {
		log.Printf("[WebSocketSyncer] Failed to rank symbols, using first %d: %v", maxSymbols, err)
		return allSymbols[:maxSymbols], nil
	}

	selectedSymbols := smartSymbols[:maxSymbols]
	log.Printf("[WebSocketSyncer] Selected top %d symbols from %d available based on activity ranking",
		len(selectedSymbols), len(allSymbols))

	return selectedSymbols, nil
}

// rankSymbolsByActivity 按交易活跃度对交易对进行排序
func (s *WebSocketSyncer) rankSymbolsByActivity(symbols []string) ([]string, error) {
	type SymbolScore struct {
		Symbol string
		Score  float64
	}

	var symbolScores []SymbolScore

	// 为每个交易对计算活跃度评分
	for _, symbol := range symbols {
		score := s.calculateSymbolActivityScore(symbol)
		symbolScores = append(symbolScores, SymbolScore{
			Symbol: symbol,
			Score:  score,
		})
	}

	// 按评分降序排序
	for i := 0; i < len(symbolScores)-1; i++ {
		for j := i + 1; j < len(symbolScores); j++ {
			if symbolScores[i].Score < symbolScores[j].Score {
				symbolScores[i], symbolScores[j] = symbolScores[j], symbolScores[i]
			}
		}
	}

	// 提取排序后的交易对
	result := make([]string, len(symbolScores))
	for i, ss := range symbolScores {
		result[i] = ss.Symbol
	}

	return result, nil
}

// calculateSymbolActivityScore 计算交易对的活跃度评分
func (s *WebSocketSyncer) calculateSymbolActivityScore(symbol string) float64 {
	score := 0.0

	// 因素1: 是否有缓存的价格数据（表示最近活跃）
	if cache, err := pdb.GetPriceCache(s.db, symbol, "spot"); err == nil && cache != nil {
		// 价格数据新鲜度（最近1小时内的数据加分）
		hoursSinceUpdate := time.Since(cache.LastUpdated).Hours()
		if hoursSinceUpdate < 1 {
			score += 10.0
		} else if hoursSinceUpdate < 24 {
			score += 5.0
		}
	}

	// 因素2: 交易量大小（从24小时统计数据获取）
	if stats, err := s.get24hStats(symbol); err == nil {
		// 基于交易量和报价量计算活跃度
		volumeScore := parseFloat(stats.Volume) / 1000000.0            // 标准化到百万级别
		quoteVolumeScore := parseFloat(stats.QuoteVolume) / 10000000.0 // 标准化到千万级别

		score += volumeScore + quoteVolumeScore
	}

	// 因素3: 价格变动幅度（表示波动性）
	if cache, err := pdb.GetPriceCache(s.db, symbol, "spot"); err == nil && cache != nil {
		// 有价格变动数据表示活跃
		if cache.PriceChange24h != nil {
			volatility := parseFloat(*cache.PriceChange24h)
			score += math.Abs(volatility) * 2 // 波动性加分
		}
	}

	// 因素4: 是否为核心交易对
	coreSymbols := map[string]bool{
		"BTCUSDT": true, "ETHUSDT": true, "BNBUSDT": true,
		"ADAUSDT": true, "SOLUSDT": true, "DOTUSDT": true,
		"DOGEUSDT": true, "AVAXUSDT": true, "LTCUSDT": true,
	}
	if coreSymbols[symbol] {
		score += 15.0 // 核心交易对额外加分
	}

	return score
}

// get24hStats 获取24小时统计数据
func (s *WebSocketSyncer) get24hStats(symbol string) (*struct {
	Volume      string `json:"volume"`
	QuoteVolume string `json:"quoteVolume"`
}, error) {
	// 优先从数据库获取最新的24小时统计数据
	var stats pdb.Binance24hStats
	err := s.db.Where("symbol = ? AND market_type = ?", symbol, "spot").Order("close_time DESC").First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果没有数据，返回零值而不是模拟数据
			return &struct {
				Volume      string `json:"volume"`
				QuoteVolume string `json:"quoteVolume"`
			}{
				Volume:      "0",
				QuoteVolume: "0",
			}, nil
		}
		return nil, fmt.Errorf("failed to get 24h stats from database: %w", err)
	}

	// 返回真实的数据库数据
	return &struct {
		Volume      string `json:"volume"`
		QuoteVolume string `json:"quoteVolume"`
	}{
		Volume:      strconv.FormatFloat(stats.Volume, 'f', -1, 64),
		QuoteVolume: strconv.FormatFloat(stats.QuoteVolume, 'f', -1, 64),
	}, nil
}

// receiveLoop 接收数据循环
// receiveLoop 启动接收循环
func (s *WebSocketSyncer) receiveLoop(ctx context.Context) {
	// 启动现货连接池中所有连接的接收goroutine
	spotConnections := s.spotPool.GetAllConnections()
	for _, conn := range spotConnections {
		if conn != nil && conn.conn != nil && conn.isHealthy {
			go s.receiveFromConnection(ctx, conn.conn, conn.connType)
		}
	}

	// 启动期货连接池中所有连接的接收goroutine
	futuresConnections := s.futuresPool.GetAllConnections()
	for _, conn := range futuresConnections {
		if conn != nil && conn.conn != nil && conn.isHealthy {
			go s.receiveFromConnection(ctx, conn.conn, conn.connType)
		}
	}

	totalConnections := len(spotConnections) + len(futuresConnections)
	if totalConnections == 0 {
		log.Printf("[WebSocketSyncer] No connections available for receive loop")
	} else {
		log.Printf("[WebSocketSyncer] Started receive loops for %d connections (%d spot, %d futures)",
			totalConnections, len(spotConnections), len(futuresConnections))
	}
}

// receiveFromConnection 从指定连接接收消息
func (s *WebSocketSyncer) receiveFromConnection(ctx context.Context, conn *websocket.Conn, connType string) {
	defer func() {
		log.Printf("[WebSocketSyncer] %s receive loop ended", connType)
	}()

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WebSocketSyncer] %s receive loop stopped due to context cancellation", connType)
			return
		default:
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				consecutiveErrors++
				log.Printf("[WebSocketSyncer] %s read error (consecutive: %d/%d): %v",
					connType, consecutiveErrors, maxConsecutiveErrors, err)

				// 分类处理不同类型的错误
				errorType := s.classifyError(err)

				switch errorType {
				case "timeout":
					// 网络超时错误 - 表示连接可能有问题
					log.Printf("[WebSocketSyncer] %s network timeout detected (%d/%d), triggering reconnect",
						connType, consecutiveErrors, maxConsecutiveErrors)
					if err := s.reconnectConnection(connType); err != nil {
						log.Printf("[WebSocketSyncer] %s timeout reconnect failed: %v", connType, err)
						if consecutiveErrors >= maxConsecutiveErrors {
							log.Printf("[WebSocketSyncer] %s too many timeout errors, terminating receive loop", connType)
							return
						}
					}
					consecutiveErrors = 0 // 重连成功后重置计数器
					continue

				case "policy_violation":
					// 策略违规错误 - 通常是永久性错误，直接退出goroutine
					log.Printf("[WebSocketSyncer] %s policy violation detected, terminating receive loop to prevent panic", connType)
					return

				case "connection_closed":
					// 连接关闭，立即尝试重连
					log.Printf("[WebSocketSyncer] %s connection closed, attempting immediate reconnect", connType)
					if err := s.reconnectConnection(connType); err != nil {
						log.Printf("[WebSocketSyncer] %s immediate reconnect failed: %v", connType, err)
					}
					consecutiveErrors = 0
					continue

				case "protocol_error":
					// 协议错误，可能需要重新订阅
					log.Printf("[WebSocketSyncer] %s protocol error, attempting resubscribe", connType)
					if err := s.resubscribeConnection(connType); err != nil {
						log.Printf("[WebSocketSyncer] %s resubscribe failed: %v", connType, err)
						if consecutiveErrors >= maxConsecutiveErrors {
							if err := s.reconnectConnection(connType); err != nil {
								log.Printf("[WebSocketSyncer] %s reconnect failed: %v", connType, err)
								return
							}
						}
					}
					consecutiveErrors = 0
					continue

				default: // 其他错误
					if consecutiveErrors >= maxConsecutiveErrors {
						log.Printf("[WebSocketSyncer] %s too many consecutive errors, triggering reconnect", connType)
						if err := s.reconnectConnection(connType); err != nil {
							log.Printf("[WebSocketSyncer] %s reconnect failed after max errors: %v", connType, err)
							return
						}
						consecutiveErrors = 0
						continue
					}

					// 使用指数退避策略
					backoff := time.Duration(consecutiveErrors*consecutiveErrors) * time.Second
					if backoff > 30*time.Second {
						backoff = 30 * time.Second
					}
					log.Printf("[WebSocketSyncer] %s waiting %v before retry", connType, backoff)
					time.Sleep(backoff)
				}
				continue
			}

			// 成功读取，重置错误计数
			consecutiveErrors = 0

			// 更新读取超时
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// 处理接收到的数据，传入连接类型用于区分
			s.processMessage(msg, connType)
		}
	}
}

// processMessage 处理接收到的消息
func (s *WebSocketSyncer) processMessage(msg map[string]interface{}, connType string) {
	stream, ok := msg["stream"].(string)
	if !ok {
		return
	}

	// 解析流类型
	if strings.Contains(stream, "@ticker") {
		startTime := time.Now()
		s.processTickerData(msg, connType)

		// 更新处理统计
		s.stats.mu.Lock()
		s.stats.messagesProcessed++
		processingTime := time.Since(startTime)
		// 计算移动平均处理时间
		if s.stats.averageProcessingTime == 0 {
			s.stats.averageProcessingTime = processingTime
		} else {
			// 简单移动平均
			s.stats.averageProcessingTime = (s.stats.averageProcessingTime + processingTime) / 2
		}
		s.stats.mu.Unlock()
	} else if strings.Contains(stream, "@kline_") {
		startTime := time.Now()
		s.processKlineData(msg, connType)

		// 更新处理统计
		s.stats.mu.Lock()
		s.stats.messagesProcessed++
		processingTime := time.Since(startTime)
		// 计算移动平均处理时间
		if s.stats.averageProcessingTime == 0 {
			s.stats.averageProcessingTime = processingTime
		} else {
			// 简单移动平均
			s.stats.averageProcessingTime = (s.stats.averageProcessingTime + processingTime) / 2
		}
		s.stats.mu.Unlock()
	} else if strings.Contains(stream, "@depth") {
		startTime := time.Now()
		s.processDepthData(msg, connType)

		// 更新处理统计
		s.stats.mu.Lock()
		s.stats.messagesProcessed++
		processingTime := time.Since(startTime)
		// 计算移动平均处理时间
		if s.stats.averageProcessingTime == 0 {
			s.stats.averageProcessingTime = processingTime
		} else {
			// 简单移动平均
			s.stats.averageProcessingTime = (s.stats.averageProcessingTime + processingTime) / 2
		}
		s.stats.mu.Unlock()
	} else if strings.Contains(stream, "@trade") {
		startTime := time.Now()
		s.processTradeData(msg, connType)

		// 更新处理统计
		s.stats.mu.Lock()
		s.stats.messagesProcessed++
		processingTime := time.Since(startTime)
		// 计算移动平均处理时间
		if s.stats.averageProcessingTime == 0 {
			s.stats.averageProcessingTime = processingTime
		} else {
			// 简单移动平均
			s.stats.averageProcessingTime = (s.stats.averageProcessingTime + processingTime) / 2
		}
		s.stats.mu.Unlock()
	}
}

// processTickerData 处理价格数据
func (s *WebSocketSyncer) processTickerData(msg map[string]interface{}, connType string) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}

	symbol, ok := data["s"].(string)
	if !ok {
		return
	}

	priceStr, ok := data["c"].(string)
	if !ok {
		return
	}

	// 直接使用字符串格式的价格，与数据库保持一致
	timestamp := time.Now().UnixMilli()

	s.cacheMu.Lock()
	if connType == "futures" {
		// 期货价格数据
		s.futuresCache[symbol] = FuturesData{
			Symbol: symbol,
			Price:  priceStr, // 保持字符串格式，与数据库一致
			Time:   timestamp,
		}

		// 每100条消息打印一次调试信息
		s.stats.mu.Lock()
		s.stats.totalFuturesPriceUpdates++
		if s.stats.totalFuturesPriceUpdates%100 == 0 {
			log.Printf("[WebSocketSyncer] 📈 Cached %d futures price updates, latest: %s = %s",
				s.stats.totalFuturesPriceUpdates, symbol, priceStr)
		}
		s.stats.mu.Unlock()
	} else {
		// 现货价格数据
		s.priceCache[symbol] = PriceData{
			Symbol: symbol,
			Price:  priceStr, // 保持字符串格式，与数据库一致
			Time:   timestamp,
		}

		// 每100条消息打印一次调试信息
		s.stats.mu.Lock()
		s.stats.totalSpotPriceUpdates++
		if s.stats.totalSpotPriceUpdates%100 == 0 {
			log.Printf("[WebSocketSyncer] 📈 Cached %d spot price updates, latest: %s = %s",
				s.stats.totalSpotPriceUpdates, symbol, priceStr)
		}
		s.stats.mu.Unlock()
	}
	s.cacheMu.Unlock()
}

// processKlineData 处理K线数据
func (s *WebSocketSyncer) processKlineData(msg map[string]interface{}, connType string) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}

	// 解析K线数据
	kline, ok := data["k"].(map[string]interface{})
	if !ok {
		return
	}

	symbol, ok := data["s"].(string)
	if !ok {
		return
	}

	// 解析K线字段
	openTime, _ := kline["t"].(float64)
	closeTime, _ := kline["T"].(float64)
	interval, _ := kline["i"].(string)
	openPrice, _ := kline["o"].(string)
	highPrice, _ := kline["h"].(string)
	lowPrice, _ := kline["l"].(string)
	closePrice, _ := kline["c"].(string)
	volume, _ := kline["v"].(string)
	quoteVolume, _ := kline["q"].(string)
	tradeCountFloat, _ := kline["n"].(float64)

	klineData := KlineData{
		Symbol:      symbol,
		Interval:    interval,
		OpenTime:    int64(openTime),
		CloseTime:   int64(closeTime),
		OpenPrice:   openPrice,
		HighPrice:   highPrice,
		LowPrice:    lowPrice,
		ClosePrice:  closePrice,
		Volume:      volume,
		QuoteVolume: quoteVolume,
		TradeCount:  int(tradeCountFloat),
	}

	// 生成缓存键，包含symbol、connType、interval和时间戳以唯一标识
	cacheKey := fmt.Sprintf("%s_%s_%s_%d", symbol, connType, interval, int64(openTime))

	s.cacheMu.Lock()
	s.klineCache[cacheKey] = klineData
	s.cacheMu.Unlock()

	// 更新K线统计
	s.stats.mu.Lock()
	// 可以添加专门的K线统计字段，如果需要的话
	s.stats.mu.Unlock()
}

// processDepthData 处理深度数据
func (s *WebSocketSyncer) processDepthData(msg map[string]interface{}, connType string) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}

	symbol, ok := data["s"].(string)
	if !ok {
		return
	}

	lastUpdateID, _ := data["u"].(float64)
	bidsRaw, _ := data["b"].([]interface{})
	asksRaw, _ := data["a"].([]interface{})

	// 转换bids和asks为字符串数组
	bids := make([][]string, 0, len(bidsRaw))
	for _, bid := range bidsRaw {
		if bidArr, ok := bid.([]interface{}); ok && len(bidArr) >= 2 {
			price, _ := bidArr[0].(string)
			quantity, _ := bidArr[1].(string)
			bids = append(bids, []string{price, quantity})
		}
	}

	asks := make([][]string, 0, len(asksRaw))
	for _, ask := range asksRaw {
		if askArr, ok := ask.([]interface{}); ok && len(askArr) >= 2 {
			price, _ := askArr[0].(string)
			quantity, _ := askArr[1].(string)
			asks = append(asks, []string{price, quantity})
		}
	}

	depthData := DepthData{
		Symbol:       symbol,
		LastUpdateID: int64(lastUpdateID),
		Bids:         bids,
		Asks:         asks,
		Timestamp:    time.Now().UnixMilli(),
	}

	// 生成缓存键，包含symbol和kind以唯一标识
	cacheKey := fmt.Sprintf("%s_%s", symbol, connType)

	s.cacheMu.Lock()
	s.depthCache[cacheKey] = depthData
	s.cacheMu.Unlock()

	// 更新深度数据统计
	s.stats.mu.Lock()
	// 可以添加专门的深度数据统计字段，如果需要的话
	s.stats.mu.Unlock()
}

// processTradeData 处理交易数据
func (s *WebSocketSyncer) processTradeData(msg map[string]interface{}, connType string) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}

	symbol, ok := data["s"].(string)
	if !ok {
		return
	}

	tradeID, _ := data["t"].(float64)
	price, _ := data["p"].(string)
	quantity, _ := data["q"].(string)
	tradeTime, _ := data["T"].(float64)
	isBuyerMaker, _ := data["m"].(bool)

	tradeData := TradeData{
		Symbol:       symbol,
		TradeID:      int64(tradeID),
		Price:        price,
		Quantity:     quantity,
		TradeTime:    int64(tradeTime),
		IsBuyerMaker: isBuyerMaker,
	}

	s.cacheMu.Lock()
	s.tradeCache = append(s.tradeCache, tradeData)
	s.cacheMu.Unlock()

	// 更新交易数据统计
	s.stats.mu.Lock()
	// 可以添加专门的交易数据统计字段，如果需要的话
	s.stats.mu.Unlock()
}

// batchSaveLoop 批量保存循环
func (s *WebSocketSyncer) batchSaveLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.saveCachedData()
		}
	}
}

// saveCachedData 保存缓存的数据
func (s *WebSocketSyncer) saveCachedData() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// 保存现货价格数据
	if len(s.priceCache) > 0 {
		s.savePriceData(s.priceCache)
		s.priceCache = make(map[string]PriceData) // 清空缓存
	}

	// 保存期货价格数据
	if len(s.futuresCache) > 0 {
		s.saveFuturesData(s.futuresCache)
		s.futuresCache = make(map[string]FuturesData) // 清空缓存
	}

	// 保存K线数据
	if len(s.klineCache) > 0 {
		s.saveKlineData(s.klineCache)
		s.klineCache = make(map[string]KlineData) // 清空缓存
	}

	// 保存深度数据
	if len(s.depthCache) > 0 {
		s.saveDepthData(s.depthCache)
		s.depthCache = make(map[string]DepthData) // 清空缓存
	}

	// 保存交易数据
	if len(s.tradeCache) > 0 {
		s.saveTradeData(s.tradeCache)
		s.tradeCache = make([]TradeData, 0) // 清空缓存
	}
}

// savePriceData 保存现货价格数据
func (s *WebSocketSyncer) savePriceData(data map[string]PriceData) {
	// 批量保存到数据库（复用价格同步器的逻辑）
	for symbol, priceData := range data {
		// 创建价格缓存记录
		priceRecord := &pdb.PriceCache{
			Symbol:      symbol,
			Kind:        "spot",
			Price:       priceData.Price, // 已经是字符串格式
			LastUpdated: time.UnixMilli(priceData.Time),
		}

		// 保存到数据库
		if err := pdb.SavePriceCache(s.db, priceRecord); err != nil {
			log.Printf("[WebSocketSyncer] ❌ Failed to save spot price for %s: %v", symbol, err)
			continue
		}

		log.Printf("[WebSocketSyncer] ✅ Saved spot price: %s = %s", symbol, priceData.Price)
	}
}

// saveFuturesData 保存期货价格数据
func (s *WebSocketSyncer) saveFuturesData(data map[string]FuturesData) {
	// 批量保存到数据库（复用期货同步器的逻辑）
	for symbol, futuresData := range data {
		// 创建价格缓存记录
		priceRecord := pdb.PriceCache{
			Symbol:      symbol,
			Kind:        "futures",
			Price:       futuresData.Price,
			LastUpdated: time.UnixMilli(futuresData.Time),
		}

		// 保存到数据库
		if err := s.db.Create(&priceRecord).Error; err != nil {
			log.Printf("[WebSocketSyncer] Failed to save futures price for %s: %v", symbol, err)
			continue
		}

		log.Printf("[WebSocketSyncer] ✅ Saved futures price: %s = %.4f", symbol, futuresData.Price)
	}
}

// saveKlineData 保存K线数据
func (s *WebSocketSyncer) saveKlineData(data map[string]KlineData) {
	// 转换为数据库格式并批量保存
	var klines []pdb.MarketKline
	for cacheKey, klineData := range data {
		// 从缓存键中解析信息：格式为 symbol_connType_interval_timestamp
		parts := strings.Split(cacheKey, "_")
		if len(parts) < 4 {
			log.Printf("[WebSocketSyncer] ⚠️ Invalid kline cache key format: %s", cacheKey)
			continue
		}

		symbol := parts[0]
		connType := parts[1] // "spot" 或 "futures"
		// parts[2] 是 interval, parts[3] 是 timestamp

		// 清理symbol中的类型后缀（如果有的话）
		cleanSymbol := strings.TrimSuffix(symbol, "_futures")
		cleanSymbol = strings.TrimSuffix(cleanSymbol, "_spot")

		kline := pdb.MarketKline{
			Symbol:      cleanSymbol,
			Kind:        connType, // 使用从缓存键中解析的类型
			Interval:    klineData.Interval,
			OpenTime:    time.UnixMilli(klineData.OpenTime),
			OpenPrice:   klineData.OpenPrice,
			HighPrice:   klineData.HighPrice,
			LowPrice:    klineData.LowPrice,
			ClosePrice:  klineData.ClosePrice,
			Volume:      klineData.Volume,
			QuoteVolume: &klineData.QuoteVolume,
			TradeCount:  &klineData.TradeCount,
		}
		klines = append(klines, kline)
	}

	// 批量保存到数据库
	if len(klines) > 0 {
		if err := pdb.SaveMarketKlines(s.db, klines); err != nil {
			log.Printf("[WebSocketSyncer] ❌ Failed to save %d klines: %v", len(klines), err)
		} else {
			log.Printf("[WebSocketSyncer] ✅ Saved %d klines to database", len(klines))
		}
	}
}

// saveDepthData 保存深度数据
func (s *WebSocketSyncer) saveDepthData(data map[string]DepthData) {
	// 转换为数据库格式并批量保存
	var depths []pdb.BinanceOrderBookDepth
	for cacheKey, depthData := range data {
		// 从缓存键中解析symbol和kind
		parts := strings.Split(cacheKey, "_")
		if len(parts) < 2 {
			log.Printf("[WebSocketSyncer] ⚠️ Invalid depth cache key: %s", cacheKey)
			continue
		}
		symbol := parts[0]
		kind := parts[1] // "spot" or "futures"

		// 将bids和asks转换为JSON字符串
		bidsJSON, _ := json.Marshal(depthData.Bids)
		asksJSON, _ := json.Marshal(depthData.Asks)

		depth := pdb.BinanceOrderBookDepth{
			Symbol:       symbol,
			MarketType:   kind,
			LastUpdateID: depthData.LastUpdateID,
			Bids:         string(bidsJSON),
			Asks:         string(asksJSON),
			SnapshotTime: depthData.Timestamp,
		}
		depths = append(depths, depth)
	}

	// 批量保存到数据库
	if len(depths) > 0 {
		if err := pdb.SaveOrderBookDepth(s.db, depths); err != nil {
			log.Printf("[WebSocketSyncer] ❌ Failed to save %d depth snapshots: %v", len(depths), err)
		} else {
			log.Printf("[WebSocketSyncer] ✅ Saved %d depth snapshots to database", len(depths))
		}
	}
}

// saveTradeData 保存交易数据
func (s *WebSocketSyncer) saveTradeData(data []TradeData) {
	// 转换为数据库格式并批量保存
	var trades []pdb.BinanceTrade
	for _, tradeData := range data {
		trade := pdb.BinanceTrade{
			Symbol:       tradeData.Symbol,
			MarketType:   "spot", // 目前只处理现货交易
			TradeID:      tradeData.TradeID,
			Price:        tradeData.Price,
			Quantity:     tradeData.Quantity,
			TradeTime:    tradeData.TradeTime,
			IsBuyerMaker: tradeData.IsBuyerMaker,
		}
		trades = append(trades, trade)
	}

	// 批量保存到数据库
	if len(trades) > 0 {
		if err := pdb.SaveTrades(s.db, trades); err != nil {
			log.Printf("[WebSocketSyncer] ❌ Failed to save %d trades: %v", len(trades), err)
		} else {
			log.Printf("[WebSocketSyncer] ✅ Saved %d trades to database", len(trades))
		}
	}
}

// reconnectConnection 重新连接指定的连接类型
func (s *WebSocketSyncer) reconnectConnection(connType string) error {
	// 检查重连冷却时间
	if time.Since(s.lastReconnectTime) < s.reconnectCooldown {
		log.Printf("[WebSocketSyncer] %s reconnect blocked by cooldown (%v remaining)",
			connType, s.reconnectCooldown-time.Since(s.lastReconnectTime))
		return fmt.Errorf("reconnect blocked by cooldown")
	}

	log.Printf("[WebSocketSyncer] Attempting to reconnect %s connection", connType)
	s.lastReconnectTime = time.Now()

	// 更新重连统计
	s.stats.mu.Lock()
	s.stats.reconnectCount++
	s.stats.mu.Unlock()

	maxRetries := 3
	baseDelay := time.Duration(s.config.WebSocketReconnectDelay) * time.Second
	if baseDelay <= 0 {
		baseDelay = 5 * time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[WebSocketSyncer] %s reconnect attempt %d/%d", connType, attempt, maxRetries)

		// 创建新连接
		newConn, err := s.createConnection(connType)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				delay := baseDelay * time.Duration(1<<(attempt-1))
				log.Printf("[WebSocketSyncer] %s reconnect failed, retrying in %v: %v", connType, delay, err)
				time.Sleep(delay)
			}
			continue
		}

		// 添加到连接池
		if strings.Contains(connType, "futures") {
			s.futuresPool.AddConnection(newConn)
			// 对于分布式架构，单个连接重连不重新订阅
			// 订阅由全局订阅流程管理
		} else {
			s.spotPool.AddConnection(newConn)
			// 对于分布式架构，单个连接重连不重新订阅
			// 订阅由全局订阅流程管理
		}

		// 为重连的连接启动接收goroutine
		go s.receiveFromConnection(context.Background(), newConn.conn, connType)

		log.Printf("[WebSocketSyncer] %s reconnect successful", connType)
		return nil
	}

	return fmt.Errorf("failed to reconnect %s after %d attempts: %w", connType, maxRetries, lastErr)
}

// Sync 实现DataSyncer接口（用于兼容性）
func (s *WebSocketSyncer) Sync(ctx context.Context) error {
	// WebSocket是持续连接，不需要定期同步
	return nil
}

// GetStats 获取统计信息
func (s *WebSocketSyncer) GetStats() map[string]interface{} {
	s.cacheMu.RLock()
	priceCacheSize := len(s.priceCache)
	futuresCacheSize := len(s.futuresCache)
	s.cacheMu.RUnlock()

	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	// 计算连接池状态
	spotConnections := s.spotPool.GetAllConnections()
	futuresConnections := s.futuresPool.GetAllConnections()
	totalConnections := len(spotConnections) + len(futuresConnections)

	connectionStatus := fmt.Sprintf("pool: %d spot + %d futures = %d total",
		len(spotConnections), len(futuresConnections), totalConnections)

	// 计算消息处理率
	var messagesPerSecond float64
	if !s.stats.lastMessageTime.IsZero() {
		elapsed := time.Since(s.stats.lastMessageTime)
		if elapsed.Seconds() > 0 {
			messagesPerSecond = float64(s.stats.messagesProcessed) / elapsed.Seconds()
		}
	}

	return map[string]interface{}{
		// 连接状态
		"is_running":        s.isRunning,
		"connection_status": connectionStatus,

		// 订阅信息
		"subscribed_count": len(s.subscribedSymbols),

		// 缓存状态
		"price_cache_size":   priceCacheSize,
		"futures_cache_size": futuresCacheSize,

		// 性能指标
		"messages_received":       s.stats.messagesReceived,
		"messages_processed":      s.stats.messagesProcessed,
		"messages_per_second":     messagesPerSecond,
		"last_message_time":       s.stats.lastMessageTime,
		"reconnect_count":         s.stats.reconnectCount,
		"cache_hit_rate":          s.stats.cacheHitRate,
		"average_processing_time": s.stats.averageProcessingTime.String(),
	}
}

// Name 返回同步器名称
func (s *WebSocketSyncer) Name() string {
	return "websocket"
}

// healthCheckLoop 健康检查循环
func (s *WebSocketSyncer) healthCheckLoop(ctx context.Context) {
	interval := 30 * time.Second // 默认30秒
	if s.config.WebSocketHealthCheckInterval > 0 {
		interval = time.Duration(s.config.WebSocketHealthCheckInterval) * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[WebSocketSyncer] Health check started with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (s *WebSocketSyncer) performHealthCheck() {
	s.mu.RLock()
	isRunning := s.isRunning
	s.mu.RUnlock()

	if !isRunning {
		return
	}

	// 检查连接池状态
	spotConnections := s.spotPool.GetAllConnections()
	futuresConnections := s.futuresPool.GetAllConnections()

	// 检查现货连接
	for _, conn := range spotConnections {
		if conn.conn == nil || !conn.isHealthy {
			log.Printf("[WebSocketSyncer] Health check: spot connection unhealthy, triggering reconnect")
			go s.reconnectConnection("spot")
			break
		}
	}

	// 检查期货连接
	for _, conn := range futuresConnections {
		if conn.conn == nil || !conn.isHealthy {
			log.Printf("[WebSocketSyncer] Health check: futures connection unhealthy, triggering reconnect")
			go s.reconnectConnection("futures")
			break
		}
	}

	// 检查最后消息时间（如果超过5分钟没有收到消息，可能连接有问题）
	s.stats.mu.RLock()
	lastMessageTime := s.stats.lastMessageTime
	s.stats.mu.RUnlock()

	if !lastMessageTime.IsZero() && time.Since(lastMessageTime) > 5*time.Minute {
		log.Printf("[WebSocketSyncer] Health check: no messages for %v, triggering automatic reconnection",
			time.Since(lastMessageTime))

		// 触发所有连接的自动重连
		if err := s.triggerGlobalReconnection(); err != nil {
			log.Printf("[WebSocketSyncer] ❌ Global reconnection failed: %v", err)
		} else {
			log.Printf("[WebSocketSyncer] ✅ Global reconnection initiated")
		}
	}

	// 检查缓存大小，如果过大可能是处理不过来
	s.cacheMu.RLock()
	cacheSize := len(s.priceCache) + len(s.futuresCache) + len(s.klineCache) + len(s.depthCache) + len(s.tradeCache)
	s.cacheMu.RUnlock()

	if cacheSize > 1000 { // 缓存超过1000条可能是处理延迟
		log.Printf("[WebSocketSyncer] Health check: total cache size is %d, may indicate processing delay", cacheSize)
	}

	// 动态调整订阅（如果配置了的话）
	s.adjustSubscriptionsDynamically()
}

// adjustSubscriptionsDynamically 动态调整订阅
func (s *WebSocketSyncer) adjustSubscriptionsDynamically() {
	// 检查是否启用自动调整
	if !s.config.WebSocketEnableAutoAdjust {
		return
	}

	// 检查当前订阅利用率
	s.stats.mu.RLock()
	timeSinceLastMessage := time.Since(s.stats.lastMessageTime)
	var messagesPerSecond float64
	if timeSinceLastMessage.Seconds() > 0 {
		messagesPerSecond = float64(s.stats.messagesProcessed) / timeSinceLastMessage.Seconds()
	}
	s.stats.mu.RUnlock()

	maxSymbols := s.config.WebSocketMaxSymbols
	currentSymbols := len(s.subscribedSymbols)

	// 动态调整策略
	if messagesPerSecond > 15 && currentSymbols < maxSymbols && currentSymbols < 150 {
		// 消息处理率很高且订阅数没有达到上限，增加少量订阅
		addCount := min(10, maxSymbols-currentSymbols) // 最多增加10个
		if addCount > 0 {
			log.Printf("[WebSocketSyncer] High message rate (%.1f msg/s), expanding subscriptions by %d (current: %d/%d)",
				messagesPerSecond, addCount, currentSymbols, maxSymbols)
			s.expandSubscriptions(addCount)
		}

	} else if messagesPerSecond < 0.5 && currentSymbols > 50 && timeSinceLastMessage > time.Minute {
		// 消息处理率很低且订阅数很多，减少订阅
		reduceCount := min(20, currentSymbols-50) // 最多减少到50个
		if reduceCount > 0 {
			log.Printf("[WebSocketSyncer] Low message rate (%.1f msg/s), reducing subscriptions by %d (current: %d)",
				messagesPerSecond, reduceCount, currentSymbols)
			s.reduceSubscriptions(reduceCount)
		}
	}
}

// expandSubscriptions 扩展订阅
func (s *WebSocketSyncer) expandSubscriptions(count int) {
	// 获取所有可用的交易对
	allSymbols, err := pdb.GetUSDTTradingPairs(s.db)
	if err != nil {
		log.Printf("[WebSocketSyncer] Failed to get symbols for expansion: %v", err)
		return
	}

	// 找出未订阅的交易对
	subscribedMap := make(map[string]bool)
	for _, sym := range s.subscribedSymbols {
		subscribedMap[sym] = true
	}

	var newSymbols []string
	for _, sym := range allSymbols {
		if !subscribedMap[sym] {
			newSymbols = append(newSymbols, sym)
			if len(newSymbols) >= count {
				break
			}
		}
	}

	if len(newSymbols) > 0 {
		// 获取一个可用的连接来发送订阅消息
		conn := s.spotPool.GetBalancedConnection()
		if conn == nil || conn.conn == nil {
			log.Printf("[WebSocketSyncer] No available connection for subscription expansion")
			return
		}

		// 发送订阅消息
		streams := make([]string, len(newSymbols))
		for i, symbol := range newSymbols {
			streams[i] = fmt.Sprintf("%s@ticker", strings.ToLower(symbol))
		}

		subscribeMsg := map[string]interface{}{
			"method": "SUBSCRIBE",
			"params": streams,
			"id":     time.Now().Unix(),
		}

		if err := conn.conn.WriteJSON(subscribeMsg); err != nil {
			log.Printf("[WebSocketSyncer] Failed to subscribe to %d new symbols: %v", len(newSymbols), err)
			return
		}

		// 更新连接的订阅列表
		conn.AddSymbols(newSymbols)

		// 更新全局订阅列表
		s.subscribedSymbols = append(s.subscribedSymbols, newSymbols...)
		log.Printf("[WebSocketSyncer] Successfully subscribed to %d additional symbols (total: %d)",
			len(newSymbols), len(s.subscribedSymbols))
	}
}

// reduceSubscriptions 减少订阅
func (s *WebSocketSyncer) reduceSubscriptions(count int) {
	if count <= 0 || len(s.subscribedSymbols) <= 50 {
		return
	}

	// 选择要取消订阅的交易对（选择活跃度最低的）
	reduceSymbols := s.selectSymbolsToReduce(count)

	if len(reduceSymbols) > 0 {
		// 获取连接池中的所有连接，发送取消订阅消息
		spotConnections := s.spotPool.GetAllConnections()

		// 发送取消订阅消息到所有现货连接
		for _, conn := range spotConnections {
			if conn.conn != nil && conn.isHealthy {
				streams := make([]string, len(reduceSymbols))
				for i, symbol := range reduceSymbols {
					streams[i] = fmt.Sprintf("%s@ticker", strings.ToLower(symbol))
				}

				unsubscribeMsg := map[string]interface{}{
					"method": "UNSUBSCRIBE",
					"params": streams,
					"id":     time.Now().Unix(),
				}

				if err := conn.conn.WriteJSON(unsubscribeMsg); err != nil {
					log.Printf("[WebSocketSyncer] Failed to unsubscribe from %d symbols on connection: %v", len(reduceSymbols), err)
					continue
				}

				// 从连接的订阅列表中移除
				conn.RemoveSymbols(reduceSymbols)
			}
		}

		// 更新订阅列表
		newSubscribed := make([]string, 0, len(s.subscribedSymbols)-len(reduceSymbols))
		reduceMap := make(map[string]bool)
		for _, sym := range reduceSymbols {
			reduceMap[sym] = true
		}

		for _, sym := range s.subscribedSymbols {
			if !reduceMap[sym] {
				newSubscribed = append(newSubscribed, sym)
			}
		}

		s.subscribedSymbols = newSubscribed
		log.Printf("[WebSocketSyncer] Successfully unsubscribed from %d symbols (total: %d)",
			len(reduceSymbols), len(s.subscribedSymbols))
	}
}

// selectSymbolsToReduce 选择要减少订阅的交易对
func (s *WebSocketSyncer) selectSymbolsToReduce(count int) []string {
	if len(s.subscribedSymbols) <= count+50 {
		return s.subscribedSymbols[len(s.subscribedSymbols)-count:]
	}

	// 按活跃度排序，选择活跃度最低的
	symbolScores := make([]struct {
		Symbol string
		Score  float64
	}, len(s.subscribedSymbols))

	for i, symbol := range s.subscribedSymbols {
		symbolScores[i] = struct {
			Symbol string
			Score  float64
		}{
			Symbol: symbol,
			Score:  s.calculateSymbolActivityScore(symbol),
		}
	}

	// 按分数升序排序（活跃度最低的排在前面）
	for i := 0; i < len(symbolScores)-1; i++ {
		for j := i + 1; j < len(symbolScores); j++ {
			if symbolScores[i].Score > symbolScores[j].Score {
				symbolScores[i], symbolScores[j] = symbolScores[j], symbolScores[i]
			}
		}
	}

	// 选择分数最低的count个
	result := make([]string, min(count, len(symbolScores)))
	for i := 0; i < len(result); i++ {
		result[i] = symbolScores[i].Symbol
	}

	return result
}

// parseFloat 解析字符串为float64
// ===== 连接池管理方法 =====

// NewWebSocketConnectionPool 创建连接池
func NewWebSocketConnectionPool(maxConnPerType int) *WebSocketConnectionPool {
	return &WebSocketConnectionPool{
		connections:    make([]*WebSocketConnection, 0),
		maxConnPerType: maxConnPerType,
	}
}

// AddConnection 添加连接到池中
func (p *WebSocketConnectionPool) AddConnection(conn *WebSocketConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 对于分布式订阅，允许更多的连接来分散流负载
	// 每个流类型可以有多个连接组
	maxConnections := p.maxConnPerType
	if strings.Contains(conn.connType, "_ticker_") ||
		strings.Contains(conn.connType, "_kline_") ||
		strings.Contains(conn.connType, "_depth_") ||
		strings.Contains(conn.connType, "_trade_") ||
		strings.Contains(conn.connType, "futures_ticker_") ||
		strings.Contains(conn.connType, "futures_kline_") ||
		strings.Contains(conn.connType, "futures_depth_") ||
		strings.Contains(conn.connType, "futures_trade_") {
		maxConnections = 20 // 分布式连接允许更多
	}

	if len(p.connections) < maxConnections {
		p.connections = append(p.connections, conn)
		log.Printf("[ConnectionPool] Added %s connection to pool (total: %d/%d)",
			conn.connType, len(p.connections), maxConnections)
	} else {
		// 连接池满时，尝试替换一个不健康的连接
		replaced := false
		for i, existingConn := range p.connections {
			if existingConn != nil && !existingConn.isHealthy {
				// 关闭不健康的连接
				existingConn.mu.Lock()
				if existingConn.conn != nil {
					existingConn.conn.Close()
				}
				existingConn.mu.Unlock()

				// 替换为新连接
				p.connections[i] = conn
				log.Printf("[ConnectionPool] Replaced unhealthy %s connection with new %s connection (total: %d/%d)",
					existingConn.connType, conn.connType, len(p.connections), maxConnections)
				replaced = true
				break
			}
		}

		// 如果没有不健康的连接可替换，且是分布式连接，允许动态扩容
		if !replaced && (strings.Contains(conn.connType, "_") || strings.Contains(conn.connType, "futures_")) {
			p.connections = append(p.connections, conn)
			log.Printf("[ConnectionPool] Pool expanded for %s connection (total: %d, expanded beyond limit: %d)",
				conn.connType, len(p.connections), maxConnections)
		} else if !replaced {
			log.Printf("[ConnectionPool] Connection pool full for %s (%d connections), rejecting new connection",
				conn.connType, len(p.connections))
		}
	}
}

// RemoveConnection 从池中移除连接
func (p *WebSocketConnectionPool) RemoveConnection(conn *websocket.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, c := range p.connections {
		if c.conn == conn {
			p.connections = append(p.connections[:i], p.connections[i+1:]...)
			log.Printf("[ConnectionPool] Removed connection from pool (remaining: %d)", len(p.connections))
			return
		}
	}
}

// GetBalancedConnection 获取负载均衡的连接
func (p *WebSocketConnectionPool) GetBalancedConnection() *WebSocketConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.connections) == 0 {
		return nil
	}

	// 简单的轮询负载均衡，选择订阅交易对最少的连接
	minSymbols := int(^uint(0) >> 1) // max int
	var selectedConn *WebSocketConnection

	for _, conn := range p.connections {
		conn.mu.RLock()
		if conn.isHealthy && len(conn.symbols) < minSymbols {
			minSymbols = len(conn.symbols)
			selectedConn = conn
		}
		conn.mu.RUnlock()
	}

	return selectedConn
}

// GetAllConnections 获取所有连接
func (p *WebSocketConnectionPool) GetAllConnections() []*WebSocketConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connections := make([]*WebSocketConnection, len(p.connections))
	copy(connections, p.connections)
	return connections
}

// UpdateConnectionHealth 更新连接健康状态
func (conn *WebSocketConnection) UpdateConnectionHealth(isHealthy bool) {
	conn.mu.Lock()
	conn.isHealthy = isHealthy
	conn.lastActive = time.Now()
	conn.mu.Unlock()
}

// AddSymbols 添加交易对到连接
func (conn *WebSocketConnection) AddSymbols(symbols []string) {
	conn.mu.Lock()
	conn.symbols = append(conn.symbols, symbols...)
	conn.lastActive = time.Now()
	conn.mu.Unlock()
}

// RemoveSymbols 从连接移除交易对
func (conn *WebSocketConnection) RemoveSymbols(symbols []string) {
	conn.mu.Lock()
	symbolSet := make(map[string]bool)
	for _, s := range symbols {
		symbolSet[s] = true
	}

	newSymbols := make([]string, 0)
	for _, s := range conn.symbols {
		if !symbolSet[s] {
			newSymbols = append(newSymbols, s)
		}
	}
	conn.symbols = newSymbols
	conn.lastActive = time.Now()
	conn.mu.Unlock()
}

// IsRunning 检查WebSocket同步器是否正在运行
func (s *WebSocketSyncer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// IsHealthy 检查WebSocket连接是否健康
func (s *WebSocketSyncer) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning {
		return false
	}

	// 检查连接池状态
	spotConnections := s.spotPool.GetAllConnections()
	futuresConnections := s.futuresPool.GetAllConnections()

	// 至少需要有一个活跃连接
	totalHealthy := 0
	for _, conn := range spotConnections {
		if conn != nil && conn.isHealthy {
			totalHealthy++
		}
	}
	for _, conn := range futuresConnections {
		if conn != nil && conn.isHealthy {
			totalHealthy++
		}
	}

	// 检查最近是否收到消息
	s.stats.mu.RLock()
	lastMessage := s.stats.lastMessageTime
	s.stats.mu.RUnlock()

	// 如果5分钟内没有收到消息，认为不健康
	if time.Since(lastMessage) > 5*time.Minute {
		return false
	}

	return totalHealthy > 0
}

// GetHealthStatus 获取详细的健康状态
func (s *WebSocketSyncer) GetHealthStatus() map[string]interface{} {
	s.mu.RLock()
	isRunning := s.isRunning
	s.mu.RUnlock()

	s.stats.mu.RLock()
	lastMessage := s.stats.lastMessageTime
	messagesReceived := s.stats.messagesReceived
	reconnectCount := s.stats.reconnectCount
	healthFailures := s.stats.healthCheckFailures
	s.stats.mu.RUnlock()

	spotConnections := s.spotPool.GetAllConnections()
	futuresConnections := s.futuresPool.GetAllConnections()

	healthySpot := 0
	healthyFutures := 0
	for _, conn := range spotConnections {
		if conn != nil && conn.isHealthy {
			healthySpot++
		}
	}
	for _, conn := range futuresConnections {
		if conn != nil && conn.isHealthy {
			healthyFutures++
		}
	}

	return map[string]interface{}{
		"is_running":              isRunning,
		"is_healthy":              s.IsHealthy(),
		"spot_connections":        len(spotConnections),
		"healthy_spot":            healthySpot,
		"futures_connections":     len(futuresConnections),
		"healthy_futures":         healthyFutures,
		"last_message_time":       lastMessage,
		"messages_received":       messagesReceived,
		"time_since_last_message": time.Since(lastMessage).String(),
		"reconnect_count":         reconnectCount,
		"health_check_failures":   healthFailures,
	}
}

// GetWebSocketStats 获取WebSocket统计信息
func (s *WebSocketSyncer) GetWebSocketStats() *server.WebSocketStats {
	s.mu.RLock()
	isRunning := s.isRunning
	s.mu.RUnlock()

	s.stats.mu.RLock()
	lastMessage := s.stats.lastMessageTime
	messagesReceived := s.stats.messagesReceived
	spotPriceUpdates := s.stats.totalSpotPriceUpdates
	futuresPriceUpdates := s.stats.totalFuturesPriceUpdates
	s.stats.mu.RUnlock()

	spotConnections := s.spotPool.GetAllConnections()
	futuresConnections := s.futuresPool.GetAllConnections()

	healthySpot := 0
	healthyFutures := 0
	for _, conn := range spotConnections {
		if conn != nil && conn.isHealthy {
			healthySpot++
		}
	}
	for _, conn := range futuresConnections {
		if conn != nil && conn.isHealthy {
			healthyFutures++
		}
	}

	var lastMessageTime *time.Time
	if !lastMessage.IsZero() {
		lastMessageTime = &lastMessage
	}

	return &server.WebSocketStats{
		IsRunning:                isRunning,
		IsHealthy:                s.IsHealthy(),
		SpotConnections:          len(spotConnections),
		HealthySpot:              healthySpot,
		FuturesConnections:       len(futuresConnections),
		HealthyFutures:           healthyFutures,
		MessagesReceived:         messagesReceived,
		LastMessageTime:          lastMessageTime,
		TotalSpotPriceUpdates:    spotPriceUpdates,
		TotalFuturesPriceUpdates: futuresPriceUpdates,
		TotalKlineUpdates:        0, // 暂时设为0，后续可扩展
		TotalDepthUpdates:        0, // 暂时设为0，后续可扩展
	}
}

// GetLatestPrice 获取最新的价格数据（从WebSocket缓存中）
func (s *WebSocketSyncer) GetLatestPrice(symbol, kind string) (string, time.Time, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	// 尝试从现货价格缓存中获取
	if kind == "spot" || kind == "" {
		if priceData, exists := s.priceCache[symbol]; exists {
			return priceData.Price, time.UnixMilli(priceData.Time), true
		}
	}

	// 尝试从期货价格缓存中获取
	if kind == "futures" || kind == "" {
		if priceData, exists := s.futuresCache[symbol]; exists {
			return priceData.Price, time.UnixMilli(priceData.Time), true
		}
	}

	return "", time.Time{}, false
}

// GetAllLatestPrices 获取所有最新的价格数据
func (s *WebSocketSyncer) GetAllLatestPrices() map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	result := make(map[string]interface{})

	// 复制现货价格数据
	for symbol, priceData := range s.priceCache {
		result[symbol+"_spot"] = map[string]interface{}{
			"price": priceData.Price,
			"time":  priceData.Time,
			"kind":  "spot",
		}
	}

	// 复制期货价格数据
	for symbol, priceData := range s.futuresCache {
		result[symbol+"_futures"] = map[string]interface{}{
			"price": priceData.Price,
			"time":  priceData.Time,
			"kind":  "futures",
		}
	}

	return result
}

// IsPriceDataFresh 检查价格数据是否足够新鲜
func (s *WebSocketSyncer) IsPriceDataFresh(symbol, kind string, maxAge time.Duration) bool {
	_, updateTime, exists := s.GetLatestPrice(symbol, kind)
	if !exists {
		return false
	}

	return time.Since(updateTime) <= maxAge
}

// triggerGlobalReconnection 触发所有连接的全局重连
func (s *WebSocketSyncer) triggerGlobalReconnection() error {
	log.Printf("[WebSocketSyncer] Triggering global reconnection for all connections")

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("WebSocket syncer is not running")
	}

	// 记录重连开始
	reconnectStart := time.Now()
	s.stats.mu.Lock()
	s.stats.reconnectCount++
	s.stats.mu.Unlock()

	// 关闭所有现有连接
	log.Printf("[WebSocketSyncer] Closing all existing connections")
	s.stopConnectionPool(s.spotPool, "spot")
	s.stopConnectionPool(s.futuresPool, "futures")

	// 等待一小段时间确保连接完全关闭
	time.Sleep(2 * time.Second)

	// 重新初始化连接
	log.Printf("[WebSocketSyncer] Reinitializing connections")

	if err := s.connect(); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	// 重新订阅数据流
	if err := s.subscribeToStreams(); err != nil {
		return fmt.Errorf("failed to resubscribe after reconnection: %w", err)
	}

	// 重新启动数据接收
	go s.receiveLoop(context.Background())

	reconnectDuration := time.Since(reconnectStart)
	log.Printf("[WebSocketSyncer] Global reconnection completed in %v", reconnectDuration)

	// 重置最后消息时间以避免立即再次触发重连
	s.stats.mu.Lock()
	s.stats.lastMessageTime = time.Now()
	s.stats.mu.Unlock()

	return nil
}

// classifyError 分类错误类型以便进行不同处理
func (s *WebSocketSyncer) classifyError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()

	// 超时错误
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return "timeout"
	}

	// 连接关闭错误
	if strings.Contains(errStr, "connection closed") || strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "websocket: close") {
		return "connection_closed"
	}

	// 策略违规错误 - 通常是订阅过多流导致的永久性错误
	if strings.Contains(errStr, "policy violation") || strings.Contains(errStr, "Invalid request") {
		return "policy_violation"
	}

	// 协议错误
	if strings.Contains(errStr, "invalid frame") || strings.Contains(errStr, "protocol error") ||
		strings.Contains(errStr, "unexpected EOF") {
		return "protocol_error"
	}

	// 网络错误
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return "timeout"
		}
		return "network_error"
	}

	return "unknown"
}

// resubscribeConnection 重新订阅连接的数据流
func (s *WebSocketSyncer) resubscribeConnection(connType string) error {
	log.Printf("[WebSocketSyncer] Attempting to resubscribe %s connection", connType)

	// 对于分布式架构，重新订阅意味着重新运行完整的订阅流程
	if connType == "spot" {
		// 重新运行现货分布式订阅
		if err := s.subscribeSpotStreamsDistributed(s.subscribedSymbols); err != nil {
			return fmt.Errorf("failed to resubscribe spot streams: %w", err)
		}
	} else if connType == "futures" {
		// 重新运行期货分布式订阅
		if err := s.subscribeFuturesStreamsDistributed(s.subscribedSymbols); err != nil {
			return fmt.Errorf("failed to resubscribe futures streams: %w", err)
		}
	} else {
		return fmt.Errorf("unknown connection type: %s", connType)
	}

	log.Printf("[WebSocketSyncer] Successfully resubscribed %s connection", connType)
	return nil
}
