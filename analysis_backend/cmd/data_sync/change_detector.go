package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"
)

// ===== 变化检测器 =====
// 检测涨幅榜变化是否显著，避免频繁无效的数据库保存

// ChangeRecord 变化记录
type ChangeRecord struct {
	Timestamp          time.Time
	PriceChange        float64 // 价格变化百分比
	PriceChangePercent float64 // 涨跌幅变化百分比
	RankChanges        int     // 排名变化数量
	VolumeChange       float64 // 成交量变化百分比
	ShouldSave         bool    // 是否应该保存
}

// ChangeDetectionConfig 变化检测配置
type ChangeDetectionConfig struct {
	// 检测启用标志
	EnableRankDetection               bool // 是否启用排名变化检测
	EnablePriceDetection              bool // 是否启用价格变化检测
	EnablePriceChangePercentDetection bool // 是否启用涨跌幅变化检测
	EnableVolumeDetection             bool // 是否启用成交量变化检测

	// 检测阈值
	RankChangeThreshold         int     // 排名变化阈值（前N名中有多少个位置变化）
	PriceChangeThreshold        float64 // 价格变化阈值百分比
	PriceChangePercentThreshold float64 // 涨跌幅变化阈值百分比
	VolumeChangeThreshold       float64 // 成交量变化阈值百分比

	// 时间控制
	MinSaveInterval time.Duration // 最小保存间隔
	MaxSaveInterval time.Duration // 最大保存间隔
}

// ChangeDetector 变化检测器
type ChangeDetector struct {
	// 检测配置
	config *ChangeDetectionConfig

	// 智能检测配置（运行时状态）
	consecutiveChanges int       // 连续变化计数
	lastSaveTime       time.Time // 最后保存时间

	// 自适应阈值
	adaptiveThreshold float64 // 自适应价格变化阈值
	marketVolatility  float64 // 市场波动性指标

	// 状态跟踪
	lastGainers   []RealtimeGainerItem // 上次保存的涨幅榜
	changeHistory []ChangeRecord       // 变化历史记录
}

// NewChangeDetector 创建变化检测器（使用默认配置）
func NewChangeDetector() *ChangeDetector {
	return NewChangeDetectorWithConfig(&ChangeDetectionConfig{
		EnableRankDetection:               false,            // 默认关闭排名检测
		EnablePriceDetection:              false,            // 默认关闭价格检测
		EnablePriceChangePercentDetection: true,             // 默认开启涨跌幅检测
		EnableVolumeDetection:             false,            // 默认关闭成交量检测
		RankChangeThreshold:               3,                // 前15名中有3个排名变化算显著
		PriceChangeThreshold:              0.5,              // 价格变化0.5%算显著
		PriceChangePercentThreshold:       0.1,              // 涨跌幅变化0.1%算显著
		VolumeChangeThreshold:             5.0,              // 成交量变化5%算显著
		MinSaveInterval:                   30 * time.Second, // 最少30秒保存一次
		MaxSaveInterval:                   5 * time.Minute,  // 最多5分钟保存一次
	})
}

// NewChangeDetectorWithConfig 创建具有指定配置的变化检测器
func NewChangeDetectorWithConfig(config *ChangeDetectionConfig) *ChangeDetector {
	return &ChangeDetector{
		config:             config,
		consecutiveChanges: 0,
		lastSaveTime:       time.Now(),
		adaptiveThreshold:  config.PriceChangeThreshold, // 初始自适应阈值等于配置阈值
		marketVolatility:   0.0,                         // 初始市场波动性
		changeHistory:      make([]ChangeRecord, 0, 10),
	}
}

