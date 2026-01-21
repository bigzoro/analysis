package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 策略处理器 - 负责处理策略相关的HTTP API请求
// ============================================================================

// StrategyHandler 策略处理器结构体
type StrategyHandler struct {
	server *Server
}

// NewStrategyHandler 创建策略处理器
func NewStrategyHandler(server *Server) *StrategyHandler {
	return &StrategyHandler{
		server: server,
	}
}

// ============================================================================
// 策略执行API接口
// ============================================================================

// ExecuteStrategy 执行策略判断
func (h *StrategyHandler) ExecuteStrategy(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req struct {
		Symbol             string             `json:"symbol" binding:"required"`
		StrategyID         uint               `json:"strategy_id" binding:"required"`
		StrategyMarketData StrategyMarketData `json:"market_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.server.JSONBindError(c, err)
		return
	}

	// 获取策略
	strategy, err := pdb.GetTradingStrategy(h.server.db.DB(), uid, req.StrategyID)
	if err != nil {
		h.server.DatabaseError(c, "获取策略", err)
		return
	}

	// 执行策略判断
	result := executeStrategyLogic(strategy, req.Symbol, req.StrategyMarketData)

	// 如果返回allow，表示需要外部依赖，进一步处理
	if result.Action == "allow" {
		result = h.server.executeStrategyWithNewExecutors(c.Request.Context(), req.Symbol, req.StrategyMarketData, strategy.Conditions, strategy)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// BatchExecuteStrategies 批量执行策略（用于定时任务）
func (h *StrategyHandler) BatchExecuteStrategies(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	type batchRequest struct {
		Orders []struct {
			Symbol     string             `json:"symbol"`
			StrategyID uint               `json:"strategy_id"`
			MarketData StrategyMarketData `json:"market_data"`
		} `json:"orders"`
	}

	var req batchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.server.JSONBindError(c, err)
		return
	}

	results := make(map[string]StrategyDecisionResult)

	for _, order := range req.Orders {
		// 获取策略
		strategy, err := pdb.GetTradingStrategy(h.server.db.DB(), uid, order.StrategyID)
		if err != nil {
			results[order.Symbol] = StrategyDecisionResult{
				Action:     "error",
				Reason:     fmt.Sprintf("获取策略失败: %v", err),
				Multiplier: 1.0,
			}
			continue
		}

		// 执行策略判断
		result := executeStrategyLogic(strategy, order.Symbol, order.MarketData)
		results[order.Symbol] = result
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
	})
}

// ============================================================================
// 策略扫描API接口
// ============================================================================

// ScanEligibleSymbols 扫描符合策略的币种
func (h *StrategyHandler) ScanEligibleSymbols(c *gin.Context) {
	// 并发控制：尝试获取扫描锁
	if !h.acquireScanLock() {
		log.Printf("[ScanEligible] 扫描正在进行中，拒绝并发请求")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "扫描正在进行中，请稍后再试",
		})
		return
	}
	defer h.releaseScanLock()

	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req struct {
		StrategyID uint `json:"strategy_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.server.JSONBindError(c, err)
		return
	}

	// 获取策略
	strategy, err := pdb.GetTradingStrategy(h.server.db.DB(), uid, req.StrategyID)
	if err != nil {
		h.server.DatabaseError(c, "获取策略", err)
		return
	}

	log.Printf("[ScanEligible] 开始扫描符合条件的币种，策略ID: %d", req.StrategyID)

	// 性能监控：记录扫描开始时间
	scanStartTime := time.Now()

	// 使用扫描器注册表选择合适的策略扫描器
	scanner := h.server.scannerRegistry.SelectScanner(strategy)
	if scanner == nil {
		log.Printf("[ScanEligible] 未找到合适的扫描器")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "未找到合适的策略扫描器，请检查策略配置和系统状态",
		})
		return
	}

	log.Printf("[ScanEligible] 使用扫描器: %s", scanner.GetStrategyType())

	// 执行扫描
	rawResults, err := scanner.Scan(context.Background(), strategy)
	scanDuration := time.Since(scanStartTime)

	if err != nil {
		log.Printf("[ScanEligible] 策略扫描失败: %v (耗时: %v)", err, scanDuration)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("策略扫描失败: %v", err),
		})
		return
	}

	// 转换结果为EligibleSymbol
	eligibleSymbols := make([]EligibleSymbol, 0, len(rawResults))
	for _, raw := range rawResults {
		if symbolMap, ok := raw.(map[string]interface{}); ok {
			symbol := EligibleSymbol{
				Symbol:      getStringValue(symbolMap, "symbol"),
				Action:      getStringValue(symbolMap, "action"),
				Reason:      getStringValue(symbolMap, "reason"),
				Multiplier:  getFloat64Value(symbolMap, "multiplier"),
				MarketCap:   getFloat64Value(symbolMap, "market_cap"),
				GainersRank: int(getFloat64Value(symbolMap, "gainers_rank")),
			}
			eligibleSymbols = append(eligibleSymbols, symbol)
		}
	}

	log.Printf("[ScanEligible] 扫描完成，发现%d个符合条件的币种 (耗时: %v)",
		len(eligibleSymbols), scanDuration)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"eligible_symbols": eligibleSymbols,
		"total_count":      len(eligibleSymbols),
		"strategy_type":    scanner.GetStrategyType(),
	})
}

