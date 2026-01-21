package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// runBuyAndHoldStrategy 执行买入持有策略
func (be *BacktestEngine) runBuyAndHoldStrategy(result *BacktestResult, data []MarketData) error {
	if len(data) < 2 {
		return fmt.Errorf("数据点不足")
	}

	position := 0.0
	var entryPrice float64

	for i, marketData := range data {
		currentPrice := marketData.Price

		// 买入持有策略：第一次买入，一直持有
		if i == 0 {
			position = result.Config.InitialCash / currentPrice
			entryPrice = currentPrice

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:     result.Config.Symbol,
				Side:       "buy",
				Quantity:   position,
				Price:      currentPrice,
				Timestamp:  marketData.LastUpdated,
				Commission: 0,
				PnL:        0,
				Reason:     "买入持有策略初始买入",
			})

			log.Printf("[TRADE] 买入持有策略买入: %.4f @ %.2f", position, currentPrice)
		}

		// 最后一天卖出
		if i == len(data)-1 && position > 0 {
			exitPrice := currentPrice
			pnl := (exitPrice - entryPrice) * position

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:     result.Config.Symbol,
				Side:       "sell",
				Quantity:   position,
				Price:      exitPrice,
				Timestamp:  marketData.LastUpdated,
				Commission: 0,
				PnL:        pnl,
				Reason:     "买入持有策略最终卖出",
			})

			log.Printf("[TRADE] 买入持有策略卖出: %.4f @ %.2f, PnL: %.2f", position, exitPrice, pnl)
		}
	}

	return nil
}

// runMLPredictionStrategy 执行机器学习预测策略
func (be *BacktestEngine) runMLPredictionStrategy(ctx context.Context, result *BacktestResult, data []MarketData) error {
	if len(data) < 50 {
		return fmt.Errorf("数据点不足，无法进行机器学习预测")
	}

	position := 0.0
	cash := result.Config.InitialCash

	for i := 50; i < len(data); i++ {
		marketData := data[i]
		currentPrice := marketData.Price

		// 使用AI预测价格方向
		prediction, confidence := be.predictPriceWithAI(ctx, data[:i], result.Config.Symbol)

		// 基于预测结果做决策
		if prediction > 0 && confidence > 0.6 && position == 0 {
			// 预测上涨且信心足够，买入
			maxPosition := cash / currentPrice
			position = maxPosition * 0.5 // 只用50%的现金买入

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       result.Config.Symbol,
				Side:         "buy",
				Quantity:     position,
				Price:        currentPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          0,
				AIConfidence: confidence,
				Reason:       "ML预测上涨买入",
			})

			cash -= position * currentPrice
			log.Printf("[TRADE] ML预测买入: %.4f @ %.2f, 信心: %.2f", position, currentPrice, confidence)

		} else if prediction < 0 && confidence > 0.6 && position > 0 {
			// 预测下跌且信心足够，卖出
			exitPrice := currentPrice
			pnl := (exitPrice - result.Trades[len(result.Trades)-1].Price) * position

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       result.Config.Symbol,
				Side:         "sell",
				Quantity:     position,
				Price:        exitPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          pnl,
				AIConfidence: confidence,
				Reason:       "ML预测下跌卖出",
			})

			cash += position * exitPrice
			position = 0
			log.Printf("[TRADE] ML预测卖出: %.4f @ %.2f, PnL: %.2f", position, exitPrice, pnl)
		}
	}

	return nil
}

// runEnsembleStrategy 执行集成策略
func (be *BacktestEngine) runEnsembleStrategy(ctx context.Context, result *BacktestResult, data []MarketData) error {
	if len(data) < 50 {
		return fmt.Errorf("数据点不足，无法进行集成策略")
	}

	position := 0.0
	cash := result.Config.InitialCash

	for i := 50; i < len(data); i++ {
		marketData := data[i]
		currentPrice := marketData.Price

		// 使用AI预测
		prediction, confidence := be.predictPriceWithAI(ctx, data[:i], result.Config.Symbol)

		// ===== 阶段四优化：增强信号一致性检查 =====
		// 计算多周期趋势
		trend5 := be.analyzeTrendStrength(data[:i], i, 5)
		trend10 := be.analyzeTrendStrength(data[:i], i, 10)
		trend20 := be.analyzeTrendStrength(data[:i], i, 20)
		rsi := be.calculateRSI(data[i-20:i], 14)
		volatility := be.calculateVolatility(data[i-20:i], 20)

		// 多周期趋势一致性检查
		trendConsistency := be.calculateTrendConsistency(trend5, trend10, trend20)
		weightedTrendScore := (trend5.Strength*0.5 + trend10.Strength*0.3 + trend20.Strength*0.2) * trendConsistency

		// 增强信号一致性
		signalConsistency := be.calculateSignalConsistency(prediction, weightedTrendScore, (rsi-50)/50, volatility)

		// 综合评分 - 考虑一致性
		ensembleScore := prediction*0.4 + weightedTrendScore*0.35 + (rsi-50)/50*0.15 + signalConsistency*0.1

		log.Printf("[SIGNAL_ENHANCEMENT_V4] 信号增强: 趋势一致性%.2f, 信号一致性%.2f, 综合得分%.3f",
			trendConsistency, signalConsistency, ensembleScore)

		// 基于波动率调整决策阈值
		buyThreshold := 0.3
		sellThreshold := -0.3

		if volatility > 0.03 {
			buyThreshold = 0.4 // 高波动时要求更强的信号
			sellThreshold = -0.4
		}

		if ensembleScore > buyThreshold && confidence > 0.5 && position == 0 {
			// 买入
			maxPosition := cash / currentPrice
			position = maxPosition * 0.6 // 用60%的现金

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       result.Config.Symbol,
				Side:         "buy",
				Quantity:     position,
				Price:        currentPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          0,
				AIConfidence: confidence,
				Reason:       "集成策略买入",
			})

			cash -= position * currentPrice
			log.Printf("[TRADE] 集成策略买入: %.4f @ %.2f", position, currentPrice)

		} else if ensembleScore < sellThreshold && position > 0 {
			// 卖出
			exitPrice := currentPrice
			pnl := (exitPrice - result.Trades[len(result.Trades)-1].Price) * position

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       result.Config.Symbol,
				Side:         "sell",
				Quantity:     position,
				Price:        exitPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          pnl,
				AIConfidence: confidence,
				Reason:       "集成策略卖出",
			})

			cash += position * exitPrice
			position = 0
			log.Printf("[TRADE] 集成策略卖出: %.4f @ %.2f, PnL: %.2f", position, exitPrice, pnl)
		}
	}

	return nil
}

// predictPriceDirection 预测价格方向
func (be *BacktestEngine) predictPriceDirection(ctx context.Context, data []MarketData, symbol string) float64 {
	if len(data) < 20 {
		return 0
	}

	// 简单的趋势预测
	recentTrend := be.analyzeTrendStrength(data, len(data)-1, 20)
	rsi := be.calculateRSI(data[len(data)-20:], 14)

	// 综合判断
	trendStrength := recentTrend.Strength
	score := trendStrength*0.6 + (rsi-50)/50*0.4

	return score
}

// predictPriceWithAI 使用AI预测价格
func (be *BacktestEngine) predictPriceWithAI(ctx context.Context, data []MarketData, symbol string) (prediction float64, confidence float64) {
	if len(data) < 50 {
		return 0, 0
	}

	// 构建特征
	features := be.extractFeatures(data)

	if len(features) == 0 {
		return 0, 0
	}

	// 使用集成模型预测
	ensemblePrediction := 0.0
	validModels := 0

	for range be.ensembleModels {
		// 暂时简化实现，实际应该根据模型接口进行预测
		// 临时返回默认预测值
		ensemblePrediction += 0.0
		validModels++
	}

	if validModels == 0 {
		// 如果没有有效的AI模型，使用技术指标预测
		return be.predictPriceDirection(ctx, data, symbol), 0.5
	}

	ensemblePrediction /= float64(validModels)

	// 计算置信度（基于模型一致性，这里简化处理）
	confidence = 0.7

	return ensemblePrediction, confidence
}

// combineAIPredictionWithStrategy 结合AI预测和策略
func (be *BacktestEngine) combineAIPredictionWithStrategy(prediction, confidence float64, strategyType string) string {
	// 基于预测结果和置信度决定行动
	if confidence > 0.7 {
		if prediction > 0.2 {
			return "buy"
		} else if prediction < -0.2 {
			return "sell"
		}
	} else if confidence > 0.5 {
		if prediction > 0.3 {
			return "buy"
		} else if prediction < -0.3 {
			return "sell"
		}
	}

	return "hold"
}

