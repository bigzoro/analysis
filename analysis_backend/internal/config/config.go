package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy struct {
		Enable bool   `yaml:"enable"`
		All    string `yaml:"all_proxy"`
		HTTP   string `yaml:"http_proxy"`
		HTTPS  string `yaml:"https_proxy"`
		No     string `yaml:"no_proxy"`
	} `yaml:"proxy"`

	Pricing struct {
		Enable            bool              `yaml:"enable"`
		CoinGeckoEndpoint string            `yaml:"coingecko_endpoint"`
		Map               map[string]string `yaml:"map"`
	} `yaml:"pricing"`

	CoinCap struct {
		SymbolToAssetID map[string]string `yaml:"symbol_to_asset_id"`
	} `yaml:"coincap"`

	DataSources struct {
		NewsAPI struct {
			APIKey string `yaml:"api_key"`
		} `yaml:"newsapi"`
		LunarCrush struct {
			APIKey string `yaml:"api_key"`
		} `yaml:"lunarcrush"`
		CoinGecko struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"coingecko"`
		// BinanceFutures 相关配置已统一到 exchange.binance 下
		// 代码会自动使用 exchange.binance 的配置进行数据获取
	} `yaml:"data_sources"`

	Chains []struct {
		Name    string       `yaml:"name"`
		Type    string       `yaml:"type"` // bitcoin/evm/solana/tron
		RPC     string       `yaml:"rpc,omitempty"`
		Esplora string       `yaml:"esplora,omitempty"`
		ERC20   []TokenERC20 `yaml:"erc20,omitempty"`
		SPL     []TokenSPL   `yaml:"spl,omitempty"`
		TRC20   []TokenTRC20 `yaml:"trc20,omitempty"`
	} `yaml:"chains"`

	Entities []EntityCfg `yaml:"entities"`

	Services struct {
		EnableDataAnalysis bool `yaml:"enable_data_analysis"` // 是否启用数据分析服务（AI分析模块）
	} `yaml:"services"`

	Backtest struct {
		Mode string `yaml:"mode"` // "full" or "lightweight"
	} `yaml:"backtest"`

	Database struct {
		DSN          string `yaml:"dsn"`
		Automigrate  bool   `yaml:"automigrate"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
	} `yaml:"database"`

	Twitter struct {
		Bearer          string   `yaml:"bearer"`
		MonitorUsers    []string `yaml:"monitor_users"`    // 扫描器用
		IntervalSeconds int      `yaml:"interval_seconds"` // 扫描器用
	} `yaml:"twitter"`

	Redis struct {
		Enable   bool   `yaml:"enable"`
		Addr     string `yaml:"addr"`     // 例如: localhost:6379
		Password string `yaml:"password"` // 密码，空字符串表示无密码
		DB       int    `yaml:"db"`       // 数据库编号，默认 0
	} `yaml:"redis"`

	Arkham struct {
		BaseURL         string `yaml:"base_url"`
		APIKey          string `yaml:"api_key"`
		APISecret       string `yaml:"api_secret"`
		IntervalSeconds int    `yaml:"interval_seconds"`
	} `yaml:"arkham"`

	Nansen struct {
		BaseURL         string `yaml:"base_url"`
		APIKey          string `yaml:"api_key"`
		APISecret       string `yaml:"api_secret"`
		IntervalSeconds int    `yaml:"interval_seconds"`
	} `yaml:"nansen"`

	WhaleMonitoring struct {
		Arkham struct {
			BaseURL         string `yaml:"base_url"`
			APIKey          string `yaml:"api_key"`
			APISecret       string `yaml:"api_secret"`
			IntervalSeconds int    `yaml:"interval_seconds"`
		} `yaml:"arkham"`
		Nansen struct {
			BaseURL         string `yaml:"base_url"`
			APIKey          string `yaml:"api_key"`
			APISecret       string `yaml:"api_secret"`
			IntervalSeconds int    `yaml:"interval_seconds"`
		} `yaml:"nansen"`
	} `yaml:"whale_monitoring"`

	DataQuality struct {
		AlertThresholds struct {
			MaxFreshnessSeconds    int64   `yaml:"max_freshness_seconds"`
			MinCompletenessPercent float64 `yaml:"min_completeness_percent"`
			MaxErrorRatePercent    float64 `yaml:"max_error_rate_percent"`
			MinAccuracyPercent     float64 `yaml:"min_accuracy_percent"`
		} `yaml:"alert_thresholds"`
		Fallback struct {
			System struct {
				Enabled             bool          `yaml:"enabled"`
				HealthCheckInterval time.Duration `yaml:"health_check_interval"`
				MaxHistorySize      int           `yaml:"max_history_size"`
			} `yaml:"system"`
			Strategy struct {
				CandidateFallback bool `yaml:"candidate_fallback"`
				ScannerFallback   bool `yaml:"scanner_fallback"`
				DataQualityRelax  bool `yaml:"data_quality_relax"`
			} `yaml:"strategy"`
		} `yaml:"fallback"`
	} `yaml:"data_quality"`

	Exchange struct {
		// 环境选择：testnet 或 mainnet
		Environment string `yaml:"environment"`
		Binance     struct {
			Testnet struct {
				APIKey    string `yaml:"api_key"`
				SecretKey string `yaml:"secret_key"`
				Enabled   bool   `yaml:"enabled"`
			} `yaml:"testnet"`
			Mainnet struct {
				APIKey    string `yaml:"api_key"`
				SecretKey string `yaml:"secret_key"`
				Enabled   bool   `yaml:"enabled"`
			} `yaml:"mainnet"`
			// 向后兼容字段 - 用于获取当前环境的配置
			APIKey    string `yaml:"-"`
			SecretKey string `yaml:"-"`
			IsTestnet bool   `yaml:"-"` // 重命名为 IsTestnet 避免冲突
		} `yaml:"binance"`
	} `yaml:"exchange"`

	// 通知服务配置
	Notification struct {
		Enabled bool `yaml:"enabled"` // 是否启用通知功能，默认关闭
		SMTP    struct {
			Server    string `yaml:"server"`
			Port      int    `yaml:"port"`
			Username  string `yaml:"username"`
			Password  string `yaml:"password"`
			FromEmail string `yaml:"from_email"`
		} `yaml:"smtp"`
		SMS struct {
			APIKey    string `yaml:"api_key"`
			APISecret string `yaml:"api_secret"`
			Sender    string `yaml:"sender"`
		} `yaml:"sms"`
	} `yaml:"notification"`

	GridTrading struct {
		SimulationMode         bool    `yaml:"simulation_mode"`           // 是否启用模拟交易模式
		MaxSingleOrderAmount   float64 `yaml:"max_single_order_amount"`   // 单笔订单最大金额(USDT)
		MaxDailyTradingVolume  float64 `yaml:"max_daily_trading_volume"`  // 日最大交易量(USDT)
		EmergencyStopEnabled   bool    `yaml:"emergency_stop_enabled"`    // 是否启用紧急停止
		OrderRetryAttempts     int     `yaml:"order_retry_attempts"`      // 订单重试次数
		OrderRetryDelaySeconds int     `yaml:"order_retry_delay_seconds"` // 重试间隔(秒)
		RiskLimits             struct {
			MaxDrawdownPercent float64 `yaml:"max_drawdown_percent"` // 最大回撤百分比
			MaxPositionSize    float64 `yaml:"max_position_size"`    // 最大持仓比例
			MinOrderAmount     float64 `yaml:"min_order_amount"`     // 最小订单金额
			MaxOrderAmount     float64 `yaml:"max_order_amount"`     // 最大订单金额
		} `yaml:"risk_limits"`
		PerformanceMonitoring struct {
			EnableMetrics          bool    `yaml:"enable_metrics"`           // 启用性能指标收集
			MetricsIntervalMinutes int     `yaml:"metrics_interval_minutes"` // 指标收集间隔(分钟)
			AlertWinRateThreshold  float64 `yaml:"alert_win_rate_threshold"` // 胜率告警阈值
		} `yaml:"performance_monitoring"`
	} `yaml:"grid_trading"`
}

