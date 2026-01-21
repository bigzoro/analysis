// +build ignore

package main

import (
	"fmt"
	"go/parser"
	"go/token"
)

func main() {
	fset := token.NewFileSet()

	// 解析文件
	_, err := parser.ParseFile(fset, "analysis_backend/internal/server/strategy_scanner_mean_reversion.go", nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("语法错误: %v\n", err)
		return
	}

	fmt.Println("语法检查通过")
}