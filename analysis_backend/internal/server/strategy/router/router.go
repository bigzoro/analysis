package router

import (
	pdb "analysis/internal/db"
	arb_execution "analysis/internal/server/strategy/arbitrage/execution"
	grid_execution "analysis/internal/server/strategy/grid_trading/execution"
	mr_execution "analysis/internal/server/strategy/mean_reversion/execution"
	ma_execution "analysis/internal/server/strategy/moving_average/execution"
	"analysis/internal/server/strategy/shared/execution"
	traditional_execution "analysis/internal/server/strategy/traditional/execution"
	"fmt"
	"time"
)

// ============================================================================
// 策略路由器 - 替代硬编码的策略选择逻辑
// ============================================================================

// StrategyRouter 策略路由器
type StrategyRouter struct {
	routes []StrategyRoute
}

// StrategyRoute 策略路由规则
type StrategyRoute struct {
	StrategyType      string
	Priority          int // 优先级，越高越优先
	ConditionCheck    func(pdb.StrategyConditions) bool
	ConfigBuilder     func(pdb.StrategyConditions) interface{}
	MarketDataBuilder func(StrategyMarketData) *execution.MarketData
	ContextBuilder    func(symbol, strategyType string, userID uint, strategyID uint) *execution.ExecutionContext
}

// StrategyMarketData 策略市场数据（兼容现有接口）
type StrategyMarketData struct {
	Symbol      string  `json:"symbol"`
	MarketCap   float64 `json:"market_cap"`
	GainersRank int     `json:"gainers_rank"`
	HasSpot     bool    `json:"has_spot"`
	HasFutures  bool    `json:"has_futures"`
}

// NewStrategyRouter 创建策略路由器
func NewStrategyRouter() *StrategyRouter {
	router := &StrategyRouter{
		routes: make([]StrategyRoute, 0),
	}
	router.registerRoutes()
	return router
}

// Register 注册策略路由
func (r *StrategyRouter) Register(route StrategyRoute) {
	r.routes = append(r.routes, route)
	// 按优先级排序
	r.sortRoutes()
}

// SelectRoute 根据策略条件选择路由
func (r *StrategyRouter) SelectRoute(conditions pdb.StrategyConditions) *StrategyRoute {
	for _, route := range r.routes {
		if route.ConditionCheck(conditions) {
			return &route
		}
	}
	return nil
}

// GetAllRoutes 获取所有路由（用于调试）
func (r *StrategyRouter) GetAllRoutes() []StrategyRoute {
	return r.routes
}

// registerRoutes 注册所有内置路由规则
func (r *StrategyRouter) registerRoutes() {
	// 均值回归策略 - 最高优先级
	r.Register(StrategyRoute{
		StrategyType: "mean_reversion",
		Priority:     100,
		ConditionCheck: func(c pdb.StrategyConditions) bool {
			return c.MeanReversionEnabled
		},
		ConfigBuilder:     r.buildMeanReversionConfig,
		MarketDataBuilder: r.buildExecutionMarketData,
		ContextBuilder:    r.buildExecutionContext,
	})

	// 均线策略
	r.Register(StrategyRoute{
		StrategyType: "moving_average",
		Priority:     90,
		ConditionCheck: func(c pdb.StrategyConditions) bool {
			return c.MovingAverageEnabled
		},
		ConfigBuilder:     r.buildMovingAverageConfig,
		MarketDataBuilder: r.buildExecutionMarketData,
		ContextBuilder:    r.buildExecutionContext,
	})

	// 套利策略
	r.Register(StrategyRoute{
		StrategyType: "arbitrage",
		Priority:     80,
		ConditionCheck: func(c pdb.StrategyConditions) bool {
			return c.CrossExchangeArbEnabled || c.SpotFutureArbEnabled ||
				c.TriangleArbEnabled || c.StatArbEnabled || c.FuturesSpotArbEnabled
		},
		ConfigBuilder:     r.buildArbitrageConfig,
		MarketDataBuilder: r.buildExecutionMarketData,
		ContextBuilder:    r.buildExecutionContext,
	})

	// 传统策略
	r.Register(StrategyRoute{
		StrategyType: "traditional",
		Priority:     70,
		ConditionCheck: func(c pdb.StrategyConditions) bool {
			return c.ShortOnGainers || c.LongOnSmallGainers || c.FuturesPriceShortStrategyEnabled
		},
		ConfigBuilder:     r.buildTraditionalConfig,
		MarketDataBuilder: r.buildExecutionMarketData,
		ContextBuilder:    r.buildExecutionContext,
	})

	// 网格交易策略 - 默认策略，最低优先级
	r.Register(StrategyRoute{
		StrategyType: "grid_trading",
		Priority:     10,
		ConditionCheck: func(c pdb.StrategyConditions) bool {
			return c.GridTradingEnabled
		},
		ConfigBuilder:     r.buildGridTradingConfig,
		MarketDataBuilder: r.buildExecutionMarketData,
		ContextBuilder:    r.buildExecutionContext,
	})
}

