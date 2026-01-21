package server

import (
	pdb "analysis/internal/db"
	"analysis/internal/netutil"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// MarketDepth 市场深度数据结构
type MarketDepth struct {
	Symbol    string      `json:"symbol"`
	Bids      [][]float64 `json:"bids"`      // 买单 [价格, 数量]
	Asks      [][]float64 `json:"asks"`      // 卖单 [价格, 数量]
	Timestamp int64       `json:"timestamp"` // 时间戳
}

// GridPosition 网格持仓信息
type GridPosition struct {
	Symbol       string  `json:"symbol"`
	Level        int     `json:"level"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	Side         string  `json:"side"`
	Timestamp    int64   `json:"timestamp"`
	GridRange    string  `json:"grid_range,omitempty"`
	ProfitTarget float64 `json:"profit_target,omitempty"`
}

// GridRiskManager 网格风险管理器
type GridRiskManager struct {
	MaxDrawdown          float64 `json:"max_drawdown"`
	MaxPositionSize      float64 `json:"max_position_size"`
	StopLossMultiplier   float64 `json:"stop_loss_multiplier"`
	TakeProfitMultiplier float64 `json:"take_profit_multiplier"`
}

// DynamicGridRange 动态网格范围
type DynamicGridRange struct {
	UpperPrice float64 `json:"upper_price"`
	LowerPrice float64 `json:"lower_price"`
	Levels     int     `json:"levels"`
	Reason     string  `json:"reason"`
}

// 给采集进程写的入口：POST /ingest/binance/market
func (s *Server) IngestBinanceMarket(c *gin.Context) {
	var body struct {
		Kind      string `json:"kind"`
		Bucket    string `json:"bucket"`
		FetchedAt string `json:"fetched_at"`
		Items     []struct {
			Symbol             string   `json:"symbol"`
			LastPrice          string   `json:"last_price"`
			Volume             string   `json:"volume"`
			PriceChangePercent float64  `json:"price_change_percent"`
			MarketCapUSD       *float64 `json:"market_cap_usd"`
			FDVUSD             *float64 `json:"fdv_usd"`
			CirculatingSupply  *float64 `json:"circulating_supply"`
			TotalSupply        *float64 `json:"total_supply"`
		} `json:"items"`
	}
	if err := c.BindJSON(&body); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if body.Kind == "" {
		body.Kind = "spot"
	}

	bucket, err := time.Parse(time.RFC3339, body.Bucket)
	if err != nil {
		s.BadRequest(c, "时间桶格式错误", err)
		return
	}

	fetchedAt := time.Now().UTC()
	if body.FetchedAt != "" {
		if t, e := time.Parse(time.RFC3339, body.FetchedAt); e == nil {
			fetchedAt = t
		}
	}

	// 存库统一用 UTC + 1h 对齐
	bucket = bucket.UTC().Truncate(1 * time.Hour)

	rows := make([]pdb.BinanceMarketTop, 0, len(body.Items))
	for i, it := range body.Items {
		rows = append(rows, pdb.BinanceMarketTop{
			Symbol:            it.Symbol,
			LastPrice:         it.LastPrice,
			Volume:            it.Volume,
			PctChange:         it.PriceChangePercent,
			Rank:              i + 1,
			MarketCapUSD:      it.MarketCapUSD,
			FDVUSD:            it.FDVUSD,
			CirculatingSupply: it.CirculatingSupply,
			TotalSupply:       it.TotalSupply,
		})
	}

	if _, err := pdb.SaveBinanceMarket(s.db.DB(), body.Kind, bucket, fetchedAt, rows); err != nil {
		s.DatabaseError(c, "保存市场数据", err)
		return
	}

	// 失效市场数据缓存，使新数据立即生效
	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// binanceMarketParams 市场查询参数
type binanceMarketParams struct {
	Kind        string
	IntervalMin int
	Location    *time.Location
	Date        string
	Slot        string
	Category    string // 新增：币种分类参数
}

// parseBinanceMarketParams 解析市场查询参数
func parseBinanceMarketParams(c *gin.Context) (*binanceMarketParams, error) {
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	if kind != "spot" && kind != "futures" {
		kind = "futures"
	}

	intervalMin := 120
	if v := c.Query("interval"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalMin = n
		}
	}

	tzName := c.Query("tz")
	if tzName == "" {
		tzName = "Asia/Taipei"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.FixedZone("CST-8", 8*3600)
	}

	return &binanceMarketParams{
		Kind:        kind,
		IntervalMin: intervalMin,
		Location:    loc,
		Date:        strings.TrimSpace(c.Query("date")),
		Slot:        strings.TrimSpace(c.Query("slot")),
		Category:    strings.TrimSpace(c.Query("category")),
	}, nil
}

// calculateTimeRange 计算时间范围
func calculateTimeRange(params *binanceMarketParams) (time.Time, time.Time, error) {
	if params.Date == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("date is required")
	}

	dayStartLocal, err := time.ParseInLocation("2006-01-02", params.Date, params.Location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("日期格式错误，应为 YYYY-MM-DD: %w", err)
	}

	var startLocal, endLocal time.Time
	if params.Slot != "" {
		slot, err := strconv.Atoi(params.Slot)
		if err != nil || slot < 0 || slot > (24*60/params.IntervalMin-1) {
			return time.Time{}, time.Time{}, fmt.Errorf("时间段编号无效")
		}
		startLocal = dayStartLocal.Add(time.Duration(slot) * time.Minute * time.Duration(params.IntervalMin))
		endLocal = startLocal.Add(time.Duration(params.IntervalMin) * time.Minute)
	} else {
		startLocal = dayStartLocal
		endLocal = dayStartLocal.Add(24 * time.Hour)
	}

	return startLocal.UTC(), endLocal.UTC(), nil
}

// filterAndFormatMarketData 过滤黑名单并格式化市场数据
func (s *Server) filterAndFormatMarketData(snaps []pdb.BinanceMarketSnapshot, tops map[uint][]pdb.BinanceMarketTop, kind string, ctx context.Context) ([]gin.H, error) {
	// 获取黑名单（现货和期货都支持）- 使用缓存
	blacklistMap, err := s.getCachedBlacklistMap(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get cached blacklist (kind=%s), falling back to direct query: %v", kind, err)
		// 缓存失败时降级到直接查询，但不影响主流程
		blacklistMap = make(map[string]bool)
		if blacklist, e := s.db.GetBinanceBlacklist(kind); e == nil {
			for _, symbol := range blacklist {
				blacklistMap[strings.ToUpper(symbol)] = true
			}
		}
	}

	// 优化：预估输出切片大小
	out := make([]gin.H, 0, len(snaps))
	for _, snap := range snaps {
		list := tops[snap.ID]
		// 过滤黑名单（Symbol 已是大写，直接使用）
		// 优化：预估过滤后的切片大小（假设最多保留10个）
		estimatedSize := len(list)
		if estimatedSize > 10 {
			estimatedSize = 10
		}
		filtered := make([]pdb.BinanceMarketTop, 0, estimatedSize)
		for _, it := range list {
			if !blacklistMap[it.Symbol] {
				filtered = append(filtered, it)
				// 优化：如果已经达到10个，提前退出循环
				if len(filtered) >= 10 {
					break
				}
			}
		}
		// 取前10个（如果超过10个）
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}
		// 优化：预估 items 切片大小
		items := make([]gin.H, 0, len(filtered))
		for _, it := range filtered {
			items = append(items, gin.H{
				"symbol":             it.Symbol,
				"last_price":         it.LastPrice,
				"volume":             it.Volume,
				"pct_change":         it.PctChange,
				"rank":               it.Rank,
				"market_cap_usd":     it.MarketCapUSD,
				"fdv_usd":            it.FDVUSD,
				"circulating_supply": it.CirculatingSupply,
				"total_supply":       it.TotalSupply,
			})
		}
		out = append(out, gin.H{
			"bucket":     snap.Bucket,    // UTC
			"fetched_at": snap.FetchedAt, // UTC
			"kind":       snap.Kind,
			"items":      items,
		})
	}
	return out, nil
}

// filterAndFormatMarketDataWithCategory 过滤黑名单和分类并格式化市场数据
func (s *Server) filterAndFormatMarketDataWithCategory(snaps []pdb.BinanceMarketSnapshot, tops map[uint][]pdb.BinanceMarketTop, kind string, category string, ctx context.Context) ([]gin.H, error) {
	// 获取黑名单（现货和期货都支持）- 使用缓存
	blacklistMap, err := s.getCachedBlacklistMap(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get cached blacklist (kind=%s), falling back to direct query: %v", kind, err)
		// 缓存失败时降级到直接查询，但不影响主流程
		blacklistMap = make(map[string]bool)
		if blacklist, e := s.db.GetBinanceBlacklist(kind); e == nil {
			for _, symbol := range blacklist {
				blacklistMap[strings.ToUpper(symbol)] = true
			}
		}
	}

	// 获取exchangeInfo数据用于分类筛选
	exchangeInfo, err := s.getExchangeInfoForCategory(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get exchange info for category filtering: %v", err)
		// 如果获取失败，继续处理但不进行分类筛选
	}

	// 优化：预估输出切片大小
	out := make([]gin.H, 0, len(snaps))
	for _, snap := range snaps {
		list := tops[snap.ID]

		// 首先过滤黑名单和分类
		filtered := s.filterMarketDataByCategoryAndBlacklist(list, blacklistMap, category, exchangeInfo, kind)

		// 取前10个（如果超过10个）
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}

		// 优化：预估 items 切片大小
		items := make([]gin.H, 0, len(filtered))
		for _, it := range filtered {
			items = append(items, gin.H{
				"symbol":             it.Symbol,
				"last_price":         it.LastPrice,
				"volume":             it.Volume,
				"pct_change":         it.PctChange,
				"rank":               it.Rank,
				"market_cap_usd":     it.MarketCapUSD,
				"fdv_usd":            it.FDVUSD,
				"circulating_supply": it.CirculatingSupply,
				"total_supply":       it.TotalSupply,
			})
		}
		out = append(out, gin.H{
			"bucket":     snap.Bucket,    // UTC
			"fetched_at": snap.FetchedAt, // UTC
			"kind":       snap.Kind,
			"items":      items,
		})
	}
	return out, nil
}

// filterMarketDataByCategoryAndBlacklist 根据分类和黑名单过滤市场数据
func (s *Server) filterMarketDataByCategoryAndBlacklist(list []pdb.BinanceMarketTop, blacklistMap map[string]bool, category string, exchangeInfo map[string]ExchangeInfoItem, kind string) []pdb.BinanceMarketTop {
	if category == "" || category == "all" {
		// 如果没有分类要求，只过滤黑名单
		filtered := make([]pdb.BinanceMarketTop, 0, len(list))
		for _, it := range list {
			if !blacklistMap[it.Symbol] {
				filtered = append(filtered, it)
				if len(filtered) >= 10 {
					break
				}
			}
		}
		return filtered
	}

	// 根据分类进行筛选
	filtered := make([]pdb.BinanceMarketTop, 0, len(list))
	matchedCount := 0
	for _, it := range list {
		// 先检查黑名单
		if blacklistMap[it.Symbol] {
			continue
		}

		// 根据分类进行筛选
		if s.matchesCategory(it.Symbol, category, exchangeInfo, kind) {
			filtered = append(filtered, it)
			matchedCount++
			if len(filtered) >= 10 {
				break
			}
		}
	}
	return filtered
}

// ExchangeInfoItem exchangeInfo中的交易对信息
type ExchangeInfoItem struct {
	Symbol      string   `json:"symbol"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
	BaseAsset   string   `json:"baseAsset"`
	QuoteAsset  string   `json:"quoteAsset"`
}

// matchesCategory 检查交易对是否匹配指定的分类
func (s *Server) matchesCategory(symbol, category string, exchangeInfo map[string]ExchangeInfoItem, kind string) bool {
	// 如果没有exchangeInfo数据，默认通过
	if exchangeInfo == nil {
		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	info, exists := exchangeInfo[symbol]
	if !exists {
		// 如果exchangeInfo中没有这个交易对，默认通过
		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	switch category {
	case "trading":
		result := info.Status == "TRADING"
		return result
	case "break":
		result := info.Status == "BREAK"
		return result
	case "major", "stable", "defi":
		// 使用exchangeInfo的baseAsset进行智能分类
		result := s.isAssetTypeMatch(info.BaseAsset, category)
		return result
	case "layer1":
		// Layer1资产特殊处理
		if info.BaseAsset != "" {
			result := s.isAssetTypeMatch(info.BaseAsset, category)
			return result
		}
		// 降级到基于交易对符号的检查
		layer1Assets := []string{"ATOM", "NEAR", "FTM", "ONE", "EGLD", "FLOW"}
		baseSymbol := s.getBaseSymbol(symbol)
		return s.containsString(layer1Assets, baseSymbol)
	case "meme":
		// Meme资产特殊处理
		if info.BaseAsset != "" {
			result := s.isAssetTypeMatch(info.BaseAsset, category)
			return result
		}
		// 降级到基于交易对符号的检查
		memeAssets := []string{"SHIB", "DOGE", "PEPE", "BONK", "WIF", "TURBO"}
		baseSymbol := s.getBaseSymbol(symbol)
		return s.containsString(memeAssets, baseSymbol)
	case "spot_only":
		result := s.containsString(info.Permissions, "SPOT") && !s.containsString(info.Permissions, "LEVERAGED")
		return result
	case "margin":
		result := s.containsString(info.Permissions, "MARGIN")
		return result
	case "leveraged":
		result := s.containsString(info.Permissions, "LEVERAGED")
		return result
	default:
		return true
	}
}

// matchesCategoryBySymbolOnly 仅基于交易对符号进行分类匹配（当没有exchangeInfo时使用）
func (s *Server) matchesCategoryBySymbolOnly(symbol, category string) bool {
	baseSymbol := s.getBaseSymbol(symbol)

	// 当没有exchangeInfo时，也使用智能分类
	return s.isAssetTypeMatch(baseSymbol, category)
}

// getBaseSymbol 获取交易对的基础币种
func (s *Server) getBaseSymbol(symbol string) string {
	// 去掉常见的后缀
	suffixes := []string{"USDT", "USDC", "BUSD", "BTC", "ETH", "BNB"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToUpper(symbol), suffix) {
			return strings.TrimSuffix(strings.ToUpper(symbol), suffix)
		}
	}
	return strings.ToUpper(symbol)
}

// containsString 检查字符串切片是否包含指定字符串
func (s *Server) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isAssetTypeMatch 基于资产类型进行智能分类
func (s *Server) isAssetTypeMatch(baseAsset, category string) bool {
	asset := strings.ToUpper(baseAsset)

	switch category {
	case "major":
		// 主流币种：基于市场认可度和交易量
		majorCoins := map[string]bool{
			"BTC": true, "ETH": true, "BNB": true, "ADA": true, "SOL": true,
			"DOT": true, "AVAX": true, "MATIC": true, "LINK": true, "LTC": true,
			"ALGO": true, "VET": true, "ICP": true, "FIL": true, "TRX": true,
			"ETC": true, "XLM": true, "THETA": true, "FTM": true, "HBAR": true,
		}
		return majorCoins[asset]

	case "stable":
		// 稳定币：基于是否为稳定币
		stableCoins := map[string]bool{
			"USDT": true, "USDC": true, "BUSD": true, "DAI": true, "TUSD": true,
			"USDP": true, "FRAX": true, "LUSD": true, "SUSD": true, "USDJ": true,
			"USTC": true, "CUSD": true, "EUROC": true, "XSGD": true, "CEUR": true,
		}
		return stableCoins[asset]

	case "defi":
		// DeFi代币：基于是否为去中心化金融协议代币
		defiTokens := map[string]bool{
			"UNI": true, "AAVE": true, "SUSHI": true, "COMP": true, "MKR": true,
			"SNX": true, "CRV": true, "YFI": true, "BAL": true, "REN": true,
			"LRC": true, "REP": true, "LDO": true, "APE": true, "GAL": true,
			"ENS": true, "GRT": true, "ANT": true, "STORJ": true, "BAT": true,
			"CREAM": true, "ALCX": true, "BADGER": true, "CVX": true, "FXS": true,
			"Tribe": true, "TRIBE": true, "RBN": true, "AURA": true, "PENDLE": true,
		}
		return defiTokens[asset]

	case "layer1":
		// Layer1公链：基于是否为一层区块链网络
		layer1Chains := map[string]bool{
			"ATOM": true, "NEAR": true, "FTM": true, "ONE": true, "EGLD": true,
			"FLOW": true, "MINA": true, "CELO": true, "KAVA": true, "SCRT": true,
			"GLMR": true, "MOVR": true, "CFG": true, "SDN": true, "ASTR": true,
			"ACA": true, "KAR": true, "BNC": true, "PKEX": true, "XPRT": true,
		}
		return layer1Chains[asset]

	case "meme":
		// Meme币：基于是否为模因币
		memeCoins := map[string]bool{
			"SHIB": true, "DOGE": true, "PEPE": true, "BONK": true, "WIF": true,
			"TURBO": true, "BALD": true, "DEGEN": true, "CUMMIES": true, "HODL": true,
			"MEW": true, "PUMP": true, "NEIRO": true, "BRETT": true, "COTI": true,
			"FOXY": true, "GROYPER": true, "HYPER": true, "KEKE": true, "LANDWOLF": true,
		}
		return memeCoins[asset]

	default:
		return false
	}
}

// getExchangeInfoForCategory 获取exchangeInfo数据用于分类筛选
func (s *Server) getExchangeInfoForCategory(ctx context.Context, kind string) (map[string]ExchangeInfoItem, error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("exchange_info_%s", kind)
	if cached, exists := s.getCachedExchangeInfo(cacheKey); exists {
		return cached, nil
	}

	// 从币安API获取exchangeInfo
	var url string
	switch kind {
	case "spot":
		url = "https://api.binance.com/api/v3/exchangeInfo"
	case "futures":
		url = "https://fapi.binance.com/fapi/v1/exchangeInfo"
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}

	var response struct {
		Symbols []ExchangeInfoItem `json:"symbols"`
	}

	if err := netutil.GetJSON(ctx, url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch exchange info: %w", err)
	}

	log.Printf("[ExchangeInfo] 从 %s 获取到 %d 个交易对信息", url, len(response.Symbols))

	// 转换为map以便快速查找
	exchangeInfoMap := make(map[string]ExchangeInfoItem)
	for _, symbol := range response.Symbols {
		exchangeInfoMap[symbol.Symbol] = symbol
	}

	// 缓存结果（缓存1小时）
	s.cacheExchangeInfo(cacheKey, exchangeInfoMap, time.Hour)

	return exchangeInfoMap, nil
}

// getCachedExchangeInfo 从缓存获取exchangeInfo
func (s *Server) getCachedExchangeInfo(key string) (map[string]ExchangeInfoItem, bool) {
	ctx := context.Background()
	cachedData, err := s.cache.Get(ctx, key)
	if err != nil || len(cachedData) == 0 {
		return nil, false
	}

	var exchangeInfo map[string]ExchangeInfoItem
	if err := json.Unmarshal(cachedData, &exchangeInfo); err != nil {
		log.Printf("[WARN] Failed to unmarshal cached exchange info: %v", err)
		return nil, false
	}

	return exchangeInfo, true
}

// cacheExchangeInfo 缓存exchangeInfo
func (s *Server) cacheExchangeInfo(key string, data map[string]ExchangeInfoItem, duration time.Duration) {
	ctx := context.Background()
	cacheData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WARN] Failed to marshal exchange info for cache: %v", err)
		return
	}

	if err := s.cache.Set(ctx, key, cacheData, duration); err != nil {
		log.Printf("[WARN] Failed to cache exchange info: %v", err)
	}
}

