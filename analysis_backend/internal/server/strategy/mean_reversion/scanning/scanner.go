package scanning

import (
	"analysis/internal/server/strategy/mean_reversion"
	"analysis/internal/server/strategy/mean_reversion/config"
	"analysis/internal/server/strategy/mean_reversion/indicators"
	"analysis/internal/server/strategy/mean_reversion/risk"
	"analysis/internal/server/strategy/mean_reversion/signals"
	"analysis/internal/server/strategy/mean_reversion/validation"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	pdb "analysis/internal/db"
)

// Scanner 均值回归策略扫描器
type Scanner struct {
	configManager    mean_reversion.MRConfigManager
	indicatorFactory mean_reversion.MRIndicatorFactory
	signalProcessor  mean_reversion.MRSignalProcessor
	riskManager      mean_reversion.MRRiskManager
	validator        mean_reversion.MRValidator
	db               interface{} // 数据库连接
}

// NewScanner 创建均值回归策略扫描器
func NewScanner(db interface{}) *Scanner {
	scanner := &Scanner{
		db: db,
	}

	// 初始化各个组件
	scanner.configManager = config.NewMRConfigManager()
	scanner.indicatorFactory = indicators.NewMRIndicatorFactory()
	scanner.signalProcessor = signals.NewMRSignalProcessor()
	scanner.riskManager = risk.NewMRRiskManager()
	scanner.validator = validation.NewMRValidator(scanner)

	return scanner
}

// Scan 执行单个币种的策略扫描
func (s *Scanner) Scan(ctx context.Context, symbol string, marketData *mean_reversion.StrategyMarketData, config *mean_reversion.MeanReversionConfig) (*mean_reversion.EligibleSymbol, error) {
	// 验证输入
	if symbol == "" || marketData == nil || config == nil {
		return nil, fmt.Errorf("无效的输入参数")
	}

	// 验证配置
	if err := s.configManager.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 验证市场数据
	if err := s.validator.ValidateStrategy(config, marketData); err != nil {
		return nil, fmt.Errorf("策略验证失败: %w", err)
	}

	// 生成指标信号
	indicatorSignals := make([]*mean_reversion.IndicatorSignal, 0, len(config.Core.Indicators))

	for _, indicatorName := range config.Core.Indicators {
		indicator, err := s.indicatorFactory.Create(indicatorName, s.getIndicatorParams(config, indicatorName))
		if err != nil {
			return nil, fmt.Errorf("创建指标失败 %s: %w", indicatorName, err)
		}

		signal, err := indicator.Calculate(marketData.Prices, s.getIndicatorParams(config, indicatorName))
		if err != nil {
			return nil, fmt.Errorf("指标计算失败 %s: %w", indicatorName, err)
		}

		indicatorSignals = append(indicatorSignals, &signal)
	}

	// 处理信号强度
	signalStrength, err := s.signalProcessor.ProcessSignals(indicatorSignals, config)
	if err != nil {
		return nil, fmt.Errorf("信号处理失败: %w", err)
	}

	// 做出决策
	decision, err := s.signalProcessor.MakeDecision(signalStrength, config)
	if err != nil {
		return nil, fmt.Errorf("决策失败: %w", err)
	}

	// 如果是持有信号，返回nil
	if decision.Action == "hold" {
		return nil, nil
	}

	// 计算风险参数
	currentPrice := marketData.Prices[len(marketData.Prices)-1]
	stopLoss := s.riskManager.CalculateStopLoss(currentPrice, decision.Action, config)
	_ = s.riskManager.CalculateTakeProfit(currentPrice, stopLoss, decision.Action, config) // 暂时计算但不使用

	// 生成交易信号
	eligibleSymbol := &mean_reversion.EligibleSymbol{
		Symbol: symbol,
		Action: decision.Action,
		Reason: fmt.Sprintf("智能均值回归%s信号 (强度:%.1f%%, 置信度:%.1f%%) - %s",
			decision.Action, decision.Strength*100, decision.Confidence*100, decision.Reason),
		Multiplier: 1.0,
		MarketCap:  marketData.MarketCap,
	}

	// 验证风险限制
	positionSize := s.riskManager.CalculatePositionSize(currentPrice, config.RiskManagement.StopLoss, 10000, config) // 假设1万美元资金
	if !s.riskManager.ValidateRiskLimits(positionSize, 0, config) {
		return nil, fmt.Errorf("风险限制验证失败")
	}

	return eligibleSymbol, nil
}

