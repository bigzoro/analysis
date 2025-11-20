package server

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	pdb "analysis/internal/db"
)

// SentimentResult 情绪分析结果
type SentimentResult struct {
	Score        float64 // 情绪得分 0-10
	Positive     int    // 正面推文数
	Negative     int    // 负面推文数
	Neutral      int    // 中性推文数
	Total        int    // 总推文数
	Mentions     int    // 提及次数
	Trend        string // "bullish"/"bearish"/"neutral"
	KeyPhrases   []string // 关键短语
}

// 正面关键词（英文）
var positiveKeywords = []string{
	"bullish", "moon", "pump", "buy", "buying", "long", "hodl", "hold",
	"gains", "profit", "surge", "rally", "breakout", "breakthrough",
	"adoption", "partnership", "upgrade", "launch", "listing",
	"amazing", "great", "excellent", "strong", "solid", "bull run",
	"🚀", "📈", "💎", "🔥", "💪", "✅", "🎯",
}

// 负面关键词（英文）
var negativeKeywords = []string{
	"bearish", "dump", "crash", "sell", "selling", "short", "fud",
	"loss", "drop", "fall", "decline", "bear market", "correction",
	"hack", "scam", "rug", "exit", "delist", "ban", "warning",
	"bad", "terrible", "weak", "dump", "crash", "bear run",
	"📉", "🔻", "⚠️", "❌", "💀", "🚨",
}

// 中性关键词（用于过滤噪音）
var neutralKeywords = []string{
	"update", "announcement", "news", "info", "analysis", "review",
}

// GetTwitterSentimentForSymbol 获取指定币种的Twitter情绪分析
// 查询最近24小时内包含该币种符号的推文，并分析情绪
func (s *Server) GetTwitterSentimentForSymbol(ctx context.Context, baseSymbol string) (*SentimentResult, error) {
	// 查询最近24小时的推文
	since := time.Now().UTC().Add(-24 * time.Hour)
	
	// 构建查询：查找包含币种符号的推文
	// 使用LIKE查询，匹配币种符号（考虑大小写不敏感）
	symbolUpper := strings.ToUpper(baseSymbol)
	symbolLower := strings.ToLower(baseSymbol)
	
	var posts []pdb.TwitterPost
	err := s.db.DB().Where(
		"tweet_time >= ? AND (text LIKE ? OR text LIKE ? OR text LIKE ?)",
		since,
		fmt.Sprintf("%%%s%%", symbolUpper), // BTC
		fmt.Sprintf("%%%s%%", symbolLower), // btc
		fmt.Sprintf("%%$%s%%", symbolUpper), // $BTC
	).Order("tweet_time DESC").
		Limit(500). // 最多分析500条推文
		Find(&posts).Error
	
	if err != nil {
		return nil, fmt.Errorf("查询推文失败: %w", err)
	}
	
	if len(posts) == 0 {
		// 没有推文，返回中性得分
		return &SentimentResult{
			Score:   5.0,
			Neutral: 0,
			Total:   0,
			Mentions: 0,
			Trend:   "neutral",
		}, nil
	}
	
	// 分析每条推文的情绪
	positive := 0
	negative := 0
	neutral := 0
	keyPhrases := make(map[string]int)
	
	for _, post := range posts {
		sentiment := analyzeTweetSentiment(post.Text, baseSymbol)
		switch sentiment {
		case "positive":
			positive++
		case "negative":
			negative++
		default:
			neutral++
		}
		
		// 提取关键短语
		phrases := extractKeyPhrases(post.Text, baseSymbol)
		for _, phrase := range phrases {
			keyPhrases[phrase]++
		}
	}
	
	total := len(posts)
	
	// 计算情绪得分：0-10分
	// 公式：(正面数 - 负面数) / 总数 * 5 + 5
	var score float64
	if total > 0 {
		diff := float64(positive - negative)
		score = (diff / float64(total)) * 5 + 5
		score = math.Max(0, math.Min(10, score)) // 限制在0-10之间
	} else {
		score = 5.0
	}
	
	// 判断趋势
	var trend string
	if score > 6.5 {
		trend = "bullish"
	} else if score < 3.5 {
		trend = "bearish"
	} else {
		trend = "neutral"
	}
	
	// 获取最频繁的关键短语（最多5个）
	topPhrases := getTopPhrases(keyPhrases, 5)
	
	return &SentimentResult{
		Score:      score,
		Positive:   positive,
		Negative:   negative,
		Neutral:    neutral,
		Total:      total,
		Mentions:   total,
		Trend:      trend,
		KeyPhrases: topPhrases,
	}, nil
}

// analyzeTweetSentiment 分析单条推文的情绪
// 返回 "positive", "negative", "neutral"
func analyzeTweetSentiment(text, symbol string) string {
	textLower := strings.ToLower(text)
	
	// 检查是否真的提及了该币种（避免误匹配）
	if !containsSymbol(text, symbol) {
		return "neutral"
	}
	
	positiveCount := 0
	negativeCount := 0
	
	// 统计正面关键词
	for _, keyword := range positiveKeywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			positiveCount++
		}
	}
	
	// 统计负面关键词
	for _, keyword := range negativeKeywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			negativeCount++
		}
	}
	
	// 判断情绪
	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}
	
	return "neutral"
}

