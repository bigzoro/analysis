package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ===== WebSocket管理器 =====
// 管理实时涨幅榜的WebSocket连接，采用单连接多流架构

// RealtimeWSManager WebSocket连接管理器
type RealtimeWSManager struct {
	// 基础配置
	ctx     context.Context
	kind    string // 市场类型："spot" 或 "futures"
	baseURL string // WebSocket基础URL

	// 单连接管理
	conn        *websocket.Conn // 单个WebSocket连接
	isConnected bool            // 连接状态
	connMux     sync.RWMutex    // 连接锁

	// 订阅管理
	subscriptions map[string]bool // 当前订阅的交易对
	subMux        sync.RWMutex    // 订阅映射的读写锁

	// 控制参数
	reconnectInterval    time.Duration // 重连间隔
	maxReconnectAttempts int           // 最大重连次数
	heartbeatInterval    time.Duration // 心跳间隔

	// 连接控制
	reconnectCount int       // 重连次数
	lastMessage    time.Time // 最后消息时间
	isReconnecting bool      // 是否正在重连

	// 统计信息
	stats *WSStats // WebSocket统计信息

	// 错误处理增强
	errorHandler *ErrorHandler // 错误处理器
	retryConfig  RetryConfig   // 重试配置
}

// WSStats WebSocket统计信息
type WSStats struct {
	mu sync.RWMutex

	// 连接统计
	totalConnections  int64
	activeConnections int64
	totalReconnects   int64
	failedConnections int64

	// 消息统计
	messagesReceived int64
	messagesSent     int64
	lastMessageTime  time.Time

	// 错误统计
	errorsCount   int64
	lastError     error
	lastErrorTime time.Time
}

// NewRealtimeWSManager 创建WebSocket管理器
func NewRealtimeWSManager(ctx context.Context, kind string) *RealtimeWSManager {
	manager := &RealtimeWSManager{
		ctx:                  ctx,
		kind:                 kind,
		subscriptions:        make(map[string]bool),
		reconnectInterval:    5 * time.Second,
		maxReconnectAttempts: 10,
		heartbeatInterval:    30 * time.Second,
		stats:                &WSStats{},
		errorHandler:         NewErrorHandler(),
		retryConfig: RetryConfig{
			MaxRetries:    3,
			BaseDelay:     time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
	}

	// 根据市场类型设置WebSocket URL
	switch kind {
	case "spot":
		manager.baseURL = "wss://stream.binance.com:9443/ws/"
	case "futures":
		manager.baseURL = "wss://fstream.binance.com/ws/"
	default:
		manager.baseURL = "wss://stream.binance.com:9443/ws/"
	}

	// WebSocket管理器初始化完成
	return manager
}

// UpdateSubscriptions 更新订阅列表
func (m *RealtimeWSManager) UpdateSubscriptions(symbols []string, updateChan chan<- PriceUpdate) {
	// 更新订阅列表

	// 计算需要添加和移除的订阅
	toAdd, toRemove := m.calculateSubscriptionChanges(symbols)

	// 如果没有变化，直接返回
	if len(toAdd) == 0 && len(toRemove) == 0 {
		// 订阅列表无变化
		return
	}

	// 更新订阅映射
	m.updateSubscriptions(symbols)

	// 如果连接已建立，发送订阅更新命令
	if m.isConnected && m.conn != nil {
		if err := m.sendSubscriptionUpdate(toAdd, toRemove); err != nil {
			log.Printf("[RealtimeWSManager] 发送订阅更新失败: %v", err)
		}
	} else {
		// 如果连接未建立，启动连接过程
		// 连接未建立，开始建立连接
		go m.startConnection(updateChan)
	}

	// 更新统计信息
	m.stats.mu.Lock()
	m.stats.activeConnections = int64(len(symbols))
	m.stats.mu.Unlock()

	// 订阅更新完成
}

// calculateSubscriptionChanges 计算订阅变化
func (m *RealtimeWSManager) calculateSubscriptionChanges(newSymbols []string) (toAdd, toRemove []string) {
	m.subMux.Lock()
	defer m.subMux.Unlock()

	// 去重新订阅列表
	uniqueSymbols := make(map[string]bool)
	for _, symbol := range newSymbols {
		uniqueSymbols[symbol] = true
	}

	// 创建新订阅集合
	newSubs := make(map[string]bool)
	for symbol := range uniqueSymbols {
		newSubs[symbol] = true
	}

	// 找出需要添加的订阅（在新列表中但不在当前订阅中）
	for symbol := range uniqueSymbols {
		if !m.subscriptions[symbol] {
			toAdd = append(toAdd, symbol)
		}
	}

	// 找出需要移除的订阅（在当前订阅中但不在新列表中）
	for symbol := range m.subscriptions {
		found := false
		for _, newSymbol := range newSymbols {
			if symbol == newSymbol {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, symbol)
		}
	}

	return toAdd, toRemove
}

// updateSubscriptions 更新订阅映射
func (m *RealtimeWSManager) updateSubscriptions(symbols []string) {
	m.subMux.Lock()
	defer m.subMux.Unlock()

	// 清空当前订阅
	m.subscriptions = make(map[string]bool)

	// 添加新订阅
	for _, symbol := range symbols {
		m.subscriptions[symbol] = true
	}

	// 订阅映射已更新
}

// sendSubscriptionUpdate 发送订阅更新命令
func (m *RealtimeWSManager) sendSubscriptionUpdate(toAdd, toRemove []string) error {
	// 获取当前所有订阅
	m.subMux.RLock()
	allSubscriptions := make([]string, 0, len(m.subscriptions))
	for symbol := range m.subscriptions {
		allSubscriptions = append(allSubscriptions, symbol)
	}
	m.subMux.RUnlock()

	// 生成流名称列表
	streams := make([]string, len(allSubscriptions))
	for i, symbol := range allSubscriptions {
		streams[i] = m.convertSymbolToStream(symbol)
	}

	// 发送SUBSCRIBE命令订阅所有流
	subscribeMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": streams,
		"id":     time.Now().Unix(),
	}

	message, err := json.Marshal(subscribeMsg)
	if err != nil {
		return fmt.Errorf("序列化订阅消息失败: %w", err)
	}

	m.connMux.RLock()
	conn := m.conn
	m.connMux.RUnlock()

	if conn == nil {
		return fmt.Errorf("连接不存在")
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("发送订阅消息失败: %w", err)
	}

	// 更新统计信息
	m.stats.mu.Lock()
	m.stats.messagesSent++
	m.stats.mu.Unlock()

	// 发送订阅更新
	return nil
}

