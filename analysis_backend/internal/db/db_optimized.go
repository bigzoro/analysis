package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ==================== 优化后的数据库连接 ====================

// OptimizedOptions 优化后的数据库选项
type OptimizedOptions struct {
	DSN          string
	Automigrate  bool
	MaxOpenConns int
	MaxIdleConns int
	// 新增选项
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SlowThreshold   time.Duration // 慢查询阈值
	LogLevel        logger.LogLevel
}

// DefaultOptimizedOptions 默认优化选项
func DefaultOptimizedOptions() OptimizedOptions {
	return OptimizedOptions{
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		SlowThreshold:   1 * time.Second,
		LogLevel:        logger.Warn,
	}
}

// OpenMySQLOptimized 打开优化后的 MySQL 连接
func OpenMySQLOptimized(opt OptimizedOptions) (*gorm.DB, error) {
	// 配置日志
	config := &gorm.Config{
		Logger: logger.Default.LogMode(opt.LogLevel),
		// 禁用默认事务（提高性能）
		SkipDefaultTransaction: false,
		// 准备语句缓存
		PrepareStmt: true,
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
	if opt.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(opt.ConnMaxLifetime)
	}
	if opt.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(opt.ConnMaxIdleTime)
	}

	// 自动迁移
	if opt.Automigrate {
		if err := gdb.AutoMigrate(
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
			return nil, err
		}

		// 创建优化索引
		if err := CreateOptimizedIndexes(gdb); err != nil {
			// 索引创建失败不影响启动，只记录日志
			// log.Printf("Warning: failed to create optimized indexes: %v", err)
		}
	}

	return gdb, nil
}

// ==================== 数据库健康检查 ====================

// HealthCheck 数据库健康检查
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 检查连接
	if err := sqlDB.Ping(); err != nil {
		return err
	}

	// 检查连接池状态
	stats := sqlDB.Stats()
	if stats.OpenConnections >= stats.MaxOpenConnections {
		return fmt.Errorf("connection pool exhausted: %d/%d", stats.OpenConnections, stats.MaxOpenConnections)
	}

	return nil
}

// GetConnectionStats 获取连接池统计信息
func GetConnectionStats(db *gorm.DB) (map[string]interface{}, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}, nil
}