// extractFeatures 提取特征
func (be *BacktestEngine) extractFeatures(data []MarketData) [][]float64 {
	if len(data) < 20 {
		return nil
	}

	// 提取技术指标作为特征
	features := make([]float64, 10)

	// 趋势特征
	trend20 := be.analyzeTrendStrength(data, len(data)-1, 20)
	trend50 := be.analyzeTrendStrength(data, len(data)-1, 50)
	features[0] = trend20.Strength
	features[1] = trend50.Strength

	// 波动率特征
	features[2] = be.calculateVolatility(data[len(data)-20:], 20)

	// 动量特征
	features[3] = be.calculateRSI(data[len(data)-20:], 14)
	features[4] = be.calculateMACDSignal(data[len(data)-30:])

	// 成交量特征
	features[5] = be.calculateVolumeTrend(data[len(data)-20:], 20)

	// 价格位置特征
	features[6] = be.calculateBollingerPosition(data[len(data)-20:])

	// 支撑阻力特征
	features[7] = be.calculateSupportLevel(data[len(data)-20:])
	features[8] = be.calculateResistanceLevel(data[len(data)-20:])

	// 随机指标
	features[9] = be.calculateStochasticK(data[len(data)-20:])

	return [][]float64{features}
}

// runDeepLearningStrategy 执行深度学习策略
func (be *BacktestEngine) runDeepLearningStrategy(ctx context.Context, result *BacktestResult, data []MarketData) error {
	log.Printf("[DEEP_LEARNING] 开始执行深度学习策略，数据点数量: %d", len(data))
	if len(data) < 50 {
		log.Printf("[DEEP_LEARNING] 数据点不足: %d < 50，无法进行深度学习策略", len(data))
		return fmt.Errorf("数据点不足，无法进行深度学习策略")
	}

	config := &result.Config
	log.Printf("[DEEP_LEARNING] 配置: symbol=%s, strategy=%s, initialCash=%.2f, maxPosition=%.3f",
		config.Symbol, config.Strategy, config.InitialCash, config.MaxPosition)

	// 性能优化：预计算所有周期的特征并缓存
	log.Printf("[PERFORMANCE_OPT] 开始预计算特征缓存")
	precomputeStart := time.Now()
	err := be.precomputeFeatures(ctx, data, *config)
	if err != nil {
		log.Printf("[PERFORMANCE_OPT] 特征预计算失败: %v，将使用实时计算", err)
	} else {
		log.Printf("[PERFORMANCE_OPT] 特征预计算完成，耗时: %.2fs", time.Since(precomputeStart).Seconds())
	}

	// 性能优化：预计算所有周期的ML预测并缓存
	log.Printf("[PERFORMANCE_OPT] 开始预计算ML预测缓存")
	mlPrecomputeStart := time.Now()
	mlErr := be.precomputeMLPredictions(ctx, data, *config)
	if mlErr != nil {
		log.Printf("[PERFORMANCE_OPT] ML预测预计算失败: %v，将使用实时计算", mlErr)
	} else {
		log.Printf("[PERFORMANCE_OPT] ML预测预计算完成，耗时: %.2fs", time.Since(mlPrecomputeStart).Seconds())
	}

	// 在开始回测前训练机器学习模型
	log.Printf("[ML_INIT] 开始为 %s 训练机器学习模型", config.Symbol)

	// 优化：只在数据足够时才训练模型，避免不必要的计算
	if len(data) >= 100 {
		err := be.trainMLModelForSymbol(ctx, config.Symbol, data)
		if err != nil {
			log.Printf("[ML_INIT] 模型训练失败，将使用规则决策: %v", err)
		} else {
			log.Printf("[ML_INIT] 模型训练完成，将使用机器学习增强决策")
		}
	} else {
		log.Printf("[ML_INIT] 数据点不足(%d < 100)，跳过ML训练，使用规则决策", len(data))
	}

	log.Printf("[DEEP_LEARNING] 初始化完成，开始回测循环，总数据点: %d", len(data))
	position := 0.0
	cash := config.InitialCash
	holdTime := 0
	lastTradeIndex := -10 // 初始化为足够小的值，确保开始时可以交易

	// 初始化强化学习代理
	agent := be.initializeRLAgent(config)

	// 获取推荐系统的策略建议
	strategyRecommendation := be.getStrategyRecommendationFromAPI(config.Symbol)
	if strategyRecommendation != nil {
		log.Printf("[INFO] 集成推荐系统策略: %s, 入场区间: %.4f-%.4f",
			strategyRecommendation.StrategyType, strategyRecommendation.EntryMin, strategyRecommendation.EntryMax)
		agent["recommended_strategy"] = strategyRecommendation.StrategyType
		agent["entry_zone_min"] = strategyRecommendation.EntryMin
		agent["entry_zone_max"] = strategyRecommendation.EntryMax
		agent["stop_loss_price"] = strategyRecommendation.StopLoss
		agent["take_profit_price"] = strategyRecommendation.TakeProfit
	}

	// 初始化每日收益记录，避免索引越界
	if len(data) > 0 {
		result.DailyReturns = append(result.DailyReturns, DailyReturn{
			Date:   data[0].LastUpdated,
			Value:  cash,
			Return: 0,
		})
	}

	for i, marketData := range data {
		if i < 50 { // 需要足够的历史数据进行深度学习
			if i%20 == 0 { // 每20个周期输出一次进度
				log.Printf("[DEEP_LEARNING] 预热阶段: %d/%d 数据点", i, 50)
			}
			continue
		}

		if i == 50 {
			log.Printf("[DEEP_LEARNING] 预热完成，开始正式交易")
		}

		currentPrice := marketData.Price

		// 1. 获取缓存的特征（预计算，避免重复计算）
		state := be.getCachedFeature(ctx, data, marketData, i, config.Symbol, result.Config.StartDate, result.Config.EndDate)

		// 2. 更新agent状态
		agent["has_position"] = position > 0
		agent["hold_time"] = holdTime
		agent["current_price"] = currentPrice

		// 更新持仓价格（如果有持仓）
		if position > 0 {
			// 如果是第一次设置entry_price，或者需要更新
			if entryPrice, exists := agent["entry_price"].(float64); !exists || entryPrice == 0 {
				agent["entry_price"] = currentPrice
			}
		} else {
			// 没有持仓时重置entry_price
			agent["entry_price"] = 0.0
		}

		// 3. 交易频率控制
		timeSinceLastTrade := i - lastTradeIndex
		action := "hold"
		confidence := 0.0

		minTradeInterval := 2 // 进一步降低到2个周期，提高交易频率
		if timeSinceLastTrade < minTradeInterval {
			// 距离上次交易太近，强制观望
			log.Printf("[TRADE_FREQUENCY] 距离上次交易%d周期 < 最小间隔%d，强制观望", timeSinceLastTrade, minTradeInterval)
		} else {
			// 4. 增强的AI决策：结合市场状况和趋势确认
			action, confidence = be.enhancedTradingDecision(state, agent, i, data[:i+1])

			// 输出决策详情（可配置是否启用详细日志）
			if i%50 == 0 || action != "hold" { // 每50步或有交易时输出
				log.Printf("[AI_DECISION] i=%d, action=%s, confidence=%.3f, has_pos=%v, hold_time=%d",
					i, action, confidence, agent["has_position"], holdTime)
				if action != "hold" {
					log.Printf("[AI_DECISION] 关键指标: trend_5=%.3f, trend_20=%.3f, rsi=%.3f, vol=%.3f",
						state["trend_5"], state["trend_20"], state["rsi_14"], state["volatility_20"])
				}
			}
		}

		// 5. 风险管理：结合VaR和动态止损
		action = be.applyRiskManagement(action, confidence, position, currentPrice, config, result, holdTime, state)

		// 6. 执行交易
		if action == "buy" && position == 0 && cash > 0 {
			log.Printf("[TRADE_CHECK] 满足买入条件: action=buy, position=0, cash=%.2f", cash)

			// 买入逻辑 - 使用智能仓位管理
			positionSize := be.calculateAdvancedPositionSize(cash, currentPrice, config, state, agent)

			log.Printf("[TRADE_EXEC] 计算仓位: 现金=%.2f, 价格=%.4f, 仓位=%.4f",
				cash, currentPrice, positionSize)

			position = positionSize
			cash -= position * currentPrice

			// 更新agent状态
			agent["entry_price"] = currentPrice
			agent["has_position"] = true

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       config.Symbol,
				Side:         "buy",
				Quantity:     position,
				Price:        currentPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          0,
				AIConfidence: confidence,
				Reason:       "深度学习策略买入",
			})

			holdTime = 0       // 重置持仓时间
			lastTradeIndex = i // 更新最后交易索引
			log.Printf("[TRADE] ✅ 执行买入: 价格=%.2f, 数量=%.4f, 总价值=%.2f, 剩余现金=%.2f, 置信度=%.3f",
				currentPrice, position, position*currentPrice, cash, confidence)

			// 在线学习：记录成功的买入决策
			be.addOnlineLearningSample(ctx, state, 1.0, config.Symbol)

		} else if action == "short" && position == 0 && cash > 0 {
			log.Printf("[TRADE_CHECK] 满足做空条件: action=short, position=0, cash=%.2f", cash)

			// 做空逻辑 - 使用智能仓位管理（做空使用更保守的仓位）
			positionSize := be.calculateAdvancedPositionSize(cash, currentPrice, config, state, agent) * 0.7 // 做空更保守

			log.Printf("[TRADE_EXEC] 计算做空仓位: 现金=%.2f, 价格=%.4f, 仓位=%.4f",
				cash, currentPrice, positionSize)

			position = -positionSize            // 负数表示空头仓位
			cash += positionSize * currentPrice // 收到做空所得资金

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       config.Symbol,
				Side:         "short",
				Quantity:     positionSize,
				Price:        currentPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          0,
				AIConfidence: confidence,
				Reason:       "深度学习策略做空",
			})

			holdTime = 0       // 重置持仓时间
			lastTradeIndex = i // 更新最后交易索引
			log.Printf("[TRADE] ✅ 执行做空: 价格=%.2f, 数量=%.4f, 所得资金=%.2f, 剩余现金=%.2f, 置信度=%.3f",
				currentPrice, positionSize, positionSize*currentPrice, cash, confidence)

			// 在线学习：记录成功的做空决策
			be.addOnlineLearningSample(ctx, state, -1.0, config.Symbol)

		} else if action == "cover" && position < 0 {
			log.Printf("[TRADE_CHECK] 满足平空条件: action=cover, position=%.4f", position)

			// 平空逻辑
			shortQuantity := -position // 空头数量（正数）
			coverPrice := currentPrice
			entryPrice := result.Trades[len(result.Trades)-1].Price

			// 计算空头盈利：卖出价 - 平仓价
			pnl := (entryPrice - coverPrice) * shortQuantity

			result.Trades = append(result.Trades, TradeRecord{
				Symbol:       config.Symbol,
				Side:         "cover",
				Quantity:     shortQuantity,
				Price:        coverPrice,
				Timestamp:    marketData.LastUpdated,
				Commission:   0,
				PnL:          pnl,
				AIConfidence: confidence,
				Reason:       "深度学习策略平空",
			})

			cash -= shortQuantity * coverPrice // 支付平仓资金
			position = 0                       // 清空仓位

			// 更新agent状态
			agent["has_position"] = false
			agent["entry_price"] = 0.0
			holdTime = 0       // 重置持仓时间
			lastTradeIndex = i // 更新最后交易索引
			log.Printf("[TRADE] ✅ 执行平空: 价格=%.2f, 数量=%.4f, 盈亏=%.2f, 现金=%.2f, 置信度=%.3f",
				coverPrice, shortQuantity, pnl, cash, confidence)

			// 在线学习：记录成功的平空决策
			be.addOnlineLearningSample(ctx, state, 1.0, config.Symbol)

		} else if action == "sell" && position > 0 {
			log.Printf("[TRADE_CHECK] 满足卖出条件: action=sell, position=%.4f", position)

			// 增强卖出逻辑：检查是否应该部分卖出还是全部卖出
			exitStrategy := be.determineExitStrategy(position, result.Trades[len(result.Trades)-1].Price, currentPrice, state, holdTime)

			if exitStrategy.shouldExit {
				// 计算卖出数量和价格
				exitQuantity := position * exitStrategy.exitRatio
				exitPrice := currentPrice
				entryPrice := result.Trades[len(result.Trades)-1].Price

				// 计算部分头寸的盈亏
				positionPNL := (exitPrice - entryPrice) * exitQuantity

				result.Trades = append(result.Trades, TradeRecord{
					Symbol:       config.Symbol,
					Side:         "sell",
					Quantity:     exitQuantity,
					Price:        exitPrice,
					Timestamp:    marketData.LastUpdated,
					Commission:   0,
					PnL:          positionPNL,
					AIConfidence: confidence,
					Reason:       fmt.Sprintf("深度学习策略卖出-%s", exitStrategy.reason),
				})

				cash += exitQuantity * exitPrice
				position -= exitQuantity

				// 更新agent状态
				if position == 0 {
					agent["has_position"] = false
					agent["entry_price"] = 0.0
					holdTime = 0
				}

				lastTradeIndex = i // 更新最后交易索引
				log.Printf("[TRADE] ✅ 执行卖出: 价格=%.2f, 数量=%.4f (%.1f%%), 盈亏=%.2f, 剩余仓位=%.4f, 现金=%.2f, 原因=%s",
					exitPrice, exitQuantity, exitStrategy.exitRatio*100, positionPNL, position, cash, exitStrategy.reason)

				// 在线学习：记录成功的卖出决策
				be.addOnlineLearningSample(ctx, state, -1.0, config.Symbol)
			}
		}

		// 更新持仓时间
		if position > 0 {
			holdTime++
		}

		// 更新每日收益
		result.DailyReturns = append(result.DailyReturns, DailyReturn{
			Date:   marketData.LastUpdated,
			Value:  cash + position*currentPrice,
			Return: 0, // 这里可以计算日收益率
		})
	}

	// 计算最终绩效指标
	be.calculateSummary(result, config.InitialCash)
	be.calculatePerformanceMetrics(result)

	// 详细的执行总结
	buyTrades := 0
	sellTrades := 0
	totalPnL := 0.0
	for _, trade := range result.Trades {
		if trade.Side == "buy" {
			buyTrades++
		} else if trade.Side == "sell" {
			sellTrades++
			totalPnL += trade.PnL
		}
	}

	// 更新权重控制器
	if be.server != nil && be.server.backtestEngine != nil && be.server.backtestEngine.weightController != nil {
		// 创建简化的性能指标用于权重学习
		metrics := map[string]float64{
			"win_rate":     result.Performance.WinRate,
			"total_return": result.Performance.TotalReturn,
			"sharpe_ratio": result.Performance.SharpeRatio,
			"max_drawdown": result.Performance.MaxDrawdown,
			"total_trades": float64(len(result.Trades)),
		}

		// 更新全局指标
		be.server.backtestEngine.weightController.UpdateGlobalMetrics(metrics)

		// 更新币种特定指标
		manager := be.server.backtestEngine.weightController.GetManager(config.Symbol)
		manager.UpdatePerformance(config.Strategy, metrics)
	}

	log.Printf("[DEEP_LEARNING] ============= 执行总结 =============")
	log.Printf("[DEEP_LEARNING] 策略: %s", config.Strategy)
	log.Printf("[DEEP_LEARNING] 币种: %s", config.Symbol)
	log.Printf("[DEEP_LEARNING] 时间范围: %s 到 %s", config.StartDate, config.EndDate)
	log.Printf("[DEEP_LEARNING] 数据点总数: %d", len(data))
	log.Printf("[DEEP_LEARNING] 交易周期: %d (跳过前50个数据点)", len(data)-50)
	log.Printf("[DEEP_LEARNING] 买入交易: %d", buyTrades)
	log.Printf("[DEEP_LEARNING] 卖出交易: %d", sellTrades)
	log.Printf("[DEEP_LEARNING] 总交易笔数: %d", len(result.Trades))
	log.Printf("[DEEP_LEARNING] 总盈亏: %.2f", totalPnL)
	log.Printf("[DEEP_LEARNING] 最终资金: %.2f", cash)
	log.Printf("[DEEP_LEARNING] 总收益率: %.2f%%", result.Performance.TotalReturn*100)
	log.Printf("[DEEP_LEARNING] 胜率: %.2f%%", result.Performance.WinRate*100)
	log.Printf("[DEEP_LEARNING] ===================================")

	return nil
}

