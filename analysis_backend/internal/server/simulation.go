package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"analysis/internal/netutil"
	"analysis/internal/price"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

const maxPriceHistoryDays = 365

// CreateSimulatedTrade 创建模拟交易
// POST /recommendations/simulation/trade
func (s *Server) CreateSimulatedTrade(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req struct {
		RecommendationID *uint  `json:"recommendation_id"`
		Symbol           string `json:"symbol"`
		BaseSymbol       string `json:"base_symbol"`
		Kind             string `json:"kind"`
		Quantity         string `json:"quantity"`
		Price            string `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Symbol == "" || req.Quantity == "" || req.Price == "" {
		s.ValidationError(c, "", "symbol、quantity和price不能为空")
		return
	}

	if req.Kind == "" {
		req.Kind = "spot"
	}

	// 计算总价值
	quantity, _ := strconv.ParseFloat(req.Quantity, 64)
	price, _ := strconv.ParseFloat(req.Price, 64)
	totalValue := quantity * price

	trade := &pdb.SimulatedTrade{
		UserID:           uid,
		RecommendationID: req.RecommendationID,
		Symbol:           strings.ToUpper(req.Symbol),
		BaseSymbol:       strings.ToUpper(req.BaseSymbol),
		Kind:             strings.ToLower(req.Kind),
		Side:             "BUY",
		Quantity:         req.Quantity,
		Price:            req.Price,
		TotalValue:       fmt.Sprintf("%.8f", totalValue),
		IsOpen:           true,
		CurrentPrice:     &req.Price, // 初始价格等于买入价格
	}

	if err := pdb.CreateSimulatedTrade(s.db.DB(), trade); err != nil {
		s.DatabaseError(c, "创建模拟交易", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": trade.ID})
}

// GetSimulatedTrades 获取模拟交易列表
// GET /recommendations/simulation/trades?is_open=true
func (s *Server) GetSimulatedTrades(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var isOpen *bool
	if isOpenStr := c.Query("is_open"); isOpenStr != "" {
		if isOpenStr == "true" {
			val := true
			isOpen = &val
		} else if isOpenStr == "false" {
			val := false
			isOpen = &val
		}
	}

	trades, err := pdb.GetSimulatedTrades(s.db.DB(), uid, isOpen)
	if err != nil {
		s.DatabaseError(c, "查询模拟交易", err)
		return
	}

	// 计算并设置当前价格与未实现盈亏
	priceMap := map[string]float64{}
	if s.cfg != nil && s.cfg.Pricing.Enable {
		symbolSet := map[string]struct{}{}
		for _, trade := range trades {
			if trade.IsOpen && trade.BaseSymbol != "" {
				symbolSet[strings.ToUpper(trade.BaseSymbol)] = struct{}{}
			}
		}

		if len(symbolSet) > 0 {
			symbols := make([]string, 0, len(symbolSet))
			for sym := range symbolSet {
				symbols = append(symbols, sym)
			}
			fetched, fetchErr := price.FetchPrices(c.Request.Context(), *s.cfg, symbols)
			if fetchErr != nil {
				log.Printf("[warn] 获取模拟交易价格失败: %v", fetchErr)
			} else {
				priceMap = fetched
			}
		}
	}

	for i := range trades {
		trade := &trades[i]
		if !trade.IsOpen || trade.BaseSymbol == "" {
			continue
		}
		currentPrice, ok := priceMap[strings.ToUpper(trade.BaseSymbol)]
		if !ok {
			continue
		}
		priceStr := fmt.Sprintf("%.8f", currentPrice)
		trade.CurrentPrice = ptrString(priceStr)

		buyPrice, err1 := strconv.ParseFloat(trade.Price, 64)
		quantity, err2 := strconv.ParseFloat(trade.Quantity, 64)
		if err1 != nil || err2 != nil || buyPrice == 0 {
			continue
		}

		unrealized := (currentPrice - buyPrice) * quantity
		unrealizedPercent := ((currentPrice - buyPrice) / buyPrice) * 100
		trade.UnrealizedPnl = ptrString(fmt.Sprintf("%.8f", unrealized))
		trade.UnrealizedPnlPercent = ptrFloat64(unrealizedPercent)
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
		"total":  len(trades),
	})
}

// CloseSimulatedTrade 平仓模拟交易
// POST /recommendations/simulation/trades/:id/close
func (s *Server) CloseSimulatedTrade(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的ID")
		return
	}

	var req struct {
		Price string `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if req.Price == "" {
		s.ValidationError(c, "price", "卖出价格不能为空")
		return
	}

	trade, err := pdb.GetSimulatedTradeByID(s.db.DB(), uint(id), uid)
	if err != nil {
		s.NotFound(c, "交易不存在")
		return
	}

	if !trade.IsOpen {
		s.ValidationError(c, "status", "交易已平仓")
		return
	}

	// 计算盈亏
	buyPrice, _ := strconv.ParseFloat(trade.Price, 64)
	sellPrice, _ := strconv.ParseFloat(req.Price, 64)
	quantity, _ := strconv.ParseFloat(trade.Quantity, 64)

	realizedPnl := (sellPrice - buyPrice) * quantity
	realizedPnlPercent := ((sellPrice - buyPrice) / buyPrice) * 100

	now := time.Now().UTC()
	trade.IsOpen = false
	trade.SoldAt = &now
	sellPriceStr := req.Price
	trade.CurrentPrice = &sellPriceStr
	realizedPnlStr := fmt.Sprintf("%.8f", realizedPnl)
	trade.RealizedPnl = &realizedPnlStr
	realizedPnlPercentVal := realizedPnlPercent
	trade.RealizedPnlPercent = &realizedPnlPercentVal

	if err := pdb.UpdateSimulatedTrade(s.db.DB(), trade); err != nil {
		s.DatabaseError(c, "平仓交易", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated":              1,
		"realized_pnl":         realizedPnl,
		"realized_pnl_percent": realizedPnlPercent,
	})
}

// UpdateSimulatedTradePrice 更新模拟交易当前价格（用于实时更新盈亏）
// POST /recommendations/simulation/trades/:id/update-price
func (s *Server) UpdateSimulatedTradePrice(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		s.ValidationError(c, "id", "无效的ID")
		return
	}

	var req struct {
		CurrentPrice string `json:"current_price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	trade, err := pdb.GetSimulatedTradeByID(s.db.DB(), uint(id), uid)
	if err != nil {
		s.NotFound(c, "交易不存在")
		return
	}

	if !trade.IsOpen {
		s.ValidationError(c, "status", "交易已平仓，无法更新价格")
		return
	}

	// 更新当前价格和未实现盈亏
	buyPrice, _ := strconv.ParseFloat(trade.Price, 64)
	currentPrice, _ := strconv.ParseFloat(req.CurrentPrice, 64)
	quantity, _ := strconv.ParseFloat(trade.Quantity, 64)

	unrealizedPnl := (currentPrice - buyPrice) * quantity
	unrealizedPnlPercent := ((currentPrice - buyPrice) / buyPrice) * 100

	trade.CurrentPrice = &req.CurrentPrice
	unrealizedPnlStr := fmt.Sprintf("%.8f", unrealizedPnl)
	trade.UnrealizedPnl = &unrealizedPnlStr
	unrealizedPnlPercentVal := unrealizedPnlPercent
	trade.UnrealizedPnlPercent = &unrealizedPnlPercentVal

	if err := pdb.UpdateSimulatedTrade(s.db.DB(), trade); err != nil {
		s.DatabaseError(c, "更新价格", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"updated":                1,
		"unrealized_pnl":         unrealizedPnl,
		"unrealized_pnl_percent": unrealizedPnlPercent,
	})
}

// GetMarketPriceHistory 返回指定币种的价格历史（支持K线图）
func (s *Server) GetMarketPriceHistory(c *gin.Context) {
	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	if symbol == "" {
		s.ValidationError(c, "symbol", "币种不能为空")
		return
	}

	if s.cfg == nil || !s.cfg.Pricing.Enable {
		s.ValidationError(c, "pricing", "未配置价格服务")
		return
	}

	coinID := s.cfg.Pricing.Map[symbol]
	if coinID == "" {
		// 尝试从CoinGecko搜索币种ID（自动查找）
		var err error
		coinID, err = s.findCoinGeckoID(c.Request.Context(), symbol)
		if err != nil || coinID == "" {
			// 如果自动查找失败，返回友好的错误提示
			s.NotFound(c, fmt.Sprintf("币种 %s 不在价格服务支持列表中。请在配置文件的 pricing.map 中添加映射，例如：%s: coin-id", symbol, symbol))
			return
		}
		// 可选：记录日志，提示管理员添加配置
		log.Printf("[INFO] 自动查找到币种 %s 的 CoinGecko ID: %s，建议添加到配置文件", symbol, coinID)
	}

	daysStr := strings.TrimSpace(c.DefaultQuery("days", "30"))
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}
	if days > maxPriceHistoryDays {
		days = maxPriceHistoryDays
	}

	// CoinGecko API说明：
	// - days <= 1: 必须使用 interval=hourly
	// - 2 <= days <= 90: 自动返回hourly数据，不需要指定interval（Enterprise计划才能显式指定interval=hourly）
	// - days > 90: 自动使用daily间隔
	// 因此我们只在days <= 1时指定interval=hourly，其他情况不指定interval参数

	var url string
	endpoint := s.cfg.Pricing.CoinGeckoEndpoint
	// CoinGecko API路径：/api/v3/coins/{id}/market_chart
	// 如果endpoint是 /api/v3/simple/price，需要提取基础URL
	baseURL := endpoint
	if strings.Contains(endpoint, "/api/v3") {
		// 移除 /api/v3 及其后面的路径，保留基础域名
		parts := strings.Split(endpoint, "/api/v3")
		if len(parts) > 0 {
			baseURL = strings.TrimSuffix(parts[0], "/")
		}
	}

	// 只在days <= 1时指定interval=hourly，其他情况让API自动选择
	if days <= 1 {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d&interval=hourly", baseURL, coinID, days)
	} else {
		// 2-90天会自动返回hourly数据，>90天会自动使用daily
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d", baseURL, coinID, days)
	}

	var resp struct {
		Prices       [][]float64 `json:"prices"`
		MarketCaps   [][]float64 `json:"market_caps"`
		TotalVolumes [][]float64 `json:"total_volumes"`
	}

	// 创建带超时的context（30秒超时，足够CoinGecko API响应）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 重试机制：最多重试3次
	var lastErr error
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：1秒, 2秒, 4秒
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("[retry] 获取价格历史失败，%v后重试 (尝试 %d/%d): %v", delay, attempt+1, maxRetries, lastErr)
			select {
			case <-ctx.Done():
				s.InternalServerError(c, "获取价格历史失败", ctx.Err())
				return
			case <-time.After(delay):
			}
		}

		err := netutil.GetJSON(ctx, url, &resp)
		if err == nil {
			break // 成功
		}
		lastErr = err

		// 检查是否是超时或网络错误，如果是则重试
		errStr := strings.ToLower(err.Error())
		isTimeout := strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "tls handshake timeout")
		isNetworkErr := strings.Contains(errStr, "connection") || strings.Contains(errStr, "eof") || isTimeout

		if !isNetworkErr || attempt == maxRetries-1 {
			// 非网络错误或最后一次尝试，直接返回错误
			s.InternalServerError(c, "获取价格历史失败", err)
			return
		}
	}

	if lastErr != nil {
		s.InternalServerError(c, "获取价格历史失败", lastErr)
		return
	}

	// 确定实际使用的间隔（用于响应）
	actualInterval := "daily"
	if days <= 1 {
		actualInterval = "hourly"
	} else if days <= 90 {
		actualInterval = "hourly" // CoinGecko在2-90天时自动返回hourly数据
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":        symbol,
		"coin_id":       coinID,
		"days":          days,
		"interval":      actualInterval,
		"prices":        resp.Prices,
		"market_caps":   resp.MarketCaps,
		"total_volumes": resp.TotalVolumes,
	})
}

func ptrString(v string) *string {
	str := v
	return &str
}

// GetAutoExecuteSettings 获取用户的自动执行设置
// GET /user/auto-execute/settings
func (s *Server) GetAutoExecuteSettings(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	settings, err := pdb.GetOrCreateAutoExecuteSettings(s.db.DB(), uid)
	if err != nil {
		s.DatabaseError(c, "获取自动执行设置", err)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateAutoExecuteSettings 更新用户的自动执行设置
// PUT /user/auto-execute/settings
func (s *Server) UpdateAutoExecuteSettings(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req struct {
		Enabled        bool     `json:"enabled"`
		RiskLevel      string   `json:"risk_level"`
		MaxPosition    float64  `json:"max_position"`
		MinConfidence  float64  `json:"min_confidence"`
		MaxDailyTrades int      `json:"max_daily_trades"`
		Symbols        []string `json:"symbols"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	// 验证参数
	if req.RiskLevel != "" && req.RiskLevel != "conservative" && req.RiskLevel != "moderate" && req.RiskLevel != "aggressive" {
		s.ValidationError(c, "risk_level", "风险等级只能是: conservative, moderate, aggressive")
		return
	}

	if req.MaxPosition < 0.1 || req.MaxPosition > 20 {
		s.ValidationError(c, "max_position", "最大单次仓位必须在0.1%-20%之间")
		return
	}

	if req.MinConfidence < 0.5 || req.MinConfidence > 1.0 {
		s.ValidationError(c, "min_confidence", "最小置信度必须在0.5-1.0之间")
		return
	}

	if req.MaxDailyTrades < 1 || req.MaxDailyTrades > 20 {
		s.ValidationError(c, "max_daily_trades", "每日最大交易次数必须在1-20之间")
		return
	}

	// 将symbols数组转换为逗号分隔的字符串
	symbolsStr := ""
	if len(req.Symbols) > 0 {
		symbolsStr = strings.Join(req.Symbols, ",")
	}

	settings := &pdb.AutoExecuteSettings{
		Enabled:        req.Enabled,
		RiskLevel:      req.RiskLevel,
		MaxPosition:    req.MaxPosition,
		MinConfidence:  req.MinConfidence,
		MaxDailyTrades: req.MaxDailyTrades,
		Symbols:        symbolsStr,
	}

	if err := pdb.UpdateAutoExecuteSettings(s.db.DB(), uid, settings); err != nil {
		s.DatabaseError(c, "更新自动执行设置", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "设置已更新"})
}

// ExecuteRecommendations 自动执行推荐
// POST /recommendations/auto-execute
func (s *Server) ExecuteRecommendations(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req struct {
		Date      string   `json:"date"`       // YYYY-MM-DD格式，可选
		Symbols   []string `json:"symbols"`    // 可选，指定要执行的币种
		RiskLevel string   `json:"risk_level"` // 可选，覆盖用户设置
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	// 获取用户设置
	settings, err := pdb.GetOrCreateAutoExecuteSettings(s.db.DB(), uid)
	if err != nil {
		s.DatabaseError(c, "获取用户设置", err)
		return
	}

	if !settings.Enabled {
		s.ValidationError(c, "auto_execute", "自动执行未启用，请先在设置中启用")
		return
	}

	// 获取推荐数据
	var recommendations []gin.H
	if req.Date != "" {
		// 获取指定日期的历史推荐
		resp, err := s.getHistoricalRecommendationsForDate(req.Date, "spot", req.Symbols)
		if err != nil {
			s.InternalServerError(c, "获取历史推荐", err)
			return
		}
		recommendations = resp
	} else {
		// 获取当前推荐 - GetAIRecommendations会直接处理响应，这里不执行
		s.InternalServerError(c, "暂不支持当前推荐自动执行", fmt.Errorf("请指定日期"))
		return
	}

	// 应用风险等级过滤
	riskLevel := req.RiskLevel
	if riskLevel == "" {
		riskLevel = settings.RiskLevel
	}

	// 过滤符合条件的推荐
	executableRecommendations := s.filterExecutableRecommendations(recommendations, settings, riskLevel)

	if len(executableRecommendations) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":        "没有符合执行条件的推荐",
			"executed_count": 0,
		})
		return
	}

	// 检查今日交易次数限制
	today := time.Now().Format("2006-01-02")
	todayTrades, err := s.getTodayTradeCount(uid, today)
	if err != nil {
		s.DatabaseError(c, "检查今日交易次数", err)
		return
	}

	remainingTrades := settings.MaxDailyTrades - todayTrades
	if remainingTrades <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":      "今日交易次数已达上限",
			"today_trades": todayTrades,
			"max_trades":   settings.MaxDailyTrades,
		})
		return
	}

	// 执行交易
	executedCount := 0
	maxExecutions := min(remainingTrades, len(executableRecommendations))

	for i := 0; i < maxExecutions; i++ {
		rec := executableRecommendations[i]
		if err := s.executeRecommendationAsTrade(uid, rec, settings); err != nil {
			log.Printf("[WARN] 执行推荐失败: %v, 推荐: %+v", err, rec)
			continue
		}
		executedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          fmt.Sprintf("成功执行 %d 个推荐", executedCount),
		"executed_count":   executedCount,
		"total_candidates": len(executableRecommendations),
		"daily_limit":      settings.MaxDailyTrades,
		"today_trades":     todayTrades + executedCount,
	})
}