type EntityCfg struct {
	Name     string              `yaml:"name"`
	Networks map[string][]string `yaml:"networks"`
}

type TokenERC20 struct{ Symbol, Address string }
type TokenSPL struct{ Symbol, Mint string }
type TokenTRC20 struct{ Symbol, Contract string }

func MustLoad(path string, out *Config) {
	// 设置默认值
	setDefaults(out)

	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, out); err != nil {
			panic(err)
		}
	}

	// 初始化配置（设置向后兼容字段）
	initializeConfig(out)

	// 验证配置
	validateConfig(out)
}

// initializeConfig 初始化配置，设置向后兼容字段
func initializeConfig(cfg *Config) {
	// 根据环境选择设置当前使用的配置
	switch cfg.Exchange.Environment {
	case "mainnet":
		if cfg.Exchange.Binance.Mainnet.Enabled {
			cfg.Exchange.Binance.APIKey = cfg.Exchange.Binance.Mainnet.APIKey
			cfg.Exchange.Binance.SecretKey = cfg.Exchange.Binance.Mainnet.SecretKey
			cfg.Exchange.Binance.IsTestnet = false
		} else {
			// 如果mainnet未启用，回退到testnet
			cfg.Exchange.Binance.APIKey = cfg.Exchange.Binance.Testnet.APIKey
			cfg.Exchange.Binance.SecretKey = cfg.Exchange.Binance.Testnet.SecretKey
			cfg.Exchange.Binance.IsTestnet = true
			cfg.Exchange.Environment = "testnet" // 同步环境状态
		}
	case "testnet", "":
		cfg.Exchange.Binance.APIKey = cfg.Exchange.Binance.Testnet.APIKey
		cfg.Exchange.Binance.SecretKey = cfg.Exchange.Binance.Testnet.SecretKey
		cfg.Exchange.Binance.IsTestnet = true
		cfg.Exchange.Environment = "testnet" // 设置默认值
	default:
		log.Printf("[WARN] 未知的环境配置: %s，使用testnet作为默认环境", cfg.Exchange.Environment)
		cfg.Exchange.Binance.APIKey = cfg.Exchange.Binance.Testnet.APIKey
		cfg.Exchange.Binance.SecretKey = cfg.Exchange.Binance.Testnet.SecretKey
		cfg.Exchange.Binance.IsTestnet = true
		cfg.Exchange.Environment = "testnet"
	}
}