// HasSignificantChanges 检测是否有显著变化
func (d *ChangeDetector) HasSignificantChanges(currentGainers []RealtimeGainerItem) bool {
	now := time.Now()

	// 第一次运行必须保存
	if len(d.lastGainers) == 0 {
		d.recordChange(now, 0, 0, 0, 0, true)
		return true
	}

	// 检查最小保存间隔
	if now.Sub(d.lastSaveTime) < d.config.MinSaveInterval {
		return false
	}

	// 检查最大保存间隔（强制保存）
	if now.Sub(d.lastSaveTime) >= d.config.MaxSaveInterval {
		log.Printf("[ChangeDetector] 达到最大保存间隔，强制保存")
		d.recordChange(now, 0, 0, 0, 0, true)
		return true
	}

	// 根据配置检测各种变化
	var rankChanges int
	var priceChanges float64
	var priceChangePercentChanges float64
	var volumeChanges float64

	// 只在启用时进行检测
	if d.config.EnableRankDetection {
		rankChanges = d.detectRankChanges(currentGainers)
	}
	if d.config.EnablePriceDetection {
		priceChanges = d.detectPriceChanges(currentGainers)
		// 更新市场波动性（仅当价格检测启用时）
		d.updateMarketVolatility(priceChanges)
	}
	if d.config.EnablePriceChangePercentDetection {
		priceChangePercentChanges = d.detectPriceChangePercentChanges(currentGainers)
		// 更新市场波动性（涨跌幅检测也使用相同的波动性更新）
		d.updateMarketVolatility(priceChangePercentChanges)
	}
	if d.config.EnableVolumeDetection {
		volumeChanges = d.detectVolumeChanges(currentGainers)
	}

	// 计算自适应阈值（当价格检测或涨跌幅检测启用时）
	var adaptiveThreshold float64
	if d.config.EnablePriceDetection || d.config.EnablePriceChangePercentDetection {
		adaptiveThreshold = d.calculateAdaptiveThreshold()
	}

	// 多维度变化检测
	hasSignificantChanges := d.evaluateChanges(priceChanges, priceChangePercentChanges, rankChanges, volumeChanges, adaptiveThreshold)

	// 记录变化
	d.recordChange(now, priceChanges, priceChangePercentChanges, rankChanges, volumeChanges, hasSignificantChanges)

	if hasSignificantChanges {
		// 构建日志消息
		var changeDetails []string
		if d.config.EnablePriceDetection {
			changeDetails = append(changeDetails, fmt.Sprintf("价格:%.4f%%(阈值:%.4f%%)", priceChanges, adaptiveThreshold))
		}
		if d.config.EnableRankDetection {
			changeDetails = append(changeDetails, fmt.Sprintf("排名:%d", rankChanges))
		}
		if d.config.EnableVolumeDetection {
			changeDetails = append(changeDetails, fmt.Sprintf("成交量:%.2f%%", volumeChanges))
		}

		log.Printf("[ChangeDetector] 检测到显著变化 - %s", strings.Join(changeDetails, ", "))
		d.consecutiveChanges = 0
	} else {
		d.consecutiveChanges++
		// 如果连续多次没有显著变化且价格检测启用，降低阈值
		if d.consecutiveChanges > 5 && d.config.EnablePriceDetection {
			d.adaptiveThreshold = math.Max(0.1, d.adaptiveThreshold*0.9)
		}
	}

	return hasSignificantChanges
}

// detectRankChanges 检测排名变化
func (d *ChangeDetector) detectRankChanges(currentGainers []RealtimeGainerItem) int {
	if len(currentGainers) == 0 || len(d.lastGainers) == 0 {
		return 0
	}

	// 创建上次的排名映射：symbol -> rank
	lastRanks := make(map[string]int)
	for _, gainer := range d.lastGainers {
		lastRanks[gainer.Symbol] = gainer.Rank
	}

	changes := 0
	for _, current := range currentGainers {
		if lastRank, exists := lastRanks[current.Symbol]; exists {
			// 计算排名变化
			rankDiff := int(math.Abs(float64(current.Rank - lastRank)))
			if rankDiff > 0 {
				changes++
			}
		} else {
			// 新出现的交易对，算作变化
			changes++
		}
	}

	return changes
}

// detectPriceChanges 检测价格变化
func (d *ChangeDetector) detectPriceChanges(currentGainers []RealtimeGainerItem) float64 {
	if len(currentGainers) == 0 || len(d.lastGainers) == 0 {
		return 0
	}

	// 创建上次的symbol映射：symbol -> price
	lastPrices := make(map[string]float64)
	for _, gainer := range d.lastGainers {
		lastPrices[gainer.Symbol] = gainer.CurrentPrice
	}

	totalChangePercent := 0.0
	changeCount := 0

	for _, current := range currentGainers {
		if lastPrice, exists := lastPrices[current.Symbol]; exists && lastPrice > 0 {
			// 计算价格变化百分比
			changePercent := math.Abs((current.CurrentPrice - lastPrice) / lastPrice * 100)
			totalChangePercent += changePercent
			changeCount++
		}
	}

	if changeCount == 0 {
		return 0
	}

	// 返回平均价格变化百分比
	return totalChangePercent / float64(changeCount)
}

