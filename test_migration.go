package main

import (
	"fmt"
	"analysis/internal/strategy/mean_reversion/core"
)

func main() {
	strategy, err := core.GetMRStrategy()
	if err != nil {
		panic(fmt.Sprintf("获取新策略失败: %v", err))
	}

	fmt.Println("新的模块化均值回归架构加载成功!")

	// 测试适配器
	adapter := strategy.ToStrategyScanner()
	if adapter != nil {
		fmt.Println("适配器创建成功!")
	} else {
		fmt.Println("适配器创建失败")
	}
}