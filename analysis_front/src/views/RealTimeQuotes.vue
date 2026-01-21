<!-- src/views/RealTimeQuotes.vue -->
<template>
  <div class="page">
    <header class="page-header">
      <div class="header-top">
        <div class="header-info">
          <h2 class="page-title">
            <span class="title-icon">📊</span>
            实时行情
          </h2>
          <div class="market-status">
            <div class="status-indicator" :class="{ online: wsConnected, offline: !wsConnected }">
              <span class="status-dot"></span>
              <span class="status-text">{{ wsConnected ? '实时连接' : '连接中...' }}</span>
            </div>
            <div class="last-update" v-if="lastUpdateTime">
              最后更新: {{ formatLastUpdate(lastUpdateTime) }}
            </div>
          </div>
        </div>

        <div class="header-controls">
          <!-- 筛选和配置控件 -->
          <div class="filter-controls">
            <!-- 显示数量选择器已移除 -->
          </div>
        </div>
      </div>
    </header>

    <!-- 主要内容区域 -->
    <div class="content">
    <!-- 全局加载状态 -->
    <div v-if="initialLoading || loading" class="loading">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">正在获取实时行情数据...</div>
        <div class="loading-hint">首次加载可能需要10-15秒</div>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error-state">
      <div class="error-icon">⚠️</div>
      <div class="error-title">数据加载失败</div>
      <div class="error-message">{{ error }}</div>
      <button class="retry-btn" @click="refreshData">
        <span>🔄</span>
        重试
      </button>
    </div>
      <div v-else-if="visibleSymbols.length > 0" class="symbols-table-container">
        <table class="symbols-table">
          <thead>
            <tr>
              <th class="col-symbol sortable" @click="sortBy('symbol')">
                <div class="header-content">
                  <span>币种</span>
                  <span class="sort-icon">{{ getSortIcon('symbol') }}</span>
                </div>
              </th>
              <th class="col-price sortable" @click="sortBy('price')">
                <div class="header-content">
                  <span>最新价</span>
                  <span class="sort-icon">{{ getSortIcon('price') }}</span>
                </div>
              </th>
              <th class="col-change sortable" @click="sortBy('change')">
                <div class="header-content">
                  <span>24h涨跌</span>
                  <span class="sort-icon">{{ getSortIcon('change') }}</span>
                </div>
              </th>
              <th class="col-high sortable" @click="sortBy('high')">
                <div class="header-content">
                  <span>24h最高</span>
                  <span class="sort-icon">{{ getSortIcon('high') }}</span>
                </div>
              </th>
              <th class="col-low sortable" @click="sortBy('low')">
                <div class="header-content">
                  <span>24h最低</span>
                  <span class="sort-icon">{{ getSortIcon('low') }}</span>
                </div>
              </th>
              <th class="col-volume sortable" @click="sortBy('volume')">
                <div class="header-content">
                  <span>成交量</span>
                  <span class="sort-icon">{{ getSortIcon('volume') }}</span>
                </div>
              </th>
              <th class="col-marketcap sortable" @click="sortBy('marketcap')">
                <div class="header-content">
                  <span>市值</span>
                  <span class="sort-icon">{{ getSortIcon('marketcap') }}</span>
                </div>
              </th>
              <th class="col-expand">
                <div class="header-content">
                  <span>图表</span>
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            <template v-for="(symbol, index) in visibleSymbols" :key="symbol">
              <tr
                :class="['symbol-row', { expanded: expandedRows.has(symbol) }]"
                @click="toggleRowExpansion(symbol)"
              >
                <td class="col-symbol">
                  <div class="symbol-info">
                    <span class="symbol-name">{{ formatSymbolName(symbol) }}</span>
                    <span class="symbol-full">{{ symbol }}</span>
                  </div>
                </td>
                <td class="col-price">
                  <div class="price-container">
                    <div class="price" :class="{ 'price-up': getPriceChange(symbol) > 0, 'price-down': getPriceChange(symbol) < 0 }">
                      {{ formatPrice(getCurrentPrice(symbol)) }}
                    </div>
                    <div
                      v-if="getPriceChangeIndicator(symbol)"
                      class="price-indicator"
                      :class="getPriceChangeIndicator(symbol).type"
                    >
                      <span class="indicator-icon">
                        {{ getPriceChangeIndicator(symbol).type === 'up' ? '↗' : '↘' }}
                      </span>
                    </div>
                  </div>
                </td>
                <td class="col-change">
                  <div class="change" :class="{ 'change-up': getPriceChange(symbol) > 0, 'change-down': getPriceChange(symbol) < 0 }">
                    {{ formatChange(getPriceChange(symbol)) }}
                  </div>
                </td>
                <td class="col-high">{{ formatPrice(getHigh24h(symbol)) }}</td>
                <td class="col-low">{{ formatPrice(getLow24h(symbol)) }}</td>
                <td class="col-volume">
                  <div class="volume-content">
                    <div class="volume-text">{{ formatVolume(getVolume24h(symbol)) }}</div>
                    <div class="volume-bar">
                      <div class="volume-fill" :style="{ width: getVolumePercentage(symbol) + '%' }"></div>
                    </div>
                  </div>
                </td>
                <td class="col-marketcap">
                  {{ formatMarketCap(calculateMarketCap(symbol)) }}
                </td>
                <td class="col-expand">
                  <div class="expand-icon" :class="{ expanded: expandedRows.has(symbol) }">
                    <span>▼</span>
                  </div>
                </td>
              </tr>
              <!-- 展开的K线图行 -->
              <tr v-if="expandedRows.has(symbol)" class="chart-row">
                <td colspan="7" class="chart-cell">
                  <div class="chart-container">
                    <div class="chart-header">
                      <h4>{{ formatSymbolName(symbol) }} K线图</h4>
                      <div class="timeframe-selector">
                        <button
                          v-for="tf in timeframes"
                          :key="tf.value"
                          :class="['tf-btn', { active: getSelectedTimeframeForSymbol(symbol) === tf.value }]"
                          @click.stop="setTimeframeForSymbol(symbol, tf.value)"
                        >
                          {{ tf.label }}
                        </button>
                      </div>
                    </div>
                    <div class="chart-wrapper">
                      <CandlestickChart
                        v-if="getKlineDataForSymbol(symbol).length > 0"
                        :data="getKlineDataForSymbol(symbol)"
                        :title="`${formatSymbolName(symbol)} ${getTimeframeLabel(getSelectedTimeframeForSymbol(symbol))} K线图`"
                        :showVolume="true"
                        :showMA="true"
                        :qualityThreshold="80"
                        :key="`${symbol}_${getSelectedTimeframeForSymbol(symbol)}`"
                      />
                      <div v-else-if="getKlineLoadingForSymbol(symbol)" class="chart-loading">加载K线数据中...</div>
                      <div v-else class="chart-empty">暂无K线数据</div>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>

        <!-- 分页组件 -->
        <div class="pagination-container" v-if="totalPages > 1">
          <button
            class="page-btn"
            :disabled="currentPage === 1"
            @click="goToPage(1)"
            :class="{ disabled: currentPage === 1 }"
          >
            首页
          </button>
          <button
            class="page-btn"
            :disabled="currentPage === 1"
            @click="goToPage(currentPage - 1)"
            :class="{ disabled: currentPage === 1 }"
          >
            上一页
          </button>

          <span class="page-info">
            第 {{ currentPage }} / {{ totalPages }} 页
            (共 {{ totalItems }} 个币种)
          </span>

          <button
            class="page-btn"
            :disabled="currentPage === totalPages"
            @click="goToPage(currentPage + 1)"
            :class="{ disabled: currentPage === totalPages }"
          >
            下一页
          </button>
          <button
            class="page-btn"
            :disabled="currentPage === totalPages"
            @click="goToPage(totalPages)"
            :class="{ disabled: currentPage === totalPages }"
          >
            末页
          </button>
        </div>
      </div>

      <!-- 无数据状态 -->
      <div v-else class="no-data-state">
        <div class="no-data-icon">📊</div>
        <div class="no-data-title">暂无符合条件的币种</div>
        <div class="no-data-message">
          没有找到市值小于5000万的币种数据<br>
          请确保已同步CoinCap市值数据
        </div>
        <button class="refresh-btn" @click="refreshData">
          <span>🔄</span>
          刷新数据
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick, shallowRef, markRaw } from 'vue'
import { api } from '../api/api.js'
import { handleError } from '../utils/errorHandler.js'
import CandlestickChart from '../components/CandlestickChart.vue'

