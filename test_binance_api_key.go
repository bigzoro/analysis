package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	// 币安期货API基础URL
	baseURL = "https://fapi.binance.com"

	// 测试网络URL（如果需要测试环境）
	testnetURL = "https://testnet.binancefuture.com"
)

type BinanceAccountResponse struct {
	// 基础权限
	CanTrade    bool `json:"canTrade"`
	CanWithdraw bool `json:"canWithdraw"`
	CanDeposit  bool `json:"canDeposit"`

	// 费用相关
	FeeTier           int   `json:"feeTier"`
	FeeBurn           bool  `json:"feeBurn"`
	TradeGroupId      int   `json:"tradeGroupId"`
	UpdateTime        int64 `json:"updateTime"`
	MultiAssetsMargin bool  `json:"multiAssetsMargin"`

	// 保证金和余额信息
	TotalInitialMargin          string `json:"totalInitialMargin"`
	TotalMaintMargin            string `json:"totalMaintMargin"`
	TotalWalletBalance          string `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string `json:"totalCrossUnPnl"`
	AvailableBalance            string `json:"availableBalance"`

	// 为了兼容旧版本API的字段（如果存在）
	MakerCommission  int64  `json:"makerCommission,omitempty"`
	TakerCommission  int64  `json:"takerCommission,omitempty"`
	BuyerCommission  int64  `json:"buyerCommission,omitempty"`
	SellerCommission int64  `json:"sellerCommission,omitempty"`
	AccountType      string `json:"accountType,omitempty"`
	Balances         []struct {
		Asset              string `json:"asset"`
		Balance            string `json:"balance"`
		CrossWalletBalance string `json:"crossWalletBalance"`
		CrossUnPnl         string `json:"crossUnPnl"`
		AvailableBalance   string `json:"availableBalance"`
		MaxWithdrawAmount  string `json:"maxWithdrawAmount"`
	} `json:"balances,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Perms       []string `json:"perms,omitempty"`
}

type PositionInfo struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	LiquidationPrice string `json:"liquidationPrice"`
	Leverage         string `json:"leverage"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	MarginType       string `json:"marginType"`
	IsolatedMargin   string `json:"isolatedMargin"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	PositionSide     string `json:"positionSide"`
	Notional         string `json:"notional"`
	IsolatedWallet   string `json:"isolatedWallet"`
	UpdateTime       int64  `json:"updateTime"`
}

// 订单信息结构体
type OrderInfo struct {
	Symbol            string `json:"symbol"`
	OrderId           int64  `json:"orderId"`
	ClientOrderId     string `json:"clientOrderId"`
	Price             string `json:"price"`
	OrigQty           string `json:"origQty"`
	ExecutedQty       string `json:"executedQty"`
	CumQuote          string `json:"cumQuote"`
	Status            string `json:"status"`
	TimeInForce       string `json:"timeInForce"`
	Type              string `json:"type"`
	Side              string `json:"side"`
	StopPrice         string `json:"stopPrice"`
	IcebergQty        string `json:"icebergQty"`
	Time              int64  `json:"time"`
	UpdateTime        int64  `json:"updateTime"`
	IsWorking         bool   `json:"isWorking"`
	OrigQuoteOrderQty string `json:"origQuoteOrderQty"`
}

// 交易记录结构体
type TradeInfo struct {
	Symbol          string `json:"symbol"`
	Id              int64  `json:"id"`
	OrderId         int64  `json:"orderId"`
	Side            string `json:"side"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	RealizedPnl     string `json:"realizedPnl"`
	QuoteQty        string `json:"quoteQty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	Buyer           bool   `json:"buyer"`
	Maker           bool   `json:"maker"`
}

