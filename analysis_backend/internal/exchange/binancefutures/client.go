package binancefutures

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Base string // https://fapi.binance.com 或 https://testnet.binancefuture.com
	Key  string
	Sec  string
	HTTP *http.Client

	// 时间同步相关字段
	timeOffset   int64 // 与服务器时间的偏移量（毫秒）
	lastSyncTime int64 // 最后同步时间戳
	syncInterval int64 // 同步间隔（毫秒），默认5分钟
}

func New(testnet bool, apiKey, apiSecret string) *Client {
	base := "https://fapi.binance.com"
	if testnet {
		base = "https://testnet.binancefuture.com"
	}

	key := apiKey
	sec := apiSecret

	// 如果配置未设置，记录警告
	if key == "" || sec == "" {
		log.Printf("[WARN] Binance Futures API credentials not configured. Please set api_key and secret_key in config file under exchange.binance.")
		// 注意：这里不强制退出，因为某些功能可能不需要 Binance API
		// 但在实际使用时，如果没有凭证，API 调用会失败
	}

	client := &Client{
		Base:         base,
		Key:          key,
		Sec:          sec,
		HTTP:         &http.Client{Timeout: 15 * time.Second},
		timeOffset:   0,
		lastSyncTime: 0,
		syncInterval: 5 * 60 * 1000, // 5分钟同步一次
	}

	// 如果有API密钥，立即同步时间
	if key != "" && sec != "" {
		if err := client.SyncTime(); err != nil {
			log.Printf("[WARN] Initial time sync failed: %v", err)
		}
	}

	return client
}

func (c *Client) sign(qs string) string {
	mac := hmac.New(sha256.New, []byte(c.Sec))
	mac.Write([]byte(qs))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) doSigned(method, path string, params url.Values) (int, []byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("timestamp", strconv.FormatInt(c.getTimestamp(), 10))
	params.Set("recvWindow", "10000") // 增加到10秒以适应网络延迟
	qs := params.Encode()
	sig := c.sign(qs)
	full := c.Base + path + "?" + qs + "&signature=" + sig

	req, _ := http.NewRequest(method, full, nil)
	req.Header.Set("X-MBX-APIKEY", c.Key)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, body, nil
}

type PlaceOrderResp struct {
	OrderId int64  `json:"orderId"`
	Status  string `json:"status"`
}

// 服务器时间响应
type ServerTimeResp struct {
	ServerTime int64 `json:"serverTime"`
}

func (c *Client) SetLeverage(symbol string, lev int) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("leverage", strconv.Itoa(lev))
	return c.doSigned(http.MethodPost, "/fapi/v1/leverage", v)
}

// SetMarginType 设置保证金模式
// marginType: "ISOLATED" 或 "CROSSED"
func (c *Client) SetMarginType(symbol, marginType string) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("marginType", strings.ToUpper(marginType))
	return c.doSigned(http.MethodPost, "/fapi/v1/marginType", v)
}

// GetMarginType 获取当前保证金模式
func (c *Client) GetMarginType(symbol string) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	return c.doSigned(http.MethodGet, "/fapi/v2/positionRisk", v)
}

func (c *Client) PlaceOrder(symbol, side, otype, qty, price string, reduceOnly bool, clientOrderId string) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("side", strings.ToUpper(side))  // BUY/SELL
	v.Set("type", strings.ToUpper(otype)) // MARKET/LIMIT
	v.Set("quantity", qty)
	if strings.ToUpper(otype) == "LIMIT" {
		v.Set("timeInForce", "GTC")
		v.Set("price", price)
	}
	if reduceOnly {
		v.Set("reduceOnly", "true")
	}
	if clientOrderId != "" {
		v.Set("newClientOrderId", clientOrderId)
	}
	return c.doSigned(http.MethodPost, "/fapi/v1/order", v)
}

func ParsePlaceOrderResp(b []byte) (*PlaceOrderResp, error) {
	var r PlaceOrderResp
	err := json.Unmarshal(b, &r)
	return &r, err
}

// CancelOrder 撤单
func (c *Client) CancelOrder(symbol, clientOrderId string) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	if clientOrderId != "" {
		v.Set("origClientOrderId", clientOrderId)
	}
	return c.doSigned(http.MethodDelete, "/fapi/v1/order", v)
}

