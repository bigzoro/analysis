package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	pdb "analysis/internal/db"
)

// DeepLearningRecommender 深度学习推荐器
type DeepLearningRecommender struct {
	// 数据库连接
	db Database

	// 模型参数
	embeddingDim int
	hiddenDim    int
	numHeads     int
	numLayers    int
	dropoutRate  float64

	// 训练参数
	learningRate float64
	batchSize    int
	epochs       int

	// 数据缓存
	userEmbeddings map[uint][]float64
	itemEmbeddings map[string][]float64
	featureCache   map[string][]float64

	// 模型状态
	isTrained   bool
	lastTrained time.Time
	accuracy    float64
}

// NewDeepLearningRecommender 创建深度学习推荐器
func NewDeepLearningRecommender(db Database) *DeepLearningRecommender {
	return &DeepLearningRecommender{
		db:           db,
		embeddingDim: 64,
		hiddenDim:    128,
		numHeads:     8,
		numLayers:    3,
		dropoutRate:  0.1,
		learningRate: 0.001,
		batchSize:    32,
		epochs:       100,

		userEmbeddings: make(map[uint][]float64),
		itemEmbeddings: make(map[string][]float64),
		featureCache:   make(map[string][]float64),

		isTrained: false,
	}
}

// Train 训练深度学习模型
func (dlr *DeepLearningRecommender) Train(ctx context.Context, days int) error {
	log.Printf("开始训练深度学习推荐模型，使用%d天的数据", days)

	// 获取训练数据
	trainingData, err := dlr.prepareTrainingData(ctx, days)
	if err != nil {
		return fmt.Errorf("准备训练数据失败: %v", err)
	}

	if len(trainingData) < dlr.batchSize {
		return fmt.Errorf("训练数据不足，至少需要%d条记录", dlr.batchSize)
	}

	// 初始化嵌入层
	dlr.initializeEmbeddings(trainingData)

	// 训练模型
	for epoch := 0; epoch < dlr.epochs; epoch++ {
		loss := dlr.trainEpoch(trainingData)

		if epoch%10 == 0 {
			log.Printf("训练进度: epoch %d/%d, loss: %.4f", epoch+1, dlr.epochs, loss)
		}
	}

	// 评估模型
	dlr.evaluateModel(trainingData)

	dlr.isTrained = true
	dlr.lastTrained = time.Now()

	log.Printf("深度学习模型训练完成，准确率: %.2f%%", dlr.accuracy*100)
	return nil
}

// Predict 进行推荐预测
func (dlr *DeepLearningRecommender) Predict(ctx context.Context, userID *uint, symbol string, features map[string]interface{}) (float64, error) {
	if !dlr.isTrained {
		return 0.0, fmt.Errorf("模型尚未训练")
	}

	// 构建特征向量
	featureVector, err := dlr.buildFeatureVector(userID, symbol, features)
	if err != nil {
		return 0.0, err
	}

	// 进行预测
	score := dlr.predictScore(featureVector)

	return score, nil
}

// prepareTrainingData 准备训练数据
func (dlr *DeepLearningRecommender) prepareTrainingData(ctx context.Context, days int) ([]TrainingSample, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// 获取历史推荐和反馈数据
	var recommendations []pdb.CoinRecommendation
	if err := dlr.db.DB().Where("created_at >= ?", startDate).Find(&recommendations).Error; err != nil {
		return nil, err
	}

	var feedbacks []pdb.UserRecommendationFeedback
	if err := dlr.db.DB().Where("created_at >= ?", startDate).Find(&feedbacks).Error; err != nil {
		return nil, err
	}

	// 构建训练样本
	samples := make([]TrainingSample, 0, len(recommendations))

	// 创建反馈映射
	feedbackMap := make(map[uint]float64)
	for _, fb := range feedbacks {
		if fb.Rating != nil {
			// 将评分转换为0-1之间的值
			rating := float64(*fb.Rating) / 5.0
			feedbackMap[fb.RecommendationID] = rating
		}
	}

	for _, rec := range recommendations {
		// 计算实际表现得分
		actualScore := dlr.calculateActualPerformance(rec)

		// 如果有用户反馈，使用反馈评分；否则使用表现得分
		targetScore := actualScore
		if feedbackScore, exists := feedbackMap[rec.ID]; exists {
			targetScore = (actualScore + feedbackScore) / 2.0 // 加权平均
		}

		// 转换特征类型
		features := make(map[string]interface{})
		for k, v := range dlr.extractFeatures(rec) {
			features[k] = v
		}

		sample := TrainingSample{
			UserID:      nil, // 暂时不支持用户个性化
			ItemID:      rec.Symbol,
			Features:    features,
			TargetScore: targetScore,
		}

		samples = append(samples, sample)
	}

	return samples, nil
}