// initializeRLAgent 初始化强化学习代理
func (be *BacktestEngine) initializeRLAgent(config *BacktestConfig) map[string]interface{} {
	agent := make(map[string]interface{})

	// 初始化代理状态
	agent["has_position"] = false
	agent["hold_time"] = 0
	agent["entry_price"] = 0.0
	agent["stop_loss"] = config.StopLoss
	agent["take_profit"] = config.TakeProfit
	agent["max_position"] = config.MaxPosition

	// 推荐系统相关状态
	agent["recommended_strategy"] = ""
	agent["entry_zone_min"] = 0.0
	agent["entry_zone_max"] = 0.0
	agent["stop_loss_price"] = 0.0
	agent["take_profit_price"] = 0.0

	return agent
}

// validateStateConsistency 检查state数据一致性
func (be *BacktestEngine) validateStateConsistency(state map[string]float64, context string) {
	// 检查关键指标的合理性
	if rsi, exists := state["rsi_14"]; exists {
		if rsi < 0 || rsi > 100 {
			log.Printf("[STATE_VALIDATION] %s: RSI异常值 %.2f，重置为50", context, rsi)
			state["rsi_14"] = 50.0
		}
	}

	if volatility, exists := state["volatility_20"]; exists {
		if volatility < 0 || volatility > 1.0 {
			log.Printf("[STATE_VALIDATION] %s: 波动率异常值 %.4f，重置为0.02", context, volatility)
			state["volatility_20"] = 0.02
		}
	}

	if trend, exists := state["trend_20"]; exists {
		if trend < -1.0 || trend > 1.0 {
			log.Printf("[STATE_VALIDATION] %s: 趋势异常值 %.4f，重置为0", context, trend)
			state["trend_20"] = 0.0
		}
	}

	if momentum, exists := state["momentum_10"]; exists {
		if momentum < -1.0 || momentum > 1.0 {
			log.Printf("[STATE_VALIDATION] %s: 动量异常值 %.4f，重置为0", context, momentum)
			state["momentum_10"] = 0.0
		}
	}
}

