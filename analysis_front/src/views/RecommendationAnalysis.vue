<template>
  <section class="panel">
    <div class="row">
      <h2>表现验证</h2>
      <div class="spacer"></div>
      <button @click="batchUpdate" :disabled="batchUpdating">
        {{ batchUpdating ? '更新中...' : '批量更新' }}
      </button>
      <button @click="executeStrategyBacktest" :disabled="strategyTesting" class="btn-secondary">
        {{ strategyTesting ? '策略回测中...' : '执行策略回测' }}
      </button>
      <button class="primary" @click="load">刷新</button>
    </div>
  </section>

  <!-- 统计概览 -->
  <section style="margin-top:12px;" class="panel" v-if="stats">
    <div class="row">
      <h3>📊 表现统计</h3>
      <div class="spacer"></div>
      <button @click="toggleStatsView" class="btn-secondary">
        <span v-if="statsViewMode === 'cards'">📈 图表视图</span>
        <span v-else>📋 卡片视图</span>
      </button>
    </div>

    <!-- 卡片视图 -->
    <div v-if="statsViewMode === 'cards'" class="stats-grid modern">
      <div class="stat-card primary">
        <div class="stat-icon">📈</div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.total || 0 }}</div>
          <div class="stat-label">总验证记录</div>
        </div>
      </div>
      <div class="stat-card success">
        <div class="stat-icon">✅</div>
        <div class="stat-content">
          <div class="stat-value">{{ stats.completed || 0 }}</div>
          <div class="stat-label">已完成验证</div>
        </div>
      </div>
      <div class="stat-card" :class="getPerformanceClass(stats.avg_strategy_return)">
        <div class="stat-icon">💰</div>
        <div class="stat-content">
          <div class="stat-value">{{ formatPercent(stats.avg_strategy_return) }}</div>
          <div class="stat-label">策略平均收益</div>
        </div>
      </div>
      <div class="stat-card success">
        <div class="stat-icon">🎯</div>
        <div class="stat-content">
          <div class="stat-value">{{ formatPercent(stats.strategy_win_rate) }}</div>
          <div class="stat-label">策略胜率</div>
        </div>
      </div>
      <div class="stat-card info">
        <div class="stat-icon">⏱️</div>
        <div class="stat-content">
          <div class="stat-value">{{ formatAvgHoldingTime(stats.avg_holding_period) }}</div>
          <div class="stat-label">平均持有时间</div>
        </div>
      </div>
      <div class="stat-card" :class="getPerformanceClass(stats.avg_strategy_return)">
        <div class="stat-icon">🎯</div>
        <div class="stat-content">
          <div class="stat-value">{{ formatPercent(stats.avg_strategy_return) }}</div>
          <div class="stat-label">策略平均收益</div>
        </div>
      </div>
      <div class="stat-card success">
        <div class="stat-icon">⚡</div>
        <div class="stat-content">
          <div class="stat-value">{{ formatPercent(stats.strategy_win_rate) }}</div>
          <div class="stat-label">策略胜率</div>
        </div>
      </div>
    </div>

    <!-- 图表视图 -->
    <div v-else class="stats-chart-container">
      <LineChart
        v-if="performanceTrendData.length > 0"
        :data="performanceTrendData"
        :options="performanceChartOptions"
        title="算法表现趋势"
      />
      <div v-else class="chart-placeholder">
        <p>暂无趋势数据</p>
      </div>
    </div>
  </section>

  <!-- 验证记录列表 -->
  <section style="margin-top:12px;" class="panel">
    <div class="row" style="align-items:flex-end; gap:12px; flex-wrap: wrap;">
      <h3>📋 验证记录</h3>
      <div class="spacer"></div>
      <button @click="exportData" class="btn-secondary">📥 导出数据</button>
    </div>

    <!-- 高级筛选 -->
    <div class="filters-section">
      <div class="filter-row">
        <label>
          <span class="filter-icon">🔍</span> 状态：
          <select v-model="filterStatus" @change="load">
            <option value="">全部状态</option>
            <option value="pending">⏳ 待处理</option>
            <option value="completed">✅ 已完成</option>
            <option value="failed">❌ 失败</option>
          </select>
        </label>
        <label>
          <span class="filter-icon">💰</span> 币种：
          <input v-model="filterSymbol" placeholder="输入币种代码，如 BTC" @input="debouncedLoad" />
        </label>
        <label>
          <span class="filter-icon">📅</span> 起始日期：
          <input type="date" v-model="filterDateRange.start" @change="load" />
        </label>
        <label>
          <span class="filter-icon">📅</span> 结束日期：
          <input type="date" v-model="filterDateRange.end" @change="load" />
        </label>
        <button @click="clearFilters" class="btn-secondary">清除筛选</button>
      </div>
    </div>
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>正在加载验证记录...</p>
      <div class="loading-progress">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: loadingProgress + '%' }"></div>
        </div>
        <span class="progress-text">{{ loadingProgress }}%</span>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorMessage" class="error-message">
      <p>{{ errorMessage }}</p>
      <button @click="errorMessage = ''; load()" class="btn-retry">重试</button>
    </div>
    <div v-else-if="records.length === 0" style="text-align:center; padding: 40px;">
      <p>暂无验证记录</p>
    </div>
    <div v-else class="table-container">
      <table class="data-table modern">
        <thead>
          <tr>
            <th @click="changeSort('base_symbol')" class="sortable">
              币种
              <span v-if="sortBy === 'base_symbol'" class="sort-icon">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
            </th>
            <th @click="changeSort('recommended_at')" class="sortable">
              推荐时间
              <span v-if="sortBy === 'recommended_at'" class="sort-icon">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
            </th>
            <th>推荐价格</th>
            <th @click="changeSort('total_score')" class="sortable">
              推荐得分
              <span v-if="sortBy === 'total_score'" class="sort-icon">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
            </th>
            <th>最大涨幅</th>
            <th>最大回撤</th>
            <th>策略收益</th>
            <th>持有时间</th>
            <th>退出原因</th>
            <th>验证状态</th>
            <th>操作</th>
          </tr>
        </thead>
      <tbody>
        <tr v-for="record in records" :key="record.id" class="table-row">
          <td>
            <div class="coin-info">
              <strong>{{ record.base_symbol }}</strong>
              <small>{{ record.symbol }}</small>
            </div>
          </td>
          <td>{{ formatTime(record.recommended_at) }}</td>
          <td>{{ formatPrice(record.recommended_price) }}</td>
          <td>
            <span v-if="record.total_score" class="score-badge" :class="getScoreClass(record.total_score)">
              {{ record.total_score.toFixed(1) }}
            </span>
            <span v-else class="no-data">-</span>
          </td>
          <td>
            <span v-if="record.max_gain" class="positive">
              {{ formatPercent(record.max_gain) }}
            </span>
            <span v-else class="no-data">-</span>
          </td>
          <td>
            <span v-if="record.max_drawdown" class="negative">
              {{ formatPercent(record.max_drawdown) }}
            </span>
            <span v-else class="no-data">-</span>
          </td>
          <td>
            <span v-if="record.actual_return !== undefined && record.actual_return !== null" :class="getPerformanceClass(record.actual_return)">
              {{ formatPercent(record.actual_return) }}
            </span>
            <span v-else class="no-data strategy-hint" title="点击右侧的🧪按钮执行策略回测">待执行</span>
          </td>
          <td>
            <span v-if="record.holding_period" class="info-text">
              {{ formatHoldingPeriod(record.holding_period) }}
            </span>
            <span v-else class="no-data">-</span>
          </td>
          <td>
            <span v-if="record.exit_reason" class="reason-badge">
              {{ getExitReasonText(record.exit_reason) }}
            </span>
            <span v-else class="no-data">-</span>
          </td>
          <td>
            <span :class="['status-badge', `status-${record.status}`]">
              {{ getStatusText(record.status) }}
            </span>
          </td>
          <td>
              <div class="action-buttons">
                <button class="btn-icon" @click="viewChart(record)" title="查看图表">
                  📊
                </button>
                <button v-if="record.status === 'pending'" class="btn-icon" @click="updateRecord(record.id)" title="更新验证">
                  🔄
                </button>
                <button class="btn-icon" @click="testSingleStrategy(record)" title="测试策略">
                  🧪
                </button>
                <button class="btn-icon" @click="viewDetails(record)" title="查看详情">
                  👁️
                </button>
                <button class="btn-icon" @click="generateReport(record)" title="生成报告">
                  📊
                </button>
              </div>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- 分页组件 -->
    <div v-if="totalRecords > pageSize" class="pagination">
      <button
        @click="changePage(currentPage - 1)"
        :disabled="currentPage <= 1"
        class="page-btn"
      >
        上一页
      </button>

      <span class="page-info">
        第 {{ currentPage }} 页，共 {{ Math.ceil(totalRecords / pageSize) }} 页
        (共 {{ totalRecords }} 条记录)
      </span>

      <button
        @click="changePage(currentPage + 1)"
        :disabled="currentPage >= Math.ceil(totalRecords / pageSize)"
        class="page-btn"
      >
        下一页
      </button>
    </div>
    </div>
  </section>

  <!-- K线图模态框 -->
  <div v-if="selectedRecord" class="modal-overlay" @click="selectedRecord = null">
    <div class="modal-content large" @click.stop>
      <div class="row">
        <h3>{{ selectedRecord.base_symbol }} 回测图表</h3>
        <div class="spacer"></div>
        <button @click="selectedRecord = null">关闭</button>
      </div>
      <div v-if="chartLoading" class="chart-placeholder">
        <p>加载中...</p>
      </div>
      <CandlestickChart
        v-else-if="chartData.length > 0"
        :kline-data="chartData"
        :buy-points="[{
          timestamp: new Date(selectedRecord.recommended_at).getTime(),
          price: parseFloat(selectedRecord.recommended_price),
          label: '推荐买入'
        }]"
        :sell-points="getSellPoints(selectedRecord)"
        :title="`${selectedRecord.base_symbol} 回测分析`"
      />
      <div v-else class="chart-placeholder">
        <p>暂无图表数据</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 现代化统计卡片 */
