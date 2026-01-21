<template>
  <div class="personalized-dashboard">
    <!-- 用户偏好设置面板 -->
    <section class="panel user-preferences" v-if="showPreferences">
      <div class="row">
        <h3>个性化设置</h3>
        <div class="spacer"></div>
        <button @click="showPreferences = false" class="close-btn">×</button>
      </div>

      <div class="preferences-form">
        <div class="form-section">
          <h4>风险偏好</h4>
          <div class="radio-group">
            <label v-for="option in riskOptions" :key="option.value">
              <input
                type="radio"
                v-model="preferences.riskTolerance"
                :value="option.value"
              />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </div>

        <div class="form-section">
          <h4>投资风格</h4>
          <div class="radio-group">
            <label v-for="option in styleOptions" :key="option.value">
              <input
                type="radio"
                v-model="preferences.investmentStyle"
                :value="option.value"
              />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </div>

        <div class="form-section">
          <h4>时间视野</h4>
          <div class="radio-group">
            <label v-for="option in horizonOptions" :key="option.value">
              <input
                type="radio"
                v-model="preferences.timeHorizon"
                :value="option.value"
              />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </div>

        <div class="form-section">
          <h4>偏好因子权重</h4>
          <div class="factor-weights">
            <div
              v-for="factor in factorOptions"
              :key="factor.key"
              class="factor-item"
            >
              <label>{{ factor.label }}</label>
              <input
                type="range"
                min="0"
                max="100"
                step="5"
                v-model="preferences.factorWeights[factor.key]"
                @input="updateFactorWeight(factor.key, $event)"
              />
              <span class="weight-value">{{ preferences.factorWeights[factor.key] }}%</span>
            </div>
          </div>
        </div>

        <div class="form-actions">
          <button @click="resetPreferences" class="secondary">重置</button>
          <button @click="savePreferences" class="primary">保存设置</button>
        </div>
      </div>
    </section>

    <!-- 个性化仪表盘 -->
    <div class="dashboard-content">
      <!-- 顶部工具栏 -->
      <section class="panel toolbar">
        <div class="row">
          <h2>智能投资仪表盘</h2>
          <div class="spacer"></div>
          <div class="toolbar-actions">
            <button @click="showPreferences = true" class="secondary">
              <span class="icon">⚙️</span>个性化设置
            </button>
            <button @click="toggleLayout" class="secondary">
              <span class="icon">{{ isCompactLayout ? '📱' : '🖥️' }}</span>
              {{ isCompactLayout ? '紧凑视图' : '宽屏视图' }}
            </button>
          </div>
        </div>
      </section>

      <!-- 实时推荐流 -->
      <section class="panel recommendations-stream">
        <div class="row">
          <h3>实时推荐</h3>
          <div class="spacer"></div>
          <div class="stream-controls">
            <select v-model="recommendationFilter" @change="updateRecommendations">
              <option value="all">全部</option>
              <option value="high_confidence">高置信度</option>
              <option value="trending">热门</option>
              <option value="personalized">个性化</option>
            </select>
          </div>
        </div>

        <div class="recommendations-grid" :class="{ 'compact': isCompactLayout }">
          <div
            v-for="rec in filteredRecommendations"
            :key="rec.id"
            class="recommendation-card"
            :class="{ 'high-confidence': rec.confidence > 0.8 }"
            @click="handleRecommendationClick(rec)"
          >
            <div class="card-header">
              <div class="coin-info">
                <span class="symbol">{{ rec.symbol }}</span>
                <span class="name">{{ rec.base_symbol }}</span>
              </div>
              <div class="confidence-badge" :class="getConfidenceClass(rec.confidence)">
                {{ (rec.confidence * 100).toFixed(0) }}%
              </div>
            </div>

            <div class="card-content">
              <div class="score-display">
                <div class="score-value">{{ rec.total_score.toFixed(1) }}</div>
                <div class="score-label">综合评分</div>
              </div>

              <div class="factors-preview">
                <div
                  v-for="factor in getTopFactors(rec)"
                  :key="factor.key"
                  class="factor-chip"
                >
                  {{ factor.label }}: {{ factor.score.toFixed(1) }}
                </div>
              </div>
            </div>

            <div class="card-actions">
              <button @click.stop="addToWatchlist(rec)" class="action-btn watch">
                <span class="icon">⭐</span>
                关注
              </button>
              <button @click.stop="viewDetails(rec)" class="action-btn details">
                <span class="icon">📊</span>
                详情
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- 市场概览 -->
      <div class="dashboard-grid">
        <!-- 市场情绪指标 -->
        <section class="panel market-sentiment">
          <h3>市场情绪</h3>
          <div class="sentiment-metrics">
            <div class="metric-item">
              <div class="metric-label">整体情绪</div>
              <div class="sentiment-gauge">
                <div
                  class="gauge-fill"
                  :style="{ width: (marketSentiment.overall * 100) + '%' }"
                  :class="getSentimentClass(marketSentiment.overall)"
                ></div>
              </div>
              <div class="metric-value">{{ (marketSentiment.overall * 100).toFixed(1) }}%</div>
            </div>

            <div class="sentiment-details">
              <div class="detail-item">
                <span>Twitter情绪:</span>
                <span :class="getSentimentClass(marketSentiment.twitter)">
                  {{ (marketSentiment.twitter * 100).toFixed(1) }}%
                </span>
              </div>
              <div class="detail-item">
                <span>新闻情绪:</span>
                <span :class="getSentimentClass(marketSentiment.news)">
                  {{ (marketSentiment.news * 100).toFixed(1) }}%
                </span>
              </div>
            </div>
          </div>
        </section>

        <!-- 投资组合概览 -->
        <section class="panel portfolio-overview">
          <h3>投资组合</h3>
          <div class="portfolio-stats">
            <div class="stat-item">
              <div class="stat-label">总价值</div>
              <div class="stat-value">${{ formatNumber(portfolioStats.totalValue) }}</div>
              <div class="stat-change" :class="getChangeClass(portfolioStats.totalChange)">
                {{ formatPercent(portfolioStats.totalChange) }}
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">今日盈亏</div>
              <div class="stat-value">${{ formatNumber(portfolioStats.dailyPnL) }}</div>
              <div class="stat-change" :class="getChangeClass(portfolioStats.dailyPnLPercent)">
                {{ formatPercent(portfolioStats.dailyPnLPercent) }}
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">胜率</div>
              <div class="stat-value">{{ (portfolioStats.winRate * 100).toFixed(1) }}%</div>
            </div>
          </div>
        </section>

        <!-- 智能提醒 -->
        <section class="panel smart-alerts">
          <h3>智能提醒</h3>
          <div class="alerts-list">
            <div
              v-for="alert in activeAlerts"
              :key="alert.id"
              class="alert-item"
              :class="alert.priority"
            >
              <div class="alert-icon">{{ alert.icon }}</div>
              <div class="alert-content">
                <div class="alert-title">{{ alert.title }}</div>
                <div class="alert-message">{{ alert.message }}</div>
                <div class="alert-time">{{ formatTime(alert.timestamp) }}</div>
              </div>
              <button @click="dismissAlert(alert)" class="alert-dismiss">×</button>
            </div>

            <div v-if="activeAlerts.length === 0" class="no-alerts">
              <div class="no-alerts-icon">✅</div>
              <div class="no-alerts-text">暂无重要提醒</div>
            </div>
          </div>
        </section>

        <!-- 学习建议 -->
        <section class="panel learning-suggestions">
          <h3>学习建议</h3>
          <div class="suggestions-list">
            <div
              v-for="suggestion in personalizedSuggestions"
              :key="suggestion.id"
              class="suggestion-item"
              @click="handleSuggestionClick(suggestion)"
            >
              <div class="suggestion-icon">{{ suggestion.icon }}</div>
              <div class="suggestion-content">
                <div class="suggestion-title">{{ suggestion.title }}</div>
                <div class="suggestion-description">{{ suggestion.description }}</div>
                <div class="suggestion-reason">{{ suggestion.reason }}</div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '@/api/api.js'
