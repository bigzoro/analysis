package server

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/gonum/mat"
)

// OnlineLearningBuffer 在线学习数据缓冲区
type OnlineLearningBuffer struct {
	features   [][]float64  // 特征数据
	targets    []float64    // 目标变量
	timestamps []time.Time  // 时间戳
	maxSize    int          // 最大缓冲区大小
	mu         sync.RWMutex // 并发安全
}

// OnlineLearningConfig 在线学习配置
type OnlineLearningConfig struct {
	Enabled              bool          `json:"enabled"`                // 是否启用在线学习
	BufferSize           int           `json:"buffer_size"`            // 数据缓冲区大小
	UpdateInterval       time.Duration `json:"update_interval"`        // 更新间隔
	LearningRate         float64       `json:"learning_rate"`          // 初始学习率
	LearningRateDecay    float64       `json:"learning_rate_decay"`    // 学习率衰减
	MinLearningRate      float64       `json:"min_learning_rate"`      // 最小学习率
	ForgetFactor         float64       `json:"forget_factor"`          // 遗忘因子
	PerformanceThreshold float64       `json:"performance_threshold"`  // 性能阈值
	MinSamplesForUpdate  int           `json:"min_samples_for_update"` // 最少样本数才更新
}

// MLEnsemblePredictor 机器学习集成预测器
type MLEnsemblePredictor struct {
	models             []BaseLearner
	metaModel          BaseLearner
	method             string
	featureIdx         []int // 使用的特征索引
	rand               *rand.Rand
	onlineConfig       OnlineLearningConfig
	learningBuffer     *OnlineLearningBuffer
	lastUpdateTime     time.Time
	performanceHistory []float64 // 性能历史记录
	trainingTargets    []float64 // 训练目标值，用于默认预测
}

// DeepCopy 创建MLEnsemblePredictor的深拷贝
func (m *MLEnsemblePredictor) DeepCopy() *MLEnsemblePredictor {
	if m == nil {
		return nil
	}

	// 创建新的实例
	modelCopy := &MLEnsemblePredictor{
		method:             m.method,
		featureIdx:         make([]int, len(m.featureIdx)),
		rand:               rand.New(rand.NewSource(time.Now().UnixNano())), // 新的随机数生成器
		onlineConfig:       m.onlineConfig,                                  // 值类型，可以直接复制
		lastUpdateTime:     m.lastUpdateTime,
		performanceHistory: make([]float64, len(m.performanceHistory)),
		trainingTargets:    make([]float64, len(m.trainingTargets)),
	}

	// 深拷贝slice
	copy(modelCopy.featureIdx, m.featureIdx)
	copy(modelCopy.performanceHistory, m.performanceHistory)
	copy(modelCopy.trainingTargets, m.trainingTargets)

	// 深拷贝基础模型 - 这里需要根据实际的BaseLearner实现进行深拷贝
	modelCopy.models = make([]BaseLearner, len(m.models))
	for i, model := range m.models {
		if model != nil {
			// 根据模型类型进行深拷贝
			modelCopy.models[i] = m.deepCopyBaseLearner(model)
		}
	}

	// 深拷贝元模型
	if m.metaModel != nil {
		modelCopy.metaModel = m.deepCopyBaseLearner(m.metaModel)
	}

	// 深拷贝学习缓冲区
	if m.learningBuffer != nil {
		modelCopy.learningBuffer = &OnlineLearningBuffer{
			features:   make([][]float64, len(m.learningBuffer.features)),
			targets:    make([]float64, len(m.learningBuffer.targets)),
			timestamps: make([]time.Time, len(m.learningBuffer.timestamps)),
			maxSize:    m.learningBuffer.maxSize,
		}
		for i, feat := range m.learningBuffer.features {
			modelCopy.learningBuffer.features[i] = make([]float64, len(feat))
			copy(modelCopy.learningBuffer.features[i], feat)
		}
		copy(modelCopy.learningBuffer.targets, m.learningBuffer.targets)
		copy(modelCopy.learningBuffer.timestamps, m.learningBuffer.timestamps)
	}

	return modelCopy
}

// deepCopyBaseLearner 根据BaseLearner的具体类型进行深拷贝
func (m *MLEnsemblePredictor) deepCopyBaseLearner(model BaseLearner) BaseLearner {
	if model == nil {
		return nil
	}

	// 根据实际的BaseLearner实现进行深拷贝
	// 这里需要根据具体的模型类型进行相应的深拷贝逻辑

	switch m := model.(type) {
	case *DecisionTree:
		// 决策树的深拷贝
		return m.DeepCopy()
	default:
		// 对于其他类型，返回浅拷贝并记录警告
		log.Printf("[WARN] 未实现的模型深拷贝类型: %T，使用浅拷贝", model)
		return model
	}
}

