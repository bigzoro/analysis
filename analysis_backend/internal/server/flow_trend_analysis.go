package server

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	pdb "analysis/internal/db"
)

// FlowTrendResult 资金流趋势分析结果
type FlowTrendResult struct {
	Flow24h      float64 // 24小时净流入（USD）
	Flow3d       float64 // 3天净流入（USD）
	Flow7d       float64 // 7天净流入（USD）
	Flow30d      float64 // 30天净流入（USD）
	Trend3d      float64 // 3天趋势（变化率，百分比）
	Trend7d      float64 // 7天趋势（变化率，百分比）
	Trend30d     float64 // 30天趋势（变化率，百分比）
	Acceleration float64 // 资金流加速度（3天趋势 - 7天趋势）
	Reversal     bool    // 是否出现反转信号
	Trend        string  // "increasing"/"decreasing"/"stable"/"reversing"
	LargeFlow    bool    // 是否有大额资金流入（单日>1000万USD）
}

// GetFlowTrendForSymbol 获取指定币种的资金流趋势分析
func (s *Server) GetFlowTrendForSymbol(ctx context.Context, baseSymbol string) (*FlowTrendResult, error) {
	today := time.Now().UTC()
	day30Ago := today.AddDate(0, 0, -30)

	// 查询最近30天的资金流数据
	var flows []pdb.DailyFlow
	err := s.db.DB().Where("coin = ? AND day >= ? AND day <= ?",
		baseSymbol,
		day30Ago.Format("2006-01-02"),
		today.Format("2006-01-02"),
	).Order("day ASC").Find(&flows).Error

	if err != nil {
		return nil, fmt.Errorf("查询资金流数据失败: %w", err)
	}

	if len(flows) == 0 {
		// 没有数据，返回零值
		return &FlowTrendResult{
			Trend: "stable",
		}, nil
	}

	// 获取币种价格（用于转换为USD）
	price, err := s.getCoinPrice(ctx, baseSymbol)
	if err != nil {
		// 如果无法获取价格，尝试直接使用Net字段（假设已经是USD）
		price = 1.0
	}

	// 按日期分组并转换为USD
	flowByDay := make(map[string]float64)
	largeFlowDays := make(map[string]bool)

	for _, flow := range flows {
		net, err := strconv.ParseFloat(flow.Net, 64)
		if err != nil {
			continue
		}

		// 转换为USD
		netUSD := net * price
		flowByDay[flow.Day] += netUSD

		// 检查是否为大额资金流入（单日>1000万USD）
		if netUSD > 10000000 {
			largeFlowDays[flow.Day] = true
		}
	}

	// 计算各时间段的净流入
	todayStr := today.Format("2006-01-02")
	day3Ago := today.AddDate(0, 0, -3).Format("2006-01-02")
	day7Ago := today.AddDate(0, 0, -7).Format("2006-01-02")
	day30AgoStr := day30Ago.Format("2006-01-02")

	flow24h := flowByDay[todayStr]
	flow3d := sumFlowForPeriod(flowByDay, day3Ago, todayStr)
	flow7d := sumFlowForPeriod(flowByDay, day7Ago, todayStr)
	flow30d := sumFlowForPeriod(flowByDay, day30AgoStr, todayStr)

	// 计算趋势（变化率）
	// 3天趋势 = (最近3天 - 前3天) / 前3天 * 100
	day6Ago := today.AddDate(0, 0, -6).Format("2006-01-02")
	prev3d := sumFlowForPeriod(flowByDay, day6Ago, day3Ago)
	trend3d := calculateTrend(flow3d, prev3d)

	// 7天趋势 = (最近7天 - 前7天) / 前7天 * 100
	day14Ago := today.AddDate(0, 0, -14).Format("2006-01-02")
	prev7d := sumFlowForPeriod(flowByDay, day14Ago, day7Ago)
	trend7d := calculateTrend(flow7d, prev7d)

	// 30天趋势 = (最近30天 - 前30天) / 前30天 * 100
	day60Ago := today.AddDate(0, 0, -60).Format("2006-01-02")
	prev30d := sumFlowForPeriod(flowByDay, day60Ago, day30AgoStr)
	trend30d := calculateTrend(flow30d, prev30d)

	// 计算加速度（3天趋势 - 7天趋势）
	acceleration := trend3d - trend7d

	// 判断趋势方向
	trend := determineFlowTrend(flow24h, flow3d, flow7d, trend3d, trend7d, acceleration)

	// 判断是否反转
	reversal := detectReversal(flowByDay, todayStr)

	// 判断是否有大额资金流入
	largeFlow := len(largeFlowDays) > 0

	return &FlowTrendResult{
		Flow24h:      flow24h,
		Flow3d:       flow3d,
		Flow7d:       flow7d,
		Flow30d:      flow30d,
		Trend3d:      trend3d,
		Trend7d:      trend7d,
		Trend30d:     trend30d,
		Acceleration: acceleration,
		Reversal:     reversal,
		Trend:        trend,
		LargeFlow:    largeFlow,
	}, nil
}

// GetFlowTrendForSymbols 批量获取多个币种的资金流趋势（性能优化）
func (s *Server) GetFlowTrendForSymbols(ctx context.Context, symbols []string) (map[string]*FlowTrendResult, error) {
	result := make(map[string]*FlowTrendResult)

	// 为每个币种查询趋势（可以进一步优化为批量查询）
	for _, symbol := range symbols {
		trend, err := s.GetFlowTrendForSymbol(ctx, symbol)
		if err != nil {
			// 如果某个币种查询失败，使用默认值
			result[symbol] = &FlowTrendResult{
				Trend: "stable",
			}
			continue
		}
		result[symbol] = trend
	}

	return result, nil
}