import behaviorTracker from '@/utils/behaviorTracker.js'

// 响应式数据
const showPreferences = ref(false)
const isCompactLayout = ref(false)
const recommendationFilter = ref('all')

// 用户偏好设置
const preferences = ref({
  riskTolerance: 'medium',
  investmentStyle: 'balanced',
  timeHorizon: 'medium',
  factorWeights: {
    market: 30,
    flow: 25,
    heat: 20,
    event: 15,
    sentiment: 10
  }
})

// 模拟数据（实际应该从API获取）
const recommendations = ref([
  {
    id: 1,
    symbol: 'BTCUSDT',
    base_symbol: 'BTC',
    total_score: 85.5,
    confidence: 0.92,
    market_score: 88,
    flow_score: 82,
    heat_score: 90,
    event_score: 75,
    sentiment_score: 85
  },
  {
    id: 2,
    symbol: 'ETHUSDT',
    base_symbol: 'ETH',
    total_score: 78.2,
    confidence: 0.85,
    market_score: 85,
    flow_score: 75,
    heat_score: 80,
    event_score: 70,
    sentiment_score: 78
  }
])

const marketSentiment = ref({
  overall: 0.65,
  twitter: 0.72,
  news: 0.58
})

const portfolioStats = ref({
  totalValue: 125000,
  totalChange: 0.085,
  dailyPnL: 1250,
  dailyPnLPercent: 0.032,
  winRate: 0.68
})

