# 公告重要性评分功能实现说明

## 功能概述

已成功实现公告重要性评分功能，用于完善推荐系统中的事件因子评分。该功能能够：

1. **多维度评分**：根据类型、交易所、热度、时间、验证状态综合评分
2. **智能匹配**：精确匹配币种符号，避免误匹配
3. **重要性分级**：自动判断公告重要性（高/中/低）
4. **集成到推荐系统**：将重要性得分应用到推荐评分中

## 实现文件

### 1. `analysis_backend/internal/server/announcement_scoring.go`（新建）

核心公告评分模块，包含以下功能：

#### 主要函数

- **`GetAnnouncementScoreForSymbol`**: 获取单个币种的公告重要性评分
  - 查询最近N天的公告
  - 计算每个公告的得分
  - 返回最高分的公告

- **`GetAnnouncementScoresForSymbols`**: 批量获取多个币种的公告评分（性能优化）

- **`calculateAnnouncementScore`**: 计算单个公告的重要性得分
  - 类型得分（0-10分）
  - 交易所得分（0-8分）
  - 热度得分（0-6分）
  - 时间衰减得分（0-4分）
  - 验证加分（0-2分）

#### 数据结构

```go
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
```

### 2. `analysis_backend/internal/server/recommendation.go`（修改）

#### 修改内容

1. **在`generateRecommendations`中集成公告评分**：
   ```go
   // 4.5. 获取公告重要性得分
   announcementScores, err := s.GetAnnouncementScoresForSymbols(ctx, baseSymbolsForAnnouncement, 7)
   ```

2. **更新`calculateScore`函数**：
   - 添加`announcementScore *AnnouncementScore`参数
   - 使用重要性得分计算事件得分（30分制转换为15分制）

3. **更新`RecommendationScore`结构**：
   - 在`Data`字段中添加`AnnouncementScore *AnnouncementScore`

4. **完善`generateReasons`函数**：
   - 添加公告重要性相关的推荐理由
   - 根据重要性等级、验证状态、热度等生成不同理由

## 评分算法

### 1. 类型得分（0-10分）

| 类型 | 得分 | 说明 |
|------|------|------|
| newcoin | 10.0 | 新币上线最重要 |
| listing | 9.0 | 上线公告 |
| launch | 8.5 | 发布 |
| event | 8.0 | 事件 |
| partnership | 7.5 | 合作 |
| finance | 7.0 | 金融相关 |
| update | 6.0 | 更新/升级 |
| other | 5.0 | 其他类型 |
| IsEvent=true | 10.0 | 重要事件直接满分 |

### 2. 交易所得分（0-8分）

| 交易所 | 得分 | 说明 |
|--------|------|------|
| Binance | 8.0 | 最重要 |
| Coinbase | 7.5 | 重要 |
| OKX/OKEX | 7.0 | 重要 |
| Bybit | 6.5 | 较重要 |
| Kraken | 6.0 | 较重要 |
| Upbit | 6.0 | 较重要 |
| Huobi/HTX | 5.5 | 一般 |
| Bitget | 5.5 | 一般 |
| Gate.io | 5.0 | 一般 |
| Kucoin | 5.0 | 一般 |
| 其他 | 4.0 | 基础分 |
| 无交易所 | 0.0 | 无信息 |

### 3. 热度得分（0-6分）

基于`HeatScore`字段（0-100）：
- HeatScore > 80：4分（极高热度）
- HeatScore > 60：3分（高热度）
- HeatScore > 40：2分（中等热度）
- HeatScore > 20：1分（低热度）
- HeatScore > 0：0.5分（极低热度）
- HeatScore = 0：0分

如果`IsEvent=true`，额外加2分基础热度。

### 4. 时间衰减得分（0-4分）

| 发布时间 | 得分 | 说明 |
|----------|------|------|
| 24小时内 | 4.0 | 最新，满分 |
| 3天内 | 3.0 | 较新 |
| 7天内 | 2.0 | 较新 |
| 14天内 | 1.0 | 一般 |
| 超过14天 | 0.5 | 较旧 |

### 5. 验证加分（0-2分）

- `Verified=true`：+2分（官方验证，可信度高）
- `Verified=false`：0分

### 6. 总分计算

```
总分 = 类型得分 + 交易所得分 + 热度得分 + 时间得分 + 验证加分
最高30分
```

### 7. 重要性分级

- **high**：总分 ≥ 20分
- **medium**：总分 ≥ 10分
- **low**：总分 < 10分

### 8. 事件得分转换

事件因子最高15分，需要将30分制的公告得分转换为15分制：

```
事件得分 = (公告总分 / 30.0) * 15.0
最高15分
```

## 使用示例

### 在推荐系统中使用

```go
// 获取币种的公告评分
score, err := s.GetAnnouncementScoreForSymbol(ctx, "BTC", 7)
if err != nil {
    // 使用默认值
    score = nil
}

// 使用评分计算事件得分
var eventScore float64
if score != nil {
    eventScore = (score.TotalScore / 30.0) * 15.0
    eventScore = math.Min(15, eventScore)
} else if hasAnnouncement {
    eventScore = 10 // 默认值
}
```

### 批量查询

```go
symbols := []string{"BTC", "ETH", "SOL"}
scores, err := s.GetAnnouncementScoresForSymbols(ctx, symbols, 7)
// scores["BTC"] 包含BTC的公告评分结果
```

## 推荐理由示例

根据公告评分结果，系统会自动生成推荐理由：

- **高重要性**："重要公告：Binance宣布上线XXX（得分25.5，Binance）"
- **中重要性**："公告：OKX宣布XXX合作（得分15.2）"
- **低重要性**："有相关公告（得分8.5）"
- **验证公告**："官方验证公告，可信度高"
- **高热度**："公告热度较高，市场关注度提升"
- **新币上线**："新币上线或重大事件，市场影响较大"

## 数据流程

```
1. 推荐系统生成候选币种列表
   ↓
2. 批量查询所有候选币种的公告评分
   ↓
3. 从数据库查询最近7天的相关公告
   ↓
4. 精确匹配币种符号（避免误匹配）
   ↓
5. 计算每个公告的多维度得分
   ↓
6. 选择最高分的公告
   ↓
7. 转换为事件得分（30分制→15分制）
   ↓
8. 应用到推荐评分中（事件因子15%权重）
   ↓
9. 生成包含公告信息的推荐理由
```

## 注意事项

1. **数据依赖**：需要定期运行公告扫描器收集公告数据
2. **符号匹配**：使用精确匹配避免误匹配（如BTC匹配到BTCE）
3. **时间范围**：默认查询最近7天，可根据需要调整
4. **性能考虑**：批量查询时需要注意数据库查询性能

## 后续优化建议

1. **缓存优化**：缓存公告评分结果，减少重复计算
2. **实时更新**：支持实时更新公告数据
3. **更多维度**：添加公告阅读量、转发量等指标
4. **情感分析**：分析公告的情感倾向（正面/负面）
5. **关联分析**：分析公告与价格走势的关联性

## 总结

公告重要性评分功能已成功集成到推荐系统中，能够：

- ✅ 多维度评分（类型、交易所、热度、时间、验证）
- ✅ 精确匹配币种符号
- ✅ 自动判断重要性等级（高/中/低）
- ✅ 使用重要性得分智能计算事件得分
- ✅ 应用到推荐评分中（15%权重）
- ✅ 生成包含公告信息的推荐理由

该功能显著提升了推荐系统的事件因子评分质量，从原来的简单有/无判断变为基于多维度重要性评分的智能评分。

