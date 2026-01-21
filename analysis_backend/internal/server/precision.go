package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"
)

// PricePrecisionConfig 价格精度配置
type PricePrecisionConfig struct {
	TickSize      float64 // 价格步长
	DecimalPlaces int     // 小数位数
	Description   string  // 配置描述
}

// PrecisionProfile 精度配置档案
type PrecisionProfile struct {
	SymbolPattern string                          // 币种模式匹配
	PriceRanges   []PriceRangeConfig              // 价格范围配置
	SpecialRules  map[string]PricePrecisionConfig // 特殊规则
}

// PriceRangeConfig 价格范围配置
type PriceRangeConfig struct {
	MinPrice float64              // 最小价格
	MaxPrice float64              // 最大价格
	Config   PricePrecisionConfig // 对应配置
}

// getPrecisionProfiles 获取所有精度配置档案
func (s *OrderScheduler) getPrecisionProfiles() []PrecisionProfile {
	return []PrecisionProfile{
		{
			SymbolPattern: "DEFAULT", // 默认配置
			PriceRanges: []PriceRangeConfig{
				{0, 0.000001, PricePrecisionConfig{0.00000001, 8, "极低价-8位小数"}},
				{0.000001, 0.00001, PricePrecisionConfig{0.0000001, 7, "超低价-7位小数"}},
				{0.00001, 0.0001, PricePrecisionConfig{0.000001, 6, "很低价-6位小数"}},
				{0.0001, 0.001, PricePrecisionConfig{0.00001, 5, "低价-5位小数"}},
				{0.001, 0.01, PricePrecisionConfig{0.0001, 4, "中等低价-4位小数"}},
				{0.01, 0.1, PricePrecisionConfig{0.001, 3, "中等价-3位小数"}},
				{0.1, 1.0, PricePrecisionConfig{0.01, 2, "较高价-2位小数"}},
				{1.0, 10.0, PricePrecisionConfig{0.1, 1, "高价-1位小数"}},
				{10.0, 1000.0, PricePrecisionConfig{1.0, 0, "很高价-整数"}},
				{1000.0, 1000000.0, PricePrecisionConfig{10.0, 0, "极高价-10整数倍"}},
			},
			SpecialRules: map[string]PricePrecisionConfig{
				"DEFAULT": {0.01, 2, "默认中等价位"},
			},
		},
	}
}

// getAdaptivePricePrecision 获取智能价格精度配置
func (s *OrderScheduler) getAdaptivePricePrecision(symbol, price string) PricePrecisionConfig {

	// 解析价格
	px, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return PricePrecisionConfig{0.01, 2, "解析失败-默认配置"}
	}

	// 使用配置档案系统
	profiles := s.getPrecisionProfiles()
	for _, profile := range profiles {
		if strings.Contains(strings.ToUpper(symbol), profile.SymbolPattern) ||
			profile.SymbolPattern == "DEFAULT" {

			// 检查特殊规则
			if rule, exists := profile.SpecialRules[strings.ToUpper(symbol)]; exists {
				return rule
			}

			// 检查价格范围
			for _, pr := range profile.PriceRanges {
				if px >= pr.MinPrice && px < pr.MaxPrice {
					return pr.Config
				}
			}

			// 如果没有匹配的价格范围，使用默认规则
			if defaultRule, exists := profile.SpecialRules["DEFAULT"]; exists {
				return defaultRule
			}
		}
	}

	// 最后的fallback
	return PricePrecisionConfig{0.01, 2, "最终fallback配置"}
}

