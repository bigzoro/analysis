package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==================== 查询优化器 ====================

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	db *gorm.DB
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *gorm.DB) *QueryOptimizer {
	return &QueryOptimizer{db: db}
}

// OptimizeTransferQuery 优化转账查询
// 优化点：
// 1. 避免使用 LOWER/UPPER 函数，在存储时统一大小写
// 2. 使用覆盖索引
// 3. 优化排序和限制
func (qo *QueryOptimizer) OptimizeTransferQuery(entity, chain, coin string, limit int) *gorm.DB {
	q := qo.db.Model(&TransferEvent{})

	// 按优先级添加 WHERE 条件（利用复合索引）
	if entity != "" {
		q = q.Where("entity = ?", entity)
	}
	if chain != "" {
		// 注意：如果存储时统一了小写，这里不需要 LOWER()
		// 否则需要在应用层统一处理
		q = q.Where("chain = ?", strings.ToLower(chain))
	}
	if coin != "" {
		q = q.Where("coin = ?", strings.ToUpper(coin))
	}

	// 使用 occurred_at 索引进行排序（DESC 用于最新数据）
	q = q.Order("occurred_at DESC, id DESC").Limit(limit)

	return q
}

// OptimizeFlowQuery 优化资金流查询
func (qo *QueryOptimizer) OptimizeFlowQuery(entity string, coins []string, start, end string, latest bool, runID string) *gorm.DB {
	q := qo.db.Model(&DailyFlow{})

	// 优先使用 entity 索引
	q = q.Where("entity = ?", entity)

	if latest && runID != "" {
		q = q.Where("run_id = ?", runID)
	}

	if len(coins) > 0 {
		q = q.Where("coin IN ?", coins)
	}

	if start != "" {
		q = q.Where("day >= ?", start)
	}
	if end != "" {
		q = q.Where("day <= ?", end)
	}

	// 使用复合索引优化排序
	q = q.Order("coin ASC, day ASC")

	return q
}

// OptimizePortfolioQuery 优化资产组合查询
func (qo *QueryOptimizer) OptimizePortfolioQuery(entity, runID string) *gorm.DB {
	q := qo.db.Model(&Holding{})

	q = q.Where("run_id = ? AND entity = ?", runID, entity)
	q = q.Order("chain ASC, symbol ASC")

	return q
}

// ==================== 查询性能分析 ====================

// ExplainQuery 分析查询计划
func (qo *QueryOptimizer) ExplainQuery(query *gorm.DB) (string, error) {
	var result []map[string]interface{}

	// 获取 SQL 和参数
	sql := query.Statement.SQL.String()
	vars := query.Statement.Vars

	// 构建 EXPLAIN 查询
	explainSQL := "EXPLAIN " + sql

	rows, err := qo.db.Raw(explainSQL, vars...).Rows()
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}
		result = append(result, entry)
	}

	// 格式化输出
	var output strings.Builder
	for _, row := range result {
		output.WriteString(fmt.Sprintf("Table: %v, Type: %v, Key: %v, Rows: %v\n",
			row["table"], row["type"], row["key"], row["rows"]))
	}

	return output.String(), nil
}

// ==================== 慢查询检测 ====================

// QueryTimer 查询计时器
type QueryTimer struct {
	start time.Time
	query string
}

// StartTimer 开始计时
func StartTimer(query string) *QueryTimer {
	return &QueryTimer{
		start: time.Now(),
		query: query,
	}
}

// Stop 停止计时并记录
func (qt *QueryTimer) Stop(rowsAffected int64) {
	duration := time.Since(qt.start)
	if duration > SlowQueryThreshold {
		LogSlowQuery(qt.query, duration, rowsAffected)
	}
}

// ==================== 批量操作优化 ====================

// BatchInsert 批量插入优化
func BatchInsert[T any](db *gorm.DB, items []T, batchSize int) error {
	if len(items) == 0 {
		return nil
	}

	if batchSize <= 0 {
		batchSize = 1000
	}

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		if err := db.Create(items[i:end]).Error; err != nil {
			return fmt.Errorf("batch insert failed at offset %d: %w", i, err)
		}
	}

	return nil
}

// BatchUpsert 批量更新或插入
// 注意：MySQL 使用 INSERT ... ON DUPLICATE KEY UPDATE
func BatchUpsert[T any](db *gorm.DB, items []T, conflictColumns []string, updateColumns []string) error {
	if len(items) == 0 {
		return nil
	}

	// 构建更新列（使用 clause.AssignmentColumns）
	columns := make([]clause.Column, len(conflictColumns))
	for i, col := range conflictColumns {
		columns[i] = clause.Column{Name: col}
	}

	updateCols := make([]string, len(updateColumns))
	copy(updateCols, updateColumns)

	// 使用 GORM 的 OnConflict（MySQL 使用 ON DUPLICATE KEY UPDATE）
	return db.Clauses(clause.OnConflict{
		Columns:   columns,
		DoUpdates: clause.AssignmentColumns(updateCols),
	}).Create(items).Error
}

func buildUpdateSet(columns []string) string {
	var parts []string
	for _, col := range columns {
		parts = append(parts, fmt.Sprintf("%s = VALUES(%s)", col, col))
	}
	return strings.Join(parts, ", ")
}

// ==================== 数据预加载优化 ====================

// PreloadWithConditions 条件预加载
func PreloadWithConditions(db *gorm.DB, association string, conditions ...func(*gorm.DB) *gorm.DB) *gorm.DB {
	q := db
	for _, condition := range conditions {
		q = condition(q)
	}
	return q.Preload(association)
}

// ==================== 查询结果缓存键生成 ====================

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(prefix string, params map[string]interface{}) string {
	var parts []string
	parts = append(parts, prefix)

	// 按 key 排序以保证一致性
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}

	for _, k := range keys {
		v := params[k]
		parts = append(parts, fmt.Sprintf("%s:%v", k, v))
	}

	return strings.Join(parts, ":")
}
