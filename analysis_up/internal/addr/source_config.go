package addr

import (
	"analysis/internal/config"
	"analysis/internal/models"
	"analysis/internal/util"
)

func RowsFromConfig(cfg config.Config) []models.AddressRow {
	var out []models.AddressRow
	for _, e := range cfg.Entities {
		for net, addrs := range e.Networks {
			chain := util.NormalizeChainNameLoose(net)
			for _, a := range addrs {
				a = stringsTrimSpace(a)
				if a == "" {
					continue
				}
				out = append(out, models.AddressRow{
					Entity:  e.Name,
					Chain:   chain,
					Address: a,
					Source:  "config",
				})
			}
		}
	}
	return out
}

func stringsTrimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	for i < j && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\t' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
