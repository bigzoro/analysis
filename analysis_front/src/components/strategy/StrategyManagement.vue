<template>
  <div class="strategy-management-tab-content">
    <div class="row topbar">
      <div class="spacer"></div>
      <RouterLink to="/create-strategy" class="btn primary" :class="{ disabled: !isAuthed }" :tabindex="!isAuthed ? -1 : undefined">新建策略</RouterLink>
      <button class="btn" @click="loadStrategiesForManagement" :disabled="!isAuthed">刷新</button>
    </div>

    <div class="grid strategies-container">
      <!-- 策略列表 -->
      <div class="box">
        <h3 class="strategies-title">策略列表</h3>
        <div v-if="!isAuthed" class="empty">
          请先 <RouterLink to="/login">登录</RouterLink> 才能管理交易策略
        </div>
        <div v-else-if="strategiesLoading" class="loading">加载中...</div>
        <div v-else-if="strategies.length === 0" class="empty">暂无策略</div>
        <div class="strategies-grid">
          <div v-for="strategy in strategies" :key="strategy.id" class="strategy-item">
            <div class="strategy-header">
              <div class="strategy-title-section">
                <h4>{{ strategy.name }}</h4>
                <div v-if="strategy.is_running" class="strategy-health" @click="showStrategyHealth(strategy.id)">
                  <span class="health-indicator" :class="getHealthStatus(strategy)"></span>
                  <span class="health-text">{{ getHealthText(strategy) }}</span>
                </div>
              </div>
              <div class="strategy-actions">
                <button
                  v-if="!strategy.is_running"
                  class="btn btn-run"
                  @click="startStrategy(strategy.id)"
                  :disabled="startingStrategy === strategy.id"
                >
                  {{ startingStrategy === strategy.id ? '启动中...' : '启动' }}
                </button>
                <button
                  v-else
                  class="btn btn-stop"
                  @click="stopStrategy(strategy.id)"
                  :disabled="stoppingStrategy === strategy.id"
                >
                  {{ stoppingStrategy === strategy.id ? '停止中...' : '停止' }}
                </button>
                <button class="btn btn-stats" @click="viewStrategyStats(strategy.id)" title="查看运行统计">
                  统计
                </button>
                <button class="btn btn-backtest" @click="backtestStrategy(strategy)" title="策略回测分析">
                  回测
                </button>
                <button class="btn btn-edit" @click="editStrategy(strategy)">
                  编辑
                </button>
                <button class="btn btn-delete" @click="deleteStrategy(strategy.id)">
                  删除
                </button>
              </div>
            </div>
            <div class="strategy-content">
              <div class="condition-summary">
                <!-- 基础信息显示 -->
                <div v-if="strategy.conditions.trading_type && strategy.conditions.trading_type !== ''" class="condition-tag basic-info">
                  交易类型: {{ getTradingTypeText(strategy.conditions.trading_type) }}
                </div>
                <div v-if="strategy.conditions.margin_mode" class="condition-tag basic-info">
                  保证金模式: {{ getMarginModeText(strategy.conditions.margin_mode) }}
                </div>
                <div v-if="strategy.conditions.spot_contract" class="condition-tag">
                  需要现货+合约
                </div>
                <div v-if="strategy.conditions.no_short_below_market_cap" class="condition-tag">
                  市值<{{ strategy.conditions.market_cap_limit_short }}万不开空
                </div>
                <div v-if="strategy.conditions.short_on_gainers" class="condition-tag">
                  涨幅前{{ strategy.conditions.gainers_rank_limit }} & 市值>{{ strategy.conditions.market_cap_limit_short }}万 → 开空{{ strategy.conditions.short_multiplier }}倍
                </div>
                <div v-if="strategy.conditions.long_on_small_gainers" class="condition-tag">
                  市值<{{ strategy.conditions.market_cap_limit_long }}万 & 涨幅前{{ strategy.conditions.gainers_rank_limit_long }} → 开多{{ strategy.conditions.long_multiplier }}倍
                </div>

                <!-- 合约涨幅开空策略 -->
                <div v-if="strategy.conditions.futures_price_short_strategy_enabled" class="condition-tag futures-short">
                  📈 合约涨幅开空: 市值>{{ strategy.conditions.futures_price_short_min_market_cap }}万 & 前{{ strategy.conditions.futures_price_short_max_rank }}名 & 资金费率>{{ strategy.conditions.futures_price_short_min_funding_rate }}% → 开空{{ strategy.conditions.futures_price_short_leverage }}倍
                </div>

                <!-- 技术指标策略条件 -->
                <div v-if="strategy.conditions.moving_average_enabled" class="condition-tag tech-indicator">
                  📈 均线策略: [{{ getMASignalModeText(strategy.conditions.ma_signal_mode) }}] {{ strategy.conditions.ma_type }}({{ strategy.conditions.short_ma_period }},{{ strategy.conditions.long_ma_period }}) {{ getMACrossSignalText(strategy.conditions.ma_cross_signal) }}{{ strategy.conditions.ma_trend_filter ? '(' + getMATrendDirectionText(strategy.conditions.ma_trend_direction) + ')' : '' }}
                </div>

                <!-- 均值回归策略条件 -->
                <div v-if="strategy.conditions.mean_reversion_enabled" class="condition-tag mean-reversion">
                  🔄 均值回归策略
                  <span v-if="strategy.conditions.mean_reversion_mode === 'enhanced'">
                    [{{ getMeanReversionSubModeText(strategy.conditions.mean_reversion_sub_mode) }}]
                  </span>
                  <span v-else>
                    [{{ getMRSignalModeText(strategy.conditions.mr_signal_mode) }}]
                  </span>
                  : 周期{{ strategy.conditions.mr_period }}天
                  <span v-if="strategy.conditions.mr_bollinger_bands_enabled"> | 布林带{{ strategy.conditions.mr_bollinger_multiplier }}倍</span>
                  <span v-if="strategy.conditions.mr_rsi_enabled"> | RSI({{ strategy.conditions.mr_rsi_oversold }}-{{ strategy.conditions.mr_rsi_overbought }})</span>
                  <span v-if="strategy.conditions.mr_price_channel_enabled"> | 价格通道{{ strategy.conditions.mr_channel_period }}天</span>
                  <span v-if="strategy.conditions.mr_min_reversion_strength"> | 最小强度{{ strategy.conditions.mr_min_reversion_strength }}</span>
                </div>

                <!-- 套利策略条件 -->
                <div v-if="strategy.conditions.cross_exchange_arb_enabled" class="condition-tag arb-strategy">
                  🔄 跨交易所套利 (价差>{{ strategy.conditions.price_diff_threshold }}%)
                </div>
                <div v-if="strategy.conditions.spot_future_arb_enabled" class="condition-tag arb-strategy">
                  🔄 现货-合约套利 (基差>{{ strategy.conditions.basis_threshold }}%)
                </div>
                <div v-if="strategy.conditions.triangle_arb_enabled" class="condition-tag arb-strategy">
                  🔄 三角套利 (阈值>{{ strategy.conditions.triangle_threshold }}%，自动选择币种)
                </div>
                <div v-if="strategy.conditions.stat_arb_enabled" class="condition-tag arb-strategy">
                  🔄 统计套利 (Z分数>{{ strategy.conditions.zscore_threshold }})
                </div>
                <div v-if="strategy.conditions.futures_spot_arb_enabled" class="condition-tag arb-strategy">
                  🔄 期现套利 (到期<{{ strategy.conditions.expiry_threshold }}天, 价差>{{ strategy.conditions.spot_future_spread }}%)
                </div>

                <!-- 风险控制条件 -->
                <div v-if="strategy.conditions.enable_stop_loss" class="condition-tag risk-control">
                  🛡️ 止损: {{ strategy.conditions.stop_loss_percent }}%
                </div>
                <div v-if="strategy.conditions.enable_take_profit" class="condition-tag risk-control">
                  🛡️ 止盈: {{ strategy.conditions.take_profit_percent }}%
                </div>
                <div v-if="strategy.conditions.enable_margin_loss_stop_loss" class="condition-tag risk-control">
                  💰 保证金止损: {{ strategy.conditions.margin_loss_stop_loss_percent }}%
                </div>
                <div v-if="strategy.conditions.enable_margin_profit_take_profit" class="condition-tag risk-control">
                  💰 保证金止盈: {{ strategy.conditions.margin_profit_take_profit_percent }}%
                </div>
                <div v-if="strategy.conditions.enable_leverage" class="condition-tag risk-control">
                  ⚡ 杠杆: {{ strategy.conditions.default_leverage }}倍
                </div>
                <div v-if="strategy.conditions.dynamic_positioning" class="condition-tag risk-control">
                  📊 动态仓位: 最大{{ strategy.conditions.max_position_size }}%，步长{{ strategy.conditions.position_size_step }}%
                </div>
                <div v-if="strategy.conditions.volatility_filter_enabled" class="condition-tag risk-control">
                  📈 波动率过滤: >{{ strategy.conditions.max_volatility }}% 或 {{ strategy.conditions.volatility_period }}天
                </div>

                <!-- 交易配置条件 -->
                <div v-if="strategy.conditions.skip_held_positions" class="condition-tag trading-config">
                  🚫 跳过已有持仓
                </div>
                <div v-if="strategy.conditions.skip_close_orders_hours > 0" class="condition-tag trading-config">
                  🕐 跳过{{ strategy.conditions.skip_close_orders_hours }}h内平仓币种
                </div>
                <div v-if="strategy.conditions.use_symbol_whitelist && strategy.conditions.symbol_whitelist && strategy.conditions.symbol_whitelist.length > 0" class="condition-tag symbol-filter">
                  📋 白名单: {{ strategy.conditions.symbol_whitelist.join(', ') }}
                </div>
                <div v-if="strategy.conditions.use_symbol_blacklist && strategy.conditions.symbol_blacklist && strategy.conditions.symbol_blacklist.length > 0" class="condition-tag symbol-filter">
                  🚫 黑名单: {{ strategy.conditions.symbol_blacklist.join(', ') }}
                </div>
                <div v-if="strategy.conditions.profit_scaling_enabled" class="condition-tag trading-config">
                  📈 盈利{{ strategy.conditions.profit_scaling_percent }}%加仓{{ strategy.conditions.profit_scaling_amount }}USDT (最多{{ strategy.conditions.profit_scaling_max_count }}次)
                </div>
                <div v-if="strategy.conditions.overall_stop_loss_enabled" class="condition-tag risk-control">
                  🛡️ {{ getOverallStopLossText(strategy.conditions) }}
                </div>

                <!-- 时间和市场过滤条件 -->
                <div v-if="strategy.conditions.time_filter_enabled" class="condition-tag timing-filter">
                  🕐 时间过滤: {{ strategy.conditions.start_hour }}:00-{{ strategy.conditions.end_hour }}:00{{ strategy.conditions.weekend_trading ? '(含周末)' : '(工作日)' }}
                </div>
                <div v-if="strategy.conditions.market_regime_filter_enabled" class="condition-tag timing-filter">
                  📊 市场过滤: 阈值{{ strategy.conditions.market_regime_threshold }}，偏好{{ strategy.conditions.preferred_regime || '不限制' }}
                </div>

                <!-- 交易方向 -->
                <div v-if="strategy.conditions.allowed_directions && strategy.conditions.allowed_directions !== 'LONG'" class="condition-tag trading-direction">
                  📈 方向: {{ strategy.conditions.allowed_directions.replace(',', '+') }}
                </div>
              </div>
              <div class="strategy-meta">
                <small class="muted">创建时间: {{ formatDate(strategy.created_at) }}</small>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 策略启动参数弹窗 -->
    <div v-if="showStartStrategyModal" class="modal-overlay" @click="cancelStartStrategy">
      <div class="modal strategy-start-modal" @click.stop>
        <div class="modal-header">
          <h3>启动策略运行</h3>
          <button class="modal-close" @click="cancelStartStrategy">&times;</button>
        </div>
        <div class="modal-body">
          <div class="start-strategy-form">
            <!-- 运行间隔 -->
            <div class="form-group">
              <label class="form-label">
                运行间隔
                <span v-if="!startStrategyForm.auto_stop" class="required-mark">*</span>
              </label>
              <select
                v-model="startStrategyForm.run_interval"
                class="form-select"
                :disabled="startStrategyForm.auto_stop"
              >
                <option :value="1">1分钟</option>
                <option :value="3">3分钟</option>
                <option :value="5">5分钟</option>
                <option :value="15">15分钟</option>
                <option :value="30">30分钟</option>
                <option :value="60">1小时</option>
                <option :value="120">2小时</option>
                <option :value="240">4小时</option>
                <option :value="480">8小时</option>
                <option :value="1440">1天</option>
              </select>
              <div class="form-hint" :class="{ 'text-muted': startStrategyForm.auto_stop }">
                {{ startStrategyForm.auto_stop ? '执行一次后自动停止时，此设置无效' : '策略每次执行的时间间隔' }}
              </div>
            </div>

            <!-- 最大运行次数 -->
            <div class="form-group">
              <label class="form-label">最大运行次数</label>
              <input
                v-model.number="startStrategyForm.max_runs"
                type="number"
                class="form-input"
                min="0"
                :disabled="startStrategyForm.auto_stop"
                :placeholder="startStrategyForm.auto_stop ? '自动设置为1' : '0表示无限运行'"
              />
              <div class="form-hint" :class="{ 'text-muted': startStrategyForm.auto_stop }">
                {{ startStrategyForm.auto_stop ? '执行一次后自动停止时，自动设置为1次' : '达到指定次数后自动停止，0表示无限运行' }}
              </div>
            </div>

            <!-- 自动停止选项 -->
            <div class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="startStrategyForm.auto_stop" />
                执行一次后自动停止
              </label>
              <div class="form-hint">选中后策略执行一次后会自动停止运行状态</div>
            </div>

            <!-- 自动创建订单选项 -->
            <div class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="startStrategyForm.create_orders" />
                自动创建订单
              </label>
              <div class="form-hint">当策略发现符合条件的交易对时，自动创建定时订单</div>
            </div>

            <!-- 执行延迟设置 -->
            <div v-if="startStrategyForm.create_orders" class="form-group">
              <label class="form-label">执行延迟</label>
              <select
                v-model="startStrategyForm.execution_delay"
                class="form-select"
              >
                <option :value="30">30秒</option>
                <option :value="60">1分钟</option>
                <option :value="120">2分钟</option>
                <option :value="300">5分钟</option>
                <option :value="600">10分钟</option>
              </select>
              <div class="form-hint">订单创建后延迟执行的时间，避免市场波动</div>
            </div>

            <!-- 每一单金额设置 -->
            <div v-if="startStrategyForm.create_orders" class="form-group">
              <label class="form-label">每一单金额</label>
              <input
                v-model.number="startStrategyForm.per_order_amount"
                type="number"
                class="form-input"
                min="0"
                step="0.01"
                placeholder="0表示使用默认金额"
              />
              <div class="form-hint">每单交易使用的USDT金额，0表示使用系统默认金额</div>
            </div>
          </div>

          <div class="form-actions">
            <button class="btn btn-secondary" @click="cancelStartStrategy">取消</button>
            <button class="btn btn-primary" @click="confirmStartStrategy" :disabled="startingStrategy">
              {{ startingStrategy ? '启动中...' : '启动策略' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { api } from '../../api/api.js'
import { useAuth } from '../../stores/auth.js'

const { isAuthed } = useAuth()
const router = useRouter()

// 整体止盈止损显示文本
const getOverallStopLossText = (conditions) => {
  const stopLoss = conditions.overall_stop_loss_percent
  const takeProfit = conditions.overall_take_profit_percent

  if (stopLoss > 0 && takeProfit > 0) {
    return `整体止损${stopLoss}%，止盈${takeProfit}%`
  } else if (stopLoss > 0) {
    return `整体止损${stopLoss}%`
  } else if (takeProfit > 0) {
    return `整体止盈${takeProfit}%`
  } else {
    return '整体止盈止损（无具体阈值）'
  }
}

// 策略管理相关状态
const strategies = ref([])
const strategiesLoading = ref(false)

// 策略运行相关状态
const startingStrategy = ref(null)
const stoppingStrategy = ref(null)

// 策略启动参数弹窗
const showStartStrategyModal = ref(false)
const startStrategyForm = reactive({
  strategy_id: null,
  run_interval: 60,      // 运行间隔（分钟）
  max_runs: 0,          // 最大运行次数，0表示无限
  auto_stop: false,     // 执行后自动停止
  create_orders: true,  // 是否自动创建订单
  execution_delay: 60,  // 执行延迟（秒）
  per_order_amount: 0   // 每一单的金额（U单位），0表示使用默认金额
})

// ===== 策略管理辅助函数 =====

// 获取交易类型文本
function getTradingTypeText(tradingType) {
  const typeMap = {
    'futures': '合约交易',
    'spot': '现货交易',
    'both': '两者皆可'
  }
  return typeMap[tradingType] || tradingType
}

// 获取保证金模式文本
function getMarginModeText(marginMode) {
  const modeMap = {
    'isolated': '逐仓模式',
    'cross': '全仓模式'
  }
  return modeMap[marginMode] || marginMode
}

// 格式化时间显示
function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 获取均线信号模式文本
function getMASignalModeText(mode) {
  const modeMap = {
    'cross': '交叉信号',
    'trend': '趋势跟随',
    'both': '交叉+趋势'
  }
  return modeMap[mode] || mode
}

// 获取均线交叉信号文本
function getMACrossSignalText(signal) {
  const signalMap = {
    'golden_cross': '金叉买入',
    'dead_cross': '死叉卖出',
    'both': '金叉买入+死叉卖出'
  }
  return signalMap[signal] || signal
}

// 获取均线趋势方向文本
function getMATrendDirectionText(direction) {
  const directionMap = {
    'up': '上涨趋势',
    'down': '下跌趋势',
    'both': '双向趋势'
  }
  return directionMap[direction] || direction
}

// 获取均值回归信号模式文本
function getMRSignalModeText(mode) {
  const modeMap = {
    'oversold': '超卖信号',
    'overbought': '超买信号',
    'both': '双向信号'
  }
  return modeMap[mode] || mode
}

// 获取均值回归子模式文本
function getMeanReversionSubModeText(subMode) {
  const modeMap = {
    'bollinger_rsi': '布林带+RSI',
    'channel_rsi': '价格通道+RSI',
    'bollinger_channel': '布林带+价格通道',
    'all': '全指标组合'
  }
  return modeMap[subMode] || subMode
}

// 加载策略列表（策略管理页面）
async function loadStrategiesForManagement() {
  console.log('开始加载策略列表...')
  console.log('用户认证状态:', isAuthed.value)

  if (!isAuthed.value) {
    console.error('用户未登录，无法加载策略列表')
    strategiesLoading.value = false
    return
  }

  strategiesLoading.value = true
  try {
    console.log('调用API: listTradingStrategies')
    const res = await api.listTradingStrategies()
    console.log('API响应:', res)
    strategies.value = res.data || []
    console.log('策略列表加载完成，共', strategies.value.length, '个策略')
  } catch (e) {
    console.error('加载策略失败:', e)
    // 如果是认证错误，显示提示
    if (e.message && (e.message.includes('token') || e.message.includes('auth'))) {
      console.error('认证失败，请重新登录')
    }
  } finally {
    // 确保loading状态总是被重置
    strategiesLoading.value = false
  }
}

// 删除策略
async function deleteStrategy(id) {
  if (!confirm('确认删除该策略？')) return

  try {
    await api.deleteTradingStrategy(id)
    // 重新加载策略列表
    await loadStrategiesForManagement()
  } catch (e) {
    console.error('删除策略失败:', e)
  }
}

// 启动策略运行
function startStrategy(strategyId) {
  // 初始化表单
  startStrategyForm.strategy_id = strategyId
  startStrategyForm.run_interval = 60
  startStrategyForm.max_runs = 0
  startStrategyForm.auto_stop = false
  startStrategyForm.create_orders = true
  startStrategyForm.per_order_amount = 0

  // 显示弹窗
  showStartStrategyModal.value = true
}

// 监听自动停止选项变化
import { watch } from 'vue'
watch(() => startStrategyForm.auto_stop, (newValue) => {
  if (newValue) {
    // 当选择执行一次后自动停止时，自动设置最大运行次数为1
    startStrategyForm.max_runs = 1
  } else {
    // 当取消选择时，重置为默认值
    startStrategyForm.max_runs = 0
  }
})

// 确认启动策略
async function confirmStartStrategy() {
  startingStrategy.value = startStrategyForm.strategy_id

  try {
    const params = {
      strategy_id: startStrategyForm.strategy_id,
      run_interval: startStrategyForm.run_interval,
      max_runs: startStrategyForm.max_runs,
      auto_stop: startStrategyForm.auto_stop,
      create_orders: startStrategyForm.create_orders,
      execution_delay: startStrategyForm.execution_delay,
      per_order_amount: startStrategyForm.per_order_amount
    }

    const response = await api.startStrategyExecution(params)
    if (response.success) {
      await loadStrategiesForManagement()
      alert('策略已启动运行')
      showStartStrategyModal.value = false
    } else {
      alert('启动失败: ' + (response.message || '未知错误'))
    }
  } catch (error) {
    console.error('启动策略失败:', error)
    alert('启动失败: ' + (error.message || '网络错误'))
  } finally {
    startingStrategy.value = null
  }
}

// 取消启动策略
function cancelStartStrategy() {
  showStartStrategyModal.value = false
}

// 停止策略运行
async function stopStrategy(strategyId) {
  if (!confirm('确定要停止这个策略的自动运行吗？')) {
    return
  }

  stoppingStrategy.value = strategyId
  try {
    const response = await api.stopStrategyExecution(strategyId)
    if (response.success) {
      await loadStrategiesForManagement()
      alert(`策略已停止运行，共停止了${response.stopped}个执行实例`)
    } else {
      alert('停止失败: ' + (response.message || '未知错误'))
    }
  } catch (error) {
    console.error('停止策略失败:', error)
    alert('停止失败: ' + (error.message || '网络错误'))
  } finally {
    stoppingStrategy.value = null
  }
}

// 查看策略运行统计
function viewStrategyStats(strategyId) {
  router.push(`/strategy-stats/${strategyId}`)
}

function backtestStrategy(strategy) {
  // 跳转到回测页面，传递策略信息
  router.push(`/backtest?strategy_id=${strategy.id}&strategy_name=${encodeURIComponent(strategy.name)}`)
}

function editStrategy(strategy) {
  // 跳转到策略编辑页面，传递策略ID
  router.push(`/create-strategy?edit=${strategy.id}`)
}

// 显示策略健康状态
async function showStrategyHealth(strategyId) {
  try {
    const response = await api.getStrategyHealth(strategyId)
    if (response.success) {
      const health = response.data
      let message = `策略状态: ${getHealthStatusText(health.status)}\n`

      if (health.last_execution) {
        const exec = health.last_execution
        message += `\n最后执行:\n`
        message += `状态: ${exec.status}\n`
        message += `开始时间: ${formatDateTime(exec.start_time)}\n`
        if (exec.end_time) {
          message += `结束时间: ${formatDateTime(exec.end_time)}\n`
        }
        message += `订单数: ${exec.total_orders}\n`
        message += `胜率: ${exec.win_rate}%\n`
      }

      if (health.next_run_time) {
        message += `\n下次执行: ${formatDateTime(health.next_run_time)}\n`
      }

      alert(message)
    } else {
      alert('获取策略健康状态失败')
    }
  } catch (error) {
    console.error('获取策略健康状态失败:', error)
    alert('获取策略健康状态失败: ' + (error.message || '网络错误'))
  }
}

// 获取健康状态样式类
function getHealthStatus(strategy) {
  // 这里可以根据实际健康检查结果返回不同的状态
  // 暂时基于运行状态返回
  return strategy.is_running ? 'healthy' : 'stopped'
}

// 获取健康状态文本
function getHealthText(strategy) {
  if (!strategy.is_running) return '已停止'
  // 这里可以根据实际健康检查结果返回不同的文本
  return '健康'
}

// 获取健康状态文本（用于alert）
function getHealthStatusText(status) {
  const statusMap = {
    'waiting': '等待执行',
    'pending_execution': '等待执行',
    'executing': '正在执行',
    'stopped': '已停止',
    'never_executed': '从未执行',
    'unknown': '未知状态'
  }
  return statusMap[status] || status
}

// 格式化日期时间
function formatDateTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 组件挂载时加载数据
onMounted(async () => {
  await loadStrategiesForManagement()
})

// 点击其他地方时关闭下拉菜单（策略管理组件目前没有下拉菜单）
function closeDropdowns() {
  // 策略管理组件目前没有需要关闭的下拉菜单
}

// 暴露一些方法给父组件使用
defineExpose({
  loadStrategiesForManagement,
  closeDropdowns
})
</script>

<style scoped>
/* ===== 策略管理容器样式 ===== */
.strategies-container {
  margin-top: 20px;
}

.strategies-title {
  margin: 0 0 20px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.strategy-title-section {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.strategy-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
}

.strategy-status.running {
  color: #059669;
}

.strategy-status.running .status-indicator {
  width: 8px;
  height: 8px;
  background: #10b981;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.strategy-health {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #059669;
  cursor: pointer;
  margin-top: 4px;
}

.strategy-health:hover {
  color: #047857;
}

.health-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.health-indicator.healthy {
  background: #10b981;
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.4);
}

.health-indicator.warning {
  background: #f59e0b;
  box-shadow: 0 0 6px rgba(245, 158, 11, 0.4);
}

.health-indicator.error {
  background: #ef4444;
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.4);
}

.health-indicator.stopped {
  background: #6b7280;
}

.btn-run {
  background: #10b981;
  color: white;
  border: 1px solid #10b981;
}

.btn-run:hover {
  background: #059669;
  border-color: #059669;
}

.btn-stop {
  background: #ef4444;
  color: white;
  border: 1px solid #ef4444;
}

.btn-stop:hover {
  background: #dc2626;
  border-color: #dc2626;
}

.btn-stats {
  background: #f3e8ff;
  color: #6b21a8;
  border: 1px solid #c4b5fd;
  min-width: 60px;
}

.btn-stats:hover {
  background: #e9d5ff;
  border-color: #a78bfa;
  color: #581c87;
}

.btn-backtest {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fbbf24;
  min-width: 60px;
}

.btn-backtest:hover {
  background: #fde68a;
  border-color: #f59e0b;
  color: #78350f;
}

/* ===== 策略列表样式优化 ===== */
.strategies-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.strategy-item {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  transition: all 0.15s;
}

.strategy-item:hover {
  border-color: #d1d5db;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.strategy-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f3f4f6;
}

.strategy-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
  display: flex;
  align-items: center;
  gap: 8px;
}

