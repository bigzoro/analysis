<template>
  <div ref="el" class="chart"></div>
</template>

<script setup>
import * as echarts from 'echarts'
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  xData: { type: Array, default: () => [] },
  series: { type: Array, default: () => [] },
  title: { type: String, default: '' },
  yLabel: { type: String, default: '' },
})

const el = ref(null)
let chart

const render = () => {
  if (!chart) return
  chart.setOption({
    title: { text: props.title, left: 'center', textStyle: { color: '#e7ecf2', fontSize: 14 } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    legend: { top: 32, textStyle: { color: '#98a2b3' } },
    grid: { left: 40, right: 20, top: 70, bottom: 40 },
    dataZoom: [{ type: 'inside' }, { type: 'slider', height: 14, bottom: 6 }],
    xAxis: { type: 'category', data: props.xData, boundaryGap: false, axisLabel: { color: '#98a2b3' }, axisLine: { lineStyle: { color: '#2a3242' } }},
    yAxis: { type: 'value', name: props.yLabel, axisLabel: { color: '#98a2b3' }, splitLine: { lineStyle: { color: '#1f2836' } } },
    series: props.series.map(s => ({ ...s, type: 'line', showSymbol: false, smooth: true })),
    backgroundColor: 'transparent',
  }, true)
  chart.resize()
}

onMounted(() => {
  chart = echarts.init(el.value)
  render()
  const ro = new ResizeObserver(() => chart && chart.resize())
  ro.observe(el.value)
  el.value._ro = ro
})
onBeforeUnmount(() => {
  if (el.value && el.value._ro) el.value._ro.disconnect()
  if (chart) chart.dispose()
})
watch(() => [props.xData, props.series, props.title], render, { deep: true })
</script>

<style scoped lang="scss">
.chart { width: 100%; height: 520px; }
</style>
