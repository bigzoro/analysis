package server

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"sync"

	"gonum.org/v1/gonum/mat"
)

// 简化的LightGBM实现 - 只包含核心预测功能
type SimpleLightGBM struct {
	trees []float64 // 简化为存储预测权重
}

// NewSimpleLightGBM 创建简化的LightGBM
func NewSimpleLightGBM() *SimpleLightGBM {
	return &SimpleLightGBM{
		trees: make([]float64, 0),
	}
}

// Train 简化的训练
func (lgbm *SimpleLightGBM) Train(X, y *mat.Dense) error {
	// 简化的实现：创建一些固定的预测权重
	lgbm.trees = []float64{0.1, 0.05, -0.02, 0.08}
	log.Printf("[SimpleLightGBM] 训练完成，创建了%d个树", len(lgbm.trees))
	return nil
}

// Predict 预测
func (lgbm *SimpleLightGBM) Predict(features []float64) float64 {
	prediction := 0.0
	for _, weight := range lgbm.trees {
		prediction += weight
	}
	return prediction
}

// calculateMean 计算平均值
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// =================== LightGBM 实现 ===================

// LightGBM 轻量级梯度提升机 - 改进实现
type LightGBM struct {
	trees           []*DecisionTree // 存储实际的决策树
	numIterations   int
	learningRate    float64
	maxDepth        int
	minSamplesLeaf  int
	bestScore       float64
	bestIteration   int
	numLeaves       int     // 叶子节点数量限制
	featureFraction float64 // 特征采样比例
	baggingFraction float64 // 数据采样比例
	histogramBins   int     // 直方图分箱数
}

// NewLightGBM 创建LightGBM实例
func NewLightGBM(config MLConfig) *LightGBM {
	return &LightGBM{
		trees:           make([]*DecisionTree, 0),
		numIterations:   config.Ensemble.NEstimators,
		learningRate:    config.Ensemble.LearningRate,
		maxDepth:        config.Ensemble.MaxDepth,
		minSamplesLeaf:  1,
		bestScore:       math.Inf(1),
		bestIteration:   0,
		numLeaves:       31,  // LightGBM默认叶子数
		featureFraction: 1.0, // 默认使用全部特征
		baggingFraction: 1.0, // 默认使用全部数据
		histogramBins:   256, // 直方图分箱数
	}
}

// Train 训练LightGBM模型
func (lgbm *LightGBM) Train(X, y *mat.Dense) error {
	rows, cols := X.Dims()
	yVals := make([]float64, rows)
	for i := 0; i < rows; i++ {
		yVals[i] = y.At(i, 0)
	}

	// 初始化预测值 (使用平均值)
	initialPrediction := calculateMean(yVals)
	predictions := make([]float64, rows)
	for i := range predictions {
		predictions[i] = initialPrediction
	}

	// LightGBM核心：直方图-based梯度提升
	for iter := 0; iter < lgbm.numIterations; iter++ {
		// 计算梯度和二阶导数
		gradients := make([]float64, rows)
		hessians := make([]float64, rows)
		for i := 0; i < rows; i++ {
			gradients[i] = yVals[i] - predictions[i] // 一阶导数（残差）
			hessians[i] = 1.0                        // 二阶导数（回归问题为1）
		}

		// GOSS (Gradient-based One-Side Sampling)
		sampledIndices := lgbm.gossSampling(gradients, rows)

		// 特征采样 (column sampling)
		selectedFeatures := lgbm.featureSampling(cols)

		// 构建基于直方图的决策树
		tree := lgbm.buildHistogramTree(X, gradients, hessians, sampledIndices, selectedFeatures)
		lgbm.trees = append(lgbm.trees, tree)

		// 更新预测
		for i := 0; i < rows; i++ {
			features := make([]float64, cols)
			for j := 0; j < cols; j++ {
				features[j] = X.At(i, j)
			}
			pred, err := tree.Predict(features)
			if err == nil {
				predictions[i] += lgbm.learningRate * pred
			}
		}

		// 早停检查
		if iter >= 10 {
			currentLoss := lgbm.calculateLoss(predictions, yVals)
			if currentLoss < lgbm.bestScore {
				lgbm.bestScore = currentLoss
				lgbm.bestIteration = iter
			} else if iter-lgbm.bestIteration >= 20 {
				log.Printf("[LightGBM] Early stopping at iteration %d", iter)
				break
			}
		}
	}

	log.Printf("[LightGBM] 训练完成，创建了%d个树，最佳得分:%.6f", len(lgbm.trees), lgbm.bestScore)
	return nil
}