// 状态管理 - 使用浅响应式优化性能
const selectedKind = ref('spot') // 'spot' 或 'futures'
const loading = ref(false)
const initialLoading = ref(true) // 初始加载状态
const error = ref('')
const wsConnected = ref(false) // WebSocket连接状态
const lastUpdateTime = ref(null) // 最后更新时间
// 小市值筛选始终启用，不再需要用户控制

// 排序管理
const sortField = ref('marketcap') // 排序字段 - 默认按市值排序
const sortDirection = ref('desc') // 排序方向: 'asc' | 'desc' - 默认降序（市值高的在上面）

// 分页管理
const currentPage = ref(1) // 当前页码
const pageSize = ref(10) // 每页显示数量，默认10个

// 展开状态管理 - 使用浅响应式Set优化性能
const expandedRows = shallowRef(new Set())

// 虚拟滚动配置
const virtualScroll = {
  itemHeight: 80,      // 每行高度
  containerHeight: 600, // 容器高度
  bufferSize: 5,       // 缓冲区大小
  startIndex: 0,       // 开始索引
  endIndex: 0,         // 结束索引
  visibleItems: []     // 可见项
}

// 数据 - 使用浅响应式优化性能
const symbols = shallowRef([])
const prices = shallowRef(new Map()) // symbol -> price data

// K线数据缓存 - 非响应式，减少开销
const klineDataCache = new Map() // symbol -> kline data array (非响应式)
const klineLoadingMap = shallowRef(new Map()) // symbol -> loading state
const selectedTimeframes = shallowRef(new Map()) // symbol -> selected timeframe

// 价格更新防抖缓存
const priceUpdateCache = new Map() // symbol -> last update timestamp
const PRICE_UPDATE_DEBOUNCE = 100 // 100ms防抖

// 价格变化标记
const priceChangeIndicators = shallowRef(new Map()) // symbol -> {type: 'up'|'down', timestamp: number}
const PRICE_CHANGE_DURATION = 3000 // 标记显示3秒

// 虚拟滚动容器引用
const scrollContainer = ref(null)

// 时间周期选项
const timeframes = [
  { value: '5m', label: '5分' },
  { value: '15m', label: '15分' },
  { value: '1h', label: '1小时' },
  { value: '4h', label: '4小时' },
  { value: '1d', label: '日线' }
]

// 计算市值（智能处理：真实市值 > 估算市值 > 筛选大值）
function calculateMarketCap(symbol) {
  const priceData = prices.value.get(symbol)

  // 优先使用CoinCap提供的真实市值
  if (priceData && priceData.marketCapUSD && priceData.marketCapUSD > 0) {
    return priceData.marketCapUSD
  }

  // 没有真实市值就返回大值（被筛选掉）
  return 100000000 // 1亿美元，远大于5000万的筛选阈值

  // 如果没有启用小市值筛选，进行估算（用于显示）
  const price = getCurrentPrice(symbol)
  const volume = getVolume24h(symbol)

  if (price && volume && volume > 0) {
    // 估算公式：市值 ≈ 价格 × (24h成交量 / 价格) × 调整系数
    const baseValue = volume * Math.min(price / 100, 1) // 价格越高，调整系数越小
    const estimated = Math.max(baseValue, volume * 0.1) // 至少是成交量的10%

    return estimated
  }

  // 如果没有任何数据，返回最小值（仍然显示，但排在最后）
  return 1000 // 1千美元的最小市值
}

// 计算属性 - 筛选和排序后的符号列表
const sortedSymbols = computed(() => {
  let symbolsArray = [...symbols.value]

  // 小币种筛选（市值 < 5000万）- 默认启用
  symbolsArray = symbolsArray.filter(symbol => {
    const marketCap = calculateMarketCap(symbol)
    // 始终只显示市值<5000万的币种
    return marketCap > 0 && marketCap < 50000000
  })

  // 排序
  return symbolsArray.sort((a, b) => {
    let aVal, bVal

    switch (sortField.value) {
      case 'symbol':
        aVal = a
        bVal = b
        break
      case 'price':
        aVal = getCurrentPrice(a)
        bVal = getCurrentPrice(b)
        break
      case 'change':
        aVal = getPriceChange(a)
        bVal = getPriceChange(b)
        break
      case 'high':
        aVal = getHigh24h(a)
        bVal = getHigh24h(b)
        break
      case 'low':
        aVal = getLow24h(a)
        bVal = getLow24h(b)
        break
      case 'volume':
        aVal = getVolume24h(a)
        bVal = getVolume24h(b)
        break
      case 'marketcap':
        aVal = calculateMarketCap(a)
        bVal = calculateMarketCap(b)
        break
      default:
        return 0
    }

    // 处理数值比较
    if (typeof aVal === 'string') {
      aVal = parseFloat(aVal) || 0
    }
    if (typeof bVal === 'string') {
      bVal = parseFloat(bVal) || 0
    }

    if (sortDirection.value === 'asc') {
      return aVal > bVal ? 1 : aVal < bVal ? -1 : 0
    } else {
      return aVal < bVal ? 1 : aVal > bVal ? -1 : 0
    }
  })
})

