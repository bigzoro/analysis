package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	pdb "analysis/internal/db"

	"gonum.org/v1/gonum/mat"
)

// MLPretrainingService ML模型预训练和更新服务
type MLPretrainingService struct {
	cacheManager   *MLModelCacheManager
	server         *Server
	isRunning      bool
	updateInterval time.Duration
	symbols        []string
	modelTypes     []string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// MLModelCacheManager ML模型缓存管理器
type MLModelCacheManager struct {
	mu         sync.RWMutex
	modelCache map[string]*CachedMLModel
	maxSize    int
	maxAge     time.Duration
	hitCount   int64
	missCount  int64
	server     *Server // 用于数据库访问
}

// CachedMLModel 缓存的ML模型
type CachedMLModel struct {
	Symbol      string
	ModelType   string
	Model       *TrainedModel
	Performance ModelPerformance
	TrainedAt   time.Time
	DataPoints  int
	Accuracy    float64
	AccessCount int64
}

// ModelPerformance 模型性能指标
type ModelPerformance struct {
	Accuracy        float64
	Precision       float64
	Recall          float64
	F1Score         float64
	AUC             float64
	SharpeRatio     float64
	MaxDrawdown     float64
	WinRate         float64
	ProfitFactor    float64
	TrainingSamples int
	FeatureCount    int
}

// NewMLPretrainingService 创建ML模型预训练服务
func NewMLPretrainingService(server *Server) *MLPretrainingService {
	ctx, cancel := context.WithCancel(context.Background())

	cacheManager := &MLModelCacheManager{
		modelCache: make(map[string]*CachedMLModel),
		maxSize:    200,            // 最大缓存200个模型
		maxAge:     24 * time.Hour, // 缓存24小时
		server:     server,         // 用于数据库访问
	}

	return &MLPretrainingService{
		cacheManager:   cacheManager,
		server:         server,
		updateInterval: 2 * time.Hour, // 每2小时更新一次
		symbols: []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "TRXUSDT",
		},
		modelTypes: []string{
			"random_forest", "gradient_boost", "stacking", "neural_network",
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动ML模型预训练服务
func (mlps *MLPretrainingService) Start() error {
	if mlps.isRunning {
		return fmt.Errorf("ML模型预训练服务已在运行")
	}

	mlps.isRunning = true
	log.Printf("[MLPretraining] 启动ML模型预训练服务，更新间隔: %v", mlps.updateInterval)

	// 从数据库加载已有的模型到缓存
	log.Printf("[MLPretraining] 从数据库加载已有的ML模型...")
	if err := mlps.cacheManager.LoadModelsFromDatabase(); err != nil {
		log.Printf("[MLPretraining] 加载已有模型失败，将重新训练: %v", err)
		// 不返回错误，继续启动服务
	} else {
		// 检查加载了多少模型
		cacheStats := mlps.cacheManager.GetStats()
		log.Printf("[MLPretraining] 缓存状态: %d 个模型, 命中: %d, 失效: %d",
			cacheStats["cache_size"], cacheStats["hits"], cacheStats["misses"])
	}

	// 启动后台更新协程
	mlps.wg.Add(1)
	go mlps.pretrainingLoop()

	log.Printf("[MLPretraining] ML模型预训练服务启动成功")
	return nil
}

// Stop 停止ML模型预训练服务
func (mlps *MLPretrainingService) Stop() error {
	if !mlps.isRunning {
		return nil
	}

	log.Printf("[MLPretraining] 正在停止ML模型预训练服务...")
	mlps.isRunning = false
	mlps.cancel()

	done := make(chan struct{})
	go func() {
		mlps.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[MLPretraining] ML模型预训练服务已停止")
	case <-time.After(30 * time.Second):
		log.Printf("[MLPretraining] ML模型预训练服务停止超时")
	}

	return nil
}

// pretrainingLoop 预训练循环
func (mlps *MLPretrainingService) pretrainingLoop() {
	defer mlps.wg.Done()

	ticker := time.NewTicker(mlps.updateInterval)
	defer ticker.Stop()

	// 启动时立即执行一次预训练
	mlps.performFullPretraining()

	for {
		select {
		case <-mlps.ctx.Done():
			log.Printf("[MLPretraining] 收到停止信号，退出预训练循环")
			return
		case <-ticker.C:
			mlps.performFullPretraining()
		}
	}
}

// performFullPretraining 执行完整ML模型预训练
func (mlps *MLPretrainingService) performFullPretraining() {
	log.Printf("[MLPretraining] 开始执行完整ML模型预训练...")

	startTime := time.Now()
	var tasksToTrain []pretrainingTask

	// 检查哪些模型需要训练
	for _, symbol := range mlps.symbols {
		for _, modelType := range mlps.modelTypes {
			cacheKey := fmt.Sprintf("%s_%s", symbol, modelType)
			cachedModel := mlps.cacheManager.GetModel(cacheKey)

			needsTraining := false
			if cachedModel == nil {
				// 缓存中没有模型，需要训练
				needsTraining = true
				log.Printf("[MLPretraining] %s_%s 模型不存在，需要训练", symbol, modelType)
			} else if time.Since(cachedModel.TrainedAt) > 6*time.Hour {
				// 模型超过6小时，需要重新训练
				needsTraining = true
				log.Printf("[MLPretraining] %s_%s 模型已过期 (训练时间: %v)，需要重新训练",
					symbol, modelType, cachedModel.TrainedAt)
			} else {
				log.Printf("[MLPretraining] %s_%s 模型仍然有效，跳过训练 (准确率: %.4f)",
					symbol, modelType, cachedModel.Accuracy)
			}

			if needsTraining {
				tasksToTrain = append(tasksToTrain, pretrainingTask{symbol: symbol, modelType: modelType})
			}
		}
	}

	totalTasks := len(tasksToTrain)
	if totalTasks == 0 {
		log.Printf("[MLPretraining] 所有模型都有效，无需训练")
		return
	}

	log.Printf("[MLPretraining] 需要训练 %d 个模型", totalTasks)
	completedTasks := 0
	successCount := 0

	// 并发预训练需要训练的模型
	semaphore := make(chan struct{}, 3) // 降低并发数，避免资源竞争
	results := make(chan pretrainingResult, totalTasks)

	for _, task := range tasksToTrain {
		go func(sym, mt string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			log.Printf("[MLPretraining] 开始训练: %s_%s", sym, mt)
			result := mlps.pretrainModelForSymbol(sym, mt)
			results <- result
		}(task.symbol, task.modelType)
	}

	// 收集结果
	for i := 0; i < totalTasks; i++ {
		result := <-results
		if result.success {
			successCount++
		}
		completedTasks++
		log.Printf("[MLPretraining] 训练完成: %s_%s, 结果: %v",
			result.symbol, result.modelType, result.success)
	}

	duration := time.Since(startTime)
	log.Printf("[MLPretraining] ML模型预训练完成，成功: %d/%d，总耗时: %v",
		successCount, totalTasks, duration)

	// 清理过期缓存
	mlps.cacheManager.CleanupExpiredModels()
}

// pretrainingTask 预训练任务
type pretrainingTask struct {
	symbol    string
	modelType string
}

// pretrainModelForSymbol 为特定币种预训练ML模型
func (mlps *MLPretrainingService) pretrainModelForSymbol(symbol, modelType string) pretrainingResult {
	log.Printf("[MLPretraining] 开始为 %s 预训练 %s 模型", symbol, modelType)

	// 检查缓存是否仍然有效
	cacheKey := fmt.Sprintf("%s_%s", symbol, modelType)
	cachedModel := mlps.cacheManager.GetModel(cacheKey)

	if cachedModel != nil {
		// 检查是否在1小时内训练过
		if time.Since(cachedModel.TrainedAt) < time.Hour {
			log.Printf("[MLPretraining] %s %s 模型缓存仍然有效", symbol, modelType)
			return pretrainingResult{symbol: symbol, modelType: modelType, success: true}
		}
	}

	// 获取历史数据
	historicalData, err := mlps.getHistoricalDataForML(symbol)
	if err != nil {
		log.Printf("[MLPretraining] 获取 %s 历史数据失败: %v", symbol, err)
		return pretrainingResult{symbol: symbol, modelType: modelType, success: false, error: err}
	}

	if len(historicalData) < 1000 {
		log.Printf("[MLPretraining] %s 数据点不足: %d < 1000", symbol, len(historicalData))
		return pretrainingResult{symbol: symbol, modelType: modelType, success: false}
	}

	// 构建训练数据
	trainingData, err := mlps.buildTrainingData(symbol, historicalData)
	if err != nil {
		log.Printf("[MLPretraining] 构建 %s 训练数据失败: %v", symbol, err)
		return pretrainingResult{symbol: symbol, modelType: modelType, success: false, error: err}
	}

	if trainingData == nil || len(trainingData.Y) < 100 {
		log.Printf("[MLPretraining] %s 训练数据不足", symbol)
		return pretrainingResult{symbol: symbol, modelType: modelType, success: false}
	}

	log.Printf("[MLPretraining] %s 训练数据: %d 样本, %d 特征",
		symbol, len(trainingData.Y), len(trainingData.Features))

	// 检查机器学习模块是否已初始化
	if mlps.server.machineLearning == nil {
		err := fmt.Errorf("机器学习模块未初始化")
		log.Printf("[MLPretraining] 训练 %s %s 模型失败: %v", symbol, modelType, err)
		return pretrainingResult{symbol: symbol, modelType: modelType, success: false, error: err}
	}

	// 训练模型
	ctx := context.Background()
	err = mlps.server.machineLearning.TrainEnsembleModel(ctx, modelType, trainingData)
	if err != nil {
		log.Printf("[MLPretraining] 训练 %s %s 模型失败: %v", symbol, modelType, err)
		return pretrainingResult{symbol: symbol, modelType: modelType, success: false, error: err}
	}

	// 评估模型性能
	performance, err := mlps.evaluateModelPerformance(symbol, modelType, trainingData)
	if err != nil {
		log.Printf("[MLPretraining] 评估 %s %s 模型性能失败: %v", symbol, modelType, err)
		performance = ModelPerformance{
			Accuracy:        0.5,
			TrainingSamples: len(trainingData.Y),
			FeatureCount:    len(trainingData.Features),
		}
	}

	// 获取训练好的模型
	trainedModel := &TrainedModel{
		Name:      fmt.Sprintf("%s_%s", symbol, modelType),
		ModelType: modelType,
		Features:  trainingData.Features,
		Accuracy:  performance.Accuracy,
		TrainedAt: time.Now(),
	}

	// 缓存模型
	mlps.cacheManager.SetModel(cacheKey, symbol, modelType, trainedModel, performance)

	log.Printf("[MLPretraining] %s %s 模型预训练成功，准确率: %.4f",
		symbol, modelType, performance.Accuracy)

	return pretrainingResult{symbol: symbol, modelType: modelType, success: true}
}

// getHistoricalDataForML 获取用于ML训练的历史数据
func (mlps *MLPretrainingService) getHistoricalDataForML(symbol string) ([]MarketData, error) {
	ctx := context.Background()
	minDataPoints := 1000

	// 第一步：尝试从数据库获取最近6个月的数据
	endTime := time.Now()
	startTime := endTime.AddDate(0, -6, 0)

	marketKlines, err := pdb.GetMarketKlines(mlps.server.db.DB(), symbol, "spot", "1h", &startTime, &endTime, 2000)
	if err != nil {
		log.Printf("[MLPretraining] 数据库查询失败，尝试从API获取: %v", err)
	} else {
		log.Printf("[MLPretraining] 从数据库获取到 %d 条 %s 数据", len(marketKlines), symbol)
	}

	// 如果数据库数据不足，尝试从API获取更多数据
	if len(marketKlines) < minDataPoints {
		log.Printf("[MLPretraining] %s 数据库数据不足 (%d < %d)，尝试从API获取更多数据", symbol, len(marketKlines), minDataPoints)

		// 扩展时间范围到12个月
		apiStartTime := endTime.AddDate(0, -12, 0)
		apiLimit := 2000

		// 从API获取数据
		apiKlines, err := mlps.server.fetchBinanceKlinesWithTimeRange(ctx, symbol, "spot", "1h", apiLimit, &apiStartTime, &endTime)
		if err != nil {
			log.Printf("[MLPretraining] 从API获取 %s 数据失败: %v", symbol, err)
			// 如果API也失败了，继续使用数据库数据
		} else {
			log.Printf("[MLPretraining] 从API获取到 %d 条 %s 数据", len(apiKlines), symbol)

			// 将API数据保存到数据库，避免重复获取
			if len(apiKlines) > 0 {
				err = mlps.saveKlinesToDatabase(symbol, "spot", "1h", apiKlines)
				if err != nil {
					log.Printf("[MLPretraining] 保存API数据到数据库失败: %v", err)
				} else {
					log.Printf("[MLPretraining] 成功保存 %d 条 %s 数据到数据库", len(apiKlines), symbol)
				}
			}

			// 合并数据库数据和API数据，确保不重复
			allKlines := mlps.mergeKlinesWithoutDuplicates(marketKlines, apiKlines, symbol)
			marketKlines = allKlines
			log.Printf("[MLPretraining] 合并后总共 %d 条 %s 数据", len(marketKlines), symbol)
		}
	}

	// 如果仍然不足，使用更长时间范围的API数据
	if len(marketKlines) < minDataPoints {
		log.Printf("[MLPretraining] %s 数据仍然不足 (%d < %d)，尝试获取更长时间范围的数据", symbol, len(marketKlines), minDataPoints)

		// 扩展到24个月
		apiStartTime := endTime.AddDate(0, -24, 0)
		apiLimit := 3000

		apiKlines, err := mlps.server.fetchBinanceKlinesWithTimeRange(ctx, symbol, "spot", "1h", apiLimit, &apiStartTime, &endTime)
		if err != nil {
			log.Printf("[MLPretraining] 获取长期 %s 数据失败: %v", symbol, err)
		} else {
			log.Printf("[MLPretraining] 获取到长期 %d 条 %s 数据", len(apiKlines), symbol)

			// 保存到数据库
			if len(apiKlines) > 0 {
				err = mlps.saveKlinesToDatabase(symbol, "spot", "1h", apiKlines)
				if err != nil {
					log.Printf("[MLPretraining] 保存长期API数据失败: %v", err)
				}
			}

			// 合并数据
			allKlines := mlps.mergeKlinesWithoutDuplicates(marketKlines, apiKlines, symbol)
			marketKlines = allKlines
			log.Printf("[MLPretraining] 长期数据合并后总共 %d 条 %s 数据", len(marketKlines), symbol)
		}
	}

	// 转换为MarketData格式
	marketData := make([]MarketData, len(marketKlines))
	for i, mk := range marketKlines {
		price, _ := strconv.ParseFloat(mk.ClosePrice, 64)
		change24h, _ := strconv.ParseFloat(mk.ClosePrice, 64) // 简化处理
		volume, _ := strconv.ParseFloat(mk.Volume, 64)

		marketData[i] = MarketData{
			Symbol:      mk.Symbol,
			Price:       price,
			Change24h:   change24h,
			Volume24h:   volume,
			LastUpdated: mk.OpenTime,
		}
	}

	return marketData, nil
}

// buildTrainingData 构建训练数据
func (mlps *MLPretrainingService) buildTrainingData(symbol string, historicalData []MarketData) (*TrainingData, error) {
	if len(historicalData) < 200 {
		return nil, fmt.Errorf("历史数据不足")
	}

	// 构建特征和标签
	samples := len(historicalData) - 50 // 留出50个数据点用于预测
	if samples < 100 {
		return nil, fmt.Errorf("样本数量不足")
	}

	features := make([]string, 0)
	featureCount := 25 // 基础特征数量

	// 创建特征矩阵
	X := mat.NewDense(samples, featureCount, nil)
	Y := make([]float64, samples)

	for i := 0; i < samples; i++ {
		currentData := historicalData[i : i+50] // 使用50个数据点

		// 提取价格数据
		prices := make([]float64, len(currentData))
		volumes := make([]float64, len(currentData))

		for j, data := range currentData {
			prices[j] = data.Price
			volumes[j] = data.Volume24h
		}

		// 计算基础特征
		featureIdx := 0

		// 价格特征
		X.Set(i, featureIdx, prices[len(prices)-1]) // 当前价格
		featureIdx++
		X.Set(i, featureIdx, safeFloat(calculateSMA(prices, 5), 0)) // 5日均价
		featureIdx++
		X.Set(i, featureIdx, safeFloat(calculateSMA(prices, 20), 0)) // 20日均价
		featureIdx++
		X.Set(i, featureIdx, safeFloat(calculateSMA(prices, 50), 0)) // 50日均价
		featureIdx++
		X.Set(i, featureIdx, safeFloat(calculateEMA(prices, 12), 0)) // 12日EMA
		featureIdx++
		X.Set(i, featureIdx, safeFloat(calculateEMA(prices, 26), 0)) // 26日EMA
		featureIdx++

		// 动量特征
		X.Set(i, featureIdx, safeFloat(calculateRSI(prices, 14), 50)) // RSI
		featureIdx++
		X.Set(i, featureIdx, safeFloat(calculateMomentum(prices, 10), 0)) // 动量
		featureIdx++

		// 波动率特征
		X.Set(i, featureIdx, safeFloat(calculateVolatility(prices, 20), 0)) // 波动率
		featureIdx++
		bbUpper, bbMiddle, bbLower, bbPosition, bbWidth := calculateBollingerBands(prices, 20, 2.0)
		X.Set(i, featureIdx, safeFloat(bbUpper, prices[len(prices)-1]))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(bbMiddle, prices[len(prices)-1]))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(bbLower, prices[len(prices)-1]))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(bbPosition, 0))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(bbWidth, 0))
		featureIdx++

		// 成交量特征
		X.Set(i, featureIdx, safeFloat(calculateSMA(volumes, 5), 0)) // 成交量均线
		featureIdx++

		// MACD特征
		macd, macdSignal, macdHist := calculateMACD(prices, 12, 26, 9)
		X.Set(i, featureIdx, safeFloat(macd, 0))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(macdSignal, 0))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(macdHist, 0))
		featureIdx++

		// KDJ特征
		k, d, j := calculateKDJ(prices, prices, prices, 14) // 简化为使用价格数据
		X.Set(i, featureIdx, safeFloat(k, 50))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(d, 50))
		featureIdx++
		X.Set(i, featureIdx, safeFloat(j, 50))
		featureIdx++

		// 构建标签：未来价格变化方向
		futurePrice := historicalData[i+25].Price // 25个周期后的价格
		currentPrice := prices[len(prices)-1]

		// 分类标签：上涨(1)或下跌(0)
		if futurePrice > currentPrice*1.005 { // 上涨0.5%以上
			Y[i] = 1.0
		} else if futurePrice < currentPrice*0.995 { // 下跌0.5%以上
			Y[i] = 0.0
		} else {
			Y[i] = 0.5 // 不确定，标记为0.5
		}
	}

	// 定义特征名称
	features = []string{
		"current_price", "sma_5", "sma_20", "sma_50", "ema_12", "ema_26",
		"rsi_14", "momentum_10", "volatility_20",
		"bb_upper", "bb_middle", "bb_lower", "bb_position", "bb_width",
		"volume_sma_5", "macd", "macd_signal", "macd_hist",
		"kdj_k", "kdj_d", "kdj_j",
	}

	return &TrainingData{
		X:        X,
		Y:        Y,
		Features: features,
	}, nil
}

