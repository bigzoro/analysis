package integration

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"analysis/internal/db"

	"github.com/stretchr/testify/suite"
)

// EndToEndTestSuite 端到端集成测试套件
type EndToEndTestSuite struct {
	IntegrationTestSuite
}

// TestEndToEnd_StrategyLifecycle 测试完整的策略生命周期
func (suite *EndToEndTestSuite) TestEndToEnd_StrategyLifecycle() {
	ts := suite.GetTestServer()

	// 1. 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers:       true,
		GainersRankLimit:     10,
		MarketCapLimitShort:  1000000,
		ShortMultiplier:      1.0,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 2. 验证策略已创建
	retrievedStrategy := &db.Strategy{}
	err = ts.db.First(retrievedStrategy, strategy.ID).Error
	suite.NoError(err, "Failed to retrieve strategy from database")
	assert.Equal(suite.T(), strategy.ID, retrievedStrategy.ID, "Strategy ID should match")
	assert.Equal(suite.T(), "traditional", retrievedStrategy.Type, "Strategy type should match")

	// 3. 执行策略扫描
	scanResponse := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)
	suite.AssertResponseStatus(scanResponse, 200)

	// 解析扫描结果
	var scanResult map[string]interface{}
	err = json.Unmarshal(scanResponse.Body.Bytes(), &scanResult)
	suite.NoError(err, "Failed to parse scan response")

	// 4. 执行策略
	execResponse := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)
	suite.AssertResponseStatus(execResponse, 200)

	// 解析执行结果
	var execResult map[string]interface{}
	err = json.Unmarshal(execResponse.Body.Bytes(), &execResult)
	suite.NoError(err, "Failed to parse execution response")

	// 5. 验证执行结果包含必要的字段
	execData, ok := execResult["data"].(map[string]interface{})
	suite.True(ok, "Execution result should contain data")

	suite.Contains(execData, "strategy_id", "Execution result should contain strategy_id")
	suite.Contains(execData, "symbol", "Execution result should contain symbol")
	suite.Contains(execData, "action", "Execution result should contain action")

	// 6. 验证策略ID一致性
	scanStrategyID, _ := scanResult["data"].(map[string]interface{})["strategy_id"].(float64)
	execStrategyID, _ := execData["strategy_id"].(float64)
	assert.Equal(suite.T(), scanStrategyID, execStrategyID, "Strategy ID should be consistent between scan and execution")
	assert.Equal(suite.T(), float64(strategy.ID), execStrategyID, "Strategy ID should match created strategy")
}

// TestEndToEnd_MultipleStrategies 测试多个策略的端到端流程
func (suite *EndToEndTestSuite) TestEndToEnd_MultipleStrategies() {
	ts := suite.GetTestServer()

	// 创建多个不同类型的策略
	strategies := []struct {
		name       string
		strategyType string
		conditions db.StrategyConditions
	}{
		{
			name:       "traditional",
			strategyType: "traditional",
			conditions: db.StrategyConditions{
				ShortOnGainers: true,
				GainersRankLimit: 10,
			},
		},
		{
			name:       "moving_average",
			strategyType: "moving_average",
			conditions: db.StrategyConditions{
				MovingAverageEnabled: true,
				MAType: "SMA",
				ShortMAPeriod: 5,
				LongMAPeriod: 20,
			},
		},
		{
			name:       "mean_reversion",
			strategyType: "mean_reversion",
			conditions: db.StrategyConditions{
				MeanReversionEnabled: true,
				MRPeriod: 20,
			},
		},
	}

	createdStrategies := make([]*db.Strategy, 0, len(strategies))

	// 1. 创建所有策略
	for _, s := range strategies {
		strategy, err := ts.CreateTestStrategy(s.strategyType, s.conditions)
		suite.Require().NoError(err, "Failed to create %s strategy", s.name)
		createdStrategies = append(createdStrategies, strategy)
	}

	// 2. 为每个策略执行扫描和执行
	for i, strategy := range createdStrategies {
		suite.Run(fmt.Sprintf("strategy_%d_%s", i, strategy.Type), func() {
			// 扫描
			scanResponse := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)
			suite.AssertResponseStatus(scanResponse, 200)

			// 执行
			execResponse := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)
			suite.AssertResponseStatus(execResponse, 200)

			// 验证执行结果
			var execResult map[string]interface{}
			err := json.Unmarshal(execResponse.Body.Bytes(), &execResult)
			suite.NoError(err, "Failed to parse execution response for %s", strategy.Type)

			execData, ok := execResult["data"].(map[string]interface{})
			suite.True(ok, "Execution result should contain data for %s", strategy.Type)

			strategyID, ok := execData["strategy_id"].(float64)
			suite.True(ok, "Execution result should contain strategy_id for %s", strategy.Type)
			assert.Equal(suite.T(), float64(strategy.ID), strategyID, "Strategy ID should match for %s", strategy.Type)
		})
	}
}

