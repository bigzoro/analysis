<template>
  <section class="panel">
    <div class="row">
      <h2>资金流</h2>
      <div class="spacer"></div>

      <label>交易所：</label>
      <select v-model="entity" @change="fetchData">
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>

      <label>模式：</label>
      <select v-model="mode" @change="fetchData">
        <option value="daily">按日</option>
        <option value="weekly">按周</option>
      </select>

      <label>币种：</label>
      <div class="row" style="gap:8px;">
        <label v-for="c in allCoins" :key="c" class="badge">
          <input type="checkbox" v-model="coins" :value="c" @change="fetchData" />
          {{ c }}
        </label>
      </div>

      <label>指标：</label>
      <select v-model="metric" @change="updateChart">
        <option value="net">净流入</option>
        <option value="in">流入</option>
        <option value="out">流出</option>
      </select>

      <template v-if="mode === 'daily'">
        <label>开始：</label><input type="date" v-model="start" @change="fetchData" />
        <label>结束：</label><input type="date" v-model="end" @change="fetchData" />
      </template>
    </div>
  </section>

  <section style="margin-top:12px;" class="panel">
    <LineChart :xData="xData" :series="series" :title="title" yLabel="数量" />
  </section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import dayjs from 'dayjs'
import LineChart from '../components/LineChart.vue'
import { api } from '../api/api.js'

const allCoins = ['BTC','ETH','SOL','USDT','USDC']
const entities = ref([])
const entity = ref('binance')
const mode = ref('daily') // daily | weekly
const coins = ref(['BTC','ETH','SOL','USDT','USDC'])
const metric = ref('net') // net | in | out
const start = ref(dayjs().subtract(30,'day').format('YYYY-MM-DD'))
const end = ref(dayjs().format('YYYY-MM-DD'))
const raw = ref(null)
const xData = ref([])
const series = ref([])

const metricLabel = computed(() => ({ net: '净流入', in: '流入', out: '流出' }[metric.value]))
const modeLabel = computed(() => (mode.value === 'daily' ? '按日' : '按周'))
const title = computed(() => `${modeLabel.value} ${metricLabel.value} - ${entity.value}（${coins.value.join('、')}）`)

function buildSeries() {
  if (!raw.value) { xData.value = []; series.value = []; return }
  const isDaily = mode.value === 'daily'
  const dataObj = raw.value.data || {}
  const xsSet = new Set()
  Object.values(dataObj).forEach(arr => { arr.forEach(r => xsSet.add(isDaily ? r.day : r.week)) })
  const xs = Array.from(xsSet).sort()
  xData.value = xs
  const ser = []
  for (const c of coins.value) {
    const rows = dataObj[c] || []
    const byKey = {}
    rows.forEach(r => { byKey[isDaily ? r.day : r.week] = r })
    ser.push({ name: c, data: xs.map(k => byKey[k]?.[metric.value] ?? null), type: 'line', showSymbol:false, smooth:true })
  }
  series.value = ser
}

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes('binance') && entities.value.length) entity.value = entities.value[0]
}
async function fetchData() {
  const coinParam = coins.value.join(',')
  raw.value = mode.value === 'weekly'
      ? await api.weeklyFlows({ entity: entity.value, coin: coinParam, latest: true })
      : await api.dailyFlows({ entity: entity.value, coin: coinParam, latest: true, start: start.value, end: end.value })
  buildSeries()
}
function updateChart(){ buildSeries() }

onMounted(async () => { await initEntities(); await fetchData() })
</script>