// detectPriceChangePercentChanges 检测涨跌幅变化
func (d *ChangeDetector) detectPriceChangePercentChanges(currentGainers []RealtimeGainerItem) float64 {
	if len(currentGainers) == 0 || len(d.lastGainers) == 0 {
		return 0
	}

	// 创建上次的symbol映射：symbol -> priceChangePercent
	lastChangePercents := make(map[string]float64)
	for _, gainer := range d.lastGainers {
		lastChangePercents[gainer.Symbol] = gainer.ChangePercent
	}

	totalChangePercent := 0.0
	changeCount := 0

	for _, current := range currentGainers {
		if lastChangePercent, exists := lastChangePercents[current.Symbol]; exists {
			// 计算涨跌幅变化百分比（绝对值）
			changePercent := math.Abs(current.ChangePercent - lastChangePercent)
			totalChangePercent += changePercent
			changeCount++
		}
	}

	if changeCount == 0 {
		return 0
	}

	// 返回平均涨跌幅变化百分比
	return totalChangePercent / float64(changeCount)
}

// detectVolumeChanges 检测成交量变化
func (d *ChangeDetector) detectVolumeChanges(currentGainers []RealtimeGainerItem) float64 {
	if len(currentGainers) == 0 || len(d.lastGainers) == 0 {
		return 0
	}

	// 创建上次的symbol映射：symbol -> volume
	lastVolumes := make(map[string]float64)
	for _, gainer := range d.lastGainers {
		lastVolumes[gainer.Symbol] = gainer.Volume24h
	}

	totalChangePercent := 0.0
	changeCount := 0

	for _, current := range currentGainers {
		if lastVolume, exists := lastVolumes[current.Symbol]; exists && lastVolume > 0 {
			// 计算成交量变化百分比
			changePercent := math.Abs((current.Volume24h - lastVolume) / lastVolume * 100)
			totalChangePercent += changePercent
			changeCount++
		}
	}

	if changeCount == 0 {
		return 0
	}

	// 返回平均成交量变化百分比
	return totalChangePercent / float64(changeCount)
}

// UpdateLastGainers 更新最后保存的涨幅榜
func (d *ChangeDetector) UpdateLastGainers(gainers []RealtimeGainerItem) {
	// 创建深拷贝
	d.lastGainers = make([]RealtimeGainerItem, len(gainers))
	copy(d.lastGainers, gainers)

	log.Printf("[ChangeDetector] 更新最后涨幅榜: %d个交易对", len(d.lastGainers))
}

// GetLastGainers 获取最后保存的涨幅榜
func (d *ChangeDetector) GetLastGainers() []RealtimeGainerItem {
	if len(d.lastGainers) == 0 {
		return []RealtimeGainerItem{}
	}

	// 返回副本
	result := make([]RealtimeGainerItem, len(d.lastGainers))
	copy(result, d.lastGainers)
	return result
}

// CalculateTopMovers 计算涨幅最大的交易对
func (d *ChangeDetector) CalculateTopMovers(currentGainers []RealtimeGainerItem, topN int) []RealtimeGainerItem {
	if len(currentGainers) == 0 {
		return []RealtimeGainerItem{}
	}

	// 按涨跌幅降序排序
	sorted := make([]RealtimeGainerItem, len(currentGainers))
	copy(sorted, currentGainers)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ChangePercent > sorted[j].ChangePercent
	})

	// 返回前N个
	if len(sorted) > topN {
		return sorted[:topN]
	}
	return sorted
}

