package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

//func atofDef(s string, def float64) float64 {
//	if s == "" {
//		return def
//	}
//	f, err := strconv.ParseFloat(s, 64)
//	if err != nil {
//		return def
//	}
//	return f
//}

func isAll(s string) bool {
	return strings.TrimSpace(s) == "" || strings.EqualFold(s, "all")
}

// GET /flows/daily_by_chain?entity=all&chain=all&start=2025-08-06&end=2025-09-28&coin=USDT
// 支持 entity=all / chain=all（或留空）表示不筛选该条件
func (s *Server) GetDailyFlowsByChain(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	chain := strings.TrimSpace(c.Query("chain"))
	coin := strings.TrimSpace(c.Query("coin")) // 可选

	// 解析日期（UTC 零点）
	startStr := strings.TrimSpace(c.Query("start"))
	endStr := strings.TrimSpace(c.Query("end"))

	var start, end time.Time
	var err error
	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start"})
			return
		}
	}
	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end"})
			return
		}
	}

	// 默认近30天
	if start.IsZero() && end.IsZero() {
		end = time.Now().UTC().Truncate(24 * time.Hour)
		start = end.AddDate(0, 0, -30)
	} else {
		if start.IsZero() {
			start = end.AddDate(0, 0, -30)
		}
		if end.IsZero() {
			end = time.Now().UTC().Truncate(24 * time.Hour)
		}
	}
	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)
	endExclusive := end.Add(24 * time.Hour)

	// 查询
	var events []pdb.TransferEvent
	q := s.db.Where("occurred_at >= ? AND occurred_at < ?", start, endExclusive)

	if !isAll(entity) {
		q = q.Where("LOWER(entity) = ?", strings.ToLower(entity))
	}
	if !isAll(chain) {
		q = q.Where("LOWER(chain) = ?", strings.ToLower(chain))
	}
	if coin != "" && strings.ToLower(coin) != "all" {
		q = q.Where("LOWER(coin) = ?", strings.ToLower(coin))
	}

	if err := q.Order("occurred_at asc, id asc").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 有事件的日子先聚合
	type agg struct{ In, Out float64 }
	raw := map[string]*agg{}
	for _, ev := range events {
		day := ev.OccurredAt.UTC().Format("2006-01-02")
		a := raw[day]
		if a == nil {
			a = &agg{}
			raw[day] = a
		}
		amt := atofDef(ev.Amount, 0)
		switch strings.ToLower(ev.Direction) {
		case "in":
			a.In += amt
		case "out":
			a.Out += amt
		}
	}

	// 补齐区间内每天
	type Row struct {
		Day string  `json:"day"`
		In  float64 `json:"in"`
		Out float64 `json:"out"`
		Net float64 `json:"net"`
	}
	var rows []Row
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		ds := d.Format("2006-01-02")
		if a, ok := raw[ds]; ok {
			rows = append(rows, Row{Day: ds, In: a.In, Out: a.Out, Net: a.In - a.Out})
		} else {
			rows = append(rows, Row{Day: ds, In: 0, Out: 0, Net: 0})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Day < rows[j].Day })

	c.JSON(http.StatusOK, gin.H{
		"entity": entity,
		"chain":  chain,
		"coin":   coin,
		"start":  start.Format("2006-01-02"),
		"end":    end.Format("2006-01-02"),
		"data":   rows,
	})
}
