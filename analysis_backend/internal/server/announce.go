package server

import (
	pdb "analysis/internal/db"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

const (
	// 唯一数据源
	sourceCoincarp = "coincarp"
)

type binanceIngestItem struct {
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Tags       []string  `json:"tags"`
	Summary    string    `json:"summary"`
	ReleaseMS  int64     `json:"release_ms"`
	ReleaseISO string    `json:"release_iso"`
	CreatedAt  time.Time `json:"created_at"`
}

type upbitIngestItem struct {
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Tags       []string  `json:"tags"`
	Summary    string    `json:"summary"`
	ReleaseMS  int64     `json:"release_ms"`
	ReleaseISO string    `json:"release_iso"`
	CreatedAt  time.Time `json:"created_at"`
}

type binanceIngestRequest struct {
	Items []binanceIngestItem `json:"items"`
}
type upbitIngestRequest struct {
	Items []upbitIngestItem `json:"items"`
}

func (s *Server) normalizeAnnouncement(
	code, title, url string,
	tags []string, summary string,
	releaseMS int64, releaseISO string, createdAt time.Time,
	newsCode string,
) pdb.Announcement {
	// 1) 时间处理
	var ts time.Time
	switch {
	case releaseMS > 0:
		ts = time.UnixMilli(releaseMS).UTC()
	case strings.TrimSpace(releaseISO) != "":
		if t, err := time.Parse(time.RFC3339, releaseISO); err == nil {
			ts = t.UTC()
		}
	case !createdAt.IsZero():
		ts = createdAt.UTC()
	default:
		ts = time.Now().UTC()
	}

	// 2) 分类
	cat := classifyCategory(title, summary, tags)

	// 3) 原始 JSON
	raw := map[string]any{
		"code":        code,
		"title":       title,
		"url":         url,
		"tags":        tags,
		"summary":     summary,
		"release_ms":  releaseMS,
		"release_iso": releaseISO,
		"created_at":  createdAt,
	}
	// 优化：添加错误处理
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		// JSON 序列化失败，使用空 JSON
		rawJSON = []byte("{}")
	}

	// 3) 组装公告对象
	return pdb.Announcement{
		Source:      sourceCoincarp,
		ExternalID:  code,
		NewsCode:    newsCode,
		Title:       title,
		Summary:     summary,
		URL:         url,
		Category:    cat,
		Tags:        datatypes.NewJSONType(tags),
		ReleaseTime: ts,
		Raw:         datatypes.JSON(rawJSON),
		// 扩展字段默认值
		IsEvent:   false,
		Sentiment: "",
		HeatScore: 0,
		Exchange:  "",
		Verified:  false,
	}
}

// ---- Ingest ----

func (s *Server) IngestBinanceAnnouncements(c *gin.Context) {
	var req binanceIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": 0})
		return
	}
	rows := make([]pdb.Announcement, 0, len(req.Items))
	for _, it := range req.Items {
		ann := s.normalizeAnnouncement(it.Code, it.Title, it.URL, it.Tags, it.Summary, it.ReleaseMS, it.ReleaseISO, it.CreatedAt, "")
		ann.Verified = true // 官方源验证标记
		rows = append(rows, ann)
	}
	out, err := pdb.SaveAnnouncements(s.db.DB(), rows)
	if err != nil {
		s.DatabaseError(c, "保存公告", err)
		return
	}
	// 清除公告相关缓存
	if s.cache != nil {
		_ = s.InvalidateAnnouncementsCache(c.Request.Context())
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "saved": len(out)})
}

func (s *Server) IngestUpbitAnnouncements(c *gin.Context) {
	var req upbitIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": 0})
		return
	}
	rows := make([]pdb.Announcement, 0, len(req.Items))
	for _, it := range req.Items {
		ann := s.normalizeAnnouncement(it.Code, it.Title, it.URL, it.Tags, it.Summary, it.ReleaseMS, it.ReleaseISO, it.CreatedAt, "")
		ann.Verified = true // 官方源验证标记
		rows = append(rows, ann)
	}
	out, err := pdb.SaveAnnouncements(s.db.DB(), rows)
	if err != nil {
		s.DatabaseError(c, "保存公告", err)
		return
	}
	// 清除公告相关缓存
	if s.cache != nil {
		_ = s.InvalidateAnnouncementsCache(c.Request.Context())
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "saved": len(out)})
}

