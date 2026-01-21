package server

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FeatureCacheManager 特征缓存管理器
type FeatureCacheManager struct {
	mu           sync.RWMutex
	featureCache map[string]*CachedFeatureSet
	maxSize      int
	maxAge       time.Duration
	hitCount     int64
	missCount    int64
	server       *Server // 用于数据库访问
}

// CachedFeatureSet 缓存的特征集合
type CachedFeatureSet struct {
	Symbol      string
	TimeWindow  int // 时间窗口（小时）
	DataPoints  int // 数据点数量
	Features    map[string]float64
	Quality     FeatureQuality
	ComputedAt  time.Time
	AccessCount int64
}

// NewFeatureCacheManager 创建特征缓存管理器
func NewFeatureCacheManager(server *Server) *FeatureCacheManager {
	return &FeatureCacheManager{
		featureCache: make(map[string]*CachedFeatureSet),
		maxSize:      5000,          // 最大缓存5000个特征集合
		maxAge:       2 * time.Hour, // 缓存2小时
		server:       server,        // 用于数据库访问
	}
}

// generateFeatureKey 生成特征缓存键
func (fcm *FeatureCacheManager) generateFeatureKey(symbol string, timeWindow int, data []MarketData) string {
	if len(data) == 0 {
		return ""
	}

	// 使用符号、时间窗口、数据点数量和最新时间戳生成键
	keyData := fmt.Sprintf("%s_%d_%d_%s",
		symbol,
		timeWindow,
		len(data),
		data[len(data)-1].LastUpdated.Format("2006-01-02-15"))

	hash := md5.Sum([]byte(keyData))
	return fmt.Sprintf("%x", hash)
}

// GetFeatures 获取缓存的特征
func (fcm *FeatureCacheManager) GetFeatures(key string) map[string]float64 {
	// 解析缓存键格式: symbol_timeWindow (可能是哈希或直接格式)
	parts := fcm.parseFeatureKey(key)
	if parts == nil {
		log.Printf("[FeatureCache] 无法解析缓存键格式: %s", key)
		return nil
	}

	// 使用解析出的 symbol 和 timeWindow 生成标准缓存键
	cacheKey := fmt.Sprintf("%s_%d", parts.symbol, parts.timeWindow)

	fcm.mu.RLock()

	// 首先检查内存缓存
	if cached, exists := fcm.featureCache[cacheKey]; exists {
		// 检查是否过期
		if time.Since(cached.ComputedAt) > fcm.maxAge {
			fcm.mu.RUnlock()
			fcm.mu.Lock()
			delete(fcm.featureCache, cacheKey)
			fcm.mu.Unlock()
			return nil
		}

		// 更新访问计数
		cached.AccessCount++
		fcm.hitCount++
		fcm.mu.RUnlock()
		return cached.Features
	}

	fcm.mu.RUnlock()

	// 内存缓存未命中，尝试从数据库获取
	dbFeatures, err := fcm.getFeaturesFromDatabase(parts.symbol, parts.timeWindow)
	if err != nil {
		log.Printf("[FeatureCache] 从数据库获取缓存失败 %s: %v", cacheKey, err)
		return nil
	}

	if dbFeatures == nil {
		return nil
	}

	// 创建缓存对象并存入内存
	cached := &CachedFeatureSet{
		Symbol:      parts.symbol,
		TimeWindow:  parts.timeWindow,
		DataPoints:  0, // 从数据库加载时不确定数据点数
		Features:    dbFeatures,
		Quality:     FeatureQuality{Overall: 0.8}, // 默认质量评分
		ComputedAt:  time.Now(),                   // 简化处理，使用当前时间
		AccessCount: 1,
	}

	fcm.mu.Lock()
	fcm.featureCache[cacheKey] = cached
	fcm.mu.Unlock()

	fcm.hitCount++
	log.Printf("[FeatureCache] 从数据库加载特征缓存成功: %s", cacheKey)
	return dbFeatures
}

// parseFeatureKey 解析特征缓存键
func (fcm *FeatureCacheManager) parseFeatureKey(key string) *struct {
	symbol     string
	timeWindow int
} {
	// 简单解析逻辑，假设格式为 symbol_timeWindow
	// 这里可以根据实际需要改进解析逻辑
	parts := strings.Split(key, "_")
	if len(parts) < 2 {
		return nil
	}

	symbol := parts[0]
	timeWindowStr := parts[len(parts)-1]

	timeWindow := 24 // 默认24小时
	if tw, err := strconv.Atoi(timeWindowStr); err == nil {
		timeWindow = tw
	}

	return &struct {
		symbol     string
		timeWindow int
	}{symbol: symbol, timeWindow: timeWindow}
}

