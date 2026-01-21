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

// ===== 期货信息同步器 =====

type FuturesSyncer struct {
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	// 已同步的期货合约列表（用于资金费率同步）
	futuresSymbols []string

	stats struct {
		mu                   sync.RWMutex
		totalSyncs           int64
		successfulSyncs      int64
		failedSyncs          int64
		lastSyncTime         time.Time
		totalContractUpdates int64
	}
}

func NewFuturesSyncer(db *gorm.DB, cfg *config.Config, config *DataSyncConfig) *FuturesSyncer {
	return &FuturesSyncer{
		db:     db,
		cfg:    cfg,
		config: config,
	}
}

func (s *FuturesSyncer) Name() string {
	return "futures"
}

func (s *FuturesSyncer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[FuturesSyncer] Started with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[FuturesSyncer] Stopped")
			return
		case <-ticker.C:
			if err := s.Sync(ctx); err != nil {
				log.Printf("[FuturesSyncer] Sync failed: %v", err)
			}
		}
	}
}

func (s *FuturesSyncer) Stop() {
	log.Printf("[FuturesSyncer] Stop signal received")
}

func (s *FuturesSyncer) Sync(ctx context.Context) error {
	s.stats.mu.Lock()
	s.stats.totalSyncs++
	s.stats.lastSyncTime = time.Now()
	s.stats.mu.Unlock()

	log.Printf("[FuturesSyncer] Starting futures info sync...")

	// 同步合约信息
	contractUpdates, err := s.syncContractInfo(ctx)
	if err != nil {
		log.Printf("[FuturesSyncer] Contract info sync failed: %v", err)
		s.stats.mu.Lock()
		s.stats.failedSyncs++
		s.stats.mu.Unlock()
		return err
	}

	// 同步资金费率
	fundingUpdates, err := s.syncFundingRates(ctx)
	if err != nil {
		log.Printf("[FuturesSyncer] Funding rates sync failed: %v", err)
		// 不返回错误，继续
	}

	totalUpdates := contractUpdates + fundingUpdates

	s.stats.mu.Lock()
	s.stats.successfulSyncs++
	s.stats.totalContractUpdates += int64(totalUpdates)
	s.stats.mu.Unlock()

	log.Printf("[FuturesSyncer] Futures info sync completed: %d updates", totalUpdates)
	return nil
}

