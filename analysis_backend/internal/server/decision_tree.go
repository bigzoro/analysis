package server

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

// DecisionTree 决策树学习器
type DecisionTree struct {
	Root              *TreeNode
	MaxDepth          int
	MinSamplesSplit   int
	MinSamplesLeaf    int
	featureImportance []float64 // 特征重要性得分
}

// Serialize 序列化决策树
func (dt *DecisionTree) Serialize() ([]byte, error) {
	// 简化的序列化实现
	// TODO: 实现完整的树结构序列化
	data := map[string]interface{}{
		"maxDepth":          dt.MaxDepth,
		"minSamplesSplit":   dt.MinSamplesSplit,
		"minSamplesLeaf":    dt.MinSamplesLeaf,
		"featureImportance": dt.featureImportance,
		// 暂时不序列化Root，训练时需要重新构建
	}

	return json.Marshal(data)
}

// Deserialize 反序列化决策树
func (dt *DecisionTree) Deserialize(data []byte) error {
	var treeData map[string]interface{}
	if err := json.Unmarshal(data, &treeData); err != nil {
		return err
	}

	if maxDepth, ok := treeData["maxDepth"].(float64); ok {
		dt.MaxDepth = int(maxDepth)
	}

	if minSamplesSplit, ok := treeData["minSamplesSplit"].(float64); ok {
		dt.MinSamplesSplit = int(minSamplesSplit)
	}

	if minSamplesLeaf, ok := treeData["minSamplesLeaf"].(float64); ok {
		dt.MinSamplesLeaf = int(minSamplesLeaf)
	}

	if importance, ok := treeData["featureImportance"].([]interface{}); ok {
		dt.featureImportance = make([]float64, len(importance))
		for i, v := range importance {
			if val, ok := v.(float64); ok {
				dt.featureImportance[i] = val
			}
		}
	}

	// Root需要重新训练时构建
	dt.Root = nil

	return nil
}

// TreeNode 决策树节点
type TreeNode struct {
	FeatureIndex int
	Threshold    float64
	LeftChild    *TreeNode
	RightChild   *TreeNode
	Value        float64 // 叶子节点的值
	IsLeaf       bool
	Samples      int   // 节点包含的样本数
	ClassCounts  []int // 各类别样本数量，用于计算基尼不纯度
}

// NewDecisionTree 创建决策树学习器 - Phase 10优化：大幅增加正则化防止过拟合
func NewDecisionTree() *DecisionTree {
	return &DecisionTree{
		MaxDepth:        5,  // Phase 10: 从10降低到5，防止过深生长
		MinSamplesSplit: 10, // Phase 10: 从2增加到10，减少分割
		MinSamplesLeaf:  5,  // Phase 10: 从1增加到5，确保叶子节点有足够样本
	}
}

// Clone 克隆学习器
func (dt *DecisionTree) Clone() BaseLearner {
	return NewDecisionTree()
}

// DeepCopy 创建决策树的深拷贝
func (dt *DecisionTree) DeepCopy() *DecisionTree {
	if dt == nil {
		return nil
	}

	treeCopy := &DecisionTree{
		MaxDepth:        dt.MaxDepth,
		MinSamplesSplit: dt.MinSamplesSplit,
		MinSamplesLeaf:  dt.MinSamplesLeaf,
	}

	// 深拷贝特征重要性
	if dt.featureImportance != nil {
		treeCopy.featureImportance = make([]float64, len(dt.featureImportance))
		copy(treeCopy.featureImportance, dt.featureImportance)
	}

	// 递归深拷贝树结构
	if dt.Root != nil {
		treeCopy.Root = dt.deepCopyTreeNode(dt.Root)
	}

	return treeCopy
}

// deepCopyTreeNode 递归深拷贝树节点
func (dt *DecisionTree) deepCopyTreeNode(node *TreeNode) *TreeNode {
	if node == nil {
		return nil
	}

	nodeCopy := &TreeNode{
		FeatureIndex: node.FeatureIndex,
		Threshold:    node.Threshold,
		Value:        node.Value,
		IsLeaf:       node.IsLeaf,
		Samples:      node.Samples,
		ClassCounts:  make([]int, len(node.ClassCounts)),
	}

	// 复制类别计数
	copy(nodeCopy.ClassCounts, node.ClassCounts)

	// 递归拷贝子节点
	nodeCopy.LeftChild = dt.deepCopyTreeNode(node.LeftChild)
	nodeCopy.RightChild = dt.deepCopyTreeNode(node.RightChild)

	return nodeCopy
}

