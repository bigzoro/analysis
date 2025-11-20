package main

import (
	"analysis/internal/addr"
	"analysis/internal/config"
	"analysis/internal/models"
	"analysis/internal/netutil"
	"analysis/internal/util"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var transferTopic = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

type rpcReq struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}
type rpcResp struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// -------- BTC (Esplora) minimal --------
type btcTx struct {
	Txid   string    `json:"txid"`
	Vin    []btcVin  `json:"vin"`
	Vout   []btcVout `json:"vout"`
	Status struct {
		BlockTime int64 `json:"block_time"`
	} `json:"status"`
}
type btcVin struct {
	Prevout *btcVout `json:"prevout,omitempty"`
}
type btcVout struct {
	Value               int64  `json:"value"`
	ScriptPubKeyAddress string `json:"scriptpubkey_address"`
}

// -------- Solana 解析结果 --------
type solTransfer struct {
	isSOL       bool
	mint        string
	decimals    int
	amountDec   string
	source      string
	destination string
}

/*************** HTTP Client ***************/
var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 32,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second, // 增加 TLS 握手超时
		DisableCompression:  false,
		DisableKeepAlives:   false, // 保持连接复用
	},
	Timeout: 60 * time.Second, // 增加总超时时间到 60 秒
}

/*************** main ***************/
func main() {
	cfgPath := flag.String("config", "config.yaml", "config file")
	only := flag.String("only", "BTC,ETH,SOL,USDC,USDT", "symbols to include")
	//only := flag.String("only", "BTC,ETH,SOL,USDC,USDT", "symbols to include")
	//only := flag.String("only", "BTC,ETH,SOL,USDC,USDT,BNB,XRP,ADA,DOGE,TON", "symbols to include")
	//only := flag.String("only", "BNB,XRP,ADA,DOGE,TON", "symbols to include")
	apiBase := flag.String("api", "http://localhost:8010", "api base for ingest")
	entityArg := flag.String("entity", "", "only this entity (optional)")

	// PoR
	zipBinance := flag.String("zip-binance", "wallet_address_20250801.zip", "Binance PoR zip file")
	binanceEntity := flag.String("binance-entity", "binance", "entity tag for binance")
	binanceIncludeDeposit := flag.Bool("binance-include-deposit", false, "include deposit addresses")
	okxPOR := flag.String("okx-por", "okx_por_202507042112.csv.zip", "OKX PoR zip/csv")
	okxEntity := flag.String("okx-entity", "okx", "entity tag for okx")
	okxIncludeDeposit := flag.Bool("okx-include-deposit", true, "include OKX deposit addresses (if any)")
	okxIncludeStaking := flag.Bool("okx-include-staking", false, "include OKX ETH staking addresses")

	// 起始/轮询
	startFrom := flag.Int64("start-block", -5, "start block if no cursor (EVM: latest-4, BTC: latest-1, Solana: latest-200)")
	poll := flag.Duration("poll", 4*time.Second, "poll interval")

	// 过滤链
	excludeChainsFlag := flag.String("exclude-chains", "bsc,arbitrum,polygon,base", "comma/space separated chains to exclude, e.g. 'bsc, arbitrum'")

	// Solana 限速/退避
	solRPS := flag.Float64("sol-rps", 8, "Solana per-endpoint target requests per second (approx; <=0 to disable pacing)")
	sol429Cooldown := flag.Duration("sol-429-cooldown", 8*time.Second, "initial cooldown for HTTP 429 backoff (exponential)")

	// 日志
	verbose := flag.Bool("v", true, "verbose logging")
	logEvery := flag.Int("log-every", 200, "log progress every N blocks/slots")

	flag.Parse()
	util.SetAllowed(*only)

	logv := func(format string, args ...any) {
		if *verbose {
			log.Printf(format, args...)
		}
	}
	rangeStr := func(a, b uint64) string { return fmt.Sprintf("%d-%d", a, b) }
	summarize := func(evts []models.Event) (minT, maxT time.Time, byCoin map[string]int) {
		byCoin = map[string]int{}
		for i, e := range evts {
			if i == 0 || e.TS.Before(minT) {
				minT = e.TS
			}
			if i == 0 || e.TS.After(maxT) {
				maxT = e.TS
			}
			byCoin[e.Coin]++
		}
		return
	}

	// 配置
	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)
	config.ApplyProxy(&cfg)

	excludeSet := map[string]bool{}
	if s := strings.TrimSpace(*excludeChainsFlag); s != "" {
		for _, x := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			if x = strings.ToLower(strings.TrimSpace(x)); x != "" {
				excludeSet[x] = true
			}
		}
	}
	if len(excludeSet) > 0 {
		var xs []string
		for k := range excludeSet {
			xs = append(xs, k)
		}
		sort.Strings(xs)
		log.Printf("[exclude] chains: %v", xs)
	}

	// 地址来源
	rows := addr.RowsFromConfig(cfg)
	if *zipBinance != "" {
		rs, err := addr.RowsFromBinancePORZip(*zipBinance, *binanceEntity, *binanceIncludeDeposit)
		if err != nil {
			log.Fatalf("read binance por: %v", err)
		}
		rows = append(rows, rs...)
	}
	if *okxPOR != "" {
		rs, err := addr.RowsFromOKXPOR(*okxPOR, *okxEntity, *okxIncludeDeposit, *okxIncludeStaking)
		if err != nil {
			log.Fatalf("read okx por: %v", err)
		}
		rows = append(rows, rs...)
	}
	if len(rows) == 0 {
		log.Fatal("no addresses from config/zip")
	}

	chainCfg := config.BuildChainCfg(&cfg)

	// 分组：EVM/Bitcoin/Solana
	addressesEVM := map[string]map[string][]string{} // chain -> entity -> addrs
	addressesBTC := map[string][]string{}
	addressesSOL := map[string][]string{}
	for _, r := range rows {
		ent := r.Entity
		if ent == "" {
			ent = "unknown"
		}
		ch := strings.ToLower(strings.TrimSpace(r.Chain))
		if excludeSet[ch] {
			continue
		}
		switch ch {
		case "bitcoin", "btc":
			addressesBTC[ent] = append(addressesBTC[ent], strings.TrimSpace(r.Address))
		case "solana", "sol":
			addressesSOL[ent] = append(addressesSOL[ent], strings.TrimSpace(r.Address))
		default:
			if _, ok := addressesEVM[ch]; !ok {
				addressesEVM[ch] = map[string][]string{}
			}
			addressesEVM[ch][ent] = append(addressesEVM[ch][ent], strings.ToLower(strings.TrimSpace(r.Address)))
		}
	}
	logv("[init] entities evm=%d chains, btc=%d entities, sol=%d entities", len(addressesEVM), len(addressesBTC), len(addressesSOL))

	/*************** EVM 初始化（支持多 RPC + fallback） ***************/
	type evmChain struct {
		name             string
		rpcList          []string
		rpcIdx           int
		contractToSym    map[string]string // lowerAddr -> SYMBOL
		decimalsCache    map[string]int
		addressesByEnt   map[string][]string
		includeNativeETH bool // 仅以太坊主网
		nativeSymbol     string
	}
	evmChains := []evmChain{}

	for ch, ents := range addressesEVM {
		cc, ok := chainCfg[ch]
		if !ok || strings.TrimSpace(cc.RPC) == "" {
			log.Printf("[warn] chain %s not configured or no rpc, skip", ch)
			continue
		}
		rpcs := parseRPCList(cc.RPC) // <= 关键：解析多端点
		if len(rpcs) == 0 {
			log.Printf("[warn] chain %s rpc list is empty after parsing", ch)
			continue
		}
		contractToSymbol := map[string]string{}
		for _, t := range cc.ERC20 {
			addr := strings.ToLower(strings.TrimSpace(t.Address))
			if addr == "" {
				continue
			}
			contractToSymbol[addr] = strings.ToUpper(strings.TrimSpace(t.Symbol))
		}
		evmChains = append(evmChains, evmChain{
			name:             ch,
			rpcList:          rpcs,
			rpcIdx:           0,
			contractToSym:    contractToSymbol,
			decimalsCache:    map[string]int{},
			addressesByEnt:   ents,
			includeNativeETH: ch == "ethereum",
			nativeSymbol:     evmNativeSymbol(ch),
		})
	}
	for _, ec := range evmChains {
		logv("[init] evm %s rpc=%v tokens=%v", ec.name, ec.rpcList, keys(ec.contractToSym))
	}

	/*************** BTC 初始化 ***************/
	var btcAPIs []string
	var btcAPIIdx int
	if len(addressesBTC) > 0 && !excludeSet["bitcoin"] && !excludeSet["btc"] {
		btc, ok := chainCfg["bitcoin"]
		if !ok || strings.TrimSpace(btc.Esplora) == "" {
			log.Fatal("chains.bitcoin.esplora not configured")
		}
		btcAPIs = parseEsploraEndpoints(btc.Esplora)
		if len(btcAPIs) == 0 {
			log.Fatal("chains.bitcoin.esplora resolved empty endpoints")
		}
		logv("[init] bitcoin esplora=%v", btcAPIs)
	}

	/*************** Solana 初始化 ***************/
	var solRPCs []string
	var solRPCIdx int
	var mintToSymbol = map[string]string{}
	if len(addressesSOL) > 0 && !excludeSet["solana"] && !excludeSet["sol"] {
		sol, ok := chainCfg["solana"]
		if !ok || strings.TrimSpace(sol.RPC) == "" {
			log.Fatal("chains.solana.rpc not configured")
		}
		solRPCs = parseRPCList(sol.RPC)
		if len(solRPCs) == 0 {
			log.Fatal("chains.solana.rpc empty after parsing")
		}
		for _, sp := range sol.SPL {
			m := strings.ToLower(strings.TrimSpace(sp.Mint))
			if m != "" {
				mintToSymbol[m] = strings.ToUpper(strings.TrimSpace(sp.Symbol))
			}
		}
		logv("[init] solana rpc=%v spl=%v", solRPCs, keys(mintToSymbol))
	}

	/*************** RPC helpers ***************/
	// —— EVM：多端点轮询封装（带重试和指数退避）
	evmPost := func(ctx context.Context, ec *evmChain, method string, params []interface{}, out *rpcResp) error {
		var lastErr error
		maxRetries := len(ec.rpcList) * 2 // 每个端点最多重试 2 次
		baseDelay := 200 * time.Millisecond
		maxDelay := 5 * time.Second

		for attempt := 0; attempt < maxRetries; attempt++ {
			idx := (ec.rpcIdx + attempt) % len(ec.rpcList)
			base := strings.TrimRight(ec.rpcList[idx], "/")

			// 创建带超时的 context（每次重试都重新创建）
			rpcCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			err := postRPC(rpcCtx, base, method, params, out)
			cancel()

			if err == nil {
				ec.rpcIdx = idx
				return nil
			}

			lastErr = fmt.Errorf("rpc %s by %s => %w", method, base, err)

			// 判断错误类型
			errStr := strings.ToLower(err.Error())
			isEOF := strings.Contains(errStr, "eof") || strings.Contains(errStr, "connection reset")
			isTimeout := strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded")
			isNetworkErr := isEOF || isTimeout || strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host")

			if isNetworkErr {
				// 网络错误：指数退避
				delay := baseDelay
				if attempt > 0 {
					retryLevel := attempt / len(ec.rpcList)
					if retryLevel > 4 {
						retryLevel = 4
					}
					// 指数退避：baseDelay * 2^retryLevel
					multiplier := 1 << uint(retryLevel)
					delay = baseDelay * time.Duration(multiplier)
					if delay > maxDelay {
						delay = maxDelay
					}
				}
				log.Printf("[%s] %v (retry in %v, attempt %d/%d)", ec.name, lastErr, delay, attempt+1, maxRetries)
				time.Sleep(delay)
			} else {
				// 其他错误（如 RPC 错误）：短暂延迟后切换端点
				log.Printf("[%s] %v (switching endpoint)", ec.name, lastErr)
				time.Sleep(300 * time.Millisecond)
			}
		}
		return fmt.Errorf("[%s] all endpoints failed, last error: %w", ec.name, lastErr)
	}
	evmLatestBlock := func(ctx context.Context, ec *evmChain) (uint64, error) {
		var out rpcResp
		if err := evmPost(ctx, ec, "eth_blockNumber", []interface{}{}, &out); err != nil {
			return 0, err
		}
		var x string
		if err := json.Unmarshal(out.Result, &x); err != nil {
			return 0, err
		}
		n, _ := new(big.Int).SetString(strings.TrimPrefix(x, "0x"), 16)
		return n.Uint64(), nil
	}
	evmGetBlock := func(ctx context.Context, ec *evmChain, num uint64) (map[string]any, error) {
		var out rpcResp
		if err := evmPost(ctx, ec, "eth_getBlockByNumber", []interface{}{fmt.Sprintf("0x%x", num), true}, &out); err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(out.Result, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	evmGetLogs := func(ctx context.Context, ec *evmChain, from, to uint64, contract string, fromAddrs, toAddrs []string) ([]map[string]any, error) {
		p := map[string]any{
			"fromBlock": fmt.Sprintf("0x%x", from),
			"toBlock":   fmt.Sprintf("0x%x", to),
			"address":   contract,
			"topics": []any{
				transferTopic.Hex(),
				orTopic(fromAddrs),
				orTopic(toAddrs),
			},
		}
		var out rpcResp
		if err := evmPost(ctx, ec, "eth_getLogs", []interface{}{p}, &out); err != nil {
			return nil, err
		}
		if len(out.Result) == 0 || string(out.Result) == "null" {
			return nil, fmt.Errorf("eth_getLogs via %s: empty result", strings.Join(ec.rpcList, ","))
		}
		var arr []map[string]any
		if err := json.Unmarshal(out.Result, &arr); err != nil {
			return nil, err
		}
		return arr, nil
	}
	evmDecimals := func(ctx context.Context, ec *evmChain, contract string) (int, error) {
		if v, ok := ec.decimalsCache[contract]; ok {
			return v, nil
		}
		call := map[string]any{"to": contract, "data": "0x313ce567"}
		var out rpcResp
		if err := evmPost(ctx, ec, "eth_call", []interface{}{call, "latest"}, &out); err != nil {
			return 18, err
		}
		var x string
		if err := json.Unmarshal(out.Result, &x); err != nil {
			return 18, err
		}
		n, _ := new(big.Int).SetString(strings.TrimPrefix(x, "0x"), 16)
		d := int(n.Int64())
		if d <= 0 || d > 36 {
			d = 18
		}
		ec.decimalsCache[contract] = d
		return d, nil
	}

	// —— BTC（带 fallback）
	btcGetText := func(ctx context.Context, path string) (string, error) {
		var lastErr error
		for i := 0; i < len(btcAPIs); i++ {
			idx := (btcAPIIdx + i) % len(btcAPIs)
			base := strings.TrimRight(btcAPIs[idx], "/")
			url := base + path
			txt, err := getText(ctx, url)
			if err == nil {
				btcAPIIdx = idx
				return txt, nil
			}
			lastErr = err
			log.Printf("[btc] fallback %s: %v", url, err)
		}
		return "", lastErr
	}
	btcGetJSON := func(ctx context.Context, path string, out any) error {
		var lastErr error
		for i := 0; i < len(btcAPIs); i++ {
			idx := (btcAPIIdx + i) % len(btcAPIs)
			base := strings.TrimRight(btcAPIs[idx], "/")
			url := base + path
			if err := getJSON(ctx, url, out); err == nil {
				btcAPIIdx = idx
				return nil
			} else {
				lastErr = err
				log.Printf("[btc] fallback %s: %v", url, err)
			}
		}
		return lastErr
	}
	btcTipHeight := func(ctx context.Context) (uint64, error) {
		txt, err := btcGetText(ctx, "/blocks/tip/height")
		if err != nil {
			return 0, err
		}
		n := new(big.Int)
		n.SetString(strings.TrimSpace(txt), 10)
		return n.Uint64(), nil
	}
	btcBlockHash := func(ctx context.Context, height uint64) (string, error) {
		return btcGetText(ctx, fmt.Sprintf("/block-height/%d", height))
	}
	btcBlockTxs := func(ctx context.Context, blockHash string) ([]btcTx, error) {
		const pageSize = 25
		var all []btcTx
		offset := 0
		for {
			path := fmt.Sprintf("/block/%s/txs", blockHash)
			if offset > 0 {
				path = fmt.Sprintf("/block/%s/txs/%d", blockHash, offset)
			}
			var arr []btcTx
			if err := btcGetJSON(ctx, path, &arr); err != nil {
				if offset == 0 {
					return all, err
				}
				break
			}
			if len(arr) == 0 {
				break
			}
			all = append(all, arr...)
			if len(arr) < pageSize {
				break
			}
			offset += pageSize
			if offset > 20000 {
				break
			}
		}
		return all, nil
	}

	/*************** Solana（多端点 fallback + 限速 + 封禁/冷却 + 降级/退避） ***************/
	var (
		// 端点健康状态
		solBan                = map[string]time.Time{}     // endpoint -> unbanTime (403/-32052)
		solCooldown           = map[string]time.Time{}     // endpoint -> coolUntil (429)
		solCooldownDur        = map[string]time.Duration{} // endpoint -> 当前退避时长（指数退避）
		solLastCall           = map[string]time.Time{}     // endpoint -> 上次调用时间（限速）
		solLastDegradeAttempt = map[string]time.Time{}     // endpoint -> 最近一次降级尝试时间
	)

	// 限速：每端点目标 RPS（<=0 表示不节流）
	var solMinInterval time.Duration
	if *solRPS > 0 {
		solMinInterval = time.Duration(float64(time.Second) / *solRPS)
		if solMinInterval < 10*time.Millisecond {
			solMinInterval = 10 * time.Millisecond
		}
	} else {
		solMinInterval = 0
	}
	// 429 初始冷却
	baseCooldown := *sol429Cooldown
	if baseCooldown <= 0 {
		baseCooldown = 8 * time.Second
	}
	// 降级策略：只在冷却即将结束时（<=1s）尝试一次；同端点降级尝试最小间隔 6s
	degradeHeadstart := 1 * time.Second
	degradeMinGap := 6 * time.Second
	maxBackoff := 60 * time.Second

	errNoSolEndpoint := fmt.Errorf("no solana endpoint available (banned/cooling)")

	solHealth := func(now time.Time) string {
		var okList, coolList, banList []string
		for _, ep := range solRPCs {
			ep = strings.TrimRight(ep, "/")
			if until, banned := solBan[ep]; banned && now.Before(until) {
				banList = append(banList, fmt.Sprintf("%s(until=%s)", ep, until.UTC().Format(time.RFC3339)))
				continue
			}
			if until, cooling := solCooldown[ep]; cooling && now.Before(until) {
				coolList = append(coolList, fmt.Sprintf("%s(until=%s)", ep, until.UTC().Format(time.RFC3339)))
				continue
			}
			okList = append(okList, ep)
		}
		return fmt.Sprintf("healthy=%v cooling=%v banned=%v", okList, coolList, banList)
	}
	shouldBanSol := func(err error) (bool, time.Duration, string) {
		if err == nil {
			return false, 0, ""
		}
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "403") ||
			strings.Contains(s, "forbidden") ||
			strings.Contains(s, "not allowed") ||
			strings.Contains(s, "api key") ||
			strings.Contains(s, "apikey") ||
			strings.Contains(s, "-32052") {
			return true, 30 * time.Minute, "permission/plan"
		}
		return false, 0, ""
	}
	is429 := func(err error) bool {
		if err == nil {
			return false
		}
		s := strings.ToLower(err.Error())
		return strings.Contains(s, "429") || strings.Contains(s, "too many requests")
	}
	isBanned := func(ep string, now time.Time) (bool, time.Time) {
		ep = strings.TrimRight(ep, "/")
		if until, ok := solBan[ep]; ok && now.Before(until) {
			return true, until
		}
		return false, time.Time{}
	}
	isCooling := func(ep string, now time.Time) (bool, time.Time) {
		ep = strings.TrimRight(ep, "/")
		if until, ok := solCooldown[ep]; ok && now.Before(until) {
			return true, until
		}
		return false, time.Time{}
	}
	waitRate := func(ep string) {
		ep = strings.TrimRight(ep, "/")
		if solMinInterval <= 0 {
			return
		}
		if last, ok := solLastCall[ep]; ok {
			sleep := last.Add(solMinInterval).Sub(time.Now())
			if sleep > 0 && sleep < 5*time.Second {
				time.Sleep(sleep)
			}
		}
	}

	// 端点选择：优先健康端点；否则仅在冷却即将结束（<=1s）且距上次降级>=6s 的端点上进行一次“降级尝试”
	chooseSolEndpoint := func(now time.Time) (base string, degraded bool) {
		// 健康优先
		for i := 0; i < len(solRPCs); i++ {
			idx := (solRPCIdx + i) % len(solRPCs)
			cand := strings.TrimRight(solRPCs[idx], "/")
			if banned, _ := isBanned(cand, now); banned {
				continue
			}
			if cooling, _ := isCooling(cand, now); cooling {
				continue
			}
			solRPCIdx = idx
			return cand, false
		}
		// 降级：找最早解冻且满足“即将结束冷却 + 降级间隔”的端点
		type cand struct {
			ep    string
			until time.Time
		}
		var cds []cand
		for _, ep := range solRPCs {
			base := strings.TrimRight(ep, "/")
			if banned, _ := isBanned(base, now); banned {
				continue
			}
			if until, cooling := solCooldown[base]; cooling && now.Before(until) {
				remain := until.Sub(now)
				if remain <= degradeHeadstart {
					if last, ok := solLastDegradeAttempt[base]; !ok || now.Sub(last) >= degradeMinGap {
						cds = append(cds, cand{ep: base, until: until})
					}
				}
			}
		}
		if len(cds) == 0 {
			return "", false
		}
		sort.Slice(cds, func(i, j int) bool { return cds[i].until.Before(cds[j].until) })
		return cds[0].ep, true
	}

	solPost := func(ctx context.Context, method string, params []any, out *rpcResp) error {
		if len(solRPCs) == 0 {
			return fmt.Errorf("no solana rpc configured")
		}
		now := time.Now()
		var tried bool
		var lastErr error

		for attempt := 0; attempt < len(solRPCs); attempt++ {
			base, degraded := chooseSolEndpoint(now)
			if base == "" {
				break
			}
			tried = true
			if degraded {
				log.Printf("[solana] DEGRADE try cooldown endpoint %s", base)
				solLastDegradeAttempt[base] = now
			}

			waitRate(base)
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			err := postRPC(cctx, base, method, params, out)
			cancel()
			solLastCall[base] = time.Now()

			if err == nil {
				// 成功：清理冷却记录
				delete(solCooldown, base)
				delete(solCooldownDur, base)
				return nil
			}

			// 分类处理错误
			lastErr = fmt.Errorf("rpc %s by %s => %w", method, base, err)

			// 403/权限问题：长时间封禁
			if ban, dur, why := shouldBanSol(err); ban {
				until := time.Now().Add(dur)
				solBan[base] = until
				log.Printf("[solana] BAN %s for %s reason=%s err=%v", base, dur, why, err)
			} else if is429(err) {
				// 429：指数退避
				cur := solCooldownDur[base]
				if cur <= 0 {
					cur = baseCooldown
				} else {
					cur = cur * 2
					if cur > maxBackoff {
						cur = maxBackoff
					}
				}
				solCooldownDur[base] = cur
				until := time.Now().Add(cur)
				solCooldown[base] = until
				log.Printf("[solana] COOL %s for %s reason=429 err=%v", base, cur, err)

				// 若本次是降级尝试，避免在同一次调用里继续循环降级；交给上层下一轮再来
				if degraded {
					break
				}
			} else {
				// 其他错误：仅记录失败（网络、超时等）
				log.Printf("[solana] fail %s: %v", base, err)
			}

			// 下一轮尝试其它端点
			time.Sleep(200 * time.Millisecond)
			now = time.Now()
		}

		if !tried {
			log.Printf("[solana] all endpoints skipped (%s)", solHealth(time.Now()))
			return errNoSolEndpoint
		}
		log.Printf("[solana] all endpoints failed: %v", lastErr)
		log.Printf("[solana] health: %s", solHealth(time.Now()))
		return lastErr
	}
	solLatestSlot := func(ctx context.Context) (uint64, error) {
		var out rpcResp
		if err := solPost(ctx, "getSlot", []any{}, &out); err != nil {
			return 0, err
		}
		if len(out.Result) == 0 || string(out.Result) == "null" {
			return 0, fmt.Errorf("getSlot empty result")
		}
		var n uint64
		if err := json.Unmarshal(out.Result, &n); err != nil {
			return 0, err
		}
		return n, nil
	}
	solGetBlock := func(ctx context.Context, slot uint64) (map[string]any, error) {
		opts := map[string]any{
			"encoding":                       "jsonParsed",
			"transactionDetails":             "full",
			"rewards":                        false,
			"maxSupportedTransactionVersion": 0,
			"commitment":                     "confirmed",
		}
		var out rpcResp
		if err := solPost(ctx, "getBlock", []any{slot, opts}, &out); err != nil {
			return nil, err
		}
		if len(out.Result) == 0 || string(out.Result) == "null" {
			return nil, fmt.Errorf("block %d not available", slot)
		}
		var blk map[string]any
		if err := json.Unmarshal(out.Result, &blk); err != nil {
			return nil, err
		}
		return blk, nil
	}

	/*************** 读取游标 ***************/
	ctx := context.Background()

	// EVM
	cursorEVM := map[string]map[string]uint64{} // chain->entity->block
	for i := range evmChains {
		ec := &evmChains[i]
		latest, err := evmLatestBlock(ctx, ec)
		if err != nil {
			log.Printf("[cursor] %s latest error: %v", ec.name, err)
			continue
		}
		if cursorEVM[ec.name] == nil {
			cursorEVM[ec.name] = map[string]uint64{}
		}
		for entity := range ec.addressesByEnt {
			if *entityArg != "" && !strings.EqualFold(*entityArg, entity) {
				continue
			}
			var curResp struct {
				Block uint64 `json:"block"`
			}
			url := fmt.Sprintf("%s/sync/cursor?entity=%s&chain=%s", strings.TrimRight(*apiBase, "/"), entity, ec.name)
			if err := getJSON(ctx, url, &curResp); err != nil || curResp.Block == 0 {
				if *startFrom >= 0 {
					cursorEVM[ec.name][entity] = uint64(*startFrom)
				} else if latest > 4 {
					cursorEVM[ec.name][entity] = latest - 4
				} else {
					cursorEVM[ec.name][entity] = latest
				}
			} else {
				cursorEVM[ec.name][entity] = curResp.Block
			}
			log.Printf("[cursor] %s entity=%s start=%d (latest=%d)", ec.name, entity, cursorEVM[ec.name][entity], latest)
		}
	}

	// BTC
	cursorBTC := map[string]uint64{}
	if len(addressesBTC) > 0 {
		latest, err := btcTipHeight(ctx)
		if err != nil {
			log.Printf("[cursor] btc latest error: %v", err)
		} else {
			for entity := range addressesBTC {
				if *entityArg != "" && !strings.EqualFold(*entityArg, entity) {
					continue
				}
				var curResp struct {
					Block uint64 `json:"block"`
				}
				url := fmt.Sprintf("%s/sync/cursor?entity=%s&chain=bitcoin", strings.TrimRight(*apiBase, "/"), entity)
				if err := getJSON(ctx, url, &curResp); err != nil || curResp.Block == 0 {
					if *startFrom >= 0 {
						cursorBTC[entity] = uint64(*startFrom)
					} else if latest > 1 {
						cursorBTC[entity] = latest - 1
					} else {
						cursorBTC[entity] = latest
					}
				} else {
					cursorBTC[entity] = curResp.Block
				}
				log.Printf("[cursor] btc entity=%s start=%d (latest=%d)", entity, cursorBTC[entity], latest)
			}
		}
	}

	// SOL
	cursorSOL := map[string]uint64{}
	if len(addressesSOL) > 0 {
		latest, err := solLatestSlot(ctx)
		if err != nil {
			log.Printf("[cursor] sol latest error: %v", err)
		} else {
			for entity := range addressesSOL {
				if *entityArg != "" && !strings.EqualFold(*entityArg, entity) {
					continue
				}
				var curResp struct {
					Block uint64 `json:"block"`
				}
				url := fmt.Sprintf("%s/sync/cursor?entity=%s&chain=solana", strings.TrimRight(*apiBase, "/"), entity)
				if err := getJSON(ctx, url, &curResp); err != nil || curResp.Block == 0 {
					if *startFrom >= 0 {
						cursorSOL[entity] = uint64(*startFrom)
					} else if latest > 200 {
						cursorSOL[entity] = latest - 200
					} else {
						cursorSOL[entity] = latest
					}
				} else {
					cursorSOL[entity] = curResp.Block
				}
				log.Printf("[cursor] sol entity=%s start=%d (latest=%d)", entity, cursorSOL[entity], latest)
			}
		}
	}

	/*************** 扫描循环 ***************/
	for {
		progressed := false

		// —— EVM 各链
		for i := range evmChains {
			ec := &evmChains[i]
			for entity, addrs := range ec.addressesByEnt {
				if *entityArg != "" && !strings.EqualFold(*entityArg, entity) {
					continue
				}
				latest, err := evmLatestBlock(ctx, ec)
				if err != nil {
					log.Printf("[latest] %s error: %v", ec.name, err)
					continue
				}
				cur := cursorEVM[ec.name][entity]
				if cur >= latest {
					continue
				}
				to := cur + 500
				if to > latest {
					to = latest
				}
				addrSet := toSetLower(addrs)
				events := make([]models.Event, 0, 256)
				scanStart := time.Now()
				logv("[%s] entity=%s window=%s latest=%d addrs=%d", ec.name, entity, rangeStr(cur, to), latest, len(addrs))

				// ETH 原生（仅以太坊主网）
				//if ec.includeNativeETH && util.IsAllowed("ETH") {
				if ec.nativeSymbol != "" && util.IsAllowed(ec.nativeSymbol) {
					for b := cur; b <= to; b++ {
						if (b-cur)%uint64(*logEvery) == 0 {
							logv("[%s] block %d/%d (+%d)", ec.name, b, to, b-cur)
						}
						blk, err := evmGetBlock(ctx, ec, b)
						if err != nil {
							log.Printf("[%s] getBlock %d: %v", ec.name, b, err)
							continue
						}
						txs, _ := blk["transactions"].([]any)
						ts := parseBlockTime(blk)
						for _, it := range txs {
							tx := it.(map[string]any)
							from := strings.ToLower(str(tx["from"]))
							toA := strings.ToLower(str(tx["to"]))
							valHex := str(tx["value"])
							if valHex == "" {
								continue
							}
							wei := new(big.Int)
							_, _ = wei.SetString(strings.TrimPrefix(valHex, "0x"), 16)
							if wei.Sign() == 0 {
								continue
							}
							if addrSet[from] || (toA != "" && addrSet[toA]) {
								amt := toDecimal(wei, 18)
								dir := "in"
								target := toA
								if addrSet[from] && !addrSet[toA] {
									dir = "out"
									target = from
								}
								events = append(events, models.Event{
									Entity: entity, Chain: ec.name, Coin: ec.nativeSymbol, Direction: dir, Amount: amt,
									TS: ts, TxID: str(tx["hash"]), From: from, To: toA, Address: target, LogIndex: -1,
								})
							}
						}
					}
				}

				// ERC20（按配置）
				if len(ec.contractToSym) > 0 {
					const chunk = 100 // 可按 RPC 限制调整
					// 地址唯一化
					addrList := uniqueLower(addrs)
					addrSet := toSetLower(addrList)

					// 记录去重：txHash#logIndex
					seen := map[string]struct{}{}

					for contract, symbol := range ec.contractToSym {
						if !util.IsAllowed(symbol) {
							continue
						}
						decimals, derr := evmDecimals(ctx, ec, contract)
						if derr != nil {
							log.Printf("[%s] decimals %s: %v (use 18)", ec.name, contract, derr)
							decimals = 18
						}

						// 1) fromChunk：topics = [Transfer, OR(from), nil]
						for i := 0; i < len(addrList); i += chunk {
							end := i + chunk
							if end > len(addrList) {
								end = len(addrList)
							}
							fc := addrList[i:end]

							if *verbose {
								log.Printf("[%s] getLogs %s %s %s fromChunk %d/%d size=%d",
									ec.name, symbol, contract, rangeStr(cur, to),
									(i/chunk)+1, (len(addrList)+chunk-1)/chunk, len(fc))
							}

							logsArr, err := evmGetLogs(ctx, ec, cur, to, contract, fc, nil)
							if err != nil {
								log.Printf("[%s] getLogs(from) %s %s %s: %v", ec.name, symbol, contract, rangeStr(cur, to), err)
								continue
							}
							for _, lg := range logsArr {
								topics, _ := lg["topics"].([]any)
								if len(topics) < 3 {
									// 容错：部分节点会返回异常日志
									continue
								}
								from := topicAddr(topics[1])
								toA := topicAddr(topics[2])

								// 只要 from 在监控集即可
								if !addrSet[from] {
									continue
								}

								val := new(big.Int)
								_, _ = val.SetString(strings.TrimPrefix(str(lg["data"]), "0x"), 16)
								if val.Sign() == 0 {
									continue
								}
								amt := toDecimal(val, decimals)
								hash := str(lg["transactionHash"])
								lidx := int(hexToUint64(str(lg["logIndex"])))
								key := hash + "#" + fmt.Sprint(lidx)
								if _, ok := seen[key]; ok {
									continue
								}
								seen[key] = struct{}{}

								blkTs := time.Now().UTC()
								if n := hexToUint64(str(lg["blockNumber"])); n > 0 {
									if blk, err := evmGetBlock(ctx, ec, n); err == nil {
										blkTs = parseBlockTime(blk)
									}
								}

								// 如果 to 不在集，就判定为 out；否则记为 in
								dir := "in"
								target := toA
								if !addrSet[toA] {
									dir = "out"
									target = from
								}

								events = append(events, models.Event{
									Entity: entity, Chain: ec.name, Coin: symbol, Direction: dir, Amount: amt,
									TS: blkTs, TxID: hash, From: from, To: toA, Address: target, LogIndex: lidx,
								})
							}
						}

						// 2) toChunk：topics = [Transfer, nil, OR(to)]
						for i := 0; i < len(addrList); i += chunk {
							end := i + chunk
							if end > len(addrList) {
								end = len(addrList)
							}
							tc := addrList[i:end]

							if *verbose {
								log.Printf("[%s] getLogs %s %s %s toChunk %d/%d size=%d",
									ec.name, symbol, contract, rangeStr(cur, to),
									(i/chunk)+1, (len(addrList)+chunk-1)/chunk, len(tc))
							}

							logsArr, err := evmGetLogs(ctx, ec, cur, to, contract, nil, tc)
							if err != nil {
								log.Printf("[%s] getLogs(to) %s %s %s: %v", ec.name, symbol, contract, rangeStr(cur, to), err)
								continue
							}
							for _, lg := range logsArr {
								topics, _ := lg["topics"].([]any)
								if len(topics) < 3 {
									continue
								}
								from := topicAddr(topics[1])
								toA := topicAddr(topics[2])

								// 只要 to 在监控集即可
								if !addrSet[toA] {
									continue
								}

								val := new(big.Int)
								_, _ = val.SetString(strings.TrimPrefix(str(lg["data"]), "0x"), 16)
								if val.Sign() == 0 {
									continue
								}
								amt := toDecimal(val, decimals)
								hash := str(lg["transactionHash"])
								lidx := int(hexToUint64(str(lg["logIndex"])))
								key := hash + "#" + fmt.Sprint(lidx)
								if _, ok := seen[key]; ok {
									continue
								} // 避免与 fromChunk 重复
								seen[key] = struct{}{}

								blkTs := time.Now().UTC()
								if n := hexToUint64(str(lg["blockNumber"])); n > 0 {
									if blk, err := evmGetBlock(ctx, ec, n); err == nil {
										blkTs = parseBlockTime(blk)
									}
								}

								// to 命中 => in（from 也在集的情况前面已去重）
								dir := "in"
								target := toA
								if addrSet[from] && !addrSet[toA] {
									dir = "out"
									target = from
								}
								events = append(events, models.Event{
									Entity: entity, Chain: ec.name, Coin: symbol, Direction: dir, Amount: amt,
									TS: blkTs, TxID: hash, From: from, To: toA, Address: target, LogIndex: lidx,
								})
							}
						}
					}
				}

				minT, maxT, byCoin := summarize(events)
				if len(events) == 0 {
					logv("[%s] entity=%s no-events window=%s duration=%s", ec.name, entity, rangeStr(cur, to), time.Since(scanStart))
				} else {
					logv("[%s] entity=%s events=%d window=%s ts=[%s .. %s] byCoin=%v duration=%s",
						ec.name, entity, len(events), rangeStr(cur, to),
						minT.UTC().Format(time.RFC3339), maxT.UTC().Format(time.RFC3339), byCoin, time.Since(scanStart))
				}
				if len(events) > 0 {
					u := fmt.Sprintf("%s/ingest/events?entity=%s", strings.TrimRight(*apiBase, "/"), entity)
					var resp struct {
						OK    bool   `json:"ok"`
						Saved int    `json:"saved"`
						RunID string `json:"run_id"`
					}
					if err := netutil.PostJSON(context.Background(), u, events, &resp); err != nil {
						log.Printf("ingest error (%s): %v", ec.name, err)
					} else {
						log.Printf("ingest ok (%s): entity=%s saved=%d run_id=%s", ec.name, entity, resp.Saved, resp.RunID)
					}
				}
				next := to + 1
				if err := netutil.PostJSON(context.Background(),
					fmt.Sprintf("%s/sync/cursor?entity=%s&chain=%s", strings.TrimRight(*apiBase, "/"), entity, ec.name),
					map[string]uint64{"block": next}, &struct {
						OK bool `json:"ok"`
					}{},
				); err != nil {
					log.Printf("[cursor] set %s %s -> %d error: %v", ec.name, entity, next, err)
				} else {
					cursorEVM[ec.name][entity] = next
					progressed = true
				}
			}
		}

		// —— BTC
		if len(addressesBTC) > 0 {
			latest, err := btcTipHeight(ctx)
			if err != nil {
				log.Printf("[latest] btc error: %v", err)
			} else {
				for entity, addrs := range addressesBTC {
					if *entityArg != "" && !strings.EqualFold(*entityArg, entity) {
						continue
					}
					cur := cursorBTC[entity]
					if cur >= latest {
						continue
					}
					to := cur + 6
					if to > latest {
						to = latest
					}
					addrSetExact := toSetExact(addrs)
					addrSetLower := toSetLower(addrs)
					events := make([]models.Event, 0, 512)
					scanStart := time.Now()
					logv("[bitcoin] entity=%s window=%s latest=%d addrs=%d", entity, rangeStr(cur, to), latest, len(addrs))
					for h := cur; h <= to; h++ {
						if (h-cur)%uint64(*logEvery) == 0 {
							logv("[bitcoin] height %d/%d (+%d)", h, to, h-cur)
						}
						bh, err := btcBlockHash(ctx, h)
						if err != nil || strings.TrimSpace(bh) == "" {
							log.Printf("[bitcoin] block hash %d: %v", h, err)
							continue
						}
						txs, err := btcBlockTxs(ctx, strings.TrimSpace(bh))
						if err != nil {
							log.Printf("[bitcoin] block txs %d: %v", h, err)
							continue
						}
						for _, tx := range txs {
							ts := time.Unix(tx.Status.BlockTime, 0).UTC()
							for i, vin := range tx.Vin {
								if vin.Prevout == nil {
									continue
								}
								addr := strings.TrimSpace(vin.Prevout.ScriptPubKeyAddress)
								if addr == "" || vin.Prevout.Value <= 0 {
									continue
								}
								if !(addrSetExact[addr] || addrSetLower[strings.ToLower(addr)]) {
									continue
								}
								amt := satsToDecimal(vin.Prevout.Value)
								toAddr := firstVoutAddr(tx.Vout)
								events = append(events, models.Event{
									Entity: entity, Chain: "bitcoin", Coin: "BTC", Direction: "out", Amount: amt,
									TS: ts, TxID: tx.Txid, From: addr, To: toAddr, Address: addr, LogIndex: -(i + 1),
								})
							}
							for i, vout := range tx.Vout {
								addr := strings.TrimSpace(vout.ScriptPubKeyAddress)
								if addr == "" || vout.Value <= 0 {
									continue
								}
								if !(addrSetExact[addr] || addrSetLower[strings.ToLower(addr)]) {
									continue
								}
								amt := satsToDecimal(vout.Value)
								fromAddr := firstVinAddr(tx.Vin)
								events = append(events, models.Event{
									Entity: entity, Chain: "bitcoin", Coin: "BTC", Direction: "in", Amount: amt,
									TS: ts, TxID: tx.Txid, From: fromAddr, To: addr, Address: addr, LogIndex: i,
								})
							}
						}
					}
					minT, maxT, byCoin := summarize(events)
					if len(events) == 0 {
						logv("[bitcoin] entity=%s no-events window=%s duration=%s", entity, rangeStr(cur, to), time.Since(scanStart))
					} else {
						logv("[bitcoin] entity=%s events=%d window=%s ts=[%s .. %s] byCoin=%v duration=%s",
							entity, len(events), rangeStr(cur, to),
							minT.UTC().Format(time.RFC3339), maxT.UTC().Format(time.RFC3339), byCoin, time.Since(scanStart))
					}
					if len(events) > 0 {
						u := fmt.Sprintf("%s/ingest/events?entity=%s", strings.TrimRight(*apiBase, "/"), entity)
						var resp struct {
							OK    bool   `json:"ok"`
							Saved int    `json:"saved"`
							RunID string `json:"run_id"`
						}
						if err := netutil.PostJSON(context.Background(), u, events, &resp); err != nil {
							log.Printf("ingest error (btc): %v", err)
						} else {
							log.Printf("ingest ok (btc): entity=%s saved=%d run_id=%s", entity, resp.Saved, resp.RunID)
						}
					}
					next := to + 1
					if err := netutil.PostJSON(context.Background(),
						fmt.Sprintf("%s/sync/cursor?entity=%s&chain=bitcoin", strings.TrimRight(*apiBase, "/"), entity),
						map[string]uint64{"block": next}, &struct {
							OK bool `json:"ok"`
						}{},
					); err != nil {
						log.Printf("[cursor] set BTC %s -> %d error: %v", entity, next, err)
					} else {
						cursorBTC[entity] = next
						progressed = true
					}
				}
			}
		}

		// —— Solana
		if len(addressesSOL) > 0 {
			latest, err := solLatestSlot(ctx)
			if err != nil {
				log.Printf("[latest] solana error: %v; %s", err, solHealth(time.Now()))
			} else {
				const step = 200
				for entity, addrs := range addressesSOL {
					if *entityArg != "" && !strings.EqualFold(*entityArg, entity) {
						continue
					}
					cur := cursorSOL[entity]
					if cur >= latest {
						continue
					}
					to := cur + step
					if to > latest {
						to = latest
					}
					addrSet := toSetExact(addrs)
					addrLower := toSetLower(addrs)
					events := make([]models.Event, 0, 256)
					logIndex := 0
					scanStart := time.Now()
					rpcInUse := ""
					if len(solRPCs) > 0 {
						rpcInUse = strings.TrimRight(solRPCs[solRPCIdx], "/")
					}
					logv("[solana] entity=%s window=%s latest=%d addrs=%d rpc=%s", entity, rangeStr(cur, to), latest, len(addrs), rpcInUse)

					for slot := cur; slot <= to; slot++ {
						if (slot-cur)%uint64(*logEvery) == 0 {
							logv("[solana] slot %d/%d (+%d)", slot, to, slot-cur)
						}
						blk, err := solGetBlock(ctx, slot)
						if err != nil {
							log.Printf("[solana] getBlock slot=%d rpc=%s err=%v", slot, rpcInUse, err)
							continue
						}
						blkt := time.Now().UTC()
						if v := blk["blockTime"]; v != nil {
							switch vv := v.(type) {
							case float64:
								blkt = time.Unix(int64(vv), 0).UTC()
							case int64:
								blkt = time.Unix(vv, 0).UTC()
							}
						}
						txs, _ := blk["transactions"].([]any)
						for _, ti := range txs {
							tx := ti.(map[string]any)
							sigs, _ := tx["transaction"].(map[string]any)["signatures"].([]any)
							var txid string
							if len(sigs) > 0 {
								txid = str(sigs[0])
							}

							// 指令解析
							trs := parseSolanaTransfers(tx)
							for _, tr := range trs {
								symbol := "SOL"
								if !tr.isSOL {
									symbol = mintToSymbol[strings.ToLower(tr.mint)]
									if symbol == "" {
										continue
									}
								}
								if !util.IsAllowed(symbol) {
									continue
								}
								hitOut := addrSet[tr.source] || addrLower[strings.ToLower(tr.source)]
								hitIn := addrSet[tr.destination] || addrLower[strings.ToLower(tr.destination)]
								if !(hitOut || hitIn) {
									continue
								}
								dir := "in"
								addr := tr.destination
								if hitOut && !hitIn {
									dir = "out"
									addr = tr.source
								}
								events = append(events, models.Event{
									Entity: entity, Chain: "solana", Coin: symbol, Direction: dir, Amount: tr.amountDec,
									TS: blkt, TxID: txid, From: tr.source, To: tr.destination, Address: addr, LogIndex: logIndex,
								})
								logIndex++
							}

							// 余额差兜底
							meta, _ := tx["meta"].(map[string]any)
							if meta == nil {
								continue
							}
							if util.IsAllowed("SOL") {
								if preB, ok := toInt64Slice(meta["preBalances"]); ok {
									if postB, ok2 := toInt64Slice(meta["postBalances"]); ok2 {
										msg := tx["transaction"].(map[string]any)["message"]
										var accountKeys []string
										switch ak := msg.(map[string]any)["accountKeys"].(type) {
										case []any:
											for _, k := range ak {
												switch kv := k.(type) {
												case string:
													accountKeys = append(accountKeys, kv)
												case map[string]any:
													accountKeys = append(accountKeys, str(kv["pubkey"]))
												}
											}
										}
										for i := 0; i < len(preB) && i < len(postB) && i < len(accountKeys); i++ {
											a := accountKeys[i]
											if !(addrSet[a] || addrLower[strings.ToLower(a)]) {
												continue
											}
											diff := postB[i] - preB[i]
											if diff == 0 {
												continue
											}
											amt := lamportsToSOL(diff)
											dir := "in"
											if diff < 0 {
												dir = "out"
											}
											events = append(events, models.Event{
												Entity: entity, Chain: "solana", Coin: "SOL", Direction: dir, Amount: amt,
												TS: blkt, TxID: txid, From: "", To: "", Address: a, LogIndex: logIndex,
											})
											logIndex++
										}
									}
								}
							}
							// SPL 余额差
							preTB, _ := meta["preTokenBalances"].([]any)
							postTB, _ := meta["postTokenBalances"].([]any)
							type tokenState struct {
								owner, mint, amount string
								decimals            int
							}
							preMap := map[int]tokenState{}
							postMap := map[int]tokenState{}
							for _, it := range preTB {
								m := it.(map[string]any)
								idx := intFromAny(m["accountIndex"])
								mint := strings.ToLower(str(m["mint"]))
								owner := str(m["owner"])
								ui, _ := m["uiTokenAmount"].(map[string]any)
								amt := str(ui["amount"])
								dec := intFromAny(ui["decimals"])
								preMap[idx] = tokenState{owner: owner, mint: mint, amount: amt, decimals: dec}
							}
							for _, it := range postTB {
								m := it.(map[string]any)
								idx := intFromAny(m["accountIndex"])
								mint := strings.ToLower(str(m["mint"]))
								owner := str(m["owner"])
								ui, _ := m["uiTokenAmount"].(map[string]any)
								amt := str(ui["amount"])
								dec := intFromAny(ui["decimals"])
								postMap[idx] = tokenState{owner: owner, mint: mint, amount: amt, decimals: dec}
							}
							for idx, pre := range preMap {
								post, ok := postMap[idx]
								if !ok || pre.mint != post.mint {
									continue
								}
								owner := post.owner
								if owner == "" {
									owner = pre.owner
								}
								if !(addrSet[owner] || addrLower[strings.ToLower(owner)]) {
									continue
								}
								dec := post.decimals
								if dec <= 0 {
									dec = pre.decimals
								}
								diff := bigIntSub(post.amount, pre.amount)
								if diff.Sign() == 0 {
									continue
								}
								sym := mintToSymbol[strings.ToLower(pre.mint)]
								if sym == "" || !util.IsAllowed(sym) {
									continue
								}
								amount := toDecimal(new(big.Int).Abs(diff), dec)
								dir := "in"
								if diff.Sign() < 0 {
									dir = "out"
								}
								events = append(events, models.Event{
									Entity: entity, Chain: "solana", Coin: sym, Direction: dir, Amount: amount,
									TS: blkt, TxID: txid, From: "", To: "", Address: owner, LogIndex: logIndex,
								})
								logIndex++
							}
						}
					}

					minT, maxT, byCoin := summarize(events)
					if len(events) == 0 {
						logv("[solana] entity=%s no-events window=%s duration=%s", entity, rangeStr(cur, to), time.Since(scanStart))
					} else {
						logv("[solana] entity=%s events=%d window=%s ts=[%s .. %s] byCoin=%v duration=%s",
							entity, len(events), rangeStr(cur, to),
							minT.UTC().Format(time.RFC3339), maxT.UTC().Format(time.RFC3339), byCoin, time.Since(scanStart))
					}
					if len(events) > 0 {
						u := fmt.Sprintf("%s/ingest/events?entity=%s", strings.TrimRight(*apiBase, "/"), entity)
						var resp struct {
							OK    bool   `json:"ok"`
							Saved int    `json:"saved"`
							RunID string `json:"run_id"`
						}
						if err := netutil.PostJSON(context.Background(), u, events, &resp); err != nil {
							log.Printf("ingest error (sol): %v", err)
						} else {
							log.Printf("ingest ok (sol): entity=%s saved=%d run_id=%s", entity, resp.Saved, resp.RunID)
						}
					}
					next := to + 1
					if err := netutil.PostJSON(context.Background(),
						fmt.Sprintf("%s/sync/cursor?entity=%s&chain=solana", strings.TrimRight(*apiBase, "/"), entity),
						map[string]uint64{"block": next}, &struct {
							OK bool `json:"ok"`
						}{},
					); err != nil {
						log.Printf("[cursor] set SOL %s -> %d error: %v", entity, next, err)
					} else {
						cursorSOL[entity] = next
						progressed = true
					}
				}
			}
		}

		if !progressed {
			logv("[idle] no chain progressed; sleep=%s", *poll)
			time.Sleep(*poll)
		}
	}
}