// minInt 返回两个整数中的较小值（优化辅助函数）
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) GetBinanceMarket(c *gin.Context) {
	params, err := parseBinanceMarketParams(c)
	if err != nil {
		s.BadRequest(c, "参数解析失败", err)
		return
	}

	// 如果没传 date，默认今天（本地时区）
	if params.Date == "" {
		day := time.Now().In(params.Location).Format("2006-01-02")
		q := c.Request.URL.Query()
		q.Set("date", day)
		c.Request.URL.RawQuery = q.Encode()
		// 重新解析参数
		params, err = parseBinanceMarketParams(c)
		if err != nil {
			s.BadRequest(c, "参数解析失败", err)
			return
		}
	}

	// 计算时间范围
	startUTC, endUTC, err := calculateTimeRange(params)
	if err != nil {
		s.ValidationError(c, "date", err.Error())
		return
	}

	// 查询市场数据
	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), params.Kind, startUTC, endUTC)
	if err != nil {
		s.DatabaseError(c, "查询市场数据", err)
		return
	}

	// 过滤和格式化数据
	out, err := s.filterAndFormatMarketDataWithCategory(snaps, tops, params.Kind, params.Category, c.Request.Context())
	if err != nil {
		s.InternalServerError(c, "处理市场数据失败", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"kind":     params.Kind,
		"interval": params.IntervalMin,
		"data":     out,
	})
}

// WSRealTimeGainers WebSocket实时涨幅榜 - 新版本（使用数据同步器）
// 这个新版本使用后台数据同步器自动更新的realtime_gainers_items表数据
// 不再需要实时从binance_24h_stats查询，大幅提升性能和响应速度
func (s *Server) WSRealTimeGainers(c *gin.Context) {
	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] 涨幅榜连接升级失败: %v", err)
		return
	}
	defer ws.Close()

	clientIP := c.ClientIP()
	log.Printf("[WebSocket] 涨幅榜新连接建立: %s", clientIP)

	// 读取客户端的订阅消息
	_, message, err := ws.ReadMessage()
	if err != nil {
		log.Printf("[WebSocket] 读取涨幅榜订阅消息失败: %v", err)
		return
	}

	// 解析订阅请求
	var subscription struct {
		Action   string `json:"action"`
		Kind     string `json:"kind"`     // "spot" 或 "futures"
		Category string `json:"category"` // 分类筛选，如 "trading", "all" 等
		Limit    int    `json:"limit"`    // 返回数量限制
		Interval int    `json:"interval"` // 更新间隔(秒)，默认10秒
	}

	if err := json.Unmarshal(message, &subscription); err != nil {
		log.Printf("[WebSocket] 解析涨幅榜订阅消息失败: %v", err)
		ws.WriteJSON(gin.H{"error": "无效的订阅格式"})
		return
	}

	log.Printf("[WebSocket] 📨 收到订阅请求: action=%s, kind=%s, category=%s, limit=%d, interval=%d",
		subscription.Action, subscription.Kind, subscription.Category, subscription.Limit, subscription.Interval)

	if subscription.Action != "subscribe" {
		log.Printf("[WebSocket] ❌ 不支持的操作: %s", subscription.Action)
		ws.WriteJSON(gin.H{"error": "不支持的操作"})
		return
	}

	// 设置默认值
	if subscription.Kind == "" {
		subscription.Kind = "spot"
	}
	log.Printf("[WebSocket] 🔧 处理订阅请求，市场类型: %s", subscription.Kind)
	if subscription.Limit <= 0 || subscription.Limit > 100 {
		subscription.Limit = 15 // 允许更大的limit用于筛选
	}
	if subscription.Interval <= 0 || subscription.Interval > 300 {
		subscription.Interval = 10 // 默认10秒，频率更高
	}

	log.Printf("[WebSocket] 涨幅榜订阅确认: kind=%s, limit=%d, interval=%ds",
		subscription.Kind, subscription.Limit, subscription.Interval)

	// 发送确认消息
	ws.WriteJSON(gin.H{
		"type":    "subscription_confirmed",
		"message": "实时涨幅榜订阅成功（数据同步器版本）",
		"config": gin.H{
			"kind":     subscription.Kind,
			"category": subscription.Category,
			"limit":    subscription.Limit,
			"interval": subscription.Interval,
		},
	})

	ctx := context.Background()
	var lastGainersData []gin.H // 缓存上次的数据，用于比较变化

	// 立即发送第一批数据（从realtime_gainers_items表获取）
	log.Printf("[WebSocket] 📊 从数据同步器获取初始涨幅榜数据，市场: %s, 限制: %d...",
		subscription.Kind, subscription.Limit)
	gainersData, err := s.getRealtimeGainersFromSyncer(subscription.Kind, subscription.Limit)
	if err != nil {
		log.Printf("[WebSocket] ❌ 获取初始涨幅榜数据失败: %v", err)
		// 发送错误消息给客户端
		ws.WriteJSON(gin.H{
			"type":    "error",
			"message": "获取涨幅榜数据失败",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[WebSocket] ✅ 成功获取涨幅榜数据，条数: %d", len(gainersData))
	if len(gainersData) > 0 {
		log.Printf("[WebSocket] 📊 示例数据: %s: %.2f%%", gainersData[0]["symbol"], gainersData[0]["price_change_24h"])
	}

	// 深拷贝数据用于比较变化
	if len(gainersData) > 0 {
		lastGainersData = make([]gin.H, len(gainersData))
		for i, gainer := range gainersData {
			lastGainersData[i] = make(gin.H)
			for k, v := range gainer {
				lastGainersData[i][k] = v
			}
		}
	}

	// 发送初始数据
	response := gin.H{
		"type":        "gainers_update",
		"timestamp":   time.Now().Unix(),
		"kind":        subscription.Kind,
		"limit":       subscription.Limit,
		"data_source": "syncer", // 标记数据来源
		"gainers":     gainersData,
	}

	if err := ws.WriteJSON(response); err != nil {
		log.Printf("[WebSocket] ❌ 发送初始涨幅榜数据失败: %v", err)
		return
	}
	log.Printf("[WebSocket] ✅ 初始涨幅榜数据发送成功（数据同步器版本），发送%d条数据", len(gainersData))

	// 创建定时器发送实时更新
	ticker := time.NewTicker(time.Duration(subscription.Interval) * time.Second)
	defer ticker.Stop()

	// 创建心跳定时器（每30秒发送一次心跳）
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// 定期发送计数器，即使没有显著变化也要定期发送（每5次检查强制发送一次）
	updateCounter := 0
	const forceUpdateInterval = 5

	// 发送实时更新
	for {
		select {
		case <-ticker.C:
			log.Printf("[WebSocket] 定时器触发，从数据同步器获取涨幅榜更新...")

			// 从数据同步器获取最新数据
			gainersData, err := s.getRealtimeGainersFromSyncer(subscription.Kind, subscription.Limit)
			if err != nil {
				log.Printf("[WebSocket] 获取涨幅榜数据失败: %v", err)
				continue
			}

			updateCounter++
			hasSignificantChanges := s.hasSignificantChanges(lastGainersData, gainersData)

			// 检查数据是否有显著变化，或者达到定期发送间隔
			if !hasSignificantChanges && updateCounter < forceUpdateInterval {
				log.Printf("[WebSocket] 数据无显著变化，跳过本次更新 (计数器: %d/%d)", updateCounter, forceUpdateInterval)
				continue
			}

			// 如果是定期强制发送，重置计数器
			if !hasSignificantChanges && updateCounter >= forceUpdateInterval {
				log.Printf("[WebSocket] 达到定期发送间隔，强制发送更新数据 (计数器: %d)", updateCounter)
				updateCounter = 0
			} else if hasSignificantChanges {
				log.Printf("[WebSocket] 数据有显著变化，发送更新")
				updateCounter = 0 // 有显著变化时也重置计数器
			}

			// 更新缓存
			lastGainersData = make([]gin.H, len(gainersData))
			for i, gainer := range gainersData {
				lastGainersData[i] = make(gin.H)
				for k, v := range gainer {
					lastGainersData[i][k] = v
				}
			}

			log.Printf("[WebSocket] 发送定时涨幅榜更新，条数: %d", len(gainersData))

			// 发送更新
			response := gin.H{
				"type":        "gainers_update",
				"timestamp":   time.Now().Unix(),
				"kind":        subscription.Kind,
				"limit":       subscription.Limit,
				"data_source": "syncer", // 标记数据来源
				"gainers":     gainersData,
			}

			if err := ws.WriteJSON(response); err != nil {
				log.Printf("[WebSocket] 发送涨幅榜更新失败: %v", err)
				return
			}
			log.Printf("[WebSocket] 定时涨幅榜更新发送成功")

		case <-heartbeatTicker.C:
			// 发送心跳消息，保持连接活跃
			heartbeat := gin.H{
				"type":      "heartbeat",
				"timestamp": time.Now().Unix(),
				"message":   "connection_alive",
			}

			if err := ws.WriteJSON(heartbeat); err != nil {
				log.Printf("[WebSocket] 发送心跳失败: %v", err)
				return
			}
			log.Printf("[WebSocket] 心跳发送成功")

		case <-ctx.Done():
			log.Printf("[WebSocket] 涨幅榜连接上下文取消")
			return
		}
	}
}

// getRealtimeGainersFromSyncer 从数据同步器获取实时涨幅榜数据
// 这个方法直接查询realtime_gainers_items表，数据由后台同步器自动更新
func (s *Server) getRealtimeGainersFromSyncer(kind string, limit int) ([]gin.H, error) {
	// 优化查询：分两步执行，避免复杂的JOIN查询
	// 第一步：获取最新的快照ID
	var snapshotID uint
	snapshotQuery := `
		SELECT id
		FROM realtime_gainers_snapshots
		WHERE kind = ?
		ORDER BY timestamp DESC, id DESC
		LIMIT 1
	`

	err := s.db.DB().Raw(snapshotQuery, kind).Scan(&snapshotID).Error
	if err != nil {
		return nil, fmt.Errorf("查询最新快照失败: %w", err)
	}

	if snapshotID == 0 {
		log.Printf("[WebSocket] %s市场没有找到快照数据", kind)
		return []gin.H{}, nil
	}

	// 第二步：使用GORM链式查询获取对应快照的数据项

	var results []pdb.RealtimeGainersItem

	// 使用GORM链式查询替代Raw SQL

	// 使用GORM链式查询替代Raw SQL，更类型安全和可维护
	err = s.db.DB().
		Select("symbol, `rank`, current_price, price_change24h, volume24h, data_source, created_at").
		Where("snapshot_id = ?", snapshotID).
		Order("`rank` ASC").
		Limit(limit).
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("从数据同步器查询涨幅榜数据失败: %w", err)
	}

	if len(results) == 0 {
		log.Printf("[WebSocket] 数据同步器中没有%s市场数据，返回空结果", kind)
		return []gin.H{}, nil
	}

	// 转换为前端需要的格式
	gainers := make([]gin.H, 0, len(results))
	for _, item := range results {
		gainer := gin.H{
			"symbol":           item.Symbol,
			"current_price":    item.CurrentPrice,
			"price_change_24h": item.PriceChange24h,
			"volume_24h":       item.Volume24h,
			"rank":             item.Rank,
			"data_source":      item.DataSource,
			"timestamp":        item.CreatedAt.Unix(),
		}

		// 处理可选字段
		if item.PriceChangePercent != nil {
			gainer["price_change_percent"] = *item.PriceChangePercent
		} else {
			gainer["price_change_percent"] = item.PriceChange24h // 向后兼容
		}

		if item.Confidence != nil {
			gainer["confidence"] = *item.Confidence
		}

		gainers = append(gainers, gainer)
	}

	log.Printf("[WebSocket] 从数据同步器获取到%d条%s涨幅榜数据", len(gainers), kind)
	return gainers, nil
}

// applyCategoryFilter 对涨幅榜数据应用category筛选
func (s *Server) applyCategoryFilter(gainers []gin.H, category string) ([]gin.H, error) {
	if category == "" || category == "all" {
		return gainers, nil
	}

	log.Printf("[涨幅榜] 对%d条数据应用%s分类筛选", len(gainers), category)

	filtered := make([]gin.H, 0, len(gainers))
	for _, gainer := range gainers {
		symbol, ok := gainer["symbol"].(string)
		if !ok {
			continue
		}

		shouldInclude := false

		switch category {
		case "trading":
			// 正常交易：排除暂停交易的币种
			// 注意：同步器数据可能不包含交易状态信息，我们假设都为正常交易
			shouldInclude = true

		case "break":
			// 暂停交易：这里我们无法准确判断，保守起见不包含
			shouldInclude = false

		case "major":
			// 主流币种：BTC, ETH, BNB等
			majorSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT", "SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "LTCUSDT"}
			for _, major := range majorSymbols {
				if symbol == major {
					shouldInclude = true
					break
				}
			}

		case "stable":
			// 稳定币对：包含USDT, USDC, BUSD等
			shouldInclude = strings.Contains(symbol, "USDT") || strings.Contains(symbol, "USDC") || strings.Contains(symbol, "BUSD")

		case "defi":
			// DeFi代币：这里无法准确判断，保守起见包含所有
			shouldInclude = true

		case "layer1":
			// Layer1公链：BTC, ETH, BNB, SOL, ADA等
			layer1Symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT"}
			for _, layer1 := range layer1Symbols {
				if symbol == layer1 {
					shouldInclude = true
					break
				}
			}

		case "meme":
			// Meme币：这里无法准确判断，保守起见包含所有
			shouldInclude = true

		case "spot_only":
			// 纯现货：所有币种（因为同步器数据就是现货数据）
			shouldInclude = true

		case "margin":
			// 杠杆交易：这里无法准确判断，保守起见包含所有
			shouldInclude = true

		case "leveraged":
			// 合约交易：这里无法准确判断，保守起见包含所有
			shouldInclude = true

		default:
			// 未知分类，包含所有
			shouldInclude = true
		}

		if shouldInclude {
			filtered = append(filtered, gainer)
		}
	}

	log.Printf("[涨幅榜] 分类筛选完成: %d -> %d 条数据", len(gainers), len(filtered))
	return filtered, nil
}

