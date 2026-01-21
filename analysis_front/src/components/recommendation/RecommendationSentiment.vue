<template>
  <div class="sentiment-analysis-card">
    <div class="card-header">
      <h3>😊 市场情绪分析</h3>
      <div class="sentiment-score" :class="getSentimentClass(sentimentData?.overall_score)">
        {{ sentimentData?.overall_score?.toFixed(1) || 'N/A' }}
      </div>
    </div>
    <div class="card-body">
      <div class="sentiment-overview">
        <div class="sentiment-metrics">
          <div class="sentiment-item">
            <span class="sentiment-label">恐惧贪婪指数:</span>
            <span class="sentiment-value">{{ sentimentData?.fear_greed_index || 'N/A' }}/100</span>
            <span class="sentiment-status">{{ getFearGreedStatus(sentimentData?.fear_greed_index) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  sentimentData: {
    type: Object,
    default: () => ({})
  }
})

const getSentimentClass = (score) => {
  if (!score) return ''
  if (score >= 70) return 'bullish'
  if (score >= 40) return 'neutral'
  return 'bearish'
}

const getFearGreedStatus = (index) => {
  if (!index) return '未知'
  if (index >= 75) return '极度贪婪'
  if (index >= 55) return '贪婪'
  if (index >= 45) return '中性'
  if (index >= 25) return '恐惧'
  return '极度恐惧'
}
</script>

<style scoped lang="scss">
.sentiment-analysis-card {
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

    .sentiment-score {
      padding: 8px 16px;
      border-radius: 20px;
      font-size: 16px;
      font-weight: 600;

      &.bullish {
        background: #dcfce7;
        color: #166534;
      }

      &.neutral {
        background: #fef3c7;
        color: #92400e;
      }

      &.bearish {
        background: #fee2e2;
        color: #991b1b;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .sentiment-overview {
    margin-bottom: 30px;

    .sentiment-metrics {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 16px;

      .sentiment-item {
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;
        display: flex;
        justify-content: space-between;
        align-items: center;

        .sentiment-label {
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }

        .sentiment-value {
          font-size: 16px;
          font-weight: 600;
          color: #1a1a1a;
          margin-right: 12px;
        }

        .sentiment-status {
          font-size: 12px;
          padding: 4px 8px;
          border-radius: 12px;
          font-weight: 500;
        }
      }
    }
  }
}
</style>
