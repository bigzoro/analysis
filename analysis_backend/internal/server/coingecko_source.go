package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CoinGeckoSource CoinGecko数据源
type CoinGeckoSource struct {
	httpClient *HTTPClient
	baseURL    string
	apiKey     string // CoinGecko Pro API Key (可选)
}

// NewCoinGeckoSource 创建CoinGecko数据源
func NewCoinGeckoSource() *CoinGeckoSource {
	return &CoinGeckoSource{
		httpClient: NewHTTPClient(),
		baseURL:    "https://api.coingecko.com/api/v3",
		apiKey:     "", // 可以配置Pro API Key
	}
}

// Name 数据源名称
func (cg *CoinGeckoSource) Name() string {
	return "coingecko"
}

// IsAvailable 检查数据源是否可用
func (cg *CoinGeckoSource) IsAvailable() bool {
	// 测试API连通性
	url := fmt.Sprintf("%s/ping", cg.baseURL)
	_, err := cg.httpClient.Get(url)
	return err == nil
}

// FetchMarketData 获取市场数据
func (cg *CoinGeckoSource) FetchMarketData(ctx context.Context, symbols []string) ([]MarketData, error) {
	// CoinGecko使用coin ID而不是symbol，所以需要先转换
	coinIds := cg.symbolsToCoinIds(symbols)

	if len(coinIds) == 0 {
		return nil, fmt.Errorf("no valid coin IDs found for symbols: %v", symbols)
	}

	// 构建API URL
	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&ids=%s&order=market_cap_desc&per_page=100&page=1&sparkline=false&price_change_percentage=1h,24h,7d,30d",
		cg.baseURL, strings.Join(coinIds, ","))

	// 添加API Key (如果有)
	if cg.apiKey != "" {
		url += fmt.Sprintf("&x_cg_demo_api_key=%s", cg.apiKey)
	}

	data, err := cg.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CoinGecko market data: %w", err)
	}

	var coinGeckoResponse []CoinGeckoMarketData
	if err := json.Unmarshal(data, &coinGeckoResponse); err != nil {
		return nil, fmt.Errorf("failed to parse CoinGecko response: %w", err)
	}

	result := make([]MarketData, 0, len(coinGeckoResponse))
	for _, coin := range coinGeckoResponse {
		marketData := MarketData{
			Symbol:      strings.ToUpper(coin.Symbol),
			Source:      cg.Name(),
			Price:       coin.CurrentPrice,
			Volume24h:   coin.TotalVolume,
			MarketCap:   coin.MarketCap,
			LastUpdated: time.Now(),
		}

		// 解析价格变化
		if coin.PriceChangePct24h != 0 {
			marketData.Change24h = coin.PriceChangePct24h
		}

		result = append(result, marketData)
	}

	return result, nil
}

// FetchNewsData 获取新闻数据 (CoinGecko免费版不支持)
func (cg *CoinGeckoSource) FetchNewsData(ctx context.Context, symbols []string) ([]NewsData, error) {
	// CoinGecko免费版不支持新闻数据
	return []NewsData{}, nil
}

// FetchSocialData 获取社交数据
func (cg *CoinGeckoSource) FetchSocialData(ctx context.Context, symbols []string) ([]SocialData, error) {
	coinIds := cg.symbolsToCoinIds(symbols)

	if len(coinIds) == 0 {
		return nil, fmt.Errorf("no valid coin IDs found for symbols: %v", symbols)
	}

	result := make([]SocialData, 0)

	for _, coinId := range coinIds {
		// 获取社交媒体统计
		url := fmt.Sprintf("%s/coins/%s", cg.baseURL, coinId)
		if cg.apiKey != "" {
			url += fmt.Sprintf("?x_cg_demo_api_key=%s", cg.apiKey)
		}

		data, err := cg.httpClient.Get(url)
		if err != nil {
			continue // 跳过错误，继续下一个
		}

		var coinDetail CoinGeckoCoinDetail
		if err := json.Unmarshal(data, &coinDetail); err != nil {
			continue
		}

		// 从社区数据中提取社交信息
		if coinDetail.CommunityData != nil {
			symbol := cg.coinIdToSymbol(coinId)
			if symbol != "" {
				socialData := SocialData{
					Platform:   "coingecko_community",
					Symbol:     strings.ToUpper(symbol),
					Mentions:   coinDetail.CommunityData.TwitterFollowers,
					Sentiment:  0.5, // CoinGecko没有情感数据，默认中性
					Engagement: coinDetail.CommunityData.RedditSubscribers,
					PostedAt:   time.Now(),
				}
				result = append(result, socialData)
			}
		}

		// 避免API限速
		time.Sleep(100 * time.Millisecond)
	}

	return result, nil
}

