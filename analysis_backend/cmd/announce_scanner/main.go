// cmd/announce_scanner/main.go
package main

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"analysis/internal/netutil"

	"gorm.io/gorm"
)

// =============================
//         入库结构体
// =============================

// 发给 /ingest/binance/announcements 的条目
type binanceIngestItem struct {
	Source    string `json:"source"`     // 固定 "binance"
	CatalogID int    `json:"catalog_id"` // 48/49/93...
	Code      string `json:"code"`       // 唯一 code
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	URL       string `json:"url"`
	ReleaseMS int64  `json:"release_ms"` // 毫秒（UTC）
}
type binanceIngestReq struct {
	Items []binanceIngestItem `json:"items"`
}

// 发给 /ingest/upbit/announcements 的条目
type upbitIngestItem struct {
	Source    string `json:"source"` // 固定 "upbit"
	ID        int64  `json:"id"`     // Upbit 公告 ID
	Title     string `json:"title"`
	URL       string `json:"url"`        // 可能为相对路径；这里会补全
	ReleaseMS int64  `json:"release_ms"` // 毫秒（UTC）
}
type upbitIngestReq struct {
	Items []upbitIngestItem `json:"items"`
}

// =============================
//        Binance 抓取
// =============================

// Binance CMS 列表（catalog 模式）
type binanceCMSResp struct {
	Data struct {
		Articles []struct {
			CatalogID  int    `json:"catalogId"`
			Code       string `json:"code"`
			Title      string `json:"title"`
			Summary    string `json:"summary"`
			ReleaseTS  int64  `json:"releaseDate"` // 有些目录给 releaseDate（有时是秒，有时是毫秒）
			Link       string `json:"link"`        // 可能为空或相对路径
			PublishTs2 int64  `json:"publishTime"` // 另一个可能的时间字段（同样存在秒/毫秒差异）
		} `json:"articles"`
	} `json:"data"`
}

func fetchBinance(ctx context.Context, client *http.Client, catalogs []int, pageSize int) ([]binanceIngestItem, error) {
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	items := make([]binanceIngestItem, 0, pageSize*len(catalogs))

	for _, cat := range catalogs {
		u := fmt.Sprintf(
			"https://www.binance.com/bapi/composite/v1/public/cms/article/catalog/list/query?catalogId=%d&pageNo=1&pageSize=%d",
			cat, pageSize,
		)

		var resp binanceCMSResp
		if err := httpGetJSON(ctx, client, u, &resp); err != nil {
			// Binance API 可能偶尔失败，记录错误但继续处理其他分类
			log.Printf("[binance] fetch catalog %d err: %v", cat, err)
			continue
		}

		for _, a := range resp.Data.Articles {
			// ---- 时间兜底（releaseDate / publishTime，且可能是“秒”也可能是“毫秒”）----
			ms := a.ReleaseTS
			if ms <= 0 {
				ms = a.PublishTs2
			}
			// 如果是“秒”，转成“毫秒”
			if ms > 0 && ms < 1e12 {
				ms *= 1000
			}

			// 仍拿不到就用当前时间兜底（避免前端空白）
			if ms <= 0 {
				ms = time.Now().UTC().UnixMilli()
			}

			// 兜底：仍然拿不到就给当前时间，避免前端空白
			if ms <= 0 {
				ms = time.Now().UTC().UnixMilli()
			}

			// ---- 链接兜底（空/相对路径 -> 绝对路径；完全缺失时用 code 拼详情页）----
			link := strings.TrimSpace(a.Link)
			if link == "" {
				// 你也可以改成 zh-CN
				link = fmt.Sprintf("https://www.binance.com/en/support/announcement/%s", a.Code)
			} else if strings.HasPrefix(link, "/") {
				link = "https://www.binance.com" + link
			}

			items = append(items, binanceIngestItem{
				Source:    "binance",
				CatalogID: a.CatalogID,
				Code:      a.Code,
				Title:     a.Title,
				Summary:   a.Summary,
				URL:       link,
				ReleaseMS: ms,
			})
		}
		time.Sleep(200 * time.Millisecond) // 轻微限速
	}
	return items, nil
}

type binanceDetailResp struct {
	Data struct {
		ReleaseTS  int64  `json:"releaseDate"` // 同样存在秒/毫秒差异
		PublishTs2 int64  `json:"publishTime"`
		Title      string `json:"title"`
		Code       string `json:"code"`
	} `json:"data"`
}