// startConnection 启动WebSocket连接
func (m *RealtimeWSManager) startConnection(updateChan chan<- PriceUpdate) {
	// 防止并发启动多个连接
	m.connMux.Lock()
	if m.isReconnecting {
		m.connMux.Unlock()
		return
	}
	m.isReconnecting = true
	m.connMux.Unlock()

	defer func() {
		m.connMux.Lock()
		m.isReconnecting = false
		m.connMux.Unlock()
	}()

	for {
		select {
		case <-m.ctx.Done():
			// 连接管理器停止
			return
		default:
			if m.connectAndListen(updateChan) {
				// 连接正常结束，可能是需要重新连接
				if m.reconnectCount < m.maxReconnectAttempts {
					time.Sleep(m.reconnectInterval)
					m.reconnectCount++
				} else {
					log.Printf("[RealtimeWSManager] 达到最大重连次数，停止重连")
					return
				}
			} else {
				// 连接失败或被取消
				return
			}
		}
	}
}

// connectAndListen 连接并监听消息
func (m *RealtimeWSManager) connectAndListen(updateChan chan<- PriceUpdate) bool {
	// 获取当前订阅列表
	m.subMux.RLock()
	subscriptions := make([]string, 0, len(m.subscriptions))
	for symbol := range m.subscriptions {
		subscriptions = append(subscriptions, symbol)
	}
	m.subMux.RUnlock()

	if len(subscriptions) == 0 {
		log.Printf("[RealtimeWSManager] 无订阅交易对，跳过连接")
		return false
	}

	// 生成组合流URL（多个流组合）
	streamNames := make([]string, len(subscriptions))
	for i, symbol := range subscriptions {
		streamNames[i] = m.convertSymbolToStream(symbol)
	}

	// Binance支持多流组合，格式为: stream1/stream2/stream3
	combinedStream := strings.Join(streamNames, "/")
	connURL := m.baseURL + combinedStream

	log.Printf("[RealtimeWSManager] 建立多流WebSocket连接: %s", connURL)

	// 建立WebSocket连接（带错误处理和重试）
	var conn *websocket.Conn
	err := m.executeWithRetry(func() error {
		dialer := websocket.DefaultDialer
		dialer.HandshakeTimeout = 10 * time.Second

		wsConn, _, dialErr := dialer.Dial(connURL, nil)
		if dialErr != nil {
			return dialErr
		}
		conn = wsConn
		return nil
	}, "WebSocketDial", true)

	if err != nil {
		log.Printf("[RealtimeWSManager] 建立WebSocket连接失败（已重试）: %v", err)

		// 更新连接状态
		m.connMux.Lock()
		m.conn = nil
		m.isConnected = false
		m.connMux.Unlock()

		// 更新统计信息
		m.stats.mu.Lock()
		m.stats.failedConnections++
		m.stats.errorsCount++
		m.stats.lastError = err
		m.stats.lastErrorTime = time.Now()
		m.stats.mu.Unlock()

		return true // 返回true表示需要重试
	}

	// 更新连接状态
	m.connMux.Lock()
	m.conn = conn
	m.isConnected = true
	m.lastMessage = time.Now()
	m.reconnectCount = 0 // 重置重连计数
	m.connMux.Unlock()

	// 多流WebSocket连接建立成功

	// 启动心跳goroutine
	heartbeatDone := make(chan struct{})
	go m.sendHeartbeat(conn, heartbeatDone)

	// 监听消息
	messageChan := make(chan []byte, 100)
	errorChan := make(chan error, 1)

	// 启动消息读取goroutine
	go func() {
		defer close(messageChan)
		defer close(errorChan)

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errorChan <- err
				return
			}
			messageChan <- message
		}
	}()

	// 处理消息循环
	for {
		select {
		case <-m.ctx.Done():
			conn.Close()
			close(heartbeatDone)
			return false

		case err := <-errorChan:
			log.Printf("[RealtimeWSManager] WebSocket错误: %v", err)
			conn.Close()
			close(heartbeatDone)

			// 更新连接状态
			m.connMux.Lock()
			m.conn = nil
			m.isConnected = false
			m.connMux.Unlock()

			// 更新统计信息
			m.stats.mu.Lock()
			m.stats.errorsCount++
			m.stats.lastError = err
			m.stats.lastErrorTime = time.Now()
			m.stats.mu.Unlock()

			return true

		case message := <-messageChan:
			m.processMessage(message, updateChan)

			// 更新最后消息时间
			m.connMux.Lock()
			m.lastMessage = time.Now()
			m.connMux.Unlock()

			// 更新统计信息
			m.stats.mu.Lock()
			m.stats.messagesReceived++
			m.stats.lastMessageTime = time.Now()
			m.stats.mu.Unlock()
		}
	}
}