const activeAlerts = ref([
  {
    id: 1,
    icon: '🚀',
    title: 'BTC突破新高',
    message: '比特币价格突破历史新高，建议关注',
    priority: 'high',
    timestamp: new Date()
  },
  {
    id: 2,
    icon: '📈',
    title: 'ETH资金流入增加',
    message: '以太坊24h资金流入量显著增加',
    priority: 'medium',
    timestamp: new Date(Date.now() - 3600000)
  }
])

const personalizedSuggestions = ref([
  {
    id: 1,
    icon: '📚',
    title: '学习DeFi基础知识',
    description: '了解去中心化金融的基本概念和运作原理',
    reason: '基于您对DEFI板块的关注度'
  },
  {
    id: 2,
    icon: '🎯',
    title: '优化风险管理',
    description: '学习如何设置止损和仓位管理',
    reason: '您的交易历史显示需要改进风险控制'
  }
])

// 计算属性
const filteredRecommendations = computed(() => {
  let filtered = recommendations.value

  switch (recommendationFilter.value) {
    case 'high_confidence':
      filtered = filtered.filter(r => r.confidence > 0.8)
      break
    case 'trending':
      filtered = filtered.filter(r => r.heat_score > 80)
      break
    case 'personalized':
      // 基于用户偏好过滤
      filtered = filtered.filter(r => matchesUserPreferences(r))
      break
  }

  return filtered
})

// 选项配置
const riskOptions = [
  { value: 'low', label: '保守型 - 优先稳定，接受较低收益' },
  { value: 'medium', label: '平衡型 - 收益与风险均衡' },
  { value: 'high', label: '激进型 - 追求高收益，接受高风险' }
]

const styleOptions = [
  { value: 'conservative', label: '保守风格 - 长期持有，稳健增值' },
  { value: 'balanced', label: '平衡风格 - 适度轮动，均衡配置' },
  { value: 'aggressive', label: '激进风格 - 积极交易，追求超额收益' }
]

const horizonOptions = [
  { value: 'short', label: '短期 - 1-3个月' },
  { value: 'medium', label: '中期 - 3-12个月' },
  { value: 'long', label: '长期 - 1年以上' }
]

const factorOptions = [
  { key: 'market', label: '市场表现' },
  { key: 'flow', label: '资金流向' },
  { key: 'heat', label: '市场热度' },
  { key: 'event', label: '事件影响' },
  { key: 'sentiment', label: '情绪分析' }
]

// 方法
function getConfidenceClass(confidence) {
  if (confidence >= 0.8) return 'high'
  if (confidence >= 0.6) return 'medium'
  return 'low'
}

function getSentimentClass(sentiment) {
  if (sentiment >= 0.6) return 'positive'
  if (sentiment <= 0.4) return 'negative'
  return 'neutral'
}

function getChangeClass(change) {
  return change >= 0 ? 'positive' : 'negative'
}

function getTopFactors(rec) {
  const factors = [
    { key: 'market', label: '市场', score: rec.market_score },
    { key: 'flow', label: '资金', score: rec.flow_score },
    { key: 'heat', label: '热度', score: rec.heat_score },
    { key: 'event', label: '事件', score: rec.event_score },
    { key: 'sentiment', label: '情绪', score: rec.sentiment_score }
  ]

  return factors
    .sort((a, b) => b.score - a.score)
    .slice(0, 3)
}

function matchesUserPreferences(rec) {
  // 简化的偏好匹配逻辑
  const userRiskPreference = preferences.value.riskTolerance
  const recRiskLevel = rec.confidence > 0.8 ? 'high' : rec.confidence > 0.6 ? 'medium' : 'low'

  // 保守型用户偏好低风险，中等风险用户接受中等风险等
  const riskMatch = {
    low: ['low'],
    medium: ['low', 'medium'],
    high: ['low', 'medium', 'high']
  }

  return riskMatch[userRiskPreference].includes(recRiskLevel)
}