// GetConfigManager 获取配置管理器
func (s *Scanner) GetConfigManager() mean_reversion.MRConfigManager {
	return s.configManager
}

// GetIndicatorFactory 获取指标工厂
func (s *Scanner) GetIndicatorFactory() mean_reversion.MRIndicatorFactory {
	return s.indicatorFactory
}

// GetSignalProcessor 获取信号处理器
func (s *Scanner) GetSignalProcessor() mean_reversion.MRSignalProcessor {
	return s.signalProcessor
}

// GetRiskManager 获取风险管理器
func (s *Scanner) GetRiskManager() mean_reversion.MRRiskManager {
	return s.riskManager
}

// GetValidator 获取验证器
func (s *Scanner) GetValidator() mean_reversion.MRValidator {
	return s.validator
}

// GetStrategyType 获取策略类型
func (s *Scanner) GetStrategyType() string {
	return "mean_reversion"
}

// IsEnabled 检查策略是否启用
func (s *Scanner) IsEnabled(config *mean_reversion.MeanReversionConfig) bool {
	return config != nil && config.Enabled
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
	log.Printf("[MeanReversion] 开始扫描策略ID: %d", tradingStrategy.ID)

	// 转换旧的StrategyConditions为新的MeanReversionConfig
	config, err := a.convertTradingStrategy(tradingStrategy)
	if err != nil {
		log.Printf("[MeanReversion] 策略ID %d 配置转换失败: %v", tradingStrategy.ID, err)
		return nil, fmt.Errorf("配置转换失败: %w", err)
	}

	// 如果策略未启用，返回空结果
	if !config.Enabled {
		log.Printf("[MeanReversion] 策略ID %d 未启用，跳过扫描", tradingStrategy.ID)
		return []interface{}{}, nil
	}

	log.Printf("[MeanReversion] 策略ID %d 配置转换成功，使用模式: %s", tradingStrategy.ID, config.Core.Mode)

	// 获取市场数据
	marketData, err := a.getMarketDataForStrategy(tradingStrategy)
	if err != nil {
		log.Printf("[MeanReversion] 策略ID %d 获取市场数据失败: %v", tradingStrategy.ID, err)
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	log.Printf("[MeanReversion] 策略ID %d 成功获取 %d 个交易对的市场数据", tradingStrategy.ID, len(marketData))

	var results []interface{}
	var successCount int
	var skipCount int

	// 扫描所有启用的交易对
	symbols := a.getSymbolsToScan(tradingStrategy)
	log.Printf("[MeanReversion] 策略ID %d 开始扫描 %d 个交易对", tradingStrategy.ID, len(symbols))

	for _, symbol := range symbols {
		// 为每个交易对准备市场数据
		symbolMarketData := a.prepareSymbolMarketData(symbol, marketData)

		// 执行策略扫描
		signal, err := a.scanner.Scan(ctx, symbol, symbolMarketData, config)
		if err != nil {
			// 记录详细错误信息但继续处理其他交易对
			log.Printf("[MeanReversion] 扫描交易对 %s 时出错: %v", symbol, err)
			log.Printf("[MeanReversion] 跳过 %s，继续处理其他交易对", symbol)
			skipCount++
			continue
		}

		if signal != nil {
			// 找到有效信号
			log.Printf("[MeanReversion] 交易对 %s 发现信号: %s (%s)", symbol, signal.Action, signal.Reason)

			// 转换信号格式为interface{}
			symbolMap := map[string]interface{}{
				"symbol":       signal.Symbol,
				"action":       signal.Action,
				"reason":       signal.Reason,
				"multiplier":   signal.Multiplier,
				"market_cap":   signal.MarketCap,
				"gainers_rank": 0,
			}
			results = append(results, symbolMap)
			successCount++
		} else {
			log.Printf("[MeanReversion] 交易对 %s 未发现有效信号", symbol)
		}
	}

	log.Printf("[MeanReversion] 策略ID %d 扫描完成: 成功%d个，跳过%d个，总共%d个交易对",
		tradingStrategy.ID, successCount, skipCount, len(symbols))

	return results, nil
}

// GetStrategyType 获取策略类型
func (a *strategyScannerAdapter) GetStrategyType() string {
	return "mean_reversion"
}

// convertTradingStrategy 转换TradingStrategy为MeanReversionConfig
func (a *strategyScannerAdapter) convertTradingStrategy(tradingStrategy *pdb.TradingStrategy) (*mean_reversion.MeanReversionConfig, error) {
	return a.scanner.GetConfigManager().ConvertToUnifiedConfig(tradingStrategy.Conditions)
}

// getMarketDataForStrategy 获取策略需要的市场数据
func (a *strategyScannerAdapter) getMarketDataForStrategy(tradingStrategy *pdb.TradingStrategy) (map[string]*mean_reversion.StrategyMarketData, error) {
	symbols := a.getSymbolsToScan(tradingStrategy)
	log.Printf("[MeanReversion] 开始获取 %d 个交易对的市场数据", len(symbols))

	marketDataMap := make(map[string]*mean_reversion.StrategyMarketData)

	// 批量获取市值数据
	marketCapMap, err := a.getMarketCapData(symbols)
	if err != nil {
		log.Printf("[MeanReversion] 获取市值数据失败: %v", err)
		// 继续执行，使用默认市值
	} else {
		log.Printf("[MeanReversion] 成功获取 %d 个交易对的市值数据", len(marketCapMap))
	}

	for _, symbol := range symbols {
		marketData, err := a.getRealMarketData(symbol, marketCapMap[symbol])
		if err != nil {
			log.Printf("[MeanReversion] 获取%s市场数据失败: %v，跳过此交易对", symbol, err)
			continue // 跳过此交易对，继续处理其他交易对
		}
		marketDataMap[symbol] = marketData
	}

	log.Printf("[MeanReversion] 成功获取 %d 个交易对的完整市场数据", len(marketDataMap))
	return marketDataMap, nil
}

// getRealMarketData 获取真实的市场数据
func (a *strategyScannerAdapter) getRealMarketData(symbol string, marketCap float64) (*mean_reversion.StrategyMarketData, error) {
	// 类型断言获取数据库连接
	db, ok := a.scanner.db.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("数据库连接类型断言失败")
	}

	// 获取K线数据（1小时级别，最近100个数据点）
	klines, err := pdb.GetMarketKlines(db, symbol, "spot", "1h", nil, nil, 100)
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 20 {
		return nil, fmt.Errorf("K线数据不足，需要至少20个数据点，当前%d个", len(klines))
	}

	// 转换为价格和成交量数组
	prices := make([]float64, len(klines))
	volumes := make([]float64, len(klines))

	for i, kline := range klines {
		if price, err := strconv.ParseFloat(kline.ClosePrice, 64); err == nil {
			prices[i] = price
		} else {
			prices[i] = 0
		}

		if volume, err := strconv.ParseFloat(kline.Volume, 64); err == nil {
			volumes[i] = volume
		} else {
			volumes[i] = 0
		}
	}

	// 计算技术指标
	volatility := a.calculateVolatility(prices)
	trendStrength := a.calculateTrendStrength(prices)
	oscillationScore := a.calculateOscillationScore(prices)

	return &mean_reversion.StrategyMarketData{
		Symbol:           symbol,
		Prices:           prices,
		Volumes:          volumes,
		MarketCap:        marketCap,
		Volatility:       volatility,
		TrendStrength:    trendStrength,
		OscillationScore: oscillationScore,
	}, nil
}

