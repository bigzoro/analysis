package validation

import (
	"analysis/internal/server/strategy/traditional"
	"fmt"
	"log"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// Validator 策略验证器实现
type Validator struct {
	db             *gorm.DB
	priceValidator traditional.PriceValidator
}

// NewValidator 创建验证器
func NewValidator(db *gorm.DB, priceValidator traditional.PriceValidator) traditional.StrategyValidator {
	return &Validator{
		db:             db,
		priceValidator: priceValidator,
	}
}

// ValidateForShort 验证开空条件
func (v *Validator) ValidateForShort(candidate *traditional.CandidateWithRank, config *traditional.TraditionalConfig) *traditional.ValidationResult {
	result := &traditional.ValidationResult{
		Symbol:  candidate.Symbol,
		IsValid: false,
		Action:  "short",
	}

	// 检查资金费率过滤条件
	if config.FundingRateFilterEnabled {
		fundingRate := v.getCurrentFundingRate(candidate.Symbol)
		if fundingRate < config.MinFundingRate {
			result.Reason = fmt.Sprintf("资金费率%.4f%%低于最低要求%.4f%%", fundingRate*100, config.MinFundingRate*100)
			return result
		}
	}

	// 检查基本条件
	if !v.priceValidator.ValidatePriceRange(candidate.Price, config) {
		result.Reason = fmt.Sprintf("价格%.4f超出范围[%.6f, %.2f]", candidate.Price, config.MinPriceThreshold, config.MaxPriceThreshold)
		return result
	}

	if !v.priceValidator.ValidateVolume(candidate.Volume, config) {
		result.Reason = fmt.Sprintf("交易量%.0f低于最低阈值%.0f", candidate.Volume, config.MinVolumeThreshold)
		return result
	}

	if !v.priceValidator.ValidateChangePercent(candidate.ChangePercent, config) {
		result.Reason = fmt.Sprintf("涨跌幅%.2f超出范围[%.2f, %.2f]", candidate.ChangePercent, config.MinChangePercent, config.MaxChangePercent)
		return result
	}

	// 检查排名限制
	if candidate.Rank > config.GainersRankLimit {
		result.Reason = fmt.Sprintf("排名%d超过限制%d", candidate.Rank, config.GainersRankLimit)
		return result
	}

	// 计算适应性评分
	result.Score = v.CalculateSuitabilityScore(candidate, config)
	result.IsValid = result.Score >= 0.6 // 评分阈值
	result.Reason = fmt.Sprintf("符合开空条件，评分%.2f", result.Score)

	return result
}

// ValidateForLong 验证开多条件
func (v *Validator) ValidateForLong(candidate *traditional.CandidateWithRank, config *traditional.TraditionalConfig) *traditional.ValidationResult {
	result := &traditional.ValidationResult{
		Symbol:  candidate.Symbol,
		IsValid: false,
		Action:  "long",
	}

	// 检查资金费率过滤条件
	if config.FundingRateFilterEnabled {
		fundingRate := v.getCurrentFundingRate(candidate.Symbol)
		if fundingRate < config.MinFundingRate {
			result.Reason = fmt.Sprintf("资金费率%.4f%%低于最低要求%.4f%%", fundingRate*100, config.MinFundingRate*100)
			return result
		}
	}

	// 检查基本条件
	if !v.priceValidator.ValidatePriceRange(candidate.Price, config) {
		result.Reason = fmt.Sprintf("价格%.4f超出范围[%.6f, %.2f]", candidate.Price, config.MinPriceThreshold, config.MaxPriceThreshold)
		return result
	}

	if !v.priceValidator.ValidateVolume(candidate.Volume, config) {
		result.Reason = fmt.Sprintf("交易量%.0f低于最低阈值%.0f", candidate.Volume, config.MinVolumeThreshold)
		return result
	}

	// 对于开多，涨幅应该相对温和（小幅上涨）
	if candidate.ChangePercent > 5.0 { // 小幅上涨阈值
		result.Reason = fmt.Sprintf("涨幅%.2f过高，不适合开多", candidate.ChangePercent)
		return result
	}

	// 检查排名限制（开多使用不同的排名限制）
	if candidate.Rank > config.GainersRankLimitLong {
		result.Reason = fmt.Sprintf("排名%d超过开多限制%d", candidate.Rank, config.GainersRankLimitLong)
		return result
	}

	// 计算适应性评分
	result.Score = v.CalculateSuitabilityScore(candidate, config)
	result.IsValid = result.Score >= 0.5 // 开多评分阈值略低
	result.Reason = fmt.Sprintf("符合开多条件，评分%.2f", result.Score)

	return result
}

// ValidateForFuturesPriceShort 验证合约涨幅开空策略条件
func (v *Validator) ValidateForFuturesPriceShort(candidate *traditional.CandidateWithRank, config *traditional.TraditionalConfig) *traditional.ValidationResult {
	result := &traditional.ValidationResult{
		Symbol:  candidate.Symbol,
		IsValid: false,
		Action:  "short",
	}

	log.Printf("[TraditionalValidator] 🔍 开始验证合约涨幅开空策略: %s", candidate.Symbol)
	log.Printf("[TraditionalValidator] 📊 合约涨幅开空策略特有条件:")
	log.Printf("[TraditionalValidator]    • 市值 ≥ %.0f万", config.FuturesPriceShortMinMarketCap)
	log.Printf("[TraditionalValidator]    • 涨幅排名 ≤ %d", config.FuturesPriceShortMaxRank)
	log.Printf("[TraditionalValidator]    • 资金费率 ≥ %.4f%%", config.FuturesPriceShortMinFundingRate*100)

	log.Printf("[TraditionalValidator] 📈 币种自身条件:")
	log.Printf("[TraditionalValidator]    • 市值: %.0f万", candidate.MarketCap)
	log.Printf("[TraditionalValidator]    • 涨幅排名: %d", candidate.Rank)

	log.Printf("[TraditionalValidator] 🔧 通用基础条件:")
	log.Printf("[TraditionalValidator]    • 价格范围: [%.6f, %.2f]", config.MinPriceThreshold, config.MaxPriceThreshold)
	log.Printf("[TraditionalValidator]    • 当前价格: %.4f", candidate.Price)

	// 检查市值条件
	log.Printf("[TraditionalValidator] ✅ 检查市值条件: %.0f万 ≥ %.0f万", candidate.MarketCap, config.FuturesPriceShortMinMarketCap)
	if config.FuturesPriceShortMinMarketCap > 0 && candidate.MarketCap < config.FuturesPriceShortMinMarketCap {
		log.Printf("[TraditionalValidator] ❌ 市值条件不满足: %.0f万 < %.0f万", candidate.MarketCap, config.FuturesPriceShortMinMarketCap)
		result.Reason = fmt.Sprintf("市值%.0f万低于最低要求%.0f万", candidate.MarketCap, config.FuturesPriceShortMinMarketCap)
		return result
	}
	log.Printf("[TraditionalValidator] ✅ 市值条件满足")

	// 检查涨幅排名条件
	log.Printf("[TraditionalValidator] ✅ 检查涨幅排名条件: %d ≤ %d", candidate.Rank, config.FuturesPriceShortMaxRank)
	if candidate.Rank > config.FuturesPriceShortMaxRank {
		log.Printf("[TraditionalValidator] ❌ 涨幅排名条件不满足: %d > %d", candidate.Rank, config.FuturesPriceShortMaxRank)
		result.Reason = fmt.Sprintf("涨幅排名%d超出限制%d", candidate.Rank, config.FuturesPriceShortMaxRank)
		return result
	}
	log.Printf("[TraditionalValidator] ✅ 涨幅排名条件满足")

	// 检查资金费率条件
	fundingRate := v.getCurrentFundingRate(candidate.Symbol)
	log.Printf("[TraditionalValidator] ✅ 检查资金费率条件: %.4f%% ≥ %.4f%%", fundingRate*100, config.FuturesPriceShortMinFundingRate*100)
	if fundingRate < config.FuturesPriceShortMinFundingRate {
		log.Printf("[TraditionalValidator] ❌ 资金费率条件不满足: %.4f%% < %.4f%%", fundingRate*100, config.FuturesPriceShortMinFundingRate*100)
		result.Reason = fmt.Sprintf("资金费率%.4f%%低于最低要求%.4f%%", fundingRate*100, config.FuturesPriceShortMinFundingRate*100)
		return result
	}
	log.Printf("[TraditionalValidator] ✅ 资金费率条件满足")

	// 检查基本条件
	log.Printf("[TraditionalValidator] ✅ 检查价格范围条件: %.4f 在 [%.6f, %.2f] 范围内", candidate.Price, config.MinPriceThreshold, config.MaxPriceThreshold)
	if !v.priceValidator.ValidatePriceRange(candidate.Price, config) {
		log.Printf("[TraditionalValidator] ❌ 价格范围条件不满足: %.4f 不在 [%.6f, %.2f] 范围内", candidate.Price, config.MinPriceThreshold, config.MaxPriceThreshold)
		result.Reason = fmt.Sprintf("价格%.4f超出范围[%.6f, %.2f]", candidate.Price, config.MinPriceThreshold, config.MaxPriceThreshold)
		return result
	}
	log.Printf("[TraditionalValidator] ✅ 价格范围条件满足")

	// 计算适应性评分
	result.Score = v.CalculateSuitabilityScore(candidate, config)
	log.Printf("[TraditionalValidator] 📊 计算适应性评分: %.2f (阈值: 0.6)", result.Score)

	result.IsValid = result.Score >= 0.6 // 使用相同的评分阈值
	if result.IsValid {
		log.Printf("[TraditionalValidator] 🎉 验证通过: 符合合约涨幅开空条件")
		result.Reason = fmt.Sprintf("符合合约涨幅开空条件，评分%.2f，资金费率%.4f%%", result.Score, fundingRate*100)
	} else {
		log.Printf("[TraditionalValidator] ❌ 验证失败: 评分%.2f低于阈值0.6", result.Score)
		result.Reason = fmt.Sprintf("适应性评分%.2f低于阈值0.6", result.Score)
	}

	log.Printf("[TraditionalValidator] 📋 验证结果: %s - %s", func() string {
		if result.IsValid {
			return "✅ 通过"
		}
		return "❌ 失败"
	}(), result.Reason)

	return result
}

// CalculateSuitabilityScore 计算适应性评分
func (v *Validator) CalculateSuitabilityScore(candidate *traditional.CandidateWithRank, config *traditional.TraditionalConfig) float64 {
	score := 0.0
	totalWeight := 0.0

	// 价格合理性评分（权重20%）
	priceScore := 1.0
	if candidate.Price < config.MinPriceThreshold*10 { // 太便宜可能有风险
		priceScore = 0.5
	}
	score += priceScore * 0.2
	totalWeight += 0.2

	// 交易量评分（权重30%）
	volumeScore := candidate.Volume / 100000.0 // 标准化到10万为基准
	if volumeScore > 1.0 {
		volumeScore = 1.0
	}
	score += volumeScore * 0.3
	totalWeight += 0.3

	// 排名评分（权重25%）- 排名越前分数越高
	rankScore := 1.0 - float64(candidate.Rank-1)/100.0 // 前100名线性衰减
	if rankScore < 0 {
		rankScore = 0
	}
	score += rankScore * 0.25
	totalWeight += 0.25

	// 涨跌幅合理性评分（权重25%）
	changeScore := 1.0
	absChange := candidate.ChangePercent
	if absChange < 0 {
		absChange = -absChange
	}
	if absChange > 20.0 { // 涨跌幅过大风险较高
		changeScore = 0.3
	} else if absChange > 10.0 {
		changeScore = 0.7
	}
	score += changeScore * 0.25
	totalWeight += 0.25

	if totalWeight == 0 {
		return 0
	}

	return score / totalWeight
}

// ============================================================================
// 价格验证器实现
// ============================================================================

// PriceValidatorImpl 价格验证器实现
type PriceValidatorImpl struct{}

// NewPriceValidator 创建价格验证器
func NewPriceValidator() traditional.PriceValidator {
	return &PriceValidatorImpl{}
}

// ValidatePriceRange 验证价格范围
func (pv *PriceValidatorImpl) ValidatePriceRange(price float64, config *traditional.TraditionalConfig) bool {
	return price >= config.MinPriceThreshold && price <= config.MaxPriceThreshold
}

// ValidateVolume 验证交易量
func (pv *PriceValidatorImpl) ValidateVolume(volume float64, config *traditional.TraditionalConfig) bool {
	return volume >= config.MinVolumeThreshold
}

// ValidateChangePercent 验证涨跌幅
func (pv *PriceValidatorImpl) ValidateChangePercent(changePercent float64, config *traditional.TraditionalConfig) bool {
	return changePercent >= config.MinChangePercent && changePercent <= config.MaxChangePercent
}

// ============================================================================
// 资金费率验证辅助方法
// ============================================================================

// getCurrentFundingRate 获取当前资金费率
// 从数据库中查询最新的资金费率数据
func (v *Validator) getCurrentFundingRate(symbol string) float64 {
	// 默认资金费率（如果查询失败或无数据）
	defaultFundingRate := 0.01 // 1%

	// 从数据库查询最新资金费率
	var fundingRateRecord pdb.BinanceFundingRate
	result := v.db.Where("symbol = ?", symbol).Order("funding_time DESC").First(&fundingRateRecord)

	if result.Error != nil {
		log.Printf("[TraditionalValidator] 查询资金费率失败 %s: %v，使用默认值 %.4f%%", symbol, result.Error, defaultFundingRate*100)
		return defaultFundingRate
	}

	// 检查数据是否过期（超过24小时）
	currentTime := fundingRateRecord.CreatedAt.Unix()
	timeDiff := currentTime - fundingRateRecord.FundingTime
	if timeDiff > 86400 { // 24小时 = 86400秒
		log.Printf("[TraditionalValidator] 资金费率数据过期 %s: %d秒前，使用默认值 %.4f%%", symbol, timeDiff, defaultFundingRate*100)
		return defaultFundingRate
	}

	log.Printf("[TraditionalValidator] 📊 获取到资金费率 %s: %.4f%% (时间戳: %d, 数据新鲜度: %d秒)",
		symbol, fundingRateRecord.FundingRate*100, fundingRateRecord.FundingTime, timeDiff)
	return fundingRateRecord.FundingRate
}
