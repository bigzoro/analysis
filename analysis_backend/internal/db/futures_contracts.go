package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ===== 期货合约信息数据库操作 =====

// SaveFuturesContracts 批量保存期货合约信息
func SaveFuturesContracts(gdb *gorm.DB, contracts []BinanceFuturesContract) error {
	if len(contracts) == 0 {
		return nil
	}

	tx := gdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, contract := range contracts {
		contract.UpdatedAt = time.Now()
		if contract.CreatedAt.IsZero() {
			contract.CreatedAt = time.Now()
		}

		err := tx.Exec(`
			INSERT INTO binance_futures_contracts (
				symbol, status, contract_type, base_asset, quote_asset,
				margin_asset, price_precision, quantity_precision,
				base_asset_precision, quote_precision, underlying_type,
				underlying_sub_type, settle_plan, trigger_protect,
				filters, order_types, time_in_force, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				contract_type = VALUES(contract_type),
				margin_asset = VALUES(margin_asset),
				price_precision = VALUES(price_precision),
				quantity_precision = VALUES(quantity_precision),
				base_asset_precision = VALUES(base_asset_precision),
				quote_precision = VALUES(quote_precision),
				underlying_type = VALUES(underlying_type),
				underlying_sub_type = VALUES(underlying_sub_type),
				settle_plan = VALUES(settle_plan),
				trigger_protect = VALUES(trigger_protect),
				filters = VALUES(filters),
				order_types = VALUES(order_types),
				time_in_force = VALUES(time_in_force),
				updated_at = VALUES(updated_at)
		`,
			contract.Symbol, contract.Status, contract.ContractType,
			contract.BaseAsset, contract.QuoteAsset, contract.MarginAsset,
			contract.PricePrecision, contract.QuantityPrecision,
			contract.BaseAssetPrecision, contract.QuotePrecision,
			contract.UnderlyingType, contract.UnderlyingSubType,
			contract.SettlePlan, contract.TriggerProtect,
			contract.Filters, contract.OrderTypes, contract.TimeInForce,
			contract.CreatedAt, contract.UpdatedAt).Error

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("保存期货合约失败 %s: %w", contract.Symbol, err)
		}
	}

	return tx.Commit().Error
}