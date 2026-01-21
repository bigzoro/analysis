<!-- src/views/BinanceGainers.vue -->
<template>
  <div class="page">
    <header class="page-header">
      <div class="header-top">
        <div class="selectors-group">
          <div class="type-selector">
            <button
              :class="['type-btn', { active: selectedKind === 'spot' }]"
              @click="selectedKind = 'spot'"
            >
              现货
            </button>
            <button
              :class="['type-btn', { active: selectedKind === 'futures' }]"
              @click="selectedKind = 'futures'"
            >
              合约
            </button>
          </div>
          <div class="category-selector">
            <select v-model="selectedCategory" class="category-select" @change="handleCategoryChange">
              <option v-for="option in categoryOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>
        </div>
        <a href="https://cointt.com" target="_blank" rel="noopener noreferrer" class="invite-link">
          邀请链接
        </a>
      </div>
      <div class="header-row">
        <div class="controls">
          <label>日期：</label>
          <input type="date" v-model="date" class="select" @change="load" />
          <button class="btn" @click="load" :disabled="loading">
            {{ loading ? '加载中...' : '刷新' }}
          </button>
          <button class="btn btn-secondary" @click="showBlacklistDialog = true">
            管理黑名单
          </button>
        </div>
        <div class="quick-dates">
          <span class="quick-label">快速选择：</span>
          <button
            v-for="quick in quickDates"
            :key="quick.value"
            class="quick-btn"
            :class="{ active: date === quick.value }"
            @click="selectDate(quick.value)"
          >
            {{ quick.label }}
          </button>
        </div>
      </div>
    </header>

    <!-- 黑名单管理对话框 -->
    <div v-if="showBlacklistDialog" class="dialog-overlay" @click.self="showBlacklistDialog = false">
      <div class="dialog">
        <div class="dialog-header">
          <h3>币种黑名单管理</h3>
          <button class="btn-close" @click="showBlacklistDialog = false">×</button>
        </div>
        <div class="dialog-body">
          <div class="blacklist-tabs">
            <button
              class="tab-btn"
              :class="{ active: blacklistKind === 'spot' }"
              @click="switchBlacklistKind('spot')"
            >
              现货
            </button>
            <button
              class="tab-btn"
              :class="{ active: blacklistKind === 'futures' }"
              @click="switchBlacklistKind('futures')"
            >
              期货
            </button>
          </div>
          <div class="blacklist-add">
            <input
              v-model="newSymbol"
              type="text"
              :placeholder="blacklistKind === 'spot' ? '输入币种符号，如 BTCUSDT' : '输入币种符号，如 BTCUSD_PERP'"
              class="input"
              @keyup.enter="addBlacklist"
            />
            <button class="btn" @click="addBlacklist" :disabled="!newSymbol || adding">
              {{ adding ? '添加中...' : '添加' }}
            </button>
          </div>
          <div v-if="blacklistLoading" class="loading-small">加载中...</div>
          <div v-else class="blacklist-list">
            <div v-if="blacklist.length === 0" class="empty-text">暂无黑名单</div>
            <div v-else class="blacklist-items">
              <div v-for="item in blacklist" :key="item.id" class="blacklist-item">
                <span class="symbol">{{ item.symbol }}</span>
                <span class="kind-tag" :class="item.kind === 'spot' ? 'kind-spot' : 'kind-fut'">
                  {{ item.kind === 'spot' ? '现货' : '期货' }}
                </span>
                <button class="btn-delete" @click="deleteBlacklist(item.kind, item.symbol)" :disabled="deleting">
                  {{ deleting ? '删除中...' : '删除' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <section v-if="initialLoading || loading" class="loading">正在获取数据...</section>

    <section v-else>
      <div v-for="row in rows" :key="row.key" class="grid4">
        <div
            v-for="cell in row.cells"
            :key="cell.key"
            class="card"
        >
          <div class="card-head">
            <div>
              <div class="bucket">{{ cell.slot.label }}</div>
              <div class="fetched" v-if="cell.group">拉取时间：{{ fmtDate(cell.group.fetched_at) }}</div>
              <div class="fetched" v-else>暂无数据</div>
            </div>
            <div class="tag" :class="cell.kind === 'spot' ? 'tag-spot' : 'tag-fut'">
              {{ cell.kind === 'spot' ? '现货' : '合约' }}
            </div>
          </div>

          <div class="tbl-wrap" v-if="cell.group && cell.group.items && cell.group.items.length">
            <table class="tbl">
              <thead>
              <tr>
                <th class="col-rank">#</th>
                <th class="col-symbol">币种</th>
                <th class="col-num">涨幅</th>
                <th class="col-num">最新价</th>
              </tr>
              </thead>
              <tbody>
              <template v-for="item in cell.group.items" :key="item.symbol">
                <tr :class="getHighlightClass(cell.changedSymbols, item.symbol)">
                  <td class="col-rank">{{ item.rank }}</td>
                  <td class="col-symbol">
                    <a
                      v-if="isMajorPair(item.symbol)"
                      :href="getBinanceUrl(item.symbol, selectedKind)"
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
                  <td
                      class="col-num"
                      :class="item.pct_change >= 0 ? 'up' : 'down'"
                      :title="formatPctFull(item.pct_change)"
                  >
                    {{ formatPct(item.pct_change) }}
                  </td>
                  <td class="col-num" :title="item.last_price">
                    {{ formatPrice(item.last_price) }}
                  </td>
                </tr>
                <tr class="meta-row">
                  <td colspan="4">
                    <span class="muted">流通：{{ fmtUSD(item.market_cap_usd) }}</span>
                    <span class="mid-dot">·</span>
                    <span class="muted">全部：{{ fmtUSD(item.fdv_usd) }}</span>
                  </td>
                </tr>
              </template>
              </tbody>
            </table>
          </div>

          <div v-else class="empty">这一时间段没有数据</div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, watchEffect } from 'vue'
import { api } from '../api/api.js'
import { handleError, handleSuccess } from '../utils/errorHandler.js'

const date = ref(new Date().toISOString().slice(0, 10))
const selectedKind = ref('spot') // 'spot' 或 'futures'
const selectedCategory = ref('trading') // 分类选择器
const loading = ref(false)
const initialLoading = ref(true) // 页面初始化加载状态

// 币种分类选项
const categoryOptions = [
  { value: 'trading', label: '正常交易', icon: '✅', status: 'TRADING' },
  { value: 'break', label: '暂停交易', icon: '⏸️', status: 'BREAK' },
  { value: 'major', label: '主流币种', icon: '⭐', assets: ['BTC', 'ETH', 'BNB', 'ADA', 'SOL', 'DOT', 'AVAX', 'MATIC'] },
  { value: 'stable', label: '稳定币对', icon: '🛡️', assets: ['USDT', 'USDC', 'BUSD', 'DAI', 'TUSD', 'USDP'] },
  { value: 'defi', label: 'DeFi代币', icon: '🔗', assets: ['UNI', 'AAVE', 'SUSHI', 'COMP', 'MKR', 'SNX', 'CRV'] },
  { value: 'layer1', label: 'Layer1公链', icon: '⛓️', assets: ['ATOM', 'NEAR', 'FTM', 'ONE', 'EGLD', 'FLOW'] },
  { value: 'meme', label: 'Meme币', icon: '🐕', assets: ['SHIB', 'DOGE', 'PEPE', 'BONK', 'WIF', 'TURBO'] },
  { value: 'spot_only', label: '纯现货', icon: '💰', permissions: ['SPOT'] },
  { value: 'margin', label: '杠杆交易', icon: '📈', permissions: ['MARGIN'] },
  { value: 'leveraged', label: '合约交易', icon: '⚡', permissions: ['LEVERAGED'] },
  { value: 'all', label: '全部币种', icon: '📊' }
]

const quickDates = computed(() => {
  const today = new Date()
  // 生成从10天前到今天的日期，时间越靠前的在左边，时间越靠后的在右边
  return Array.from({ length: 10 }, (_, i) => {
    const d = new Date(today)
    d.setDate(today.getDate() - (9 - i)) // 从10天前开始，到今天结束
    const value = d.toISOString().slice(0, 10)
    return {
      value,
      label: `${d.getMonth() + 1}/${pad2(d.getDate())}`,
    }
  })
})

const groupsSpot = ref([])     // 现货数据
const groupsFut  = ref([])     // 合约数据
const browserTZ  = Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Taipei'

// 黑名单管理
const showBlacklistDialog = ref(false)
const blacklistKind = ref('spot') // 当前查看的黑名单类型
const blacklist = ref([])
const blacklistLoading = ref(false)
const newSymbol = ref('')
const adding = ref(false)
const deleting = ref(false)

function fmtDate (s) {
  if (!s) return '-'
  return new Date(s).toLocaleString()
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

function fmtUSD (v) {
  const n = Number(v)
  if (!isFinite(n) || n <= 0) return '—'
  const abs = Math.abs(n)
  const fmt = (x, unit = '') => '$' + (Number.isInteger(x) ? x.toFixed(0) : x.toFixed(2)) + unit
  if (abs >= 1e12) return fmt(n / 1e12, 'T')
  if (abs >= 1e9)  return fmt(n / 1e9,  'B')
  if (abs >= 1e6)  return fmt(n / 1e6,  'M')
  if (abs >= 1e3)  return fmt(n / 1e3,  'K')
  return fmt(n)
}

// 获取高亮CSS类
function getHighlightClass(changedSymbols, symbol) {
  if (changedSymbols.has(symbol)) {
    const direction = changedSymbols.get(symbol)
    return `highlight-${direction}`
  }
  return ''
}

// --- 槽位划分（本地时区，每 1 小时一段）
function pad2 (n) { return String(n).padStart(2, '0') }
function bucketToLocalSlotKey (bucketISO) {
  const d = new Date(bucketISO)
  const y = d.getFullYear(), m = d.getMonth(), dd = d.getDate(), h = d.getHours()
  const slotStartH = Math.floor(h / 1) * 1
  const localStart = new Date(y, m, dd, slotStartH, 0, 0, 0)
  return localStart.getTime()
}
const daySlots = computed(() => {
  const base = new Date(date.value + 'T00:00:00')
  return Array.from({ length: 24 }, (_, i) => {
    const start = new Date(base.getFullYear(), base.getMonth(), base.getDate(), i * 1, 0, 0, 0)
    const end   = new Date(start.getTime() + 1 * 60 * 60 * 1000)
    return {
      key: start.getTime(),
      start, end,
      label: `${pad2(start.getHours())}:00 - ${pad2(end.getHours())}:00`,
    }
  })
})

// 把返回的组按“本地槽位起始时刻”映射
function mapBySlot (list) {
  const m = new Map()
  for (const g of list) {
    const k = bucketToLocalSlotKey(g.bucket)
    m.set(k, g)
  }
  return m
}

// 分析前后时间段变化的币种
const changedSymbols = computed(() => {
  const mapSpot = mapBySlot(groupsSpot.value || [])
  const mapFut  = mapBySlot(groupsFut.value || [])
  const dataMap = selectedKind.value === 'spot' ? mapSpot : mapFut
  const changes = new Map()

  const sortedSlots = daySlots.value.map(slot => slot.key).sort((a, b) => a - b)

  // 对于每个时间段，找出相对于前一个时间段有变化的币种
  for (let i = 1; i < sortedSlots.length; i++) {
    const currentSlot = sortedSlots[i]
    const prevSlot = sortedSlots[i - 1]

    const currentGroup = dataMap.get(currentSlot)
    const prevGroup = dataMap.get(prevSlot)

    if (!currentGroup || !prevGroup) continue

    const prevSymbols = new Map(prevGroup.items.map(item => [item.symbol, item]))
    const slotChanges = new Map() // 改为 Map，存储 symbol -> direction

    currentGroup.items.forEach((item, currentRank) => {
      const prevItem = prevSymbols.get(item.symbol)

      if (prevItem) {
        // 币种在前后时间段都存在，检查涨幅是否有变化
        const pctChangeDiff = Math.abs(item.pct_change - prevItem.pct_change)

        // 只要涨幅有任何变化就高亮，并记录变化方向
        if (pctChangeDiff > 0) {
          const direction = item.pct_change >= prevItem.pct_change ? 'up' : 'down'
          slotChanges.set(item.symbol, direction)
        }
      } else {
        // 新出现的币种，认为是上涨（新出现通常是上涨）
        slotChanges.set(item.symbol, 'up')
      }
    })

    if (slotChanges.size > 0) {
      changes.set(currentSlot, slotChanges)
    }
  }

  return changes
})

// 筛选功能现在由后端处理，前端不再需要筛选逻辑

// 根据选择的类型显示对应的数据，每行4个卡片
const rows = computed(() => {
  const mapSpot = mapBySlot(groupsSpot.value || [])
  const mapFut  = mapBySlot(groupsFut.value || [])
  const out = []

  // 根据选择的类型决定使用哪个数据映射
  const dataMap = selectedKind.value === 'spot' ? mapSpot : mapFut
  const kind = selectedKind.value

  // 每行显示4个时间段的卡片（四个相邻的时间段）
  for (let i = 0; i < daySlots.value.length; i += 4) {
    const s0 = daySlots.value[i]
    const s1 = daySlots.value[i + 1]
    const s2 = daySlots.value[i + 2]
    const s3 = daySlots.value[i + 3]

    const cells = []
    // 时间段 i
    if (s0) {
      const group = dataMap.get(s0.key) || null
      // 重新分配rank序号，确保显示为1,2,3,4...
      if (group && group.items) {
        group.items.forEach((item, index) => {
          item.rank = index + 1
        })
      }
      cells.push({
        key: `${kind}-${s0.key}`,
        kind: kind,
        slot: s0,
        group: group,
        changedSymbols: changedSymbols.value.get(s0.key) || new Set(),
      })
    }
    // 时间段 i+1
    if (s1) {
      const group = dataMap.get(s1.key) || null
      // 重新分配rank序号，确保显示为1,2,3,4...
      if (group && group.items) {
        group.items.forEach((item, index) => {
          item.rank = index + 1
        })
      }
      cells.push({
        key: `${kind}-${s1.key}`,
        kind: kind,
        slot: s1,
        group: group,
        changedSymbols: changedSymbols.value.get(s1.key) || new Set(),
      })
    }
    // 时间段 i+2
    if (s2) {
      const group = dataMap.get(s2.key) || null
      // 重新分配rank序号，确保显示为1,2,3,4...
      if (group && group.items) {
        group.items.forEach((item, index) => {
          item.rank = index + 1
        })
      }
      cells.push({
        key: `${kind}-${s2.key}`,
        kind: kind,
        slot: s2,
        group: group,
        changedSymbols: changedSymbols.value.get(s2.key) || new Set(),
      })
    }
    // 时间段 i+3
    if (s3) {
      const group = dataMap.get(s3.key) || null
      // 重新分配rank序号，确保显示为1,2,3,4...
      if (group && group.items) {
        group.items.forEach((item, index) => {
          item.rank = index + 1
        })
      }
      cells.push({
        key: `${kind}-${s3.key}`,
        kind: kind,
        slot: s3,
        group: group,
        changedSymbols: changedSymbols.value.get(s3.key) || new Set(),
      })
    }

    if (cells.length > 0) {
      out.push({ key: `row-${i}`, cells })
    }
  }
  return out
})

function selectDate (value) {
  if (date.value === value) return
  date.value = value
  load()
}

// 处理分类选择器变化
function handleCategoryChange() {
  // 分类变化时需要重新加载数据，因为后端会根据分类进行筛选
  if (!loading.value) {
    load()
  }
}

// 初始化加载两种类型的数据
async function loadInitial () {
  initialLoading.value = true
  loading.value = true
  try {
    const [spot, fut] = await Promise.all([
      api.binanceTop({ kind: 'spot',    interval: 60, date: date.value, tz: browserTZ, category: selectedCategory.value }),
      api.binanceTop({ kind: 'futures', interval: 60, date: date.value, tz: browserTZ, category: selectedCategory.value }),
    ])
    groupsSpot.value = Array.isArray(spot.data) ? spot.data : []
    groupsFut.value  = Array.isArray(fut.data)  ? fut.data  : []
  } catch (err) {
    handleError(err, '加载数据', { showToast: false }) // 加载失败不显示 Toast，避免干扰
    groupsSpot.value = []
    groupsFut.value  = []
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

// 手动刷新数据
async function load () {
  loading.value = true
  try {
    const [spot, fut] = await Promise.all([
      api.binanceTop({ kind: 'spot',    interval: 60, date: date.value, tz: browserTZ, category: selectedCategory.value }),
      api.binanceTop({ kind: 'futures', interval: 60, date: date.value, tz: browserTZ, category: selectedCategory.value }),
    ])
    groupsSpot.value = Array.isArray(spot.data) ? spot.data : []
    groupsFut.value  = Array.isArray(fut.data)  ? fut.data  : []
  } catch (err) {
    handleError(err, '加载数据', { showToast: false }) // 加载失败不显示 Toast，避免干扰
    groupsSpot.value = []
    groupsFut.value  = []
  } finally {
    loading.value = false
  }
}

// 切换黑名单类型
function switchBlacklistKind (kind) {
  blacklistKind.value = kind
  loadBlacklist()
}

// 加载黑名单
async function loadBlacklist () {
  blacklistLoading.value = true
  try {
    const res = await api.listBinanceBlacklist({ kind: blacklistKind.value })
    blacklist.value = Array.isArray(res.data) ? res.data : []
  } catch (err) {
    handleError(err, '加载黑名单', { showToast: false })
    blacklist.value = []
  } finally {
    blacklistLoading.value = false
  }
}

// 添加黑名单
async function addBlacklist () {
  const symbol = newSymbol.value.trim().toUpperCase()
  if (!symbol) return
  adding.value = true
  try {
    await api.addBinanceBlacklist({ kind: blacklistKind.value, symbol })
    newSymbol.value = ''
    await loadBlacklist()
    // 重新加载页面数据，使黑名单过滤立即生效
    await load()
    handleSuccess('黑名单添加成功')
  } catch (err) {
    handleError(err, '添加黑名单')
  } finally {
    adding.value = false
  }
}

// 删除黑名单
async function deleteBlacklist (kind, symbol) {
  if (!confirm(`确定要删除 ${symbol} (${kind === 'spot' ? '现货' : '期货'}) 吗？`)) return
  deleting.value = true
  try {
    await api.deleteBinanceBlacklist(kind, symbol)
    await loadBlacklist()
    // 重新加载页面数据，使黑名单过滤立即生效
    await load()
    handleSuccess('黑名单删除成功')
  } catch (err) {
    handleError(err, '删除黑名单')
  } finally {
    deleting.value = false
  }
}

// 打开对话框时加载黑名单
watch(showBlacklistDialog, (show) => {
  if (show) {
    blacklistKind.value = 'spot'
    loadBlacklist()
  }
})

onMounted(loadInitial)
</script>

<style scoped>
.page {
  max-width: 1300px;
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

.selectors-group {
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
}

.category-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}
.page-header h2 { 
  font-size: 18px; 
  font-weight: 600; 
  margin: 0;
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
  flex-direction: column;
  gap: 12px;
}
.controls { 
  display: flex; 
  align-items: center; 
  gap: 8px; 
  flex-wrap: wrap; 
}
.quick-dates {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.quick-label { color: #555; font-size: 13px; }
.quick-btn {
  height: 28px;
  padding: 0 10px;
  border: 1px solid rgba(0,0,0,.15);
  background: #fff;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}
.quick-btn.active {
  background: #2563eb;
  color: #fff;
  border-color: #2563eb;
}
.quick-btn:hover:not(.active) {
  background: rgba(0,0,0,.04);
}

/* 控件样式 */
.select {
  height: 32px;
  padding: 0 10px;
  border: 1px solid rgba(0,0,0,.15);
  border-radius: 6px;
}
.btn {
  height: 32px;
  padding: 0 12px;
  border: 1px solid rgba(0,0,0,.15);
  background: #fff;
  border-radius: 6px;
  cursor: pointer;
}
.btn:disabled { opacity: .6; cursor: not-allowed; }

.loading {
  padding: 80px 0;
  text-align: center;
  color: #888;
}

/* 每行固定四列 */
.grid4 {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

/* 卡片与表格 */
.card {
  background: rgba(255,255,255,.02);
  border: 1px solid darkgray;
  border-radius: 12px;
  overflow: hidden;
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  border-bottom: 1px solid rgba(0,0,0,.06);
}
.bucket { font-weight: 600; }
.fetched { font-size: 12px; color: #888; margin-top: 2px; }
.tag { font-size: 12px; padding: 2px 6px; border-radius: 999px; }
.tag-spot { background: rgba(16,185,129,.12); color: #10b981; }
.tag-fut  { background: rgba(59,130,246,.12); color: #3b82f6; }

.tbl-wrap {
  overflow-x: auto;
}
.tbl { width: 100%; border-collapse: collapse; table-layout: fixed; }
.tbl th, .tbl td { padding: 4px 6px;}
.tbl thead th { font-size: 12px; color: #666; font-weight: 500; }
.tbl tbody td { font-size: 13px; font-weight: 600; }

/* 列宽 */
.col-rank { width: 36px; text-align: right; }
.col-symbol { width: 92px; font-weight: 600; text-align: center; }
.col-num { text-align: center; font-variant-numeric: tabular-nums; }

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


/* 小字的市值行 */
.meta-row td {
  padding-top: 2px;
  padding-bottom: 4px;
  font-size: 12px;
  color: #888;
  border-bottom: 1px solid rgba(0,0,0,.06);
}
.meta-row .mid-dot { margin: 0 6px; opacity: .6; }
.muted { color: #888; margin-left: 12px; font-weight: normal; }

/* 颜色 */
.up { color: #22c55e; font-weight: 500; }
.down { color: #ef4444; font-weight: 500; }

/* 高亮变化的币种 - 上涨绿色 */
.highlight-up {
  background: linear-gradient(90deg, rgba(34, 197, 94, 0.08) 0%, rgba(34, 197, 94, 0.03) 100%);
  border-left: 3px solid #22c55e;
  box-shadow: 0 1px 3px rgba(34, 197, 94, 0.1);
}
.highlight-up td {
  font-weight: 600;
}

/* 高亮变化的币种 - 下跌红色 */
.highlight-down {
  background: linear-gradient(90deg, rgba(239, 68, 68, 0.08) 0%, rgba(239, 68, 68, 0.03) 100%);
  border-left: 3px solid #ef4444;
  box-shadow: 0 1px 3px rgba(239, 68, 68, 0.1);
}
.highlight-down td {
  font-weight: 600;
}

@media (max-width: 768px) {
  .page-header { flex-direction: column; align-items: flex-start; }
  .grid4 { grid-template-columns: 1fr; }
}

.empty{
  margin-left: 8px;
  margin-top: 6px;
  margin-bottom: 6px;
}

/* 黑名单管理对话框 */
.dialog-overlay {
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
.dialog {
  background: #fff;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}
.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}
.dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.btn-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #666;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}
.btn-close:hover {
  background: rgba(0, 0, 0, 0.05);
}
.dialog-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}
.blacklist-add {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
}
.input {
  flex: 1;
  height: 32px;
  padding: 0 10px;
  border: 1px solid rgba(0, 0, 0, 0.15);
  border-radius: 6px;
  font-size: 14px;
}
.blacklist-list {
  max-height: 400px;
  overflow-y: auto;
}
.empty-text {
  text-align: center;
  color: #888;
  padding: 40px 0;
}
.blacklist-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.blacklist-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.05);
}
.blacklist-item .symbol {
  font-weight: 600;
  font-size: 14px;
}
.btn-delete {
  height: 28px;
  padding: 0 12px;
  border: 1px solid rgba(239, 68, 68, 0.3);
  background: #fff;
  color: #ef4444;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
}
.btn-delete:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.1);
}
.btn-delete:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-secondary {
  background: #f3f4f6;
  color: #374151;
}
.btn-secondary:hover:not(:disabled) {
  background: #e5e7eb;
}
.loading-small {
  text-align: center;
  padding: 20px;
  color: #888;
}

/* 黑名单标签页 */
.blacklist-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}
.tab-btn {
  padding: 8px 16px;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab-btn:hover {
  color: #333;
}
.tab-btn.active {
  color: #3b82f6;
  border-bottom-color: #3b82f6;
  font-weight: 500;
}

/* 黑名单项中的类型标签 */
.kind-tag {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 999px;
  margin-left: 8px;
}
.kind-spot {
  background: rgba(16,185,129,.12);
  color: #10b981;
}
.kind-fut {
  background: rgba(59,130,246,.12);
  color: #3b82f6;
}
.blacklist-item {
  display: flex;
  align-items: center;
  gap: 8px;
}
.blacklist-item .symbol {
  flex: 1;
}
</style>