func (s *FuturesSyncer) syncContractInfo(ctx context.Context) (int, error) {
	// 获取期货合约信息
	url := "https://fapi.binance.com/fapi/v1/exchangeInfo"

	type ContractInfo struct {
		Symbol             string      `json:"symbol"`
		Status             string      `json:"status"`
		ContractType       string      `json:"contractType"`
		BaseAsset          string      `json:"baseAsset"`
		QuoteAsset         string      `json:"quoteAsset"`
		MarginAsset        string      `json:"marginAsset"`
		PricePrecision     int         `json:"pricePrecision"`
		QuantityPrecision  int         `json:"quantityPrecision"`
		BaseAssetPrecision int         `json:"baseAssetPrecision"`
		QuotePrecision     int         `json:"quotePrecision"`
		UnderlyingType     string      `json:"underlyingType"`
		UnderlyingSubType  interface{} `json:"underlyingSubType"` // Can be string or []string
		SettlePlan         int         `json:"settlePlan"`
		TriggerProtect     string      `json:"triggerProtect"`
		Filters            interface{} `json:"filters"`
		OrderTypes         []string    `json:"orderTypes"`
		TimeInForce        []string    `json:"timeInForce"`
	}

	type ExchangeInfo struct {
		Symbols []ContractInfo `json:"symbols"`
	}

	var info ExchangeInfo
	if err := netutil.GetJSON(ctx, url, &info); err != nil {
		return 0, fmt.Errorf("failed to get exchange info: %w", err)
	}

	log.Printf("[FuturesSyncer] Fetched %d futures contracts from Binance", len(info.Symbols))

	var contracts []pdb.BinanceFuturesContract
	for _, contract := range info.Symbols {
		// 调试日志：检查contractType长度
		if len(contract.ContractType) > 20 {
			log.Printf("[FuturesSyncer] Long contract_type found: %s (length: %d)", contract.ContractType, len(contract.ContractType))
		}

		// 将数组转换为JSON字符串
		orderTypesJSON, _ := json.Marshal(contract.OrderTypes)
		timeInForceJSON, _ := json.Marshal(contract.TimeInForce)
		filtersJSON, _ := json.Marshal(contract.Filters)
		underlyingSubTypeJSON, _ := json.Marshal(contract.UnderlyingSubType)

		// 转换triggerProtect
		triggerProtect := parseFloat(contract.TriggerProtect)

		contractData := pdb.BinanceFuturesContract{
			Symbol:             contract.Symbol,
			Status:             contract.Status,
			ContractType:       contract.ContractType,
			BaseAsset:          contract.BaseAsset,
			QuoteAsset:         contract.QuoteAsset,
			MarginAsset:        contract.MarginAsset,
			PricePrecision:     contract.PricePrecision,
			QuantityPrecision:  contract.QuantityPrecision,
			BaseAssetPrecision: contract.BaseAssetPrecision,
			QuotePrecision:     contract.QuotePrecision,
			UnderlyingType:     contract.UnderlyingType,
			UnderlyingSubType:  string(underlyingSubTypeJSON),
			SettlePlan:         contract.SettlePlan,
			TriggerProtect:     triggerProtect,
			Filters:            string(filtersJSON),
			OrderTypes:         string(orderTypesJSON),
			TimeInForce:        string(timeInForceJSON),
		}
		contracts = append(contracts, contractData)
	}

	// 批量保存到数据库
	if err := pdb.SaveFuturesContracts(s.db, contracts); err != nil {
		return 0, fmt.Errorf("failed to save futures contracts: %w", err)
	}

	// 保存期货合约符号列表（用于资金费率同步）
	s.futuresSymbols = make([]string, 0, len(contracts))
	for _, contract := range contracts {
		if contract.Status == "TRADING" {
			s.futuresSymbols = append(s.futuresSymbols, contract.Symbol)
		}
	}

	log.Printf("[FuturesSyncer] Successfully saved %d futures contracts to database (%d active trading contracts)", len(contracts), len(s.futuresSymbols))
	return len(contracts), nil
}

