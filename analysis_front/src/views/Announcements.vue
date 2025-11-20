<template>
  <div class="page">
    <section class="panel">
      <div class="row topbar">
        <h2>交易所公告</h2>
        <div class="spacer"></div>

        <!-- 来源固定为 CoinCarp，不再显示选择 -->

        <label>分类：</label>
        <button
            class="chip"
            :class="{ active: cats.newcoin }"
            @click="toggleCat('newcoin')"
            title="新币上架 / New Listing"
        >新币</button>
        <button
            class="chip"
            :class="{ active: cats.finance }"
            @click="toggleCat('finance')"
            title="Earn / Staking / 理财"
        >理财</button>
        <button
            class="chip"
            :class="{ active: cats.event }"
            @click="toggleCat('event')"
            title="重要事件"
        >事件</button>
        <button
            class="chip"
            :class="{ active: cats.other }"
            @click="toggleCat('other')"
        >其它</button>

        <div class="gap"></div>

        <label>筛选：</label>
        <select v-model="filters.exchange" @change="handleFilterChange('exchange')" class="select">
          <option value="">全部交易所</option>
          <option v-for="ex in availableExchanges" :key="ex" :value="ex">{{ formatExchangeName(ex) }}</option>
        </select>

        <div class="gap"></div>

        <input
            class="search"
            v-model.trim="keyword"
            placeholder="搜索标题/摘要…"
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
          <label>日期范围：</label>
          <input 
            type="date" 
            v-model="filters.startDate" 
            @change="handleFilterChange('date')"
            class="date-input"
            placeholder="开始日期"
          />
          <span class="date-separator">至</span>
          <input 
            type="date" 
            v-model="filters.endDate" 
            @change="handleFilterChange('date')"
            class="date-input"
            placeholder="结束日期"
          />
        </div>
        
        <div class="filter-row">
          <label>重要事件：</label>
          <select v-model="filters.isEvent" @change="handleFilterChange('isEvent')" class="select">
            <option value="">全部</option>
            <option value="true">仅重要事件</option>
            <option value="false">排除重要事件</option>
          </select>
        </div>

        <div class="filter-row">
          <label>已验证：</label>
          <select v-model="filters.verified" @change="handleFilterChange('verified')" class="select">
            <option value="">全部</option>
            <option value="true">仅已验证</option>
            <option value="false">未验证</option>
          </select>
        </div>

        <div class="filter-row">
          <label>情绪：</label>
          <select v-model="filters.sentiment" @change="handleFilterChange('sentiment')" class="select">
            <option value="">全部</option>
            <option value="positive">积极</option>
            <option value="neutral">中性</option>
            <option value="negative">消极</option>
          </select>
        </div>
      </div>

      <!-- 当前激活的筛选条件显示 -->
      <div v-if="hasActiveFilters" class="active-filters">
        <span class="filter-label">当前筛选：</span>
        <span v-if="filters.exchange" class="filter-tag">
          交易所: {{ formatExchangeName(filters.exchange) }}
          <button class="tag-close" @click="filters.exchange = ''; handleFilterChange('exchange')">×</button>
        </span>
        <span v-if="filters.startDate || filters.endDate" class="filter-tag">
          日期: {{ filters.startDate || '开始' }} ~ {{ filters.endDate || '结束' }}
          <button class="tag-close" @click="filters.startDate = ''; filters.endDate = ''; handleFilterChange('date')">×</button>
        </span>
        <span v-if="filters.isEvent" class="filter-tag">
          重要事件: {{ filters.isEvent === 'true' ? '是' : '否' }}
          <button class="tag-close" @click="filters.isEvent = ''; handleFilterChange('isEvent')">×</button>
        </span>
        <span v-if="filters.verified" class="filter-tag">
          已验证: {{ filters.verified === 'true' ? '是' : '否' }}
          <button class="tag-close" @click="filters.verified = ''; handleFilterChange('verified')">×</button>
        </span>
        <span v-if="filters.sentiment" class="filter-tag">
          情绪: {{ sentimentLabel(filters.sentiment) }}
          <button class="tag-close" @click="filters.sentiment = ''; handleFilterChange('sentiment')">×</button>
        </span>
        <span v-if="keyword" class="filter-tag">
          关键词: {{ keyword }}
          <button class="tag-close" @click="keyword = ''; handleSearch()">×</button>
        </span>
      </div>

      <div v-if="error" class="error">加载失败：{{ error }}</div>

      <template v-if="!loading">
        <table class="table ann-table" v-if="items.length">
          <thead>
          <tr>
            <th>标题</th>
            <th class="col-cat">分类</th>
            <th class="col-exchange">交易所</th>
            <th class="col-time">时间</th>
            <th class="col-link">链接</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="n in items" :key="rowKey(n)" :class="{ 'row-event': n.is_event, 'row-verified': n.verified }">
            <td class="text">
              <span v-if="n.is_event" class="event-badge" title="重要事件">⭐</span>
              <span v-if="n.verified" class="verified-icon" title="已验证">✓</span>
              {{ n.title }}
            </td>
            <td>
              <span class="badge" :data-cat="n.category">{{ catLabel(n.category) }}</span>
            </td>
            <td class="exchange">
              <span v-if="n.exchange" class="exchange-badge">{{ n.exchange }}</span>
              <span v-else class="muted">-</span>
            </td>
            <td class="mono time">{{ fmt(n.release_time) }}</td>
            <td class="link-cell">
              <a class="link" :href="getAnnouncementUrl(n)" target="_blank" rel="noopener">打开</a>
            </td>
          </tr>
          </tbody>
        </table>

        <div v-else class="empty">没有数据</div>
      </template>
      
      <div v-if="loading" class="loading">加载中…</div>

      <!-- 统一分页组件（放在条件块外面，只要有数据就显示） -->
      <Pagination
        v-if="!loading && total > 0"
        v-model:page="page"
        v-model:pageSize="pageSize"
        :total="total"
        :totalPages="totalPages"
        :loading="loading"
        :pageSizeOptions="[10, 20, 50, 100, 200]"
        @change="onPaginationChange"
      />
    </section>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'
