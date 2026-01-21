package db

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CoinCapMappingService CoinCap资产映射服务
type CoinCapMappingService struct {
	db *gorm.DB
}

// NewCoinCapMappingService 创建CoinCap映射服务
func NewCoinCapMappingService(db *gorm.DB) *CoinCapMappingService {
	return &CoinCapMappingService{db: db}
}

// UpsertAssetMapping 插入或更新资产映射
func (s *CoinCapMappingService) UpsertAssetMapping(ctx context.Context, symbol, assetID, name, rank string) error {
	mapping := &CoinCapAssetMapping{
		Symbol:    strings.ToUpper(strings.TrimSpace(symbol)),
		AssetID:   strings.ToLower(strings.TrimSpace(assetID)),
		Name:      name,
		Rank:      rank,
		UpdatedAt: time.Now(),
	}

	// 使用 Upsert 操作
	err := s.db.WithContext(ctx).Where(CoinCapAssetMapping{Symbol: mapping.Symbol}).
		Assign(map[string]interface{}{
			"asset_id":   mapping.AssetID,
			"name":       mapping.Name,
			"rank":       mapping.Rank,
			"updated_at": mapping.UpdatedAt,
		}).
		FirstOrCreate(mapping).Error

	return err
}

// BatchUpsertAssetMappings 批量插入或更新资产映射
func (s *CoinCapMappingService) BatchUpsertAssetMappings(ctx context.Context, mappings []CoinCapAssetMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	// 标准化数据
	for i := range mappings {
		mappings[i].Symbol = strings.ToUpper(strings.TrimSpace(mappings[i].Symbol))
		mappings[i].AssetID = strings.ToLower(strings.TrimSpace(mappings[i].AssetID))
		mappings[i].UpdatedAt = time.Now()
	}

	// 使用兼容的Upsert语法
	for _, mapping := range mappings {
		err := s.UpsertAssetMapping(ctx, mapping.Symbol, mapping.AssetID, mapping.Name, mapping.Rank)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAssetIDBySymbol 根据symbol获取assetID
func (s *CoinCapMappingService) GetAssetIDBySymbol(ctx context.Context, symbol string) (string, error) {
	var mapping CoinCapAssetMapping
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	err := s.db.WithContext(ctx).Where("symbol = ?", symbol).First(&mapping).Error
	if err != nil {
		return "", err
	}

	return mapping.AssetID, nil
}

// GetAllMappings 获取所有资产映射
func (s *CoinCapMappingService) GetAllMappings(ctx context.Context) ([]CoinCapAssetMapping, error) {
	var mappings []CoinCapAssetMapping
	err := s.db.WithContext(ctx).Order("rank ASC, symbol ASC").Find(&mappings).Error
	return mappings, err
}

// GetMappingsBySymbols 根据多个symbol获取映射
func (s *CoinCapMappingService) GetMappingsBySymbols(ctx context.Context, symbols []string) (map[string]string, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	// 标准化symbols
	normalizedSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		normalizedSymbols[i] = strings.ToUpper(strings.TrimSpace(symbol))
	}

	var mappings []CoinCapAssetMapping
	err := s.db.WithContext(ctx).Where("symbol IN ?", normalizedSymbols).Find(&mappings).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, mapping := range mappings {
		result[mapping.Symbol] = mapping.AssetID
	}

	return result, nil
}

// GetSymbolByAssetID 根据assetID获取symbol
func (s *CoinCapMappingService) GetSymbolByAssetID(ctx context.Context, assetID string) (string, error) {
	var mapping CoinCapAssetMapping
	assetID = strings.ToLower(strings.TrimSpace(assetID))

	err := s.db.WithContext(ctx).Where("asset_id = ?", assetID).First(&mapping).Error
	if err != nil {
		return "", err
	}

	return mapping.Symbol, nil
}

// ClearAllMappings 清空所有映射（用于重新同步）
func (s *CoinCapMappingService) ClearAllMappings(ctx context.Context) error {
	return s.db.WithContext(ctx).Where("1 = 1").Delete(&CoinCapAssetMapping{}).Error
}

// GetMappingStats 获取映射统计信息
func (s *CoinCapMappingService) GetMappingStats(ctx context.Context) (map[string]interface{}, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&CoinCapAssetMapping{}).Count(&count).Error
	if err != nil {
		return nil, err
	}

	var latestUpdate time.Time
	err = s.db.WithContext(ctx).Model(&CoinCapAssetMapping{}).
		Select("MAX(updated_at) as latest_update").
		Scan(&latestUpdate).Error
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_mappings": count,
		"latest_update":  latestUpdate,
	}, nil
}

