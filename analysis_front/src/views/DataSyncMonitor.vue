<template>
  <div class="data-sync-monitor">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1><i class="fas fa-sync-alt"></i> 数据同步监控</h1>
      <div class="header-actions">
        <button @click="refreshData" class="btn-primary" :disabled="loading">
          <i class="fas fa-refresh" :class="{ 'fa-spin': loading }"></i>
          刷新数据
        </button>
        <button @click="toggleAutoRefresh" class="btn-secondary">
          <i :class="autoRefresh ? 'fas fa-pause' : 'fas fa-play'"></i>
          {{ autoRefresh ? '暂停' : '开始' }}自动刷新
        </button>
      </div>
    </div>

    <!-- 全局状态概览 -->
    <div class="status-overview">
      <div class="status-card" :class="globalHealthClass">
        <div class="status-icon">
          <i :class="globalHealthIcon"></i>
        </div>
        <div class="status-content">
          <h3>系统整体状态</h3>
          <p>{{ globalHealthText }}</p>
          <small>最后检查: {{ lastCheckTime }}</small>
        </div>
      </div>
    </div>

    <!-- 同步器状态 -->
    <div class="section">
      <h2><i class="fas fa-server"></i> 同步器状态</h2>
      <div class="syncers-grid">
        <div v-for="syncer in syncers" :key="syncer.name" class="syncer-card" :class="syncer.statusClass">
          <div class="syncer-header">
            <h4>{{ syncer.displayName }}</h4>
            <span class="status-badge" :class="syncer.statusClass">{{ syncer.status }}</span>
          </div>
          <div class="syncer-stats">
            <div class="stat-item">
              <span class="stat-label">总同步次数</span>
              <span class="stat-value">{{ syncer.totalSyncs || 0 }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">成功率</span>
              <span class="stat-value">{{ syncer.successRate || '0%' }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">最后同步</span>
              <span class="stat-value">{{ formatTime(syncer.lastSyncTime) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- WebSocket状态 -->
    <div class="section">
      <h2><i class="fas fa-plug"></i> WebSocket连接状态</h2>
      <div class="websocket-status">
        <div class="ws-overview">
          <div class="ws-card">
            <div class="ws-header">
              <h4>连接状态</h4>
              <span :class="['status-indicator', websocketStatus.running ? 'online' : 'offline']"></span>
            </div>
            <div class="ws-stats">
              <div class="stat-row">
                <span>运行状态:</span>
                <span>{{ websocketStatus.running ? '运行中' : '已停止' }}</span>
              </div>
              <div class="stat-row">
                <span>健康状态:</span>
                <span>{{ websocketStatus.healthy ? '健康' : '异常' }}</span>
              </div>
              <div class="stat-row">
                <span>最后消息:</span>
                <span>{{ formatTime(websocketStatus.lastMessageTime) }}</span>
              </div>
            </div>
          </div>

          <div class="ws-card">
            <div class="ws-header">
              <h4>连接池状态</h4>
            </div>
            <div class="ws-stats">
              <div class="stat-row">
                <span>现货连接:</span>
                <span>{{ websocketStatus.spotHealthy }}/{{ websocketStatus.spotTotal }} 健康</span>
              </div>
              <div class="stat-row">
                <span>期货连接:</span>
                <span>{{ websocketStatus.futuresHealthy }}/{{ websocketStatus.futuresTotal }} 健康</span>
              </div>
              <div class="stat-row">
                <span>消息接收:</span>
                <span>{{ websocketStatus.messagesReceived || 0 }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="ws-data">
          <h4>WebSocket数据统计</h4>
          <div class="data-stats">
            <div class="data-stat">
              <span class="data-label">现货价格更新</span>
              <span class="data-value">{{ websocketStatus.spotPriceUpdates || 0 }}</span>
            </div>
            <div class="data-stat">
              <span class="data-label">期货价格更新</span>
              <span class="data-value">{{ websocketStatus.futuresPriceUpdates || 0 }}</span>
            </div>
            <div class="data-stat">
              <span class="data-label">K线数据更新</span>
              <span class="data-value">{{ websocketStatus.klineUpdates || 0 }}</span>
            </div>
            <div class="data-stat">
              <span class="data-label">深度数据更新</span>
              <span class="data-value">{{ websocketStatus.depthUpdates || 0 }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- API调用统计 -->
    <div class="section">
      <h2><i class="fas fa-chart-line"></i> API调用统计</h2>
      <div class="api-stats-grid">
        <div class="api-stat-card">
          <div class="api-stat-header">
            <h4>价格同步器</h4>
          </div>
          <div class="api-stat-content">
            <div class="api-metric">
              <span class="metric-label">WebSocket命中</span>
              <span class="metric-value">{{ apiStats.price?.websocketHits || 0 }}</span>
            </div>
            <div class="api-metric">
              <span class="metric-label">REST API调用</span>
              <span class="metric-value">{{ apiStats.price?.restApiCalls || 0 }}</span>
            </div>
            <div class="api-metric">
              <span class="metric-label">命中率</span>
              <span class="metric-value">{{ apiStats.price?.hitRate || '0%' }}</span>
            </div>
          </div>
        </div>

        <div class="api-stat-card">
          <div class="api-stat-header">
            <h4>K线同步器</h4>
          </div>
          <div class="api-stat-content">
            <div class="api-metric">
              <span class="metric-label">总API调用</span>
              <span class="metric-value">{{ apiStats.kline?.totalCalls || 0 }}</span>
            </div>
            <div class="api-metric">
              <span class="metric-label">成功率</span>
              <span class="metric-value">{{ apiStats.kline?.successRate || '0%' }}</span>
            </div>
            <div class="api-metric">
              <span class="metric-label">平均延迟</span>
              <span class="metric-value">{{ formatDuration(apiStats.kline?.avgLatency) }}</span>
            </div>
          </div>
        </div>

        <div class="api-stat-card">
          <div class="api-stat-header">
            <h4>深度同步器</h4>
          </div>
          <div class="api-stat-content">
            <div class="api-metric">
              <span class="metric-label">总API调用</span>
              <span class="metric-value">{{ apiStats.depth?.totalCalls || 0 }}</span>
            </div>
            <div class="api-metric">
              <span class="metric-label">成功率</span>
              <span class="metric-value">{{ apiStats.depth?.successRate || '0%' }}</span>
            </div>
            <div class="api-metric">
              <span class="metric-label">平均延迟</span>
              <span class="metric-value">{{ formatDuration(apiStats.depth?.avgLatency) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 数据一致性检查 -->
    <div class="section">
      <h2><i class="fas fa-balance-scale"></i> 数据一致性检查</h2>
      <div class="consistency-check">
        <div class="consistency-overview">
          <div class="consistency-score">
            <div class="score-circle" :class="consistencyScoreClass">
              <span class="score-value">{{ consistency.consistencyScore || 0 }}%</span>
              <span class="score-label">一致性得分</span>
            </div>
          </div>
          <div class="consistency-details">
            <div class="detail-item">
              <span class="detail-label">总检查次数</span>
              <span class="detail-value">{{ consistency.totalChecks || 0 }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">发现问题</span>
              <span class="detail-value">{{ consistency.issuesFound || 0 }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">最后检查</span>
              <span class="detail-value">{{ formatTime(consistency.lastCheck) }}</span>
            </div>
          </div>
        </div>

        <div class="recent-issues" v-if="consistency.recentIssues && consistency.recentIssues.length > 0">
          <h4>最近问题</h4>
          <div class="issues-list">
            <div v-for="issue in consistency.recentIssues.slice(0, 5)" :key="issue.timestamp"
                 class="issue-item" :class="issue.severity">
              <div class="issue-header">
                <span class="issue-type">{{ issue.dataType }}</span>
                <span class="issue-severity">{{ issue.severity }}</span>
              </div>
              <div class="issue-message">{{ issue.description }}</div>
              <div class="issue-time">{{ formatTime(issue.timestamp) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 告警信息 -->
    <div class="section" v-if="alerts && alerts.length > 0">
      <h2><i class="fas fa-exclamation-triangle"></i> 活动告警 ({{ alerts.length }})</h2>
      <div class="alerts-list">
        <div v-for="alert in alerts" :key="alert.id" class="alert-item" :class="alert.severity">
          <div class="alert-header">
            <span class="alert-title">{{ alert.title }}</span>
            <span class="alert-time">{{ formatTime(alert.timestamp) }}</span>
          </div>
          <div class="alert-message">{{ alert.message }}</div>
          <div class="alert-details">
            <span class="alert-component">{{ alert.component }}</span>
            <span class="alert-metric">{{ alert.metric }}: {{ alert.value }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 手动操作 -->
    <div class="section">
      <h2><i class="fas fa-cogs"></i> 手动操作</h2>
      <div class="manual-actions">
        <button @click="triggerSync('price')" class="btn-action" :disabled="loading">
          <i class="fas fa-dollar-sign"></i>
          触发价格同步
        </button>
        <button @click="triggerSync('kline')" class="btn-action" :disabled="loading">
          <i class="fas fa-chart-line"></i>
          触发K线同步
        </button>
        <button @click="triggerSync('depth')" class="btn-action" :disabled="loading">
          <i class="fas fa-layer-group"></i>
          触发深度同步
        </button>
        <button @click="triggerConsistencyCheck" class="btn-action" :disabled="loading">
          <i class="fas fa-balance-scale"></i>
          执行一致性检查
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getDataSyncStatus, triggerManualSync, getDataConsistencyStatus, getAlerts } from '@/api/api.js'

export default {
  name: 'DataSyncMonitor',
  setup() {
    const loading = ref(false)
    const autoRefresh = ref(false)
    const refreshInterval = ref(null)

    // 状态数据
    const globalHealth = ref('unknown')
    const lastCheckTime = ref('')
    const syncers = ref([])
    const websocketStatus = ref({
      running: false,
      healthy: false,
      spotTotal: 0,
      spotHealthy: 0,
      futuresTotal: 0,
      futuresHealthy: 0,
      messagesReceived: 0,
      lastMessageTime: null,
      spotPriceUpdates: 0,
      futuresPriceUpdates: 0,
      klineUpdates: 0,
      depthUpdates: 0
    })
    const apiStats = ref({
      price: { websocketHits: 0, restApiCalls: 0, hitRate: '0%' },
      kline: { totalCalls: 0, successRate: '0%', avgLatency: null },
      depth: { totalCalls: 0, successRate: '0%', avgLatency: null }
    })
    const consistency = ref({
      consistencyScore: 0,
      totalChecks: 0,
      issuesFound: 0,
      lastCheck: null,
      recentIssues: []
    })
    const alerts = ref([])

    // 加载数据
    const loadData = async () => {
      try {
        loading.value = true

        // 获取同步服务状态
        const statusResponse = await getDataSyncStatus()
        updateStatusData(statusResponse)

        // 获取一致性检查状态
        const consistencyResponse = await getDataConsistencyStatus()
        consistency.value = consistencyResponse

        // 获取告警信息
        const alertsResponse = await getAlerts()
        alerts.value = alertsResponse.active_alerts || []

      } catch (error) {
        console.error('Failed to load data sync status:', error)
      } finally {
        loading.value = false
      }
    }

    // 更新状态数据
    const updateStatusData = (data) => {
      // 全局健康状态
      globalHealth.value = data.global_health || 'unknown'
      lastCheckTime.value = formatTime(data.last_check)

      // 同步器状态
      syncers.value = [
        {
          name: 'price',
          displayName: '价格同步器',
          status: data.price?.websocket_available ? 'WebSocket优先' : 'REST API',
          statusClass: data.price?.websocket_available ? 'healthy' : 'warning',
          totalSyncs: data.price?.total_syncs,
          successRate: data.price?.websocket_hit_rate,
          lastSyncTime: data.price?.last_sync_time
        },
        {
          name: 'kline',
          displayName: 'K线同步器',
          status: '运行中',
          statusClass: 'healthy',
          totalSyncs: data.kline?.total_syncs,
          successRate: data.kline?.api_success_rate,
          lastSyncTime: data.kline?.last_sync_time
        },
        {
          name: 'depth',
          displayName: '深度同步器',
          status: '运行中',
          statusClass: 'healthy',
          totalSyncs: data.depth?.total_syncs,
          successRate: data.depth?.api_success_rate,
          lastSyncTime: data.depth?.last_sync_time
        },
        {
          name: 'websocket',
          displayName: 'WebSocket同步器',
          status: data.websocket?.is_running ? '运行中' : '已停止',
          statusClass: data.websocket?.is_healthy ? 'healthy' : 'error',
          totalSyncs: null,
          successRate: null,
          lastSyncTime: data.websocket?.last_message_time
        }
      ]

      // WebSocket状态
      websocketStatus.value = {
        running: data.websocket?.is_running || false,
        healthy: data.websocket?.is_healthy || false,
        spotTotal: data.websocket?.spot_connections || 0,
        spotHealthy: data.websocket?.healthy_spot || 0,
        futuresTotal: data.websocket?.futures_connections || 0,
        futuresHealthy: data.websocket?.healthy_futures || 0,
        messagesReceived: data.websocket?.messages_received || 0,
        lastMessageTime: data.websocket?.last_message_time,
        spotPriceUpdates: data.websocket?.total_spot_price_updates || 0,
        futuresPriceUpdates: data.websocket?.total_futures_price_updates || 0,
        klineUpdates: data.websocket?.total_kline_updates || 0,
        depthUpdates: data.websocket?.total_depth_updates || 0
      }

      // API统计
      apiStats.value = {
        price: {
          websocketHits: data.price?.websocket_hits || 0,
          restApiCalls: data.price?.rest_api_calls || 0,
          hitRate: data.price?.websocket_hit_rate || '0%'
        },
        kline: {
          totalCalls: data.kline?.api_calls_total || 0,
          successRate: data.kline?.api_success_rate || '0%',
          avgLatency: data.kline?.api_avg_latency
        },
        depth: {
          totalCalls: data.depth?.api_calls_total || 0,
          successRate: data.depth?.api_success_rate || '0%',
          avgLatency: data.depth?.api_avg_latency
        }
      }
    }

    // 刷新数据
    const refreshData = () => {
      loadData()
    }

    // 切换自动刷新
    const toggleAutoRefresh = () => {
      autoRefresh.value = !autoRefresh.value
      if (autoRefresh.value) {
        refreshInterval.value = setInterval(() => {
          loadData()
        }, 30000) // 30秒刷新一次
      } else {
        if (refreshInterval.value) {
          clearInterval(refreshInterval.value)
          refreshInterval.value = null
        }
      }
    }

    // 触发手动同步
    const triggerSync = async (syncerType) => {
      try {
        loading.value = true
        await triggerManualSync(syncerType)
        // 刷新数据以显示最新状态
        setTimeout(() => loadData(), 1000)
      } catch (error) {
        console.error(`Failed to trigger ${syncerType} sync:`, error)
      } finally {
        loading.value = false
      }
    }

    // 执行一致性检查
    const triggerConsistencyCheck = async () => {
      try {
        loading.value = true
        // 这里可以调用后端的一致性检查API
        setTimeout(() => loadData(), 1000)
      } catch (error) {
        console.error('Failed to trigger consistency check:', error)
      } finally {
        loading.value = false
      }
    }

    // 格式化时间
    const formatTime = (timeStr) => {
      if (!timeStr) return '未知'
      try {
        const date = new Date(timeStr)
        return date.toLocaleString()
      } catch {
        return timeStr
      }
    }

    // 格式化持续时间
    const formatDuration = (duration) => {
      if (!duration) return '0ms'
      return duration
    }

    // 计算全局健康状态样式
    const globalHealthClass = computed(() => {
      switch (globalHealth.value) {
        case 'healthy': return 'status-healthy'
        case 'warning': return 'status-warning'
        case 'critical': return 'status-critical'
        default: return 'status-unknown'
      }
    })

    // 计算全局健康状态图标
    const globalHealthIcon = computed(() => {
      switch (globalHealth.value) {
        case 'healthy': return 'fas fa-check-circle'
        case 'warning': return 'fas fa-exclamation-triangle'
        case 'critical': return 'fas fa-times-circle'
        default: return 'fas fa-question-circle'
      }
    })

    // 计算全局健康状态文本
    const globalHealthText = computed(() => {
      switch (globalHealth.value) {
        case 'healthy': return '系统运行正常'
        case 'warning': return '系统存在警告'
        case 'critical': return '系统存在严重问题'
        default: return '系统状态未知'
      }
    })

    // 计算一致性得分样式
    const consistencyScoreClass = computed(() => {
      const score = consistency.value.consistencyScore || 0
      if (score >= 90) return 'score-excellent'
      if (score >= 70) return 'score-good'
      if (score >= 50) return 'score-warning'
      return 'score-critical'
    })

    onMounted(() => {
      loadData()
    })

    onUnmounted(() => {
      if (refreshInterval.value) {
        clearInterval(refreshInterval.value)
      }
    })

    return {
      loading,
      autoRefresh,
      globalHealth,
      lastCheckTime,
      syncers,
      websocketStatus,
      apiStats,
      consistency,
      alerts,
      globalHealthClass,
      globalHealthIcon,
      globalHealthText,
      consistencyScoreClass,
      loadData,
      refreshData,
      toggleAutoRefresh,
      triggerSync,
      triggerConsistencyCheck,
      formatTime,
      formatDuration
    }
  }
}
</script>

<style scoped>
.data-sync-monitor {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 2px solid #e1e5e9;
}

.page-header h1 {
  color: #2c3e50;
  margin: 0;
  font-size: 28px;
}

.page-header h1 i {
  margin-right: 10px;
  color: #3498db;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.btn-primary, .btn-secondary, .btn-action {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2980b9;
}

.btn-secondary {
  background: #95a5a6;
  color: white;
}

.btn-secondary:hover {
  background: #7f8c8d;
}

.btn-action {
  background: #27ae60;
  color: white;
}

.btn-action:hover:not(:disabled) {
  background: #229954;
}

.btn-primary:disabled, .btn-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.section {
  margin-bottom: 40px;
}

.section h2 {
  color: #2c3e50;
  margin-bottom: 20px;
  font-size: 22px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.section h2 i {
  color: #3498db;
}

/* 状态概览 */
.status-overview {
  margin-bottom: 30px;
}

.status-card {
  display: flex;
  align-items: center;
  padding: 20px;
  border-radius: 8px;
  background: white;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.status-card.status-healthy {
  border-left: 4px solid #27ae60;
}

.status-card.status-warning {
  border-left: 4px solid #f39c12;
}

.status-card.status-critical {
  border-left: 4px solid #e74c3c;
}

.status-card.status-unknown {
  border-left: 4px solid #95a5a6;
}

.status-icon {
  font-size: 48px;
  margin-right: 20px;
}

.status-icon i {
  color: inherit;
}

.status-content h3 {
  margin: 0 0 5px 0;
  font-size: 18px;
}

.status-content p {
  margin: 0 0 5px 0;
  font-size: 16px;
  color: #7f8c8d;
}

.status-content small {
  color: #95a5a6;
  font-size: 12px;
}

/* 同步器状态 */
.syncers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.syncer-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  border-left: 4px solid #3498db;
}

.syncer-card.healthy {
  border-left-color: #27ae60;
}

.syncer-card.warning {
  border-left-color: #f39c12;
}

.syncer-card.error {
  border-left-color: #e74c3c;
}

.syncer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.syncer-header h4 {
  margin: 0;
  font-size: 16px;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: bold;
}

.status-badge.healthy {
  background: #d5f4e6;
  color: #27ae60;
}

.status-badge.warning {
  background: #fdeaa7;
  color: #f39c12;
}

.status-badge.error {
  background: #fadbd8;
  color: #e74c3c;
}

.syncer-stats {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 10px;
}

.stat-item {
  text-align: center;
}

.stat-label {
  display: block;
  font-size: 12px;
  color: #7f8c8d;
  margin-bottom: 5px;
}

.stat-value {
  display: block;
  font-size: 16px;
  font-weight: bold;
  color: #2c3e50;
}

/* WebSocket状态 */
.websocket-status {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.ws-overview {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.ws-card {
  padding: 15px;
  border: 1px solid #e1e5e9;
  border-radius: 6px;
}

.ws-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.ws-header h4 {
  margin: 0;
  font-size: 14px;
}

.status-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #95a5a6;
}

.status-indicator.online {
  background: #27ae60;
}

.status-indicator.offline {
  background: #e74c3c;
}

.ws-stats {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  font-size: 14px;
}

.stat-row span:first-child {
  color: #7f8c8d;
}

.ws-data h4 {
  margin: 0 0 15px 0;
  font-size: 16px;
}

.data-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 15px;
}

.data-stat {
  text-align: center;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 4px;
}

.data-label {
  display: block;
  font-size: 12px;
  color: #7f8c8d;
  margin-bottom: 5px;
}

.data-value {
  display: block;
  font-size: 18px;
  font-weight: bold;
  color: #2c3e50;
}

/* API统计 */
.api-stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.api-stat-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.api-stat-header h4 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #2c3e50;
}

.api-stat-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.api-metric {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f1f2f6;
}

.api-metric:last-child {
  border-bottom: none;
}

.metric-label {
  font-size: 14px;
  color: #7f8c8d;
}

.metric-value {
  font-size: 16px;
  font-weight: bold;
  color: #2c3e50;
}

/* 数据一致性 */
.consistency-check {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.consistency-overview {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.consistency-score {
  display: flex;
  justify-content: center;
  align-items: center;
}

.score-circle {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  border: 8px solid #e1e5e9;
  position: relative;
}

.score-circle.score-excellent {
  border-color: #27ae60;
  background: linear-gradient(135deg, #d5f4e6, #a8e6cf);
}

.score-circle.score-good {
  border-color: #f39c12;
  background: linear-gradient(135deg, #fdeaa7, #f9ca24);
}

.score-circle.score-warning {
  border-color: #e67e22;
  background: linear-gradient(135deg, #f8c471, #e67e22);
}

.score-circle.score-critical {
  border-color: #e74c3c;
  background: linear-gradient(135deg, #fadbd8, #e74c3c);
}

.score-value {
  font-size: 24px;
  font-weight: bold;
  color: #2c3e50;
}

.score-label {
  font-size: 12px;
  color: #7f8c8d;
  margin-top: 5px;
}

.consistency-details {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 15px;
}

.detail-item {
  text-align: center;
}

.detail-label {
  display: block;
  font-size: 12px;
  color: #7f8c8d;
  margin-bottom: 5px;
}

.detail-value {
  display: block;
  font-size: 18px;
  font-weight: bold;
  color: #2c3e50;
}

.recent-issues h4 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #2c3e50;
}

.issues-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.issue-item {
  padding: 12px;
  border-radius: 6px;
  border-left: 4px solid;
}

.issue-item.low {
  border-left-color: #95a5a6;
  background: #f8f9fa;
}

.issue-item.medium {
  border-left-color: #f39c12;
  background: #fdeaa7;
}

.issue-item.high {
  border-left-color: #e67e22;
  background: #f8c471;
}

.issue-item.critical {
  border-left-color: #e74c3c;
  background: #fadbd8;
}

.issue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
}

.issue-type {
  font-weight: bold;
  font-size: 14px;
}

.issue-severity {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 3px;
  background: rgba(255,255,255,0.5);
}

.issue-message {
  font-size: 14px;
  color: #2c3e50;
  margin-bottom: 5px;
}

.issue-time {
  font-size: 12px;
  color: #7f8c8d;
}

/* 告警信息 */
.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.alert-item {
  background: white;
  border-radius: 8px;
  padding: 15px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  border-left: 4px solid;
}

.alert-item.info {
  border-left-color: #3498db;
}

.alert-item.warning {
  border-left-color: #f39c12;
}

.alert-item.error {
  border-left-color: #e74c3c;
}

.alert-item.critical {
  border-left-color: #c0392b;
}

.alert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.alert-title {
  font-size: 16px;
  font-weight: bold;
  color: #2c3e50;
}

.alert-time {
  font-size: 12px;
  color: #95a5a6;
}

.alert-message {
  font-size: 14px;
  color: #34495e;
  margin-bottom: 8px;
}

.alert-details {
  display: flex;
  gap: 15px;
  font-size: 12px;
  color: #7f8c8d;
}

/* 手动操作 */
.manual-actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
}

@media (max-width: 768px) {
  .data-sync-monitor {
    padding: 10px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 15px;
  }

  .header-actions {
    width: 100%;
    justify-content: center;
  }

  .syncers-grid,
  .api-stats-grid {
    grid-template-columns: 1fr;
  }

  .ws-overview {
    grid-template-columns: 1fr;
  }

  .consistency-overview {
    grid-template-columns: 1fr;
  }

  .consistency-details {
    grid-template-columns: 1fr;
  }

  .manual-actions {
    grid-template-columns: 1fr;
  }
}

/* 动画效果 */
@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.fa-spin {
  animation: spin 1s linear infinite;
}
</style>