// adjustPricePrecision 根据交易对的价格精度限制调整价格
func (s *OrderScheduler) adjustPricePrecision(symbol, price string) string {
	if price == "" {
		return price
	}

	// 移除特殊币种配置，统一使用API返回的精度信息

	// 获取交易对的价格精度信息和限制
	tickSize, minPrice, maxPrice, err := s.getPriceFilterInfo(symbol)
	if err != nil {
		log.Printf("[scheduler] 获取 %s 价格精度信息失败，使用自适应精度: %v", symbol, err)
		// 使用自适应精度配置而不是直接返回原始价格
		adaptiveConfig := s.getAdaptivePricePrecision(symbol, price)
		tickSize = adaptiveConfig.TickSize
		minPrice = 0.00000001 // 使用合理的默认最小价格
		maxPrice = 999999999  // 使用合理的默认最大价格
		log.Printf("[scheduler] %s 使用自适应精度: %s", symbol, adaptiveConfig.Description)
	}

	// 解析价格
	px, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Printf("[scheduler] 解析价格失败 %s: %v", price, err)
		return price
	}

	// 检查价格是否在允许范围内，但要避免明显不合理的限制
	// 如果minPrice比当前价格高出100倍以上，认为这个限制不合理，忽略它
	minPriceReasonable := minPrice <= px*2 || minPrice <= 100.0

	// 智能检测API数据合理性，如果不合理则使用自适应精度
	apiDataUnreasonable := !minPriceReasonable || tickSize >= 0.01 || tickSize <= 0
	if apiDataUnreasonable {
		log.Printf("[scheduler] %s API数据不合理(tickSize=%.8f, minPrice=%.8f)，使用自适应精度", symbol, tickSize, minPrice)
		// 使用自适应精度配置
		adaptiveConfig := s.getAdaptivePricePrecision(symbol, price)
		tickSize = adaptiveConfig.TickSize
		minPrice = 0.00000001 // 使用合理的默认最小价格
		minPriceReasonable = true
		log.Printf("[scheduler] %s 切换到自适应精度: %s", symbol, adaptiveConfig.Description)
	}

	if minPriceReasonable && px < minPrice {
		log.Printf("[scheduler] %s 价格 %.8f 低于最小限制 %.8f，使用最小价格", symbol, px, minPrice)
		px = minPrice
	} else if px > maxPrice {
		log.Printf("[scheduler] %s 价格 %.8f 超过最大限制 %.8f，使用最大价格", symbol, px, maxPrice)
		px = maxPrice
	} else if !minPriceReasonable {
		log.Printf("[scheduler] %s minPrice %.8f 明显不合理 (当前价格 %.8f)，忽略此限制", symbol, minPrice, px)
	}

	// 根据tickSize调整精度
	adjusted := math.Round(px/tickSize) * tickSize

	// 检查调整后的价格是否合理
	// 如果调整后价格变成0或者比原始价格小太多，说明tickSize太大，使用更保守的调整
	if adjusted <= 0 || adjusted < px*0.1 {
		log.Printf("[scheduler] %s tickSize调整导致不合理价格 %.8f (原始: %.8f)，使用更小的tickSize", symbol, adjusted, px)
		// 使用更小的tickSize进行调整，比如原始tickSize的1/10
		smallerTickSize := tickSize / 10
		adjusted = math.Round(px/smallerTickSize) * smallerTickSize
		log.Printf("[scheduler] %s 使用较小tickSize %.8f调整结果: %.8f", symbol, smallerTickSize, adjusted)
	}

	// 只有在minPrice合理的情况下，才确保调整后的价格不低于最小限制
	if minPriceReasonable && adjusted < minPrice {
		adjusted = minPrice
		// 再次调整精度
		adjusted = math.Round(adjusted/tickSize) * tickSize
	}

	// 转换为字符串，保留适当的小数位数
	precision := s.getDecimalPlaces(tickSize)
	// 对于价格很低的币种，使用更高的精度
	if adjusted < 1.0 && precision < 8 {
		precision = 8 // 确保至少8位小数精度
	}
	result := strconv.FormatFloat(adjusted, 'f', precision, 64)

	minPriceStr := strconv.FormatFloat(minPrice, 'f', -1, 64)
	if !minPriceReasonable {
		minPriceStr += "(忽略)"
	}
	log.Printf("[scheduler] 调整 %s 价格精度: %s -> %s (tickSize: %s, precision: %d, minPrice: %s, maxPrice: %s)",
		symbol, price, result, strconv.FormatFloat(tickSize, 'f', -1, 64), precision,
		minPriceStr, strconv.FormatFloat(maxPrice, 'f', -1, 64))
	return result
}

// adjustPricePrecisionStrict 使用更严格的精度调整价格（用于重试）
func (s *OrderScheduler) adjustPricePrecisionStrict(symbol, price string) string {
	if price == "" {
		return price
	}

	// 获取智能精度配置
	config := s.getAdaptivePricePrecision(symbol, price)

	// 解析价格
	px, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Printf("[scheduler] 解析价格失败 %s: %v", price, err)
		return price
	}

	// 使用配置的精度进行调整
	adjusted := math.Round(px/config.TickSize) * config.TickSize
	result := strconv.FormatFloat(adjusted, 'f', config.DecimalPlaces, 64)

	log.Printf("[scheduler] 严格价格精度调整 %s: %s -> %s (%s)", symbol, price, result, config.Description)
	return result
}

// adjustPriceWithConfig 使用指定的精度配置调整价格
func (s *OrderScheduler) adjustPriceWithConfig(price string, config PricePrecisionConfig) string {
	// 解析价格
	px, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Printf("[scheduler] 解析价格失败 %s: %v", price, err)
		return price
	}

	// 使用配置的精度进行调整
	adjusted := math.Round(px/config.TickSize) * config.TickSize
	result := strconv.FormatFloat(adjusted, 'f', config.DecimalPlaces, 64)

	log.Printf("[scheduler] 特殊配置价格调整 %s: %s -> %s (%s)", "FILUSDT", price, result, config.Description)
	return result
}

