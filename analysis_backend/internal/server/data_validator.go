package server

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// DataValidator 数据验证器
type DataValidator struct {
	strictMode bool // 严格模式下，任何验证失败都会拒绝数据
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid      bool
	Errors       []string
	Warnings     []string
	CleanedData  interface{} // 清理后的数据
	QualityScore float64     // 数据质量评分 (0-1)
}

// NewDataValidator 创建数据验证器
func NewDataValidator(strictMode bool) *DataValidator {
	return &DataValidator{
		strictMode: strictMode,
	}
}

// ValidateMarketData 验证市场数据
func (dv *DataValidator) ValidateMarketData(data *MarketDataPoint) *ValidationResult {
	result := &ValidationResult{
		IsValid:      true,
		Errors:       make([]string, 0),
		Warnings:     make([]string, 0),
		CleanedData:  *data,
		QualityScore: 1.0,
	}

	cleanedData := *data

	// 1. 验证价格
	if err := dv.validatePrice(data.Price); err != nil {
		if dv.strictMode {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("价格验证失败: %v", err))
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("价格异常: %v", err))
			result.QualityScore *= 0.8
		}
	}

	// 2. 验证价格变化
	if err := dv.validatePriceChange(data.PriceChange24h); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("价格变化异常: %v", err))
		result.QualityScore *= 0.9
	}

	// 3. 验证成交量
	if cleanedVol, err := dv.validateVolume(data.Volume24h); err != nil {
		if dv.strictMode {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("成交量验证失败: %v", err))
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("成交量异常: %v", err))
			cleanedData.Volume24h = cleanedVol
			result.QualityScore *= 0.9
		}
	}

	// 4. 验证市值
	if data.MarketCap != nil {
		if cleanedMC, err := dv.validateMarketCap(*data.MarketCap); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("市值异常: %v", err))
			cleanedMC := cleanedMC
			cleanedData.MarketCap = &cleanedMC
			result.QualityScore *= 0.95
		}
	}

	// 5. 验证时间戳
	if err := dv.validateTimestamp(data.Timestamp); err != nil {
		if dv.strictMode {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("时间戳验证失败: %v", err))
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("时间戳异常: %v", err))
			cleanedData.Timestamp = time.Now() // 使用当前时间
			result.QualityScore *= 0.9
		}
	}

	// 6. 验证币种符号
	if err := dv.validateSymbol(data.Symbol); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("币种符号验证失败: %v", err))
	}

	result.CleanedData = cleanedData
	return result
}

// ValidateTechnicalData 验证技术指标数据
func (dv *DataValidator) ValidateTechnicalData(tech *TechnicalIndicators) *ValidationResult {
	result := &ValidationResult{
		IsValid:      true,
		Errors:       make([]string, 0),
		Warnings:     make([]string, 0),
		CleanedData:  *tech,
		QualityScore: 1.0,
	}

	if tech == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "技术指标数据为空")
		return result
	}

	cleanedData := *tech

	// 验证RSI
	if err := dv.validateRSI(tech.RSI); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("RSI异常: %v", err))
		cleanedData.RSI = dv.cleanRSI(tech.RSI)
		result.QualityScore *= 0.95
	}

	// 验证MACD
	if err := dv.validateMACD(tech.MACD, tech.MACDSignal); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("MACD异常: %v", err))
		result.QualityScore *= 0.9
	}

	// 验证布林带
	if err := dv.validateBollingerBands(tech.BBPosition); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("布林带异常: %v", err))
		cleanedData.BBPosition = dv.cleanBBPosition(tech.BBPosition)
		result.QualityScore *= 0.95
	}

	// 验证均线
	if err := dv.validateMovingAverages(tech.MA5, tech.MA10, tech.MA20); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("均线异常: %v", err))
		result.QualityScore *= 0.9
	}

	result.CleanedData = cleanedData
	return result
}

// ValidateSentimentData 验证情绪数据
func (dv *DataValidator) ValidateSentimentData(sentiment *SentimentResult) *ValidationResult {
	result := &ValidationResult{
		IsValid:      true,
		Errors:       make([]string, 0),
		Warnings:     make([]string, 0),
		CleanedData:  *sentiment,
		QualityScore: 1.0,
	}

	if sentiment == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, "情绪数据为空")
		return result
	}

	// 验证情感评分范围
	if sentiment.Score < -1.0 || sentiment.Score > 1.0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("情感评分超出范围: %.2f", sentiment.Score))
		result.QualityScore *= 0.9
	}

	// 验证提及次数
	if sentiment.Mentions < 0 {
		result.Warnings = append(result.Warnings, "提及次数为负数")
		result.QualityScore *= 0.95
	}

	// 验证趋势
	if sentiment.Trend != "positive" && sentiment.Trend != "negative" && sentiment.Trend != "neutral" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("未知趋势类型: %s", sentiment.Trend))
		result.QualityScore *= 0.9
	}

	return result
}

// 具体的验证函数

