package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 使用默认的数据库连接字符串（根据项目配置）
	dsn := "user:password@tcp(localhost:3306)/trading?charset=utf8mb4&parseTime=True&loc=Local"
	if envDSN := os.Getenv("DB_DSN"); envDSN != "" {
		dsn = envDSN
	}

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	fmt.Println("=== BDXNUSDT 交易对分析 ===")

	// 查询BDXNUSDT的完整信息
	query := `
		SELECT
			symbol, status, market_type, is_active,
			deactivated_at, last_seen_active,
			created_at, updated_at
		FROM binance_exchange_info
		WHERE symbol = ?
	`

	var symbol, status, marketType string
	var isActive bool
	var deactivatedAt, lastSeenActive, createdAt, updatedAt sql.NullTime

	err = db.QueryRow(query, "BDXNUSDT").Scan(
		&symbol, &status, &marketType, &isActive,
		&deactivatedAt, &lastSeenActive, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("❌ BDXNUSDT 不在数据库中")
			return
		}
		log.Fatal("查询失败:", err)
	}

	fmt.Printf("📊 基本信息:\n")
	fmt.Printf("  交易对: %s\n", symbol)
	fmt.Printf("  状态: %s\n", status)
	fmt.Printf("  市场类型: %s\n", marketType)
	fmt.Printf("  活跃状态: %v\n", isActive)

	if deactivatedAt.Valid {
		fmt.Printf("  下架时间: %v\n", deactivatedAt.Time.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  下架时间: 未下架\n")
	}

	if lastSeenActive.Valid {
		fmt.Printf("  最后活跃时间: %v\n", lastSeenActive.Time.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  最后活跃时间: 无记录\n")
	}

	fmt.Printf("  创建时间: %v\n", createdAt.Time.Format("2006-01-02 15:04:05"))
	fmt.Printf("  更新时间: %v\n", updatedAt.Time.Format("2006-01-02 15:04:05"))

	// 查询整体统计
	var total, active, inactive int64
	db.QueryRow("SELECT COUNT(*) FROM binance_exchange_info").Scan(&total)
	db.QueryRow("SELECT COUNT(*) FROM binance_exchange_info WHERE is_active = 1").Scan(&active)
	db.QueryRow("SELECT COUNT(*) FROM binance_exchange_info WHERE is_active = 0").Scan(&inactive)

	fmt.Printf("\n📈 整体统计:\n")
	fmt.Printf("  总交易对数: %d\n", total)
	fmt.Printf("  活跃交易对数: %d\n", active)
	fmt.Printf("  非活跃交易对数: %d\n", inactive)

	fmt.Println("\n=== 分析完成 ===")
}
