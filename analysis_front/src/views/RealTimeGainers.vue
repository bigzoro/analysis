<!-- src/views/RealTimeGainers.vue -->
<template>
  <div class="page">
    <header class="page-header">
      <div class="header-top">
        <div class="page-title-section">
          <h1 class="page-title">实时涨幅榜</h1>
          <div class="title-description">
            <span class="top15-notice">🔥 实时监控热门币种涨幅榜，智能更新</span>
          </div>
        </div>
        <div class="selectors">
          <div class="type-selector">
            <button
              :class="['type-btn', { active: safeSelectedKind === 'spot' }]"
              @click="handleKindChange('spot')"
            >
              现货
            </button>
            <button
              :class="['type-btn', { active: safeSelectedKind === 'futures' }]"
              @click="handleKindChange('futures')"
            >
              合约
            </button>
          </div>
          <div class="category-selector">
            <select v-model="selectedCategory" @change="handleCategoryChange" class="category-select">
              <option value="trading">正常交易</option>
              <option value="break">暂停交易</option>
              <option value="major">主流币种</option>
              <option value="stable">稳定币对</option>
              <option value="defi">DeFi代币</option>
              <option value="layer1">Layer1公链</option>
              <option value="meme">Meme币</option>
              <option value="spot_only">纯现货</option>
              <option value="margin">杠杆交易</option>
              <option value="leveraged">合约交易</option>
              <option value="all">全部币种</option>
            </select>
          </div>
        </div>
      </div>
      <div class="header-row">
        <div class="controls">
          <div class="connection-status" :class="{ connected: connectionState.status === 'connected', reconnecting: connectionState.status === 'reconnecting' }">
            <span class="status-dot"></span>
            {{ connectionStatusText }}
          </div>
          <div class="last-update" :class="{ stale: isDataStale }">
            最后更新: {{ dataState.lastUpdate }}
            <span v-if="isDataStale" class="stale-indicator">⚠️</span>
          </div>
        </div>
      </div>
    </header>

    <!-- 变化提示 -->
    <div v-if="notificationState.show" class="change-notification">
      <span class="change-icon">📈</span>
      <span class="change-message">{{ notificationState.message }}</span>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorState.show" class="error-banner">
      <span class="error-icon">⚠️</span>
      <span class="error-message">{{ errorState.message }}</span>
      <div class="error-actions">
        <button class="error-retry" @click="forceResetErrors" v-if="errorStats.consecutiveFailures > 2">
          🔄 重置
        </button>
        <button class="error-close" @click="updateErrorState(false)">✕</button>
      </div>
    </div>

    <section v-if="isLoading" class="loading">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">正在获取实时数据...</div>
        <div class="loading-hint">首次加载可能需要10-15秒</div>
      </div>
    </section>

    <section v-else>
      <div class="realtime-table">
        <table class="tbl">
          <thead>
          <tr>
            <th class="col-rank">#</th>
            <th class="col-symbol">币种</th>
            <th class="col-num">最新价</th>
            <th class="col-num">24h涨跌幅</th>
            <th class="col-num">24h成交量</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="(item, index) in filteredGainers" :key="item.symbol"
              :class="{ 'highlight-row': isHighlighted(item.symbol) }">
            <td class="col-rank">
              <span :class="{ 'top15-badge': item.is_top15 }">{{ item.rank }}</span>
              <span v-if="item.is_top15" class="top15-indicator" title="前15名重点监控">🔥</span>
            </td>
            <td class="col-symbol">
                <a
                  v-if="isMajorPair(item.symbol)"
                  :href="getBinanceUrl(item.symbol, safeSelectedKind)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="symbol-link"
                  :title="'点击跳转到币安 ' + formatSymbol(item.symbol) + ' 交易页面'"
                >
                  {{ formatSymbol(item.symbol) }}
                </a>
                <span
                  v-else
                  class="symbol-text"
                  :title="'暂不支持 ' + formatSymbol(item.symbol) + ' 的跳转'"
                >
                  {{ formatSymbol(item.symbol) }}
                </span>
            </td>
            <td class="col-num price-cell">
              <span class="price-value">{{ formatPrice(item.current_price) }}</span>
              <span v-if="item.price_change" class="price-trend"
                    :class="item.price_change > 0 ? 'up' : 'down'">
                {{ item.price_change > 0 ? '↗' : '↘' }}
              </span>
            </td>
            <td
                class="col-num change-cell"
                :class="getChangeCellClass(item.price_change_24h)"
                :title="formatPctFull(item.price_change_24h)"
            >
              <span class="change-value">{{ formatPct(item.price_change_24h) }}</span>
              <span class="change-bar" :style="getChangeBarStyle(item.price_change_24h)"></span>
            </td>
            <td class="col-num volume-cell">
              <span class="volume-value">{{ formatVolume(item.volume_24h) }}</span>
              <span v-if="item.volume_change_24h" class="volume-change"
                    :class="item.volume_change_24h >= 0 ? 'up' : 'down'">
                ({{ formatPct(item.volume_change_24h) }})
              </span>
            </td>
          </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, onActivated, onDeactivated, watch, nextTick } from 'vue'
import { api } from '../api/api.js'
import { handleError } from '../utils/errorHandler.js'

// ===== 状态定义 =====
// 连接状态枚举
const CONNECTION_STATES = {
  DISCONNECTED: 'disconnected',
  CONNECTING: 'connecting',
  CONNECTED: 'connected',
  RECONNECTING: 'reconnecting',
  FAILED: 'failed'
}

// 加载状态枚举
const LOADING_STATES = {
  IDLE: 'idle',
  INITIAL: 'initial',
  REFRESHING: 'refreshing'
}

// 错误处理枚举
const ERROR_TYPES = {
  NETWORK: 'network',
  WEBSOCKET: 'websocket',
  API: 'api',
  TIMEOUT: 'timeout',
  UNKNOWN: 'unknown'
}

// 熔断器状态枚举
const CIRCUIT_STATES = {
  CLOSED: 'closed',     // 正常状态
  OPEN: 'open',         // 熔断开启，拒绝请求
  HALF_OPEN: 'half_open' // 半开状态，允许试探性请求
}

// 核心状态
const selectedKind = ref('spot') // 'spot' 或 'futures'

console.log('[初始化] selectedKind初始值:', selectedKind.value)

// 确保selectedKind始终有值
const safeSelectedKind = computed(() => {
  const value = selectedKind.value
  // 如果值无效，返回默认值'spot'，但不修改原始值
  return (value === 'spot' || value === 'futures') ? value : 'spot'
})

// 处理市场类型变化
function handleKindChange(kind) {
  console.log('[UI] handleKindChange被调用，参数:', kind, '当前selectedKind:', selectedKind.value)

  // 直接赋值
  selectedKind.value = kind

  // 使用nextTick确保DOM更新
  nextTick(() => {
    console.log('[UI] 更新后selectedKind:', selectedKind.value, 'safeSelectedKind:', safeSelectedKind)
  })
}

// 监听selectedKind变化，用于调试
watch(selectedKind, (newValue, oldValue) => {
  console.log('[监听] selectedKind变化:', oldValue, '->', newValue, '类型:', typeof newValue)
  if (newValue === undefined) {
    console.error('[监听] selectedKind变为undefined，这是不正确的！')
  }
})

// 移除定时器检查，避免干扰响应式更新
// Vue的计算属性应该自动处理undefined情况

