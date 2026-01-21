<template>
  <div class="single-column">
    <!-- 左：创建计划单 -->
    <div class="box">
      <div class="box-header">
        <h3>新建定时订单</h3>
        <div class="box-description">创建定时执行的交易订单</div>
      </div>
      <div class="order-form">
        <!-- 策略设置 -->
        <div class="form-section">
          <h4 class="section-title">交易策略</h4>
          <div class="form-group">
            <label class="form-label">使用策略</label>
            <select v-model="form.strategy_id" class="form-select">
              <option value="">不使用策略</option>
              <option v-for="strategy in availableStrategies" :key="strategy.id" :value="strategy.id">
                {{ strategy.name }}
              </option>
            </select>
          </div>

          <!-- 策略详情 -->
          <div v-if="selectedStrategy" class="strategy-info-card">
            <div class="strategy-header">
              <h5>策略详情：{{ selectedStrategy.name }}</h5>
            </div>
            <div class="strategy-conditions">
              <!-- 基础信息显示 -->
              <div v-if="selectedStrategy.conditions.trading_type && selectedStrategy.conditions.trading_type !== ''" class="condition-item basic-info">
                📊 交易类型：{{ getTradingTypeText(selectedStrategy.conditions.trading_type) }}
              </div>
              <div v-if="selectedStrategy.conditions.margin_mode" class="condition-item basic-info">
                💰 保证金模式：{{ getMarginModeText(selectedStrategy.conditions.margin_mode) }}
              </div>

              <div v-if="selectedStrategy.conditions.spot_contract" class="condition-item">
                需要现货+合约
              </div>
              <div v-if="selectedStrategy.conditions.no_short_below_market_cap" class="condition-item">
                市值&lt;{{ selectedStrategy.conditions.market_cap_limit_short }}万不开空
              </div>
              <div v-if="selectedStrategy.conditions.short_on_gainers" class="condition-item">
                涨幅前{{ selectedStrategy.conditions.gainers_rank_limit }} &amp; 市值&gt;{{ selectedStrategy.conditions.market_cap_limit_short }}万 → 开空{{ selectedStrategy.conditions.short_multiplier }}倍
              </div>
              <div v-if="selectedStrategy.conditions.long_on_small_gainers" class="condition-item">
                市值&lt;{{ selectedStrategy.conditions.market_cap_limit_long }}万 &amp; 涨幅前{{ selectedStrategy.conditions.gainers_rank_limit_long }} → 开多{{ selectedStrategy.conditions.long_multiplier }}倍
              </div>

              <!-- 合约涨幅开空策略 -->
              <div v-if="selectedStrategy.conditions.futures_price_short_strategy_enabled" class="condition-item futures-short">
                📈 合约涨幅开空策略：市值高于 {{ selectedStrategy.conditions.futures_price_short_min_market_cap }}万，涨幅排名前 {{ selectedStrategy.conditions.futures_price_short_max_rank }} 名以内，资金费率高于 {{ selectedStrategy.conditions.futures_price_short_min_funding_rate }}% 时，直接开空 {{ selectedStrategy.conditions.futures_price_short_leverage }} 倍杠杆
              </div>

              <!-- 技术指标策略条件 -->
              <div v-if="selectedStrategy.conditions.moving_average_enabled" class="condition-item tech-indicator">
                📈 均线策略：[{{ getMASignalModeText(selectedStrategy.conditions.ma_signal_mode) }}] {{ selectedStrategy.conditions.ma_type }}({{ selectedStrategy.conditions.short_ma_period }},{{ selectedStrategy.conditions.long_ma_period }}) -
                {{ getMACrossSignalText(selectedStrategy.conditions.ma_cross_signal) }}
                {{ selectedStrategy.conditions.ma_trend_filter ? '(' + getMATrendDirectionText(selectedStrategy.conditions.ma_trend_direction) + ')' : '' }}
              </div>

              <!-- 均值回归策略条件 -->
              <div v-if="selectedStrategy.conditions.mean_reversion_enabled" class="condition-item mean-reversion">
                🔄 均值回归策略
                <span v-if="selectedStrategy.conditions.mean_reversion_mode === 'enhanced'">
                  [{{ getMeanReversionSubModeText(selectedStrategy.conditions.mean_reversion_sub_mode) }}]
                </span>
                <span v-else>
                  [{{ getMRSignalModeText(selectedStrategy.conditions.mr_signal_mode) }}]
                </span>
                ：<br>
                • 计算周期：{{ selectedStrategy.conditions.mr_period }} 天<br>
                <span v-if="selectedStrategy.conditions.mr_bollinger_bands_enabled">
                  • 布林带指标：{{ selectedStrategy.conditions.mr_bollinger_multiplier }} 倍标准差<br>
                </span>
                <span v-if="selectedStrategy.conditions.mr_rsi_enabled">
                  • RSI指标：超卖阈值 {{ selectedStrategy.conditions.mr_rsi_oversold }}，超买阈值 {{ selectedStrategy.conditions.mr_rsi_overbought }}<br>
                </span>
                <span v-if="selectedStrategy.conditions.mr_price_channel_enabled">
                  • 价格通道：周期 {{ selectedStrategy.conditions.mr_channel_period }} 天<br>
                </span>
                <span v-if="selectedStrategy.conditions.mr_min_reversion_strength">
                  • 最小回归强度：{{ selectedStrategy.conditions.mr_min_reversion_strength }}<br>
                </span>
              </div>

              <!-- 套利策略条件 -->
              <div v-if="selectedStrategy.conditions.cross_exchange_arb_enabled" class="condition-item arb-strategy">
                🔄 跨交易所套利：价差超过 {{ selectedStrategy.conditions.price_diff_threshold }}%，最小套利金额 {{ selectedStrategy.conditions.min_arb_amount }} USDT
              </div>
              <div v-if="selectedStrategy.conditions.spot_future_arb_enabled" class="condition-item arb-strategy">
                🔄 现货-合约套利：基差超过 {{ selectedStrategy.conditions.basis_threshold }}%，资金费率超过 {{ selectedStrategy.conditions.funding_rate_threshold }}%
              </div>
      <div v-if="selectedStrategy.conditions.triangle_arb_enabled" class="condition-item arb-strategy">
        🔄 三角套利：套利机会超过 {{ selectedStrategy.conditions.triangle_threshold }}%，系统自动选择合适的币种组合
      </div>
              <div v-if="selectedStrategy.conditions.stat_arb_enabled" class="condition-item arb-strategy">
                🔄 统计套利：Z分数超过 {{ selectedStrategy.conditions.zscore_threshold }}，协整周期 {{ selectedStrategy.conditions.cointegration_period }} 天，套利对：{{ selectedStrategy.conditions.stat_arb_pairs }}
              </div>
              <div v-if="selectedStrategy.conditions.futures_spot_arb_enabled" class="condition-item arb-strategy">
                🔄 期现套利：到期前 {{ selectedStrategy.conditions.expiry_threshold }} 天，价差超过 {{ selectedStrategy.conditions.spot_future_spread }}%
              </div>

              <!-- 风险控制条件 -->
              <div v-if="selectedStrategy.conditions.enable_stop_loss" class="condition-item risk-control">
                🛡️ 止损设置：{{ selectedStrategy.conditions.stop_loss_percent }}%
              </div>
              <div v-if="selectedStrategy.conditions.enable_take_profit" class="condition-item risk-control">
                🛡️ 止盈设置：{{ selectedStrategy.conditions.take_profit_percent }}%
              </div>
              <div v-if="selectedStrategy.conditions.enable_margin_loss_stop_loss" class="condition-item risk-control">
                💰 保证金损失止损：{{ selectedStrategy.conditions.margin_loss_stop_loss_percent }}%
              </div>
              <div v-if="selectedStrategy.conditions.enable_margin_profit_take_profit" class="condition-item risk-control">
                💰 保证金盈利止盈：{{ selectedStrategy.conditions.margin_profit_take_profit_percent }}%
              </div>
              <div v-if="selectedStrategy.conditions.enable_leverage" class="condition-item risk-control">
                ⚡ 杠杆倍数：{{ selectedStrategy.conditions.default_leverage }} 倍
              </div>
              <div v-if="selectedStrategy.conditions.dynamic_positioning" class="condition-item risk-control">
                📊 动态仓位管理：最大仓位 {{ selectedStrategy.conditions.max_position_size }}%，调整步长 {{ selectedStrategy.conditions.position_size_step }}%
              </div>
              <div v-if="selectedStrategy.conditions.volatility_filter_enabled" class="condition-item risk-control">
                📈 波动率过滤：波动率超过 {{ selectedStrategy.conditions.max_volatility }}% 或周期超过 {{ selectedStrategy.conditions.volatility_period }} 天时跳过交易
              </div>

              <!-- 交易配置条件 -->
              <div v-if="selectedStrategy.conditions.skip_held_positions" class="condition-item trading-config">
                🚫 跳过已有持仓：如果某个币种已经有未平仓的持仓，则跳过该币种的交易
              </div>
              <div v-if="selectedStrategy.conditions.skip_close_orders_hours > 0" class="condition-item trading-config">
                🕐 跳过{{ selectedStrategy.conditions.skip_close_orders_hours }}h内平仓币种：如果某个币种在过去{{ selectedStrategy.conditions.skip_close_orders_hours }}小时内有平仓订单记录，则跳过该币种的交易
              </div>
              <div v-if="selectedStrategy.conditions.use_symbol_whitelist && selectedStrategy.conditions.symbol_whitelist && selectedStrategy.conditions.symbol_whitelist.length > 0" class="condition-item symbol-filter">
                📋 币种白名单：{{ selectedStrategy.conditions.symbol_whitelist.join(', ') }}
              </div>
              <div v-if="selectedStrategy.conditions.use_symbol_blacklist && selectedStrategy.conditions.symbol_blacklist && selectedStrategy.conditions.symbol_blacklist.length > 0" class="condition-item symbol-filter">
                🚫 币种黑名单：{{ selectedStrategy.conditions.symbol_blacklist.join(', ') }}
              </div>
              <div v-if="selectedStrategy.conditions.profit_scaling_enabled" class="condition-item trading-config">
                📈 盈利加仓：当持仓盈利达到 {{ selectedStrategy.conditions.profit_scaling_percent }}% 时，自动加仓 {{ selectedStrategy.conditions.profit_scaling_amount }} USDT（最多 {{ selectedStrategy.conditions.profit_scaling_max_count }} 次）
              </div>
              <div v-if="selectedStrategy.conditions.overall_stop_loss_enabled" class="condition-item risk-control">
                🛡️ 整体止盈止损：{{ getOverallStopLossText(selectedStrategy.conditions) }}
              </div>

              <!-- 时间和市场过滤条件 -->
              <div v-if="selectedStrategy.conditions.time_filter_enabled" class="condition-item timing-filter">
                🕐 时间过滤：只在 UTC {{ selectedStrategy.conditions.start_hour }}:00 - {{ selectedStrategy.conditions.end_hour }}:00 之间交易{{ selectedStrategy.conditions.weekend_trading ? '（包含周末）' : '（仅工作日）' }}
              </div>
              <div v-if="selectedStrategy.conditions.market_regime_filter_enabled" class="condition-item timing-filter">
                📊 市场状态过滤：阈值 {{ selectedStrategy.conditions.market_regime_threshold }}，偏好状态：{{ selectedStrategy.conditions.preferred_regime || '不限制' }}
              </div>

              <!-- 交易方向 -->
              <div v-if="selectedStrategy.conditions.allowed_directions && selectedStrategy.conditions.allowed_directions !== 'LONG'" class="condition-item trading-direction">
                📈 允许交易方向：{{ selectedStrategy.conditions.allowed_directions.replace(',', ', ') }}
              </div>
            </div>

            <!-- 策略执行预览 -->
            <div class="strategy-preview-section">
              <div class="preview-header">
                <span>🔍 策略预览</span>
                <button
                  class="btn btn-outline"
                  @click="previewStrategy"
                  :disabled="previewing"
                >
                  {{ previewing ? '分析中...' : '扫描符合币种' }}
                </button>
              </div>

              <!-- 符合条件的币种列表 -->
              <div v-if="eligibleSymbols.length > 0" class="eligible-symbols-section">
                <div class="symbols-header">
                  <span>符合策略的币种 ({{ eligibleSymbols.length }}个)</span>
                </div>
                <div class="symbols-list">
                  <div
                    v-for="symbol in eligibleSymbols"
                    :key="symbol.symbol"
                    class="symbol-item"
                    :class="{ selected: selectedSymbols.includes(symbol.symbol) }"
                  >
                    <div class="symbol-checkbox">
                      <input
                        type="checkbox"
                        :value="symbol.symbol"
                        v-model="selectedSymbols"
                        @change="onSymbolSelectionChange"
                      />
                    </div>
                    <div class="symbol-info" @click="toggleSymbolSelection(symbol)">
                      <div class="symbol-name">{{ symbol.symbol }}</div>
                      <div class="symbol-details">
                        <!-- 三角套利路径显示 -->
                        <div v-if="symbol.triangle_path" class="triangle-path">
                          <span class="path-label">套利路径:</span>
                          <span class="path-symbols">{{ symbol.triangle_path.join(' → ') }}</span>
                          <span class="price-diff" :class="{ positive: symbol.price_diff > 0, negative: symbol.price_diff < 0 }">
                            价差: {{ symbol.price_diff > 0 ? '+' : '' }}{{ symbol.price_diff.toFixed(3) }}%
                          </span>
                        </div>
                        <!-- 普通交易对显示 -->
                        <template v-else>
                          <span class="market-cap">市值: {{ fmtUSD(symbol.market_cap) }}</span>
                          <span class="rank">排名: #{{ symbol.gainers_rank }}</span>
                        </template>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 清除选择按钮 -->
                <div v-if="selectedSymbols.length > 0" class="batch-actions">
                  <button class="btn batch-clear-btn" @click="clearSelection">
                    清除选择
                  </button>
                </div>

                <!-- 表单未完成提示 -->
                <div v-else-if="selectedSymbols.length > 0 && !isFormValid" class="form-incomplete-notice">
                  <div class="notice-icon">⚠️</div>
                  <div class="notice-text">
                    请先完整填写下面的订单参数，然后才能批量创建订单
                  </div>
                </div>
              </div>

            </div>
          </div>
        </div>

        <!-- 基础设置 -->
        <div class="form-section">
          <h4 class="section-title">交易基础信息</h4>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label">
                交易所
                <span class="required-mark">*</span>
              </label>
              <select v-model="form.exchange" class="form-select">
                <option value="binance_futures">Binance Futures</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">环境</label>
              <select v-model="form.testnet" class="form-select">
                <option :value="true">测试网</option>
                <option :value="false">正式网</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">
                交易对
                <span class="required-mark">*</span>
              </label>
              <input
                v-model="form.symbol"
                class="form-input"
                placeholder="例如：ETHUSDT"
              />
            </div>

            <div class="form-group">
              <label class="form-label">
                操作类型
                <span class="required-mark">*</span>
              </label>
              <select v-model="form.side" class="form-select">
                <option value="BUY">{{ form.reduce_only ? '平空仓位' : '开多仓位' }}</option>
                <option value="SELL">{{ form.reduce_only ? '平多仓位' : '开空仓位' }}</option>
              </select>
              <div class="form-hint">
                当前操作: {{ currentOperationDescription }}
              </div>
              <div class="form-warning" v-if="operationRiskHint.includes('⚠️')">
                {{ operationRiskHint }}
              </div>
            </div>
          </div>
        </div>

        <!-- 订单参数 -->
        <div class="form-section">
          <h4 class="section-title">订单参数</h4>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label">
                订单类型
                <span class="required-mark">*</span>
              </label>
              <select v-model="form.order_type" class="form-select">
                <option value="MARKET">MARKET (市价)</option>
                <option value="LIMIT">LIMIT (限价)</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">
                数量（基础币）
                <span class="required-mark">*</span>
              </label>
              <input
                v-model="form.quantity"
                class="form-input"
                placeholder="例如：0.010"
              />
            </div>

            <div v-if="form.order_type==='LIMIT'" class="form-group">
              <label class="form-label">
                限价
                <span class="required-mark">*</span>
              </label>
              <input
                v-model="form.price"
                class="form-input"
                placeholder="仅限价单必填"
              />
            </div>

            <div class="form-group">
              <label class="form-label">杠杆倍数</label>
              <input
                v-model.number="form.leverage"
                class="form-input"
                type="number"
                min="0"
                placeholder="0 或 正整数"
              />
            </div>

            <div class="form-group">
              <label class="form-label">仓位操作</label>
              <select v-model="form.reduce_only" class="form-select">
                <option :value="false">开仓 (建立新仓位)</option>
                <option :value="true">平仓 (关闭现有仓位)</option>
              </select>
              <div class="form-hint">
                {{ form.reduce_only ? '平仓操作：关闭现有仓位' : '开仓操作：建立新的仓位' }}
              </div>
            </div>
          </div>
        </div>

        <!-- 一键三连设置和执行时间 -->
        <div class="form-section">
          <div class="two-column-layout">
            <!-- 左列：一键三连设置 -->
            <div class="form-column">
              <h4 class="section-title">一键三连设置</h4>
              <div class="form-group">
                <label class="form-label">启用一键三连</label>
                <select v-model="form.bracket_enabled" class="form-select">
                  <option :value="false">禁用</option>
                  <option :value="true">启用</option>
                </select>
              </div>

              <div v-if="form.bracket_enabled" class="bracket-settings">
                <div class="bracket-notice">
                  💡 一键三连将在主订单成交后自动设置止盈止损订单
                </div>

                <div class="form-grid">
                  <div class="form-group">
                    <label class="form-label">止盈(%)</label>
                    <input
                      v-model.number="form.tp_percent"
                      class="form-input"
                      type="number"
                      min="0"
                      step="0.01"
                      placeholder="例如 2 表示 +2%"
                    />
                  </div>

                  <div class="form-group">
                    <label class="form-label">止损(%)</label>
                    <input
                      v-model.number="form.sl_percent"
                      class="form-input"
                      type="number"
                      min="0"
                      step="0.01"
                      placeholder="例如 1 表示 -1%"
                    />
                  </div>
                </div>

                <div class="bracket-divider">
                  <span>或直接使用绝对价格（百分比优先）</span>
                </div>

                <div class="form-grid">
                  <div class="form-group">
                    <label class="form-label">止盈价</label>
                    <input
                      v-model="form.tp_price"
                      class="form-input"
                      placeholder="可选"
                    />
                  </div>

                  <div class="form-group">
                    <label class="form-label">止损价</label>
                    <input
                      v-model="form.sl_price"
                      class="form-input"
                      placeholder="可选"
                    />
                  </div>

                  <div class="form-group">
                    <label class="form-label">触发价格类型</label>
                    <select v-model="form.working_type" class="form-select">
                      <option value="MARK_PRICE">MARK_PRICE (默认)</option>
                      <option value="CONTRACT_PRICE">CONTRACT_PRICE</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>

            <!-- 右列：执行时间 -->
            <div class="form-column">
              <h4 class="section-title">执行时间</h4>
              <div class="form-group">
                <label class="form-label">
                  触发时间
                  <span class="required-mark">*</span>
                </label>
                <input
                  type="datetime-local"
                  v-model="triggerLocal"
                  class="form-input"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="form-actions">
          <button class="btn btn-primary btn-large" @click="create">
            创建定时订单
          </button>
          <!-- 批量创建订单按钮 -->
          <button
            v-if="canShowBatchCreateInForm"
            class="btn btn-batch-create btn-large"
            @click="createBatchOrders"
          >
            📝 批量创建订单 ({{ selectedSymbols.length }}个)
          </button>
        </div>

        <!-- 状态消息 -->
        <div v-if="err || ok" class="form-message" :class="{ error: err, success: ok }">
          {{ err || ok }}
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { reactive, ref, computed, watch } from 'vue'
import { api } from '../../api/api.js'
import { fmtUSD } from '../../utils/utils.js'

