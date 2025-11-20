// cmd/market_scanner/coincap.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CoinCap v3: https://rest.coincap.io/v3
// 我们仅需 /assets?search={symbol}，取 priceUsd/supply/maxSupply/marketCapUsd

type CCMeta struct {
	MarketCapUSD      *float64
	FDVUSD            *float64
	CirculatingSupply *float64 // supply
	TotalSupply       *float64 // maxSupply
}

type coinCapClient struct {
	baseURL string
	apiKey  string
	httpc   *http.Client

	mu    sync.RWMutex
	cache map[string]ccCacheItem // key: UPPER(symbol)
	ttl   time.Duration
}

type ccCacheItem struct {
	meta CCMeta
	ts   time.Time
}

func newCoinCapClient() *coinCapClient {
	// 使用环境变量中的代理配置（通过 config.ApplyProxy 设置）
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	return &coinCapClient{
		baseURL: "https://rest.coincap.io/v3",
		apiKey:  "292ca5251c7eab03e55f5f01f960dc635f00e2294e3963d0293764e36ff69080",
		httpc: &http.Client{
			Timeout:   15 * time.Second, // 增加超时时间，服务器网络可能较慢
			Transport: transport,
		},
		cache: make(map[string]ccCacheItem),
		ttl:   5 * time.Minute,
	}
}

func (c *coinCapClient) GetMeta(ctx context.Context, symbol string) (CCMeta, error) {
	key := strings.ToUpper(strings.TrimSpace(symbol))
	if key == "" {
		return CCMeta{}, fmt.Errorf("empty symbol")
	}
	// cache
	c.mu.RLock()
	if it, ok := c.cache[key]; ok && time.Since(it.ts) < c.ttl {
		c.mu.RUnlock()
		return it.meta, nil
	}
	c.mu.RUnlock()

	// fetch
	meta, err := c.fetch(ctx, key)
	if err != nil {
		return CCMeta{}, fmt.Errorf("coincap GetMeta for %s: %w", key, err)
	}
	c.mu.Lock()
	c.cache[key] = ccCacheItem{meta: meta, ts: time.Now()}
	c.mu.Unlock()
	return meta, nil
}

func (c *coinCapClient) fetch(ctx context.Context, symbol string) (CCMeta, error) {
	q := url.Values{}
	q.Set("search", symbol)
	if c.apiKey != "" {
		q.Set("apiKey", c.apiKey) // 兼容 query 传参
	}
	u := fmt.Sprintf("%s/assets?%s", c.baseURL, q.Encode())

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "market-scanner/coincap")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey) // 兼容 header 传参
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return CCMeta{}, fmt.Errorf("coincap request failed: %w (url: %s)", err, u)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		// 读取响应体以便调试
		bodyBytes := make([]byte, 512) // 只读前 512 字节
		resp.Body.Read(bodyBytes)
		return CCMeta{}, fmt.Errorf("coincap %s => %d, body: %s", u, resp.StatusCode, string(bodyBytes))
	}

	var raw struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return CCMeta{}, fmt.Errorf("coincap decode error for %s: %w", symbol, err)
	}
	if len(raw.Data) == 0 {
		// 记录详细的搜索信息以便调试
		log.Printf("[coincap] search returned empty for symbol=%s, url=%s", symbol, u)
		return CCMeta{}, fmt.Errorf("coincap: empty for %s (no results from search)", symbol)
	}

	// 记录搜索到的结果数量
	log.Printf("[coincap] search for %s returned %d results", symbol, len(raw.Data))

	// 选择策略：优先符号精确匹配，多个时选 marketCapUsd 最大；否则选返回里市值最大者
	var picked map[string]any
	bestMC := -1.0
	var matchedSymbols []string // 用于调试

	// 第一轮：精确匹配
	for _, it := range raw.Data {
		sym := getString(it["symbol"])
		matchedSymbols = append(matchedSymbols, sym)
		if strings.EqualFold(sym, symbol) {
			if v := getFloat(it["marketCapUsd"]); v != nil && *v > bestMC {
				picked = it
				bestMC = *v
			}
		}
	}

	// 如果精确匹配失败，记录所有返回的符号以便调试
	if picked == nil {
		log.Printf("[coincap] no exact match for %s, found symbols: %v", symbol, matchedSymbols)
		// 第二轮：选择市值最大的
		for _, it := range raw.Data {
			if v := getFloat(it["marketCapUsd"]); v != nil && *v > bestMC {
				picked = it
				bestMC = *v
			}
		}
		if picked != nil {
			pickedSym := getString(picked["symbol"])
			log.Printf("[coincap] using best match for %s: %s (mcap=%.2f)", symbol, pickedSym, bestMC)
		}
	} else {
		pickedSym := getString(picked["symbol"])
		log.Printf("[coincap] exact match for %s: %s (mcap=%.2f)", symbol, pickedSym, bestMC)
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

	return CCMeta{
		MarketCapUSD:      marketCap,
		FDVUSD:            fdv,
		CirculatingSupply: supply,
		TotalSupply:       maxSupply,
	}, nil
}

func getString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
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