// generateRealtimeGainersFrom24hStats 直接从 binance_24h_stats 生成涨幅榜数据（优化版本）
func (s *Server) generateRealtimeGainersFrom24hStats(ctx context.Context, kind string, category string, limit int) ([]gin.H, error) {
	// 缓存键
	cacheKey := fmt.Sprintf("gainers_24h_%s_%s_%d", kind, category, limit)

	// 检查缓存
	if cached, exists := s.getCachedGainers(cacheKey); exists {
		log.Printf("[涨幅榜:24h] 使用缓存数据: %s", cacheKey)
		return cached, nil
	}

	log.Printf("[涨幅榜:24h] 缓存未命中，从 binance_24h_stats 生成数据: %s", cacheKey)

	// 确定实际返回数量：前端涨幅榜固定15个，其他调用可返回更多用于筛选
	actualLimit := 15 // 前端默认15个
	if limit > 15 && limit <= 100 {
		// 策略扫描器等调用允许返回更多数据用于筛选
		actualLimit = limit
	}

	// 优化查询：直接使用ORDER BY和LIMIT，避免窗口函数
	// 添加更精确的时间过滤，确保使用索引
	query := fmt.Sprintf(`
		SELECT
			symbol,
			price_change_percent,
			volume,
			quote_volume,
			last_price
		FROM binance_24h_stats
		WHERE market_type = ?
			AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			AND volume > 1000  -- 提高最低成交量阈值，过滤低流动性币种
			AND last_price > 0.000001  -- 过滤价格过低的币种
			AND price_change_percent BETWEEN -99 AND 1000  -- 过滤异常数据
		ORDER BY
			price_change_percent DESC,
			volume DESC,
			quote_volume DESC  -- 添加quote_volume作为次要排序条件
		LIMIT %d
	`, actualLimit)

	var results []struct {
		Symbol             string  `json:"symbol"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		QuoteVolume        float64 `json:"quote_volume"`
		LastPrice          float64 `json:"last_price"`
	}

	err := s.db.DB().Raw(query, kind).Scan(&results).Error
	if err != nil {
		log.Printf("[涨幅榜:24h] 查询 binance_24h_stats 失败: %v", err)
		return nil, fmt.Errorf("查询涨幅榜数据失败: %w", err)
	}

	if len(results) == 0 {
		log.Printf("[涨幅榜:24h] 没有找到 %s 市场的数据", kind)
		return []gin.H{}, nil
	}

	// 转换为前端期望的格式（直接返回前N名，无需额外过滤）
	gainers := make([]gin.H, 0, len(results))
	for i, item := range results {
		gainer := gin.H{
			"symbol":           item.Symbol,
			"current_price":    item.LastPrice,
			"price_change_24h": item.PriceChangePercent,
			"volume_24h":       item.Volume,
			"quote_volume_24h": item.QuoteVolume,
			"rank":             i + 1, // 基于结果顺序分配排名
			"data_source":      "24h_stats",
			"price_change":     item.PriceChangePercent, // 兼容旧字段
			"change":           item.PriceChangePercent, // 前端可能使用的字段
		}

		// 添加市值估算
		if item.QuoteVolume > 0 {
			gainer["market_cap"] = item.QuoteVolume // 简化市值估算
		}

		gainers = append(gainers, gainer)
	}

	// 缓存结果（5分钟）
	s.cacheGainersWithDuration(cacheKey, gainers, 5*time.Minute)

	log.Printf("[涨幅榜:24h] 成功生成前15名 %s 市场涨幅榜数据，共 %d 条", kind, len(gainers))
	return gainers, nil
}

// generateRealtimeGainersData 生成实时涨幅榜数据（保留旧版本用于兼容）
func (s *Server) generateRealtimeGainersData(ctx context.Context, kind string, category string, limit int) ([]gin.H, error) {
	// 缓存键
	cacheKey := fmt.Sprintf("gainers_%s_%s_%d", kind, category, limit)

	// 智能缓存策略：根据市场活跃度动态调整缓存时间
	cacheDuration := s.getDynamicCacheDuration(kind)

	// 检查缓存（带过期时间检查）
	if cached, exists := s.getCachedGainersWithDuration(cacheKey, cacheDuration); exists {
		log.Printf("[涨幅榜] 使用缓存数据: %s (缓存时长: %v)", cacheKey, cacheDuration)
		return cached, nil
	}

	log.Printf("[涨幅榜] 缓存未命中，开始获取新数据: %s", cacheKey)

	// 获取热门币种列表 - 从binance_market_snapshots和binance_market_tops获取最新的一个快照的数据
	var symbols []string
	dbInstance := s.db.DB()

	// 从最新的快照中获取交易对数据，按照涨幅榜排序
	query := `
		SELECT t.symbol
		FROM binance_market_tops t
		INNER JOIN binance_market_snapshots s ON t.snapshot_id = s.id
		WHERE s.kind = ? AND s.id = (
			SELECT id FROM binance_market_snapshots
			WHERE kind = ?
			ORDER BY bucket DESC
			LIMIT 1
		)
		ORDER BY CAST(t.pct_change AS DECIMAL(10,6)) DESC
		LIMIT ?
	`

	rows, err := dbInstance.Raw(query, kind, kind, limit*10).Rows()
	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var symbol string
			if err := rows.Scan(&symbol); err != nil {
				continue
			}
			symbols = append(symbols, symbol)
		}
	} else {
		log.Printf("[涨幅榜] 数据库查询失败，无法获取币种列表: %v", err)
		return nil, fmt.Errorf("获取可用币种列表失败: %w", err)
	}

	// 如果没有获取到数据，返回错误
	if len(symbols) == 0 {
		log.Printf("[涨幅榜] 数据库中没有可用币种数据")
		return nil, fmt.Errorf("数据库中没有可用币种数据")
	}

	// 使用并发获取数据以提高性能
	gainersChan := make(chan gin.H, len(symbols))
	var wg sync.WaitGroup
	var wsCount, httpCount, fallbackCount int32 // 使用原子操作
	var processedCount int32

	// 限制并发数量，避免过载
	maxConcurrency := 8 // 减少并发数量以提高稳定性
	semaphore := make(chan struct{}, maxConcurrency)

	// 添加超时控制
	timeout := 15 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			defer atomic.AddInt32(&processedCount, 1)

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctxWithTimeout.Done():
				return
			}

			// 检查上下文是否已取消
			select {
			case <-ctxWithTimeout.Done():
				return
			default:
			}

			realtimeData, success := s.getRealtimeDataConcurrently(ctxWithTimeout, sym, kind)
			if !success {
				log.Printf("[涨幅榜] %s 数据获取失败，尝试降级获取", sym)
				// 尝试降级获取（简化版数据）
				fallbackData := s.getFallbackRealtimeData(sym, kind)
				if fallbackData.LastPrice > 0 {
					log.Printf("[涨幅榜] %s 降级数据获取成功", sym)
					// 获取分类信息
					category := s.getSymbolCategory(fallbackData.Symbol, kind)
					categoryInfo := gin.H{
						"status":      category.Status,
						"asset_type":  category.AssetType,
						"market_cap":  category.MarketCap,
						"trade_type":  category.TradeType,
						"order_level": category.OrderLevel,
						"is_active":   category.IsActive,
					}

					gainer := gin.H{
						"symbol":               fallbackData.Symbol,
						"current_price":        fallbackData.LastPrice,
						"price_change_24h":     fallbackData.ChangePercent,
						"volume_24h":           fallbackData.Volume,
						"price_change_percent": fallbackData.ChangePercent,
						"data_source":          fallbackData.DataSource,
						"timestamp":            fallbackData.Timestamp,
						"category":             categoryInfo,
					}
					select {
					case gainersChan <- gainer:
					case <-ctxWithTimeout.Done():
					}
				}
				return
			}

			// 统计数据源
			switch realtimeData.DataSource {
			case "websocket":
				atomic.AddInt32(&wsCount, 1)
			case "http_api":
				atomic.AddInt32(&httpCount, 1)
			default:
				atomic.AddInt32(&fallbackCount, 1)
			}

			// 数据质量检查和异常检测
			if !s.validateRealtimeData(realtimeData) {
				return
			}

			// 获取交易对分类信息
			var categoryInfo gin.H
			if realtimeData.Category != nil {
				categoryInfo = gin.H{
					"status":      realtimeData.Category.Status,
					"permissions": realtimeData.Category.Permissions,
					"order_types": realtimeData.Category.OrderTypes,
					"base_asset":  realtimeData.Category.BaseAsset,
					"quote_asset": realtimeData.Category.QuoteAsset,
					"asset_type":  realtimeData.Category.AssetType,
					"market_cap":  realtimeData.Category.MarketCap,
					"trade_type":  realtimeData.Category.TradeType,
					"order_level": realtimeData.Category.OrderLevel,
					"is_active":   realtimeData.Category.IsActive,
				}
			} else {
				// 默认分类信息
				categoryInfo = gin.H{
					"status":      "UNKNOWN",
					"asset_type":  "emerging",
					"market_cap":  "mid",
					"trade_type":  "spot_only",
					"order_level": "basic",
					"is_active":   true,
				}
			}

			// 转换为前端期望的格式
			gainer := gin.H{
				"symbol":               realtimeData.Symbol,
				"current_price":        realtimeData.LastPrice,
				"price_change_24h":     realtimeData.ChangePercent,
				"volume_24h":           realtimeData.Volume,
				"price_change_percent": realtimeData.ChangePercent,
				"data_source":          realtimeData.DataSource,
				"timestamp":            realtimeData.Timestamp,
				"category":             categoryInfo,
			}

			// 非阻塞发送结果
			select {
			case gainersChan <- gainer:
			case <-ctxWithTimeout.Done():
			}
		}(symbol)
	}

	// 等待所有goroutine完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[涨幅榜] 所有数据获取完成，共处理 %d 个币种", atomic.LoadInt32(&processedCount))
	case <-ctxWithTimeout.Done():
		log.Printf("[涨幅榜] 数据获取超时，已处理 %d/%d 个币种", atomic.LoadInt32(&processedCount), len(symbols))
		// 等待所有goroutine真正完成后再关闭channel，避免"send on closed channel"panic
		wg.Wait()
	}

	close(gainersChan)

	// 收集结果
	gainers := make([]gin.H, 0, len(symbols))
	for gainer := range gainersChan {
		gainers = append(gainers, gainer)
	}

	// 数据异常监控
	s.monitorDataQuality(gainers, kind)

	// 记录数据源统计
	log.Printf("[涨幅榜] 数据获取完成: 总计%d个币种, WebSocket=%d, HTTP_API=%d, 降级=%d",
		len(symbols), atomic.LoadInt32(&wsCount), atomic.LoadInt32(&httpCount), atomic.LoadInt32(&fallbackCount))

	// 使用动态缓存时长
	s.cacheGainersWithDuration(cacheKey, gainers, cacheDuration)

	// 使用更高效的排序算法
	s.sortGainersByChangePercent(gainers)

	// 获取黑名单并进行黑名单+分类筛选（参考涨幅榜的逻辑）
	blacklistMap, err := s.getCachedBlacklistMap(context.Background(), kind)
	if err != nil {
		log.Printf("[WARN] 获取黑名单失败，使用空黑名单: %v", err)
		blacklistMap = make(map[string]bool)
	}

	// 获取exchangeInfo用于分类筛选
	exchangeInfo, err := s.getExchangeInfoForCategory(ctx, kind)
	if err != nil {
		log.Printf("[WARN] 获取exchangeInfo失败: %v", err)
	}

	// 进行黑名单和分类筛选
	gainers = s.filterGainersByBlacklistAndCategory(gainers, blacklistMap, category, exchangeInfo, kind, limit)

	// 限制返回数量
	if len(gainers) > limit {
		gainers = gainers[:limit]
	}

	// 添加排名
	for i, gainer := range gainers {
		gainer["rank"] = i + 1
	}

	return gainers, nil
}

// filterGainersByBlacklistAndCategory 根据黑名单和分类筛选实时涨幅榜数据（参考涨幅榜逻辑）
func (s *Server) filterGainersByBlacklistAndCategory(gainers []gin.H, blacklistMap map[string]bool, category string, exchangeInfo map[string]ExchangeInfoItem, kind string, maxCount int) []gin.H {
	if category == "" || category == "all" {
		// 如果没有分类要求，只过滤黑名单
		filtered := make([]gin.H, 0, len(gainers))
		for _, gainer := range gainers {
			symbol, _ := gainer["symbol"].(string)
			if !blacklistMap[strings.ToUpper(symbol)] {
				filtered = append(filtered, gainer)
				if len(filtered) >= maxCount {
					break
				}
			}
		}
		return filtered
	}

	// 根据分类进行筛选
	filtered := make([]gin.H, 0, len(gainers))
	matchedCount := 0
	for _, gainer := range gainers {
		symbol, _ := gainer["symbol"].(string)

		// 先检查黑名单
		if blacklistMap[strings.ToUpper(symbol)] {
			continue
		}

		// 根据分类进行筛选
		if s.matchesGainerCategoryForRealtime(gainer, category, exchangeInfo, kind) {
			filtered = append(filtered, gainer)
			matchedCount++
			if len(filtered) >= maxCount {
				break
			}
		}
	}
	return filtered
}

// matchesGainerCategoryForRealtime 检查实时涨幅榜条目是否匹配指定的分类（参考涨幅榜逻辑）
func (s *Server) matchesGainerCategoryForRealtime(gainer gin.H, category string, exchangeInfo map[string]ExchangeInfoItem, kind string) bool {
	symbol, _ := gainer["symbol"].(string)

	// 如果没有exchangeInfo数据，使用基于symbol的匹配
	if exchangeInfo == nil {
		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	info, exists := exchangeInfo[symbol]
	if !exists {
		// 如果exchangeInfo中没有这个交易对，使用基于symbol的匹配
		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	switch category {
	case "trading":
		return info.Status == "TRADING"
	case "break":
		return info.Status == "BREAK"
	case "major", "stable", "defi":
		// 使用exchangeInfo的baseAsset进行智能分类
		return s.isAssetTypeMatch(info.BaseAsset, category)
	case "layer1":
		// Layer1资产特殊处理
		if info.BaseAsset != "" {
			return s.isAssetTypeMatch(info.BaseAsset, category)
		}
		// 降级到基于交易对符号的检查
		layer1Assets := []string{"ATOM", "NEAR", "FTM", "ONE", "EGLD", "FLOW"}
		baseSymbol := s.getBaseSymbol(symbol)
		return s.containsString(layer1Assets, baseSymbol)
	case "meme":
		// Meme资产特殊处理
		if info.BaseAsset != "" {
			return s.isAssetTypeMatch(info.BaseAsset, category)
		}
		// 降级到基于交易对符号的检查
		memeAssets := []string{"SHIB", "DOGE", "PEPE", "BONK", "WIF", "TURBO"}
		baseSymbol := s.getBaseSymbol(symbol)
		return s.containsString(memeAssets, baseSymbol)
	case "spot_only":
		return s.containsString(info.Permissions, "SPOT") && !s.containsString(info.Permissions, "LEVERAGED")
	case "margin":
		return s.containsString(info.Permissions, "MARGIN")
	case "leveraged":
		return s.containsString(info.Permissions, "LEVERAGED")
	default:
		return true
	}
}

// containsPermission 检查权限列表是否包含指定权限
func (s *Server) containsPermission(permissions []interface{}, permission string) bool {
	for _, p := range permissions {
		if perm, ok := p.(string); ok && perm == permission {
			return true
		}
	}
	return false
}

// SaveRealtimeGainersData 保存实时涨幅榜数据（内部API）
func (s *Server) SaveRealtimeGainersData(ctx context.Context, kind string, gainers []gin.H) error {
	if len(gainers) == 0 {
		return nil
	}

	// 转换为数据库结构
	items := make([]pdb.RealtimeGainersItem, 0, len(gainers))
	for i, gainer := range gainers {
		rank := i + 1
		if r, ok := gainer["rank"].(int); ok && r > 0 {
			rank = r
		}

		item := pdb.RealtimeGainersItem{
			Symbol:         gainer["symbol"].(string),
			Rank:           rank,
			CurrentPrice:   gainer["current_price"].(float64),
			PriceChange24h: gainer["price_change_24h"].(float64),
			Volume24h:      gainer["volume_24h"].(float64),
			DataSource:     gainer["data_source"].(string),
		}

		// 可选字段
		if pc, ok := gainer["price_change_percent"].(float64); ok {
			item.PriceChangePercent = &pc
		}
		if conf, ok := gainer["confidence"].(float64); ok {
			item.Confidence = &conf
		}

		items = append(items, item)
	}

	// 保存到数据库
	_, err := pdb.SaveRealtimeGainers(s.db.DB(), kind, time.Now(), items)
	if err != nil {
		log.Printf("[涨幅榜] 保存历史数据失败: %v", err)
		return err
	}

	log.Printf("[涨幅榜] 成功保存 %d 条涨幅榜历史数据", len(items))
	return nil
}

// GetRealtimeGainersHistoryAPI 获取涨幅榜历史数据API
// GET /market/binance/realtime-gainers/history?kind=spot&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&symbol=BTC&limit=10
func (s *Server) GetRealtimeGainersHistoryAPI(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	symbol := strings.TrimSpace(c.Query("symbol"))
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// 解析时间
	var startTime, endTime time.Time
	if startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}
	if endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	// 获取历史数据
	snapshots, itemsMap, err := pdb.GetRealtimeGainersHistory(s.db.DB(), kind, startTime, endTime, symbol, limit)
	if err != nil {
		s.DatabaseError(c, "获取涨幅榜历史数据", err)
		return
	}

	// 转换为前端格式
	result := make([]gin.H, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items := itemsMap[snapshot.ID]
		formattedItems := make([]gin.H, len(items))
		for i, item := range items {
			formattedItems[i] = gin.H{
				"symbol":               item.Symbol,
				"rank":                 item.Rank,
				"current_price":        item.CurrentPrice,
				"price_change_24h":     item.PriceChange24h,
				"volume_24h":           item.Volume24h,
				"data_source":          item.DataSource,
				"price_change_percent": item.PriceChangePercent,
				"confidence":           item.Confidence,
				"timestamp":            item.CreatedAt.Unix(),
			}
		}

		result = append(result, gin.H{
			"id":        snapshot.ID,
			"kind":      snapshot.Kind,
			"timestamp": snapshot.Timestamp.Unix(),
			"datetime":  snapshot.Timestamp.Format(time.RFC3339),
			"gainers":   formattedItems,
			"count":     len(formattedItems),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":          result,
		"count":         len(result),
		"kind":          kind,
		"symbol_filter": symbol,
		"time_range": gin.H{
			"start": startTimeStr,
			"end":   endTimeStr,
		},
	})
}

// GetRealtimeGainersStatsAPI 获取涨幅榜数据统计API
// GET /market/binance/realtime-gainers/stats
func (s *Server) GetRealtimeGainersStatsAPI(c *gin.Context) {
	stats, err := pdb.GetRealtimeGainersStats(s.db.DB())
	if err != nil {
		s.DatabaseError(c, "获取涨幅榜统计数据", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"timestamp": time.Now().Unix(),
	})
}

// CleanRealtimeGainersDataAPI 清理旧的涨幅榜数据API
// POST /market/binance/realtime-gainers/clean?keep_days=30
func (s *Server) CleanRealtimeGainersDataAPI(c *gin.Context) {
	keepDaysStr := c.DefaultQuery("keep_days", "30")
	keepDays, err := strconv.Atoi(keepDaysStr)
	if err != nil || keepDays <= 0 || keepDays > 365 {
		keepDays = 30
	}

	err = pdb.CleanOldRealtimeGainers(s.db.DB(), keepDays)
	if err != nil {
		s.DatabaseError(c, "清理涨幅榜数据", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "涨幅榜数据清理完成",
		"keep_days": keepDays,
	})
}

// getDynamicCacheDuration 根据市场活跃度动态计算缓存时长
func (s *Server) getDynamicCacheDuration(kind string) time.Duration {
	// 获取当前时间
	now := time.Now()

	// 工作日和周末的不同缓存策略
	weekday := now.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	// 获取当前小时
	hour := now.Hour()

	// 亚洲交易时段（0-8点）：活跃度较低
	// 欧洲交易时段（8-16点）：中等活跃度
	// 美洲交易时段（16-24点）：最高活跃度
	var baseDuration time.Duration
	switch {
	case hour >= 0 && hour < 8: // 亚洲时段
		baseDuration = 30 * time.Second
	case hour >= 8 && hour < 16: // 欧洲时段
		baseDuration = 20 * time.Second
	default: // 美洲时段（16-24点）
		baseDuration = 15 * time.Second
	}

	// 周末适当增加缓存时间
	if isWeekend {
		baseDuration = time.Duration(float64(baseDuration) * 1.5)
	}

	// 对于合约，缓存时间稍短（市场更活跃）
	if kind == "futures" {
		baseDuration = time.Duration(float64(baseDuration) * 0.8)
	}

	// 确保缓存时间在合理范围内
	if baseDuration < 15*time.Second {
		baseDuration = 15 * time.Second
	}
	if baseDuration > 120*time.Second {
		baseDuration = 120 * time.Second
	}

	return baseDuration
}

// sortGainersByChangePercent 使用更高效的排序算法按涨幅排序
func (s *Server) sortGainersByChangePercent(gainers []gin.H) {
	// 使用sort包进行更高效的排序
	sort.Slice(gainers, func(i, j int) bool {
		changeI, okI := gainers[i]["price_change_24h"].(float64)
		changeJ, okJ := gainers[j]["price_change_24h"].(float64)

		// 如果类型不匹配，按原顺序保持
		if !okI && !okJ {
			return false
		}
		if !okI {
			return false // 有问题的数据排在后面
		}
		if !okJ {
			return true // 有问题的数据排在后面
		}

		// 降序排列：涨幅高的在前
		return changeI > changeJ
	})
}

// SymbolCategory 交易对分类信息
type SymbolCategory struct {
	Symbol      string   `json:"symbol"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
	OrderTypes  []string `json:"order_types"`
	BaseAsset   string   `json:"base_asset"`
	QuoteAsset  string   `json:"quote_asset"`
	AssetType   string   `json:"asset_type"`  // 资产类型: major, stable, defi, layer1, meme, nft_gaming, emerging
	MarketCap   string   `json:"market_cap"`  // 市值规模: large, mid, small
	TradeType   string   `json:"trade_type"`  // 交易类型: spot_only, margin, leveraged, trading_groups
	OrderLevel  string   `json:"order_level"` // 订单级别: basic, stop_loss, take_profit, advanced, full_featured
	IsActive    bool     `json:"is_active"`   // 是否活跃交易
}

// RealtimeData 统一的实时数据结构
type RealtimeData struct {
	Symbol        string          `json:"symbol"`
	LastPrice     float64         `json:"current_price"`
	ChangePercent float64         `json:"price_change_24h"`
	Volume        float64         `json:"volume_24h"`
	DataSource    string          `json:"data_source"` // "websocket", "http_api", "kline_calc"
	Timestamp     int64           `json:"timestamp"`
	Category      *SymbolCategory `json:"category,omitempty"` // 分类信息
}

// getSymbolCategory 获取交易对分类信息
func (s *Server) getSymbolCategory(symbol string, kind string) *SymbolCategory {
	// 尝试从exchangeInfo获取真实的分类信息
	ctx := context.Background()
	exchangeInfo, err := s.getExchangeInfoForCategory(ctx, kind)
	if err != nil {
		log.Printf("[WARN] 获取exchangeInfo失败，使用默认分类: %v", err)
		return s.getDefaultSymbolCategory(symbol, kind)
	}

	info, exists := exchangeInfo[symbol]
	if !exists {
		log.Printf("[WARN] exchangeInfo中未找到交易对 %s，使用默认分类", symbol)
		return s.getDefaultSymbolCategory(symbol, kind)
	}

	// 从真实的exchangeInfo构建分类信息
	category := &SymbolCategory{
		Symbol:      symbol,
		Status:      info.Status,
		Permissions: info.Permissions,
		BaseAsset:   info.BaseAsset,
		QuoteAsset:  info.QuoteAsset,
		IsActive:    info.Status == "TRADING",
	}

	// 根据基础资产确定资产类型
	assetType := "emerging"
	switch info.BaseAsset {
	case "BTC", "ETH", "BNB", "ADA", "SOL", "DOT", "AVAX", "MATIC", "LINK", "LTC", "XRP", "TRX", "ETC", "BCH":
		assetType = "major"
	case "USDT", "USDC", "BUSD", "DAI", "TUSD", "USDP":
		assetType = "stable"
	case "UNI", "AAVE", "SUSHI", "COMP", "MKR", "SNX", "CRV":
		assetType = "defi"
	case "SHIB", "DOGE", "PEPE", "BONK", "TURBO":
		assetType = "meme"
	case "MANA", "SAND", "GALA", "AXS", "ENJ":
		assetType = "nft_gaming"
	case "ATOM", "NEAR", "FTM", "ONE", "EGLD", "FLOW":
		assetType = "layer1"
	}
	category.AssetType = assetType

	// 根据权限确定交易类型
	if s.containsString(info.Permissions, "LEVERAGED") {
		category.TradeType = "leveraged"
	} else if s.containsString(info.Permissions, "MARGIN") {
		category.TradeType = "margin"
	} else {
		category.TradeType = "spot_only"
	}

	// 设置市值规模（这里使用简化的逻辑）
	category.MarketCap = "mid"

	// 设置订单级别（简化为basic）
	category.OrderLevel = "basic"

	return category
}

// getDefaultSymbolCategory 获取默认的交易对分类信息（当无法获取真实数据时使用）
func (s *Server) getDefaultSymbolCategory(symbol string, kind string) *SymbolCategory {
	// 解析基础资产
	baseAsset := symbol
	if strings.HasSuffix(symbol, "USDT") {
		baseAsset = strings.TrimSuffix(symbol, "USDT")
	} else if strings.HasSuffix(symbol, "BUSD") {
		baseAsset = strings.TrimSuffix(symbol, "BUSD")
	} else if strings.HasSuffix(symbol, "USDC") {
		baseAsset = strings.TrimSuffix(symbol, "USDC")
	} else if strings.HasSuffix(symbol, "_PERP") {
		baseAsset = strings.TrimSuffix(symbol, "_PERP")
	}

	// 简单的资产类型分类
	assetType := "emerging"
	switch baseAsset {
	case "BTC", "ETH", "BNB", "ADA", "SOL", "DOT", "AVAX", "MATIC", "LINK", "LTC", "XRP", "TRX", "ETC", "BCH":
		assetType = "major"
	case "USDT", "USDC", "BUSD", "DAI", "TUSD", "USDP":
		assetType = "stable"
	case "UNI", "AAVE", "SUSHI", "COMP", "MKR", "SNX", "CRV":
		assetType = "defi"
	case "SHIB", "DOGE", "PEPE", "BONK", "TURBO":
		assetType = "meme"
	case "MANA", "SAND", "GALA", "AXS", "ENJ":
		assetType = "nft_gaming"
	case "ATOM", "NEAR", "FTM", "ONE", "EGLD", "FLOW":
		assetType = "layer1"
	}

	// 根据交易类型设置权限
	var permissions []string
	var tradeType string
	if kind == "futures" {
		permissions = []string{"SPOT", "MARGIN", "LEVERAGED"}
		tradeType = "leveraged"
	} else {
		permissions = []string{"SPOT"}
		tradeType = "spot_only"
	}

	return &SymbolCategory{
		Symbol:      symbol,
		Status:      "TRADING",
		Permissions: permissions,
		BaseAsset:   baseAsset,
		QuoteAsset:  "USDT",
		AssetType:   assetType,
		MarketCap:   "mid",
		TradeType:   tradeType,
		OrderLevel:  "basic",
		IsActive:    true,
	}
}

// getRealtimeDataConcurrently 并发获取实时数据
func (s *Server) getRealtimeDataConcurrently(ctx context.Context, symbol string, kind string) (RealtimeData, bool) {
	// 1. 尝试从币安WebSocket获取实时数据（最高优先级）
	if realtimeData, success := s.getRealtimeDataFromWS(symbol, kind); success {
		return realtimeData, true
	}

	// 2. WebSocket失败时，降级使用HTTP API组合数据
	var price, change24h, volume24h float64

	// 使用并发获取价格和24h数据
	var wg sync.WaitGroup
	var priceErr error

	wg.Add(3)

	// 获取当前价格
	go func() {
		defer wg.Done()
		p, err := s.priceService.GetCurrentPrice(ctx, symbol, kind)
		if err == nil && p > 0 {
			price = p
		} else {
			priceErr = err
		}
	}()

	// 获取24小时涨跌幅
	go func() {
		defer wg.Done()
		change24h = s.getPriceChange24hWithKind(symbol, kind)
	}()

	// 获取24小时成交量
	go func() {
		defer wg.Done()
		volume24h = s.getVolume24hWithKind(symbol, kind)
	}()

	wg.Wait()

	// 检查是否获取到有效价格
	if price <= 0 {
		if priceErr != nil {
			log.Printf("[涨幅榜] %s 无法获取价格: %v", symbol, priceErr)
		} else {
			log.Printf("[涨幅榜] %s 价格无效: %.4f", symbol, price)
		}
		return RealtimeData{}, false
	}

	// 获取分类信息
	category := s.getSymbolCategory(symbol, kind)

	realtimeData := RealtimeData{
		Symbol:        symbol,
		LastPrice:     price,
		ChangePercent: change24h,
		Volume:        volume24h,
		DataSource:    "http_api",
		Timestamp:     time.Now().Unix(),
		Category:      category,
	}

	return realtimeData, true
}

// getRealtimeDataFromWS 从WebSocket获取实时数据
func (s *Server) getRealtimeDataFromWS(symbol string, kind string) (RealtimeData, bool) {
	if s.binanceWSClient == nil || !s.binanceWSClient.IsConnected() {
		return RealtimeData{}, false
	}

	// 转换交易对格式以匹配WebSocket数据
	var wsSymbol string
	switch kind {
	case "futures":
		if strings.HasSuffix(symbol, "USDT") {
			baseSymbol := strings.TrimSuffix(symbol, "USDT")
			wsSymbol = baseSymbol + "USD_PERP"
		} else if strings.HasSuffix(symbol, "USD_PERP") {
			wsSymbol = symbol // 已经是币本位格式
		} else {
			wsSymbol = symbol + "USD_PERP"
		}
	default:
		wsSymbol = symbol + "USDT" // 现货统一使用USDT格式
	}

	// 获取WebSocket数据
	if ticker, exists := s.binanceWSClient.GetTicker24h(wsSymbol); exists {
		lastPrice, err1 := strconv.ParseFloat(ticker.LastPrice, 64)
		changePercent, err2 := strconv.ParseFloat(ticker.PriceChangePercent, 64)
		volume, err3 := strconv.ParseFloat(ticker.TotalTradedBaseAsset, 64)

		// 数据验证
		if err1 != nil || err2 != nil || err3 != nil || lastPrice <= 0 {
			log.Printf("[DEBUG] WebSocket数据解析失败 %s -> %s: price=%s, change=%s, volume=%s",
				symbol, wsSymbol, ticker.LastPrice, ticker.PriceChangePercent, ticker.TotalTradedBaseAsset)
			return RealtimeData{}, false
		}

		// 获取分类信息
		category := s.getSymbolCategory(symbol, kind)

		return RealtimeData{
			Symbol:        symbol,
			LastPrice:     lastPrice,
			ChangePercent: changePercent,
			Volume:        volume,
			DataSource:    "websocket",
			Timestamp:     time.Now().Unix(),
			Category:      category,
		}, true
	}

	return RealtimeData{}, false
}

// validateRealtimeData 验证实时数据的质量和合理性
func (s *Server) validateRealtimeData(data RealtimeData) bool {
	// 基本数据完整性检查
	if data.Symbol == "" {
		log.Printf("[数据验证] 缺少交易对符号")
		return false
	}

	// 交易对格式验证
	if !s.isValidSymbolFormat(data.Symbol) {
		log.Printf("[数据验证] %s 交易对格式无效", data.Symbol)
		return false
	}

	// 价格验证
	if data.LastPrice <= 0 {
		log.Printf("[数据验证] %s 价格异常: %.4f", data.Symbol, data.LastPrice)
		return false
	}

	// 价格合理性检查（根据币种类型设置不同的阈值）
	maxPrice, minPrice := s.getPriceThresholds(data.Symbol)
	if data.LastPrice > maxPrice || data.LastPrice < minPrice {
		log.Printf("[数据验证] %s 价格超出合理范围 [%.8f, %.0f]: %.8f",
			data.Symbol, minPrice, maxPrice, data.LastPrice)
		return false
	}

	// 涨跌幅合理性检查
	if math.Abs(data.ChangePercent) > 1000 {
		log.Printf("[数据验证] %s 涨跌幅异常: %.2f%%", data.Symbol, data.ChangePercent)
		return false
	}

	// 智能涨跌幅检查（基于历史波动率）
	if math.Abs(data.ChangePercent) > 100 {
		// 对于高波动币种放宽限制
		if !s.isHighVolatilitySymbol(data.Symbol) {
			log.Printf("[数据验证] %s 涨跌幅过高: %.2f%%", data.Symbol, data.ChangePercent)
			return false
		}
	}

	// 成交量合理性检查
	if data.Volume < 0 {
		log.Printf("[数据验证] %s 成交量为负数: %.2f", data.Symbol, data.Volume)
		return false
	}

	// 成交量下限检查（避免虚假数据）
	minVolume := s.getMinVolumeThreshold(data.Symbol)
	if data.Volume < minVolume {
		//log.Printf("[数据验证] %s 成交量过低: %.2f (最低要求: %.2f)",
		//	data.Symbol, data.Volume, minVolume)
		return false
	}

	// 时间戳检查（不允许超过30分钟的数据）
	if data.Timestamp > 0 {
		age := time.Now().Unix() - data.Timestamp
		if age > 1800 { // 30分钟
			log.Printf("[数据验证] %s 数据太旧: %d秒前", data.Symbol, age)
			return false
		}
		if age < -300 { // 不允许未来5分钟的数据
			log.Printf("[数据验证] %s 时间戳异常（未来时间）: %d秒后", data.Symbol, -age)
			return false
		}
	}

	// 数据源有效性检查
	validSources := map[string]bool{
		"websocket":  true,
		"http_api":   true,
		"kline_calc": true,
	}
	if !validSources[data.DataSource] {
		log.Printf("[数据验证] %s 数据源无效: %s", data.Symbol, data.DataSource)
		return false
	}

	return true
}

// getPriceThresholds 根据币种获取价格合理性阈值
func (s *Server) getPriceThresholds(symbol string) (maxPrice, minPrice float64) {
	// BTC相关
	if strings.Contains(symbol, "BTC") {
		return 1000000, 0.00000001
	}

	// ETH相关
	if strings.Contains(symbol, "ETH") {
		return 100000, 0.0000001
	}

	// 主流币种
	if strings.Contains(symbol, "BNB") || strings.Contains(symbol, "ADA") ||
		strings.Contains(symbol, "XRP") || strings.Contains(symbol, "SOL") ||
		strings.Contains(symbol, "DOT") {
		return 10000, 0.000001
	}

	// 默认值
	return 100000, 0.00000001
}

// isHighVolatilitySymbol 检查是否为高波动性币种
func (s *Server) isHighVolatilitySymbol(symbol string) bool {
	highVolatilitySymbols := []string{
		"SHIB", "DOGE", "PEPE", "BONK", "TURBO", "PUMP", "NEIRO",
		"DEGEN", "WIF", "MEW", "CUMMIES", "BALD", "HODL",
	}

	baseSymbol := symbol
	if idx := strings.Index(symbol, "USDT"); idx > 0 {
		baseSymbol = symbol[:idx]
	}

	for _, highVolSymbol := range highVolatilitySymbols {
		if strings.Contains(baseSymbol, highVolSymbol) {
			return true
		}
	}

	return false
}

// getMinVolumeThreshold 获取最小成交量阈值
func (s *Server) getMinVolumeThreshold(symbol string) float64 {
	// 大市值币种要求更高的成交量
	if strings.Contains(symbol, "BTC") || strings.Contains(symbol, "ETH") {
		return 100000 // 10万美金
	}

	// 中等市值币种
	if strings.Contains(symbol, "BNB") || strings.Contains(symbol, "ADA") ||
		strings.Contains(symbol, "XRP") || strings.Contains(symbol, "SOL") {
		return 10000 // 1万美金
	}

	// 小币种放宽要求
	return 1000 // 1000美金
}

// getFallbackRealtimeData 获取降级实时数据（当主要数据源失败时使用）
func (s *Server) getFallbackRealtimeData(symbol string, kind string) RealtimeData {
	// 尝试从缓存中获取最近的数据
	cacheKey := fmt.Sprintf("price_%s_%s", symbol, kind)
	if cached, exists := s.getCachedPriceData(cacheKey); exists {
		log.Printf("[降级数据] %s 使用缓存价格数据", symbol)
		return cached
	}

	// 如果缓存也没有，尝试从数据库获取最近的历史数据
	if historicalData, err := s.getHistoricalPriceData(symbol, kind); err == nil {
		log.Printf("[降级数据] %s 使用历史价格数据", symbol)
		return historicalData
	}

	// 如果都没有，返回空数据
	log.Printf("[降级数据] %s 无可用降级数据", symbol)
	return RealtimeData{
		Symbol:     symbol,
		DataSource: "unavailable",
		Timestamp:  time.Now().Unix(),
	}
}

// getCachedPriceData 从缓存获取价格数据
func (s *Server) getCachedPriceData(key string) (RealtimeData, bool) {
	// 这里可以实现一个简单的价格缓存
	// 为了简化，我们返回false表示没有缓存
	return RealtimeData{}, false
}

// getHistoricalPriceData 从数据库获取历史价格数据
func (s *Server) getHistoricalPriceData(symbol string, kind string) (RealtimeData, error) {
	// 尝试从数据库获取最近的K线数据作为降级数据
	klines, err := s.getKlinesData(symbol, kind, 1, 1) // 获取最近1根K线
	if err != nil || len(klines) == 0 {
		return RealtimeData{}, fmt.Errorf("no historical data available")
	}

	kline := klines[0]

	// 转换字符串价格为float64
	closePrice, err := strconv.ParseFloat(kline.Close, 64)
	if err != nil {
		return RealtimeData{}, fmt.Errorf("invalid close price: %s", kline.Close)
	}

	volume, err := strconv.ParseFloat(kline.Volume, 64)
	if err != nil {
		return RealtimeData{}, fmt.Errorf("invalid volume: %s", kline.Volume)
	}

	// 计算24小时涨跌幅（需要更多历史数据，这里简化处理）
	changePercent := 0.0
	if len(klines) > 1 {
		prevKline := klines[1]
		if prevClose, err := strconv.ParseFloat(prevKline.Close, 64); err == nil && prevClose > 0 {
			changePercent = (closePrice - prevClose) / prevClose * 100
		}
	}

	return RealtimeData{
		Symbol:        symbol,
		LastPrice:     closePrice,
		ChangePercent: changePercent,
		Volume:        volume,
		DataSource:    "historical_fallback",
		Timestamp:     int64(kline.OpenTime / 1000), // 转换为秒
	}, nil
}

// getKlinesData 获取K线数据（简化版，用于降级数据获取）
func (s *Server) getKlinesData(symbol, kind string, limit, interval int) ([]KlineData, error) {
	// 这里应该调用实际的K线数据获取逻辑
	// 为了简化，返回空结果
	return []KlineData{}, fmt.Errorf("kline data not available")
}

// isValidSymbolFormat 验证交易对格式是否有效
func (s *Server) isValidSymbolFormat(symbol string) bool {
	if symbol == "" {
		return false
	}

	// 检查是否包含常见的交易对格式
	validPatterns := []string{
		"USDT$", "USDC$", "BUSD$", "BTC$", "ETH$", "BNB$", "ADA$", "SOL$", "DOT$",
		"_PERP$", // 合约后缀
	}

	for _, pattern := range validPatterns {
		matched, _ := regexp.MatchString(pattern, symbol)
		if matched {
			return true
		}
	}

	// 允许一些特殊格式（如稳定币对）
	if strings.Contains(symbol, "USD") || strings.Contains(symbol, "EUR") {
		return true
	}

	return false
}

// monitorDataQuality 监控数据质量和异常情况
func (s *Server) monitorDataQuality(gainers []gin.H, kind string) {
	if len(gainers) == 0 {
		log.Printf("[数据监控] 警告: %s市场没有获取到任何涨幅数据", kind)
		return
	}

	// 统计各种指标
	stats := s.calculateDataStats(gainers)

	// 检测异常情况
	warnings := s.detectDataAnomalies(stats, gainers)

	// 数据源分布统计
	dataSourceStats := s.calculateDataSourceStats(gainers)

	// 输出监控结果
	log.Printf("[数据监控] %s市场统计: 总数=%d, 上涨=%d, 下跌=%d, 平盘=%d",
		kind, stats.totalCount, stats.positiveCount, stats.negativeCount, stats.zeroCount)
	log.Printf("[数据监控] %s市场指标: 平均涨幅=%.2f%%, 平均成交量=%.0f, 波动率=%.2f%%",
		kind, stats.avgChange, stats.avgVolume, stats.volatility)
	log.Printf("[数据监控] %s市场极值: 最高%.1f%%, 最低%.1f%%, 最大成交量%.0f",
		kind, stats.maxChange, stats.minChange, stats.maxVolume)
	log.Printf("[数据监控] %s数据源分布: WebSocket=%d, HTTP_API=%d, K线计算=%d",
		kind, dataSourceStats.websocket, dataSourceStats.httpApi, dataSourceStats.klineCalc)

	if len(warnings) > 0 {
		log.Printf("[数据监控] %s市场异常检测: %v", kind, warnings)
	}
}

// DataStats 数据统计结构
type DataStats struct {
	totalCount    int
	positiveCount int
	negativeCount int
	zeroCount     int
	totalChange   float64
	totalVolume   float64
	avgChange     float64
	avgVolume     float64
	maxChange     float64
	minChange     float64
	maxVolume     float64
	minVolume     float64
	volatility    float64
}

// DataSourceStats 数据源统计
type DataSourceStats struct {
	websocket int
	httpApi   int
	klineCalc int
}

// calculateDataStats 计算数据统计
func (s *Server) calculateDataStats(gainers []gin.H) *DataStats {
	stats := &DataStats{
		minChange: 999,
		minVolume: 999999999,
		maxChange: -999,
	}

	for _, gainer := range gainers {
		change, _ := gainer["price_change_24h"].(float64)
		volume, _ := gainer["volume_24h"].(float64)

		stats.totalCount++
		stats.totalChange += change
		stats.totalVolume += volume

		if change > 0 {
			stats.positiveCount++
		} else if change < 0 {
			stats.negativeCount++
		} else {
			stats.zeroCount++
		}

		if change > stats.maxChange {
			stats.maxChange = change
		}
		if change < stats.minChange {
			stats.minChange = change
		}

		if volume > stats.maxVolume {
			stats.maxVolume = volume
		}
		if volume < stats.minVolume && volume > 0 {
			stats.minVolume = volume
		}
	}

	if stats.totalCount > 0 {
		stats.avgChange = stats.totalChange / float64(stats.totalCount)
		stats.avgVolume = stats.totalVolume / float64(stats.totalCount)

		// 计算波动率（标准差）
		if stats.totalCount > 1 {
			var sumSquares float64
			for _, gainer := range gainers {
				change, _ := gainer["price_change_24h"].(float64)
				diff := change - stats.avgChange
				sumSquares += diff * diff
			}
			variance := sumSquares / float64(stats.totalCount-1)
			stats.volatility = math.Sqrt(variance)
		}
	}

	return stats
}

// calculateDataSourceStats 计算数据源分布
func (s *Server) calculateDataSourceStats(gainers []gin.H) *DataSourceStats {
	stats := &DataSourceStats{}

	for _, gainer := range gainers {
		dataSource, _ := gainer["data_source"].(string)
		switch dataSource {
		case "websocket":
			stats.websocket++
		case "http_api":
			stats.httpApi++
		case "kline_calc":
			stats.klineCalc++
		}
	}

	return stats
}

// detectDataAnomalies 检测数据异常
func (s *Server) detectDataAnomalies(stats *DataStats, gainers []gin.H) []string {
	warnings := []string{}

	// 检查数据分布是否正常
	zeroRatio := float64(stats.zeroCount) / float64(stats.totalCount) * 100
	if zeroRatio > 50 {
		warnings = append(warnings, fmt.Sprintf("超过%.1f%%的数据涨幅为0", zeroRatio))
	}

	// 检查是否有极端涨幅
	if math.Abs(stats.maxChange) > 100 || math.Abs(stats.minChange) > 100 {
		warnings = append(warnings, fmt.Sprintf("存在极端涨幅: 最高%.1f%%, 最低%.1f%%", stats.maxChange, stats.minChange))
	}

	// 检查波动率是否异常
	if stats.volatility > 20 {
		warnings = append(warnings, fmt.Sprintf("波动率过高: %.2f%%", stats.volatility))
	}

	// 检查成交量是否异常
	if stats.avgVolume < 1000 {
		warnings = append(warnings, fmt.Sprintf("平均成交量过低: %.0f", stats.avgVolume))
	}

	// 检查数据源单一性
	if len(gainers) > 10 {
		// 如果95%以上的数据来自单一数据源，可能存在问题
		maxSourceCount := 0
		for _, gainer := range gainers {
			dataSource, _ := gainer["data_source"].(string)
			count := 0
			for _, g := range gainers {
				if ds, _ := g["data_source"].(string); ds == dataSource {
					count++
				}
			}
			if count > maxSourceCount {
				maxSourceCount = count
			}
		}

		sourceDominance := float64(maxSourceCount) / float64(len(gainers)) * 100
		if sourceDominance > 95 {
			warnings = append(warnings, fmt.Sprintf("数据源过于单一: %.1f%%来自同一数据源", sourceDominance))
		}
	}

	return warnings
}

// 涨幅榜数据缓存
var gainersCache = make(map[string]cachedGainersData)
var gainersCacheMu sync.RWMutex

type cachedGainersData struct {
	data      []gin.H
	expiresAt time.Time
}

// cacheGainers 缓存涨幅榜数据
func (s *Server) cacheGainers(key string, data []gin.H) {
	gainersCacheMu.Lock()
	defer gainersCacheMu.Unlock()

	gainersCache[key] = cachedGainersData{
		data:      data,
		expiresAt: time.Now().Add(30 * time.Second),
	}
}

// getCachedGainers 获取缓存的涨幅榜数据
func (s *Server) getCachedGainers(key string) ([]gin.H, bool) {
	gainersCacheMu.RLock()
	defer gainersCacheMu.RUnlock()

	if cached, exists := gainersCache[key]; exists && time.Now().Before(cached.expiresAt) {
		return cached.data, true
	}

	return nil, false
}

// getCachedGainersWithDuration 获取指定时长内的缓存数据
func (s *Server) getCachedGainersWithDuration(key string, maxAge time.Duration) ([]gin.H, bool) {
	gainersCacheMu.RLock()
	defer gainersCacheMu.RUnlock()

	if cached, exists := gainersCache[key]; exists {
		age := time.Since(cached.expiresAt.Add(-maxAge))
		if age <= maxAge {
			return cached.data, true
		}
	}

	return nil, false
}

// cacheGainersWithDuration 使用指定时长缓存涨幅榜数据
func (s *Server) cacheGainersWithDuration(key string, data []gin.H, duration time.Duration) {
	gainersCacheMu.Lock()
	defer gainersCacheMu.Unlock()

	gainersCache[key] = cachedGainersData{
		data:      data,
		expiresAt: time.Now().Add(duration),
	}
}

// filterAndSortGainers 筛选和排序涨幅榜数据
func (s *Server) filterAndSortGainers(gainers []gin.H, sortBy, sortOrder string, filterPositiveOnly, filterLargeCap bool, minVolume float64, limit int) []gin.H {
	if len(gainers) == 0 {
		return gainers
	}

	// 应用筛选条件
	filtered := make([]gin.H, 0, len(gainers))
	for _, gainer := range gainers {
		// 只显示上涨币种筛选
		if filterPositiveOnly {
			if change, ok := gainer["price_change_24h"].(float64); !ok || change <= 0 {
				continue
			}
		}

		// 大市值币种筛选
		if filterLargeCap {
			price, priceOk := gainer["current_price"].(float64)
			volume, volumeOk := gainer["volume_24h"].(float64)
			if !priceOk || !volumeOk {
				continue
			}
			// 简单的市值计算：价格 * 成交量 > 100万
			if price*volume <= 1000000 {
				continue
			}
		}

		// 最小成交量筛选
		if minVolume > 0 {
			if volume, ok := gainer["volume_24h"].(float64); !ok || volume < minVolume {
				continue
			}
		}

		filtered = append(filtered, gainer)
	}

	// 应用排序
	sort.Slice(filtered, func(i, j int) bool {
		var compareResult bool

		switch sortBy {
		case "volume":
			volI, _ := filtered[i]["volume_24h"].(float64)
			volJ, _ := filtered[j]["volume_24h"].(float64)
			compareResult = volI < volJ // 升序：小成交量在前
		case "symbol":
			symI, _ := filtered[i]["symbol"].(string)
			symJ, _ := filtered[j]["symbol"].(string)
			compareResult = symI < symJ // 字典序
		case "change":
		default: // 默认按涨幅排序
			changeI, _ := filtered[i]["price_change_24h"].(float64)
			changeJ, _ := filtered[j]["price_change_24h"].(float64)
			compareResult = changeI < changeJ // 升序：涨幅小的在前
		}

		// 根据排序顺序决定是否反转
		if sortOrder == "desc" {
			return !compareResult
		}
		return compareResult
	})

	// 限制返回数量
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// 重新分配排名
	for i, gainer := range filtered {
		gainer["rank"] = i + 1
	}

	return filtered
}

// hasSignificantChanges 检查涨幅榜数据是否有显著变化
func (s *Server) hasSignificantChanges(oldData, newData []gin.H) bool {
	if len(oldData) != len(newData) {
		return true // 数量不同，肯定有变化
	}

	// 只检查涨幅变化，不再参考价格和排名
	for i := 0; i < len(oldData) && i < 10; i++ {
		oldGainer := oldData[i]
		newGainer := newData[i]

		// 检查涨幅变化（超过0.1%的变化算显著）
		oldChange, _ := oldGainer["price_change_24h"].(float64)
		newChange, _ := newGainer["price_change_24h"].(float64)
		if math.Abs(newChange-oldChange) > 0.1 {
			return true
		}
	}

	return false // 无显著变化
}

// GetRealTimeGainers 获取实时涨幅榜
// GET /market/binance/realtime-gainers?kind=spot&limit=15&sort_by=change&sort_order=desc&filter_positive_only=false&filter_large_cap=false
func (s *Server) GetRealTimeGainers(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	category := strings.ToLower(strings.TrimSpace(c.DefaultQuery("category", "all")))
	limitStr := c.DefaultQuery("limit", "15")
	sortBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_by", "change")))     // change, volume, symbol
	sortOrder := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_order", "desc"))) // asc, desc
	filterPositiveOnlyStr := c.DefaultQuery("filter_positive_only", "false")
	filterLargeCapStr := c.DefaultQuery("filter_large_cap", "false")
	minVolumeStr := c.DefaultQuery("min_volume", "")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 15 // 默认15个
	}

	// 解析筛选参数
	filterPositiveOnly := strings.ToLower(filterPositiveOnlyStr) == "true"
	filterLargeCap := strings.ToLower(filterLargeCapStr) == "true"
	var minVolume float64
	if minVolumeStr != "" {
		if mv, err := strconv.ParseFloat(minVolumeStr, 64); err == nil && mv >= 0 {
			minVolume = mv
		}
	}

	// 获取基础数据（获取更多数据用于筛选）
	baseLimit := limit * 10 // 获取10倍数据用于筛选，确保有足够的数据
	if baseLimit > 500 {
		baseLimit = 500
	}

	// 优先使用同步器数据（与WebSocket使用相同数据源）
	log.Printf("[涨幅榜] 尝试从数据同步器获取最新数据，市场类型: %s", kind)
	gainers, err := s.getRealtimeGainersFromSyncer(kind, baseLimit)
	if err != nil {
		log.Printf("[涨幅榜] 数据同步器数据不可用，降级到数据库查询: %v", err)

		// 降级：使用优化版本（直接从 binance_24h_stats 查询）
		gainers, err = s.generateRealtimeGainersFrom24hStats(c.Request.Context(), kind, category, baseLimit)
		if err != nil {
			log.Printf("[涨幅榜] 优化版本失败，降级到传统版本: %v", err)
			// 降级到传统版本
			gainers, err = s.generateRealtimeGainersData(c.Request.Context(), kind, category, baseLimit)
			if err != nil {
				log.Printf("[ERROR] 传统版本也失败: %v", err)
				s.InternalServerError(c, "获取涨幅榜数据失败", err)
				return
			}
		}
	} else {
		log.Printf("[涨幅榜] ✅ 成功从数据同步器获取%d条数据，现在应用%s分类筛选", len(gainers), category)

		// 对同步器数据应用category筛选
		gainers, err = s.applyCategoryFilter(gainers, category)
		if err != nil {
			log.Printf("[涨幅榜] 分类筛选失败，降级到数据库查询: %v", err)
			// 降级到数据库查询
			gainers, err = s.generateRealtimeGainersFrom24hStats(c.Request.Context(), kind, category, baseLimit)
			if err != nil {
				log.Printf("[涨幅榜] 优化版本失败，降级到传统版本: %v", err)
				gainers, err = s.generateRealtimeGainersData(c.Request.Context(), kind, category, baseLimit)
				if err != nil {
					log.Printf("[ERROR] 传统版本也失败: %v", err)
					s.InternalServerError(c, "获取涨幅榜数据失败", err)
					return
				}
			}
		}
	}

	// 应用筛选和排序
	filteredGainers := s.filterAndSortGainers(gainers, sortBy, sortOrder, filterPositiveOnly, filterLargeCap, minVolume, limit)

	c.JSON(http.StatusOK, gin.H{
		"kind":                 kind,
		"limit":                limit,
		"sort_by":              sortBy,
		"sort_order":           sortOrder,
		"filter_positive_only": filterPositiveOnly,
		"filter_large_cap":     filterLargeCap,
		"min_volume":           minVolume,
		"gainers":              filteredGainers,
		"count":                len(filteredGainers),
		"total_available":      len(gainers),
		"timestamp":            time.Now().Unix(),
	})
}

