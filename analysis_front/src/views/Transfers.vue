<template>
  <div class="container">
    <section class="panel">
    <div class="row topbar">
      <h2>转账实时</h2>
      <div class="spacer"></div>

      <label>交易所：</label>
      <select v-model="entity" @change="reload" class="select">
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>

      <label>链：</label>
      <select v-model="chain" @change="reload" class="select">
        <option value="">全部</option>
        <option v-for="c in chainOptions" :key="c" :value="c">{{ c }}</option>
      </select>

      <label>币种：</label>
      <select v-model="coin" @change="reload" class="select">
        <option value="">全部</option>
        <option v-for="s in coinOptions" :key="s" :value="s">{{ s }}</option>
      </select>

      <div class="gap"></div>

      <input
        class="search"
        v-model.trim="keyword"
        placeholder="搜索 TxID/地址…"
        @keyup.enter="handleSearch"
      />
      <button class="btn" @click="handleSearch">搜索</button>
      <button class="btn btn-secondary" @click="showAdvancedFilters = !showAdvancedFilters">
        {{ showAdvancedFilters ? '收起' : '高级筛选' }}
      </button>
      <button 
        v-if="hasActiveFilters" 
        class="btn btn-clear" 
        @click="clearFilters"
        title="清除所有筛选"
      >
        清除筛选
      </button>
    </div>

    <!-- 高级筛选面板 -->
    <div v-if="showAdvancedFilters" class="advanced-filters">
      <div class="filter-row">
        <label>时间范围：</label>
        <input 
          type="datetime-local" 
          v-model="filters.startTime" 
          @change="handleFilterChange('time')"
          class="date-input"
        />
        <span class="date-separator">至</span>
        <input 
          type="datetime-local" 
          v-model="filters.endTime" 
          @change="handleFilterChange('time')"
          class="date-input"
        />
      </div>
      
      <div class="filter-row">
        <label>方向：</label>
        <select v-model="filters.direction" @change="handleFilterChange('direction')" class="select">
          <option value="">全部</option>
          <option value="in">流入</option>
          <option value="out">流出</option>
        </select>
      </div>

      <div class="filter-row">
        <label>金额范围：</label>
        <input 
          type="number" 
          v-model.number="filters.minAmount" 
          @change="handleFilterChange('amount')"
          class="number-input"
          placeholder="最小金额"
          step="0.00000001"
        />
        <span class="date-separator">至</span>
        <input 
          type="number" 
          v-model.number="filters.maxAmount" 
          @change="handleFilterChange('amount')"
          class="number-input"
          placeholder="最大金额"
          step="0.00000001"
        />
      </div>
    </div>

    <!-- 当前激活的筛选条件显示 -->
    <div v-if="hasActiveFilters" class="active-filters">
      <span class="filter-label">当前筛选：</span>
      <span v-if="keyword" class="filter-tag">
        关键词: {{ keyword }}
        <button class="tag-close" @click="keyword = ''; handleSearch()">×</button>
      </span>
      <span v-if="filters.direction" class="filter-tag">
        方向: {{ filters.direction === 'in' ? '流入' : '流出' }}
        <button class="tag-close" @click="filters.direction = ''; handleFilterChange('direction')">×</button>
      </span>
      <span v-if="filters.startTime || filters.endTime" class="filter-tag">
        时间: {{ filters.startTime || '开始' }} ~ {{ filters.endTime || '结束' }}
        <button class="tag-close" @click="filters.startTime = ''; filters.endTime = ''; handleFilterChange('time')">×</button>
      </span>
      <span v-if="filters.minAmount || filters.maxAmount" class="filter-tag">
        金额: {{ filters.minAmount || '0' }} ~ {{ filters.maxAmount || '∞' }}
        <button class="tag-close" @click="filters.minAmount = null; filters.maxAmount = null; handleFilterChange('amount')">×</button>
      </span>
    </div>
  </section>

  <section class="panel" style="margin-top:12px;">
    <div class="row" style="margin-bottom:10px;">
      <div class="badge">连接：<span :class="{'ok': connected, 'err': !connected}">{{ connected ? '已连接' : '未连接' }}</span></div>
      <div class="spacer"></div>
      <button class="primary" @click="reconnect">重连</button>
    </div>

    <div class="table-wrap">
      <table class="table">
        <thead>
        <tr>
          <th>交易时间 (UTC)</th>
          <th>同步时间 (UTC)</th>
          <th>链</th>
          <th>币种</th>
          <th>方向</th>
          <th>数量</th>
          <th>命中地址</th>
          <th>From → To</th>
          <th>TxID</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="it in items" :key="it.id">
          <td class="mono">{{ fmtTime(it.occurred_at) }}</td>
          <td class="mono" :title="timeDiff(it.occurred_at, it.created_at)">
            {{ fmtTime(it.created_at) }}
            <span v-if="it.created_at && it.occurred_at" class="time-diff">
              (延迟 {{ timeDiffText(it.occurred_at, it.created_at) }})
            </span>
          </td>
          <td>{{ it.chain }}</td>
          <td>{{ it.coin }}</td>
          <td><span :class="['pill', it.direction === 'in' ? 'in' : 'out']">{{ it.direction }}</span></td>
          <td class="mono">{{ fmtAmount(it.amount) }}</td>
          <td class="mono">{{ short(it.address) }}</td>
          <td class="mono">{{ short(it.from) }} → {{ short(it.to) }}</td>
          <td class="mono">
            <a class="link" :href="txLink(it.chain, it.txid)" target="_blank" rel="noreferrer">{{ short(it.txid) }}</a>
          </td>
        </tr>
        </tbody>
      </table>
    </div>

    <!-- 统一分页组件 -->
    <Pagination
      v-if="total > 0"
      v-model:page="page"
      v-model:pageSize="pageSize"
      :total="total"
      :totalPages="totalPages"
      :loading="loading"
      @change="onPaginationChange"
    />
  </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'

