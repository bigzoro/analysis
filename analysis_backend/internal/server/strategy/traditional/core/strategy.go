package core

import (
	pdb "analysis/internal/db"
	"analysis/internal/server/strategy/traditional"
	"analysis/internal/server/strategy/traditional/config"
	"analysis/internal/server/strategy/traditional/scanning"
	"context"

	"gorm.io/gorm"
)

// Strategy 传统策略实现
type Strategy struct {
	configManager traditional.ConfigManager
	scanner       *scanning.Scanner
	db            interface{} // 数据库连接，用于创建用户特定的扫描器
}

// NewStrategy 创建传统策略（已废弃，请使用NewStrategyWithDB）
func NewStrategy(db interface{}) *Strategy {
	return NewStrategyWithDB(db)
}

// NewStrategyWithDB 创建带有数据库连接的传统策略
func NewStrategyWithDB(db interface{}) *Strategy {
	configManager := config.NewManager()
	// 创建一个基础scanner，用于ToStrategyScanner方法
	// 注意：这个scanner的db字段会在ToStrategyScanner中被正确设置
	var scannerDB *gorm.DB
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			scannerDB = gormDB
		}
	}
	scanner := scanning.NewScanner(scannerDB, 0)

	return &Strategy{
		configManager: configManager,
		scanner:       scanner,
		db:            db,
	}
}

// Scan 执行传统策略扫描
func (s *Strategy) Scan(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	return s.scanner.Scan(ctx, config)
}

// IsEnabled 检查策略是否启用
func (s *Strategy) IsEnabled(config *traditional.TraditionalConfig) bool {
	return config.Enabled
}

// ExecuteShortOnGainers 执行涨幅开空策略
func (s *Strategy) ExecuteShortOnGainers(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	return s.scanner.ExecuteShortOnGainers(ctx, config)
}

// ExecuteLongOnSmallGainers 执行小幅上涨开多策略
func (s *Strategy) ExecuteLongOnSmallGainers(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	return s.scanner.ExecuteLongOnSmallGainers(ctx, config)
}

// ToStrategyScanner 创建适配器
func (s *Strategy) ToStrategyScanner() interface{} {
	// 使用scanning包中正确实现的适配器，避免代码重复和bug
	return s.scanner.ToStrategyScanner()
}

// ============================================================================
// 注册表
// ============================================================================

var globalStrategy *Strategy
var globalDatabase pdb.Database

// GetTraditionalStrategy 获取传统策略实例
func GetTraditionalStrategy(db interface{}) *Strategy {
	if globalStrategy == nil {
		globalStrategy = NewStrategyWithDB(db)
	}
	return globalStrategy
}
