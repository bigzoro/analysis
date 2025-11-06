package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	pdb "analysis/internal/db"
)

type Server struct {
	db *gorm.DB
}

func New(gdb *gorm.DB) *Server { return &Server{db: gdb} }

// GET /entities
func (s *Server) ListEntities(c *gin.Context) {
	var ents []string
	if err := s.db.Model(&pdb.PortfolioSnapshot{}).Distinct("entity").Order("entity").Pluck("entity", &ents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entities": ents})
}

// GET /runs?entity=
func (s *Server) ListRuns(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	q := s.db.Model(&pdb.PortfolioSnapshot{})
	if entity != "" {
		q = q.Where("entity = ?", entity)
	}
	var snaps []pdb.PortfolioSnapshot
	if err := q.Order("created_at desc").Limit(200).Find(&snaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type runItem struct {
		RunID    string    `json:"run_id"`
		Entity   string    `json:"entity"`
		AsOf     time.Time `json:"as_of"`
		Created  time.Time `json:"created_at"`
		TotalUSD string    `json:"total_usd"`
	}
	out := make([]runItem, 0, len(snaps))
	for _, s2 := range snaps {
		out = append(out, runItem{
			RunID:    s2.RunID,
			Entity:   s2.Entity,
			AsOf:     s2.AsOf,
			Created:  s2.CreatedAt,
			TotalUSD: s2.TotalUSD,
		})
	}
	c.JSON(http.StatusOK, gin.H{"runs": out})
}

// —— helper —— //
func (s *Server) latestRunID(entity string) (string, *pdb.PortfolioSnapshot, error) {
	var snap pdb.PortfolioSnapshot
	q := s.db.Where("entity = ?", entity).Order("created_at desc").Limit(1)
	if err := q.First(&snap).Error; err != nil {
		return "", nil, err
	}
	return snap.RunID, &snap, nil
}

// GET /portfolio/latest?entity=binance
func (s *Server) GetLatestPortfolio(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing entity"})
		return
	}
	runID, snap, err := s.latestRunID(entity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no snapshot for entity"})
		return
	}
	var hs []pdb.Holding
	if err := s.db.Where("run_id = ? AND entity = ?", runID, entity).
		Order("chain asc, symbol asc").Find(&hs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type HoldingDTO struct {
		Chain    string  `json:"chain"`
		Symbol   string  `json:"symbol"`
		Decimals int     `json:"decimals"`
		Amount   string  `json:"amount"`
		ValueUSD float64 `json:"value_usd"`
	}
	resp := struct {
		Entity   string       `json:"entity"`
		RunID    string       `json:"run_id"`
		AsOf     time.Time    `json:"as_of"`
		TotalUSD float64      `json:"total_usd"`
		Holdings []HoldingDTO `json:"holdings"`
	}{
		Entity: entity, RunID: runID, AsOf: snap.AsOf,
		TotalUSD: atofDef(snap.TotalUSD, 0),
	}
	for _, h := range hs {
		resp.Holdings = append(resp.Holdings, HoldingDTO{
			Chain: h.Chain, Symbol: h.Symbol, Decimals: h.Decimals,
			Amount: h.Amount, ValueUSD: atofDef(h.ValueUSD, 0),
		})
	}
	c.JSON(http.StatusOK, resp)
}

func atofDef(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

// GET /flows/daily?entity=binance&coin=BTC,ETH&latest=true&start=2025-08-01&end=2025-08-31
func (s *Server) GetDailyFlows(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing entity"})
		return
	}
	latest := c.DefaultQuery("latest", "true") != "false"
	coinsStr := strings.TrimSpace(c.Query("coin"))
	var coins []string
	if coinsStr != "" {
		for _, x := range strings.Split(coinsStr, ",") {
			x = strings.ToUpper(strings.TrimSpace(x))
			if x != "" {
				coins = append(coins, x)
			}
		}
	}
	start := strings.TrimSpace(c.Query("start"))
	end := strings.TrimSpace(c.Query("end"))

	q := s.db.Model(&pdb.DailyFlow{}).Where("entity = ?", entity)
	if latest {
		runID, _, err := s.latestRunID(entity)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no snapshot for entity"})
			return
		}
		q = q.Where("run_id = ?", runID)
	}
	if len(coins) > 0 {
		q = q.Where("coin IN ?", coins)
	}
	if start != "" {
		q = q.Where("day >= ?", start)
	}
	if end != "" {
		q = q.Where("day <= ?", end)
	}

	var rows []pdb.DailyFlow
	if err := q.Order("coin asc, day asc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type Row struct {
		Day string  `json:"day"`
		In  float64 `json:"in"`
		Out float64 `json:"out"`
		Net float64 `json:"net"`
	}
	out := map[string][]Row{} // coin -> rows
	for _, r := range rows {
		out[r.Coin] = append(out[r.Coin], Row{
			Day: r.Day, In: atofDef(r.In, 0), Out: atofDef(r.Out, 0), Net: atofDef(r.Net, 0),
		})
	}
	// 排序
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Day < out[k][j].Day })
	}
	c.JSON(http.StatusOK, gin.H{
		"entity": entity,
		"latest": latest,
		"coins":  coins,
		"data":   out,
	})
}

// GET /flows/weekly?entity=binance&coin=BTC,ETH&latest=true
func (s *Server) GetWeeklyFlows(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing entity"})
		return
	}
	latest := c.DefaultQuery("latest", "true") != "false"
	coinsStr := strings.TrimSpace(c.Query("coin"))
	var coins []string
	if coinsStr != "" {
		for _, x := range strings.Split(coinsStr, ",") {
			x = strings.ToUpper(strings.TrimSpace(x))
			if x != "" {
				coins = append(coins, x)
			}
		}
	}

	q := s.db.Model(&pdb.WeeklyFlow{}).Where("entity = ?", entity)
	if latest {
		runID, _, err := s.latestRunID(entity)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no snapshot for entity"})
			return
		}
		q = q.Where("run_id = ?", runID)
	}
	if len(coins) > 0 {
		q = q.Where("coin IN ?", coins)
	}

	var rows []pdb.WeeklyFlow
	if err := q.Order("coin asc, week asc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type Row struct {
		Week string  `json:"week"`
		In   float64 `json:"in"`
		Out  float64 `json:"out"`
		Net  float64 `json:"net"`
	}
	out := map[string][]Row{}
	for _, r := range rows {
		out[r.Coin] = append(out[r.Coin], Row{
			Week: r.Week, In: atofDef(r.In, 0), Out: atofDef(r.Out, 0), Net: atofDef(r.Net, 0),
		})
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Week < out[k][j].Week })
	}
	c.JSON(http.StatusOK, gin.H{
		"entity": entity,
		"latest": latest,
		"coins":  coins,
		"data":   out,
	})
}
