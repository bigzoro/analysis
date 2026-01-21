<template>
  <div class="market-data-card">
    <div class="card-header">
      <h3>🌍 市场数据</h3>
      <div class="market-cap">
        <span class="label">市值:</span>
        <span class="value">${{ formatLargeNumber(recommendationData?.market_data?.market_cap) }}</span>
      </div>
    </div>
    <div class="card-body">
      <div class="market-metrics">
        <div class="metric-item">
          <div class="metric-header">
            <span class="metric-name">24h涨跌幅</span>
            <span class="metric-value" :class="getPriceChangeClass(recommendationData?.market_data?.price_change_24h)">
              {{ recommendationData?.market_data?.price_change_24h >= 0 ? '+' : '' }}{{ recommendationData?.market_data?.price_change_24h?.toFixed(2) || 'N/A' }}%
            </span>
          </div>
          <div class="metric-trend" :class="getChangeClass(recommendationData?.market_data?.price_change_24h)">
            <span class="trend-icon">{{ recommendationData?.market_data?.price_change_24h >= 0 ? '📈' : '📉' }}</span>
            <span class="trend-text">{{ recommendationData?.market_data?.price_change_24h >= 0 ? '上涨' : '下跌' }}</span>
          </div>
        </div>

        <div class="metric-item">
          <div class="metric-header">
            <span class="metric-name">24h成交量</span>
            <span class="metric-value">${{ formatLargeNumber(recommendationData?.market_data?.volume_24h) }}</span>
          </div>
          <div class="volume-comparison">
            <span class="comparison-text">相比昨日</span>
            <span class="comparison-value positive">+12.5%</span>
          </div>
        </div>
      </div>

      <div class="price-ranges">
        <h4>价格区间</h4>
        <div class="ranges-grid">
          <div class="range-item">
            <span class="range-label">24h 最高</span>
            <span class="range-value">${{ recommendationData?.market_data?.price_ranges?.high_24h?.toFixed(2) || 'N/A' }}</span>
          </div>
          <div class="range-item">
            <span class="range-label">24h 最低</span>
            <span class="range-value">${{ recommendationData?.market_data?.price_ranges?.low_24h?.toFixed(2) || 'N/A' }}</span>
          </div>
          <div class="range-item">
            <span class="range-label">7d 最高</span>
            <span class="range-value">${{ recommendationData?.market_data?.price_ranges?.high_7d?.toFixed(2) || 'N/A' }}</span>
          </div>
          <div class="range-item">
            <span class="range-label">7d 最低</span>
            <span class="range-value">${{ recommendationData?.market_data?.price_ranges?.low_7d?.toFixed(2) || 'N/A' }}</span>
          </div>
        </div>
      </div>

      <!-- 当前价格 -->
      <div class="current-price-display">
        <div class="price-info">
          <span class="price-label">当前价格</span>
          <span class="current-price">${{ recommendationData?.price?.toFixed(2) || 'N/A' }}</span>
        </div>
        <div class="price-position">
          <span class="position-text">相对位置</span>
          <div class="position-bar">
            <div class="position-indicator" :style="{ left: (recommendationData?.technical_indicators?.bb_position * 100) + '%' }"></div>
            <div class="position-zones">
              <span class="zone-label low">低估</span>
              <span class="zone-label mid">合理</span>
              <span class="zone-label high">高估</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  recommendationData: {
    type: Object,
    default: () => ({})
  }
})

const getPriceChangeClass = (change) => {
  if (!change) return ''
  if (change > 0) return 'positive'
  if (change < 0) return 'negative'
  return 'neutral'
}

const getChangeClass = (change) => {
  if (!change) return ''
  return change >= 0 ? 'positive' : 'negative'
}

const formatLargeNumber = (num) => {
  if (!num) return 'N/A'
  if (num >= 1e12) return (num / 1e12).toFixed(2) + 'T'
  if (num >= 1e9) return (num / 1e9).toFixed(2) + 'B'
  if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M'
  if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K'
  return num.toString()
}
</script>

<style scoped lang="scss">
.market-data-card {
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

    .market-cap {
      background: linear-gradient(135deg, #f59e0b, #d97706);
      color: white;
      padding: 8px 16px;
      border-radius: 20px;
      font-size: 14px;

      .label {
        opacity: 0.9;
      }

      .value {
        font-weight: 600;
        margin-left: 8px;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .market-metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 20px;

    .metric-item {
      padding: 20px;
      background: #f8f9fa;
      border-radius: 12px;

      .metric-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;

        .metric-name {
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }

        .metric-value {
          font-size: 18px;
          font-weight: 700;
          color: #1a1a1a;
        }
      }

        .metric-trend {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;

        .trend-icon {
          font-size: 16px;
        }

        .trend-text {
          color: #666;
        }

        &.positive .trend-text {
          color: #10b981;
        }

        &.negative .trend-text {
          color: #ef4444;
        }
      }

      .volume-comparison {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;

        .comparison-text {
          color: #666;
        }

        .comparison-value {
          font-weight: 600;

          &.positive {
            color: #10b981;
          }

          &.negative {
            color: #ef4444;
          }
        }
      }
    }
  }

  .price-ranges {
    margin-bottom: 30px;

    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }

    .ranges-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
      gap: 16px;

      .range-item {
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;
        text-align: center;

        .range-label {
          font-size: 12px;
          color: #666;
          margin-bottom: 4px;
          display: block;
        }

        .range-value {
          font-size: 16px;
          font-weight: 600;
          color: #1a1a1a;
        }
      }
    }
  }

  .current-price-display {
    background: linear-gradient(135deg, #667eea 0%, #764ba2);
    color: white;
    padding: 24px;
    border-radius: 12px;

    .price-info {
      text-align: center;
      margin-bottom: 20px;

      .price-label {
        font-size: 14px;
        opacity: 0.9;
        margin-bottom: 8px;
        display: block;
      }

      .current-price {
        font-size: 32px;
        font-weight: 700;
      }
    }

    .price-position {
      .position-text {
        font-size: 12px;
        opacity: 0.9;
        margin-bottom: 8px;
        display: block;
        text-align: center;
      }

      .position-bar {
        position: relative;
        height: 8px;
        background: rgba(255, 255, 255, 0.2);
        border-radius: 4px;
        margin-bottom: 8px;

        .position-indicator {
          position: absolute;
          top: -2px;
          width: 12px;
          height: 12px;
          background: #ffd700;
          border: 2px solid white;
          border-radius: 50%;
          transform: translateX(-50%);
        }
      }

      .position-zones {
        display: flex;
        justify-content: space-between;
        font-size: 10px;
        opacity: 0.9;

        .zone-label {
          &.low { color: #10b981; }
          &.mid { color: #f59e0b; }
          &.high { color: #ef4444; }
        }
      }
    }
  }
}
</style>
