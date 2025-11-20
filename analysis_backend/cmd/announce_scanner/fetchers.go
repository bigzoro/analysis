// cmd/announce_scanner/fetchers.go
// 多层次公告抓取实现

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// =============================
// 第一层：CoinCarp Exchange Latest Announcements
// =============================

type coincarpItem struct {
	Source     string   `json:"source"`
	ExternalID string   `json:"external_id"`
	NewsCode   string   `json:"news_code"` // CoinCarp newscode
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	URL        string   `json:"url"`
	Tags       []string `json:"tags"`
	ReleaseMS  int64    `json:"release_ms"`
	Exchange   string   `json:"exchange"`
}

// 抓取 CoinCarp 交易所公告（使用 API）
// issuetime: 获取此时间之后的公告（Unix 时间戳，秒）。如果为 0，则获取最近 24 小时的公告
func fetchCoinCarp(ctx context.Context, client *http.Client, issuetime int64, limit int) ([]coincarpItem, error) {
	// CoinCarp API: 获取所有交易所公告
	// 如果 issuetime 为 0，使用最近 24 小时作为起始点
	if issuetime == 0 {
		issuetime = time.Now().Add(-24 * time.Hour).Unix()
	}
	url := fmt.Sprintf("https://sapi.coincarp.com/api/v1/news/exchange/channelannoucement?channelcode=notice&tagcode=all&issuetime=%d&lang=zh-CN", issuetime)

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Newscode     string `json:"newscode"`
			Newstitle    string `json:"newstitle"`
			Logo         string `json:"logo"`
			Description  string `json:"description"`
			Relatedcode  string `json:"relatedcode"`
			Relatedname  string `json:"relatedname"`
			Issuetime    int64  `json:"issuetime"`
			Issuetimestr string `json:"issuetimestr"`
		} `json:"data"`
	}

	if err := httpGetJSON(ctx, client, url, &resp); err != nil {
		return nil, fmt.Errorf("coincarp api: %w", err)
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("coincarp api error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	maxItems := limit
	if len(resp.Data) < limit {
		maxItems = len(resp.Data)
	}
	items := make([]coincarpItem, 0, maxItems)
	for i, d := range resp.Data {
		if i >= limit {
			break
		}

		// 转换时间戳（秒 -> 毫秒）
		releaseMS := d.Issuetime * 1000
		if releaseMS == 0 {
			releaseMS = time.Now().UTC().UnixMilli()
		}

		// 构建 URL（使用 newscode），根据 CoinCarp 实际 URL 格式
		// 格式：https://www.coincarp.com/zh/exchange/announcement/{newscode}/
		url := fmt.Sprintf("https://www.coincarp.com/zh/exchange/announcement/%s/", strings.TrimSpace(d.Newscode))
		// 标准化 URL：去除末尾斜杠和空格（但保留路径中的斜杠）
		url = strings.TrimRight(strings.TrimSpace(url), "/")

		// 提取摘要（限制长度）
		summary := strings.TrimSpace(d.Description)
		if len(summary) > 500 {
			summary = summary[:500]
		}

		// 提取标签
		tags := extractTags(d.Newstitle, summary)

		items = append(items, coincarpItem{
			Source:     "coincarp",
			ExternalID: d.Newscode,
			NewsCode:   strings.TrimSpace(d.Newscode),
			Title:      strings.TrimSpace(d.Newstitle),
			Summary:    summary,
			URL:        url,
			Tags:       tags,
			ReleaseMS:  releaseMS,
			Exchange:   strings.ToLower(d.Relatedcode), // 使用 relatedcode 作为交易所代码
		})
	}

	return items, nil
}

// =============================
// 第二层：CryptoPanic + CoinMarketCal
// =============================

type cryptopanicItem struct {
	Source     string   `json:"source"`
	ExternalID string   `json:"external_id"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	URL        string   `json:"url"`
	Tags       []string `json:"tags"`
	ReleaseMS  int64    `json:"release_ms"`
	IsEvent    bool     `json:"is_event"`
	Sentiment  string   `json:"sentiment"`
	HeatScore  int      `json:"heat_score"`
}

// CryptoPanic API（需要 API Key，免费版有限制）
func fetchCryptoPanic(ctx context.Context, client *http.Client, apiKey string, limit int) ([]cryptopanicItem, error) {
	if apiKey == "" {
		// 如果没有 API Key，尝试爬取网页
		return fetchCryptoPanicWeb(ctx, client, limit)
	}

	url := fmt.Sprintf("https://cryptopanic.com/api/v1/posts/?auth_token=%s&public=true&filter=hot&limit=%d", apiKey, limit)

	var resp struct {
		Results []struct {
			ID      int    `json:"id"`
			Title   string `json:"title"`
			URL     string `json:"url"`
			Created string `json:"created_at"`
			Votes   struct {
				Positive int `json:"positive"`
				Negative int `json:"negative"`
			} `json:"votes"`
			Source struct {
				Title string `json:"title"`
			} `json:"source"`
		} `json:"results"`
	}

	if err := httpGetJSON(ctx, client, url, &resp); err != nil {
		return nil, fmt.Errorf("cryptopanic api: %w", err)
	}

	items := make([]cryptopanicItem, 0, len(resp.Results))
	for _, r := range resp.Results {
		releaseMS := parseTimeString(r.Created)
		if releaseMS == 0 {
			releaseMS = time.Now().UTC().UnixMilli()
		}

		// 计算情绪和热度
		sentiment := "neutral"
		if r.Votes.Positive > r.Votes.Negative*2 {
			sentiment = "positive"
		} else if r.Votes.Negative > r.Votes.Positive*2 {
			sentiment = "negative"
		}

		heatScore := r.Votes.Positive + r.Votes.Negative
		if heatScore > 100 {
			heatScore = 100
		}

		// 判断是否为重要事件
		isEvent := isImportantEvent(r.Title, r.Source.Title)

		items = append(items, cryptopanicItem{
			Source:     "cryptopanic",
			ExternalID: fmt.Sprintf("cp_%d", r.ID),
			Title:      r.Title,
			Summary:    "",
			URL:        r.URL,
			Tags:       extractTags(r.Title, ""),
			ReleaseMS:  releaseMS,
			IsEvent:    isEvent,
			Sentiment:  sentiment,
			HeatScore:  heatScore,
		})
	}

	return items, nil
}

// CryptoPanic 网页爬取（备用方案，暂时不使用，因为需要 goquery）
func fetchCryptoPanicWeb(ctx context.Context, client *http.Client, limit int) ([]cryptopanicItem, error) {
	// 暂时返回空，如果 API 失败，可以后续实现网页爬取
	log.Printf("[cryptopanic] web scraping not implemented, please use API key")
	return nil, nil
}

// CoinMarketCal API
func fetchCoinMarketCal(ctx context.Context, client *http.Client, limit int) ([]cryptopanicItem, error) {
	// CoinMarketCal 需要 API Key 且被 Cloudflare 保护，暂时跳过
	// 如果需要使用，请注册获取 API Key 并配置
	log.Printf("[coinmarketcal] skipped: requires API key and Cloudflare protection")
	return nil, nil

	// 以下代码保留，如果未来有 API Key 可以使用
	/*
		url := fmt.Sprintf("https://coinmarketcal.com/api/v1/events?page=1&max=%d", limit)

		var resp struct {
			Body []struct {
				ID          int    `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Date        string `json:"date_event"`
				URL         string `json:"url"`
				Coins       []struct {
					Name string `json:"name"`
				} `json:"coins"`
			} `json:"body"`
		}

		if err := httpGetJSON(ctx, client, url, &resp); err != nil {
			return nil, fmt.Errorf("coinmarketcal: %w", err)
		}

		items := make([]cryptopanicItem, 0, len(resp.Body))
		for _, e := range resp.Body {
			releaseMS := parseTimeString(e.Date)
			if releaseMS == 0 {
				releaseMS = time.Now().UTC().UnixMilli()
			}

			tags := make([]string, 0, len(e.Coins))
			for _, c := range e.Coins {
				tags = append(tags, c.Name)
			}

			items = append(items, cryptopanicItem{
				Source:     "coinmarketcal",
				ExternalID: fmt.Sprintf("cmc_%d", e.ID),
				Title:      e.Title,
				Summary:    e.Description,
				URL:        e.URL,
				Tags:       tags,
				ReleaseMS:  releaseMS,
				IsEvent:    true, // CoinMarketCal 都是事件
				Sentiment:  "neutral",
				HeatScore:  50, // 默认中等热度
			})
		}

		return items, nil
	*/
}

