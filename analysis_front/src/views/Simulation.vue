<template>
  <section class="panel">
    <div class="row">
      <h2>模拟交易</h2>
      <div class="spacer"></div>
      <button class="primary" @click="showCreateModal = true">新建交易</button>
      <button @click="load">刷新</button>
    </div>
  </section>

  <!-- 持仓统计 -->
  <section style="margin-top:12px;" class="panel" v-if="openTrades.length > 0">
    <h3>持仓统计</h3>
    <div class="stats-summary">
      <div class="summary-item">
        <span class="label">持仓数量：</span>
        <span class="value">{{ openTrades.length }}</span>
      </div>
      <div class="summary-item">
        <span class="label">总投入：</span>
        <span class="value">{{ formatUSD(totalInvested) }}</span>
      </div>
      <div class="summary-item">
        <span class="label">总盈亏：</span>
        <span class="value" :class="totalUnrealizedPnl >= 0 ? 'positive' : 'negative'">
          {{ formatUSD(totalUnrealizedPnl) }}
        </span>
      </div>
      <div class="summary-item">
        <span class="label">总收益率：</span>
        <span class="value" :class="totalReturnPercent >= 0 ? 'positive' : 'negative'">
          {{ formatPercent(totalReturnPercent) }}
        </span>
      </div>
    </div>
  </section>

  <!-- 持仓列表 -->
  <section style="margin-top:12px;" class="panel" v-if="openTrades.length > 0">
    <h3>当前持仓</h3>
    <table class="data-table">
      <thead>
        <tr>
          <th>币种</th>
          <th>数量</th>
          <th>买入价格</th>
          <th>当前价格</th>
          <th>总价值</th>
          <th>未实现盈亏</th>
          <th>收益率</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="trade in openTrades"
          :key="trade.id"
          class="trade-row"
          :class="{ 'selected-row': (trade.base_symbol || trade.symbol || '').toUpperCase() === chartSymbolSelection }"
          @click="selectTrade(trade)"
        >
          <td><strong>{{ trade.base_symbol }}</strong><br><small>{{ trade.symbol }}</small></td>
          <td>{{ formatQuantity(trade.quantity) }}</td>
          <td>{{ formatPrice(trade.price) }}</td>
          <td>{{ formatPrice(trade.current_price) }}</td>
          <td>{{ formatUSD(calculateCurrentValue(trade)) }}</td>
          <td :class="getPnlClass(trade.unrealized_pnl)">
            {{ formatUSD(parseFloat(trade.unrealized_pnl || 0)) }}
          </td>
          <td :class="getPnlClass(trade.unrealized_pnl)">
            {{ formatPercent(trade.unrealized_pnl_percent) }}
          </td>
          <td>
            <button class="btn-small danger" @click.stop="showCloseModal(trade)">平仓</button>
          </td>
        </tr>
      </tbody>
    </table>
  </section>

  <!-- K线图 -->
  <section style="margin-top:12px;" class="panel">
    <div class="row" style="align-items:flex-end; gap:12px;">
      <h3>K线图</h3>
      <div class="spacer"></div>
      <label style="margin-bottom:0;">
        币种：
        <input
          v-model="chartSymbolSelection"
          :list="'symbol-options'"
          placeholder="输入或选择币种"
          @input="onSymbolInput"
        />
        <datalist id="symbol-options">
          <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol" />
        </datalist>
      </label>
      <label style="margin-bottom:0;">
        时间周期：
        <select v-model="klineInterval">
          <option value="hourly">小时线</option>
          <option value="daily">日线</option>
        </select>
      </label>
      <label style="margin-bottom:0;">
        区间：
        <select v-model.number="klineDays">
          <option :value="7">7天</option>
          <option :value="30">30天</option>
          <option :value="90">90天</option>
          <option :value="180">180天</option>
        </select>
      </label>
      <label style="margin-bottom:0;">
        <input type="checkbox" v-model="showVolume" />
        显示成交量
      </label>
    </div>
    <div v-if="klineLoading" class="chart-placeholder">
      <p>加载中...</p>
    </div>
    <div v-else-if="chartErrorMessage" class="chart-error">
      {{ chartErrorMessage }}
    </div>
    <CandlestickChart
      v-else-if="klineData.length > 0"
      :kline-data="klineData"
      :buy-points="buyPoints"
      :sell-points="sellPoints"
      :title="chartTitle"
      :show-volume="showVolume"
      :volume-data="volumeData"
    />
    <div v-else class="chart-placeholder">
      <p>{{ chartPlaceholder }}</p>
    </div>
  </section>

  <!-- 历史交易 -->
  <section style="margin-top:12px;" class="panel">
    <h3>历史交易</h3>
    <div class="row" style="margin-bottom: 16px;">
      <label>
        <input type="checkbox" v-model="showClosed" @change="load" />
        显示已平仓
      </label>
    </div>
    <div v-if="loading" style="text-align:center; padding: 40px;">
      <p>加载中...</p>
    </div>
    <div v-else-if="closedTrades.length === 0" style="text-align:center; padding: 40px;">
      <p>暂无历史交易</p>
    </div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>币种</th>
          <th>数量</th>
          <th>买入价格</th>
          <th>卖出价格</th>
          <th>已实现盈亏</th>
          <th>收益率</th>
          <th>卖出时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="trade in closedTrades" :key="trade.id">
          <td><strong>{{ trade.base_symbol }}</strong><br><small>{{ trade.symbol }}</small></td>
          <td>{{ formatQuantity(trade.quantity) }}</td>
          <td>{{ formatPrice(trade.price) }}</td>
          <td>{{ formatPrice(trade.current_price) }}</td>
          <td :class="getPnlClass(trade.realized_pnl)">
            {{ formatUSD(parseFloat(trade.realized_pnl || 0)) }}
          </td>
          <td :class="getPnlClass(trade.realized_pnl)">
            {{ formatPercent(trade.realized_pnl_percent) }}
          </td>
          <td>{{ formatTime(trade.sold_at) }}</td>
        </tr>
      </tbody>
    </table>
  </section>

  <!-- 创建交易模态框 -->
  <div v-if="showCreateModal" class="modal-overlay" @click="showCreateModal = false">
    <div class="modal-content" @click.stop>
      <h3>新建模拟交易</h3>
      <form @submit.prevent="createTrade">
        <div class="form-group">
          <label>币种符号 *</label>
          <input v-model="newTrade.symbol" type="text" placeholder="BTCUSDT" required />
        </div>
        <div class="form-group">
          <label>基础币种 *</label>
          <input v-model="newTrade.base_symbol" type="text" placeholder="BTC" required />
        </div>
        <div class="form-group">
          <label>类型</label>
          <select v-model="newTrade.kind">
            <option value="spot">现货</option>
            <option value="futures">期货</option>
          </select>
        </div>
        <div class="form-group">
          <label>数量 *</label>
          <input v-model="newTrade.quantity" type="number" step="0.00000001" placeholder="0.001" required />
        </div>
        <div class="form-group">
          <label>买入价格 *</label>
          <input v-model="newTrade.price" type="number" step="0.01" placeholder="42000" required />
        </div>
        <div class="form-actions">
          <button type="button" @click="showCreateModal = false">取消</button>
          <button type="submit" class="primary">创建</button>
        </div>
      </form>
    </div>
  </div>

  <!-- 平仓模态框 -->
  <div v-if="closeTrade" class="modal-overlay" @click="closeTrade = null">
    <div class="modal-content" @click.stop>
      <h3>平仓交易</h3>
      <div class="trade-info">
        <p><strong>币种：</strong>{{ closeTrade.base_symbol }} ({{ closeTrade.symbol }})</p>
        <p><strong>数量：</strong>{{ formatQuantity(closeTrade.quantity) }}</p>
        <p><strong>买入价格：</strong>{{ formatPrice(closeTrade.price) }}</p>
        <p><strong>当前价格：</strong>{{ formatPrice(closeTrade.current_price) }}</p>
      </div>
      <form @submit.prevent="closeTradeConfirm">
        <div class="form-group">
          <label>卖出价格 *</label>
          <input v-model="closePrice" type="number" step="0.01" :placeholder="closeTrade.current_price" required />
        </div>
        <div class="form-actions">
          <button type="button" @click="closeTrade = null">取消</button>
          <button type="submit" class="primary">确认平仓</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { api } from '../api/api.js'