// filterExecutableRecommendations 过滤可执行的推荐
func (s *Server) filterExecutableRecommendations(recommendations []gin.H, settings *pdb.AutoExecuteSettings, riskLevel string) []gin.H {
	var executable []gin.H

	// 解析允许的币种列表
	allowedSymbols := make(map[string]bool)
	if settings.Symbols != "" {
		symbols := strings.Split(settings.Symbols, ",")
		for _, symbol := range symbols {
			allowedSymbols[strings.TrimSpace(strings.ToUpper(symbol))] = true
		}
	}

	for _, rec := range recommendations {
		symbol, ok := rec["symbol"].(string)
		if !ok {
			continue
		}

		// 检查币种是否在允许列表中
		if len(allowedSymbols) > 0 && !allowedSymbols[strings.ToUpper(symbol)] {
			continue
		}

		// 检查置信度
		confidence, ok := rec["ml_confidence"].(float64)
		if !ok || confidence < settings.MinConfidence {
			continue
		}

		// 根据风险等级进行额外过滤
		if !s.isRecommendationSuitableForRiskLevel(rec, riskLevel) {
			continue
		}

		executable = append(executable, rec)
	}

	return executable
}

// isRecommendationSuitableForRiskLevel 检查推荐是否适合当前风险等级
func (s *Server) isRecommendationSuitableForRiskLevel(rec gin.H, riskLevel string) bool {
	riskScore, ok := rec["risk_score"].(float64)
	if !ok {
		return false
	}

	switch riskLevel {
	case "conservative":
		// 保守策略：只执行低风险推荐
		return riskScore <= 0.3
	case "moderate":
		// 稳健策略：执行中等风险推荐
		return riskScore <= 0.6
	case "aggressive":
		// 激进策略：执行所有推荐
		return true
	default:
		return riskScore <= 0.5
	}
}

