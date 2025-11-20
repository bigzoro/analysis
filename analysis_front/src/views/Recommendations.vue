<template>
  <!-- 标签页导航 -->
  <section class="panel">
    <div class="tabs">
      <button 
        :class="['tab', { active: activeTab === 'recommendations' }]"
        @click="activeTab = 'recommendations'"
      >
        推荐列表
      </button>
      <button 
        :class="['tab', { active: activeTab === 'backtest' }]"
        @click="activeTab = 'backtest'"
      >
        回测分析
      </button>
      <button 
        :class="['tab', { active: activeTab === 'simulation' }]"
        @click="activeTab = 'simulation'"
      >
        模拟交易
      </button>
    </div>
  </section>

  <!-- 推荐列表标签页 -->
  <div v-if="activeTab === 'recommendations'">
    <section class="panel">
      <div class="row">
        <h2>币种推荐</h2>
        <div class="spacer"></div>
        <label>类型：</label>
        <select v-model="kind" @change="load">
          <option value="spot">现货</option>
          <option value="futures">期货</option>
        </select>
        <button class="primary" @click="load(true)">刷新</button>
      </div>
    </section>

  <section style="margin-top:12px;" class="panel" v-if="loading">
    <div style="text-align:center; padding: 40px;">
      <p>正在生成推荐...</p>
    </div>
  </section>

  <section style="margin-top:12px;" class="panel" v-else-if="data && data.recommendations">
    <div class="meta-bar" style="margin-bottom: 16px;">
      <span class="chip">生成时间：{{ formatTime(data.generated_at) }}</span>
      <span class="chip" v-if="data.cached">使用缓存</span>
    </div>

    <div v-if="data.recommendations.length === 0" style="text-align:center; padding: 40px;">
      <p>暂无推荐数据</p>
    </div>

    <div v-else class="recommendations-list">
      <div 
        v-for="rec in data.recommendations" 
        :key="rec.rank"
        class="recommendation-card"
      >
        <div class="card-header">
          <div class="rank-badge" :class="`rank-${rec.rank}`">
            #{{ rec.rank }}
          </div>
          <div class="symbol-info">
            <h3>{{ rec.base_symbol }}</h3>
            <span class="symbol-pair">{{ rec.symbol }}</span>
          </div>
          <div class="total-score">
            <div class="score-value">{{ rec.total_score.toFixed(1) }}</div>
            <div class="score-label">总分</div>
          </div>
        </div>

        <div class="card-body">
          <div class="score-breakdown">
            <div class="score-item">
              <span class="score-label">市场表现</span>
              <span class="score-value">{{ rec.scores.market.toFixed(1) }}</span>
            </div>
            <div class="score-item">
              <span class="score-label">资金流</span>
              <span class="score-value">{{ rec.scores.flow.toFixed(1) }}</span>
            </div>
            <div class="score-item">
              <span class="score-label">市场热度</span>
              <span class="score-value">{{ rec.scores.heat.toFixed(1) }}</span>
            </div>
            <div class="score-item">
              <span class="score-label">事件</span>
              <span class="score-value">{{ rec.scores.event.toFixed(1) }}</span>
            </div>
            <div class="score-item">
              <span class="score-label">情绪</span>
              <span class="score-value">{{ rec.scores.sentiment.toFixed(1) }}</span>
            </div>
          </div>

          <div class="data-info">
            <div class="data-item" v-if="rec.data.price_change_24h !== null">
              <span class="data-label">24h涨幅：</span>
              <span :class="['data-value', rec.data.price_change_24h >= 0 ? 'positive' : 'negative']">
                {{ rec.data.price_change_24h >= 0 ? '+' : '' }}{{ rec.data.price_change_24h.toFixed(2) }}%
              </span>
            </div>
            <div class="data-item" v-if="rec.data.volume_24h !== null">
              <span class="data-label">24h成交量：</span>
              <span class="data-value">{{ formatVolume(rec.data.volume_24h) }}</span>
            </div>
            <div class="data-item" v-if="rec.data.market_cap_usd !== null">
              <span class="data-label">市值：</span>
              <span class="data-value">{{ formatUSD(rec.data.market_cap_usd) }}</span>
            </div>
            <div class="data-item" v-if="rec.data.net_flow_24h !== null && rec.data.net_flow_24h !== 0">
              <span class="data-label">净流入：</span>
              <span :class="['data-value', rec.data.net_flow_24h >= 0 ? 'positive' : 'negative']">
                {{ rec.data.net_flow_24h >= 0 ? '+' : '' }}{{ formatUSD(rec.data.net_flow_24h) }}
              </span>
            </div>
          </div>

          <!-- 风险评级 -->
          <div class="risk-section" v-if="rec.risk">
            <div class="risk-header">
              <h4>风险评级</h4>
              <span 
                class="risk-badge" 
                :class="`risk-${rec.risk.risk_level || 'medium'}`"
              >
                {{ getRiskLevelText(rec.risk.risk_level) }}
              </span>
            </div>
            <div class="risk-metrics">
              <div class="risk-item">
                <span class="risk-label">综合风险：</span>
                <span class="risk-value" :class="getRiskClass(rec.risk.overall_risk)">
                  {{ rec.risk.overall_risk?.toFixed(1) || 0 }}
                </span>
              </div>
              <div class="risk-breakdown">
                <div class="risk-metric">
                  <span class="metric-label">波动率</span>
                  <span class="metric-value">{{ rec.risk.volatility_risk?.toFixed(0) || 0 }}</span>
                </div>
                <div class="risk-metric">
                  <span class="metric-label">流动性</span>
                  <span class="metric-value">{{ rec.risk.liquidity_risk?.toFixed(0) || 0 }}</span>
                </div>
                <div class="risk-metric">
                  <span class="metric-label">市场</span>
                  <span class="metric-value">{{ rec.risk.market_risk?.toFixed(0) || 0 }}</span>
                </div>
                <div class="risk-metric">
                  <span class="metric-label">技术</span>
                  <span class="metric-value">{{ rec.risk.technical_risk?.toFixed(0) || 0 }}</span>
                </div>
              </div>
            </div>
            <div class="risk-warnings" v-if="rec.risk.risk_warnings && rec.risk.risk_warnings.length > 0">
              <div class="warning-title">⚠️ 风险提示</div>
              <ul class="warning-list">
                <li v-for="(warning, idx) in rec.risk.risk_warnings" :key="idx">{{ warning }}</li>
              </ul>
            </div>
          </div>

          <div class="reasons" v-if="rec.reasons && rec.reasons.length > 0">
            <div class="reasons-title">推荐理由：</div>
            <ul class="reasons-list">
              <li v-for="(reason, idx) in rec.reasons" :key="idx">{{ reason }}</li>
            </ul>
          </div>

          <!-- 价格预测 -->
          <PricePrediction v-if="rec.prediction" :prediction="rec.prediction" />

          <!-- 技术指标 -->
          <div class="technical-section" v-if="rec.technical">
            <div class="technical-header">
              <h4>技术指标</h4>
            </div>
            <div class="technical-metrics">
              <!-- RSI -->
              <div class="technical-item">
                <span class="technical-label">RSI：</span>
                <span class="technical-value" :class="getRSIClass(rec.technical.rsi)">
                  {{ rec.technical.rsi?.toFixed(2) || '-' }}
                </span>
                <span class="technical-hint" v-if="rec.technical.rsi">
                  <span v-if="rec.technical.rsi > 70">(超买)</span>
                  <span v-else-if="rec.technical.rsi < 30">(超卖)</span>
                  <span v-else>(正常)</span>
                </span>
              </div>
              <!-- MACD -->
              <div class="technical-item">
                <span class="technical-label">MACD：</span>
                <span class="technical-value">{{ rec.technical.macd?.toFixed(4) || '-' }}</span>
              </div>
              <div class="technical-item">
                <span class="technical-label">信号线：</span>
                <span class="technical-value">{{ rec.technical.macd_signal?.toFixed(4) || '-' }}</span>
              </div>
              <!-- 布林带 -->
              <div class="technical-item" v-if="rec.technical.bb_position !== undefined">
                <span class="technical-label">布林带位置：</span>
                <span class="technical-value">{{ (rec.technical.bb_position * 100).toFixed(1) }}%</span>
                <span class="technical-hint">
                  <span v-if="rec.technical.bb_position < 0.2">(接近下轨)</span>
                  <span v-else-if="rec.technical.bb_position > 0.8">(接近上轨)</span>
                  <span v-else>(正常)</span>
                </span>
              </div>
              <!-- KDJ -->
              <div class="technical-item" v-if="rec.technical.k !== undefined">
                <span class="technical-label">KDJ：</span>
                <span class="technical-value">
                  K:{{ rec.technical.k?.toFixed(1) || '-' }} 
                  D:{{ rec.technical.d?.toFixed(1) || '-' }} 
                  J:{{ rec.technical.j?.toFixed(1) || '-' }}
                </span>
              </div>
              <!-- 均线 -->
              <div class="technical-item" v-if="rec.technical.ma5 !== undefined && rec.technical.ma5 > 0">
                <span class="technical-label">均线：</span>
                <span class="technical-value">
                  MA5:{{ rec.technical.ma5?.toFixed(2) || '-' }}
                  MA20:{{ rec.technical.ma20?.toFixed(2) || '-' }}
                </span>
              </div>
              <!-- 成交量 -->
              <div class="technical-item" v-if="rec.technical.volume_ratio !== undefined">
                <span class="technical-label">成交量比率：</span>
                <span class="technical-value" :class="rec.technical.volume_ratio > 1 ? 'positive' : 'negative'">
                  {{ rec.technical.volume_ratio?.toFixed(2) || '-' }}x
                </span>
              </div>
              <!-- 支撑位/阻力位 -->
              <div class="technical-item" v-if="rec.technical.support_level !== undefined && rec.technical.support_level > 0">
                <span class="technical-label">支撑位：</span>
                <span class="technical-value">{{ rec.technical.support_level?.toFixed(2) || '-' }}</span>
                <span class="technical-hint" v-if="rec.technical.distance_to_support !== undefined">
                  (距离{{ rec.technical.distance_to_support?.toFixed(1) }}%)
                </span>
              </div>
              <div class="technical-item" v-if="rec.technical.resistance_level !== undefined && rec.technical.resistance_level > 0">
                <span class="technical-label">阻力位：</span>
                <span class="technical-value">{{ rec.technical.resistance_level?.toFixed(2) || '-' }}</span>
                <span class="technical-hint" v-if="rec.technical.distance_to_resistance !== undefined">
                  (距离{{ rec.technical.distance_to_resistance?.toFixed(1) }}%)
                </span>
              </div>
              <!-- 趋势 -->
              <div class="technical-item">
                <span class="technical-label">趋势：</span>
                <span class="technical-value" :class="getTrendClass(rec.technical.trend)">
                  {{ getTrendText(rec.technical.trend) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
  </div>

  <!-- 回测分析标签页 -->
  <BacktestView v-if="activeTab === 'backtest'" />

  <!-- 模拟交易标签页 -->
  <SimulationView v-if="activeTab === 'simulation'" />
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api/api.js'
import BacktestView from './Backtest.vue'
import SimulationView from './Simulation.vue'
import PricePrediction from '../components/PricePrediction.vue'

const activeTab = ref('recommendations')
const kind = ref('spot')
const limit = ref(5)
const data = ref(null)
const loading = ref(false)

function formatTime(timeStr) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

function formatUSD(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1e9) {
    return (value / 1e9).toFixed(2) + 'B'
  } else if (value >= 1e6) {
    return (value / 1e6).toFixed(2) + 'M'
  } else if (value >= 1e3) {
    return (value / 1e3).toFixed(2) + 'K'
  }
  return value.toFixed(2)
}

function formatVolume(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1e9) {
    return (value / 1e9).toFixed(2) + 'B'
  } else if (value >= 1e6) {
    return (value / 1e6).toFixed(2) + 'M'
  }
  return value.toFixed(2)
}

