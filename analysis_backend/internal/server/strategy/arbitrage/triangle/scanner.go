package triangle

import (
	"analysis/internal/server/strategy/arbitrage"
	"context"
	"fmt"
	"log"
)

// Scanner 三角套利扫描器实现
type Scanner struct {
	// TODO: 注入依赖，如数据提供者等
}

// NewScanner 创建三角套利扫描器
func NewScanner() arbitrage.TriangleArbitrageScanner {
	return &Scanner{}
}

// FindArbitragePaths 查找套利路径
func (s *Scanner) FindArbitragePaths(ctx context.Context, baseSymbols []string) ([]arbitrage.TrianglePath, error) {
	var paths []arbitrage.TrianglePath

	// 从基础交易对构建所有可能的三角路径
	for _, base := range baseSymbols {
		// 为每个基础货币构建三角路径
		basePaths := s.buildTrianglePathsForBase(base)
		paths = append(paths, basePaths...)
	}

	log.Printf("[TriangleArbitrage] 找到%d个潜在的三角路径", len(paths))
	return paths, nil
}

// buildTrianglePathsForBase 为基础货币构建三角路径
func (s *Scanner) buildTrianglePathsForBase(baseSymbol string) []arbitrage.TrianglePath {
	var paths []arbitrage.TrianglePath

	// 常见的三角套利模式：
	// 模式1: BASE/QUOTE -> ALT/BASE -> ALT/QUOTE
	// 例如: BTC/USDT -> ETH/BTC -> ETH/USDT

	// 模式2: BASE/QUOTE -> QUOTE/ALT -> BASE/ALT
	// 例如: BTC/USDT -> USDT/ETH -> BTC/ETH

	// 这里我们实现一个简化的版本
	commonAlts := []string{"ETH", "BNB", "ADA", "SOL", "DOT"}

	for _, alt := range commonAlts {
		if alt == baseSymbol {
			continue
		}

		// 构建路径: BASE/USDT -> ALT/BASE -> ALT/USDT
		path1 := []string{
			fmt.Sprintf("%sUSDT", baseSymbol),
			fmt.Sprintf("%s%s", alt, baseSymbol),
			fmt.Sprintf("%sUSDT", alt),
		}
		paths = append(paths, arbitrage.TrianglePath{
			Path:        path1,
			StartAmount: 1000.0, // 起始金额1000 USDT
		})

		// 构建路径: BASE/USDT -> USDT/ALT -> BASE/ALT
		path2 := []string{
			fmt.Sprintf("%sUSDT", baseSymbol),
			fmt.Sprintf("USDT%s", alt),
			fmt.Sprintf("%s%s", baseSymbol, alt),
		}
		paths = append(paths, arbitrage.TrianglePath{
			Path:        path2,
			StartAmount: 1000.0,
		})
	}

	return paths
}

// ValidateTrianglePath 验证三角套利路径
func (s *Scanner) ValidateTrianglePath(ctx context.Context, path arbitrage.TrianglePath, config *arbitrage.ArbitrageConfig) (*arbitrage.ValidationResult, error) {
	result := &arbitrage.ValidationResult{
		Opportunity: &arbitrage.ArbitrageOpportunity{
			Type:         "triangle",
			Path:         path.Path,
			ProfitAmount: 0,
			Volume:       1000.0,
			Confidence:   0.0,
			Timestamp:    0,
		},
		IsValid:   false,
		Action:    "arbitrage",
		RiskLevel: "medium",
		Score:     0.0,
	}

	// 计算利润
	amounts := []float64{1.0, 1.0, 1.0} // 简化的交易比例
	profitPercent, err := s.CalculateTriangleProfit(path.Path, amounts)
	if err != nil {
		result.Reason = fmt.Sprintf("利润计算失败: %v", err)
		return result, nil
	}

	result.Opportunity.ProfitPercent = profitPercent
	result.Opportunity.ProfitAmount = path.StartAmount * profitPercent / 100

	// 检查是否满足最小利润要求
	if profitPercent < config.TriangleMinProfitPercent {
		result.Reason = fmt.Sprintf("利润不足: %.4f%% (需要>=%.2f%%)",
			profitPercent, config.TriangleMinProfitPercent)
		return result, nil
	}

	// 检查是否超过滑点限制
	if profitPercent < config.MaxSlippagePercent {
		result.Reason = fmt.Sprintf("利润低于滑点风险: %.4f%% (滑点%.2f%%)",
			profitPercent, config.MaxSlippagePercent)
		result.RiskLevel = "high"
		return result, nil
	}

	// 计算置信度和评分
	result.Opportunity.Confidence = s.calculateTriangleConfidence(profitPercent, config)
	result.Score = result.Opportunity.Confidence

	// 最终验证
	result.IsValid = true
	result.Reason = fmt.Sprintf("三角套利机会: 利润%.4f%%, 置信度%.2f",
		profitPercent, result.Opportunity.Confidence)

	return result, nil
}

// calculateTriangleConfidence 计算三角套利置信度
func (s *Scanner) calculateTriangleConfidence(profitPercent float64, config *arbitrage.ArbitrageConfig) float64 {
	// 基于利润百分比计算置信度
	if profitPercent >= 1.0 {
		return 0.9 // 高利润，高置信度
	} else if profitPercent >= 0.5 {
		return 0.7 // 中等利润，中等置信度
	} else if profitPercent >= 0.3 {
		return 0.5 // 低利润，低置信度
	}

	return 0.2 // 非常低的利润
}

// CalculateTriangleProfit 计算三角套利利润
func (s *Scanner) CalculateTriangleProfit(path []string, amounts []float64) (float64, error) {
	if len(path) != 3 || len(amounts) != 3 {
		return 0.0, fmt.Errorf("路径或金额数据不完整")
	}

	// 简化的利润计算（实际应该从市场数据获取实时价格）
	// 这里使用模拟的价格数据

	// 起始金额
	startAmount := 1000.0
	currentAmount := startAmount

	// 模拟交易过程
	for i, pair := range path {
		price := s.getSimulatedPrice(pair)
		fee := amounts[i] * 0.001 // 假设0.1%的交易费

		if i%2 == 0 {
			// 买入操作
			currentAmount = (currentAmount - fee) / price * amounts[i]
		} else {
			// 卖出操作
			currentAmount = (currentAmount - fee) * price / amounts[i]
		}
	}

	// 计算利润百分比
	profitPercent := ((currentAmount - startAmount) / startAmount) * 100

	return profitPercent, nil
}

// getSimulatedPrice 获取模拟价格（实际应该从市场数据获取）
func (s *Scanner) getSimulatedPrice(pair string) float64 {
	// 模拟价格数据
	prices := map[string]float64{
		"BTCUSDT": 50000.0,
		"ETHBTC":  0.03,
		"ETHUSDT": 1500.0,
		"BNBUSDT": 300.0,
		"ADAUSDT": 1.2,
		"SOLUSDT": 100.0,
		"DOTUSDT": 15.0,
	}

	if price, exists := prices[pair]; exists {
		return price
	}

	// 默认价格
	return 1.0
}
