package analysis

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"time"

	"gorm.io/gorm"
)

// BacktestEngine 回测引擎
type BacktestEngine struct {
	db             Database
	dataManager    interface{}
	ensembleModels map[string]interface{}
}

// NewBacktestEngine 创建回测引擎
func NewBacktestEngine(db Database, dataManager interface{}, ensembleModels map[string]interface{}) *BacktestEngine {
	return &BacktestEngine{
		db:             db,
		dataManager:    dataManager,
		ensembleModels: ensembleModels,
	}
}

// BacktestConfig 回测配置
type BacktestConfig struct {
	Symbol      string    `json:"symbol"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	InitialCash float64   `json:"initial_cash"`
	Strategy    string    `json:"strategy"` // "buy_and_hold", "ml_prediction", "ensemble"
	Timeframe   string    `json:"timeframe"`
	MaxPosition float64   `json:"max_position"` // 最大仓位比例
	StopLoss    float64   `json:"stop_loss"`    // 止损比例
	TakeProfit  float64   `json:"take_profit"`  // 止盈比例
	Commission  float64   `json:"commission"`   // 基础手续费率

	// 增强现实性参数
	Slippage        float64 `json:"slippage"`         // 滑点比例 (0.001 = 0.1%)
	MarketImpact    float64 `json:"market_impact"`    // 市场冲击系数
	TradingDelay    int     `json:"trading_delay"`    // 交易延迟(分钟)
	Spread          float64 `json:"spread"`           // 买卖价差
	MinOrderSize    float64 `json:"min_order_size"`   // 最小订单大小
	MaxOrderSize    float64 `json:"max_order_size"`   // 最大订单大小
	LiquidityFactor float64 `json:"liquidity_factor"` // 流动性因子
}

// BacktestResult 回测结果
type BacktestResult struct {
	Config       BacktestConfig     `json:"config"`
	Summary      BacktestSummary    `json:"summary"`
	Trades       []TradeRecord      `json:"trades"`
	DailyReturns []DailyReturn      `json:"daily_returns"`
	RiskMetrics  RiskMetrics        `json:"risk_metrics"`
	Performance  PerformanceMetrics `json:"performance"`
}

// BacktestSummary 回测汇总
type BacktestSummary struct {
	TotalReturn    float64 `json:"total_return"`
	AnnualReturn   float64 `json:"annual_return"`
	WinRate        float64 `json:"win_rate"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	TotalTrades    int     `json:"total_trades"`
	ProfitTrades   int     `json:"profit_trades"`
	LossTrades     int     `json:"loss_trades"`
	AvgProfit      float64 `json:"avg_profit"`
	AvgLoss        float64 `json:"avg_loss"`
	ProfitFactor   float64 `json:"profit_factor"`
	RecoveryFactor float64 `json:"recovery_factor"`
}

// TradeRecord 交易记录
type TradeRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	Side       string    `json:"side"` // "buy" or "sell"
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
	Commission float64   `json:"commission"`
	Pnl        float64   `json:"pnl,omitempty"` // 盈亏
	Reason     string    `json:"reason"`        // 交易原因
}

