package server

import (
	pdb "analysis/internal/db"
	"fmt"
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
		s.JSONBindError(c, err)
		return
	}
	if req.Exchange == "" || req.Symbol == "" || req.Side == "" || req.OrderType == "" || req.Quantity == "" || req.TriggerTime == "" {
		s.ValidationError(c, "", "缺少必填字段：exchange, symbol, side, order_type, quantity, trigger_time")
		return
	}
	tt, err := time.Parse(time.RFC3339, req.TriggerTime)
	if err != nil {
		s.ValidationError(c, "trigger_time", "触发时间格式错误，应为 RFC3339 格式")
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
	if err := s.db.CreateScheduledOrder(ord); err != nil {
		s.DatabaseError(c, "创建定时订单", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": ord.ID})
}
// GET /orders/schedule?page=1&page_size=50
func (s *Server) ListScheduledOrders(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	
	// 分页参数
	pagination := ParsePaginationParams(
		c.Query("page"),
		c.Query("page_size"),
		50,  // 默认每页数量
		200, // 最大每页数量
	)
	
	// 使用接口方法查询
	orders, total, err := s.db.ListScheduledOrders(uid, pagination)
	if err != nil {
		s.DatabaseError(c, "查询定时订单列表", err)
		return
	}
	
	// 计算总页数
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}
	
	c.JSON(http.StatusOK, gin.H{
		"items":       orders,
		"total":       total,
		"page":        pagination.Page,
		"page_size":   pagination.PageSize,
		"total_pages": totalPages,
	})
}

func (s *Server) CancelScheduledOrder(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	idStr := c.Param("id")
	
	// 解析 ID
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		s.ValidationError(c, "id", "无效的订单ID")
		return
	}
	
	// 获取订单
	order, err := s.db.GetScheduledOrderByID(id)
	if err != nil {
		s.NotFound(c, "订单不存在")
		return
	}
	
	// 检查权限
	if order.UserID != uid {
		s.Forbidden(c, "无权操作此订单")
		return
	}
	
	// 检查状态
	if order.Status != "pending" && order.Status != "processing" {
		s.ValidationError(c, "status", "只能取消待执行或处理中的订单")
		return
	}
	
	// 更新状态
	order.Status = "canceled"
	if err := s.db.UpdateScheduledOrder(order); err != nil {
		s.DatabaseError(c, "取消定时订单", err)
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"updated": 1})
}
