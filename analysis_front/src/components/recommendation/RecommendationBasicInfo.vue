<template>
  <div class="info-card">
    <div class="card-header">
      <h3>📊 推荐概览</h3>
      <div class="rank-badge">
        <span class="rank-number">#{{ recommendationData?.rank || 'N/A' }}</span>
      </div>
    </div>
    <div class="card-body">
      <div class="info-grid">
        <div class="info-item">
          <div class="metric">
            <span class="label">综合评分</span>
            <span class="value">{{ (recommendationData?.overall_score * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: (recommendationData?.overall_score * 100) + '%' }"></div>
          </div>
        </div>

        <div class="info-item">
          <div class="metric">
            <span class="label">预期收益</span>
            <span class="value">{{ (recommendationData?.expected_return * 100)?.toFixed(2) || 'N/A' }}%</span>
          </div>
          <div class="change" :class="getChangeClass(recommendationData?.expected_return)">
            {{ recommendationData?.expected_return >= 0 ? '+' : '' }}{{ (recommendationData?.expected_return * 100)?.toFixed(2) || '0.00' }}%
          </div>
        </div>

        <div class="info-item">
          <div class="metric">
            <span class="label">风险评分</span>
            <span class="value">{{ (recommendationData?.risk_score * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="risk-indicator" :class="getRiskLevelClass(recommendationData?.risk_score)">
            {{ getRiskIcon(recommendationData?.risk_level) }} {{ getRiskText(recommendationData?.risk_level) }}
          </div>
        </div>

        <div class="info-item">
          <div class="metric">
            <span class="label">技术评分</span>
            <span class="value">{{ (recommendationData?.technical_score * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill technical" :style="{ width: (recommendationData?.technical_score * 100) + '%' }"></div>
          </div>
        </div>

        <div class="info-item">
          <div class="metric">
            <span class="label">基本面评分</span>
            <span class="value">{{ (recommendationData?.fundamental_score * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill fundamental" :style="{ width: (recommendationData?.fundamental_score * 100) + '%' }"></div>
          </div>
        </div>

        <div class="info-item">
          <div class="metric">
            <span class="label">情绪评分</span>
            <span class="value">{{ (recommendationData?.sentiment_score * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill sentiment" :style="{ width: (recommendationData?.sentiment_score * 100) + '%' }"></div>
          </div>
        </div>
      </div>

      <div class="reasons-section">
        <h4>推荐理由</h4>
        <ul class="reasons-list">
          <li v-for="reason in recommendationData?.reasons" :key="reason" class="reason-item">
            {{ reason }}
          </li>
        </ul>
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

const getChangeClass = (change) => {
  if (!change) return ''
  return change >= 0 ? 'positive' : 'negative'
}

const getRiskLevelClass = (score) => {
  if (!score) return 'low'
  if (score >= 0.7) return 'critical'
  if (score >= 0.5) return 'high'
  if (score >= 0.3) return 'medium'
  return 'low'
}

const getRiskIcon = (level) => {
  const icons = {
    'low': '🟢',
    'medium': '🟡',
    'high': '🟠',
    'critical': '🔴'
  }
  return icons[level] || '⚪'
}

const getRiskText = (level) => {
  const texts = {
    'low': '低风险',
    'medium': '中等风险',
    'high': '高风险',
    'critical': '极高风险'
  }
  return texts[level] || level
}
</script>

<style scoped lang="scss">
.info-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  margin-bottom: 24px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  overflow: hidden;

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

    .rank-badge {
      background: linear-gradient(135deg, #ffd700, #ffb347);
      color: #333;
      padding: 8px 16px;
      border-radius: 20px;
      font-weight: 600;

      .rank-number {
        font-size: 16px;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
    margin-bottom: 30px;

    .info-item {
      padding: 20px;
      background: #f8f9fa;
      border-radius: 12px;
      border-left: 4px solid #667eea;

      .metric {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;

        .label {
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }

        .value {
          font-size: 24px;
          font-weight: 700;
          color: #1a1a1a;
        }
      }

      .progress-bar {
        height: 8px;
        background: #e5e7eb;
        border-radius: 4px;
        overflow: hidden;

        .progress-fill {
          height: 100%;
          background: linear-gradient(90deg, #667eea, #764ba2);
          border-radius: 4px;
          transition: width 0.3s ease;

          &.technical { background: linear-gradient(90deg, #10b981, #059669); }
          &.fundamental { background: linear-gradient(90deg, #f59e0b, #d97706); }
          &.sentiment { background: linear-gradient(90deg, #8b5cf6, #7c3aed); }
        }
      }

      .change {
        font-size: 14px;
        font-weight: 600;
        padding: 4px 8px;
        border-radius: 12px;

        &.positive {
          color: #10b981;
          background: #dcfce7;
        }

        &.negative {
          color: #ef4444;
          background: #fee2e2;
        }
      }

      .risk-indicator {
        font-size: 12px;
        font-weight: 500;
        padding: 4px 8px;
        border-radius: 12px;
        text-align: center;

        &.low {
          color: #10b981;
          background: #dcfce7;
        }

        &.medium {
          color: #f59e0b;
          background: #fef3c7;
        }

        &.high {
          color: #ef4444;
          background: #fee2e2;
        }

        &.critical {
          color: #7f1d1d;
          background: #fef2f2;
        }
      }
    }
  }

  .reasons-section {
    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }

    .reasons-list {
      .reason-item {
        padding: 12px 16px;
        background: #f8f9fa;
        border-radius: 8px;
        margin-bottom: 8px;
        border-left: 3px solid #667eea;
        font-size: 14px;
        line-height: 1.5;
      }
    }
  }
}
</style>