// =============================
// 第三层：重点交易所直接抓取（校验和补齐）
// =============================

// OKX 公告抓取
func fetchOKX(ctx context.Context, client *http.Client, limit int) ([]binanceIngestItem, error) {
	// OKX 公告 API
	url := "https://www.okx.com/api/v5/announcement/public?locale=zh_CN&limit=" + strconv.Itoa(limit)

	var resp struct {
		Code string `json:"code"`
		Data []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Summary   string `json:"summary"`
			URL       string `json:"url"`
			PublishTS int64  `json:"publishTime"`
		} `json:"data"`
	}

	if err := httpGetJSON(ctx, client, url, &resp); err != nil {
		// OKX API 可能偶尔失败，返回空列表而不是错误
		log.Printf("[okx] fetch err (will retry next time): %v", err)
		return nil, nil
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: code=%s", resp.Code)
	}

	items := make([]binanceIngestItem, 0, len(resp.Data))
	for _, d := range resp.Data {
		releaseMS := d.PublishTS
		if releaseMS == 0 {
			releaseMS = time.Now().UTC().UnixMilli()
		}

		items = append(items, binanceIngestItem{
			Source:    "okx",
			Code:      d.ID,
			Title:     d.Title,
			Summary:   d.Summary,
			URL:       d.URL,
			ReleaseMS: releaseMS,
		})
	}

	return items, nil
}

