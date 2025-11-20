package server

import (
	"context"
	"encoding/json"
	"log"
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
	db      Database // 使用接口而非具体实现
	Mailer  Mailer
	XBearer string
	cache   pdb.CacheInterface // 缓存接口
}

// New 创建 Server 实例（使用接口）
func New(db Database) *Server {
	return &Server{db: db}
}

// NewWithGorm 从 GORM DB 创建 Server 实例（向后兼容）
func NewWithGorm(gdb *gorm.DB) *Server {
	return &Server{db: NewGormDatabase(gdb)}
}

// SetCache 设置缓存
func (s *Server) SetCache(cache pdb.CacheInterface) {
	s.cache = cache
}

// GET /entities
func (s *Server) ListEntities(c *gin.Context) {
	ents, err := s.db.ListEntities()
	if err != nil {
		s.DatabaseError(c, "查询实体列表", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entities": ents})
}

// GET /runs?entity=&page=1&page_size=50
func (s *Server) ListRuns(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))

	// 分页参数
	pagination := ParsePaginationParams(
		c.Query("page"),
		c.Query("page_size"),
		50,  // 默认每页数量
		200, // 最大每页数量
	)

	// 搜索和过滤参数
	keyword := strings.TrimSpace(c.Query("keyword"))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	// 使用接口方法查询
	params := PortfolioSnapshotQueryParams{
		Entity:           entity,
		Keyword:          keyword,
		StartDate:        startDate,
		EndDate:          endDate,
		PaginationParams: pagination,
	}

	snaps, total, err := s.db.ListPortfolioSnapshots(params)
	if err != nil {
		s.DatabaseError(c, "查询运行记录", err)
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

	// 计算总页数
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       out,
		"total":       total,
		"page":        pagination.Page,
		"page_size":   pagination.PageSize,
		"total_pages": totalPages,
		// 兼容字段
		"runs": out,
	})
}

// —— helper —— //
func (s *Server) latestRunID(entity string) (string, *pdb.PortfolioSnapshot, error) {
	snap, err := s.db.GetLatestPortfolioSnapshot(entity)
	if err != nil {
		return "", nil, err
	}
	return snap.RunID, snap, nil
}

