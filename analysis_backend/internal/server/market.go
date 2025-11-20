package server

import (
	"analysis/internal/db"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 给采集进程写的入口：POST /ingest/binance/market
func (s *Server) IngestBinanceMarket(c *gin.Context) {
	var body struct {
		Kind      string `json:"kind"`
		Bucket    string `json:"bucket"`
		FetchedAt string `json:"fetched_at"`
		Items     []struct {
			Symbol             string   `json:"symbol"`
			LastPrice          string   `json:"last_price"`
			Volume             string   `json:"volume"`
			PriceChangePercent float64  `json:"price_change_percent"`
			MarketCapUSD       *float64 `json:"market_cap_usd"`
			FDVUSD             *float64 `json:"fdv_usd"`
			CirculatingSupply  *float64 `json:"circulating_supply"`
			TotalSupply        *float64 `json:"total_supply"`
		} `json:"items"`
	}
	if err := c.BindJSON(&body); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if body.Kind == "" {
		body.Kind = "spot"
	}

	bucket, err := time.Parse(time.RFC3339, body.Bucket)
	if err != nil {
		s.BadRequest(c, "时间桶格式错误", err)
		return
	}

	fetchedAt := time.Now().UTC()
	if body.FetchedAt != "" {
		if t, e := time.Parse(time.RFC3339, body.FetchedAt); e == nil {
			fetchedAt = t
		}
	}

	// 存库统一用 UTC + 2h 对齐
	bucket = bucket.UTC().Truncate(2 * time.Hour)

	rows := make([]db.BinanceMarketTop, 0, len(body.Items))
	for i, it := range body.Items {
		rows = append(rows, db.BinanceMarketTop{
			Symbol:            it.Symbol,
			LastPrice:         it.LastPrice,
			Volume:            it.Volume,
			PctChange:         it.PriceChangePercent,
			Rank:              i + 1,
			MarketCapUSD:      it.MarketCapUSD,
			FDVUSD:            it.FDVUSD,
			CirculatingSupply: it.CirculatingSupply,
			TotalSupply:       it.TotalSupply,
		})
	}

	if _, err := db.SaveBinanceMarket(s.db.DB(), body.Kind, bucket, fetchedAt, rows); err != nil {
		s.DatabaseError(c, "保存市场数据", err)
		return
	}

	// 失效市场数据缓存，使新数据立即生效
	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

var coincap *coinCapCache

// binanceMarketParams 市场查询参数
type binanceMarketParams struct {
	Kind        string
	IntervalMin int
	Location    *time.Location
	Date        string
	Slot        string
}

// parseBinanceMarketParams 解析市场查询参数
func parseBinanceMarketParams(c *gin.Context) (*binanceMarketParams, error) {
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	if kind != "spot" && kind != "futures" {
		kind = "futures"
	}

	intervalMin := 120
	if v := c.Query("interval"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalMin = n
		}
	}

	tzName := c.Query("tz")
	if tzName == "" {
		tzName = "Asia/Taipei"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.FixedZone("CST-8", 8*3600)
	}

	return &binanceMarketParams{
		Kind:        kind,
		IntervalMin: intervalMin,
		Location:    loc,
		Date:        strings.TrimSpace(c.Query("date")),
		Slot:        strings.TrimSpace(c.Query("slot")),
	}, nil
}

// calculateTimeRange 计算时间范围
func calculateTimeRange(params *binanceMarketParams) (time.Time, time.Time, error) {
	if params.Date == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("date is required")
	}

	dayStartLocal, err := time.ParseInLocation("2006-01-02", params.Date, params.Location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("日期格式错误，应为 YYYY-MM-DD: %w", err)
	}

	var startLocal, endLocal time.Time
	if params.Slot != "" {
		slot, err := strconv.Atoi(params.Slot)
		if err != nil || slot < 0 || slot > (24*60/params.IntervalMin-1) {
			return time.Time{}, time.Time{}, fmt.Errorf("时间段编号无效")
		}
		startLocal = dayStartLocal.Add(time.Duration(slot) * time.Minute * time.Duration(params.IntervalMin))
		endLocal = startLocal.Add(time.Duration(params.IntervalMin) * time.Minute)
	} else {
		startLocal = dayStartLocal
		endLocal = dayStartLocal.Add(24 * time.Hour)
	}

	return startLocal.UTC(), endLocal.UTC(), nil
}

