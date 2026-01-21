package main

import (
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 策略条件结构体
type StrategyConditions struct {
	MeanReversionEnabled    bool    `json:"mean_reversion_enabled"`
	MeanReversionMode       string  `json:"mean_reversion_mode"`
	MeanReversionSubMode    string  `json:"mean_reversion_sub_mode"`
	MRBollingerBandsEnabled bool    `json:"mr_bollinger_bands_enabled"`
	MRRSIEnabled            bool    `json:"mr_rsi_enabled"`
	MRPriceChannelEnabled   bool    `json:"mr_price_channel_enabled"`
	MRPeriod                int     `json:"mr_period"`
	MRBollingerMultiplier   float64 `json:"mr_bollinger_multiplier"`
	MRRSIOversold           int     `json:"mr_rsi_oversold"`
	MRRSIOverbought         int     `json:"mr_rsi_overbought"`
	MRChannelPeriod         int     `json:"mr_channel_period"`
	MRMinReversionStrength  float64 `json:"mr_min_reversion_strength"`
	MRSignalMode            string  `json:"mr_signal_mode"`
}

// 策略结构体
type TradingStrategy struct {
	ID          uint               `json:"id"`
	UserID      uint               `json:"user_id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Conditions  StrategyConditions `json:"conditions" gorm:"type:json"`
}

func main() {
	fmt.Println("🔍 检查策略ID=30的均值回归设置")
	fmt.Println("===================================")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 查询策略ID=30
	var strategy TradingStrategy
	err = db.Table("trading_strategies").Where("id = ?", 30).First(&strategy).Error
	if err != nil {
		log.Fatalf("❌ 查询策略失败: %v", err)
	}

	fmt.Printf("\n📋 策略信息:\n")
	fmt.Printf("ID: %d\n", strategy.ID)
	fmt.Printf("名称: %s\n", strategy.Name)
	fmt.Printf("描述: %s\n", strategy.Description)
	fmt.Printf("用户ID: %d\n", strategy.UserID)

	fmt.Printf("\n🔄 均值回归策略设置:\n")
	fmt.Printf("启用状态: %v\n", strategy.Conditions.MeanReversionEnabled)
	fmt.Printf("策略模式: %s\n", strategy.Conditions.MeanReversionMode)
	fmt.Printf("子模式: %s\n", strategy.Conditions.MeanReversionSubMode)

	if strategy.Conditions.MeanReversionSubMode == "adaptive" {
		fmt.Println("✅ 成功！策略已更新为自适应模式")
	} else {
		fmt.Printf("❌ 策略子模式仍为: %s\n", strategy.Conditions.MeanReversionSubMode)
	}

	fmt.Printf("\n📊 技术指标设置:\n")
	fmt.Printf("布林带启用: %v\n", strategy.Conditions.MRBollingerBandsEnabled)
	fmt.Printf("RSI启用: %v\n", strategy.Conditions.MRRSIEnabled)
	fmt.Printf("价格通道启用: %v\n", strategy.Conditions.MRPriceChannelEnabled)
	fmt.Printf("计算周期: %d\n", strategy.Conditions.MRPeriod)
	fmt.Printf("布林带倍数: %.1f\n", strategy.Conditions.MRBollingerMultiplier)
	fmt.Printf("RSI超卖: %d\n", strategy.Conditions.MRRSIOversold)
	fmt.Printf("RSI超买: %d\n", strategy.Conditions.MRRSIOverbought)
	fmt.Printf("价格通道周期: %d\n", strategy.Conditions.MRChannelPeriod)
	fmt.Printf("最小回归强度: %.2f\n", strategy.Conditions.MRMinReversionStrength)
	fmt.Printf("信号模式: %s\n", strategy.Conditions.MRSignalMode)

	// 显示完整的JSON结构以供验证
	conditionsJSON, _ := json.MarshalIndent(strategy.Conditions, "", "  ")
	fmt.Printf("\n📄 完整条件配置:\n%s\n", string(conditionsJSON))

	fmt.Println("\n🎯 检查完成")
}