import { handleError } from '../utils/errorHandler.js'

/** 来源：固定使用 CoinCarp */
const src = ref({ 
  coincarp: true    // 唯一数据源
})

/** 分类：默认启用所有分类 */
const cats = ref({ newcoin: true, finance: true, event: true, other: true })

/** 筛选条件 */
const filters = ref({
  exchange: '',      // 交易所筛选
  startDate: '',     // 开始日期
  endDate: '',       // 结束日期
  isEvent: '',       // 重要事件：'true' | 'false' | ''
  verified: '',      // 已验证：'true' | 'false' | ''
  sentiment: ''      // 情绪：'positive' | 'neutral' | 'negative' | ''
})

const showAdvancedFilters = ref(false) // 是否显示高级筛选面板

// 常见交易所列表
const commonExchanges = ['binancepro', 'okx', 'bybit', 'upbit', 'coinbase', 'kraken', 'gate', 'huobipro', 'kucoin', 'bitget']

// 从数据中提取的交易所列表
const exchangesFromData = ref(new Set())

// 可用交易所列表（常见交易所 + 从数据中提取的）
const availableExchanges = computed(() => {
  const all = new Set([...commonExchanges, ...Array.from(exchangesFromData.value)])
  return Array.from(all).sort()
})

const keyword = ref('')
const items = ref([])
const page = ref(1) // 当前页码
const pageSize = ref(10) // 每页显示数量
const total = ref(0) // 总记录数
const totalPages = ref(1) // 总页数
const loading = ref(false)
const error = ref('')

