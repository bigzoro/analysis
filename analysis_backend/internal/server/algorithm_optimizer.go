package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// AlgorithmOptimizer 算法优化器
type AlgorithmOptimizer struct {
	db         *gorm.DB
	feedbackDB *gorm.DB
}

// NewAlgorithmOptimizer 创建算法优化器
func NewAlgorithmOptimizer(db *gorm.DB) *AlgorithmOptimizer {
	return &AlgorithmOptimizer{
		db:         db,
		feedbackDB: db, // 可以分离为不同的数据库
	}
}

// OptimizeWeights 基于用户反馈优化推荐算法权重
func (ao *AlgorithmOptimizer) OptimizeWeights(ctx context.Context, days int) (*OptimizedWeights, error) {
	// 获取指定天数内的反馈数据
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	var feedbacks []pdb.UserRecommendationFeedback
	if err := ao.feedbackDB.Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Find(&feedbacks).Error; err != nil {
		return nil, fmt.Errorf("获取反馈数据失败: %v", err)
	}

	if len(feedbacks) < 10 {
		return nil, fmt.Errorf("反馈数据不足，至少需要10条反馈数据")
	}

	// 计算因子表现
	factorPerformance := ao.calculateFactorPerformance(feedbacks)

	// 计算用户偏好权重
	userPreferences := ao.calculateUserPreferences(feedbacks)

	// 计算时间衰减权重（越新的反馈权重越大）
	timeWeights := ao.calculateTimeWeights(feedbacks)

	// 综合计算优化后的权重
	optimizedWeights := ao.combineWeights(factorPerformance, userPreferences, timeWeights)

	// 记录优化结果
	ao.recordOptimizationResult(optimizedWeights, len(feedbacks), days)

	return optimizedWeights, nil
}

// calculateFactorPerformance 计算各因子的实际表现
func (ao *AlgorithmOptimizer) calculateFactorPerformance(feedbacks []pdb.UserRecommendationFeedback) FactorPerformance {
	factorStats := make(map[string][]float64) // 因子 -> 评分列表

	// 获取每个反馈对应的推荐详情
	for _, feedback := range feedbacks {
		var rec pdb.CoinRecommendation
		if err := ao.db.First(&rec, feedback.RecommendationID).Error; err != nil {
			continue
		}

		// 计算实际收益率（简化版）
		actualReturn := ao.calculateActualReturn(rec.Symbol, rec.CreatedAt, time.Now())

		// 分析各因子对结果的贡献
		if feedback.Rating != nil && *feedback.Rating >= 1 && *feedback.Rating <= 5 {
			actualScore := ao.convertActualReturnToScore(actualReturn)

			// 市场表现因子
			marketWeight := ao.calculateMarketFactorWeight(rec, actualScore)
			factorStats["market"] = append(factorStats["market"], marketWeight)

			// 资金流因子
			flowWeight := ao.calculateFlowFactorWeight(rec, actualScore)
			factorStats["flow"] = append(factorStats["flow"], flowWeight)

			// 热度因子
			heatWeight := ao.calculateHeatFactorWeight(rec, actualScore)
			factorStats["heat"] = append(factorStats["heat"], heatWeight)

			// 事件因子
			eventWeight := ao.calculateEventFactorWeight(rec, actualScore)
			factorStats["event"] = append(factorStats["event"], eventWeight)

			// 情绪因子
			sentimentWeight := ao.calculateSentimentFactorWeight(rec, actualScore)
			factorStats["sentiment"] = append(factorStats["sentiment"], sentimentWeight)
		}
	}

	// 计算每个因子的平均表现得分
	result := FactorPerformance{}
	for factor, scores := range factorStats {
		if len(scores) > 0 {
			sum := 0.0
			for _, score := range scores {
				sum += score
			}
			avgScore := sum / float64(len(scores))

			switch factor {
			case "market":
				result.MarketScore = avgScore
			case "flow":
				result.FlowScore = avgScore
			case "heat":
				result.HeatScore = avgScore
			case "event":
				result.EventScore = avgScore
			case "sentiment":
				result.SentimentScore = avgScore
			}
		}
	}

	return result
}