// getTodayTradeCount 获取用户今日交易次数
func (s *Server) getTodayTradeCount(userID uint, today string) (int, error) {
	var count int64
	startOfDay := today + " 00:00:00"
	endOfDay := today + " 23:59:59"

	err := s.db.DB().Model(&pdb.SimulatedTrade{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startOfDay, endOfDay).
		Count(&count).Error

	return int(count), err
}

// executeRecommendationAsTrade 执行推荐作为交易
func (s *Server) executeRecommendationAsTrade(userID uint, rec gin.H, settings *pdb.AutoExecuteSettings) error {
	symbol, _ := rec["symbol"].(string)
	price, _ := rec["price"].(float64)
	riskScore, _ := rec["risk_score"].(float64)

	// 获取市场波动率等额外因素
	marketVolatility := s.getMarketVolatility(symbol)
	marketTrend := s.getMarketTrend(symbol)

	// 根据风险评分和用户设置计算智能仓位
	positionSize := s.calculateSmartPositionSize(riskScore, settings, marketVolatility, marketTrend)

	// 检查当前持仓情况，避免过度集中
	currentExposure := s.getCurrentPortfolioExposure(userID, symbol)
	maxAllowedExposure := settings.MaxPosition / 100.0 * 1.5 // 允许一定程度的超限
	if currentExposure+positionSize > maxAllowedExposure {
		positionSize = math.Max(0, maxAllowedExposure-currentExposure)
	}

	if positionSize <= 0 {
		return fmt.Errorf("仓位调整后无可用仓位")
	}

	// 计算交易数量（基于价格和仓位）
	totalValue := positionSize * 10000 // 假设总资金10,000 USD
	quantity := totalValue / price

	currentPrice := fmt.Sprintf("%.8f", price)
	trade := &pdb.SimulatedTrade{
		UserID:       userID,
		Symbol:       strings.ToUpper(symbol),
		BaseSymbol:   "USDT", // 假设交易对是XXX/USDT
		Kind:         "spot",
		Side:         "BUY",
		Quantity:     fmt.Sprintf("%.8f", quantity),
		Price:        fmt.Sprintf("%.8f", price),
		TotalValue:   fmt.Sprintf("%.8f", totalValue),
		IsOpen:       true,
		CurrentPrice: &currentPrice, // 设置当前价格
	}

	return pdb.CreateSimulatedTrade(s.db.DB(), trade)
}

// calculatePositionSize 根据风险评分和用户设置计算仓位大小
func (s *Server) calculatePositionSize(riskScore float64, settings *pdb.AutoExecuteSettings) float64 {
	// 基础仓位 = 用户设置的最大仓位
	basePosition := settings.MaxPosition / 100.0 // 转换为小数

	var position float64

	// 根据风险等级采用不同的仓位策略
	switch settings.RiskLevel {
	case "conservative":
		// 保守策略：风险评分越低，仓位越大，但总体偏小
		if riskScore <= 0.2 {
			position = basePosition * 0.8 // 低风险，高仓位
		} else if riskScore <= 0.4 {
			position = basePosition * 0.5 // 中低风险，中等仓位
		} else if riskScore <= 0.6 {
			position = basePosition * 0.2 // 中等风险，低仓位
		} else {
			position = 0 // 高风险，不执行
		}

	case "moderate":
		// 稳健策略：平衡风险与收益
		if riskScore <= 0.3 {
			position = basePosition * 0.9 // 低风险，高仓位
		} else if riskScore <= 0.5 {
			position = basePosition * 0.6 // 中等风险，中等仓位
		} else if riskScore <= 0.7 {
			position = basePosition * 0.3 // 中高风险，低仓位
		} else {
			position = basePosition * 0.1 // 高风险，极低仓位
		}

	case "aggressive":
		// 激进策略：追求收益，接受较高风险
		if riskScore <= 0.4 {
			position = basePosition // 低风险，全仓位
		} else if riskScore <= 0.6 {
			position = basePosition * 0.7 // 中等风险，高仓位
		} else if riskScore <= 0.8 {
			position = basePosition * 0.4 // 高风险，中等仓位
		} else {
			position = basePosition * 0.2 // 极高风险，低仓位
		}

	default:
		// 默认中等策略
		position = basePosition * (1.0 - riskScore)
	}

	// 应用最小仓位限制
	minPosition := 0.001 // 0.1%
	if position < minPosition {
		position = minPosition
	}

	// 确保不超过最大仓位
	if position > basePosition {
		position = basePosition
	}

	return position
}

// calculateSmartPositionSize 智能计算仓位大小，考虑多个因素
func (s *Server) calculateSmartPositionSize(riskScore float64, settings *pdb.AutoExecuteSettings, marketVolatility float64, marketTrend string) float64 {
	// 基础仓位计算
	basePosition := s.calculatePositionSize(riskScore, settings)

	// 波动率调整因子
	var volatilityFactor float64 = 1.0
	if marketVolatility > 0 {
		if marketVolatility > 0.05 { // 高波动
			volatilityFactor = 0.7
		} else if marketVolatility > 0.03 { // 中等波动
			volatilityFactor = 0.85
		} else { // 低波动
			volatilityFactor = 1.0
		}
	}

	// 趋势调整因子
	var trendFactor float64 = 1.0
	switch marketTrend {
	case "bull_strong":
		trendFactor = 1.1 // 强势上涨，增加仓位
	case "bull_moderate":
		trendFactor = 1.0 // 温和上涨，保持仓位
	case "sideways":
		trendFactor = 0.9 // 震荡，减少仓位
	case "bear_moderate":
		trendFactor = 0.7 // 温和下跌，大幅减少仓位
	case "bear_strong":
		trendFactor = 0.5 // 强势下跌，最小仓位
	default:
		trendFactor = 1.0
	}

	// 应用调整因子
	smartPosition := basePosition * volatilityFactor * trendFactor

	// 确保在合理范围内
	minPosition := 0.0005                             // 0.05%
	maxPosition := settings.MaxPosition / 100.0 * 1.2 // 允许一定程度的超限

	if smartPosition < minPosition {
		smartPosition = minPosition
	}
	if smartPosition > maxPosition {
		smartPosition = maxPosition
	}

	return smartPosition
}

// getMarketVolatility 获取市场波动率
func (s *Server) getMarketVolatility(symbol string) float64 {
	ctx := context.Background()

	// 获取最近24小时的K线数据来计算波动率
	klines, err := s.fetchBinanceKlines(ctx, symbol, "spot", "1h", 24)
	if err != nil || len(klines) < 2 {
		return 0.03 // 默认中等波动率
	}

	// 计算价格的标准差作为波动率指标
	var prices []float64
	for _, kline := range klines {
		if closePrice, err := strconv.ParseFloat(kline.Close, 64); err == nil {
			prices = append(prices, closePrice)
		}
	}

	if len(prices) < 2 {
		return 0.03
	}

	// 计算标准差
	mean := 0.0
	for _, price := range prices {
		mean += price
	}
	mean /= float64(len(prices))

	variance := 0.0
	for _, price := range prices {
		variance += (price - mean) * (price - mean)
	}
	variance /= float64(len(prices))

	volatility := math.Sqrt(variance) / mean // 系数波动率

	return math.Min(volatility, 0.2) // 限制最大波动率为20%
}

// getMarketTrend 获取市场趋势
func (s *Server) getMarketTrend(symbol string) string {
	ctx := context.Background()

	// 获取技术指标来判断趋势
	indicators, err := s.CalculateTechnicalIndicators(ctx, symbol, "spot")
	if err != nil {
		return "sideways" // 默认震荡
	}

	trend := indicators.Trend

	// 根据技术指标判断趋势强度
	switch trend {
	case "up":
		// 检查是否强势上涨
		if indicators.RSI > 70 && indicators.MACD > indicators.MACDSignal {
			return "bull_strong"
		}
		return "bull_moderate"
	case "down":
		// 检查是否强势下跌
		if indicators.RSI < 30 && indicators.MACD < indicators.MACDSignal {
			return "bear_strong"
		}
		return "bear_moderate"
	default:
		return "sideways"
	}
}

// getCurrentPortfolioExposure 获取当前投资组合中某币种的暴露度
func (s *Server) getCurrentPortfolioExposure(userID uint, symbol string) float64 {
	var totalValue float64
	var currentPositions []pdb.SimulatedTrade

	// 获取用户当前未平仓的交易
	err := s.db.DB().Where("user_id = ? AND is_open = ? AND symbol = ?", userID, true, strings.ToUpper(symbol)).Find(&currentPositions).Error
	if err != nil {
		return 0
	}

	// 计算总暴露价值（假设总资金10,000 USD）
	totalFunds := 10000.0
	for _, position := range currentPositions {
		if quantity, err := strconv.ParseFloat(position.Quantity, 64); err == nil {
			if price, err := strconv.ParseFloat(position.Price, 64); err == nil {
				totalValue += quantity * price
			}
		}
	}

	return totalValue / totalFunds // 返回占总资金的比例
}

// getHistoricalRecommendationsForDate 获取指定日期的历史推荐
func (s *Server) getHistoricalRecommendationsForDate(date string, kind string, symbols []string) ([]gin.H, error) {
	// 这里应该调用现有的历史推荐API逻辑
	// 暂时返回空数组，实际实现需要调用recommendation_performance.go中的逻辑
	return []gin.H{}, nil
}

// 辅助函数已移除，使用math包中的函数

// findCoinGeckoID 从CoinGecko搜索API查找币种ID
func (s *Server) findCoinGeckoID(ctx context.Context, symbol string) (string, error) {
	if s.cfg == nil || !s.cfg.Pricing.Enable {
		return "", fmt.Errorf("价格服务未启用")
	}

	endpoint := s.cfg.Pricing.CoinGeckoEndpoint
	baseURL := endpoint
	if strings.Contains(endpoint, "/api/v3") {
		parts := strings.Split(endpoint, "/api/v3")
		if len(parts) > 0 {
			baseURL = strings.TrimSuffix(parts[0], "/")
		}
	}

	// 使用CoinGecko搜索API
	url := fmt.Sprintf("%s/api/v3/search?query=%s", baseURL, symbol)

	var resp struct {
		Coins []struct {
			ID     string `json:"id"`
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		} `json:"coins"`
	}

	// 创建带超时的context（5秒超时）
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := netutil.GetJSON(ctxWithTimeout, url, &resp)
	if err != nil {
		return "", fmt.Errorf("搜索币种失败: %w", err)
	}

	if len(resp.Coins) == 0 {
		return "", fmt.Errorf("未找到币种 %s", symbol)
	}

	// 优先精确匹配symbol（不区分大小写）
	symbolUpper := strings.ToUpper(symbol)
	for _, coin := range resp.Coins {
		if strings.ToUpper(coin.Symbol) == symbolUpper {
			return coin.ID, nil
		}
	}

	// 如果没有精确匹配，返回第一个结果
	return resp.Coins[0].ID, nil
}

func ptrFloat64(v float64) *float64 {
	val := v
	return &val
}

// ClearUserTrades 清理用户的模拟交易记录
// DELETE /recommendations/simulations/trades
func (s *Server) ClearUserTrades(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	// 删除用户的所有模拟交易
	result := s.db.DB().Where("user_id = ?", uid).Delete(&pdb.SimulatedTrade{})
	if result.Error != nil {
		s.DatabaseError(c, "清理模拟交易", result.Error)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "成功清理模拟交易记录",
		"deleted_count": result.RowsAffected,
	})
}
