<template>
  <div class="risk-assessment-panel">
    <div class="panel-header">
      <h3>🔍 AI推荐风险评估</h3>
      <p>多维度分析AI推荐的风险特征，对比不同时期的表现</p>
    </div>

    <!-- 时间段选择 -->
    <div class="period-comparison">
      <div class="period-selection">
        <div class="period-card" v-for="(period, index) in comparisonPeriods" :key="index">
          <h4>时间段 {{ index + 1 }}</h4>
          <div class="period-inputs">
            <input type="date" v-model="period.startDate" :max="period.endDate" />
            <span>至</span>
            <input type="date" v-model="period.endDate" :min="period.startDate" :max="today" />
          </div>
          <div class="period-stats" v-if="period.riskData">
            <div class="stat-item">
              <span class="stat-label">整体风险：</span>
              <span class="stat-value" :class="getRiskClass(period.riskData.overallRiskScore)">
                {{ (period.riskData.overallRiskScore * 100).toFixed(1) }}%
              </span>
            </div>
          </div>
        </div>

        <div class="comparison-actions">
          <button @click="addPeriod" :disabled="comparisonPeriods.length >= 4">+</button>
          <button @click="removePeriod" :disabled="comparisonPeriods.length <= 2">-</button>
          <button @click="runComparison" :disabled="running" class="run-comparison-btn">
            {{ running ? '分析中...' : '开始对比分析' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 风险对比结果 -->
    <div v-if="comparisonResults" class="comparison-results">
      <!-- 整体风险对比 -->
      <div class="comparison-section">
        <h4>📊 整体风险评分对比</h4>
        <div class="chart-container">
          <BarChart
            :data="overallRiskData"
            :options="barChartOptions"
            v-if="overallRiskData.series.length > 0"
          />
        </div>
      </div>

      <!-- 风险维度雷达图 -->
      <div class="comparison-section">
        <h4>🎯 风险维度对比</h4>
        <div class="radar-container">
          <RadarChart
            :data="riskDimensionsData"
            :options="radarOptions"
            v-if="riskDimensionsData.series.length > 0"
          />
        </div>
      </div>

      <!-- VaR分析 -->
      <div class="comparison-section">
        <h4>💰 VaR (价值-at-风险) 分析</h4>
        <div class="var-analysis">
          <div class="var-table">
            <table>
              <thead>
                <tr>
                  <th>时间段</th>
                  <th>日VaR (95%)</th>
                  <th>周VaR (95%)</th>
                  <th>月VaR (95%)</th>
                  <th>预期亏空</th>
                  <th>最大回撤</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(period, index) in comparisonResults.periods" :key="index">
                  <td>{{ getPeriodLabel(index) }}</td>
                  <td class="var-value">{{ formatCurrency(period.var.daily) }}</td>
                  <td class="var-value">{{ formatCurrency(period.var.weekly) }}</td>
                  <td class="var-value">{{ formatCurrency(period.var.monthly) }}</td>
                  <td class="var-value">{{ formatCurrency(period.expectedShortfall) }}</td>
                  <td class="var-value negative">{{ (period.maxDrawdown * 100).toFixed(1) }}%</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="var-explanation">
            <h5>VaR指标说明</h5>
            <div class="explanation-items">
              <div class="explanation-item">
                <span class="explanation-label">日VaR(95%):</span>
                <span class="explanation-desc">在95%置信度下，一天可能的最大亏损</span>
              </div>
              <div class="explanation-item">
                <span class="explanation-label">预期亏空:</span>
                <span class="explanation-desc">VaR突破时的预期平均亏损</span>
              </div>
              <div class="explanation-item">
                <span class="explanation-label">最大回撤:</span>
                <span class="explanation-desc">从峰值到谷底的最大跌幅</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 波动率分析 -->
      <div class="comparison-section">
        <h4>📈 波动率分析</h4>
        <div class="volatility-analysis">
          <div class="volatility-charts">
            <div class="volatility-chart">
              <h5>波动率趋势</h5>
              <LineChart
                :xData="volatilityTrendData.xData"
                :series="volatilityTrendData.series"
                :yLabel="'波动率 (%)'"
                v-if="volatilityTrendData.xData.length > 0"
              />
            </div>

            <div class="volatility-distribution">
              <h5>波动率分布对比</h5>
              <BoxPlotChart
                :data="volatilityDistributionData"
                v-if="volatilityDistributionData.length > 0"
              />
            </div>
          </div>

          <div class="volatility-metrics">
            <div class="metric-cards">
              <div class="metric-card">
                <h6>平均波动率</h6>
                <div class="metric-values">
                  <div v-for="(period, index) in comparisonResults.periods" :key="index" class="metric-value">
                    <span class="period-label">{{ getPeriodLabel(index) }}:</span>
                    <span class="value">{{ (period.volatility.avg * 100).toFixed(1) }}%</span>
                  </div>
                </div>
              </div>

              <div class="metric-card">
                <h6>波动率范围</h6>
                <div class="metric-values">
                  <div v-for="(period, index) in comparisonResults.periods" :key="index" class="metric-value">
                    <span class="period-label">{{ getPeriodLabel(index) }}:</span>
                    <span class="value">{{ (period.volatility.min * 100).toFixed(1) }}% - {{ (period.volatility.max * 100).toFixed(1) }}%</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险归因分析 -->
      <div class="comparison-section">
        <h4>🔍 风险归因分析</h4>
        <div class="attribution-analysis">
          <div class="attribution-chart">
            <WaterfallChart
              :data="riskAttributionData"
              v-if="riskAttributionData.length > 0"
            />
          </div>

          <div class="attribution-breakdown">
            <div class="attribution-item" v-for="factor in riskAttributionFactors" :key="factor.name">
              <div class="factor-header">
                <span class="factor-name">{{ factor.name }}</span>
                <span class="factor-contribution" :class="factor.contribution >= 0 ? 'positive' : 'negative'">
                  {{ formatPercent(factor.contribution) }}
                </span>
              </div>
              <div class="factor-description">{{ factor.description }}</div>
              <div class="factor-weight">
                <div class="weight-bar">
                  <div
                    class="weight-fill"
                    :style="{ width: Math.abs(factor.contribution) * 100 + '%' }"
                    :class="factor.contribution >= 0 ? 'positive' : 'negative'"
                  ></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险缓解策略 -->
      <div class="comparison-section">
        <h4>🛡️ 风险缓解策略建议</h4>
        <div class="mitigation-strategies">
          <div class="strategy-cards">
            <div
              v-for="strategy in mitigationStrategies"
              :key="strategy.id"
              class="strategy-card"
              :class="strategy.priority"
            >
              <div class="strategy-header">
                <h5>{{ strategy.title }}</h5>
                <span class="priority-badge" :class="strategy.priority">
                  {{ getPriorityLabel(strategy.priority) }}
                </span>
              </div>

              <div class="strategy-content">
                <p>{{ strategy.description }}</p>

                <div class="strategy-metrics">
                  <div class="metric">
                    <span class="metric-label">风险降低预期：</span>
                    <span class="metric-value">{{ formatPercent(strategy.expectedRiskReduction) }}</span>
                  </div>

                  <div class="metric">
                    <span class="metric-label">实施难度：</span>
                    <span class="difficulty-level" :class="strategy.difficulty">
                      {{ getDifficultyLabel(strategy.difficulty) }}
                    </span>
                  </div>

                  <div class="metric">
                    <span class="metric-label">成本影响：</span>
                    <span class="cost-impact" :class="strategy.costImpact">
                      {{ getCostImpactLabel(strategy.costImpact) }}
                    </span>
                  </div>
                </div>

                <div class="strategy-actions">
                  <button @click="implementStrategy(strategy)">实施策略</button>
                  <button @click="viewStrategyDetails(strategy)">查看详情</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险警报 -->
      <div v-if="activeAlerts.length > 0" class="risk-alerts">
        <h4>🚨 风险警报</h4>
        <div class="alert-list">
          <div
            v-for="alert in activeAlerts"
            :key="alert.id"
            class="alert-item"
            :class="alert.severity"
          >
            <div class="alert-icon">{{ getAlertIcon(alert.type) }}</div>
            <div class="alert-content">
              <h4>{{ alert.title }}</h4>
              <p>{{ alert.description }}</p>
              <div class="alert-meta">
                <span class="alert-time">{{ formatDateTime(alert.timestamp) }}</span>
                <span class="alert-severity" :class="alert.severity">
                  {{ getSeverityLabel(alert.severity) }}
                </span>
              </div>
            </div>
            <div class="alert-actions">
              <button @click="acknowledgeAlert(alert)">确认</button>
              <button @click="viewAlertDetails(alert)">详情</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="running" class="loading-overlay">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">正在进行风险对比分析...</div>
        <div class="loading-progress">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="progress-text">{{ progressPercent.toFixed(0) }}%</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import BarChart from '../BarChart.vue'
import RadarChart from '../RadarChart.vue'
import LineChart from '../LineChart.vue'
import BoxPlotChart from '../BoxPlotChart.vue'
import WaterfallChart from '../WaterfallChart.vue'

export default {
  name: 'RiskAssessmentPanel',
  components: {
    BarChart,
    RadarChart,
    LineChart,
    BoxPlotChart,
    WaterfallChart
  },
  props: {
    symbols: {
      type: Array,
      default: () => ['BTC']
    },
    selectedDate: {
      type: String,
      default: null
    }
  },
  emits: ['risk-alert'],
  data() {
    return {
      running: false,
      progressPercent: 0,
      comparisonPeriods: [
        {
          startDate: this.getDefaultStartDate(0),
          endDate: this.getDefaultEndDate(0),
          riskData: null
        },
        {
          startDate: this.getDefaultStartDate(1),
          endDate: this.getDefaultEndDate(1),
          riskData: null
        }
      ],
      today: new Date().toISOString().split('T')[0],
      comparisonResults: null,
      activeAlerts: []
    }
  },
  computed: {
    overallRiskData() {
      if (!this.comparisonResults || !this.comparisonResults.periods) return { xData: [], series: [] }

      return {
        xData: this.comparisonResults.periods.map((_, index) => this.getPeriodLabel(index)),
        series: [{
          name: '整体风险评分',
          data: this.comparisonResults.periods.map(p => p.overallRiskScore * 100),
          type: 'bar',
          itemStyle: {
            color: function(params) {
              const value = params.data
              if (value < 30) return '#10b981'
              if (value < 60) return '#f59e0b'
              return '#ef4444'
            }
          }
        }]
      }
    },

    barChartOptions() {
      return {
        legend: { show: false },
        xAxis: { type: 'category' },
        yAxis: {
          type: 'value',
          name: '风险评分 (%)',
          axisLabel: { formatter: '{value}%' }
        },
        series: this.overallRiskData.series
      }
    },

    riskDimensionsData() {
      if (!this.comparisonResults || !this.comparisonResults.periods) return { series: [] }

      const indicators = ['市场风险', '波动风险', '流动性风险', '执行风险', '模型风险']
      const series = this.comparisonResults.periods.map((period, index) => ({
        name: this.getPeriodLabel(index),
        type: 'radar',
        data: [{
          value: [
            period.marketRisk * 100,
            period.volatilityRisk * 100,
            period.liquidityRisk * 100,
            period.executionRisk * 100,
            period.modelRisk * 100
          ],
          name: this.getPeriodLabel(index)
        }]
      }))

      return { series }
    },

    radarOptions() {
      return {
        legend: { data: this.comparisonResults ? this.comparisonResults.periods.map((_, index) => this.getPeriodLabel(index)) : [] },
        radar: {
          indicator: [
            { name: '市场风险', max: 100 },
            { name: '波动风险', max: 100 },
            { name: '流动性风险', max: 100 },
            { name: '执行风险', max: 100 },
            { name: '模型风险', max: 100 }
          ]
        },
        series: this.riskDimensionsData.series
      }
    },

    volatilityTrendData() {
      if (!this.comparisonResults || !this.comparisonResults.periods) return { xData: [], series: [] }

      return {
        xData: this.comparisonResults.periods.map((_, index) => this.getPeriodLabel(index)),
        series: [{
          name: '平均波动率',
          data: this.comparisonResults.periods.map(p => p.volatility.avg * 100),
          type: 'line',
          smooth: true
        }]
      }
    },

    volatilityDistributionData() {
      if (!this.comparisonResults || !this.comparisonResults.periods) return []

      return this.comparisonResults.periods.map((period, index) => ({
        name: this.getPeriodLabel(index),
        data: period.volatility.distribution || []
      }))
    },

    riskAttributionData() {
      if (!this.comparisonResults || !this.comparisonResults.attribution) return []

      return this.comparisonResults.attribution.factors.map(factor => ({
        name: factor.name,
        value: factor.contribution * 100
      }))
    },

    riskAttributionFactors() {
      return [
        { name: '市场风险', contribution: 0.35, description: '系统性风险，影响所有资产' },
        { name: '波动率风险', contribution: 0.28, description: '价格波动带来的不确定性' },
        { name: '流动性风险', contribution: 0.15, description: '买卖差价和交易成本' },
        { name: '执行风险', contribution: 0.12, description: '订单执行时的滑点风险' },
        { name: '模型风险', contribution: 0.10, description: 'AI预测模型的局限性' }
      ]
    },

    mitigationStrategies() {
      return [
        {
          id: 1,
          title: '分散投资组合',
          description: '通过投资多种资产降低非系统性风险',
          priority: 'high',
          expectedRiskReduction: 0.25,
          difficulty: 'medium',
          costImpact: 'low'
        },
        {
          id: 2,
          title: '动态止损机制',
          description: '根据市场波动自动调整止损位',
          priority: 'high',
          expectedRiskReduction: 0.18,
          difficulty: 'low',
          costImpact: 'low'
        },
        {
          id: 3,
          title: '规模控制',
          description: '根据风险评估调整单次投资规模',
          priority: 'medium',
          expectedRiskReduction: 0.15,
          difficulty: 'low',
          costImpact: 'none'
        },
        {
          id: 4,
          title: '期权对冲',
          description: '使用期权合约对冲下跌风险',
          priority: 'medium',
          expectedRiskReduction: 0.30,
          difficulty: 'high',
          costImpact: 'high'
        }
      ]
    }
  },
  methods: {
    getDefaultStartDate(offset) {
      const date = new Date()
      date.setMonth(date.getMonth() - (3 + offset * 3))
      return date.toISOString().split('T')[0]
    },

    getDefaultEndDate(offset) {
      const date = new Date()
      date.setMonth(date.getMonth() - offset * 3)
      date.setDate(date.getDate() - 1)
      return date.toISOString().split('T')[0]
    },

    addPeriod() {
      if (this.comparisonPeriods.length >= 4) return

      const lastPeriod = this.comparisonPeriods[this.comparisonPeriods.length - 1]
      const newStartDate = new Date(lastPeriod.startDate)
      newStartDate.setMonth(newStartDate.getMonth() - 3)
      const newEndDate = new Date(lastPeriod.startDate)
      newEndDate.setDate(newEndDate.getDate() - 1)

      this.comparisonPeriods.push({
        startDate: newStartDate.toISOString().split('T')[0],
        endDate: newEndDate.toISOString().split('T')[0],
        riskData: null
      })
    },

    removePeriod() {
      if (this.comparisonPeriods.length <= 2) return
      this.comparisonPeriods.pop()
    },

    async runComparison() {
      if (this.comparisonPeriods.length < 2) return

      this.running = true
      this.progressPercent = 0
      this.comparisonResults = null

      try {
        // 模拟进度
        const progressInterval = setInterval(() => {
          this.progressPercent += Math.random() * 20
          if (this.progressPercent > 90) {
            this.progressPercent = 90
          }
        }, 300)

        // 执行风险对比分析
        this.comparisonResults = await this.performRiskComparison()

        clearInterval(progressInterval)
        this.progressPercent = 100

        // 生成风险警报
        this.activeAlerts = this.generateRiskAlerts()

      } catch (error) {
        console.error('风险对比分析失败:', error)
        this.$message.error('风险对比分析失败，请稍后重试')
      } finally {
        this.running = false
        setTimeout(() => {
          this.progressPercent = 0
        }, 1000)
      }
    },

    async performRiskComparison() {
      // 模拟风险对比分析
      const periods = []

      for (let i = 0; i < this.comparisonPeriods.length; i++) {
        periods.push({
          overallRiskScore: 0.3 + Math.random() * 0.5,
          marketRisk: 0.25 + Math.random() * 0.4,
          volatilityRisk: 0.2 + Math.random() * 0.5,
          liquidityRisk: 0.1 + Math.random() * 0.3,
          executionRisk: 0.15 + Math.random() * 0.4,
          modelRisk: 0.1 + Math.random() * 0.2,
          var: {
            daily: -1500 - Math.random() * 2000,
            weekly: -3500 - Math.random() * 4000,
            monthly: -12000 - Math.random() * 10000
          },
          expectedShortfall: -2000 - Math.random() * 3000,
          maxDrawdown: 0.05 + Math.random() * 0.15,
          volatility: {
            avg: 0.15 + Math.random() * 0.2,
            min: 0.05 + Math.random() * 0.1,
            max: 0.3 + Math.random() * 0.3,
            distribution: Array.from({length: 20}, () => Math.random() * 0.4)
          }
        })
      }

      return {
        periods,
        attribution: {
          factors: this.riskAttributionFactors
        }
      }
    },

    generateRiskAlerts() {
      const alerts = []
      const highRiskPeriod = Math.floor(Math.random() * this.comparisonPeriods.length)

      alerts.push({
        id: 1,
        type: 'volatility',
        title: '波动率风险上升',
        description: `时间段${highRiskPeriod + 1}的波动率显著高于其他时期，建议调整仓位`,
        severity: 'medium',
        timestamp: new Date()
      })

      if (Math.random() > 0.6) {
        alerts.push({
          id: 2,
          type: 'drawdown',
          title: '回撤风险警告',
          description: '检测到潜在的较大回撤风险，建议实施风险控制措施',
          severity: 'high',
          timestamp: new Date()
        })
      }

      return alerts
    },

    implementStrategy(strategy) {
      console.log('实施策略:', strategy)
      // 这里可以跳转到策略实施页面
    },

    viewStrategyDetails(strategy) {
      console.log('查看策略详情:', strategy)
      // 这里可以打开策略详情弹窗
    },

    acknowledgeAlert(alert) {
      this.activeAlerts = this.activeAlerts.filter(a => a.id !== alert.id)
      this.$emit('risk-alert', { type: 'acknowledged', alert })
    },

    viewAlertDetails(alert) {
      console.log('查看警报详情:', alert)
      // 这里可以打开警报详情弹窗
    },

    // 辅助方法
    getPeriodLabel(index) {
      return `时间段${index + 1}`
    },

    getRiskClass(score) {
      if (score < 0.3) return 'low'
      if (score < 0.6) return 'medium'
      return 'high'
    },

    getPriorityLabel(priority) {
      const labels = { high: '高', medium: '中', low: '低' }
      return labels[priority] || priority
    },

    getDifficultyLabel(difficulty) {
      const labels = { low: '容易', medium: '中等', high: '困难' }
      return labels[difficulty] || difficulty
    },

    getCostImpactLabel(impact) {
      const labels = { none: '无成本', low: '低成本', medium: '中等成本', high: '高成本' }
      return labels[impact] || impact
    },

    getAlertIcon(type) {
      const icons = {
        volatility: '📊',
        drawdown: '📉',
        liquidity: '💧',
        execution: '⚡'
      }
      return icons[type] || '🚨'
    },

    getSeverityLabel(severity) {
      const labels = { low: '低', medium: '中', high: '高', critical: '严重' }
      return labels[severity] || severity
    },

    formatCurrency(value) {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 0,
        maximumFractionDigits: 0
      }).format(value)
    },

    formatPercent(value) {
      return (value * 100).toFixed(1) + '%'
    },

    formatDateTime(date) {
      return new Date(date).toLocaleString('zh-CN')
    }
  }
}
</script>

