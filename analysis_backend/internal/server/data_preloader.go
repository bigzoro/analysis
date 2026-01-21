package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// DataPreloader 数据预加载服务
// 负责定期从API获取数据并保存到数据库，优化推荐系统的响应速度
type DataPreloader struct {
	server *Server
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 配置
	config *DataPreloaderConfig

	// 定时器
	marketDataTicker *time.Ticker
	klineDataTicker  *time.Ticker
	technicalTicker  *time.Ticker
	predictionTicker *time.Ticker

	// 状态
	isRunning bool
	mu        sync.RWMutex
}

// DataPreloaderConfig 预加载服务配置
type DataPreloaderConfig struct {
	// 市场数据更新间隔
	MarketDataInterval time.Duration

	// K线数据更新间隔
	KlineDataInterval time.Duration

	// 技术指标更新间隔
	TechnicalInterval time.Duration

	// 价格预测更新间隔
	PredictionInterval time.Duration

	// K线数据配置
	KlineSymbols    []string // 需要缓存的交易对
	KlineTimeframes []string // 时间周期: 1m, 5m, 15m, 1h, 4h, 1d
	KlineLimit      int      // 每个周期获取的K线数量

	// 技术指标配置
	TechnicalSymbols []string // 需要计算技术指标的交易对

	// 价格预测配置
	PredictionSymbols []string // 需要价格预测的交易对
	PredictionKinds   []string // 交易类型: spot, futures
}

// DefaultDataPreloaderConfig 默认配置
func DefaultDataPreloaderConfig() *DataPreloaderConfig {
	return &DataPreloaderConfig{
		MarketDataInterval: 5 * time.Minute,  // 每5分钟更新市场数据
		KlineDataInterval:  10 * time.Minute, // 每10分钟更新K线数据
		TechnicalInterval:  15 * time.Minute, // 每15分钟更新技术指标
		PredictionInterval: 30 * time.Minute, // 每30分钟更新价格预测

		KlineSymbols: []string{
			"BTCUSD_PERP", "ETHUSD_PERP", "BNBUSD_PERP", "LINKUSD_PERP", "XRPUSD_PERP",
			"SOLUSD_PERP", "DOTUSD_PERP", "DOGEUSD_PERP", "AVAXUSD_PERP", "MATICUSD_PERP",
		},
		KlineTimeframes: []string{"1m", "5m", "15m", "1h", "4h", "1d"},
		KlineLimit:      500,

		TechnicalSymbols: []string{
			"BTCUSD_PERP", "ETHUSD_PERP", "BNBUSD_PERP", "LINKUSD_PERP", "XRPUSD_PERP",
			"SOLUSD_PERP", "DOTUSD_PERP", "DOGEUSD_PERP", "AVAXUSD_PERP", "MATICUSD_PERP",
		},

		PredictionSymbols: []string{
			"BTCUSD_PERP", "ETHUSD_PERP", "BNBUSD_PERP", "LINKUSD_PERP", "XRPUSD_PERP",
			"SOLUSD_PERP", "DOTUSD_PERP", "DOGEUSD_PERP", "AVAXUSD_PERP", "MATICUSD_PERP",
		},
		PredictionKinds: []string{"spot", "futures"},
	}
}

// NewDataPreloader 创建数据预加载服务
func NewDataPreloader(server *Server, config *DataPreloaderConfig) *DataPreloader {
	if config == nil {
		config = DefaultDataPreloaderConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DataPreloader{
		server: server,
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}
}

// Start 启动数据预加载服务
func (dp *DataPreloader) Start() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.isRunning {
		return fmt.Errorf("data preloader is already running")
	}

	dp.isRunning = true
	log.Println("[DataPreloader] Starting data preloader service...")

	// 启动各个定时任务
	dp.startMarketDataUpdater()
	//dp.startKlineDataUpdater()
	//dp.startTechnicalUpdater()
	//dp.startPredictionUpdater()

	log.Println("[DataPreloader] Data preloader service started successfully")
	return nil
}