const entities = ref([])
const entity = ref('binance')

/** 新增筛选：链/币种 */
const chainOptions = ['bitcoin', 'ethereum', 'tron', 'solana']
const coinOptions  = ['BTC', 'ETH', 'USDT', 'USDC', 'SOL']
const chain = ref('')  // '' 表示全部
const coin  = ref('')  // '' 表示全部

const items = ref([])
const pageSize = ref(50)
const page = ref(1)
const total = ref(0)
const totalPages = ref(1)
const loading = ref(false)

let ws = null
const connected = ref(false)

// 搜索和过滤
const keyword = ref('')
const showAdvancedFilters = ref(false)
const filters = ref({
  startTime: '',
  endTime: '',
  direction: '',
  minAmount: null,
  maxAmount: null
})

const hasActiveFilters = computed(() => {
  return !!(
    keyword.value ||
    filters.value.direction ||
    filters.value.startTime ||
    filters.value.endTime ||
    filters.value.minAmount !== null ||
    filters.value.maxAmount !== null
  )
})

function handleSearch() {
  page.value = 1
  loadRecent()
}

function handleFilterChange(type) {
  page.value = 1
  loadRecent()
}

function clearFilters() {
  keyword.value = ''
  filters.value = {
    startTime: '',
    endTime: '',
    direction: '',
    minAmount: null,
    maxAmount: null
  }
  page.value = 1
  loadRecent()
}

const fmtTime = (s) => {
  if (!s) return '-'
  try {
    // 格式化为 YYYY-MM-DD HH:mm:ss，精确到秒
    const d = new Date(s)
    const year = d.getUTCFullYear()
    const month = String(d.getUTCMonth() + 1).padStart(2, '0')
    const day = String(d.getUTCDate()).padStart(2, '0')
    const hours = String(d.getUTCHours()).padStart(2, '0')
    const minutes = String(d.getUTCMinutes()).padStart(2, '0')
    const seconds = String(d.getUTCSeconds()).padStart(2, '0')
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  } catch {
    return s
  }
}
const fmtAmount = (s) => Number(s).toLocaleString(undefined, { maximumFractionDigits: 8 })
const short = (s) => !s ? '-' : (String(s).length > 12 ? String(s).slice(0,6) + '…' + String(s).slice(-6) : s)

// 计算时间差（毫秒）
const timeDiff = (occurred, created) => {
  if (!occurred || !created) return 0
  try {
    const occurredTime = new Date(occurred).getTime()
    const createdTime = new Date(created).getTime()
    return createdTime - occurredTime
  } catch {
    return 0
  }
}

// 格式化时间差文本
const timeDiffText = (occurred, created) => {
  const diff = timeDiff(occurred, created)
  if (diff <= 0) return '0秒'
  
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return `${seconds}秒`
  
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}分钟`
  
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时`
  
  const days = Math.floor(hours / 24)
  return `${days}天`
}
const txLink = (chain, txid) => {
  const c = String(chain).toLowerCase()
  if (c.includes('btc') || c === 'bitcoin') return `https://mempool.space/tx/${txid}`
  if (c.includes('eth') || c === 'ethereum') return `https://etherscan.io/tx/${txid}`
  if (c.includes('sol')) return `https://solscan.io/tx/${txid}`
  if (c.includes('tron')) return `https://tronscan.org/#/transaction/${txid}`
  return '#'
}

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes(entity.value) && entities.value.length) {
    entity.value = entities.value[0]
  }
}

