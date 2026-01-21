package server

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	pdb "analysis/internal/db"
)

// AnnouncementScore 公告重要性评分结果
type AnnouncementScore struct {
	TotalScore    float64 // 总分 0-30
	CategoryScore float64 // 类型得分 0-10
	ExchangeScore float64 // 交易所得分 0-8
	HeatScore     float64 // 热度得分 0-6
	TimeScore     float64 // 时间衰减得分 0-4
	VerifiedBonus float64 // 验证加分 0-2
	Importance    string  // "high"/"medium"/"low"
	Details       string  // 公告详情（标题摘要）
	Exchange      string  // 交易所名称
}

// GetAnnouncementScoreForSymbol 获取指定币种的公告重要性评分
// 查询最近days天的公告，计算最高分的公告
func (s *Server) GetAnnouncementScoreForSymbol(ctx context.Context, baseSymbol string, days int) (*AnnouncementScore, error) {
	if days <= 0 {
		days = 7 // 默认7天
	}

	since := time.Now().UTC().AddDate(0, 0, -days)

	// 查询包含该币种的公告
	// 使用更精确的匹配方式
	symbolUpper := strings.ToUpper(baseSymbol)
	symbolLower := strings.ToLower(baseSymbol)

	var announcements []pdb.Announcement
	err := s.db.DB().Where("release_time >= ?", since).
		Where("(tags LIKE ? OR tags LIKE ? OR title LIKE ? OR title LIKE ? OR summary LIKE ? OR summary LIKE ?)",
			fmt.Sprintf("%%\"%s\"%%", symbolUpper), // JSON格式的tags
			fmt.Sprintf("%%\"%s\"%%", symbolLower),
			fmt.Sprintf("%%%s%%", symbolUpper),
			fmt.Sprintf("%%%s%%", symbolLower),
			fmt.Sprintf("%%%s%%", symbolUpper),
			fmt.Sprintf("%%%s%%", symbolLower),
		).
		Order("release_time DESC").
		Limit(50). // 最多查询50条，取最高分
		Find(&announcements).Error

	if err != nil {
		return nil, fmt.Errorf("查询公告失败: %w", err)
	}

	if len(announcements) == 0 {
		return nil, nil // 没有公告
	}

	// 计算每个公告的得分，取最高分
	var maxScore *AnnouncementScore
	maxTotalScore := 0.0

	for _, ann := range announcements {
		score := s.calculateAnnouncementScore(ann)
		if score.TotalScore > maxTotalScore {
			maxTotalScore = score.TotalScore
			maxScore = score
		}
	}

	return maxScore, nil
}

// GetAnnouncementScoresForSymbols 批量获取多个币种的公告评分（性能优化）
func (s *Server) GetAnnouncementScoresForSymbols(ctx context.Context, symbols []string, days int) (map[string]*AnnouncementScore, error) {
	result := make(map[string]*AnnouncementScore)

	// 为每个币种查询评分（可以进一步优化为批量查询）
	for _, symbol := range symbols {
		score, err := s.GetAnnouncementScoreForSymbol(ctx, symbol, days)
		if err != nil {
			// 如果某个币种查询失败，跳过
			continue
		}
		if score != nil {
			result[symbol] = score
		}
	}

	return result, nil
}

// calculateAnnouncementScore 计算单个公告的重要性得分
func (s *Server) calculateAnnouncementScore(ann pdb.Announcement) *AnnouncementScore {
	score := &AnnouncementScore{
		Details:  ann.Title,
		Exchange: ann.Exchange,
	}

	// 1. 类型得分（0-10分）
	score.CategoryScore = s.getCategoryScore(ann.Category, ann.IsEvent)

	// 2. 交易所得分（0-8分）
	score.ExchangeScore = s.getExchangeScore(ann.Exchange)

	// 3. 热度得分（0-6分）
	score.HeatScore = s.getHeatScore(ann.HeatScore, ann.IsEvent)

	// 4. 时间衰减得分（0-4分）
	score.TimeScore = s.getTimeScore(ann.ReleaseTime)

	// 5. 验证加分（0-2分）
	score.VerifiedBonus = s.getVerifiedBonus(ann.Verified)

	// 计算总分
	score.TotalScore = score.CategoryScore + score.ExchangeScore +
		score.HeatScore + score.TimeScore + score.VerifiedBonus

	// 限制最高30分
	score.TotalScore = math.Min(30, score.TotalScore)

	// 判断重要性等级
	if score.TotalScore >= 20 {
		score.Importance = "high"
	} else if score.TotalScore >= 10 {
		score.Importance = "medium"
	} else {
		score.Importance = "low"
	}

	return score
}

