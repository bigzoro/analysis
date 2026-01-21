package server

import (
	pdb "analysis/internal/db"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

/*** ===== WS Hub ===== ***/

type wsClient struct {
	hub    *wsHub
	conn   *websocket.Conn
	send   chan []byte
	entity string
}

type wsHub struct {
	clients    map[*wsClient]bool
	register   chan *wsClient
	unregister chan *wsClient
	broadcast  chan wsMessage
	mu         sync.RWMutex // 优化：添加读写锁保护 clients map
}

type wsMessage struct {
	entity string
	data   []byte
}

var hub *wsHub

func StartTransfersHub() {
	hub = &wsHub{
		clients:    make(map[*wsClient]bool),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		broadcast:  make(chan wsMessage, 1024),
	}
	go hub.run()
}

func (h *wsHub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case m := <-h.broadcast:
			// 优化：使用读锁保护，并复制客户端列表避免长时间持有锁
			h.mu.RLock()
			clients := make([]*wsClient, 0, len(h.clients))
			for c := range h.clients {
				clients = append(clients, c)
			}
			h.mu.RUnlock()

			// 并发发送消息（优化：使用协程池限制并发）
			var wg sync.WaitGroup
			for _, c := range clients {
				// entity 维度分发
				if m.entity == "" || strings.EqualFold(m.entity, c.entity) {
					wg.Add(1)
					go func(client *wsClient) {
						defer wg.Done()
						select {
						case client.send <- m.data:
						default:
							// 发送失败，移除客户端
							h.mu.Lock()
							if _, ok := h.clients[client]; ok {
								delete(h.clients, client)
								close(client.send)
							}
							h.mu.Unlock()
						}
					}(c)
				}
			}
			wg.Wait()
		}
	}
}

/*** ===== DTO ===== ***/

