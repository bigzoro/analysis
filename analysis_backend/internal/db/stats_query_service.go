package db

import (
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

// ==================== 智能查询服务 ====================

// QueryType 查询类型枚举
type QueryType string

const (
	QueryTypeLatest    QueryType = "latest"     // 最新数据查询
	QueryTypeTimeRange QueryType = "time_range" // 时间范围查询
	QueryTypeTechnical QueryType = "technical"  // 技术分析查询
	QueryTypeBacktest  QueryType = "backtest"   // 回测数据查询
	QueryTypeMixed     QueryType = "mixed"      // 混合查询（实时+历史）
)

// StatsQuery 查询参数结构体
type StatsQuery struct {
	// 基础查询参数
	Symbol     string    `json:"symbol"`      // 交易对符号
	MarketType string    `json:"market_type"` // 市场类型
	QueryType  QueryType `json:"query_type"`  // 查询类型

	// 时间范围参数
	StartTime *time.Time `json:"start_time,omitempty"` // 开始时间
	EndTime   *time.Time `json:"end_time,omitempty"`   // 结束时间

	// 分页参数
	Limit  int `json:"limit,omitempty"`  // 限制返回数量
	Offset int `json:"offset,omitempty"` // 偏移量

	// 排序参数
	SortBy    string `json:"sort_by,omitempty"`    // 排序字段
	SortOrder string `json:"sort_order,omitempty"` // 排序顺序 (asc/desc)

	// 高级查询参数
	PriceChangeMin *float64 `json:"price_change_min,omitempty"` // 价格变化最小值
	PriceChangeMax *float64 `json:"price_change_max,omitempty"` // 价格变化最大值
	VolumeMin      *float64 `json:"volume_min,omitempty"`       // 交易量最小值
	VolumeMax      *float64 `json:"volume_max,omitempty"`       // 交易量最大值
}

// StatsResult 查询结果结构体
type StatsResult struct {
	// 基础字段
	ID                 uint    `json:"id"`
	Symbol             string  `json:"symbol"`
	MarketType         string  `json:"market_type"`
	PriceChange        float64 `json:"price_change"`
	PriceChangePercent float64 `json:"price_change_percent"`
	WeightedAvgPrice   float64 `json:"weighted_avg_price"`
	PrevClosePrice     float64 `json:"prev_close_price"`
	LastPrice          float64 `json:"last_price"`
	LastQty            float64 `json:"last_qty"`
	BidPrice           float64 `json:"bid_price"`
	BidQty             float64 `json:"bid_qty"`
	AskPrice           float64 `json:"ask_price"`
	AskQty             float64 `json:"ask_qty"`
	OpenPrice          float64 `json:"open_price"`
	HighPrice          float64 `json:"high_price"`
	LowPrice           float64 `json:"low_price"`
	Volume             float64 `json:"volume"`
	QuoteVolume        float64 `json:"quote_volume"`
	OpenTime           int64   `json:"open_time"`
	CloseTime          int64   `json:"close_time"`
	FirstId            int64   `json:"first_id"`
	LastId             int64   `json:"last_id"`
	Count              int64   `json:"count"`

	// 历史表特有字段
	WindowStart    *time.Time `json:"window_start,omitempty"`
	WindowEnd      *time.Time `json:"window_end,omitempty"`
	WindowDuration *int       `json:"window_duration,omitempty"`

	// 元数据
	CreatedAt  time.Time `json:"created_at"`
	DataSource string    `json:"data_source"` // "realtime" 或 "history"
}

// StatsQueryService 智能查询服务
type StatsQueryService struct {
	db *gorm.DB
}

// NewStatsQueryService 创建新的查询服务实例
func NewStatsQueryService(db *gorm.DB) *StatsQueryService {
	return &StatsQueryService{
		db: db,
	}
}

// Query 执行智能查询
func (s *StatsQueryService) Query(query *StatsQuery) ([]StatsResult, error) {
	if query == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	// 参数验证
	if err := s.validateQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// 根据查询类型路由到相应的处理方法
	switch query.QueryType {
	case QueryTypeLatest:
		return s.queryLatest(query)
	case QueryTypeTimeRange:
		return s.queryTimeRange(query)
	case QueryTypeTechnical:
		return s.queryTechnical(query)
	case QueryTypeBacktest:
		return s.queryBacktest(query)
	case QueryTypeMixed:
		return s.queryMixed(query)
	default:
		return nil, fmt.Errorf("unsupported query type: %s", query.QueryType)
	}
}

// validateQuery 验证查询参数
func (s *StatsQueryService) validateQuery(query *StatsQuery) error {
	if query.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if query.MarketType == "" {
		return fmt.Errorf("market_type cannot be empty")
	}

	// 时间范围验证
	if query.QueryType == QueryTypeTimeRange || query.QueryType == QueryTypeTechnical ||
		query.QueryType == QueryTypeBacktest || query.QueryType == QueryTypeMixed {
		if query.StartTime == nil || query.EndTime == nil {
			return fmt.Errorf("start_time and end_time are required for %s queries", query.QueryType)
		}
		if query.StartTime.After(*query.EndTime) {
			return fmt.Errorf("start_time cannot be after end_time")
		}
	}

	// 分页参数验证
	if query.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if query.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	// 排序参数验证
	if query.SortOrder != "" && query.SortOrder != "asc" && query.SortOrder != "desc" {
		return fmt.Errorf("sort_order must be 'asc' or 'desc'")
	}

	return nil
}

// queryLatest 查询最新数据（从实时表）
func (s *StatsQueryService) queryLatest(query *StatsQuery) ([]StatsResult, error) {
	var realtimeStats Binance24hStats

	err := s.db.Where("symbol = ? AND market_type = ?", query.Symbol, query.MarketType).
		First(&realtimeStats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []StatsResult{}, nil // 返回空结果
		}
		return nil, fmt.Errorf("failed to query latest stats: %w", err)
	}

	// 转换为统一结果格式
	result := s.convertRealtimeToResult(realtimeStats)
	return []StatsResult{result}, nil
}

