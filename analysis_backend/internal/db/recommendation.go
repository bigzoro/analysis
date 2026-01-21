package db

import (
	"log"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CoinRecommendation 币种推荐结果
type CoinRecommendation struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	GeneratedAt  time.Time `gorm:"index:idx_generated_at" json:"generated_at"`
	Kind         string    `gorm:"size:16;index:idx_generated_at" json:"kind"`  // spot/futures
	Symbol       string    `gorm:"size:32;index" json:"symbol"`                 // BTCUSDT
	BaseSymbol   string    `gorm:"size:16" json:"base_symbol"`                  // BTC
	Rank         int       `gorm:"index" json:"rank"`                           // 1-5
	TotalScore   float64   `gorm:"type:decimal(5,2)" json:"total_score"`        // 总分 0-100
	StrategyType string    `gorm:"size:16;default:'LONG'" json:"strategy_type"` // 策略类型: LONG/SHORT/RANGE

	// 各因子得分
	MarketScore    float64 `gorm:"type:decimal(5,2)" json:"market_score"`    // 市场表现得分
	FlowScore      float64 `gorm:"type:decimal(5,2)" json:"flow_score"`      // 资金流得分
	HeatScore      float64 `gorm:"type:decimal(5,2)" json:"heat_score"`      // 市场热度得分
	EventScore     float64 `gorm:"type:decimal(5,2)" json:"event_score"`     // 事件得分
	SentimentScore float64 `gorm:"type:decimal(5,2)" json:"sentiment_score"` // 情绪得分

	// 原始数据快照
	PriceChange24h   *float64 `gorm:"type:decimal(10,4)" json:"price_change_24h"`
	Volume24h        *float64 `gorm:"type:decimal(20,8)" json:"volume_24h"`
	MarketCapUSD     *float64 `gorm:"type:decimal(20,2)" json:"market_cap_usd"`
	NetFlow24h       *float64 `gorm:"type:decimal(20,8)" json:"net_flow_24h"`
	RecommendedPrice *float64 `gorm:"type:decimal(20,8)" json:"recommended_price"` // 推荐时币种价格
	HasNewListing    bool     `json:"has_new_listing"`
	HasAnnouncement  bool     `json:"has_announcement"`
	TwitterMentions  *int     `json:"twitter_mentions"`

	// 推荐理由（JSON格式）
	Reasons datatypes.JSON `json:"reasons"`

	// 风险评级字段
	VolatilityRisk *float64       `gorm:"type:decimal(5,2)" json:"volatility_risk"` // 波动率风险 0-100
	LiquidityRisk  *float64       `gorm:"type:decimal(5,2)" json:"liquidity_risk"`  // 流动性风险 0-100
	MarketRisk     *float64       `gorm:"type:decimal(5,2)" json:"market_risk"`     // 市场风险 0-100
	TechnicalRisk  *float64       `gorm:"type:decimal(5,2)" json:"technical_risk"`  // 技术风险 0-100
	OverallRisk    *float64       `gorm:"type:decimal(5,2)" json:"overall_risk"`    // 综合风险 0-100
	RiskLevel      *string        `gorm:"size:16" json:"risk_level"`                // low/medium/high
	RiskWarnings   datatypes.JSON `json:"risk_warnings"`                            // 风险提示（JSON数组）

	// 技术指标
	TechnicalIndicators datatypes.JSON `json:"technical_indicators"` // 技术指标（JSON格式）

	// 价格预测
	PricePrediction datatypes.JSON `json:"price_prediction"` // 价格预测（JSON格式）

	// 用户行为统计（用于A/B测试和效果分析）
	Impressions      int     `gorm:"default:0" json:"impressions"`                              // 曝光次数
	Clicks           int     `gorm:"default:0" json:"clicks"`                                   // 点击次数
	Saves            int     `gorm:"default:0" json:"saves"`                                    // 收藏次数
	Follows          int     `gorm:"default:0" json:"follows"`                                  // 关注次数
	AvgRating        float64 `gorm:"type:decimal(3,2);default:0.00" json:"avg_rating"`          // 平均评分
	FeedbackCount    int     `gorm:"default:0" json:"feedback_count"`                           // 反馈次数
	PerformanceScore float64 `gorm:"type:decimal(5,4);default:0.0000" json:"performance_score"` // 实际表现得分

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
		Order("generated_at DESC, `rank` ASC").
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

// GetRecommendationsByDate 根据日期获取推荐结果
// date: 日期字符串，格式 YYYY-MM-DD，会查询该日期当天的所有推荐
func GetRecommendationsByDate(gdb *gorm.DB, kind string, date time.Time) ([]CoinRecommendation, error) {
	var recommendations []CoinRecommendation

	// 计算日期范围（当天00:00:00到23:59:59）
	// 确保使用UTC时区，与数据库存储一致
	dateUTC := date.UTC()
	startTime := time.Date(dateUTC.Year(), dateUTC.Month(), dateUTC.Day(), 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	err := gdb.Where("kind = ? AND generated_at >= ? AND generated_at < ?", kind, startTime, endTime).
		Order("generated_at DESC, `rank` ASC").
		Find(&recommendations).Error

	return recommendations, err
}

// GetRecommendationsByDatePaginated 根据日期获取推荐结果（分页）
func GetRecommendationsByDatePaginated(gdb *gorm.DB, kind string, date time.Time, page, pageSize int) ([]CoinRecommendation, int64, error) {
	var recommendations []CoinRecommendation
	var total int64

	// 计算日期范围（当天00:00:00到23:59:59）
	// 确保使用UTC时区，与数据库存储一致
	dateUTC := date.UTC()
	startTime := time.Date(dateUTC.Year(), dateUTC.Month(), dateUTC.Day(), 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	// 查询总数
	err := gdb.Model(&CoinRecommendation{}).
		Where("kind = ? AND generated_at >= ? AND generated_at < ?", kind, startTime, endTime).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = gdb.Where("kind = ? AND generated_at >= ? AND generated_at < ?", kind, startTime, endTime).
		Order("generated_at DESC, `rank` ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&recommendations).Error

	return recommendations, total, err
}

// GetRecommendationsByTimeRange 根据时间范围获取推荐结果
func GetRecommendationsByTimeRange(gdb *gorm.DB, kind string, startTime, endTime time.Time) ([]CoinRecommendation, error) {
	var recommendations []CoinRecommendation

	err := gdb.Where("kind = ? AND generated_at >= ? AND generated_at <= ?", kind, startTime, endTime).
		Order("generated_at DESC, `rank` ASC").
		Find(&recommendations).Error

	return recommendations, err
}

// GetRecommendationTimeList 获取有推荐记录的时间列表（用于时间选择器）
func GetRecommendationTimeList(gdb *gorm.DB, kind string, limit int) ([]time.Time, error) {
	// 使用原生SQL查询，兼容MySQL和SQLite
	var results []struct {
		DateStr string `gorm:"column:date_str"`
	}

	// 注意：MySQL的DATE()函数返回的是DATE类型，需要确保正确解析
	err := gdb.Raw(`
		SELECT DISTINCT DATE(generated_at) as date_str 
		FROM coin_recommendations 
		WHERE kind = ? 
		ORDER BY date_str DESC 
		LIMIT ?
	`, kind, limit).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 将日期字符串转换为time.Time
	times := make([]time.Time, 0, len(results))
	for _, result := range results {
		// 处理不同数据库返回的日期格式
		dateStr := result.DateStr

		// MySQL可能返回 "2006-01-02" 或 "2006-01-02 00:00:00" 或 time.Time
		// 如果是time.Time类型，需要先转换为字符串
		if dateStr == "" {
			continue
		}

		// 如果包含时间部分，只取日期部分
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}

		// 尝试解析日期
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			times = append(times, t)
		} else {
			// 如果解析失败，记录日志但继续处理其他日期
			log.Printf("[GetRecommendationTimeList] Failed to parse date '%s': %v", dateStr, err)
		}
	}

	return times, nil
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