// adjustQuantityPrecision 根据交易对的精度限制调整数量
func (s *OrderScheduler) adjustQuantityPrecision(symbol, quantity, orderType string) string {

	// 获取交易对的精度信息和最小名义价值
	stepSize, minNotional, maxQty, _, err := s.getLotSizeAndMinNotional(symbol, orderType)
	if err != nil {
		log.Printf("[scheduler] 获取 %s 精度信息失败，使用保守的默认值进行调整: %v", symbol, err)

		// 使用保守的默认值来避免下单失败
		// 大多数币安期货交易对的stepSize是0.001或0.01，minNotional是5.0
		stepSize = 0.001  // 保守的步长
		minNotional = 5.0 // 保守的最小名义价值
		maxQty = 1000.0   // 保守的最大数量

		log.Printf("[scheduler] 使用默认精度调整: stepSize=%.6f, minNotional=%.2f, maxQty=%.2f", stepSize, minNotional, maxQty)
	}

	// 解析数量
	qty, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		log.Printf("[scheduler] 解析数量失败 %s: %v", quantity, err)
		return quantity
	}

	// 获取当前价格来计算名义价值
	ctx := context.Background()
	currentPrice, err := s.getCurrentPrice(ctx, symbol, "futures")
	if err != nil {
		log.Printf("[scheduler] 获取 %s 当前价格失败，使用原始数量: %v", symbol, err)
		// 如果获取价格失败，至少调整精度
		adjusted := math.Round(qty/stepSize) * stepSize
		precision := s.getDecimalPlaces(stepSize)
		result := strconv.FormatFloat(adjusted, 'f', precision, 64)
		log.Printf("[scheduler] 调整 %s 数量精度: %s -> %s (stepSize: %s)", symbol, quantity, result, strconv.FormatFloat(stepSize, 'f', -1, 64))
		return result
	}

	// 计算当前名义价值
	notionalValue := qty * currentPrice

	// 如果名义价值小于最小要求，调整数量
	if notionalValue < minNotional {
		minQty := minNotional / currentPrice
		// 向上取整到stepSize的倍数
		qty = math.Ceil(minQty/stepSize) * stepSize
		log.Printf("[scheduler] %s 名义价值不足 (%.4f < %.4f)，调整数量到最小值", symbol, notionalValue, minNotional)
	}

	// 检查是否超过最大数量限制
	if maxQty > 0 && qty > maxQty {
		log.Printf("[scheduler] %s 数量超过最大限制 (%.4f > %.4f)，使用最大允许数量", symbol, qty, maxQty)
		qty = maxQty
	}

	// 根据stepSize调整精度
	adjusted := math.Round(qty/stepSize) * stepSize

	// 转换为字符串，保留适当的小数位数
	precision := s.getDecimalPlaces(stepSize)
	result := strconv.FormatFloat(adjusted, 'f', precision, 64)

	log.Printf("[scheduler] 调整 %s 数量精度: %s -> %s (stepSize: %s, minNotional: %.2f, 当前价格: %.6f)",
		symbol, quantity, result, strconv.FormatFloat(stepSize, 'f', -1, 64), minNotional, currentPrice)
	return result
}

// adjustQuantityWithConfig 使用指定的配置调整数量
func (s *OrderScheduler) adjustQuantityWithConfig(quantity string, stepSize, minNotional, maxQty float64, symbol string) string {
	// 解析数量
	qty, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		log.Printf("[scheduler] 解析数量失败 %s: %v", quantity, err)
		return quantity
	}

	// 获取当前价格用于名义价值计算
	currentPrice, err := s.getCurrentPriceFromFutures(context.Background(), symbol)
	if err != nil {
		currentPrice = 1.0 // 默认价格
	}

	// 计算名义价值
	notionalValue := qty * currentPrice

	// 检查最小名义价值
	if notionalValue < minNotional {
		// 计算满足最小名义价值的最小数量
		minQty := math.Ceil(minNotional/currentPrice/stepSize) * stepSize
		qty = minQty
		log.Printf("[scheduler] %s 数量 %.8f 名义价值 %.2f 低于最小要求 %.2f，调整为 %.8f",
			symbol, qty, notionalValue, minNotional, minQty)
	}

	// 检查最大数量
	if qty > maxQty {
		qty = maxQty
		log.Printf("[scheduler] %s 数量 %.8f 超过最大限制 %.8f，调整为最大值", symbol, qty, maxQty)
	}

	// 根据stepSize调整精度
	adjustedQty := math.Floor(qty/stepSize) * stepSize

	// 再次检查最小名义价值（调整精度后）
	finalNotional := adjustedQty * currentPrice
	if finalNotional < minNotional && adjustedQty < maxQty {
		// 如果调整后仍然低于最小名义价值，增加到下一个step
		adjustedQty += stepSize
		log.Printf("[scheduler] %s 调整精度后名义价值 %.2f 仍低于最小要求，增加到 %.8f",
			symbol, finalNotional, adjustedQty)
	}

	result := strconv.FormatFloat(adjustedQty, 'f', -1, 64)
	log.Printf("[scheduler] 特殊配置数量调整 %s: %s -> %s (stepSize: %s, minNotional: %.2f)",
		symbol, quantity, result, strconv.FormatFloat(stepSize, 'f', -1, 64), minNotional)
	return result
}

