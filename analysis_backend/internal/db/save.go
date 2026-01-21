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

// SaveFilterCorrection 保存或更新过滤器修正记录
func SaveFilterCorrection(gdb *gorm.DB, correction *FilterCorrection) error {
	if correction == nil {
		return nil
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 查找是否已存在该交易对的修正记录
		var existing FilterCorrection
		err := tx.Where("symbol = ? AND exchange = ?", correction.Symbol, correction.Exchange).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// 新记录，直接创建
			correction.CorrectionCount = 1
			correction.LastCorrectedAt = time.Now()
			return tx.Create(correction).Error
		} else if err != nil {
			return err
		}

		// 已存在记录，更新计数和时间戳
		existing.CorrectionCount++
		existing.LastCorrectedAt = time.Now()

		// 更新最新的修正数据
		existing.OriginalStepSize = correction.OriginalStepSize
		existing.OriginalMinNotional = correction.OriginalMinNotional
		existing.OriginalMaxQty = correction.OriginalMaxQty
		existing.OriginalMinQty = correction.OriginalMinQty

		existing.CorrectedStepSize = correction.CorrectedStepSize
		existing.CorrectedMinNotional = correction.CorrectedMinNotional
		existing.CorrectedMaxQty = correction.CorrectedMaxQty
		existing.CorrectedMinQty = correction.CorrectedMinQty

		existing.CorrectionType = correction.CorrectionType
		existing.CorrectionReason = correction.CorrectionReason
		existing.IsSmallCapSymbol = correction.IsSmallCapSymbol

		return tx.Save(&existing).Error
	})
}

// GetFilterCorrectionStats 获取过滤器修正统计信息
func GetFilterCorrectionStats(gdb *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总修正记录数
	var totalCount int64
	if err := gdb.Model(&FilterCorrection{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_corrections"] = totalCount

	// 各交易对修正次数统计
	type SymbolStats struct {
		Symbol         string `json:"symbol"`
		CorrectionCount int    `json:"correction_count"`
		LastCorrectedAt time.Time `json:"last_corrected_at"`
	}
	var symbolStats []SymbolStats
	if err := gdb.Model(&FilterCorrection{}).
		Select("symbol, correction_count, last_corrected_at").
		Order("correction_count DESC").
		Limit(10).
		Find(&symbolStats).Error; err != nil {
		return nil, err
	}
	stats["top_corrected_symbols"] = symbolStats

	// 小币种修正统计
	var smallCapCount int64
	if err := gdb.Model(&FilterCorrection{}).Where("is_small_cap_symbol = ?", true).Count(&smallCapCount).Error; err != nil {
		return nil, err
	}
	stats["small_cap_corrections"] = smallCapCount

	// 修正类型分布
	type CorrectionTypeStats struct {
		CorrectionType string `json:"correction_type"`
		Count         int64  `json:"count"`
	}
	var correctionTypeStats []CorrectionTypeStats
	if err := gdb.Model(&FilterCorrection{}).
		Select("correction_type, COUNT(*) as count").
		Group("correction_type").
		Order("count DESC").
		Find(&correctionTypeStats).Error; err != nil {
		return nil, err
	}
	stats["correction_types"] = correctionTypeStats

	// 最近修正记录
	var recentCorrections []FilterCorrection
	if err := gdb.Model(&FilterCorrection{}).
		Order("last_corrected_at DESC").
		Limit(5).
		Find(&recentCorrections).Error; err != nil {
		return nil, err
	}
	stats["recent_corrections"] = recentCorrections

	return stats, nil
}

// GetFilterCorrectionsBySymbol 获取指定交易对的修正历史
func GetFilterCorrectionsBySymbol(gdb *gorm.DB, symbol string) ([]FilterCorrection, error) {
	var corrections []FilterCorrection
	err := gdb.Where("symbol = ?", symbol).Order("created_at DESC").Find(&corrections).Error
	return corrections, err
}

// CleanupOldCorrections 清理旧的修正记录（保留最近N天的记录）
func CleanupOldCorrections(gdb *gorm.DB, daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)
	return gdb.Where("created_at < ?", cutoffDate).Delete(&FilterCorrection{}).Error
}
