<template>
  <div class="performance-card">
    <div class="card-header">
      <h3>📊 历史表现分析</h3>
      <div class="performance-period">
        <select v-model="performancePeriod" @change="$emit('periodChange', performancePeriod)">
          <option value="7d">7天</option>
          <option value="30d">30天</option>
          <option value="90d">90天</option>
          <option value="1y">1年</option>
        </select>
      </div>
    </div>
    <div class="card-body">
      <!-- 性能评分概览 -->
      <div class="performance-overview">
        <div class="score-visualization">
          <div class="score-circle">
            <div class="score-ring" :style="{ background: getScoreRingGradient(props.performanceData?.overall_score || 0) }">
              <div class="score-content">
                <div class="score-value">{{ props.performanceData?.overall_score?.toFixed(1) || 'N/A' }}</div>
                <div class="score-label">综合评分</div>
              </div>
            </div>
          </div>
          <div class="score-breakdown">
            <div class="score-factor">
              <span class="factor-label">收益率贡献</span>
              <div class="factor-bar">
                <div class="factor-fill return" :style="{ width: getFactorWidth(props.performanceData?.return_factor || 0) }"></div>
              </div>
              <span class="factor-value">{{ props.performanceData?.return_factor?.toFixed(1) || '0.0' }}</span>
            </div>
            <div class="score-factor">
              <span class="factor-label">风险控制</span>
              <div class="factor-bar">
                <div class="factor-fill risk" :style="{ width: getFactorWidth(props.performanceData?.risk_factor || 0) }"></div>
              </div>
              <span class="factor-value">{{ props.performanceData?.risk_factor?.toFixed(1) || '0.0' }}</span>
            </div>
            <div class="score-factor">
              <span class="factor-label">一致性</span>
              <div class="factor-bar">
                <div class="factor-fill consistency" :style="{ width: getFactorWidth(props.performanceData?.consistency_factor || 0) }"></div>
              </div>
              <span class="factor-value">{{ props.performanceData?.consistency_factor?.toFixed(1) || '0.0' }}</span>
            </div>
            <div class="score-factor">
              <span class="factor-label">时机把握</span>
              <div class="factor-bar">
                <div class="factor-fill timing" :style="{ width: getFactorWidth(props.performanceData?.timing_factor || 0) }"></div>
              </div>
              <span class="factor-value">{{ props.performanceData?.timing_factor?.toFixed(1) || '0.0' }}</span>
            </div>
          </div>
        </div>
        <div class="score-insights">
          <div class="insight">
            <span class="insight-icon">🎯</span>
            <div class="insight-text">
              <strong>{{ getScoreLevel(props.performanceData?.overall_score || 0) }}</strong>
              <br>
              <small>{{ getScoreDescription(props.performanceData?.overall_score || 0) }}</small>
            </div>
          </div>
          <div class="insight">
            <span class="insight-icon">📈</span>
            <div class="insight-text">
              <strong>最佳表现:</strong> {{ getBestFactor() }}
              <br>
              <small>评分贡献最大的因素</small>
            </div>
          </div>
        </div>
      </div>

      <!-- 核心性能指标 -->
      <div class="performance-section">
        <h4>核心指标</h4>
        <div class="performance-grid">
          <div class="performance-item">
            <div class="metric">
              <span class="label">推荐准确率</span>
              <span class="value">{{ props.performanceData?.accuracy?.toFixed(1) || 'N/A' }}%</span>
            </div>
            <div class="change" :class="getChangeClass(props.performanceData?.accuracy_change)">
              {{ props.performanceData?.accuracy_change >= 0 ? '+' : '' }}{{ props.performanceData?.accuracy_change?.toFixed(1) || '0.0' }}%
            </div>
          </div>
          <div class="performance-item">
            <div class="metric">
              <span class="label">平均收益率</span>
              <span class="value">{{ props.performanceData?.avg_return?.toFixed(2) || 'N/A' }}%</span>
            </div>
            <div class="change" :class="getChangeClass(props.performanceData?.avg_return)">
              {{ props.performanceData?.avg_return >= 0 ? '+' : '' }}{{ props.performanceData?.avg_return?.toFixed(2) || '0.00' }}%
            </div>
          </div>
          <div class="performance-item">
            <div class="metric">
              <span class="label">胜率</span>
              <span class="value">{{ props.performanceData?.win_rate?.toFixed(1) || 'N/A' }}%</span>
            </div>
            <div class="change" :class="getChangeClass(props.performanceData?.win_rate - 50)">
              {{ (props.performanceData?.win_rate - 50 >= 0 ? '+' : '') + (props.performanceData?.win_rate - 50)?.toFixed(1) || '0.0' }}%
            </div>
          </div>
          <div class="performance-item">
            <div class="metric">
              <span class="label">最大回撤</span>
              <span class="value">{{ props.performanceData?.max_drawdown?.toFixed(2) || 'N/A' }}%</span>
            </div>
            <div class="change negative">
              -{{ props.performanceData?.max_drawdown?.toFixed(2) || '0.00' }}%
            </div>
          </div>
        </div>
      </div>

      <!-- 扩展性能指标 -->
      <div class="performance-section">
        <h4>详细分析</h4>
        <div class="extended-metrics">
          <div class="metrics-row">
            <div class="metric-group">
              <div class="metric-item">
                <span class="metric-name">平均持仓时间</span>
                <span class="metric-value">{{ props.performanceData?.avg_holding_time || 'N/A' }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-name">总推荐次数</span>
                <span class="metric-value">{{ props.performanceData?.total_recommendations || 'N/A' }}</span>
              </div>
            </div>
            <div class="metric-group">
              <div class="metric-item">
                <span class="metric-name">夏普比率</span>
                <span class="metric-value">{{ props.performanceData?.sharpe_ratio?.toFixed(2) || 'N/A' }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-name">信息比率</span>
                <span class="metric-value">{{ props.performanceData?.information_ratio?.toFixed(2) || 'N/A' }}</span>
              </div>
            </div>
          </div>

          <div class="metrics-row">
            <div class="metric-group">
              <div class="metric-item">
                <span class="metric-name">最佳月度收益</span>
                <span class="metric-value positive">{{ props.performanceData?.best_monthly_return?.toFixed(2) || 'N/A' }}%</span>
              </div>
              <div class="metric-item">
                <span class="metric-name">最差月度收益</span>
                <span class="metric-value negative">{{ props.performanceData?.worst_monthly_return?.toFixed(2) || 'N/A' }}%</span>
              </div>
            </div>
            <div class="metric-group">
              <div class="metric-item">
                <span class="metric-name">月度胜率</span>
                <span class="metric-value">{{ props.performanceData?.monthly_win_rate?.toFixed(1) || 'N/A' }}%</span>
              </div>
              <div class="metric-item">
                <span class="metric-name">收益波动率</span>
                <span class="metric-value">{{ performanceData?.volatility?.toFixed(2) || 'N/A' }}%</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 时间维度分析 -->
      <div class="performance-section">
        <h4>时间维度分析</h4>
        <div class="time-dimension-analysis">
          <div class="time-periods">
            <div v-for="period in props.performanceData?.time_analysis" :key="period.period" class="period-item">
              <div class="period-header">
                <span class="period-name">{{ getPeriodName(period.period) }}</span>
                <span class="period-score" :class="getScoreClass(period.performance_score)">
                  {{ period.performance_score?.toFixed(1) || 'N/A' }}
                </span>
              </div>
              <div class="period-metrics">
                <div class="metric">
                  <span class="label">收益率</span>
                  <span class="value" :class="getChangeClass(period.return)">
                    {{ period.return >= 0 ? '+' : '' }}{{ period.return?.toFixed(2) || '0.00' }}%
                  </span>
                </div>
                <div class="metric">
                  <span class="label">胜率</span>
                  <span class="value">{{ period.win_rate?.toFixed(1) || 'N/A' }}%</span>
                </div>
                <div class="metric">
                  <span class="label">最大回撤</span>
                  <span class="value negative">{{ period.max_drawdown?.toFixed(2) || 'N/A' }}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 历史收益图表 -->
      <div class="performance-chart">
        <div class="chart-header">
          <h4>收益曲线对比</h4>
          <div class="chart-controls">
            <label class="chart-toggle">
              <input type="checkbox" v-model="showBenchmark" @change="$emit('benchmarkToggle', showBenchmark)">
              <span>显示基准对比</span>
            </label>
          </div>
        </div>
        <div class="chart-container">
          <div v-if="performanceLoading" class="chart-loading">
            <div class="loading-spinner"></div>
            <p>正在加载性能数据...</p>
          </div>
          <canvas v-else-if="performanceChartData" ref="performanceChart" class="performance-chart-canvas"></canvas>
          <div v-else class="chart-placeholder">
            <div class="placeholder-icon">📈</div>
            <p>暂无历史数据</p>
            <small>需要更多交易记录来生成收益曲线</small>
          </div>
        </div>
      </div>

      <!-- 性能对比分析 -->
      <div class="performance-comparison">
        <div class="section-header">
          <h4>性能对比分析</h4>
          <div class="comparison-selector">
            <select v-model="comparisonAsset" @change="$emit('comparisonChange', comparisonAsset)">
              <option value="BTC">比特币 (BTC)</option>
              <option value="ETH">以太坊 (ETH)</option>
              <option value="SPY">标普500 (SPY)</option>
              <option value="QQQ">纳指100 (QQQ)</option>
            </select>
          </div>
        </div>
        <div class="comparison-grid">
          <div class="comparison-item">
            <div class="comparison-header">
              <span class="asset-name">{{ symbol }}</span>
              <span class="asset-type">AI推荐策略</span>
            </div>
            <div class="comparison-metrics">
              <div class="metric">
                <span class="label">总收益率</span>
                <span class="value" :class="getChangeClass(props.performanceData?.total_return)">
                  {{ props.performanceData?.total_return >= 0 ? '+' : '' }}{{ props.performanceData?.total_return?.toFixed(2) || '0.00' }}%
                </span>
              </div>
              <div class="metric">
                <span class="label">年化收益率</span>
                <span class="value" :class="getChangeClass(props.performanceData?.annualized_return)">
                  {{ props.performanceData?.annualized_return >= 0 ? '+' : '' }}{{ props.performanceData?.annualized_return?.toFixed(2) || '0.00' }}%
                </span>
              </div>
              <div class="metric">
                <span class="label">夏普比率</span>
                <span class="value">{{ props.performanceData?.sharpe_ratio?.toFixed(2) || 'N/A' }}</span>
              </div>
            </div>
          </div>

          <div class="comparison-vs">
            <span class="vs-text">对比</span>
          </div>

          <div class="comparison-item">
            <div class="comparison-header">
              <span class="asset-name">{{ comparisonAsset }}</span>
              <span class="asset-type">基准资产</span>
            </div>
            <div class="comparison-metrics">
              <div class="metric">
                <span class="label">总收益率</span>
                <span class="value" :class="getChangeClass(comparisonData?.total_return)">
                  {{ comparisonData?.total_return >= 0 ? '+' : '' }}{{ comparisonData?.total_return?.toFixed(2) || '0.00' }}%
                </span>
              </div>
              <div class="metric">
                <span class="label">年化收益率</span>
                <span class="value" :class="getChangeClass(comparisonData?.annualized_return)">
                  {{ comparisonData?.annualized_return >= 0 ? '+' : '' }}{{ comparisonData?.annualized_return?.toFixed(2) || '0.00' }}%
                </span>
              </div>
              <div class="metric">
                <span class="label">波动率</span>
                <span class="value">{{ comparisonData?.volatility?.toFixed(2) || 'N/A' }}%</span>
              </div>
            </div>
          </div>
        </div>

        <div class="comparison-insights">
          <div class="insight-item">
            <span class="insight-icon">📊</span>
            <div class="insight-content">
              <span class="insight-title">超额收益</span>
              <span class="insight-value" :class="getChangeClass(getExcessReturn())">
                {{ getExcessReturn() >= 0 ? '+' : '' }}{{ getExcessReturn()?.toFixed(2) || '0.00' }}%
              </span>
            </div>
          </div>
          <div class="insight-item">
            <span class="insight-icon">🎯</span>
            <div class="insight-content">
              <span class="insight-title">风险调整收益</span>
              <span class="insight-value" :class="getChangeClass(getRiskAdjustedReturn())">
                {{ getRiskAdjustedReturn()?.toFixed(2) || 'N/A' }}
              </span>
            </div>
          </div>
          <div class="insight-item">
            <span class="insight-icon">⚡</span>
            <div class="insight-content">
              <span class="insight-title">表现评级</span>
              <span class="insight-value" :class="getPerformanceRatingClass()">
                {{ getPerformanceRating() }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 历史推荐记录 -->
      <div class="historical-recommendations">
        <div class="section-header">
          <h4>历史推荐记录</h4>
          <div class="filter-controls">
            <select v-model="recommendationFilter" @change="$emit('filterChange', recommendationFilter)">
              <option value="all">全部推荐</option>
              <option value="profitable">盈利推荐</option>
              <option value="loss">亏损推荐</option>
              <option value="recent">最近7天</option>
            </select>
          </div>
        </div>
        <div class="recommendations-list">
          <div v-if="filteredRecommendations?.length === 0" class="no-data">
            <p>暂无历史推荐记录</p>
          </div>
          <div v-else v-for="rec in filteredRecommendations" :key="rec.id" class="recommendation-item" :class="getRecommendationClass(rec)">
            <div class="rec-header">
              <span class="rec-date">{{ formatDate(rec.recommended_at) }}</span>
              <span class="rec-type" :class="'rec-' + rec.recommendation_type.toLowerCase()">
                {{ getRecommendationTypeText(rec.recommendation_type) }}
              </span>
              <span class="rec-result" :class="getResultClass(rec.actual_return)">
                {{ rec.actual_return >= 0 ? '+' : '' }}{{ rec.actual_return?.toFixed(2) || '0.00' }}%
              </span>
            </div>
            <div class="rec-details">
              <div class="rec-price">
                <span class="label">推荐价格:</span>
                <span class="value">${{ rec.recommended_price?.toFixed(2) }}</span>
              </div>
              <div class="rec-price">
                <span class="label">当前价格:</span>
                <span class="value">${{ rec.current_price?.toFixed(2) }}</span>
              </div>
              <div class="rec-duration">
                <span class="label">持有时间:</span>
                <span class="value">{{ rec.holding_duration || '进行中' }}</span>
              </div>
            </div>
            <div class="rec-reason">{{ rec.reason }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits, ref, watch } from 'vue'

const props = defineProps({
  performanceData: {
    type: Object,
    default: () => ({})
  },
  performanceLoading: {
    type: Boolean,
    default: false
  },
  performanceChartData: {
    type: Object,
    default: null
  },
  comparisonData: {
    type: Object,
    default: () => ({})
  },
  filteredRecommendations: {
    type: Array,
    default: () => []
  },
  symbol: {
    type: String,
    required: true
  }
})

const emit = defineEmits([
  'periodChange',
  'benchmarkToggle',
  'comparisonChange',
  'filterChange'
])

const performancePeriod = ref('30d')
const showBenchmark = ref(true)
const comparisonAsset = ref('BTC')
const recommendationFilter = ref('all')

const performanceChart = ref(null)

// 调试信息
console.log('RecommendationPerformance组件初始化')
console.log('Props:', props)
console.log('PerformanceData:', props.performanceData)

// 监听props变化
watch(() => props.performanceData, (newData) => {
  console.log('PerformanceData变化:', newData)
}, { immediate: true, deep: true })

watch(() => props, (newProps) => {
  console.log('Props整体变化:', newProps)
}, { immediate: true, deep: true })

// 工具函数
const getChangeClass = (change) => {
  if (!change) return ''
  return change >= 0 ? 'positive' : 'negative'
}

const getScoreRingGradient = (score) => {
  if (!score) return 'conic-gradient(#e5e7eb 0deg, #e5e7eb 360deg)'

  const percentage = (score / 10) * 100
  const angle = (percentage / 100) * 360

  if (score >= 8.5) {
    return `conic-gradient(#10b981 0deg, #10b981 ${angle}deg, #e5e7eb ${angle}deg, #e5e7eb 360deg)`
  } else if (score >= 7.0) {
    return `conic-gradient(#3b82f6 0deg, #3b82f6 ${angle}deg, #e5e7eb ${angle}deg, #e5e7eb 360deg)`
  } else if (score >= 6.0) {
    return `conic-gradient(#f59e0b 0deg, #f59e0b ${angle}deg, #e5e7eb ${angle}deg, #e5e7eb 360deg)`
  } else {
    return `conic-gradient(#ef4444 0deg, #ef4444 ${angle}deg, #e5e7eb ${angle}deg, #e5e7eb 360deg)`
  }
}

const getFactorWidth = (factor) => {
  return Math.min((factor / 10) * 100, 100) + '%'
}

const getScoreLevel = (score) => {
  if (score >= 8.5) return '优秀'
  if (score >= 7.0) return '良好'
  if (score >= 6.0) return '一般'
  if (score >= 5.0) return '待改善'
  return '需要关注'
}

const getScoreDescription = (score) => {
  if (score >= 8.5) return '表现卓越，各项指标均达到优秀水平'
  if (score >= 7.0) return '表现良好，具有稳定的盈利能力'
  if (score >= 6.0) return '表现一般，需要进一步优化'
  if (score >= 5.0) return '表现不佳，建议调整策略'
  return '表现较差，需要重点关注风险控制'
}

const getBestFactor = () => {
  if (!props.performanceData) return '无数据'

  const factors = {
    '收益率贡献': props.performanceData.return_factor || 0,
    '风险控制': props.performanceData.risk_factor || 0,
    '一致性': props.performanceData.consistency_factor || 0,
    '时机把握': props.performanceData.timing_factor || 0
  }

  const bestFactor = Object.keys(factors).reduce((a, b) =>
    factors[a] > factors[b] ? a : b
  )

  return bestFactor
}

const getPeriodName = (period) => {
  const names = {
    '1w': '1周',
    '1m': '1个月',
    '3m': '3个月',
    '6m': '6个月',
    '1y': '1年'
  }
  return names[period] || period
}

const getScoreClass = (score) => {
  if (!score) return ''
  if (score >= 8.5) return 'excellent'
  if (score >= 7.5) return 'good'
  if (score >= 6.5) return 'fair'
  return 'poor'
}

const getExcessReturn = () => {
  if (!props.performanceData?.total_return || !props.comparisonData?.total_return) return 0
  return props.performanceData.total_return - props.comparisonData.total_return
}

const getRiskAdjustedReturn = () => {
  if (!props.performanceData?.sharpe_ratio || !props.comparisonData?.volatility) return null
  return props.performanceData.sharpe_ratio * (props.comparisonData.volatility / 100)
}

const getPerformanceRating = () => {
  const excessReturn = getExcessReturn()
  const riskAdjusted = getRiskAdjustedReturn()

  if (excessReturn > 10 && riskAdjusted > 1.5) return '优秀'
  if (excessReturn > 5 && riskAdjusted > 1.0) return '良好'
  if (excessReturn > 0 && riskAdjusted > 0.5) return '一般'
  if (excessReturn < -5) return '较差'
  return '待改善'
}

const getPerformanceRatingClass = () => {
  const rating = getPerformanceRating()
  const classes = {
    '优秀': 'excellent',
    '良好': 'good',
    '一般': 'fair',
    '较差': 'poor',
    '待改善': 'neutral'
  }
  return classes[rating] || 'neutral'
}

const getRecommendationTypeText = (type) => {
  const texts = {
    'BUY': '买入',
    'SELL': '卖出',
    'HOLD': '持有'
  }
  return texts[type] || type
}

const getRecommendationClass = (rec) => {
  return rec.actual_return >= 0 ? 'profitable' : 'loss'
}

const getResultClass = (returnValue) => {
  if (!returnValue) return ''
  return returnValue >= 0 ? 'positive' : 'negative'
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
.performance-card {
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

    .performance-period {
      select {
        padding: 8px 12px;
        border: 1px solid #ddd;
        border-radius: 6px;
        background: white;
        font-size: 14px;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .performance-overview {
    background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
    border-radius: 12px;
    padding: 24px;
    margin-bottom: 32px;
    border: 1px solid #e5e7eb;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 24px;

    @media (max-width: 768px) {
      grid-template-columns: 1fr;
      gap: 16px;
    }

    .score-visualization {
      .score-circle {
        display: flex;
        justify-content: center;
        margin-bottom: 20px;

        .score-ring {
          width: 120px;
          height: 120px;
          border-radius: 50%;
          position: relative;
          display: flex;
          align-items: center;
          justify-content: center;

          .score-content {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            text-align: center;
            color: #1a1a1a;

            .score-value {
              font-size: 28px;
              font-weight: 700;
              line-height: 1;
            }

            .score-label {
              font-size: 12px;
              color: #6b7280;
              margin-top: 4px;
            }
          }
        }
      }

      .score-breakdown {
        .score-factor {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-bottom: 12px;

          &:last-child {
            margin-bottom: 0;
          }

          .factor-label {
            font-size: 13px;
            color: #6b7280;
            min-width: 70px;
          }

          .factor-bar {
            flex: 1;
            height: 6px;
            background: #e5e7eb;
            border-radius: 3px;
            overflow: hidden;

            .factor-fill {
              height: 100%;
              border-radius: 3px;
              transition: width 0.3s ease;

              &.return {
                background: linear-gradient(90deg, #10b981, #059669);
              }

              &.risk {
                background: linear-gradient(90deg, #3b82f6, #2563eb);
              }

              &.consistency {
                background: linear-gradient(90deg, #f59e0b, #d97706);
              }

              &.timing {
                background: linear-gradient(90deg, #8b5cf6, #7c3aed);
              }
            }
          }

          .factor-value {
            font-size: 13px;
            font-weight: 600;
            color: #1a1a1a;
            min-width: 30px;
            text-align: right;
          }
        }
      }
    }

    .score-insights {
      .insight {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        margin-bottom: 16px;

        &:last-child {
          margin-bottom: 0;
        }

        .insight-icon {
          font-size: 24px;
        }

        .insight-text {
          strong {
            color: #1a1a1a;
            font-size: 14px;
          }

          small {
            color: #6b7280;
            font-size: 12px;
            line-height: 1.4;
          }
        }
      }
    }
  }

  .performance-section {
    margin-bottom: 32px;

    h4 {
      font-size: 18px;
      font-weight: 600;
      color: #1a1a1a;
      margin-bottom: 16px;
    }
  }

  .performance-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
    margin-bottom: 24px;

    .performance-item {
      padding: 20px;
      background: #f8f9fa;
      border-radius: 8px;
      display: flex;
      justify-content: space-between;
      align-items: center;

      .metric {
        .label {
          font-size: 14px;
          color: #666;
          margin-bottom: 4px;
        }

        .value {
          font-size: 24px;
          font-weight: 700;
          color: #1a1a1a;
        }
      }

      .change {
        font-size: 14px;
        font-weight: 600;
        padding: 4px 8px;
        border-radius: 12px;

        &.positive {
          color: #10b981;
          background: #dcfce7;
        }

        &.negative {
          color: #ef4444;
          background: #fee2e2;
        }
      }
    }
  }

  .extended-metrics {
    .metrics-row {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 20px;
      margin-bottom: 20px;

      .metric-group {
        display: flex;
        flex-direction: column;
        gap: 12px;

        .metric-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 12px 16px;
          background: #f8f9fa;
          border-radius: 6px;

          .metric-name {
            font-size: 14px;
            color: #666;
            font-weight: 500;
          }

          .metric-value {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;

            &.positive {
              color: #10b981;
            }

            &.negative {
              color: #ef4444;
            }
          }
        }
      }
    }
  }

  .time-dimension-analysis {
    .time-periods {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;

      .period-item {
        background: #f8f9fa;
        border-radius: 8px;
        padding: 16px;
        border: 1px solid #e5e7eb;

        .period-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;

          .period-name {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;
          }

          .period-score {
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 14px;
            font-weight: 600;

            &.excellent {
              background: #dcfce7;
              color: #166534;
            }

            &.good {
              background: #dbeafe;
              color: #1e40af;
            }

            &.fair {
              background: #fef3c7;
              color: #92400e;
            }

            &.poor {
              background: #fee2e2;
              color: #991b1b;
            }
          }
        }

        .period-metrics {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 12px;

          .metric {
            text-align: center;

            .label {
              font-size: 11px;
              color: #9ca3af;
              margin-bottom: 4px;
              display: block;
            }

            .value {
              font-size: 14px;
              font-weight: 600;
              color: #1a1a1a;

              &.positive {
                color: #10b981;
              }

              &.negative {
                color: #ef4444;
              }
            }
          }
        }
      }
    }
  }

  .performance-chart {
    margin-bottom: 32px;

    .chart-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      h4 {
        font-size: 18px;
        font-weight: 600;
        color: #1a1a1a;
        margin: 0;
      }

      .chart-controls {
        .chart-toggle {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 14px;
          color: #666;

          input[type="checkbox"] {
            width: 16px;
            height: 16px;
          }
        }
      }
    }

    .chart-container {
      height: 300px;
      background: #f8f9fa;
      border-radius: 8px;
      position: relative;

      .chart-loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;

        .loading-spinner {
          width: 32px;
          height: 32px;
          border: 3px solid #f3f3f3;
          border-top: 3px solid #667eea;
          border-radius: 50%;
          animation: spin 1s linear infinite;
          margin-bottom: 12px;
        }

        p {
          color: #666;
          margin: 0;
        }
      }

      .performance-chart-canvas {
        width: 100%;
        height: 100%;
      }

      .chart-placeholder {
        display: flex;
        align-items: center;
        justify-content: center;
        flex-direction: column;
        height: 100%;

        .placeholder-icon {
          font-size: 48px;
          margin-bottom: 16px;
          opacity: 0.5;
        }

        p {
          color: #666;
          margin: 4px 0;
          font-weight: 500;
        }

        small {
          color: #9ca3af;
          font-size: 12px;
        }
      }
    }
  }

  .performance-comparison {
    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;

      h4 {
        font-size: 18px;
        font-weight: 600;
        color: #1a1a1a;
        margin: 0;
      }

      .comparison-selector {
        select {
          padding: 6px 10px;
          border: 1px solid #ddd;
          border-radius: 4px;
          background: white;
          font-size: 12px;
        }
      }
    }

    .comparison-grid {
      display: grid;
      grid-template-columns: 1fr auto 1fr;
      gap: 16px;
      align-items: center;
      margin-bottom: 24px;

      .comparison-item {
        background: #f8f9fa;
        border-radius: 8px;
        padding: 16px;
        border: 1px solid #e5e7eb;

        .comparison-header {
          margin-bottom: 12px;

          .asset-name {
            font-size: 16px;
            font-weight: 600;
            color: #1a1a1a;
            display: block;
          }

          .asset-type {
            font-size: 12px;
            color: #9ca3af;
          }
        }

        .comparison-metrics {
          .metric {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;

            &:last-child {
              margin-bottom: 0;
            }

            .label {
              font-size: 13px;
              color: #6b7280;
            }

            .value {
              font-size: 14px;
              font-weight: 600;
              color: #1a1a1a;

              &.positive {
                color: #10b981;
              }

              &.negative {
                color: #ef4444;
              }
            }
          }
        }
      }

      .comparison-vs {
        text-align: center;

        .vs-text {
          background: #667eea;
          color: white;
          padding: 8px 16px;
          border-radius: 20px;
          font-size: 12px;
          font-weight: 600;
        }
      }
    }

    .comparison-insights {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 16px;

      .insight-item {
        display: flex;
        align-items: center;
        gap: 12px;
        background: #f8f9fa;
        padding: 12px 16px;
        border-radius: 8px;
        border: 1px solid #e5e7eb;

        .insight-icon {
          font-size: 20px;
        }

        .insight-content {
          flex: 1;

          .insight-title {
            font-size: 12px;
            color: #9ca3af;
            display: block;
            margin-bottom: 2px;
          }

          .insight-value {
            font-size: 14px;
            font-weight: 600;
            color: #1a1a1a;

            &.positive {
              color: #10b981;
            }

            &.negative {
              color: #ef4444;
            }

            &.excellent {
              color: #166534;
              background: #dcfce7;
              padding: 2px 6px;
              border-radius: 8px;
            }

            &.good {
              color: #1e40af;
              background: #dbeafe;
              padding: 2px 6px;
              border-radius: 8px;
            }

            &.fair {
              color: #92400e;
              background: #fef3c7;
              padding: 2px 6px;
              border-radius: 8px;
            }

            &.poor {
              color: #991b1b;
              background: #fee2e2;
              padding: 2px 6px;
              border-radius: 8px;
            }

            &.neutral {
              color: #6b7280;
              background: #f3f4f6;
              padding: 2px 6px;
              border-radius: 8px;
            }
          }
        }
      }
    }
  }

  .historical-recommendations {
    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;

      h4 {
        font-size: 18px;
        font-weight: 600;
        color: #1a1a1a;
        margin: 0;
      }

      .filter-controls {
        select {
          padding: 6px 10px;
          border: 1px solid #ddd;
          border-radius: 4px;
          background: white;
          font-size: 12px;
        }
      }
    }

    .recommendations-list {
      max-height: 400px;
      overflow-y: auto;

      .no-data {
        text-align: center;
        padding: 40px;
        color: #9ca3af;

        p {
          margin: 0;
          font-size: 14px;
        }
      }

      .recommendation-item {
        background: #f8f9fa;
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 12px;
        border-left: 4px solid #e5e7eb;
        transition: all 0.3s ease;

        &.profitable {
          border-left-color: #10b981;
          background: #f0fdf4;
        }

        &.loss {
          border-left-color: #ef4444;
          background: #fef2f2;
        }

        .rec-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;

          .rec-date {
            font-size: 12px;
            color: #9ca3af;
          }

          .rec-type {
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 11px;
            font-weight: 500;
            text-transform: uppercase;

            &.rec-buy {
              background: #dcfce7;
              color: #166534;
            }

            &.rec-sell {
              background: #fee2e2;
              color: #991b1b;
            }

            &.rec-hold {
              background: #fef3c7;
              color: #92400e;
            }
          }

          .rec-result {
            font-size: 16px;
            font-weight: 600;

            &.positive {
              color: #10b981;
            }

            &.negative {
              color: #ef4444;
            }
          }
        }

        .rec-details {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
          gap: 12px;
          margin-bottom: 8px;

          .rec-price, .rec-duration {
            .label {
              font-size: 11px;
              color: #9ca3af;
              margin-bottom: 2px;
              display: block;
            }

            .value {
              font-size: 13px;
              color: #1a1a1a;
              font-weight: 500;
            }
          }
        }

        .rec-reason {
          font-size: 13px;
          color: #6b7280;
          line-height: 1.4;
        }
      }
    }
  }
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>
