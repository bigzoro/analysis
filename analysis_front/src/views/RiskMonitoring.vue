<template>
  <div class="risk-monitoring">
    <div class="page-header">
      <h1>🛡️ 风险监控中心</h1>
      <p class="subtitle">实时风险评估与智能告警系统</p>
    </div>

    <!-- 风险概览面板 -->
    <div class="overview-panel">
      <div class="metric-cards">
        <div class="metric-card">
          <div class="metric-icon">📊</div>
          <div class="metric-content">
            <div class="metric-value">{{ riskReport?.summary?.avg_risk_score?.toFixed(1) || '0.0' }}</div>
            <div class="metric-label">平均风险评分</div>
            <div class="metric-trend" :class="getTrendClass(riskTrends.overall)">
              {{ getTrendIcon(riskTrends.overall) }} {{ Math.abs(riskTrends.overall || 0).toFixed(1) }}%
            </div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-icon">⚠️</div>
          <div class="metric-content">
            <div class="metric-value">{{ riskReport?.summary?.high_risk_count || 0 }}</div>
            <div class="metric-label">高风险资产</div>
            <div class="metric-desc">需要重点关注</div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-icon">🚨</div>
          <div class="metric-content">
            <div class="metric-value">{{ activeAlerts.length }}</div>
            <div class="metric-label">活跃告警</div>
            <div class="metric-desc">{{ getCriticalCount() }}个紧急</div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-icon">🔄</div>
          <div class="metric-content">
            <div class="metric-value">{{ riskReport?.summary?.total_symbols || 0 }}</div>
            <div class="metric-label">监控资产</div>
            <div class="metric-desc">实时监控中</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 控制面板 -->
    <div class="control-panel">
      <div class="control-tabs">
        <button
          @click="activeTab = 'alerts'"
          :class="{ active: activeTab === 'alerts' }"
          class="tab-btn"
        >
          🚨 风险告警
        </button>
        <button
          @click="activeTab = 'assessment'"
          :class="{ active: activeTab === 'assessment' }"
          class="tab-btn"
        >
          📊 风险评估
        </button>
        <button
          @click="activeTab = 'portfolio'"
          :class="{ active: activeTab === 'portfolio' }"
          class="tab-btn"
        >
          📁 组合分析
        </button>
      </div>

      <div class="control-actions">
        <button @click="refreshData" :disabled="loading" class="refresh-btn">
          {{ loading ? '刷新中...' : '🔄 刷新数据' }}
        </button>
        <button @click="exportReport" class="export-btn">
          📄 导出报告
        </button>
      </div>
    </div>

    <!-- 风险告警标签页 -->
    <div v-if="activeTab === 'alerts'" class="tab-content">
      <div class="alerts-section">
        <div class="alerts-header">
          <h3>活跃风险告警</h3>
          <div class="alert-filters">
            <select v-model="alertFilter.severity" @change="fetchAlerts">
              <option value="">全部级别</option>
              <option value="critical">紧急</option>
              <option value="high">高</option>
              <option value="medium">中等</option>
              <option value="low">低</option>
            </select>
            <select v-model="alertFilter.status" @change="fetchAlerts">
              <option value="active">活跃</option>
              <option value="acknowledged">已确认</option>
              <option value="resolved">已解决</option>
            </select>
          </div>
        </div>

        <div v-if="filteredAlerts.length === 0" class="empty-state">
          <div class="empty-icon">✅</div>
          <p>暂无{{ alertFilter.status === 'active' ? '活跃' : '' }}风险告警</p>
        </div>

        <div v-else class="alerts-list">
          <div
            v-for="alert in filteredAlerts"
            :key="alert.id"
            class="alert-item"
            :class="alert.severity"
          >
            <div class="alert-header">
              <div class="alert-symbol">
                <span class="symbol">{{ alert.symbol }}</span>
                <span class="alert-type">{{ getAlertTypeText(alert.alert_type) }}</span>
              </div>
              <div class="alert-severity" :class="alert.severity">
                {{ getSeverityText(alert.severity) }}
              </div>
              <div class="alert-time">
                {{ formatTime(alert.timestamp) }}
              </div>
            </div>

            <div class="alert-content">
              <div class="alert-message">{{ alert.message }}</div>
              <div class="alert-details">
                <span class="detail-item">
                  当前值: <strong>{{ alert.risk_score || 'N/A' }}</strong>
                </span>
                <span class="detail-item">
                  阈值: <strong>{{ alert.threshold || 'N/A' }}</strong>
                </span>
              </div>
            </div>

            <div v-if="alert.status === 'active'" class="alert-actions">
              <button @click="acknowledgeAlert(alert)" class="acknowledge-btn">
                ✅ 确认处理
              </button>
              <button @click="viewDetails(alert)" class="details-btn">
                📋 查看详情
              </button>
            </div>

            <div v-if="alert.status === 'acknowledged'" class="alert-status">
              <span class="acknowledged-badge">已确认</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 风险评估标签页 -->
    <div v-if="activeTab === 'assessment'" class="tab-content">
      <div class="assessment-section">
        <div class="symbol-selector">
          <label>选择要评估的资产：</label>
          <div class="symbol-inputs">
            <input
              v-model="assessmentSymbol"
              placeholder="输入币种代码，如 BTC"
              @keyup.enter="assessRisk"
            />
            <button @click="assessRisk" :disabled="!assessmentSymbol || assessing" class="assess-btn">
              {{ assessing ? '评估中...' : '🔍 风险评估' }}
            </button>
          </div>
        </div>

        <div v-if="currentAssessment" class="assessment-result">
          <div class="assessment-header">
            <h3>{{ currentAssessment.symbol }} 风险评估报告</h3>
            <div class="assessment-time">
              评估时间: {{ formatTime(currentAssessment.risk_profile?.last_updated) }}
            </div>
          </div>

          <div class="risk-overview">
            <div class="risk-score-display">
              <div class="main-score">
                <div class="score-circle" :class="getRiskLevelClass(currentAssessment.risk_profile?.risk_level)">
                  {{ currentAssessment.risk_profile?.risk_score?.toFixed(1) }}
                </div>
                <div class="score-label">
                  {{ getRiskLevelText(currentAssessment.risk_profile?.risk_level) }}
                </div>
              </div>
            </div>

            <div class="risk-factors">
              <h4>风险因子分析</h4>
              <div class="factors-grid">
                <div
                  v-for="(factor, key) in currentAssessment.risk_profile?.risk_factors || {}"
                  :key="key"
                  class="factor-item"
                >
                  <div class="factor-name">{{ getFactorName(key) }}</div>
                  <div class="factor-bar">
                    <div
                      class="factor-fill"
                      :style="{ width: (factor * 100) + '%' }"
                      :class="getFactorColor(factor)"
                    ></div>
                  </div>
                  <div class="factor-value">{{ (factor * 100).toFixed(1) }}%</div>
                </div>
              </div>
            </div>
          </div>

          <div class="position-limits">
            <h4>建议仓位限制</h4>
            <div class="limits-grid">
              <div class="limit-item">
                <span class="limit-label">最大仓位</span>
                <span class="limit-value">{{ ((currentAssessment.risk_profile?.position_limits?.max_position || 0) * 100).toFixed(1) }}%</span>
              </div>
              <div class="limit-item">
                <span class="limit-label">最大回撤</span>
                <span class="limit-value">{{ ((currentAssessment.risk_profile?.position_limits?.max_drawdown || 0) * 100).toFixed(1) }}%</span>
              </div>
              <div class="limit-item">
                <span class="limit-label">分散度要求</span>
                <span class="limit-value">{{ currentAssessment.risk_profile?.position_limits?.diversification || 0 }}个</span>
              </div>
            </div>
          </div>

          <div v-if="currentAssessment.historical_risk?.length > 0" class="historical-risk">
            <h4>历史风险数据</h4>
            <div class="history-chart">
              <!-- 这里可以集成图表库显示历史风险趋势 -->
              <div class="history-points">
                <div
                  v-for="point in currentAssessment.historical_risk.slice(0, 7)"
                  :key="point.timestamp"
                  class="history-point"
                >
                  <span class="date">{{ formatDate(point.timestamp) }}</span>
                  <span class="score">{{ point.risk_score.toFixed(1) }}</span>
                  <span class="pnl" :class="point.pnl >= 0 ? 'positive' : 'negative'">
                    {{ (point.pnl * 100).toFixed(2) }}%
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="assessmentError" class="error-state">
          <div class="error-icon">⚠️</div>
          <h4>评估失败</h4>
          <p>{{ assessmentError }}</p>
          <button @click="assessRisk" class="retry-btn">重试</button>
        </div>
      </div>
    </div>

    <!-- 投资组合分析标签页 -->
    <div v-if="activeTab === 'portfolio'" class="tab-content">
      <div class="portfolio-section">
        <div class="portfolio-input">
          <h3>投资组合风险分析</h3>
          <div class="portfolio-form">
            <div class="form-group">
              <label>投资金额：</label>
              <input
                v-model.number="portfolioConfig.totalValue"
                type="number"
                placeholder="100000"
                step="1000"
              />
            </div>

            <div class="form-group">
              <label>风险偏好：</label>
              <select v-model="portfolioConfig.riskTolerance">
                <option value="conservative">保守型</option>
                <option value="moderate">稳健型</option>
                <option value="aggressive">激进型</option>
              </select>
            </div>

            <div class="positions-input">
              <label>持仓配置：</label>
              <div class="position-items">
                <div
                  v-for="(position, symbol) in portfolioConfig.positions"
                  :key="symbol"
                  class="position-item"
                >
                  <span class="symbol">{{ symbol }}</span>
                  <input
                    v-model.number="portfolioConfig.positions[symbol]"
                    type="number"
                    step="0.01"
                    min="0"
                    max="1"
                    placeholder="0.00"
                  />
                  <button @click="removePosition(symbol)" class="remove-btn">×</button>
                </div>
              </div>

              <div class="add-position">
                <input
                  v-model="newPositionSymbol"
                  placeholder="输入币种代码"
                  @keyup.enter="addPosition"
                />
                <button @click="addPosition" :disabled="!newPositionSymbol" class="add-btn">
                  ➕ 添加
                </button>
              </div>
            </div>

            <button @click="analyzePortfolio" :disabled="analyzingPortfolio" class="analyze-btn">
              {{ analyzingPortfolio ? '分析中...' : '📊 分析组合' }}
            </button>
          </div>
        </div>

        <div v-if="portfolioAnalysis" class="portfolio-result">
          <div class="analysis-header">
            <h3>组合风险分析报告</h3>
            <div class="analysis-summary">
              <div class="summary-item">
                <span class="label">整体风险评分</span>
                <span class="value">{{ portfolioAnalysis.portfolio_risk?.overall_risk_score?.toFixed(1) }}</span>
              </div>
              <div class="summary-item">
                <span class="label">预期收益率</span>
                <span class="value positive">{{ ((portfolioAnalysis.portfolio_analysis?.expected_portfolio_return || 0) * 100).toFixed(1) }}%</span>
              </div>
              <div class="summary-item">
                <span class="label">组合波动率</span>
                <span class="value">{{ ((portfolioAnalysis.portfolio_risk?.portfolio_volatility || 0) * 100).toFixed(1) }}%</span>
              </div>
              <div class="summary-item">
                <span class="label">夏普比率</span>
                <span class="value">{{ portfolioAnalysis.portfolio_analysis?.portfolio_sharpe_ratio?.toFixed(2) }}</span>
              </div>
            </div>
          </div>

          <div class="position-analysis">
            <h4>持仓分析</h4>
            <div class="position-analysis-grid">
              <div
                v-for="(analysis, symbol) in portfolioAnalysis.position_analysis || {}"
                :key="symbol"
                class="position-analysis-item"
              >
                <div class="position-header">
                  <span class="symbol">{{ symbol }}</span>
                  <span class="weight">{{ ((analysis.weight || 0) * 100).toFixed(1) }}%</span>
                </div>
                <div class="position-details">
                  <div class="detail-item">
                    <span class="label">贡献风险</span>
                    <span class="value">{{ ((analysis.contribution_to_risk || 0) * 100).toFixed(1) }}%</span>
                  </div>
                  <div class="detail-item">
                    <span class="label">建议权重</span>
                    <span class="value">{{ ((analysis.recommended_weight || 0) * 100).toFixed(1) }}%</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="recommendations">
            <h4>优化建议</h4>
            <ul class="recommendations-list">
              <li v-for="rec in portfolioAnalysis.recommendations || []" :key="rec">
                {{ rec }}
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { api } from '@/api/api.js'