func (dv *DataValidator) validatePrice(price float64) error {
	if price <= 0 {
		return fmt.Errorf("价格必须为正数，当前值: %.8f", price)
	}
	if price > 10000000 { // 超过1000万美元的币价不合理
		return fmt.Errorf("价格过高，可能异常: %.2f", price)
	}
	if price < 0.00000001 { // 低于0.00000001的币价太低
		return fmt.Errorf("价格过低，可能异常: %.8f", price)
	}
	return nil
}

func (dv *DataValidator) validatePriceChange(change float64) error {
	if math.Abs(change) > 1000 { // 超过1000%的变化不合理
		return fmt.Errorf("价格变化幅度过大: %.2f%%", change)
	}
	return nil
}

func (dv *DataValidator) validateVolume(volume float64) (float64, error) {
	if volume < 0 {
		return 0, fmt.Errorf("成交量不能为负数: %.2f", volume)
	}
	if volume > 100000000000 { // 超过1万亿美元的成交量不合理
		return volume * 0.1, fmt.Errorf("成交量过大，已按10%%调整: %.2f", volume)
	}
	return volume, nil
}

func (dv *DataValidator) validateMarketCap(marketCap float64) (float64, error) {
	if marketCap < 0 {
		return 0, fmt.Errorf("市值不能为负数: %.2f", marketCap)
	}
	if marketCap > 10000000000000 { // 超过1万亿美元的市值不合理
		return marketCap * 0.1, fmt.Errorf("市值过大，已按10%%调整: %.2f", marketCap)
	}
	return marketCap, nil
}

func (dv *DataValidator) validateTimestamp(ts time.Time) error {
	now := time.Now()
	if ts.After(now.Add(time.Hour)) { // 未来时间
		return fmt.Errorf("时间戳不能是未来时间: %v", ts)
	}
	if ts.Before(now.AddDate(-1, 0, 0)) { // 超过1年的旧数据
		return fmt.Errorf("数据过于陈旧: %v", ts)
	}
	return nil
}

func (dv *DataValidator) validateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("币种符号不能为空")
	}
	if len(symbol) > 20 {
		return fmt.Errorf("币种符号过长: %s", symbol)
	}
	// 检查是否包含非法字符
	if strings.ContainsAny(symbol, " \t\n\r") {
		return fmt.Errorf("币种符号包含非法字符: %s", symbol)
	}
	return nil
}

func (dv *DataValidator) validateRSI(rsi float64) error {
	if rsi < 0 || rsi > 100 {
		return fmt.Errorf("RSI超出有效范围[0,100]: %.2f", rsi)
	}
	return nil
}

func (dv *DataValidator) cleanRSI(rsi float64) float64 {
	if rsi < 0 {
		return 0
	}
	if rsi > 100 {
		return 100
	}
	return rsi
}

func (dv *DataValidator) validateMACD(macd, signal float64) error {
	// MACD值通常在合理范围内
	if math.Abs(macd) > 100000 {
		return fmt.Errorf("MACD值异常: %.2f", macd)
	}
	if math.Abs(signal) > 100000 {
		return fmt.Errorf("MACD信号值异常: %.2f", signal)
	}
	return nil
}

func (dv *DataValidator) validateBollingerBands(position float64) error {
	if position < -1 || position > 1 {
		return fmt.Errorf("布林带位置超出有效范围[-1,1]: %.2f", position)
	}
	return nil
}

func (dv *DataValidator) cleanBBPosition(position float64) float64 {
	if position < -1 {
		return -1
	}
	if position > 1 {
		return 1
	}
	return position
}

func (dv *DataValidator) validateMovingAverages(ma5, ma10, ma20 float64) error {
	if ma5 <= 0 || ma10 <= 0 || ma20 <= 0 {
		return fmt.Errorf("均线值必须为正数: MA5=%.4f, MA10=%.4f, MA20=%.4f", ma5, ma10, ma20)
	}
	return nil
}

// BatchValidate 批量验证数据
func (dv *DataValidator) BatchValidate(marketData []MarketDataPoint) []*ValidationResult {
	results := make([]*ValidationResult, len(marketData))

	for i, data := range marketData {
		results[i] = dv.ValidateMarketData(&data)

		// 记录验证结果
		if !results[i].IsValid {
			log.Printf("[DataValidator] 数据验证失败 - %s: %v", data.Symbol, results[i].Errors)
		} else if len(results[i].Warnings) > 0 {
			log.Printf("[DataValidator] 数据验证警告 - %s: %v", data.Symbol, results[i].Warnings)
		}
	}

	return results
}

// GetValidationStats 获取验证统计信息
func (dv *DataValidator) GetValidationStats(results []*ValidationResult) map[string]interface{} {
	total := len(results)
	valid := 0
	warnings := 0
	errors := 0

	for _, result := range results {
		if result.IsValid {
			valid++
		}
		if len(result.Warnings) > 0 {
			warnings++
		}
		if len(result.Errors) > 0 {
			errors++
		}
	}

	avgQuality := 0.0
	for _, result := range results {
		avgQuality += result.QualityScore
	}
	if total > 0 {
		avgQuality /= float64(total)
	}

	return map[string]interface{}{
		"total_records":         total,
		"valid_records":         valid,
		"records_with_warnings": warnings,
		"records_with_errors":   errors,
		"average_quality_score": avgQuality,
		"validation_rate":       float64(valid) / float64(total),
	}
}
