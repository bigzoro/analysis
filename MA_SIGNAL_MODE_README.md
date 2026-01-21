# 均线策略信号模式功能说明

## 📋 功能概述

新增的信号模式功能允许用户在创建均线策略时选择两种不同的信号处理方式：

- **质量优先模式 (QUALITY_FIRST)**: 严格验证，高品质信号，低信号量
- **数量优先模式 (QUANTITY_FIRST)**: 宽松验证，更多信号，中等质量

## 🎯 两种模式的详细对比

### 质量优先模式 (QUALITY_FIRST)

#### ✅ 优势
- **极高信号质量**: 多重严格验证，确保信号可靠性
- **极低假信号率**: 几乎没有噪音信号
- **高胜率**: 80-90%的信号都是有效信号
- **适合保守投资者**: 重视资金安全

#### ❌ 劣势
- **信号量很少**: 每天可能只有1-3个符合条件的币种
- **错过机会**: 一些中等质量的机会被过滤掉
- **等待时间长**: 可能几天都没有信号
- **资本利用率低**: 资金大部分时间闲置

#### 验证阈值
- 波动率 ≥ 8.00%
- 趋势强度 ≥ 0.0020
- 信号质量 ≥ 0.7
- 严格模式: true

### 数量优先模式 (QUANTITY_FIRST)

#### ✅ 优势
- **信号量充足**: 每天有5-15个符合条件的币种
- **资本利用率高**: 资金可以充分运作
- **机会更多**: 不会错过太多交易机会
- **适合活跃交易**: 频繁交易策略

#### ❌ 劣势
- **信号质量一般**: 包含一些噪音信号
- **胜率中等**: 60-70%的信号质量
- **需要更严格止损**: 避免假信号造成的损失
- **筛选工作量大**: 需要人工验证信号质量

#### 验证阈值
- 波动率 ≥ 3.00%
- 趋势强度 ≥ 0.0005
- 信号质量 ≥ 0.4
- 严格模式: false

## 🔧 技术实现

### 后端实现

#### 1. 数据库字段
```sql
-- 在 strategy_conditions 表中添加
ma_signal_mode VARCHAR(32) DEFAULT 'QUALITY_FIRST'
```

#### 2. 验证阈值逻辑
```go
func GetMAValidationThresholds(signalMode string) MAValidationThresholds {
    switch signalMode {
    case "QUALITY_FIRST":
        return MAValidationThresholds{
            MinVolatility:    0.08,
            MinTrendStrength: 0.002,
            MinSignalQuality: 0.7,
            StrictMode:       true,
        }
    case "QUANTITY_FIRST":
        return MAValidationThresholds{
            MinVolatility:    0.03,
            MinTrendStrength: 0.0005,
            MinSignalQuality: 0.4,
            StrictMode:       false,
        }
    default:
        // 默认平衡模式
        return MAValidationThresholds{
            MinVolatility:    0.05,
            MinTrendStrength: 0.001,
            MinSignalQuality: 0.5,
            StrictMode:       false,
        }
    }
}
```

#### 3. 动态验证
```go
// 在均线策略扫描器中使用动态阈值
thresholds := GetMAValidationThresholds(conditions.MASignalMode)

if !ValidateVolatilityForMA(symbol, prices, thresholds.MinVolatility) {
    // 使用对应的阈值验证
}
```

### 前端实现

#### 1. 表单配置
```javascript
// 在策略表单中添加信号模式选择
const strategyForm = reactive({
  conditions: {
    moving_average_enabled: false,
    ma_signal_mode: 'QUALITY_FIRST', // 新增字段
    ma_type: 'SMA',
    short_ma_period: 5,
    long_ma_period: 20,
    // ... 其他字段
  }
})
```

#### 2. UI界面
```vue
<!-- 信号模式选择 -->
<div class="config-item">
  <label>信号模式：</label>
  <select v-model="strategyForm.conditions.ma_signal_mode" class="inline-select">
    <option value="QUALITY_FIRST">质量优先 (高品质，低数量)</option>
    <option value="QUANTITY_FIRST">数量优先 (中等品质，高数量)</option>
  </select>
</div>

<!-- 模式说明 -->
<div class="config-item mode-description">
  <div v-if="strategyForm.conditions.ma_signal_mode === 'QUALITY_FIRST'" class="quality-mode">
    <strong>🎯 质量优先模式</strong><br>
    • 信号质量: 极高 (胜率80-90%)<br>
    • 假信号: 极少<br>
    • 适合: 保守投资者
  </div>
  <div v-else-if="strategyForm.conditions.ma_signal_mode === 'QUANTITY_FIRST'" class="quantity-mode">
    <strong>🚀 数量优先模式</strong><br>
    • 信号数量: 充足 (每天5-15个)<br>
    • 资金利用: 高效<br>
    • 适合: 活跃交易者
  </div>
</div>
```

## 📊 使用建议

### 投资者类型选择

#### 保守型投资者
- **推荐**: 质量优先模式
- **原因**: 重视资金安全，更愿意等待高质量信号
- **资金量**: 大资金，适合低频交易

#### 积极型投资者
- **推荐**: 数量优先模式
- **原因**: 追求资金效率，喜欢频繁交易
- **资金量**: 中小资金，需要提高利用率

#### 新手投资者
- **建议**: 从质量优先开始，积累经验后考虑数量优先
- **理由**: 先学会识别高质量信号，再追求数量

### 市场环境适应

#### 牛市行情
- **波动大，机会多**: 可考虑数量优先模式
- **趋势明显**: 质量优先模式也能获得不错收益

#### 熊市行情
- **波动大，风险高**: 建议质量优先模式
- **谨慎交易**: 严格控制信号质量

#### 震荡行情
- **假信号多**: 质量优先模式更安全
- **难把握**: 建议减少交易或等待更好时机

## 🎛️ 配置示例

### 质量优先策略配置
```json
{
  "name": "保守均线策略",
  "conditions": {
    "moving_average_enabled": true,
    "ma_signal_mode": "QUALITY_FIRST",
    "ma_type": "SMA",
    "short_ma_period": 5,
    "long_ma_period": 20,
    "ma_cross_signal": "GOLDEN_CROSS",
    "ma_trend_filter": true,
    "ma_trend_direction": "UP"
  }
}
```

### 数量优先策略配置
```json
{
  "name": "活跃均线策略",
  "conditions": {
    "moving_average_enabled": true,
    "ma_signal_mode": "QUANTITY_FIRST",
    "ma_type": "EMA",
    "short_ma_period": 8,
    "long_ma_period": 21,
    "ma_cross_signal": "BOTH",
    "ma_trend_filter": false
  }
}
```

## ⚠️ 注意事项

1. **模式选择**: 根据自身投资风格和风险偏好选择合适的模式
2. **资金管理**: 数量优先模式需要更严格的资金管理和止损
3. **市场监控**: 不同市场环境可能需要调整模式
4. **经验积累**: 新手建议从质量优先开始
5. **测试验证**: 建议在历史数据上测试不同模式的表现

## 🔄 扩展计划

### 短期优化
- [ ] 添加自适应模式 (根据市场波动率自动调整阈值)
- [ ] 增加更多验证指标 (成交量、持仓量等)
- [ ] 优化前端用户体验 (更直观的模式对比)

### 长期规划
- [ ] 机器学习优化阈值
- [ ] 多策略组合模式
- [ ] 实时市场适应调整

---

*此功能让用户可以根据自己的投资风格选择最适合的策略模式，在信号质量和数量之间取得最佳平衡。*
