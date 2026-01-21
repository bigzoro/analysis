package integration

import (
	"testing"

	"analysis/internal/db"
	"analysis/internal/server/strategy/factory"
	"analysis/internal/server/strategy/router"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RouterFactoryTestSuite 路由器和工厂集成测试套件
type RouterFactoryTestSuite struct {
	IntegrationTestSuite
}

// TestRouterFactory_Integration 测试路由器和工厂的集成
func (suite *RouterFactoryTestSuite) TestRouterFactory_Integration() {
	ts := suite.GetTestServer()

	// 测试路由器和工厂的协作
	router := ts.server.StrategyRouter
	factory := ts.server.StrategyFactory

	suite.NotNil(router, "Strategy router should be initialized")
	suite.NotNil(factory, "Strategy factory should be initialized")

	// 测试不同策略类型的路由和工厂集成
	testCases := []struct {
		name         string
		conditions   db.StrategyConditions
		expectedType string
	}{
		{
			name: "traditional strategy",
			conditions: db.StrategyConditions{
				ShortOnGainers: true,
			},
			expectedType: "traditional",
		},
		{
			name: "moving average strategy",
			conditions: db.StrategyConditions{
				MovingAverageEnabled: true,
			},
			expectedType: "moving_average",
		},
		{
			name: "mean reversion strategy",
			conditions: db.StrategyConditions{
				MeanReversionEnabled: true,
			},
			expectedType: "mean_reversion",
		},
		{
			name: "arbitrage strategy",
			conditions: db.StrategyConditions{
				TriangleArbEnabled: true,
			},
			expectedType: "arbitrage",
		},
		{
			name: "grid trading strategy",
			conditions: db.StrategyConditions{
				GridTradingEnabled: true,
			},
			expectedType: "grid_trading",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 路由器选择路由
			route := router.SelectRoute(tc.conditions)

			if tc.expectedType != "" {
				suite.NotNil(route, "Route should be found for %s", tc.name)
				assert.Equal(suite.T(), tc.expectedType, route.StrategyType, "Route type should match")

				// 工厂创建执行器
				executor, config, err := factory.CreateExecutor(route.StrategyType, tc.conditions)

				suite.NoError(err, "Factory should create executor without error")
				suite.NotNil(executor, "Executor should not be nil")
				suite.NotNil(config, "Config should not be nil")

				// 验证执行器类型
				assert.Equal(suite.T(), tc.expectedType, executor.GetStrategyType(), "Executor type should match")
			} else {
				suite.Nil(route, "Route should not be found for invalid conditions")
			}
		})
	}
}

// TestRouterFactory_PriorityHandling 测试优先级处理
func (suite *RouterFactoryTestSuite) TestRouterFactory_PriorityHandling() {
	ts := suite.GetTestServer()
	router := ts.server.StrategyRouter

	// 测试高优先级策略的优先选择
	conditions := db.StrategyConditions{
		MeanReversionEnabled: true, // 最高优先级 (100)
		MovingAverageEnabled: true, // 高优先级 (90)
		ShortOnGainers:       true, // 中优先级 (70)
		TriangleArbEnabled:   true, // 低优先级 (80)
		GridTradingEnabled:   true, // 默认优先级 (10)
	}

	route := router.SelectRoute(conditions)
	suite.NotNil(route, "Route should be found")
	assert.Equal(suite.T(), "mean_reversion", route.StrategyType, "Highest priority strategy should be selected")
}

// TestRouterFactory_ConfigValidation 测试配置验证
func (suite *RouterFactoryTestSuite) TestRouterFactory_ConfigValidation() {
	ts := suite.GetTestServer()
	router := ts.server.StrategyRouter
	factory := ts.server.StrategyFactory

	// 测试无效配置
	invalidConditions := db.StrategyConditions{
		ShortMAPeriod:        0, // 无效：周期不能为0
		LongMAPeriod:         0, // 无效：周期不能为0
		MovingAverageEnabled: true,
	}

	route := router.SelectRoute(invalidConditions)
	if route != nil {
		// 如果路由器选择了路由，验证工厂是否能处理
		_, _, err := factory.CreateExecutor(route.StrategyType, invalidConditions)
		// 工厂可能仍然创建执行器，但执行时会失败
		suite.NoError(err, "Factory should still create executor even with invalid config")
	}
}

// TestRouterFactory_ErrorHandling 测试错误处理
func (suite *RouterFactoryTestSuite) TestRouterFactory_ErrorHandling() {
	ts := suite.GetTestServer()
	factory := ts.server.StrategyFactory

	// 测试不存在的策略类型
	_, _, err := factory.CreateExecutor("nonexistent_strategy", db.StrategyConditions{})
	suite.Error(err, "Factory should return error for nonexistent strategy type")
}

