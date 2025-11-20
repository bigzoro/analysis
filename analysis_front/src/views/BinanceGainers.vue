<!-- src/views/BinanceGainers.vue -->
<template>
  <div class="page">
    <header class="page-header">
      <h2>币安涨幅榜（2 小时一段）</h2>
      <div class="controls">
        <div class="date-row">
          <label>日期：</label>
          <input type="date" v-model="date" class="select" @change="load" />
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
        <div class="actions">
          <button class="btn" @click="load" :disabled="loading">
            {{ loading ? '加载中...' : '刷新' }}
          </button>
          <button class="btn btn-secondary" @click="showBlacklistDialog = true">
            管理黑名单
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

    <section v-if="loading" class="loading">正在获取数据...</section>

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
                <tr>
                  <td class="col-rank">{{ item.rank }}</td>
                  <td class="col-symbol">{{ item.symbol }}</td>
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
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api/api.js'
import { handleError, handleSuccess } from '../utils/errorHandler.js'

const date = ref(new Date().toISOString().slice(0, 10))
const loading = ref(false)

const quickDates = computed(() => {
  const today = new Date()
  return Array.from({ length: 10 }, (_, i) => {
    const d = new Date(today)
    d.setDate(today.getDate() - (9 - i))
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

// --- 槽位划分（本地时区，每 2 小时一段）
function pad2 (n) { return String(n).padStart(2, '0') }
function bucketToLocalSlotKey (bucketISO) {
  const d = new Date(bucketISO)
  const y = d.getFullYear(), m = d.getMonth(), dd = d.getDate(), h = d.getHours()
  const slotStartH = Math.floor(h / 2) * 2
  const localStart = new Date(y, m, dd, slotStartH, 0, 0, 0)
  return localStart.getTime()
}
const daySlots = computed(() => {
  const base = new Date(date.value + 'T00:00:00')
  return Array.from({ length: 12 }, (_, i) => {
    const start = new Date(base.getFullYear(), base.getMonth(), base.getDate(), i * 2, 0, 0, 0)
    const end   = new Date(start.getTime() + 2 * 60 * 60 * 1000)
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

// 组合成“每行 4 个卡片：Spot/Futures 成对出现”
const rows = computed(() => {
  const mapSpot = mapBySlot(groupsSpot.value || [])
  const mapFut  = mapBySlot(groupsFut.value || [])
  const out = []

  for (let i = 0; i < daySlots.value.length; i += 2) {
    const s0 = daySlots.value[i]
    const s1 = daySlots.value[i + 1]

    const cells = []
    // 现货 i
    cells.push({
      key: `spot-${s0.key}`,
      kind: 'spot',
      slot: s0,
      group: mapSpot.get(s0.key) || null,
    })
    // 现货 i+1
    cells.push({
      key: s1 ? `spot-${s1.key}` : `spot-${s0.key}-dup`,
      kind: 'spot',
      slot: s1 || s0,
      group: s1 ? (mapSpot.get(s1.key) || null) : null,
    })
    // 合约 i
    cells.push({
      key: `fut-${s0.key}`,
      kind: 'futures',
      slot: s0,
      group: mapFut.get(s0.key) || null,
    })
    // 合约 i+1
    cells.push({
      key: s1 ? `fut-${s1.key}` : `fut-${s0.key}-dup`,
      kind: 'futures',
      slot: s1 || s0,
      group: s1 ? (mapFut.get(s1.key) || null) : null,
    })

    out.push({ key: `row-${i}`, cells })
  }
  return out
})

function selectDate (value) {
  if (date.value === value) return
  date.value = value
  load()
}

async function load () {
  loading.value = true
  try {
    const [spot, fut] = await Promise.all([
      api.binanceTop({ kind: 'spot',    interval: 120, date: date.value, tz: browserTZ }),
      api.binanceTop({ kind: 'futures', interval: 120, date: date.value, tz: browserTZ }),
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

onMounted(load)
</script>

<style scoped>
.page {
  max-width: 1300px;
  margin: 20px auto;
  padding: 0 14px 40px;
}
.page-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.page-header h2 { font-size: 18px; font-weight: 600; }
.controls {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.date-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.actions {
  display: flex;
  align-items: center;
  gap: 8px;
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
  padding: 10px 12px;
  border-bottom: 1px solid rgba(0,0,0,.06);
}
.bucket { font-weight: 600; }
.fetched { font-size: 12px; color: #888; margin-top: 2px; }
.tag { font-size: 12px; padding: 2px 6px; border-radius: 999px; }
.tag-spot { background: rgba(16,185,129,.12); color: #10b981; }
.tag-fut  { background: rgba(59,130,246,.12); color: #3b82f6; }

.tbl-wrap {
}
.tbl { width: 100%; border-collapse: collapse; table-layout: fixed; }
.tbl th, .tbl td { padding: 6px 8px;}
.tbl thead th { font-size: 12px; color: #666; font-weight: 500; }
.tbl tbody td { font-size: 13px; }

/* 列宽 */
.col-rank { width: 36px; text-align: right; }
.col-symbol { width: 92px; font-weight: 600; }
.col-num { text-align: center; font-variant-numeric: tabular-nums; }

/* 小字的市值行 */
.meta-row td {
  padding-top: 4px;
  padding-bottom: 8px;
  font-size: 12px;
  color: #888;
  border-bottom: 1px solid rgba(0,0,0,.06);
}
.meta-row .mid-dot { margin: 0 6px; opacity: .6; }
.muted { color: #888; margin-left: 12px}

/* 颜色 */
.up { color: #22c55e; font-weight: 500; }
.down { color: #ef4444; font-weight: 500; }

@media (max-width: 768px) {
  .page-header { flex-direction: column; align-items: flex-start; }
  .grid4 { grid-template-columns: 1fr; }
}

.empty{
  margin-left: 15px;
  margin-top: 10px;
  margin-bottom: 10px;
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
