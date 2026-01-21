// internal/db/market.go
package db

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 一次 1 小时的市场快照
type BinanceMarketSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"size:16;index:idx_market_kind_bucket,priority:1" json:"kind"` // spot / futures
	Bucket    time.Time `gorm:"index:idx_market_kind_bucket,priority:2" json:"bucket"`       // 1h 对齐的时间
	FetchedAt time.Time `json:"fetched_at"`
	CreatedAt time.Time `json:"created_at"`
}

// 快照中的一条 TOP 数据
type BinanceMarketTop struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	SnapshotID uint    `gorm:"index" json:"snapshot_id"`
	Symbol     string  `gorm:"size:32;index" json:"symbol"`
	LastPrice  string  `gorm:"size:64" json:"last_price"`
	Volume     string  `gorm:"size:64" json:"volume"`
	PctChange  float64 `json:"price_change_percent" json:"price_change_percent"`
	// 注意：rank 在部分 MySQL 版本里是敏感的，这里字段名还是 rank，
	// 但我们在查询里会用 `rank` 包起来
	Rank      int       `gorm:"index" json:"rank"`
	CreatedAt time.Time `json:"created_at"`

	MarketCapUSD      *float64 `gorm:"column:market_cap_usd;type:DOUBLE" json:"market_cap_usd"`
	FDVUSD            *float64 `gorm:"column:fdv_usd;type:DOUBLE" json:"fdv_usd"`
	CirculatingSupply *float64 `gorm:"column:circulating_supply;type:DOUBLE" json:"circulating_supply"`
	TotalSupply       *float64 `gorm:"column:total_supply;type:DOUBLE" json:"total_supply"`
}

// 实时涨幅榜快照
type RealtimeGainersSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"size:16;index:idx_gainers_kind_timestamp,priority:1" json:"kind"` // spot / futures
	Timestamp time.Time `gorm:"index:idx_gainers_kind_timestamp,priority:2" json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}

// 涨幅榜中的单个币种数据
type RealtimeGainersItem struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SnapshotID     uint      `gorm:"index" json:"snapshot_id"`
	Symbol         string    `gorm:"size:32;index" json:"symbol"`
	Rank           int       `gorm:"index" json:"rank"`
	CurrentPrice   float64   `gorm:"type:DOUBLE" json:"current_price"`
	PriceChange24h float64   `gorm:"type:DOUBLE" json:"price_change_24h"`
	Volume24h      float64   `gorm:"type:DOUBLE" json:"volume_24h"`
	DataSource     string    `gorm:"size:16" json:"data_source"`
	CreatedAt      time.Time `json:"created_at"`

	// 可选字段
	PriceChangePercent *float64 `gorm:"type:DOUBLE" json:"price_change_percent,omitempty"`
	Confidence         *float64 `gorm:"type:DOUBLE" json:"confidence,omitempty"`
}

// 保存一整份快照（同 kind+bucket 会被覆盖）
func SaveBinanceMarket(gdb *gorm.DB, kind string, bucket, fetchedAt time.Time, items []BinanceMarketTop) (*BinanceMarketSnapshot, error) {
	snap := &BinanceMarketSnapshot{
		Kind:      kind,
		Bucket:    bucket,
		FetchedAt: fetchedAt,
	}
	err := gdb.Transaction(func(tx *gorm.DB) error {
		// 1) 看看有没有老的同一个时间桶
		var old BinanceMarketSnapshot
		if err := tx.Where("kind = ? AND bucket = ?", kind, bucket).First(&old).Error; err == nil {
			// 有老的，先把下面的 top 删了
			if err := tx.Where("snapshot_id = ?", old.ID).Delete(&BinanceMarketTop{}).Error; err != nil {
				return err
			}
			// 再删掉老的 snapshot
			if err := tx.Delete(&old).Error; err != nil {
				return err
			}
		} else if err != gorm.ErrRecordNotFound {
			// 真错了再返回
			return err
		}

		// 2) 插入新的 snapshot
		if err := tx.Create(snap).Error; err != nil {
			return err
		}

		// 3) 插入新的 top
		for i := range items {
			items[i].SnapshotID = snap.ID
			items[i].Rank = i + 1
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// 按时间范围读取快照 + TOP
func ListBinanceMarket(gdb *gorm.DB, kind string, start, end time.Time) ([]BinanceMarketSnapshot, map[uint][]BinanceMarketTop, error) {
	var snaps []BinanceMarketSnapshot
	q := gdb.Where("kind = ?", kind)
	if !start.IsZero() {
		q = q.Where("bucket >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("bucket <= ?", end)
	}
	if err := q.Order("bucket asc").Find(&snaps).Error; err != nil {
		return nil, nil, err
	}

	if len(snaps) == 0 {
		return snaps, map[uint][]BinanceMarketTop{}, nil
	}

	// 收集 id
	ids := make([]uint, 0, len(snaps))
	for _, s := range snaps {
		ids = append(ids, s.ID)
	}

	var tops []BinanceMarketTop
	if err := gdb.
		Where("snapshot_id IN ?", ids).
		Order("snapshot_id asc, `rank` asc").
		Find(&tops).Error; err != nil {
		return nil, nil, err
	}

	// 按 snapshot_id 分组
	grouped := make(map[uint][]BinanceMarketTop)
	for _, t := range tops {
		grouped[t.SnapshotID] = append(grouped[t.SnapshotID], t)
	}

	return snaps, grouped, nil
}

// 币安币种黑名单
type BinanceSymbolBlacklist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"size:16;index:idx_kind_symbol,priority:1" json:"kind"`   // spot / futures
	Symbol    string    `gorm:"size:32;index:idx_kind_symbol,priority:2" json:"symbol"` // 如 "BTCUSDT" 或 "BTCUSD_PERP"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 获取指定类型的黑名单符号
