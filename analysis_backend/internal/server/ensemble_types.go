package server

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/mat"
)

// EnsembleType 集成学习类型
type EnsembleType string

const (
	EnsembleTypeBagging  EnsembleType = "bagging"  // Bagging (Bootstrap Aggregating)
	EnsembleTypeBoosting EnsembleType = "boosting" // Boosting (AdaBoost, Gradient Boosting)
	EnsembleTypeStacking EnsembleType = "stacking" // Stacking
	EnsembleTypeBlending EnsembleType = "blending" // Blending
)

// BaseLearner 基础学习器接口
type BaseLearner interface {
	Train(features [][]float64, targets []float64) error
	Predict(features []float64) (float64, error)
	GetName() string
	Clone() BaseLearner              // 用于创建学习器副本
	GetFeatureImportance() []float64 // 获取特征重要性
}

// EnsemblePredictor 集成学习预测器
type EnsemblePredictor struct {
	Type         EnsembleType
	BaseLearners []BaseLearner
	Weights      []float64   // 每个学习器的权重
	MetaLearner  BaseLearner // Stacking中的元学习器

	// 训练参数
	NumLearners    int     // 学习器数量
	LearningRate   float64 // 学习率（用于Boosting）
	SubsampleRatio float64 // 子采样比例（用于Bagging）

	// 性能监控
	TrainingTime  time.Duration
	InferenceTime time.Duration
	Accuracy      float64
}

// NewEnsemblePredictor 创建集成学习预测器
func NewEnsemblePredictor(ensembleType EnsembleType, numLearners int) *EnsemblePredictor {
	return &EnsemblePredictor{
		Type:           ensembleType,
		BaseLearners:   make([]BaseLearner, 0, numLearners),
		Weights:        make([]float64, 0, numLearners),
		NumLearners:    numLearners,
		LearningRate:   0.1,
		SubsampleRatio: 1.0,
	}
}

// Predict 进行集成预测
func (ep *EnsemblePredictor) Predict(X *mat.Dense) []float64 {
	if len(ep.BaseLearners) == 0 {
		return nil
	}

	switch ep.Type {
	case EnsembleTypeBagging:
		return ep.predictBagging(X)
	case EnsembleTypeBoosting:
		return ep.predictBoosting(X)
	case EnsembleTypeStacking:
		return ep.predictStacking(X)
	default:
		return ep.predictBagging(X)
	}
}

// predictBagging Bagging预测
func (ep *EnsemblePredictor) predictBagging(X *mat.Dense) []float64 {
	nSamples, _ := X.Dims()
	predictions := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		samplePredictions := make([]float64, len(ep.BaseLearners))

		// 获取样本特征
		sample := make([]float64, X.RawMatrix().Cols)
		for j := range sample {
			sample[j] = X.At(i, j)
		}

		// 收集所有基础学习器的预测
		for j, learner := range ep.BaseLearners {
			pred, err := learner.Predict(sample)
			if err != nil {
				continue
			}
			samplePredictions[j] = pred
		}

		// 加权平均
		weightedSum := 0.0
		totalWeight := 0.0
		for j, pred := range samplePredictions {
			weight := 1.0
			if j < len(ep.Weights) {
				weight = ep.Weights[j]
			}
			weightedSum += pred * weight
			totalWeight += weight
		}

		if totalWeight > 0 {
			predictions[i] = weightedSum / totalWeight
		} else {
			predictions[i] = samplePredictions[0] // 默认使用第一个预测
		}
	}

	return predictions
}

// predictBoosting Boosting预测
func (ep *EnsemblePredictor) predictBoosting(X *mat.Dense) []float64 {
	nSamples, _ := X.Dims()
	predictions := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		sample := make([]float64, X.RawMatrix().Cols)
		for j := range sample {
			sample[j] = X.At(i, j)
		}

		prediction := 0.0
		for j, learner := range ep.BaseLearners {
			pred, err := learner.Predict(sample)
			if err != nil {
				continue
			}
			weight := 1.0
			if j < len(ep.Weights) {
				weight = ep.Weights[j]
			}
			prediction += pred * weight
		}
		predictions[i] = prediction
	}

	return predictions
}

