package server

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// FeatureSelector 特征选择器
type FeatureSelector struct {
	config MLConfig
}

// Select 执行特征选择
func (fs *FeatureSelector) Select(trainingData *TrainingData) ([]string, error) {
	switch fs.config.FeatureSelection.Method {
	case "recursive":
		return fs.recursiveFeatureElimination(trainingData)
	case "lasso":
		return fs.lassoFeatureSelection(trainingData)
	case "mutual_info":
		return fs.mutualInformationSelection(trainingData)
	case "correlation":
		return fs.correlationBasedSelection(trainingData)
	default:
		return fs.recursiveFeatureElimination(trainingData)
	}
}

// recursiveFeatureElimination 递归特征消除
func (fs *FeatureSelector) recursiveFeatureElimination(trainingData *TrainingData) ([]string, error) {
	X := trainingData.X
	y := trainingData.Y
	featureNames := trainingData.Features

	// 初始特征集
	selectedIndices := make([]int, len(featureNames))
	for i := range selectedIndices {
		selectedIndices[i] = i
	}

	// 递归消除特征
	for len(selectedIndices) > fs.config.FeatureSelection.MaxFeatures {
		// 训练模型并获取特征重要性
		importance := fs.trainAndGetImportance(X, y, selectedIndices)

		// 找到最不重要的特征
		minImportance := math.MaxFloat64
		worstFeatureIdx := -1

		for i, idx := range selectedIndices {
			if importance[i] < minImportance {
				minImportance = importance[i]
				worstFeatureIdx = idx
			}
		}

		// 如果重要性太低，停止消除
		if minImportance >= fs.config.FeatureSelection.MinImportance {
			break
		}

		// 移除最不重要的特征
		newIndices := make([]int, 0, len(selectedIndices)-1)
		for _, idx := range selectedIndices {
			if idx != worstFeatureIdx {
				newIndices = append(newIndices, idx)
			}
		}
		selectedIndices = newIndices
	}

	// 返回选中的特征名称
	selectedFeatures := make([]string, len(selectedIndices))
	for i, idx := range selectedIndices {
		selectedFeatures[i] = featureNames[idx]
	}

	return selectedFeatures, nil
}

