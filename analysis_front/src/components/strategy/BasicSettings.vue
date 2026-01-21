<template>
  <div class="config-grid">
    <!-- 基础条件 -->
    <div class="config-card">
      <h5 class="card-title">基础条件</h5>
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.spot_contract" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">交易对要求</span>
        </div>
        <div class="condition-description">
          必须同时有现货和合约交易对才能执行策略
        </div>
      </div>

      <!-- 交易类型选择 -->
      <div class="condition-card">
        <div class="condition-header">
          <span class="condition-title">交易类型</span>
        </div>
        <div class="condition-description">
          <div class="trading-type-selection">
            <label class="trading-type-option">
              <input
                type="radio"
                value="spot"
                v-model="conditions.trading_type"
              />
              <span class="radio-checkmark"></span>
              <div class="type-content">
                <div class="type-title">现货交易</div>
                <div class="type-description">仅使用现货市场进行交易</div>
              </div>
            </label>
            <label class="trading-type-option">
              <input
                type="radio"
                value="futures"
                v-model="conditions.trading_type"
              />
              <span class="radio-checkmark"></span>
              <div class="type-content">
                <div class="type-title">合约交易</div>
                <div class="type-description">仅使用期货合约进行交易</div>
              </div>
            </label>
            <label class="trading-type-option">
              <input
                type="radio"
                value="both"
                v-model="conditions.trading_type"
              />
              <span class="radio-checkmark"></span>
              <div class="type-content">
                <div class="type-title">两者皆可</div>
                <div class="type-description">根据市场条件选择现货或合约交易</div>
              </div>
            </label>
          </div>
          <div v-if="validationErrors.trading_type" class="field-error">{{ validationErrors.trading_type }}</div>
        </div>
      </div>
    </div>

    <!-- 交易配置 -->
    <div class="config-card">
      <h5 class="card-title">交易配置</h5>

      <!-- 交易方向选择 -->
      <div class="condition-card">
        <div class="condition-header">
          <span class="condition-title">允许交易方向</span>
        </div>
        <div class="condition-description">
          <div class="direction-selection">
            <label class="direction-option">
              <input
                type="checkbox"
                value="LONG"
                v-model="directionsArray"
              />
              <span class="checkmark-small"></span>
              <span>做多 (LONG)</span>
            </label>
            <label class="direction-option">
              <input
                type="checkbox"
                value="SHORT"
                v-model="directionsArray"
              />
              <span class="checkmark-small"></span>
              <span>做空 (SHORT)</span>
            </label>
          </div>
          <div v-if="validationErrors.directions" class="field-error">{{ validationErrors.directions }}</div>
        </div>
      </div>

      <!-- 杠杆配置 -->
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.enable_leverage" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">
            杠杆配置
            <span class="help-tooltip" data-tooltip="杠杆可以放大收益同时放大风险，请谨慎使用">?</span>
          </span>
        </div>
        <div class="condition-description">
          <div v-if="conditions.enable_leverage" class="leverage-config">
            <div class="config-item">
              <label>杠杆倍数：</label>
              <input
                v-model.number="conditions.default_leverage"
                class="inline-input small"
                type="number"
                min="1"
                max="100"
                placeholder="1"
              /> 倍
            </div>
          </div>
          <div class="config-note">
            💡 杠杆倍数会放大收益同时放大风险，请谨慎设置
          </div>
        </div>
      </div>

      <!-- 保证金模式选择 -->
      <div class="condition-card">
        <div class="condition-header">
          <span class="condition-title">保证金模式</span>
        </div>
        <div class="condition-description">
          <div class="margin-mode-selection">
            <label class="margin-mode-option">
              <input
                type="radio"
                value="ISOLATED"
                v-model="conditions.margin_mode"
              />
              <span class="radio-checkmark"></span>
              <div class="mode-content">
                <div class="mode-title">逐仓 (ISOLATED)</div>
                <div class="mode-description">每个交易对独立保证金，风险可控，推荐新手使用</div>
              </div>
            </label>
            <label class="margin-mode-option">
              <input
                type="radio"
                value="CROSS"
                v-model="conditions.margin_mode"
              />
              <span class="radio-checkmark"></span>
              <div class="mode-content">
                <div class="mode-title">全仓 (CROSS)</div>
                <div class="mode-description">共享账户保证金，资金利用率高，风险较高</div>
              </div>
            </label>
          </div>
          <div class="config-note">
            💡 逐仓模式更安全，全仓模式资金效率更高
          </div>
        </div>
      </div>

      <!-- 持仓过滤 -->
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.skip_held_positions" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">跳过已在持仓的币种</span>
        </div>
        <div class="condition-description">
          如果某个币种已经有未平仓的持仓，则跳过该币种的交易，避免重复买入
          <div class="config-note">
            💡 建议启用，避免过度集中和重复交易
          </div>
        </div>
      </div>

      <!-- 平仓过滤 -->
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="skipCloseOrdersEnabled" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">跳过指定时间内有平仓记录的币种</span>
        </div>
        <div class="condition-description">
          如果某个币种在过去指定时间内有平仓订单记录，则跳过该币种的交易，避免频繁操作
          <div v-if="skipCloseOrdersEnabled" class="time-config">
            <div class="config-item">
              <label>跳过时间：</label>
              <input
                v-model.number="conditions.skip_close_orders_hours"
                class="inline-input small"
                type="number"
                min="0"
                max="720"
                step="1"
                placeholder="24"
              /> 小时
            </div>
          </div>
          <div class="config-note">
            💡 适合保守策略，避免对同一币种进行过于频繁的交易。设置为0表示不跳过。
          </div>
        </div>
      </div>

      <!-- 币种黑名单配置 -->
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.use_symbol_blacklist" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">启用币种黑名单</span>
        </div>
        <div class="condition-description">
          禁止交易指定的币种，即使它们满足其他所有条件也不会被选中
          <div v-if="conditions.use_symbol_blacklist" class="symbol-config">
            <div class="config-item">
              <label>黑名单币种：</label>
              <textarea
                v-model="blacklistText"
                class="symbol-textarea"
                placeholder="输入币种符号，每行一个，例如：&#10;BTCUSDT&#10;ETHUSDT&#10;BNBUSDT"
                rows="4"
              ></textarea>
            </div>
          </div>
          <div class="config-note">
            💡 支持USDT和BUSD交易对。黑名单中的币种将被完全排除在交易选择之外。
          </div>
        </div>
      </div>

      <!-- 盈利加仓策略 -->
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.profit_scaling_enabled" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">盈利加仓策略</span>
        </div>
        <div class="condition-description">
          当持仓盈利达到指定百分比时，自动加仓指定金额
          <div v-if="conditions.profit_scaling_enabled" class="scaling-config">
            <div class="config-row">
              <label>触发加仓的盈利百分比：</label>
              <input
                v-model.number="conditions.profit_scaling_percent"
                class="inline-input"
                type="number"
                min="0.1"
                max="100"
                step="0.1"
                placeholder="5.0"
              /> %
            </div>
            <div class="config-row">
              <label>加仓金额：</label>
              <input
                v-model.number="conditions.profit_scaling_amount"
                class="inline-input"
                type="number"
                min="1"
                step="1"
                placeholder="100"
              /> USDT
            </div>
            <div class="config-row">
              <label>最大加仓次数：</label>
              <input
                v-model.number="conditions.profit_scaling_max_count"
                class="inline-input"
                type="number"
                min="1"
                max="10"
                step="1"
                placeholder="3"
              /> 次
            </div>
          </div>
          <div class="config-note">
            💡 在趋势向好时自动增加仓位，提高盈利潜力
          </div>
        </div>
      </div>

      <!-- 整体仓位止盈止损 -->
      <div class="condition-card">
        <div class="condition-header">
          <label class="condition-checkbox">
            <input type="checkbox" v-model="conditions.overall_stop_loss_enabled" />
            <span class="checkmark"></span>
          </label>
          <span class="condition-title">整体仓位止盈止损</span>
        </div>
        <div class="condition-description">
          当整体仓位达到指定盈亏百分比时，自动全部平仓。可选择只设置止损、只设置止盈，或两者都设置
          <div v-if="conditions.overall_stop_loss_enabled" class="scaling-config">
            <div class="config-row">
              <label>整体止盈：</label>
              <input
                v-model.number="conditions.overall_take_profit_percent"
                class="inline-input"
                type="number"
                min="0"
                max="500"
                step="1"
                placeholder="50（留空表示不设置）"
              /> %
            </div>
            <div class="config-row">
              <label>整体止损：</label>
              <input
                v-model.number="conditions.overall_stop_loss_percent"
                class="inline-input"
                type="number"
                min="0"
                max="100"
                step="1"
                placeholder="20（留空表示不设置）"
              /> %
            </div>
          </div>
          <div class="config-note">
            💡 保护整体仓位的安全，避免过度亏损或错过最佳盈利机会
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'

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
const emit = defineEmits(['update:conditions', 'update:directions'])

