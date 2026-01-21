<template>
  <div class="box-plot-chart" :style="{ height: height + 'px' }">
    <div class="chart-placeholder">
      <div class="placeholder-icon">📊</div>
      <div class="placeholder-text">箱线图</div>
      <div class="placeholder-desc">波动率分布对比分析</div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'BoxPlotChart',
  props: {
    data: {
      type: Array,
      default: () => []
    },
    height: {
      type: Number,
      default: 300
    }
  }
}
</script>

<style scoped>
.box-plot-chart {
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