// 计算属性 - 直接使用排序后的符号列表（分页将在后端处理）

// 计算属性 - 分页后的符号列表
const paginatedSymbols = computed(() => {
  const startIndex = (currentPage.value - 1) * pageSize.value
  const endIndex = startIndex + pageSize.value
  return sortedSymbols.value.slice(startIndex, endIndex)
})

// 计算属性 - 总页数
const totalPages = computed(() => {
  return Math.ceil(sortedSymbols.value.length / pageSize.value)
})

// 计算属性 - 总项目数
const totalItems = computed(() => {
  return sortedSymbols.value.length
})

// 计算属性 - 当前显示的符号列表（兼容现有代码）
const visibleSymbols = computed(() => {
  return paginatedSymbols.value
})

// WebSocket连接
let ws = null
let reconnectTimer = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 5
const RECONNECT_INTERVAL = 5000 // 5秒重连间隔
const WS_URL = import.meta.env.DEV ? 'ws://127.0.0.1:8010/ws/prices' : '/ws/prices'

// 格式化函数
function formatSymbolName(symbol) {
  if (!symbol) return symbol
  // 去掉_PERP后缀和USDT等
  return symbol.replace('_PERP', '').replace(/USDT$|BUSD$|USDC$/i, '')
}

function formatPrice(price) {
  if (!price || price === 0) return '--'
  if (price >= 1) {
    return price.toLocaleString(undefined, { maximumFractionDigits: 2, minimumFractionDigits: 2 })
  } else {
    return price.toPrecision(4)
  }
}

function formatChange(change) {
  if (change === null || change === undefined) return '--'
  const sign = change >= 0 ? '+' : ''
  return `${sign}${change.toFixed(2)}%`
}

function formatVolume(volume) {
  if (!volume) return '--'
  if (volume >= 1e9) return `${(volume / 1e9).toFixed(1)}B`
  if (volume >= 1e6) return `${(volume / 1e6).toFixed(1)}M`
  if (volume >= 1e3) return `${(volume / 1e3).toFixed(1)}K`
  return volume.toFixed(0)
}

function formatMarketCap(marketCap) {
  if (!marketCap || marketCap === 0) return '--'

  if (marketCap >= 1e12) { // 万亿以上
    return `${(marketCap / 1e12).toFixed(2)}T`
  } else if (marketCap >= 1e9) { // 十亿以上
    return `${(marketCap / 1e9).toFixed(2)}B`
  } else if (marketCap >= 1e6) { // 百万以上
    return `${(marketCap / 1e6).toFixed(2)}M`
  } else if (marketCap >= 1e3) { // 千以上
    return `${(marketCap / 1e3).toFixed(1)}K`
  } else {
    return marketCap.toLocaleString(undefined, { maximumFractionDigits: 0 })
  }
}

function formatTime(timestamp) {
  const date = new Date(timestamp * 1000)
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    month: '2-digit',
    day: '2-digit'
  })
}

function formatLastUpdate(timestamp) {
  if (!timestamp) return '--'
  const now = Date.now()
  const diff = now - timestamp

  if (diff < 1000) return '刚刚'
  if (diff < 60000) return `${Math.floor(diff / 1000)}秒前`
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  return `${Math.floor(diff / 3600000)}小时前`
}

function getTimeframeLabel(value) {
  const tf = timeframes.find(t => t.value === value)
  return tf ? tf.label : value
}



// 获取成交量百分比（相对于当前页面中最大成交量的比例）
function getVolumePercentage(symbol) {
  // 使用当前页面的symbols列表来计算相对比例
  const currentSymbols = paginatedSymbols.value
  const volumes = currentSymbols.map(s => parseFloat(getVolume24h(s)) || 0)
  const maxVolume = Math.max(...volumes)
  const currentVolume = parseFloat(getVolume24h(symbol)) || 0

  if (maxVolume === 0) return 0
  return Math.min((currentVolume / maxVolume) * 100, 100)
}

// 排序功能
function sortBy(field) {
  if (sortField.value === field) {
    // 切换排序方向
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    // 设置新的排序字段
    sortField.value = field
    sortDirection.value = 'desc' // 市值等数值字段默认降序
  }

  // 排序变化时重置到第一页
  currentPage.value = 1

  // 对symbols进行排序
  symbols.value.sort((a, b) => {
    let aVal, bVal

    switch (field) {
      case 'symbol':
        aVal = a
        bVal = b
        break
      case 'price':
        aVal = getCurrentPrice(a)
        bVal = getCurrentPrice(b)
        break
      case 'change':
        aVal = getPriceChange(a)
        bVal = getPriceChange(b)
        break
      case 'high':
        aVal = getHigh24h(a)
        bVal = getHigh24h(b)
        break
      case 'low':
        aVal = getLow24h(a)
        bVal = getLow24h(b)
        break
      case 'volume':
        aVal = getVolume24h(a)
        bVal = getVolume24h(b)
        break
      case 'marketcap':
        aVal = calculateMarketCap(a)
        bVal = calculateMarketCap(b)
        break
      default:
        return 0
    }

    // 处理数值比较
    if (typeof aVal === 'string') {
      aVal = parseFloat(aVal) || 0
    }
    if (typeof bVal === 'string') {
      bVal = parseFloat(bVal) || 0
    }

    if (sortDirection.value === 'asc') {
      return aVal > bVal ? 1 : aVal < bVal ? -1 : 0
    } else {
      return aVal < bVal ? 1 : aVal > bVal ? -1 : 0
    }
  })
}

// 获取排序图标
function getSortIcon(field) {
  if (sortField.value !== field) return '↕️'
  return sortDirection.value === 'asc' ? '↑' : '↓'
}

// 分页方法
function goToPage(page) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
  }
}

