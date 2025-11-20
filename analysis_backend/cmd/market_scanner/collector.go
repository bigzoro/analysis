package main

import (
	"analysis/internal/netutil"
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/keighl/postmark"
)

// 币安 24h 行情一条
type binance24hTicker struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	Volume             string `json:"volume"`
	PriceChangePercent string `json:"priceChangePercent"`
}

// POST 给后端的数据结构（扩展了市值/供给字段）
type postItem struct {
	Symbol            string   `json:"symbol"`
	LastPrice         string   `json:"last_price"`
	Volume            string   `json:"volume"`
	PriceChangePct    float64  `json:"price_change_percent"`
	MarketCapUSD      *float64 `json:"market_cap_usd,omitempty"`
	FDVUSD            *float64 `json:"fdv_usd,omitempty"`
	CirculatingSupply *float64 `json:"circulating_supply,omitempty"`
	TotalSupply       *float64 `json:"total_supply,omitempty"`
}

type postBody struct {
	Kind      string     `json:"kind"` // spot/futures
	Bucket    string     `json:"bucket"`
	FetchedAt string     `json:"fetched_at"`
	Items     []postItem `json:"items"`
}

// 用于跨时间段比较的“快照”
type snapshot struct {
	Kind      string
	BucketUTC time.Time
	Prices    map[string]float64 // symbol -> lastPrice
	Pcts      map[string]float64 // symbol -> 24h pct
}

// 告警配置（Postmark）
type AlertConfig struct {
	Enable    bool
	Threshold float64 // 0.10 = 10%

	PostmarkServerToken  string
	PostmarkAccountToken string // 可留空
	PostmarkStream       string // outbound / broadcast

	From  string
	ToCSV string // 逗号分隔
}

type BinanceMarketCollector struct {
	apiBase       string
	interval      time.Duration
	limit         int
	enableSpot    bool
	enableFutures bool
	loc           *time.Location

	alert AlertConfig

	// 注意：黑名单不再在同步时使用，前端显示时会过滤黑名单

	// 上一段快照（用于回调比较）
	prevSpot    *snapshot
	prevFutures *snapshot

	// 内部统一存 UTC 的下次时间
	nextUTC time.Time

	cc *coinCapClient
}

// ===== 构造 =====

func NewBinanceMarketCollector(
	apiBase string,
	interval time.Duration,
	limit int,
	enableSpot, enableFutures bool,
	loc *time.Location,
	alert AlertConfig,
) *BinanceMarketCollector {

	nowLocal := time.Now().In(loc)
	nextLocal := nowLocal.Truncate(interval)
	if nextLocal.Before(nowLocal) {
		nextLocal = nextLocal.Add(interval)
	}
	nextUTC := nextLocal.UTC()

	return &BinanceMarketCollector{
		apiBase:       apiBase,
		interval:      interval,
		limit:         limit,
		enableSpot:    enableSpot,
		enableFutures: enableFutures,
		loc:           loc,
		alert:         alert,
		nextUTC:       nextUTC,
		cc:            newCoinCapClient(),
	}
}

func (c *BinanceMarketCollector) NextTimeUTC() time.Time   { return c.nextUTC }
func (c *BinanceMarketCollector) NextTimeLocal() time.Time { return c.nextUTC.In(c.loc) }

// ===== 主流程 =====

func (c *BinanceMarketCollector) RunOnce(ctx context.Context) error {
	// 首次建立“上一段”基线
	if c.enableSpot {
		if snap, err := c.collectPostBuild(ctx, "spot"); err != nil {
			log.Printf("[market_scanner] spot init error: %v", err)
		} else {
			c.prevSpot = snap
		}
	}
	if c.enableFutures {
		if snap, err := c.collectPostBuild(ctx, "futures"); err != nil {
			log.Printf("[market_scanner] futures init error: %v", err)
		} else {
			c.prevFutures = snap
		}
	}
	return nil
}