const selectedCategory = ref('trading') // 选中的分类

// 连接状态（合并多个相关状态）
const connectionState = ref({
  status: CONNECTION_STATES.DISCONNECTED,
  attempts: 0,
  websocket: null,
  reconnectTimer: null,
  lastMessageTime: 0,  // 最后收到消息的时间
  messageTimeoutTimer: null  // 消息超时检测定时器
})

// 加载状态（合并loading相关状态）
const loadingState = ref({
  status: LOADING_STATES.INITIAL
})

// 数据状态
const dataState = ref({
  gainers: [],
  lastUpdate: '--',
  lastUpdateTimestamp: null
})

// 错误状态（合并错误相关状态）
const errorState = ref({
  show: false,
  message: '',
  networkStatus: navigator.onLine,
  lastNetworkCheck: Date.now()
})

// 筛选状态（合并筛选相关状态）
const filterState = ref({
  showPositiveOnly: true,  // 默认只显示上涨
  showLargeCapOnly: true   // 默认大市值币种
})

// 变化提示状态
const notificationState = ref({
  show: false,
  message: '',
  timeout: null
})

// ===== 错误处理增强 =====
// 指数退避重试配置
const retryConfig = ref({
  maxRetries: 3,
  baseDelay: 1000,    // 基础延迟1秒
  maxDelay: 30000,    // 最大延迟30秒
  backoffFactor: 2    // 退避因子
})

// 熔断器配置
const circuitBreaker = ref({
  state: CIRCUIT_STATES.CLOSED,
  failureCount: 0,
  successCount: 0,
  nextAttemptTime: 0,
  failureThreshold: 5,     // 失败阈值
  recoveryTimeout: 60000,  // 恢复超时60秒
  successThreshold: 3      // 成功阈值（半开状态下）
})

// 错误统计
const errorStats = ref({
  consecutiveFailures: 0,
  lastErrorTime: 0,
  totalErrors: 0,
  errorTypes: new Map()
})

// 临时状态（不需要持久化）
const highlightedSymbols = ref(new Set())


// ===== 计算属性 =====
// 连接状态文本
const connectionStatusText = computed(() => {
  const state = connectionState.value.status
  switch (state) {
    case CONNECTION_STATES.CONNECTED: return '实时连接'
    case CONNECTION_STATES.CONNECTING: return '连接中...'
    case CONNECTION_STATES.RECONNECTING: return '重连中...'
    case CONNECTION_STATES.FAILED: return '连接失败'
    default: return '未连接'
  }
})

// 数据是否过期
const isDataStale = computed(() => {
  const timestamp = dataState.value.lastUpdateTimestamp
  if (!timestamp) return false
  const now = Date.now()
  const diff = now - timestamp
  return diff > 5 * 60 * 1000 // 5分钟
})

// 是否正在加载
const isLoading = computed(() => {
  return loadingState.value.status !== LOADING_STATES.IDLE
})

// 是否为初始加载
const isInitialLoading = computed(() => {
  return loadingState.value.status === LOADING_STATES.INITIAL
})

// 筛选和排序后的数据（涨幅榜需要1-10的正确序号）
const filteredGainers = computed(() => {
  let filtered = [...dataState.value.gainers]

  // 应用筛选条件
  if (filterState.value.showPositiveOnly) {
    filtered = filtered.filter(item => item.price_change_24h >= 0)
  }

  if (filterState.value.showLargeCapOnly) {
    filtered = filtered.filter(item => {
      const price = parseFloat(item.current_price) || 0
      const volume = parseFloat(item.volume_24h) || 0
      return price * volume > 1000000 // 简单的市值筛选
    })
  }

  // 固定使用涨幅排序（降序）
  filtered.sort((a, b) => {
    const aChange = a.price_change_24h || 0
    const bChange = b.price_change_24h || 0
    return bChange - aChange // 降序：涨幅高的在前
  })

  // 总是重新分配1-10的正确排名
  filtered.forEach((item, index) => {
    item.rank = index + 1
  })

  return filtered
})

// 主要交易对列表（原生币）
const majorPairs = [
  'BTCUSDT', 'ETHUSDT', 'BNBUSDT', 'ADAUSDT', 'XRPUSDT', 'SOLUSDT', 'DOTUSDT',
  'DOGEUSDT', 'AVAXUSDT', 'LTCUSDT', 'TRXUSDT', 'ETCUSDT', 'BCHUSDT',
  'LINKUSDT', 'MATICUSDT', 'ICPUSDT', 'FILUSDT', 'XLMUSDT', 'VETUSDT'
]

// 检查是否为主要交易对
function isMajorPair(symbol) {
  return majorPairs.includes(symbol)
}


// ===== 辅助函数 =====
function highlightSymbol(symbol) {
  highlightedSymbols.value.add(symbol)
  setTimeout(() => {
    highlightedSymbols.value.delete(symbol)
  }, 3000)
}

function isHighlighted(symbol) {
  return highlightedSymbols.value.has(symbol)
}

// 状态更新辅助函数
function updateConnectionState(status, attempts = null) {
  const oldStatus = connectionState.value.status
  connectionState.value.status = status
  if (attempts !== null) {
    connectionState.value.attempts = attempts
  }
  console.log('[Connection] 状态变化:', oldStatus, '->', status, 'attempts:', connectionState.value.attempts)
}

function updateLoadingState(status) {
  const oldStatus = loadingState.value.status
  loadingState.value.status = status
  console.log('[Loading] 状态变化:', oldStatus, '->', status, '调用栈:', new Error().stack.split('\n')[2])
}

function updateDataState(gainers, timestamp = null) {
  dataState.value.gainers = gainers || []
  dataState.value.lastUpdate = timestamp ? new Date(timestamp * 1000).toLocaleTimeString() : new Date().toLocaleTimeString()
  dataState.value.lastUpdateTimestamp = timestamp || Date.now()
}

function updateErrorState(show, message = '') {
  errorState.value.show = show
  errorState.value.message = message
}

function updateNotificationState(show, message = '') {
  notificationState.value.show = show
  notificationState.value.message = message

  // 清除之前的定时器
  if (notificationState.value.timeout) {
    clearTimeout(notificationState.value.timeout)
  }

  // 如果显示通知，设置自动隐藏
  if (show && message) {
    notificationState.value.timeout = setTimeout(() => {
      updateNotificationState(false, '')
    }, 4000)
  }
}

// ===== 错误处理增强函数 =====

// 指数退避计算
function calculateBackoffDelay(retryCount) {
  const config = retryConfig.value
  const delay = config.baseDelay * Math.pow(config.backoffFactor, retryCount)
  return Math.min(delay, config.maxDelay)
}

// 熔断器检查
function checkCircuitBreaker() {
  const breaker = circuitBreaker.value
  const now = Date.now()

  switch (breaker.state) {
    case CIRCUIT_STATES.OPEN:
      if (now >= breaker.nextAttemptTime) {
        // 进入半开状态
        breaker.state = CIRCUIT_STATES.HALF_OPEN
        breaker.successCount = 0
        console.log('[CircuitBreaker] 进入半开状态，允许试探性请求')
        return true
      }
      console.log('[CircuitBreaker] 熔断器开启，拒绝请求')
      return false

    case CIRCUIT_STATES.HALF_OPEN:
      // 半开状态允许请求，但会严格检查结果
      return true

    case CIRCUIT_STATES.CLOSED:
    default:
      return true
  }
}