// CoinCapMarketDataService CoinCap市值数据服务
type CoinCapMarketDataService struct {
	db *gorm.DB
}

// NewCoinCapMarketDataService 创建CoinCap市值数据服务
func NewCoinCapMarketDataService(db *gorm.DB) *CoinCapMarketDataService {
	return &CoinCapMarketDataService{db: db}
}

// UpsertMarketData 插入或更新市值数据
func (s *CoinCapMarketDataService) UpsertMarketData(ctx context.Context, data *CoinCapMarketData) error {
	return s.db.WithContext(ctx).Where(CoinCapMarketData{Symbol: data.Symbol}).
		Assign(data).
		FirstOrCreate(data).Error
}

// BatchUpsertMarketData 批量插入或更新市值数据
func (s *CoinCapMarketDataService) BatchUpsertMarketData(ctx context.Context, dataList []*CoinCapMarketData) error {
	for _, data := range dataList {
		if err := s.UpsertMarketData(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

// GetMarketDataBySymbol 根据符号获取市值数据
func (s *CoinCapMarketDataService) GetMarketDataBySymbol(ctx context.Context, symbol string) (*CoinCapMarketData, error) {
	var data CoinCapMarketData
	err := s.db.WithContext(ctx).Where("symbol = ?", symbol).First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// GetMarketDataBySymbols 批量获取市值数据
func (s *CoinCapMarketDataService) GetMarketDataBySymbols(ctx context.Context, symbols []string) (map[string]*CoinCapMarketData, error) {
	var dataList []CoinCapMarketData
	err := s.db.WithContext(ctx).Where("symbol IN ?", symbols).Find(&dataList).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*CoinCapMarketData)
	for i := range dataList {
		result[dataList[i].Symbol] = &dataList[i]
	}
	return result, nil
}

// GetSymbolsByMarketCapRange 获取市值在指定范围内的币种
func (s *CoinCapMarketDataService) GetSymbolsByMarketCapRange(ctx context.Context, minCap, maxCap float64) ([]string, error) {
	var symbols []string
	// 由于market_cap_usd现在是字符串类型，我们需要转换为数值进行比较
	query := `
		SELECT symbol
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
		ORDER BY CAST(market_cap_usd AS DECIMAL(30,8)) ASC
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap).Scan(&symbols).Error
	return symbols, err
}

// GetAllMarketData 获取所有市值数据（用于调试）
func (s *CoinCapMarketDataService) GetAllMarketData(ctx context.Context) ([]*CoinCapMarketData, error) {
	var dataList []*CoinCapMarketData
	err := s.db.WithContext(ctx).Find(&dataList).Error
	return dataList, err
}

// GetMarketDataStats 获取市值数据统计信息
func (s *CoinCapMarketDataService) GetMarketDataStats(ctx context.Context) (map[string]interface{}, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&CoinCapMarketData{}).Count(&count).Error
	if err != nil {
		return nil, err
	}

	var latestUpdate time.Time
	err = s.db.WithContext(ctx).Model(&CoinCapMarketData{}).
		Select("MAX(updated_at) as latest_update").
		Scan(&latestUpdate).Error
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_records": count,
		"latest_update": latestUpdate,
	}, nil
}

// CleanOldData 清理旧的市值数据（保留最近N天的）
func (s *CoinCapMarketDataService) CleanOldData(ctx context.Context, days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	return s.db.WithContext(ctx).Where("updated_at < ?", cutoff).Delete(&CoinCapMarketData{}).Error
}

// GetMarketDataByMarketCapRange 获取市值在指定范围内的完整数据（优化为一次查询）
func (s *CoinCapMarketDataService) GetMarketDataByMarketCapRange(ctx context.Context, minCap, maxCap float64, limit int) ([]*CoinCapMarketData, error) {
	var dataList []*CoinCapMarketData
	// 直接查询符合市值条件的所有数据，避免分两步查询
	query := `
		SELECT *
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
		ORDER BY CAST(market_cap_usd AS DECIMAL(30,8)) DESC
		LIMIT ?
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap, limit).Scan(&dataList).Error
	return dataList, err
}

// GetMarketDataCountByMarketCapRange 获取市值在指定范围内的数据总数
func (s *CoinCapMarketDataService) GetMarketDataCountByMarketCapRange(ctx context.Context, minCap, maxCap float64) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*)
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap).Scan(&count).Error
	return count, err
}

