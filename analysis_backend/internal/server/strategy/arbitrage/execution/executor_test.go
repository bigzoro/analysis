package execution

import (
	"context"
	"testing"

	"analysis/internal/server/strategy/shared/execution"
)

// Mock dependencies for testing
type mockMarketDataProvider struct{}
type mockOrderManager struct{}
type mockRiskManager struct{}

func (m *mockMarketDataProvider) GetMarketData(symbol string) (*execution.MarketData, error) {
	return &execution.MarketData{
		Symbol: symbol,
		Price:  50000.0,
		Volume: 1000.0,
	}, nil
}

func (m *mockMarketDataProvider) GetRealTimePrice(symbol string) (float64, error) {
	return 50000.0, nil
}

func (m *mockMarketDataProvider) GetKlineData(symbol, interval string, limit int) ([]*execution.KlineData, error) {
	return []*execution.KlineData{}, nil
}

func (m *mockOrderManager) PlaceOrder(symbol, side string, quantity, price float64) (string, error) {
	return "mock_order_id", nil
}

func (m *mockOrderManager) CancelOrder(orderID string) error {
	return nil
}

func (m *mockOrderManager) GetOrderStatus(orderID string) (*execution.OrderStatus, error) {
	return &execution.OrderStatus{OrderID: orderID, Status: "filled"}, nil
}

func (m *mockRiskManager) ValidateRisk(symbol string, positionSize float64) error {
	return nil
}

func (m *mockRiskManager) CalculateStopLoss(entryPrice float64, riskPercent float64) float64 {
	return entryPrice * (1 - riskPercent/100)
}

func (m *mockRiskManager) CalculateTakeProfit(entryPrice float64, rewardPercent float64) float64 {
	return entryPrice * (1 + rewardPercent/100)
}

func (m *mockRiskManager) CheckPositionLimits(symbol string, newPositionSize float64) error {
	return nil
}

func createMockDeps() *ExecutionDependencies {
	return &ExecutionDependencies{
		MarketDataProvider: &mockMarketDataProvider{},
		OrderManager:       &mockOrderManager{},
		RiskManager:        &mockRiskManager{},
	}
}

func TestNewExecutor(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}

	if executor.GetStrategyType() != "arbitrage" {
		t.Errorf("Expected strategy type 'arbitrage', got '%s'", executor.GetStrategyType())
	}
}

func TestExecutor_IsEnabled(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	tests := []struct {
		name   string
		config *ArbitrageExecutionConfig
		want   bool
	}{
		{
			name: "enabled with triangle arb",
			config: &ArbitrageExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: true},
				TriangleArbEnabled: true,
			},
			want: true,
		},
		{
			name: "enabled with spot future arb",
			config: &ArbitrageExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: true},
				SpotFutureArbEnabled: true,
			},
			want: true,
		},
		{
			name: "disabled - execution disabled",
			config: &ArbitrageExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: false},
				TriangleArbEnabled: true,
			},
			want: false,
		},
		{
			name: "disabled - no arb enabled",
			config: &ArbitrageExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := executor.IsEnabled(tt.config); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutor_ValidateExecution(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	marketData := &execution.MarketData{
		Symbol:    "BTCUSDT",
		HasSpot:   true,
		HasFutures: true,
	}
	config := ArbitrageExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		TriangleArbEnabled: true,
		MinProfitThreshold: 0.5,
	}

	err := executor.ValidateExecution("BTCUSDT", marketData, &config)
	if err != nil {
		t.Errorf("ValidateExecution() error = %v", err)
	}
}

func TestExecutor_Execute(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol: "BTCUSDT",
		Price:  50000.0,
	}
	config := ArbitrageExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		TriangleArbEnabled: true,
		MinProfitThreshold: 0.5,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	result, err := executor.Execute(ctx, "BTCUSDT", marketData, &config, execContext)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
		return
	}

	if result == nil {
		t.Error("Execute() returned nil result")
		return
	}

	if result.Symbol != "BTCUSDT" {
		t.Errorf("Expected symbol BTCUSDT, got %s", result.Symbol)
	}
}

func TestExecutor_ExecuteTriangleArbitrage(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol: "BTCUSDT",
		Price:  50000.0,
	}
	config := ArbitrageExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		TriangleArbEnabled: true,
		MinProfitThreshold: 0.5,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	result, err := executor.ExecuteTriangleArbitrage(ctx, "BTCUSDT", marketData, &config, execContext)
	if err != nil {
		t.Errorf("ExecuteTriangleArbitrage() error = %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteTriangleArbitrage() returned nil result")
	}
}

func TestExecutor_ExecuteSpotFutureArbitrage(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol: "BTCUSDT",
		Price:  50000.0,
	}
	config := ArbitrageExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		SpotFutureArbEnabled: true,
		MinProfitThreshold: 0.5,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	result, err := executor.ExecuteSpotFutureArbitrage(ctx, "BTCUSDT", marketData, &config, execContext)
	if err != nil {
		t.Errorf("ExecuteSpotFutureArbitrage() error = %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteSpotFutureArbitrage() returned nil result")
	}
}

func BenchmarkExecutor_Execute(b *testing.B) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol: "BTCUSDT",
		Price:  50000.0,
	}
	config := ArbitrageExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		TriangleArbEnabled: true,
		MinProfitThreshold: 0.5,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(ctx, "BTCUSDT", marketData, &config, execContext)
	}
}