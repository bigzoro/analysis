package server

import (
	pdb "analysis/internal/db"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type scheduleReq struct {
	// Bracket 扩展参数
	BracketEnabled bool    `json:"bracket_enabled"`
	TPPercent      float64 `json:"tp_percent"`
	SLPercent      float64 `json:"sl_percent"`
	TPPrice        string  `json:"tp_price"`
	SLPrice        string  `json:"sl_price"`
	WorkingType    string  `json:"working_type"`

	Exchange    string `json:"exchange"`     // "binance_futures"
	Testnet     bool   `json:"testnet"`      // true=走测试网
	Symbol      string `json:"symbol"`       // e.g. BTCUSDT
	Side        string `json:"side"`         // BUY/SELL
	OrderType   string `json:"order_type"`   // MARKET/LIMIT
	Quantity    string `json:"quantity"`     // 下单数量(合约张/币数，按交易所规则)
	Price       string `json:"price"`        // 限价单需要
	Leverage    int    `json:"leverage"`     // 0=不设置
	ReduceOnly  bool   `json:"reduce_only"`  // 可选
	TriggerTime string `json:"trigger_time"` // ISO8601，本地前端传 UTC 或带时区
}

func (s *Server) CreateScheduledOrder(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req scheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.Exchange == "" || req.Symbol == "" || req.Side == "" || req.OrderType == "" || req.Quantity == "" || req.TriggerTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}
	tt, err := time.Parse(time.RFC3339, req.TriggerTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trigger_time (RFC3339)"})
		return
	}
	ord := &pdb.ScheduledOrder{
		UserID:     uid,
		Exchange:   strings.ToLower(req.Exchange),
		Testnet:    req.Testnet,
		Symbol:     strings.ToUpper(req.Symbol),
		Side:       strings.ToUpper(req.Side),
		OrderType:  strings.ToUpper(req.OrderType),
		Quantity:   strings.TrimSpace(req.Quantity),
		Price:      strings.TrimSpace(req.Price),
		Leverage:   req.Leverage,
		ReduceOnly: req.ReduceOnly,

		// === 保存 Bracket 参数 ===
		BracketEnabled: req.BracketEnabled,
		TPPercent:      req.TPPercent,
		SLPercent:      req.SLPercent,
		TPPrice:        strings.TrimSpace(req.TPPrice),
		SLPrice:        strings.TrimSpace(req.SLPrice),
		WorkingType:    strings.ToUpper(strings.TrimSpace(req.WorkingType)),

		TriggerTime: tt.UTC(),
		Status:      "pending",
	}
	if err := s.db.Create(ord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": ord.ID})
}
func (s *Server) ListScheduledOrders(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	var rows []pdb.ScheduledOrder
	if err := s.db.Where("user_id = ?", uid).Order("trigger_time desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (s *Server) CancelScheduledOrder(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	id := c.Param("id")
	// 仅允许取消自己且未执行的
	res := s.db.Model(&pdb.ScheduledOrder{}).
		Where("id = ? AND user_id = ? AND status IN ('pending','processing')", id, uid).
		Update("status", "canceled")
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": res.RowsAffected})
}
