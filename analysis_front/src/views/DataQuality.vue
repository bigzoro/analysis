<template>
  <section class="panel">
    <div class="row">
      <h2>数据质量监控</h2>
      <div class="spacer"></div>
      <button @click="refreshData" :disabled="loading">
        {{ loading ? '刷新中...' : '刷新数据' }}
      </button>
    </div>

    <!-- 数据源状态 -->
    <section class="panel" v-if="sourcesData">
      <div class="row">
        <h3>数据源状态</h3>
        <div class="spacer"></div>
        <span class="chip">活跃: {{ activeSources }} / {{ totalSources }}</span>
      </div>

      <div class="sources-grid">
        <div
          v-for="source in sourcesData.sources"
          :key="source.name"
          class="source-card"
          :class="{ 'source-active': source.available, 'source-inactive': !source.available }"
        >
          <div class="source-header">
            <h4>{{ getSourceDisplayName(source.name) }}</h4>
            <span class="status-badge" :class="source.available ? 'status-up' : 'status-down'">
              {{ source.available ? '正常' : '异常' }}
            </span>
          </div>
          <div class="source-info">
            <div class="info-item">
              <span class="label">类型:</span>
              <span class="value">{{ getSourceType(source.name) }}</span>
            </div>
            <div class="info-item">
              <span class="label">费用:</span>
              <span class="value">免费</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 缓存统计 -->
    <section class="panel" v-if="sourcesData && sourcesData.cache">
      <div class="row">
        <h3>缓存状态</h3>
        <div class="spacer"></div>
        <span class="chip" :class="sourcesData.cache.is_expired ? 'error' : 'success'">
          {{ sourcesData.cache.is_expired ? '已过期' : '正常' }}
        </span>
      </div>

      <div class="cache-stats">
        <div class="stat-item">
          <span class="stat-label">缓存币种数:</span>
          <span class="stat-value">{{ sourcesData.cache.symbol_count }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">最后更新:</span>
          <span class="stat-value">{{ formatTime(sourcesData.cache.last_update) }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">距今时间:</span>
          <span class="stat-value">{{ sourcesData.cache.time_since_update }}</span>
        </div>
      </div>
    </section>

    <!-- 质量指标 -->
    <section class="panel" v-if="qualityReport && qualityReport.quality_metrics">
      <div class="row">
        <h3>质量指标</h3>
      </div>

      <div class="quality-metrics">
        <div class="metric-card">
          <div class="metric-header">
            <h4>缓存命中率</h4>
            <div class="metric-value large">
              {{ (qualityReport.quality_metrics.cache_hit_ratio * 100).toFixed(1) }}%
            </div>
          </div>
          <div class="metric-bar">
            <div
              class="metric-fill"
              :style="{ width: (qualityReport.quality_metrics.cache_hit_ratio * 100) + '%' }"
            ></div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <h4>数据新鲜度</h4>
            <div class="metric-value" :class="qualityReport.quality_metrics.data_freshness === 'fresh' ? 'success' : 'error'">
              {{ qualityReport.quality_metrics.data_freshness === 'fresh' ? '新鲜' : '过期' }}
            </div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <h4>数据源多样性</h4>
            <div class="metric-value large">
              {{ (qualityReport.quality_metrics.source_diversity * 100).toFixed(1) }}%
            </div>
          </div>
          <div class="metric-bar">
            <div
              class="metric-fill"
              :style="{ width: (qualityReport.quality_metrics.source_diversity * 100) + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </section>

    <!-- 多源数据测试 -->
    <section class="panel">
      <div class="row">
        <h3>多源数据测试</h3>
        <div class="spacer"></div>
        <div class="test-controls">
          <select v-model="testSymbol" class="test-select">
            <option value="BTC">BTC</option>
            <option value="ETH">ETH</option>
            <option value="BNB">BNB</option>
            <option value="ADA">ADA</option>
            <option value="XRP">XRP</option>
          </select>
          <button @click="testMultiSource" :disabled="testing" class="test-btn">
            {{ testing ? '测试中...' : '测试数据' }}
          </button>
        </div>
      </div>

      <div v-if="testResult" class="test-result">
        <div class="result-header">
          <h4>{{ testSymbol }} 多源数据结果</h4>
          <span class="result-time">{{ formatTime(testResult.timestamp) }}</span>
        </div>

        <div v-if="testResult.data.symbol_data[testSymbol]" class="symbol-data">
          <div class="unified-price">
            <span class="price-label">综合价格:</span>
            <span class="price-value">${{ formatNumber(testResult.data.symbol_data[testSymbol].unified_price) }}</span>
            <span class="confidence" :class="getConfidenceClass(testResult.data.symbol_data[testSymbol].confidence)">
              置信度: {{ (testResult.data.symbol_data[testSymbol].confidence * 100).toFixed(1) }}%
            </span>
          </div>

          <div class="sources-comparison">
            <h5>数据源对比</h5>
            <div class="source-list">
              <div
                v-for="(sourceData, sourceName) in testResult.data.symbol_data[testSymbol].sources"
                :key="sourceName"
                class="source-item"
              >
                <span class="source-name">{{ getSourceDisplayName(sourceName) }}:</span>
                <span class="source-price">${{ formatNumber(sourceData.price) }}</span>
                <span class="source-time">{{ formatTime(sourceData.last_updated) }}</span>
              </div>
            </div>
          </div>

          <div v-if="testResult.data.symbol_data[testSymbol].news.length > 0" class="news-section">
            <h5>相关新闻 ({{ testResult.data.symbol_data[testSymbol].news.length }})</h5>
            <div class="news-list">
              <div
                v-for="(news, index) in testResult.data.symbol_data[testSymbol].news.slice(0, 3)"
                :key="index"
                class="news-item"
              >
                <div class="news-title">{{ news.title }}</div>
                <div class="news-meta">
                  <span class="news-source">{{ news.source }}</span>
                  <span class="news-sentiment" :class="getSentimentClass(news.sentiment)">
                    情感: {{ (news.sentiment * 100).toFixed(0) }}%
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="no-data">
          <p>未找到 {{ testSymbol }} 的数据</p>
        </div>
      </div>
    </section>
  </section>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api/api.js'

// 响应式数据
const loading = ref(false)
const testing = ref(false)
const sourcesData = ref(null)
const qualityReport = ref(null)
const testSymbol = ref('BTC')
const testResult = ref(null)

// 获取数据源显示名称
function getSourceDisplayName(name) {
  const names = {
    'coingecko': 'CoinGecko',
    'newsapi': 'NewsAPI',
    'reddit': 'Reddit',
    'lunarcrush': 'LunarCrush',
    'binance': 'Binance'
  }
  return names[name] || name
}

// 获取数据源类型
function getSourceType(name) {
  const types = {
    'coingecko': '市场数据',
    'newsapi': '新闻数据',
    'reddit': '社交数据',
    'lunarcrush': '社交数据',
    'binance': '市场数据'
  }
  return types[name] || '未知'
}

// 获取置信度样式类
function getConfidenceClass(confidence) {
  if (confidence >= 0.8) return 'high'
  if (confidence >= 0.6) return 'medium'
  return 'low'
}

// 获取情感样式类
function getSentimentClass(sentiment) {
  if (sentiment > 0.6) return 'positive'
  if (sentiment < 0.4) return 'negative'
  return 'neutral'
}

// 格式化数字
function formatNumber(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else {
    return value.toFixed(4)
  }
}

// 格式化时间
function formatTime(timeStr) {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString('zh-CN')
}

// 计算活跃数据源数量
const activeSources = computed(() => {
  if (!sourcesData.value || !sourcesData.value.sources) return 0
  return sourcesData.value.sources.filter(s => s.available).length
})

// 计算总数据源数量
const totalSources = computed(() => {
  if (!sourcesData.value || !sourcesData.value.sources) return 0
  return sourcesData.value.sources.length
})

// 刷新数据
async function refreshData() {
  loading.value = true
  try {
    // 并行获取数据源信息和质量报告
    const [sourcesRes, qualityRes] = await Promise.allSettled([
      api.getDataSources(),
      api.getDataQualityReport()
    ])

    if (sourcesRes.status === 'fulfilled') {
      sourcesData.value = sourcesRes.value
    }

    if (qualityRes.status === 'fulfilled') {
      qualityReport.value = qualityRes.value
    }
  } catch (error) {
    console.error('刷新数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 测试多源数据
async function testMultiSource() {
  testing.value = true
  try {
    testResult.value = await api.getMultiSourceData(testSymbol.value)
  } catch (error) {
    console.error('测试多源数据失败:', error)
  } finally {
    testing.value = false
  }
}

// 组件挂载时加载数据
onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.sources-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.source-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s ease;
}

.source-active {
  border-color: #28a745;
  background: linear-gradient(135deg, rgba(40, 167, 69, 0.05) 0%, rgba(40, 167, 69, 0.02) 100%);
}

.source-inactive {
  border-color: #dc3545;
  background: linear-gradient(135deg, rgba(220, 53, 69, 0.05) 0%, rgba(220, 53, 69, 0.02) 100%);
}

.source-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.source-header h4 {
  margin: 0;
  color: #333;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.8em;
  font-weight: bold;
}

.status-up {
  background: #d4edda;
  color: #155724;
}

.status-down {
  background: #f8d7da;
  color: #721c24;
}

.source-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  font-size: 0.9em;
}

.info-item .label {
  color: #666;
}

.info-item .value {
  color: #333;
  font-weight: 500;
}

.cache-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.stat-label {
  color: #666;
  font-weight: 500;
}

.stat-value {
  color: #333;
  font-weight: 600;
}

.quality-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-top: 16px;
}

.metric-card {
  border: 1px solid #e9ecef;
  border-radius: 8px;
  padding: 16px;
  background: white;
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.metric-header h4 {
  margin: 0;
  color: #333;
}

.metric-value {
  font-size: 1.5em;
  font-weight: bold;
  color: #333;
}

.metric-value.large {
  font-size: 2em;
}

.metric-value.success {
  color: #28a745;
}

.metric-value.error {
  color: #dc3545;
}

.metric-bar {
  height: 8px;
  background: #e9ecef;
  border-radius: 4px;
  overflow: hidden;
}

.metric-fill {
  height: 100%;
  background: linear-gradient(90deg, #28a745, #20c997);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.test-controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.test-select {
  padding: 8px 12px;
  border: 1px solid #ced4da;
  border-radius: 4px;
  background: white;
}

.test-btn {
  padding: 8px 16px;
  background: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.test-btn:disabled {
  background: #6c757d;
  cursor: not-allowed;
}

.test-result {
  margin-top: 16px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.result-header h4 {
  margin: 0;
  color: #333;
}

.result-time {
  color: #666;
  font-size: 0.9em;
}

.unified-price {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  padding: 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.price-label {
  color: #666;
  font-weight: 500;
}

.price-value {
  font-size: 1.2em;
  font-weight: bold;
  color: #333;
}

.confidence {
  font-size: 0.9em;
  padding: 4px 8px;
  border-radius: 12px;
  font-weight: 500;
}

.confidence.high {
  background: #d4edda;
  color: #155724;
}

.confidence.medium {
  background: #fff3cd;
  color: #856404;
}

.confidence.low {
  background: #f8d7da;
  color: #721c24;
}

.sources-comparison,
.news-section {
  margin-top: 16px;
}

.sources-comparison h5,
.news-section h5 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 1em;
}

.source-list,
.news-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.source-item,
.news-item {
  padding: 8px 12px;
  background: white;
  border-radius: 4px;
  border: 1px solid #e9ecef;
}

.source-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.source-name {
  color: #666;
  font-weight: 500;
}

.source-price {
  color: #333;
  font-weight: bold;
}

.source-time {
  color: #999;
  font-size: 0.8em;
}

.news-title {
  color: #333;
  font-weight: 500;
  margin-bottom: 4px;
  line-height: 1.4;
}

.news-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.9em;
}

.news-source {
  color: #666;
}

.news-sentiment {
  font-weight: 500;
}

.news-sentiment.positive {
  color: #28a745;
}

.news-sentiment.negative {
  color: #dc3545;
}

.news-sentiment.neutral {
  color: #6c757d;
}

.no-data {
  text-align: center;
  color: #666;
  padding: 32px;
}
</style>