import CandlestickChart from '../components/CandlestickChart.vue'

const loading = ref(false)
const showCreateModal = ref(false)
const closeTrade = ref(null)
const closePrice = ref('')
const showClosed = ref(true)

const newTrade = ref({
  symbol: '',
  base_symbol: '',
  kind: 'spot',
  quantity: '',
  price: ''
})

const allTrades = ref([])

const openTrades = computed(() => {
  return allTrades.value.filter(t => t.is_open)
})

const closedTrades = computed(() => {
  if (!showClosed.value) return []
  return allTrades.value.filter(t => !t.is_open)
})

const totalInvested = computed(() => {
  return openTrades.value.reduce((sum, t) => {
    return sum + parseFloat(t.total_value || 0)
  }, 0)
})

const totalUnrealizedPnl = computed(() => {
  return openTrades.value.reduce((sum, t) => {
    return sum + parseFloat(t.unrealized_pnl || 0)
  }, 0)
})

const totalReturnPercent = computed(() => {
  if (totalInvested.value === 0) return 0
  return (totalUnrealizedPnl.value / totalInvested.value) * 100
})

const klineDays = ref(30)
const klineInterval = ref('daily')
const chartSymbolSelection = ref('')
const klineData = ref([])
const volumeData = ref([])
const klineLoading = ref(false)
const chartErrorMessage = ref('')
const showVolume = ref(false)
const MIN_SYMBOL_LENGTH = 2