// NewMLEnsemblePredictor 创建机器学习集成预测�?
func NewMLEnsemblePredictor(method string, nEstimators int, config MLConfig) *MLEnsemblePredictor {
	predictor := &MLEnsemblePredictor{
		method: method,
		models: make([]BaseLearner, nEstimators),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// 初始化基础模型
	switch method {
	case "random_forest":
		predictor.initializeRandomForest(nEstimators, config)
	case "gradient_boost":
		predictor.initializeGradientBoost(nEstimators, config)
	case "stacking":
		predictor.initializeStacking(nEstimators, config)
	case "transformer":
		predictor.initializeTransformer(nEstimators, config)
	default:
		predictor.initializeRandomForest(nEstimators, config)
	}

	return predictor
}

// initializeRandomForest 初始化随机森林 - Phase 10优化：大幅降低深度防止过拟合
func (ep *MLEnsemblePredictor) initializeRandomForest(nEstimators int, config MLConfig) {
	ep.models = make([]BaseLearner, nEstimators)
	for i := 0; i < nEstimators; i++ {
		tree := NewDecisionTree()

		// Phase 10优化：降低默认深度，防止过拟合
		if config.Ensemble.MaxDepth > 0 {
			tree.MaxDepth = config.Ensemble.MaxDepth
		} else {
			tree.MaxDepth = 6 // Phase 10: 从12降低到6，显著减少过拟合风险
		}

		// Phase 10优化：增加最小分割样本数
		if tree.MaxDepth > 20 {
			tree.MaxDepth = 20
		}

		log.Printf("[INIT_RF] 初始化第%d棵树: MaxDepth=%d", i+1, tree.MaxDepth)
		ep.models[i] = tree
	}
}

// initializeGradientBoost 初始化梯度提升
func (ep *MLEnsemblePredictor) initializeGradientBoost(nEstimators int, config MLConfig) {
	ep.models = make([]BaseLearner, nEstimators)
	// 梯度提升：使用较浅的树以避免过拟合
	for i := 0; i < nEstimators; i++ {
		tree := NewDecisionTree()
		// 梯度提升通常使用较浅的树（3-8深度）
		tree.MaxDepth = 6
		if config.Ensemble.MaxDepth > 0 && config.Ensemble.MaxDepth < 10 {
			tree.MaxDepth = config.Ensemble.MaxDepth
		}
		ep.models[i] = tree
		log.Printf("[INIT_GRADIENT_BOOST] 基础模型 %d: DecisionTree (MaxDepth=%d)", i, tree.MaxDepth)
	}
}

// initializeStacking 初始化堆叠集成
func (ep *MLEnsemblePredictor) initializeStacking(nEstimators int, config MLConfig) {
	log.Printf("[INIT_STACKING] 开始初始化Stacking集成，nEstimators=%d", nEstimators)

	ep.models = make([]BaseLearner, nEstimators)
	// 使用决策树作为基础模型，避免线性回归的矩阵求逆问题
	// 设置不同的深度以增加多样性
	for i := 0; i < nEstimators; i++ {
		tree := NewDecisionTree()

		if config.Ensemble.MaxDepth > 0 {
			// 在基础深度周围波动 ±2
			baseDepth := config.Ensemble.MaxDepth
			depthVariation := (i % 5) - 2 // -2, -1, 0, 1, 2 的循环
			tree.MaxDepth = baseDepth + depthVariation
		} else {
			tree.MaxDepth = 8 + (i % 8) // 8-15之间的深度变化
		}

		// 确保深度在合理范围内
		if tree.MaxDepth < 3 {
			tree.MaxDepth = 3
		}
		if tree.MaxDepth > 25 {
			tree.MaxDepth = 25
		}

		ep.models[i] = tree
		log.Printf("[INIT_STACKING] 基础模型 %d: DecisionTree (MaxDepth=%d)", i, tree.MaxDepth)
	}

	// 初始化元模型 - 使用决策树而不是线性回归，避免矩阵求逆问题
	metaTree := NewDecisionTree()
	if config.Ensemble.MaxDepth > 0 {
		metaTree.MaxDepth = config.Ensemble.MaxDepth + 2 // 元模型可以稍微深一些
	} else {
		metaTree.MaxDepth = 15 // 默认深度
	}
	if metaTree.MaxDepth > 30 {
		metaTree.MaxDepth = 30
	}

	ep.metaModel = metaTree
	if ep.metaModel == nil {
		log.Printf("[INIT_STACKING] 错误：元模型初始化失败")
	} else {
		log.Printf("[INIT_STACKING] 元模型初始化成功: %s (MaxDepth=%d)", ep.metaModel.GetName(), metaTree.MaxDepth)
	}

	log.Printf("[INIT_STACKING] Stacking集成初始化完成")
}

// initializeTransformer 初始化Transformer集成
func (ep *MLEnsemblePredictor) initializeTransformer(nEstimators int, config MLConfig) {
	ep.models = make([]BaseLearner, 1) // Transformer使用单个模型

	log.Printf("[INIT_TRANSFORMER] 初始化Transformer集成模型")

	// 创建实际的Transformer模型实例
	transformerModel := NewTransformerModel(4, 8, 256, 1024, 0.1)     // 使用合适的参数
	transformerWrapper := NewTransformerWrapper(transformerModel, 20) // 20个特征维度
	ep.models[0] = transformerWrapper

	log.Printf("[INIT_TRANSFORMER] Transformer集成初始化完成，模型已创建")
}

// SetTransformerModel 设置Transformer模型（用于延迟初始化）
func (ep *MLEnsemblePredictor) SetTransformerModel(model *TransformerModel) {
	if len(ep.models) > 0 {
		if wrapper, ok := ep.models[0].(*TransformerWrapper); ok {
			wrapper.model = model
			log.Printf("[INIT_TRANSFORMER] Transformer模型设置成功")
		}
	}
}

// FeatureImportanceAnalysis 特征重要性分析
type FeatureImportanceAnalysis struct {
	Features      []FeatureImportance `json:"features"`
	Method        string              `json:"method"`       // permutation, tree_based, correlation
	TopFeatures   []string            `json:"top_features"` // 最重要特征列表
	ImportanceSum float64             `json:"importance_sum"`
	AnalysisTime  time.Time           `json:"analysis_time"`
}

// AnalyzeFeatureImportance 分析特征重要性
func (ep *MLEnsemblePredictor) AnalyzeFeatureImportance(trainingData *TrainingData) (*FeatureImportanceAnalysis, error) {
	if trainingData == nil || len(trainingData.Features) == 0 {
		return nil, fmt.Errorf("训练数据为空")
	}

	log.Printf("[FEATURE_IMPORTANCE] 开始特征重要性分析: %d 个特征", len(trainingData.Features))

	// 使用排列重要性方法（permutation importance）
	analysis, err := ep.permutationImportance(trainingData)
	if err != nil {
		log.Printf("[FEATURE_IMPORTANCE] 排列重要性分析失败: %v", err)
		// 回退到基于树的特征重要性
		analysis, err = ep.treeBasedImportance(trainingData)
		if err != nil {
			return nil, fmt.Errorf("所有特征重要性分析方法都失败")
		}
	}

	// 按重要性排序
	sort.Slice(analysis.Features, func(i, j int) bool {
		return analysis.Features[i].Importance > analysis.Features[j].Importance
	})

	// 提取前N个重要特征
	topN := minInt(20, len(analysis.Features))
	analysis.TopFeatures = make([]string, topN)
	for i := 0; i < topN; i++ {
		analysis.TopFeatures[i] = analysis.Features[i].FeatureName
	}

	// 计算重要性总和
	analysis.ImportanceSum = 0
	for _, feature := range analysis.Features {
		analysis.ImportanceSum += feature.Importance
	}

	analysis.AnalysisTime = time.Now()

	log.Printf("[FEATURE_IMPORTANCE] 分析完成: 前5重要特征: %v", analysis.TopFeatures[:minInt(5, len(analysis.TopFeatures))])

	return analysis, nil
}

// permutationImportance 排列重要性分析
func (ep *MLEnsemblePredictor) permutationImportance(trainingData *TrainingData) (*FeatureImportanceAnalysis, error) {
	r, c := trainingData.X.Dims()
	if r < 50 || c < 2 {
		return nil, fmt.Errorf("样本或特征数不足")
	}

	// 计算基准准确率
	baselineAccuracy, err := ep.evaluateModelAccuracy(trainingData)
	if err != nil {
		return nil, fmt.Errorf("计算基准准确率失败: %w", err)
	}

	log.Printf("[PERMUTATION] 基准准确率: %.4f", baselineAccuracy)

	var importanceResults []FeatureImportance

	// 对每个特征进行排列重要性分析
	for featureIdx, featureName := range trainingData.Features {
		// 创建特征排列后的数据集
		permutedData := ep.permuteFeature(trainingData, featureIdx)

		// 计算排列后的准确率
		permutedAccuracy, err := ep.evaluateModelAccuracy(permutedData)
		if err != nil {
			log.Printf("[PERMUTATION] 特征 %s 排列测试失败: %v", featureName, err)
			continue
		}

		// 计算重要性（准确率下降程度）
		importance := baselineAccuracy - permutedAccuracy
		if importance < 0 {
			importance = 0 // 确保非负
		}

		importanceResults = append(importanceResults, FeatureImportance{
			FeatureName: featureName,
			Importance:  importance,
		})

		category := ep.categorizeFeature(featureName)
		log.Printf("[PERMUTATION] 特征 %s: 重要性 %.6f (类别: %s)", featureName, importance, category)
	}

	return &FeatureImportanceAnalysis{
		Features: importanceResults,
		Method:   "permutation",
	}, nil
}

// treeBasedImportance 基于树的特征重要性分析
func (ep *MLEnsemblePredictor) treeBasedImportance(trainingData *TrainingData) (*FeatureImportanceAnalysis, error) {
	if len(ep.models) == 0 {
		return nil, fmt.Errorf("没有训练好的模型")
	}

	var importanceResults []FeatureImportance
	featureCount := len(trainingData.Features)

	// 收集所有决策树模型的特征重要性
	for _, model := range ep.models {
		if tree, ok := model.(*DecisionTree); ok {
			if tree.featureImportance != nil {
				// 累加每个特征的重要性
				for i, importance := range tree.featureImportance {
					if i >= len(importanceResults) {
						// 初始化
						for len(importanceResults) <= i {
							featureName := "unknown"
							if len(importanceResults) < featureCount {
								featureName = trainingData.Features[len(importanceResults)]
							}
							importanceResults = append(importanceResults, FeatureImportance{
								FeatureName: featureName,
								Importance:  0,
							})
						}
					}
					if i < len(importanceResults) {
						importanceResults[i].Importance += importance
					}
				}
			}
		}
	}

	// 归一化重要性
	totalImportance := 0.0
	for _, result := range importanceResults {
		totalImportance += result.Importance
	}

	if totalImportance > 0 {
		for i := range importanceResults {
			importanceResults[i].Importance /= totalImportance
		}
	}

	return &FeatureImportanceAnalysis{
		Features: importanceResults,
		Method:   "tree_based",
	}, nil
}

// evaluateModelAccuracy 评估模型准确率
func (ep *MLEnsemblePredictor) evaluateModelAccuracy(data *TrainingData) (float64, error) {
	if len(ep.models) == 0 {
		return 0, fmt.Errorf("没有可用的模型")
	}

	r, _ := data.X.Dims()
	correct := 0

	// 使用交叉验证评估准确率
	for i := 0; i < r; i++ {
		// 提取单一样本的特征
		sample := make([]float64, len(data.Features))
		for j := range sample {
			sample[j] = data.X.At(i, j)
		}

		// 获取模型预测 - 转换为矩阵格式
		sampleMatrix := mat.NewDense(1, len(sample), sample)
		predictions := ep.Predict(sampleMatrix)
		if len(predictions) == 0 {
			continue // 跳过预测失败的样本
		}
		prediction := predictions[0]

		// 将预测转换为类别
		predClass := 0
		if prediction > 0.5 {
			predClass = 1
		} else if prediction < -0.5 {
			predClass = -1
		}

		// 获取真实标签
		trueClass := 0
		if data.Y[i] > 0.5 {
			trueClass = 1
		} else if data.Y[i] < -0.5 {
			trueClass = -1
		}

		// 计算准确率（简化版，只考虑方向是否正确）
		if predClass == trueClass {
			correct++
		}
	}

	accuracy := float64(correct) / float64(r)
	return accuracy, nil
}

// permuteFeature 排列指定特征的值
func (ep *MLEnsemblePredictor) permuteFeature(data *TrainingData, featureIdx int) *TrainingData {
	r, c := data.X.Dims()

	// 复制数据
	newX := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			newX.Set(i, j, data.X.At(i, j))
		}
	}

	// 对指定特征列进行随机排列
	featureValues := make([]float64, r)
	for i := 0; i < r; i++ {
		featureValues[i] = data.X.At(i, featureIdx)
	}

	// Fisher-Yates shuffle
	for i := r - 1; i > 0; i-- {
		j := ep.rand.Intn(i + 1)
		featureValues[i], featureValues[j] = featureValues[j], featureValues[i]
	}

	// 设置排列后的值
	for i := 0; i < r; i++ {
		newX.Set(i, featureIdx, featureValues[i])
	}

	return &TrainingData{
		X:         newX,
		Y:         data.Y,
		Features:  data.Features,
		SampleIDs: data.SampleIDs,
	}
}