// DailyReturn 每日收益
type DailyReturn struct {
	Date   time.Time `json:"date"`
	Value  float64   `json:"value"`
	Return float64   `json:"return"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	ValueAtRisk95     float64 `json:"value_at_risk_95"`
	ValueAtRisk99     float64 `json:"value_at_risk_99"`
	ExpectedShortfall float64 `json:"expected_shortfall"`
	Beta              float64 `json:"beta"`
	Alpha             float64 `json:"alpha"`
	Volatility        float64 `json:"volatility"`
}

// PerformanceMetrics 绩效指标
type PerformanceMetrics struct {
	CalmarRatio       float64 `json:"calmar_ratio"`
	SortinoRatio      float64 `json:"sortino_ratio"`
	OmegaRatio        float64 `json:"omega_ratio"`
	DownsideDeviation float64 `json:"downside_deviation"`
	InformationRatio  float64 `json:"information_ratio"`
}

// Database 数据库接口（匹配服务器接口）
type Database interface {
	DB() *gorm.DB
}

// DataManager 和 EnsemblePredictor 使用服务器的类型

// RunBacktest 运行回测
func (be *BacktestEngine) RunBacktest(ctx context.Context, config BacktestConfig) (*BacktestResult, error) {
	log.Printf("[BacktestEngine] 开始运行回测: %s, 策略: %s", config.Symbol, config.Strategy)

	// 获取历史数据
	klines, err := be.getHistoricalData(ctx, config.Symbol, config.StartDate, config.EndDate)
	if err != nil {
		return nil, fmt.Errorf("获取历史数据失败: %w", err)
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("没有找到历史数据")
	}

	// 根据策略运行回测
	var result *BacktestResult
	switch config.Strategy {
	case "buy_and_hold":
		result, err = be.runBuyAndHoldStrategy(klines, config)
	case "ml_prediction":
		result, err = be.runMLPredictionStrategy(klines, config)
	case "ensemble":
		result, err = be.runEnsembleStrategy(klines, config)
	default:
		return nil, fmt.Errorf("不支持的策略: %s", config.Strategy)
	}

	if err != nil {
		return nil, fmt.Errorf("策略执行失败: %w", err)
	}

	// 计算绩效指标
	be.calculatePerformanceMetrics(result)

	log.Printf("[BacktestEngine] 回测完成: 总收益率 %.2f%%, 胜率 %.2f%%",
		result.Summary.TotalReturn*100, result.Summary.WinRate*100)

	return result, nil
}

// getHistoricalData 获取历史数据
func (be *BacktestEngine) getHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]KlineDataNumeric, error) {
	// 这里应该从数据管理器获取历史K线数据
	// 暂时返回模拟数据
	return be.generateMockKlines(symbol, start, end), nil
}

// generateMockKlines 生成模拟K线数据
func (be *BacktestEngine) generateMockKlines(symbol string, start, end time.Time) []KlineDataNumeric {
	var klines []KlineDataNumeric
	current := start
	basePrice := 100.0 + rand.Float64()*400 // 随机基础价格

	for current.Before(end) {
		// 生成日K线数据
		open := basePrice * (0.95 + rand.Float64()*0.1)             // ±5%波动
		high := open * (1 + rand.Float64()*0.05)                    // 最高价
		low := open * (1 - rand.Float64()*0.05)                     // 最低价
		close := (open+high+low)/3 + (rand.Float64()-0.5)*open*0.02 // 收盘价
		volume := rand.Float64() * 1000000                          // 成交量

		klines = append(klines, KlineDataNumeric{
			Timestamp: current.Unix() * 1000,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})

		// 更新基础价格
		basePrice = close
		current = current.AddDate(0, 0, 1) // 下一天
	}

	return klines
}

// runBuyAndHoldStrategy 买入持有策略 - 增强现实性版本
func (be *BacktestEngine) runBuyAndHoldStrategy(klines []KlineDataNumeric, config BacktestConfig) (*BacktestResult, error) {
	result := &BacktestResult{
		Config:       config,
		Trades:       []TradeRecord{},
		DailyReturns: []DailyReturn{},
	}

	// 设置现实性参数的默认值
	setRealisticDefaults(&config)

	cash := config.InitialCash
	position := 0.0
	entryPrice := 0.0
	var trades []TradeRecord
	var dailyReturns []DailyReturn

	// 买入第一天 - 使用现实交易执行
	if len(klines) > 0 {
		firstKline := klines[0]
		buyOrder := TradeOrder{
			Side:      "buy",
			Quantity:  (cash * config.MaxPosition) / firstKline.Open,
			OrderType: "market",
			Timestamp: time.Unix(firstKline.Timestamp/1000, 0),
			Reason:    "initial_buy",
		}

		executionResult, err := be.executeRealisticTrade(config, buyOrder, klines, 0)
		if err != nil {
			return nil, fmt.Errorf("初始买入交易失败: %w", err)
		}

		position = executionResult.ExecutedQuantity
		entryPrice = executionResult.ExecutedPrice
		cash -= (position * executionResult.ExecutedPrice) + executionResult.TotalCost

		trades = append(trades, TradeRecord{
			Timestamp:  executionResult.ExecutionTime,
			Side:       "buy",
			Quantity:   position,
			Price:      executionResult.ExecutedPrice,
			Commission: executionResult.Commission,
			Reason:     buyOrder.Reason,
		})
	}

	// 计算每日收益
	lastValue := cash + position*entryPrice
	for i, kline := range klines {
		currentValue := cash + position*kline.Close
		dailyReturn := (currentValue - lastValue) / lastValue

		dailyReturns = append(dailyReturns, DailyReturn{
			Date:   time.Unix(kline.Timestamp/1000, 0),
			Value:  currentValue,
			Return: dailyReturn,
		})

		lastValue = currentValue

		// 最后一天卖出 - 使用现实交易执行
		if i == len(klines)-1 && position > 0 {
			sellOrder := TradeOrder{
				Side:      "sell",
				Quantity:  position,
				OrderType: "market",
				Timestamp: time.Unix(kline.Timestamp/1000, 0),
				Reason:    "final_sell",
			}

			executionResult, err := be.executeRealisticTrade(config, sellOrder, klines, i)
			if err != nil {
				log.Printf("[WARNING] 最终卖出交易失败: %v", err)
				// 即使失败，也要记录简化版本
				trades = append(trades, TradeRecord{
					Timestamp: time.Unix(kline.Timestamp/1000, 0),
					Side:      "sell",
					Quantity:  position,
					Price:     kline.Close,
					Pnl:       (kline.Close - entryPrice) * position,
					Reason:    "final_sell_fallback",
				})
			} else {
				pnl := (executionResult.ExecutedPrice - entryPrice) * executionResult.ExecutedQuantity
				cash += (executionResult.ExecutedQuantity * executionResult.ExecutedPrice) - executionResult.TotalCost

				trades = append(trades, TradeRecord{
					Timestamp:  executionResult.ExecutionTime,
					Side:       "sell",
					Quantity:   executionResult.ExecutedQuantity,
					Price:      executionResult.ExecutedPrice,
					Commission: executionResult.Commission,
					Pnl:        pnl,
					Reason:     sellOrder.Reason,
				})
			}

			position = 0
		}
	}

	result.Trades = trades
	result.DailyReturns = dailyReturns
	be.calculateSummary(result)

	return result, nil
}

// runMLPredictionStrategy 机器学习预测策略
func (be *BacktestEngine) runMLPredictionStrategy(klines []KlineDataNumeric, config BacktestConfig) (*BacktestResult, error) {
	return be.runTechnicalIndicatorStrategy(klines, config, "ml_prediction")
}

// runEnsembleStrategy 集成学习策略
func (be *BacktestEngine) runEnsembleStrategy(klines []KlineDataNumeric, config BacktestConfig) (*BacktestResult, error) {
	return be.runTechnicalIndicatorStrategy(klines, config, "ensemble")
}

// runTechnicalIndicatorStrategy 基于技术指标的交易策略 - 增强现实性版本
func (be *BacktestEngine) runTechnicalIndicatorStrategy(klines []KlineDataNumeric, config BacktestConfig, strategyType string) (*BacktestResult, error) {
	result := &BacktestResult{
		Config:       config,
		Trades:       []TradeRecord{},
		DailyReturns: []DailyReturn{},
	}

	// 设置现实性参数的默认值
	setRealisticDefaults(&config)

	cash := config.InitialCash
	position := 0.0
	positionSize := 0.0
	entryPrice := 0.0
	var trades []TradeRecord
	var dailyReturns []DailyReturn

	// 简化的技术指标交易策略
	// 使用移动平均线交叉作为交易信号

	for i, kline := range klines {
		currentValue := cash + position*kline.Close
		dailyReturn := 0.0
		if i > 0 {
			lastValue := cash + position*klines[i-1].Close
			if lastValue > 0 {
				dailyReturn = (currentValue - lastValue) / lastValue
			}
		}

		dailyReturns = append(dailyReturns, DailyReturn{
			Date:   time.Unix(kline.Timestamp/1000, 0),
			Value:  currentValue,
			Return: dailyReturn,
		})

		// 生成交易信号（简化的技术指标逻辑）
		signal := be.generateTradingSignal(klines, i, strategyType)

		// 使用现实交易执行器执行交易
		if signal == "buy" && position == 0 && cash > 100 {
			// 创建买入订单
			order := TradeOrder{
				Side:      "buy",
				Quantity:  (cash * config.MaxPosition) / kline.Close,
				OrderType: "market",
				Timestamp: time.Unix(kline.Timestamp/1000, 0),
				Reason:    "technical_buy_" + strategyType,
			}

			// 执行现实交易
			executionResult, err := be.executeRealisticTrade(config, order, klines, i)
			if err != nil {
				log.Printf("[WARNING] 买入交易执行失败: %v", err)
				continue
			}

			// 更新账户状态
			positionSize = executionResult.ExecutedQuantity
			position = positionSize
			entryPrice = executionResult.ExecutedPrice
			cash -= (positionSize * executionResult.ExecutedPrice) + executionResult.TotalCost

			// 记录交易
			trades = append(trades, TradeRecord{
				Timestamp:  executionResult.ExecutionTime,
				Side:       "buy",
				Quantity:   positionSize,
				Price:      executionResult.ExecutedPrice,
				Commission: executionResult.Commission,
				Reason:     order.Reason,
			})

		} else if signal == "sell" && position > 0 {
			// 创建卖出订单
			order := TradeOrder{
				Side:      "sell",
				Quantity:  position,
				OrderType: "market",
				Timestamp: time.Unix(kline.Timestamp/1000, 0),
				Reason:    "technical_sell_" + strategyType,
			}

			// 执行现实交易
			executionResult, err := be.executeRealisticTrade(config, order, klines, i)
			if err != nil {
				log.Printf("[WARNING] 卖出交易执行失败: %v", err)
				continue
			}

			// 计算盈亏
			pnl := (executionResult.ExecutedPrice - entryPrice) * executionResult.ExecutedQuantity

			// 更新账户状态
			cash += (executionResult.ExecutedQuantity * executionResult.ExecutedPrice) - executionResult.TotalCost

			// 记录交易
			trades = append(trades, TradeRecord{
				Timestamp:  executionResult.ExecutionTime,
				Side:       "sell",
				Quantity:   executionResult.ExecutedQuantity,
				Price:      executionResult.ExecutedPrice,
				Commission: executionResult.Commission,
				Pnl:        pnl,
				Reason:     order.Reason,
			})

			position = 0
			positionSize = 0
			entryPrice = 0
		}

		// 检查止损止盈 - 使用现实交易执行
		if position > 0 {
			currentPrice := kline.Close

			// 止盈检查
			if config.TakeProfit > 0 && (currentPrice-entryPrice)/entryPrice >= config.TakeProfit {
				order := TradeOrder{
					Side:      "sell",
					Quantity:  position,
					OrderType: "market",
					Timestamp: time.Unix(kline.Timestamp/1000, 0),
					Reason:    "take_profit_" + strategyType,
				}

				executionResult, err := be.executeRealisticTrade(config, order, klines, i)
				if err == nil {
					pnl := (executionResult.ExecutedPrice - entryPrice) * executionResult.ExecutedQuantity
					cash += (executionResult.ExecutedQuantity * executionResult.ExecutedPrice) - executionResult.TotalCost

					trades = append(trades, TradeRecord{
						Timestamp:  executionResult.ExecutionTime,
						Side:       "sell",
						Quantity:   executionResult.ExecutedQuantity,
						Price:      executionResult.ExecutedPrice,
						Commission: executionResult.Commission,
						Pnl:        pnl,
						Reason:     order.Reason,
					})

					position = 0
					positionSize = 0
					entryPrice = 0
				}
			}

			// 止损检查
			if config.StopLoss > 0 && (entryPrice-currentPrice)/entryPrice >= config.StopLoss {
				order := TradeOrder{
					Side:      "sell",
					Quantity:  position,
					OrderType: "market",
					Timestamp: time.Unix(kline.Timestamp/1000, 0),
					Reason:    "stop_loss_" + strategyType,
				}

				executionResult, err := be.executeRealisticTrade(config, order, klines, i)
				if err == nil {
					pnl := (executionResult.ExecutedPrice - entryPrice) * executionResult.ExecutedQuantity
					cash += (executionResult.ExecutedQuantity * executionResult.ExecutedPrice) - executionResult.TotalCost

					trades = append(trades, TradeRecord{
						Timestamp:  executionResult.ExecutionTime,
						Side:       "sell",
						Quantity:   executionResult.ExecutedQuantity,
						Price:      executionResult.ExecutedPrice,
						Commission: executionResult.Commission,
						Pnl:        pnl,
						Reason:     "stop_loss_" + strategyType,
					})

					position = 0
					positionSize = 0
					entryPrice = 0
				}
			}
		}
	}

	// 最后一天平仓（如果还有持仓）
	if position > 0 && len(klines) > 0 {
		lastIndex := len(klines) - 1
		order := TradeOrder{
			Side:      "sell",
			Quantity:  position,
			OrderType: "market",
			Timestamp: time.Unix(klines[lastIndex].Timestamp/1000, 0),
			Reason:    "final_close_" + strategyType,
		}

		executionResult, err := be.executeRealisticTrade(config, order, klines, lastIndex)
		if err == nil {
			pnl := (executionResult.ExecutedPrice - entryPrice) * executionResult.ExecutedQuantity
			cash += (executionResult.ExecutedQuantity * executionResult.ExecutedPrice) - executionResult.TotalCost

			trades = append(trades, TradeRecord{
				Timestamp:  executionResult.ExecutionTime,
				Side:       "sell",
				Quantity:   executionResult.ExecutedQuantity,
				Price:      executionResult.ExecutedPrice,
				Commission: executionResult.Commission,
				Pnl:        pnl,
				Reason:     order.Reason,
			})
		}
	}

	result.Trades = trades
	result.DailyReturns = dailyReturns
	be.calculateSummary(result)

	return result, nil
}

// generateTradingSignal 生成交易信号（适配不同策略类型的指标逻辑）
func (be *BacktestEngine) generateTradingSignal(klines []KlineDataNumeric, currentIndex int, strategyType string) string {
	switch strategyType {
	case "ml_prediction":
		// 机器学习预测策略：使用趋势跟踪（适合LONG/SHORT市场）
		return be.generateTrendSignal(klines, currentIndex)
	case "ensemble":
		// 集成学习策略：结合多种指标（适合各种市场）
		return be.generateEnsembleSignal(klines, currentIndex)
	default:
		// 默认使用趋势信号
		return be.generateTrendSignal(klines, currentIndex)
	}
}

// generateTrendSignal 趋势跟踪信号（适合LONG/SHORT市场）
func (be *BacktestEngine) generateTrendSignal(klines []KlineDataNumeric, currentIndex int) string {
	if currentIndex < 25 { // 需要足够的历史数据
		return "hold"
	}

	// 移动平均线交叉策略（适合趋势市场）
	shortMA := be.calculateSMAForSignal(klines, currentIndex, 5)
	longMA := be.calculateSMAForSignal(klines, currentIndex, 20)

	if shortMA > longMA {
		// 检查是否为金叉
		prevShortMA := be.calculateSMAForSignal(klines, currentIndex-1, 5)
		prevLongMA := be.calculateSMAForSignal(klines, currentIndex-1, 20)

		if prevShortMA <= prevLongMA { // 金叉
			return "buy"
		}
	} else if shortMA < longMA {
		// 检查是否为死叉
		prevShortMA := be.calculateSMAForSignal(klines, currentIndex-1, 5)
		prevLongMA := be.calculateSMAForSignal(klines, currentIndex-1, 20)

		if prevShortMA >= prevLongMA { // 死叉
			return "sell"
		}
	}

	return "hold"
}

// generateEnsembleSignal 集成信号（适合RANGE市场）
func (be *BacktestEngine) generateEnsembleSignal(klines []KlineDataNumeric, currentIndex int) string {
	if currentIndex < 15 { // 较少的历史数据要求
		return "hold"
	}

	current := klines[currentIndex]

	// 简化的震荡策略：基于价格波动和均线关系
	shortMA := be.calculateSMAForSignal(klines, currentIndex, 5)
	longMA := be.calculateSMAForSignal(klines, currentIndex, 10)

	// 计算价格波动率（最近5天的标准差）
	volatility := be.calculateVolatilityForPeriod(klines, currentIndex, 5)

	// 震荡策略逻辑：
	// 1. 价格在均线附近波动
	// 2. 波动率适中
	// 3. 基于简单的反转逻辑

	priceDiff := (current.Close - shortMA) / shortMA
	maDiff := (shortMA - longMA) / longMA

	// 在震荡区间内寻找交易机会
	if math.Abs(priceDiff) < 0.02 && math.Abs(maDiff) < 0.01 && volatility > 0.005 {
		// 检查前一天的状态
		if currentIndex > 0 {
			prevPrice := klines[currentIndex-1].Close
			prevShortMA := be.calculateSMAForSignal(klines, currentIndex-1, 5)

			// 简单的反转逻辑
			if prevPrice < prevShortMA && current.Close > shortMA {
				return "buy" // 从下方突破均线
			} else if prevPrice > prevShortMA && current.Close < shortMA {
				return "sell" // 从上方跌破均线
			}
		}
	}

	return "hold"
}

// calculateVolatilityForPeriod 计算指定期间的波动率
func (be *BacktestEngine) calculateVolatilityForPeriod(klines []KlineDataNumeric, endIndex, period int) float64 {
	if endIndex < period-1 {
		return 0
	}

	start := endIndex - period + 1
	returns := make([]float64, period-1)

	for i := 0; i < period-1; i++ {
		if i+1 < period {
			curr := klines[start+i].Close
			next := klines[start+i+1].Close
			if curr > 0 {
				returns[i] = (next - curr) / curr
			}
		}
	}

	// 计算标准差
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// calculateSMAForSignal 计算简单移动平均线（用于信号生成）
func (be *BacktestEngine) calculateSMAForSignal(klines []KlineDataNumeric, endIndex, period int) float64 {
	if endIndex < period-1 {
		return 0
	}

	sum := 0.0
	start := endIndex - period + 1
	for i := start; i <= endIndex; i++ {
		sum += klines[i].Close
	}

	return sum / float64(period)
}

// calculateSummary 计算汇总统计
func (be *BacktestEngine) calculateSummary(result *BacktestResult) {
	if len(result.DailyReturns) == 0 {
		return
	}

	// 总收益率
	firstValue := result.DailyReturns[0].Value
	lastValue := result.DailyReturns[len(result.DailyReturns)-1].Value
	result.Summary.TotalReturn = (lastValue - firstValue) / firstValue

	// 年化收益率
	days := len(result.DailyReturns)
	if days > 0 {
		years := float64(days) / 365
		if years > 0 {
			result.Summary.AnnualReturn = math.Pow(1+result.Summary.TotalReturn, 1/years) - 1
		}
	}

	// 胜率
	winningDays := 0
	for _, dr := range result.DailyReturns {
		if dr.Return > 0 {
			winningDays++
		}
	}
	result.Summary.WinRate = float64(winningDays) / float64(len(result.DailyReturns))

	// 最大回撤
	maxValue := firstValue
	maxDrawdown := 0.0
	for _, dr := range result.DailyReturns {
		if dr.Value > maxValue {
			maxValue = dr.Value
		}
		drawdown := (maxValue - dr.Value) / maxValue
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	result.Summary.MaxDrawdown = maxDrawdown

	// 夏普比率（简化计算）
	if len(result.DailyReturns) > 0 {
		totalReturn := result.Summary.TotalReturn
		volatility := be.calculateVolatility(result.DailyReturns)
		if volatility > 0 {
			result.Summary.SharpeRatio = totalReturn / volatility
		}
	}

	// 交易统计
	result.Summary.TotalTrades = len(result.Trades)
	profitTrades := 0
	totalProfit := 0.0
	totalLoss := 0.0

	for _, trade := range result.Trades {
		if trade.Pnl != 0 {
			if trade.Pnl > 0 {
				profitTrades++
				totalProfit += trade.Pnl
			} else {
				totalLoss += math.Abs(trade.Pnl)
			}
		}
	}

	result.Summary.ProfitTrades = profitTrades
	result.Summary.LossTrades = result.Summary.TotalTrades - profitTrades

	if profitTrades > 0 {
		result.Summary.AvgProfit = totalProfit / float64(profitTrades)
	}
	if result.Summary.LossTrades > 0 {
		result.Summary.AvgLoss = totalLoss / float64(result.Summary.LossTrades)
	}
	if totalLoss > 0 {
		result.Summary.ProfitFactor = totalProfit / totalLoss
	}
}

// calculateVolatility 计算波动率
func (be *BacktestEngine) calculateVolatility(dailyReturns []DailyReturn) float64 {
	if len(dailyReturns) <= 1 {
		return 0
	}

	// 计算平均收益率
	sum := 0.0
	for _, dr := range dailyReturns {
		sum += dr.Return
	}
	mean := sum / float64(len(dailyReturns))

	// 计算方差
	variance := 0.0
	for _, dr := range dailyReturns {
		variance += math.Pow(dr.Return-mean, 2)
	}
	variance /= float64(len(dailyReturns) - 1)

	return math.Sqrt(variance)
}

// calculatePerformanceMetrics 计算绩效指标
func (be *BacktestEngine) calculatePerformanceMetrics(result *BacktestResult) {
	// Calmar比率 = 年化收益率 / 最大回撤
	if result.Summary.MaxDrawdown > 0 {
		result.Performance.CalmarRatio = result.Summary.AnnualReturn / result.Summary.MaxDrawdown
	}

	// Sortino比率（简化为夏普比率）
	result.Performance.SortinoRatio = result.Summary.SharpeRatio

	// Omega比率（简化为1 + 夏普比率）
	result.Performance.OmegaRatio = 1 + result.Summary.SharpeRatio

	// 风险指标
	be.calculateRiskMetrics(result)
}

// calculateRiskMetrics 计算风险指标
func (be *BacktestEngine) calculateRiskMetrics(result *BacktestResult) {
	if len(result.DailyReturns) == 0 {
		return
	}

	returns := make([]float64, len(result.DailyReturns))
	for i, dr := range result.DailyReturns {
		returns[i] = dr.Return
	}

	// 简化的VaR计算（95%置信度）
	sort.Float64s(returns)
	var95Index := int(float64(len(returns)) * 0.05)
	if var95Index < len(returns) {
		result.RiskMetrics.ValueAtRisk95 = -returns[var95Index]
	}

	// 简化的VaR计算（99%置信度）
	var99Index := int(float64(len(returns)) * 0.01)
	if var99Index < len(returns) {
		result.RiskMetrics.ValueAtRisk99 = -returns[var99Index]
	}

	// 期望亏空（简化为VaR95）
	result.RiskMetrics.ExpectedShortfall = result.RiskMetrics.ValueAtRisk95

	// 波动率
	result.RiskMetrics.Volatility = be.calculateVolatility(result.DailyReturns)
}

// CompareStrategies 策略对比
func (be *BacktestEngine) CompareStrategies(ctx context.Context, configs []BacktestConfig) (*StrategyComparison, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("至少需要一个策略配置")
	}

	comparison := &StrategyComparison{
		Strategies: []StrategyResult{},
	}

	bestSharpe := -1000.0
	bestReturn := -1000.0

	for _, config := range configs {
		result, err := be.RunBacktest(ctx, config)
		if err != nil {
			log.Printf("[BacktestEngine] 策略 %s 执行失败: %v", config.Strategy, err)
			continue
		}

		strategyResult := StrategyResult{
			Name:    config.Strategy,
			Result:  *result,
			Ranking: 0, // 稍后计算
		}

		comparison.Strategies = append(comparison.Strategies, strategyResult)

		// 记录最佳策略
		if result.Summary.SharpeRatio > bestSharpe {
			bestSharpe = result.Summary.SharpeRatio
			comparison.BestStrategyBySharpe = config.Strategy
		}

		if result.Summary.TotalReturn > bestReturn {
			bestReturn = result.Summary.TotalReturn
			comparison.BestStrategyByReturn = config.Strategy
		}
	}

	// 计算排名
	be.rankStrategies(comparison)

	return comparison, nil
}

// StrategyComparison 策略对比结果
type StrategyComparison struct {
	Strategies           []StrategyResult `json:"strategies"`
	BestStrategyByReturn string           `json:"best_strategy_by_return"`
	BestStrategyBySharpe string           `json:"best_strategy_by_sharpe"`
}

// StrategyResult 策略结果
type StrategyResult struct {
	Name    string         `json:"name"`
	Result  BacktestResult `json:"result"`
	Ranking int            `json:"ranking"`
}

// rankStrategies 对策略进行排名
func (be *BacktestEngine) rankStrategies(comparison *StrategyComparison) {
	// 按夏普比率降序排序
	sort.Slice(comparison.Strategies, func(i, j int) bool {
		return comparison.Strategies[i].Result.Summary.SharpeRatio >
			comparison.Strategies[j].Result.Summary.SharpeRatio
	})

	// 分配排名
	for i := range comparison.Strategies {
		comparison.Strategies[i].Ranking = i + 1
	}
}

// RunBatchBacktest 批量运行回测
func (be *BacktestEngine) RunBatchBacktest(ctx context.Context, configs []BacktestConfig) ([]*BacktestResult, error) {
	results := make([]*BacktestResult, 0, len(configs))

	for _, config := range configs {
		result, err := be.RunBacktest(ctx, config)
		if err != nil {
			log.Printf("[BacktestEngine] 批量回测失败 %s: %v", config.Strategy, err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// OptimizeStrategy 策略优化
func (be *BacktestEngine) OptimizeStrategy(ctx context.Context, baseConfig BacktestConfig, paramRanges map[string][]float64) (*OptimizationResult, error) {
	result := &OptimizationResult{
		BaseConfig:   baseConfig,
		ParamRanges:  paramRanges,
		Combinations: []ParameterCombination{},
	}

	// 生成参数组合（简化实现，只优化一个参数）
	for param, values := range paramRanges {
		for _, value := range values {
			// 复制基础配置
			config := baseConfig

			// 设置参数值
			switch param {
			case "max_position":
				config.MaxPosition = value
			case "stop_loss":
				config.StopLoss = value
			case "take_profit":
				config.TakeProfit = value
			case "commission":
				config.Commission = value
			}

			// 运行回测
			backtestResult, err := be.RunBacktest(ctx, config)
			if err != nil {
				log.Printf("[BacktestEngine] 优化参数失败: %v", err)
				continue
			}

			combination := ParameterCombination{
				Parameters: map[string]float64{param: value},
				Result:     *backtestResult,
			}

			result.Combinations = append(result.Combinations, combination)

			// 记录最优结果
			if backtestResult.Summary.SharpeRatio > result.BestResult.Summary.SharpeRatio {
				result.BestResult = *backtestResult
				result.BestParameters = combination.Parameters
			}
		}
	}

	return result, nil
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	BaseConfig     BacktestConfig         `json:"base_config"`
	ParamRanges    map[string][]float64   `json:"param_ranges"`
	Combinations   []ParameterCombination `json:"combinations"`
	BestResult     BacktestResult         `json:"best_result"`
	BestParameters map[string]float64     `json:"best_parameters"`
}

// ParameterCombination 参数组合
type ParameterCombination struct {
	Parameters map[string]float64 `json:"parameters"`
	Result     BacktestResult     `json:"result"`
}

// ============================================================================
// 增强现实性交易执行引擎
// ============================================================================

// RealisticTradeExecutor 现实交易执行器
type RealisticTradeExecutor struct {
	config *BacktestConfig
}

// NewRealisticTradeExecutor 创建现实交易执行器
func NewRealisticTradeExecutor(config *BacktestConfig) *RealisticTradeExecutor {
	return &RealisticTradeExecutor{
		config: config,
	}
}

// TradeOrder 交易订单
type TradeOrder struct {
	Side       string    // "buy" or "sell"
	Quantity   float64   // 订单数量
	LimitPrice float64   // 限价 (可选)
	OrderType  string    // "market", "limit"
	Timestamp  time.Time // 订单时间
	Reason     string    // 交易原因
}

// TradeExecutionResult 交易执行结果
type TradeExecutionResult struct {
	Order            TradeOrder
	ExecutedPrice    float64   // 实际执行价格
	ExecutedQuantity float64   // 实际执行数量
	Commission       float64   // 手续费
	Slippage         float64   // 滑点成本
	MarketImpact     float64   // 市场冲击成本
	Spread           float64   // 买卖价差成本
	TotalCost        float64   // 总成本
	ExecutionTime    time.Time // 执行时间
	Success          bool      // 是否成功执行
	Error            string    // 错误信息
}

// ExecuteOrder 执行交易订单
func (rte *RealisticTradeExecutor) ExecuteOrder(order TradeOrder, marketData []KlineDataNumeric, currentIndex int) *TradeExecutionResult {
	result := &TradeExecutionResult{
		Order:         order,
		ExecutionTime: order.Timestamp,
		Success:       true,
	}

	// 1. 检查订单有效性
	if !rte.validateOrder(order) {
		result.Success = false
		result.Error = "订单无效"
		return result
	}

	// 2. 应用交易延迟
	executionIndex := rte.applyTradingDelay(currentIndex, order.Timestamp, marketData)
	if executionIndex >= len(marketData) {
		result.Success = false
		result.Error = "超出数据范围"
		return result
	}

	// 3. 获取市场价格
	marketPrice := rte.getMarketPrice(order, marketData[executionIndex])

	// 4. 计算滑点
	slippage := rte.calculateSlippage(order, marketPrice, marketData[executionIndex])
	result.Slippage = slippage

	// 5. 计算实际执行价格
	executionPrice := rte.calculateExecutionPrice(order, marketPrice, slippage)
	result.ExecutedPrice = executionPrice

	// 6. 计算市场冲击
	marketImpact := rte.calculateMarketImpact(order, marketData, executionIndex)
	result.MarketImpact = marketImpact

	// 7. 计算买卖价差成本
	spreadCost := rte.calculateSpreadCost(order, marketData[executionIndex])
	result.Spread = spreadCost

	// 8. 计算手续费
	commission := rte.calculateCommission(order, executionPrice)
	result.Commission = commission

	// 9. 计算总成本
	result.TotalCost = commission + slippage + marketImpact + spreadCost

	// 10. 确定实际执行数量
	result.ExecutedQuantity = rte.calculateExecutedQuantity(order, result.TotalCost, executionPrice)

	// 11. 更新执行时间
	result.ExecutionTime = time.Unix(marketData[executionIndex].Timestamp/1000, 0)

	return result
}

// validateOrder 验证订单
func (rte *RealisticTradeExecutor) validateOrder(order TradeOrder) bool {
	// 检查订单大小
	if order.Quantity <= 0 {
		return false
	}

	// 检查最小订单大小
	if rte.config.MinOrderSize > 0 && order.Quantity < rte.config.MinOrderSize {
		return false
	}

	// 检查最大订单大小
	if rte.config.MaxOrderSize > 0 && order.Quantity > rte.config.MaxOrderSize {
		return false
	}

	// 检查订单类型
	if order.OrderType != "market" && order.OrderType != "limit" {
		return false
	}

	// 检查买卖方向
	if order.Side != "buy" && order.Side != "sell" {
		return false
	}

	return true
}

// applyTradingDelay 应用交易延迟
func (rte *RealisticTradeExecutor) applyTradingDelay(currentIndex int, orderTime time.Time, marketData []KlineDataNumeric) int {
	if rte.config.TradingDelay <= 0 {
		return currentIndex
	}

	// 转换为分钟延迟
	delayMinutes := rte.config.TradingDelay

	// 找到延迟后的执行时间点
	targetTime := orderTime.Add(time.Duration(delayMinutes) * time.Minute)
	executionIndex := currentIndex

	// 在市场数据中找到最接近的执行时间点
	for i := currentIndex; i < len(marketData); i++ {
		klineTime := time.Unix(marketData[i].Timestamp/1000, 0)
		if klineTime.After(targetTime) || klineTime.Equal(targetTime) {
			executionIndex = i
			break
		}
	}

	return executionIndex
}

// getMarketPrice 获取市场价格
func (rte *RealisticTradeExecutor) getMarketPrice(order TradeOrder, kline KlineDataNumeric) float64 {
	switch order.OrderType {
	case "market":
		// 市价单使用当前价格
		if order.Side == "buy" {
			// 买入使用卖价（ask），这里用收盘价近似
			return kline.Close
		} else {
			// 卖出使用买价（bid），这里用收盘价近似
			return kline.Close
		}
	case "limit":
		// 限价单使用指定的限价
		return order.LimitPrice
	default:
		return kline.Close
	}
}

// calculateSlippage 计算滑点
func (rte *RealisticTradeExecutor) calculateSlippage(order TradeOrder, marketPrice float64, kline KlineDataNumeric) float64 {
	if rte.config.Slippage <= 0 {
		return 0
	}

	// 基础滑点
	baseSlippage := marketPrice * rte.config.Slippage

	// 波动率调整滑点
	volatility := rte.calculateVolatilityForKline(kline)
	volatilityMultiplier := 1.0 + volatility*2.0 // 波动率越高，滑点越大

	// 成交量调整滑点
	volumeMultiplier := 1.0
	if kline.Volume > 0 {
		avgVolume := 1000000.0                                // 假设平均成交量
		volumeMultiplier = 1.0 + (avgVolume/kline.Volume)*0.5 // 成交量越低，滑点越大
	}

	// 订单大小调整滑点
	sizeMultiplier := 1.0 + (order.Quantity/1000.0)*0.1 // 大订单滑点更大

	slippage := baseSlippage * volatilityMultiplier * volumeMultiplier * sizeMultiplier

	return slippage
}

// calculateVolatilityForKline 计算单根K线的波动率
func (rte *RealisticTradeExecutor) calculateVolatilityForKline(kline KlineDataNumeric) float64 {
	if kline.High == kline.Low {
		return 0
	}

	// 使用真实波动幅度 (True Range) 的简化版
	highLow := (kline.High - kline.Low) / kline.Close

	return math.Min(highLow, 0.1) // 限制最大波动率为10%
}

// calculateMarketImpact 计算市场冲击
func (rte *RealisticTradeExecutor) calculateMarketImpact(order TradeOrder, marketData []KlineDataNumeric, executionIndex int) float64 {
	if rte.config.MarketImpact <= 0 {
		return 0
	}

	// 市场冲击 = 冲击系数 * 订单大小 * 价格 * 流动性因子
	impact := rte.config.MarketImpact * order.Quantity * marketData[executionIndex].Close * rte.config.LiquidityFactor

	// 限制最大冲击
	maxImpact := marketData[executionIndex].Close * 0.01 // 最大1%的冲击
	if impact > maxImpact {
		impact = maxImpact
	}

	return impact
}

// calculateSpreadCost 计算买卖价差成本
func (rte *RealisticTradeExecutor) calculateSpreadCost(order TradeOrder, kline KlineDataNumeric) float64 {
	if rte.config.Spread <= 0 {
		return 0
	}

	// 价差成本 = 价差比例 * 交易金额
	spreadCost := rte.config.Spread * order.Quantity * kline.Close

	return spreadCost
}

// calculateCommission 计算手续费
func (rte *RealisticTradeExecutor) calculateCommission(order TradeOrder, executionPrice float64) float64 {
	// 基础佣金
	baseCommission := rte.config.Commission * order.Quantity * executionPrice

	// 可能添加固定费用等
	return baseCommission
}

// calculateExecutionPrice 计算实际执行价格
func (rte *RealisticTradeExecutor) calculateExecutionPrice(order TradeOrder, marketPrice, slippage float64) float64 {
	if order.Side == "buy" {
		// 买入时价格会上涨（不利滑点）
		return marketPrice + slippage
	} else {
		// 卖出时价格会下跌（不利滑点）
		return marketPrice - slippage
	}
}

// calculateExecutedQuantity 计算实际执行数量
func (rte *RealisticTradeExecutor) calculateExecutedQuantity(order TradeOrder, totalCost, executionPrice float64) float64 {
	// 在考虑所有成本后，计算实际可执行的数量
	// 这里简化处理，实际可能因成本过高而减少执行数量

	// 计算理论执行金额
	theoreticalValue := order.Quantity * executionPrice

	// 如果成本过高，可能无法完全执行
	if totalCost > theoreticalValue*0.1 { // 成本超过10%
		// 减少执行数量以控制成本
		maxCostRatio := 0.05 // 最大5%成本
		adjustedQuantity := order.Quantity * (1.0 - maxCostRatio)
		return math.Max(adjustedQuantity, order.Quantity*0.5) // 至少执行50%
	}

	return order.Quantity
}

// ============================================================================
// 增强的回测执行方法
// ============================================================================

// executeRealisticTrade 执行现实交易（替换原有的简化交易逻辑）
func (be *BacktestEngine) executeRealisticTrade(
	config BacktestConfig,
	order TradeOrder,
	marketData []KlineDataNumeric,
	currentIndex int,
) (*TradeExecutionResult, error) {

	executor := NewRealisticTradeExecutor(&config)
	result := executor.ExecuteOrder(order, marketData, currentIndex)

	if !result.Success {
		return nil, fmt.Errorf("交易执行失败: %s", result.Error)
	}

	return result, nil
}

// setRealisticDefaults 为现实性参数设置默认值
func setRealisticDefaults(config *BacktestConfig) {
	// 设置默认的现实性参数
	if config.Slippage == 0 {
		config.Slippage = 0.001 // 0.1% 滑点
	}
	if config.MarketImpact == 0 {
		config.MarketImpact = 0.0001 // 0.01% 市场冲击系数
	}
	if config.TradingDelay == 0 {
		config.TradingDelay = 5 // 5分钟延迟
	}
	if config.Spread == 0 {
		config.Spread = 0.0005 // 0.05% 买卖价差
	}
	if config.MinOrderSize == 0 {
		config.MinOrderSize = 10 // 最小10个单位
	}
	if config.MaxOrderSize == 0 {
		config.MaxOrderSize = 10000 // 最大10000个单位
	}
	if config.LiquidityFactor == 0 {
		config.LiquidityFactor = 1.0 // 正常流动性
	}
}
