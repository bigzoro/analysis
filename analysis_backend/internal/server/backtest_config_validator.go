package server

import (
	"fmt"
	"time"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateBacktestConfig 验证回测配置
func (cv *ConfigValidator) ValidateBacktestConfig(config *BacktestConfig) error {
	// 验证基本字段
	if config.Symbol == "" {
		return fmt.Errorf("交易对符号不能为空")
	}

	if config.StartDate.After(config.EndDate) {
		return fmt.Errorf("开始日期不能晚于结束日期")
	}

	if config.EndDate.After(time.Now()) {
		return fmt.Errorf("结束日期不能超过当前时间")
	}

	// 验证时间范围合理性
	duration := config.EndDate.Sub(config.StartDate)
	minDuration := 24 * time.Hour * 30      // 最少30天
	maxDuration := 24 * time.Hour * 365 * 2 // 最多2年

	if duration < minDuration {
		return fmt.Errorf("回测时间范围太短，至少需要%d天", int(minDuration.Hours()/24))
	}

	if duration > maxDuration {
		return fmt.Errorf("回测时间范围太长，最多支持%d天", int(maxDuration.Hours()/24))
	}

	// 验证初始资金
	if config.InitialCash <= 0 {
		return fmt.Errorf("初始资金必须大于0")
	}

	if config.InitialCash > 10000000 { // 1000万美元上限
		return fmt.Errorf("初始资金不能超过1000万美元")
	}

	// 验证策略参数
	if err := cv.validateStrategyConfig(config); err != nil {
		return err
	}

	// 验证风险参数
	if err := cv.validateRiskConfig(config); err != nil {
		return err
	}

	return nil
}

// validateStrategyConfig 验证策略配置
func (cv *ConfigValidator) validateStrategyConfig(config *BacktestConfig) error {
	validStrategies := map[string]bool{
		"buy_and_hold":  true,
		"ml_prediction": true,
		"ensemble":      true,
		"deep_learning": true,
	}

	if !validStrategies[config.Strategy] {
		return fmt.Errorf("不支持的策略类型: %s", config.Strategy)
	}

	// 验证时间框架
	validTimeframes := map[string]bool{
		"1m":  true,
		"5m":  true,
		"15m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
		"1w":  true,
	}

	if config.Timeframe != "" && !validTimeframes[config.Timeframe] {
		return fmt.Errorf("不支持的时间框架: %s", config.Timeframe)
	}

	return nil
}

// validateRiskConfig 验证风险配置
func (cv *ConfigValidator) validateRiskConfig(config *BacktestConfig) error {
	// 验证最大仓位
	if config.MaxPosition <= 0 || config.MaxPosition > 1 {
		return fmt.Errorf("最大仓位必须在0-1之间，当前值: %.2f", config.MaxPosition)
	}

	// 验证止损比例
	if config.StopLoss >= 0 || config.StopLoss < -0.5 {
		return fmt.Errorf("止损比例必须为负数且绝对值不超过50%%，当前值: %.2f", config.StopLoss)
	}

	// 验证止盈比例
	if config.TakeProfit <= 0 || config.TakeProfit > 1 {
		return fmt.Errorf("止盈比例必须在0-1之间，当前值: %.2f", config.TakeProfit)
	}

	// 验证手续费率
	if config.Commission < 0 || config.Commission > 0.01 {
		return fmt.Errorf("手续费率必须在0-1%%之间，当前值: %.4f", config.Commission)
	}

	return nil
}

// ValidateWalkForwardConfig 验证步进分析配置
func (cv *ConfigValidator) ValidateWalkForwardConfig(config *WalkForwardAnalysis) error {
	if config.InSamplePeriod <= 0 {
		return fmt.Errorf("样本内周期必须大于0")
	}

	if config.OutOfSamplePeriod <= 0 {
		return fmt.Errorf("样本外周期必须大于0")
	}

	if config.StepSize <= 0 {
		return fmt.Errorf("步长必须大于0")
	}

	if config.StartDate.After(config.EndDate) {
		return fmt.Errorf("开始日期不能晚于结束日期")
	}

	return nil
}

// ValidateMonteCarloConfig 验证蒙特卡洛配置
func (cv *ConfigValidator) ValidateMonteCarloConfig(config *MonteCarloAnalysis) error {
	if config.Simulations <= 0 || config.Simulations > 100000 {
		return fmt.Errorf("模拟次数必须在1-100000之间")
	}

	if config.ConfidenceLevel <= 0 || config.ConfidenceLevel >= 1 {
		return fmt.Errorf("置信水平必须在0-1之间")
	}

	return nil
}