// calculateUserPreferences 计算用户偏好权重
func (ao *AlgorithmOptimizer) calculateUserPreferences(feedbacks []pdb.UserRecommendationFeedback) UserPreferenceWeights {
	preferences := make(map[string]float64)

	// 统计不同反馈类型的分布
	actionCounts := make(map[string]int)

	for _, feedback := range feedbacks {
		actionCounts[feedback.Action]++

		// 根据反馈理由分析偏好
		if feedback.Reason != "" {
			// 简单的关键词分析（可以扩展为NLP）
			if ao.containsKeywords(feedback.Reason, []string{"技术", "指标", "RSI", "MACD"}) {
				preferences["technical"] += 1.0
			}
			if ao.containsKeywords(feedback.Reason, []string{"资金", "流入", "流出", "净流"}) {
				preferences["flow"] += 1.0
			}
			if ao.containsKeywords(feedback.Reason, []string{"新闻", "公告", "事件"}) {
				preferences["event"] += 1.0
			}
			if ao.containsKeywords(feedback.Reason, []string{"情绪", "社交", "Twitter"}) {
				preferences["sentiment"] += 1.0
			}
		}
	}

	// 归一化偏好权重
	total := 0.0
	for _, count := range preferences {
		total += count
	}

	if total > 0 {
		for key := range preferences {
			preferences[key] /= total
		}
	}

	return UserPreferenceWeights{
		TechnicalPreference: preferences["technical"],
		FlowPreference:      preferences["flow"],
		EventPreference:     preferences["event"],
		SentimentPreference: preferences["sentiment"],
	}
}

// calculateTimeWeights 计算时间权重（新反馈权重更高）
func (ao *AlgorithmOptimizer) calculateTimeWeights(feedbacks []pdb.UserRecommendationFeedback) TimeWeights {
	if len(feedbacks) == 0 {
		return TimeWeights{}
	}

	// 按时间排序
	sort.Slice(feedbacks, func(i, j int) bool {
		return feedbacks[i].CreatedAt.After(feedbacks[j].CreatedAt)
	})

	// 计算时间衰减权重
	now := time.Now()
	totalWeight := 0.0
	weights := make([]float64, len(feedbacks))

	for i, feedback := range feedbacks {
		// 指数衰减：越新的权重越大
		hoursAgo := now.Sub(feedback.CreatedAt).Hours()
		weight := math.Exp(-hoursAgo / 168.0) // 168小时 = 7天半衰期
		weights[i] = weight
		totalWeight += weight
	}

	// 归一化
	if totalWeight > 0 {
		for i := range weights {
			weights[i] /= totalWeight
		}
	}

	return TimeWeights{
		Weights:       weights,
		AverageWeight: totalWeight / float64(len(feedbacks)),
	}
}

// combineWeights 综合计算优化后的权重
func (ao *AlgorithmOptimizer) combineWeights(
	factorPerf FactorPerformance,
	userPrefs UserPreferenceWeights,
	timeWeights TimeWeights,
) *OptimizedWeights {

	// 基础权重（当前算法权重）
	baseWeights := map[string]float64{
		"market":    0.30,
		"flow":      0.25,
		"heat":      0.20,
		"event":     0.15,
		"sentiment": 0.10,
	}

	// 因子表现调整
	performanceAdjustment := map[string]float64{
		"market":    ao.adjustWeightByPerformance(factorPerf.MarketScore),
		"flow":      ao.adjustWeightByPerformance(factorPerf.FlowScore),
		"heat":      ao.adjustWeightByPerformance(factorPerf.HeatScore),
		"event":     ao.adjustWeightByPerformance(factorPerf.EventScore),
		"sentiment": ao.adjustWeightByPerformance(factorPerf.SentimentScore),
	}

	// 用户偏好调整
	preferenceAdjustment := map[string]float64{
		"market":    1.0, // 市场表现通常是基础
		"flow":      1.0 + userPrefs.FlowPreference*0.5,
		"heat":      1.0, // 热度相对固定
		"event":     1.0 + userPrefs.EventPreference*0.5,
		"sentiment": 1.0 + userPrefs.SentimentPreference*0.5,
	}

	// 计算最终权重
	finalWeights := make(map[string]float64)
	totalWeight := 0.0

	for factor := range baseWeights {
		adjustedWeight := baseWeights[factor] *
			performanceAdjustment[factor] *
			preferenceAdjustment[factor]

		finalWeights[factor] = adjustedWeight
		totalWeight += adjustedWeight
	}

	// 归一化
	if totalWeight > 0 {
		for factor := range finalWeights {
			finalWeights[factor] /= totalWeight
		}
	}

	return &OptimizedWeights{
		MarketWeight:      finalWeights["market"],
		FlowWeight:        finalWeights["flow"],
		HeatWeight:        finalWeights["heat"],
		EventWeight:       finalWeights["event"],
		SentimentWeight:   finalWeights["sentiment"],
		OptimizationScore: ao.calculateOptimizationScore(factorPerf, userPrefs),
		LastOptimized:     time.Now(),
		SampleSize:        len(timeWeights.Weights),
	}
}

// adjustWeightByPerformance 根据表现调整权重
func (ao *AlgorithmOptimizer) adjustWeightByPerformance(score float64) float64 {
	if score <= 0 {
		return 0.5 // 表现差的因子降低权重
	} else if score >= 1.0 {
		return 1.5 // 表现好的因子增加权重
	}
	return 0.5 + score // 线性调整
}

