<template>
  <div class="create-strategy-page">
    <div class="page-header">
      <div class="breadcrumb">
        <RouterLink to="/scheduled-orders" class="breadcrumb-link">交易中心</RouterLink>
        <span class="breadcrumb-separator">></span>
        <span class="breadcrumb-current">{{ isEditMode ? '编辑策略' : '新建策略' }}</span>
      </div>
      <h1 class="page-title">{{ isEditMode ? '编辑交易策略' : '新建交易策略' }}</h1>
      <p class="page-description">{{ isEditMode ? '修改现有的量化交易策略配置' : '创建智能化的量化交易策略，让AI自动执行您的交易逻辑' }}</p>
    </div>

    <div class="strategy-form-container">
      <form @submit.prevent.stop="handleFormSubmit" class="strategy-form">
        <!-- 策略名称 -->
        <div class="form-section">
          <div class="form-group">
            <label class="form-label required">策略名称</label>
            <input
              v-model="strategyForm.name"
              class="form-input"
              :class="{ 'error': validationErrors.name }"
              placeholder="输入策略名称"
              maxlength="50"
              required
            />
            <div class="form-hint">给您的策略起一个清晰易懂的名字</div>
            <div v-if="validationErrors.name" class="field-error">{{ validationErrors.name }}</div>
          </div>
        </div>

        <!-- 策略配置和概览并排显示 -->
        <div class="config-and-overview-row">
          <!-- 配置概览（左侧） -->
          <div class="overview-panel">
            <div class="config-overview">
            <h3 class="section-title">配置概览</h3>
            <p class="section-description">查看当前策略的所有配置条件</p>

            <div class="overview-content">
              <!-- 基础条件概览 -->
              <div class="overview-section">
                <h4 class="overview-section-title">基础条件</h4>
                <div class="overview-items">
                  <div class="overview-item">
                    <span class="item-label">交易对要求：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.conditions.spot_contract, 'status-disabled': !strategyForm.conditions.spot_contract }">
                      {{ strategyForm.conditions.spot_contract ? '✓ 必须有现货+合约' : '⚠️ 无要求' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">策略名称：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.name.trim(), 'status-disabled': !strategyForm.name.trim() }">
                      {{ strategyForm.name.trim() ? `✓ ${strategyForm.name}` : '⚠️ 未设置' }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- 交易配置概览 -->
              <div class="overview-section">
                <h4 class="overview-section-title">交易配置</h4>
                <div class="overview-items">
                  <div class="overview-item">
                    <span class="item-label">交易方向：</span>
                    <span class="item-value" :class="{ 'status-enabled': allowedDirectionsArray.length > 0, 'status-disabled': allowedDirectionsArray.length === 0 }">
                      {{ allowedDirectionsArray.length > 0 ? `✓ ${allowedDirectionsArray.join(' + ')}` : '⚠️ 未选择' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">杠杆配置：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.conditions.enable_leverage, 'status-optional': !strategyForm.conditions.enable_leverage }">
                      {{ strategyForm.conditions.enable_leverage ? `✓ ${strategyForm.conditions.default_leverage}倍杠杆` : '○ 不使用杠杆' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">持仓过滤：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.conditions.skip_held_positions, 'status-optional': !strategyForm.conditions.skip_held_positions }">
                      {{ strategyForm.conditions.skip_held_positions ? '✓ 跳过已有持仓' : '○ 不跳过' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">平仓过滤：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.conditions.skip_close_orders_within_24_hours, 'status-optional': !strategyForm.conditions.skip_close_orders_within_24_hours }">
                      {{ strategyForm.conditions.skip_close_orders_hours > 0 ? `✓ 跳过${strategyForm.conditions.skip_close_orders_hours}h内平仓币种` : '○ 不跳过' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">盈利加仓：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.conditions.profit_scaling_enabled, 'status-optional': !strategyForm.conditions.profit_scaling_enabled }">
                      {{ strategyForm.conditions.profit_scaling_enabled ? `✓ 盈利${strategyForm.conditions.profit_scaling_percent}%时加仓${strategyForm.conditions.profit_scaling_amount}USDT，最多${strategyForm.conditions.profit_scaling_max_count}次` : '○ 未启用' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">整体止盈止损：</span>
                    <span class="item-value" :class="{ 'status-enabled': strategyForm.conditions.overall_stop_loss_enabled, 'status-optional': !strategyForm.conditions.overall_stop_loss_enabled }">
                      {{ getOverallStopLossDisplayText }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- 交易策略概览 -->
              <div class="overview-section">
                <h4 class="overview-section-title">交易策略</h4>
                <div class="overview-items">
                  <div v-if="strategyForm.conditions.no_short_below_market_cap" class="overview-item">
                    <span class="item-label">不开空限制：</span>
                    <span class="item-value">市值 < {{ strategyForm.conditions.market_cap_limit_short }}万不开空</span>
                  </div>
                  <div v-if="strategyForm.conditions.short_on_gainers" class="overview-item">
                    <span class="item-label">涨幅开空：</span>
                    <span class="item-value">
                      市值 > {{ strategyForm.conditions.market_cap_limit_short }}万，前{{ strategyForm.conditions.gainers_rank_limit }}名开空{{ strategyForm.conditions.short_multiplier }}倍
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.long_on_small_gainers" class="overview-item">
                    <span class="item-label">小市值涨幅开多：</span>
                    <span class="item-value">
                      市值 < {{ strategyForm.conditions.market_cap_limit_long }}万，前{{ strategyForm.conditions.gainers_rank_limit_long }}名开多{{ strategyForm.conditions.long_multiplier }}倍
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.futures_price_short_strategy_enabled" class="overview-item">
                    <span class="item-label">合约涨幅开空：</span>
                    <span class="item-value">
                      市值 > {{ strategyForm.conditions.futures_price_short_min_market_cap }}万，前{{ strategyForm.conditions.futures_price_short_max_rank }}名，资金费率 > {{ (strategyForm.conditions.futures_price_short_min_funding_rate * 100).toFixed(2) }}%，开空{{ strategyForm.conditions.futures_price_short_leverage }}倍
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.cross_exchange_arb_enabled" class="overview-item">
                    <span class="item-label">跨交易所套利：</span>
                    <span class="item-value">价差 > {{ strategyForm.conditions.price_diff_threshold }}%，金额 > {{ strategyForm.conditions.min_arb_amount }}USDT</span>
                  </div>
                  <div v-if="strategyForm.conditions.spot_future_arb_enabled" class="overview-item">
                    <span class="item-label">现货-合约套利：</span>
                    <span class="item-value">
                      基差 > {{ strategyForm.conditions.basis_threshold }}% 或资金费率 > {{ strategyForm.conditions.funding_rate_threshold }}%
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.moving_average_enabled" class="overview-item">
                    <span class="item-label">均线策略：</span>
                    <span class="item-value">
                      [{{ strategyForm.conditions.ma_signal_mode === 'QUALITY_FIRST' ? '质量优先' : '数量优先' }}]
                      {{ strategyForm.conditions.ma_type }}({{ strategyForm.conditions.short_ma_period }},{{ strategyForm.conditions.long_ma_period }}) -
                      金叉信号
                      {{ strategyForm.conditions.ma_trend_filter ? '(上升趋势)' : '' }}
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.mean_reversion_enabled" class="overview-item">
                    <span class="item-label">均值回归策略：</span>
                    <span class="item-value">
                      [{{ strategyForm.conditions.mr_signal_mode === 'CONSERVATIVE' ? '保守' : '激进' }}]
                      价格偏离
                      (周期{{ strategyForm.conditions.mr_period }})
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.grid_trading_enabled" class="overview-item">
                    <span class="item-label">网格交易策略：</span>
                    <span class="item-value">
                      价格区间 {{ strategyForm.conditions.grid_lower_price }}-{{ strategyForm.conditions.grid_upper_price }}USDT，
                      {{ strategyForm.conditions.grid_levels }}层网格，
                      利润{{ strategyForm.conditions.grid_profit_percent }}%，
                      投资{{ strategyForm.conditions.grid_investment_amount }}USDT
                    </span>
                  </div>
                  <div v-if="!hasAnyStrategyEnabled" class="overview-item">
                    <span class="item-value status-disabled">⚠️ 暂无交易策略配置</span>
                  </div>
                  <div v-else class="overview-item">
                    <span class="item-value status-enabled">✓ 已配置 {{ getEnabledStrategiesCount() }}{{ getEnabledStrategiesCount() > 1 ? ' 个策略' : ' 个策略' }}</span>
                  </div>
                </div>
              </div>

              <!-- 风险控制概览 -->
              <div class="overview-section">
                <h4 class="overview-section-title">风险控制</h4>
                <div class="overview-items">
                  <div class="overview-item">
                    <span class="item-label">止损：</span>
                    <span class="item-value">
                      {{ strategyForm.conditions.enable_stop_loss ? `✓ ${strategyForm.conditions.stop_loss_percent}%` : '✗ 未启用' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">止盈：</span>
                    <span class="item-value">
                      {{ strategyForm.conditions.enable_take_profit ? `✓ ${strategyForm.conditions.take_profit_percent}%` : '✗ 未启用' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">保证金止损：</span>
                    <span class="item-value">
                      {{ strategyForm.conditions.enable_margin_loss_stop_loss ? `✓ ${strategyForm.conditions.margin_loss_stop_loss_percent}%` : '✗ 未启用' }}
                    </span>
                  </div>
                  <div class="overview-item">
                    <span class="item-label">保证金止盈：</span>
                    <span class="item-value">
                      {{ strategyForm.conditions.enable_margin_profit_take_profit ? `✓ ${strategyForm.conditions.margin_profit_take_profit_percent}%` : '✗ 未启用' }}
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.dynamic_positioning" class="overview-item">
                    <span class="item-label">动态仓位：</span>
                    <span class="item-value">最大{{ strategyForm.conditions.max_position_size }}%，步长{{ strategyForm.conditions.position_size_step }}%</span>
                  </div>
                  <div v-if="strategyForm.conditions.volatility_filter_enabled" class="overview-item">
                    <span class="item-label">波动率过滤：</span>
                    <span class="item-value">
                      > {{ strategyForm.conditions.max_volatility }}% 或周期 > {{ strategyForm.conditions.volatility_period }}天
                    </span>
                  </div>
                </div>
              </div>

              <!-- 市场时机概览 -->
              <div class="overview-section">
                <h4 class="overview-section-title">市场时机</h4>
                <div class="overview-items">
                  <div v-if="strategyForm.conditions.time_filter_enabled" class="overview-item">
                    <span class="item-label">时间过滤：</span>
                    <span class="item-value">
                      UTC {{ strategyForm.conditions.start_hour }}:00 - {{ strategyForm.conditions.end_hour }}:00
                      {{ strategyForm.conditions.weekend_trading ? '(含周末)' : '(工作日)' }}
                    </span>
                  </div>
                  <div v-if="strategyForm.conditions.market_regime_filter_enabled" class="overview-item">
                    <span class="item-label">市场状态过滤：</span>
                    <span class="item-value">
                      阈值{{ strategyForm.conditions.market_regime_threshold }}，
                      偏好{{ strategyForm.conditions.preferred_regime || '不限制' }}
                    </span>
                  </div>
                  <div v-if="!strategyForm.conditions.time_filter_enabled && !strategyForm.conditions.market_regime_filter_enabled" class="overview-item">
                    <span class="item-value text-muted">无时间和市场状态限制</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          </div>

          <!-- 策略配置标签页（右侧） -->
          <div class="config-section">
            <h3 class="section-title">策略配置</h3>
            <p class="section-description">分步骤配置您的量化交易策略</p>

          <!-- 标签页导航 -->
          <div class="tab-navigation">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="tab-button"
              :class="{ active: activeTab === tab.id }"
              @click="switchTab(tab.id, $event)"
            >
              <span class="tab-label">{{ tab.label }}</span>
              <span v-if="getTabCompleteness(tab.id).completed > 0" class="tab-completeness">
                {{ getTabCompleteness(tab.id).completed }}/{{ getTabCompleteness(tab.id).total }}
              </span>
            </button>
          </div>

          <!-- 标签页内容 -->
          <div class="tab-content">

            <!-- 基础设置标签页 -->
            <div v-if="activeTab === 'basic'" class="tab-pane">
              <BasicSettings
                :conditions="strategyForm.conditions"
                :validation-errors="validationErrors"
                @update:directions="handleDirectionsUpdate"
                @update:conditions="handleConditionsUpdate"
              />
            </div>

            <!-- 交易策略标签页 -->
            <TradingStrategies
              v-if="activeTab === 'trading'"
              :conditions="strategyForm.conditions"
              :validation-errors="validationErrors"
              @update:conditions="handleTradingConditionsUpdate"
            />

            <!-- 风险控制标签页 -->
            <RiskManagement
              v-if="activeTab === 'risk'"
              :conditions="strategyForm.conditions"
              :validation-errors="validationErrors"
              @update:conditions="handleRiskConditionsUpdate"
            />

            <!-- 市场时机标签页 -->
            <MarketTiming
              v-if="activeTab === 'timing'"
              :conditions="strategyForm.conditions"
              :validation-errors="validationErrors"
              @update:conditions="handleTimingConditionsUpdate"
            />

          </div>
          </div>

        </div>

        <!-- 表单操作 -->
        <div class="form-actions">
          <RouterLink to="/scheduled-orders" class="btn btn-secondary">取消</RouterLink>
          <button type="submit" class="btn btn-primary" :disabled="saving">
            {{ saving ? (isEditMode ? '更新中...' : '创建中...') : (isEditMode ? '更新策略' : '创建策略') }}
          </button>
        </div>

        <!-- 状态消息 -->
        <div v-if="error || success" class="form-message" :class="{ error: error, success: success }">
          {{ error || success }}
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, watch, computed } from 'vue'
import { RouterLink, useRouter, useRoute } from 'vue-router'
import { api } from '../api/api.js'
import { useAuth } from '../stores/auth.js'
import BasicSettings from '../components/strategy/BasicSettings.vue'
import TradingStrategies from '../components/strategy/TradingStrategies.vue'
import RiskManagement from '../components/strategy/RiskManagement.vue'
import MarketTiming from '../components/strategy/MarketTiming.vue'

const { isAuthed } = useAuth()
const router = useRouter()
const route = useRoute()

// 表单状态
const saving = ref(false)
const error = ref('')
const success = ref('')

// 编辑模式状态 - 在组件初始化时就检查路由参数
const editId = route.query.edit
const isEditMode = ref(!!editId)
const editingStrategyId = ref(editId || null)

// 标签页状态
const activeTab = ref('basic')

// 表单验证状态
const validationErrors = ref({})
// 整体止盈止损显示文本
const getOverallStopLossDisplayText = computed(() => {
  if (!strategyForm.conditions.overall_stop_loss_enabled) {
    return '○ 未启用'
  }

  const stopLoss = strategyForm.conditions.overall_stop_loss_percent
  const takeProfit = strategyForm.conditions.overall_take_profit_percent

  if (stopLoss > 0 && takeProfit > 0) {
    return `✓ 止损${stopLoss}%，止盈${takeProfit}%`
  } else if (stopLoss > 0) {
    return `✓ 止损${stopLoss}%`
  } else if (takeProfit > 0) {
    return `✓ 止盈${takeProfit}%`
  } else {
    return '✓ 已启用（无具体阈值）'
  }
})

// 标签页配置
const tabs = [
  { id: 'basic', label: '基础设置' },
  { id: 'trading', label: '交易策略' },
  { id: 'risk', label: '风险控制' },
  { id: 'timing', label: '市场时机' }
]

// 策略表单
const strategyForm = reactive({
  name: '',
  conditions: {
    // 基础条件 - 默认只勾选交易对要求
    spot_contract: true,

    // 交易配置 - 默认只允许做多，不启用杠杆，默认跳过已在持仓的币种
    allowed_directions: 'LONG', // 后端使用逗号分隔的字符串
    enable_leverage: false,
    default_leverage: 1,
    max_leverage: 10,
    skip_held_positions: true, // 默认启用，避免重复买入
    skip_close_orders_within_24_hours: false, // 已废弃
    skip_close_orders_hours: 0, // 默认不启用，允许重新选择（0表示不跳过）
    profit_scaling_enabled: false, // 默认不启用盈利加仓
    profit_scaling_percent: 5.0, // 默认5%盈利触发
    profit_scaling_amount: 100, // 默认加仓100USDT
    profit_scaling_max_count: 3, // 默认最多加仓3次
    profit_scaling_current_count: 0, // 当前已加仓次数（运行时更新）

    // 整体仓位止盈止损
    overall_stop_loss_enabled: true, // 默认启用整体止损
    overall_stop_loss_percent: 20.0, // 默认20%亏损时止损
    overall_take_profit_enabled: true, // 默认启用整体止盈
    overall_take_profit_percent: 50.0, // 默认50%盈利时止盈

    // 传统交易策略 - 默认不勾选
    no_short_below_market_cap: false,
    market_cap_limit_short: 5000,
    short_on_gainers: false,
    gainers_rank_limit: 7,
    short_multiplier: 3.0,
    long_on_small_gainers: false,
    market_cap_limit_long: 2500,
    gainers_rank_limit_long: 20,
    long_multiplier: 1.0,

    // 技术指标策略 - 默认不勾选
    moving_average_enabled: false,
    ma_signal_mode: 'QUALITY_FIRST', // 默认质量优先
    ma_type: 'SMA',
    short_ma_period: 5,
    long_ma_period: 20,
    ma_cross_signal: 'BOTH',
    ma_trend_filter: false,
    ma_trend_direction: 'UP',

    // 均值回归策略 - 默认不勾选
    mean_reversion_enabled: false,
    mean_reversion_mode: 'enhanced',         // 优化：默认增强模式
    mean_reversion_sub_mode: 'adaptive',     // 优化：默认自适应模式
    mr_bollinger_bands_enabled: true,        // 默认启用布林带
    mr_rsi_enabled: true,                    // 默认启用RSI
    mr_price_channel_enabled: false,         // 默认不启用价格通道
    mr_period: 20,                           // 优化：20周期
    mr_bollinger_multiplier: 2.0,            // 优化：2倍标准差
    mr_rsi_overbought: 75,                   // 优化：75超买 (从70上调)
    mr_rsi_oversold: 25,                     // 优化：25超卖 (从30下调)
    mr_channel_period: 20,                   // 默认20周期
    mr_min_reversion_strength: 0.15,         // 优化：15%强度 (从0.5大幅降低)
    mr_signal_mode: 'ADAPTIVE_OSCILLATION',  // 优化：自适应震荡模式

    // 均值回归风险管理参数 - 根据子模式动态设置
    // 自适应模式：平衡风险和收益
    mr_stop_loss_multiplier: 2.5,            // 止损倍数：2.5倍标准差 (中等宽松)
    mr_take_profit_multiplier: 1.12,         // 止盈倍数：12%收益 (中等收益目标)
    mr_max_position_size: 0.025,             // 最大仓位：2.5% (中等仓位控制)
    mr_max_hold_hours: 36,                   // 最大持仓：36小时 (中等持仓时间)

    // 增强功能默认值 (优化配置)
    market_environment_detection: true,      // 启用市场环境检测
    intelligent_weights: true,               // 启用智能权重
    advanced_risk_management: true,          // 启用高级风险管理
    performance_monitoring: false,           // 默认不启用性能监控

    // 套利策略 - 默认不勾选
    cross_exchange_arb_enabled: false,
    price_diff_threshold: 0.5,
    min_arb_amount: 100,
    spot_future_arb_enabled: false,
    basis_threshold: 0.2,
    funding_rate_threshold: 0.01,
    triangle_arb_enabled: false,
    triangle_threshold: 0.1,
    base_symbols: '',
    stat_arb_enabled: false,
    cointegration_period: 30,

    // 网格交易策略 - 默认不勾选
    grid_trading_enabled: false,
    grid_upper_price: 0,
    grid_lower_price: 0,
    grid_levels: 10,
    grid_profit_percent: 1.0,
    grid_investment_amount: 1000,
    grid_rebalance_enabled: true,
    grid_stop_loss_enabled: true,
    grid_stop_loss_percent: 10.0,

    // 币种选择 - 网格策略默认启用白名单
    use_symbol_whitelist: true,
    symbol_whitelist: [],

    // 风险控制 - 默认启用基础止损止盈
    enable_stop_loss: true,
    stop_loss_percent: 2.0,
    enable_take_profit: true,
    take_profit_percent: 5.0,

    // 保证金损失止损 - 默认不启用
    enable_margin_loss_stop_loss: false,
    margin_loss_stop_loss_percent: 30.0,

    // 保证金盈利止盈 - 默认不启用
    enable_margin_profit_take_profit: false,
    margin_profit_take_profit_percent: 100.0,
    dynamic_positioning: false,
    max_position_size: 20,
    position_size_step: 1.0,
    volatility_filter_enabled: false,
    max_volatility: 50,
    volatility_period: 30,

    // 市场时机
    time_filter_enabled: false,
    start_hour: 9,
    end_hour: 17,
    weekend_trading: false,
    market_regime_filter_enabled: false,
    market_regime_threshold: 0.1,
    preferred_regime: '',
  }
})

// 交易方向数组（用于多选框）
const allowedDirectionsArray = ref(['LONG'])

// 监听交易方向变化
watch(allowedDirectionsArray, (newValue) => {
  strategyForm.conditions.allowed_directions = newValue.join(',')
}, { immediate: true })

// 计算属性：检查是否有任何交易策略被启用
const hasAnyStrategyEnabled = computed(() => {
  return strategyForm.conditions.short_on_gainers ||
         strategyForm.conditions.long_on_small_gainers ||
         strategyForm.conditions.futures_price_short_strategy_enabled ||
         strategyForm.conditions.cross_exchange_arb_enabled ||
         strategyForm.conditions.spot_future_arb_enabled ||
         strategyForm.conditions.triangle_arb_enabled ||
         strategyForm.conditions.stat_arb_enabled ||
         strategyForm.conditions.moving_average_enabled ||
         strategyForm.conditions.mean_reversion_enabled ||
         strategyForm.conditions.grid_trading_enabled
})

// 表单验证规则
const validationRules = {
  name: {
    required: true,
    minLength: 2,
    maxLength: 50,
    pattern: /^[^\s].*[^\s]$/,
    message: {
      required: '策略名称不能为空',
      minLength: '策略名称至少需要2个字符',
      maxLength: '策略名称不能超过50个字符',
      pattern: '策略名称首尾不能有空格'
    }
  }
}

// 验证单个字段
function validateField(fieldName, value) {
  const rule = validationRules[fieldName]
  if (!rule) return null

  if (rule.required && (!value || value.toString().trim() === '')) {
    return rule.message.required
  }

  if (value && rule.minLength && value.length < rule.minLength) {
    return rule.message.minLength
  }

  if (value && rule.maxLength && value.length > rule.maxLength) {
    return rule.message.maxLength
  }

  if (value && rule.pattern && !rule.pattern.test(value)) {
    return rule.message.pattern
  }

  return null
}

// 整体表单验证
function validateForm() {
  const errors = {}
  let isValid = true

  // 验证策略名称
  const nameError = validateField('name', strategyForm.name)
  if (nameError) {
    errors.name = nameError
    isValid = false
  }

  // 验证至少选择一个交易方向
  if (allowedDirectionsArray.value.length === 0) {
    errors.directions = '请至少选择一个交易方向'
    isValid = false
  }

  // 验证至少启用一个交易策略
  if (!hasAnyStrategyEnabled.value) {
    errors.strategy = '请至少启用一个交易策略'
    isValid = false
  }

  // 验证均值回归策略配置
  if (strategyForm.conditions.mean_reversion_enabled) {
    const mrIndicators = [
      strategyForm.conditions.mr_bollinger_bands_enabled,
      strategyForm.conditions.mr_rsi_enabled,
      strategyForm.conditions.mr_price_channel_enabled
    ].filter(Boolean)

    if (mrIndicators.length === 0) {
      errors.mean_reversion = '均值回归策略至少要启用一个指标'
      isValid = false
    }
  }

  validationErrors.value = errors
  return isValid
}


// 获取已启用策略数量
function getEnabledStrategiesCount() {
  const strategies = [
    'short_on_gainers', 'long_on_small_gainers', 'futures_price_short_strategy_enabled',
    'cross_exchange_arb_enabled', 'spot_future_arb_enabled',
    'moving_average_enabled', 'mean_reversion_enabled'
  ]
  return strategies.filter(key => strategyForm.conditions[key]).length
}

// 处理基础设置组件的方向更新
function handleDirectionsUpdate(directions) {
  allowedDirectionsArray.value = directions
}

// 处理基础设置组件的条件更新
function handleConditionsUpdate(conditions) {
  Object.assign(strategyForm.conditions, conditions)
}

// 处理交易策略组件的条件更新
function handleTradingConditionsUpdate(conditions) {
  Object.assign(strategyForm.conditions, conditions)
  // 当均值回归子模式改变时，更新风险管理参数
  updateMRRiskParamsForSubMode()
}

// 根据均值回归子模式更新风险管理参数
function updateMRRiskParamsForSubMode() {
  const subMode = strategyForm.conditions.mean_reversion_sub_mode

  switch (subMode) {
    case 'conservative':
      // 保守模式：高止损倍数、低止盈倍数、小仓位、长持仓
      strategyForm.conditions.mr_stop_loss_multiplier = 3.0   // 3倍标准差，宽松止损
      strategyForm.conditions.mr_take_profit_multiplier = 1.06 // 6%止盈，保守收益目标
      strategyForm.conditions.mr_max_position_size = 0.015    // 1.5%仓位，严格控制风险
      strategyForm.conditions.mr_max_hold_hours = 48          // 48小时，等待合适时机
      break

    case 'aggressive':
      // 激进模式：低止损倍数、高止盈倍数、大仓位、短持仓
      strategyForm.conditions.mr_stop_loss_multiplier = 2.0   // 2倍标准差，严格止损
      strategyForm.conditions.mr_take_profit_multiplier = 1.20 // 20%止盈，激进收益目标
      strategyForm.conditions.mr_max_position_size = 0.04     // 4%仓位，充分利用资金
      strategyForm.conditions.mr_max_hold_hours = 12          // 12小时，快速进出
      break

    case 'adaptive':
    default:
      // 自适应模式：平衡参数，智能调整
      strategyForm.conditions.mr_stop_loss_multiplier = 2.5   // 2.5倍标准差，中等宽松
      strategyForm.conditions.mr_take_profit_multiplier = 1.12 // 12%止盈，中等收益目标
      strategyForm.conditions.mr_max_position_size = 0.025    // 2.5%仓位，中等仓位控制
      strategyForm.conditions.mr_max_hold_hours = 36          // 36小时，中等持仓时间
      break
  }
}

// 处理风险控制组件的条件更新
function handleRiskConditionsUpdate(conditions) {
  Object.assign(strategyForm.conditions, conditions)
}

// 处理市场时机组件的条件更新
function handleTimingConditionsUpdate(conditions) {
  Object.assign(strategyForm.conditions, conditions)
}


// 获取标签页完成度
function getTabCompleteness(tabId) {
  let completed = 0
  let total = 0

  switch (tabId) {
    case 'basic':
      total = 3
      if (strategyForm.name.trim()) completed++
      if (allowedDirectionsArray.value.length > 0) completed++
      if (strategyForm.conditions.spot_contract) completed++
      break
    case 'trading':
      total = 6
      if (strategyForm.conditions.short_on_gainers) completed++
      if (strategyForm.conditions.long_on_small_gainers) completed++
      if (strategyForm.conditions.futures_price_short_strategy_enabled) completed++
      if (strategyForm.conditions.cross_exchange_arb_enabled) completed++
      if (strategyForm.conditions.spot_future_arb_enabled) completed++
      if (strategyForm.conditions.moving_average_enabled) completed++
      break
    case 'risk':
      total = 4
      if (strategyForm.conditions.enable_stop_loss) completed++
      if (strategyForm.conditions.enable_take_profit) completed++
      if (strategyForm.conditions.enable_margin_loss_stop_loss) completed++
      if (strategyForm.conditions.dynamic_positioning) completed++
      if (strategyForm.conditions.volatility_filter_enabled) completed++
      break
    case 'timing':
      total = 2
      if (strategyForm.conditions.time_filter_enabled) completed++
      if (strategyForm.conditions.market_regime_filter_enabled) completed++
      break
  }

  return {
    completed,
    total,
    percentage: total > 0 ? Math.round((completed / total) * 100) : 0
  }
}

// 切换标签页
function switchTab(tabId, event) {
  // 防止事件冒泡和默认行为，确保不会触发表单提交
  if (event) {
    event.preventDefault()
    event.stopPropagation()
    event.stopImmediatePropagation()
  }

  activeTab.value = tabId
}

// 处理表单提交
function handleFormSubmit(event) {
  // 确保阻止默认行为和事件冒泡
  event.preventDefault()
  event.stopPropagation()
  event.stopImmediatePropagation()

  saveStrategy()
}

// 保存策略
async function saveStrategy() {
  if (!validateForm()) {
    error.value = '请检查表单填写是否正确'
    return
  }

  saving.value = true
  error.value = ''
  success.value = ''

  try {
    let response
    if (isEditMode.value) {
      response = await api.updateTradingStrategy(editingStrategyId.value, strategyForm)
      success.value = `策略"${strategyForm.name}"更新成功！`
    } else {
      response = await api.createTradingStrategy(strategyForm)
    success.value = `策略"${strategyForm.name}"创建成功！`
    }

    // 1秒后跳转回策略列表
    setTimeout(() => {
      router.push('/scheduled-orders')
    }, 1000)
  } catch (e) {
    error.value = e?.message || '创建策略失败'
    // 如果是验证错误，显示在对应字段
    if (e?.message?.includes('名称')) {
      validationErrors.value.name = e.message
    }
  } finally {
    saving.value = false
  }
}

// 页面标题和数据加载
onMounted(async () => {
  // 设置页面标题
  document.title = isEditMode.value ? '编辑策略 - 区块链量化交易平台' : '新建策略 - 区块链量化交易平台'

  // 初始化交易方向数组
  if (!isEditMode.value) {
    // 新建模式：根据默认值设置数组
    if (strategyForm.conditions.allowed_directions) {
      allowedDirectionsArray.value = strategyForm.conditions.allowed_directions.split(',').filter(dir => dir.trim() !== '')
    }
  }

  // 数据初始化现在由各个组件自行处理


  // 如果是编辑模式，加载现有策略数据
  if (isEditMode.value && editingStrategyId.value) {
    try {
      const response = await api.getTradingStrategy(editingStrategyId.value)
      if (response.success) {
        const strategy = response.data
        console.log('加载的策略数据:', strategy)

        // 填充表单数据 - 确保正确映射字段
        strategyForm.name = strategy.name || ''
        // 深度合并 conditions，确保所有现有字段都被正确设置
        if (strategy.conditions) {
          Object.assign(strategyForm.conditions, strategy.conditions)

          // 特殊处理：将交易方向字符串转换为数组
          if (strategy.conditions.allowed_directions) {
            allowedDirectionsArray.value = strategy.conditions.allowed_directions.split(',').filter(dir => dir.trim() !== '')
          } else {
            // 如果没有设置，默认只选择LONG
            allowedDirectionsArray.value = ['LONG']
          }
        }

        console.log('填充后的表单数据:', strategyForm)
      } else {
        error.value = '加载策略失败：' + response.message
      }
    } catch (err) {
      error.value = '加载策略失败：' + err.message
    }
  }
})

</script>

<style scoped>
/* CSS 变量定义 */
.create-strategy-page {
  --primary-color: #3b82f6;
  --primary-hover: #2563eb;
  --success-color: #10b981;
  --warning-color: #f59e0b;
  --error-color: #dc2626;
  --text-primary: #111827;
  --text-secondary: #6b7280;
  --text-muted: #9ca3af;
  --bg-primary: #ffffff;
  --bg-secondary: #f9fafb;
  --bg-tertiary: #f3f4f6;
  --border-color: #e5e7eb;
  --border-hover: #d1d5db;
  --shadow-sm: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --transition: all 0.2s ease;
}

.create-strategy-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.page-header {
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e5e7eb;
}

.breadcrumb {
  margin-bottom: 10px;
  font-size: 14px;
  color: #6b7280;
}

.breadcrumb-link {
  color: #3b82f6;
  text-decoration: none;
}

.breadcrumb-link:hover {
  text-decoration: underline;
}

.breadcrumb-separator {
  margin: 0 8px;
}

.breadcrumb-current {
  color: #374151;
  font-weight: 500;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: #111827;
  margin: 0 0 8px 0;
}

.page-description {
  font-size: 16px;
  color: #6b7280;
  margin: 0;
}

.strategy-form-container {
  background: var(--bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
}

.strategy-form {
  padding: 30px;
}

.form-section {
  margin-bottom: 40px;
}

.form-section:last-child {
  margin-bottom: 0;
}

.section-title {
  font-size: 20px;
  font-weight: 600;
  color: #111827;
  margin: 0 0 8px 0;
}

.section-description {
  font-size: 14px;
  color: #6b7280;
  margin: 0 0 24px 0;
}

/* 标签页样式 */
.tab-navigation {
  display: flex;
  border-bottom: 1px solid #e5e7eb;
  margin-bottom: 32px;
  overflow-x: auto;
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: #6b7280;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.tab-button:hover {
  color: #374151;
  background: #f9fafb;
}

.tab-button.active {
  color: #3b82f6;
  border-bottom-color: #3b82f6;
}

.tab-label {
  font-weight: 600;
}

.tab-completeness {
  font-size: 11px;
  font-weight: 600;
  background: var(--success-color);
  color: white;
  padding: 2px 6px;
  border-radius: 10px;
  margin-left: 6px;
  min-width: 24px;
  text-align: center;
}

.tab-content {
  min-height: 400px;
}

.tab-pane {
  animation: fadeIn 0.3s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.form-group {
  margin-bottom: 24px;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  margin-bottom: 8px;
}

.form-label.required::after {
  content: ' *';
  color: #ef4444;
}

.form-input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--border-hover);
  border-radius: var(--radius-md);
  font-size: 16px;
  transition: var(--transition);
  background: var(--bg-primary);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input.error {
  border-color: var(--error-color);
  box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.1);
}

.form-input::placeholder {
  color: var(--text-muted);
}

.form-hint {
  margin-top: 4px;
  font-size: 12px;
  color: #6b7280;
}

.field-error {
  margin-top: 4px;
  font-size: 12px;
  color: #dc2626;
  display: flex;
  align-items: center;
  gap: 4px;
}

.field-error::before {
  content: '⚠️';
  font-size: 11px;
}


.form-input.error {
  border-color: #dc2626;
  box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.1);
}

/* 配置概览样式 */
.config-overview {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 20px;
}




.overview-content {
  animation: fadeIn 0.3s ease-in-out;
}


.overview-section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 12px 0;
  padding-bottom: 8px;
  border-bottom: 2px solid #e2e8f0;
}

.overview-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.overview-item {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 4px 0;
  border-bottom: 1px solid #f1f5f9;
  flex-direction: column;
  gap: 2px;
}

.overview-item:last-child {
  border-bottom: none;
}

.item-label {
  font-weight: 500;
  color: #64748b;
  font-size: 13px;
}

.item-value {
  font-weight: 500;
  color: #1e293b;
  font-size: 13px;
  line-height: 1.4;
  word-break: break-word;
}

.item-value.text-muted {
  color: #94a3b8;
  font-style: italic;
}

.status-enabled {
  color: #10b981;
  font-weight: 500;
}

.status-disabled {
  color: #f59e0b;
  font-weight: 500;
}

.status-optional {
  color: #6b7280;
  font-weight: 400;
}

/* 并排布局样式 */
.config-and-overview-row {
  display: grid;
  grid-template-columns: 350px 1fr;
  gap: 24px;
  margin-bottom: 32px;
}

.config-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
  padding: 24px;
  height: fit-content;
}

.overview-panel {
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
  height: fit-content;
  position: sticky;
  top: 20px;
}

.overview-section {
  margin-bottom: 16px;
  padding: 14px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid #e5e7eb;
}

/* 通用按钮样式 */
.btn {
  padding: 12px 24px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 500;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: var(--transition);
  border: none;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--primary-hover);
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-hover);
}

.btn-secondary:hover:not(:disabled) {
  background: #e5e7eb;
}

.form-message {
  margin-top: 16px;
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-message.error {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.form-message.success {
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #bbf7d0;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .config-and-overview-row {
    grid-template-columns: 320px 1fr;
    gap: 20px;
  }
}

@media (max-width: 768px) {

  .create-strategy-page {
    padding: 16px;
  }

  .strategy-form {
    padding: 20px;
  }

  .page-title {
    font-size: 24px;
  }

  .form-actions {
    flex-direction: column;
    gap: 8px;
  }


  .tab-navigation {
    padding: 0 4px;
    overflow-x: auto;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }

  .tab-navigation::-webkit-scrollbar {
    display: none;
  }

  .tab-button {
    padding: 10px 12px;
    font-size: 13px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* 移动端并排布局变为垂直 */
  .config-and-overview-row {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .config-section {
    padding: 16px;
    max-height: none;
    overflow-y: visible;
    position: static;
  }

  .overview-panel {
    max-height: none;
    overflow-y: visible;
    position: static;
    top: auto;
  }

  .overview-section {
    padding: 12px;
  }

  .overview-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    padding: 8px 0;
  }

  .item-label {
    min-width: auto;
    font-size: 13px;
  }

  .item-value {
    text-align: left;
    font-size: 13px;
    line-height: 1.4;
  }

@media (max-width: 480px) {
  .create-strategy-page {
    padding: 12px;
  }

  .strategy-form {
    padding: 16px;
  }

  .page-title {
    font-size: 20px;
  }

  .tab-label {
    display: none;
  }

  .tab-button {
    padding: 8px 10px;
    min-width: 60px;
    justify-content: center;
  }

  .config-overview {
    padding: 12px;
  }

  .overview-section {
    padding: 8px;
    margin-bottom: 12px;
  }

  .overview-section-title {
    font-size: 15px;
    margin-bottom: 8px;
  }
}


  .item-label, .item-value {
    font-size: 13px;
  }

  .form-actions .btn {
    width: 100%;
    padding: 14px;
    font-size: 15px;
  }
}
</style>