// SetFeatures 缓存特征
func (fcm *FeatureCacheManager) SetFeatures(key string, features map[string]float64) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	now := time.Now()
	parts := fcm.parseFeatureKey(key)

	// 检查解析结果是否有效
	if parts == nil {
		log.Printf("[FeatureCache] 无法解析缓存键格式: %s", key)
		return
	}

	cached := &CachedFeatureSet{
		Symbol:     parts.symbol,
		TimeWindow: parts.timeWindow,
		DataPoints: 0, // 预计算服务不跟踪具体数据点数
		Features:   features,
		Quality:    FeatureQuality{Overall: 0.8}, // 默认质量评分
		ComputedAt: now,
	}

	fcm.featureCache[key] = cached

	// 清理过期数据
	fcm.cleanupExpiredFeatures()

	// 异步写入数据库
	go func() {
		if err := fcm.saveFeaturesToDatabase(parts.symbol, parts.timeWindow, features, cached.Quality); err != nil {
			log.Printf("[FeatureCache] 保存到数据库失败 %s: %v", key, err)
		}
	}()
}

// SetDetailedFeatures 缓存详细的特征信息
func (fcm *FeatureCacheManager) SetDetailedFeatures(key string, symbol string, timeWindow int, dataPoints int, features map[string]float64, quality FeatureQuality) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	now := time.Now()
	cached := &CachedFeatureSet{
		Symbol:      symbol,
		TimeWindow:  timeWindow,
		DataPoints:  dataPoints,
		Features:    features,
		Quality:     quality,
		ComputedAt:  now,
		AccessCount: 0,
	}

	// 使用 symbol_timeWindow 格式作为缓存键，而不是哈希
	cacheKey := fmt.Sprintf("%s_%d", symbol, timeWindow)
	fcm.featureCache[cacheKey] = cached

	// 清理过期数据
	fcm.cleanupExpiredFeatures()

	// 异步写入数据库
	go func() {
		if err := fcm.saveFeaturesToDatabase(symbol, timeWindow, features, quality); err != nil {
			log.Printf("[FeatureCache] 保存详细特征到数据库失败 %s_%d: %v", symbol, timeWindow, err)
		}
	}()
}

// cleanupExpiredFeatures 清理过期特征
func (fcm *FeatureCacheManager) cleanupExpiredFeatures() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, cached := range fcm.featureCache {
		if now.Sub(cached.ComputedAt) > fcm.maxAge {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(fcm.featureCache, key)
	}

	// 如果缓存过大，清理最少访问的条目
	if len(fcm.featureCache) > fcm.maxSize {
		fcm.evictLeastAccessed()
	}
}