.stats-grid.modern {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border: 1px solid #e1e5e9;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.stat-card.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
}

.stat-card.success {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: white;
  border: none;
}

.stat-card.positive {
  border-left: 4px solid #10b981;
}

.stat-card.negative {
  border-left: 4px solid #ef4444;
}

.stat-card.info {
  border-left: 4px solid #3b82f6;
}

.stat-icon {
  font-size: 32px;
  opacity: 0.9;
}

.stat-content {
  flex: 1;
}

.stat-content .stat-value {
  font-size: 28px;
  font-weight: bold;
  margin-bottom: 4px;
}

.stat-content .stat-label {
  font-size: 14px;
  opacity: 0.9;
}

/* 筛选区域 */
.filters-section {
  background: #f8fafc;
  border-radius: 8px;
  padding: 16px;
  margin: 16px 0;
  border: 1px solid #e2e8f0;
}

.filter-row {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}

.filter-row label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.filter-icon {
  font-size: 16px;
}

.filter-row input,
.filter-row select {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  min-width: 120px;
}

/* 现代化表格 */
.table-container {
  overflow-x: auto;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: white;
}

.data-table.modern {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.data-table.modern th {
  background: #f9fafb;
  padding: 16px 12px;
  text-align: left;
  font-weight: 600;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
  position: sticky;
  top: 0;
  z-index: 10;
}

.data-table.modern th.sortable {
  cursor: pointer;
  user-select: none;
  transition: background-color 0.2s;
}

.data-table.modern th.sortable:hover {
  background: #f3f4f6;
}

.sort-icon {
  margin-left: 4px;
  font-size: 12px;
}

.data-table.modern td {
  padding: 16px 12px;
  border-bottom: 1px solid #f3f4f6;
  vertical-align: top;
}

.table-row:hover {
  background: #f9fafb;
}

.coin-info strong {
  font-size: 16px;
  color: #1f2937;
}

.coin-info small {
  color: #6b7280;
  font-size: 12px;
}

/* 评分徽章 */
.score-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.score-badge.excellent {
  background: #dcfce7;
  color: #166534;
}

.score-badge.good {
  background: #dbeafe;
  color: #1e40af;
}

.score-badge.average {
  background: #fef3c7;
  color: #92400e;
}

.score-badge.poor {
  background: #fee2e2;
  color: #991b1b;
}

/* 状态徽章 */
.status-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-completed {
  background: #dcfce7;
  color: #166534;
}

.status-failed {
  background: #fee2e2;
  color: #991b1b;
}

.status-tracking {
  background: #fef3c7;
  color: #92400e;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
}

.btn-icon {
  padding: 6px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  background: white;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
}

/* 分页组件 */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 20px;
  padding: 16px;
}