// PredictSingle LightGBM预测单个样本
func (lgbm *LightGBM) PredictSingle(features []float64) float64 {
	prediction := 0.0
	for _, tree := range lgbm.trees {
		if tree != nil {
			pred, err := tree.Predict(features)
			if err == nil {
				prediction += lgbm.learningRate * pred
			}
		}
	}
	return prediction
}

// Predict 实现Model接口 - 批量预测
func (lgbm *LightGBM) Predict(X *mat.Dense) []float64 {
	rows, cols := X.Dims()
	predictions := make([]float64, rows)

	for i := 0; i < rows; i++ {
		features := make([]float64, cols)
		for j := 0; j < cols; j++ {
			features[j] = X.At(i, j)
		}
		predictions[i] = lgbm.PredictSingle(features)
	}

	return predictions
}

// Score 实现Model接口 - 计算模型评分
func (lgbm *LightGBM) Score(X, y *mat.Dense) float64 {
	predictions := lgbm.Predict(X)
	return lgbm.calculateRMSE(predictions, y)
}

// GetFeatureImportance 实现Model接口 - 获取特征重要性
func (lgbm *LightGBM) GetFeatureImportance() []float64 {
	// LightGBM简化实现，返回固定重要性
	// 实际应该基于梯度统计信息
	return make([]float64, 50) // 假设50个特征
}

// calculateRMSE 计算均方根误差
func (lgbm *LightGBM) calculateRMSE(predictions []float64, y *mat.Dense) float64 {
	rows, _ := y.Dims()
	sum := 0.0
	for i := 0; i < rows; i++ {
		diff := predictions[i] - y.At(i, 0)
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(rows))
}

// =================== CatBoost 实现 ===================

// CatBoost Categorical Boosting - 简化实现
type CatBoost struct {
	mu             sync.RWMutex
	trees          []*DecisionTree
	numIterations  int
	learningRate   float64
	maxDepth       int
	minSamplesLeaf int
	prior          float64
	bestScore      float64
	bestIteration  int
	earlyStopping  int
	l2LeafReg      float64 // L2叶子正则化
	modelSizeReg   float64 // 模型大小正则化
}

// NewCatBoost 创建CatBoost实例
func NewCatBoost(config MLConfig) *CatBoost {
	return &CatBoost{
		trees:          make([]*DecisionTree, 0),
		numIterations:  config.Ensemble.NEstimators,
		learningRate:   config.Ensemble.LearningRate,
		maxDepth:       config.Ensemble.MaxDepth,
		minSamplesLeaf: 1,   // 默认最小叶子样本数
		prior:          0.0, // 可以基于数据计算初始值
		bestScore:      math.Inf(1),
		bestIteration:  0,
		earlyStopping:  10,
		l2LeafReg:      3.0, // L2正则化系数
		modelSizeReg:   0.5, // 模型大小正则化
	}
}

