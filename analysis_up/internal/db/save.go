package db

import (
	"analysis/internal/models"
	"math/big"
	"time"

	"gorm.io/gorm"
)

func fstr(x *big.Float, prec int) string {
	if x == nil {
		return "0"
	}
	s := x.Text('f', prec)
	for len(s) > 1 && s[len(s)-1] == '0' && s[len(s)-2] != '.' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

func SaveAll(gdb *gorm.DB, runID string, asOf time.Time, portfolios []models.Portfolio, weekly []models.WeeklyResult, daily []models.DailyResult) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, p := range portfolios {
			ps := PortfolioSnapshot{
				RunID:    runID,
				Entity:   p.Entity,
				TotalUSD: fstr(new(big.Float).SetFloat64(p.TotalUSD), 8),
				AsOf:     asOf.UTC(),
			}
			if err := tx.Create(&ps).Error; err != nil {
				return err
			}
			for _, h := range p.Holdings {
				txh := Holding{
					RunID:    runID,
					Entity:   p.Entity,
					Chain:    h.Chain,
					Symbol:   h.Symbol,
					Amount:   h.Amount,
					Decimals: h.Decimals,
					ValueUSD: fstr(new(big.Float).SetFloat64(h.ValueUSD), 8),
					//AsOf:     asOf.UTC(),
				}
				if err := tx.Create(&txh).Error; err != nil {
					return err
				}
			}
		}
		for _, wr := range weekly {
			for coin, weeks := range wr.Data {
				for wk, io := range weeks {
					net := new(big.Float)
					if io.In != nil {
						net.Add(net, io.In)
					}
					if io.Out != nil {
						net.Sub(net, io.Out)
					}
					item := WeeklyFlow{
						RunID:  runID,
						Entity: wr.Entity,
						Coin:   coin,
						Week:   string(wk),
						In:     fstr(io.In, 18),
						Out:    fstr(io.Out, 18),
						Net:    fstr(net, 18),
					}
					if err := tx.Create(&item).Error; err != nil {
						return err
					}
				}
			}
		}
		for _, dr := range daily {
			for coin, days := range dr.Data {
				for dk, io := range days {
					net := new(big.Float)
					if io.In != nil {
						net.Add(net, io.In)
					}
					if io.Out != nil {
						net.Sub(net, io.Out)
					}
					item := DailyFlow{
						RunID:  runID,
						Entity: dr.Entity,
						Coin:   coin,
						Day:    string(dk),
						In:     fstr(io.In, 18),
						Out:    fstr(io.Out, 18),
						Net:    fstr(net, 18),
					}
					if err := tx.Create(&item).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
