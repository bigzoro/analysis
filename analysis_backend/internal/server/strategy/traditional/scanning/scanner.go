package scanning

import (
	"analysis/internal/server/strategy/traditional"
	"analysis/internal/server/strategy/traditional/config"
	"analysis/internal/server/strategy/traditional/ranking"
	"analysis/internal/server/strategy/traditional/selection"
	"analysis/internal/server/strategy/traditional/validation"
	"context"
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// Scanner 传统策略扫描器
type Scanner struct {
	configManager     traditional.ConfigManager
	candidateSelector traditional.CandidateSelector
	rankCalculator    traditional.RankCalculator
	strategyValidator traditional.StrategyValidator
	db                *gorm.DB // 数据库连接，用于检查持仓
	userID            uint     // 用户ID，用于检查持仓
}

// NewScanner 创建传统策略扫描器
func NewScanner(db *gorm.DB, userID uint) *Scanner {
	configManager := config.NewManager()
	database := pdb.NewDatabase(db)
	candidateSelector := selection.NewSelector(database)
	rankCalculator := ranking.NewCalculator()
	priceValidator := validation.NewPriceValidator()
	strategyValidator := validation.NewValidator(db, priceValidator)

	return &Scanner{
		configManager:     configManager,
		candidateSelector: candidateSelector,
		rankCalculator:    rankCalculator,
		strategyValidator: strategyValidator,
		db:                db,
		userID:            userID,
	}
}

// Scan 执行传统策略扫描
func (s *Scanner) Scan(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	log.Printf("[Traditional] 开始扫描传统策略候选币种，用户ID: %d", s.userID)

	var allResults []traditional.ValidationResult

	// 执行涨幅榜开空策略
	if config.ShortOnGainers {
		results, err := s.ExecuteShortOnGainers(ctx, config)
		if err != nil {
			log.Printf("[Traditional] 涨幅榜开空策略执行失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	// 执行小幅上涨开多策略
	if config.LongOnSmallGainers {
		results, err := s.ExecuteLongOnSmallGainers(ctx, config)
		if err != nil {
			log.Printf("[Traditional] 小幅上涨开多策略执行失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	// 执行合约涨幅开空策略
	if config.FuturesPriceShortStrategyEnabled {
		results, err := s.ExecuteFuturesPriceShort(ctx, config)
		if err != nil {
			log.Printf("[Traditional] 合约涨幅开空策略执行失败: %v", err)
		} else {
			allResults = append(allResults, results...)
		}
	}

	log.Printf("[Traditional] 传统策略扫描完成，共找到%d个符合条件的币种", len(allResults))
	return allResults, nil
}

// checkHeldPosition 检查用户是否有某个币种的持仓或最近失败记录
func (s *Scanner) checkHeldPosition(symbol string) (bool, error) {
	if s.db == nil {
		return false, nil
	}

	// 1. 检查是否有未平仓的开仓持仓（已完成但未结束的开仓订单）
	// 条件：filled状态 + reduce_only=false + close_order_ids为空（表示未关联平仓订单）
	var positionCount int64
	err := s.db.Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND status = ? AND reduce_only = ? AND (close_order_ids IS NULL OR close_order_ids = '')",
			s.userID, symbol, "filled", false).
		Count(&positionCount).Error

	if err != nil {
		log.Printf("[Traditional] 检查持仓失败 %s: %v", symbol, err)
		return false, err
	}

	if positionCount > 0 {
		log.Printf("[Traditional] 用户已有 %s 未平仓持仓（%d个开仓订单），跳过此币种", symbol, positionCount)
		return true, nil
	}

	// 2. 检查最近24小时内是否有失败的订单记录
	// 如果有失败记录，说明之前尝试过但失败了，应该跳过避免重复失败
	var failedCount int64
	err = s.db.Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND status IN (?, ?) AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)",
			s.userID, symbol, "failed", "canceled").
		Count(&failedCount).Error

	if err != nil {
		log.Printf("[Traditional] 检查失败记录失败 %s: %v", symbol, err)
		// 检查失败时不跳过，继续尝试
		return false, nil
	}

	if failedCount > 0 {
		log.Printf("[Traditional] %s 最近24小时内有%d个失败订单，跳过此币种避免重复失败", symbol, failedCount)
		return true, nil
	}

	return false, nil
}

// filterHeldPositions 过滤掉已有持仓或最近失败记录的币种
func (s *Scanner) filterHeldPositions(results []traditional.ValidationResult, skipHeldPositions bool) []traditional.ValidationResult {
	if !skipHeldPositions {
		return results
	}

	var filtered []traditional.ValidationResult
	for _, result := range results {
		hasPosition, err := s.checkHeldPosition(result.Symbol)
		if err != nil {
			// 如果检查失败，为了安全起见，保留这个币种
			log.Printf("[Traditional] 持仓检查失败，保留币种 %s: %v", result.Symbol, err)
			filtered = append(filtered, result)
			continue
		}

		if !hasPosition {
			filtered = append(filtered, result)
		} else {
			log.Printf("[Traditional] 跳过已有持仓或失败记录的币种: %s", result.Symbol)
		}
	}

	log.Printf("[Traditional] 持仓过滤完成，过滤前: %d个，过滤后: %d个", len(results), len(filtered))
	return filtered
}

// filterSymbolBlacklist 过滤掉黑名单中的币种
func (s *Scanner) filterSymbolBlacklist(results []traditional.ValidationResult, useBlacklist bool, blacklist []string) []traditional.ValidationResult {
	if !useBlacklist || len(blacklist) == 0 {
		return results
	}

	var filtered []traditional.ValidationResult
	for _, result := range results {
		// 检查币种是否在黑名单中
		isBlacklisted := false
		for _, blacklistedSymbol := range blacklist {
			if result.Symbol == blacklistedSymbol {
				isBlacklisted = true
				log.Printf("[Traditional] 跳过黑名单币种: %s", result.Symbol)
				break
			}
		}

		if !isBlacklisted {
			filtered = append(filtered, result)
		}
	}

	log.Printf("[Traditional] 黑名单过滤完成，过滤前: %d个，过滤后: %d个", len(results), len(filtered))
	return filtered
}

// checkRecentCloseOrder 检查指定时间范围内是否有平仓订单
func (s *Scanner) checkRecentCloseOrder(symbol string, timeRange time.Duration) (bool, error) {
	if s.db == nil {
		return false, nil
	}

	// 检查最近N小时内是否有完成的平仓订单
	// 使用UTC时间确保与数据库时区一致（数据库配置loc=UTC）
	// 平仓订单可能有多种完成状态：filled, completed, success
	var closeOrderCount int64
	cutoffTime := time.Now().UTC().Add(-timeRange)

	err := s.db.Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND status IN (?) AND reduce_only = ? AND created_at >= ?",
			s.userID, symbol, []string{"filled", "completed", "success"}, true, cutoffTime).
		Count(&closeOrderCount).Error

	if err != nil {
		log.Printf("[Traditional] 检查24小时内平仓订单失败 %s: %v", symbol, err)
		return false, err
	}

	if closeOrderCount > 0 {
		log.Printf("[Traditional] 发现 %s 在最近时间内有 %d 个平仓订单", symbol, closeOrderCount)
		return true, nil
	}

	return false, nil
}

// filterCloseOrdersWithinHours 过滤掉指定时间内有平仓订单的币种
func (s *Scanner) filterCloseOrdersWithinHours(results []traditional.ValidationResult, skipCloseOrdersHours int) []traditional.ValidationResult {
	if skipCloseOrdersHours <= 0 {
		return results
	}

	var filtered []traditional.ValidationResult
	duration := time.Duration(skipCloseOrdersHours) * time.Hour

	for _, result := range results {
		hasRecentCloseOrder, err := s.checkRecentCloseOrder(result.Symbol, duration)
		if err != nil {
			// 如果检查失败，为了安全起见，保留这个币种
			log.Printf("[Traditional] %d小时平仓订单检查失败，保留币种 %s: %v", skipCloseOrdersHours, result.Symbol, err)
			filtered = append(filtered, result)
			continue
		}

		if !hasRecentCloseOrder {
			filtered = append(filtered, result)
		} else {
			log.Printf("[Traditional] 跳过%d小时内有平仓订单的币种: %s", skipCloseOrdersHours, result.Symbol)
		}
	}

	log.Printf("[Traditional] %d小时平仓订单过滤完成，过滤前: %d个，过滤后: %d个", skipCloseOrdersHours, len(results), len(filtered))
	return filtered
}

// ExecuteShortOnGainers 执行涨幅榜开空策略
func (s *Scanner) ExecuteShortOnGainers(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	log.Printf("[Traditional] 执行涨幅榜开空策略")

	// 选择涨幅榜候选（现货市场涨幅榜）
	candidates, err := s.candidateSelector.SelectGainersWithRank(ctx, config.MaxCandidates, "spot")
	if err != nil {
		log.Printf("[Traditional] 获取涨幅榜候选失败: %v", err)
		return nil, fmt.Errorf("获取涨幅榜候选失败: %w", err)
	}

	// 计算排名
	candidates = s.rankCalculator.CalculateGainersRank(candidates)

	// 按排名过滤
	candidates = s.rankCalculator.FilterByRank(candidates, config.GainersRankLimit)

	log.Printf("[Traditional] 选择了%d个涨幅榜候选进行开空验证", len(candidates))

	// 验证每个候选
	var results []traditional.ValidationResult
	for _, candidate := range candidates {
		result := s.strategyValidator.ValidateForShort(&candidate, config)
		results = append(results, *result)

		// 限制返回数量
		if len(results) >= config.MaxCandidates {
			break
		}
	}

	validCount := 0
	for _, result := range results {
		if result.IsValid {
			validCount++
		}
	}

	log.Printf("[Traditional] 涨幅榜开空策略完成，%d/%d个币种符合条件", validCount, len(results))

	// 过滤掉已有持仓的币种
	results = s.filterHeldPositions(results, config.SkipHeldPositions)

	// 过滤掉24小时内有平仓订单的币种
	results = s.filterCloseOrdersWithinHours(results, config.SkipCloseOrdersHours)

	// 过滤掉黑名单中的币种
	results = s.filterSymbolBlacklist(results, config.UseSymbolBlacklist, config.SymbolBlacklist)

	return results, nil
}

// ExecuteLongOnSmallGainers 执行小幅上涨开多策略
func (s *Scanner) ExecuteLongOnSmallGainers(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	log.Printf("[Traditional] 执行小幅上涨开多策略")

	// 选择小幅上涨候选
	candidates, err := s.candidateSelector.SelectSmallGainersWithRank(ctx, config.MaxCandidates)
	if err != nil {
		log.Printf("[Traditional] 获取小幅上涨候选失败: %v", err)
		return nil, fmt.Errorf("获取小幅上涨候选失败: %w", err)
	}

	// 计算排名
	candidates = s.rankCalculator.CalculateGainersRank(candidates)

	// 按排名过滤
	candidates = s.rankCalculator.FilterByRank(candidates, config.GainersRankLimitLong)

	log.Printf("[Traditional] 选择了%d个小幅上涨候选进行开多验证", len(candidates))

	// 验证每个候选
	var results []traditional.ValidationResult
	for _, candidate := range candidates {
		result := s.strategyValidator.ValidateForLong(&candidate, config)
		results = append(results, *result)

		// 限制返回数量
		if len(results) >= config.MaxCandidates {
			break
		}
	}

	validCount := 0
	for _, result := range results {
		if result.IsValid {
			validCount++
		}
	}

	log.Printf("[Traditional] 小幅上涨开多策略完成，%d/%d个币种符合条件", validCount, len(results))

	// 过滤掉已有持仓的币种
	results = s.filterHeldPositions(results, config.SkipHeldPositions)

	// 过滤掉24小时内有平仓订单的币种
	results = s.filterCloseOrdersWithinHours(results, config.SkipCloseOrdersHours)

	// 过滤掉黑名单中的币种
	results = s.filterSymbolBlacklist(results, config.UseSymbolBlacklist, config.SymbolBlacklist)

	return results, nil
}

// ExecuteFuturesPriceShort 执行合约涨幅开空策略
func (s *Scanner) ExecuteFuturesPriceShort(ctx context.Context, config *traditional.TraditionalConfig) ([]traditional.ValidationResult, error) {
	log.Printf("[Traditional] 执行合约涨幅开空策略")

	// 选择涨幅榜候选（从合约市场涨幅榜选择币种）
	candidates, err := s.candidateSelector.SelectGainersWithRank(ctx, config.MaxCandidates, "futures")
	if err != nil {
		log.Printf("[Traditional] 获取涨幅榜候选失败: %v", err)
		return nil, fmt.Errorf("获取涨幅榜候选失败: %w", err)
	}

	// 计算排名
	candidates = s.rankCalculator.CalculateGainersRank(candidates)

	// 按排名过滤（使用新策略的排名限制）
	candidates = s.rankCalculator.FilterByRank(candidates, config.FuturesPriceShortMaxRank)

	// 按市值过滤（使用新策略的市值限制）
	if config.FuturesPriceShortMinMarketCap > 0 {
		filteredCandidates := make([]traditional.CandidateWithRank, 0)
		for _, candidate := range candidates {
			if candidate.MarketCap >= config.FuturesPriceShortMinMarketCap {
				filteredCandidates = append(filteredCandidates, candidate)
			}
		}
		candidates = filteredCandidates
		log.Printf("[Traditional] 市值过滤后剩余%d个候选（市值≥%.0f万）", len(candidates), config.FuturesPriceShortMinMarketCap)
	}

	log.Printf("[Traditional] 选择了%d个涨幅榜候选进行合约开空验证", len(candidates))

	// 验证每个候选
	var results []traditional.ValidationResult
	for _, candidate := range candidates {
		result := s.strategyValidator.ValidateForFuturesPriceShort(&candidate, config)
		results = append(results, *result)

		// 限制返回数量
		if len(results) >= config.MaxCandidates {
			break
		}
	}

	validCount := 0
	for _, result := range results {
		if result.IsValid {
			validCount++
		}
	}

	log.Printf("[Traditional] 合约涨幅开空策略完成，%d/%d个币种符合条件", validCount, len(results))

	// 过滤掉已有持仓的币种
	results = s.filterHeldPositions(results, config.SkipHeldPositions)

	// 过滤掉24小时内有平仓订单的币种
	results = s.filterCloseOrdersWithinHours(results, config.SkipCloseOrdersHours)

	// 过滤掉黑名单中的币种
	results = s.filterSymbolBlacklist(results, config.UseSymbolBlacklist, config.SymbolBlacklist)

	return results, nil
}

// ============================================================================
// 适配器和注册表
// ============================================================================

// strategyScannerAdapter 策略扫描器适配器
type strategyScannerAdapter struct {
	scanner *Scanner
	db      interface{} // 数据库连接
}

// Scan 实现StrategyScanner接口
func (a *strategyScannerAdapter) Scan(ctx context.Context, tradingStrategy *pdb.TradingStrategy) ([]interface{}, error) {
	// 转换配置
	configManager := config.NewManager()
	traditionalConfig := configManager.ConvertConfig(tradingStrategy.Conditions)

	// 为当前用户创建一个新的扫描器实例（包含数据库连接和用户ID）
	// 将interface{}转换为*gorm.DB
	var db *gorm.DB
	if a.db != nil {
		if gormDB, ok := a.db.(*gorm.DB); ok {
			db = gormDB
		} else {
			log.Printf("[StrategyScannerAdapter] 数据库连接类型转换失败: %T", a.db)
		}
	} else {
		log.Printf("[StrategyScannerAdapter] 数据库连接为空")
	}

	if db == nil {
		return nil, fmt.Errorf("数据库连接无效，无法创建用户扫描器")
	}

	userScanner := NewScanner(db, tradingStrategy.UserID)

	// 执行扫描
	results, err := userScanner.Scan(ctx, traditionalConfig)
	if err != nil {
		return nil, err
	}

	// 转换结果为interface{}切片
	eligibleSymbols := make([]interface{}, 0, len(results))
	for _, result := range results {
		if result.IsValid {
			action := "short"
			if result.Action == "long" {
				action = "long"
			}

			multiplier := 1.0
			// 为合约涨幅开空策略设置杠杆倍数
			if traditionalConfig.FuturesPriceShortStrategyEnabled {
				multiplier = float64(traditionalConfig.FuturesPriceShortLeverage)
			}

			symbolMap := map[string]interface{}{
				"symbol":       result.Symbol,
				"action":       action,
				"reason":       result.Reason,
				"multiplier":   multiplier,
				"market_cap":   0.0, // 暂时设为0，后续可以从候选数据获取
				"gainers_rank": 0,
			}
			eligibleSymbols = append(eligibleSymbols, symbolMap)
		}
	}

	return eligibleSymbols, nil
}

// GetStrategyType 获取策略类型
func (a *strategyScannerAdapter) GetStrategyType() string {
	return "traditional"
}

// ToStrategyScanner 创建适配器
func (s *Scanner) ToStrategyScanner() interface{} {
	return &strategyScannerAdapter{scanner: s, db: s.db}
}

// ============================================================================
// 注册表
// ============================================================================

var globalScanner *Scanner

// GetTraditionalScanner 获取传统策略扫描器实例（已废弃，请使用NewScanner(db, userID)）
func GetTraditionalScanner() *Scanner {
	if globalScanner == nil {
		globalScanner = NewScanner(nil, 0) // 向后兼容，但功能受限
	}
	return globalScanner
}
