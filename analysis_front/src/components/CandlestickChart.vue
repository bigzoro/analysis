<template>
  <div ref="el" class="candlestick-chart" :style="{ width: '100%', height: '100%' }"></div>
</template>

<script setup>
import * as echarts from 'echarts'
import { onMounted, onBeforeUnmount, ref, watch, nextTick } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] },
  title: { type: String, default: '' },
  showVolume: { type: Boolean, default: true },
  showMA: { type: Boolean, default: true },
  qualityThreshold: { type: Number, default: 80 }, // 数据质量阈值
})

const el = ref(null)
let chart
let isInitialized = false

// 计算K线数据，包含质量检查
const calculateOHLC = (data) => {
  if (!data || data.length === 0) return []


  const processed = data.map(item => {
    const open = parseFloat(item.open) || parseFloat(item.price) || 0
    const high = parseFloat(item.high) || parseFloat(item.price) || 0
    const low = parseFloat(item.low) || parseFloat(item.price) || 0
    const close = parseFloat(item.close) || parseFloat(item.price) || 0

    // 检查数据质量
    const isValid = item.is_valid !== false // 默认认为是有效的
    const quality = item.data_quality || 100

    return {
      ohlc: [open, close, low, high],
      volume: parseFloat(item.volume) || 0,
      timestamp: item.timestamp,
      isValid: isValid,
      quality: quality,
      // 标记低质量数据
      isLowQuality: quality < props.qualityThreshold
    }
  })

  return processed
}

// 计算移动平均线 - 使用处理后的数据
const calculateMA = (processedData, period) => {
  if (!processedData || processedData.length < period) {
    return []
  }

  const ma = []
  for (let i = period - 1; i < processedData.length; i++) {
    let sum = 0
    let validCount = 0
    for (let j = 0; j < period; j++) {
      // 使用处理后的数据，close值在ohlc[1]位置
      const close = processedData[i - j].ohlc[1] // ohlc: [open, close, low, high]
      if (close > 0) {
        sum += close
        validCount++
      }
    }
    // 只有当有足够的有效数据时才计算MA
    if (validCount >= period) {
      ma.push(sum / period)
    } else {
      ma.push(null)
    }
  }

  return ma
}