// Bybit 公告抓取
func fetchBybit(ctx context.Context, client *http.Client, limit int) ([]binanceIngestItem, error) {
	// Bybit 公告页面
	url := "https://api.bybit.com/v5/announcements/index?locale=zh-CN&limit=" + strconv.Itoa(limit)

	var resp struct {
		RetCode int `json:"retCode"`
		Result  struct {
			List []struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Summary   string `json:"summary"`
				URL       string `json:"url"`
				CreatedAt int64  `json:"createdAt"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := httpGetJSON(ctx, client, url, &resp); err != nil {
		// Bybit API 可能偶尔失败，返回空列表而不是错误
		log.Printf("[bybit] fetch err (will retry next time): %v", err)
		return nil, nil
	}

	if resp.RetCode != 0 {
		// 500 错误可能是临时问题，记录但不中断
		if resp.RetCode == 500 {
			log.Printf("[bybit] api error: retCode=%d (temporary server error, will retry next time)", resp.RetCode)
			return nil, nil
		}
		return nil, fmt.Errorf("bybit api error: retCode=%d", resp.RetCode)
	}

	items := make([]binanceIngestItem, 0, len(resp.Result.List))
	for _, d := range resp.Result.List {
		releaseMS := d.CreatedAt
		if releaseMS == 0 {
			releaseMS = time.Now().UTC().UnixMilli()
		}

		items = append(items, binanceIngestItem{
			Source:    "bybit",
			Code:      d.ID,
			Title:     d.Title,
			Summary:   d.Summary,
			URL:       d.URL,
			ReleaseMS: releaseMS,
		})
	}

	return items, nil
}

// =============================
// 工具函数
// =============================

func extractExchange(title, text string) string {
	exchanges := []string{"binance", "okx", "bybit", "upbit", "coinbase", "kraken", "huobi", "gate"}
	lower := strings.ToLower(title + " " + text)
	for _, ex := range exchanges {
		if strings.Contains(lower, ex) {
			return ex
		}
	}
	return ""
}

func extractTags(title, summary string) []string {
	tags := []string{}
	text := strings.ToLower(title + " " + summary)

	// 提取币种标签
	coinPattern := regexp.MustCompile(`\b([A-Z]{2,10})\b`)
	coins := coinPattern.FindAllString(title, -1)
	for _, c := range coins {
		if len(c) >= 2 && len(c) <= 10 {
			tags = append(tags, c)
		}
	}

	// 提取关键词标签
	keywords := []string{"listing", "delisting", "maintenance", "upgrade", "staking", "airdrop"}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			tags = append(tags, kw)
		}
	}

	return tags
}

func parseTimeString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// 尝试多种时间格式
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"Jan 2, 2006",
		"2 hours ago",
		"1 day ago",
	}

	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t.UTC().UnixMilli()
		}
	}

	// 处理相对时间
	if strings.Contains(s, "ago") || strings.Contains(s, "前") {
		return time.Now().UTC().UnixMilli()
	}

	return 0
}

func generateID(prefix, url string) string {
	// 从 URL 提取唯一标识
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if last != "" {
			return prefix + "_" + last
		}
	}
	return prefix + "_" + strconv.FormatInt(time.Now().Unix(), 10)
}

func isImportantEvent(title, source string) bool {
	text := strings.ToLower(title + " " + source)
	keywords := []string{
		"listing", "上币", "mainnet", "主网上线",
		"halving", "减半", "fork", "硬分叉",
		"partnership", "合作", "investment", "投资",
		"launch", "发布", "upgrade", "升级",
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func detectSentiment(text string) string {
	text = strings.ToLower(text)
	positive := []string{"launch", "partnership", "growth", "upgrade", "success", "gain"}
	negative := []string{"hack", "crash", "delist", "ban", "warning", "loss", "down"}

	for _, p := range positive {
		if strings.Contains(text, p) {
			return "positive"
		}
	}
	for _, n := range negative {
		if strings.Contains(text, n) {
			return "negative"
		}
	}
	return "neutral"
}
