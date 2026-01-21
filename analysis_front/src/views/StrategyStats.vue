<template>
  <div class="strategy-stats-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-main">
        <div class="title-section">
          <h1 class="page-title">策略运行统计</h1>
          <div class="strategy-info">
            <span class="strategy-name">{{ strategy?.name }}</span>
            <span class="strategy-id">ID: {{ strategyId }}</span>
          </div>
        </div>
        <div class="header-actions">
          <button class="btn btn-outline" @click="goBack">
            返回策略列表
          </button>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading">
      <div class="loading-spinner"></div>
      <p>加载统计数据中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error-message">
      <h3>❌ 加载失败</h3>
      <p>{{ error }}</p>
      <button class="btn btn-primary" @click="loadStats">重试</button>
    </div>

    <!-- 统计内容 -->
    <div v-else-if="stats" class="stats-content">
      <!-- 总体统计卡片 -->
      <div class="stats-overview">
        <div class="stat-card">
          <div class="stat-value">{{ stats.total_executions }}</div>
          <div class="stat-label">总执行次数</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.total_orders }}</div>
          <div class="stat-label">总订单数</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.success_orders }}</div>
          <div class="stat-label">成功订单</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.total_pnl?.toFixed(2) || '0.00' }}</div>
          <div class="stat-label">总盈亏</div>
          <div class="stat-trend" :class="stats.total_pnl >= 0 ? 'profit' : 'loss'">
            {{ stats.total_pnl >= 0 ? '↑' : '↓' }}
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.total_pnl_percentage?.toFixed(2) || '0.00' }}%</div>
          <div class="stat-label">盈亏百分比</div>
          <div class="stat-trend" :class="stats.total_pnl_percentage >= 0 ? 'profit' : 'loss'">
            {{ stats.total_pnl_percentage >= 0 ? '↑' : '↓' }}
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.total_investment?.toFixed(2) || '0.00' }}</div>
          <div class="stat-label">买入总金额</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.current_value?.toFixed(2) || '0.00' }}</div>
          <div class="stat-label">当前资产价值</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ stats.avg_win_rate?.toFixed(1) || '0.0' }}%</div>
          <div class="stat-label">平均胜率</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ ((stats.success_orders / Math.max(stats.total_orders, 1)) * 100).toFixed(1) }}%</div>
          <div class="stat-label">整体胜率</div>
        </div>
      </div>

      <!-- 图表区域 -->
      <div class="charts-section">
        <div class="chart-container">
          <h3>盈亏走势</h3>
          <div class="pnl-chart">
            <canvas ref="pnlChart"></canvas>
          </div>
        </div>

        <div class="chart-container">
          <h3>胜率趋势</h3>
          <div class="winrate-chart">
            <canvas ref="winrateChart"></canvas>
          </div>
        </div>
      </div>

      <!-- 当前执行状态 -->
      <div v-if="currentExecution" class="current-execution-section">
        <h3>{{ currentExecution.status === 'running' ? '当前执行状态' : '最新执行详情' }}</h3>
        <div class="current-execution-card">
          <div class="execution-info">
            <div class="execution-status">
              <span class="status-indicator running"></span>
              <span class="status-text">{{ getExecutionStatusText(currentExecution.status) }}</span>
            </div>
            <div class="execution-details">
              <div class="detail-item">
                <span class="label">当前步骤:</span>
                <span class="value">{{ currentExecution.current_step }}</span>
              </div>
              <div class="detail-item">
                <span class="label">处理交易对:</span>
                <span class="value">{{ currentExecution.current_symbol || '无' }}</span>
              </div>
              <div class="detail-item">
                <span class="label">步骤进度:</span>
                <div class="progress-bar">
                  <div class="progress-fill" :style="{ width: currentExecution.step_progress + '%' }"></div>
                  <span class="progress-text">{{ currentExecution.step_progress }}%</span>
                </div>
              </div>
              <div class="detail-item">
                <span class="label">总体进度:</span>
                <div class="progress-bar">
                  <div class="progress-fill" :style="{ width: currentExecution.total_progress + '%' }"></div>
                  <span class="progress-text">{{ currentExecution.total_progress }}%</span>
                </div>
              </div>
            </div>
          </div>
          <div v-if="currentExecution.error_message" class="error-message">
            <strong>错误:</strong> {{ currentExecution.error_message }}
          </div>
        </div>
      </div>

      <!-- 执行步骤详情 -->
      <div v-if="showExecutionSteps && executionSteps && executionSteps.length > 0" class="steps-section">
        <div class="section-header">
          <h3>执行步骤详情</h3>
          <button class="btn btn-small" @click="hideExecutionSteps">
            隐藏详情
          </button>
        </div>
        <div class="steps-timeline">
          <div
            v-for="step in executionSteps"
            :key="step.id"
            class="step-item"
            :class="{ active: step.status === 'running', completed: step.status === 'completed', failed: step.status === 'failed' }"
          >
            <div class="step-header">
              <div class="step-status">
                <span class="status-icon" :class="step.status"></span>
                <span class="step-name">{{ step.step_name }}</span>
              </div>
              <div class="step-time">
                {{ formatDateTime(step.start_time) }}
                <span v-if="step.end_time"> - {{ formatDateTime(step.end_time) }}</span>
              </div>
            </div>
            <div class="step-details">
              <div v-if="step.symbol" class="step-symbol">交易对: {{ step.symbol }}</div>
              <div v-if="step.result" class="step-result">{{ step.result }}</div>
              <div v-if="step.error_message" class="step-error">错误: {{ step.error_message }}</div>
              <div v-if="step.data" class="step-data">
                <details>
                  <summary>详细信息</summary>
                  <pre>{{ step.data }}</pre>
                </details>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 执行记录表格 -->
      <div class="executions-section">
        <h3>执行记录详情</h3>

        <!-- 分页信息 -->
        <div v-if="stats?.pagination" class="pagination-info">
          <span>执行记录：第 {{ stats.pagination.page }} / {{ stats.pagination.total_pages }} 页，共 {{ stats.pagination.total_records }} 条记录</span>
        </div>

        <div class="executions-table-container">
          <table class="executions-table">
            <thead>
              <tr>
                <th>执行时间</th>
                <th>状态</th>
                <th>持续时间</th>
                <th>订单数</th>
                <th>成功订单</th>
                <th>失败订单</th>
                <th>盈亏</th>
                <th>胜率</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="execution in stats.executions" :key="execution.id">
                <td>{{ formatDateTime(execution.start_time) }}</td>
                <td>
                  <span class="status-badge" :class="execution.status">
                    {{ getExecutionStatusText(execution.status) }}
                  </span>
                </td>
                <td>{{ formatDuration(execution.duration) }}</td>
                <td>{{ execution.total_orders }}</td>
                <td class="success-count">{{ execution.success_orders }}</td>
                <td class="fail-count">{{ execution.failed_orders }}</td>
                <td :class="execution.total_pnl >= 0 ? 'profit' : 'loss'">
                  {{ execution.total_pnl?.toFixed(2) || '0.00' }}
                </td>
                <td>{{ execution.win_rate?.toFixed(1) || '0.0' }}%</td>
                <td class="action-cell">
                  <div class="action-buttons">
                    <button class="btn btn-small" @click="viewExecutionDetail(execution.id)">
                      详情
                    </button>
                    <button class="btn btn-small btn-danger" @click="confirmDeleteExecution(execution.id)">
                      删除
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 分页组件 -->
        <Pagination
          v-if="stats?.pagination"
          v-model:page="currentPage"
          v-model:pageSize="pageSize"
          :total="stats.pagination.total_records"
          :totalPages="stats.pagination.total_pages"
          :loading="loading"
          @change="onPaginationChange"
        />
      </div>

      <!-- 策略订单记录 -->
      <div class="strategy-orders-section">
        <h3>策略订单记录</h3>

        <!-- 订单分页信息 -->
        <div v-if="strategyOrders?.pagination" class="pagination-info">
          <span>订单记录：第 {{ strategyOrders.pagination.page }} / {{ strategyOrders.pagination.total_pages }} 页，共 {{ strategyOrders.pagination.total_records }} 条记录</span>
        </div>

        <div class="orders-table-container">
          <table class="orders-table">
            <thead>
              <tr>
                <th>创建时间</th>
                <th>交易对</th>
                <th>操作类型</th>
                <th>订单类型</th>
                <th>成交数量</th>
                <th>成交价</th>
                <th>杠杆</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="order in strategyOrders?.orders || []" :key="order.id">
                <td>{{ formatDateTime(order.created_at) }}</td>
                <td>{{ order.symbol }}</td>
                <td>
                  <span class="operation-badge" :class="order.operation_type?.class || getOperationClass(order.side, order.reduce_only)">
                    {{ order.operation_type?.type || '未知' }}
                  </span>
                </td>
                <td>{{ order.order_type === 'MARKET' ? '市价' : '限价' }}</td>
                <td>{{ order.executed_quantity || '-' }}</td>
                <td>{{ (order.avg_price || order.price) ? (parseFloat(order.avg_price || order.price)).toFixed(5) : '-' }}</td>
                <td>{{ order.leverage > 0 ? `${order.leverage}x` : '-' }}</td>
                <td>
                  <span class="status-badge" :class="order.status">
                    {{ getOrderStatusText(order.status) }}
                  </span>
                </td>
                <td class="action-cell">
                  <div class="action-buttons">
                    <button class="btn btn-small" @click="viewOrderDetail(order.id)">
                      详情
                    </button>
                    <button class="btn btn-small btn-danger" @click="confirmDeleteOrder(order.id)">
                      删除
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 订单分页组件 -->
        <Pagination
          v-if="strategyOrders?.pagination"
          v-model:page="ordersCurrentPage"
          v-model:pageSize="ordersPageSize"
          :total="strategyOrders.pagination.total_records"
          :totalPages="strategyOrders.pagination.total_pages"
          :loading="loadingOrders"
          @change="onOrdersPaginationChange"
        />
      </div>

      <!-- 执行日志 -->
      <div class="logs-section">
        <h3>执行日志 {{ currentExecution ? `(执行ID: ${currentExecution.id})` : '' }}</h3>
        <div class="logs-container">
          <div v-if="currentExecution && currentExecution.logs" class="logs-content">
            <pre>{{ currentExecution.logs }}</pre>
          </div>
          <div v-else-if="currentExecution && !currentExecution.logs" class="no-logs">
            此执行暂无日志内容
          </div>
          <div v-else class="no-logs">
            暂无执行记录
          </div>
        </div>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="showDeleteDialog" class="modal-overlay" @click="cancelDelete">
      <div class="modal-content" @click.stop>
        <h3>{{ deleteType === 'confirm' ? '确认删除' : '删除失败' }}</h3>
        <p>{{ deleteMessage }}</p>
        <div class="modal-actions">
          <button v-if="deleteType === 'confirm'" class="btn btn-danger" @click="performDelete">确认删除</button>
          <button class="btn" @click="cancelDelete">
            {{ deleteType === 'confirm' ? '取消' : '关闭' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/api.js'
import Chart from 'chart.js/auto'
import Pagination from '../components/Pagination.vue'

const route = useRoute()
const router = useRouter()

const strategyId = ref(route.params.id)
const loading = ref(true)
const error = ref('')
const stats = ref(null)
const strategy = ref(null)
const currentExecution = ref(null)
const executionSteps = ref([])
const showExecutionSteps = ref(false) // 控制执行步骤详情的显示
const refreshInterval = ref(null)

// 分页相关
const currentPage = ref(1)
const pageSize = ref(5) // 默认5条记录

// 订单分页相关
const ordersCurrentPage = ref(1)
const ordersPageSize = ref(10) // 默认10条记录
const strategyOrders = ref(null)
const loadingOrders = ref(false)

// 图表引用
const pnlChart = ref(null)
const winrateChart = ref(null)

// 图表实例
let pnlChartInstance = null
let winrateChartInstance = null

// 跟踪数据变化
let lastExecutionCount = 0
let lastExecutionId = null

// 删除确认对话框
const showDeleteDialog = ref(false)
const deleteType = ref('') // 'confirm' 或 'error'
const deleteAction = ref('') // 'execution' 或 'order'
const deleteId = ref(0)
const deleteMessage = ref('')

// 加载统计数据
async function loadStats(isInitialLoad = true) {
  loading.value = true
  error.value = ''

  // 加载新数据时隐藏执行步骤详情
  showExecutionSteps.value = false
  executionSteps.value = []

  try {
    const response = await api.getStrategyExecutionStats(strategyId.value, {
      page: currentPage.value,
      page_size: pageSize.value
    })
    if (response.success) {
      stats.value = response.data

      // 加载策略信息
      const strategyResponse = await api.getTradingStrategy(strategyId.value)
      if (strategyResponse.success) {
        strategy.value = strategyResponse.data
      }

      // 查找当前正在运行的执行，如果没有则显示最新的执行记录
      if (stats.value.executions && stats.value.executions.length > 0) {
        currentExecution.value = stats.value.executions.find(ex => ex.status === 'running')

        // 如果没有正在运行的执行，则显示最新的执行记录（用于查看日志）
        if (!currentExecution.value) {
          currentExecution.value = stats.value.executions[0] // 获取最新的执行记录
        }

        // 如果有执行记录，加载其步骤详情
        if (currentExecution.value) {
          await loadExecutionSteps(currentExecution.value.id)
        }
      }

      // 只有在数据发生变化时才重新渲染图表
      const currentExecutionCount = stats.value.executions.length
      const latestExecution = stats.value.executions[0] // 最新的执行记录
      const currentExecutionId = latestExecution?.id

      if (currentExecutionCount !== lastExecutionCount || currentExecutionId !== lastExecutionId || isInitialLoad) {
        setTimeout(() => {
          renderCharts()
        }, 100)

        // 更新跟踪变量
        lastExecutionCount = currentExecutionCount
        lastExecutionId = currentExecutionId
      }

      // 加载策略订单记录
      await loadStrategyOrders()
    } else {
      error.value = response.message || '加载统计数据失败'
    }
  } catch (err) {
    console.error('加载统计数据失败:', err)
    error.value = err.message || '网络错误'
  } finally {
    loading.value = false
  }
}

// 加载执行步骤详情
async function loadExecutionSteps(executionId) {
  try {
    const response = await api.getStrategyExecutionSteps(executionId)
    if (response.success) {
      executionSteps.value = response.data.steps
      currentExecution.value = response.data.execution
    }
  } catch (err) {
    console.error('加载执行步骤失败:', err)
  }
}

// 轻量级状态检查（用于实时更新）
async function checkExecutionStatus() {
  if (!currentExecution.value || currentExecution.value.status !== 'running') {
    return
  }

  try {
    // 只检查当前执行的状态，不重新加载所有数据
    const response = await api.getStrategyExecutionSteps(currentExecution.value.id)
    if (response.success && response.data.execution) {
      const updatedExecution = response.data.execution

      // 检查是否需要更新
      const needsUpdate = updatedExecution.status !== currentExecution.value.status ||
                         updatedExecution.end_time !== currentExecution.value.end_time ||
                         updatedExecution.total_pnl !== currentExecution.value.total_pnl ||
                         updatedExecution.success_orders !== currentExecution.value.success_orders ||
                         updatedExecution.failed_orders !== currentExecution.value.failed_orders

      if (needsUpdate) {
        // 只更新当前执行对象的特定字段，避免触发整个组件重新渲染
        Object.assign(currentExecution.value, {
          status: updatedExecution.status,
          end_time: updatedExecution.end_time,
          total_pnl: updatedExecution.total_pnl,
          success_orders: updatedExecution.success_orders,
          failed_orders: updatedExecution.failed_orders,
          duration: updatedExecution.duration,
          win_rate: updatedExecution.win_rate
        })

        // 如果执行完成，重新加载完整统计数据
        if (updatedExecution.status !== 'running') {
          await loadStats(false)
        } else {
          // 执行中时，更新步骤详情（只在有变化时更新）
          const newSteps = response.data.steps || []
          if (JSON.stringify(newSteps) !== JSON.stringify(executionSteps.value)) {
            executionSteps.value = newSteps
          }
        }
      }
    }
  } catch (err) {
    console.error('检查执行状态失败:', err)
  }
}

// 启动自动刷新（用于实时状态更新）
function startAutoRefresh() {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }

  refreshInterval.value = setInterval(async () => {
    await checkExecutionStatus()
  }, 3000) // 每3秒检查一次状态
}

// 停止自动刷新
function stopAutoRefresh() {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
}

// 渲染图表
function renderCharts() {
  if (!stats.value?.executions?.length) return

  // 准备数据
  const executions = stats.value.executions.slice().reverse() // 反转以显示时间顺序

  // 盈亏走势图表
  const pnlCtx = pnlChart.value?.getContext('2d')
  if (pnlCtx) {
    const pnlData = {
      labels: executions.map((_, index) => `执行${index + 1}`),
      datasets: [{
        label: '单次执行盈亏',
        data: executions.map(ex => ex.total_pnl || 0),
        borderColor: '#3b82f6',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        tension: 0.4,
        fill: true
      }]
    }

    if (!pnlChartInstance) {
      // 首次创建图表
      pnlChartInstance = new Chart(pnlCtx, {
        type: 'line',
        data: pnlData,
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: {
              display: false
            }
          },
          scales: {
            y: {
              beginAtZero: true,
              grid: {
                color: '#f3f4f6'
              }
            },
            x: {
              grid: {
                color: '#f3f4f6'
              }
            }
          }
        }
      })
    } else {
      // 更新现有图表数据
      pnlChartInstance.data = pnlData
      pnlChartInstance.update('none') // 'none' 表示不重新动画
    }
  }

  // 胜率趋势图表
  const winrateCtx = winrateChart.value?.getContext('2d')
  if (winrateCtx) {
    const winrateData = {
      labels: executions.map((_, index) => `执行${index + 1}`),
      datasets: [{
        label: '胜率 (%)',
        data: executions.map(ex => ex.win_rate || 0),
        borderColor: '#10b981',
        backgroundColor: 'rgba(16, 185, 129, 0.1)',
        tension: 0.4,
        fill: true
      }]
    }

    if (!winrateChartInstance) {
      // 首次创建图表
      winrateChartInstance = new Chart(winrateCtx, {
        type: 'line',
        data: winrateData,
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: {
              display: false
            }
          },
          scales: {
            y: {
              beginAtZero: true,
              max: 100,
              grid: {
                color: '#f3f4f6'
              }
            },
            x: {
              grid: {
                color: '#f3f4f6'
              }
            }
          }
        }
      })
    } else {
      // 更新现有图表数据
      winrateChartInstance.data = winrateData
      winrateChartInstance.update('none') // 'none' 表示不重新动画
    }
  }
}

