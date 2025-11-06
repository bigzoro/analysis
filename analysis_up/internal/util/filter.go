package util

import "strings"

var (
	allowed  = map[string]bool{}
	allowAll = false
)

// SetAllowed 接受逗号分隔的币种，比如 "BTC,ETH,USDT"
// 也可以传 "*" 或 "all" 表示全部允许
func SetAllowed(list string) {
	list = strings.TrimSpace(list)
	if list == "" {
		list = "BTC,ETH,SOL,USDT,USDC"
	}
	if list == "*" || strings.EqualFold(list, "all") {
		allowAll = true
		allowed = map[string]bool{}
		return
	}

	allowAll = false
	m := make(map[string]bool)
	for _, s := range strings.Split(list, ",") {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s != "" {
			m[s] = true
		}
	}
	allowed = m
}

func IsAllowed(sym string) bool {
	if allowAll {
		return true
	}
	return allowed[strings.ToUpper(sym)]
}