// 交易方向数组（用于多选框）
const directionsArray = ref(['LONG'])

// 监听交易方向变化
watch(directionsArray, (newValue) => {
  emit('update:directions', newValue)
}, { immediate: true })

// 监听父组件传入的方向数据变化
watch(() => props.conditions.allowed_directions, (newValue) => {
  if (newValue) {
    directionsArray.value = newValue.split(',').filter(d => d)
  }
}, { immediate: true })

// 计算属性：平仓过滤是否启用
const skipCloseOrdersEnabled = computed({
  get: () => props.conditions.skip_close_orders_hours > 0,
  set: (value) => {
    if (value) {
      // 启用时，如果当前值为0则设置为默认24小时
      if (props.conditions.skip_close_orders_hours === 0) {
        props.conditions.skip_close_orders_hours = 24
      }
    } else {
      // 禁用时设置为0
      props.conditions.skip_close_orders_hours = 0
    }
    emit('update:conditions', props.conditions)
  }
})

// 黑名单文本的双向绑定
const blacklistText = computed({
  get: () => {
    if (props.conditions.symbol_blacklist && Array.isArray(props.conditions.symbol_blacklist)) {
      return props.conditions.symbol_blacklist.join('\n')
    }
    return ''
  },
  set: (value) => {
    const symbols = value.split('\n')
      .map(s => s.trim())
      .filter(s => s.length > 0)
    props.conditions.symbol_blacklist = symbols
    emit('update:conditions', props.conditions)
  }
})

