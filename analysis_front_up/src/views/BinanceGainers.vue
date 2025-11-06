<!-- src/views/BinanceGainers.vue -->
<template>
  <div class="page">
    <header class="page-header">
      <h2>币安涨幅榜（2 小时一段）</h2>
      <div class="controls">
        <label>市场：</label>
        <select v-model="kind" class="select" @change="load">
          <option value="spot">现货</option>
          <option value="futures">合约</option>
        </select>

        <label>日期：</label>
        <input type="date" v-model="date" class="select" @change="load" />

        <button class="btn" @click="load" :disabled="loading">
          {{ loading ? '加载中...' : '刷新' }}
        </button>
      </div>
    </header>

    <section v-if="loading" class="loading">正在获取数据...</section>

    <section v-else>
      <div class="grid">
        <div v-for="slot in displaySlots" :key="slot.key" class="card">
          <div class="card-head">
            <div>
              <div class="bucket">{{ slot.label }}</div>
              <div class="fetched" v-if="slot.group">拉取时间：{{ fmtDate(slot.group.fetched_at) }}</div>
              <div class="fetched" v-else>暂无数据</div>
            </div>
            <div class="tag">{{ kindLabel }}</div>
          </div>

          <div class="tbl-wrap" v-if="slot.group && slot.group.items && slot.group.items.length">
            <table class="tbl">
              <thead>
              <tr>
                <th class="col-rank">#</th>
                <th class="col-symbol">币种</th>
                <th class="col-num">涨幅(24h)</th>
                <th class="col-num">最新价</th>
                <th class="col-num">成交量</th>
              </tr>
              </thead>
              <tbody>
              <tr v-for="item in slot.group.items" :key="item.symbol">
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
                <td class="col-num" :title="item.volume">
                  {{ formatAmount(item.volume) }}
                </td>
              </tr>
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
import { ref, computed, onMounted } from 'vue'
import { api } from '../api/api.js'

const kind = ref('spot')
const date = ref(new Date().toISOString().slice(0, 10))
const loading = ref(false)

const groupsRaw = ref([])
const browserTZ = Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Taipei'

const kindLabel = computed(() => (kind.value === 'spot' ? '现货' : '合约'))

function fmtDate (s) {
  if (!s) return '-'
  return new Date(s).toLocaleString()
}

// ------- 数字格式化：更易读且不截断 -------
function formatPct (p) {
  const n = Number(p)
  if (!isFinite(n)) return p
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}%`
}
function formatPctFull (p) {
  const n = Number(p)
  if (!isFinite(n)) return p
  return `${n >= 0 ? '+' : ''}${n.toFixed(4)}%`
}
function formatPrice (s) {
  const n = Number(s)
  if (!isFinite(n) || n === 0) return s
  // >=1 的保留最多 4 位小数；<1 的保留 6 位有效数字
  if (n >= 1) {
    return n.toLocaleString(undefined, { maximumFractionDigits: 4, useGrouping: false })
        .replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
  } else {
    // 小币种给 6 位有效数字，去掉多余 0
    return Number(n.toPrecision(6)).toString()
  }
}
function formatAmount (s) {
  const n = Number(s)
  if (!isFinite(n)) return s
  const abs = Math.abs(n)
  if (abs >= 1e9) return (n / 1e9).toFixed(2) + 'B'
  if (abs >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (abs >= 1e3) return (n / 1e3).toFixed(2) + 'K'
  return n.toLocaleString()
}

// ------- 槽位划分（本地时区 2h 一段） -------
function pad2 (n) { return String(n).padStart(2, '0') }
function bucketToLocalSlotKey (bucketISO) {
  const d = new Date(bucketISO)
  const y = d.getFullYear(), m = d.getMonth(), dd = d.getDate(), h = d.getHours()
  const slotStartH = Math.floor(h / 2) * 2
  const localStart = new Date(y, m, dd, slotStartH, 0, 0, 0)
  return localStart.getTime()
}

const displaySlots = computed(() => {
  const base = new Date(date.value + 'T00:00:00')
  const slots = Array.from({ length: 12 }, (_, i) => {
    const start = new Date(base.getFullYear(), base.getMonth(), base.getDate(), i * 2, 0, 0, 0)
    const end = new Date(start.getTime() + 2 * 60 * 60 * 1000)
    return {
      key: start.getTime(),
      start, end,
      label: `${pad2(start.getHours())}:00 - ${pad2(end.getHours())}:00`,
      group: null,
    }
  })
  const map = new Map()
  for (const g of groupsRaw.value) {
    map.set(bucketToLocalSlotKey(g.bucket), g)
  }
  for (const s of slots) {
    if (map.has(s.key)) s.group = map.get(s.key)
  }
  return slots
})

async function load () {
  loading.value = true
  try {
    const res = await api.binanceTop({
      kind: kind.value,
      interval: 120,
      date: date.value,
      tz: browserTZ,
    })
    groupsRaw.value = Array.isArray(res.data) ? res.data : []
  } catch (err) {
    console.error(err)
    groupsRaw.value = []
  } finally {
    loading.value = false
  }
}

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
}
.page-header h2 { font-size: 18px; font-weight: 600; }
.controls { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

/* 控件样式 */
.select {
  background: #1f2937;
  border: 1px solid rgba(255,255,255,.12);
  color: #fff;
  border-radius: 6px;
  padding: 4px 6px;
  font-size: 13px;
}
:deep(select option) { color: #111; background: #fff; }
.btn { background: #5b74ff; border: none; color: #fff; font-size: 13px; padding: 5px 12px; border-radius: 6px; cursor: pointer; }

.loading, .empty {
  background: rgba(255,255,255,.02);
  border: 1px solid rgba(255,255,255,.02);
  padding: 16px;
  border-radius: 10px;
  color: rgba(255,255,255,.42);
}

/* 12 段网格：加大最小宽度，避免挤压数字 */
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: 12px;
}

/* 卡片与表格 */
.card {
  background: rgba(255,255,255,.02);
  border: 1px solid rgba(255,255,255,.06);
  border-radius: 12px;
  overflow: hidden;
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 6px;
}
.bucket { font-weight: 600; }
.fetched { font-size: 12px; color: rgba(255,255,255,.45); }
.tag {
  background: rgba(91,116,255,.15);
  border: 1px solid rgba(91,116,255,.5);
  border-radius: 9999px;
  padding: 2px 12px;
  font-size: 12px;
}

/* 表格容器，必要时允许横向滚动，不截断数字 */
.tbl-wrap { overflow-x: auto; }
.tbl { width: 100%; border-collapse: collapse; table-layout: auto; margin-top: 8px; }

/* 表头/单元格 */
th, td {
  padding: 6px 10px;
  border-bottom: 1px solid rgba(255,255,255,.05);
  font-size: 13px;
  white-space: nowrap;
}
th { text-align: left; background: rgba(255,255,255,.04); }
tr:last-child td { border-bottom: none; }

.col-rank { width: 36px; text-align: right; }
.col-symbol { width: 92px; font-weight: 600; }
.col-num {
  text-align: right;
  font-variant-numeric: tabular-nums; /* 等宽数字 */
}


/* 颜色 */
.up { color: #22c55e; font-weight: 500; }
.down { color: #ef4444; font-weight: 500; }

@media (max-width: 768px) {
  .page-header { flex-direction: column; align-items: flex-start; }
  .grid { grid-template-columns: 1fr; }
  th:nth-child(5), td:nth-child(5) { display: none; } /* 移动端隐藏成交量列 */
}
</style>