// getCategoryScore 计算类型得分
func (s *Server) getCategoryScore(category string, isEvent bool) float64 {
	// 如果是重要事件，直接给高分
	if isEvent {
		return 10.0
	}

	// 根据类型评分
	switch strings.ToLower(category) {
	case "newcoin":
		return 10.0 // 新币上线最重要
	case "listing":
		return 9.0 // 上线公告
	case "event":
		return 8.0 // 事件
	case "finance", "financial":
		return 7.0 // 金融相关
	case "update", "upgrade":
		return 6.0 // 更新/升级
	case "partnership":
		return 7.5 // 合作
	case "launch":
		return 8.5 // 发布
	default:
		return 5.0 // 其他类型
	}
}

// getExchangeScore 计算交易所得分
func (s *Server) getExchangeScore(exchange string) float64 {
	exchangeUpper := strings.ToUpper(exchange)

	// 根据交易所重要性评分
	switch exchangeUpper {
	case "BINANCE":
		return 8.0 // Binance最重要
	case "OKX", "OKEX":
		return 7.0
	case "BYBIT":
		return 6.5
	case "COINBASE":
		return 7.5
	case "KRAKEN":
		return 6.0
	case "UPBIT":
		return 6.0
	case "BITGET":
		return 5.5
	case "GATE.IO", "GATEIO":
		return 5.0
	case "KUCOIN":
		return 5.0
	case "HUOBI", "HTX":
		return 5.5
	default:
		// 如果有交易所名称但不在列表中，给基础分
		if exchangeUpper != "" {
			return 4.0
		}
		return 0.0 // 没有交易所信息
	}
}

// getHeatScore 计算热度得分
func (s *Server) getHeatScore(heatScore int, isEvent bool) float64 {
	// 如果是重要事件，基础热度更高
	baseHeat := 0.0
	if isEvent {
		baseHeat = 2.0
	}

	// HeatScore范围是0-100，转换为0-4分
	heatFloat := float64(heatScore)
	if heatFloat > 80 {
		return baseHeat + 4.0 // 极高热度
	} else if heatFloat > 60 {
		return baseHeat + 3.0 // 高热度
	} else if heatFloat > 40 {
		return baseHeat + 2.0 // 中等热度
	} else if heatFloat > 20 {
		return baseHeat + 1.0 // 低热度
	} else if heatFloat > 0 {
		return baseHeat + 0.5 // 极低热度
	}

	return baseHeat // 无热度数据
}

// getTimeScore 计算时间衰减得分
func (s *Server) getTimeScore(releaseTime time.Time) float64 {
	now := time.Now().UTC()
	age := now.Sub(releaseTime)

	// 时间衰减：越新越重要
	if age < 24*time.Hour {
		return 4.0 // 24小时内：满分
	} else if age < 3*24*time.Hour {
		return 3.0 // 3天内：3分
	} else if age < 7*24*time.Hour {
		return 2.0 // 7天内：2分
	} else if age < 14*24*time.Hour {
		return 1.0 // 14天内：1分
	} else {
		return 0.5 // 超过14天：0.5分
	}
}

// getVerifiedBonus 计算验证加分
func (s *Server) getVerifiedBonus(verified bool) float64 {
	if verified {
		return 2.0 // 官方验证，额外加2分
	}
	return 0.0
}

// GetAnnouncementScoreForSymbolsWithDetails 获取公告评分（包含详细信息）
// 返回每个币种的最佳公告及其评分
func (s *Server) GetAnnouncementScoreForSymbolsWithDetails(ctx context.Context, symbols []string, days int) (map[string]*AnnouncementScore, error) {
	result := make(map[string]*AnnouncementScore)

	for _, symbol := range symbols {
		score, err := s.GetAnnouncementScoreForSymbol(ctx, symbol, days)
		if err != nil {
			continue
		}
		if score != nil {
			result[symbol] = score
		}
	}

	return result, nil
}
