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
		s.BadRequest(c, "鏃堕棿妗舵牸寮忛敊璇?, err)
		return
	}

	fetchedAt := time.Now().UTC()
	if body.FetchedAt != "" {
		if t, e := time.Parse(time.RFC3339, body.FetchedAt); e == nil {
			fetchedAt = t
		}
	}

	// 瀛樺簱缁熶竴鐢?UTC + 1h 瀵归綈
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
		s.DatabaseError(c, "淇濆瓨甯傚満鏁版嵁", err)
		return
	}

	// 澶辨晥甯傚満鏁版嵁缂撳瓨锛屼娇鏂版暟鎹珛鍗崇敓鏁?	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// binanceMarketParams 甯傚満鏌ヨ鍙傛暟
type binanceMarketParams struct {
	Kind        string
	IntervalMin int
	Location    *time.Location
	Date        string
	Slot        string
	Category    string // 鏂板锛氬竵绉嶅垎绫诲弬鏁?}

// parseBinanceMarketParams 瑙ｆ瀽甯傚満鏌ヨ鍙傛暟
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

// calculateTimeRange 璁＄畻鏃堕棿鑼冨洿
func calculateTimeRange(params *binanceMarketParams) (time.Time, time.Time, error) {
	if params.Date == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("date is required")
	}

	dayStartLocal, err := time.ParseInLocation("2006-01-02", params.Date, params.Location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("鏃ユ湡鏍煎紡閿欒锛屽簲涓?YYYY-MM-DD: %w", err)
	}

	var startLocal, endLocal time.Time
	if params.Slot != "" {
		slot, err := strconv.Atoi(params.Slot)
		if err != nil || slot < 0 || slot > (24*60/params.IntervalMin-1) {
			return time.Time{}, time.Time{}, fmt.Errorf("鏃堕棿娈电紪鍙锋棤鏁?)
		}
		startLocal = dayStartLocal.Add(time.Duration(slot) * time.Minute * time.Duration(params.IntervalMin))
		endLocal = startLocal.Add(time.Duration(params.IntervalMin) * time.Minute)
	} else {
		startLocal = dayStartLocal
		endLocal = dayStartLocal.Add(24 * time.Hour)
	}

	return startLocal.UTC(), endLocal.UTC(), nil
}

// filterAndFormatMarketData 杩囨护榛戝悕鍗曞苟鏍煎紡鍖栧競鍦烘暟鎹?func (s *Server) filterAndFormatMarketData(snaps []pdb.BinanceMarketSnapshot, tops map[uint][]pdb.BinanceMarketTop, kind string, ctx context.Context) ([]gin.H, error) {
	// 鑾峰彇榛戝悕鍗曪紙鐜拌揣鍜屾湡璐ч兘鏀寔锛? 浣跨敤缂撳瓨
	blacklistMap, err := s.getCachedBlacklistMap(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get cached blacklist (kind=%s), falling back to direct query: %v", kind, err)
		// 缂撳瓨澶辫触鏃堕檷绾у埌鐩存帴鏌ヨ锛屼絾涓嶅奖鍝嶄富娴佺▼
		blacklistMap = make(map[string]bool)
		if blacklist, e := s.db.GetBinanceBlacklist(kind); e == nil {
			for _, symbol := range blacklist {
				blacklistMap[strings.ToUpper(symbol)] = true
			}
		}
	}

	// 浼樺寲锛氶浼拌緭鍑哄垏鐗囧ぇ灏?	out := make([]gin.H, 0, len(snaps))
	for _, snap := range snaps {
		list := tops[snap.ID]
		// 杩囨护榛戝悕鍗曪紙Symbol 宸叉槸澶у啓锛岀洿鎺ヤ娇鐢級
		// 浼樺寲锛氶浼拌繃婊ゅ悗鐨勫垏鐗囧ぇ灏忥紙鍋囪鏈€澶氫繚鐣?0涓級
		estimatedSize := len(list)
		if estimatedSize > 10 {
			estimatedSize = 10
		}
		filtered := make([]pdb.BinanceMarketTop, 0, estimatedSize)
		for _, it := range list {
			if !blacklistMap[it.Symbol] {
				filtered = append(filtered, it)
				// 浼樺寲锛氬鏋滃凡缁忚揪鍒?0涓紝鎻愬墠閫€鍑哄惊鐜?				if len(filtered) >= 10 {
					break
				}
			}
		}
		// 鍙栧墠10涓紙濡傛灉瓒呰繃10涓級
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}
		// 浼樺寲锛氶浼?items 鍒囩墖澶у皬
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