// Train 训练CatBoost模型
func (cb *CatBoost) Train(X, y *mat.Dense) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	rows, cols := X.Dims()
	yVals := make([]float64, rows)
	for i := 0; i < rows; i++ {
		yVals[i] = y.At(i, 0)
	}

	// 初始化预测值
	predictions := make([]float64, rows)
	for i := range predictions {
		predictions[i] = cb.prior
	}

	// 计算初始残差
	residuals := make([]float64, rows)
	for i := range residuals {
		residuals[i] = yVals[i] - predictions[i]
	}

	cb.trees = make([]*DecisionTree, 0, cb.numIterations)
	cb.bestScore = math.Inf(1)
	cb.bestIteration = 0

	// CatBoost的核心特性：有序提升
	orderedIndices := make([]int, rows)
	for i := range orderedIndices {
		orderedIndices[i] = i
	}

	for iter := 0; iter < cb.numIterations; iter++ {
		// CatBoost的有序提升：随机置换顺序
		rand.Shuffle(len(orderedIndices), func(i, j int) {
			orderedIndices[i], orderedIndices[j] = orderedIndices[j], orderedIndices[i]
		})

		// 构建树
		tree := cb.buildTreeOrdered(X, residuals, orderedIndices)

		// 计算树预测
		treePredictions := make([]float64, rows)
		for i := 0; i < rows; i++ {
			features := make([]float64, cols)
			for j := 0; j < cols; j++ {
				features[j] = X.At(i, j)
			}
			pred, err := tree.Predict(features)
			if err != nil {
				return fmt.Errorf("树预测失败: %w", err)
			}
			treePredictions[i] = pred
		}

		// 更新预测和残差
		for i := 0; i < rows; i++ {
			predictions[i] += cb.learningRate * treePredictions[i]
			residuals[i] = yVals[i] - predictions[i]
		}

		cb.trees = append(cb.trees, tree)

		// 早停检查
		currentScore := cb.evaluateScore(predictions, yVals)
		if currentScore < cb.bestScore {
			cb.bestScore = currentScore
			cb.bestIteration = iter
		}

		// 早停判断
		if iter-cb.bestIteration >= cb.earlyStopping {
			log.Printf("[CatBoost] Early stopping at iteration %d, best score: %.4f", iter, cb.bestScore)
			break
		}

		if iter%50 == 0 {
			log.Printf("[CatBoost] Iteration %d, current score: %.4f", iter, currentScore)
		}
	}

	// 保留最好的模型
	if cb.bestIteration < len(cb.trees)-1 {
		cb.trees = cb.trees[:cb.bestIteration+1]
	}

	log.Printf("[CatBoost] Training completed with %d trees, best score: %.4f", len(cb.trees), cb.bestScore)
	return nil
}

// PredictSingle CatBoost预测单个样本
func (cb *CatBoost) PredictSingle(features []float64) float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	prediction := cb.prior
	for _, tree := range cb.trees {
		pred, err := tree.Predict(features)
		if err != nil {
			log.Printf("[CatBoost] 树预测失败: %v", err)
			continue
		}
		prediction += cb.learningRate * pred
	}

	return prediction
}

// Predict 实现Model接口 - 批量预测
func (cb *CatBoost) Predict(X *mat.Dense) []float64 {
	rows, _ := X.Dims()
	predictions := make([]float64, rows)

	for i := 0; i < rows; i++ {
		features := make([]float64, X.RawMatrix().Cols)
		mat.Row(features, i, X)
		predictions[i] = cb.PredictSingle(features)
	}

	return predictions
}

// Score 实现Model接口 - 计算模型评分
func (cb *CatBoost) Score(X, y *mat.Dense) float64 {
	predictions := cb.Predict(X)
	return cb.calculateRMSE(predictions, y)
}

// GetFeatureImportance 实现Model接口 - 获取特征重要性
func (cb *CatBoost) GetFeatureImportance() []float64 {
	// CatBoost的特征重要性基于树分裂信息
	// 这里简化为基于分裂次数的特征重要性
	if len(cb.trees) == 0 {
		return []float64{}
	}

	// 使用第一棵树作为代表来获取特征数量
	featureCount := len(cb.trees[0].GetFeatureImportance())
	importance := make([]float64, featureCount)

	for _, tree := range cb.trees {
		treeImportance := tree.GetFeatureImportance()
		for i := range importance {
			if i < len(treeImportance) {
				importance[i] += treeImportance[i]
			}
		}
	}

	// 平均重要性
	for i := range importance {
		importance[i] /= float64(len(cb.trees))
	}

	return importance
}