// enhancedTradingDecision 增强的交易决策系统
func (be *BacktestEngine) enhancedTradingDecision(state map[string]float64, agent map[string]interface{}, currentIndex int, historicalData []MarketData) (string, float64) {
	// 数据一致性检查
	be.validateStateConsistency(state, "开始决策")

	// 1. 市场状况分析
	marketRegime := be.analyzeMarketRegime(state, historicalData)

	// === 熊市环境优先检测 ===
	isBearMarket := be.detectBearMarketEnvironment(state, historicalData, marketRegime)
	if isBearMarket {
		return be.handleBearMarketDecision(state, agent, marketRegime)
	}

	// 2. 趋势强度和确认
	trendStrength := be.calculateEnhancedTrendStrength(state)
	trendConfirmation := be.confirmTrendDirection(state, historicalData)

	// 3. 动量和反转信号
	momentumSignal := be.analyzeMomentumSignals(state)

	// 4. 波动率调整
	volatility := state["volatility_20"]
	position := agent["has_position"].(bool)

	// 5. 获取ML决策和规则决策
	mlDecision := be.getMLDecision(state, agent, marketRegime, position)
	ruleDecision := be.getRuleDecision(state, agent, marketRegime, trendStrength, trendConfirmation, momentumSignal, volatility, position)

	// 6. 决策融合
	finalAction, finalConfidence := be.fuseDecisions(mlDecision, ruleDecision, state, marketRegime, position)

	log.Printf("[ENHANCED_DECISION] 市场状况:%s, 趋势强度:%.2f, 确认度:%.2f, 动量:%.2f, 波动率:%.3f",
		marketRegime, trendStrength, trendConfirmation, momentumSignal, volatility)

	// 记录融合详情
	symbol := ""
	if sym, ok := agent["symbol"].(string); ok {
		symbol = sym
	}
	// 计算实际使用的阈值（基于市场环境）
	actualBuyThreshold := 0.03   // 默认值，与sideways市场一致
	actualSellThreshold := -0.15 // 默认值
	switch marketRegime {
	case "strong_bull":
		actualBuyThreshold = 0.15
		actualSellThreshold = -0.20
	case "weak_bull":
		actualBuyThreshold = 0.20
		actualSellThreshold = -0.25
	case "sideways", "true_sideways":
		actualBuyThreshold = 0.03 // 与上面修改一致
		actualSellThreshold = -0.15
	case "strong_bear", "weak_bear":
		actualBuyThreshold = 0.30
		actualSellThreshold = -0.20
	}
	be.logFusionDetails(mlDecision, ruleDecision, finalAction, finalConfidence, state, marketRegime, symbol, actualBuyThreshold, actualSellThreshold)

	return finalAction, finalConfidence
}

// analyzeMarketRegime 分析市场状况 - 增强版，更健壮
func (be *BacktestEngine) analyzeMarketRegime(state map[string]float64, historicalData []MarketData) string {
	// 基于多个指标判断市场状况，使用更宽松的检查

	// 获取可用指标，默认值处理
	trendStrength := 0.0
	if ts, exists := state["trend_strength_20"]; exists && ts > 0 {
		trendStrength = ts
	} else if ts, exists := state["trend_strength"]; exists && ts > 0 {
		trendStrength = ts
	}

	volatility := 0.05 // 默认低波动
	if vol, exists := state["volatility_20"]; exists && vol > 0 {
		volatility = vol
	}

	rsi := 50.0 // 默认中性
	if r, exists := state["rsi_14"]; exists && r >= 0 && r <= 100 {
		rsi = r
	} else if exists && (r < 0 || r > 100) {
		log.Printf("[MARKET_REGIME_WARN] RSI值异常: %.2f，使用默认值50", r)
		rsi = 50.0
	}

	momentum := 0.0
	if m, exists := state["momentum_5"]; exists {
		momentum = m
	} else if m, exists := state["momentum_10"]; exists {
		momentum = m
	}

	// 1. 基于趋势强度和波动率的初步判断
	if trendStrength > 0.6 && volatility < 0.2 {
		// 强趋势 + 低波动 = 明确趋势
		if momentum > 0.3 {
			return "strong_bull"
		} else if momentum < -0.3 {
			return "strong_bear"
		}
	}

	if trendStrength > 0.3 && trendStrength <= 0.6 && volatility < 0.3 {
		// 中等趋势 + 适中波动 = 弱趋势
		if momentum > 0.1 {
			return "weak_bull"
		} else if momentum < -0.1 {
			return "weak_bear"
		}
	}

	// 2. 基于波动率和RSI的震荡判断
	if volatility > 0.15 {
		// 高波动环境
		if rsi > 65 {
			return "overbought_range"
		} else if rsi < 35 {
			return "oversold_range"
		} else {
			return "volatile"
		}
	}

	// 3. 低波动环境的判断
	if volatility <= 0.1 {
		if trendStrength > 0.2 {
			return "trending" // 有一定趋势，低波动
		} else {
			return "sideways" // 无明显趋势，低波动
		}
	}

	// 4. 默认情况：中等波动，无明确信号
	// 检查是否有任何积极信号
	hasPositiveSignals := (trendStrength > 0.1) || (math.Abs(momentum) > 0.1) ||
		(rsi > 55 || rsi < 45) || (volatility > 0.1)

	if hasPositiveSignals {
		return "neutral" // 有一些信号但不够明确
	} else {
		return "sideways" // 完全没有信号，纯震荡
	}
}

// calculateEnhancedTrendStrength 计算增强趋势强度
func (be *BacktestEngine) calculateEnhancedTrendStrength(state map[string]float64) float64 {
	trend5 := state["trend_5"]
	trend20 := state["trend_20"]
	adx := state["adx_14"]

	// 综合趋势强度
	trendConsistency := 1.0 - math.Abs(trend5-trend20) // 趋势一致性
	trendMagnitude := math.Abs(trend20)                // 趋势大小

	strength := (trendConsistency*0.4 + trendMagnitude*0.4 + adx*0.2)
	return math.Min(strength, 1.0) // 归一化到[0,1]
}

// confirmTrendDirection 确认趋势方向
func (be *BacktestEngine) confirmTrendDirection(state map[string]float64, historicalData []MarketData) float64 {
	// 使用多个指标确认趋势
	confirmations := 0.0
	totalChecks := 0.0

	// 1. 移动平均线排列
	if ma5, ok := state["ma_5"]; ok {
		if ma20, ok := state["ma_20"]; ok {
			totalChecks++
			if ma5 > ma20 {
				confirmations++ // 多头排列
			}
		}
	}

	// 2. MACD确认
	if macd, ok := state["macd"]; ok {
		if signal, ok := state["macd_signal"]; ok {
			totalChecks++
			if macd > signal {
				confirmations++ // MACD多头
			}
		}
	}

	// 3. RSI趋势
	if rsi, ok := state["rsi_14"]; ok {
		totalChecks++
		if rsi > 50 {
			confirmations += 0.5 // RSI中性偏多
		}
	}

	if totalChecks > 0 {
		return confirmations / totalChecks
	}
	return 0.5 // 默认中性
}

// analyzeMomentumSignals 分析动量信号
func (be *BacktestEngine) analyzeMomentumSignals(state map[string]float64) float64 {
	momentum := 0.0

	// RSI动量
	if rsi, ok := state["rsi_14"]; ok {
		if rsi > 70 {
			momentum -= 0.3 // 超买，负动量
		} else if rsi < 30 {
			momentum += 0.3 // 超卖，正动量
		}
	}

	// 价格动量 (短期vs长期)
	if trend5, ok := state["trend_5"]; ok {
		if trend20, ok := state["trend_20"]; ok {
			momentumDiff := trend5 - trend20
			momentum += momentumDiff * 0.4 // 动量差异
		}
	}

	// 成交量动量
	if volRatio, ok := state["volume_ratio"]; ok {
		momentum += volRatio * 0.2
	}

	return momentum
}

