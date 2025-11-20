package server

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// CacheMiddleware 缓存中间件（优化版，支持新的缓存策略）
// cache: 缓存接口
// cacheType: 缓存类型（用于确定 TTL），如果为 -1 则使用 ttl 参数
// ttl: 缓存过期时间（当 cacheType 为 -1 时使用，保持向后兼容）
// keyGenerator: 缓存键生成函数，如果为 nil 则使用默认生成器
func CacheMiddleware(cache pdb.CacheInterface, cacheType pdb.CacheType, ttl time.Duration, keyGenerator func(*gin.Context) string) gin.HandlerFunc {
	if cache == nil {
		// 如果没有缓存，直接跳过
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// 只缓存 GET 请求
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// 生成缓存键
		var key string
		if keyGenerator != nil {
			key = keyGenerator(c)
		} else {
			key = defaultCacheKey(c)
		}

		// 尝试从缓存获取
		ctx := c.Request.Context()
		cached, err := cache.Get(ctx, key)
		if err == nil && len(cached) > 0 {
			// 设置响应头
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", cached)
			c.Abort()
			
			// 记录统计（简化处理）
			// keyPrefix := extractKeyPrefix(key)
			// 这里需要访问统计收集器
			
			return
		}

		// 缓存未命中，继续处理请求
		c.Header("X-Cache", "MISS")
		
		// 记录统计
		keyPrefix := extractKeyPrefix(key)
		pdb.GetCacheStats(keyPrefix) // 初始化统计
		// 这里需要访问统计收集器，暂时简化处理
		
		// 使用自定义 ResponseWriter 捕获响应
		w := &cacheResponseWriter{
			ResponseWriter: c.Writer,
			body:          make([]byte, 0),
		}
		c.Writer = w

		c.Next()

		// 只缓存成功的响应（状态码 200）
		if c.Writer.Status() == http.StatusOK && len(w.body) > 0 {
			// 根据缓存类型获取 TTL（如果 cacheType 为 -1，使用传入的 ttl）
			var cacheTTL time.Duration
			if cacheType < 0 {
				cacheTTL = ttl
			} else {
				cacheTTL = pdb.DefaultCacheTTL.GetTTL(cacheType)
			}
			
			// 优化：使用协程池异步写入缓存，避免创建过多 goroutine
			cacheKey := key
			cacheData := make([]byte, len(w.body))
			copy(cacheData, w.body)
			
			// 使用全局缓存写入池（如果存在）
			if globalCachePool != nil {
				globalCachePool.Submit(func() {
					if err := cache.Set(context.Background(), cacheKey, cacheData, cacheTTL); err != nil {
						log.Printf("[ERROR] Failed to set cache (key=%s): %v", cacheKey, err)
					}
				})
			} else {
				// 降级到直接创建 goroutine
				go func() {
					if err := cache.Set(context.Background(), cacheKey, cacheData, cacheTTL); err != nil {
						log.Printf("[ERROR] Failed to set cache (key=%s): %v", cacheKey, err)
					}
				}()
			}
		}
	}
}

// cacheResponseWriter 用于捕获响应内容
type cacheResponseWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *cacheResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// defaultCacheKey 默认缓存键生成器（优化：使用字符串构建器）
func defaultCacheKey(c *gin.Context) string {
	// 使用 URL 路径和查询参数生成键
	url := c.Request.URL.Path + "?" + c.Request.URL.RawQuery
	hash := md5.Sum([]byte(url))
	// 优化：使用字符串构建器
	return BuildCacheKeyWithHash("cache:v1:default", fmt.Sprintf("%x", hash))
}

// extractKeyPrefix 提取缓存键前缀（用于统计）
func extractKeyPrefix(key string) string {
	// 从 "cache:v1:type:..." 中提取 "type"
	parts := strings.Split(key, ":")
	if len(parts) >= 3 {
		return parts[2] // 返回类型部分
	}
	return "default"
}

// 全局缓存写入协程池（优化：限制缓存写入的并发数）
var globalCachePool *WorkerPool

// InitCachePool 初始化缓存写入协程池
func InitCachePool(maxWorkers int) {
	globalCachePool = NewWorkerPool(maxWorkers)
}

// ShutdownCachePool 关闭缓存写入协程池
func ShutdownCachePool(timeout time.Duration) error {
	if globalCachePool != nil {
		return globalCachePool.Shutdown(timeout)
	}
	return nil
}

