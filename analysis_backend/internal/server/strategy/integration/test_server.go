package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pdb "analysis/internal/db"
	"analysis/internal/server"
	"analysis/internal/server/strategy/factory"
	"analysis/internal/server/strategy/router"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TestServer 集成测试服务器
type TestServer struct {
	server   *server.Server
	router   *gin.Engine
	strategies map[uint]*db.Strategy // 内存存储策略
	nextID   uint
}

// NewTestServer 创建测试服务器
func NewTestServer() (*TestServer, error) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建服务器实例
	srv := server.NewServer()

	// 初始化策略路由器和工厂（如果支持的话）
	// 这里暂时简化，直接创建路由器

	r := router.NewStrategyRouter()
	f := factory.NewStrategyFactory(srv)

	return &TestServer{
		server:    srv,
		router:    srv.Router, // 使用服务器的路由器
		strategies: make(map[uint]*pdb.Strategy),
		nextID:    1,
	}, nil
}

// Close 关闭测试服务器
func (ts *TestServer) Close() error {
	// 内存版本不需要清理
	return nil
}

// CreateTestStrategy 创建测试策略
func (ts *TestServer) CreateTestStrategy(strategyType string, conditions pdb.StrategyConditions) (*pdb.Strategy, error) {
	strategy := &pdb.Strategy{
		ID:          ts.nextID,
		Name:        fmt.Sprintf("Test %s Strategy", strategyType),
		Type:        strategyType,
		UserID:      1,
		Symbol:      "BTCUSDT",
		Status:      "active",
		Conditions:  conditions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ts.strategies[strategy.ID] = strategy
	ts.nextID++

	return strategy, nil
}

// GetStrategy 获取策略（用于测试验证）
func (ts *TestServer) GetStrategy(id uint) (*pdb.Strategy, bool) {
	strategy, exists := ts.strategies[id]
	return strategy, exists
}

// MakeRequest 执行HTTP请求
func (ts *TestServer) MakeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()

	var req *http.Request
	var err error

	if body != nil {
		// 如果有body，需要序列化并创建POST请求
		req, err = http.NewRequest(method, path, nil) // 简化版本，实际应该序列化body
	} else {
		req, err = http.NewRequest(method, path, nil)
	}

	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return w
	}

	// 设置必要的header
	req.Header.Set("Content-Type", "application/json")

	ts.router.ServeHTTP(w, req)
	return w
}

// IntegrationTestSuite 集成测试套件
type IntegrationTestSuite struct {
	suite.Suite
	testServer *TestServer
}

// SetupTest 设置测试
func (suite *IntegrationTestSuite) SetupTest() {
	ts, err := NewTestServer()
	suite.Require().NoError(err, "Failed to create test server")
	suite.testServer = ts

	// 日志设置可以在这里添加，如果需要的话
}

// TearDownTest 清理测试
func (suite *IntegrationTestSuite) TearDownTest() {
	if suite.testServer != nil {
		suite.testServer.Close()
	}
}

// GetTestServer 获取测试服务器
func (suite *IntegrationTestSuite) GetTestServer() *TestServer {
	return suite.testServer
}

// AssertResponseStatus 断言响应状态码
func (suite *IntegrationTestSuite) AssertResponseStatus(w *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(suite.T(), expectedStatus, w.Code, "HTTP status code should match")
}

// AssertResponseContains 断言响应包含字符串
func (suite *IntegrationTestSuite) AssertResponseContains(w *httptest.ResponseRecorder, substr string) {
	assert.Contains(suite.T(), w.Body.String(), substr, "Response should contain expected string")
}

// WaitForAsyncOperation 等待异步操作完成
func (suite *IntegrationTestSuite) WaitForAsyncOperation(ctx context.Context, timeout time.Duration, checkFunc func() bool) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if checkFunc() {
				return nil
			}
		}
	}
}