// applyEnhancedDecisionLogic 应用增强决策逻辑
func (be *BacktestEngine) applyEnhancedDecisionLogic(baseAction string, baseConfidence float64,
	marketRegime string, trendStrength float64, trendConfirmation float64,
	momentumSignal float64, volatility float64, hasPosition bool, state map[string]float64) (string, float64) {

	action := baseAction
	confidence := baseConfidence

	// 根据市场状况调整决策
	switch marketRegime {
	case "strong_bull":
		// 强牛市：更倾向于买入，降低卖出阈值
		if !hasPosition && baseAction == "hold" && trendConfirmation > 0.6 {
			action = "buy"
			confidence = math.Min(baseConfidence+0.2, 0.9)
			log.Printf("[MARKET_ADAPT] 强牛市调整：hold→buy, 趋势确认度=%.2f", trendConfirmation)
		}

	case "strong_bear":
		// 强熊市：非常谨慎，只在极端情况下交易
		if hasPosition && trendConfirmation < 0.3 {
			action = "sell"
			confidence = math.Min(baseConfidence+0.1, 0.8)
			log.Printf("[MARKET_ADAPT] 强熊市调整：持有→卖出, 趋势确认度=%.2f", trendConfirmation)
		} else if !hasPosition {
			// 熊市不买入，除非有非常强的信号
			action = "hold"
			confidence = math.Max(baseConfidence-0.2, 0.1)
			log.Printf("[MARKET_ADAPT] 强熊市调整：不买入，保持观望")
		}

	case "overbought_range":
		// 震荡超买：倾向于卖出
		if hasPosition && momentumSignal < -0.1 {
			action = "sell"
			confidence = math.Min(baseConfidence+0.15, 0.85)
			log.Printf("[MARKET_ADAPT] 震荡超买：增加卖出倾向")
		}

	case "oversold_range":
		// 震荡超卖：倾向于买入
		if !hasPosition && momentumSignal > 0.1 {
			action = "buy"
			confidence = math.Min(baseConfidence+0.15, 0.85)
			log.Printf("[MARKET_ADAPT] 震荡超卖：增加买入倾向")
		}
	}

	// 波动率调整
	if volatility > 0.20 {
		// 高波动：降低信心，减少交易
		confidence *= 0.8
		log.Printf("[VOLATILITY_ADAPT] 高波动环境：信心降低至%.3f", confidence)
	} else if volatility < 0.08 {
		// 低波动：适度增加信心，但不超过0.85
		confidence = math.Min(confidence*1.05, 0.85)
	}

	// 趋势强度过滤
	if trendStrength < 0.3 && math.Abs(momentumSignal) < 0.1 {
		// 弱趋势且无明显动量：倾向于hold
		if action != "hold" {
			log.Printf("[TREND_FILTER] 弱趋势环境：%s→hold", action)
			action = "hold"
			confidence *= 0.7
		}
	}

	return action, confidence
}

// ExitStrategy 退出策略
type ExitStrategy struct {
	shouldExit bool
	exitRatio  float64 // 退出比例 (0.0-1.0)
	reason     string
}

// determineExitStrategy 确定退出策略
func (be *BacktestEngine) determineExitStrategy(position, entryPrice, currentPrice float64, state map[string]float64, holdTime int) ExitStrategy {
	// 1. 检查时间-based退出 (持有太久)
	if holdTime > 50 { // 超过50周期强制退出
		return ExitStrategy{
			shouldExit: true,
			exitRatio:  1.0,
			reason:     "时间退出-持有过久",
		}
	}

	// 2. 检查收益目标 (盈利时部分退出)
	currentReturn := (currentPrice - entryPrice) / entryPrice
	if currentReturn > 0.05 { // 盈利5%以上
		if currentReturn > 0.10 { // 盈利10%以上，全出
			return ExitStrategy{
				shouldExit: true,
				exitRatio:  1.0,
				reason:     "止盈退出-高收益",
			}
		} else { // 盈利5-10%，出50%
			return ExitStrategy{
				shouldExit: true,
				exitRatio:  0.5,
				reason:     "部分止盈",
			}
		}
	}

	// 3. 检查止损条件
	if currentReturn < -0.03 { // 亏损3%以上
		return ExitStrategy{
			shouldExit: true,
			exitRatio:  1.0,
			reason:     "止损退出",
		}
	}

	// 4. 追踪止损 (基于移动平均)
	if ma20, ok := state["ma_20"]; ok {
		if currentPrice < ma20*0.98 { // 跌破20日均线2%
			return ExitStrategy{
				shouldExit: true,
				exitRatio:  1.0,
				reason:     "追踪止损-MA突破",
			}
		}
	}

	// 5. 动量反转信号
	if rsi, ok := state["rsi_14"]; ok {
		if rsi > 75 { // RSI超买
			return ExitStrategy{
				shouldExit: true,
				exitRatio:  0.3, // 小幅减仓
				reason:     "动量反转-RSI超买",
			}
		}
	}

	// 6. 趋势反转信号
	if trend5, ok := state["trend_5"]; ok {
		if trend20, ok := state["trend_20"]; ok {
			if trend5 < -0.01 && trend20 > 0.005 { // 短期转跌但长期仍涨
				return ExitStrategy{
					shouldExit: true,
					exitRatio:  0.4,
					reason:     "趋势警告-短期反转",
				}
			}
		}
	}

	// 默认：不退出
	return ExitStrategy{
		shouldExit: false,
		exitRatio:  0.0,
		reason:     "继续持有",
	}
}

// Decision 决策结构体
type Decision struct {
	Action     string
	Confidence float64
	Score      float64
	Quality    float64
}

// getMLDecision 获取机器学习决策
func (be *BacktestEngine) getMLDecision(state map[string]float64, agent map[string]interface{}, marketRegime string, position bool) *Decision {
	positionStatus := "no_position"
	if position {
		positionStatus = "long_position"
	}

	// 如果有机器学习模块，使用增强决策
	if be.machineLearning != nil {
		symbol := agent["symbol"].(string)
		mlResult, err := be.machineLearning.MakeMLDecision(context.Background(), symbol, marketRegime, positionStatus)
		if err == nil {
			return &Decision{
				Action:     mlResult.Action,
				Confidence: mlResult.Confidence,
				Score:      mlResult.Score,
				Quality:    mlResult.Quality,
			}
		}
		log.Printf("[ML_DECISION] ML决策失败，使用规则决策: %v", err)
	}

	// 回退到基础AI决策
	baseAction, baseConfidence := be.deepLearningDecision(state, agent)

	return &Decision{
		Action:     baseAction,
		Confidence: baseConfidence,
		Score:      baseConfidence, // 简化处理
		Quality:    0.5,            // 默认中等质量
	}
}

// getRuleDecision 获取规则决策
func (be *BacktestEngine) getRuleDecision(state map[string]float64, agent map[string]interface{}, marketRegime string, trendStrength, trendConfirmation, momentumSignal, volatility float64, position bool) *Decision {
	// 使用现有的增强决策逻辑
	baseAction, baseConfidence := be.deepLearningDecision(state, agent)

	action, confidence := be.applyEnhancedDecisionLogic(baseAction, baseConfidence, marketRegime,
		trendStrength, trendConfirmation, momentumSignal, volatility, position, state)

	return &Decision{
		Action:     action,
		Confidence: confidence,
		Score:      confidence, // 规则决策的分数等于置信度
		Quality:    0.8,        // 规则决策质量较高
	}
}

