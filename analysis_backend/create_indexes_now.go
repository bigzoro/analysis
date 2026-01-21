package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔧 开始创建关键数据库索引...")

	// 加载配置
	var cfg config.Config
	config.MustLoad("config.yaml", &cfg)
	config.ApplyProxy(&cfg)

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  false,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer gdb.Close()

	fmt.Println("📡 数据库连接成功")

	// 定义要创建的索引
	indexes := []struct {
		table   string
		name    string
		columns string
		desc    string
	}{
		{
			table:   "market_klines",
			name:    "idx_open_time_symbol",
			columns: "open_time ASC, symbol",
			desc:    "时间+符号索引 (解决时间范围+多币种IN查询)",
		},
		{
			table:   "market_klines",
			name:    "idx_symbol_open_time",
			columns: "symbol, open_time ASC",
			desc:    "符号+时间索引 (解决多币种IN+时间范围查询)",
		},
		{
			table:   "binance_24h_stats",
			name:    "idx_quote_volume",
			columns: "quote_volume",
			desc:    "交易量索引 (解决交易量排序查询)",
		},
		{
			table:   "binance_24h_stats",
			name:    "idx_symbol_created_at",
			columns: "symbol, created_at",
			desc:    "币种+时间索引 (解决币种时间范围查询)",
		},
	}

	created := 0
	skipped := 0

	// 创建每个索引
	for _, idx := range indexes {
		fmt.Printf("⏳ 检查索引 %s.%s...\n", idx.table, idx.name)

		// 检查索引是否已存在
		var count int64
		checkSQL := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM information_schema.statistics
			WHERE table_schema = DATABASE()
			AND table_name = ?
			AND index_name = ?
		`, idx.table, idx.name)

		err := gdb.GormDB().Raw(checkSQL, idx.table, idx.name).Scan(&count).Error
		if err != nil {
			fmt.Printf("⚠️  检查索引失败: %v\n", err)
			continue
		}

		if count > 0 {
			fmt.Printf("⏭️  索引 %s 已存在，跳过\n", idx.name)
			skipped++
			continue
		}

		// 创建索引
		createSQL := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", idx.name, idx.table, idx.columns)
		fmt.Printf("🔨 创建索引: %s\n", idx.desc)

		startTime := time.Now()
		err = gdb.GormDB().Exec(createSQL).Error
		duration := time.Since(startTime)

		if err != nil {
			// 检查是否是索引已存在的错误
			if isIndexExistsError(err) {
				fmt.Printf("⏭️  索引 %s 已存在，跳过\n", idx.name)
				skipped++
			} else {
				fmt.Printf("❌ 创建索引失败: %v\n", err)
			}
			continue
		}

		fmt.Printf("✅ 索引 %s 创建成功 (耗时: %v)\n", idx.name, duration)
		created++
	}

	// 验证结果
	fmt.Println("\n📊 索引创建总结:")
	fmt.Printf("✅ 新创建: %d 个索引\n", created)
	fmt.Printf("⏭️  已存在: %d 个索引\n", skipped)
	fmt.Printf("📈 总计: %d 个索引\n", created+skipped)

	if created > 0 {
		fmt.Println("\n🎉 索引创建完成！")
		fmt.Println("📝 性能提升预期:")
		fmt.Println("   • 时间+符号查询: 500ms → <50ms")
		fmt.Println("   • 符号+时间查询: 14秒 → <100ms")
		fmt.Println("   • 整体响应: 显著提升")
	} else {
		fmt.Println("\nℹ️  所有索引都已存在，无需创建")
	}

	fmt.Println("\n🚀 现在可以重启应用测试性能提升！")
}

// 检查是否是索引已存在的错误
func isIndexExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsAny(errStr, []string{
		"Duplicate key name",
		"already exists",
		"Duplicate index",
		"1061", // MySQL error code
	})
}

func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
