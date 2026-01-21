package server

import (
	"sync"
	"time"
)

// DataCache 数据缓存系统
type DataCache struct {
	mu          sync.RWMutex
	symbolData  map[string]*SymbolUnifiedData
	unifiedData *UnifiedMarketData
	lastUpdate  time.Time
	ttl         time.Duration
}

// NewDataCache 创建数据缓存
func NewDataCache() *DataCache {
	return &DataCache{
		symbolData: make(map[string]*SymbolUnifiedData),
		ttl:        15 * time.Minute, // 15分钟TTL
	}
}

// StoreUnifiedData 存储统一数据
func (dc *DataCache) StoreUnifiedData(data *UnifiedMarketData) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.unifiedData = data
	dc.lastUpdate = time.Now()

	// 存储单个币种数据
	for symbol, symbolData := range data.SymbolData {
		dc.symbolData[symbol] = symbolData
	}
}

// GetSymbolData 获取币种数据
func (dc *DataCache) GetSymbolData(symbol string) (*SymbolUnifiedData, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	data, exists := dc.symbolData[symbol]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Since(dc.lastUpdate) > dc.ttl {
		return nil, false
	}

	return data, true
}

// GetUnifiedData 获取完整统一数据
func (dc *DataCache) GetUnifiedData() (*UnifiedMarketData, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	if dc.unifiedData == nil {
		return nil, false
	}

	// 检查是否过期
	if time.Since(dc.lastUpdate) > dc.ttl {
		return nil, false
	}

	return dc.unifiedData, true
}

// IsExpired 检查是否过期
func (dc *DataCache) IsExpired() bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return time.Since(dc.lastUpdate) > dc.ttl
}

// ClearExpired 清理过期数据
func (dc *DataCache) ClearExpired() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if time.Since(dc.lastUpdate) > dc.ttl {
		dc.symbolData = make(map[string]*SymbolUnifiedData)
		dc.unifiedData = nil
	}
}

// GetStats 获取缓存统计信息
func (dc *DataCache) GetStats() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return map[string]interface{}{
		"symbol_count":      len(dc.symbolData),
		"last_update":       dc.lastUpdate,
		"is_expired":        dc.IsExpired(),
		"time_since_update": time.Since(dc.lastUpdate).String(),
	}
}
