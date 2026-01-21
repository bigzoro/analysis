<template>
  <div ref="el" class="chart" :style="{ width: '100%', height: '100%' }"></div>
</template>

<script setup>
import * as echarts from 'echarts'
import { onMounted, onBeforeUnmount, ref, watch, nextTick } from 'vue'

const props = defineProps({
  xData: { type: Array, default: () => [] },
  series: { type: Array, default: () => [] },
  title: { type: String, default: '' },
  yLabel: { type: String, default: '' },
  yAxis: { type: [Object, Array], default: null },
})

const el = ref(null)
let chart
let isInitialized = false

// 缓存基础配置，避免重复创建
const getBaseOption = () => ({
  title: { text: props.title, left: 'center', textStyle: { color: '#e7ecf2', fontSize: 14 } },
  tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
  legend: { top: 20, textStyle: { color: '#98a2b3' } },
  grid: { left: 40, right: 20, top: 50, bottom: 30 },
  dataZoom: [{ type: 'inside' }, { type: 'slider', height: 12, bottom: 2 }],
  backgroundColor: 'transparent',
})

// 优化的渲染函数
const render = async () => {
  if (!chart || !isInitialized) return

  await nextTick()

  try {
    // 检查数据是否有效
    if (!props.xData || props.xData.length === 0) {
      console.warn('LineChart: xData is empty, skipping render')
      return
    }

    if (!props.series || props.series.length === 0) {
      console.warn('LineChart: series is empty, skipping render')
      return
    }

    const option = {
      ...getBaseOption(),
      xAxis: {
        type: 'category',
        data: props.xData,
        boundaryGap: false,
        axisLabel: { color: '#98a2b3' },
        axisLine: { lineStyle: { color: '#2a3242' } }
      },
      yAxis: props.yAxis || {
        type: 'value',
        name: props.yLabel,
        axisLabel: { color: '#98a2b3' },
        splitLine: { lineStyle: { color: '#1f2836' } }
      },
      series: props.series.map(s => ({
        ...s,
        type: 'line',
        showSymbol: false,
        smooth: true,
        // 优化：减少不必要的重绘
        animation: props.series.length <= 5, // 少量系列时启用动画
      })),
    }

    // 使用notMerge=false来优化性能，避免不必要的合并
    chart.setOption(option, false, false)

    // 延迟resize调用，避免在主进程中调用
    setTimeout(() => {
      if (chart && !chart.isDisposed()) {
        chart.resize()
      }
    }, 0)

  } catch (error) {
    console.error('LineChart render error:', error)
  }
}

// 防抖更新，避免频繁重绘
let updateTimer = null
const debouncedRender = () => {
  if (updateTimer) clearTimeout(updateTimer)
  updateTimer = setTimeout(() => {
    render()
  }, 100) // 100ms防抖
}

onMounted(async () => {
  if (!el.value) return

  try {
    // 使用canvas渲染以提高性能
    chart = echarts.init(el.value, null, {
      renderer: 'canvas', // 强制使用canvas渲染
      devicePixelRatio: window.devicePixelRatio || 1,
      width: el.value.clientWidth,
      height: 520,
    })

    // 设置初始化标志
    isInitialized = true

    // 初始渲染
    await render()

    // 优化：使用防抖的resize监听
    let resizeTimer = null
    const ro = new ResizeObserver(() => {
      if (resizeTimer) clearTimeout(resizeTimer)
      resizeTimer = setTimeout(() => {
        if (chart && !chart.isDisposed()) {
          chart.resize()
        }
      }, 200)
    })
    ro.observe(el.value)
    el.value._ro = ro

  } catch (error) {
    console.error('LineChart initialization error:', error)
  }
})

onBeforeUnmount(() => {
  // 清理定时器
  if (updateTimer) {
    clearTimeout(updateTimer)
    updateTimer = null
  }

  // 清理resize观察器
  if (el.value && el.value._ro) {
    el.value._ro.disconnect()
  }

  // 销毁图表实例
  if (chart && !chart.isDisposed()) {
    chart.dispose()
    chart = null
  }
})

// 优化监听：分别监听不同类型的属性变化
watch(() => props.title, render, { immediate: false }) // title变化时重绘
watch(() => props.yLabel, render, { immediate: false }) // y轴标签变化时重绘
watch(() => props.yAxis, render, { immediate: false, deep: true }) // y轴配置变化时重绘

// 数据变化使用防抖更新
watch(() => [props.xData, props.series], debouncedRender, {
  deep: false, // 避免深度监听，提高性能
  immediate: false
})
</script>

<style scoped>
.chart {
  width: 100%;
  height: 100%;
  min-height: 200px;
}
</style>
