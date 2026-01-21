package db

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

// deduplicateKlines 去重K线数据，基于(symbol, kind, interval, open_time)的组合
func deduplicateKlines(klines []MarketKline) []MarketKline {
	if len(klines) <= 1 {
		return klines
	}

	// 使用map来跟踪唯一记录
	seen := make(map[string]MarketKline)
	var result []MarketKline

	for _, kline := range klines {
		// 创建唯一键
		key := fmt.Sprintf("%s:%s:%s:%d",
			kline.Symbol,
			kline.Kind,
			kline.Interval,
			kline.OpenTime.Unix())

		// 如果这个键还没有出现过，或者新记录的更新时间更晚，则保留
		if existing, exists := seen[key]; !exists {
			seen[key] = kline
			result = append(result, kline)
		} else {
			// 如果已存在相同记录，保留更新时间更晚的那个
			if kline.UpdatedAt.After(existing.UpdatedAt) {
				seen[key] = kline
				// 更新result中的记录（需要找到并替换）
				for i, r := range result {
					if r.Symbol == kline.Symbol && r.Kind == kline.Kind &&
						r.Interval == kline.Interval && r.OpenTime.Equal(kline.OpenTime) {
						result[i] = kline
						break
					}
				}
			}
		}
	}

	return result
}

// insertKlinesBatch 使用INSERT ... ON DUPLICATE KEY UPDATE批量插入K线数据
// 使用更安全的upsert策略避免死锁
func insertKlinesBatch(tx *gorm.DB, klines []MarketKline) error {
	if len(klines) == 0 {
		return nil
	}

	// 构建INSERT ... ON DUPLICATE KEY UPDATE语句
	var valueStrings []string
	var valueArgs []interface{}

	for _, kline := range klines {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime,
			kline.OpenPrice, kline.HighPrice, kline.LowPrice, kline.ClosePrice,
			kline.Volume, kline.QuoteVolume, kline.TradeCount,
			kline.TakerBuyVolume, kline.TakerBuyQuoteVolume,
			kline.CreatedAt, kline.UpdatedAt)
	}

	query := fmt.Sprintf(`INSERT INTO market_klines
		(symbol, kind, `+"`interval`"+`, open_time, open_price, high_price, low_price, close_price, volume, quote_volume, trade_count, taker_buy_volume, taker_buy_quote_volume, created_at, updated_at)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			open_price = VALUES(open_price),
			high_price = VALUES(high_price),
			low_price = VALUES(low_price),
			close_price = VALUES(close_price),
			volume = VALUES(volume),
			quote_volume = VALUES(quote_volume),
			trade_count = VALUES(trade_count),
			taker_buy_volume = VALUES(taker_buy_volume),
			taker_buy_quote_volume = VALUES(taker_buy_quote_volume),
			updated_at = VALUES(updated_at)`,
		strings.Join(valueStrings, ", "))

	return tx.Exec(query, valueArgs...).Error
}

// insertSingleKlineUpsert 使用INSERT ... ON DUPLICATE KEY UPDATE插入单条K线数据
// 这种方式比INSERT IGNORE更安全，避免死锁问题
func insertSingleKlineUpsert(tx *gorm.DB, kline MarketKline) error {
	query := `INSERT INTO market_klines
		(symbol, kind, ` + "`interval`" + `, open_time, open_price, high_price, low_price, close_price, volume, quote_volume, trade_count, taker_buy_volume, taker_buy_quote_volume, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			open_price = VALUES(open_price),
			high_price = VALUES(high_price),
			low_price = VALUES(low_price),
			close_price = VALUES(close_price),
			volume = VALUES(volume),
			quote_volume = VALUES(quote_volume),
			trade_count = VALUES(trade_count),
			taker_buy_volume = VALUES(taker_buy_volume),
			taker_buy_quote_volume = VALUES(taker_buy_quote_volume),
			updated_at = VALUES(updated_at)`

	return tx.Exec(query,
		kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime,
		kline.OpenPrice, kline.HighPrice, kline.LowPrice, kline.ClosePrice,
		kline.Volume, kline.QuoteVolume, kline.TradeCount,
		kline.TakerBuyVolume, kline.TakerBuyQuoteVolume,
		kline.CreatedAt, kline.UpdatedAt).Error
}

// insertSingleKlineIgnore 保留原有函数以向后兼容，但内部使用更安全的upsert
func insertSingleKlineIgnore(tx *gorm.DB, kline MarketKline) error {
	return insertSingleKlineUpsert(tx, kline)
}