// Stop 停止数据预加载服务
func (dp *DataPreloader) Stop() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if !dp.isRunning {
		return
	}

	log.Println("[DataPreloader] Stopping data preloader service...")
	dp.isRunning = false

	// 取消上下文
	dp.cancel()

	// 停止定时器
	if dp.marketDataTicker != nil {
		dp.marketDataTicker.Stop()
	}
	if dp.klineDataTicker != nil {
		dp.klineDataTicker.Stop()
	}
	if dp.technicalTicker != nil {
		dp.technicalTicker.Stop()
	}
	if dp.predictionTicker != nil {
		dp.predictionTicker.Stop()
	}

	// 等待所有goroutine完成
	dp.wg.Wait()
	log.Println("[DataPreloader] Data preloader service stopped")
}

// startMarketDataUpdater 启动市场数据更新器
func (dp *DataPreloader) startMarketDataUpdater() {
	dp.marketDataTicker = time.NewTicker(dp.config.MarketDataInterval)

	dp.wg.Add(1)
	go func() {
		defer dp.wg.Done()

		// 立即执行一次
		dp.updateMarketData()

		for {
			select {
			case <-dp.marketDataTicker.C:
				dp.updateMarketData()
			case <-dp.ctx.Done():
				return
			}
		}
	}()
}

// startKlineDataUpdater 启动K线数据更新器
func (dp *DataPreloader) startKlineDataUpdater() {
	dp.klineDataTicker = time.NewTicker(dp.config.KlineDataInterval)

	dp.wg.Add(1)
	go func() {
		defer dp.wg.Done()

		// 立即执行一次
		dp.updateKlineData()

		for {
			select {
			case <-dp.klineDataTicker.C:
				dp.updateKlineData()
			case <-dp.ctx.Done():
				return
			}
		}
	}()
}

// startTechnicalUpdater 启动技术指标更新器
func (dp *DataPreloader) startTechnicalUpdater() {
	dp.technicalTicker = time.NewTicker(dp.config.TechnicalInterval)

	dp.wg.Add(1)
	go func() {
		defer dp.wg.Done()

		// 延迟启动，避免与K线数据更新冲突
		time.Sleep(2 * time.Minute)
		dp.updateTechnicalIndicators()

		for {
			select {
			case <-dp.technicalTicker.C:
				dp.updateTechnicalIndicators()
			case <-dp.ctx.Done():
				return
			}
		}
	}()
}

// startPredictionUpdater 启动价格预测更新器
func (dp *DataPreloader) startPredictionUpdater() {
	dp.predictionTicker = time.NewTicker(dp.config.PredictionInterval)

	dp.wg.Add(1)
	go func() {
		defer dp.wg.Done()

		// 延迟启动，避免与技术指标更新冲突
		time.Sleep(5 * time.Minute)
		dp.updatePricePredictions()

		for {
			select {
			case <-dp.predictionTicker.C:
				dp.updatePricePredictions()
			case <-dp.ctx.Done():
				return
			}
		}
	}()
}

// updateMarketData 更新市场数据
func (dp *DataPreloader) updateMarketData() {
	log.Println("[DataPreloader] Updating market data...")

	for _, kind := range []string{"spot", "futures"} {
		if err := dp.updateMarketDataForKind(kind); err != nil {
			log.Printf("[DataPreloader] Failed to update market data for %s: %v", kind, err)
		}
	}

	log.Println("[DataPreloader] Market data update completed")
}

// updateMarketDataForKind 为指定类型更新市场数据
func (dp *DataPreloader) updateMarketDataForKind(kind string) error {
	// 从数据库获取最新的市场数据（过去24小时）
	now := time.Now().UTC()
	startTime := now.Add(-24 * time.Hour)

	snaps, tops, err := pdb.ListBinanceMarket(dp.server.db.DB(), kind, startTime, now)
	if err != nil {
		return fmt.Errorf("failed to fetch market data from db: %w", err)
	}

	if len(snaps) == 0 {
		log.Printf("[DataPreloader] No recent market data found for %s, skipping update", kind)
		return nil
	}

	// 获取最新的快照数据
	latestSnap := snaps[len(snaps)-1]
	items, exists := tops[latestSnap.ID]
	if !exists || len(items) == 0 {
		return fmt.Errorf("no market data items found for latest snapshot")
	}

	// 只保存前200个
	if len(items) > 200 {
		items = items[:200]
	}

	log.Printf("[DataPreloader] Found %d market data items for %s from database", len(items), kind)

	// 注意：数据已经在数据库中，我们只需要确保缓存是最新的
	// 这里可以添加一些额外的处理逻辑，比如更新缓存等

	return nil
}

