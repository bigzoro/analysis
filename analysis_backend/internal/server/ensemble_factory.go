package server

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// NeuralNetworkLearner 神经网络学习器包装器，实现 BaseLearner 接口
type NeuralNetworkLearner struct {
	Network *NeuralNetwork
}

// Train 实现 BaseLearner 接口
func (nnl *NeuralNetworkLearner) Train(features [][]float64, targets []float64) error {
	// 转换为矩阵格式
	nSamples := len(features)
	if nSamples == 0 {
		return fmt.Errorf("no training data")
	}
	nFeatures := len(features[0])

	X := mat.NewDense(nSamples, nFeatures, nil)
	for i := 0; i < nSamples; i++ {
		for j := 0; j < nFeatures; j++ {
			X.Set(i, j, features[i][j])
		}
	}

	y := mat.NewDense(nSamples, 1, nil)
	for i := 0; i < nSamples; i++ {
		y.Set(i, 0, targets[i])
	}

	return nnl.Network.Train(X, y)
}

// Predict 实现 BaseLearner 接口
func (nnl *NeuralNetworkLearner) Predict(features []float64) (float64, error) {
	// 转换为矩阵格式
	X := mat.NewDense(1, len(features), features)
	predictions := nnl.Network.Predict(X)

	if len(predictions) == 0 {
		return 0, fmt.Errorf("no prediction returned")
	}

	return predictions[0], nil
}

// GetName 实现 BaseLearner 接口
func (nnl *NeuralNetworkLearner) GetName() string {
	return "NeuralNetwork"
}

// Clone 实现 BaseLearner 接口
func (nnl *NeuralNetworkLearner) Clone() BaseLearner {
	return &NeuralNetworkLearner{Network: NewNeuralNetwork(21, []int{64, 32})}
}

// GetFeatureImportance 实现 BaseLearner 接口
func (nnl *NeuralNetworkLearner) GetFeatureImportance() []float64 {
	// 神经网络目前不支持特征重要性计算，返回空切片
	// TODO: 实现基于权重或梯度的特征重要性计算
	return []float64{}
}

// LearnerType 学习器类型
type LearnerType string

const (
	LearnerTypeLinearRegression LearnerType = "linear_regression"
	LearnerTypeDecisionTree     LearnerType = "decision_tree"
	LearnerTypeNeuralNetwork    LearnerType = "neural_network"
)

// LearnerFactory 学习器工厂
type LearnerFactory struct{}

// NewLearnerFactory 创建学习器工厂
func NewLearnerFactory() *LearnerFactory {
	return &LearnerFactory{}
}

// CreateLearner 根据类型创建学习器
func (lf *LearnerFactory) CreateLearner(learnerType LearnerType) (BaseLearner, error) {
	switch learnerType {
	case LearnerTypeLinearRegression:
		return NewLinearRegression(), nil
	case LearnerTypeDecisionTree:
		return NewDecisionTree(), nil
	case LearnerTypeNeuralNetwork:
		return &NeuralNetworkLearner{Network: NewNeuralNetwork(21, []int{64, 32})}, nil
	default:
		return nil, fmt.Errorf("unsupported learner type: %s", learnerType)
	}
}

// CreateEnsemblePredictor 创建预配置的集成预测器
func (lf *LearnerFactory) CreateEnsemblePredictor(ensembleType EnsembleType, config EnsembleConfig) (*EnsemblePredictor, error) {
	predictor := NewEnsemblePredictor(ensembleType, config.NumLearners)
	predictor.LearningRate = config.LearningRate
	predictor.SubsampleRatio = config.SubsampleRatio

	// 添加基础学习器
	for _, learnerType := range config.BaseLearners {
		learner, err := lf.CreateLearner(learnerType)
		if err != nil {
			return nil, fmt.Errorf("failed to create learner %s: %w", learnerType, err)
		}
		predictor.AddBaseLearner(learner)
	}

	// 设置元学习器（用于Stacking）
	if ensembleType == EnsembleTypeStacking && config.MetaLearner != "" {
		metaLearner, err := lf.CreateLearner(config.MetaLearner)
		if err != nil {
			return nil, fmt.Errorf("failed to create meta learner %s: %w", config.MetaLearner, err)
		}
		predictor.SetMetaLearner(metaLearner)
	}

	return predictor, nil
}

// EnsembleConfig 集成配置
type EnsembleConfig struct {
	EnsembleType   EnsembleType  `json:"ensemble_type"`
	NumLearners    int           `json:"num_learners"`
	BaseLearners   []LearnerType `json:"base_learners"`
	MetaLearner    LearnerType   `json:"meta_learner,omitempty"`
	LearningRate   float64       `json:"learning_rate"`
	SubsampleRatio float64       `json:"subsample_ratio"`
}

// DefaultConfigs 默认配置
var DefaultConfigs = map[string]EnsembleConfig{
	"bagging_basic": {
		EnsembleType:   EnsembleTypeBagging,
		NumLearners:    10,
		BaseLearners:   []LearnerType{LearnerTypeDecisionTree},
		LearningRate:   0.1,
		SubsampleRatio: 0.8,
	},
	"boosting_basic": {
		EnsembleType:   EnsembleTypeBoosting,
		NumLearners:    10,
		BaseLearners:   []LearnerType{LearnerTypeDecisionTree},
		LearningRate:   0.1,
		SubsampleRatio: 1.0,
	},
	"stacking_advanced": {
		EnsembleType:   EnsembleTypeStacking,
		NumLearners:    5,
		BaseLearners:   []LearnerType{LearnerTypeLinearRegression, LearnerTypeDecisionTree, LearnerTypeNeuralNetwork},
		MetaLearner:    LearnerTypeLinearRegression,
		LearningRate:   0.1,
		SubsampleRatio: 1.0,
	},
	"bagging_mixed": {
		EnsembleType:   EnsembleTypeBagging,
		NumLearners:    15,
		BaseLearners:   []LearnerType{LearnerTypeLinearRegression, LearnerTypeDecisionTree, LearnerTypeNeuralNetwork},
		LearningRate:   0.1,
		SubsampleRatio: 0.7,
	},
}

// CreateDefaultPredictor 创建默认集成预测器
func (lf *LearnerFactory) CreateDefaultPredictor(configName string) (*EnsemblePredictor, error) {
	config, exists := DefaultConfigs[configName]
	if !exists {
		return nil, fmt.Errorf("unknown config: %s", configName)
	}

	return lf.CreateEnsemblePredictor(config.EnsembleType, config)
}

// GetAvailableConfigs 获取可用配置
func (lf *LearnerFactory) GetAvailableConfigs() []string {
	configs := make([]string, 0, len(DefaultConfigs))
	for name := range DefaultConfigs {
		configs = append(configs, name)
	}
	return configs
}

// ValidateConfig 验证配置
func (lf *LearnerFactory) ValidateConfig(config EnsembleConfig) error {
	if config.NumLearners <= 0 {
		return fmt.Errorf("num_learners must be positive")
	}

	if len(config.BaseLearners) == 0 {
		return fmt.Errorf("base_learners cannot be empty")
	}

	if config.EnsembleType == EnsembleTypeStacking && config.MetaLearner == "" {
		return fmt.Errorf("meta_learner is required for stacking")
	}

	if config.LearningRate <= 0 || config.LearningRate > 1 {
		return fmt.Errorf("learning_rate must be between 0 and 1")
	}

	if config.SubsampleRatio <= 0 || config.SubsampleRatio > 1 {
		return fmt.Errorf("subsample_ratio must be between 0 and 1")
	}

	return nil
}