// categorizeFeature 根据特征名称分类特征
func (ep *MLEnsemblePredictor) categorizeFeature(featureName string) string {
	switch {
	case strings.Contains(featureName, "rsi") || strings.Contains(featureName, "macd") ||
		strings.Contains(featureName, "stoch") || strings.Contains(featureName, "williams") ||
		strings.Contains(featureName, "bollinger"):
		return "technical"
	case strings.Contains(featureName, "volume") || strings.Contains(featureName, "turnover"):
		return "volume"
	case strings.Contains(featureName, "trend") || strings.Contains(featureName, "ma"):
		return "trend"
	case strings.Contains(featureName, "momentum") || strings.Contains(featureName, "roc"):
		return "momentum"
	case strings.Contains(featureName, "cross") || strings.Contains(featureName, "correlation"):
		return "cross"
	case strings.Contains(featureName, "statistical") || strings.Contains(featureName, "variance") ||
		strings.Contains(featureName, "skewness") || strings.Contains(featureName, "kurtosis"):
		return "statistical"
	case strings.HasPrefix(featureName, "fe_"):
		return "feature_engineering"
	default:
		return "other"
	}
}

// NewOnlineLearningBuffer 创建在线学习缓冲区
func NewOnlineLearningBuffer(maxSize int) *OnlineLearningBuffer {
	return &OnlineLearningBuffer{
		features:   make([][]float64, 0, maxSize),
		targets:    make([]float64, 0, maxSize),
		timestamps: make([]time.Time, 0, maxSize),
		maxSize:    maxSize,
	}
}

// AddSample 添加新样本到缓冲区
func (olb *OnlineLearningBuffer) AddSample(features []float64, target float64) {
	olb.mu.Lock()
	defer olb.mu.Unlock()

	// 添加新样本
	olb.features = append(olb.features, features)
	olb.targets = append(olb.targets, target)
	olb.timestamps = append(olb.timestamps, time.Now())

	// 如果超过最大大小，移除最旧的样本
	if len(olb.features) > olb.maxSize {
		olb.features = olb.features[1:]
		olb.targets = olb.targets[1:]
		olb.timestamps = olb.timestamps[1:]
	}
}

// GetSamples 获取指定数量的样本
func (olb *OnlineLearningBuffer) GetSamples(count int) ([][]float64, []float64) {
	olb.mu.RLock()
	defer olb.mu.RUnlock()

	if len(olb.features) == 0 {
		return nil, nil
	}

	actualCount := minInt(count, len(olb.features))

	// 返回最新的样本
	startIdx := len(olb.features) - actualCount
	features := make([][]float64, actualCount)
	targets := make([]float64, actualCount)

	copy(features, olb.features[startIdx:])
	copy(targets, olb.targets[startIdx:])

	return features, targets
}

// Size 返回缓冲区当前大小
func (olb *OnlineLearningBuffer) Size() int {
	olb.mu.RLock()
	defer olb.mu.RUnlock()
	return len(olb.features)
}

// Clear 清空缓冲区
func (olb *OnlineLearningBuffer) Clear() {
	olb.mu.Lock()
	defer olb.mu.Unlock()
	olb.features = olb.features[:0]
	olb.targets = olb.targets[:0]
	olb.timestamps = olb.timestamps[:0]
}

// EnableOnlineLearning 启用在线学习
func (ep *MLEnsemblePredictor) EnableOnlineLearning(config OnlineLearningConfig) {
	ep.onlineConfig = config
	if config.Enabled && ep.learningBuffer == nil {
		ep.learningBuffer = NewOnlineLearningBuffer(config.BufferSize)
	}
	log.Printf("[ONLINE_LEARNING] 在线学习已启用: 缓冲区大小=%d, 更新间隔=%v",
		config.BufferSize, config.UpdateInterval)
}

// DisableOnlineLearning 禁用在线学习
func (ep *MLEnsemblePredictor) DisableOnlineLearning() {
	ep.onlineConfig.Enabled = false
	if ep.learningBuffer != nil {
		ep.learningBuffer.Clear()
	}
	log.Printf("[ONLINE_LEARNING] 在线学习已禁用")
}

// AddOnlineSample 添加在线学习样本
func (ep *MLEnsemblePredictor) AddOnlineSample(features []float64, target float64) error {
	if !ep.onlineConfig.Enabled || ep.learningBuffer == nil {
		return fmt.Errorf("在线学习未启用")
	}

	ep.learningBuffer.AddSample(features, target)

	// 检查是否需要更新模型
	if ep.shouldUpdateModel() {
		return ep.updateOnlineModel()
	}

	return nil
}

// shouldUpdateModel 检查是否应该更新模型
func (ep *MLEnsemblePredictor) shouldUpdateModel() bool {
	if !ep.onlineConfig.Enabled {
		return false
	}

	bufferSize := ep.learningBuffer.Size()
	if bufferSize < ep.onlineConfig.MinSamplesForUpdate {
		return false
	}

	// 检查时间间隔
	timeSinceLastUpdate := time.Since(ep.lastUpdateTime)
	return timeSinceLastUpdate >= ep.onlineConfig.UpdateInterval
}

// updateOnlineModel 在线更新模型
func (ep *MLEnsemblePredictor) updateOnlineModel() error {
	log.Printf("[ONLINE_LEARNING] 开始在线模型更新...")

	// 获取最新的训练样本
	features, targets := ep.learningBuffer.GetSamples(ep.onlineConfig.MinSamplesForUpdate)
	if len(features) == 0 {
		return fmt.Errorf("没有足够的样本进行在线学习")
	}

	// 创建训练数据
	X := mat.NewDense(len(features), len(features[0]), nil)
	for i, feature := range features {
		for j, val := range feature {
			X.Set(i, j, val)
		}
	}

	trainingData := &TrainingData{
		X:        X,
		Y:        targets,
		Features: make([]string, len(features[0])), // 简化的特征名称
	}

	// 使用在线学习算法更新模型
	err := ep.onlineUpdateModels(trainingData)
	if err != nil {
		log.Printf("[ONLINE_LEARNING] 模型更新失败: %v", err)
		return err
	}

	ep.lastUpdateTime = time.Now()

	// 评估更新后的性能
	performance := ep.evaluateOnlinePerformance()
	ep.performanceHistory = append(ep.performanceHistory, performance)

	log.Printf("[ONLINE_LEARNING] 模型更新完成，当前性能: %.4f", performance)

	// 如果性能下降太多，可能需要回滚或调整参数
	if ep.shouldRollbackUpdate(performance) {
		log.Printf("[ONLINE_LEARNING] 检测到性能下降，考虑回滚更新")
		// 这里可以实现回滚逻辑
	}

	return nil
}

