package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"analysis/internal/netutil"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type BinanceExchangeInfo struct {
	ID                         uint      `gorm:"primarykey" json:"id"`
	Symbol                     string    `gorm:"size:20;not null;index" json:"symbol"`
	Status                     string    `gorm:"size:20;not null" json:"status"`
	BaseAsset                  string    `gorm:"size:20;not null" json:"base_asset"`
	QuoteAsset                 string    `gorm:"size:20;not null" json:"quote_asset"`
	MarketType                 string    `gorm:"size:10;not null;default:spot" json:"market_type"`
	BaseAssetPrecision         int       `gorm:"not null" json:"base_asset_precision"`
	QuoteAssetPrecision        int       `gorm:"not null" json:"quote_asset_precision"`
	BaseCommissionPrecision    int       `gorm:"not null" json:"base_commission_precision"`
	QuoteCommissionPrecision   int       `gorm:"not null" json:"quote_commission_precision"`
	OrderTypes                 string    `gorm:"type:text" json:"order_types"`
	IcebergAllowed             bool      `gorm:"default:false" json:"iceberg_allowed"`
	OcoAllowed                 bool      `gorm:"default:false" json:"oco_allowed"`
	QuoteOrderQtyMarketAllowed bool      `gorm:"default:false" json:"quote_order_qty_market_allowed"`
	AllowTrailingStop          bool      `gorm:"default:false" json:"allow_trailing_stop"`
	CancelReplaceAllowed       bool      `gorm:"default:false" json:"cancel_replace_allowed"`
	IsSpotTradingAllowed       bool      `gorm:"default:true" json:"is_spot_trading_allowed"`
	IsMarginTradingAllowed     bool      `gorm:"default:false" json:"is_margin_trading_allowed"`
	Filters                    string    `gorm:"type:text" json:"filters"`
	Permissions                string    `gorm:"type:text" json:"permissions"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

func main() {
	fmt.Println("=== 初始化交易所信息 ===")

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("analysis_backend/analysis.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 从币安获取期货交易对信息
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response struct {
		Symbols []struct {
			Symbol                     string      `json:"symbol"`
			Status                     string      `json:"status"`
			BaseAsset                  string      `json:"baseAsset"`
			QuoteAsset                 string      `json:"quoteAsset"`
			BaseAssetPrecision         int         `json:"baseAssetPrecision"`
			QuoteAssetPrecision        int         `json:"quoteAssetPrecision"`
			BaseCommissionPrecision    int         `json:"baseCommissionPrecision"`
			QuoteCommissionPrecision   int         `json:"quoteCommissionPrecision"`
			OrderTypes                 []string    `json:"orderTypes"`
			IcebergAllowed             bool        `json:"icebergAllowed"`
			OcoAllowed                 bool        `json:"ocoAllowed"`
			QuoteOrderQtyMarketAllowed bool        `json:"quoteOrderQtyMarketAllowed"`
			AllowTrailingStop          bool        `json:"allowTrailingStop"`
			CancelReplaceAllowed       bool        `json:"cancelReplaceAllowed"`
			IsSpotTradingAllowed       bool        `json:"isSpotTradingAllowed"`
			IsMarginTradingAllowed     bool        `json:"isMarginTradingAllowed"`
			Filters                    interface{} `json:"filters"`
			Permissions                []string    `json:"permissions"`
		} `json:"symbols"`
	}

	url := "https://fapi.binance.com/fapi/v1/exchangeInfo"
	fmt.Printf("正在从 %s 获取交易对信息...\n", url)

	if err := netutil.GetJSON(ctx, url, &response); err != nil {
		log.Fatalf("获取交易所信息失败: %v", err)
	}

	fmt.Printf("✅ 获取到 %d 个交易对信息\n", len(response.Symbols))

	// 保存到数据库
	saved := 0
	for _, symbol := range response.Symbols {
		// 只处理TRADING状态的交易对
		if symbol.Status != "TRADING" {
			continue
		}

		// 将数组转换为JSON字符串
		orderTypesJSON, _ := json.Marshal(symbol.OrderTypes)
		permissionsJSON, _ := json.Marshal(symbol.Permissions)
		filtersJSON, _ := json.Marshal(symbol.Filters)

		info := BinanceExchangeInfo{
			Symbol:                     symbol.Symbol,
			Status:                     symbol.Status,
			BaseAsset:                  symbol.BaseAsset,
			QuoteAsset:                 symbol.QuoteAsset,
			MarketType:                 "futures",
			BaseAssetPrecision:         symbol.BaseAssetPrecision,
			QuoteAssetPrecision:        symbol.QuoteAssetPrecision,
			BaseCommissionPrecision:    symbol.BaseCommissionPrecision,
			QuoteCommissionPrecision:   symbol.QuoteCommissionPrecision,
			OrderTypes:                 string(orderTypesJSON),
			IcebergAllowed:             symbol.IcebergAllowed,
			OcoAllowed:                 symbol.OcoAllowed,
			QuoteOrderQtyMarketAllowed: symbol.QuoteOrderQtyMarketAllowed,
			AllowTrailingStop:          symbol.AllowTrailingStop,
			CancelReplaceAllowed:       symbol.CancelReplaceAllowed,
			IsSpotTradingAllowed:       symbol.IsSpotTradingAllowed,
			IsMarginTradingAllowed:     symbol.IsMarginTradingAllowed,
			Filters:                    string(filtersJSON),
			Permissions:                string(permissionsJSON),
			CreatedAt:                  time.Now(),
			UpdatedAt:                  time.Now(),
		}

		// 保存或更新
		result := db.Where("symbol = ?", symbol.Symbol).Assign(info).FirstOrCreate(&info)
		if result.Error != nil {
			log.Printf("保存 %s 失败: %v", symbol.Symbol, result.Error)
		} else {
			saved++
		}
	}

	fmt.Printf("✅ 成功保存 %d 个交易对信息到数据库\n", saved)

	// 验证保存的数据
	var count int64
	db.Model(&BinanceExchangeInfo{}).Count(&count)
	fmt.Printf("数据库中现在有 %d 个交易对信息\n", count)

	// 检查一些重要的交易对
	symbols := []string{"BTCUSDT", "ETHUSDT", "FILUSDT", "FHEUSDT", "RIVERUSDT"}
	for _, symbol := range symbols {
		var info BinanceExchangeInfo
		if err := db.Where("symbol = ?", symbol).First(&info).Error; err != nil {
			fmt.Printf("❌ %s 不存在: %v\n", symbol, err)
		} else {
			fmt.Printf("✅ %s 存在，过滤器长度: %d 字符\n", symbol, len(info.Filters))
		}
	}

	fmt.Println("\n🎯 初始化完成！现在精度调整应该可以正常工作了。")
}
