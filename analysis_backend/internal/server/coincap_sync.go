package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"analysis/internal/db"
)

// CoinCapAssetSyncService CoinCap资产同步服务
type CoinCapAssetSyncService struct {
	baseURL string
	apiKey  string
	httpc   *http.Client
	db      *db.CoinCapMappingService
}

// CoinCapAssetResponse CoinCap API响应结构
type CoinCapAssetResponse struct {
	Data []CoinCapAssetItem `json:"data"`
}

// CoinCapAssetItem CoinCap资产项结构
type CoinCapAssetItem struct {
	ID         string `json:"id"`
	Rank       string `json:"rank"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	Supply     string `json:"supply"`
	MaxSupply  string `json:"maxSupply"`
	MarketCap  string `json:"marketCapUsd"`
	Volume24Hr string `json:"volumeUsd24Hr"`
	Price      string `json:"priceUsd"`
	Change24Hr string `json:"changePercent24Hr"`
	VWAP24Hr   string `json:"vwap24Hr"`
	Explorer   string `json:"explorer"`
}

// NewCoinCapAssetSyncService 创建CoinCap资产同步服务
func NewCoinCapAssetSyncService(dbService *db.CoinCapMappingService, apiKey string) *CoinCapAssetSyncService {
	return &CoinCapAssetSyncService{
		baseURL: "https://rest.coincap.io/v3",
		apiKey:  apiKey,
		httpc: &http.Client{
			Timeout: 30 * time.Second,
		},
		db: dbService,
	}
}

// SyncAllAssets 同步所有CoinCap资产到数据库
func (s *CoinCapAssetSyncService) SyncAllAssets(ctx context.Context) error {
	log.Printf("[coincap-sync] 开始同步CoinCap资产映射数据...")

	// 获取所有资产数据
	assets, err := s.fetchAllAssets(ctx)
	if err != nil {
		return fmt.Errorf("获取CoinCap资产数据失败: %w", err)
	}

	log.Printf("[coincap-sync] 获取到 %d 个资产，开始保存到数据库...", len(assets))

	// 清空现有映射（可选，如果需要全量更新）
	// 注意：生产环境可能需要更谨慎的处理
	// err = s.db.ClearAllMappings(ctx)
	// if err != nil {
	// 	return fmt.Errorf("清空现有映射失败: %w", err)
	// }

	// 转换为映射结构
	mappings := make([]db.CoinCapAssetMapping, 0, len(assets))
	for _, asset := range assets {
		// 跳过空数据
		if asset.Symbol == "" || asset.ID == "" {
			continue
		}

		mappings = append(mappings, db.CoinCapAssetMapping{
			Symbol:  asset.Symbol,
			AssetID: asset.ID,
			Name:    asset.Name,
			Rank:    asset.Rank,
		})
	}

	// 批量保存到数据库
	err = s.db.BatchUpsertAssetMappings(ctx, mappings)
	if err != nil {
		return fmt.Errorf("批量保存资产映射失败: %w", err)
	}

	// 获取统计信息
	stats, err := s.db.GetMappingStats(ctx)
	if err != nil {
		log.Printf("[coincap-sync] 获取统计信息失败: %v", err)
	} else {
		log.Printf("[coincap-sync] 同步完成！总映射数量: %v, 最后更新时间: %v",
			stats["total_mappings"], stats["latest_update"])
	}

	return nil
}

// fetchAllAssets 从CoinCap API获取所有资产
func (s *CoinCapAssetSyncService) fetchAllAssets(ctx context.Context) ([]CoinCapAssetItem, error) {
	u := fmt.Sprintf("%s/assets", s.baseURL)

	// 构建查询参数
	q := url.Values{}
	q.Set("limit", "2000") // CoinCap最多支持2000个资产
	if s.apiKey != "" {
		q.Set("apiKey", s.apiKey)
	}
	u += "?" + q.Encode()

	log.Printf("[coincap-sync] 请求URL: %s", u)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "coincap-asset-sync/1.0")
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("x-api-key", s.apiKey)
	}

	resp, err := s.httpc.Do(req)
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

	log.Printf("[coincap-sync] 成功获取 %d 个资产", len(response.Data))
	return response.Data, nil
}

// ValidateMappings 验证映射数据的完整性
func (s *CoinCapAssetSyncService) ValidateMappings(ctx context.Context) error {
	log.Printf("[coincap-sync] 开始验证映射数据完整性...")

	mappings, err := s.db.GetAllMappings(ctx)
	if err != nil {
		return fmt.Errorf("获取映射数据失败: %w", err)
	}

	if len(mappings) == 0 {
		return fmt.Errorf("映射数据为空，请先执行同步")
	}

	// 检查数据完整性
	validCount := 0
	invalidCount := 0

	for _, mapping := range mappings {
		if mapping.Symbol == "" || mapping.AssetID == "" {
			log.Printf("[coincap-sync] 发现无效映射: symbol=%s, asset_id=%s", mapping.Symbol, mapping.AssetID)
			invalidCount++
		} else {
			validCount++
		}
	}

	log.Printf("[coincap-sync] 验证完成: 有效映射 %d 个, 无效映射 %d 个", validCount, invalidCount)

	if invalidCount > 0 {
		return fmt.Errorf("发现 %d 个无效映射", invalidCount)
	}

	return nil
}

// GetPopularSymbols 获取热门交易符号（市值排名前100）
func (s *CoinCapAssetSyncService) GetPopularSymbols(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}

	mappings, err := s.db.GetAllMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取映射数据失败: %w", err)
	}

	// 按rank排序并限制数量
	symbols := make([]string, 0, limit)
	for i, mapping := range mappings {
		if i >= limit {
			break
		}
		if mapping.Rank != "" {
			// 尝试解析rank为数字
			if rank, err := strconv.Atoi(mapping.Rank); err == nil && rank > 0 {
				symbols = append(symbols, mapping.Symbol)
			}
		}
	}

	return symbols, nil
}

// SearchAssets 搜索资产（按symbol或name）
func (s *CoinCapAssetSyncService) SearchAssets(ctx context.Context, query string) ([]db.CoinCapAssetMapping, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	mappings, err := s.db.GetAllMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取映射数据失败: %w", err)
	}

	// 简单搜索（生产环境建议使用数据库全文搜索）
	query = strings.ToLower(query)
	var results []db.CoinCapAssetMapping

	for _, mapping := range mappings {
		if strings.Contains(strings.ToLower(mapping.Symbol), query) ||
			strings.Contains(strings.ToLower(mapping.Name), query) {
			results = append(results, mapping)
		}
	}

	return results, nil
}
