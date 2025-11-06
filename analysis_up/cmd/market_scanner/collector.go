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

	// 上一段快照（用于回调比较）
	prevSpot    *snapshot
	prevFutures *snapshot

	// 内部统一存 UTC 的下次时间
	nextUTC time.Time
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
		log.Printf(
			"[market_scanner] tick skipped, now_local=%s now_utc=%s < next_local=%s next_utc=%s",
			nowUTC.In(c.loc).Format(time.RFC3339),
			nowUTC.Format(time.RFC3339),
			c.nextUTC.In(c.loc).Format(time.RFC3339),
			c.nextUTC.Format(time.RFC3339),
		)
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
		url = "https://fapi.binance.com/fapi/v1/ticker/24hr"
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

	// 3) USDT 过滤（保持你当前口径）
	filtered := make([]binance24hTicker, 0, len(tickers))
	for _, t := range tickers {
		if strings.HasSuffix(strings.ToUpper(t.Symbol), "USDT") {
			filtered = append(filtered, t)
		}
	}
	log.Printf("[market_scanner] [%s] filtered USDT tickers: %d", kind, len(filtered))

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
		Kind      string           `json:"kind"`
		Bucket    string           `json:"bucket"`
		FetchedAt string           `json:"fetched_at"`
		Items     []map[string]any `json:"items"`
	}{
		Kind:      kind,
		Bucket:    bucketUTC.Format(time.RFC3339),
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     make([]map[string]any, 0, len(filtered)),
	}
	for _, f := range filtered {
		pct, _ := strconv.ParseFloat(f.PriceChangePercent, 64)
		payload.Items = append(payload.Items, map[string]any{
			"symbol":               f.Symbol,
			"last_price":           f.LastPrice,
			"volume":               f.Volume,
			"price_change_percent": pct,
		})
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