// containsSymbol 检查文本是否包含币种符号
func containsSymbol(text, symbol string) bool {
	textUpper := strings.ToUpper(text)
	symbolUpper := strings.ToUpper(symbol)
	
	// 匹配方式：
	// 1. 直接匹配：BTC
	// 2. 带$符号：$BTC
	// 3. 单词边界匹配（避免部分匹配）
	patterns := []string{
		fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(symbolUpper)),
		fmt.Sprintf("\\$%s\\b", regexp.QuoteMeta(symbolUpper)),
		fmt.Sprintf("#%s\\b", regexp.QuoteMeta(symbolUpper)),
	}
	
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, textUpper)
		if matched {
			return true
		}
	}
	
	return false
}

// extractKeyPhrases 从推文中提取关键短语
func extractKeyPhrases(text, symbol string) []string {
	phrases := make([]string, 0)
	textLower := strings.ToLower(text)
	
	// 提取包含币种符号的短语（最多10个词）
	words := strings.Fields(textLower)
	symbolLower := strings.ToLower(symbol)
	
	for i, word := range words {
		if strings.Contains(word, symbolLower) {
			// 提取前后各5个词作为短语
			start := max(0, i-5)
			end := min(len(words), i+6)
			phrase := strings.Join(words[start:end], " ")
			if len(phrase) > 0 && len(phrase) < 200 { // 限制长度
				phrases = append(phrases, phrase)
			}
		}
	}
	
	return phrases
}

// getTopPhrases 获取最频繁的短语
func getTopPhrases(phraseMap map[string]int, limit int) []string {
	type phraseCount struct {
		phrase string
		count  int
	}
	
	counts := make([]phraseCount, 0, len(phraseMap))
	for phrase, count := range phraseMap {
		counts = append(counts, phraseCount{phrase: phrase, count: count})
	}
	
	// 按频率排序
	for i := 0; i < len(counts)-1; i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}
	
	// 取前N个
	result := make([]string, 0, limit)
	for i := 0; i < len(counts) && i < limit; i++ {
		if counts[i].count > 0 {
			result = append(result, counts[i].phrase)
		}
	}
	
	return result
}

// GetTwitterSentimentForSymbols 批量获取多个币种的情绪分析（优化性能）
func (s *Server) GetTwitterSentimentForSymbols(ctx context.Context, symbols []string) (map[string]*SentimentResult, error) {
	result := make(map[string]*SentimentResult)
	
	// 为每个币种查询情绪（可以优化为批量查询）
	for _, symbol := range symbols {
		sentiment, err := s.GetTwitterSentimentForSymbol(ctx, symbol)
		if err != nil {
			// 如果某个币种查询失败，使用默认值
			result[symbol] = &SentimentResult{
				Score:  5.0,
				Trend:  "neutral",
				Total:  0,
			}
			continue
		}
		result[symbol] = sentiment
	}
	
	return result, nil
}

// GetTwitterSentimentFromHistory 从历史推文数据计算情绪（如果实时查询失败）
func (s *Server) GetTwitterSentimentFromHistory(ctx context.Context, baseSymbol string, days int) (*SentimentResult, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	
	symbolUpper := strings.ToUpper(baseSymbol)
	
	var posts []pdb.TwitterPost
	err := s.db.DB().Where(
		"tweet_time >= ? AND (text LIKE ? OR text LIKE ? OR text LIKE ?)",
		since,
		fmt.Sprintf("%%%s%%", symbolUpper),
		fmt.Sprintf("%%%s%%", strings.ToLower(baseSymbol)),
		fmt.Sprintf("%%$%s%%", symbolUpper),
	).Order("tweet_time DESC").
		Limit(1000).
		Find(&posts).Error
	
	if err != nil {
		return nil, fmt.Errorf("查询历史推文失败: %w", err)
	}
	
	if len(posts) == 0 {
		return &SentimentResult{
			Score:  5.0,
			Trend:  "neutral",
			Total:  0,
		}, nil
	}
	
	// 分析情绪
	positive := 0
	negative := 0
	neutral := 0
	
	for _, post := range posts {
		sentiment := analyzeTweetSentiment(post.Text, baseSymbol)
		switch sentiment {
		case "positive":
			positive++
		case "negative":
			negative++
		default:
			neutral++
		}
	}
	
	total := len(posts)
	var score float64
	if total > 0 {
		diff := float64(positive - negative)
		score = (diff / float64(total)) * 5 + 5
		score = math.Max(0, math.Min(10, score))
	} else {
		score = 5.0
	}
	
	var trend string
	if score > 6.5 {
		trend = "bullish"
	} else if score < 3.5 {
		trend = "bearish"
	} else {
		trend = "neutral"
	}
	
	return &SentimentResult{
		Score:    score,
		Positive: positive,
		Negative: negative,
		Neutral:  neutral,
		Total:    total,
		Mentions: total,
		Trend:    trend,
	}, nil
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

