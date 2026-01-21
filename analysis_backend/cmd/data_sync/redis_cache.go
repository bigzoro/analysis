package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// RedisInvalidSymbolCache Redis无效符号缓存管理器
type RedisInvalidSymbolCache struct {
	client  *redis.Client
	prefix  string
	ttl     time.Duration
	enabled bool
}

// NewRedisInvalidSymbolCache 创建Redis无效符号缓存管理器
func NewRedisInvalidSymbolCache(client *redis.Client, prefix string, ttl time.Duration) *RedisInvalidSymbolCache {
	enabled := client != nil
	if enabled {
		log.Printf("[RedisCache] Initialized Redis invalid symbol cache with prefix: %s, TTL: %v", prefix, ttl)
	} else {
		log.Printf("[RedisCache] Redis client not available, using fallback mode")
	}

	return &RedisInvalidSymbolCache{
		client:  client,
		prefix:  prefix,
		ttl:     ttl,
		enabled: enabled,
	}
}

// MarkInvalid 标记符号为无效
func (r *RedisInvalidSymbolCache) MarkInvalid(symbol, kind string) error {
	if !r.enabled {
		return nil // 不启用时不报错，静默跳过
	}

	key := r.buildKey(symbol, kind)
	ctx := context.Background()

	err := r.client.Set(ctx, key, "invalid", r.ttl).Err()
	if err != nil {
		log.Printf("[RedisCache] Failed to mark invalid symbol %s %s: %v", symbol, kind, err)
		return err
	}

	log.Printf("[RedisCache] 🛑 Marked %s %s as invalid symbol in Redis", symbol, kind)
	return nil
}

// IsInvalid 检查符号是否无效
func (r *RedisInvalidSymbolCache) IsInvalid(symbol, kind string) bool {
	if !r.enabled {
		return false // 不启用时认为都有效
	}

	key := r.buildKey(symbol, kind)
	ctx := context.Background()

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("[RedisCache] Failed to check invalid symbol %s %s: %v", symbol, kind, err)
		return false // 出错时认为有效，避免误判
	}

	return exists > 0
}

// GetAllInvalidSymbols 获取所有无效符号（用于调试）
func (r *RedisInvalidSymbolCache) GetAllInvalidSymbols() (map[string]bool, error) {
	if !r.enabled {
		return nil, fmt.Errorf("redis cache not enabled")
	}

	pattern := r.prefix + "*"
	ctx := context.Background()

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	result := make(map[string]bool)
	for _, key := range keys {
		// 从key中提取symbol和kind
		if len(key) > len(r.prefix) {
			symbolKind := key[len(r.prefix):]
			result[symbolKind] = true
		}
	}

	return result, nil
}

// ClearExpired 清理过期的无效符号（可选，Redis会自动过期）
func (r *RedisInvalidSymbolCache) ClearExpired() error {
	if !r.enabled {
		return nil
	}

	log.Printf("[RedisCache] Redis TTL will automatically clear expired entries")
	return nil
}

// CleanupInvalidSymbols 清理不再存在于数据库中的无效符号（验证后再清理）
func (r *RedisInvalidSymbolCache) CleanupInvalidSymbols(db *gorm.DB) error {
	if !r.enabled {
		return nil
	}

	log.Printf("[RedisCache] 🧹 Starting smart cleanup of invalid symbols cache...")

	// 获取Redis中所有的无效符号
	invalidSymbols, err := r.GetAllInvalidSymbols()
	if err != nil {
		return fmt.Errorf("failed to get all invalid symbols: %w", err)
	}

	if len(invalidSymbols) == 0 {
		log.Printf("[RedisCache] ✅ No invalid symbols in cache to cleanup")
		return nil
	}

	log.Printf("[RedisCache] 📊 Found %d invalid symbols in cache, validating with database and API...", len(invalidSymbols))

	// 获取数据库中当前有效的交易对（按市场类型分组）
	validSymbols, err := getValidSymbolsByMarket(db)
	if err != nil {
		log.Printf("[RedisCache] ⚠️ Failed to get valid symbols by market: %v", err)
		validSymbols = map[string]map[string]bool{
			"spot":    make(map[string]bool),
			"futures": make(map[string]bool),
		}
	}

	// 创建API客户端用于验证（无统计回调的轻量级版本）
	apiClient := NewBinanceAPIClient()
	ctx := context.Background()

	var symbolsToRemove []string
	var symbolsToKeep []string

	// 对每个缓存的无效符号进行验证
	for symbolKind := range invalidSymbols {
		// 解析symbol和kind (格式: symbol_kind)
		parts := strings.Split(symbolKind, "_")
		if len(parts) != 2 {
			log.Printf("[RedisCache] ⚠️ Invalid symbol format in cache: %s", symbolKind)
			symbolsToRemove = append(symbolsToRemove, symbolKind) // 清理格式错误的key
			continue
		}

		symbol := parts[0]
		kind := parts[1]

		// 检查数据库状态（按市场类型验证）
		marketValidSymbols, exists := validSymbols[kind]
		if !exists {
			log.Printf("[RedisCache] ⚠️ Unknown market type in cache: %s", kind)
			symbolsToRemove = append(symbolsToRemove, symbolKind)
			continue
		}

		if !marketValidSymbols[symbol] {
			// 符号在该市场类型中不活跃，清理缓存
			symbolsToRemove = append(symbolsToRemove, symbolKind)
			log.Printf("[RedisCache] 🗑️ 计划清理：%s %s (在%s市场不活跃)", symbol, kind, kind)
		} else {
			// 符号在数据库中是活跃的，需要API验证确认
			log.Printf("[RedisCache] 🔍 验证中：%s %s (在数据库中活跃但缓存为无效)", symbol, kind)

			if r.isSymbolValidAPI(ctx, apiClient, symbol, kind) {
				// API验证成功，符号有效，清理缓存
				symbolsToRemove = append(symbolsToRemove, symbolKind)
				log.Printf("[RedisCache] ✅ 验证成功，清理缓存：%s %s 已确认有效", symbol, kind)
			} else {
				// API验证失败，保留缓存
				symbolsToKeep = append(symbolsToKeep, symbolKind)
				log.Printf("[RedisCache] ⚠️ 验证失败，保留缓存：%s %s API确认无效", symbol, kind)
			}
		}
	}

	// 清理需要移除的符号
	if len(symbolsToRemove) > 0 {
		ctx := context.Background()
		keysToDelete := make([]string, len(symbolsToRemove))
		for i, symbolKind := range symbolsToRemove {
			keysToDelete[i] = r.buildKeyBySymbolKind(symbolKind)
		}

		deletedCount, err := r.client.Del(ctx, keysToDelete...).Result()
		if err != nil {
			return fmt.Errorf("failed to delete validated invalid symbols: %w", err)
		}

		log.Printf("[RedisCache] 🗑️ Successfully cleaned up %d invalid symbols from cache", deletedCount)
	}

	// 输出清理结果统计
	log.Printf("[RedisCache] 📊 Cleanup completed: removed %d, kept %d invalid symbols",
		len(symbolsToRemove), len(symbolsToKeep))

	return nil
}