const availableSymbols = computed(() => {
  const map = new Map()
  openTrades.value.forEach(trade => {
    const symbol = (trade.base_symbol || trade.symbol || '').toUpperCase()
    if (!symbol) return
    if (!map.has(symbol)) {
      map.set(symbol, symbol)
    }
  })
  return Array.from(map.values())
})

const chartTitle = computed(() => {
  const symbol = chartSymbolSelection.value
  return symbol ? `${symbol} 价格趋势` : 'K线图'
})

const chartPlaceholder = computed(() => {
  return chartSymbolSelection.value ? '当前币种暂无K线数据' : '请选择持仓查看K线'
})

// 计算买入点和卖出点
const buyPoints = computed(() => {
  if (!chartSymbolSelection.value) return []
  const symbol = chartSymbolSelection.value.toUpperCase()
  return allTrades.value
    .filter(trade => {
      const tradeSymbol = (trade.base_symbol || trade.symbol || '').toUpperCase()
      return tradeSymbol === symbol && trade.is_open
    })
    .map(trade => ({
      timestamp: new Date(trade.created_at).getTime(),
      price: parseFloat(trade.price),
      label: `买入 ${trade.quantity}`
    }))
})

const sellPoints = computed(() => {
  if (!chartSymbolSelection.value) return []
  const symbol = chartSymbolSelection.value.toUpperCase()
  return allTrades.value
    .filter(trade => {
      const tradeSymbol = (trade.base_symbol || trade.symbol || '').toUpperCase()
      return tradeSymbol === symbol && !trade.is_open && trade.sold_at
    })
    .map(trade => ({
      timestamp: new Date(trade.sold_at).getTime(),
      price: parseFloat(trade.current_price || trade.price),
      label: `卖出 ${trade.quantity}`
    }))
})

function formatTime(timeStr) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

async function loadKlineForSymbol(symbol) {
  if (!symbol) {
    klineData.value = []
    volumeData.value = []
    chartErrorMessage.value = ''
    return
  }

  if (symbol.length < MIN_SYMBOL_LENGTH) {
    klineData.value = []
    volumeData.value = []
    chartErrorMessage.value = `请输入至少 ${MIN_SYMBOL_LENGTH} 个字符再加载`
    return
  }

  chartErrorMessage.value = ''
  klineLoading.value = true
  try {
    const res = await api.getMarketPriceHistory({ 
      symbol, 
      days: klineDays.value,
      interval: klineInterval.value 
    })
    const prices = Array.isArray(res?.prices) ? res.prices : []
    const volumes = Array.isArray(res?.total_volumes) ? res.total_volumes : []
    
    // 转换价格数据为K线格式 [timestamp, price]
    klineData.value = prices.map(item => [item[0], item[1]])
    
    // 转换成交量数据
    volumeData.value = volumes.map(item => [item[0], item[1]])
    
    if (!prices.length) {
      chartErrorMessage.value = '找不到历史数据'
    }
  } catch (error) {
    console.error('加载K线失败:', error)
    chartErrorMessage.value = error?.message || 'K线加载失败，请检查币种是否受支持'
    klineData.value = []
    volumeData.value = []
  } finally {
    klineLoading.value = false
  }
}

async function ensureSelectedTrade(reload = true) {
  if (openTrades.value.length === 0) {
    chartSymbolSelection.value = ''
    klineData.value = []
    volumeData.value = []
    return
  }

  if (!chartSymbolSelection.value || !availableSymbols.value.includes(chartSymbolSelection.value)) {
    chartSymbolSelection.value = availableSymbols.value[0]
  }
}