// updateKlineData 更新K线数据
func (dp *DataPreloader) updateKlineData() {
	log.Println("[DataPreloader] Updating kline data...")

	for _, symbol := range dp.config.KlineSymbols {
		for _, timeframe := range dp.config.KlineTimeframes {
			if err := dp.updateKlineDataForSymbol(symbol, timeframe); err != nil {
				log.Printf("[DataPreloader] Failed to update kline data for %s %s: %v", symbol, timeframe, err)
			}
		}
	}

	log.Println("[DataPreloader] Kline data update completed")
}

// updateKlineDataForSymbol 为指定交易对和时间周期更新K线数据
func (dp *DataPreloader) updateKlineDataForSymbol(symbol, timeframe string) error {
	// 根据交易对后缀确定API类型
	kind := "spot"
	if strings.HasSuffix(symbol, "USD_PERP") {
		kind = "futures"
	}

	// 检查数据库中是否已有足够的新鲜数据
	if dp.hasSufficientData(symbol, kind, timeframe) {
		return nil
	}

	// 获取K线数据
	klines, err := dp.server.fetchBinanceKlines(dp.ctx, symbol, kind, timeframe, dp.config.KlineLimit)
	if err != nil {
		return fmt.Errorf("failed to fetch klines: %w", err)
	}

	if len(klines) == 0 {
		return fmt.Errorf("no kline data received")
	}

	// 保存到数据库
	if err := dp.saveKlinesToDatabase(symbol, kind, timeframe, klines); err != nil {
		log.Printf("[DataPreloader] Failed to save kline data to database for %s %s: %v", symbol, timeframe, err)
		return fmt.Errorf("failed to save kline data to database: %w", err)
	}

	// 同时保存到Redis缓存以提高查询性能
	cacheKey := fmt.Sprintf("kline:%s:%s:latest", symbol, timeframe)
	data, err := json.Marshal(klines)
	if err != nil {
		log.Printf("[DataPreloader] Failed to marshal kline data for cache: %v", err)
	} else {
		// 使用较长的TTL，因为K线数据变化不频繁
		ttl := 30 * time.Minute
		if err := dp.server.cache.Set(dp.ctx, cacheKey, data, ttl); err != nil {
			log.Printf("[DataPreloader] Failed to cache kline data for %s %s: %v", symbol, timeframe, err)
			// 不返回错误，继续处理
		}
	}

	log.Printf("[DataPreloader] Saved %d kline data points to database for %s %s", len(klines), symbol, timeframe)
	return nil
}

// hasSufficientData 检查数据库中是否已有足够的新鲜数据
func (dp *DataPreloader) hasSufficientData(symbol, kind, interval string) bool {
	gdb := dp.server.db.DB()
	if gdb == nil {
		return false
	}

	// 计算期望的数据点数量
	expectedCount := dp.config.KlineLimit

	// 检查数据库中的记录数量
	var count int64
	query := "SELECT COUNT(*) FROM market_klines WHERE symbol = ? AND kind = ? AND `interval` = ?"
	if err := gdb.Raw(query, symbol, kind, interval).Scan(&count).Error; err != nil {
		log.Printf("[DataPreloader] Failed to check data count for %s %s %s: %v", symbol, kind, interval, err)
		return false
	}

	// 如果数据点数量足够，检查数据的新鲜度
	if count >= int64(expectedCount) {
		// 检查最新数据的更新时间
		var latestUpdate time.Time
		latestQuery := "SELECT MAX(updated_at) FROM market_klines WHERE symbol = ? AND kind = ? AND `interval` = ?"
		if err := gdb.Raw(latestQuery, symbol, kind, interval).Scan(&latestUpdate).Error; err != nil {
			log.Printf("[DataPreloader] Failed to check latest update for %s %s %s: %v", symbol, kind, interval, err)
			return false
		}

		// 如果数据在最近1小时内更新过，认为是新鲜的
		if time.Since(latestUpdate) < time.Hour {
			return true
		}
	}

	return false
}

