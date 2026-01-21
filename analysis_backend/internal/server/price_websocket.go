package server

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// PriceWebSocketClient 价格WebSocket客户端
type PriceWebSocketClient struct {
	conn       *websocket.Conn
	send       chan []byte
	symbols    []string
	lastActive time.Time
	mu         sync.RWMutex
	writeMu    sync.Mutex // 保护直接写入操作
}

// PriceWebSocketHub 价格WebSocket Hub
type PriceWebSocketHub struct {
	clients    map[*PriceWebSocketClient]bool
	broadcast  chan []byte
	register   chan *PriceWebSocketClient
	unregister chan *PriceWebSocketClient
	mu         sync.RWMutex
}

// 全局价格Hub
var priceHub *PriceWebSocketHub
var priceHubOnce sync.Once

// GetPriceHub 获取价格WebSocket Hub单例
func GetPriceHub() *PriceWebSocketHub {
	priceHubOnce.Do(func() {
		priceHub = &PriceWebSocketHub{
			clients:    make(map[*PriceWebSocketClient]bool),
			broadcast:  make(chan []byte, 1024),
			register:   make(chan *PriceWebSocketClient),
			unregister: make(chan *PriceWebSocketClient),
		}
		go priceHub.run()
	})
	return priceHub
}

// run 运行Hub
func (h *PriceWebSocketHub) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[PriceWebSocket] 客户端注册，当前连接数: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[PriceWebSocket] 客户端注销，当前连接数: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*PriceWebSocketClient, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			// 并发广播消息
			for _, client := range clients {
				select {
				case client.send <- message:
				default:
					// 发送失败，移除客户端
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
				}
			}

		case <-ticker.C:
			// 清理非活跃客户端
			h.mu.Lock()
			for client := range h.clients {
				client.mu.RLock()
				if time.Since(client.lastActive) > 5*time.Minute {
					delete(h.clients, client)
					close(client.send)
					log.Printf("[PriceWebSocket] 清理非活跃客户端")
				}
				client.mu.RUnlock()
			}
			h.mu.Unlock()
		}
	}
}

// WSPrices WebSocket价格实时推送
func (s *Server) WSPrices(c *gin.Context) {
	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[PriceWebSocket] 升级失败: %v", err)
		return
	}
	defer ws.Close()

	clientIP := c.ClientIP()
	log.Printf("[PriceWebSocket] 新连接建立: %s", clientIP)

	// 创建客户端
	client := &PriceWebSocketClient{
		conn:       ws,
		send:       make(chan []byte, 256),
		symbols:    []string{}, // 默认订阅热门币种
		lastActive: time.Now(),
	}

	// 注册客户端
	hub := GetPriceHub()
	hub.register <- client

	// 启动读取和写入协程
	go client.writePump()
	go client.readPump(hub)

	// 读取客户端订阅消息
	_, message, err := ws.ReadMessage()
	if err != nil {
		log.Printf("[PriceWebSocket] 读取订阅消息失败: %v", err)
		hub.unregister <- client
		return
	}

	// 解析订阅请求
	var subscription struct {
		Action  string   `json:"action"`
		Symbols []string `json:"symbols"`
	}

	if err := json.Unmarshal(message, &subscription); err != nil {
		log.Printf("[PriceWebSocket] 解析订阅消息失败: %v", err)
		client.writeMu.Lock()
		ws.WriteJSON(gin.H{"error": "无效的订阅格式"})
		client.writeMu.Unlock()
		hub.unregister <- client
		return
	}

	if subscription.Action != "subscribe" {
		client.writeMu.Lock()
		ws.WriteJSON(gin.H{"error": "不支持的操作"})
		client.writeMu.Unlock()
		hub.unregister <- client
		return
	}

	// 验证并设置订阅的币种
	if err := s.validatePriceSubscription(&subscription); err != nil {
		log.Printf("[PriceWebSocket] 订阅参数验证失败: %v", err)
		client.writeMu.Lock()
		ws.WriteJSON(gin.H{"error": err.Error()})
		client.writeMu.Unlock()
		hub.unregister <- client
		return
	}

	client.mu.Lock()
	client.symbols = subscription.Symbols
	client.mu.Unlock()

	log.Printf("[PriceWebSocket] 客户端订阅: symbols=%v", subscription.Symbols)

	// 发送确认消息
	client.writeMu.Lock()
	ws.WriteJSON(gin.H{
		"type":    "subscription_confirmed",
		"message": "价格实时订阅成功",
		"config": gin.H{
			"symbols": subscription.Symbols,
		},
	})
	client.writeMu.Unlock()

	// 保持连接，直到客户端断开
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			// 定期更新活跃时间
			client.mu.Lock()
			client.lastActive = time.Now()
			client.mu.Unlock()
		}
	}
}