// Props定义
const props = defineProps({
  onOrderCreated: {
    type: Function,
    default: () => {}
  }
})

// Emits定义
const emit = defineEmits(['order-created'])

// 表单数据
const form = reactive({
  exchange: 'binance_futures',
  testnet: true,
  symbol: 'ETHUSDT',
  side: 'BUY',
  order_type: 'MARKET',
  quantity: '0.010',
  price: '',
  leverage: 0,
  reduce_only: false,
  strategy_id: '',

  // === Bracket ===
  bracket_enabled: false,
  tp_percent: 0,
  sl_percent: 0,
  tp_price: '',
  sl_price: '',
  working_type: 'MARK_PRICE',
})

// 触发时间：默认当前时间 + 1 分钟（使用本地时间）
function getLocalDateTimeString(offsetMs = 0) {
  const d = new Date(Date.now() + offsetMs)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 改进的时间转换函数，确保正确处理本地时间
function toRFC3339FromLocal(dtLocal) {
  if (!dtLocal) return ''
  // 如果输入已经是完整的datetime-local格式，直接解析
  const d = new Date(dtLocal)
  return d.toISOString()
}
const triggerLocal = ref(getLocalDateTimeString(60_000))

const err = ref('')
const ok = ref('')
const availableStrategies = ref([])
const previewing = ref(false)
const strategyPreview = ref(null)
const eligibleSymbols = ref([]) // 符合策略的币种列表
const selectedSymbols = ref([]) // 选中的币种列表

// 整体止盈止损显示文本
const getOverallStopLossText = (conditions) => {
  const stopLoss = conditions.overall_stop_loss_percent
  const takeProfit = conditions.overall_take_profit_percent

  if (stopLoss > 0 && takeProfit > 0) {
    return `止损 ${stopLoss}%，止盈 ${takeProfit}%`
  } else if (stopLoss > 0) {
    return `止损 ${stopLoss}%`
  } else if (takeProfit > 0) {
    return `止盈 ${takeProfit}%`
  } else {
    return '无具体阈值'
  }
}

// 计算属性：获取当前选中的策略
const selectedStrategy = computed(() => {
  if (!form.strategy_id) return null
  return availableStrategies.value.find(s => s.id == form.strategy_id) || null
})

// 监听策略选择变化，自动设置表单参数
watch(selectedStrategy, (newStrategy, oldStrategy) => {
  if (newStrategy && newStrategy !== oldStrategy) {
    applyStrategyDefaults(newStrategy)
  }
})

// 计算属性：当前操作的完整描述
const currentOperationDescription = computed(() => {
  const operation = getOperationDescription(form.side, form.reduce_only)
  const operationType = getOperationType(form.side, form.reduce_only)

  return `${operationType} (${operation})`
})

// 计算属性：操作风险提示
const operationRiskHint = computed(() => {
  if (form.reduce_only) {
    return '平仓操作：请确保有对应的持仓，否则订单会失败'
  } else {
    const operation = getOperationType(form.side, form.reduce_only)
    if (operation === '开空') {
      return '开空操作：需要足够的保证金，市场风险较高'
    } else {
      return '开多操作：相对较低的风险'
    }
  }
})

// 计算属性：验证表单是否完整填写
const isFormValid = computed(() => {
  // 基础必填字段
  if (!form.exchange || !form.side || !form.order_type || !form.quantity || !triggerLocal.value) {
    return false
  }

  // 如果是限价单，必须填写价格
  if (form.order_type === 'LIMIT' && !form.price) {
    return false
  }

  return true
})

// 计算属性：是否可以显示批量创建按钮
const canShowBatchCreate = computed(() => {
  return selectedSymbols.value.length > 0 && isFormValid.value
})

// 计算属性：是否在表单底部显示批量创建按钮
const canShowBatchCreateInForm = computed(() => {
  return selectedSymbols.value.length > 0 && isFormValid.value && eligibleSymbols.value.length > 0
})

// 根据策略自动设置表单默认值
function applyStrategyDefaults(strategy) {
  if (!strategy || !strategy.conditions) return

  const conditions = strategy.conditions

  // 根据策略条件设置操作方向
  if (conditions.futures_price_short_strategy_enabled) {
    // 合约涨幅开空策略，专门用于开空
    form.side = 'SELL'
    form.reduce_only = false
  } else if (conditions.short_on_gainers && !conditions.long_on_small_gainers) {
    // 只有开空条件，默认开空
    form.side = 'SELL'
    form.reduce_only = false
  } else if (conditions.long_on_small_gainers && !conditions.short_on_gainers) {
    // 只有开多条件，默认开多
    form.side = 'BUY'
    form.reduce_only = false
  } else if (conditions.short_on_gainers && conditions.long_on_small_gainers) {
    // 既有开空又有开多条件，默认开多（相对安全）
    form.side = 'BUY'
    form.reduce_only = false
  }

  // 根据策略倍数设置杠杆（取整）
  if (conditions.futures_price_short_strategy_enabled && form.side === 'SELL') {
    // 合约涨幅开空策略使用专门的杠杆倍数
    form.leverage = Math.floor(conditions.futures_price_short_leverage) || 1
  } else if (conditions.short_on_gainers && form.side === 'SELL') {
    form.leverage = Math.floor(conditions.short_multiplier) || 1
  } else if (conditions.long_on_small_gainers && form.side === 'BUY') {
    form.leverage = Math.floor(conditions.long_multiplier) || 1
  } else {
    form.leverage = 1 // 默认无杠杆
  }

  // 设置默认订单参数（如果为空）
  if (!form.order_type) {
    form.order_type = 'MARKET' // 默认市价单
  }

  if (!form.quantity || form.quantity === '0.010') {
    // 根据操作方向和策略类型设置不同的默认数量
    if (form.side === 'SELL') {
      if (conditions.futures_price_short_strategy_enabled) {
        form.quantity = '0.001' // 合约涨幅开空策略用更小的数量
      } else {
        form.quantity = '0.001' // 其他开空策略也用更小的数量
      }
    } else {
      form.quantity = '0.010' // 开多用默认数量
    }
  }

  // 设置一键三连（从策略条件中读取止盈止损设置）
  if (conditions.enable_take_profit || conditions.enable_stop_loss) {
    form.bracket_enabled = true
    // 从策略条件中读取止盈止损设置
    if (!form.tp_percent && conditions.enable_take_profit) {
      form.tp_percent = conditions.take_profit_percent
    }
    if (!form.sl_percent && conditions.enable_stop_loss) {
      form.sl_percent = conditions.stop_loss_percent
    }
  } else if (form.leverage > 1 && !form.bracket_enabled) {
    // 如果策略没有设置止盈止损但有杠杆，则使用默认值
    form.bracket_enabled = true
    if (!form.tp_percent) form.tp_percent = 20 // 20%止盈
    if (!form.sl_percent) form.sl_percent = 5  // 5%止损
  }

  console.log(`策略 "${strategy.name}" 已自动设置表单参数:`, {
    side: form.side,
    leverage: form.leverage,
    quantity: form.quantity,
    bracket_enabled: form.bracket_enabled
  })
}

// 根据side和reduce_only判断准确的操作类型
function getOperationType(side, reduceOnly) {
  if (reduceOnly) {
    // 平仓操作
    return side === 'BUY' ? '平空' : '平多'
  } else {
    // 开仓操作
    return side === 'BUY' ? '开多' : '开空'
  }
}

// 获取操作类型的详细说明
function getOperationDescription(side, reduceOnly) {
  if (reduceOnly) {
    // 平仓操作
    return side === 'BUY' ? '平空头仓位' : '平多头仓位'
  } else {
    // 开仓操作
    return side === 'BUY' ? '开多头仓位' : '开空头仓位'
  }
}

// 加载可用策略
async function loadStrategies() {
  try {
    const res = await api.listTradingStrategies()
    availableStrategies.value = res.data || []
  } catch (e) {
    console.error('加载策略失败:', e)
  }
}

// 预览策略执行结果 - 扫描所有符合条件的币种
async function previewStrategy() {
  if (!selectedStrategy.value) return

  previewing.value = true
  try {
    // 调用新的扫描API，获取所有符合策略的币种
    const result = await api.scanEligibleSymbols(selectedStrategy.value.id)

    eligibleSymbols.value = result.eligible_symbols || []

    // 如果有符合条件的币种，设置预览为提示用户选择
    if (eligibleSymbols.value.length > 0) {
      strategyPreview.value = {
        action: 'select',
        reason: `发现${eligibleSymbols.value.length}个符合条件的币种，请勾选要创建订单的币种`,
        multiplier: 1.0
      }
    } else {
      strategyPreview.value = {
        action: 'no_op',
        reason: '没有找到符合策略的币种',
        multiplier: 1.0
      }
    }
  } catch (e) {
    console.error('策略预览失败:', e)
    eligibleSymbols.value = []
    strategyPreview.value = {
      action: 'error',
      reason: '预览失败: ' + e.message,
      multiplier: 1.0
    }
  } finally {
    previewing.value = false
  }
}

// 切换单个币种的选择状态
function toggleSymbolSelection(symbol) {
  const index = selectedSymbols.value.indexOf(symbol.symbol)
  if (index > -1) {
    selectedSymbols.value.splice(index, 1)
  } else {
    selectedSymbols.value.push(symbol.symbol)
  }
  onSymbolSelectionChange()
}

// 币种选择变化处理
function onSymbolSelectionChange() {
  // 如果只选择了一个币种，自动填充到表单
  if (selectedSymbols.value.length === 1) {
    const selectedSymbol = eligibleSymbols.value.find(s => s.symbol === selectedSymbols.value[0])
    if (selectedSymbol) {
      form.symbol = selectedSymbol.symbol
    }
  }
}

// 清除所有选择
function clearSelection() {
  selectedSymbols.value = []
}

// 批量创建订单
async function createBatchOrders() {
  if (selectedSymbols.value.length === 0) {
    err.value = '请先选择要创建订单的币种'
    return
  }

  err.value = ''
  ok.value = ''

  // 前端表单验证
  if (!form.exchange) {
    err.value = '请选择交易所'
    return
  }
  if (!form.side) {
    err.value = '请选择操作类型'
    return
  }
  if (!form.order_type) {
    err.value = '请选择订单类型'
    return
  }
  if (!form.quantity) {
    err.value = '请输入下单数量'
    return
  }
  if (!triggerLocal.value) {
    err.value = '请选择触发时间'
    return
  }

  // 批量操作类型确认验证
  const operationType = getOperationType(form.side, form.reduce_only)
  const operationDesc = getOperationDescription(form.side, form.reduce_only)
  const confirmMessage = `确认批量创建 ${selectedSymbols.value.length} 个订单？\n\n操作类型: ${operationType}\n详细说明: ${operationDesc}\n每个订单数量: ${form.quantity}\n交易对: ${selectedSymbols.value.join(', ')}`

  if (!confirm(confirmMessage)) {
    return
  }

  try {
    // 构建批量订单数据
    const orders = []
    for (const symbolName of selectedSymbols.value) {
      const symbolData = eligibleSymbols.value.find(s => s.symbol === symbolName)
      if (!symbolData) continue

      orders.push({
        exchange: form.exchange,
        testnet: form.testnet,
        symbol: symbolName,
        side: form.side, // 使用用户选择的统一操作方向
        order_type: form.order_type,
        quantity: form.quantity,
        price: form.order_type === 'LIMIT' ? form.price : '',
        leverage: symbolData.multiplier > 1 ? Math.floor(symbolData.multiplier) : form.leverage,
        reduce_only: form.reduce_only,
        strategy_id: form.strategy_id || null,
        trigger_time: toRFC3339FromLocal(triggerLocal.value),

        // bracket
        bracket_enabled: form.bracket_enabled,
        tp_percent: form.tp_percent,
        sl_percent: form.sl_percent,
        tp_price: form.tp_price,
        sl_price: form.sl_price,
        working_type: form.working_type,
      })
    }

    // 使用批量API一次性创建所有订单
    const result = await api.createBatchScheduledOrders({ orders })

    // 处理结果
    const successCount = result.success_count || 0
    const failCount = result.fail_count || 0

    if (successCount > 0) {
      ok.value = `批量创建完成：成功${successCount}个${failCount > 0 ? `，失败${failCount}个` : ''}`

      // 清除选择
      clearSelection()
    } else {
      err.value = `批量创建失败：所有${orders.length}个订单创建失败`
    }

    // 记录详细结果
    console.log('批量创建订单结果:', result)

  } catch (e) {
    console.error('批量创建订单失败:', e)
    err.value = e?.message || '批量创建订单失败'
  }
}

async function create() {
  err.value = ''
  ok.value = ''

  // 前端表单验证
  if (!form.exchange) {
    err.value = '请选择交易所'
    return
  }
  if (!form.symbol) {
    err.value = '请输入交易对'
    return
  }
  if (!form.side) {
    err.value = '请选择操作类型'
    return
  }

  // 操作类型确认验证
  const operationType = getOperationType(form.side, form.reduce_only)
  const operationDesc = getOperationDescription(form.side, form.reduce_only)
  const confirmMessage = `确认创建订单？\n\n操作类型: ${operationType}\n详细说明: ${operationDesc}\n交易对: ${form.symbol}\n数量: ${form.quantity}`

  if (!confirm(confirmMessage)) {
    return
  }
  if (!form.order_type) {
    err.value = '请选择订单类型'
    return
  }
  if (!form.quantity) {
    err.value = '请输入下单数量'
    return
  }
  if (!triggerLocal.value) {
    err.value = '请选择触发时间'
    return
  }

  try {
    const payload = {
      exchange: form.exchange,
      testnet: form.testnet,
      symbol: form.symbol,
      side: form.side,
      order_type: form.order_type,
      quantity: form.quantity,
      price: form.order_type === 'LIMIT' ? form.price : '',
      leverage: form.leverage,
      reduce_only: form.reduce_only,
      strategy_id: form.strategy_id || null,
      trigger_time: toRFC3339FromLocal(triggerLocal.value),

      // bracket
      bracket_enabled: form.bracket_enabled,
      tp_percent: form.tp_percent,
      sl_percent: form.sl_percent,
      tp_price: form.tp_price,
      sl_price: form.sl_price,
      working_type: form.working_type,
    }

    const r = await api.createScheduledOrder(payload)
    ok.value = r?.id ? `创建成功（ID: ${r.id}）` : '创建成功'

    // 触发父组件回调
    emit('order-created')

  } catch (e) {
    err.value = e?.message || '创建失败'
  }
}

// 策略相关的辅助函数
function getTradingTypeText(tradingType) {
  const typeMap = {
    'futures': '合约交易',
    'spot': '现货交易',
    'both': '两者皆可'
  }
  return typeMap[tradingType] || tradingType
}

function getMarginModeText(marginMode) {
  const modeMap = {
    'isolated': '逐仓模式',
    'cross': '全仓模式'
  }
  return modeMap[marginMode] || marginMode
}

function getMASignalModeText(mode) {
  const modeMap = {
    'cross': '交叉信号',
    'trend': '趋势跟随',
    'both': '交叉+趋势'
  }
  return modeMap[mode] || mode
}

function getMACrossSignalText(signal) {
  const signalMap = {
    'golden_cross': '金叉买入',
    'dead_cross': '死叉卖出',
    'both': '金叉买入+死叉卖出'
  }
  return signalMap[signal] || signal
}

function getMATrendDirectionText(direction) {
  const directionMap = {
    'up': '上涨趋势',
    'down': '下跌趋势',
    'both': '双向趋势'
  }
  return directionMap[direction] || direction
}

function getMeanReversionSubModeText(mode) {
  const modeMap = {
    'bollinger_rsi': '布林带+RSI',
    'channel_rsi': '价格通道+RSI',
    'bollinger_channel': '布林带+价格通道',
    'all': '全指标组合'
  }
  return modeMap[mode] || mode
}

function getMRSignalModeText(mode) {
  const modeMap = {
    'oversold': '超卖信号',
    'overbought': '超买信号',
    'both': '双向信号'
  }
  return modeMap[mode] || mode
}

// 组件挂载时加载数据
import { onMounted } from 'vue'
onMounted(() => {
  loadStrategies()
})
</script>

<style scoped>
/* 这里需要包含所有相关的样式 */
.single-column {
  max-width: 100%;
  margin: 0 auto;
}

.box {
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  margin-bottom: var(--space-6);
  box-shadow: var(--shadow-sm);
}

.box-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-light);
}