// 熔断器状态更新
function updateCircuitBreaker(success) {
  const breaker = circuitBreaker.value

  if (success) {
    breaker.successCount++

    if (breaker.state === CIRCUIT_STATES.HALF_OPEN) {
      if (breaker.successCount >= breaker.successThreshold) {
        // 恢复正常
        breaker.state = CIRCUIT_STATES.CLOSED
        breaker.failureCount = 0
        console.log('[CircuitBreaker] 熔断器关闭，恢复正常')
      }
    } else if (breaker.state === CIRCUIT_STATES.CLOSED) {
      // 重置失败计数
      breaker.failureCount = 0
    }
  } else {
    breaker.failureCount++

    if (breaker.failureCount >= breaker.failureThreshold) {
      // 开启熔断
      breaker.state = CIRCUIT_STATES.OPEN
      breaker.nextAttemptTime = Date.now() + breaker.recoveryTimeout
      console.log(`[CircuitBreaker] 熔断器开启，${breaker.recoveryTimeout}ms后重试`)
    }
  }
}

// 错误分类
function categorizeError(error) {
  if (!navigator.onLine) return ERROR_TYPES.NETWORK
  if (error?.code === 1006 || error?.type === 'close') return ERROR_TYPES.WEBSOCKET
  if (error?.status >= 400) return ERROR_TYPES.API
  if (error?.name === 'TimeoutError') return ERROR_TYPES.TIMEOUT
  return ERROR_TYPES.UNKNOWN
}

// 错误统计更新
function updateErrorStats(error, errorType) {
  const stats = errorStats.value
  stats.totalErrors++
  stats.lastErrorTime = Date.now()
  stats.consecutiveFailures++

  // 更新错误类型统计
  const currentCount = stats.errorTypes.get(errorType) || 0
  stats.errorTypes.set(errorType, currentCount + 1)

  console.log(`[ErrorStats] 错误统计更新 - 类型:${errorType}, 连续失败:${stats.consecutiveFailures}, 总错误:${stats.totalErrors}`)
}

// 智能重试决策
function shouldRetry(error, retryCount) {
  const errorType = categorizeError(error)

  // 某些错误类型不应该重试
  const nonRetryableErrors = [ERROR_TYPES.API] // API错误通常不应该重试

  if (nonRetryableErrors.includes(errorType)) {
    return false
  }

  // 检查熔断器
  if (!checkCircuitBreaker()) {
    return false
  }

  // 检查重试次数
  return retryCount < retryConfig.value.maxRetries
}

// 增强的异步操作执行器（带重试和熔断）
async function executeWithRetry(operation, operationName = 'operation') {
  let retryCount = 0

  while (true) {
    try {
      // 执行操作
      const result = await operation()

      // 成功：重置错误统计，更新熔断器
      errorStats.value.consecutiveFailures = 0
      updateCircuitBreaker(true)

      return result

    } catch (error) {
      const errorType = categorizeError(error)

      // 更新错误统计
      updateErrorStats(error, errorType)

      // 记录错误
      console.error(`[${operationName}] 执行失败 (尝试 ${retryCount + 1}/${retryConfig.value.maxRetries + 1}):`, error)

      // 检查是否应该重试
      if (!shouldRetry(error, retryCount)) {
        // 更新熔断器状态
        updateCircuitBreaker(false)

        // 显示用户友好的错误信息
        const userMessage = getUserFriendlyErrorMessage(error, errorType, operationName)
        updateErrorState(true, userMessage)

        throw error
      }

      // 计算重试延迟
      const delay = calculateBackoffDelay(retryCount)
      console.log(`[${operationName}] ${delay}ms后重试 (${retryCount + 1}/${retryConfig.value.maxRetries})`)

      // 等待重试
      await new Promise(resolve => setTimeout(resolve, delay))
      retryCount++
    }
  }
}

// 获取用户友好的错误信息
function getUserFriendlyErrorMessage(error, errorType, operationName) {
  const circuitState = circuitBreaker.value.state

  // 熔断器开启时的特殊提示
  if (circuitState === CIRCUIT_STATES.OPEN) {
    return '系统暂时不可用，请稍后再试'
  }

  switch (errorType) {
    case ERROR_TYPES.NETWORK:
      return '网络连接失败，请检查网络设置'
    case ERROR_TYPES.WEBSOCKET:
      return '实时连接断开，正在尝试重连...'
    case ERROR_TYPES.API:
      return '服务器暂时不可用，请稍后再试'
    case ERROR_TYPES.TIMEOUT:
      return '请求超时，请检查网络连接'
    default:
      return `操作失败：${operationName}，请稍后重试`
  }
}

// 检查变化详情并返回变化信息
function checkForSignificantChanges(oldData, newData) {
  if (!oldData || oldData.length === 0) return { hasChanges: true, changes: [] } // 首次数据算作有变化

  const changes = []

  // 检查排名变化
  const rankChanges = oldData.filter((oldItem, index) => {
    const newItem = newData[index]
    return newItem && oldItem.symbol !== newItem.symbol
  }).length

  if (rankChanges > 0) {
    changes.push({ type: 'rank', message: `${rankChanges}个币种排名发生变化` })
  }

  // 检查价格变化（超过0.1%的算显著变化）
  const priceChanges = []
  oldData.forEach(oldItem => {
    const newItem = newData.find(item => item.symbol === oldItem.symbol)
    if (!newItem) return

    const oldPrice = parseFloat(oldItem.current_price) || 0
    const newPrice = parseFloat(newItem.current_price) || 0

    if (oldPrice === 0) return
    const changePercent = (newPrice - oldPrice) / oldPrice * 100

    if (Math.abs(changePercent) >= 0.1) { // 0.1%以上的变化
      const direction = changePercent > 0 ? '上涨' : '下跌'
      priceChanges.push({
        symbol: oldItem.symbol,
        changePercent: changePercent,
        direction: direction,
        message: `${formatSymbol(oldItem.symbol)} ${direction} ${Math.abs(changePercent).toFixed(2)}%`
      })
    }
  })

  // 只显示前3个价格变化，避免消息过长
  if (priceChanges.length > 0) {
    const topChanges = priceChanges.slice(0, 3)
    if (priceChanges.length > 3) {
      changes.push({
        type: 'price',
        message: `${topChanges.map(c => c.message).join('、')} 等${priceChanges.length}个币种价格变化`
      })
    } else {
      changes.push({
        type: 'price',
        message: topChanges.map(c => c.message).join('、')
      })
    }
  }

  return {
    hasChanges: rankChanges > 0 || priceChanges.length > 0,
    changes: changes
  }
}

// 显示变化通知
// 显示变化通知（已集成到updateNotificationState中，此函数保留兼容性）
function showChangeNotificationForChanges(changes) {
  // 根据变化类型生成不同的消息
  let message = '📈 涨幅榜已更新'

  if (changes && changes.length > 0) {
    // 优先显示价格变化，如果没有则显示排名变化
    const priceChange = changes.find(c => c.type === 'price')
    if (priceChange) {
      message = `📈 ${priceChange.message}`
    } else {
      const rankChange = changes.find(c => c.type === 'rank')
      if (rankChange) {
        message = `🔄 ${rankChange.message}`
      }
    }
  }

  updateNotificationState(true, message)
}


// 更新筛选
function updateFilters() {
  // 筛选逻辑在computed中处理
}