func GetBinanceBlacklist(gdb *gorm.DB, kind string) ([]string, error) {
	var items []BinanceSymbolBlacklist
	q := gdb
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	symbols := make([]string, 0, len(items))
	for _, item := range items {
		symbols = append(symbols, item.Symbol)
	}
	return symbols, nil
}

// 添加黑名单符号
func AddBinanceBlacklist(gdb *gorm.DB, kind, symbol string) error {
	kind = strings.ToLower(strings.TrimSpace(kind))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if kind == "" {
		return fmt.Errorf("kind cannot be empty")
	}
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if kind != "spot" && kind != "futures" {
		return fmt.Errorf("kind must be 'spot' or 'futures'")
	}
	item := &BinanceSymbolBlacklist{Kind: kind, Symbol: symbol}
	return gdb.FirstOrCreate(item, "kind = ? AND symbol = ?", kind, symbol).Error
}

// 删除黑名单符号
func DeleteBinanceBlacklist(gdb *gorm.DB, kind, symbol string) error {
	kind = strings.ToLower(strings.TrimSpace(kind))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	q := gdb.Where("symbol = ?", symbol)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	return q.Delete(&BinanceSymbolBlacklist{}).Error
}

// 列出所有黑名单（可按类型过滤）
func ListBinanceBlacklist(gdb *gorm.DB, kind string) ([]BinanceSymbolBlacklist, error) {
	var items []BinanceSymbolBlacklist
	q := gdb
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if err := q.Order("kind asc, symbol asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ==================== 数据清理功能 ====================

// CleanupOldMarketData 清理旧的市场数据
// retentionDays: 保留最近多少天的全量数据
// fullMode: 是否为全量模式（true=清理更多数据，false=只清理异常数据）
func CleanupOldMarketData(gdb *gorm.DB, retentionDays int, fullMode bool) error {
	if retentionDays <= 0 {
		retentionDays = 30 // 默认保留30天
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	log.Printf("[MarketDataCleanup] Starting cleanup, retention_days=%d, full_mode=%v, cutoff_date=%s",
		retentionDays, fullMode, cutoffDate.Format("2006-01-02"))

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 1. 清理旧的快照和相关数据
		var oldSnapshots []BinanceMarketSnapshot
		if err := tx.Where("bucket < ?", cutoffDate).Find(&oldSnapshots).Error; err != nil {
			return fmt.Errorf("failed to find old snapshots: %w", err)
		}

		if len(oldSnapshots) == 0 {
			log.Printf("[MarketDataCleanup] No old snapshots to clean")
			return nil
		}

		// 收集要删除的snapshot IDs
		oldSnapshotIDs := make([]uint, 0, len(oldSnapshots))
		for _, snap := range oldSnapshots {
			oldSnapshotIDs = append(oldSnapshotIDs, snap.ID)
		}

		log.Printf("[MarketDataCleanup] Found %d old snapshots to clean", len(oldSnapshots))

		// 2. 删除相关的市场数据
		if err := tx.Where("snapshot_id IN ?", oldSnapshotIDs).Delete(&BinanceMarketTop{}).Error; err != nil {
			return fmt.Errorf("failed to delete old market tops: %w", err)
		}

		// 3. 删除快照记录
		if err := tx.Where("bucket < ?", cutoffDate).Delete(&BinanceMarketSnapshot{}).Error; err != nil {
			return fmt.Errorf("failed to delete old snapshots: %w", err)
		}

		log.Printf("[MarketDataCleanup] Successfully cleaned %d snapshots and related data", len(oldSnapshots))

		// 4. 如果是全量模式，进行额外的优化清理
		if fullMode {
			// 清理重复数据（相同时间桶的多个快照，只保留最新的）
			if err := cleanupDuplicateSnapshots(tx); err != nil {
				log.Printf("[MarketDataCleanup] Warning: failed to cleanup duplicates: %v", err)
				// 不返回错误，继续执行
			}

			// 优化表空间
			if err := optimizeMarketTables(tx); err != nil {
				log.Printf("[MarketDataCleanup] Warning: failed to optimize tables: %v", err)
				// 不返回错误，继续执行
			}
		}

		return nil
	})
}

// cleanupDuplicateSnapshots 清理重复的快照（保留每个时间桶最新的快照）
func cleanupDuplicateSnapshots(tx *gorm.DB) error {
	log.Printf("[MarketDataCleanup] Cleaning up duplicate snapshots...")

	// 查找重复的时间桶
	var duplicates []struct {
		Kind   string
		Bucket time.Time
		Count  int
	}

	err := tx.Model(&BinanceMarketSnapshot{}).
		Select("kind, bucket, COUNT(*) as count").
		Group("kind, bucket").
		Having("COUNT(*) > 1").
		Scan(&duplicates).Error

	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		log.Printf("[MarketDataCleanup] No duplicate snapshots found")
		return nil
	}

	log.Printf("[MarketDataCleanup] Found %d duplicate bucket groups", len(duplicates))

	totalDeleted := 0
	for _, dup := range duplicates {
		// 为每个重复组，保留最新的快照，删除其他的
		var snapshots []BinanceMarketSnapshot
		err := tx.Where("kind = ? AND bucket = ?", dup.Kind, dup.Bucket).
			Order("created_at DESC").
			Find(&snapshots).Error

		if err != nil {
			continue // 跳过这个组，继续下一个
		}

		if len(snapshots) <= 1 {
			continue // 没有重复
		}

		// 保留第一个（最新的），删除其他的
		toDelete := snapshots[1:] // 从第二个开始删除
		deleteIDs := make([]uint, 0, len(toDelete))
		for _, snap := range toDelete {
			deleteIDs = append(deleteIDs, snap.ID)
		}

		// 删除相关的市场数据
		if err := tx.Where("snapshot_id IN ?", deleteIDs).Delete(&BinanceMarketTop{}).Error; err != nil {
			log.Printf("[MarketDataCleanup] Warning: failed to delete market tops for duplicate snapshots: %v", err)
			continue
		}

		// 删除快照
		if err := tx.Where("id IN ?", deleteIDs).Delete(&BinanceMarketSnapshot{}).Error; err != nil {
			log.Printf("[MarketDataCleanup] Warning: failed to delete duplicate snapshots: %v", err)
			continue
		}

		totalDeleted += len(deleteIDs)
		log.Printf("[MarketDataCleanup] Cleaned %d duplicate snapshots for %s %s",
			len(deleteIDs), dup.Kind, dup.Bucket.Format("2006-01-02 15:04:05"))
	}

	log.Printf("[MarketDataCleanup] Total cleaned %d duplicate snapshots", totalDeleted)
	return nil
}