// writePump 写入协程
func (c *PriceWebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second) // WebSocket ping间隔
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.writeMu.Lock()
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.writeMu.Unlock()
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.writeMu.Unlock()
				log.Printf("[PriceWebSocket] 写入消息失败: %v", err)
				return
			}
			c.writeMu.Unlock()

		case <-ticker.C:
			// 发送ping消息保持连接
			c.writeMu.Lock()
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.writeMu.Unlock()
				log.Printf("[PriceWebSocket] 发送ping失败: %v", err)
				return
			}
			c.writeMu.Unlock()
		}
	}
}

// readPump 读取协程
func (c *PriceWebSocketClient) readPump(hub *PriceWebSocketHub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[PriceWebSocket] 连接异常关闭: %v", err)
			}
			break
		}
	}
}

// validatePriceSubscription 验证价格订阅参数
func (s *Server) validatePriceSubscription(sub *struct {
	Action  string   `json:"action"`
	Symbols []string `json:"symbols"`
}) error {
	// 如果没有指定币种，使用默认热门币种
	if len(sub.Symbols) == 0 {
		sub.Symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT", "SOLUSDT", "DOTUSDT"}
		return nil
	}

	if len(sub.Symbols) > 50 {
		return fmt.Errorf("订阅币种数量不能超过50个")
	}

	// 验证并清理币种名称
	validSymbols := make([]string, 0, len(sub.Symbols))
	symbolSet := make(map[string]bool)

	for _, symbol := range sub.Symbols {
		cleanSymbol := strings.TrimSpace(strings.ToUpper(symbol))
		if cleanSymbol == "" {
			continue
		}

		// 检查格式 (允许数字和字母，以及_PERP后缀)
		if !regexp.MustCompile(`^[A-Z0-9]+(?:_PERP)?$`).MatchString(cleanSymbol) {
			return fmt.Errorf("无效的币种名称: %s", symbol)
		}

		// 去重
		if !symbolSet[cleanSymbol] {
			symbolSet[cleanSymbol] = true
			validSymbols = append(validSymbols, cleanSymbol)
		}
	}

	if len(validSymbols) == 0 {
		return fmt.Errorf("没有有效的币种名称")
	}

	sub.Symbols = validSymbols
	return nil
}

// BinancePriceStreamer Binance价格数据流
type BinancePriceStreamer struct {
	conn      *websocket.Conn
	hub       *PriceWebSocketHub
	symbols   []string
	reconnect bool
	mu        sync.RWMutex
}

// StartBinancePriceStreaming 启动Binance价格数据流
func (s *Server) StartBinancePriceStreaming(symbols []string) error {
	streamer := &BinancePriceStreamer{
		hub:       GetPriceHub(),
		symbols:   symbols,
		reconnect: true,
	}

	go streamer.startStreaming()
	log.Printf("[BinanceStreamer] 启动价格数据流，订阅币种: %v", symbols)
	return nil
}

// startStreaming 开始流式传输
func (b *BinancePriceStreamer) startStreaming() {
	for b.reconnect {
		if err := b.connectAndStream(); err != nil {
			log.Printf("[BinanceStreamer] 连接失败，重试中: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}

// connectAndStream 连接并开始流式传输
func (b *BinancePriceStreamer) connectAndStream() error {
	// 使用基础URL，不包含streams参数
	url := "wss://stream.binance.com:9443/ws"

	// 连接WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("连接Binance WebSocket失败: %w", err)
	}

	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		if b.conn != nil {
			b.conn.Close()
			b.conn = nil
		}
		b.mu.Unlock()
	}()

	// 发送订阅消息
	if err := b.sendSubscription(conn); err != nil {
		conn.Close()
		return fmt.Errorf("发送订阅消息失败: %w", err)
	}

	// 发送心跳
	go b.sendHeartbeat()

	// 设置pong处理器
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 读取消息
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("读取消息失败: %w", err)
		}

		// 处理不同类型的消息
		switch messageType {
		case websocket.TextMessage:
			var message map[string]interface{}
			if err := json.Unmarshal(data, &message); err != nil {
				log.Printf("[BinanceStreamer] 解析消息失败: %v", err)
				continue
			}

			// 处理价格更新
			if err := b.handlePriceUpdate(message); err != nil {
				log.Printf("[BinanceStreamer] 处理价格更新失败: %v", err)
			}
		case websocket.CloseMessage:
			log.Printf("[BinanceStreamer] 收到关闭消息")
			return nil
		}
	}
}

