// 前端显示统一修复方案
// 在 TradingStrategies.vue 中统一显示逻辑

// 当前的问题代码片段：
/*
<input v-model.number="conditions.futures_price_short_min_funding_rate"
       class="inline-input"
       type="number"
       min="-1"
       max="1"
       step="0.001"
       placeholder="-0.005"
/> %，直接开空
*/

// 修复后的显示逻辑：
/*
<input v-model.number="conditions.futures_price_short_min_funding_rate"
       class="inline-input"
       type="number"
       min="-1"
       max="1"
       step="0.001"
       placeholder="-0.005"
/> % (输入如 -0.005 表示 -0.5%)，直接开空
<span class="help-text">
  当前值相当于 {{ (conditions.futures_price_short_min_funding_rate * 100).toFixed(2) }}%
</span>
*/