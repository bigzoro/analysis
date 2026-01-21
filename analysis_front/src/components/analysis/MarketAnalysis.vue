<template>
  <div class="market-analysis">
    <div class="analysis-header">
      <h3>市场环境分析</h3>
      <p class="analysis-description">实时分析当前市场环境，推荐适合的交易策略</p>
      <button class="btn primary refresh-btn" @click="refreshAnalysis" :disabled="loading">
        {{ loading ? '分析中...' : '刷新分析' }}
      </button>
    </div>

    <!-- 市场概览 -->
    <div class="analysis-grid">
      <div class="analysis-card">
        <h4>市场状态</h4>
        <div class="market-status">
          <div class="status-item">
            <span class="status-label">整体波动率:</span>
            <span class="status-value" :class="marketData.volatilityLevel">
              {{ marketData.volatility }}%
            </span>
          </div>
          <div class="status-item">
            <span class="status-label">市场趋势:</span>
            <span class="status-value" :class="marketData.trendDirection">
              {{ marketData.trend }}
            </span>
          </div>
          <div class="status-item">
            <span class="status-label">震荡程度:</span>
            <span class="status-value" :class="marketData.oscillationLevel">
              {{ marketData.oscillation }}%
            </span>
          </div>
        </div>
      </div>

      <div class="analysis-card">
        <div class="strategy-header-section">
          <h4>推荐策略</h4>
          <div class="strategy-controls">
            <button v-if="strategyFilter !== 'all'" @click="toggleExpandedView" class="btn-link">
              {{ showAllStrategies ? '收起' : '展开全部' }}
            </button>
            <select v-model="strategyFilter" class="strategy-filter">
              <option value="top3">前3名</option>
              <option value="high_score">高评分(>7分)</option>
              <option value="low_risk">低风险</option>
              <option value="all">全部策略</option>
            </select>
          </div>
        </div>

        <div class="strategy-recommendations">

          <!-- 策略列表 -->
          <div
            v-for="(strategy, index) in filteredStrategies"
            :key="strategy.type"
            class="strategy-item"
            :class="{
              'strategy-expanded': expandedStrategies.includes(strategy.type),
              'strategy-top': index < 3,
              'strategy-top-item': strategy.type === topStrategy?.type,
              'strategy-exists': strategy.exists,
              'strategy-new': !strategy.exists
            }"
          >
            <!-- 简要信息行 -->
            <div class="strategy-summary" @click="toggleStrategyExpansion(strategy.type)">
              <div class="strategy-main-info">
                <span class="strategy-name">{{ strategy.name }}</span>
                <div class="strategy-quick-metrics">
                  <span class="metric score">{{ strategy.score }}/10</span>
                  <span class="metric confidence">{{ strategy.confidence }}%</span>
                  <span class="metric risk" :class="strategy.risk_level">
                    {{ strategy.risk_level === 'low' ? '低风险' : strategy.risk_level === 'medium' ? '中风险' : '高风险' }}
                  </span>
                </div>
              </div>
              <div class="strategy-status">
                <span class="status-badge" :class="{ 'status-exists': strategy.exists, 'status-new': !strategy.exists }">
                  {{ strategy.exists ? '已配置' : '新策略' }}
                </span>
                <span class="expand-icon">{{ expandedStrategies.includes(strategy.type) ? '▼' : '▶' }}</span>
              </div>
            </div>

            <!-- 展开的详细信息 -->
            <div v-if="expandedStrategies.includes(strategy.type)" class="strategy-details">
              <div class="strategy-reason">{{ strategy.reason }}</div>

              <!-- 完整的性能指标 -->
              <div class="strategy-performance-metrics" v-if="strategy.win_rate">
                <div class="metrics-grid">
                  <div class="metric-row">
                    <div class="metric-item">
                      <span class="metric-label">胜率</span>
                      <span class="metric-value win-rate">{{ (strategy.win_rate * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="metric-item">
                      <span class="metric-label">最大回撤</span>
                      <span class="metric-value drawdown">{{ (strategy.max_drawdown * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="metric-item">
                      <span class="metric-label">夏普比率</span>
                      <span class="metric-value sharpe">{{ strategy.sharpe_ratio.toFixed(2) }}</span>
                    </div>
                  </div>
                  <div class="metric-row">
                    <div class="metric-item">
                      <span class="metric-label">总交易</span>
                      <span class="metric-value trades">{{ strategy.total_trades }}</span>
                    </div>
                    <div class="metric-item">
                      <span class="metric-label">平均利润</span>
                      <span class="metric-value profit" :class="{ 'negative': strategy.avg_profit < 0 }">
                        {{ (strategy.avg_profit * 100).toFixed(2) }}%
                      </span>
                    </div>
                    <div class="metric-item">
                      <span class="metric-label">波动率</span>
                      <span class="metric-value volatility">{{ (strategy.volatility * 100).toFixed(1) }}%</span>
                    </div>
                  </div>
                </div>
              </div>

              <div class="strategy-market-fit">
                <span class="market-tag">{{ strategy.suitable_market }}</span>
              </div>

              <!-- 策略分析洞察 -->
              <div class="strategy-insights" v-if="strategy.win_rate">
                <div class="insight-item">
                  <span class="insight-label">预期年化收益</span>
                  <span class="insight-value" :class="{ 'positive': strategy.avg_profit > 0, 'negative': strategy.avg_profit < 0 }">
                    {{ calculateExpectedReturn(strategy).toFixed(2) }}%
                  </span>
                </div>
                <div class="insight-item">
                  <span class="insight-label">风险收益比</span>
                  <span class="insight-value" :class="{ 'good': calculateRiskRewardRatio(strategy) > 1, 'poor': calculateRiskRewardRatio(strategy) <= 1 }">
                    {{ calculateRiskRewardRatio(strategy).toFixed(2) }}
                  </span>
                </div>
                <div class="insight-item">
                  <span class="insight-label">置信度等级</span>
                  <span class="insight-value confidence-level" :class="getConfidenceLevel(strategy.confidence)">
                    {{ getConfidenceLevelText(strategy.confidence) }}
                  </span>
                </div>
              </div>

              <div class="strategy-actions">
                <button v-if="!strategy.exists" class="btn-secondary small">了解详情</button>
                <button v-if="strategy.exists" class="btn-link small">查看配置</button>
                <button class="btn-primary small">立即使用</button>
              </div>
            </div>
          </div>

          <!-- 策略统计摘要 -->
          <div class="strategy-summary-footer">
            <div class="summary-stats">
              <span>共 {{ recommendedStrategies.length }} 个策略</span>
              <span>{{ highScoreStrategies.length }} 个高评分策略</span>
              <span>{{ existingStrategies.length }} 个已配置</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 详细分析 -->
    <div class="detailed-analysis">
      <div class="analysis-card full-width">
        <h4>市场详细分析</h4>
        <div class="market-details">
          <div class="detail-section">
            <h5>波动率分析</h5>
            <p>{{ marketAnalysis.volatilityAnalysis }}</p>
          </div>
          <div class="detail-section">
            <h5>趋势分析</h5>
            <p>{{ marketAnalysis.trendAnalysis }}</p>
          </div>
          <div class="detail-section">
            <h5>策略适用性</h5>
            <p>{{ marketAnalysis.strategyAnalysis }}</p>
          </div>
        </div>
      </div>

      <div class="analysis-card full-width">
        <h4>投资建议</h4>
        <div class="investment-advice">
          <div class="advice-item" v-for="advice in investmentAdvice" :key="advice.type">
            <div class="advice-header">
              <span class="advice-title">{{ advice.title }}</span>
            </div>
            <p class="advice-content">{{ advice.content }}</p>
          </div>
        </div>
      </div>
    </div>
    </div>

    <!-- 技术指标监控 -->
    <div class="analysis-card full-width">
      <h4>技术指标监控</h4>
      <div class="technical-indicators">
        <div class="indicator-grid">
          <div class="indicator-item">
            <span class="indicator-name">BTC波动率</span>
            <span class="indicator-value">{{ technicalIndicators.btcVolatility }}%</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">市场平均RSI</span>
            <span class="indicator-value">{{ technicalIndicators.avgRSI }}</span>
          </div>
        <div class="indicator-item">
          <span class="indicator-name">强势币种数量</span>
          <span class="indicator-value">{{ technicalIndicators.strongSymbols }}</span>
        </div>
        <div class="indicator-item">
          <span class="indicator-name">弱势币种数量</span>
          <span class="indicator-value">{{ technicalIndicators.weakSymbols }}</span>
        </div>
      </div>

      <!-- 市场宽度指标 -->
      <div class="analysis-card">
        <h4>市场宽度指标</h4>
        <div class="indicator-grid">
          <div class="indicator-item">
            <span class="indicator-name">涨跌比</span>
            <span class="indicator-value">{{ technicalIndicators.advanceDeclineRatio }}</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">大涨币种 (>5%)</span>
            <span class="indicator-value">{{ technicalIndicators.bigGainers }}</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">大跌币种 (<-5%)</span>
            <span class="indicator-value">{{ technicalIndicators.bigLosers }}</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">震荡币种 (-2%~2%)</span>
            <span class="indicator-value">{{ technicalIndicators.neutralSymbols }}</span>
          </div>
        </div>
      </div>

      <!-- 成交量指标 -->
      <div class="analysis-card">
        <h4>成交量指标</h4>
        <div class="indicator-grid">
          <div class="indicator-item">
            <span class="indicator-name">放量上涨币种</span>
            <span class="indicator-value">{{ technicalIndicators.volumeGainers }}</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">缩量下跌币种</span>
            <span class="indicator-value">{{ technicalIndicators.volumeDecliners }}</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">平均成交量变化</span>
            <span class="indicator-value">{{ technicalIndicators.avgVolumeChange }}%</span>
          </div>
        </div>
      </div>

      <!-- 波动率指标 -->
      <div class="analysis-card">
        <h4>波动率指标</h4>
        <div class="indicator-grid">
          <div class="indicator-item">
            <span class="indicator-name">市场平均波动率</span>
            <span class="indicator-value">{{ technicalIndicators.marketVolatility }}%</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">高波动率币种</span>
            <span class="indicator-value">{{ technicalIndicators.highVolatilitySymbols }}</span>
          </div>
          <div class="indicator-item">
            <span class="indicator-name">低波动率币种</span>
            <span class="indicator-value">{{ technicalIndicators.lowVolatilitySymbols }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../../api/api.js'

// 响应式数据
const loading = ref(false)
const marketData = ref({
  volatility: '0.00',
  volatilityLevel: 'low',
  trend: '震荡',
  trendDirection: 'neutral',
  oscillation: '0.00',
  oscillationLevel: 'medium'
})

const marketAnalysis = ref({
  volatilityAnalysis: '正在分析市场波动率...',
  trendAnalysis: '正在分析市场趋势...',
  strategyAnalysis: '正在评估策略适用性...'
})

const recommendedStrategies = ref([
  {
    type: 'grid_trading',
    name: '网格交易策略',
    score: 10,
    confidence: 85,
    recommended: true,
    reason: '市场横盘震荡，网格策略可以在价格区间内多次交易获利'
  },
  {
    type: 'mean_reversion',
    name: '均值回归策略',
    score: 5,
    confidence: 50,
    recommended: false,
    reason: '需要较高震荡环境，当前市场震荡程度较低'
  },
  {
    type: 'traditional',
    name: '传统策略',
    score: 6,
    confidence: 45,
    recommended: false,
    reason: '适合趋势行情，当前市场震荡程度较高'
  }
])

const investmentAdvice = ref([
  {
    type: 'primary',
    title: '主要建议',
    content: '当前市场处于横盘震荡格局，建议优先使用网格交易策略，在价格区间内设置多个买卖点，获得稳定收益。'
  },
  {
    type: 'risk',
    title: '风险提醒',
    content: '市场波动较大，注意控制仓位，避免过度集中投资。建议单次交易不超过总资金的20%。'
  },
  {
    type: 'timing',
    title: '时机建议',
    content: '震荡市场适合日内交易或短期持仓，建议持仓时间控制在5-15个交易日。'
  }
])

const technicalIndicators = ref({
  // 基础指标
  btcVolatility: '0.00',
  avgRSI: '0.0',
  strongSymbols: 0,
  weakSymbols: 0,

  // 市场宽度指标
  advanceDeclineRatio: '0.00',
  bigGainers: 0,
  bigLosers: 0,
  neutralSymbols: 0,

  // 成交量指标
  volumeGainers: 0,
  volumeDecliners: 0,
  avgVolumeChange: '0.00',

  // 波动率指标
  marketVolatility: '0.00',
  highVolatilitySymbols: 0,
  lowVolatilitySymbols: 0
})

// 策略展示优化相关状态
const showAllStrategies = ref(false)
const expandedStrategies = ref([])
const strategyFilter = ref('top3') // 默认只显示前3个

// 计算属性：最佳策略
const topStrategy = computed(() => {
  return recommendedStrategies.value.length > 0 ? recommendedStrategies.value[0] : null
})

// 计算属性：筛选后的策略列表（包含所有符合条件的策略）
const filteredStrategies = computed(() => {
  let strategies = [...recommendedStrategies.value]

  // 根据筛选条件过滤策略
  let filtered = []
  switch (strategyFilter.value) {
    case 'top3':
      filtered = strategies.slice(0, 3)
      break
    case 'high_score':
      filtered = strategies.filter(s => s.score >= 7)
      break
    case 'low_risk':
      filtered = strategies.filter(s => s.risk_level === 'low')
      break
    case 'all':
      filtered = strategies // 全部策略时显示所有策略
      break
    default:
      filtered = showAllStrategies.value ? strategies : strategies.slice(0, 3)
      break
  }

  return filtered
})


// 计算属性：高评分策略数量
const highScoreStrategies = computed(() => {
  return recommendedStrategies.value.filter(s => s.score >= 7)
})

// 计算属性：已配置策略数量
const existingStrategies = computed(() => {
  return recommendedStrategies.value.filter(s => s.exists)
})

// 切换展开视图
const toggleExpandedView = () => {
  showAllStrategies.value = !showAllStrategies.value
}

// 切换单个策略展开状态
const toggleStrategyExpansion = (strategyType) => {
  const index = expandedStrategies.value.indexOf(strategyType)
  if (index > -1) {
    expandedStrategies.value.splice(index, 1)
  } else {
    expandedStrategies.value.push(strategyType)
  }
}

// 计算预期年化收益
const calculateExpectedReturn = (strategy) => {
  if (!strategy.win_rate || !strategy.avg_profit) return 0
  // 基于策略的平均利润和胜率估算年化收益
  // 假设每月进行合理次数的交易（比如网格策略每月10-20次）
  const monthlyTrades = strategy.type === 'grid_trading' ? 15 : 8
  const expectedMonthlyReturn = strategy.win_rate * strategy.avg_profit * monthlyTrades
  return expectedMonthlyReturn * 12 * 100 // 年化并转换为百分比
}

// 计算风险收益比
const calculateRiskRewardRatio = (strategy) => {
  if (!strategy.avg_profit || !strategy.max_drawdown) return 0
  return Math.abs(strategy.avg_profit / strategy.max_drawdown)
}

// 获取置信度等级
const getConfidenceLevel = (confidence) => {
  if (confidence >= 80) return 'high'
  if (confidence >= 60) return 'medium'
  return 'low'
}

// 获取置信度等级文本
const getConfidenceLevelText = (confidence) => {
  if (confidence >= 80) return '高'
  if (confidence >= 60) return '中'
  return '低'
}

// 分析市场环境（使用新的综合接口）
const analyzeMarketEnvironment = async () => {
  try {
    loading.value = true

    // 使用新的综合接口，一次获取所有数据
    const comprehensiveResponse = await api.getComprehensiveMarketAnalysis()
    if (!comprehensiveResponse) {
      throw new Error('API返回null响应，可能存在网络或服务器问题')
    }
    if (comprehensiveResponse.success) {
      const data = comprehensiveResponse.data

      // 数据验证和更新
      if (!data) {
        throw new Error('返回数据为空')
      }

      // 更新市场数据（带验证）
      if (data.market_analysis) {
        validateMarketAnalysisData(data.market_analysis)
        updateMarketData(data.market_analysis)
      } else {
        console.warn('缺少市场分析数据')
      }

      // 更新技术指标（带验证）
      if (data.technical_indicators) {
        validateTechnicalIndicatorsData(data.technical_indicators)
        updateTechnicalIndicators(data.technical_indicators)
      } else {
        console.warn('缺少技术指标数据')
      }

      // 更新策略推荐（带验证）
      if (data.strategy_recommendations && Array.isArray(data.strategy_recommendations)) {
        validateStrategyRecommendationsData(data.strategy_recommendations)
        recommendedStrategies.value = data.strategy_recommendations
        // 数据更新时重置展开状态，默认展开最佳策略
        expandedStrategies.value = []
        if (topStrategy.value) {
          expandedStrategies.value.push(topStrategy.value.type)
        }
      } else {
        console.warn('缺少或无效的策略推荐数据')
      }

      // 分析策略适用性
      analyzeStrategySuitability()

      // 记录处理时间（用于性能监控）
      if (comprehensiveResponse.meta) {
        const isCached = comprehensiveResponse.meta.cached
        const processingTime = comprehensiveResponse.meta.processing_time_ms
        console.log(`市场分析完成，处理时间: ${processingTime}ms, 缓存: ${isCached ? '是' : '否'}`)

        // 可以在这里添加缓存状态显示
        // 例如：在界面上显示"数据来自缓存"或"实时计算"
      }
    } else {
      // 回退到旧的接口方式（兼容性保障）
      console.warn('综合接口失败，使用传统接口')
      await fallbackToLegacyAPIs()
    }

  } catch (error) {
    console.error('市场分析失败:', error)

    // 设置错误状态，保持与后端一致
    if (error.response?.status === 500) {
      // 服务器内部错误，可能是数据问题
      console.warn('服务器内部错误，可能存在数据问题')
    } else if (error.response?.status === 404) {
      // 接口不存在
      console.warn('API接口不存在，请检查后端服务')
    } else if (error.code === 'NETWORK_ERROR') {
      // 网络错误
      console.warn('网络连接失败，请检查网络连接')
    }

    // 如果综合接口失败，尝试使用传统接口
    try {
      await fallbackToLegacyAPIs()
    } catch (fallbackError) {
      console.error('回退接口也失败:', fallbackError)

      // 设置默认状态，避免页面空白
      setDefaultMarketData()
    }
  } finally {
    loading.value = false
  }
}

// 回退到传统接口（兼容性保障）
const fallbackToLegacyAPIs = async () => {
  try {
    // 获取市场数据
    const marketResponse = await api.getMarketAnalysis()
    if (marketResponse && marketResponse.success) {
      updateMarketData(marketResponse.data)
    }

    // 获取技术指标
    const technicalResponse = await api.getTechnicalIndicators()
    if (technicalResponse && technicalResponse.success) {
      updateTechnicalIndicators(technicalResponse.data)
    }

    // 获取策略推荐
    const strategyResponse = await api.getStrategyRecommendations()
    if (strategyResponse && strategyResponse.success) {
      recommendedStrategies.value = strategyResponse.data
    }

    // 分析策略适用性
    analyzeStrategySuitability()

  } catch (error) {
    console.error('传统接口回退失败:', error)
  }
}

// 更新市场数据
const updateMarketData = (data) => {
  // 验证数据有效性
  const volatility = typeof data.volatility === 'number' ? data.volatility : 0
  const oscillation = typeof data.oscillation === 'number' ? data.oscillation : 0
  const trend = typeof data.trend === 'string' ? data.trend : '震荡'

  marketData.value = {
    volatility: volatility.toFixed(2),
    volatilityLevel: getVolatilityLevel(volatility),
    trend: trend,
    trendDirection: getTrendClass(trend),
    oscillation: oscillation.toFixed(2),
    oscillationLevel: getOscillationLevel(oscillation),
    // 保存原始数值供计算使用
    _volatility: volatility,
    _oscillation: oscillation
  }

  marketAnalysis.value = {
    volatilityAnalysis: generateVolatilityAnalysis({ volatility, trend, oscillation }),
    trendAnalysis: generateTrendAnalysis({ volatility, trend, oscillation }),
    strategyAnalysis: generateStrategyAnalysis({ volatility, trend, oscillation })
  }
}

// 更新技术指标
const updateTechnicalIndicators = (data) => {
  technicalIndicators.value = {
    // 基础指标
    btcVolatility: data.btc_volatility?.toFixed(2) || '0.00',
    avgRSI: data.avg_rsi?.toFixed(1) || '0.0',
    strongSymbols: data.strong_symbols || 0,
    weakSymbols: data.weak_symbols || 0,

    // 市场宽度指标
    advanceDeclineRatio: data.advance_decline_ratio?.toFixed(2) || '0.00',
    bigGainers: data.big_gainers || 0,
    bigLosers: data.big_losers || 0,
    neutralSymbols: data.neutral_symbols || 0,

    // 成交量指标
    volumeGainers: data.volume_gainers || 0,
    volumeDecliners: data.volume_decliners || 0,
    avgVolumeChange: data.avg_volume_change?.toFixed(2) || '0.00',

    // 波动率指标
    marketVolatility: data.market_volatility?.toFixed(2) || '0.00',
    highVolatilitySymbols: data.high_volatility_symbols || 0,
    lowVolatilitySymbols: data.low_volatility_symbols || 0
  }
}

// 分析策略适用性 (简化版，不覆盖后端评分)
const analyzeStrategySuitability = () => {
  // 前端不再重新计算评分，完全依赖后端返回的数据
  // 只在这里添加一些前端辅助逻辑，比如推荐标识

  recommendedStrategies.value = recommendedStrategies.value.map(strategy => {
    // 基于后端评分添加推荐标识
    const recommended = strategy.score >= 7

    return {
      ...strategy,
      recommended
    }
  })

  // 更新最佳策略
  if (recommendedStrategies.value.length > 0) {
    topStrategy.value = recommendedStrategies.value[0]
  }
}

// 获取原始数值用于计算
const getRawVolatility = () => marketData.value._volatility || 0
const getRawOscillation = () => marketData.value._oscillation || 0

// 数据验证函数
const validateMarketAnalysisData = (data) => {
  if (typeof data !== 'object' || data === null) {
    throw new Error('市场分析数据格式错误')
  }

  if (typeof data.volatility !== 'number' || data.volatility < 0) {
    console.warn('波动率数据异常:', data.volatility)
    data.volatility = 0
  }

  if (typeof data.oscillation !== 'number' || data.oscillation < 0) {
    console.warn('震荡度数据异常:', data.oscillation)
    data.oscillation = 0
  }

  if (typeof data.trend !== 'string' || data.trend.trim() === '') {
    console.warn('趋势数据异常:', data.trend)
    data.trend = '震荡'
  }
}

const validateTechnicalIndicatorsData = (data) => {
  if (typeof data !== 'object' || data === null) {
    throw new Error('技术指标数据格式错误')
  }

  // 验证数值类型字段
  const numericFields = ['btc_volatility', 'avg_rsi', 'strong_symbols', 'weak_symbols']
  numericFields.forEach(field => {
    if (typeof data[field] !== 'number' && typeof data[field] !== 'undefined') {
      console.warn(`${field}数据类型异常:`, data[field])
      data[field] = 0
    }
  })
}

const validateStrategyRecommendationsData = (data) => {
  if (!Array.isArray(data)) {
    throw new Error('策略推荐数据必须是数组')
  }

  data.forEach((strategy, index) => {
    if (typeof strategy !== 'object' || strategy === null) {
      console.warn(`策略${index}数据格式错误`)
      return
    }

    // 验证必需字段
    if (typeof strategy.name !== 'string') {
      console.warn(`策略${index}缺少name字段`)
      strategy.name = '未知策略'
    }

    if (typeof strategy.score !== 'number') {
      console.warn(`策略${index}score字段异常:`, strategy.score)
      strategy.score = 0
    }

    if (typeof strategy.confidence !== 'number') {
      console.warn(`策略${index}confidence字段异常:`, strategy.confidence)
      strategy.confidence = 0
    }
  })
}

// 设置默认市场数据（错误回退）
const setDefaultMarketData = () => {
  marketData.value = {
    volatility: '0.00',
    volatilityLevel: 'low',
    trend: '数据获取失败',
    trendDirection: 'neutral',
    oscillation: '0.00',
    oscillationLevel: 'low',
    _volatility: 0,
    _oscillation: 0
  }

  technicalIndicators.value = {
    // 基础指标
    btcVolatility: '0.00',
    avgRSI: '0.0',
    strongSymbols: 0,
    weakSymbols: 0,

    // 市场宽度指标
    advanceDeclineRatio: '0.00',
    bigGainers: 0,
    bigLosers: 0,
    neutralSymbols: 0,

    // 成交量指标
    volumeGainers: 0,
    volumeDecliners: 0,
    avgVolumeChange: '0.00',

    // 波动率指标
    marketVolatility: '0.00',
    highVolatilitySymbols: 0,
    lowVolatilitySymbols: 0
  }

  recommendedStrategies.value = []
  marketAnalysis.value = {
    volatilityAnalysis: '数据获取失败，请稍后重试',
    trendAnalysis: '无法获取市场趋势分析',
    strategyAnalysis: '无法获取策略建议'
  }
}

// 工具函数
const getVolatilityLevel = (volatility) => {
  if (volatility > 50) return 'high'
  if (volatility > 25) return 'medium'
  return 'low'
}

const getTrendClass = (trend) => {
  if (trend.includes('上涨')) return 'bullish'
  if (trend.includes('下跌')) return 'bearish'
  return 'neutral'
}

const getOscillationLevel = (oscillation) => {
  if (oscillation > 70) return 'high'
  if (oscillation > 40) return 'medium'
  return 'low'
}

const generateVolatilityAnalysis = (data) => {
  const vol = data.volatility || 0
  if (vol > 50) {
    return `市场波动率较高(${vol.toFixed(1)}%)，适合积极的交易策略，但需要严格的风险控制。`
  } else if (vol > 25) {
    return `市场波动率适中(${vol.toFixed(1)}%)，各类策略都有发挥空间。`
  } else {
    return `市场波动率较低(${vol.toFixed(1)}%)，建议谨慎交易，关注突破信号。`
  }
}

const generateTrendAnalysis = (data) => {
  const trend = data.trend || '震荡'
  const osc = data.oscillation || 0

  if (osc > 70) {
    return `市场处于强烈震荡状态(${osc.toFixed(1)}%)，缺乏明确方向，适合无趋势策略。`
  } else if (osc > 40) {
    return `市场震荡明显(${osc.toFixed(1)}%)，多空力量均衡，适合均值回归策略。`
  } else {
    return `市场有一定趋势性(${trend})，但震荡程度适中，可结合多种策略。`
  }
}

const generateStrategyAnalysis = (data) => {
  const osc = data.oscillation || 0

  if (osc > 60) {
    return '强烈推荐均值回归策略，其他策略在当前市场环境下表现不佳。'
  } else if (osc > 40) {
    return '均值回归策略最适用，其次可以考虑传统策略。'
  } else {
    return '多种策略都有机会，建议根据个人风险偏好选择。'
  }
}

// 刷新分析
const refreshAnalysis = () => {
  analyzeMarketEnvironment()
}

// 组件挂载时执行分析
onMounted(() => {
  analyzeMarketEnvironment()
})
</script>

<style scoped>
.market-analysis {
  padding: 20px 0;
}

.analysis-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
}

.analysis-header h3 {
  margin: 0;
  color: var(--text-primary);
}

.analysis-description {
  margin: 8px 0 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.refresh-btn {
  font-size: 14px;
  padding: 8px 16px;
}

.analysis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.analysis-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: 20px;
  border: 1px solid var(--border-color);
}

.analysis-card h4 {
  margin: 0 0 16px 0;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 600;
}

.market-status .status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-light);
}

.market-status .status-item:last-child {
  border-bottom: none;
}

.status-label {
  color: var(--text-secondary);
  font-size: 14px;
}

.status-value {
  font-weight: 600;
  font-size: 14px;
}

.status-value.low { color: #10b981; }
.status-value.medium { color: #f59e0b; }
.status-value.high { color: #ef4444; }

.status-value.bullish { color: #10b981; }
.status-value.bearish { color: #ef4444; }
.status-value.neutral { color: #6b7280; }

/* 保留基础策略推荐样式 */
.strategy-recommendations {
  display: flex;
  flex-direction: column;
}

.strategy-item.strategy-exists {
  border-left: 4px solid #10b981;
}

.strategy-item.strategy-new {
  border-left: 4px solid #f59e0b;
}

.strategy-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.strategy-badges {
  display: flex;
  align-items: center;
  gap: 8px;
}

.strategy-status {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

/* 旧样式已由优化版本替代 */

.strategy-reason {
  color: var(--text-secondary);
  font-size: 13px;
  margin-bottom: 8px;
}

.strategy-confidence {
  display: flex;
  align-items: center;
  gap: 8px;
}

.strategy-performance {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border-light);
}

.performance-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  margin-bottom: 8px;
}

.performance-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
}

.performance-label {
  color: var(--text-secondary);
}

.performance-value {
  font-weight: 600;
}

.performance-value.win-rate {
  color: #10b981;
}

.performance-value.drawdown {
  color: #ef4444;
}

.performance-value.sharpe {
  color: #3b82f6;
}

.performance-value.trades {
  color: var(--text-primary);
}

.strategy-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  margin-top: 4px;
}

.risk-level {
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.risk-level.low {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
}

.risk-level.medium {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
}

.risk-level.high {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}

.suitable-market {
  color: var(--text-secondary);
  font-style: italic;
}

.confidence-bar {
  flex: 1;
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.confidence-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-color), #10b981);
  transition: width 0.3s ease;
}

.confidence-text {
  font-size: 12px;
  color: var(--text-secondary);
  min-width: 80px;
}

.detailed-analysis {
  margin-bottom: 24px;
}

.full-width {
  grid-column: 1 / -1;
}

.market-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.detail-section h5 {
  margin: 0 0 8px 0;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.detail-section p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.5;
}

.investment-advice {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.advice-item {
  padding: 16px;
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  border-left: 4px solid var(--primary-color);
}

.advice-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}


.advice-title {
  font-weight: 600;
  color: var(--text-primary);
}

.advice-content {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.5;
}

.advice-item.primary {
  border-left-color: #10b981;
  background: rgba(16, 185, 129, 0.05);
}

.advice-item.risk {
  border-left-color: #f59e0b;
  background: rgba(245, 158, 11, 0.05);
}

.advice-item.timing {
  border-left-color: #3b82f6;
  background: rgba(59, 130, 246, 0.05);
}

.technical-indicators {
  margin-top: 16px;
}

.indicator-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.indicator-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px;
  background: var(--bg-primary);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-light);
}

.indicator-name {
  color: var(--text-secondary);
  font-size: 12px;
  margin-bottom: 4px;
}

.indicator-value {
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 600;
}

/* ========== 策略展示优化样式 ========== */

/* 策略头部区域 */
.strategy-header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.strategy-header-section h4 {
  margin: 0;
}

.strategy-controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.btn-link {
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  font-size: 14px;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.2s ease;
}

.btn-link:hover {
  background: var(--primary-light);
}

.strategy-filter {
  padding: 4px 8px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 12px;
  background: var(--bg-primary);
  color: var(--text-primary);
}


/* 策略项优化 */
.strategy-item {
  border: 1px solid var(--border-light);
  border-radius: var(--radius-md);
  margin-bottom: 8px;
  overflow: hidden;
  transition: all 0.2s ease;
}

.strategy-item:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.strategy-item.strategy-top {
  border-left: 3px solid var(--primary-color);
}

.strategy-item.strategy-top-item {
  border-left: 4px solid var(--primary-color);
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.05), rgba(29, 78, 216, 0.05));
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.1);
}

.strategy-item.strategy-top-item .strategy-name {
  font-weight: 700;
  color: var(--primary-color);
}

/* 简要信息行 */
.strategy-summary {
  padding: 12px 16px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--bg-primary);
  transition: background-color 0.2s ease;
}

.strategy-summary:hover {
  background: var(--bg-secondary);
}

.strategy-main-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.strategy-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.strategy-quick-metrics {
  display: flex;
  gap: 8px;
  align-items: center;
}

.strategy-quick-metrics .metric {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
  min-width: 32px;
  text-align: center;
}

.metric.score {
  background: var(--primary-light);
  color: var(--primary-color);
}

.metric.confidence {
  background: var(--success-light);
  color: var(--success-color);
}

.metric.risk {
  font-size: 14px;
  padding: 1px 4px;
}

.strategy-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-badge {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.status-badge.status-exists {
  background: var(--success-light);
  color: var(--success-color);
}

.status-badge.status-new {
  background: var(--info-light);
  color: var(--info-color);
}

.expand-icon {
  font-size: 12px;
  color: var(--text-secondary);
  transition: transform 0.2s ease;
}

.strategy-item.strategy-expanded .expand-icon {
  transform: rotate(180deg);
}

/* 详细信息区域 */
.strategy-details {
  padding: 16px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-light);
}

.strategy-reason {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 12px;
  line-height: 1.5;
}

/* 完整的性能指标 */
.strategy-performance-metrics {
  margin-bottom: 12px;
}

.metrics-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-row {
  display: flex;
  gap: 16px;
}

.metric-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  flex: 1;
  padding: 8px;
  background: var(--bg-secondary);
  border-radius: 6px;
  border: 1px solid var(--border-light);
}

