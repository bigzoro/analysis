package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const binanceCMSBase = "https://www.binance.com"
const binanceCMSPath = "/bapi/composite/v1/public/cms/article/list"

// AnnouncementFetcher fetches Binance CMS announcements.
type AnnouncementFetcher struct {
	client   *http.Client
	lang     string
	pageSize int
}

// NewAnnouncementFetcher creates a fetcher with sensible defaults.
func NewAnnouncementFetcher(client *http.Client, lang string, pageSize int) *AnnouncementFetcher {
	if client == nil {
		client = newHTTPClient()
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return &AnnouncementFetcher{client: client, lang: lang, pageSize: pageSize}
}

// Fetch retrieves announcements for the given catalog.
func (f *AnnouncementFetcher) Fetch(ctx context.Context, catalogID int) ([]Announcement, error) {
	params := url.Values{}
	params.Set("pageNo", "1")
	params.Set("pageSize", strconv.Itoa(f.pageSize))
	params.Set("type", "1")
	if catalogID > 0 {
		params.Set("catalogId", strconv.Itoa(catalogID))
	}
	if f.lang != "" {
		params.Set("lang", f.lang)
	}

	endpoint := fmt.Sprintf("%s%s?%s", binanceCMSBase, binanceCMSPath, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("clienttype", "web")
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "Mozilla/5.0 (compatible; BinanceNotifier/1.0)")
	if f.lang != "" {
		req.Header.Set("lang", f.lang)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance cms %d: %s", resp.StatusCode, string(body))
	}

	var body cmsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	if body.Code != "" && body.Code != "000000" {
		return nil, fmt.Errorf("binance cms error: code=%s message=%s", body.Code, body.Message)
	}

	if body.Data == nil {
		return []Announcement{}, nil
	}

	announcements := make([]Announcement, 0, len(body.Data.Articles))
	for _, article := range body.Data.Articles {
		code := firstNonEmpty(article.Code, article.ID)
		link := firstNonEmpty(article.ArticleURL, article.URL)
		if link != "" && !strings.HasPrefix(link, "http") {
			link = binanceCMSBase + link
		}

		releasedAt := time.Now()
		if article.ReleaseDate > 0 {
			releasedAt = time.UnixMilli(article.ReleaseDate)
		}

		announcements = append(announcements, Announcement{
			Code:       code,
			Title:      strings.TrimSpace(article.Title),
			URL:        link,
			ReleasedAt: releasedAt,
			CatalogID:  article.CatalogID,
		})
	}

	return announcements, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type cmsListResponse struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Data    *cmsListData `json:"data"`
}

type cmsListData struct {
	Total    int          `json:"total"`
	Articles []cmsArticle `json:"articles"`
}

type cmsArticle struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	ReleaseDate int64  `json:"releaseDate"`
	ArticleURL  string `json:"articleUrl"`
	URL         string `json:"url"`
	CatalogID   string `json:"catalogId"`
}
