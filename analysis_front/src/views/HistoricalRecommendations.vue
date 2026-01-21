<template>
  <div class="historical-recommendations" style="padding: 0;">
    <!-- 页面标题和时间选择器 -->
    <section class="panel">
      <div class="row">
        <h2 style="color: #2c3e50; font-weight: 600; margin: 0;">📊 历史推荐查询</h2>
        <div class="spacer"></div>
        <label>类型：</label>
        <select v-model="kind" @change="loadTimeList">
          <option value="spot">现货</option>
          <option value="futures">期货</option>
        </select>
        <label style="margin-left: 12px;">日期：</label>
        <input 
          type="date" 
          v-model="selectedDate" 
          @change="handleDateChange"
          :max="maxDate"
          :min="minDate"
          :disabled="loading"
          style="padding: 6px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;"
        />
        <label style="margin-left: 12px;">排序：</label>
        <select 
          v-model="sortBy" 
          @change="applySorting"
          :disabled="loading || recommendations.length === 0"
          style="padding: 6px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;"
        >
          <option value="score">推荐得分</option>
          <option value="return_24h">24h收益率</option>
          <option value="return_7d">7天收益率</option>
          <option value="return_30d">30天收益率</option>
          <option value="max_gain">最大涨幅</option>
          <option value="max_drawdown">最大回撤</option>
        </select>
        <label style="margin-left: 12px;">筛选：</label>
        <select 
          v-model="filterBy" 
          @change="applyFiltering"
          :disabled="loading || recommendations.length === 0"
          style="padding: 6px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px;"
        >
          <option value="all">全部</option>
          <option value="profit">盈利</option>
          <option value="loss">亏损</option>
          <option value="no_data">无数据</option>
        </select>
        <button class="primary" @click="loadRecommendations" :disabled="!selectedDate || loading" style="margin-left: 8px;">
          {{ loading ? '加载中...' : '查询' }}
        </button>
        <button
          v-if="selectedDate && !hasDataForDate"
          @click="generateRecommendations"
          :disabled="generating"
          style="margin-left: 8px;"
        >
          {{ generating ? '生成中...' : '生成推荐' }}
        </button>
        <button
          v-if="selectedDate && hasDataForDate && recommendations.length > 0"
          @click="updateBacktestData"
          :disabled="updatingBacktest"
          style="margin-left: 8px; background: #f39c12; color: white; border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer;"
        >
          {{ updatingBacktest ? '更新中...' : '更新回测' }}
        </button>
      </div>
    </section>

    <!-- 加载状态 -->
    <section class="panel" v-if="loading">
      <div class="loading-container">
        <div class="spinner"></div>
        <p>正在加载推荐数据...</p>
      </div>
    </section>

    <!-- 空状态 -->
    <section class="panel" v-else-if="!selectedDate">
      <div class="empty-state">
        <p>请选择一个日期查看历史推荐</p>
      </div>

      <!-- 分页组件 -->
      <div style="margin-top: 20px;" v-if="pagination.total > 0">
        <Pagination
          :page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          :loading="loading"
          @change="handlePageChange"
        />
      </div>
    </section>

    <!-- 无数据提示（有日期但无数据） -->
    <section class="panel" v-else-if="selectedDate && !loading && !hasDataForDate && recommendations.length === 0">
      <div class="empty-state">
        <p>该日期（{{ selectedDate }}）暂无推荐数据</p>
        <p style="margin-top: 12px; color: #666; font-size: 14px;">
          点击"生成推荐"按钮，系统将使用该日期的历史市场数据重新生成推荐。
        </p>
      </div>
    </section>

    <!-- 推荐列表 -->
    <section class="panel" v-else-if="recommendations.length > 0">
      <div class="row">
        <h3 style="color: #2c3e50; font-weight: 600; margin: 0;">📈 推荐列表（{{ selectedDate }}）</h3>
        <div class="spacer"></div>
        <label style="margin-right: 8px;">每页：</label>
        <select 
          v-model="pagination.pageSize" 
          @change="handlePageSizeChange"
          :disabled="loading"
          style="padding: 4px 8px; margin-right: 12px;"
        >
          <option :value="10">10</option>
          <option :value="20">20</option>
          <option :value="50">50</option>
        </select>
        <span class="info-text">共 {{ pagination.total }} 个推荐（当前页显示 {{ filteredRecommendations.length }} 个）</span>
      </div>

      <!-- 统计信息（当前页统计） -->
      <div class="stats-summary" v-if="filteredRecommendations.length > 0">
        <div class="stat-item">
          <span class="stat-label">总推荐数：</span>
          <span class="stat-value">{{ pagination.total }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前页盈利：</span>
          <span class="stat-value positive">{{ profitCount }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前页亏损：</span>
          <span class="stat-value negative">{{ lossCount }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前页胜率：</span>
          <span class="stat-value">{{ winRate }}%</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前页盈利比率：</span>
          <span class="stat-value positive">{{ profitRate }}%</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前页亏损比率：</span>
          <span class="stat-value negative">{{ lossRate }}%</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">当前页平均24h收益：</span>
          <span class="stat-value" :class="getPerformanceClass(avgReturn24h)">
            {{ formatPercent(avgReturn24h) }}
          </span>
        </div>
      </div>

      <div class="recommendations-grid">
        <div
          v-for="(rec, index) in filteredRecommendations"
          :key="`${rec.id}-${index}`"
          class="recommendation-card"
          :class="{ 
            'has-performance': rec.performance,
            'profit': isProfit(rec),
            'loss': isLoss(rec)
          }"
        >
          <!-- 推荐基本信息 -->
          <div class="card-header">
            <div class="rank-badge">#{{ rec.rank }}</div>
            <div class="symbol-info">
              <h4>{{ rec.symbol }}</h4>
              <span class="base-symbol">{{ rec.base_symbol }}</span>
            </div>
            <div class="header-right">
              <div class="status-badge" :class="getStatusClass(rec)">
                {{ getStatusText(rec) }}
              </div>
              <div class="score-badge" :class="getScoreClass(rec.total_score)">
                {{ rec.total_score.toFixed(1) }}分
              </div>
            </div>
          </div>

          <!-- 推荐得分 -->
          <div class="scores-section">
            <div class="score-item">
              <span class="score-label">市场</span>
              <span class="score-value">{{ rec.scores.market.toFixed(1) }}</span>
            </div>
            <div class="score-item">
              <span class="score-label">资金流</span>
              <span class="score-value">{{ rec.scores.flow.toFixed(1) }}</span>
            </div>
            <div class="score-item">
              <span class="score-label">热度</span>
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

          <!-- 币种统计信息 -->
          <div class="symbol-stats-section" v-if="symbolStats[rec.base_symbol]">
            <div class="symbol-stats-header">
              <h5>{{ rec.base_symbol }} 统计</h5>
            </div>
            <div class="symbol-stats-grid">
              <div class="symbol-stat-item">
                <span class="symbol-stat-label">推荐次数</span>
                <span class="symbol-stat-value">{{ symbolStats[rec.base_symbol].total }}</span>
              </div>
              <div class="symbol-stat-item">
                <span class="symbol-stat-label">盈利次数</span>
                <span class="symbol-stat-value positive">{{ symbolStats[rec.base_symbol].profit }}</span>
              </div>
              <div class="symbol-stat-item">
                <span class="symbol-stat-label">亏损次数</span>
                <span class="symbol-stat-value negative">{{ symbolStats[rec.base_symbol].loss }}</span>
              </div>
              <div class="symbol-stat-item">
                <span class="symbol-stat-label">胜率</span>
                <span class="symbol-stat-value" :class="getPerformanceClass(symbolStats[rec.base_symbol].winRate - 50)">
                  {{ symbolStats[rec.base_symbol].winRate }}%
                </span>
              </div>
              <div class="symbol-stat-item">
                <span class="symbol-stat-label">平均收益</span>
                <span class="symbol-stat-value" :class="getPerformanceClass(symbolStats[rec.base_symbol].avgReturn24h)">
                  {{ formatPercent(symbolStats[rec.base_symbol].avgReturn24h) }}
                </span>
              </div>
            </div>
          </div>

          <!-- 实际表现数据 -->
          <div v-if="rec.performance" class="performance-section">
            <div class="performance-header">
              <h4>实际表现</h4>
            </div>
            <div class="performance-grid">
              <div class="performance-item">
                <span class="perf-label">推荐价格</span>
                <span class="perf-value">${{ formatNumber(rec.performance.recommended_price) }}</span>
              </div>
              <div class="performance-item" v-if="rec.performance.current_price">
                <span class="perf-label">当前价格</span>
                <span class="perf-value">${{ formatNumber(rec.performance.current_price) }}</span>
              </div>
              <div class="performance-item" v-if="rec.performance.current_return !== null && rec.performance.current_return !== undefined">
                <span class="perf-label">当前收益</span>
                <span class="perf-value" :class="getReturnClass(rec.performance.current_return)">
                  {{ formatPercent(rec.performance.current_return) }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.return_24h !== null && rec.performance.return_24h !== undefined">
                <span class="perf-label">24h收益</span>
                <span class="perf-value" :class="getReturnClass(rec.performance.return_24h)">
                  {{ formatPercent(rec.performance.return_24h) }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.return_7d !== null && rec.performance.return_7d !== undefined">
                <span class="perf-label">7天收益</span>
                <span class="perf-value" :class="getReturnClass(rec.performance.return_7d)">
                  {{ formatPercent(rec.performance.return_7d) }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.return_30d !== null && rec.performance.return_30d !== undefined">
                <span class="perf-label">30天收益</span>
                <span class="perf-value" :class="getReturnClass(rec.performance.return_30d)">
                  {{ formatPercent(rec.performance.return_30d) }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.max_gain !== null && rec.performance.max_gain !== undefined">
                <span class="perf-label">最大涨幅</span>
                <span class="perf-value positive">
                  +{{ formatPercent(rec.performance.max_gain) }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.max_drawdown !== null && rec.performance.max_drawdown !== undefined">
                <span class="perf-label">最大回撤</span>
                <span class="perf-value negative">
                  {{ formatPercent(rec.performance.max_drawdown) }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.is_win !== null && rec.performance.is_win !== undefined">
                <span class="perf-label">24h结果</span>
                <span class="perf-value" :class="rec.performance.is_win ? 'positive' : 'negative'">
                  {{ rec.performance.is_win ? '盈利' : '亏损' }}
                </span>
              </div>
              <div class="performance-item" v-if="rec.performance.performance_rating">
                <span class="perf-label">评级</span>
                <span class="perf-value" :class="getRatingClass(rec.performance.performance_rating)">
                  {{ getRatingText(rec.performance.performance_rating) }}
                </span>
              </div>
            </div>

            <!-- 交易信号和策略 -->
            <div v-if="rec.prediction && rec.prediction.trading_strategy" class="trading-strategy-section">
              <div class="strategy-header">
                <h5>📈 交易策略</h5>
              </div>
              <div class="strategy-content">
                <div class="strategy-item">
                  <span class="strategy-label">策略类型</span>
                  <span class="strategy-value" :class="getStrategyClass(rec.prediction.trading_strategy.strategy_type)">
                    {{ getStrategyText(rec.prediction.trading_strategy.strategy_type) }}
                  </span>
                </div>
                <div class="strategy-item">
                  <span class="strategy-label">入场区间</span>
                  <span class="strategy-value">
                    ${{ formatNumber(rec.prediction.trading_strategy.entry_zone.min) }} -
                    ${{ formatNumber(rec.prediction.trading_strategy.entry_zone.max) }}
                  </span>
                </div>
                <div class="strategy-item" v-if="rec.prediction.trading_strategy.exit_targets.length > 0">
                  <span class="strategy-label">目标价格</span>
                  <span class="strategy-value positive">
                    ${{ formatNumber(rec.prediction.trading_strategy.exit_targets[0].avg) }}
                  </span>
                </div>
                <div class="strategy-item" v-if="rec.prediction.trading_strategy.stop_loss_levels.length > 0">
                  <span class="strategy-label">止损价格</span>
                  <span class="strategy-value negative">
                    ${{ formatNumber(rec.prediction.trading_strategy.stop_loss_levels[0].level) }}
                  </span>
                </div>
                <div class="strategy-item">
                  <span class="strategy-label">建议仓位</span>
                  <span class="strategy-value">
                    {{ (rec.prediction.trading_strategy.position_sizing.adjusted_position * 100).toFixed(1) }}%
                  </span>
                </div>
                <div class="strategy-item">
                  <span class="strategy-label">风险收益比</span>
                  <span class="strategy-value" :class="rec.prediction.trading_strategy.risk_management.risk_reward_ratio >= 2 ? 'positive' : 'neutral'">
                    1:{{ rec.prediction.trading_strategy.risk_management.risk_reward_ratio.toFixed(1) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- 技术指标信号 -->
            <div v-if="rec.technical && rec.technical.trading_signal" class="technical-signal-section">
              <div class="signal-header">
                <h5>🎯 技术信号</h5>
              </div>
              <div class="signal-content">
                <div class="signal-item">
                  <span class="signal-label">交易信号</span>
                  <span class="signal-value" :class="getSignalClass(rec.technical.trading_signal.signal)">
                    {{ getSignalText(rec.technical.trading_signal.signal) }}
                  </span>
                </div>
                <div class="signal-item">
                  <span class="signal-label">信号强度</span>
                  <span class="signal-value" :class="getSignalStrengthClass(rec.technical.trading_signal.strength)">
                    {{ rec.technical.trading_signal.strength.toFixed(1) }}%
                  </span>
                </div>
                <div class="signal-item" v-if="rec.technical.trading_signal.signal !== 'HOLD'">
                  <span class="signal-label">建议入场</span>
                  <span class="signal-value">
                    ${{ formatNumber(rec.technical.trading_signal.entry_price) }}
                  </span>
                </div>
                <div class="signal-item" v-if="rec.technical.trading_signal.stop_loss > 0">
                  <span class="signal-label">止损价格</span>
                  <span class="signal-value negative">
                    ${{ formatNumber(rec.technical.trading_signal.stop_loss) }}
                  </span>
                </div>
                <div class="signal-item" v-if="rec.technical.trading_signal.take_profit > 0">
                  <span class="signal-label">止盈价格</span>
                  <span class="signal-value positive">
                    ${{ formatNumber(rec.technical.trading_signal.take_profit) }}
                  </span>
                </div>
                <div class="signal-item">
                  <span class="signal-label">风险等级</span>
                  <span class="signal-value" :class="getRiskLevelClass(rec.technical.position_management.risk_level)">
                    {{ getRiskLevelText(rec.technical.position_management.risk_level) }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- 无表现数据提示 -->
          <div v-else class="no-performance">
            <p>暂无表现数据</p>
          </div>

          <!-- 推荐理由 -->
          <div v-if="rec.reasons && rec.reasons.length > 0" class="reasons-section">
            <h5>推荐理由</h5>
            <ul>
              <li v-for="(reason, idx) in rec.reasons" :key="idx">{{ reason }}</li>
            </ul>
          </div>
        </div>
      </div>
    </section>


    <!-- 错误提示 -->
    <section class="panel error-panel" v-if="error">
      <div class="error-message">
        <p>❌ {{ error }}</p>
        <button @click="loadRecommendations">重试</button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed, nextTick } from 'vue'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'

const kind = ref('spot')
const selectedDate = ref('')
const availableDates = ref([])
const recommendations = ref([])
const loading = ref(false)
const loadingDates = ref(false)
const generating = ref(false)
const updatingBacktest = ref(false)
const error = ref(null)
const hasDataForDate = ref(false)

// 排序和筛选
const sortBy = ref('score')
const filterBy = ref('all')

// 分页信息
const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
  totalPages: 0
})

// 计算日期选择器的最大和最小日期
const today = new Date()
const maxDate = today.toISOString().split('T')[0]
const minDate = new Date(today.getFullYear() - 1, today.getMonth(), today.getDay()).toISOString().split('T')[0]

// 加载时间列表（用于显示已有数据的日期）
async function loadTimeList() {
  loadingDates.value = true
  error.value = null
  try {
    const res = await api.getRecommendationTimeList({ kind: kind.value, limit: 100 })
    availableDates.value = res.dates || []
    
    // 默认选择今天
    if (!selectedDate.value) {
      selectedDate.value = maxDate
      handleDateChange()
    }
  } catch (err) {
    error.value = `加载时间列表失败: ${err.message || '未知错误'}`
    console.error('Failed to load time list:', err)
  } finally {
    loadingDates.value = false
  }
}

// 处理日期变化
async function handleDateChange() {
  if (!selectedDate.value) {
    recommendations.value = []
    hasDataForDate.value = false
    pagination.value.page = 1
    pagination.value.total = 0
    return
  }
  
  // 重置分页到第一页
  pagination.value.page = 1
  
  // 检查该日期是否有数据
  const hasData = availableDates.value.includes(selectedDate.value)
  hasDataForDate.value = hasData
  
  // 如果有数据，直接加载
  if (hasData) {
    await loadRecommendations()
  } else {
    // 如果没有数据，先尝试查询一次（可能数据刚生成）
    await loadRecommendations()
    // 如果还是没有数据，hasDataForDate保持false，显示生成按钮
    if (pagination.value.total === 0) {
      hasDataForDate.value = false
    }
  }
}

// 加载推荐数据
async function loadRecommendations() {
  if (!selectedDate.value) {
    recommendations.value = []
    hasDataForDate.value = false
    pagination.value.total = 0
    return
  }

  console.log('[HistoricalRecommendations] Starting loadRecommendations, setting loading=true')
  loading.value = true
  error.value = null
  try {
    const res = await api.getHistoricalRecommendations({
      kind: kind.value,
      date: selectedDate.value,
      includePerformance: true,
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    })

    console.log('[HistoricalRecommendations] API call completed successfully')
    const rawRecommendations = res.recommendations || []

    console.log('[HistoricalRecommendations] Raw response:', {
      total: res.total,
      recommendationsCount: rawRecommendations.length,
      hasRecommendations: rawRecommendations.length > 0,
      sample: rawRecommendations.length > 0 ? {
        id: rawRecommendations[0].id,
        symbol: rawRecommendations[0].symbol,
        total_score: rawRecommendations[0].total_score
      } : null
    })

    // 去重：根据ID去重，避免重复显示
    const uniqueMap = new Map()
    for (const rec of rawRecommendations) {
      if (rec.id && !uniqueMap.has(rec.id)) {
        uniqueMap.set(rec.id, rec)
      }
    }
    recommendations.value = Array.from(uniqueMap.values())

    console.log('[HistoricalRecommendations] After processing:', {
      originalCount: rawRecommendations.length,
      uniqueCount: recommendations.value.length,
      hasDataForDate: hasDataForDate.value,
      paginationTotal: pagination.value.total
    })

    // 调试：检查数据结构
    if (recommendations.value.length > 0) {
      console.log('[HistoricalRecommendations] Loaded recommendations:', {
        total: recommendations.value.length,
        sample: {
          id: recommendations.value[0].id,
          symbol: recommendations.value[0].symbol,
          hasPerformance: !!recommendations.value[0].performance,
          performance: recommendations.value[0].performance
        }
      })
      
      // 检查是否有重复的ID
      const ids = recommendations.value.map(r => r.id)
      const uniqueIds = new Set(ids)
      if (ids.length !== uniqueIds.size) {
        console.warn('[HistoricalRecommendations] Found duplicate IDs:', {
          total: ids.length,
          unique: uniqueIds.size,
          duplicates: ids.filter((id, index) => ids.indexOf(id) !== index)
        })
      }
    }
    
    // 更新分页信息
    pagination.value.total = res.total || 0
    pagination.value.page = res.page || 1
    pagination.value.pageSize = res.page_size || 10
    pagination.value.totalPages = res.total_pages || 0
    
    // 更新数据存在状态
    hasDataForDate.value = pagination.value.total > 0

    // 如果找到数据，更新可用日期列表
    if (pagination.value.total > 0 && !availableDates.value.includes(selectedDate.value)) {
      availableDates.value.push(selectedDate.value)
      availableDates.value.sort().reverse() // 按日期降序排列
    }

    // 强制触发响应式更新
    await nextTick()
    console.log('[HistoricalRecommendations] Data updated, forcing UI refresh')
  } catch (err) {
    error.value = `加载推荐数据失败: ${err.message || '未知错误'}`
    console.error('Failed to load recommendations:', err)
    recommendations.value = []
    hasDataForDate.value = false
    pagination.value.total = 0
  } finally {
    console.log('[HistoricalRecommendations] loadRecommendations completed, setting loading=false')
    loading.value = false
  }
}

// 处理分页变化
async function handlePageChange(newPage) {
  pagination.value.page = newPage
  await loadRecommendations()
  // 滚动到顶部
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 处理每页数量变化
function handlePageSizeChange() {
  pagination.value.page = 1 // 重置到第一页
  loadRecommendations()
  // 滚动到顶部
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 判断是否盈利（优先使用is_win字段，否则使用return_24h）
function isProfit(rec) {
  if (!rec.performance) return false
  
  // 优先使用is_win字段
  if (rec.performance.is_win !== null && rec.performance.is_win !== undefined) {
    return rec.performance.is_win === true
  }
  
  // 如果没有is_win，使用return_24h判断
  if (rec.performance.return_24h !== null && rec.performance.return_24h !== undefined) {
    return rec.performance.return_24h > 0
  }
  
  return false
}

// 判断是否亏损
function isLoss(rec) {
  if (!rec.performance) return false
  
  // 优先使用is_win字段
  if (rec.performance.is_win !== null && rec.performance.is_win !== undefined) {
    return rec.performance.is_win === false
  }
  
  // 如果没有is_win，使用return_24h判断
  if (rec.performance.return_24h !== null && rec.performance.return_24h !== undefined) {
    return rec.performance.return_24h < 0
  }
  
  return false
}

// 判断是否有数据
function hasPerformanceData(rec) {
  if (!rec.performance) return false
  
  // 有is_win字段就算有数据
  if (rec.performance.is_win !== null && rec.performance.is_win !== undefined) {
    return true
  }
  
  // 或者有return_24h字段也算有数据
  if (rec.performance.return_24h !== null && rec.performance.return_24h !== undefined) {
    return true
  }
  
  return false
}

// 应用筛选
const filteredRecommendations = computed(() => {
  console.log('[HistoricalRecommendations] Computing filteredRecommendations:', {
    recommendationsLength: recommendations.value.length,
    filterBy: filterBy.value,
    sortBy: sortBy.value
  })

  let filtered = [...recommendations.value]
  
  // 应用筛选
  if (filterBy.value === 'profit') {
    filtered = filtered.filter(rec => isProfit(rec))
  } else if (filterBy.value === 'loss') {
    filtered = filtered.filter(rec => isLoss(rec))
  } else if (filterBy.value === 'no_data') {
    filtered = filtered.filter(rec => !hasPerformanceData(rec))
  }
  
  // 应用排序
  filtered.sort((a, b) => {
    if (sortBy.value === 'score') {
      return (b.total_score || 0) - (a.total_score || 0)
    } else if (sortBy.value === 'return_24h') {
      const aReturn = a.performance?.return_24h ?? -Infinity
      const bReturn = b.performance?.return_24h ?? -Infinity
      return bReturn - aReturn
    } else if (sortBy.value === 'return_7d') {
      const aReturn = a.performance?.return_7d ?? -Infinity
      const bReturn = b.performance?.return_7d ?? -Infinity
      return bReturn - aReturn
    } else if (sortBy.value === 'return_30d') {
      const aReturn = a.performance?.return_30d ?? -Infinity
      const bReturn = b.performance?.return_30d ?? -Infinity
      return bReturn - aReturn
    } else if (sortBy.value === 'max_gain') {
      const aGain = a.performance?.max_gain ?? -Infinity
      const bGain = b.performance?.max_gain ?? -Infinity
      return bGain - aGain
    } else if (sortBy.value === 'max_drawdown') {
      const aDrawdown = a.performance?.max_drawdown ?? Infinity
      const bDrawdown = b.performance?.max_drawdown ?? Infinity
      return aDrawdown - bDrawdown // 回撤越小越好，所以升序
    }
    return 0
  })
  
  return filtered
})

// 统计信息（基于当前页数据，注意：这是当前页的统计，不是全部数据的统计）
const profitCount = computed(() => {
  return recommendations.value.filter(rec => isProfit(rec)).length
})

const lossCount = computed(() => {
  return recommendations.value.filter(rec => isLoss(rec)).length
})

const winRate = computed(() => {
  const total = profitCount.value + lossCount.value
  if (total === 0) return 0
  return ((profitCount.value / total) * 100).toFixed(1)
})

const profitRate = computed(() => {
  const total = recommendations.value.length
  if (total === 0) return 0
  return ((profitCount.value / total) * 100).toFixed(1)
})

const lossRate = computed(() => {
  const total = recommendations.value.length
  if (total === 0) return 0
  return ((lossCount.value / total) * 100).toFixed(1)
})

const avgReturn24h = computed(() => {
  const returns = recommendations.value
    .map(rec => {
      // 优先使用return_24h，如果没有则尝试其他收益率字段
      if (rec.performance?.return_24h !== null && rec.performance?.return_24h !== undefined) {
        return rec.performance.return_24h
      }
      if (rec.performance?.current_return !== null && rec.performance?.current_return !== undefined) {
        return rec.performance.current_return
      }
      return null
    })
    .filter(r => r !== null && r !== undefined)

  if (returns.length === 0) return 0
  const sum = returns.reduce((a, b) => a + b, 0)
  return sum / returns.length
})

// 币种统计信息
const symbolStats = computed(() => {
  const stats = {}

  recommendations.value.forEach(rec => {
    const symbol = rec.base_symbol
    if (!stats[symbol]) {
      stats[symbol] = {
        total: 0,
        profit: 0,
        loss: 0,
        winRate: 0,
        avgReturn24h: 0,
        returns: []
      }
    }

    stats[symbol].total++

    if (isProfit(rec)) {
      stats[symbol].profit++
    }

    if (isLoss(rec)) {
      stats[symbol].loss++
    }

    // 收集收益率数据
    let returnValue = null
    if (rec.performance?.return_24h !== null && rec.performance?.return_24h !== undefined) {
      returnValue = rec.performance.return_24h
    } else if (rec.performance?.current_return !== null && rec.performance?.current_return !== undefined) {
      returnValue = rec.performance.current_return
    }

    if (returnValue !== null) {
      stats[symbol].returns.push(returnValue)
    }
  })

  // 计算胜率和平均收益率
  Object.keys(stats).forEach(symbol => {
    const stat = stats[symbol]
    const profitableTrades = stat.profit + stat.loss
    if (profitableTrades > 0) {
      stat.winRate = ((stat.profit / profitableTrades) * 100).toFixed(1)
    } else {
      stat.winRate = 0
    }

    if (stat.returns.length > 0) {
      stat.avgReturn24h = stat.returns.reduce((a, b) => a + b, 0) / stat.returns.length
    } else {
      stat.avgReturn24h = 0
    }
  })

  return stats
})

// 应用排序
function applySorting() {
  // 排序逻辑已在computed中实现
}

// 应用筛选
function applyFiltering() {
  // 筛选逻辑已在computed中实现
}

// 获取表现样式类
function getPerformanceClass(value) {
  if (value === null || value === undefined) return ''
  return value >= 0 ? 'positive' : 'negative'
}

// 生成推荐数据（支持历史日期）
async function generateRecommendations() {
  if (!selectedDate.value) return

  generating.value = true
  error.value = null

  try {
    // 调用生成推荐API（为指定日期生成）
    await api.generateRecommendationsForDate({
      kind: kind.value,
      date: selectedDate.value,
      limit: 10
    })

    // 等待一下让数据保存完成
    await new Promise(resolve => setTimeout(resolve, 1000))

    // 生成后重新加载该日期的数据
    await loadRecommendations()

    // 更新可用日期列表
    if (!availableDates.value.includes(selectedDate.value)) {
      availableDates.value.push(selectedDate.value)
      availableDates.value.sort().reverse()
    }
  } catch (err) {
    error.value = `生成推荐失败: ${err.message || '未知错误'}`
    console.error('Failed to generate recommendations:', err)
  } finally {
    generating.value = false
  }
}

// 更新回测数据
async function updateBacktestData() {
  if (!selectedDate.value) return

  updatingBacktest.value = true
  error.value = null

  try {
    // 调用批量更新回测API
    await api.batchUpdateBacktestRecords({})

    // 等待更新完成
    await new Promise(resolve => setTimeout(resolve, 2000))

    // 重新加载数据
    await loadRecommendations()
  } catch (err) {
    error.value = `更新回测数据失败: ${err.message || '未知错误'}`
    console.error('Failed to update backtest data:', err)
  } finally {
    updatingBacktest.value = false
  }
}

// 格式化日期（用于下拉框显示，显示完整日期）
function formatDateForSelect(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr + 'T00:00:00')
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  const todayStr = today.toISOString().split('T')[0]
  const yesterdayStr = yesterday.toISOString().split('T')[0]

  // 在下拉框中显示完整日期，但标注今天/昨天
  if (dateStr === todayStr) {
    return `${dateStr} (今天)`
  } else if (dateStr === yesterdayStr) {
    return `${dateStr} (昨天)`
  } else {
    return dateStr
  }
}

// 格式化日期（用于页面显示）
function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr + 'T00:00:00')
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  if (dateStr === today.toISOString().split('T')[0]) {
    return '今天'
  } else if (dateStr === yesterday.toISOString().split('T')[0]) {
    return '昨天'
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

// 格式化百分比
function formatPercent(value) {
  if (value === null || value === undefined) return '-'
  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
}

// 格式化数字
function formatNumber(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  }
  return value.toFixed(4)
}

// 获取得分样式类
function getScoreClass(score) {
  if (score >= 80) return 'score-high'
  if (score >= 60) return 'score-medium'
  return 'score-low'
}

// 获取收益率样式类
function getReturnClass(value) {
  if (value === null || value === undefined) return ''
  return value >= 0 ? 'positive' : 'negative'
}

// 获取评级样式类
function getRatingClass(rating) {
  if (!rating) return ''
  const ratingMap = {
    'excellent': 'rating-excellent',
    'good': 'rating-good',
    'average': 'rating-average',
    'poor': 'rating-poor'
  }
  return ratingMap[rating] || ''
}

// 获取评级文本
function getRatingText(rating) {
  if (!rating) return '-'
  const textMap = {
    'excellent': '优秀',
    'good': '良好',
    'average': '一般',
    'poor': '较差'
  }
  return textMap[rating] || rating
}

// 获取策略类型样式类
function getStrategyClass(strategyType) {
  const classMap = {
    'LONG': 'positive',
    'SHORT': 'negative',
    'RANGE': 'neutral'
  }
  return classMap[strategyType] || 'neutral'
}

// 获取策略类型文本
function getStrategyText(strategyType) {
  const textMap = {
    'LONG': '多头策略',
    'SHORT': '空头策略',
    'RANGE': '震荡策略'
  }
  return textMap[strategyType] || strategyType
}

// 获取信号样式类
function getSignalClass(signal) {
  const classMap = {
    'BUY': 'positive',
    'SELL': 'negative',
    'HOLD': 'neutral'
  }
  return classMap[signal] || 'neutral'
}

// 获取信号文本
function getSignalText(signal) {
  const textMap = {
    'BUY': '买入',
    'SELL': '卖出',
    'HOLD': '观望'
  }
  return textMap[signal] || signal
}

// 获取信号强度样式类
function getSignalStrengthClass(strength) {
  if (strength >= 70) return 'positive'
  if (strength >= 40) return 'neutral'
  return 'negative'
}

// 获取风险等级样式类
function getRiskLevelClass(riskLevel) {
  const classMap = {
    'low': 'positive',
    'medium': 'neutral',
    'high': 'negative'
  }
  return classMap[riskLevel] || 'neutral'
}

// 获取风险等级文本
function getRiskLevelText(riskLevel) {
  const textMap = {
    'low': '低风险',
    'medium': '中风险',
    'high': '高风险'
  }
  return textMap[riskLevel] || riskLevel
}

// 获取状态文本（盈利/亏损/计算中/无数据/失败）
function getStatusText(rec) {
  if (!rec.performance) {
    return '无数据'
  }

  const backtestStatus = rec.performance.backtest_status
  const status = rec.performance.status

  // 优先检查是否有24h数据，如果有数据就显示盈利/亏损
  const has24hData = rec.performance.return_24h !== null && rec.performance.return_24h !== undefined
  const hasWinData = rec.performance.is_win !== null && rec.performance.is_win !== undefined

  // 如果有24h数据或is_win数据，优先显示盈利/亏损
  if (hasWinData) {
    return rec.performance.is_win ? '盈利' : '亏损'
  }

  if (has24hData) {
    return rec.performance.return_24h > 0 ? '盈利' : rec.performance.return_24h < 0 ? '亏损' : '持平'
  }

  // 根据backtest_status显示状态
  if (backtestStatus === 'failed') {
    return '获取失败'
  }

  if (backtestStatus === 'completed') {
    return '已完成'
  }

  if (backtestStatus === 'pending' || backtestStatus === 'processing') {
    return '计算中'
  }

  if (backtestStatus === 'tracking') {
    return '更新中'
  }

  // 默认显示无数据
  return '无数据'
}

// 获取状态样式类
function getStatusClass(rec) {
  if (!rec.performance) {
    return 'status-no-data'
  }

  const backtestStatus = rec.performance.backtest_status
  const has24hData = rec.performance.return_24h !== null && rec.performance.return_24h !== undefined
  const hasWinData = rec.performance.is_win !== null && rec.performance.is_win !== undefined

  // 优先检查是否有数据，如果有数据就显示盈利/亏损样式
  if (hasWinData) {
    return rec.performance.is_win ? 'status-profit' : 'status-loss'
  }

  if (has24hData) {
    if (rec.performance.return_24h > 0) return 'status-profit'
    if (rec.performance.return_24h < 0) return 'status-loss'
    return 'status-neutral'
  }

  // 根据backtest_status显示样式
  if (backtestStatus === 'failed') {
    return 'status-failed'
  }

  if (backtestStatus === 'completed') {
    return 'status-completed'
  }

  if (backtestStatus === 'pending' || backtestStatus === 'processing') {
    return 'status-calculating'
  }

  if (backtestStatus === 'tracking') {
    return 'status-calculating'
  }

  // 默认显示"无数据"样式
  return 'status-no-data'
}

// 监听类型变化，重新加载时间列表
watch(kind, () => {
  selectedDate.value = ''
  recommendations.value = []
  loadTimeList()
})

onMounted(() => {
  loadTimeList()
})
</script>

<style scoped>
.historical-recommendations {
  padding: 20px;
}

.loading-container,
.empty-state {
  text-align: center;
  padding: 40px;
  color: #666;
}

.spinner {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.recommendations-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(450px, 1fr));
  gap: 24px;
  margin-top: 24px;
}

@media (max-width: 768px) {
  .recommendations-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .stats-summary {
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 12px;
    padding: 16px;
  }

  .recommendation-card {
    padding: 16px;
  }

  .symbol-stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

.recommendation-card {
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 12px;
  padding: 24px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  position: relative;
  overflow: hidden;
}

.recommendation-card:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.recommendation-card.has-performance {
  border-left: 4px solid #3498db;
}

.recommendation-card.profit {
  border-left: 4px solid #28a745;
  background: linear-gradient(135deg, rgba(40, 167, 69, 0.02) 0%, rgba(40, 167, 69, 0.05) 100%);
}

.recommendation-card.loss {
  border-left: 4px solid #dc3545;
  background: linear-gradient(135deg, rgba(220, 53, 69, 0.02) 0%, rgba(220, 53, 69, 0.05) 100%);
}

.trading-strategy-section,
.technical-signal-section {
  margin-top: 20px;
  padding: 16px;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-radius: 8px;
  border: 1px solid #dee2e6;
}

.strategy-header h5,
.signal-header h5 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #495057;
  font-weight: 600;
}

.strategy-content,
.signal-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}

.strategy-item,
.signal-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.strategy-label,
.signal-label {
  font-size: 11px;
  color: #6c757d;
  font-weight: 500;
}

.strategy-value,
.signal-value {
  font-size: 13px;
  font-weight: 600;
  color: #495057;
}

.strategy-value.positive,
.signal-value.positive {
  color: #28a745;
}

.strategy-value.negative,
.signal-value.negative {
  color: #dc3545;
}

.strategy-value.neutral,
.signal-value.neutral {
  color: #6c757d;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 2px solid #f8f9fa;
  position: relative;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.rank-badge {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  font-size: 14px;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.symbol-info {
  flex: 1;
}

.symbol-info h4 {
  margin: 0 0 4px 0;
  font-size: 20px;
  color: #2c3e50;
  font-weight: 600;
}

.base-symbol {
  color: #7f8c8d;
  font-size: 13px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.score-badge {
  padding: 8px 14px;
  border-radius: 20px;
  font-weight: bold;
  font-size: 14px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.score-high {
  background: #2ecc71;
  color: white;
}

.score-medium {
  background: #f39c12;
  color: white;
}

.score-low {
  background: #e74c3c;
  color: white;
}

.status-badge {
  padding: 8px 14px;
  border-radius: 20px;
  font-weight: bold;
  font-size: 12px;
  white-space: nowrap;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.status-profit {
  background: #2ecc71;
  color: white;
}

.status-loss {
  background: #e74c3c;
  color: white;
}

.status-calculating {
  background: #f39c12;
  color: white;
  animation: pulse 2s infinite;
}

.status-neutral {
  background: #95a5a6;
  color: white;
}

.status-no-data {
  background: #bdc3c7;
  color: #7f8c8d;
}

.status-failed {
  background: #e74c3c;
  color: white;
}

.status-completed {
  background: #27ae60;
  color: white;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

.scores-section {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 8px;
  margin-bottom: 16px;
}

.score-item {
  text-align: center;
  padding: 8px;
  background: #f8f9fa;
  border-radius: 4px;
}

.score-label {
  display: block;
  font-size: 11px;
  color: #666;
  margin-bottom: 4px;
}

.score-value {
  display: block;
  font-weight: bold;
  font-size: 14px;
  color: #333;
}

.symbol-stats-section {
  margin-bottom: 16px;
  padding: 12px;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-radius: 8px;
  border: 1px solid #dee2e6;
}

.symbol-stats-header h5 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #495057;
  font-weight: 600;
  text-align: center;
}

.symbol-stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
  gap: 8px;
}

.symbol-stat-item {
  text-align: center;
  padding: 6px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 4px;
  border: 1px solid #e9ecef;
}

.symbol-stat-label {
  display: block;
  font-size: 10px;
  color: #6c757d;
  margin-bottom: 2px;
  font-weight: 500;
}

.symbol-stat-value {
  display: block;
  font-weight: bold;
  font-size: 12px;
  color: #495057;
}

.symbol-stat-value.positive {
  color: #28a745;
}

.symbol-stat-value.negative {
  color: #dc3545;
}

.performance-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 2px solid #f8f9fa;
  background: linear-gradient(135deg, #fdfdfe 0%, #f8f9fa 100%);
  border-radius: 8px;
  padding: 20px;
  margin: 20px -4px 0 -4px;
}

.performance-header h4 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 16px;
}

.performance-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.performance-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.perf-label {
  font-size: 12px;
  color: #666;
}

.perf-value {
  font-size: 16px;
  font-weight: bold;
}

.perf-value.positive {
  color: #2ecc71;
}

.perf-value.negative {
  color: #e74c3c;
}

.rating-excellent {
  color: #2ecc71;
}

.rating-good {
  color: #3498db;
}

.rating-average {
  color: #f39c12;
}

.rating-poor {
  color: #e74c3c;
}

.no-performance {
  text-align: center;
  padding: 20px;
  color: #999;
  font-style: italic;
}

.reasons-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.reasons-section h5 {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #333;
}

.reasons-section ul {
  margin: 0;
  padding-left: 20px;
  color: #666;
  font-size: 13px;
}

.reasons-section li {
  margin-bottom: 4px;
}

.error-panel {
  background: #fee;
  border-color: #e74c3c;
}

.error-message {
  text-align: center;
  padding: 20px;
}

.error-message p {
  margin: 0 0 12px 0;
  color: #e74c3c;
}

.info-text {
  color: #666;
  font-size: 14px;
}

.stats-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 16px;
  padding: 20px;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-radius: 12px;
  margin-bottom: 24px;
  border: 1px solid #dee2e6;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-label {
  color: #666;
  font-size: 14px;
}

.stat-value {
  font-size: 16px;
  font-weight: bold;
}

.stat-value.positive {
  color: #2ecc71;
}

.stat-value.negative {
  color: #e74c3c;
}

.recommendation-card.profit {
  border-left: 4px solid #2ecc71;
}

.recommendation-card.loss {
  border-left: 4px solid #e74c3c;
}
</style>

