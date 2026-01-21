package server

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"

	"github.com/go-redis/redis/v8"
)

// RecommendationCache 推荐缓存
type RecommendationCache struct {
	mu              sync.RWMutex
	recommendations map[string]*CachedRecommendations
	flowData        map[string]*CachedFlowData
	marketState     *CachedMarketState
	blacklist       []string
	lastUpdate      time.Time
	cacheDuration   time.Duration

	// 新增：Redis缓存支持
	redisClient  *redis.Client
	redisEnabled bool

	// 新增：预计算支持
	precomputeTasks map[string]*PrecomputeTask
	precomputeTTL   time.Duration

	// 新增：性能统计
	hitCount   int64
	missCount  int64
	statsMutex sync.RWMutex
}

// CachedRecommendations 缓存的推荐结果
type CachedRecommendations struct {
	Data      []pdb.CoinRecommendation
	Timestamp time.Time
	Kind      string
	Limit     int
}

// CachedFlowData 缓存的资金流数据
type CachedFlowData struct {
	Data      map[string]float64
	Timestamp time.Time
}

// CachedMarketState 缓存的市场状态
type CachedMarketState struct {
	State     MarketState
	Timestamp time.Time
}

// NewRecommendationCache 创建推荐缓存
func NewRecommendationCache(cacheDuration time.Duration) *RecommendationCache {
	return &RecommendationCache{
		recommendations: make(map[string]*CachedRecommendations),
		flowData:        make(map[string]*CachedFlowData),
		cacheDuration:   cacheDuration,
	}
}

// GetRecommendations 获取缓存的推荐
func (rc *RecommendationCache) GetRecommendations(kind string, limit int) ([]pdb.CoinRecommendation, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	cacheKey := rc.generateCacheKey(kind, limit)
	cached, exists := rc.recommendations[cacheKey]

	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Since(cached.Timestamp) > rc.cacheDuration {
		return nil, false
	}

	// 返回副本避免外部修改
	result := make([]pdb.CoinRecommendation, len(cached.Data))
	copy(result, cached.Data)
	return result, true
}

// SetRecommendations 设置推荐缓存
func (rc *RecommendationCache) SetRecommendations(kind string, limit int, recommendations []pdb.CoinRecommendation) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	cacheKey := rc.generateCacheKey(kind, limit)

	// 创建副本避免外部修改
	cached := &CachedRecommendations{
		Data:      make([]pdb.CoinRecommendation, len(recommendations)),
		Timestamp: time.Now(),
		Kind:      kind,
		Limit:     limit,
	}
	copy(cached.Data, recommendations)

	rc.recommendations[cacheKey] = cached

	// 限制缓存大小
	rc.cleanupExpiredCache(context.Background())
}

// GetFlowData 获取缓存的资金流数据
func (rc *RecommendationCache) GetFlowData() (map[string]float64, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// 检查所有缓存的资金流数据，取最新的
	var latestData *CachedFlowData
	for _, data := range rc.flowData {
		if latestData == nil || data.Timestamp.After(latestData.Timestamp) {
			latestData = data
		}
	}

	if latestData == nil {
		return nil, false
	}

	// 检查是否过期
	if time.Since(latestData.Timestamp) > rc.cacheDuration {
		return nil, false
	}

	// 返回副本
	result := make(map[string]float64)
	for k, v := range latestData.Data {
		result[k] = v
	}
	return result, true
}

// SetFlowData 设置资金流数据缓存
func (rc *RecommendationCache) SetFlowData(data map[string]float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// 创建副本
	cached := &CachedFlowData{
		Data:      make(map[string]float64),
		Timestamp: time.Now(),
	}
	for k, v := range data {
		cached.Data[k] = v
	}

	// 按时间戳作为键
	key := fmt.Sprintf("flow_%d", cached.Timestamp.Unix())
	rc.flowData[key] = cached

	// 限制资金流缓存数量
	if len(rc.flowData) > 5 {
		rc.cleanupOldFlowData()
	}
}