// filterAndFormatMarketDataWithCategory 杩囨护榛戝悕鍗曞拰鍒嗙被骞舵牸寮忓寲甯傚満鏁版嵁
func (s *Server) filterAndFormatMarketDataWithCategory(snaps []pdb.BinanceMarketSnapshot, tops map[uint][]pdb.BinanceMarketTop, kind string, category string, ctx context.Context) ([]gin.H, error) {
	// 鑾峰彇榛戝悕鍗曪紙鐜拌揣鍜屾湡璐ч兘鏀寔锛? 浣跨敤缂撳瓨
	blacklistMap, err := s.getCachedBlacklistMap(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get cached blacklist (kind=%s), falling back to direct query: %v", kind, err)
		// 缂撳瓨澶辫触鏃堕檷绾у埌鐩存帴鏌ヨ锛屼絾涓嶅奖鍝嶄富娴佺▼
		blacklistMap = make(map[string]bool)
		if blacklist, e := s.db.GetBinanceBlacklist(kind); e == nil {
			for _, symbol := range blacklist {
				blacklistMap[strings.ToUpper(symbol)] = true
			}
		}
	}

	// 鑾峰彇exchangeInfo鏁版嵁鐢ㄤ簬鍒嗙被绛涢€?	exchangeInfo, err := s.getExchangeInfoForCategory(ctx, kind)
	if err != nil {
		log.Printf("[WARN] Failed to get exchange info for category filtering: %v", err)
		// 濡傛灉鑾峰彇澶辫触锛岀户缁鐞嗕絾涓嶈繘琛屽垎绫荤瓫閫?	}

	// 浼樺寲锛氶浼拌緭鍑哄垏鐗囧ぇ灏?	out := make([]gin.H, 0, len(snaps))
	for _, snap := range snaps {
		list := tops[snap.ID]

		// 棣栧厛杩囨护榛戝悕鍗曞拰鍒嗙被
		filtered := s.filterMarketDataByCategoryAndBlacklist(list, blacklistMap, category, exchangeInfo, kind)

		// 鍙栧墠10涓紙濡傛灉瓒呰繃10涓級
		if len(filtered) > 10 {
			filtered = filtered[:10]
		}

		// 浼樺寲锛氶浼?items 鍒囩墖澶у皬
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

// filterMarketDataByCategoryAndBlacklist 鏍规嵁鍒嗙被鍜岄粦鍚嶅崟杩囨护甯傚満鏁版嵁
func (s *Server) filterMarketDataByCategoryAndBlacklist(list []pdb.BinanceMarketTop, blacklistMap map[string]bool, category string, exchangeInfo map[string]ExchangeInfoItem, kind string) []pdb.BinanceMarketTop {
	if category == "" || category == "all" {
		// 濡傛灉娌℃湁鍒嗙被瑕佹眰锛屽彧杩囨护榛戝悕鍗?		filtered := make([]pdb.BinanceMarketTop, 0, len(list))
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

	// 鏍规嵁鍒嗙被杩涜绛涢€?	filtered := make([]pdb.BinanceMarketTop, 0, len(list))
	matchedCount := 0
	for _, it := range list {
		// 鍏堟鏌ラ粦鍚嶅崟
		if blacklistMap[it.Symbol] {
			continue
		}

		// 鏍规嵁鍒嗙被杩涜绛涢€?		if s.matchesCategory(it.Symbol, category, exchangeInfo, kind) {
			filtered = append(filtered, it)
			matchedCount++
			if len(filtered) >= 10 {
				break
			}
		}
	}
	return filtered
}

// ExchangeInfoItem exchangeInfo涓殑浜ゆ槗瀵逛俊鎭?type ExchangeInfoItem struct {
	Symbol      string   `json:"symbol"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
	BaseAsset   string   `json:"baseAsset"`
	QuoteAsset  string   `json:"quoteAsset"`
}

// matchesCategory 妫€鏌ヤ氦鏄撳鏄惁鍖归厤鎸囧畾鐨勫垎绫?func (s *Server) matchesCategory(symbol, category string, exchangeInfo map[string]ExchangeInfoItem, kind string) bool {
	// 濡傛灉娌℃湁exchangeInfo鏁版嵁锛岄粯璁ら€氳繃
	if exchangeInfo == nil {
		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	info, exists := exchangeInfo[symbol]
	if !exists {
		// 濡傛灉exchangeInfo涓病鏈夎繖涓氦鏄撳锛岄粯璁ら€氳繃
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
		// 浣跨敤exchangeInfo鐨刡aseAsset杩涜鏅鸿兘鍒嗙被
		result := s.isAssetTypeMatch(info.BaseAsset, category)
		return result
	case "layer1":
		// Layer1璧勪骇鐗规畩澶勭悊
		if info.BaseAsset != "" {
			result := s.isAssetTypeMatch(info.BaseAsset, category)
			return result
		}
		// 闄嶇骇鍒板熀浜庝氦鏄撳绗﹀彿鐨勬鏌?		layer1Assets := []string{"ATOM", "NEAR", "FTM", "ONE", "EGLD", "FLOW"}
		baseSymbol := s.getBaseSymbol(symbol)
		return s.containsString(layer1Assets, baseSymbol)
	case "meme":
		// Meme璧勪骇鐗规畩澶勭悊
		if info.BaseAsset != "" {
			result := s.isAssetTypeMatch(info.BaseAsset, category)
			return result
		}
		// 闄嶇骇鍒板熀浜庝氦鏄撳绗﹀彿鐨勬鏌?		memeAssets := []string{"SHIB", "DOGE", "PEPE", "BONK", "WIF", "TURBO"}
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

// matchesCategoryBySymbolOnly 浠呭熀浜庝氦鏄撳绗﹀彿杩涜鍒嗙被鍖归厤锛堝綋娌℃湁exchangeInfo鏃朵娇鐢級
func (s *Server) matchesCategoryBySymbolOnly(symbol, category string) bool {
	baseSymbol := s.getBaseSymbol(symbol)

	// 褰撴病鏈塭xchangeInfo鏃讹紝涔熶娇鐢ㄦ櫤鑳藉垎绫?	return s.isAssetTypeMatch(baseSymbol, category)
}

// getBaseSymbol 鑾峰彇浜ゆ槗瀵圭殑鍩虹甯佺
func (s *Server) getBaseSymbol(symbol string) string {
	// 鍘绘帀甯歌鐨勫悗缂€
	suffixes := []string{"USDT", "USDC", "BUSD", "BTC", "ETH", "BNB"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToUpper(symbol), suffix) {
			return strings.TrimSuffix(strings.ToUpper(symbol), suffix)
		}
	}
	return strings.ToUpper(symbol)
}

// containsString 妫€鏌ュ瓧绗︿覆鍒囩墖鏄惁鍖呭惈鎸囧畾瀛楃涓?func (s *Server) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isAssetTypeMatch 鍩轰簬璧勪骇绫诲瀷杩涜鏅鸿兘鍒嗙被
func (s *Server) isAssetTypeMatch(baseAsset, category string) bool {
	asset := strings.ToUpper(baseAsset)

	switch category {
	case "major":
		// 涓绘祦甯佺锛氬熀浜庡競鍦鸿鍙害鍜屼氦鏄撻噺
		majorCoins := map[string]bool{
			"BTC": true, "ETH": true, "BNB": true, "ADA": true, "SOL": true,
			"DOT": true, "AVAX": true, "MATIC": true, "LINK": true, "LTC": true,
			"ALGO": true, "VET": true, "ICP": true, "FIL": true, "TRX": true,
			"ETC": true, "XLM": true, "THETA": true, "FTM": true, "HBAR": true,
		}
		return majorCoins[asset]

	case "stable":
		// 绋冲畾甯侊細鍩轰簬鏄惁涓虹ǔ瀹氬竵
		stableCoins := map[string]bool{
			"USDT": true, "USDC": true, "BUSD": true, "DAI": true, "TUSD": true,
			"USDP": true, "FRAX": true, "LUSD": true, "SUSD": true, "USDJ": true,
			"USTC": true, "CUSD": true, "EUROC": true, "XSGD": true, "CEUR": true,
		}
		return stableCoins[asset]

	case "defi":
		// DeFi浠ｅ竵锛氬熀浜庢槸鍚︿负鍘讳腑蹇冨寲閲戣瀺鍗忚浠ｅ竵
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
		// Layer1鍏摼锛氬熀浜庢槸鍚︿负涓€灞傚尯鍧楅摼缃戠粶
		layer1Chains := map[string]bool{
			"ATOM": true, "NEAR": true, "FTM": true, "ONE": true, "EGLD": true,
			"FLOW": true, "MINA": true, "CELO": true, "KAVA": true, "SCRT": true,
			"GLMR": true, "MOVR": true, "CFG": true, "SDN": true, "ASTR": true,
			"ACA": true, "KAR": true, "BNC": true, "PKEX": true, "XPRT": true,
		}
		return layer1Chains[asset]

	case "meme":
		// Meme甯侊細鍩轰簬鏄惁涓烘ā鍥犲竵
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

// getExchangeInfoForCategory 鑾峰彇exchangeInfo鏁版嵁鐢ㄤ簬鍒嗙被绛涢€?func (s *Server) getExchangeInfoForCategory(ctx context.Context, kind string) (map[string]ExchangeInfoItem, error) {
	// 灏濊瘯浠庣紦瀛樿幏鍙?	cacheKey := fmt.Sprintf("exchange_info_%s", kind)
	if cached, exists := s.getCachedExchangeInfo(cacheKey); exists {
		return cached, nil
	}

	// 浠庡竵瀹堿PI鑾峰彇exchangeInfo
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

	log.Printf("[ExchangeInfo] 浠?%s 鑾峰彇鍒?%d 涓氦鏄撳淇℃伅", url, len(response.Symbols))

	// 杞崲涓簃ap浠ヤ究蹇€熸煡鎵?	exchangeInfoMap := make(map[string]ExchangeInfoItem)
	for _, symbol := range response.Symbols {
		exchangeInfoMap[symbol.Symbol] = symbol
	}

	// 缂撳瓨缁撴灉锛堢紦瀛?灏忔椂锛?	s.cacheExchangeInfo(cacheKey, exchangeInfoMap, time.Hour)

	return exchangeInfoMap, nil
}

// getCachedExchangeInfo 浠庣紦瀛樿幏鍙杄xchangeInfo
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

// cacheExchangeInfo 缂撳瓨exchangeInfo
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

// minInt 杩斿洖涓や釜鏁存暟涓殑杈冨皬鍊硷紙浼樺寲杈呭姪鍑芥暟锛?func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) GetBinanceMarket(c *gin.Context) {
	params, err := parseBinanceMarketParams(c)
	if err != nil {
		s.BadRequest(c, "鍙傛暟瑙ｆ瀽澶辫触", err)
		return
	}

	// 濡傛灉娌′紶 date锛岄粯璁や粖澶╋紙鏈湴鏃跺尯锛?	if params.Date == "" {
		day := time.Now().In(params.Location).Format("2006-01-02")
		q := c.Request.URL.Query()
		q.Set("date", day)
		c.Request.URL.RawQuery = q.Encode()
		// 閲嶆柊瑙ｆ瀽鍙傛暟
		params, err = parseBinanceMarketParams(c)
		if err != nil {
			s.BadRequest(c, "鍙傛暟瑙ｆ瀽澶辫触", err)
			return
		}
	}

	// 璁＄畻鏃堕棿鑼冨洿
	startUTC, endUTC, err := calculateTimeRange(params)
	if err != nil {
		s.ValidationError(c, "date", err.Error())
		return
	}

	// 鏌ヨ甯傚満鏁版嵁
	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), params.Kind, startUTC, endUTC)
	if err != nil {
		s.DatabaseError(c, "鏌ヨ甯傚満鏁版嵁", err)
		return
	}

	// 杩囨护鍜屾牸寮忓寲鏁版嵁
	out, err := s.filterAndFormatMarketDataWithCategory(snaps, tops, params.Kind, params.Category, c.Request.Context())
	if err != nil {
		s.InternalServerError(c, "澶勭悊甯傚満鏁版嵁澶辫触", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"kind":     params.Kind,
		"interval": params.IntervalMin,
		"data":     out,
	})
}

// WSRealTimeGainers WebSocket瀹炴椂娑ㄥ箙姒?func (s *Server) WSRealTimeGainers(c *gin.Context) {
	// 鍗囩骇HTTP杩炴帴涓篧ebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] 娑ㄥ箙姒滆繛鎺ュ崌绾уけ璐? %v", err)
		return
	}
	defer ws.Close()

	clientIP := c.ClientIP()
	log.Printf("[WebSocket] 娑ㄥ箙姒滄柊杩炴帴寤虹珛: %s", clientIP)

	// 璇诲彇瀹㈡埛绔殑璁㈤槄娑堟伅
	_, message, err := ws.ReadMessage()
	if err != nil {
		log.Printf("[WebSocket] 璇诲彇娑ㄥ箙姒滆闃呮秷鎭け璐? %v", err)
		return
	}

	// 瑙ｆ瀽璁㈤槄璇锋眰
	var subscription struct {
		Action   string `json:"action"`
		Kind     string `json:"kind"`     // "spot" 鎴?"futures"
		Category string `json:"category"` // 鍒嗙被绛涢€夛紝濡?"trading", "all" 绛?		Limit    int    `json:"limit"`    // 杩斿洖鏁伴噺闄愬埗
		Interval int    `json:"interval"` // 鏇存柊闂撮殧(绉?锛岄粯璁?0绉?	}

	if err := json.Unmarshal(message, &subscription); err != nil {
		log.Printf("[WebSocket] 瑙ｆ瀽娑ㄥ箙姒滆闃呮秷鎭け璐? %v", err)
		ws.WriteJSON(gin.H{"error": "鏃犳晥鐨勮闃呮牸寮?})
		return
	}

	log.Printf("[WebSocket] 鏀跺埌璁㈤槄璇锋眰: action=%s, kind=%s, category=%s, limit=%d, interval=%d",
		subscription.Action, subscription.Kind, subscription.Category, subscription.Limit, subscription.Interval)

	if subscription.Action != "subscribe" {
		log.Printf("[WebSocket] 涓嶆敮鎸佺殑鎿嶄綔: %s", subscription.Action)
		ws.WriteJSON(gin.H{"error": "涓嶆敮鎸佺殑鎿嶄綔"})
		return
	}

	// 璁剧疆榛樿鍊?	if subscription.Kind == "" {
		subscription.Kind = "spot"
	}
	if subscription.Limit <= 0 || subscription.Limit > 50 {
		subscription.Limit = 15
	}
	if subscription.Interval <= 0 || subscription.Interval > 300 {
		subscription.Interval = 20 // 榛樿20绉?	}

	log.Printf("[WebSocket] 娑ㄥ箙姒滆闃? kind=%s, limit=%d, interval=%ds",
		subscription.Kind, subscription.Limit, subscription.Interval)

	// 鍙戦€佺‘璁ゆ秷鎭?	ws.WriteJSON(gin.H{
		"type":    "subscription_confirmed",
		"message": "瀹炴椂娑ㄥ箙姒滆闃呮垚鍔?,
		"config": gin.H{
			"kind":     subscription.Kind,
			"category": subscription.Category,
			"limit":    subscription.Limit,
			"interval": subscription.Interval,
		},
	})

	ctx := context.Background()
	var lastGainersData []gin.H // 缂撳瓨涓婃鐨勬暟鎹紝鐢ㄤ簬姣旇緝鍙樺寲

	// 绔嬪嵆鍙戦€佺涓€鎵规暟鎹?	log.Printf("[WebSocket] 寮€濮嬬敓鎴愬垵濮嬫定骞呮鏁版嵁...")
	gainersData, err := s.generateRealtimeGainersData(ctx, subscription.Kind, subscription.Category, subscription.Limit)
	if err != nil {
		log.Printf("[WebSocket] 鐢熸垚鍒濆娑ㄥ箙姒滄暟鎹け璐? %v", err)
	} else {
		log.Printf("[WebSocket] 鎴愬姛鐢熸垚娑ㄥ箙姒滄暟鎹紝鏉℃暟: %d", len(gainersData))
		lastGainersData = make([]gin.H, len(gainersData))
		copy(lastGainersData, gainersData) // 娣辨嫹璐濇暟鎹敤浜庢瘮杈?
		response := gin.H{
			"type":      "gainers_update",
			"timestamp": time.Now().Unix(),
			"kind":      subscription.Kind,
			"limit":     subscription.Limit,
			"gainers":   gainersData,
		}
		//log.Printf("[WebSocket] 鍙戦€佸垵濮嬫定骞呮鏁版嵁...")
		if err := ws.WriteJSON(response); err != nil {
			log.Printf("[WebSocket] 鍙戦€佸垵濮嬫定骞呮鏁版嵁澶辫触: %v", err)
			return
		}
		log.Printf("[WebSocket] 鍒濆娑ㄥ箙姒滄暟鎹彂閫佹垚鍔?)
	}

	// 鍒涘缓瀹氭椂鍣ㄥ彂閫佸疄鏃舵洿鏂?	ticker := time.NewTicker(time.Duration(subscription.Interval) * time.Second)
	defer ticker.Stop()

	// 鍙戦€佸疄鏃舵洿鏂?	for {
		select {
		case <-ticker.C:
			log.Printf("[WebSocket] 瀹氭椂鍣ㄨЕ鍙戯紝寮€濮嬬敓鎴愭定骞呮鏇存柊鏁版嵁...")
			// 鑾峰彇瀹炴椂娑ㄥ箙姒滄暟鎹?			gainersData, err := s.generateRealtimeGainersData(ctx, subscription.Kind, subscription.Category, subscription.Limit)
			if err != nil {
				log.Printf("[WebSocket] 鐢熸垚娑ㄥ箙姒滄暟鎹け璐? %v", err)
				continue
			}

			// 妫€鏌ユ暟鎹槸鍚︽湁鏄捐憲鍙樺寲
			if !s.hasSignificantChanges(lastGainersData, gainersData) {
				log.Printf("[WebSocket] 鏁版嵁鏃犳樉钁楀彉鍖栵紝璺宠繃鏈鏇存柊")
				continue
			}

			// 鏇存柊缂撳瓨
			lastGainersData = make([]gin.H, len(gainersData))
			copy(lastGainersData, gainersData)

			// 寮傛淇濆瓨鍘嗗彶鏁版嵁锛堜笉闃诲WebSocket鍝嶅簲锛?			// 娉ㄦ剰锛氬疄鏃舵暟鎹幇鍦ㄧ洿鎺ヤ粠 binance_24h_stats 鐢熸垚锛屼笉鍐嶉渶瑕佷繚瀛樺埌 realtime_gainers_items
			// 鍙湪闇€瑕佸巻鍙叉暟鎹瓨妗ｆ椂鎵嶄繚瀛橈紙鍙互鏍规嵁閰嶇疆鍐冲畾锛?			log.Printf("[WebSocket] 瀹炴椂娑ㄥ箙姒滄暟鎹凡浼樺寲锛岀洿鎺ヤ粠 binance_24h_stats 鐢熸垚")

			log.Printf("[WebSocket] 鍙戦€佸畾鏃舵定骞呮鏇存柊锛屾潯鏁? %d", len(gainersData))
			// 鍙戦€佹洿鏂?			response := gin.H{
				"type":      "gainers_update",
				"timestamp": time.Now().Unix(),
				"kind":      subscription.Kind,
				"limit":     subscription.Limit,
				"gainers":   gainersData,
			}

			if err := ws.WriteJSON(response); err != nil {
				log.Printf("[WebSocket] 鍙戦€佹定骞呮鏇存柊澶辫触: %v", err)
				return
			}
			log.Printf("[WebSocket] 瀹氭椂娑ㄥ箙姒滄洿鏂板彂閫佹垚鍔?)

		case <-ctx.Done():
			log.Printf("[WebSocket] 娑ㄥ箙姒滆繛鎺ヤ笂涓嬫枃鍙栨秷")
			return
		}
	}
}

// generateRealtimeGainersFrom24hStats 鐩存帴浠?binance_24h_stats 鐢熸垚娑ㄥ箙姒滄暟鎹紙浼樺寲鐗堟湰锛?func (s *Server) generateRealtimeGainersFrom24hStats(ctx context.Context, kind string, category string, limit int) ([]gin.H, error) {
	// 缂撳瓨閿?	cacheKey := fmt.Sprintf("gainers_24h_%s_%s_%d", kind, category, limit)

	// 妫€鏌ョ紦瀛?	if cached, exists := s.getCachedGainers(cacheKey); exists {
		log.Printf("[娑ㄥ箙姒?24h] 浣跨敤缂撳瓨鏁版嵁: %s", cacheKey)
		return cached, nil
	}

	log.Printf("[娑ㄥ箙姒?24h] 缂撳瓨鏈懡涓紝浠?binance_24h_stats 鐢熸垚鏁版嵁: %s", cacheKey)

	// 纭畾瀹為檯杩斿洖鏁伴噺锛氬墠绔定骞呮鍥哄畾15涓紝鍏朵粬璋冪敤鍙繑鍥炴洿澶氱敤浜庣瓫閫?	actualLimit := 15 // 鍓嶇榛樿15涓?	if limit > 15 && limit <= 100 {
		// 绛栫暐鎵弿鍣ㄧ瓑璋冪敤鍏佽杩斿洖鏇村鏁版嵁鐢ㄤ簬绛涢€?		actualLimit = limit
	}

	// 鐩存帴浠?binance_24h_stats 鏌ヨ娑ㄥ箙姒滄暟鎹?	query := fmt.Sprintf(`
		SELECT
			symbol,
			price_change_percent,
			volume,
			quote_volume,
			last_price,
			ROW_NUMBER() OVER (ORDER BY price_change_percent DESC, volume DESC) as ranking
		FROM binance_24h_stats
		WHERE market_type = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
			AND volume > 0 AND last_price > 0  -- 棰勮繃婊ゆ棤鏁堟暟鎹?		ORDER BY price_change_percent DESC, volume DESC
		LIMIT %d
	`, actualLimit)

	var results []struct {
		Symbol             string  `json:"symbol"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		QuoteVolume        float64 `json:"quote_volume"`
		LastPrice          float64 `json:"last_price"`
		Rank               int     `json:"rank"`
	}

	err := s.db.DB().Raw(query, kind).Scan(&results).Error // 鐩存帴鏌ヨ鍓?5鍚?	if err != nil {
		log.Printf("[娑ㄥ箙姒?24h] 鏌ヨ binance_24h_stats 澶辫触: %v", err)
		return nil, fmt.Errorf("鏌ヨ娑ㄥ箙姒滄暟鎹け璐? %w", err)
	}

	if len(results) == 0 {
		log.Printf("[娑ㄥ箙姒?24h] 娌℃湁鎵惧埌 %s 甯傚満鐨勬暟鎹?, kind)
		return []gin.H{}, nil
	}

	// 杞崲涓哄墠绔湡鏈涚殑鏍煎紡锛堢洿鎺ヨ繑鍥炲墠15鍚嶏紝鏃犻渶棰濆杩囨护锛?	gainers := make([]gin.H, 0, len(results))
	for _, item := range results {
		gainer := gin.H{
			"symbol":           item.Symbol,
			"current_price":    item.LastPrice,
			"price_change_24h": item.PriceChangePercent,
			"volume_24h":       item.Volume,
			"quote_volume_24h": item.QuoteVolume,
			"rank":             item.Rank,
			"data_source":      "24h_stats",
			"price_change":     item.PriceChangePercent, // 鍏煎鏃у瓧娈?			"change":           item.PriceChangePercent, // 鍓嶇鍙兘浣跨敤鐨勫瓧娈?		}

		// 娣诲姞甯傚€间及绠?		if item.QuoteVolume > 0 {
			gainer["market_cap"] = item.QuoteVolume // 绠€鍖栧競鍊间及绠?		}

		gainers = append(gainers, gainer)
	}

	// 缂撳瓨缁撴灉锛?鍒嗛挓锛?	s.cacheGainersWithDuration(cacheKey, gainers, 5*time.Minute)

	log.Printf("[娑ㄥ箙姒?24h] 鎴愬姛鐢熸垚鍓?5鍚?%s 甯傚満娑ㄥ箙姒滄暟鎹紝鍏?%d 鏉?, kind, len(gainers))
	return gainers, nil
}

// generateRealtimeGainersData 鐢熸垚瀹炴椂娑ㄥ箙姒滄暟鎹紙淇濈暀鏃х増鏈敤浜庡吋瀹癸級
func (s *Server) generateRealtimeGainersData(ctx context.Context, kind string, category string, limit int) ([]gin.H, error) {
	// 缂撳瓨閿?	cacheKey := fmt.Sprintf("gainers_%s_%s_%d", kind, category, limit)

	// 鏅鸿兘缂撳瓨绛栫暐锛氭牴鎹競鍦烘椿璺冨害鍔ㄦ€佽皟鏁寸紦瀛樻椂闂?	cacheDuration := s.getDynamicCacheDuration(kind)

	// 妫€鏌ョ紦瀛橈紙甯﹁繃鏈熸椂闂存鏌ワ級
	if cached, exists := s.getCachedGainersWithDuration(cacheKey, cacheDuration); exists {
		log.Printf("[娑ㄥ箙姒淽 浣跨敤缂撳瓨鏁版嵁: %s (缂撳瓨鏃堕暱: %v)", cacheKey, cacheDuration)
		return cached, nil
	}

	log.Printf("[娑ㄥ箙姒淽 缂撳瓨鏈懡涓紝寮€濮嬭幏鍙栨柊鏁版嵁: %s", cacheKey)

	// 鑾峰彇鐑棬甯佺鍒楄〃 - 浠巄inance_market_snapshots鍜宐inance_market_tops鑾峰彇鏈€鏂扮殑涓€涓揩鐓х殑鏁版嵁
	var symbols []string
	dbInstance := s.db.DB()

	// 浠庢渶鏂扮殑蹇収涓幏鍙栦氦鏄撳鏁版嵁锛屾寜鐓ф定骞呮鎺掑簭
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
		log.Printf("[娑ㄥ箙姒淽 鏁版嵁搴撴煡璇㈠け璐ワ紝鏃犳硶鑾峰彇甯佺鍒楄〃: %v", err)
		return nil, fmt.Errorf("鑾峰彇鍙敤甯佺鍒楄〃澶辫触: %w", err)
	}

	// 濡傛灉娌℃湁鑾峰彇鍒版暟鎹紝杩斿洖閿欒
	if len(symbols) == 0 {
		log.Printf("[娑ㄥ箙姒淽 鏁版嵁搴撲腑娌℃湁鍙敤甯佺鏁版嵁")
		return nil, fmt.Errorf("鏁版嵁搴撲腑娌℃湁鍙敤甯佺鏁版嵁")
	}

	// 浣跨敤骞跺彂鑾峰彇鏁版嵁浠ユ彁楂樻€ц兘
	gainersChan := make(chan gin.H, len(symbols))
	var wg sync.WaitGroup
	var wsCount, httpCount, fallbackCount int32 // 浣跨敤鍘熷瓙鎿嶄綔
	var processedCount int32

	// 闄愬埗骞跺彂鏁伴噺锛岄伩鍏嶈繃杞?	maxConcurrency := 8 // 鍑忓皯骞跺彂鏁伴噺浠ユ彁楂樼ǔ瀹氭€?	semaphore := make(chan struct{}, maxConcurrency)

	// 娣诲姞瓒呮椂鎺у埗
	timeout := 15 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			defer atomic.AddInt32(&processedCount, 1)

			// 鑾峰彇淇″彿閲?			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctxWithTimeout.Done():
				return
			}

			// 妫€鏌ヤ笂涓嬫枃鏄惁宸插彇娑?			select {
			case <-ctxWithTimeout.Done():
				return
			default:
			}

			realtimeData, success := s.getRealtimeDataConcurrently(ctxWithTimeout, sym, kind)
			if !success {
				log.Printf("[娑ㄥ箙姒淽 %s 鏁版嵁鑾峰彇澶辫触锛屽皾璇曢檷绾ц幏鍙?, sym)
				// 灏濊瘯闄嶇骇鑾峰彇锛堢畝鍖栫増鏁版嵁锛?				fallbackData := s.getFallbackRealtimeData(sym, kind)
				if fallbackData.LastPrice > 0 {
					log.Printf("[娑ㄥ箙姒淽 %s 闄嶇骇鏁版嵁鑾峰彇鎴愬姛", sym)
					// 鑾峰彇鍒嗙被淇℃伅
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

			// 缁熻鏁版嵁婧?			switch realtimeData.DataSource {
			case "websocket":
				atomic.AddInt32(&wsCount, 1)
			case "http_api":
				atomic.AddInt32(&httpCount, 1)
			default:
				atomic.AddInt32(&fallbackCount, 1)
			}

			// 鏁版嵁璐ㄩ噺妫€鏌ュ拰寮傚父妫€娴?			if !s.validateRealtimeData(realtimeData) {
				return
			}

			// 鑾峰彇浜ゆ槗瀵瑰垎绫讳俊鎭?			var categoryInfo gin.H
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
				// 榛樿鍒嗙被淇℃伅
				categoryInfo = gin.H{
					"status":      "UNKNOWN",
					"asset_type":  "emerging",
					"market_cap":  "mid",
					"trade_type":  "spot_only",
					"order_level": "basic",
					"is_active":   true,
				}
			}

			// 杞崲涓哄墠绔湡鏈涚殑鏍煎紡
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

			// 闈為樆濉炲彂閫佺粨鏋?			select {
			case gainersChan <- gainer:
			case <-ctxWithTimeout.Done():
			}
		}(symbol)
	}

	// 绛夊緟鎵€鏈塯oroutine瀹屾垚鎴栬秴鏃?	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[娑ㄥ箙姒淽 鎵€鏈夋暟鎹幏鍙栧畬鎴愶紝鍏卞鐞?%d 涓竵绉?, atomic.LoadInt32(&processedCount))
	case <-ctxWithTimeout.Done():
		log.Printf("[娑ㄥ箙姒淽 鏁版嵁鑾峰彇瓒呮椂锛屽凡澶勭悊 %d/%d 涓竵绉?, atomic.LoadInt32(&processedCount), len(symbols))
		// 绛夊緟鎵€鏈塯oroutine鐪熸瀹屾垚鍚庡啀鍏抽棴channel锛岄伩鍏?send on closed channel"panic
		wg.Wait()
	}

	close(gainersChan)

	// 鏀堕泦缁撴灉
	gainers := make([]gin.H, 0, len(symbols))
	for gainer := range gainersChan {
		gainers = append(gainers, gainer)
	}

	// 鏁版嵁寮傚父鐩戞帶
	s.monitorDataQuality(gainers, kind)

	// 璁板綍鏁版嵁婧愮粺璁?	log.Printf("[娑ㄥ箙姒淽 鏁版嵁鑾峰彇瀹屾垚: 鎬昏%d涓竵绉? WebSocket=%d, HTTP_API=%d, 闄嶇骇=%d",
		len(symbols), atomic.LoadInt32(&wsCount), atomic.LoadInt32(&httpCount), atomic.LoadInt32(&fallbackCount))

	// 浣跨敤鍔ㄦ€佺紦瀛樻椂闀?	s.cacheGainersWithDuration(cacheKey, gainers, cacheDuration)

	// 浣跨敤鏇撮珮鏁堢殑鎺掑簭绠楁硶
	s.sortGainersByChangePercent(gainers)

	// 鑾峰彇榛戝悕鍗曞苟杩涜榛戝悕鍗?鍒嗙被绛涢€夛紙鍙傝€冩定骞呮鐨勯€昏緫锛?	blacklistMap, err := s.getCachedBlacklistMap(context.Background(), kind)
	if err != nil {
		log.Printf("[WARN] 鑾峰彇榛戝悕鍗曞け璐ワ紝浣跨敤绌洪粦鍚嶅崟: %v", err)
		blacklistMap = make(map[string]bool)
	}

	// 鑾峰彇exchangeInfo鐢ㄤ簬鍒嗙被绛涢€?	exchangeInfo, err := s.getExchangeInfoForCategory(ctx, kind)
	if err != nil {
		log.Printf("[WARN] 鑾峰彇exchangeInfo澶辫触: %v", err)
	}

	// 杩涜榛戝悕鍗曞拰鍒嗙被绛涢€?	gainers = s.filterGainersByBlacklistAndCategory(gainers, blacklistMap, category, exchangeInfo, kind, limit)

	// 闄愬埗杩斿洖鏁伴噺
	if len(gainers) > limit {
		gainers = gainers[:limit]
	}

	// 娣诲姞鎺掑悕
	for i, gainer := range gainers {
		gainer["rank"] = i + 1
	}

	return gainers, nil
}

