package models

import "time"

// 扫描器 -> API 上报的统一事件
type Event struct {
	Entity    string    `json:"entity"`
	Chain     string    `json:"chain"`
	Coin      string    `json:"coin"`
	Direction string    `json:"direction"` // "in" / "out"
	Amount    string    `json:"amount"`    // 十进制字符串
	TS        time.Time `json:"ts"`        // 发生时间(UTC)
	TxID      string    `json:"txid"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Address   string    `json:"address"`   // 命中的监控地址
	LogIndex  int       `json:"log_index"` // ERC20: 链上 logIndex；原生: -1
}
