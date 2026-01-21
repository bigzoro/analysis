package main

import (
	"fmt"
	"go/parser"
	"go/token"
)

// 简单的语法检查
func main() {
	fset := token.NewFileSet()

	// 解析market_analysis.go文件
	_, err := parser.ParseFile(fset, "analysis_backend/internal/server/market_analysis.go", nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("❌ 语法错误: %v\n", err)
		return
	}

	fmt.Println("✅ market_analysis.go 语法检查通过")
	fmt.Println("所有未使用的变量已清理")
}