// filterGainersByBlacklistAndCategory 鏍规嵁榛戝悕鍗曞拰鍒嗙被绛涢€夊疄鏃舵定骞呮鏁版嵁锛堝弬鑰冩定骞呮閫昏緫锛?func (s *Server) filterGainersByBlacklistAndCategory(gainers []gin.H, blacklistMap map[string]bool, category string, exchangeInfo map[string]ExchangeInfoItem, kind string, maxCount int) []gin.H {
	if category == "" || category == "all" {
		// 濡傛灉娌℃湁鍒嗙被瑕佹眰锛屽彧杩囨护榛戝悕鍗?		filtered := make([]gin.H, 0, len(gainers))
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

	// 鏍规嵁鍒嗙被杩涜绛涢€?	filtered := make([]gin.H, 0, len(gainers))
	matchedCount := 0
	for _, gainer := range gainers {
		symbol, _ := gainer["symbol"].(string)

		// 鍏堟鏌ラ粦鍚嶅崟
		if blacklistMap[strings.ToUpper(symbol)] {
			continue
		}

		// 鏍规嵁鍒嗙被杩涜绛涢€?		if s.matchesGainerCategoryForRealtime(gainer, category, exchangeInfo, kind) {
			filtered = append(filtered, gainer)
			matchedCount++
			if len(filtered) >= maxCount {
				break
			}
		}
	}
	return filtered
}

// matchesGainerCategoryForRealtime 妫€鏌ュ疄鏃舵定骞呮鏉＄洰鏄惁鍖归厤鎸囧畾鐨勫垎绫伙紙鍙傝€冩定骞呮閫昏緫锛?func (s *Server) matchesGainerCategoryForRealtime(gainer gin.H, category string, exchangeInfo map[string]ExchangeInfoItem, kind string) bool {
	symbol, _ := gainer["symbol"].(string)

	// 濡傛灉娌℃湁exchangeInfo鏁版嵁锛屼娇鐢ㄥ熀浜巗ymbol鐨勫尮閰?	if exchangeInfo == nil {
		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	info, exists := exchangeInfo[symbol]
	if !exists {
		// 濡傛灉exchangeInfo涓病鏈夎繖涓氦鏄撳锛屼娇鐢ㄥ熀浜巗ymbol鐨勫尮閰?		return s.matchesCategoryBySymbolOnly(symbol, category)
	}

	switch category {
	case "trading":
		return info.Status == "TRADING"
	case "break":
		return info.Status == "BREAK"
	case "major", "stable", "defi":
		// 浣跨敤exchangeInfo鐨刡aseAsset杩涜鏅鸿兘鍒嗙被
		return s.isAssetTypeMatch(info.BaseAsset, category)
	case "layer1":
		// Layer1璧勪骇鐗规畩澶勭悊
		if info.BaseAsset != "" {
			return s.isAssetTypeMatch(info.BaseAsset, category)
		}
		// 闄嶇骇鍒板熀浜庝氦鏄撳绗﹀彿鐨勬鏌?		layer1Assets := []string{"ATOM", "NEAR", "FTM", "ONE", "EGLD", "FLOW"}
		baseSymbol := s.getBaseSymbol(symbol)
		return s.containsString(layer1Assets, baseSymbol)
	case "meme":
		// Meme璧勪骇鐗规畩澶勭悊
		if info.BaseAsset != "" {
			return s.isAssetTypeMatch(info.BaseAsset, category)
		}
		// 闄嶇骇鍒板熀浜庝氦鏄撳绗﹀彿鐨勬鏌?		memeAssets := []string{"SHIB", "DOGE", "PEPE", "BONK", "WIF", "TURBO"}
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

// containsPermission 妫€鏌ユ潈闄愬垪琛ㄦ槸鍚﹀寘鍚寚瀹氭潈闄?func (s *Server) containsPermission(permissions []interface{}, permission string) bool {
	for _, p := range permissions {
		if perm, ok := p.(string); ok && perm == permission {
			return true
		}
	}
	return false
}

// SaveRealtimeGainersData 淇濆瓨瀹炴椂娑ㄥ箙姒滄暟鎹紙鍐呴儴API锛?func (s *Server) SaveRealtimeGainersData(ctx context.Context, kind string, gainers []gin.H) error {
	if len(gainers) == 0 {
		return nil
	}

	// 杞崲涓烘暟鎹簱缁撴瀯
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

		// 鍙€夊瓧娈?		if pc, ok := gainer["price_change_percent"].(float64); ok {
			item.PriceChangePercent = &pc
		}
		if conf, ok := gainer["confidence"].(float64); ok {
			item.Confidence = &conf
		}

		items = append(items, item)
	}

	// 淇濆瓨鍒版暟鎹簱
	_, err := pdb.SaveRealtimeGainers(s.db.DB(), kind, time.Now(), items)
	if err != nil {
		log.Printf("[娑ㄥ箙姒淽 淇濆瓨鍘嗗彶鏁版嵁澶辫触: %v", err)
		return err
	}

	log.Printf("[娑ㄥ箙姒淽 鎴愬姛淇濆瓨 %d 鏉℃定骞呮鍘嗗彶鏁版嵁", len(items))
	return nil
}

// GetRealtimeGainersHistoryAPI 鑾峰彇娑ㄥ箙姒滃巻鍙叉暟鎹瓵PI
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

	// 瑙ｆ瀽鏃堕棿
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

	// 鑾峰彇鍘嗗彶鏁版嵁
	snapshots, itemsMap, err := pdb.GetRealtimeGainersHistory(s.db.DB(), kind, startTime, endTime, symbol, limit)
	if err != nil {
		s.DatabaseError(c, "鑾峰彇娑ㄥ箙姒滃巻鍙叉暟鎹?, err)
		return
	}

	// 杞崲涓哄墠绔牸寮?	result := make([]gin.H, 0, len(snapshots))
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

// GetRealtimeGainersStatsAPI 鑾峰彇娑ㄥ箙姒滄暟鎹粺璁PI
// GET /market/binance/realtime-gainers/stats
func (s *Server) GetRealtimeGainersStatsAPI(c *gin.Context) {
	stats, err := pdb.GetRealtimeGainersStats(s.db.DB())
	if err != nil {
		s.DatabaseError(c, "鑾峰彇娑ㄥ箙姒滅粺璁℃暟鎹?, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"timestamp": time.Now().Unix(),
	})
}

// CleanRealtimeGainersDataAPI 娓呯悊鏃х殑娑ㄥ箙姒滄暟鎹瓵PI
// POST /market/binance/realtime-gainers/clean?keep_days=30
func (s *Server) CleanRealtimeGainersDataAPI(c *gin.Context) {
	keepDaysStr := c.DefaultQuery("keep_days", "30")
	keepDays, err := strconv.Atoi(keepDaysStr)
	if err != nil || keepDays <= 0 || keepDays > 365 {
		keepDays = 30
	}

	err = pdb.CleanOldRealtimeGainers(s.db.DB(), keepDays)
	if err != nil {
		s.DatabaseError(c, "娓呯悊娑ㄥ箙姒滄暟鎹?, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "娑ㄥ箙姒滄暟鎹竻鐞嗗畬鎴?,
		"keep_days": keepDays,
	})
}

// getDynamicCacheDuration 鏍规嵁甯傚満娲昏穬搴﹀姩鎬佽绠楃紦瀛樻椂闀?func (s *Server) getDynamicCacheDuration(kind string) time.Duration {
	// 鑾峰彇褰撳墠鏃堕棿
	now := time.Now()

	// 宸ヤ綔鏃ュ拰鍛ㄦ湯鐨勪笉鍚岀紦瀛樼瓥鐣?	weekday := now.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	// 鑾峰彇褰撳墠灏忔椂
	hour := now.Hour()

	// 浜氭床浜ゆ槗鏃舵锛?-8鐐癸級锛氭椿璺冨害杈冧綆
	// 娆ф床浜ゆ槗鏃舵锛?-16鐐癸級锛氫腑绛夋椿璺冨害
	// 缇庢床浜ゆ槗鏃舵锛?6-24鐐癸級锛氭渶楂樻椿璺冨害
	var baseDuration time.Duration
	switch {
	case hour >= 0 && hour < 8: // 浜氭床鏃舵
		baseDuration = 30 * time.Second
	case hour >= 8 && hour < 16: // 娆ф床鏃舵
		baseDuration = 20 * time.Second
	default: // 缇庢床鏃舵锛?6-24鐐癸級
		baseDuration = 15 * time.Second
	}

	// 鍛ㄦ湯閫傚綋澧炲姞缂撳瓨鏃堕棿
	if isWeekend {
		baseDuration = time.Duration(float64(baseDuration) * 1.5)
	}

	// 瀵逛簬鍚堢害锛岀紦瀛樻椂闂寸◢鐭紙甯傚満鏇存椿璺冿級
	if kind == "futures" {
		baseDuration = time.Duration(float64(baseDuration) * 0.8)
	}

	// 纭繚缂撳瓨鏃堕棿鍦ㄥ悎鐞嗚寖鍥村唴
	if baseDuration < 15*time.Second {
		baseDuration = 15 * time.Second
	}
	if baseDuration > 120*time.Second {
		baseDuration = 120 * time.Second
	}

	return baseDuration
}

