package db

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Options struct {
	DSN          string
	Automigrate  bool
	MaxOpenConns int
	MaxIdleConns int
}

func OpenMySQL(opt Options) (*gorm.DB, error) {
	gdb, err := gorm.Open(mysql.Open(opt.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	if opt.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(opt.MaxOpenConns)
	}
	if opt.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(opt.MaxIdleConns)
	}
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if opt.Automigrate {
                if err := gdb.AutoMigrate(
                        &PortfolioSnapshot{},
                        &Holding{},
                        &WeeklyFlow{},
                        &DailyFlow{},
                        &TransferEvent{},
                        &ScheduledOrder{},
                        &BracketLink{},
                        &BinanceMarketSnapshot{},
                        &BinanceMarketTop{},
                        &BinanceAnnouncement{},
                ); err != nil {
                        return nil, err
                }
        }
	return gdb, nil
}