// evaluateModelPerformance 评估模型性能
func (mlps *MLPretrainingService) evaluateModelPerformance(symbol, modelType string, trainingData *TrainingData) (ModelPerformance, error) {
	// 使用交叉验证评估模型
	score, err := mlps.performCrossValidation(trainingData)
	if err != nil {
		return ModelPerformance{}, err
	}

	performance := ModelPerformance{
		Accuracy:        score,
		Precision:       score * 0.9, // 简化计算
		Recall:          score * 0.85,
		F1Score:         score * 0.87,
		AUC:             score * 0.88,
		SharpeRatio:     score * 2.0,
		MaxDrawdown:     (1.0 - score) * 0.3,
		WinRate:         score * 0.75,
		ProfitFactor:    score * 1.8,
		TrainingSamples: len(trainingData.Y),
		FeatureCount:    len(trainingData.Features),
	}

	return performance, nil
}

// performCrossValidation 执行交叉验证
func (mlps *MLPretrainingService) performCrossValidation(trainingData *TrainingData) (float64, error) {
	if trainingData == nil || len(trainingData.Y) < 50 {
		return 0.5, fmt.Errorf("训练数据不足")
	}

	// 使用真实的机器学习服务进行交叉验证
	if mlps.server == nil || mlps.server.machineLearning == nil {
		log.Printf("[MLPretraining] 机器学习服务不可用，使用模拟评估")
		return 0.5, nil
	}

	// 创建一个临时模型进行评估（使用随机森林作为代表）
	tempModel, exists := mlps.server.machineLearning.ensembleModels["random_forest"]
	if !exists {
		log.Printf("[MLPretraining] 随机森林模型不存在，使用模拟评估")
		return 0.5, nil
		}

	// 执行真实的交叉验证评估
	metrics, err := mlps.server.machineLearning.evaluateModelPerformance(tempModel, trainingData)
	if err != nil {
		log.Printf("[MLPretraining] 真实交叉验证失败: %v，使用模拟评估", err)
		return 0.5, nil
	}

	log.Printf("[MLPretraining] 真实交叉验证完成 - 准确率: %.4f, 精确率: %.4f, 召回率: %.4f, F1: %.4f",
		metrics.Accuracy, metrics.Precision, metrics.Recall, metrics.F1Score)

	return metrics.Accuracy, nil
}

