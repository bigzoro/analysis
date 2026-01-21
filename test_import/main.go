package main

import (
	"fmt"
	"analysis/internal/server"
)

func main() {
	// This won't actually run since server has dependencies, but it should compile the struct
	p := &server.PredictionResult{
		Symbol:     "TEST",
		Score:      0.5,
		Confidence: 0.8,
		Quality:    0.9,
	}
	fmt.Printf("Quality: %.3f\n", p.Quality)
}