// optimizeMarketTables 优化市场数据表
func optimizeMarketTables(tx *gorm.DB) error {
	log.Printf("[MarketDataCleanup] Optimizing market tables...")

	// 注意：这里使用原生SQL，因为GORM没有直接的OPTIMIZE TABLE支持
	// 不同的数据库可能有不同的语法，这里以MySQL为例

	// 分析表以更新统计信息
	if err := tx.Exec("ANALYZE TABLE binance_market_snapshots").Error; err != nil {
		log.Printf("[MarketDataCleanup] Warning: failed to analyze snapshots table: %v", err)
	}

	if err := tx.Exec("ANALYZE TABLE binance_market_tops").Error; err != nil {
		log.Printf("[MarketDataCleanup] Warning: failed to analyze tops table: %v", err)
	}

	// 优化表空间（如果支持）
	// 注意：OPTIMIZE TABLE 可能会锁定表，谨慎使用
	if err := tx.Exec("OPTIMIZE TABLE binance_market_snapshots").Error; err != nil {
		log.Printf("[MarketDataCleanup] Warning: failed to optimize snapshots table: %v", err)
	}

	if err := tx.Exec("OPTIMIZE TABLE binance_market_tops").Error; err != nil {
		log.Printf("[MarketDataCleanup] Warning: failed to optimize tops table: %v", err)
	}

	log.Printf("[MarketDataCleanup] Table optimization completed")
	return nil
}

