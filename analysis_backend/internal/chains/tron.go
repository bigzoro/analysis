package chains

import (
	"analysis/internal/config"
	"analysis/internal/flow"
	"analysis/internal/models"
	"analysis/internal/netutil"
	"analysis/internal/util"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"
)

func TronTRC20(ctx context.Context, addr string, trc20 []config.TokenTRC20) (map[string]*big.Int, map[string]int, error) {
	var t struct {
		TokenBalances []struct{ TokenId, Balance string } `json:"trc20token_balances"`
	}
	if err := netutil.GetJSON(ctx, "https://apilist.tronscanapi.com/api/account/tokens?address="+addr, &t); err != nil {
		return nil, nil, err
	}
	want := map[string]config.TokenTRC20{}
	for _, x := range trc20 {
		want[strings.ToLower(x.Contract)] = x
	}
	out := map[string]*big.Int{}
	dec := map[string]int{}
	for _, it := range t.TokenBalances {
		if tok, ok := want[strings.ToLower(it.TokenId)]; ok {
			if !util.IsAllowed(tok.Symbol) {
				continue
			}
			z := new(big.Int)
			z.SetString(it.Balance, 10)
			out[tok.Symbol] = z
			dec[tok.Symbol] = 6
		}
	}
	return out, dec, nil
}

func TronTRC20Flows(ctx context.Context, addr, contract string, start, end time.Time, symbol string, wb models.WeeklyBucket, db models.DailyBucket) error {
	startTS := start.Unix() * 1000
	endTS := end.Unix() * 1000
	offset := 0
	for {
		u := fmt.Sprintf("https://apilist.tronscanapi.com/api/token_trc20/transfers?limit=50&start=%d&contract=%s&address=%s", offset, contract, addr)
		var r struct {
			TokenTransfers []struct {
				BlockTS int64  `json:"block_ts"`
				From    string `json:"from_address"`
				To      string `json:"to_address"`
				Value   string `json:"quant"`
			} `json:"token_transfers"`
		}
		if err := netutil.GetJSON(ctx, u, &r); err != nil {
			return err
		}
		if len(r.TokenTransfers) == 0 {
			break
		}
		for _, t := range r.TokenTransfers {
			if t.BlockTS < startTS {
				return nil
			}
			if t.BlockTS > endTS {
				continue
			}
			amt := new(big.Int)
			amt.SetString(t.Value, 10)
			dec := util.Pow10(6)
			when := time.UnixMilli(t.BlockTS).UTC()
			if strings.EqualFold(t.To, addr) {
				q := new(big.Float).Quo(new(big.Float).SetInt(amt), dec)
				flow.AddWeekly(wb, symbol, when, true, q)
				flow.AddDaily(db, symbol, when, true, q)
			} else if strings.EqualFold(t.From, addr) {
				q := new(big.Float).Quo(new(big.Float).SetInt(amt), dec)
				flow.AddWeekly(wb, symbol, when, false, q)
				flow.AddDaily(db, symbol, when, false, q)
			}
		}
		offset += 50
	}
	return nil
}