// 查询订单状态响应
type QueryOrderResp struct {
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Status        string `json:"status"`
	ExecutedQty   string `json:"executedQty"`
	AvgPrice      string `json:"avgPrice"`
	Side          string `json:"side"`
	Symbol        string `json:"symbol"`
}

// 查询订单状态
func (c *Client) QueryOrder(symbol, clientOrderId string) (*QueryOrderResp, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("origClientOrderId", clientOrderId)

	code, body, err := c.doSigned(http.MethodGet, "/fapi/v1/order", v)
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("query order failed: %s", string(body))
	}

	var resp QueryOrderResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryAlgoOrder 查询Algo订单状态（用于止盈止损条件单）
func (c *Client) QueryAlgoOrder(symbol, clientAlgoId string) (*AlgoOrderResp, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("clientAlgoId", clientAlgoId)

	code, body, err := c.doSigned(http.MethodGet, "/fapi/v1/algoOrder", v)
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("query algo order failed: %s", string(body))
	}

	// 调试日志：打印完整的Algo订单响应
	log.Printf("[币安客户端] Algo订单查询响应: code=%d, body=%s", code, string(body))

	var resp AlgoOrderResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	// 调试日志：打印解析结果
	log.Printf("[币安客户端] Algo订单解析结果: %+v", resp)

	return &resp, nil
}

// CancelAlgoOrder 取消Algo订单（用于止盈止损条件单）
func (c *Client) CancelAlgoOrder(symbol, clientAlgoId string) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("clientAlgoId", clientAlgoId)

	return c.doSigned(http.MethodDelete, "/fapi/v1/algoOrder", v)
}

