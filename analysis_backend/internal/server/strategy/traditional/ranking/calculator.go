package ranking

import (
	"analysis/internal/server/strategy/traditional"
	"sort"
)

// Calculator 排名计算器实现
type Calculator struct{}

// NewCalculator 创建排名计算器
func NewCalculator() traditional.RankCalculator {
	return &Calculator{}
}

// CalculateGainersRank 计算涨幅排名
func (c *Calculator) CalculateGainersRank(candidates []traditional.CandidateWithRank) []traditional.CandidateWithRank {
	if len(candidates) == 0 {
		return candidates
	}

	// 复制切片避免修改原数据
	result := make([]traditional.CandidateWithRank, len(candidates))
	copy(result, candidates)

	// 按涨跌幅降序排序（涨幅最大的排在前面）
	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangePercent > result[j].ChangePercent
	})

	// 重新分配排名
	for i := range result {
		result[i].Rank = i + 1
	}

	return result
}

// CalculateVolumeRank 计算交易量排名
func (c *Calculator) CalculateVolumeRank(candidates []traditional.CandidateWithRank) []traditional.CandidateWithRank {
	if len(candidates) == 0 {
		return candidates
	}

	// 复制切片避免修改原数据
	result := make([]traditional.CandidateWithRank, len(candidates))
	copy(result, candidates)

	// 按交易量降序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Volume > result[j].Volume
	})

	// 重新分配排名
	for i := range result {
		result[i].Rank = i + 1
	}

	return result
}

// FilterByRank 按排名过滤
func (c *Calculator) FilterByRank(candidates []traditional.CandidateWithRank, maxRank int) []traditional.CandidateWithRank {
	if maxRank <= 0 {
		return candidates
	}

	var filtered []traditional.CandidateWithRank
	for _, candidate := range candidates {
		if candidate.Rank <= maxRank {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}