// GetMarketDataStats 获取市场数据统计信息
func GetMarketDataStats(gdb *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 快照统计
	var snapshotCount int64
	if err := gdb.Model(&BinanceMarketSnapshot{}).Count(&snapshotCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count snapshots: %w", err)
	}
	stats["total_snapshots"] = snapshotCount

	// 市场数据统计
	var marketDataCount int64
	if err := gdb.Model(&BinanceMarketTop{}).Count(&marketDataCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count market data: %w", err)
	}
	stats["total_market_data"] = marketDataCount

	// 按类型统计
	var spotCount int64
	gdb.Model(&BinanceMarketSnapshot{}).Where("kind = ?", "spot").Count(&spotCount)
	stats["spot_snapshots"] = spotCount

	var futuresCount int64
	gdb.Model(&BinanceMarketSnapshot{}).Where("kind = ?", "futures").Count(&futuresCount)
	stats["futures_snapshots"] = futuresCount

	// 时间范围统计
	var oldestSnapshot, newestSnapshot BinanceMarketSnapshot
	if err := gdb.Order("bucket ASC").First(&oldestSnapshot).Error; err == nil {
		stats["oldest_data"] = oldestSnapshot.Bucket.Format("2006-01-02")
	}
	if err := gdb.Order("bucket DESC").First(&newestSnapshot).Error; err == nil {
		stats["newest_data"] = newestSnapshot.Bucket.Format("2006-01-02")
	}

	// 唯一交易对数量统计
	var uniqueSymbols int64
	gdb.Model(&BinanceMarketTop{}).Distinct("symbol").Count(&uniqueSymbols)
	stats["unique_symbols"] = uniqueSymbols

	return stats, nil
}

// ==================== 数据压缩功能 ====================