// GetMarketDataByMarketCapRangePaged 获取市值在指定范围内的分页数据
func (s *CoinCapMarketDataService) GetMarketDataByMarketCapRangePaged(ctx context.Context, minCap, maxCap float64, limit int, offset int) ([]*CoinCapMarketData, error) {
	var dataList []*CoinCapMarketData
	query := `
		SELECT *
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
		ORDER BY CAST(market_cap_usd AS DECIMAL(30,8)) DESC
		LIMIT ? OFFSET ?
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap, limit, offset).Scan(&dataList).Error
	return dataList, err
}

// GetMarketDataCountByMarketCapRangeAndSymbols 获取市值在指定范围内且在指定符号列表中的数据总数
func (s *CoinCapMarketDataService) GetMarketDataCountByMarketCapRangeAndSymbols(ctx context.Context, minCap, maxCap float64, symbols []string) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*)
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
		AND symbol IN ?
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap, symbols).Scan(&count).Error
	return count, err
}

// GetMarketDataByMarketCapRangeAndSymbols 获取市值在指定范围内且在指定符号列表中的数据（不分页）
func (s *CoinCapMarketDataService) GetMarketDataByMarketCapRangeAndSymbols(ctx context.Context, minCap, maxCap float64, symbols []string, limit int) ([]*CoinCapMarketData, error) {
	var dataList []*CoinCapMarketData
	query := `
		SELECT *
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
		AND symbol IN ?
		ORDER BY CAST(market_cap_usd AS DECIMAL(30,8)) DESC
		LIMIT ?
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap, symbols, limit).Scan(&dataList).Error
	return dataList, err
}

// GetMarketDataByMarketCapRangeAndSymbolsPaged 获取市值在指定范围内且在指定符号列表中的分页数据
func (s *CoinCapMarketDataService) GetMarketDataByMarketCapRangeAndSymbolsPaged(ctx context.Context, minCap, maxCap float64, symbols []string, limit int, offset int) ([]*CoinCapMarketData, error) {
	var dataList []*CoinCapMarketData
	query := `
		SELECT *
		FROM coin_cap_market_data
		WHERE CAST(market_cap_usd AS DECIMAL(30,8)) >= ?
		AND CAST(market_cap_usd AS DECIMAL(30,8)) <= ?
		AND market_cap_usd != ''
		AND symbol IN ?
		ORDER BY CAST(market_cap_usd AS DECIMAL(30,8)) DESC
		LIMIT ? OFFSET ?
	`
	err := s.db.WithContext(ctx).Raw(query, minCap, maxCap, symbols, limit, offset).Scan(&dataList).Error
	return dataList, err
}