// 获取涨跌幅单元格的CSS类
function getChangeCellClass(changePercent) {
  const numValue = parseFloat(changePercent) || 0
  const isPositive = numValue >= 0
  const className = isPositive ? 'up' : 'down'

  // 调试信息：只在开发环境下显示
  if (import.meta.env.DEV && Math.random() < 0.1) { // 10%的概率显示调试信息
    console.log(`[ChangeCell] 涨幅: ${changePercent} -> ${className}`)
  }

  return `change-cell ${className}`
}

// 获取涨跌幅条样式
function getChangeBarStyle(changePercent) {
  const percent = Math.abs(changePercent || 0)
  const maxPercent = 20 // 最大显示20%
  const width = Math.min(percent / maxPercent * 100, 100)
  const color = changePercent >= 0 ? '#22c55e' : '#ef4444'
  return {
    width: width + '%',
    backgroundColor: color,
    opacity: 0.3
  }
}


// 格式化时间戳
function formatTimestamp(timestamp) {
  if (!timestamp) return '--'
  const date = new Date(timestamp * 1000)
  return date.toLocaleTimeString()
}

// 处理刷新
async function handleRefresh() {
  if (loadingState.value.status !== LOADING_STATES.IDLE) return

  updateLoadingState(LOADING_STATES.REFRESHING)
  updateErrorState(false)

  // 实现指数退避重试
  let retryCount = 0
  const maxRetries = 3

  while (retryCount <= maxRetries) {
    try {
      await loadInitialData()
      break // 成功则跳出重试循环
    } catch (err) {
      retryCount++
      const isLastAttempt = retryCount > maxRetries

      if (isLastAttempt) {
        updateErrorState(true, err.message || '刷新数据失败，已达到最大重试次数')
        break
      }

      // 指数退避：1秒、2秒、4秒
      const delay = Math.pow(2, retryCount - 1) * 1000
      console.log(`[Refresh] 第${retryCount}次重试，等待${delay}ms...`)
      await new Promise(resolve => setTimeout(resolve, delay))
    }
  }

  updateLoadingState(LOADING_STATES.IDLE)
}

// 生成币安页面URL
function getBinanceUrl (symbol, kind) {
  if (!symbol) return '#'

  // 原生币：直接跳转到交易页面
  let tradeSymbol = symbol

  // 处理常见的交易对格式，按优先级从长到短匹配
  const quoteAssets = ['USDT', 'BUSD', 'USDC', 'BTC', 'ETH', 'BNB', 'ADA', 'SOL', 'DOT']
  let matched = false

  for (const quote of quoteAssets) {
    if (tradeSymbol.endsWith(quote)) {
      tradeSymbol = tradeSymbol.replace(quote, '_' + quote)
      matched = true
      break
    }
  }

  // 如果没有匹配到任何后缀，尝试添加 _USDT
  if (!matched) {
    tradeSymbol = tradeSymbol + '_USDT'
  }

  return `https://www.binance.com/zh-CN/trade/${tradeSymbol}?type=spot`
}

function formatPct (n) {
  const v = Number(n)
  if (!isFinite(v)) return n
  return (v >= 0 ? '+' : '') + v.toFixed(2) + '%'
}

function formatPctFull (n) {
  const v = Number(n)
  if (!isFinite(v)) return n
  return (v >= 0 ? '+' : '') + v.toFixed(6) + '%'
}

function formatPrice (s) {
  const n = Number(s)
  if (!isFinite(n)) return s
  if (n === 0) return '0'
  // >=1 的保留最多 4 位小数；<1 的保留 6 位有效数字
  if (n >= 1) {
    return n
        .toLocaleString(undefined, { maximumFractionDigits: 4, useGrouping: false })
        .replace(/(\.\d*?)0+$/, '$1')
        .replace(/\.$/, '')
  } else {
    return Number(n.toPrecision(6)).toString()
  }
}

function formatSymbol (symbol) {
  if (!symbol) return symbol

  // 对于合约交易对，去掉_PERP后缀
  if (symbol.endsWith('_PERP')) {
    return symbol.replace('_PERP', '')
  }

  // 对于现货交易对，去掉常见的后缀
  const quoteCurrencies = ['USDT', 'USDC', 'BUSD', 'BTC', 'ETH', 'BNB']
  for (const quote of quoteCurrencies) {
    if (symbol.endsWith(quote)) {
      return symbol.replace(quote, '')
    }
  }

  return symbol
}

function formatVolume (volume) {
  const n = Number(volume)
  if (!isFinite(n) || n <= 0) return '--'

  const units = [
    { value: 1e12, unit: 'T' },
    { value: 1e9, unit: 'B' },
    { value: 1e6, unit: 'M' },
    { value: 1e3, unit: 'K' }
  ]

  for (const { value, unit } of units) {
    if (n >= value) {
      return '$' + (n / value).toFixed(2) + unit
    }
  }

  return '$' + n.toFixed(2)
}