// sendSubscription 发送订阅消息
func (b *BinancePriceStreamer) sendSubscription(conn *websocket.Conn) error {
	// 构建订阅参数
	params := make([]string, len(b.symbols))
	for i, symbol := range b.symbols {
		params[i] = strings.ToLower(symbol) + "@ticker"
	}

	// 发送订阅消息
	subscription := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": params,
		"id":     1,
	}

	b.mu.Lock()
	if err := conn.WriteJSON(subscription); err != nil {
		b.mu.Unlock()
		return fmt.Errorf("发送订阅消息失败: %w", err)
	}
	b.mu.Unlock()

	log.Printf("[BinanceStreamer] 发送订阅请求，币种数量: %d", len(b.symbols))
	return nil
}

// sendHeartbeat 发送心跳
func (b *BinancePriceStreamer) sendHeartbeat() {
	ticker := time.NewTicker(30 * time.Minute) // Binance推荐30分钟ping一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.mu.Lock()
			if b.conn != nil {
				// 发送ping帧
				if err := b.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					log.Printf("[BinanceStreamer] 发送心跳失败: %v", err)
				}
			}
			b.mu.Unlock()
		}
	}
}

// handlePriceUpdate 处理价格更新
func (b *BinancePriceStreamer) handlePriceUpdate(message map[string]interface{}) error {
	// 检查是否是订阅确认消息
	if _, hasResult := message["result"]; hasResult {
		//log.Printf("[BinanceStreamer] 收到订阅确认消息")
		return nil // 订阅确认消息，正常跳过
	}

	var data map[string]interface{}

	// 检查消息格式：可能是包装在data字段中，也可能是直接的数据
	if dataField, ok := message["data"].(map[string]interface{}); ok {
		// 标准格式：{"stream": "...", "data": {...}}
		data = dataField
	} else if eventType, hasEvent := message["e"]; hasEvent && eventType == "24hrTicker" {
		// 直接格式：ticker数据直接在消息中
		data = message
	} else {
		// 可能是其他类型的消息，静默跳过
		log.Printf("[BinanceStreamer] 收到未知消息类型: %v", message)
		return nil
	}

	// 提取完整的价格和统计信息
	symbol, _ := data["s"].(string)
	price, _ := data["c"].(string)            // 最新成交价
	changePercent, _ := data["P"].(string)    // 24h涨跌幅百分比
	changeAmount, _ := data["p"].(string)     // 24h涨跌金额
	weightedAvgPrice, _ := data["w"].(string) // 加权平均价
	prevClosePrice, _ := data["x"].(string)   // 前一交易日收盘价
	lastQty, _ := data["Q"].(string)          // 最新成交量
	open24h, _ := data["o"].(string)          // 24h开盘价
	high24h, _ := data["h"].(string)          // 24h最高价
	low24h, _ := data["l"].(string)           // 24h最低价
	volume24h, _ := data["v"].(string)        // 24h成交量
	quoteVolume24h, _ := data["q"].(string)   // 24h成交额
	trades24h, _ := data["n"].(float64)       // 24h成交笔数

	// 验证必要字段
	if symbol == "" || price == "" {
		return nil // 跳过无效数据
	}

	// 构建完整的广播消息，包含所有统计数据
	update := gin.H{
		"type":               "price_update",
		"timestamp":          time.Now().Unix(),
		"symbol":             symbol,
		"price":              price,
		"change_percent":     changePercent,
		"change_amount":      changeAmount,
		"weighted_avg_price": weightedAvgPrice,
		"prev_close_price":   prevClosePrice,
		"last_qty":           lastQty,
		"open_24h":           open24h,
		"high_24h":           high24h,
		"low_24h":            low24h,
		"volume_24h":         volume24h,
		"quote_volume_24h":   quoteVolume24h,
		"trades_24h":         int(trades24h),
	}

	// 序列化消息
	messageBytes, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 广播给所有客户端
	b.hub.broadcast <- messageBytes

	// 记录详细的统计信息日志
	//log.Printf("[BinanceStreamer] 广播价格更新: %s = %s (%s%%), 24h区间: %s-%s, 成交量: %s, 笔数: %d",
	//	symbol, price, changePercent, low24h, high24h, volume24h, int(trades24h))

	return nil
}

// Stop 停止流式传输
func (b *BinancePriceStreamer) Stop() {
	b.mu.Lock()
	b.reconnect = false
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
	b.mu.Unlock()
	log.Printf("[BinanceStreamer] 停止价格数据流")
}