// 查看执行详情
async function viewExecutionDetail(executionId) {
  await loadExecutionSteps(executionId)
  showExecutionSteps.value = true

  // 滚动到执行步骤区域
  const stepsSection = document.querySelector('.steps-section')
  if (stepsSection) {
    stepsSection.scrollIntoView({ behavior: 'smooth' })
  }
}

// 隐藏执行步骤详情
function hideExecutionSteps() {
  showExecutionSteps.value = false
}

// 返回策略列表
function goBack() {
  router.push('/scheduled-orders?tab=strategies')
}

// 执行删除操作
async function performDelete() {
  try {
    let response
    if (deleteAction.value === 'execution') {
      // 删除执行记录
      response = await api.deleteStrategyExecution(deleteId.value)
    } else if (deleteAction.value === 'order') {
      // 删除订单
      response = await api.deleteScheduledOrder(deleteId.value)
    }

    // 检查删除是否成功（兼容新旧响应格式）
    const isSuccess = response.success || (deleteAction.value === 'order' && response.deleted === 1)

    if (isSuccess) {
      // 关闭对话框
      showDeleteDialog.value = false

      // 重新加载数据
      if (deleteAction.value === 'execution') {
        await loadStats(false)
      } else if (deleteAction.value === 'order') {
        await loadStrategyOrders()
      }

      // 显示成功消息（如果有的话）
      console.log(`${deleteAction.value === 'execution' ? '执行记录' : '订单'}删除成功`)
    } else {
      // 显示错误信息
      let errorMessage = response.message || '删除失败'
      if (response.error === 'DATABASE_ERROR') {
        if (deleteAction.value === 'execution') {
          errorMessage = '删除失败：该执行记录包含相关的步骤记录，请稍后重试或联系管理员'
        } else {
          errorMessage = '删除失败：该订单可能正在处理中或有关联记录'
        }
      }

      // 更新对话框显示错误信息
      deleteType.value = 'error'
      deleteMessage.value = errorMessage
      console.error('删除失败:', response.message || response)
    }
  } catch (err) {
    console.error('删除操作失败:', err)
    // 显示错误信息
    deleteType.value = 'error'
    deleteMessage.value = `删除失败：${err.message || '网络错误，请稍后重试'}`
  }
}