// fuseDecisions 融合ML决策和规则决策
func (be *BacktestEngine) fuseDecisions(mlDecision, ruleDecision *Decision, state map[string]float64, marketRegime string, position bool) (string, float64) {
	// === P2优化：趋势确认机制 - 在弱势市场减少交易 ===
	trendStrength := 0.0
	if ts, exists := state["trend_20"]; exists {
		trendStrength = math.Abs(ts)
	}

	momentumStrength := 0.0
	if ms, exists := state["momentum_10"]; exists {
		momentumStrength = math.Abs(ms)
	}

	// P2优化：放宽趋势确认条件，在横盘市场允许小仓位交易
	if marketRegime == "sideways" || marketRegime == "true_sideways" {
		minTrendStrength := 0.001    // P2优化：降低趋势强度阈值，从0.005降至0.001
		minMomentumStrength := 0.005 // P2优化：降低动量阈值，从0.03降至0.005

		// 如果趋势和动量都过弱，则强制观望
		if trendStrength < minTrendStrength && momentumStrength < minMomentumStrength {
			log.Printf("[TREND_CONFIRMATION] 横盘市场趋势和动量均过弱(%.4f, %.4f)，强制观望", trendStrength, momentumStrength)
			return "hold", 0.1
		}

		// 如果只有一方弱，则降低置信度但允许交易
		if trendStrength < minTrendStrength || momentumStrength < minMomentumStrength {
			log.Printf("[TREND_CONFIRMATION] 横盘市场趋势或动量偏弱(%.4f, %.4f)，降低置信度继续交易", trendStrength, momentumStrength)
			// 降低决策置信度，但不完全禁止
			if mlDecision != nil {
				mlDecision.Confidence *= 0.8
			}
			if ruleDecision != nil {
				ruleDecision.Confidence *= 0.8
			}
		}
	}

	// 在弱牛/弱熊市场，需要更强的趋势确认
	if marketRegime == "weak_bull" || marketRegime == "weak_bear" {
		minTrendStrength := 0.008 // 弱势市场需要更强趋势
		if trendStrength < minTrendStrength {
			log.Printf("[TREND_CONFIRMATION] 弱势市场趋势不足(%.4f < %.4f)，减少交易", trendStrength, minTrendStrength)
			// 降低决策置信度，但不完全禁止
			mlDecision.Confidence *= 0.7
			ruleDecision.Confidence *= 0.7
		}
	}

	// === 熊市检测 ===
	isBearMarket := strings.Contains(marketRegime, "bear") || marketRegime == "weak_bear" || marketRegime == "strong_bear"

	// 1. 权重调整（基于质量、市场环境和历史学习）
	featureQuality := 1.0
	if completeness := state["feature_completeness"]; completeness > 0 {
		featureQuality = completeness
	}

	// 市场条件评分（基于波动率和趋势）
	marketConditionScore := 0.5
	if volatility, ok := state["volatility_20"]; ok {
		if volatility < 0.02 {
			marketConditionScore = 0.8 // 低波动，条件好
		} else if volatility > 0.08 {
			marketConditionScore = 0.3 // 高波动，条件差
		}
	}

	weightAdjustment := be.machineLearning.AdjustWeightsForQuality(mlDecision.Quality, featureQuality, marketConditionScore)

	mlWeight := weightAdjustment.MLWeight
	ruleWeight := weightAdjustment.RuleWeight

	// === 熊市权重重构：大幅提高规则权重 - 第一阶段改进 ===
	if isBearMarket {
		// 熊市中ML预测不可靠，优先使用规则策略
		originalMLWeight := mlWeight
		originalRuleWeight := ruleWeight

		ruleWeight = math.Min(0.85, ruleWeight*2.5) // 规则权重最高85%，至少翻2.5倍
		mlWeight = 1.0 - ruleWeight                 // ML权重相应降低

		log.Printf("[BEAR_DECISION_FUSION] 熊市环境权重调整: ML %.2f->%.2f, 规则 %.2f->%.2f",
			originalMLWeight, mlWeight, originalRuleWeight, ruleWeight)

		// === 熊市一致性要求：必须ML和规则决策一致才交易 ===
		mlAction := "hold"
		if mlDecision.Score >= 0.5 { // 使用较低阈值判断ML倾向
			mlAction = mlDecision.Action
		}

		ruleAction := "hold"
		if ruleDecision.Score >= 0.5 { // 使用较低阈值判断规则倾向
			ruleAction = ruleDecision.Action
		}

		// 如果ML和规则决策不一致，熊市环境下直接返回hold
		if mlAction != ruleAction && mlAction != "hold" && ruleAction != "hold" {
			log.Printf("[BEAR_CONSISTENCY_CHECK] 熊市环境ML(%s)和规则(%s)决策不一致，强制观望",
				mlAction, ruleAction)
			return "hold", 0.1
		}
	}

	// 2. 分数融合
	mlScore := mlDecision.Score * mlDecision.Confidence
	ruleScore := ruleDecision.Score * ruleDecision.Confidence

	fusedScore := (mlScore * mlWeight) + (ruleScore * ruleWeight)

	// 3. 决策映射 - 第一阶段改进：提前设置阈值
	var finalAction string
	var finalConfidence float64

	// 设置市场环境相关的阈值 - 熊市增强版
	buyThreshold := 0.25   // 提高默认买入阈值
	sellThreshold := -0.25 // 降低默认卖出阈值
	shortThreshold := -0.9 // 提高做空阈值

	// === 第一阶段改进：添加决策一致性检查 ===
	// 在非极端市场环境下，要求ML和规则决策方向一致
	requireConsistency := marketRegime == "sideways" || marketRegime == "true_sideways" || marketRegime == "weak_bull" || marketRegime == "neutral"
	if requireConsistency {
		mlAction := "hold"
		if mlScore >= buyThreshold {
			mlAction = "buy"
		} else if mlScore <= sellThreshold {
			mlAction = "sell"
		}

		ruleAction := "hold"
		if ruleScore >= buyThreshold {
			ruleAction = "buy"
		} else if ruleScore <= sellThreshold {
			ruleAction = "sell"
		}

		// P1优化：改进决策一致性检查，允许一定程度的不一致
		if mlAction != ruleAction && mlAction != "hold" && ruleAction != "hold" {
			// 检查置信度差异，如果一方置信度显著高于另一方，减少惩罚
			mlConfidence := mlDecision.Confidence
			ruleConfidence := ruleDecision.Confidence

			confidenceDiff := math.Abs(mlConfidence - ruleConfidence)
			maxConfidence := math.Max(mlConfidence, ruleConfidence)

			// 如果置信度差异不大或者一方有很高置信度，减少惩罚
			penaltyFactor := 0.8 // 默认降低20%的权重
			if confidenceDiff < 0.2 && maxConfidence > 0.8 {
				// 双方置信度相近且都很高：只降低10%
				penaltyFactor = 0.9
				log.Printf("[DECISION_CONSISTENCY] 高置信度不一致，适度降低权重")
			} else if maxConfidence > 0.9 {
				// 一方置信度极高：只降低5%
				penaltyFactor = 0.95
				log.Printf("[DECISION_CONSISTENCY] 一方高置信度，微调权重")
			} else {
				log.Printf("[DECISION_CONSISTENCY] 一般不一致，降低融合权重")
			}

			fusedScore *= penaltyFactor

			// 在真正横盘市场，如果决策不一致且没有高置信度，直接返回hold
			if marketRegime == "true_sideways" && maxConfidence < 0.85 {
				log.Printf("[TRUE_SIDEWAYS_CONSERVATIVE] 真正横盘市场决策不一致且置信度不足，强制观望")
				return "hold", math.Max(0.1, 1.0-math.Abs(fusedScore))
			}
		}
	}

	if isBearMarket {
		// === 熊市决策阈值：极度保守 - 第一阶段改进 ===
		buyThreshold = 0.70   // 熊市买入阈值大幅提高到70%（从60%提高）
		sellThreshold = -0.05 // 熊市卖出阈值降低到-5%（从-10%收紧）
		shortThreshold = -0.8 // 熊市做空阈值稍微降低

		log.Printf("[BEAR_DECISION_THRESHOLD] 熊市环境阈值调整: 买入=%.2f, 卖出=%.2f, 做空=%.2f",
			buyThreshold, sellThreshold, shortThreshold)
	} else {
		// 非熊市环境的正常阈值
		switch marketRegime {
		case "strong_bull":
			buyThreshold = 0.15 // 强牛市可以稍微激进
			sellThreshold = -0.20
		case "strong_bear":
			buyThreshold = 0.30 // 强熊市非常谨慎
			sellThreshold = -0.15
		case "weak_bull":
			buyThreshold = 0.20
			sellThreshold = -0.25
		case "weak_bear":
			buyThreshold = 0.30
			sellThreshold = -0.20
		case "sideways":
			buyThreshold = 0.03 // 紧急修复：大幅降低买入阈值，从0.20降至0.03，激活横盘市场交易
			sellThreshold = -0.15
		case "true_sideways":
			buyThreshold = 0.02 // 紧急修复：大幅降低买入阈值，从0.40降至0.02，激活真正横盘市场的小额交易
			sellThreshold = -0.10
			shortThreshold = -0.95 // 几乎不可能做空
		case "volatile":
			buyThreshold = 0.40 // 高波动市场极度谨慎
			sellThreshold = -0.40
		case "trending":
			buyThreshold = 0.18 // 明确趋势市场可以稍激进
			sellThreshold = -0.18
		case "neutral":
			// 中性市场：适度保守，保持一定活跃度
			buyThreshold = 0.25 // 降低买入阈值，增加交易机会
			sellThreshold = -0.50
			shortThreshold = -0.95
		default:
			// 未知市场环境：保守处理
			buyThreshold = 0.40
			sellThreshold = -0.40
		}
	}

	// 决策优先级：增强保守策略
	// 只有在规则决策具有压倒性优势且市场环境支持时才优先执行
	ruleAdvantage := ruleDecision.Confidence - mlDecision.Confidence

	// 在中性或弱信号市场环境下，适度提高决策门槛
	marketConservative := marketRegime == "neutral" || marketRegime == "sideways"
	if marketConservative {
		buyThreshold *= 1.1  // 在保守市场中适度提高买入阈值（从1.2降到1.1）
		sellThreshold *= 1.1 // 适度提高卖出阈值
	}

	// 应用动态阈值调整（基于历史表现）
	performance := be.getPerformanceMetrics()
	buyThreshold, sellThreshold = adjustThresholdsDynamically(buyThreshold, sellThreshold, performance, marketRegime)

	// 规则决策优先条件：需要更高的置信度和优势
	ruleOverrideCondition := ruleDecision.Confidence > 0.95 &&
		ruleAdvantage > 0.3 &&
		ruleDecision.Score > buyThreshold

	if ruleOverrideCondition {
		log.Printf("[DECISION_OVERRIDE] 规则决策压倒性优势 (%.2f vs %.2f)，优先执行规则: %s",
			ruleDecision.Confidence, mlDecision.Confidence, ruleDecision.Action)
		finalAction = ruleDecision.Action
		finalConfidence = ruleDecision.Confidence
	} else {
		// 基于融合分数决策
		if fusedScore >= buyThreshold {
			// 在保守市场中，需要适度的信号确认（降低阈值从1.2倍到1.1倍）
			if marketConservative && fusedScore < buyThreshold*1.1 {
				finalAction = "hold"
				finalConfidence = math.Max(0.2, 1.0-fusedScore)
				log.Printf("[CONSERVATIVE_DECISION] 中性市场适度信号，保持观望: 融合分数%.3f < 阈值%.3f",
					fusedScore, buyThreshold*1.1)
			} else {
				finalAction = "buy"
				finalConfidence = math.Min(fusedScore, 1.0)
			}
		} else if fusedScore <= sellThreshold {
			finalAction = "sell"
			finalConfidence = math.Min(-fusedScore, 1.0)
		} else if fusedScore <= shortThreshold && !position {
			finalAction = "short"
			finalConfidence = math.Min(-fusedScore, 1.0)
		} else {
			finalAction = "hold"
			finalConfidence = math.Max(0.1, 1.0-math.Abs(fusedScore))
		}
	}

	log.Printf("[FUSION] 增强融合完成: ML(%s,%.3f,c=%.2f) + 规则(%s,%.3f) -> 最终: %s(%.3f)",
		mlDecision.Action, mlDecision.Score, mlDecision.Confidence,
		ruleDecision.Action, ruleDecision.Score, finalAction, finalConfidence)

	// === 小仓位测试交易：激活横盘市场交易 ===
	// 在横盘市场环境下，如果分数接近阈值但未达到，允许小仓位测试交易
	if finalAction == "hold" && (marketRegime == "sideways" || marketRegime == "true_sideways") {
		// 检查是否接近买入阈值（阈值的80%-120%之间）
		nearBuyThreshold := fusedScore >= buyThreshold*0.8 && fusedScore < buyThreshold
		nearSellThreshold := fusedScore <= sellThreshold*0.8 && fusedScore > sellThreshold

		if nearBuyThreshold && !position {
			// 允许小仓位买入测试
			finalAction = "test_buy"                                     // 特殊标记，表示小仓位测试买入
			finalConfidence = math.Max(0.1, fusedScore/buyThreshold*0.5) // 降低置信度
			log.Printf("[TEST_TRADE] 横盘市场小仓位测试买入: 分数%.3f 接近阈值%.3f", fusedScore, buyThreshold)

		} else if nearSellThreshold && position {
			// 允许小仓位卖出测试
			finalAction = "test_sell"                                      // 特殊标记，表示小仓位测试卖出
			finalConfidence = math.Max(0.1, -fusedScore/sellThreshold*0.5) // 降低置信度
			log.Printf("[TEST_TRADE] 横盘市场小仓位测试卖出: 分数%.3f 接近阈值%.3f", fusedScore, sellThreshold)
		}
	}

	return finalAction, finalConfidence
}

