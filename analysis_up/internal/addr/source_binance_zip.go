package addr

import (
	"analysis/internal/models"
	"analysis/internal/util"
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"regexp"
	"strings"
)

func RowsFromBinancePORZip(zipPath, entity string, includeDeposit bool) ([]models.AddressRow, error) {
	if zipPath == "" {
		return nil, nil
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var out []models.AddressRow
	depRE := regexp.MustCompile(`(?i)(deposit|充值|收款|充币|入金)`)

	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if !strings.HasSuffix(name, ".csv") {
			continue
		}
		if !includeDeposit && strings.Contains(name, "deposit") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		rdr := csv.NewReader(bytes.NewReader(data))
		rdr.FieldsPerRecord = -1
		rdr.LazyQuotes = true
		rows, err := rdr.ReadAll()
		if err != nil || len(rows) == 0 {
			continue
		}
		head := rows[0]

		find := func(cands ...string) int {
			for i, h := range head {
				for _, c := range cands {
					if strings.EqualFold(strings.TrimSpace(h), c) {
						return i
					}
				}
			}
			return -1
		}
		ic := find("coin", "asset", "currency")
		in := find("network", "chain", "blockchain")
		ia := find("address", "addr")
		it := find("type", "address_type", "wallet_type", "label")
		if ic < 0 || in < 0 || ia < 0 {
			continue
		}

		for _, row := range rows[1:] {
			if ic >= len(row) || in >= len(row) || ia >= len(row) {
				continue
			}
			asset := strings.ToUpper(strings.TrimSpace(row[ic]))
			if !util.IsAllowed(asset) {
				continue
			}
			net := strings.TrimSpace(row[in])
			chain := util.NormalizeChainNameLoose(net)
			addr := strings.TrimSpace(row[ia])
			if addr == "" || chain == "" {
				continue
			}
			if !includeDeposit && it >= 0 {
				if depRE.MatchString(strings.TrimSpace(row[it])) {
					continue
				}
			}
			out = append(out, models.AddressRow{
				Entity:  entity,
				Chain:   chain,
				Address: addr,
				Source:  f.Name,
			})
		}
	}
	return out, nil
}
