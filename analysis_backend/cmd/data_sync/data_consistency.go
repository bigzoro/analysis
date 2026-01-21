package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// DataConsistencyChecker 数据一致性检查器
type DataConsistencyChecker struct {
	db        *gorm.DB
	websocket *WebSocketSyncer
	kline     *KlineSyncer
	depth     *DepthSyncer
	price     *PriceSyncer

	// 检查配置
	checkInterval     time.Duration
	consistencyWindow time.Duration // 检查时间窗口（前后多少分钟的数据）
	maxDataAge        time.Duration // 允许的最大数据年龄

	// 统计信息
	stats struct {
		mu                      sync.RWMutex
		totalChecks             int64
		consistencyIssues       int64
		lastConsistencyCheck    time.Time
		averageConsistencyScore float64
		recentInconsistencies   []ConsistencyIssue
	}

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// ConsistencyIssue 数据不一致问题
type ConsistencyIssue struct {
	Timestamp   time.Time
	DataType    string // "price", "kline", "depth"
	Symbol      string
	Description string
	Severity    string // "low", "medium", "high", "critical"
	Resolved    bool
}

// NewDataConsistencyChecker 创建数据一致性检查器
func NewDataConsistencyChecker(
	db *gorm.DB,
	websocket *WebSocketSyncer,
	kline *KlineSyncer,
	depth *DepthSyncer,
	price *PriceSyncer,
) *DataConsistencyChecker {

	ctx, cancel := context.WithCancel(context.Background())

	return &DataConsistencyChecker{
		db:        db,
		websocket: websocket,
		kline:     kline,
		depth:     depth,
		price:     price,

		checkInterval:     5 * time.Minute,  // 默认值，后续可从配置读取
		consistencyWindow: 30 * time.Minute, // 默认值，后续可从配置读取
		maxDataAge:        10 * time.Minute, // 默认值，后续可从配置读取

		ctx:    ctx,
		cancel: cancel,
	}
}

// NewDataConsistencyCheckerWithConfig 使用配置创建数据一致性检查器
func NewDataConsistencyCheckerWithConfig(
	db *gorm.DB,
	websocket *WebSocketSyncer,
	kline *KlineSyncer,
	depth *DepthSyncer,
	price *PriceSyncer,
	config *DataSyncConfig,
) *DataConsistencyChecker {

	ctx, cancel := context.WithCancel(context.Background())

	return &DataConsistencyChecker{
		db:        db,
		websocket: websocket,
		kline:     kline,
		depth:     depth,
		price:     price,

		checkInterval:     time.Duration(config.Timeouts.ConsistencyCheckInterval) * time.Second,
		consistencyWindow: 30 * time.Minute, // 可以后续配置化
		maxDataAge:        time.Duration(config.Timeouts.DataAgeMax) * time.Second,

		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动数据一致性检查
func (c *DataConsistencyChecker) Start() {
	log.Printf("[DataConsistencyChecker] Starting data consistency checker...")

	go c.consistencyCheckLoop()

	log.Printf("[DataConsistencyChecker] Data consistency checker started")
}

// Stop 停止数据一致性检查
func (c *DataConsistencyChecker) Stop() {
	c.cancel()
	log.Printf("[DataConsistencyChecker] Stopped")
}

// consistencyCheckLoop 一致性检查循环
func (c *DataConsistencyChecker) consistencyCheckLoop() {
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.performConsistencyCheck()
		}
	}
}

// performConsistencyCheck 执行一致性检查
func (c *DataConsistencyChecker) performConsistencyCheck() {
	log.Printf("[DataConsistencyChecker] Performing data consistency check...")

	c.stats.mu.Lock()
	c.stats.totalChecks++
	startTime := time.Now()
	c.stats.lastConsistencyCheck = startTime
	c.stats.mu.Unlock()

	issues := []ConsistencyIssue{}

	// 检查价格数据一致性
	if priceIssues := c.checkPriceConsistency(); len(priceIssues) > 0 {
		issues = append(issues, priceIssues...)
	}

	// 检查K线数据一致性
	if klineIssues := c.checkKlineConsistency(); len(klineIssues) > 0 {
		issues = append(issues, klineIssues...)
	}

	// 检查深度数据一致性
	if depthIssues := c.checkDepthConsistency(); len(depthIssues) > 0 {
		issues = append(issues, depthIssues...)
	}

	// 检查数据时效性
	if ageIssues := c.checkDataTimeliness(); len(ageIssues) > 0 {
		issues = append(issues, ageIssues...)
	}

	// 更新统计信息
	c.stats.mu.Lock()
	c.stats.consistencyIssues += int64(len(issues))

	// 保持最近的10个不一致问题
	c.stats.recentInconsistencies = append(c.stats.recentInconsistencies, issues...)
	if len(c.stats.recentInconsistencies) > 10 {
		c.stats.recentInconsistencies = c.stats.recentInconsistencies[len(c.stats.recentInconsistencies)-10:]
	}

	// 计算一致性得分（0-100，100为完全一致）
	totalPossibleIssues := 100 // 假设的基准
	actualIssues := len(issues)
	consistencyScore := float64(totalPossibleIssues-actualIssues) / float64(totalPossibleIssues) * 100
	if consistencyScore < 0 {
		consistencyScore = 0
	}

	// 更新移动平均一致性得分
	if c.stats.averageConsistencyScore == 0 {
		c.stats.averageConsistencyScore = consistencyScore
	} else {
		c.stats.averageConsistencyScore = (c.stats.averageConsistencyScore + consistencyScore) / 2
	}
	c.stats.mu.Unlock()

	duration := time.Since(startTime)

	if len(issues) > 0 {
		log.Printf("[DataConsistencyChecker] ❌ Found %d consistency issues in %v", len(issues), duration)
		for _, issue := range issues {
			log.Printf("[DataConsistencyChecker]   %s: %s", issue.Severity, issue.Description)
		}
	} else {
		log.Printf("[DataConsistencyChecker] ✅ All data consistent (checked in %v)", duration)
	}
}

// checkPriceConsistency 检查价格数据一致性
func (c *DataConsistencyChecker) checkPriceConsistency() []ConsistencyIssue {
	issues := []ConsistencyIssue{}

	// 检查WebSocket和数据库的价格数据是否同步
	// 这里可以比较最近的价格数据

	// 示例：检查是否有价格数据缺失
	now := time.Now()
	cutoff := now.Add(-c.consistencyWindow)

	var priceCount int64
	c.db.Model(&pdb.PriceCache{}).Where("last_updated > ?", cutoff).Count(&priceCount)

	if priceCount == 0 {
		issues = append(issues, ConsistencyIssue{
			Timestamp:   now,
			DataType:    "price",
			Description: "No recent price data found in database",
			Severity:    "critical",
		})
	}

	return issues
}

// checkKlineConsistency 检查K线数据一致性
func (c *DataConsistencyChecker) checkKlineConsistency() []ConsistencyIssue {
	issues := []ConsistencyIssue{}

	// 检查不同时间间隔的K线数据是否连续
	now := time.Now()
	cutoff := now.Add(-c.consistencyWindow)

	// 检查主要时间间隔的K线数据
	intervals := []string{"1m", "5m", "1h"}
	for _, interval := range intervals {
		var klineCount int64
		c.db.Model(&pdb.MarketKline{}).Where("`interval` = ? AND open_time > ?", interval, cutoff).Count(&klineCount)

		if klineCount == 0 {
			issues = append(issues, ConsistencyIssue{
				Timestamp:   now,
				DataType:    "kline",
				Description: fmt.Sprintf("No %s kline data found in recent %v", interval, c.consistencyWindow),
				Severity:    "high",
			})
		}
	}

	// 检查K线数据的时间连续性
	// 这里可以检查是否有时间间隔过大的K线数据

	return issues
}

// checkDepthConsistency 检查深度数据一致性
func (c *DataConsistencyChecker) checkDepthConsistency() []ConsistencyIssue {
	issues := []ConsistencyIssue{}

	// 检查深度数据的更新频率
	now := time.Now()
	cutoff := now.Add(-c.consistencyWindow)

	var depthCount int64
	c.db.Model(&pdb.BinanceOrderBookDepth{}).Where("snapshot_time > ?", cutoff.UnixMilli()).Count(&depthCount)

	if depthCount == 0 {
		issues = append(issues, ConsistencyIssue{
			Timestamp:   now,
			DataType:    "depth",
			Description: "No recent depth data found in database",
			Severity:    "high",
		})
	}

	return issues
}

// checkDataTimeliness 检查数据时效性
func (c *DataConsistencyChecker) checkDataTimeliness() []ConsistencyIssue {
	issues := []ConsistencyIssue{}

	now := time.Now()
	maxAge := c.maxDataAge

	// 检查价格数据的时效性
	var latestPrice pdb.PriceCache
	if err := c.db.Order("last_updated DESC").First(&latestPrice).Error; err == nil {
		if now.Sub(latestPrice.LastUpdated) > maxAge {
			issues = append(issues, ConsistencyIssue{
				Timestamp: now,
				DataType:  "price",
				Symbol:    latestPrice.Symbol,
				Description: fmt.Sprintf("Price data is %v old (max allowed: %v)",
					now.Sub(latestPrice.LastUpdated), maxAge),
				Severity: "medium",
			})
		}
	}

	// 检查K线数据的时效性
	var latestKline pdb.MarketKline
	if err := c.db.Order("open_time DESC").First(&latestKline).Error; err == nil {
		if now.Sub(latestKline.OpenTime) > maxAge {
			issues = append(issues, ConsistencyIssue{
				Timestamp: now,
				DataType:  "kline",
				Symbol:    latestKline.Symbol,
				Description: fmt.Sprintf("Kline data is %v old (max allowed: %v)",
					now.Sub(latestKline.OpenTime), maxAge),
				Severity: "medium",
			})
		}
	}

	return issues
}

// GetStats 获取一致性检查统计信息
func (c *DataConsistencyChecker) GetStats() map[string]interface{} {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	recentIssues := make([]map[string]interface{}, 0, len(c.stats.recentInconsistencies))
	for _, issue := range c.stats.recentInconsistencies {
		recentIssues = append(recentIssues, map[string]interface{}{
			"timestamp":   issue.Timestamp,
			"data_type":   issue.DataType,
			"symbol":      issue.Symbol,
			"description": issue.Description,
			"severity":    issue.Severity,
			"resolved":    issue.Resolved,
		})
	}

	return map[string]interface{}{
		"total_checks":              c.stats.totalChecks,
		"consistency_issues":        c.stats.consistencyIssues,
		"last_consistency_check":    c.stats.lastConsistencyCheck,
		"average_consistency_score": fmt.Sprintf("%.1f%%", c.stats.averageConsistencyScore),
		"recent_inconsistencies":    recentIssues,
		"check_interval":            c.checkInterval.String(),
		"consistency_window":        c.consistencyWindow.String(),
		"max_data_age":              c.maxDataAge.String(),
	}
}

// GetConsistencyScore 获取当前一致性得分
func (c *DataConsistencyChecker) GetConsistencyScore() float64 {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return c.stats.averageConsistencyScore
}

// ResolveIssue 标记问题为已解决
func (c *DataConsistencyChecker) ResolveIssue(index int) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	if index >= 0 && index < len(c.stats.recentInconsistencies) {
		c.stats.recentInconsistencies[index].Resolved = true
	}
}
