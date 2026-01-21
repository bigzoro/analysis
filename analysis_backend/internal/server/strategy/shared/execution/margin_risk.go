package execution

import (
	"fmt"
	"log"
	"math"
	"strconv"

	"analysis/internal/exchange/binancefutures"
)

// ============================================================================
// 保证金风险管理器 - 基于保证金亏损的止损逻辑
// ============================================================================

// MarginRiskManager 保证金风险管理器
type MarginRiskManager struct {
	exchangeClient *binancefutures.Client
}

// NewMarginRiskManager 创建保证金风险管理器
func NewMarginRiskManager(client *binancefutures.Client) *MarginRiskManager {
	return &MarginRiskManager{
		exchangeClient: client,
	}
}

// CalculateMarginStopLoss 计算保证金亏损止损价格
// 当保证金亏损达到指定百分比时，计算对应的止损价格
func (mrm *MarginRiskManager) CalculateMarginStopLoss(symbol string, marginLossPercent float64) (float64, error) {
	if marginLossPercent <= 0 || marginLossPercent > 100 {
		return 0, fmt.Errorf("保证金亏损百分比必须在0-100之间: %.2f", marginLossPercent)
	}

	// 1. 获取当前持仓信息
	positions, err := mrm.exchangeClient.GetPositions()
	if err != nil {
		return 0, fmt.Errorf("获取持仓信息失败: %v", err)
	}

	// 2. 找到指定交易对的持仓
	var targetPosition *binancefutures.Position
	for _, pos := range positions {
		if pos.Symbol == symbol {
			targetPosition = &pos
			break
		}
	}

	if targetPosition == nil {
		return 0, fmt.Errorf("未找到%s的持仓信息", symbol)
	}

	// 3. 解析持仓数据
	entryPrice, err := strconv.ParseFloat(targetPosition.EntryPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("解析入场价格失败: %v", err)
	}

	positionAmt, err := strconv.ParseFloat(targetPosition.PositionAmt, 64)
	if err != nil {
		return 0, fmt.Errorf("解析持仓数量失败: %v", err)
	}

	unrealizedProfit, err := strconv.ParseFloat(targetPosition.UnRealizedProfit, 64)
	if err != nil {
		return 0, fmt.Errorf("解析未实现盈亏失败: %v", err)
	}

	// 计算名义价值（持仓价值）
	notional, err := strconv.ParseFloat(targetPosition.Notional, 64)
	if err != nil {
		return 0, fmt.Errorf("解析名义价值失败: %v", err)
	}

	leverage, err := strconv.ParseFloat(targetPosition.Leverage, 64)
	if err != nil {
		return 0, fmt.Errorf("解析杠杆倍数失败: %v", err)
	}

	// 计算初始保证金 = 名义价值 / 杠杆
	initialMargin := notional / leverage

	// 4. 计算当前保证金亏损比例
	currentLossPercent := 0.0
	if initialMargin > 0 {
		currentLossPercent = (unrealizedProfit / initialMargin) * 100
	}

	log.Printf("[MarginRiskManager] %s 当前状态 - 入场价格:%.4f, 持仓量:%.4f, 未实现盈亏:%.4f, 初始保证金:%.4f, 当前亏损比例:%.2f%%",
		symbol, entryPrice, positionAmt, unrealizedProfit, initialMargin, currentLossPercent)

	// 5. 计算目标亏损金额
	targetLoss := initialMargin * (marginLossPercent / 100)

	// 6. 计算需要达到目标亏损的价格变动
	// 价格变动 = (目标亏损 - 当前未实现亏损) / |持仓数量|
	remainingLoss := targetLoss - unrealizedProfit
	priceChange := remainingLoss / math.Abs(positionAmt)

	// 7. 根据持仓方向计算止损价格
	var stopPrice float64
	if positionAmt < 0 {
		// 空头仓位：亏损时价格上涨，止损价格 = 入场价格 + 价格变动
		stopPrice = entryPrice + priceChange
		log.Printf("[MarginRiskManager] %s 空头仓位止损计算 - 目标亏损:%.4f, 剩余亏损:%.4f, 价格变动:%.4f, 止损价格:%.4f",
			symbol, targetLoss, remainingLoss, priceChange, stopPrice)
	} else {
		// 多头仓位：亏损时价格下跌，止损价格 = 入场价格 - 价格变动
		stopPrice = entryPrice - priceChange
		log.Printf("[MarginRiskManager] %s 多头仓位止损计算 - 目标亏损:%.4f, 剩余亏损:%.4f, 价格变动:%.4f, 止损价格:%.4f",
			symbol, targetLoss, remainingLoss, priceChange, stopPrice)
	}

	return stopPrice, nil
}

