package selection

import (
	"analysis/internal/server/strategy/moving_average"
	"context"
)

// Selector 候选选择器实现
type Selector struct {
	// 这里可以注入依赖，如市场数据服务等
}

// NewSelector 创建候选选择器
func NewSelector() moving_average.CandidateSelector {
	return &Selector{}
}

// SelectByVolume 按交易量选择候选
func (s *Selector) SelectByVolume(ctx context.Context, maxCount int) ([]string, error) {
	// 这里实现按交易量选择逻辑
	// 暂时返回模拟数据
	candidates := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
		"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "LTCUSDT",
		"LINKUSDT", "UNIUSDT", "ALGOUSDT", "VETUSDT", "ICPUSDT",
		"FILUSDT", "TRXUSDT", "ETCUSDT", "XLMUSDT", "THETAUSDT",
	}

	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}

	return candidates, nil
}

// SelectByMarketCap 按市值选择候选
func (s *Selector) SelectByMarketCap(ctx context.Context, maxCount int) ([]string, error) {
	// 这里实现按市值选择逻辑
	// 暂时返回模拟数据
	candidates := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
		"XRPUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "LINKUSDT",
		"UNIUSDT", "ALGOUSDT", "VETUSDT", "ICPUSDT", "FILUSDT",
	}

	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}

	return candidates, nil
}

// FallbackToDefaults 降级到默认列表
func (s *Selector) FallbackToDefaults(maxCount int) ([]string, error) {
	// 默认的适合均线策略的币种列表
	defaultCandidates := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
		"DOTUSDT", "AVAXUSDT", "LTCUSDT", "LINKUSDT", "UNIUSDT",
		"ALGOUSDT", "VETUSDT", "ICPUSDT", "FILUSDT", "TRXUSDT",
		"ETCUSDT", "XLMUSDT", "THETAUSDT", "HBARUSDT", "FTMUSDT",
		"NEARUSDT", "FLOWUSDT", "SANDUSDT", "MANAUSDT", "ENJUSDT",
	}

	if len(defaultCandidates) > maxCount {
		defaultCandidates = defaultCandidates[:maxCount]
	}

	return defaultCandidates, nil
}
