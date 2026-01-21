// cmd/recommendation_scanner/main.go
package main

import (
	"analysis/internal/config"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"analysis/internal/netutil"
)

// =============================
//         推荐扫描器
// =============================

// RecommendationScanner 推荐扫描器
type RecommendationScanner struct {
	apiBase    string
	config     *config.Config
	mode       string // 运行模式: "warmup"(预热缓存) 或 "generate"(生成历史推荐)
	isRunning  bool
	lastRun    *time.Time
	nextRun    *time.Time
	totalRuns  int64
	httpServer *http.Server
}

// NewRecommendationScanner 创建推荐扫描器
func NewRecommendationScanner(apiBase string, cfg *config.Config, generationMode string) *RecommendationScanner {
	return &RecommendationScanner{
		apiBase: apiBase,
		config:  cfg,
		mode:    generationMode,
	}
}

func main() {
	// 命令行参数
	apiBase := flag.String("api", "http://127.0.0.1:8010", "API服务器地址")
	configPath := flag.String("config", "./config.yaml", "配置文件路径")
	mode := flag.String("mode", "continuous", "运行模式: once(单次运行), continuous(持续运行), generate(生成推荐), server(仅HTTP服务器)")
	generationMode := flag.String("gen-mode", "generate", "生成模式: generate(生成历史推荐), warmup(预热当前推荐缓存)")
	interval := flag.Duration("interval", 30*time.Minute, "连续模式下的运行间隔")
	kind := flag.String("kind", "spot", "推荐类型: spot(现货), futures(期货)")
	limit := flag.Int("limit", 5, "推荐数量限制")
	forceRefresh := flag.Bool("force-refresh", false, "强制刷新推荐（忽略缓存）")
	port := flag.String("port", "8011", "HTTP服务器端口（仅server模式）")

	flag.Parse()

	log.Printf("[recommendation_scanner] 启动推荐扫描器...")
	log.Printf("[recommendation_scanner] API: %s, 运行模式: %s, 生成模式: %s, 类型: %s, 数量: %d", *apiBase, *mode, *generationMode, *kind, *limit)

	// 加载配置
	var cfg config.Config
	config.MustLoad(*configPath, &cfg)
	config.ApplyProxy(&cfg)

	// 创建扫描器
	scanner := NewRecommendationScanner(*apiBase, &cfg, *generationMode)

	// 启动HTTP控制服务器
	go scanner.startHTTPServer(*port)

	// 等待一秒让HTTP服务器启动
	time.Sleep(time.Second)

	// 根据模式运行
	ctx := context.Background()

	switch *mode {
	case "once":
		log.Printf("[recommendation_scanner] 执行单次推荐生成...")
		if err := scanner.generateOnce(ctx, *kind, *limit, *forceRefresh); err != nil {
			log.Fatalf("[recommendation_scanner] 单次生成失败: %v", err)
		}
		log.Printf("[recommendation_scanner] 单次生成完成")

	case "continuous":
		log.Printf("[recommendation_scanner] 启动持续推荐生成模式，间隔: %v", *interval)
		scanner.runContinuous(ctx, *interval, *kind, *limit, *forceRefresh)

	case "generate":
		log.Printf("[recommendation_scanner] 执行推荐生成...")
		if err := scanner.generateRecommendations(ctx, *kind, *limit, *forceRefresh); err != nil {
			log.Fatalf("[recommendation_scanner] 推荐生成失败: %v", err)
		}

	case "server":
		log.Printf("[recommendation_scanner] 仅启动HTTP服务器模式...")
		// 只启动HTTP服务器，不执行任何扫描
		select {}

	default:
		log.Fatalf("[recommendation_scanner] 不支持的模式: %s", *mode)
	}
}

// generateOnce 单次生成推荐
func (rs *RecommendationScanner) generateOnce(ctx context.Context, kind string, limit int, forceRefresh bool) error {
	log.Printf("[recommendation_scanner] 开始单次推荐生成...")

	return rs.generateRecommendations(ctx, kind, limit, forceRefresh)
}