// CalculatePriceVolatility 计算价格波动性
func (d *ChangeDetector) CalculatePriceVolatility(currentGainers []RealtimeGainerItem) float64 {
	if len(currentGainers) == 0 || len(d.lastGainers) == 0 {
		return 0
	}

	// 计算每个交易对的价格波动
	totalVolatility := 0.0
	count := 0

	// 创建上次的symbol映射
	lastPrices := make(map[string]float64)
	for _, gainer := range d.lastGainers {
		lastPrices[gainer.Symbol] = gainer.CurrentPrice
	}

	for _, current := range currentGainers {
		if lastPrice, exists := lastPrices[current.Symbol]; exists && lastPrice > 0 {
			// 计算波动率（价格变化的绝对值）
			volatility := math.Abs((current.CurrentPrice - lastPrice) / lastPrice)
			totalVolatility += volatility
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalVolatility / float64(count)
}

// GetChangeStats 获取变化统计信息
func (d *ChangeDetector) GetChangeStats(currentGainers []RealtimeGainerItem) map[string]interface{} {
	// 根据配置决定计算哪些统计信息
	var rankChanges int
	var priceChanges float64
	var priceChangePercentChanges float64
	var volumeChanges float64
	var priceVolatility float64
	var adaptiveThreshold float64

	if d.config.EnableRankDetection {
		rankChanges = d.detectRankChanges(currentGainers)
	}
	if d.config.EnablePriceDetection {
		priceChanges = d.detectPriceChanges(currentGainers)
		priceVolatility = d.CalculatePriceVolatility(currentGainers)
		adaptiveThreshold = d.calculateAdaptiveThreshold()
	}
	if d.config.EnablePriceChangePercentDetection {
		priceChangePercentChanges = d.detectPriceChangePercentChanges(currentGainers)
		if adaptiveThreshold == 0 { // 如果价格检测没启用，但涨跌幅检测启用了
			adaptiveThreshold = d.calculateAdaptiveThreshold()
		}
	}
	if d.config.EnableVolumeDetection {
		volumeChanges = d.detectVolumeChanges(currentGainers)
	}

	hasSignificantChanges := d.evaluateChanges(priceChanges, priceChangePercentChanges, rankChanges, volumeChanges, adaptiveThreshold)

	return map[string]interface{}{
		// 变化检测结果
		"rank_changes":                 rankChanges,
		"price_changes_percent":        priceChanges,
		"price_change_percent_changes": priceChangePercentChanges,
		"volume_changes_percent":       volumeChanges,
		"price_volatility":             priceVolatility,
		"has_significant_changes":      hasSignificantChanges,

		// 配置信息
		"enable_rank_detection":                 d.config.EnableRankDetection,
		"enable_price_detection":                d.config.EnablePriceDetection,
		"enable_price_change_percent_detection": d.config.EnablePriceChangePercentDetection,
		"enable_volume_detection":               d.config.EnableVolumeDetection,
		"rank_change_threshold":                 d.config.RankChangeThreshold,
		"price_change_threshold":                d.config.PriceChangeThreshold,
		"price_change_percent_threshold":        d.config.PriceChangePercentThreshold,
		"volume_change_threshold":               d.config.VolumeChangeThreshold,
		"min_save_interval":                     d.config.MinSaveInterval.String(),
		"max_save_interval":                     d.config.MaxSaveInterval.String(),

		// 运行时状态
		"adaptive_threshold":    adaptiveThreshold,
		"market_volatility":     d.marketVolatility,
		"consecutive_changes":   d.consecutiveChanges,
		"last_save_time":        d.lastSaveTime,
		"time_since_last_save":  time.Since(d.lastSaveTime).String(),
		"change_history_count":  len(d.changeHistory),
		"last_gainers_count":    len(d.lastGainers),
		"current_gainers_count": len(currentGainers),
	}
}

// updateMarketVolatility 更新市场波动性指标
func (d *ChangeDetector) updateMarketVolatility(priceChange float64) {
	// 使用指数移动平均更新市场波动性
	if d.marketVolatility == 0 {
		d.marketVolatility = priceChange
	} else {
		d.marketVolatility = d.marketVolatility*0.8 + priceChange*0.2
	}
}

// calculateAdaptiveThreshold 计算自适应阈值
func (d *ChangeDetector) calculateAdaptiveThreshold() float64 {
	// 基于市场波动性调整阈值
	baseThreshold := d.config.PriceChangeThreshold

	// 高波动市场降低阈值（更容易触发保存）
	if d.marketVolatility > 2.0 {
		baseThreshold *= 0.5
	} else if d.marketVolatility > 1.0 {
		baseThreshold *= 0.7
	} else if d.marketVolatility < 0.2 {
		// 低波动市场提高阈值（减少不必要的保存）
		baseThreshold *= 1.5
	}

	// 确保阈值在合理范围内
	return math.Max(0.1, math.Min(2.0, baseThreshold))
}

// evaluateChanges 评估变化是否显著
func (d *ChangeDetector) evaluateChanges(priceChange float64, priceChangePercentChange float64, rankChange int, volumeChange float64, adaptiveThreshold float64) bool {
	// 根据配置检查各种变化

	// 价格变化检测
	if d.config.EnablePriceDetection && priceChange >= adaptiveThreshold {
		return true
	}

	// 涨跌幅变化检测
	if d.config.EnablePriceChangePercentDetection && priceChangePercentChange >= d.config.PriceChangePercentThreshold {
		return true
	}

	// 排名变化检测
	if d.config.EnableRankDetection && rankChange >= d.config.RankChangeThreshold {
		return true
	}

	// 大幅成交量变化
	if d.config.EnableVolumeDetection && volumeChange >= d.config.VolumeChangeThreshold {
		return true
	}

	// 检查最近的变化趋势（如果连续几次小幅变化，也触发保存）
	// 只有在至少启用了价格检测时才有意义
	if d.config.EnablePriceDetection && d.hasAccumulatedChanges() {
		return true
	}

	return false
}

// hasAccumulatedChanges 检查是否有累积的变化
func (d *ChangeDetector) hasAccumulatedChanges() bool {
	if len(d.changeHistory) < 3 {
		return false
	}

	// 检查最近3次变化的累积效应
	totalPriceChange := 0.0
	totalRankChange := 0
	for i := len(d.changeHistory) - 3; i < len(d.changeHistory); i++ {
		record := d.changeHistory[i]
		totalPriceChange += record.PriceChange
		totalRankChange += record.RankChanges
	}

	// 如果累积价格变化超过1%，或累积排名变化超过5个，触发保存
	return totalPriceChange >= 1.0 || totalRankChange >= 5
}

// recordChange 记录变化
func (d *ChangeDetector) recordChange(timestamp time.Time, priceChange float64, priceChangePercentChange float64, rankChange int, volumeChange float64, shouldSave bool) {
	record := ChangeRecord{
		Timestamp:          timestamp,
		PriceChange:        priceChange,
		PriceChangePercent: priceChangePercentChange,
		RankChanges:        rankChange,
		VolumeChange:       volumeChange,
		ShouldSave:         shouldSave,
	}

	// 保持最近10条记录
	d.changeHistory = append(d.changeHistory, record)
	if len(d.changeHistory) > 10 {
		d.changeHistory = d.changeHistory[1:]
	}

	if shouldSave {
		d.lastSaveTime = timestamp
	}
}

// SetThresholds 设置检测阈值（向后兼容）
func (d *ChangeDetector) SetThresholds(rankThreshold int, priceThreshold, volumeThreshold float64) {
	d.config.RankChangeThreshold = rankThreshold
	d.config.PriceChangeThreshold = priceThreshold
	d.config.VolumeChangeThreshold = volumeThreshold

	log.Printf("[ChangeDetector] 更新检测阈值 - 排名:%d, 价格:%.2f%%, 成交量:%.2f%%",
		rankThreshold, priceThreshold, volumeThreshold)
}

// SetDetectionConfig 设置完整的检测配置
func (d *ChangeDetector) SetDetectionConfig(config *ChangeDetectionConfig) {
	d.config = config
	log.Printf("[ChangeDetector] 更新检测配置 - 排名检测:%v, 价格检测:%v, 成交量检测:%v",
		config.EnableRankDetection, config.EnablePriceDetection, config.EnableVolumeDetection)
}

// GetDetectionConfig 获取当前检测配置
func (d *ChangeDetector) GetDetectionConfig() *ChangeDetectionConfig {
	return d.config
}

// Reset 重置检测器状态
func (d *ChangeDetector) Reset() {
	d.lastGainers = nil
	log.Printf("[ChangeDetector] 检测器状态已重置")
}