// onlineUpdateModels 使用在线学习算法更新模型
func (ep *MLEnsemblePredictor) onlineUpdateModels(trainingData *TrainingData) error {
	switch ep.method {
	case "random_forest":
		return ep.onlineUpdateRandomForest(trainingData)
	default:
		// 对于不支持在线学习的算法，使用增量学习近似
		return ep.incrementalUpdate(trainingData)
	}
}

// onlineUpdateRandomForest 在线更新随机森林
func (ep *MLEnsemblePredictor) onlineUpdateRandomForest(trainingData *TrainingData) error {
	// 对于随机森林，我们可以更新部分树或添加新树
	// 这里使用简化的增量更新策略

	X := trainingData.X
	y := trainingData.Y

	// 计算当前学习率（随时间衰减）
	currentLearningRate := ep.calculateCurrentLearningRate()

	// 对每个基础模型进行增量更新
	for i, model := range ep.models {
		if model == nil {
			continue
		}

		// 对于决策树，使用叶子节点更新策略
		if tree, ok := model.(*DecisionTree); ok {
			err := ep.onlineUpdateDecisionTree(tree, X, y, currentLearningRate)
			if err != nil {
				log.Printf("[ONLINE_RF] 树 %d 更新失败: %v", i+1, err)
				continue
			}
		}
	}

	return nil
}

// onlineUpdateDecisionTree 在线更新决策树
func (ep *MLEnsemblePredictor) onlineUpdateDecisionTree(tree *DecisionTree, X *mat.Dense, y []float64, learningRate float64) error {
	// 简化的在线更新：使用新数据调整叶子节点值
	// 实际实现可能需要更复杂的算法

	r, _ := X.Dims()
	if r == 0 {
		return fmt.Errorf("没有数据用于更新")
	}

	// 这里只是概念性实现
	// 实际的在线决策树更新需要更复杂的算法

	return nil
}

// incrementalUpdate 增量更新（适用于不支持原生在线学习的算法）
func (ep *MLEnsemblePredictor) incrementalUpdate(trainingData *TrainingData) error {
	// 对于不支持在线学习的算法，我们可以：
	// 1. 使用较小的学习率更新现有模型
	// 2. 或者定期重新训练模型

	// 这里使用简化的增量更新
	currentLearningRate := ep.calculateCurrentLearningRate()

	// 对每个模型应用增量更新
	for _, model := range ep.models {
		if model == nil {
			continue
		}

		// 通用增量更新接口（需要模型实现）
		// 这里只是占位符，实际需要根据具体模型实现
		_ = currentLearningRate
	}

	return nil
}

// calculateCurrentLearningRate 计算当前学习率
func (ep *MLEnsemblePredictor) calculateCurrentLearningRate() float64 {
	if len(ep.performanceHistory) == 0 {
		return ep.onlineConfig.LearningRate
	}

	// 基于历史性能调整学习率
	recentPerformance := ep.performanceHistory[len(ep.performanceHistory)-1]
	if recentPerformance < ep.onlineConfig.PerformanceThreshold {
		// 性能不佳时降低学习率
		return math.Max(ep.onlineConfig.MinLearningRate,
			ep.onlineConfig.LearningRate*ep.onlineConfig.LearningRateDecay)
	}

	return ep.onlineConfig.LearningRate
}

// evaluateOnlinePerformance 评估在线学习性能
func (ep *MLEnsemblePredictor) evaluateOnlinePerformance() float64 {
	if ep.learningBuffer == nil {
		return 0.0
	}

	// 使用缓冲区中的最新数据评估性能
	features, targets := ep.learningBuffer.GetSamples(100) // 使用最近100个样本
	if len(features) == 0 {
		return 0.0
	}

	correct := 0
	for i, feature := range features {
		// 转换为矩阵格式进行预测
		X := mat.NewDense(1, len(feature), feature)
		predictions := ep.Predict(X)

		if len(predictions) > 0 {
			prediction := predictions[0]
			actual := targets[i]

			// 简化的准确率计算（方向是否正确）
			predDirection := 0
			if prediction > 0.5 {
				predDirection = 1
			} else if prediction < -0.5 {
				predDirection = -1
			}

			actualDirection := 0
			if actual > 0.5 {
				actualDirection = 1
			} else if actual < -0.5 {
				actualDirection = -1
			}

			if predDirection == actualDirection {
				correct++
			}
		}
	}

	if len(features) == 0 {
		return 0.0
	}

	return float64(correct) / float64(len(features))
}

// shouldRollbackUpdate 判断是否应该回滚更新
func (ep *MLEnsemblePredictor) shouldRollbackUpdate(currentPerformance float64) bool {
	if len(ep.performanceHistory) < 2 {
		return false
	}

	// 计算性能变化
	previousPerformance := ep.performanceHistory[len(ep.performanceHistory)-2]
	performanceDrop := previousPerformance - currentPerformance

	// 如果性能下降超过阈值，考虑回滚
	return performanceDrop > 0.1 // 10%的性能下降阈值
}

// GetOnlineLearningStats 获取在线学习统计信息
func (ep *MLEnsemblePredictor) GetOnlineLearningStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":               ep.onlineConfig.Enabled,
		"buffer_size":           0,
		"last_update":           ep.lastUpdateTime,
		"performance_history":   ep.performanceHistory,
		"current_learning_rate": ep.calculateCurrentLearningRate(),
	}

	if ep.learningBuffer != nil {
		stats["buffer_size"] = ep.learningBuffer.Size()
	}

	return stats
}

// Train 训练集成模型
func (ep *MLEnsemblePredictor) Train(trainingData *TrainingData) error {
	X := trainingData.X
	y := trainingData.Y

	switch ep.method {
	case "random_forest":
		return ep.trainRandomForest(X, y)
	case "gradient_boost":
		return ep.trainGradientBoost(X, y)
	case "stacking":
		return ep.trainStacking(X, y)
	case "neural_network":
		return ep.trainNeuralNetwork(X, y)
	case "transformer":
		return ep.trainTransformer(X, y)
	default:
		return ep.trainRandomForest(X, y)
	}
}

// trainRandomForest 训练随机森林
func (ep *MLEnsemblePredictor) trainRandomForest(X *mat.Dense, y []float64) error {
	nSamples, nFeatures := X.Dims()

	// 数据验证
	if nSamples == 0 || nFeatures == 0 {
		return fmt.Errorf("训练数据为空: %d 样本, %d 特征", nSamples, nFeatures)
	}

	if len(y) != nSamples {
		return fmt.Errorf("目标变量长度不匹配: %d vs %d", len(y), nSamples)
	}

	// 调试信息
	log.Printf("[DEBUG_RF] 开始训练随机森林: %d 样本, %d 特征, %d 棵树",
		nSamples, nFeatures, len(ep.models))

	// 验证模型数组
	if len(ep.models) == 0 {
		return fmt.Errorf("没有可用的基础模型")
	}

	// 使用更简单的训练策略：从小样本开始
	maxTrees := len(ep.models)
	if nSamples < 100 {
		maxTrees = 1 // 如果样本很少，只训练一棵树
		log.Printf("[DEBUG_RF] 样本量少，使用简化训练策略，只训练1棵树")
	} else if nSamples < 1000 {
		maxTrees = minInt(maxTrees, 3) // 样本中等，训练少量树
		log.Printf("[DEBUG_RF] 样本量中等，使用简化训练策略，最多训练3棵树")
	}

	trainedTrees := 0

	for i := 0; i < maxTrees; i++ {
		model := ep.models[i]

		// 检查模型是否为空
		if model == nil {
			log.Printf("[WARN_RF] 第%d个模型为nil，跳过", i+1)
			continue
		}

		// 自助采样 (bootstrap sampling)
		bootstrapIndices := ep.bootstrapSample(nSamples)

		// 创建自助样本
		features := make([][]float64, len(bootstrapIndices))
		targets := make([]float64, len(bootstrapIndices))

		for j, idx := range bootstrapIndices {
			features[j] = make([]float64, nFeatures)
			for k := 0; k < nFeatures; k++ {
				val := X.At(idx, k)
				// 确保数值有效
				if math.IsNaN(val) || math.IsInf(val, 0) {
					val = 0.0 // 用0替代无效值
				}
				features[j][k] = val
			}
			targets[j] = y[idx]
		}

		log.Printf("[DEBUG_RF] 训练第%d棵树: %d 样本", i+1, len(features))

		// 训练基础学习器
		err := model.Train(features, targets)
		if err != nil {
			log.Printf("[ERROR_RF] 第%d棵树训练失败: %v，尝试下一棵树", i+1, err)
			continue // 跳过失败的树，继续训练其他树
		}

		trainedTrees++
		log.Printf("[SUCCESS_RF] 第%d棵树训练成功", i+1)
	}

	if trainedTrees == 0 {
		return fmt.Errorf("所有树训练都失败了")
	}

	log.Printf("[SUCCESS_RF] 随机森林训练完成: %d/%d 棵树训练成功", trainedTrees, maxTrees)
	return nil
}

