package factory

import (
	"testing"

	pdb "analysis/internal/db"
)

// createMockDeps creates mock dependencies for testing
func createMockDeps() *ExecutionDependencies {
	// For testing purposes, we can use nil for dependencies
	// as the actual tests focus on factory behavior
	return &ExecutionDependencies{}
}

func TestNewStrategyFactory(t *testing.T) {
	deps := createMockDeps()
	factory := NewStrategyFactory(deps)
	if factory == nil {
		t.Fatal("NewStrategyFactory() returned nil")
	}
	if factory.deps != deps {
		t.Error("Factory dependencies not set correctly")
	}
}

func TestStrategyFactory_ErrorHandling(t *testing.T) {
	factory := NewStrategyFactory(createMockDeps())

	_, _, err := factory.CreateExecutor("nonexistent_strategy", pdb.StrategyConditions{})
	if err == nil {
		t.Error("Expected error for nonexistent strategy")
	}
}

func TestStrategyFactory_ConfigBuilders(t *testing.T) {
	factory := NewStrategyFactory(createMockDeps())

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

	// Test config creation doesn't panic
	t.Run("traditional config", func(t *testing.T) {
		config := factory.buildTraditionalConfig(conditions)
		if config == nil {
			t.Error("Expected non-nil config")
		}
	})

	t.Run("moving average config", func(t *testing.T) {
		config := factory.buildMovingAverageConfig(conditions)
		if config == nil {
			t.Error("Expected non-nil config")
		}
	})

	t.Run("arbitrage config", func(t *testing.T) {
		config := factory.buildArbitrageConfig(conditions)
		if config == nil {
			t.Error("Expected non-nil config")
		}
	})

	t.Run("grid trading config", func(t *testing.T) {
		config := factory.buildGridTradingConfig(conditions)
		if config == nil {
			t.Error("Expected non-nil config")
		}
	})

	t.Run("mean reversion config", func(t *testing.T) {
		config := factory.buildMeanReversionConfig(conditions)
		if config == nil {
			t.Error("Expected non-nil config")
		}
	})
}

func BenchmarkStrategyFactory_CreateExecutor(b *testing.B) {
	factory := NewStrategyFactory(createMockDeps())
	conditions := pdb.StrategyConditions{ShortOnGainers: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateExecutor("traditional", conditions)
	}
}