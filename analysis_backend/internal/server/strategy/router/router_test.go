package router

import (
	"testing"

	pdb "analysis/internal/db"
)

func TestNewStrategyRouter(t *testing.T) {
	router := NewStrategyRouter()
	if router == nil {
		t.Fatal("NewStrategyRouter() returned nil")
	}
	if len(router.routes) == 0 {
		t.Error("Router should have default routes")
	}
}

func TestStrategyRouter_SelectRoute(t *testing.T) {
	router := NewStrategyRouter()

	tests := []struct {
		name       string
		conditions pdb.StrategyConditions
		wantType   string
	}{
		{
			name: "mean reversion enabled",
			conditions: pdb.StrategyConditions{
				MeanReversionEnabled: true,
			},
			wantType: "mean_reversion",
		},
		{
			name: "moving average enabled",
			conditions: pdb.StrategyConditions{
				MovingAverageEnabled: true,
			},
			wantType: "moving_average",
		},
		{
			name: "traditional short enabled",
			conditions: pdb.StrategyConditions{
				ShortOnGainers: true,
			},
			wantType: "traditional",
		},
		{
			name: "arbitrage enabled",
			conditions: pdb.StrategyConditions{
				TriangleArbEnabled: true,
			},
			wantType: "arbitrage",
		},
		{
			name: "grid trading enabled",
			conditions: pdb.StrategyConditions{
				GridTradingEnabled: true,
			},
			wantType: "grid_trading",
		},
		{
			name:       "no strategy enabled",
			conditions: pdb.StrategyConditions{},
			wantType:   "", // Should return nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := router.SelectRoute(tt.conditions)
			if tt.wantType == "" {
				if route != nil {
					t.Errorf("Expected nil route, got %v", route.StrategyType)
				}
				return
			}
			if route == nil {
				t.Errorf("Expected route type %s, got nil", tt.wantType)
				return
			}
			if route.StrategyType != tt.wantType {
				t.Errorf("Expected route type %s, got %s", tt.wantType, route.StrategyType)
			}
		})
	}
}

func TestStrategyRouter_Priority(t *testing.T) {
	router := NewStrategyRouter()

	// Test that higher priority routes are selected first
	conditions := pdb.StrategyConditions{
		MeanReversionEnabled: true, // Priority 100
		MovingAverageEnabled: true, // Priority 90
		ShortOnGainers:       true, // Priority 70
	}

	route := router.SelectRoute(conditions)
	if route == nil {
		t.Fatal("Expected route, got nil")
	}
	if route.StrategyType != "mean_reversion" {
		t.Errorf("Expected highest priority route 'mean_reversion', got '%s'", route.StrategyType)
	}
}

func TestStrategyRouter_Validation(t *testing.T) {
	router := NewStrategyRouter()

	invalidConditions := []pdb.StrategyConditions{
		{GainersRankLimit: -1},                // 负数排名
		{ShortMAPeriod: 0},                    // 无效周期
		{LongMAPeriod: 0},                     // 无效周期
		{ShortMAPeriod: 20, LongMAPeriod: 10}, // 短期大于长期
		{MarketCapLimitShort: -100},           // 负市值
		{ShortMultiplier: 0},                  // 无效倍数
		{MAType: "INVALID"},                   // 无效类型
	}

	for _, conditions := range invalidConditions {
		route := router.SelectRoute(conditions)
		// 无效输入应该返回nil
		if route != nil {
			t.Errorf("Expected nil route for invalid conditions, got %v", route.StrategyType)
		}
	}
}

func TestStrategyRouter_BuildExecutionMarketData(t *testing.T) {
	router := NewStrategyRouter()

	input := StrategyMarketData{
		Symbol:      "BTCUSDT",
		MarketCap:   1000000.0,
		GainersRank: 5,
		HasSpot:     true,
		HasFutures:  true,
	}

	result := router.buildExecutionMarketData(input)

	if result.Symbol != input.Symbol {
		t.Errorf("Expected Symbol %s, got %s", input.Symbol, result.Symbol)
	}
	if result.MarketCap != input.MarketCap {
		t.Errorf("Expected MarketCap %f, got %f", input.MarketCap, result.MarketCap)
	}
	if result.GainersRank != input.GainersRank {
		t.Errorf("Expected GainersRank %d, got %d", input.GainersRank, result.GainersRank)
	}
	if result.HasSpot != input.HasSpot {
		t.Errorf("Expected HasSpot %v, got %v", input.HasSpot, result.HasSpot)
	}
	if result.HasFutures != input.HasFutures {
		t.Errorf("Expected HasFutures %v, got %v", input.HasFutures, result.HasFutures)
	}
}

func TestStrategyRouter_BuildExecutionContext(t *testing.T) {
	router := NewStrategyRouter()

	symbol := "BTCUSDT"
	strategyType := "traditional"
	userID := uint(123)
	strategyID := uint(456)

	result := router.buildExecutionContext(symbol, strategyType, userID, strategyID)

	if result.Symbol != symbol {
		t.Errorf("Expected Symbol %s, got %s", symbol, result.Symbol)
	}
	if result.StrategyType != strategyType {
		t.Errorf("Expected StrategyType %s, got %s", strategyType, result.StrategyType)
	}
	if result.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, result.UserID)
	}
	if result.RequestID != "traditional-456-BTCUSDT" {
		t.Errorf("Expected RequestID 'traditional-456-BTCUSDT', got '%s'", result.RequestID)
	}
}

func TestStrategyRouter_ConfigBuilders(t *testing.T) {
	router := NewStrategyRouter()

	conditions := pdb.StrategyConditions{
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

	// Test that config builders don't panic and return non-nil values
	tests := []struct {
		name    string
		builder func(pdb.StrategyConditions) interface{}
	}{
		{"traditional config", router.buildTraditionalConfig},
		{"moving average config", router.buildMovingAverageConfig},
		{"arbitrage config", router.buildArbitrageConfig},
		{"grid trading config", router.buildGridTradingConfig},
		{"mean reversion config", router.buildMeanReversionConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.builder(conditions)
			if config == nil {
				t.Errorf("%s builder returned nil", tt.name)
			}
		})
	}
}

func TestStrategyRouter_GetAllRoutes(t *testing.T) {
	router := NewStrategyRouter()

	routes := router.GetAllRoutes()
	if len(routes) == 0 {
		t.Error("Expected at least one route")
	}

	// Verify routes are sorted by priority
	for i := 1; i < len(routes); i++ {
		if routes[i-1].Priority < routes[i].Priority {
			t.Errorf("Routes not sorted by priority: %d < %d", routes[i-1].Priority, routes[i].Priority)
		}
	}
}

func BenchmarkStrategyRouter_SelectRoute(b *testing.B) {
	router := NewStrategyRouter()
	conditions := pdb.StrategyConditions{
		MeanReversionEnabled: true,
		MovingAverageEnabled: true,
		ShortOnGainers:       true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.SelectRoute(conditions)
	}
}
