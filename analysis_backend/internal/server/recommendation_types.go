package server

import "time"

// DynamicWeights 动态权重结构
type DynamicWeights struct {
	MarketWeight    float64 // 市场表现权重
	FlowWeight      float64 // 资金流权重
	HeatWeight      float64 // 市场热度权重
	EventWeight     float64 // 事件权重
	SentimentWeight float64 // 情绪权重
}

// MarketState 市场状态
type MarketState struct {
	State        string  // "bull" / "bear" / "sideways"
	AvgChange    float64 // 平均涨幅
	UpRatio      float64 // 上涨币种比例
	Volatility   float64 // 波动率
	VolumeChange float64 // 成交量变化
}

// RecommendationScore 推荐评分结构
type RecommendationScore struct {
	Symbol       string
	BaseSymbol   string
	TotalScore   float64
	StrategyType string // 策略类型: "LONG", "SHORT", "RANGE"
	Scores       Scores
	Data         struct {
		Price             float64
		PriceChange24h    float64
		Volume24h         float64
		MarketCapUSD      *float64
		NetFlow24h        float64
		HasNewListing     bool
		HasAnnouncement   bool
		TwitterMentions   int
		FlowTrend         *FlowTrendResult   // 资金流趋势数据
		AnnouncementScore *AnnouncementScore // 公告重要性得分
	}
	Technical  *TechnicalIndicators // 技术指标
	Prediction *PricePrediction     // 价格预测
	Risk       struct {
		VolatilityRisk float64  // 波动率风险 0-100
		LiquidityRisk  float64  // 流动性风险 0-100
		MarketRisk     float64  // 市场风险 0-100
		TechnicalRisk  float64  // 技术风险 0-100
		OverallRisk    float64  // 综合风险 0-100
		RiskLevel      string   // "low"/"medium"/"high"
		RiskWarnings   []string // 风险提示
	}
	Reasons             []string
	Confidence          float64
	RiskLevel           string
	ExpectedReturn      float64
	RecommendedPosition float64
	MarketCap           float64
	Volume24h           float64
	PriceChange24h      float64
	LastUpdated         time.Time
}

// Scores 推荐评分细分结构
type Scores struct {
	Market      float64
	Flow        float64
	Heat        float64
	Event       float64
	Sentiment   float64
	Risk        float64
	Momentum    float64
	Technical   float64
	Fundamental float64
}