// GetFeatureImportance 实现 BaseLearner 接口
func (dt *DecisionTree) GetFeatureImportance() []float64 {
	if dt.featureImportance != nil && len(dt.featureImportance) > 0 {
		return dt.featureImportance
	}

	// 如果还没有计算过，计算特征重要性
	return dt.calculateFeatureImportance()
}

// ===== 阶段一优化：改进特征重要性计算 =====
func (dt *DecisionTree) calculateFeatureImportance() []float64 {
	if dt.Root == nil {
		return []float64{}
	}

	// 初始化特征重要性数组
	featureImportance := make([]float64, dt.getFeatureCount())

	// 从根节点开始递归计算 - 改进版
	dt.calculateNodeImportanceV2(dt.Root, featureImportance)

	// 增强的归一化和过滤
	totalImportance := 0.0
	validFeatures := 0
	for _, imp := range featureImportance {
		if imp > 0 {
			totalImportance += imp
			validFeatures++
		}
	}

	// 如果有效特征太少，使用均匀分布，但仍然返回所有特征
	if validFeatures < 2 { // 降低阈值，从5降到2
		log.Printf("[FEATURE_IMPORTANCE_V2] 有效特征过少 (%d), 使用均匀分布", validFeatures)
		uniformWeight := 1.0 / float64(len(featureImportance))
		for i := range featureImportance {
			featureImportance[i] = uniformWeight
		}
		// 保存计算结果
		dt.featureImportance = featureImportance
		return featureImportance
	}

	// 归一化重要性得分
	if totalImportance > 0 {
		for i := range featureImportance {
			if featureImportance[i] > 0 {
				featureImportance[i] /= totalImportance
			} else {
				featureImportance[i] = 0.01 // 给无效特征最小权重
			}
		}
	}

	log.Printf("[FEATURE_IMPORTANCE_V2] 完成特征重要性计算，有效特征: %d/%d",
		validFeatures, len(featureImportance))

	// 保存计算结果
	dt.featureImportance = featureImportance

	return featureImportance
}

// ===== 阶段一优化：改进的节点重要性计算 =====
func (dt *DecisionTree) calculateNodeImportanceV2(node *TreeNode, importance []float64) float64 {
	if node == nil {
		return 0.0
	}

	if node.IsLeaf {
		// 叶子节点返回其基尼不纯度
		return dt.calculateGiniImpurity(node.ClassCounts)
	}

	// 计算该节点的基尼不纯度
	nodeImpurity := dt.calculateGiniImpurity(node.ClassCounts)
	totalSamples := float64(node.Samples)

	// 递归计算左右子节点的基尼不纯度
	leftImpurity := dt.calculateNodeImportanceV2(node.LeftChild, importance)
	rightImpurity := dt.calculateNodeImportanceV2(node.RightChild, importance)

	// 计算信息增益
	leftSamples := float64(node.LeftChild.Samples)
	rightSamples := float64(node.RightChild.Samples)

	// 信息增益 = 父节点不纯度 - 加权子节点不纯度
	informationGain := nodeImpurity - (leftSamples/totalSamples)*leftImpurity - (rightSamples/totalSamples)*rightImpurity

	if node.FeatureIndex >= 0 && node.FeatureIndex < len(importance) && informationGain > 0 {
		// 使用信息增益作为特征重要性，并乘以样本数进行加权
		featureImportance := informationGain * totalSamples
		importance[node.FeatureIndex] += featureImportance

		// 调试日志
		if featureImportance > 1.0 { // 只记录重要特征
			//log.Printf("[NODE_IMPORTANCE_V2] 特征%d: 信息增益=%.4f, 重要性=%.4f, 样本=%d",
			//	node.FeatureIndex, informationGain, featureImportance, int(totalSamples))
		}
	}

	return nodeImpurity
}

// calculateClassCounts 计算各类别样本数量
func (dt *DecisionTree) calculateClassCounts(targets []float64, indices []int) []int {
	classMap := make(map[float64]int)
	for _, idx := range indices {
		class := targets[idx]
		classMap[class]++
	}

	// 为三分类问题创建计数数组（假设类别值为-1, 0, 1）
	// 可以根据实际类别动态调整
	counts := make([]int, 3)
	for class, count := range classMap {
		if class == -1.0 {
			counts[0] = count
		} else if class == 0.0 {
			counts[1] = count
		} else if class == 1.0 {
			counts[2] = count
		}
	}

	return counts
}