function updateFactorWeight(factor, event) {
  preferences.value.factorWeights[factor] = parseInt(event.target.value)
}

function resetPreferences() {
  preferences.value = {
    riskTolerance: 'medium',
    investmentStyle: 'balanced',
    timeHorizon: 'medium',
    factorWeights: {
      market: 30,
      flow: 25,
      heat: 20,
      event: 15,
      sentiment: 10
    }
  }
}

async function savePreferences() {
  try {
    // 这里应该调用API保存用户偏好
    // await api.saveUserPreferences(preferences.value)

    showPreferences.value = false

    // 行为追踪
    behaviorTracker.track('settings_change', 'user_preferences', {
      risk_tolerance: preferences.value.riskTolerance,
      investment_style: preferences.value.investmentStyle,
      time_horizon: preferences.value.timeHorizon
    })

    // 重新加载个性化内容
    updateRecommendations()
  } catch (error) {
    console.error('保存偏好失败:', error)
  }
}

function toggleLayout() {
  isCompactLayout.value = !isCompactLayout.value
  behaviorTracker.track('ui_interaction', 'layout_toggle', {
    layout: isCompactLayout.value ? 'compact' : 'wide'
  })
}

function updateRecommendations() {
  // 这里应该重新获取个性化推荐
  behaviorTracker.track('filter_change', 'recommendations', {
    filter: recommendationFilter.value
  })
}

function handleRecommendationClick(rec) {
  behaviorTracker.trackRecommendationClick(rec, 0)
  // 这里可以导航到详情页面或展开详情
}

function addToWatchlist(rec) {
  behaviorTracker.trackRecommendationSave(rec)
  // 这里应该添加到用户的关注列表
}

function viewDetails(rec) {
  behaviorTracker.trackRecommendationView(rec, 0)
  // 这里可以打开详情模态框或导航到详情页面
}

function dismissAlert(alert) {
  activeAlerts.value = activeAlerts.value.filter(a => a.id !== alert.id)
  behaviorTracker.track('alert_dismiss', alert.title)
}

function handleSuggestionClick(suggestion) {
  behaviorTracker.track('suggestion_click', suggestion.title)
  // 这里可以导航到学习内容或相关页面
}

// 工具函数
function formatNumber(num) {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(2) + 'M'
  } else if (num >= 1000) {
    return (num / 1000).toFixed(2) + 'K'
  }
  return num.toFixed(2)
}

function formatPercent(num) {
  return (num * 100).toFixed(2) + '%'
}

function formatTime(date) {
  const now = new Date()
  const diff = now - new Date(date)
  const minutes = Math.floor(diff / 60000)

  if (minutes < 60) {
    return `${minutes}分钟前`
  } else if (minutes < 1440) {
    return `${Math.floor(minutes / 60)}小时前`
  } else {
    return `${Math.floor(minutes / 1440)}天前`
  }
}

// 生命周期
onMounted(() => {
  // 加载用户偏好设置
  loadUserPreferences()

  // 行为追踪
  behaviorTracker.trackPageView('personalized_dashboard')
})

// 监听偏好变化
watch(preferences, () => {
  updateRecommendations()
}, { deep: true })

async function loadUserPreferences() {
  try {
    // 这里应该从API加载用户偏好
    // const prefs = await api.getUserPreferences()
    // preferences.value = { ...preferences.value, ...prefs }
  } catch (error) {
    console.error('加载用户偏好失败:', error)
  }
}
</script>

<style scoped>
.personalized-dashboard {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
  gap: 20px;
}

/* 偏好设置面板 */
.user-preferences {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  overflow-y: auto;
  z-index: 1000;
  background: white;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
}

.preferences-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-section h4 {
  margin: 0;
  color: #333;
  font-size: 16px;
  font-weight: 600;
}

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-group label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.radio-group label:hover {
  background: #f8f9fa;
}

.radio-group input[type="radio"] {
  margin: 0;
}

.factor-weights {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.factor-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.factor-item label {
  min-width: 80px;
  font-weight: 500;
}

.factor-item input[type="range"] {
  flex: 1;
}

.weight-value {
  min-width: 40px;
  text-align: right;
  font-weight: 600;
  color: #007bff;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #e9ecef;
}

/* 仪表盘布局 */
.dashboard-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.toolbar {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.toolbar h2 {
  margin: 0;
  color: white;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

.toolbar-actions button {
  color: white;
  border-color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.1);
}

.toolbar-actions button:hover {
  background: rgba(255, 255, 255, 0.2);
}

/* 推荐流 */
.recommendations-stream {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
}

.stream-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stream-controls select {
  padding: 6px 12px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.recommendations-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.recommendations-grid.compact {
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
}

.recommendation-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s, box-shadow 0.2s;
  cursor: pointer;
}

.recommendation-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.15);
}