// validateConfig 验证配置
func validateConfig(cfg *Config) {
	// 验证测试环境配置
	if cfg.Exchange.Binance.Testnet.APIKey == "" || cfg.Exchange.Binance.Testnet.SecretKey == "" {
		log.Printf("[WARN] exchange.binance.testnet API密钥未配置，测试环境功能将受限")
	}

	// 验证生产环境配置
	if cfg.Exchange.Binance.Mainnet.APIKey == "" || cfg.Exchange.Binance.Mainnet.SecretKey == "" {
		log.Printf("[WARN] exchange.binance.mainnet API密钥未配置，生产环境无法使用")
		log.Printf("[INFO] 如需使用生产环境，请在 exchange.binance.mainnet 下配置API密钥")
	}

	// 检查当前环境配置
	if cfg.Exchange.Binance.APIKey == "" || cfg.Exchange.Binance.SecretKey == "" {
		env := cfg.Exchange.Environment
		log.Printf("[ERROR] 当前环境 (%s) 的API密钥配置不完整，Binance相关功能将无法正常工作", env)
		log.Printf("[INFO] 请检查 exchange.binance.%s 的配置", env)
	} else {
		env := cfg.Exchange.Environment
		log.Printf("[INFO] 当前使用环境: %s", env)
	}
}

