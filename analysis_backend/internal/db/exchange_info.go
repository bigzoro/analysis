package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ===== 交易对信息数据库操作 =====

// SaveExchangeInfo 批量保存交易对信息
func SaveExchangeInfo(gdb *gorm.DB, exchangeInfo []BinanceExchangeInfo) error {
	if len(exchangeInfo) == 0 {
		return nil
	}

	// 使用事务批量插入或更新
	tx := gdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, info := range exchangeInfo {
		info.UpdatedAt = time.Now()
		if info.CreatedAt.IsZero() {
			info.CreatedAt = time.Now()
		}

		// 使用ON DUPLICATE KEY UPDATE实现插入或更新
		err := tx.Exec(`
			INSERT INTO binance_exchange_info (
				symbol, status, base_asset, quote_asset, market_type,
				base_asset_precision, quote_asset_precision,
				base_commission_precision, quote_commission_precision,
				order_types, iceberg_allowed, oco_allowed,
				quote_order_qty_market_allowed, allow_trailing_stop,
				cancel_replace_allowed, is_spot_trading_allowed,
				is_margin_trading_allowed, filters, permissions,
				is_active, last_seen_active, deactivated_at,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				base_asset_precision = VALUES(base_asset_precision),
				quote_asset_precision = VALUES(quote_asset_precision),
				base_commission_precision = VALUES(base_commission_precision),
				quote_commission_precision = VALUES(quote_commission_precision),
				order_types = VALUES(order_types),
				iceberg_allowed = VALUES(iceberg_allowed),
				oco_allowed = VALUES(oco_allowed),
				quote_order_qty_market_allowed = VALUES(quote_order_qty_market_allowed),
				allow_trailing_stop = VALUES(allow_trailing_stop),
				cancel_replace_allowed = VALUES(cancel_replace_allowed),
				is_spot_trading_allowed = VALUES(is_spot_trading_allowed),
				is_margin_trading_allowed = VALUES(is_margin_trading_allowed),
				filters = VALUES(filters),
				permissions = VALUES(permissions),
				is_active = VALUES(is_active),
				last_seen_active = VALUES(last_seen_active),
				deactivated_at = VALUES(deactivated_at),
				updated_at = VALUES(updated_at)
		`,
			info.Symbol, info.Status, info.BaseAsset, info.QuoteAsset, info.MarketType,
			info.BaseAssetPrecision, info.QuoteAssetPrecision,
			info.BaseCommissionPrecision, info.QuoteCommissionPrecision,
			info.OrderTypes, info.IcebergAllowed, info.OcoAllowed,
			info.QuoteOrderQtyMarketAllowed, info.AllowTrailingStop,
			info.CancelReplaceAllowed, info.IsSpotTradingAllowed,
			info.IsMarginTradingAllowed, info.Filters, info.Permissions,
			info.IsActive, info.LastSeenActive, info.DeactivatedAt,
			info.CreatedAt, info.UpdatedAt).Error

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("保存交易对信息失败 %s: %w", info.Symbol, err)
		}
	}

	return tx.Commit().Error
}

// GetUSDTTradingPairs 获取所有活跃的USDT交易对
func GetUSDTTradingPairs(gdb *gorm.DB) ([]string, error) {
	var symbols []string
	err := gdb.Model(&BinanceExchangeInfo{}).
		Where("quote_asset = ? AND status = ? AND is_active = ?",
			"USDT", "TRADING", true).
		Order("symbol").
		Pluck("symbol", &symbols).Error
	return symbols, err
}

// GetUSDTTradingPairsByMarket 按市场获取活跃的USDT交易对
func GetUSDTTradingPairsByMarket(gdb *gorm.DB, marketType string) ([]string, error) {
	var symbols []string
	err := gdb.Model(&BinanceExchangeInfo{}).
		Where("quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?",
			"USDT", "TRADING", marketType, true).
		Order("symbol").
		Pluck("symbol", &symbols).Error
	return symbols, err
}

// GetAllTradingPairsByMarket 按市场获取所有交易对（包括非活跃的）
func GetAllTradingPairsByMarket(gdb *gorm.DB, marketType string) ([]BinanceExchangeInfo, error) {
	var symbols []BinanceExchangeInfo
	err := gdb.Model(&BinanceExchangeInfo{}).
		Where("market_type = ?", marketType).
		Order("is_active DESC, symbol"). // 活跃的排在前面
		Find(&symbols).Error
	return symbols, err
}

// GetInactiveTradingPairsByMarket 获取指定市场的非活跃交易对
func GetInactiveTradingPairsByMarket(gdb *gorm.DB, marketType string) ([]BinanceExchangeInfo, error) {
	var symbols []BinanceExchangeInfo
	err := gdb.Model(&BinanceExchangeInfo{}).
		Where("market_type = ? AND is_active = ?", marketType, false).
		Order("deactivated_at DESC, symbol").
		Find(&symbols).Error
	return symbols, err
}

