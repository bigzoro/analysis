<template>
  <div class="historical-analysis-panel">
    <div class="panel-header">
      <h3>🎯 历史分析仪表板</h3>
      <p>研究过去特定时间点的市场状况和推荐质量</p>
    </div>

    <!-- 时间段选择器 -->
    <div class="period-selector">
      <div class="preset-periods">
        <button
          v-for="period in presetPeriods"
          :key="period.key"
          @click="selectPresetPeriod(period)"
          :class="{ active: selectedPeriod && selectedPeriod.key === period.key }"
          class="preset-btn"
        >
          {{ period.label }}
        </button>
      </div>

      <div class="custom-period">
        <div class="date-input-group">
          <label>自定义时间段：</label>
          <input type="date" v-model="customStartDate" :max="customEndDate" />
          <span>至</span>
          <input type="date" v-model="customEndDate" :min="customStartDate" :max="today" />
          <button @click="applyCustomPeriod" class="apply-btn">应用</button>
        </div>
      </div>
    </div>

    <!-- 市场概览卡片 -->
    <div v-if="marketOverview" class="market-overview">
      <div class="overview-cards">
        <div class="overview-card">
          <div class="card-icon">📊</div>
          <div class="card-content">
            <div class="card-title">市场环境分布</div>
            <div class="card-value">{{ marketOverview.marketCondition }}</div>
            <div class="card-desc">主要市场特征</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">📈</div>
          <div class="card-content">
            <div class="card-title">波动率水平</div>
            <div class="card-value">{{ (marketOverview.volatility * 100).toFixed(1) }}%</div>
            <div class="card-desc" :class="getVolatilityLevel(marketOverview.volatility)">
              {{ getVolatilityDesc(marketOverview.volatility) }}
            </div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">🎯</div>
          <div class="card-content">
            <div class="card-title">推荐准确率</div>
            <div class="card-value" :class="marketOverview.accuracy >= 0.6 ? 'good' : 'fair'">
              {{ (marketOverview.accuracy * 100).toFixed(1) }}%
            </div>
            <div class="card-desc">{{ marketOverview.totalRecommendations }} 个推荐</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">💰</div>
          <div class="card-content">
            <div class="card-title">平均收益率</div>
            <div class="card-value" :class="marketOverview.avgReturn >= 0 ? 'positive' : 'negative'">
              {{ (marketOverview.avgReturn * 100).toFixed(1) }}%
            </div>
            <div class="card-desc">推荐持有期收益</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 分析图表区域 -->
    <div class="analysis-charts">
      <!-- 市场趋势图 -->
      <div class="chart-section">
        <h4>📈 市场趋势分析</h4>
        <div class="chart-container">
          <LineChart
            :xData="marketTrendData.xData"
            :series="marketTrendData.series"
            :yAxis="marketTrendData.yAxis"
            v-if="marketTrendData.xData.length > 0"
          />
          <div v-else class="no-data">暂无数据</div>
        </div>
      </div>

      <!-- 推荐表现对比 -->
      <div class="chart-section">
        <h4>🎯 推荐表现对比</h4>
        <div class="chart-container">
          <BarChart
            :data="performanceComparisonData"
            :options="performanceChartOptions"
            v-if="performanceComparisonData.series.length > 0"
          />
          <div v-else class="no-data">暂无数据</div>
        </div>
      </div>
    </div>

    <!-- 详细分析表格 -->
    <div class="detailed-analysis">
      <h4>📋 详细分析</h4>

      <!-- 市场环境详细分析 -->
      <div class="analysis-table-section">
        <h5>市场环境分析</h5>
        <div class="table-container">
          <table class="analysis-table">
            <thead>
              <tr>
                <th>时间段</th>
                <th>市场环境</th>
                <th>波动率</th>
                <th>趋势强度</th>
                <th>推荐数量</th>
                <th>准确率</th>
                <th>平均收益</th>
                <th>风险等级</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="period in detailedPeriods"
                :key="period.id"
                @click="selectPeriodDetail(period)"
                :class="{ selected: selectedPeriodDetail && selectedPeriodDetail.id === period.id }"
                class="period-row"
              >
                <td>{{ formatDateRange(period.startDate, period.endDate) }}</td>
                <td>
                  <span class="market-condition" :class="period.marketCondition">
                    {{ getMarketConditionLabel(period.marketCondition) }}
                  </span>
                </td>
                <td>{{ (period.volatility * 100).toFixed(1) }}%</td>
                <td>
                  <div class="trend-strength-bar">
                    <div class="strength-fill" :style="{ width: period.trendStrength * 100 + '%' }"></div>
                    <span class="strength-text">{{ (period.trendStrength * 100).toFixed(0) }}%</span>
                  </div>
                </td>
                <td>{{ period.recommendationCount }}</td>
                <td :class="period.accuracy >= 0.6 ? 'good' : period.accuracy >= 0.4 ? 'fair' : 'poor'">
                  {{ (period.accuracy * 100).toFixed(1) }}%
                </td>
                <td :class="period.avgReturn >= 0 ? 'positive' : 'negative'">
                  {{ (period.avgReturn * 100).toFixed(1) }}%
                </td>
                <td>
                  <span class="risk-level" :class="period.riskLevel">
                    {{ getRiskLevelLabel(period.riskLevel) }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 技术指标热力图 -->
      <div v-if="selectedPeriodDetail" class="technical-heatmap">
        <h5>技术指标热力图 - {{ formatDateRange(selectedPeriodDetail.startDate, selectedPeriodDetail.endDate) }}</h5>
        <div class="heatmap-container">
          <div class="heatmap-grid">
          <div
            v-for="indicator in (selectedPeriodDetail?.technicalIndicators || [])"
            :key="indicator.name"
            class="heatmap-cell"
              :style="{ backgroundColor: getHeatmapColor(indicator.value, indicator.min, indicator.max) }"
            >
              <div class="indicator-name">{{ indicator.name }}</div>
              <div class="indicator-value">{{ indicator.value.toFixed(2) }}</div>
            </div>
          </div>
        </div>

        <div class="heatmap-legend">
          <div class="legend-item">
            <div class="legend-color" style="background: #dc2626"></div>
            <span>高值</span>
          </div>
          <div class="legend-item">
            <div class="legend-color" style="background: #ea580c"></div>
            <span>较高</span>
          </div>
          <div class="legend-item">
            <div class="legend-color" style="background: #ca8a04"></div>
            <span>中等</span>
          </div>
          <div class="legend-item">
            <div class="legend-color" style="background: #16a34a"></div>
            <span>较低</span>
          </div>
          <div class="legend-item">
            <div class="legend-color" style="background: #2563eb"></div>
            <span>低值</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 关键洞察 -->
    <div v-if="insights.length > 0" class="key-insights">
      <h4>🔍 关键洞察</h4>
      <div class="insights-grid">
        <div
          v-for="insight in insights"
          :key="insight.id"
          class="insight-card"
          :class="insight.priority"
        >
          <div class="insight-header">
            <span class="insight-type">{{ insight.type }}</span>
            <span class="insight-priority" :class="insight.priority">
              {{ getPriorityLabel(insight.priority) }}
            </span>
          </div>
          <div class="insight-content">
            <h5>{{ insight.title }}</h5>
            <p>{{ insight.description }}</p>
            <div class="insight-metrics">
              <span class="metric">置信度: {{ (insight.confidence * 100).toFixed(0) }}%</span>
              <span class="metric">影响度: {{ getImpactLabel(insight.impact) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import LineChart from '../LineChart.vue'
import BarChart from '../BarChart.vue'

export default {
  name: 'HistoricalAnalysisPanel',
  components: {
    LineChart,
    BarChart
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
  emits: ['period-selected'],
  data() {
    return {
      selectedPeriod: null,
      customStartDate: null,
      customEndDate: null,
      today: new Date().toISOString().split('T')[0],
      marketOverview: null,
      detailedPeriods: [],
      selectedPeriodDetail: null,
      insights: [],
      presetPeriods: [
        {
          key: '1w',
          label: '最近1周',
          days: 7
        },
        {
          key: '1m',
          label: '最近1月',
          days: 30
        },
        {
          key: '3m',
          label: '最近3月',
          days: 90
        },
        {
          key: '6m',
          label: '最近6月',
          days: 180
        },
        {
          key: '1y',
          label: '最近1年',
          days: 365
        }
      ]
    }
  },
  computed: {
    marketTrendData() {
      if (!this.detailedPeriods.length) return { xData: [], series: [], yAxis: [] }

      return {
        xData: this.detailedPeriods.map(p => this.formatDateRange(p.startDate, p.endDate)),
        yAxis: [
          {
            type: 'value',
            name: '波动率 (%)',
            position: 'left',
            axisLabel: { color: '#98a2b3' },
            splitLine: { lineStyle: { color: '#1f2836' } }
          },
          {
            type: 'value',
            name: '准确率 (%)',
            position: 'right',
            axisLabel: { color: '#98a2b3' },
            splitLine: { show: false }
          }
        ],
        series: [
          {
            name: '波动率',
            data: this.detailedPeriods.map(p => p.volatility * 100),
            type: 'line',
            smooth: true,
            yAxisIndex: 0
          },
          {
            name: '推荐准确率',
            data: this.detailedPeriods.map(p => p.accuracy * 100),
            type: 'line',
            smooth: true,
            yAxisIndex: 1
          }
        ]
      }
    },

    performanceComparisonData() {
      if (!this.detailedPeriods.length) return { xData: [], series: [] }

      return {
        xData: this.detailedPeriods.map(p => this.getMarketConditionLabel(p.marketCondition)),
        series: [
          {
            name: '平均收益率',
            data: this.detailedPeriods.map(p => p.avgReturn * 100),
            type: 'bar'
          },
          {
            name: '推荐准确率',
            data: this.detailedPeriods.map(p => p.accuracy * 100),
            type: 'bar'
          }
        ]
      }
    },

    performanceChartOptions() {
      return {
        legend: { data: ['平均收益率', '推荐准确率'] },
        xAxis: { type: 'category' },
        yAxis: { type: 'value' },
        series: this.performanceComparisonData.series
      }
    }
  },
  mounted() {
    this.selectPresetPeriod(this.presetPeriods[2]) // 默认选择3个月
  },
  methods: {
    selectPresetPeriod(period) {
      this.selectedPeriod = period
      const endDate = new Date()
      const startDate = new Date()
      startDate.setDate(endDate.getDate() - period.days)

      this.customStartDate = startDate.toISOString().split('T')[0]
      this.customEndDate = endDate.toISOString().split('T')[0]

      this.loadHistoricalData()
    },

    applyCustomPeriod() {
      if (!this.customStartDate || !this.customEndDate) return

      this.selectedPeriod = null
      this.loadHistoricalData()
    },

    async loadHistoricalData() {
      // 这里应该调用后端API获取历史分析数据
      // 目前使用模拟数据

      this.marketOverview = {
        marketCondition: 'bull_market',
        volatility: 0.25,
        accuracy: 0.68,
        avgReturn: 0.15,
        totalRecommendations: 245
      }

      this.detailedPeriods = this.generateMockPeriods()
      this.insights = this.generateMockInsights()
    },

    generateMockPeriods() {
      const periods = []
      const marketConditions = ['bull_market', 'bear_market', 'sideways', 'volatile']

      for (let i = 0; i < 12; i++) {
        const startDate = new Date()
        startDate.setMonth(startDate.getMonth() - (11 - i))
        startDate.setDate(1)

        const endDate = new Date(startDate)
        endDate.setMonth(endDate.getMonth() + 1)
        endDate.setDate(0)

        periods.push({
          id: i + 1,
          startDate: startDate.toISOString().split('T')[0],
          endDate: endDate.toISOString().split('T')[0],
          marketCondition: marketConditions[Math.floor(Math.random() * marketConditions.length)],
          volatility: 0.1 + Math.random() * 0.3,
          trendStrength: Math.random(),
          recommendationCount: Math.floor(Math.random() * 50) + 10,
          accuracy: 0.4 + Math.random() * 0.4,
          avgReturn: (Math.random() - 0.3) * 0.4,
          riskLevel: Math.random() > 0.7 ? 'high' : Math.random() > 0.4 ? 'medium' : 'low'
        })
      }

      return periods
    },

    generateMockInsights() {
      return [
        {
          id: 1,
          type: '📊',
          title: '波动率与准确率相关性',
          description: '在低波动市场环境中，AI推荐准确率显著高于高波动环境',
          priority: 'high',
          confidence: 0.85,
          impact: 'high'
        },
        {
          id: 2,
          type: '🎯',
          title: '市场时机把握',
          description: 'AI在上涨初期推荐的准确率比下跌末期高出25%',
          priority: 'high',
          confidence: 0.78,
          impact: 'medium'
        },
        {
          id: 3,
          type: '📈',
          title: '持有期最优策略',
          description: '7-14天的推荐持有期收益最优，过长或过短都会降低表现',
          priority: 'medium',
          confidence: 0.72,
          impact: 'high'
        },
        {
          id: 4,
          type: '⚠️',
          title: '风险集中度警告',
          description: '在极端市场条件下，AI推荐可能出现过度集中于特定策略',
          priority: 'medium',
          confidence: 0.65,
          impact: 'medium'
        }
      ]
    },

    selectPeriodDetail(period) {
      this.selectedPeriodDetail = {
        ...period,
        technicalIndicators: this.generateMockTechnicalIndicators()
      }
      this.$emit('period-selected', period)
    },

    generateMockTechnicalIndicators() {
      const indicators = [
        'RSI', 'MACD', '布林带上轨', '布林带下轨', 'MA5', 'MA20', 'MA50',
        '威廉指标', '随机指标K', '随机指标D', 'CCI', '动量指标'
      ]

      return indicators.map(name => ({
        name,
        value: Math.random() * 100,
        min: 0,
        max: 100
      }))
    },

    // 辅助方法
    getVolatilityLevel(volatility) {
      if (volatility < 0.15) return 'low'
      if (volatility < 0.25) return 'medium'
      return 'high'
    },

    getVolatilityDesc(volatility) {
      if (volatility < 0.15) return '低波动'
      if (volatility < 0.25) return '中等波动'
      return '高波动'
    },

    getMarketConditionLabel(condition) {
      const labels = {
        bull_market: '牛市',
        bear_market: '熊市',
        sideways: '震荡市',
        volatile: '高波动'
      }
      return labels[condition] || condition
    },

    getRiskLevelLabel(level) {
      const labels = {
        low: '低风险',
        medium: '中风险',
        high: '高风险'
      }
      return labels[level] || level
    },

    getHeatmapColor(value, min, max) {
      const ratio = (value - min) / (max - min)
      if (ratio > 0.8) return '#dc2626' // 红色 - 高值
      if (ratio > 0.6) return '#ea580c' // 橙色 - 较高
      if (ratio > 0.4) return '#ca8a04' // 黄色 - 中等
      if (ratio > 0.2) return '#16a34a' // 绿色 - 较低
      return '#2563eb' // 蓝色 - 低值
    },

    getPriorityLabel(priority) {
      const labels = { high: '高', medium: '中', low: '低' }
      return labels[priority] || priority
    },

    getImpactLabel(impact) {
      const labels = { high: '高', medium: '中', low: '低' }
      return labels[impact] || impact
    },

    formatDateRange(start, end) {
      const startDate = new Date(start)
      const endDate = new Date(end)
      return `${startDate.getMonth() + 1}/${startDate.getDate()} - ${endDate.getMonth() + 1}/${endDate.getDate()}`
    }
  }
}
</script>

<style scoped>
.historical-analysis-panel {
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

.period-selector {
  background: #f8fafc;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
}

.preset-periods {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.preset-btn {
  padding: 8px 16px;
  border: 2px solid #e5e7eb;
  background: white;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s ease;
}

.preset-btn:hover {
  border-color: #3b82f6;
}

.preset-btn.active {
  background: #3b82f6;
  color: white;
  border-color: #3b82f6;
}

.custom-period {
  display: flex;
  align-items: center;
  gap: 12px;
}

.date-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-input-group input {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
}

.apply-btn {
  padding: 8px 16px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
}

.apply-btn:hover {
  background: #2563eb;
}

.market-overview {
  margin-bottom: 32px;
}

.overview-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.overview-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 16px;
}

.card-icon {
  font-size: 2rem;
}

.card-content {
  flex: 1;
}

.card-title {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 4px;
}

.card-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 4px;
}

.card-value.good {
  color: #10b981;
}

.card-value.fair {
  color: #f59e0b;
}

.card-value.positive {
  color: #10b981;
}

.card-value.negative {
  color: #ef4444;
}

.card-desc {
  font-size: 13px;
  color: #9ca3af;
}

.card-desc.low {
  color: #10b981;
}

.card-desc.medium {
  color: #f59e0b;
}

.card-desc.high {
  color: #ef4444;
}

.analysis-charts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  margin-bottom: 32px;
}

.chart-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.chart-section h4 {
  margin: 0 0 20px 0;
  color: #1f2937;
}

.chart-container {
  height: 300px;
  position: relative;
}

.no-data {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #9ca3af;
  font-size: 1rem;
}

.detailed-analysis {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 24px;
}

.detailed-analysis h4 {
  margin: 0 0 24px 0;
  color: #1f2937;
}

.analysis-table-section h5 {
  margin: 0 0 16px 0;
  color: #374151;
  font-size: 1.1rem;
}

.table-container {
  overflow-x: auto;
  margin-bottom: 32px;
}

.analysis-table {
  width: 100%;
  border-collapse: collapse;
}

.analysis-table th, .analysis-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.analysis-table th {
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
}

.period-row {
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.period-row:hover {
  background: #f9fafb;
}

.period-row.selected {
  background: #eff6ff;
  border-left: 4px solid #3b82f6;
}

.market-condition {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.market-condition.bull_market {
  background: #dcfce7;
  color: #166534;
}

.market-condition.bear_market {
  background: #fee2e2;
  color: #991b1b;
}

.market-condition.sideways {
  background: #fef3c7;
  color: #92400e;
}

.market-condition.volatile {
  background: #fed7d7;
  color: #c53030;
}

.trend-strength-bar {
  position: relative;
  width: 80px;
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.strength-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981 0%, #f59e0b 50%, #ef4444 100%);
  transition: width 0.3s ease;
}

.strength-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 10px;
  font-weight: 600;
  color: #374151;
}

.good {
  color: #10b981;
}

.fair {
  color: #f59e0b;
}

.poor {
  color: #ef4444;
}

.positive {
  color: #10b981;
}

.negative {
  color: #ef4444;
}

.risk-level {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 500;
}

.risk-level.low {
  background: #dcfce7;
  color: #166534;
}

.risk-level.medium {
  background: #fef3c7;
  color: #92400e;
}

.risk-level.high {
  background: #fee2e2;
  color: #991b1b;
}

.technical-heatmap {
  margin-top: 32px;
}

.technical-heatmap h5 {
  margin: 0 0 16px 0;
  color: #374151;
}

.heatmap-container {
  margin-bottom: 20px;
}

.heatmap-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 8px;
  margin-bottom: 16px;
}

.heatmap-cell {
  padding: 12px;
  border-radius: 6px;
  text-align: center;
  color: white;
  font-weight: 500;
  min-height: 60px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.indicator-name {
  font-size: 12px;
  margin-bottom: 4px;
  opacity: 0.9;
}

.indicator-value {
  font-size: 14px;
  font-weight: 700;
}

.heatmap-legend {
  display: flex;
  gap: 16px;
  justify-content: center;
  flex-wrap: wrap;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #6b7280;
}

.legend-color {
  width: 16px;
  height: 16px;
  border-radius: 3px;
}

.key-insights {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.key-insights h4 {
  margin: 0 0 20px 0;
  color: #1f2937;
}

.insights-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.insight-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  transition: all 0.2s ease;
}

.insight-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.insight-card.high {
  border-left: 4px solid #ef4444;
}

.insight-card.medium {
  border-left: 4px solid #f59e0b;
}

.insight-card.low {
  border-left: 4px solid #10b981;
}

.insight-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.insight-type {
  font-size: 1.2rem;
}

.insight-priority {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.insight-priority.high {
  background: #fee2e2;
  color: #991b1b;
}

.insight-priority.medium {
  background: #fef3c7;
  color: #92400e;
}

.insight-priority.low {
  background: #dcfce7;
  color: #166534;
}

.insight-content h5 {
  margin: 0 0 8px 0;
  color: #1f2937;
}

.insight-content p {
  margin: 0 0 12px 0;
  color: #6b7280;
  font-size: 14px;
  line-height: 1.5;
}

.insight-metrics {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #9ca3af;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .analysis-charts {
    grid-template-columns: 1fr;
  }

  .overview-cards {
    grid-template-columns: 1fr;
  }

  .insights-grid {
    grid-template-columns: 1fr;
  }

  .preset-periods {
    justify-content: center;
  }

  .custom-period {
    flex-direction: column;
    align-items: stretch;
  }

  .date-input-group {
    justify-content: center;
  }
}
</style>
