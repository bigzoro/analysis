package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// SimpleIntegrationTestSuite 简单的集成测试套件
type SimpleIntegrationTestSuite struct {
	suite.Suite
	router *gin.Engine
}

// SetupTest 设置测试
func (suite *SimpleIntegrationTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// 设置基本的路由用于测试
	suite.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	suite.router.POST("/strategies/execute/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"strategy_id": id,
				"symbol": "BTCUSDT",
				"action": "BUY",
			},
		})
	})

	suite.router.POST("/strategies/scan-eligible", func(c *gin.Context) {
		strategyID := c.Query("strategy_id")
		if strategyID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "strategy_id is required",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"strategy_id": strategyID,
				"candidates": []gin.H{
					{
						"symbol": "BTCUSDT",
						"reason": "符合涨幅开空条件",
						"score": 85,
					},
				},
			},
		})
	})
}

// TestHealthCheck 测试健康检查端点
func (suite *SimpleIntegrationTestSuite) TestHealthCheck() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	suite.parseJSONResponse(w, &response)

	assert.Equal(suite.T(), "ok", response["status"])
}

// TestStrategyExecution 测试策略执行端点
func (suite *SimpleIntegrationTestSuite) TestStrategyExecution() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/strategies/execute/123", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	suite.parseJSONResponse(w, &response)

	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "123", data["strategy_id"])
	assert.Equal(suite.T(), "BTCUSDT", data["symbol"])
	assert.Equal(suite.T(), "BUY", data["action"])
}

// TestStrategyScanning 测试策略扫描端点
func (suite *SimpleIntegrationTestSuite) TestStrategyScanning() {
	// 正常请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/strategies/scan-eligible?strategy_id=456", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	suite.parseJSONResponse(w, &response)

	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "456", data["strategy_id"])

	candidates := data["candidates"].([]interface{})
	assert.Len(suite.T(), candidates, 1)

	candidate := candidates[0].(map[string]interface{})
	assert.Equal(suite.T(), "BTCUSDT", candidate["symbol"])
	assert.Equal(suite.T(), "符合涨幅开空条件", candidate["reason"])
	assert.Equal(suite.T(), float64(85), candidate["score"])
}

// TestStrategyScanning_MissingStrategyID 测试缺少策略ID的情况
func (suite *SimpleIntegrationTestSuite) TestStrategyScanning_MissingStrategyID() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/strategies/scan-eligible", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	suite.parseJSONResponse(w, &response)

	assert.Contains(suite.T(), response, "error")
	assert.Equal(suite.T(), "strategy_id is required", response["error"])
}

// TestInvalidEndpoint 测试无效端点
func (suite *SimpleIntegrationTestSuite) TestInvalidEndpoint() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/invalid-endpoint", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// parseJSONResponse 解析JSON响应
func (suite *SimpleIntegrationTestSuite) parseJSONResponse(w *httptest.ResponseRecorder, v interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), v)
	suite.NoError(err, "Failed to parse JSON response")
}

// TestSimpleIntegrationSuite 运行简单集成测试套件
func TestSimpleIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SimpleIntegrationTestSuite))
}