package util

import (
	"analysis/internal/models"
	"fmt"
	"math/big"
)

type HoldingLike interface {
	SetMeta(symbol string, decimals int, chain string)
	AddAmount(q *big.Float)
	AddValueUSD(v float64)
}

func Pow10(n int) *big.Float {
	x := big.NewFloat(1)
	ten := big.NewFloat(10)
	for i := 0; i < n; i++ {
		x = new(big.Float).Mul(x, ten)
	}
	return x
}

func AddHolding(sum map[string]models.Holding, chain, symbol string, dec int, amt *big.Int, px map[string]float64) {
	symU := stringsToUpper(symbol)
	if !IsAllowed(symU) {
		return
	}
	key := fmt.Sprintf("%s:%s", chain, symU)
	q := new(big.Float).Quo(new(big.Float).SetInt(amt), Pow10(dec))
	val := 0.0
	if p, ok := px[symU]; ok {
		f, _ := q.Float64()
		val = f * p
	}
	h := sum[key]
	h.Symbol, h.Decimals, h.Chain = symU, dec, chain
	if h.Amount == "" {
		h.Amount = q.Text('f', 8)
		h.ValueUSD = val
	} else {
		old, _ := new(big.Float).SetString(h.Amount)
		old = new(big.Float).Add(old, q)
		h.Amount = old.Text('f', 8)
		h.ValueUSD += val
	}
	sum[key] = h
}

func stringsToUpper(s string) string {
	b := []byte(s)
	for i := range b {
		if 'a' <= b[i] && b[i] <= 'z' {
			b[i] = b[i] - 'a' + 'A'
		}
	}
	return string(b)
}