// predictStacking Stacking预测
func (ep *EnsemblePredictor) predictStacking(X *mat.Dense) []float64 {
	nSamples, nFeatures := X.Dims()

	// 第一层预测
	metaFeatures := make([][]float64, nSamples)
	for i := 0; i < nSamples; i++ {
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		sampleMetaFeatures := make([]float64, len(ep.BaseLearners))
		for j, learner := range ep.BaseLearners {
			pred, err := learner.Predict(sample)
			if err != nil {
				continue
			}
			sampleMetaFeatures[j] = pred
		}
		metaFeatures[i] = sampleMetaFeatures
	}

	// 元学习器预测
	if ep.MetaLearner == nil {
		return ep.predictBagging(X) // 回退到Bagging
	}

	predictions := make([]float64, nSamples)
	for i, metaFeature := range metaFeatures {
		pred, err := ep.MetaLearner.Predict(metaFeature)
		if err != nil {
			predictions[i] = 0.0
		} else {
			predictions[i] = pred
		}
	}

	return predictions
}

// AddBaseLearner 添加基础学习器
func (ep *EnsemblePredictor) AddBaseLearner(learner BaseLearner) {
	ep.BaseLearners = append(ep.BaseLearners, learner)
	ep.Weights = append(ep.Weights, 1.0) // 默认权重为1.0
}

// SetMetaLearner 设置元学习器（用于Stacking）
func (ep *EnsemblePredictor) SetMetaLearner(learner BaseLearner) {
	ep.MetaLearner = learner
}

// Train 训练集成模型
func (ep *EnsemblePredictor) Train(features [][]float64, targets []float64) error {
	if len(ep.BaseLearners) == 0 {
		return fmt.Errorf("no base learners configured")
	}

	switch ep.Type {
	case EnsembleTypeBagging:
		return ep.trainBagging(features, targets)
	case EnsembleTypeBoosting:
		return ep.trainBoosting(features, targets)
	case EnsembleTypeStacking:
		return ep.trainStacking(features, targets)
	default:
		return ep.trainBagging(features, targets)
	}
}

// trainBagging 训练Bagging集成
func (ep *EnsemblePredictor) trainBagging(features [][]float64, targets []float64) error {
	nSamples := len(features)

	for _, learner := range ep.BaseLearners {
		// Bootstrap采样
		sampleFeatures, sampleTargets := ep.bootstrapSample(features, targets, nSamples)

		// 训练学习器
		err := learner.Train(sampleFeatures, sampleTargets)
		if err != nil {
			return fmt.Errorf("failed to train base learner: %w", err)
		}
	}

	return nil
}

// trainBoosting 训练Boosting集成（AdaBoost算法）
func (ep *EnsemblePredictor) trainBoosting(features [][]float64, targets []float64) error {
	nSamples := len(features)

	// 初始化样本权重
	sampleWeights := make([]float64, nSamples)
	for i := range sampleWeights {
		sampleWeights[i] = 1.0 / float64(nSamples)
	}

	// 初始化学习器权重
	learnerWeights := make([]float64, 0, len(ep.BaseLearners))

	for learnerIdx, learner := range ep.BaseLearners {
		// 使用加权样本训练学习器
		weightedFeatures, weightedTargets := ep.resampleWithWeights(features, targets, sampleWeights)

		// 训练学习器
		err := learner.Train(weightedFeatures, weightedTargets)
		if err != nil {
			return fmt.Errorf("failed to train base learner %d: %w", learnerIdx, err)
		}

		// 计算学习器在训练集上的加权错误率
		weightedError := ep.calculateWeightedError(learner, features, targets, sampleWeights)
		if weightedError >= 0.5 {
			// 如果错误率过高，降低学习率或跳过这个学习器
			ep.LearningRate *= 0.5
			continue
		}

		// 计算学习器权重
		learnerWeight := ep.LearningRate * math.Log((1-weightedError)/weightedError)
		learnerWeights = append(learnerWeights, learnerWeight)

		// 更新样本权重
		totalWeight := 0.0
		for i := range sampleWeights {
			// 计算预测误差
			prediction, err := learner.Predict(features[i])
			if err != nil {
				continue
			}

			// 计算预测是否正确（简化为回归问题的误差）
			error := math.Abs(prediction - targets[i])
			correct := error < (ep.calculateMedianAbsoluteError(targets) * 0.1) // 误差小于中位数绝对误差的10%

			if correct {
				sampleWeights[i] *= math.Exp(-learnerWeight)
			} else {
				sampleWeights[i] *= math.Exp(learnerWeight)
			}

			totalWeight += sampleWeights[i]
		}

		// 归一化样本权重
		if totalWeight > 0 {
			for i := range sampleWeights {
				sampleWeights[i] /= totalWeight
			}
		}
	}

	// 设置学习器权重
	ep.Weights = learnerWeights

	return nil
}

