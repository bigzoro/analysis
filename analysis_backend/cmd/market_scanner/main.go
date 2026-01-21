package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"analysis/internal/config"
	"analysis/internal/db"
	"analysis/internal/netutil"
)

type Binance24hrTicker struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	BidPrice           string `json:"bidPrice"`
	BidQty             string `json:"bidQty"`
	AskPrice           string `json:"askPrice"`
	AskQty             string `json:"askQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstId            int64  `json:"firstId"`
	LastId             int64  `json:"lastId"`
	Count              int64  `json:"count"`
}

type MarketDataItem struct {
	Symbol             string   `json:"symbol"`
	LastPrice          string   `json:"last_price"`
	Volume             string   `json:"volume"`
	PriceChangePercent float64  `json:"price_change_percent"`
	MarketCapUSD       *float64 `json:"market_cap_usd,omitempty"`
	FDVUSD             *float64 `json:"fdv_usd,omitempty"`
	CirculatingSupply  *float64 `json:"circulating_supply,omitempty"`
	TotalSupply        *float64 `json:"total_supply,omitempty"`
}

type MarketDataRequest struct {
	Kind      string           `json:"kind"`
	Bucket    string           `json:"bucket"`
	FetchedAt string           `json:"fetched_at"`
	Items     []MarketDataItem `json:"items"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	apiBase := flag.String("api", "http://localhost:8010", "api base url")
	interval := flag.Duration("interval", 1*time.Hour, "scan interval")
	flag.Parse()

	log.Printf("启动参数: config=%s, api=%s, interval=%v", *configPath, *apiBase, *interval)

	// 加载配置
	var cfg config.Config
	config.MustLoad(*configPath, &cfg)
	config.ApplyProxy(&cfg)

	log.Printf("启动 market_scanner，API: %s，间隔: %v", *apiBase, *interval)

	// 初始化数据库连接
	database, err := db.OpenMySQL(db.Options{
		DSN:             cfg.Database.DSN,
		Automigrate:     false, // 不自动迁移，只查询
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()

	gormDB, err := database.DB()
	if err != nil {
		log.Fatalf("获取GORM数据库实例失败: %v", err)
	}

	// 创建CoinCap市值数据服务
	marketDataService := db.NewCoinCapMarketDataService(gormDB)

	// HTTP客户端（支持代理）
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConns:        128,
			MaxIdleConnsPerHost: 32,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 15 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false, // 保持连接复用
		},
		Timeout: 30 * time.Second,
	}

	ctx := context.Background()
	isFirstRun := true

	for {
		startTime := time.Now()

		if isFirstRun {
			// 第一次运行时，同时扫描前一个小时和当前小时的数据
			log.Printf("首次运行，同时扫描前一个小时和当前小时的数据")

			// 扫描前一个小时的数据
			if err := scanMarketWithBucket(ctx, client, marketDataService, "spot", *apiBase+"/ingest/binance/market", startTime.Add(-1*time.Hour)); err != nil {
				log.Printf("扫描前一个小时现货市场失败: %v", err)
			}
			if err := scanMarketWithBucket(ctx, client, marketDataService, "futures", *apiBase+"/ingest/binance/market", startTime.Add(-1*time.Hour)); err != nil {
				log.Printf("扫描前一个小时期货市场失败: %v", err)
			}

			isFirstRun = false
		}

		// 扫描当前小时的数据
		if err := scanMarket(ctx, client, marketDataService, "spot", *apiBase+"/ingest/binance/market"); err != nil {
			log.Printf("扫描现货市场失败: %v", err)
		}

		if err := scanMarket(ctx, client, marketDataService, "futures", *apiBase+"/ingest/binance/market"); err != nil {
			log.Printf("扫描期货市场失败: %v", err)
		}

		// 计算下次执行时间（1小时对齐）
		nextBucket := startTime.UTC().Truncate(1 * time.Hour).Add(1 * time.Hour)
		sleepDuration := nextBucket.Sub(time.Now())
		if sleepDuration <= 0 {
			sleepDuration = 1 * time.Hour
		}

		log.Printf("扫描完成，下次执行时间: %s，等待 %v", nextBucket.Format(time.RFC3339), sleepDuration)
		time.Sleep(sleepDuration)
	}
}

func scanMarketWithBucket(ctx context.Context, client *http.Client, marketDataService *db.CoinCapMarketDataService, kind, apiURL string, bucketTime time.Time) error {
	return scanMarketInternal(ctx, client, marketDataService, kind, apiURL, &bucketTime)
}

func scanMarket(ctx context.Context, client *http.Client, marketDataService *db.CoinCapMarketDataService, kind, apiURL string) error {
	return scanMarketInternal(ctx, client, marketDataService, kind, apiURL, nil)
}

