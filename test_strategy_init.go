package main

import (
	"fmt"
	traditional_core "analysis/internal/server/strategy/traditional/core"
)

func main() {
	fmt.Println("Testing strategy initialization...")

	strategy := traditional_core.GetTraditionalStrategy()
	if strategy == nil {
		fmt.Println("ERROR: GetTraditionalStrategy returned nil")
		return
	}

	fmt.Println("SUCCESS: Strategy initialized")

	adapter := strategy.ToStrategyScanner()
	if adapter == nil {
		fmt.Println("ERROR: ToStrategyScanner returned nil")
		return
	}

	fmt.Println("SUCCESS: Adapter created")

	// Try type assertion
	if scanner, ok := adapter.(interface{ GetStrategyType() string }); ok {
		strategyType := scanner.GetStrategyType()
		fmt.Printf("SUCCESS: Strategy type: %s\n", strategyType)
	} else {
		fmt.Println("ERROR: Type assertion failed")
	}
}