// filterAndFormatMarketData 过滤黑名单并格式化市场数据
func (s *Server) filterAndFormatMarketData(snaps []db.BinanceMarketSnapshot, tops map[uint][]db.BinanceMarketTop, kind string, ctx context.Context) ([]gin.H, error) {
	// 获取黑名单（现货和期货都支持）- 使用缓存
	blacklistMap, err := s.getCachedBlacklistMap(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get cached blacklist (kind=%s), falling back to direct query: %v", kind, err)
		// 缓存失败时降级到直接查询，但不影响主流程
		blacklistMap = make(map[string]bool)
		if blacklist, e := s.db.GetBinanceBlacklist(kind); e == nil {
			for _, symbol := range blacklist {
				blacklistMap[strings.ToUpper(symbol)] = true
			}
		}
	}

	// 优化：预估输出切片大小
	out := make([]gin.H, 0, len(snaps))
	for _, snap := range snaps {
		list := tops[snap.ID]
		// 过滤黑名单（Symbol 已是大写，直接使用）
		// 优化：预估过滤后的切片大小（假设最多保留10个）
		estimatedSize := len(list)
		if estimatedSize > 10 {
			estimatedSize = 10
		}
		filtered := make([]db.BinanceMarketTop, 0, estimatedSize)
		for _, it := range list {
			if !blacklistMap[it.Symbol] {
				filtered = append(filtered, it)
				// 优化：如果已经达到10个，提前退出循环
				if len(filtered) >= 10 {
					break
				}
			}
		}
		// 取前10个（如果超过10个）
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}
		// 优化：预估 items 切片大小
		items := make([]gin.H, 0, len(filtered))
		for _, it := range filtered {
			items = append(items, gin.H{
				"symbol":             it.Symbol,
				"last_price":         it.LastPrice,
				"volume":             it.Volume,
				"pct_change":         it.PctChange,
				"rank":               it.Rank,
				"market_cap_usd":     it.MarketCapUSD,
				"fdv_usd":            it.FDVUSD,
				"circulating_supply": it.CirculatingSupply,
				"total_supply":       it.TotalSupply,
			})
		}
		out = append(out, gin.H{
			"bucket":     snap.Bucket,    // UTC
			"fetched_at": snap.FetchedAt, // UTC
			"kind":       snap.Kind,
			"items":      items,
		})
	}
	return out, nil
}

// minInt 返回两个整数中的较小值（优化辅助函数）
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) GetBinanceMarket(c *gin.Context) {
	params, err := parseBinanceMarketParams(c)
	if err != nil {
		s.BadRequest(c, "参数解析失败", err)
		return
	}

	// 如果没传 date，默认今天（本地时区）
	if params.Date == "" {
		day := time.Now().In(params.Location).Format("2006-01-02")
		q := c.Request.URL.Query()
		q.Set("date", day)
		c.Request.URL.RawQuery = q.Encode()
		// 重新解析参数
		params, err = parseBinanceMarketParams(c)
		if err != nil {
			s.BadRequest(c, "参数解析失败", err)
			return
		}
	}

	// 计算时间范围
	startUTC, endUTC, err := calculateTimeRange(params)
	if err != nil {
		s.ValidationError(c, "date", err.Error())
		return
	}

	// 查询市场数据
	snaps, tops, err := db.ListBinanceMarket(s.db.DB(), params.Kind, startUTC, endUTC)
	if err != nil {
		s.DatabaseError(c, "查询市场数据", err)
		return
	}

	// 过滤和格式化数据
	out, err := s.filterAndFormatMarketData(snaps, tops, params.Kind, c.Request.Context())
	if err != nil {
		s.InternalServerError(c, "处理市场数据失败", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"kind":     params.Kind,
		"interval": params.IntervalMin,
		"data":     out,
	})
}

// ===== 黑名单管理 API =====

// GET /market/binance/blacklist?kind=spot|futures - 获取黑名单
func (s *Server) ListBinanceBlacklist(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	items, err := s.db.ListBinanceBlacklist(kind)
	if err != nil {
		s.DatabaseError(c, "查询黑名单", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// POST /market/binance/blacklist - 添加黑名单
func (s *Server) AddBinanceBlacklist(c *gin.Context) {
	var body struct {
		Kind   string `json:"kind"` // spot / futures
		Symbol string `json:"symbol"`
	}
	if err := c.BindJSON(&body); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if body.Kind == "" {
		s.ValidationError(c, "kind", "类型不能为空，必须为 spot 或 futures")
		return
	}
	if body.Symbol == "" {
		s.ValidationError(c, "symbol", "币种符号不能为空")
		return
	}
	if err := s.db.AddBinanceBlacklist(body.Kind, body.Symbol); err != nil {
		s.DatabaseError(c, "添加黑名单", err)
		return
	}
	// 失效市场数据缓存和黑名单缓存，使黑名单变更立即生效
	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}
	if err := s.InvalidateBlacklistCache(c.Request.Context(), body.Kind); err != nil {
		log.Printf("[WARN] Failed to invalidate blacklist cache (kind=%s): %v", body.Kind, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /market/binance/blacklist/:kind/:symbol - 删除黑名单
func (s *Server) DeleteBinanceBlacklist(c *gin.Context) {
	kind := strings.TrimSpace(c.Param("kind"))
	symbol := strings.TrimSpace(c.Param("symbol"))
	if symbol == "" {
		s.ValidationError(c, "symbol", "币种符号不能为空")
		return
	}
	if err := s.db.DeleteBinanceBlacklist(kind, symbol); err != nil {
		s.DatabaseError(c, "删除黑名单", err)
		return
	}
	// 失效市场数据缓存和黑名单缓存，使黑名单变更立即生效
	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}
	if err := s.InvalidateBlacklistCache(c.Request.Context(), kind); err != nil {
		log.Printf("[WARN] Failed to invalidate blacklist cache (kind=%s): %v", kind, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
