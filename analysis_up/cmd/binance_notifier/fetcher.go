package main

import (
	"bytes"
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

var binanceCMSPaths = []string{
	"/bapi/composite/v1/public/cms/article/list",
	"/bapi/cms/v1/friendly/cms/article/list",
}

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
	var errs []string
	for _, path := range binanceCMSPaths {
		anns, err := f.fetchWithPath(ctx, path, catalogID)
		if err == nil {
			return anns, nil
		}
		errs = append(errs, err.Error())
	}

	if len(errs) == 0 {
		return nil, fmt.Errorf("binance cms: no endpoints configured")
	}

	return nil, fmt.Errorf("binance cms failed: %s", strings.Join(errs, "; "))
}

func (f *AnnouncementFetcher) fetchWithPath(ctx context.Context, path string, catalogID int) ([]Announcement, error) {
	endpoint := fmt.Sprintf("%s%s", binanceCMSBase, path)

	payload := map[string]any{
		"pageNo":   1,
		"pageSize": f.pageSize,
		"type":     1,
	}
	if catalogID > 0 {
		payload["catalogId"] = strconv.Itoa(catalogID)
	}
	if f.lang != "" {
		payload["lang"] = f.lang
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("clienttype", "web")
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "Mozilla/5.0 (compatible; BinanceNotifier/1.0)")
	req.Header.Set("x-ui-request", "true")
	if f.lang != "" {
		req.Header.Set("lang", f.lang)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		// Some regions still expose a GET variant, try it as a fallback before giving up.
		return f.fetchWithQuery(ctx, path, catalogID)
	}

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 256 {
			body = body[:256]
		}
		return nil, fmt.Errorf("%s %d: %s", path, resp.StatusCode, string(body))
	}

	return decodeCMSResponse(resp.Body)
}

func (f *AnnouncementFetcher) fetchWithQuery(ctx context.Context, path string, catalogID int) ([]Announcement, error) {
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

	endpoint := fmt.Sprintf("%s%s?%s", binanceCMSBase, path, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("clienttype", "web")
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "Mozilla/5.0 (compatible; BinanceNotifier/1.0)")
	req.Header.Set("x-ui-request", "true")
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
		if len(body) > 256 {
			body = body[:256]
		}
		return nil, fmt.Errorf("%s %d: %s", path, resp.StatusCode, string(body))
	}

	return decodeCMSResponse(resp.Body)
}

func decodeCMSResponse(r io.Reader) ([]Announcement, error) {
	var body cmsListResponse
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return nil, err
	}

	if body.Code != "" && body.Code != "000000" {
		return nil, fmt.Errorf("cms error code=%s message=%s", body.Code, body.Message)
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

		catalogID := parseCatalogID(article.CatalogID)

		announcements = append(announcements, Announcement{
			Code:       code,
			Title:      strings.TrimSpace(article.Title),
			URL:        link,
			ReleasedAt: releasedAt,
			CatalogID:  catalogID,
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
	ID          string          `json:"id"`
	Code        string          `json:"code"`
	Title       string          `json:"title"`
	ReleaseDate int64           `json:"releaseDate"`
	ArticleURL  string          `json:"articleUrl"`
	URL         string          `json:"url"`
	CatalogID   json.RawMessage `json:"catalogId"`
}

func parseCatalogID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}

	var asNumber json.Number
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return asNumber.String()
	}

	return strings.TrimSpace(string(raw))
}