// AlgoOrderResp Algo订单响应结构
type AlgoOrderResp struct {
	AlgoId       int64  `json:"algoId"`
	ClientAlgoId string `json:"clientAlgoId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	Type         string `json:"type"`
	Status       string `json:"algoStatus"` // 正确的状态字段名
	AlgoType     string `json:"algoType"`
	TriggerPrice string `json:"triggerPrice"`
	Quantity     string `json:"quantity"`
	WorkingType  string `json:"workingType"`
	AvgPrice     string `json:"avgPrice"`
	ExecutedQty  string `json:"executedQty"`
	UpdateTime   int64  `json:"updateTime"`
}

// === 新增：公共 GET 请求（无需签名） ===
func (c *Client) doPublic(method, path string, params url.Values) (int, []byte, error) {
	if params == nil {
		params = url.Values{}
	}
	full := c.Base + path + "?" + params.Encode()
	req, _ := http.NewRequest(method, full, nil)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, body, nil
}

type premiumIndexResp struct {
	Symbol    string `json:"symbol"`
	MarkPrice string `json:"markPrice"`
}

// === 新增：获取标记价（用于按百分比计算 TP/SL 基准价） ===
func (c *Client) GetMarkPrice(symbol string) (float64, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	code, body, err := c.doPublic(http.MethodGet, "/fapi/v1/premiumIndex", v)
	if err != nil || code >= 400 {
		return 0, fmt.Errorf("premiumIndex %d %s %v", code, string(body), err)
	}
	var r premiumIndexResp
	if e := json.Unmarshal(body, &r); e != nil {
		return 0, e
	}
	f, _ := strconv.ParseFloat(r.MarkPrice, 64)
	return f, nil
}

// === 新增：时间同步功能 ===

// SyncTime 同步服务器时间，计算时间偏移量
func (c *Client) SyncTime() error {
	code, body, err := c.doPublic(http.MethodGet, "/fapi/v1/time", nil)
	if err != nil || code >= 400 {
		return fmt.Errorf("time sync failed: code=%d body=%s err=%v", code, string(body), err)
	}

	var resp ServerTimeResp
	if e := json.Unmarshal(body, &resp); e != nil {
		return fmt.Errorf("failed to parse server time: %v", e)
	}

	// 计算时间偏移量：服务器时间 - 本地时间
	localTime := time.Now().UnixMilli()
	c.timeOffset = resp.ServerTime - localTime
	c.lastSyncTime = localTime

	return nil
}

// getTimestamp 获取同步后的时间戳
func (c *Client) getTimestamp() int64 {
	now := time.Now().UnixMilli()

	// 如果距离上次同步超过5分钟，则重新同步
	if now-c.lastSyncTime > c.syncInterval {
		if err := c.SyncTime(); err != nil {
			log.Printf("[WARN] Failed to sync time: %v, using local time", err)
			return now
		}
	}

	return now + c.timeOffset
}

// === 新增：获取交易所信息（包含支持的交易对）===
type ExchangeInfoResp struct {
	Symbols []struct {
		Symbol     string `json:"symbol"`
		Status     string `json:"status"`
		BaseAsset  string `json:"baseAsset"`
		QuoteAsset string `json:"quoteAsset"`
	} `json:"symbols"`
}

// GetExchangeInfo 获取交易所信息，包含所有支持的交易对
func (c *Client) GetExchangeInfo() (*ExchangeInfoResp, error) {
	code, body, err := c.doPublic(http.MethodGet, "/fapi/v1/exchangeInfo", nil)
	if err != nil || code >= 400 {
		return nil, fmt.Errorf("exchangeInfo failed: code=%d body=%s err=%v", code, string(body), err)
	}

	var resp ExchangeInfoResp
	if e := json.Unmarshal(body, &resp); e != nil {
		return nil, fmt.Errorf("failed to parse exchange info: %v", e)
	}

	return &resp, nil
}

// IsSymbolSupported 检查交易对是否被支持（状态为TRADING）
func (c *Client) IsSymbolSupported(symbol string) (bool, error) {
	info, err := c.GetExchangeInfo()
	if err != nil {
		return false, err
	}

	symbol = strings.ToUpper(symbol)
	for _, s := range info.Symbols {
		if s.Symbol == symbol && s.Status == "TRADING" {
			return true, nil
		}
	}

	return false, nil
}

// === 新增：挂 reduceOnly 的出场单（TAKE_PROFIT_MARKET / STOP_MARKET）===
// 根据币安2025-12-09迁移：止盈止损订单统一使用Algo订单API
func (c *Client) PlaceConditionalClose(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	reduceOnly bool,
	closePosition bool,
	clientOrderId string,
) (int, []byte, error) {
	// 止盈止损条件单：必须使用Algo订单API
	if orderType == "TAKE_PROFIT_MARKET" || orderType == "STOP_MARKET" ||
		orderType == "TAKE_PROFIT" || orderType == "STOP" {
		return c.placeAlgoConditionalOrder(symbol, side, orderType, stopPrice, quantity, workingType, clientOrderId)
	}

	// 其他订单类型使用传统API
	return c.placeTraditionalConditionalOrder(symbol, side, orderType, stopPrice, quantity, workingType, reduceOnly, closePosition, clientOrderId)
}

// placeAlgoConditionalOrder 使用正确的Algo订单API（根据币安2025-12-09迁移）
func (c *Client) placeAlgoConditionalOrder(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	clientOrderId string,
) (int, []byte, error) {
	v := url.Values{}

	// 关键参数：algoType=CONDITIONAL（标识这是条件单）
	v.Set("algoType", "CONDITIONAL")
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("side", strings.ToUpper(side))

	// Algo订单支持的条件单类型：STOP_MARKET, TAKE_PROFIT_MARKET
	v.Set("type", strings.ToUpper(orderType))

	// 验证必需参数
	if quantity == "" {
		return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'quantity' was not sent, was empty/null, or malformed."}`), fmt.Errorf("quantity parameter is empty for algo %s order", orderType)
	}
	if stopPrice == "" {
		return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'triggerPrice' was not sent, was empty/null, or malformed."}`), fmt.Errorf("triggerPrice parameter is empty for algo %s order", orderType)
	}

	v.Set("quantity", quantity)
	// 关键变更：使用triggerPrice而不是stopPrice
	v.Set("triggerPrice", stopPrice)

	// workingType 默认为 MARK_PRICE
	if workingType == "" {
		workingType = "MARK_PRICE"
	}
	v.Set("workingType", workingType)

	// 关键变更：使用clientAlgoId而不是clientOrderId
	if clientOrderId != "" {
		v.Set("clientAlgoId", clientOrderId)
	}

	// 调试日志：打印正确的Algo订单请求参数
	log.Printf("[币安客户端] 发送Algo条件%s订单请求: symbol=%s, side=%s, type=%s, triggerPrice=%s, quantity=%s, workingType=%s, clientAlgoId=%s",
		orderType, symbol, side, orderType, stopPrice, quantity, workingType, clientOrderId)

	// 使用正确的Algo订单端点：/fapi/v1/algoOrder
	code, body, err := c.doSigned(http.MethodPost, "/fapi/v1/algoOrder", v)

	// 调试日志：记录Algo订单响应
	if err != nil {
		log.Printf("[币安客户端] Algo条件%s订单请求失败: %v", orderType, err)
	} else {
		log.Printf("[币安客户端] Algo条件%s订单响应: code=%d, body_length=%d", orderType, code, len(body))
		if code >= 400 {
			log.Printf("[币安客户端] Algo条件%s订单错误响应: %s", orderType, string(body))
		}
	}

	return code, body, err
}

