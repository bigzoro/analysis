# 资金费率单位转换指南

## 📊 当前数据流分析

### 1. 前端输入
- 用户在输入框输入数值（如 -0.005）
- 界面显示为 "%" 单位
- 发送给后端的是原始数值

### 2. 后端保存
- 直接保存前端发送的数值
- 数据库存储原始值

### 3. 比较逻辑
```go
fundingRate := v.getCurrentFundingRate(symbol)  // API返回小数，如 0.0001
configValue := config.FuturesPriceShortMinFundingRate  // 前端输入值，如 -0.005

if fundingRate < configValue {
    // 0.0001 < -0.005 → false (不会过滤)
}
```

## 🎯 解决方案（不修改代码）

### 方案一：前端显示统一（推荐）

**修改 TradingStrategies.vue 显示逻辑：**

```vue
<!-- 当前显示 -->
<input v-model.number="conditions.futures_price_short_min_funding_rate"
       placeholder="-0.005" /> %，直接开空

<!-- 修改为 -->
<input v-model.number="conditions.futures_price_short_min_funding_rate"
       placeholder="-0.005" /> (输入小数，直接开空)
<span class="value-display">
  相当于 {{ (conditions.futures_price_short_min_funding_rate * 100).toFixed(2) }}%
</span>
```

### 方案二：数据迁移修正

**如果发现数据异常，运行迁移脚本：**

```sql
-- 检查当前数据
SELECT id, name, conditions->>'$.futures_price_short_min_funding_rate' as rate
FROM trading_strategies
WHERE conditions->>'$.futures_price_short_min_funding_rate' IS NOT NULL;

-- 如果数值过大（如 > 1 或 < -1），说明保存的是百分比，需要转换
UPDATE trading_strategies
SET conditions = JSON_SET(
    conditions,
    '$.futures_price_short_min_funding_rate',
    CAST(conditions->>'$.futures_price_short_min_funding_rate' AS DECIMAL(10,4)) / 100
)
WHERE CAST(conditions->>'$.futures_price_short_min_funding_rate' AS DECIMAL(10,4)) > 1
   OR CAST(conditions->>'$.futures_price_short_min_funding_rate' AS DECIMAL(10,4)) < -1;
```

### 方案三：用户输入规范（最简单）

**明确告知用户正确的输入格式：**

```
资金费率输入说明：
• 输入格式：小数形式（如 -0.005）
• 表示含义：-0.005 = -0.5%
• 取值范围：-1.0 到 1.0
• 负值表示允许负资金费率
```

## 🔍 验证方法

### 1. 测试比较逻辑
```javascript
// 前端输入：-0.005 (表示 -0.5%)
// API获取：0.0001 (表示 0.01%)
// 比较：0.0001 < -0.005 → false (正确，不会过滤)
```

### 2. 边界测试
- 输入 0.01（1%）：应该过滤掉费率 < 1%的合约
- 输入 -0.005（-0.5%）：应该允许费率 > -0.5%的合约

## ✅ 推荐执行步骤

1. **检查当前数据**：确认数据库中的数值是否正确
2. **统一前端显示**：添加小数到百分比的转换显示
3. **用户培训**：明确告知正确的输入格式
4. **定期检查**：监控策略执行是否符合预期

这样可以在不修改核心代码的情况下解决单位不一致的问题。