// logFusionDetails 记录融合详情
func (be *BacktestEngine) logFusionDetails(mlDecision, ruleDecision *Decision, finalAction string, finalConfidence float64, state map[string]float64, marketRegime string, symbol string, buyThreshold, sellThreshold float64) {
	// 特征质量评估
	featureQuality := 1.0
	if completeness := state["feature_completeness"]; completeness > 0 {
		featureQuality = completeness
	}

	// 权重计算
	weightAdjustment := be.machineLearning.AdjustWeightsForQuality(mlDecision.Quality, featureQuality, 0.5)

	log.Printf("[FUSION_DETAIL] 特征质量: %.2f, ML权重: %.2f, 规则权重: %.2f, 融合分数: %.3f",
		featureQuality, weightAdjustment.MLWeight, weightAdjustment.RuleWeight, finalConfidence)

	log.Printf("[ADAPTIVE_THRESHOLD] 市场环境: %s, 买入阈值: %.2f, 卖出阈值: %.2f, 最终决策: %s(%.3f)",
		marketRegime, buyThreshold, sellThreshold, finalAction, finalConfidence)

	// 记录决策用于历史学习（这里记录决策本身，实际收益在交易执行后记录）
	if symbol != "" && be.machineLearning != nil {
		finalDecision := &Decision{
			Action:     finalAction,
			Confidence: finalConfidence,
			Score:      finalConfidence,
			Quality:    0.8, // 融合决策质量
		}

		// 临时记录，实际收益将在executeMultiSymbolTrade后更新
		be.machineLearning.RecordDecisionOutcome(symbol, mlDecision, ruleDecision, finalDecision, 0.0, marketRegime)
	}
}

// === 熊市环境适应性增强函数 ===

// detectBearMarketEnvironment 检测熊市环境
func (be *BacktestEngine) detectBearMarketEnvironment(state map[string]float64, historicalData []MarketData, marketRegime string) bool {
	// 条件1：明确熊市市场环境
	if marketRegime == "strong_bear" || marketRegime == "weak_bear" {
		return true
	}

	// 条件2：多指标熊市确认
	bearishSignals := 0
	totalSignals := 0

	// 趋势方向检查
	if trend, exists := state["trend_20"]; exists && trend < -0.01 {
		bearishSignals++
	}
	totalSignals++

	// RSI超卖检查
	if rsi, exists := state["rsi_14"]; exists && rsi < 40 {
		bearishSignals++
	}
	totalSignals++

	// 动量负值检查
	if momentum, exists := state["momentum_10"]; exists && momentum < -0.02 {
		bearishSignals++
	}
	totalSignals++

	// MACD死叉检查
	if macd, exists := state["macd"]; exists {
		if signal, exists := state["macd_signal"]; exists && macd < signal {
			bearishSignals++
		}
		totalSignals++
	}

	// 熊市信号占比超过50%
	if totalSignals > 0 && float64(bearishSignals)/float64(totalSignals) > 0.5 {
		return true
	}

	// 条件3：连续下跌周期检查（如果有足够的历史数据）
	if len(historicalData) >= 10 {
		consecutiveDeclines := 0
		for i := len(historicalData) - 1; i >= len(historicalData)-10 && i >= 1; i-- {
			if historicalData[i].Price < historicalData[i-1].Price {
				consecutiveDeclines++
			} else {
				break
			}
		}

		// 连续下跌超过5天，认为是熊市
		if consecutiveDeclines >= 5 {
			return true
		}
	}

	return false
}