// GetMarketState 获取缓存的市场状态
func (rc *RecommendationCache) GetMarketState() (MarketState, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if rc.marketState == nil {
		return MarketState{}, false
	}

	// 检查是否过期
	if time.Since(rc.marketState.Timestamp) > rc.cacheDuration {
		return MarketState{}, false
	}

	return rc.marketState.State, true
}

// SetMarketState 设置市场状态缓存
func (rc *RecommendationCache) SetMarketState(state MarketState) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.marketState = &CachedMarketState{
		State:     state,
		Timestamp: time.Now(),
	}
}

// GetBlacklist 获取缓存的黑名单
func (rc *RecommendationCache) GetBlacklist() ([]string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// 黑名单缓存1小时
	if time.Since(rc.lastUpdate) > time.Hour {
		return nil, false
	}

	// 返回副本
	result := make([]string, len(rc.blacklist))
	copy(result, rc.blacklist)
	return result, true
}

// SetBlacklist 设置黑名单缓存
func (rc *RecommendationCache) SetBlacklist(blacklist []string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.blacklist = make([]string, len(blacklist))
	copy(rc.blacklist, blacklist)
	rc.lastUpdate = time.Now()
}

// generateCacheKey 生成缓存键（兼容旧接口）
func (rc *RecommendationCache) generateCacheKey(kind string, limit int) string {
	params := RecommendationQueryParams{Kind: kind, Limit: limit}
	return rc.generateCacheKeyFromParams(params)
}

// generateCacheKeyFromParams 从参数生成缓存键
func (rc *RecommendationCache) generateCacheKeyFromParams(params RecommendationQueryParams) string {
	paramBytes, _ := json.Marshal(params)
	hash := md5.Sum(paramBytes)
	return fmt.Sprintf("rec:%x", hash)
}

// cleanupOldFlowData 清理旧的资金流数据
func (rc *RecommendationCache) cleanupOldFlowData() {
	// 保留最新的5个
	type flowItem struct {
		key  string
		time time.Time
	}

	items := make([]flowItem, 0, len(rc.flowData))
	for key, data := range rc.flowData {
		items = append(items, flowItem{key: key, time: data.Timestamp})
	}

	// 按时间排序
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].time.After(items[j].time) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// 删除最旧的
	for i := 0; i < len(items)-5; i++ {
		delete(rc.flowData, items[i].key)
	}
}

// Clear 清空所有缓存
func (rc *RecommendationCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.recommendations = make(map[string]*CachedRecommendations)
	rc.flowData = make(map[string]*CachedFlowData)
	rc.marketState = nil
	rc.blacklist = nil
	rc.lastUpdate = time.Time{}
}

// Stats 获取缓存统计信息
func (rc *RecommendationCache) Stats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	rc.statsMutex.RLock()
	defer rc.statsMutex.RUnlock()

	total := rc.hitCount + rc.missCount
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(rc.hitCount) / float64(total)
	}

	return map[string]interface{}{
		"recommendations_count": len(rc.recommendations),
		"flow_data_count":       len(rc.flowData),
		"has_market_state":      rc.marketState != nil,
		"blacklist_count":       len(rc.blacklist),
		"cache_duration":        rc.cacheDuration.String(),
		"last_update":           rc.lastUpdate.Format(time.RFC3339),
		"redis_enabled":         rc.redisEnabled,
		"precompute_tasks":      len(rc.precomputeTasks),
		"hit_count":             rc.hitCount,
		"miss_count":            rc.missCount,
		"hit_rate":              hitRate,
		"total_requests":        total,
	}
}

// ==================== 新增：预计算和Redis缓存功能 ====================

// PrecomputeTask 预计算任务
type PrecomputeTask struct {
	TaskID      string
	UserID      *uint
	QueryParams RecommendationQueryParams
	Status      string // pending, running, completed, failed
	CreatedAt   time.Time
	CompletedAt *time.Time
	Result      *CachedRecommendations
	Error       string
}

// RecommendationQueryParams 推荐查询参数
type RecommendationQueryParams struct {
	UserID    *uint   `json:"user_id,omitempty"`
	Kind      string  `json:"kind"`
	Limit     int     `json:"limit"`
	MinScore  float64 `json:"min_score,omitempty"`
	TimeRange string  `json:"time_range,omitempty"`
	SortBy    string  `json:"sort_by,omitempty"`
}

