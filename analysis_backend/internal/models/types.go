package models

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Holding struct {
	Symbol   string  `json:"symbol"`
	Amount   string  `json:"amount"`
	Decimals int     `json:"decimals"`
	ValueUSD float64 `json:"value_usd"`
	Chain    string  `json:"chain"`
}

// 实现 util.HoldingLike
func (h *Holding) SetMeta(symbol string, decimals int, chain string) {
	h.Symbol, h.Decimals, h.Chain = symbol, decimals, chain
}
func (h *Holding) AddAmount(q *big.Float) {
	if h.Amount == "" {
		h.Amount = q.Text('f', 8)
		return
	}
	old, _ := new(big.Float).SetString(h.Amount)
	old = new(big.Float).Add(old, q)
	h.Amount = old.Text('f', 8)
}
func (h *Holding) AddValueUSD(v float64) { h.ValueUSD += v }

type Portfolio struct {
	Entity   string             `json:"entity"`
	Holdings map[string]Holding `json:"holdings"`
	TotalUSD float64            `json:"total_usd"`
	TS       int64              `json:"timestamp"`
}

type AddressRow struct {
	Entity  string
	Chain   string
	Address string
	Source  string
}

func (r AddressRow) EVM() common.Address { return common.HexToAddress(r.Address) }

type WeekKey string
type DayKey string

type FlowIO struct {
	In, Out *big.Float
}

type WeeklyBucket map[string]map[WeekKey]*FlowIO
type DailyBucket map[string]map[DayKey]*FlowIO

type WeeklyResult struct {
	Entity string
	Data   WeeklyBucket
}
type DailyResult struct {
	Entity string
	Data   DailyBucket
}
