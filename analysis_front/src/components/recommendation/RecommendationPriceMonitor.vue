<template>
  <div class="price-monitor-card">
    <div class="card-header">
      <h3>💰 实时价格监控</h3>
      <div class="price-update-info">
        <span class="update-time">最后更新: {{ formatTime(new Date()) }}</span>
        <button @click="$emit('refreshPrice')" class="refresh-btn" :disabled="priceLoading">
          {{ priceLoading ? '刷新中...' : '🔄 刷新' }}
        </button>
      </div>
    </div>
    <div class="card-body">
      <div class="current-price-section">
        <div class="price-display">
          <div class="main-price">
            <span class="price-value">${{ currentPrice?.toFixed(2) || '加载中...' }}</span>
            <span class="price-change" :class="getPriceChangeClass(currentPriceChange)">
              {{ currentPriceChange >= 0 ? '+' : '' }}{{ currentPriceChange?.toFixed(2) || '0.00' }}%
            </span>
          </div>
          <div class="price-details">
            <div class="price-item">
              <span class="label">24h 最高</span>
              <span class="value">${{ priceRanges?.high_24h?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="price-item">
              <span class="label">24h 最低</span>
              <span class="value">${{ priceRanges?.low_24h?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="price-item">
              <span class="label">24h 成交量</span>
              <span class="value">{{ formatVolume(priceData?.volume_24h) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  currentPrice: Number,
  currentPriceChange: Number,
  priceRanges: Object,
  priceData: Object,
  priceLoading: Boolean
})

const emit = defineEmits(['refreshPrice'])

const formatTime = (date) => {
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const getPriceChangeClass = (change) => {
  if (!change) return ''
  if (change > 0) return 'positive'
  if (change < 0) return 'negative'
  return 'neutral'
}

const formatVolume = (volume) => {
  if (!volume) return 'N/A'
  if (volume >= 1e9) return (volume / 1e9).toFixed(2) + 'B'
  if (volume >= 1e6) return (volume / 1e6).toFixed(2) + 'M'
  if (volume >= 1e3) return (volume / 1e3).toFixed(2) + 'K'
  return volume.toString()
}
</script>

<style scoped lang="scss">
.price-monitor-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  margin-bottom: 24px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);

  .card-header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 20px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;

    h3 {
      margin: 0;
      font-size: 20px;
      font-weight: 600;
    }

    .price-update-info {
      display: flex;
      align-items: center;
      gap: 16px;

      .update-time {
        font-size: 12px;
        opacity: 0.9;
      }

      .refresh-btn {
        padding: 6px 12px;
        background: rgba(255, 255, 255, 0.2);
        color: white;
        border: none;
        border-radius: 6px;
        cursor: pointer;
        font-size: 12px;
        transition: background 0.3s;

        &:hover:not(:disabled) {
          background: rgba(255, 255, 255, 0.3);
        }

        &:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .current-price-section {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 24px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2);
    border-radius: 12px;
    color: white;

    .price-display {
      text-align: center;

      .main-price {
        margin-bottom: 16px;

        .price-value {
          font-size: 36px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .price-change {
          font-size: 18px;
          font-weight: 600;
          padding: 4px 12px;
          border-radius: 20px;
          background: rgba(255, 255, 255, 0.2);

          &.positive {
            background: rgba(16, 185, 129, 0.8);
          }

          &.negative {
            background: rgba(239, 68, 68, 0.8);
          }
        }
      }

      .price-details {
        display: flex;
        gap: 24px;
        justify-content: center;

        .price-item {
          text-align: center;

          .label {
            font-size: 12px;
            opacity: 0.8;
            margin-bottom: 4px;
          }

          .value {
            font-size: 14px;
            font-weight: 600;
          }
        }
      }
    }
  }
}
</style>
