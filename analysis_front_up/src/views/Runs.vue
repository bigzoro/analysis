<template>
  <section class="panel">
    <div class="row">
      <h2>运行记录</h2>
      <div class="spacer"></div>
      <label>交易所：</label>
      <select v-model="entity" @change="load">
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>
      <button class="primary" @click="load">刷新</button>
    </div>
  </section>

  <section style="margin-top:12px;" class="panel">
    <table class="tbl">
      <thead>
      <tr>
        <th>运行 ID</th>
        <th>交易所</th>
        <th>统计时点（UTC）</th>
        <th>入库时间</th>
        <th>资产总额（USD）</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="r in runs" :key="r.run_id">
        <td style="font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px;">{{ r.run_id }}</td>
        <td>{{ r.entity }}</td>
        <td>{{ new Date(r.as_of).toLocaleString() }}</td>
        <td>{{ new Date(r.created_at).toLocaleString() }}</td>
        <td>{{ r.total_usd }}</td>
      </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api/api.js'

const entities = ref([])
const entity = ref('binance')
const runs = ref([])

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes('binance') && entities.value.length) entity.value = entities.value[0]
}
async function load() {
  const r = await api.listRuns(entity.value)
  runs.value = (r.runs || []).map(x => ({ ...x, total_usd: Number.parseFloat(x.total_usd || '0').toLocaleString() }))
}

onMounted(async () => { await initEntities(); await load() })
</script>

<style scoped lang="scss">
.tbl { width:100%; border-collapse:collapse;
  th,td{ padding:10px; border-bottom:1px solid var(--border); }
  th{ color:var(--muted); text-align:left; }
  tbody tr:hover{ background:#121722; }
}
</style>
