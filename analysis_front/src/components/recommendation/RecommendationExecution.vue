<template>
  <div class="execution-plan-card">
    <div class="card-header">
      <h3>⚡ 执行计划</h3>
      <div class="execution-type-badge execution-long">
        分批执行
      </div>
    </div>
    <div class="card-body">
      <!-- 执行概览 -->
      <div class="execution-overview">
        <div class="overview-item">
          <span class="overview-label">总仓位:</span>
          <span class="overview-value">{{ (recommendationData?.execution_plan?.total_position * 100)?.toFixed(1) || 'N/A' }}%</span>
        </div>
        <div class="overview-item">
          <span class="overview-label">当前价格:</span>
          <span class="overview-value">${{ recommendationData?.execution_plan?.current_price?.toFixed(2) || 'N/A' }}</span>
        </div>
        <div class="overview-item">
          <span class="overview-label">执行时长:</span>
          <span class="overview-value">{{ recommendationData?.execution_plan?.timeline?.expected_duration || 'N/A' }}</span>
        </div>
      </div>

      <!-- 入场计划 -->
      <div class="entry-plan">
        <h4>📈 入场执行计划</h4>
        <div class="plan-steps">
          <div v-for="step in recommendationData?.execution_plan?.entry_plan" :key="step.stage_number" class="plan-step">
            <div class="step-header">
              <span class="step-number">阶段 {{ step.stage_number }}</span>
              <span class="step-percentage">{{ step.percentage }}%</span>
            </div>
            <div class="step-details">
              <div class="step-info">
                <span class="info-label">价格区间:</span>
                <span class="info-value">${{ step.price_range?.min?.toFixed(2) || 'N/A' }} - ${{ step.price_range?.max?.toFixed(2) || 'N/A' }}</span>
              </div>
              <div class="step-info">
                <span class="info-label">平均价格:</span>
                <span class="info-value">${{ step.price_range?.avg?.toFixed(2) || 'N/A' }}</span>
              </div>
              <div class="step-info">
                <span class="info-label">执行条件:</span>
                <span class="info-value">{{ step.condition }}</span>
              </div>
              <div class="step-info">
                <span class="info-label">最大滑点:</span>
                <span class="info-value">{{ (step.max_slippage * 100)?.toFixed(3) || 'N/A' }}%</span>
              </div>
              <div class="step-info">
                <span class="info-label">时间限制:</span>
                <span class="info-value">{{ step.time_limit }}</span>
              </div>
              <div class="step-priority" :class="'priority-' + step.priority.toLowerCase()">
                {{ getPriorityText(step.priority) }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 出场计划 -->
      <div class="exit-plan">
        <h4>📉 出场执行计划</h4>
        <div class="plan-steps">
          <div v-for="step in recommendationData?.execution_plan?.exit_plan" :key="step.stage_number" class="plan-step exit-step">
            <div class="step-header">
              <span class="step-number">阶段 {{ step.stage_number }}</span>
              <span class="step-percentage">{{ step.percentage }}%</span>
            </div>
            <div class="step-details">
              <div class="step-info">
                <span class="info-label">止盈区间:</span>
                <span class="info-value">${{ step.price_range?.min?.toFixed(2) || 'N/A' }} - ${{ step.price_range?.max?.toFixed(2) || 'N/A' }}</span>
              </div>
              <div class="step-info">
                <span class="info-label">目标收益率:</span>
                <span class="info-value">{{ (step.profit_target_percentage * 100)?.toFixed(1) || 'N/A' }}%</span>
              </div>
              <div class="step-info">
                <span class="info-label">风险收益比:</span>
                <span class="info-value">{{ step.risk_reward_ratio || 'N/A' }}:1</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险控制 -->
      <div class="risk-controls">
        <h4>🛡️ 风险控制措施</h4>
        <div class="controls-grid">
          <div class="control-item">
            <span class="control-label">单笔最大亏损:</span>
            <span class="control-value">{{ (recommendationData?.execution_plan?.risk_controls?.max_loss_per_trade * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="control-item">
            <span class="control-label">每日最大亏损:</span>
            <span class="control-value">{{ (recommendationData?.execution_plan?.risk_controls?.max_daily_loss * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="control-item">
            <span class="control-label">最长持仓时间:</span>
            <span class="control-value">{{ recommendationData?.execution_plan?.risk_controls?.max_holding_period || 'N/A' }}</span>
          </div>
          <div class="control-item">
            <span class="control-label">追踪止损:</span>
            <span class="control-value">{{ (recommendationData?.execution_plan?.risk_controls?.trailing_stop_percentage * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
        </div>
      </div>

      <!-- 时间里程碑 -->
      <div class="timeline">
        <h4>⏰ 执行时间表</h4>
        <div class="milestones">
          <div v-for="milestone in recommendationData?.execution_plan?.timeline?.key_milestones" :key="milestone.time" class="milestone">
            <div class="milestone-time">{{ milestone.time }}</div>
            <div class="milestone-content">
              <div class="milestone-event">{{ milestone.event }}</div>
              <div class="milestone-description">{{ milestone.description }}</div>
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

const getPriorityText = (priority) => {
  const texts = {
    'HIGH': '高优先级',
    'MEDIUM': '中优先级',
    'LOW': '低优先级'
  }
  return texts[priority] || priority
}
</script>

<style scoped lang="scss">
.execution-plan-card {
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

    .execution-type-badge {
      padding: 6px 12px;
      border-radius: 20px;
      font-size: 12px;
      font-weight: 500;
      text-transform: uppercase;

      &.execution-long {
        background: #10b981;
      }

      &.execution-short {
        background: #ef4444;
      }

      &.execution-range {
        background: #f59e0b;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .execution-overview {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 16px;
    margin-bottom: 30px;

    .overview-item {
      padding: 16px;
      background: #f8f9fa;
      border-radius: 8px;
      text-align: center;

      .overview-label {
        font-size: 12px;
        color: #666;
        margin-bottom: 4px;
        display: block;
      }

      .overview-value {
        font-size: 16px;
        font-weight: 600;
        color: #1a1a1a;
      }
    }
  }

  .entry-plan, .exit-plan {
    margin-bottom: 30px;

    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }

    .plan-steps {
      .plan-step {
        background: #f8f9fa;
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 16px;
        border-left: 4px solid #667eea;

        &.exit-step {
          border-left-color: #10b981;
        }

        .step-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 16px;

          .step-number {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;
          }

          .step-percentage {
            font-size: 14px;
            font-weight: 600;
            color: #667eea;
            background: rgba(102, 126, 234, 0.1);
            padding: 4px 8px;
            border-radius: 12px;
          }
        }

        .step-details {
          .step-info {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;

            .info-label {
              font-size: 13px;
              color: #666;
              font-weight: 500;
            }

            .info-value {
              font-size: 13px;
              color: #1a1a1a;
              font-weight: 600;
            }
          }

          .step-priority {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 11px;
            font-weight: 500;
            text-transform: uppercase;
            margin-top: 8px;

            &.priority-high {
              background: #fee2e2;
              color: #dc2626;
            }

            &.priority-medium {
              background: #fef3c7;
              color: #d97706;
            }

            &.priority-low {
              background: #f3f4f6;
              color: #6b7280;
            }
          }
        }
      }
    }
  }

  .risk-controls {
    margin-bottom: 30px;

    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }

    .controls-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 16px;

      .control-item {
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;
        display: flex;
        justify-content: space-between;
        align-items: center;

        .control-label {
          font-size: 14px;
          color: #666;
          font-weight: 500;
        }

        .control-value {
          font-size: 14px;
          color: #1a1a1a;
          font-weight: 600;
        }
      }
    }
  }

  .timeline {
    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }

    .milestones {
      .milestone {
        display: flex;
        gap: 16px;
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;
        margin-bottom: 12px;

        .milestone-time {
          font-size: 14px;
          font-weight: 600;
          color: #667eea;
          min-width: 80px;
        }

        .milestone-content {
          flex: 1;

          .milestone-event {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;
            margin-bottom: 4px;
          }

          .milestone-description {
            font-size: 12px;
            color: #666;
          }
        }
      }
    }
  }
}
</style>
