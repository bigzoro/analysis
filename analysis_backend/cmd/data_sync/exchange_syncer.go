package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/netutil"

	"gorm.io/gorm"
)

// ===== 交易对信息同步器 =====

type ExchangeInfoSyncer struct {
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	stats struct {
		mu              sync.RWMutex
		totalSyncs      int64
		successfulSyncs int64
		failedSyncs     int64
		lastSyncTime    time.Time
		totalSymbols    int64
	}
}

func NewExchangeInfoSyncer(db *gorm.DB, cfg *config.Config, config *DataSyncConfig) *ExchangeInfoSyncer {
	return &ExchangeInfoSyncer{
		db:     db,
		cfg:    cfg,
		config: config,
	}
}

func (s *ExchangeInfoSyncer) Name() string {
	return "exchange_info"
}

func (s *ExchangeInfoSyncer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[ExchangeInfoSyncer] Starting exchange info sync with interval %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[ExchangeInfoSyncer] Stopping exchange info sync")
			return
		case <-ticker.C:
			if err := s.Sync(ctx); err != nil {
				log.Printf("[ExchangeInfoSyncer] Sync failed: %v", err)
			}
		}
	}
}

func (s *ExchangeInfoSyncer) Stop() {
	log.Printf("[ExchangeInfoSyncer] Exchange info syncer stopped")
}

func (s *ExchangeInfoSyncer) Sync(ctx context.Context) error {
	s.stats.mu.Lock()
	s.stats.totalSyncs++
	syncStartTime := time.Now()
	s.stats.lastSyncTime = syncStartTime
	s.stats.mu.Unlock()

	log.Printf("[ExchangeInfoSyncer] Starting exchange info sync with soft delete support...")

	// 获取现货交易对信息
	spotSymbols, err := s.fetchExchangeInfo(ctx, "spot")
	if err != nil {
		s.stats.mu.Lock()
		s.stats.failedSyncs++
		s.stats.mu.Unlock()
		return fmt.Errorf("failed to fetch spot exchange info: %w", err)
	}

	// 获取期货交易对信息
	futuresSymbols, err := s.fetchExchangeInfo(ctx, "futures")
	if err != nil {
		s.stats.mu.Lock()
		s.stats.failedSyncs++
		s.stats.mu.Unlock()
		return fmt.Errorf("failed to fetch futures exchange info: %w", err)
	}

	// 合并交易对信息
	allSymbols := append(spotSymbols, futuresSymbols...)
	log.Printf("[ExchangeInfoSyncer] Fetched %d total symbols from API (%d spot, %d futures)",
		len(allSymbols), len(spotSymbols), len(futuresSymbols))

	// 执行软删除同步
	if err := s.syncWithSoftDelete(ctx, allSymbols); err != nil {
		s.stats.mu.Lock()
		s.stats.failedSyncs++
		s.stats.mu.Unlock()
		return fmt.Errorf("failed to sync with soft delete: %w", err)
	}

	syncDuration := time.Since(syncStartTime)
	s.stats.mu.Lock()
	s.stats.successfulSyncs++
	s.stats.totalSymbols = int64(len(allSymbols))
	s.stats.mu.Unlock()

	log.Printf("[ExchangeInfoSyncer] Exchange info sync completed in %v: %d symbols processed",
		syncDuration.Round(time.Millisecond), len(allSymbols))
	return nil
}