// CheckMarginLoss 检查是否达到保证金亏损阈值
// 返回：是否触发止损，当前亏损比例，错误
func (mrm *MarginRiskManager) CheckMarginLoss(symbol string, marginLossPercent float64) (bool, float64, error) {
	if marginLossPercent <= 0 || marginLossPercent > 100 {
		return false, 0, fmt.Errorf("保证金亏损百分比必须在0-100之间: %.2f", marginLossPercent)
	}

	// 1. 获取当前持仓信息
	positions, err := mrm.exchangeClient.GetPositions()
	if err != nil {
		return false, 0, fmt.Errorf("获取持仓信息失败: %v", err)
	}

	// 2. 找到指定交易对的持仓
	var targetPosition *binancefutures.Position
	for _, pos := range positions {
		if pos.Symbol == symbol {
			targetPosition = &pos
			break
		}
	}

	if targetPosition == nil {
		return false, 0, fmt.Errorf("未找到%s的持仓信息", symbol)
	}

	// 3. 解析持仓数据
	unrealizedProfit, err := strconv.ParseFloat(targetPosition.UnRealizedProfit, 64)
	if err != nil {
		return false, 0, fmt.Errorf("解析未实现盈亏失败: %v", err)
	}

	// 计算名义价值和杠杆来得到初始保证金
	notional, err := strconv.ParseFloat(targetPosition.Notional, 64)
	if err != nil {
		return false, 0, fmt.Errorf("解析名义价值失败: %v", err)
	}

	leverage, err := strconv.ParseFloat(targetPosition.Leverage, 64)
	if err != nil {
		return false, 0, fmt.Errorf("解析杠杆倍数失败: %v", err)
	}

	initialMargin := notional / leverage

	// 4. 计算当前保证金亏损比例
	currentLossPercent := 0.0
	if initialMargin > 0 {
		currentLossPercent = (unrealizedProfit / initialMargin) * 100
	}

	// 5. 检查是否触发止损
	shouldStopLoss := currentLossPercent <= -marginLossPercent // 负值表示亏损

	log.Printf("[MarginRiskManager] %s 保证金检查 - 当前亏损比例:%.2f%%, 阈值:%.2f%%, 触发止损:%v",
		symbol, currentLossPercent, marginLossPercent, shouldStopLoss)

	return shouldStopLoss, currentLossPercent, nil
}

// GetPositionMarginInfo 获取持仓保证金信息
func (mrm *MarginRiskManager) GetPositionMarginInfo(symbol string) (*PositionMarginInfo, error) {
	// 1. 获取当前持仓信息
	positions, err := mrm.exchangeClient.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓信息失败: %v", err)
	}

	// 2. 找到指定交易对的持仓
	var targetPosition *binancefutures.Position
	for _, pos := range positions {
		if pos.Symbol == symbol {
			targetPosition = &pos
			break
		}
	}

	if targetPosition == nil {
		return nil, fmt.Errorf("未找到%s的持仓信息", symbol)
	}

	// 3. 解析所有相关数据
	entryPrice, _ := strconv.ParseFloat(targetPosition.EntryPrice, 64)
	positionAmt, _ := strconv.ParseFloat(targetPosition.PositionAmt, 64)
	unrealizedProfit, _ := strconv.ParseFloat(targetPosition.UnRealizedProfit, 64)
	leverage, _ := strconv.ParseFloat(targetPosition.Leverage, 64)
	notional, _ := strconv.ParseFloat(targetPosition.Notional, 64)

	// 计算初始保证金 = 名义价值 / 杠杆
	initialMargin := notional / leverage

	// 4. 计算保证金相关指标
	marginLossPercent := 0.0
	if initialMargin > 0 {
		marginLossPercent = (unrealizedProfit / initialMargin) * 100
	}

	// 检查是否为逐仓模式
	isIsolated := targetPosition.MarginType == "isolated"

	return &PositionMarginInfo{
		Symbol:            symbol,
		EntryPrice:        entryPrice,
		PositionAmount:    positionAmt,
		UnrealizedProfit:  unrealizedProfit,
		InitialMargin:     initialMargin,
		MaintMargin:       0, // Binance API中没有直接提供，需要计算或设为0
		Leverage:          leverage,
		MarginLossPercent: marginLossPercent,
		IsIsolated:        isIsolated,
	}, nil
}

// ValidateMarginStopLossConfig 验证保证金止损配置
func (mrm *MarginRiskManager) ValidateMarginStopLossConfig(marginLossPercent float64) error {
	if marginLossPercent <= 0 {
		return fmt.Errorf("保证金亏损百分比必须大于0")
	}
	if marginLossPercent > 100 {
		return fmt.Errorf("保证金亏损百分比不能超过100%%")
	}
	if marginLossPercent < 5 {
		return fmt.Errorf("保证金亏损百分比建议不低于5%%以避免过度敏感")
	}
	return nil
}

