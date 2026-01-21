<template>
  <div class="config-grid">
    <!-- 时间过滤 -->
    <div class="config-card">
      <h5 class="card-title">时间过滤</h5>

      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.time_filter_enabled" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">时间过滤</span>
        </div>
        <div class="condition-description">
          只在 UTC
          <input
            v-model.number="conditions.start_hour"
            class="inline-input small"
            type="number"
            min="0"
            max="23"
            placeholder="9"
          />:00 -
          <input
            v-model.number="conditions.end_hour"
            class="inline-input small"
            type="number"
            min="0"
            max="23"
            placeholder="17"
          />:00 之间交易
          <label class="checkbox-inline">
            <input type="checkbox" v-model="conditions.weekend_trading" />
            包含周末
          </label>
        </div>
      </div>
    </div>

    <!-- 市场状态过滤 -->
    <div class="config-card">
      <h5 class="card-title">📊 市场状态过滤</h5>

      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.market_regime_filter_enabled" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">市场状态过滤</span>
        </div>
        <div class="condition-description">
          阈值：
          <input
            v-model.number="conditions.market_regime_threshold"
            class="inline-input"
            type="number"
            min="0.01"
            max="1"
            step="0.01"
            placeholder="0.1"
          />，偏好状态：
          <select v-model="conditions.preferred_regime" class="inline-select">
            <option value="bull">牛市</option>
            <option value="bear">熊市</option>
            <option value="sideways">横盘</option>
            <option value="">不限制</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { watch } from 'vue'

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
  emit('update:conditions', newConditions)
}, { deep: true })
</script>

<style scoped>
/* 市场时机组件的样式 */
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
}

.condition-description {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.5;
}

.checkbox-inline {
  display: inline-flex;
  align-items: center;
  margin-left: 12px;
  cursor: pointer;
  user-select: none;
  font-size: 14px;
  color: #6b7280;
}

.checkbox-inline input {
  margin-right: 6px;
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

.inline-select {
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  background: white;
}

.inline-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

@media (max-width: 768px) {
  .config-grid {
    grid-template-columns: 1fr;
  }
}
</style>