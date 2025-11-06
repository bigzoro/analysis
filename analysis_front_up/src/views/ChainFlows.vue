<template>
  <section class="panel">
    <!-- 顶部：同一行 交易所 + 链 -->
    <div class="row topbar">
      <h2>资金链</h2>
      <div class="spacer"></div>

      <label>交易所：</label>
      <select class="top-select" v-model="entity" @change="fetchData">
        <option value="all">全部交易所</option>
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>

      <label style="margin-left:12px;">链：</label>
      <select class="top-select" v-model="chain" @change="fetchData">
        <option value="all">全部链</option>
        <option v-for="c in chainOptions" :key="c.value" :value="c.value">{{ c.label }}</option>
      </select>
    </div>

    <!-- 主体：左 = 日期&币种，右 = 结果（保持你的固定高样式） -->
    <div class="grid">
      <aside class="left">
        <div class="box">
          <div class="row"><h3>日期</h3></div>
          <div class="vstack">
            <label class="muted">开始</label>
            <input class="control" type="date" v-model="start" @change="fetchData" />
            <label class="muted" style="margin-top:8px;">结束</label>
            <input class="control" type="date" v-model="end" @change="fetchData" />
          </div>
        </div>

        <div class="box">
          <div class="row"><h3>币种</h3></div>
          <select class="control" v-model="coin" @change="fetchData">
            <option v-for="o in coinOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
          <div class="muted tip">不选或“全部币种”=统计该链所有币种。</div>
        </div>
      </aside>

      <main class="right box">
        <div class="row head-row">
          <strong>结果</strong>
          <div class="spacer"></div>
          <span class="muted">范围：{{ start }} ~ {{ end }}</span>
        </div>

        <div class="summary" v-if="xData.length">
          <div>
            区间合计：净流入 <b class="num">{{ fmt(sumNet) }}</b> ｜ 流入 <b class="num">{{ fmt(sumIn) }}</b> ｜ 流出 <b class="num">{{ fmt(sumOut) }}</b>
          </div>
          <div class="muted">（{{ entityLabel(entity) }} / {{ chainLabel(chain) }} / {{ coinLabel(coin) }}）</div>
        </div>

        <div v-if="xData.length" class="list">
          <div v-for="(d, i) in xData" :key="i" class="item">
            <div class="date">{{ d }}</div>
            <div class="nums">
              <span>净流入：<b class="num">{{ fmt(series.net[i]) }}</b></span>
              <span>流入：<b class="num pos">{{ fmt(series.in[i]) }}</b></span>
              <span>流出：<b class="num neg">{{ fmt(series.out[i]) }}</b></span>
            </div>
          </div>
        </div>
        <div v-else class="empty muted">暂无数据</div>
      </main>
    </div>
  </section>
</template>

<script setup>
import dayjs from 'dayjs'
import { ref, computed, onMounted } from 'vue'
import { api } from '../api/api.js'

const chainOptions = [
  { value: 'bitcoin',  label: 'Bitcoin'  },
  { value: 'ethereum', label: 'Ethereum' },
  { value: 'solana',   label: 'Solana'   },
  { value: 'tron',     label: 'TRON'     },
]

const coinOptions = [
  { value: '',     label: '全部币种' },
  { value: 'BTC',  label: 'BTC' },
  { value: 'ETH',  label: 'ETH' },
  { value: 'SOL',  label: 'SOL' },
  { value: 'USDT', label: 'USDT' },
  { value: 'USDC', label: 'USDC' },
]

const entities = ref([])
const entity   = ref('binance')   // 也可以预设为 'all'
const chain    = ref('all')  // 也可以预设为 'all'
const coin     = ref('')

const start = ref(dayjs().subtract(30, 'day').format('YYYY-MM-DD'))
const end   = ref(dayjs().format('YYYY-MM-DD'))

const xData = ref([])
const series = ref({ net: [], in: [], out: [] })

const sumIn  = computed(() => series.value.in.reduce((a, b) => a + b, 0))
const sumOut = computed(() => series.value.out.reduce((a, b) => a + b, 0))
const sumNet = computed(() => sumIn.value - sumOut.value)