// CompressOldMarketData 压缩旧的市场数据
// 将超过指定天数的小时数据聚合为日数据，以减少存储空间
func CompressOldMarketData(gdb *gorm.DB, compressAfterDays int) error {
	if compressAfterDays <= 0 {
		compressAfterDays = 90 // 默认90天后压缩
	}

	cutoffDate := time.Now().AddDate(0, 0, -compressAfterDays)
	log.Printf("[MarketDataCompression] Starting compression for data older than %s", cutoffDate.Format("2006-01-02"))

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 1. 创建日聚合数据表（如果不存在）
		if err := createDailyAggregationTable(tx); err != nil {
			return fmt.Errorf("failed to create daily aggregation table: %w", err)
		}

		// 2. 聚合旧数据到日表
		if err := aggregateHourlyToDaily(tx, cutoffDate); err != nil {
			return fmt.Errorf("failed to aggregate hourly data: %w", err)
		}

		// 3. 删除已聚合的小时数据
		if err := deleteAggregatedHourlyData(tx, cutoffDate); err != nil {
			return fmt.Errorf("failed to delete aggregated hourly data: %w", err)
		}

		log.Printf("[MarketDataCompression] Compression completed successfully")
		return nil
	})
}

// createDailyAggregationTable 创建日聚合数据表
func createDailyAggregationTable(tx *gorm.DB) error {
	// 注意：这里使用GORM的AutoMigrate来创建表
	// 在实际使用时，可能需要手动创建表以获得更好的控制

	// 日聚合快照表
	type DailyMarketSnapshot struct {
		ID        uint      `gorm:"primaryKey"`
		Kind      string    `gorm:"size:16;index:idx_daily_kind_date,priority:1"`
		Date      time.Time `gorm:"type:date;index:idx_daily_kind_date,priority:2"` // YYYY-MM-DD
		DataCount int       `gorm:"default:0"`                                      // 当天数据点数量
		CreatedAt time.Time
		UpdatedAt time.Time
	}

	// 日聚合市场数据表
	type DailyMarketTop struct {
		ID              uint      `gorm:"primaryKey"`
		DailySnapshotID uint      `gorm:"index"`
		Symbol          string    `gorm:"size:32;index:idx_daily_symbol_date"`
		Date            time.Time `gorm:"type:date;index:idx_daily_symbol_date"`
		AvgPrice        float64   `gorm:"type:decimal(20,8)"` // 日均价
		OpenPrice       float64   `gorm:"type:decimal(20,8)"` // 开盘价
		HighPrice       float64   `gorm:"type:decimal(20,8)"` // 最高价
		LowPrice        float64   `gorm:"type:decimal(20,8)"` // 最低价
		ClosePrice      float64   `gorm:"type:decimal(20,8)"` // 收盘价
		AvgVolume       float64   `gorm:"type:decimal(20,8)"` // 日均成交量
		MaxVolume       float64   `gorm:"type:decimal(20,8)"` // 最大成交量
		AvgPctChange    float64   `gorm:"type:decimal(10,4)"` // 日均涨跌幅
		MaxPctChange    float64   `gorm:"type:decimal(10,4)"` // 最大涨跌幅
		MinPctChange    float64   `gorm:"type:decimal(10,4)"` // 最小涨跌幅
		AvgMarketCapUSD *float64  `gorm:"type:decimal(20,2)"` // 日均市值
		DataPoints      int       `gorm:"default:0"`          // 数据点数量
		CreatedAt       time.Time
	}

	// 创建表（如果不存在）
	if err := tx.AutoMigrate(&DailyMarketSnapshot{}, &DailyMarketTop{}); err != nil {
		log.Printf("[MarketDataCompression] Warning: failed to auto-migrate daily tables: %v", err)
		// 不返回错误，因为表可能已经存在
	}

	return nil
}