// ===== WebSocket连接管理 =====
function connectWebSocket() {
  // 清理之前的连接，避免多个连接同时存在
  disconnectWebSocket()

  // 检查熔断器
  if (!checkCircuitBreaker()) {
    console.log('[WebSocket] 熔断器开启，跳过连接尝试')
    return
  }

  // 智能重试控制
  const maxAttempts = Math.max(5, retryConfig.value.maxRetries + 2)
  if (connectionState.value.attempts > maxAttempts) {
    console.log('[WebSocket] 超过智能重试次数，停止重连')
    updateConnectionState(CONNECTION_STATES.FAILED)
    updateCircuitBreaker(false)
    return
  }

  // WebSocket需要直接连接到后端API服务器
  // 在开发环境中，后端通常运行在8010端口
  const isDev = import.meta.env.DEV
  const API_BASE = import.meta.env.VITE_API_BASE || 'http://127.0.0.1:8010'
  let wsUrl

  if (isDev) {
    // 开发环境：从API_BASE解析后端地址
    if (API_BASE.startsWith('http')) {
      const apiUrl = new URL(API_BASE)
      wsUrl = `${apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'}//${apiUrl.host}/ws/realtime-gainers`
    } else {
      // 如果API_BASE是相对路径，默认使用8010端口
      wsUrl = 'ws://127.0.0.1:8010/ws/realtime-gainers'
    }
  } else {
    // 生产环境：使用相对URL（通过Nginx反代）
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    wsUrl = `${protocol}//${window.location.host}/ws/realtime-gainers`
  }

  console.log('[WebSocket] 环境:', isDev ? '开发' : '生产')
  console.log('[WebSocket] API_BASE:', API_BASE)
  console.log('[WebSocket] 正在连接到:', wsUrl, '尝试次数:', connectionState.value.attempts + 1)

  updateConnectionState(CONNECTION_STATES.CONNECTING, connectionState.value.attempts + 1)

  const ws = new WebSocket(wsUrl)
  connectionState.value.websocket = ws

  // 设置连接超时
  const connectionTimeout = setTimeout(() => {
    if (ws && ws.readyState === WebSocket.CONNECTING) {
      console.log('[WebSocket] 连接超时，关闭连接')
      ws.close()
    }
  }, 10000)

  ws.onopen = () => {
    console.log('[WebSocket] 🔗 涨幅榜连接已建立，连接市场:', connectingKind, '当前选择:', safeSelectedKind)
    clearTimeout(connectionTimeout)
    updateConnectionState(CONNECTION_STATES.CONNECTED, 0) // 重置尝试次数
    updateErrorState(false) // 清除错误状态

    // 启动消息超时检测
    updateLastMessageTime()
    startMessageTimeoutDetection()

    console.log('[WebSocket] 🔄 连接状态更新为已连接，等待订阅确认...')

    // 发送订阅消息 - 使用connectingKind确保使用正确的市场类型
    const subscription = {
      action: 'subscribe',
      kind: connectingKind,
      category: selectedCategory.value,
      limit: 15,
      interval: 20
    }

    console.log('[WebSocket] 📤 发送订阅消息:', subscription, '(connectingKind:', connectingKind, 'selectedKind:', safeSelectedKind + ')')
    try {
      ws.send(JSON.stringify(subscription))
      console.log('[WebSocket] ✅ 订阅消息发送成功')
    } catch (error) {
      console.error('[WebSocket] ❌ 发送订阅消息失败:', error)
    }
  }

  ws.onmessage = (event) => {
    // 更新最后消息时间
    updateLastMessageTime()

    console.log('[WebSocket] 📨 收到消息，当前连接状态:', connectionState.value.status, '数据长度:', event.data.length)
    try {
      const data = JSON.parse(event.data)
      console.log('[WebSocket] 📨 解析后消息类型:', data.type, '完整数据:', data)

      if (data.type === 'subscription_confirmed') {
        console.log('[WebSocket] ✅ 涨幅榜订阅确认:', data.message, '连接市场:', connectingKind, '当前市场:', safeSelectedKind)

        // 验证消息的市场类型是否与当前连接匹配
        if (data.kind && data.kind !== connectingKind) {
          console.warn('[WebSocket] ⚠️ 收到不匹配的订阅确认，期望:', connectingKind, '收到:', data.kind, '忽略此消息')
          return
        }

        updateConnectionState(CONNECTION_STATES.CONNECTED)

        // 重置最后消息时间，避免立即触发超时检测
        updateLastMessageTime()

        // 验证连接状态设置是否成功
        nextTick(() => {
          if (connectionState.value.status !== CONNECTION_STATES.CONNECTED) {
            console.error('[WebSocket] ⚠️ 连接状态设置失败，强制重新设置')
            updateConnectionState(CONNECTION_STATES.CONNECTED)
          }
        })

        // 不需要调用HTTP API，等待WebSocket推送第一批数据
      } else if (data.type === 'heartbeat') {
        console.log('[WebSocket] 💓 收到心跳消息:', data.message, '时间戳:', data.timestamp)
        // 心跳消息也需要更新最后消息时间，防止超时检测
        updateLastMessageTime()
      } else if (data.type === 'gainers_update') {
        console.log('[WebSocket] 📊 收到涨幅榜数据，条数:', data.gainers?.length || 0, '推送市场:', data.kind, '连接市场:', connectingKind, '当前市场:', safeSelectedKind)

        // 验证消息的市场类型是否与当前连接匹配
        if (data.kind && data.kind !== connectingKind) {
          console.warn('[WebSocket] ⚠️ 收到不匹配的市场数据，期望:', connectingKind, '收到:', data.kind, '忽略此消息')
          return
        }
        console.log('[WebSocket] 📊 收到消息时loading状态:', loadingState.value.status, 'isLoading:', isLoading.value)

        // 实时更新数据（局部更新，不是页面刷新）
        const newData = data.gainers || []
        if (newData.length > 0) {
          console.log('[WebSocket] 📊 更新数据:', newData.slice(0, 3).map(g => `${g.symbol}: ${g.price_change_24h}%`))

          // 检查变化详情
          const changeResult = checkForSignificantChanges(dataState.value.gainers, newData)

          // 使用后端发送的时间戳（秒级），转换为毫秒
          const serverTimestamp = data.timestamp ? data.timestamp * 1000 : Date.now()
          updateDataState(newData, serverTimestamp)

          // 如果有变化，显示友好的通知
          if (changeResult.hasChanges) {
            updateNotificationState(true, changeResult.changes.map(c => c.message).join('、'))
          }

          console.log('[WebSocket] ✅ 数据局部更新完成')
        } else {
          console.log('[WebSocket] ⚠️ 收到空数据数组')
        }

        // 收到第一批数据后隐藏加载状态
        console.log('[WebSocket] 🔄 准备设置加载状态为IDLE，当前状态:', loadingState.value.status)

        // 使用Promise确保状态设置的原子性
        new Promise((resolve) => {
          setTimeout(() => {
            updateLoadingState(LOADING_STATES.IDLE)
            console.log('[WebSocket] ✅ 加载状态已设置为IDLE，当前状态:', loadingState.value.status, 'isLoading:', isLoading.value)
            resolve()
          }, 100)
        }).then(() => {
          // 立即检查状态是否正确
          if (loadingState.value.status !== LOADING_STATES.IDLE) {
            console.error('[WebSocket] ⚠️ 状态设置失败，强制重新设置')
            updateLoadingState(LOADING_STATES.IDLE)
          }

          // 延迟检查是否有其他地方修改了状态
          setTimeout(() => {
            if (loadingState.value.status !== LOADING_STATES.IDLE) {
              console.error('[WebSocket] ⚠️ 状态被意外修改，强制修复，当前状态:', loadingState.value.status)
              updateLoadingState(LOADING_STATES.IDLE)
            }
          }, 1000)

          // nextTick验证
          nextTick(() => {
            console.log('[WebSocket] ✅ nextTick验证最终状态:', loadingState.value.status, 'isLoading:', isLoading.value)
          })
        })

        // 强制触发响应式更新
        nextTick(() => {
          console.log('[WebSocket] ✅ nextTick后状态:', loadingState.value.status, 'isLoading:', isLoading.value)
        })
      } else if (data.type === 'error') {
        console.error('[WebSocket] ❌ 服务器错误:', data.message, '错误详情:', data.error)
        updateErrorState(true, data.message || '服务器错误')
      } else {
        console.log('[WebSocket] ❓ 未知消息类型:', data.type, '完整消息:', data)
      }
    } catch (error) {
      console.error('[WebSocket] 解析消息失败:', error, '原始数据:', event.data)
      updateErrorState(true, '数据解析失败')
    }
  }

  ws.onclose = (event) => {
    console.log('[WebSocket] 涨幅榜连接已关闭，代码:', event.code, '原因:', event.reason)
    clearTimeout(connectionTimeout)

    // 更新错误统计
    updateErrorStats({ code: event.code, reason: event.reason }, ERROR_TYPES.WEBSOCKET)

    // 如果是异常关闭，智能重连
    if (event.code !== 1000 && shouldRetry({ code: event.code }, connectionState.value.attempts)) {
      const delay = calculateBackoffDelay(connectionState.value.attempts)
      console.log(`[WebSocket] 异常关闭，${delay}ms后重连...`)

      // 停止消息超时检测，避免在重连过程中触发额外重连
      stopMessageTimeoutDetection()

      updateConnectionState(CONNECTION_STATES.RECONNECTING)

      connectionState.value.reconnectTimer = setTimeout(() => {
        connectWebSocket()
      }, delay)
    } else if (event.code !== 1000) {
      console.log('[WebSocket] 达到最大重连次数，停止重连')
      updateConnectionState(CONNECTION_STATES.FAILED)
      updateCircuitBreaker(false)
      updateErrorState(true, '连接失败，请刷新页面重试')
    } else {
      // 正常关闭
      updateConnectionState(CONNECTION_STATES.DISCONNECTED)
      updateCircuitBreaker(true) // 正常关闭算成功
    }

    // 停止消息超时检测
    stopMessageTimeoutDetection()
  }

  ws.onerror = (error) => {
    console.error('[WebSocket] 涨幅榜连接错误:', error)
    clearTimeout(connectionTimeout)

    // 更新错误统计
    updateErrorStats(error, ERROR_TYPES.WEBSOCKET)

    updateConnectionState(CONNECTION_STATES.DISCONNECTED)

    // 停止消息超时检测
    stopMessageTimeoutDetection()

    // 只有在没有其他错误消息时才设置网络连接错误
    if (!errorState.value.message || errorState.value.message.includes('网络')) {
      const userMessage = getUserFriendlyErrorMessage(error, ERROR_TYPES.WEBSOCKET, 'WebSocket连接')
      updateErrorState(true, userMessage)
    }
  }
}