// updateTechnicalIndicators 更新技术指标
func (dp *DataPreloader) updateTechnicalIndicators() {
	log.Println("[DataPreloader] Updating technical indicators...")

	for _, symbol := range dp.config.TechnicalSymbols {
		// 根据交易对后缀确定API类型
		kind := "spot"
		if strings.HasSuffix(symbol, "USD_PERP") {
			kind = "futures"
		}

		if err := dp.updateTechnicalIndicatorsForSymbol(symbol, kind); err != nil {
			log.Printf("[DataPreloader] Failed to update technical indicators for %s %s: %v", symbol, kind, err)
		}
	}

	log.Println("[DataPreloader] Technical indicators update completed")
}

// updateTechnicalIndicatorsForSymbol 为指定交易对更新技术指标
func (dp *DataPreloader) updateTechnicalIndicatorsForSymbol(symbol, kind string) error {
	// 获取技术指标
	indicators, err := dp.server.CalculateTechnicalIndicators(dp.ctx, symbol, kind)
	if err != nil {
		// 检查是否为无效交易对错误，如果是则跳过不报错
		if strings.Contains(err.Error(), "Invalid symbol") ||
			strings.Contains(err.Error(), "获取K线数据失败") {
			log.Printf("[DataPreloader] Skipping invalid symbol %s %s: %v", symbol, kind, err)
			return nil // 返回nil跳过这个交易对，继续处理其他交易对
		}
		return fmt.Errorf("failed to calculate technical indicators: %w", err)
	}

	if indicators == nil {
		return fmt.Errorf("no technical indicators calculated")
	}

	// 保存到缓存
	cacheKey := fmt.Sprintf("technical:%s:%s:latest", symbol, kind)
	data, err := json.Marshal(indicators)
	if err != nil {
		return fmt.Errorf("failed to marshal technical indicators: %w", err)
	}

	// 技术指标缓存时间稍短，因为需要相对实时
	ttl := 10 * time.Minute
	if err := dp.server.cache.Set(dp.ctx, cacheKey, data, ttl); err != nil {
		log.Printf("[DataPreloader] Failed to cache technical indicators for %s %s: %v", symbol, kind, err)
		// 不返回错误，继续处理
	}

	log.Printf("[DataPreloader] Cached technical indicators for %s %s", symbol, kind)
	return nil
}

// updatePricePredictions 更新价格预测
func (dp *DataPreloader) updatePricePredictions() {
	log.Println("[DataPreloader] Updating price predictions...")

	for _, symbol := range dp.config.PredictionSymbols {
		// 根据交易对后缀确定API类型
		kind := "spot"
		if strings.HasSuffix(symbol, "USD_PERP") {
			kind = "futures"
		}

		if err := dp.updatePricePredictionForSymbol(symbol, kind); err != nil {
			log.Printf("[DataPreloader] Failed to update price prediction for %s %s: %v", symbol, kind, err)
		}
	}

	log.Println("[DataPreloader] Price predictions update completed")
}