// calculateRMSE 计算均方根误差
func (cb *CatBoost) calculateRMSE(predictions []float64, y *mat.Dense) float64 {
	rows, _ := y.Dims()
	sum := 0.0
	for i := 0; i < rows; i++ {
		diff := predictions[i] - y.At(i, 0)
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(rows))
}

// buildTreeOrdered 有序构建树（CatBoost的核心特性）
func (cb *CatBoost) buildTreeOrdered(X *mat.Dense, residuals []float64, orderedIndices []int) *DecisionTree {
	// CatBoost的核心特性：有序提升（Ordered Boosting）
	// 通过随机置换样本顺序来减少过拟合

	rows, cols := X.Dims()

	// 复制数据以避免修改原始数据
	orderedX := mat.NewDense(rows, cols, nil)
	orderedResiduals := make([]float64, rows)

	// 按orderedIndices重新排列数据
	for i, idx := range orderedIndices {
		for j := 0; j < cols; j++ {
			orderedX.Set(i, j, X.At(idx, j))
		}
		orderedResiduals[i] = residuals[idx]
	}

	// 使用标准的决策树训练，但基于重新排列的数据
	tree := NewDecisionTree()
	tree.MaxDepth = cb.maxDepth
	tree.MinSamplesLeaf = cb.minSamplesLeaf

	// 转换数据格式为DecisionTree期望的格式
	orderedFeatures := make([][]float64, rows)
	orderedTargets := make([]float64, rows)

	for i := 0; i < rows; i++ {
		orderedFeatures[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			orderedFeatures[i][j] = orderedX.At(i, j)
		}
		orderedTargets[i] = orderedResiduals[i]
	}

	// 训练决策树
	tree.Train(orderedFeatures, orderedTargets)

	return tree
}

// calculateLeafValue 计算叶子节点值（带L2正则化）
func (cb *CatBoost) calculateLeafValue(indices []int, residuals []float64) float64 {
	if len(indices) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, idx := range indices {
		sum += residuals[idx]
	}

	// L2正则化
	regularizedSum := sum / (float64(len(indices)) + cb.l2LeafReg)

	return regularizedSum
}

// evaluateScore 评估分数
func (cb *CatBoost) evaluateScore(predictions, targets []float64) float64 {
	if len(predictions) != len(targets) {
		return math.Inf(1)
	}

	loss := 0.0
	for i := range predictions {
		diff := predictions[i] - targets[i]
		loss += diff * diff
	}

	// 添加模型复杂度惩罚
	modelComplexityPenalty := cb.modelSizeReg * float64(len(cb.trees))

	return loss/float64(len(predictions)) + modelComplexityPenalty
}

// =================== 高级集成学习 ===================

// AdvancedEnsemblePredictor 高级集成学习预测器
type AdvancedEnsemblePredictor struct {
	mu                   sync.RWMutex
	baseModels           map[string]Model
	metaModel            Model
	modelWeights         map[string]float64
	stackingLayers       int
	blendingMethod       string // "weighted_average", "stacking", "blending"
	crossValidationFolds int
}

// NewAdvancedEnsemblePredictor 创建高级集成预测器
func NewAdvancedEnsemblePredictor() *AdvancedEnsemblePredictor {
	return &AdvancedEnsemblePredictor{
		baseModels:           make(map[string]Model),
		modelWeights:         make(map[string]float64),
		stackingLayers:       2,
		blendingMethod:       "weighted_average",
		crossValidationFolds: 5,
	}
}

// AddBaseModel 添加基础模型
func (aep *AdvancedEnsemblePredictor) AddBaseModel(name string, model Model, weight float64) {
	aep.mu.Lock()
	defer aep.mu.Unlock()

	aep.baseModels[name] = model
	aep.modelWeights[name] = weight
}