func (c *BinanceMarketCollector) Tick(ctx context.Context, nowUTC time.Time) bool {
	if nowUTC.Before(c.nextUTC) {
		// 日志已移到 main 函数中控制频率
		return false
	}
	log.Printf("[market_scanner] tick started, now_local=%s now_utc=%s",
		nowUTC.In(c.loc).Format(time.RFC3339), nowUTC.Format(time.RFC3339))

	// 进入下一段：采集 → 入库 → 与上一段比较（可能发告警）
	if c.enableSpot {
		if snap, err := c.collectPostBuild(ctx, "spot"); err != nil {
			log.Printf("[market_scanner] spot error: %v", err)
		} else {
			c.maybeAlertPullback("spot", c.prevSpot, snap)
			c.prevSpot = snap
		}
	}
	if c.enableFutures {
		if snap, err := c.collectPostBuild(ctx, "futures"); err != nil {
			log.Printf("[market_scanner] futures error: %v", err)
		} else {
			c.maybeAlertPullback("futures", c.prevFutures, snap)
			c.prevFutures = snap
		}
	}

	// 下一个对齐点（按本地时区）
	nextLocal := c.nextUTC.In(c.loc).Add(c.interval)
	c.nextUTC = nextLocal.UTC()
	return true
}

// ===== 采集 + 入库 + 构建快照 =====

