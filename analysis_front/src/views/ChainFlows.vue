<template>
  <div class="container">
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

    <!-- 主体：左 = 日期&币种，右 = 结果 -->
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
          <!-- 反向显示（最新在上） -->
          <div
              v-for="k in xData.length"
              :key="xData[xData.length - k]"
              class="item"
          >
            <div class="date">{{ xData[xData.length - k] }}</div>
            <div class="nums">
              <span>净流入：<b class="num">{{ fmt(series.net[xData.length - k]) }}</b></span>
              <span>流入：<b class="num pos">{{ fmt(series.in[xData.length - k]) }}</b></span>
              <span>流出：<b class="num neg">{{ fmt(series.out[xData.length - k]) }}</b></span>
            </div>
          </div>
        </div>
        <div v-else class="empty muted">暂无数据</div>
      </main>
    </div>
  </section>
  </div>
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
  { value: 'TRX',  label: 'TRX' },
  { value: 'USDT', label: 'USDT' },
  { value: 'USDC', label: 'USDC' },
]

const entities   = ref([])
const entity     = ref('all')  // 或者默认 'binance'，按你后端约定
const chain      = ref('all')
const coin       = ref('')     // 空字符串表示全部币种
const start      = ref(dayjs().subtract(30, 'day').format('YYYY-MM-DD'))
const end        = ref(dayjs().format('YYYY-MM-DD'))

const xData  = ref([])                 // ['2025-11-01', ...]
const series = ref({ net: [], in: [], out: [] })

const sumNet = computed(() => series.value.net.reduce((a,b)=>a+(+b||0),0))
const sumIn  = computed(() => series.value.in.reduce((a,b)=>a+(+b||0),0))
const sumOut = computed(() => series.value.out.reduce((a,b)=>a+(+b||0),0))

function chainLabel (v) {
  if (v === 'all') return '全部链'
  const item = chainOptions.find(i => i.value === v)
  return item ? item.label : v
}
function entityLabel (v) {
  if (v === 'all') return '全部交易所'
  return v || '-'
}
function coinLabel (v) {
  if (!v) return '全部币种'
  const item = coinOptions.find(i => i.value === v)
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

<style scoped>
:root { --card-radius: 12px; --control-h: 32px; --control-w: 180px; --gap: 14px; }

.topbar { flex-wrap: wrap; }

.top-select {
  height: var(--control-h);
  line-height: var(--control-h);
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  color: var(--text);
}

.grid { display: grid; grid-template-columns: 320px 1fr; gap: var(--gap); }

.left {
  display: grid;
  grid-template-rows: auto auto;
  gap: var(--gap);
  height: 500px;
  position: sticky;
  top: 12px;
}

.box {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--card-radius);
  padding: 14px;
}

.right.box { min-height: 500px; overflow: auto; }

.row { display: flex; align-items: center; gap: 10px; }
.vstack { display: grid; gap: 6px; }
.spacer { flex: 1; }
.muted { color: var(--muted); font-size: 12px; }
.tip { margin-top: 6px; }

.control {
  height: 32px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0 10px;
  background: #fff;
  color: var(--text);
}

.head-row { margin-bottom: 6px; }

.summary {
  display: grid;
  gap: 6px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: #f9fafb;
  padding: 10px 12px;
  margin-bottom: 10px;
}

/* 列表 */
.list { display: grid; gap: 10px; }

.item {
  display: grid;
  grid-template-columns: 150px 1fr;
  align-items: center;
  gap: 14px;
  padding: 12px;
  background: #fff;
  border: 1px dashed var(--border);
  border-radius: 10px;
  transition: background .15s ease, border-color .15s ease;
}

.item:hover { background: #f9fafb; border-color: var(--border); }

.item .date { color: var(--text); font-weight: 600; }

.item .nums { display: flex; gap: 16px; flex-wrap: wrap; align-items: baseline; }

.num {
  font-feature-settings: "tnum";
  letter-spacing: .2px;
  font-variant-numeric: tabular-nums;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.pos { color: #16a34a; }
.neg { color: #ef4444; }

.empty { padding: 24px 0; }

@media (max-width: 900px) {
  .grid { grid-template-columns: 1fr; }
  .right.box { min-height: auto; }
  .item { grid-template-columns: 1fr; }
}
</style>
