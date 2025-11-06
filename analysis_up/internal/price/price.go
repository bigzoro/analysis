package price

import (
	"analysis/internal/config"
	"analysis/internal/netutil"
	"context"
	"fmt"
	"strings"
)

func FetchPrices(ctx context.Context, cfg config.Config, syms []string) (map[string]float64, error) {
	if !cfg.Pricing.Enable {
		return map[string]float64{}, nil
	}
	idset := map[string]struct{}{}
	m := map[string]string{}
	for _, s := range syms {
		id := cfg.Pricing.Map[strings.ToUpper(s)]
		if id != "" {
			m[strings.ToUpper(s)] = id
			idset[id] = struct{}{}
		}
	}
	if len(m) == 0 {
		return map[string]float64{}, nil
	}
	ids := make([]string, 0, len(idset))
	for id := range idset {
		ids = append(ids, id)
	}
	u := fmt.Sprintf("%s?ids=%s&vs_currencies=usd", cfg.Pricing.CoinGeckoEndpoint, strings.Join(ids, ","))
	var raw map[string]map[string]float64
	if err := netutil.GetJSON(ctx, u, &raw); err != nil {
		return nil, err
	}
	out := map[string]float64{}
	for sym, id := range m {
		if v, ok := raw[id]["usd"]; ok {
			out[sym] = v
		}
	}
	return out, nil
}