// GetCurrentPriceHTTP 获取指定币种的当前价格 (HTTP handler)
// GET /api/v1/market/price/:symbol?kind=spot
func (s *Server) GetCurrentPriceHTTP(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	kind := c.DefaultQuery("kind", "spot")

	// 获取当前价格
	price, err := s.getCurrentPrice(c.Request.Context(), symbol, kind)
	if err != nil {
		log.Printf("[ERROR] 获取当前价格失败 %s: %v", symbol, err)
		c.JSON(500, gin.H{"error": "获取价格失败"})
		return
	}

	c.JSON(200, gin.H{
		"symbol":    symbol,
		"price":     price,
		"timestamp": time.Now().Unix(),
	})
}

// GetBatchCurrentPrices 批量获取当前价格
// POST /api/v1/market/batch-prices
func (s *Server) GetBatchCurrentPrices(c *gin.Context) {
	var body struct {
		Symbols []string `json:"symbols"`
		Kind    string   `json:"kind"`
	}

	if err := c.BindJSON(&body); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if len(body.Symbols) == 0 {
		c.JSON(400, gin.H{"error": "symbols array cannot be empty"})
		return
	}

	if body.Kind == "" {
		body.Kind = "spot"
	}

	// 限制最大数量
	if len(body.Symbols) > 100 {
		c.JSON(400, gin.H{"error": "too many symbols, maximum 100 allowed"})
		return
	}

	// 批量获取价格
	prices, err := s.priceService.BatchGetCurrentPrices(c.Request.Context(), body.Symbols, body.Kind)
	if err != nil {
		log.Printf("[ERROR] 批量获取价格失败: %v", err)
		c.JSON(500, gin.H{"error": "批量获取价格失败"})
		return
	}

	// 转换为前端需要的格式
	result := make([]gin.H, 0, len(body.Symbols))
	for _, symbol := range body.Symbols {
		price := prices[symbol]
		result = append(result, gin.H{
			"symbol": symbol,
			"price":  price,
		})
	}

	c.JSON(200, gin.H{
		"data":  result,
		"count": len(result),
	})
}

