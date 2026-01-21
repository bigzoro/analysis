package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		DSN          string `yaml:"dsn"`
		Automigrate  bool   `yaml:"automigrate"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
	} `yaml:"database"`
	Exchange struct {
		Binance struct {
			APIKey    string `yaml:"api_key"`
			SecretKey string `yaml:"secret_key"`
			Testnet   bool   `yaml:"testnet"`
		} `yaml:"binance"`
	} `yaml:"exchange"`
	GridTrading struct {
		SimulationMode bool `yaml:"simulation_mode"`
	} `yaml:"grid_trading"`
}

func main() {
	fmt.Println("🔍 检查网格交易模拟模式配置")
	fmt.Println("=====================================")

	// 1. 检查配置文件
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Printf("❌ 加载配置失败: %v", err)
	} else {
		fmt.Printf("✅ 配置文件加载成功\n")
		fmt.Printf("📋 配置内容:\n")
		if cfg.GridTrading.SimulationMode {
			fmt.Printf("  网格交易模拟模式: ✅ 启用\n")
		} else {
			fmt.Printf("  网格交易模拟模式: ❌ 禁用 (应该实际下单)\n")
		}

		if cfg.Exchange.Binance.APIKey != "" && cfg.Exchange.Binance.SecretKey != "" {
			fmt.Printf("  币安API密钥: ✅ 已配置\n")
			fmt.Printf("  测试网络: %v\n", cfg.Exchange.Binance.IsTestnet)
		} else {
			fmt.Printf("  币安API密钥: ❌ 未配置\n")
		}
	}

	// 2. 检查数据库中的策略配置
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

	// 查询策略配置
	var strategies []struct {
		ID                 uint   `json:"id"`
		Name               string `json:"name"`
		IsRunning          bool   `json:"is_running"`
		GridTradingEnabled bool   `json:"grid_trading_enabled"`
		UseSymbolWhitelist bool   `json:"use_symbol_whitelist"`
		SymbolWhitelist    string `json:"symbol_whitelist"`
	}

	query := `
		SELECT id, name, is_running, grid_trading_enabled,
			   use_symbol_whitelist, symbol_whitelist
		FROM trading_strategies
		WHERE grid_trading_enabled = true
	`

	err = gdb.Raw(query).Scan(&strategies).Error
	if err != nil {
		log.Fatalf("❌ 查询策略失败: %v", err)
	}

	fmt.Printf("\n📊 数据库中的网格交易策略:\n")
	for _, strategy := range strategies {
		fmt.Printf("  策略 #%d (%s):\n", strategy.ID, strategy.Name)
		fmt.Printf("    运行状态: %v\n", strategy.IsRunning)
		fmt.Printf("    网格交易: ✅ 启用\n")
		fmt.Printf("    白名单模式: %v\n", strategy.UseSymbolWhitelist)
		if strategy.UseSymbolWhitelist {
			fmt.Printf("    白名单: %s\n", strategy.SymbolWhitelist)
		}
	}

	// 3. 模拟GridOrderManager的isSimulationMode逻辑
	fmt.Printf("\n🎯 模拟GridOrderManager.isSimulationMode():\n")

	if cfg == nil {
		fmt.Printf("  配置对象: nil → 返回 true (模拟模式)\n")
		fmt.Printf("  ❌ 问题：配置未正确加载，网格交易使用模拟模式！\n")
	} else {
		fmt.Printf("  配置对象: 存在\n")
		simulationMode := cfg.GridTrading.SimulationMode
		fmt.Printf("  GridTrading.SimulationMode: %v\n", simulationMode)

		if simulationMode {
			fmt.Printf("  ❌ 返回: true (模拟模式) - 不会实际下单\n")
		} else {
			fmt.Printf("  ✅ 返回: false (实盘模式) - 应该实际下单\n")

			// 检查API密钥
			if cfg.Exchange.Binance.APIKey == "" || cfg.Exchange.Binance.SecretKey == "" {
				fmt.Printf("  ⚠️  警告: API密钥未配置，可能导致下单失败\n")
			}
		}
	}

	fmt.Printf("\n🔧 解决方案:\n")
	if cfg != nil && cfg.GridTrading.SimulationMode {
		fmt.Printf("  1. 在 config.yaml 中设置: grid_trading.simulation_mode: false\n")
	}
	if cfg != nil && (cfg.Exchange.Binance.APIKey == "" || cfg.Exchange.Binance.SecretKey == "") {
		fmt.Printf("  2. 在 config.yaml 中配置币安API密钥\n")
		fmt.Printf("     exchange.binance.api_key: \"你的API密钥\"\n")
		fmt.Printf("     exchange.binance.secret_key: \"你的密钥\"\n")
	}
	if cfg == nil {
		fmt.Printf("  3. 检查配置文件路径和格式是否正确\n")
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
