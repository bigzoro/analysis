<template>
  <!-- 现代化统计概览 -->
  <section class="stats-overview">
    <div class="stats-header">
      <h2 class="stats-title">
        <i class="icon-stats">📊</i>
        监控概览
      </h2>
      <p class="stats-subtitle">实时监控状态与关键指标</p>
    </div>

    <div class="stats-grid">
      <!-- 监控地址卡片 -->
      <div class="stat-card primary" :class="{ 'pulse': summary.totalWatchers > 0 }">
        <div class="card-content">
          <div class="stat-icon">
            <i class="icon-monitor">📍</i>
          </div>
          <div class="stat-details">
            <div class="stat-value animate-number" :data-target="summary.totalWatchers">
              {{ summary.totalWatchers }}
            </div>
            <div class="stat-label">监控地址</div>
            <div class="stat-meta">
              <span class="meta-indicator active"></span>
              正在监控
            </div>
          </div>
        </div>
      </div>

      <!-- 活跃地址卡片 -->
      <div class="stat-card success" :class="{ 'bounce': summary.activeWatchers > 0 }">
        <div class="card-content">
          <div class="stat-icon">
            <i class="icon-active">🎯</i>
          </div>
          <div class="stat-details">
            <div class="stat-value animate-number" :data-target="summary.activeWatchers">
              {{ summary.activeWatchers }}
            </div>
            <div class="stat-label">活跃地址</div>
            <div class="stat-meta">
              <span class="meta-indicator success"></span>
              最近交易
            </div>
          </div>
        </div>
      </div>

      <!-- 交易事件卡片 -->
      <div class="stat-card info">
        <div class="card-content">
          <div class="stat-icon">
            <i class="icon-events">💹</i>
          </div>
          <div class="stat-details">
            <div class="stat-value animate-number" :data-target="summary.totalEvents">
              {{ summary.totalEvents }}
            </div>
            <div class="stat-label">交易事件</div>
            <div class="stat-meta">
              <span class="meta-indicator info"></span>
              累计命中
            </div>
          </div>
        </div>
      </div>

      <!-- 最大交易卡片 -->
      <div class="stat-card warning">
        <div class="card-content">
          <div class="stat-icon">
            <i class="icon-amount">💰</i>
          </div>
          <div class="stat-details">
            <div class="stat-value large-amount">
              {{ summary.largestLabel || '暂无' }}
            </div>
            <div class="stat-label">最大单笔</div>
            <div class="stat-meta">
              <span class="meta-indicator warning"></span>
              按金额排序
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 数据可视化区域 -->
    <div class="visualization-section">
      <div class="charts-grid">
        <!-- 链分布图 -->
        <div class="chart-card">
          <div class="chart-header">
            <h3 class="chart-title">📊 链分布</h3>
            <p class="chart-subtitle">监控地址按区块链分布</p>
          </div>
          <div class="chart-content">
            <div class="chain-distribution">
              <div
                v-for="chain in chainDistribution"
                :key="chain.name"
                class="chain-item"
              >
                <div class="chain-info">
                  <span class="chain-icon">{{ chain.icon }}</span>
                  <span class="chain-name">{{ chain.name }}</span>
                  <span class="chain-count">{{ chain.count }}</span>
                </div>
                <div class="chain-bar">
                  <div
                    class="chain-bar-fill"
                    :style="{ width: chain.percentage + '%' }"
                  ></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 活跃度趋势图 -->
        <div class="chart-card">
          <div class="chart-header">
            <h3 class="chart-title">📈 活跃度趋势</h3>
            <p class="chart-subtitle">过去24小时的交易活动</p>
          </div>
          <div class="chart-content">
            <div class="activity-trend">
              <div class="trend-bars">
                <div
                  v-for="(activity, index) in activityTrend"
                  :key="index"
                  class="trend-bar"
                  :style="{ height: activity.height + '%' }"
                  :title="activity.label"
                >
                  <div class="trend-value">{{ activity.value }}</div>
                </div>
              </div>
              <div class="trend-axis">
                <span>0</span>
                <span>{{ Math.max(...activityTrend.map(a => a.value)) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 实时状态栏 -->
    <div class="status-dashboard">
      <div class="status-metrics">
        <div class="metric-item">
          <div class="metric-icon">
            <div class="status-pulse" :class="{ active: !loading }"></div>
          </div>
          <div class="metric-content">
            <div class="metric-label">同步状态</div>
            <div class="metric-value">{{ loading ? '更新中...' : '已同步' }}</div>
          </div>
        </div>

        <div class="metric-item">
          <div class="metric-icon">
            <i class="icon-time">🕐</i>
          </div>
          <div class="metric-content">
            <div class="metric-label">最后更新</div>
            <div class="metric-value">{{ summary.lastRefreshLabel || '从未更新' }}</div>
          </div>
        </div>

        <div class="metric-item">
          <div class="metric-icon">
            <i class="icon-source">🔗</i>
          </div>
          <div class="metric-content">
            <div class="metric-label">数据源</div>
            <div class="metric-value">{{ getDataSourceLabel(dataSource) }}</div>
          </div>
        </div>
      </div>

      <!-- 进度指示器 -->
      <div v-if="loading" class="progress-indicator">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
        </div>
        <div class="progress-text">{{ progressText }}</div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { defineProps, computed } from 'vue'

const props = defineProps({
  summary: {
    type: Object,
    required: true
  },
  dataSource: {
    type: String,
    default: 'basic'
  },
  loading: {
    type: Boolean,
    default: false
  },
  progressPercent: {
    type: Number,
    default: 0
  },
  progressText: {
    type: String,
    default: ''
  },
  watchlist: {
    type: Array,
    default: () => []
  }
})

const getDataSourceLabel = (dataSource) => {
  const labels = {
    basic: '📊 基本监控',
    arkham: '🔍 Arkham',
    nansen: '📈 Nansen'
  }
  return labels[dataSource] || dataSource
}

// 链分布数据
const chainDistribution = computed(() => {
  const chains = {}
  const watchlist = props.watchlist || []

  watchlist.forEach(watch => {
    const chain = watch.chain || 'unknown'
    chains[chain] = (chains[chain] || 0) + 1
  })

  const total = watchlist.length
  const chainIcons = {
    ethereum: '🔷',
    bsc: '🟡',
    solana: '🟣',
    bitcoin: '🟠',
    polygon: '🟣',
    arbitrum: '🔵',
    optimism: '🔴',
    unknown: '⛓️'
  }

  return Object.entries(chains)
    .map(([name, count]) => ({
      name: name.charAt(0).toUpperCase() + name.slice(1),
      icon: chainIcons[name] || '⛓️',
      count,
      percentage: total > 0 ? (count / total) * 100 : 0
    }))
    .sort((a, b) => b.count - a.count)
})

// 活跃度趋势数据（模拟数据）
const activityTrend = computed(() => {
  const hours = 24
  const data = []

  for (let i = hours - 1; i >= 0; i--) {
    const hour = new Date(Date.now() - i * 60 * 60 * 1000).getHours()
    const value = Math.floor(Math.random() * 20) + 1 // 随机活跃度
    const maxValue = 25

    data.push({
      label: `${hour}:00`,
      value,
      height: (value / maxValue) * 100
    })
  }

  return data
})
</script>

<style scoped>
/* 现代化统计概览样式 */
.stats-overview {
  margin-bottom: 2.5rem;
}

.stats-header {
  margin-bottom: 1.5rem;
}

.stats-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1f2937;
  margin: 0 0 0.25rem 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.stats-subtitle {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

/* 统计卡片网格 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

/* 现代化统计卡片 */
.stat-card {
  position: relative;
  background: white;
  border-radius: 16px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1), 0 1px 2px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15), 0 4px 10px rgba(0, 0, 0, 0.1);
}

.stat-card.primary {
  background: #f0f9ff;
  border-color: #0ea5e9;
  color: #0c4a6e;
}

.stat-card.success {
  background: #f0fdf4;
  border-color: #22c55e;
  color: #166534;
}

.stat-card.info {
  background: #eff6ff;
  border-color: #3b82f6;
  color: #1e40af;
}

.stat-card.warning {
  background: #fffbeb;
  border-color: #f59e0b;
  color: #92400e;
}

/* 卡片内容 */
.card-content {
  position: relative;
  z-index: 2;
  padding: 1.5rem;
  display: flex;
  align-items: center;
  gap: 1rem;
}

.stat-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  backdrop-filter: blur(10px);
}

