package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ===== 24小时统计数据数据库操作 =====

// Save24hStats 批量保存24小时统计数据
func Save24hStats(gdb *gorm.DB, stats []Binance24hStats) error {
	if len(stats) == 0 {
		return nil
	}

	// 去重处理：移除重复的统计数据
	uniqueStats := deduplicate24hStats(stats)
	if len(uniqueStats) != len(stats) {
		log.Printf("[Save24hStats] Removed %d duplicate stats, saving %d unique records",
			len(stats)-len(uniqueStats), len(uniqueStats))
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 在事务开始前，检查数据库中是否已存在记录
		// 对于24小时统计，我们基于symbol和market_type来更新
		if len(uniqueStats) > 0 {
			for i := 0; i < len(uniqueStats); i++ {
				stat := uniqueStats[i]

				// 检查数据库中是否已存在相同symbol和market_type的记录
				// 对于24小时统计，我们保留最新的记录
				var existingCount int64
				checkErr := tx.Table("binance_24h_stats").Where(
					"symbol = ? AND market_type = ?",
					stat.Symbol, stat.MarketType,
				).Count(&existingCount).Error

				if checkErr != nil {
					log.Printf("[Save24hStats] Failed to check existing stats for %s %s: %v",
						stat.Symbol, stat.MarketType, checkErr)
					continue
				}

				// 设置时间戳
				now := time.Now()
				if stat.CreatedAt.IsZero() {
					stat.CreatedAt = now
				}

				if existingCount > 0 {
					// 记录已存在，更新
					updateErr := tx.Table("binance_24h_stats").Where(
						"symbol = ? AND market_type = ?",
						stat.Symbol, stat.MarketType,
					).Updates(map[string]interface{}{
						"price_change":         stat.PriceChange,
						"price_change_percent": stat.PriceChangePercent,
						"weighted_avg_price":   stat.WeightedAvgPrice,
						"prev_close_price":     stat.PrevClosePrice,
						"last_price":           stat.LastPrice,
						"bid_price":            stat.BidPrice,
						"ask_price":            stat.AskPrice,
						"open_price":           stat.OpenPrice,
						"high_price":           stat.HighPrice,
						"low_price":            stat.LowPrice,
						"volume":               stat.Volume,
						"quote_volume":         stat.QuoteVolume,
						"open_time":            stat.OpenTime,
						"close_time":           stat.CloseTime,
						"count":                stat.Count,
						"last_qty":             stat.LastQty,
						"bid_qty":              stat.BidQty,
						"ask_qty":              stat.AskQty,
						"first_id":             stat.FirstId,
						"last_id":              stat.LastId,
						"updated_at":           now, // 直接设置更新时间
					}).Error

					if updateErr != nil {
						log.Printf("[Save24hStats] Failed to update stats for %s %s: %v",
							stat.Symbol, stat.MarketType, updateErr)
					} else {
						log.Printf("[Save24hStats] Updated existing 24h stats: %s %s",
							stat.Symbol, stat.MarketType)
					}
				} else {
					// 插入新记录
					if err := tx.Create(&stat).Error; err != nil {
						log.Printf("[Save24hStats] Failed to insert stats for %s: %v", stat.Symbol, err)
						continue
					}
				}
			}
		}

		return nil
	})
}

// deduplicate24hStats 去重24小时统计数据
func deduplicate24hStats(stats []Binance24hStats) []Binance24hStats {
	if len(stats) <= 1 {
		return stats
	}

	// 使用map来跟踪唯一记录，基于symbol + market_type
	seen := make(map[string]Binance24hStats)
	var result []Binance24hStats

	for _, stat := range stats {
		key := fmt.Sprintf("%s:%s", stat.Symbol, stat.MarketType)

		// 如果这个键还没有出现过，或者记录更新，则保留
		if existing, exists := seen[key]; !exists {
			seen[key] = stat
			result = append(result, stat)
		} else {
			// 如果已存在相同记录，保留created_at更晚的（更新的记录）
			if stat.CreatedAt.After(existing.CreatedAt) {
				seen[key] = stat
				// 找到result中的对应记录并更新
				for j, r := range result {
					if r.Symbol == stat.Symbol && r.MarketType == stat.MarketType {
						result[j] = stat
						break
					}
				}
			}
		}
	}

	return result
}