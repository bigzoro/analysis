package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitialGainersPopulator 涨幅榜初始化数据填充器
// 功能：系统启动时提供初始涨幅榜数据，确保数据库中有基础数据可用
// 与实时同步器(realtime_gainers_syncer)配合使用，提供完整的数据生态
//
// 注意：data_source字段使用"init_populate"标识，避免超出数据库字段长度限制
type InitialGainersPopulator struct {
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	// 统计信息
	stats struct {
		totalPopulations int64         // 总填充次数
		lastPopulation   time.Time     // 最后填充时间
		dataPopulated    int64         // 填充的数据条数
		populationTime   time.Duration // 平均填充时间
	}
}

// NewInitialGainersPopulator 创建初始化数据填充器
func NewInitialGainersPopulator(db *gorm.DB, cfg *config.Config, config *DataSyncConfig) *InitialGainersPopulator {
	return &InitialGainersPopulator{
		db:     db,
		cfg:    cfg,
		config: config,
	}
}

// PopulateInitialData 填充初始涨幅榜数据
// 这个方法在系统启动时调用，确保数据库中有初始数据
func (p *InitialGainersPopulator) PopulateInitialData(ctx context.Context) error {
	log.Printf("[InitialGainersPopulator] 开始填充初始涨幅榜数据...")

	startTime := time.Now()

	// 检查是否需要填充数据
	if !p.shouldPopulateData() {
		log.Printf("[InitialGainersPopulator] 数据已存在，跳过填充")
		return nil
	}

	var totalRecords int64
	var hasErrors bool

	// 填充现货涨幅榜数据
	if err := p.populateMarketData(ctx, "spot"); err != nil {
		log.Printf("[InitialGainersPopulator] 填充现货数据失败: %v", err)
		hasErrors = true
	} else {
		records, _ := p.getPopulationRecords("spot")
		totalRecords += records
	}

	// 填充期货涨幅榜数据
	if err := p.populateMarketData(ctx, "futures"); err != nil {
		log.Printf("[InitialGainersPopulator] 填充期货数据失败: %v", err)
		hasErrors = true
	} else {
		records, _ := p.getPopulationRecords("futures")
		totalRecords += records
	}

	populationDuration := time.Since(startTime)

	// 更新统计信息
	p.updateStats(totalRecords, populationDuration)

	log.Printf("[InitialGainersPopulator] 初始数据填充完成 - 耗时:%v, 数据条数:%d",
		populationDuration, totalRecords)

	if hasErrors {
		return fmt.Errorf("部分数据填充失败，请检查日志")
	}

	return nil
}

// shouldPopulateData 检查是否需要填充数据
func (p *InitialGainersPopulator) shouldPopulateData() bool {
	// 检查realtime_gainers_items表是否有最近的数据
	var count int64
	oneHourAgo := time.Now().Add(-time.Hour)

	err := p.db.Model(&pdb.RealtimeGainersItem{}).
		Where("created_at > ?", oneHourAgo).
		Count(&count).Error

	if err != nil {
		log.Printf("[InitialGainersPopulator] 检查现有数据失败: %v", err)
		return true // 出错时仍尝试填充
	}

	// 如果数据量太少，需要填充
	return count < 50 // 少于50条数据时需要填充
}

// populateMarketData 填充指定市场的涨幅榜数据
func (p *InitialGainersPopulator) populateMarketData(ctx context.Context, kind string) error {
	log.Printf("[InitialGainersPopulator] 填充%s市场数据...", kind)

	// 从binance_24h_stats表获取涨幅榜数据
	gainersData, err := p.generateGainersFrom24hStats(kind, 50) // 获取前50个
	if err != nil {
		return fmt.Errorf("生成%s涨幅榜数据失败: %w", kind, err)
	}

	if len(gainersData) == 0 {
		log.Printf("[InitialGainersPopulator] %s市场无有效数据", kind)
		return nil
	}

	// 保存到数据库
	err = p.saveGainersDataToDB(kind, gainersData)
	if err != nil {
		return fmt.Errorf("保存%s涨幅榜数据失败: %w", kind, err)
	}

	log.Printf("[InitialGainersPopulator] %s市场数据填充完成: %d条", kind, len(gainersData))
	return nil
}