func (s *ExchangeInfoSyncer) fetchExchangeInfo(ctx context.Context, kind string) ([]pdb.BinanceExchangeInfo, error) {
	var url string
	switch kind {
	case "spot":
		url = "https://api.binance.com/api/v3/exchangeInfo"
	case "futures":
		url = "https://fapi.binance.com/fapi/v1/exchangeInfo"
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}

	var response struct {
		Symbols []struct {
			Symbol                     string      `json:"symbol"`
			Status                     string      `json:"status"`
			BaseAsset                  string      `json:"baseAsset"`
			QuoteAsset                 string      `json:"quoteAsset"`
			BaseAssetPrecision         int         `json:"baseAssetPrecision"`
			QuoteAssetPrecision        int         `json:"quoteAssetPrecision"`
			BaseCommissionPrecision    int         `json:"baseCommissionPrecision"`
			QuoteCommissionPrecision   int         `json:"quoteCommissionPrecision"`
			OrderTypes                 []string    `json:"orderTypes"`
			IcebergAllowed             bool        `json:"icebergAllowed"`
			OcoAllowed                 bool        `json:"ocoAllowed"`
			QuoteOrderQtyMarketAllowed bool        `json:"quoteOrderQtyMarketAllowed"`
			AllowTrailingStop          bool        `json:"allowTrailingStop"`
			CancelReplaceAllowed       bool        `json:"cancelReplaceAllowed"`
			IsSpotTradingAllowed       bool        `json:"isSpotTradingAllowed"`
			IsMarginTradingAllowed     bool        `json:"isMarginTradingAllowed"`
			Filters                    interface{} `json:"filters"`
			Permissions                []string    `json:"permissions"`
		} `json:"symbols"`
	}

	if err := netutil.GetJSON(ctx, url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch exchange info from %s: %w", url, err)
	}

	log.Printf("[ExchangeInfoSyncer] Fetched %d %s symbols from %s", len(response.Symbols), kind, url)

	var symbols []pdb.BinanceExchangeInfo
	for _, symbol := range response.Symbols {
		// 将数组转换为JSON字符串
		orderTypesJSON, _ := json.Marshal(symbol.OrderTypes)
		permissionsJSON, _ := json.Marshal(symbol.Permissions)
		filtersJSON, _ := json.Marshal(symbol.Filters)

		info := pdb.BinanceExchangeInfo{
			Symbol:                     symbol.Symbol,
			Status:                     symbol.Status,
			BaseAsset:                  symbol.BaseAsset,
			QuoteAsset:                 symbol.QuoteAsset,
			MarketType:                 kind, // 设置市场类型：spot 或 futures
			BaseAssetPrecision:         symbol.BaseAssetPrecision,
			QuoteAssetPrecision:        symbol.QuoteAssetPrecision,
			BaseCommissionPrecision:    symbol.BaseCommissionPrecision,
			QuoteCommissionPrecision:   symbol.QuoteCommissionPrecision,
			OrderTypes:                 string(orderTypesJSON),
			IcebergAllowed:             symbol.IcebergAllowed,
			OcoAllowed:                 symbol.OcoAllowed,
			QuoteOrderQtyMarketAllowed: symbol.QuoteOrderQtyMarketAllowed,
			AllowTrailingStop:          symbol.AllowTrailingStop,
			CancelReplaceAllowed:       symbol.CancelReplaceAllowed,
			IsSpotTradingAllowed:       symbol.IsSpotTradingAllowed,
			IsMarginTradingAllowed:     symbol.IsMarginTradingAllowed,
			Filters:                    string(filtersJSON),
			Permissions:                string(permissionsJSON),
		}
		symbols = append(symbols, info)
	}

	return symbols, nil
}