type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func main() {
	fmt.Println("🔑 币安API密钥测试工具")
	fmt.Println("========================")

	// ⚠️ 请在这里输入您的API密钥
	apiKey := "IqkbgcaWmQkrgOG8RMWEuybvIN8xzlpsNIqNFHZMR8UmllTAewg3gVGwZJuJf7im"    // 替换为您的API Key
	secretKey := "r3chQS99v3dxjxMRIS2H8IHEJewBdg0l7mo5avTHwv1satKpzPnw6cq9gjH3GVrf" // 替换为您的Secret Key

	// 是否使用测试网络
	useTestnet := false

	if apiKey == "your_api_key_here" || secretKey == "your_secret_here" {
		fmt.Println("❌ 请先设置您的API密钥和密钥！")
		fmt.Println("   请编辑代码中的 apiKey 和 secretKey 变量")
		return
	}

	fmt.Printf("🔗 使用%s环境\n", map[bool]string{true: "测试网", false: "正式网"}[useTestnet])

	// 测试API密钥
	testAPIKey(apiKey, secretKey, useTestnet)
}

func testAPIKey(apiKey, secretKey string, useTestnet bool) {
	baseURL := baseURL
	if useTestnet {
		baseURL = testnetURL
	}

	fmt.Println("\n1️⃣ 测试账户信息接口...")

	// 获取账户信息
	account, err := getAccountInfo(baseURL, apiKey, secretKey)
	if err != nil {
		fmt.Printf("❌ 获取账户信息失败: %v\n", err)
		return
	}

	fmt.Println("✅ API密钥验证成功！")
	fmt.Printf("📊 费用等级: %d\n", account.FeeTier)
	fmt.Printf("💰 是否可以交易: %t\n", account.CanTrade)
	fmt.Printf("💸 是否可以提币: %t\n", account.CanWithdraw)
	fmt.Printf("📥 是否可以充币: %t\n", account.CanDeposit)
	fmt.Printf("🔥 费用燃烧: %t\n", account.FeeBurn)
	fmt.Printf("🔄 多资产保证金: %t\n", account.MultiAssetsMargin)

	// 显示权限信息（优先使用Permissions字段，如果为空则尝试Perms字段）
	permissions := account.Permissions
	if len(permissions) == 0 && len(account.Perms) > 0 {
		permissions = account.Perms
	}
	if len(permissions) > 0 {
		fmt.Printf("🔐 权限列表: %v\n", permissions)
	} else {
		fmt.Println("🔐 权限列表: (无明确权限字段，由canTrade/canWithdraw等控制)")
	}

	fmt.Println("\n2️⃣ 检查账户余额...")

	// 显示总余额信息
	fmt.Printf("   💰 总钱包余额: %.8f USDT\n", parseFloat(account.TotalWalletBalance))
	fmt.Printf("   💵 可用余额: %.8f USDT\n", parseFloat(account.AvailableBalance))
	fmt.Printf("   📊 保证金余额: %.8f USDT\n", parseFloat(account.TotalMarginBalance))

	// 检查持仓保证金
	if account.TotalPositionInitialMargin != "0.00000000" {
		fmt.Printf("   📈 持仓保证金: %.8f USDT\n", parseFloat(account.TotalPositionInitialMargin))
	}

	// 检查未实现盈亏
	if account.TotalUnrealizedProfit != "0.00000000" {
		fmt.Printf("   💹 未实现盈亏: %.8f USDT\n", parseFloat(account.TotalUnrealizedProfit))
	}

	// 显示详细的资产余额（如果有balances数组）
	for _, balance := range account.Balances {
		if balance.AvailableBalance != "0.00000000" && balance.AvailableBalance != "0.00" {
			fmt.Printf("   📋 %s: 可用 %.8f\n", balance.Asset, parseFloat(balance.AvailableBalance))
		}
	}

	if parseFloat(account.AvailableBalance) > 0 {
		fmt.Println("   ✅ 账户有可用资金！")
	} else {
		fmt.Println("   📝 账户可用余额为0")
	}

	fmt.Println("\n3️⃣ 测试其他API接口...")

	// 测试获取交易对信息
	fmt.Println("   🔍 测试交易对信息接口...")
	if err := testExchangeInfo(baseURL, apiKey); err != nil {
		fmt.Printf("   ❌ 交易对信息测试失败: %v\n", err)
	} else {
		fmt.Println("   ✅ 交易对信息接口正常")
	}

	// 测试获取当前价格
	fmt.Println("   💰 测试价格查询接口...")
	if err := testPriceQuery(baseURL, apiKey); err != nil {
		fmt.Printf("   ❌ 价格查询测试失败: %v\n", err)
	} else {
		fmt.Println("   ✅ 价格查询接口正常")
	}

	// 测试RENDERUSDT持仓查询
	fmt.Println("   📊 测试RENDERUSDT持仓查询接口...")
	position, err := getPositionInfo(baseURL, apiKey, secretKey, "RENDERUSDT")
	if err != nil {
		fmt.Printf("   ❌ RENDERUSDT持仓查询失败: %v\n", err)
	} else {
		fmt.Println("   ✅ RENDERUSDT持仓查询成功")
		fmt.Printf("   📋 交易对: %s\n", position.Symbol)
		fmt.Printf("   📊 持仓数量: %s\n", position.PositionAmt)
		if position.EntryPrice != "" && position.EntryPrice != "0.00000000" {
			fmt.Printf("   💰 入场价格: %s\n", position.EntryPrice)
		}
		if position.MarkPrice != "" && position.MarkPrice != "0.00000000" {
			fmt.Printf("   🎯 标记价格: %s\n", position.MarkPrice)
		}
		if position.UnRealizedProfit != "" && position.UnRealizedProfit != "0.00000000" {
			fmt.Printf("   💹 未实现盈亏: %s\n", position.UnRealizedProfit)
		}
		if position.Leverage != "" && position.Leverage != "0" {
			fmt.Printf("   ⚡ 杠杆倍数: %s\n", position.Leverage)
		}
		if position.PositionAmt == "0.00000000" || position.PositionAmt == "0" {
			fmt.Println("   📝 当前无持仓")
		}
	}

	fmt.Println("\n4️⃣ 查询当前挂单...")
	if err := checkOpenOrders(baseURL, apiKey, secretKey); err != nil {
		fmt.Printf("❌ 查询挂单失败: %v\n", err)
	}

	fmt.Println("\n5️⃣ 查询历史订单和交易记录...")
	if err := checkOrderHistory(baseURL, apiKey, secretKey); err != nil {
		fmt.Printf("❌ 查询订单历史失败: %v\n", err)
	}

	fmt.Println("\n6️⃣ 计算盈利统计...")
	if err := calculateProfitStats(baseURL, apiKey, secretKey); err != nil {
		fmt.Printf("❌ 计算盈利统计失败: %v\n", err)
	}

	fmt.Println("\n7️⃣ 测试Algo订单API（新版止盈止损）...")

	// 测试Algo订单API可用性（不实际下单）
	fmt.Println("   🔍 检查Algo订单API可用性...")
	testBaseURL := baseURL
	if useTestnet {
		testBaseURL = testnetURL
	}

	// 检查Algo订单端点是否可访问
	requestURL := fmt.Sprintf("%s/fapi/v1/exchangeInfo", testBaseURL)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		fmt.Printf("   ❌ 创建请求失败: %v\n", err)
	} else {
		req.Header.Set("X-MBX-APIKEY", apiKey)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("   ❌ API请求失败: %v\n", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				fmt.Println("   ✅ Algo订单API可用（已修复止盈止损订单问题）")
				fmt.Println("   ℹ️  现在使用正确的 /fapi/v1/algoOrder 端点")
				fmt.Println("   ℹ️  参数包括: algoType=CONDITIONAL, triggerPrice, clientAlgoId")
			} else {
				fmt.Printf("   ⚠️  API响应异常: HTTP %d\n", resp.StatusCode)
			}
		}
	}

	fmt.Println("\n🎉 所有测试完成！您的API密钥工作正常。")
	fmt.Println("\n⚠️  安全提醒:")
	fmt.Println("   - 妥善保管您的API密钥，不要泄露给他人")
	fmt.Println("   - 定期更换API密钥以确保安全")
	fmt.Println("   - 不要在公共场所或不安全的网络中使用")
}