func scanMarketInternal(ctx context.Context, client *http.Client, marketDataService *db.CoinCapMarketDataService, kind, apiURL string, bucketTimeOverride *time.Time) error {
	log.Printf("开始扫描 %s 市场", kind)

	// 获取Binance 24hr统计数据
	tickers, err := getBinance24hrTickers(ctx, kind)
	if err != nil {
		return fmt.Errorf("获取 %s 市场数据失败: %w", kind, err)
	}

	// 过滤和排序
	filtered := filterAndSortTickers(tickers, kind)
	if len(filtered) == 0 {
		log.Printf("%s 市场没有有效数据", kind)
		return nil
	}

	// 构建请求数据
	now := time.Now().UTC()
	bucket := now.Truncate(1 * time.Hour)

	// 如果指定了bucket时间，使用指定的时间
	if bucketTimeOverride != nil {
		bucket = bucketTimeOverride.UTC().Truncate(1 * time.Hour)
	}

	// 提取所有交易对的币种符号（去掉USDT后缀）
	symbols := make([]string, 0, len(filtered))
	for _, ticker := range filtered {
		symbol := strings.TrimSuffix(ticker.Symbol, "USDT")
		if symbol != "" {
			symbols = append(symbols, symbol)
		}
	}

	// 批量查询市值数据
	marketDataMap, err := marketDataService.GetMarketDataBySymbols(ctx, symbols)
	if err != nil {
		log.Printf("查询市值数据失败: %v，将继续处理但市值数据为空", err)
		marketDataMap = make(map[string]*db.CoinCapMarketData)
	}

	items := make([]MarketDataItem, 0, len(filtered))
	for _, ticker := range filtered {
		pctChange, _ := strconv.ParseFloat(ticker.PriceChangePercent, 64)

		// 提取币种符号（去掉USDT后缀）
		symbol := strings.TrimSuffix(ticker.Symbol, "USDT")

		item := MarketDataItem{
			Symbol:             ticker.Symbol,
			LastPrice:          ticker.LastPrice,
			Volume:             ticker.Volume,
			PriceChangePercent: pctChange,
		}

		// 从CoinCap数据中获取市值信息
		if marketData, exists := marketDataMap[symbol]; exists {
			if marketCap, err := strconv.ParseFloat(marketData.MarketCapUSD, 64); err == nil {
				item.MarketCapUSD = &marketCap
			}
			if circulatingSupply, err := strconv.ParseFloat(marketData.CirculatingSupply, 64); err == nil {
				item.CirculatingSupply = &circulatingSupply
			}
			if totalSupply, err := strconv.ParseFloat(marketData.TotalSupply, 64); err == nil {
				item.TotalSupply = &totalSupply
			}
		}

		items = append(items, item)
	}

	req := MarketDataRequest{
		Kind:      kind,
		Bucket:    bucket.Format(time.RFC3339),
		FetchedAt: now.Format(time.RFC3339),
		Items:     items,
	}

	// 发送到API
	return sendToAPI(ctx, client, apiURL, req)
}

func getBinance24hrTickers(ctx context.Context, kind string) ([]Binance24hrTicker, error) {
	var url string
	switch kind {
	case "spot":
		url = "https://api.binance.com/api/v3/ticker/24hr"
	case "futures":
		url = "https://fapi.binance.com/fapi/v1/ticker/24hr"
	default:
		return nil, fmt.Errorf("不支持的市场类型: %s", kind)
	}

	var tickers []Binance24hrTicker
	err := netutil.GetJSON(ctx, url, &tickers)
	if err != nil {
		return nil, err
	}

	return tickers, nil
}

func filterAndSortTickers(tickers []Binance24hrTicker, kind string) []Binance24hrTicker {
	var filtered []Binance24hrTicker

	for _, ticker := range tickers {
		// 过滤掉非USDT交易对（可选，根据需求调整）
		if !strings.HasSuffix(ticker.Symbol, "USDT") {
			continue
		}

		// 过滤掉成交量为0的交易对
		if ticker.Volume == "0" || ticker.Volume == "0.00000000" {
			continue
		}

		// 过滤掉价格为0的交易对
		if ticker.LastPrice == "0" || ticker.LastPrice == "0.00000000" {
			continue
		}

		filtered = append(filtered, ticker)
	}

	// 按涨幅降序排序（涨幅最高的排在前面，更符合涨幅榜的含义）
	sort.Slice(filtered, func(i, j int) bool {
		pctI, _ := strconv.ParseFloat(filtered[i].PriceChangePercent, 64)
		pctJ, _ := strconv.ParseFloat(filtered[j].PriceChangePercent, 64)
		return pctI > pctJ
	})

	return filtered
}

func sendToAPI(ctx context.Context, client *http.Client, url string, req MarketDataRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API响应错误: %d", resp.StatusCode)
	}

	log.Printf("成功发送 %s 市场数据，交易对数量: %d", req.Kind, len(req.Items))
	return nil
}
