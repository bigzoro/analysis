package db

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ==================== 索引优化 ====================

// CreateOptimizedIndexes 创建优化后的索引
// 在数据库迁移后调用，或作为独立迁移脚本
func CreateOptimizedIndexes(gdb *gorm.DB) error {
	indexes := []struct {
		table   string
		name    string
		columns []string
		unique  bool
	}{
		// Trading Strategies 表优化索引（策略调度优化）
		{"trading_strategies", "idx_strategies_running_last_run", []string{"is_running", "last_run_at"}, false},

		// TransferEvent 表优化索引
		{"transfer_events", "idx_te_entity_occurred", []string{"entity", "occurred_at"}, false},
		{"transfer_events", "idx_te_chain_occurred", []string{"chain", "occurred_at"}, false},
		{"transfer_events", "idx_te_coin_occurred", []string{"coin", "occurred_at"}, false},
		{"transfer_events", "idx_te_entity_chain_occurred", []string{"entity", "chain", "occurred_at"}, false},
		{"transfer_events", "idx_te_entity_coin_occurred", []string{"entity", "coin", "occurred_at"}, false},
		{"transfer_events", "idx_te_created_at", []string{"created_at"}, false},
		{"transfer_events", "idx_te_txid", []string{"tx_id"}, false},
		{"transfer_events", "idx_te_address_occurred", []string{"address", "occurred_at"}, false},

		// PortfolioSnapshot 表优化索引
		{"portfolio_snapshots", "idx_ps_entity_created", []string{"entity", "created_at"}, false},
		{"portfolio_snapshots", "idx_ps_as_of", []string{"as_of"}, false},

		// Holding 表优化索引
		{"holdings", "idx_h_entity_chain", []string{"entity", "chain"}, false},
		{"holdings", "idx_h_run_entity", []string{"run_id", "entity"}, false},

		// DailyFlow 表优化索引
		{"daily_flows", "idx_df_entity_day", []string{"entity", "day"}, false},
		{"daily_flows", "idx_df_coin_day", []string{"coin", "day"}, false},
		{"daily_flows", "idx_df_day", []string{"day"}, false},

		// WeeklyFlow 表优化索引
		{"weekly_flows", "idx_wf_entity_week", []string{"entity", "week"}, false},
		{"weekly_flows", "idx_wf_coin_week", []string{"coin", "week"}, false},

		// BinanceMarketTop 表优化索引
		{"binance_market_tops", "idx_bmt_snapshot_rank", []string{"snapshot_id", "rank"}, false},
		{"binance_market_tops", "idx_bmt_symbol_rank", []string{"symbol", "rank"}, false},

		// Announcement 表优化索引
		{"announcements", "idx_ann_source_release", []string{"source", "release_time"}, false},
		{"announcements", "idx_ann_category_release", []string{"category", "release_time"}, false},
		{"announcements", "idx_ann_release", []string{"release_time"}, false},
		{"announcements", "idx_ann_title_ft", []string{"title(255)"}, false}, // 全文索引（如果支持）

		// TwitterPost 表优化索引
		{"twitter_posts", "idx_tp_username_time", []string{"username", "tweet_time"}, false},
		{"twitter_posts", "idx_tp_time", []string{"tweet_time"}, false},

		// ScheduledOrder 表优化索引
		{"scheduled_orders", "idx_so_user_status", []string{"user_id", "status"}, false},
		{"scheduled_orders", "idx_so_status_trigger", []string{"status", "trigger_time"}, false},
		{"scheduled_orders", "idx_so_trigger", []string{"trigger_time"}, false},
		// 策略相关查询优化索引（高优先级）
		{"scheduled_orders", "idx_orders_strategy_symbol_reduce_created", []string{"strategy_id", "symbol", "reduce_only", "created_at"}, false},
		{"scheduled_orders", "idx_orders_strategy_symbol_status_reduce", []string{"strategy_id", "symbol", "status", "reduce_only"}, false},
		{"scheduled_orders", "idx_orders_trigger_status", []string{"trigger_time", "status"}, false},

		// BinanceExchangeInfo 表优化索引
		{"binance_exchange_info", "idx_exchange_info_symbol_market", []string{"symbol", "market_type"}, true}, // 复合唯一索引
		{"binance_exchange_info", "idx_exchange_info_quote_market", []string{"quote_asset", "market_type"}, false},
		{"binance_exchange_info", "idx_exchange_info_status_market", []string{"status", "market_type"}, false},
		{"binance_exchange_info", "idx_exchange_info_symbol_status", []string{"symbol", "status"}, false},
		{"binance_exchange_info", "idx_exchange_info_quote_asset_status", []string{"quote_asset", "status"}, false},
		{"binance_exchange_info", "idx_exchange_info_updated_at", []string{"updated_at"}, false},

		// 状态管理相关索引
		{"binance_exchange_info", "idx_exchange_info_is_active", []string{"is_active"}, false},
		{"binance_exchange_info", "idx_exchange_info_deactivated_at", []string{"deactivated_at"}, false},
		{"binance_exchange_info", "idx_exchange_info_last_seen_active", []string{"last_seen_active"}, false},
		{"binance_exchange_info", "idx_exchange_info_status_active", []string{"status", "is_active"}, false},
		{"binance_exchange_info", "idx_exchange_info_market_active", []string{"market_type", "is_active"}, false},

		// Market Klines 表优化索引（防止重复数据）
		{"market_klines", "idx_market_klines_symbol_kind_interval_open_time", []string{"symbol", "kind", "interval", "open_time"}, true}, // 复合唯一索引
		{"market_klines", "idx_market_klines_symbol_interval_time", []string{"symbol", "interval", "open_time"}, false},
		{"market_klines", "idx_market_klines_kind_interval_time", []string{"kind", "interval", "open_time"}, false},
		{"market_klines", "idx_market_klines_open_time", []string{"open_time"}, false},
		{"market_klines", "idx_market_klines_symbol_open_time", []string{"symbol", "open_time"}, false},
		{"market_klines", "idx_market_klines_volume_quote_volume", []string{"volume", "quote_volume"}, false},

		// Binance Order Book Depth 表优化索引（防止重复数据）
		{"binance_order_book_depth", "idx_order_book_symbol_market_update_id", []string{"symbol", "market_type", "last_update_id"}, true}, // 复合唯一索引
		{"binance_order_book_depth", "idx_order_book_symbol_market", []string{"symbol", "market_type"}, false},
		{"binance_order_book_depth", "idx_order_book_last_update_id", []string{"last_update_id"}, false},
		{"binance_order_book_depth", "idx_order_book_snapshot_time", []string{"snapshot_time"}, false},

		// Binance 24h Stats 表优化索引（防止重复数据）
		{"binance_24h_stats", "idx_24h_stats_symbol_market", []string{"symbol", "market_type"}, true},                // 复合唯一索引
		{"binance_24h_stats", "idx_24h_stats_market_change", []string{"market_type", "price_change_percent"}, false}, // 涨幅榜查询优化
		{"binance_24h_stats", "idx_24h_stats_symbol_created_at", []string{"symbol", "created_at"}, false},
		{"binance_24h_stats", "idx_24h_stats_quote_volume_created_at", []string{"quote_volume", "created_at"}, false},
		{"binance_24h_stats", "idx_24h_stats_created_at_symbol", []string{"created_at", "symbol"}, false},
		{"binance_24h_stats", "idx_24h_stats_created_at", []string{"created_at"}, false},

		// Binance 24h Stats History 表优化索引（时间序列查询优化）
		{"binance_24h_stats_history", "idx_history_stats_symbol_market_window", []string{"symbol", "market_type", "window_start"}, true}, // 时间序列唯一索引
		{"binance_24h_stats_history", "idx_history_stats_time_series", []string{"symbol", "market_type", "window_start"}, false},         // 时间序列查询优化
		{"binance_24h_stats_history", "idx_history_stats_symbol_window", []string{"symbol", "window_start"}, false},                      // 单个symbol时间序列查询
		{"binance_24h_stats_history", "idx_history_stats_market_window", []string{"market_type", "window_start"}, false},                 // 市场类型时间序列查询
		{"binance_24h_stats_history", "idx_history_stats_window_range", []string{"window_start", "window_end"}, false},                   // 时间范围查询优化
		{"binance_24h_stats_history", "idx_history_stats_created_at", []string{"created_at"}, false},                                     // 创建时间索引

		// Binance Trades 表优化索引（已有唯一索引，添加查询优化索引）
		{"binance_trades", "idx_trades_symbol_market_time", []string{"symbol", "market_type", "trade_time"}, false},
		{"binance_trades", "idx_trades_trade_time", []string{"trade_time"}, false},
		{"binance_trades", "idx_trades_symbol_time", []string{"symbol", "trade_time"}, false},

		// Futures Contracts 表优化索引
		// 注意：symbol字段由GORM的uniqueIndex标签管理，不在此处重复定义
		{"binance_futures_contracts", "idx_futures_contracts_status", []string{"status"}, false},
		{"binance_futures_contracts", "idx_futures_contracts_updated_at", []string{"updated_at"}, false},

		// Funding Rates 表优化索引
		{"binance_funding_rates", "idx_funding_rates_symbol_time", []string{"symbol", "funding_time"}, true}, // 复合唯一索引
		{"binance_funding_rates", "idx_funding_rates_symbol", []string{"symbol"}, false},
		{"binance_funding_rates", "idx_funding_rates_funding_time", []string{"funding_time"}, false},

		// Price Cache 表优化索引
		{"price_caches", "idx_price_caches_symbol_kind", []string{"symbol", "kind"}, true}, // 复合唯一索引
		{"price_caches", "idx_price_caches_last_updated", []string{"last_updated"}, false},

		// External Operations 表优化索引
		{"external_operations", "idx_eo_symbol_operation", []string{"symbol", "operation_type"}, false},
		{"external_operations", "idx_eo_user_detected", []string{"user_id", "detected_at"}, false},
		{"external_operations", "idx_eo_status_detected", []string{"status", "detected_at"}, false},
		{"external_operations", "idx_eo_confidence", []string{"confidence"}, false},

		// Operation Logs 表优化索引
		{"operation_logs", "idx_ol_entity_type_id", []string{"entity_type", "entity_id"}, false},
		{"operation_logs", "idx_ol_user_action", []string{"user_id", "action"}, false},
		{"operation_logs", "idx_ol_source_level", []string{"source", "level"}, false},
		{"operation_logs", "idx_ol_created_at", []string{"created_at"}, false},

		// Audit Trails 表优化索引
		{"audit_trails", "idx_at_session_user", []string{"session_id", "user_id"}, false},
		{"audit_trails", "idx_at_resource_action", []string{"resource_type", "action"}, false},
		{"audit_trails", "idx_at_timestamp", []string{"timestamp"}, false},
		{"audit_trails", "idx_at_success", []string{"success"}, false},
	}

	for _, idx := range indexes {
		// 先检查索引是否已存在
		exists, err := checkIndexExists(gdb, idx.table, idx.name)
		if err != nil {
			// 如果检查失败，尝试直接创建（可能是权限问题）
			// 继续执行创建逻辑
		} else if exists {
			// 索引已存在，跳过
			continue
		}

		// 如果是唯一索引，先清理可能存在的重复数据
		if idx.unique {
			if err := cleanupDuplicateDataForUniqueIndex(gdb, idx.table, idx.columns); err != nil {
				log.Printf("[Optimization] 清理表 %s 的重复数据失败: %v", idx.table, err)
				// 继续尝试创建索引，可能失败但会被错误处理捕获
			}
		}

		unique := ""
		if idx.unique {
			unique = "UNIQUE"
		}

		columns := ""
		for i, col := range idx.columns {
			if i > 0 {
				columns += ", "
			}
			// 用反引号括起列名，避免保留关键字问题（如 rank）
			// 如果列名已经包含括号（如 title(255)），则不添加反引号
			if strings.Contains(col, "(") {
				columns += col
			} else {
				columns += "`" + col + "`"
			}
		}

		// MySQL 不支持 IF NOT EXISTS，直接创建
		sql := fmt.Sprintf(
			"CREATE %s INDEX %s ON %s (%s)",
			unique, idx.name, idx.table, columns,
		)

		if err := gdb.Exec(sql).Error; err != nil {
			// 忽略已存在的索引错误
			if !isIndexExistsError(err) {
				return fmt.Errorf("create index %s on %s: %w", idx.name, idx.table, err)
			}
		}
	}

	return nil
}