// 排序路由（按优先级降序）
func (r *StrategyRouter) sortRoutes() {
	for i := 0; i < len(r.routes)-1; i++ {
		for j := i + 1; j < len(r.routes); j++ {
			if r.routes[i].Priority < r.routes[j].Priority {
				r.routes[i], r.routes[j] = r.routes[j], r.routes[i]
			}
		}
	}
}

// ============================================================================
// 配置构建器方法
// ============================================================================

// buildTraditionalConfig 构建传统策略配置
func (r *StrategyRouter) buildTraditionalConfig(conditions pdb.StrategyConditions) interface{} {
	return &traditional_execution.TraditionalExecutionConfig{
		ExecutionConfig:               execution.ExecutionConfig{Enabled: true},
		ShortOnGainers:                conditions.ShortOnGainers,
		LongOnSmallGainers:            conditions.LongOnSmallGainers,
		GainersRankLimit:              conditions.GainersRankLimit,
		LongGainersRankLimit:          conditions.GainersRankLimitLong,
		MarketCapLimitShort:           conditions.MarketCapLimitShort * 10000, // 转换为万元
		MarketCapLimitLong:            conditions.MarketCapLimitLong * 10000,  // 转换为万元
		ShortMultiplier:               conditions.ShortMultiplier,
		LongMultiplier:                conditions.LongMultiplier,
		FuturesPriceRankFilterEnabled: conditions.FuturesPriceRankFilterEnabled,
		MaxFuturesPriceRank:           conditions.MaxFuturesPriceRank,
		TradingType:                   conditions.TradingType,
		// 新增：合约涨幅开空策略
		FuturesPriceShortStrategyEnabled: conditions.FuturesPriceShortStrategyEnabled,
		FuturesPriceShortMaxRank:         conditions.FuturesPriceShortMaxRank,
		FuturesPriceShortMinFundingRate:  conditions.FuturesPriceShortMinFundingRate,
		FuturesPriceShortLeverage:        conditions.FuturesPriceShortLeverage,

		// 保证金损失止损
		EnableMarginLossStopLoss:  conditions.EnableMarginLossStopLoss,
		MarginLossStopLossPercent: conditions.MarginLossStopLossPercent,

		// 保证金盈利止盈
		EnableMarginProfitTakeProfit:  conditions.EnableMarginProfitTakeProfit,
		MarginProfitTakeProfitPercent: conditions.MarginProfitTakeProfitPercent,

		// 杠杆配置
		EnableLeverage:  conditions.EnableLeverage,
		DefaultLeverage: conditions.DefaultLeverage,
		MaxLeverage:     conditions.MaxLeverage,
	}
}