function disconnectWebSocket() {
  if (connectionState.value.websocket) {
    connectionState.value.websocket.close()
    connectionState.value.websocket = null
  }
  if (connectionState.value.reconnectTimer) {
    clearTimeout(connectionState.value.reconnectTimer)
    connectionState.value.reconnectTimer = null
  }

  // 停止消息超时检测
  stopMessageTimeoutDetection()

  updateConnectionState(CONNECTION_STATES.DISCONNECTED)
  console.log('[WebSocket] 🔌 连接已断开，所有状态已重置')
}

// 初次加载数据（WebSocket连接确认后调用）
async function loadInitialData() {
  await executeWithRetry(async () => {
    console.log('[Data] 开始加载初始数据...')
    const response = await api.realtimeGainers({
      kind: safeSelectedKind,
      category: selectedCategory.value,
      limit: 15, // 只显示15个币种
      sort_by: 'change',
      sort_order: 'desc', // 涨幅从高到低排序
      filter_positive_only: filterState.value.showPositiveOnly,
      filter_large_cap: filterState.value.showLargeCapOnly
    })

    if (response.gainers && response.gainers.length > 0) {
      updateDataState(response.gainers, Date.now())
      updateErrorState(false)
      console.log('[Data] 成功加载', response.gainers.length, '条数据，总共可用', response.total_available || response.gainers.length, '条')
      return response
    } else {
      throw new Error('未获取到有效数据')
    }
  }, '加载初始数据').catch(err => {
    // executeWithRetry已经处理了错误，这里只需要最后的fallback
    updateDataState([], Date.now())
    updateLoadingState(LOADING_STATES.IDLE)
  }).finally(() => {
    updateLoadingState(LOADING_STATES.IDLE)
  })
}

// 传统HTTP加载（降级方案）
async function loadFallbackData() {
  updateLoadingState(LOADING_STATES.REFRESHING)

  await executeWithRetry(async () => {
    console.log('[Data] 使用HTTP降级加载数据...')
    const response = await api.realtimeGainers({
      kind: safeSelectedKind,
      category: selectedCategory.value,
      limit: 15, // 只显示15个币种
      sort_by: 'change',
      sort_order: 'desc', // 涨幅从高到低排序
      filter_positive_only: filterState.value.showPositiveOnly,
      filter_large_cap: filterState.value.showLargeCapOnly
    })

    if (response.gainers && response.gainers.length > 0) {
      updateDataState(response.gainers, Date.now())
      updateErrorState(false)
      console.log('[Data] HTTP降级加载成功', response.gainers.length, '条数据')
      return response
    } else {
      updateDataState([], Date.now())
      throw new Error('未获取到有效数据')
    }
  }, 'HTTP降级加载').catch(err => {
    // executeWithRetry已经处理了错误，这里只需要最后的fallback
    updateDataState([], Date.now())
  }).finally(() => {
    updateLoadingState(LOADING_STATES.IDLE)
  })
}

// 处理分类选择器变化
function handleCategoryChange() {
  // 重置状态
  updateLoadingState(LOADING_STATES.REFRESHING)
  updateErrorState(false)
  connectionState.value.attempts = 0

  // 重新连接WebSocket（使用新的分类）
  disconnectWebSocket()
  setupWebSocketConnection()
}

// 监听交易类型变化
// 存储当前正在连接的市场类型，避免竞态条件
let connectingKind = 'spot'

watch(selectedKind, (newKind) => {
  const targetKind = newKind
  console.log('[切换] 🔄 开始切换到市场:', targetKind)
  console.log('[切换] 📊 当前状态: connectionState=', connectionState.value.status, 'loadingState=', loadingState.value.status)

  // 设置正在连接的市场类型
  connectingKind = targetKind

  // 重置状态 - 先断开连接，再重置状态，避免竞态条件
  console.log('[切换] 🔌 断开旧连接...')
  disconnectWebSocket()

  // 重置状态
  updateLoadingState(LOADING_STATES.REFRESHING)
  updateErrorState(false)
  connectionState.value.attempts = 0
  console.log('[切换] 🔄 状态已重置，loading=refreshing')

  // 重新连接WebSocket（使用新的交易类型）
  console.log('[切换] 🔗 建立新连接...')
  setupWebSocketConnection()
  console.log('[切换] ✅ 切换完成，等待连接结果...')
})

// 监听分类变化
watch(selectedCategory, () => {
  handleCategoryChange()
})

// 监听筛选条件变化
watch([() => filterState.value.showPositiveOnly, () => filterState.value.showLargeCapOnly], () => {
  console.log('[Filter] ⚠️ 筛选条件变化触发，连接状态:', connectionState.value.status, 'loading状态:', loadingState.value.status)

  // 如果是WebSocket连接状态，只重新排序前端数据
  if (connectionState.value.status === CONNECTION_STATES.CONNECTED) {
    console.log('[Filter] 筛选条件变化，只更新前端显示')
    return
  }

  // 如果是HTTP降级状态，重新获取数据
  if (connectionState.value.status !== CONNECTION_STATES.CONNECTED && loadingState.value.status === LOADING_STATES.IDLE) {
    console.log('[Filter] ⚠️ 筛选条件变化，重新获取数据')
    updateLoadingState(LOADING_STATES.REFRESHING)
    loadInitialData()
  }
})

// 页面激活时的处理函数（包括初次加载和从keep-alive恢复）
const handlePageActivated = () => {
  console.log('[RealTimeGainers] ⚡ 页面激活，当前连接状态:', connectionState.value.status, 'loading状态:', loadingState.value.status)
  console.log('[RealTimeGainers] 数据时间戳:', dataState.value.lastUpdateTimestamp, '当前时间:', Date.now())

  // 检查数据是否过期（超过5分钟）
  if (dataState.value.lastUpdateTimestamp && Date.now() - dataState.value.lastUpdateTimestamp > 5 * 60 * 1000) {
    console.log('[RealTimeGainers] ⚠️ 数据过期，重新加载')
    updateLoadingState(LOADING_STATES.REFRESHING)
    setupWebSocketConnection()
  } else if (connectionState.value.status !== CONNECTION_STATES.CONNECTED) {
    console.log('[RealTimeGainers] WebSocket未连接，开始连接...')
    setupWebSocketConnection()
  } else {
    console.log('[RealTimeGainers] WebSocket已连接，保持现有连接')
    // 如果页面是激活状态但还在加载，尝试获取最新数据
    if (loadingState.value.status === LOADING_STATES.INITIAL) {
      loadFallbackData()
    }
  }
}

