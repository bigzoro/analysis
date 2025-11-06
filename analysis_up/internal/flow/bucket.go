package flow

import (
	"analysis/internal/models"
	"fmt"
	"math/big"
	"time"
)

func weekKey(t time.Time) models.WeekKey {
	year, wk := t.ISOWeek()
	return models.WeekKey(fmt.Sprintf("%04d-W%02d", year, wk))
}
func dayKey(t time.Time) models.DayKey { return models.DayKey(t.Format("2006-01-02")) }

func AddWeekly(b models.WeeklyBucket, coin string, t time.Time, in bool, amt *big.Float) {
	if b == nil {
		return
	}
	coin = stringsToUpper(coin)
	wk := weekKey(t)
	if _, ok := b[coin]; !ok {
		b[coin] = map[models.WeekKey]*models.FlowIO{}
	}
	if _, ok := b[coin][wk]; !ok {
		b[coin][wk] = &models.FlowIO{}
	}
	if in {
		if b[coin][wk].In == nil {
			b[coin][wk].In = new(big.Float)
		}
		b[coin][wk].In = new(big.Float).Add(b[coin][wk].In, amt)
	} else {
		if b[coin][wk].Out == nil {
			b[coin][wk].Out = new(big.Float)
		}
		b[coin][wk].Out = new(big.Float).Add(b[coin][wk].Out, amt)
	}
}

func AddDaily(b models.DailyBucket, coin string, t time.Time, in bool, amt *big.Float) {
	if b == nil {
		return
	}
	coin = stringsToUpper(coin)
	dk := dayKey(t)
	if _, ok := b[coin]; !ok {
		b[coin] = map[models.DayKey]*models.FlowIO{}
	}
	if _, ok := b[coin][dk]; !ok {
		b[coin][dk] = &models.FlowIO{}
	}
	if in {
		if b[coin][dk].In == nil {
			b[coin][dk].In = new(big.Float)
		}
		b[coin][dk].In = new(big.Float).Add(b[coin][dk].In, amt)
	} else {
		if b[coin][dk].Out == nil {
			b[coin][dk].Out = new(big.Float)
		}
		b[coin][dk].Out = new(big.Float).Add(b[coin][dk].Out, amt)
	}
}

func stringsToUpper(s string) string {
	b := []byte(s)
	for i := range b {
		if 'a' <= b[i] && b[i] <= 'z' {
			b[i] = b[i] - 'a' + 'A'
		}
	}
	return string(b)
}