function toggleCat(key) {
  cats.value[key] = !cats.value[key]
  // 至少保证一个选中；如全被点掉则恢复全选
  if (!cats.value.newcoin && !cats.value.finance && !cats.value.event && !cats.value.other) {
    cats.value = { newcoin: true, finance: true, event: true, other: true }
  }
  // 切换分类时重置到第一页
  page.value = 1
  load()
}

function catLabel(cat) {
  switch (cat) {
    case 'newcoin': return '新币'
    case 'finance': return '理财'
    case 'event': return '事件'
    default: return '其它'
  }
}

function formatExchangeName(exchange) {
  // 将交易所名称转换为更友好的显示格式
  const nameMap = {
    'binance': 'Binance',
    'binancepro': 'Binance',
    'okx': 'OKX',
    'bybit': 'Bybit',
    'upbit': 'Upbit',
    'coinbase': 'Coinbase',
    'kraken': 'Kraken',
    'gate': 'Gate.io',
    'huobi': 'Huobi',
    'huobipro': 'Huobi',
    'kucoin': 'KuCoin',
    'bitget': 'Bitget'
  }
  return nameMap[exchange.toLowerCase()] || exchange.charAt(0).toUpperCase() + exchange.slice(1)
}

function sentimentLabel(sentiment) {
  const map = {
    'positive': '积极',
    'neutral': '中性',
    'negative': '消极'
  }
  return map[sentiment] || sentiment
}

// 检查是否有激活的筛选条件
const hasActiveFilters = computed(() => {
  return !!(
    filters.value.exchange ||
    filters.value.startDate ||
    filters.value.endDate ||
    filters.value.isEvent ||
    filters.value.verified ||
    filters.value.sentiment ||
    keyword.value
  )
})

function clearFilters() {
  filters.value = {
    exchange: '',
    startDate: '',
    endDate: '',
    isEvent: '',
    verified: '',
    sentiment: ''
  }
  keyword.value = ''
  page.value = 1
  load()
}

function getAnnouncementUrl(item) {
  // 如果有 newscode，使用 newscode 构建 URL
  if (item.news_code && item.news_code.trim()) {
    return `https://www.coincarp.com/zh/exchange/announcement/${item.news_code}/`
  }
  // 否则使用原有的 URL
  return item.url || '#'
}

// sourceLabel, sentimentLabel, normalizeTags 函数已移除（不再显示来源和标签）

function fmt(iso) {
  try {
    const d = new Date(iso)
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    const hh = String(d.getHours()).padStart(2, '0')
    const mm = String(d.getMinutes()).padStart(2, '0')
    return `${y}-${m}-${dd} ${hh}:${mm}`
  } catch { return iso }
}

function rowKey(n) {
  return `${n.external_id || n.id || n.url}`
}

async function load() {
  loading.value = true
  error.value = ''
  items.value = []
  try {
    const selCats = []
    if (cats.value.newcoin) selCats.push('newcoin')
    if (cats.value.finance) selCats.push('finance')
    if (cats.value.event) selCats.push('event')
    if (cats.value.other) selCats.push('other')

    const params = {
      categories: selCats.join(','),
      q: keyword.value || '',
      page: page.value,
      page_size: pageSize.value
    }

    // 添加筛选参数
    if (filters.value.exchange) {
      params.exchange = filters.value.exchange
    }
    if (filters.value.startDate) {
      params.start_date = filters.value.startDate
    }
    if (filters.value.endDate) {
      params.end_date = filters.value.endDate
    }
    if (filters.value.isEvent) {
      params.is_event = filters.value.isEvent === 'true'
    }
    if (filters.value.verified) {
      params.verified = filters.value.verified === 'true'
    }
    if (filters.value.sentiment) {
      params.sentiment = filters.value.sentiment
    }

    const data = await api.listAnnouncements(params)
    items.value = Array.isArray(data?.items) ? data.items : []
    total.value = data?.total || 0
    totalPages.value = data?.total_pages || 1
    page.value = data?.page || page.value
    
    // 从数据中提取交易所列表
    const exchanges = new Set()
    items.value.forEach(item => {
      if (item.exchange && item.exchange.trim()) {
        exchanges.add(item.exchange.toLowerCase())
      }
    })
    exchangesFromData.value = exchanges
  } catch (e) {
    handleError(e, '加载公告', { showToast: false })
    error.value = e?.message || String(e)
    total.value = 0
    totalPages.value = 1
  } finally {
    loading.value = false
  }
}