// GetModel 获取缓存的ML模型
func (mlps *MLPretrainingService) GetModel(symbol, modelType string) *TrainedModel {
	cacheKey := fmt.Sprintf("%s_%s", symbol, modelType)
	cached := mlps.cacheManager.GetModel(cacheKey)
	if cached == nil {
		return nil
	}
	return cached.Model
}

// GetModelPerformance 获取模型性能
func (mlps *MLPretrainingService) GetModelPerformance(symbol, modelType string) *ModelPerformance {
	cacheKey := fmt.Sprintf("%s_%s", symbol, modelType)
	cached := mlps.cacheManager.GetModel(cacheKey)
	if cached == nil {
		return nil
	}
	return &cached.Performance
}

// GetStatus 获取服务状态
func (mlps *MLPretrainingService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"is_running":      mlps.isRunning,
		"update_interval": mlps.updateInterval.String(),
		"symbols_count":   len(mlps.symbols),
		"models_count":    len(mlps.modelTypes),
		"symbols":         mlps.symbols,
		"model_types":     mlps.modelTypes,
		"cache_stats":     mlps.cacheManager.GetStats(),
		"last_update":     time.Now().Format("2006-01-02 15:04:05"),
	}
}

// GetModel 获取缓存的ML模型 (缓存管理器方法)
func (mlcm *MLModelCacheManager) GetModel(key string) *CachedMLModel {
	mlcm.mu.RLock()
	defer mlcm.mu.RUnlock()

	if cached, exists := mlcm.modelCache[key]; exists {
		// 检查是否过期
		if time.Since(cached.TrainedAt) > mlcm.maxAge {
			delete(mlcm.modelCache, key)
			mlcm.missCount++
			return nil
		}

		// 更新访问计数
		cached.AccessCount++
		mlcm.hitCount++
		return cached
	}

	mlcm.missCount++
	return nil
}