// autoAdjustQuantityPrecision 自动调整数量精度以避免"Precision is over the maximum defined for this asset"错误
func (s *OrderScheduler) autoAdjustQuantityPrecision(symbol, currentQuantity, orderType string) string {
	log.Printf("[scheduler] 自动调整%s的数量精度: %s", symbol, currentQuantity)

	// 解析当前数量
	currentQty, parseErr := strconv.ParseFloat(currentQuantity, 64)
	if parseErr != nil {
		log.Printf("[scheduler] 解析数量失败: %v", parseErr)
		return currentQuantity
	}

	// 获取原始的stepSize信息
	_, _, maxQty, minQty, err := s.getLotSizeAndMinNotional(symbol, orderType)
	if err != nil {
		log.Printf("[scheduler] 获取%s的精度信息失败，使用原始数量: %v", symbol, err)
		return currentQuantity
	}

	// 尝试使用更大的stepSize（更保守的精度）
	// 从1.0开始尝试，这样可以确保整数结果
	testStepSizes := []float64{1.0, 10.0, 100.0, 1000.0}

	for _, testStepSize := range testStepSizes {
		// 使用新的stepSize调整数量
		adjustedQty := math.Round(currentQty/testStepSize) * testStepSize

		// 确保在合理范围内
		if adjustedQty >= minQty && adjustedQty <= maxQty && adjustedQty > 0 {
			newQuantity := strconv.FormatFloat(adjustedQty, 'f', -1, 64)
			if newQuantity != currentQuantity {
				log.Printf("[scheduler] %s 尝试stepSize %.1f: %s -> %s", symbol, testStepSize, currentQuantity, newQuantity)
				return newQuantity
			}
		}
	}

	log.Printf("[scheduler] %s 无法找到合适的精度调整，使用原始数量", symbol)
	return currentQuantity
}

