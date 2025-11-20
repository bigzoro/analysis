package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// CreateSimulatedTrade 创建模拟交易
// POST /recommendations/simulation/trade
func (s *Server) CreateSimulatedTrade(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req struct {
		RecommendationID *uint  `json:"recommendation_id"`
		Symbol           string `json:"symbol"`
		BaseSymbol       string `json:"base_symbol"`
		Kind             string `json:"kind"`
		Quantity         string `json:"quantity"`
		Price            string `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Symbol == "" || req.Quantity == "" || req.Price == "" {
		s.ValidationError(c, "", "symbol、quantity和price不能为空")
		return
	}

	if req.Kind == "" {
		req.Kind = "spot"
	}

	// 计算总价值
	quantity, _ := strconv.ParseFloat(req.Quantity, 64)
	price, _ := strconv.ParseFloat(req.Price, 64)
	totalValue := quantity * price

	trade := &pdb.SimulatedTrade{
		UserID:          uid,
		RecommendationID: req.RecommendationID,
		Symbol:          strings.ToUpper(req.Symbol),
		BaseSymbol:      strings.ToUpper(req.BaseSymbol),
		Kind:            strings.ToLower(req.Kind),
		Side:            "BUY",
		Quantity:        req.Quantity,
		Price:           req.Price,
		TotalValue:      fmt.Sprintf("%.8f", totalValue),
		IsOpen:          true,
		CurrentPrice:    &req.Price, // 初始价格等于买入价格
	}

	if err := pdb.CreateSimulatedTrade(s.db.DB(), trade); err != nil {
		s.DatabaseError(c, "创建模拟交易", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": trade.ID})
}

// GetSimulatedTrades 获取模拟交易列表
// GET /recommendations/simulation/trades?is_open=true
func (s *Server) GetSimulatedTrades(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var isOpen *bool
	if isOpenStr := c.Query("is_open"); isOpenStr != "" {
		if isOpenStr == "true" {
			val := true
			isOpen = &val
		} else if isOpenStr == "false" {
			val := false
			isOpen = &val
		}
	}

	trades, err := pdb.GetSimulatedTrades(s.db.DB(), uid, isOpen)
	if err != nil {
		s.DatabaseError(c, "查询模拟交易", err)
		return
	}

	// 计算当前盈亏（需要实时价格，这里简化处理）
	// 实际应该调用价格API更新current_price和unrealized_pnl

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
		"total":  len(trades),
	})
}

// CloseSimulatedTrade 平仓模拟交易
// POST /recommendations/simulation/trades/:id/close
func (s *Server) CloseSimulatedTrade(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的ID")
		return
	}

	var req struct {
		Price string `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Price == "" {
		s.ValidationError(c, "price", "卖出价格不能为空")
		return
	}

	trade, err := pdb.GetSimulatedTradeByID(s.db.DB(), uint(id), uid)
	if err != nil {
		s.NotFound(c, "交易不存在")
		return
	}

	if !trade.IsOpen {
		s.ValidationError(c, "status", "交易已平仓")
		return
	}

	// 计算盈亏
	buyPrice, _ := strconv.ParseFloat(trade.Price, 64)
	sellPrice, _ := strconv.ParseFloat(req.Price, 64)
	quantity, _ := strconv.ParseFloat(trade.Quantity, 64)

	realizedPnl := (sellPrice - buyPrice) * quantity
	realizedPnlPercent := ((sellPrice - buyPrice) / buyPrice) * 100

	now := time.Now().UTC()
	trade.IsOpen = false
	trade.SoldAt = &now
	sellPriceStr := req.Price
	trade.CurrentPrice = &sellPriceStr
	realizedPnlStr := fmt.Sprintf("%.8f", realizedPnl)
	trade.RealizedPnl = &realizedPnlStr
	realizedPnlPercentVal := realizedPnlPercent
	trade.RealizedPnlPercent = &realizedPnlPercentVal

	if err := pdb.UpdateSimulatedTrade(s.db.DB(), trade); err != nil {
		s.DatabaseError(c, "平仓交易", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": 1,
		"realized_pnl": realizedPnl,
		"realized_pnl_percent": realizedPnlPercent,
	})
}

// UpdateSimulatedTradePrice 更新模拟交易当前价格（用于实时更新盈亏）
// POST /recommendations/simulation/trades/:id/update-price
func (s *Server) UpdateSimulatedTradePrice(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的ID")
		return
	}

	var req struct {
		CurrentPrice string `json:"current_price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	trade, err := pdb.GetSimulatedTradeByID(s.db.DB(), uint(id), uid)
	if err != nil {
		s.NotFound(c, "交易不存在")
		return
	}

	if !trade.IsOpen {
		s.ValidationError(c, "status", "交易已平仓，无法更新价格")
		return
	}

	// 更新当前价格和未实现盈亏
	buyPrice, _ := strconv.ParseFloat(trade.Price, 64)
	currentPrice, _ := strconv.ParseFloat(req.CurrentPrice, 64)
	quantity, _ := strconv.ParseFloat(trade.Quantity, 64)

	unrealizedPnl := (currentPrice - buyPrice) * quantity
	unrealizedPnlPercent := ((currentPrice - buyPrice) / buyPrice) * 100

	trade.CurrentPrice = &req.CurrentPrice
	unrealizedPnlStr := fmt.Sprintf("%.8f", unrealizedPnl)
	trade.UnrealizedPnl = &unrealizedPnlStr
	unrealizedPnlPercentVal := unrealizedPnlPercent
	trade.UnrealizedPnlPercent = &unrealizedPnlPercentVal

	if err := pdb.UpdateSimulatedTrade(s.db.DB(), trade); err != nil {
		s.DatabaseError(c, "更新价格", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated": 1,
		"unrealized_pnl": unrealizedPnl,
		"unrealized_pnl_percent": unrealizedPnlPercent,
	})
}
