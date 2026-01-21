package db

import (
	"time"

	"gorm.io/gorm"
)

// WhaleWatch 记录需要持续监控的钱包地址（可配合 Arkham 等链上情报服务）
type WhaleWatch struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Label     string    `gorm:"size:128" json:"label"`
	Address   string    `gorm:"size:128;uniqueIndex" json:"address"`
	Chain     string    `gorm:"size:32;index" json:"chain"`
	Entity    string    `gorm:"size:64;index" json:"entity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListWhaleWatches 查询所有监控地址，按创建时间倒序返回
func ListWhaleWatches(gdb *gorm.DB) ([]WhaleWatch, error) {
	var watches []WhaleWatch
	if err := gdb.Order("created_at DESC").Find(&watches).Error; err != nil {
		return nil, err
	}
	return watches, nil
}

// GetWhaleWatchByAddress 按地址查询
func GetWhaleWatchByAddress(gdb *gorm.DB, address string) (*WhaleWatch, error) {
	var watch WhaleWatch
	if err := gdb.Where("address = ?", address).First(&watch).Error; err != nil {
		return nil, err
	}
	return &watch, nil
}

// CreateWhaleWatch 创建新监控
func CreateWhaleWatch(gdb *gorm.DB, watch *WhaleWatch) error {
	return gdb.Create(watch).Error
}

// DeleteWhaleWatchByAddress 删除
func DeleteWhaleWatchByAddress(gdb *gorm.DB, address string) error {
	return gdb.Where("address = ?", address).Delete(&WhaleWatch{}).Error
}