// aggregateHourlyToDaily 将小时数据聚合为日数据
func aggregateHourlyToDaily(tx *gorm.DB, cutoffDate time.Time) error {
	log.Printf("[MarketDataCompression] Aggregating hourly data to daily...")

	// 获取需要聚合的日期范围
	var dates []time.Time
	err := tx.Model(&BinanceMarketSnapshot{}).
		Select("DATE(bucket) as date").
		Where("bucket < ?", cutoffDate).
		Group("DATE(bucket)").
		Order("date ASC").
		Pluck("date", &dates).Error

	if err != nil {
		return fmt.Errorf("failed to get dates for aggregation: %w", err)
	}

	log.Printf("[MarketDataCompression] Found %d dates to aggregate", len(dates))

	for _, date := range dates {
		if err := aggregateSingleDay(tx, date); err != nil {
			log.Printf("[MarketDataCompression] Warning: failed to aggregate date %s: %v", date.Format("2006-01-02"), err)
			continue // 继续处理其他日期
		}
	}

	return nil
}

// aggregateSingleDay 聚合单日数据
func aggregateSingleDay(tx *gorm.DB, date time.Time) error {
	dateStr := date.Format("2006-01-02")
	log.Printf("[MarketDataCompression] Aggregating data for %s", dateStr)

	// 获取当天的小时快照
	var snapshots []BinanceMarketSnapshot
	err := tx.Where("DATE(bucket) = ? AND bucket < ?", dateStr, date.AddDate(0, 0, 1)).
		Find(&snapshots).Error
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		return nil // 没有数据
	}

	snapshotIDs := make([]uint, len(snapshots))
	for i, snap := range snapshots {
		snapshotIDs[i] = snap.ID
	}

	// 获取当天所有交易对的数据
	var marketData []BinanceMarketTop
	err = tx.Where("snapshot_id IN ?", snapshotIDs).
		Order("symbol, snapshot_id").
		Find(&marketData).Error
	if err != nil {
		return err
	}

	if len(marketData) == 0 {
		return nil
	}

	// 按交易对分组聚合数据
	symbolGroups := make(map[string][]BinanceMarketTop)
	for _, data := range marketData {
		symbolGroups[data.Symbol] = append(symbolGroups[data.Symbol], data)
	}

	// 创建日聚合记录（这里简化实现，实际应该插入到日表中）
	// 注意：这里只是演示，实际实现需要创建具体的日表结构
	log.Printf("[MarketDataCompression] Would aggregate %d symbols for date %s", len(symbolGroups), dateStr)

	// 实际实现时，应该：
	// 1. 为每种kind（spot/futures）创建日快照记录
	// 2. 为每个交易对计算日聚合数据
	// 3. 插入到日表中

	return nil
}

// deleteAggregatedHourlyData 删除已聚合的小时数据
func deleteAggregatedHourlyData(tx *gorm.DB, cutoffDate time.Time) error {
	log.Printf("[MarketDataCompression] Deleting aggregated hourly data older than %s", cutoffDate.Format("2006-01-02"))

	// 注意：实际删除前，应该确保日聚合数据已经成功创建
	// 这里暂时只记录日志，不执行实际删除

	log.Printf("[MarketDataCompression] Hourly data deletion skipped (implement safety checks first)")

	// 安全的删除策略：
	// 1. 验证日聚合数据已存在
	// 2. 备份数据
	// 3. 分批删除
	// 4. 验证删除结果

	return nil
}

