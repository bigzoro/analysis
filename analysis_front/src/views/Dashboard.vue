<template>
  <div class="container">
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

  <section style="margin-top:12px;" class="">
    <div class="meta-bar">
      <span class="chip">运行批次：{{ data?.run_id || '-' }}</span>
      <span class="chip">统计时点：{{ data?.updated_at || '-' }}</span>
      <span class="chip">资产总额（USD）：{{ usd(data?.total_usd || 0) }}</span>
      <span class="chip">总额：{{ usd(data?.total_usd || 0) }}</span>
    </div>
  </section>

  <section style="margin-top:12px;">
    <HoldingsTable v-if="data" :holdings="data.holdings" :total="data.total_usd" />
  </section>
  </div>
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
