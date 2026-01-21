<template>
  <div class="backtest-records-panel">
    <div class="panel-header">
      <h3>📋 回测记录</h3>
      <p>查看和管理所有回测任务的执行状态和结果</p>
    </div>

    <!-- 筛选和刷新 -->
    <div class="controls-section">
      <div class="filter-controls">
        <label>
          <span class="filter-icon">🔍</span> 状态：
          <select v-model="filterStatus" @change="loadRecords">
            <option value="">全部状态</option>
            <option value="pending">⏳ 等待中</option>
            <option value="running">🔄 执行中</option>
            <option value="completed">✅ 已完成</option>
            <option value="failed">❌ 失败</option>
          </select>
        </label>
        <label>
          <span class="filter-icon">💰</span> 币种：
          <input v-model="filterSymbol" placeholder="输入币种代码，如 BTC" @input="debouncedLoad" />
        </label>
      </div>
      <div class="action-controls">
        <div v-if="hasRunningBacktests" class="auto-refresh-indicator">
          <div class="refresh-spinner"></div>
          <span>自动刷新中</span>
        </div>
        <button @click="loadRecords" :disabled="loading" class="refresh-btn">
          {{ loading ? '刷新中...' : '🔄 刷新' }}
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>正在加载回测记录...</p>
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
      <button @click="errorMessage = ''; loadRecords()" class="btn-retry">重试</button>
    </div>

    <!-- 空状态 -->
    <div v-else-if="records.length === 0" class="empty-state">
      <div class="empty-icon">📋</div>
      <h4>暂无回测记录</h4>
      <p>在"策略回测"标签页中启动回测任务后，记录将显示在这里</p>
    </div>

    <!-- 记录列表 -->
    <div v-else class="records-container">
      <div class="records-grid">
        <div
          v-for="record in records"
          :key="record.id"
          class="record-card"
          :class="{ 'running': record.status === 'running' }"
        >
          <!-- 卡片头部 -->
          <div class="record-header">
            <div class="record-info">
              <h4>{{ record.symbol }} - {{ getStrategyName(record.strategy) }}</h4>
              <div class="record-meta">
                <span class="record-id">#{{ record.id }}</span>
                <span class="record-time">{{ formatDateTime(record.created_at) }}</span>
              </div>
            </div>
            <div class="record-status">
              <span :class="['status-badge', `status-${record.status}`]">
                {{ getStatusText(record.status) }}
              </span>
            </div>
          </div>

          <!-- 回测参数 -->
          <div class="record-params">
            <div class="param-item">
              <span class="param-label">时间范围：</span>
              <span class="param-value">{{ record.start_date }} 至 {{ record.end_date }}</span>
            </div>
            <div class="param-item">
              <span class="param-label">初始资金：</span>
              <span class="param-value">${{ record.initial_capital.toLocaleString() }}</span>
            </div>
            <div class="param-item">
              <span class="param-label">仓位大小：</span>
              <span class="param-value">{{ (record.position_size * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <!-- 执行中状态 -->
          <div v-if="record.status === 'running'" class="running-status">
            <div class="running-spinner"></div>
            <span>回测执行中，请稍候...</span>
          </div>

          <!-- 完成状态结果 -->
          <div v-else-if="record.status === 'completed' && record.result" class="result-summary">
            <div class="result-metrics">
              <div class="metric">
                <span class="metric-label">总收益率</span>
                <span
                  class="metric-value"
                  :class="{ 'positive': record.result.total_return >= 0, 'negative': record.result.total_return < 0 }"
                >
                  {{ (record.result.total_return * 100).toFixed(2) }}%
                </span>
              </div>
              <div class="metric">
                <span class="metric-label">夏普比率</span>
                <span class="metric-value">{{ record.result.sharpe_ratio?.toFixed(2) || 'N/A' }}</span>
              </div>
              <div class="metric">
                <span class="metric-label">最大回撤</span>
                <span class="metric-value">{{ (record.result.max_drawdown * 100)?.toFixed(2) || 'N/A' }}%</span>
              </div>
              <div class="metric">
                <span class="metric-label">胜率</span>
                <span class="metric-value">{{ (record.result.win_rate * 100)?.toFixed(1) || 'N/A' }}%</span>
              </div>
            </div>
            <div class="result-actions">
              <button @click="viewResult(record)" class="btn-view">查看详情</button>
              <button @click="exportResult(record)" class="btn-export">导出报告</button>
            </div>
          </div>

          <!-- 失败状态 -->
          <div v-else-if="record.status === 'failed'" class="error-status">
            <div class="error-icon">❌</div>
            <div class="error-details">
              <p class="error-title">回测执行失败</p>
              <p class="error-message">{{ record.error_message || '未知错误' }}</p>
            </div>
            <button @click="retryRecord(record)" class="btn-retry">重试</button>
          </div>

          <!-- 卡片底部操作 -->
          <div class="record-actions">
            <button @click="deleteRecord(record)" class="btn-delete" :disabled="record.status === 'running'">
              🗑️ 删除
            </button>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="totalPages > 1" class="pagination">
        <button
          @click="changePage(currentPage - 1)"
          :disabled="currentPage <= 1"
          class="page-btn"
        >
          上一页
        </button>

        <span class="page-info">
          第 {{ currentPage }} 页，共 {{ totalPages }} 页
        </span>

        <button
          @click="changePage(currentPage + 1)"
          :disabled="currentPage >= totalPages"
          class="page-btn"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/api.js'

export default {
  name: 'BacktestRecordsPanel',
  props: {
    symbols: {
      type: Array,
      default: () => ['BTC']
    }
  },
  emits: ['record-selected'],
  setup(props, { emit }) {
    const router = useRouter()
    const loading = ref(false)
    const loadingProgress = ref(0)
    const errorMessage = ref('')
    const records = ref([])
    const currentPage = ref(1)
    const pageSize = ref(10)
    const totalPages = ref(1)

    // 筛选条件
    const filterStatus = ref('')
    const filterSymbol = ref('')

    // 自动刷新
    const autoRefreshEnabled = ref(false)
    const autoRefreshInterval = ref(null)
    const hasRunningBacktests = computed(() => {
      return records.value.some(record => record.status === 'running')
    })

    let loadTimeout = null
    const debouncedLoad = () => {
      if (loadTimeout) clearTimeout(loadTimeout)
      loadTimeout = setTimeout(() => loadRecords(), 500)
    }

    const startAutoRefresh = () => {
      if (autoRefreshEnabled.value && !autoRefreshInterval.value) {
        autoRefreshInterval.value = setInterval(() => {
          // 只有当有正在运行的回测时才自动刷新
          if (hasRunningBacktests.value) {
            loadRecords()
          }
        }, 5000) // 每5秒刷新一次
      }
    }

    const stopAutoRefresh = () => {
      if (autoRefreshInterval.value) {
        clearInterval(autoRefreshInterval.value)
        autoRefreshInterval.value = null
      }
    }

    const loadRecords = async () => {
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
          limit: pageSize.value
        }

        if (filterStatus.value) params.status = filterStatus.value
        if (filterSymbol.value) params.symbol = filterSymbol.value

        const response = await api.getBacktestRecords(params)

        clearInterval(progressInterval)
        loadingProgress.value = 100

        if (response.success) {
          records.value = response.records || []
          totalPages.value = response.pagination?.pages || 1
        } else {
          throw new Error(response.error || '加载失败')
        }

        // 短暂延迟以显示100%状态
        setTimeout(() => {
          loading.value = false
          // 加载完成后启动自动刷新（如果有正在运行的回测）
          startAutoRefresh()
        }, 300)

      } catch (error) {
        console.error('加载回测记录失败:', error)
        errorMessage.value = error.message || '加载失败，请稍后重试'
        loading.value = false
        loadingProgress.value = 0
      }
    }

    const getStrategyName = (strategy) => {
      const strategyMap = {
        'conservative': '保守策略',
        'moderate': '稳健策略',
        'aggressive': '激进策略',
        'deep_learning': '深度学习策略'
      }
      return strategyMap[strategy] || strategy
    }

    const getStatusText = (status) => {
      const statusMap = {
        'pending': '等待中',
        'running': '执行中',
        'completed': '已完成',
        'failed': '失败'
      }
      return statusMap[status] || status
    }

    const formatDateTime = (dateTimeStr) => {
      if (!dateTimeStr) return '-'
      const date = new Date(dateTimeStr)
      return date.toLocaleString('zh-CN')
    }

    const viewResult = (record) => {
      // 导航到回测详情页面
      router.push(`/backtest/${record.id}`)
    }

    const exportResult = (record) => {
      // 导出回测结果
      console.log('导出回测报告:', record)
      alert('导出功能开发中')
    }

    const retryRecord = (record) => {
      // 重试失败的回测
      console.log('重试回测:', record)
      alert('重试功能开发中')
    }

    const deleteRecord = async (record) => {
      if (!confirm(`确定要删除回测记录 #${record.id} 吗？此操作不可撤销。`)) {
        return
      }

      try {
        const response = await api.deleteBacktestRecord(record.id)
        if (response.success) {
          alert('回测记录已删除')
          loadRecords() // 重新加载列表
        } else {
          throw new Error(response.error || '删除失败')
        }
      } catch (error) {
        console.error('删除回测记录失败:', error)
        alert('删除失败: ' + (error.message || '未知错误'))
      }
    }

    const changePage = (page) => {
      currentPage.value = page
      loadRecords()
    }

    onMounted(() => {
      loadRecords()
    })

    onUnmounted(() => {
      stopAutoRefresh()
    })

    return {
      loading,
      loadingProgress,
      errorMessage,
      records,
      currentPage,
      totalPages,
      filterStatus,
      filterSymbol,
      hasRunningBacktests,
      autoRefreshEnabled,
      debouncedLoad,
      loadRecords,
      getStrategyName,
      getStatusText,
      formatDateTime,
      viewResult,
      exportResult,
      retryRecord,
      deleteRecord,
      changePage,
      startAutoRefresh,
      stopAutoRefresh
    }
  }
}
</script>

<style scoped>
.backtest-records-panel {
  padding: 24px;
}

.panel-header {
  text-align: center;
  margin-bottom: 32px;
}

.panel-header h3 {
  margin: 0 0 8px 0;
  font-size: 1.5rem;
  color: #1f2937;
}

.panel-header p {
  margin: 0;
  color: #6b7280;
}

.controls-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.filter-controls {
  display: flex;
  gap: 16px;
  align-items: center;
}

.filter-controls label {
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

.filter-controls input,
.filter-controls select {
  padding: 6px 10px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  min-width: 120px;
}

.action-controls {
  display: flex;
  gap: 12px;
}

.refresh-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  background: #3b82f6;
  color: white;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.refresh-btn:hover:not(:disabled) {
  background: #2563eb;
}

.refresh-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.auto-refresh-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #10b981;
  font-size: 14px;
  font-weight: 500;
}

.refresh-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid #e5e7eb;
  border-top: 2px solid #10b981;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.records-container {
  margin-top: 24px;
}

