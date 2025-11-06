package db

import "time"

// BracketLink 记录一次“一键三连”的三张订单的 ClientOrderID
type BracketLink struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ScheduleID    uint      `gorm:"index"      json:"schedule_id"`
	Symbol        string    `gorm:"size:32;index" json:"symbol"`
	GroupID       string    `gorm:"size:64;uniqueIndex" json:"group_id"`
	EntryClientID string    `gorm:"size:64" json:"entry_client_id"`
	TPClientID    string    `gorm:"size:64" json:"tp_client_id"`
	SLClientID    string    `gorm:"size:64" json:"sl_client_id"`
	Status        string    `gorm:"size:16;index;" json:"status"` // active/closed/cancelled
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
