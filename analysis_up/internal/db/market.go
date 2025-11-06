// internal/db/market.go
package db

import (
	"time"

	"gorm.io/gorm"
)

// 一次 2 小时的市场快照
type BinanceMarketSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"size:16;index:idx_market_kind_bucket,priority:1" json:"kind"` // spot / futures
	Bucket    time.Time `gorm:"index:idx_market_kind_bucket,priority:2" json:"bucket"`       // 2h 对齐的时间
	FetchedAt time.Time `json:"fetched_at"`
	CreatedAt time.Time `json:"created_at"`
}

// 快照中的一条 TOP 数据
type BinanceMarketTop struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	SnapshotID uint    `gorm:"index" json:"snapshot_id"`
	Symbol     string  `gorm:"size:32;index" json:"symbol"`
	LastPrice  string  `gorm:"size:64" json:"last_price"`
	Volume     string  `gorm:"size:64" json:"volume"`
	PctChange  float64 `json:"pct_change"`
	// 注意：rank 在部分 MySQL 版本里是敏感的，这里字段名还是 rank，
	// 但我们在查询里会用 `rank` 包起来
	Rank      int       `gorm:"index" json:"rank"`
	CreatedAt time.Time `json:"created_at"`
}

// 保存一整份快照（同 kind+bucket 会被覆盖）
func SaveBinanceMarket(gdb *gorm.DB, kind string, bucket, fetchedAt time.Time, items []BinanceMarketTop) (*BinanceMarketSnapshot, error) {
	snap := &BinanceMarketSnapshot{
		Kind:      kind,
		Bucket:    bucket,
		FetchedAt: fetchedAt,
	}
	err := gdb.Transaction(func(tx *gorm.DB) error {
		// 1) 看看有没有老的同一个时间桶
		var old BinanceMarketSnapshot
		if err := tx.Where("kind = ? AND bucket = ?", kind, bucket).First(&old).Error; err == nil {
			// 有老的，先把下面的 top 删了
			if err := tx.Where("snapshot_id = ?", old.ID).Delete(&BinanceMarketTop{}).Error; err != nil {
				return err
			}
			// 再删掉老的 snapshot
			if err := tx.Delete(&old).Error; err != nil {
				return err
			}
		} else if err != gorm.ErrRecordNotFound {
			// 真错了再返回
			return err
		}

		// 2) 插入新的 snapshot
		if err := tx.Create(snap).Error; err != nil {
			return err
		}

		// 3) 插入新的 top
		for i := range items {
			items[i].SnapshotID = snap.ID
			items[i].Rank = i + 1
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// 按时间范围读取快照 + TOP
func ListBinanceMarket(gdb *gorm.DB, kind string, start, end time.Time) ([]BinanceMarketSnapshot, map[uint][]BinanceMarketTop, error) {
	var snaps []BinanceMarketSnapshot
	q := gdb.Where("kind = ?", kind)
	if !start.IsZero() {
		q = q.Where("bucket >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("bucket <= ?", end)
	}
	if err := q.Order("bucket asc").Find(&snaps).Error; err != nil {
		return nil, nil, err
	}

	if len(snaps) == 0 {
		return snaps, map[uint][]BinanceMarketTop{}, nil
	}

	// 收集 id
	ids := make([]uint, 0, len(snaps))
	for _, s := range snaps {
		ids = append(ids, s.ID)
	}

	var tops []BinanceMarketTop
	if err := gdb.
		Where("snapshot_id IN ?", ids).
		Order("snapshot_id asc, `rank` asc").
		Find(&tops).Error; err != nil {
		return nil, nil, err
	}

	// 按 snapshot_id 分组
	grouped := make(map[uint][]BinanceMarketTop)
	for _, t := range tops {
		grouped[t.SnapshotID] = append(grouped[t.SnapshotID], t)
	}

	return snaps, grouped, nil
}
