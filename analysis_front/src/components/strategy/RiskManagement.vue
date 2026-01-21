<template>
  <div class="config-grid">
    <!-- 止损止盈 -->
    <div class="config-card">
      <h5 class="card-title">止损止盈</h5>

      <div class="condition-card">
        <div class="condition-row">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.enable_stop_loss" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">止损</span>
          <input
            v-model.number="conditions.stop_loss_percent"
            class="inline-input small"
            type="number"
            min="0.1"
            max="50"
            step="0.1"
            placeholder="2.0"
          /> %
        </div>
        <div class="condition-row">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.enable_take_profit" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">止盈</span>
          <input
            v-model.number="conditions.take_profit_percent"
            class="inline-input small"
            type="number"
            min="0.1"
            max="100"
            step="0.1"
            placeholder="5.0"
          /> %
        </div>
        </div>

        <!-- 保证金损失止损 -->
        <div class="condition-card">
          <div class="condition-row">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.enable_margin_loss_stop_loss" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">保证金损失止损</span>
            <input
              v-model.number="conditions.margin_loss_stop_loss_percent"
              class="inline-input small"
              type="number"
              min="0.1"
              max="80"
              step="0.1"
              placeholder="30.0"
              @input="validateMarginLossStopLoss"
              @blur="validateMarginLossStopLoss"
            /> %
          </div>
          <div class="condition-description">
            💡 当持仓保证金亏损达到设定百分比时触发止损，更加精准的风险控制。建议设置5%以上以避免过度敏感。适用于合约交易。
          </div>
        </div>

        <!-- 保证金盈利止盈 -->
        <div class="condition-card">
          <div class="condition-row">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.enable_margin_profit_take_profit" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">保证金盈利止盈</span>
            <input
              v-model.number="conditions.margin_profit_take_profit_percent"
              class="inline-input small"
              type="number"
              min="0.1"
              max="500"
              step="0.1"
              placeholder="100.0"
              @input="validateMarginProfitTakeProfit"
              @blur="validateMarginProfitTakeProfit"
            /> %
          </div>
          <div class="condition-description">
            💡 当持仓保证金盈利达到设定百分比时触发止盈，锁定盈利并避免利润回吐。适用于合约交易。
          </div>
        </div>
      </div>

      <!-- 仓位管理 -->
    <div class="config-card">
      <h5 class="card-title">📊 仓位管理</h5>

      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.dynamic_positioning" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">
            动态仓位管理
            <span class="help-tooltip" data-tooltip="根据市场条件自动调整仓位大小，控制风险">?</span>
          </span>
        </div>
        <div class="condition-description">
          最大仓位：
          <input
            v-model.number="conditions.max_position_size"
            class="inline-input"
            type="number"
            min="1"
            max="100"
            step="1"
            placeholder="20"
          /> %，调整步长：
          <input
            v-model.number="conditions.position_size_step"
            class="inline-input"
            type="number"
            min="0.1"
            max="10"
            step="0.1"
            placeholder="1.0"
          /> %
        </div>
      </div>
    </div>

    <!-- 波动率过滤 -->
    <div class="config-card">
      <h5 class="card-title">波动率过滤</h5>

      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.volatility_filter_enabled" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">
            波动率过滤
            <span class="help-tooltip" data-tooltip="避免在高波动率市场中交易，降低风险">?</span>
          </span>
        </div>
        <div class="condition-description">
          波动率超过
          <input
            v-model.number="conditions.max_volatility"
            class="inline-input"
            type="number"
            min="1"
            max="200"
            step="1"
            placeholder="50"
          /> % 或周期超过
          <input
            v-model.number="conditions.volatility_period"
            class="inline-input"
            type="number"
            min="1"
            max="365"
            placeholder="30"
          /> 天时跳过交易
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { watch, nextTick } from 'vue'

// Props
const props = defineProps({
  conditions: {
    type: Object,
    required: true
  },
  validationErrors: {
    type: Object,
    default: () => ({})
  }
})

// Emits
const emit = defineEmits(['update:conditions'])

// 监听条件变化
watch(() => props.conditions, (newConditions) => {
  console.log('[RiskManagement] 条件更新:', newConditions)
  emit('update:conditions', newConditions)
}, { deep: true })