// 通用公告 ingest（仅支持 coincarp 数据源）
type genericIngestItem struct {
	ExternalID string   `json:"external_id"`
	NewsCode   string   `json:"news_code"` // CoinCarp newscode
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	URL        string   `json:"url"`
	Tags       []string `json:"tags"`
	ReleaseMS  int64    `json:"release_ms"`
	Exchange   string   `json:"exchange"`
	IsEvent    bool     `json:"is_event"`
	Sentiment  string   `json:"sentiment"`
	HeatScore  int      `json:"heat_score"`
	Verified   bool     `json:"verified"`
}

type genericIngestRequest struct {
	Items []genericIngestItem `json:"items"`
}

func (s *Server) IngestGenericAnnouncements(c *gin.Context) {
	var req genericIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": 0})
		return
	}

	rows := make([]pdb.Announcement, 0, len(req.Items))
	for _, it := range req.Items {
		ann := s.normalizeAnnouncement(it.ExternalID, it.Title, it.URL, it.Tags, it.Summary, it.ReleaseMS, "", time.Time{}, it.NewsCode)
		// 设置扩展字段
		ann.Exchange = it.Exchange
		ann.IsEvent = it.IsEvent
		ann.Sentiment = it.Sentiment
		ann.HeatScore = it.HeatScore
		ann.Verified = it.Verified
		rows = append(rows, ann)
	}

	err := pdb.MergeAnnouncements(s.db.DB(), rows)
	if err != nil {
		s.DatabaseError(c, "合并公告", err)
		return
	}
	// 清除公告相关缓存
	if s.cache != nil {
		_ = s.InvalidateAnnouncementsCache(c.Request.Context())
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "saved": len(rows)})
}

// ---- Query ----