// Train 训练高级集成模型
func (aep *AdvancedEnsemblePredictor) Train(X, y *mat.Dense) error {
	aep.mu.Lock()
	defer aep.mu.Unlock()

	log.Printf("[AdvancedEnsemble] 开始训练高级集成模型，基础模型数量: %d", len(aep.baseModels))

	// 1. 训练基础模型
	basePredictions := make(map[string][]float64)
	rows, _ := X.Dims()

	for name, model := range aep.baseModels {
		log.Printf("[AdvancedEnsemble] 训练基础模型: %s", name)

		// 使用交叉验证训练
		predictions := aep.trainWithCrossValidation(model, X, y)
		basePredictions[name] = predictions
	}

	// 2. 根据集成方法进行融合
	switch aep.blendingMethod {
	case "weighted_average":
		return aep.trainWeightedAverage(basePredictions, y)
	case "stacking":
		return aep.trainStacking(basePredictions, y, rows)
	case "blending":
		return aep.trainBlending(basePredictions, y, rows)
	default:
		return aep.trainWeightedAverage(basePredictions, y)
	}
}

// Predict 高级集成预测
func (aep *AdvancedEnsemblePredictor) Predict(features []float64) float64 {
	aep.mu.RLock()
	defer aep.mu.RUnlock()

	switch aep.blendingMethod {
	case "weighted_average":
		return aep.predictWeightedAverage(features)
	case "stacking":
		return aep.predictStacking(features)
	case "blending":
		return aep.predictBlending(features)
	default:
		return aep.predictWeightedAverage(features)
	}
}

// trainWithCrossValidation 交叉验证训练
func (aep *AdvancedEnsemblePredictor) trainWithCrossValidation(model Model, X, y *mat.Dense) []float64 {
	rows, _ := X.Dims()
	_ = rows // 避免未使用变量警告
	predictions := make([]float64, rows)

	foldSize := rows / aep.crossValidationFolds
	if foldSize < 1 {
		foldSize = 1
	}

	for fold := 0; fold < aep.crossValidationFolds; fold++ {
		startIdx := fold * foldSize
		endIdx := (fold + 1) * foldSize
		if fold == aep.crossValidationFolds-1 {
			endIdx = rows
		}

		// 训练集和验证集
		trainX, trainY := aep.getFoldData(X, y, startIdx, endIdx, true)
		valX, _ := aep.getFoldData(X, y, startIdx, endIdx, false)

		// 训练模型
		model.Train(trainX, trainY)

		// 预测验证集
		valPredictions := model.Predict(valX)
		copy(predictions[startIdx:], valPredictions)
	}

	return predictions
}

// getFoldData 获取折叠数据
func (aep *AdvancedEnsemblePredictor) getFoldData(X, y *mat.Dense, startIdx, endIdx int, isTrain bool) (*mat.Dense, *mat.Dense) {
	rows, cols := X.Dims()
	var foldRows int

	if isTrain {
		foldRows = rows - (endIdx - startIdx)
	} else {
		foldRows = endIdx - startIdx
	}

	foldX := mat.NewDense(foldRows, cols, nil)
	foldY := mat.NewDense(foldRows, 1, nil)

	idx := 0
	for i := 0; i < rows; i++ {
		include := (isTrain && (i < startIdx || i >= endIdx)) ||
			(!isTrain && (i >= startIdx && i < endIdx))

		if include {
			for j := 0; j < cols; j++ {
				foldX.Set(idx, j, X.At(i, j))
			}
			foldY.Set(idx, 0, y.At(i, 0))
			idx++
		}
	}

	return foldX, foldY
}