.recommendation-card.high-confidence {
  border: 2px solid #28a745;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.coin-info {
  display: flex;
  flex-direction: column;
}

.coin-info .symbol {
  font-size: 18px;
  font-weight: bold;
  color: #333;
}

.coin-info .name {
  font-size: 14px;
  color: #666;
}

.confidence-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: bold;
}

.confidence-badge.high {
  background: #d4edda;
  color: #155724;
}

.confidence-badge.medium {
  background: #fff3cd;
  color: #856404;
}

.confidence-badge.low {
  background: #f8d7da;
  color: #721c24;
}

.card-content {
  margin-bottom: 16px;
}

.score-display {
  text-align: center;
  margin-bottom: 12px;
}

.score-value {
  font-size: 32px;
  font-weight: bold;
  color: #007bff;
}

.score-label {
  font-size: 14px;
  color: #666;
}

.factors-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.factor-chip {
  background: #e9ecef;
  color: #495057;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.card-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  flex: 1;
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: background-color 0.2s;
}

.action-btn.watch {
  background: #ffc107;
  color: #212529;
}

.action-btn.watch:hover {
  background: #e0a800;
}

.action-btn.details {
  background: #007bff;
  color: white;
}

.action-btn.details:hover {
  background: #0056b3;
}

/* 仪表盘网格 */
.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 20px;
}

/* 市场情绪 */
.sentiment-metrics {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.metric-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric-label {
  font-weight: 500;
  color: #333;
}

.sentiment-gauge {
  height: 8px;
  background: #e9ecef;
  border-radius: 4px;
  overflow: hidden;
}

.gauge-fill {
  height: 100%;
  transition: width 0.3s ease;
}

.gauge-fill.positive {
  background: linear-gradient(90deg, #28a745, #20c997);
}

.gauge-fill.neutral {
  background: linear-gradient(90deg, #ffc107, #fd7e14);
}

.gauge-fill.negative {
  background: linear-gradient(90deg, #dc3545, #fd7e14);
}

.metric-value {
  font-weight: bold;
  text-align: right;
}

.sentiment-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
}

/* 投资组合 */
.portfolio-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.stat-item {
  text-align: center;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
}

.stat-label {
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 18px;
  font-weight: bold;
  color: #333;
  margin-bottom: 4px;
}

.stat-change {
  font-size: 14px;
  font-weight: 600;
}

.stat-change.positive {
  color: #28a745;
}

.stat-change.negative {
  color: #dc3545;
}

/* 智能提醒 */
.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.alert-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  border-left: 4px solid;
}

.alert-item.high {
  background: #f8d7da;
  border-left-color: #dc3545;
}

.alert-item.medium {
  background: #fff3cd;
  border-left-color: #ffc107;
}

.alert-item.low {
  background: #d1ecf1;
  border-left-color: #17a2b8;
}

.alert-icon {
  font-size: 20px;
}

.alert-content {
  flex: 1;
}

.alert-title {
  font-weight: 600;
  color: #333;
  margin-bottom: 2px;
}

.alert-message {
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
}

.alert-time {
  font-size: 12px;
  color: #999;
}

.alert-dismiss {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: #666;
  padding: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.no-alerts {
  text-align: center;
  padding: 40px 20px;
  color: #666;
}

.no-alerts-icon {
  font-size: 48px;
  margin-bottom: 12px;
}

.no-alerts-text {
  font-size: 16px;
}

/* 学习建议 */
.suggestions-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.suggestion-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.suggestion-item:hover {
  background: #e9ecef;
}

.suggestion-icon {
  font-size: 24px;
}

.suggestion-content {
  flex: 1;
}

.suggestion-title {
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.suggestion-description {
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
}

.suggestion-reason {
  font-size: 12px;
  color: #007bff;
  font-style: italic;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .personalized-dashboard {
    padding: 10px;
  }

  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .recommendations-grid {
    grid-template-columns: 1fr;
  }

  .portfolio-stats {
    grid-template-columns: repeat(2, 1fr);
  }

  .user-preferences {
    width: 95%;
    max-height: 90vh;
  }
}

@media (max-width: 480px) {
  .portfolio-stats {
    grid-template-columns: 1fr;
  }

  .factor-weights {
    gap: 16px;
  }

  .factor-item {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
}
</style>