// EnhancedRecommendationCache 创建增强版推荐缓存（支持Redis和预计算）
func NewEnhancedRecommendationCache(cacheDuration time.Duration, redisAddr string, precomputeTTL time.Duration) (*RecommendationCache, error) {
	rc := &RecommendationCache{
		recommendations: make(map[string]*CachedRecommendations),
		flowData:        make(map[string]*CachedFlowData),
		cacheDuration:   cacheDuration,
		precomputeTasks: make(map[string]*PrecomputeTask),
		precomputeTTL:   precomputeTTL,
	}

	// 初始化Redis客户端
	if redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{
			Addr: redisAddr,
			DB:   3, // 使用DB 3存储推荐缓存
		})

		// 测试连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Redis连接失败，将使用本地缓存: %v", err)
		} else {
			rc.redisClient = rdb
			rc.redisEnabled = true
			log.Printf("Redis缓存已启用")
		}
	}

	// 注意：预计算任务处理器已移至独立的investment服务
	// 如需启动预计算处理器，请运行: ./investment -service=investment -mode=scheduler
	// go rc.startPrecomputeProcessor()

	return rc, nil
}

// GetRecommendationsWithCache 获取推荐结果（带多级缓存策略）
func (rc *RecommendationCache) GetRecommendationsWithCache(ctx context.Context, params RecommendationQueryParams) ([]pdb.CoinRecommendation, error) {
	cacheKey := rc.generateCacheKeyFromParams(params)
	// log.Printf("缓存调试: 缓存键=%s, Redis启用=%v", cacheKey, rc.redisEnabled)

	// 1. 尝试从本地缓存获取
	if cached, found := rc.getLocalCache(cacheKey); found && rc.isCacheValid(cached) {
		rc.recordHit()
		// log.Printf("缓存调试: 本地缓存命中")
		return cached.Data, nil
	}
	// log.Printf("缓存调试: 本地缓存未命中")

	// 2. 尝试从Redis缓存获取
	if rc.redisEnabled {
		if cached, err := rc.getRedisCache(ctx, cacheKey); err == nil && cached != nil && rc.isCacheValid(cached) {
			rc.recordHit()
			// 同步到本地缓存
			rc.setLocalCache(cacheKey, cached)
			// log.Printf("缓存调试: Redis缓存命中")
			return cached.Data, nil
		} else {
			// log.Printf("缓存调试: Redis缓存未命中, 错误=%v", err)
		}
	} else {
		// log.Printf("缓存调试: Redis未启用")
	}

	rc.recordMiss()

	// 3. 检查预计算结果
	if precomputed, err := rc.getPrecomputedResult(ctx, params); err == nil && precomputed != nil && rc.isCacheValid(precomputed) {
		rc.setCache(ctx, cacheKey, precomputed)
		// log.Printf("缓存调试: 预计算结果命中")
		return precomputed.Data, nil
	} else {
		// log.Printf("缓存调试: 预计算结果未命中, 错误=%v", err)
	}

	// 4. 都没有，触发异步预计算
	// log.Printf("缓存调试: 触发预计算")
	go rc.triggerPrecomputation(params)

	// 5. 返回实时计算结果（降级策略）
	// log.Printf("缓存调试: 执行实时计算")
	recommendations, err := rc.computeRealtimeRecommendations(params)
	if err != nil {
		log.Printf("实时计算推荐失败: %v", err)
		return nil, err
	}

	// 将实时计算结果缓存起来，避免下次重复计算
	cacheKey = rc.generateCacheKeyFromParams(params)
	cachedRec := &CachedRecommendations{
		Data:      recommendations,
		Timestamp: time.Now(),
		Kind:      params.Kind,
		Limit:     params.Limit,
	}
	rc.setCache(context.Background(), cacheKey, cachedRec)
	// log.Printf("缓存调试: 实时计算结果已缓存，下次请求将命中缓存")

	return recommendations, nil
}

