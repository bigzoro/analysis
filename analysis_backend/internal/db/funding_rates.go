package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ===== 资金费率数据库操作 =====

// SaveFundingRates 批量保存资金费率
func SaveFundingRates(gdb *gorm.DB, rates []BinanceFundingRate) error {
	if len(rates) == 0 {
		return nil
	}

	tx := gdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, rate := range rates {
		now := time.Now()
		if rate.CreatedAt.IsZero() {
			rate.CreatedAt = now
		}
		rate.UpdatedAt = now

		err := tx.Exec(`
			INSERT INTO binance_funding_rates (
				symbol, funding_rate, funding_time, mark_price,
				index_price, estimated_settle_price, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				funding_rate = VALUES(funding_rate),
				mark_price = VALUES(mark_price),
				index_price = VALUES(index_price),
				estimated_settle_price = VALUES(estimated_settle_price),
				updated_at = VALUES(updated_at)
		`,
			rate.Symbol, rate.FundingRate, rate.FundingTime,
			rate.MarkPrice, rate.IndexPrice, rate.EstimatedSettlePrice,
			rate.CreatedAt, rate.UpdatedAt).Error

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("保存资金费率失败 %s: %w", rate.Symbol, err)
		}
	}

	return tx.Commit().Error
}