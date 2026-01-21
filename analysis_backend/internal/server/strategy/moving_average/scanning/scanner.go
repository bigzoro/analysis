package scanning

import (
	"analysis/internal/server/strategy/moving_average"
	"analysis/internal/server/strategy/moving_average/config"
	"analysis/internal/server/strategy/moving_average/indicators"
	"analysis/internal/server/strategy/moving_average/selection"
	"analysis/internal/server/strategy/moving_average/signals"
	"analysis/internal/server/strategy/moving_average/validation"
	"context"
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
)

// Scanner 均线策略扫描器
type Scanner struct {
	configManager     moving_average.ConfigManager
	indicatorCalc     moving_average.IndicatorCalculator
	signalProcessor   moving_average.SignalProcessor
	candidateSelector moving_average.CandidateSelector
	validator         moving_average.Validator
}

// NewScanner 创建均线策略扫描器
func NewScanner() *Scanner {
	configManager := config.NewManager()
	indicatorCalc := indicators.NewCalculator()
	signalProcessor := signals.NewProcessor()
	candidateSelector := selection.NewSelector()
	validator := validation.NewValidator(signalProcessor)

	return &Scanner{
		configManager:     configManager,
		indicatorCalc:     indicatorCalc,
		signalProcessor:   signalProcessor,
		candidateSelector: candidateSelector,
		validator:         validator,
	}
}

// Scan 执行均线策略扫描
func (s *Scanner) Scan(ctx context.Context, config *moving_average.MovingAverageConfig) ([]moving_average.ValidationResult, error) {
	log.Printf("[MovingAverage] 开始扫描均线策略候选币种")

	var allResults []moving_average.ValidationResult

	// 选择候选币种
	candidates, err := s.selectCandidates(ctx, config)
	if err != nil {
		log.Printf("[MovingAverage] 获取候选币种失败: %v", err)
		return nil, fmt.Errorf("获取候选币种失败: %w", err)
	}

	log.Printf("[MovingAverage] 选择了%d个候选币种进行均线分析", len(candidates))

	// 对每个候选币种执行均线分析
	for _, symbol := range candidates {
		results, err := s.DetectCrossSignals(ctx, symbol, config)
		if err != nil {
			log.Printf("[MovingAverage] 分析币种%s失败: %v", symbol, err)
			continue
		}

		// 验证信号
		validResults, err := s.ValidateSignals(ctx, results, config)
		if err != nil {
			log.Printf("[MovingAverage] 验证信号失败: %v", err)
			continue
		}

		allResults = append(allResults, validResults...)

		// 限制总结果数量
		if len(allResults) >= config.MaxCandidates {
			break
		}
	}

	log.Printf("[MovingAverage] 均线策略扫描完成，共找到%d个有效信号", len(allResults))
	return allResults, nil
}

// DetectCrossSignals 检测交叉信号
func (s *Scanner) DetectCrossSignals(ctx context.Context, symbol string, config *moving_average.MovingAverageConfig) ([]moving_average.CrossSignal, error) {
	// 获取价格数据（模拟数据，实际应该从数据源获取）
	prices := s.getSimulatedPriceData(symbol)

	if len(prices) < config.LongPeriod {
		return nil, fmt.Errorf("价格数据不足，至少需要%d个数据点", config.LongPeriod)
	}

	// 计算均线
	shortMA := s.indicatorCalc.CalculateSMA(prices, config.ShortPeriod)
	longMA := s.indicatorCalc.CalculateSMA(prices, config.LongPeriod)

	if len(shortMA) == 0 || len(longMA) == 0 {
		return nil, fmt.Errorf("均线计算失败")
	}

	// 检测交叉
	crosses := s.indicatorCalc.DetectCross(shortMA, longMA)

	// 转换为信号
	var signals []moving_average.CrossSignal
	for _, cross := range crosses {
		if s.shouldIncludeCross(cross, config) {
			signal := moving_average.CrossSignal{
				Symbol:          symbol,
				SignalType:      cross.Type + "_cross",
				CrossPrice:      cross.Price,
				CrossStrength:   cross.Strength,
				VolumeConfirmed: false, // 暂时设为false，实际需要交易量确认
				Timestamp:       time.Now().Unix(),
				Confidence:      cross.Strength, // 简单使用交叉强度作为置信度
			}
			signals = append(signals, signal)
		}
	}

	return signals, nil
}

// ValidateSignals 验证信号
func (s *Scanner) ValidateSignals(ctx context.Context, signals []moving_average.CrossSignal, config *moving_average.MovingAverageConfig) ([]moving_average.ValidationResult, error) {
	var results []moving_average.ValidationResult

	for _, signal := range signals {
		var result *moving_average.ValidationResult

		switch signal.SignalType {
		case "golden_cross":
			if config.UseGoldenCross {
				result = s.signalProcessor.ProcessGoldenCross(&signal, config)
			}
		case "death_cross":
			if config.UseDeathCross {
				result = s.signalProcessor.ProcessDeathCross(&signal, config)
			}
		}

		if result != nil {
			// 计算最终评分
			result.Score = s.validator.CalculateOverallScore(result, config)

			// 应用最终验证
			if result.Score >= 0.6 { // 最终阈值
				result.IsValid = true
			}

			results = append(results, *result)
		}
	}

	return results, nil
}

// selectCandidates 选择候选币种
func (s *Scanner) selectCandidates(ctx context.Context, config *moving_average.MovingAverageConfig) ([]string, error) {
	if config.UseVolumeBasedSelection {
		return s.candidateSelector.SelectByVolume(ctx, config.MaxCandidates)
	}
	return s.candidateSelector.SelectByMarketCap(ctx, config.MaxCandidates)
}

// shouldIncludeCross 判断是否应该包含交叉信号
func (s *Scanner) shouldIncludeCross(cross moving_average.CrossInfo, config *moving_average.MovingAverageConfig) bool {
	// 检查交叉强度
	if cross.Strength < config.MinCrossStrength {
		return false
	}

	// 检查交叉类型
	switch cross.Type {
	case "golden":
		return config.UseGoldenCross
	case "death":
		return config.UseDeathCross
	default:
		return false
	}
}

// getSimulatedPriceData 获取模拟价格数据（实际实现应该从数据源获取）
func (s *Scanner) getSimulatedPriceData(symbol string) []float64 {
	// 模拟价格数据，用于测试
	// 实际应该从数据库或API获取真实数据
	basePrice := 100.0
	prices := make([]float64, 50)

	for i := range prices {
		// 生成一些趋势和波动
		trend := float64(i) * 0.1
		noise := (float64(i%10) - 5.0) * 0.5
		prices[i] = basePrice + trend + noise
	}

	return prices
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
	maConfig := configManager.ConvertConfig(tradingStrategy.Conditions)

	// 执行扫描
	results, err := a.scanner.Scan(ctx, maConfig)
	if err != nil {
		return nil, err
	}

	// 转换结果为interface{}切片
	eligibleSymbols := make([]interface{}, 0, len(results))
	for _, result := range results {
		if result.IsValid {
			symbolMap := map[string]interface{}{
				"symbol":       result.Symbol,
				"action":       result.Action,
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
	return "moving_average"
}

// ToStrategyScanner 创建适配器
func (s *Scanner) ToStrategyScanner() interface{} {
	return &strategyScannerAdapter{scanner: s}
}

// ============================================================================
// 注册表
// ============================================================================

var globalScanner *Scanner

// GetMovingAverageScanner 获取均线策略扫描器实例
func GetMovingAverageScanner() *Scanner {
	if globalScanner == nil {
		globalScanner = NewScanner()
	}
	return globalScanner
}