// validateAndAdjustNotional 智能验证和调整名义价值
func (s *OrderScheduler) validateAndAdjustNotional(symbol, orderType string, qty, notionalPrice float64, currentQuantity string, leverage int) (adjustedQuantity string, skipOrder bool, reason string) {
	finalNotional := qty * notionalPrice

	// 如果名义价值已经满足要求，直接返回
	if finalNotional >= 5.0 {
		return currentQuantity, false, ""
	}

	log.Printf("[scheduler] 名义价值不足 (%.4f < 5.0)，开始智能调整: %s", finalNotional, symbol)

	// 获取交易对的精度信息
	stepSize, _, maxQty, minQty, err := s.getLotSizeAndMinNotional(symbol, orderType)
	if err != nil {
		reason = fmt.Sprintf("名义价值不足无法下单: %s 参考价格%.8f，名义价值只有%.4f USDT，不满足币安最低5 USDT要求。建议选择价格更高的交易对。",
			symbol, notionalPrice, finalNotional)
		return "", true, reason
	}

	// 智能算法：根据币种价格特征和杠杆倍数选择合适的名义价值目标
	targetNotional := s.calculateSmartTargetNotional(notionalPrice, symbol, leverage)

	// 计算需要的精确数量
	requiredQty := targetNotional / notionalPrice

	log.Printf("[scheduler] %s 智能调整 - 价格:%.8f, 目标名义价值:%.1f USDT, 需要数量:%.0f",
		symbol, notionalPrice, targetNotional, requiredQty)

	// 边界检查
	if maxQty > 0 && requiredQty > maxQty {
		// 如果所需数量超过最大限制，尝试使用更小的名义价值目标
		reducedTarget := maxQty * notionalPrice * 0.95 // 使用95%的最大可能名义价值
		if reducedTarget >= 5.0 {
			requiredQty = maxQty
			targetNotional = reducedTarget
			log.Printf("[scheduler] %s 数量超过上限，调整目标名义价值: %.1f USDT", symbol, targetNotional)
		} else {
			reason = fmt.Sprintf("名义价值不足无法下单: %s 参考价格%.8f，即使使用最大数量%.0f也只有%.4f USDT，不满足币安最低5 USDT要求。建议选择价格更高的交易对。",
				symbol, notionalPrice, maxQty, maxQty*notionalPrice)
			return "", true, reason
		}
	}

	if requiredQty < minQty {
		requiredQty = minQty
	}

	// 智能stepSize选择：根据数量大小选择合适的步长
	smartStepSize := s.chooseSmartStepSize(requiredQty, stepSize, symbol)

	// 调整数量到合适的步长倍数
	adjustedQty := math.Ceil(requiredQty/smartStepSize) * smartStepSize

	// 最终边界检查
	if maxQty > 0 && adjustedQty > maxQty {
		adjustedQty = maxQty
	}
	if adjustedQty < minQty {
		adjustedQty = minQty
	}

	// 计算最终名义价值
	finalAdjustedNotional := adjustedQty * notionalPrice

	// 如果调整后的名义价值仍然不足5 USDT，尝试更大的目标
	if finalAdjustedNotional < 5.0 && targetNotional == 5.0 {
		// 尝试10 USDT的目标（对于极低价币种）
		largerTarget := 10.0
		largerRequiredQty := largerTarget / notionalPrice

		if largerRequiredQty <= maxQty || maxQty == 0 {
			adjustedQty = math.Ceil(largerRequiredQty/smartStepSize) * smartStepSize
			if maxQty > 0 && adjustedQty > maxQty {
				adjustedQty = maxQty
			}
			finalAdjustedNotional = adjustedQty * notionalPrice
			log.Printf("[scheduler] %s 扩大目标到%.1f USDT，调整数量: %.0f -> %.0f",
				symbol, largerTarget, requiredQty, adjustedQty)
		}
	}

	// 最终验证
	if finalAdjustedNotional >= 5.0 {
		adjustedQuantity = strconv.FormatFloat(adjustedQty, 'f', -1, 64)
		log.Printf("[scheduler] ✅ 智能调整成功: %s %.0f -> %.0f (名义价值: %.4f -> %.4f USDT)",
			symbol, qty, adjustedQty, finalNotional, finalAdjustedNotional)
		return adjustedQuantity, false, ""
	} else {
		reason = fmt.Sprintf("智能调整失败: %s 无法达到最低名义价值要求。即使调整到%.0f个也只有%.4f USDT。建议选择价格更高的交易对。",
			symbol, adjustedQty, finalAdjustedNotional)
		return "", true, reason
	}
}

// calculateSmartTargetNotional 根据币种价格特征和杠杆倍数计算合适的名义价值目标
func (s *OrderScheduler) calculateSmartTargetNotional(price float64, symbol string, leverage int) float64 {
	// 根据杠杆倍数设置最小保证金目标
	minMarginTarget := 10.0 // 目标保证金至少10 USDT

	// 计算需要的名义价值：保证金 × 杠杆
	baseTarget := minMarginTarget * float64(leverage)

	// 确保名义价值不低于币安最低要求
	baseTarget = math.Max(baseTarget, 5.0)

	// 根据价格区间调整目标 - 更细粒度的分类
	var target float64
	if price < 0.0001 { // 极极低价币种（<0.01美分）
		target = math.Max(baseTarget, 35.0) // 适度提高目标
	} else if price < 0.001 { // 极低价币种（<0.1美分）
		target = math.Max(baseTarget, 30.0) // 提高到30 USDT目标
	} else if price < 0.01 { // 低价币种（<1美分）
		target = math.Max(baseTarget, 20.0) // 稍微提高目标
	} else if price < 0.1 { // 中低价币种（<10美分）
		target = math.Max(baseTarget, 15.0) // 小幅提高目标
	} else if price > 100 { // 高价币种（>100 USDT）
		target = math.Max(baseTarget, 5.0) // 保持最低要求
	} else {
		target = baseTarget // 中等价格币种使用杠杆计算的目标
	}

	// 特殊币种调整
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	if strings.Contains(strings.ToLower(baseSymbol), "shib") || strings.Contains(strings.ToLower(baseSymbol), "doge") {
		target = math.Max(target, baseTarget) // meme币使用杠杆计算的目标
	}

	// 对于特定已知低价币种，进一步调整
	if strings.Contains(strings.ToLower(baseSymbol), "arc") {
		target = math.Max(target, baseTarget+5.0) // ARC特殊处理
	}

	log.Printf("[scheduler] %s 价格=%.8f, 杠杆=%dx, 目标名义价值=%.1f USDT (保证金≥%.1f USDT)",
		symbol, price, leverage, target, target/float64(leverage))

	return target
}