// WebSocket连接和降级逻辑
const setupWebSocketConnection = () => {
  // 如果正在连接中，跳过
  if (connectionState.value.status === CONNECTION_STATES.CONNECTING) {
    console.log('[RealTimeGainers] ⚠️ 正在连接中，跳过重复连接')
    return
  }

  console.log('[RealTimeGainers] 🚀 设置WebSocket连接，目标市场:', safeSelectedKind)
  updateConnectionState(CONNECTION_STATES.CONNECTING)
  console.log('[WebSocket] 🔒 连接锁已设置')
  connectWebSocket()

  // 如果15秒后还没有收到数据，使用HTTP API作为降级方案
  const fallbackTimer = setTimeout(() => {
    if (loadingState.value.status === LOADING_STATES.INITIAL && connectionState.value.status !== CONNECTION_STATES.CONNECTED) {
      console.log('[WebSocket] 连接超时，降级使用HTTP API')
      loadFallbackData()
    }
  }, 15000)

  // 如果30秒后仍无数据，显示错误
  const errorTimer = setTimeout(() => {
    if (loadingState.value.status === LOADING_STATES.INITIAL) {
      console.log('[WebSocket] 数据加载超时')
      updateErrorState(true, '数据加载超时，请检查网络连接')
      updateLoadingState(LOADING_STATES.IDLE)
    }
  }, 30000)

  // 清理定时器
  const cleanup = () => {
    clearTimeout(fallbackTimer)
    clearTimeout(errorTimer)
  }

  // 当收到数据时清理定时器
  watch([() => connectionState.value.status === CONNECTION_STATES.CONNECTED, () => dataState.value.gainers.length], ([connected, dataLength]) => {
    if (connected || dataLength > 0) {
      cleanup()
    }
  })

  return cleanup
}

// 页面失活时的处理函数（进入keep-alive缓存）
const handlePageDeactivated = () => {
  console.log('[RealTimeGainers] 页面失活，保持WebSocket连接')
  // 不在这里断开连接，让keep-alive保持连接
}

// 页面挂载时建立WebSocket连接
onMounted(() => {
  console.log('[RealTimeGainers] 页面挂载')
  setupWebSocketConnection()

  // 监听网络状态变化
  window.addEventListener('online', handleNetworkOnline)
  window.addEventListener('offline', handleNetworkOffline)

  // 定期检查网络状态
  setInterval(checkNetworkStatus, 30000) // 每30秒检查一次
})

// 网络恢复处理
function handleNetworkOnline() {
  console.log('[Network] 网络已恢复')
  errorState.value.networkStatus = true
  errorState.value.lastNetworkCheck = Date.now()

  // 如果之前连接失败，尝试重连
  if (connectionState.value.status !== CONNECTION_STATES.CONNECTED && loadingState.value.status === LOADING_STATES.IDLE) {
    console.log('[Network] 网络恢复，尝试重连...')
    updateErrorState(false)
    setupWebSocketConnection()
  }

  // 如果WebSocket连接正常，清除网络相关的错误消息
  if (connectionState.value.status === CONNECTION_STATES.CONNECTED && errorState.value.message.includes('网络')) {
    console.log('[Network] WebSocket连接正常，清除网络错误消息')
    updateErrorState(false)
  }
}

// 网络断开处理
function handleNetworkOffline() {
  console.log('[Network] 网络已断开')
  errorState.value.networkStatus = false
  errorState.value.lastNetworkCheck = Date.now()

  // 显示网络错误，但只有在WebSocket也断开的情况下才显示
  if (connectionState.value.status !== CONNECTION_STATES.CONNECTED && !errorState.value.message.includes('网络')) {
    updateErrorState(true, '网络连接已断开，请检查网络设置')
  }
}

// ===== WebSocket消息超时检测 =====

// 启动消息超时检测
function startMessageTimeoutDetection() {
  // 清除之前的定时器
  if (connectionState.value.messageTimeoutTimer) {
    clearInterval(connectionState.value.messageTimeoutTimer)
  }

  // 每10秒检查一次消息超时
  connectionState.value.messageTimeoutTimer = setInterval(() => {
    checkMessageTimeout()
  }, 10000)
}

// 停止消息超时检测
function stopMessageTimeoutDetection() {
  if (connectionState.value.messageTimeoutTimer) {
    clearInterval(connectionState.value.messageTimeoutTimer)
    connectionState.value.messageTimeoutTimer = null
  }
}

// 检查消息超时
function checkMessageTimeout() {
  const now = Date.now()
  const lastMessageTime = connectionState.value.lastMessageTime
  const timeoutThreshold = 60000 // 60秒超时阈值

  // 只在"已连接"状态下检查超时，在"重连中"状态时不检查避免重复触发
  if (connectionState.value.status === CONNECTION_STATES.CONNECTED &&
      lastMessageTime > 0 &&
      now - lastMessageTime > timeoutThreshold) {

    console.log(`[WebSocket] 消息接收超时: ${now - lastMessageTime}ms 未收到消息，触发重连`)

    // 强制断开连接，这会触发重连逻辑
    if (connectionState.value.websocket) {
      connectionState.value.websocket.close(4000, '消息接收超时')
    } else {
      // 如果WebSocket对象不存在，直接更新状态
      updateConnectionState(CONNECTION_STATES.DISCONNECTED)
      updateErrorState(true, '连接超时，请检查网络连接')
    }
  }
}

// 更新最后消息时间
function updateLastMessageTime() {
  connectionState.value.lastMessageTime = Date.now()
}

// ===== 简化的错误处理 =====


// 强制重置错误状态（用于极端情况）
function forceResetErrors() {
  console.log('[ErrorRecovery] 强制重置所有错误状态')

  // 重置熔断器
  circuitBreaker.value.state = CIRCUIT_STATES.CLOSED
  circuitBreaker.value.failureCount = 0
  circuitBreaker.value.successCount = 0
  circuitBreaker.value.nextAttemptTime = 0

  // 重置错误统计
  errorStats.value.consecutiveFailures = 0
  errorStats.value.totalErrors = 0
  errorStats.value.errorTypes.clear()

  // 清除错误状态
  updateErrorState(false)

  // 重新尝试连接
  disconnectWebSocket()
  setTimeout(() => {
    setupWebSocketConnection()
  }, 500)
}

// 检查网络状态
async function checkNetworkStatus() {
  try {
    // 尝试ping一个可靠的服务来检查网络
    const response = await fetch('/healthz', {
      method: 'HEAD',
      cache: 'no-cache',
      timeout: 5000
    })
    const isOnline = response.ok

    if (isOnline !== errorState.value.networkStatus) {
      errorState.value.networkStatus = isOnline
      errorState.value.lastNetworkCheck = Date.now()

      if (isOnline) {
        handleNetworkOnline()
      } else {
        handleNetworkOffline()
      }
    }
  } catch (error) {
    // 网络请求失败，认为是离线状态
    if (errorState.value.networkStatus) {
      handleNetworkOffline()
    }
  }
}

