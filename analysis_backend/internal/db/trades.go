package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ===== 交易数据数据库操作 =====

// SaveTrades 批量保存交易数据
func SaveTrades(gdb *gorm.DB, trades []BinanceTrade) error {
	if len(trades) == 0 {
		return nil
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, trade := range trades {
			trade.CreatedAt = time.Now()
			if err := tx.Create(&trade).Error; err != nil {
				return fmt.Errorf("failed to save trade for %s: %w", trade.Symbol, err)
			}
		}
		return nil
	})
}