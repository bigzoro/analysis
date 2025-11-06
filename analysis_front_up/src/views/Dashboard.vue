<template>
  <section class="panel">
    <div class="row">
      <h2>仪表盘</h2>
      <div class="spacer"></div>
      <label>交易所：</label>
      <select v-model="entity" @change="load">
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>
      <button class="primary" @click="load">刷新</button>
    </div>
  </section>

  <section style="margin-top:12px;" class="panel">
    <div class="row">
      <div class="badge">运行批次：{{ data?.run_id || '-' }}</div>
      <div class="spacer"></div>
      <div class="badge">统计时点：{{ data?.as_of ? new Date(data.as_of).toLocaleString() : '-' }}</div>
      <div class="badge">资产总额（USD）：{{ data ? usd(data.total_usd) : '-' }}</div>
    </div>
  </section>

  <section style="margin-top:12px;">
    <HoldingsTable v-if="data" :holdings="data.holdings" :total="data.total_usd" />
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api/api.js'
import { fmtUSD as usd } from '../utils/utils.js'
import HoldingsTable from '../components/HoldingsTable.vue'

const entities = ref([])
const entity = ref('binance')
const data = ref(null)

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes('binance') && entities.value.length) entity.value = entities.value[0]
}
async function load() { data.value = await api.latestPortfolio(entity.value) }

onMounted(async () => { await initEntities(); await load() })
</script>