// queryTimeRange 查询时间范围数据（从历史表）
func (s *StatsQueryService) queryTimeRange(query *StatsQuery) ([]StatsResult, error) {
	var historyStats []Binance24hStatsHistory

	// 构建基础查询
	dbQuery := s.db.Table("binance_24h_stats_history").Where(
		"symbol = ? AND market_type = ? AND window_start >= ? AND window_end <= ?",
		query.Symbol, query.MarketType, query.StartTime, query.EndTime,
	)

	// 应用高级过滤条件
	dbQuery = s.applyAdvancedFilters(dbQuery, query)

	// 应用排序
	dbQuery = s.applySorting(dbQuery, query)

	// 应用分页
	if query.Limit > 0 {
		dbQuery = dbQuery.Limit(query.Limit)
	}
	if query.Offset > 0 {
		dbQuery = dbQuery.Offset(query.Offset)
	}

	err := dbQuery.Find(&historyStats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query time range: %w", err)
	}

	// 转换为统一结果格式
	results := make([]StatsResult, len(historyStats))
	for i, stat := range historyStats {
		results[i] = s.convertHistoryToResult(stat)
	}

	return results, nil
}

// queryTechnical 技术分析查询（优化版时间范围查询）
func (s *StatsQueryService) queryTechnical(query *StatsQuery) ([]StatsResult, error) {
	// 技术分析通常需要更多历史数据，默认限制最近30天
	if query.EndTime == nil {
		now := time.Now()
		query.EndTime = &now
	}
	if query.StartTime == nil {
		startTime := query.EndTime.Add(-30 * 24 * time.Hour)
		query.StartTime = &startTime
	}

	// 设置技术分析的默认排序（按时间升序）
	if query.SortBy == "" {
		query.SortBy = "window_start"
		query.SortOrder = "asc"
	}

	// 调用时间范围查询
	return s.queryTimeRange(query)
}