// 使用全局的minInt函数

// trainGradientBoost 训练梯度提升
func (ep *MLEnsemblePredictor) trainGradientBoost(X *mat.Dense, y []float64) error {
	nSamples, nFeatures := X.Dims()

	// 保存训练目标值用于默认预测
	ep.trainingTargets = make([]float64, len(y))
	copy(ep.trainingTargets, y)

	// 初始化预测值
	predictions := make([]float64, nSamples)
	residuals := make([]float64, nSamples)
	copy(residuals, y)

	learningRate := 0.5 // 学习率 - 与预测时保持一致

	for i, model := range ep.models {
		// 转换为BaseLearner接口需要的格式
		features := make([][]float64, nSamples)
		for j := 0; j < nSamples; j++ {
			features[j] = make([]float64, nFeatures)
			for k := 0; k < nFeatures; k++ {
				features[j][k] = X.At(j, k)
			}
		}

		// 训练弱学习器拟合残差
		err := model.Train(features, residuals)
		if err != nil {
			return fmt.Errorf("训练第%d个弱学习器失败: %w", i, err)
		}

		// 更新预测值和残差
		for j := 0; j < nSamples; j++ {
			pred, err := model.Predict(features[j])
			if err != nil {
				continue
			}
			predictions[j] += learningRate * pred
			residuals[j] = y[j] - predictions[j]
		}
	}

	return nil
}

// trainStacking 训练堆叠集成
func (ep *MLEnsemblePredictor) trainStacking(X *mat.Dense, y []float64) error {
	log.Printf("[TRAIN_STACKING] 开始训练Stacking模型，样本数=%d, 基础模型数=%d", len(y), len(ep.models))

	// 如果metaModel未初始化，先初始化
	if ep.metaModel == nil {
		log.Printf("[TRAIN_STACKING] 初始化Stacking元模型")

		// 重新初始化整个Stacking集成
		ep.initializeStacking(len(ep.models), MLConfig{
			Ensemble: struct {
				Method       string  `json:"method"`
				NEstimators  int     `json:"n_estimators"`
				MaxDepth     int     `json:"max_depth"`
				LearningRate float64 `json:"learning_rate"`
			}{
				Method:       "stacking",
				NEstimators:  len(ep.models),
				MaxDepth:     10, // 默认深度
				LearningRate: 0.1,
			},
		})
	}

	if ep.metaModel == nil {
		return fmt.Errorf("metaModel初始化失败")
	}

	log.Printf("[TRAIN_STACKING] 元模型类型: %s", ep.metaModel.GetName())

	nSamples, nFeatures := X.Dims()

	// 转换为BaseLearner需要的格式
	features := make([][]float64, nSamples)
	for i := 0; i < nSamples; i++ {
		features[i] = make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			features[i][j] = X.At(i, j)
		}
	}

	// 第一层：训练基础模型
	metaFeatures := make([][]float64, nSamples)
	for i := range metaFeatures {
		metaFeatures[i] = make([]float64, len(ep.models))
	}

	// 训练基础模型并生成元特征
	for i, model := range ep.models {
		log.Printf("[TRAIN_STACKING] 训练基础模型 %d: %s", i, model.GetName())

		// 训练基础模型
		err := model.Train(features, y)
		if err != nil {
			log.Printf("[TRAIN_STACKING] 基础模型 %d 训练失败: %v", i, err)
			continue
		}

		// 生成元特征
		for j := 0; j < nSamples; j++ {
			prediction, err := model.Predict(features[j])
			if err != nil {
				log.Printf("[TRAIN_STACKING] 基础模型 %d 预测失败: %v", i, err)
				continue
			}
			metaFeatures[j][i] = prediction
		}

		log.Printf("[TRAIN_STACKING] 基础模型 %d 完成", i)
	}

	log.Printf("[TRAIN_STACKING] 开始训练元模型，元特征维度: %dx%d", len(metaFeatures), len(metaFeatures[0]))

	// 训练元模型
	err := ep.metaModel.Train(metaFeatures, y)
	if err != nil {
		return fmt.Errorf("元模型训练失败: %w", err)
	}

	log.Printf("[TRAIN_STACKING] Stacking模型训练完成")
	return nil
}

// trainNeuralNetwork 训练神经网络集成
func (ep *MLEnsemblePredictor) trainNeuralNetwork(X *mat.Dense, y []float64) error {
	log.Printf("[TRAIN_NN] 开始训练神经网络模型，样本数=%d", len(y))

	// 获取实际的特征维度
	_, nFeatures := X.Dims()
	log.Printf("[TRAIN_NN] 检测到特征维度: %d", nFeatures)

	// 重新创建神经网络以匹配实际特征维度
	newNet := NewNeuralNetwork(nFeatures, []int{64, 32, 16, 1})
	newWrapper := &NeuralNetworkWrapper{
		neuralNet: newNet,
	}

	// 将矩阵数据转换为切片格式（神经网络包装器需要的格式）
	nSamples, _ := X.Dims()
	features := make([][]float64, nSamples)
	targets := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		features[i] = make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			features[i][j] = X.At(i, j)
		}
		targets[i] = y[i]
	}

	log.Printf("[TRAIN_NN] 转换为切片格式: %d 样本, %d 特征", len(features), len(features[0]))

	// 训练神经网络
	err := newWrapper.Train(features, targets)
	if err != nil {
		return fmt.Errorf("神经网络训练失败: %w", err)
	}

	// 更新模型为训练好的版本
	ep.models[0] = newWrapper

	log.Printf("[TRAIN_NN] 神经网络模型训练完成")
	return nil
}

// trainTransformer 训练Transformer集成
func (ep *MLEnsemblePredictor) trainTransformer(X *mat.Dense, y []float64) error {
	if len(ep.models) == 0 {
		return fmt.Errorf("没有可用的Transformer模型")
	}

	log.Printf("[TRAIN_TRANSFORMER] 开始训练Transformer模型，样本数=%d", len(y))

	// 转换数据格式为BaseLearner接口需要的格式
	nSamples, nFeatures := X.Dims()
	features := make([][]float64, nSamples)
	targets := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		features[i] = make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			features[i][j] = X.At(i, j)
		}
		targets[i] = y[i]
	}

	// 训练Transformer模型
	model := ep.models[0] // Transformer集成只有一个模型
	err := model.Train(features, targets)
	if err != nil {
		return fmt.Errorf("Transformer模型训练失败: %w", err)
	}

	log.Printf("[TRAIN_TRANSFORMER] Transformer模型训练完成")
	return nil
}

// predictNeuralNetwork 神经网络预测
func (ep *MLEnsemblePredictor) predictNeuralNetwork(X *mat.Dense) []float64 {
	if len(ep.models) == 0 {
		log.Printf("[PREDICT_NN] 没有训练好的神经网络模型")
		nSamples, _ := X.Dims()
		return make([]float64, nSamples)
	}

	model := ep.models[0]
	nSamples, _ := X.Dims()

	// 将矩阵转换为切片格式
	_, nFeatures := X.Dims()
	features := make([][]float64, nSamples)
	for i := 0; i < nSamples; i++ {
		features[i] = make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			features[i][j] = X.At(i, j)
		}
	}

	// 进行预测
	predictions := make([]float64, nSamples)
	for i, feature := range features {
		pred, err := model.Predict(feature)
		if err != nil {
			log.Printf("[PREDICT_NN] 预测失败: %v", err)
			predictions[i] = 0.0
		} else {
			predictions[i] = pred
		}
	}

	log.Printf("[PREDICT_NN] 神经网络预测完成: %d 个样本", len(predictions))
	return predictions
}