// updatePricePredictionForSymbol 为指定交易对更新价格预测
func (dp *DataPreloader) updatePricePredictionForSymbol(symbol, kind string) error {
	// 获取价格预测
	prediction, err := dp.server.GetPricePrediction(dp.ctx, symbol, kind)
	if err != nil {
		// 检查是否为无效交易对错误，如果是则跳过不报错
		if strings.Contains(err.Error(), "Invalid symbol") ||
			strings.Contains(err.Error(), "获取K线数据失败") {
			log.Printf("[DataPreloader] Skipping invalid symbol %s %s: %v", symbol, kind, err)
			return nil // 返回nil跳过这个交易对，继续处理其他交易对
		}
		return fmt.Errorf("failed to calculate price prediction: %w", err)
	}

	if prediction == nil {
		return fmt.Errorf("no price prediction calculated")
	}

	// 保存到缓存
	cacheKey := fmt.Sprintf("prediction:%s:%s:latest", symbol, kind)
	data, err := json.Marshal(prediction)
	if err != nil {
		return fmt.Errorf("failed to marshal price prediction: %w", err)
	}

	// 价格预测缓存时间稍短，因为预测需要相对新鲜
	ttl := 15 * time.Minute
	if err := dp.server.cache.Set(dp.ctx, cacheKey, data, ttl); err != nil {
		log.Printf("[DataPreloader] Failed to cache price prediction for %s %s: %v", symbol, kind, err)
		// 不返回错误，继续处理
	}

	log.Printf("[DataPreloader] Cached price prediction for %s %s", symbol, kind)
	return nil
}

// GetStats 获取预加载服务统计信息
func (dp *DataPreloader) GetStats() map[string]interface{} {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	return map[string]interface{}{
		"is_running":               dp.isRunning,
		"market_data_interval":     dp.config.MarketDataInterval.String(),
		"kline_data_interval":      dp.config.KlineDataInterval.String(),
		"technical_interval":       dp.config.TechnicalInterval.String(),
		"prediction_interval":      dp.config.PredictionInterval.String(),
		"kline_symbols_count":      len(dp.config.KlineSymbols),
		"technical_symbols_count":  len(dp.config.TechnicalSymbols),
		"prediction_symbols_count": len(dp.config.PredictionSymbols),
		"kline_timeframes":         strings.Join(dp.config.KlineTimeframes, ","),
	}
}

// AddSymbol 添加需要预加载的交易对
func (dp *DataPreloader) AddSymbol(symbol string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	// 添加到各个配置列表中（去重）
	addToList := func(list *[]string, item string) {
		for _, existing := range *list {
			if existing == item {
				return
			}
		}
		*list = append(*list, item)
	}

	addToList(&dp.config.KlineSymbols, symbol)
	addToList(&dp.config.TechnicalSymbols, symbol)
	addToList(&dp.config.PredictionSymbols, symbol)

	log.Printf("[DataPreloader] Added symbol %s to preloader configuration", symbol)
}

// RemoveSymbol 移除需要预加载的交易对
func (dp *DataPreloader) RemoveSymbol(symbol string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	// 从各个配置列表中移除
	removeFromList := func(list *[]string, item string) {
		for i, existing := range *list {
			if existing == item {
				*list = append((*list)[:i], (*list)[i+1:]...)
				break
			}
		}
	}

	removeFromList(&dp.config.KlineSymbols, symbol)
	removeFromList(&dp.config.TechnicalSymbols, symbol)
	removeFromList(&dp.config.PredictionSymbols, symbol)

	log.Printf("[DataPreloader] Removed symbol %s from preloader configuration", symbol)
}

// API处理器

// GetDataPreloaderStats 获取数据预加载服务统计信息
// GET /api/data-preloader/stats
func (s *Server) GetDataPreloaderStats(c *gin.Context) {
	if s.dataPreloader == nil {
		c.JSON(200, gin.H{
			"error": "data preloader service not available",
		})
		return
	}

	stats := s.dataPreloader.GetStats()
	c.JSON(200, stats)
}