// isUniqueConstraintError 检查错误是否是唯一索引冲突
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// MySQL唯一约束错误
	if strings.Contains(errStr, "duplicate entry") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "duplicate key") {
		return true
	}

	// PostgreSQL唯一约束错误
	if strings.Contains(errStr, "unique_violation") ||
		strings.Contains(errStr, "duplicate key value") {
		return true
	}

	// SQLite唯一约束错误
	if strings.Contains(errStr, "unique constraint failed") ||
		strings.Contains(errStr, "constraint failed") {
		return true
	}

	return false
}

// ============================================================================
// K线数据存储和查询
// ============================================================================

// MarketKline K线数据模型
type MarketKline struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Symbol              string    `gorm:"size:32;index:idx_symbol_kind_interval_time,priority:1" json:"symbol"`
	Kind                string    `gorm:"size:16;index:idx_symbol_kind_interval_time,priority:2;index:idx_kind_interval_time,priority:1" json:"kind"`
	Interval            string    `gorm:"size:8;index:idx_symbol_kind_interval_time,priority:3;index:idx_kind_interval_time,priority:2" json:"interval"`
	OpenTime            time.Time `gorm:"index:idx_symbol_kind_interval_time,priority:4;index:idx_symbol_time,priority:2;index:idx_kind_interval_time,priority:3" json:"open_time"`
	OpenPrice           string    `gorm:"size:32" json:"open_price"`
	HighPrice           string    `gorm:"size:32" json:"high_price"`
	LowPrice            string    `gorm:"size:32" json:"low_price"`
	ClosePrice          string    `gorm:"size:32" json:"close_price"`
	Volume              string    `gorm:"size:32" json:"volume"`
	QuoteVolume         *string   `gorm:"size:32" json:"quote_volume,omitempty"`
	TradeCount          *int      `json:"trade_count,omitempty"`
	TakerBuyVolume      *string   `gorm:"size:32" json:"taker_buy_volume,omitempty"`
	TakerBuyQuoteVolume *string   `gorm:"size:32" json:"taker_buy_quote_volume,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// TechnicalIndicatorsCache 技术指标缓存模型
type TechnicalIndicatorsCache struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Symbol       string    `gorm:"size:32;index:idx_symbol_kind_updated,priority:1" json:"symbol"`
	Kind         string    `gorm:"size:16;index:idx_symbol_kind_updated,priority:2" json:"kind"`
	Interval     string    `gorm:"size:8" json:"interval"`
	DataPoints   int       `gorm:"index:uk_symbol_kind_interval_data_points,priority:4" json:"data_points"`
	Indicators   []byte    `gorm:"type:json" json:"indicators"` // JSON数据
	CalculatedAt time.Time `gorm:"index:idx_calculated_at" json:"calculated_at"`
	DataFrom     time.Time `json:"data_from"`
	DataTo       time.Time `json:"data_to"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PriceCache 价格缓存模型