function selectTrade(trade) {
  if (!trade) return
  const symbol = (trade.base_symbol || trade.symbol || '').toUpperCase()
  if (!symbol) return
  chartSymbolSelection.value = symbol
}

function formatPrice(price) {
  if (!price) return '-'
  const p = parseFloat(price)
  return p.toFixed(8)
}

function formatQuantity(qty) {
  if (!qty) return '-'
  const q = parseFloat(qty)
  return q.toFixed(8)
}

function formatUSD(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1e9) {
    return (value / 1e9).toFixed(2) + 'B'
  } else if (value >= 1e6) {
    return (value / 1e6).toFixed(2) + 'M'
  } else if (value >= 1e3) {
    return (value / 1e3).toFixed(2) + 'K'
  }
  return value.toFixed(2)
}

function formatPercent(value) {
  if (value === null || value === undefined) return '-'
  return (value >= 0 ? '+' : '') + value.toFixed(2) + '%'
}

function getPnlClass(pnl) {
  if (!pnl) return ''
  const val = parseFloat(pnl)
  return val >= 0 ? 'positive' : 'negative'
}

function calculateCurrentValue(trade) {
  if (!trade.current_price) return 0
  const price = parseFloat(trade.current_price)
  const qty = parseFloat(trade.quantity)
  return price * qty
}

function onSymbolInput(event) {
  chartSymbolSelection.value = (event.target?.value || '').toUpperCase()
}

watch([klineDays, klineInterval], () => {
  if (chartSymbolSelection.value && chartSymbolSelection.value.length >= MIN_SYMBOL_LENGTH) {
    loadKlineForSymbol(chartSymbolSelection.value)
  }
})

watch(chartSymbolSelection, (symbol) => {
  if (!symbol) {
    klineData.value = []
    volumeData.value = []
    chartErrorMessage.value = ''
    return
  }

  loadKlineForSymbol(symbol)
})

async function load() {
  loading.value = true
  try {
    const res = await api.getSimulatedTrades({ is_open: null })
    allTrades.value = res.trades || []
    await ensureSelectedTrade()
  } catch (error) {
    console.error('加载模拟交易失败:', error)
    alert('加载失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

async function createTrade() {
  try {
    await api.createSimulatedTrade(newTrade.value)
    showCreateModal.value = false
    newTrade.value = { symbol: '', base_symbol: '', kind: 'spot', quantity: '', price: '' }
    await load()
  } catch (error) {
    console.error('创建交易失败:', error)
    alert('创建失败: ' + (error.message || '未知错误'))
  }
}

function showCloseModal(trade) {
  closeTrade.value = trade
  closePrice.value = trade.current_price || trade.price
}

async function closeTradeConfirm() {
  if (!closeTrade.value) return
  try {
    await api.closeSimulatedTrade(closeTrade.value.id, { price: closePrice.value })
    closeTrade.value = null
    closePrice.value = ''
    await load()
  } catch (error) {
    console.error('平仓失败:', error)
    alert('平仓失败: ' + (error.message || '未知错误'))
  }
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.stats-summary {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  margin-top: 16px;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.summary-item .label {
  color: #666;
}

.summary-item .value {
  font-weight: bold;
  font-size: 18px;
}

.summary-item .value.positive {
  color: #10b981;
}

.summary-item .value.negative {
  color: #ef4444;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 16px;
}

.data-table th,
.data-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #e0e0e0;
}

.data-table th {
  background: #f8f9fa;
  font-weight: bold;
}

.positive {
  color: #10b981;
  font-weight: bold;
}

.negative {
  color: #ef4444;
  font-weight: bold;
}

.btn-small {
  padding: 4px 12px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid #ddd;
  background: #fff;
  cursor: pointer;
}

.btn-small.danger {
  color: #ef4444;
  border-color: #ef4444;
}

.btn-small.danger:hover {
  background: #fee2e2;
}

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
}

.modal-content {
  background: #fff;
  padding: 24px;
  border-radius: 8px;
  min-width: 400px;
  max-width: 90%;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: bold;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.form-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
}

.trade-info {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}

.trade-info p {
  margin: 8px 0;
}

.chart-error {
  text-align: center;
  padding: 24px;
  color: #ef4444;
  font-weight: bold;
}

.trade-row {
  cursor: pointer;
  transition: background 0.2s;
}

.trade-row:hover {
  background: #f5f7fb;
}

.selected-row {
  background: #eef2ff;
}

.chart-placeholder {
  text-align: center;
  padding: 40px;
  color: #666;
}
</style>