.stat-details {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 2rem;
  font-weight: 800;
  line-height: 1;
  margin-bottom: 0.25rem;
  color: inherit;
}

.large-amount {
  font-size: 1.125rem !important;
  font-weight: 700 !important;
  word-break: break-word;
  line-height: 1.3;
}

.stat-label {
  font-size: 0.875rem;
  font-weight: 600;
  opacity: 0.9;
  margin-bottom: 0.25rem;
}

.stat-meta {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  opacity: 0.8;
}

.meta-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.meta-indicator.active {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
}

.meta-indicator.success {
  background: #10b981;
}

.meta-indicator.info {
  background: #3b82f6;
}

.meta-indicator.warning {
  background: #f59e0b;
}

/* 状态仪表板 */
.status-dashboard {
  background: white;
  border-radius: 16px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.status-metrics {
  display: flex;
  gap: 2rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.metric-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.metric-icon {
  width: 32px;
  height: 32px;
  background: #f3f4f6;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
}

.metric-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.metric-label {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
}

.metric-value {
  font-size: 0.875rem;
  color: #1f2937;
  font-weight: 600;
}

/* 状态脉冲动画 */
.status-pulse {
  width: 12px;
  height: 12px;
  background: #ef4444;
  border-radius: 50%;
}

.status-pulse.active {
  background: #10b981;
}

/* 进度指示器 */
.progress-indicator {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.progress-bar {
  flex: 1;
  height: 4px;
  background: #e5e7eb;
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.875rem;
  color: #6b7280;
  font-weight: 500;
}

/* 数据可视化区域 */
.visualization-section {
  margin-bottom: 2rem;
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
}

.chart-card {
  background: white;
  border-radius: 16px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.chart-header {
  padding: 1.5rem 1.5rem 1rem 1.5rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}

.chart-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: #1f2937;
  margin: 0 0 0.25rem 0;
}

.chart-subtitle {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

.chart-content {
  padding: 1.5rem;
}

/* 链分布图 */
.chain-distribution {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.chain-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.chain-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
}

.chain-icon {
  font-size: 1rem;
}

.chain-name {
  flex: 1;
  color: #374151;
}

.chain-count {
  color: #6b7280;
  font-weight: 600;
}

.chain-bar {
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
}

.chain-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
}

/* 活跃度趋势图 */
.activity-trend {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.trend-bars {
  display: flex;
  align-items: end;
  gap: 2px;
  height: 120px;
}

.trend-bar {
  flex: 1;
  background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
  border-radius: 2px 2px 0 0;
  position: relative;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.trend-bar:hover {
  opacity: 0.8;
}

.trend-value {
  position: absolute;
  top: -20px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 0.75rem;
  color: #374151;
  font-weight: 600;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.trend-bar:hover .trend-value {
  opacity: 1;
}

.trend-axis {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: #6b7280;
  padding: 0 0.25rem;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .charts-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .card-content {
    padding: 1.25rem;
    gap: 0.75rem;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
    font-size: 1rem;
  }

  .stat-value {
    font-size: 1.5rem;
  }

  .status-metrics {
    gap: 1rem;
  }

  .metric-item {
    flex: 1;
    min-width: 120px;
  }

  .chart-content {
    padding: 1rem;
  }

  .trend-bars {
    height: 100px;
  }
}
</style>