// GET /portfolio/latest?entity=binance
func (s *Server) GetLatestPortfolio(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}

	// 尝试使用缓存
	if s.cache != nil {
		// 先获取最新的 runID
		runID, _, err := s.latestRunID(entity)
		if err != nil {
			s.NotFound(c, "未找到该实体的快照数据")
			return
		}

		// 尝试从缓存获取
		key := BuildCacheKey("cache:portfolio:latest", entity, runID)
		cached, err := s.cache.Get(c.Request.Context(), key)
		if err == nil && len(cached) > 0 {
			var cachedData struct {
				Snapshot pdb.PortfolioSnapshot `json:"snapshot"`
				Holdings []pdb.Holding         `json:"holdings"`
			}
			if err := json.Unmarshal(cached, &cachedData); err == nil {
				// 返回缓存数据
				holdings := make([]HoldingDTO, 0, len(cachedData.Holdings))
				for _, h := range cachedData.Holdings {
					holdings = append(holdings, HoldingDTO{
						Chain: h.Chain, Symbol: h.Symbol, Decimals: h.Decimals,
						Amount: h.Amount, ValueUSD: atofDef(h.ValueUSD, 0),
					})
				}
				c.JSON(http.StatusOK, gin.H{
					"entity":    entity,
					"run_id":    runID,
					"as_of":     cachedData.Snapshot.AsOf,
					"total_usd": atofDef(cachedData.Snapshot.TotalUSD, 0),
					"holdings":  holdings,
				})
				return
			}
		}
	}

	// 缓存未命中，查询数据库
	runID, snap, err := s.latestRunID(entity)
	if err != nil {
		s.NotFound(c, "未找到该实体的快照数据")
		return
	}

	// 使用接口方法查询持仓
	startTime := time.Now()
	hs, err := s.db.GetHoldingsByRunID(runID, entity)
	if err != nil {
		s.DatabaseError(c, "查询持仓数据", err)
		return
	}
	duration := time.Since(startTime)

	// 记录慢查询
	if duration > 1*time.Second {
		pdb.LogSlowQuery("GetLatestPortfolio", duration, int64(len(hs)))
	}

	// 优化：使用协程池异步写入缓存
	if s.cache != nil {
		cacheData := struct {
			Snapshot pdb.PortfolioSnapshot `json:"snapshot"`
			Holdings []pdb.Holding         `json:"holdings"`
		}{
			Snapshot: *snap,
			Holdings: hs,
		}
		data, err := json.Marshal(cacheData)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal cache data for portfolio latest (entity=%s, runID=%s): %v", entity, runID, err)
		} else {
			key := BuildCacheKey("cache:portfolio:latest", entity, runID)
			cacheKey := key
			cacheDataBytes := make([]byte, len(data))
			copy(cacheDataBytes, data)

			if globalCachePool != nil {
				globalCachePool.Submit(func() {
					if err := s.cache.Set(context.Background(), cacheKey, cacheDataBytes, 5*time.Minute); err != nil {
						log.Printf("[ERROR] Failed to set cache for portfolio latest (entity=%s, runID=%s, key=%s): %v", entity, runID, cacheKey, err)
					} else {
						log.Printf("[INFO] Successfully cached portfolio latest (entity=%s, runID=%s)", entity, runID)
					}
				})
			} else {
				go func() {
					if err := s.cache.Set(context.Background(), cacheKey, cacheDataBytes, 5*time.Minute); err != nil {
						log.Printf("[ERROR] Failed to set cache for portfolio latest (entity=%s, runID=%s, key=%s): %v", entity, runID, cacheKey, err)
					} else {
						log.Printf("[INFO] Successfully cached portfolio latest (entity=%s, runID=%s)", entity, runID)
					}
				}()
			}
		}
	}
	resp := struct {
		Entity   string       `json:"entity"`
		RunID    string       `json:"run_id"`
		AsOf     time.Time    `json:"as_of"`
		TotalUSD float64      `json:"total_usd"`
		Holdings []HoldingDTO `json:"holdings"`
		Meta     gin.H        `json:"_meta,omitempty"` // 开发环境显示性能指标
	}{
		Entity: entity, RunID: runID, AsOf: snap.AsOf,
		TotalUSD: atofDef(snap.TotalUSD, 0),
	}
	holdings := make([]HoldingDTO, 0, len(hs))
	for _, h := range hs {
		holdings = append(holdings, HoldingDTO{
			Chain: h.Chain, Symbol: h.Symbol, Decimals: h.Decimals,
			Amount: h.Amount, ValueUSD: atofDef(h.ValueUSD, 0),
		})
	}
	resp.Holdings = holdings

	// 开发环境添加性能指标
	if gin.Mode() == gin.DebugMode {
		resp.Meta = gin.H{
			"query_time_ms":  duration.Milliseconds(),
			"holdings_count": len(holdings),
		}
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

// GetDailyFlows 获取日度资金流（已优化：使用查询优化器，添加性能监控）
func (s *Server) GetDailyFlows(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}
	latest := c.DefaultQuery("latest", "true") != "false"
	coins := parseCoinsParam(strings.TrimSpace(c.Query("coin")))
	start := strings.TrimSpace(c.Query("start"))
	end := strings.TrimSpace(c.Query("end"))

	// 获取 runID（如果需要）
	var runID string
	if latest {
		var err error
		runID, _, err = s.latestRunID(entity)
		if err != nil {
			s.NotFound(c, "未找到该实体的快照数据")
			return
		}
	}

	// 使用接口方法查询
	params := FlowQueryParams{
		Entity: entity,
		Coins:  coins,
		Latest: latest,
		RunID:  runID,
		Start:  start,
		End:    end,
	}

	startTime := time.Now()
	rows, err := s.db.GetDailyFlows(params)
	if err != nil {
		s.DatabaseError(c, "查询日度资金流", err)
		return
	}
	duration := time.Since(startTime)

	// 记录慢查询
	if duration > 1*time.Second {
		pdb.LogSlowQuery("GetDailyFlows", duration, int64(len(rows)))
	}

	// 转换数据
	out := map[string][]flowRow{} // coin -> rows
	for _, r := range rows {
		out[r.Coin] = append(out[r.Coin], flowRow{
			Day: r.Day,
			In:  atofDef(r.In, 0),
			Out: atofDef(r.Out, 0),
			Net: atofDef(r.Net, 0),
		})
	}

	// 排序
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Day < out[k][j].Day })
	}

	response := gin.H{
		"entity": entity,
		"latest": latest,
		"coins":  coins,
		"data":   out,
	}
	// 开发环境添加性能指标
	if gin.Mode() == gin.DebugMode {
		response["_meta"] = gin.H{
			"query_time_ms": duration.Milliseconds(),
			"rows_count":    len(rows),
		}
	}
	c.JSON(http.StatusOK, response)
}