// placeTraditionalConditionalOrder 使用传统订单API下单（兼容老版本）
func (c *Client) placeTraditionalConditionalOrder(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	reduceOnly bool,
	closePosition bool,
	clientOrderId string,
) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("side", strings.ToUpper(side))
	v.Set("type", strings.ToUpper(orderType))

	// 对于TAKE_PROFIT_MARKET和STOP_MARKET，必须提供quantity和stopPrice
	if orderType == "TAKE_PROFIT_MARKET" || orderType == "STOP_MARKET" {
		if quantity == "" {
			return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'quantity' was not sent, was empty/null, or malformed."}`), fmt.Errorf("quantity parameter is empty for %s order", orderType)
		}
		if stopPrice == "" {
			return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'stopPrice' was not sent, was empty/null, or malformed."}`), fmt.Errorf("stopPrice parameter is empty for %s order", orderType)
		}
		v.Set("quantity", quantity)
		v.Set("stopPrice", stopPrice)
		v.Set("reduceOnly", "true") // 止盈止损订单应该是reduceOnly
	} else {
		// 其他订单类型使用原来的逻辑
		if reduceOnly {
			v.Set("reduceOnly", "true")
		}
		if closePosition {
			v.Set("closePosition", "true")
		}
	}

	if workingType == "" {
		workingType = "MARK_PRICE"
	}
	v.Set("workingType", workingType)

	if clientOrderId != "" {
		v.Set("newClientOrderId", clientOrderId)
	}

	// 调试日志：打印请求参数
	log.Printf("[币安客户端] 发送%s订单请求: symbol=%s, side=%s, type=%s, stopPrice=%s, quantity=%s, workingType=%s, clientOrderId=%s",
		orderType, symbol, side, orderType, stopPrice, quantity, workingType, clientOrderId)

	// 发送请求并记录详细的响应信息
	code, body, err := c.doSigned(http.MethodPost, "/fapi/v1/order", v)

	// 调试日志：记录响应
	if err != nil {
		log.Printf("[币安客户端] %s订单请求失败: %v", orderType, err)
	} else {
		log.Printf("[币安客户端] %s订单响应: code=%d, body_length=%d", orderType, code, len(body))
		if code >= 400 {
			log.Printf("[币安客户端] %s订单错误响应: %s", orderType, string(body))
		}
	}

	return code, body, err
}

// placeConditionalOrder 使用币安官方条件订单接口 (/fapi/v1/conditional/newOrder)
func (c *Client) placeConditionalOrder(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	reduceOnly bool,
	closePosition bool,
	clientOrderId string,
) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("side", strings.ToUpper(side))
	v.Set("type", strings.ToUpper(orderType))

	// 根据币安官方文档，条件订单的必需参数
	if stopPrice == "" {
		return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'stopPrice' was not sent, was empty/null, or malformed."}`), fmt.Errorf("stopPrice parameter is empty for conditional %s order", orderType)
	}
	if quantity == "" {
		return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'quantity' was not sent, was empty/null, or malformed."}`), fmt.Errorf("quantity parameter is empty for conditional %s order", orderType)
	}

	v.Set("stopPrice", stopPrice)
	v.Set("quantity", quantity)

	// workingType 默认为 MARK_PRICE（与scheduler保持一致）
	if workingType == "" {
		workingType = "MARK_PRICE"
	}
	v.Set("workingType", workingType)

	// 条件订单的其他参数
	if reduceOnly {
		v.Set("reduceOnly", "true")
	}
	if closePosition {
		v.Set("closePosition", "true")
	}
	if clientOrderId != "" {
		v.Set("newClientOrderId", clientOrderId)
	}

	// 调试日志：打印条件订单请求参数
	log.Printf("[币安客户端] 发送条件%s订单请求: symbol=%s, side=%s, type=%s, stopPrice=%s, quantity=%s, workingType=%s, clientOrderId=%s",
		orderType, symbol, side, orderType, stopPrice, quantity, workingType, clientOrderId)

	code, body, err := c.doSigned(http.MethodPost, "/fapi/v1/conditional/newOrder", v)

	// 调试日志：记录条件订单响应
	if err != nil {
		log.Printf("[币安客户端] 条件%s订单请求失败: %v", orderType, err)
	} else {
		log.Printf("[币安客户端] 条件%s订单响应: code=%d, body_length=%d", orderType, code, len(body))
		if code >= 400 {
			log.Printf("[币安客户端] 条件%s订单错误响应: %s", orderType, string(body))
		}
	}

	return code, body, err
}

