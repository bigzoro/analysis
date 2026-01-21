package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	"analysis/internal/db"
)

func main() {
	// 加载配置
	var cfg config.Config
	config.MustLoad("analysis_backend/config.yaml", &cfg)
	config.ApplyProxy(&cfg)

	// 初始化数据库连接
	database, err := db.OpenMySQL(db.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     false,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()

	gormDB, err := database.DB()
	if err != nil {
		log.Fatalf("获取GORM数据库实例失败: %v", err)
	}

	// 测试交易对
	testSymbols := []string{
		"BTCUSDT",
		"ETHUSDT",
		"ETHBTC",
		"BNBBTC",
		"ADABTC",
		"SOLBTC",
		"DOGEBTC",
	}

	fmt.Println("测试交易对存在性:")
	for _, symbol := range testSymbols {
		var count int64
		err := gormDB.Table("binance_exchange_info").Where("symbol = ? AND status = ?", symbol, "TRADING").Count(&count).Error
		if err != nil {
			fmt.Printf("%s: 错误 - %v\n", symbol, err)
		} else {
			fmt.Printf("%s: %s (count=%d)\n", symbol, map[bool]string{true: "存在", false: "不存在"}[count > 0], count)
		}
	}
}
