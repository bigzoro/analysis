package main

import (
	"log"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 直接连接数据库（简化版本）
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	db, err := gdb.DB()
	if err != nil {
		log.Fatalf("获取数据库连接失败: %v", err)
	}
	defer db.Close()

	log.Println("开始创建market_klines表的优化索引...")

	// 创建关键索引
	indexes := []struct {
		name        string
		table       string
		columns     []string
		description string
	}{
		{
			name:        "idx_open_time_symbol",
			table:       "market_klines",
			columns:     []string{"open_time", "symbol"},
			description: "优化时间范围+币种查询 (517ms慢查询)",
		},
		{
			name:        "idx_symbol_open_time",
			table:       "market_klines",
			columns:     []string{"symbol", "open_time"},
			description: "优化币种+时间范围查询 (14.6s慢查询)",
		},
		{
			name:        "idx_binance_24h_stats_quote_volume_created_at",
			table:       "binance_24h_stats",
			columns:     []string{"quote_volume", "created_at"},
			description: "优化24h统计表按交易量和时间查询 (227ms慢查询)",
		},
		{
			name:        "idx_binance_24h_stats_created_at_symbol",
			table:       "binance_24h_stats",
			columns:     []string{"created_at", "symbol"},
			description: "优化24h统计表按时间和币种查询",
		},
	}

	for _, idx := range indexes {
		// 检查索引是否已存在
		exists, err := checkIndexExists(gdb, idx.table, idx.name)
		if err != nil {
			log.Printf("检查索引 %s 失败: %v", idx.name, err)
			continue
		}

		if exists {
			log.Printf("✅ 索引 %s 已存在 - %s", idx.name, idx.description)
			continue
		}

		// 创建索引
		var sql string
		if idx.table == "market_klines" {
			sql = "CREATE INDEX " + idx.name + " ON " + idx.table + " (open_time, symbol)"
			if idx.name == "idx_symbol_open_time" {
				sql = "CREATE INDEX " + idx.name + " ON " + idx.table + " (symbol, open_time)"
			}
		} else if idx.table == "binance_24h_stats" {
			if idx.name == "idx_binance_24h_stats_quote_volume_created_at" {
				sql = "CREATE INDEX " + idx.name + " ON " + idx.table + " (quote_volume, created_at)"
			} else if idx.name == "idx_binance_24h_stats_created_at_symbol" {
				sql = "CREATE INDEX " + idx.name + " ON " + idx.table + " (created_at, symbol)"
			}
		}

		if err := gdb.Exec(sql).Error; err != nil {
			// 忽略"already exists"错误
			if !isIndexExistsError(err) {
				log.Printf("❌ 创建索引 %s 失败: %v", idx.name, err)
				continue
			}
		}

		log.Printf("✅ 索引 %s 创建成功 - %s", idx.name, idx.description)
	}

	log.Println("索引创建完成！慢查询性能应该显著改善。")
}

// 复制检查索引函数（避免导入循环）
func checkIndexExists(gdb *gorm.DB, tableName, indexName string) (bool, error) {
	var count int64
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
		strings.Contains(errStr, "1061") // MySQL error code for duplicate key
}