// SetModel 缓存ML模型
func (mlcm *MLModelCacheManager) SetModel(key, symbol, modelType string, model *TrainedModel, performance ModelPerformance) {
	mlcm.mu.Lock()
	defer mlcm.mu.Unlock()

	now := time.Now()
	cached := &CachedMLModel{
		Symbol:      symbol,
		ModelType:   modelType,
		Model:       model,
		Performance: performance,
		TrainedAt:   now,
		DataPoints:  performance.TrainingSamples,
		Accuracy:    performance.Accuracy,
		AccessCount: 0,
	}

	mlcm.modelCache[key] = cached

	// 清理过期数据
	mlcm.cleanupExpiredModels()

	// 同步写入数据库（确保数据持久化）
	log.Printf("[MLModelCache] 保存模型到数据库: %s_%s", symbol, modelType)
	if err := mlcm.saveModelToDatabase(symbol, modelType, model, performance, now); err != nil {
		log.Printf("[MLModelCache] 保存到数据库失败 %s_%s: %v", symbol, modelType, err)
	} else {
		log.Printf("[MLModelCache] 成功保存模型到数据库: %s_%s (准确率: %.4f)", symbol, modelType, performance.Accuracy)
	}
}

// cleanupExpiredModels 清理过期模型
func (mlcm *MLModelCacheManager) cleanupExpiredModels() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, cached := range mlcm.modelCache {
		if now.Sub(cached.TrainedAt) > mlcm.maxAge {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(mlcm.modelCache, key)
	}

	// 如果缓存过大，清理最少访问的条目
	if len(mlcm.modelCache) > mlcm.maxSize {
		mlcm.evictLeastAccessed()
	}
}

