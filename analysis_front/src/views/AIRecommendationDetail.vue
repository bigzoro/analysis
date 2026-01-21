<template>
  <div class="ai-recommendation-detail">
    <div class="content-container">
      <div class="page-header">
        <div class="header-content">
          <button @click="$router.go(-1)" class="back-btn">
            ← 返回
          </button>
          <div class="title-section">
            <h1>📊 {{ recommendationData?.symbol }} 详细分析</h1>
            <p class="subtitle">AI智能推荐详情分析报告</p>
          </div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="shouldShowLoading" class="loading-state">
        <div class="loading-spinner"></div>
        <p>正在加载推荐详情...</p>
      </div>

      <!-- 错误状态 -->
      <div v-else-if="shouldShowError" class="error-state">
        <div class="error-icon">⚠️</div>
        <h3>加载失败</h3>
        <p>{{ error }}</p>
        <button @click="loadRecommendationDetail" class="retry-btn">重试</button>
      </div>

      <!-- 详情内容 -->
      <div v-else-if="shouldShowContent" class="detail-content">
        <!-- 基本信息卡片 -->
        <RecommendationBasicInfo
          :recommendation-data="recommendationData"
          v-if="componentsLoaded.basicInfo"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 技术指标分析卡片 -->
        <RecommendationTechnicalIndicators
          :recommendation-data="recommendationData"
          v-if="componentsLoaded.technicalIndicators"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 实时价格监控卡片 -->
        <RecommendationPriceMonitor
          :current-price="currentPrice"
          :current-price-change="currentPriceChange"
          :price-ranges="priceRanges"
          :price-data="priceData"
          :price-loading="priceLoading"
          @refresh-price="refreshPriceData"
          v-if="componentsLoaded.priceMonitor"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 市场数据卡片 -->
        <RecommendationMarketData
          :recommendation-data="recommendationData"
          v-if="componentsLoaded.marketData"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 历史表现分析卡片 -->
        <RecommendationPerformance
          :performance-data="performanceData"
          :performance-loading="performanceLoading"
          :performance-chart-data="performanceChartData"
          :comparison-data="comparisonData"
          :filtered-recommendations="filteredRecommendations"
          :symbol="symbol"
          @period-change="handlePerformancePeriodChange"
          @benchmark-toggle="handleBenchmarkToggle"
          @comparison-change="handleComparisonChange"
          @filter-change="handleFilterChange"
          v-if="componentsLoaded.performance"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 交易策略卡片 -->
        <RecommendationStrategy
          :recommendation-data="recommendationData"
          v-if="componentsLoaded.strategy"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 执行计划卡片 -->
        <RecommendationExecution
          :recommendation-data="recommendationData"
          v-if="componentsLoaded.execution"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 价格提醒卡片 -->
        <RecommendationAlerts
          :recommendation-data="recommendationData"
          @edit-alert="handleEditAlert"
          @toggle-alert="handleToggleAlert"
          @delete-alert="handleDeleteAlert"
          @add-alert="handleAddAlert"
          @manage-alerts="handleManageAlerts"
          v-if="componentsLoaded.alerts"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>

        <!-- 市场情绪分析卡片 -->
        <RecommendationSentiment
          :sentiment-data="sentimentData"
          v-if="componentsLoaded.sentiment"
        />
        <div v-else class="component-placeholder">
          <div class="loading-spinner small"></div>
          <span>加载中...</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, nextTick, defineAsyncComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/api.js'

// 异步组件定义
const RecommendationBasicInfo = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationBasicInfo.vue')
)

const RecommendationTechnicalIndicators = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationTechnicalIndicators.vue')
)

const RecommendationPriceMonitor = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationPriceMonitor.vue')
)

const RecommendationMarketData = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationMarketData.vue')
)

const RecommendationPerformance = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationPerformance.vue')
)

const RecommendationStrategy = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationStrategy.vue')
)

const RecommendationSentiment = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationSentiment.vue')
)

const RecommendationExecution = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationExecution.vue')
)

const RecommendationAlerts = defineAsyncComponent(() =>
  import('@/components/recommendation/RecommendationAlerts.vue')
)

// 路由和响应式数据
const route = useRoute()
const router = useRouter()

// 基础响应式数据
const loading = ref(true)
const error = ref(null)
const recommendationData = ref(null)
const symbol = ref(route.params.symbol)
const rank = ref(parseInt(route.query.rank) || 1)

