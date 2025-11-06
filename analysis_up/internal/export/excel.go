package export

import (
	"analysis/internal/models"
	"fmt"
	"math/big"
	"regexp"
	"sort"

	"github.com/xuri/excelize/v2"
)

func WriteExcel(filename string, results []models.Portfolio, weekly []models.WeeklyResult, daily []models.DailyResult) error {
	if len(results) == 0 {
		return fmt.Errorf("no data")
	}
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "Summary")
	head, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	num, _ := f.NewStyle(&excelize.Style{NumFmt: 4})
	text, _ := f.NewStyle(&excelize.Style{NumFmt: 49})

	// Summary
	f.SetCellValue("Summary", "A1", "entity")
	f.SetCellValue("Summary", "B1", "total_usd")
	_ = f.SetCellStyle("Summary", "A1", "B1", head)
	sort.Slice(results, func(i, j int) bool { return results[i].TotalUSD > results[j].TotalUSD })
	for i, p := range results {
		r := i + 2
		f.SetCellValue("Summary", fmt.Sprintf("A%d", r), p.Entity)
		f.SetCellValue("Summary", fmt.Sprintf("B%d", r), p.TotalUSD)
		_ = f.SetCellStyle("Summary", fmt.Sprintf("B%d", r), fmt.Sprintf("B%d", r), num)
	}

	// Entity sheets
	for _, p := range results {
		sh := sanitize(p.Entity)
		_, _ = f.NewSheet(sh)
		f.SetCellValue(sh, "A1", "chain")
		f.SetCellValue(sh, "B1", "symbol")
		f.SetCellValue(sh, "C1", "amount")
		f.SetCellValue(sh, "D1", "value_usd")
		_ = f.SetCellStyle(sh, "A1", "D1", head)

		type row struct {
			ch, sym, amt string
			val          float64
		}
		var rows []row
		for _, h := range p.Holdings {
			rows = append(rows, row{h.Chain, h.Symbol, h.Amount, h.ValueUSD})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].ch == rows[j].ch {
				return rows[i].sym < rows[j].sym
			}
			return rows[i].ch < rows[j].ch
		})
		for i, r := range rows {
			line := i + 2
			f.SetCellValue(sh, fmt.Sprintf("A%d", line), r.ch)
			f.SetCellValue(sh, fmt.Sprintf("B%d", line), r.sym)
			f.SetCellValue(sh, fmt.Sprintf("C%d", line), r.amt)
			_ = f.SetCellStyle(sh, fmt.Sprintf("C%d", line), fmt.Sprintf("C%d", line), text)
			f.SetCellValue(sh, fmt.Sprintf("D%d", line), r.val)
			_ = f.SetCellStyle(sh, fmt.Sprintf("D%d", line), fmt.Sprintf("D%d", line), num)
		}
	}

	// Weekly
	for _, wr := range weekly {
		sh := sanitize(wr.Entity) + "_weekly"
		_, _ = f.NewSheet(sh)
		f.SetCellValue(sh, "A1", "coin")
		f.SetCellValue(sh, "B1", "week")
		f.SetCellValue(sh, "C1", "in")
		f.SetCellValue(sh, "D1", "out")
		f.SetCellValue(sh, "E1", "net")
		_ = f.SetCellStyle(sh, "A1", "E1", head)
		row := 2
		coins := make([]string, 0, len(wr.Data))
		for c := range wr.Data {
			coins = append(coins, c)
		}
		sort.Strings(coins)
		for _, c := range coins {
			weeks := wr.Data[c]
			var keys []models.WeekKey
			for k := range weeks {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			for _, k := range keys {
				io := weeks[k]
				inS, outS, netS := "0", "0", "0"
				if io.In != nil {
					inS = io.In.Text('f', 8)
				}
				if io.Out != nil {
					outS = io.Out.Text('f', 8)
				}
				if io.In != nil || io.Out != nil {
					net := new(big.Float)
					if io.In != nil {
						net = new(big.Float).Add(net, io.In)
					}
					if io.Out != nil {
						net = new(big.Float).Sub(net, io.Out)
					}
					netS = net.Text('f', 8)
				}
				f.SetCellValue(sh, fmt.Sprintf("A%d", row), c)
				f.SetCellValue(sh, fmt.Sprintf("B%d", row), string(k))
				f.SetCellValue(sh, fmt.Sprintf("C%d", row), inS)
				f.SetCellValue(sh, fmt.Sprintf("D%d", row), outS)
				f.SetCellValue(sh, fmt.Sprintf("E%d", row), netS)
				row++
			}
		}
	}

	// Daily
	for _, dr := range daily {
		sh := sanitize(dr.Entity) + "_daily"
		_, _ = f.NewSheet(sh)
		f.SetCellValue(sh, "A1", "coin")
		f.SetCellValue(sh, "B1", "day")
		f.SetCellValue(sh, "C1", "in")
		f.SetCellValue(sh, "D1", "out")
		f.SetCellValue(sh, "E1", "net")
		_ = f.SetCellStyle(sh, "A1", "E1", head)
		row := 2
		coins := make([]string, 0, len(dr.Data))
		for c := range dr.Data {
			coins = append(coins, c)
		}
		sort.Strings(coins)
		for _, c := range coins {
			days := dr.Data[c]
			var keys []models.DayKey
			for k := range days {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			for _, k := range keys {
				io := days[k]
				inS, outS, netS := "0", "0", "0"
				if io.In != nil {
					inS = io.In.Text('f', 8)
				}
				if io.Out != nil {
					outS = io.Out.Text('f', 8)
				}
				if io.In != nil || io.Out != nil {
					net := new(big.Float)
					if io.In != nil {
						net = new(big.Float).Add(net, io.In)
					}
					if io.Out != nil {
						net = new(big.Float).Sub(net, io.Out)
					}
					netS = net.Text('f', 8)
				}
				f.SetCellValue(sh, fmt.Sprintf("A%d", row), c)
				f.SetCellValue(sh, fmt.Sprintf("B%d", row), string(k))
				f.SetCellValue(sh, fmt.Sprintf("C%d", row), inS)
				f.SetCellValue(sh, fmt.Sprintf("D%d", row), outS)
				f.SetCellValue(sh, fmt.Sprintf("E%d", row), netS)
				row++
			}
		}
	}
	return f.SaveAs(filename)
}

func sanitize(s string) string {
	if s == "" {
		s = "sheet"
	}
	re := regexp.MustCompile(`[\\/*?:\[\]]+`)
	s = re.ReplaceAllString(s, "_")
	if len(s) > 31 {
		s = s[:31]
	}
	return s
}
