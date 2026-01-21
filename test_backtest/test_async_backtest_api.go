// test_async_backtest_api.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AsyncBacktestRequest 异步回测请求结构
type AsyncBacktestRequest struct {
	// 基础参数
	StartDate      string   `json:"start_date"`
	EndDate        string   `json:"end_date"`
	Strategy       string   `json:"strategy"`
	InitialCapital float64  `json:"initial_capital"`
	PositionSize   float64  `json:"position_size"`

	// 自动选择币种参数
	AutoSelectSymbol        bool     `json:"auto_select_symbol"`
	Symbols                 []string `json:"symbols,omitempty"` // 多币种列表
	MaxSymbolsToEvaluate    int      `json:"max_symbols_to_evaluate"`
	SymbolSelectionCriteria string   `json:"symbol_selection_criteria"`

	// 自动执行参数
	AutoExecute          bool    `json:"auto_execute"`
	AutoExecuteRiskLevel string  `json:"auto_execute_risk_level"`
	MinConfidence        float64 `json:"min_confidence"`
	MaxPositionPercent   float64 `json:"max_position_percent"`
	SkipExistingTrades   bool    `json:"skip_existing_trades"`

	// 渐进式执行参数
	ProgressiveExecution  bool `json:"progressive_execution"`
	MaxBatches            int  `json:"max_batches"`
	BatchSize             int  `json:"batch_size"`
	DynamicSizing         bool `json:"dynamic_sizing"`
	MarketConditionFilter bool `json:"market_condition_filter"`
}

// AsyncBacktestResponse 异步回测响应结构
type AsyncBacktestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	RecordID uint  `json:"record_id,omitempty"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details,omitempty"`
	} `json:"error,omitempty"`
}

func main() {
	fmt.Println("=== 异步回测API综合测试 ===")
	fmt.Println("测试功能：自动选择币种 + 市场热度智能 + 深度学习策略 + 自动执行")

	// 测试用例1：自动选择币种 + 市场热度智能
	testAutoSelectWithMarketHeat()

	// 测试用例2：深度学习策略 + 自动执行
	testDeepLearningWithAutoExecute()

	// 测试用例3：多币种组合策略
	testMultiSymbolStrategy()

	// 测试用例4：渐进式执行
	testProgressiveExecution()

	// 监控回测进度
	monitorBacktestProgress()
}

// 测试用例1：自动选择币种 + 市场热度智能
func testAutoSelectWithMarketHeat() {
	fmt.Println("\n--- 测试用例1：自动选择币种 + 市场热度智能 ---")

	request := AsyncBacktestRequest{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		Strategy:       "enhanced_ai",
		InitialCapital: 10000.0,
		PositionSize:   1000.0,

		// 自动选择币种参数
		AutoSelectSymbol:        true,
		MaxSymbolsToEvaluate:    20,
		SymbolSelectionCriteria: "market_heat",

		// 自动执行参数
		AutoExecute:          true,
		AutoExecuteRiskLevel: "medium",
		MinConfidence:        0.7,
		MaxPositionPercent:   0.2,
		SkipExistingTrades:   false,

		// 渐进式执行
		ProgressiveExecution:  true,
		MaxBatches:            10,
		BatchSize:             50,
		DynamicSizing:         true,
		MarketConditionFilter: true,
	}

	sendRequest("自动选择币种+市场热度智能", request)
}

// 测试用例2：深度学习策略 + 自动执行
func testDeepLearningWithAutoExecute() {
	fmt.Println("\n--- 测试用例2：深度学习策略 + 自动执行 ---")

	request := AsyncBacktestRequest{
		StartDate:      "2024-06-01",
		EndDate:        "2024-12-01",
		Strategy:       "transformer_enhanced",
		InitialCapital: 15000.0,
		PositionSize:   1500.0,

		// 自动选择币种参数
		AutoSelectSymbol:        true,
		MaxSymbolsToEvaluate:    15,
		SymbolSelectionCriteria: "ai_recommended",

		// 自动执行参数
		AutoExecute:          true,
		AutoExecuteRiskLevel: "conservative",
		MinConfidence:        0.8,
		MaxPositionPercent:   0.15,
		SkipExistingTrades:   true,

		// 渐进式执行
		ProgressiveExecution:  true,
		MaxBatches:            8,
		BatchSize:             30,
		DynamicSizing:         true,
		MarketConditionFilter: true,
	}

	sendRequest("深度学习策略+自动执行", request)
}

// 测试用例3：多币种组合策略
func testMultiSymbolStrategy() {
	fmt.Println("\n--- 测试用例3：多币种组合策略 ---")

	request := AsyncBacktestRequest{
		StartDate:      "2024-03-01",
		EndDate:        "2024-09-01",
		Strategy:       "multi_asset_ai",
		InitialCapital: 20000.0,
		PositionSize:   2000.0,

		// 指定多个币种
		AutoSelectSymbol:        false,
		Symbols:                 []string{"BTC", "ETH", "BNB", "ADA", "SOL"},
		SymbolSelectionCriteria: "",

		// 自动执行参数
		AutoExecute:          true,
		AutoExecuteRiskLevel: "aggressive",
		MinConfidence:        0.6,
		MaxPositionPercent:   0.25,
		SkipExistingTrades:   false,

		// 渐进式执行
		ProgressiveExecution:  true,
		MaxBatches:            12,
		BatchSize:             40,
		DynamicSizing:         true,
		MarketConditionFilter: true,
	}

	sendRequest("多币种组合策略", request)
}