// 渲染图表
const render = async () => {
  if (!chart || !isInitialized || !props.data.length) return

  await nextTick()

  try {
    const processedData = calculateOHLC(props.data)
    const ohlcData = processedData.map(item => item.ohlc)
    const volumeData = processedData.map(item => item.volume)


    // 计算技术指标 - 使用处理后的数据
    let ma5 = [], ma10 = []
    if (props.showMA) {
      ma5 = calculateMA(processedData, 5)
      ma10 = calculateMA(processedData, 10)

    }

    // 准备时间轴数据
    const dates = props.data.map(item => {
      const timestamp = item.timestamp * 1000
      return new Date(timestamp).toLocaleTimeString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    })

    // 为低质量数据着色
    const klineColors = processedData.map(item => ({
      color: item.isLowQuality ? '#fbbf24' : '#22c55e',      // 低质量用黄色
      color0: item.isLowQuality ? '#f59e0b' : '#ef4444',     // 低质量用橙色
      borderColor: item.isLowQuality ? '#fbbf24' : '#22c55e',
      borderColor0: item.isLowQuality ? '#f59e0b' : '#ef4444'
    }))

    const series = []

    // K线系列
    series.push({
      name: 'K线',
      type: 'candlestick',
      data: ohlcData,
      itemStyle: {
        color: '#22c55e',      // 阳线填充色
        color0: '#ef4444',     // 阴线填充色
        borderColor: '#22c55e', // 阳线边框色
        borderColor0: '#ef4444' // 阴线边框色
      },
      animation: false
    })

    // 成交量柱状图
    if (props.showVolume) {
      series.push({
        name: '成交量',
        type: 'bar',
        data: volumeData,
        xAxisIndex: 1,
        yAxisIndex: 1,
        itemStyle: {
          color: '#8884d8' // 统一的紫色，便于识别
        },
        animation: false
      })
    }

    // 移动平均线
    if (props.showMA) {
      // 为MA线数据填充前面的null值，使其与K线数据长度一致
      const padData = (maData, period) => {
        const padded = new Array(period - 1).fill(null) // 前period-1个点填充null
        return padded.concat(maData)
      }

      if (ma5.length > 0) {
        const paddedMa5 = padData(ma5, 5)
        series.push({
          name: 'MA5',
          type: 'line',
          data: paddedMa5,
          smooth: true,
          lineStyle: { color: '#3b82f6', width: 2 }, // 增加线宽便于观察
          showSymbol: false,
          animation: false,
          connectNulls: true // 连接null值，让折线连续
        })
      }

      if (ma10.length > 0) {
        const paddedMa10 = padData(ma10, 10)
        series.push({
          name: 'MA10',
          type: 'line',
          data: paddedMa10,
          smooth: true,
          lineStyle: { color: '#f59e0b', width: 2 }, // 增加线宽便于观察
          showSymbol: false,
          animation: false,
          connectNulls: true // 连接null值，让折线连续
        })
      }
    }

    const option = {
      title: {
        text: props.title,
        left: 'center',
        top: 5,
        textStyle: {
          color: '#ffffff',
          fontSize: 14,
          fontWeight: 'bold'
        }
      },
      legend: {
        data: ['K线', 'MA5', 'MA10', '成交量'].filter(name => {
          if (name === 'MA5') return ma5.length > 0
          if (name === 'MA10') return ma10.length > 0
          if (name === '成交量') return props.showVolume
          return true
        }),
        top: 30,
        textStyle: { color: '#98a2b3' }
      },
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        formatter: function (params) {
          let result = `<div style="font-weight: bold;">${params[0].name}</div>`
          let hasKline = false, hasVolume = false

          params.forEach(param => {
            if (param.seriesName === 'K线') {
              hasKline = true
              const data = param.data
              const quality = processedData[param.dataIndex]?.quality || 100
              const qualityText = quality < props.qualityThreshold ? ` (质量: ${quality}%)` : ''

              result += `
                <div>开盘: ${data[0].toFixed(4)}</div>
                <div>收盘: ${data[1].toFixed(4)}</div>
                <div>最低: ${data[2].toFixed(4)}</div>
                <div>最高: ${data[3].toFixed(4)}</div>
                <div>数据质量: ${quality}%${qualityText}</div>
              `
            } else if (param.seriesName === '成交量') {
              hasVolume = true
              result += `<div>成交量: ${param.value.toLocaleString()}</div>`
            } else {
              result += `<div>${param.seriesName}: ${param.value.toFixed(4)}</div>`
            }
          })

          // 如果只有成交量没有K线，添加基本信息
          if (!hasKline && hasVolume && params.length > 0) {
            const dataIndex = params[0].dataIndex
            const ohlc = ohlcData[dataIndex]
            if (ohlc) {
              result += `
                <div>开盘: ${ohlc[0].toFixed(4)}</div>
                <div>收盘: ${ohlc[1].toFixed(4)}</div>
                <div>最低: ${ohlc[2].toFixed(4)}</div>
                <div>最高: ${ohlc[3].toFixed(4)}</div>
              `
            }
          }

          return result
        }
      },
      legend: {
        top: 20,
        textStyle: { color: '#98a2b3' }
      },
      grid: props.showVolume ? [
        { left: 40, right: 20, top: 60, height: '55%' },
        { left: 40, right: 20, top: '78%', height: '18%' }
      ] : { left: 40, right: 20, top: 50, bottom: 30 },
      dataZoom: [
        { type: 'inside', xAxisIndex: props.showVolume ? [0, 1] : [0] },
        { type: 'slider', height: 12, bottom: 2, xAxisIndex: props.showVolume ? [0, 1] : [0] }
      ],
      backgroundColor: 'transparent',
      xAxis: props.showVolume ? [
        {
          type: 'category',
          data: dates,
          axisLabel: { color: '#98a2b3' },
          axisLine: { lineStyle: { color: '#2a3242' } }
        },
        {
          type: 'category',
          data: dates,
          gridIndex: 1,
          axisLabel: { show: false },
          axisLine: { lineStyle: { color: '#2a3242' } }
        }
      ] : {
        type: 'category',
        data: dates,
        axisLabel: { color: '#98a2b3' },
        axisLine: { lineStyle: { color: '#2a3242' } }
      },
      yAxis: props.showVolume ? [
        {
          type: 'value',
          name: '价格 (USDT)',
          axisLabel: { color: '#98a2b3' },
          splitLine: { lineStyle: { color: '#1f2836' } }
        },
        {
          type: 'value',
          name: '成交量',
          gridIndex: 1,
          axisLabel: { color: '#98a2b3' },
          splitLine: { lineStyle: { color: '#1f2836' } }
        }
      ] : {
        type: 'value',
        name: '价格 (USDT)',
        axisLabel: { color: '#98a2b3' },
        splitLine: { lineStyle: { color: '#1f2836' } }
      },
      series: series
    }

    chart.setOption(option, false, false)

    setTimeout(() => {
      if (chart && !chart.isDisposed()) {
        chart.resize()
      }
    }, 0)

  } catch (error) {
    console.error('CandlestickChart render error:', error)
  }
}

// 防抖更新
let updateTimer = null
const debouncedRender = () => {
  if (updateTimer) clearTimeout(updateTimer)
  updateTimer = setTimeout(() => {
    render()
  }, 100)
}

onMounted(async () => {
  if (!el.value) return

  try {
    chart = echarts.init(el.value, null, {
      renderer: 'canvas',
      devicePixelRatio: window.devicePixelRatio || 1,
      width: el.value.clientWidth,
      height: 520,
    })

    isInitialized = true
    await render()

    // 监听resize
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
    console.error('CandlestickChart initialization error:', error)
  }
})

onBeforeUnmount(() => {
  if (updateTimer) {
    clearTimeout(updateTimer)
    updateTimer = null
  }

  if (el.value && el.value._ro) {
    el.value._ro.disconnect()
  }

  if (chart && !chart.isDisposed()) {
    chart.dispose()
    chart = null
  }
})

// 监听数据变化
watch(() => props.data, debouncedRender, { deep: false, immediate: false })
watch(() => props.title, render, { immediate: false })
</script>

<style scoped>
.candlestick-chart {
  width: 100%;
  height: 100%;
  min-height: 200px;
}
</style>