// placeOcoConditionalOrder 使用OCO (One-Cancels-Other) 订单API
func (c *Client) placeOcoConditionalOrder(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	clientOrderId string,
) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))

	// OCO订单需要同时设置主要订单和止损订单
	// 对于止盈止损，我们设置主要订单为市价，关联订单为条件订单

	// 主要订单参数（市价单）
	v.Set("side", strings.ToUpper(side))
	v.Set("quantity", quantity)

	// 根据订单类型设置关联的止损或止盈订单
	if orderType == "STOP_MARKET" {
		// 设置止损订单
		v.Set("stopPrice", stopPrice)
		if workingType == "" {
			workingType = "MARK_PRICE"
		}
		v.Set("workingType", workingType)
		v.Set("stopClientOrderId", clientOrderId)
	} else if orderType == "TAKE_PROFIT_MARKET" {
		// 设置止盈订单 - OCO中没有直接的takeProfit参数，需要特殊处理
		// 实际上，OCO主要用于止损，我们可能需要不同的方法
		log.Printf("[币安客户端] OCO订单不支持TAKE_PROFIT_MARKET类型，跳过")
		return 400, []byte(`{"code":-1100,"msg":"OCO order does not support TAKE_PROFIT_MARKET type"}`), fmt.Errorf("OCO order does not support TAKE_PROFIT_MARKET")
	}

	// 调试日志：打印OCO订单请求参数
	log.Printf("[币安客户端] 发送OCO%s订单请求: symbol=%s, side=%s, stopPrice=%s, quantity=%s, workingType=%s, clientOrderId=%s",
		orderType, symbol, side, stopPrice, quantity, workingType, clientOrderId)

	code, body, err := c.doSigned(http.MethodPost, "/fapi/v1/order/oco", v)

	// 调试日志：记录OCO订单响应
	if err != nil {
		log.Printf("[币安客户端] OCO%s订单请求失败: %v", orderType, err)
	} else {
		log.Printf("[币安客户端] OCO%s订单响应: code=%d, body_length=%d", orderType, code, len(body))
		if code >= 400 {
			log.Printf("[币安客户端] OCO%s订单错误响应: %s", orderType, string(body))
		}
	}

	return code, body, err
}

// placePositionManagementOrder 使用持仓管理API设置止盈止损
func (c *Client) placePositionManagementOrder(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	clientOrderId string,
) (int, []byte, error) {
	// 持仓管理API可能需要不同的参数结构
	// 尝试设置止盈止损作为持仓属性

	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))

	// 根据订单类型设置不同的参数
	if orderType == "TAKE_PROFIT_MARKET" {
		// 设置止盈价格
		v.Set("takeProfitPrice", stopPrice)
		v.Set("takeProfitType", "MARKET")
	} else if orderType == "STOP_MARKET" {
		// 设置止损价格
		v.Set("stopLossPrice", stopPrice)
		v.Set("stopLossType", "MARKET")
	}

	if workingType == "" {
		workingType = "MARK_PRICE"
	}
	v.Set("workingType", workingType)

	if clientOrderId != "" {
		v.Set("clientOrderId", clientOrderId)
	}

	// 尝试持仓管理相关的端点
	pmEndpoints := []string{
		"/fapi/v1/position/takeProfitAndStopLoss", // 止盈止损设置
		"/fapi/v1/position/risk",                  // 持仓风险管理
		"/fapi/v1/position/protection",            // 持仓保护
	}

	log.Printf("[币安客户端] 发送持仓管理%s订单请求: symbol=%s, stopPrice=%s, workingType=%s, clientOrderId=%s",
		orderType, symbol, stopPrice, workingType, clientOrderId)

	var lastCode int
	var lastBody []byte
	var lastErr error

	for _, endpoint := range pmEndpoints {
		log.Printf("[币安客户端] 尝试持仓管理端点: %s", endpoint)
		code, body, err := c.doSigned(http.MethodPost, endpoint, v)
		lastCode, lastBody, lastErr = code, body, err

		if err == nil && code < 400 {
			return code, body, err
		}
		log.Printf("[币安客户端] 持仓管理端点 %s 返回错误 code=%d", endpoint, code)
	}

	// 如果所有持仓管理端点都失败，返回最后一个错误
	return lastCode, lastBody, lastErr
}

