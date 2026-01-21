package scanning

import (
	"analysis/internal/server/strategy/grid_trading"
	"analysis/internal/server/strategy/grid_trading/candidates"
	"analysis/internal/server/strategy/grid_trading/config"
	"analysis/internal/server/strategy/grid_trading/scoring"
	"analysis/internal/server/strategy/grid_trading/validation"
	"context"
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

// Scanner 网格交易策略扫描器
type Scanner struct {
	configManager     grid_trading.ConfigManager
	scoringEngine     grid_trading.ScoringEngine
	candidateSelector grid_trading.CandidateSelector
	validator         grid_trading.Validator
	calculator        grid_trading.GridCalculator
}

// NewScanner 创建网格交易策略扫描器
func NewScanner() *Scanner {
	configManager := config.NewManager()
	scoringEngine := scoring.NewEngine()
	candidateSelector := candidates.NewSelector()

	// 创建一个简单的网格计算器实现
	calculator := &GridCalculatorImpl{}

	validator := validation.NewValidator(scoringEngine, calculator)

	return &Scanner{
		configManager:     configManager,
		scoringEngine:     scoringEngine,
		candidateSelector: candidateSelector,
		validator:         validator,
		calculator:        calculator,
	}
}

// Scan 执行网格交易策略扫描
func (s *Scanner) Scan(ctx context.Context, config *grid_trading.GridTradingConfig) ([]grid_trading.CandidateResult, error) {
	log.Printf("[GridTrading] 开始扫描网格交易策略候选币种")

	var candidates []string
	var err error

	// 根据配置选择候选币种
	if config.UseMarketCap {
		candidates, err = s.candidateSelector.SelectByMarketCap(ctx, config.MaxCandidates)
	} else if config.UseVolume {
		candidates, err = s.candidateSelector.SelectByVolume(ctx, config.MaxCandidates)
	} else {
		candidates, err = s.candidateSelector.FallbackToDefaults(config.MaxCandidates)
	}

	if err != nil {
		log.Printf("[GridTrading] 候选选择失败，使用默认列表: %v", err)
		candidates, err = s.candidateSelector.FallbackToDefaults(config.MaxCandidates)
		if err != nil {
			return nil, fmt.Errorf("获取候选币种失败: %w", err)
		}
	}

	log.Printf("[GridTrading] 选择了%d个候选币种进行网格策略分析", len(candidates))

	var results []grid_trading.CandidateResult

	// 验证每个候选币种
	for _, symbol := range candidates {
		result, err := s.validator.ValidateCandidate(ctx, symbol, config)
		if err != nil {
			log.Printf("[GridTrading] 验证候选币种%s失败: %v", symbol, err)
			continue
		}

		results = append(results, *result)

		// 限制返回数量
		if len(results) >= config.MaxCandidates {
			break
		}
	}

	log.Printf("[GridTrading] 网格交易策略扫描完成，共找到%d个符合条件的币种", len(results))
	return results, nil
}

// CalculateGridRange 计算网格范围
func (s *Scanner) CalculateGridRange(currentPrice float64, config *grid_trading.GridTradingConfig) grid_trading.GridRange {
	return s.calculator.CalculateDynamicRange(currentPrice, config)
}

// ============================================================================
// 适配器和注册表
// ============================================================================

// strategyScannerAdapter 策略扫描器适配器
type strategyScannerAdapter struct {
	scanner *Scanner
}

// Scan 实现StrategyScanner接口
func (a *strategyScannerAdapter) Scan(ctx context.Context, tradingStrategy *pdb.TradingStrategy) ([]interface{}, error) {
	// 转换配置
	configManager := config.NewManager()
	gridConfig := configManager.ConvertConfig(tradingStrategy.Conditions)

	// 执行扫描
	results, err := a.scanner.Scan(ctx, gridConfig)
	if err != nil {
		return nil, err
	}

	// 转换结果为interface{}切片
	eligibleSymbols := make([]interface{}, 0, len(results))
	for _, result := range results {
		if result.IsEligible {
			// 根据网格交易策略，action默认为"grid_setup"
			action := "grid_setup"
			symbolMap := map[string]interface{}{
				"symbol":       result.Symbol,
				"action":       action,
				"reason":       result.Reason,
				"multiplier":   1.0,
				"market_cap":   result.MarketCap,
				"gainers_rank": 0,
			}
			eligibleSymbols = append(eligibleSymbols, symbolMap)
		}
	}

	return eligibleSymbols, nil
}

// GetStrategyType 获取策略类型
func (a *strategyScannerAdapter) GetStrategyType() string {
	return "grid_trading"
}

// ToStrategyScanner 创建适配器
func (s *Scanner) ToStrategyScanner() interface{} {
	return &strategyScannerAdapter{scanner: s}
}

// ============================================================================
// 注册表
// ============================================================================

var globalScanner *Scanner

// GetGridTradingScanner 获取网格交易策略扫描器实例
func GetGridTradingScanner() *Scanner {
	if globalScanner == nil {
		globalScanner = NewScanner()
	}
	return globalScanner
}

// ============================================================================
// 简单的网格计算器实现
// ============================================================================

// GridCalculatorImpl 网格计算器实现
type GridCalculatorImpl struct{}

// CalculateDynamicRange 计算动态网格范围
func (c *GridCalculatorImpl) CalculateDynamicRange(currentPrice float64, config *grid_trading.GridTradingConfig) grid_trading.GridRange {
	// 简单的实现，实际应该有更复杂的逻辑
	rangePercent := config.GridSpacingPercent / 100.0
	upperPrice := currentPrice * (1.0 + rangePercent)
	lowerPrice := currentPrice * (1.0 - rangePercent)

	return grid_trading.GridRange{
		Upper: upperPrice,
		Lower: lowerPrice,
	}
}

// ValidateRange 验证网格范围
func (c *GridCalculatorImpl) ValidateRange(gridRange grid_trading.GridRange, config *grid_trading.GridTradingConfig) bool {
	// 基本的范围验证
	if gridRange.Upper <= gridRange.Lower {
		return false
	}

	// 范围不能超过配置的最大值
	rangePercent := (gridRange.Upper - gridRange.Lower) / ((gridRange.Upper + gridRange.Lower) / 2)
	if rangePercent > config.MaxGridRange {
		return false
	}

	return true
}