// sendHeartbeat 发送心跳
func (m *RealtimeWSManager) sendHeartbeat(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(m.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// 发送ping消息
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				// 心跳失败，连接可能断开
				return
			}
		}
	}
}

// convertSymbolToStream 转换交易对为流名称
func (m *RealtimeWSManager) convertSymbolToStream(symbol string) string {
	// 统一转换为小写
	symbol = strings.ToLower(symbol)

	// 根据市场类型添加后缀
	switch m.kind {
	case "futures":
		// 期货交易对使用与现货相同的ticker格式
		// Binance期货WebSocket: btcusdt@ticker
		return symbol + "@ticker"
	default:
		// 现货交易对
		return symbol + "@ticker"
	}
}

// extractSymbolFromStream 从流名称中提取交易对符号
func (m *RealtimeWSManager) extractSymbolFromStream(stream string) string {
	// 移除@ticker后缀
	stream = strings.TrimSuffix(stream, "@ticker")

	// 转换为大写格式
	return strings.ToUpper(stream)
}

// processMessage 处理WebSocket消息
func (m *RealtimeWSManager) processMessage(message []byte, updateChan chan<- PriceUpdate) {
	// 移除频繁的消息接收日志

	// 解析消息
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("[RealtimeWSManager] 解析消息失败: %v", err)
		return
	}

	// 检查是否为订阅确认消息
	if id, exists := msg["id"]; exists {
		log.Printf("[RealtimeWSManager] 收到订阅确认: ID=%v", id)
		return
	}

	// 处理多流消息格式
	stream, hasStream := msg["stream"]
	data, hasData := msg["data"]

	if hasStream && hasData {
		// 多流格式: {"stream": "btcusdt@ticker", "data": {...}}
		streamStr := stream.(string)

		// 从流名称中提取交易对符号
		symbol := m.extractSymbolFromStream(streamStr)
		if symbol == "" {
			log.Printf("[RealtimeWSManager] ⚠️ 无法从流名称提取交易对: %s", streamStr)
			return
		}

		m.processTickerData(symbol, data.(map[string]interface{}), updateChan)
	} else if eventType, hasEvent := msg["e"]; hasEvent && eventType == "24hrTicker" {
		// 单流格式 - 直接的24hrTicker消息 (Binance实际格式)
		symbol, hasSymbol := msg["s"]
		if !hasSymbol {
			log.Printf("[RealtimeWSManager] ⚠️ 24hrTicker消息缺少交易对符号")
			return
		}

		symbolStr := symbol.(string)

		m.processTickerData(symbolStr, msg, updateChan)
	} else {
		// 可能是单流格式或其他消息
		log.Printf("[RealtimeWSManager] ⚠️ 未知消息格式: %v", getMapKeys(msg))
	}
}