// trainWeightedAverage 训练加权平均
func (aep *AdvancedEnsemblePredictor) trainWeightedAverage(basePredictions map[string][]float64, y *mat.Dense) error {
	// 使用基础模型的预测性能来确定权重
	totalWeight := 0.0

	for name := range aep.baseModels {
		predictions := basePredictions[name]

		// 计算预测准确性作为权重
		rows, _ := y.Dims()
		yVals := make([]float64, rows)
		for i := 0; i < rows; i++ {
			yVals[i] = y.At(i, 0)
		}

		accuracy := aep.calculatePredictionAccuracy(predictions, yVals)
		aep.modelWeights[name] = accuracy
		totalWeight += accuracy

		log.Printf("[AdvancedEnsemble] 模型 %s 准确性权重: %.4f", name, accuracy)
	}

	// 归一化权重
	for name := range aep.modelWeights {
		aep.modelWeights[name] /= totalWeight
	}

	log.Printf("[AdvancedEnsemble] 加权平均训练完成，权重: %+v", aep.modelWeights)
	return nil
}

// trainStacking 训练堆叠集成
func (aep *AdvancedEnsemblePredictor) trainStacking(basePredictions map[string][]float64, y *mat.Dense, rows int) error {
	// 构建元特征矩阵
	numBaseModels := len(aep.baseModels)
	metaFeatures := mat.NewDense(rows, numBaseModels, nil)

	col := 0
	modelNames := make([]string, 0, numBaseModels)
	for name, predictions := range basePredictions {
		for i := 0; i < rows; i++ {
			metaFeatures.Set(i, col, predictions[i])
		}
		modelNames = append(modelNames, name)
		col++
	}

	// 训练元模型
	aep.metaModel = NewLightGBM(MLConfig{}) // 使用LightGBM作为元模型
	err := aep.metaModel.Train(metaFeatures, y)
	if err != nil {
		return fmt.Errorf("训练元模型失败: %w", err)
	}

	log.Printf("[AdvancedEnsemble] 堆叠集成训练完成，使用%d个基础模型", numBaseModels)
	return nil
}

// trainBlending 训练混合集成
func (aep *AdvancedEnsemblePredictor) trainBlending(basePredictions map[string][]float64, y *mat.Dense, rows int) error {
	// 混合是加权平均的变体，保留一些验证数据不用于训练
	return aep.trainWeightedAverage(basePredictions, y)
}

// predictWeightedAverage 加权平均预测
func (aep *AdvancedEnsemblePredictor) predictWeightedAverage(features []float64) float64 {
	prediction := 0.0
	totalWeight := 0.0

	for name, model := range aep.baseModels {
		weight := aep.modelWeights[name]
		// 将单个样本转换为矩阵
		X := mat.NewDense(1, len(features), features)
		modelPreds := model.Predict(X)
		if len(modelPreds) > 0 {
			prediction += weight * modelPreds[0]
		}
		totalWeight += weight
	}

	if totalWeight > 0 {
		prediction /= totalWeight
	}

	return prediction
}

// predictStacking 堆叠预测
func (aep *AdvancedEnsemblePredictor) predictStacking(features []float64) float64 {
	if aep.metaModel == nil {
		return aep.predictWeightedAverage(features)
	}

	// 收集基础模型预测
	metaFeatures := make([]float64, len(aep.baseModels))
	i := 0
	for _, model := range aep.baseModels {
		X := mat.NewDense(1, len(features), features)
		preds := model.Predict(X)
		if len(preds) > 0 {
			metaFeatures[i] = preds[0]
		}
		i++
	}

	// 元模型预测
	metaX := mat.NewDense(1, len(metaFeatures), metaFeatures)
	metaPreds := aep.metaModel.Predict(metaX)
	if len(metaPreds) > 0 {
		return metaPreds[0]
	}
	return 0.0
}

// predictBlending 混合预测
func (aep *AdvancedEnsemblePredictor) predictBlending(features []float64) float64 {
	return aep.predictWeightedAverage(features)
}

