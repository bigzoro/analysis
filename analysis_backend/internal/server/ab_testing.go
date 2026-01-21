package server

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ABTestingService A/B测试服务
type ABTestingService struct {
	db          *gorm.DB
	testConfigs map[string]*pdb.ABTestConfig
	userGroups  map[string]string // 用户ID -> 测试组
	rand        *rand.Rand
}

// ABTestGroup 测试组配置（用于内存操作，不存储在数据库中）
type ABTestGroup struct {
	GroupName   string                 `json:"group_name"`
	Description string                 `json:"description"`
	Weight      float64                `json:"weight"` // 分配权重 0-1
	Config      map[string]interface{} `json:"config"` // 组特定配置
}

// GroupResult 组结果（用于内存操作，不存储在数据库中）
type GroupResult struct {
	GroupName     string     `json:"group_name"`
	SampleSize    int        `json:"sample_size"`
	MetricValue   float64    `json:"metric_value"`
	StdDev        float64    `json:"std_dev"`
	ConfidenceInt [2]float64 `json:"confidence_interval"`
}

// NewABTestingService 创建A/B测试服务
func NewABTestingService(db *gorm.DB) *ABTestingService {
	return &ABTestingService{
		db:          db,
		testConfigs: make(map[string]*pdb.ABTestConfig),
		userGroups:  make(map[string]string),
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Initialize 初始化A/B测试服务
func (ats *ABTestingService) Initialize() error {
	// 加载活跃的测试配置
	configs, err := pdb.GetABTestConfigs(ats.db, "active", 0)
	if err != nil {
		return err
	}

	for _, config := range configs {
		ats.testConfigs[config.TestName] = &config
	}

	log.Printf("已加载 %d 个活跃的A/B测试", len(configs))
	return nil
}

// AssignUserToGroup 为用户分配测试组
func (ats *ABTestingService) AssignUserToGroup(userID uint, testName string) string {
	cacheKey := fmt.Sprintf("%d_%s", userID, testName)

	// 检查缓存
	if group, exists := ats.userGroups[cacheKey]; exists {
		return group
	}

	config, exists := ats.testConfigs[testName]
	if !exists {
		return "control" // 默认控制组
	}

	// 解析Groups JSON
	var groups []ABTestGroup
	if config.Groups != "" {
		if err := json.Unmarshal([]byte(config.Groups), &groups); err != nil {
			log.Printf("Failed to parse groups JSON: %v", err)
			return "control"
		}
	}

	if len(groups) == 0 {
		return "control"
	}

	// 使用加权随机分配
	r := ats.rand.Float64()
	cumulativeWeight := 0.0

	for _, group := range groups {
		cumulativeWeight += group.Weight
		if r <= cumulativeWeight {
			ats.userGroups[cacheKey] = group.GroupName

			// 记录分配结果
			assignment := map[string]interface{}{
				"user_id":     userID,
				"test_name":   testName,
				"group":       group.GroupName,
				"assigned_at": time.Now(),
			}

			// 异步保存到数据库
			go ats.saveAssignment(assignment)

			return group.GroupName
		}
	}

	// 默认分配到第一个组
	defaultGroup := groups[0].GroupName
	ats.userGroups[cacheKey] = defaultGroup
	return defaultGroup
}

// GetGroupConfig 获取用户组配置
func (ats *ABTestingService) GetGroupConfig(userID uint, testName string) map[string]interface{} {
	groupName := ats.AssignUserToGroup(userID, testName)

	config, exists := ats.testConfigs[testName]
	if !exists {
		return map[string]interface{}{}
	}

	// 解析Groups JSON
	var groups []ABTestGroup
	if config.Groups != "" {
		if err := json.Unmarshal([]byte(config.Groups), &groups); err != nil {
			log.Printf("Failed to parse groups JSON: %v", err)
			return map[string]interface{}{}
		}
	}

	// 查找组配置
	for _, group := range groups {
		if group.GroupName == groupName {
			return group.Config
		}
	}

	return map[string]interface{}{}
}

// TrackMetric 跟踪测试指标
func (ats *ABTestingService) TrackMetric(testName, groupName, metricName string, value float64, userID *uint, metadata map[string]interface{}) {
	metricData := map[string]interface{}{
		metricName: value,
		"user_id":  userID,
		"metadata": metadata,
	}

	metricJSON, _ := json.Marshal(metricData)

	metric := pdb.AlgorithmPerformance{
		AlgorithmVersion: testName,
		TestGroup:        groupName,
		TimeRange:        time.Now().Format("2006-01-02"),
		SampleSize:       1,
		Metrics:          metricJSON,
		CreatedAt:        time.Now(),
	}

	// 异步保存
	go func() {
		if err := ats.db.Create(&metric).Error; err != nil {
			log.Printf("保存测试指标失败: %v", err)
		}
	}()
}

// AnalyzeTestResults 分析测试结果
func (ats *ABTestingService) AnalyzeTestResults(testName string) (*pdb.ABTestResult, error) {
	config, exists := ats.testConfigs[testName]
	if !exists {
		return nil, fmt.Errorf("测试不存在: %s", testName)
	}

	// 查询测试数据
	var metrics []pdb.AlgorithmPerformance
	startTime := config.StartTime
	endTime := time.Now()
	if config.EndTime != nil {
		endTime = *config.EndTime
	}

	if err := ats.db.Where("algorithm_version = ? AND created_at BETWEEN ? AND ?",
		testName, startTime, endTime).Find(&metrics).Error; err != nil {
		return nil, err
	}

	// 按组聚合数据
	groupData := make(map[string][]float64)
	totalSampleSize := 0

	for _, metric := range metrics {
		var metricData map[string]interface{}
		if err := json.Unmarshal(metric.Metrics, &metricData); err != nil {
			continue
		}

		if value, exists := metricData[config.TargetMetric]; exists {
			if floatVal, ok := value.(float64); ok {
				groupData[metric.TestGroup] = append(groupData[metric.TestGroup], floatVal)
				totalSampleSize++
			}
		}
	}

	// 计算各组统计
	var groupResults []GroupResult
	var bestGroup string
	var bestScore float64

	for groupName, values := range groupData {
		if len(values) == 0 {
			continue
		}

		result := ats.calculateGroupStats(groupName, values)
		groupResults = append(groupResults, result)

		if result.MetricValue > bestScore {
			bestScore = result.MetricValue
			bestGroup = groupName
		}
	}

	// 计算统计显著性
	statisticalSig := ats.calculateStatisticalSignificance(groupResults)

	// 计算改进率
	var controlScore float64
	for _, result := range groupResults {
		if result.GroupName == "control" {
			controlScore = result.MetricValue
			break
		}
	}

	improvementRate := 0.0
	if controlScore > 0 {
		improvementRate = (bestScore - controlScore) / controlScore
	}

	// 将GroupResults序列化为JSON
	groupResultsJSON, err := json.Marshal(groupResults)
	if err != nil {
		return nil, fmt.Errorf("序列化组结果失败: %v", err)
	}

	result := &pdb.ABTestResult{
		TestName:        testName,
		GroupResults:    string(groupResultsJSON),
		BestGroup:       bestGroup,
		ConfidenceLevel: 0.95, // 95%置信区间
		StatisticalSig:  statisticalSig,
		ImprovementRate: improvementRate,
		SampleSize:      totalSampleSize,
		Duration:        time.Since(startTime).Nanoseconds(), // 转换为纳秒数
	}

	// 生成推荐
	result.Recommendation = ats.generateRecommendation(result)

	return result, nil
}

// calculateGroupStats 计算组统计信息
func (ats *ABTestingService) calculateGroupStats(groupName string, values []float64) GroupResult {
	n := float64(len(values))
	sum := 0.0
	sumSq := 0.0

	for _, v := range values {
		sum += v
		sumSq += v * v
	}

	mean := sum / n
	variance := (sumSq/n - mean*mean)
	stdDev := math.Sqrt(variance)

	// 95%置信区间 (t分布近似)
	tValue := 1.96 // 95%置信水平
	margin := tValue * stdDev / math.Sqrt(n)
	confidenceInt := [2]float64{mean - margin, mean + margin}

	return GroupResult{
		GroupName:     groupName,
		SampleSize:    len(values),
		MetricValue:   mean,
		StdDev:        stdDev,
		ConfidenceInt: confidenceInt,
	}
}

// calculateStatisticalSignificance 计算统计显著性
func (ats *ABTestingService) calculateStatisticalSignificance(results []GroupResult) float64 {
	if len(results) < 2 {
		return 0.0
	}

	// 简化的t检验实现
	var controlResult, testResult GroupResult
	for _, result := range results {
		if result.GroupName == "control" {
			controlResult = result
		} else if result.SampleSize > testResult.SampleSize {
			testResult = result
		}
	}

	if controlResult.SampleSize == 0 || testResult.SampleSize == 0 {
		return 0.0
	}

	// 计算t统计量
	meanDiff := testResult.MetricValue - controlResult.MetricValue
	se := math.Sqrt(
		math.Pow(controlResult.StdDev, 2)/float64(controlResult.SampleSize) +
			math.Pow(testResult.StdDev, 2)/float64(testResult.SampleSize),
	)

	if se == 0 {
		return 1.0
	}

	tStat := math.Abs(meanDiff / se)

	// 简化的p值计算（正态分布近似）
	pValue := 2 * (1 - ats.normalCDF(tStat))

	return 1 - pValue // 显著性水平
}

// normalCDF 正态分布累积分布函数近似
func (ats *ABTestingService) normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

// generateRecommendation 生成测试推荐
func (ats *ABTestingService) generateRecommendation(result *pdb.ABTestResult) string {
	if result.StatisticalSig < 0.95 {
		return "样本量不足，建议继续测试"
	}

	if result.ImprovementRate < 0.05 {
		return "改进不显著，建议重新设计测试"
	}

	return fmt.Sprintf("推荐使用%s组，改进率%.1f%%，统计显著性%.1f%%",
		result.BestGroup,
		result.ImprovementRate*100,
		result.StatisticalSig*100)
}

// saveAssignment 保存用户分配记录
func (ats *ABTestingService) saveAssignment(assignment map[string]interface{}) {
	// 这里可以保存到专门的分配表中
	log.Printf("用户分配: %+v", assignment)
}

// CreateTest 创建新的A/B测试
func (ats *ABTestingService) CreateTest(c *gin.Context) {
	// 使用map来接收输入，避免类型冲突
	var inputData map[string]interface{}
	if err := c.ShouldBindJSON(&inputData); err != nil {
		c.JSON(400, gin.H{"error": "无效的测试配置"})
		return
	}

	// 转换为ABTestGroup数组
	var groups []ABTestGroup
	if groupsData, ok := inputData["groups"].([]interface{}); ok {
		for _, g := range groupsData {
			if groupMap, ok := g.(map[string]interface{}); ok {
				group := ABTestGroup{}
				if name, ok := groupMap["group_name"].(string); ok {
					group.GroupName = name
				}
				if desc, ok := groupMap["description"].(string); ok {
					group.Description = desc
				}
				if weight, ok := groupMap["weight"].(float64); ok {
					group.Weight = weight
				}
				if config, ok := groupMap["config"].(map[string]interface{}); ok {
					group.Config = config
				}
				groups = append(groups, group)
			}
		}
	}

	// 将Groups转换为JSON字符串
	groupsJSON, err := json.Marshal(groups)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的测试组配置"})
		return
	}

	// 获取其他字段
	testName, _ := inputData["test_name"].(string)
	description, _ := inputData["description"].(string)
	targetMetric, _ := inputData["target_metric"].(string)
	minSampleSizeFloat, _ := inputData["min_sample_size"].(float64)
	minSampleSize := int(minSampleSizeFloat)

	// 将Metadata转换为JSON字符串
	var metadataJSON string
	if metadata, ok := inputData["metadata"].(map[string]interface{}); ok {
		if jsonBytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	// 获取用户ID
	userID, exists := c.Get("uid")
	if !exists {
		c.JSON(401, gin.H{"error": "用户未登录"})
		return
	}

	config := &pdb.ABTestConfig{
		TestName:      testName,
		Description:   description,
		Status:        "active",
		Groups:        string(groupsJSON),
		TargetMetric:  targetMetric,
		MinSampleSize: minSampleSize,
		StartTime:     time.Now(),
		CreatedBy:     userID.(uint),
		Metadata:      metadataJSON,
	}

	// 保存到数据库
	if err := pdb.CreateABTestConfig(ats.db, config); err != nil {
		c.JSON(500, gin.H{"error": "创建测试失败"})
		return
	}

	// 添加到内存缓存
	ats.testConfigs[config.TestName] = config

	c.JSON(200, gin.H{
		"success":   true,
		"test_name": config.TestName,
	})
}

// GetTestResults 获取测试结果
func (ats *ABTestingService) GetTestResults(c *gin.Context) {
	testName := c.Param("test_name")
	if testName == "" {
		c.JSON(400, gin.H{"error": "缺少测试名称"})
		return
	}

	result, err := ats.AnalyzeTestResults(testName)
	if err != nil {
		c.JSON(500, gin.H{"error": "分析测试结果失败: " + err.Error()})
		return
	}

	// 解析GroupResults JSON
	var groupResults []GroupResult
	if result.GroupResults != "" {
		json.Unmarshal([]byte(result.GroupResults), &groupResults)
	}

	// 返回格式化的结果
	response := gin.H{
		"test_name":                result.TestName,
		"group_results":            groupResults,
		"best_group":               result.BestGroup,
		"confidence_level":         result.ConfidenceLevel,
		"statistical_significance": result.StatisticalSig,
		"improvement_rate":         result.ImprovementRate,
		"sample_size":              result.SampleSize,
		"duration":                 result.Duration,
		"recommendation":           result.Recommendation,
		"created_at":               result.CreatedAt,
	}

	c.JSON(200, response)
}

// ListActiveTests 列出活跃测试
func (ats *ABTestingService) ListActiveTests(c *gin.Context) {
	tests := make([]gin.H, 0, len(ats.testConfigs))
	for _, config := range ats.testConfigs {
		// 解析Groups JSON
		var groups []ABTestGroup
		if config.Groups != "" {
			json.Unmarshal([]byte(config.Groups), &groups)
		}

		// 解析Metadata JSON
		var metadata map[string]interface{}
		if config.Metadata != "" {
			json.Unmarshal([]byte(config.Metadata), &metadata)
		}

		test := gin.H{
			"id":              config.ID,
			"test_name":       config.TestName,
			"description":     config.Description,
			"status":          config.Status,
			"groups":          groups,
			"target_metric":   config.TargetMetric,
			"min_sample_size": config.MinSampleSize,
			"start_time":      config.StartTime,
			"end_time":        config.EndTime,
			"created_by":      config.CreatedBy,
			"metadata":        metadata,
			"created_at":      config.CreatedAt,
			"updated_at":      config.UpdatedAt,
		}
		tests = append(tests, test)
	}

	c.JSON(200, gin.H{
		"tests": tests,
		"count": len(tests),
	})
}