type PriceCache struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Symbol         string    `gorm:"size:32;uniqueIndex:uk_symbol_kind" json:"symbol"`
	Kind           string    `gorm:"size:16;uniqueIndex:uk_symbol_kind" json:"kind"`
	Price          string    `gorm:"size:32" json:"price"`
	PriceChange24h *string   `gorm:"column:price_change_24h;size:16" json:"price_change_24h,omitempty"`
	LastUpdated    time.Time `gorm:"index:idx_last_updated" json:"last_updated"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ============================================================================
// K线数据操作
// ============================================================================

// SaveMarketKlines 批量保存K线数据（覆盖模式，同时间点数据会被替换）
func SaveMarketKlines(gdb *gorm.DB, klines []MarketKline) error {
	if len(klines) == 0 {
		return nil
	}

	// 去重处理：移除重复的K线数据
	uniqueKlines := deduplicateKlines(klines)
	if len(uniqueKlines) != len(klines) {
		log.Printf("[SaveMarketKlines] Removed %d duplicate klines from batch, saving %d unique records",
			len(klines)-len(uniqueKlines), len(uniqueKlines))
	}

	// 优化事务策略：按交易对分组，使用更小的事务范围
	return saveMarketKlinesOptimized(gdb, uniqueKlines)
}

// saveMarketKlinesOptimized 优化版K线保存：按交易对分组事务，进一步分批处理减少死锁风险
func saveMarketKlinesOptimized(gdb *gorm.DB, klines []MarketKline) error {
	if len(klines) == 0 {
		return nil
	}

	log.Printf("[SaveMarketKlines] Inserting %d unique klines with optimized transaction strategy", len(klines))

	// 按交易对分组K线数据
	klinesBySymbol := groupKlinesBySymbol(klines)
	log.Printf("[SaveMarketKlines] Grouped into %d symbol groups for smaller transactions", len(klinesBySymbol))

	totalInserted := 0
	totalErrors := 0
	errorStats := make(map[string]int)

	// 为每个交易对使用独立的小事务，分批处理K线数据
	for symbol, symbolKlines := range klinesBySymbol {
		log.Printf("[SaveMarketKlines] Processing symbol %s: %d klines", symbol, len(symbolKlines))

		symbolInserted, symbolErrors := saveSymbolKlinesInBatches(gdb, symbol, symbolKlines)
		totalInserted += symbolInserted
		totalErrors += symbolErrors

		// 小延迟避免对数据库造成过大压力
		time.Sleep(5 * time.Millisecond)
	}

	log.Printf("[SaveMarketKlines] Successfully processed %d out of %d klines, %d errors",
		totalInserted, len(klines), totalErrors)

	// 输出错误统计摘要
	if totalErrors > 0 {
		log.Printf("[SaveMarketKlines] 错误统计摘要:")
		for errorType, count := range errorStats {
			log.Printf("[SaveMarketKlines]   %s: %d 次", errorType, count)
		}

		if totalInserted > 0 {
			log.Printf("[SaveMarketKlines] 部分成功: %d 条插入成功, %d 条失败", totalInserted, totalErrors)
		} else {
			log.Printf("[SaveMarketKlines] 完全失败: 所有 %d 条记录插入失败", totalErrors)
			return fmt.Errorf("all %d kline insertions failed", totalErrors)
		}
	}

	return nil
}

// saveSymbolKlinesInBatches 将单个交易对的K线数据分批保存，进一步减少死锁风险
func saveSymbolKlinesInBatches(gdb *gorm.DB, symbol string, klines []MarketKline) (int, int) {
	if len(klines) == 0 {
		return 0, 0
	}

	// 设置批次大小：单条插入避免死锁，10条一批平衡性能
	batchSize := 10
	totalInserted := 0
	totalErrors := 0

	// 分批处理
	for i := 0; i < len(klines); i += batchSize {
		end := i + batchSize
		if end > len(klines) {
			end = len(klines)
		}

		batchKlines := klines[i:end]
		batchInserted, batchErrors := saveKlineBatch(gdb, symbol, batchKlines)
		totalInserted += batchInserted
		totalErrors += batchErrors

		// 批次间小延迟
		if i+batchSize < len(klines) {
			time.Sleep(1 * time.Millisecond)
		}
	}

	return totalInserted, totalErrors
}

// saveKlineBatch 保存一批K线数据，使用单个事务
func saveKlineBatch(gdb *gorm.DB, symbol string, klines []MarketKline) (int, int) {
	if len(klines) == 0 {
		return 0, 0
	}

	err := gdb.Transaction(func(tx *gorm.DB) error {
		// 设置时间戳
		now := time.Now()
		for i := range klines {
			klines[i].CreatedAt = now
			klines[i].UpdatedAt = now
		}

		inserted := 0
		errors := 0

		// 单条插入，避免批量操作的死锁风险
		for i, kline := range klines {
			err := insertKlineWithSmartRetry(tx, kline, i+1, len(klines))
			if err != nil {
				errors++
				errorType := classifyDatabaseError(err)
				if errorType == "deadlock" {
					log.Printf("[SaveMarketKlines] 🔴 死锁错误 %s %d/%d: %s %s %s %v",
						symbol, i+1, len(klines), kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime)
				}
			} else {
				inserted++
			}
		}

		if inserted == 0 && len(klines) > 0 {
			return fmt.Errorf("all %d kline insertions failed for symbol %s", len(klines), symbol)
		}

		return nil
	})

	if err != nil {
		log.Printf("[SaveMarketKlines] Transaction failed for symbol %s batch: %v", symbol, err)
		return 0, len(klines)
	}

	return len(klines), 0 // 假设事务成功，所有记录都插入了
}

// groupKlinesBySymbol 按交易对分组K线数据
func groupKlinesBySymbol(klines []MarketKline) map[string][]MarketKline {
	groups := make(map[string][]MarketKline)

	for _, kline := range klines {
		key := kline.Symbol + "_" + kline.Kind // 使用 symbol_kind 作为分组键
		groups[key] = append(groups[key], kline)
	}

	return groups
}

// insertKlineWithSmartRetry 使用智能重试策略插入单条K线数据
func insertKlineWithSmartRetry(tx *gorm.DB, kline MarketKline, index, total int) error {
	maxRetries := 10                    // 进一步增加最大重试次数
	baseDelay := 500 * time.Millisecond // 增加基础延迟

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := insertSingleKlineUpsert(tx, kline)
		if err == nil {
			// 成功插入
			if attempt > 1 {
				log.Printf("[SaveMarketKlines] Successfully inserted kline after %d attempts for %s %s %s %v",
					attempt, kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime)
			}
			return nil
		}

		// 分析错误类型
		errorType := classifyDatabaseError(err)

		// 根据错误类型决定是否重试
		if !isRetryableError(errorType) || attempt == maxRetries {
			// 不可重试的错误或已达到最大重试次数
			log.Printf("[SaveMarketKlines] Failed to insert kline after %d attempts for %s %s %s %v: %v (error type: %s)",
				attempt, kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime, err, errorType)
			return err
		}

		// 计算重试延迟（指数退避 + 随机抖动）
		backoffDelay := calculateBackoffDelay(attempt, errorType, baseDelay)
		log.Printf("[SaveMarketKlines] Database error detected (%s), retrying %d/%d after %v for %s %s %s %v: %v",
			errorType, attempt, maxRetries, backoffDelay, kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime, err)

		time.Sleep(backoffDelay)
	}

	// 不应该到达这里
	return fmt.Errorf("unexpected error in insertKlineWithSmartRetry")
}

// classifyDatabaseError 分类数据库错误类型
func classifyDatabaseError(err error) string {
	if err == nil {
		return "none"
	}

	errMsg := strings.ToLower(err.Error())

	// 死锁错误
	if strings.Contains(errMsg, "1213") || strings.Contains(errMsg, "deadlock") ||
		strings.Contains(errMsg, "try restarting transaction") || strings.Contains(errMsg, "40001") {
		return "deadlock"
	}

	// 锁等待超时
	if strings.Contains(errMsg, "lock wait timeout") || strings.Contains(errMsg, "1205") {
		return "lock_timeout"
	}

	// 连接错误
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "dial tcp") ||
		strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "connection refused") {
		return "connection"
	}

	// 网络超时
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "i/o timeout") {
		return "network_timeout"
	}

	// 唯一约束冲突（通常不应该重试）
	if strings.Contains(errMsg, "duplicate entry") || strings.Contains(errMsg, "unique constraint") {
		return "unique_violation"
	}

	// 其他服务器错误
	if strings.Contains(errMsg, "server error") || strings.Contains(errMsg, "internal server error") {
		return "server_error"
	}

	// 未知错误
	return "unknown"
}

// isRetryableError 判断错误是否可以重试
func isRetryableError(errorType string) bool {
	switch errorType {
	case "deadlock", "lock_timeout", "connection", "network_timeout", "server_error":
		return true
	case "unique_violation", "none", "unknown":
		return false
	default:
		return false
	}
}

// calculateBackoffDelay 根据错误类型和尝试次数计算重试延迟
func calculateBackoffDelay(attempt int, errorType string, baseDelay time.Duration) time.Duration {
	var multiplier float64

	// 根据错误类型设置不同的基础乘数
	switch errorType {
	case "deadlock":
		// 死锁需要较长的等待时间，避免立即再次冲突
		multiplier = 2.0
	case "lock_timeout":
		// 锁超时也需要较长等待
		multiplier = 1.8
	case "connection", "network_timeout":
		// 网络问题可以使用较短的指数退避
		multiplier = 1.5
	case "server_error":
		// 服务器错误使用中等延迟
		multiplier = 1.3
	default:
		multiplier = 1.2
	}

	// 指数退避：delay = baseDelay * multiplier^attempt
	delay := time.Duration(float64(baseDelay) * math.Pow(multiplier, float64(attempt-1)))

	// 添加随机抖动，避免惊群效应（±25%）
	jitter := time.Duration(float64(delay) * 0.25 * (2.0*rand.Float64() - 1.0))
	delay += jitter

	// 设置最大延迟上限（死锁情况下允许更长的等待）
	maxDelay := 15 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}

	// 设置最小延迟下限
	minDelay := 100 * time.Millisecond
	if delay < minDelay {
		delay = minDelay
	}

	return delay
}

// GetMarketKlines 获取K线数据（优先从数据库查询，缺失时返回空）
func GetMarketKlines(gdb *gorm.DB, symbol, kind, interval string, startTime, endTime *time.Time, limit int) ([]MarketKline, error) {
	query := gdb.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, kind, interval)

	if startTime != nil {
		query = query.Where("open_time >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("open_time <= ?", *endTime)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	query = query.Order("open_time DESC")

	var klines []MarketKline
	if err := query.Find(&klines).Error; err != nil {
		return nil, fmt.Errorf("failed to query klines: %w", err)
	}

	// 反转顺序，从旧到新
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	return klines, nil
}

// GetLatestKline 获取最新的K线数据
func GetLatestKline(gdb *gorm.DB, symbol, kind, interval string) (*MarketKline, error) {
	var kline MarketKline
	err := gdb.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, kind, interval).
		Order("open_time DESC").
		First(&kline).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有找到数据
		}
		return nil, fmt.Errorf("failed to get latest kline: %w", err)
	}

	return &kline, nil
}

// IsKlineDataFresh 检查K线数据是否新鲜（是否有最近的数据）
func IsKlineDataFresh(gdb *gorm.DB, symbol, kind, interval string, maxAge time.Duration) (bool, error) {
	var latest MarketKline
	err := gdb.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, kind, interval).
		Order("open_time DESC").
		First(&latest).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // 没有数据，肯定不新鲜
		}
		return false, fmt.Errorf("failed to check kline freshness: %w", err)
	}

	// 检查数据是否在允许的时间范围内
	return time.Since(latest.OpenTime) <= maxAge, nil
}

// ============================================================================
// 技术指标缓存操作
// ============================================================================

// SaveTechnicalIndicatorsCache 保存技术指标缓存
func SaveTechnicalIndicatorsCache(gdb *gorm.DB, cache *TechnicalIndicatorsCache) error {
	// 使用ON DUPLICATE KEY UPDATE处理重复数据
	sql := fmt.Sprintf(`
		INSERT INTO technical_indicators_caches (
			symbol, kind, %s, data_points, indicators,
			calculated_at, data_from, data_to, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE
			indicators = VALUES(indicators),
			calculated_at = VALUES(calculated_at),
			data_from = VALUES(data_from),
			data_to = VALUES(data_to),
			updated_at = NOW(3)
	`, "`interval`")

	return gdb.Exec(sql,
		cache.Symbol, cache.Kind, cache.Interval, cache.DataPoints, cache.Indicators,
		cache.CalculatedAt, cache.DataFrom, cache.DataTo,
	).Error
}

// GetTechnicalIndicatorsCache 获取技术指标缓存
func GetTechnicalIndicatorsCache(gdb *gorm.DB, symbol, kind, interval string, dataPoints int) (*TechnicalIndicatorsCache, error) {
	var cache TechnicalIndicatorsCache
	err := gdb.Where("symbol = ? AND kind = ? AND `interval` = ? AND data_points = ?",
		symbol, kind, interval, dataPoints).
		Order("calculated_at DESC").
		First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有缓存数据
		}
		return nil, fmt.Errorf("failed to get technical indicators cache: %w", err)
	}

	return &cache, nil
}

// IsTechnicalIndicatorsCacheFresh 检查技术指标缓存是否新鲜
func IsTechnicalIndicatorsCacheFresh(gdb *gorm.DB, symbol, kind, interval string, dataPoints int, maxAge time.Duration) (bool, error) {
	var cache TechnicalIndicatorsCache
	err := gdb.Where("symbol = ? AND kind = ? AND `interval` = ? AND data_points = ?",
		symbol, kind, interval, dataPoints).
		Order("calculated_at DESC").
		First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // 没有缓存
		}
		return false, fmt.Errorf("failed to check cache freshness: %w", err)
	}

	return time.Since(cache.CalculatedAt) <= maxAge, nil
}

// ============================================================================
// 价格缓存操作
// ============================================================================

// SavePriceCache 保存价格缓存
func SavePriceCache(gdb *gorm.DB, cache *PriceCache) error {
	return gdb.Exec(`
		INSERT INTO price_caches (
			symbol, kind, price, price_change_24h, last_updated, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE
			price = VALUES(price),
			price_change_24h = VALUES(price_change_24h),
			last_updated = VALUES(last_updated),
			updated_at = NOW(3)
	`,
		cache.Symbol, cache.Kind, cache.Price, cache.PriceChange24h, cache.LastUpdated,
	).Error
}

// GetPriceCache 获取价格缓存
func GetPriceCache(gdb *gorm.DB, symbol, kind string) (*PriceCache, error) {
	var cache PriceCache
	err := gdb.Where("symbol = ? AND kind = ?", symbol, kind).First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有缓存数据，这是正常情况，不记录错误日志
		}
		return nil, fmt.Errorf("failed to get price cache: %w", err)
	}

	return &cache, nil
}

// IsPriceCacheFresh 检查价格缓存是否新鲜
func IsPriceCacheFresh(gdb *gorm.DB, symbol, kind string, maxAge time.Duration) (bool, error) {
	var cache PriceCache
	err := gdb.Where("symbol = ? AND kind = ?", symbol, kind).First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // 没有缓存
		}
		return false, fmt.Errorf("failed to check price cache freshness: %w", err)
	}

	return time.Since(cache.LastUpdated) <= maxAge, nil
}

// ============================================================================
// 数据清理操作
// ============================================================================

// CleanupOldKlineData 清理过期的K线数据
func CleanupOldKlineData(gdb *gorm.DB, interval string, retentionDays int) error {
	if retentionDays <= 0 {
		return nil // 不清理
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := gdb.Where("`interval` = ? AND open_time < ?", interval, cutoffDate).
		Delete(&MarketKline{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old kline data: %w", result.Error)
	}

	// 记录清理的行数
	if result.RowsAffected > 0 {
		// 这里可以添加日志记录清理的行数
	}

	return nil
}

// CleanupOldTechnicalIndicatorsCache 清理过期的技术指标缓存
func CleanupOldTechnicalIndicatorsCache(gdb *gorm.DB, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	result := gdb.Where("calculated_at < ?", cutoffDate).
		Delete(&TechnicalIndicatorsCache{})

	return result.Error
}

// GetKlineDataStats 获取K线数据统计信息
func GetKlineDataStats(gdb *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// K线数据统计
	var totalKlines int64
	gdb.Model(&MarketKline{}).Count(&totalKlines)
	stats["total_klines"] = totalKlines

	// 按间隔统计
	var intervalStats []struct {
		Interval string `json:"interval"`
		Count    int64  `json:"count"`
	}
	gdb.Model(&MarketKline{}).
		Select("`interval`, COUNT(*) as count").
		Group("`interval`").
		Scan(&intervalStats)
	stats["interval_stats"] = intervalStats

	// 技术指标缓存统计
	var totalCache int64
	gdb.Model(&TechnicalIndicatorsCache{}).Count(&totalCache)
	stats["total_technical_cache"] = totalCache

	// 价格缓存统计
	var totalPriceCache int64
	gdb.Model(&PriceCache{}).Count(&totalPriceCache)
	stats["total_price_cache"] = totalPriceCache

	// 时间范围统计
	var oldestKline, newestKline MarketKline
	if err := gdb.Order("open_time ASC").First(&oldestKline).Error; err == nil {
		stats["oldest_kline"] = oldestKline.OpenTime.Format("2006-01-02")
	}
	if err := gdb.Order("open_time DESC").First(&newestKline).Error; err == nil {
		stats["newest_kline"] = newestKline.OpenTime.Format("2006-01-02")
	}

	return stats, nil
}

// ============================================================================
// 特征数据缓存操作
// ============================================================================

// SaveFeatureCache 保存特征数据缓存
func SaveFeatureCache(gdb *gorm.DB, cache *FeatureCache) error {
	// 使用ON DUPLICATE KEY UPDATE处理重复数据
	sql := `
		INSERT INTO feature_cache (
			symbol, features, computed_at, expires_at,
			feature_count, quality_score, source, time_window, data_points,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE
			features = VALUES(features),
			computed_at = VALUES(computed_at),
			expires_at = VALUES(expires_at),
			feature_count = VALUES(feature_count),
			quality_score = VALUES(quality_score),
			source = VALUES(source),
			data_points = VALUES(data_points),
			updated_at = NOW(3)
	`

	return gdb.Exec(sql,
		cache.Symbol,       // symbol
		cache.Features,     // features
		cache.ComputedAt,   // computed_at
		cache.ExpiresAt,    // expires_at
		cache.FeatureCount, // feature_count
		cache.QualityScore, // quality_score
		cache.Source,       // source
		cache.TimeWindow,   // time_window
		cache.DataPoints,   // data_points
	).Error
}

// GetFeatureCache 获取特征数据缓存
func GetFeatureCache(gdb *gorm.DB, symbol string, timeWindow int) (*FeatureCache, error) {
	var cache FeatureCache
	err := gdb.Where("symbol = ? AND time_window = ? AND expires_at > NOW()",
		symbol, timeWindow).
		Order("computed_at DESC").
		First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有缓存数据
		}
		return nil, fmt.Errorf("failed to get feature cache: %w", err)
	}

	return &cache, nil
}

// IsFeatureCacheFresh 检查特征缓存是否新鲜
func IsFeatureCacheFresh(gdb *gorm.DB, symbol string, timeWindow int, maxAge time.Duration) (bool, error) {
	var cache FeatureCache
	err := gdb.Where("symbol = ? AND time_window = ?",
		symbol, timeWindow).
		Order("computed_at DESC").
		First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check feature cache freshness: %w", err)
	}

	// 检查是否在允许的时间范围内
	return time.Since(cache.ComputedAt) <= maxAge, nil
}

// CleanupExpiredFeatureCache 清理过期的特征缓存
func CleanupExpiredFeatureCache(gdb *gorm.DB) error {
	result := gdb.Where("expires_at < NOW()").Delete(&FeatureCache{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired feature cache: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("[FeatureCache] 清理了 %d 条过期的特征缓存记录", result.RowsAffected)
	}

	return nil
}

// GetFeatureCacheStats 获取特征缓存统计信息
func GetFeatureCacheStats(gdb *gorm.DB) (map[string]interface{}, error) {
	var stats struct {
		TotalRecords    int64
		ExpiredRecords  int64
		FreshRecords    int64
		AvgQualityScore float64
		AvgFeatureCount float64
	}

	// 总记录数
	gdb.Model(&FeatureCache{}).Count(&stats.TotalRecords)

	// 过期记录数
	gdb.Model(&FeatureCache{}).Where("expires_at < NOW()").Count(&stats.ExpiredRecords)

	// 新鲜记录数
	stats.FreshRecords = stats.TotalRecords - stats.ExpiredRecords

	// 平均质量评分
	gdb.Model(&FeatureCache{}).Where("expires_at > NOW()").Select("COALESCE(AVG(quality_score), 0)").Scan(&stats.AvgQualityScore)

	// 平均特征数量
	gdb.Model(&FeatureCache{}).Where("expires_at > NOW()").Select("COALESCE(AVG(feature_count), 0)").Scan(&stats.AvgFeatureCount)

	return map[string]interface{}{
		"total_records":     stats.TotalRecords,
		"expired_records":   stats.ExpiredRecords,
		"fresh_records":     stats.FreshRecords,
		"avg_quality_score": stats.AvgQualityScore,
		"avg_feature_count": stats.AvgFeatureCount,
	}, nil
}

// ============================================================================
// ML模型存储操作
// ============================================================================

// SaveMLModel 保存ML模型
func SaveMLModel(gdb *gorm.DB, model *MLModel) error {
	// 使用ON DUPLICATE KEY UPDATE处理重复数据
	sql := `
		INSERT INTO ml_models (
			symbol, model_type, model_name, model_data, performance,
			trained_at, expires_at, training_samples, feature_count,
			accuracy, ` + "`precision`" + `, ` + "`recall`" + `, f1_score, auc,
			sharpe_ratio, max_drawdown, win_rate, profit_factor,
			status, version, description, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE
			model_data = VALUES(model_data),
			performance = VALUES(performance),
			trained_at = VALUES(trained_at),
			expires_at = VALUES(expires_at),
			training_samples = VALUES(training_samples),
			feature_count = VALUES(feature_count),
			accuracy = VALUES(accuracy),
			` + "`precision`" + ` = VALUES(` + "`precision`" + `),
			` + "`recall`" + ` = VALUES(` + "`recall`" + `),
			f1_score = VALUES(f1_score),
			auc = VALUES(auc),
			sharpe_ratio = VALUES(sharpe_ratio),
			max_drawdown = VALUES(max_drawdown),
			win_rate = VALUES(win_rate),
			profit_factor = VALUES(profit_factor),
			status = VALUES(status),
			version = VALUES(version),
			description = VALUES(description),
			updated_at = NOW(3)
	`

	return gdb.Exec(sql,
		model.Symbol,          // symbol
		model.ModelType,       // model_type
		model.ModelName,       // model_name
		model.ModelData,       // model_data
		model.Performance,     // performance
		model.TrainedAt,       // trained_at
		model.ExpiresAt,       // expires_at
		model.TrainingSamples, // training_samples
		model.FeatureCount,    // feature_count
		model.Accuracy,        // accuracy
		model.Precision,       // precision
		model.Recall,          // recall
		model.F1Score,         // f1_score
		model.AUC,             // auc
		model.SharpeRatio,     // sharpe_ratio
		model.MaxDrawdown,     // max_drawdown
		model.WinRate,         // win_rate
		model.ProfitFactor,    // profit_factor
		model.Status,          // status
		model.Version,         // version
		model.Description,     // description
	).Error
}

// GetMLModel 获取ML模型
func GetMLModel(gdb *gorm.DB, symbol, modelType string) (*MLModel, error) {
	var model MLModel
	err := gdb.Where("symbol = ? AND model_type = ? AND expires_at > NOW() AND status = 'active'",
		symbol, modelType).
		Order("version DESC, trained_at DESC").
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有找到模型
		}
		return nil, fmt.Errorf("failed to get ML model: %w", err)
	}

	return &model, nil
}

// GetMLModelsBySymbol 获取指定交易对的所有模型
func GetMLModelsBySymbol(gdb *gorm.DB, symbol string, includeExpired bool) ([]MLModel, error) {
	var models []MLModel
	query := gdb.Where("symbol = ?", symbol)

	if !includeExpired {
		query = query.Where("expires_at > NOW()")
	}

	err := query.Order("trained_at DESC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ML models by symbol: %w", err)
	}

	return models, nil
}

// GetBestMLModels 获取表现最好的模型
func GetBestMLModels(gdb *gorm.DB, modelType string, limit int) ([]MLModel, error) {
	var models []MLModel
	err := gdb.Where("model_type = ? AND expires_at > NOW() AND status = 'active'", modelType).
		Order("accuracy DESC, trained_at DESC").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get best ML models: %w", err)
	}

	return models, nil
}

// UpdateMLModelStatus 更新模型状态
func UpdateMLModelStatus(gdb *gorm.DB, symbol, modelType string, status string) error {
	return gdb.Model(&MLModel{}).
		Where("symbol = ? AND model_type = ?", symbol, modelType).
		Update("status", status).Error
}

// CleanupExpiredMLModels 清理过期的ML模型
func CleanupExpiredMLModels(gdb *gorm.DB) error {
	result := gdb.Where("expires_at < NOW()").Delete(&MLModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired ML models: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("[MLModel] 清理了 %d 条过期的ML模型记录", result.RowsAffected)
	}

	return nil
}

// GetMLModelStats 获取ML模型统计信息
func GetMLModelStats(gdb *gorm.DB) (map[string]interface{}, error) {
	var stats struct {
		TotalModels        int64
		ActiveModels       int64
		ExpiredModels      int64
		AvgAccuracy        float64
		AvgTrainingSamples float64
		BestAccuracy       float64
		WorstAccuracy      float64
	}

	// 总模型数
	gdb.Model(&MLModel{}).Count(&stats.TotalModels)

	// 活跃模型数
	gdb.Model(&MLModel{}).Where("expires_at > NOW() AND status = 'active'").Count(&stats.ActiveModels)

	// 过期模型数
	stats.ExpiredModels = stats.TotalModels - stats.ActiveModels

	// 平均准确率
	gdb.Model(&MLModel{}).Where("expires_at > NOW() AND status = 'active'").
		Select("COALESCE(AVG(accuracy), 0)").Scan(&stats.AvgAccuracy)

	// 平均训练样本数
	gdb.Model(&MLModel{}).Where("expires_at > NOW() AND status = 'active'").
		Select("COALESCE(AVG(training_samples), 0)").Scan(&stats.AvgTrainingSamples)

	// 最佳准确率
	gdb.Model(&MLModel{}).Where("expires_at > NOW() AND status = 'active'").
		Select("MAX(accuracy)").Scan(&stats.BestAccuracy)

	// 最差准确率
	gdb.Model(&MLModel{}).Where("expires_at > NOW() AND status = 'active'").
		Select("MIN(accuracy)").Scan(&stats.WorstAccuracy)

	// 按模型类型统计
	var modelTypeStats []struct {
		ModelType   string
		Count       int64
		AvgAccuracy float64
	}

	gdb.Model(&MLModel{}).
		Select("model_type, COUNT(*) as count, AVG(accuracy) as avg_accuracy").
		Where("expires_at > NOW() AND status = 'active'").
		Group("model_type").
		Find(&modelTypeStats)

	modelTypeMap := make(map[string]interface{})
	for _, stat := range modelTypeStats {
		modelTypeMap[stat.ModelType] = map[string]interface{}{
			"count":        stat.Count,
			"avg_accuracy": stat.AvgAccuracy,
		}
	}

	return map[string]interface{}{
		"total_models":         stats.TotalModels,
		"active_models":        stats.ActiveModels,
		"expired_models":       stats.ExpiredModels,
		"avg_accuracy":         stats.AvgAccuracy,
		"avg_training_samples": stats.AvgTrainingSamples,
		"best_accuracy":        stats.BestAccuracy,
		"worst_accuracy":       stats.WorstAccuracy,
		"model_types":          modelTypeMap,
	}, nil
}
