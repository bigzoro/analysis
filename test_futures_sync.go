package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     false,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * 60 * 1000000000,
		ConnMaxIdleTime: 10 * 60 * 1000000000,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// 创建期货同步器
	dataSyncConfig := struct {
		MaxRetries           int  `yaml:"max_retries"`
		RetryDelay           int  `yaml:"retry_delay"`
		BatchSize            int  `yaml:"batch_size"`
		EnableHistoricalSync bool `yaml:"enable_historical_sync"`
	}{
		MaxRetries:           3,
		RetryDelay:           5,
		BatchSize:            100,
		EnableHistoricalSync: false,
	}

	syncer := NewFuturesSyncer(gdb, &cfg, (*DataSyncConfig)(&dataSyncConfig))

	// 执行同步
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("开始测试期货数据同步...")
	startTime := time.Now()

	if err := syncer.Sync(ctx); err != nil {
		log.Fatalf("同步失败: %v", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("同步完成，耗时: %v\n", duration)

	// 检查结果
	var contractCount int64
	if err := gdb.Model(&pdb.BinanceFuturesContract{}).Count(&contractCount).Error; err != nil {
		log.Printf("查询合约表失败: %v", err)
	} else {
		fmt.Printf("期货合约表记录数: %d\n", contractCount)
	}

	var fundingCount int64
	if err := gdb.Model(&pdb.BinanceFundingRate{}).Count(&fundingCount).Error; err != nil {
		log.Printf("查询资金费率表失败: %v", err)
	} else {
		fmt.Printf("资金费率表记录数: %d\n", fundingCount)
	}
}

// 简化的类型定义
type DataSyncConfig struct {
	MaxRetries           int  `yaml:"max_retries"`
	RetryDelay           int  `yaml:"retry_delay"`
	BatchSize            int  `yaml:"batch_size"`
	EnableHistoricalSync bool `yaml:"enable_historical_sync"`
}

// 简化的FuturesSyncer构造函数
func NewFuturesSyncer(db *gorm.DB, cfg *config.Config, config *DataSyncConfig) *FuturesSyncer {
	return &FuturesSyncer{
		db:     db,
		cfg:    cfg,
		config: config,
	}
}