export default {
  name: 'RiskMonitoring',
  data() {
    return {
      activeTab: 'alerts',
      loading: false,
      riskReport: null,
      activeAlerts: [],
      alertFilter: {
        severity: '',
        status: 'active'
      },
      assessmentSymbol: '',
      assessing: false,
      currentAssessment: null,
      assessmentError: null,
      portfolioConfig: {
        positions: {
          BTC: 0.4,
          ETH: 0.35,
          ADA: 0.25
        },
        totalValue: 100000,
        riskTolerance: 'moderate'
      },
      newPositionSymbol: '',
      analyzingPortfolio: false,
      portfolioAnalysis: null,
      riskTrends: {
        overall: 0,
        highRisk: 0,
        alerts: 0
      }
    }
  },
  computed: {
    filteredAlerts() {
      return this.activeAlerts.filter(alert => {
        const severityMatch = !this.alertFilter.severity || alert.severity === this.alertFilter.severity
        const statusMatch = !this.alertFilter.status || alert.status === this.alertFilter.status
        return severityMatch && statusMatch
      })
    }
  },
  mounted() {
    this.fetchData()
    this.connectRealtimeAlerts()
  },
  beforeUnmount() {
    if (this.ws) {
      this.ws.close()
    }
  },
  methods: {
    async fetchData() {
      this.loading = true
      try {
        const [reportRes, alertsRes] = await Promise.all([
          api.getRiskReport(),
          api.getRiskAlerts({ status: 'active' })
        ])

        this.riskReport = reportRes
        this.activeAlerts = alertsRes.alerts || []

        // 计算趋势（模拟）
        this.calculateTrends()
      } catch (error) {
        console.error('获取风险数据失败:', error)
        this.$toast?.error('获取风险数据失败')
      } finally {
        this.loading = false
      }
    },

    async fetchAlerts() {
      try {
        const params = {
          status: this.alertFilter.status
        }
        if (this.alertFilter.severity) {
          params.severity = this.alertFilter.severity
        }
        const response = await api.getRiskAlerts(params)
        this.activeAlerts = response.alerts || []
      } catch (error) {
        console.error('获取告警失败:', error)
      }
    },

    async assessRisk() {
      if (!this.assessmentSymbol.trim()) return

      this.assessing = true
      this.assessmentError = null
      this.currentAssessment = null

      try {
        const data = await api.assessRisk({
          symbol: this.assessmentSymbol.toUpperCase(),
          include_history: true,
          time_range: '30d'
        })
        this.currentAssessment = data
      } catch (error) {
        this.assessmentError = error.message || '风险评估失败'
        console.error('风险评估失败:', error)
      } finally {
        this.assessing = false
      }
    },

    async acknowledgeAlert(alert) {
      try {
        await api.acknowledgeAlert(alert.id, '通过前端确认处理')
        // 重新获取告警列表
        await this.fetchAlerts()
        this.$toast?.success('告警已确认')
      } catch (error) {
        console.error('确认告警失败:', error)
        this.$toast?.error('确认告警失败')
      }
    },

    async analyzePortfolio() {
      this.analyzingPortfolio = true
      try {
        const data = await api.analyzePortfolio(
          this.portfolioConfig.positions,
          {
            totalValue: this.portfolioConfig.totalValue,
            riskTolerance: this.portfolioConfig.riskTolerance
          }
        )
        this.portfolioAnalysis = data
      } catch (error) {
        console.error('投资组合分析失败:', error)
        this.$toast?.error('投资组合分析失败')
      } finally {
        this.analyzingPortfolio = false
      }
    },

    addPosition() {
      if (!this.newPositionSymbol.trim()) return

      const symbol = this.newPositionSymbol.toUpperCase()
      if (!this.portfolioConfig.positions[symbol]) {
        this.$set(this.portfolioConfig.positions, symbol, 0)
      }
      this.newPositionSymbol = ''
    },

    removePosition(symbol) {
      this.$delete(this.portfolioConfig.positions, symbol)
    },

    connectRealtimeAlerts() {
      // 这里可以实现WebSocket连接来接收实时告警
      // 暂时使用定时刷新
      setInterval(() => {
        if (this.activeTab === 'alerts') {
          this.fetchAlerts()
        }
      }, 30000) // 30秒刷新一次
    },

    calculateTrends() {
      // 模拟计算趋势
      this.riskTrends.overall = (Math.random() - 0.5) * 10
      this.riskTrends.highRisk = Math.floor(Math.random() * 5) - 2
      this.riskTrends.alerts = Math.floor(Math.random() * 3) - 1
    },

    getTrendClass(trend) {
      if (trend > 0) return 'up'
      if (trend < 0) return 'down'
      return 'stable'
    },

    getTrendIcon(trend) {
      if (trend > 0) return '↗️'
      if (trend < 0) return '↘️'
      return '→'
    },

    getCriticalCount() {
      return this.activeAlerts.filter(alert => alert.severity === 'critical').length
    },

    getAlertTypeText(type) {
      const types = {
        volatility: '波动率',
        drawdown: '回撤',
        liquidity: '流动性',
        correlation: '相关性'
      }
      return types[type] || type
    },

    getSeverityText(severity) {
      const severities = {
        critical: '紧急',
        high: '高',
        medium: '中等',
        low: '低'
      }
      return severities[severity] || severity
    },

    getRiskLevelClass(level) {
      return level || 'unknown'
    },

    getRiskLevelText(level) {
      const levels = {
        low: '低风险',
        medium: '中等风险',
        high: '高风险',
        critical: '极高风险'
      }
      return levels[level] || '未知风险'
    },

    getFactorName(key) {
      const names = {
        volatility: '波动率风险',
        liquidity: '流动性风险',
        market_risk: '市场风险',
        credit_risk: '信用风险',
        operational: '操作风险'
      }
      return names[key] || key
    },

    getFactorColor(factor) {
      if (factor < 0.3) return 'low'
      if (factor < 0.6) return 'medium'
      return 'high'
    },

    formatTime(timestamp) {
      if (!timestamp) return '未知时间'
      return new Date(timestamp).toLocaleString('zh-CN')
    },

    formatDate(timestamp) {
      if (!timestamp) return '未知日期'
      return new Date(timestamp).toLocaleDateString('zh-CN')
    },

    refreshData() {
      this.fetchData()
    },

    exportReport() {
      // 导出风险报告
      console.log('导出风险报告')
      this.$toast?.info('报告导出功能开发中')
    },

    viewDetails(alert) {
      // 查看告警详情
      console.log('查看告警详情:', alert)
      // 可以实现详情查看逻辑
    }
  }
}
</script>

