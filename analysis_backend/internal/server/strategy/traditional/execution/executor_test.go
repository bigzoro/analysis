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
		Symbol:      symbol,
		Price:       50000.0,
		Volume:      1000.0,
		MarketCap:   1000000.0,
		GainersRank: 5,
		HasSpot:     true,
		HasFutures:  true,
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

	if executor.GetStrategyType() != "traditional" {
		t.Errorf("Expected strategy type 'traditional', got '%s'", executor.GetStrategyType())
	}
}

func TestExecutor_IsEnabled(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	tests := []struct {
		name   string
		config *TraditionalExecutionConfig
		want   bool
	}{
		{
			name: "enabled with short on gainers",
			config: &TraditionalExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: true},
				ShortOnGainers:  true,
			},
			want: true,
		},
		{
			name: "enabled with long on small gainers",
			config: &TraditionalExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: true},
				LongOnSmallGainers: true,
			},
			want: true,
		},
		{
			name: "disabled",
			config: &TraditionalExecutionConfig{
				ExecutionConfig: execution.ExecutionConfig{Enabled: false},
				ShortOnGainers:  true,
			},
			want: false,
		},
		{
			name: "no strategy enabled",
			config: &TraditionalExecutionConfig{
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

	marketData := &execution.MarketData{Symbol: "BTCUSDT"}
	config := TraditionalExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		ShortOnGainers:  true,
		GainersRankLimit: 10,
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
		Symbol:      "BTCUSDT",
		GainersRank: 5,
	}
	config := TraditionalExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		ShortOnGainers:  true,
		GainersRankLimit: 10,
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

func TestExecutor_ExecuteShortOnGainers(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol:      "BTCUSDT",
		GainersRank: 5,
		MarketCap:   1000000.0,
	}
	config := TraditionalExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		ShortOnGainers:  true,
		GainersRankLimit: 10,
		MarketCapLimitShort: 500000.0, // 低于市场市值限制
		ShortMultiplier: 1.0,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	result, err := executor.ExecuteShortOnGainers(ctx, "BTCUSDT", marketData, &config, execContext)
	if err != nil {
		t.Errorf("ExecuteShortOnGainers() error = %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteShortOnGainers() returned nil result")
	}
}

func TestExecutor_ExecuteLongOnSmallGainers(t *testing.T) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol:      "BTCUSDT",
		GainersRank: 15, // 高排名表示涨幅较小
		MarketCap:   1000000.0,
	}
	config := TraditionalExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		LongOnSmallGainers: true,
		LongGainersRankLimit: 20,
		MarketCapLimitLong: 2000000.0, // 高于市场市值限制
		LongMultiplier: 1.0,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	result, err := executor.ExecuteLongOnSmallGainers(ctx, "BTCUSDT", marketData, &config, execContext)
	if err != nil {
		t.Errorf("ExecuteLongOnSmallGainers() error = %v", err)
		return
	}

	if result == nil {
		t.Error("ExecuteLongOnSmallGainers() returned nil result")
	}
}

func BenchmarkExecutor_Execute(b *testing.B) {
	deps := createMockDeps()
	executor := NewExecutor(deps)

	ctx := context.Background()
	marketData := &execution.MarketData{
		Symbol:      "BTCUSDT",
		GainersRank: 5,
	}
	config := TraditionalExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		ShortOnGainers:  true,
		GainersRankLimit: 10,
	}
	execContext := &execution.ExecutionContext{
		Symbol: "BTCUSDT",
		UserID: 123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(ctx, "BTCUSDT", marketData, config, execContext)
	}
}