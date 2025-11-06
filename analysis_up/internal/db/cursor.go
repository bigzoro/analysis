package db

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetCursor(gdb *gorm.DB, entity, chain string) (uint64, error) {
	var c TransferCursor
	err := gdb.Where("entity = ? AND chain = ?", entity, chain).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return c.Block, nil
}

func UpsertCursor(gdb *gorm.DB, entity, chain string, block uint64) error {
	now := time.Now().UTC()
	c := TransferCursor{
		Entity: entity,
		Chain:  chain,
		Block:  block,
	}
	return gdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "entity"}, {Name: "chain"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"block": block, "updated_at": now}),
	}).Create(&c).Error
}
