package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// ==================== 定时推荐生成器 ====================

// RecommendationScheduler 推荐调度器
type RecommendationScheduler struct {
	server    *Server
	mu        sync.RWMutex
	isRunning bool
	stopChan  chan struct{}

	// 配置
	interval time.Duration // 生成间隔
	maxAge   time.Duration // 推荐最大年龄
	kinds    []string      // 推荐类型
	limits   []int         // 推荐数量

	// 统计
	generatedCount int64
	lastRunTime    *time.Time
	nextRunTime    *time.Time
}

// NewRecommendationScheduler 创建推荐调度器
func NewRecommendationScheduler(server *Server) *RecommendationScheduler {
	return &RecommendationScheduler{
		server:   server,
		stopChan: make(chan struct{}),
		interval: 30 * time.Minute, // 默认30分钟生成一次
		maxAge:   45 * time.Minute, // 推荐45分钟后过期
		kinds:    []string{"spot", "futures"},
		limits:   []int{5, 10, 20}, // 生成不同数量的推荐
	}
}

// Start 启动定时推荐生成
func (rs *RecommendationScheduler) Start() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	rs.isRunning = true
	log.Printf("[RecommendationScheduler] Starting with interval: %v", rs.interval)

	go rs.runScheduler()
	return nil
}

// Stop 停止定时推荐生成
func (rs *RecommendationScheduler) Stop() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if !rs.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	rs.isRunning = false
	close(rs.stopChan)
	log.Printf("[RecommendationScheduler] Stopped")

	return nil
}

// runScheduler 运行调度器主循环
func (rs *RecommendationScheduler) runScheduler() {
	ticker := time.NewTicker(rs.interval)
	defer ticker.Stop()

	// 启动时立即执行一次
	rs.generateRecommendations()

	for {
		select {
		case <-rs.stopChan:
			return
		case <-ticker.C:
			rs.generateRecommendations()
		}
	}
}

// generateRecommendations 生成推荐
func (rs *RecommendationScheduler) generateRecommendations() {
	rs.mu.Lock()
	now := time.Now().UTC()
	rs.lastRunTime = &now
	rs.nextRunTime = &time.Time{}
	*rs.nextRunTime = now.Add(rs.interval)
	rs.mu.Unlock()

	log.Printf("[RecommendationScheduler] Starting scheduled recommendation generation at %s", now.Format(time.RFC3339))

	ctx := context.Background()
	totalGenerated := 0

	// 为每种类型和数量组合生成推荐
	for _, kind := range rs.kinds {
		for _, limit := range rs.limits {
			if err := rs.generateRecommendationBatch(ctx, kind, limit); err != nil {
				log.Printf("[RecommendationScheduler] Failed to generate recommendations (kind=%s, limit=%d): %v", kind, limit, err)
			} else {
				totalGenerated++
			}
		}
	}

	rs.mu.Lock()
	rs.generatedCount += int64(totalGenerated)
	rs.mu.Unlock()

	log.Printf("[RecommendationScheduler] Completed scheduled generation: %d batches generated", totalGenerated)
}

// generateRecommendationBatch 生成一批推荐
func (rs *RecommendationScheduler) generateRecommendationBatch(ctx context.Context, kind string, limit int) error {
	// 检查是否已经有足够新的推荐
	if rs.hasRecentRecommendations(kind, limit) {
		log.Printf("[RecommendationScheduler] Skipping generation (kind=%s, limit=%d): recent recommendations exist", kind, limit)
		return nil
	}

	// 生成推荐
	recommendations, err := rs.server.generateRecommendations(ctx, kind, limit)
	if err != nil {
		return fmt.Errorf("failed to generate recommendations: %w", err)
	}

	if len(recommendations) == 0 {
		log.Printf("[RecommendationScheduler] No recommendations generated (kind=%s, limit=%d)", kind, limit)
		return nil
	}

	// 保存到数据库
	if err := rs.saveScheduledRecommendations(ctx, recommendations, kind); err != nil {
		return fmt.Errorf("failed to save recommendations: %w", err)
	}

	log.Printf("[RecommendationScheduler] Generated and saved %d recommendations (kind=%s, limit=%d)",
		len(recommendations), kind, limit)
	return nil
}

// hasRecentRecommendations 检查是否有足够新的推荐
func (rs *RecommendationScheduler) hasRecentRecommendations(kind string, limit int) bool {
	// 查询最近的推荐
	var count int64
	rs.server.db.DB().Model(&pdb.CoinRecommendation{}).
		Where("kind = ? AND created_at > ?",
			kind,
			time.Now().UTC().Add(-rs.maxAge)).
		Count(&count)

	return count > 0
}

// saveScheduledRecommendations 保存定时生成的推荐
func (rs *RecommendationScheduler) saveScheduledRecommendations(ctx context.Context, recommendations []pdb.CoinRecommendation, kind string) error {
	// 设置生成时间
	now := time.Now().UTC()
	for i := range recommendations {
		recommendations[i].GeneratedAt = now
	}

	// 保存到数据库
	return pdb.SaveRecommendations(rs.server.db.DB(), kind, now, recommendations)
}

