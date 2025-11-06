package chains

import (
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

func SolNative(ctx context.Context, rpcURL, addr string) (*big.Int, error) {
	req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "getBalance", "params": []any{addr}}
	var out struct {
		Result struct{ Value uint64 } `json:"result"`
		Error  *struct {
			Code    int
			Message string
		} `json:"error,omitempty"`
	}
	if err := netutil.PostJSON(ctx, rpcURL, req, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("solana rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	return big.NewInt(int64(out.Result.Value)), nil
}

func SolSPL(ctx context.Context, rpcURL, owner, mint string) (*big.Int, int, error) {
	req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "getTokenAccountsByOwner", "params": []any{owner, map[string]any{"mint": mint}, map[string]any{"encoding": "jsonParsed"}}}
	var out struct {
		Result struct {
			Value []struct {
				Account struct {
					Data struct {
						Parsed struct{ Info map[string]any } `json:"parsed"`
					} `json:"data"`
				} `json:"account"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := netutil.PostJSON(ctx, rpcURL, req, &out); err != nil {
		return nil, 0, err
	}
	var total big.Int
	dec := 0
	for _, v := range out.Result.Value {
		ta, _ := v.Account.Data.Parsed.Info["tokenAmount"].(map[string]any)
		if ta == nil {
			continue
		}
		if d, ok := ta["decimals"].(float64); ok {
			dec = int(d)
		}
		if s, ok := ta["amount"].(string); ok {
			z := new(big.Int)
			z.SetString(s, 10)
			total.Add(&total, z)
		}
	}
	return &total, dec, nil
}

func solListSigs(ctx context.Context, rpc, owner string, start time.Time) ([]string, error) {
	type sigRes struct {
		Result []struct {
			Signature string
			BlockTime *int64
		} `json:"result"`
	}
	var out []string
	before := ""
	for {
		req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "getSignaturesForAddress", "params": []any{owner, map[string]any{"before": before, "limit": 1000}}}
		var r sigRes
		if err := netutil.PostJSON(ctx, rpc, req, &r); err != nil {
			return nil, err
		}
		if len(r.Result) == 0 {
			break
		}
		stop := false
		for _, it := range r.Result {
			if it.BlockTime == nil {
				continue
			}
			tm := time.Unix(*it.BlockTime, 0).UTC()
			if tm.Before(start) {
				stop = true
				break
			}
			out = append(out, it.Signature)
		}
		before = r.Result[len(r.Result)-1].Signature
		if stop {
			break
		}
	}
	return out, nil
}

func SolFlowsSOL(ctx context.Context, rpc, owner string, start, end time.Time, wb models.WeeklyBucket, db models.DailyBucket) error {
	sigs, err := solListSigs(ctx, rpc, owner, start)
	if err != nil {
		return err
	}
	for _, sig := range sigs {
		req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "getTransaction", "params": []any{sig, map[string]any{"encoding": "json", "maxSupportedTransactionVersion": 0}}}
		var tx struct {
			Result struct {
				BlockTime   *int64                                      `json:"blockTime"`
				Meta        struct{ PreBalances, PostBalances []int64 } `json:"meta"`
				Transaction struct {
					Message struct{ AccountKeys []string }
				} `json:"transaction"`
			} `json:"result"`
		}
		if err := netutil.PostJSON(ctx, rpc, req, &tx); err != nil {
			continue
		}
		if tx.Result.BlockTime == nil {
			continue
		}
		tm := time.Unix(*tx.Result.BlockTime, 0).UTC()
		if tm.Before(start) || tm.After(end) {
			continue
		}
		idx := -1
		for i, k := range tx.Result.Transaction.Message.AccountKeys {
			if strings.EqualFold(k, owner) {
				idx = i
				break
			}
		}
		if idx < 0 || idx >= len(tx.Result.Meta.PreBalances) || idx >= len(tx.Result.Meta.PostBalances) {
			continue
		}
		delta := tx.Result.Meta.PostBalances[idx] - tx.Result.Meta.PreBalances[idx]
		amt := new(big.Float).Quo(big.NewFloat(float64(delta)), util.Pow10(9))
		if delta > 0 {
			flow.AddWeekly(wb, "SOL", tm, true, amt)
			flow.AddDaily(db, "SOL", tm, true, amt)
		}
		if delta < 0 {
			out := new(big.Float).Mul(amt, big.NewFloat(-1))
			flow.AddWeekly(wb, "SOL", tm, false, out)
			flow.AddDaily(db, "SOL", tm, false, out)
		}
	}
	return nil
}

func SolFlowsSPL(ctx context.Context, rpc, owner, mint, symbol string, start, end time.Time, wb models.WeeklyBucket, db models.DailyBucket) error {
	sigs, err := solListSigs(ctx, rpc, owner, start)
	if err != nil {
		return err
	}
	for _, sig := range sigs {
		req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "getTransaction", "params": []any{sig, map[string]any{"encoding": "json", "maxSupportedTransactionVersion": 0}}}
		var tx struct {
			Result struct {
				BlockTime *int64 `json:"blockTime"`
				Meta      struct {
					PreTokenBalances []struct {
						Owner         string `json:"owner"`
						Mint          string `json:"mint"`
						UITokenAmount struct {
							Amount   string `json:"amount"`
							Decimals int    `json:"decimals"`
						} `json:"uiTokenAmount"`
					} `json:"preTokenBalances"`
					PostTokenBalances []struct {
						Owner         string `json:"owner"`
						Mint          string `json:"mint"`
						UITokenAmount struct {
							Amount   string `json:"amount"`
							Decimals int    `json:"decimals"`
						} `json:"uiTokenAmount"`
					} `json:"postTokenBalances"`
				} `json:"meta"`
			} `json:"result"`
		}
		if err := netutil.PostJSON(ctx, rpc, req, &tx); err != nil {
			continue
		}
		if tx.Result.BlockTime == nil {
			continue
		}
		tm := time.Unix(*tx.Result.BlockTime, 0).UTC()
		if tm.Before(start) || tm.After(end) {
			continue
		}
		pre := big.NewInt(0)
		post := big.NewInt(0)
		dec := 6
		for _, b := range tx.Result.Meta.PreTokenBalances {
			if strings.EqualFold(b.Owner, owner) && strings.EqualFold(b.Mint, mint) {
				pre.SetString(b.UITokenAmount.Amount, 10)
				dec = b.UITokenAmount.Decimals
			}
		}
		for _, b := range tx.Result.Meta.PostTokenBalances {
			if strings.EqualFold(b.Owner, owner) && strings.EqualFold(b.Mint, mint) {
				post.SetString(b.UITokenAmount.Amount, 10)
				dec = b.UITokenAmount.Decimals
			}
		}
		delta := new(big.Int).Sub(post, pre)
		if delta.Sign() == 0 {
			continue
		}
		scale := util.Pow10(dec)
		if delta.Sign() > 0 {
			q := new(big.Float).Quo(new(big.Float).SetInt(delta), scale)
			flow.AddWeekly(wb, symbol, tm, true, q)
			flow.AddDaily(db, symbol, tm, true, q)
		} else {
			q := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Abs(delta)), scale)
			flow.AddWeekly(wb, symbol, tm, false, q)
			flow.AddDaily(db, symbol, tm, false, q)
		}
	}
	return nil
}
