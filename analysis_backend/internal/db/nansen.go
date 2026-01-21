package db

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// NansenWhaleWatch 存储 Nansen 监控数据
type NansenWhaleWatch struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	Address          string          `gorm:"size:128;uniqueIndex" json:"address"`
	Label            string          `gorm:"size:128" json:"label"`
	Chain            string          `gorm:"size:32;index" json:"chain"`
	Entity           string          `gorm:"size:64;index" json:"entity"`
	BalanceUSD       string          `gorm:"size:64" json:"balance_usd"`
	LastActiveAt     time.Time       `json:"last_active_at"`
	LastSnapshotAt   time.Time       `json:"last_snapshot_at"`
	TransactionsJSON json.RawMessage `gorm:"type:json" json:"transactions_json"`
	MetadataJSON     json.RawMessage `gorm:"type:json" json:"metadata_json"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// ListNansenWatches 查询所有 Nansen 监控地址，按创建时间倒序返回
func ListNansenWatches(gdb *gorm.DB) ([]NansenWhaleWatch, error) {
	var watches []NansenWhaleWatch
	if err := gdb.Order("created_at DESC").Find(&watches).Error; err != nil {
		return nil, err
	}
	return watches, nil
}

// GetNansenWatchByAddress 按地址查询
func GetNansenWatchByAddress(gdb *gorm.DB, address string) (*NansenWhaleWatch, error) {
	var watch NansenWhaleWatch
	if err := gdb.Where("address = ?", address).First(&watch).Error; err != nil {
		return nil, err
	}
	return &watch, nil
}

// CreateOrUpdateNansenWatch 创建或更新 Nansen 监控
func CreateOrUpdateNansenWatch(gdb *gorm.DB, watch *NansenWhaleWatch) error {
	return gdb.Where(NansenWhaleWatch{Address: watch.Address}).
		Assign(*watch).
		FirstOrCreate(watch).Error
}

// DeleteNansenWatchByAddress 删除
func DeleteNansenWatchByAddress(gdb *gorm.DB, address string) error {
	return gdb.Where("address = ?", address).Delete(&NansenWhaleWatch{}).Error
}