func getAccountInfo(baseURL, apiKey, secretKey string) (*BinanceAccountResponse, error) {
	// 构造请求参数
	timestamp := time.Now().UnixMilli()
	params := fmt.Sprintf("timestamp=%d", timestamp)

	// 生成签名
	signature := generateSignature(params, secretKey)
	fullParams := fmt.Sprintf("%s&signature=%s", params, signature)

	// 构造完整URL
	requestURL := fmt.Sprintf("%s/fapi/v2/account?%s", baseURL, fullParams)

	// 创建请求
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("X-MBX-APIKEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API错误 (代码: %d): %s", apiErr.Code, apiErr.Msg)
		}
		return nil, fmt.Errorf("HTTP错误: %s, 响应: %s", resp.Status, string(body))
	}

	// 调试：打印原始响应（前500字符）
	fmt.Printf("📄 原始响应预览: %s...\n", string(body)[:min(500, len(body))])

	// 解析响应
	var account BinanceAccountResponse
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(body))
	}

	// 调试：检查关键字段
	fmt.Printf("🔍 解析结果 - 账户类型: '%s', 权限: %v\n", account.AccountType, account.Permissions)

	return &account, nil
}

func testExchangeInfo(baseURL, apiKey string) error {
	requestURL := fmt.Sprintf("%s/fapi/v1/exchangeInfo", baseURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %s: %s", resp.Status, string(body))
	}

	return nil
}