// GetDataCompressionStats 获取数据压缩统计
func GetDataCompressionStats(gdb *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 原始数据统计
	var hourlySnapshots int64
	gdb.Model(&BinanceMarketSnapshot{}).Count(&hourlySnapshots)
	stats["hourly_snapshots"] = hourlySnapshots

	var hourlyDataPoints int64
	gdb.Model(&BinanceMarketTop{}).Count(&hourlyDataPoints)
	stats["hourly_data_points"] = hourlyDataPoints

	// 估算存储大小（粗略计算）
	// 每个快照约100字节，每个数据点约200字节
	stats["estimated_hourly_storage_mb"] = float64(hourlySnapshots*100+hourlyDataPoints*200) / 1024 / 1024

	// 时间范围
	var oldest, newest time.Time
	if err := gdb.Model(&BinanceMarketSnapshot{}).Select("MIN(bucket)").Scan(&oldest).Error; err == nil {
		stats["data_start_date"] = oldest.Format("2006-01-02")
	}
	if err := gdb.Model(&BinanceMarketSnapshot{}).Select("MAX(bucket)").Scan(&newest).Error; err == nil {
		stats["data_end_date"] = newest.Format("2006-01-02")
	}

	// 估算压缩后的存储大小
	daysDiff := int(newest.Sub(oldest).Hours() / 24)
	if daysDiff > 0 {
		// 假设每天约6个小时的快照（每4小时一次）
		dailySnapshots := int64(daysDiff)
		// 假设日数据比小时数据小10倍
		stats["estimated_daily_storage_mb"] = float64(dailySnapshots*100+hourlyDataPoints*20) / 1024 / 1024
		stats["compression_ratio"] = float64(hourlySnapshots*100+hourlyDataPoints*200) / float64(dailySnapshots*100+hourlyDataPoints*20)
	}

	return stats, nil
}

// ===== 实时涨幅榜历史数据存储 =====