// GetKlines 获取K线数据
// GET /api/v1/market/klines/:symbol?interval=1h&limit=100&kind=spot&aggregate=4h
func (s *Server) GetKlines(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	interval := c.DefaultQuery("interval", "1h")
	limitStr := c.DefaultQuery("limit", "100")
	kind := c.DefaultQuery("kind", "spot")
	aggregate := c.Query("aggregate") // 可选的聚合目标间隔

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100 // 默认100条
	}

	// 获取K线数据（使用缓存机制和数据验证）
	klines, err := s.getKlinesWithCache(c.Request.Context(), symbol, kind, interval, limit)
	if err != nil {
		log.Printf("[ERROR] 获取K线数据失败 %s: %v", symbol, err)
		c.JSON(500, gin.H{"error": "获取K线数据失败"})
		return
	}

	// 如果指定了聚合间隔，进行数据聚合
	if aggregate != "" && aggregate != interval {
		aggregatedKlines, err := s.ConvertKlineInterval(klines, interval, aggregate, symbol, kind)
		if err != nil {
			log.Printf("[WARNING] K线数据聚合失败 %s: %v", symbol, err)
			// 聚合失败时使用原始数据
		} else {
			klines = aggregatedKlines
			interval = aggregate // 更新返回的间隔信息
			log.Printf("[KlineAggregation] 聚合成功: %s %s → %d 条", symbol, aggregate, len(klines))
		}
	}

	// 获取验证和处理后的数据
	validatedKlines, err := s.ValidateAndCleanKlines(klines, symbol, interval, kind)
	if err != nil {
		log.Printf("[WARNING] K线数据验证失败 %s: %v", symbol, err)
		// 即使验证失败，也返回原始数据
	}

	log.Printf("[DEBUG] 获取到K线数据: symbol=%s, interval=%s, count=%d", symbol, interval, len(klines))

	// 转换为前端需要的格式，包含数据质量信息
	result := make([]gin.H, len(klines))
	for i, kline := range klines {
		klineData := gin.H{
			"timestamp": kline.OpenTime,
			"open":      kline.Open,
			"high":      kline.High,
			"low":       kline.Low,
			"close":     kline.Close,
			"volume":    kline.Volume,
		}

		// 如果有验证数据，添加额外信息
		if i < len(validatedKlines) {
			klineData["is_valid"] = validatedKlines[i].IsValid
			klineData["data_quality"] = validatedKlines[i].DataQuality
		}

		result[i] = klineData
	}

	response := gin.H{
		"symbol":   symbol,
		"interval": interval,
		"data":     result,
		"count":    len(result),
	}

	// 如果进行了聚合，添加聚合信息
	if aggregate != "" && aggregate != c.DefaultQuery("interval", "1h") {
		response["aggregated"] = true
		response["original_interval"] = c.DefaultQuery("interval", "1h")
	}

	c.JSON(200, response)
}

// GetRecommendationPerformance 获取推荐历史表现
// GET /api/v1/recommend/performance/:symbol?period=30d
func (s *Server) GetRecommendationPerformance(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	period := c.DefaultQuery("period", "30d")

	// 解析时间周期
	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "1y":
		days = 365
	default:
		days = 30
	}

	// 从数据库获取该symbol的历史性能数据
	performances, err := pdb.GetPerformanceBySymbol(s.db.DB(), symbol, 1000) // 获取最近1000条记录
	if err != nil {
		log.Printf("[ERROR] 获取%s历史性能数据失败: %v", symbol, err)
		c.JSON(500, gin.H{"error": "获取历史数据失败"})
		return
	}

	log.Printf("[DEBUG] Found %d total performances for %s", len(performances), symbol)

	// 过滤指定时间周期内的数据
	now := time.Now().UTC()
	cutoffTime := now.AddDate(0, 0, -days)
	var filteredPerformances []pdb.RecommendationPerformance

	for _, perf := range performances {
		if perf.RecommendedAt.After(cutoffTime) {
			filteredPerformances = append(filteredPerformances, perf)
		}
	}

	log.Printf("[DEBUG] Found %d performances within %d days for %s", len(filteredPerformances), days, symbol)

	// 计算基于真实数据的统计指标
	stats := s.calculateRealPerformanceStats(filteredPerformances, symbol, period, days)

	// 如果没有历史数据，基于技术指标生成更真实的模拟数据
	if len(filteredPerformances) == 0 {
		log.Printf("[INFO] No historical data for %s, generating realistic simulated data based on technical indicators", symbol)
		stats = s.generateRealisticSimulatedStats(symbol, period, days)
	}

	// 获取当前价格
	currentPrice := 45000.0 // 默认价格
	if price, err := s.getCurrentPrice(c.Request.Context(), symbol, "spot"); err == nil && price > 0 {
		currentPrice = price
	}

	// 构建完整的性能数据响应
	performance := gin.H{
		"symbol":            symbol,
		"period":            period,
		"overall_score":     stats.OverallScore,
		"technical_score":   stats.TechnicalScore,
		"fundamental_score": stats.FundamentalScore,
		"sentiment_score":   stats.SentimentScore,
		"momentum_score":    stats.MomentumScore,

		// 收益相关因子
		"return_factor":      stats.ReturnFactor,
		"risk_factor":        stats.RiskFactor,
		"consistency_factor": stats.ConsistencyFactor,
		"timing_factor":      stats.TimingFactor,

		// 传统性能指标
		"total_return":      stats.TotalReturn,
		"annualized_return": stats.AnnualizedReturn,
		"max_drawdown":      stats.MaxDrawdown,
		"sharpe_ratio":      stats.SharpeRatio,
		"win_rate":          stats.WinRate,
		"profit_factor":     stats.ProfitFactor,

		// 风险指标
		"volatility":         stats.Volatility,
		"var_95":             stats.VaR95,
		"expected_shortfall": stats.ExpectedShortfall,

		// 市场数据
		"current_price":    currentPrice,
		"price_change_24h": s.getPriceChange24h(symbol),
		"volume_24h":       s.getVolume24h(symbol),

		// 额外的性能统计数据
		"accuracy":              stats.Accuracy,
		"avg_return":            stats.AvgReturn,
		"avg_holding_time":      stats.AvgHoldingTime,
		"total_recommendations": stats.TotalRecommendations,
		"best_monthly_return":   stats.BestMonthlyReturn,
		"worst_monthly_return":  stats.WorstMonthlyReturn,

		// 时间戳
		"timestamp":     time.Now().Unix(),
		"calculated_at": time.Now().Format(time.RFC3339),
	}

	c.JSON(200, gin.H{
		"success":     true,
		"performance": performance,
		"period":      period,
		"timestamp":   time.Now().Unix(),
	})
}

