package main

import (
	"fmt"
)

// 模拟前端updateTechnicalIndicators函数（修复后）
func updateTechnicalIndicatorsFixed(data map[string]interface{}) map[string]string {
	return map[string]string{
		"btcVolatility": formatFloat(data["btc_volatility"], 2, "0.00"),
		"avgRSI":        formatFloat(data["avg_rsi"], 1, "0.0"),
		"strongSymbols": fmt.Sprintf("%v", data["strong_symbols"]),
		"weakSymbols":   fmt.Sprintf("%v", data["weak_symbols"]),
	}
}

// 模拟前端updateTechnicalIndicators函数（修复前）
func updateTechnicalIndicatorsBroken(data map[string]interface{}) map[string]string {
	return map[string]string{
		"btcVolatility": formatFloat(data["btcVolatility"], 2, "0.00"), // 错误的字段名
		"avgRSI":        formatFloat(data["avgRSI"], 1, "0.0"),         // 错误的字段名
		"strongSymbols": fmt.Sprintf("%v", data["strongSymbols"]),      // 错误的字段名
		"weakSymbols":   fmt.Sprintf("%v", data["weakSymbols"]),        // 错误的字段名
	}
}

func formatFloat(value interface{}, decimals int, defaultValue string) string {
	if v, ok := value.(float64); ok && v != 0 {
		return fmt.Sprintf("%.*f", decimals, v)
	}
	return defaultValue
}

func main() {
	fmt.Println("🎯 验证技术指标字段名修复")

	// 模拟后端返回的数据（蛇形命名）
	backendData := map[string]interface{}{
		"btc_volatility": 1.26,
		"avg_rsi":        47.64,
		"strong_symbols": 22,
		"weak_symbols":   147,
	}

	fmt.Println("\n📊 后端返回的原始数据:")
	fmt.Printf("  btc_volatility: %.2f\n", backendData["btc_volatility"])
	fmt.Printf("  avg_rsi: %.2f\n", backendData["avg_rsi"])
	fmt.Printf("  strong_symbols: %v\n", backendData["strong_symbols"])
	fmt.Printf("  weak_symbols: %v\n", backendData["weak_symbols"])

	fmt.Println("\n🔧 修复前的前端处理结果:")
	brokenResult := updateTechnicalIndicatorsBroken(backendData)
	fmt.Printf("  btcVolatility: '%s'\n", brokenResult["btcVolatility"])
	fmt.Printf("  avgRSI: '%s'\n", brokenResult["avgRSI"])
	fmt.Printf("  strongSymbols: '%s'\n", brokenResult["strongSymbols"])
	fmt.Printf("  weakSymbols: '%s'\n", brokenResult["weakSymbols"])

	fmt.Println("\n✅ 修复后的前端处理结果:")
	fixedResult := updateTechnicalIndicatorsFixed(backendData)
	fmt.Printf("  btcVolatility: '%s'\n", fixedResult["btcVolatility"])
	fmt.Printf("  avgRSI: '%s'\n", fixedResult["avgRSI"])
	fmt.Printf("  strongSymbols: '%s'\n", fixedResult["strongSymbols"])
	fmt.Printf("  weakSymbols: '%s'\n", fixedResult["weakSymbols"])

	fmt.Println("\n🎉 修复效果对比:")
	allZeroBefore := brokenResult["btcVolatility"] == "0.00" &&
					 brokenResult["avgRSI"] == "0.0" &&
					 brokenResult["strongSymbols"] == "0" &&
					 brokenResult["weakSymbols"] == "0"

	allCorrectAfter := fixedResult["btcVolatility"] == "1.26" &&
					   fixedResult["avgRSI"] == "47.6" &&
					   fixedResult["strongSymbols"] == "22" &&
					   fixedResult["weakSymbols"] == "147"

	if allZeroBefore && allCorrectAfter {
		fmt.Println("✅ 修复完全成功！")
		fmt.Println("   • 修复前：所有技术指标显示为0")
		fmt.Println("   • 修复后：显示正确的数据值")
		fmt.Println("   • 原因：字段名不匹配导致数据无法正确映射")
		fmt.Println("   • 解决方案：统一前后端字段命名规范")
	} else {
		fmt.Println("❌ 修复可能不完整")
	}

	fmt.Println("\n💡 技术细节:")
	fmt.Println("   • 后端使用蛇形命名法 (btc_volatility)")
	fmt.Println("   • 前端使用驼峰命名法 (btcVolatility)")
	fmt.Println("   • JSON序列化保持后端命名")
	fmt.Println("   • 前端需要正确映射字段名")

	fmt.Println("\n🎯 现在前端技术指标监控应该正常显示数据了！")
}