// 虚拟滚动更新
function updateVirtualScroll(scrollTop = 0) {
  const totalItems = sortedSymbols.value.length
  if (totalItems === 0) {
    virtualScroll.visibleItems = []
    return
  }

  const { itemHeight, containerHeight, bufferSize } = virtualScroll
  const visibleCount = Math.ceil(containerHeight / itemHeight)
  const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - bufferSize)
  const endIndex = Math.min(totalItems - 1, startIndex + visibleCount + bufferSize * 2)

  virtualScroll.startIndex = startIndex
  virtualScroll.endIndex = endIndex
  virtualScroll.visibleItems = sortedSymbols.value.slice(startIndex, endIndex + 1)
}

// 虚拟滚动事件处理
function handleScroll(event) {
  const scrollTop = event.target.scrollTop
  updateVirtualScroll(scrollTop)
}

// 获取虚拟滚动样式
function getVirtualScrollStyle() {
  return {
    height: `${sortedSymbols.value.length * virtualScroll.itemHeight}px`,
    paddingTop: `${virtualScroll.startIndex * virtualScroll.itemHeight}px`
  }
}

// 数据获取函数
function getCurrentPrice(symbol) {
  const data = prices.value.get(symbol)
  return data ? data.price : 0
}

function getPriceChange(symbol) {
  const data = prices.value.get(symbol)
  return data ? data.change24h : 0
}

function getHigh24h(symbol) {
  const data = prices.value.get(symbol)
  return data ? data.high24h : 0
}

function getLow24h(symbol) {
  const data = prices.value.get(symbol)
  return data ? data.low24h : 0
}

function getVolume24h(symbol) {
  const data = prices.value.get(symbol)
  return data ? data.volume24h : 0
}

// 图表数据 - 现在由CandlestickChart组件处理

// 切换行展开状态
function toggleRowExpansion(symbol) {
  if (expandedRows.value.has(symbol)) {
    expandedRows.value.delete(symbol)
  } else {
    expandedRows.value.add(symbol)
    // 首次展开时加载K线数据
    const timeframe = getSelectedTimeframeForSymbol(symbol)
    const cacheKey = `${symbol}_${timeframe}`
    if (!klineDataCache.has(cacheKey)) {
      loadKlineDataForSymbol(symbol)
    }
  }
}

// 为指定币种设置时间周期
function setTimeframeForSymbol(symbol, timeframe) {
  selectedTimeframes.value.set(symbol, timeframe)
  loadKlineDataForSymbol(symbol)
}

// 获取指定币种的选中时间周期
function getSelectedTimeframeForSymbol(symbol) {
  return selectedTimeframes.value.get(symbol) || '1h'
}

// 获取指定币种的K线数据
function getKlineDataForSymbol(symbol) {
  const timeframe = getSelectedTimeframeForSymbol(symbol)
  const cacheKey = `${symbol}_${timeframe}`
  const cachedData = klineDataCache.get(cacheKey)
  const data = cachedData ? cachedData.data : []


  return data
}

// 获取指定币种的K线加载状态
function getKlineLoadingForSymbol(symbol) {
  return klineLoadingMap.value.get(symbol) || false
}

// 为指定币种加载K线数据
async function loadKlineDataForSymbol(symbol) {
  const timeframe = getSelectedTimeframeForSymbol(symbol)
  const cacheKey = `${symbol}_${timeframe}`

  // 检查缓存
  if (klineDataCache.has(cacheKey)) {
    const cachedData = klineDataCache.get(cacheKey)
    const cacheTime = cachedData._cacheTime || 0
    const now = Date.now()

    // 缓存5分钟有效
    if (now - cacheTime < 5 * 60 * 1000) {
      // 数据已经在缓存中，直接返回
      return
    } else {
      // 清除过期缓存
      klineDataCache.delete(cacheKey)
    }
  }

  // 检查是否已经在加载中
  if (klineLoadingMap.value.get(symbol)) {
    return // 避免重复加载
  }

  klineLoadingMap.value.set(symbol, true)

  try {
    const response = await api.getKlines(symbol, timeframe, 200) // 请求更多数据点以确保能计算MA
    const data = response.data || []

    // 存入缓存
    klineDataCache.set(cacheKey, {
      data: data,
      _cacheTime: Date.now()
    })

    // 数据已在缓存中，无需额外设置

    // 限制缓存大小，防止内存泄漏
    if (klineDataCache.size > 100) {
      const firstKey = klineDataCache.keys().next().value
      klineDataCache.delete(firstKey)
    }

  } catch (err) {
    console.error(`加载 ${symbol} K线数据失败:`, err)

    // 即使失败也缓存空数据，避免重复请求
    klineDataCache.set(cacheKey, {
      data: [],
      _cacheTime: Date.now()
    })

  } finally {
    klineLoadingMap.value.set(symbol, false)
  }
}

// 刷新数据
async function refreshData() {
  await loadSymbols()
}