// 实时价格数据
const currentPrice = ref(null)
const currentPriceChange = ref(0)
const priceData = ref(null)
const priceRanges = ref(null)
const priceLoading = ref(false)

// 图表数据
const chartTimeframe = ref('1h')
const chartData = ref(null)
const chartLoading = ref(false)
const chartError = ref(null)
const showIndicators = ref(true)

// 历史表现数据
const performancePeriod = ref('30d')
const performanceData = ref(null)
const performanceLoading = ref(false)
const performanceChartData = ref(null)
const showBenchmark = ref(true)
const historicalRecommendations = ref([])
const recommendationFilter = ref('all')
const filteredRecommendations = ref([])

// 性能对比数据
const comparisonAsset = ref('BTC')
const comparisonData = ref(null)

// 市场情绪数据
const sentimentData = ref(null)

// 组件加载状态
const componentsLoaded = ref({
  basicInfo: false,
  technicalIndicators: false,
  priceMonitor: false,
  marketData: false,
  performance: false,
  strategy: false,
  execution: false,
  alerts: false,
  sentiment: false
})

// 计算属性
const formatTime = (date) => {
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 计算显示状态
const shouldShowLoading = computed(() => {
  return loading.value && !recommendationData.value && !error.value
})

const shouldShowError = computed(() => {
  return !!error.value && !recommendationData.value
})

const shouldShowContent = computed(() => {
  return !!recommendationData.value && !loading.value
})

// 工具函数
const formatPrice = (price) => {
  if (price >= 1000) {
    return price.toLocaleString('en-US', { maximumFractionDigits: 2 })
  }
  return price.toFixed(price < 1 ? 6 : 2)
}

const formatVolume = (volume) => {
  if (!volume) return 'N/A'
  if (volume >= 1e9) return (volume / 1e9).toFixed(2) + 'B'
  if (volume >= 1e6) return (volume / 1e6).toFixed(2) + 'M'
  if (volume >= 1e3) return (volume / 1e3).toFixed(2) + 'K'
  return volume.toString()
}

const formatLargeNumber = (num) => {
  if (!num) return 'N/A'
  if (num >= 1e12) return (num / 1e12).toFixed(2) + 'T'
  if (num >= 1e9) return (num / 1e9).toFixed(2) + 'B'
  if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M'
  if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K'
  return num.toString()
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

const getCurrentPrice = (symbol) => {
  // 从推荐数据中获取当前价格，如果还没有加载则使用默认值
  if (recommendationData.value && recommendationData.value.price) {
    return recommendationData.value.price
  }

  // 默认价格映射
  const defaultPrices = {
    'BTC': 45000,
    'ETH': 2800,
    'ADA': 0.45,
    'SOL': 95,
    'DOT': 7.2
  }
  return defaultPrices[symbol] || 1.0
}

// 组件引用
const priceChart = ref(null)
const performanceChart = ref(null)

// 组件加载函数
const loadComponents = async () => {
  // 核心组件 - 立即加载
  componentsLoaded.value.basicInfo = true
  componentsLoaded.value.priceMonitor = true

  // 延迟加载其他组件
  setTimeout(() => {
    componentsLoaded.value.technicalIndicators = true
  }, 100)

  setTimeout(() => {
    componentsLoaded.value.marketData = true
  }, 200)

  setTimeout(() => {
    console.log('加载performance组件，当前performanceData:', performanceData.value)
    componentsLoaded.value.performance = true
  }, 300)

  setTimeout(() => {
    componentsLoaded.value.strategy = true
  }, 400)

  setTimeout(() => {
    componentsLoaded.value.execution = true
  }, 500)

  setTimeout(() => {
    componentsLoaded.value.alerts = true
  }, 600)

  setTimeout(() => {
    componentsLoaded.value.sentiment = true
  }, 700)
}

// 事件处理器
const handlePerformancePeriodChange = (period) => {
  performancePeriod.value = period
  loadPerformanceData()
}

const handleBenchmarkToggle = (show) => {
  showBenchmark.value = show
}

const handleComparisonChange = (asset) => {
  comparisonAsset.value = asset
  loadComparisonData()
}

const handleFilterChange = (filter) => {
  recommendationFilter.value = filter
  filterRecommendations()
}

const filterRecommendations = () => {
  if (!historicalRecommendations.value || historicalRecommendations.value.length === 0) {
    filteredRecommendations.value = []
    return
  }

  const filter = recommendationFilter.value
  if (filter === 'all') {
    filteredRecommendations.value = historicalRecommendations.value
  } else if (filter === 'profitable') {
    filteredRecommendations.value = historicalRecommendations.value.filter(rec => rec.return_value > 0)
  } else if (filter === 'loss') {
    filteredRecommendations.value = historicalRecommendations.value.filter(rec => rec.return_value < 0)
  } else if (filter === 'recent') {
    const sevenDaysAgo = new Date()  
    sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7)
    filteredRecommendations.value = historicalRecommendations.value.filter(rec =>
      new Date(rec.created_at) > sevenDaysAgo
    )
  }
}

// 价格提醒事件处理器
const handleEditAlert = (alert) => {
  // 处理编辑提醒的逻辑
  console.log('编辑提醒:', alert)
  // 这里可以打开编辑提醒的弹窗或导航到编辑页面
}

const handleToggleAlert = (alert) => {
  // 处理切换提醒状态的逻辑
  console.log('切换提醒状态:', alert)
  // 这里可以调用API来切换提醒的激活状态
}

const handleDeleteAlert = (alert) => {
  // 处理删除提醒的逻辑
  console.log('删除提醒:', alert)
  // 这里可以显示确认对话框，然后调用API删除提醒
}

const handleAddAlert = () => {
  // 处理添加新提醒的逻辑
  console.log('添加新提醒')
  // 这里可以打开添加提醒的弹窗或导航到添加页面
}

const handleManageAlerts = () => {
  // 处理管理提醒的逻辑
  console.log('管理提醒')
  // 这里可以导航到提醒管理页面
}

// 数据加载函数
const loadRecommendationDetail = async () => {
  try {
    loading.value = true
    error.value = null

    // 从路由参数获取symbol
    const symbol = route.params.symbol

    if (!symbol) {
      error.value = '未指定交易对'
      loading.value = false
      return
    }

    // 调用API获取推荐详情
    const response = await api.getRecommendationDetail(symbol)

    if (response && response.success && response.recommendation) {
      recommendationData.value = response.recommendation
      loading.value = false
    } else {
      error.value = '未找到推荐数据'
      loading.value = false
    }
  } catch (err) {
    console.error('加载推荐详情失败:', err)
    error.value = err.message || '加载推荐详情失败，请检查网络连接或稍后重试'
    loading.value = false
  }
}

const loadPriceData = async () => {
  try {
    if (!recommendationData.value) return

    const symbol = recommendationData.value.symbol
    const response = await api.getCurrentPrice(symbol)

    if (response && response.price !== undefined) {
      currentPrice.value = response.price
      // 这里可以根据需要设置其他价格相关数据
    } else {
      console.warn('未获取到价格数据')
    }
  } catch (err) {
    console.error('加载价格数据失败:', err)
    // 不使用模拟数据，只是记录错误
  }
}

const loadChartData = async () => {
  try {
    if (!recommendationData.value) return

    const symbol = recommendationData.value.symbol
    const response = await api.getKlines(symbol, chartTimeframe.value, 100)

    if (response && Array.isArray(response)) {
      chartData.value = response
      chartLoading.value = false
      // updatePriceChart() // 暂时注释掉未定义的函数
    } else {
      chartError.value = '未获取到图表数据'
      chartLoading.value = false
    }
  } catch (err) {
    console.error('加载图表数据失败:', err)
    chartError.value = '加载图表数据失败'
    chartLoading.value = false
  }
}

const loadPerformanceData = async () => {
  try {
    console.log('开始加载性能数据...')
    if (!recommendationData.value) {
      console.log('推荐数据不存在，跳过性能数据加载')
      return
    }

    performanceLoading.value = true
    const symbol = recommendationData.value.symbol
    console.log('加载性能数据:', symbol, performancePeriod.value)

    const response = await api.getRecommendationPerformance(symbol, performancePeriod.value)
    console.log('性能数据API响应:', response)

    if (response && response.performance) {
      performanceData.value = response.performance
      performanceLoading.value = false
      console.log('性能数据设置成功:', performanceData.value)
      // updatePerformanceChart() // 暂时注释掉未定义的函数
    } else {
      console.log('API响应中没有performance数据')
      performanceData.value = null
      performanceLoading.value = false
    }
  } catch (err) {
    console.error('加载性能数据失败:', err)
    performanceData.value = null
    performanceLoading.value = false
  }
}

const loadSentimentData = async () => {
  try {
    if (!recommendationData.value) return

    const symbol = recommendationData.value.symbol
    const response = await api.getSentimentAnalysis(symbol)

    if (response && response.sentiment) {
      sentimentData.value = response.sentiment
    } else {
      sentimentData.value = null
    }
  } catch (err) {
    console.error('加载情绪数据失败:', err)
    sentimentData.value = null
  }
}

const loadComparisonData = async () => {
  try {
    if (!recommendationData.value) return

    const symbol = recommendationData.value.symbol
    // 暂时没有对比数据API，使用默认数据结构
    comparisonData.value = {
      benchmark_return: 8.5,
      strategy_return: 15.5,
      excess_return: 7.0,
      tracking_error: 2.5,
      information_ratio: 2.8
    }
    // updatePerformanceChart() // 暂时注释掉未定义的函数
  } catch (err) {
    console.error('加载对比数据失败:', err)
    comparisonData.value = null
  }
}

// 图表辅助函数
const generateBenchmarkData = (length) => {
  const data = []
  let value = 100

  for (let i = 0; i < length; i++) {
    value += (Math.random() - 0.5) * 0.5
    data.push(value)
  }

  return data
}

// 生命周期
onMounted(async () => {
  console.log('页面挂载，开始加载数据')

  // 首先加载推荐详情，这是其他数据的基础
  await loadRecommendationDetail()
  console.log('推荐详情加载完成')

  // 然后加载其他依赖于推荐详情的数据
  await Promise.all([
    loadPriceData(),
    loadChartData(),
    loadSentimentData(),
    loadComparisonData()
  ])

  // 单独加载性能数据，确保它最后执行
  await loadPerformanceData()
  console.log('性能数据加载完成，performanceData:', performanceData.value)

  // 确保数据已经设置完成
  await nextTick()
  console.log('nextTick完成后，performanceData:', performanceData.value)

  // 最后加载组件
  loadComponents()
  console.log('所有数据加载完成')
})
</script>

<style scoped lang="scss">
.ai-recommendation-detail {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;

  // 添加内容容器以控制最大宽度和居中
  .content-container {
    max-width: 1200px;
    margin: 0 auto;
    width: 100%;
    padding: 0 20px; // 添加左右边距

    // 在移动端减少边距
    @media (max-width: 768px) {
      padding: 0 15px;
    }

    @media (max-width: 480px) {
      padding: 0 10px;
    }
  }

  .page-header {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-radius: 16px;
    padding: 24px;
    margin-bottom: 24px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);

    .header-content {
      display: flex;
      align-items: center;
      gap: 24px;

      .back-btn {
        background: #f0f2f5;
        border: none;
        border-radius: 8px;
        padding: 12px 16px;
        font-size: 14px;
        font-weight: 500;
        color: #666;
        cursor: pointer;
        transition: all 0.3s ease;

        &:hover {
          background: #e6e8eb;
          color: #333;
        }
      }

      .title-section {
        h1 {
          font-size: 32px;
          font-weight: 700;
          color: #1a1a1a;
          margin: 0 0 8px 0;
        }

        .subtitle {
          font-size: 16px;
          color: #666;
          margin: 0;
        }
      }
    }
  }

  .loading-state, .error-state {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-radius: 16px;
    padding: 48px;
    text-align: center;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  }

  .loading-spinner {
    width: 48px;
    height: 48px;
    border: 4px solid #f3f3f3;
    border-top: 4px solid #667eea;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 16px;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .error-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .retry-btn {
    background: #667eea;
    color: white;
    border: none;
    border-radius: 8px;
    padding: 12px 24px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.3s ease;

    &:hover {
      background: #5a67d8;
    }
  }

  // 组件占位符样式
    background: rgba(255, 255, 255, 0.95);
  .component-placeholder {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-radius: 16px;
    margin-bottom: 24px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
    padding: 48px;
    text-align: center;
    border: 1px solid #e5e7eb;

    .loading-spinner.small {
      width: 24px;
      height: 24px;
      border: 2px solid #f3f3f3;
      border-top: 2px solid #667eea;
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin: 0 auto 12px;
    }

    span {
      color: #6b7280;
      font-size: 14px;
    }
  }

  // 响应式设计
  @media (max-width: 768px) {
    padding: 16px;

    .page-header {
      .header-content {
        flex-direction: column;
        align-items: flex-start;
        gap: 16px;

        .title-section {
          h1 {
            font-size: 24px;
          }
        }
      }
    }

    .price-monitor-card {
      .price-details {
        flex-direction: column;
        gap: 12px;
      }
    }

    .chart-card {
      .chart-controls {
        flex-direction: column;
        align-items: stretch;
      }
    }

    .performance-card {
      .performance-grid {
        grid-template-columns: 1fr;
      }
    }
  }
}
</style>