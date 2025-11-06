//package db
//
//import "time"
//
//type PortfolioSnapshot struct {
//	ID        uint      `gorm:"primaryKey"`
//	RunID     string    `gorm:"type:char(36);index"`
//	Entity    string    `gorm:"size:64;index"`
//	AsOf      time.Time `gorm:"index"`
//	TotalUSD  string    `gorm:"type:decimal(38,8);default:0"`
//	CreatedAt time.Time
//}
//
//type Holding struct {
//	ID       uint   `gorm:"primaryKey"`
//	RunID    string `gorm:"type:char(36);index"`
//	Entity   string `gorm:"size:64;index"`
//	Chain    string `gorm:"size:32;index"`
//	Symbol   string `gorm:"size:16;index"`
//	Decimals int
//	Amount   string `gorm:"type:decimal(38,18)"`
//	ValueUSD string `gorm:"type:decimal(38,8)"`
//}
//
//type DailyFlow struct {
//	ID     uint   `gorm:"primaryKey"`
//	RunID  string `gorm:"type:char(36);index"`
//	Entity string `gorm:"size:64;index"`
//	Day    string `gorm:"size:10;index"` // YYYY-MM-DD
//	Coin   string `gorm:"size:16;index"`
//	In     string `gorm:"type:decimal(38,18)"`
//	Out    string `gorm:"type:decimal(38,18)"`
//	Net    string `gorm:"type:decimal(38,18)"`
//}
//
//type WeeklyFlow struct {
//	ID     uint   `gorm:"primaryKey"`
//	RunID  string `gorm:"type:char(36);index"`
//	Entity string `gorm:"size:64;index"`
//	Week   string `gorm:"size:10;index"` // YYYY-Www
//	Coin   string `gorm:"size:16;index"`
//	In     string `gorm:"type:decimal(38,18)"`
//	Out    string `gorm:"type:decimal(38,18)"`
//	Net    string `gorm:"type:decimal(38,18)"`
//}
//
//type TransferEvent struct {
//	ID        uint      `gorm:"primaryKey"`
//	RunID     string    `gorm:"type:char(36);index"`
//	Entity    string    `gorm:"size:64;index:idx_ent_coin_created,priority:1;uniqueIndex:uniq_evt,priority:1"`
//	Chain     string    `gorm:"size:32;index"`
//	Coin      string    `gorm:"size:16;index:idx_ent_coin_created,priority:2;uniqueIndex:uniq_evt,priority:2"`
//	Direction string    `gorm:"size:8;index;uniqueIndex:uniq_evt,priority:6"` // in/out
//	Amount    string    `gorm:"type:decimal(38,18);uniqueIndex:uniq_evt,priority:7"`
//	TS        time.Time `gorm:"index;uniqueIndex:uniq_evt,priority:3"`
//	TxID      string    `gorm:"size:100;index;uniqueIndex:uniq_evt,priority:4"`
//	From      string    `gorm:"size:100"`
//	To        string    `gorm:"size:100"`
//	Address   string    `gorm:"size:100;index;uniqueIndex:uniq_evt,priority:5"`
//	CreatedAt time.Time `gorm:"index:idx_ent_coin_created,priority:3"`
//}

package db

import "time"

// 资产/资金流（保持不变）
type PortfolioSnapshot struct {
	ID        uint      `gorm:"primaryKey"`
	RunID     string    `gorm:"type:char(36);index:idx_ps_run_ent,unique"`
	Entity    string    `gorm:"size:64;index:idx_ps_run_ent,unique"`
	TotalUSD  string    `gorm:"type:decimal(38,8)"`
	AsOf      time.Time `gorm:"index"`
	CreatedAt time.Time
}

type Holding struct {
	ID        uint   `gorm:"primaryKey"`
	RunID     string `gorm:"type:char(36);index:idx_h_run_ent_chain_sym,unique"`
	Entity    string `gorm:"size:64;index:idx_h_run_ent_chain_sym,unique"`
	Chain     string `gorm:"size:32;index:idx_h_run_ent_chain_sym,unique"`
	Symbol    string `gorm:"size:32;index:idx_h_run_ent_chain_sym,unique"`
	Amount    string `gorm:"type:decimal(38,18)"`
	Decimals  int
	ValueUSD  string `gorm:"type:decimal(38,8)"`
	CreatedAt time.Time
}

type WeeklyFlow struct {
	ID        uint   `gorm:"primaryKey"`
	RunID     string `gorm:"type:char(36);index:idx_w_run_ent_coin_week,unique"`
	Entity    string `gorm:"size:64;index:idx_w_run_ent_coin_week,unique"`
	Coin      string `gorm:"size:16;index:idx_w_run_ent_coin_week,unique"`
	Week      string `gorm:"size:10;index:idx_w_run_ent_coin_week,unique"` // 2025-W35
	In        string `gorm:"type:decimal(38,18)"`
	Out       string `gorm:"type:decimal(38,18)"`
	Net       string `gorm:"type:decimal(38,18)"`
	CreatedAt time.Time
}

type DailyFlow struct {
	ID        uint   `gorm:"primaryKey"`
	RunID     string `gorm:"type:char(36);index:idx_d_run_ent_coin_day,unique"`
	Entity    string `gorm:"size:64;index:idx_d_run_ent_coin_day,unique"`
	Coin      string `gorm:"size:16;index:idx_d_run_ent_coin_day,unique"`
	Day       string `gorm:"type:date;index:idx_d_run_ent_coin_day,unique"`
	In        string `gorm:"type:decimal(38,18)"`
	Out       string `gorm:"type:decimal(38,18)"`
	Net       string `gorm:"type:decimal(38,18)"`
	CreatedAt time.Time
}

// 实时转账事件（复合唯一键 + LogIndex）
type TransferEvent struct {
	ID         uint      `gorm:"primaryKey"`
	RunID      string    `gorm:"type:char(36);index"`
	Entity     string    `gorm:"size:64;uniqueIndex:ux_te"`
	Chain      string    `gorm:"size:32;uniqueIndex:ux_te"`
	Coin       string    `gorm:"size:16;uniqueIndex:ux_te"`
	Direction  string    `gorm:"size:8;uniqueIndex:ux_te"` // "in"/"out"
	Amount     string    `gorm:"type:decimal(38,18)"`
	TxID       string    `gorm:"size:128;uniqueIndex:ux_te"`
	Address    string    `gorm:"size:128;uniqueIndex:ux_te"` // 命中的监控地址
	From       string    `gorm:"size:128"`
	To         string    `gorm:"size:128"`
	LogIndex   int       `gorm:"uniqueIndex:ux_te;default:-1"` // ERC20: 链上 logIndex；原生: -1
	OccurredAt time.Time `gorm:"index"`
	CreatedAt  time.Time
}

// 扫描游标（断点续扫）
type TransferCursor struct {
	ID        uint   `gorm:"primaryKey"`
	Entity    string `gorm:"size:64;uniqueIndex:ux_cursor"`
	Chain     string `gorm:"size:32;uniqueIndex:ux_cursor"`
	Block     uint64 `gorm:"type:bigint unsigned"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