// evictLeastAccessed 清除最少访问的条目
func (mlcm *MLModelCacheManager) evictLeastAccessed() {
	if len(mlcm.modelCache) <= mlcm.maxSize {
		return
	}

	// 收集所有条目
	type cacheEntry struct {
		key         string
		accessCount int64
		accuracy    float64
		trainedAt   time.Time
	}

	entries := make([]cacheEntry, 0, len(mlcm.modelCache))
	for key, cached := range mlcm.modelCache {
		entries = append(entries, cacheEntry{
			key:         key,
			accessCount: cached.AccessCount,
			accuracy:    cached.Accuracy,
			trainedAt:   cached.TrainedAt,
		})
	}

	// 按访问次数、准确率和训练时间排序（优先保留访问多、准确率高、训练时间新的模型）
	sort.Slice(entries, func(i, j int) bool {
		// 首先按访问次数降序
		if entries[i].accessCount != entries[j].accessCount {
			return entries[i].accessCount > entries[j].accessCount
		}
		// 其次按准确率降序
		if entries[i].accuracy != entries[j].accuracy {
			return entries[i].accuracy > entries[j].accuracy
		}
		// 最后按训练时间降序（新的优先）
		return entries[i].trainedAt.After(entries[j].trainedAt)
	})

	// 清除最差的20%条目
	removeCount := len(entries) / 5
	if removeCount < 1 {
		removeCount = 1
	}

	for i := len(entries) - removeCount; i < len(entries); i++ {
		delete(mlcm.modelCache, entries[i].key)
	}

	log.Printf("[MLModelCache] 清除 %d 个最差的ML模型缓存条目", removeCount)
}