.box-header h3 {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.box-description {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin: 0;
}

.order-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.form-section {
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: var(--space-4);
}

.section-title {
  margin: 0 0 var(--space-4) 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--space-4);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.form-label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.required-mark {
  color: var(--error-500);
  font-weight: var(--font-bold);
}

.form-input,
.form-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border-medium);
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: var(--text-sm);
  transition: all var(--transition-fast);
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px var(--primary-100);
}

.form-hint {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: var(--space-1);
}

.form-warning {
  font-size: var(--text-xs);
  color: var(--warning-600);
  margin-top: var(--space-1);
  padding: var(--space-2);
  background: var(--warning-50);
  border-radius: var(--radius-sm);
  border: 1px solid var(--warning-200);
}

.strategy-info-card {
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  margin-top: var(--space-3);
}

.strategy-header h5 {
  margin: 0 0 var(--space-3) 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--primary-600);
}

.strategy-conditions {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.condition-item {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-tertiary);
  border-radius: var(--radius-sm);
  border-left: 3px solid var(--primary-400);
}

.condition-item.basic-info {
  border-left-color: var(--info-500);
  background: var(--info-50);
}

.condition-item.futures-short {
  border-left-color: var(--warning-500);
  background: var(--warning-50);
}

.condition-item.tech-indicator {
  border-left-color: var(--success-500);
  background: var(--success-50);
}