.page-btn {
  padding: 8px 16px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  background: #f3f4f6;
  border-color: #9ca3af;
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: 14px;
  color: #6b7280;
}

/* 图表容器 */
.stats-chart-container {
  margin-top: 20px;
  height: 400px;
  position: relative;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .mobile-hide {
    display: none;
  }

  .filter-row {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-row label {
    justify-content: space-between;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-content .stat-value {
    font-size: 24px;
  }

  .table-container {
    font-size: 12px;
  }

  .data-table.modern th,
  .data-table.modern td {
    padding: 8px 6px;
  }
}

/* 加载状态样式 */
.loading-container {
  text-align: center;
  padding: 60px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f4f6;
  border-top: 4px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-progress {
  width: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: #f3f4f6;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #10b981);
  transition: width 0.3s ease;
  border-radius: 4px;
}

.progress-text {
  font-size: 12px;
  color: #6b7280;
  font-weight: 500;
}

.btn-retry {
  background: #dc2626;
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  margin-top: 8px;
}

.btn-retry:hover {
  background: #b91c1c;
}

/* 错误提示样式 */
.error-message {
  background: #fef2f2;
  color: #dc2626;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #fecaca;
  margin: 16px 0;
  text-align: center;
}

/* 无数据样式 */
.no-data {
  color: #9ca3af;
  font-style: italic;
}

.strategy-hint {
  color: #f59e0b;
  font-style: italic;
  cursor: help;
}

/* 性能类 */
.positive {
  color: #10b981;
  font-weight: 600;
}

.negative {
  color: #ef4444;
  font-weight: 600;
}

/* 信息文本 */
.info-text {
  color: #3b82f6;
  font-size: 12px;
}

/* 退出原因徽章 */
.reason-badge {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 500;
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}
</style>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { api } from '../api/api.js'
import CandlestickChart from '../components/CandlestickChart.vue'
import LineChart from '../components/LineChart.vue'

const stats = ref(null)
const records = ref([])
const loading = ref(false)
const loadingProgress = ref(0)
const batchUpdating = ref(false)
const strategyTesting = ref(false)
const filterStatus = ref('')
const filterSymbol = ref('')
const selectedRecord = ref(null)
const chartData = ref([])
const chartLoading = ref(false)
const statsViewMode = ref('cards') // 'cards' or 'chart'
const performanceTrendData = ref([])
const currentPage = ref(1)
const pageSize = ref(20)
const totalRecords = ref(0)
const sortBy = ref('recommended_at')
const sortOrder = ref('desc')
const filterDateRange = ref({ start: '', end: '' })
const errorMessage = ref('')

let loadTimeout = null
function debouncedLoad() {
  if (loadTimeout) clearTimeout(loadTimeout)
  loadTimeout = setTimeout(() => load(), 500)
}

function toggleStatsView() {
  statsViewMode.value = statsViewMode.value === 'cards' ? 'chart' : 'cards'
  if (statsViewMode.value === 'chart' && performanceTrendData.value.length === 0) {
    loadPerformanceTrend()
  }
}

async function loadPerformanceTrend() {
  try {
    const trendData = await api.getPerformanceTrend({ days: 30 })
    performanceTrendData.value = trendData.map(item => ({
      date: new Date(item.date).toLocaleDateString('zh-CN'),
      '策略平均收益': item.avg_strategy_return,
      '策略胜率': item.strategy_win_rate,
      '平均持有时间(小时)': (item.avg_holding_period || 0) / 60
    }))
  } catch (error) {
    console.error('加载表现趋势失败:', error)
  }
}

function formatTime(timeStr) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

function formatPrice(price) {
  if (!price) return '-'
  const p = parseFloat(price)
  return p.toFixed(8)
}

function formatPercent(value) {
  if (value === null || value === undefined) return '-'
  return (value >= 0 ? '+' : '') + value.toFixed(2) + '%'
}

function getPerformanceClass(value) {
  if (value === null || value === undefined) return ''
  return value >= 0 ? 'positive' : 'negative'
}

function getStatusText(status) {
  const map = {
    pending: '待处理',
    completed: '已完成',
    failed: '失败',
    tracking: '追踪中'
  }
  return map[status] || status
}

function getSellPoints(record) {
  const points = []
  const recommendedAt = new Date(record.recommended_at).getTime()

  if (record.price_after_24h) {
    points.push({
      timestamp: recommendedAt + 24 * 3600 * 1000,
      price: parseFloat(record.price_after_24h),
      label: '24h后'
    })
  }

  return points
}

// 图表配置
const performanceChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: {
      beginAtZero: false,
      ticks: {
        callback: function(value) {
          return value + '%'
        }
      }
    }
  },
  plugins: {
    legend: {
      position: 'top',
    },
    tooltip: {
      callbacks: {
        label: function(context) {
          return context.dataset.label + ': ' + context.parsed.y.toFixed(2) + '%'
        }
      }
    }
  }
}))

