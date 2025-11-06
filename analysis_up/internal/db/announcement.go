package db

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BinanceAnnouncement represents a tracked Binance announcement entry.
type BinanceAnnouncement struct {
	ID uint `gorm:"primaryKey"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Type string `gorm:"size:32;index:idx_binance_announcement_type_code,priority:1"`
	Code string `gorm:"size:128;index:idx_binance_announcement_type_code,priority:2"`

	Title      string    `gorm:"size:512"`
	URL        string    `gorm:"size:1024"`
	ReleasedAt time.Time `gorm:"index"`
}

// TableName overrides the default table name.
func (BinanceAnnouncement) TableName() string {
	return "binance_announcements"
}

// InsertBinanceAnnouncement inserts the record if it does not exist. It returns true when inserted.
func InsertBinanceAnnouncement(gdb *gorm.DB, ann *BinanceAnnouncement) (bool, error) {
	if ann == nil {
		return false, nil
	}
	res := gdb.Clauses(clause.OnConflict{DoNothing: true}).Create(ann)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