// getLocalCache 从本地缓存获取
func (rc *RecommendationCache) getLocalCache(key string) (*CachedRecommendations, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	cached, exists := rc.recommendations[key]
	return cached, exists
}

// setLocalCache 设置本地缓存
func (rc *RecommendationCache) setLocalCache(key string, cached *CachedRecommendations) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.recommendations[key] = cached
}

// getRedisCache 从Redis缓存获取
func (rc *RecommendationCache) getRedisCache(ctx context.Context, key string) (*CachedRecommendations, error) {
	if !rc.redisEnabled {
		return nil, fmt.Errorf("redis not enabled")
	}

	cacheKey := fmt.Sprintf("cache:%s", key)
	val, err := rc.redisClient.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	var cached CachedRecommendations
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// setCache 设置缓存（同时设置本地和Redis）
func (rc *RecommendationCache) setCache(ctx context.Context, key string, cached *CachedRecommendations) error {
	// 设置本地缓存
	rc.setLocalCache(key, cached)

	// 设置Redis缓存
	if rc.redisEnabled {
		cacheKey := fmt.Sprintf("cache:%s", key)
		cacheBytes, _ := json.Marshal(cached)

		ttl := rc.getCacheTTL(cached)
		return rc.redisClient.Set(ctx, cacheKey, cacheBytes, ttl).Err()
	}

	return nil
}

// getCacheTTL 获取缓存TTL
func (rc *RecommendationCache) getCacheTTL(cached *CachedRecommendations) time.Duration {
	// 用户个性化缓存时间较短
	if cached.Kind == "personalized" {
		return 10 * time.Minute
	}
	// 全局缓存时间较长（配合预热机制）
	return 15 * time.Minute
}

// isCacheValid 检查缓存是否有效
func (rc *RecommendationCache) isCacheValid(cached *CachedRecommendations) bool {
	if cached == nil {
		return false
	}
	return time.Since(cached.Timestamp) < rc.cacheDuration
}

// getPrecomputedResult 获取预计算结果
func (rc *RecommendationCache) getPrecomputedResult(ctx context.Context, params RecommendationQueryParams) (*CachedRecommendations, error) {
	if !rc.redisEnabled {
		return nil, fmt.Errorf("redis not enabled")
	}

	taskKey := rc.generatePrecomputeTaskKey(params)
	taskJSON, err := rc.redisClient.Get(ctx, fmt.Sprintf("precompute:task:%s", taskKey)).Result()
	if err != nil {
		return nil, err
	}

	var task PrecomputeTask
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return nil, err
	}

	if task.Status == "completed" && task.Result != nil {
		return task.Result, nil
	}

	return nil, fmt.Errorf("precompute task not ready")
}