/*************** 工具函数 ***************/
func postRPC(ctx context.Context, url, method string, params []interface{}, out *rpcResp) error {
	body, _ := json.Marshal(rpcReq{Jsonrpc: "2.0", ID: 1, Method: method, Params: params})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request %s %s: %w", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "scanner/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		// 检查是否是 EOF 或连接错误
		errStr := err.Error()
		if strings.Contains(errStr, "EOF") || strings.Contains(errStr, "connection reset") {
			return fmt.Errorf("connection closed (EOF): %w", err)
		}
		return fmt.Errorf("do %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("rpc %s => %d: %s", method, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	// 检查响应体是否为空
	if resp.ContentLength == 0 {
		return fmt.Errorf("rpc %s: empty response body", method)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("rpc %s decode error: %w", method, err)
	}

	// 检查 RPC 错误
	if out.Error != nil {
		return fmt.Errorf("rpc %s error [%d]: %s", method, out.Error.Code, out.Error.Message)
	}

	return nil
}
func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("new get %s: %w", url, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("get %s => %d: %s", url, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
func getText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("new get %s: %w", url, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("get %s => %d: %s", url, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
func toSetLower(ss []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range ss {
		m[strings.ToLower(strings.TrimSpace(s))] = true
	}
	return m
}
func toSetExact(ss []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			m[s] = true
		}
	}
	return m
}
func sliceLower(ss []string, i, j int) []string {
	if i > len(ss) {
		return nil
	}
	if j > len(ss) {
		j = len(ss)
	}
	out := make([]string, 0, j-i)
	for _, s := range ss[i:j] {
		out = append(out, strings.ToLower(strings.TrimSpace(s)))
	}
	return out
}
func parseBlockTime(blk map[string]any) time.Time {
	tsHex, _ := blk["timestamp"].(string)
	if tsHex == "" {
		return time.Now().UTC()
	}
	n, _ := new(big.Int).SetString(strings.TrimPrefix(tsHex, "0x"), 16)
	return time.Unix(n.Int64(), 0).UTC()
}
func toDecimal(v *big.Int, decimals int) string {
	if decimals <= 0 {
		decimals = 18
	}
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	r := new(big.Rat).SetFrac(v, base)
	return r.FloatString(8)
}
func satsToDecimal(sats int64) string {
	if sats <= 0 {
		return "0"
	}
	v := new(big.Int).SetInt64(sats)
	return toDecimal(v, 8)
}
func lamportsToSOL(lam int64) string {
	neg := lam < 0
	if neg {
		lam = -lam
	}
	v := new(big.Int).SetInt64(lam)
	out := toDecimal(v, 9)
	if neg {
		return "-" + out
	}
	return out
}
func orTopic(addrs []string) any {
	if len(addrs) == 0 {
		return nil
	}
	arr := make([]string, 0, len(addrs))
	for _, a := range addrs {
		a = strings.ToLower(strings.TrimSpace(a))
		if strings.HasPrefix(a, "0x") {
			a = a[2:]
		}
		if len(a) > 64 {
			a = a[len(a)-64:]
		}
		pad := strings.Repeat("0", 64-len(a))
		arr = append(arr, "0x"+pad+a)
	}
	return arr
}
func topicAddr(topic any) string {
	s := str(topic)
	s = strings.Trim(s, "\"")
	if len(s) >= 66 {
		return "0x" + s[len(s)-40:]
	}
	return s
}
func hexToUint64(h string) uint64 {
	h = strings.Trim(h, "\"")
	if h == "" {
		return 0
	}
	n, _ := new(big.Int).SetString(strings.TrimPrefix(h, "0x"), 16)
	return n.Uint64()
}
func firstVoutAddr(vouts []btcVout) string {
	for _, v := range vouts {
		a := strings.TrimSpace(v.ScriptPubKeyAddress)
		if a != "" {
			return a
		}
	}
	return ""
}
func firstVinAddr(vins []btcVin) string {
	for _, vin := range vins {
		if vin.Prevout == nil {
			continue
		}
		a := strings.TrimSpace(vin.Prevout.ScriptPubKeyAddress)
		if a != "" {
			return a
		}
	}
	return ""
}
func parseEsploraEndpoints(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, sep := range []string{",", ";", "\n"} {
		s = strings.ReplaceAll(s, sep, " ")
	}
	fields := strings.Fields(s)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		u := strings.TrimRight(strings.TrimSpace(f), "/")
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}
func parseRPCList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, sep := range []string{",", ";", "\n"} {
		s = strings.ReplaceAll(s, sep, " ")
	}
	fields := strings.Fields(s)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		u := strings.TrimRight(strings.TrimSpace(f), "/")
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}
func toInt64Slice(v any) ([]int64, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]int64, 0, len(arr))
	for _, x := range arr {
		switch t := x.(type) {
		case float64:
			out = append(out, int64(t))
		case int64:
			out = append(out, t)
		case int:
			out = append(out, int64(t))
		case string:
			if n, ok := new(big.Int).SetString(t, 10); ok {
				out = append(out, n.Int64())
			} else {
				out = append(out, 0)
			}
		default:
			out = append(out, 0)
		}
	}
	return out, true
}
func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		if n, ok := new(big.Int).SetString(t, 10); ok {
			return int(n.Int64())
		}
		return 0
	default:
		return 0
	}
}
func bigIntSub(aStr, bStr string) *big.Int {
	a := new(big.Int)
	b := new(big.Int)
	a.SetString(aStr, 10)
	b.SetString(bStr, 10)
	return new(big.Int).Sub(a, b)
}
func parseSolanaTransfers(tx map[string]any) []solTransfer {
	var out []solTransfer
	var parseInstrList func([]any)
	parseInstrList = func(list []any) {
		for _, it := range list {
			inst, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if sub, ok := inst["instructions"].([]any); ok {
				parseInstrList(sub)
				continue
			}
			prog := strings.ToLower(str(inst["program"]))
			parsed, _ := inst["parsed"].(map[string]any)
			if parsed == nil {
				continue
			}
			typ := strings.ToLower(str(parsed["type"]))
			info, _ := parsed["info"].(map[string]any)
			if info == nil {
				continue
			}

			switch {
			case prog == "system" && typ == "transfer":
				src := str(info["source"])
				dst := str(info["destination"])
				lam := int64(0)
				switch v := info["lamports"].(type) {
				case float64:
					lam = int64(v)
				case int64:
					lam = v
				case string:
					if n, ok := new(big.Int).SetString(v, 10); ok {
						lam = n.Int64()
					}
				}
				if lam <= 0 {
					continue
				}
				out = append(out, solTransfer{
					isSOL: true, decimals: 9, amountDec: toDecimal(big.NewInt(lam), 9),
					source: src, destination: dst,
				})
			case prog == "spl-token" && (typ == "transfer" || typ == "transferchecked"):
				src := str(info["source"])
				dst := str(info["destination"])
				mint := strings.ToLower(str(info["mint"]))
				dec := 0
				var amountDec string
				if ta, ok := info["tokenAmount"].(map[string]any); ok {
					amountDec = str(ta["uiAmountString"])
					if amountDec == "" {
						raw := str(ta["amount"])
						dec = intFromAny(ta["decimals"])
						if n, ok := new(big.Int).SetString(raw, 10); ok {
							if dec <= 0 {
								dec = 6
							}
							amountDec = toDecimal(n, dec)
						}
					}
					if dec == 0 {
						dec = intFromAny(ta["decimals"])
					}
				}
				if amountDec == "" {
					raw := str(info["amount"])
					if n, ok := new(big.Int).SetString(raw, 10); ok {
						if dec == 0 {
							dec = 6
						}
						amountDec = toDecimal(n, dec)
					}
				}
				out = append(out, solTransfer{
					isSOL: false, mint: mint, decimals: dec, amountDec: amountDec,
					source: src, destination: dst,
				})
			}
		}
	}
	if txObj, ok := tx["transaction"].(map[string]any); ok {
		if msg, ok := txObj["message"].(map[string]any); ok {
			if list, ok := msg["instructions"].([]any); ok {
				parseInstrList(list)
			}
		}
	}
	if meta, ok := tx["meta"].(map[string]any); ok {
		if inners, ok := meta["innerInstructions"].([]any); ok {
			parseInstrList(inners)
		}
	}
	return out
}
func keys[M ~map[string]string](m M) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniqueLower(ss []string) []string {
	m := map[string]struct{}{}
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" {
			continue
		}
		if _, ok := m[k]; ok {
			continue
		}
		m[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func evmNativeSymbol(chain string) string {
	c := strings.ToLower(strings.TrimSpace(chain))
	switch c {
	case "ethereum", "eth":
		return "ETH"
	case "bsc", "bnb", "bnbchain", "bnbsmartchain":
		return "BNB"
	case "polygon", "matic":
		return "MATIC"
	case "avalanche", "avax", "avaxc", "avalanchec":
		return "AVAX"
	case "fantom", "ftm":
		return "FTM"
	case "op", "optimism":
		return "ETH"
	case "arbitrum", "arb", "arbitrumone":
		return "ETH"
	case "base":
		return "ETH"
	default:
		return ""
	}
}