.metric-label {
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
}

.metric-value {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.2;
}

.metric-value.win-rate {
  color: #10b981;
}

.metric-value.drawdown {
  color: #ef4444;
}

.metric-value.sharpe {
  color: #3b82f6;
}

.metric-value.trades {
  color: #6b7280;
}

.metric-value.profit.negative {
  color: #ef4444;
}

.metric-value.profit:not(.negative) {
  color: #10b981;
}

.metric-value.volatility {
  color: #f59e0b;
}

/* 市场适用性标签 */
.strategy-market-fit {
  margin-bottom: 12px;
}

.market-tag {
  background: var(--info-light);
  color: var(--info-color);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  display: inline-block;
}

/* 策略分析洞察 */
.strategy-insights {
  margin-bottom: 12px;
  padding: 12px;
  background: var(--bg-tertiary);
  border-radius: 6px;
  border: 1px solid var(--border-light);
}

.insight-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid var(--border-light);
}

.insight-item:last-child {
  border-bottom: none;
}

.insight-label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.insight-value {
  font-size: 13px;
  font-weight: 600;
}

.insight-value.positive {
  color: #10b981;
}

.insight-value.negative {
  color: #ef4444;
}

.insight-value.good {
  color: #10b981;
}

.insight-value.poor {
  color: #ef4444;
}

