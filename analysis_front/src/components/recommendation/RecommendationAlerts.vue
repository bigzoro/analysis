<template>
  <div class="price-alerts-card">
    <div class="card-header">
      <h3>🔔 价格提醒</h3>
      <div class="alerts-count">
        {{ recommendationData?.price_alerts?.length || 0 }} 个活跃提醒
      </div>
    </div>
    <div class="card-body">
      <div class="alerts-list">
        <div v-if="recommendationData?.price_alerts?.length === 0" class="no-alerts">
          <div class="no-alerts-icon">🔔</div>
          <p>暂无价格提醒</p>
          <small>设置价格提醒，及时把握交易机会</small>
        </div>
        <div v-else v-for="alert in recommendationData?.price_alerts" :key="alert.id" class="alert-item" :class="{ 'alert-active': alert.is_active }">
          <div class="alert-icon" :class="'alert-' + alert.alert_type">
            {{ alert.alert_type === 'entry' ? '📈' : alert.alert_type === 'stop_loss' ? '🛑' : '🎯' }}
          </div>
          <div class="alert-content">
            <div class="alert-header">
              <span class="alert-type">{{ getAlertTypeText(alert.alert_type) }}</span>
              <span class="alert-status" :class="{ 'status-active': alert.is_active }">
                {{ alert.is_active ? '活跃' : '暂停' }}
              </span>
            </div>
            <div class="alert-details">
              <span class="alert-condition">{{ getConditionText(alert.condition) }}</span>
              <span class="alert-price">${{ alert.price_level?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="alert-message">{{ alert.message }}</div>
            <div class="alert-meta">
              <span class="alert-priority" :class="'priority-' + alert.priority.toLowerCase()">
                {{ getPriorityText(alert.priority) }}
              </span>
              <span class="alert-time">{{ formatDate(alert.created_at) }}</span>
            </div>
          </div>
          <div class="alert-actions">
            <button class="alert-action-btn edit-btn" @click="$emit('editAlert', alert)" title="编辑提醒">
              ✏️
            </button>
            <button class="alert-action-btn toggle-btn"
                    :class="{ 'active': alert.is_active }"
                    @click="$emit('toggleAlert', alert)"
                    :title="alert.is_active ? '暂停提醒' : '激活提醒'">
              {{ alert.is_active ? '⏸️' : '▶️' }}
            </button>
            <button class="alert-action-btn delete-btn" @click="$emit('deleteAlert', alert)" title="删除提醒">
              🗑️
            </button>
          </div>
        </div>
      </div>

      <div class="alert-actions">
        <button class="add-alert-btn" @click="$emit('addAlert')">
          ➕ 添加新提醒
        </button>
        <button class="manage-alerts-btn" @click="$emit('manageAlerts')">
          ⚙️ 管理提醒
        </button>
      </div>

      <!-- 提醒统计 -->
      <div v-if="recommendationData?.price_alerts?.length > 0" class="alerts-stats">
        <div class="stats-grid">
          <div class="stat-item">
            <span class="stat-label">活跃提醒</span>
            <span class="stat-value">{{ activeAlertsCount }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">今日触发</span>
            <span class="stat-value">{{ todaysTriggers }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">本周触发</span>
            <span class="stat-value">{{ weeklyTriggers }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">成功率</span>
            <span class="stat-value">{{ triggerSuccessRate }}%</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits, computed } from 'vue'

const props = defineProps({
  recommendationData: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits([
  'editAlert',
  'toggleAlert',
  'deleteAlert',
  'addAlert',
  'manageAlerts'
])

// 计算属性
const activeAlertsCount = computed(() => {
  return props.recommendationData?.price_alerts?.filter(alert => alert.is_active)?.length || 0
})

const todaysTriggers = computed(() => {
  // 这里可以计算今日触发的提醒数量
  // 暂时返回模拟数据
  return Math.floor(Math.random() * 5)
})

const weeklyTriggers = computed(() => {
  // 这里可以计算本周触发的提醒数量
  // 暂时返回模拟数据
  return Math.floor(Math.random() * 15) + 5
})

const triggerSuccessRate = computed(() => {
  // 这里可以计算提醒成功率
  // 暂时返回模拟数据
  return Math.floor(Math.random() * 20) + 75
})

// 工具函数
const getAlertTypeText = (alertType) => {
  const texts = {
    'entry': '入场提醒',
    'exit': '出场提醒',
    'stop_loss': '止损提醒',
    'profit_target': '止盈提醒'
  }
  return texts[alertType] || alertType
}

const getConditionText = (condition) => {
  const texts = {
    'above': '上涨至',
    'below': '下跌至',
    'cross': '穿越',
    'breakout': '突破',
    'pullback': '回调至'
  }
  return texts[condition] || condition
}

const getPriorityText = (priority) => {
  const texts = {
    'HIGH': '高优先级',
    'MEDIUM': '中优先级',
    'LOW': '低优先级'
  }
  return texts[priority] || priority
}

const formatDate = (dateString) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped lang="scss">
.price-alerts-card {
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

    .alerts-count {
      font-size: 14px;
      opacity: 0.9;
      background: rgba(255, 255, 255, 0.2);
      padding: 6px 12px;
      border-radius: 20px;
    }
  }

  .card-body {
    padding: 24px;
  }

  .alerts-list {
    margin-bottom: 24px;

    .no-alerts {
      text-align: center;
      padding: 40px;
      color: #9ca3af;

      .no-alerts-icon {
        font-size: 48px;
        margin-bottom: 16px;
        opacity: 0.5;
      }

      p {
        margin: 8px 0;
        font-size: 16px;
        color: #6b7280;
      }

      small {
        font-size: 12px;
      }
    }

    .alert-item {
      display: flex;
      gap: 16px;
      padding: 16px;
      background: #f8f9fa;
      border-radius: 8px;
      margin-bottom: 12px;
      border-left: 4px solid #e5e7eb;
      transition: all 0.3s ease;

      &.alert-active {
        border-left-color: #10b981;
        background: #f0fdf4;
      }

      .alert-icon {
        font-size: 24px;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: #e5e7eb;
        border-radius: 8px;
        flex-shrink: 0;

        &.alert-entry {
          background: #dcfce7;
        }

        &.alert-stop_loss {
          background: #fee2e2;
        }

        &.alert-profit_target {
          background: #fef3c7;
        }
      }

      .alert-content {
        flex: 1;

        .alert-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;

          .alert-type {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;
          }

          .alert-status {
            font-size: 12px;
            padding: 2px 8px;
            border-radius: 12px;

            &.status-active {
              background: #dcfce7;
              color: #166534;
            }

            &:not(.status-active) {
              background: #f3f4f6;
              color: #6b7280;
            }
          }
        }

        .alert-details {
          display: flex;
          gap: 16px;
          margin-bottom: 8px;

          .alert-condition {
            font-size: 12px;
            color: #666;
          }

          .alert-price {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;
          }
        }

        .alert-message {
          font-size: 13px;
          color: #374151;
          margin-bottom: 8px;
        }

        .alert-meta {
          display: flex;
          justify-content: space-between;
          align-items: center;

          .alert-priority {
            font-size: 11px;
            padding: 2px 6px;
            border-radius: 8px;
            text-transform: uppercase;
            font-weight: 500;

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

          .alert-time {
            font-size: 11px;
            color: #9ca3af;
          }
        }
      }

      .alert-actions {
        display: flex;
        flex-direction: column;
        gap: 4px;

        .alert-action-btn {
          width: 28px;
          height: 28px;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-size: 12px;
          display: flex;
          align-items: center;
          justify-content: center;
          transition: all 0.2s;

          &.edit-btn {
            background: #e5e7eb;

            &:hover {
              background: #d1d5db;
            }
          }

          &.toggle-btn {
            background: #fef3c7;

            &.active {
              background: #dcfce7;
            }

            &:hover {
              opacity: 0.8;
            }
          }

          &.delete-btn {
            background: #fee2e2;

            &:hover {
              background: #fecaca;
            }
          }
        }
      }
    }
  }

  .alert-actions {
    display: flex;
    gap: 12px;
    justify-content: center;
    margin-bottom: 24px;

    .add-alert-btn, .manage-alerts-btn {
      padding: 12px 24px;
      border: none;
      border-radius: 8px;
      font-size: 14px;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.3s ease;

      &.add-alert-btn {
        background: #667eea;
        color: white;

        &:hover {
          background: #5a67d8;
        }
      }

      &.manage-alerts-btn {
        background: #f3f4f6;
        color: #374151;

        &:hover {
          background: #e5e7eb;
        }
      }
    }
  }

  .alerts-stats {
    border-top: 1px solid #e5e7eb;
    padding-top: 24px;

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
      gap: 16px;

      .stat-item {
        text-align: center;
        padding: 16px;
        background: #f8f9fa;
        border-radius: 8px;

        .stat-label {
          font-size: 12px;
          color: #666;
          margin-bottom: 4px;
          display: block;
        }

        .stat-value {
          font-size: 18px;
          font-weight: 600;
          color: #1a1a1a;
        }
      }
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .price-alerts-card {
    .alerts-list {
      .alert-item {
        flex-direction: column;
        gap: 12px;

        .alert-actions {
          flex-direction: row;
          justify-content: center;
        }
      }
    }

    .alert-actions {
      flex-direction: column;

      .add-alert-btn, .manage-alerts-btn {
        width: 100%;
      }
    }

    .alerts-stats {
      .stats-grid {
        grid-template-columns: repeat(2, 1fr);
      }
    }
  }
}
</style>
