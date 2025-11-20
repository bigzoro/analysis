package db

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CoinRecommendation 币种推荐结果
type CoinRecommendation struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	GeneratedAt time.Time   `gorm:"index:idx_generated_at" json:"generated_at"`
	Kind      string         `gorm:"size:16;index:idx_generated_at" json:"kind"` // spot/futures
	Symbol    string         `gorm:"size:32;index" json:"symbol"`               // BTCUSDT
	BaseSymbol string        `gorm:"size:16" json:"base_symbol"`                  // BTC
	Rank      int            `gorm:"index" json:"rank"`                         // 1-5
	TotalScore float64       `gorm:"type:decimal(5,2)" json:"total_score"`      // 总分 0-100

	// 各因子得分
	MarketScore   float64 `gorm:"type:decimal(5,2)" json:"market_score"`   // 市场表现得分
	FlowScore     float64 `gorm:"type:decimal(5,2)" json:"flow_score"`     // 资金流得分
	HeatScore     float64 `gorm:"type:decimal(5,2)" json:"heat_score"`     // 市场热度得分
	EventScore    float64 `gorm:"type:decimal(5,2)" json:"event_score"`    // 事件得分
	SentimentScore float64 `gorm:"type:decimal(5,2)" json:"sentiment_score"` // 情绪得分

	// 原始数据快照
	PriceChange24h *float64 `gorm:"type:decimal(10,4)" json:"price_change_24h"`
	Volume24h      *float64 `gorm:"type:decimal(20,8)" json:"volume_24h"`
	MarketCapUSD   *float64 `gorm:"type:decimal(20,2)" json:"market_cap_usd"`
	NetFlow24h     *float64 `gorm:"type:decimal(20,8)" json:"net_flow_24h"`
	HasNewListing  bool     `json:"has_new_listing"`
	HasAnnouncement bool    `json:"has_announcement"`
	TwitterMentions *int    `json:"twitter_mentions"`

	// 推荐理由（JSON格式）
	Reasons datatypes.JSON `json:"reasons"`

	// 风险评级字段
	VolatilityRisk  *float64      `gorm:"type:decimal(5,2)" json:"volatility_risk"`  // 波动率风险 0-100
	LiquidityRisk   *float64      `gorm:"type:decimal(5,2)" json:"liquidity_risk"`   // 流动性风险 0-100
	MarketRisk      *float64      `gorm:"type:decimal(5,2)" json:"market_risk"`      // 市场风险 0-100
	TechnicalRisk   *float64      `gorm:"type:decimal(5,2)" json:"technical_risk"`  // 技术风险 0-100
	OverallRisk     *float64      `gorm:"type:decimal(5,2)" json:"overall_risk"`    // 综合风险 0-100
	RiskLevel       *string        `gorm:"size:16" json:"risk_level"`                // low/medium/high
	RiskWarnings    datatypes.JSON `json:"risk_warnings"`                            // 风险提示（JSON数组）

	// 技术指标
	TechnicalIndicators datatypes.JSON `json:"technical_indicators"` // 技术指标（JSON格式）

	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (CoinRecommendation) TableName() string {
	return "coin_recommendations"
}

// SaveRecommendations 保存推荐结果（会先删除同时间的旧数据）
func SaveRecommendations(gdb *gorm.DB, kind string, generatedAt time.Time, recommendations []CoinRecommendation) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		// 删除同时间的旧推荐
		if err := tx.Where("kind = ? AND generated_at = ?", kind, generatedAt).
			Delete(&CoinRecommendation{}).Error; err != nil {
			return err
		}

		// 插入新推荐
		if len(recommendations) > 0 {
			if err := tx.Create(&recommendations).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetLatestRecommendations 获取最新的推荐结果
func GetLatestRecommendations(gdb *gorm.DB, kind string, limit int) ([]CoinRecommendation, error) {
	var recommendations []CoinRecommendation
	q := gdb.Where("kind = ?", kind).
		Order("generated_at DESC, rank ASC").
		Limit(limit)
	
	// 只取最新一批的推荐
	var latestTime time.Time
	if err := gdb.Model(&CoinRecommendation{}).
		Where("kind = ?", kind).
		Select("MAX(generated_at)").
		Scan(&latestTime).Error; err != nil {
		return nil, err
	}
	
	if !latestTime.IsZero() {
		q = q.Where("generated_at = ?", latestTime)
	}
	
	if err := q.Find(&recommendations).Error; err != nil {
		return nil, err
	}
	return recommendations, nil
}

// GetRecommendationBySymbol 根据币种获取最新推荐
func GetRecommendationBySymbol(gdb *gorm.DB, symbol string) (*CoinRecommendation, error) {
	var rec CoinRecommendation
	err := gdb.Where("symbol = ?", symbol).
		Order("generated_at DESC").
		First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