// PerformanceStats 基于真实数据的性能统计
type PerformanceStats struct {
	OverallScore     float64
	TechnicalScore   float64
	FundamentalScore float64
	SentimentScore   float64
	MomentumScore    float64

	ReturnFactor      float64
	RiskFactor        float64
	ConsistencyFactor float64
	TimingFactor      float64

	TotalReturn      float64
	AnnualizedReturn float64
	MaxDrawdown      float64
	SharpeRatio      float64
	WinRate          float64
	ProfitFactor     float64

	Volatility        float64
	VaR95             float64
	ExpectedShortfall float64

	Accuracy             float64
	AvgReturn            float64
	AvgHoldingTime       string
	TotalRecommendations int
	BestMonthlyReturn    float64
	WorstMonthlyReturn   float64
}

// calculateRealPerformanceStats 基于真实历史数据计算性能统计
func (s *Server) calculateRealPerformanceStats(performances []pdb.RecommendationPerformance, symbol, period string, days int) *PerformanceStats {
	stats := &PerformanceStats{}

	if len(performances) == 0 {
		// 如果没有历史数据，返回默认值
		stats.OverallScore = 5.0
		stats.TechnicalScore = 5.0
		stats.FundamentalScore = 5.0
		stats.SentimentScore = 5.0
		stats.MomentumScore = 5.0
		stats.ReturnFactor = 5.0
		stats.RiskFactor = 3.0
		stats.ConsistencyFactor = 5.0
		stats.TimingFactor = 5.0
		stats.TotalReturn = 0.0
		stats.AnnualizedReturn = 0.0
		stats.MaxDrawdown = 5.0
		stats.SharpeRatio = 1.0
		stats.WinRate = 50.0
		stats.ProfitFactor = 1.0
		stats.Volatility = 10.0
		stats.VaR95 = 8.0
		stats.ExpectedShortfall = 10.0
		stats.Accuracy = 50.0
		stats.AvgReturn = 0.0
		stats.AvgHoldingTime = "3.0天"
		stats.TotalRecommendations = 0
		stats.BestMonthlyReturn = 5.0
		stats.WorstMonthlyReturn = -5.0
		return stats
	}

	// 计算基础统计
	totalRecords := len(performances)
	stats.TotalRecommendations = totalRecords

	// 计算胜率和准确率
	winCount := 0
	totalReturn := 0.0
	returns := make([]float64, 0, totalRecords)
	holdingPeriods := make([]int, 0)

	for _, perf := range performances {
		// 计算胜率（基于24小时收益率）
		if perf.Return24h != nil && *perf.Return24h > 0 {
			winCount++
		}

		// 收集收益率数据
		if perf.Return24h != nil {
			returns = append(returns, *perf.Return24h)
			totalReturn += *perf.Return24h
		}

		// 收集持仓周期数据
		if perf.HoldingPeriod != nil && *perf.HoldingPeriod > 0 {
			holdingPeriods = append(holdingPeriods, *perf.HoldingPeriod)
		}
	}

	// 计算胜率
	if totalRecords > 0 {
		stats.WinRate = float64(winCount) / float64(totalRecords) * 100
		stats.Accuracy = stats.WinRate // 准确率等于胜率
	}

	// 计算平均收益率
	if len(returns) > 0 {
		stats.AvgReturn = totalReturn / float64(len(returns))
		stats.TotalReturn = stats.AvgReturn * float64(days) / 30.0 // 按周期调整
	}

	// 计算年化收益率
	if days > 0 {
		stats.AnnualizedReturn = stats.TotalReturn * 365.0 / float64(days)
	}

	// 计算平均持仓时间
	if len(holdingPeriods) > 0 {
		totalMinutes := 0
		for _, period := range holdingPeriods {
			totalMinutes += period
		}
		avgMinutes := float64(totalMinutes) / float64(len(holdingPeriods))
		avgHours := avgMinutes / 60.0
		avgDays := avgHours / 24.0
		stats.AvgHoldingTime = fmt.Sprintf("%.1f天", avgDays)
	} else {
		stats.AvgHoldingTime = "3.0天"
	}

	// 计算最佳和最差月度收益（这里简化为最佳和最差单次收益）
	if len(returns) > 0 {
		best := returns[0]
		worst := returns[0]
		for _, ret := range returns {
			if ret > best {
				best = ret
			}
			if ret < worst {
				worst = ret
			}
		}
		stats.BestMonthlyReturn = best
		stats.WorstMonthlyReturn = worst
	} else {
		stats.BestMonthlyReturn = 5.0
		stats.WorstMonthlyReturn = -5.0
	}

	// 计算波动率（收益率的标准差）
	if len(returns) > 1 {
		mean := stats.AvgReturn
		sumSquares := 0.0
		for _, ret := range returns {
			diff := ret - mean
			sumSquares += diff * diff
		}
		variance := sumSquares / float64(len(returns)-1)
		stats.Volatility = math.Sqrt(variance)

		// 计算VaR95和Expected Shortfall
		stats.VaR95 = -stats.Volatility * 1.645           // 95%置信区间
		stats.ExpectedShortfall = -stats.Volatility * 2.0 // 简化的预期短缺
	} else {
		stats.Volatility = 10.0
		stats.VaR95 = 8.0
		stats.ExpectedShortfall = 10.0
	}

	// 计算最大回撤（简化为波动率的倍数）
	stats.MaxDrawdown = stats.Volatility * 2.0

	// 计算夏普比率
	if stats.Volatility > 0 {
		stats.SharpeRatio = stats.AvgReturn / stats.Volatility
	} else {
		stats.SharpeRatio = 1.0
	}

	// 计算利润因子
	if stats.WinRate > 0 && stats.WinRate < 100 {
		avgWin := 0.0
		avgLoss := 0.0
		winCount := 0
		lossCount := 0

		for _, ret := range returns {
			if ret > 0 {
				avgWin += ret
				winCount++
			} else {
				avgLoss += math.Abs(ret)
				lossCount++
			}
		}

		if winCount > 0 {
			avgWin /= float64(winCount)
		}
		if lossCount > 0 {
			avgLoss /= float64(lossCount)
		}

		if avgLoss > 0 {
			stats.ProfitFactor = avgWin / avgLoss
		} else {
			stats.ProfitFactor = 2.0
		}
	} else {
		stats.ProfitFactor = 1.5
	}

	// 基于历史数据计算评分因子
	// 收益因子：基于平均收益率
	stats.ReturnFactor = math.Min(10.0, math.Max(0.0, (stats.AvgReturn+10.0)*0.5))

	// 风险因子：基于波动率和最大回撤的反向
	riskScore := 10.0 - math.Min(10.0, stats.Volatility*0.5+stats.MaxDrawdown*0.3)
	stats.RiskFactor = math.Max(0.0, riskScore)

	// 一致性因子：基于胜率
	stats.ConsistencyFactor = stats.WinRate * 0.1

	// 时机把握因子：基于夏普比率
	stats.TimingFactor = math.Min(10.0, math.Max(0.0, stats.SharpeRatio*2.0))

	// 计算综合评分
	stats.OverallScore = (stats.ReturnFactor*0.4 + stats.RiskFactor*0.3 + stats.ConsistencyFactor*0.2 + stats.TimingFactor*0.1)
	stats.OverallScore = math.Round(stats.OverallScore*100) / 100

	// 其他评分（暂时使用默认值，未来可以基于历史数据改进）
	stats.TechnicalScore = 6.0
	stats.FundamentalScore = 5.5
	stats.SentimentScore = 5.8
	stats.MomentumScore = 6.2

	// 四舍五入所有数值
	stats.ReturnFactor = math.Round(stats.ReturnFactor*10) / 10
	stats.RiskFactor = math.Round(stats.RiskFactor*10) / 10
	stats.ConsistencyFactor = math.Round(stats.ConsistencyFactor*10) / 10
	stats.TimingFactor = math.Round(stats.TimingFactor*10) / 10
	stats.TotalReturn = math.Round(stats.TotalReturn*100) / 100
	stats.AnnualizedReturn = math.Round(stats.AnnualizedReturn*100) / 100
	stats.MaxDrawdown = math.Round(stats.MaxDrawdown*100) / 100
	stats.SharpeRatio = math.Round(stats.SharpeRatio*100) / 100
	stats.WinRate = math.Round(stats.WinRate*100) / 100
	stats.ProfitFactor = math.Round(stats.ProfitFactor*100) / 100
	stats.Volatility = math.Round(stats.Volatility*100) / 100
	stats.VaR95 = math.Round(stats.VaR95*100) / 100
	stats.ExpectedShortfall = math.Round(stats.ExpectedShortfall*100) / 100
	stats.Accuracy = math.Round(stats.Accuracy*100) / 100
	stats.AvgReturn = math.Round(stats.AvgReturn*100) / 100
	stats.BestMonthlyReturn = math.Round(stats.BestMonthlyReturn*100) / 100
	stats.WorstMonthlyReturn = math.Round(stats.WorstMonthlyReturn*100) / 100

	return stats
}

// generateRealisticSimulatedStats 基于技术指标生成更真实的模拟数据
func (s *Server) generateRealisticSimulatedStats(symbol, period string, days int) *PerformanceStats {
	// 获取技术指标数据
	multiIndicators, err := s.GetMultiTimeframeIndicators(context.Background(), symbol, "spot")
	if err != nil {
		log.Printf("[WARN] 获取技术指标失败，使用基础模拟数据: %v", err)
		// 返回基础默认值
		return &PerformanceStats{
			OverallScore: 5.0, TechnicalScore: 5.0, FundamentalScore: 5.0, SentimentScore: 5.0, MomentumScore: 5.0,
			ReturnFactor: 5.0, RiskFactor: 3.0, ConsistencyFactor: 5.0, TimingFactor: 5.0,
			TotalReturn: 0.0, AnnualizedReturn: 0.0, MaxDrawdown: 5.0, SharpeRatio: 1.0,
			WinRate: 50.0, ProfitFactor: 1.0, Volatility: 10.0, VaR95: 8.0, ExpectedShortfall: 10.0,
			Accuracy: 50.0, AvgReturn: 0.0, AvgHoldingTime: "3.0天", TotalRecommendations: 25,
			BestMonthlyReturn: 5.0, WorstMonthlyReturn: -5.0,
		}
	}

	// 基于技术指标计算各种评分
	technicalScore := s.calculateTechnicalScore(multiIndicators)
	fundamentalScore := s.calculateFundamentalScore(symbol)
	sentimentScore := 0.7 // 默认情绪得分
	momentumScore := s.calculateMomentumScore(multiIndicators)

	// 根据symbol调整基础数据
	baseMultiplier := 1.0
	switch symbol {
	case "BTC":
		baseMultiplier = 1.2
	case "ETH":
		baseMultiplier = 1.1
	case "BNB":
		baseMultiplier = 0.9
	case "ADA":
		baseMultiplier = 0.8
	default:
		baseMultiplier = 1.0
	}

	// 根据时间周期调整
	periodMultiplier := 1.0
	switch period {
	case "7d":
		periodMultiplier = 0.7
	case "30d":
		periodMultiplier = 1.0
	case "90d":
		periodMultiplier = 1.3
	case "1y":
		periodMultiplier = 1.5
	}

	// 生成基于技术指标的真实感数据
	stats := &PerformanceStats{}

	// 评分系统 - 基于技术指标
	stats.TechnicalScore = math.Round(technicalScore*10*100) / 100
	stats.FundamentalScore = math.Round(fundamentalScore*10*100) / 100
	stats.SentimentScore = math.Round(sentimentScore*10*100) / 100
	stats.MomentumScore = math.Round(momentumScore*10*100) / 100

	// 综合评分
	stats.OverallScore = math.Round((stats.TechnicalScore*0.4+stats.FundamentalScore*0.3+
		stats.SentimentScore*0.2+stats.MomentumScore*0.1)*100) / 100

	// 收益因子 - 基于动量和技术指标
	stats.ReturnFactor = math.Round((technicalScore*0.6+momentumScore*0.4)*10*10) / 10

	// 风险因子 - 技术指标的反向
	stats.RiskFactor = math.Round((1.0-technicalScore)*6*10) / 10

	// 一致性因子 - 基于技术指标稳定性
	stats.ConsistencyFactor = math.Round(technicalScore*8*10) / 10

	// 时机因子 - 基于动量
	stats.TimingFactor = math.Round(momentumScore*8*10) / 10

	// 传统性能指标
	baseReturn := (technicalScore*0.4 + momentumScore*0.3 + fundamentalScore*0.3) * baseMultiplier * periodMultiplier
	stats.TotalReturn = math.Round(baseReturn*15*100) / 100
	stats.AnnualizedReturn = math.Round((baseReturn*12+2)*100) / 100
	stats.MaxDrawdown = math.Round((1.0-technicalScore)*8*100) / 100
	riskFactor := 1.0 - technicalScore
	stats.SharpeRatio = math.Round((technicalScore/riskFactor*1.5+0.5)*100) / 100
	stats.WinRate = math.Round((technicalScore*30+50)*100) / 100
	stats.ProfitFactor = math.Round((technicalScore*1.5+1.0)*100) / 100

	// 风险指标
	stats.Volatility = math.Round((1.0-technicalScore)*15+5*100) / 100
	stats.VaR95 = math.Round(stats.Volatility*0.8*100) / 100
	stats.ExpectedShortfall = math.Round(stats.Volatility*1.2*100) / 100

	// 额外的统计数据
	stats.Accuracy = stats.WinRate
	stats.AvgReturn = math.Round(baseReturn*8*100) / 100
	stats.AvgHoldingTime = fmt.Sprintf("%.1f天", (technicalScore*5 + 2))
	stats.TotalRecommendations = int(math.Round((technicalScore*50 + 10) * baseMultiplier))
	stats.BestMonthlyReturn = math.Round((baseReturn*12+5)*100) / 100
	stats.WorstMonthlyReturn = math.Round((baseReturn*(-8)-3)*100) / 100

	return stats
}

// GetSentimentAnalysis 获取情绪分析数据
// GET /api/v1/sentiment/:symbol
func (s *Server) GetSentimentAnalysis(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	// 调用情绪分析
	result, err := s.getSentimentAnalysis(c.Request.Context(), symbol)
	if err != nil {
		log.Printf("[ERROR] 获取情绪分析失败 %s: %v", symbol, err)
		c.JSON(500, gin.H{"error": "获取情绪分析失败"})
		return
	}

	c.JSON(200, gin.H{
		"symbol":    symbol,
		"sentiment": result,
		"timestamp": time.Now().Unix(),
	})
}

// GetAvailableSymbols 获取可用的交易对列表
// GET /api/v1/market/symbols?kind=spot&limit=50
func (s *Server) GetAvailableSymbols(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50 // 默认50个
	}

	// 获取币种列表
	symbols, err := s.getAvailableSymbols(c.Request.Context(), kind, limit)
	if err != nil {
		log.Printf("[ERROR] 获取可用币种列表失败: %v", err)
		c.JSON(500, gin.H{"error": "获取币种列表失败"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    symbols,
		"count":   len(symbols),
		"kind":    kind,
	})
}

// GET /api/v1/market/symbols-with-marketcap?kind=spot&limit=50
func (s *Server) GetSymbolsWithMarketCap(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	limitStr := c.DefaultQuery("limit", "50")
	pageStr := c.DefaultQuery("page", "1")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50 // 默认50个
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1 // 默认第1页
	}

	// 获取包含市值信息的币种列表（支持分页）
	symbolsData, totalCount, err := s.getSymbolsWithMarketCapPaged(c.Request.Context(), kind, limit, page)
	if err != nil {
		log.Printf("[ERROR] 获取带市值信息的币种列表失败: %v", err)
		c.JSON(500, gin.H{"error": "获取币种列表失败"})
		return
	}

	c.JSON(200, gin.H{
		"symbols":    symbolsData,
		"count":      len(symbolsData),
		"total":      totalCount,
		"page":       page,
		"limit":      limit,
		"totalPages": (totalCount + limit - 1) / limit, // 向上取整计算总页数
		"kind":       kind,
	})
}

// AnalyzeGridStrategy 分析网格策略性能
func (s *Server) AnalyzeGridStrategy(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "币种符号不能为空"})
		return
	}

	// 获取查询参数
	upperPrice := parseFloatParam(c.Query("upper_price"), 0)
	lowerPrice := parseFloatParam(c.Query("lower_price"), 0)
	levels := parseIntParam(c.Query("levels"), 10)
	profitPercent := parseFloatParam(c.Query("profit_percent"), 1.0)
	investmentAmount := parseFloatParam(c.Query("investment_amount"), 1000.0)

	if upperPrice <= lowerPrice || levels <= 0 {
		c.JSON(400, gin.H{"error": "无效的网格参数"})
		return
	}

	ctx := c.Request.Context()

	// 获取历史价格数据（过去90天）
	historicalPrices, err := s.getHistoricalPricesForSymbol(ctx, symbol, 90)
	if err != nil || len(historicalPrices) < 30 {
		log.Printf("[WARN] 获取%s历史价格失败: %v，使用有限数据进行分析", symbol, err)
		if len(historicalPrices) < 10 {
			c.JSON(500, gin.H{"error": "历史数据不足，无法进行策略分析"})
			return
		}
	}

	// 执行网格策略回测
	backtestResult := s.performGridBacktest(historicalPrices, upperPrice, lowerPrice, levels, profitPercent, investmentAmount)

	// 计算性能指标
	performanceMetrics := s.calculateGridPerformanceMetrics(backtestResult, historicalPrices)

	// 生成优化建议
	optimizationSuggestions := s.generateGridOptimizationSuggestions(performanceMetrics, upperPrice, lowerPrice, levels, profitPercent)

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"symbol":                   symbol,
			"backtest_result":          backtestResult,
			"performance_metrics":      performanceMetrics,
			"optimization_suggestions": optimizationSuggestions,
			"analysis_timestamp":       time.Now().Unix(),
		},
	})
}

// parseFloatParam 解析浮点数参数
func parseFloatParam(param string, defaultValue float64) float64 {
	if param == "" {
		return defaultValue
	}
	if value, err := strconv.ParseFloat(param, 64); err == nil {
		return value
	}
	return defaultValue
}