// AddPreloaderSymbol 添加预加载交易对
// POST /api/data-preloader/symbols
func (s *Server) AddPreloaderSymbol(c *gin.Context) {
	if s.dataPreloader == nil {
		c.JSON(400, gin.H{
			"error": "data preloader service not available",
		})
		return
	}

	var req struct {
		Symbol string `json:"symbol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.ValidationError(c, "request", "无效的请求格式")
		return
	}

	// 验证交易对格式（支持USDT结尾的U本位和USD_PERP结尾的币本位）
	symbol := strings.ToUpper(req.Symbol)
	if !strings.HasSuffix(symbol, "USDT") && !strings.HasSuffix(symbol, "USD_PERP") {
		s.ValidationError(c, "symbol", "交易对必须以USDT结尾（U本位）或USD_PERP结尾（币本位）")
		return
	}

	s.dataPreloader.AddSymbol(strings.ToUpper(req.Symbol))

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("成功添加交易对 %s 到预加载列表", req.Symbol),
	})
}

// RemovePreloaderSymbol 移除预加载交易对
// DELETE /api/data-preloader/symbols/:symbol
func (s *Server) RemovePreloaderSymbol(c *gin.Context) {
	if s.dataPreloader == nil {
		c.JSON(400, gin.H{
			"error": "data preloader service not available",
		})
		return
	}

	symbol := strings.ToUpper(c.Param("symbol"))
	if symbol == "" {
		s.ValidationError(c, "symbol", "交易对不能为空")
		return
	}

	s.dataPreloader.RemoveSymbol(symbol)

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("成功移除交易对 %s 从预加载列表", symbol),
	})
}

// TriggerDataPreloaderUpdate 手动触发数据预加载更新
// POST /api/data-preloader/trigger-update
func (s *Server) TriggerDataPreloaderUpdate(c *gin.Context) {
	if s.dataPreloader == nil {
		c.JSON(400, gin.H{
			"error": "data preloader service not available",
		})
		return
	}

	var req struct {
		DataTypes []string `json:"data_types"` // market, kline, technical, prediction
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.DataTypes = []string{"market", "kline", "technical", "prediction"} // 默认全部更新
	}

	log.Printf("[DataPreloader] Manual data update triggered for types: %v", req.DataTypes)

	// 异步执行更新
	go func() {
		for _, dataType := range req.DataTypes {
			switch dataType {
			case "market":
				s.dataPreloader.updateMarketData()
			case "kline":
				s.dataPreloader.updateKlineData()
			case "technical":
				s.dataPreloader.updateTechnicalIndicators()
			case "prediction":
				s.dataPreloader.updatePricePredictions()
			}
			// 每次更新之间稍作延迟，避免API限流
			time.Sleep(1 * time.Second)
		}
	}()

	c.JSON(200, gin.H{
		"message": fmt.Sprintf("已触发数据更新: %v", req.DataTypes),
	})
}

// saveKlinesToDatabase 将K线数据保存到数据库
func (dp *DataPreloader) saveKlinesToDatabase(symbol, kind, interval string, klines []BinanceKline) error {
	if len(klines) == 0 {
		return fmt.Errorf("no kline data to save")
	}

	gdb := dp.server.db.DB()
	if gdb == nil {
		return fmt.Errorf("database connection is nil")
	}

	// 开启事务
	tx := gdb.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 为每个K线数据创建数据库记录
	for _, kline := range klines {
		marketKline := pdb.MarketKline{
			Symbol:     symbol,
			Kind:       kind,
			Interval:   interval,
			OpenTime:   time.Unix(int64(kline.OpenTime/1000), 0),
			OpenPrice:  kline.Open,
			HighPrice:  kline.High,
			LowPrice:   kline.Low,
			ClosePrice: kline.Close,
			Volume:     kline.Volume,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// 使用ON DUPLICATE KEY UPDATE来避免重复插入
		query := `
			INSERT INTO market_klines
			(symbol, kind, ` + "`interval`" + `, open_time, open_price, high_price, low_price, close_price, volume, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				open_price = VALUES(open_price),
				high_price = VALUES(high_price),
				low_price = VALUES(low_price),
				close_price = VALUES(close_price),
				volume = VALUES(volume),
				updated_at = VALUES(updated_at)
		`

		if err := tx.Exec(query,
			marketKline.Symbol,
			marketKline.Kind,
			marketKline.Interval,
			marketKline.OpenTime,
			marketKline.OpenPrice,
			marketKline.HighPrice,
			marketKline.LowPrice,
			marketKline.ClosePrice,
			marketKline.Volume,
			marketKline.CreatedAt,
			marketKline.UpdatedAt,
		).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to save kline for %s %s %s: %w",
				marketKline.Symbol, marketKline.Kind, marketKline.Interval, err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[DataPreloader] Successfully saved %d kline records for %s %s %s",
		len(klines), symbol, kind, interval)
	return nil
}