<style scoped>
.risk-monitoring {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  text-align: center;
  margin-bottom: 30px;
}

.page-header h1 {
  font-size: 2.5rem;
  margin-bottom: 10px;
  background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  color: #666;
  font-size: 1.1rem;
}

.overview-panel {
  margin-bottom: 30px;
}

.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.metric-card {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  gap: 15px;
}

.metric-icon {
  font-size: 2rem;
}

.metric-content {
  flex: 1;
}

.metric-value {
  font-size: 2rem;
  font-weight: bold;
  color: #333;
  margin-bottom: 5px;
}

.metric-label {
  font-size: 0.9rem;
  color: #666;
  margin-bottom: 5px;
}

.metric-trend, .metric-desc {
  font-size: 0.8rem;
  color: #888;
}

.metric-trend.up {
  color: #10b981;
}

.metric-trend.down {
  color: #ef4444;
}

.control-panel {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  margin-bottom: 30px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.control-tabs {
  display: flex;
  gap: 10px;
}

.tab-btn {
  padding: 10px 20px;
  border: none;
  background: #f0f0f0;
  color: #666;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn.active, .tab-btn:hover {
  background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
  color: white;
}

.control-actions {
  display: flex;
  gap: 10px;
}

.refresh-btn, .export-btn {
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.refresh-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.refresh-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.export-btn {
  background: #f0f0f0;
  color: #333;
}

.export-btn:hover {
  background: #e0e0e0;
}

.tab-content {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  padding: 20px;
}

/* 告警样式 */
.alerts-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.alerts-header h3 {
  margin: 0;
  color: #333;
}

.alert-filters {
  display: flex;
  gap: 10px;
}

.alert-filters select {
  padding: 6px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.alert-item {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 15px;
  transition: all 0.2s;
}

.alert-item:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.alert-item.critical {
  border-left: 4px solid #dc2626;
  background: rgba(220, 38, 38, 0.05);
}

.alert-item.high {
  border-left: 4px solid #ea580c;
  background: rgba(234, 88, 12, 0.05);
}

.alert-item.medium {
  border-left: 4px solid #d97706;
  background: rgba(217, 119, 6, 0.05);
}

.alert-item.low {
  border-left: 4px solid #65a30d;
  background: rgba(101, 163, 13, 0.05);
}

.alert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.alert-symbol {
  display: flex;
  flex-direction: column;
}

.alert-symbol .symbol {
  font-size: 1.2rem;
  font-weight: bold;
  color: #333;
}

.alert-type {
  font-size: 0.8rem;
  color: #666;
}

.alert-severity {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 600;
}

.alert-severity.critical {
  background: #fee2e2;
  color: #dc2626;
}

.alert-severity.high {
  background: #fed7aa;
  color: #ea580c;
}

.alert-severity.medium {
  background: #fde68a;
  color: #d97706;
}

.alert-severity.low {
  background: #d9f99d;
  color: #65a30d;
}

.alert-time {
  font-size: 0.8rem;
  color: #888;
}

.alert-content {
  margin-bottom: 10px;
}

.alert-message {
  font-size: 0.95rem;
  color: #333;
  margin-bottom: 8px;
}

.alert-details {
  display: flex;
  gap: 20px;
  font-size: 0.85rem;
  color: #666;
}

.alert-actions {
  display: flex;
  gap: 10px;
}

.acknowledge-btn, .details-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.acknowledge-btn {
  background: #10b981;
  color: white;
}

.acknowledge-btn:hover {
  background: #059669;
}

.details-btn {
  background: #f0f0f0;
  color: #333;
}

.details-btn:hover {
  background: #e0e0e0;
}

.acknowledged-badge {
  background: #e0e0e0;
  color: #666;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.8rem;
}

/* 评估样式 */
.symbol-selector {
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  gap: 15px;
}

.symbol-inputs {
  display: flex;
  gap: 10px;
}

.symbol-inputs input {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
  width: 150px;
}

.assess-btn {
  background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.assess-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(220, 38, 38, 0.3);
}

.assess-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.assessment-result {
  margin-top: 20px;
}

.assessment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid #e0e0e0;
}

.assessment-header h3 {
  margin: 0;
  color: #333;
}

.assessment-time {
  font-size: 0.9rem;
  color: #666;
}

.risk-overview {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 30px;
  margin-bottom: 30px;
}

.risk-score-display {
  display: flex;
  justify-content: center;
  align-items: center;
}

.score-circle {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 2rem;
  font-weight: bold;
  color: white;
  margin-bottom: 10px;
}

.score-circle.low {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.score-circle.medium {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.score-circle.high {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
}

.score-circle.critical {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
}

.score-label {
  text-align: center;
  font-size: 1.1rem;
  font-weight: 600;
  color: #333;
}

.factors-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 15px;
}

.factor-item {
  display: flex;
  align-items: center;
  gap: 15px;
}

.factor-name {
  width: 120px;
  font-size: 0.9rem;
  color: #666;
}

.factor-bar {
  flex: 1;
  height: 12px;
  background: #f0f0f0;
  border-radius: 6px;
  overflow: hidden;
}

.factor-fill {
  height: 100%;
  border-radius: 6px;
  transition: width 0.3s ease;
}

.factor-fill.low {
  background: linear-gradient(90deg, #10b981 0%, #059669 100%);
}

.factor-fill.medium {
  background: linear-gradient(90deg, #f59e0b 0%, #d97706 100%);
}

.factor-fill.high {
  background: linear-gradient(90deg, #ef4444 0%, #dc2626 100%);
}

.factor-value {
  width: 60px;
  text-align: right;
  font-size: 0.9rem;
  font-weight: 600;
  color: #333;
}

.position-limits, .historical-risk {
  margin-bottom: 30px;
}

.position-limits h4, .historical-risk h4 {
  margin: 0 0 15px 0;
  color: #333;
}

.limits-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
}

.limit-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 8px;
}

.limit-label {
  font-size: 0.9rem;
  color: #666;
}

.limit-value {
  font-size: 1rem;
  font-weight: 600;
  color: #333;
}

.history-points {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.history-point {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
}

.history-point .date {
  font-size: 0.85rem;
  color: #666;
  width: 80px;
}

.history-point .score {
  font-size: 0.9rem;
  font-weight: 600;
  color: #333;
  width: 50px;
  text-align: center;
}

.history-point .pnl {
  font-size: 0.9rem;
  font-weight: 600;
  width: 60px;
  text-align: right;
}

.history-point .pnl.positive {
  color: #10b981;
}

.history-point .pnl.negative {
  color: #ef4444;
}

/* 组合分析样式 */
.portfolio-section {
  max-width: 100%;
}

.portfolio-input {
  margin-bottom: 30px;
}

.portfolio-input h3 {
  margin: 0 0 20px 0;
  color: #333;
}

.portfolio-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-weight: 600;
  color: #333;
}

.form-group input, .form-group select {
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.positions-input {
  grid-column: 1 / -1;
}

.position-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 15px;
  max-height: 200px;
  overflow-y: auto;
}

.position-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
}

.position-item .symbol {
  font-weight: 600;
  color: #333;
  width: 50px;
}

.position-item input {
  flex: 1;
  padding: 6px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.9rem;
  width: 80px;
}

.remove-btn {
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  cursor: pointer;
  font-size: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.add-position {
  display: flex;
  gap: 10px;
}

.add-position input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.add-btn {
  background: #10b981;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.add-btn:hover:not(:disabled) {
  background: #059669;
}

.add-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.analyze-btn {
  grid-column: 1 / -1;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.analyze-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.analyze-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.portfolio-result {
  margin-top: 30px;
}

.analysis-header {
  margin-bottom: 20px;
}

.analysis-header h3 {
  margin: 0 0 15px 0;
  color: #333;
}

.analysis-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
}

.summary-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
}

.summary-item .label {
  font-size: 0.9rem;
  color: #666;
}

.summary-item .value {
  font-size: 1.1rem;
  font-weight: 600;
  color: #333;
}

.summary-item .value.positive {
  color: #10b981;
}

.position-analysis {
  margin-bottom: 30px;
}

.position-analysis h4 {
  margin: 0 0 15px 0;
  color: #333;
}

.position-analysis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 15px;
}

.position-analysis-item {
  padding: 15px;
  background: #f8f9fa;
  border-radius: 8px;
}

.position-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.position-header .symbol {
  font-size: 1.1rem;
  font-weight: 600;
  color: #333;
}

.position-header .weight {
  font-size: 0.9rem;
  color: #666;
}

.position-details {
  display: flex;
  justify-content: space-between;
  gap: 20px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
}

.detail-item .label {
  font-size: 0.8rem;
  color: #666;
}

.detail-item .value {
  font-size: 1rem;
  font-weight: 600;
  color: #333;
}

.recommendations {
  margin-bottom: 20px;
}

.recommendations h4 {
  margin: 0 0 15px 0;
  color: #333;
}

.recommendations-list {
  padding-left: 20px;
}

.recommendations-list li {
  margin-bottom: 8px;
  color: #555;
  line-height: 1.5;
}

/* 空状态和错误状态 */
.empty-state, .error-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-icon, .error-icon {
  font-size: 4rem;
  margin-bottom: 20px;
}

.empty-state h3, .error-state h3 {
  color: #333;
  margin-bottom: 10px;
}

.empty-state p, .error-state p {
  color: #666;
  margin-bottom: 20px;
}

.retry-btn {
  background: #667eea;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.retry-btn:hover {
  background: #5a67d8;
}

@media (max-width: 768px) {
  .control-panel {
    flex-direction: column;
    gap: 20px;
  }

  .alert-header, .summary-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .risk-overview {
    grid-template-columns: 1fr;
  }

  .portfolio-form {
    grid-template-columns: 1fr;
  }

  .position-details {
    flex-direction: column;
    gap: 10px;
  }

  .analysis-summary {
    grid-template-columns: 1fr;
  }

  .position-analysis-grid {
    grid-template-columns: 1fr;
  }
}
</style>