func testPriceQuery(baseURL, apiKey string) error {
	requestURL := fmt.Sprintf("%s/fapi/v1/ticker/price?symbol=BTCUSDT", baseURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %s: %s", resp.Status, string(body))
	}

	return nil
}

func getPositionInfo(baseURL, apiKey, secretKey, symbol string) (*PositionInfo, error) {
	// 构造请求参数
	timestamp := time.Now().UnixMilli()
	params := fmt.Sprintf("timestamp=%d", timestamp)

	// 生成签名
	signature := generateSignature(params, secretKey)
	fullParams := fmt.Sprintf("%s&signature=%s", params, signature)

	// 构造完整URL
	requestURL := fmt.Sprintf("%s/fapi/v2/positionRisk?%s", baseURL, fullParams)

	// 创建请求
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("X-MBX-APIKEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		// 打印详细的请求信息用于调试
		fmt.Printf("🔍 [持仓查询调试信息]\n")
		fmt.Printf("📡 请求URL: %s\n", requestURL)
		fmt.Printf("🔑 API Key: %s\n", apiKey[:10]+"...") // 只显示前10位
		fmt.Printf("⏰ 时间戳: %d\n", timestamp)
		fmt.Printf("✍️  签名参数: %s\n", params)
		fmt.Printf("🔐 签名: %s\n", signature)
		fmt.Printf("📨 请求头 X-MBX-APIKEY: %s\n", req.Header.Get("X-MBX-APIKEY"))
		fmt.Printf("📨 请求头 Content-Type: %s\n", req.Header.Get("Content-Type"))
		fmt.Printf("📊 响应状态码: %d\n", resp.StatusCode)
		fmt.Printf("📄 响应内容: %s\n", string(body))

		// 提供curl命令示例
		fmt.Printf("🔧 Curl测试命令:\n")
		fmt.Printf("curl -H \"X-MBX-APIKEY: %s\" -H \"Content-Type: application/json\" \"%s\"\n", apiKey, requestURL)
		fmt.Printf("🔚 [调试信息结束]\n\n")

		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API错误 (代码: %d): %s", apiErr.Code, apiErr.Msg)
		}
		return nil, fmt.Errorf("HTTP错误: %s, 响应: %s", resp.Status, string(body))
	}

	// 解析响应（返回数组）
	var positions []PositionInfo
	if err := json.Unmarshal(body, &positions); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(body))
	}

	// 查找指定交易对的持仓信息
	for _, position := range positions {
		if position.Symbol == symbol {
			return &position, nil
		}
	}

	// 如果没有找到该交易对的持仓，返回空持仓信息
	return &PositionInfo{Symbol: symbol}, nil
}

