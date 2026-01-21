<template>
  <div class="recommendation-stats">
    <div class="stats-header">
      <h3>📊 推荐统计</h3>
      <div class="time-range">
        <select v-model="timeRange" @change="updateStats">
          <option value="1h">最近1小时</option>
          <option value="24h">最近24小时</option>
          <option value="7d">最近7天</option>
          <option value="30d">最近30天</option>
        </select>
      </div>
    </div>

    <div class="stats-grid">
      <!-- 评分分布 -->
      <div class="stat-card">
        <div class="card-header">
          <div class="card-icon">📈</div>
          <div class="card-info">
            <h4>评分分布</h4>
            <p>推荐币种评分区间</p>
          </div>
        </div>
        <div class="distribution-chart">
          <div class="distribution-bar">
            <div
              v-for="(range, index) in scoreRanges"
              :key="index"
              class="range-segment"
              :style="{ width: range.percentage + '%', backgroundColor: range.color }"
            >
              <span class="range-label">{{ range.label }}</span>
              <span class="range-count">{{ range.count }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险分布 -->
      <div class="stat-card">
        <div class="card-header">
          <div class="card-icon">⚠️</div>
          <div class="card-info">
            <h4>风险分布</h4>
            <p>各风险等级占比</p>
          </div>
        </div>
        <div class="risk-distribution">
          <div
            v-for="risk in riskStats"
            :key="risk.level"
            class="risk-item"
          >
            <div class="risk-info">
              <span class="risk-icon">{{ getRiskIcon(risk.level) }}</span>
              <span class="risk-label">{{ getRiskText(risk.level) }}</span>
            </div>
            <div class="risk-bar">
              <div
                class="risk-fill"
                :style="{ width: risk.percentage + '%', backgroundColor: risk.color }"
              ></div>
            </div>
            <span class="risk-percentage">{{ risk.percentage }}%</span>
          </div>
        </div>
      </div>

      <!-- AI信心度 -->
      <div class="stat-card">
        <div class="card-header">
          <div class="card-icon">🤖</div>
          <div class="card-info">
            <h4>AI信心度</h4>
            <p>机器学习预测信心分布</p>
          </div>
        </div>
        <div class="confidence-gauge">
          <div class="gauge-container">
            <div class="gauge-background"></div>
            <div
              class="gauge-fill"
              :style="{ '--confidence': avgConfidence }"
            ></div>
            <div class="gauge-center">
              <div class="confidence-value">{{ Math.round(avgConfidence * 100) }}%</div>
              <div class="confidence-label">平均信心</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 市场状态 -->
      <div class="stat-card">
        <div class="card-header">
          <div class="card-icon">🌊</div>
          <div class="card-info">
            <h4>市场状态</h4>
            <p>当前市场环境分析</p>
          </div>
        </div>
        <div class="market-status">
          <div class="market-indicator" :class="marketState">
            <div class="market-icon">{{ getMarketIcon(marketState) }}</div>
            <div class="market-info">
              <div class="market-label">{{ getMarketText(marketState) }}</div>
              <div class="market-trend">{{ marketTrend }}</div>
            </div>
          </div>
          <div class="market-metrics">
            <div class="metric">
              <span class="metric-label">波动率</span>
              <span class="metric-value">{{ volatility.toFixed(2) }}%</span>
            </div>
            <div class="metric">
              <span class="metric-label">上涨比例</span>
              <span class="metric-value">{{ (upRatio * 100).toFixed(1) }}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 实时更新指示器 -->
    <div class="update-indicator" v-if="lastUpdate">
      <span class="update-time">最后更新: {{ formatTime(lastUpdate) }}</span>
      <div class="update-status" :class="{ 'updating': isUpdating }">
        <div class="status-dot"></div>
        <span>{{ isUpdating ? '更新中...' : '实时同步' }}</span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'RecommendationStats',
  props: {
    recommendations: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {
      timeRange: '24h',
      lastUpdate: new Date(),
      isUpdating: false,
      marketState: 'sideways',
      volatility: 0,
      upRatio: 0
    }
  },
  computed: {
    scoreRanges() {
      if (!this.recommendations.length) return []

      const scores = this.recommendations.map(r => r.overall_score)
      const ranges = [
        { min: 0, max: 0.6, label: '0-60', color: '#ef4444', count: 0 },
        { min: 0.6, max: 0.7, label: '60-70', color: '#f59e0b', count: 0 },
        { min: 0.7, max: 0.8, label: '70-80', color: '#3b82f6', count: 0 },
        { min: 0.8, max: 0.9, label: '80-90', color: '#10b981', count: 0 },
        { min: 0.9, max: 1.0, label: '90-100', color: '#8b5cf6', count: 0 }
      ]

      ranges.forEach(range => {
        range.count = scores.filter(score => score >= range.min && score < range.max).length
      })

      const total = scores.length
      ranges.forEach(range => {
        range.percentage = total > 0 ? (range.count / total) * 100 : 0
      })

      return ranges
    },

    riskStats() {
      if (!this.recommendations.length) return []

      const riskLevels = ['low', 'medium', 'high', 'critical']
      const colors = {
        low: '#10b981',
        medium: '#f59e0b',
        high: '#ef4444',
        critical: '#7f1d1d'
      }

      return riskLevels.map(level => {
        const count = this.recommendations.filter(r => r.risk_level === level).length
        const percentage = this.recommendations.length > 0 ? (count / this.recommendations.length) * 100 : 0

        return {
          level,
          count,
          percentage: Math.round(percentage),
          color: colors[level]
        }
      }).filter(stat => stat.count > 0)
    },

    avgConfidence() {
      if (!this.recommendations.length) return 0
      const total = this.recommendations.reduce((sum, r) => sum + r.ml_confidence, 0)
      return total / this.recommendations.length
    },

    marketTrend() {
      const trend = Math.random() > 0.5 ? '上升' : '震荡'
      return `趋势: ${trend}`
    }
  },
  methods: {
    updateStats() {
      this.isUpdating = true
      // 模拟数据更新
      setTimeout(() => {
        this.lastUpdate = new Date()
        this.marketState = ['bull', 'bear', 'sideways'][Math.floor(Math.random() * 3)]
        this.volatility = Math.random() * 5 + 1
        this.upRatio = Math.random() * 0.6 + 0.2
        this.isUpdating = false
      }, 1000)
    },

    getRiskIcon(level) {
      const icons = {
        low: '🟢',
        medium: '🟡',
        high: '🔴',
        critical: '⚠️'
      }
      return icons[level] || '❓'
    },

    getRiskText(level) {
      const texts = {
        low: '低风险',
        medium: '中等风险',
        high: '高风险',
        critical: '极高风险'
      }
      return texts[level] || '未知风险'
    },

    getMarketIcon(state) {
      const icons = {
        bull: '🚀',
        bear: '📉',
        sideways: '➡️'
      }
      return icons[state] || '❓'
    },

    getMarketText(state) {
      const texts = {
        bull: '牛市',
        bear: '熊市',
        sideways: '震荡市'
      }
      return texts[state] || '未知'
    },

    formatTime(date) {
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    }
  },

  mounted() {
    this.updateStats()
  },

  watch: {
    recommendations: {
      handler() {
        this.lastUpdate = new Date()
      },
      deep: true
    }
  }
}
</script>