// processTickerData 处理ticker数据
func (m *RealtimeWSManager) processTickerData(symbol string, data map[string]interface{}, updateChan chan<- PriceUpdate) {
	//log.Printf("[RealtimeWSManager] 🔍 开始处理ticker数据 %s, 数据键: %v", symbol, getMapKeys(data))

	// 提取价格信息
	lastPrice, err1 := m.extractFloat64(data, "c", "lastPrice")
	priceChangePercent, err2 := m.extractFloat64(data, "P", "priceChangePercent")
	volume, err3 := m.extractFloat64(data, "v", "volume")

	// 移除频繁的数据提取结果日志

	if err1 != nil {
		log.Printf("[RealtimeWSManager] ❌ 提取最新价格失败 %s: %v", symbol, err1)
		return
	}

	// 处理涨跌幅数据
	var changePercentPtr *float64
	if err2 == nil {
		changePercentPtr = &priceChangePercent
	} else {
		log.Printf("[RealtimeWSManager] ⚠️ 提取涨跌幅失败 %s: %v", symbol, err2)
		changePercentPtr = nil
	}

	// 处理成交量数据
	if err3 != nil {
		log.Printf("[RealtimeWSManager] ⚠️ 提取成交量失败 %s，使用默认值0: %v", symbol, err3)
		volume = 0
	}

	// 移除频繁的成功处理日志
	// 仅在debug模式下记录价格更新详情
	// if changePercentPtr != nil {
	//     log.Printf("[RealtimeWSManager] 📥 收到价格更新: %s = %.8f (%.2f%%), 成交量: %.2f", symbol, lastPrice, *changePercentPtr, volume)
	// } else {
	//     log.Printf("[RealtimeWSManager] 📥 收到价格更新: %s = %.8f (涨跌幅未设置), 成交量: %.2f", symbol, lastPrice, volume)
	// }

	// 创建价格更新对象
	update := PriceUpdate{
		Symbol:        symbol,
		Price:         lastPrice,
		Volume:        volume,
		ChangePercent: changePercentPtr,
		Timestamp:     time.Now(),
		Source:        "websocket",
	}

	// 发送到更新通道（非阻塞）
	select {
	case updateChan <- update:
		// 发送成功
	default:
		// 通道已满，丢弃更新（静默处理）
	}
}

// extractFloat64 提取float64字段
func (m *RealtimeWSManager) extractFloat64(data map[string]interface{}, keys ...string) (float64, error) {
	for _, key := range keys {
		if value, exists := data[key]; exists {
			switch v := value.(type) {
			case float64:
				return v, nil
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					return parsed, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("无法提取字段: %v", keys)
}

// executeWithRetry 带重试的执行器
func (m *RealtimeWSManager) executeWithRetry(operation func() error, operationName string, retryable bool) error {
	var lastErr error
	retryCount := 0

	for {
		err := operation()
		if err == nil {
			// 成功，重置错误统计
			m.errorHandler.RecordSuccess()
			return nil
		}

		lastErr = err
		m.errorHandler.RecordError(err, operationName, retryable)

		// 检查是否应该重试
		if !retryable || !m.errorHandler.ShouldRetry(retryCount, m.retryConfig) {
			break
		}

		// 计算重试延迟
		delay := m.errorHandler.CalculateRetryDelay(retryCount, m.retryConfig)
		log.Printf("[%s] 操作失败，重试%d/%d，延迟%v: %v",
			operationName, retryCount+1, m.retryConfig.MaxRetries, delay, err)

		select {
		case <-time.After(delay):
			retryCount++
		case <-m.ctx.Done():
			return m.ctx.Err()
		}
	}

	return lastErr
}

// GetStats 获取统计信息
func (m *RealtimeWSManager) GetStats() map[string]interface{} {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":  m.stats.totalConnections,
		"active_connections": m.stats.activeConnections,
		"total_reconnects":   m.stats.totalReconnects,
		"failed_connections": m.stats.failedConnections,
		"messages_received":  m.stats.messagesReceived,
		"messages_sent":      m.stats.messagesSent,
		"last_message_time":  m.stats.lastMessageTime,
		"errors_count":       m.stats.errorsCount,
		"last_error":         fmt.Sprintf("%v", m.stats.lastError),
		"last_error_time":    m.stats.lastErrorTime,
	}
}

// Close 关闭WebSocket管理器
func (m *RealtimeWSManager) Close() {
	log.Printf("[RealtimeWSManager] 正在关闭WebSocket管理器...")

	// 关闭连接
	m.connMux.Lock()
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	m.isConnected = false
	m.connMux.Unlock()

	log.Printf("[RealtimeWSManager] WebSocket管理器已关闭")
}

// getMapKeys 获取map的所有键名（用于调试）
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
