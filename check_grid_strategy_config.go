package main

import (
	"fmt"
	"log"
	"os"

	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		DSN          string `yaml:"dsn"`
		Automigrate  bool   `yaml:"automigrate"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
	} `yaml:"database"`
}

func main() {
	// 加载配置
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  false,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}

	fmt.Println("🔍 检查网格交易策略配置")
	fmt.Println("=====================================")

	// 查询所有网格交易策略
	var strategies []struct {
		ID                    uint    `json:"id"`
		Name                  string  `json:"name"`
		UserID                uint    `json:"user_id"`
		IsRunning             bool    `json:"is_running"`
		GridTradingEnabled    bool    `json:"grid_trading_enabled"`
		GridUpperPrice        float64 `json:"grid_upper_price"`
		GridLowerPrice        float64 `json:"grid_lower_price"`
		GridLevels            int     `json:"grid_levels"`
		GridInvestmentAmount  float64 `json:"grid_investment_amount"`
		UseSymbolWhitelist    bool    `json:"use_symbol_whitelist"`
		SymbolWhitelist       string  `json:"symbol_whitelist"`
	}

	query := `
		SELECT
			ts.id, ts.name, ts.user_id, ts.is_running,
			ts.grid_trading_enabled, ts.grid_upper_price, ts.grid_lower_price,
			ts.grid_levels, ts.grid_investment_amount,
			ts.use_symbol_whitelist, ts.symbol_whitelist
		FROM trading_strategies ts
		WHERE ts.grid_trading_enabled = true
		ORDER BY ts.id
	`

	err = gdb.Raw(query).Scan(&strategies).Error
	if err != nil {
		log.Fatalf("查询网格交易策略失败: %v", err)
	}

	if len(strategies) == 0 {
		fmt.Println("❌ 未找到启用的网格交易策略")
		return
	}

	fmt.Printf("📊 找到 %d 个网格交易策略:\n\n", len(strategies))

	for i, strategy := range strategies {
		fmt.Printf("策略 #%d:\n", i+1)
		fmt.Printf("  ID: %d\n", strategy.ID)
		fmt.Printf("  名称: %s\n", strategy.Name)
		fmt.Printf("  用户ID: %d\n", strategy.UserID)
		fmt.Printf("  运行状态: %v\n", strategy.IsRunning)
		fmt.Printf("  网格交易启用: %v\n", strategy.GridTradingEnabled)
		fmt.Printf("  网格范围: [%.4f, %.4f]\n", strategy.GridLowerPrice, strategy.GridUpperPrice)
		fmt.Printf("  网格层数: %d\n", strategy.GridLevels)
		fmt.Printf("  投资金额: %.2f USDT\n", strategy.GridInvestmentAmount)
		fmt.Printf("  使用白名单: %v\n", strategy.UseSymbolWhitelist)
		if strategy.UseSymbolWhitelist {
			fmt.Printf("  白名单: %s\n", strategy.SymbolWhitelist)
		}
		fmt.Println()

		// 检查是否有未完成的执行记录
		var pendingExecutions int64
		err = gdb.Model(&struct{}{}).Table("strategy_executions").
			Where("strategy_id = ? AND status = 'pending'", strategy.ID).
			Count(&pendingExecutions).Error

		if err == nil && pendingExecutions > 0 {
			fmt.Printf("  ⚠️  有 %d 个待执行记录\n", pendingExecutions)
		} else {
			fmt.Printf("  ✅ 无待执行记录\n")
		}
		fmt.Println()
	}

	// 检查最近的执行日志
	fmt.Println("📋 最近的网格交易执行日志:")
	fmt.Println("=====================================")

	var executions []struct {
		ID         uint   `json:"id"`
		StrategyID uint   `json:"strategy_id"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		CreatedAt  string `json:"created_at"`
	}

	logQuery := `
		SELECT se.id, se.strategy_id, se.status, se.message, se.created_at
		FROM strategy_executions se
		INNER JOIN trading_strategies ts ON se.strategy_id = ts.id
		WHERE ts.grid_trading_enabled = true
		ORDER BY se.created_at DESC
		LIMIT 10
	`

	err = gdb.Raw(logQuery).Scan(&executions).Error
	if err != nil {
		log.Printf("查询执行日志失败: %v", err)
	} else {
		for _, exec := range executions {
			fmt.Printf("执行ID %d (策略 %d): %s - %s [%s]\n",
				exec.ID, exec.StrategyID, exec.Status, exec.Message, exec.CreatedAt)
		}
	}
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