// ============================================================================
// 套利机会发现API接口
// ============================================================================

// DiscoverArbitrageOpportunities 发现套利机会API
func (h *StrategyHandler) DiscoverArbitrageOpportunities(c *gin.Context) {
	log.Printf("[DiscoverArbitrage] 开始发现套利机会...")

	// 检查用户权限
	uidVal, _ := c.Get("uid")
	_ = uint(uidVal.(uint)) // 用户ID暂时未使用

	// 创建市场数据服务
	mds := NewMarketDataService(h.server)

	// 动态获取所有USDT交易对
	discoverySymbols, err := mds.getAllUSDTTradingPairs(context.Background())
	if err != nil {
		log.Printf("[DiscoverArbitrage] 获取交易对列表失败: %v，使用默认列表", err)
		// 降级到默认列表
		discoverySymbols = []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "LINKUSDT",
		}
	}

	// 限制扫描数量，避免API限流（最多扫描前200个交易对）
	maxScan := 200
	if len(discoverySymbols) > maxScan {
		log.Printf("[DiscoverArbitrage] 交易对数量过多(%d)，限制为前%d个", len(discoverySymbols), maxScan)
		discoverySymbols = discoverySymbols[:maxScan]
	}

	type ArbitrageOpportunity struct {
		Symbol           string  `json:"symbol"`
		Action           string  `json:"action"`
		Reason           string  `json:"reason"`
		PriceDiffPercent float64 `json:"price_diff_percent"`
		SpotPrice        float64 `json:"spot_price"`
		FuturesPrice     float64 `json:"futures_price"`
	}

	// 创建批量操作上下文
	batchCtx := context.WithValue(context.Background(), "batch_operation", true)

	opportunities := []ArbitrageOpportunity{}
	totalScanned := 0
	scanErrors := 0

	// 扫描每个币种
	for _, symbol := range discoverySymbols {
		totalScanned++

		// 获取现货价格
		spotPrice, spotErr := h.server.getCurrentPrice(batchCtx, symbol, "spot")
		if spotErr != nil {
			scanErrors++
			continue
		}

		// 获取期货价格
		futuresPrice, futuresErr := h.server.getCurrentPrice(batchCtx, symbol, "futures")
		if futuresErr != nil {
			scanErrors++
			continue
		}

		// 计算价差
		priceDiff := futuresPrice - spotPrice
		priceDiffPercent := (priceDiff / spotPrice) * 100

		// 检查是否有套利机会（使用0.5%的阈值）
		if math.Abs(priceDiffPercent) >= 0.5 {
			action := "arb_buy_futures_sell_spot"
			if priceDiffPercent > 0 {
				action = "arb_sell_futures_buy_spot"
			}

			opportunity := ArbitrageOpportunity{
				Symbol:           symbol,
				Action:           action,
				Reason:           fmt.Sprintf("期现价差%.2f%%", priceDiffPercent),
				PriceDiffPercent: priceDiffPercent,
				SpotPrice:        spotPrice,
				FuturesPrice:     futuresPrice,
			}

			opportunities = append(opportunities, opportunity)
		}
	}

	// 按价差绝对值排序，最好的机会排在前面
	sort.Slice(opportunities, func(i, j int) bool {
		return math.Abs(opportunities[i].PriceDiffPercent) > math.Abs(opportunities[j].PriceDiffPercent)
	})

	log.Printf("[DiscoverArbitrage] 扫描完成，共扫描%d个币种，发现%d个套利机会，%d个扫描错误",
		totalScanned, len(opportunities), scanErrors)

	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"total_scanned":       totalScanned,
		"opportunities":       opportunities,
		"opportunities_count": len(opportunities),
		"scan_errors":         scanErrors,
		"scan_timestamp":      time.Now().Unix(),
	})
}

// ============================================================================
// 扫描锁相关方法
// ============================================================================

// acquireScanLock 尝试获取扫描锁（非阻塞）
func (h *StrategyHandler) acquireScanLock() bool {
	// 使用channel实现非阻塞锁检查
	ch := make(chan bool, 1)
	go func() {
		h.server.scanMutex.Lock()
		ch <- true
	}()

	select {
	case <-ch:
		return true
	case <-time.After(100 * time.Millisecond): // 100ms超时
		return false
	}
}

// releaseScanLock 释放扫描锁
func (h *StrategyHandler) releaseScanLock() {
	h.server.scanMutex.Unlock()
}

// ============================================================================
// 辅助函数
// ============================================================================

// 辅助函数：从map中安全获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// 辅助函数：从map中安全获取float64值
func getFloat64Value(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0.0
}
