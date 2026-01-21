package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ===== 订单簿深度数据库操作 =====

// SaveOrderBookDepth 保存订单簿深度
func SaveOrderBookDepth(gdb *gorm.DB, depths []BinanceOrderBookDepth) error {
	if len(depths) == 0 {
		return nil
	}

	// 去重处理：移除重复的深度数据
	uniqueDepths := deduplicateOrderBookDepths(depths)
	if len(uniqueDepths) != len(depths) {
		log.Printf("[SaveOrderBookDepth] Removed %d duplicate depths, saving %d unique records",
			len(depths)-len(uniqueDepths), len(uniqueDepths))
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 在事务开始前，检查数据库中是否已存在记录
		// 由于没有唯一索引，我们需要手动检查避免重复
		if len(uniqueDepths) > 0 {
			for i := 0; i < len(uniqueDepths); i++ {
				depth := uniqueDepths[i]

				// 检查数据库中是否已存在相同symbol和market_type的记录
				// 如果存在，跳过插入（保留最新的记录）
				var existingCount int64
				checkErr := tx.Table("binance_order_book_depth").Where(
					"symbol = ? AND market_type = ? AND last_update_id = ?",
					depth.Symbol, depth.MarketType, depth.LastUpdateID,
				).Count(&existingCount).Error

				if checkErr != nil {
					log.Printf("[SaveOrderBookDepth] Failed to check existing depth for %s %s: %v",
						depth.Symbol, depth.MarketType, checkErr)
					continue
				}

				if existingCount > 0 {
					// 记录已存在，跳过
					log.Printf("[SaveOrderBookDepth] Skipping existing depth: %s %s (update_id: %d)",
						depth.Symbol, depth.MarketType, depth.LastUpdateID)
					continue
				}

				// 设置时间戳
				if depth.CreatedAt.IsZero() {
					depth.CreatedAt = time.Now()
				}

				if err := tx.Create(&depth).Error; err != nil {
					log.Printf("[SaveOrderBookDepth] Failed to save depth for %s: %v", depth.Symbol, err)
					continue
				}
			}
		}

		return nil
	})
}

// deduplicateOrderBookDepths 去重订单簿深度数据
func deduplicateOrderBookDepths(depths []BinanceOrderBookDepth) []BinanceOrderBookDepth {
	if len(depths) <= 1 {
		return depths
	}

	// 使用map来跟踪唯一记录，基于symbol + market_type + last_update_id
	seen := make(map[string]BinanceOrderBookDepth)
	var result []BinanceOrderBookDepth

	for _, depth := range depths {
		key := fmt.Sprintf("%s:%s:%d", depth.Symbol, depth.MarketType, depth.LastUpdateID)

		// 如果这个键还没有出现过，或者记录更新，则保留
		if existing, exists := seen[key]; !exists {
			seen[key] = depth
			result = append(result, depth)
		} else {
			// 如果已存在相同记录，保留created_at更晚的（更新的记录）
			if depth.CreatedAt.After(existing.CreatedAt) {
				seen[key] = depth
				// 找到result中的对应记录并更新
				for j, r := range result {
					if r.Symbol == depth.Symbol && r.MarketType == depth.MarketType && r.LastUpdateID == depth.LastUpdateID {
						result[j] = depth
						break
					}
				}
			}
		}
	}

	return result
}