// =============================
//         Upbit 抓取
// =============================

// 不同地区与版本差异较大：多域名/多路径尝试 + 对 per_page 做降级重试
func fetchUpbit(ctx context.Context, client *http.Client, limit int) ([]upbitIngestItem, error) {
	// 先把用户传入的 limit 收敛一下（接口常见允许 20；再不行降级到 10）
	// 这里的 perPageCandidates 会在每个候选 URL 上轮流尝试
	perPageCandidates := []int{limit}
	if perPageCandidates[0] <= 0 || perPageCandidates[0] > 50 {
		perPageCandidates[0] = 50
	}
	// 把更稳的 20、10 放在前面，避免一上来就 400
	perPageCandidates = []int{min(perPageCandidates[0], 20), 10}

	// 优先新接口 + 备用域名，不再尝试旧的 /notices（你那边已证实 404）
	baseCandidates := []string{
		"https://api-manager.upbit.com/api/v1/announcements?os=web&page=1&per_page=%d&category=all",
	}

	var lastErr error
	for _, tpl := range baseCandidates {
		for _, pp := range perPageCandidates {
			u := fmt.Sprintf(tpl, pp)

			var raw map[string]any
			err := httpGetJSON(ctx, client, u, &raw)
			if err != nil {
				// 如果是 400 且与 per_page 相关，继续尝试下一档更小的 per_page
				if he, ok := err.(httpStatusError); ok && he.Code == 400 && strings.Contains(strings.ToLower(he.Msg), "per_page") {
					lastErr = err
					continue
				}
				// 429/403 等直接切下一个候选 URL
				lastErr = err
				break
			}

			items := extractUpbitItems(raw)
			if len(items) == 0 {
				lastErr = fmt.Errorf("upbit: empty list from %s", u)
				continue
			}
			return items, nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("upbit: all candidates failed")
	}
	return nil, lastErr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 从 Upbit 原始 JSON 里尽量抽取通用字段（支持 data.notices / data.fixed_notices）
func extractUpbitItems(raw map[string]any) []upbitIngestItem {
	var arr []any

	// 1) 新结构：data.notices [+ data.fixed_notices]
	if v, ok := raw["data"]; ok {
		if dm, ok := v.(map[string]any); ok {
			if l, ok := dm["notices"].([]any); ok && len(l) > 0 {
				arr = append(arr, l...)
			}
			// 置底附加置顶公告（固定公告）
			if l2, ok := dm["fixed_notices"].([]any); ok && len(l2) > 0 {
				arr = append(arr, l2...)
			}
			// 兼容历史结构
			if len(arr) == 0 {
				if l, ok := dm["list"].([]any); ok {
					arr = l
				} else if l2, ok2 := dm["announcements"].([]any); ok2 {
					arr = l2
				}
			}
		}
	}
	// 2) 顶层也可能直接给
	if len(arr) == 0 {
		if l, ok := raw["announcements"].([]any); ok {
			arr = l
		} else if l2, ok2 := raw["list"].([]any); ok2 {
			arr = l2
		}
	}
	if len(arr) == 0 {
		return nil
	}

	out := make([]upbitIngestItem, 0, len(arr))
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}

		// id
		var id int64
		switch v := m["id"].(type) {
		case float64:
			id = int64(v)
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				id = n
			}
		}
		if id == 0 {
			continue
		}

		// 标题
		title := firstString(m, "title", "subject")
		if title == "" {
			continue
		}

		// 时间：优先 listed_at，其次 first_listed_at；都是 RFC3339（含 +08:00）
		tsStr := firstString(m, "listed_at", "first_listed_at", "created_at", "createdAt", "posted_at", "regDate", "date")
		t := parseUpbitTime(tsStr)
		if t.IsZero() {
			//t = time.Now()
			t = time.Now().UTC()
		}

		// 该接口不直接给 URL，拼一个可访问的公告详情
		urlStr := fmt.Sprintf("https://upbit.com/service_center/notice?id=%d", id)

		out = append(out, upbitIngestItem{
			Source:    "upbit",
			ID:        id,
			Title:     title,
			URL:       urlStr,
			ReleaseMS: t.UTC().UnixMilli(),
		})
	}
	return out
}