// calculateGiniImpurity 计算基尼不纯度
func (dt *DecisionTree) calculateGiniImpurity(classCounts []int) float64 {
	if len(classCounts) == 0 {
		return 0.0
	}

	totalSamples := 0
	for _, count := range classCounts {
		totalSamples += count
	}

	if totalSamples == 0 {
		return 0.0
	}

	gini := 1.0
	for _, count := range classCounts {
		prob := float64(count) / float64(totalSamples)
		gini -= prob * prob
	}

	return gini
}

// calculateNodeImportance 递归计算节点的特征重要性 (原有版本)
func (dt *DecisionTree) calculateNodeImportance(node *TreeNode, importance []float64) float64 {
	if node == nil {
		return 0.0
	}

	if node.IsLeaf {
		// 叶子节点没有贡献信息增益
		return float64(node.Samples)
	}

	// 计算该节点的总样本数
	totalSamples := float64(node.Samples)

	// 计算左子树的样本数比例
	leftSamples := dt.calculateNodeImportance(node.LeftChild, importance)
	rightSamples := dt.calculateNodeImportance(node.RightChild, importance)

	// 计算信息增益（Gini重要性）
	// 对于分类树，特征重要性 = 该特征在所有节点上的加权信息增益
	if node.FeatureIndex >= 0 && node.FeatureIndex < len(importance) {
		// 使用加权基尼重要性
		giniImportance := totalSamples - (leftSamples + rightSamples)
		importance[node.FeatureIndex] += giniImportance
	}

	return totalSamples
}

// getFeatureCount 获取特征数量（需要从训练数据推断）
func (dt *DecisionTree) getFeatureCount() int {
	// 遍历整棵树找到最大的特征索引
	return dt.findMaxFeatureIndex(dt.Root) + 1
}

// findMaxFeatureIndex 找到树中最大的特征索引
func (dt *DecisionTree) findMaxFeatureIndex(node *TreeNode) int {
	if node == nil {
		return -1
	}

	maxIndex := node.FeatureIndex

	if node.LeftChild != nil {
		leftMax := dt.findMaxFeatureIndex(node.LeftChild)
		if leftMax > maxIndex {
			maxIndex = leftMax
		}
	}

	if node.RightChild != nil {
		rightMax := dt.findMaxFeatureIndex(node.RightChild)
		if rightMax > maxIndex {
			maxIndex = rightMax
		}
	}

	return maxIndex
}

// Train 训练决策树
func (dt *DecisionTree) Train(features [][]float64, targets []float64) error {
	if len(features) == 0 || len(targets) == 0 {
		return fmt.Errorf("empty training data")
	}

	if len(features) != len(targets) {
		return fmt.Errorf("features and targets length mismatch: %d vs %d", len(features), len(targets))
	}

	// 添加超时控制，防止训练过程卡住
	done := make(chan error, 1)
	go func() {
		done <- dt.trainInternal(features, targets)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Minute): // 5分钟超时
		log.Printf("[WARN_DT] 决策树训练超时，取消训练")
		return fmt.Errorf("决策树训练超时")
	}
}

