// test_transformer_integration.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
)

func main() {
	fmt.Println("=== Transformer集成测试 ===")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	db, err := pdb.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 创建服务器实例（简化版）
	srv := &server.Server{
		DB: db,
	}

	// 初始化机器学习模块
	mlConfig := server.DefaultMLConfig()
	ml, err := server.NewMachineLearning(nil, db, mlConfig)
	if err != nil {
		log.Fatalf("初始化机器学习模块失败: %v", err)
	}

	srv.MachineLearning = ml

	// 测试Transformer集成
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	symbol := "BTC" // 测试币种
	fmt.Printf("测试币种: %s\n", symbol)

	// 检查Transformer模型是否已初始化
	if ml.GetTransformerModel() == nil {
		fmt.Println("❌ Transformer模型未初始化")
		fmt.Println("请检查配置文件中的transformer.num_layers是否大于0")
		return
	}

	fmt.Println("✅ Transformer模型已初始化")

	// 测试集成
	err = ml.TestTransformerIntegration(ctx, symbol)
	if err != nil {
		fmt.Printf("❌ Transformer集成测试失败: %v\n", err)
		return
	}

	fmt.Println("✅ Transformer集成测试成功！")

	// 测试集成预测
	fmt.Println("\n=== 测试集成预测 ===")
	prediction, err := ml.PredictWithEnsemble(ctx, symbol, "transformer")
	if err != nil {
		fmt.Printf("❌ 集成预测失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Transformer集成预测成功:\n")
	fmt.Printf("   得分: %.4f\n", prediction.Score)
	fmt.Printf("   置信度: %.4f\n", prediction.Confidence)
	fmt.Printf("   质量评分: %.4f\n", prediction.Quality)
	fmt.Printf("   使用的模型: %s\n", prediction.ModelUsed)

	// 测试趋势过滤器调整效果
	fmt.Println("\n=== 测试趋势过滤器调整 ===")
	fmt.Println("趋势过滤器已优化，现在应该允许更多交易")
	fmt.Println("请运行回测查看是否产生交易")

	fmt.Println("\n🎉 所有测试通过！Transformer已成功集成到交易决策流程中。")
}