.records-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.record-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border: 1px solid #e5e7eb;
  transition: all 0.2s ease;
}

.record-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.record-card.running {
  border-left: 4px solid #3b82f6;
  background: linear-gradient(135deg, #eff6ff 0%, #ffffff 100%);
}

.record-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.record-info h4 {
  margin: 0 0 8px 0;
  font-size: 1.1rem;
  color: #1f2937;
}

.record-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #6b7280;
}

.record-id {
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-running {
  background: #dbeafe;
  color: #1e40af;
}

.status-completed {
  background: #dcfce7;
  color: #166534;
}

.status-failed {
  background: #fee2e2;
  color: #991b1b;
}

.record-params {
  margin-bottom: 16px;
}

.param-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 14px;
}

.param-label {
  color: #6b7280;
}

.param-value {
  font-weight: 500;
  color: #1f2937;
}

.running-status {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #eff6ff;
  border-radius: 8px;
  margin-bottom: 16px;
  color: #1e40af;
  font-weight: 500;
}

.running-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid #3b82f6;
  border-top: 2px solid transparent;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.result-summary {
  margin-bottom: 16px;
}

.result-metrics {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}

.metric {
  text-align: center;
  padding: 8px;
  background: #f8fafc;
  border-radius: 6px;
}

.metric-label {
  display: block;
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 4px;
}