func (s *FuturesSyncer) syncFundingRates(ctx context.Context) (int, error) {
	updates := 0

	// 如果还没有同步合约信息，先尝试获取
	if len(s.futuresSymbols) == 0 {
		log.Printf("[FuturesSyncer] No futures contracts synced yet, syncing contracts first...")
		if _, err := s.syncContractInfo(ctx); err != nil {
			return 0, fmt.Errorf("failed to sync contracts before funding rates: %w", err)
		}
	}

	// 只对有效的期货合约获取资金费率
	if len(s.futuresSymbols) == 0 {
		log.Printf("[FuturesSyncer] No active futures contracts found, skipping funding rates sync")
		return 0, nil
	}

	log.Printf("[FuturesSyncer] Syncing funding rates for %d futures contracts", len(s.futuresSymbols))

	for _, symbol := range s.futuresSymbols {
		var fundingRateData pdb.BinanceFundingRate
		var fundingRate float64
		var fundingTime int64
		var success bool

		// 优先级策略：最新4小时历史费率 > 实时预测费率 > 8小时已结算费率

		// 1. 首先尝试获取最新4小时的历史资金费率（优先级最高）
		if s.config.EnableFundingHistory {
			now := time.Now()
			hours := s.config.FundingHistoryHours
			if hours <= 0 {
				hours = 4 // 默认4小时
			}
			startTime := now.Add(-time.Duration(hours) * time.Hour).UnixMilli()
			endTime := now.UnixMilli()

			historyURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/fundingRate?symbol=%s&startTime=%d&endTime=%d&limit=1",
				symbol, startTime, endTime)

			type FundingRate struct {
				Symbol      string `json:"symbol"`
				FundingRate string `json:"fundingRate"`
				FundingTime int64  `json:"fundingTime"`
			}

			var rates []FundingRate
			if err := netutil.GetJSON(ctx, historyURL, &rates); err != nil {
				log.Printf("[FuturesSyncer] Failed to get recent historical funding rate for %s: %v", symbol, err)
			} else if len(rates) > 0 {
				// 使用最新的资金费率记录（数组中第一个是最新的）
				latestRate := rates[0]
				fundingRate = parseFloat(latestRate.FundingRate)
				fundingTime = latestRate.FundingTime
				fundingRateData.Symbol = latestRate.Symbol
				success = true

				log.Printf("[FuturesSyncer] ✅ Using recent historical funding rate for %s: %.8f (within last %d hours)",
					symbol, fundingRate, hours)
			}
		}

		// 2. 如果没有获取到历史数据，尝试实时预测费率
		if !success {
			premiumURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

			type PremiumIndex struct {
				Symbol          string `json:"symbol"`
				LastFundingRate string `json:"lastFundingRate"` // 实时预测费率
				Time            int64  `json:"time"`            // 当前时间戳
			}

			var premium PremiumIndex
			if err := netutil.GetJSON(ctx, premiumURL, &premium); err != nil {
				log.Printf("[FuturesSyncer] Failed to get premium index for %s: %v", symbol, err)
			} else {
				// 成功获取实时预测费率
				fundingRate = parseFloat(premium.LastFundingRate)
				fundingTime = premium.Time
				fundingRateData.Symbol = premium.Symbol
				success = true

				log.Printf("[FuturesSyncer] 📊 Using real-time funding rate for %s: %.8f", symbol, fundingRate)
			}
		}

		// 3. 如果都没有获取到，降级到获取最近的已结算费率（8小时周期）
		if !success {
			settledURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/fundingRate?symbol=%s&limit=1", symbol)

			type FundingRate struct {
				Symbol      string `json:"symbol"`
				FundingRate string `json:"fundingRate"`
				FundingTime int64  `json:"fundingTime"`
			}

			var rates []FundingRate
			if err := netutil.GetJSON(ctx, settledURL, &rates); err != nil {
				log.Printf("[FuturesSyncer] Failed to get settled funding rate for %s: %v", symbol, err)
				continue
			}

			if len(rates) > 0 {
				rate := rates[0]
				fundingRate = parseFloat(rate.FundingRate)
				fundingTime = rate.FundingTime
				fundingRateData.Symbol = rate.Symbol
				success = true

				log.Printf("[FuturesSyncer] 🗂️ Using settled funding rate for %s: %.8f", symbol, fundingRate)
			}
		}

		// 如果所有方法都失败，跳过这个交易对
		if !success {
			log.Printf("[FuturesSyncer] ❌ All funding rate sources failed for %s, skipping", symbol)
			continue
		}

		if success {
			fundingRateData.FundingRate = fundingRate
			fundingRateData.FundingTime = fundingTime

			// 使用批量保存函数处理重复数据（UPSERT）
			if err := pdb.SaveFundingRates(s.db, []pdb.BinanceFundingRate{fundingRateData}); err != nil {
				log.Printf("[FuturesSyncer] Failed to save funding rate for %s: %v", symbol, err)
				continue
			}

			log.Printf("[FuturesSyncer] Saved funding rate for %s: %.8f at %d", symbol, fundingRateData.FundingRate, fundingRateData.FundingTime)
			updates++
		}
	}

	return updates, nil
}

func (s *FuturesSyncer) GetStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_syncs":      s.stats.totalSyncs,
		"successful_syncs": s.stats.successfulSyncs,
		"failed_syncs":     s.stats.failedSyncs,
		"last_sync_time":   s.stats.lastSyncTime,
		"total_updates":    s.stats.totalContractUpdates,
	}
}