// cleanupDuplicateDataForUniqueIndex 为唯一索引清理重复数据
func cleanupDuplicateDataForUniqueIndex(gdb *gorm.DB, tableName string, columns []string) error {
	if len(columns) == 0 {
		return nil
	}

	// 构建 GROUP BY 子句
	groupBy := ""
	for i, col := range columns {
		if i > 0 {
			groupBy += ", "
		}
		groupBy += "`" + col + "`"
	}

	// 查找重复数据并保留ID最小的记录
	cleanupSQL := fmt.Sprintf(`
		DELETE t1 FROM %s t1
		INNER JOIN (
			SELECT %s, MIN(id) as min_id
			FROM %s
			GROUP BY %s
			HAVING COUNT(*) > 1
		) t2 ON %s
		WHERE t1.id > t2.min_id
	`, tableName, groupBy, tableName, groupBy, buildJoinCondition(columns))

	if err := gdb.Exec(cleanupSQL).Error; err != nil {
		return fmt.Errorf("清理重复数据失败: %w", err)
	}

	return nil
}

// buildJoinCondition 构建JOIN条件
func buildJoinCondition(columns []string) string {
	condition := ""
	for i, col := range columns {
		if i > 0 {
			condition += " AND "
		}
		condition += fmt.Sprintf("t1.`%s` = t2.`%s`", col, col)
	}
	return condition
}

