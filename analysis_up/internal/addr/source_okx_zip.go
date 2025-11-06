// internal/addr/okx.go
package addr

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	// 按你的 go.mod 路径导入
	"analysis/internal/models"
	"analysis/internal/util"
)

// RowsFromOKXPOR 读取 OKX 官方 zip/csv，返回地址清单（models.AddressRow）。
// includeDeposit：当前这批 CSV 不含 Type=Deposit，不生效（仅日志提示）。
// includeStaking：是否纳入 staking CSV 中的地址（deposit/withdrawal）。
// 过滤规则：与 Binance 一致——按资产白名单 util.IsAllowed(asset) 过滤。
func RowsFromOKXPOR(path string, entity string, includeDeposit bool, includeStaking bool) ([]models.AddressRow, error) {
	if path == "" {
		return nil, errors.New("okx por path is empty")
	}

	var all []models.AddressRow
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".zip":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		st, _ := f.Stat()
		zr, err := zip.NewReader(f, st.Size())
		if err != nil {
			return nil, err
		}

		for _, zf := range zr.File {
			if !strings.HasSuffix(strings.ToLower(zf.Name), ".csv") {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				return nil, err
			}
			buf := new(bytes.Buffer)
			_, _ = io.Copy(buf, rc)
			rc.Close()

			rows, stats := parseOneCSV(buf.Bytes(), zf.Name, entity, includeStaking)
			all = append(all, rows...)

			log.Printf("okx POR file parsed: name=%s main_rows=%d staking_rows=%d included_staking=%d filtered_asset=%d total_detected=%d %s",
				stats.File, stats.MainAddrRows, stats.StakingRows, stats.IncludedStakingRows, stats.FilteredByAsset, stats.TotalRows, stats.DepositFlagNote)
		}

	default: // 单个 csv
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rows, stats := parseOneCSV(data, filepath.Base(path), entity, includeStaking)
		all = append(all, rows...)
		log.Printf("okx POR file parsed: name=%s main_rows=%d staking_rows=%d included_staking=%d filtered_asset=%d total_detected=%d %s",
			stats.File, stats.MainAddrRows, stats.StakingRows, stats.IncludedStakingRows, stats.FilteredByAsset, stats.TotalRows, stats.DepositFlagNote)
	}

	// 汇总去重
	all = dedupModelRows(all)
	log.Printf("okx POR summary: file=%s final_unique=%d (include_staking=%v, include_deposit=%v but not used)",
		filepath.Base(path), len(all), includeStaking, includeDeposit)

	return all, nil
}

// ===================== 解析实现 =====================

type parseStats struct {
	File                string
	TotalRows           int
	MainAddrRows        int
	StakingRows         int
	IncludedStakingRows int
	FilteredByAsset     int
	DepositFlagNote     string // 仅日志提示：当前 CSV 无 Type=Deposit
}

// 解析单个 CSV：识别“主清单”或“staking”表头，并抽取地址。
// Source 统一使用文件名（与 RowsFromBinancePORZip / RowsFromConfig 的 Source 风格保持一致）。
func parseOneCSV(data []byte, name string, entity string, includeStaking bool) ([]models.AddressRow, parseStats) {
	stats := parseStats{File: name}
	recs, _ := readCSV(data)
	if len(recs) == 0 {
		return nil, stats
	}

	// 主清单表头（示例）：coin,Network,Snapshot Height,address,amount,message,signature1,signature2,...
	mainIdx := findHeaderIndex(recs, []string{"coin", "network", "snapshot height", "address"})
	// ETH 质押表头（示例）：deposit address,validator publickey,amount,withdrawal address
	stakingIdx := findHeaderIndex(recs, []string{"deposit address", "validator publickey", "amount", "withdrawal address"})

	var out []models.AddressRow

	// 主清单
	if mainIdx >= 0 {
		hdr := indexHeader(recs[mainIdx])
		for i := mainIdx + 1; i < len(recs); i++ {
			assetRaw := pick(recs[i], hdr, "coin")
			asset := normalizeAssetSymbol(assetRaw) // e.g. "USDT(ERC20)" -> "USDT"
			network := pick(recs[i], hdr, "network")
			addr := pick(recs[i], hdr, "address")
			if network == "" || addr == "" {
				continue
			}
			stats.TotalRows++

			// ✅ 与 Binance 一致：资产白名单过滤
			if !util.IsAllowed(asset) {
				stats.FilteredByAsset++
				continue
			}

			out = append(out, models.AddressRow{
				Entity:  entity,
				Chain:   normalizeChainKey(network),
				Address: strings.TrimSpace(addr),
				Source:  name, // 文件名作为来源
			})
			stats.MainAddrRows++
		}
	}

	// Staking（仅在 includeStaking=true 时纳入）
	if stakingIdx >= 0 {
		hdr := indexHeader(recs[stakingIdx])
		for i := stakingIdx + 1; i < len(recs); i++ {
			dep := pick(recs[i], hdr, "deposit address")
			withd := pick(recs[i], hdr, "withdrawal address")
			if dep == "" && withd == "" {
				continue
			}
			stats.TotalRows++

			// ✅ staking 资产按 ETH 过滤
			if !util.IsAllowed("ETH") {
				stats.FilteredByAsset++
				continue
			}

			if includeStaking {
				if dep != "" {
					out = append(out, models.AddressRow{
						Entity:  entity,
						Chain:   "ethereum",
						Address: strings.TrimSpace(dep),
						Source:  name,
					})
					stats.IncludedStakingRows++
				}
				if withd != "" {
					out = append(out, models.AddressRow{
						Entity:  entity,
						Chain:   "ethereum",
						Address: strings.TrimSpace(withd),
						Source:  name,
					})
					stats.IncludedStakingRows++
				}
			}
			stats.StakingRows++
		}
	}

	// 当前文件没有 Type=Deposit 列，提示一下
	stats.DepositFlagNote = "(no Type=Deposit in CSV; include_deposit flag not applied)"
	return out, stats
}