function entityLabel(v) {
  return (v && v.toLowerCase() !== 'all') ? v : '全部交易所'
}
function chainLabel(v) {
  if (!v || v.toLowerCase() === 'all') return '全部链'
  const item = chainOptions.find(x => x.value === v)
  return item ? item.label : v
}
function coinLabel(v) {
  const item = coinOptions.find(x => x.value === v)
  return item ? item.label : v
}
function fmt(n) {
  return Number(n || 0).toFixed(4).replace(/\.?0+$/, '')
}

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes(entity.value) && entities.value.length) entity.value = entities.value[0]
}

async function fetchData() {
  const r = await api.dailyFlowsByChain({
    entity: entity.value,  // 支持 all
    chain:  chain.value,   // 支持 all
    start:  start.value,
    end:    end.value,
    coin:   coin.value,
  })
  const rows = (r.data || []).slice().sort((a, b) => a.day.localeCompare(b.day))
  xData.value = rows.map(d => d.day)
  series.value = {
    net: rows.map(d => d.net),
    in:  rows.map(d => d.in),
    out: rows.map(d => d.out),
  }
}

onMounted(async () => {
  await initEntities()
  await fetchData()
})
</script>

<style scoped lang="scss">
:root { --card-radius: 10px; --control-h: 36px; --control-w: 180px; --gap: 14px; }
.topbar { flex-wrap: wrap; }
.top-select {
  height: var(--control-h); line-height: var(--control-h); padding: 0 10px;
  border: 1px solid var(--border); border-radius: 8px; background: #0b1320; color: var(--text);
}
.grid { display: grid; grid-template-columns: 320px 1fr; gap: var(--gap); }
.left { display: grid; grid-template-rows: auto auto; gap: var(--gap); height: 500px; position: sticky; top: 12px; }
.box { background: #0f1420; border: 1px solid var(--border); border-radius: var(--card-radius); padding: 14px; }
.right.box { min-height: 500px; overflow: auto; }
.row { display: flex; align-items: center; gap: 10px; }
.vstack { display: grid; gap: 6px; }
.spacer { flex: 1; }
.muted { color: var(--muted); font-size: 12px; }
.tip { margin-top: 6px; }

/* 统一控件尺寸 */
.control {
  width: var(--control-w); min-width: var(--control-w); max-width: var(--control-w);
  height: var(--control-h); min-height: var(--control-h); line-height: var(--control-h);
  padding: 0 10px; box-sizing: border-box; border: 1px solid var(--border);
  border-radius: 8px; background: #0b1320; color: var(--text); font-size: 14px; outline: none;
}
.control:focus { border-color: #3b82f6; box-shadow: 0 0 0 2px rgba(59,130,246,.25); }
select.control { appearance: none; -webkit-appearance: none; -moz-appearance: none; padding-right: 28px;
  background-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24'%3E%3Cpath fill='%23aab2c5' d='M7 10l5 5 5-5z'/%3E%3C/svg%3E");
  background-repeat:no-repeat; background-position:right 8px center; }
input[type="date"].control::-webkit-datetime-edit { padding: 0 2px; }
input[type="date"].control::-webkit-calendar-picker-indicator { margin-right: 6px; }

/* 结果列表 */
.head-row { margin-bottom: 8px; }
.summary { margin-bottom: 6px; display: grid; gap: 4px; }
.list { display: grid; gap: 8px; margin-top: 10px; }
.item { display: grid; grid-template-columns: 120px 1fr; gap: 12px; padding: 10px;
  border: 1px dashed rgba(255,255,255,.06); border-radius: 8px; transition: background .15s, border-color .15s; }
.item:hover { background: rgba(255,255,255,.03); border-color: rgba(255,255,255,.12); }
.item .date { color: #c7d2fe; font-weight: 600; }
.item .nums { display: flex; gap: 16px; flex-wrap: wrap; align-items: baseline; }
.num { font-feature-settings: "tnum"; letter-spacing: .2px; }
.pos { color: #9ae6b4; }
.neg { color: #fca5a5; }
.empty { padding: 24px 0; }

@media (max-width: 900px) {
  .grid { grid-template-columns: 1fr; }
  .right.box { min-height: auto; }
  .item { grid-template-columns: 1fr; }
}
</style>