// isSymbolValidAPI 通过API验证符号是否有效
func (r *RedisInvalidSymbolCache) isSymbolValidAPI(ctx context.Context, apiClient *BinanceAPIClient, symbol, kind string) bool {
	// 创建一个短超时的上下文用于验证（5秒超时，避免阻塞太久）
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 尝试获取最近1分钟的K线数据来验证符号有效性
	// 只获取1条记录，避免不必要的网络开销
	klines, err := apiClient.FetchKlines(verifyCtx, symbol, kind, "1m", 1)

	if err != nil {
		// 记录验证失败的原因（用于调试）
		log.Printf("[RedisCache] 🔍 API验证失败: %s %s - %v", symbol, kind, err)
		return false
	}

	if len(klines) == 0 {
		log.Printf("[RedisCache] 🔍 API验证失败: %s %s - 无返回数据", symbol, kind)
		return false
	}

	log.Printf("[RedisCache] 🔍 API验证成功: %s %s", symbol, kind)
	return true
}

// getValidSymbols 获取数据库中所有有效的USDT交易对
// getValidSymbols 获取所有活跃的USDT交易对（不区分市场类型）
func getValidSymbols(db *gorm.DB) (map[string]bool, error) {
	var symbols []string
	err := db.Table("binance_exchange_info").
		Where("quote_asset = ? AND status = ? AND is_active = ?",
			"USDT", "TRADING", true). // 只获取活跃的交易对
		Order("symbol").
		Pluck("symbol", &symbols).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, symbol := range symbols {
		result[symbol] = true
	}

	return result, nil
}

// getValidSymbolsByMarket 按市场类型获取活跃的USDT交易对
func getValidSymbolsByMarket(db *gorm.DB) (map[string]map[string]bool, error) {
	result := map[string]map[string]bool{
		"spot":    make(map[string]bool),
		"futures": make(map[string]bool),
	}

	// 获取现货活跃交易对
	var spotSymbols []string
	err := db.Table("binance_exchange_info").
		Where("quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?",
			"USDT", "TRADING", "spot", true).
		Order("symbol").
		Pluck("symbol", &spotSymbols).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get spot symbols: %w", err)
	}

	for _, symbol := range spotSymbols {
		result["spot"][symbol] = true
	}

	// 获取期货活跃交易对
	var futuresSymbols []string
	err = db.Table("binance_exchange_info").
		Where("quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?",
			"USDT", "TRADING", "futures", true).
		Order("symbol").
		Pluck("symbol", &futuresSymbols).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get futures symbols: %w", err)
	}

	for _, symbol := range futuresSymbols {
		result["futures"][symbol] = true
	}

	log.Printf("[RedisCache] 📊 Valid symbols by market - spot: %d, futures: %d",
		len(spotSymbols), len(futuresSymbols))

	return result, nil
}

// buildKeyBySymbolKind 根据symbol_kind字符串构建Redis键
func (r *RedisInvalidSymbolCache) buildKeyBySymbolKind(symbolKind string) string {
	return r.prefix + symbolKind
}

// buildKey 构建Redis键
func (r *RedisInvalidSymbolCache) buildKey(symbol, kind string) string {
	return fmt.Sprintf("%s%s_%s", r.prefix, symbol, kind)
}
