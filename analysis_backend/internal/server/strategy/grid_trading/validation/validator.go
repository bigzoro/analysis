package validation

import (
	"analysis/internal/server/strategy/grid_trading"
	"context"
	"fmt"
)

// Validator 验证器实现
type Validator struct {
	scoringEngine grid_trading.ScoringEngine
	calculator    grid_trading.GridCalculator
}

// NewValidator 创建验证器
func NewValidator(scoringEngine grid_trading.ScoringEngine, calculator grid_trading.GridCalculator) grid_trading.Validator {
	return &Validator{
		scoringEngine: scoringEngine,
		calculator:    calculator,
	}
}

// ValidateCandidate 验证候选币种
func (v *Validator) ValidateCandidate(ctx context.Context, symbol string, config *grid_trading.GridTradingConfig) (*grid_trading.CandidateResult, error) {
	result := &grid_trading.CandidateResult{
		Symbol:     symbol,
		IsEligible: false,
	}

	// 计算各项评分
	volatilityScore, err := v.scoringEngine.CalculateVolatilityScore(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("计算波动率评分失败: %w", err)
	}

	liquidityScore, err := v.scoringEngine.CalculateLiquidityScore(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("计算流动性评分失败: %w", err)
	}

	stabilityScore, err := v.scoringEngine.CalculateStabilityScore(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("计算稳定性评分失败: %w", err)
	}

	// 计算整体评分
	overallScore := v.scoringEngine.CalculateOverallScore(volatilityScore, liquidityScore, stabilityScore)

	result.Score = grid_trading.ScoringResult{
		VolatilityScore: volatilityScore,
		LiquidityScore:  liquidityScore,
		StabilityScore:  stabilityScore,
		OverallScore:    overallScore,
	}

	// 验证评分是否满足阈值
	if !v.ValidateScoringResult(&result.Score, config) {
		result.Reason = fmt.Sprintf("评分不足: 整体评分%.2f (需要>=%.2f)",
			overallScore, config.OverallScoreThreshold)
		return result, nil
	}

	// 计算网格范围（需要当前价格，这里用模拟值）
	currentPrice := 1.0 // 临时模拟值，实际应该从市场数据获取
	result.GridRange = v.calculator.CalculateDynamicRange(currentPrice, config)

	// 验证网格范围
	if !v.calculator.ValidateRange(result.GridRange, config) {
		result.Reason = fmt.Sprintf("网格范围无效: [%.4f, %.4f]", result.GridRange.Lower, result.GridRange.Upper)
		return result, nil
	}

	// 所有验证通过
	result.IsEligible = true
	result.Reason = fmt.Sprintf("符合网格交易条件: 整体评分%.2f", overallScore)

	return result, nil
}

// ValidateScoringResult 验证评分结果
func (v *Validator) ValidateScoringResult(result *grid_trading.ScoringResult, config *grid_trading.GridTradingConfig) bool {
	return result.VolatilityScore >= config.MinVolatilityScore &&
		result.LiquidityScore >= config.MinLiquidityScore &&
		result.StabilityScore >= config.MinStabilityScore &&
		result.OverallScore >= config.OverallScoreThreshold
}
