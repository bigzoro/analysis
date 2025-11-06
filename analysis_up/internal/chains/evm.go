package chains

import (
	"analysis/internal/config"
	"analysis/internal/flow"
	"analysis/internal/models"
	"analysis/internal/netutil"
	"analysis/internal/util"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

var erc20ABI = mustParseABI(`[
	{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"}
]`)

func mustParseABI(s string) abi.ABI {
	a, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return a
}

func EVMNativeBalance(ctx context.Context, rpcURL string, addr common.Address) (*big.Int, error) {
	rpc, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer rpc.Close()
	var hexbal string
	if err := rpc.CallContext(ctx, &hexbal, "eth_getBalance", addr.Hex(), "latest"); err != nil {
		return nil, err
	}
	z := new(big.Int)
	z.SetString(strings.TrimPrefix(hexbal, "0x"), 16)
	return z, nil
}

func EVMERC20Balance(ctx context.Context, rpcURL string, token, owner common.Address) (*big.Int, int, error) {
	rpc, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, 0, err
	}
	defer rpc.Close()
	var resp string
	data, _ := erc20ABI.Pack("decimals")
	if err := rpc.CallContext(ctx, &resp, "eth_call", map[string]any{"to": token.Hex(), "data": "0x" + hex.EncodeToString(data)}, "latest"); err != nil {
		return nil, 0, err
	}
	dec := new(big.Int)
	dec.SetString(strings.TrimPrefix(resp, "0x"), 16)
	decimals := int(dec.Int64())
	data, _ = erc20ABI.Pack("balanceOf", owner)
	if err := rpc.CallContext(ctx, &resp, "eth_call", map[string]any{"to": token.Hex(), "data": "0x" + hex.EncodeToString(data)}, "latest"); err != nil {
		return nil, 0, err
	}
	bal := new(big.Int)
	bal.SetString(strings.TrimPrefix(resp, "0x"), 16)
	return bal, decimals, nil
}

func EVMERC20Decimals(ctx context.Context, rpcURL string, token common.Address) (int, error) {
	rpc, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return 0, err
	}
	defer rpc.Close()
	var resp string
	data, _ := erc20ABI.Pack("decimals")
	if err := rpc.CallContext(ctx, &resp, "eth_call", map[string]any{"to": token.Hex(), "data": "0x" + hex.EncodeToString(data)}, "latest"); err != nil {
		return 0, err
	}
	dec := new(big.Int)
	dec.SetString(strings.TrimPrefix(resp, "0x"), 16)
	return int(dec.Int64()), nil
}

// —— Flows (ERC20 Transfer 事件)
var transferTopic = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

