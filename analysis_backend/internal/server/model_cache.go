package server

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ModelCache 模型缓存系统
type ModelCache struct {
	mu         sync.RWMutex
	cache      map[string]*CachedModel
	maxSize    int
	defaultTTL time.Duration
}

// CachedModel 缓存的模型
type CachedModel struct {
	Model     BaseLearner
	Ensemble  *EnsemblePredictor
	DataHash  string // 数据哈希，用于验证数据是否变化
	CreatedAt time.Time
	LastUsed  time.Time
	HitCount  int
	TTL       time.Duration
}

// NewModelCache 创建模型缓存
func NewModelCache(maxSize int, defaultTTL time.Duration) *ModelCache {
	return &ModelCache{
		cache:      make(map[string]*CachedModel),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
	}
}

// Get 获取缓存的模型
func (mc *ModelCache) Get(key string) (BaseLearner, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	cached, exists := mc.cache[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Since(cached.CreatedAt) > cached.TTL {
		return nil, false
	}

	// 更新使用统计
	cached.LastUsed = time.Now()
	cached.HitCount++

	// 克隆模型以避免并发修改
	if cached.Model != nil {
		return cached.Model.Clone(), true
	}

	return nil, false
}

// GetEnsemble 获取缓存的集成模型
func (mc *ModelCache) GetEnsemble(key string) (*EnsemblePredictor, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	cached, exists := mc.cache[key]
	if !exists || cached.Ensemble == nil {
		return nil, false
	}

	// 检查是否过期
	if time.Since(cached.CreatedAt) > cached.TTL {
		return nil, false
	}

	// 更新使用统计
	cached.LastUsed = time.Now()
	cached.HitCount++

	return cached.Ensemble, true
}

// Put 存储模型到缓存
func (mc *ModelCache) Put(key string, model BaseLearner, dataHash string) {
	mc.putModel(key, &CachedModel{
		Model:     model,
		DataHash:  dataHash,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		HitCount:  0,
		TTL:       mc.defaultTTL,
	})
}

// PutEnsemble 存储集成模型到缓存
func (mc *ModelCache) PutEnsemble(key string, ensemble *EnsemblePredictor, dataHash string) {
	mc.putModel(key, &CachedModel{
		Ensemble:  ensemble,
		DataHash:  dataHash,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		HitCount:  0,
		TTL:       mc.defaultTTL,
	})
}

// putModel 内部存储方法
func (mc *ModelCache) putModel(key string, cached *CachedModel) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 如果缓存已满，清理最少使用的项目
	if len(mc.cache) >= mc.maxSize {
		mc.evictLeastRecentlyUsed()
	}

	mc.cache[key] = cached
}

// evictLeastRecentlyUsed 清理最少使用的缓存项
func (mc *ModelCache) evictLeastRecentlyUsed() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range mc.cache {
		if oldestKey == "" || cached.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.LastUsed
		}
	}

	if oldestKey != "" {
		delete(mc.cache, oldestKey)
	}
}

// Invalidate 使缓存失效
func (mc *ModelCache) Invalidate(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.cache, key)
}

// Clear 清空缓存
func (mc *ModelCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.cache = make(map[string]*CachedModel)
}

// Stats 获取缓存统计信息
func (mc *ModelCache) Stats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	totalHits := 0
	totalItems := len(mc.cache)

	for _, cached := range mc.cache {
		totalHits += cached.HitCount
	}

	return map[string]interface{}{
		"total_items":       totalItems,
		"max_size":          mc.maxSize,
		"total_hits":        totalHits,
		"hit_rate":          mc.calculateHitRate(),
		"cache_utilization": float64(totalItems) / float64(mc.maxSize),
	}
}

// calculateHitRate 计算缓存命中率
func (mc *ModelCache) calculateHitRate() float64 {
	totalRequests := 0
	totalHits := 0

	for _, cached := range mc.cache {
		totalRequests += cached.HitCount + 1 // +1 for initial miss
		totalHits += cached.HitCount
	}

	if totalRequests == 0 {
		return 0
	}

	return float64(totalHits) / float64(totalRequests)
}

