// coincap_market_sync.go - CoinCap市值数据同步服务
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"analysis/internal/db"
)

// CoinCapMarketDataSyncService CoinCap市值数据同步服务
type CoinCapMarketDataSyncService struct {
	marketDataService *db.CoinCapMarketDataService
	baseURL           string
	apiKey            string
	httpClient        *http.Client
}

// NewCoinCapMarketDataSyncService 创建CoinCap市值数据同步服务
func NewCoinCapMarketDataSyncService(marketDataService *db.CoinCapMarketDataService, apiKey string) *CoinCapMarketDataSyncService {
	return &CoinCapMarketDataSyncService{
		marketDataService: marketDataService,
		baseURL:           "https://rest.coincap.io/v3",
		apiKey:            apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SyncAllMarketData 同步所有CoinCap市值数据
func (s *CoinCapMarketDataSyncService) SyncAllMarketData(ctx context.Context) error {
	log.Printf("[coincap-market-sync] 开始同步CoinCap市值数据...")

	// 获取所有资产数据
	assets, err := s.fetchAllAssets(ctx)
	if err != nil {
		return fmt.Errorf("获取CoinCap资产数据失败: %w", err)
	}

	log.Printf("[coincap-market-sync] 获取到 %d 个资产，开始保存市值数据...", len(assets))

	// 转换为数据库模型并批量保存
	dataList := make([]*db.CoinCapMarketData, 0, len(assets))
	for _, asset := range assets {
		data := &db.CoinCapMarketData{
			Symbol:            strings.ToUpper(strings.TrimSuffix(asset.Symbol, "USDT")),
			AssetID:           asset.ID,
			Name:              asset.Name,
			Rank:              asset.Rank,
			PriceUSD:          asset.Price,
			Change24Hr:        asset.Change24Hr,
			MarketCapUSD:      asset.MarketCap,
			CirculatingSupply: asset.Supply,
			TotalSupply:       asset.MaxSupply,
			Volume24Hr:        asset.Volume24Hr,
			VWAP24Hr:          asset.VWAP24Hr,
			Explorer:          asset.Explorer,
			UpdatedAt:         time.Now(),
		}
		dataList = append(dataList, data)
	}

	// 批量保存到数据库
	if err := s.marketDataService.BatchUpsertMarketData(ctx, dataList); err != nil {
		return fmt.Errorf("批量保存市值数据失败: %w", err)
	}

	log.Printf("[coincap-market-sync] 市值数据同步完成，保存了 %d 条记录", len(dataList))
	return nil
}

// fetchAllAssets 从CoinCap API获取所有资产
func (s *CoinCapMarketDataSyncService) fetchAllAssets(ctx context.Context) ([]CoinCapAssetItem, error) {
	u := fmt.Sprintf("%s/assets", s.baseURL)

	// 构建查询参数
	q := url.Values{}
	q.Set("limit", "2000") // CoinCap最多支持2000个资产
	if s.apiKey != "" {
		q.Set("apiKey", s.apiKey)
	}
	u += "?" + q.Encode()

	log.Printf("[coincap-market-sync] 请求URL: %s", u)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "coincap-market-sync/1.0")
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("x-api-key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败: %s => %d, body: %s", u, resp.StatusCode, string(bodyBytes))
	}

	var response CoinCapAssetResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	log.Printf("[coincap-market-sync] 成功获取 %d 个资产数据", len(response.Data))
	return response.Data, nil
}