// 测试用例4：渐进式执行
func testProgressiveExecution() {
	fmt.Println("\n--- 测试用例4：渐进式执行 ---")

	request := AsyncBacktestRequest{
		StartDate:      "2024-09-01",
		EndDate:        "2024-12-01",
		Strategy:       "risk_parity",
		InitialCapital: 12000.0,
		PositionSize:   1200.0,

		// 自动选择币种参数
		AutoSelectSymbol:        true,
		MaxSymbolsToEvaluate:    10,
		SymbolSelectionCriteria: "volatility",

		// 自动执行参数
		AutoExecute:          true,
		AutoExecuteRiskLevel: "low",
		MinConfidence:        0.75,
		MaxPositionPercent:   0.1,
		SkipExistingTrades:   true,

		// 渐进式执行（重点测试）
		ProgressiveExecution:  true,
		MaxBatches:            15,
		BatchSize:             20,
		DynamicSizing:         true,
		MarketConditionFilter: true,
	}

	sendRequest("渐进式执行", request)
}

// 发送请求的通用函数
func sendRequest(testName string, request AsyncBacktestRequest) {
	fmt.Printf("发送请求：%s\n", testName)

	// 将请求序列化为JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		return
	}

	// 创建HTTP请求
	url := "http://127.0.0.1:8010/api/backtest/async/start?tz=Asia%2FShanghai"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	// 注意：实际使用时需要添加认证token
	// req.Header.Set("Authorization", "Bearer YOUR_TOKEN")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 发送请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	// 解析响应
	var response AsyncBacktestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		fmt.Printf("原始响应: %s\n", string(body))
		return
	}

	// 处理响应
	if response.Success {
		fmt.Printf("✅ %s - 成功启动，记录ID: %d\n", testName, response.RecordID)
		fmt.Printf("   消息: %s\n", response.Message)

		// 记录ID用于后续监控
		recordIDs = append(recordIDs, response.RecordID)
	} else {
		fmt.Printf("❌ %s - 启动失败\n", testName)
		fmt.Printf("   错误代码: %s\n", response.Error.Code)
		fmt.Printf("   错误消息: %s\n", response.Error.Message)
		if response.Error.Details != "" {
			fmt.Printf("   错误详情: %s\n", response.Error.Details)
		}
	}

	fmt.Printf("   HTTP状态码: %d\n", resp.StatusCode)
}

// 全局变量存储记录ID，用于监控
var recordIDs []uint

// 监控回测进度
func monitorBacktestProgress() {
	fmt.Println("\n--- 监控回测进度 ---")

	if len(recordIDs) == 0 {
		fmt.Println("没有记录ID需要监控")
		return
	}

	fmt.Printf("监控 %d 个回测任务...\n", len(recordIDs))

	for _, recordID := range recordIDs {
		fmt.Printf("\n监控记录ID: %d\n", recordID)

		// 检查记录状态
		checkBacktestStatus(recordID)

		// 检查交易记录
		checkBacktestTrades(recordID)

		// 等待一段时间再检查下一个
		time.Sleep(2 * time.Second)
	}
}

// 检查回测状态
func checkBacktestStatus(recordID uint) {
	url := fmt.Sprintf("http://127.0.0.1:8010/api/backtest/async/records/%d", recordID)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("❌ 获取状态失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("回测状态响应: %s\n", string(body))
}

// 检查交易记录
func checkBacktestTrades(recordID uint) {
	url := fmt.Sprintf("http://127.0.0.1:8010/api/backtest/async/trades/%d?page=1&limit=10", recordID)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("❌ 获取交易记录失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	fmt.Printf("交易记录响应: %s\n", string(body))
}

// 性能分析函数
func analyzePerformance() {
	fmt.Println("\n=== 性能分析和优化建议 ===")

	// 这里可以添加基于日志的性能分析
	fmt.Println("1. 检查Transformer模型参与情况")
	fmt.Println("   查找日志: [ENSEMBLE] 模型 transformer")

	fmt.Println("\n2. 检查自动选择币种效果")
	fmt.Println("   查找日志: [AUTO_SELECT]")

	fmt.Println("\n3. 检查自动执行效果")
	fmt.Println("   查找日志: [AUTO_EXECUTE]")

	fmt.Println("\n4. 检查渐进式执行")
	fmt.Println("   查找日志: [PROGRESSIVE]")

	fmt.Println("\n5. 优化建议:")
	fmt.Println("   - 如果Transformer权重过低，考虑调整初始权重")
	fmt.Println("   - 如果交易次数过少，检查趋势过滤器阈值")
	fmt.Println("   - 如果自动执行过于保守，调整MinConfidence参数")
	fmt.Println("   - 监控系统资源使用情况，避免过载")
}

// 主函数结尾添加性能分析
func init() {
	// 程序结束时进行性能分析
	time.AfterFunc(10*time.Second, analyzePerformance)
}