// 确认删除执行记录
function confirmDeleteExecution(executionId) {
  deleteType.value = 'confirm'
  deleteAction.value = 'execution'
  deleteId.value = executionId
  deleteMessage.value = '确定要删除这条执行记录吗？这将同时删除相关的执行步骤记录。此操作不可恢复。'
  showDeleteDialog.value = true
}

// 确认删除订单
function confirmDeleteOrder(orderId) {
  deleteType.value = 'confirm'
  deleteAction.value = 'order'
  deleteId.value = orderId
  deleteMessage.value = '确定要删除这条订单记录吗？此操作不可恢复。'
  showDeleteDialog.value = true
}

// 取消删除
function cancelDelete() {
  showDeleteDialog.value = false
  deleteType.value = ''
  deleteAction.value = ''
  deleteId.value = 0
  deleteMessage.value = ''
}

// 分页变化处理
function onPaginationChange({ page, pageSize: newPageSize }) {
  currentPage.value = page
  pageSize.value = newPageSize
  loadStats()
}

// 加载策略订单记录
async function loadStrategyOrders() {
  loadingOrders.value = true

  try {
    const response = await api.getStrategyOrders(strategyId.value, {
      page: ordersCurrentPage.value,
      page_size: ordersPageSize.value
    })
    if (response.success) {
      strategyOrders.value = response.data
    } else {
      console.error('加载策略订单记录失败:', response.message)
    }
  } catch (err) {
    console.error('加载策略订单记录失败:', err)
  } finally {
    loadingOrders.value = false
  }
}