// resampleWithWeights 根据权重重采样数据
func (ep *EnsemblePredictor) resampleWithWeights(features [][]float64, targets []float64, weights []float64) ([][]float64, []float64) {
	nSamples := len(features)
	resampledFeatures := make([][]float64, nSamples)
	resampledTargets := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		// 根据权重随机选择样本
		r := rand.Float64()
		cumulativeWeight := 0.0

		for j := 0; j < nSamples; j++ {
			cumulativeWeight += weights[j]
			if r <= cumulativeWeight {
				resampledFeatures[i] = make([]float64, len(features[j]))
				copy(resampledFeatures[i], features[j])
				resampledTargets[i] = targets[j]
				break
			}
		}
	}

	return resampledFeatures, resampledTargets
}

// calculateWeightedError 计算加权错误率
func (ep *EnsemblePredictor) calculateWeightedError(learner BaseLearner, features [][]float64, targets []float64, weights []float64) float64 {
	totalWeightedError := 0.0
	totalWeight := 0.0

	for i := range features {
		prediction, err := learner.Predict(features[i])
		if err != nil {
			continue
		}

		// 计算预测误差
		error := math.Abs(prediction - targets[i])
		medianError := ep.calculateMedianAbsoluteError(targets)

		// 判断预测是否正确
		isCorrect := error < (medianError * 0.1) // 误差小于中位数绝对误差的10%

		if !isCorrect {
			totalWeightedError += weights[i]
		}
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return 0.5 // 默认中等错误率
	}

	return totalWeightedError / totalWeight
}

// calculateMedianAbsoluteError 计算中位数绝对误差
func (ep *EnsemblePredictor) calculateMedianAbsoluteError(targets []float64) float64 {
	if len(targets) == 0 {
		return 1.0
	}

	// 计算所有目标值的平均值
	mean := 0.0
	for _, target := range targets {
		mean += target
	}
	mean /= float64(len(targets))

	// 计算绝对误差
	errors := make([]float64, len(targets))
	for i, target := range targets {
		errors[i] = math.Abs(target - mean)
	}

	// 排序并返回中位数
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i] > errors[j] {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	mid := len(errors) / 2
	if len(errors)%2 == 0 {
		return (errors[mid-1] + errors[mid]) / 2
	}
	return errors[mid]
}

// trainStacking 训练Stacking集成
func (ep *EnsemblePredictor) trainStacking(features [][]float64, targets []float64) error {
	// 第一层：训练基础学习器
	for _, learner := range ep.BaseLearners {
		err := learner.Train(features, targets)
		if err != nil {
			return fmt.Errorf("failed to train base learner: %w", err)
		}
	}

	// 第二层：训练元学习器（如果配置了）
	if ep.MetaLearner != nil {
		// 生成元特征
		metaFeatures := make([][]float64, len(features))
		for i := range metaFeatures {
			metaFeatures[i] = make([]float64, len(ep.BaseLearners))
			for j, learner := range ep.BaseLearners {
				pred, err := learner.Predict(features[i])
				if err != nil {
					return fmt.Errorf("failed to get prediction from base learner: %w", err)
				}
				metaFeatures[i][j] = pred
			}
		}

		err := ep.MetaLearner.Train(metaFeatures, targets)
		if err != nil {
			return fmt.Errorf("failed to train meta learner: %w", err)
		}
	}

	return nil
}

// bootstrapSample 自助采样
func (ep *EnsemblePredictor) bootstrapSample(features [][]float64, targets []float64, sampleSize int) ([][]float64, []float64) {
	sampledFeatures := make([][]float64, sampleSize)
	sampledTargets := make([]float64, sampleSize)

	for i := 0; i < sampleSize; i++ {
		idx := rand.Intn(len(features))
		sampledFeatures[i] = make([]float64, len(features[idx]))
		copy(sampledFeatures[i], features[idx])
		sampledTargets[i] = targets[idx]
	}

	return sampledFeatures, sampledTargets
}