// parseIntParam 解析整数参数
func parseIntParam(param string, defaultValue int) int {
	if param == "" {
		return defaultValue
	}
	if value, err := strconv.Atoi(param); err == nil {
		return value
	}
	return defaultValue
}

// GridBacktestResult 网格回测结果
type GridBacktestResult struct {
	TotalTrades       int         `json:"total_trades"`
	SuccessfulTrades  int         `json:"successful_trades"`
	FailedTrades      int         `json:"failed_trades"`
	TotalProfit       float64     `json:"total_profit"`
	MaxDrawdown       float64     `json:"max_drawdown"`
	SharpeRatio       float64     `json:"sharpe_ratio"`
	WinRate           float64     `json:"win_rate"`
	AvgProfitPerTrade float64     `json:"avg_profit_per_trade"`
	Trades            []GridTrade `json:"trades"`
}

// GridTrade 网格交易记录
type GridTrade struct {
	Type      string  `json:"type"` // "buy" or "sell"
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	GridLevel int     `json:"grid_level"`
	Profit    float64 `json:"profit"`
	Timestamp int64   `json:"timestamp"`
}

// performGridBacktest 执行网格策略回测
func (s *Server) performGridBacktest(historicalPrices []float64, upperPrice, lowerPrice float64, levels int, profitPercent, investmentAmount float64) GridBacktestResult {
	if len(historicalPrices) < 2 {
		return GridBacktestResult{}
	}

	gridSpacing := (upperPrice - lowerPrice) / float64(levels)
	gridAmount := investmentAmount / float64(levels)

	var trades []GridTrade
	totalProfit := 0.0
	maxDrawdown := 0.0
	peakProfit := 0.0

	// 网格持仓跟踪
	gridPositions := make(map[int]float64) // gridLevel -> quantity

	for i, price := range historicalPrices {
		// 检查价格是否在网格范围内
		if price < lowerPrice || price > upperPrice {
			continue
		}

		// 计算当前网格级别
		gridLevel := int(math.Floor((price - lowerPrice) / gridSpacing))
		if gridLevel >= levels {
			gridLevel = levels - 1
		}
		if gridLevel < 0 {
			gridLevel = 0
		}

		// 模拟网格交易逻辑
		if gridLevel < levels/2 {
			// 买入区域
			if _, exists := gridPositions[gridLevel]; !exists {
				quantity := gridAmount / price
				gridPositions[gridLevel] = quantity

				trades = append(trades, GridTrade{
					Type:      "buy",
					Price:     price,
					Quantity:  quantity,
					GridLevel: gridLevel,
					Timestamp: int64(i),
				})
			}
		} else {
			// 卖出区域
			if quantity, exists := gridPositions[gridLevel-levels/2]; exists {
				// 计算利润
				buyPrice := lowerPrice + float64(gridLevel-levels/2)*gridSpacing
				profit := (price - buyPrice) * quantity * (profitPercent / 100)
				totalProfit += profit

				trades = append(trades, GridTrade{
					Type:      "sell",
					Price:     price,
					Quantity:  quantity,
					GridLevel: gridLevel,
					Profit:    profit,
					Timestamp: int64(i),
				})

				delete(gridPositions, gridLevel-levels/2)
			}
		}

		// 更新最大回撤
		if totalProfit > peakProfit {
			peakProfit = totalProfit
		}
		currentDrawdown := peakProfit - totalProfit
		if currentDrawdown > maxDrawdown {
			maxDrawdown = currentDrawdown
		}
	}

	// 计算胜率和夏普比率
	successfulTrades := 0
	totalTrades := len(trades)
	for _, trade := range trades {
		if trade.Type == "sell" && trade.Profit > 0 {
			successfulTrades++
		}
	}

	winRate := 0.0
	avgProfitPerTrade := 0.0
	if totalTrades > 0 {
		winRate = float64(successfulTrades) / float64(totalTrades) * 100
		avgProfitPerTrade = totalProfit / float64(totalTrades)
	}

	// 计算夏普比率（简化版）
	sharpeRatio := 0.0
	if len(trades) > 1 {
		profits := make([]float64, 0, len(trades))
		for _, trade := range trades {
			if trade.Type == "sell" {
				profits = append(profits, trade.Profit)
			}
		}

		if len(profits) > 1 {
			avgProfit := 0.0
			for _, p := range profits {
				avgProfit += p
			}
			avgProfit /= float64(len(profits))

			variance := 0.0
			for _, p := range profits {
				variance += math.Pow(p-avgProfit, 2)
			}
			stdDev := math.Sqrt(variance / float64(len(profits)))

			if stdDev > 0 {
				sharpeRatio = avgProfit / stdDev * math.Sqrt(252) // 年化
			}
		}
	}

	return GridBacktestResult{
		TotalTrades:       totalTrades,
		SuccessfulTrades:  successfulTrades,
		FailedTrades:      totalTrades - successfulTrades,
		TotalProfit:       totalProfit,
		MaxDrawdown:       maxDrawdown,
		SharpeRatio:       sharpeRatio,
		WinRate:           winRate,
		AvgProfitPerTrade: avgProfitPerTrade,
		Trades:            trades,
	}
}

// calculateGridPerformanceMetrics 计算网格性能指标
func (s *Server) calculateGridPerformanceMetrics(result GridBacktestResult, historicalPrices []float64) map[string]interface{} {
	metrics := make(map[string]interface{})

	// 基础指标
	metrics["total_return"] = result.TotalProfit
	metrics["total_return_percent"] = result.TotalProfit / 1000 * 100 // 基于1000USDT投资
	metrics["max_drawdown"] = result.MaxDrawdown
	metrics["max_drawdown_percent"] = result.MaxDrawdown / 1000 * 100
	metrics["win_rate"] = result.WinRate
	metrics["total_trades"] = result.TotalTrades
	metrics["sharpe_ratio"] = result.SharpeRatio

	// 风险调整指标
	if result.MaxDrawdown > 0 {
		metrics["return_to_drawdown"] = result.TotalProfit / result.MaxDrawdown
	} else {
		metrics["return_to_drawdown"] = 0
	}

	// 月化收益率（简化计算）
	days := len(historicalPrices)
	if days > 0 {
		dailyReturn := result.TotalProfit / 1000 / float64(days)
		metrics["annualized_return"] = dailyReturn * 365 * 100
	}

	// 策略评估
	if result.SharpeRatio > 1.5 {
		metrics["performance_rating"] = "优秀"
	} else if result.SharpeRatio > 1.0 {
		metrics["performance_rating"] = "良好"
	} else if result.SharpeRatio > 0.5 {
		metrics["performance_rating"] = "一般"
	} else {
		metrics["performance_rating"] = "较差"
	}

	return metrics
}

// generateGridOptimizationSuggestions 生成网格优化建议
func (s *Server) generateGridOptimizationSuggestions(metrics map[string]interface{}, upperPrice, lowerPrice float64, levels int, profitPercent float64) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	// 基于夏普比率的建议
	if sharpe, ok := metrics["sharpe_ratio"].(float64); ok {
		if sharpe < 0.5 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "risk_adjustment",
				"priority":    "high",
				"title":       "风险调整不足",
				"description": "夏普比率较低，建议增加利润率或减少网格层数",
				"action":      "increase_profit_or_reduce_levels",
			})
		}
	}

	// 基于胜率的建议
	if winRate, ok := metrics["win_rate"].(float64); ok {
		if winRate < 50 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "strategy_adjustment",
				"priority":    "medium",
				"title":       "胜率偏低",
				"description": fmt.Sprintf("当前胜率%.1f%%，建议调整利润百分比或网格范围", winRate),
				"action":      "adjust_profit_percent",
			})
		}
	}

	// 基于最大回撤的建议
	if maxDD, ok := metrics["max_drawdown_percent"].(float64); ok {
		if maxDD > 20 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "risk_management",
				"priority":    "high",
				"title":       "回撤风险较高",
				"description": fmt.Sprintf("最大回撤%.1f%%过高，建议启用止损或调整网格参数", maxDD),
				"action":      "enable_stop_loss",
			})
		}
	}

	// 基于交易频率的建议
	if totalTrades, ok := metrics["total_trades"].(float64); ok && totalTrades < 10 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "parameter_optimization",
			"priority":    "medium",
			"title":       "交易频率过低",
			"description": "历史回测中交易次数较少，建议扩大网格范围或增加层数",
			"action":      "expand_grid_range",
		})
	}

	return suggestions
}

// GetGridTradingSymbols 获取适合网格交易的币种列表
func (s *Server) GetGridTradingSymbols(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	_ = kind // 暂时保留参数以保持API一致性
	limitStr := c.DefaultQuery("limit", "50")
	pageStr := c.DefaultQuery("page", "1")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	// 创建一个临时的网格交易策略配置来使用扫描器
	// 对于扫描模式，我们需要提供合理的默认网格参数来评估币种适应性
	tempStrategy := &pdb.TradingStrategy{
		Conditions: pdb.StrategyConditions{
			GridTradingEnabled:   true,
			GridUpperPrice:       1000.0, // 临时默认上限价格
			GridLowerPrice:       10.0,   // 临时默认下限价格
			GridLevels:           10,     // 临时默认网格层数
			GridProfitPercent:    1.0,    // 临时默认利润百分比
			GridInvestmentAmount: 1000.0, // 临时默认投资金额
			GridStopLossEnabled:  true,   // 启用止损
			GridStopLossPercent:  10.0,   // 10%止损
			DynamicPositioning:   true,   // 启用动态仓位
			MaxPositionSize:      50.0,   // 最大50%持仓
		},
	}

	// 从策略注册表获取网格交易扫描器
	scanner := s.scannerRegistry.SelectScanner(tempStrategy)
	if scanner == nil {
		log.Printf("[GetGridTradingSymbols] 未找到网格交易策略扫描器")
		c.JSON(500, gin.H{"error": "网格交易策略不可用"})
		return
	}

	// 执行扫描
	rawResults, err := scanner.Scan(c.Request.Context(), tempStrategy)
	if err != nil {
		log.Printf("[ERROR] 网格交易币种扫描失败: %v", err)
		c.JSON(500, gin.H{"error": "网格交易币种筛选失败"})
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

	// 分页处理
	totalSymbols := len(eligibleSymbols)
	start := offset
	end := offset + limit

	if start >= totalSymbols {
		c.JSON(200, gin.H{
			"success": true,
			"symbols": []gin.H{},
			"total":   totalSymbols,
			"page":    page,
			"limit":   limit,
		})
		return
	}

	if end > totalSymbols {
		end = totalSymbols
	}

	pagedSymbols := eligibleSymbols[start:end]

	// 转换为前端需要的格式，添加网格交易排序指标
	var symbolsData []gin.H
	for _, symbol := range pagedSymbols {
		// 从Reason字段解析评分信息
		// 网格交易Reason格式: "适合网格交易(评分:X.XX)-波动率:X.XX,流动性:X.XX,稳定性:X.XX"
		reason := symbol.Reason
		volatilityScore := 0.0
		liquidityScore := 0.0
		stabilityScore := 0.0
		overallScore := 0.0

		// 解析Reason字符串
		if strings.Contains(reason, "适合网格交易") {
			// 提取综合评分 - 使用更简单的方法
			if strings.Contains(reason, "(评分:") && strings.Contains(reason, ")") {
				// 找到"(评分:"和")"之间的内容
				start := strings.Index(reason, "(评分:") + len("(评分:")
				end := strings.Index(reason[start:], ")") + start
				if start < end && end <= len(reason) {
					scoreStr := reason[start:end]
					if val, err := strconv.ParseFloat(scoreStr, 64); err == nil {
						overallScore = val
					}
				}
			}

			// 提取各项评分
			if dashIndex := strings.Index(reason, "-"); dashIndex != -1 {
				scorePart := reason[dashIndex+1:] // "波动率:X.XX,流动性:X.XX,稳定性:X.XX"
				parts := strings.Split(scorePart, ",")

				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.Contains(part, "波动率:") {
						if val, err := strconv.ParseFloat(strings.TrimPrefix(part, "波动率:"), 64); err == nil {
							volatilityScore = val
						}
					} else if strings.Contains(part, "流动性:") {
						if val, err := strconv.ParseFloat(strings.TrimPrefix(part, "流动性:"), 64); err == nil {
							liquidityScore = val
						}
					} else if strings.Contains(part, "稳定性:") {
						if val, err := strconv.ParseFloat(strings.TrimPrefix(part, "稳定性:"), 64); err == nil {
							stabilityScore = val
						}
					}
				}
			}

			// 如果解析失败，提供默认评分
			if overallScore == 0 {
				overallScore = 0.8
			}
			if volatilityScore == 0 {
				volatilityScore = 0.7
			}
			if liquidityScore == 0 {
				liquidityScore = 0.8
			}
			if stabilityScore == 0 {
				stabilityScore = 0.7
			}
		} else {
			// 非网格交易币种，使用默认评分
			overallScore = 0.5
			volatilityScore = 0.5
			liquidityScore = 0.5
			stabilityScore = 0.5
		}

		symbolsData = append(symbolsData, gin.H{
			"symbol":               symbol.Symbol,
			"current_price":        0, // 网格扫描器不关注价格
			"price_change_percent": 0,
			"volume_24h":           0,
			"market_cap_usd":       symbol.MarketCap,
			// 网格交易专用排序指标
			"grid_volatility_score": volatilityScore, // 波动率评分
			"grid_liquidity_score":  liquidityScore,  // 流动性评分
			"grid_stability_score":  stabilityScore,  // 稳定性评分
			"grid_overall_score":    overallScore,    // 综合评分
			"grid_score_reason":     reason,          // 详细评分说明
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"symbols": symbolsData,
		"total":   totalSymbols,
		"page":    page,
		"limit":   limit,
	})
}

// AnalyzeSymbolForGridTrading 分析币种用于网格交易参数优化
func (s *Server) AnalyzeSymbolForGridTrading(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "币种符号不能为空"})
		return
	}

	ctx := c.Request.Context()

	// 获取当前价格
	currentPrice, err := s.getCurrentPrice(ctx, symbol, "spot")
	if err != nil {
		log.Printf("[ERROR] 获取%s当前价格失败: %v", symbol, err)
		c.JSON(500, gin.H{"error": "获取当前价格失败"})
		return
	}

	// 获取历史价格数据（过去30天）
	historicalPrices, err := s.getHistoricalPricesForSymbol(ctx, symbol, 30)
	if err != nil {
		log.Printf("[WARN] 获取%s历史价格失败: %v，使用当前价格", symbol, err)
		historicalPrices = []float64{currentPrice}
	}

	// 计算波动率
	volatility := s.calculatePriceVolatility(historicalPrices)

	// 基于波动率推荐网格参数
	recommendedLevels := s.calculateRecommendedGridLevels(volatility)
	recommendedUpper, recommendedLower := s.calculateRecommendedPriceRange(currentPrice, volatility, historicalPrices)

	// 返回分析结果
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"symbol":             symbol,
			"current_price":      currentPrice,
			"volatility":         volatility,
			"recommended_levels": recommendedLevels,
			"recommended_upper":  recommendedUpper,
			"recommended_lower":  recommendedLower,
			"historical_prices":  historicalPrices,
			"analysis_timestamp": time.Now().Unix(),
		},
	})
}

// getMarketDepth 获取市场深度数据
func (s *Server) getMarketDepth(ctx context.Context, symbol string, limit int) (*MarketDepth, error) {
	log.Printf("[MarketDepth] 开始获取市场深度: %s limit=%d", symbol, limit)

	// 首先尝试从数据库获取最新的深度数据
	log.Printf("[MarketDepth] 尝试从数据库获取: %s", symbol)
	depth, err := s.getMarketDepthFromDB(symbol, "spot")
	log.Printf("[MarketDepth] 数据库查询结果: err=%v, depth=%v", err, depth != nil)

	if err == nil && depth != nil && len(depth.Bids) > 0 && len(depth.Asks) > 0 {
		// 如果数据库中有数据且相对较新（5分钟内），直接返回
		log.Printf("[MarketDepth] 从数据库返回有效数据: bids=%d, asks=%d", len(depth.Bids), len(depth.Asks))
		return depth, nil
	}

	// 如果数据库没有数据或数据过旧，从币安API获取
	log.Printf("[MarketDepth] 从API获取数据: %s", symbol)
	return s.getMarketDepthFromBinance(ctx, symbol, "spot", limit)
}