// lassoFeatureSelection LASSO特征选择
func (fs *FeatureSelector) lassoFeatureSelection(trainingData *TrainingData) ([]string, error) {
	X := trainingData.X
	y := trainingData.Y
	featureNames := trainingData.Features

	// 简化的LASSO实现
	// 实际应该使用专业的LASSO回归库

	// 计算每个特征与目标变量的相关性
	correlations := make([]float64, len(featureNames))
	for i := range featureNames {
		featureCol := mat.Col(nil, i, X)
		correlation := stat.Correlation(featureCol, y, nil)
		correlations[i] = math.Abs(correlation)
	}

	// 按相关性排序并选择前N个特征
	type featureScore struct {
		index int
		score float64
	}

	scores := make([]featureScore, len(featureNames))
	for i, corr := range correlations {
		scores[i] = featureScore{index: i, score: corr}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 选择相关性最高的特征
	maxFeatures := fs.config.FeatureSelection.MaxFeatures
	if maxFeatures > len(scores) {
		maxFeatures = len(scores)
	}

	selectedFeatures := make([]string, maxFeatures)
	for i := 0; i < maxFeatures; i++ {
		selectedFeatures[i] = featureNames[scores[i].index]
	}

	return selectedFeatures, nil
}

// mutualInformationSelection 互信息特征选择
func (fs *FeatureSelector) mutualInformationSelection(trainingData *TrainingData) ([]string, error) {
	X := trainingData.X
	y := trainingData.Y
	featureNames := trainingData.Features

	// 计算互信息
	miScores := make([]float64, len(featureNames))
	for i := range featureNames {
		featureCol := mat.Col(nil, i, X)
		miScores[i] = fs.calculateMutualInformation(featureCol, y)
	}

	// 按互信息排序并选择
	type featureScore struct {
		index int
		score float64
	}

	scores := make([]featureScore, len(featureNames))
	for i, mi := range miScores {
		scores[i] = featureScore{index: i, score: mi}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	maxFeatures := fs.config.FeatureSelection.MaxFeatures
	if maxFeatures > len(scores) {
		maxFeatures = len(scores)
	}

	selectedFeatures := make([]string, maxFeatures)
	for i := 0; i < maxFeatures; i++ {
		selectedFeatures[i] = featureNames[scores[i].index]
	}

	return selectedFeatures, nil
}

// correlationBasedSelection 基于相关性的特征选择
func (fs *FeatureSelector) correlationBasedSelection(trainingData *TrainingData) ([]string, error) {
	X := trainingData.X
	y := trainingData.Y
	featureNames := trainingData.Features

	// 计算特征间的相关性矩阵
	nFeatures := len(featureNames)
	corrMatrix := mat.NewDense(nFeatures, nFeatures, nil)

	for i := 0; i < nFeatures; i++ {
		featureI := mat.Col(nil, i, X)
		for j := i; j < nFeatures; j++ {
			featureJ := mat.Col(nil, j, X)
			corr := stat.Correlation(featureI, featureJ, nil)
			corrMatrix.Set(i, j, corr)
			corrMatrix.Set(j, i, corr)
		}
	}

	// 计算每个特征与目标变量的相关性
	targetCorrelations := make([]float64, nFeatures)
	for i := 0; i < nFeatures; i++ {
		featureCol := mat.Col(nil, i, X)
		corr := stat.Correlation(featureCol, y, nil)
		targetCorrelations[i] = math.Abs(corr)
	}

	// 使用贪心算法选择特征，避免高度相关的特征
	selected := make([]int, 0)
	remaining := make([]int, nFeatures)
	for i := range remaining {
		remaining[i] = i
	}

	for len(selected) < fs.config.FeatureSelection.MaxFeatures && len(remaining) > 0 {
		// 找到与目标变量相关性最高且与其他选中特征相关性最低的特征
		bestScore := -1.0
		bestIdx := -1

		for _, idx := range remaining {
			score := targetCorrelations[idx]

			// 惩罚与已选特征高度相关的特征
			for _, selectedIdx := range selected {
				corr := math.Abs(corrMatrix.At(idx, selectedIdx))
				if corr > 0.8 { // 相关性阈值
					score *= (1 - corr)
				}
			}

			if score > bestScore {
				bestScore = score
				bestIdx = idx
			}
		}

		if bestIdx == -1 || bestScore < fs.config.FeatureSelection.MinImportance {
			break
		}

		selected = append(selected, bestIdx)
		// 从剩余列表中移除
		for i, idx := range remaining {
			if idx == bestIdx {
				remaining = append(remaining[:i], remaining[i+1:]...)
				break
			}
		}
	}

	// 返回选中的特征名称
	selectedFeatures := make([]string, len(selected))
	for i, idx := range selected {
		selectedFeatures[i] = featureNames[idx]
	}

	return selectedFeatures, nil
}

// trainAndGetImportance 训练模型并获取特征重要性
func (fs *FeatureSelector) trainAndGetImportance(X *mat.Dense, y []float64, featureIndices []int) []float64 {
	// 创建子集特征矩阵
	nSamples, _ := X.Dims()
	nSelected := len(featureIndices)
	XSubset := mat.NewDense(nSamples, nSelected, nil)

	for i := 0; i < nSamples; i++ {
		for j, featureIdx := range featureIndices {
			XSubset.Set(i, j, X.At(i, featureIdx))
		}
	}

	// 使用随机森林估算特征重要性
	// 这里简化为使用相关性作为重要性度量
	importance := make([]float64, nSelected)
	for i, featureIdx := range featureIndices {
		featureCol := mat.Col(nil, featureIdx, X)
		corr := stat.Correlation(featureCol, y, nil)
		importance[i] = math.Abs(corr)
	}

	return importance
}

// calculateMutualInformation 计算互信息
func (fs *FeatureSelector) calculateMutualInformation(x, y []float64) float64 {
	// 简化的互信息计算
	// 实际应该使用更精确的算法

	// 使用归一化互相关作为近似
	corr := stat.Correlation(x, y, nil)
	mi := -0.5 * math.Log(1-math.Pow(corr, 2)) // 高斯互信息近似

	return math.Max(0, mi)
}

// GetFeatureScores 获取特征评分
func (fs *FeatureSelector) GetFeatureScores(trainingData *TrainingData) ([]FeatureScore, error) {
	X := trainingData.X
	y := trainingData.Y
	featureNames := trainingData.Features

	scores := make([]FeatureScore, len(featureNames))

	for i, featureName := range featureNames {
		featureCol := mat.Col(nil, i, X)

		// 计算多种评分指标
		correlation := stat.Correlation(featureCol, y, nil)
		variance := stat.Variance(featureCol, nil)
		mi := fs.calculateMutualInformation(featureCol, y)

		// 综合评分
		compositeScore := math.Abs(correlation)*0.4 + mi*0.4 + math.Sqrt(variance)*0.2

		scores[i] = FeatureScore{
			FeatureName: featureName,
			Correlation: correlation,
			Variance:    variance,
			MutualInfo:  mi,
			Composite:   compositeScore,
		}
	}

	// 按综合评分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Composite > scores[j].Composite
	})

	return scores, nil
}

// FeatureScore 特征评分
type FeatureScore struct {
	FeatureName string
	Correlation float64
	Variance    float64
	MutualInfo  float64
	Composite   float64
}