// generateDataHash 生成数据哈希
func generateDataHash(features [][]float64, targets []float64) string {
	data := fmt.Sprintf("%v-%v", features, targets)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// ParallelTrainer 并行训练器
type ParallelTrainer struct {
	maxWorkers int
}

// NewParallelTrainer 创建并行训练器
func NewParallelTrainer(maxWorkers int) *ParallelTrainer {
	return &ParallelTrainer{
		maxWorkers: maxWorkers,
	}
}

// TrainEnsemble 并行训练集成模型
func (pt *ParallelTrainer) TrainEnsemble(ensemble *EnsemblePredictor, features [][]float64, targets []float64) error {
	// 对于Bagging，可以并行训练各个基础学习器
	if ensemble.Type == EnsembleTypeBagging {
		return pt.trainBaggingParallel(ensemble, features, targets)
	}

	// 对于其他类型，使用串行训练
	return ensemble.Train(features, targets)
}

// trainBaggingParallel 并行训练Bagging集成
func (pt *ParallelTrainer) trainBaggingParallel(ensemble *EnsemblePredictor, features [][]float64, targets []float64) error {
	numLearners := len(ensemble.BaseLearners)
	if numLearners == 0 {
		return fmt.Errorf("no base learners to train")
	}

	// 创建工作通道
	jobs := make(chan trainingJob, numLearners)
	results := make(chan trainingResult, numLearners)

	// 启动工作协程
	for w := 0; w < pt.maxWorkers; w++ {
		go pt.worker(jobs, results)
	}

	// 发送训练任务
	for i, learner := range ensemble.BaseLearners {
		// Bootstrap采样
		sampleFeatures, sampleTargets := ensemble.bootstrapSample(features, targets, len(features))

		jobs <- trainingJob{
			LearnerIndex: i,
			Learner:      learner,
			Features:     sampleFeatures,
			Targets:      sampleTargets,
		}
	}
	close(jobs)

	// 收集结果
	trainedLearners := make([]BaseLearner, numLearners)
	for i := 0; i < numLearners; i++ {
		result := <-results
		if result.Error != nil {
			return fmt.Errorf("failed to train learner %d: %w", result.LearnerIndex, result.Error)
		}
		trainedLearners[result.LearnerIndex] = result.TrainedLearner
	}

	// 更新集成模型
	ensemble.BaseLearners = trainedLearners
	ensemble.TrainingTime = time.Since(time.Now().Add(-ensemble.TrainingTime)) // 重置训练时间

	return nil
}

// trainingJob 训练任务
type trainingJob struct {
	LearnerIndex int
	Learner      BaseLearner
	Features     [][]float64
	Targets      []float64
}

// trainingResult 训练结果
type trainingResult struct {
	LearnerIndex   int
	TrainedLearner BaseLearner
	Error          error
}

// worker 工作协程
func (pt *ParallelTrainer) worker(jobs <-chan trainingJob, results chan<- trainingResult) {
	for job := range jobs {
		// 克隆学习器以避免并发修改
		learner := job.Learner.Clone()

		// 训练学习器
		err := learner.Train(job.Features, job.Targets)

		results <- trainingResult{
			LearnerIndex:   job.LearnerIndex,
			TrainedLearner: learner,
			Error:          err,
		}
	}
}

// ModelPretrainer 模型预训练器
type ModelPretrainer struct {
	cache    *ModelCache
	trainer  *ParallelTrainer
	baseData map[string]TrainingDataset // 预训练数据集
}

// TrainingDataset 训练数据集
type TrainingDataset struct {
	Features [][]float64
	Targets  []float64
	DataHash string
}

// NewModelPretrainer 创建模型预训练器
func NewModelPretrainer(cache *ModelCache, trainer *ParallelTrainer) *ModelPretrainer {
	return &ModelPretrainer{
		cache:    cache,
		trainer:  trainer,
		baseData: make(map[string]TrainingDataset),
	}
}

// PretrainEnsemble 预训练集成模型
func (mp *ModelPretrainer) PretrainEnsemble(configName string, config EnsembleConfig) error {
	factory := NewLearnerFactory()

	// 检查缓存中是否已有预训练模型
	cacheKey := fmt.Sprintf("pretrained_%s", configName)
	if _, exists := mp.cache.GetEnsemble(cacheKey); exists {
		// 对于预训练模型，我们认为它是有效的（简化处理）
		return nil // 模型仍然有效
	}

	// 创建新的集成模型
	ensemble, err := factory.CreateEnsemblePredictor(config.EnsembleType, config)
	if err != nil {
		return fmt.Errorf("failed to create ensemble: %w", err)
	}

	// 获取预训练数据
	dataset, exists := mp.baseData[configName]
	if !exists {
		// 生成合成数据用于预训练
		dataset = mp.generateSyntheticData()
		mp.baseData[configName] = dataset
	}

	// 并行训练模型
	err = mp.trainer.TrainEnsemble(ensemble, dataset.Features, dataset.Targets)
	if err != nil {
		return fmt.Errorf("failed to pretrain ensemble: %w", err)
	}

	// 缓存预训练模型
	mp.cache.PutEnsemble(cacheKey, ensemble, dataset.DataHash)

	return nil
}

// generateSyntheticData 生成合成训练数据
func (mp *ModelPretrainer) generateSyntheticData() TrainingDataset {
	// 生成1000个样本，每个样本5个特征
	numSamples := 1000
	numFeatures := 5

	features := make([][]float64, numSamples)
	targets := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		features[i] = make([]float64, numFeatures)

		// 生成随机特征
		for j := 0; j < numFeatures; j++ {
			features[i][j] = rand.Float64()*100 - 50 // [-50, 50]
		}

		// 生成目标值（基于特征的线性组合加上噪声）
		target := features[i][0]*2 + features[i][1]*-1.5 + features[i][2]*0.8 +
			features[i][3]*0.3 + features[i][4]*-0.7 + rand.Float64()*10 - 5

		targets[i] = target
	}

	dataStr := fmt.Sprintf("%v-%v", features, targets)
	hash := md5.Sum([]byte(dataStr))

	return TrainingDataset{
		Features: features,
		Targets:  targets,
		DataHash: fmt.Sprintf("%x", hash),
	}
}

// GetPretrainedModel 获取预训练模型
func (mp *ModelPretrainer) GetPretrainedModel(configName string) (*EnsemblePredictor, bool) {
	cacheKey := fmt.Sprintf("pretrained_%s", configName)
	return mp.cache.GetEnsemble(cacheKey)
}
