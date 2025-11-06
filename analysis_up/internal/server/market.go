// internal/server/market.go
package server

import (
	"analysis/internal/db"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 给采集进程写的入口：POST /ingest/binance/market
func IngestBinanceMarket(gdb *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Kind      string `json:"kind"`
			Bucket    string `json:"bucket"`
			FetchedAt string `json:"fetched_at"`
			Items     []struct {
				Symbol             string  `json:"symbol"`
				LastPrice          string  `json:"last_price"`
				Volume             string  `json:"volume"`
				PriceChangePercent float64 `json:"price_change_percent"`
			} `json:"items"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
			return
		}
		if body.Kind == "" {
			body.Kind = "spot"
		}

		bucket, err := time.Parse(time.RFC3339, body.Bucket)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket: " + err.Error()})
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
				Symbol:    it.Symbol,
				LastPrice: it.LastPrice,
				Volume:    it.Volume,
				PctChange: it.PriceChangePercent,
				Rank:      i + 1,
			})
		}

		if _, err := db.SaveBinanceMarket(gdb, body.Kind, bucket, fetchedAt, rows); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// 给前端查的：GET /market/binance/top
// 重点：按前端传入的时区来“还原”这一天的第几段，再转回 UTC 去查
func (s *Server) GetBinanceMarket(c *gin.Context) {
	kind := c.DefaultQuery("kind", "spot")
	// 前端显示的是 2 小时一段，这里就保留这个字段让前端知道
	intervalMin, _ := strconv.Atoi(c.DefaultQuery("interval", "120"))

	// 新增：前端告诉后端“我是在这个时区看的”
	// 不传就默认按台湾时间来对齐
	tz := c.DefaultQuery("tz", "Asia/Taipei")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// 万一前端传错时区，就退回 UTC
		loc = time.UTC
	}

	dateStr := c.Query("date") // 2025-10-31
	slotStr := c.Query("slot") // 0..11

	// ========== 情况 1：前端点了“第几段” ==========
	if slotStr != "" {
		slot, err := strconv.Atoi(slotStr)
		if err != nil || slot < 0 || slot > 11 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slot, must be 0..11"})
			return
		}

		// 1) 先得到“这个时区下面的那一天 00:00:00”
		var dayLocal time.Time
		if dateStr != "" {
			// 前端指定了日期
			d, err := time.ParseInLocation("2006-01-02", dateStr, loc)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
				return
			}
			dayLocal = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		} else {
			// 没指定就用“这个时区的今天”
			nowLocal := time.Now().In(loc)
			dayLocal = time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
		}

		// 2) 这个时区下面的第 slot 个 2 小时时段
		// slot=0 → 00:00-02:00
		// slot=1 → 02:00-04:00
		// ...
		startLocal := dayLocal.Add(time.Duration(slot) * 2 * time.Hour)
		endLocal := startLocal.Add(2 * time.Hour)

		// 3) 存库是 UTC，要转换成 UTC 再去查
		startUTC := startLocal.UTC()
		endUTC := endLocal.UTC()

		snaps, tops, err := db.ListBinanceMarket(s.db, kind, startUTC, endUTC)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		out := make([]gin.H, 0, len(snaps))
		for _, snap := range snaps {
			list := tops[snap.ID]
			if len(list) > 10 {
				list = list[:10]
			}
			items := make([]gin.H, 0, len(list))
			for _, it := range list {
				items = append(items, gin.H{
					"symbol":     it.Symbol,
					"last_price": it.LastPrice,
					"volume":     it.Volume,
					"pct_change": it.PctChange,
					"rank":       it.Rank,
				})
			}
			out = append(out, gin.H{
				// 这里还是返回 UTC 时间，前端自己 toLocaleString 就会变回本地了
				"bucket":     snap.Bucket,
				"fetched_at": snap.FetchedAt,
				"kind":       snap.Kind,
				"items":      items,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"kind":     kind,
			"interval": intervalMin,
			"data":     out,
		})
		return
	}

	// ========== 情况 2：前端没选“第几段”，就是按天查 ==========
	// 这个也要按前端时区来理解那一天
	startStr := c.Query("start")
	endStr := c.Query("end")

	var startUTC, endUTC time.Time

	if startStr == "" && endStr == "" && dateStr != "" {
		// ?date=2025-10-31
		d, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
			return
		}
		dayLocal := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		startUTC = dayLocal.UTC()
		endUTC = dayLocal.Add(24 * time.Hour).UTC()
	} else if startStr == "" && endStr == "" {
		// 默认查“今天”（按前端时区）
		nowLocal := time.Now().In(loc)
		dayLocal := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)
		startUTC = dayLocal.UTC()
		endUTC = dayLocal.Add(24 * time.Hour).UTC()
	} else {
		if startStr != "" {
			d, err := time.ParseInLocation("2006-01-02", startStr, loc)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start"})
				return
			}
			startUTC = d.UTC()
		}
		if endStr != "" {
			d, err := time.ParseInLocation("2006-01-02", endStr, loc)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end"})
				return
			}
			endUTC = d.Add(24 * time.Hour).UTC()
		}
	}

	snaps, tops, err := db.ListBinanceMarket(s.db, kind, startUTC, endUTC)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]gin.H, 0, len(snaps))
	for _, snap := range snaps {
		list := tops[snap.ID]
		if len(list) > 10 {
			list = list[:10]
		}
		items := make([]gin.H, 0, len(list))
		for _, it := range list {
			items = append(items, gin.H{
				"symbol":     it.Symbol,
				"last_price": it.LastPrice,
				"volume":     it.Volume,
				"pct_change": it.PctChange,
				"rank":       it.Rank,
			})
		}
		out = append(out, gin.H{
			"bucket":     snap.Bucket,
			"fetched_at": snap.FetchedAt,
			"kind":       snap.Kind,
			"items":      items,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"kind":     kind,
		"interval": intervalMin,
		"data":     out,
	})
}
