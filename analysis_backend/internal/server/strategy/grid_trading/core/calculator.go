package core

import (
	"analysis/internal/server/strategy/grid_trading"
	"math"
)

// Calculator 网格计算器实现
type Calculator struct{}

// NewCalculator 创建网格计算器
func NewCalculator() grid_trading.GridCalculator {
	return &Calculator{}
}

// CalculateDynamicRange 计算动态网格范围
func (c *Calculator) CalculateDynamicRange(currentPrice float64, config *grid_trading.GridTradingConfig) grid_trading.GridRange {
	if currentPrice <= 0 {
		return grid_trading.GridRange{Lower: 0, Upper: 0}
	}

	// 计算网格间距
	spacing := currentPrice * config.GridSpacingPercent / 100.0

	// 计算网格层数的一半
	halfLevels := float64(config.GridLevels) / 2.0

	// 计算上下限
	lower := currentPrice - (spacing * halfLevels)
	upper := currentPrice + (spacing * halfLevels)

	// 确保下限不小于0
	lower = math.Max(lower, 0.00000001) // 避免除零错误

	// 应用范围限制
	rangePercent := (upper - lower) / currentPrice * 100.0
	if rangePercent < config.MinGridRange {
		// 扩大范围
		expandPercent := config.MinGridRange - rangePercent
		expandAmount := currentPrice * expandPercent / 100.0 / 2.0
		lower -= expandAmount
		upper += expandAmount
	} else if rangePercent > config.MaxGridRange {
		// 缩小范围
		shrinkPercent := rangePercent - config.MaxGridRange
		shrinkAmount := currentPrice * shrinkPercent / 100.0 / 2.0
		lower += shrinkAmount
		upper -= shrinkAmount
	}

	return grid_trading.GridRange{
		Lower: math.Max(lower, 0.00000001),
		Upper: upper,
	}
}

// ValidateRange 验证网格范围
func (c *Calculator) ValidateRange(gridRange grid_trading.GridRange, config *grid_trading.GridTradingConfig) bool {
	if gridRange.Lower >= gridRange.Upper || gridRange.Lower <= 0 {
		return false
	}

	rangePercent := (gridRange.Upper - gridRange.Lower) / ((gridRange.Lower + gridRange.Upper) / 2.0) * 100.0

	return rangePercent >= config.MinGridRange && rangePercent <= config.MaxGridRange
}