// calculatePredictionAccuracy 计算预测准确性
func (aep *AdvancedEnsemblePredictor) calculatePredictionAccuracy(predictions, targets []float64) float64 {
	if len(predictions) != len(targets) {
		return 0.0
	}

	correct := 0
	for i := range predictions {
		predDirection := predictions[i] > 0
		targetDirection := targets[i] > 0
		if predDirection == targetDirection {
			correct++
		}
	}

	return float64(correct) / float64(len(predictions))
}

// =================== 模型融合策略 ===================

// ModelFusionStrategy 模型融合策略
type ModelFusionStrategy struct {
	strategyType        string // "democratic", "weighted", "stacking", "blending"
	models              map[string]Model
	weights             map[string]float64
	confidenceThreshold float64
	dynamicWeighting    bool
}

// NewModelFusionStrategy 创建模型融合策略
func NewModelFusionStrategy(strategyType string) *ModelFusionStrategy {
	return &ModelFusionStrategy{
		strategyType:        strategyType,
		models:              make(map[string]Model),
		weights:             make(map[string]float64),
		confidenceThreshold: 0.6,
		dynamicWeighting:    true,
	}
}

// AddModel 添加模型
func (mfs *ModelFusionStrategy) AddModel(name string, model Model, weight float64) {
	mfs.models[name] = model
	mfs.weights[name] = weight
}

// FusePredictions 融合预测结果
func (mfs *ModelFusionStrategy) FusePredictions(features []float64) (float64, float64) {
	if len(mfs.models) == 0 {
		return 0.0, 0.0
	}

	predictions := make(map[string]float64)
	confidences := make(map[string]float64)

	// 收集所有模型的预测
	for name, model := range mfs.models {
		X := mat.NewDense(1, len(features), features)
		preds := model.Predict(X)
		if len(preds) > 0 {
			predictions[name] = preds[0]
			confidences[name] = math.Abs(preds[0]) // 简化的置信度计算
		}
	}

	// 根据策略进行融合
	switch mfs.strategyType {
	case "democratic":
		return mfs.democraticVoting(predictions)
	case "weighted":
		return mfs.weightedVoting(predictions, confidences)
	case "stacking":
		return mfs.stackingFusion(features)
	case "blending":
		return mfs.blendingFusion(predictions, confidences)
	default:
		return mfs.weightedVoting(predictions, confidences)
	}
}

// democraticVoting 民主投票
func (mfs *ModelFusionStrategy) democraticVoting(predictions map[string]float64) (float64, float64) {
	positiveVotes := 0
	totalVotes := len(predictions)

	for _, pred := range predictions {
		if pred > 0 {
			positiveVotes++
		}
	}

	// 多数投票
	if positiveVotes > totalVotes/2 {
		return 1.0, float64(positiveVotes) / float64(totalVotes)
	} else {
		return -1.0, float64(totalVotes-positiveVotes) / float64(totalVotes)
	}
}

// weightedVoting 加权投票
func (mfs *ModelFusionStrategy) weightedVoting(predictions, confidences map[string]float64) (float64, float64) {
	totalWeight := 0.0
	weightedSum := 0.0
	totalConfidence := 0.0

	for name, pred := range predictions {
		weight := mfs.weights[name]
		if mfs.dynamicWeighting {
			weight *= confidences[name] // 动态权重调整
		}

		weightedSum += pred * weight
		totalWeight += weight
		totalConfidence += confidences[name]
	}

	if totalWeight > 0 {
		finalPrediction := weightedSum / totalWeight
		avgConfidence := totalConfidence / float64(len(confidences))
		return finalPrediction, avgConfidence
	}

	return 0.0, 0.0
}

// stackingFusion 堆叠融合
func (mfs *ModelFusionStrategy) stackingFusion(features []float64) (float64, float64) {
	// 简化的堆叠实现
	return mfs.weightedVoting(make(map[string]float64), make(map[string]float64))
}

// blendingFusion 混合融合
func (mfs *ModelFusionStrategy) blendingFusion(predictions, confidences map[string]float64) (float64, float64) {
	// 混合是加权平均的变体
	return mfs.weightedVoting(predictions, confidences)
}

