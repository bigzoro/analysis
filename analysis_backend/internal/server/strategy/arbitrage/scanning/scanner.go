package scanning

import (
	"analysis/internal/server/strategy/arbitrage"
	"analysis/internal/server/strategy/arbitrage/config"
	"analysis/internal/server/strategy/arbitrage/cross_exchange"
	"analysis/internal/server/strategy/arbitrage/spot_future"
	"analysis/internal/server/strategy/arbitrage/statistical"
	"analysis/internal/server/strategy/arbitrage/triangle"
	"context"
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

// Scanner 套利策略扫描器
type Scanner struct {
	configManager        arbitrage.ConfigManager
	triangleScanner      arbitrage.TriangleArbitrageScanner
	crossExchangeScanner arbitrage.CrossExchangeScanner
	spotFutureScanner    arbitrage.SpotFutureScanner
	statisticalScanner   arbitrage.StatisticalScanner
}

// NewScanner 创建套利策略扫描器
func NewScanner() *Scanner {
	configManager := config.NewManager()
	triangleScanner := triangle.NewScanner()
	crossExchangeScanner := cross_exchange.NewScanner()
	spotFutureScanner := spot_future.NewScanner()
	statisticalScanner := statistical.NewScanner()

	return &Scanner{
		configManager:        configManager,
		triangleScanner:      triangleScanner,
		crossExchangeScanner: crossExchangeScanner,
		spotFutureScanner:    spotFutureScanner,
		statisticalScanner:   statisticalScanner,
	}
}

// Scan 执行套利策略扫描
func (s *Scanner) Scan(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	log.Printf("[Arbitrage] 开始扫描套利策略")

	var allResults []arbitrage.ValidationResult

	// 根据配置执行不同类型的套利扫描
	if config.TriangleArbEnabled {
		results, err := s.ScanTriangleArbitrage(ctx, config)
		if err != nil {
			log.Printf("[Arbitrage] 三角套利扫描失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	if config.CrossExchangeArbEnabled {
		results, err := s.ScanCrossExchangeArbitrage(ctx, config)
		if err != nil {
			log.Printf("[Arbitrage] 跨交易所套利扫描失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	if config.SpotFutureArbEnabled {
		results, err := s.ScanSpotFutureArbitrage(ctx, config)
		if err != nil {
			log.Printf("[Arbitrage] 现货期货套利扫描失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	if config.StatArbEnabled {
		results, err := s.ScanStatisticalArbitrage(ctx, config)
		if err != nil {
			log.Printf("[Arbitrage] 统计套利扫描失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	log.Printf("[Arbitrage] 套利策略扫描完成，共找到%d个套利机会", len(allResults))
	return allResults, nil
}

// ScanTriangleArbitrage 扫描三角套利机会
func (s *Scanner) ScanTriangleArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	log.Printf("[Arbitrage] 扫描三角套利机会")

	// TODO: 获取基础交易对列表
	baseSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"} // 临时列表

	// 查找套利路径
	paths, err := s.triangleScanner.FindArbitragePaths(ctx, baseSymbols)
	if err != nil {
		return nil, err
	}

	log.Printf("[Arbitrage] 找到%d个三角套利路径", len(paths))

	var results []arbitrage.ValidationResult
	for _, path := range paths {
		// 计算套利利润
		profitPercent, err := s.triangleScanner.CalculateTriangleProfit(path.Path, []float64{1000.0}) // 假设1000USDT本金
		if err != nil {
			continue
		}

		// 验证是否满足最小利润阈值
		if profitPercent >= config.MinProfitThreshold {
			result := arbitrage.ValidationResult{
				Opportunity: &arbitrage.ArbitrageOpportunity{
					Symbol:        path.Path[0],
					Type:          "triangle",
					ProfitPercent: profitPercent,
					Path:          path.Path,
				},
				Action:    "triangle_arbitrage",
				Reason:    fmt.Sprintf("三角套利机会，预计利润: %.2f%%", profitPercent*100),
				IsValid:   true,
				RiskLevel: "medium",
				Score:     0.8,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// ScanCrossExchangeArbitrage 扫描跨交易所套利机会
func (s *Scanner) ScanCrossExchangeArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	log.Printf("[Arbitrage] 扫描跨交易所套利机会")

	// TODO: 获取需要扫描的交易对列表
	symbols := []string{"BTCUSDT", "ETHUSDT"} // 临时列表

	var results []arbitrage.ValidationResult
	for _, symbol := range symbols {
		opportunities, err := s.crossExchangeScanner.CompareExchangePrices(ctx, symbol, config.ExchangePairs)
		if err != nil {
			continue
		}

		for _, opportunity := range opportunities {
			if opportunity.ProfitPercent >= config.MinProfitThreshold {
				result := arbitrage.ValidationResult{
					Opportunity: &opportunity, // 使用现有的opportunity
					Action:      "cross_exchange_arbitrage",
					Reason:      fmt.Sprintf("跨交易所套利，利润: %.2f%%", opportunity.ProfitPercent*100),
					IsValid:     true,
					RiskLevel:   "low",
					Score:       0.9,
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// ScanSpotFutureArbitrage 扫描现货期货套利机会
func (s *Scanner) ScanSpotFutureArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	log.Printf("[Arbitrage] 扫描现货期货套利机会")

	// TODO: 获取需要扫描的期货交易对列表
	symbols := []string{"BTCUSDT", "ETHUSDT"} // 临时列表

	var results []arbitrage.ValidationResult
	for _, symbol := range symbols {
		opportunities, err := s.spotFutureScanner.CompareSpotFuturePrices(ctx, symbol)
		if err != nil {
			continue
		}

		for _, opportunity := range opportunities {
			if opportunity.ProfitPercent >= config.MinProfitThreshold {
				result := arbitrage.ValidationResult{
					Opportunity: &opportunity, // 使用现有的opportunity
					Action:      "spot_future_arbitrage",
					Reason:      fmt.Sprintf("现货期货套利，利润: %.2f%%", opportunity.ProfitPercent*100),
					IsValid:     true,
					RiskLevel:   "medium",
					Score:       0.7,
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// ScanStatisticalArbitrage 扫描统计套利机会
func (s *Scanner) ScanStatisticalArbitrage(ctx context.Context, config *arbitrage.ArbitrageConfig) ([]arbitrage.ValidationResult, error) {
	log.Printf("[Arbitrage] 扫描统计套利机会")

	// TODO: 获取需要分析的交易对组合
	symbols := [][]string{
		{"BTCUSDT", "ETHUSDT"},
		{"ETHUSDT", "BNBUSDT"},
	} // 临时组合列表

	var results []arbitrage.ValidationResult
	for _, symbolPair := range symbols {
		opportunities, err := s.statisticalScanner.FindStatArbOpportunities(ctx, symbolPair, config)
		if err != nil {
			continue
		}

		for _, opportunity := range opportunities {
			if opportunity.ProfitPercent >= config.MinProfitThreshold {
				result := arbitrage.ValidationResult{
					Opportunity: &opportunity, // 使用现有的opportunity
					Action:      "statistical_arbitrage",
					Reason:      fmt.Sprintf("统计套利，价差: %.2f%%", opportunity.ProfitPercent*100),
					IsValid:     true,
					RiskLevel:   "high",
					Score:       0.6,
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
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
	arbConfig := configManager.ConvertConfig(tradingStrategy.Conditions)

	// 执行扫描
	results, err := a.scanner.Scan(ctx, arbConfig)
	if err != nil {
		return nil, err
	}

	// 转换结果为interface{}切片
	eligibleSymbols := make([]interface{}, 0, len(results))
	for _, result := range results {
		if result.IsValid {
			symbol := result.Opportunity.Symbol
			symbolMap := map[string]interface{}{
				"symbol":       symbol,
				"action":       result.Action,
				"reason":       result.Reason,
				"multiplier":   1.0,
				"market_cap":   0.0, // 套利策略不依赖市值
				"gainers_rank": 0,
			}
			eligibleSymbols = append(eligibleSymbols, symbolMap)
		}
	}

	return eligibleSymbols, nil
}

// GetStrategyType 获取策略类型
func (a *strategyScannerAdapter) GetStrategyType() string {
	return "arbitrage"
}

// ToStrategyScanner 创建适配器
func (s *Scanner) ToStrategyScanner() interface{} {
	return &strategyScannerAdapter{scanner: s}
}

// ============================================================================
// 注册表
// ============================================================================

var globalScanner *Scanner

// GetArbitrageScanner 获取套利策略扫描器实例
func GetArbitrageScanner() *Scanner {
	if globalScanner == nil {
		globalScanner = NewScanner()
	}
	return globalScanner
}