// 加载币种列表和初始数据
async function loadSymbols() {
  loading.value = true
  error.value = ''

  try {
    // 尝试从API获取包含市值信息的币种列表
    try {
      const response = await api.getSymbolsWithMarketCap({
        kind: selectedKind.value,
        limit: 50 // 获取足够的数据用于前端分页和筛选
      })
      console.log("abc:", response)

      if (response && response.symbols && response.symbols.length > 0) {
        // symbols.value 只存储symbol字符串，用于WebSocket订阅
        symbols.value = response.symbols.map(item => item.symbol)
        // 将市值数据存储到prices Map中，用于前端计算
        let marketCapCount = 0
        response.symbols.forEach(item => {
          console.log(`币种 ${item.symbol}: market_cap_usd = ${item.market_cap_usd} (类型: ${typeof item.market_cap_usd})`)

          // 检查市值数据是否有效（不为null、undefined，且为有效数字）
          const marketCap = item.market_cap_usd
          const isValidMarketCap = marketCap !== null &&
                                   marketCap !== undefined &&
                                   !isNaN(marketCap) &&
                                   marketCap >= 0

          if (isValidMarketCap) {
            // 如果已有价格数据，合并市值信息
            const existingData = prices.value.get(item.symbol) || {}
            prices.value.set(item.symbol, {
              ...existingData,
              marketCapUSD: marketCap,
              lastUpdated: Date.now()
            })
            marketCapCount++
          } else {
            console.warn(`币种 ${item.symbol} 的市值数据无效:`, marketCap)
          }
        })

        console.log(`从API加载了 ${symbols.value.length} 个 ${selectedKind.value} 币种，其中 ${marketCapCount} 个有市值信息`)
      } else {
        throw new Error('API返回数据无效')
      }
    } catch (apiErr) {
      console.warn('API获取带市值信息的币种列表失败:', apiErr.message)
      console.warn('提示: 如果这是首次运行，请先运行以下命令同步市值数据:')
      console.warn('go run cmd/coincap_sync/main.go -action=market-data')
      // 不设置默认列表，让页面显示无数据状态
    }

    console.log(`最终加载了 ${symbols.value.length} 个币种进行监控`)

    // 加载初始价格数据
    await loadInitialPrices()

    // 如果WebSocket已连接，发送正确的订阅消息
    if (ws && ws.readyState === WebSocket.OPEN && symbols.value.length > 0) {
      console.log('数据加载完成，发送最终的WebSocket订阅消息:', symbols.value)
      const subscription = {
        action: 'subscribe',
        symbols: symbols.value
      }
      ws.send(JSON.stringify(subscription))
    }

  } catch (err) {
    error.value = '加载数据失败'
    console.error('加载币种数据失败:', err)
    // 不设置默认列表，让页面显示错误状态

    handleError(err, '加载币种数据')
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

// 加载初始价格数据（仅在页面加载时使用）
async function loadInitialPrices() {
  try {
    // 使用批量API获取所有币种的价格数据
    const response = await api.getBatchCurrentPrices(symbols.value, selectedKind.value)
    const priceData = response.data || []

    // 转换为Map格式，设置初始数据
    const priceMap = new Map()

    priceData.forEach(item => {
      const price = parseFloat(item.price) || 0

      // 获取已有的市值等数据（如果存在）
      const existingData = prices.value.get(item.symbol) || {}

      // 为初始数据提供完整的统计信息结构
      // 保留已有的市值信息（如marketCapUSD）
      priceMap.set(item.symbol, {
        ...existingData, // 保留现有数据，包括市值信息
        symbol: item.symbol,
        price: price,

        // 24h涨跌统计（初始时使用模拟数据，WebSocket会更新为真实数据）
        change24h: Math.random() * 20 - 10,
        changeAmount24h: 0,

        // 价格区间统计
        open24h: price,
        high24h: price * (1 + Math.random() * 0.05),
        low24h: price * (1 - Math.random() * 0.05),

        // 成交统计
        volume24h: Math.random() * 1000000 + 100000,
        quoteVolume24h: 0,
        trades24h: 0,

        // 其他统计信息
        weightedAvgPrice: price,
        prevClosePrice: price,
        lastQty: 0,

        // 更新时间戳
        lastUpdate: Date.now()
      })
    })

    prices.value = priceMap

  } catch (err) {
    console.error('加载初始价格数据失败:', err)
    // 如果批量获取失败，回退到逐个获取
    await loadPricesFallback()
  }
}

// 加载价格数据（仅用于初始化，现在主要由WebSocket处理）
async function loadPrices() {
  // 现在价格更新主要通过WebSocket处理
  // 这个函数保留用于初始化或手动刷新
  await loadInitialPrices()
}

// 回退方法：逐个获取价格（当批量API失败时使用）
async function loadPricesFallback() {
  try {
    const promises = symbols.value.map(async (symbol) => {
      try {
        const response = await api.getCurrentPrice(symbol)
        const price = parseFloat(response.price) || 0

        return {
          symbol,
          price: price,

          // 24h涨跌统计（回退时使用模拟数据）
          change24h: Math.random() * 20 - 10,
          changeAmount24h: 0,

          // 价格区间统计
          open24h: price,
          high24h: price * (1 + Math.random() * 0.1),
          low24h: price * (1 - Math.random() * 0.1),

          // 成交统计
          volume24h: Math.random() * 1000000 + 100000,
          quoteVolume24h: 0,
          trades24h: 0,

          // 其他统计信息
          weightedAvgPrice: price,
          prevClosePrice: price,
          lastQty: 0,

          // 更新时间戳
          lastUpdate: Date.now()
        }
      } catch (err) {
        console.warn(`获取 ${symbol} 价格失败:`, err)
        return {
          symbol,
          price: 0,
          change24h: 0,
          changeAmount24h: 0,
          open24h: 0,
          high24h: 0,
          low24h: 0,
          volume24h: 0,
          quoteVolume24h: 0,
          trades24h: 0,
          weightedAvgPrice: 0,
          prevClosePrice: 0,
          lastQty: 0,
          lastUpdate: Date.now()
        }
      }
    })

    const results = await Promise.all(promises)
    prices.value = new Map(results.map(r => [r.symbol, r]))

  } catch (err) {
    console.error('回退价格获取也失败:', err)
  }
}

// 加载K线数据
async function loadKlineData() {
  if (!selectedSymbol.value) return

  klineLoading.value = true
  try {
    const response = await api.getKlines(selectedSymbol.value, selectedTimeframe.value, 100)
    klineData.value = response.data || []
  } catch (err) {
    console.error('加载K线数据失败:', err)
    klineData.value = []
  } finally {
    klineLoading.value = false
  }
}

// WebSocket连接管理
function connectWebSocket() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    console.log('WebSocket already connected')
    return
  }

  console.log('连接WebSocket实时数据...')

  try {
    ws = new WebSocket(WS_URL)

    ws.onopen = () => {
      console.log('WebSocket连接成功')
      wsConnected.value = true
      reconnectAttempts = 0

      // 发送订阅消息 - 使用当前有效的symbols
      if (symbols.value.length > 0) {
        console.log('WebSocket连接成功，发送订阅消息:', symbols.value)
        const subscription = {
          action: 'subscribe',
          symbols: symbols.value
        }
        ws.send(JSON.stringify(subscription))
      } else {
        console.log('WebSocket连接成功，但symbols.value为空，等待symbols加载完成后订阅')
      }
    }

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)

        if (message.type === 'subscription_confirmed') {
          console.log('WebSocket订阅确认:', message.message)
        } else if (message.type === 'price_update') {
          handlePriceUpdate(message)
        }
      } catch (err) {
        console.error('解析WebSocket消息失败:', err)
      }
    }

    ws.onclose = (event) => {
      console.log('WebSocket连接关闭:', event.code, event.reason)
      wsConnected.value = false
      ws = null

      // 如果是非正常关闭，尝试重连
      if (event.code !== 1000 && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
        scheduleReconnect()
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket连接错误:', error)
      wsConnected.value = false
      ws = null
    }

  } catch (err) {
    console.error('创建WebSocket连接失败:', err)
    scheduleReconnect()
  }
}

function disconnectWebSocket() {
  if (ws) {
    ws.close(1000, 'Component unmounting')
    ws = null
  }

  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }

  reconnectAttempts = 0
}

