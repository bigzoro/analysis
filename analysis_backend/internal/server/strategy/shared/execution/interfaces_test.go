package execution

import (
	"context"
	"testing"
)

// Test that all interface types can be instantiated (compile-time check)
func TestInterfaceTypes(t *testing.T) {
	// This test ensures that all interface types can be used as types
	var _ StrategyExecutor = (*mockStrategyExecutor)(nil)
	var _ MarketDataProvider = (*mockMarketDataProvider)(nil)
	var _ OrderManager = (*mockOrderManager)(nil)
	var _ RiskManager = (*mockRiskManager)(nil)
	var _ ConfigProvider = (*mockConfigProvider)(nil)
}

// Mock implementations for interface testing
type mockStrategyExecutor struct{}
type mockMarketDataProvider struct{}
type mockOrderManager struct{}
type mockRiskManager struct{}
type mockConfigProvider struct{}

func (m *mockStrategyExecutor) GetStrategyType() string { return "mock" }
func (m *mockStrategyExecutor) IsEnabled(config interface{}) bool { return true }
func (m *mockStrategyExecutor) ValidateExecution(ctx context.Context, marketData *MarketData, config interface{}, execContext *ExecutionContext) error { return nil }
func (m *mockStrategyExecutor) Execute(ctx context.Context, symbol string, marketData *MarketData, config interface{}, execContext *ExecutionContext) (*ExecutionResult, error) { return &ExecutionResult{}, nil }

func (m *mockMarketDataProvider) GetMarketData(symbol string) (*MarketData, error) { return &MarketData{}, nil }
func (m *mockMarketDataProvider) GetRealTimePrice(symbol string) (float64, error) { return 0, nil }
func (m *mockMarketDataProvider) GetKlineData(symbol, interval string, limit int) ([]*KlineData, error) { return []*KlineData{}, nil }

func (m *mockOrderManager) PlaceOrder(symbol, side string, quantity, price float64) (string, error) { return "", nil }
func (m *mockOrderManager) CancelOrder(orderID string) error { return nil }
func (m *mockOrderManager) GetOrderStatus(orderID string) (*OrderStatus, error) { return &OrderStatus{}, nil }

func (m *mockRiskManager) ValidateRisk(symbol string, positionSize float64) error { return nil }
func (m *mockRiskManager) CalculateStopLoss(entryPrice float64, riskPercent float64) float64 { return 0 }
func (m *mockRiskManager) CalculateTakeProfit(entryPrice float64, rewardPercent float64) float64 { return 0 }
func (m *mockRiskManager) CheckPositionLimits(symbol string, newPositionSize float64) error { return nil }

func (m *mockConfigProvider) GetStrategyConfig(strategyType string, userID uint) (interface{}, error) { return nil, nil }
func (m *mockConfigProvider) GetGlobalConfig(key string) (interface{}, error) { return nil, nil }
func (m *mockConfigProvider) UpdateStrategyConfig(strategyType string, userID uint, config interface{}) error { return nil }

func TestExecutionResult(t *testing.T) {
	result := &ExecutionResult{
		Symbol:      "BTCUSDT",
		StrategyType: "test",
		Action:      "BUY",
		Quantity:    1.0,
		Price:       50000.0,
		UserID:      123,
		StrategyID:  456,
		OrderID:     "order_123",
		Timestamp:   1609459200,
		Reason:      "test execution",
	}

	if result.Symbol != "BTCUSDT" {
		t.Errorf("Expected Symbol 'BTCUSDT', got '%s'", result.Symbol)
	}
	if result.StrategyType != "test" {
		t.Errorf("Expected StrategyType 'test', got '%s'", result.StrategyType)
	}
	if result.Action != "BUY" {
		t.Errorf("Expected Action 'BUY', got '%s'", result.Action)
	}
}

func TestMarketData(t *testing.T) {
	md := &MarketData{
		Symbol:      "BTCUSDT",
		Price:       50000.0,
		Volume:      1000.0,
		MarketCap:   1000000.0,
		GainersRank: 5,
		HasSpot:     true,
		HasFutures:  true,
	}

	if md.Symbol != "BTCUSDT" {
		t.Errorf("Expected Symbol 'BTCUSDT', got '%s'", md.Symbol)
	}
	if md.Price != 50000.0 {
		t.Errorf("Expected Price 50000.0, got %f", md.Price)
	}
	if !md.HasSpot {
		t.Error("Expected HasSpot to be true")
	}
}

func TestExecutionContext(t *testing.T) {
	ctx := &ExecutionContext{
		Symbol:       "BTCUSDT",
		StrategyType: "test",
		UserID:       123,
		RequestID:    "req_123",
		Timestamp:    1609459200,
	}

	if ctx.Symbol != "BTCUSDT" {
		t.Errorf("Expected Symbol 'BTCUSDT', got '%s'", ctx.Symbol)
	}
	if ctx.UserID != 123 {
		t.Errorf("Expected UserID 123, got %d", ctx.UserID)
	}
}

func TestExecutionConfig(t *testing.T) {
	config := &ExecutionConfig{
		Enabled: true,
	}

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
}