// trainInternal 内部训练方法
func (dt *DecisionTree) trainInternal(features [][]float64, targets []float64) error {

	// 增强数据验证和清理
	validFeatures := make([][]float64, 0, len(features))
	validTargets := make([]float64, 0, len(targets))

	numFeatures := len(features[0])

	for i, feature := range features {
		// 检查特征向量长度一致性
		if len(feature) != numFeatures {
			log.Printf("[WARN_DT] 跳过特征向量长度不一致的样本 %d: 期望 %d, 实际 %d", i, numFeatures, len(feature))
			continue
		}

		// 检查特征值有效性
		isValid := true
		for j, val := range feature {
			if math.IsNaN(val) || math.IsInf(val, 0) {
				log.Printf("[WARN_DT] 跳过无效特征值的样本 %d[%d]: %f", i, j, val)
				isValid = false
				break
			}
		}

		// 检查目标值有效性
		if math.IsNaN(targets[i]) || math.IsInf(targets[i], 0) {
			log.Printf("[WARN_DT] 跳过无效目标值的样本 %d: %f", i, targets[i])
			isValid = false
		}

		if isValid {
			validFeatures = append(validFeatures, feature)
			validTargets = append(validTargets, targets[i])
		}
	}

	log.Printf("[INFO_DT] 数据清理完成: %d -> %d 样本", len(features), len(validFeatures))

	if len(validFeatures) == 0 {
		return fmt.Errorf("所有训练样本都被过滤，无效数据过多")
	}

	if len(validFeatures) < 2 {
		return fmt.Errorf("有效训练样本太少: %d，需要至少2个样本", len(validFeatures))
	}

	// 使用清理后的数据
	features = validFeatures
	targets = validTargets

	indices := make([]int, len(features))
	for i := range indices {
		indices[i] = i
	}

	log.Printf("[DEBUG_DT] 开始构建决策树，样本数: %d, 特征数: %d", len(features), len(features[0]))

	startTime := time.Now()
	dt.Root = dt.buildTreeWithProgress(features, targets, indices, 0, 1)
	buildDuration := time.Since(startTime)

	log.Printf("[DEBUG_DT] 决策树构建完成，耗时: %.2fs", buildDuration.Seconds())

	// 计算特征重要性
	if len(features) > 0 {
		featureStart := time.Now()
		dt.featureImportance = dt.calculateFeatureImportance()
		featureDuration := time.Since(featureStart)
		log.Printf("[DEBUG_DT] 特征重要性计算完成，耗时: %.2fs", featureDuration.Seconds())
	}

	return nil
}

// buildTree 递归构建决策树
func (dt *DecisionTree) buildTree(features [][]float64, targets []float64, indices []int, depth int) *TreeNode {
	node := &TreeNode{
		Samples:     len(indices),
		ClassCounts: dt.calculateClassCounts(targets, indices),
	}

	// 检查停止条件
	if dt.shouldStop(features, targets, indices, depth) {
		node.IsLeaf = true
		node.Value = dt.calculateLeafValue(targets, indices)
		return node
	}

	// 找到最佳分割
	bestSplit := dt.findBestSplit(features, targets, indices)
	if bestSplit == nil {
		// 无法分割，创建叶子节点
		node.IsLeaf = true
		node.Value = dt.calculateLeafValue(targets, indices)
		return node
	}

	// 创建内部节点
	node.FeatureIndex = bestSplit.FeatureIndex
	node.Threshold = bestSplit.Threshold

	// 分割数据
	leftIndices, rightIndices := dt.splitData(features, indices, bestSplit.FeatureIndex, bestSplit.Threshold)

	// 递归构建子树
	node.LeftChild = dt.buildTree(features, targets, leftIndices, depth+1)
	node.RightChild = dt.buildTree(features, targets, rightIndices, depth+1)

	return node
}

// buildTreeWithProgress 带进度监控的树构建
func (dt *DecisionTree) buildTreeWithProgress(features [][]float64, targets []float64, indices []int, depth int, treeIndex int) *TreeNode {
	if depth == 0 {
		log.Printf("[DEBUG_DT] 开始构建第%d棵树的根节点，样本数: %d", treeIndex, len(indices))
	}

	node := &TreeNode{
		Samples:     len(indices),
		ClassCounts: dt.calculateClassCounts(targets, indices),
	}

	// 检查停止条件
	if dt.shouldStop(features, targets, indices, depth) {
		node.IsLeaf = true
		node.Value = dt.calculateLeafValue(targets, indices)
		return node
	}

	// 找到最佳分割
	bestSplit := dt.findBestSplit(features, targets, indices)
	if bestSplit == nil {
		// 无法分割，创建叶子节点
		node.IsLeaf = true
		node.Value = dt.calculateLeafValue(targets, indices)
		return node
	}

	// 创建内部节点
	node.FeatureIndex = bestSplit.FeatureIndex
	node.Threshold = bestSplit.Threshold

	// 分割数据
	leftIndices, rightIndices := dt.splitData(features, indices, bestSplit.FeatureIndex, bestSplit.Threshold)

	// 递归构建子树
	node.LeftChild = dt.buildTreeWithProgress(features, targets, leftIndices, depth+1, treeIndex)
	node.RightChild = dt.buildTreeWithProgress(features, targets, rightIndices, depth+1, treeIndex)

	return node
}

// shouldStop 检查是否应该停止分割
func (dt *DecisionTree) shouldStop(features [][]float64, targets []float64, indices []int, depth int) bool {
	// 达到最大深度
	if depth >= dt.MaxDepth {
		return true
	}

	// 样本数太少
	if len(indices) == 0 || len(indices) < dt.MinSamplesSplit {
		return true
	}

	// 检查是否所有样本的目标值相同
	firstTarget := targets[indices[0]]
	allSame := true
	for _, idx := range indices[1:] {
		if targets[idx] != firstTarget {
			allSame = false
			break
		}
	}
	return allSame
}