// syncWithSoftDelete 使用软删除策略同步交易对信息
func (s *ExchangeInfoSyncer) syncWithSoftDelete(ctx context.Context, currentSymbols []pdb.BinanceExchangeInfo) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// 收集当前活跃的交易对信息（symbol + market_type）
	activeSymbolKeys := make(map[string]bool)
	activeSymbolsMap := make(map[string]pdb.BinanceExchangeInfo)

	for _, symbol := range currentSymbols {
		key := symbol.Symbol + "_" + symbol.MarketType
		activeSymbolKeys[key] = true
		activeSymbolsMap[key] = symbol
	}

	log.Printf("[ExchangeInfoSyncer] Processing %d active symbols from API", len(currentSymbols))

	// 1. 更新或插入当前活跃的交易对
	for _, symbol := range currentSymbols {
		// 设置状态管理字段
		symbol.IsActive = true
		symbol.LastSeenActive = &now
		symbol.DeactivatedAt = nil // 清除下架时间

		// 使用Upsert操作
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
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			symbol.Symbol, symbol.Status, symbol.BaseAsset, symbol.QuoteAsset, symbol.MarketType,
			symbol.BaseAssetPrecision, symbol.QuoteAssetPrecision,
			symbol.BaseCommissionPrecision, symbol.QuoteCommissionPrecision,
			symbol.OrderTypes, symbol.IcebergAllowed, symbol.OcoAllowed,
			symbol.QuoteOrderQtyMarketAllowed, symbol.AllowTrailingStop,
			symbol.CancelReplaceAllowed, symbol.IsSpotTradingAllowed,
			symbol.IsMarginTradingAllowed, symbol.Filters, symbol.Permissions,
			symbol.IsActive, symbol.LastSeenActive, symbol.DeactivatedAt,
			now, now).Error

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert symbol %s: %w", symbol.Symbol, err)
		}
	}

	// 2. 将不再出现在API中的交易对标记为非活跃（软删除）
	inactiveCount := 0
	for _, marketType := range []string{"spot", "futures"} {
		var dbSymbols []struct {
			Symbol     string
			MarketType string
		}

		// 查询数据库中当前活跃的交易对
		err := tx.Table("binance_exchange_info").
			Select("symbol, market_type").
			Where("market_type = ? AND is_active = ?", marketType, true).
			Find(&dbSymbols).Error

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to query active symbols for market %s: %w", marketType, err)
		}

		// 检查哪些交易对不再活跃
		for _, dbSymbol := range dbSymbols {
			key := dbSymbol.Symbol + "_" + dbSymbol.MarketType
			if !activeSymbolKeys[key] {
				// 这个交易对不再出现在API中，标记为非活跃
				err := tx.Table("binance_exchange_info").
					Where("symbol = ? AND market_type = ?", dbSymbol.Symbol, dbSymbol.MarketType).
					Updates(map[string]interface{}{
						"is_active":      false,
						"deactivated_at": now,
						"updated_at":     now,
					}).Error

				if err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to deactivate symbol %s: %w", dbSymbol.Symbol, err)
				}

				inactiveCount++
				log.Printf("[ExchangeInfoSyncer] 🗑️ Deactivated symbol: %s %s", dbSymbol.Symbol, dbSymbol.MarketType)
			}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[ExchangeInfoSyncer] ✅ Soft delete sync completed: %d activated, %d deactivated",
		len(currentSymbols), inactiveCount)
	return nil
}

// GetExchangeInfoStats 获取交易对状态统计
func (s *ExchangeInfoSyncer) GetExchangeInfoStats() map[string]interface{} {
	// 查询活跃和非活跃交易对的数量统计
	var stats struct {
		ActiveCount     int64 `json:"active_count"`
		InactiveCount   int64 `json:"inactive_count"`
		SpotActive      int64 `json:"spot_active"`
		SpotInactive    int64 `json:"spot_inactive"`
		FuturesActive   int64 `json:"futures_active"`
		FuturesInactive int64 `json:"futures_inactive"`
	}

	// 总体统计
	s.db.Model(&pdb.BinanceExchangeInfo{}).Where("is_active = ?", true).Count(&stats.ActiveCount)
	s.db.Model(&pdb.BinanceExchangeInfo{}).Where("is_active = ?", false).Count(&stats.InactiveCount)

	// 现货统计
	s.db.Model(&pdb.BinanceExchangeInfo{}).Where("market_type = ? AND is_active = ?", "spot", true).Count(&stats.SpotActive)
	s.db.Model(&pdb.BinanceExchangeInfo{}).Where("market_type = ? AND is_active = ?", "spot", false).Count(&stats.SpotInactive)

	// 期货统计
	s.db.Model(&pdb.BinanceExchangeInfo{}).Where("market_type = ? AND is_active = ?", "futures", true).Count(&stats.FuturesActive)
	s.db.Model(&pdb.BinanceExchangeInfo{}).Where("market_type = ? AND is_active = ?", "futures", false).Count(&stats.FuturesInactive)

	return map[string]interface{}{
		"active_symbols":   stats.ActiveCount,
		"inactive_symbols": stats.InactiveCount,
		"spot_active":      stats.SpotActive,
		"spot_inactive":    stats.SpotInactive,
		"futures_active":   stats.FuturesActive,
		"futures_inactive": stats.FuturesInactive,
	}
}

func (s *ExchangeInfoSyncer) GetStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	// 获取基础统计
	baseStats := map[string]interface{}{
		"total_syncs":      s.stats.totalSyncs,
		"successful_syncs": s.stats.successfulSyncs,
		"failed_syncs":     s.stats.failedSyncs,
		"last_sync_time":   s.stats.lastSyncTime,
		"total_symbols":    s.stats.totalSymbols,
	}

	// 添加交易对状态统计
	exchangeStats := s.GetExchangeInfoStats()
	for k, v := range exchangeStats {
		baseStats[k] = v
	}

	return baseStats
}