// setDefaults 设置配置默认值
func setDefaults(cfg *Config) {
	// 服务开关默认值
	cfg.Services.EnableDataAnalysis = true // 默认启用数据分析服务

	// 数据质量降级默认值
	cfg.DataQuality.Fallback.System.Enabled = true              // 系统级降级默认启用
	cfg.DataQuality.Fallback.Strategy.CandidateFallback = false // 候选币种降级默认关闭
	cfg.DataQuality.Fallback.Strategy.ScannerFallback = false   // 扫描器降级默认关闭
	cfg.DataQuality.Fallback.Strategy.DataQualityRelax = false  // 数据质量放宽默认关闭

	// 交易所配置默认值
	cfg.Exchange.Environment = "testnet"         // 默认使用测试环境
	cfg.Exchange.Binance.Testnet.Enabled = true  // 测试环境默认启用
	cfg.Exchange.Binance.Mainnet.Enabled = false // 生产环境默认禁用

	// 网格交易配置默认值
	cfg.GridTrading.SimulationMode = true          // 默认启用模拟模式，确保安全
	cfg.GridTrading.MaxSingleOrderAmount = 100.0   // 单笔订单最大100USDT
	cfg.GridTrading.MaxDailyTradingVolume = 1000.0 // 日最大交易量1000USDT
	cfg.GridTrading.EmergencyStopEnabled = true    // 默认启用紧急停止
	cfg.GridTrading.OrderRetryAttempts = 3         // 订单重试3次
	cfg.GridTrading.OrderRetryDelaySeconds = 1     // 重试间隔1秒

	// 网格交易风险限制默认值
	cfg.GridTrading.RiskLimits.MaxDrawdownPercent = 20.0 // 最大回撤20%
	cfg.GridTrading.RiskLimits.MaxPositionSize = 50.0    // 最大持仓比例50%
	cfg.GridTrading.RiskLimits.MinOrderAmount = 10.0     // 最小订单金额10USDT
	cfg.GridTrading.RiskLimits.MaxOrderAmount = 500.0    // 最大订单金额500USDT

	// 网格交易性能监控默认值
	cfg.GridTrading.PerformanceMonitoring.EnableMetrics = true        // 默认启用性能指标收集
	cfg.GridTrading.PerformanceMonitoring.MetricsIntervalMinutes = 5  // 每5分钟收集一次指标
	cfg.GridTrading.PerformanceMonitoring.AlertWinRateThreshold = 0.4 // 胜率低于40%时告警
}

func ApplyProxy(cfg *Config) {
	if !cfg.Proxy.Enable {
		return
	}
	if cfg.Proxy.All != "" {
		os.Setenv("ALL_PROXY", cfg.Proxy.All)
	}
	if cfg.Proxy.HTTP != "" {
		os.Setenv("HTTP_PROXY", cfg.Proxy.HTTP)
	}
	if cfg.Proxy.HTTPS != "" {
		os.Setenv("HTTPS_PROXY", cfg.Proxy.HTTPS)
	}
	if cfg.Proxy.No != "" {
		os.Setenv("NO_PROXY", cfg.Proxy.No)
	}
}

type ChainCfg struct {
	Name, Type, RPC, Esplora string
	ERC20                    []TokenERC20
	SPL                      []TokenSPL
	TRC20                    []TokenTRC20
}

func BuildChainCfg(cfg *Config) map[string]ChainCfg {
	out := map[string]ChainCfg{}
	for _, c := range cfg.Chains {
		out[c.Name] = ChainCfg{
			Name:    c.Name,
			Type:    c.Type,
			RPC:     c.RPC,
			Esplora: c.Esplora,
			ERC20:   c.ERC20,
			SPL:     c.SPL,
			TRC20:   c.TRC20,
		}
	}
	// 兜底
	if _, ok := out["bitcoin"]; !ok {
		out["bitcoin"] = ChainCfg{Name: "bitcoin", Type: "bitcoin", Esplora: "https://mempool.space/api,https://blockstream.info/api"}
	}
	if _, ok := out["ethereum"]; !ok {
		out["ethereum"] = ChainCfg{
			Name: "ethereum", Type: "evm", RPC: "https://eth.llamarpc.com",
			ERC20: []TokenERC20{
				{Symbol: "USDT", Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7"},
				{Symbol: "USDC", Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
			},
		}
	}
	if _, ok := out["solana"]; !ok {
		out["solana"] = ChainCfg{
			Name: "solana", Type: "solana", RPC: "https://api.mainnet-beta.solana.com",
			SPL: []TokenSPL{
				{Symbol: "USDT", Mint: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"},
				{Symbol: "USDC", Mint: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"},
			},
		}
	}
	return out
}