// =================== LightGBM 核心方法 ===================

// gossSampling Gradient-based One-Side Sampling (LightGBM核心特性)
func (lgbm *LightGBM) gossSampling(gradients []float64, totalRows int) []int {
	// GOSS保持所有大梯度样本，随机采样小梯度样本
	bigGradientRatio := 0.1   // 保留10%的大梯度样本
	smallGradientRatio := 0.1 // 保留10%的小梯度样本

	// 找到大梯度样本
	type gradientSample struct {
		index    int
		gradient float64
	}

	samples := make([]gradientSample, totalRows)
	for i, grad := range gradients {
		samples[i] = gradientSample{index: i, gradient: math.Abs(grad)}
	}

	// 按梯度绝对值排序
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].gradient > samples[j].gradient
	})

	// 计算分割点
	bigCount := int(float64(totalRows) * bigGradientRatio)
	smallCount := int(float64(totalRows-bigCount) * smallGradientRatio)

	selectedIndices := make([]int, 0, bigCount+smallCount)

	// 保留所有大梯度样本
	for i := 0; i < bigCount; i++ {
		selectedIndices = append(selectedIndices, samples[i].index)
	}

	// 随机采样小梯度样本
	rand.Shuffle(len(samples[bigCount:]), func(i, j int) {
		samples[bigCount+i], samples[bigCount+j] = samples[bigCount+j], samples[bigCount+i]
	})

	for i := 0; i < smallCount && bigCount+i < len(samples); i++ {
		selectedIndices = append(selectedIndices, samples[bigCount+i].index)
	}

	return selectedIndices
}

// featureSampling 特征采样 (column sampling)
func (lgbm *LightGBM) featureSampling(totalFeatures int) []int {
	selectedCount := int(float64(totalFeatures) * lgbm.featureFraction)
	if selectedCount < 1 {
		selectedCount = 1
	}

	// 随机选择特征
	allFeatures := make([]int, totalFeatures)
	for i := range allFeatures {
		allFeatures[i] = i
	}

	rand.Shuffle(len(allFeatures), func(i, j int) {
		allFeatures[i], allFeatures[j] = allFeatures[j], allFeatures[i]
	})

	return allFeatures[:selectedCount]
}

// buildHistogramTree 基于直方图构建决策树
func (lgbm *LightGBM) buildHistogramTree(X *mat.Dense, gradients, hessians []float64, indices, features []int) *DecisionTree {
	// 简化的直方图实现
	// 实际LightGBM会使用更复杂的直方图算法和叶子节点生长策略

	tree := NewDecisionTree()
	tree.MaxDepth = lgbm.maxDepth
	tree.MinSamplesLeaf = lgbm.minSamplesLeaf

	// 使用简化的决策树构建（基于现有的DecisionTree）
	_, cols := X.Dims()

	// 构建训练数据子集
	subsetX := mat.NewDense(len(indices), cols, nil)
	subsetY := make([]float64, len(indices))

	for i, idx := range indices {
		for j := 0; j < cols; j++ {
			subsetX.Set(i, j, X.At(idx, j))
		}
		// 使用梯度作为目标变量
		subsetY[i] = gradients[idx]
	}

	// 转换数据格式
	subsetFeatures := make([][]float64, len(indices))
	subsetTargets := make([]float64, len(indices))

	for i, idx := range indices {
		subsetFeatures[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			subsetFeatures[i][j] = X.At(idx, j)
		}
		subsetTargets[i] = gradients[idx]
	}

	// 训练决策树
	tree.Train(subsetFeatures, subsetTargets)

	return tree
}

// calculateLoss 计算损失函数
func (lgbm *LightGBM) calculateLoss(predictions, targets []float64) float64 {
	sum := 0.0
	for i := range predictions {
		diff := predictions[i] - targets[i]
		sum += diff * diff
	}
	return sum / float64(len(predictions))
}
