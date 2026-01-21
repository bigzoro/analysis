package selection

import (
	"analysis/internal/server/strategy/traditional"
	"context"
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

// Selector 候选选择器实现
type Selector struct {
	db pdb.Database // 数据库连接
}

// NewSelector 创建候选选择器
func NewSelector(db pdb.Database) traditional.CandidateSelector {
	return &Selector{db: db}
}

// SelectGainersWithRank 选择涨幅榜并计算排名
// marketType: "spot" - 只选择现货, "futures" - 只选择合约, "both" - 选择同时有现货和合约的
func (s *Selector) SelectGainersWithRank(ctx context.Context, limit int, marketType string) ([]traditional.CandidateWithRank, error) {
	// 检查数据库连接
	if s.db == nil {
		log.Printf("[Selector] 数据库连接不可用")
		return nil, fmt.Errorf("数据库连接不可用")
	}

	// 从 realtime_gainers_items 表获取涨幅榜数据（前15名实时数据）
	// 根据marketType参数选择不同的查询逻辑
	var query string
	switch marketType {
	case "futures":
		// 只选择合约交易的币种
		query = `
			SELECT
				i.symbol,
				i.current_price as last_price,
				i.price_change_percent,
				i.volume24h as volume,
				i.rank as ranking
			FROM realtime_gainers_items i
			INNER JOIN realtime_gainers_snapshots s ON i.snapshot_id = s.id
			WHERE s.kind = 'futures'
				AND s.id = (
					SELECT MAX(id) FROM realtime_gainers_snapshots
					WHERE kind = 'futures'
					AND timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
				)
				AND i.price_change_percent > 0
				AND i.volume24h > 0
			ORDER BY i.rank ASC
			LIMIT ?
		`
	case "both":
		// 选择同时有spot和futures数据的币种
		query = `
			SELECT
				i.symbol,
				i.current_price as last_price,
				i.price_change_percent,
				i.volume24h as volume,
				i.rank as ranking
			FROM realtime_gainers_items i
			INNER JOIN realtime_gainers_snapshots s ON i.snapshot_id = s.id
			WHERE s.kind = 'spot'
				AND s.id = (
					SELECT MAX(id) FROM realtime_gainers_snapshots
					WHERE kind = 'spot'
					AND timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
				)
				AND i.price_change_percent > 0
				AND i.volume24h > 0
				AND i.symbol IN (
					SELECT DISTINCT fi.symbol
					FROM realtime_gainers_items fi
					INNER JOIN realtime_gainers_snapshots fs ON fi.snapshot_id = fs.id
					WHERE fs.id = (
						SELECT MAX(id) FROM realtime_gainers_snapshots
						WHERE kind = 'futures'
						AND timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
					)
				)
			ORDER BY i.rank ASC
			LIMIT ?
		`
	default: // "spot"
		// 只选择现货交易的币种
		query = `
			SELECT
				i.symbol,
				i.current_price as last_price,
				i.price_change_percent,
				i.volume24h as volume,
				i.rank as ranking
			FROM realtime_gainers_items i
			INNER JOIN realtime_gainers_snapshots s ON i.snapshot_id = s.id
			WHERE s.kind = 'spot'
				AND s.id = (
					SELECT MAX(id) FROM realtime_gainers_snapshots
					WHERE kind = 'spot'
					AND timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
				)
				AND i.price_change_percent > 0
				AND i.volume24h > 0
			ORDER BY i.rank ASC
			LIMIT ?
		`
	}

	var results []struct {
		Symbol             string  `json:"symbol"`
		LastPrice          float64 `json:"last_price"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		Ranking            int     `json:"rank"`
	}

	db, err := s.db.DB()
	if err != nil {
		log.Printf("[Selector] 获取数据库连接失败: %v", err)
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	err = db.Raw(query, limit).Scan(&results).Error
	if err != nil {
		log.Printf("[Selector] 查询涨幅榜数据失败: %v", err)
		return nil, fmt.Errorf("查询涨幅榜数据失败: %w", err)
	}

	if len(results) == 0 {
		log.Printf("[Selector] 没有找到涨幅榜数据")
		return nil, fmt.Errorf("没有找到涨幅榜数据")
	}

	// 转换为CandidateWithRank格式
	candidates := make([]traditional.CandidateWithRank, 0, len(results))
	for _, item := range results {
		// 估算市值（简化计算）
		marketCap := item.Volume * 10 // 粗略估算

		candidate := traditional.CandidateWithRank{
			Symbol:        item.Symbol,
			Rank:          item.Ranking,
			Price:         item.LastPrice,
			ChangePercent: item.PriceChangePercent,
			Volume:        item.Volume,
			MarketCap:     marketCap,
		}
		candidates = append(candidates, candidate)
	}

	log.Printf("[Selector] 成功获取%d个真实涨幅榜候选", len(candidates))
	return candidates, nil
}

// SelectSmallGainersWithRank 选择小幅上涨币种并计算排名
func (s *Selector) SelectSmallGainersWithRank(ctx context.Context, limit int) ([]traditional.CandidateWithRank, error) {
	// 检查数据库连接
	if s.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}

	// 从数据库获取涨幅在1%-5%之间的币种（小幅上涨）
	query := `
		SELECT
			symbol,
			last_price,
			price_change_percent,
			volume,
			ROW_NUMBER() OVER (ORDER BY price_change_percent DESC, volume DESC) as ranking
		FROM binance_24h_stats
		WHERE market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			AND volume > 0 AND last_price > 0
			AND price_change_percent BETWEEN 1.0 AND 5.0  -- 小幅上涨：1%-5%
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT ?
	`

	var results []struct {
		Symbol             string  `json:"symbol"`
		LastPrice          float64 `json:"last_price"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		Ranking            int     `json:"rank"`
	}

	db, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	err = db.Raw(query, limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询小幅上涨数据失败: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("没有找到小幅上涨数据")
	}

	// 转换为CandidateWithRank格式
	candidates := make([]traditional.CandidateWithRank, 0, len(results))
	for _, item := range results {
		// 估算市值（简化计算）
		marketCap := item.Volume * 10 // 粗略估算

		candidate := traditional.CandidateWithRank{
			Symbol:        item.Symbol,
			Rank:          item.Ranking,
			Price:         item.LastPrice,
			ChangePercent: item.PriceChangePercent,
			Volume:        item.Volume,
			MarketCap:     marketCap,
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// SelectByVolume 按交易量选择候选
func (s *Selector) SelectByVolume(ctx context.Context, limit int) ([]traditional.CandidateWithRank, error) {
	// 检查数据库连接
	if s.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}

	// 从数据库获取交易量最大的币种
	query := `
		SELECT
			symbol,
			last_price,
			price_change_percent,
			volume,
			ROW_NUMBER() OVER (ORDER BY volume DESC) as ranking
		FROM binance_24h_stats
		WHERE market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			AND volume > 0 AND last_price > 0
		ORDER BY volume DESC
		LIMIT ?
	`

	var results []struct {
		Symbol             string  `json:"symbol"`
		LastPrice          float64 `json:"last_price"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		Ranking            int     `json:"rank"`
	}

	db, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	err = db.Raw(query, limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询交易量数据失败: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("没有找到交易量数据")
	}

	// 转换为CandidateWithRank格式
	candidates := make([]traditional.CandidateWithRank, 0, len(results))
	for _, item := range results {
		// 估算市值（简化计算）
		marketCap := item.Volume * 10 // 粗略估算

		candidate := traditional.CandidateWithRank{
			Symbol:        item.Symbol,
			Rank:          item.Ranking,
			Price:         item.LastPrice,
			ChangePercent: item.PriceChangePercent,
			Volume:        item.Volume,
			MarketCap:     marketCap,
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}