// queryBacktest 回测数据查询（大量历史数据）
func (s *StatsQueryService) queryBacktest(query *StatsQuery) ([]StatsResult, error) {
	// 回测通常需要大量历史数据，默认限制最近90天
	if query.EndTime == nil {
		now := time.Now()
		query.EndTime = &now
	}
	if query.StartTime == nil {
		startTime := query.EndTime.Add(-90 * 24 * time.Hour)
		query.StartTime = &startTime
	}

	// 设置回测的默认排序（按时间升序，便于时间序列分析）
	if query.SortBy == "" {
		query.SortBy = "window_start"
		query.SortOrder = "asc"
	}

	// 回测数据通常不需要分页限制
	if query.Limit <= 0 {
		query.Limit = 10000 // 设置合理的上限
	}

	return s.queryTimeRange(query)
}

// queryMixed 混合查询（实时数据 + 历史数据）
func (s *StatsQueryService) queryMixed(query *StatsQuery) ([]StatsResult, error) {
	var allResults []StatsResult

	// 1. 获取实时数据
	latestResults, err := s.queryLatest(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest data in mixed query: %w", err)
	}

	// 2. 获取历史数据（排除最近的数据以避免重复）
	if query.StartTime != nil && query.EndTime != nil {
		// 调整历史查询的时间范围，避免与实时数据重复
		historyEndTime := time.Now().Add(-time.Hour) // 1小时前的数据
		if historyEndTime.After(*query.StartTime) {
			historyQuery := *query
			historyQuery.EndTime = &historyEndTime
			historyQuery.QueryType = QueryTypeTimeRange

			historyResults, err := s.queryTimeRange(&historyQuery)
			if err != nil {
				return nil, fmt.Errorf("failed to query history data in mixed query: %w", err)
			}
			allResults = append(allResults, historyResults...)
		}
	}

	// 3. 添加实时数据
	allResults = append(allResults, latestResults...)

	// 4. 统一排序
	allResults = s.sortResults(allResults, query)

	// 5. 应用分页
	if query.Limit > 0 && len(allResults) > query.Limit {
		if query.Offset >= len(allResults) {
			return []StatsResult{}, nil
		}
		end := query.Offset + query.Limit
		if end > len(allResults) {
			end = len(allResults)
		}
		allResults = allResults[query.Offset:end]
	}

	return allResults, nil
}

// applyAdvancedFilters 应用高级过滤条件
func (s *StatsQueryService) applyAdvancedFilters(dbQuery *gorm.DB, query *StatsQuery) *gorm.DB {
	if query.PriceChangeMin != nil {
		dbQuery = dbQuery.Where("price_change >= ?", *query.PriceChangeMin)
	}
	if query.PriceChangeMax != nil {
		dbQuery = dbQuery.Where("price_change <= ?", *query.PriceChangeMax)
	}
	if query.VolumeMin != nil {
		dbQuery = dbQuery.Where("volume >= ?", *query.VolumeMin)
	}
	if query.VolumeMax != nil {
		dbQuery = dbQuery.Where("volume <= ?", *query.VolumeMax)
	}

	return dbQuery
}

// applySorting 应用排序
func (s *StatsQueryService) applySorting(dbQuery *gorm.DB, query *StatsQuery) *gorm.DB {
	if query.SortBy == "" {
		query.SortBy = "window_start"
	}
	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}

	orderClause := fmt.Sprintf("%s %s", query.SortBy, query.SortOrder)
	return dbQuery.Order(orderClause)
}