// SplitInfo 分割信息
type SplitInfo struct {
	FeatureIndex int
	Threshold    float64
	Gain         float64
	LeftIndices  []int
	RightIndices []int
}

// findBestSplit 找到最佳分割（优化版本）
func (dt *DecisionTree) findBestSplit(features [][]float64, targets []float64, indices []int) *SplitInfo {
	if len(indices) < 2 {
		return nil
	}

	numFeatures := len(features[0])
	bestSplit := &SplitInfo{Gain: -math.MaxFloat64}

	// 优化：限制每个特征尝试的分割点数量
	maxCandidates := 50 // 最多尝试50个候选分割点
	if len(indices) < maxCandidates {
		maxCandidates = len(indices) - 1
	}

	for featureIdx := 0; featureIdx < numFeatures; featureIdx++ {
		// 获取该特征的所有值并排序
		values := make([]float64, len(indices))
		for i, idx := range indices {
			values[i] = features[idx][featureIdx]
		}
		sort.Float64s(values)

		// 优化：使用均匀采样选择候选分割点，而不是尝试所有点
		step := len(values) / maxCandidates
		if step < 1 {
			step = 1
		}

		for i := 0; i < len(values)-1; i += step {
			threshold := (values[i] + values[i+1]) / 2

			leftIndices, rightIndices := dt.splitData(features, indices, featureIdx, threshold)

			if len(leftIndices) < dt.MinSamplesLeaf || len(rightIndices) < dt.MinSamplesLeaf {
				continue
			}

			// 计算信息增益（使用方差减少）
			gain := dt.calculateVarianceReduction(targets, indices, leftIndices, rightIndices)

			if gain > bestSplit.Gain {
				bestSplit = &SplitInfo{
					FeatureIndex: featureIdx,
					Threshold:    threshold,
					Gain:         gain,
					LeftIndices:  leftIndices,
					RightIndices: rightIndices,
				}
			}
		}

		// 如果采样没有找到好的分割，尝试最佳和最差值之间的中点
		if bestSplit.Gain == -math.MaxFloat64 && len(values) >= 2 {
			midThreshold := (values[0] + values[len(values)-1]) / 2
			leftIndices, rightIndices := dt.splitData(features, indices, featureIdx, midThreshold)

			if len(leftIndices) >= dt.MinSamplesLeaf && len(rightIndices) >= dt.MinSamplesLeaf {
				gain := dt.calculateVarianceReduction(targets, indices, leftIndices, rightIndices)
				if gain > bestSplit.Gain {
					bestSplit = &SplitInfo{
						FeatureIndex: featureIdx,
						Threshold:    midThreshold,
						Gain:         gain,
						LeftIndices:  leftIndices,
						RightIndices: rightIndices,
					}
				}
			}
		}
	}

	if bestSplit.Gain == -math.MaxFloat64 {
		return nil
	}

	return bestSplit
}

// splitData 根据特征和阈值分割数据
func (dt *DecisionTree) splitData(features [][]float64, indices []int, featureIdx int, threshold float64) ([]int, []int) {
	leftIndices := make([]int, 0)
	rightIndices := make([]int, 0)

	for _, idx := range indices {
		if features[idx][featureIdx] <= threshold {
			leftIndices = append(leftIndices, idx)
		} else {
			rightIndices = append(rightIndices, idx)
		}
	}

	return leftIndices, rightIndices
}

// calculateVarianceReduction 计算方差减少（信息增益）
func (dt *DecisionTree) calculateVarianceReduction(targets []float64, parentIndices, leftIndices, rightIndices []int) float64 {
	parentVariance := dt.calculateVariance(targets, parentIndices)
	leftVariance := dt.calculateVariance(targets, leftIndices)
	rightVariance := dt.calculateVariance(targets, rightIndices)

	totalSamples := float64(len(parentIndices))
	leftWeight := float64(len(leftIndices)) / totalSamples
	rightWeight := float64(len(rightIndices)) / totalSamples

	return parentVariance - (leftWeight*leftVariance + rightWeight*rightVariance)
}