// 监听条件变化
watch(() => props.conditions, (newConditions) => {
  // 设置交易类型的默认值
  if (!newConditions.trading_type) {
    newConditions.trading_type = 'both'
  }
  // 初始化平仓过滤小时数
  if (newConditions.skip_close_orders_hours === undefined) {
    // 如果有旧的24小时设置，则迁移到新字段
    if (newConditions.skip_close_orders_within_24_hours) {
      newConditions.skip_close_orders_hours = 24
    } else {
      newConditions.skip_close_orders_hours = 0
    }
  }
  // 初始化黑名单
  if (!newConditions.symbol_blacklist) {
    newConditions.symbol_blacklist = []
  }
  emit('update:conditions', newConditions)
}, { deep: true, immediate: true })

// 监听小时数变化，确保数据一致性
watch(() => props.conditions.skip_close_orders_hours, (newHours) => {
  // 同步更新旧字段（向后兼容）
  props.conditions.skip_close_orders_within_24_hours = newHours > 0
}, { immediate: true })
</script>

<style scoped>
/* 基础设置组件的样式 */
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

.direction-selection {
  display: flex;
  gap: 16px;
}

.direction-option {
  display: flex;
  align-items: center;
  cursor: pointer;
  user-select: none;
}

.direction-option input {
  display: none;
}

.checkmark-small {
  width: 16px;
  height: 16px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  margin-right: 8px;
  position: relative;
  transition: all 0.2s;
}

.direction-option input:checked + .checkmark-small {
  background: #10b981;
  border-color: #10b981;
}

.direction-option input:checked + .checkmark-small::after {
  content: '✓';
  position: absolute;
  top: -2px;
  left: 1px;
  color: white;
  font-size: 12px;
  font-weight: bold;
}

.leverage-config {
  margin-top: 12px;
  padding: 12px;
  background: #f3f4f6;
  border-radius: 6px;
}

.scaling-config {
  margin-top: 12px;
  padding: 12px;
  background: #f3f4f6;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.config-row label {
  color: #6b7280;
  white-space: nowrap;
  min-width: 120px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 8px;
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

.config-note {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
  font-style: italic;
  display: flex;
  align-items: flex-start;
  gap: 4px;
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

.field-error {
  color: #f44336;
  font-size: 12px;
  margin-top: 4px;
}

/* 保证金模式选择样式 */
.margin-mode-selection {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.margin-mode-option {
  display: flex;
  align-items: flex-start;
  cursor: pointer;
  user-select: none;
  padding: 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  transition: all 0.2s;
  background: white;
}

.margin-mode-option:hover {
  border-color: #d1d5db;
  box-shadow: 0 2px 4px 0 rgba(0, 0, 0, 0.1);
}

.margin-mode-option input {
  display: none;
}

.margin-mode-option input:checked + .radio-checkmark {
  background: #3b82f6;
  border-color: #3b82f6;
}

.margin-mode-option input:checked + .radio-checkmark::after {
  opacity: 1;
  transform: scale(1);
}

.radio-checkmark {
  width: 20px;
  height: 20px;
  border: 2px solid #d1d5db;
  border-radius: 50%;
  margin-right: 12px;
  position: relative;
  flex-shrink: 0;
  margin-top: 2px;
  transition: all 0.2s;
}

.radio-checkmark::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 8px;
  height: 8px;
  background: white;
  border-radius: 50%;
  opacity: 0;
  transform: scale(0.5);
  transition: all 0.2s;
}

.mode-content {
  flex: 1;
}

.mode-title {
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 4px;
}

.mode-description {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.4;
}

@media (max-width: 768px) {
  .config-grid {
    grid-template-columns: 1fr;
  }

  .direction-selection {
    flex-direction: column;
    gap: 8px;
  }
}

/* 交易类型选择样式 */
.trading-type-selection {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.trading-type-option {
  display: flex;
  align-items: flex-start;
  cursor: pointer;
  user-select: none;
  padding: 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  transition: all 0.2s;
  background: white;
}

.trading-type-option:hover {
  border-color: #d1d5db;
  box-shadow: 0 2px 4px 0 rgba(0, 0, 0, 0.1);
}

.trading-type-option input {
  display: none;
}

.trading-type-option input:checked + .radio-checkmark {
  background: #3b82f6;
  border-color: #3b82f6;
}

.trading-type-option input:checked + .radio-checkmark::after {
  opacity: 1;
  transform: scale(1);
}

.type-content {
  flex: 1;
}

.type-title {
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 4px;
}

.type-description {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.4;
}

.symbol-config {
  margin-top: 12px;
}

.symbol-textarea {
  width: 100%;
  min-height: 80px;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.4;
  resize: vertical;
  transition: border-color 0.2s;
}

.symbol-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}
</style>