// sumFlowForPeriod 计算指定时间段内的资金流总和
func sumFlowForPeriod(flowByDay map[string]float64, startDay, endDay string) float64 {
	var total float64

	// 解析日期
	start, err := time.Parse("2006-01-02", startDay)
	if err != nil {
		return 0
	}
	end, err := time.Parse("2006-01-02", endDay)
	if err != nil {
		return 0
	}

	// 遍历日期范围内的每一天
	current := start
	for !current.After(end) {
		dayStr := current.Format("2006-01-02")
		if flow, ok := flowByDay[dayStr]; ok {
			total += flow
		}
		current = current.AddDate(0, 0, 1)
	}

	return total
}

// calculateTrend 计算趋势（变化率百分比）
func calculateTrend(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100 // 从0到正数，视为100%增长
		} else if current < 0 {
			return -100 // 从0到负数，视为100%下降
		}
		return 0
	}
	return ((current - previous) / math.Abs(previous)) * 100
}

// determineFlowTrend 判断资金流趋势方向
func determineFlowTrend(flow24h, flow3d, flow7d, trend3d, trend7d, acceleration float64) string {
	// 反转信号
	if acceleration < -50 && flow24h > 0 && flow3d < 0 {
		return "reversing" // 从流出转为流入
	}
	if acceleration > 50 && flow24h < 0 && flow3d > 0 {
		return "reversing" // 从流入转为流出
	}

	// 加速流入
	if acceleration > 20 && flow3d > 0 && flow7d > 0 {
		return "increasing"
	}

	// 加速流出
	if acceleration < -20 && flow3d < 0 && flow7d < 0 {
		return "decreasing"
	}

	// 稳定流入
	if flow3d > 0 && flow7d > 0 && math.Abs(acceleration) < 10 {
		return "stable"
	}

	// 稳定流出
	if flow3d < 0 && flow7d < 0 && math.Abs(acceleration) < 10 {
		return "stable"
	}

	// 默认稳定
	return "stable"
}

// detectReversal 检测资金流反转信号
func detectReversal(flowByDay map[string]float64, todayStr string) bool {
	// 获取最近7天的数据
	days := make([]string, 0, 7)
	today, _ := time.Parse("2006-01-02", todayStr)
	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		days = append(days, day)
	}

	// 检查是否有明显的反转模式
	// 反转模式：前3天流出，后3天流入（或相反）
	first3d := sumFlowForPeriod(flowByDay, days[6], days[3])
	last3d := sumFlowForPeriod(flowByDay, days[3], days[0])

	// 如果前3天和后3天的符号相反，且变化幅度>50%，视为反转
	if first3d*last3d < 0 {
		change := math.Abs((last3d - first3d) / math.Max(math.Abs(first3d), math.Abs(last3d)))
		if change > 0.5 {
			return true
		}
	}

	return false
}

// getCoinPrice 获取币种价格（USD）
func (s *Server) getCoinPrice(ctx context.Context, baseSymbol string) (float64, error) {
	// 优先使用CoinCap
	coinCapCache := newCoinCapCache()
	meta, err := coinCapCache.Get(ctx, baseSymbol)
	if err == nil && meta.MarketCapUSD != nil {
		if meta.Circulating != nil && *meta.Circulating > 0 {
			price := *meta.MarketCapUSD / *meta.Circulating
			return price, nil
		}
	}

	// 如果CoinCap失败，尝试从Binance市场数据获取
	var marketTop pdb.BinanceMarketTop
	err = s.db.DB().Where("symbol LIKE ?", baseSymbol+"%USDT").
		Order("created_at DESC").
		First(&marketTop).Error
	if err == nil {
		price, err := strconv.ParseFloat(marketTop.LastPrice, 64)
		if err == nil {
			return price, nil
		}
	}

	// 如果都失败，返回错误
	return 0, fmt.Errorf("无法获取币种价格: %s", baseSymbol)
}

// CalculateFlowScoreWithTrend 使用趋势数据计算资金流得分
// 公式：24h净流入 * 0.6 + 3天趋势 * 0.3 + 7天趋势 * 0.1
func CalculateFlowScoreWithTrend(flow24h, trend3d, trend7d float64) float64 {
	// 24h净流入得分（0-15分）
	var flow24hScore float64
	if flow24h > 10000000 { // 1000万
		flow24hScore = 15
	} else if flow24h > 5000000 { // 500万
		flow24hScore = 10 + (flow24h-5000000)/5000000*5
	} else if flow24h > 1000000 { // 100万
		flow24hScore = 5 + (flow24h-1000000)/4000000*5
	} else if flow24h > 0 {
		flow24hScore = flow24h / 1000000 * 5
	} else {
		flow24hScore = 0
	}

	// 3天趋势得分（0-10分）
	// 趋势>50%：10分，趋势>20%：7分，趋势>0%：5分，趋势<0%：0分
	var trend3dScore float64
	if trend3d > 50 {
		trend3dScore = 10
	} else if trend3d > 20 {
		trend3dScore = 7 + (trend3d-20)/30*3
	} else if trend3d > 0 {
		trend3dScore = 5 + (trend3d/20)*2
	} else {
		trend3dScore = 0
	}

	// 7天趋势得分（0-10分）
	var trend7dScore float64
	if trend7d > 50 {
		trend7dScore = 10
	} else if trend7d > 20 {
		trend7dScore = 7 + (trend7d-20)/30*3
	} else if trend7d > 0 {
		trend7dScore = 5 + (trend7d/20)*2
	} else {
		trend7dScore = 0
	}

	// 加权平均：24h * 0.6 + 3天趋势 * 0.3 + 7天趋势 * 0.1
	totalScore := flow24hScore*0.6 + trend3dScore*0.3 + trend7dScore*0.1

	return math.Min(25, totalScore) // 资金流因子最高25分
}