// getMarketDepthFromDB 从数据库获取市场深度数据
func (s *Server) getMarketDepthFromDB(symbol, kind string) (*MarketDepth, error) {
	log.Printf("[MarketDepth] 开始从数据库获取深度数据: %s %s", symbol, kind)

	query := `
		SELECT bids, asks, created_at
		FROM binance_order_book_depth
		WHERE symbol = ? AND market_type = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	var bidsStr, asksStr string
	var updatedAt time.Time
	log.Printf("[MarketDepth] 执行查询: %s", query)

	// 先检查表是否存在以及是否有数据
	var count int64
	countQuery := "SELECT COUNT(*) FROM binance_order_book_depth WHERE symbol = ? AND market_type = ?"
	if countErr := s.db.DB().Raw(countQuery, symbol, kind).Scan(&count).Error; countErr != nil {
		log.Printf("[MarketDepth] 检查数据计数失败: %v", countErr)
		return nil, countErr
	}

	if count == 0 {
		log.Printf("[MarketDepth] 数据库中没有%s的深度数据", symbol)
		return nil, fmt.Errorf("no depth data found for %s", symbol)
	}

	log.Printf("[MarketDepth] 找到%d条记录，开始查询最新数据", count)

	// 使用Take方法而不是Row().Scan()
	type DepthRecord struct {
		Bids      string
		Asks      string
		CreatedAt time.Time
	}

	var record DepthRecord
	err := s.db.DB().Raw(query, symbol, kind).Take(&record).Error
	if err != nil {
		log.Printf("[MarketDepth] 查询记录失败: %v", err)
		return nil, err
	}

	bidsStr = record.Bids
	asksStr = record.Asks
	updatedAt = record.CreatedAt
	log.Printf("[MarketDepth] 从数据库获取到市场深度数据")

	// 检查数据是否过旧（5分钟）
	if time.Since(updatedAt) > 5*time.Minute {
		return nil, fmt.Errorf("depth data too old")
	}

	// 解析JSON数据
	var bids, asks [][]float64
	if err := json.Unmarshal([]byte(bidsStr), &bids); err != nil {
		return nil, fmt.Errorf("failed to parse bids: %v", err)
	}
	if err := json.Unmarshal([]byte(asksStr), &asks); err != nil {
		return nil, fmt.Errorf("failed to parse asks: %v", err)
	}

	return &MarketDepth{
		Bids: bids,
		Asks: asks,
	}, nil
}

// getMarketDepthFromBinance 从币安API获取市场深度数据
func (s *Server) getMarketDepthFromBinance(ctx context.Context, symbol, kind string, limit int) (*MarketDepth, error) {
	// 限制limit在1-100之间
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	// 构建API URL
	var url string
	if kind == "spot" {
		url = fmt.Sprintf("https://api.binance.com/api/v3/depth?symbol=%s&limit=%d", symbol, limit)
	} else {
		// 期货
		url = fmt.Sprintf("https://fapi.binance.com/fapi/v1/depth?symbol=%s&limit=%d", symbol, limit)
	}

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch depth data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析响应
	var depthData struct {
		LastUpdateID int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"` // 币安API返回字符串数组
		Asks         [][]string `json:"asks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&depthData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 转换字符串为float64
	bids := make([][]float64, 0, len(depthData.Bids))
	for _, bid := range depthData.Bids {
		if len(bid) >= 2 {
			price, err1 := strconv.ParseFloat(bid[0], 64)
			quantity, err2 := strconv.ParseFloat(bid[1], 64)
			if err1 == nil && err2 == nil {
				bids = append(bids, []float64{price, quantity})
			}
		}
	}

	asks := make([][]float64, 0, len(depthData.Asks))
	for _, ask := range depthData.Asks {
		if len(ask) >= 2 {
			price, err1 := strconv.ParseFloat(ask[0], 64)
			quantity, err2 := strconv.ParseFloat(ask[1], 64)
			if err1 == nil && err2 == nil {
				asks = append(asks, []float64{price, quantity})
			}
		}
	}

	if len(bids) == 0 || len(asks) == 0 {
		return nil, fmt.Errorf("no valid depth data received")
	}

	result := &MarketDepth{
		Bids: bids,
		Asks: asks,
	}

	// 异步保存到数据库（不阻塞主流程）
	go func() {
		if err := s.saveMarketDepthToDB(symbol, kind, result); err != nil {
			log.Printf("[MarketDepth] Failed to save depth data to DB: %v", err)
		}
	}()

	return result, nil
}

// saveMarketDepthToDB 保存市场深度数据到数据库
func (s *Server) saveMarketDepthToDB(symbol, kind string, depth *MarketDepth) error {
	bidsJSON, err := json.Marshal(depth.Bids)
	if err != nil {
		return fmt.Errorf("failed to marshal bids: %v", err)
	}

	asksJSON, err := json.Marshal(depth.Asks)
	if err != nil {
		return fmt.Errorf("failed to marshal asks: %v", err)
	}

	query := `
		INSERT INTO binance_order_book_depth (symbol, market_type, bids, asks, last_update_id, snapshot_time)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			bids = VALUES(bids),
			asks = VALUES(asks),
			last_update_id = VALUES(last_update_id),
			snapshot_time = VALUES(snapshot_time)
	`

	return s.db.DB().Exec(query, symbol, kind, string(bidsJSON), string(asksJSON), time.Now().Unix(), time.Now().Unix()*1000).Error
}

// getSymbolsWithMarketCap 获取包含市值信息的币种列表（优化为一次查询）
func (s *Server) getSymbolsWithMarketCap(ctx context.Context, kind string, limit int) ([]gin.H, error) {
	// 首先获取币安可用的交易对列表
	availableSymbols, err := s.getAvailableSymbols(ctx, kind, 10000) // 获取足够多的可用币种
	if err != nil {
		log.Printf("[WARN] 获取币安可用交易对失败: %v，将返回空列表", err)
		return []gin.H{}, nil
	}

	if len(availableSymbols) == 0 {
		log.Printf("[INFO] 没有找到币安可用的交易对数据")
		return []gin.H{}, nil
	}

	// 将带后缀的币安交易对转换为不带后缀的币种符号（用于匹配CoinCap数据）
	var coinCapSymbols []string
	for _, symbol := range availableSymbols {
		// 去掉常见的交易对后缀
		coinCapSymbol := s.normalizeBinanceSymbolToCoinCap(symbol)
		if coinCapSymbol != "" {
			coinCapSymbols = append(coinCapSymbols, coinCapSymbol)
		}
	}

	if len(coinCapSymbols) == 0 {
		log.Printf("[INFO] 转换后没有有效的CoinCap币种符号")
		return []gin.H{}, nil
	}

	// 创建市值数据服务
	marketDataService := pdb.NewCoinCapMarketDataService(s.db.DB())

	// 一次性获取市值小于5000万且币安支持的完整数据
	dataList, err := marketDataService.GetMarketDataByMarketCapRangeAndSymbols(ctx, 0, 50000000, coinCapSymbols, limit*2) // 获取更多数据用于筛选
	if err != nil {
		log.Printf("[WARN] 查询市值范围内且币安支持的币种数据失败: %v", err)
		return []gin.H{}, nil
	}

	// 如果没有找到符合条件的币种，返回空列表
	if len(dataList) == 0 {
		log.Printf("[INFO] 没有找到市值<5000万的币种，CoinCap数据可能还未同步，请运行: go run cmd/coincap_sync/main.go -action=market-data")
		return []gin.H{}, nil
	}

	// 预验证币种实时数据可用性，只返回能获取到实时数据的币种
	var validatedSymbolsData []gin.H
	validationTimeout := 5 * time.Second
	validationCtx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()

	log.Printf("[INFO] 开始验证 %d 个币种的实时数据可用性...", len(dataList))

	for _, data := range dataList {
		// 构造币安交易对格式用于验证
		binanceSymbol := data.Symbol + "USDT"
		if kind == "futures" {
			binanceSymbol = data.Symbol + "USDT" // 合约也使用USDT格式验证
		}

		// 验证是否能获取到实时数据
		_, success := s.getRealtimeDataConcurrently(validationCtx, binanceSymbol, kind)
		if success {
			// 解析字符串为float64以便前端使用
			price, err := strconv.ParseFloat(data.PriceUSD, 64)
			if err != nil {
				log.Printf("[WARN] 解析价格失败 %s: %s", data.Symbol, data.PriceUSD)
				price = 0
			}

			changePercent, err := strconv.ParseFloat(data.Change24Hr, 64)
			if err != nil {
				log.Printf("[WARN] 解析涨跌幅失败 %s: %s", data.Symbol, data.Change24Hr)
				changePercent = 0
			}

			volume, err := strconv.ParseFloat(data.Volume24Hr, 64)
			if err != nil {
				log.Printf("[WARN] 解析成交量失败 %s: %s", data.Symbol, data.Volume24Hr)
				volume = 0
			}

			marketCap, err := strconv.ParseFloat(data.MarketCapUSD, 64)
			if err != nil {
				log.Printf("[WARN] 解析市值失败 %s: %s (原始数据: %s)", data.Symbol, data.MarketCapUSD, data.MarketCapUSD)
				marketCap = 0
			} else if data.MarketCapUSD == "" {
				log.Printf("[WARN] 市值为空字符串 %s", data.Symbol)
				marketCap = 0
			}

			symbolData := gin.H{
				"symbol":               data.Symbol,
				"current_price":        price,
				"price_change_percent": changePercent,
				"volume_24h":           volume,
				"market_cap_usd":       marketCap,
				"last_updated":         data.UpdatedAt.Unix(),
			}
			validatedSymbolsData = append(validatedSymbolsData, symbolData)

			// 达到限制数量时停止
			if len(validatedSymbolsData) >= limit {
				break
			}
		} else {
			log.Printf("[INFO] 币种 %s 无法获取实时数据，已过滤", data.Symbol)
		}
	}

	if len(validatedSymbolsData) == 0 {
		log.Printf("[INFO] 验证后没有有效的实时数据币种")
		return []gin.H{}, nil
	}

	log.Printf("[INFO] 验证完成，返回 %d 个有实时数据的市值<5000万的币种", len(validatedSymbolsData))
	return validatedSymbolsData, nil
}

// getSymbolsWithMarketCapPaged 获取包含市值信息的币种列表（支持分页）
func (s *Server) getSymbolsWithMarketCapPaged(ctx context.Context, kind string, limit int, page int) ([]gin.H, int, error) {
	// 首先获取币安可用的交易对列表
	availableSymbols, err := s.getAvailableSymbols(ctx, kind, 10000) // 获取足够多的可用币种
	if err != nil {
		log.Printf("[WARN] 获取币安可用交易对失败: %v，将返回空列表", err)
		return []gin.H{}, 0, nil
	}

	if len(availableSymbols) == 0 {
		log.Printf("[INFO] 没有找到币安可用的交易对数据")
		return []gin.H{}, 0, nil
	}

	// 将带后缀的币安交易对转换为不带后缀的币种符号（用于匹配CoinCap数据）
	var coinCapSymbols []string
	for _, symbol := range availableSymbols {
		// 去掉常见的交易对后缀
		coinCapSymbol := s.normalizeBinanceSymbolToCoinCap(symbol)
		if coinCapSymbol != "" {
			coinCapSymbols = append(coinCapSymbols, coinCapSymbol)
		}
	}

	if len(coinCapSymbols) == 0 {
		log.Printf("[INFO] 转换后没有有效的CoinCap币种符号")
		return []gin.H{}, 0, nil
	}

	// 创建市值数据服务
	marketDataService := pdb.NewCoinCapMarketDataService(s.db.DB())

	// 计算偏移量
	offset := (page - 1) * limit

	// 获取总数（只统计币安支持的币种）
	totalCountInt64, err := marketDataService.GetMarketDataCountByMarketCapRangeAndSymbols(ctx, 0, 50000000, coinCapSymbols)
	if err != nil {
		log.Printf("[WARN] 查询市值范围内且币安支持的币种总数失败: %v", err)
		totalCountInt64 = 0
	}
	totalCount := int(totalCountInt64)

	// 获取分页数据（只获取币安支持的币种）
	dataList, err := marketDataService.GetMarketDataByMarketCapRangeAndSymbolsPaged(ctx, 0, 50000000, coinCapSymbols, limit, offset)
	if err != nil {
		log.Printf("[WARN] 查询市值范围内且币安支持的分页币种数据失败: %v", err)
		return []gin.H{}, totalCount, nil
	}

	// 如果没有找到符合条件的币种，返回空列表
	if len(dataList) == 0 {
		log.Printf("[INFO] 第%d页没有找到市值<5000万的币种数据", page)
		return []gin.H{}, totalCount, nil
	}

	// 转换为前端需要的格式
	var symbolsData []gin.H
	for _, data := range dataList {
		// 解析字符串为float64以便前端使用
		price, err := strconv.ParseFloat(data.PriceUSD, 64)
		if err != nil {
			log.Printf("[WARN] 解析价格失败 %s: %s", data.Symbol, data.PriceUSD)
			price = 0
		}

		changePercent, err := strconv.ParseFloat(data.Change24Hr, 64)
		if err != nil {
			log.Printf("[WARN] 解析涨跌幅失败 %s: %s", data.Symbol, data.Change24Hr)
			changePercent = 0
		}

		volume, err := strconv.ParseFloat(data.Volume24Hr, 64)
		if err != nil {
			log.Printf("[WARN] 解析成交量失败 %s: %s", data.Symbol, data.Volume24Hr)
			volume = 0
		}

		marketCap, err := strconv.ParseFloat(data.MarketCapUSD, 64)
		if err != nil {
			log.Printf("[WARN] 解析市值失败 %s: %s (原始数据: %s)", data.Symbol, data.MarketCapUSD, data.MarketCapUSD)
			marketCap = 0
		} else if data.MarketCapUSD == "" {
			log.Printf("[WARN] 市值为空字符串 %s", data.Symbol)
			marketCap = 0
		}

		symbolData := gin.H{
			"symbol":               data.Symbol,
			"current_price":        price,
			"price_change_percent": changePercent,
			"volume_24h":           volume,
			"market_cap_usd":       marketCap,
			"last_updated":         data.UpdatedAt.Unix(),
		}
		symbolsData = append(symbolsData, symbolData)
	}

	log.Printf("[INFO] 返回第%d页 %d 个市值<5000万的币种数据 (总数: %d)", page, len(symbolsData), totalCount)
	return symbolsData, totalCount, nil
}

// getAvailableSymbols 获取可用的交易对列表
func (s *Server) getAvailableSymbols(ctx context.Context, kind string, limit int) ([]string, error) {
	// 首先尝试从数据库获取数据
	var symbols []string

	// 获取GORM数据库实例
	dbInstance := s.db.DB()

	// 尝试数据库查询（从最新的快照中获取数据）
	query := `
			SELECT t.symbol
			FROM binance_market_tops t
			INNER JOIN binance_market_snapshots s ON t.snapshot_id = s.id
			WHERE s.kind = ?
			GROUP BY t.symbol
			ORDER BY
				MAX(CASE WHEN t.volume REGEXP '^[0-9]+(\\\\.?[0-9]+)?$' THEN CAST(t.volume AS DECIMAL(20,8)) ELSE 0 END) DESC,
				MAX(CAST(t.market_cap_usd AS DECIMAL(20,8))) DESC
			LIMIT ?
		`

	rows, err := dbInstance.Raw(query, kind, limit).Rows()
	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var symbol string
			if err := rows.Scan(&symbol); err != nil {
				continue
			}
			symbols = append(symbols, symbol)
		}
	} else {
		log.Printf("[INFO] 数据库查询失败，使用默认币种列表: %v", err)
	}

	// 如果数据库查询失败或没有数据，不返回默认币种列表
	if len(symbols) == 0 {
		log.Printf("[INFO] 数据库中没有可用币种数据")
	}

	return symbols, nil
}

// ===== 黑名单管理 API =====

// GET /market/binance/blacklist?kind=spot|futures - 获取黑名单
func (s *Server) ListBinanceBlacklist(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	items, err := s.db.ListBinanceBlacklist(kind)
	if err != nil {
		s.DatabaseError(c, "查询黑名单", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// POST /market/binance/blacklist - 添加黑名单
func (s *Server) AddBinanceBlacklist(c *gin.Context) {
	var body struct {
		Kind   string `json:"kind"` // spot / futures
		Symbol string `json:"symbol"`
	}
	if err := c.BindJSON(&body); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if body.Kind == "" {
		s.ValidationError(c, "kind", "类型不能为空，必须为 spot 或 futures")
		return
	}
	if body.Symbol == "" {
		s.ValidationError(c, "symbol", "币种符号不能为空")
		return
	}
	if err := s.db.AddBinanceBlacklist(body.Kind, body.Symbol); err != nil {
		s.DatabaseError(c, "添加黑名单", err)
		return
	}
	// 失效市场数据缓存和黑名单缓存，使黑名单变更立即生效
	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}
	if err := s.InvalidateBlacklistCache(c.Request.Context(), body.Kind); err != nil {
		log.Printf("[WARN] Failed to invalidate blacklist cache (kind=%s): %v", body.Kind, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /market/binance/blacklist/:kind/:symbol - 删除黑名单
func (s *Server) DeleteBinanceBlacklist(c *gin.Context) {
	kind := strings.TrimSpace(c.Param("kind"))
	symbol := strings.TrimSpace(c.Param("symbol"))
	if symbol == "" {
		s.ValidationError(c, "symbol", "币种符号不能为空")
		return
	}
	if err := s.db.DeleteBinanceBlacklist(kind, symbol); err != nil {
		s.DatabaseError(c, "删除黑名单", err)
		return
	}
	// 失效市场数据缓存和黑名单缓存，使黑名单变更立即生效
	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}
	if err := s.InvalidateBlacklistCache(c.Request.Context(), kind); err != nil {
		log.Printf("[WARN] Failed to invalidate blacklist cache (kind=%s): %v", kind, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// normalizeBinanceSymbolToCoinCap 将币安交易对符号转换为CoinCap使用的币种符号
func (s *Server) normalizeBinanceSymbolToCoinCap(binanceSymbol string) string {
	if binanceSymbol == "" {
		return ""
	}

	// 定义常见的交易对后缀，按长度降序排列以确保正确匹配
	suffixes := []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToUpper(binanceSymbol), suffix) {
			// 去掉后缀，返回基础币种符号
			baseSymbol := strings.TrimSuffix(strings.ToUpper(binanceSymbol), suffix)
			// 确保基础符号不为空
			if baseSymbol != "" {
				return baseSymbol
			}
		}
	}

	// 如果没有匹配到常见后缀，返回原符号（可能是一些特殊交易对）
	log.Printf("[WARN] 无法识别币安交易对后缀: %s", binanceSymbol)
	return binanceSymbol
}

// ============================================================================
// 网格交易分析辅助函数
// ============================================================================

// getHistoricalPricesForSymbol 获取币种的历史价格数据
func (s *Server) getHistoricalPricesForSymbol(ctx context.Context, symbol string, days int) ([]float64, error) {
	if days <= 0 {
		days = 30 // 默认30天
	}

	// 从数据库获取K线数据
	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ? AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
		AND open_time >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? DAY)
		ORDER BY open_time ASC
		LIMIT ?
	`

	rows, err := s.db.DB().Raw(query, symbol, days, days*2).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询历史价格失败: %v", err)
	}
	defer rows.Close()

	var prices []float64
	for rows.Next() {
		var price float64
		if err := rows.Scan(&price); err != nil {
			continue
		}
		prices = append(prices, price)
	}

	// 如果数据库没有足够数据，返回一些模拟数据
	if len(prices) < 10 {
		log.Printf("[WARN] %s历史价格数据不足，使用模拟数据", symbol)
		// 生成一些基于当前价格的模拟数据
		currentPrice, err := s.getCurrentPrice(ctx, symbol, "spot")
		if err != nil {
			currentPrice = 1.0 // 默认价格
		}

		prices = []float64{currentPrice * 0.95, currentPrice * 0.97, currentPrice * 0.99, currentPrice * 1.01, currentPrice * 1.03, currentPrice * 1.05}
	}

	return prices, nil
}

// calculatePriceVolatility 计算价格波动率
func (s *Server) calculatePriceVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.05 // 默认波动率
	}

	var returns []float64
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			ret := (prices[i] - prices[i-1]) / prices[i-1]
			returns = append(returns, ret)
		}
	}

	if len(returns) == 0 {
		return 0.05
	}

	// 计算标准差作为波动率
	sum := 0.0
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	sumSquares := 0.0
	for _, ret := range returns {
		sumSquares += math.Pow(ret-mean, 2)
	}

	variance := sumSquares / float64(len(returns))
	return math.Sqrt(variance)
}

// calculateRecommendedGridLevels 基于波动率计算推荐的网格层数
func (s *Server) calculateRecommendedGridLevels(volatility float64) int {
	// 波动率越高，网格层数越少（避免过度交易）
	if volatility > 0.15 {
		return 5 // 高波动：5层
	} else if volatility > 0.10 {
		return 8 // 中高波动：8层
	} else if volatility > 0.05 {
		return 12 // 中等波动：12层
	} else if volatility > 0.02 {
		return 15 // 低波动：15层
	} else {
		return 20 // 极低波动：20层
	}
}

// calculateRecommendedPriceRange 计算推荐的价格区间
func (s *Server) calculateRecommendedPriceRange(currentPrice, volatility float64, historicalPrices []float64) (upper, lower float64) {
	if currentPrice <= 0 {
		return 0, 0
	}

	// 基于历史价格计算价格范围
	minPrice, maxPrice := currentPrice, currentPrice
	for _, price := range historicalPrices {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}

	// 计算价格范围的安全边际
	priceRange := maxPrice - minPrice
	if priceRange <= 0 {
		priceRange = currentPrice * volatility * 2 // 基于波动率的默认范围
	}

	// 设置安全边际（避免价格突破网格）
	safetyMargin := math.Max(priceRange*0.2, currentPrice*volatility)

	// 计算推荐的上下限
	upper = currentPrice + (priceRange/2)*1.1 + safetyMargin
	lower = currentPrice - (priceRange/2)*1.1 - safetyMargin

	// 确保下限不小于0
	if lower <= 0 {
		lower = currentPrice * 0.1 // 最低10%的安全边际
	}

	return upper, lower
}