.condition-item.mean-reversion {
  border-left-color: var(--secondary-500);
  background: var(--secondary-50);
}

.condition-item.symbol-filter {
  border-left-color: var(--warning-600);
  background: var(--warning-50);
  color: var(--warning-800);
}

.condition-item.arb-strategy {
  border-left-color: var(--accent-500);
  background: var(--accent-50);
}

.condition-item.risk-control {
  border-left-color: var(--error-500);
  background: var(--error-50);
}

.condition-item.trading-config {
  border-left-color: var(--gray-500);
  background: var(--gray-50);
}

.condition-item.timing-filter {
  border-left-color: var(--purple-500);
  background: var(--purple-50);
}

.condition-item.trading-direction {
  border-left-color: var(--orange-500);
  background: var(--orange-50);
}

.strategy-preview-section {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-light);
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}

.preview-header span {
  font-weight: var(--font-medium);
  color: var(--text-primary);
}

.eligible-symbols-section {
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  margin-top: var(--space-3);
}

.symbols-header {
  margin-bottom: var(--space-3);
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--border-light);
}

.symbols-header span {
  font-weight: var(--font-medium);
  color: var(--text-primary);
}

.symbols-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 300px;
  overflow-y: auto;
}

.symbol-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.symbol-item:hover {
  background: var(--bg-tertiary);
  border-color: var(--primary-300);
}

