<template>
  <div class="backtest-detail">
    <!-- 页面头部 -->
    <section class="panel header-panel">
      <div class="row">
        <div class="header-content">
          <h1>📊 回测详情</h1>
          <p class="subtitle">{{ record?.symbol }} - {{ getStrategyName(record?.strategy) }} 回测分析报告</p>
        </div>
        <div class="header-actions">
          <button @click="goBack" class="back-btn">
            ← 返回列表
          </button>
          <button @click="exportReport" class="export-btn">
            📊 导出报告
          </button>
        </div>
      </div>
    </section>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>正在加载回测详情...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="errorMessage" class="error-message">
      <p>{{ errorMessage }}</p>
      <button @click="loadData" class="btn-retry">重试</button>
    </div>

    <!-- 回测详情内容 -->
    <div v-else-if="record && backtestResult" class="detail-content">
      <!-- 回测概览 -->
      <section class="panel overview-section">
        <h3>📈 回测概览</h3>
        <div class="overview-grid">
          <div class="overview-card">
            <div class="card-icon">💰</div>
            <div class="card-content">
              <div class="card-value">{{ formatCurrency(backtestResult.summary?.total_return || 0) }}</div>
              <div class="card-label">总收益率</div>
              <div class="card-change" :class="(backtestResult.summary?.total_return || 0) >= 0 ? 'positive' : 'negative'">
                {{ ((backtestResult.summary?.total_return || 0) * 100).toFixed(2) }}%
              </div>
            </div>
          </div>

          <div class="overview-card">
            <div class="card-icon">📊</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.summary?.sharpe_ratio?.toFixed(2) || 'N/A' }}</div>
              <div class="card-label">夏普比率</div>
              <div class="card-desc">{{ getSharpeDesc(backtestResult.summary?.sharpe_ratio) }}</div>
            </div>
          </div>

          <div class="overview-card">
            <div class="card-icon">📉</div>
            <div class="card-content">
              <div class="card-value">{{ ((backtestResult.summary?.max_drawdown || 0) * 100).toFixed(2) }}%</div>
              <div class="card-label">最大回撤</div>
              <div class="card-desc">{{ getDrawdownDesc(backtestResult.summary?.max_drawdown) }}</div>
            </div>
          </div>

          <div class="overview-card">
            <div class="card-icon">🎯</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.summary?.win_rate ? (backtestResult.summary.win_rate * 100).toFixed(1) : 'N/A' }}%</div>
              <div class="card-label">胜率</div>
              <div class="card-desc">{{ backtestResult.trade_count || 0 }} 次交易</div>
            </div>
          </div>
        </div>
      </section>

      <!-- 回测配置信息 -->
      <section class="panel config-section">
        <h3>⚙️ 回测配置</h3>
        <div class="config-grid">
          <div class="config-item">
            <span class="config-label">币种:</span>
            <span class="config-value">{{ record.symbol }}</span>
          </div>
          <div class="config-item">
            <span class="config-label">策略:</span>
            <span class="config-value">{{ getStrategyName(record.strategy) }}</span>
          </div>
          <div class="config-item">
            <span class="config-label">时间范围:</span>
            <span class="config-value">{{ record.start_date }} 至 {{ record.end_date }}</span>
          </div>
          <div class="config-item">
            <span class="config-label">初始资金:</span>
            <span class="config-value">{{ formatCurrency(record.initial_capital) }}</span>
          </div>
          <div class="config-item">
            <span class="config-label">仓位大小:</span>
            <span class="config-value">{{ (record.position_size * 100).toFixed(1) }}%</span>
          </div>
          <div class="config-item">
            <span class="config-label">开始时间:</span>
            <span class="config-value">{{ formatDateTime(record.created_at) }}</span>
          </div>
          <div class="config-item">
            <span class="config-label">完成时间:</span>
            <span class="config-value">{{ record.completed_at ? formatDateTime(record.completed_at) : '未完成' }}</span>
          </div>
          <div class="config-item">
            <span class="config-label">状态:</span>
            <span class="config-value status" :class="'status-' + record.status">{{ getStatusText(record.status) }}</span>
          </div>
        </div>
      </section>

      <!-- AI分析洞察 -->
      <section class="panel insights-section" v-if="backtestResult.backtest_insights">
        <h3>🤖 AI分析洞察</h3>
        <div class="insights-list">
          <div v-for="insight in backtestResult.backtest_insights" :key="insight" class="insight-item">
            <span class="insight-icon">💡</span>
            <span class="insight-text">{{ insight }}</span>
          </div>
        </div>
      </section>

      <!-- AI预测准确性分析 -->
      <section class="panel accuracy-section" v-if="backtestResult.ai_prediction_accuracy">
        <h3>🎯 AI预测准确性分析</h3>
        <div class="accuracy-grid">
          <div class="accuracy-card">
            <div class="card-icon">📊</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.ai_prediction_accuracy.total_trades || 0 }}</div>
              <div class="card-label">总交易次数</div>
            </div>
          </div>

          <div class="accuracy-card">
            <div class="card-icon">✅</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.ai_prediction_accuracy.profitable_trades || 0 }}</div>
              <div class="card-label">盈利交易</div>
            </div>
          </div>

          <div class="accuracy-card">
            <div class="card-icon">🎯</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.ai_prediction_accuracy.win_rate ? (backtestResult.ai_prediction_accuracy.win_rate * 100).toFixed(1) : 'N/A' }}%</div>
              <div class="card-label">AI预测胜率</div>
              <div class="card-desc">{{ getAccuracyDesc(backtestResult.ai_prediction_accuracy.accuracy_score) }}</div>
            </div>
          </div>

          <div class="accuracy-card">
            <div class="card-icon">⭐</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.ai_prediction_accuracy.accuracy_score ? (backtestResult.ai_prediction_accuracy.accuracy_score * 100).toFixed(1) : 'N/A' }}%</div>
              <div class="card-label">准确性评分</div>
              <div class="card-desc">{{ getAccuracyLevel(backtestResult.ai_prediction_accuracy.accuracy_score) }}</div>
            </div>
          </div>
        </div>
      </section>

      <!-- 推荐有效性分析 -->
      <section class="panel effectiveness-section" v-if="backtestResult.recommendation_effectiveness">
        <h3>📈 推荐有效性分析</h3>
        <div class="effectiveness-grid">
          <div class="effectiveness-card">
            <div class="card-icon">💰</div>
            <div class="card-content">
              <div class="card-value">{{ formatCurrency(backtestResult.recommendation_effectiveness.actual_return || 0) }}</div>
              <div class="card-label">实际收益率</div>
            </div>
          </div>

          <div class="effectiveness-card">
            <div class="card-icon">🎯</div>
            <div class="card-content">
              <div class="card-value">{{ ((backtestResult.recommendation_effectiveness.expected_return || 0) * 100).toFixed(1) }}%</div>
              <div class="card-label">预期收益率</div>
            </div>
          </div>

          <div class="effectiveness-card">
            <div class="card-icon">📊</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.recommendation_effectiveness.effectiveness ? (backtestResult.recommendation_effectiveness.effectiveness * 100).toFixed(1) : 'N/A' }}%</div>
              <div class="card-label">有效性比率</div>
              <div class="card-desc">{{ getEffectivenessDesc(backtestResult.recommendation_effectiveness.effectiveness) }}</div>
            </div>
          </div>

          <div class="effectiveness-card">
            <div class="card-icon">✅</div>
            <div class="card-content">
              <div class="card-value">{{ backtestResult.recommendation_effectiveness.performance ? '优秀' : '需改进' }}</div>
              <div class="card-label">整体表现</div>
              <div class="card-desc">{{ backtestResult.recommendation_effectiveness.performance ? '超出预期' : '未达预期' }}</div>
            </div>
          </div>
        </div>
      </section>

      <!-- AI推荐上下文 -->
      <section class="panel context-section" v-if="backtestResult.recommendation_context">
        <h3>🧠 AI推荐上下文</h3>
        <div class="context-grid">
          <div class="context-item">
            <span class="context-label">综合评分:</span>
            <div class="score-bar">
              <div class="score-fill" :style="{ width: (backtestResult.recommendation_context.overall_score || 0) * 100 + '%' }"></div>
              <span class="score-text">{{ ((backtestResult.recommendation_context.overall_score || 0) * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <div class="context-item">
            <span class="context-label">技术分析:</span>
            <div class="score-bar">
              <div class="score-fill" :style="{ width: (backtestResult.recommendation_context.technical_score || 0) * 100 + '%' }"></div>
              <span class="score-text">{{ ((backtestResult.recommendation_context.technical_score || 0) * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <div class="context-item">
            <span class="context-label">基本面分析:</span>
            <div class="score-bar">
              <div class="score-fill" :style="{ width: (backtestResult.recommendation_context.fundamental_score || 0) * 100 + '%' }"></div>
              <span class="score-text">{{ ((backtestResult.recommendation_context.fundamental_score || 0) * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <div class="context-item">
            <span class="context-label">情绪分析:</span>
            <div class="score-bar">
              <div class="score-fill" :style="{ width: (backtestResult.recommendation_context.sentiment_score || 0) * 100 + '%' }"></div>
              <span class="score-text">{{ ((backtestResult.recommendation_context.sentiment_score || 0) * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <div class="context-item">
            <span class="context-label">动量分析:</span>
            <div class="score-bar">
              <div class="score-fill" :style="{ width: (backtestResult.recommendation_context.momentum_score || 0) * 100 + '%' }"></div>
              <span class="score-text">{{ ((backtestResult.recommendation_context.momentum_score || 0) * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <div class="context-item">
            <span class="context-label">预期收益率:</span>
            <span class="context-value">{{ ((backtestResult.recommendation_context.expected_return || 0) * 100).toFixed(1) }}%</span>
          </div>
        </div>
      </section>

      <!-- 交易记录 -->
      <section class="panel trades-section">
        <div class="section-header">
          <h3>📋 交易记录</h3>
          <div class="trade-stats">
            <span>共 {{ totalTrades }} 笔交易</span>
          </div>
        </div>

        <!-- 交易记录表格 -->
        <div v-if="trades.length > 0" class="table-container">
          <table class="trades-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>操作</th>
                <th>价格</th>
                <th>数量</th>
                <th>价值</th>
                <th>手续费</th>
                <th>盈亏</th>
                <th>原因</th>
                <th>AI置信度</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="trade in trades" :key="trade.timestamp">
                <td>{{ formatDateTime(trade.timestamp) }}</td>
                <td>
                  <span class="trade-action" :class="trade.side?.toLowerCase()">
                    {{ trade.side === 'BUY' ? '买入' : '卖出' }}
                  </span>
                </td>
                <td>${{ trade.price?.toFixed(2) }}</td>
                <td>{{ trade.quantity?.toFixed(6) }}</td>
                <td>${{ (trade.price * trade.quantity)?.toFixed(2) }}</td>
                <td>${{ trade.commission?.toFixed(4) }}</td>
                <td :class="trade.pnl >= 0 ? 'positive' : 'negative'">
                  {{ trade.pnl ? formatCurrency(trade.pnl) : '-' }}
                </td>
                <td>{{ trade.reason || '-' }}</td>
                <td>
                  <div class="confidence-bar">
                    <div class="confidence-fill" :style="{ width: (trade.ai_confidence || 0) * 100 + '%' }"></div>
                    <span class="confidence-text">{{ ((trade.ai_confidence || 0) * 100).toFixed(0) }}%</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
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

        <!-- 无交易记录 -->
        <div v-else class="no-trades">
          <p>暂无交易记录</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/api.js'

export default {
  name: 'BacktestDetail',
  setup() {
    const route = useRoute()
    const router = useRouter()

    const loading = ref(false)
    const errorMessage = ref('')
    const record = ref(null)
    const backtestResult = ref(null)
    const trades = ref([])
    const currentPage = ref(1)
    const totalTrades = ref(0)
    const totalPages = ref(1)

    const recordId = route.params.id

    const loadData = async () => {
      if (!recordId) {
        errorMessage.value = '缺少回测记录ID'
        return
      }

      loading.value = true
      errorMessage.value = ''

      try {
        // 获取回测记录详情
        const recordResponse = await api.getBacktestRecord(recordId)
        if (!recordResponse.success) {
          throw new Error(recordResponse.error || '获取回测记录失败')
        }

        record.value = recordResponse.record

        // 如果回测已完成，解析结果
        if (record.value.status === 'completed' && record.value.result) {
          backtestResult.value = JSON.parse(record.value.result)

          // 加载交易记录
          await loadTrades()
        }

      } catch (error) {
        console.error('加载回测详情失败:', error)
        errorMessage.value = error.message || '加载失败，请稍后重试'
      } finally {
        loading.value = false
      }
    }

    const loadTrades = async () => {
      try {
        const response = await api.getBacktestTrades(recordId, {
          page: currentPage.value,
          limit: 20
        })

        if (response.success) {
          trades.value = response.trades || []
          totalTrades.value = response.pagination?.total || 0
          totalPages.value = response.pagination?.pages || 1
        } else {
          throw new Error(response.error || '获取交易记录失败')
        }
      } catch (error) {
        console.error('加载交易记录失败:', error)
        // 不显示错误，只记录日志
      }
    }

    const changePage = (page) => {
      currentPage.value = page
      loadTrades()
    }

    const goBack = () => {
      router.push('/ai-analysis-dashboard')
    }

    const exportReport = () => {
      alert('导出功能开发中')
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

    const getSharpeDesc = (ratio) => {
      if (!ratio) return '暂无数据'
      if (ratio >= 2) return '优秀'
      if (ratio >= 1) return '良好'
      if (ratio >= 0) return '一般'
      return '较差'
    }

    const getDrawdownDesc = (drawdown) => {
      if (drawdown <= 0.05) return '很低'
      if (drawdown <= 0.10) return '可接受'
      if (drawdown <= 0.20) return '较高'
      return '很高'
    }

    const getAccuracyDesc = (accuracy) => {
      if (!accuracy) return '暂无数据'
      if (accuracy >= 0.8) return '优秀'
      if (accuracy >= 0.6) return '良好'
      if (accuracy >= 0.4) return '一般'
      return '需改进'
    }

    const getAccuracyLevel = (score) => {
      if (!score) return '未知'
      if (score >= 0.8) return '高准确性'
      if (score >= 0.6) return '中等准确性'
      if (score >= 0.4) return '低准确性'
      return '准确性不足'
    }

    const getEffectivenessDesc = (effectiveness) => {
      if (!effectiveness) return '暂无数据'
      if (effectiveness >= 1.5) return '大幅超出预期'
      if (effectiveness >= 1.0) return '达到预期'
      if (effectiveness >= 0.5) return '部分达成预期'
      return '未达预期'
    }

    const formatCurrency = (value) => {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2
      }).format(value)
    }

    const formatDateTime = (dateTimeStr) => {
      if (!dateTimeStr) return '-'
      const date = new Date(dateTimeStr)
      return date.toLocaleString('zh-CN')
    }

    onMounted(() => {
      loadData()
    })

    return {
      loading,
      errorMessage,
      record,
      backtestResult,
      trades,
      currentPage,
      totalTrades,
      totalPages,
      loadData,
      loadTrades,
      changePage,
      goBack,
      exportReport,
      getStrategyName,
      getStatusText,
      getSharpeDesc,
      getDrawdownDesc,
      getAccuracyDesc,
      getAccuracyLevel,
      getEffectivenessDesc,
      formatCurrency,
      formatDateTime
    }
  }
}
</script>

<style scoped>
.backtest-detail {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
  background: #f8fafc;
  min-height: 100vh;
}

.header-panel {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  margin-bottom: 20px;
}

.header-content h1 {
  margin: 0 0 8px 0;
  font-size: 2rem;
  font-weight: 700;
}

.subtitle {
  margin: 0;
  opacity: 0.9;
  font-size: 1rem;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.back-btn, .export-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.back-btn {
  background: rgba(255, 255, 255, 0.1);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.back-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.export-btn {
  background: rgba(255, 255, 255, 0.2);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.export-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.panel h3 {
  margin: 0 0 20px 0;
  color: #1f2937;
  font-size: 1.25rem;
}

/* 概览卡片 */
.overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.overview-card {
  background: #f8fafc;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 1px solid #e5e7eb;
}

.card-icon {
  font-size: 2rem;
  opacity: 0.8;
}

.card-content {
  flex: 1;
}

.card-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 4px;
}

.card-label {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 4px;
}

.card-change {
  font-size: 13px;
  font-weight: 600;
}

.card-change.positive {
  color: #10b981;
}

.card-change.negative {
  color: #ef4444;
}

.card-desc {
  font-size: 12px;
  color: #9ca3af;
}

/* 配置信息 */
.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.config-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f3f4f6;
}

.config-item:last-child {
  border-bottom: none;
}

.config-label {
  font-weight: 500;
  color: #374151;
}

.config-value {
  color: #6b7280;
}

.config-value.status {
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
}

.status-completed {
  background: #dcfce7;
  color: #166534;
}

.status-running {
  background: #dbeafe;
  color: #1e40af;
}

.status-failed {
  background: #fee2e2;
  color: #991b1b;
}

/* AI洞察 */
.insights-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.insight-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  background: #f8fafc;
  border-radius: 6px;
  border-left: 4px solid #3b82f6;
}

.insight-icon {
  font-size: 1.2rem;
  margin-top: 2px;
}

.insight-text {
  flex: 1;
  color: #374151;
  line-height: 1.5;
}

/* AI预测准确性 */
.accuracy-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.accuracy-card {
  background: linear-gradient(135deg, #e0f2fe 0%, #b3e5fc 100%);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid #90caf9;
}

.accuracy-card .card-icon {
  color: #1976d2;
}

.accuracy-card .card-content {
  flex: 1;
}

.accuracy-card .card-value {
  font-size: 1.25rem;
  font-weight: 700;
  color: #0d47a1;
  margin-bottom: 4px;
}

.accuracy-card .card-label {
  font-size: 13px;
  color: #1565c0;
  margin-bottom: 2px;
}

.accuracy-card .card-desc {
  font-size: 11px;
  color: #1976d2;
  opacity: 0.8;
}

/* 推荐有效性 */
.effectiveness-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.effectiveness-card {
  background: linear-gradient(135deg, #f3e5f5 0%, #e1bee7 100%);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid #ba68c8;
}

.effectiveness-card .card-icon {
  color: #7b1fa2;
}

.effectiveness-card .card-content {
  flex: 1;
}

.effectiveness-card .card-value {
  font-size: 1.25rem;
  font-weight: 700;
  color: #4a148c;
  margin-bottom: 4px;
}

.effectiveness-card .card-label {
  font-size: 13px;
  color: #6a1b9a;
  margin-bottom: 2px;
}

.effectiveness-card .card-desc {
  font-size: 11px;
  color: #7b1fa2;
  opacity: 0.8;
}

/* AI推荐上下文 */
.context-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.context-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.context-label {
  font-weight: 600;
  color: #374151;
  font-size: 14px;
}

.score-bar {
  position: relative;
  width: 100%;
  height: 20px;
  background: #e5e7eb;
  border-radius: 10px;
  overflow: hidden;
}

.score-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981 0%, #059669 100%);
  transition: width 0.3s ease;
  border-radius: 10px;
}

.score-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 12px;
  font-weight: 600;
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

.context-value {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  padding: 8px 0;
}

/* 交易记录 */
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.trade-stats {
  color: #6b7280;
  font-size: 14px;
}

.table-container {
  overflow-x: auto;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  background: white;
}

.trades-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.trades-table th,
.trades-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #f3f4f6;
}

.trades-table th {
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
  position: sticky;
  top: 0;
}

.trades-table tr:hover {
  background: #f9fafb;
}

.trade-action {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.trade-action.buy {
  background: #dcfce7;
  color: #166534;
}

.trade-action.sell {
  background: #fee2e2;
  color: #991b1b;
}

.positive {
  color: #10b981;
  font-weight: 600;
}

.negative {
  color: #ef4444;
  font-weight: 600;
}

.confidence-bar {
  position: relative;
  width: 60px;
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.confidence-fill {
  height: 100%;
  background: linear-gradient(90deg, #ef4444 0%, #f59e0b 50%, #10b981 100%);
  transition: width 0.3s ease;
}

.confidence-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 10px;
  font-weight: 600;
  color: #374151;
}

/* 分页 */
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

.no-trades {
  text-align: center;
  padding: 40px;
  color: #6b7280;
}

/* 加载和错误状态 */
.loading-container,
.error-message {
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

.error-message p {
  color: #dc2626;
}

.btn-retry {
  background: #dc2626;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
}

.btn-retry:hover {
  background: #b91c1c;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .backtest-detail {
    padding: 10px;
  }

  .header-content h1 {
    font-size: 1.5rem;
  }

  .header-actions {
    flex-direction: column;
    width: 100%;
  }

  .overview-grid {
    grid-template-columns: 1fr;
  }

  .config-grid {
    grid-template-columns: 1fr;
  }

  .section-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .trades-table {
    font-size: 12px;
  }

  .trades-table th,
  .trades-table td {
    padding: 8px 6px;
  }
}
</style>