func evmFindBlockByTime(ctx context.Context, rpc *gethrpc.Client, ts int64) (uint64, error) {
	var latestHex string
	if err := rpc.CallContext(ctx, &latestHex, "eth_blockNumber"); err != nil {
		return 0, err
	}
	latest, _ := new(big.Int).SetString(strings.TrimPrefix(latestHex, "0x"), 16)
	lo, hi := uint64(0), latest.Uint64()
	for lo < hi {
		mid := (lo + hi) / 2
		var blk map[string]any
		if err := rpc.CallContext(ctx, &blk, "eth_getBlockByNumber", fmt.Sprintf("0x%x", mid), false); err != nil {
			return 0, err
		}
		tsHex, _ := blk["timestamp"].(string)
		bts, _ := new(big.Int).SetString(strings.TrimPrefix(tsHex, "0x"), 16)
		if bts.Int64() < ts {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo, nil
}

type evmLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
	Block   string   `json:"blockNumber"`
}

func evmFetchTransferLogs(ctx context.Context, rpcURL string, token common.Address, fromBlock, toBlock uint64, topic1 *common.Hash, topic2 *common.Hash) ([]evmLog, error) {
	rpc, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer rpc.Close()
	params := map[string]any{
		"address":   token.Hex(),
		"fromBlock": fmt.Sprintf("0x%x", fromBlock),
		"toBlock":   fmt.Sprintf("0x%x", toBlock),
		"topics":    []any{transferTopic, topic1, topic2},
	}
	var logs []evmLog
	if err := rpc.CallContext(ctx, &logs, "eth_getLogs", params); err != nil {
		return nil, err
	}
	return logs, nil
}

func EVMERC20Flows(ctx context.Context, rpcURL string, token config.TokenERC20, owner common.Address, start, end time.Time, wb models.WeeklyBucket, db models.DailyBucket) error {
	rpc, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer rpc.Close()

	fromBlk, err := evmFindBlockByTime(ctx, rpc, start.Unix())
	if err != nil {
		return err
	}
	toBlk, err := evmFindBlockByTime(ctx, rpc, end.Unix())
	if err != nil {
		return err
	}

	dec, err := EVMERC20Decimals(ctx, rpcURL, common.HexToAddress(token.Address))
	if err != nil {
		dec = 6
	}
	scale := util.Pow10(dec)

	addrTopic := common.BytesToHash(common.LeftPadBytes(owner.Bytes(), 32))
	inLogs, err := evmFetchTransferLogs(ctx, rpcURL, common.HexToAddress(token.Address), fromBlk, toBlk, nil, &addrTopic)
	if err != nil {
		return err
	}
	outLogs, err := evmFetchTransferLogs(ctx, rpcURL, common.HexToAddress(token.Address), fromBlk, toBlk, &addrTopic, nil)
	if err != nil {
		return err
	}

	blockTime := func(hexNum string) (time.Time, error) {
		num, _ := new(big.Int).SetString(strings.TrimPrefix(hexNum, "0x"), 16)
		var blk map[string]any
		if err := rpc.CallContext(ctx, &blk, "eth_getBlockByNumber", fmt.Sprintf("0x%x", num.Uint64()), false); err != nil {
			return time.Time{}, err
		}
		tsHex, _ := blk["timestamp"].(string)
		its, _ := new(big.Int).SetString(strings.TrimPrefix(tsHex, "0x"), 16)
		return time.Unix(its.Int64(), 0).UTC(), nil
	}

	for _, lg := range inLogs {
		tm, err := blockTime(lg.Block)
		if err != nil || tm.Before(start) || tm.After(end) {
			continue
		}
		val := new(big.Int)
		val.SetString(strings.TrimPrefix(lg.Data, "0x"), 16)
		q := new(big.Float).Quo(new(big.Float).SetInt(val), scale)
		flow.AddWeekly(wb, token.Symbol, tm, true, q)
		flow.AddDaily(db, token.Symbol, tm, true, q)
	}
	for _, lg := range outLogs {
		tm, err := blockTime(lg.Block)
		if err != nil || tm.Before(start) || tm.After(end) {
			continue
		}
		val := new(big.Int)
		val.SetString(strings.TrimPrefix(lg.Data, "0x"), 16)
		q := new(big.Float).Quo(new(big.Float).SetInt(val), scale)
		flow.AddWeekly(wb, token.Symbol, tm, false, q)
		flow.AddDaily(db, token.Symbol, tm, false, q)
	}
	return nil
}

func ETHNativeFlowsEtherscan(ctx context.Context, etherscanKey, rpcURL, addr string, start, end time.Time, wb models.WeeklyBucket, db models.DailyBucket) error {
	if etherscanKey == "" || !util.IsAllowed("ETH") {
		return nil
	}
	rpc, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer rpc.Close()
	fromBlk, err := evmFindBlockByTime(ctx, rpc, start.Unix())
	if err != nil {
		return err
	}
	toBlk, err := evmFindBlockByTime(ctx, rpc, end.Unix())
	if err != nil {
		return err
	}
	api := fmt.Sprintf("https://api.etherscan.io/api?module=account&action=txlist&address=%s&startblock=%d&endblock=%d&sort=asc&apikey=%s", addr, fromBlk, toBlk, etherscanKey)

	var resp struct {
		Result []struct {
			TimeStamp string `json:"timeStamp"`
			From      string `json:"from"`
			To        string `json:"to"`
			Value     string `json:"value"`
			GasUsed   string `json:"gasUsed"`
			GasPrice  string `json:"gasPrice"`
		} `json:"result"`
	}
	if err := netutil.GetJSON(ctx, api, &resp); err != nil {
		return err
	}
	scale := util.Pow10(18)
	me := strings.ToLower(addr)
	for _, tx := range resp.Result {
		ts, _ := new(big.Int).SetString(tx.TimeStamp, 10)
		tm := time.Unix(ts.Int64(), 0).UTC()
		if tm.Before(start) || tm.After(end) {
			continue
		}
		val := new(big.Int)
		val.SetString(tx.Value, 10)
		gused := new(big.Int)
		gused.SetString(tx.GasUsed, 10)
		gp := new(big.Int)
		gp.SetString(tx.GasPrice, 10)
		fee := new(big.Int).Mul(gused, gp)
		from := strings.ToLower(tx.From)
		to := strings.ToLower(tx.To)
		if to == me && val.Sign() > 0 {
			q := new(big.Float).Quo(new(big.Float).SetInt(val), scale)
			flow.AddWeekly(wb, "ETH", tm, true, q)
			flow.AddDaily(db, "ETH", tm, true, q)
		}
		if from == me {
			out := new(big.Int).Add(val, fee)
			if out.Sign() > 0 {
				q := new(big.Float).Quo(new(big.Float).SetInt(out), scale)
				flow.AddWeekly(wb, "ETH", tm, false, q)
				flow.AddDaily(db, "ETH", tm, false, q)
			}
		}
	}
	return nil
}
