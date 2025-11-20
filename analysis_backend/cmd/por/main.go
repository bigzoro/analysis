package main

import (
	"analysis/internal/addr"
	"analysis/internal/chains"
	"analysis/internal/collector"
	"analysis/internal/config"
	"analysis/internal/db"
	"analysis/internal/models"
	"analysis/internal/price"
	"analysis/internal/util"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

func main() {
	// ---------- Flags ----------
	cfgPath := flag.String("config", "config.yaml", "config file")
	only := flag.String("only", "BTC, ETH, USDT, USDC, SOL", "symbols to include")

	// Binance PoR ZIP
	zipBinance := flag.String("zip-binance", "wallet_address_20250901.zip", "Binance PoR zip file")
	binanceEntity := flag.String("binance-entity", "binance", "entity name")
	binanceIncludeDeposit := flag.Bool("binance-include-deposit", false, "include deposit addresses")

	// OKX PoR
	okxPOR := flag.String("okx-por", "", "path to OKX PoR zip/csv (may contain multiple csv files)")
	okxEntity := flag.String("okx-entity", "okx", "entity name for OKX")
	okxIncludeDeposit := flag.Bool("okx-include-deposit", true, "include OKX deposit addresses from POR")
	okxIncludeStaking := flag.Bool("okx-include-staking", false, "include OKX staking addresses from POR")

	// Weekly / Daily
	weeks := flag.Int("weeks", 2, "weeks look back")
	fromDate := flag.String("from", "", "weekly start date YYYY-MM-DD (optional)")
	withWeekly := flag.Bool("weekly", false, "compute weekly flows")

	withDaily := flag.Bool("daily", false, "compute daily flows")
	tzName := flag.String("tz", "Asia/Taipei", "timezone for daily bucketing")
	dailyDate := flag.String("daily-date", "", "specific day YYYY-MM-DD (optional)")

	// ETH native via etherscan
	etherscanKey := flag.String("etherscan-key", "", "optional Etherscan API key for ETH native flows")

	flag.Parse()

	// ---------- Log: start ----------
	startTs := time.Now()
	log.Printf("[por] start at %s", startTs.Format(time.RFC3339))
	log.Printf("[por] using config=%s", *cfgPath)
	log.Printf("[por] only symbols=%s", *only)
	log.Printf("[por] binance zip=%s entity=%s include_deposit=%v", *zipBinance, *binanceEntity, *binanceIncludeDeposit)
	log.Printf("[por] okx por=%s entity=%s include_deposit=%v include_staking=%v", *okxPOR, *okxEntity, *okxIncludeDeposit, *okxIncludeStaking)
	log.Printf("[por] weekly=%v weeks=%d from=%s daily=%v tz=%s dailyDate=%s", *withWeekly, *weeks, *fromDate, *withDaily, *tzName, *dailyDate)

	util.SetAllowed(*only)

	// ---------- Load config & proxy ----------
	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)
	config.ApplyProxy(&cfg)
	chainsCfg := config.BuildChainCfg(&cfg)

	if cfg.Proxy.Enable {
		log.Printf("[proxy] enabled: all=%s http=%s https=%s no=%s", cfg.Proxy.All, cfg.Proxy.HTTP, cfg.Proxy.HTTPS, cfg.Proxy.No)
	} else {
		log.Printf("[proxy] disabled")
	}
	for name, cc := range chainsCfg {
		log.Printf("[chain] %s type=%s rpc=%s esplora=%s erc20=%d spl=%d trc20=%d",
			name, cc.Type, cc.RPC, cc.Esplora, len(cc.ERC20), len(cc.SPL), len(cc.TRC20))
	}

	// ---------- Prices ----------
	if !cfg.Pricing.Enable {
		cfg.Pricing.Enable = true
		cfg.Pricing.CoinGeckoEndpoint = "https://api.coingecko.com/api/v3/simple/price"
		cfg.Pricing.Map = map[string]string{
			"BTC": "bitcoin", "ETH": "ethereum", "SOL": "solana", "USDT": "tether", "USDC": "usd-coin",
		}
	}
	px, _ := price.FetchPrices(context.Background(), cfg, []string{"BTC", "ETH", "SOL", "USDT", "USDC"})
	if bs, err := json.Marshal(px); err == nil {
		log.Printf("[price] fetched: %s", string(bs))
	}

	// ---------- Address sources ----------
	rows := addr.RowsFromConfig(cfg)
	log.Printf("[addr] from config: %d rows", len(rows))

	if *zipBinance != "" {
		rs, err := addr.RowsFromBinancePORZip(*zipBinance, *binanceEntity, *binanceIncludeDeposit)
		if err != nil {
			panic(err)
		}
		rows = append(rows, rs...)
		log.Printf("[addr] +binance por: %d rows (total=%d)", len(rs), len(rows))
	}
	if *okxPOR != "" {
		orows, err := addr.RowsFromOKXPOR(*okxPOR, *okxEntity, *okxIncludeDeposit, *okxIncludeStaking)
		if err != nil {
			log.Fatalf("parse okx por(%s): %v", *okxPOR, err)
		}
		rows = append(rows, orows...)
		log.Printf("addr: +%d rows from okx por", len(orows))
		log.Printf("[addr] +okx por: %d rows (total=%d)", len(orows), len(rows))
	}

	// ---------- Group by entity ----------
	group := map[string][]models.AddressRow{}
	for _, r := range rows {
		ent := r.Entity
		if ent == "" {
			ent = "unknown"
		}
		group[ent] = append(group[ent], r)
	}
	log.Printf("[addr] grouped entities: %d", len(group))
	for k, v := range group {
		log.Printf("[addr] entity=%s addrs=%d", k, len(v))
	}

	// ---------- Time windows ----------
	weeklyEnd := time.Now().UTC()
	weeklyStart := weeklyEnd.AddDate(0, 0, -7*(*weeks))
	if *fromDate != "" {
		if t, err := time.Parse("2006-01-02", *fromDate); err == nil {
			weeklyStart = t.UTC()
		}
	}
	loc, err := time.LoadLocation(*tzName)
	if err != nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	var dailyStart, dailyEnd time.Time
	if *dailyDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", *dailyDate, loc); err == nil {
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
			dailyStart = day.In(time.UTC)
			dailyEnd = day.Add(24 * time.Hour).In(time.UTC)
		} else {
			now := time.Now().In(loc)
			day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
			dailyStart = day.In(time.UTC)
			dailyEnd = day.Add(24 * time.Hour).In(time.UTC)
		}
	} else {
		now := time.Now().In(loc)
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		dailyStart = day.In(time.UTC)
		dailyEnd = day.Add(24 * time.Hour).In(time.UTC)
	}
	log.Printf("[window] weekly [%s, %s)", weeklyStart.Format(time.RFC3339), weeklyEnd.Format(time.RFC3339))
	log.Printf("[window] daily   [%s, %s) tz=%s", dailyStart.Format(time.RFC3339), dailyEnd.Format(time.RFC3339), loc.String())

	// ---------- DB ----------
	gdb, err := db.OpenMySQL(db.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  cfg.Database.Automigrate,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("[db] opened. automigrate=%v maxOpen=%d maxIdle=%d", cfg.Database.Automigrate, cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)

	runID := uuid.NewString()
	asOf := time.Now().UTC()
	log.Printf("[run] run_id=%s as_of=%s", runID, asOf.Format(time.RFC3339))

	// ---------- Output summary ----------
	type outSummary struct {
		Portfolios    []models.Portfolio    `json:"portfolios"`
		WeeklyResults []models.WeeklyResult `json:"weeklyResults"`
		DailyResults  []models.DailyResult  `json:"dailyResults"`
	}
	sum := outSummary{}

	// ---------- Process per entity ----------
	for ent, rs := range group {
		log.Printf("processing entity=%s addrs=%d ...", ent, len(rs))

		// 1) Portfolio snapshot
		if p, err := collector.ComputePortfolio(context.Background(), ent, rs, chainsCfg, px); err != nil {
			log.Printf("compute portfolio %s: %v", ent, err)
		} else {
			if err := db.SaveAll(gdb, runID, asOf, []models.Portfolio{p}, nil, nil); err != nil {
				log.Printf("     (portfolio, entity=%s) error: %v", ent, err)
			} else {
				log.Printf("✔ flushed portfolio entity=%s", ent)
				sum.Portfolios = append(sum.Portfolios, p)
			}
		}

		// 2) Weekly flows  —— 注意：WeeklyBucket 是 map，值传递
		if *withWeekly {
			wb := models.WeeklyBucket{}
			for i, r := range rs {
				fmt.Printf("b%v", i)
				switch r.Chain {
				case "bitcoin":
					if util.IsAllowed("BTC") && chainsCfg["bitcoin"].Esplora != "" {
						_ = chains.BTCFlows(context.Background(), chainsCfg["bitcoin"].Esplora, r.Address, weeklyStart, weeklyEnd, wb, nil)
					}
				case "solana":
					if util.IsAllowed("SOL") && chainsCfg["solana"].RPC != "" {
						_ = chains.SolFlowsSOL(context.Background(), chainsCfg["solana"].RPC, r.Address, weeklyStart, weeklyEnd, wb, nil)
					}
					for _, t := range chainsCfg["solana"].SPL {
						if util.IsAllowed(t.Symbol) {
							_ = chains.SolFlowsSPL(context.Background(), chainsCfg["solana"].RPC, r.Address, t.Mint, t.Symbol, weeklyStart, weeklyEnd, wb, nil)
						}
					}
				case "tron":
					for _, t := range chainsCfg["tron"].TRC20 {
						if util.IsAllowed(t.Symbol) {
							_ = chains.TronTRC20Flows(context.Background(), r.Address, t.Contract, weeklyStart, weeklyEnd, t.Symbol, wb, nil)
						}
					}
				default: // EVM-like
					cc := chainsCfg[r.Chain]
					owner := r.EVM()
					for _, tok := range cc.ERC20 {
						if util.IsAllowed(tok.Symbol) && (tok.Symbol == "USDT" || tok.Symbol == "USDC") {
							_ = chains.EVMERC20Flows(context.Background(), cc.RPC, tok, owner, weeklyStart, weeklyEnd, wb, nil)
						}
					}
					if r.Chain == "ethereum" && *etherscanKey != "" && util.IsAllowed("ETH") {
						_ = chains.ETHNativeFlowsEtherscan(context.Background(), *etherscanKey, cc.RPC, r.Address, weeklyStart, weeklyEnd, wb, nil)
					}
				}
			}

			if len(wb) > 0 {
				wres := models.WeeklyResult{Entity: ent, Data: wb}
				if err := db.SaveAll(gdb, runID, asOf, nil, []models.WeeklyResult{wres}, nil); err != nil {
					log.Printf("     (weekly, entity=%s) error: %v", ent, err)
				} else {
					log.Printf("✔ flushed weekly entity=%s", ent)
					sum.WeeklyResults = append(sum.WeeklyResults, wres)
				}
			}
		}

		// 3) Daily flows —— 注意：DailyBucket 是 map，值传递
		if *withDaily {
			dbkt := models.DailyBucket{}
			for i, r := range rs {
				fmt.Printf("c%v", i)
				switch r.Chain {
				case "bitcoin":
					if util.IsAllowed("BTC") && chainsCfg["bitcoin"].Esplora != "" {
						_ = chains.BTCFlows(context.Background(), chainsCfg["bitcoin"].Esplora, r.Address, dailyStart, dailyEnd, nil, dbkt)
					}
				case "solana":
					if util.IsAllowed("SOL") && chainsCfg["solana"].RPC != "" {
						_ = chains.SolFlowsSOL(context.Background(), chainsCfg["solana"].RPC, r.Address, dailyStart, dailyEnd, nil, dbkt)
					}
					for _, t := range chainsCfg["solana"].SPL {
						if util.IsAllowed(t.Symbol) {
							_ = chains.SolFlowsSPL(context.Background(), chainsCfg["solana"].RPC, r.Address, t.Mint, t.Symbol, dailyStart, dailyEnd, nil, dbkt)
						}
					}
				case "tron":
					for _, t := range chainsCfg["tron"].TRC20 {
						if util.IsAllowed(t.Symbol) {
							_ = chains.TronTRC20Flows(context.Background(), r.Address, t.Contract, dailyStart, dailyEnd, t.Symbol, nil, dbkt)
						}
					}
				default: // EVM-like
					cc := chainsCfg[r.Chain]
					owner := r.EVM()
					for _, tok := range cc.ERC20 {
						if util.IsAllowed(tok.Symbol) && (tok.Symbol == "USDT" || tok.Symbol == "USDC") {
							_ = chains.EVMERC20Flows(context.Background(), cc.RPC, tok, owner, dailyStart, dailyEnd, nil, dbkt)
						}
					}
					if r.Chain == "ethereum" && *etherscanKey != "" && util.IsAllowed("ETH") {
						_ = chains.ETHNativeFlowsEtherscan(context.Background(), *etherscanKey, cc.RPC, r.Address, dailyStart, dailyEnd, nil, dbkt)
					}
				}
			}
			if len(dbkt) > 0 {
				dres := models.DailyResult{Entity: ent, Data: dbkt}
				if err := db.SaveAll(gdb, runID, asOf, nil, nil, []models.DailyResult{dres}); err != nil {
					log.Printf("     (daily, entity=%s) error: %v", ent, err)
				} else {
					log.Printf("✔ flushed daily entity=%s", ent)
					sum.DailyResults = append(sum.DailyResults, dres)
				}
			}
		}
	}

	// ---------- Print summary ----------
	bs, _ := json.MarshalIndent(sum, "", "  ")
	fmt.Println(string(bs))
	log.Printf("[por] finished compute. duration=%s", time.Since(startTs))

	abs, _ := filepath.Abs(*cfgPath)
	fmt.Println("✔ 增量写入完成（余额/周度/日度分开保存），run_id =", runID)
	fmt.Println("✔ 使用配置：", abs)
}