// 解析 Upbit 返回的时间字符串（优先 RFC3339，兜底常见格式）
func parseUpbitTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	// RFC3339（例：2025-11-06T11:17:01+08:00）
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	// 兜底：无时区的常见格式（当作 KST/Asia/Seoul 也可；这里直接按本地或 UTC 解析都不致命）
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		//return t
		return t.UTC()
	}
	return time.Time{}
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch vv := v.(type) {
			case string:
				if strings.TrimSpace(vv) != "" {
					return vv
				}
			case float64:
				return strconv.FormatInt(int64(vv), 10)
			}
		}
	}
	return ""
}

// =============================
//        HTTP & 工具函数
// =============================

type httpStatusError struct {
	Code int
	Msg  string
}

func (e httpStatusError) Error() string { return e.Msg }

func httpGetJSON(ctx context.Context, client *http.Client, u string, out any) error {
	return httpGetJSONWithRetry(ctx, client, u, out, 3)
}

func httpGetJSONWithRetry(ctx context.Context, client *http.Client, u string, out any, maxRetries int) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：100ms, 200ms, 400ms
			delay := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}

		// 根据 URL 设置不同的请求头
		if strings.Contains(u, "binance.com") {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
			req.Header.Set("Referer", "https://www.binance.com/")
			req.Header.Set("Origin", "https://www.binance.com")
		} else if strings.Contains(u, "okx.com") {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
			req.Header.Set("Referer", "https://www.okx.com/")
			req.Header.Set("Origin", "https://www.okx.com")
		} else if strings.Contains(u, "bybit.com") {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
			req.Header.Set("Referer", "https://www.bybit.com/")
			req.Header.Set("Origin", "https://www.bybit.com")
		} else {
			// 默认请求头
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "application/json, text/plain, */*")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			// EOF 错误通常是网络问题，可以重试
			if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "connection") {
				continue
			}
			return err
		}

		// 读取响应体
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode/100 != 2 {
			lastErr = httpStatusError{
				Code: resp.StatusCode,
				Msg:  fmt.Sprintf("GET %s => %d: %s", u, resp.StatusCode, string(body[:min(len(body), 768)])),
			}
			// 5xx 错误可以重试，4xx 错误（除了 403/429）不重试
			if resp.StatusCode >= 500 || resp.StatusCode == 429 {
				continue
			}
			return lastErr
		}

		// 解析 JSON
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if err := json.Unmarshal(body, out); err != nil {
			lastErr = err
			// JSON 解析错误通常不应该重试，但如果是空响应可能是网络问题
			if len(body) == 0 {
				continue
			}
			return err
		}

		return nil
	}
	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

// 构造带代理/IPv4/DNS 的 http.Client
func newHTTPClient(proxyURL string, forceIPv4 bool) *http.Client {
	// 代理
	var proxy func(*http.Request) (*url.URL, error)
	if proxyURL != "" {
		target, err := url.Parse(proxyURL)
		if err == nil {
			proxy = http.ProxyURL(target)
		} else {
			proxy = http.ProxyFromEnvironment
		}
	} else {
		proxy = http.ProxyFromEnvironment
	}

	// 强制 IPv4 拨号
	dialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
		d := &net.Dialer{Timeout: 15 * time.Second}
		if forceIPv4 {
			return d.DialContext(ctx, "tcp4", address)
		}
		return d.DialContext(ctx, "tcp", address)
	}

	tr := &http.Transport{
		Proxy:               proxy,
		DialContext:         dialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        64,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 12 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}
}

// 覆盖系统 DNS（可选）
func initCustomDNS(servers []string) {
	if len(servers) == 0 {
		return
	}
	// 使用 Go 解析器 + 指定 DNS 服务器
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			// 轮询使用第一个 DNS（简单处理；需要更复杂可加随机/循环）
			target := strings.TrimSpace(servers[0])
			if !strings.Contains(target, ":") {
				target += ":53"
			}
			// 使用 UDP 够用，TCP 可改 "tcp"
			return d.DialContext(ctx, "udp", target)
		},
	}
}

// =============================
//            主流程
// =============================

