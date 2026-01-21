<template>
  <div class="ai-dashboard">
    <div class="dashboard-header">
      <h1>🚀 AI投资仪表盘</h1>
      <p class="subtitle">实时数据驱动的智能投资决策平台</p>

      <!-- 全局控制面板 -->
      <div class="global-controls">
        <div class="control-group">
          <label>数据刷新频率:</label>
          <select v-model="refreshInterval" @change="updateRefreshInterval">
            <option value="30">30秒</option>
            <option value="60">1分钟</option>
            <option value="300">5分钟</option>
            <option value="900">15分钟</option>
          </select>
        </div>

        <div class="control-group">
          <label>市场偏好:</label>
          <select v-model="marketPreference">
            <option value="all">全部市场</option>
            <option value="mainstream">主流币种</option>
            <option value="altcoins">山寨币</option>
            <option value="defi">DeFi</option>
          </select>
        </div>

        <button @click="refreshAllData" :disabled="isRefreshing" class="refresh-all-btn">
          {{ isRefreshing ? '🔄 刷新中...' : '🔄 刷新全部数据' }}
        </button>
      </div>
    </div>

    <!-- 实时状态栏 -->
    <div class="status-bar">
      <div class="status-item">
        <div class="status-icon">📊</div>
        <div class="status-info">
          <div class="status-label">市场状态</div>
          <div class="status-value" :class="marketState">{{ getMarketText(marketState) }}</div>
        </div>
      </div>

      <div class="status-item">
        <div class="status-icon">⚡</div>
        <div class="status-info">
          <div class="status-label">实时连接</div>
          <div class="status-value" :class="connectionStatus.class">{{ connectionStatus.text }}</div>
        </div>
      </div>

      <div class="status-item">
        <div class="status-icon">🤖</div>
        <div class="status-info">
          <div class="status-label">AI状态</div>
          <div class="status-value ai-active">活跃</div>
        </div>
      </div>

      <div class="status-item">
        <div class="status-icon">📈</div>
        <div class="status-info">
          <div class="status-label">最后更新</div>
          <div class="status-value">{{ formatTime(lastUpdate) }}</div>
        </div>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="dashboard-content">
      <!-- 左侧边栏 - 快速操作 -->
      <div class="sidebar">
        <div class="sidebar-section">
          <h3>⚡ 快速操作</h3>
          <div class="quick-actions">
            <button @click="showRecommendations" class="action-btn primary">
              <div class="action-icon">🎯</div>
              <div class="action-info">
                <div class="action-title">AI推荐</div>
                <div class="action-desc">智能币种推荐</div>
              </div>
            </button>

            <button @click="showRiskMonitoring" class="action-btn warning">
              <div class="action-icon">⚠️</div>
              <div class="action-info">
                <div class="action-title">风险监控</div>
                <div class="action-desc">实时风险评估</div>
              </div>
            </button>

            <button @click="showPortfolioAnalysis" class="action-btn success">
              <div class="action-icon">📊</div>
              <div class="action-info">
                <div class="action-title">组合分析</div>
                <div class="action-desc">投资组合优化</div>
              </div>
            </button>

            <button @click="showMarketOverview" class="action-btn info">
              <div class="action-icon">🌍</div>
              <div class="action-info">
                <div class="action-title">市场概览</div>
                <div class="action-desc">全市场分析</div>
              </div>
            </button>
          </div>
        </div>

        <!-- AI洞察 -->
        <div class="sidebar-section">
          <h3>🤖 AI洞察</h3>
          <div class="ai-insights">
            <div class="insight-item">
              <div class="insight-icon">📈</div>
              <div class="insight-content">
                <div class="insight-title">市场趋势</div>
                <div class="insight-value">{{ marketTrend }}</div>
              </div>
            </div>

            <div class="insight-item">
              <div class="insight-icon">🎯</div>
              <div class="insight-content">
                <div class="insight-title">最佳时机</div>
                <div class="insight-value">{{ bestTiming }}</div>
              </div>
            </div>

            <div class="insight-item">
              <div class="insight-icon">💎</div>
              <div class="insight-content">
                <div class="insight-title">潜力币种</div>
                <div class="insight-value">{{ topPick }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 主内容区 -->
      <div class="main-content">
        <!-- AI推荐概览 -->
        <div class="content-section">
          <div class="section-header">
            <h2>🎯 AI智能推荐</h2>
            <div class="section-actions">
              <button @click="$router.push('/ai-recommendations')" class="view-all-btn">
                查看全部 →
              </button>
            </div>
          </div>

          <div v-if="recommendations.length > 0" class="recommendations-preview">
            <div class="recommendation-cards">
              <div
                v-for="(rec, index) in recommendations.slice(0, 4)"
                :key="rec.symbol"
                class="recommendation-card-mini"
                :class="getCardClass(rec)"
              >
                <div class="card-rank">#{{ rec.rank }}</div>
                <div class="card-symbol">{{ rec.symbol }}</div>
                <div class="card-score">{{ (rec.overall_score * 100).toFixed(0) }}</div>
                <div class="card-change" :class="{ positive: rec.ml_prediction > 0.5 }">
                  {{ rec.ml_prediction > 0.5 ? '↗' : '↘' }}
                </div>
              </div>
            </div>
          </div>

          <div v-else class="empty-recommendations">
            <div class="empty-icon">🤖</div>
            <p>暂无推荐数据，点击上方刷新</p>
          </div>
        </div>

        <!-- 市场概览图表 -->
        <div class="content-section">
          <div class="section-header">
            <h2>📊 市场趋势分析</h2>
            <div class="section-actions">
              <select v-model="chartTimeframe" @change="updateChart">
                <option value="1h">1小时</option>
                <option value="4h">4小时</option>
                <option value="24h">24小时</option>
                <option value="7d">7天</option>
              </select>
            </div>
          </div>

          <div class="market-chart">
            <LineChart
              :x-data="marketChartData.xData"
              :series="marketChartData.series"
              :title="`市场价格趋势 (${chartTimeframe})`"
              :y-label="'价格 (USD)'"
            />
          </div>
        </div>

        <!-- 风险监控概览 -->
        <div class="content-section">
          <div class="section-header">
            <h2>⚠️ 风险监控中心</h2>
            <div class="section-actions">
              <button @click="$router.push('/risk-monitoring')" class="view-all-btn">
                查看详情 →
              </button>
            </div>
          </div>

          <div class="risk-overview">
            <div class="risk-metrics">
              <div class="metric-card">
                <div class="metric-icon">📊</div>
                <div class="metric-info">
                  <div class="metric-value">{{ riskMetrics.totalAlerts }}</div>
                  <div class="metric-label">活跃告警</div>
                </div>
              </div>

              <div class="metric-card warning">
                <div class="metric-icon">⚠️</div>
                <div class="metric-info">
                  <div class="metric-value">{{ riskMetrics.highRisk }}</div>
                  <div class="metric-label">高风险</div>
                </div>
              </div>

              <div class="metric-card success">
                <div class="metric-icon">🛡️</div>
                <div class="metric-info">
                  <div class="metric-value">{{ riskMetrics.protected }}</div>
                  <div class="metric-label">已保护</div>
                </div>
              </div>
            </div>

            <div class="risk-alerts-preview">
              <div class="alert-item" v-for="alert in riskAlerts.slice(0, 3)" :key="alert.id">
                <div class="alert-icon" :class="alert.severity">{{ getAlertIcon(alert.severity) }}</div>
                <div class="alert-content">
                  <div class="alert-message">{{ alert.message }}</div>
                  <div class="alert-time">{{ formatTime(new Date(alert.timestamp)) }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 实时通知区域 -->
    <div class="notifications-panel" v-if="notifications.length > 0">
      <div class="notifications-header">
        <h3>🔔 实时通知</h3>
        <button @click="clearNotifications" class="clear-btn">清除全部</button>
      </div>

      <div class="notifications-list">
        <div
          v-for="notification in notifications.slice(0, 5)"
          :key="notification.id"
          class="notification-item"
          :class="notification.type"
        >
          <div class="notification-icon">{{ getNotificationIcon(notification.type) }}</div>
          <div class="notification-content">
            <div class="notification-message">{{ notification.message }}</div>
            <div class="notification-time">{{ formatTime(new Date(notification.timestamp)) }}</div>
          </div>
          <button @click="dismissNotification(notification.id)" class="dismiss-btn">✕</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { api } from '@/api/api.js'
import LineChart from '@/components/LineChart.vue'
import RecommendationStats from '@/components/RecommendationStats.vue'

export default {
  name: 'AIDashboard',
  components: {
    LineChart,
    RecommendationStats
  },
  data() {
    return {
      refreshInterval: 60,
      marketPreference: 'all',
      isRefreshing: false,
      marketState: 'sideways',
      connectionStatus: { class: 'connected', text: '已连接' },
      lastUpdate: new Date(),
      chartTimeframe: '24h',

      // 数据
      recommendations: [],
      marketChartData: { xData: [], series: [] },
      riskMetrics: {
        totalAlerts: 0,
        highRisk: 0,
        protected: 0
      },
      riskAlerts: [],
      notifications: [],

      // AI洞察
      marketTrend: '震荡上行',
      bestTiming: '适中',
      topPick: 'BTC/ETH',

      // 定时器
      refreshTimer: null,
      wsConnection: null
    }
  },

  mounted() {
    this.initializeDashboard()
    this.startAutoRefresh()
    this.connectRealtimeUpdates()
  },

  beforeUnmount() {
    this.stopAutoRefresh()
    if (this.wsConnection) {
      this.wsConnection.close()
    }
  },

  methods: {
    async initializeDashboard() {
      await this.refreshAllData()
    },

    async refreshAllData() {
      this.isRefreshing = true
      try {
        await Promise.all([
          this.loadRecommendations(),
          this.loadMarketData(),
          this.loadRiskData(),
          this.generateAIInsights()
        ])
        this.lastUpdate = new Date()
      } catch (error) {
        console.error('刷新数据失败:', error)
        this.addNotification('error', '数据刷新失败，请稍后重试')
      } finally {
        this.isRefreshing = false
      }
    },

    async loadRecommendations() {
      try {
        const data = await api.getAIRecommendations({
          symbols: ['BTC', 'ETH', 'ADA', 'SOL', 'DOT'],
          limit: 8,
          risk_level: 'moderate'
        })
        this.recommendations = data.recommendations || []
      } catch (error) {
        console.error('加载推荐失败:', error)
      }
    },

    async loadMarketData() {
      // 生成模拟的市场数据
      this.generateMarketChartData()
    },

    async loadRiskData() {
      // 模拟风险数据
      this.riskMetrics = {
        totalAlerts: Math.floor(Math.random() * 10) + 1,
        highRisk: Math.floor(Math.random() * 5),
        protected: Math.floor(Math.random() * 20) + 5
      }

      this.riskAlerts = [
        {
          id: 1,
          severity: 'high',
          message: 'BTC波动率超过阈值',
          timestamp: new Date(Date.now() - 1000 * 60 * 5)
        },
        {
          id: 2,
          severity: 'medium',
          message: 'ETH流动性风险增加',
          timestamp: new Date(Date.now() - 1000 * 60 * 15)
        }
      ]
    },

    generateMarketChartData() {
      const now = new Date()
      const points = this.chartTimeframe === '1h' ? 60 : this.chartTimeframe === '4h' ? 48 : this.chartTimeframe === '24h' ? 24 : 168

      const xData = []
      for (let i = points; i >= 0; i--) {
        const time = new Date(now.getTime() - i * this.getTimeInterval())
        xData.push(time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }))
      }

      const symbols = ['BTC', 'ETH', 'ADA', 'SOL']
      const series = symbols.map(symbol => {
        const basePrice = this.getBasePrice(symbol)
        const prices = []

        let currentPrice = basePrice * (0.98 + Math.random() * 0.04)
        prices.push(currentPrice)

        for (let i = 1; i < points; i++) {
          const change = (Math.random() - 0.5) * 0.02
          currentPrice *= (1 + change)
          prices.push(currentPrice)
        }

        return {
          name: symbol,
          data: prices,
          lineStyle: { width: 2 },
          itemStyle: { color: this.getSymbolColor(symbol) }
        }
      })

      this.marketChartData = { xData, series }
    },

    getTimeInterval() {
      switch (this.chartTimeframe) {
        case '1h': return 60 * 1000
        case '4h': return 5 * 60 * 1000
        case '24h': return 60 * 60 * 1000
        case '7d': return 4 * 60 * 60 * 1000
        default: return 60 * 60 * 1000
      }
    },

    getBasePrice(symbol) {
      const prices = { BTC: 45000, ETH: 2800, ADA: 0.45, SOL: 95 }
      return prices[symbol] || 1
    },

    getSymbolColor(symbol) {
      const colors = { BTC: '#f7931a', ETH: '#627eea', ADA: '#0033ad', SOL: '#9945ff' }
      return colors[symbol] || '#666'
    },

    generateAIInsights() {
      const trends = ['震荡上行', '稳步上涨', '高位震荡', '调整中']
      const timings = ['良好', '适中', '谨慎', '观望']
      const picks = ['BTC/ETH', 'SOL/ADA', 'LINK/UNI', 'DOT/AVAX']

      this.marketTrend = trends[Math.floor(Math.random() * trends.length)]
      this.bestTiming = timings[Math.floor(Math.random() * timings.length)]
      this.topPick = picks[Math.floor(Math.random() * picks.length)]
    },

    startAutoRefresh() {
      this.refreshTimer = setInterval(() => {
        this.refreshAllData()
      }, this.refreshInterval * 1000)
    },

    stopAutoRefresh() {
      if (this.refreshTimer) {
        clearInterval(this.refreshTimer)
        this.refreshTimer = null
      }
    },

    updateRefreshInterval() {
      this.stopAutoRefresh()
      this.startAutoRefresh()
    },

    updateChart() {
      this.generateMarketChartData()
    },

    connectRealtimeUpdates() {
      try {
        const wsUrl = api.getRealtimeRecommendWS()
        this.wsConnection = new WebSocket(wsUrl)

        this.wsConnection.onopen = () => {
          this.connectionStatus = { class: 'connected', text: '已连接' }
          this.wsConnection.send(JSON.stringify({
            action: 'subscribe',
            symbols: ['BTC', 'ETH', 'ADA', 'SOL'],
            update_frequency: '60s'
          }))
        }

        this.wsConnection.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)
            if (data.type === 'recommendation_update') {
              this.handleRealtimeUpdate(data.data)
            }
          } catch (error) {
            console.error('解析实时数据失败:', error)
          }
        }

        this.wsConnection.onclose = () => {
          this.connectionStatus = { class: 'disconnected', text: '已断开' }
          // 自动重连
          setTimeout(() => this.connectRealtimeUpdates(), 5000)
        }

        this.wsConnection.onerror = () => {
          this.connectionStatus = { class: 'error', text: '连接错误' }
        }
      } catch (error) {
        console.error('创建WebSocket连接失败:', error)
      }
    },

    handleRealtimeUpdate(data) {
      // 处理实时数据更新
      data.forEach(update => {
        this.addNotification('info', `${update.symbol}: ${update.price_change_24h > 0 ? '+' : ''}${(update.price_change_24h * 100).toFixed(2)}%`)
      })
    },

    addNotification(type, message) {
      const notification = {
        id: Date.now(),
        type,
        message,
        timestamp: new Date()
      }
      this.notifications.unshift(notification)

      // 最多保留50条通知
      if (this.notifications.length > 50) {
        this.notifications = this.notifications.slice(0, 50)
      }

      // 自动清除通知
      setTimeout(() => {
        this.dismissNotification(notification.id)
      }, 10000)
    },

    dismissNotification(id) {
      this.notifications = this.notifications.filter(n => n.id !== id)
    },

    clearNotifications() {
      this.notifications = []
    },

    // 导航方法
    showRecommendations() {
      this.$router.push('/ai-recommendations')
    },

    showRiskMonitoring() {
      this.$router.push('/risk-monitoring')
    },

    showPortfolioAnalysis() {
      this.$router.push('/dashboard')
    },

    showMarketOverview() {
      this.$router.push('/market')
    },

    // 辅助方法
    getCardClass(rec) {
      const score = rec.overall_score
      if (score >= 0.8) return 'excellent'
      if (score >= 0.7) return 'good'
      if (score >= 0.6) return 'fair'
      return 'poor'
    },

    getMarketText(state) {
      const texts = { bull: '牛市', bear: '熊市', sideways: '震荡市' }
      return texts[state] || '未知'
    },

    getAlertIcon(severity) {
      const icons = { low: '🟢', medium: '🟡', high: '🔴', critical: '⚠️' }
      return icons[severity] || '❓'
    },

    getNotificationIcon(type) {
      const icons = { info: 'ℹ️', warning: '⚠️', error: '❌', success: '✅' }
      return icons[type] || '📢'
    },

    formatTime(date) {
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    }
  }
}
</script>