// CrossValidateFeatureSelection 交叉验证特征选择
func (fs *FeatureSelector) CrossValidateFeatureSelection(trainingData *TrainingData) ([]string, error) {
	folds := fs.config.FeatureSelection.CrossValidationFolds
	if folds <= 1 {
		return fs.Select(trainingData)
	}

	nSamples, _ := trainingData.X.Dims()
	foldSize := nSamples / folds

	// 存储每次交叉验证选中的特征
	selectedFeaturesCount := make(map[string]int)

	for fold := 0; fold < folds; fold++ {
		// 创建训练集和验证集
		trainX, trainY := fs.createFoldData(trainingData, fold*foldSize, (fold+1)*foldSize)

		trainData := &TrainingData{
			X:        trainX,
			Y:        trainY,
			Features: trainingData.Features,
		}

		// 在训练集上进行特征选择
		foldFeatures, err := fs.Select(trainData)
		if err != nil {
			continue
		}

		// 统计特征被选中的次数
		for _, feature := range foldFeatures {
			selectedFeaturesCount[feature]++
		}
	}

	// 选择在大多数折中都被选中的特征
	stableFeatures := make([]string, 0)
	minVotes := folds * 3 / 4 // 至少在75%的折中被选中

	for feature, count := range selectedFeaturesCount {
		if count >= minVotes {
			stableFeatures = append(stableFeatures, feature)
		}
	}

	// 如果没有足够稳定的特征，放宽条件
	if len(stableFeatures) < 10 {
		minVotes = folds / 2
		stableFeatures = make([]string, 0)
		for feature, count := range selectedFeaturesCount {
			if count >= minVotes {
				stableFeatures = append(stableFeatures, feature)
			}
		}
	}

	// 限制最大特征数
	maxFeatures := fs.config.FeatureSelection.MaxFeatures
	if len(stableFeatures) > maxFeatures {
		stableFeatures = stableFeatures[:maxFeatures]
	}

	return stableFeatures, nil
}

// createFoldData 创建交叉验证的折数据
func (fs *FeatureSelector) createFoldData(trainingData *TrainingData, startIdx, endIdx int) (*mat.Dense, []float64) {
	nSamples, nFeatures := trainingData.X.Dims()

	// 创建训练集（排除验证集部分）
	trainSize := nSamples - (endIdx - startIdx)
	trainX := mat.NewDense(trainSize, nFeatures, nil)
	trainY := make([]float64, trainSize)

	trainIdx := 0
	for i := 0; i < nSamples; i++ {
		if i < startIdx || i >= endIdx {
			for j := 0; j < nFeatures; j++ {
				trainX.Set(trainIdx, j, trainingData.X.At(i, j))
			}
			trainY[trainIdx] = trainingData.Y[i]
			trainIdx++
		}
	}

	return trainX, trainY
}

// GetOptimalFeatureCount 获取最优特征数量
func (fs *FeatureSelector) GetOptimalFeatureCount(trainingData *TrainingData, maxFeatures int) (int, error) {
	if maxFeatures <= 5 {
		return maxFeatures, nil
	}

	// 使用学习曲线方法找到最优特征数
	bestScore := 0.0
	optimalCount := 10 // 默认值

	featureCounts := []int{5, 10, 15, 20, 30, 50}
	if maxFeatures < 50 {
		featureCounts = []int{5, 10, 15, 20}
		if maxFeatures < 20 {
			featureCounts = []int{5, 10, maxFeatures}
		}
	}

	for _, count := range featureCounts {
		if count > maxFeatures {
			continue
		}

		// 设置临时特征数限制
		originalMax := fs.config.FeatureSelection.MaxFeatures
		fs.config.FeatureSelection.MaxFeatures = count

		// 进行特征选择和评估
		selectedFeatures, err := fs.Select(trainingData)
		if err != nil {
			continue
		}

		// 评估特征子集的性能
		score := fs.evaluateFeatureSubset(trainingData, selectedFeatures)
		if score > bestScore {
			bestScore = score
			optimalCount = count
		}

		// 恢复原始设置
		fs.config.FeatureSelection.MaxFeatures = originalMax
	}

	return optimalCount, nil
}

// evaluateFeatureSubset 评估特征子集性能
func (fs *FeatureSelector) evaluateFeatureSubset(trainingData *TrainingData, selectedFeatures []string) float64 {
	// 创建特征子集
	featureIndices := make([]int, len(selectedFeatures))
	featureMap := make(map[string]int)
	for i, name := range trainingData.Features {
		featureMap[name] = i
	}

	for i, name := range selectedFeatures {
		if idx, exists := featureMap[name]; exists {
			featureIndices[i] = idx
		}
	}

	// 计算选中特征与目标变量的平均相关性
	totalCorrelation := 0.0
	validFeatures := 0

	for _, featureName := range selectedFeatures {
		if idx, exists := featureMap[featureName]; exists {
			featureCol := mat.Col(nil, idx, trainingData.X)
			corr := stat.Correlation(featureCol, trainingData.Y, nil)
			totalCorrelation += math.Abs(corr)
			validFeatures++
		}
	}

	if validFeatures == 0 {
		return 0.0
	}

	return totalCorrelation / float64(validFeatures)
}