func main() {
	// ---- flags ----
	apiBase := flag.String("api", "http://127.0.0.1:8010", "API base")
	interval := flag.Duration("interval", 60*time.Second*10, "poll interval")
	catalogs := flag.String("catalogs", "48,49,93", "binance catalog ids, comma separated")
	pageSize := flag.Int("page-size", 20, "binance page size (<=50)")
	upbitEnable := flag.Bool("upbit", false, "enable upbit fetching (temporarily disabled)")
	binanceEnable := flag.Bool("binance", false, "enable binance fetching (temporarily disabled)")
	upbitPageSize := flag.Int("upbit-page-size", 50, "upbit per_page (<=50)")
	cfgPath := flag.String("config", "config.yaml", "config file")

	// 多层次抓取标志位
	coincarpEnable := flag.Bool("coincarp", true, "enable coincarp fetching (layer 1)")
	// cryptopanicEnable := flag.Bool("cryptopanic", true, "enable cryptopanic fetching (layer 2)") // 已移除 CryptoPanic 功能
	// coinmarketcalEnable := flag.Bool("coinmarketcal", true, "enable coinmarketcal fetching (layer 2)") // 已移除 CoinMarketCal 功能
	okxEnable := flag.Bool("okx", false, "enable okx fetching (layer 3)")
	bybitEnable := flag.Bool("bybit", false, "enable bybit fetching (layer 3)")
	// cryptopanicKey := flag.String("cryptopanic-key", "", "cryptopanic API key (optional)") // 已移除 CryptoPanic 功能

	// 新增网络健壮性参数
	//proxyFlag := flag.String("proxy", "http://127.0.0.1:10808", "http(s) proxy, e.g. http://127.0.0.1:7890 (fallback to env HTTP_PROXY/HTTPS_PROXY)")
	dnsFlag := flag.String("dns", "", "custom DNS servers, comma separated (e.g. 8.8.8.8,1.1.1.1)")
	forceIPv4 := flag.Bool("force-ipv4", true, "force use IPv4 (tcp4)")

	flag.Parse()

	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)

	// 自定义 DNS（可选）
	if strings.TrimSpace(*dnsFlag) != "" {
		parts := strings.Split(*dnsFlag, ",")
		servers := make([]string, 0, len(parts))
		for _, s := range parts {
			s = strings.TrimSpace(s)
			if s != "" {
				servers = append(servers, s)
			}
		}
		if len(servers) > 0 {
			initCustomDNS(servers)
			log.Printf("[net] custom DNS enabled: %v", servers)
		}
	}

	// 解析 catalogs
	cats := parseCatalogs(*catalogs)
	if len(cats) == 0 {
		cats = []int{48, 49, 93}
	}

	// 统一 HTTP 客户端
	httpClient := newHTTPClient(strings.TrimSpace(cfg.Proxy.HTTP), *forceIPv4)

	// 连接数据库（用于读取最新公告时间）
	var gdb *gorm.DB
	if cfg.Database.DSN != "" {
		var err error
		gdb, err = pdb.OpenMySQL(pdb.Options{
			DSN:          cfg.Database.DSN,
			Automigrate:  false, // scanner 不需要自动迁移
			MaxOpenConns: 2,     // scanner 只需要少量连接
			MaxIdleConns: 1,
		})
		if err != nil {
			log.Printf("[ann_scanner] failed to connect to database: %v, will use API fallback", err)
		} else {
			log.Printf("[ann_scanner] database connected successfully")
			defer func() {
				if sqlDB, err := gdb.DB(); err == nil {
					sqlDB.Close()
				}
			}()
		}
	}

	log.Printf("[ann_scanner] start api=%s interval=%s catalogs=%v upbit=%v proxy=%v forceIPv4=%v dns=%v",
		*apiBase, interval.String(), cats, *upbitEnable, cfg.Proxy.HTTP != "", *forceIPv4, *dnsFlag != "")

	ctx := context.Background()
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// 去重缓存（本进程生命周期内）
	seen := make(map[string]struct{})

	// 记录上次获取的最新公告时间戳（用于增量获取）
	var lastFetchTime int64 = 0

	// 启动时从数据库直接读取最新的公告时间，避免重复同步
	if *coincarpEnable && gdb != nil {
		var ann pdb.Announcement
		err := gdb.Model(&pdb.Announcement{}).
			Where("source = ?", "coincarp").
			Order("release_time DESC").
			First(&ann).Error

		if err == nil {
			lastFetchTime = ann.ReleaseTime.Unix()
			log.Printf("[ann_scanner] initialized lastFetchTime from database: %d (%s)",
				lastFetchTime, ann.ReleaseTime.UTC().Format(time.RFC3339))
		} else {
			log.Printf("[ann_scanner] no existing data in database, will fetch from 24h ago")
		}
	} else if *coincarpEnable {
		log.Printf("[ann_scanner] database not available, will start from 24h ago")
	}

	runOnce := func() {
		added := 0

		// ===== 第一层：CoinCarp（主数据源） =====
		if *coincarpEnable {
			// 使用上次获取时间作为起始点，只获取新公告
			// 如果是第一次运行（lastFetchTime == 0），则获取最近 24 小时的公告
			issuetime := lastFetchTime
			if issuetime == 0 {
				// 第一次运行：获取最近 24 小时的公告
				issuetime = time.Now().Add(-24 * time.Hour).Unix()
			}

			if items, err := fetchCoinCarp(ctx, httpClient, issuetime, 50); err != nil {
				log.Printf("[coincarp] fetch err: %v", err)
			} else if len(items) > 0 {
				// 从数据库查询已存在的 URL（用于去重，避免重启后重复同步）
				existingURLs := make(map[string]struct{})
				if gdb != nil {
					var existing []pdb.Announcement
					urls := make([]string, 0, len(items))
					for _, it := range items {
						normalizedURL := strings.TrimRight(strings.TrimSpace(it.URL), "/")
						if normalizedURL != "" {
							urls = append(urls, normalizedURL)
						}
					}
					if len(urls) > 0 {
						// 批量查询已存在的 URL
						if err := gdb.Model(&pdb.Announcement{}).
							Where("url IN ?", urls).
							Select("url").
							Find(&existing).Error; err == nil {
							for _, e := range existing {
								normalized := strings.TrimRight(strings.TrimSpace(e.URL), "/")
								existingURLs[normalized] = struct{}{}
							}
						}
					}
				}

				// 转换为通用格式，并去重（内存 + 数据库）
				genericItems := make([]map[string]any, 0, len(items))
				for _, it := range items {
					// 标准化 URL（去除末尾斜杠和空格）
					normalizedURL := strings.TrimRight(strings.TrimSpace(it.URL), "/")
					if normalizedURL == "" {
						continue
					}

					// 内存去重（本进程生命周期内）
					key := "coincarp|" + normalizedURL
					if _, ok := seen[key]; ok {
						continue // 已处理过，跳过
					}

					// 数据库去重（避免重启后重复）
					if _, ok := existingURLs[normalizedURL]; ok {
						seen[key] = struct{}{} // 标记为已处理，避免下次重复查询
						continue               // 数据库中已存在，跳过
					}

					seen[key] = struct{}{}

					// 使用标准化后的 URL
					it.URL = normalizedURL

					genericItems = append(genericItems, map[string]any{
						"source":      it.Source,
						"external_id": it.ExternalID,
						"news_code":   it.NewsCode,
						"title":       it.Title,
						"summary":     it.Summary,
						"url":         it.URL,
						"tags":        it.Tags,
						"release_ms":  it.ReleaseMS,
						"exchange":    it.Exchange,
						"is_event":    false,
						"sentiment":   "",
						"heat_score":  0,
						"verified":    false,
					})
				}
				if len(genericItems) > 0 {
					payload := map[string]any{"items": genericItems}
					postURL := strings.TrimRight(*apiBase, "/") + "/ingest/coincarp/announcements"
					var out map[string]any
					if err := netutil.PostJSON(ctx, postURL, payload, &out); err != nil {
						log.Printf("[coincarp] ingest err: %v", err)
					} else {
						log.Printf("[coincarp] ingested: %v (count=%d, filtered=%d)", out, len(genericItems), len(items)-len(genericItems))
						added += len(genericItems)

						// 更新上次获取时间：使用本次获取的最新公告时间
						maxTime := issuetime
						for _, it := range items {
							// releaseMS 是毫秒，转换为秒
							itemTime := it.ReleaseMS / 1000
							if itemTime > maxTime {
								maxTime = itemTime
							}
						}
						if maxTime > lastFetchTime {
							lastFetchTime = maxTime
							log.Printf("[coincarp] updated last fetch time to %d (%s)", lastFetchTime, time.Unix(lastFetchTime, 0).UTC().Format(time.RFC3339))
						}
					}
				} else {
					log.Printf("[coincarp] all items already seen, skipped")
					// 即使没有新数据，也更新时间为当前时间（避免重复获取旧数据）
					currentTime := time.Now().Unix()
					if currentTime > lastFetchTime {
						lastFetchTime = currentTime
					}
				}
			}
		}

		// ===== 第二层：CryptoPanic（验证和过滤） =====
		// CryptoPanic 功能已移除
		// if *cryptopanicEnable {
		// 	if items, err := fetchCryptoPanic(ctx, httpClient, *cryptopanicKey, 30); err != nil {
		// 		log.Printf("[cryptopanic] fetch err: %v", err)
		// 	} else if len(items) > 0 {
		// 		genericItems := make([]map[string]any, 0, len(items))
		// 		for _, it := range items {
		// 			genericItems = append(genericItems, map[string]any{
		// 				"source":      it.Source,
		// 				"external_id": it.ExternalID,
		// 				"title":       it.Title,
		// 				"summary":     it.Summary,
		// 				"url":         it.URL,
		// 				"tags":        it.Tags,
		// 				"release_ms":  it.ReleaseMS,
		// 				"is_event":    it.IsEvent,
		// 				"sentiment":   it.Sentiment,
		// 				"heat_score":  it.HeatScore,
		// 				"verified":    false,
		// 			})
		// 		}
		// 		payload := map[string]any{"items": genericItems}
		// 		postURL := strings.TrimRight(*apiBase, "/") + "/ingest/cryptopanic/announcements"
		// 		var out map[string]any
		// 		if err := netutil.PostJSON(ctx, postURL, payload, &out); err != nil {
		// 			log.Printf("[cryptopanic] ingest err: %v", err)
		// 		} else {
		// 			log.Printf("[cryptopanic] ingested: %v (count=%d)", out, len(items))
		// 			added += len(items)
		// 		}
		// 	}
		// }

		// ===== 第二层：CoinMarketCal（事件验证） =====
		// CoinMarketCal 功能已移除
		// if *coinmarketcalEnable {
		// 	if items, err := fetchCoinMarketCal(ctx, httpClient, 20); err != nil {
		// 		log.Printf("[coinmarketcal] fetch err: %v", err)
		// 	} else if len(items) > 0 {
		// 		genericItems := make([]map[string]any, 0, len(items))
		// 		for _, it := range items {
		// 			genericItems = append(genericItems, map[string]any{
		// 				"source":      it.Source,
		// 				"external_id": it.ExternalID,
		// 				"title":       it.Title,
		// 				"summary":     it.Summary,
		// 				"url":         it.URL,
		// 				"tags":        it.Tags,
		// 				"release_ms":  it.ReleaseMS,
		// 				"is_event":    it.IsEvent,
		// 				"sentiment":   it.Sentiment,
		// 				"heat_score":  it.HeatScore,
		// 				"verified":    false,
		// 			})
		// 		}
		// 		payload := map[string]any{"items": genericItems}
		// 		postURL := strings.TrimRight(*apiBase, "/") + "/ingest/coinmarketcal/announcements"
		// 		var out map[string]any
		// 		if err := netutil.PostJSON(ctx, postURL, payload, &out); err != nil {
		// 			log.Printf("[coinmarketcal] ingest err: %v", err)
		// 		} else {
		// 			log.Printf("[coinmarketcal] ingested: %v (count=%d)", out, len(items))
		// 			added += len(items)
		// 		}
		// 	}
		// }

		// ===== 第三层：Binance（校验和补齐） =====
		// Binance 功能暂时关闭，保留代码以便将来恢复
		if *binanceEnable {
			if items, err := fetchBinance(ctx, httpClient, cats, *pageSize); err != nil {
				log.Printf("[binance] fetch err: %v", err)
			} else if len(items) > 0 {
				payload := binanceIngestReq{Items: make([]binanceIngestItem, 0, len(items))}
				for _, it := range items {
					key := "binance|" + it.Code
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					payload.Items = append(payload.Items, it)
				}
				if len(payload.Items) > 0 {
					postURL := strings.TrimRight(*apiBase, "/") + "/ingest/binance/announcements"
					var out map[string]any
					if err := netutil.PostJSON(ctx, postURL, &payload, &out); err != nil {
						log.Printf("[binance] ingest err: %v", err)
					} else {
						log.Printf("[binance] ingested: %v (count=%d)", out, len(payload.Items))
						added += len(payload.Items)
					}
				}
			}
		}

		// ===== 第三层：OKX（校验和补齐） =====
		if *okxEnable {
			if items, err := fetchOKX(ctx, httpClient, 20); err != nil {
				log.Printf("[okx] fetch err: %v", err)
			} else if len(items) > 0 {
				genericItems := make([]map[string]any, 0, len(items))
				for _, it := range items {
					// 标准化 URL
					normalizedURL := strings.TrimRight(strings.TrimSpace(it.URL), "/")
					// 使用 URL 作为去重键
					key := "okx|" + normalizedURL
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}

					// 使用标准化后的 URL
					it.URL = normalizedURL

					genericItems = append(genericItems, map[string]any{
						"source":      "okx",
						"external_id": it.Code,
						"title":       it.Title,
						"summary":     it.Summary,
						"url":         it.URL,
						"tags":        []string{},
						"release_ms":  it.ReleaseMS,
						"verified":    true, // 第三层：官方源
					})
				}
				if len(genericItems) > 0 {
					payload := map[string]any{"items": genericItems}
					postURL := strings.TrimRight(*apiBase, "/") + "/ingest/okx/announcements"
					var out map[string]any
					if err := netutil.PostJSON(ctx, postURL, payload, &out); err != nil {
						log.Printf("[okx] ingest err: %v", err)
					} else {
						log.Printf("[okx] ingested: %v (count=%d, filtered=%d)", out, len(genericItems), len(items)-len(genericItems))
						added += len(genericItems)
					}
				} else {
					log.Printf("[okx] all items already seen, skipped")
				}
			}
		}

		// ===== 第三层：Bybit（校验和补齐） =====
		if *bybitEnable {
			if items, err := fetchBybit(ctx, httpClient, 20); err != nil {
				log.Printf("[bybit] fetch err: %v", err)
			} else if len(items) > 0 {
				genericItems := make([]map[string]any, 0, len(items))
				for _, it := range items {
					// 标准化 URL
					normalizedURL := strings.TrimRight(strings.TrimSpace(it.URL), "/")
					// 使用 URL 作为去重键
					key := "bybit|" + normalizedURL
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}

					// 使用标准化后的 URL
					it.URL = normalizedURL

					genericItems = append(genericItems, map[string]any{
						"source":      "bybit",
						"external_id": it.Code,
						"title":       it.Title,
						"summary":     it.Summary,
						"url":         it.URL,
						"tags":        []string{},
						"release_ms":  it.ReleaseMS,
						"verified":    true, // 第三层：官方源
					})
				}
				if len(genericItems) > 0 {
					payload := map[string]any{"items": genericItems}
					postURL := strings.TrimRight(*apiBase, "/") + "/ingest/bybit/announcements"
					var out map[string]any
					if err := netutil.PostJSON(ctx, postURL, payload, &out); err != nil {
						log.Printf("[bybit] ingest err: %v", err)
					} else {
						log.Printf("[bybit] ingested: %v (count=%d, filtered=%d)", out, len(genericItems), len(items)-len(genericItems))
						added += len(genericItems)
					}
				} else {
					log.Printf("[bybit] all items already seen, skipped")
				}
			}
		}

		// ===== 第三层：Upbit（校验和补齐） =====
		if *upbitEnable {
			if items, err := fetchUpbit(ctx, httpClient, *upbitPageSize); err != nil {
				log.Printf("[upbit] fetch err: %v", err)
			} else if len(items) > 0 {
				payload := upbitIngestReq{Items: make([]upbitIngestItem, 0, len(items))}
				for _, it := range items {
					key := "upbit|" + strconv.FormatInt(it.ID, 10)
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					payload.Items = append(payload.Items, it)
				}
				if len(payload.Items) > 0 {
					postURL := strings.TrimRight(*apiBase, "/") + "/ingest/upbit/announcements"
					var out map[string]any
					if err := netutil.PostJSON(ctx, postURL, &payload, &out); err != nil {
						log.Printf("[upbit] ingest err: %v", err)
					} else {
						log.Printf("[upbit] ingested: %v (count=%d)", out, len(payload.Items))
						added += len(payload.Items)
					}
				}
			}
		}

		log.Printf("[ann_scanner] poll done; added=%d", added)
	}

	// 先跑一轮
	runOnce()
	for range ticker.C {
		runOnce()
	}
}

// 解析 "48,49,93" -> []int
func parseCatalogs(s string) []int {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	return out
}