// CalculateMarginTakeProfit 计算保证金盈利止盈价格
// 当保证金盈利达到指定百分比时，计算对应的止盈价格
func (mrm *MarginRiskManager) CalculateMarginTakeProfit(symbol string, marginProfitPercent float64) (float64, error) {
	if marginProfitPercent <= 0 {
		return 0, fmt.Errorf("保证金盈利百分比必须大于0: %.2f", marginProfitPercent)
	}

	// 1. 获取当前持仓信息
	positions, err := mrm.exchangeClient.GetPositions()
	if err != nil {
		return 0, fmt.Errorf("获取持仓信息失败: %v", err)
	}

	// 2. 找到指定交易对的持仓
	var targetPosition *binancefutures.Position
	for _, pos := range positions {
		if pos.Symbol == symbol {
			targetPosition = &pos
			break
		}
	}

	if targetPosition == nil {
		return 0, fmt.Errorf("未找到%s的持仓信息", symbol)
	}

	// 3. 解析持仓数据
	entryPrice, err := strconv.ParseFloat(targetPosition.EntryPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("解析入场价格失败: %v", err)
	}

	positionAmt, err := strconv.ParseFloat(targetPosition.PositionAmt, 64)
	if err != nil {
		return 0, fmt.Errorf("解析持仓数量失败: %v", err)
	}

	unrealizedProfit, err := strconv.ParseFloat(targetPosition.UnRealizedProfit, 64)
	if err != nil {
		return 0, fmt.Errorf("解析未实现盈亏失败: %v", err)
	}

	// 计算名义价值（持仓价值）
	notional, err := strconv.ParseFloat(targetPosition.Notional, 64)
	if err != nil {
		return 0, fmt.Errorf("解析名义价值失败: %v", err)
	}

	leverage, err := strconv.ParseFloat(targetPosition.Leverage, 64)
	if err != nil {
		return 0, fmt.Errorf("解析杠杆倍数失败: %v", err)
	}

	// 计算初始保证金 = 名义价值 / 杠杆
	initialMargin := notional / leverage

	// 4. 计算当前保证金盈利比例
	currentProfitPercent := 0.0
	if initialMargin > 0 {
		currentProfitPercent = (unrealizedProfit / initialMargin) * 100
	}

	log.Printf("[MarginRiskManager] %s 当前状态 - 入场价格:%.4f, 持仓量:%.4f, 未实现盈亏:%.4f, 初始保证金:%.4f, 当前盈利比例:%.2f%%",
		symbol, entryPrice, positionAmt, unrealizedProfit, initialMargin, currentProfitPercent)

	// 5. 计算目标盈利金额
	targetProfit := initialMargin * (marginProfitPercent / 100)

	// 6. 计算需要达到目标盈利的价格变动
	// 价格变动 = (目标盈利 - 当前未实现盈利) / |持仓数量|
	remainingProfit := targetProfit - unrealizedProfit
	priceChange := remainingProfit / math.Abs(positionAmt)

	// 7. 根据持仓方向计算止盈价格
	var takeProfitPrice float64
	if positionAmt > 0 {
		// 多头持仓，盈利时价格上涨
		takeProfitPrice = entryPrice + priceChange
	} else {
		// 空头持仓，盈利时价格下跌
		takeProfitPrice = entryPrice - priceChange
	}

	log.Printf("[MarginRiskManager] %s 保证金止盈计算 - 目标盈利:%.4f, 剩余盈利:%.4f, 价格变动:%.4f, 止盈价格:%.4f",
		symbol, targetProfit, remainingProfit, priceChange, takeProfitPrice)

	return takeProfitPrice, nil
}

// CalculateEstimatedMarginStopLoss 预估保证金亏损止损价格（开仓前使用）
// 基于预期的入场价格、数量和杠杆计算止损价格
func (mrm *MarginRiskManager) CalculateEstimatedMarginStopLoss(expectedEntryPrice, expectedQuantity, leverage, marginLossPercent float64, isLong bool) (float64, error) {
	if marginLossPercent <= 0 || marginLossPercent > 100 {
		return 0, fmt.Errorf("保证金亏损百分比必须在0-100之间: %.2f", marginLossPercent)
	}

	// 计算名义价值
	notional := expectedQuantity * expectedEntryPrice

	// 计算初始保证金 = 名义价值 / 杠杆
	initialMargin := notional / leverage

	// 计算目标亏损金额
	targetLoss := initialMargin * (marginLossPercent / 100)

	// 计算需要达到目标亏损的价格变动
	priceChange := targetLoss / expectedQuantity

	// 根据持仓方向计算止损价格
	var stopPrice float64
	if isLong {
		// 多头仓位：亏损时价格下跌，止损价格 = 入场价格 - 价格变动
		stopPrice = expectedEntryPrice - priceChange
	} else {
		// 空头仓位：亏损时价格上涨，止损价格 = 入场价格 + 价格变动
		stopPrice = expectedEntryPrice + priceChange
	}

	log.Printf("[MarginRiskManager] 预估保证金止损 - 入场价格:%.8f, 数量:%.4f, 杠杆:%.1f, 名义价值:%.4f, 初始保证金:%.4f, 目标亏损:%.4f, 价格变动:%.8f, 止损价格:%.8f",
		expectedEntryPrice, expectedQuantity, leverage, notional, initialMargin, targetLoss, priceChange, stopPrice)

	return stopPrice, nil
}

