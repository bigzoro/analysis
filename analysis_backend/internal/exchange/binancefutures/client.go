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
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Base string // https://fapi.binance.com 或 https://testnet.binancefuture.com
	Key  string
	Sec  string
	HTTP *http.Client
}

func New(testnet bool) *Client {
	base := "https://fapi.binance.com"
	if testnet {
		base = "https://testnet.binancefuture.com"
	}
	
	// 优化：从环境变量安全地获取 API Key 和 Secret
	key := os.Getenv("BINANCE_FUTURES_API_KEY")
	sec := os.Getenv("BINANCE_FUTURES_SECRET")
	
	// 如果环境变量未设置，记录警告
	if key == "" || sec == "" {
		log.Printf("[WARN] Binance Futures API credentials not configured. Set BINANCE_FUTURES_API_KEY and BINANCE_FUTURES_SECRET environment variables.")
		// 注意：这里不强制退出，因为某些功能可能不需要 Binance API
		// 但在实际使用时，如果没有凭证，API 调用会失败
	}
	
	return &Client{
		Base: base,
		Key:  key,
		Sec:  sec,
		HTTP: &http.Client{Timeout: 15 * time.Second},
	}
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
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")
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

func (c *Client) SetLeverage(symbol string, lev int) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("leverage", strconv.Itoa(lev))
	return c.doSigned(http.MethodPost, "/fapi/v1/leverage", v)
}

func (c *Client) PlaceOrder(symbol, side, otype, qty, price string, reduceOnly bool) (int, []byte, error) {
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
	return c.doSigned(http.MethodPost, "/fapi/v1/order", v)
}

func ParsePlaceOrderResp(b []byte) (*PlaceOrderResp, error) {
	var r PlaceOrderResp
	err := json.Unmarshal(b, &r)
	return &r, err
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

// === 新增：挂 reduceOnly 的出场单（TAKE_PROFIT_MARKET / STOP_MARKET）===
func (c *Client) PlaceConditionalClose(
	symbol, side, orderType string,
	stopPrice string,
	workingType string,
	reduceOnly bool,
	closePosition bool,
	clientOrderId string,
) (int, []byte, error) {
	v := url.Values{}
	v.Set("symbol", strings.ToUpper(symbol))
	v.Set("side", strings.ToUpper(side))
	v.Set("type", strings.ToUpper(orderType)) // TAKE_PROFIT_MARKET / STOP_MARKET
	v.Set("stopPrice", stopPrice)
	if workingType == "" {
		workingType = "MARK_PRICE"
	}
	v.Set("workingType", workingType)
	if reduceOnly {
		v.Set("reduceOnly", "true")
	}
	if closePosition {
		v.Set("closePosition", "true")
	}
	if clientOrderId != "" {
		v.Set("newClientOrderId", clientOrderId)
	}
	return c.doSigned(http.MethodPost, "/fapi/v1/order", v)
}
