<template>
  <div class="prediction-section" v-if="prediction">
    <div class="prediction-header">
      <h4>价格预测</h4>
      <span class="prediction-badge" :class="getTrendClass(prediction.trend)">
        {{ getTrendText(prediction.trend) }}
      </span>
    </div>

    <!-- 当前价格 -->
    <div class="current-price">
      <span class="price-label">当前价格：</span>
      <span class="price-value">${{ prediction.current_price?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
    </div>

    <!-- 预测时间周期 -->
    <div class="prediction-periods">
      <!-- 24小时预测 -->
      <div class="prediction-card" v-if="prediction.pred_24h">
        <div class="period-header">
          <span class="period-title">24小时预测</span>
          <span class="confidence-badge" :class="getConfidenceClass(prediction.confidence_24h)">
            置信度: {{ prediction.confidence_24h?.toFixed(1) || 0 }}%
          </span>
        </div>
        <div class="prediction-content">
          <div class="predicted-price">
            <span class="price-label">预测价格：</span>
            <span class="price-value">${{ prediction.pred_24h?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
          </div>
          <div class="price-change" :class="getChangeClass(prediction.change_24h)">
            <span class="change-icon">{{ getChangeIcon(prediction.change_24h) }}</span>
            <span class="change-value">{{ Math.abs(prediction.change_24h || 0).toFixed(2) }}%</span>
          </div>
          <div class="price-range" v-if="prediction.range_24h">
            <div class="range-item">
              <span class="range-label">最低：</span>
              <span class="range-value">${{ prediction.range_24h.min?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
            </div>
            <div class="range-item">
              <span class="range-label">最高：</span>
              <span class="range-value">${{ prediction.range_24h.max?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 7天预测 -->
      <div class="prediction-card" v-if="prediction.pred_7d">
        <div class="period-header">
          <span class="period-title">7天预测</span>
          <span class="confidence-badge" :class="getConfidenceClass(prediction.confidence_7d)">
            置信度: {{ prediction.confidence_7d?.toFixed(1) || 0 }}%
          </span>
        </div>
        <div class="prediction-content">
          <div class="predicted-price">
            <span class="price-label">预测价格：</span>
            <span class="price-value">${{ prediction.pred_7d?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
          </div>
          <div class="price-change" :class="getChangeClass(prediction.change_7d)">
            <span class="change-icon">{{ getChangeIcon(prediction.change_7d) }}</span>
            <span class="change-value">{{ Math.abs(prediction.change_7d || 0).toFixed(2) }}%</span>
          </div>
          <div class="price-range" v-if="prediction.range_7d">
            <div class="range-item">
              <span class="range-label">最低：</span>
              <span class="range-value">${{ prediction.range_7d.min?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
            </div>
            <div class="range-item">
              <span class="range-label">最高：</span>
              <span class="range-value">${{ prediction.range_7d.max?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 30天预测 -->
      <div class="prediction-card" v-if="prediction.pred_30d">
        <div class="period-header">
          <span class="period-title">30天预测</span>
          <span class="confidence-badge" :class="getConfidenceClass(prediction.confidence_30d)">
            置信度: {{ prediction.confidence_30d?.toFixed(1) || 0 }}%
          </span>
        </div>
        <div class="prediction-content">
          <div class="predicted-price">
            <span class="price-label">预测价格：</span>
            <span class="price-value">${{ prediction.pred_30d?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
          </div>
          <div class="price-change" :class="getChangeClass(prediction.change_30d)">
            <span class="change-icon">{{ getChangeIcon(prediction.change_30d) }}</span>
            <span class="change-value">{{ Math.abs(prediction.change_30d || 0).toFixed(2) }}%</span>
          </div>
          <div class="price-range" v-if="prediction.range_30d">
            <div class="range-item">
              <span class="range-label">最低：</span>
              <span class="range-value">${{ prediction.range_30d.min?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
            </div>
            <div class="range-item">
              <span class="range-label">最高：</span>
              <span class="range-value">${{ prediction.range_30d.max?.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 }) || '-' }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 预测依据 -->
    <div class="prediction-factors" v-if="prediction.factors && prediction.factors.length > 0">
      <div class="factors-title">📊 预测依据：</div>
      <ul class="factors-list">
        <li v-for="(factor, idx) in prediction.factors" :key="idx">{{ factor }}</li>
      </ul>
    </div>

    <!-- 预测时间 -->
    <div class="prediction-time" v-if="prediction.predicted_at">
      <span class="time-label">预测时间：</span>
      <span class="time-value">{{ formatTime(prediction.predicted_at) }}</span>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  prediction: {
    type: Object,
    default: null
  }
})

function formatTime(timeStr) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

function getTrendText(trend) {
  switch (trend) {
    case 'bullish':
      return '看涨'
    case 'bearish':
      return '看跌'
    case 'neutral':
      return '中性'
    default:
      return '未知'
  }
}

function getTrendClass(trend) {
  switch (trend) {
    case 'bullish':
      return 'trend-bullish'
    case 'bearish':
      return 'trend-bearish'
    case 'neutral':
      return 'trend-neutral'
    default:
      return ''
  }
}

function getChangeClass(change) {
  if (change > 0) return 'change-positive'
  if (change < 0) return 'change-negative'
  return 'change-neutral'
}

function getChangeIcon(change) {
  if (change > 0) return '↑'
  if (change < 0) return '↓'
  return '→'
}

function getConfidenceClass(confidence) {
  if (confidence >= 70) return 'confidence-high'
  if (confidence >= 50) return 'confidence-medium'
  return 'confidence-low'
}
</script>

<style scoped>
.prediction-section {
  margin-top: 20px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.prediction-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 2px solid #e0e0e0;
}

.prediction-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.prediction-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.prediction-badge.trend-bullish {
  background: #e8f5e9;
  color: #2e7d32;
}

.prediction-badge.trend-bearish {
  background: #ffebee;
  color: #c62828;
}

.prediction-badge.trend-neutral {
  background: #f5f5f5;
  color: #616161;
}

.current-price {
  margin-bottom: 20px;
  padding: 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e0e0e0;
}

.price-label {
  font-size: 14px;
  color: #666;
  margin-right: 8px;
}

.price-value {
  font-size: 20px;
  font-weight: 600;
  color: #333;
}

.prediction-periods {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.prediction-card {
  padding: 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.period-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}

.period-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.confidence-badge {
  padding: 2px 8px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 500;
}

.confidence-badge.confidence-high {
  background: #e8f5e9;
  color: #2e7d32;
}

.confidence-badge.confidence-medium {
  background: #fff3e0;
  color: #f57c00;
}

.confidence-badge.confidence-low {
  background: #fce4ec;
  color: #c2185b;
}

.prediction-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.predicted-price {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.predicted-price .price-value {
  font-size: 18px;
  font-weight: 600;
  color: #1976d2;
}

.price-change {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
}

.price-change.change-positive {
  background: #e8f5e9;
  color: #2e7d32;
}

.price-change.change-negative {
  background: #ffebee;
  color: #c62828;
}

.price-change.change-neutral {
  background: #f5f5f5;
  color: #616161;
}

.change-icon {
  font-size: 16px;
}

.price-range {
  display: flex;
  gap: 16px;
  padding-top: 8px;
  border-top: 1px solid #f0f0f0;
}

.range-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.range-label {
  font-size: 12px;
  color: #999;
}

.range-value {
  font-size: 14px;
  font-weight: 500;
  color: #666;
}

.prediction-factors {
  margin-top: 20px;
  padding: 16px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e0e0e0;
}

.factors-title {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.factors-list {
  margin: 0;
  padding-left: 20px;
  list-style: none;
}

.factors-list li {
  font-size: 13px;
  color: #666;
  line-height: 1.8;
  position: relative;
  padding-left: 12px;
}

.factors-list li::before {
  content: '•';
  position: absolute;
  left: 0;
  color: #1976d2;
  font-weight: bold;
}

.prediction-time {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e0e0e0;
  font-size: 12px;
  color: #999;
  text-align: right;
}

.time-label {
  margin-right: 4px;
}
</style>