.symbol-item.selected {
  background: var(--primary-50);
  border-color: var(--primary-400);
}

.symbol-checkbox {
  flex-shrink: 0;
}

.symbol-checkbox input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--primary-500);
}

.symbol-info {
  flex: 1;
  min-width: 0;
}

.symbol-name {
  font-weight: var(--font-medium);
  color: var(--text-primary);
  margin-bottom: var(--space-1);
}

.symbol-details {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.market-cap,
.rank {
  background: var(--bg-tertiary);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}

.triangle-path {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--bg-tertiary);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}

.path-label {
  font-weight: var(--font-medium);
  color: var(--text-primary);
}

.path-symbols {
  font-family: monospace;
  background: var(--bg-primary);
  padding: 2px var(--space-1);
  border-radius: var(--radius-xs);
  border: 1px solid var(--border-light);
}

.price-diff {
  font-weight: var(--font-medium);
}

.price-diff.positive {
  color: var(--success-600);
}

.price-diff.negative {
  color: var(--error-600);
}

.batch-actions {
  margin-top: var(--space-4);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-light);
  text-align: center;
}

.batch-clear-btn {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  border: 1px solid var(--border-medium);
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.batch-clear-btn:hover {
  background: var(--error-50);
  border-color: var(--error-300);
  color: var(--error-600);
}

.form-incomplete-notice {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding: var(--space-3);
  background: var(--warning-50);
  border: 1px solid var(--warning-200);
  border-radius: var(--radius-md);
}

.notice-icon {
  font-size: var(--text-lg);
}

.notice-text {
  font-size: var(--text-sm);
  color: var(--warning-700);
}

.two-column-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-6);
}