// checkIndexExists 检查索引是否存在
func checkIndexExists(gdb *gorm.DB, tableName, indexName string) (bool, error) {
	var count int64
	// 查询 information_schema 检查索引是否存在
	sql := `
		SELECT COUNT(*) 
		FROM information_schema.statistics 
		WHERE table_schema = DATABASE() 
		AND table_name = ? 
		AND index_name = ?
	`
	if err := gdb.Raw(sql, tableName, indexName).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func isIndexExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate key name") ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "Duplicate index") ||
		strings.Contains(errStr, "1061") || // MySQL error code for duplicate key
		strings.Contains(errStr, "23000") // MySQL error code for integrity constraint violation (duplicate entry)
}

// ==================== 分页优化 ====================

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
	MaxSize  int // 最大每页数量
}

func (p *PaginationParams) Normalize() {
	if p.PageSize <= 0 {
		p.PageSize = 50
	}
	if p.MaxSize > 0 && p.PageSize > p.MaxSize {
		p.PageSize = p.MaxSize
	}
	if p.Page <= 0 {
		p.Page = 1
	}
}

func (p *PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *PaginationParams) Limit() int {
	return p.PageSize
}

// PaginatedResult 分页结果
type PaginatedResult struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// Paginate 通用分页查询
func Paginate[T any](db *gorm.DB, params PaginationParams, query func(*gorm.DB) *gorm.DB) (*PaginatedResult, error) {
	params.Normalize()

	var total int64
	baseQuery := db.Model(new(T))
	if query != nil {
		baseQuery = query(baseQuery)
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []T
	q := baseQuery.Offset(params.Offset()).Limit(params.Limit())
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &PaginatedResult{
		Items:      items,
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}, nil
}

// ==================== 游标分页优化 ====================

// CursorPagination 游标分页（性能更好，适合大数据量）
type CursorPagination struct {
	Cursor    string // 游标值（通常是 ID 或时间戳）
	Limit     int
	Direction string // "next" 或 "prev"
}

// CursorResult 游标分页结果
type CursorResult struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
	PrevCursor string      `json:"prev_cursor,omitempty"`
	HasMore    bool        `json:"has_more"`
}