// chooseSmartStepSize 根据数量大小智能选择步长
func (s *OrderScheduler) chooseSmartStepSize(quantity, originalStepSize float64, symbol string) float64 {
	stepSize := originalStepSize

	// 根据数量大小调整步长
	if quantity >= 100000 { // 大数量（10万+）
		stepSize = math.Max(stepSize, 1000) // 使用更大的步长
	} else if quantity >= 10000 { // 中等大数量（1万+）
		stepSize = math.Max(stepSize, 100) // 使用中等步长
	} else if quantity >= 1000 { // 较大数量（1000+）
		stepSize = math.Max(stepSize, 10) // 使用稍大步长
	} else if quantity < 10 { // 小数量
		stepSize = math.Min(stepSize, 1.0) // 保持原有步长或更小
	}

	// 对于极低价币种，考虑使用整数步长
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	if strings.Contains(strings.ToLower(baseSymbol), "arc") ||
		strings.Contains(strings.ToLower(baseSymbol), "zrc") ||
		strings.Contains(strings.ToLower(baseSymbol), "ach") {
		if stepSize < 1.0 {
			stepSize = 1.0 // 强制使用整数精度
		}
	}

	return stepSize
}

// validateAndCorrectFilters 智能验证和修正过滤器数据
func (s *OrderScheduler) validateAndCorrectFilters(symbol string, stepSize, minNotional, maxQty, minQty float64) (correctedStepSize, correctedMinNotional, correctedMaxQty, correctedMinQty float64) {
	originalStepSize, originalMinNotional, originalMaxQty, originalMinQty := stepSize, minNotional, maxQty, minQty

	// 1. 基于交易对类型的智能修正
	if strings.HasSuffix(symbol, "USDT") {
		stepSize, minNotional, maxQty, minQty = s.correctUSDTFilters(symbol, stepSize, minNotional, maxQty, minQty)
	} else if strings.HasSuffix(symbol, "BUSD") || strings.HasSuffix(symbol, "USDC") {
		stepSize, minNotional, maxQty, minQty = s.correctStablecoinFilters(symbol, stepSize, minNotional, maxQty, minQty)
	}

	// 2. 通用验证和修正
	stepSize, minNotional, maxQty, minQty = s.applyUniversalCorrections(symbol, stepSize, minNotional, maxQty, minQty)

	// 3. 设置合理的默认值
	stepSize, minNotional, maxQty, minQty = s.applyDefaultValues(symbol, stepSize, minNotional, maxQty, minQty)

	// 4. 记录修正情况并监控
	if s.hasDataChanged(originalStepSize, originalMinNotional, originalMaxQty, originalMinQty, stepSize, minNotional, maxQty, minQty) {
		s.recordFilterCorrection(symbol, originalStepSize, originalMinNotional, originalMaxQty, originalMinQty, stepSize, minNotional, maxQty, minQty)
	}

	return stepSize, minNotional, maxQty, minQty
}

// correctUSDTFilters 针对USDT交易对的智能修正
func (s *OrderScheduler) correctUSDTFilters(symbol string, stepSize, minNotional, maxQty, minQty float64) (float64, float64, float64, float64) {
	// USDT交易对的常见问题模式

	// 1. 小币种stepSize异常修正 - 已移除硬编码配置

	// 2. minNotional异常值修正
	if minNotional >= 100 {
		log.Printf("[scheduler] USDT修正: %s minNotional %.2f -> 5.0", symbol, minNotional)
		minNotional = 5.0
	}

	// 3. 大币种的特殊处理（如BTCUSDT, ETHUSDT等）
	if s.isLargeCapSymbol(symbol) {
		// 大币种通常不需要特殊修正，保持API数据
	}

	return stepSize, minNotional, maxQty, minQty
}

// correctStablecoinFilters 针对稳定币交易对的智能修正
func (s *OrderScheduler) correctStablecoinFilters(symbol string, stepSize, minNotional, maxQty, minQty float64) (float64, float64, float64, float64) {
	// 稳定币交易对通常有更严格的要求
	if minNotional < 5.0 {
		minNotional = 5.0
	}
	return stepSize, minNotional, maxQty, minQty
}