.form-column {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.bracket-settings {
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  margin-top: var(--space-3);
}

.bracket-notice {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin-bottom: var(--space-3);
  padding: var(--space-2);
  background: var(--info-50);
  border: 1px solid var(--info-200);
  border-radius: var(--radius-sm);
}

.bracket-divider {
  text-align: center;
  margin: var(--space-4) 0;
  position: relative;
}

.bracket-divider::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--border-light);
}

.bracket-divider span {
  background: var(--bg-primary);
  padding: 0 var(--space-3);
  font-size: var(--text-sm);
  color: var(--text-muted);
  position: relative;
  z-index: 1;
}

.form-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: center;
  flex-wrap: wrap;
  margin-top: var(--space-4);
}

.btn {
  padding: var(--space-3) var(--space-6);
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

.btn-primary {
  background: var(--primary-500);
  color: var(--text-inverse);
}

.btn-primary:hover {
  background: var(--primary-600);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.btn-batch-create {
  background: var(--success-500);
  color: var(--text-inverse);
}

.btn-batch-create:hover {
  background: var(--success-600);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.btn-large {
  padding: var(--space-4) var(--space-8);
  font-size: var(--text-base);
}

.btn-outline {
  background: var(--bg-primary);
  color: var(--primary-600);
  border: 1px solid var(--primary-500);
}

.btn-outline:hover {
  background: var(--primary-50);
  border-color: var(--primary-600);
}

.btn-outline:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-message {
  margin-top: var(--space-4);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  text-align: center;
  font-weight: var(--font-medium);
}

.form-message.success {
  background: var(--success-50);
  color: var(--success-700);
  border: 1px solid var(--success-200);
}

.form-message.error {
  background: var(--error-50);
  color: var(--error-700);
  border: 1px solid var(--error-200);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .form-grid {
    grid-template-columns: 1fr;
  }

  .two-column-layout {
    grid-template-columns: 1fr;
    gap: var(--space-4);
  }

  .form-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .btn {
    width: 100%;
  }

  .symbol-details {
    flex-direction: column;
    gap: var(--space-2);
  }

  .triangle-path {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-1);
  }
}

@media (max-width: 480px) {
  .box {
    padding: var(--space-4);
  }

  .form-section {
    padding: var(--space-3);
  }

  .symbols-list {
    max-height: 200px;
  }

  .symbol-item {
    flex-direction: column;
    align-items: stretch;
    gap: var(--space-2);
  }

  .symbol-checkbox {
    align-self: flex-start;
  }
}
</style>