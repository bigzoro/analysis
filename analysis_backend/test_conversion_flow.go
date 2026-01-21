package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

// 模拟前端转换逻辑
func convertFundingRatesForStorage(conditions map[string]interface{}) map[string]interface{} {
	fmt.Printf("🔄 前端转换开始: %+v\n", conditions)

	result := make(map[string]interface{})
	for k, v := range conditions {
		result[k] = v
	}

	if val, exists := result["min_funding_rate"]; exists && val != nil {
		if rate, ok := val.(float64); ok {
			fmt.Printf("📊 转换前 min_funding_rate: %f\n", rate)
			result["min_funding_rate"] = rate / 100
			fmt.Printf("📊 转换后 min_funding_rate: %f\n", result["min_funding_rate"])
		}
	}

	if val, exists := result["futures_price_short_min_funding_rate"]; exists && val != nil {
		if rate, ok := val.(float64); ok {
			fmt.Printf("📊 转换前 futures_price_short_min_funding_rate: %f\n", rate)
			result["futures_price_short_min_funding_rate"] = rate / 100
			fmt.Printf("📊 转换后 futures_price_short_min_funding_rate: %f\n", result["futures_price_short_min_funding_rate"])
		}
	}

	fmt.Printf("✅ 前端转换完成: %+v\n", result)
	return result
}

// 模拟后端接收和处理
func simulateBackendProcessing(conditions map[string]interface{}) {
	fmt.Println("\n🔧 后端处理开始")

	// 模拟UpdateTradingStrategy中的逻辑
	req := struct {
		Conditions pdb.StrategyConditions
	}{}

	// 手动设置条件（模拟JSON解析）
	if val, exists := conditions["funding_rate_filter_enabled"]; exists {
		if enabled, ok := val.(bool); ok {
			req.Conditions.FundingRateFilterEnabled = enabled
		}
	}

	if val, exists := conditions["min_funding_rate"]; exists {
		if rate, ok := val.(float64); ok {
			req.Conditions.MinFundingRate = rate
		}
	}

	if val, exists := conditions["futures_price_short_strategy_enabled"]; exists {
		if enabled, ok := val.(bool); ok {
			req.Conditions.FuturesPriceShortStrategyEnabled = enabled
		}
	}

	if val, exists := conditions["futures_price_short_min_funding_rate"]; exists {
		if rate, ok := val.(float64); ok {
			req.Conditions.FuturesPriceShortMinFundingRate = rate
		}
	}

	fmt.Printf("📋 后端接收到的数据:\n")
	fmt.Printf("   FundingRateFilterEnabled: %v\n", req.Conditions.FundingRateFilterEnabled)
	fmt.Printf("   MinFundingRate: %f\n", req.Conditions.MinFundingRate)
	fmt.Printf("   FuturesPriceShortStrategyEnabled: %v\n", req.Conditions.FuturesPriceShortStrategyEnabled)
	fmt.Printf("   FuturesPriceShortMinFundingRate: %f\n", req.Conditions.FuturesPriceShortMinFundingRate)

	// 模拟保存逻辑
	fmt.Printf("💾 模拟保存到数据库:\n")
	fmt.Printf("   MinFundingRate: %f (%f%%)\n", req.Conditions.MinFundingRate, req.Conditions.MinFundingRate*100)
	fmt.Printf("   FuturesPriceShortMinFundingRate: %f (%f%%)\n",
		req.Conditions.FuturesPriceShortMinFundingRate,
		req.Conditions.FuturesPriceShortMinFundingRate*100)
}

func main() {
	fmt.Println("🧪 资金费率转换流程测试")
	fmt.Println("========================")

	// 模拟用户输入-1的情况
	fmt.Println("\n🎯 测试场景: 用户在前端输入-1（表示-1%）")

	// 1. 模拟前端表单数据
	frontendData := map[string]interface{}{
		"funding_rate_filter_enabled":                 true,
		"min_funding_rate":                            -1.0, // 用户输入-1
		"futures_price_short_strategy_enabled":        true,
		"futures_price_short_min_funding_rate":        -1.0, // 用户输入-1
	}

	fmt.Printf("📝 前端表单数据: %+v\n", frontendData)

	// 2. 前端转换
	convertedData := convertFundingRatesForStorage(frontendData)

	// 3. 后端处理
	simulateBackendProcessing(convertedData)

	fmt.Println("\n" + "="*60)
	fmt.Println("🔍 问题诊断")

	// 检查是否出现了异常数值
	problemValue := -1.0000000000000008e-202
	fmt.Printf("❌ 用户报告的异常数值: %e (%f%%)\n", problemValue, problemValue*100)

	if convertedData["min_funding_rate"] == problemValue {
		fmt.Println("🚨 发现问题：转换结果与异常数值匹配！")
	} else {
		fmt.Println("✅ 转换结果正常，不匹配异常数值")
		fmt.Printf("   预期结果: %f, 实际结果: %v\n",
			convertedData["min_funding_rate"],
			convertedData["min_funding_rate"])
	}

	fmt.Println("\n💡 可能原因分析:")
	fmt.Println("   1. 前端转换被意外多次执行")
	fmt.Println("   2. Vue响应式系统导致的重复转换")
	fmt.Println("   3. 网络传输过程中的数值精度损失")
	fmt.Println("   4. 后端JSON解析时的浮点数精度问题")

	fmt.Println("\n🔧 建议解决方案:")
	fmt.Println("   1. 添加前端调试日志，确认转换时机")
	fmt.Println("   2. 检查Vue的watch或computed是否重复触发转换")
	fmt.Println("   3. 验证网络请求中的数据格式")
	fmt.Println("   4. 后端添加数值范围验证")
}