// triggerPrecomputation 触发预计算（带并发控制）
func (rc *RecommendationCache) triggerPrecomputation(params RecommendationQueryParams) {
	if !rc.redisEnabled {
		log.Printf("预计算调试: Redis未启用，跳过预计算")
		return
	}

	ctx := context.Background()
	taskID := rc.generatePrecomputeTaskKey(params)

	// 检查是否已经有相同的任务在进行中
	existingTaskKey := fmt.Sprintf("precompute:task:%s", taskID)
	if exists, _ := rc.redisClient.Exists(ctx, existingTaskKey).Result(); exists > 0 {
		log.Printf("预计算调试: 任务 %s 已存在，跳过重复创建", taskID)
		return
	}

	// 检查是否已经在队列中
	queueKey := "precompute:queue"
	queuedTasks, _ := rc.redisClient.LRange(ctx, queueKey, 0, -1).Result()
	for _, queuedTaskID := range queuedTasks {
		if queuedTaskID == taskID {
			log.Printf("预计算调试: 任务 %s 已在队列中，跳过重复添加", taskID)
			return
		}
	}

	task := &PrecomputeTask{
		TaskID:      taskID,
		UserID:      params.UserID,
		QueryParams: params,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	log.Printf("预计算调试: 创建预计算任务 %s", task.TaskID)

	// 保存任务到Redis
	taskBytes, _ := json.Marshal(task)
	taskKey := fmt.Sprintf("precompute:task:%s", task.TaskID)

	ctx = context.Background()
	err := rc.redisClient.Set(ctx, taskKey, taskBytes, rc.precomputeTTL).Err()
	if err != nil {
		log.Printf("预计算调试: 保存任务失败: %v", err)
		return
	}

	// 提交到预计算队列
	err = rc.redisClient.LPush(ctx, "precompute:queue", task.TaskID).Err()
	if err != nil {
		log.Printf("预计算调试: 提交队列失败: %v", err)
		return
	}

	log.Printf("预计算调试: 任务已提交到队列")
}

// generatePrecomputeTaskKey 生成预计算任务键
func (rc *RecommendationCache) generatePrecomputeTaskKey(params RecommendationQueryParams) string {
	paramBytes, _ := json.Marshal(params)
	hash := md5.Sum(paramBytes)
	return fmt.Sprintf("task:%x", hash)
}

// computeRealtimeRecommendations 实时计算推荐（降级策略）
func (rc *RecommendationCache) computeRealtimeRecommendations(params RecommendationQueryParams) ([]pdb.CoinRecommendation, error) {
	// 这里应该调用实际的推荐计算逻辑
	// 为了演示，我们使用现有的计算方法

	log.Printf("实时计算推荐: %+v", params)

	// 模拟计算延迟
	time.Sleep(200 * time.Millisecond)

	// 调用现有方法获取推荐
	if cached, exists := rc.GetRecommendations(params.Kind, params.Limit); exists {
		// 将结果缓存起来
		cacheKey := rc.generateCacheKeyFromParams(params)
		cachedRec := &CachedRecommendations{
			Data:      cached,
			Timestamp: time.Now(),
			Kind:      params.Kind,
			Limit:     params.Limit,
		}
		rc.setCache(context.Background(), cacheKey, cachedRec)
		// log.Printf("缓存调试: 实时计算结果已缓存")
		return cached, nil
	}

	// 如果还是没有结果，返回空结果
	// log.Printf("缓存调试: 无法获取推荐结果，返回空结果")
	return []pdb.CoinRecommendation{}, nil
}

// startPrecomputeProcessor 启动预计算处理器
func (rc *RecommendationCache) startPrecomputeProcessor() {
	if !rc.redisEnabled {
		log.Printf("预计算调试: Redis未启用，预计算处理器不启动")
		return
	}

	log.Printf("预计算调试: 启动预计算处理器")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Printf("预计算调试: 检查待处理任务")
		rc.processPendingTasks()
	}
}

// processPendingTasks 处理待处理的任务
func (rc *RecommendationCache) processPendingTasks() {
	if !rc.redisEnabled {
		return
	}

	ctx := context.Background()

	// 获取待处理任务
	taskID, err := rc.redisClient.RPop(ctx, "precompute:queue").Result()
	if err != nil {
		return // 队列为空
	}

	// 获取任务详情
	taskKey := fmt.Sprintf("precompute:task:%s", taskID)
	taskJSON, err := rc.redisClient.Get(ctx, taskKey).Result()
	if err != nil {
		log.Printf("获取预计算任务失败: %v", err)
		return
	}

	var task PrecomputeTask
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		log.Printf("解析预计算任务失败: %v", err)
		return
	}

	// 更新任务状态为运行中
	task.Status = "running"
	taskBytes, _ := json.Marshal(task)
	rc.redisClient.Set(ctx, taskKey, taskBytes, rc.redisClient.TTL(ctx, taskKey).Val())

	// 异步执行预计算
	go rc.executePrecomputeTask(task)
}

