package integration

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"analysis/internal/db"

	"github.com/stretchr/testify/suite"
)

// StrategyScanningTestSuite 策略扫描集成测试套件
type StrategyScanningTestSuite struct {
	IntegrationTestSuite
}

// TestStrategyScanning_TraditionalStrategyScan 测试传统策略扫描
func (suite *StrategyScanningTestSuite) TestStrategyScanning_TraditionalStrategyScan() {
	ts := suite.GetTestServer()

	// 创建传统策略用于扫描
	conditions := db.StrategyConditions{
		ShortOnGainers:      true,
		GainersRankLimit:    10,
		MarketCapLimitShort: 1000000,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行扫描请求
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)

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

	// 验证扫描结果
	suite.Contains(data, "strategy_id", "Result should contain strategy_id")
	suite.Contains(data, "candidates", "Result should contain candidates")

	// 验证策略ID
	strategyID, ok := data["strategy_id"].(float64)
	suite.True(ok, "strategy_id should be a number")
	suite.Equal(float64(strategy.ID), strategyID, "Strategy ID should match")
}

// TestStrategyScanning_MovingAverageStrategyScan 测试均线策略扫描
func (suite *StrategyScanningTestSuite) TestStrategyScanning_MovingAverageStrategyScan() {
	ts := suite.GetTestServer()

	// 创建均线策略用于扫描
	conditions := db.StrategyConditions{
		MovingAverageEnabled: true,
		MAType:               "SMA",
		ShortMAPeriod:        5,
		LongMAPeriod:         20,
		MACrossSignal:        "GOLDEN_CROSS",
	}

	strategy, err := ts.CreateTestStrategy("moving_average", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行扫描请求
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)

	// 验证响应
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	suite.Contains(response, "success", "Response should contain success field")
	suite.Contains(response, "data", "Response should contain data field")
}

// TestStrategyScanning_InvalidStrategyID 测试无效策略ID的扫描
func (suite *StrategyScanningTestSuite) TestStrategyScanning_InvalidStrategyID() {
	ts := suite.GetTestServer()

	// 使用不存在的策略ID
	w := ts.MakeRequest("POST", "/strategies/scan-eligible?strategy_id=99999", nil)

	// 应该返回错误状态
	suite.AssertResponseStatus(w, 500) // 500表示内部错误（扫描器未找到）

	// 验证错误响应
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse error response")

	suite.Contains(response, "error", "Error response should contain error field")
}

// TestStrategyScanning_MissingStrategyID 测试缺少策略ID的扫描
func (suite *StrategyScanningTestSuite) TestStrategyScanning_MissingStrategyID() {
	ts := suite.GetTestServer()

	// 不提供策略ID
	w := ts.MakeRequest("POST", "/strategies/scan-eligible", nil)

	// 应该返回错误状态
	suite.AssertResponseStatus(w, 400)

	// 验证错误响应
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse error response")

	suite.Contains(response, "error", "Error response should contain error field")
}

// TestStrategyScanning_DisabledStrategy 测试禁用策略的扫描
func (suite *StrategyScanningTestSuite) TestStrategyScanning_DisabledStrategy() {
	ts := suite.GetTestServer()

	// 创建禁用策略
	conditions := db.StrategyConditions{
		ShortOnGainers: false, // 禁用所有条件
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行扫描请求
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)

	// 验证响应
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	suite.Contains(response, "success", "Response should contain success field")
	suite.Contains(response, "data", "Response should contain data field")

	// 验证扫描结果为空或很少
	data, ok := response["data"].(map[string]interface{})
	suite.True(ok, "Data should be an object")

	candidates, ok := data["candidates"].([]interface{})
	suite.True(ok, "Candidates should be an array")

	// 禁用策略应该没有候选结果
	suite.Len(candidates, 0, "Disabled strategy should have no candidates")
}

// TestStrategyScanning_Timeout 测试扫描超时
func (suite *StrategyScanningTestSuite) TestStrategyScanning_Timeout() {
	ts := suite.GetTestServer()

	// 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers:   true,
		GainersRankLimit: 10,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 设置较短的超时时间
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d&timeout=1s", strategy.ID), nil)

	// 验证响应（应该正常完成或在超时前完成）
	suite.AssertResponseStatus(w, 200)
}

// TestStrategyScanning_ConcurrentRequests 测试并发扫描请求
func (suite *StrategyScanningTestSuite) TestStrategyScanning_ConcurrentRequests() {
	ts := suite.GetTestServer()

	// 创建策略
	conditions := db.StrategyConditions{
		ShortOnGainers:   true,
		GainersRankLimit: 10,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 并发执行多个扫描请求
	numRequests := 5
	results := make(chan *httptest.ResponseRecorder, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)
			results <- w
		}()
	}

	// 收集结果
	for i := 0; i < numRequests; i++ {
		w := <-results
		suite.AssertResponseStatus(w, 200, "All concurrent scan requests should succeed")
	}
}

// TestStrategyScanning_ResultFormat 测试扫描结果格式
func (suite *StrategyScanningTestSuite) TestStrategyScanning_ResultFormat() {
	ts := suite.GetTestServer()

	// 创建传统策略用于扫描
	conditions := db.StrategyConditions{
		ShortOnGainers:   true,
		GainersRankLimit: 10,
	}

	strategy, err := ts.CreateTestStrategy("traditional", conditions)
	suite.Require().NoError(err, "Failed to create test strategy")

	// 执行扫描请求
	w := ts.MakeRequest("POST", fmt.Sprintf("/strategies/scan-eligible?strategy_id=%d", strategy.ID), nil)

	// 验证响应
	suite.AssertResponseStatus(w, 200)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err, "Failed to parse response")

	// 验证响应结构
	data, ok := response["data"].(map[string]interface{})
	suite.True(ok, "Data should be an object")

	candidates, ok := data["candidates"].([]interface{})
	suite.True(ok, "Candidates should be an array")

	// 如果有候选结果，验证格式
	if len(candidates) > 0 {
		candidate, ok := candidates[0].(map[string]interface{})
		suite.True(ok, "First candidate should be an object")

		// 验证候选结果包含必要的字段
		suite.Contains(candidate, "symbol", "Candidate should contain symbol")
		suite.Contains(candidate, "reason", "Candidate should contain reason")
		suite.Contains(candidate, "score", "Candidate should contain score")
	}
}

// TestStrategyScanningSuite 运行策略扫描测试套件
func TestStrategyScanningSuite(t *testing.T) {
	suite.Run(t, new(StrategyScanningTestSuite))
}