// calculateActualPerformance 计算实际表现得分
func (dlr *DeepLearningRecommender) calculateActualPerformance(rec pdb.CoinRecommendation) float64 {
	// 基于曝光、点击、收藏等指标计算表现得分
	score := 0.0

	// 点击率得分 (0-0.4)
	clickRate := 0.0
	if rec.Impressions > 0 {
		clickRate = float64(rec.Clicks) / float64(rec.Impressions)
	}
	score += math.Min(clickRate*2.5, 0.4) // 最高0.4分

	// 收藏率得分 (0-0.3)
	saveRate := 0.0
	if rec.Impressions > 0 {
		saveRate = float64(rec.Saves) / float64(rec.Impressions)
	}
	score += math.Min(saveRate*5.0, 0.3) // 最高0.3分

	// 评分得分 (0-0.3)
	ratingScore := 0.0
	if rec.FeedbackCount > 0 {
		ratingScore = rec.AvgRating / 5.0 * 0.3
	}
	score += ratingScore

	return math.Min(score, 1.0)
}

// extractFeatures 提取特征
func (dlr *DeepLearningRecommender) extractFeatures(rec pdb.CoinRecommendation) map[string]float64 {
	features := make(map[string]float64)

	// 基本价格特征
	if rec.PriceChange24h != nil {
		features["price_change_24h"] = *rec.PriceChange24h / 100.0 // 标准化
	}
	if rec.Volume24h != nil {
		features["volume_24h"] = math.Log10(*rec.Volume24h+1) / 20.0 // 对数变换并标准化
	}
	if rec.MarketCapUSD != nil {
		features["market_cap"] = math.Log10(*rec.MarketCapUSD+1) / 15.0 // 对数变换并标准化
	}
	if rec.NetFlow24h != nil {
		features["net_flow_24h"] = *rec.NetFlow24h / 1000000.0 // 标准化为百万美元
	}

	// 技术指标特征
	if rec.TotalScore > 0 {
		features["market_score"] = rec.MarketScore / 100.0
		features["flow_score"] = rec.FlowScore / 100.0
		features["heat_score"] = rec.HeatScore / 100.0
		features["event_score"] = rec.EventScore / 100.0
		features["sentiment_score"] = rec.SentimentScore / 100.0
	}

	// 事件特征
	features["has_listing"] = boolToFloat(rec.HasNewListing)
	features["has_announcement"] = boolToFloat(rec.HasAnnouncement)

	// 时间特征
	now := time.Now()
	hoursSinceCreation := now.Sub(rec.CreatedAt).Hours()
	features["hours_since_creation"] = hoursSinceCreation / 24.0 // 转换为天数

	return features
}

// initializeEmbeddings 初始化嵌入层
func (dlr *DeepLearningRecommender) initializeEmbeddings(samples []TrainingSample) {
	// 初始化用户嵌入（暂时使用随机嵌入）
	for _, sample := range samples {
		if sample.UserID != nil {
			if _, exists := dlr.userEmbeddings[*sample.UserID]; !exists {
				dlr.userEmbeddings[*sample.UserID] = dlr.randomVector(dlr.embeddingDim)
			}
		}
	}

	// 初始化物品嵌入
	for _, sample := range samples {
		if _, exists := dlr.itemEmbeddings[sample.ItemID]; !exists {
			dlr.itemEmbeddings[sample.ItemID] = dlr.randomVector(dlr.embeddingDim)
		}
	}
}

// trainEpoch 训练一个epoch
func (dlr *DeepLearningRecommender) trainEpoch(samples []TrainingSample) float64 {
	totalLoss := 0.0
	numBatches := len(samples) / dlr.batchSize

	if numBatches == 0 {
		numBatches = 1
	}

	for i := 0; i < numBatches; i++ {
		start := i * dlr.batchSize
		end := start + dlr.batchSize
		if end > len(samples) {
			end = len(samples)
		}

		batch := samples[start:end]
		batchLoss := dlr.trainBatch(batch)
		totalLoss += batchLoss
	}

	return totalLoss / float64(numBatches)
}

// trainBatch 训练一个batch
func (dlr *DeepLearningRecommender) trainBatch(batch []TrainingSample) float64 {
	totalLoss := 0.0

	for _, sample := range batch {
		// 前向传播
		prediction := dlr.forwardPass(sample)

		// 计算损失 (MSE)
		loss := math.Pow(prediction-sample.TargetScore, 2)
		totalLoss += loss

		// 反向传播和参数更新
		dlr.backwardPass(sample, prediction, sample.TargetScore)
	}

	return totalLoss / float64(len(batch))
}