// 处理实时价格更新
// 价格更新队列 - 批量处理优化性能
const priceUpdateQueue = new Map()
let priceUpdateTimer = null

// 处理实时价格更新 - 优化版本
function handlePriceUpdate(update) {
  const { symbol, price, change_percent, change_amount,
    high_24h, low_24h, volume_24h, quote_volume_24h, trades_24h } = update

  // 检查是否在监控列表中
  if (!symbols.value.includes(symbol)) {
    return // 忽略不在监控列表中的币种
  }

  // 防抖处理 - 避免过于频繁的更新
  const now = Date.now()
  const lastUpdate = priceUpdateCache.get(symbol) || 0
  if (now - lastUpdate < PRICE_UPDATE_DEBOUNCE) {
    return
  }
  priceUpdateCache.set(symbol, now)

  // 添加到更新队列
  priceUpdateQueue.set(symbol, {
    symbol,
    price: parseFloat(price),
    change24h: parseFloat(change_percent),
    changeAmount24h: parseFloat(change_amount) || 0,
    high24h: parseFloat(high_24h),
    low24h: parseFloat(low_24h),
    volume24h: parseFloat(volume_24h),
    quoteVolume24h: parseFloat(quote_volume_24h) || 0,
    trades24h: trades_24h || 0,
    weightedAvgPrice: parseFloat(update.weighted_avg_price) || 0,
    prevClosePrice: parseFloat(update.prev_close_price) || 0,
    lastQty: parseFloat(update.last_qty) || 0,
    lastUpdate: now
  })

  // 延迟批量更新
  if (priceUpdateTimer) clearTimeout(priceUpdateTimer)
  priceUpdateTimer = setTimeout(() => {
    batchUpdatePrices()
  }, 50) // 50ms批量更新
}

// 检查币种是否有活跃的价格变化标记
function getPriceChangeIndicator(symbol) {
  const indicator = priceChangeIndicators.value.get(symbol)
  if (!indicator) return null

  // 检查是否过期
  const now = Date.now()
  if (now - indicator.timestamp > PRICE_CHANGE_DURATION) {
    return null
  }

  return indicator
}

// 批量更新价格 - 优化性能
function batchUpdatePrices() {
  const updates = Array.from(priceUpdateQueue.values())
  priceUpdateQueue.clear()

  if (updates.length === 0) return

  // 批量更新价格数据
  const newPrices = new Map(prices.value)
  const newChangeIndicators = new Map(priceChangeIndicators.value)

  updates.forEach(update => {
    const symbol = update.symbol
    const currentPriceData = newPrices.get(symbol)
    const previousPrice = currentPriceData ? currentPriceData.price : null
    const newPrice = update.price

    // 检测价格变化
    if (previousPrice !== null && previousPrice !== newPrice) {
      const changeType = newPrice > previousPrice ? 'up' : 'down'
      newChangeIndicators.set(symbol, {
        type: changeType,
        timestamp: Date.now()
      })

      // 自动清除标记
      setTimeout(() => {
        const currentIndicators = priceChangeIndicators.value
        if (currentIndicators.get(symbol)?.timestamp === newChangeIndicators.get(symbol)?.timestamp) {
          const updatedIndicators = new Map(currentIndicators)
          updatedIndicators.delete(symbol)
          priceChangeIndicators.value = updatedIndicators
        }
      }, PRICE_CHANGE_DURATION)
    }

    if (currentPriceData) {
      newPrices.set(symbol, {
        ...currentPriceData,
        ...update
      })
    }
  })

  // 一次性更新响应式数据
  prices.value = newPrices
  priceChangeIndicators.value = newChangeIndicators
  lastUpdateTime.value = Date.now()

  // 只在开发环境下记录日志
  // if (import.meta.env.DEV) {
  //   console.log(`批量更新了 ${updates.length} 个币种的价格数据`)
  // }
}

// 调度重连
function scheduleReconnect() {
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    console.error('WebSocket重连次数超过上限，停止重连')
    return
  }

  reconnectAttempts++
  const delay = RECONNECT_INTERVAL * reconnectAttempts // 递增延迟

  console.log(`WebSocket将在 ${delay/1000} 秒后重连 (尝试 ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`)

  reconnectTimer = setTimeout(() => {
    connectWebSocket()
  }, delay)
}

// 页面加载时初始化数据

// 监听symbols变化，更新虚拟滚动和WebSocket订阅
watch(() => symbols.value.length, () => {
  nextTick(() => {
    updateVirtualScroll()
  })

  // 当symbols变化时，如果WebSocket已连接，立即发送订阅消息
  if (symbols.value.length > 0 && ws && ws.readyState === WebSocket.OPEN) {
    console.log('symbols变化，重新发送WebSocket订阅消息:', symbols.value)
    const subscription = {
      action: 'subscribe',
      symbols: symbols.value
    }
    ws.send(JSON.stringify(subscription))
  }
})

// 显示数量变化处理方法已移除（使用分页替代）

// 定期刷新币种列表（每5分钟）
let symbolRefreshTimer = null
function startSymbolAutoRefresh() {
  if (symbolRefreshTimer) {
    clearInterval(symbolRefreshTimer)
  }

  // 默认开启自动刷新 - 静默刷新，不显示日志
  symbolRefreshTimer = setInterval(() => {
    // 静默刷新币种列表，避免频繁日志输出
    loadSymbols()
  }, 5 * 60 * 1000) // 5分钟
}



// 生命周期
onMounted(() => {
  loadSymbols()
  connectWebSocket()
  startSymbolAutoRefresh() // 启动币种列表自动刷新

  // 初始化虚拟滚动
  nextTick(() => {
    if (scrollContainer.value) {
      updateVirtualScroll()
    }
  })
})

onUnmounted(() => {
  disconnectWebSocket()

  // 清理定时器和缓存
  if (priceUpdateTimer) {
    clearTimeout(priceUpdateTimer)
  }

  // 清理缓存
  klineDataCache.clear()
  priceUpdateCache.clear()
})
</script>

<style scoped>
.page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px 14px 40px;
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

.filter-controls {
  margin-left: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.display-options {
  display: flex;
  align-items: center;
  gap: 12px;
}

.option-label {
  font-size: 14px;
  color: #666;
  font-weight: 500;
}

.limit-select {
  padding: 4px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: white;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  min-width: 70px;
}

.limit-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}


.controls {
  display: flex;
  gap: 8px;
}

.btn {
  height: 32px;
  padding: 0 12px;
  border: 1px solid rgba(0,0,0,.15);
  background: #fff;
  border-radius: 6px;
  cursor: pointer;
}

