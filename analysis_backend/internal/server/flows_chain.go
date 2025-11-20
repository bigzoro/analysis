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
			s.ValidationError(c, "start", "开始日期格式错误，应为 YYYY-MM-DD")
			return
		}
	}
	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			s.ValidationError(c, "end", "结束日期格式错误，应为 YYYY-MM-DD")
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
	q := s.db.DB().Where("occurred_at >= ? AND occurred_at < ?", start, endExclusive)

	// 优化：如果数据存储时已统一大小写，直接查询，避免使用函数导致索引失效
	if !isAll(entity) {
		q = q.Where("entity = ?", strings.ToLower(entity))
	}
	if !isAll(chain) {
		q = q.Where("chain = ?", strings.ToLower(chain))
	}
	if coin != "" && strings.ToLower(coin) != "all" {
		q = q.Where("coin = ?", strings.ToUpper(coin))
	}

	if err := q.Order("occurred_at asc, id asc").Find(&events).Error; err != nil {
		s.DatabaseError(c, "查询转账事件", err)
		return
	}

	// 优化：有事件的日子先聚合（预估 map 大小）
	type agg struct{ In, Out float64 }
	// 预估 map 大小：假设最多有 events 数量的不同日期
	raw := make(map[string]*agg, len(events))
	for _, ev := range events {
		day := ev.OccurredAt.UTC().Format("2006-01-02")
		a := raw[day]
		if a == nil {
			a = &agg{}
			raw[day] = a
		}
		amt := atofDef(ev.Amount, 0)
		// 优化：避免重复调用 strings.ToLower
		direction := strings.ToLower(ev.Direction)
		switch direction {
		case "in":
			a.In += amt
		case "out":
			a.Out += amt
		}
	}

	// 优化：补齐区间内每天（预估切片大小）
	type Row struct {
		Day string  `json:"day"`
		In  float64 `json:"in"`
		Out float64 `json:"out"`
		Net float64 `json:"net"`
	}
	// 计算日期范围，预估切片大小
	days := int(end.Sub(start).Hours()/24) + 1
	rows := make([]Row, 0, days)
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
