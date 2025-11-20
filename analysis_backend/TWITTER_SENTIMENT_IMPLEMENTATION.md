# Twitter情绪分析功能实现说明

## 功能概述

已成功实现Twitter情绪分析功能，用于完善推荐系统中的情绪因子评分。该功能能够：

1. **查询相关推文**：根据币种符号查询最近24小时内包含该币种的推文
2. **情绪分析**：分析每条推文的情绪（正面/负面/中性）
3. **计算情绪得分**：基于正面和负面推文比例计算0-10分的情绪得分
4. **集成到推荐系统**：将情绪得分应用到推荐评分中

## 实现文件

### 1. `analysis_backend/internal/server/sentiment_analysis.go`（新建）

核心情绪分析模块，包含以下功能：

#### 主要函数

- **`GetTwitterSentimentForSymbol`**: 获取单个币种的Twitter情绪分析
  - 查询最近24小时的推文
  - 分析每条推文的情绪
  - 计算情绪得分和趋势
  - 提取关键短语

- **`GetTwitterSentimentForSymbols`**: 批量获取多个币种的情绪分析（性能优化）

- **`GetTwitterSentimentFromHistory`**: 从历史推文数据计算情绪（备用方案）

- **`analyzeTweetSentiment`**: 分析单条推文的情绪
  - 使用关键词匹配判断情绪
  - 支持正面/负面/中性分类

#### 情绪关键词库

**正面关键词**：
- bullish, moon, pump, buy, buying, long, hodl, hold
- gains, profit, surge, rally, breakout, breakthrough
- adoption, partnership, upgrade, launch, listing
- 🚀, 📈, 💎, 🔥, 💪, ✅, 🎯

**负面关键词**：
- bearish, dump, crash, sell, selling, short, fud
- loss, drop, fall, decline, bear market, correction
- hack, scam, rug, exit, delist, ban, warning
- 📉, 🔻, ⚠️, ❌, 💀, 🚨

#### 情绪得分计算公式

```
情绪得分 = (正面推文数 - 负面推文数) / 总推文数 * 5 + 5
```

- 得分范围：0-10分
- 5分表示中性
- >6.5分：看涨（bullish）
- <3.5分：看跌（bearish）
- 3.5-6.5分：中性（neutral）

### 2. `analysis_backend/internal/server/recommendation.go`（修改）

#### 修改内容

1. **在`generateRecommendations`中集成情绪分析**：
   ```go
   // 4.5. 获取Twitter情绪数据
   baseSymbols := make([]string, 0, len(candidates))
   // ... 收集所有候选币种的基础符号
   twitterSentimentData, err := s.GetTwitterSentimentForSymbols(ctx, baseSymbols)
   ```

2. **更新`calculateScore`函数签名**：
   - 添加`sentimentData *SentimentResult`参数
   - 使用实际的情绪得分替代默认5分

3. **更新`calculateScoreWithKind`函数**：
   - 传递情绪数据参数

4. **完善`generateReasons`函数**：
   - 添加Twitter情绪相关的推荐理由
   - 根据情绪得分和提及次数生成不同理由

## 功能特性

### 1. 智能币种匹配

- 支持多种匹配方式：
  - 直接匹配：`BTC`
  - 带$符号：`$BTC`
  - 带#标签：`#BTC`
  - 单词边界匹配（避免部分匹配）

### 2. 情绪分析算法

- **关键词匹配**：使用预定义的关键词库
- **情绪分类**：正面/负面/中性
- **得分计算**：基于正面和负面推文的比例
- **趋势判断**：自动判断看涨/看跌/中性

### 3. 性能优化

- **批量查询**：支持一次查询多个币种的情绪
- **限制查询数量**：最多分析500条推文，避免性能问题
- **缓存友好**：查询结果可以缓存

### 4. 容错处理

- 如果查询失败，使用默认值（5分）
- 如果推文数量太少（<10条），降低权重30%
- 支持从历史数据计算情绪（备用方案）

## 使用示例

### 在推荐系统中使用

```go
// 获取币种的Twitter情绪
sentiment, err := s.GetTwitterSentimentForSymbol(ctx, "BTC")
if err != nil {
    // 使用默认值
    sentiment = &SentimentResult{Score: 5.0, Trend: "neutral"}
}

// 使用情绪得分
score.Scores.Sentiment = sentiment.Score
score.Data.TwitterMentions = sentiment.Mentions
```

### 批量查询

```go
symbols := []string{"BTC", "ETH", "SOL"}
sentimentMap, err := s.GetTwitterSentimentForSymbols(ctx, symbols)
// sentimentMap["BTC"] 包含BTC的情绪分析结果
```

## 推荐理由示例

根据情绪分析结果，系统会自动生成推荐理由：

- **情绪积极**（得分>7）："Twitter情绪积极（提及120次，得分8.5）"
- **情绪负面**（得分<3）："Twitter情绪偏负面（提及80次，得分2.3），需谨慎"
- **讨论热度高**（提及>50次）："Twitter讨论热度较高（提及150次）"

## 数据流程

```
1. 推荐系统生成候选币种列表
   ↓
2. 批量查询所有候选币种的Twitter情绪
   ↓
3. 从数据库查询最近24小时的相关推文
   ↓
4. 分析每条推文的情绪（正面/负面/中性）
   ↓
5. 计算情绪得分（0-10分）
   ↓
6. 应用到推荐评分中（情绪因子10%权重）
   ↓
7. 生成包含情绪信息的推荐理由
```

## 注意事项

1. **数据依赖**：需要定期运行Twitter扫描器收集推文数据
2. **关键词库**：当前使用英文关键词，如需支持中文，需要扩展关键词库
3. **推文数量**：如果某个币种推文数量太少，情绪分析的准确性会降低
4. **API限制**：Twitter API有速率限制，批量查询时需要注意

## 后续优化建议

1. **NLP分析**：使用自然语言处理技术，提高情绪分析准确性
2. **中文支持**：添加中文关键词库
3. **KOL识别**：识别意见领袖的推文，给予更高权重
4. **情绪趋势**：分析情绪随时间的变化趋势
5. **多平台聚合**：整合Reddit、Telegram等平台的情绪数据

## 测试建议

1. **单元测试**：测试情绪分析函数
2. **集成测试**：测试与推荐系统的集成
3. **性能测试**：测试批量查询的性能
4. **准确性测试**：验证情绪分析的准确性

## 总结

Twitter情绪分析功能已成功集成到推荐系统中，能够：

- ✅ 自动查询和分析币种相关的Twitter推文
- ✅ 计算0-10分的情绪得分
- ✅ 应用到推荐评分中（10%权重）
- ✅ 生成包含情绪信息的推荐理由
- ✅ 支持批量查询，性能优化

该功能显著提升了推荐系统的情绪因子评分质量，从原来的固定5分变为基于实际Twitter数据的动态评分。

