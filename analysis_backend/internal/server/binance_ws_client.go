package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceWSClient 币安WebSocket客户端
type BinanceWSClient struct {
	conn           *websocket.Conn
	mu             sync.RWMutex
	ticker24h      map[string]Binance24hrTicker // symbol -> ticker data
	subscriptions  map[string]bool              // 已订阅的流
	reconnectTimer *time.Timer
	isConnected    bool
	ctx            context.Context
	cancel         context.CancelFunc
	updateCallback func() // 数据更新回调函数
}

// Binance24hrTicker 24小时统计数据
type Binance24hrTicker struct {
	EventType             string `json:"e"` // 事件类型
	EventTime             int64  `json:"E"` // 事件时间
	Symbol                string `json:"s"` // 交易对
	PriceChange           string `json:"p"` // 价格变动
	PriceChangePercent    string `json:"P"` // 价格变动百分比
	WeightedAvgPrice      string `json:"w"` // 加权平均价
	FirstPrice            string `json:"x"` // 首笔成交价格
	LastPrice             string `json:"c"` // 最后成交价格
	LastQty               string `json:"Q"` // 最后成交数量
	BestBidPrice          string `json:"b"` // 买方最优挂单价格
	BestBidQty            string `json:"B"` // 买方最优挂单数量
	BestAskPrice          string `json:"a"` // 卖方最优挂单价格
	BestAskQty            string `json:"A"` // 卖方最优挂单数量
	OpenPrice             string `json:"o"` // 开盘价
	HighPrice             string `json:"h"` // 最高价
	LowPrice              string `json:"l"` // 最低价
	TotalTradedBaseAsset  string `json:"v"` // 成交量(基础资产)
	TotalTradedQuoteAsset string `json:"q"` // 成交量(计价资产)
	StatisticsOpenTime    int64  `json:"O"` // 统计开始时间
	StatisticsCloseTime   int64  `json:"C"` // 统计结束时间
	FirstTradeId          int64  `json:"F"` // 首笔成交ID
	LastTradeId           int64  `json:"L"` // 最后成交ID
	TotalNumberOfTrades   int64  `json:"n"` // 成交笔数
}