// applyUniversalCorrections 应用通用修正规则
func (s *OrderScheduler) applyUniversalCorrections(symbol string, stepSize, minNotional, maxQty, minQty float64) (float64, float64, float64, float64) {
	// 1. minNotional范围检查
	if minNotional > 1000 || (minNotional > 0 && minNotional < 1) {
		log.Printf("[scheduler] 通用修正: %s minNotional %.2f -> 5.0", symbol, minNotional)
		minNotional = 5.0
	}

	// 2. stepSize合理性检查
	if stepSize < 0.000001 && stepSize > 0 {
		log.Printf("[scheduler] 通用修正: %s stepSize %.8f -> 1.0", symbol, stepSize)
		stepSize = 1.0
	}

	// 3. maxQty合理性检查
	if maxQty < 0 {
		maxQty = 10000000 // 默认大值
	}

	return stepSize, minNotional, maxQty, minQty
}

// applyDefaultValues 设置默认值
func (s *OrderScheduler) applyDefaultValues(symbol string, stepSize, minNotional, maxQty, minQty float64) (float64, float64, float64, float64) {
	if minNotional == 0 {
		minNotional = 5.0
		log.Printf("[scheduler] 默认值: %s minNotional = 5.0", symbol)
	}
	if stepSize == 0 {
		stepSize = 1.0
		log.Printf("[scheduler] 默认值: %s stepSize = 1.0", symbol)
	}
	if minQty == 0 {
		minQty = 1.0
		log.Printf("[scheduler] 默认值: %s minQty = 1.0", symbol)
	}
	if maxQty == 0 {
		maxQty = 10000000
		log.Printf("[scheduler] 默认值: %s maxQty = 10000000", symbol)
	}

	return stepSize, minNotional, maxQty, minQty
}

// isLargeCapSymbol 检测是否为大币种
func (s *OrderScheduler) isLargeCapSymbol(symbol string) bool {
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	largeCapSymbols := []string{"BTC", "ETH", "BNB", "ADA", "XRP", "SOL", "DOT", "DOGE", "AVAX", "LTC"}

	for _, capSymbol := range largeCapSymbols {
		if baseSymbol == capSymbol {
			return true
		}
	}
	return false
}

// hasDataChanged 检查数据是否发生变化
func (s *OrderScheduler) hasDataChanged(origStep, origMinNotional, origMaxQty, origMinQty, newStep, newMinNotional, newMaxQty, newMinQty float64) bool {
	return origStep != newStep || origMinNotional != newMinNotional ||
		origMaxQty != newMaxQty || origMinQty != newMinQty
}

// recordFilterCorrection 记录过滤器修正情况（用于监控和分析）
func (s *OrderScheduler) recordFilterCorrection(symbol string, origStep, origMinNotional, origMaxQty, origMinQty, newStep, newMinNotional, newMaxQty, newMinQty float64) {
	log.Printf("[scheduler] 过滤器修正记录: %s stepSize=%.6f->%.6f, minNotional=%.2f->%.2f, maxQty=%.0f->%.0f, minQty=%.6f->%.6f",
		symbol, origStep, newStep, origMinNotional, newMinNotional, origMaxQty, newMaxQty, origMinQty, newMinQty)

	// 确定修正类型和原因
	correctionType, correctionReason := s.analyzeCorrectionType(symbol, origStep, origMinNotional, origMaxQty, origMinQty, newStep, newMinNotional, newMaxQty, newMinQty)

	// 创建修正记录
	correction := &pdb.FilterCorrection{
		Symbol:   symbol,
		Exchange: "binance",

		// 原始数据
		OriginalStepSize:    origStep,
		OriginalMinNotional: origMinNotional,
		OriginalMaxQty:      origMaxQty,
		OriginalMinQty:      origMinQty,

		// 修正后数据
		CorrectedStepSize:    newStep,
		CorrectedMinNotional: newMinNotional,
		CorrectedMaxQty:      newMaxQty,
		CorrectedMinQty:      newMinQty,

		// 修正信息
		CorrectionType:   correctionType,
		CorrectionReason: correctionReason,

		// 初始化统计字段
		CorrectionCount: 1,
		LastCorrectedAt: time.Now(),
	}

	// 保存到数据库
	if err := s.db.Create(correction).Error; err != nil {
		log.Printf("[scheduler] 保存过滤器修正记录失败: %v", err)
	}
}

// analyzeCorrectionType 分析修正类型和原因
func (s *OrderScheduler) analyzeCorrectionType(symbol string, origStep, origMinNotional, origMaxQty, origMinQty, newStep, newMinNotional, newMaxQty, newMinQty float64) (string, string) {
	correctionType := "unknown"
	correctionReason := "unknown"

	// 分析修正类型
	if origStep != newStep {
		correctionType = "step_size"
		if newStep > origStep {
			correctionReason = "step_size_too_small"
		} else {
			correctionReason = "step_size_adjustment"
		}
	} else if origMinNotional != newMinNotional {
		correctionType = "min_notional"
		if newMinNotional > origMinNotional {
			correctionReason = "min_notional_too_small"
		} else {
			correctionReason = "min_notional_too_large"
		}
	} else if origMaxQty != newMaxQty {
		correctionType = "max_qty"
		correctionReason = "max_qty_invalid"
	} else if origMinQty != newMinQty {
		correctionType = "min_qty"
		correctionReason = "min_qty_invalid"
	}

	return correctionType, correctionReason
}

