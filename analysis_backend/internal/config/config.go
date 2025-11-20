package config

import (
	"os"

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
}

type EntityCfg struct {
	Name     string              `yaml:"name"`
	Networks map[string][]string `yaml:"networks"`
}

type TokenERC20 struct{ Symbol, Address string }
type TokenSPL struct{ Symbol, Mint string }
type TokenTRC20 struct{ Symbol, Contract string }

func MustLoad(path string, out *Config) {
	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, out); err != nil {
			panic(err)
		}
	}
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