// buildMovingAverageConfig 构建均线策略配置
func (r *StrategyRouter) buildMovingAverageConfig(conditions pdb.StrategyConditions) interface{} {
	return &ma_execution.MovingAverageExecutionConfig{
		ExecutionConfig:      execution.ExecutionConfig{Enabled: true},
		MovingAverageEnabled: conditions.MovingAverageEnabled,
		MAType:               conditions.MAType,
		ShortMAPeriod:        conditions.ShortMAPeriod,
		LongMAPeriod:         conditions.LongMAPeriod,
		MACrossSignal:        conditions.MACrossSignal,
		MATrendFilter:        conditions.MATrendFilter,
		MATrendDirection:     conditions.MATrendDirection,
		MASignalMode:         conditions.MASignalMode,
		LongMultiplier:       1.0, // 默认值
		ShortMultiplier:      1.0, // 默认值
	}
}

// buildArbitrageConfig 构建套利策略配置
func (r *StrategyRouter) buildArbitrageConfig(conditions pdb.StrategyConditions) interface{} {
	return &arb_execution.ArbitrageExecutionConfig{
		ExecutionConfig:         execution.ExecutionConfig{Enabled: true},
		CrossExchangeArbEnabled: conditions.CrossExchangeArbEnabled,
		SpotFutureArbEnabled:    conditions.SpotFutureArbEnabled,
		TriangleArbEnabled:      conditions.TriangleArbEnabled,
		StatisticalArbEnabled:   conditions.StatArbEnabled,
		FuturesSpotArbEnabled:   conditions.FuturesSpotArbEnabled,
		MinProfitThreshold:      conditions.SpotFutureSpread,
	}
}

// buildGridTradingConfig 构建网格交易策略配置
func (r *StrategyRouter) buildGridTradingConfig(conditions pdb.StrategyConditions) interface{} {
	return &grid_execution.GridTradingExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		// 网格交易配置可以根据需要扩展
	}
}

// buildMeanReversionConfig 构建均值回归策略配置
func (r *StrategyRouter) buildMeanReversionConfig(conditions pdb.StrategyConditions) interface{} {
	return &mr_execution.MeanReversionExecutionConfig{
		ExecutionConfig:         execution.ExecutionConfig{Enabled: true},
		MeanReversionEnabled:    conditions.MeanReversionEnabled,
		MRBollingerBandsEnabled: conditions.MRBollingerBandsEnabled,
		MeanReversionLookback:   conditions.MRPeriod,
		MeanReversionThreshold:  conditions.MRMinReversionStrength,
		MeanReversionStdDev:     conditions.MRBollingerMultiplier,
		BollingerPeriod:         conditions.MRChannelPeriod,
		BollingerStdDev:         conditions.MRBollingerMultiplier,
		LongMultiplier:          1.0, // 默认值
		ShortMultiplier:         1.0, // 默认值
	}
}

// ============================================================================
// 数据构建器方法
// ============================================================================

// buildExecutionMarketData 构建执行市场数据
func (r *StrategyRouter) buildExecutionMarketData(marketData StrategyMarketData) *execution.MarketData {
	// 根据交易类型决定市场数据可用性
	hasSpot := marketData.HasSpot
	hasFutures := marketData.HasFutures

	// 注意：这里的marketData是路由器的StrategyMarketData类型
	// 实际的交易类型过滤将在执行器中根据StrategyConditions进行

	return &execution.MarketData{
		Symbol:      marketData.Symbol,
		Price:       0, // 将通过MarketDataProvider获取实时价格
		Volume:      0, // 将通过MarketDataProvider获取实时成交量
		MarketCap:   marketData.MarketCap,
		GainersRank: marketData.GainersRank,
		HasSpot:     hasSpot,
		HasFutures:  hasFutures,
	}
}

// buildExecutionContext 构建执行上下文
func (r *StrategyRouter) buildExecutionContext(symbol, strategyType string, userID uint, strategyID uint) *execution.ExecutionContext {
	return &execution.ExecutionContext{
		Symbol:       symbol,
		StrategyType: strategyType,
		UserID:       userID,
		RequestID:    fmt.Sprintf("%s-%d-%s", strategyType, strategyID, symbol),
		Timestamp:    time.Now(),
	}
}