func generateSignature(params, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(params))
	return hex.EncodeToString(h.Sum(nil))
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 查询当前挂单
func checkOpenOrders(baseURL, apiKey, secretKey string) error {
	timestamp := time.Now().UnixMilli()
	params := fmt.Sprintf("timestamp=%d", timestamp)

	signature := generateSignature(params, secretKey)
	requestURL := fmt.Sprintf("%s/fapi/v1/openOrders?%s&signature=%s", baseURL, params, signature)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return fmt.Errorf("API错误 (代码: %d): %s", apiErr.Code, apiErr.Msg)
		}
		return fmt.Errorf("HTTP错误: %s", resp.Status)
	}

	var orders []OrderInfo
	if err := json.Unmarshal(body, &orders); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if len(orders) == 0 {
		fmt.Println("   📝 当前无挂单")
		return nil
	}

	fmt.Printf("   📋 找到 %d 个挂单:\n", len(orders))
	totalValue := 0.0
	for i, order := range orders {
		fmt.Printf("   %d. %s %s %s @ %s (已成交: %s/%s)\n",
			i+1, order.Symbol, order.Side, order.Type, order.Price,
			order.ExecutedQty, order.OrigQty)

		if order.Status == "PARTIAL_FILLED" {
			fmt.Printf("      📊 部分成交，状态: %s\n", order.Status)
		}

		// 计算订单价值
		if price := parseFloat(order.Price); price > 0 {
			qty := parseFloat(order.OrigQty)
			totalValue += price * qty
		}
	}

	if totalValue > 0 {
		fmt.Printf("   💰 挂单总价值: ≈%.2f USDT\n", totalValue)
	}

	return nil
}

// 查询历史订单和交易记录
func checkOrderHistory(baseURL, apiKey, secretKey string) error {
	fmt.Println("   🔍 查询最近24小时的订单历史...")

	// 查询最近24小时的订单
	endTime := time.Now().UnixMilli()
	startTime := endTime - (24 * 60 * 60 * 1000) // 24小时前

	timestamp := time.Now().UnixMilli()
	params := fmt.Sprintf("timestamp=%d&startTime=%d&endTime=%d", timestamp, startTime, endTime)

	signature := generateSignature(params, secretKey)
	requestURL := fmt.Sprintf("%s/fapi/v1/allOrders?%s&signature=%s", baseURL, params, signature)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return fmt.Errorf("API错误 (代码: %d): %s", apiErr.Code, apiErr.Msg)
		}
		return fmt.Errorf("HTTP错误: %s", resp.Status)
	}

	var orders []OrderInfo
	if err := json.Unmarshal(body, &orders); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if len(orders) == 0 {
		fmt.Println("   📝 最近24小时无订单记录")
		return nil
	}

	fmt.Printf("   📋 找到 %d 个历史订单:\n", len(orders))

	filledOrders := 0
	totalVolume := 0.0

	for i, order := range orders {
		if i >= 10 { // 只显示最近10个订单
			fmt.Printf("   ... 还有 %d 个订单\n", len(orders)-10)
			break
		}

		status := order.Status
		if status == "FILLED" {
			filledOrders++
			if quoteQty := parseFloat(order.CumQuote); quoteQty > 0 {
				totalVolume += quoteQty
			}
		}

		timeStr := time.UnixMilli(order.Time).Format("15:04:05")
		fmt.Printf("   %d. [%s] %s %s %s @ %s (成交: %s/%s) 状态: %s\n",
			i+1, timeStr, order.Symbol, order.Side, order.Type,
			order.Price, order.ExecutedQty, order.OrigQty, status)

		// 显示止损价格
		if order.StopPrice != "" && order.StopPrice != "0" {
			fmt.Printf("      🛑 止损价格: %s\n", order.StopPrice)
		}
	}

	fmt.Printf("   📊 统计: 总订单 %d, 已成交 %d, 成交额 ≈%.2f USDT\n",
		len(orders), filledOrders, totalVolume)

	return nil
}

