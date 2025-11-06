package server

import (
	pdb "analysis/internal/db"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
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
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			for c := range h.clients {
				// entity 维度分发（与你现有逻辑一致）
				if m.entity == "" || strings.EqualFold(m.entity, c.entity) {
					select {
					case c.send <- m.data:
					default:
						close(c.send)
						delete(h.clients, c)
					}
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

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
	OccurredAt time.Time `json:"occurred_at"`
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
		c.AbortWithStatus(http.StatusUnauthorized)
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
		})
	}
	payload, _ := json.Marshal(wsEnvelope{Type: "transfers", Data: out})
	hub.broadcast <- wsMessage{entity: entity, data: payload}
}

/*** ===== 历史列表：GET /transfers/recent ===== ***/
/*
  保持原有参数/返回：
  - Query: entity, chain, coin, limit(<=500), before_ts(RFC3339), before_id
  - Return: { "items": transferDTO[], "next_cursor": {before_ts, before_id} }
*/
func ListTransfers(gdb *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := strings.TrimSpace(c.Query("entity"))
		chain := strings.TrimSpace(c.Query("chain"))
		coin := strings.TrimSpace(c.Query("coin"))
		limit := 50
		if s := strings.TrimSpace(c.Query("limit")); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 500 {
				limit = v
			}
		}

		var rows []pdb.TransferEvent
		q := gdb.Model(&pdb.TransferEvent{})

		if entity != "" {
			q = q.Where("entity = ?", entity)
		}
		if chain != "" {
			q = q.Where("LOWER(chain) = ?", strings.ToLower(chain))
		}
		if coin != "" {
			q = q.Where("UPPER(coin) = ?", strings.ToUpper(coin))

			// ☆ 单币种时：直接按该币种阈值过滤
			if th, ok := coinThresholdUpper(strings.ToUpper(coin)); ok {
				q = q.Where("CAST(amount AS DECIMAL(38,18)) >= ?", th)
			}
		} else {
			// ☆ 未指定币种：对 BTC/ETH/SOL/USDT/USDC 套各自阈值；其它币种放行
			q = q.Where(`
				(
					  (UPPER(coin) = 'BTC'  AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (UPPER(coin) = 'ETH'  AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (UPPER(coin) = 'SOL'  AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (UPPER(coin) IN ('USDT','USDC') AND CAST(amount AS DECIMAL(38,18)) >= ?)
				   OR (UPPER(coin) NOT IN ('BTC','ETH','SOL','USDT','USDC'))
				)
			`, minBTC, minETH, minSOL, minU)
		}

		// 游标（与原逻辑保持一致）
		beforeTSStr := strings.TrimSpace(c.Query("before_ts"))
		beforeIDStr := strings.TrimSpace(c.Query("before_id"))

		var anchorTS time.Time
		var anchorID uint

		// parse before_ts (RFC3339)
		if beforeTSStr != "" {
			ts, err := time.Parse(time.RFC3339, beforeTSStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid before_ts (RFC3339 required)"})
				return
			}
			anchorTS = ts.UTC()
		}
		// parse before_id
		if beforeIDStr != "" {
			if v, err := strconv.Atoi(beforeIDStr); err == nil && v >= 0 {
				anchorID = uint(v)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid before_id"})
				return
			}
		}
		// 只给了 id，没有 ts，则用该 id 的 occurred_at 作为锚点
		if anchorTS.IsZero() && anchorID > 0 {
			var anchor pdb.TransferEvent
			qq := gdb.Model(&pdb.TransferEvent{}).Where("id = ?", anchorID)
			if entity != "" {
				qq = qq.Where("entity = ?", entity)
			}
			if chain != "" {
				qq = qq.Where("LOWER(chain) = ?", strings.ToLower(chain))
			}
			if coin != "" {
				qq = qq.Where("UPPER(coin) = ?", strings.ToUpper(coin))
			}
			if err := qq.First(&anchor).Error; err == nil {
				anchorTS = anchor.OccurredAt.UTC()
			}
		}

		// 锚点裁剪
		if !anchorTS.IsZero() && anchorID > 0 {
			q = q.Where("(occurred_at < ?) OR (occurred_at = ? AND id < ?)", anchorTS, anchorTS, anchorID)
		} else if !anchorTS.IsZero() {
			q = q.Where("occurred_at < ?", anchorTS)
		} else if anchorID > 0 {
			q = q.Where("id < ?", anchorID)
		}

		// 排序 + 限制
		if err := q.Order("occurred_at DESC").Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			})
		}

		// 下一页游标（最后一条）
		next := gin.H{}
		if n := len(rows); n > 0 {
			last := rows[n-1]
			next = gin.H{
				"before_ts": last.OccurredAt.UTC().Format(time.RFC3339),
				"before_id": last.ID,
			}
		}
		c.JSON(http.StatusOK, gin.H{"items": out, "next_cursor": next})
	}
}
