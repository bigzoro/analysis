package db

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ArkhamWatch 保存 Arkham 返回的大户/机构地址快照
type ArkhamWatch struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	Address        string          `gorm:"size:128;uniqueIndex" json:"address"`
	Label          string          `gorm:"size:128" json:"label"`
	Chain          string          `gorm:"size:32" json:"chain"`
	Entity         string          `gorm:"size:64" json:"entity"`
	BalanceUSD     string          `gorm:"size:64" json:"balance_usd"`
	LastActiveAt   time.Time       `json:"last_active_at"`
	LastSnapshotAt time.Time       `json:"last_snapshot_at"`
	EventsJSON     json.RawMessage `gorm:"type:json" json:"events"`
	MetadataJSON   json.RawMessage `gorm:"type:json" json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ListArkhamWatches 查询所有 Arkham watch 配置
func ListArkhamWatches(gdb *gorm.DB) ([]ArkhamWatch, error) {
	var list []ArkhamWatch
	if err := gdb.Order("updated_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetArkhamWatchByAddress 查询单个地址
func GetArkhamWatchByAddress(gdb *gorm.DB, address string) (*ArkhamWatch, error) {
	var watch ArkhamWatch
	if err := gdb.Where("address = ?", address).First(&watch).Error; err != nil {
		return nil, err
	}
	return &watch, nil
}

// CreateOrUpdateArkhamWatch 新增或更新地址
func CreateOrUpdateArkhamWatch(gdb *gorm.DB, watch *ArkhamWatch) error {
	return gdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		UpdateAll: true,
	}).Create(watch).Error
}

// UpdateArkhamWatchSnapshot 更新快照字段
func UpdateArkhamWatchSnapshot(gdb *gorm.DB, address string, balance string, events json.RawMessage, metadata json.RawMessage, lastActive, snapshot time.Time) error {
	return gdb.Model(&ArkhamWatch{}).
		Where("address = ?", address).
		Updates(map[string]interface{}{
			"balance_usd":      balance,
			"events":           events,
			"metadata":         metadata,
			"last_active_at":   lastActive,
			"last_snapshot_at": snapshot,
		}).Error
}

// DeleteArkhamWatchByAddress 删除监控
func DeleteArkhamWatchByAddress(gdb *gorm.DB, address string) error {
	return gdb.Where("address = ?", address).Delete(&ArkhamWatch{}).Error
}
