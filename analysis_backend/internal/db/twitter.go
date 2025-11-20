package db

import (
	"time"

	"gorm.io/gorm"
)

type TwitterPost struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:32;index:idx_user_tid,priority:1" json:"username"` // 全小写
	TweetID   string    `gorm:"size:32;index:idx_user_tid,priority:2" json:"tweet_id"`
	Text      string    `gorm:"type:text" json:"text"`
	URL       string    `gorm:"size:256" json:"url"`
	TweetTime time.Time `gorm:"index" json:"tweet_time"` // UTC
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func SaveTwitterPosts(gdb *gorm.DB, items []TwitterPost) ([]TwitterPost, error) {
	var inserted []TwitterPost
	err := gdb.Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			var exist TwitterPost
			if err := tx.Where("username=? AND tweet_id=?", it.Username, it.TweetID).First(&exist).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					if err := tx.Create(&it).Error; err != nil {
						return err
					}
					inserted = append(inserted, it)
				} else {
					return err
				}
			}
		}
		return nil
	})
	return inserted, err
}
