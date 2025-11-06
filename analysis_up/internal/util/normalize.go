package util

import "strings"

func NormalizeChainNameLoose(s string) string {
	x := strings.ToLower(strings.TrimSpace(s))
	x = strings.ReplaceAll(x, " ", "")
	x = strings.ReplaceAll(x, "-", "")
	x = strings.ReplaceAll(x, "_", "")
	switch {
	case x == "btc" || x == "bitcoin":
		return "bitcoin"
	case x == "eth" || x == "ethereum":
		return "ethereum"
	case x == "sol" || x == "solana":
		return "solana"
	case x == "trx" || x == "tron" || strings.Contains(x, "trc20"):
		return "tron"
	case x == "bsc" || strings.Contains(x, "bnb") || strings.Contains(x, "bep20"):
		return "bsc"
	case strings.Contains(x, "arbitrum") || x == "arb":
		return "arbitrum"
	case strings.Contains(x, "optimism") || x == "op":
		return "optimism"
	case strings.Contains(x, "polygon") || x == "matic":
		return "polygon"
	case x == "base":
		return "base"
	default:
		return x
	}
}