// calculateOptimizationScore 计算优化得分
func (ao *AlgorithmOptimizer) calculateOptimizationScore(factorPerf FactorPerformance, userPrefs UserPreferenceWeights) float64 {
	// 简化的优化得分计算
	avgFactorScore := (factorPerf.MarketScore + factorPerf.FlowScore + factorPerf.HeatScore +
		factorPerf.EventScore + factorPerf.SentimentScore) / 5.0

	preferenceAlignment := (userPrefs.TechnicalPreference + userPrefs.FlowPreference +
		userPrefs.EventPreference + userPrefs.SentimentPreference) / 4.0

	return (avgFactorScore + preferenceAlignment) / 2.0
}

// recordOptimizationResult 记录优化结果
func (ao *AlgorithmOptimizer) recordOptimizationResult(weights *OptimizedWeights, sampleSize, days int) {
	result := pdb.AlgorithmPerformance{
		AlgorithmVersion: fmt.Sprintf("optimized_v%s", time.Now().Format("20060102")),
		TestGroup:        "production",
		TimeRange:        fmt.Sprintf("%dd", days),
		SampleSize:       sampleSize,
		Metrics:          ao.weightsToJSON(weights),
		ImprovementRate:  weights.OptimizationScore,
		CreatedAt:        time.Now(),
	}

	if err := ao.db.Create(&result).Error; err != nil {
		log.Printf("记录优化结果失败: %v", err)
	}
}

// weightsToJSON 将权重转换为JSON
func (ao *AlgorithmOptimizer) weightsToJSON(weights *OptimizedWeights) []byte {
	data := map[string]interface{}{
		"market_weight":      weights.MarketWeight,
		"flow_weight":        weights.FlowWeight,
		"heat_weight":        weights.HeatWeight,
		"event_weight":       weights.EventWeight,
		"sentiment_weight":   weights.SentimentWeight,
		"optimization_score": weights.OptimizationScore,
		"sample_size":        weights.SampleSize,
		"last_optimized":     weights.LastOptimized,
	}

	jsonData, _ := json.Marshal(data)
	return jsonData
}

// 辅助函数
func (ao *AlgorithmOptimizer) calculateActualReturn(symbol string, startTime, endTime time.Time) float64 {
	// 简化的实际收益率计算（实际实现需要价格数据）
	// 这里应该查询历史价格数据计算实际收益率
	return 0.05 // 示例值
}

func (ao *AlgorithmOptimizer) convertActualReturnToScore(actualReturn float64) float64 {
	// 将实际收益率转换为评分 (0-1)
	if actualReturn < -0.1 {
		return 0.0
	} else if actualReturn > 0.2 {
		return 1.0
	}
	return (actualReturn + 0.1) / 0.3
}

func (ao *AlgorithmOptimizer) calculateMarketFactorWeight(rec pdb.CoinRecommendation, actualScore float64) float64 {
	// 简化的因子权重计算
	return actualScore * rec.MarketScore / 100.0
}

func (ao *AlgorithmOptimizer) calculateFlowFactorWeight(rec pdb.CoinRecommendation, actualScore float64) float64 {
	return actualScore * rec.FlowScore / 100.0
}

func (ao *AlgorithmOptimizer) calculateHeatFactorWeight(rec pdb.CoinRecommendation, actualScore float64) float64 {
	return actualScore * rec.HeatScore / 100.0
}

func (ao *AlgorithmOptimizer) calculateEventFactorWeight(rec pdb.CoinRecommendation, actualScore float64) float64 {
	return actualScore * rec.EventScore / 100.0
}

func (ao *AlgorithmOptimizer) calculateSentimentFactorWeight(rec pdb.CoinRecommendation, actualScore float64) float64 {
	return actualScore * rec.SentimentScore / 100.0
}

func (ao *AlgorithmOptimizer) containsKeywords(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// 数据结构定义
type FactorPerformance struct {
	MarketScore    float64
	FlowScore      float64
	HeatScore      float64
	EventScore     float64
	SentimentScore float64
}

type UserPreferenceWeights struct {
	TechnicalPreference float64
	FlowPreference      float64
	EventPreference     float64
	SentimentPreference float64
}

type TimeWeights struct {
	Weights       []float64
	AverageWeight float64
}

type OptimizedWeights struct {
	MarketWeight      float64   `json:"market_weight"`
	FlowWeight        float64   `json:"flow_weight"`
	HeatWeight        float64   `json:"heat_weight"`
	EventWeight       float64   `json:"event_weight"`
	SentimentWeight   float64   `json:"sentiment_weight"`
	OptimizationScore float64   `json:"optimization_score"`
	LastOptimized     time.Time `json:"last_optimized"`
	SampleSize        int       `json:"sample_size"`
}