async function loadRecent() {
  loading.value = true
  try {
    const params = {
      entity: entity.value,
      chain: chain.value || undefined,
      coin : coin.value  || undefined,
      page: page.value,
      page_size: pageSize.value
    }
    
    // 添加搜索和过滤参数
    if (keyword.value) {
      params.keyword = keyword.value
    }
    if (filters.value.direction) {
      params.direction = filters.value.direction
    }
    if (filters.value.startTime) {
      params.start_time = filters.value.startTime
    }
    if (filters.value.endTime) {
      params.end_time = filters.value.endTime
    }
    if (filters.value.minAmount !== null) {
      params.min_amount = filters.value.minAmount
    }
    if (filters.value.maxAmount !== null) {
      params.max_amount = filters.value.maxAmount
    }
    
    const r = await api.recentTransfers(params)
    items.value = (r.items || [])
    total.value = r.total || 0
    totalPages.value = r.total_pages || 1
    page.value = r.page || 1
  } finally {
    loading.value = false
  }
}

function onPaginationChange({ page: newPage, pageSize: newPageSize }) {
  page.value = newPage
  pageSize.value = newPageSize
  loadRecent()
}

function openWS() {
  const url = api.wsTransfersURL({ entity: entity.value })
  if (ws) { ws.close(); ws = null }
  ws = new WebSocket(url)
  ws.onopen = () => { connected.value = true }
  ws.onclose = () => { connected.value = false }
  ws.onerror = () => { connected.value = false }
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      if (msg.type === 'transfers' && Array.isArray(msg.data)) {
        // 前端再做一次链/币种过滤，避免筛选为 BTC 时仍插入 ETH
        const filtered = msg.data.filter(it => {
          if (chain.value && String(it.chain).toLowerCase() !== chain.value) return false
          if (coin.value  && String(it.coin).toUpperCase()   !== coin.value)  return false
          return true
        })
        if (filtered.length) {
          // WebSocket 新数据到达时，如果当前在第一页，则刷新第一页
          if (page.value === 1) {
            loadRecent()
          }
        }
      }
    } catch (e) {}
  }
}

async function reload() { 
  page.value = 1  // 重置到第一页
  await loadRecent()
  openWS() 
}
function reconnect(){ openWS() }

onMounted(async () => { await initEntities(); await reload() })
onBeforeUnmount(() => { if (ws) ws.close() })
</script>

<style scoped lang="scss">
.table-wrap { overflow: auto; }
.table { width: 100%; border-collapse: collapse; font-size: 14px; }
.table th, .table td { padding: 10px 8px; border-bottom: 1px solid var(--border); white-space: nowrap; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace; }
.pill { padding: 2px 8px; border-radius: 999px; font-size: 12px; border: 1px solid var(--border); }
.in { background: rgba(16,185,129,.12); border-color: rgba(16,185,129,.35); color: #16a34a; }
.out { background: rgba(239,68,68,.12); border-color: rgba(239,68,68,.35); color: #ef4444; }
.badge { background:#f3f4f6; border:1px solid var(--border); padding:2px 8px; border-radius:8px; font-size:12px; color:var(--muted); }
.badge .ok { color: #22c55e; } .badge .err { color: #ef4444; }
.time-diff { 
  font-size: 11px; 
  color: #6b7280; 
  margin-left: 4px;
  font-weight: normal;
}
.pagination-info {
  font-size: 14px;
  color: var(--muted);
  padding: 0 12px;
}

/* 搜索和过滤样式 */
.topbar {
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.gap {
  width: 12px;
}

.search {
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
  min-width: 200px;
}

.btn {
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  cursor: pointer;
  font-size: 14px;
}

.btn:hover {
  background: #f3f4f6;
}

.btn-secondary {
  background: #f9fafb;
  border-color: #d1d5db;
}

.btn-clear {
  background: #fee2e2;
  border-color: #fca5a5;
  color: #991b1b;
}

.btn-clear:hover {
  background: #fecaca;
}

.select {
  height: 32px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
}

.advanced-filters {
  margin-top: 12px;
  padding: 12px;
  background: #f9fafb;
  border: 1px solid var(--border);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-row label {
  min-width: 80px;
  font-size: 14px;
  color: var(--text);
}

.date-input, .number-input {
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
  flex: 1;
  max-width: 200px;
}

.date-separator {
  color: var(--muted);
  font-size: 14px;
  margin: 0 4px;
}

.active-filters {
  margin-top: 12px;
  padding: 8px 12px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.filter-label {
  font-size: 14px;
  color: #1e40af;
  font-weight: 500;
}

.filter-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #dbeafe;
  border: 1px solid #93c5fd;
  border-radius: 6px;
  font-size: 13px;
  color: #1e3a8a;
}

.tag-close {
  background: none;
  border: none;
  color: #1e3a8a;
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 0;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.tag-close:hover {
  background: #bfdbfe;
}
</style>
