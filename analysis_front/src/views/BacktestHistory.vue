<template>
  <div class="backtest-history">
    <!-- 页面头部 -->
    <section class="panel">
      <div class="row">
        <h2>📈 回测记录历史</h2>
        <div class="spacer"></div>
        <button @click="refreshRecords" :disabled="loading" class="secondary">
          {{ loading ? '刷新中...' : '🔄 刷新' }}
        </button>
      </div>

      <!-- 统计信息 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-value">{{ totalRecords }}</div>
          <div class="stat-label">总记录数</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ completedCount }}</div>
          <div class="stat-label">已完成</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ runningCount }}</div>
          <div class="stat-label">运行中</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ failedCount }}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>
    </section>

    <!-- 筛选器 -->
    <section class="panel">
      <div class="filter-row">
        <div class="filter-group">
          <label>状态:</label>
          <select v-model="filters.status" @change="loadRecords">
            <option value="">全部</option>
            <option value="completed">已完成</option>
            <option value="running">运行中</option>
            <option value="failed">失败</option>
            <option value="pending">等待中</option>
          </select>
        </div>

        <div class="filter-group">
          <label>交易对:</label>
          <input
            v-model="filters.symbol"
            @input="debounceSearch"
            placeholder="输入交易对符号"
            type="text"
          />
        </div>

        <div class="filter-group">
          <label>每页显示:</label>
          <select v-model="filters.limit" @change="loadRecords">
            <option :value="10">10</option>
            <option :value="20">20</option>
            <option :value="50">50</option>
            <option :value="100">100</option>
          </select>
        </div>
      </div>
    </section>

    <!-- 回测记录列表 -->
    <section class="panel">
      <div class="table-container">
        <table class="records-table" v-if="records.length > 0">
          <thead>
            <tr>
              <th @click="sortBy('id')" :class="{ 'sort-asc': sortField === 'id' && sortOrder === 'asc', 'sort-desc': sortField === 'id' && sortOrder === 'desc' }">
                ID
              </th>
              <th @click="sortBy('symbol')" :class="{ 'sort-asc': sortField === 'symbol' && sortOrder === 'asc', 'sort-desc': sortField === 'symbol' && sortOrder === 'desc' }">
                交易对
              </th>
              <th @click="sortBy('strategy')" :class="{ 'sort-asc': sortField === 'strategy' && sortOrder === 'asc', 'sort-desc': sortField === 'strategy' && sortOrder === 'desc' }">
                策略
              </th>
              <th @click="sortBy('created_at')" :class="{ 'sort-asc': sortField === 'created_at' && sortOrder === 'asc', 'sort-desc': sortField === 'created_at' && sortOrder === 'desc' }">
                创建时间
              </th>
              <th @click="sortBy('status')" :class="{ 'sort-asc': sortField === 'status' && sortOrder === 'asc', 'sort-desc': sortField === 'status' && sortOrder === 'desc' }">
                状态
              </th>
              <th>总收益率</th>
              <th>交易次数</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="record in records" :key="record.id" :class="record.status">
              <td>{{ record.id }}</td>
              <td>
                <span class="symbol">{{ record.symbol }}</span>
              </td>
              <td>{{ record.strategy }}</td>
              <td>{{ formatDateTime(record.created_at) }}</td>
              <td>
                <span class="status-badge" :class="record.status">
                  {{ getStatusText(record.status) }}
                </span>
              </td>
              <td v-if="record.result && record.result.total_return !== undefined">
                <span :class="{ 'positive': record.result.total_return > 0, 'negative': record.result.total_return < 0 }">
                  {{ (record.result.total_return * 100).toFixed(2) }}%
                </span>
              </td>
              <td v-else>-</td>
              <td v-if="record.result && record.result.total_trades !== undefined">
                {{ record.result.total_trades }}
              </td>
              <td v-else>-</td>
              <td>
                <div class="action-buttons">
                  <button @click="viewDetail(record.id)" class="primary small" title="查看详情">
                    👁️ 详情
                  </button>
                  <button v-if="record.status === 'completed'" @click="reRunBacktest(record)" class="secondary small" title="重新运行">
                    🔄 重跑
                  </button>
                  <button @click="deleteRecord(record.id)" class="danger small" title="删除记录">
                    🗑️ 删除
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- 空状态 -->
        <div v-else-if="!loading" class="empty-state">
          <div class="empty-icon">📊</div>
          <h3>暂无回测记录</h3>
          <p>开始您的第一个策略回测吧！</p>
          <router-link to="/backtest" class="primary">
            前往回测页面
          </router-link>
        </div>

        <!-- 加载状态 -->
        <div v-else class="loading-state">
          <div class="loading-spinner"></div>
          <p>加载中...</p>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="totalPages > 1" class="pagination">
        <button @click="goToPage(currentPage - 1)" :disabled="currentPage <= 1" class="page-btn">
          上一页
        </button>

        <span v-for="page in visiblePages" :key="page">
          <button
            v-if="page !== '...'"
            @click="goToPage(page)"
            :class="{ 'active': page === currentPage }"
            class="page-btn"
          >
            {{ page }}
          </button>
          <span v-else class="page-ellipsis">{{ page }}</span>
        </span>

        <button @click="goToPage(currentPage + 1)" :disabled="currentPage >= totalPages" class="page-btn">
          下一页
        </button>

        <div class="page-info">
          第 {{ currentPage }} / {{ totalPages }} 页，共 {{ totalRecords }} 条记录
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { api } from '../api/api.js'