// TestEndToEnd_ErrorScenarios 测试错误场景的端到端流程
func (suite *EndToEndTestSuite) TestEndToEnd_ErrorScenarios() {
	ts := suite.GetTestServer()

	// 测试不存在的策略ID
	errorScenarios := []struct {
		name         string
		endpoint     string
		expectedCode int
	}{
		{
			name:         "nonexistent_strategy_scan",
			endpoint:     "/strategies/scan-eligible?strategy_id=99999",
			expectedCode: 500,
		},
		{
			name:         "nonexistent_strategy_execute",
			endpoint:     "/strategies/execute/99999",
			expectedCode: 404,
		},
		{
			name:         "invalid_scan_request",
			endpoint:     "/strategies/scan-eligible",
			expectedCode: 400,
		},
	}

	for _, scenario := range errorScenarios {
		suite.Run(scenario.name, func() {
			response := ts.MakeRequest("POST", scenario.endpoint, nil)
			suite.AssertResponseStatus(response, scenario.expectedCode)

			// 验证错误响应包含错误信息
			var result map[string]interface{}
			err := json.Unmarshal(response.Body.Bytes(), &result)
			suite.NoError(err, "Failed to parse error response")

			if scenario.expectedCode >= 400 {
				suite.Contains(result, "error", "Error response should contain error field")
			}
		})
	}
}

// TestEndToEnd_Performance 测试性能表现
func (suite *EndToEndTestSuite) TestEndToEnd_Performance() {
	ts := suite.GetTestServer()

	// 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers: true,
		GainersRankLimit: 10,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err)

	// 执行多次请求测试性能
	numRequests := 50
	startTime := time.Now()

	results := make(chan *httptest.ResponseRecorder, numRequests)

	// 并发执行请求
	for i := 0; i < numRequests; i++ {
		go func() {
			response := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)
			results <- response
		}()
	}

	// 收集结果
	for i := 0; i < numRequests; i++ {
		response := <-results
		suite.AssertResponseStatus(response, 200)
	}

	duration := time.Since(startTime)
	avgDuration := duration / time.Duration(numRequests)

	// 验证平均响应时间在合理范围内（每个请求少于1秒）
	suite.True(avgDuration < time.Second, "Average request duration should be less than 1 second, got %v", avgDuration)

	// 记录性能指标
	suite.T().Logf("Performance test completed: %d requests in %v, average %v per request",
		numRequests, duration, avgDuration)
}

// TestEndToEnd_DataConsistency 测试数据一致性
func (suite *EndToEndTestSuite) TestEndToEnd_DataConsistency() {
	ts := suite.GetTestServer()

	// 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers: true,
		GainersRankLimit: 5,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err)

	// 多次执行相同的策略，确保结果一致性
	numExecutions := 5
	results := make([]map[string]interface{}, numExecutions)

	for i := 0; i < numExecutions; i++ {
		response := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)
		suite.AssertResponseStatus(response, 200)

		var result map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		suite.NoError(err)

		results[i] = result
	}

	// 验证所有执行的策略ID都是一致的
	for i := 1; i < numExecutions; i++ {
		prevData := results[i-1]["data"].(map[string]interface{})
		currData := results[i]["data"].(map[string]interface{})

		prevID := prevData["strategy_id"].(float64)
		currID := currData["strategy_id"].(float64)

		assert.Equal(suite.T(), prevID, currID, "Strategy ID should be consistent across executions")
	}
}

// TestEndToEnd_ResourceCleanup 测试资源清理
func (suite *EndToEndTestSuite) TestEndToEnd_ResourceCleanup() {
	ts := suite.GetTestServer()

	// 创建多个策略
	strategies := make([]*db.Strategy, 10)
	for i := 0; i < 10; i++ {
		strategy, err := ts.CreateTestStrategy("traditional", db.StrategyConditions{
			ShortOnGainers: true,
		})
		suite.Require().NoError(err)
		strategies[i] = strategy
	}

	// 执行所有策略
	for _, strategy := range strategies {
		response := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)
		suite.AssertResponseStatus(response, 200)
	}

	// 验证数据库状态
	var count int64
	err := ts.db.Model(&db.Strategy{}).Where("type = ?", "traditional").Count(&count).Error
	suite.NoError(err)
	suite.Equal(int64(10), count, "All strategies should still exist in database")

	// 清理测试数据（如果需要）
	// 注意：在实际测试中，可能需要清理测试数据以避免影响其他测试
}

// TestEndToEndSuite 运行端到端集成测试套件
func TestEndToEndSuite(t *testing.T) {
	suite.Run(t, new(EndToEndTestSuite))
}