// sortResults 对结果进行排序
func (s *StatsQueryService) sortResults(results []StatsResult, query *StatsQuery) []StatsResult {
	if len(results) <= 1 {
		return results
	}

	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sort.Slice(results, func(i, j int) bool {
		var compareResult bool

		switch sortBy {
		case "created_at":
			if results[i].CreatedAt.Equal(results[j].CreatedAt) {
				compareResult = results[i].ID < results[j].ID
			} else {
				compareResult = results[i].CreatedAt.Before(results[j].CreatedAt)
			}
		case "window_start":
			if results[i].WindowStart == nil || results[j].WindowStart == nil {
				compareResult = results[i].CreatedAt.Before(results[j].CreatedAt)
			} else if results[i].WindowStart.Equal(*results[j].WindowStart) {
				compareResult = results[i].ID < results[j].ID
			} else {
				compareResult = results[i].WindowStart.Before(*results[j].WindowStart)
			}
		case "last_price":
			compareResult = results[i].LastPrice < results[j].LastPrice
		case "volume":
			compareResult = results[i].Volume < results[j].Volume
		default:
			compareResult = results[i].CreatedAt.Before(results[j].CreatedAt)
		}

		if sortOrder == "desc" {
			return !compareResult
		}
		return compareResult
	})

	return results
}

// convertRealtimeToResult 将实时表数据转换为统一结果格式
func (s *StatsQueryService) convertRealtimeToResult(realtime Binance24hStats) StatsResult {
	return StatsResult{
		ID:                 realtime.ID,
		Symbol:             realtime.Symbol,
		MarketType:         realtime.MarketType,
		PriceChange:        realtime.PriceChange,
		PriceChangePercent: realtime.PriceChangePercent,
		WeightedAvgPrice:   realtime.WeightedAvgPrice,
		PrevClosePrice:     realtime.PrevClosePrice,
		LastPrice:          realtime.LastPrice,
		LastQty:            realtime.LastQty,
		BidPrice:           realtime.BidPrice,
		BidQty:             realtime.BidQty,
		AskPrice:           realtime.AskPrice,
		AskQty:             realtime.AskQty,
		OpenPrice:          realtime.OpenPrice,
		HighPrice:          realtime.HighPrice,
		LowPrice:           realtime.LowPrice,
		Volume:             realtime.Volume,
		QuoteVolume:        realtime.QuoteVolume,
		OpenTime:           realtime.OpenTime,
		CloseTime:          realtime.CloseTime,
		FirstId:            realtime.FirstId,
		LastId:             realtime.LastId,
		Count:              realtime.Count,
		CreatedAt:          realtime.CreatedAt,
		DataSource:         "realtime",
	}
}

// convertHistoryToResult 将历史表数据转换为统一结果格式
func (s *StatsQueryService) convertHistoryToResult(history Binance24hStatsHistory) StatsResult {
	return StatsResult{
		ID:                 history.ID,
		Symbol:             history.Symbol,
		MarketType:         history.MarketType,
		PriceChange:        history.PriceChange,
		PriceChangePercent: history.PriceChangePercent,
		WeightedAvgPrice:   history.WeightedAvgPrice,
		PrevClosePrice:     history.PrevClosePrice,
		LastPrice:          history.LastPrice,
		LastQty:            history.LastQty,
		BidPrice:           history.BidPrice,
		BidQty:             history.BidQty,
		AskPrice:           history.AskPrice,
		AskQty:             history.AskQty,
		OpenPrice:          history.OpenPrice,
		HighPrice:          history.HighPrice,
		LowPrice:           history.LowPrice,
		Volume:             history.Volume,
		QuoteVolume:        history.QuoteVolume,
		OpenTime:           history.OpenTime,
		CloseTime:          history.CloseTime,
		FirstId:            history.FirstId,
		LastId:             history.LastId,
		Count:              history.Count,
		WindowStart:        &history.WindowStart,
		WindowEnd:          &history.WindowEnd,
		WindowDuration:     &history.WindowDuration,
		CreatedAt:          history.CreatedAt,
		DataSource:         "history",
	}
}

// GetStats 获取查询服务的统计信息
func (s *StatsQueryService) GetStats() map[string]interface{} {
	// 这里可以添加查询统计信息
	return map[string]interface{}{
		"service_name": "StatsQueryService",
		"supported_query_types": []string{
			string(QueryTypeLatest),
			string(QueryTypeTimeRange),
			string(QueryTypeTechnical),
			string(QueryTypeBacktest),
			string(QueryTypeMixed),
		},
	}
}
