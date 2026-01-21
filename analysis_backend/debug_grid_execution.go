package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Database struct {
		DSN          string `yaml:"dsn"`
		Automigrate  bool   `yaml:"automigrate"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
	} `yaml:"database"`
	GridTrading struct {
		SimulationMode bool `yaml:"simulation_mode"`
	} `yaml:"grid_trading"`
}

func main() {
	fmt.Println("🔍 调试网格交易执行问题")
	fmt.Println("=====================================")

	// 1. 检查配置文件
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	fmt.Printf("📋 配置状态:\n")
	fmt.Printf("  模拟模式: %v\n", cfg.GridTrading.SimulationMode)

	// 2. 检查数据库连接和策略状态
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  false,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}

	// 3. 检查当前运行的策略
	var strategies []struct {
		ID                    uint   `json:"id"`
		Name                  string `json:"name"`
		IsRunning             bool   `json:"is_running"`
		GridTradingEnabled    bool   `json:"grid_trading_enabled"`
		UseSymbolWhitelist    bool   `json:"use_symbol_whitelist"`
		SymbolWhitelist       string `json:"symbol_whitelist"`
		RunInterval           int    `json:"run_interval"`
		LastRunAt             *string `json:"last_run_at"`
	}

	query := `
		SELECT id, name, is_running, grid_trading_enabled,
			   use_symbol_whitelist, symbol_whitelist,
			   run_interval, last_run_at
		FROM trading_strategies
		WHERE grid_trading_enabled = true AND is_running = true
	`

	err = gdb.Raw(query).Scan(&strategies).Error
	if err != nil {
		log.Fatalf("❌ 查询策略失败: %v", err)
	}

	fmt.Printf("\n📊 运行中的网格策略:\n")
	if len(strategies) == 0 {
		fmt.Printf("  ❌ 没有运行中的网格交易策略\n")
	} else {
		for _, strategy := range strategies {
			fmt.Printf("  ✅ 策略 #%d: %s\n", strategy.ID, strategy.Name)
			fmt.Printf("    - 白名单模式: %v\n", strategy.UseSymbolWhitelist)
			fmt.Printf("    - 白名单: %s\n", strategy.SymbolWhitelist)
			fmt.Printf("    - 运行间隔: %d 分钟\n", strategy.RunInterval)
			fmt.Printf("    - 最后运行: %v\n", strategy.LastRunAt)
		}
	}

	// 4. 检查最近的策略执行记录
	var executions []struct {
		ID         uint   `json:"id"`
		StrategyID uint   `json:"strategy_id"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		CreatedAt  string `json:"created_at"`
	}

	execQuery := `
		SELECT id, strategy_id, status, message, created_at
		FROM strategy_executions
		WHERE strategy_id IN (
			SELECT id FROM trading_strategies WHERE grid_trading_enabled = true
		)
		ORDER BY created_at DESC
		LIMIT 5
	`

	err = gdb.Raw(execQuery).Scan(&executions).Error
	if err != nil {
		log.Printf("❌ 查询执行记录失败: %v", err)
	} else {
		fmt.Printf("\n📋 最近的策略执行记录:\n")
		for _, exec := range executions {
			fmt.Printf("  执行 #%d (策略 %d): %s - %s\n", exec.ID, exec.StrategyID, exec.Status, exec.CreatedAt)
			fmt.Printf("    消息: %s\n", exec.Message)
		}
	}

	// 5. 检查是否有待处理的执行
	var pendingExecutions int64
	err = gdb.Model(&struct{}{}).Table("strategy_executions").
		Where("status = 'pending'").Count(&pendingExecutions).Error

	if err == nil {
		fmt.Printf("\n⏳ 待处理的执行: %d 个\n", pendingExecutions)
	}

	// 6. 检查是否有订单记录
	var orderCount int64
	err = gdb.Model(&pdb.ScheduledOrder{}).Count(&orderCount).Error
	if err == nil {
		fmt.Printf("📦 总订单数: %d 个\n", orderCount)

		// 检查最近的订单
		var recentOrders int64
		err = gdb.Model(&pdb.ScheduledOrder{}).
			Where("created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)").
			Count(&recentOrders).Error
		if err == nil {
			fmt.Printf("🕒 最近1小时订单数: %d 个\n", recentOrders)
		}
	}

	// 7. 检查FILUSDT的价格数据
	var filPriceData map[string]interface{}
	err = gdb.Raw(`
		SELECT last_price
		FROM binance_24h_stats
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&filPriceData).Error

	if err != nil {
		log.Printf("❌ 查询FIL价格失败: %v", err)
	} else if len(filPriceData) > 0 {
		if price, ok := filPriceData["last_price"].(float64); ok {
			fmt.Printf("\n💰 FILUSDT当前价格: %.4f USDT\n", price)
		} else {
			fmt.Printf("\n💰 FILUSDT价格数据存在但格式异常\n")
		}
	} else {
		fmt.Printf("\n💰 未找到FILUSDT价格数据\n")
	}

	// 8. 诊断结论
	fmt.Printf("\n🔍 诊断结论:\n")
	if cfg.GridTrading.SimulationMode {
		fmt.Printf("  ❌ 配置问题: 模拟模式仍然启用\n")
		fmt.Printf("  🔧 解决方案: 修改 config.yaml 中的 simulation_mode 为 false\n")
	} else if len(strategies) == 0 {
		fmt.Printf("  ❌ 策略问题: 没有运行中的网格交易策略\n")
		fmt.Printf("  🔧 解决方案: 启用网格交易策略\n")
	} else {
		fmt.Printf("  ✅ 配置正确: 模拟模式已禁用，有运行中的策略，价格数据正常\n")
		fmt.Printf("  🤔 可能原因:\n")
		fmt.Printf("     - 服务可能需要重启才能读取新配置\n")
		fmt.Printf("     - 策略执行时间间隔可能还没到\n")
		fmt.Printf("     - 决策逻辑可能因为其他条件未满足\n")
	}

	fmt.Printf("\n💡 建议操作:\n")
	fmt.Printf("  1. 重启网格交易调度器服务\n")
	fmt.Printf("  2. 检查服务日志中的详细执行信息\n")
	fmt.Printf("  3. 手动触发策略执行进行测试\n")
	fmt.Printf("  4. 确认API密钥和余额充足\n")
}


func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &cfg, nil
}