async function load() {
  loading.value = true
  loadingProgress.value = 0
  errorMessage.value = ''

  try {
    // 模拟加载进度
    const progressInterval = setInterval(() => {
      if (loadingProgress.value < 90) {
        loadingProgress.value += 10
      }
    }, 200)

    const params = {
      page: currentPage.value,
      limit: pageSize.value,
      sort_by: sortBy.value,
      sort_order: sortOrder.value
    }
    if (filterStatus.value) params.status = filterStatus.value
    if (filterSymbol.value) params.symbol = filterSymbol.value
    if (filterDateRange.value.start) params.start_date = filterDateRange.value.start
    if (filterDateRange.value.end) params.end_date = filterDateRange.value.end

    loadingProgress.value = 30

    const [statsRes, recordsRes] = await Promise.all([
      api.getBacktestStats(),
      api.getBacktestRecords(params)
    ])

    loadingProgress.value = 80

    stats.value = statsRes
    records.value = recordsRes.records || []
    totalRecords.value = recordsRes.total || 0

    loadingProgress.value = 100
    clearInterval(progressInterval)

    // 短暂延迟以显示100%状态
    setTimeout(() => {
      loading.value = false
    }, 300)

  } catch (error) {
    console.error('加载验证数据失败:', error)
    errorMessage.value = error.message || '加载失败，请稍后重试'
    loading.value = false
    loadingProgress.value = 0
  }
}