// handleBearMarketDecision 处理熊市决策 - 严格限制买入
func (be *BacktestEngine) handleBearMarketDecision(state map[string]float64, agent map[string]interface{}, marketRegime string) (string, float64) {
	position := agent["has_position"].(bool)
	symbol := ""
	if sym, ok := agent["symbol"].(string); ok {
		symbol = sym
	}

	// === 熊市决策策略优化 ===

	// 强熊市环境：完全禁止买入，避免在极端熊市中交易
	if marketRegime == "strong_bear" && !position {
		log.Printf("[BEAR_MARKET_DECISION] %s强熊市环境，完全停止买入，等待市场转好", symbol)
		return "hold", 0.9 // 90%置信度持有，等待机会
	}

	// 弱熊市环境：大幅提高买入阈值，需要非常强的反弹信号
	if marketRegime == "weak_bear" && !position {
		log.Printf("[BEAR_MARKET_DECISION] %s弱熊市环境，大幅提高买入要求", symbol)
		// 继续评估反弹信号，但使用更严格的标准
	}

	// 1. 如果有持仓：考虑减仓或持有（不轻易买入）
	if position {
		// 检查是否应该卖出（止损或获利了结）
		if be.shouldExitInBearMarket(state, agent) {
			log.Printf("[BEAR_MARKET_DECISION] %s熊市环境，决定卖出持仓", symbol)
			return "sell", 0.7 // 70%置信度卖出
		} else {
			// 熊市中倾向于持有，避免频繁交易
			log.Printf("[BEAR_MARKET_DECISION] %s熊市环境，持有现有仓位", symbol)
			return "hold", 0.6 // 60%置信度持有
		}
	}

	// 2. 如果无持仓：熊市中大幅降低买入概率
	// 只有在极度超卖且有反弹信号时才考虑买入

	// 检查反弹信号强度
	reversalStrength := be.calculateBearMarketReversalStrength(state)

	// 根据市场环境调整反弹信号阈值
	var strongThreshold, mediumThreshold, weakThreshold float64
	if marketRegime == "weak_bear" {
		// 弱熊市：使用更严格的阈值
		strongThreshold = 0.6 // 需要更强的反弹信号
		mediumThreshold = 0.4 // 中等反弹信号阈值提高
		weakThreshold = 0.2   // 弱反弹信号阈值提高
	} else {
		// 其他熊市环境：使用标准阈值
		strongThreshold = 0.5
		mediumThreshold = 0.3
		weakThreshold = 0.15
	}

	if reversalStrength > strongThreshold {
		log.Printf("[BEAR_MARKET_DECISION] %s熊市环境检测到强反弹信号(%.2f > %.2f)，谨慎买入", symbol, reversalStrength, strongThreshold)
		return "buy", 0.4 // 降低置信度，避免过度自信
	} else if reversalStrength > mediumThreshold {
		log.Printf("[BEAR_MARKET_DECISION] %s熊市环境检测到中等反弹信号(%.2f > %.2f)，非常谨慎买入", symbol, reversalStrength, mediumThreshold)
		return "buy", 0.2 // 更低的置信度
	} else if reversalStrength > weakThreshold {
		log.Printf("[BEAR_MARKET_DECISION] %s熊市环境检测到弱反弹信号(%.2f > %.2f)，极度谨慎买入", symbol, reversalStrength, weakThreshold)
		return "buy", 0.1 // 最低置信度
	} else {
		// 无足够反弹信号，观望
		log.Printf("[BEAR_MARKET_DECISION] %s熊市环境反弹信号不足(%.2f)，继续观望", symbol, reversalStrength)
		return "hold", 0.8 // 提高观望置信度
	}
}

// calculateBearMarketReversalStrength 计算熊市反弹信号强度
func (be *BacktestEngine) calculateBearMarketReversalStrength(state map[string]float64) float64 {
	strength := 0.0
	factors := 0

	// RSI超卖反弹（保守策略，避免在熊市中过度乐观）
	if rsi, exists := state["rsi_14"]; exists {
		factors++
		if rsi < 25 {
			strength += 0.4 // 极度超卖（提高阈值）
		} else if rsi < 30 {
			strength += 0.2 // 超卖
		} else if rsi < 35 {
			strength += 0.1 // 中性偏低
		}
	}

	// 动量反转（保守策略，避免假反弹信号）
	if momentum, exists := state["momentum_10"]; exists {
		factors++
		if momentum > 0.01 {
			strength += 0.3 // 明确正动量（提高阈值）
		} else if momentum > 0.005 {
			strength += 0.15 // 弱正动量
		} else if momentum > 0.002 {
			strength += 0.05 // 微弱正动量（降低权重）
		}
	}

	// MACD金叉信号（增强检测）
	if macd, exists := state["macd"]; exists {
		if signal, exists := state["macd_signal"]; exists {
			factors++
			if macd > signal {
				strength += 0.4 // 金叉信号（增强权重）
			} else if macd > signal-0.001 { // 接近金叉
				strength += 0.1 // 弱金叉信号（新增）
			}
		}
	}

	// 布林带下轨反弹（放宽阈值）
	if bbPos, exists := state["bb_position"]; exists {
		factors++
		if bbPos < -0.3 {
			strength += 0.3 // 触及下轨，可能反弹（从-0.5放宽到-0.3）
		} else if bbPos < -0.1 {
			strength += 0.15 // 接近下轨（新增）
		}
	}

	// 成交量放大（放宽阈值）
	if volRatio, exists := state["volume_ratio"]; exists {
		factors++
		if volRatio > 1.2 {
			strength += 0.3 // 成交量放大，信号增强（从1.5降到1.2）
		} else if volRatio > 1.1 {
			strength += 0.15 // 轻微放量（新增）
		}
	}

	// 价格形态信号（使用新添加的价格形态特征）
	if doubleTop, exists := state["pattern_double_top"]; exists {
		factors++
		strength += doubleTop * 0.3 // 双顶形态信号
	}

	if headShoulder, exists := state["pattern_head_shoulder"]; exists {
		factors++
		strength += headShoulder * 0.2 // 头肩形态信号
	}

	// ATR变化（波动率增加可能预示反弹）
	if atr14, exists := state["atr_14"]; exists {
		factors++
		if atr14 > 0.001 { // ATR不为0表示有波动
			strength += 0.2 // 波动率适中，有反弹机会
		}
	}

	// ROC指标（价格变化率）
	if roc5, exists := state["roc_5"]; exists {
		factors++
		if roc5 > 0.005 { // 正向ROC
			strength += 0.25
		} else if roc5 > 0 {
			strength += 0.1
		}
	}

	if factors > 0 {
		return strength / float64(factors) // 平均强度
	}

	return 0.0
}

// CalculateBearMarketReversalStrength 计算熊市反弹信号强度（导出方法用于测试）
func (be *BacktestEngine) CalculateBearMarketReversalStrength(state map[string]float64) float64 {
	return be.calculateBearMarketReversalStrength(state)
}

// shouldExitInBearMarket 判断熊市中是否应该卖出
func (be *BacktestEngine) shouldExitInBearMarket(state map[string]float64, agent map[string]interface{}) bool {
	// 熊市中放宽止损条件，避免过早卖出
	// 只有在大幅亏损或明确反转失败时才卖出

	// 检查持仓亏损幅度
	pnl := 0.0
	if currentPnL, exists := agent["current_pnl"].(float64); exists {
		pnl = currentPnL
	}

	// 熊市中允许更大亏损（-15% vs 正常-5%）
	if pnl < -0.15 {
		return true // 大幅亏损，止损
	}

	// 检查技术指标是否转为极度悲观
	if rsi, exists := state["rsi_14"]; exists && rsi < 20 {
		return true // RSI极度超卖，可能继续下跌
	}

	// 检查动量是否转为极度负值
	if momentum, exists := state["momentum_10"]; exists && momentum < -0.05 {
		return true // 动量极度负值，趋势可能加速下跌
	}

	return false // 其他情况继续持有
}

// ===== 阶段四优化：信号增强辅助函数 =====

// calculateTrendConsistency 计算趋势一致性
func (be *BacktestEngine) calculateTrendConsistency(trend5, trend10, trend20 *TrendAnalysis) float64 {
	// 检查趋势方向一致性
	directions := []string{trend5.Direction, trend10.Direction, trend20.Direction}
	consistencyCount := 0

	// 计算两两一致性
	for i := 0; i < len(directions)-1; i++ {
		for j := i + 1; j < len(directions); j++ {
			if directions[i] == directions[j] {
				consistencyCount++
			}
		}
	}

	// 最大可能的配对数是3 (5-10, 5-20, 10-20)
	maxPairs := 3
	consistency := float64(consistencyCount) / float64(maxPairs)

	// 强度一致性：检查趋势强度是否在合理范围内
	strengths := []float64{trend5.Strength, trend10.Strength, trend20.Strength}
	strengthVariance := 0.0
	avgStrength := (strengths[0] + strengths[1] + strengths[2]) / 3.0

	for _, s := range strengths {
		strengthVariance += (s - avgStrength) * (s - avgStrength)
	}
	strengthVariance /= 3.0

	// 强度一致性得分 (方差越小一致性越高)
	strengthConsistency := math.Max(0.0, 1.0-strengthVariance*2.0)

	// 综合一致性
	overallConsistency := consistency*0.7 + strengthConsistency*0.3

	return math.Max(0.1, math.Min(1.0, overallConsistency))
}

// calculateSignalConsistency 计算信号一致性
func (be *BacktestEngine) calculateSignalConsistency(prediction, trendScore, rsiScore, volatility float64) float64 {
	signals := []float64{prediction, trendScore, rsiScore}

	// 计算信号方向一致性
	positiveCount := 0
	negativeCount := 0

	for _, signal := range signals {
		if signal > 0.1 {
			positiveCount++
		} else if signal < -0.1 {
			negativeCount++
		}
	}

	// 计算一致性得分
	totalSignals := len(signals)
	maxConsistent := math.Max(float64(positiveCount), float64(negativeCount))
	consistency := maxConsistent / float64(totalSignals)

	// 波动率调整：高波动时降低一致性要求
	if volatility > 0.05 {
		consistency *= 0.8
	}

	// 强度调整：信号强度影响一致性权重
	avgStrength := (math.Abs(prediction) + math.Abs(trendScore) + math.Abs(rsiScore)) / 3.0
	strengthBonus := math.Min(1.0, avgStrength*0.5)

	finalConsistency := consistency * (0.7 + strengthBonus*0.3)

	return math.Max(0.0, math.Min(1.0, finalConsistency))
}