// ==================== 专用缓存键生成器 ====================

// AnnouncementsCacheKey 公告列表缓存键（优化：使用字符串构建器）
func AnnouncementsCacheKey(c *gin.Context) string {
	q := c.Query("q")
	categories := c.Query("categories")
	page := c.Query("page")
	pageSize := c.Query("page_size")
	isEvent := c.Query("is_event")
	verified := c.Query("verified")
	sentiment := c.Query("sentiment")
	exchange := c.Query("exchange")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	
	// 优化：使用 strings.Builder 构建键
	var keyBuilder strings.Builder
	keyBuilder.Grow(200) // 预估大小
	keyBuilder.WriteString("announcements:")
	keyBuilder.WriteString(q)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(categories)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(page)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(pageSize)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(isEvent)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(verified)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(sentiment)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(exchange)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(startDate)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(endDate)
	
	key := keyBuilder.String()
	hash := md5.Sum([]byte(key))
	return BuildCacheKeyWithHash("cache:v1:announcements", fmt.Sprintf("%x", hash))
}

// MarketCacheKey 市场数据缓存键（优化：使用字符串构建器）
func MarketCacheKey(c *gin.Context) string {
	kind := c.Query("kind")
	interval := c.Query("interval")
	date := c.Query("date")
	slot := c.Query("slot")
	tz := c.Query("tz")
	
	// 优化：使用 strings.Builder 构建键
	var keyBuilder strings.Builder
	keyBuilder.Grow(100)
	keyBuilder.WriteString("market:binance:")
	keyBuilder.WriteString(kind)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(interval)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(date)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(slot)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(tz)
	
	key := keyBuilder.String()
	hash := md5.Sum([]byte(key))
	return BuildCacheKeyWithHash("cache:v1:market", fmt.Sprintf("%x", hash))
}

// TwitterPostsCacheKey Twitter 推文缓存键（优化：使用字符串构建器）
func TwitterPostsCacheKey(c *gin.Context) string {
	username := c.Query("username")
	keyword := c.Query("keyword")
	page := c.Query("page")
	pageSize := c.Query("page_size")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	
	// 优化：使用 strings.Builder 构建键
	var keyBuilder strings.Builder
	keyBuilder.Grow(150)
	keyBuilder.WriteString("twitter:posts:")
	keyBuilder.WriteString(username)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(keyword)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(page)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(pageSize)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(startDate)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(endDate)
	
	key := keyBuilder.String()
	hash := md5.Sum([]byte(key))
	return BuildCacheKeyWithHash("cache:v1:twitter", fmt.Sprintf("%x", hash))
}

// PortfolioCacheKey 投资组合缓存键（优化：使用字符串构建器）
func PortfolioCacheKey(c *gin.Context) string {
	entity := c.Query("entity")
	return BuildCacheKey("cache:v1:portfolio:latest", entity)
}

// FlowsCacheKey 资金流缓存键（优化：使用字符串构建器）
func FlowsCacheKey(c *gin.Context) string {
	entity := c.Query("entity")
	coin := c.Query("coin")
	start := c.Query("start")
	end := c.Query("end")
	latest := c.Query("latest")
	
	// 优化：使用 strings.Builder 构建键
	var keyBuilder strings.Builder
	keyBuilder.Grow(100)
	keyBuilder.WriteString("flows:daily:")
	keyBuilder.WriteString(entity)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(coin)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(start)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(end)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(latest)
	
	key := keyBuilder.String()
	hash := md5.Sum([]byte(key))
	return BuildCacheKeyWithHash("cache:v1:flows", fmt.Sprintf("%x", hash))
}

// ==================== 缓存失效工具 ====================

// InvalidateAnnouncementsCache 失效公告相关缓存（优化：使用新的键命名规范）
func (s *Server) InvalidateAnnouncementsCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	// 如果缓存支持模式删除，使用模式删除
	if redisCache, ok := s.cache.(*pdb.RedisCache); ok {
		return redisCache.DeletePattern(ctx, "cache:v1:announcements:*")
	}
	// 否则只删除特定键（这里简化处理）
	return nil
}

// InvalidateMarketCache 失效市场数据缓存（优化：使用新的键命名规范）
func (s *Server) InvalidateMarketCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	if redisCache, ok := s.cache.(*pdb.RedisCache); ok {
		return redisCache.DeletePattern(ctx, "cache:v1:market:*")
	}
	return nil
}