// 页面卸载时清理连接和监听器
onUnmounted(() => {
  console.log('[RealTimeGainers] 页面卸载，清理资源')

  // 断开WebSocket连接
  disconnectWebSocket()

  // 停止消息超时检测
  stopMessageTimeoutDetection()

  // 移除网络状态监听器
  window.removeEventListener('online', handleNetworkOnline)
  window.removeEventListener('offline', handleNetworkOffline)

  // 清理变化通知定时器
  if (changeNotificationTimeout.value) {
    clearTimeout(changeNotificationTimeout.value)
    changeNotificationTimeout.value = null
  }
})

// keep-alive生命周期钩子
onActivated(handlePageActivated)
onDeactivated(handlePageDeactivated)
</script>

<style scoped>
.page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 14px 40px;
  background: transparent;
  transition: none;
}

.page-header {
  margin-bottom: 16px;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.selectors {
  display: flex;
  align-items: center;
  gap: 16px;
}

.type-selector {
  display: flex;
  gap: 4px;
  background: rgba(0,0,0,.05);
  border-radius: 8px;
  padding: 2px;
}

.type-btn {
  padding: 6px 16px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #666;
  transition: all 0.2s;
}

.type-btn:hover {
  background: rgba(0,0,0,.1);
}

.type-btn.active {
  background: #3b82f6;
  color: white;
}

.category-selector {
  display: flex;
  align-items: center;
}

.category-select {
  height: 32px;
  padding: 0 12px;
  border: 1px solid rgba(0,0,0,.15);
  border-radius: 6px;
  background: #fff;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  min-width: 140px;
  transition: all 0.2s ease;
}

.category-select:hover {
  border-color: rgba(0,0,0,.25);
}

.category-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.invite-link {
  padding: 6px 16px;
  background: #3b82f6;
  color: #fff;
  text-decoration: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  transition: background 0.2s;
}

.invite-link:hover {
  background: #2563eb;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.last-update {
  font-size: 12px;
  color: #888;
  display: flex;
  align-items: center;
  gap: 4px;
}

.stale-indicator {
  color: #f59e0b;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #888;
}

.connection-status.connected {
  color: #22c55e;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ef4444;
  animation: pulse 2s infinite;
}

.connection-status.connected .status-dot {
  background: #22c55e;
  animation: none;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}


.btn {
  height: 32px;
  padding: 0 12px;
  border: 1px solid rgba(0,0,0,.15);
  background: #fff;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.btn:hover:not(:disabled) {
  background: #f8f9fa;
  border-color: rgba(0,0,0,.2);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,.1);
}

.btn:disabled {
  opacity: .6;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.btn-icon {
  font-size: 14px;
}

.refresh-btn {
  background: #3b82f6;
  color: white;
  border-color: #3b82f6;
}

.refresh-btn:hover:not(:disabled) {
  background: #2563eb;
  border-color: #2563eb;
}


.error-banner {
  background: linear-gradient(135deg, #fee2e2, #fecaca);
  border: 1px solid #f87171;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  animation: slideIn 0.3s ease-out;
}

.error-icon {
  font-size: 16px;
}

.error-message {
  flex: 1;
  color: #dc2626;
  font-size: 14px;
}

.error-close {
  background: none;
  border: none;
  color: #dc2626;
  cursor: pointer;
  font-size: 16px;
  padding: 2px;
  border-radius: 4px;
  transition: background 0.2s;
}

.error-close:hover {
  background: rgba(220, 38, 38, 0.1);
}

.error-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.error-retry {
  background: #3b82f6;
  color: white;
  border: 1px solid #3b82f6;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.error-retry:hover {
  background: #2563eb;
  border-color: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(59, 130, 246, 0.3);
}

/* 变化通知样式 */
.change-notification {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  border: 1px solid #16a34a;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  animation: slideIn 0.3s ease-out, fadeOut 0.3s ease-out 2.7s;
  box-shadow: 0 2px 8px rgba(34, 197, 94, 0.2);
}

.change-icon {
  font-size: 16px;
  filter: brightness(1.2);
}

.change-message {
  flex: 1;
  color: #ffffff;
  font-size: 14px;
  font-weight: 500;
}

.loading {
  padding: 80px 0;
  text-align: center;
  color: #888;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #e5e7eb;
  border-top: 3px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.loading-text {
  font-size: 16px;
  font-weight: 500;
}

.loading-hint {
  font-size: 12px;
  color: #9ca3af;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.realtime-table {
  background: rgba(255,255,255,.02);
  border: 1px solid rgba(0,0,0,.1);
  border-radius: 12px;
  overflow: hidden;
}

.tbl {
  width: 100%;
  border-collapse: collapse;
}

.tbl th, .tbl td {
  padding: 12px 8px;
  text-align: center;
}

.tbl thead th {
  font-size: 12px;
  color: #666;
  font-weight: 500;
  border-bottom: 1px solid rgba(0,0,0,.06);
  background: rgba(0,0,0,.02);
}

.tbl tbody td {
  font-size: 14px;
  font-weight: 500;
  border-bottom: 1px solid rgba(0,0,0,.03);
}

.tbl tbody tr:hover {
  background: rgba(0,0,0,.01);
}

/* 列宽 */
.col-rank {
  width: 60px;
}

.col-symbol {
  width: 140px;
  font-weight: 600;
  text-align: left;
}

.col-num {
  font-variant-numeric: tabular-nums;
  width: 110px;
}



/* 表格行高亮 */
.highlight-row {
  animation: highlight 3s ease-out;
}

@keyframes highlight {
  0% { background: #fef3c7; }
  100% { background: transparent; }
}


/* 价格单元格 */
.price-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 100%;
  height: 100%;
}

.price-trend {
  font-size: 12px;
  opacity: 0.7;
}

/* 涨跌幅条 */
.change-bar {
  position: absolute;
  left: 0;
  top: 0;
  height: 100%;
  border-radius: 4px;
  z-index: -1;
}

/* 成交量变化 */
.volume-change {
  font-size: 10px;
  opacity: 0.7;
}

/* 置信度指示器 */
.confidence-indicator {
  font-size: 9px;
  padding: 1px 4px;
  border-radius: 8px;
  margin-left: 4px;
}

.confidence-indicator.high {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.confidence-indicator.medium {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.confidence-indicator.low {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

/* 币种链接样式 */
.symbol-link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 600;
  transition: color 0.2s ease;
}

.symbol-link:hover {
  color: #1d4ed8;
  text-decoration: underline;
}

/* 非原生币样式 */
.symbol-text {
  color: #000000;
  font-weight: 500;
  cursor: default;
}

/* 前15名币种特殊标识 */
.top15-badge {
  font-weight: bold;
  color: #dc2626;
}

.top15-indicator {
  margin-left: 4px;
  font-size: 12px;
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

/* 颜色 */
.up {
  color: #22c55e !important;
}

.down {
  color: #ef4444 !important;
}

/* 确保涨跌幅单元格的颜色正确应用 */
.change-cell.up .change-value {
  color: #22c55e !important;
}

.change-cell.down .change-value {
  color: #ef4444 !important;
}


@media (max-width: 768px) {
  .header-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .selectors {
    flex-wrap: wrap;
    gap: 8px;
  }

  .stats {
    width: 100%;
  }


  .realtime-table {
    overflow-x: auto;
  }

  .tbl {
    min-width: 600px;
  }

  .col-symbol {
    width: 120px;
  }


  .error-banner {
    padding: 8px 12px;
  }

  .error-message {
    font-size: 13px;
  }
}
</style>