function changePage(page) {
  currentPage.value = page
  load()
}

function changeSort(field) {
  if (sortBy.value === field) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = field
    sortOrder.value = 'desc'
  }
  load()
}

async function testSingleStrategy(record) {
  if (confirm(`确定要测试记录 ${record.base_symbol} (${record.symbol}) 的策略回测吗？`)) {
    try {
      const response = await api.testStrategyBacktest({ performance_id: record.id })

      // 等待数据保存完成
      await new Promise(resolve => setTimeout(resolve, 2000))

      // 临时清除所有筛选以确保获取所有数据
      const originalFilters = {
        status: filterStatus.value,
        symbol: filterSymbol.value,
        startDate: filterDateRange.value.start,
        endDate: filterDateRange.value.end
      }

      filterStatus.value = ''
      filterSymbol.value = ''
      filterDateRange.value = { start: '', end: '' }

      await load() // 刷新数据

      // 恢复原始筛选条件
      filterStatus.value = originalFilters.status
      filterSymbol.value = originalFilters.symbol
      filterDateRange.value.start = originalFilters.startDate
      filterDateRange.value.end = originalFilters.endDate

      // 检查特定记录是否更新
      const updatedRecord = records.value.find(r => r.id === record.id)
      if (updatedRecord && updatedRecord.actual_return !== undefined && updatedRecord.actual_return !== null) {
        // 找到记录，显示成功提示
        alert(`策略测试完成！\n记录ID: ${record.id}\n收益: ${updatedRecord.actual_return}%\n持有时间: ${updatedRecord.holding_period}分钟\n退出原因: ${updatedRecord.exit_reason}`)
      } else {
        // 尝试通过ID排序查找
        if (totalRecords.value > pageSize.value) {
          const originalSort = sortBy.value
          sortBy.value = 'id'
          sortOrder.value = 'desc'

          await load()

          sortBy.value = originalSort
          sortOrder.value = 'desc'

          const foundAfterSort = records.value.find(r => r.id === record.id)
          if (foundAfterSort && foundAfterSort.actual_return !== undefined && foundAfterSort.actual_return !== null) {
            alert(`策略测试完成！\n记录ID: ${record.id}\n收益: ${foundAfterSort.actual_return}%\n持有时间: ${foundAfterSort.holding_period}分钟\n退出原因: ${foundAfterSort.exit_reason}`)
          } else {
            alert('策略测试已执行，但未能在当前查询结果中找到记录。请手动刷新页面查看结果。')
          }
        } else {
          alert('策略测试已执行，但记录可能在其他页面。请手动刷新页面查看结果。')
        }
      }
    } catch (error) {
      console.error('策略测试失败:', error)
      alert('策略测试失败: ' + (error.message || '未知错误'))
    }
  }
}