// runContinuous 持续运行模式
func (rs *RecommendationScanner) runContinuous(ctx context.Context, interval time.Duration, kind string, limit int, forceRefresh bool) {
	log.Printf("[recommendation_scanner] 启动持续推荐生成模式...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次运行
	if err := rs.generateRecommendations(ctx, kind, limit, forceRefresh); err != nil {
		log.Printf("[recommendation_scanner] 首次运行失败: %v", err)
	}

	// 定时运行
	for {
		select {
		case <-ctx.Done():
			log.Printf("[recommendation_scanner] 收到停止信号，退出...")
			return
		case <-ticker.C:
			log.Printf("[recommendation_scanner] 执行定时推荐生成...")
			if err := rs.generateRecommendations(ctx, kind, limit, forceRefresh); err != nil {
				log.Printf("[recommendation_scanner] 定时生成失败: %v", err)
			}
		}
	}
}

// generateRecommendations 生成推荐（支持多种模式）
func (rs *RecommendationScanner) generateRecommendations(ctx context.Context, kind string, limit int, forceRefresh bool) error {
	// 更新统计信息
	now := time.Now().UTC()
	rs.lastRun = &now
	rs.totalRuns++

	log.Printf("[recommendation_scanner] 开始生成推荐: kind=%s, limit=%d, forceRefresh=%v", kind, limit, forceRefresh)

	// 根据模式选择不同的API
	mode := "generate" // 默认生成历史推荐
	if rs.mode != "" {
		mode = rs.mode
	}

	switch mode {
	case "warmup":
		// 预热模式：调用 /recommendations/coins 来预热缓存
		return rs.warmupRecommendations(ctx, kind, limit)
	case "generate":
		fallthrough
	default:
		// 生成模式：调用 /recommendations/generate 生成历史推荐
		return rs.generateHistoricalRecommendations(ctx, kind, limit, forceRefresh)
	}
}

// warmupRecommendations 预热推荐缓存
func (rs *RecommendationScanner) warmupRecommendations(ctx context.Context, kind string, limit int) error {
	log.Printf("[recommendation_scanner] 开始预热推荐缓存: kind=%s, limit=%d", kind, limit)

	// 构造API请求 - 调用 /recommendations/coins 来预热缓存
	requestBody := map[string]interface{}{
		"kind":    kind,
		"limit":   limit,
		"refresh": true, // 强制刷新以确保数据更新
	}

	url := rs.apiBase + "/recommendations/coins"
	log.Printf("[recommendation_scanner] 预热API: %s", url)

	// 发送API请求
	resp, err := rs.makeAPIRequest(ctx, "GET", url, requestBody)
	if err != nil {
		return fmt.Errorf("推荐缓存预热失败: %w", err)
	}

	// 解析响应
	if recommendations, ok := resp["recommendations"].([]interface{}); ok {
		log.Printf("[recommendation_scanner] 预热成功，缓存推荐数量: %d", len(recommendations))

		// 输出预热结果
		for i, rec := range recommendations {
			if i >= 3 { // 只显示前3个
				break
			}
			if recMap, ok := rec.(map[string]interface{}); ok {
				if symbol, ok := recMap["symbol"].(string); ok {
					if score, ok := recMap["total_score"].(float64); ok {
						log.Printf("[recommendation_scanner]   %d. %s (分数: %.2f)", i+1, symbol, score)
					}
				}
			}
		}

		// 检查是否使用了缓存
		if cached, ok := resp["cached"].(bool); ok && cached {
			log.Printf("[recommendation_scanner] 注意：预热使用了已有缓存，如需强制更新请设置refresh=true")
		} else {
			log.Printf("[recommendation_scanner] 预热完成了新的推荐计算")
		}
	} else {
		return fmt.Errorf("预热响应格式错误")
	}

	log.Printf("[recommendation_scanner] 推荐缓存预热完成")
	return nil
}

// generateHistoricalRecommendations 生成历史推荐（原有逻辑）
func (rs *RecommendationScanner) generateHistoricalRecommendations(ctx context.Context, kind string, limit int, forceRefresh bool) error {
	log.Printf("[recommendation_scanner] 开始生成历史推荐: kind=%s, limit=%d, forceRefresh=%v", kind, limit, forceRefresh)

	// 构造API请求
	requestBody := map[string]interface{}{
		"kind":    kind,
		"limit":   limit,
		"refresh": forceRefresh,
	}

	url := rs.apiBase + "/recommendations/generate"
	log.Printf("[recommendation_scanner] 调用API: %s", url)

	// 发送API请求
	resp, err := rs.makeAPIRequest(ctx, "POST", url, requestBody)
	if err != nil {
		return fmt.Errorf("历史推荐生成请求失败: %w", err)
	}

	// 解析响应
	if success, ok := resp["success"].(bool); ok && success {
		if message, ok := resp["message"].(string); ok {
			log.Printf("[recommendation_scanner] 历史推荐生成成功: %s", message)
		} else {
			log.Printf("[recommendation_scanner] 历史推荐生成成功")
		}

		// 输出推荐结果
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if recommendations, ok := data["recommendations"].([]interface{}); ok {
				log.Printf("[recommendation_scanner] 生成推荐数量: %d", len(recommendations))

				// 简单输出前几个推荐
				for i, rec := range recommendations {
					if i >= 3 { // 只显示前3个
						break
					}
					if recMap, ok := rec.(map[string]interface{}); ok {
						if symbol, ok := recMap["symbol"].(string); ok {
							if score, ok := recMap["total_score"].(float64); ok {
								log.Printf("[recommendation_scanner]   %d. %s (分数: %.2f)", i+1, symbol, score)
							}
						}
					}
				}
			}
		}
	} else {
		if message, ok := resp["error"].(string); ok {
			return fmt.Errorf("历史推荐生成失败: %s", message)
		}
		return fmt.Errorf("历史推荐生成失败")
	}

	log.Printf("[recommendation_scanner] 历史推荐生成完成")
	return nil
}

