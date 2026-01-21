package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ==================== 24小时统计历史数据操作 ====================

// Save24hStatsHistory 批量保存24小时统计历史数据
// 支持时间序列数据存储，每个时间窗口保存一条记录
func Save24hStatsHistory(gdb *gorm.DB, stats []Binance24hStatsHistory) error {
	if len(stats) == 0 {
		return nil
	}

	// 去重处理：移除重复的历史数据（基于时间窗口）
	uniqueStats := deduplicate24hStatsHistory(stats)
	if len(uniqueStats) != len(stats) {
		log.Printf("[Save24hStatsHistory] Removed %d duplicate history stats, saving %d unique records",
			len(stats)-len(uniqueStats), len(uniqueStats))
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, stat := range uniqueStats {
			// 检查该时间窗口是否已存在记录
			var existingCount int64
			checkErr := tx.Table("binance_24h_stats_history").Where(
				"symbol = ? AND market_type = ? AND window_start = ?",
				stat.Symbol, stat.MarketType, stat.WindowStart,
			).Count(&existingCount).Error

			if checkErr != nil {
				log.Printf("[Save24hStatsHistory] Failed to check existing history stats for %s %s %v: %v",
					stat.Symbol, stat.MarketType, stat.WindowStart, checkErr)
				continue
			}

			// 设置时间戳
			now := time.Now()
			if stat.CreatedAt.IsZero() {
				stat.CreatedAt = now
			}

			// 只在时间窗口不存在时插入，避免重复数据
			if existingCount == 0 {
				if err := tx.Create(&stat).Error; err != nil {
					log.Printf("[Save24hStatsHistory] Failed to insert history stats for %s %s %v: %v",
						stat.Symbol, stat.MarketType, stat.WindowStart, err)
					continue
				}
				log.Printf("[Save24hStatsHistory] Inserted history stats: %s %s %v",
					stat.Symbol, stat.MarketType, stat.WindowStart)
			} else {
				log.Printf("[Save24hStatsHistory] Skipped duplicate history stats: %s %s %v",
					stat.Symbol, stat.MarketType, stat.WindowStart)
			}
		}

		return nil
	})
}

// Get24hStatsHistory 查询指定时间范围的历史统计数据
func Get24hStatsHistory(gdb *gorm.DB, symbol, marketType string, startTime, endTime time.Time) ([]Binance24hStatsHistory, error) {
	var stats []Binance24hStatsHistory

	query := gdb.Table("binance_24h_stats_history").Where(
		"symbol = ? AND market_type = ? AND window_start >= ? AND window_end <= ?",
		symbol, marketType, startTime, endTime,
	).Order("window_start")

	err := query.Find(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get 24h stats history: %w", err)
	}

	return stats, nil
}

// GetLatest24hStatsHistory 获取最新的历史统计数据
func GetLatest24hStatsHistory(gdb *gorm.DB, symbol, marketType string) (*Binance24hStatsHistory, error) {
	var stat Binance24hStatsHistory

	err := gdb.Table("binance_24h_stats_history").Where(
		"symbol = ? AND market_type = ?",
		symbol, marketType,
	).Order("window_start DESC").First(&stat).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有找到记录
		}
		return nil, fmt.Errorf("failed to get latest 24h stats history: %w", err)
	}

	return &stat, nil
}

// Get24hStatsHistoryByTimeWindow 根据时间窗口获取历史统计数据
func Get24hStatsHistoryByTimeWindow(gdb *gorm.DB, symbol, marketType string, windowStart time.Time) (*Binance24hStatsHistory, error) {
	var stat Binance24hStatsHistory

	err := gdb.Table("binance_24h_stats_history").Where(
		"symbol = ? AND market_type = ? AND window_start = ?",
		symbol, marketType, windowStart,
	).First(&stat).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有找到记录
		}
		return nil, fmt.Errorf("failed to get 24h stats history by time window: %w", err)
	}

	return &stat, nil
}

// DeleteExpired24hStatsHistory 删除过期历史数据
func DeleteExpired24hStatsHistory(gdb *gorm.DB, beforeTime time.Time) error {
	result := gdb.Table("binance_24h_stats_history").Where("window_start < ?", beforeTime).Delete(&Binance24hStatsHistory{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete expired 24h stats history: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("[DeleteExpired24hStatsHistory] Deleted %d expired history records before %v",
			result.RowsAffected, beforeTime)
	}

	return nil
}

// Get24hStatsHistoryStats 获取历史表统计信息
func Get24hStatsHistoryStats(gdb *gorm.DB) (map[string]interface{}, error) {
	var stats struct {
		TotalRecords     int64     `json:"total_records"`
		OldestRecord     time.Time `json:"oldest_record"`
		NewestRecord     time.Time `json:"newest_record"`
		UniqueSymbols    int64     `json:"unique_symbols"`
		UniqueMarkets    int64     `json:"unique_markets"`
		AvgRecordsPerDay float64   `json:"avg_records_per_day"`
	}

	// 总记录数
	gdb.Table("binance_24h_stats_history").Count(&stats.TotalRecords)

	// 最旧和最新记录
	var oldest, newest struct {
		WindowStart time.Time
	}
	gdb.Table("binance_24h_stats_history").Select("MIN(window_start) as window_start").Scan(&oldest)
	gdb.Table("binance_24h_stats_history").Select("MAX(window_start) as window_start").Scan(&newest)
	stats.OldestRecord = oldest.WindowStart
	stats.NewestRecord = newest.WindowStart

	// 唯一交易对数
	gdb.Table("binance_24h_stats_history").Distinct("symbol").Count(&stats.UniqueSymbols)

	// 唯一市场数
	gdb.Table("binance_24h_stats_history").Distinct("market_type").Count(&stats.UniqueMarkets)

	// 平均每日记录数
	if !stats.OldestRecord.IsZero() && !stats.NewestRecord.IsZero() {
		days := stats.NewestRecord.Sub(stats.OldestRecord).Hours() / 24
		if days > 0 {
			stats.AvgRecordsPerDay = float64(stats.TotalRecords) / days
		}
	}

	return map[string]interface{}{
		"total_records":       stats.TotalRecords,
		"oldest_record":       stats.OldestRecord,
		"newest_record":       stats.NewestRecord,
		"unique_symbols":      stats.UniqueSymbols,
		"unique_markets":      stats.UniqueMarkets,
		"avg_records_per_day": stats.AvgRecordsPerDay,
		"data_period_days":    stats.NewestRecord.Sub(stats.OldestRecord).Hours() / 24,
	}, nil
}

// deduplicate24hStatsHistory 去重24小时统计历史数据
func deduplicate24hStatsHistory(stats []Binance24hStatsHistory) []Binance24hStatsHistory {
	if len(stats) <= 1 {
		return stats
	}

	// 使用map来跟踪唯一记录，基于symbol + market_type + window_start
	seen := make(map[string]Binance24hStatsHistory)
	var result []Binance24hStatsHistory

	for _, stat := range stats {
		key := fmt.Sprintf("%s:%s:%v", stat.Symbol, stat.MarketType, stat.WindowStart)

		// 如果这个键还没有出现过，则保留
		if _, exists := seen[key]; !exists {
			seen[key] = stat
			result = append(result, stat)
		}
	}

	return result
}
