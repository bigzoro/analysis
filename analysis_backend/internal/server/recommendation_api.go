package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"
	"github.com/gin-gonic/gin"
)

// GetDataQualityReport 获取数据质量报告
// GET /data-quality/report
func (s *Server) GetDataQualityReport(c *gin.Context) {
	if s.dataQualityMonitor == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "数据质量监控器未初始化",
		})
		return
	}

	report := s.dataQualityMonitor.GetHealthReport()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   report,
	})
}

// GetFallbackStatus 获取降级策略状态
// GET /fallback/status
func (s *Server) GetFallbackStatus(c *gin.Context) {
	if s.fallbackStrategy == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "error",
			"message": "降级策略管理器未初始化",
		})
		return
	}

	status := map[string]interface{}{
		"current_level":    s.fallbackStrategy.GetCurrentLevel(),
		"component_status": s.fallbackStrategy.GetComponentStatus(),
		"recommendation":   s.fallbackStrategy.GetFallbackRecommendation(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// GetCoinRecommendations 获取币种推荐
// GET /recommendations/coins?kind=spot&limit=5&refresh=false
func (s *Server) GetCoinRecommendations(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	if kind != "spot" && kind != "futures" {
		kind = "spot"
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 10 {
			limit = n
		}
	}

	refresh := c.Query("refresh") == "true"

	// 尝试从增强缓存获取（非刷新模式）
	if !refresh && s.recommendationCache != nil {
		// 构建查询参数
		params := RecommendationQueryParams{
			Kind:  kind,
			Limit: limit,
		}

		// 尝试获取缓存的推荐
		cached, err := s.recommendationCache.GetRecommendationsWithCache(c.Request.Context(), params)
		if err == nil && cached != nil {
			// 无论是否有数据，都使用缓存结果（如果缓存返回了结果，说明至少尝试了获取）
			log.Printf("[INFO] Cache query completed: %s, limit=%d, results=%d", kind, limit, len(cached))

			if len(cached) > 0 {
				// 有缓存数据，直接返回
				formattedRecs := formatRecommendations(cached, s, c.Request.Context())

				c.JSON(http.StatusOK, gin.H{
					"generated_at":    time.Now().UTC(),
					"kind":            kind,
					"recommendations": formattedRecs,
					"cached":          true,
					"cache_type":      "enhanced",
				})
				return
			} else {
				// 缓存查询成功但返回空结果，说明后台服务没有准备好数据
				log.Printf("[WARNING] Cache returned empty results, background services may not be running")
			}
		} else if err != nil {
			log.Printf("[WARNING] Cache query failed: %v", err)
		}
	}

	// 注意：没有找到推荐数据
	// 可能是后台服务没有运行，或者缓存还没有准备好
	log.Printf("[WARNING] No recommendation data available, background services may not be running")

	// 返回"数据准备中"的响应，而不是错误
	c.JSON(http.StatusOK, gin.H{
		"generated_at":    time.Now().UTC(),
		"kind":            kind,
		"recommendations": []gin.H{}, // 返回空数组
		"cached":          false,
		"cache_status":    "empty",
		"message":         "推荐数据正在准备中，请稍后刷新或联系管理员检查后台服务",
		"recommendation":  "确保 recommendation_scanner 和 investment 服务正在运行",
	})
}

// GetHistoricalRecommendations 获取历史推荐（根据时间）
// GET /recommendations/historical?kind=spot&date=2024-01-01&limit=5
func (s *Server) GetHistoricalRecommendations(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	if kind != "spot" && kind != "futures" {
		kind = "spot"
	}

	dateStr := c.DefaultQuery("date", "")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少日期参数",
		})
		return
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "日期格式错误，应为 YYYY-MM-DD",
		})
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 10 {
			limit = n
		}
	}

	ctx := c.Request.Context()
	recommendations, err := s.generateRecommendationsForDate(ctx, kind, limit, targetDate)
	if err != nil {
		log.Printf("[ERROR] 生成历史推荐失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成历史推荐失败",
		})
		return
	}

	formattedRecs := formatRecommendations(recommendations, s, ctx)

	c.JSON(http.StatusOK, gin.H{
		"generated_at":    time.Now().UTC(),
		"target_date":     targetDate.Format("2006-01-02"),
		"kind":            kind,
		"recommendations": formattedRecs,
		"cached":          false,
	})
}

// GenerateRecommendationsForDate 为指定日期生成推荐
// POST /recommendations/generate?date=2024-01-01&kind=spot
func (s *Server) GenerateRecommendationsForDate(c *gin.Context) {
	dateStr := c.DefaultQuery("date", "")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少日期参数",
		})
		return
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "日期格式错误，应为 YYYY-MM-DD",
		})
		return
	}

	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	if kind != "spot" && kind != "futures" {
		kind = "spot"
	}

	ctx := c.Request.Context()
	recommendations, err := s.generateRecommendationsForDate(ctx, kind, 10, targetDate)
	if err != nil {
		log.Printf("[ERROR] 生成推荐失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成推荐失败",
		})
		return
	}

	// 保存到数据库
	dbRecs := recommendations
	if err != nil {
		log.Printf("[ERROR] 保存推荐到数据库失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存推荐失败",
		})
		return
	}

	formattedRecs := formatRecommendations(dbRecs, s, ctx)

	c.JSON(http.StatusOK, gin.H{
		"generated_at":    time.Now().UTC(),
		"target_date":     targetDate.Format("2006-01-02"),
		"kind":            kind,
		"recommendations": formattedRecs,
		"saved_count":     len(dbRecs),
	})
}

// GetRecommendationTimeList 获取推荐时间列表
// GET /recommendations/times?kind=spot&days=7
func (s *Server) GetRecommendationTimeList(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	if kind != "spot" && kind != "futures" {
		kind = "spot"
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if n, err := strconv.Atoi(daysStr); err == nil && n > 0 && n <= 30 {
			days = n
		}
	}

	// 查询数据库中指定时间范围内的推荐时间戳
	var timestamps []time.Time
	query := s.db.DB().Model(&pdb.CoinRecommendation{}).Where("kind = ?", kind)

	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -days)
	query = query.Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	query = query.Select("DISTINCT DATE(created_at)").Order("created_at DESC")

	rows, err := query.Rows()
	if err != nil {
		log.Printf("[ERROR] 查询推荐时间失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询失败",
		})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err == nil {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				timestamps = append(timestamps, t)
			}
		}
	}

	// 格式化为日期字符串
	dateStrings := make([]string, len(timestamps))
	for i, t := range timestamps {
		dateStrings[i] = t.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"kind":       kind,
		"days":       days,
		"dates":      dateStrings,
		"count":      len(dateStrings),
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}
