package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ===== 快照管理器 =====
// 管理实时涨幅榜快照的创建、清理和查询优化

// SnapshotManager 快照管理器
type SnapshotManager struct {
	db   *gorm.DB
	kind string // "spot" 或 "futures"

	// 清理配置
	retentionPeriod time.Duration // 快照保留时长
	maxSnapshots    int           // 最大快照数量
	cleanupInterval time.Duration // 清理间隔

	// 统计信息
	totalSnapshots   int64     // 总快照数
	cleanedSnapshots int64     // 已清理快照数
	lastCleanupTime  time.Time // 最后清理时间
}

// NewSnapshotManager 创建快照管理器
func NewSnapshotManager(db *gorm.DB, kind string) *SnapshotManager {
	manager := &SnapshotManager{
		db:              db,
		kind:            kind,
		retentionPeriod: 1 * time.Hour,    // 默认保留1小时
		maxSnapshots:    10,               // 默认最多10个快照
		cleanupInterval: 10 * time.Minute, // 默认10分钟清理一次
	}

	// 启动清理goroutine
	go manager.startCleanupRoutine()

	log.Printf("[SnapshotManager] 初始化完成 - 类型:%s, 保留时长:%v, 最大数量:%d",
		kind, manager.retentionPeriod, manager.maxSnapshots)
	return manager
}

// CreateSnapshot 创建新的快照
func (m *SnapshotManager) CreateSnapshot(timestamp time.Time) (*SnapshotInfo, error) {
	snapshot := &SnapshotInfo{
		Kind:      m.kind,
		Timestamp: timestamp,
		CreatedAt: time.Now(),
	}

	// 在数据库中创建快照记录
	// 注意：这里我们直接使用gorm创建记录
	dbSnapshot := map[string]interface{}{
		"kind":       m.kind,
		"timestamp":  timestamp,
		"created_at": time.Now(),
	}

	result := m.db.Table("realtime_gainers_snapshots").Create(dbSnapshot)
	if result.Error != nil {
		return nil, fmt.Errorf("创建快照失败: %w", result.Error)
	}

	// 获取创建的快照ID
	var createdSnapshot struct {
		ID uint `gorm:"primarykey"`
	}
	m.db.Table("realtime_gainers_snapshots").Where("kind = ? AND timestamp = ?", m.kind, timestamp).Order("id DESC").First(&createdSnapshot)
	snapshot.ID = createdSnapshot.ID

	m.totalSnapshots++
	log.Printf("[SnapshotManager] 创建快照: ID=%d, 类型=%s, 时间=%v", snapshot.ID, m.kind, timestamp)
	return snapshot, nil
}

// GetLatestSnapshot 获取最新的快照
func (m *SnapshotManager) GetLatestSnapshot() (*SnapshotInfo, error) {
	var snapshot SnapshotInfo

	query := `
		SELECT id, kind, timestamp, created_at
		FROM realtime_gainers_snapshots
		WHERE kind = ?
		ORDER BY timestamp DESC, id DESC
		LIMIT 1
	`

	err := m.db.Raw(query, m.kind).Scan(&snapshot).Error
	if err != nil {
		return nil, fmt.Errorf("获取最新快照失败: %w", err)
	}

	if snapshot.ID == 0 {
		return nil, fmt.Errorf("没有找到快照")
	}

	return &snapshot, nil
}

// GetSnapshotCount 获取快照数量
func (m *SnapshotManager) GetSnapshotCount() (int64, error) {
	var count int64
	err := m.db.Model(&map[string]interface{}{}).
		Table("realtime_gainers_snapshots").
		Where("kind = ?", m.kind).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("获取快照数量失败: %w", err)
	}

	return count, nil
}

// CleanupSnapshots 清理过期快照
func (m *SnapshotManager) CleanupSnapshots() error {
	log.Printf("[SnapshotManager] 开始清理快照 - 类型:%s", m.kind)

	startTime := time.Now()
	cleanedCount := int64(0)

	// 1. 删除过期快照（按时间）
	cutoffTime := time.Now().Add(-m.retentionPeriod)
	timeDeleteQuery := `
		DELETE FROM realtime_gainers_snapshots
		WHERE kind = ? AND timestamp < ?
	`
	result := m.db.Exec(timeDeleteQuery, m.kind, cutoffTime)
	if result.Error != nil {
		return fmt.Errorf("删除过期快照失败: %w", result.Error)
	}
	timeDeleted := result.RowsAffected
	cleanedCount += timeDeleted

	// 2. 删除超出数量限制的快照（保留最新的N个）
	countDeleteQuery := `
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
	result = m.db.Exec(countDeleteQuery, m.kind, m.kind, m.maxSnapshots)
	if result.Error != nil {
		return fmt.Errorf("删除超出数量快照失败: %w", result.Error)
	}
	countDeleted := result.RowsAffected
	cleanedCount += countDeleted

	// 更新统计信息
	m.cleanedSnapshots += cleanedCount
	m.lastCleanupTime = time.Now()

	cleanupDuration := time.Since(startTime)
	log.Printf("[SnapshotManager] 清理完成 - 删除快照:%d (时间:%d, 数量:%d), 耗时:%v",
		cleanedCount, timeDeleted, countDeleted, cleanupDuration)

	return nil
}

// GetSnapshotStats 获取快照统计信息
func (m *SnapshotManager) GetSnapshotStats() (map[string]interface{}, error) {
	currentCount, err := m.GetSnapshotCount()
	if err != nil {
		return nil, err
	}

	latestSnapshot, err := m.GetLatestSnapshot()
	latestTime := time.Time{}
	if err == nil && latestSnapshot != nil {
		latestTime = latestSnapshot.Timestamp
	}

	return map[string]interface{}{
		"current_count":        currentCount,
		"total_created":        m.totalSnapshots,
		"total_cleaned":        m.cleanedSnapshots,
		"retention_period":     m.retentionPeriod.String(),
		"max_snapshots":        m.maxSnapshots,
		"cleanup_interval":     m.cleanupInterval.String(),
		"last_cleanup_time":    m.lastCleanupTime,
		"latest_snapshot_time": latestTime,
	}, nil
}

// SetRetentionPolicy 设置保留策略
func (m *SnapshotManager) SetRetentionPolicy(retentionPeriod time.Duration, maxSnapshots int) {
	if retentionPeriod > 0 {
		m.retentionPeriod = retentionPeriod
	}
	if maxSnapshots > 0 {
		m.maxSnapshots = maxSnapshots
	}

	log.Printf("[SnapshotManager] 更新保留策略 - 时长:%v, 最大数量:%d",
		m.retentionPeriod, m.maxSnapshots)
}

// SetCleanupInterval 设置清理间隔
func (m *SnapshotManager) SetCleanupInterval(interval time.Duration) {
	if interval > 0 {
		m.cleanupInterval = interval
		log.Printf("[SnapshotManager] 更新清理间隔: %v", interval)
	}
}

// startCleanupRoutine 启动清理例程
func (m *SnapshotManager) startCleanupRoutine() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := m.CleanupSnapshots(); err != nil {
			log.Printf("[SnapshotManager] 自动清理失败: %v", err)
		}
	}
}

// SnapshotInfo 快照信息
type SnapshotInfo struct {
	ID        uint      `json:"id"`
	Kind      string    `json:"kind"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}