// placeCompatibleConditionalOrder 使用STOP/TAKE_PROFIT订单类型进行兼容性下单（已废弃，保留向下兼容）
func (c *Client) placeCompatibleConditionalOrder(
	symbol, side, orderType string,
	stopPrice string,
	quantity string,
	workingType string,
	reduceOnly bool,
	closePosition bool,
	clientOrderId string,
) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("side", strings.ToUpper(side))
	v.Set("type", strings.ToUpper(orderType))

	// 对于STOP和TAKE_PROFIT订单，需要设置不同的参数
	if orderType == "STOP" || orderType == "TAKE_PROFIT" {
		if stopPrice == "" {
			return 400, []byte(`{"code":-1102,"msg":"Mandatory parameter 'stopPrice' was not sent, was empty/null, or malformed."}`), fmt.Errorf("stopPrice parameter is empty for %s order", orderType)
		}
		v.Set("stopPrice", stopPrice)

		// STOP和TAKE_PROFIT订单默认使用MARKET执行
		v.Set("quantity", quantity)

		if workingType == "" {
			workingType = "MARK_PRICE"
		}
		v.Set("workingType", workingType)

		// 止盈止损订单应该是reduceOnly
		v.Set("reduceOnly", "true")
	} else {
		// 其他订单类型的处理
		if reduceOnly {
			v.Set("reduceOnly", "true")
		}
		if closePosition {
			v.Set("closePosition", "true")
		}
	}

	if clientOrderId != "" {
		v.Set("newClientOrderId", clientOrderId)
	}

	// 调试日志：打印兼容订单请求参数
	log.Printf("[币安客户端] 发送兼容%s订单请求: symbol=%s, side=%s, type=%s, stopPrice=%s, quantity=%s, workingType=%s, clientOrderId=%s",
		orderType, symbol, side, orderType, stopPrice, quantity, workingType, clientOrderId)

	code, body, err := c.doSigned(http.MethodPost, "/fapi/v1/order", v)

	// 调试日志：记录兼容订单响应
	if err != nil {
		log.Printf("[币安客户端] 兼容%s订单请求失败: %v", orderType, err)
	} else {
		log.Printf("[币安客户端] 兼容%s订单响应: code=%d, body_length=%d", orderType, code, len(body))
		if code >= 400 {
			log.Printf("[币安客户端] 兼容%s订单错误响应: %s", orderType, string(body))
		}
	}

	return code, body, err
}