// makeAPIRequest 发送API请求的辅助方法
func (rs *RecommendationScanner) makeAPIRequest(ctx context.Context, method, url string, body interface{}) (map[string]interface{}, error) {
	log.Printf("[recommendation_scanner] 发送%s请求到: %s", method, url)

	var reqBody interface{} = nil
	if body != nil {
		reqBody = body
	}

	// 发送请求
	var result map[string]interface{}
	err := netutil.PostJSON(ctx, url, reqBody, &result)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}

	// 检查API响应状态
	if success, ok := result["success"].(bool); ok && !success {
		if message, ok := result["error"].(string); ok {
			return nil, fmt.Errorf("API返回错误: %s", message)
		}
		return nil, fmt.Errorf("API请求失败")
	}

	return result, nil
}

// =============================
//         HTTP 控制服务器
// =============================

// startHTTPServer 启动HTTP控制服务器
func (rs *RecommendationScanner) startHTTPServer(port string) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 状态查询接口
	r.GET("/status", rs.handleGetStatus)

	// 统计信息接口
	r.GET("/stats", rs.handleGetStats)

	// 控制接口
	control := r.Group("/control")
	{
		control.POST("/start", rs.handleStart)
		control.POST("/stop", rs.handleStop)
		control.POST("/generate", rs.handleGenerate)
		control.POST("/cleanup", rs.handleCleanup)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	serverAddr := ":" + port
	log.Printf("[recommendation_scanner] HTTP控制服务器启动在端口 %s", port)

	rs.httpServer = &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	if err := rs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[recommendation_scanner] HTTP服务器启动失败: %v", err)
	}
}

