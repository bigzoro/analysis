package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 检查均值回归策略配置
func main() {
	// 数据库连接
	dsn := "root:password@tcp(localhost:3306)/crypto_analysis?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("🔍 检查均值回归策略配置")
	fmt.Println("==================================================")

	// 查询所有策略
	var strategies []struct {
		ID                      uint    `json:"id"`
		Name                    string  `json:"name"`
		MeanReversionEnabled    bool    `json:"mean_reversion_enabled"`
		MRSignalMode            string  `json:"mr_signal_mode"`
		MRBollingerBandsEnabled bool    `json:"mr_bollinger_bands_enabled"`
		MRRSIEnabled            bool    `json:"mr_rsi_enabled"`
		MRPriceChannelEnabled   bool    `json:"mr_price_channel_enabled"`
		MRPeriod                int     `json:"mr_period"`
		MRBollingerMultiplier   float64 `json:"mr_bollinger_multiplier"`
		MRRSIOverbought         int     `json:"mr_rsi_overbought"`
		MRRSIOversold           int     `json:"mr_rsi_oversold"`
	}

	err = db.Table("trading_strategies").Select(
		"id", "name", "mean_reversion_enabled", "mr_signal_mode",
		"mr_bollinger_bands_enabled", "mr_rsi_enabled", "mr_price_channel_enabled",
		"mr_period", "mr_bollinger_multiplier", "mr_rsi_overbought", "mr_rsi_oversold",
	).Scan(&strategies).Error

	if err != nil {
		log.Fatal("查询失败:", err)
	}

	fmt.Printf("找到 %d 个策略\n\n", len(strategies))

	for _, strategy := range strategies {
		fmt.Printf("📋 策略 ID: %d\n", strategy.ID)
		fmt.Printf("   名称: %s\n", strategy.Name)
		fmt.Printf("   均值回归启用: %v\n", strategy.MeanReversionEnabled)

		if strategy.MeanReversionEnabled {
			fmt.Printf("   信号模式: %s\n", strategy.MRSignalMode)
			fmt.Printf("   布林带启用: %v\n", strategy.MRBollingerBandsEnabled)
			fmt.Printf("   RSI启用: %v\n", strategy.MRRSIEnabled)
			fmt.Printf("   价格通道启用: %v\n", strategy.MRPriceChannelEnabled)
			fmt.Printf("   计算周期: %d\n", strategy.MRPeriod)
			fmt.Printf("   布林带倍数: %.1f\n", strategy.MRBollingerMultiplier)
			fmt.Printf("   RSI超买: %d\n", strategy.MRRSIOverbought)
			fmt.Printf("   RSI超卖: %d\n", strategy.MRRSIOversold)

			// 计算阈值
			minSignalStrength := 0.5 // 默认
			if strategy.MRSignalMode == "AGGRESSIVE" {
				minSignalStrength = 0.33
			} else if strategy.MRSignalMode == "CONSERVATIVE" {
				minSignalStrength = 0.67
			}

			fmt.Printf("   信号阈值: %.0f%%\n", minSignalStrength*100)

			// 计算启用指标数量
			enabledIndicators := 0
			if strategy.MRBollingerBandsEnabled {
				enabledIndicators++
			}
			if strategy.MRRSIEnabled {
				enabledIndicators++
			}
			if strategy.MRPriceChannelEnabled {
				enabledIndicators++
			}
			fmt.Printf("   启用指标数: %d\n", enabledIndicators)

		} else {
			fmt.Printf("   ⚠️  未启用均值回归策略\n")
		}

		fmt.Println()
	}
}