// executePrecomputeTask 执行预计算任务
func (rc *RecommendationCache) executePrecomputeTask(task PrecomputeTask) {
	ctx := context.Background()

	// 执行推荐计算
	recommendations, err := rc.computeRealtimeRecommendations(task.QueryParams)
	if err != nil {
		log.Printf("预计算任务执行失败: %v", err)
		task.Status = "failed"
		task.Error = err.Error()
	} else {
		task.Status = "completed"
		task.Result = &CachedRecommendations{
			Data:      recommendations,
			Timestamp: time.Now(),
			Kind:      task.QueryParams.Kind,
			Limit:     task.QueryParams.Limit,
		}
	}

	now := time.Now()
	task.CompletedAt = &now

	// 保存结果
	taskBytes, _ := json.Marshal(task)
	taskKey := fmt.Sprintf("precompute:task:%s", task.TaskID)
	rc.redisClient.Set(ctx, taskKey, taskBytes, 1*time.Hour) // 任务结果保留1小时

	// 将结果同步到缓存
	if task.Result != nil {
		cacheKey := rc.generateCacheKeyFromParams(task.QueryParams)
		rc.setCache(ctx, cacheKey, task.Result)
	}
}

// recordHit 记录缓存命中
func (rc *RecommendationCache) recordHit() {
	rc.statsMutex.Lock()
	defer rc.statsMutex.Unlock()
	rc.hitCount++
}

// recordMiss 记录缓存未命中
func (rc *RecommendationCache) recordMiss() {
	rc.statsMutex.Lock()
	defer rc.statsMutex.Unlock()
	rc.missCount++
}

// InvalidateUserCache 使指定用户的缓存失效
func (rc *RecommendationCache) InvalidateUserCache(ctx context.Context, userID uint) error {
	if !rc.redisEnabled {
		return nil
	}

	// 删除Redis中该用户的所有缓存
	pattern := fmt.Sprintf("cache:rec:*\"user_id\":%d*", userID)
	keys, err := rc.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return rc.redisClient.Del(ctx, keys...).Err()
	}

	return nil
}

// WarmupCache 预热热门查询缓存
func (rc *RecommendationCache) WarmupCache(ctx context.Context, popularQueries []RecommendationQueryParams) error {
	log.Printf("开始缓存预热，共%d个查询", len(popularQueries))

	for _, query := range popularQueries {
		go func(q RecommendationQueryParams) {
			cacheKey := rc.generateCacheKeyFromParams(q)

			// 检查是否已缓存
			if _, err := rc.getRedisCache(ctx, cacheKey); err != nil {
				// 缓存不存在，计算并缓存
				if recommendations, err := rc.computeRealtimeRecommendations(q); err == nil {
					cached := &CachedRecommendations{
						Data:      recommendations,
						Timestamp: time.Now(),
						Kind:      q.Kind,
						Limit:     q.Limit,
					}
					rc.setCache(ctx, cacheKey, cached)
				}
			}
		}(query)
	}

	return nil
}

// cleanupExpiredCache 清理过期缓存（定期调用）
func (rc *RecommendationCache) cleanupExpiredCache(ctx context.Context) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// 检查本地缓存
	for key, cached := range rc.recommendations {
		if now.Sub(cached.Timestamp) > rc.cacheDuration {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// 删除过期本地缓存
	for _, key := range expiredKeys {
		delete(rc.recommendations, key)
		log.Printf("缓存清理: 删除过期本地缓存 %s", key)
	}

	// 清理Redis过期缓存（通过TTL机制自动清理，这里主要清理异常数据）
	if rc.redisEnabled {
		pattern := "recommendation:cache:*"
		keys, err := rc.redisClient.Keys(ctx, pattern).Result()
		if err == nil {
			for _, key := range keys {
				// 检查TTL，如果为-2表示已过期
				ttl, err := rc.redisClient.TTL(ctx, key).Result()
				if err == nil && ttl.Seconds() < -1 { // -2表示已过期，-1表示无TTL
					rc.redisClient.Del(ctx, key)
					log.Printf("缓存清理: 删除过期Redis缓存 %s", key)
				}
			}
		}
	}

	if len(expiredKeys) > 0 {
		log.Printf("缓存清理完成: 清理了 %d 个过期缓存项", len(expiredKeys))
	}

	return nil
}

// Close 关闭缓存管理器
func (rc *RecommendationCache) Close() error {
	if rc.redisClient != nil {
		return rc.redisClient.Close()
	}
	return nil
}