func (c *BinanceMarketCollector) collectPostBuild(ctx context.Context, kind string) (*snapshot, error) {
	// 1) 选 URL
	var url string
	switch kind {
	case "spot":
		url = "https://api.binance.com/api/v3/ticker/24hr"
	case "futures":
		url = "https://dapi.binance.com/dapi/v1/ticker/24hr"
	default:
		kind = "spot"
		url = "https://api.binance.com/api/v3/ticker/24hr"
	}

	startFetch := time.Now()
	log.Printf("[market_scanner] [%s] fetching from %s ...", kind, url)

	// 2) 拉币安
	var tickers []binance24hTicker
	if err := netutil.GetJSON(ctx, url, &tickers); err != nil {
		log.Printf("[market_scanner] [%s] fetch failed after %s: %v", kind, time.Since(startFetch), err)
		return nil, fmt.Errorf("fetch %s 24hr: %w", kind, err)
	}
	log.Printf("[market_scanner] [%s] fetched %d tickers in %s", kind, len(tickers), time.Since(startFetch))

	// 3) 过滤：现货只保留 USDT 交易对，期货保留币本位合约（USD_PERP）
	// 注意：同步时不过滤黑名单，直接保存涨幅前20个，前端显示时再过滤
	filtered := make([]binance24hTicker, 0, len(tickers))
	for _, t := range tickers {
		symbolUpper := strings.ToUpper(t.Symbol)
		var shouldInclude bool
		if kind == "spot" {
			// 现货：只保留 USDT 交易对（不检查黑名单，黑名单在前端显示时过滤）
			shouldInclude = strings.HasSuffix(symbolUpper, "USDT")
		} else if kind == "futures" {
			// 期货：保留币本位合约（USD_PERP 格式）
			shouldInclude = strings.HasSuffix(symbolUpper, "USD_PERP")
		}
		if shouldInclude {
			filtered = append(filtered, t)
		}
	}
	if kind == "spot" {
		log.Printf("[market_scanner] [%s] filtered USDT tickers: %d", kind, len(filtered))
	} else {
		log.Printf("[market_scanner] [%s] filtered USD_PERP tickers: %d", kind, len(filtered))
	}

	// 4) 按涨幅排序（降序）
	sort.Slice(filtered, func(i, j int) bool {
		pi, _ := strconv.ParseFloat(filtered[i].PriceChangePercent, 64)
		pj, _ := strconv.ParseFloat(filtered[j].PriceChangePercent, 64)
		return pi > pj
	})

	// 5) 取 Top N
	if c.limit > 0 && len(filtered) > c.limit {
		filtered = filtered[:c.limit]
	}
	if len(filtered) > 0 {
		maxPrint := len(filtered)
		if maxPrint > 3 {
			maxPrint = 3
		}
		sb := strings.Builder{}
		sb.WriteString("[market_scanner] [")
		sb.WriteString(kind)
		sb.WriteString("] top symbols: ")
		for i := 0; i < maxPrint; i++ {
			pct, _ := strconv.ParseFloat(filtered[i].PriceChangePercent, 64)
			sb.WriteString(fmt.Sprintf("%s(%.2f%%)", filtered[i].Symbol, pct))
			if i != maxPrint-1 {
				sb.WriteString(", ")
			}
		}
		log.Print(sb.String())
	}

	// 6) 本段时间桶（本地对齐 → UTC）
	nowLocal := time.Now().In(c.loc)
	bucketLocal := nowLocal.Truncate(c.interval)
	bucketUTC := bucketLocal.UTC()
	log.Printf(
		"[market_scanner] [%s] bucket_local=%s bucket_utc=%s",
		kind,
		bucketLocal.Format(time.RFC3339),
		bucketUTC.Format(time.RFC3339),
	)

	// 7) 上报回你后端
	payload := struct {
		Kind      string     `json:"kind"`
		Bucket    string     `json:"bucket"`
		FetchedAt string     `json:"fetched_at"`
		Items     []postItem `json:"items"`
	}{
		Kind:      kind,
		Bucket:    bucketUTC.Format(time.RFC3339),
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     make([]postItem, 0, len(filtered)),
	}
	for _, f := range filtered {
		pct, _ := strconv.ParseFloat(f.PriceChangePercent, 64)

		row := postItem{
			Symbol:         f.Symbol,
			LastPrice:      f.LastPrice,
			Volume:         f.Volume,
			PriceChangePct: pct,
		}

		base := baseSymbolFromPair(row.Symbol)

		if base != "" && base != row.Symbol {
			// 15s 超时，避免拖慢整体（与 coinCapClient 的 Timeout 一致）
			ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
			meta, err := c.cc.GetMeta(ctx2, base)
			cancel()
			if err == nil {
				row.MarketCapUSD = meta.MarketCapUSD
				row.FDVUSD = meta.FDVUSD
				row.CirculatingSupply = meta.CirculatingSupply
				row.TotalSupply = meta.TotalSupply
				log.Printf("[market_scanner] [%s] coincap meta for %s (from %s): mcap=%v fdv=%v", kind, base, row.Symbol,
					meta.MarketCapUSD, meta.FDVUSD)
			} else {
				// 记录错误但不中断流程
				log.Printf("[market_scanner] [%s] coincap fetch failed for %s (from %s): %v", kind, base, row.Symbol, err)
			}
		} else if base == "" {
			log.Printf("[market_scanner] [%s] warning: empty base symbol extracted from %s, skipping coincap", kind, row.Symbol)
		} else if base == row.Symbol {
			log.Printf("[market_scanner] [%s] warning: base symbol same as pair %s, skipping coincap", kind, row.Symbol)
		}

		payload.Items = append(payload.Items, row)
	}
	ingestURL := fmt.Sprintf("%s/ingest/binance/market", c.apiBase)
	log.Printf("[market_scanner] [%s] posting to %s ...", kind, ingestURL)
	var resp struct {
		OK bool `json:"ok"`
	}
	if err := netutil.PostJSON(ctx, ingestURL, &payload, &resp); err != nil {
		log.Printf("[market_scanner] [%s] post failed: %v", kind, err)
		return nil, fmt.Errorf("ingest binance market: %w", err)
	}
	log.Printf("[market_scanner] [%s] post ok, ok=%v", kind, resp.OK)

	// 8) 构建快照（给下一段比较）
	snap := &snapshot{
		Kind:      kind,
		BucketUTC: bucketUTC,
		Prices:    make(map[string]float64, len(filtered)),
		Pcts:      make(map[string]float64, len(filtered)),
	}
	for _, f := range filtered {
		price, _ := strconv.ParseFloat(f.LastPrice, 64)
		pct, _ := strconv.ParseFloat(f.PriceChangePercent, 64)
		snap.Prices[strings.ToUpper(f.Symbol)] = price
		snap.Pcts[strings.ToUpper(f.Symbol)] = pct
	}
	return snap, nil
}

