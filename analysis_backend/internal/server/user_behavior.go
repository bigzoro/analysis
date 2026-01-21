package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserBehaviorService 用户行为服务
type UserBehaviorService struct {
	db *gorm.DB
}

// NewUserBehaviorService 创建用户行为服务
func NewUserBehaviorService(db *gorm.DB) *UserBehaviorService {
	return &UserBehaviorService{db: db}
}

// TrackBehaviorRequest 行为追踪请求
type TrackBehaviorRequest struct {
	Events []BehaviorEvent `json:"events" binding:"required"`
}

// BehaviorEvent 行为事件
type BehaviorEvent struct {
	SessionID   string                 `json:"session_id" binding:"required"`
	UserID      *uint                  `json:"user_id,omitempty"`
	ActionType  string                 `json:"action_type" binding:"required"`
	ActionValue string                 `json:"action_value"`
	Page        string                 `json:"page"`
	Metadata    map[string]interface{} `json:"metadata"`
	UserAgent   string                 `json:"user_agent"`
	IPAddress   string                 `json:"ip_address"`
	DeviceInfo  map[string]interface{} `json:"device_info"`
}

// TrackUserBehavior 追踪用户行为
func (ubs *UserBehaviorService) TrackUserBehavior(c *gin.Context) {
	var req TrackBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求数据"})
		return
	}

	// 获取客户端IP
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = c.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = c.GetHeader("X-Real-IP")
	}

	// 批量保存行为数据
	behaviors := make([]pdb.UserBehavior, 0, len(req.Events))
	for _, event := range req.Events {
		// 序列化元数据
		metadataJSON, _ := json.Marshal(event.Metadata)
		deviceInfoJSON, _ := json.Marshal(event.DeviceInfo)

		behavior := pdb.UserBehavior{
			UserID:      event.UserID,
			SessionID:   event.SessionID,
			ActionType:  event.ActionType,
			ActionValue: event.ActionValue,
			Page:        event.Page,
			Metadata:    metadataJSON,
			UserAgent:   event.UserAgent,
			IPAddress:   clientIP,
			DeviceInfo:  deviceInfoJSON,
			CreatedAt:   time.Now(),
		}
		behaviors = append(behaviors, behavior)
	}

	// 批量插入
	if len(behaviors) > 0 {
		if err := ubs.db.CreateInBatches(behaviors, 100).Error; err != nil {
			log.Printf("保存用户行为数据失败: %v", err)
			c.JSON(500, gin.H{"error": "保存行为数据失败"})
			return
		}
	}

	// 异步处理行为分析（不阻塞响应）
	go ubs.processBehaviorAnalysis(behaviors)

	c.JSON(200, gin.H{"success": true, "processed": len(behaviors)})
}

// processBehaviorAnalysis 异步处理行为分析
func (ubs *UserBehaviorService) processBehaviorAnalysis(behaviors []pdb.UserBehavior) {
	ctx := context.Background()

	// 按用户/会话分组处理
	userGroups := make(map[string][]pdb.UserBehavior)
	sessionGroups := make(map[string][]pdb.UserBehavior)

	for _, behavior := range behaviors {
		if behavior.UserID != nil {
			userKey := fmt.Sprintf("user_%d", *behavior.UserID)
			userGroups[userKey] = append(userGroups[userKey], behavior)
		}

		sessionKey := fmt.Sprintf("session_%s", behavior.SessionID)
		sessionGroups[sessionKey] = append(sessionGroups[sessionKey], behavior)
	}

	// 处理用户行为分析
	for userKey, userBehaviors := range userGroups {
		if len(userBehaviors) >= 5 { // 至少5个行为才进行分析
			go ubs.analyzeUserBehavior(ctx, userKey, userBehaviors)
		}
	}

	// 处理会话行为分析
	for sessionKey, sessionBehaviors := range sessionGroups {
		if len(sessionBehaviors) >= 3 { // 至少3个行为才进行分析
			go ubs.analyzeSessionBehavior(ctx, sessionKey, sessionBehaviors)
		}
	}
}

// analyzeUserBehavior 分析用户行为模式
func (ubs *UserBehaviorService) analyzeUserBehavior(ctx context.Context, userKey string, behaviors []pdb.UserBehavior) {
	userID := behaviors[0].UserID
	if userID == nil {
		return
	}

	// 计算用户参与度
	engagement := ubs.calculateUserEngagement(behaviors)

	// 分析偏好
	preferences := ubs.analyzeUserPreferences(behaviors)

	// 分析交易模式
	tradingPattern := ubs.analyzeTradingPattern(behaviors)

	// 生成分析结果
	analysisResult := map[string]interface{}{
		"engagement_score":  engagement.Score,
		"engagement_level":  engagement.Level,
		"preferred_coins":   preferences.PreferredCoins,
		"preferred_factors": preferences.PreferredFactors,
		"risk_tolerance":    tradingPattern.RiskTolerance,
		"investment_style":  tradingPattern.InvestmentStyle,
		"time_horizon":      tradingPattern.TimeHorizon,
		"activity_pattern":  tradingPattern.ActivityPattern,
	}

	// 序列化结果
	resultJSON, _ := json.Marshal(analysisResult)

	// 保存分析结果
	analysis := pdb.UserBehaviorAnalysis{
		UserID:          userID,
		AnalysisType:    pdb.AnalysisTypeEngagement,
		AnalysisResult:  resultJSON,
		ConfidenceScore: engagement.Score,
		ProcessedAt:     time.Now(),
		CreatedAt:       time.Now(),
	}

	if err := ubs.db.Create(&analysis).Error; err != nil {
		log.Printf("保存用户行为分析失败: %v", err)
		return
	}

	// 更新用户偏好设置
	ubs.updateUserPreferences(*userID, preferences, tradingPattern)
}