// symbolsToCoinIds 将币种符号转换为CoinGecko的coin ID
func (cg *CoinGeckoSource) symbolsToCoinIds(symbols []string) []string {
	// CoinGecko coin ID映射 (常用币种)
	coinIdMap := map[string]string{
		"BTC":   "bitcoin",
		"ETH":   "ethereum",
		"BNB":   "binancecoin",
		"ADA":   "cardano",
		"XRP":   "ripple",
		"SOL":   "solana",
		"DOT":   "polkadot",
		"DOGE":  "dogecoin",
		"AVAX":  "avalanche-2",
		"MATIC": "matic-network",
		"LINK":  "chainlink",
		"UNI":   "uniswap",
		"ALGO":  "algorand",
		"VET":   "vechain",
		"ICP":   "internet-computer",
		"ETC":   "ethereum-classic",
		"XLM":   "stellar",
		"HBAR":  "hedera-hashgraph",
		"NEAR":  "near",
		"FLOW":  "flow",
		"MANA":  "decentraland",
		"SAND":  "the-sandbox",
		"AXS":   "axie-infinity",
		"CHZ":   "chiliz",
		"ENJ":   "enjincoin",
		"APE":   "apecoin",
		"LRC":   "loopring",
		"STORJ": "storj",
		"BAT":   "basic-attention-token",
		"ANT":   "aragon",
		"GRT":   "the-graph",
		"RNDR":  "render-token",
		"IMX":   "immutable-x",
		"GALA":  "gala",
		"YGG":   "yield-guild-games",
		"ILV":   "illuvium",
		"GMT":   "stepn",
		"GST":   "green-satoshi-token",
		"DASH":  "dash",
		"KAI":   "kaiten",
		"KAITO": "kaito",
	}

	var coinIds []string
	for _, symbol := range symbols {
		symbol = strings.ToUpper(symbol)

		// 首先尝试直接匹配
		if coinId, exists := coinIdMap[symbol]; exists {
			coinIds = append(coinIds, coinId)
			continue
		}

		// 如果直接匹配失败，尝试去除交易对后缀再匹配
		baseSymbol := cg.extractBaseSymbol(symbol)
		if baseSymbol != symbol {
			if coinId, exists := coinIdMap[baseSymbol]; exists {
				coinIds = append(coinIds, coinId)
				continue
			}
		}

		// 如果还是找不到，记录警告但不添加到结果中
		// 这样可以避免因为个别未知币种导致整个批量请求失败
	}

	return coinIds
}

// extractBaseSymbol 从交易对符号中提取基础币种符号
func (cg *CoinGeckoSource) extractBaseSymbol(symbol string) string {
	// 常见的交易对后缀
	quoteAssets := []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB", "ADA", "SOL", "DOT", "PERP"}

	for _, quote := range quoteAssets {
		if strings.HasSuffix(symbol, quote) {
			return strings.TrimSuffix(symbol, quote)
		}
	}

	return symbol // 如果没有匹配的后缀，返回原符号
}

// coinIdToSymbol 将CoinGecko coin ID转换为符号
func (cg *CoinGeckoSource) coinIdToSymbol(coinId string) string {
	// 反向映射
	symbolMap := map[string]string{
		"bitcoin":               "BTC",
		"ethereum":              "ETH",
		"binancecoin":           "BNB",
		"cardano":               "ADA",
		"ripple":                "XRP",
		"solana":                "SOL",
		"polkadot":              "DOT",
		"dogecoin":              "DOGE",
		"avalanche-2":           "AVAX",
		"matic-network":         "MATIC",
		"chainlink":             "LINK",
		"uniswap":               "UNI",
		"algorand":              "ALGO",
		"vechain":               "VET",
		"internet-computer":     "ICP",
		"ethereum-classic":      "ETC",
		"stellar":               "XLM",
		"hedera-hashgraph":      "HBAR",
		"near":                  "NEAR",
		"flow":                  "FLOW",
		"decentraland":          "MANA",
		"the-sandbox":           "SAND",
		"axie-infinity":         "AXS",
		"chiliz":                "CHZ",
		"enjincoin":             "ENJ",
		"apecoin":               "APE",
		"loopring":              "LRC",
		"storj":                 "STORJ",
		"basic-attention-token": "BAT",
		"aragon":                "ANT",
		"the-graph":             "GRT",
		"render-token":          "RNDR",
		"immutable-x":           "IMX",
		"gala":                  "GALA",
		"yield-guild-games":     "YGG",
		"illuvium":              "ILV",
		"stepn":                 "GMT",
		"green-satoshi-token":   "GST",
	}

	if symbol, exists := symbolMap[coinId]; exists {
		return symbol
	}

	return ""
}

// CoinGeckoCoinDetail CoinGecko详细数据结构
type CoinGeckoCoinDetail struct {
	ID                  string                  `json:"id"`
	Symbol              string                  `json:"symbol"`
	Name                string                  `json:"name"`
	CommunityData       *CoinGeckoCommunityData `json:"community_data"`
	DeveloperData       *interface{}            `json:"developer_data"`
	PublicInterestStats *interface{}            `json:"public_interest_stats"`
}

// CoinGeckoCommunityData 社区数据结构
type CoinGeckoCommunityData struct {
	FacebookLikes            int     `json:"facebook_likes"`
	TwitterFollowers         int     `json:"twitter_followers"`
	RedditAveragePosts48h    float64 `json:"reddit_average_posts_48h"`
	RedditAverageComments48h float64 `json:"reddit_average_comments_48h"`
	RedditSubscribers        int     `json:"reddit_subscribers"`
	RedditAccountsActive48h  int     `json:"reddit_accounts_active_48h"`
	TelegramChannelUserCount *int    `json:"telegram_channel_user_count"`
}
