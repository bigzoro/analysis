package server

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DataService 数据服务
type DataService struct {
	dataManager *DataManager
}

// NewDataService 创建数据服务
func NewDataService(dataManager *DataManager) *DataService {
	return &DataService{
		dataManager: dataManager,
	}
}

// GetMultiSourceDataAPI 获取多源数据的API接口
func (ds *DataService) GetMultiSourceDataAPI(c *gin.Context) {
	symbolsParam := c.Query("symbols")
	if symbolsParam == "" {
		c.JSON(400, gin.H{"error": "symbols parameter is required"})
		return
	}

	// 解析币种符号
	symbols := parseSymbols(symbolsParam)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 获取多源数据
	data, err := ds.dataManager.FetchMultiSourceData(ctx, symbols)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch multi-source data: %v", err)
		c.JSON(500, gin.H{"error": "Failed to fetch data"})
		return
	}

	c.JSON(200, gin.H{
		"data":      data,
		"timestamp": time.Now(),
		"sources":   ds.getActiveSources(),
	})
}

// GetSymbolDataAPI 获取单个币种数据的API接口
func (ds *DataService) GetSymbolDataAPI(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	// 尝试从缓存获取
	if data, exists := ds.dataManager.GetCachedData(symbol); exists {
		c.JSON(200, gin.H{
			"data":      data,
			"cached":    true,
			"timestamp": time.Now(),
		})
		return
	}

	// 从多源获取
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	multiData, err := ds.dataManager.FetchMultiSourceData(ctx, []string{symbol})
	if err != nil {
		log.Printf("[ERROR] Failed to fetch symbol data: %v", err)
		c.JSON(500, gin.H{"error": "Failed to fetch data"})
		return
	}

	if symbolData, exists := multiData.SymbolData[symbol]; exists {
		c.JSON(200, gin.H{
			"data":      symbolData,
			"cached":    false,
			"timestamp": time.Now(),
		})
		return
	}

	c.JSON(404, gin.H{"error": "Symbol not found"})
}

// GetDataSourcesAPI 获取可用数据源信息
func (ds *DataService) GetDataSourcesAPI(c *gin.Context) {
	sources := make([]gin.H, 0, len(ds.dataManager.sources))

	for _, source := range ds.dataManager.sources {
		sourceInfo := gin.H{
			"name":      source.Name(),
			"available": source.IsAvailable(),
		}
		sources = append(sources, sourceInfo)
	}

	cacheStats := ds.dataManager.cache.GetStats()

	c.JSON(200, gin.H{
		"sources":   sources,
		"cache":     cacheStats,
		"timestamp": time.Now(),
	})
}

// RefreshDataAPI 刷新数据缓存
func (ds *DataService) RefreshDataAPI(c *gin.Context) {
	symbolsParam := c.Query("symbols")
	symbols := []string{}

	if symbolsParam != "" {
		symbols = parseSymbols(symbolsParam)
	} else {
		// 默认刷新主要币种
		symbols = []string{"BTC", "ETH", "BNB", "ADA", "XRP", "SOL", "DOT", "DOGE"}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	log.Printf("[INFO] Starting data refresh for symbols: %v", symbols)

	// 强制刷新数据（忽略缓存）
	data, err := ds.dataManager.FetchMultiSourceData(ctx, symbols)
	if err != nil {
		log.Printf("[ERROR] Failed to refresh data: %v", err)
		c.JSON(500, gin.H{"error": "Failed to refresh data"})
		return
	}

	c.JSON(200, gin.H{
		"message":   "Data refreshed successfully",
		"data":      data,
		"timestamp": time.Now(),
		"symbols":   symbols,
	})
}

// GetDataQualityReportAPI 获取数据质量报告
func (ds *DataService) GetDataQualityReportAPI(c *gin.Context) {
	report := ds.generateQualityReport()

	c.JSON(200, gin.H{
		"report":    report,
		"timestamp": time.Now(),
	})
}

// parseSymbols 解析币种符号参数
func parseSymbols(symbolsParam string) []string {
	// 支持逗号分隔的格式，如 "BTC,ETH,BNB"
	symbols := []string{}
	parts := strings.Split(symbolsParam, ",")

	for _, part := range parts {
		symbol := strings.TrimSpace(strings.ToUpper(part))
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}

// getActiveSources 获取活跃数据源
func (ds *DataService) getActiveSources() []string {
	active := make([]string, 0)
	for _, source := range ds.dataManager.sources {
		if source.IsAvailable() {
			active = append(active, source.Name())
		}
	}
	return active
}

// generateQualityReport 生成质量报告
func (ds *DataService) generateQualityReport() gin.H {
	cacheStats := ds.dataManager.cache.GetStats()

	report := gin.H{
		"cache_stats":    cacheStats,
		"active_sources": len(ds.getActiveSources()),
		"total_sources":  len(ds.dataManager.sources),
		"quality_metrics": gin.H{
			"cache_hit_ratio":  ds.calculateCacheHitRatio(),
			"data_freshness":   ds.calculateDataFreshness(),
			"source_diversity": ds.calculateSourceDiversity(),
		},
	}

	return report
}

// calculateCacheHitRatio 计算缓存命中率 (模拟值)
func (ds *DataService) calculateCacheHitRatio() float64 {
	// 在实际实现中，应该跟踪缓存命中统计
	return 0.85 // 85% 命中率
}

// calculateDataFreshness 计算数据新鲜度
func (ds *DataService) calculateDataFreshness() string {
	if ds.dataManager.cache.IsExpired() {
		return "expired"
	}
	return "fresh"
}

// calculateSourceDiversity 计算数据源多样性
func (ds *DataService) calculateSourceDiversity() float64 {
	activeCount := len(ds.getActiveSources())
	totalCount := len(ds.dataManager.sources)

	if totalCount == 0 {
		return 0
	}

	return float64(activeCount) / float64(totalCount)
}

// RegisterRoutes 注册数据服务路由
func (ds *DataService) RegisterRoutes(router *gin.Engine) {
	dataGroup := router.Group("/api/data")
	{
		dataGroup.GET("/multi-source", ds.GetMultiSourceDataAPI)
		dataGroup.GET("/symbol/:symbol", ds.GetSymbolDataAPI)
		dataGroup.GET("/sources", ds.GetDataSourcesAPI)
		dataGroup.POST("/refresh", ds.RefreshDataAPI)
		dataGroup.GET("/quality-report", ds.GetDataQualityReportAPI)
	}
}