.btn:disabled {
  opacity: .6;
  cursor: not-allowed;
}

.content {
  width: 100%;
  padding: 20px 0;
}

/* 全局状态样式 */
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

.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  background: rgba(239, 68, 68, 0.05);
  border: 1px solid rgba(239, 68, 68, 0.1);
  border-radius: 12px;
  margin: 20px 0;
}

.error-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.error-title {
  font-size: 18px;
  font-weight: 600;
  color: #ef4444;
  margin-bottom: 8px;
}

.error-message {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 20px;
  max-width: 400px;
}

.retry-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
}

.retry-btn:hover {
  background: #dc2626;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3);
}

/* 无数据状态样式 */
.no-data-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 12px;
  margin: 20px 0;
}

.no-data-icon {
  font-size: 64px;
  margin-bottom: 20px;
  opacity: 0.6;
}

.no-data-title {
  font-size: 20px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 12px;
}

.no-data-message {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 24px;
  max-width: 400px;
  line-height: 1.5;
}

.refresh-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
}

.refresh-btn:hover {
  background: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.symbols-table-container {
  background: rgba(255,255,255,.02);
  border: 1px solid rgba(0,0,0,.06);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,.04);
}

.loading, .error {
  text-align: center;
  padding: 40px;
  color: #888;
}

.error {
  color: #ef4444;
}

/* 页面头部样式 */
.page-header {
  margin-bottom: 24px;
}

.header-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 16px;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 24px;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
  margin: 0;
}

.title-icon {
  font-size: 28px;
}

.market-status {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-indicator.online {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.status-indicator.offline {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.last-update {
  color: #6b7280;
  font-size: 12px;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.type-selector {
  display: flex;
  gap: 4px;
  background: rgba(0,0,0,.05);
  border-radius: 8px;
  padding: 2px;
}

.type-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #6b7280;
  transition: all 0.2s;
}

.type-btn:hover {
  background: rgba(0,0,0,.1);
  color: #374151;
}

.type-btn.active {
  background: #3b82f6;
  color: white;
}

.btn-icon {
  font-size: 14px;
}


.btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 16px;
  border: 1px solid rgba(0,0,0,.15);
  background: #fff;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
}

.btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,.1);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-primary {
  background: #3b82f6;
  color: white;
  border-color: #3b82f6;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
  border-color: #2563eb;
}

.btn-secondary {
  background: #f8fafc;
  color: #475569;
  border-color: #e2e8f0;
}

.btn-secondary:hover:not(:disabled) {
  background: #f1f5f9;
  border-color: #cbd5e1;
}

.btn.active {
  background: #1e40af;
  color: white;
}


/* 虚拟滚动容器 */
.virtual-scroll-container {
  height: 600px;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
}

.virtual-scroll-container::-webkit-scrollbar {
  width: 6px;
}

.virtual-scroll-container::-webkit-scrollbar-track {
  background: rgba(0,0,0,.05);
  border-radius: 3px;
}

.virtual-scroll-container::-webkit-scrollbar-thumb {
  background: rgba(0,0,0,.2);
  border-radius: 3px;
}

.virtual-scroll-container::-webkit-scrollbar-thumb:hover {
  background: rgba(0,0,0,.3);
}

/* 表格样式 */
.symbols-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0,0,0,.1);
}