// generateGainersFrom24hStats 从binance_24h_stats生成涨幅榜数据
func (p *InitialGainersPopulator) generateGainersFrom24hStats(kind string, limit int) ([]gin.H, error) {
	// 为每个币种选择最新的一条记录，然后按涨幅排序
	query := fmt.Sprintf(`
		SELECT
			symbol,
			price_change_percent,
			volume,
			quote_volume,
			last_price,
			ROW_NUMBER() OVER (ORDER BY price_change_percent DESC, volume DESC) as ranking
		FROM binance_24h_stats
		WHERE (market_type, symbol, created_at) IN (
			-- 为每个币种选择最新的一条记录
			SELECT market_type, symbol, MAX(created_at) as latest_time
			FROM binance_24h_stats
			WHERE market_type = ?
			  AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			  AND volume > 0
			  AND last_price > 0
			GROUP BY market_type, symbol
		)
		AND market_type = ?
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT %d
	`, limit)

	var results []struct {
		Symbol             string  `json:"symbol"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		QuoteVolume        float64 `json:"quote_volume"`
		LastPrice          float64 `json:"last_price"`
		Ranking            int     `json:"ranking"`
	}

	err := p.db.Raw(query, kind, kind).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询涨幅榜数据失败: %w", err)
	}

	// 转换为gin.H格式
	gainers := make([]gin.H, 0, len(results))
	for _, result := range results {
		gainer := gin.H{
			"symbol":               result.Symbol,
			"current_price":        result.LastPrice,
			"price_change_24h":     result.PriceChangePercent,
			"price_change_percent": result.PriceChangePercent,
			"volume_24h":           result.Volume,
			"quote_volume":         result.QuoteVolume,
			"rank":                 result.Ranking,
			"data_source":          "init_populate",
			"timestamp":            time.Now().Unix(),
		}
		gainers = append(gainers, gainer)
	}

	return gainers, nil
}

// saveGainersDataToDB 保存涨幅榜数据到数据库
func (p *InitialGainersPopulator) saveGainersDataToDB(kind string, gainers []gin.H) error {
	if len(gainers) == 0 {
		return nil
	}

	// 转换为数据库结构
	items := make([]pdb.RealtimeGainersItem, 0, len(gainers))
	for i, gainer := range gainers {
		rank := i + 1
		if r, ok := gainer["rank"].(int); ok && r > 0 {
			rank = r
		}

		item := pdb.RealtimeGainersItem{
			Symbol:         gainer["symbol"].(string),
			Rank:           rank,
			CurrentPrice:   gainer["current_price"].(float64),
			PriceChange24h: gainer["price_change_24h"].(float64),
			Volume24h:      gainer["volume_24h"].(float64),
			DataSource:     gainer["data_source"].(string),
		}

		// 可选字段
		if pc, ok := gainer["price_change_percent"].(float64); ok {
			item.PriceChangePercent = &pc
		}

		items = append(items, item)
	}

	// 使用事务保存
	err := p.db.Transaction(func(tx *gorm.DB) error {
		// 创建快照
		snapshot := &pdb.RealtimeGainersSnapshot{
			Kind:      kind,
			Timestamp: time.Now(),
		}

		if err := tx.Create(snapshot).Error; err != nil {
			return fmt.Errorf("创建快照失败: %w", err)
		}

		// 批量保存数据
		for i := range items {
			items[i].SnapshotID = snapshot.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return fmt.Errorf("保存数据失败: %w", err)
			}
		}

		return nil
	})

	return err
}

// getPopulationRecords 获取填充的记录数
func (p *InitialGainersPopulator) getPopulationRecords(kind string) (int64, error) {
	var count int64
	err := p.db.Model(&pdb.RealtimeGainersItem{}).
		Joins("JOIN realtime_gainers_snapshots s ON realtime_gainers_items.snapshot_id = s.id").
		Where("s.kind = ?", kind).
		Count(&count).Error

	return count, err
}

// updateStats 更新统计信息
func (p *InitialGainersPopulator) updateStats(records int64, duration time.Duration) {
	p.stats.totalPopulations++
	p.stats.lastPopulation = time.Now()
	p.stats.dataPopulated += records

	// 计算平均填充时间
	if p.stats.totalPopulations == 1 {
		p.stats.populationTime = duration
	} else {
		p.stats.populationTime = (p.stats.populationTime + duration) / 2
	}
}

// getInternalStats 获取内部统计信息
func (p *InitialGainersPopulator) getInternalStats() map[string]interface{} {
	return map[string]interface{}{
		"total_populations":   p.stats.totalPopulations,
		"last_population":     p.stats.lastPopulation,
		"data_populated":      p.stats.dataPopulated,
		"avg_population_time": p.stats.populationTime.String(),
	}
}

// CleanupOldData 清理旧的初始化数据
// 只保留最近的初始化快照，避免数据积累过多
func (p *InitialGainersPopulator) CleanupOldData() error {
	log.Printf("[InitialGainersPopulator] 清理旧的初始化数据...")

	// 只保留最近24小时的初始化数据快照
	cutoffTime := time.Now().Add(-24 * time.Hour)

	// 删除旧快照（注意：只删除data_source为"init_populate"的数据）
	err := p.db.Exec(`
		DELETE FROM realtime_gainers_snapshots
		WHERE id IN (
			SELECT DISTINCT snapshot_id
			FROM realtime_gainers_items
			WHERE data_source = 'init_populate'
			  AND created_at < ?
		)
	`, cutoffTime).Error

	if err != nil {
		return fmt.Errorf("清理旧初始化数据失败: %w", err)
	}

	log.Printf("[InitialGainersPopulator] 旧初始化数据清理完成")
	return nil
}

// ===== DataSyncer 接口实现 =====

// Name 返回同步器名称
func (p *InitialGainersPopulator) Name() string {
	return "initial_gainers"
}

// Start 启动同步器（DataSyncer接口）
func (p *InitialGainersPopulator) Start(ctx context.Context, interval time.Duration) {
	log.Printf("[InitialGainersPopulator] 初始化填充器启动 - 执行一次性数据填充")

	// 执行一次性数据填充
	if err := p.PopulateInitialData(ctx); err != nil {
		log.Printf("[InitialGainersPopulator] 初始化数据填充失败: %v", err)
	}

	// 清理旧数据
	if err := p.CleanupOldData(); err != nil {
		log.Printf("[InitialGainersPopulator] 清理旧数据失败: %v", err)
	}
}

// Stop 停止同步器（DataSyncer接口）
func (p *InitialGainersPopulator) Stop() {
	log.Printf("[InitialGainersPopulator] 初始化填充器停止")
}

// Sync 执行一次性同步（DataSyncer接口）
func (p *InitialGainersPopulator) Sync(ctx context.Context) error {
	log.Printf("[InitialGainersPopulator] 执行手动初始化数据填充")
	return p.PopulateInitialData(ctx)
}

// GetStats 获取统计信息（DataSyncer接口）
func (p *InitialGainersPopulator) GetStats() map[string]interface{} {
	return p.getInternalStats()
}