// ===================== 工具函数 =====================

// 读取 CSV，自动去 BOM，简单侦测分隔符（, ; \t）
func readCSV(data []byte) ([][]string, rune) {
	// 去 UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	// 分隔符粗略侦测
	delim := ','
	s := string(data[:min(len(data), 8192)])
	count := func(sep string) int { return strings.Count(s, sep) }
	cComma, cSemi, cTab := count(","), count(";"), count("\t")
	if cSemi >= cComma && cSemi >= cTab {
		delim = ';'
	} else if cTab > cComma && cTab > cSemi {
		delim = '\t'
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.ReuseRecord = true
	r.TrimLeadingSpace = true
	r.LazyQuotes = true
	r.Comma = delim

	recs, err := r.ReadAll()
	if err != nil {
		return nil, delim
	}
	return recs, delim
}

// 在前若干行内查找包含指定键集合的表头
func findHeaderIndex(recs [][]string, keys []string) int {
	limit := len(recs)
	if limit > 200 {
		limit = 200
	}
	for i := 0; i < limit; i++ {
		h := indexHeader(recs[i])
		ok := true
		for _, k := range keys {
			if _, exists := h[k]; !exists {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// 将表头行映射为 key->index（key 统一小写、去空格、去中英文冒号）
func indexHeader(hdr []string) map[string]int {
	m := make(map[string]int)
	for i, h := range hdr {
		l := norm(h)
		if l == "" {
			continue
		}
		m[l] = i
	}
	return m
}

// 取字段
func pick(rec []string, hdr map[string]int, key string) string {
	if idx, ok := hdr[key]; ok && idx >= 0 && idx < len(rec) {
		return strings.TrimSpace(rec[idx])
	}
	return ""
}

// 统一标准化：小写 + trim + 去冒号（中文/英文）
func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "：:")))
}

// 去重：entity|chain|address
func dedupModelRows(in []models.AddressRow) []models.AddressRow {
	seen := make(map[string]struct{}, len(in))
	out := make([]models.AddressRow, 0, len(in))
	for _, x := range in {
		k := x.Entity + "|" + x.Chain + "|" + strings.ToLower(strings.TrimSpace(x.Address))
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, x)
	}
	return out
}

// 链名标准化为内部键
func normalizeChainKey(v string) string {
	u := strings.ToUpper(strings.TrimSpace(v))
	switch u {
	case "BTC", "BITCOIN":
		return "bitcoin"
	case "ETH", "ETHEREUM":
		return "ethereum"
	case "TRX", "TRON":
		return "tron"
	case "SOL", "SOLANA":
		return "solana"
	case "BSC", "BNB CHAIN", "BNB SMART CHAIN", "BINANCE SMART CHAIN":
		return "bsc"
	case "ARBITRUM", "ARBITRUM ONE":
		return "arbitrum"
	case "OPTIMISM", "OP":
		return "optimism"
	case "POLYGON", "MATIC":
		return "polygon"
	case "AVAX", "AVALANCHE", "AVALANCHE C-CHAIN":
		return "avalanche"
	default:
		return strings.ToLower(strings.TrimSpace(v))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 资产名归一化：如 "USDT(ERC20)" / "usdt (trc20)" -> "USDT"
func normalizeAssetSymbol(s string) string {
	s = strings.TrimSpace(s)
	// 去掉括号后的部分
	if i := strings.IndexByte(s, '('); i >= 0 {
		s = s[:i]
	}
	if j := strings.IndexByte(s, ' '); j >= 0 {
		s = s[:j]
	}
	return strings.ToUpper(strings.TrimSpace(s))
}