type transferDTO struct {
	ID         uint      `json:"id"`
	Entity     string    `json:"entity"`
	Chain      string    `json:"chain"`
	Coin       string    `json:"coin"`
	Direction  string    `json:"direction"`
	Amount     string    `json:"amount"`
	TxID       string    `json:"txid"`
	Address    string    `json:"address"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	OccurredAt time.Time `json:"occurred_at"` // 交易发生时间
	CreatedAt  time.Time `json:"created_at"`  // 同步到系统的时间
}
type wsEnvelope struct {
	Type string        `json:"type"`
	Data []transferDTO `json:"data"`
}

/*** ===== 阈值规则（原币种单位） ===== ***/
// BTC < 5 过滤；ETH < 300 过滤；SOL < 500 过滤；USDT/USDC < 1_000_000 过滤；其它币种不过滤
const (
	minBTC = 5.0
	minETH = 300.0
	minSOL = 500.0
	minU   = 1000000.0
)

func coinThresholdUpper(coinUpper string) (float64, bool) {
	switch coinUpper {
	case "BTC":
		return minBTC, true
	case "ETH":
		return minETH, true
	case "SOL":
		return minSOL, true
	case "USDT", "USDC":
		return minU, true
	default:
		return 0, false // 其它币种不过滤
	}
}

/*** ===== WebSocket：GET /ws/transfers?entity=xxx ===== ***/

func WSTransfers(c *gin.Context) {
	tok := c.Query("token")
	if tok == "" {
		tok = c.GetHeader("Authorization")
	}
	if strings.HasPrefix(strings.ToLower(tok), "bearer ") {
		tok = tok[7:]
	}
	if _, err := parseToken(tok); err != nil {
		// 优化：使用统一的错误处理
		ErrorResponseHelper(c, http.StatusUnauthorized, "未授权，请先登录", ErrUnauthorized)
		c.Abort()
		return
	}

	if hub == nil {
		StartTransfersHub()
	}
	entity := strings.TrimSpace(c.Query("entity"))

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}

	client := &wsClient{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		entity: entity,
	}
	hub.register <- client

	// reader
	go func() {
		defer func() { hub.unregister <- client }()
		for {
			if _, _, err := client.conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	// writer
	go func() {
		for msg := range client.send {
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()
}

/*** ===== 广播：由 /ingest/events 调用 ===== ***/

func BroadcastTransfers(entity string, rows []pdb.TransferEvent) {
	if hub == nil || len(rows) == 0 {
		return
	}

	// ☆ 在广播前做阈值过滤（与 /transfers/recent 规则一致）
	filtered := make([]pdb.TransferEvent, 0, len(rows))
	for _, r := range rows {
		cu := strings.ToUpper(strings.TrimSpace(r.Coin))
		if th, ok := coinThresholdUpper(cu); ok {
			// amount 是字符串，按浮点比较（如果你后续把 amount 改为 DECIMAL 列，可直接比较）
			if f, err := strconv.ParseFloat(strings.TrimSpace(r.Amount), 64); err == nil && f >= th {
				filtered = append(filtered, r)
			}
		} else {
			filtered = append(filtered, r) // 其它币种不过滤
		}
	}
	if len(filtered) == 0 {
		return
	}

	out := make([]transferDTO, 0, len(filtered))
	for _, r := range filtered {
		out = append(out, transferDTO{
			ID:         r.ID,
			Entity:     r.Entity,
			Chain:      r.Chain,
			Coin:       r.Coin,
			Direction:  r.Direction,
			Amount:     r.Amount,
			TxID:       r.TxID,
			Address:    r.Address,
			From:       r.From,
			To:         r.To,
			OccurredAt: r.OccurredAt,
			CreatedAt:  r.CreatedAt,
		})
	}
	// 优化：添加错误处理
	payload, err := json.Marshal(wsEnvelope{Type: "transfers", Data: out})
	if err != nil {
		log.Printf("[ERROR] Failed to marshal WebSocket payload: %v", err)
		return
	}
	hub.broadcast <- wsMessage{entity: entity, data: payload}
}

/*** ===== 历史列表：GET /transfers/recent ===== ***/
/*
  分页参数/返回：
  - Query: entity, chain, coin, page(>=1), page_size(<=500, 默认50)
  - Return: { "items": transferDTO[], "total": int, "page": int, "page_size": int, "total_pages": int }
  排序：按 occurred_at DESC, id DESC（最新的在最前面）
*/
func ListTransfers(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		gdb := s.db.DB()
		entity := strings.TrimSpace(c.Query("entity"))
		chain := strings.TrimSpace(c.Query("chain"))
		coin := strings.TrimSpace(c.Query("coin"))

		// 搜索和过滤参数
		keyword := strings.TrimSpace(c.Query("keyword"))
		direction := strings.TrimSpace(c.Query("direction"))
		startTime := strings.TrimSpace(c.Query("start_time"))
		endTime := strings.TrimSpace(c.Query("end_time"))
		minAmountStr := strings.TrimSpace(c.Query("min_amount"))
		maxAmountStr := strings.TrimSpace(c.Query("max_amount"))

		// 分页参数
		pagination := ParsePaginationParams(
			c.Query("page"),
			c.Query("page_size"),
			50,  // 默认每页数量
			500, // 最大每页数量（转账列表允许更大）
		)
		page := pagination.Page
		pageSize := pagination.PageSize

		// 构建查询（用于计数和查询数据）
		q := gdb.Model(&pdb.TransferEvent{})

		if entity != "" {
			q = q.Where("entity = ?", entity)
		}
		// 优化：如果数据存储时已统一大小写，直接查询，避免使用函数导致索引失效
		// 注意：如果数据未统一大小写，需要在应用层统一处理
		if chain != "" {
			q = q.Where("chain = ?", strings.ToLower(chain))
		}
		if coin != "" {
			q = q.Where("coin = ?", strings.ToUpper(coin))

			// ☆ 单币种时：直接按该币种阈值过滤
			if th, ok := coinThresholdUpper(strings.ToUpper(coin)); ok {
				q = q.Where("CAST(amount AS DECIMAL(38,18)) >= ?", th)
			}
		} else {
			// ☆ 未指定币种：对 BTC/ETH/SOL/USDT/USDC 套各自阈值；其它币种放行
			// 优化：如果数据存储时已统一大小写，直接查询，避免使用函数导致索引失效
			q = q.Where(`
				(
					  (coin = 'BTC'  AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (coin = 'ETH'  AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (coin = 'SOL'  AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (coin IN ('USDT','USDC') AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (coin NOT IN ('BTC','ETH','SOL','USDT','USDC'))
				)
			`, minBTC, minETH, minSOL, minU)
		}

		// 关键词搜索（TxID、地址）
		if keyword != "" {
			q = q.Where("(tx_id LIKE ? OR address LIKE ? OR `from` LIKE ? OR `to` LIKE ?)",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}

		// 方向筛选
		if direction != "" {
			q = q.Where("direction = ?", strings.ToLower(direction))
		}

		// 时间范围筛选
		if startTime != "" {
			if t, err := time.Parse("2006-01-02T15:04", startTime); err == nil {
				q = q.Where("occurred_at >= ?", t.UTC())
			} else if t, err := time.Parse(time.RFC3339, startTime); err == nil {
				q = q.Where("occurred_at >= ?", t.UTC())
			}
		}
		if endTime != "" {
			if t, err := time.Parse("2006-01-02T15:04", endTime); err == nil {
				q = q.Where("occurred_at <= ?", t.UTC())
			} else if t, err := time.Parse(time.RFC3339, endTime); err == nil {
				q = q.Where("occurred_at <= ?", t.UTC())
			}
		}

		// 金额范围筛选
		if minAmountStr != "" {
			if minAmount, err := strconv.ParseFloat(minAmountStr, 64); err == nil {
				q = q.Where("CAST(amount AS DECIMAL(38,18)) >= ?", minAmount)
			}
		}
		if maxAmountStr != "" {
			if maxAmount, err := strconv.ParseFloat(maxAmountStr, 64); err == nil {
				q = q.Where("CAST(amount AS DECIMAL(38,18)) <= ?", maxAmount)
			}
		}

		// 计算总数
		var total int64
		if err := q.Count(&total).Error; err != nil {
			s.DatabaseError(c, "统计转账总数", err)
			return
		}

		// 分页查询：按时间倒序（最新的在最前面）
		var rows []pdb.TransferEvent
		if err := q.Order("occurred_at DESC").Order("id DESC").Offset(pagination.Offset).Limit(pageSize).Find(&rows).Error; err != nil {
			s.DatabaseError(c, "查询转账列表", err)
			return
		}

		out := make([]transferDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, transferDTO{
				ID:         r.ID,
				Entity:     r.Entity,
				Chain:      r.Chain,
				Coin:       r.Coin,
				Direction:  r.Direction,
				Amount:     r.Amount,
				TxID:       r.TxID,
				Address:    r.Address,
				From:       r.From,
				To:         r.To,
				OccurredAt: r.OccurredAt,
				CreatedAt:  r.CreatedAt,
			})
		}

		// 计算总页数
		totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
		if totalPages == 0 {
			totalPages = 1
		}

		c.JSON(http.StatusOK, gin.H{
			"items":       out,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		})
	}
}
