<template>
  <div class="strategy-card">
    <div class="card-header">
      <h3>🎯 交易策略</h3>
      <div class="strategy-type-badge" :class="'strategy-' + (recommendationData?.trading_strategy?.strategy_type?.toLowerCase() || 'range')">
        {{ getStrategyTypeText(recommendationData?.trading_strategy?.strategy_type) }}
      </div>
    </div>
    <div class="card-body">
      <div class="strategy-overview">
        <div class="strategy-info">
          <div class="info-row">
            <span class="label">策略类型:</span>
            <span class="value">{{ getStrategyTypeText(recommendationData?.trading_strategy?.strategy_type) || 'N/A' }}</span>
          </div>
          <div class="info-row">
            <span class="label">市场环境:</span>
            <span class="value">{{ recommendationData?.trading_strategy?.market_condition || 'N/A' }}</span>
          </div>
          <div class="info-row">
            <span class="label">基础仓位:</span>
            <span class="value">{{ (recommendationData?.trading_strategy?.position_sizing?.base_position * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="info-row">
            <span class="label">调整仓位:</span>
            <span class="value">{{ (recommendationData?.trading_strategy?.position_sizing?.adjusted_position * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
        </div>
      </div>

      <!-- 入场策略 -->
      <div class="entry-strategy">
        <h4>📈 入场策略</h4>
        <div class="strategy-details">
          <div class="entry-zone">
            <span class="zone-label">入场区间:</span>
            <div class="zone-range">
              <span class="zone-min">${{ recommendationData?.trading_strategy?.entry_zone?.min?.toFixed(2) || 'N/A' }}</span>
              <span class="zone-separator">-</span>
              <span class="zone-max">${{ recommendationData?.trading_strategy?.entry_zone?.max?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="optimal-entry">
              <span class="optimal-label">推荐价格:</span>
              <span class="optimal-value">${{ recommendationData?.trading_strategy?.entry_strategy?.recommended_entry_price?.toFixed(2) || 'N/A' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 出场策略 -->
      <div class="exit-strategy">
        <h4>📉 出场策略</h4>
        <div class="exit-targets">
          <div v-for="(target, index) in recommendationData?.trading_strategy?.exit_targets" :key="index" class="exit-target">
            <div class="target-info">
              <span class="target-number">目标 {{ index + 1 }}</span>
              <span class="target-price">${{ target.min?.toFixed(2) || 'N/A' }} - ${{ target.max?.toFixed(2) || 'N/A' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险管理 -->
      <div class="risk-management">
        <h4>🛡️ 风险管理</h4>
        <div class="risk-grid">
          <div v-for="(level, index) in recommendationData?.trading_strategy?.stop_loss_levels" :key="index" class="risk-item">
            <span class="risk-label">止损 {{ index + 1 }}:</span>
            <span class="risk-value">${{ level.price?.toFixed(2) || 'N/A' }}</span>
            <span class="risk-percentage">({{ (level.percentage * 100)?.toFixed(1) || 'N/A' }}%)</span>
          </div>
          <div class="risk-item">
            <span class="risk-label">仓位策略:</span>
            <span class="risk-value">{{ recommendationData?.trading_strategy?.position_sizing?.scaling_strategy || 'N/A' }}</span>
          </div>
          <div class="risk-item">
            <span class="risk-label">最大仓位:</span>
            <span class="risk-value">{{ (recommendationData?.trading_strategy?.position_sizing?.max_position * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="risk-item">
            <span class="risk-label">最小仓位:</span>
            <span class="risk-value">{{ (recommendationData?.trading_strategy?.position_sizing?.min_position * 100)?.toFixed(1) || 'N/A' }}%</span>
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

const getStrategyTypeText = (type) => {
  const texts = {
    'LONG': '做多策略',
    'SHORT': '做空策略',
    'RANGE': '区间策略'
  }
  return texts[type] || type
}
</script>

<style scoped lang="scss">
.strategy-card {
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

    .strategy-type-badge {
      padding: 6px 12px;
      border-radius: 20px;
      font-size: 12px;
      font-weight: 500;
      text-transform: uppercase;

      &.strategy-long {
        background: #dcfce7;
        color: #166534;
      }

      &.strategy-short {
        background: #fee2e2;
        color: #991b1b;
      }

      &.strategy-range {
        background: #fef3c7;
        color: #92400e;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .strategy-overview {
    margin-bottom: 30px;

    .strategy-info {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 16px;

      .info-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        background: #f8f9fa;
        border-radius: 8px;

        .label {
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }

        .value {
          font-size: 14px;
          color: #1a1a1a;
          font-weight: 600;
        }
      }
    }
  }

  .entry-strategy, .exit-strategy {
    margin-bottom: 30px;

    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }
  }

  .entry-strategy {
    .strategy-details {
      padding: 20px;
      background: #f8f9fa;
      border-radius: 12px;

      .entry-zone {
        .zone-label {
          font-size: 14px;
          color: #666;
          margin-bottom: 8px;
          display: block;
        }

        .zone-range {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 8px;

          .zone-min, .zone-max {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;
          }

          .zone-separator {
            color: #666;
          }
        }

        .optimal-entry {
          .optimal-label {
            font-size: 12px;
            color: #666;
          }

          .optimal-value {
            font-size: 14px;
            font-weight: 600;
            color: #10b981;
          }
        }
      }
    }
  }

  .exit-strategy {
    .exit-targets {
      display: grid;
      gap: 12px;

      .exit-target {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;

        .target-info {
          display: flex;
          align-items: center;
          gap: 16px;

          .target-number {
            font-size: 16px;
            font-weight: 700;
            color: #1a1a1a;
            min-width: 60px;
          }

          .target-price {
            font-size: 14px;
            color: #10b981;
            font-weight: 600;
          }
        }
      }
    }
  }

  .risk-management {
    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }

    .risk-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 16px;

      .risk-item {
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;
        display: flex;
        align-items: center;
        gap: 12px;

        .risk-label {
          font-size: 14px;
          color: #666;
          font-weight: 500;
          min-width: 100px;
        }

        .risk-value {
          font-size: 16px;
          font-weight: 600;
          color: #1a1a1a;
        }

        .risk-percentage {
          font-size: 12px;
          color: #666;
        }
      }
    }
  }
}
</style>