// 订单分页变化处理
function onOrdersPaginationChange({ page, pageSize: newPageSize }) {
  ordersCurrentPage.value = page
  ordersPageSize.value = newPageSize
  loadStrategyOrders()
}

// 查看订单详情
function viewOrderDetail(orderId) {
  router.push(`/orders/schedule/${orderId}`)
}

// 获取订单状态文本
function getOrderStatusText(status) {
  const statusMap = {
    'pending': '等待执行',
    'processing': '执行中',
    'sent': '已发送',
    'success': '已提交',
    'filled': '已完成',
    'completed': '已完成',
    'failed': '执行失败',
    'cancelled': '已取消'
  }
  return statusMap[status] || status
}

// 获取操作类型样式类
function getOperationClass(side, reduceOnly) {
  if (reduceOnly) {
    // 平仓操作
    return side === 'BUY' ? 'close-short' : 'close-long'
  } else {
    // 开仓操作
    return side === 'BUY' ? 'open-long' : 'open-short'
  }
}

// 格式化日期时间
function formatDateTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// 格式化持续时间
function formatDuration(seconds) {
  if (!seconds) return '0秒'

  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60

  const parts = []
  if (hours > 0) parts.push(`${hours}小时`)
  if (minutes > 0) parts.push(`${minutes}分钟`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs}秒`)

  return parts.join(' ')
}

// 获取执行状态文本
function getExecutionStatusText(status) {
  const statusMap = {
    'pending': '等待中',
    'running': '运行中',
    'completed': '已完成',
    'failed': '失败'
  }
  return statusMap[status] || status
}

onMounted(() => {
  loadStats()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()

  // 清理图表实例
  if (pnlChartInstance) {
    pnlChartInstance.destroy()
    pnlChartInstance = null
  }
  if (winrateChartInstance) {
    winrateChartInstance.destroy()
    winrateChartInstance = null
  }
})
</script>

<style scoped>
.strategy-stats-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
  color: var(--text);
}

.page-header {
  margin-bottom: 24px;
}

.header-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.title-section h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 700;
  color: #111827;
}

.strategy-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.strategy-name {
  font-size: 16px;
  font-weight: 600;
  color: #374151;
}

.strategy-id {
  font-size: 14px;
  color: #6b7280;
  background: #f3f4f6;
  padding: 4px 8px;
  border-radius: 4px;
}

.header-actions {
  flex-shrink: 0;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #6b7280;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e5e7eb;
  border-top: 3px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error-message {
  text-align: center;
  padding: 60px 20px;
  color: #dc2626;
}

.stats-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.stats-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.stat-card {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-1px);
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #111827;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
}

.stat-trend {
  font-size: 18px;
  margin-top: 8px;
}

.stat-trend.profit {
  color: #10b981;
}

.stat-trend.loss {
  color: #ef4444;
}

.charts-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
}

.chart-container {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.chart-container h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.pnl-chart,
.winrate-chart {
  height: 300px;
  position: relative;
}

.executions-section h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.pagination-info {
  margin-bottom: 16px;
  padding: 8px 12px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  font-size: 14px;
  color: #6b7280;
}

/* 当前执行状态样式 */
.current-execution-section h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.current-execution-card {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.execution-info {
  margin-bottom: 16px;
}

.execution-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.status-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.status-indicator.running {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.status-text {
  font-weight: 600;
  color: #10b981;
}

.execution-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 12px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-item .label {
  font-weight: 500;
  color: #6b7280;
  min-width: 80px;
}

.detail-item .value {
  font-weight: 600;
  color: #111827;
}

.progress-bar {
  flex: 1;
  height: 20px;
  background: #f3f4f6;
  border-radius: 10px;
  position: relative;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #10b981);
  border-radius: 10px;
  transition: width 0.3s ease;
}

.progress-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 12px;
  font-weight: 600;
  color: #ffffff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

.error-message {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  padding: 12px;
  color: #dc2626;
  font-size: 14px;
}

/* 执行日志样式 */
.logs-section h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.logs-container {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.logs-content {
  max-height: 300px;
  overflow-y: auto;
}

.logs-content pre {
  margin: 0;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: #374151;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.no-logs {
  padding: 40px 20px;
  text-align: center;
  color: #6b7280;
  font-style: italic;
}

/* 执行步骤样式 */
.steps-section .section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.steps-section .section-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.steps-timeline {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-item {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.step-item.active {
  border-color: #3b82f6;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
}

.step-item.completed {
  border-color: #10b981;
  background: #f0fdf4;
}

.step-item.failed {
  border-color: #ef4444;
  background: #fef2f2;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.step-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-icon {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-icon.running {
  background: #3b82f6;
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.5);
  animation: pulse 2s infinite;
}

.status-icon.completed {
  background: #10b981;
}

.status-icon.failed {
  background: #ef4444;
}

.status-icon.pending {
  background: #d1d5db;
}

.status-icon.skipped {
  background: #f59e0b;
}

.step-name {
  font-weight: 600;
  color: #111827;
}

.step-time {
  font-size: 12px;
  color: #6b7280;
}

.step-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.step-symbol {
  font-size: 14px;
  color: #374151;
  font-weight: 500;
}

.step-result {
  font-size: 14px;
  color: #4b5563;
  line-height: 1.4;
}

.step-error {
  font-size: 14px;
  color: #dc2626;
  background: #fef2f2;
  padding: 8px;
  border-radius: 4px;
  border-left: 3px solid #dc2626;
}

.step-data {
  margin-top: 8px;
}

.step-data summary {
  cursor: pointer;
  font-weight: 500;
  color: #6b7280;
  margin-bottom: 8px;
}

.step-data pre {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  padding: 8px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.4;
  color: #374151;
  overflow-x: auto;
  max-height: 200px;
  overflow-y: auto;
}

.executions-table-container {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.executions-table {
  width: 100%;
  border-collapse: collapse;
}

.executions-table th {
  background: #f9fafb;
  padding: 12px 16px;
  text-align: left;
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  border-bottom: 1px solid #e5e7eb;
  white-space: nowrap;
}

.executions-table td {
  padding: 12px 16px;
  border-bottom: 1px solid #f3f4f6;
  font-size: 13px;
  color: #374151;
}

.executions-table tbody tr:hover {
  background: #f9fafb;
}

/* 优化表格列宽 */
.executions-table th:nth-child(1), /* 执行时间 */
.executions-table td:nth-child(1) {
  width: 160px;
  min-width: 160px;
}

.executions-table th:nth-child(2), /* 状态 */
.executions-table td:nth-child(2) {
  width: 80px;
  min-width: 80px;
  text-align: center;
}

.executions-table th:nth-child(3), /* 持续时间 */
.executions-table td:nth-child(3) {
  width: 80px;
  min-width: 80px;
}

.executions-table th:nth-child(4), /* 订单数 */
.executions-table td:nth-child(4) {
  width: 70px;
  min-width: 70px;
  text-align: center;
}

.executions-table th:nth-child(5), /* 成功订单 */
.executions-table td:nth-child(5) {
  width: 80px;
  min-width: 80px;
  text-align: center;
}

.executions-table th:nth-child(6), /* 失败订单 */
.executions-table td:nth-child(6) {
  width: 80px;
  min-width: 80px;
  text-align: center;
}

.executions-table th:nth-child(7), /* 盈亏 */
.executions-table td:nth-child(7) {
  width: 90px;
  min-width: 90px;
  text-align: right;
}

.executions-table th:nth-child(8), /* 胜率 */
.executions-table td:nth-child(8) {
  width: 70px;
  min-width: 70px;
  text-align: center;
}

.executions-table th:nth-child(9), /* 操作 */
.executions-table td:nth-child(9) {
  width: 120px;
  min-width: 120px;
  text-align: center;
}

/* 操作列样式 */
.action-cell {
  padding: 8px 12px !important;
}

.action-buttons {
  display: flex;
  flex-direction: row;
  gap: 6px;
  align-items: center;
  justify-content: center;
}

.action-buttons .btn {
  padding: 6px 8px;
  font-size: 11px;
  min-width: 50px;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
  text-align: center;
  white-space: nowrap;
}

.status-badge.running {
  background: #dbeafe;
  color: #1e40af;
}

.status-badge.completed {
  background: #dcfce7;
  color: #166534;
}

.status-badge.failed {
  background: #fee2e2;
  color: #dc2626;
}

.status-badge.pending {
  background: #fef3c7;
  color: #92400e;
}

.success-count {
  color: #10b981;
  font-weight: 600;
}

.fail-count {
  color: #ef4444;
  font-weight: 600;
}

.profit {
  color: #10b981;
  font-weight: 600;
}

.loss {
  color: #ef4444;
  font-weight: 600;
}

.btn-small {
  height: 28px;
  padding: 0 8px;
  font-size: 12px;
  border-radius: 4px;
  min-width: 50px;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .charts-section {
    grid-template-columns: 1fr;
  }

  .stats-overview {
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  }
}

@media (max-width: 768px) {
  .strategy-stats-page {
    padding: 16px;
  }

  .header-main {
    flex-direction: column;
    align-items: stretch;
  }

  .title-section h1 {
    font-size: 24px;
  }

  .stats-overview {
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 12px;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-value {
    font-size: 24px;
  }

  .executions-table-container {
    overflow-x: auto;
  }

  .executions-table {
    min-width: 900px;
  }

  /* 移动端优化操作列 */
  .action-buttons {
    gap: 4px;
  }

  .action-buttons .btn {
    padding: 4px 6px;
    font-size: 10px;
    min-width: 40px;
  }
}

/* 策略订单记录样式 */
.strategy-orders-section {
  margin-top: 32px;
}

.strategy-orders-section h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.orders-table-container {
  overflow-x: auto;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  margin-bottom: 16px;
}

.orders-table {
  width: 100%;
  border-collapse: collapse;
}

.orders-table th {
  background: #f9fafb;
  padding: 12px 16px;
  text-align: left;
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  border-bottom: 1px solid #e5e7eb;
  white-space: nowrap;
}

.orders-table td {
  padding: 12px 16px;
  border-bottom: 1px solid #f3f4f6;
  font-size: 14px;
  color: #374151;
}

.orders-table tbody tr:hover {
  background: #f9fafb;
}

/* 优化订单表格列宽 */
.orders-table th:nth-child(1), /* 创建时间 */
.orders-table td:nth-child(1) {
  width: 160px;
  min-width: 160px;
}

.orders-table th:nth-child(2), /* 交易对 */
.orders-table td:nth-child(2) {
  width: 100px;
  min-width: 100px;
}

.orders-table th:nth-child(3), /* 操作类型 */
.orders-table td:nth-child(3) {
  width: 100px;
  min-width: 100px;
}

.orders-table th:nth-child(4), /* 订单类型 */
.orders-table td:nth-child(4) {
  width: 70px;
  min-width: 70px;
  text-align: center;
}

.orders-table th:nth-child(5), /* 成交数量 */
.orders-table td:nth-child(5) {
  width: 100px;
  min-width: 100px;
  text-align: right;
}

.orders-table th:nth-child(6), /* 成交价 */
.orders-table td:nth-child(6) {
  width: 100px;
  min-width: 100px;
  text-align: right;
}

.orders-table th:nth-child(7), /* 杠杆 */
.orders-table td:nth-child(7) {
  width: 70px;
  min-width: 70px;
  text-align: center;
}

.orders-table th:nth-child(8), /* 状态 */
.orders-table td:nth-child(8) {
  width: 90px;
  min-width: 90px;
  text-align: center;
}

.orders-table th:nth-child(9), /* 操作 */
.orders-table td:nth-child(9) {
  width: 120px;
  min-width: 120px;
  text-align: center;
}

.operation-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
}

.operation-badge.open-long {
  background: #dcfce7;
  color: #166534;
}

.operation-badge.open-short {
  background: #fee2e2;
  color: #dc2626;
}

.operation-badge.close-long {
  background: #dbeafe;
  color: #1e40af;
}

.operation-badge.close-short {
  background: #fef3c7;
  color: #92400e;
}

@media (max-width: 768px) {
  .strategy-orders-section {
    margin-top: 24px;
  }

  .orders-table-container {
    overflow-x: auto;
  }

  .orders-table {
    min-width: 900px;
  }

  /* 移动端优化订单表格操作列 */
  .orders-table .action-buttons {
    gap: 4px;
  }

  .orders-table .action-buttons .btn {
    padding: 4px 6px;
    font-size: 10px;
    min-width: 40px;
  }

  /* 移动端优化步骤详情头部 */
  .steps-section .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .steps-section .section-header .btn {
    align-self: flex-end;
  }
}

/* 删除按钮样式 */
.btn-danger {
  background-color: #dc2626;
  color: white;
  border: 1px solid #dc2626;
}

.btn-danger:hover {
  background-color: #b91c1c;
  border-color: #b91c1c;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  padding: 24px;
  max-width: 400px;
  width: 90%;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
}

.modal-content h3 {
  margin: 0 0 16px 0;
  color: #111827;
  font-size: 18px;
  font-weight: 600;
}

.modal-content p {
  margin: 0 0 24px 0;
  color: #6b7280;
  line-height: 1.5;
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.modal-actions .btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.modal-actions .btn-danger {
  background-color: #dc2626;
  color: white;
  border: 1px solid #dc2626;
}

.modal-actions .btn-danger:hover {
  background-color: #b91c1c;
  border-color: #b91c1c;
}

.modal-actions .btn {
  background-color: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.modal-actions .btn:hover {
  background-color: #e5e7eb;
}
</style>