// InvalidatePortfolioCache 失效投资组合缓存（优化：使用字符串构建器）
func (s *Server) InvalidatePortfolioCache(ctx context.Context, entity string) error {
	if s.cache == nil {
		return nil
	}
	key := BuildCacheKey("cache:v1:portfolio:latest", entity)
	return s.cache.Delete(ctx, key)
}

// InvalidateFlowsCache 失效资金流缓存（优化：使用字符串构建器）
func (s *Server) InvalidateFlowsCache(ctx context.Context, entity string) error {
	if s.cache == nil {
		return nil
	}
	if redisCache, ok := s.cache.(*pdb.RedisCache); ok {
		// 使用更精确的模式匹配
		var patternBuilder strings.Builder
		patternBuilder.Grow(50)
		patternBuilder.WriteString("cache:v1:flows:*:")
		patternBuilder.WriteString(entity)
		patternBuilder.WriteString(":*")
		pattern := patternBuilder.String()
		return redisCache.DeletePattern(ctx, pattern)
	}
	return nil
}

// InvalidateTwitterCache 失效 Twitter 相关缓存（优化：使用新的键命名规范）
func (s *Server) InvalidateTwitterCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	if redisCache, ok := s.cache.(*pdb.RedisCache); ok {
		return redisCache.DeletePattern(ctx, "cache:v1:twitter:*")
	}
	return nil
}

// ==================== 黑名单缓存 ====================

// getCachedBlacklistMap 获取缓存的黑名单映射（大写 Symbol -> bool）
func (s *Server) getCachedBlacklistMap(ctx context.Context, kind string) (map[string]bool, error) {
	if s.cache == nil {
		// 无缓存时直接查询数据库
		return s.loadBlacklistMapFromDB(kind)
	}

	key := BuildCacheKey("cache:v1:blacklist", kind)
	cached, err := s.cache.Get(ctx, key)
	if err == nil && len(cached) > 0 {
		// 尝试解析缓存
		var symbols []string
		if err := json.Unmarshal(cached, &symbols); err == nil {
			return buildBlacklistMap(symbols), nil
		}
	}

	// 缓存未命中或解析失败，从数据库加载
	blacklistMap, err := s.loadBlacklistMapFromDB(kind)
	if err != nil {
		return nil, err
	}

	// 优化：使用协程池异步写入缓存
	symbols := make([]string, 0, len(blacklistMap))
	for symbol := range blacklistMap {
		symbols = append(symbols, symbol)
	}
	data, err := json.Marshal(symbols)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal blacklist cache data (kind=%s): %v", kind, err)
	} else {
		cacheKey := key
		cacheData := make([]byte, len(data))
		copy(cacheData, data)
		
		if globalCachePool != nil {
			globalCachePool.Submit(func() {
				if err := s.cache.Set(context.Background(), cacheKey, cacheData, 5*time.Minute); err != nil {
					log.Printf("[ERROR] Failed to set blacklist cache (kind=%s, key=%s): %v", kind, cacheKey, err)
				}
			})
		} else {
			go func() {
				if err := s.cache.Set(context.Background(), cacheKey, cacheData, 5*time.Minute); err != nil {
					log.Printf("[ERROR] Failed to set blacklist cache (kind=%s, key=%s): %v", kind, cacheKey, err)
				}
			}()
		}
	}

	return blacklistMap, nil
}

// loadBlacklistMapFromDB 从数据库加载黑名单并构建映射
func (s *Server) loadBlacklistMapFromDB(kind string) (map[string]bool, error) {
	blacklist, err := s.db.GetBinanceBlacklist(kind)
	if err != nil {
		return nil, err
	}
	return buildBlacklistMap(blacklist), nil
}

// buildBlacklistMap 构建黑名单映射（统一转换为大写）
func buildBlacklistMap(symbols []string) map[string]bool {
	blacklistMap := make(map[string]bool, len(symbols))
	for _, symbol := range symbols {
		blacklistMap[strings.ToUpper(symbol)] = true
	}
	return blacklistMap
}

// InvalidateBlacklistCache 失效黑名单缓存
func (s *Server) InvalidateBlacklistCache(ctx context.Context, kind string) error {
	if s.cache == nil {
		return nil
	}
	key := BuildCacheKey("cache:v1:blacklist", kind)
	return s.cache.Delete(ctx, key)
}