func baseSymbolFromPair(pair string) string {
	up := strings.ToUpper(strings.TrimSpace(pair))
	if up == "" {
		return ""
	}
	// 先处理期货后缀 USD_PERP（必须在现货后缀之前，避免误匹配）
	if strings.HasSuffix(up, "USD_PERP") {
		return strings.TrimSuffix(up, "USD_PERP")
	}
	// 再处理现货后缀
	for _, suf := range []string{"USDT", "USDC", "FDUSD", "BUSD", "TUSD", "USDP"} {
		if strings.HasSuffix(up, suf) {
			return strings.TrimSuffix(up, suf)
		}
	}
	return up
}

// ===== 回调告警（上一段 vs 当前段）=====
func (c *BinanceMarketCollector) maybeAlertPullback(kind string, prev, curr *snapshot) {
	if !c.alert.Enable {
		return
	}
	if prev == nil || curr == nil || len(prev.Prices) == 0 || len(curr.Prices) == 0 {
		return
	}
	if c.alert.PostmarkServerToken == "" || c.alert.From == "" || c.alert.ToCSV == "" {
		log.Printf("[market_scanner] [%s] alert skipped: postmark config incomplete", kind)
		return
	}

	type item struct {
		Symbol     string
		PrevPrice  float64
		CurrPrice  float64
		DropPct    float64 // 0.12 = 12%
		PrevBucket time.Time
		CurrBucket time.Time
	}
	var hits []item

	for sym, pPrev := range prev.Prices {
		if pPrev <= 0 {
			continue
		}
		if pCurr, ok := curr.Prices[sym]; ok && pCurr >= 0 {
			drop := (pPrev - pCurr) / pPrev
			if drop >= c.alert.Threshold {
				hits = append(hits, item{
					Symbol:     sym,
					PrevPrice:  pPrev,
					CurrPrice:  pCurr,
					DropPct:    drop,
					PrevBucket: prev.BucketUTC,
					CurrBucket: curr.BucketUTC,
				})
			}
		}
	}

	if len(hits) == 0 {
		return
	}

	// 组织邮件内容
	subject := fmt.Sprintf("[Market] %s 回调告警 ≥ %.2f%%  (%s → %s)",
		strings.ToUpper(kind),
		c.alert.Threshold*100,
		prev.BucketUTC.In(c.loc).Format("01-02 15:04"),
		curr.BucketUTC.In(c.loc).Format("01-02 15:04"),
	)

	var b strings.Builder
	b.WriteString("时间段：")
	b.WriteString(prev.BucketUTC.In(c.loc).Format("2006-01-02 15:04"))
	b.WriteString(" → ")
	b.WriteString(curr.BucketUTC.In(c.loc).Format("2006-01-02 15:04"))
	b.WriteString("  (时区：")
	b.WriteString(c.loc.String())
	b.WriteString(")\n阈值：")
	b.WriteString(fmt.Sprintf("%.2f%%\n\n", c.alert.Threshold*100))
	b.WriteString("符号\t上段价\t本段价\t回调\n")
	for _, it := range hits {
		b.WriteString(fmt.Sprintf("%s\t%g\t%g\t%.2f%%\n",
			it.Symbol, it.PrevPrice, it.CurrPrice, it.DropPct*100))
	}

	if err := c.sendEmailPostmarkSDK(subject, b.String()); err != nil {
		log.Printf("[market_scanner] [%s] alert send failed: %v", kind, err)
	} else {
		log.Printf("[market_scanner] [%s] alert sent via Postmark SDK: %d items", kind, len(hits))
	}
}

// ===== Postmark SDK 发送 =====

func (c *BinanceMarketCollector) sendEmailPostmarkSDK(subject, body string) error {
	client := postmark.NewClient(c.alert.PostmarkServerToken, c.alert.PostmarkAccountToken)
	email := postmark.Email{
		From:     c.alert.From,
		To:       c.alert.ToCSV, // 逗号分隔即可
		Subject:  subject,
		TextBody: body,
		//MessageStream: c.alert.PostmarkStream, // e.g. "outbound"
	}
	resp, err := client.SendEmail(email)
	if err != nil {
		return err
	}
	if resp.ErrorCode != 0 {
		return fmt.Errorf("postmark error %d: %s", resp.ErrorCode, resp.Message)
	}
	return nil
}
