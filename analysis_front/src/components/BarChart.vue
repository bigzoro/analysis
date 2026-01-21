<template>
  <div class="bar-chart" :style="{ height: height + 'px' }">
    <div class="chart-placeholder">
      <div class="placeholder-icon">📊</div>
      <div class="placeholder-text">柱状图</div>
      <div class="placeholder-desc">数据对比分析</div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'BarChart',
  props: {
    data: {
      type: Object,
      default: () => ({ xData: [], series: [] })
    },
    options: {
      type: Object,
      default: () => ({})
    },
    height: {
      type: Number,
      default: 300
    }
  }
}
</script>

<style scoped>
.bar-chart {
  background: #f9fafb;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px dashed #d1d5db;
}

.chart-placeholder {
  text-align: center;
  color: #6b7280;
}

.placeholder-icon {
  font-size: 3rem;
  margin-bottom: 8px;
  opacity: 0.6;
}

.placeholder-text {
  font-size: 1.2rem;
  font-weight: 600;
  margin-bottom: 4px;
}

.placeholder-desc {
  font-size: 0.9rem;
  opacity: 0.8;
}
</style>