// calculateUserEngagement 计算用户参与度
func (ubs *UserBehaviorService) calculateUserEngagement(behaviors []pdb.UserBehavior) UserEngagement {
	score := 0.0
	actionWeights := map[string]float64{
		pdb.ActionTypePageView:            0.1,
		pdb.ActionTypeRecommendationView:  0.2,
		pdb.ActionTypeRecommendationClick: 0.5,
		pdb.ActionTypeRecommendationSave:  0.8,
		pdb.ActionTypeBacktestRun:         1.0,
		pdb.ActionTypeSimulationRun:       1.0,
	}

	for _, behavior := range behaviors {
		if weight, exists := actionWeights[behavior.ActionType]; exists {
			score += weight
		}
	}

	// 归一化到0-1
	score = math.Min(score/float64(len(behaviors)), 1.0)

	level := "low"
	if score >= 0.7 {
		level = "high"
	} else if score >= 0.4 {
		level = "medium"
	}

	return UserEngagement{
		Score: score,
		Level: level,
	}
}

// analyzeUserPreferences 分析用户偏好
func (ubs *UserBehaviorService) analyzeUserPreferences(behaviors []pdb.UserBehavior) UserPreferences {
	coinCount := make(map[string]int)
	factorInteractions := make(map[string]int)

	for _, behavior := range behaviors {
		// 分析币种偏好
		if behavior.ActionType == pdb.ActionTypeRecommendationClick ||
			behavior.ActionType == pdb.ActionTypeRecommendationSave ||
			behavior.ActionType == pdb.ActionTypeRecommendationFollow {
			coinCount[behavior.ActionValue]++
		}

		// 分析因子偏好（从元数据中提取）
		var metadata map[string]interface{}
		if err := json.Unmarshal(behavior.Metadata, &metadata); err == nil {
			if factor, exists := metadata["factor"]; exists {
				factorInteractions[fmt.Sprintf("%v", factor)]++
			}
		}
	}

	// 获取Top偏好币种
	type coinFreq struct {
		Coin  string
		Count int
	}
	var coins []coinFreq
	for coin, count := range coinCount {
		coins = append(coins, coinFreq{coin, count})
	}
	sort.Slice(coins, func(i, j int) bool {
		return coins[i].Count > coins[j].Count
	})

	preferredCoins := make([]string, 0, 5)
	for i, coin := range coins {
		if i >= 5 {
			break
		}
		preferredCoins = append(preferredCoins, coin.Coin)
	}

	// 获取Top偏好因子
	var factors []string
	for factor := range factorInteractions {
		factors = append(factors, factor)
	}

	return UserPreferences{
		PreferredCoins:   preferredCoins,
		PreferredFactors: factors,
	}
}

// analyzeTradingPattern 分析交易模式
func (ubs *UserBehaviorService) analyzeTradingPattern(behaviors []pdb.UserBehavior) TradingPattern {
	riskTolerance := "medium"
	investmentStyle := "balanced"
	timeHorizon := "medium"
	activityPattern := "regular"

	backtestCount := 0
	simulationCount := 0
	highRiskActions := 0

	for _, behavior := range behaviors {
		switch behavior.ActionType {
		case pdb.ActionTypeBacktestRun:
			backtestCount++
		case pdb.ActionTypeSimulationRun:
			simulationCount++
		}

		// 分析风险偏好
		if behavior.ActionType == pdb.ActionTypeRecommendationClick {
			// 从元数据中检查风险等级
			var metadata map[string]interface{}
			if err := json.Unmarshal(behavior.Metadata, &metadata); err == nil {
				if riskLevel, exists := metadata["risk_level"]; exists {
					if fmt.Sprintf("%v", riskLevel) == "high" {
						highRiskActions++
					}
				}
			}
		}
	}

	// 根据行为判断风险偏好
	totalRiskActions := backtestCount + simulationCount + highRiskActions
	if totalRiskActions > len(behaviors)/2 {
		riskTolerance = "high"
		investmentStyle = "aggressive"
	} else if totalRiskActions < len(behaviors)/4 {
		riskTolerance = "low"
		investmentStyle = "conservative"
	}

	// 根据活跃度判断时间视野
	if backtestCount > simulationCount {
		timeHorizon = "long" // 更关注历史回测
	}

	return TradingPattern{
		RiskTolerance:   riskTolerance,
		InvestmentStyle: investmentStyle,
		TimeHorizon:     timeHorizon,
		ActivityPattern: activityPattern,
	}
}