// ListAnnouncements 查询公告列表（仅 coincarp 数据源，支持分页）
// GET /announcements/recent?categories=newcoin,finance&q=listing&page=1&page_size=50&is_event=true&verified=true&sentiment=positive&exchange=binance
// 兼容旧格式：limit/offset 会自动转换为 page/page_size
func (s *Server) ListAnnouncements(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	categories := parseCSV(c.Query("categories"))

	// 分页参数：优先使用 page/page_size，兼容 limit/offset
	// 分页参数（兼容旧格式 limit 和 offset）
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	// 兼容旧格式 limit
	if pageSizeStr == "" {
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			pageSizeStr = v
		}
	}

	// 兼容旧格式 offset
	if pageStr == "" {
		if v := strings.TrimSpace(c.Query("offset")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				// 需要先知道 pageSize 才能计算 page
				pageSize := 10
				if ps := strings.TrimSpace(pageSizeStr); ps != "" {
					if psn, err := strconv.Atoi(ps); err == nil && psn > 0 && psn <= 200 {
						pageSize = psn
					}
				}
				pageStr = strconv.Itoa((n / pageSize) + 1)
			}
		}
	}

	pagination := ParsePaginationParams(
		pageStr,
		pageSizeStr,
		10,  // 默认每页数量
		200, // 最大每页数量
	)
	page := pagination.Page
	pageSize := pagination.PageSize
	offset := pagination.Offset

	// 筛选参数
	isEventStr := strings.TrimSpace(c.Query("is_event"))
	verifiedStr := strings.TrimSpace(c.Query("verified"))
	sentiment := strings.TrimSpace(c.Query("sentiment"))
	exchange := strings.TrimSpace(c.Query("exchange"))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	// 构建基础查询（用于计数和查询数据）
	baseQuery := s.db.DB().Model(&pdb.Announcement{}).
		Where("source = ?", sourceCoincarp)

	// 应用筛选条件
	if len(categories) > 0 {
		baseQuery = baseQuery.Where("category IN ?", categories)
	}
	if q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		baseQuery = baseQuery.Where("LOWER(title) LIKE ? OR LOWER(summary) LIKE ?", pat, pat)
	}
	// 处理 is_event 筛选（支持 true/false）
	if isEventStr == "true" || isEventStr == "1" {
		baseQuery = baseQuery.Where("is_event = ?", true)
	} else if isEventStr == "false" || isEventStr == "0" {
		baseQuery = baseQuery.Where("is_event = ?", false)
	}
	// 处理 verified 筛选（支持 true/false）
	if verifiedStr == "true" || verifiedStr == "1" {
		baseQuery = baseQuery.Where("verified = ?", true)
	} else if verifiedStr == "false" || verifiedStr == "0" {
		baseQuery = baseQuery.Where("verified = ?", false)
	}
	if sentiment != "" {
		baseQuery = baseQuery.Where("sentiment = ?", strings.ToLower(sentiment))
	}
	if exchange != "" {
		baseQuery = baseQuery.Where("exchange = ?", strings.ToLower(exchange))
	}
	// 日期范围筛选（优化：使用范围查询替代 DATE() 函数，可以利用索引）
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			baseQuery = baseQuery.Where("release_time >= ?", t.UTC())
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// 结束日期包含当天，所以加一天并减1秒
			endTime := t.UTC().Add(24 * time.Hour).Add(-time.Second)
			baseQuery = baseQuery.Where("release_time <= ?", endTime)
		}
	}

	// 优化：COUNT 查询优化（可以考虑缓存）
	var total int64
	countQuery := baseQuery
	if err := countQuery.Count(&total).Error; err != nil {
		s.DatabaseError(c, "统计公告总数", err)
		return
	}

	// 查询数据（分页）
	// 优化：使用独立的查询对象，避免影响 COUNT 查询
	var rows []pdb.Announcement
	dataQuery := baseQuery.Order("release_time DESC").Limit(pageSize).Offset(offset)
	if err := dataQuery.Find(&rows).Error; err != nil {
		s.DatabaseError(c, "查询公告列表", err)
		return
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       rows,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		// 兼容字段
		"count":  len(rows),
		"limit":  pageSize,
		"offset": offset,
	})
}

// GetLatestAnnouncementTime 获取最新的公告时间（用于增量同步，仅 coincarp 数据源）
// GET /announcements/latest-time
func (s *Server) GetLatestAnnouncementTime(c *gin.Context) {
	latestTime, err := s.db.GetLatestAnnouncementTime()

	if err != nil {
		// 如果没有数据，返回 0（表示从当前时间往前推 24 小时）
		c.JSON(http.StatusOK, gin.H{
			"latest_time": 0,
			"has_data":    false,
		})
		return
	}

	if latestTime == nil {
		c.JSON(http.StatusOK, gin.H{
			"latest_time": 0,
			"has_data":    false,
		})
		return
	}

	// 返回 Unix 时间戳（秒）
	c.JSON(http.StatusOK, gin.H{
		"latest_time":  latestTime.Unix(),
		"has_data":     true,
		"release_time": latestTime.UTC().Format(time.RFC3339),
	})
}

func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		t := strings.ToLower(strings.TrimSpace(p))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// classifyCategory 根据标题/摘要/标签分类 -> newcoin | finance | other
func classifyCategory(title, summary string, tags []string) string {
	txt := strings.ToLower(title + " " + summary)
	for _, t := range tags {
		txt += " " + strings.ToLower(t)
	}
	// 新币关键词
	newWords := []string{
		"new listing", "list", "listing", "上线", "上币", "新币", "新增交易",
		"상장", // 韩语：上币
	}
	for _, w := range newWords {
		if strings.Contains(txt, w) {
			return "newcoin"
		}
	}
	// 理财关键词
	finWords := []string{
		"earn", "saving", "savings", "staking", "锁仓", "理财", "质押", "活期", "定期", "收益",
	}
	for _, w := range finWords {
		if strings.Contains(txt, w) {
			return "finance"
		}
	}
	return "other"
}