/* 粘性表头 */
.sticky-header {
  position: sticky;
  top: 0;
  z-index: 10;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.symbols-table thead th {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  padding: 16px 12px;
  text-align: center; /* 表头默认居中对齐 */
  font-weight: 600;
  color: #374151;
  border-bottom: 2px solid #e2e8f0;
  white-space: nowrap;
  position: relative;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
}

.sortable {
  cursor: pointer;
  user-select: none;
  transition: background-color 0.2s;
}

.sortable:hover {
  background: rgba(59, 130, 246, 0.05);
}

.sort-icon {
  font-size: 12px;
  color: #9ca3af;
  margin-left: 4px;
  transition: color 0.2s;
}

.sortable:hover .sort-icon {
  color: #3b82f6;
}

.symbols-table tbody td {
  padding: 16px 12px;
  border-bottom: 1px solid #f1f5f9;
  vertical-align: middle;
  transition: background-color 0.2s;
  text-align: center; /* 默认居中对齐 */
}

.symbols-table tbody tr {
  transition: all 0.2s ease;
}

.symbols-table tbody tr:hover {
  background: linear-gradient(90deg, rgba(59,130,246,.02) 0%, rgba(59,130,246,.05) 100%);
  transform: translateX(2px);
}

.symbols-table tbody tr.expanded {
  background: linear-gradient(135deg, rgba(59,130,246,.08) 0%, rgba(59,130,246,.12) 100%);
  border-bottom: 2px solid #3b82f6;
  box-shadow: inset 0 2px 4px rgba(59, 130, 246, 0.1);
}

.symbols-table tbody tr.expanded:hover {
  background: linear-gradient(135deg, rgba(59,130,246,.1) 0%, rgba(59,130,246,.15) 100%);
}

/* 列宽设置 */
.col-symbol { width: 120px; text-align: center; }
.col-price { width: 120px; text-align: center; }
.col-change { width: 100px; text-align: center; }
.col-high { width: 100px; text-align: center; }
.col-low { width: 100px; text-align: center; }
.col-volume { width: 120px; text-align: center; }
.col-marketcap { width: 110px; text-align: center; }
.col-expand { width: 50px; text-align: center; }

/* 行内容样式 */
.symbol-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.symbol-name {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.symbol-full {
  font-size: 11px;
  color: #6b7280;
}

.price-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.price {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  transition: all 0.2s ease;
}

.price-indicator {
  position: absolute;
  right: -24px;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0;
  animation: priceChangeFlash 3s ease-out forwards;
}

.price-indicator.up .indicator-icon {
  color: #ef4444;
}

.price-indicator.down .indicator-icon {
  color: #10b981;
}

.indicator-icon {
  font-size: 18px;
  font-weight: bold;
  text-shadow: 0 0 6px rgba(0,0,0,0.4);
}

@keyframes priceChangeFlash {
  0% {
    opacity: 0;
    transform: translateY(-50%) scale(0.8);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scale(1.2);
  }
  20% {
    opacity: 1;
    transform: translateY(-50%) scale(1);
  }
  80% {
    opacity: 1;
    transform: translateY(-50%) scale(1);
  }
  100% {
    opacity: 0;
    transform: translateY(-50%) scale(0.9);
  }
}

.price-up {
  color: #22c55e;
  animation: priceChange 0.5s ease-out;
}

.price-down {
  color: #ef4444;
  animation: priceChange 0.5s ease-out;
}

@keyframes priceChange {
  0% {
    transform: scale(1.1);
    background: rgba(34, 197, 94, 0.1);
  }
  100% {
    transform: scale(1);
    background: transparent;
  }
}

.change {
  font-size: 13px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.change-up {
  color: #22c55e;
}

.change-down {
  color: #ef4444;
}

/* 成交量容器样式 */
.volume-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.volume-text {
  font-weight: 500;
  color: #374151;
}

/* 成交量进度条样式 */
.volume-bar {
  position: relative;
  width: 100%;
  height: 4px;
  background: rgba(0,0,0,.08);
  border-radius: 2px;
  overflow: hidden;
}

.volume-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #06b6d4);
  border-radius: 2px;
  transition: width 0.5s ease;
  box-shadow: 0 0 4px rgba(59, 130, 246, 0.3);
}

.expand-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.expand-icon span {
  font-size: 12px;
  color: #6b7280;
}

/* K线图行样式 */
.chart-row {
  background: rgba(0,0,0,.01);
  animation: slideDown 0.3s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.chart-cell {
  padding: 0;
  border-bottom: none;
  overflow: hidden;
}

.chart-container {
  padding: 24px;
  background: linear-gradient(135deg, #fefefe 0%, #f8fafc 100%);
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  margin: 12px 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,.05);
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: scale(0.98);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e2e8f0;
}

.chart-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
  display: flex;
  align-items: center;
  gap: 8px;
}

.chart-header h4::before {
  content: '📊';
  font-size: 20px;
}

.timeframe-selector {
  display: flex;
  gap: 6px;
  background: rgba(0,0,0,.02);
  border-radius: 8px;
  padding: 2px;
}

.tf-btn {
  padding: 6px 12px;
  border: 1px solid rgba(0,0,0,.1);
  background: white;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s;
  color: #6b7280;
}

.tf-btn:hover {
  background: rgba(59,130,246,.05);
  color: #3b82f6;
  border-color: rgba(59,130,246,.2);
}

.tf-btn.active {
  background: #3b82f6;
  color: white;
  border-color: #3b82f6;
  box-shadow: 0 2px 4px rgba(59,130,246,.2);
}

.chart-wrapper {
  height: 420px;
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  background: white;
  border: 1px solid #e2e8f0;
}

.chart-loading, .chart-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #888;
  font-size: 14px;
  gap: 12px;
}

.chart-loading::before {
  content: '';
  width: 24px;
  height: 24px;
  border: 2px solid #e2e8f0;
  border-top: 2px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.chart-empty::before {
  content: '📈';
  font-size: 32px;
  margin-bottom: 8px;
}

.chart-container {
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 分页样式 */
.pagination-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 20px;
  padding: 16px 0;
  background: rgba(255,255,255,.02);
  border-radius: 8px;
}

.page-btn {
  padding: 8px 16px;
  border: 1px solid rgba(59, 130, 246, 0.3);
  background: white;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #3b82f6;
  transition: all 0.2s;
  min-width: 60px;
}

.page-btn:hover:not(:disabled) {
  background: #3b82f6;
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.page-btn:disabled,
.page-btn.disabled {
  opacity: 0.4;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.page-info {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
  white-space: nowrap;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(0,0,0,.06);
}

.chart-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

.timeframe-selector {
  display: flex;
  gap: 4px;
}

.tf-btn {
  padding: 4px 8px;
  border: 1px solid rgba(0,0,0,.15);
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s;
}

.tf-btn:hover {
  background: rgba(0,0,0,.05);
}

.tf-btn.active {
  background: #3b82f6;
  color: white;
  border-color: #3b82f6;
}

.chart-wrapper {
  flex: 1;
  min-height: 400px;
}

.chart-loading, .chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #888;
}

.chart-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #888;
}

.placeholder-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.placeholder-text {
  font-size: 16px;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .header-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }


  .col-symbol { width: 100px; text-align: center; }
  .col-price { width: 100px; text-align: center; }
  .col-change { width: 80px; text-align: center; }
  .col-high, .col-low { width: 80px; text-align: center; }
  .col-volume { width: 100px; text-align: center; }
  .col-marketcap { width: 90px; text-align: center; }
}

@media (max-width: 768px) {
  .header-top {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-controls {
    justify-content: center;
  }

  .filter-controls {
    margin-left: 0;
    margin-top: 8px;
    align-items: flex-start;
  }

  .price-indicator {
    right: -20px;
  }

  .indicator-icon {
    font-size: 16px;
  }

  .display-options {
    flex-wrap: wrap;
    gap: 8px;
  }

  .limit-select {
    font-size: 13px;
    padding: 3px 6px;
    min-width: 60px;
  }

  .page-title {
    font-size: 20px;
  }

  .symbols-table {
    font-size: 12px;
  }

  .symbols-table thead th,
  .symbols-table tbody td {
    padding: 10px 6px;
  }

  .symbols-table thead th {
    text-align: center;
  }

  .col-symbol { width: 80px; text-align: center; }
  .col-price { width: 80px; text-align: center; }
  .col-change { width: 70px; text-align: center; }
  .col-high, .col-low { width: 70px; text-align: center; }
  .col-volume { width: 80px; text-align: center; }
  .col-marketcap { width: 75px; text-align: center; }

  .symbol-name {
    font-size: 14px;
  }

  .price {
    font-size: 14px;
  }


  .chart-container {
    padding: 16px;
    margin: 6px;
  }

  .chart-wrapper {
    height: 320px;
  }

  .chart-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .timeframe-selector {
    align-self: stretch;
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .page-header {
    margin-bottom: 16px;
  }

  .header-top {
    gap: 12px;
  }

  .market-status {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .btn {
    justify-content: center;
    width: 100%;
  }

  .col-symbol { width: 70px; text-align: center; }
  .col-price { width: 70px; text-align: center; }
  .col-change { width: 60px; text-align: center; }
  .col-high, .col-low { width: 60px; text-align: center; }
  .col-volume { width: 70px; text-align: center; }
  .col-marketcap { width: 65px; text-align: center; }

  .symbol-full {
    display: none;
  }

  .symbol-name {
    font-size: 13px;
  }

  .price {
    font-size: 13px;
  }

  .chart-wrapper {
    height: 280px;
  }

  .chart-container {
    padding: 12px;
    margin: 4px;
  }

}
</style>