function clearFilters() {
  filterStatus.value = ''
  filterSymbol.value = ''
  filterDateRange.value = { start: '', end: '' }
  currentPage.value = 1
  load()
}

async function executeStrategyBacktest() {
  if (confirm('确定要执行策略回测吗？这可能需要一些时间。')) {
    strategyTesting.value = true
    try {
      const response = await api.batchExecuteStrategyBacktest({ limit: 10 })
      alert(`策略回测完成: 处理 ${response.processed} 条, 成功 ${response.success} 条, 失败 ${response.failed} 条`)
      await load() // 刷新数据
    } catch (error) {
      console.error('策略回测失败:', error)
      alert('策略回测失败: ' + (error.message || '未知错误'))
    } finally {
      strategyTesting.value = false
    }
  }
}

function exportData() {
  const csvContent = generateCSV(records.value)
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `表现验证记录_${new Date().toISOString().split('T')[0]}.csv`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function viewDetails(record) {
  selectedRecord.value = record
  // 这里可以显示详细的记录信息模态框
  alert(`查看详情: ${record.base_symbol} (${record.symbol})`)
}

async function generateReport(record) {
  try {
    const response = await api.generateBacktestReport({
      performance_id: record.id,
      report_type: 'detailed'
    })

    if (response.success) {
      showReportModal(response.report)
    } else {
      alert('生成报告失败: ' + (response.error || '未知错误'))
    }
  } catch (error) {
    console.error('生成报告失败:', error)
    alert('生成报告失败: ' + (error.message || '未知错误'))
  }
}

function showReportModal(report) {
  const reportText = formatReportText(report)
  // 这里可以显示模态框展示报告
  // 暂时使用alert显示简要信息
  alert(`回测报告已生成:\n币种: ${report.symbol}\n当前收益率: ${report.basic_info.current_return?.toFixed(2) || 'N/A'}%\n状态: ${report.rating.status}`)
}

function formatReportText(report) {
  let text = `=== 表现验证报告 ===\n\n`
  text += `币种: ${report.symbol}\n`
  text += `推荐时间: ${new Date(report.basic_info.recommended_at).toLocaleString('zh-CN')}\n`
  text += `推荐价格: $${report.basic_info.recommended_price}\n`
  text += `当前价格: $${report.basic_info.current_price || 'N/A'}\n`
  text += `当前收益率: ${report.basic_info.current_return?.toFixed(2) || 'N/A'}%\n\n`

  if (report.historical_performance) {
    text += `历史表现:\n`
    if (report.historical_performance.return_24h !== undefined) {
      text += `  24h收益率: ${report.historical_performance.return_24h.toFixed(2)}%\n`
    }
    if (report.historical_performance.return_7d !== undefined) {
      text += `  7天收益率: ${report.historical_performance.return_7d.toFixed(2)}%\n`
    }
    if (report.historical_performance.return_30d !== undefined) {
      text += `  30天收益率: ${report.historical_performance.return_30d.toFixed(2)}%\n`
    }
  }

  text += `\n状态: ${report.rating.status}\n`
  text += `回测状态: ${report.rating.backtest_status}\n`

  return text
}

function getScoreClass(score) {
  if (!score) return ''
  if (score >= 80) return 'excellent'
  if (score >= 60) return 'good'
  if (score >= 40) return 'average'
  return 'poor'
}

function formatHoldingPeriod(minutes) {
  if (!minutes) return '-'
  if (minutes < 60) {
    return `${minutes}分钟`
  }
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  if (remainingMinutes === 0) {
    return `${hours}小时`
  }
  return `${hours}小时${remainingMinutes}分钟`
}

function formatAvgHoldingTime(avgMinutes) {
  if (!avgMinutes || avgMinutes <= 0) return '-'

  // 对于平均值，直接转换为小时显示
  const hours = avgMinutes / 60
  if (hours < 1) {
    return `${avgMinutes.toFixed(0)}分钟`
  } else if (hours < 24) {
    return `${hours.toFixed(1)}小时`
  } else {
    const days = hours / 24
    return `${days.toFixed(1)}天`
  }
}

function getExitReasonText(reason) {
  const reasonMap = {
    profit: '止盈',
    loss: '止损',
    time: '时间限制',
    max_hold: '最大持仓',
    force: '强制退出'
  }
  return reasonMap[reason] || reason
}

function generateCSV(records) {
  const headers = ['币种', '推荐时间', '推荐价格', '推荐得分', '最大涨幅', '最大回撤', '策略收益', '持有时间', '退出原因', '状态']
  const rows = records.map(record => [
    record.base_symbol,
    formatTime(record.recommended_at),
    record.recommended_price || '',
    record.total_score || '',
    formatPercent(record.max_gain),
    formatPercent(record.max_drawdown),
    formatPercent(record.actual_return),
    formatHoldingPeriod(record.holding_period),
    getExitReasonText(record.exit_reason),
    getStatusText(record.status)
  ])

  const csv = [headers, ...rows]
    .map(row => row.map(field => `"${field}"`).join(','))
    .join('\n')

  return '\ufeff' + csv // 添加 BOM 以支持中文
}

async function updateRecord(id) {
  try {
    await api.updateBacktestRecord(id)
    alert('验证更新已启动，请稍后刷新查看结果')
    setTimeout(() => load(), 2000)
  } catch (error) {
    console.error('更新验证记录失败:', error)
    alert('更新失败: ' + (error.message || '未知错误'))
  }
}

async function batchUpdate() {
  if (batchUpdating.value) return
  batchUpdating.value = true
  try {
    const res = await api.batchUpdateBacktestRecords({})
    alert(res.message || '批量更新已启动')
    setTimeout(() => load(), 2000)
  } catch (error) {
    console.error('批量更新失败:', error)
    alert('批量更新失败: ' + (error.message || '未知错误'))
  } finally {
    batchUpdating.value = false
  }
}

async function viewChart(record) {
  selectedRecord.value = record
  chartLoading.value = true
  chartData.value = []
  
  try {
    // 优先使用base_symbol（基础币种），如果没有则从symbol中提取
    let symbol = record.base_symbol || record.symbol
    
    // 如果symbol是交易对格式（如BTCUSDT），提取基础币种
    if (symbol && symbol.length > 4 && !symbol.includes('-')) {
      // 尝试提取常见的基础币种（BTC, ETH, SOL等）
      const commonSymbols = ['BTC', 'ETH', 'BNB', 'SOL', 'MATIC', 'AVAX', 'FTM', 'USDT', 'USDC', 'DAI']
      for (const base of commonSymbols) {
        if (symbol.startsWith(base)) {
          symbol = base
          break
        }
      }
    }
    
    if (!symbol) {
      throw new Error('无法确定币种符号')
    }
    
    const days = Math.ceil((new Date() - new Date(record.recommended_at)) / (1000 * 60 * 60 * 24)) + 1
    const res = await api.getMarketPriceHistory({ symbol, days: Math.min(days, 30) })
    
    if (!res || !res.prices || res.prices.length === 0) {
      throw new Error('未获取到价格数据')
    }
    
    chartData.value = res.prices || []
  } catch (error) {
    console.error('加载图表数据失败:', error)
    
    // 提取错误信息
    const errorMsg = error.data?.error || error.data?.message || error.message || '未知错误'
    const status = error.status || error.response?.status
    
    // 如果是404或资源不存在，提供更友好的提示
    if (status === 404 || errorMsg.includes('不存在') || errorMsg.includes('不支持') || errorMsg.includes('未配置')) {
      const symbolDisplay = record.base_symbol || record.symbol || '未知'
      const message = `加载图表失败：币种 ${symbolDisplay} 不在价格服务支持列表中。\n\n` +
        `系统已尝试自动查找，但未找到该币种的价格数据。\n\n` +
        `可能的原因：\n` +
        `1. 币种符号不正确\n` +
        `2. CoinGecko API中不存在该币种\n` +
        `3. 网络连接问题\n\n` +
        `建议：\n` +
        `- 检查币种符号是否正确\n` +
        `- 稍后重试\n` +
        `- 如需手动配置，请在配置文件的 pricing.map 中添加映射`
      alert(message)
    } else {
      alert('加载图表失败: ' + errorMsg)
    }
    
    // 清空数据，避免显示错误的图表
    chartData.value = []
  } finally {
    chartLoading.value = false
  }
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.stat-card {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  text-align: center;
}

.stat-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #333;
}

.stat-value.positive {
  color: #10b981;
}

.stat-value.negative {
  color: #ef4444;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 16px;
}

.data-table th,
.data-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #e0e0e0;
}

.data-table th {
  background: #f8f9fa;
  font-weight: bold;
}

.positive {
  color: #10b981;
  font-weight: bold;
}

.negative {
  color: #ef4444;
  font-weight: bold;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-completed {
  background: #d1fae5;
  color: #065f46;
}

.status-failed {
  background: #fee2e2;
  color: #991b1b;
}

.btn-small {
  padding: 4px 8px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid #ddd;
  background: #fff;
  cursor: pointer;
}

.btn-small:hover {
  background: #f5f5f5;
}

.modal-overlay {
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

.modal-content {
  background: #fff;
  padding: 24px;
  border-radius: 8px;
  min-width: 400px;
  max-width: 90%;
}

.modal-content.large {
  min-width: 800px;
  max-width: 95%;
}

.chart-placeholder {
  text-align: center;
  padding: 40px;
  color: #666;
}
</style>