// GetStatus 获取调度器状态
func (rs *RecommendationScheduler) GetStatus() map[string]interface{} {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	status := map[string]interface{}{
		"is_running":      rs.isRunning,
		"interval":        rs.interval.String(),
		"max_age":         rs.maxAge.String(),
		"kinds":           rs.kinds,
		"limits":          rs.limits,
		"generated_count": rs.generatedCount,
	}

	if rs.lastRunTime != nil {
		status["last_run_time"] = rs.lastRunTime.Format(time.RFC3339)
	}

	if rs.nextRunTime != nil {
		status["next_run_time"] = rs.nextRunTime.Format(time.RFC3339)
	}

	return status
}

// UpdateConfig 更新配置
func (rs *RecommendationScheduler) UpdateConfig(interval time.Duration, maxAge time.Duration, kinds []string, limits []int) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.isRunning {
		return fmt.Errorf("cannot update config while running")
	}

	rs.interval = interval
	rs.maxAge = maxAge
	rs.kinds = kinds
	rs.limits = limits

	log.Printf("[RecommendationScheduler] Config updated: interval=%v, maxAge=%v, kinds=%v, limits=%v",
		interval, maxAge, kinds, limits)

	return nil
}

// ForceGenerate 强制生成推荐（用于手动触发）
func (rs *RecommendationScheduler) ForceGenerate(ctx context.Context, kind string, limit int) error {
	log.Printf("[RecommendationScheduler] Force generating recommendations (kind=%s, limit=%d)", kind, limit)

	return rs.generateRecommendationBatch(ctx, kind, limit)
}

// CleanupOldRecommendations 清理非常旧的推荐（保留历史数据）
func (rs *RecommendationScheduler) CleanupOldRecommendations(maxAge time.Duration) error {
	// 为了保留历史数据作为参考，我们采用非常保守的清理策略
	// 只清理超过1年的数据，或者明确指定要清理的数据

	// 如果传入的时间超过30天，我们假设用户明确要求清理
	// 否则，我们只清理超过1年的数据
	actualMaxAge := maxAge
	if maxAge < 30*24*time.Hour {
		actualMaxAge = 365 * 24 * time.Hour // 1年
		log.Printf("[RecommendationScheduler] 使用保守清理策略，只清理超过1年的数据 (maxAge=%v)", maxAge)
	}

	cutoffTime := time.Now().UTC().Add(-actualMaxAge)

	// 获取将要删除的记录数量
	var count int64
	rs.server.db.DB().Model(&pdb.CoinRecommendation{}).Where("created_at < ?", cutoffTime).Count(&count)

	if count == 0 {
		log.Printf("[RecommendationScheduler] 没有找到需要清理的推荐数据")
		return nil
	}

	log.Printf("[RecommendationScheduler] 将清理 %d 条推荐记录 (早于 %s)",
		count, cutoffTime.Format("2006-01-02 15:04:05"))

	// 为了安全起见，我们不直接删除，而是记录警告
	// 如果真的需要清理，可以取消下面的注释
	/*
		result := rs.server.db.DB().Where("created_at < ?", cutoffTime).Delete(&pdb.CoinRecommendation{})
		if result.Error != nil {
			return result.Error
		}

		log.Printf("[RecommendationScheduler] 已清理 %d 条旧推荐记录", result.RowsAffected)
	*/

	log.Printf("[RecommendationScheduler] 推荐清理已禁用，以保留历史数据供参考。如需清理，请手动执行SQL")

	return nil
}

// GetRecommendationStats 获取推荐数据统计
func (rs *RecommendationScheduler) GetRecommendationStats() map[string]interface{} {
	var totalCount int64
	var recentCount int64 // 最近7天
	var monthCount int64  // 最近30天
	var oldCount int64    // 超过1年的

	now := time.Now().UTC()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	monthAgo := now.Add(-30 * 24 * time.Hour)
	yearAgo := now.Add(-365 * 24 * time.Hour)

	rs.server.db.DB().Model(&pdb.CoinRecommendation{}).Count(&totalCount)
	rs.server.db.DB().Model(&pdb.CoinRecommendation{}).Where("created_at >= ?", weekAgo).Count(&recentCount)
	rs.server.db.DB().Model(&pdb.CoinRecommendation{}).Where("created_at >= ?", monthAgo).Count(&monthCount)
	rs.server.db.DB().Model(&pdb.CoinRecommendation{}).Where("created_at < ?", yearAgo).Count(&oldCount)

	return map[string]interface{}{
		"total_recommendations": totalCount,
		"recent_7_days":         recentCount,
		"recent_30_days":        monthCount,
		"older_than_1_year":     oldCount,
		"data_retention_policy": "保留所有历史数据",
		"cleanup_disabled":      true,
	}
}