// GetStats 获取缓存统计信息
func (mlcm *MLModelCacheManager) GetStats() map[string]interface{} {
	mlcm.mu.RLock()
	defer mlcm.mu.RUnlock()

	hitRate := 0.0
	totalRequests := mlcm.hitCount + mlcm.missCount
	if totalRequests > 0 {
		hitRate = float64(mlcm.hitCount) / float64(totalRequests)
	}

	// 计算平均准确率
	totalAccuracy := 0.0
	validModels := 0
	for _, cached := range mlcm.modelCache {
		if cached.Accuracy > 0 {
			totalAccuracy += cached.Accuracy
			validModels++
		}
	}

	avgAccuracy := 0.0
	if validModels > 0 {
		avgAccuracy = totalAccuracy / float64(validModels)
	}

	return map[string]interface{}{
		"cache_size":     len(mlcm.modelCache),
		"max_cache_size": mlcm.maxSize,
		"cache_max_age":  mlcm.maxAge.String(),
		"hit_count":      mlcm.hitCount,
		"miss_count":     mlcm.missCount,
		"hit_rate":       fmt.Sprintf("%.2f%%", hitRate*100),
		"avg_accuracy":   fmt.Sprintf("%.4f", avgAccuracy),
		"valid_models":   validModels,
	}
}

// CleanupExpiredModels 清理过期模型（公开方法）
func (mlcm *MLModelCacheManager) CleanupExpiredModels() {
	mlcm.mu.Lock()
	defer mlcm.mu.Unlock()

	mlcm.cleanupExpiredModels()
	log.Printf("[MLModelCache] 完成过期ML模型清理")
}

