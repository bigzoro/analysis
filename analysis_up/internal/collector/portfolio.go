package collector

import (
	"analysis/internal/chains"
	"analysis/internal/config"
	"analysis/internal/models"
	"analysis/internal/util"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func ComputePortfolio(ctx context.Context, entity string, rows []models.AddressRow, chainsCfg map[string]config.ChainCfg, px map[string]float64) (models.Portfolio, error) {
	p := models.Portfolio{
		Entity:   entity,
		Holdings: map[string]models.Holding{},
		TS:       time.Now().UTC().Unix(),
	}

	seen := map[string]struct{}{}
	for i, r := range rows {
		fmt.Printf("a%v", i)
		k := strings.ToLower(r.Chain) + "|" + strings.ToLower(r.Address)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}

		switch r.Chain {
		case "bitcoin":
			if !util.IsAllowed("BTC") {
				continue
			}
			cc := chainsCfg["bitcoin"]
			if cc.Esplora == "" {
				continue
			}
			if bal, err := chains.BTCAddressBalance(ctx, cc.Esplora, r.Address); err == nil {
				util.AddHolding(p.Holdings, "bitcoin", "BTC", 8, bal, px)
			}

		case "solana":
			cc := chainsCfg["solana"]
			if cc.RPC == "" {
				continue
			}
			if util.IsAllowed("SOL") {
				if bal, err := chains.SolNative(ctx, cc.RPC, r.Address); err == nil {
					util.AddHolding(p.Holdings, "solana", "SOL", 9, bal, px)
				}
			}
			for _, t := range cc.SPL {
				if !util.IsAllowed(t.Symbol) {
					continue
				}
				if bal, dec, err := chains.SolSPL(ctx, cc.RPC, r.Address, t.Mint); err == nil && bal.Sign() > 0 {
					util.AddHolding(p.Holdings, "solana", t.Symbol, dec, bal, px)
				}
			}

		case "tron":
			cc := chainsCfg["tron"]
			if len(cc.TRC20) == 0 {
				continue
			}
			want := []config.TokenTRC20{}
			for _, t := range cc.TRC20 {
				if util.IsAllowed(t.Symbol) {
					want = append(want, t)
				}
			}
			if len(want) == 0 {
				continue
			}
			if m, dec, err := chains.TronTRC20(ctx, r.Address, want); err == nil {
				for sym, bal := range m {
					util.AddHolding(p.Holdings, "tron", sym, dec[sym], bal, px)
				}
			}

		default: // EVM
			cc := chainsCfg[r.Chain]
			if cc.RPC == "" {
				continue
			}
			ea := r.EVM()
			if util.IsAllowed("ETH") && (r.Chain == "ethereum" || r.Chain == "arbitrum" || r.Chain == "optimism" || r.Chain == "base") {
				if native, err := chains.EVMNativeBalance(ctx, cc.RPC, ea); err == nil && native.Sign() > 0 {
					util.AddHolding(p.Holdings, r.Chain, "ETH", 18, native, px)
				}
			}
			for _, t := range cc.ERC20 {
				if !util.IsAllowed(t.Symbol) {
					continue
				}
				if bal, dec, err := chains.EVMERC20Balance(ctx, cc.RPC, common.HexToAddress(t.Address), ea); err == nil && bal.Sign() > 0 {
					util.AddHolding(p.Holdings, r.Chain, t.Symbol, dec, bal, px)
				}
			}
		}
	}

	for _, h := range p.Holdings {
		p.TotalUSD += h.ValueUSD
	}
	return p, nil
}