function onPaginationChange({ page: newPage, pageSize: newPageSize }) {
  page.value = newPage
  pageSize.value = newPageSize
  load()
  // 滚动到顶部
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function handleSearch() {
  page.value = 1
  load()
}

function handleFilterChange(type) {
  // 筛选条件通过 v-model 自动更新
  page.value = 1
  load()
}

// 首次加载
load()
</script>

<style scoped>
/* ========== 白底黑字（跟随全局变量） ========== */
.page { padding: 16px; color: var(--text); }
.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 16px;
  box-shadow: 0 1px 2px rgba(0,0,0,.04);
}

.row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.topbar { margin-bottom: 12px; }
.spacer { flex: 1; }
.gap { width: 12px; height: 1px; }

label { color: var(--text); }

/* 复选、输入、按钮 */
.chk { display: inline-flex; align-items: center; gap: 6px; cursor: pointer; user-select: none; }
.chk input { margin: 0; }

.search {
  min-width: 220px;
  background: #fff;
  border: 1px solid var(--border);
  color: var(--text);
  padding: 6px 10px;
  border-radius: 10px;
  outline: none;
}
.btn {
  height: 32px;
  padding: 0 12px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: #f3f4f6;
  color: #111827;
  cursor: pointer;
}
.btn:hover { background: #edeff2; }

/* 标签按钮（chip） */
.chip {
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: #fff;
  color: var(--muted);
  cursor: pointer;
  font-size: 13px;
}
.chip.active {
  border-color: #c7d2fe;
  background: #eef2ff;
  color: #3b82f6;
}

/* 表格：白底 + 浅表头 + 柔和 hover */
.table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
  font-size: 14px;
}
thead th {
  position: sticky; top: 0; z-index: 1;
  background: #f9fafb;
  color: #374151;
  font-weight: 600;
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border);
}
tbody td {
  padding: 12px;
  border-bottom: 1px dashed rgba(17,24,39,.10);
  background: #fff;
  color: var(--text);
}
.ann-table tbody tr:hover td { background: #f7f8fa; }

/* 列宽与对齐 */
.col-cat { width: 100px; }
.col-exchange { width: 100px; }
.col-time { width: 180px; }
.col-link { width: 90px; }

/* 表头和数据列对齐 */
thead th.col-time {
  text-align: center;
}
tbody td.time {
  text-align: right;
}

thead th.col-link,
tbody td.link-cell {
  text-align: center;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-variant-numeric: tabular-nums;
}
.time { color:#374151; white-space: nowrap; }

.text { white-space: pre-wrap; line-height: 1.5; }

/* 链接主色 */
.link { color: var(--primary); text-decoration: none; }
.link:hover { text-decoration: underline; }

/* 分类徽章 */
.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  border: 1px solid var(--border);
  color: var(--muted);
  background: #fff;
}
.badge[data-cat="newcoin"] {
  background: #eef2ff; border-color: #c7d2fe; color: #3b82f6;
}
.badge[data-cat="finance"] {
  background: rgba(250,204,21,.15); border-color: rgba(250,204,21,.45); color: #a16207;
}
.badge[data-cat="event"] {
  background: #fef3c7; border-color: #fbbf24; color: #92400e;
}
.badge[data-cat="other"] {
  background: #f3f4f6; border-color: var(--border); color: var(--muted);
}

/* 已验证图标 */
.verified-icon {
  display: inline-block;
  margin-right: 6px;
  color: #22c55e;
  font-weight: bold;
  font-size: 12px;
}

/* 事件徽章 */
.event-badge {
  display: inline-block;
  margin-right: 6px;
  color: #f59e0b;
  font-size: 14px;
}

/* 交易所徽章 */
.exchange-badge {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  background: #e5e7eb;
  color: #374151;
}

/* 高级筛选面板 */
.advanced-filters {
  margin-top: 16px;
  padding: 16px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.filter-row:last-child {
  margin-bottom: 0;
}

.filter-row label {
  min-width: 80px;
  font-size: 14px;
  color: var(--text);
}

.date-input {
  height: 36px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 14px;
}

.date-separator {
  color: var(--muted);
  font-size: 14px;
  margin: 0 4px;
}

/* 激活的筛选条件显示 */
.active-filters {
  margin-top: 12px;
  padding: 12px;
  background: rgba(59, 130, 246, 0.05);
  border-radius: 8px;
  border: 1px solid rgba(59, 130, 246, 0.2);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-label {
  font-size: 14px;
  color: #3b82f6;
  font-weight: 500;
}

.filter-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #fff;
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 16px;
  font-size: 13px;
  color: #3b82f6;
}

.tag-close {
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 50%;
  color: #3b82f6;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.tag-close:hover {
  background: rgba(59, 130, 246, 0.2);
}

.btn-secondary {
  background: #f3f4f6;
  color: #374151;
  border-color: rgba(0, 0, 0, 0.12);
}

.btn-secondary:hover {
  background: #e5e7eb;
}

.btn-clear {
  background: #fee2e2;
  color: #dc2626;
  border-color: rgba(220, 38, 38, 0.3);
}

.btn-clear:hover {
  background: #fecaca;
}

/* 情绪和热度徽章样式已移除（标签列已删除） */

/* 行样式 */
.row-event {
  background: #fefce8 !important;
}
.row-event:hover td {
  background: #fef9c3 !important;
}
.row-verified {
  border-left: 3px solid #22c55e;
}

/* 选择框 */
.select {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  color: var(--text);
  font-size: 13px;
  cursor: pointer;
  outline: none;
}

.muted {
  color: var(--muted);
  font-size: 12px;
}

/* 状态信息 */
.loading { margin-top: 10px; opacity: .8; color: var(--muted); }
.empty { margin-top: 10px; opacity: .7; color: var(--muted); }
.error { color: #ef4444; }

/* 分页控件 */
.pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding: 16px;
  background: #f9fafb;
  border-radius: 10px;
  border: 1px solid var(--border);
  flex-wrap: wrap;
  gap: 16px;
}

.pagination-left {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.pagination-info {
  color: var(--muted);
  font-size: 13px;
}

.page-size-selector {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-size-selector label {
  color: var(--text);
  font-size: 13px;
  white-space: nowrap;
}

.select-page-size {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  color: var(--text);
  font-size: 13px;
  cursor: pointer;
  outline: none;
  min-width: 70px;
}

.select-page-size:hover {
  border-color: #9ca3af;
}

.select-page-size:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.page-size-label {
  color: var(--muted);
  font-size: 13px;
  white-space: nowrap;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.btn-page {
  min-width: 36px;
  height: 36px;
  padding: 0 10px;
  border: 1px solid var(--border);
  background: #fff;
  color: var(--text);
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}

.btn-page:hover:not(:disabled) {
  background: #f3f4f6;
  border-color: #9ca3af;
}

.btn-page:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-page.active {
  background: #3b82f6;
  color: #fff;
  border-color: #3b82f6;
  font-weight: 600;
}

.btn-page.active:hover {
  background: #2563eb;
}

.page-numbers {
  display: flex;
  gap: 4px;
  margin: 0 4px;
}
</style>