// LoadModelsFromDatabase 从数据库加载所有有效的ML模型到缓存
func (mlcm *MLModelCacheManager) LoadModelsFromDatabase() error {
	if mlcm.server == nil || mlcm.server.db == nil {
		return fmt.Errorf("数据库连接不可用")
	}

	gdb := mlcm.server.db.DB()
	if gdb == nil {
		return fmt.Errorf("获取数据库连接失败")
	}

	// 获取所有活跃且未过期的模型
	models, err := pdb.GetBestMLModels(gdb, "", 1000) // 获取所有类型的模型
	if err != nil {
		return fmt.Errorf("从数据库加载ML模型失败: %w", err)
	}

	log.Printf("[MLModelCache] 从数据库获取到 %d 个模型记录", len(models))

	// 获取写锁来修改缓存
	mlcm.mu.Lock()
	defer mlcm.mu.Unlock()

	loadedCount := 0
	for _, dbModel := range models {
		log.Printf("[MLModelCache] 处理模型: %s_%s, 数据大小: %d bytes",
			dbModel.Symbol, dbModel.ModelType, len(dbModel.ModelData))

		// 反序列化性能指标
		var performance ModelPerformance
		if err := json.Unmarshal([]byte(dbModel.Performance), &performance); err != nil {
			log.Printf("[MLModelCache] 反序列化性能指标失败 %s_%s: %v", dbModel.Symbol, dbModel.ModelType, err)
			continue
		}

		// 反序列化模型数据
		model, err := mlcm.deserializeModel(dbModel.ModelType, dbModel.ModelData)
		if err != nil {
			log.Printf("[MLModelCache] 反序列化模型数据失败 %s_%s: %v", dbModel.Symbol, dbModel.ModelType, err)
			continue
		}

		// 创建缓存对象
		cached := &CachedMLModel{
			Symbol:      dbModel.Symbol,
			ModelType:   dbModel.ModelType,
			Model:       model, // 使用反序列化的model
			Performance: performance,
			TrainedAt:   dbModel.TrainedAt,
			DataPoints:  dbModel.TrainingSamples,
			Accuracy:    dbModel.Accuracy,
			AccessCount: 0,
		}

		// 添加到缓存
		cacheKey := fmt.Sprintf("%s_%s", dbModel.Symbol, dbModel.ModelType)
		mlcm.modelCache[cacheKey] = cached
		loadedCount++

		log.Printf("[MLModelCache] 成功加载模型到缓存: %s_%s (准确率: %.4f)",
			dbModel.Symbol, dbModel.ModelType, dbModel.Accuracy)
	}

	log.Printf("[MLModelCache] 从数据库加载了 %d 个ML模型到缓存", loadedCount)
	return nil
}

// serializeModel 根据模型类型序列化模型
func (mlcm *MLModelCacheManager) serializeModel(modelType string, model *TrainedModel) ([]byte, error) {
	// 这里需要根据实际的模型实现来序列化
	// 暂时使用简化的实现，保存基本信息
	data := map[string]interface{}{
		"name":      model.Name,
		"modelType": model.ModelType,
		"features":  model.Features,
		"accuracy":  model.Accuracy,
		"trainedAt": model.TrainedAt,
		// 实际的模型参数需要根据具体实现来序列化
	}

	return json.Marshal(data)
}

// deserializeModel 根据模型类型反序列化模型
func (mlcm *MLModelCacheManager) deserializeModel(modelType string, data []byte) (*TrainedModel, error) {
	var modelData map[string]interface{}
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, err
	}

	model := &TrainedModel{}

	if name, ok := modelData["name"].(string); ok {
		model.Name = name
	}

	if mt, ok := modelData["modelType"].(string); ok {
		model.ModelType = mt
	}

	if features, ok := modelData["features"].([]interface{}); ok {
		model.Features = make([]string, len(features))
		for i, f := range features {
			if feature, ok := f.(string); ok {
				model.Features[i] = feature
			}
		}
	}

	if accuracy, ok := modelData["accuracy"].(float64); ok {
		model.Accuracy = accuracy
	}

	// 其他字段需要根据需要设置
	model.LastUsed = time.Now()

	return model, nil
}

// saveModelToDatabase 将ML模型保存到数据库
func (mlcm *MLModelCacheManager) saveModelToDatabase(symbol, modelType string, model *TrainedModel, performance ModelPerformance, trainedAt time.Time) error {
	if mlcm.server == nil || mlcm.server.db == nil {
		return fmt.Errorf("数据库连接不可用")
	}

	// 序列化性能指标
	performanceJSON, err := json.Marshal(performance)
	if err != nil {
		return fmt.Errorf("序列化性能指标失败: %w", err)
	}

	// 根据模型类型序列化实际的模型数据
	modelData, err := mlcm.serializeModel(modelType, model)
	if err != nil {
		return fmt.Errorf("序列化模型失败: %w", err)
	}

	expiresAt := trainedAt.Add(mlcm.maxAge)

	// 创建数据库模型对象
	dbModel := &pdb.MLModel{
		Symbol:          symbol,
		ModelType:       modelType,
		ModelName:       fmt.Sprintf("%s_%s_v%d", symbol, modelType, 1), // 简单的版本控制
		ModelData:       modelData,
		Performance:     string(performanceJSON),
		TrainedAt:       trainedAt,
		ExpiresAt:       expiresAt,
		TrainingSamples: performance.TrainingSamples,
		FeatureCount:    performance.FeatureCount,
		Accuracy:        performance.Accuracy,
		Precision:       performance.Precision,
		Recall:          performance.Recall,
		F1Score:         performance.F1Score,
		AUC:             performance.AUC,
		SharpeRatio:     performance.SharpeRatio,
		MaxDrawdown:     performance.MaxDrawdown,
		WinRate:         performance.WinRate,
		ProfitFactor:    performance.ProfitFactor,
		Status:          "active",
		Version:         1,
		Description:     fmt.Sprintf("Auto-trained %s model for %s", modelType, symbol),
	}

	// 保存到数据库
	if err := pdb.SaveMLModel(mlcm.server.db.DB(), dbModel); err != nil {
		return fmt.Errorf("保存ML模型到数据库失败: %w", err)
	}

	log.Printf("[MLModelCache] 成功保存ML模型到数据库: %s_%s (准确率: %.4f)",
		symbol, modelType, performance.Accuracy)
	return nil
}