// ==================== 批量查询优化 ====================

// BatchQuery 批量查询优化
func BatchQuery[T any](db *gorm.DB, batchSize int, query func(*gorm.DB, int, int) *gorm.DB, processor func([]T) error) error {
	offset := 0
	for {
		var items []T
		q := query(db, offset, batchSize)
		if err := q.Find(&items).Error; err != nil {
			return err
		}

		if len(items) == 0 {
			break
		}

		if err := processor(items); err != nil {
			return err
		}

		if len(items) < batchSize {
			break
		}

		offset += batchSize
	}
	return nil
}

// ==================== 统计查询优化 ====================

// GetTransferStats 获取转账统计（使用聚合查询）
func GetTransferStats(gdb *gorm.DB, entity, chain, coin string, start, end time.Time) (map[string]interface{}, error) {
	type Stats struct {
		TotalCount int64   `json:"total_count"`
		TotalIn    float64 `json:"total_in"`
		TotalOut   float64 `json:"total_out"`
		NetFlow    float64 `json:"net_flow"`
		MaxAmount  float64 `json:"max_amount"`
		AvgAmount  float64 `json:"avg_amount"`
	}

	var stats Stats
	q := gdb.Model(&TransferEvent{}).
		Select(`
			COUNT(*) as total_count,
			COALESCE(SUM(CASE WHEN direction = 'in' THEN CAST(amount AS DECIMAL(38,18)) ELSE 0 END), 0) as total_in,
			COALESCE(SUM(CASE WHEN direction = 'out' THEN CAST(amount AS DECIMAL(38,18)) ELSE 0 END), 0) as total_out,
			COALESCE(MAX(CAST(amount AS DECIMAL(38,18))), 0) as max_amount,
			COALESCE(AVG(CAST(amount AS DECIMAL(38,18))), 0) as avg_amount
		`).
		Where("occurred_at >= ? AND occurred_at < ?", start, end)

	if entity != "" {
		q = q.Where("entity = ?", entity)
	}
	if chain != "" {
		q = q.Where("chain = ?", chain)
	}
	if coin != "" {
		q = q.Where("coin = ?", coin)
	}

	if err := q.Scan(&stats).Error; err != nil {
		return nil, err
	}

	stats.NetFlow = stats.TotalIn - stats.TotalOut

	return map[string]interface{}{
		"total_count": stats.TotalCount,
		"total_in":    stats.TotalIn,
		"total_out":   stats.TotalOut,
		"net_flow":    stats.NetFlow,
		"max_amount":  stats.MaxAmount,
		"avg_amount":  stats.AvgAmount,
	}, nil
}

// ==================== 连接池优化 ====================

// OptimizeConnectionPool 优化数据库连接池
func OptimizeConnectionPool(sqlDB *gorm.DB) error {
	db, err := sqlDB.DB()
	if err != nil {
		return err
	}

	// 根据实际负载调整
	db.SetMaxOpenConns(25)                  // 最大打开连接数
	db.SetMaxIdleConns(10)                  // 最大空闲连接数
	db.SetConnMaxLifetime(30 * time.Minute) // 连接最大生存时间
	db.SetConnMaxIdleTime(10 * time.Minute) // 连接最大空闲时间

	return nil
}

// ==================== 查询性能监控 ====================

// QueryMetrics 查询性能指标
type QueryMetrics struct {
	Query        string
	Duration     time.Duration
	RowsAffected int64
	Error        error
}

// SlowQueryThreshold 慢查询阈值
const SlowQueryThreshold = 1 * time.Second

// LogSlowQuery 记录慢查询
func LogSlowQuery(query string, duration time.Duration, rows int64) {
	if duration > SlowQueryThreshold {
		// 这里应该使用结构化日志
		fmt.Printf("[SLOW QUERY] %s took %v, rows: %d\n", query, duration, rows)
	}
}
