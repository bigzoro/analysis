package db

import (
	"time"

	"gorm.io/datatypes"
)

// UserBehavior 用户行为追踪表
type UserBehavior struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      *uint          `gorm:"index" json:"user_id,omitempty"`   // 可为空（未登录用户）
	SessionID   string         `gorm:"size:64;index" json:"session_id"`  // 会话ID
	ActionType  string         `gorm:"size:32;index" json:"action_type"` // 行为类型
	ActionValue string         `gorm:"size:256" json:"action_value"`     // 行为值（如币种符号、页面路径等）
	Page        string         `gorm:"size:128" json:"page"`             // 页面路径
	Metadata    datatypes.JSON `json:"metadata"`                         // 额外元数据（JSON）
	UserAgent   string         `gorm:"size:512" json:"user_agent"`       // 用户代理
	IPAddress   string         `gorm:"size:45" json:"ip_address"`        // IP地址
	DeviceInfo  datatypes.JSON `json:"device_info"`                      // 设备信息
	CreatedAt   time.Time      `json:"created_at"`
}

// UserPreference 用户偏好设置
type UserPreference struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	RiskTolerance    string         `gorm:"size:16;default:'medium'" json:"risk_tolerance"`     // low/medium/high
	InvestmentStyle  string         `gorm:"size:32;default:'balanced'" json:"investment_style"` // conservative/balanced/aggressive
	TimeHorizon      string         `gorm:"size:16;default:'medium'" json:"time_horizon"`       // short/medium/long
	PreferredSectors []string       `gorm:"type:json" json:"preferred_sectors"`                 // 偏好板块
	PreferredCoins   []string       `gorm:"type:json" json:"preferred_coins"`                   // 偏好币种
	FavoriteFactors  []string       `gorm:"type:json" json:"favorite_factors"`                  // 偏好因子
	CustomWeights    datatypes.JSON `json:"custom_weights"`                                     // 自定义权重
	UpdatedAt        time.Time      `json:"updated_at"`
}

// UserRecommendationFeedback 用户对推荐的反馈
type UserRecommendationFeedback struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           *uint     `gorm:"index" json:"user_id,omitempty"`
	SessionID        string    `gorm:"size:64;index" json:"session_id"`
	RecommendationID uint      `gorm:"index" json:"recommendation_id"` // 关联的推荐ID
	Symbol           string    `gorm:"size:32;index" json:"symbol"`
	BaseSymbol       string    `gorm:"size:16;index" json:"base_symbol"`
	Action           string    `gorm:"size:16;index" json:"action"`                               // view/click/save/follow/buy/sell/ignore
	Rating           *int      `gorm:"check:rating >= 1 AND rating <= 5" json:"rating,omitempty"` // 1-5星评分
	Reason           string    `gorm:"size:512" json:"reason"`                                    // 用户反馈理由
	ActualOutcome    string    `gorm:"size:16" json:"actual_outcome"`                             // positive/negative/neutral
	OutcomeValue     *float64  `gorm:"type:decimal(10,4)" json:"outcome_value"`                   // 实际收益率
	CreatedAt        time.Time `json:"created_at"`
}

// UserBehaviorAnalysis 用户行为分析结果
type UserBehaviorAnalysis struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          *uint          `gorm:"index" json:"user_id,omitempty"`
	SessionID       string         `gorm:"size:64;index" json:"session_id"`
	AnalysisType    string         `gorm:"size:32;index" json:"analysis_type"`        // engagement/preference/trading_pattern
	AnalysisResult  datatypes.JSON `json:"analysis_result"`                           // 分析结果（JSON）
	ConfidenceScore float64        `gorm:"type:decimal(5,4)" json:"confidence_score"` // 置信度 0-1
	Recommendation  datatypes.JSON `json:"recommendation"`                            // 基于分析的推荐
	ProcessedAt     time.Time      `gorm:"index" json:"processed_at"`
	CreatedAt       time.Time      `json:"created_at"`
}

// AlgorithmPerformance 算法表现追踪
type AlgorithmPerformance struct {
	ID                      uint           `gorm:"primaryKey" json:"id"`
	AlgorithmVersion        string         `gorm:"size:32;index" json:"algorithm_version"`            // 算法版本
	TestGroup               string         `gorm:"size:16;index" json:"test_group"`                   // A/B测试组
	TimeRange               string         `gorm:"size:32;index" json:"time_range"`                   // 时间范围
	SampleSize              int            `gorm:"index" json:"sample_size"`                          // 样本大小
	Metrics                 datatypes.JSON `json:"metrics"`                                           // 性能指标
	ImprovementRate         float64        `gorm:"type:decimal(5,4)" json:"improvement_rate"`         // 改进率
	StatisticalSignificance float64        `gorm:"type:decimal(5,4)" json:"statistical_significance"` // 统计显著性
	CreatedAt               time.Time      `json:"created_at"`
}

// TableName 指定表名
func (UserBehavior) TableName() string               { return "user_behaviors" }
func (UserPreference) TableName() string             { return "user_preferences" }
func (UserRecommendationFeedback) TableName() string { return "user_recommendation_feedback" }
func (UserBehaviorAnalysis) TableName() string       { return "user_behavior_analysis" }
func (AlgorithmPerformance) TableName() string       { return "algorithm_performance" }

// 用户行为类型常量
const (
	ActionTypePageView             = "page_view"
	ActionTypeRecommendationView   = "recommendation_view"
	ActionTypeRecommendationClick  = "recommendation_click"
	ActionTypeRecommendationSave   = "recommendation_save"
	ActionTypeRecommendationFollow = "recommendation_follow"
	ActionTypeBacktestRun          = "backtest_run"
	ActionTypeSimulationRun        = "simulation_run"
	ActionTypeDataQualityCheck     = "data_quality_check"
	ActionTypeSettingsChange       = "settings_change"
	ActionTypeSearch               = "search"
	ActionTypeFilter               = "filter"
	ActionTypeExport               = "export"
)

// 反馈动作类型常量
const (
	FeedbackActionView   = "view"
	FeedbackActionClick  = "click"
	FeedbackActionSave   = "save"
	FeedbackActionFollow = "follow"
	FeedbackActionBuy    = "buy"
	FeedbackActionSell   = "sell"
	FeedbackActionIgnore = "ignore"
)

// 分析类型常量
const (
	AnalysisTypeEngagement     = "engagement"
	AnalysisTypePreference     = "preference"
	AnalysisTypeTradingPattern = "trading_pattern"
	AnalysisTypeRiskProfile    = "risk_profile"
)