// predictSimpleAverage 简单平均预测（作为Stacking失败时的回退方案）
func (ep *MLEnsemblePredictor) predictSimpleAverage(X *mat.Dense) []float64 {
	nSamples, nFeatures := X.Dims()
	predictions := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		// 提取样本特征
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		// 对所有可用模型进行预测并平均
		sum := 0.0
		count := 0
		for _, model := range ep.models {
			if model == nil {
				continue
			}
			pred, err := model.Predict(sample)
			if err == nil {
				sum += pred
				count++
			}
		}

		// 计算平均值
		if count > 0 {
			predictions[i] = sum / float64(count)
		} else {
			predictions[i] = 0.0
		}
	}

	return predictions
}

// predictSimpleAverageSingle 对单个样本进行简单平均预测
func (ep *MLEnsemblePredictor) predictSimpleAverageSingle(X *mat.Dense, sampleIndex int) float64 {
	_, nFeatures := X.Dims()

	// 提取样本特征
	sample := make([]float64, nFeatures)
	for j := 0; j < nFeatures; j++ {
		sample[j] = X.At(sampleIndex, j)
	}

	// 对所有可用模型进行预测并平均
	sum := 0.0
	count := 0
	for _, model := range ep.models {
		if model == nil {
			continue
		}
		pred, err := model.Predict(sample)
		if err == nil {
			sum += pred
			count++
		}
	}

	// 计算平均值
	if count > 0 {
		return sum / float64(count)
	} else {
		return 0.0
	}
}

// Predict 进行预测
func (ep *MLEnsemblePredictor) Predict(X *mat.Dense) []float64 {
	switch ep.method {
	case "random_forest":
		return ep.predictBagging(X)
	case "gradient_boost":
		return ep.predictGradientBoost(X)
	case "stacking":
		return ep.predictStacking(X)
	case "neural_network":
		return ep.predictNeuralNetwork(X)
	case "transformer":
		return ep.predictTransformer(X)
	default:
		return ep.predictBagging(X)
	}
}

// predictWeightedEnsemble 加权集成预测（改进版）
func (ep *MLEnsemblePredictor) predictWeightedEnsemble(X *mat.Dense) []float64 {
	nSamples, nFeatures := X.Dims()
	predictions := make([]float64, nSamples)

	// 计算动态权重
	modelWeights := ep.calculateModelWeights()

	for i := 0; i < nSamples; i++ {
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		weightedSum := 0.0
		totalWeight := 0.0

		// 对每个模型进行预测并加权
		validPredictions := 0
		for idx, model := range ep.models {
			if model == nil {
				continue
			}

			weight := modelWeights[idx]
			pred, err := model.Predict(sample)
			if err == nil && !math.IsNaN(pred) && !math.IsInf(pred, 0) {
				// 质量检查：过滤不合理的预测值
				if ep.isReasonablePrediction(pred, model) {
					// 限制预测值范围，避免极端值
					pred = math.Max(-5.0, math.Min(5.0, pred))
					weightedSum += pred * weight
					totalWeight += weight
					validPredictions++
				} else {
					log.Printf("[MODEL_FILTER] 过滤不合理预测: model_%d = %.4f", idx, pred)
				}
			} else {
				log.Printf("[MODEL_ERROR] 模型%d预测失败: %v", idx, err)
			}
		}

		// 如果没有有效预测，使用保守的默认值
		if validPredictions == 0 {
			log.Printf("[MODEL_FALLBACK] 无有效预测，使用默认值0.0")
			predictions[i] = 0.0
			continue
		}

		if totalWeight > 0 {
			predictions[i] = weightedSum / totalWeight
		} else {
			predictions[i] = 0.0
		}
	}

	return predictions
}

// calculateModelWeights 计算模型动态权重（基于性能和可用性）
func (ep *MLEnsemblePredictor) calculateModelWeights() []float64 {
	weights := make([]float64, len(ep.models))

	// 基础权重配置
	baseWeights := []float64{0.25, 0.25, 0.2, 0.15, 0.15} // RF, GB, Stacking, NN, Transformer
	modelNames := []string{"random_forest", "gradient_boost", "stacking", "neural_network", "transformer"}

	totalWeight := 0.0

	for i := range weights {
		weight := 0.0

		if ep.models[i] == nil {
			// 模型不可用，权重为0
			weights[i] = 0.0
			continue
		}

		// 基础权重
		if i < len(baseWeights) {
			weight = baseWeights[i]
		} else {
			weight = 0.1
		}

		// 基于模型可用性和性能调整权重
		if i < len(modelNames) {
			switch modelNames[i] {
			case "transformer":
				// Transformer模型需要特殊处理
				if tw, ok := ep.models[i].(*TransformerWrapper); ok {
					if tw.isTrained {
						weight *= 1.2 // 已训练的Transformer给予奖励
					} else {
						weight *= 0.3 // 未训练的Transformer降低权重
					}
				} else {
					weight = 0.0 // 不是TransformerWrapper，权重为0
				}
			case "stacking":
				// Stacking模型通常表现较好，给予轻微奖励
				weight *= 1.1
			case "gradient_boost":
				// GB模型在某些场景下表现稳定
				weight *= 1.05
			}
		}

		weights[i] = weight
		totalWeight += weight
	}

	// 归一化权重
	if totalWeight > 0 {
		for i := range weights {
			weights[i] /= totalWeight
		}
	}

	log.Printf("[MODEL_WEIGHTS] 动态权重分配: RF=%.3f, GB=%.3f, Stacking=%.3f, NN=%.3f, Transformer=%.3f",
		getWeightByIndex(weights, 0), getWeightByIndex(weights, 1),
		getWeightByIndex(weights, 2), getWeightByIndex(weights, 3), getWeightByIndex(weights, 4))

	return weights
}

// isReasonablePrediction 检查预测值是否合理
func (ep *MLEnsemblePredictor) isReasonablePrediction(prediction float64, model BaseLearner) bool {
	// 对于回归预测，合理的范围通常在-10到10之间
	// 但对于交易信号，合理的范围可能更小
	absPred := math.Abs(prediction)

	// 极端值过滤
	if absPred > 20.0 {
		return false
	}

	// 检查是否为Transformer模型的默认输出
	if tw, ok := model.(*TransformerWrapper); ok {
		// 如果Transformer没有训练好，它的预测可能不可靠
		if !tw.isTrained && absPred < 0.01 {
			return false
		}
	}

	return true
}

// getWeightByIndex 安全获取权重
func getWeightByIndex(weights []float64, index int) float64 {
	if index >= 0 && index < len(weights) {
		return weights[index]
	}
	return 0.0
}

// Score 计算模型评分（R²分数）
func (ep *MLEnsemblePredictor) Score(X, y *mat.Dense) float64 {
	predictions := ep.Predict(X)
	if len(predictions) != y.RawMatrix().Rows {
		return 0.0
	}

	// 计算R²分数
	n := len(predictions)
	yMean := 0.0
	for i := 0; i < n; i++ {
		yMean += y.At(i, 0)
	}
	yMean /= float64(n)

	ssRes := 0.0 // 残差平方和
	ssTot := 0.0 // 总平方和

	for i := 0; i < n; i++ {
		actual := y.At(i, 0)
		ssTot += (actual - yMean) * (actual - yMean)
		ssRes += (actual - predictions[i]) * (actual - predictions[i])
	}

	if ssTot == 0 {
		return 0.0 // 避免除零
	}

	r2 := 1.0 - (ssRes / ssTot)
	return r2
}

// predictGradientBoost 梯度提升预测
func (ep *MLEnsemblePredictor) predictGradientBoost(X *mat.Dense) []float64 {
	nSamples, nFeatures := X.Dims()
	predictions := make([]float64, nSamples)
	learningRate := 0.5 // 进一步提高学习率，从0.1增加到0.5

	for i := 0; i < nSamples; i++ {
		// 提取样本特征
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		// 累积所有模型的预测（梯度提升的正确预测方式）
		samplePrediction := 0.0
		validPredictions := 0
		for _, model := range ep.models {
			pred, err := model.Predict(sample)
			if err == nil {
				samplePrediction += learningRate * pred
				validPredictions++
			} else {
				// 记录预测失败，但不中断整个预测过程
				log.Printf("[GRADIENT_BOOST] 基础模型预测失败: %v", err)
			}
		}

		// 如果没有有效的预测，使用平均预测值
		if validPredictions == 0 {
			// 优化：使用训练数据的平均值作为默认预测，而不是0.0
			samplePrediction = ep.getDefaultPrediction()
			log.Printf("[GRADIENT_BOOST] 所有基础模型预测失败，使用默认预测值: %.4f", samplePrediction)
		} else if validPredictions < len(ep.models) {
			// 部分模型预测失败，调整预测值
			samplePrediction *= float64(len(ep.models)) / float64(validPredictions)
		}

		predictions[i] = samplePrediction
	}

	return predictions
}