// sortGainersByChangePercent 浣跨敤鏇撮珮鏁堢殑鎺掑簭绠楁硶鎸夋定骞呮帓搴?func (s *Server) sortGainersByChangePercent(gainers []gin.H) {
	// 浣跨敤sort鍖呰繘琛屾洿楂樻晥鐨勬帓搴?	sort.Slice(gainers, func(i, j int) bool {
		changeI, okI := gainers[i]["price_change_24h"].(float64)
		changeJ, okJ := gainers[j]["price_change_24h"].(float64)

		// 濡傛灉绫诲瀷涓嶅尮閰嶏紝鎸夊師椤哄簭淇濇寔
		if !okI && !okJ {
			return false
		}
		if !okI {
			return false // 鏈夐棶棰樼殑鏁版嵁鎺掑湪鍚庨潰
		}
		if !okJ {
			return true // 鏈夐棶棰樼殑鏁版嵁鎺掑湪鍚庨潰
		}

		// 闄嶅簭鎺掑垪锛氭定骞呴珮鐨勫湪鍓?		return changeI > changeJ
	})
}

// SymbolCategory 浜ゆ槗瀵瑰垎绫讳俊鎭?type SymbolCategory struct {
	Symbol      string   `json:"symbol"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
	OrderTypes  []string `json:"order_types"`
	BaseAsset   string   `json:"base_asset"`
	QuoteAsset  string   `json:"quote_asset"`
	AssetType   string   `json:"asset_type"`  // 璧勪骇绫诲瀷: major, stable, defi, layer1, meme, nft_gaming, emerging
	MarketCap   string   `json:"market_cap"`  // 甯傚€艰妯? large, mid, small
	TradeType   string   `json:"trade_type"`  // 浜ゆ槗绫诲瀷: spot_only, margin, leveraged, trading_groups
	OrderLevel  string   `json:"order_level"` // 璁㈠崟绾у埆: basic, stop_loss, take_profit, advanced, full_featured
	IsActive    bool     `json:"is_active"`   // 鏄惁娲昏穬浜ゆ槗
}

// RealtimeData 缁熶竴鐨勫疄鏃舵暟鎹粨鏋?type RealtimeData struct {
	Symbol        string          `json:"symbol"`
	LastPrice     float64         `json:"current_price"`
	ChangePercent float64         `json:"price_change_24h"`
	Volume        float64         `json:"volume_24h"`
	DataSource    string          `json:"data_source"` // "websocket", "http_api", "kline_calc"
	Timestamp     int64           `json:"timestamp"`
	Category      *SymbolCategory `json:"category,omitempty"` // 鍒嗙被淇℃伅
}

// getSymbolCategory 鑾峰彇浜ゆ槗瀵瑰垎绫讳俊鎭?func (s *Server) getSymbolCategory(symbol string, kind string) *SymbolCategory {
	// 灏濊瘯浠巈xchangeInfo鑾峰彇鐪熷疄鐨勫垎绫讳俊鎭?	ctx := context.Background()
	exchangeInfo, err := s.getExchangeInfoForCategory(ctx, kind)
	if err != nil {
		log.Printf("[WARN] 鑾峰彇exchangeInfo澶辫触锛屼娇鐢ㄩ粯璁ゅ垎绫? %v", err)
		return s.getDefaultSymbolCategory(symbol, kind)
	}

	info, exists := exchangeInfo[symbol]
	if !exists {
		log.Printf("[WARN] exchangeInfo涓湭鎵惧埌浜ゆ槗瀵?%s锛屼娇鐢ㄩ粯璁ゅ垎绫?, symbol)
		return s.getDefaultSymbolCategory(symbol, kind)
	}

	// 浠庣湡瀹炵殑exchangeInfo鏋勫缓鍒嗙被淇℃伅
	category := &SymbolCategory{
		Symbol:      symbol,
		Status:      info.Status,
		Permissions: info.Permissions,
		BaseAsset:   info.BaseAsset,
		QuoteAsset:  info.QuoteAsset,
		IsActive:    info.Status == "TRADING",
	}

	// 鏍规嵁鍩虹璧勪骇纭畾璧勪骇绫诲瀷
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

	// 鏍规嵁鏉冮檺纭畾浜ゆ槗绫诲瀷
	if s.containsString(info.Permissions, "LEVERAGED") {
		category.TradeType = "leveraged"
	} else if s.containsString(info.Permissions, "MARGIN") {
		category.TradeType = "margin"
	} else {
		category.TradeType = "spot_only"
	}

	// 璁剧疆甯傚€艰妯★紙杩欓噷浣跨敤绠€鍖栫殑閫昏緫锛?	category.MarketCap = "mid"

	// 璁剧疆璁㈠崟绾у埆锛堢畝鍖栦负basic锛?	category.OrderLevel = "basic"

	return category
}

// getDefaultSymbolCategory 鑾峰彇榛樿鐨勪氦鏄撳鍒嗙被淇℃伅锛堝綋鏃犳硶鑾峰彇鐪熷疄鏁版嵁鏃朵娇鐢級
func (s *Server) getDefaultSymbolCategory(symbol string, kind string) *SymbolCategory {
	// 瑙ｆ瀽鍩虹璧勪骇
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

	// 绠€鍗曠殑璧勪骇绫诲瀷鍒嗙被
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

	// 鏍规嵁浜ゆ槗绫诲瀷璁剧疆鏉冮檺
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

// getRealtimeDataConcurrently 骞跺彂鑾峰彇瀹炴椂鏁版嵁
func (s *Server) getRealtimeDataConcurrently(ctx context.Context, symbol string, kind string) (RealtimeData, bool) {
	// 1. 灏濊瘯浠庡竵瀹塛ebSocket鑾峰彇瀹炴椂鏁版嵁锛堟渶楂樹紭鍏堢骇锛?	if realtimeData, success := s.getRealtimeDataFromWS(symbol, kind); success {
		return realtimeData, true
	}

	// 2. WebSocket澶辫触鏃讹紝闄嶇骇浣跨敤HTTP API缁勫悎鏁版嵁
	var price, change24h, volume24h float64

	// 浣跨敤骞跺彂鑾峰彇浠锋牸鍜?4h鏁版嵁
	var wg sync.WaitGroup
	var priceErr error

	wg.Add(3)

	// 鑾峰彇褰撳墠浠锋牸
	go func() {
		defer wg.Done()
		p, err := s.priceService.GetCurrentPrice(ctx, symbol, kind)
		if err == nil && p > 0 {
			price = p
		} else {
			priceErr = err
		}
	}()

	// 鑾峰彇24灏忔椂娑ㄨ穼骞?	go func() {
		defer wg.Done()
		change24h = s.getPriceChange24hWithKind(symbol, kind)
	}()

	// 鑾峰彇24灏忔椂鎴愪氦閲?	go func() {
		defer wg.Done()
		volume24h = s.getVolume24hWithKind(symbol, kind)
	}()

	wg.Wait()

	// 妫€鏌ユ槸鍚﹁幏鍙栧埌鏈夋晥浠锋牸
	if price <= 0 {
		if priceErr != nil {
			log.Printf("[娑ㄥ箙姒淽 %s 鏃犳硶鑾峰彇浠锋牸: %v", symbol, priceErr)
		} else {
			log.Printf("[娑ㄥ箙姒淽 %s 浠锋牸鏃犳晥: %.4f", symbol, price)
		}
		return RealtimeData{}, false
	}

	// 鑾峰彇鍒嗙被淇℃伅
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

// getRealtimeDataFromWS 浠嶹ebSocket鑾峰彇瀹炴椂鏁版嵁
func (s *Server) getRealtimeDataFromWS(symbol string, kind string) (RealtimeData, bool) {
	if s.binanceWSClient == nil || !s.binanceWSClient.IsConnected() {
		return RealtimeData{}, false
	}

	// 杞崲浜ゆ槗瀵规牸寮忎互鍖归厤WebSocket鏁版嵁
	var wsSymbol string
	switch kind {
	case "futures":
		if strings.HasSuffix(symbol, "USDT") {
			baseSymbol := strings.TrimSuffix(symbol, "USDT")
			wsSymbol = baseSymbol + "USD_PERP"
		} else if strings.HasSuffix(symbol, "USD_PERP") {
			wsSymbol = symbol // 宸茬粡鏄竵鏈綅鏍煎紡
		} else {
			wsSymbol = symbol + "USD_PERP"
		}
	default:
		wsSymbol = symbol + "USDT" // 鐜拌揣缁熶竴浣跨敤USDT鏍煎紡
	}

	// 鑾峰彇WebSocket鏁版嵁
	if ticker, exists := s.binanceWSClient.GetTicker24h(wsSymbol); exists {
		lastPrice, err1 := strconv.ParseFloat(ticker.LastPrice, 64)
		changePercent, err2 := strconv.ParseFloat(ticker.PriceChangePercent, 64)
		volume, err3 := strconv.ParseFloat(ticker.TotalTradedBaseAsset, 64)

		// 鏁版嵁楠岃瘉
		if err1 != nil || err2 != nil || err3 != nil || lastPrice <= 0 {
			log.Printf("[DEBUG] WebSocket鏁版嵁瑙ｆ瀽澶辫触 %s -> %s: price=%s, change=%s, volume=%s",
				symbol, wsSymbol, ticker.LastPrice, ticker.PriceChangePercent, ticker.TotalTradedBaseAsset)
			return RealtimeData{}, false
		}

		// 鑾峰彇鍒嗙被淇℃伅
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

// validateRealtimeData 楠岃瘉瀹炴椂鏁版嵁鐨勮川閲忓拰鍚堢悊鎬?func (s *Server) validateRealtimeData(data RealtimeData) bool {
	// 鍩烘湰鏁版嵁瀹屾暣鎬ф鏌?	if data.Symbol == "" {
		log.Printf("[鏁版嵁楠岃瘉] 缂哄皯浜ゆ槗瀵圭鍙?)
		return false
	}

	// 浜ゆ槗瀵规牸寮忛獙璇?	if !s.isValidSymbolFormat(data.Symbol) {
		log.Printf("[鏁版嵁楠岃瘉] %s 浜ゆ槗瀵规牸寮忔棤鏁?, data.Symbol)
		return false
	}

	// 浠锋牸楠岃瘉
	if data.LastPrice <= 0 {
		log.Printf("[鏁版嵁楠岃瘉] %s 浠锋牸寮傚父: %.4f", data.Symbol, data.LastPrice)
		return false
	}

	// 浠锋牸鍚堢悊鎬ф鏌ワ紙鏍规嵁甯佺绫诲瀷璁剧疆涓嶅悓鐨勯槇鍊硷級
	maxPrice, minPrice := s.getPriceThresholds(data.Symbol)
	if data.LastPrice > maxPrice || data.LastPrice < minPrice {
		log.Printf("[鏁版嵁楠岃瘉] %s 浠锋牸瓒呭嚭鍚堢悊鑼冨洿 [%.8f, %.0f]: %.8f",
			data.Symbol, minPrice, maxPrice, data.LastPrice)
		return false
	}

	// 娑ㄨ穼骞呭悎鐞嗘€ф鏌?	if math.Abs(data.ChangePercent) > 1000 {
		log.Printf("[鏁版嵁楠岃瘉] %s 娑ㄨ穼骞呭紓甯? %.2f%%", data.Symbol, data.ChangePercent)
		return false
	}

	// 鏅鸿兘娑ㄨ穼骞呮鏌ワ紙鍩轰簬鍘嗗彶娉㈠姩鐜囷級
	if math.Abs(data.ChangePercent) > 100 {
		// 瀵逛簬楂樻尝鍔ㄥ竵绉嶆斁瀹介檺鍒?		if !s.isHighVolatilitySymbol(data.Symbol) {
			log.Printf("[鏁版嵁楠岃瘉] %s 娑ㄨ穼骞呰繃楂? %.2f%%", data.Symbol, data.ChangePercent)
			return false
		}
	}

	// 鎴愪氦閲忓悎鐞嗘€ф鏌?	if data.Volume < 0 {
		log.Printf("[鏁版嵁楠岃瘉] %s 鎴愪氦閲忎负璐熸暟: %.2f", data.Symbol, data.Volume)
		return false
	}

	// 鎴愪氦閲忎笅闄愭鏌ワ紙閬垮厤铏氬亣鏁版嵁锛?	minVolume := s.getMinVolumeThreshold(data.Symbol)
	if data.Volume < minVolume {
		//log.Printf("[鏁版嵁楠岃瘉] %s 鎴愪氦閲忚繃浣? %.2f (鏈€浣庤姹? %.2f)",
		//	data.Symbol, data.Volume, minVolume)
		return false
	}

	// 鏃堕棿鎴虫鏌ワ紙涓嶅厑璁歌秴杩?0鍒嗛挓鐨勬暟鎹級
	if data.Timestamp > 0 {
		age := time.Now().Unix() - data.Timestamp
		if age > 1800 { // 30鍒嗛挓
			log.Printf("[鏁版嵁楠岃瘉] %s 鏁版嵁澶棫: %d绉掑墠", data.Symbol, age)
			return false
		}
		if age < -300 { // 涓嶅厑璁告湭鏉?鍒嗛挓鐨勬暟鎹?			log.Printf("[鏁版嵁楠岃瘉] %s 鏃堕棿鎴冲紓甯革紙鏈潵鏃堕棿锛? %d绉掑悗", data.Symbol, -age)
			return false
		}
	}

	// 鏁版嵁婧愭湁鏁堟€ф鏌?	validSources := map[string]bool{
		"websocket":  true,
		"http_api":   true,
		"kline_calc": true,
	}
	if !validSources[data.DataSource] {
		log.Printf("[鏁版嵁楠岃瘉] %s 鏁版嵁婧愭棤鏁? %s", data.Symbol, data.DataSource)
		return false
	}

	return true
}

// getPriceThresholds 鏍规嵁甯佺鑾峰彇浠锋牸鍚堢悊鎬ч槇鍊?func (s *Server) getPriceThresholds(symbol string) (maxPrice, minPrice float64) {
	// BTC鐩稿叧
	if strings.Contains(symbol, "BTC") {
		return 1000000, 0.00000001
	}

	// ETH鐩稿叧
	if strings.Contains(symbol, "ETH") {
		return 100000, 0.0000001
	}

	// 涓绘祦甯佺
	if strings.Contains(symbol, "BNB") || strings.Contains(symbol, "ADA") ||
		strings.Contains(symbol, "XRP") || strings.Contains(symbol, "SOL") ||
		strings.Contains(symbol, "DOT") {
		return 10000, 0.000001
	}

	// 榛樿鍊?	return 100000, 0.00000001
}

// isHighVolatilitySymbol 妫€鏌ユ槸鍚︿负楂樻尝鍔ㄦ€у竵绉?func (s *Server) isHighVolatilitySymbol(symbol string) bool {
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

// getMinVolumeThreshold 鑾峰彇鏈€灏忔垚浜ら噺闃堝€?func (s *Server) getMinVolumeThreshold(symbol string) float64 {
	// 澶у競鍊煎竵绉嶈姹傛洿楂樼殑鎴愪氦閲?	if strings.Contains(symbol, "BTC") || strings.Contains(symbol, "ETH") {
		return 100000 // 10涓囩編閲?	}

	// 涓瓑甯傚€煎竵绉?	if strings.Contains(symbol, "BNB") || strings.Contains(symbol, "ADA") ||
		strings.Contains(symbol, "XRP") || strings.Contains(symbol, "SOL") {
		return 10000 // 1涓囩編閲?	}

	// 灏忓竵绉嶆斁瀹借姹?	return 1000 // 1000缇庨噾
}

// getFallbackRealtimeData 鑾峰彇闄嶇骇瀹炴椂鏁版嵁锛堝綋涓昏鏁版嵁婧愬け璐ユ椂浣跨敤锛?func (s *Server) getFallbackRealtimeData(symbol string, kind string) RealtimeData {
	// 灏濊瘯浠庣紦瀛樹腑鑾峰彇鏈€杩戠殑鏁版嵁
	cacheKey := fmt.Sprintf("price_%s_%s", symbol, kind)
	if cached, exists := s.getCachedPriceData(cacheKey); exists {
		log.Printf("[闄嶇骇鏁版嵁] %s 浣跨敤缂撳瓨浠锋牸鏁版嵁", symbol)
		return cached
	}

	// 濡傛灉缂撳瓨涔熸病鏈夛紝灏濊瘯浠庢暟鎹簱鑾峰彇鏈€杩戠殑鍘嗗彶鏁版嵁
	if historicalData, err := s.getHistoricalPriceData(symbol, kind); err == nil {
		log.Printf("[闄嶇骇鏁版嵁] %s 浣跨敤鍘嗗彶浠锋牸鏁版嵁", symbol)
		return historicalData
	}

	// 濡傛灉閮芥病鏈夛紝杩斿洖绌烘暟鎹?	log.Printf("[闄嶇骇鏁版嵁] %s 鏃犲彲鐢ㄩ檷绾ф暟鎹?, symbol)
	return RealtimeData{
		Symbol:     symbol,
		DataSource: "unavailable",
		Timestamp:  time.Now().Unix(),
	}
}

// getCachedPriceData 浠庣紦瀛樿幏鍙栦环鏍兼暟鎹?func (s *Server) getCachedPriceData(key string) (RealtimeData, bool) {
	// 杩欓噷鍙互瀹炵幇涓€涓畝鍗曠殑浠锋牸缂撳瓨
	// 涓轰簡绠€鍖栵紝鎴戜滑杩斿洖false琛ㄧず娌℃湁缂撳瓨
	return RealtimeData{}, false
}

// getHistoricalPriceData 浠庢暟鎹簱鑾峰彇鍘嗗彶浠锋牸鏁版嵁
func (s *Server) getHistoricalPriceData(symbol string, kind string) (RealtimeData, error) {
	// 灏濊瘯浠庢暟鎹簱鑾峰彇鏈€杩戠殑K绾挎暟鎹綔涓洪檷绾ф暟鎹?	klines, err := s.getKlinesData(symbol, kind, 1, 1) // 鑾峰彇鏈€杩?鏍筀绾?	if err != nil || len(klines) == 0 {
		return RealtimeData{}, fmt.Errorf("no historical data available")
	}

	kline := klines[0]

	// 杞崲瀛楃涓蹭环鏍间负float64
	closePrice, err := strconv.ParseFloat(kline.Close, 64)
	if err != nil {
		return RealtimeData{}, fmt.Errorf("invalid close price: %s", kline.Close)
	}

	volume, err := strconv.ParseFloat(kline.Volume, 64)
	if err != nil {
		return RealtimeData{}, fmt.Errorf("invalid volume: %s", kline.Volume)
	}

	// 璁＄畻24灏忔椂娑ㄨ穼骞咃紙闇€瑕佹洿澶氬巻鍙叉暟鎹紝杩欓噷绠€鍖栧鐞嗭級
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
		Timestamp:     int64(kline.OpenTime / 1000), // 杞崲涓虹
	}, nil
}

// getKlinesData 鑾峰彇K绾挎暟鎹紙绠€鍖栫増锛岀敤浜庨檷绾ф暟鎹幏鍙栵級
func (s *Server) getKlinesData(symbol, kind string, limit, interval int) ([]KlineData, error) {
	// 杩欓噷搴旇璋冪敤瀹為檯鐨凨绾挎暟鎹幏鍙栭€昏緫
	// 涓轰簡绠€鍖栵紝杩斿洖绌虹粨鏋?	return []KlineData{}, fmt.Errorf("kline data not available")
}

// isValidSymbolFormat 楠岃瘉浜ゆ槗瀵规牸寮忔槸鍚︽湁鏁?func (s *Server) isValidSymbolFormat(symbol string) bool {
	if symbol == "" {
		return false
	}

	// 妫€鏌ユ槸鍚﹀寘鍚父瑙佺殑浜ゆ槗瀵规牸寮?	validPatterns := []string{
		"USDT$", "USDC$", "BUSD$", "BTC$", "ETH$", "BNB$", "ADA$", "SOL$", "DOT$",
		"_PERP$", // 鍚堢害鍚庣紑
	}

	for _, pattern := range validPatterns {
		matched, _ := regexp.MatchString(pattern, symbol)
		if matched {
			return true
		}
	}

	// 鍏佽涓€浜涚壒娈婃牸寮忥紙濡傜ǔ瀹氬竵瀵癸級
	if strings.Contains(symbol, "USD") || strings.Contains(symbol, "EUR") {
		return true
	}

	return false
}

// monitorDataQuality 鐩戞帶鏁版嵁璐ㄩ噺鍜屽紓甯告儏鍐?func (s *Server) monitorDataQuality(gainers []gin.H, kind string) {
	if len(gainers) == 0 {
		log.Printf("[鏁版嵁鐩戞帶] 璀﹀憡: %s甯傚満娌℃湁鑾峰彇鍒颁换浣曟定骞呮暟鎹?, kind)
		return
	}

	// 缁熻鍚勭鎸囨爣
	stats := s.calculateDataStats(gainers)

	// 妫€娴嬪紓甯告儏鍐?	warnings := s.detectDataAnomalies(stats, gainers)

	// 鏁版嵁婧愬垎甯冪粺璁?	dataSourceStats := s.calculateDataSourceStats(gainers)

	// 杈撳嚭鐩戞帶缁撴灉
	log.Printf("[鏁版嵁鐩戞帶] %s甯傚満缁熻: 鎬绘暟=%d, 涓婃定=%d, 涓嬭穼=%d, 骞崇洏=%d",
		kind, stats.totalCount, stats.positiveCount, stats.negativeCount, stats.zeroCount)
	log.Printf("[鏁版嵁鐩戞帶] %s甯傚満鎸囨爣: 骞冲潎娑ㄥ箙=%.2f%%, 骞冲潎鎴愪氦閲?%.0f, 娉㈠姩鐜?%.2f%%",
		kind, stats.avgChange, stats.avgVolume, stats.volatility)
	log.Printf("[鏁版嵁鐩戞帶] %s甯傚満鏋佸€? 鏈€楂?.1f%%, 鏈€浣?.1f%%, 鏈€澶ф垚浜ら噺%.0f",
		kind, stats.maxChange, stats.minChange, stats.maxVolume)
	log.Printf("[鏁版嵁鐩戞帶] %s鏁版嵁婧愬垎甯? WebSocket=%d, HTTP_API=%d, K绾胯绠?%d",
		kind, dataSourceStats.websocket, dataSourceStats.httpApi, dataSourceStats.klineCalc)

	if len(warnings) > 0 {
		log.Printf("[鏁版嵁鐩戞帶] %s甯傚満寮傚父妫€娴? %v", kind, warnings)
	}
}

// DataStats 鏁版嵁缁熻缁撴瀯
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

// DataSourceStats 鏁版嵁婧愮粺璁?type DataSourceStats struct {
	websocket int
	httpApi   int
	klineCalc int
}

// calculateDataStats 璁＄畻鏁版嵁缁熻
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

		// 璁＄畻娉㈠姩鐜囷紙鏍囧噯宸級
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

// calculateDataSourceStats 璁＄畻鏁版嵁婧愬垎甯?func (s *Server) calculateDataSourceStats(gainers []gin.H) *DataSourceStats {
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

// detectDataAnomalies 妫€娴嬫暟鎹紓甯?func (s *Server) detectDataAnomalies(stats *DataStats, gainers []gin.H) []string {
	warnings := []string{}

	// 妫€鏌ユ暟鎹垎甯冩槸鍚︽甯?	zeroRatio := float64(stats.zeroCount) / float64(stats.totalCount) * 100
	if zeroRatio > 50 {
		warnings = append(warnings, fmt.Sprintf("瓒呰繃%.1f%%鐨勬暟鎹定骞呬负0", zeroRatio))
	}

	// 妫€鏌ユ槸鍚︽湁鏋佺娑ㄥ箙
	if math.Abs(stats.maxChange) > 100 || math.Abs(stats.minChange) > 100 {
		warnings = append(warnings, fmt.Sprintf("瀛樺湪鏋佺娑ㄥ箙: 鏈€楂?.1f%%, 鏈€浣?.1f%%", stats.maxChange, stats.minChange))
	}

	// 妫€鏌ユ尝鍔ㄧ巼鏄惁寮傚父
	if stats.volatility > 20 {
		warnings = append(warnings, fmt.Sprintf("娉㈠姩鐜囪繃楂? %.2f%%", stats.volatility))
	}

	// 妫€鏌ユ垚浜ら噺鏄惁寮傚父
	if stats.avgVolume < 1000 {
		warnings = append(warnings, fmt.Sprintf("骞冲潎鎴愪氦閲忚繃浣? %.0f", stats.avgVolume))
	}

	// 妫€鏌ユ暟鎹簮鍗曚竴鎬?	if len(gainers) > 10 {
		// 濡傛灉95%浠ヤ笂鐨勬暟鎹潵鑷崟涓€鏁版嵁婧愶紝鍙兘瀛樺湪闂
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
			warnings = append(warnings, fmt.Sprintf("鏁版嵁婧愯繃浜庡崟涓€: %.1f%%鏉ヨ嚜鍚屼竴鏁版嵁婧?, sourceDominance))
		}
	}

	return warnings
}

// 娑ㄥ箙姒滄暟鎹紦瀛?var gainersCache = make(map[string]cachedGainersData)
var gainersCacheMu sync.RWMutex

type cachedGainersData struct {
	data      []gin.H
	expiresAt time.Time
}

// cacheGainers 缂撳瓨娑ㄥ箙姒滄暟鎹?func (s *Server) cacheGainers(key string, data []gin.H) {
	gainersCacheMu.Lock()
	defer gainersCacheMu.Unlock()

	gainersCache[key] = cachedGainersData{
		data:      data,
		expiresAt: time.Now().Add(30 * time.Second),
	}
}

// getCachedGainers 鑾峰彇缂撳瓨鐨勬定骞呮鏁版嵁
func (s *Server) getCachedGainers(key string) ([]gin.H, bool) {
	gainersCacheMu.RLock()
	defer gainersCacheMu.RUnlock()

	if cached, exists := gainersCache[key]; exists && time.Now().Before(cached.expiresAt) {
		return cached.data, true
	}

	return nil, false
}

// getCachedGainersWithDuration 鑾峰彇鎸囧畾鏃堕暱鍐呯殑缂撳瓨鏁版嵁
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

// cacheGainersWithDuration 浣跨敤鎸囧畾鏃堕暱缂撳瓨娑ㄥ箙姒滄暟鎹?func (s *Server) cacheGainersWithDuration(key string, data []gin.H, duration time.Duration) {
	gainersCacheMu.Lock()
	defer gainersCacheMu.Unlock()

	gainersCache[key] = cachedGainersData{
		data:      data,
		expiresAt: time.Now().Add(duration),
	}
}

// filterAndSortGainers 绛涢€夊拰鎺掑簭娑ㄥ箙姒滄暟鎹?func (s *Server) filterAndSortGainers(gainers []gin.H, sortBy, sortOrder string, filterPositiveOnly, filterLargeCap bool, minVolume float64, limit int) []gin.H {
	if len(gainers) == 0 {
		return gainers
	}

	// 搴旂敤绛涢€夋潯浠?	filtered := make([]gin.H, 0, len(gainers))
	for _, gainer := range gainers {
		// 鍙樉绀轰笂娑ㄥ竵绉嶇瓫閫?		if filterPositiveOnly {
			if change, ok := gainer["price_change_24h"].(float64); !ok || change <= 0 {
				continue
			}
		}

		// 澶у競鍊煎竵绉嶇瓫閫?		if filterLargeCap {
			price, priceOk := gainer["current_price"].(float64)
			volume, volumeOk := gainer["volume_24h"].(float64)
			if !priceOk || !volumeOk {
				continue
			}
			// 绠€鍗曠殑甯傚€艰绠楋細浠锋牸 * 鎴愪氦閲?> 100涓?			if price*volume <= 1000000 {
				continue
			}
		}

		// 鏈€灏忔垚浜ら噺绛涢€?		if minVolume > 0 {
			if volume, ok := gainer["volume_24h"].(float64); !ok || volume < minVolume {
				continue
			}
		}

		filtered = append(filtered, gainer)
	}

	// 搴旂敤鎺掑簭
	sort.Slice(filtered, func(i, j int) bool {
		var compareResult bool

		switch sortBy {
		case "volume":
			volI, _ := filtered[i]["volume_24h"].(float64)
			volJ, _ := filtered[j]["volume_24h"].(float64)
			compareResult = volI < volJ // 鍗囧簭锛氬皬鎴愪氦閲忓湪鍓?		case "symbol":
			symI, _ := filtered[i]["symbol"].(string)
			symJ, _ := filtered[j]["symbol"].(string)
			compareResult = symI < symJ // 瀛楀吀搴?		case "change":
		default: // 榛樿鎸夋定骞呮帓搴?			changeI, _ := filtered[i]["price_change_24h"].(float64)
			changeJ, _ := filtered[j]["price_change_24h"].(float64)
			compareResult = changeI < changeJ // 鍗囧簭锛氭定骞呭皬鐨勫湪鍓?		}

		// 鏍规嵁鎺掑簭椤哄簭鍐冲畾鏄惁鍙嶈浆
		if sortOrder == "desc" {
			return !compareResult
		}
		return compareResult
	})

	// 闄愬埗杩斿洖鏁伴噺
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// 閲嶆柊鍒嗛厤鎺掑悕
	for i, gainer := range filtered {
		gainer["rank"] = i + 1
	}

	return filtered
}

// hasSignificantChanges 妫€鏌ユ定骞呮鏁版嵁鏄惁鏈夋樉钁楀彉鍖?func (s *Server) hasSignificantChanges(oldData, newData []gin.H) bool {
	if len(oldData) != len(newData) {
		return true // 鏁伴噺涓嶅悓锛岃偗瀹氭湁鍙樺寲
	}

	// 妫€鏌ュ墠10鍚嶇殑鎺掑悕鎴栦环鏍兼槸鍚︽湁鏄捐憲鍙樺寲
	for i := 0; i < len(oldData) && i < 10; i++ {
		oldGainer := oldData[i]
		newGainer := newData[i]

		// 妫€鏌ユ帓鍚嶅彉鍖?		oldRank, _ := oldGainer["rank"].(int)
		newRank, _ := newGainer["rank"].(int)
		if oldRank != newRank {
			return true
		}

		// 妫€鏌ヤ环鏍煎彉鍖栵紙瓒呰繃0.5%鐨勫彉鍖栫畻鏄捐憲锛?		oldPrice, _ := oldGainer["current_price"].(float64)
		newPrice, _ := newGainer["current_price"].(float64)
		if oldPrice > 0 && math.Abs((newPrice-oldPrice)/oldPrice) > 0.005 {
			return true
		}

		// 妫€鏌ユ定骞呭彉鍖栵紙瓒呰繃0.05%鐨勫彉鍖栫畻鏄捐憲锛?		oldChange, _ := oldGainer["price_change_24h"].(float64)
		newChange, _ := newGainer["price_change_24h"].(float64)
		if math.Abs(newChange-oldChange) > 0.05 {
			return true
		}
	}

	return false // 鏃犳樉钁楀彉鍖?}

// GetRealTimeGainers 鑾峰彇瀹炴椂娑ㄥ箙姒?// GET /market/binance/realtime-gainers?kind=spot&limit=15&sort_by=change&sort_order=desc&filter_positive_only=false&filter_large_cap=false
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
		limit = 15 // 榛樿15涓?	}

	// 瑙ｆ瀽绛涢€夊弬鏁?	filterPositiveOnly := strings.ToLower(filterPositiveOnlyStr) == "true"
	filterLargeCap := strings.ToLower(filterLargeCapStr) == "true"
	var minVolume float64
	if minVolumeStr != "" {
		if mv, err := strconv.ParseFloat(minVolumeStr, 64); err == nil && mv >= 0 {
			minVolume = mv
		}
	}

	// 鑾峰彇鍩虹鏁版嵁锛堣幏鍙栨洿澶氭暟鎹敤浜庣瓫閫夛級
	baseLimit := limit * 10 // 鑾峰彇10鍊嶆暟鎹敤浜庣瓫閫夛紝纭繚鏈夎冻澶熺殑鏁版嵁
	if baseLimit > 500 {
		baseLimit = 500
	}

	// 浼樺厛浣跨敤浼樺寲鐗堟湰锛堢洿鎺ヤ粠 binance_24h_stats 鏌ヨ锛?	gainers, err := s.generateRealtimeGainersFrom24hStats(c.Request.Context(), kind, category, baseLimit)
	if err != nil {
		log.Printf("[娑ㄥ箙姒淽 浼樺寲鐗堟湰澶辫触锛岄檷绾у埌浼犵粺鐗堟湰: %v", err)
		// 闄嶇骇鍒颁紶缁熺増鏈?		gainers, err = s.generateRealtimeGainersData(c.Request.Context(), kind, category, baseLimit)
		if err != nil {
			log.Printf("[ERROR] 浼犵粺鐗堟湰涔熷け璐? %v", err)
			s.InternalServerError(c, "鑾峰彇娑ㄥ箙姒滄暟鎹け璐?, err)
			return
		}
	}
	if err != nil {
		log.Printf("[ERROR] 鐢熸垚瀹炴椂娑ㄥ箙姒滄暟鎹け璐? %v", err)
		s.InternalServerError(c, "鑾峰彇娑ㄥ箙姒滄暟鎹け璐?, err)
		return
	}

	// 搴旂敤绛涢€夊拰鎺掑簭
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

// GetCurrentPriceHTTP 鑾峰彇鎸囧畾甯佺鐨勫綋鍓嶄环鏍?(HTTP handler)
// GET /api/v1/market/price/:symbol?kind=spot
func (s *Server) GetCurrentPriceHTTP(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	kind := c.DefaultQuery("kind", "spot")

	// 鑾峰彇褰撳墠浠锋牸
	price, err := s.getCurrentPrice(c.Request.Context(), symbol, kind)
	if err != nil {
		log.Printf("[ERROR] 鑾峰彇褰撳墠浠锋牸澶辫触 %s: %v", symbol, err)
		c.JSON(500, gin.H{"error": "鑾峰彇浠锋牸澶辫触"})
		return
	}

	c.JSON(200, gin.H{
		"symbol":    symbol,
		"price":     price,
		"timestamp": time.Now().Unix(),
	})
}

// GetBatchCurrentPrices 鎵归噺鑾峰彇褰撳墠浠锋牸
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

	// 闄愬埗鏈€澶ф暟閲?	if len(body.Symbols) > 100 {
		c.JSON(400, gin.H{"error": "too many symbols, maximum 100 allowed"})
		return
	}

	// 鎵归噺鑾峰彇浠锋牸
	prices, err := s.priceService.BatchGetCurrentPrices(c.Request.Context(), body.Symbols, body.Kind)
	if err != nil {
		log.Printf("[ERROR] 鎵归噺鑾峰彇浠锋牸澶辫触: %v", err)
		c.JSON(500, gin.H{"error": "鎵归噺鑾峰彇浠锋牸澶辫触"})
		return
	}

	// 杞崲涓哄墠绔渶瑕佺殑鏍煎紡
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

// GetKlines 鑾峰彇K绾挎暟鎹?// GET /api/v1/market/klines/:symbol?interval=1h&limit=100&kind=spot&aggregate=4h
func (s *Server) GetKlines(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	interval := c.DefaultQuery("interval", "1h")
	limitStr := c.DefaultQuery("limit", "100")
	kind := c.DefaultQuery("kind", "spot")
	aggregate := c.Query("aggregate") // 鍙€夌殑鑱氬悎鐩爣闂撮殧

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100 // 榛樿100鏉?	}

	// 鑾峰彇K绾挎暟鎹紙浣跨敤缂撳瓨鏈哄埗鍜屾暟鎹獙璇侊級
	klines, err := s.getKlinesWithCache(c.Request.Context(), symbol, kind, interval, limit)
	if err != nil {
		log.Printf("[ERROR] 鑾峰彇K绾挎暟鎹け璐?%s: %v", symbol, err)
		c.JSON(500, gin.H{"error": "鑾峰彇K绾挎暟鎹け璐?})
		return
	}

	// 濡傛灉鎸囧畾浜嗚仛鍚堥棿闅旓紝杩涜鏁版嵁鑱氬悎
	if aggregate != "" && aggregate != interval {
		aggregatedKlines, err := s.ConvertKlineInterval(klines, interval, aggregate, symbol, kind)
		if err != nil {
			log.Printf("[WARNING] K绾挎暟鎹仛鍚堝け璐?%s: %v", symbol, err)
			// 鑱氬悎澶辫触鏃朵娇鐢ㄥ師濮嬫暟鎹?		} else {
			klines = aggregatedKlines
			interval = aggregate // 鏇存柊杩斿洖鐨勯棿闅斾俊鎭?			log.Printf("[KlineAggregation] 鑱氬悎鎴愬姛: %s %s 鈫?%d 鏉?, symbol, aggregate, len(klines))
		}
	}

	// 鑾峰彇楠岃瘉鍜屽鐞嗗悗鐨勬暟鎹?	validatedKlines, err := s.ValidateAndCleanKlines(klines, symbol, interval, kind)
	if err != nil {
		log.Printf("[WARNING] K绾挎暟鎹獙璇佸け璐?%s: %v", symbol, err)
		// 鍗充娇楠岃瘉澶辫触锛屼篃杩斿洖鍘熷鏁版嵁
	}

	log.Printf("[DEBUG] 鑾峰彇鍒癒绾挎暟鎹? symbol=%s, interval=%s, count=%d", symbol, interval, len(klines))

	// 杞崲涓哄墠绔渶瑕佺殑鏍煎紡锛屽寘鍚暟鎹川閲忎俊鎭?	result := make([]gin.H, len(klines))
	for i, kline := range klines {
		klineData := gin.H{
			"timestamp": kline.OpenTime,
			"open":      kline.Open,
			"high":      kline.High,
			"low":       kline.Low,
			"close":     kline.Close,
			"volume":    kline.Volume,
		}

		// 濡傛灉鏈夐獙璇佹暟鎹紝娣诲姞棰濆淇℃伅
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

	// 濡傛灉杩涜浜嗚仛鍚堬紝娣诲姞鑱氬悎淇℃伅
	if aggregate != "" && aggregate != c.DefaultQuery("interval", "1h") {
		response["aggregated"] = true
		response["original_interval"] = c.DefaultQuery("interval", "1h")
	}

	c.JSON(200, response)
}

// GetRecommendationPerformance 鑾峰彇鎺ㄨ崘鍘嗗彶琛ㄧ幇
// GET /api/v1/recommend/performance/:symbol?period=30d
func (s *Server) GetRecommendationPerformance(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	period := c.DefaultQuery("period", "30d")

	// 瑙ｆ瀽鏃堕棿鍛ㄦ湡
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

	// 浠庢暟鎹簱鑾峰彇璇ymbol鐨勫巻鍙叉€ц兘鏁版嵁
	performances, err := pdb.GetPerformanceBySymbol(s.db.DB(), symbol, 1000) // 鑾峰彇鏈€杩?000鏉¤褰?	if err != nil {
		log.Printf("[ERROR] 鑾峰彇%s鍘嗗彶鎬ц兘鏁版嵁澶辫触: %v", symbol, err)
		c.JSON(500, gin.H{"error": "鑾峰彇鍘嗗彶鏁版嵁澶辫触"})
		return
	}

	log.Printf("[DEBUG] Found %d total performances for %s", len(performances), symbol)

	// 杩囨护鎸囧畾鏃堕棿鍛ㄦ湡鍐呯殑鏁版嵁
	now := time.Now().UTC()
	cutoffTime := now.AddDate(0, 0, -days)
	var filteredPerformances []pdb.RecommendationPerformance

	for _, perf := range performances {
		if perf.RecommendedAt.After(cutoffTime) {
			filteredPerformances = append(filteredPerformances, perf)
		}
	}

	log.Printf("[DEBUG] Found %d performances within %d days for %s", len(filteredPerformances), days, symbol)

	// 璁＄畻鍩轰簬鐪熷疄鏁版嵁鐨勭粺璁℃寚鏍?	stats := s.calculateRealPerformanceStats(filteredPerformances, symbol, period, days)

	// 濡傛灉娌℃湁鍘嗗彶鏁版嵁锛屽熀浜庢妧鏈寚鏍囩敓鎴愭洿鐪熷疄鐨勬ā鎷熸暟鎹?	if len(filteredPerformances) == 0 {
		log.Printf("[INFO] No historical data for %s, generating realistic simulated data based on technical indicators", symbol)
		stats = s.generateRealisticSimulatedStats(symbol, period, days)
	}

	// 鑾峰彇褰撳墠浠锋牸
	currentPrice := 45000.0 // 榛樿浠锋牸
	if price, err := s.getCurrentPrice(c.Request.Context(), symbol, "spot"); err == nil && price > 0 {
		currentPrice = price
	}

	// 鏋勫缓瀹屾暣鐨勬€ц兘鏁版嵁鍝嶅簲
	performance := gin.H{
		"symbol":            symbol,
		"period":            period,
		"overall_score":     stats.OverallScore,
		"technical_score":   stats.TechnicalScore,
		"fundamental_score": stats.FundamentalScore,
		"sentiment_score":   stats.SentimentScore,
		"momentum_score":    stats.MomentumScore,

		// 鏀剁泭鐩稿叧鍥犲瓙
		"return_factor":      stats.ReturnFactor,
		"risk_factor":        stats.RiskFactor,
		"consistency_factor": stats.ConsistencyFactor,
		"timing_factor":      stats.TimingFactor,

		// 浼犵粺鎬ц兘鎸囨爣
		"total_return":      stats.TotalReturn,
		"annualized_return": stats.AnnualizedReturn,
		"max_drawdown":      stats.MaxDrawdown,
		"sharpe_ratio":      stats.SharpeRatio,
		"win_rate":          stats.WinRate,
		"profit_factor":     stats.ProfitFactor,

		// 椋庨櫓鎸囨爣
		"volatility":         stats.Volatility,
		"var_95":             stats.VaR95,
		"expected_shortfall": stats.ExpectedShortfall,

		// 甯傚満鏁版嵁
		"current_price":    currentPrice,
		"price_change_24h": s.getPriceChange24h(symbol),
		"volume_24h":       s.getVolume24h(symbol),

		// 棰濆鐨勬€ц兘缁熻鏁版嵁
		"accuracy":              stats.Accuracy,
		"avg_return":            stats.AvgReturn,
		"avg_holding_time":      stats.AvgHoldingTime,
		"total_recommendations": stats.TotalRecommendations,
		"best_monthly_return":   stats.BestMonthlyReturn,
		"worst_monthly_return":  stats.WorstMonthlyReturn,

		// 鏃堕棿鎴?		"timestamp":     time.Now().Unix(),
		"calculated_at": time.Now().Format(time.RFC3339),
	}

	c.JSON(200, gin.H{
		"success":     true,
		"performance": performance,
		"period":      period,
		"timestamp":   time.Now().Unix(),
	})
}

// PerformanceStats 鍩轰簬鐪熷疄鏁版嵁鐨勬€ц兘缁熻
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

// calculateRealPerformanceStats 鍩轰簬鐪熷疄鍘嗗彶鏁版嵁璁＄畻鎬ц兘缁熻
func (s *Server) calculateRealPerformanceStats(performances []pdb.RecommendationPerformance, symbol, period string, days int) *PerformanceStats {
	stats := &PerformanceStats{}

	if len(performances) == 0 {
		// 濡傛灉娌℃湁鍘嗗彶鏁版嵁锛岃繑鍥為粯璁ゅ€?		stats.OverallScore = 5.0
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
		stats.AvgHoldingTime = "3.0澶?
		stats.TotalRecommendations = 0
		stats.BestMonthlyReturn = 5.0
		stats.WorstMonthlyReturn = -5.0
		return stats
	}

	// 璁＄畻鍩虹缁熻
	totalRecords := len(performances)
	stats.TotalRecommendations = totalRecords

	// 璁＄畻鑳滅巼鍜屽噯纭巼
	winCount := 0
	totalReturn := 0.0
	returns := make([]float64, 0, totalRecords)
	holdingPeriods := make([]int, 0)

	for _, perf := range performances {
		// 璁＄畻鑳滅巼锛堝熀浜?4灏忔椂鏀剁泭鐜囷級
		if perf.Return24h != nil && *perf.Return24h > 0 {
			winCount++
		}

		// 鏀堕泦鏀剁泭鐜囨暟鎹?		if perf.Return24h != nil {
			returns = append(returns, *perf.Return24h)
			totalReturn += *perf.Return24h
		}

		// 鏀堕泦鎸佷粨鍛ㄦ湡鏁版嵁
		if perf.HoldingPeriod != nil && *perf.HoldingPeriod > 0 {
			holdingPeriods = append(holdingPeriods, *perf.HoldingPeriod)
		}
	}

	// 璁＄畻鑳滅巼
	if totalRecords > 0 {
		stats.WinRate = float64(winCount) / float64(totalRecords) * 100
		stats.Accuracy = stats.WinRate // 鍑嗙‘鐜囩瓑浜庤儨鐜?	}

	// 璁＄畻骞冲潎鏀剁泭鐜?	if len(returns) > 0 {
		stats.AvgReturn = totalReturn / float64(len(returns))
		stats.TotalReturn = stats.AvgReturn * float64(days) / 30.0 // 鎸夊懆鏈熻皟鏁?	}

	// 璁＄畻骞村寲鏀剁泭鐜?	if days > 0 {
		stats.AnnualizedReturn = stats.TotalReturn * 365.0 / float64(days)
	}

	// 璁＄畻骞冲潎鎸佷粨鏃堕棿
	if len(holdingPeriods) > 0 {
		totalMinutes := 0
		for _, period := range holdingPeriods {
			totalMinutes += period
		}
		avgMinutes := float64(totalMinutes) / float64(len(holdingPeriods))
		avgHours := avgMinutes / 60.0
		avgDays := avgHours / 24.0
		stats.AvgHoldingTime = fmt.Sprintf("%.1f澶?, avgDays)
	} else {
		stats.AvgHoldingTime = "3.0澶?
	}

	// 璁＄畻鏈€浣冲拰鏈€宸湀搴︽敹鐩婏紙杩欓噷绠€鍖栦负鏈€浣冲拰鏈€宸崟娆℃敹鐩婏級
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

	// 璁＄畻娉㈠姩鐜囷紙鏀剁泭鐜囩殑鏍囧噯宸級
	if len(returns) > 1 {
		mean := stats.AvgReturn
		sumSquares := 0.0
		for _, ret := range returns {
			diff := ret - mean
			sumSquares += diff * diff
		}
		variance := sumSquares / float64(len(returns)-1)
		stats.Volatility = math.Sqrt(variance)

		// 璁＄畻VaR95鍜孍xpected Shortfall
		stats.VaR95 = -stats.Volatility * 1.645           // 95%缃俊鍖洪棿
		stats.ExpectedShortfall = -stats.Volatility * 2.0 // 绠€鍖栫殑棰勬湡鐭己
	} else {
		stats.Volatility = 10.0
		stats.VaR95 = 8.0
		stats.ExpectedShortfall = 10.0
	}

	// 璁＄畻鏈€澶у洖鎾わ紙绠€鍖栦负娉㈠姩鐜囩殑鍊嶆暟锛?	stats.MaxDrawdown = stats.Volatility * 2.0

	// 璁＄畻澶忔櫘姣旂巼
	if stats.Volatility > 0 {
		stats.SharpeRatio = stats.AvgReturn / stats.Volatility
	} else {
		stats.SharpeRatio = 1.0
	}

	// 璁＄畻鍒╂鼎鍥犲瓙
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

	// 鍩轰簬鍘嗗彶鏁版嵁璁＄畻璇勫垎鍥犲瓙
	// 鏀剁泭鍥犲瓙锛氬熀浜庡钩鍧囨敹鐩婄巼
	stats.ReturnFactor = math.Min(10.0, math.Max(0.0, (stats.AvgReturn+10.0)*0.5))

	// 椋庨櫓鍥犲瓙锛氬熀浜庢尝鍔ㄧ巼鍜屾渶澶у洖鎾ょ殑鍙嶅悜
	riskScore := 10.0 - math.Min(10.0, stats.Volatility*0.5+stats.MaxDrawdown*0.3)
	stats.RiskFactor = math.Max(0.0, riskScore)

	// 涓€鑷存€у洜瀛愶細鍩轰簬鑳滅巼
	stats.ConsistencyFactor = stats.WinRate * 0.1

	// 鏃舵満鎶婃彙鍥犲瓙锛氬熀浜庡鏅瘮鐜?	stats.TimingFactor = math.Min(10.0, math.Max(0.0, stats.SharpeRatio*2.0))

	// 璁＄畻缁煎悎璇勫垎
	stats.OverallScore = (stats.ReturnFactor*0.4 + stats.RiskFactor*0.3 + stats.ConsistencyFactor*0.2 + stats.TimingFactor*0.1)
	stats.OverallScore = math.Round(stats.OverallScore*100) / 100

	// 鍏朵粬璇勫垎锛堟殏鏃朵娇鐢ㄩ粯璁ゅ€硷紝鏈潵鍙互鍩轰簬鍘嗗彶鏁版嵁鏀硅繘锛?	stats.TechnicalScore = 6.0
	stats.FundamentalScore = 5.5
	stats.SentimentScore = 5.8
	stats.MomentumScore = 6.2

	// 鍥涜垗浜斿叆鎵€鏈夋暟鍊?	stats.ReturnFactor = math.Round(stats.ReturnFactor*10) / 10
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

// generateRealisticSimulatedStats 鍩轰簬鎶€鏈寚鏍囩敓鎴愭洿鐪熷疄鐨勬ā鎷熸暟鎹?func (s *Server) generateRealisticSimulatedStats(symbol, period string, days int) *PerformanceStats {
	// 鑾峰彇鎶€鏈寚鏍囨暟鎹?	multiIndicators, err := s.GetMultiTimeframeIndicators(context.Background(), symbol, "spot")
	if err != nil {
		log.Printf("[WARN] 鑾峰彇鎶€鏈寚鏍囧け璐ワ紝浣跨敤鍩虹妯℃嫙鏁版嵁: %v", err)
		// 杩斿洖鍩虹榛樿鍊?		return &PerformanceStats{
			OverallScore: 5.0, TechnicalScore: 5.0, FundamentalScore: 5.0, SentimentScore: 5.0, MomentumScore: 5.0,
			ReturnFactor: 5.0, RiskFactor: 3.0, ConsistencyFactor: 5.0, TimingFactor: 5.0,
			TotalReturn: 0.0, AnnualizedReturn: 0.0, MaxDrawdown: 5.0, SharpeRatio: 1.0,
			WinRate: 50.0, ProfitFactor: 1.0, Volatility: 10.0, VaR95: 8.0, ExpectedShortfall: 10.0,
			Accuracy: 50.0, AvgReturn: 0.0, AvgHoldingTime: "3.0澶?, TotalRecommendations: 25,
			BestMonthlyReturn: 5.0, WorstMonthlyReturn: -5.0,
		}
	}

	// 鍩轰簬鎶€鏈寚鏍囪绠楀悇绉嶈瘎鍒?	technicalScore := s.calculateTechnicalScore(multiIndicators)
	fundamentalScore := s.calculateFundamentalScore(symbol)
	sentimentScore := 0.7 // 榛樿鎯呯华寰楀垎
	momentumScore := s.calculateMomentumScore(multiIndicators)

	// 鏍规嵁symbol璋冩暣鍩虹鏁版嵁
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

	// 鏍规嵁鏃堕棿鍛ㄦ湡璋冩暣
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

	// 鐢熸垚鍩轰簬鎶€鏈寚鏍囩殑鐪熷疄鎰熸暟鎹?	stats := &PerformanceStats{}

	// 璇勫垎绯荤粺 - 鍩轰簬鎶€鏈寚鏍?	stats.TechnicalScore = math.Round(technicalScore*10*100) / 100
	stats.FundamentalScore = math.Round(fundamentalScore*10*100) / 100
	stats.SentimentScore = math.Round(sentimentScore*10*100) / 100
	stats.MomentumScore = math.Round(momentumScore*10*100) / 100

	// 缁煎悎璇勫垎
	stats.OverallScore = math.Round((stats.TechnicalScore*0.4+stats.FundamentalScore*0.3+
		stats.SentimentScore*0.2+stats.MomentumScore*0.1)*100) / 100

	// 鏀剁泭鍥犲瓙 - 鍩轰簬鍔ㄩ噺鍜屾妧鏈寚鏍?	stats.ReturnFactor = math.Round((technicalScore*0.6+momentumScore*0.4)*10*10) / 10

	// 椋庨櫓鍥犲瓙 - 鎶€鏈寚鏍囩殑鍙嶅悜
	stats.RiskFactor = math.Round((1.0-technicalScore)*6*10) / 10

	// 涓€鑷存€у洜瀛?- 鍩轰簬鎶€鏈寚鏍囩ǔ瀹氭€?	stats.ConsistencyFactor = math.Round(technicalScore*8*10) / 10

	// 鏃舵満鍥犲瓙 - 鍩轰簬鍔ㄩ噺
	stats.TimingFactor = math.Round(momentumScore*8*10) / 10

	// 浼犵粺鎬ц兘鎸囨爣
	baseReturn := (technicalScore*0.4 + momentumScore*0.3 + fundamentalScore*0.3) * baseMultiplier * periodMultiplier
	stats.TotalReturn = math.Round(baseReturn*15*100) / 100
	stats.AnnualizedReturn = math.Round((baseReturn*12+2)*100) / 100
	stats.MaxDrawdown = math.Round((1.0-technicalScore)*8*100) / 100
	riskFactor := 1.0 - technicalScore
	stats.SharpeRatio = math.Round((technicalScore/riskFactor*1.5+0.5)*100) / 100
	stats.WinRate = math.Round((technicalScore*30+50)*100) / 100
	stats.ProfitFactor = math.Round((technicalScore*1.5+1.0)*100) / 100

	// 椋庨櫓鎸囨爣
	stats.Volatility = math.Round((1.0-technicalScore)*15+5*100) / 100
	stats.VaR95 = math.Round(stats.Volatility*0.8*100) / 100
	stats.ExpectedShortfall = math.Round(stats.Volatility*1.2*100) / 100

	// 棰濆鐨勭粺璁℃暟鎹?	stats.Accuracy = stats.WinRate
	stats.AvgReturn = math.Round(baseReturn*8*100) / 100
	stats.AvgHoldingTime = fmt.Sprintf("%.1f澶?, (technicalScore*5 + 2))
	stats.TotalRecommendations = int(math.Round((technicalScore*50 + 10) * baseMultiplier))
	stats.BestMonthlyReturn = math.Round((baseReturn*12+5)*100) / 100
	stats.WorstMonthlyReturn = math.Round((baseReturn*(-8)-3)*100) / 100

	return stats
}

// GetSentimentAnalysis 鑾峰彇鎯呯华鍒嗘瀽鏁版嵁
// GET /api/v1/sentiment/:symbol
func (s *Server) GetSentimentAnalysis(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(400, gin.H{"error": "symbol parameter is required"})
		return
	}

	// 璋冪敤鎯呯华鍒嗘瀽
	result, err := s.getSentimentAnalysis(c.Request.Context(), symbol)
	if err != nil {
		log.Printf("[ERROR] 鑾峰彇鎯呯华鍒嗘瀽澶辫触 %s: %v", symbol, err)
		c.JSON(500, gin.H{"error": "鑾峰彇鎯呯华鍒嗘瀽澶辫触"})
		return
	}

	c.JSON(200, gin.H{
		"symbol":    symbol,
		"sentiment": result,
		"timestamp": time.Now().Unix(),
	})
}

// GetAvailableSymbols 鑾峰彇鍙敤鐨勪氦鏄撳鍒楄〃
// GET /api/v1/market/symbols?kind=spot&limit=50
func (s *Server) GetAvailableSymbols(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.DefaultQuery("kind", "spot")))
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50 // 榛樿50涓?	}

	// 鑾峰彇甯佺鍒楄〃
	symbols, err := s.getAvailableSymbols(c.Request.Context(), kind, limit)
	if err != nil {
		log.Printf("[ERROR] 鑾峰彇鍙敤甯佺鍒楄〃澶辫触: %v", err)
		c.JSON(500, gin.H{"error": "鑾峰彇甯佺鍒楄〃澶辫触"})
		return
	}

	c.JSON(200, gin.H{
		"symbols": symbols,
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
		limit = 50 // 榛樿50涓?	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1 // 榛樿绗?椤?	}

	// 鑾峰彇鍖呭惈甯傚€间俊鎭殑甯佺鍒楄〃锛堟敮鎸佸垎椤碉級
	symbolsData, totalCount, err := s.getSymbolsWithMarketCapPaged(c.Request.Context(), kind, limit, page)
	if err != nil {
		log.Printf("[ERROR] 鑾峰彇甯﹀競鍊间俊鎭殑甯佺鍒楄〃澶辫触: %v", err)
		c.JSON(500, gin.H{"error": "鑾峰彇甯佺鍒楄〃澶辫触"})
		return
	}

	c.JSON(200, gin.H{
		"symbols":    symbolsData,
		"count":      len(symbolsData),
		"total":      totalCount,
		"page":       page,
		"limit":      limit,
		"totalPages": (totalCount + limit - 1) / limit, // 鍚戜笂鍙栨暣璁＄畻鎬婚〉鏁?		"kind":       kind,
	})
}

// getSymbolsWithMarketCap 鑾峰彇鍖呭惈甯傚€间俊鎭殑甯佺鍒楄〃锛堜紭鍖栦负涓€娆℃煡璇級
func (s *Server) getSymbolsWithMarketCap(ctx context.Context, kind string, limit int) ([]gin.H, error) {
	// 棣栧厛鑾峰彇甯佸畨鍙敤鐨勪氦鏄撳鍒楄〃
	availableSymbols, err := s.getAvailableSymbols(ctx, kind, 10000) // 鑾峰彇瓒冲澶氱殑鍙敤甯佺
	if err != nil {
		log.Printf("[WARN] 鑾峰彇甯佸畨鍙敤浜ゆ槗瀵瑰け璐? %v锛屽皢杩斿洖绌哄垪琛?, err)
		return []gin.H{}, nil
	}

	if len(availableSymbols) == 0 {
		log.Printf("[INFO] 娌℃湁鎵惧埌甯佸畨鍙敤鐨勪氦鏄撳鏁版嵁")
		return []gin.H{}, nil
	}

	// 灏嗗甫鍚庣紑鐨勫竵瀹変氦鏄撳杞崲涓轰笉甯﹀悗缂€鐨勫竵绉嶇鍙凤紙鐢ㄤ簬鍖归厤CoinCap鏁版嵁锛?	var coinCapSymbols []string
	for _, symbol := range availableSymbols {
		// 鍘绘帀甯歌鐨勪氦鏄撳鍚庣紑
		coinCapSymbol := s.normalizeBinanceSymbolToCoinCap(symbol)
		if coinCapSymbol != "" {
			coinCapSymbols = append(coinCapSymbols, coinCapSymbol)
		}
	}

	if len(coinCapSymbols) == 0 {
		log.Printf("[INFO] 杞崲鍚庢病鏈夋湁鏁堢殑CoinCap甯佺绗﹀彿")
		return []gin.H{}, nil
	}

	// 鍒涘缓甯傚€兼暟鎹湇鍔?	marketDataService := pdb.NewCoinCapMarketDataService(s.db.DB())

	// 涓€娆℃€ц幏鍙栧競鍊煎皬浜?000涓囦笖甯佸畨鏀寔鐨勫畬鏁存暟鎹?	dataList, err := marketDataService.GetMarketDataByMarketCapRangeAndSymbols(ctx, 0, 50000000, coinCapSymbols, limit*2) // 鑾峰彇鏇村鏁版嵁鐢ㄤ簬绛涢€?	if err != nil {
		log.Printf("[WARN] 鏌ヨ甯傚€艰寖鍥村唴涓斿竵瀹夋敮鎸佺殑甯佺鏁版嵁澶辫触: %v", err)
		return []gin.H{}, nil
	}

	// 濡傛灉娌℃湁鎵惧埌绗﹀悎鏉′欢鐨勫竵绉嶏紝杩斿洖绌哄垪琛?	if len(dataList) == 0 {
		log.Printf("[INFO] 娌℃湁鎵惧埌甯傚€?5000涓囩殑甯佺锛孋oinCap鏁版嵁鍙兘杩樻湭鍚屾锛岃杩愯: go run cmd/coincap_sync/main.go -action=market-data")
		return []gin.H{}, nil
	}

	// 棰勯獙璇佸竵绉嶅疄鏃舵暟鎹彲鐢ㄦ€э紝鍙繑鍥炶兘鑾峰彇鍒板疄鏃舵暟鎹殑甯佺
	var validatedSymbolsData []gin.H
	validationTimeout := 5 * time.Second
	validationCtx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()

	log.Printf("[INFO] 寮€濮嬮獙璇?%d 涓竵绉嶇殑瀹炴椂鏁版嵁鍙敤鎬?..", len(dataList))

	for _, data := range dataList {
		// 鏋勯€犲竵瀹変氦鏄撳鏍煎紡鐢ㄤ簬楠岃瘉
		binanceSymbol := data.Symbol + "USDT"
		if kind == "futures" {
			binanceSymbol = data.Symbol + "USDT" // 鍚堢害涔熶娇鐢║SDT鏍煎紡楠岃瘉
		}

		// 楠岃瘉鏄惁鑳借幏鍙栧埌瀹炴椂鏁版嵁
		_, success := s.getRealtimeDataConcurrently(validationCtx, binanceSymbol, kind)
		if success {
			// 瑙ｆ瀽瀛楃涓蹭负float64浠ヤ究鍓嶇浣跨敤
			price, err := strconv.ParseFloat(data.PriceUSD, 64)
			if err != nil {
				log.Printf("[WARN] 瑙ｆ瀽浠锋牸澶辫触 %s: %s", data.Symbol, data.PriceUSD)
				price = 0
			}

			changePercent, err := strconv.ParseFloat(data.Change24Hr, 64)
			if err != nil {
				log.Printf("[WARN] 瑙ｆ瀽娑ㄨ穼骞呭け璐?%s: %s", data.Symbol, data.Change24Hr)
				changePercent = 0
			}

			volume, err := strconv.ParseFloat(data.Volume24Hr, 64)
			if err != nil {
				log.Printf("[WARN] 瑙ｆ瀽鎴愪氦閲忓け璐?%s: %s", data.Symbol, data.Volume24Hr)
				volume = 0
			}

			marketCap, err := strconv.ParseFloat(data.MarketCapUSD, 64)
			if err != nil {
				log.Printf("[WARN] 瑙ｆ瀽甯傚€煎け璐?%s: %s (鍘熷鏁版嵁: %s)", data.Symbol, data.MarketCapUSD, data.MarketCapUSD)
				marketCap = 0
			} else if data.MarketCapUSD == "" {
				log.Printf("[WARN] 甯傚€间负绌哄瓧绗︿覆 %s", data.Symbol)
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

			// 杈惧埌闄愬埗鏁伴噺鏃跺仠姝?			if len(validatedSymbolsData) >= limit {
				break
			}
		} else {
			log.Printf("[INFO] 甯佺 %s 鏃犳硶鑾峰彇瀹炴椂鏁版嵁锛屽凡杩囨护", data.Symbol)
		}
	}

	if len(validatedSymbolsData) == 0 {
		log.Printf("[INFO] 楠岃瘉鍚庢病鏈夋湁鏁堢殑瀹炴椂鏁版嵁甯佺")
		return []gin.H{}, nil
	}

	log.Printf("[INFO] 楠岃瘉瀹屾垚锛岃繑鍥?%d 涓湁瀹炴椂鏁版嵁鐨勫競鍊?5000涓囩殑甯佺", len(validatedSymbolsData))
	return validatedSymbolsData, nil
}

// getSymbolsWithMarketCapPaged 鑾峰彇鍖呭惈甯傚€间俊鎭殑甯佺鍒楄〃锛堟敮鎸佸垎椤碉級
func (s *Server) getSymbolsWithMarketCapPaged(ctx context.Context, kind string, limit int, page int) ([]gin.H, int, error) {
	// 棣栧厛鑾峰彇甯佸畨鍙敤鐨勪氦鏄撳鍒楄〃
	availableSymbols, err := s.getAvailableSymbols(ctx, kind, 10000) // 鑾峰彇瓒冲澶氱殑鍙敤甯佺
	if err != nil {
		log.Printf("[WARN] 鑾峰彇甯佸畨鍙敤浜ゆ槗瀵瑰け璐? %v锛屽皢杩斿洖绌哄垪琛?, err)
		return []gin.H{}, 0, nil
	}

	if len(availableSymbols) == 0 {
		log.Printf("[INFO] 娌℃湁鎵惧埌甯佸畨鍙敤鐨勪氦鏄撳鏁版嵁")
		return []gin.H{}, 0, nil
	}

	// 灏嗗甫鍚庣紑鐨勫竵瀹変氦鏄撳杞崲涓轰笉甯﹀悗缂€鐨勫竵绉嶇鍙凤紙鐢ㄤ簬鍖归厤CoinCap鏁版嵁锛?	var coinCapSymbols []string
	for _, symbol := range availableSymbols {
		// 鍘绘帀甯歌鐨勪氦鏄撳鍚庣紑
		coinCapSymbol := s.normalizeBinanceSymbolToCoinCap(symbol)
		if coinCapSymbol != "" {
			coinCapSymbols = append(coinCapSymbols, coinCapSymbol)
		}
	}

	if len(coinCapSymbols) == 0 {
		log.Printf("[INFO] 杞崲鍚庢病鏈夋湁鏁堢殑CoinCap甯佺绗﹀彿")
		return []gin.H{}, 0, nil
	}

	// 鍒涘缓甯傚€兼暟鎹湇鍔?	marketDataService := pdb.NewCoinCapMarketDataService(s.db.DB())

	// 璁＄畻鍋忕Щ閲?	offset := (page - 1) * limit

	// 鑾峰彇鎬绘暟锛堝彧缁熻甯佸畨鏀寔鐨勫竵绉嶏級
	totalCountInt64, err := marketDataService.GetMarketDataCountByMarketCapRangeAndSymbols(ctx, 0, 50000000, coinCapSymbols)
	if err != nil {
		log.Printf("[WARN] 鏌ヨ甯傚€艰寖鍥村唴涓斿竵瀹夋敮鎸佺殑甯佺鎬绘暟澶辫触: %v", err)
		totalCountInt64 = 0
	}
	totalCount := int(totalCountInt64)

	// 鑾峰彇鍒嗛〉鏁版嵁锛堝彧鑾峰彇甯佸畨鏀寔鐨勫竵绉嶏級
	dataList, err := marketDataService.GetMarketDataByMarketCapRangeAndSymbolsPaged(ctx, 0, 50000000, coinCapSymbols, limit, offset)
	if err != nil {
		log.Printf("[WARN] 鏌ヨ甯傚€艰寖鍥村唴涓斿竵瀹夋敮鎸佺殑鍒嗛〉甯佺鏁版嵁澶辫触: %v", err)
		return []gin.H{}, totalCount, nil
	}

	// 濡傛灉娌℃湁鎵惧埌绗﹀悎鏉′欢鐨勫竵绉嶏紝杩斿洖绌哄垪琛?	if len(dataList) == 0 {
		log.Printf("[INFO] 绗?d椤垫病鏈夋壘鍒板競鍊?5000涓囩殑甯佺鏁版嵁", page)
		return []gin.H{}, totalCount, nil
	}

	// 杞崲涓哄墠绔渶瑕佺殑鏍煎紡
	var symbolsData []gin.H
	for _, data := range dataList {
		// 瑙ｆ瀽瀛楃涓蹭负float64浠ヤ究鍓嶇浣跨敤
		price, err := strconv.ParseFloat(data.PriceUSD, 64)
		if err != nil {
			log.Printf("[WARN] 瑙ｆ瀽浠锋牸澶辫触 %s: %s", data.Symbol, data.PriceUSD)
			price = 0
		}

		changePercent, err := strconv.ParseFloat(data.Change24Hr, 64)
		if err != nil {
			log.Printf("[WARN] 瑙ｆ瀽娑ㄨ穼骞呭け璐?%s: %s", data.Symbol, data.Change24Hr)
			changePercent = 0
		}

		volume, err := strconv.ParseFloat(data.Volume24Hr, 64)
		if err != nil {
			log.Printf("[WARN] 瑙ｆ瀽鎴愪氦閲忓け璐?%s: %s", data.Symbol, data.Volume24Hr)
			volume = 0
		}

		marketCap, err := strconv.ParseFloat(data.MarketCapUSD, 64)
		if err != nil {
			log.Printf("[WARN] 瑙ｆ瀽甯傚€煎け璐?%s: %s (鍘熷鏁版嵁: %s)", data.Symbol, data.MarketCapUSD, data.MarketCapUSD)
			marketCap = 0
		} else if data.MarketCapUSD == "" {
			log.Printf("[WARN] 甯傚€间负绌哄瓧绗︿覆 %s", data.Symbol)
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

	log.Printf("[INFO] 杩斿洖绗?d椤?%d 涓競鍊?5000涓囩殑甯佺鏁版嵁 (鎬绘暟: %d)", page, len(symbolsData), totalCount)
	return symbolsData, totalCount, nil
}

// getAvailableSymbols 鑾峰彇鍙敤鐨勪氦鏄撳鍒楄〃
func (s *Server) getAvailableSymbols(ctx context.Context, kind string, limit int) ([]string, error) {
	// 棣栧厛灏濊瘯浠庢暟鎹簱鑾峰彇鏁版嵁
	var symbols []string

	// 鑾峰彇GORM鏁版嵁搴撳疄渚?	dbInstance := s.db.DB()

	// 灏濊瘯鏁版嵁搴撴煡璇紙浠庢渶鏂扮殑蹇収涓幏鍙栨暟鎹級
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
		log.Printf("[INFO] 鏁版嵁搴撴煡璇㈠け璐ワ紝浣跨敤榛樿甯佺鍒楄〃: %v", err)
	}

	// 濡傛灉鏁版嵁搴撴煡璇㈠け璐ユ垨娌℃湁鏁版嵁锛屼笉杩斿洖榛樿甯佺鍒楄〃
	if len(symbols) == 0 {
		log.Printf("[INFO] 鏁版嵁搴撲腑娌℃湁鍙敤甯佺鏁版嵁")
	}

	return symbols, nil
}

// ===== 榛戝悕鍗曠鐞?API =====

// GET /market/binance/blacklist?kind=spot|futures - 鑾峰彇榛戝悕鍗?func (s *Server) ListBinanceBlacklist(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	items, err := s.db.ListBinanceBlacklist(kind)
	if err != nil {
		s.DatabaseError(c, "鏌ヨ榛戝悕鍗?, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// POST /market/binance/blacklist - 娣诲姞榛戝悕鍗?func (s *Server) AddBinanceBlacklist(c *gin.Context) {
	var body struct {
		Kind   string `json:"kind"` // spot / futures
		Symbol string `json:"symbol"`
	}
	if err := c.BindJSON(&body); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if body.Kind == "" {
		s.ValidationError(c, "kind", "绫诲瀷涓嶈兘涓虹┖锛屽繀椤讳负 spot 鎴?futures")
		return
	}
	if body.Symbol == "" {
		s.ValidationError(c, "symbol", "甯佺绗﹀彿涓嶈兘涓虹┖")
		return
	}
	if err := s.db.AddBinanceBlacklist(body.Kind, body.Symbol); err != nil {
		s.DatabaseError(c, "娣诲姞榛戝悕鍗?, err)
		return
	}
	// 澶辨晥甯傚満鏁版嵁缂撳瓨鍜岄粦鍚嶅崟缂撳瓨锛屼娇榛戝悕鍗曞彉鏇寸珛鍗崇敓鏁?	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}
	if err := s.InvalidateBlacklistCache(c.Request.Context(), body.Kind); err != nil {
		log.Printf("[WARN] Failed to invalidate blacklist cache (kind=%s): %v", body.Kind, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /market/binance/blacklist/:kind/:symbol - 鍒犻櫎榛戝悕鍗?func (s *Server) DeleteBinanceBlacklist(c *gin.Context) {
	kind := strings.TrimSpace(c.Param("kind"))
	symbol := strings.TrimSpace(c.Param("symbol"))
	if symbol == "" {
		s.ValidationError(c, "symbol", "甯佺绗﹀彿涓嶈兘涓虹┖")
		return
	}
	if err := s.db.DeleteBinanceBlacklist(kind, symbol); err != nil {
		s.DatabaseError(c, "鍒犻櫎榛戝悕鍗?, err)
		return
	}
	// 澶辨晥甯傚満鏁版嵁缂撳瓨鍜岄粦鍚嶅崟缂撳瓨锛屼娇榛戝悕鍗曞彉鏇寸珛鍗崇敓鏁?	if err := s.InvalidateMarketCache(c.Request.Context()); err != nil {
		log.Printf("[WARN] Failed to invalidate market cache: %v", err)
	}
	if err := s.InvalidateBlacklistCache(c.Request.Context(), kind); err != nil {
		log.Printf("[WARN] Failed to invalidate blacklist cache (kind=%s): %v", kind, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// normalizeBinanceSymbolToCoinCap 灏嗗竵瀹変氦鏄撳绗﹀彿杞崲涓篊oinCap浣跨敤鐨勫竵绉嶇鍙?func (s *Server) normalizeBinanceSymbolToCoinCap(binanceSymbol string) string {
	if binanceSymbol == "" {
		return ""
	}

	// 瀹氫箟甯歌鐨勪氦鏄撳鍚庣紑锛屾寜闀垮害闄嶅簭鎺掑垪浠ョ‘淇濇纭尮閰?	suffixes := []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToUpper(binanceSymbol), suffix) {
			// 鍘绘帀鍚庣紑锛岃繑鍥炲熀纭€甯佺绗﹀彿
			baseSymbol := strings.TrimSuffix(strings.ToUpper(binanceSymbol), suffix)
			// 纭繚鍩虹绗﹀彿涓嶄负绌?			if baseSymbol != "" {
				return baseSymbol
			}
		}
	}

	// 濡傛灉娌℃湁鍖归厤鍒板父瑙佸悗缂€锛岃繑鍥炲師绗﹀彿锛堝彲鑳芥槸涓€浜涚壒娈婁氦鏄撳锛?	log.Printf("[WARN] 鏃犳硶璇嗗埆甯佸畨浜ゆ槗瀵瑰悗缂€: %s", binanceSymbol)
	return binanceSymbol
}