// 验证API导入是否正常工作
console.log('BacktestHistory: API imported successfully', api)

// 验证用户认证状态
import { useAuth } from '../stores/auth.js'
const { isAuthed, token } = useAuth()
console.log('BacktestHistory: 用户认证状态:', isAuthed.value, 'Token:', token.value ? '已设置' : '未设置')

export default {
  name: 'BacktestHistory',
  data() {
    return {
      loading: false,
      records: [],
      currentPage: 1,
      totalPages: 0,
      totalRecords: 0,
      sortField: 'created_at',
      sortOrder: 'desc',
      filters: {
        status: '',
        symbol: '',
        limit: 20
      },
      searchTimeout: null
    }
  },
  computed: {
    completedCount() {
      return this.records.filter(r => r.status === 'completed').length
    },
    runningCount() {
      return this.records.filter(r => r.status === 'running').length
    },
    failedCount() {
      return this.records.filter(r => r.status === 'failed').length
    },
    visiblePages() {
      const pages = []
      const total = this.totalPages
      const current = this.currentPage

      if (total <= 7) {
        for (let i = 1; i <= total; i++) {
          pages.push(i)
        }
      } else {
        if (current <= 4) {
          for (let i = 1; i <= 5; i++) {
            pages.push(i)
          }
          pages.push('...')
          pages.push(total)
        } else if (current >= total - 3) {
          pages.push(1)
          pages.push('...')
          for (let i = total - 4; i <= total; i++) {
            pages.push(i)
          }
        } else {
          pages.push(1)
          pages.push('...')
          for (let i = current - 1; i <= current + 1; i++) {
            pages.push(i)
          }
          pages.push('...')
          pages.push(total)
        }
      }

      return pages
    }
  },
  mounted() {
    this.loadRecords()
  },
  methods: {
    async loadRecords() {
      this.loading = true
      try {
        const response = await api.getBacktestRecords({
          page: this.currentPage,
          limit: this.filters.limit,
          status: this.filters.status || undefined,
          symbol: this.filters.symbol || undefined,
          sort_by: this.sortField,
          sort_order: this.sortOrder
        })

        // 解析结果 - 匹配后端返回的数据结构
        console.log('API响应:', response)
        console.log('Records字段:', response.records)
        console.log('Records类型:', typeof response.records)
        if (response.records && Array.isArray(response.records)) {
          this.records = response.records.map(record => {
            // 如果有result字段，解析JSON
            if (record.result && typeof record.result === 'string') {
              try {
                record.result = JSON.parse(record.result)
              } catch (e) {
                console.warn('解析回测结果失败:', e)
                record.result = null
              }
            }
            return record
          })
          this.totalRecords = response.pagination?.total || 0
          this.totalPages = response.pagination?.pages || 1
        } else {
          this.records = []
          this.totalRecords = 0
          this.totalPages = 1
        }
      } catch (error) {
        console.error('加载回测记录失败:', error)
        this.$toast?.error('加载回测记录失败')
        this.records = []
        this.totalRecords = 0
        this.totalPages = 1
      } finally {
        this.loading = false
      }
    },

    async refreshRecords() {
      this.currentPage = 1
      await this.loadRecords()
    },

    sortBy(field) {
      if (this.sortField === field) {
        this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortField = field
        this.sortOrder = 'desc'
      }
      this.loadRecords()
    },

    debounceSearch() {
      clearTimeout(this.searchTimeout)
      this.searchTimeout = setTimeout(() => {
        this.currentPage = 1
        this.loadRecords()
      }, 500)
    },

    goToPage(page) {
      if (page >= 1 && page <= this.totalPages) {
        this.currentPage = page
        this.loadRecords()
      }
    },

    viewDetail(recordId) {
      this.$router.push(`/backtest/${recordId}`)
    },

    async reRunBacktest(record) {
      // 这里可以实现重新运行回测的功能
      // 暂时跳转到回测页面
      this.$router.push({
        path: '/backtest',
        query: {
          symbol: record.symbol,
          start_date: record.start_date,
          end_date: record.end_date,
          strategy: record.strategy
        }
      })
    },

    async deleteRecord(recordId) {
      if (!confirm('确定要删除这条回测记录吗？此操作不可恢复。')) {
        return
      }

      try {
        await api.deleteBacktestRecord(recordId)
        this.$toast?.success('回测记录已删除')
        await this.loadRecords() // 重新加载列表
      } catch (error) {
        console.error('删除回测记录失败:', error)
        this.$toast?.error('删除回测记录失败')
      }
    },

    getStatusText(status) {
      const statusMap = {
        'pending': '等待中',
        'running': '运行中',
        'completed': '已完成',
        'failed': '失败'
      }
      return statusMap[status] || status
    },

    formatDateTime(dateString) {
      if (!dateString) return '-'
      const date = new Date(dateString)
      return date.toLocaleString('zh-CN')
    }
  }
}
</script>

