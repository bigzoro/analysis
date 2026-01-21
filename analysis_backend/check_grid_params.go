package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查网格交易参数设置")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}

	// 1. 检查策略的完整配置
	fmt.Printf("📊 策略完整配置:\n")
	var strategy struct {
		ID                   uint    `json:"id"`
		Name                 string  `json:"name"`
		GridTradingEnabled   bool    `json:"grid_trading_enabled"`
		GridUpperPrice       float64 `json:"grid_upper_price"`
		GridLowerPrice       float64 `json:"grid_lower_price"`
		GridLevels           int     `json:"grid_levels"`
		GridInvestmentAmount float64 `json:"grid_investment_amount"`
		GridProfitPercent    float64 `json:"grid_profit_percent"`
		GridStopLossEnabled  bool    `json:"grid_stop_loss_enabled"`
		GridStopLossPercent  float64 `json:"grid_stop_loss_percent"`
		GridRebalanceEnabled bool    `json:"grid_rebalance_enabled"`
		UseSymbolWhitelist   bool    `json:"use_symbol_whitelist"`
		SymbolWhitelist      string  `json:"symbol_whitelist"`
		DynamicPositioning   bool    `json:"dynamic_positioning"`
		MaxPositionSize      float64 `json:"max_position_size"`
	}

	err = gdb.Raw(`
		SELECT
			id, name, grid_trading_enabled, grid_upper_price, grid_lower_price,
			grid_levels, grid_investment_amount, grid_profit_percent,
			grid_stop_loss_enabled, grid_stop_loss_percent, grid_rebalance_enabled,
			use_symbol_whitelist, symbol_whitelist, dynamic_positioning, max_position_size
		FROM trading_strategies
		WHERE grid_trading_enabled = true AND id = 29
	`).Scan(&strategy).Error

	if err != nil {
		log.Fatalf("❌ 查询策略配置失败: %v", err)
	}

	fmt.Printf("  策略ID: %d\n", strategy.ID)
	fmt.Printf("  策略名称: %s\n", strategy.Name)
	fmt.Printf("  网格交易: ✅ 启用\n")
	fmt.Printf("  网格范围: [%.4f, %.4f]\n", strategy.GridLowerPrice, strategy.GridUpperPrice)
	fmt.Printf("  网格层数: %d\n", strategy.GridLevels)
	fmt.Printf("  投资金额: %.2f USDT\n", strategy.GridInvestmentAmount)
	fmt.Printf("  利润百分比: %.2f%%\n", strategy.GridProfitPercent)
	fmt.Printf("  止损启用: %v\n", strategy.GridStopLossEnabled)
	fmt.Printf("  止损百分比: %.2f%%\n", strategy.GridStopLossPercent)
	fmt.Printf("  再平衡: %v\n", strategy.GridRebalanceEnabled)
	fmt.Printf("  白名单模式: %v\n", strategy.UseSymbolWhitelist)
	if strategy.UseSymbolWhitelist {
		fmt.Printf("  白名单: %s\n", strategy.SymbolWhitelist)
	}
	fmt.Printf("  动态仓位: %v\n", strategy.DynamicPositioning)
	fmt.Printf("  最大仓位: %.2f%%\n", strategy.MaxPositionSize)

	// 2. 检查FILUSDT当前价格和网格位置
	fmt.Printf("\n💰 FILUSDT当前状态:\n")
	var filPrice struct {
		LastPrice float64 `json:"last_price"`
	}

	err = gdb.Raw(`
		SELECT last_price
		FROM binance_24h_stats
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&filPrice).Error

	if err != nil {
		log.Printf("❌ 查询价格失败: %v", err)
	} else {
		fmt.Printf("  当前价格: %.4f USDT\n", filPrice.LastPrice)

		// 计算网格位置
		gridSpacing := (strategy.GridUpperPrice - strategy.GridLowerPrice) / float64(strategy.GridLevels)
		gridLevel := int((filPrice.LastPrice - strategy.GridLowerPrice) / gridSpacing)
		if gridLevel >= strategy.GridLevels {
			gridLevel = strategy.GridLevels - 1
		}
		if gridLevel < 0 {
			gridLevel = 0
		}

		fmt.Printf("  网格间距: %.4f USDT\n", gridSpacing)
		fmt.Printf("  当前网格层: %d/%d\n", gridLevel, strategy.GridLevels)

		// 检查是否在网格范围内
		inRange := filPrice.LastPrice >= strategy.GridLowerPrice && filPrice.LastPrice <= strategy.GridUpperPrice
		if inRange {
			fmt.Printf("  价格状态: ✅ 在网格范围内\n")
		} else {
			fmt.Printf("  价格状态: ❌ 超出网格范围\n")
			if filPrice.LastPrice < strategy.GridLowerPrice {
				deviation := (strategy.GridLowerPrice - filPrice.LastPrice) / strategy.GridLowerPrice * 100
				fmt.Printf("    偏离下限: %.4f USDT (%.2f%%)\n",
					strategy.GridLowerPrice-filPrice.LastPrice, deviation)
			} else {
				deviation := (filPrice.LastPrice - strategy.GridUpperPrice) / strategy.GridUpperPrice * 100
				fmt.Printf("    偏离上限: %.4f USDT (%.2f%%)\n",
					filPrice.LastPrice-strategy.GridUpperPrice, deviation)
			}
		}
	}

	// 3. 检查技术指标
	fmt.Printf("\n📈 技术指标状态:\n")
	var indicators map[string]interface{}
	err = gdb.Raw(`
		SELECT indicators
		FROM technical_indicators_caches
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&indicators).Error

	if err != nil {
		log.Printf("❌ 查询技术指标失败: %v", err)
	} else if _, ok := indicators["indicators"]; ok {
		fmt.Printf("  ✅ 技术指标数据存在\n")
	} else {
		fmt.Printf("  ⚠️  技术指标数据不存在\n")
	}

	// 4. 分析可能的决策问题
	fmt.Printf("\n🔍 决策逻辑分析:\n")

	// 检查网格参数合理性
	gridRange := strategy.GridUpperPrice - strategy.GridLowerPrice
	gridSpacing := gridRange / float64(strategy.GridLevels)

	fmt.Printf("  网格总范围: %.4f USDT\n", gridRange)
	fmt.Printf("  网格间距: %.4f USDT\n", gridSpacing)
	fmt.Printf("  每层投资: %.4f USDT\n", strategy.GridInvestmentAmount/float64(strategy.GridLevels))

	// 检查参数是否合理
	if gridRange <= 0 {
		fmt.Printf("  ❌ 网格范围无效: 上限(%.4f) <= 下限(%.4f)\n",
			strategy.GridUpperPrice, strategy.GridLowerPrice)
	} else if strategy.GridLevels <= 0 {
		fmt.Printf("  ❌ 网格层数无效: %d\n", strategy.GridLevels)
	} else if strategy.GridInvestmentAmount <= 0 {
		fmt.Printf("  ❌ 投资金额无效: %.4f\n", strategy.GridInvestmentAmount)
	} else {
		fmt.Printf("  ✅ 网格参数看起来合理\n")
	}

	// 5. 建议的调试步骤
	fmt.Printf("\n💡 调试建议:\n")
	fmt.Printf("  1. 检查服务日志中的 'GridStrategy' 相关消息\n")
	fmt.Printf("  2. 查看决策评分计算过程\n")
	fmt.Printf("  3. 确认技术指标数据完整性\n")
	fmt.Printf("  4. 考虑临时降低决策阈值进行测试\n")
	fmt.Printf("  5. 检查是否有持仓冲突或风险控制限制\n")

	fmt.Printf("\n🛠️ 可能的解决方案:\n")
	fmt.Printf("  - 扩大网格范围以包含当前价格\n")
	fmt.Printf("  - 降低买入/卖出决策阈值\n")
	fmt.Printf("  - 检查技术指标计算是否正常\n")
	fmt.Printf("  - 确认没有其他策略条件限制\n")
}
