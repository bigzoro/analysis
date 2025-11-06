package chains

import (
	"analysis/internal/flow"
	"analysis/internal/models"
	"analysis/internal/netutil"
	"analysis/internal/util"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// 余额
func BTCAddressBalance(ctx context.Context, esplora, address string) (*big.Int, error) {
	type stat struct {
		FundedTxoSum int64 `json:"funded_txo_sum"`
		SpentTxoSum  int64 `json:"spent_txo_sum"`
	}
	type resp struct {
		ChainStats   stat `json:"chain_stats"`
		MempoolStats stat `json:"mempool_stats"`
	}
	ends := strings.Split(esplora, ",")
	var last error
	for _, ep := range ends {
		ep = strings.TrimSpace(ep)
		if ep == "" {
			continue
		}
		var r resp
		u := fmt.Sprintf("%s/address/%s", strings.TrimRight(ep, "/"), address)
		if err := netutil.GetJSON(ctx, u, &r); err != nil {
			last = err
			continue
		}
		bal := r.ChainStats.FundedTxoSum - r.ChainStats.SpentTxoSum +
			r.MempoolStats.FundedTxoSum - r.MempoolStats.SpentTxoSum
		return big.NewInt(bal), nil
	}
	if last == nil {
		last = fmt.Errorf("no esplora endpoint")
	}
	return nil, last
}

// 放在文件顶部（或包级）——复用连接/启用超时
var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 32,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false, // 使用默认 gzip
	},
	Timeout: 30 * time.Second, // 整体请求超时
}

// 抗限流分页 + 死循环保护
func btcListTxs(ctx context.Context, esplora, addr string, start, end time.Time) ([]map[string]any, error) {
	bases0 := strings.Split(esplora, ",")
	var bases []string
	for _, b := range bases0 {
		b = strings.TrimSpace(b)
		if b != "" {
			bases = append(bases, strings.TrimRight(b, "/"))
		}
	}
	if len(bases) == 0 {
		return nil, fmt.Errorf("no esplora endpoint configured")
	}

	var all []map[string]any
	lastSeen := ""
	rot := 0
	backoff := 300 * time.Millisecond
	maxBackoff := 5 * time.Second
	pageDelay := 250 * time.Millisecond

	const (
		maxConsecErrs = 40
		maxNoProgress = 3
		maxPages      = 10000
	)
	consecErrs := 0
	noProgress := 0
	pages := 0
	seenTails := make(map[string]struct{})

	doFetch := func(fullURL string, out *[]map[string]any) (int, error) {
		req, _ := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		req.Header.Set("User-Agent", "por-collector")
		req.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		status := resp.StatusCode
		if status == 429 || status >= 500 {
			// 限流/服务端错误：交由上层重试
			return status, fmt.Errorf("esplora %s => %d", fullURL, status)
		}
		if status/100 != 2 {
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return status, fmt.Errorf("GET %s => %d: %s", fullURL, status, strings.TrimSpace(string(snippet)))
		}

		var arr []map[string]any
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&arr); err != nil {
			return status, err
		}
		*out = arr
		return status, nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return all, err
		}
		if pages >= maxPages {
			return all, fmt.Errorf("esplora pagination exceeded maxPages=%d", maxPages)
		}

		var arr []map[string]any
		var err error
		tried := 0

		for tried < len(bases) {
			base := bases[(rot+tried)%len(bases)]
			var url string
			if lastSeen == "" {
				url = fmt.Sprintf("%s/address/%s/txs", base, addr)
			} else {
				url = fmt.Sprintf("%s/address/%s/txs/chain/%s", base, addr, lastSeen)
			}
			_, err = doFetch(url, &arr)
			if err == nil {
				rot = (rot + tried) % len(bases)
				break
			}
			tried++
		}

		if err != nil {
			consecErrs++
			if consecErrs >= maxConsecErrs {
				return all, fmt.Errorf("esplora consecutive errors reached %d: last err: %v", maxConsecErrs, err)
			}
			jitter := time.Duration(50+int(time.Now().UnixNano()%100)) * time.Millisecond
			time.Sleep(backoff + jitter)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		consecErrs = 0
		backoff = 300 * time.Millisecond

		if len(arr) == 0 {
			break
		}
		pages++

		for _, tx := range arr {
			st, _ := tx["status"].(map[string]any)
			ts := int64(0)
			if st != nil {
				if f, ok := st["block_time"].(float64); ok {
					ts = int64(f)
				}
			}
			if ts == 0 {
				continue
			}
			t := time.Unix(ts, 0).UTC()
			if t.After(end) {
				continue
			}
			if t.Before(start) {
				return all, nil
			}
			all = append(all, tx)
		}

		v, ok := arr[len(arr)-1]["txid"].(string)
		if !ok || v == "" {
			break
		}
		if v == lastSeen {
			noProgress++
			if noProgress >= maxNoProgress {
				return all, fmt.Errorf("esplora pagination stuck at %s", lastSeen)
			}
		} else {
			noProgress = 0
			if _, seen := seenTails[v]; seen {
				return all, fmt.Errorf("esplora pagination loop at %s", v)
			}
			seenTails[v] = struct{}{}
			lastSeen = v
		}

		time.Sleep(pageDelay)
	}
	return all, nil
}

func BTCFlows(ctx context.Context, esplora, addr string, start, end time.Time, wb models.WeeklyBucket, db models.DailyBucket) error {
	txs, err := btcListTxs(ctx, esplora, addr, start, end)
	if err != nil {
		return err
	}
	scale := util.Pow10(8)
	for _, tx := range txs {
		st, _ := tx["status"].(map[string]any)
		ts := int64(0)
		if st != nil {
			if f, ok := st["block_time"].(float64); ok {
				ts = int64(f)
			}
		}
		if ts == 0 {
			continue
		}
		tm := time.Unix(ts, 0).UTC()

		ins, _ := tx["vin"].([]any)
		outs, _ := tx["vout"].([]any)
		sent := big.NewInt(0)
		recv := big.NewInt(0)

		for _, v := range ins {
			m := v.(map[string]any)
			prevout, _ := m["prevout"].(map[string]any)
			if prevout == nil {
				continue
			}
			spk, _ := prevout["scriptpubkey_address"].(string)
			if strings.EqualFold(spk, addr) {
				if val, ok := prevout["value"].(float64); ok {
					sent.Add(sent, big.NewInt(int64(val)))
				}
			}
		}
		for _, v := range outs {
			m := v.(map[string]any)
			spk, _ := m["scriptpubkey_address"].(string)
			if strings.EqualFold(spk, addr) {
				if val, ok := m["value"].(float64); ok {
					recv.Add(recv, big.NewInt(int64(val)))
				}
			}
		}
		if recv.Sign() > 0 {
			q := new(big.Float).Quo(new(big.Float).SetInt(recv), scale)
			flow.AddWeekly(wb, "BTC", tm, true, q)
			flow.AddDaily(db, "BTC", tm, true, q)
		}
		if sent.Sign() > 0 {
			q := new(big.Float).Quo(new(big.Float).SetInt(sent), scale)
			flow.AddWeekly(wb, "BTC", tm, false, q)
			flow.AddDaily(db, "BTC", tm, false, q)
		}
	}
	return nil
}