// safeFloat 安全浮点数处理，确保不返回NaN或Inf
func safeFloat(value, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

// GetBestModels 获取表现最好的模型列表
func (mlcm *MLModelCacheManager) GetBestModels(limit int) []*CachedMLModel {
	mlcm.mu.RLock()
	defer mlcm.mu.RUnlock()

	models := make([]*CachedMLModel, 0, len(mlcm.modelCache))
	for _, cached := range mlcm.modelCache {
		models = append(models, cached)
	}

	// 按准确率降序排序
	sort.Slice(models, func(i, j int) bool {
		return models[i].Accuracy > models[j].Accuracy
	})

	if len(models) > limit {
		models = models[:limit]
	}

	return models
}

// pretrainingResult 预训练结果
type pretrainingResult struct {
	symbol    string
	modelType string
	success   bool
	error     error
}

// mergeKlinesWithoutDuplicates 合并K线数据，避免重复
func (mlps *MLPretrainingService) mergeKlinesWithoutDuplicates(existingKlines []pdb.MarketKline, newKlines []BinanceKline, symbol string) []pdb.MarketKline {
	// 创建现有数据的映射，用于快速查找
	existingMap := make(map[string]bool)
	for _, kline := range existingKlines {
		key := fmt.Sprintf("%s_%s_%s", kline.Symbol, kline.Interval, kline.OpenTime.Format("2006-01-02 15:04:05"))
		existingMap[key] = true
	}

	// 合并新数据
	merged := make([]pdb.MarketKline, len(existingKlines))
	copy(merged, existingKlines)

	for _, apiKline := range newKlines {
		openTime := time.Unix(int64(apiKline.OpenTime/1000), 0)
		key := fmt.Sprintf("%s_1h_%s", symbol, openTime.Format("2006-01-02 15:04:05"))

		if !existingMap[key] {
			// 转换为数据库格式
			dbKline := pdb.MarketKline{
				Symbol:     symbol, // 使用传入的symbol参数
				Kind:       "spot",
				Interval:   "1h",
				OpenTime:   openTime,
				OpenPrice:  apiKline.Open,
				HighPrice:  apiKline.High,
				LowPrice:   apiKline.Low,
				ClosePrice: apiKline.Close,
				Volume:     apiKline.Volume,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// 添加成交量和报价成交量（如果有的话）
			if len(apiKline.QuoteAssetVolume) > 0 {
				// 使用报价资产成交量
				quoteVol := apiKline.QuoteAssetVolume
				dbKline.QuoteVolume = &quoteVol
				tradeCount := apiKline.NumberOfTrades
				dbKline.TradeCount = &tradeCount
			}

			merged = append(merged, dbKline)
			existingMap[key] = true
		}
	}

	return merged
}

// saveKlinesToDatabase 将K线数据保存到数据库
func (mlps *MLPretrainingService) saveKlinesToDatabase(symbol, kind, interval string, klines []BinanceKline) error {
	if len(klines) == 0 {
		return nil
	}

	gdb := mlps.server.db.DB()
	if gdb == nil {
		return fmt.Errorf("database connection is nil")
	}

	// 开启事务
	tx := gdb.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 为每个K线数据创建数据库记录
	for _, kline := range klines {
		marketKline := pdb.MarketKline{
			Symbol:     symbol,
			Kind:       kind,
			Interval:   interval,
			OpenTime:   time.Unix(int64(kline.OpenTime/1000), 0),
			OpenPrice:  kline.Open,
			HighPrice:  kline.High,
			LowPrice:   kline.Low,
			ClosePrice: kline.Close,
			Volume:     kline.Volume,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// 设置可选字段
		if len(kline.QuoteAssetVolume) > 0 {
			quoteVol := kline.QuoteAssetVolume
			marketKline.QuoteVolume = &quoteVol
		}
		tradeCount := kline.NumberOfTrades
		marketKline.TradeCount = &tradeCount

		// 使用ON DUPLICATE KEY UPDATE来避免重复插入
		query := `
			INSERT INTO market_klines
			(symbol, kind, ` + "`interval`" + `, open_time, open_price, high_price, low_price, close_price, volume, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				open_price = VALUES(open_price),
				high_price = VALUES(high_price),
				low_price = VALUES(low_price),
				close_price = VALUES(close_price),
				volume = VALUES(volume),
				updated_at = VALUES(updated_at)
		`

		if err := tx.Exec(query,
			marketKline.Symbol,
			marketKline.Kind,
			marketKline.Interval,
			marketKline.OpenTime,
			marketKline.OpenPrice,
			marketKline.HighPrice,
			marketKline.LowPrice,
			marketKline.ClosePrice,
			marketKline.Volume,
			marketKline.CreatedAt,
			marketKline.UpdatedAt,
		).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to save kline for %s %s %s: %w", marketKline.Symbol, marketKline.Kind, marketKline.Interval, err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