// handleGetStatus 获取状态
func (rs *RecommendationScanner) handleGetStatus(c *gin.Context) {
	status := map[string]interface{}{
		"is_running": rs.isRunning,
		"total_runs": rs.totalRuns,
	}

	if rs.lastRun != nil {
		status["last_run"] = rs.lastRun.UTC().Format(time.RFC3339)
	}
	if rs.nextRun != nil {
		status["next_run"] = rs.nextRun.UTC().Format(time.RFC3339)
	}

	c.JSON(200, gin.H{
		"status": "success",
		"data":   status,
	})
}

// handleGetStats 获取统计信息
func (rs *RecommendationScanner) handleGetStats(c *gin.Context) {
	stats := map[string]interface{}{
		"total_runs": rs.totalRuns,
		"is_running": rs.isRunning,
		"uptime":     time.Since(time.Now().Add(-time.Hour)).String(), // 简化的uptime
	}

	c.JSON(200, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// handleStart 启动调度器
func (rs *RecommendationScanner) handleStart(c *gin.Context) {
	if rs.isRunning {
		c.JSON(400, gin.H{
			"status":  "error",
			"message": "调度器已经在运行中",
		})
		return
	}

	rs.isRunning = true
	log.Printf("[recommendation_scanner] 通过HTTP接口启动调度器")

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "调度器已启动",
	})
}

// handleStop 停止调度器
func (rs *RecommendationScanner) handleStop(c *gin.Context) {
	if !rs.isRunning {
		c.JSON(400, gin.H{
			"status":  "error",
			"message": "调度器未在运行",
		})
		return
	}

	rs.isRunning = false
	log.Printf("[recommendation_scanner] 通过HTTP接口停止调度器")

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "调度器已停止",
	})
}

// handleGenerate 强制生成推荐
func (rs *RecommendationScanner) handleGenerate(c *gin.Context) {
	kind := c.DefaultQuery("kind", "spot")
	limitStr := c.DefaultQuery("limit", "5")
	genMode := c.DefaultQuery("gen_mode", rs.mode) // 支持动态指定生成模式

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		c.JSON(400, gin.H{
			"status":  "error",
			"message": "无效的limit参数",
		})
		return
	}

	// 临时设置生成模式
	originalMode := rs.mode
	rs.mode = genMode
	defer func() { rs.mode = originalMode }() // 恢复原始模式

	modeDesc := "推荐"
	if genMode == "warmup" {
		modeDesc = "推荐缓存预热"
	} else if genMode == "generate" {
		modeDesc = "历史推荐生成"
	}

	log.Printf("[recommendation_scanner] 通过HTTP接口强制%s: kind=%s, limit=%d", modeDesc, kind, limit)

	// 执行推荐生成
	ctx := context.Background()
	if err := rs.generateRecommendations(ctx, kind, limit, false); err != nil {
		c.JSON(500, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("%s失败: %v", modeDesc, err),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("%s完成", modeDesc),
		"data": gin.H{
			"kind":     kind,
			"limit":    limit,
			"gen_mode": genMode,
		},
	})
}

// handleCleanup 清理旧推荐
func (rs *RecommendationScanner) handleCleanup(c *gin.Context) {
	maxAgeStr := c.DefaultQuery("max_age_hours", "8760")
	maxAgeHours, err := strconv.Atoi(maxAgeStr)
	if err != nil || maxAgeHours <= 0 {
		c.JSON(400, gin.H{
			"status":  "error",
			"message": "无效的max_age_hours参数",
		})
		return
	}

	log.Printf("[recommendation_scanner] 通过HTTP接口清理旧推荐: max_age_hours=%d", maxAgeHours)

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "清理功能暂未实现",
		"data": gin.H{
			"max_age_hours": maxAgeHours,
			"note":          "清理功能将在后续版本中实现",
		},
	})
}
