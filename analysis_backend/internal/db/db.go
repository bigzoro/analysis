package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Options struct {
	DSN             string
	Automigrate     bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration // 连接最大生存时间
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

func OpenMySQL(opt Options) (*gorm.DB, error) {
	// 配置日志和准备语句缓存（优化性能）
	config := &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Warn),
		PrepareStmt:            true, // 启用准备语句缓存，提高性能
		SkipDefaultTransaction: false,
	}

	gdb, err := gorm.Open(mysql.Open(opt.DSN), config)
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}

	// 优化连接池设置
	if opt.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(opt.MaxOpenConns)
	}
	if opt.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(opt.MaxIdleConns)
	}

	// 连接最大生存时间（默认30分钟）
	if opt.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(opt.ConnMaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}

	// 连接最大空闲时间（默认10分钟）
	if opt.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(opt.ConnMaxIdleTime)
	} else {
		sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	}

	if opt.Automigrate {
		// 使用 Set 方法确保字段会被添加/修改
		if err := gdb.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(
			&PortfolioSnapshot{},
			&Holding{},
			&WeeklyFlow{},
			&DailyFlow{},
			&TransferEvent{},
			&TransferCursor{},
			&ScheduledOrder{},
			&BracketLink{},
			&BinanceMarketSnapshot{},
			&BinanceMarketTop{},
			&BinanceSymbolBlacklist{},
			&Announcement{},
			&TwitterPost{},
			&User{},
			&CoinRecommendation{},
			&BacktestRecord{},
			&SimulatedTrade{},
		); err != nil {
			return nil, fmt.Errorf("AutoMigrate failed: %w", err)
		}

		// 创建优化索引（如果函数存在）
		if err := CreateOptimizedIndexes(gdb); err != nil {
			// 索引创建失败不影响启动，只记录日志
			// 这里不记录日志，因为可能函数不存在
		}
	}
	return gdb, nil
}
