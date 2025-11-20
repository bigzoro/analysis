package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// GetBacktestRecords 获取回测记录
// GET /recommendations/backtest?limit=50
func (s *Server) GetBacktestRecords(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	records, err := pdb.GetBacktestRecords(s.db.DB(), limit)
	if err != nil {
		s.DatabaseError(c, "查询回测记录", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   len(records),
	})
}

// GetBacktestStats 获取回测统计
// GET /recommendations/backtest/stats
func (s *Server) GetBacktestStats(c *gin.Context) {
	stats, err := pdb.GetBacktestStats(s.db.DB())
	if err != nil {
		s.DatabaseError(c, "查询回测统计", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateBacktestFromRecommendation 从推荐创建回测记录
// POST /recommendations/backtest
func (s *Server) CreateBacktestFromRecommendation(c *gin.Context) {
	var req struct {
		RecommendationID uint   `json:"recommendation_id"`
		Symbol           string `json:"symbol"`
		BaseSymbol       string `json:"base_symbol"`
		Price            string `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Symbol == "" || req.Price == "" {
		s.ValidationError(c, "", "symbol和price不能为空")
		return
	}

	rec := &pdb.BacktestRecord{
		RecommendationID: req.RecommendationID,
		Symbol:           strings.ToUpper(req.Symbol),
		BaseSymbol:       strings.ToUpper(req.BaseSymbol),
		RecommendedAt:    time.Now().UTC(),
		RecommendedPrice: req.Price,
		Status:           "pending",
	}

	if err := pdb.CreateBacktestRecord(s.db.DB(), rec); err != nil {
		s.DatabaseError(c, "创建回测记录", err)
		return
	}

	// 异步更新回测结果（定期任务会处理）
	go s.updateBacktestResult(context.Background(), rec.ID)

	c.JSON(http.StatusOK, gin.H{"id": rec.ID})
}

// updateBacktestResult 更新回测结果（异步）
func (s *Server) updateBacktestResult(ctx context.Context, recordID uint) {
	// 这里应该查询当前价格并计算收益率
	// 简化处理：实际应该调用价格API获取历史价格
	// 暂时留空，由定时任务处理
}

// UpdateBacktestRecord 更新回测记录（手动触发或定时任务调用）
// POST /recommendations/backtest/:id/update
func (s *Server) UpdateBacktestRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的ID")
		return
	}

	var req struct {
		PriceAfter24h  *string  `json:"price_after_24h"`
		PriceAfter7d   *string  `json:"price_after_7d"`
		PriceAfter30d  *string  `json:"price_after_30d"`
		Performance24h *float64 `json:"performance_24h"`
		Performance7d  *float64 `json:"performance_7d"`
		Performance30d *float64 `json:"performance_30d"`
		Status         string   `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	var rec pdb.BacktestRecord
	if err := s.db.DB().First(&rec, id).Error; err != nil {
		s.NotFound(c, "回测记录不存在")
		return
	}

	if req.PriceAfter24h != nil {
		rec.PriceAfter24h = req.PriceAfter24h
	}
	if req.PriceAfter7d != nil {
		rec.PriceAfter7d = req.PriceAfter7d
	}
	if req.PriceAfter30d != nil {
		rec.PriceAfter30d = req.PriceAfter30d
	}
	if req.Performance24h != nil {
		rec.Performance24h = req.Performance24h
	}
	if req.Performance7d != nil {
		rec.Performance7d = req.Performance7d
	}
	if req.Performance30d != nil {
		rec.Performance30d = req.Performance30d
	}
	if req.Status != "" {
		rec.Status = req.Status
	}

	if err := pdb.UpdateBacktestRecord(s.db.DB(), &rec); err != nil {
		s.DatabaseError(c, "更新回测记录", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"updated": 1})
}