// Position 持仓信息结构体
type Position struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	LiquidationPrice string `json:"liquidationPrice"`
	Leverage         string `json:"leverage"`
	MaxNotional      string `json:"maxNotional"`
	MarginType       string `json:"marginType"`
	IsolatedMargin   string `json:"isolatedMargin"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	PositionSide     string `json:"positionSide"`
	Notional         string `json:"notional"`
	IsolatedWallet   string `json:"isolatedWallet"`
	UpdateTime       int64  `json:"updateTime"`
}

// GetPositions 获取账户所有持仓信息
func (c *Client) GetPositions() ([]Position, error) {
	code, body, err := c.doSigned(http.MethodGet, "/fapi/v2/positionRisk", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("获取持仓信息失败: %w", err)
	}

	if code != 200 {
		// 构造请求URL用于调试（模拟doSigned中的逻辑）
		params := url.Values{}
		params.Set("timestamp", strconv.FormatInt(c.getTimestamp(), 10))
		params.Set("recvWindow", "10000")
		qs := params.Encode()
		sig := c.sign(qs)
		fullURL := c.Base + "/fapi/v2/positionRisk?" + qs + "&signature=" + sig

		// 打印详细的调试信息
		log.Printf("[持仓查询调试] 请求URL: %s", fullURL)
		log.Printf("[持仓查询调试] API Key: %s", c.Key[:10]+"...")
		log.Printf("[持仓查询调试] 时间戳: %d", c.getTimestamp())
		log.Printf("[持仓查询调试] 签名参数: %s", qs)
		log.Printf("[持仓查询调试] 签名: %s", sig)
		log.Printf("[持仓查询调试] 响应状态码: %d", code)
		log.Printf("[持仓查询调试] 响应内容: %s", string(body))

		// 提供curl命令示例
		log.Printf("[持仓查询调试] Curl测试命令: curl -H \"X-MBX-APIKEY: %s\" \"%s\"", c.Key, fullURL)

		return nil, fmt.Errorf("获取持仓信息API返回错误状态码: %d, 响应: %s", code, string(body))
	}

	var positions []Position
	if err := json.Unmarshal(body, &positions); err != nil {
		return nil, fmt.Errorf("解析持仓信息失败: %w", err)
	}

	// 过滤掉空仓位（positionAmt为0的）
	var activePositions []Position
	for _, pos := range positions {
		if pos.PositionAmt != "0" && pos.PositionAmt != "0.0" && pos.PositionAmt != "" {
			activePositions = append(activePositions, pos)
		}
	}

	return activePositions, nil
}

// AccountInfo 账户信息结构体
type AccountInfo struct {
	Assets []struct {
		Asset                  string `json:"asset"`
		WalletBalance          string `json:"walletBalance"`
		UnrealizedProfit       string `json:"unrealizedProfit"`
		MarginBalance          string `json:"marginBalance"`
		MaintMargin            string `json:"maintMargin"`
		InitialMargin          string `json:"initialMargin"`
		PositionInitialMargin  string `json:"positionInitialMargin"`
		OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
		CrossWalletBalance     string `json:"crossWalletBalance"`
		CrossUnPnl             string `json:"crossUnPnl"`
		AvailableBalance       string `json:"availableBalance"`
		MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
		MarginAvailable        bool   `json:"marginAvailable"`
		UpdateTime             int64  `json:"updateTime"`
	} `json:"assets"`
	Positions []struct {
		Symbol                 string `json:"symbol"`
		InitialMargin          string `json:"initialMargin"`
		MaintMargin            string `json:"maintMargin"`
		UnrealizedProfit       string `json:"unrealizedProfit"`
		PositionInitialMargin  string `json:"positionInitialMargin"`
		OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
		Leverage               string `json:"leverage"`
		Isolated               bool   `json:"isolated"`
		EntryPrice             string `json:"entryPrice"`
		MaxNotional            string `json:"maxNotional"`
		PositionSide           string `json:"positionSide"`
		PositionAmt            string `json:"positionAmt"`
		Notional               string `json:"notional"`
		IsolatedWallet         string `json:"isolatedWallet"`
		UpdateTime             int64  `json:"updateTime"`
	} `json:"positions"`
	CanDeposit                  bool   `json:"canDeposit"`
	CanTrade                    bool   `json:"canTrade"`
	CanWithdraw                 bool   `json:"canWithdraw"`
	FeeTier                     int    `json:"feeTier"`
	MaxWithdrawAmount           string `json:"maxWithdrawAmount"`
	TakerCommissionRate         string `json:"takerCommissionRate"`
	MakerCommissionRate         string `json:"makerCommissionRate"`
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
	MaxWithdrawAmount2          string `json:"maxWithdrawAmount"`
	UpdateTime                  int64  `json:"updateTime"`
}

// GetAccountInfo 获取账户信息
func (c *Client) GetAccountInfo() (*AccountInfo, error) {
	// 检查API密钥是否已配置
	if c.Key == "" || c.Key == "your_binance_api_key_here" {
		return nil, fmt.Errorf("API密钥未配置")
	}

	if c.Sec == "" || c.Sec == "your_binance_secret_key_here" {
		return nil, fmt.Errorf("API密钥未配置")
	}

	code, body, err := c.doSigned(http.MethodGet, "/fapi/v2/account", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("获取账户信息失败: %w", err)
	}

	if code != 200 {
		return nil, fmt.Errorf("获取账户信息API返回错误状态码: %d, 响应: %s", code, string(body))
	}

	var accountInfo AccountInfo
	if err := json.Unmarshal(body, &accountInfo); err != nil {
		return nil, fmt.Errorf("解析账户信息失败: %w", err)
	}

	return &accountInfo, nil
}