// CalculateEstimatedMarginTakeProfit 预估保证金盈利止盈价格（开仓前使用）
// 基于预期的入场价格、数量和杠杆计算止盈价格
func (mrm *MarginRiskManager) CalculateEstimatedMarginTakeProfit(expectedEntryPrice, expectedQuantity, leverage, marginProfitPercent float64, isLong bool) (float64, error) {
	if marginProfitPercent <= 0 {
		return 0, fmt.Errorf("保证金盈利百分比必须大于0: %.2f", marginProfitPercent)
	}

	// 计算名义价值
	notional := expectedQuantity * expectedEntryPrice

	// 计算初始保证金 = 名义价值 / 杠杆
	initialMargin := notional / leverage

	// 计算目标盈利金额
	targetProfit := initialMargin * (marginProfitPercent / 100)

	// 计算需要达到目标盈利的价格变动
	priceChange := targetProfit / expectedQuantity

	// 根据持仓方向计算止盈价格
	var takeProfitPrice float64
	if isLong {
		// 多头仓位：盈利时价格上涨，止盈价格 = 入场价格 + 价格变动
		takeProfitPrice = expectedEntryPrice + priceChange
	} else {
		// 空头仓位：盈利时价格下跌，止盈价格 = 入场价格 - 价格变动
		takeProfitPrice = expectedEntryPrice - priceChange
	}

	log.Printf("[MarginRiskManager] 预估保证金止盈 - 入场价格:%.8f, 数量:%.4f, 杠杆:%.1f, 名义价值:%.4f, 初始保证金:%.4f, 目标盈利:%.4f, 价格变动:%.8f, 止盈价格:%.8f",
		expectedEntryPrice, expectedQuantity, leverage, notional, initialMargin, targetProfit, priceChange, takeProfitPrice)

	return takeProfitPrice, nil
}

// CheckMarginTakeProfit 检查是否应该触发保证金盈利止盈
func (mrm *MarginRiskManager) CheckMarginTakeProfit(symbol string, marginProfitPercent float64) (bool, error) {
	if marginProfitPercent <= 0 {
		return false, fmt.Errorf("保证金盈利百分比必须大于0: %.2f", marginProfitPercent)
	}

	// 获取当前持仓信息
	positions, err := mrm.exchangeClient.GetPositions()
	if err != nil {
		return false, fmt.Errorf("获取持仓信息失败: %v", err)
	}

	// 找到指定交易对的持仓
	var targetPosition *binancefutures.Position
	for _, pos := range positions {
		if pos.Symbol == symbol {
			targetPosition = &pos
			break
		}
	}

	if targetPosition == nil {
		return false, fmt.Errorf("未找到%s的持仓信息", symbol)
	}

	// 解析持仓数据
	unrealizedProfit, err := strconv.ParseFloat(targetPosition.UnRealizedProfit, 64)
	if err != nil {
		return false, fmt.Errorf("解析未实现盈亏失败: %v", err)
	}

	// 计算名义价值和杠杆
	notional, err := strconv.ParseFloat(targetPosition.Notional, 64)
	if err != nil {
		return false, fmt.Errorf("解析名义价值失败: %v", err)
	}

	leverage, err := strconv.ParseFloat(targetPosition.Leverage, 64)
	if err != nil {
		return false, fmt.Errorf("解析杠杆倍数失败: %v", err)
	}

	// 计算初始保证金 = 名义价值 / 杠杆
	initialMargin := notional / leverage

	// 计算当前保证金盈利比例
	currentProfitPercent := 0.0
	if initialMargin > 0 {
		currentProfitPercent = (unrealizedProfit / initialMargin) * 100
	}

	log.Printf("[MarginRiskManager] %s 保证金检查 - 当前盈利比例:%.2f%%, 阈值:%.2f%%, 触发止盈:%v",
		symbol, currentProfitPercent, marginProfitPercent, currentProfitPercent >= marginProfitPercent)

	return currentProfitPercent >= marginProfitPercent, nil
}
