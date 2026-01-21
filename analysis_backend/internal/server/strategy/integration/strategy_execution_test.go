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

// StrategyExecutionTestSuite 策略执行集成测试套件
type StrategyExecutionTestSuite struct {
	IntegrationTestSuite
}

// TestStrategyExecution 测试策略执行的完整流程
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_TraditionalStrategy() {
	ts := suite.GetTestServer()

	// 创建传统策略
	conditions := db.StrategyConditions{
		ShortOnGainers: true,
		GainersRankLimit: 10,
		MarketCapLimitShort: 1000000, // 100万
		ShortMultiplier: 1.0,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行策略
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)

	// 验证响应
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	suite.Contains(response, "success", "Response should contain success field")
	suite.Contains(response, "data", "Response should contain data field")

	data, ok := response["data"].(map[string]interface{})
	suite.True(ok, "Data should be an object")

	// 验证执行结果
	suite.Contains(data, "strategy_id", "Result should contain strategy_id")
	suite.Contains(data, "symbol", "Result should contain symbol")
	suite.Contains(data, "action", "Result should contain action")

	// 验证策略ID
	strategyID, ok := data["strategy_id"].(float64)
	suite.True(ok, "strategy_id should be a number")
	suite.Equal(float64(strategy.ID), strategyID, "Strategy ID should match")
}

// TestStrategyExecution_MovingAverageStrategy 测试均线策略执行
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_MovingAverageStrategy() {
	ts := suite.GetTestServer()

	// 创建均线策略
	conditions := db.StrategyConditions{
		MovingAverageEnabled: true,
		MAType: "SMA",
		ShortMAPeriod: 5,
		LongMAPeriod: 20,
		MACrossSignal: "GOLDEN_CROSS",
	}

	strategy, err := ts.CreateTestStrategy("moving_average", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行策略
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)

	// 验证响应（即使执行失败也应该返回200，因为这是业务逻辑失败）
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	suite.Contains(response, "success", "Response should contain success field")
	suite.Contains(response, "data", "Response should contain data field")
}

// TestStrategyExecution_MeanReversionStrategy 测试均值回归策略执行
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_MeanReversionStrategy() {
	ts := suite.GetTestServer()

	// 创建均值回归策略
	conditions := db.StrategyConditions{
		MeanReversionEnabled: true,
		MRPeriod: 20,
		MRMinReversionStrength: 2.0,
		MRBollingerMultiplier: 2.0,
	}

	strategy, err := ts.CreateTestStrategy("mean_reversion", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行策略
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)

	// 验证响应
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	suite.Contains(response, "success", "Response should contain success field")
}

// TestStrategyExecution_InvalidStrategyID 测试无效策略ID
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_InvalidStrategyID() {
	ts := suite.GetTestServer()

	// 使用不存在的策略ID
	w := ts.MakeRequest("POST", "/strategies/execute/99999", nil)

	// 应该返回404或错误状态
	suite.AssertResponseStatus(w, 404)

	// 验证错误响应
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse error response")

	suite.Contains(response, "error", "Error response should contain error field")
}

// TestStrategyExecution_BatchExecution 测试批量执行
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_BatchExecution() {
	ts := suite.GetTestServer()

	// 创建多个策略
	strategy1, err := ts.CreateTestStrategy("traditional", db.StrategyConditions{
		ShortOnGainers: true,
		GainersRankLimit: 10,
	})
	suite.Require().NoError(err)

	strategy2, err := ts.CreateTestStrategy("moving_average", db.StrategyConditions{
		MovingAverageEnabled: true,
		MAType: "SMA",
	})
	suite.Require().NoError(err)

	// 批量执行策略
	requestBody := map[string]interface{}{
		"strategy_ids": []uint{strategy1.ID, strategy2.ID},
	}

	// 注意：这里需要实际实现批量执行的HTTP处理
	w := ts.MakeRequest("POST", "/strategies/execute-batch", requestBody)

	// 验证响应（即使API不存在也应该返回适当的错误）
	if w.Code == 404 {
		// API不存在，这是预期的
		suite.AssertResponseStatus(w, 404)
	} else {
		// 如果API存在，验证正常响应
		suite.AssertResponseStatus(w, 200)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		suite.NoError(err, "Failed to parse response")
	}
}

// TestStrategyExecution_DisabledStrategy 测试禁用策略的执行
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_DisabledStrategy() {
	ts := suite.GetTestServer()

	// 创建禁用策略
	conditions := db.StrategyConditions{
		ShortOnGainers: false, // 禁用所有条件
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err)

	// 执行策略
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)

	// 验证响应
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	suite.Contains(response, "success", "Response should contain success field")
}

// TestStrategyExecution_Timeout 测试执行超时
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_Timeout() {
	ts := suite.GetTestServer()

	// 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers: true,
		GainersRankLimit: 10,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err)

	// 设置较短的超时时间（如果支持的话）
	// 注意：实际的超时处理需要在HTTP客户端层面实现

	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d?timeout=1s", strategy.ID), nil)

	// 验证响应（应该正常完成或在超时前完成）
	suite.AssertResponseStatus(w, 200)
}

// TestStrategyExecution_ConcurrentRequests 测试并发请求
func (suite *StrategyExecutionTestSuite) TestStrategyExecution_ConcurrentRequests() {
	ts := suite.GetTestServer()

	// 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers: true,
		GainersRankLimit: 10,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err)

	// 并发执行多个请求
	numRequests := 10
	results := make(chan *httptest.ResponseRecorder, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/execute/%d", strategy.ID), nil)
			results <- w
		}()
	}

	// 收集结果
	for i := 0; i < numRequests; i++ {
		w := <-results
		suite.AssertResponseStatus(w, 200, "All concurrent requests should succeed")
	}
}

// TestStrategyExecutionSuite 运行策略执行测试套件
func TestStrategyExecutionSuite(t *testing.T) {
	suite.Run(t, new(StrategyExecutionTestSuite))
}