.strategy-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.strategy-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.condition-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.condition-tag {
  background: #f3f4f6;
  color: #374151;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  border: 1px solid #e5e7eb;
}

.condition-tag.symbol-filter {
  background: #fef3c7;
  color: #92400e;
  border-color: #fbbf24;
}

.strategy-meta {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.strategy-meta small {
  color: #6b7280;
  font-size: 12px;
}

/* ===== 策略操作按钮样式 ===== */
.btn-edit {
  height: 32px;
  padding: 0 16px;
  border: 1px solid #3b82f6;
  background: #eff6ff;
  color: #1e40af;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 60px;
}

.btn-edit:hover {
  background: #dbeafe;
  border-color: #2563eb;
  color: #1d4ed8;
}

.btn-delete {
  height: 32px;
  padding: 0 16px;
  border: 1px solid #ef4444;
  background: #fef2f2;
  color: #dc2626;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 60px;
}

.btn-delete:hover {
  background: #fee2e2;
  border-color: #dc2626;
  color: #b91c1c;
}

/* ===== 模态框样式 ===== */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.strategy-start-modal {
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  max-width: 500px;
  width: 90%;
  max-height: 80vh;
  animation: modalSlideIn 0.2s ease-out;
  display: flex;
  flex-direction: column;
}

.strategy-start-modal .modal-body {
  display: flex;
  flex-direction: column;
  min-height: 0; /* 允许内容区域缩小 */
}

.strategy-start-modal .form-actions {
  margin-top: auto; /* 将按钮推到底部 */
}

@keyframes modalSlideIn {
  from {
    opacity: 0;
    transform: translateY(-20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px 32px 20px;
  border-bottom: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.modal-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #111827;
}

.modal-close {
  background: none;
  border: none;
  font-size: 28px;
  color: #6b7280;
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.modal-close:hover {
  background: #f3f4f6;
  color: #374151;
}

.modal-body {
  padding: 32px;
  padding-bottom: 0; /* 底部padding由form-actions提供 */
  flex: 1;
  overflow-y: auto;
  min-height: 0; /* 允许flex子项缩小 */
}

/* ===== 启动策略表单样式 ===== */
.start-strategy-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.start-strategy-form .form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.start-strategy-form .form-label {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  display: flex;
  align-items: center;
  gap: 6px;
}

.start-strategy-form .required-mark {
  color: #dc2626;
  font-weight: 700;
  font-size: 14px;
}

.start-strategy-form .form-input,
.start-strategy-form .form-select {
  height: 40px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  color: #111827;
  padding: 0 12px;
  font-size: 14px;
  transition: border-color 0.15s;
}

.start-strategy-form .form-input:focus,
.start-strategy-form .form-select:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.start-strategy-form .form-hint {
  font-size: 12px;
  color: #6b7280;
  font-weight: 400;
}

.start-strategy-form .checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #374151;
  user-select: none;
}

.start-strategy-form .checkbox-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: #2563eb;
  border-radius: 4px;
}

.start-strategy-form .form-input:disabled,
.start-strategy-form .form-select:disabled {
  background-color: #f9fafb;
  color: #9ca3af;
  cursor: not-allowed;
  opacity: 0.6;
}

.start-strategy-form .text-muted {
  color: #9ca3af;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 32px;
  border-top: 1px solid #e5e7eb;
  flex-shrink: 0; /* 防止按钮区域被压缩 */
  background: white; /* 确保按钮背景是白色 */
}

.btn {
  height: 40px;
  padding: 0 20px;
  border-radius: 8px;
  border: none;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.btn-secondary {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.btn-secondary:hover {
  background: #e5e7eb;
  border-color: #9ca3af;
}

.btn-primary {
  background: #2563eb;
  color: white;
}

.btn-primary:hover {
  background: #1d4ed8;
  transform: translateY(-1px);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none !important;
}

/* ===== 移动端样式 ===== */
@media (max-width: 768px) {
  .strategies-container {
    margin-top: 16px;
  }

  .strategies-title {
    font-size: 16px;
    margin-bottom: 16px;
  }

  .strategies-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .strategy-item {
    padding: 16px;
  }

  .strategy-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .strategy-header h4 {
    font-size: 16px;
  }

  .strategy-actions {
    align-self: stretch;
    justify-content: flex-end;
  }

  .condition-summary {
    gap: 4px;
  }

  .condition-tag {
    font-size: 11px;
    padding: 3px 6px;
  }
}
</style>