package db

import (
	"analysis/internal/models"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SaveTransferEvents(gdb *gorm.DB, runID, entity string, events []models.Event) ([]TransferEvent, error) {
	if len(events) == 0 {
		return nil, nil
	}
	now := time.Now().UTC()
	rows := make([]TransferEvent, 0, len(events))

	// 过滤 amount == 0
	for _, e := range events {
		if isZero(e.Amount) {
			continue
		}
		ent := e.Entity
		if ent == "" {
			ent = entity
		}
		ts := e.TS
		if ts.IsZero() {
			ts = now
		}
		rows = append(rows, TransferEvent{
			RunID:      runID,
			Entity:     ent,
			Chain:      e.Chain,
			Coin:       e.Coin,
			Direction:  e.Direction,
			Amount:     strings.TrimSpace(e.Amount),
			TxID:       e.TxID,
			Address:    e.Address,
			From:       e.From,
			To:         e.To,
			LogIndex:   e.LogIndex,
			OccurredAt: ts.UTC(),
			CreatedAt:  now,
		})
	}
	if len(rows) == 0 {
		return nil, nil
	}

	// 唯一键冲突忽略
	if err := gdb.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
		return nil, err
	}

	// 仅返回真正新插入的记录（ID>0）
	inserted := make([]TransferEvent, 0, len(rows))
	for _, r := range rows {
		if r.ID > 0 {
			inserted = append(inserted, r)
		}
	}
	return inserted, nil
}

func isZero(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return false
	}
	return r.Cmp(new(big.Rat)) == 0
}
