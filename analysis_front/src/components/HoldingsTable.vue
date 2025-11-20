<template>
  <div class="panel">
    <div class="row" style="margin-bottom:10px;">
      <strong>持仓</strong>
      <div class="spacer"></div>
      <span class="badge">总额：{{ totalUSD }}</span>
    </div>
    <table class="table table-holdings">
      <thead>
      <tr>
        <th @click="sortBy('chain')">网络</th>
        <th @click="sortBy('symbol')">币种</th>
        <th @click="sortBy('amountNum')">数量</th>
        <th @click="sortBy('value_usd')">金额（USD）</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(h, i) in sorted" :key="i">
        <td>{{ h.chain }}</td>
        <td>{{ h.symbol }}</td>
        <td>{{ fmtAmount(h.amountNum, 8) }}</td>
        <td>{{ fmtUSD(h.value_usd) }}</td>
      </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { fmtUSD, fmtAmount } from '../utils/utils.js'

const props = defineProps({ holdings: { type: Array, default: () => [] }, total: { type: Number, default: 0 } })
const sort = ref({ key: 'value_usd', asc: false })
const rows = computed(() => (props.holdings || []).map(h => ({ ...h, amountNum: parseFloat(h.amount || '0') })))
const sorted = computed(() => {
  const arr = [...rows.value]
  arr.sort((a, b) => {
    const k = sort.value.key, av = a[k] ?? 0, bv = b[k] ?? 0
    if (av === bv) return 0
    return (av > bv ? 1 : -1) * (sort.value.asc ? 1 : -1)
  })
  return arr
})
const totalUSD = computed(() => fmtUSD(props.total))
function sortBy(k){ if (sort.value.key===k) sort.value.asc=!sort.value.asc; else { sort.value.key=k; sort.value.asc=true } }
</script>

<style scoped lang="scss">
.table{ width:100%; border-collapse:separate; border-spacing:0; font-size:14px; }
.table th, .table td{ padding:10px 12px; border-bottom:1px solid var(--border); }
.table thead th{ color:#374151; font-weight:600; text-align:left; cursor:pointer; }
.table tbody tr:hover{ background:#f9fafb; }

/* 右对齐数量/金额 */
.table-holdings th:nth-child(3),
.table-holdings td:nth-child(3),
.table-holdings th:nth-child(4),
.table-holdings td:nth-child(4){
  text-align:right;
  font-variant-numeric: tabular-nums;
  font-family: ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
}
</style>