<style scoped>
.ai-dashboard {
  padding: 20px;
  max-width: 1600px;
  margin: 0 auto;
  background: #f8f9fa;
  min-height: 100vh;
}

.dashboard-header {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.dashboard-header h1 {
  margin: 0 0 8px 0;
  font-size: 2rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  color: #666;
  font-size: 1.1rem;
  margin-bottom: 20px;
}

.global-controls {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
}

.control-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.control-group label {
  font-weight: 600;
  color: #333;
  white-space: nowrap;
}

.control-group select {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.refresh-all-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.refresh-all-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.refresh-all-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 状态栏 */
.status-bar {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.status-item {
  background: white;
  padding: 16px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-icon {
  font-size: 1.5rem;
}

.status-info {
  flex: 1;
}

.status-label {
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 4px;
}

.status-value {
  font-size: 1rem;
  font-weight: 600;
}

.status-value.connected {
  color: #10b981;
}

.status-value.disconnected {
  color: #ef4444;
}

.status-value.error {
  color: #ef4444;
}

.status-value.ai-active {
  color: #3b82f6;
}

/* 主内容区域 */
.dashboard-content {
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: 24px;
  margin-bottom: 24px;
}

/* 侧边栏 */
.sidebar {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  height: fit-content;
}

.sidebar-section {
  margin-bottom: 24px;
}

.sidebar-section h3 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 1.1rem;
  border-bottom: 2px solid #667eea;
  padding-bottom: 8px;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.action-btn.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.action-btn.warning {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
}

.action-btn.success {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
}

.action-btn.info {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: white;
}

.action-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(0,0,0,0.2);
}

.action-icon {
  font-size: 1.2rem;
}

.action-info {
  flex: 1;
}

.action-title {
  font-weight: 600;
  font-size: 0.9rem;
}

.action-desc {
  font-size: 0.8rem;
  opacity: 0.9;
  margin-top: 2px;
}

.ai-insights {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.insight-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.insight-icon {
  font-size: 1.2rem;
}

.insight-content {
  flex: 1;
}

.insight-title {
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 4px;
}

.insight-value {
  font-weight: 600;
  color: #333;
}

/* 主内容区 */
.main-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.content-section {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h2 {
  margin: 0;
  color: #333;
  font-size: 1.25rem;
}

.section-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.view-all-btn {
  background: #f0f0f0;
  color: #333;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
}

.view-all-btn:hover {
  background: #e0e0e0;
}

.recommendations-preview {
  margin-top: 16px;
}

.recommendation-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
}

.recommendation-card-mini {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
  text-align: center;
  position: relative;
  transition: all 0.2s;
}

.recommendation-card-mini:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.recommendation-card-mini.excellent {
  border-left: 3px solid #10b981;
}

.recommendation-card-mini.good {
  border-left: 3px solid #3b82f6;
}

.recommendation-card-mini.fair {
  border-left: 3px solid #f59e0b;
}

.recommendation-card-mini.poor {
  border-left: 3px solid #ef4444;
}

.card-rank {
  position: absolute;
  top: -8px;
  right: -8px;
  background: #667eea;
  color: white;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: bold;
}

.card-symbol {
  font-weight: bold;
  color: #333;
  margin-bottom: 4px;
}

.card-score {
  font-size: 1.2rem;
  font-weight: bold;
  color: #667eea;
  margin-bottom: 4px;
}

.card-change {
  font-size: 1rem;
  font-weight: bold;
}

.card-change.positive {
  color: #10b981;
}

.card-change:not(.positive) {
  color: #ef4444;
}

.empty-recommendations {
  text-align: center;
  padding: 40px 20px;
  color: #666;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 16px;
}

.market-chart {
  margin-top: 16px;
  height: 300px;
}

.risk-overview {
  margin-top: 16px;
}

.risk-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.metric-card {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.metric-card.warning {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border: 1px solid #f59e0b;
}

.metric-card.success {
  background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
  border: 1px solid #10b981;
}

.metric-icon {
  font-size: 1.5rem;
}

.metric-info {
  flex: 1;
}

.metric-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #333;
}

.metric-label {
  font-size: 0.8rem;
  color: #666;
  margin-top: 4px;
}

.risk-alerts-preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.alert-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  border-left: 3px solid #666;
}

.alert-item .alert-icon.high {
  border-left-color: #ef4444;
}

.alert-item .alert-icon.medium {
  border-left-color: #f59e0b;
}

.alert-content {
  flex: 1;
}

.alert-message {
  font-size: 0.9rem;
  color: #333;
  margin-bottom: 4px;
}

.alert-time {
  font-size: 0.8rem;
  color: #666;
}

/* 通知面板 */
.notifications-panel {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.notifications-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.notifications-header h3 {
  margin: 0;
  color: #333;
  font-size: 1.1rem;
}

.clear-btn {
  background: #ef4444;
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
}

.notifications-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.notification-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  border-left: 3px solid #666;
}

.notification-item.info {
  background: #eff6ff;
  border-left-color: #3b82f6;
}

.notification-item.warning {
  background: #fffbeb;
  border-left-color: #f59e0b;
}

.notification-item.error {
  background: #fef2f2;
  border-left-color: #ef4444;
}

.notification-item.success {
  background: #f0fdf4;
  border-left-color: #10b981;
}

.notification-icon {
  font-size: 1.2rem;
}

.notification-content {
  flex: 1;
}

.notification-message {
  font-size: 0.9rem;
  color: #333;
  margin-bottom: 4px;
}

.notification-time {
  font-size: 0.8rem;
  color: #666;
}

.dismiss-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 1rem;
  padding: 4px;
}

@media (max-width: 1024px) {
  .dashboard-content {
    grid-template-columns: 1fr;
  }

  .sidebar {
    order: 2;
  }

  .main-content {
    order: 1;
  }
}

@media (max-width: 768px) {
  .global-controls {
    flex-direction: column;
    align-items: stretch;
  }

  .status-bar {
    grid-template-columns: 1fr;
  }

  .recommendation-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .risk-metrics {
    grid-template-columns: 1fr;
  }

  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
}
</style>