// updateUserPreferences 更新用户偏好设置
func (ubs *UserBehaviorService) updateUserPreferences(userID uint, preferences UserPreferences, pattern TradingPattern) {
	preference := pdb.UserPreference{
		UserID:          userID,
		RiskTolerance:   pattern.RiskTolerance,
		InvestmentStyle: pattern.InvestmentStyle,
		TimeHorizon:     pattern.TimeHorizon,
		PreferredCoins:  preferences.PreferredCoins,
		FavoriteFactors: preferences.PreferredFactors,
		UpdatedAt:       time.Now(),
	}

	// 使用Upsert更新
	ubs.db.Where(pdb.UserPreference{UserID: userID}).
		Assign(preference).
		FirstOrCreate(&preference)
}

// analyzeSessionBehavior 分析会话行为
func (ubs *UserBehaviorService) analyzeSessionBehavior(ctx context.Context, sessionKey string, behaviors []pdb.UserBehavior) {
	sessionID := behaviors[0].SessionID

	// 计算会话质量
	sessionQuality := ubs.calculateSessionQuality(behaviors)

	// 保存会话分析结果
	analysisResult := map[string]interface{}{
		"duration":                    sessionQuality.Duration,
		"page_views":                  sessionQuality.PageViews,
		"actions_count":               sessionQuality.ActionsCount,
		"recommendation_interactions": sessionQuality.RecommendationInteractions,
		"bounce_rate":                 sessionQuality.BounceRate,
		"engagement_score":            sessionQuality.EngagementScore,
	}

	resultJSON, _ := json.Marshal(analysisResult)

	analysis := pdb.UserBehaviorAnalysis{
		SessionID:       sessionID,
		AnalysisType:    pdb.AnalysisTypeEngagement,
		AnalysisResult:  resultJSON,
		ConfidenceScore: sessionQuality.EngagementScore,
		ProcessedAt:     time.Now(),
		CreatedAt:       time.Now(),
	}

	ubs.db.Create(&analysis)
}

// calculateSessionQuality 计算会话质量
func (ubs *UserBehaviorService) calculateSessionQuality(behaviors []pdb.UserBehavior) SessionQuality {
	if len(behaviors) == 0 {
		return SessionQuality{}
	}

	sort.Slice(behaviors, func(i, j int) bool {
		return behaviors[i].CreatedAt.Before(behaviors[j].CreatedAt)
	})

	startTime := behaviors[0].CreatedAt
	endTime := behaviors[len(behaviors)-1].CreatedAt
	duration := endTime.Sub(startTime).Minutes()

	pageViews := 0
	recommendationInteractions := 0
	uniquePages := make(map[string]bool)

	for _, behavior := range behaviors {
		if behavior.ActionType == pdb.ActionTypePageView {
			pageViews++
			uniquePages[behavior.Page] = true
		}

		if behavior.ActionType == pdb.ActionTypeRecommendationClick ||
			behavior.ActionType == pdb.ActionTypeRecommendationSave ||
			behavior.ActionType == pdb.ActionTypeRecommendationFollow {
			recommendationInteractions++
		}
	}

	bounceRate := 0.0
	if len(uniquePages) == 1 {
		bounceRate = 1.0
	}

	// 计算参与度得分
	engagementScore := 0.0
	engagementScore += float64(pageViews) * 0.1
	engagementScore += float64(recommendationInteractions) * 0.3
	engagementScore += (1 - bounceRate) * 0.2
	if duration > 5 {
		engagementScore += 0.4
	}
	engagementScore = math.Min(engagementScore, 1.0)

	return SessionQuality{
		Duration:                   duration,
		PageViews:                  pageViews,
		ActionsCount:               len(behaviors),
		RecommendationInteractions: recommendationInteractions,
		BounceRate:                 bounceRate,
		EngagementScore:            engagementScore,
	}
}

// 数据结构定义
type UserEngagement struct {
	Score float64 `json:"score"`
	Level string  `json:"level"`
}

type UserPreferences struct {
	PreferredCoins   []string `json:"preferred_coins"`
	PreferredFactors []string `json:"preferred_factors"`
}

type TradingPattern struct {
	RiskTolerance   string `json:"risk_tolerance"`
	InvestmentStyle string `json:"investment_style"`
	TimeHorizon     string `json:"time_horizon"`
	ActivityPattern string `json:"activity_pattern"`
}

type SessionQuality struct {
	Duration                   float64 `json:"duration"`
	PageViews                  int     `json:"page_views"`
	ActionsCount               int     `json:"actions_count"`
	RecommendationInteractions int     `json:"recommendation_interactions"`
	BounceRate                 float64 `json:"bounce_rate"`
	EngagementScore            float64 `json:"engagement_score"`
}