// validateAndAdjustTPSLPrices 验证和调整TP/SL价格
func (s *OrderScheduler) validateAndAdjustTPSLPrices(o pdb.ScheduledOrder, tpPrice, slPrice *string, refPx string) error {
	tpVal, tpErr := strconv.ParseFloat(*tpPrice, 64)
	slVal, slErr := strconv.ParseFloat(*slPrice, 64)
	if tpErr != nil || slErr != nil {
		return fmt.Errorf("TP/SL价格解析失败: tp=%v, sl=%v", tpErr, slErr)
	}

	// 获取当前市场价格
	ctx := context.Background()
	currentPrice, priceErr := s.getCurrentPrice(ctx, o.Symbol, "futures")
	if priceErr != nil {
		log.Printf("[scheduler] 获取当前价格失败，使用参考价格进行验证: %v", priceErr)
		currentPrice = 0 // 标记为无效
	} else {
		log.Printf("[scheduler] 当前市场价格: %.8f", currentPrice)
	}

	isBuyOrder := strings.ToUpper(o.Side) == "BUY"
	tpShouldBeAbove := isBuyOrder // BUY订单TP应高于当前价
	slShouldBeBelow := isBuyOrder // BUY订单SL应低于当前价

	// 检查基本合理性
	priceConflict := (isBuyOrder && tpVal <= slVal) || (!isBuyOrder && tpVal >= slVal)
	if priceConflict {
		return fmt.Errorf("%s订单TP/SL价格设置冲突: TP(%.8f) %s SL(%.8f)",
			strings.ToUpper(o.Side), tpVal, ifelse(isBuyOrder, "<=", ">="), slVal)
	}

	// 检查是否会立即触发
	if currentPrice > 0 {
		wouldTrigger := (tpShouldBeAbove && tpVal <= currentPrice) ||
			(!tpShouldBeAbove && tpVal >= currentPrice) ||
			(slShouldBeBelow && slVal >= currentPrice) ||
			(!slShouldBeBelow && slVal <= currentPrice)

		if wouldTrigger {
			log.Printf("[scheduler] 检测到可能立即触发的TP/SL，使用tickSize调整价格")

			// 自动调整价格以避免立即触发
			tickSize, _, _, err := s.getPriceFilterInfo(o.Symbol)
			if err == nil {
				minDistance := tickSize * 3 // 至少3个tick的距离
				requiredTpPercent := (minDistance / currentPrice) * 100
				requiredSlPercent := (minDistance / currentPrice) * 100

				// 调整百分比
				actualTpPercent := math.Max(o.TPPercent, requiredTpPercent)
				actualSlPercent := math.Max(o.SLPercent, requiredSlPercent)
				actualTpPercent = math.Min(actualTpPercent, 50.0) // 最多50%
				actualSlPercent = math.Min(actualSlPercent, 50.0)

				// 重新计算价格
				if isBuyOrder {
					newTpPrice := currentPrice * (1.0 + actualTpPercent/100.0)
					newSlPrice := currentPrice * (1.0 - actualSlPercent/100.0)
					*tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", newTpPrice))
					*slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", newSlPrice))
				} else {
					newTpPrice := currentPrice * (1.0 - actualTpPercent/100.0)
					newSlPrice := currentPrice * (1.0 + actualSlPercent/100.0)
					*tpPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", newTpPrice))
					*slPrice = s.adjustPricePrecision(o.Symbol, fmt.Sprintf("%.8f", newSlPrice))
				}

				log.Printf("[scheduler] 已调整TP/SL价格避免立即触发: TP=%s, SL=%s", *tpPrice, *slPrice)
			}
		}
	}

	return nil
}

// ifelse 简单的三元运算符
func ifelse(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

// getDecimalPlaces 根据stepSize计算需要的小数位数
func (s *OrderScheduler) getDecimalPlaces(stepSize float64) int {
	if stepSize >= 1 {
		return 0
	}

	// 计算小数位数
	str := strconv.FormatFloat(stepSize, 'f', -1, 64)
	parts := strings.Split(str, ".")
	if len(parts) != 2 {
		return 8 // 默认8位小数
	}

	// 移除尾随的0
	decimalPart := strings.TrimRight(parts[1], "0")
	return len(decimalPart)
}