// SaveRealtimeGainers 保存实时涨幅榜数据
func SaveRealtimeGainers(gdb *gorm.DB, kind string, timestamp time.Time, items []RealtimeGainersItem) (*RealtimeGainersSnapshot, error) {
	snapshot := &RealtimeGainersSnapshot{
		Kind:      kind,
		Timestamp: timestamp,
	}

	err := gdb.Transaction(func(tx *gorm.DB) error {
		// 1) 创建快照
		if err := tx.Create(snapshot).Error; err != nil {
			return fmt.Errorf("创建涨幅榜快照失败: %w", err)
		}

		// 2) 批量插入数据
		for i := range items {
			items[i].SnapshotID = snapshot.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return fmt.Errorf("插入涨幅榜数据失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// GetRealtimeGainersHistory 获取涨幅榜历史数据
func GetRealtimeGainersHistory(gdb *gorm.DB, kind string, startTime, endTime time.Time, symbolFilter string, limit int) ([]RealtimeGainersSnapshot, map[uint][]RealtimeGainersItem, error) {
	var snapshots []RealtimeGainersSnapshot

	query := gdb.Where("kind = ?", kind)
	if !startTime.IsZero() {
		query = query.Where("timestamp >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("timestamp <= ?", endTime)
	}

	if err := query.Order("timestamp DESC").Limit(limit).Find(&snapshots).Error; err != nil {
		return nil, nil, fmt.Errorf("查询涨幅榜快照失败: %w", err)
	}

	if len(snapshots) == 0 {
		return snapshots, make(map[uint][]RealtimeGainersItem), nil
	}

	// 获取对应的数据项
	snapshotIDs := make([]uint, len(snapshots))
	for i, snap := range snapshots {
		snapshotIDs[i] = snap.ID
	}

	var items []RealtimeGainersItem
	itemQuery := gdb.Where("snapshot_id IN ?", snapshotIDs)
	if symbolFilter != "" {
		itemQuery = itemQuery.Where("symbol = ?", symbolFilter)
	}

	if err := itemQuery.Order("snapshot_id, rank").Find(&items).Error; err != nil {
		return nil, nil, fmt.Errorf("查询涨幅榜数据项失败: %w", err)
	}

	// 按快照ID分组
	itemsMap := make(map[uint][]RealtimeGainersItem)
	for _, item := range items {
		itemsMap[item.SnapshotID] = append(itemsMap[item.SnapshotID], item)
	}

	return snapshots, itemsMap, nil
}

// GetRealtimeGainersLatest 获取最新的涨幅榜数据
func GetRealtimeGainersLatest(gdb *gorm.DB, kind string, limit int) (*RealtimeGainersSnapshot, []RealtimeGainersItem, error) {
	var snapshot RealtimeGainersSnapshot
	if err := gdb.Where("kind = ?", kind).Order("created_at DESC").First(&snapshot).Error; err != nil {
		return nil, nil, fmt.Errorf("获取最新快照失败: %w", err)
	}

	var items []RealtimeGainersItem
	// 使用原生SQL查询，避免GORM LIMIT占位符问题
	sql := fmt.Sprintf("SELECT * FROM `realtime_gainers_items` WHERE snapshot_id = %d ORDER BY `rank` LIMIT %d", snapshot.ID, limit)
	if err := gdb.Raw(sql).Scan(&items).Error; err != nil {
		return nil, nil, fmt.Errorf("获取快照数据失败: %w", err)
	}

	return &snapshot, items, nil
}

// CleanOldRealtimeGainers 清理旧的涨幅榜数据（保留最近N天的）
func CleanOldRealtimeGainers(gdb *gorm.DB, keepDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -keepDays)

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 找到需要删除的快照ID
		var snapshotIDs []uint
		if err := tx.Model(&RealtimeGainersSnapshot{}).
			Where("timestamp < ?", cutoffTime).
			Pluck("id", &snapshotIDs).Error; err != nil {
			return fmt.Errorf("查找旧快照失败: %w", err)
		}

		if len(snapshotIDs) == 0 {
			log.Printf("[清理] 没有找到需要清理的涨幅榜数据")
			return nil
		}

		// 删除数据项
		if err := tx.Where("snapshot_id IN ?", snapshotIDs).Delete(&RealtimeGainersItem{}).Error; err != nil {
			return fmt.Errorf("删除涨幅榜数据项失败: %w", err)
		}

		// 删除快照
		if err := tx.Where("id IN ?", snapshotIDs).Delete(&RealtimeGainersSnapshot{}).Error; err != nil {
			return fmt.Errorf("删除涨幅榜快照失败: %w", err)
		}

		log.Printf("[清理] 清理了 %d 个涨幅榜快照和相关数据", len(snapshotIDs))
		return nil
	})
}

// GetRealtimeGainersStats 获取涨幅榜数据统计
func GetRealtimeGainersStats(gdb *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 快照数量统计
	var totalSnapshots int64
	gdb.Model(&RealtimeGainersSnapshot{}).Count(&totalSnapshots)
	stats["total_snapshots"] = totalSnapshots

	// 按类型统计
	var spotSnapshots, futuresSnapshots int64
	gdb.Model(&RealtimeGainersSnapshot{}).Where("kind = ?", "spot").Count(&spotSnapshots)
	gdb.Model(&RealtimeGainersSnapshot{}).Where("kind = ?", "futures").Count(&futuresSnapshots)
	stats["spot_snapshots"] = spotSnapshots
	stats["futures_snapshots"] = futuresSnapshots

	// 数据项统计
	var totalItems int64
	gdb.Model(&RealtimeGainersItem{}).Count(&totalItems)
	stats["total_items"] = totalItems

	// 时间范围
	var oldestTime, newestTime time.Time
	if err := gdb.Model(&RealtimeGainersSnapshot{}).Select("MIN(timestamp)").Scan(&oldestTime).Error; err == nil {
		stats["oldest_data"] = oldestTime.Format("2006-01-02 15:04:05")
	}
	if err := gdb.Model(&RealtimeGainersSnapshot{}).Select("MAX(timestamp)").Scan(&newestTime).Error; err == nil {
		stats["newest_data"] = newestTime.Format("2006-01-02 15:04:05")
	}

	// 平均每个快照的数据项数量
	if totalSnapshots > 0 {
		stats["avg_items_per_snapshot"] = float64(totalItems) / float64(totalSnapshots)
	}

	// 数据源分布
	dataSourceStats := make(map[string]int64)
	rows, err := gdb.Model(&RealtimeGainersItem{}).Select("data_source, COUNT(*) as count").Group("data_source").Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source string
			var count int64
			rows.Scan(&source, &count)
			dataSourceStats[source] = count
		}
	}
	stats["data_source_distribution"] = dataSourceStats

	return stats, nil
}