.metric-value {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.metric-value.positive {
  color: #10b981;
}

.metric-value.negative {
  color: #ef4444;
}

.result-actions {
  display: flex;
  gap: 8px;
}

.btn-view,
.btn-export {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  color: #374151;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-view:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
}

.btn-export {
  background: #10b981;
  color: white;
  border-color: #10b981;
}

.btn-export:hover {
  background: #059669;
}

.error-status {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #fef2f2;
  border-radius: 8px;
  margin-bottom: 16px;
}

.error-icon {
  font-size: 24px;
}

.error-details {
  flex: 1;
}

.error-title {
  margin: 0 0 4px 0;
  font-weight: 600;
  color: #dc2626;
}

.error-message {
  margin: 0;
  font-size: 14px;
  color: #991b1b;
}

.record-actions {
  display: flex;
  justify-content: flex-end;
}

.btn-delete {
  padding: 6px 12px;
  border: 1px solid #dc2626;
  border-radius: 4px;
  background: #dc2626;
  color: white;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-delete:hover:not(:disabled) {
  background: #b91c1c;
  border-color: #b91c1c;
}

.btn-delete:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-retry {
  padding: 6px 12px;
  border: 1px solid #d97706;
  border-radius: 4px;
  background: #d97706;
  color: white;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-retry:hover {
  background: #b45309;
  border-color: #b45309;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 32px;
  padding: 16px;
}

.page-btn {
  padding: 8px 16px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
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

.loading-container,
.empty-state {
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

.empty-icon {
  font-size: 48px;
  opacity: 0.5;
}

.empty-state h4 {
  margin: 0;
  color: #6b7280;
  font-size: 1.2rem;
}

.empty-state p {
  margin: 0;
  color: #9ca3af;
  font-size: 14px;
}

.error-message {
  background: #fef2f2;
  color: #dc2626;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #fecaca;
  margin: 16px 0;
  text-align: center;
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

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .controls-section {
    flex-direction: column;
    gap: 16px;
  }

  .filter-controls {
    flex-direction: column;
    width: 100%;
  }

  .filter-controls label {
    justify-content: space-between;
  }

  .records-grid {
    grid-template-columns: 1fr;
  }

  .result-metrics {
    grid-template-columns: 1fr;
  }

  .result-actions {
    flex-direction: column;
  }
}
</style>