<style scoped>
.recommendation-stats {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  margin-bottom: 30px;
}

.stats-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.stats-header h3 {
  margin: 0;
  color: #333;
  font-size: 1.25rem;
}

.time-range select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.stat-card {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e9ecef;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.card-icon {
  font-size: 1.5rem;
}

.card-info h4 {
  margin: 0 0 4px 0;
  color: #333;
  font-size: 1rem;
}

.card-info p {
  margin: 0;
  color: #666;
  font-size: 0.85rem;
}

/* 评分分布 */
.distribution-chart {
  margin-top: 16px;
}

.distribution-bar {
  display: flex;
  height: 40px;
  border-radius: 6px;
  overflow: hidden;
  background: #e9ecef;
}

.range-segment {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: white;
  font-size: 0.8rem;
  font-weight: 600;
  min-width: 30px;
  transition: all 0.3s ease;
}

.range-segment:hover {
  transform: translateY(-2px);
}

.range-label {
  font-size: 0.7rem;
  opacity: 0.9;
}

/* 风险分布 */
.risk-distribution {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.risk-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.risk-info {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 80px;
}

.risk-icon {
  font-size: 1rem;
}

.risk-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: #333;
}

.risk-bar {
  flex: 1;
  height: 8px;
  background: #e9ecef;
  border-radius: 4px;
  overflow: hidden;
}

.risk-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.risk-percentage {
  min-width: 40px;
  text-align: right;
  font-size: 0.85rem;
  font-weight: 600;
  color: #666;
}

/* AI信心度 */
.confidence-gauge {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}

.gauge-container {
  position: relative;
  width: 120px;
  height: 120px;
}

.gauge-background {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: conic-gradient(
    from 0deg,
    #e9ecef 0deg,
    #e9ecef 360deg
  );
}

.gauge-fill {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: conic-gradient(
    from 0deg,
    #10b981 0deg,
    #10b981 calc(var(--confidence) * 360deg),
    #e9ecef calc(var(--confidence) * 360deg),
    #e9ecef 360deg
  );
}

.gauge-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
}

.confidence-value {
  font-size: 1.25rem;
  font-weight: bold;
  color: #333;
}

.confidence-label {
  font-size: 0.75rem;
  color: #666;
  margin-top: 2px;
}

/* 市场状态 */
.market-status {
  margin-top: 16px;
}

.market-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  margin-bottom: 12px;
}

.market-indicator.bull {
  background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
  border: 1px solid #10b981;
}

.market-indicator.bear {
  background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
  border: 1px solid #ef4444;
}

.market-indicator.sideways {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border: 1px solid #f59e0b;
}

.market-icon {
  font-size: 1.5rem;
}

.market-info {
  flex: 1;
}

.market-label {
  font-weight: 600;
  color: #333;
}

.market-trend {
  font-size: 0.85rem;
  color: #666;
  margin-top: 2px;
}

.market-metrics {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.metric {
  text-align: center;
  padding: 8px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 6px;
}

.metric-label {
  display: block;
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 4px;
}

.metric-value {
  display: block;
  font-size: 1rem;
  font-weight: 600;
  color: #333;
}

/* 实时更新指示器 */
.update-indicator {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.update-time {
  font-size: 0.85rem;
  color: #666;
}

.update-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  color: #666;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
}

.update-status.updating .status-dot {
  background: #f59e0b;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .stats-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .market-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
