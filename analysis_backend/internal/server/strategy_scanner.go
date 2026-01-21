package server

import (
	"context"
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

// ============================================================================
// 策略扫描器核心 - 批量筛选符合条件的交易对
// ============================================================================

// 符合条件的交易对信息
type EligibleSymbol struct {
	Symbol      string  `json:"symbol"`
	Action      string  `json:"action"`
	Reason      string  `json:"reason"`
	Multiplier  float64 `json:"multiplier"`
	MarketCap   float64 `json:"market_cap"`
	GainersRank int     `json:"gainers_rank"`
	// 三角套利专用字段
	TrianglePath []string `json:"triangle_path,omitempty"` // 三角套利路径
	PriceDiff    float64  `json:"price_diff,omitempty"`    // 价差百分比
	// 风险管理字段
	StopLossPrice   float64 `json:"stop_loss_price,omitempty"`   // 止损价格
	TakeProfitPrice float64 `json:"take_profit_price,omitempty"` // 止盈价格
	MaxPositionSize float64 `json:"max_position_size,omitempty"` // 最大仓位比例
	MaxHoldHours    int     `json:"max_hold_hours,omitempty"`    // 最大持仓小时数
	RiskLevel       float64 `json:"risk_level,omitempty"`        // 风险等级 (0-1)
}

// 策略扫描器接口
type StrategyScanner interface {
	Scan(ctx context.Context, strategy *pdb.TradingStrategy) ([]interface{}, error)
	GetStrategyType() string
}

// ============================================================================
// 策略扫描器注册表
// ============================================================================

// 策略扫描器注册表
type StrategyScannerRegistry struct {
	scanners map[string]StrategyScanner
}

// 创建扫描器注册表
func NewStrategyScannerRegistry() *StrategyScannerRegistry {
	registry := &StrategyScannerRegistry{
		scanners: make(map[string]StrategyScanner),
	}

	registry.registerScanners()
	return registry
}

// 注册所有策略扫描器
func (r *StrategyScannerRegistry) registerScanners() {
	// 这里将在创建扫描器时动态注册，因为扫描器需要Server实例
}

// 注册扫描器（需要Server实例）
func (r *StrategyScannerRegistry) RegisterScanner(server *Server) error {
	log.Printf("🔄 [StrategyRegistry] ===== 开始注册策略扫描器 =====")

	// 传统策略扫描器 - 使用新的模块化架构
	log.Printf("🔍 [StrategyRegistry] 尝试加载传统策略...")
	newStrategy, err := getNewTraditionalStrategy(server.db.DB())
	if err != nil {
		log.Printf("❌ [StrategyRegistry] 传统策略注册失败: %v", err)
		return fmt.Errorf("注册传统策略失败: %w", err)
	}
	r.scanners["traditional"] = newStrategy
	log.Printf("✅ [StrategyRegistry] 成功注册新的模块化传统策略")

	// 均线策略扫描器 - 使用新的模块化架构
	log.Printf("🔍 [StrategyRegistry] 尝试加载均线策略...")
	newStrategy, err = getNewMovingAverageStrategy()
	if err != nil {
		log.Printf("❌ [StrategyRegistry] 均线策略注册失败: %v", err)
		return fmt.Errorf("注册均线策略失败: %w", err)
	}
	r.scanners["moving_average"] = newStrategy
	log.Printf("✅ [StrategyRegistry] 成功注册新的模块化均线策略")

	// 套利策略扫描器 - 使用新的模块化架构
	log.Printf("🔍 [StrategyRegistry] 尝试加载套利策略...")
	newStrategy, err = getNewArbitrageStrategy()
	if err != nil {
		log.Printf("❌ [StrategyRegistry] 套利策略注册失败: %v", err)
		return fmt.Errorf("注册套利策略失败: %w", err)
	}
	r.scanners["arbitrage"] = newStrategy
	log.Printf("✅ [StrategyRegistry] 成功注册新的模块化套利策略")

	// 均值回归策略扫描器 - 使用新的模块化架构
	log.Printf("🔍 [StrategyRegistry] 尝试加载均值回归策略...")
	newStrategy, err = getNewMeanReversionStrategy(server.db.DB())
	if err != nil {
		log.Printf("❌ [StrategyRegistry] 均值回归策略注册失败: %v", err)
		return fmt.Errorf("注册均值回归策略失败: %w", err)
	}
	r.scanners["mean_reversion"] = newStrategy
	log.Printf("✅ [StrategyRegistry] 成功注册新的模块化均值回归策略")

	// 网格交易策略扫描器 - 使用新的模块化架构
	log.Printf("🔍 [StrategyRegistry] 尝试加载网格交易策略...")
	newStrategy, err = getNewGridTradingStrategy()
	if err != nil {
		log.Printf("❌ [StrategyRegistry] 网格交易策略注册失败: %v", err)
		return fmt.Errorf("注册网格交易策略失败: %w", err)
	}
	r.scanners["grid_trading"] = newStrategy
	log.Printf("✅ [StrategyRegistry] 成功注册新的模块化网格交易策略")

	log.Printf("✅ [StrategyRegistry] 策略扫描器注册完成")
	log.Printf("📋 [StrategyRegistry] 已注册扫描器: %v", getRegisteredScannerTypes(r.scanners))
	log.Printf("🎯 [StrategyRegistry] ===== 注册过程结束 =====")
	return nil
}

// 获取扫描器
func (r *StrategyScannerRegistry) GetScanner(strategyType string) StrategyScanner {
	return r.scanners[strategyType]
}

// 根据策略条件选择合适的扫描器
func (r *StrategyScannerRegistry) SelectScanner(strategy *pdb.TradingStrategy) StrategyScanner {
	conditions := strategy.Conditions

	log.Printf("[SelectScanner] 策略ID: %d, 条件检查:", strategy.ID)
	log.Printf("[SelectScanner] TriangleArb: %v, GridTrading: %v, MovingAverage: %v",
		conditions.TriangleArbEnabled, conditions.GridTradingEnabled, conditions.MovingAverageEnabled)
	log.Printf("[SelectScanner] MeanReversion: %v, ShortOnGainers: %v, LongOnSmallGainers: %v",
		conditions.MeanReversionEnabled, conditions.ShortOnGainers, conditions.LongOnSmallGainers)
	log.Printf("[SelectScanner] 其他套利: CrossExchange=%v, SpotFuture=%v, Stat=%v, FuturesSpot=%v",
		conditions.CrossExchangeArbEnabled, conditions.SpotFutureArbEnabled,
		conditions.StatArbEnabled, conditions.FuturesSpotArbEnabled)

	// 优先检查特殊策略
	if conditions.TriangleArbEnabled {
		log.Printf("[SelectScanner] 选择套利策略 (三角套利)")
		scanner := r.scanners["arbitrage"]
		if scanner == nil {
			log.Printf("[SelectScanner] 套利扫描器未注册!")
		}
		return scanner
	}

	// 检查网格交易策略
	if conditions.GridTradingEnabled {
		log.Printf("[SelectScanner] 选择网格交易策略")
		scanner := r.scanners["grid_trading"]
		if scanner == nil {
			log.Printf("[SelectScanner] 网格交易扫描器未注册!")
		}
		return scanner
	}

	// 检查均线策略
	if conditions.MovingAverageEnabled {
		log.Printf("[SelectScanner] 选择均线策略")
		scanner := r.scanners["moving_average"]
		if scanner == nil {
			log.Printf("[SelectScanner] 均线扫描器未注册!")
		}
		return scanner
	}

	// 检查均值回归策略
	if conditions.MeanReversionEnabled {
		log.Printf("[SelectScanner] 选择均值回归策略")
		scanner := r.scanners["mean_reversion"]
		if scanner == nil {
			log.Printf("[SelectScanner] 均值回归扫描器未注册!")
		}
		return scanner
	}

	// 检查传统策略
	if conditions.ShortOnGainers || conditions.LongOnSmallGainers {
		log.Printf("[SelectScanner] 选择传统策略")
		scanner := r.scanners["traditional"]
		if scanner == nil {
			log.Printf("[SelectScanner] 传统扫描器未注册!")
		}
		return scanner
	}

	// 检查其他套利策略
	if conditions.CrossExchangeArbEnabled || conditions.SpotFutureArbEnabled ||
		conditions.StatArbEnabled || conditions.FuturesSpotArbEnabled {
		log.Printf("[SelectScanner] 选择套利策略 (其他)")
		scanner := r.scanners["arbitrage"]
		if scanner == nil {
			log.Printf("[SelectScanner] 套利扫描器未注册!")
		}
		return scanner
	}

	// 默认使用传统策略扫描器
	log.Printf("[SelectScanner] 使用默认传统策略")
	scanner := r.scanners["traditional"]
	if scanner == nil {
		log.Printf("[SelectScanner] 默认传统扫描器也未注册!")
	}
	return scanner
}

// getRegisteredScannerTypes 获取已注册的扫描器类型列表（用于调试）
func getRegisteredScannerTypes(scanners map[string]StrategyScanner) []string {
	var types []string
	for scannerType, scanner := range scanners {
		if scanner != nil {
			types = append(types, scannerType)
		} else {
			types = append(types, scannerType+"(nil)")
		}
	}
	return types
}