// evictLeastAccessed 清除最少访问的条目
func (fcm *FeatureCacheManager) evictLeastAccessed() {
	if len(fcm.featureCache) <= fcm.maxSize {
		return
	}

	// 收集所有条目
	type cacheEntry struct {
		key         string
		accessCount int64
		computedAt  time.Time
	}

	entries := make([]cacheEntry, 0, len(fcm.featureCache))
	for key, cached := range fcm.featureCache {
		entries = append(entries, cacheEntry{
			key:         key,
			accessCount: cached.AccessCount,
			computedAt:  cached.ComputedAt,
		})
	}

	// 按访问次数和计算时间排序（先按访问次数升序，再按时间升序）
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			// 如果访问次数相同，比较计算时间（越早计算的优先清除）
			shouldSwap := false
			if entries[i].accessCount > entries[j].accessCount {
				shouldSwap = true
			} else if entries[i].accessCount == entries[j].accessCount {
				if entries[i].computedAt.Before(entries[j].computedAt) {
					shouldSwap = true
				}
			}

			if shouldSwap {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// 清除最少访问的20%条目
	removeCount := len(entries) / 5
	if removeCount < 1 {
		removeCount = 1
	}

	for i := 0; i < removeCount && i < len(entries); i++ {
		delete(fcm.featureCache, entries[i].key)
	}

	log.Printf("[FeatureCacheManager] 清除 %d 个最少访问的特征缓存条目", removeCount)
}

// GetStats 获取缓存统计信息
func (fcm *FeatureCacheManager) GetStats() map[string]interface{} {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	totalFeatures := 0
	qualitySum := 0.0
	validQualityCount := 0

	for _, cached := range fcm.featureCache {
		totalFeatures += len(cached.Features)
		if cached.Quality.Overall > 0 {
			qualitySum += cached.Quality.Overall
			validQualityCount++
		}
	}

	avgQuality := 0.0
	if validQualityCount > 0 {
		avgQuality = qualitySum / float64(validQualityCount)
	}

	hitRate := 0.0
	totalRequests := fcm.hitCount + fcm.missCount
	if totalRequests > 0 {
		hitRate = float64(fcm.hitCount) / float64(totalRequests)
	}

	return map[string]interface{}{
		"cache_size":     len(fcm.featureCache),
		"max_cache_size": fcm.maxSize,
		"cache_max_age":  fcm.maxAge.String(),
		"total_features": totalFeatures,
		"avg_features_per_set": func() float64 {
			if len(fcm.featureCache) == 0 {
				return 0
			}
			return float64(totalFeatures) / float64(len(fcm.featureCache))
		}(),
		"avg_quality": avgQuality,
		"hit_count":   fcm.hitCount,
		"miss_count":  fcm.missCount,
		"hit_rate":    fmt.Sprintf("%.2f%%", hitRate*100),
	}
}

// CleanupExpiredFeatures 清理过期特征（公开方法）
func (fcm *FeatureCacheManager) CleanupExpiredFeatures() {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	fcm.cleanupExpiredFeatures()
	log.Printf("[FeatureCacheManager] 完成过期特征清理")
}

// GetPopularSymbols 获取最受欢迎的符号列表
func (fcm *FeatureCacheManager) GetPopularSymbols(limit int) []string {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	symbolCounts := make(map[string]int64)

	for _, cached := range fcm.featureCache {
		symbolCounts[cached.Symbol] += cached.AccessCount
	}

	type symbolCount struct {
		symbol string
		count  int64
	}

	counts := make([]symbolCount, 0, len(symbolCounts))
	for symbol, count := range symbolCounts {
		counts = append(counts, symbolCount{symbol: symbol, count: count})
	}

	// 按访问次数降序排序
	for i := 0; i < len(counts)-1; i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[i].count < counts[j].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	result := make([]string, 0, limit)
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, counts[i].symbol)
	}

	return result
}

// getFeaturesFromDatabase 从数据库获取特征数据
func (fcm *FeatureCacheManager) getFeaturesFromDatabase(symbol string, timeWindow int) (map[string]float64, error) {
	if fcm.server == nil || fcm.server.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}

	// 查询数据库中未过期的数据
	query := `
		SELECT features FROM feature_cache
		WHERE symbol = ? AND time_window = ? AND expires_at > NOW()
		ORDER BY computed_at DESC LIMIT 1
	`

	var featuresJSON string
	err := fcm.server.db.DB().Raw(query, symbol, timeWindow).Scan(&featuresJSON).Error
	if err != nil {
		return nil, fmt.Errorf("查询特征缓存失败: %w", err)
	}

	if featuresJSON == "" {
		return nil, nil // 没有找到缓存数据
	}

	// 解析JSON数据
	var features map[string]float64
	if err := json.Unmarshal([]byte(featuresJSON), &features); err != nil {
		return nil, fmt.Errorf("解析特征数据失败: %w", err)
	}

	return features, nil
}

// saveFeaturesToDatabase 将特征数据保存到数据库
func (fcm *FeatureCacheManager) saveFeaturesToDatabase(symbol string, timeWindow int, features map[string]float64, quality FeatureQuality) error {
	if fcm.server == nil || fcm.server.db == nil {
		return fmt.Errorf("数据库连接不可用")
	}

	// 序列化特征数据
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return fmt.Errorf("序列化特征数据失败: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(fcm.maxAge)

	// 使用UPSERT操作插入或更新数据
	query := `
		INSERT INTO feature_cache (
			symbol, features, computed_at, expires_at,
			feature_count, quality_score, source, time_window, data_points
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			features = VALUES(features),
			computed_at = VALUES(computed_at),
			expires_at = VALUES(expires_at),
			feature_count = VALUES(feature_count),
			quality_score = VALUES(quality_score),
			updated_at = NOW(3)
	`

	err = fcm.server.db.DB().Exec(query,
		symbol,               // symbol
		string(featuresJSON), // features
		now,                  // computed_at
		expiresAt,            // expires_at
		len(features),        // feature_count
		quality.Overall,      // quality_score
		"computed",           // source
		timeWindow,           // time_window
		0,                    // data_points (预计算服务不确定)
	).Error

	if err != nil {
		return fmt.Errorf("保存特征缓存失败: %w", err)
	}

	log.Printf("[FeatureCache] 成功保存特征到数据库: %s_%dh (%d features)",
		symbol, timeWindow, len(features))
	return nil
}