// GetTransferStats 获取转账统计（使用聚合查询）
func (s *Server) GetTransferStats(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	chain := strings.TrimSpace(c.Query("chain"))
	coin := strings.TrimSpace(c.Query("coin"))

	// 解析时间范围
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
	} else {
		start = time.Now().AddDate(0, 0, -7) // 默认最近7天
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			s.ValidationError(c, "end", "结束日期格式错误，应为 YYYY-MM-DD")
			return
		}
		end = end.Add(24 * time.Hour) // 包含结束日
	} else {
		end = time.Now()
	}

	params := TransferStatsParams{
		Entity: entity,
		Chain:  chain,
		Coin:   coin,
		Start:  start,
		End:    end,
	}

	stats, err := s.db.GetTransferStats(params)
	if err != nil {
		s.DatabaseError(c, "查询转账统计", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// BatchGetEntities 批量获取实体列表（使用 IN 查询）
func (s *Server) BatchGetEntities(c *gin.Context) {
	entitiesStr := strings.TrimSpace(c.Query("entities"))
	if entitiesStr == "" {
		s.ValidationError(c, "entities", "实体列表不能为空")
		return
	}

	entities := strings.Split(entitiesStr, ",")
	for i := range entities {
		entities[i] = strings.TrimSpace(entities[i])
	}

	// 优化：使用一次查询替代循环查询，提高性能
	result := make(map[string][]pdb.PortfolioSnapshot)

	// 使用 IN 查询一次性获取所有实体的数据
	var allSnaps []pdb.PortfolioSnapshot
	if err := s.db.DB().Model(&pdb.PortfolioSnapshot{}).
		Where("entity IN ?", entities).
		Order("entity ASC, created_at DESC").
		Find(&allSnaps).Error; err != nil {
		s.DatabaseError(c, "批量查询实体", err)
		return
	}

	// 按实体分组，每个实体最多保留100条
	for _, snap := range allSnaps {
		if len(result[snap.Entity]) < 100 {
			result[snap.Entity] = append(result[snap.Entity], snap)
		}
	}

	c.JSON(http.StatusOK, gin.H{"entities": result})
}

// GET /flows/weekly?entity=binance&coin=BTC,ETH&latest=true
// GetWeeklyFlows 获取周度资金流（已优化：添加性能监控）
func (s *Server) GetWeeklyFlows(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}
	latest := c.DefaultQuery("latest", "true") != "false"
	coins := parseCoinsParam(strings.TrimSpace(c.Query("coin")))

	// 获取 runID（如果需要）
	var runID string
	if latest {
		var err error
		runID, _, err = s.latestRunID(entity)
		if err != nil {
			s.NotFound(c, "未找到该实体的快照数据")
			return
		}
	}

	// 使用接口方法查询
	params := FlowQueryParams{
		Entity: entity,
		Coins:  coins,
		Latest: latest,
		RunID:  runID,
	}

	startTime := time.Now()
	rows, err := s.db.GetWeeklyFlows(params)
	if err != nil {
		s.DatabaseError(c, "查询周度资金流", err)
		return
	}
	duration := time.Since(startTime)

	// 记录慢查询
	if duration > 1*time.Second {
		pdb.LogSlowQuery("GetWeeklyFlows", duration, int64(len(rows)))
	}

	// 转换数据
	out := map[string][]weeklyFlowRow{}
	for _, r := range rows {
		out[r.Coin] = append(out[r.Coin], weeklyFlowRow{
			Week: r.Week,
			In:   atofDef(r.In, 0),
			Out:  atofDef(r.Out, 0),
			Net:  atofDef(r.Net, 0),
		})
	}

	// 排序
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Week < out[k][j].Week })
	}

	response := gin.H{
		"entity": entity,
		"latest": latest,
		"coins":  coins,
		"data":   out,
	}
	// 开发环境添加性能指标
	if gin.Mode() == gin.DebugMode {
		response["_meta"] = gin.H{
			"query_time_ms": duration.Milliseconds(),
			"rows_count":    len(rows),
		}
	}
	c.JSON(http.StatusOK, response)
}
