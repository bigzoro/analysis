package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// ===== 保存控制器 =====
// 控制实时涨幅榜数据的数据库保存，包括事务管理和性能优化

// SaveController 保存控制器
type SaveController struct {
	db     *gorm.DB
	kind   string // "spot" 或 "futures"

	// 保存配置
	batchSize      int           // 批量保存大小
	saveTimeout    time.Duration // 保存超时时间
	retryAttempts  int           // 重试次数
	retryDelay     time.Duration // 重试延迟

	// 统计信息
	totalSaves     int64
	successSaves   int64
	failedSaves    int64
	avgSaveTime    time.Duration
	lastSaveTime   time.Time
}

// NewSaveController 创建保存控制器
func NewSaveController(db *gorm.DB, kind string) *SaveController {
	return &SaveController{
		db:             db,
		kind:           kind,
		batchSize:      50,                // 默认批量50条
		saveTimeout:    30 * time.Second,  // 默认30秒超时
		retryAttempts:  3,                 // 默认重试3次
		retryDelay:     1 * time.Second,   // 默认1秒重试延迟
	}
}

// SaveRealtimeGainers 保存实时涨幅榜数据
func (c *SaveController) SaveRealtimeGainers(gainers []RealtimeGainerItem) error {
	if len(gainers) == 0 {
		return fmt.Errorf("涨幅榜数据为空")
	}

	startTime := time.Now()
	c.totalSaves++

	// 设置保存超时
	ctx, cancel := context.WithTimeout(context.Background(), c.saveTimeout)
	defer cancel()

	var lastErr error

	// 重试逻辑
	for attempt := 0; attempt < c.retryAttempts; attempt++ {
		if attempt > 0 {
			log.Printf("[SaveController] 保存重试 %d/%d, 延迟 %v", attempt+1, c.retryAttempts, c.retryDelay)
			time.Sleep(c.retryDelay)
		}

		err := c.saveWithTransaction(ctx, gainers)
		if err == nil {
			// 保存成功
			saveDuration := time.Since(startTime)
			c.successSaves++
			c.lastSaveTime = time.Now()

			// 更新平均保存时间
			if c.successSaves == 1 {
				c.avgSaveTime = saveDuration
			} else {
				c.avgSaveTime = (c.avgSaveTime + saveDuration) / 2
			}

			log.Printf("[SaveController] 保存成功: %d个项目, 耗时:%v, 平均耗时:%v",
				len(gainers), saveDuration, c.avgSaveTime)
			return nil
		}

		lastErr = err
		log.Printf("[SaveController] 保存失败 (尝试 %d/%d): %v", attempt+1, c.retryAttempts, err)
	}

	// 所有重试都失败了
	c.failedSaves++
	log.Printf("[SaveController] 保存最终失败: %d个项目, 总耗时:%v, 错误:%v",
		len(gainers), time.Since(startTime), lastErr)
	return fmt.Errorf("保存失败 after %d attempts: %w", c.retryAttempts, lastErr)
}

// saveWithTransaction 使用事务保存数据
func (c *SaveController) saveWithTransaction(ctx context.Context, gainers []RealtimeGainerItem) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建快照
		snapshot := &pdb.RealtimeGainersSnapshot{
			Kind:      c.kind,
			Timestamp: time.Now(),
		}

		if err := tx.Create(snapshot).Error; err != nil {
			return fmt.Errorf("创建快照失败: %w", err)
		}

		// 2. 批量保存涨幅榜数据
		return c.batchSaveGainers(tx, snapshot.ID, gainers)
	})
}

// batchSaveGainers 批量保存涨幅榜数据
func (c *SaveController) batchSaveGainers(tx *gorm.DB, snapshotID uint, gainers []RealtimeGainerItem) error {
	// 分批保存，避免一次性保存太多数据
	batchSize := c.batchSize
	totalBatches := (len(gainers) + batchSize - 1) / batchSize

	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(gainers) {
			end = len(gainers)
		}

		batch := gainers[start:end]
		if err := c.saveBatch(tx, snapshotID, batch); err != nil {
			return fmt.Errorf("保存批次 %d/%d 失败: %w", i+1, totalBatches, err)
		}
	}

	return nil
}

