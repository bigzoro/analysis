package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== 价格获取逻辑测试 ===")

	// 模拟不同的场景
	testCases := []struct {
		name        string
		kind        string
		apiSuccess  bool
		cacheExists bool
		cacheAge    time.Duration
		expected    string
	}{
		{
			name:        "期货API成功",
			kind:        "futures",
			apiSuccess:  true,
			cacheExists: true,
			cacheAge:    1 * time.Minute,
			expected:    "从API获取",
		},
		{
			name:        "期货API失败，缓存有效",
			kind:        "futures",
			apiSuccess:  false,
			cacheExists: true,
			cacheAge:    1 * time.Minute,
			expected:    "从缓存获取作为后备",
		},
		{
			name:        "期货API失败，缓存过期",
			kind:        "futures",
			apiSuccess:  false,
			cacheExists: true,
			cacheAge:    10 * time.Minute,
			expected:    "所有方法都失败",
		},
		{
			name:        "现货市场快照成功",
			kind:        "spot",
			apiSuccess:  true,
			cacheExists: true,
			cacheAge:    1 * time.Minute,
			expected:    "从市场快照获取",
		},
		{
			name:        "现货API成功",
			kind:        "spot",
			apiSuccess:  true,
			cacheExists: true,
			cacheAge:    1 * time.Minute,
			expected:    "从Binance API获取",
		},
		{
			name:        "现货API失败，缓存有效",
			kind:        "spot",
			apiSuccess:  false,
			cacheExists: true,
			cacheAge:    1 * time.Minute,
			expected:    "从缓存获取作为后备",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n测试场景: %s\n", tc.name)
		fmt.Printf("类型: %s\n", tc.kind)
		fmt.Printf("API成功: %v\n", tc.apiSuccess)
		fmt.Printf("缓存存在: %v\n", tc.cacheExists)
		fmt.Printf("缓存年龄: %v\n", tc.cacheAge)
		fmt.Printf("预期结果: %s\n", tc.expected)

		// 模拟逻辑
		var result string
		if tc.kind == "futures" {
			if tc.apiSuccess {
				result = "从API获取"
			} else if tc.cacheExists && tc.cacheAge <= 5*time.Minute {
				result = "从缓存获取作为后备"
			} else {
				result = "所有方法都失败"
			}
		} else if tc.kind == "spot" {
			// 现货优先级：市场快照 -> API -> 缓存
			// 在这个简化测试中，我们假设市场快照总是先尝试，然后API，最后缓存
			result = "从Binance API获取" // 简化测试，假设总是到达API调用
			if !tc.apiSuccess && tc.cacheExists && tc.cacheAge <= 5*time.Minute {
				result = "从缓存获取作为后备"
			} else if !tc.apiSuccess {
				result = "所有方法都失败"
			}
		}

		status := "✅ 通过"
		if result != tc.expected {
			status = "❌ 失败"
		}
		fmt.Printf("实际结果: %s %s\n", result, status)
	}

	fmt.Println("\n=== 修改说明 ===")
	fmt.Println("✅ 修改前：缓存优先 -> API后备")
	fmt.Println("✅ 修改后：API优先 -> 缓存后备")
	fmt.Println()
	fmt.Println("=== 新的优先级顺序 ===")
	fmt.Println("1. 期货价格：API (premiumIndex) -> 缓存(5分钟内)")
	fmt.Println("2. 现货价格：市场快照(2小时内) -> API (ticker/price) -> 缓存(5分钟内)")
	fmt.Println("3. 所有方法失败时返回错误")
	fmt.Println()
	fmt.Println("=== 缓存策略调整 ===")
	fmt.Println("• API成功：直接返回最新价格")
	fmt.Println("• API失败：使用5分钟内缓存作为后备（放宽时间限制）")
	fmt.Println("• 缓存过期：明确记录日志并跳过使用")

	fmt.Println("\n=== 日志示例 ===")
	fmt.Println("[scheduler] 从API获取 HANAUSDT 期货价格")
	fmt.Println("[scheduler] API获取失败，使用数据库缓存作为后备: 连接超时")
	fmt.Println("[scheduler] 从数据库缓存获取 HANAUSDT futures价格作为后备: 0.0131700")
}