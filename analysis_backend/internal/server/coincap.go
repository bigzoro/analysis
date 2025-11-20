// internal/server/coincap.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ----- v3 说明 -----
// v3 基础域名： https://rest.coincap.io/v3
// 常用端点：    GET /assets?search={symbol}
// 认证：        建议同时支持 query 参数 ?apiKey=... 与 Header x-api-key
// 参考：Swagger 显示 v3 /assets 与 /assets/{slug}；CS50 示例用法包含 ?apiKey=...。
// Docs: rest.coincap.io/api-docs（Swagger UI）；pro.coincap.io/api-docs（带 Authorize 按钮）。
// ---------------------------------------------

// v3 /assets 返回我们关心的字段是动态类型（可能是字符串或数字），
// 这里用 map[string]any 更稳妥，再做统一转换。
type marketMeta struct {
	MarketCapUSD *float64
	FDVUSD       *float64
	Circulating  *float64 // supply
	TotalSupply  *float64 // maxSupply
	FetchedAt    time.Time
}

type coinCapCache struct {
	baseURL    string
	apiKey     string
	ttl        time.Duration
	httpClient *http.Client

	mu    sync.RWMutex
	items map[string]marketMeta // key: upper symbol
}

func newCoinCapCache() *coinCapCache {
	apiKey := "292ca5251c7eab03e55f5f01f960dc635f00e2294e3963d0293764e36ff69080"
	return &coinCapCache{
		baseURL: "https://rest.coincap.io/v3",
		apiKey:  apiKey,
		ttl:     5 * time.Minute,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		items: make(map[string]marketMeta),
	}
}

func (c *coinCapCache) Get(ctx context.Context, symbol string) (marketMeta, error) {
	key := strings.ToUpper(strings.TrimSpace(symbol))
	if key == "" {
		return marketMeta{}, fmt.Errorf("empty symbol")
	}

	// fast path（命中有效缓存）
	c.mu.RLock()
	m, ok := c.items[key]
	c.mu.RUnlock()
	if ok && time.Since(m.FetchedAt) < c.ttl {
		return m, nil
	}

	// miss：拉取
	val, err := c.fetchAssetMeta(ctx, key)
	if err != nil {
		return marketMeta{}, err
	}

	c.mu.Lock()
	c.items[key] = val
	c.mu.Unlock()
	return val, nil
}

func (c *coinCapCache) fetchAssetMeta(ctx context.Context, symbol string) (marketMeta, error) {
	// v3 支持 /assets?search={symbol}
	q := url.Values{}
	q.Set("search", symbol)
	if c.apiKey != "" {
		// 同时放 query 与 header，最大化兼容
		q.Set("apiKey", c.apiKey)
	}

	u := fmt.Sprintf("%s/assets?%s", c.baseURL, q.Encode())

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "analysis-up/market-meta")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey) // 常见写法
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return marketMeta{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return marketMeta{}, fmt.Errorf("coincap %s => %d", u, resp.StatusCode)
	}

	var raw struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return marketMeta{}, err
	}
	if len(raw.Data) == 0 {
		return marketMeta{}, fmt.Errorf("coincap: no data for symbol=%s", symbol)
	}

	// 选择规则：
	// 1) 优先符号精确匹配（不区分大小写）
	// 2) 多个精确匹配时取 marketCapUsd 最大的
	// 3) 精确匹配没有命中时，若仅返回 1 个结果，则选它
	var picked map[string]any
	var pickedMC float64 = -1

	for _, it := range raw.Data {
		sym := getString(it["symbol"])
		if strings.EqualFold(sym, symbol) {
			mc := getFloat(it["marketCapUsd"])
			mcVal := derefOr(mc, -1)
			if mcVal > pickedMC {
				picked = it
				pickedMC = mcVal
			}
		}
	}
	if picked == nil {
		if len(raw.Data) == 1 {
			picked = raw.Data[0]
		} else {
			// 兜底：选 marketCapUsd 最大者
			for _, it := range raw.Data {
				mc := getFloat(it["marketCapUsd"])
				mcVal := derefOr(mc, -1)
				if mcVal > pickedMC {
					picked = it
					pickedMC = mcVal
				}
			}
			if picked == nil {
				return marketMeta{}, fmt.Errorf("coincap: cannot pick item for symbol=%s", symbol)
			}
		}
	}

	price := getFloat(picked["priceUsd"])
	supply := getFloat(picked["supply"])
	maxSupply := getFloat(picked["maxSupply"])
	marketCap := getFloat(picked["marketCapUsd"])

	var fdv *float64
	if price != nil && maxSupply != nil {
		v := (*price) * (*maxSupply)
		fdv = &v
	}

	return marketMeta{
		MarketCapUSD: marketCap,
		FDVUSD:       fdv,
		Circulating:  supply,
		TotalSupply:  maxSupply,
		FetchedAt:    time.Now(),
	}, nil
}

// --------- 小工具：把 string/number 转 float64 ----------

func getString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		// 避免科学计数法，简单转成十进制字符串
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func getFloat(v any) *float64 {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return &f
		}
	case json.Number:
		if f, err := strconv.ParseFloat(t.String(), 64); err == nil {
			return &f
		}
	case float64:
		return &t
	case float32:
		f := float64(t)
		return &f
	case int, int32, int64, uint, uint32, uint64:
		f, _ := strconv.ParseFloat(fmt.Sprint(t), 64)
		return &f
	}
	return nil
}

func derefOr(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}