<style scoped>
.risk-assessment-panel {
  padding: 24px;
}

.panel-header {
  text-align: center;
  margin-bottom: 32px;
}

.panel-header h3 {
  margin: 0 0 8px 0;
  font-size: 1.5rem;
  color: #1f2937;
}

.panel-header p {
  margin: 0;
  color: #6b7280;
}

.period-comparison {
  background: #f8fafc;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
}

.period-selection {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)) auto;
  gap: 20px;
  align-items: start;
  margin-bottom: 24px;
}

.period-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.period-card h4 {
  margin: 0 0 12px 0;
  color: #1f2937;
  font-size: 1rem;
}

.period-inputs {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.period-inputs input {
  padding: 6px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
}

.period-stats {
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.stat-item:last-child {
  margin-bottom: 0;
}

.stat-label {
  font-size: 14px;
  color: #6b7280;
}

.stat-value {
  font-weight: 600;
}

.stat-value.low {
  color: #10b981;
}

.stat-value.medium {
  color: #f59e0b;
}

.stat-value.high {
  color: #ef4444;
}

.comparison-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.comparison-actions button {
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s ease;
}

.comparison-actions button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.run-comparison-btn {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  color: white;
}

.run-comparison-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.comparison-results {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.comparison-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.comparison-section h4 {
  margin: 0 0 20px 0;
  color: #1f2937;
}

.chart-container, .radar-container {
  height: 350px;
}

.var-analysis {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 24px;
}

.var-table {
  overflow-x: auto;
}

.var-table table {
  width: 100%;
  border-collapse: collapse;
}

.var-table th, .var-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.var-table th {
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
}

.var-value {
  font-weight: 500;
}

.var-value.negative {
  color: #ef4444;
}

.var-explanation h5 {
  margin: 0 0 12px 0;
  color: #374151;
}

.explanation-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.explanation-item {
  display: flex;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid #f3f4f6;
}

.explanation-item:last-child {
  border-bottom: none;
}

.explanation-label {
  font-weight: 600;
  color: #374151;
  min-width: 100px;
}

.explanation-desc {
  color: #6b7280;
  flex: 1;
}

.volatility-analysis {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 24px;
}

.volatility-charts {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.volatility-chart, .volatility-distribution {
  background: #f9fafb;
  border-radius: 8px;
  padding: 16px;
}

.volatility-chart h5, .volatility-distribution h5 {
  margin: 0 0 12px 0;
  color: #374151;
  font-size: 1rem;
}

.volatility-distribution {
  height: 250px;
}

.volatility-metrics {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.metric-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.metric-card h6 {
  margin: 0 0 12px 0;
  color: #374151;
  font-size: 1rem;
}

.metric-values {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric-value {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid #f3f4f6;
}

.metric-value:last-child {
  border-bottom: none;
}

.period-label {
  font-weight: 500;
  color: #6b7280;
}

.value {
  font-weight: 600;
  color: #1f2937;
}

.attribution-analysis {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.attribution-chart {
  height: 300px;
  background: #f9fafb;
  border-radius: 8px;
  padding: 16px;
}

.attribution-breakdown {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.attribution-item {
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.factor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.factor-name {
  font-weight: 600;
  color: #1f2937;
}

.factor-contribution {
  font-weight: 700;
}

.factor-contribution.positive {
  color: #ef4444;
}

.factor-contribution.negative {
  color: #10b981;
}

.factor-description {
  color: #6b7280;
  font-size: 14px;
  margin-bottom: 12px;
}

.factor-weight {
  margin-top: 8px;
}

.weight-bar {
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 4px;
}

.weight-fill {
  height: 100%;
  transition: width 0.3s ease;
}

.weight-fill.positive {
  background: linear-gradient(90deg, #10b981 0%, #f59e0b 50%, #ef4444 100%);
}

.weight-fill.negative {
  background: linear-gradient(90deg, #ef4444 0%, #f59e0b 50%, #10b981 100%);
}

.mitigation-strategies {
  margin-top: 24px;
}

.strategy-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.strategy-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  border-left: 4px solid;
  transition: all 0.2s ease;
}

.strategy-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.strategy-card.high {
  border-left-color: #ef4444;
}

.strategy-card.medium {
  border-left-color: #f59e0b;
}

.strategy-card.low {
  border-left-color: #10b981;
}

.strategy-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.strategy-header h5 {
  margin: 0;
  color: #1f2937;
}

.priority-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.priority-badge.high {
  background: #fef2f2;
  color: #dc2626;
}

.priority-badge.medium {
  background: #fefce8;
  color: #d97706;
}

.priority-badge.low {
  background: #f0fdf4;
  color: #16a34a;
}

.strategy-content p {
  margin: 0 0 16px 0;
  color: #6b7280;
  line-height: 1.5;
}

.strategy-metrics {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.metric {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid #f3f4f6;
}

.metric:last-child {
  border-bottom: none;
}

.metric-label {
  font-size: 14px;
  color: #6b7280;
}

.metric-value {
  font-weight: 600;
  color: #1f2937;
}

.difficulty-level, .cost-impact {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
  font-weight: 500;
}

.difficulty-level.低, .difficulty-level.容易 {
  background: #dcfce7;
  color: #166534;
}

.difficulty-level.中等 {
  background: #fef3c7;
  color: #92400e;
}

.difficulty-level.困难 {
  background: #fee2e2;
  color: #991b1b;
}

.cost-impact.无成本, .cost-impact.低成本 {
  background: #dcfce7;
  color: #166534;
}

.cost-impact.中等成本 {
  background: #fef3c7;
  color: #92400e;
}

.cost-impact.高成本 {
  background: #fee2e2;
  color: #991b1b;
}

.strategy-actions {
  display: flex;
  gap: 8px;
}

.strategy-actions button {
  padding: 6px 12px;
  border: 1px solid #d1d5db;
  background: white;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.strategy-actions button:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

.risk-alerts {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.risk-alerts h4 {
  margin: 0 0 20px 0;
  color: #1f2937;
}

.alert-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.alert-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid;
}

.alert-item.medium {
  background: #fefce8;
  border-left-color: #f59e0b;
}

.alert-item.high {
  background: #fef2f2;
  border-left-color: #ef4444;
}

.alert-item.critical {
  background: #fef2f2;
  border-left-color: #dc2626;
}

.alert-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.alert-content {
  flex: 1;
}

.alert-content h4 {
  margin: 0 0 8px 0;
  color: #1f2937;
}

.alert-content p {
  margin: 0 0 12px 0;
  color: #6b7280;
  line-height: 1.5;
}

.alert-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #9ca3af;
}

.alert-severity {
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: 500;
}

.alert-severity.low {
  background: #dcfce7;
  color: #166534;
}

.alert-severity.medium {
  background: #fef3c7;
  color: #92400e;
}

.alert-severity.high {
  background: #fee2e2;
  color: #991b1b;
}

.alert-severity.critical {
  background: #7f1d1d;
  color: white;
}

.alert-actions {
  display: flex;
  gap: 6px;
}

.alert-actions button {
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  background: white;
  border-radius: 3px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.alert-actions button:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

.loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.loading-content {
  background: white;
  border-radius: 12px;
  padding: 32px;
  text-align: center;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #e5e7eb;
  border-top: 4px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-text {
  color: #374151;
  font-weight: 500;
  margin-bottom: 16px;
}

.loading-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
}

.progress-bar {
  width: 200px;
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #1d4ed8 100%);
  transition: width 0.3s ease;
}

.progress-text {
  font-weight: 600;
  color: #374151;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .period-selection {
    grid-template-columns: 1fr;
  }

  .comparison-section {
    padding: 16px;
  }

  .var-analysis, .volatility-analysis, .attribution-analysis {
    grid-template-columns: 1fr;
  }

  .strategy-cards {
    grid-template-columns: 1fr;
  }

  .alert-item {
    flex-direction: column;
    gap: 8px;
  }

  .alert-meta {
    justify-content: space-between;
  }
}
</style>
