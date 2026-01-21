package server

import (
	"encoding/json"
	"log"
	"math"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RecommendationFeedbackService 推荐反馈服务
type RecommendationFeedbackService struct {
	db *gorm.DB
}

// NewRecommendationFeedbackService 创建推荐反馈服务
func NewRecommendationFeedbackService(db *gorm.DB) *RecommendationFeedbackService {
	return &RecommendationFeedbackService{db: db}
}

// SubmitFeedbackRequest 提交反馈请求
type SubmitFeedbackRequest struct {
	RecommendationID uint   `json:"recommendation_id" binding:"required"`
	Symbol           string `json:"symbol" binding:"required"`
	BaseSymbol       string `json:"base_symbol" binding:"required"`
	Action           string `json:"action" binding:"required,oneof=view click save follow buy sell ignore"`
	Rating           *int   `json:"rating,omitempty"`
	Reason           string `json:"reason,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
}

// SubmitFeedback 提交推荐反馈
func (rfs *RecommendationFeedbackService) SubmitFeedback(c *gin.Context) {
	var req SubmitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求数据"})
		return
	}

	// 获取用户ID
	userID := rfs.getUserID(c)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = c.GetHeader("X-Session-ID")
	}

	// 创建反馈记录
	feedback := pdb.UserRecommendationFeedback{
		UserID:           userID,
		SessionID:        sessionID,
		RecommendationID: req.RecommendationID,
		Symbol:           req.Symbol,
		BaseSymbol:       req.BaseSymbol,
		Action:           req.Action,
		Rating:           req.Rating,
		Reason:           req.Reason,
		CreatedAt:        time.Now(),
	}

	// 保存反馈
	if err := rfs.db.Create(&feedback).Error; err != nil {
		log.Printf("保存推荐反馈失败: %v", err)
		c.JSON(500, gin.H{"error": "保存反馈失败"})
		return
	}

	// 异步更新推荐统计
	go rfs.updateRecommendationStats(req.RecommendationID, req.Action, req.Rating)

	c.JSON(200, gin.H{
		"success":     true,
		"feedback_id": feedback.ID,
	})
}

// GetRecommendationStats 获取推荐统计信息
func (rfs *RecommendationFeedbackService) GetRecommendationStats(c *gin.Context) {
	recommendationID := c.Query("recommendation_id")
	if recommendationID == "" {
		c.JSON(400, gin.H{"error": "缺少recommendation_id参数"})
		return
	}

	var stats struct {
		Impressions   int     `json:"impressions"`
		Clicks        int     `json:"clicks"`
		Saves         int     `json:"saves"`
		Follows       int     `json:"follows"`
		FeedbackCount int     `json:"feedback_count"`
		AvgRating     float64 `json:"avg_rating"`
		ClickRate     float64 `json:"click_rate"`
		SaveRate      float64 `json:"save_rate"`
		FollowRate    float64 `json:"follow_rate"`
	}

	// 查询统计数据
	query := rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("recommendation_id = ?", recommendationID)

	// 计算各种统计
	var totalFeedbacks int64
	query.Count(&totalFeedbacks)
	stats.FeedbackCount = int(totalFeedbacks)

	// 计算各项指标
	actionStats := make(map[string]int64)
	rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("recommendation_id = ?", recommendationID).
		Select("action, COUNT(*) as count").
		Group("action").
		Pluck("count", &actionStats)

	stats.Clicks = int(actionStats["click"])
	stats.Saves = int(actionStats["save"])
	stats.Follows = int(actionStats["follow"])

	// 计算评分统计
	var avgRating struct {
		Avg float64
	}
	rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("recommendation_id = ? AND rating IS NOT NULL", recommendationID).
		Select("COALESCE(AVG(rating), 0) as avg").
		Scan(&avgRating)
	stats.AvgRating = math.Round(avgRating.Avg*100) / 100

	// 计算转化率（需要从推荐表获取曝光数）
	var rec pdb.CoinRecommendation
	if err := rfs.db.First(&rec, recommendationID).Error; err == nil {
		stats.Impressions = rec.Impressions
		if stats.Impressions > 0 {
			stats.ClickRate = float64(stats.Clicks) / float64(stats.Impressions)
			stats.SaveRate = float64(stats.Saves) / float64(stats.Impressions)
			stats.FollowRate = float64(stats.Follows) / float64(stats.Impressions)
		}
	}

	c.JSON(200, stats)
}

// updateRecommendationStats 更新推荐统计信息
func (rfs *RecommendationFeedbackService) updateRecommendationStats(recommendationID uint, action string, rating *int) {
	// 使用事务更新统计
	rfs.db.Transaction(func(tx *gorm.DB) error {
		// 获取当前统计
		var rec pdb.CoinRecommendation
		if err := tx.First(&rec, recommendationID).Error; err != nil {
			return err
		}

		// 更新相应计数
		switch action {
		case "click":
			rec.Clicks++
		case "save":
			rec.Saves++
		case "follow":
			rec.Follows++
		}

		// 更新平均评分
		if rating != nil && *rating >= 1 && *rating <= 5 {
			currentTotal := rec.AvgRating * float64(rec.FeedbackCount)
			rec.FeedbackCount++
			rec.AvgRating = (currentTotal + float64(*rating)) / float64(rec.FeedbackCount)
		} else if action != "view" {
			rec.FeedbackCount++
		}

		// 保存更新
		return tx.Save(&rec).Error
	})
}

// GetUserFeedbackHistory 获取用户反馈历史
func (rfs *RecommendationFeedbackService) GetUserFeedbackHistory(c *gin.Context) {
	userID := rfs.getUserID(c)
	if userID == nil {
		c.JSON(401, gin.H{"error": "用户未登录"})
		return
	}

	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if n, err := json.Number(p).Int64(); err == nil && n > 0 {
			page = int(n)
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := json.Number(l).Int64(); err == nil && n > 0 && n <= 100 {
			limit = int(n)
		}
	}

	offset := (page - 1) * limit

	var feedbacks []pdb.UserRecommendationFeedback
	var total int64

	// 查询总数
	rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("user_id = ?", *userID).
		Count(&total)

	// 查询数据
	if err := rfs.db.Where("user_id = ?", *userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&feedbacks).Error; err != nil {
		c.JSON(500, gin.H{"error": "查询反馈历史失败"})
		return
	}

	// 格式化响应
	response := gin.H{
		"total": total,
		"page":  page,
		"limit": limit,
		"data":  feedbacks,
	}

	c.JSON(200, response)
}

// GetFeedbackAnalytics 获取反馈分析数据
func (rfs *RecommendationFeedbackService) GetFeedbackAnalytics(c *gin.Context) {
	// 时间范围参数
	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := json.Number(d).Int64(); err == nil && n > 0 && n <= 365 {
			days = int(n)
		}
	}

	startDate := time.Now().AddDate(0, 0, -days)

	var analytics struct {
		TotalFeedbacks     int64                    `json:"total_feedbacks"`
		ActionDistribution map[string]int64         `json:"action_distribution"`
		AvgRatings         map[string]float64       `json:"avg_ratings"`
		DailyStats         []map[string]interface{} `json:"daily_stats"`
		TopCoins           []map[string]interface{} `json:"top_coins"`
	}

	// 总反馈数
	rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("created_at >= ?", startDate).
		Count(&analytics.TotalFeedbacks)

	// 行为分布
	analytics.ActionDistribution = make(map[string]int64)
	rows, _ := rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("created_at >= ?", startDate).
		Select("action, COUNT(*) as count").
		Group("action").
		Rows()

	for rows.Next() {
		var action string
		var count int64
		rows.Scan(&action, &count)
		analytics.ActionDistribution[action] = count
	}
	rows.Close()

	// 平均评分（按行为类型）
	analytics.AvgRatings = make(map[string]float64)
	ratingRows, _ := rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("created_at >= ? AND rating IS NOT NULL", startDate).
		Select("action, AVG(rating) as avg_rating").
		Group("action").
		Rows()

	for ratingRows.Next() {
		var action string
		var avgRating float64
		ratingRows.Scan(&action, &avgRating)
		analytics.AvgRatings[action] = math.Round(avgRating*100) / 100
	}
	ratingRows.Close()

	// 热门币种
	analytics.TopCoins = make([]map[string]interface{}, 0)
	coinRows, _ := rfs.db.Model(&pdb.UserRecommendationFeedback{}).
		Where("created_at >= ?", startDate).
		Select("symbol, COUNT(*) as feedback_count, AVG(CASE WHEN rating IS NOT NULL THEN rating ELSE NULL END) as avg_rating").
		Group("symbol").
		Order("feedback_count DESC").
		Limit(10).
		Rows()

	for coinRows.Next() {
		var symbol string
		var feedbackCount int64
		var avgRating float64
		coinRows.Scan(&symbol, &feedbackCount, &avgRating)

		coin := map[string]interface{}{
			"symbol":         symbol,
			"feedback_count": feedbackCount,
			"avg_rating":     math.Round(avgRating*100) / 100,
		}
		analytics.TopCoins = append(analytics.TopCoins, coin)
	}
	coinRows.Close()

	c.JSON(200, analytics)
}

// getUserID 从上下文中获取用户ID
func (rfs *RecommendationFeedbackService) getUserID(c *gin.Context) *uint {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		return nil
	}

	if id, ok := userID.(uint); ok {
		return &id
	}

	if id, ok := userID.(int); ok {
		uid := uint(id)
		return &uid
	}

	return nil
}