// TestRouterFactory_AllRoutes 测试所有路由
func (suite *RouterFactoryTestSuite) TestRouterFactory_AllRoutes() {
	ts := suite.GetTestServer()
	router := ts.server.StrategyRouter
	factory := ts.server.StrategyFactory

	routes := router.GetAllRoutes()
	suite.NotEmpty(routes, "Router should have routes")

	// 测试每个路由都能创建执行器
	for _, route := range routes {
		suite.Run(route.StrategyType, func() {
			// 创建合适的测试条件
			conditions := suite.createTestConditionsForStrategy(route.StrategyType)

			executor, config, err := factory.CreateExecutor(route.StrategyType, conditions)

			suite.NoError(err, "Factory should create executor for route %s", route.StrategyType)
			suite.NotNil(executor, "Executor should not be nil for route %s", route.StrategyType)
			suite.NotNil(config, "Config should not be nil for route %s", route.StrategyType)

			assert.Equal(suite.T(), route.StrategyType, executor.GetStrategyType(),
				"Executor type should match route type")
		})
	}
}

// createTestConditionsForStrategy 为指定策略类型创建测试条件
func (suite *RouterFactoryTestSuite) createTestConditionsForStrategy(strategyType string) db.StrategyConditions {
	switch strategyType {
	case "traditional":
		return db.StrategyConditions{
			ShortOnGainers:     true,
			GainersRankLimit:   10,
			MarketCapLimitShort: 1000000,
			ShortMultiplier:    1.0,
		}
	case "moving_average":
		return db.StrategyConditions{
			MovingAverageEnabled: true,
			MAType:               "SMA",
			ShortMAPeriod:        5,
			LongMAPeriod:         20,
			MACrossSignal:        "GOLDEN_CROSS",
		}
	case "mean_reversion":
		return db.StrategyConditions{
			MeanReversionEnabled: true,
			MRPeriod:             20,
			MRMinReversionStrength: 2.0,
			MRBollingerMultiplier: 2.0,
		}
	case "arbitrage":
		return db.StrategyConditions{
			TriangleArbEnabled: true,
			SpotFutureSpread:   0.5,
		}
	case "grid_trading":
		return db.StrategyConditions{
			GridTradingEnabled: true,
		}
	default:
		return db.StrategyConditions{}
	}
}

// TestRouterFactory_ConfigBuilders 测试配置构建器
func (suite *RouterFactoryTestSuite) TestRouterFactory_ConfigBuilders() {
	ts := suite.GetTestServer()
	router := ts.server.StrategyRouter

	conditions := db.StrategyConditions{
		ShortOnGainers:       true,
		LongOnSmallGainers:   true,
		GainersRankLimit:     10,
		MovingAverageEnabled: true,
		MAType:               "SMA",
		ShortMAPeriod:        5,
		LongMAPeriod:         20,
		MeanReversionEnabled: true,
		MRPeriod:             20,
		TriangleArbEnabled:   true,
		SpotFutureSpread:     0.5,
		GridTradingEnabled:   true,
	}

	routes := router.GetAllRoutes()

	for _, route := range routes {
		suite.Run("config_"+route.StrategyType, func() {
			config := route.ConfigBuilder(conditions)
			suite.NotNil(config, "Config builder should return non-nil config for %s", route.StrategyType)
		})
	}
}

// TestRouterFactory_MarketDataBuilders 测试市场数据构建器
func (suite *RouterFactoryTestSuite) TestRouterFactory_MarketDataBuilders() {
	ts := suite.GetTestServer()
	router := ts.server.StrategyRouter

	inputData := router.StrategyMarketData{
		Symbol:      "BTCUSDT",
		MarketCap:   1000000.0,
		GainersRank: 5,
		HasSpot:     true,
		HasFutures:  true,
	}

	routes := router.GetAllRoutes()

	for _, route := range routes {
		suite.Run("market_data_"+route.StrategyType, func() {
			marketData := route.MarketDataBuilder(inputData)
			suite.NotNil(marketData, "Market data builder should return non-nil data for %s", route.StrategyType)

			// 验证基本字段
			assert.Equal(suite.T(), inputData.Symbol, marketData.Symbol, "Symbol should match")
			assert.Equal(suite.T(), inputData.MarketCap, marketData.MarketCap, "MarketCap should match")
		})
	}
}

// TestRouterFactory_ContextBuilders 测试上下文构建器
func (suite *RouterFactoryTestSuite) TestRouterFactory_ContextBuilders() {
	ts := suite.GetTestServer()
	router := ts.server.StrategyRouter

	symbol := "BTCUSDT"
	userID := uint(123)
	strategyID := uint(456)

	routes := router.GetAllRoutes()

	for _, route := range routes {
		suite.Run("context_"+route.StrategyType, func() {
			context := route.ContextBuilder(symbol, route.StrategyType, userID, strategyID)
			suite.NotNil(context, "Context builder should return non-nil context for %s", route.StrategyType)

			// 验证基本字段
			assert.Equal(suite.T(), symbol, context.Symbol, "Symbol should match")
			assert.Equal(suite.T(), route.StrategyType, context.StrategyType, "StrategyType should match")
			assert.Equal(suite.T(), userID, context.UserID, "UserID should match")
			suite.NotZero(context.Timestamp, "Timestamp should be set")
			suite.Contains(context.RequestID, route.StrategyType, "RequestID should contain strategy type")
		})
	}
}

// TestRouterFactorySuite 运行路由器和工厂集成测试套件
func TestRouterFactorySuite(t *testing.T) {
	suite.Run(t, new(RouterFactoryTestSuite))
}