// getMarketCapData 批量获取市值数据
func (a *strategyScannerAdapter) getMarketCapData(symbols []string) (map[string]float64, error) {
	if a.scanner.db == nil {
		return make(map[string]float64), nil
	}

	// 类型断言获取数据库连接
	db, ok := a.scanner.db.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("数据库连接类型断言失败")
	}

	marketCapService := pdb.NewCoinCapMarketDataService(db)
	marketDataMap, err := marketCapService.GetMarketDataBySymbols(context.Background(), symbols)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for symbol, data := range marketDataMap {
		if data != nil {
			// 解析市值字符串为float64
			if cap, err := strconv.ParseFloat(data.MarketCapUSD, 64); err == nil {
				result[symbol] = cap
			} else {
				result[symbol] = 0
			}
		} else {
			result[symbol] = 0
		}
	}

	return result, nil
}

// calculateVolatility 计算波动率
func (a *strategyScannerAdapter) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算收益率
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// 计算标准差
	return a.calculateStdDev(returns)
}

// calculateTrendStrength 计算趋势强度
func (a *strategyScannerAdapter) calculateTrendStrength(prices []float64) float64 {
	if len(prices) < 10 {
		return 0
	}

	// 计算线性回归斜率
	n := float64(len(prices))
	sumX := n * (n - 1) / 2
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, price := range prices {
		x := float64(i)
		sumY += price
		sumXY += x * price
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 计算平均价格
	avgPrice := sumY / n

	// 趋势强度 = 斜率 / 平均价格
	if avgPrice != 0 {
		return slope / avgPrice
	}

	return 0
}

// calculateOscillationScore 计算震荡分数
func (a *strategyScannerAdapter) calculateOscillationScore(prices []float64) float64 {
	if len(prices) < 20 {
		return 0
	}

	// 计算价格的变异系数 (标准差/均值)
	mean := a.calculateMean(prices)
	if mean == 0 {
		return 0
	}

	stdDev := a.calculateStdDev(prices)
	coefficientOfVariation := stdDev / mean

	// 计算价格范围系数 ((最大值-最小值)/均值)
	minPrice := a.findMin(prices)
	maxPrice := a.findMax(prices)
	rangeCoefficient := (maxPrice - minPrice) / mean

	// 震荡分数 = (变异系数 + 范围系数) / 2
	return (coefficientOfVariation + rangeCoefficient) / 2
}

// 辅助计算函数
func (a *strategyScannerAdapter) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (a *strategyScannerAdapter) calculateStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := a.calculateMean(values)
	sumSquares := 0.0

	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func (a *strategyScannerAdapter) findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (a *strategyScannerAdapter) findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

// getSymbolsToScan 获取需要扫描的交易对列表
func (a *strategyScannerAdapter) getSymbolsToScan(tradingStrategy *pdb.TradingStrategy) []string {
	conditions := tradingStrategy.Conditions

	// 1. 检查是否有自定义的白名单
	if conditions.UseSymbolWhitelist && a.isSymbolWhitelistValid(conditions.SymbolWhitelist) {
		// 从JSON配置中解析交易对列表
		whitelistSymbols, err := a.parseSymbolWhitelist(conditions.SymbolWhitelist)
		if err != nil {
			log.Printf("[MeanReversion] 解析白名单失败: %v，使用默认列表", err)
		} else if len(whitelistSymbols) > 0 {
			log.Printf("[MeanReversion] 使用白名单模式，包含%d个交易对", len(whitelistSymbols))
			return a.filterByMarketCapAndType(whitelistSymbols, conditions)
		}
	}

	// 2. 默认模式：获取活跃的主要交易对
	symbols, err := a.getActiveMajorSymbols(conditions.TradingType)
	if err != nil {
		log.Printf("[MeanReversion] 获取活跃交易对失败: %v", err)
		return []string{} // 返回空列表，让上层处理
	}

	// 3. 根据市值限制和交易类型过滤
	filteredSymbols := a.filterByMarketCapAndType(symbols, conditions)

	// 4. 限制扫描数量（避免性能问题）
	maxSymbols := 10
	if len(filteredSymbols) > maxSymbols {
		filteredSymbols = filteredSymbols[:maxSymbols]
	}

	log.Printf("[MeanReversion] 最终扫描%d个交易对: %v", len(filteredSymbols), filteredSymbols)
	return filteredSymbols
}

// prepareSymbolMarketData 为特定交易对准备市场数据
func (a *strategyScannerAdapter) prepareSymbolMarketData(symbol string, marketDataMap map[string]*mean_reversion.StrategyMarketData) *mean_reversion.StrategyMarketData {
	if data, exists := marketDataMap[symbol]; exists {
		return data
	}

	// 返回默认的市场数据结构
	return &mean_reversion.StrategyMarketData{
		Symbol:           symbol,
		Prices:           []float64{50000.0}, // 示例价格
		Volumes:          []float64{100.0},   // 示例成交量
		MarketCap:        1000000000.0,       // 示例市值
		Volatility:       0.05,               // 示例波动率
		TrendStrength:    0.1,                // 示例趋势强度
		OscillationScore: 0.8,                // 示例震荡指数
	}
}

// ToStrategyScanner 创建适配器
func (s *Scanner) ToStrategyScanner() interface{} {
	return &strategyScannerAdapter{scanner: s, db: s.db}
}

// ============================================================================
// 注册表
// ============================================================================

var globalScanner *Scanner

// GetMeanReversionScanner 获取均值回归策略扫描器实例
func GetMeanReversionScanner(db interface{}) *Scanner {
	if globalScanner == nil {
		globalScanner = NewScanner(db)
	}
	return globalScanner
}

// getIndicatorParams 获取指标参数
func (s *Scanner) getIndicatorParams(config *mean_reversion.MeanReversionConfig, indicatorName string) map[string]interface{} {
	params := make(map[string]interface{})

	switch indicatorName {
	case "bollinger":
		params["period"] = config.Core.Period
		params["multiplier"] = config.Indicators.Bollinger.Multiplier
	case "rsi":
		params["overbought"] = config.Indicators.RSI.Overbought
		params["oversold"] = config.Indicators.RSI.Oversold
		params["period"] = 14 // RSI固定周期
	case "price_channel":
		params["period"] = config.Core.Period
	}

	return params
}

// ============================================================================
// 辅助函数
// ============================================================================

// parseSymbolWhitelist 解析JSON格式的交易对白名单
func (a *strategyScannerAdapter) parseSymbolWhitelist(whitelistJSON []byte) ([]string, error) {
	if len(whitelistJSON) == 0 {
		return nil, fmt.Errorf("白名单数据为空")
	}

	var symbols []string
	err := json.Unmarshal(whitelistJSON, &symbols)
	if err != nil {
		return nil, fmt.Errorf("解析白名单JSON失败: %w", err)
	}

	return symbols, nil
}

// isSymbolWhitelistValid 检查白名单是否有效且非空
func (a *strategyScannerAdapter) isSymbolWhitelistValid(whitelist datatypes.JSON) bool {
	// 检查数据长度不为0且不为"null"
	if len(whitelist) == 0 || string(whitelist) == "null" {
		return false
	}

	// 检查是否为空数组 []
	if string(whitelist) == "[]" {
		return false
	}

	// 尝试解析并检查是否为非空数组
	var symbols []string
	if err := json.Unmarshal(whitelist, &symbols); err != nil {
		return false
	}

	return len(symbols) > 0
}

// getActiveMajorSymbols 从数据库获取活跃的主要交易对
func (a *strategyScannerAdapter) getActiveMajorSymbols(tradingType string) ([]string, error) {
	// 类型断言获取数据库连接
	db, ok := a.scanner.db.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("数据库连接类型断言失败")
	}

	// 根据交易类型确定查询条件
	var marketType string
	switch tradingType {
	case "spot":
		marketType = "spot"
	case "futures":
		marketType = "futures"
	case "both":
		marketType = "spot" // 默认查询现货，后续过滤
	default:
		marketType = "spot"
	}

	// 从realtime_gainers_items表获取最近的活跃交易对
	query := `
		SELECT i.symbol
		FROM realtime_gainers_items i
		INNER JOIN realtime_gainers_snapshots s ON i.snapshot_id = s.id
		WHERE s.kind = ?
			AND s.id = (
				SELECT MAX(id) FROM realtime_gainers_snapshots
				WHERE kind = ?
				AND timestamp >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			)
			AND i.volume24h > 1000000  -- 24小时成交量大于100万
		ORDER BY i.volume24h DESC
		LIMIT 20
	`

	var symbols []string
	err := db.Debug().Raw(query, marketType, marketType).Scan(&symbols).Error
	if err != nil {
		return nil, fmt.Errorf("查询活跃交易对失败: %w", err)
	}

	return symbols, nil
}

// filterByMarketCapAndType 根据市值和交易类型过滤交易对
func (a *strategyScannerAdapter) filterByMarketCapAndType(symbols []string, conditions pdb.StrategyConditions) []string {
	if len(symbols) == 0 {
		return symbols
	}

	// 获取市值数据
	marketCapMap, err := a.getMarketCapData(symbols)
	if err != nil {
		log.Printf("[MeanReversion] 获取市值数据失败，使用所有交易对: %v", err)
		return symbols
	}

	var filteredSymbols []string
	for _, symbol := range symbols {
		marketCap := marketCapMap[symbol]

		// 检查市值限制（如果启用）
		if conditions.NoShortBelowMarketCap && marketCap > 0 {
			// 对于做空，检查市值是否超过限制
			if marketCap < conditions.MarketCapLimitShort {
				log.Printf("[MeanReversion] 跳过%s，市值%.2f低于限制%.2f",
					symbol, marketCap, conditions.MarketCapLimitShort)
				continue
			}
		}

		// 可以在这里添加更多的过滤逻辑
		// 例如：交易量过滤、价格范围过滤等

		filteredSymbols = append(filteredSymbols, symbol)
	}

	return filteredSymbols
}