function getRiskLevelText(level) {
  const map = {
    'low': '低风险',
    'medium': '中风险',
    'high': '高风险'
  }
  return map[level] || '未知'
}

function getRiskClass(risk) {
  if (!risk) return ''
  if (risk < 30) return 'risk-low'
  if (risk < 60) return 'risk-medium'
  return 'risk-high'
}

function getRSIClass(rsi) {
  if (!rsi) return ''
  if (rsi > 70) return 'rsi-overbought'
  if (rsi < 30) return 'rsi-oversold'
  return 'rsi-normal'
}

function getTrendText(trend) {
  const map = {
    'up': '上涨',
    'down': '下跌',
    'sideways': '震荡'
  }
  return map[trend] || '未知'
}

function getTrendClass(trend) {
  if (trend === 'up') return 'trend-up'
  if (trend === 'down') return 'trend-down'
  return 'trend-sideways'
}

async function load(refresh = false) {
  loading.value = true
  try {
    data.value = await api.getCoinRecommendations({ kind: kind.value, limit: 5, refresh })
  } catch (error) {
    console.error('加载推荐失败:', error)
    alert('加载推荐失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.recommendations-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.recommendation-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 20px;
  background: #fff;
  transition: box-shadow 0.2s;
}

.recommendation-card:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.rank-badge {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  font-size: 18px;
  color: #fff;
}

.rank-1 { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.rank-2 { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.rank-3 { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.rank-4 { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }
.rank-5 { background: linear-gradient(135deg, #fa709a 0%, #fee140 100%); }

.symbol-info {
  flex: 1;
}

.symbol-info h3 {
  margin: 0;
  font-size: 24px;
  font-weight: bold;
}

.symbol-pair {
  color: #666;
  font-size: 14px;
}

.total-score {
  text-align: center;
}

.score-value {
  font-size: 32px;
  font-weight: bold;
  color: #667eea;
}

.score-label {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.score-breakdown {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
  gap: 12px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
}

.score-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.score-item .score-label {
  font-size: 12px;
  color: #666;
}

.score-item .score-value {
  font-size: 18px;
  font-weight: bold;
  color: #333;
}

.data-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.data-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 4px;
}

.data-label {
  color: #666;
  font-size: 14px;
}

.data-value {
  font-weight: bold;
  font-size: 14px;
}

.data-value.positive {
  color: #10b981;
}

.data-value.negative {
  color: #ef4444;
}

.reasons {
  margin-top: 8px;
}

.reasons-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #333;
}

.reasons-list {
  margin: 0;
  padding-left: 20px;
  color: #666;
}

.reasons-list li {
  margin-bottom: 4px;
}

/* 风险评级样式 */
.risk-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e0e0e0;
}

.risk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.risk-header h4 {
  margin: 0;
  color: #333;
  font-size: 1em;
}

.risk-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.85em;
  font-weight: bold;
}

.risk-badge.risk-low {
  background-color: #d4edda;
  color: #155724;
}

.risk-badge.risk-medium {
  background-color: #fff3cd;
  color: #856404;
}

.risk-badge.risk-high {
  background-color: #f8d7da;
  color: #721c24;
}

.risk-metrics {
  margin-bottom: 12px;
}

.risk-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding: 8px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.risk-label {
  font-size: 0.9em;
  color: #666;
}

.risk-value {
  font-size: 1.1em;
  font-weight: bold;
}

.risk-value.risk-low {
  color: #28a745;
}

.risk-value.risk-medium {
  color: #ffc107;
}

.risk-value.risk-high {
  color: #dc3545;
}

.risk-breakdown {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin-top: 8px;
}

.risk-metric {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.risk-metric .metric-label {
  font-size: 0.8em;
  color: #999;
  margin-bottom: 4px;
}

.risk-metric .metric-value {
  font-size: 1em;
  font-weight: bold;
  color: #333;
}

.risk-warnings {
  margin-top: 12px;
  padding: 12px;
  background-color: #fff3cd;
  border-left: 3px solid #ffc107;
  border-radius: 4px;
}

.warning-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #856404;
}

.warning-list {
  margin: 0;
  padding-left: 20px;
  list-style-type: disc;
}

.warning-list li {
  margin-bottom: 4px;
  font-size: 0.9em;
  color: #856404;
}

/* 技术指标样式 */
.technical-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e0e0e0;
}

.technical-header {
  margin-bottom: 12px;
}

.technical-header h4 {
  margin: 0;
  color: #333;
  font-size: 1em;
}

.technical-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}

.technical-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.technical-label {
  font-size: 0.9em;
  color: #666;
  font-weight: 500;
}

.technical-value {
  font-size: 1em;
  font-weight: bold;
  color: #333;
}

.technical-hint {
  font-size: 0.8em;
  color: #999;
}

.rsi-overbought {
  color: #dc3545;
}

.rsi-oversold {
  color: #28a745;
}

.rsi-normal {
  color: #333;
}

.trend-up {
  color: #28a745;
}

.trend-down {
  color: #dc3545;
}

.trend-sideways {
  color: #666;
}
</style>