.confidence-level.high {
  color: #10b981;
}

.confidence-level.medium {
  color: #f59e0b;
}

.confidence-level.low {
  color: #ef4444;
}

/* 策略操作按钮 */
.strategy-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.btn-secondary.small {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-secondary.small:hover {
  background: var(--bg-tertiary);
}

/* 统计摘要 */
.strategy-summary-footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--border-light);
  text-align: center;
}

.summary-stats {
  display: flex;
  justify-content: center;
  gap: 16px;
  font-size: 12px;
  color: var(--text-secondary);
}

/* 响应式设计优化 */
@media (max-width: 768px) {
  .analysis-grid {
    grid-template-columns: 1fr;
  }

  .market-details {
    grid-template-columns: 1fr;
  }

  .indicator-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  /* 移动端策略展示优化 */
  .strategy-header-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .strategy-controls {
    width: 100%;
    justify-content: space-between;
  }

  .top-strategy-content {
    flex-direction: column;
    text-align: center;
    gap: 12px;
  }

  .top-strategy-metrics {
    justify-content: center;
  }

  .strategy-summary {
    padding: 8px 12px;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .strategy-main-info {
    width: 100%;
    justify-content: space-between;
  }

  .strategy-status {
    align-self: flex-end;
  }

  .metric-group {
    flex-direction: column;
    gap: 8px;
  }

  .strategy-actions {
    justify-content: center;
  }

  .summary-stats {
    flex-direction: column;
    gap: 4px;
  }

  /* 移动端性能指标优化 */
  .metric-row {
    flex-direction: column;
    gap: 8px;
  }

  .metric-item {
    padding: 6px;
  }

  .metric-label {
    font-size: 10px;
  }

  .metric-value {
    font-size: 13px;
  }

  /* 移动端策略洞察优化 */
  .strategy-insights {
    padding: 8px;
  }

  .insight-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .insight-value {
    font-size: 12px;
  }

  /* 移动端第一名策略优化 */
  .strategy-item.strategy-top-item .strategy-name {
    font-size: 15px;
  }
}
</style>