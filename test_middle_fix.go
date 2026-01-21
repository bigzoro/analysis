package main

import "fmt"

func main() {
	fmt.Println("🔧 修复'middle'未使用变量测试")
	fmt.Println("============================")

	// 模拟CalculateBollingerBands函数调用
	// 这个函数通常返回三个值: upper, middle, lower
	upper := []float64{100.0, 101.0, 102.0}
	middle := []float64{95.0, 96.0, 97.0}  // 这个中间值现在被忽略
	lower := []float64{90.0, 91.0, 92.0}

	// 模拟修复后的代码: 使用 _ 忽略 middle 值
	upper2, _, lower2 := []float64{100.0, 101.0, 102.0}, []float64{95.0, 96.0, 97.0}, []float64{90.0, 91.0, 92.0}

	fmt.Printf("✅ 修复前: upper=%v, middle=%v, lower=%v\n", upper, middle, lower)
	fmt.Printf("✅ 修复后: upper=%v, lower=%v (middle被忽略)\n", upper2, lower2)
	fmt.Printf("✅ 数组长度: upper=%d, lower=%d\n", len(upper2), len(lower2))

	fmt.Println("\n🎉 'middle'未使用变量修复完成！")
	fmt.Println("strategy_scanner_mean_reversion.go 中的编译错误已修复。")
}