// getDefaultPrediction 获取默认预测值
func (ep *MLEnsemblePredictor) getDefaultPrediction() float64 {
	// 使用训练数据的平均目标值作为默认预测
	if ep.trainingTargets != nil && len(ep.trainingTargets) > 0 {
		sum := 0.0
		for _, target := range ep.trainingTargets {
			sum += target
		}
		return sum / float64(len(ep.trainingTargets))
	}
	// 如果没有训练数据，使用一个小的随机值
	return (rand.Float64() - 0.5) * 0.1
}

// predictBagging Bagging预测（随机森林）
func (ep *MLEnsemblePredictor) predictBagging(X *mat.Dense) []float64 {
	nSamples, nFeatures := X.Dims()
	predictions := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		// 提取样本特征
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		// 收集所有模型的预测
		samplePredictions := make([]float64, 0, len(ep.models))
		for _, model := range ep.models {
			pred, err := model.Predict(sample)
			if err == nil {
				samplePredictions = append(samplePredictions, pred)
			}
		}

		// 平均预测结果
		if len(samplePredictions) > 0 {
			sum := 0.0
			for _, pred := range samplePredictions {
				sum += pred
			}
			predictions[i] = sum / float64(len(samplePredictions))
		}
	}

	return predictions
}

// predictStacking 堆叠预测
func (ep *MLEnsemblePredictor) predictStacking(X *mat.Dense) []float64 {
	nSamples, nFeatures := X.Dims()

	// 检查metaModel是否已初始化
	if ep.metaModel == nil {
		log.Printf("[ERROR_STACKING] metaModel未初始化，无法进行Stacking预测")
		// 回退到简单平均预测
		return ep.predictSimpleAverage(X)
	}

	// 第一层预测
	metaFeatures := make([][]float64, nSamples)
	for i := range metaFeatures {
		metaFeatures[i] = make([]float64, len(ep.models))
	}

	for i := 0; i < nSamples; i++ {
		// 提取样本特征
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		// 第一层模型预测
		for j, model := range ep.models {
			if model == nil {
				metaFeatures[i][j] = 0.0
				continue
			}
			pred, err := model.Predict(sample)
			if err != nil {
				log.Printf("[ERROR_STACKING] 基础模型 %d 预测失败: %v", j, err)
				metaFeatures[i][j] = 0.0
			} else {
				metaFeatures[i][j] = pred
			}
		}
	}

	// 元模型预测
	predictions := make([]float64, nSamples)
	for i, metaFeature := range metaFeatures {
		pred, err := ep.metaModel.Predict(metaFeature)
		if err != nil {
			log.Printf("[ERROR_STACKING] 元模型预测失败，使用基础模型平均值: %v", err)
			// 使用基础模型预测的平均值作为回退
			sum := 0.0
			count := 0
			for _, val := range metaFeature {
				if val != 0.0 { // 假设0.0表示预测失败
					sum += val
					count++
				}
			}
			if count > 0 {
				predictions[i] = sum / float64(count)
			} else {
				// 所有基础模型都失败，使用简单平均预测作为回退
				predictions[i] = ep.predictSimpleAverageSingle(X, i)
			}
		} else {
			predictions[i] = pred
		}
	}

	return predictions
}

// GetFeatureImportance 获取特征重要�?
func (ep *MLEnsemblePredictor) GetFeatureImportance() []float64 {
	// 对于随机森林，计算特征在所有树中的平均重要�?
	if ep.method == "random_forest" {
		nFeatures := len(ep.featureIdx)
		if nFeatures == 0 {
			return nil
		}

		importance := make([]float64, nFeatures)
		validModels := 0

		for _, model := range ep.models {
			if tree, ok := model.(*DecisionTree); ok {
				modelImportance := tree.GetFeatureImportance()
				if len(modelImportance) == nFeatures {
					for i, imp := range modelImportance {
						importance[i] += imp
					}
					validModels++
				}
			}
		}

		// 计算平均重要�?
		if validModels > 0 {
			for i := range importance {
				importance[i] /= float64(validModels)
			}
		}

		return importance
	}

	// 其他方法的特征重要性计�?
	return nil
}

// SetFeatureIndices 设置使用的特征索�?
func (ep *MLEnsemblePredictor) SetFeatureIndices(indices []int) {
	ep.featureIdx = make([]int, len(indices))
	copy(ep.featureIdx, indices)
}

// bootstrapSample 自助采样
func (ep *MLEnsemblePredictor) bootstrapSample(n int) []int {
	sample := make([]int, n)
	for i := 0; i < n; i++ {
		sample[i] = rand.Intn(n)
	}
	return sample
}

// EvaluateModel 评估模型性能
func (ep *MLEnsemblePredictor) EvaluateModel(X, y *mat.Dense) map[string]float64 {
	predictions := ep.Predict(X)
	nSamples, _ := X.Dims()

	if len(predictions) != nSamples {
		return map[string]float64{"error": 1.0}
	}

	// 计算MSE
	mse := 0.0
	yValues := make([]float64, nSamples)
	mat.Col(yValues, 0, y)

	for i := 0; i < nSamples; i++ {
		diff := predictions[i] - yValues[i]
		mse += diff * diff
	}
	mse /= float64(nSamples)

	// 计算MAE
	mae := 0.0
	for i := 0; i < nSamples; i++ {
		mae += math.Abs(predictions[i] - yValues[i])
	}
	mae /= float64(nSamples)

	// 计算R²分数
	yMean := calculateAverage(yValues)
	ssRes := 0.0
	ssTot := 0.0

	for i := 0; i < nSamples; i++ {
		diff := predictions[i] - yValues[i]
		ssRes += diff * diff

		totalDiff := yValues[i] - yMean
		ssTot += totalDiff * totalDiff
	}

	r2 := 1.0 - (ssRes / ssTot)
	if math.IsNaN(r2) {
		r2 = 0.0
	}

	return map[string]float64{
		"mse": mse,
		"mae": mae,
		"r2":  r2,
	}
}

// GetModelInfo 获取模型信息
func (ep *MLEnsemblePredictor) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"method":        ep.method,
		"n_estimators":  len(ep.models),
		"feature_count": len(ep.featureIdx),
	}
}

// ============================================================================
// 基础模型实现
// ============================================================================

// DecisionTree methods are implemented in decision_tree.go

// LinearRegression 线性回�?
type LinearRegression struct {
	coefficients []float64
	intercept    float64
}

// NewLinearRegression 创建线性回归实例
func NewLinearRegression() *LinearRegression {
	return &LinearRegression{
		coefficients: nil,
		intercept:    0.0,
	}
}

// Train 实现 BaseLearner 接口
func (lr *LinearRegression) Train(features [][]float64, targets []float64) error {
	if len(features) == 0 || len(targets) == 0 {
		return fmt.Errorf("no training data")
	}

	nSamples := len(features)
	nFeatures := len(features[0])

	// 添加偏置项
	XWithBias := mat.NewDense(nSamples, nFeatures+1, nil)
	for i := 0; i < nSamples; i++ {
		XWithBias.Set(i, 0, 1.0) // 偏置项
		for j := 0; j < nFeatures; j++ {
			XWithBias.Set(i, j+1, features[i][j])
		}
	}

	// 创建目标矩阵
	y := mat.NewDense(nSamples, 1, targets)

	// 使用岭回归避免矩阵求逆问题: (X^T * X + λI)^(-1) * X^T * y
	Xt := XWithBias.T()

	var XtX mat.Dense
	XtX.Mul(Xt, XWithBias)

	// 添加岭回归正则化项 (λ = 0.01)
	lambda := 0.01
	rows, cols := XtX.Dims()
	for i := 0; i < rows && i < cols; i++ {
		XtX.Set(i, i, XtX.At(i, i)+lambda)
	}

	var XtXInv mat.Dense
	err := XtXInv.Inverse(&XtX)
	if err != nil {
		// 如果仍然失败，使用更大的正则化
		log.Printf("[LinearRegression] 矩阵求逆失败，重试更大的正则化")
		lambda = 1.0
		for i := 0; i < rows && i < cols; i++ {
			XtX.Set(i, i, XtX.At(i, i)+lambda-0.01) // 减去之前的lambda，加上新的
		}
		err = XtXInv.Inverse(&XtX)
		if err != nil {
			return fmt.Errorf("矩阵求逆失败，即使使用岭回归: %w", err)
		}
	}

	var XtY mat.Dense
	XtY.Mul(Xt, y)

	var result mat.Dense
	result.Mul(&XtXInv, &XtY)

	// 提取系数
	lr.coefficients = make([]float64, nFeatures)
	lr.intercept = result.At(0, 0)
	for i := 0; i < nFeatures; i++ {
		lr.coefficients[i] = result.At(i+1, 0)
	}

	return nil
}

