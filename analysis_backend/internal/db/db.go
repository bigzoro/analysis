package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据库接口
type Database interface {
	// 基础数据库操作
	DB() (*gorm.DB, error)
	GormDB() *gorm.DB
	Close() error
}

// databaseImpl Database接口的实现
type databaseImpl struct {
	gormDB *gorm.DB
}

// NewDatabase 创建数据库实例
func NewDatabase(gormDB *gorm.DB) Database {
	return &databaseImpl{
		gormDB: gormDB,
	}
}

// DB 获取GORM数据库实例
func (d *databaseImpl) DB() (*gorm.DB, error) {
	if d.gormDB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	return d.gormDB, nil
}

// GormDB 获取GORM数据库实例（直接返回，无错误检查）
func (d *databaseImpl) GormDB() *gorm.DB {
	return d.gormDB
}

// Close 关闭数据库连接
func (d *databaseImpl) Close() error {
	if d.gormDB != nil {
		sqlDB, err := d.gormDB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

type Options struct {
	DSN             string
	Automigrate     bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration // 连接最大生存时间
	ConnMaxIdleTime time.Duration // 连接最大空闲时间
}

func OpenMySQL(opt Options) (Database, error) {
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

	// 死锁优化：设置连接最大等待时间，避免长时间等待导致的死锁
	sqlDB.SetConnMaxLifetime(30 * time.Minute)  // 连接最大生存时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)   // 连接最大空闲时间

	// 优化查询超时设置，减少死锁等待时间
	gdb.Exec("SET SESSION innodb_lock_wait_timeout = 10")     // InnoDB锁等待超时10秒
	gdb.Exec("SET SESSION transaction_isolation = 'READ-COMMITTED'") // 使用读已提交隔离级别，减少锁竞争

	if opt.Automigrate {
		// 使用 Set 方法确保字段会被添加/修改
		if err := gdb.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(
			&PortfolioSnapshot{},
			&Holding{},
			&WeeklyFlow{},
			&DailyFlow{},
			&TransferEvent{},
			&TransferCursor{},
			&ArkhamWatch{},
			&WhaleWatch{},
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
			&RecommendationPerformance{},
			&MarketKline{},
			&PriceCache{},
			&TechnicalIndicatorsCache{},
			&FeatureCache{},
			&MLModel{},
			&AutoExecuteSettings{},
			&CoinCapAssetMapping{},
			&CoinCapMarketData{},
			&StrategyExecution{},
			&StrategyExecutionStep{},
			&BinanceExchangeInfo{},
			&BinanceFuturesContract{},
			&BinanceFundingRate{},
			&BinanceOrderBookDepth{},
			&Binance24hStats{},
			&Binance24hStatsHistory{}, // 24小时统计数据历史表
			&BinanceTrade{},
			&FilterCorrection{},
			// 新增的系统完善相关的表
			&ExternalOperation{}, // 外部操作记录
			&OperationLog{},      // 操作日志记录
			&AuditTrail{},        // 审计追踪记录
		); err != nil {
			return nil, fmt.Errorf("AutoMigrate failed: %w", err)
		}

		// 创建优化索引（如果函数存在）
		if err := CreateOptimizedIndexes(gdb); err != nil {
			// 索引创建失败不影响启动，只记录日志
			// 这里不记录日志，因为可能函数不存在
		}
	}
	return NewDatabase(gdb), nil
}
