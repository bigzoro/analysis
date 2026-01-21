package main

import (
	"os"
)

func main() {
	files := []string{
		"test_compile_final.go",
		"syntax_check.go",
		"test_realtime_adaptation.go",
		"test_tiered_optimization_final.go",
		"test_tiered_optimization.go",
		"test_multi_dimensional_quality.go",
		"test_multi_source_pool.go",
		"test_balanced_selection.go",
		"test_compile_check.go",
		"test_compile_fix.go",
	}

	for _, file := range files {
		os.Remove(file)
	}
}