// NewBinanceWSClient 创建币安WebSocket客户端
func NewBinanceWSClient() *BinanceWSClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &BinanceWSClient{
		ticker24h:     make(map[string]Binance24hrTicker),
		subscriptions: make(map[string]bool),
		isConnected:   false,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// SetUpdateCallback 设置数据更新回调函数
func (bwc *BinanceWSClient) SetUpdateCallback(callback func()) {
	bwc.mu.Lock()
	defer bwc.mu.Unlock()
	bwc.updateCallback = callback
}

// Connect 连接到币安WebSocket
func (bwc *BinanceWSClient) Connect(marketType string) error {
	bwc.mu.Lock()
	defer bwc.mu.Unlock()

	var wsURL string
	switch marketType {
	case "spot":
		wsURL = "wss://stream.binance.com:9443/ws/!ticker@arr"
	case "futures":
		// 可以选择USDT期货或币本位期货
		wsURL = "wss://fstream.binance.com/ws/!ticker@arr"
	case "coin_futures":
		wsURL = "wss://dstream.binance.com/ws/!ticker@arr"
	default:
		return fmt.Errorf("不支持的市场类型: %s", marketType)
	}

	log.Printf("[BinanceWS] 连接到 %s", wsURL)

	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("解析WebSocket URL失败: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("连接WebSocket失败: %w", err)
	}

	bwc.conn = conn
	bwc.isConnected = true

	// 启动消息处理协程
	go bwc.handleMessages()

	log.Printf("[BinanceWS] 成功连接到币安%s市场WebSocket", marketType)
	return nil
}

// SubscribeTicker24h 订阅24小时统计数据
func (bwc *BinanceWSClient) SubscribeTicker24h(symbols []string, marketType string) error {
	bwc.mu.Lock()
	defer bwc.mu.Unlock()

	if !bwc.isConnected {
		return fmt.Errorf("WebSocket未连接")
	}

	// 对于全市场ticker流，我们已经订阅了所有交易对
	// 这里可以根据需要订阅特定的交易对
	for _, symbol := range symbols {
		streamName := strings.ToLower(symbol) + "@ticker"
		if !bwc.subscriptions[streamName] {
			// 发送订阅消息
			subMsg := map[string]interface{}{
				"method": "SUBSCRIBE",
				"params": []string{streamName},
				"id":     time.Now().Unix(),
			}

			if err := bwc.conn.WriteJSON(subMsg); err != nil {
				log.Printf("[BinanceWS] 订阅失败 %s: %v", streamName, err)
				continue
			}

			bwc.subscriptions[streamName] = true
			log.Printf("[BinanceWS] 订阅成功: %s", streamName)
		}
	}

	return nil
}

// handleMessages 处理WebSocket消息
func (bwc *BinanceWSClient) handleMessages() {
	defer func() {
		bwc.mu.Lock()
		bwc.isConnected = false
		bwc.mu.Unlock()
	}()

	for {
		select {
		case <-bwc.ctx.Done():
			return
		default:
			_, message, err := bwc.conn.ReadMessage()
			if err != nil {
				log.Printf("[BinanceWS] 读取消息失败: %v", err)
				bwc.scheduleReconnect()
				return
			}

			bwc.processMessage(message)
		}
	}
}

// processMessage 处理接收到的消息
func (bwc *BinanceWSClient) processMessage(message []byte) {
	var dataUpdated bool

	// 尝试解析为单个ticker消息
	var singleTicker Binance24hrTicker
	if err := json.Unmarshal(message, &singleTicker); err == nil && singleTicker.EventType == "24hrTicker" {
		bwc.mu.Lock()
		bwc.ticker24h[singleTicker.Symbol] = singleTicker
		bwc.mu.Unlock()
		dataUpdated = true
		return
	}

	// 尝试解析为ticker数组（全市场数据）
	var tickerArray []Binance24hrTicker
	if err := json.Unmarshal(message, &tickerArray); err == nil {
		bwc.mu.Lock()
		for _, ticker := range tickerArray {
			bwc.ticker24h[ticker.Symbol] = ticker
		}
		bwc.mu.Unlock()
		//log.Printf("[BinanceWS] 更新了 %d 个交易对的24hr数据", len(tickerArray))
		dataUpdated = true
	}

	// 其他消息类型（订阅确认等）
	if !dataUpdated {
		log.Printf("[BinanceWS] 收到其他消息: %s", string(message))
	}

	// 如果数据有更新，调用回调函数
	if dataUpdated && bwc.updateCallback != nil {
		go bwc.updateCallback()
	}
}

// GetTicker24h 获取24小时统计数据
func (bwc *BinanceWSClient) GetTicker24h(symbol string) (Binance24hrTicker, bool) {
	bwc.mu.RLock()
	defer bwc.mu.RUnlock()

	ticker, exists := bwc.ticker24h[symbol]
	return ticker, exists
}

// GetAllTicker24h 获取所有24小时统计数据
func (bwc *BinanceWSClient) GetAllTicker24h() map[string]Binance24hrTicker {
	bwc.mu.RLock()
	defer bwc.mu.RUnlock()

	result := make(map[string]Binance24hrTicker)
	for k, v := range bwc.ticker24h {
		result[k] = v
	}
	return result
}

// scheduleReconnect 调度重连
func (bwc *BinanceWSClient) scheduleReconnect() {
	bwc.mu.Lock()
	defer bwc.mu.Unlock()

	if bwc.reconnectTimer != nil {
		bwc.reconnectTimer.Stop()
	}

	bwc.reconnectTimer = time.AfterFunc(5*time.Second, func() {
		log.Printf("[BinanceWS] 尝试重连...")
		// 这里需要重新连接的逻辑
		// 由于需要marketType参数，这里只是示例
	})
}

// Close 关闭连接
func (bwc *BinanceWSClient) Close() error {
	bwc.cancel()

	bwc.mu.Lock()
	defer bwc.mu.Unlock()

	if bwc.reconnectTimer != nil {
		bwc.reconnectTimer.Stop()
	}

	if bwc.conn != nil {
		return bwc.conn.Close()
	}

	return nil
}

// IsConnected 检查连接状态
func (bwc *BinanceWSClient) IsConnected() bool {
	bwc.mu.RLock()
	defer bwc.mu.RUnlock()
	return bwc.isConnected
}