// GetRecentlyDeactivatedSymbols 获取最近下架的交易对
func GetRecentlyDeactivatedSymbols(gdb *gorm.DB, marketType string, since time.Time) ([]BinanceExchangeInfo, error) {
	var symbols []BinanceExchangeInfo
	err := gdb.Model(&BinanceExchangeInfo{}).
		Where("market_type = ? AND is_active = ? AND deactivated_at >= ?",
			marketType, false, since).
		Order("deactivated_at DESC").
		Find(&symbols).Error
	return symbols, err
}

// GetExchangeInfo 获取交易对信息
func GetExchangeInfo(gdb *gorm.DB, symbol string) (*BinanceExchangeInfo, error) {
	var info BinanceExchangeInfo
	err := gdb.Where("symbol = ?", symbol).First(&info).Error
	return &info, err
}

// GetAllExchangeInfo 获取所有交易对信息
func GetAllExchangeInfo(gdb *gorm.DB) ([]BinanceExchangeInfo, error) {
	var infos []BinanceExchangeInfo
	err := gdb.Order("symbol").Find(&infos).Error
	return infos, err
}

// GetExchangeInfoByQuoteAsset 根据计价资产获取交易对信息
func GetExchangeInfoByQuoteAsset(gdb *gorm.DB, quoteAsset string) ([]BinanceExchangeInfo, error) {
	var infos []BinanceExchangeInfo
	err := gdb.Where("quote_asset = ? AND status = ?", quoteAsset, "TRADING").
		Order("symbol").Find(&infos).Error
	return infos, err
}

// GetActiveExchangeInfoByQuoteAsset 根据计价资产获取活跃交易对信息
func GetActiveExchangeInfoByQuoteAsset(gdb *gorm.DB, quoteAsset string) ([]BinanceExchangeInfo, error) {
	var infos []BinanceExchangeInfo
	err := gdb.Where("quote_asset = ? AND status = ? AND is_active = ?",
		quoteAsset, "TRADING", true).
		Order("symbol").Find(&infos).Error
	return infos, err
}

// GetExchangeInfoCount 获取交易对总数
func GetExchangeInfoCount(gdb *gorm.DB) (int64, error) {
	var count int64
	err := gdb.Model(&BinanceExchangeInfo{}).Count(&count).Error
	return count, err
}

// GetActiveExchangeInfoCount 获取活跃交易对总数
func GetActiveExchangeInfoCount(gdb *gorm.DB) (int64, error) {
	var count int64
	err := gdb.Model(&BinanceExchangeInfo{}).Where("is_active = ?", true).Count(&count).Error
	return count, err
}

// GetExchangeInfoStats 获取交易对状态统计
func GetExchangeInfoStats(gdb *gorm.DB) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 总交易对数
	var totalCount int64
	err := gdb.Model(&BinanceExchangeInfo{}).Count(&totalCount).Error
	if err != nil {
		return nil, err
	}
	stats["total"] = totalCount

	// 活跃交易对数
	var activeCount int64
	err = gdb.Model(&BinanceExchangeInfo{}).Where("is_active = ?", true).Count(&activeCount).Error
	if err != nil {
		return nil, err
	}
	stats["active"] = activeCount

	// 非活跃交易对数
	var inactiveCount int64
	err = gdb.Model(&BinanceExchangeInfo{}).Where("is_active = ?", false).Count(&inactiveCount).Error
	if err != nil {
		return nil, err
	}
	stats["inactive"] = inactiveCount

	// 现货活跃交易对
	var spotActiveCount int64
	err = gdb.Model(&BinanceExchangeInfo{}).Where("market_type = ? AND is_active = ?", "spot", true).Count(&spotActiveCount).Error
	if err != nil {
		return nil, err
	}
	stats["spot_active"] = spotActiveCount

	// 期货活跃交易对
	var futuresActiveCount int64
	err = gdb.Model(&BinanceExchangeInfo{}).Where("market_type = ? AND is_active = ?", "futures", true).Count(&futuresActiveCount).Error
	if err != nil {
		return nil, err
	}
	stats["futures_active"] = futuresActiveCount

	return stats, nil
}

// GetLastExchangeInfoUpdate 获取最后更新时间
func GetLastExchangeInfoUpdate(gdb *gorm.DB) (*time.Time, error) {
	var latest time.Time
	err := gdb.Model(&BinanceExchangeInfo{}).Select("MAX(updated_at)").Scan(&latest).Error
	if err != nil {
		return nil, err
	}
	return &latest, nil
}