// calculateVariance 计算方差
func (dt *DecisionTree) calculateVariance(targets []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0
	}

	sum := 0.0
	sumSq := 0.0

	for _, idx := range indices {
		value := targets[idx]
		sum += value
		sumSq += value * value
	}

	mean := sum / float64(len(indices))
	variance := (sumSq / float64(len(indices))) - (mean * mean)

	return math.Max(0, variance) // 确保非负
}

// calculateLeafValue 计算叶子节点的值（平均值）
func (dt *DecisionTree) calculateLeafValue(targets []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0
	}

	sum := 0.0
	for _, idx := range indices {
		sum += targets[idx]
	}

	return sum / float64(len(indices))
}

// Predict 进行预测
func (dt *DecisionTree) Predict(features []float64) (float64, error) {
	if dt.Root == nil {
		return 0, fmt.Errorf("model not trained")
	}

	// 检查特征数量是否合理
	if len(features) == 0 {
		return 0, fmt.Errorf("no features provided")
	}

	// 检查特征数量是否匹配
	if dt.Root != nil && dt.Root.FeatureIndex >= len(features) {
		log.Printf("[DECISION_TREE] 特征数量不匹配: 期望至少%d个特征, 实际%d个，返回默认值0", dt.Root.FeatureIndex+1, len(features))
		return 0.0, nil // 返回默认值而不是错误
	}

	return dt.predictRecursive(dt.Root, features), nil
}

// predictRecursive 递归预测
func (dt *DecisionTree) predictRecursive(node *TreeNode, features []float64) float64 {
	if node.IsLeaf {
		return node.Value
	}

	// 边界检查：确保特征索引有效
	if node.FeatureIndex < 0 || node.FeatureIndex >= len(features) {
		log.Printf("[DECISION_TREE] 特征索引越界: index=%d, features_len=%d，使用默认值", node.FeatureIndex, len(features))
		// 特征索引越界时，返回默认值0.0
		return 0.0
	}

	if features[node.FeatureIndex] <= node.Threshold {
		if node.LeftChild != nil {
			return dt.predictRecursive(node.LeftChild, features)
		}
	} else {
		if node.RightChild != nil {
			return dt.predictRecursive(node.RightChild, features)
		}
	}

	// 无法继续遍历，返回当前节点的值
	return node.Value
}

// calculateImportanceRecursive 递归计算特征重要性
func (dt *DecisionTree) calculateImportanceRecursive(node *TreeNode, features [][]float64, targets []float64, indices []int, importance []float64) float64 {
	if node == nil || node.IsLeaf {
		// 计算叶子节点的方差
		if len(indices) == 0 {
			return 0
		}

		sum := 0.0
		for _, idx := range indices {
			sum += targets[idx]
		}
		mean := sum / float64(len(indices))

		variance := 0.0
		for _, idx := range indices {
			diff := targets[idx] - mean
			variance += diff * diff
		}
		return variance / float64(len(indices))
	}

	// 计算当前节点的方差
	currentVariance := dt.calculateNodeVariance(targets, indices)

	// 分割数据
	leftIndices, rightIndices := dt.splitData(features, indices, node.FeatureIndex, node.Threshold)

	// 计算子节点的方差
	leftVariance := dt.calculateImportanceRecursive(node.LeftChild, features, targets, leftIndices, importance)
	rightVariance := dt.calculateImportanceRecursive(node.RightChild, features, targets, rightIndices, importance)

	// 计算加权方差减少
	totalSamples := float64(len(indices))
	leftWeight := float64(len(leftIndices)) / totalSamples
	rightWeight := float64(len(rightIndices)) / totalSamples

	weightedVariance := leftWeight*leftVariance + rightWeight*rightVariance
	varianceReduction := currentVariance - weightedVariance

	// 更新特征重要性
	if node.FeatureIndex >= 0 && node.FeatureIndex < len(importance) {
		importance[node.FeatureIndex] += varianceReduction
	}

	return currentVariance
}

// calculateNodeVariance 计算节点的方差
func (dt *DecisionTree) calculateNodeVariance(targets []float64, indices []int) float64 {
	if len(indices) == 0 {
		return 0
	}

	sum := 0.0
	for _, idx := range indices {
		sum += targets[idx]
	}
	mean := sum / float64(len(indices))

	variance := 0.0
	for _, idx := range indices {
		diff := targets[idx] - mean
		variance += diff * diff
	}

	return variance / float64(len(indices))
}

// GetName 获取学习器名称
func (dt *DecisionTree) GetName() string {
	return "DecisionTree"
}