// 监听保证金损失止损的变化
watch(() => props.conditions.enable_margin_loss_stop_loss, (newValue) => {
  console.log('[RiskManagement] 保证金损失止损启用状态变化:', newValue)
  if (newValue) {
    // 当启用时，延迟聚焦到输入框
    nextTick(() => {
      const input = document.querySelector('input[placeholder="30.0"]')
      if (input) {
        input.focus()
        input.select()
      }
    })
  }
})

// 监听保证金盈利止盈的变化
watch(() => props.conditions.enable_margin_profit_take_profit, (newValue) => {
  console.log('[RiskManagement] 保证金盈利止盈启用状态变化:', newValue)
  if (newValue) {
    // 当启用时，延迟聚焦到输入框
    nextTick(() => {
      const input = document.querySelector('input[placeholder="100.0"]')
      if (input) {
        input.focus()
        input.select()
      }
    })
  }
})

watch(() => props.conditions.margin_loss_stop_loss_percent, (newValue) => {
  console.log('[RiskManagement] 保证金损失止损百分比变化:', newValue)
})

// 验证和修正保证金损失止损值
function validateMarginLossStopLoss(event) {
  const input = event.target
  let value = parseFloat(input.value)

  console.log('[RiskManagement] 验证保证金损失止损值:', value)

  // 自动修正无效值
  if (isNaN(value) || value <= 0) {
    value = 30.0 // 默认值
  } else if (value > 80) {
    value = 80 // 最大值
  }

  // 更新值
  if (value !== parseFloat(input.value)) {
    console.log('[RiskManagement] 修正值从', input.value, '到', value)
    input.value = value
    props.conditions.margin_loss_stop_loss_percent = value
  }
}

// 验证和修正保证金盈利止盈值
function validateMarginProfitTakeProfit(event) {
  const input = event.target
  let value = parseFloat(input.value)

  console.log('[RiskManagement] 验证保证金盈利止盈值:', value)

  // 自动修正无效值
  if (isNaN(value) || value <= 0) {
    value = 100.0 // 默认值
  } else if (value > 500) {
    value = 500 // 最大值
  }

  // 更新值
  if (value !== parseFloat(input.value)) {
    console.log('[RiskManagement] 修正值从', input.value, '到', value)
    input.value = value
    props.conditions.margin_profit_take_profit_percent = value
  }
}
</script>

<style scoped>
/* 风险控制组件的样式 */
.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
}

.config-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  transition: all 0.2s;
}

.config-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}

.condition-card {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 16px;
  transition: all 0.2s;
}

.condition-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
}

.condition-card:last-child {
  margin-bottom: 0;
}

.condition-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.condition-row:last-child {
  margin-bottom: 0;
}

.condition-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.condition-checkbox {
  display: flex;
  align-items: center;
  margin-right: 12px;
  cursor: pointer;
  user-select: none;
}

.condition-checkbox input {
  display: none;
}

.checkmark {
  width: 20px;
  height: 20px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  margin-right: 8px;
  position: relative;
  transition: all 0.2s;
}

.condition-checkbox input:checked + .checkmark {
  background: #3b82f6;
  border-color: #3b82f6;
}

.condition-checkbox input:checked + .checkmark::after {
  content: '✓';
  position: absolute;
  top: -2px;
  left: 2px;
  color: white;
  font-size: 14px;
  font-weight: bold;
}

.condition-title {
  font-weight: 500;
  color: #374151;
  flex: 1;
}

.condition-description {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.5;
}

.inline-input {
  width: 80px;
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  text-align: center;
}

.inline-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.inline-input.small {
  width: 60px;
}

.help-tooltip {
  position: relative;
  display: inline-block;
  margin-left: 6px;
  cursor: help;
  color: var(--text-secondary);
  font-size: 12px;
  vertical-align: middle;
}

.help-tooltip:hover::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: var(--text-primary);
  color: white;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  white-space: nowrap;
  z-index: 1000;
}

@media (max-width: 768px) {
  .config-grid {
    grid-template-columns: 1fr;
  }
}
</style>