<style scoped>
.backtest-history {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
  margin-top: 20px;
}

.stat-card {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
  border: 1px solid #e9ecef;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #2c3e50;
  margin-bottom: 5px;
}

.stat-label {
  color: #6c757d;
  font-size: 14px;
}

.filter-row {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-group label {
  font-weight: 500;
  color: #495057;
  white-space: nowrap;
}

.filter-group input,
.filter-group select {
  padding: 8px 12px;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font-size: 14px;
  min-width: 120px;
}

.table-container {
  overflow-x: auto;
  margin-top: 20px;
}

.records-table {
  width: 100%;
  border-collapse: collapse;
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.records-table th {
  background: #f8f9fa;
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: #495057;
  border-bottom: 1px solid #dee2e6;
  cursor: pointer;
  user-select: none;
  position: relative;
}

.records-table th:hover {
  background: #e9ecef;
}

.records-table th.sort-asc::after {
  content: ' ↑';
}

.records-table th.sort-desc::after {
  content: ' ↓';
}

.records-table td {
  padding: 12px 16px;
  border-bottom: 1px solid #dee2e6;
  vertical-align: middle;
}

.records-table tbody tr:hover {
  background: #f8f9fa;
}

.records-table tbody tr.completed {
  background: rgba(40, 167, 69, 0.05);
}

.records-table tbody tr.failed {
  background: rgba(220, 53, 69, 0.05);
}

.records-table tbody tr.running {
  background: rgba(255, 193, 7, 0.05);
}

.symbol {
  font-weight: 600;
  color: #2c3e50;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
}

.status-badge.completed {
  background: #d4edda;
  color: #155724;
}

.status-badge.running {
  background: #fff3cd;
  color: #856404;
}

.status-badge.failed {
  background: #f8d7da;
  color: #721c24;
}

.status-badge.pending {
  background: #e2e3e5;
  color: #383d41;
}

.positive {
  color: #28a745;
  font-weight: 600;
}

.negative {
  color: #dc3545;
  font-weight: 600;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.action-buttons button {
  padding: 6px 12px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.action-buttons button:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
}

.action-buttons button.small {
  padding: 4px 8px;
  font-size: 11px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #6c757d;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 20px;
}

.empty-state h3 {
  margin-bottom: 10px;
  color: #495057;
}

.empty-state p {
  margin-bottom: 20px;
}

.loading-state {
  text-align: center;
  padding: 60px 20px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #007bff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 10px;
  margin-top: 20px;
  flex-wrap: wrap;
}

.page-btn {
  padding: 8px 12px;
  border: 1px solid #ced4da;
  background: white;
  color: #495057;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  background: #f8f9fa;
  border-color: #adb5bd;
}

.page-btn.active {
  background: #007bff;
  color: white;
  border-color: #007bff;
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-ellipsis {
  padding: 8px 4px;
  color: #6c757d;
}

.page-info {
  margin-left: 20px;
  color: #6c757d;
  font-size: 14px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .filter-row {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-group {
    justify-content: space-between;
  }

  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .action-buttons {
    flex-direction: column;
  }

  .pagination {
    justify-content: center;
  }

  .page-info {
    margin-left: 0;
    margin-top: 10px;
    text-align: center;
    width: 100%;
  }
}
</style>
