package integration

import (
	"testing"

	pdb "analysis/internal/db"
	"analysis/internal/server/strategy/factory"
	"analysis/internal/server/strategy/router"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RouterFactoryIntegrationTestSuite 路由器和工厂集成测试套件
type RouterFactoryIntegrationTestSuite struct {
	suite.Suite
}

// TestRouterFactoryIntegration 测试路由器和工厂的集成
func (suite *RouterFactoryIntegrationTestSuite) TestRouterFactoryIntegration() {
	// 创建路由器
	r := router.NewStrategyRouter()
	assert.NotNil(suite.T(), r, "Strategy router should be initialized")

	// 创建工厂（使用nil作为服务器参数进行基本测试）
	f := factory.NewStrategyFactory(nil)
	assert.NotNil(suite.T(), f, "Strategy factory should be initialized")

	// 测试路由器可以选择路由
	testCases := []struct {
		name         string
		conditions   pdb.StrategyConditions
		expectRoute  bool
		expectedType string
	}{
		{
			name: "traditional strategy",
			conditions: pdb.StrategyConditions{
				ShortOnGainers: true,
			},
			expectRoute:  true,
			expectedType: "traditional",
		},
		{
			name: "moving average strategy",
			conditions: pdb.StrategyConditions{
				MovingAverageEnabled: true,
			},
			expectRoute:  true,
			expectedType: "moving_average",
		},
		{
			name: "mean reversion strategy",
			conditions: pdb.StrategyConditions{
				MeanReversionEnabled: true,
			},
			expectRoute:  true,
			expectedType: "mean_reversion",
		},
		{
			name: "no strategy enabled",
			conditions: pdb.StrategyConditions{},
			expectRoute:  false,
			expectedType: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 路由器选择路由
			route := r.SelectRoute(tc.conditions)

			if tc.expectRoute {
				assert.NotNil(suite.T(), route, "Route should be found for %s", tc.name)
				assert.Equal(suite.T(), tc.expectedType, route.StrategyType, "Route type should match")
			} else {
				assert.Nil(suite.T(), route, "Route should not be found for %s", tc.name)
			}
		})
	}
}

// TestRouterPriority 测试路由器优先级
func (suite *RouterFactoryIntegrationTestSuite) TestRouterPriority() {
	r := router.NewStrategyRouter()

	// 测试高优先级策略的优先选择
	conditions := pdb.StrategyConditions{
		MeanReversionEnabled: true, // 最高优先级 (100)
		MovingAverageEnabled: true, // 高优先级 (90)
		ShortOnGainers:       true, // 中优先级 (70)
		TriangleArbEnabled:   true, // 低优先级 (80)
		GridTradingEnabled:   true, // 默认优先级 (10)
	}

	route := r.SelectRoute(conditions)
	assert.NotNil(suite.T(), route, "Route should be found")
	assert.Equal(suite.T(), "mean_reversion", route.StrategyType, "Highest priority strategy should be selected")
}

// TestRouterAllRoutes 测试所有路由
func (suite *RouterFactoryIntegrationTestSuite) TestRouterAllRoutes() {
	r := router.NewStrategyRouter()
	routes := r.GetAllRoutes()

	assert.NotEmpty(suite.T(), routes, "Router should have routes")

	// 验证路由的基本属性
	for _, route := range routes {
		assert.NotEmpty(suite.T(), route.StrategyType, "Route should have strategy type")
		assert.Greater(suite.T(), route.Priority, 0, "Route should have positive priority")
		assert.NotNil(suite.T(), route.ConditionCheck, "Route should have condition check function")
		assert.NotNil(suite.T(), route.ConfigBuilder, "Route should have config builder function")
		assert.NotNil(suite.T(), route.MarketDataBuilder, "Route should have market data builder function")
		assert.NotNil(suite.T(), route.ContextBuilder, "Route should have context builder function")
	}
}

// TestFactoryErrorHandling 测试工厂错误处理
func (suite *RouterFactoryIntegrationTestSuite) TestFactoryErrorHandling() {
	f := factory.NewStrategyFactory(nil)

	// 测试不存在的策略类型
	_, _, err := f.CreateExecutor("nonexistent_strategy", pdb.StrategyConditions{})
	assert.Error(suite.T(), err, "Factory should return error for nonexistent strategy type")
}

// TestRouterFactoryIntegrationSuite 运行路由器和工厂集成测试套件
func TestRouterFactoryIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RouterFactoryIntegrationTestSuite))
}