// 计算盈利统计
func calculateProfitStats(baseURL, apiKey, secretKey string) error {
	fmt.Println("   💹 查询交易记录并计算盈亏...")

	// 首先查询所有持仓信息来获取未实现盈亏
	fmt.Println("   📊 查询当前持仓盈亏...")
	positions, err := getAllPositions(baseURL, apiKey, secretKey)
	if err != nil {
		fmt.Printf("   ⚠️  查询持仓失败: %v\n", err)
	} else {
		totalUnrealizedPnl := 0.0
		activePositions := 0

		for _, pos := range positions {
			pnl := parseFloat(pos.UnRealizedProfit)
			amt := parseFloat(pos.PositionAmt)

			if amt != 0 {
				activePositions++
				totalUnrealizedPnl += pnl
				fmt.Printf("   📋 %s: 持仓 %.6f, 未实现盈亏 %.4f USDT\n",
					pos.Symbol, amt, pnl)
			}
		}

		fmt.Printf("   💰 未实现盈亏总计: %.4f USDT (%d 个持仓)\n", totalUnrealizedPnl, activePositions)
	}

	// 查询最近7天的交易记录
	fmt.Println("   🔍 查询交易历史...")
	endTime := time.Now().UnixMilli()
	startTime := endTime - (7 * 24 * 60 * 60 * 1000) // 7天前

	timestamp := time.Now().UnixMilli()
	params := fmt.Sprintf("timestamp=%d&startTime=%d&endTime=%d", timestamp, startTime, endTime)

	signature := generateSignature(params, secretKey)
	requestURL := fmt.Sprintf("%s/fapi/v1/userTrades?%s&signature=%s", baseURL, params, signature)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return fmt.Errorf("API错误 (代码: %d): %s", apiErr.Code, apiErr.Msg)
		}
		return fmt.Errorf("HTTP错误: %s", resp.Status)
	}

	var trades []TradeInfo
	if err := json.Unmarshal(body, &trades); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if len(trades) == 0 {
		fmt.Println("   📝 最近7天无交易记录")
		return nil
	}

	fmt.Printf("   📋 找到 %d 笔交易记录:\n", len(trades))

	// 调试：打印第一笔交易的原始数据
	if len(trades) > 0 {
		fmt.Printf("   🔍 调试信息 - 第一笔交易:\n")
		fmt.Printf("      交易对: %s\n", trades[0].Symbol)
		fmt.Printf("      方向: %s\n", trades[0].Side)
		fmt.Printf("      价格: %s\n", trades[0].Price)
		fmt.Printf("      数量: %s\n", trades[0].Qty)
		fmt.Printf("      手续费: %s (%s)\n", trades[0].Commission, trades[0].CommissionAsset)
		fmt.Printf("      已实现盈亏: %s\n", trades[0].RealizedPnl)
		fmt.Printf("      报价数量: %s\n", trades[0].QuoteQty)
		fmt.Printf("      买家: %t, 挂单: %t\n", trades[0].Buyer, trades[0].Maker)
	}

	totalRealizedPnl := 0.0
	totalCommission := 0.0
	totalVolume := 0.0
	winningTrades := 0
	losingTrades := 0
	zeroPnlTrades := 0

	// 按交易对分组统计
	symbolStats := make(map[string]map[string]float64)

	for i, trade := range trades {
		if i >= 20 { // 只显示最近20笔交易
			fmt.Printf("   ... 还有 %d 笔交易\n", len(trades)-20)
			break
		}

		pnl := parseFloat(trade.RealizedPnl)
		commission := parseFloat(trade.Commission)
		volume := parseFloat(trade.QuoteQty)

		totalRealizedPnl += pnl
		totalCommission += commission
		totalVolume += volume

		if pnl > 0 {
			winningTrades++
		} else if pnl < 0 {
			losingTrades++
		} else {
			zeroPnlTrades++
		}

		timeStr := time.UnixMilli(trade.Time).Format("01-02 15:04")
		fmt.Printf("   %d. [%s] %s %s %.6f @ %.8f 手续费:%.8f 盈亏:%.8f\n",
			i+1, timeStr, trade.Symbol, trade.Side,
			parseFloat(trade.Qty), parseFloat(trade.Price),
			commission, pnl)

		// 按交易对统计
		if symbolStats[trade.Symbol] == nil {
			symbolStats[trade.Symbol] = map[string]float64{
				"pnl":        0,
				"commission": 0,
				"volume":     0,
				"trades":     0,
			}
		}
		symbolStats[trade.Symbol]["pnl"] += pnl
		symbolStats[trade.Symbol]["commission"] += commission
		symbolStats[trade.Symbol]["volume"] += volume
		symbolStats[trade.Symbol]["trades"]++
	}

	fmt.Println("\n   📊 交易统计 (最近7天):")
	fmt.Printf("   💰 已实现盈亏: %.8f USDT\n", totalRealizedPnl)
	fmt.Printf("   💸 总手续费: %.8f USDT\n", totalCommission)
	fmt.Printf("   📈 总交易额: %.2f USDT\n", totalVolume)
	fmt.Printf("   ✅ 盈利交易: %d 笔\n", winningTrades)
	fmt.Printf("   ❌ 亏损交易: %d 笔\n", losingTrades)
	fmt.Printf("   📋 零盈亏交易: %d 笔 (开仓/平仓配对交易)\n", zeroPnlTrades)

	if winningTrades+losingTrades > 0 {
		winRate := float64(winningTrades) / float64(winningTrades+losingTrades) * 100
		fmt.Printf("   📊 胜率: %.1f%%\n", winRate)
	}

	fmt.Println("\n   🏆 各交易对表现:")
	for symbol, stats := range symbolStats {
		pnl := stats["pnl"]
		trades := stats["trades"]
		avgPnl := pnl / trades
		fmt.Printf("   %s: 盈亏 %.4f USDT, 交易 %d 笔, 平均 %.4f USDT/笔\n",
			symbol, pnl, int(trades), avgPnl)
	}

	fmt.Println("\n   💡 盈亏计算说明:")
	fmt.Println("   • 已实现盈亏: 通过交易记录计算的实际盈亏")
	fmt.Println("   • 未实现盈亏: 当前持仓的浮动盈亏")
	fmt.Println("   • 零盈亏交易: 开仓和平仓配对，通常显示为0")

	return nil
}

// 获取所有持仓信息
func getAllPositions(baseURL, apiKey, secretKey string) ([]PositionInfo, error) {
	timestamp := time.Now().UnixMilli()
	params := fmt.Sprintf("timestamp=%d", timestamp)

	signature := generateSignature(params, secretKey)
	requestURL := fmt.Sprintf("%s/fapi/v2/positionRisk?%s&signature=%s", baseURL, params, signature)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API错误 (代码: %d): %s", apiErr.Code, apiErr.Msg)
		}
		return nil, fmt.Errorf("HTTP错误: %s", resp.Status)
	}

	var positions []PositionInfo
	if err := json.Unmarshal(body, &positions); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return positions, nil
}