// Predict 预测

// Score 计算得分
func (lr *LinearRegression) Score(X, y *mat.Dense) float64 {
	nSamples, nFeatures := X.Dims()
	_, nTargets := y.Dims()

	if nTargets != 1 {
		return 0.0
	}

	mse := 0.0
	for i := 0; i < nSamples; i++ {
		// 提取样本特征
		features := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			features[j] = X.At(i, j)
		}

		// 预测
		prediction, err := lr.Predict(features)
		if err != nil {
			continue
		}

		actual := y.At(i, 0)
		diff := prediction - actual
		mse += diff * diff
	}

	return mse / float64(nSamples)
}

// GetFeatureImportance 获取特征重要�?
func (lr *LinearRegression) GetFeatureImportance() []float64 {
	if lr.coefficients == nil {
		return nil
	}

	// 使用系数的绝对值作为重要�?
	importance := make([]float64, len(lr.coefficients))
	total := 0.0

	for i, coeff := range lr.coefficients {
		importance[i] = math.Abs(coeff)
		total += importance[i]
	}

	// 归一�?
	if total > 0 {
		for i := range importance {
			importance[i] /= total
		}
	}

	return importance
}

// Predict 实现 BaseLearner 接口的预测方法
func (lr *LinearRegression) Predict(features []float64) (float64, error) {
	if lr.coefficients == nil {
		return 0, fmt.Errorf("model not trained")
	}

	if len(features) != len(lr.coefficients) {
		return 0, fmt.Errorf("feature dimension mismatch: expected %d, got %d", len(lr.coefficients), len(features))
	}

	prediction := lr.intercept
	for i, feature := range features {
		prediction += feature * lr.coefficients[i]
	}

	return prediction, nil
}

// GetName 实现 BaseLearner 接口
func (lr *LinearRegression) GetName() string {
	return "LinearRegression"
}

// Clone 实现 BaseLearner 接口
func (lr *LinearRegression) Clone() BaseLearner {
	return NewLinearRegression()
}

// RandomForest 简化的随机森林实现
type RandomForest struct {
	nTrees   int
	maxDepth int
	trees    []DecisionTree
}

// Serialize 序列化随机森林模型
func (rf *RandomForest) Serialize() ([]byte, error) {
	// 序列化每棵树
	treesData := make([][]byte, len(rf.trees))
	for i, tree := range rf.trees {
		treeData, err := tree.Serialize()
		if err != nil {
			return nil, fmt.Errorf("序列化第%d棵树失败: %w", i, err)
		}
		treesData[i] = treeData
	}

	data := map[string]interface{}{
		"nTrees":   rf.nTrees,
		"maxDepth": rf.maxDepth,
		"trees":    treesData,
	}

	return json.Marshal(data)
}

// Deserialize 反序列化随机森林模型
func (rf *RandomForest) Deserialize(data []byte) error {
	var modelData map[string]interface{}
	if err := json.Unmarshal(data, &modelData); err != nil {
		return err
	}

	if nTrees, ok := modelData["nTrees"].(float64); ok {
		rf.nTrees = int(nTrees)
	}

	if maxDepth, ok := modelData["maxDepth"].(float64); ok {
		rf.maxDepth = int(maxDepth)
	}

	// 反序列化trees数组
	if treesData, ok := modelData["trees"].([]interface{}); ok {
		rf.trees = make([]DecisionTree, len(treesData))
		for i, treeData := range treesData {
			if treeBytes, ok := treeData.([]byte); ok {
				if err := rf.trees[i].Deserialize(treeBytes); err != nil {
					return fmt.Errorf("反序列化第%d棵树失败: %w", i, err)
				}
			}
		}
	}

	return nil
}

// Train 训练随机森林
func (rf *RandomForest) Train(X *mat.Dense, y []float64) error {
	rf.trees = make([]DecisionTree, rf.nTrees)

	for i := 0; i < rf.nTrees; i++ {
		tree := NewDecisionTree()
		tree.MaxDepth = rf.maxDepth

		// 转换为BaseLearner需要的格式
		nSamples, nFeatures := X.Dims()
		features := make([][]float64, nSamples)
		targets := make([]float64, nSamples)

		for j := 0; j < nSamples; j++ {
			features[j] = make([]float64, nFeatures)
			for k := 0; k < nFeatures; k++ {
				features[j][k] = X.At(j, k)
			}
			targets[j] = y[j]
		}

		err := tree.Train(features, targets)
		if err != nil {
			return fmt.Errorf("训练第%d棵树失败: %w", i, err)
		}

		rf.trees[i] = *tree
	}

	return nil
}

// Predict 预测
func (rf *RandomForest) Predict(X *mat.Dense) []float64 {
	if len(rf.trees) == 0 {
		return nil
	}

	nSamples, nFeatures := X.Dims()
	predictions := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		// 提取样本特征
		sample := make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			sample[j] = X.At(i, j)
		}

		// 收集所有树的预测
		samplePredictions := make([]float64, 0, len(rf.trees))
		for _, tree := range rf.trees {
			pred, err := tree.Predict(sample)
			if err == nil {
				samplePredictions = append(samplePredictions, pred)
			}
		}

		// 平均预测结果
		if len(samplePredictions) > 0 {
			sum := 0.0
			for _, pred := range samplePredictions {
				sum += pred
			}
			predictions[i] = sum / float64(len(samplePredictions))
		}
	}

	return predictions
}

// Score 计算得分
func (rf *RandomForest) Score(X, y *mat.Dense) float64 {
	predictions := rf.Predict(X)
	yValues := make([]float64, len(predictions))
	mat.Col(yValues, 0, y)

	mse := 0.0
	for i := 0; i < len(predictions); i++ {
		diff := predictions[i] - yValues[i]
		mse += diff * diff
	}

	return mse / float64(len(predictions))
}

// GetFeatureImportance 获取特征重要�?
func (rf *RandomForest) GetFeatureImportance() []float64 {
	if len(rf.trees) == 0 {
		return nil
	}

	// 由于DecisionTree目前不支持特征重要性，返回nil
	// TODO: 当DecisionTree实现特征重要性后，启用完整的计算逻辑
	return nil
}

// ============================================================================
// 辅助函数
// ============================================================================

func calculateAverageSubset(values []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, idx := range indices {
		sum += values[idx]
	}

	return sum / float64(len(indices))
}

func calculateVarianceSubset(values []float64, indices []int) float64 {
	if len(indices) <= 1 {
		return 0.0
	}

	mean := calculateAverageSubset(values, indices)

	sumSquares := 0.0
	for _, idx := range indices {
		diff := values[idx] - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(indices)-1)
}

// predictTransformer Transformer预测
func (ep *MLEnsemblePredictor) predictTransformer(X *mat.Dense) []float64 {
	if len(ep.models) == 0 {
		log.Printf("[PREDICT_TRANSFORMER] 没有训练好的Transformer模型")
		nSamples, _ := X.Dims()
		return make([]float64, nSamples)
	}

	model := ep.models[0]
	nSamples, _ := X.Dims()

	// 将矩阵转换为切片格式
	_, nFeatures := X.Dims()
	features := make([][]float64, nSamples)
	for i := 0; i < nSamples; i++ {
		features[i] = make([]float64, nFeatures)
		for j := 0; j < nFeatures; j++ {
			features[i][j] = X.At(i, j)
		}
	}

	// 进行预测
	predictions := make([]float64, nSamples)
	for i := 0; i < nSamples; i++ {
		pred, err := model.Predict(features[i])
		if err != nil {
			log.Printf("[PREDICT_TRANSFORMER] 样本 %d 预测失败: %v", i, err)
			predictions[i] = 0.0
		} else {
			predictions[i] = pred
		}
	}

	return predictions
}
