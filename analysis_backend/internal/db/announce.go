package db

import (
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Announcement struct {
	ID          uint64                       `gorm:"primaryKey;autoIncrement" json:"id"`
	Source      string                       `gorm:"type:varchar(32);not null;index" json:"source"` // coincarp, cryptopanic, coinmarketcal, binance, okx, upbit, bybit
	ExternalID  string                       `gorm:"type:varchar(128);not null" json:"external_id"`
	NewsCode    string                       `gorm:"type:varchar(256);index" json:"news_code"` // CoinCarp newscode，用于构建详情页 URL
	Title       string                       `gorm:"type:varchar(512);not null" json:"title"`
	Summary     string                       `gorm:"type:text" json:"summary"`
	URL         string                       `gorm:"type:varchar(1024);unique'" json:"url"`
	Category    string                       `gorm:"type:varchar(16);index" json:"category"` // newcoin | finance | other | event
	Tags        datatypes.JSONType[[]string] `gorm:"type:json" json:"tags"`
	ReleaseTime time.Time                    `gorm:"index" json:"release_time"`
	Raw         datatypes.JSON               `gorm:"type:json" json:"raw"`
	// 新增字段：多层次抓取支持
	IsEvent   bool      `gorm:"default:false;index" json:"is_event"`     // 是否为重要事件（第二层验证标记）
	Sentiment string    `gorm:"type:varchar(16);index" json:"sentiment"` // positive | neutral | negative
	HeatScore int       `gorm:"default:0;index" json:"heat_score"`       // 热度分数 0-100
	Exchange  string    `gorm:"type:varchar(32);index" json:"exchange"`  // 交易所名称（从 coincarp 提取）
	Verified  bool      `gorm:"default:false" json:"verified"`           // 是否经过官方源验证（第三层）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 批量Upsert（URL 唯一，支持多数据源合并）
func SaveAnnouncements(db *gorm.DB, items []Announcement) ([]Announcement, error) {
	if len(items) == 0 {
		return nil, nil
	}
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "summary", "category", "tags", "release_time", "raw",
			"is_event", "sentiment", "heat_score", "exchange", "verified", "news_code", "updated_at",
		}),
	}).Create(&items).Error
	return items, err
}

// 合并多数据源的公告（用于去重和验证）
func MergeAnnouncements(db *gorm.DB, items []Announcement) error {
	if len(items) == 0 {
		return nil
	}
	// 按 URL 分组，合并不同数据源的信息
	urlMap := make(map[string]*Announcement)
	for i := range items {
		item := &items[i]
		url := strings.TrimSpace(item.URL)
		if url == "" {
			continue
		}
		if existing, ok := urlMap[url]; ok {
			// 合并逻辑：保留更权威的数据源
			if item.Verified && !existing.Verified {
				*existing = *item
			} else if item.IsEvent && !existing.IsEvent {
				existing.IsEvent = true
				if item.HeatScore > existing.HeatScore {
					existing.HeatScore = item.HeatScore
				}
				if item.Sentiment != "" && existing.Sentiment == "" {
					existing.Sentiment = item.Sentiment
				}
			}
		} else {
			urlMap[url] = item
		}
	}
	merged := make([]Announcement, 0, len(urlMap))
	for _, item := range urlMap {
		merged = append(merged, *item)
	}
	_, err := SaveAnnouncements(db, merged)
	return err
}