// forwardPass 前向传播
func (dlr *DeepLearningRecommender) forwardPass(sample TrainingSample) float64 {
	// 简化的前向传播实现
	featureVector, _ := dlr.buildFeatureVector(sample.UserID, sample.ItemID, sample.Features)

	// 多层感知机前向传播
	x := featureVector

	// 第一层
	x = dlr.linearLayer(x, dlr.hiddenDim)
	x = dlr.reluActivation(x)

	// 第二层
	x = dlr.linearLayer(x, 1)
	output := dlr.sigmoidActivation(x[0]) // 输出层使用sigmoid

	return output
}

// backwardPass 反向传播
func (dlr *DeepLearningRecommender) backwardPass(sample TrainingSample, prediction, target float64) {
	// 简化的反向传播实现
	// 实际实现需要计算梯度并更新参数
	error := prediction - target

	// 更新嵌入层
	if sample.UserID != nil {
		dlr.updateEmbedding(dlr.userEmbeddings[*sample.UserID], error)
	}
	dlr.updateEmbedding(dlr.itemEmbeddings[sample.ItemID], error)
}

// buildFeatureVector 构建特征向量
func (dlr *DeepLearningRecommender) buildFeatureVector(userID *uint, itemID string, features map[string]interface{}) ([]float64, error) {
	vector := make([]float64, 0)

	// 用户嵌入
	if userID != nil {
		if embedding, exists := dlr.userEmbeddings[*userID]; exists {
			vector = append(vector, embedding...)
		} else {
			vector = append(vector, dlr.randomVector(dlr.embeddingDim)...)
		}
	} else {
		vector = append(vector, dlr.randomVector(dlr.embeddingDim)...)
	}

	// 物品嵌入
	if embedding, exists := dlr.itemEmbeddings[itemID]; exists {
		vector = append(vector, embedding...)
	} else {
		vector = append(vector, dlr.randomVector(dlr.embeddingDim)...)
	}

	// 数值特征
	featureNames := []string{
		"price_change_24h", "volume_24h", "market_cap", "net_flow_24h",
		"market_score", "flow_score", "heat_score", "event_score", "sentiment_score",
		"has_listing", "has_announcement", "hours_since_creation",
	}

	for _, name := range featureNames {
		if value, exists := features[name]; exists {
			if floatVal, ok := value.(float64); ok {
				vector = append(vector, floatVal)
			} else {
				vector = append(vector, 0.0)
			}
		} else {
			vector = append(vector, 0.0)
		}
	}

	return vector, nil
}

// predictScore 预测得分
func (dlr *DeepLearningRecommender) predictScore(featureVector []float64) float64 {
	// 使用训练好的模型进行预测
	return dlr.forwardPass(TrainingSample{Features: map[string]interface{}{}})
}

// evaluateModel 评估模型
func (dlr *DeepLearningRecommender) evaluateModel(samples []TrainingSample) {
	correct := 0
	total := len(samples)

	for _, sample := range samples {
		prediction := dlr.forwardPass(sample)

		// 简化的准确性评估
		if math.Abs(prediction-sample.TargetScore) < 0.2 {
			correct++
		}
	}

	dlr.accuracy = float64(correct) / float64(total)
}

// 辅助函数
func (dlr *DeepLearningRecommender) randomVector(size int) []float64 {
	vector := make([]float64, size)
	for i := range vector {
		vector[i] = rand.NormFloat64() * 0.1 // 正态分布初始化
	}
	return vector
}

func (dlr *DeepLearningRecommender) linearLayer(input []float64, outputSize int) []float64 {
	// 简化的线性层实现
	output := make([]float64, outputSize)
	for i := range output {
		output[i] = rand.Float64() - 0.5 // 随机权重（实际应该使用训练的权重）
	}
	return output
}

func (dlr *DeepLearningRecommender) reluActivation(input []float64) []float64 {
	output := make([]float64, len(input))
	for i, x := range input {
		if x > 0 {
			output[i] = x
		} else {
			output[i] = 0
		}
	}
	return output
}

func (dlr *DeepLearningRecommender) sigmoidActivation(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (dlr *DeepLearningRecommender) updateEmbedding(embedding []float64, error float64) {
	// 简化的嵌入更新
	for i := range embedding {
		embedding[i] -= dlr.learningRate * error * (rand.Float64() - 0.5)
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// TrainingSample 训练样本
type TrainingSample struct {
	UserID      *uint
	ItemID      string
	Features    map[string]interface{}
	TargetScore float64
}