// saveBatch 保存单个批次的数据
func (c *SaveController) saveBatch(tx *gorm.DB, snapshotID uint, batch []RealtimeGainerItem) error {
	items := make([]pdb.RealtimeGainersItem, 0, len(batch))

	for _, gainer := range batch {
		item := pdb.RealtimeGainersItem{
			SnapshotID:     snapshotID,
			Symbol:         gainer.Symbol,
			Rank:           gainer.Rank,
			CurrentPrice:   gainer.CurrentPrice,
			PriceChange24h: gainer.ChangePercent,
			Volume24h:      gainer.Volume24h,
			DataSource:     gainer.DataSource,
		}

		// 可选字段处理
		if gainer.ChangePercent != 0 {
			pc := gainer.ChangePercent
			item.PriceChangePercent = &pc
		}

		items = append(items, item)
	}

	// 批量插入
	if err := tx.CreateInBatches(items, len(items)).Error; err != nil {
		return fmt.Errorf("批量插入失败: %w", err)
	}

	log.Printf("[SaveController] 批次保存完成: %d个项目", len(items))
	return nil
}

// CleanupOldSnapshots 清理旧快照
func (c *SaveController) CleanupOldSnapshots(retentionPeriod time.Duration, maxSnapshots int) error {
	log.Printf("[SaveController] 开始清理旧快照 - 保留时长:%v, 最大数量:%d", retentionPeriod, maxSnapshots)

	// 计算清理截止时间
	cutoffTime := time.Now().Add(-retentionPeriod)

	// 1. 删除过期快照
	deleteQuery := `
		DELETE FROM realtime_gainers_snapshots
		WHERE kind = ? AND timestamp < ?
	`
	result := c.db.Exec(deleteQuery, c.kind, cutoffTime)
	if result.Error != nil {
		return fmt.Errorf("删除过期快照失败: %w", result.Error)
	}
	expiredDeleted := result.RowsAffected

	// 2. 限制快照数量（保留最新的N个）
	limitQuery := `
		DELETE FROM realtime_gainers_snapshots
		WHERE kind = ? AND id NOT IN (
			SELECT id FROM (
				SELECT id FROM realtime_gainers_snapshots
				WHERE kind = ?
				ORDER BY timestamp DESC, id DESC
				LIMIT ?
			) latest_snapshots
		)
	`
	result = c.db.Exec(limitQuery, c.kind, c.kind, maxSnapshots)
	if result.Error != nil {
		return fmt.Errorf("限制快照数量失败: %w", result.Error)
	}
	limitDeleted := result.RowsAffected

	log.Printf("[SaveController] 清理完成 - 删除过期快照:%d, 删除超出数量快照:%d",
		expiredDeleted, limitDeleted)
	return nil
}

// GetSaveStats 获取保存统计信息
func (c *SaveController) GetSaveStats() map[string]interface{} {
	return map[string]interface{}{
		"total_saves":      c.totalSaves,
		"success_saves":    c.successSaves,
		"failed_saves":     c.failedSaves,
		"success_rate":     c.calculateSuccessRate(),
		"avg_save_time":    c.avgSaveTime.String(),
		"last_save_time":   c.lastSaveTime,
		"batch_size":       c.batchSize,
		"save_timeout":     c.saveTimeout.String(),
		"retry_attempts":   c.retryAttempts,
		"retry_delay":      c.retryDelay.String(),
	}
}

// calculateSuccessRate 计算成功率
func (c *SaveController) calculateSuccessRate() float64 {
	if c.totalSaves == 0 {
		return 0
	}
	return float64(c.successSaves) / float64(c.totalSaves) * 100
}

// SetBatchSize 设置批量大小
func (c *SaveController) SetBatchSize(size int) {
	if size > 0 {
		c.batchSize = size
		log.Printf("[SaveController] 批量大小已更新: %d", size)
	}
}

// SetTimeouts 设置超时参数
func (c *SaveController) SetTimeouts(saveTimeout time.Duration, retryDelay time.Duration) {
	if saveTimeout > 0 {
		c.saveTimeout = saveTimeout
	}
	if retryDelay > 0 {
		c.retryDelay = retryDelay
	}
	log.Printf("[SaveController] 超时参数已更新 - 保存超时:%v, 重试延迟:%v",
		c.saveTimeout, c.retryDelay)
}

// SetRetryAttempts 设置重试次数
func (c *SaveController) SetRetryAttempts(attempts int) {
	if attempts > 0 {
		c.retryAttempts = attempts
		log.Printf("[SaveController] 重试次数已更新: %d", attempts)
	}
}